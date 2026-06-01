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
	"fmt"
	"strings"
	"sync"

	"github.com/manulengineer/manulheart/pkg/browser"
	"github.com/manulengineer/manulheart/pkg/config"
	"github.com/manulengineer/manulheart/pkg/dsl"
	"github.com/manulengineer/manulheart/pkg/explain"
	"github.com/manulengineer/manulheart/pkg/runtime"
	"github.com/manulengineer/manulheart/pkg/utils"
)

// Options configures a Session.
type Options struct {
	// Headless runs Chrome without a visible window. Default: false.
	Headless bool
	// Port is the CDP debug port for a Launch-managed Chrome. 0 → 9222.
	// Ignored by Attach.
	Port int
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
	mu      sync.Mutex
	rt      *runtime.Runtime
	page    browser.Page
	chrome  *browser.ChromeProcess // non-nil only when we launched Chrome
	closed  bool
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

	return newSession(opts, page, cp), nil
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
	return newSession(opts, page, nil), nil
}

func newSession(opts Options, page browser.Page, cp *browser.ChromeProcess) *Session {
	cfg := config.Default()
	if opts.Config != nil {
		cfg = *opts.Config
	}
	logger := opts.Logger
	if logger == nil {
		logger = utils.NewLogger(nil) // discards output
	}
	return &Session{
		rt:     runtime.New(cfg, page, logger),
		page:   page,
		chrome: cp,
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
		// a failure the caller has to string-match.
		if isNotFound(res, err) {
			return Value{Found: false}, nil
		}
		return Value{}, fmt.Errorf("agent: read %q: %w", target, err)
	}
	return Value{Text: res.ActionValue, Found: res.ActionValue != ""}, nil
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
		for _, r := range hr.Results {
			out.Results = append(out.Results, outcomeFrom(r, errFromResult(r)))
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
