// Package agent is the batteries-included facade for embedding ManulHeart in
// agent and assistant applications.
//
// It owns the entire browser lifecycle so a consumer never has to spawn
// Chrome, speak CDP, or assemble the runtime/scorer pipeline itself:
//
//	sess, err := agent.Launch(ctx, agent.Options{Headless: true})
//	defer sess.Close()
//	v, _ := sess.Read(ctx, "the order total")   // zero-scan text read
//	out, _ := sess.Step(ctx, "Click the 'Checkout' button")
//
// The API returns compact, agent-friendly results (no full scorer breakdown)
// and machine-readable failure reasons, so callers don't have to parse
// human-readable error strings.
//
// Concurrency: a Session owns one runtime.Runtime, which is single-goroutine
// by design (see the package docs in pkg/runtime). Do not call methods on the
// same Session from multiple goroutines concurrently — Session serializes its
// own calls with a mutex to make accidental misuse safe, but it does not make
// concurrent browser work parallel. For parallel hunts, use one Session per
// goroutine (or pkg/worker).
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/alexbeatnik/ManulHeart/pkg/browser"
	"github.com/alexbeatnik/ManulHeart/pkg/config"
	"github.com/alexbeatnik/ManulHeart/pkg/dsl"
	"github.com/alexbeatnik/ManulHeart/pkg/explain"
	"github.com/alexbeatnik/ManulHeart/pkg/heuristics"
	"github.com/alexbeatnik/ManulHeart/pkg/runtime"
	"github.com/alexbeatnik/ManulHeart/pkg/scan"
	"github.com/alexbeatnik/ManulHeart/pkg/utils"
)

// Options configures a Session.
type Options struct {
	// Headless runs Chrome without a visible window. Default: false.
	Headless bool
	// Port is the CDP debug port for a Launch-managed Chrome. 0 → 9222.
	// Ignored by Attach. Connect uses it to both probe for an existing Chrome
	// and, failing that, to Launch one.
	Port int
	// CDPURL is an explicit CDP HTTP endpoint (e.g. "http://127.0.0.1:9222").
	// Used by Connect: when set it takes precedence over Port for the
	// attach probe. Ignored by Launch.
	CDPURL string
	// ExecutablePath overrides the Chrome binary location (Launch only).
	ExecutablePath string
	// UserDataDir overrides the Chrome profile directory (Launch only).
	// Empty → a unique temp profile is created and removed on Close.
	UserDataDir string
	// Config is the engine configuration. Zero value → config.Default().
	Config *config.Config
	// Logger sinks engine logs. Nil → a logger that discards output, so an
	// embedding app's stdout/stderr stays clean unless it opts in.
	Logger *utils.Logger
}

// Session is a live, owned browser connection plus the targeting runtime.
// Create one with Launch (ManulHeart spawns Chrome) or Attach (connect to an
// already-running Chrome). Always Close it.
type Session struct {
	mu       sync.Mutex
	rt       *runtime.Runtime
	page     browser.Page
	chrome   *browser.ChromeProcess // non-nil only when we launched Chrome
	endpoint string                 // CDP HTTP endpoint, for opening background tabs
	closed   bool
}

// Launch spawns a Chrome process owned by ManulHeart, attaches to its first
// page, and returns a ready Session. The Chrome process (and its temp profile,
// when one was created) is torn down by Close.
func Launch(ctx context.Context, opts Options) (*Session, error) {
	chromeOpts := browser.DefaultChromeOptions()
	chromeOpts.Headless = opts.Headless
	if opts.Port != 0 {
		chromeOpts.Port = opts.Port
	}
	if opts.ExecutablePath != "" {
		chromeOpts.ExecutablePath = opts.ExecutablePath
	}
	if opts.UserDataDir != "" {
		chromeOpts.UserDataDir = opts.UserDataDir
	}

	cp, err := browser.LaunchChrome(ctx, chromeOpts)
	if err != nil {
		return nil, fmt.Errorf("agent: launch chrome: %w", err)
	}

	page, err := browser.NewCDPBrowser(cp.Endpoint()).FirstPage(ctx)
	if err != nil {
		_ = cp.Close()
		return nil, fmt.Errorf("agent: attach to launched chrome: %w", err)
	}

	return newSession(opts, page, cp, cp.Endpoint()), nil
}

// Attach connects to an already-running Chrome at the given CDP HTTP endpoint
// (e.g. "http://127.0.0.1:9222"). ManulHeart does NOT own that Chrome process,
// so Close leaves it running. When urlSubstr is non-empty, the most recently
// active page whose URL contains it is selected; otherwise the first page.
func Attach(ctx context.Context, cdpURL, urlSubstr string, opts Options) (*Session, error) {
	if cdpURL == "" {
		return nil, fmt.Errorf("agent: Attach requires a CDP endpoint URL")
	}
	page, err := browser.NewCDPBrowser(cdpURL).PageMatching(ctx, urlSubstr)
	if err != nil {
		return nil, fmt.Errorf("agent: attach: %w", err)
	}
	return newSession(opts, page, nil, cdpURL), nil
}

func newSession(opts Options, page browser.Page, cp *browser.ChromeProcess, endpoint string) *Session {
	cfg := config.Default()
	if opts.Config != nil {
		cfg = *opts.Config
	}
	logger := opts.Logger
	if logger == nil {
		// Truly discard: utils.NewLogger(nil) writes to os.Stdout (the nil is
		// the optional logFile, not the sink), which would corrupt the
		// structured JSON the CLI/agent consumers parse. Route to io.Discard.
		logger = utils.NewLoggerTo(io.Discard, nil)
	}
	return &Session{
		rt:       runtime.New(cfg, page, logger),
		page:     page,
		chrome:   cp,
		endpoint: endpoint,
	}
}

// Close releases the page connection and, when this Session launched Chrome,
// terminates that Chrome process and removes any temp profile it created.
// Safe to call multiple times.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true

	var firstErr error
	if s.page != nil {
		if err := s.page.Close(); err != nil {
			firstErr = err
		}
	}
	if s.chrome != nil {
		if err := s.chrome.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Value is the result of a Read.
type Value struct {
	// Text is the extracted, trimmed text. Empty when Found is false.
	Text string
	// Found reports whether the target resolved to a non-empty value.
	Found bool
	// Reason classifies the outcome — ReasonOK when Found, ReasonNotFound
	// otherwise. Lets a caller branch without string-matching. (Read uses the
	// dedicated extraction probe, not the scorer pipeline, so it cannot offer
	// Near candidates — use Step/Map to retarget after a miss.)
	Reason Reason
}

// Read extracts the text of the element matching target, using only the
// dedicated extraction probe — a single CDP round-trip with no full-DOM
// snapshot. This is the cheapest way for an agent to read one piece of the
// page (a heading, a price, a status) without paying for a whole page scan.
//
// A target that doesn't resolve (or resolves to empty) returns Found=false
// with a nil error — "nothing there" is a normal answer, not a failure.
func (s *Session) Read(ctx context.Context, target string) (Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Value{}, fmt.Errorf("agent: session closed")
	}

	cmd := dsl.Command{
		Type:            dsl.CmdExtract,
		Target:          target,
		ExtractVar:      "_agent_read",
		InteractionMode: dsl.ModeNone,
	}
	res, err := s.rt.RunCommand(ctx, cmd)
	if err != nil {
		// CmdExtract reports "not found or empty" as an error; for Read we
		// translate that into a clean not-found rather than surfacing it as
		// a failure the caller has to string-match. Surface the top
		// candidates so the caller can retarget without a follow-up scan.
		if isNotFound(res, err) {
			return Value{Found: false, Reason: ReasonNotFound}, nil
		}
		return Value{}, fmt.Errorf("agent: read %q: %w", target, err)
	}
	found := res.ActionValue != ""
	reason := ReasonOK
	if !found {
		reason = ReasonNotFound
	}
	return Value{Text: res.ActionValue, Found: found, Reason: reason}, nil
}

// ReadText returns the case-preserved, shadow-DOM-aware visible text of the
// page, sanitized of markup noise (base64 blobs, data-* attributes, SVG path
// data) so it's fit to hand to an LLM. Pass a CSS selector to scope extraction
// to one region; pass "" for the whole document body.
//
// This complements Read: Read resolves ONE element by a human label and
// returns its value; ReadText dumps a whole region's prose — the right tool
// when an agent needs to read content the semantic scan can't see (dynamic
// answer panels, article bodies). It is a single probe round-trip — no full
// snapshot, no scan.
func (s *Session) ReadText(ctx context.Context, selector string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", fmt.Errorf("agent: session closed")
	}

	raw, err := s.page.CallProbe(ctx, heuristics.BuildPageTextProbe(), selector)
	if err != nil {
		return "", fmt.Errorf("agent: read text: %w", err)
	}
	// CallProbe returns the JSON-encoded string value; unquote it.
	var text string
	if jsonErr := json.Unmarshal(raw, &text); jsonErr != nil {
		// Some backends hand back the bare string; use it as-is.
		text = string(raw)
	}
	return sanitizeText(text), nil
}

// sanitizeText strips markup noise from page text so an LLM isn't drowned in
// base64 blobs, data-* attribute dumps, or SVG path data. Mirrors the cleanup
// consumers (e.g. OS-MANUL) previously hand-rolled around raw CDP text.
func sanitizeText(raw string) string {
	if raw == "" {
		return ""
	}
	var cleaned []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Base64 / data URIs.
		if strings.HasPrefix(line, "data:image/") || strings.HasPrefix(line, "data:text/") ||
			(len(line) > 80 && !strings.Contains(line, " ") && !strings.Contains(line, "-")) {
			continue
		}
		// HTML attribute / framework noise.
		if strings.HasPrefix(line, "data-") || strings.HasPrefix(line, "jsaction=") ||
			strings.HasPrefix(line, "jscontroller=") || strings.HasPrefix(line, "jsuid=") {
			continue
		}
		// SVG path data (long M…Z command strings).
		if strings.Contains(line, "M") && strings.Contains(line, "Z") && len(line) > 100 {
			continue
		}
		// Drop consecutive duplicate lines (repeated nav/chrome text) so the
		// LLM isn't billed for the same string twice in a row.
		if len(cleaned) > 0 && cleaned[len(cleaned)-1] == line {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}

// TruncateText caps text to maxChars (counted by rune), appending a short
// "[+N chars truncated]" marker so a consumer knows the read was budgeted.
// maxChars <= 0 returns s unchanged. This is the cheap way for an LLM caller
// to bound how much page prose lands in a prompt.
func TruncateText(s string, maxChars int) string {
	if maxChars <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	dropped := len(runes) - maxChars
	return strings.TrimRight(string(runes[:maxChars]), " \n") +
		fmt.Sprintf("\n[+%d chars truncated]", dropped)
}

// Reason is the agent-facing, machine-readable classification of a step's
// outcome. It mirrors explain.FailureReason but is owned by the agent API so
// callers depend on a stable surface, not on internal runtime types.
type Reason string

const (
	ReasonOK           Reason = "ok"
	ReasonNotFound     Reason = "not_found"
	ReasonAmbiguous    Reason = "ambiguous"
	ReasonTimeout      Reason = "timeout"
	ReasonVerifyFailed Reason = "verify_failed"
	ReasonActionFailed Reason = "action_failed"
)

func reasonFrom(fr explain.FailureReason) Reason {
	switch fr {
	case explain.ReasonNotFound:
		return ReasonNotFound
	case explain.ReasonAmbiguous:
		return ReasonAmbiguous
	case explain.ReasonTimeout:
		return ReasonTimeout
	case explain.ReasonVerifyFailed:
		return ReasonVerifyFailed
	case explain.ReasonActionFailed:
		return ReasonActionFailed
	default:
		return ReasonActionFailed
	}
}

// Cand is a compact candidate: just the human-visible label and its score.
// Surfaced on a failed/low-confidence Step so an agent can retarget ("you
// almost matched 'Log In' at 0.18") without a follow-up page scan.
type Cand struct {
	Text  string  `json:"text"`
	Score float64 `json:"score"`
}

// StepOutcome is the compact result of one Step — the agent-facing subset of
// explain.ExecutionResult, with the full scorer breakdown deliberately dropped.
type StepOutcome struct {
	// OK is true when the step succeeded.
	OK bool `json:"ok"`
	// Step is the raw DSL line that was executed (e.g. "CLICK 'Login'").
	// Carried so DescribeForLLM can echo the exact command back to a model.
	Step string `json:"step,omitempty"`
	// Action is the lowercase command kind (click, fill, navigate, …).
	Action string `json:"action"`
	// Value is the value used/extracted (fill value, extracted text, URL).
	Value string `json:"value,omitempty"`
	// URL is the page URL after the step ran.
	URL string `json:"url,omitempty"`
	// Reason classifies the outcome — ReasonOK on success.
	Reason Reason `json:"reason"`
	// Error is the raw error message when OK is false (for logs/diagnostics).
	Error string `json:"error,omitempty"`
	// Score is the winning candidate's score (0 when no target was resolved).
	Score float64 `json:"score,omitempty"`
	// Near lists the top candidates, populated only when the step failed to
	// resolve a target or matched with low confidence. Empty otherwise.
	Near []Cand `json:"near,omitempty"`
}

// lowConfidence is the score below which a "successful" target match is worth
// surfacing candidates for. Picked to match OS-MANUL's empirically-tuned 0.35.
const lowConfidence = 0.35

// Step runs a single plain-English instruction (one DSL line, e.g.
// "Click the 'Login' button") and returns a compact outcome. Failures carry a
// machine-readable Reason and, for target-resolution problems, the top
// candidates in Near — so an agent can correct course without scanning.
func (s *Session) Step(ctx context.Context, instruction string) (StepOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return StepOutcome{}, fmt.Errorf("agent: session closed")
	}

	hunt, perr := dsl.Parse(strings.NewReader(instruction))
	if perr != nil {
		return StepOutcome{OK: false, Reason: ReasonActionFailed, Error: perr.Error()}, perr
	}
	if len(hunt.Commands) == 0 {
		return StepOutcome{OK: true, Reason: ReasonOK}, nil
	}

	res, execErr := s.rt.RunCommand(ctx, hunt.Commands[0])
	return outcomeFrom(res, execErr), execErr
}

// RunOutcome is the compact aggregate of running a whole hunt script.
type RunOutcome struct {
	OK         bool          `json:"ok"`
	URL        string        `json:"url,omitempty"`
	TotalSteps int           `json:"total_steps"`
	Passed     int           `json:"passed"`
	Failed     int           `json:"failed"`
	Results    []StepOutcome `json:"results,omitempty"`
	Duration   int64         `json:"duration_ms"`
}

// Run executes a full .hunt script (multiple lines, STEP blocks, loops, etc.)
// against the session's page and returns a compact aggregate. Per-step compact
// outcomes are in Steps_, in order. Use this for the "agent emits a whole
// script" path; use Step for one-instruction-at-a-time control.
func (s *Session) Run(ctx context.Context, huntScript string) (RunOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return RunOutcome{}, fmt.Errorf("agent: session closed")
	}

	hunt, perr := dsl.Parse(strings.NewReader(huntScript))
	if perr != nil {
		return RunOutcome{}, fmt.Errorf("agent: parse hunt: %w", perr)
	}
	hunt.SourcePath = "<agent>"
	if err := dsl.ResolveImports(hunt); err != nil {
		return RunOutcome{}, fmt.Errorf("agent: resolve imports: %w", err)
	}
	if err := hunt.Expand(); err != nil {
		return RunOutcome{}, fmt.Errorf("agent: expand hunt: %w", err)
	}

	hr, runErr := s.rt.RunHunt(ctx, hunt)
	out := RunOutcome{}
	if hr != nil {
		out.OK = hr.Success
		out.TotalSteps = hr.TotalSteps
		out.Passed = hr.Passed
		out.Failed = hr.Failed
		out.Duration = hr.TotalDurationMS
		// The final URL lives on RunOutcome; per-step URLs are only emitted
		// when they CHANGE from the previous step, so a multi-step run on one
		// page doesn't repeat the same URL on every StepOutcome.
		prevURL := ""
		for _, r := range hr.Results {
			so := outcomeFrom(r, errFromResult(r))
			if so.URL == prevURL {
				so.URL = ""
			} else if so.URL != "" {
				prevURL = so.URL
			}
			out.Results = append(out.Results, so)
		}
		out.URL = lastURL(hr.Results)
	}
	return out, runErr
}

// errFromResult reconstructs a non-nil error for a recorded failed step so
// outcomeFrom classifies it correctly. The HuntResult stores the message, not
// the error value.
func errFromResult(r explain.ExecutionResult) error {
	if r.Success {
		return nil
	}
	if r.Error != "" {
		return fmt.Errorf("%s", r.Error)
	}
	return fmt.Errorf("step failed")
}

func lastURL(results []explain.ExecutionResult) string {
	for i := len(results) - 1; i >= 0; i-- {
		if results[i].PageURL != "" {
			return results[i].PageURL
		}
	}
	return ""
}

// outcomeFrom collapses a full ExecutionResult into the compact StepOutcome,
// attaching Near candidates on failure or low-confidence success.
func outcomeFrom(res explain.ExecutionResult, execErr error) StepOutcome {
	out := StepOutcome{
		OK:     execErr == nil,
		Step:   res.Step,
		Action: res.ActionPerformed,
		Value:  res.ActionValue,
		URL:    res.PageURL,
		Score:  res.WinnerScore,
		Error:  res.Error,
	}
	if execErr == nil {
		out.Reason = ReasonOK
		// Surface candidates even on success when the match was weak, so the
		// agent can decide whether the click really landed where intended.
		if res.WinnerScore > 0 && res.WinnerScore < lowConfidence {
			out.Near = topCandidates(res.RankedCandidates, 3)
		}
		return out
	}
	out.Reason = reasonFrom(res.FailureReason)
	out.Near = topCandidates(res.RankedCandidates, 3)
	return out
}

// topCandidates trims the ranked list to the n most descriptive labels.
func topCandidates(cands []explain.Candidate, n int) []Cand {
	out := make([]Cand, 0, n)
	for _, c := range cands {
		if len(out) >= n {
			break
		}
		label := candidateLabel(c)
		if label == "" {
			continue
		}
		out = append(out, Cand{Text: label, Score: c.Score.Total})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// candidateLabel picks the most human-meaningful string for a candidate, in
// the order a person would name the element.
func candidateLabel(c explain.Candidate) string {
	for _, s := range []string{c.VisibleText, c.AriaLabel, c.Placeholder} {
		if t := strings.TrimSpace(s); t != "" {
			return t
		}
	}
	return ""
}

// MapBudget bounds a Map call so the result stays cheap for an LLM prompt.
type MapBudget struct {
	// MaxPerGroup caps how many elements are returned per landmark group.
	// 0 → DefaultMaxPerGroup.
	MaxPerGroup int
	// IncludeUnlabeled keeps elements with no human-visible label. Off by
	// default — unlabeled controls are rarely useful targets for an agent.
	IncludeUnlabeled bool
}

// DefaultMaxPerGroup is the per-group cap when MapBudget.MaxPerGroup is 0.
const DefaultMaxPerGroup = 8

// MapElement is one interactive element in a PageMap group.
type MapElement struct {
	Label string `json:"label"`
	Role  string `json:"role"`
}

// MapGroup is a landmark region and its (budgeted) elements.
type MapGroup struct {
	// Name is the landmark label ("Page", "Main", "Nav: …", "… [shadow]").
	Name string `json:"name"`
	// Elements are the kept, deduped, capped elements in this group.
	Elements []MapElement `json:"elements"`
	// Truncated is how many elements were dropped by the per-group cap.
	Truncated int `json:"truncated,omitempty"`
}

// PageMap is the landmark-grouped, budgeted view of the current page. Groups
// are ordered for an agent: Page first, then useful landmarks (Main / forms /
// results), then chrome (header / nav / footer), then the rest alphabetically.
type PageMap struct {
	URL    string     `json:"url,omitempty"`
	Groups []MapGroup `json:"groups"`
}

// Map returns a landmark-grouped, budgeted scan of the page currently loaded
// in the session — the structural map an agent uses to pick a target by its
// exact visible label. It runs a single full-scan JS probe (no navigation,
// no Chrome lifecycle) and applies the budget in Go.
//
// This is the deliberate, bounded alternative to dumping the whole DOM: the
// caller controls the cost via MapBudget, and gets back stable, deduped,
// ranked groups instead of raw element noise.
func (s *Session) Map(ctx context.Context, budget MapBudget) (PageMap, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return PageMap{}, fmt.Errorf("agent: session closed")
	}

	groups, err := scan.ScanCurrentPageFull(ctx, s.page)
	if err != nil {
		return PageMap{}, fmt.Errorf("agent: map: %w", err)
	}

	max := budget.MaxPerGroup
	if max <= 0 {
		max = DefaultMaxPerGroup
	}

	url, _ := s.page.CurrentURL(ctx)
	pm := PageMap{URL: url}

	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sortGroupsForAgent(names)

	for _, name := range names {
		kept := dedupeUseful(groups[name], budget.IncludeUnlabeled)
		if len(kept) == 0 {
			continue
		}
		g := MapGroup{Name: name}
		if len(kept) > max {
			g.Truncated = len(kept) - max
			kept = kept[:max]
		}
		for _, e := range kept {
			role := e.Role
			if role == "" {
				role = e.Tag
			}
			g.Elements = append(g.Elements, MapElement{Label: strings.TrimSpace(e.Label), Role: role})
		}
		pm.Groups = append(pm.Groups, g)
	}
	return pm, nil
}

// dedupeUseful drops unlabeled (unless allowed) and duplicate-label elements,
// preserving order. Mirrors the filtering a consumer would otherwise hand-roll.
func dedupeUseful(items []scan.FullElement, includeUnlabeled bool) []scan.FullElement {
	seen := make(map[string]bool, len(items))
	out := make([]scan.FullElement, 0, len(items))
	for _, e := range items {
		label := strings.TrimSpace(e.Label)
		if label == "" && !includeUnlabeled {
			continue
		}
		key := strings.ToLower(label)
		if key == "" {
			key = e.Tag + "|" + e.Locator
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e)
	}
	return out
}

// sortGroupsForAgent orders landmark groups so the most target-rich regions
// come first. The ranking is guidance, not contract — callers should treat
// the order as a hint. Within the same rank, groups sort alphabetically.
func sortGroupsForAgent(groups []string) {
	sort.Slice(groups, func(i, j int) bool {
		ri, rj := groupRank(groups[i]), groupRank(groups[j])
		if ri != rj {
			return ri < rj
		}
		return groups[i] < groups[j]
	})
}

func groupRank(name string) int {
	u := strings.ToUpper(name)
	switch {
	case u == "PAGE":
		return 0
	case strings.HasPrefix(u, "MAIN"):
		return 1
	case strings.Contains(u, "SEARCH") || strings.Contains(u, "RESULT"):
		return 2
	case strings.HasPrefix(u, "FORM") || strings.HasPrefix(u, "DIALOG"):
		return 3
	case strings.HasPrefix(u, "ARTICLE") || strings.HasPrefix(u, "SECTION"):
		return 4
	case strings.HasPrefix(u, "HEADER") || strings.Contains(u, "MASTHEAD"):
		return 7
	case strings.HasPrefix(u, "NAV"):
		return 8
	case strings.HasPrefix(u, "ASIDE") || strings.HasPrefix(u, "SIDEBAR"):
		return 9
	case strings.HasPrefix(u, "FOOTER"):
		return 10
	}
	return 5
}

// isNotFound reports whether an EXTRACT result represents "target resolved to
// nothing" rather than a real execution failure (probe crash, page gone, etc).
// CmdExtract signals empty/missing targets via a sentinel error message; we
// match it here so Read can map it to a clean Found=false.
func isNotFound(res explain.ExecutionResult, err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "extract target not found or empty")
}
