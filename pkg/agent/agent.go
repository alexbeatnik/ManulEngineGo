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
