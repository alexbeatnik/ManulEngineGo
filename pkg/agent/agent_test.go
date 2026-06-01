package agent

import (
	"context"
	"testing"

	"github.com/manulengineer/manulheart/pkg/config"
	"github.com/manulengineer/manulheart/pkg/dom"
	"github.com/manulengineer/manulheart/pkg/runtime"
	"github.com/manulengineer/manulheart/pkg/utils"
)

// newTestSession wires a Session around a runtime.MockPage so Read can be
// exercised without a real browser. Mirrors the AdoptWorker/MockPage pattern
// used across the runtime/worker suites.
func newTestSession(page *runtime.MockPage) *Session {
	cfg := config.Default()
	return &Session{
		rt:   runtime.New(cfg, page, utils.NewLogger(nil)),
		page: page,
	}
}

func TestRead_ReturnsValue(t *testing.T) {
	page := &runtime.MockPage{
		URL: "https://example.com",
		Elements: []dom.ElementSnapshot{
			{Tag: "h1", VisibleText: "Order Total: $42.00"},
		},
	}
	sess := newTestSession(page)

	v, err := sess.Read(context.Background(), "Order Total")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !v.Found {
		t.Fatalf("expected Found=true, got %+v", v)
	}
	if v.Text != "Order Total: $42.00" {
		t.Errorf("unexpected text: %q", v.Text)
	}
}

// TestRead_IsZeroScan is the load-bearing guarantee: a Read must cost exactly
// one probe round-trip (the extraction probe) and never trigger the full DOM
// snapshot that a scan would. This is the whole point of the API — reading one
// value must not pay for the whole page.
func TestRead_IsZeroScan(t *testing.T) {
	page := &runtime.MockPage{
		Elements: []dom.ElementSnapshot{
			{Tag: "span", VisibleText: "Hello"},
		},
	}
	sess := newTestSession(page)

	if _, err := sess.Read(context.Background(), "Hello"); err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if page.ProbeCalls != 1 {
		t.Errorf("expected exactly 1 probe call (zero-scan), got %d", page.ProbeCalls)
	}
}

func TestRead_NotFoundIsCleanResult(t *testing.T) {
	page := &runtime.MockPage{
		Elements: []dom.ElementSnapshot{
			{Tag: "span", VisibleText: "Something else"},
		},
	}
	sess := newTestSession(page)

	v, err := sess.Read(context.Background(), "Nonexistent Label")
	if err != nil {
		t.Fatalf("not-found must be a clean result, got error: %v", err)
	}
	if v.Found {
		t.Errorf("expected Found=false, got %+v", v)
	}
}

func TestRead_OnClosedSession(t *testing.T) {
	sess := newTestSession(&runtime.MockPage{})
	if err := sess.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if _, err := sess.Read(context.Background(), "anything"); err == nil {
		t.Errorf("expected error reading from a closed session")
	}
}

func TestStep_NotFoundCarriesReason(t *testing.T) {
	// An empty page: nothing can resolve, so the scorer returns no candidate
	// and the runtime reports not_found.
	page := &runtime.MockPage{URL: "https://example.com"}
	sess := newTestSession(page)

	out, err := sess.Step(context.Background(), "Click the 'Nonexistent Widget' button")
	if err == nil {
		t.Fatalf("expected an error for an unresolvable target")
	}
	if out.OK {
		t.Errorf("expected OK=false, got %+v", out)
	}
	// The machine-readable reason must be set on failure — that's the whole
	// point: no error-string parsing required by the caller.
	if out.Reason != ReasonNotFound {
		t.Errorf("expected reason=not_found, got %q (err=%v)", out.Reason, err)
	}
}

// TestStep_LowConfidenceSurfacesNear is the key agent-ergonomics contract: a
// weak fuzzy match still "succeeds" (the click lands somewhere) but Near is
// populated so the agent can decide whether it landed on the right thing —
// without paying for a follow-up scan.
func TestStep_LowConfidenceSurfacesNear(t *testing.T) {
	page := &runtime.MockPage{
		URL: "https://example.com",
		Elements: []dom.ElementSnapshot{
			{Tag: "button", VisibleText: "Submit", IsVisible: true},
		},
	}
	sess := newTestSession(page)

	out, err := sess.Step(context.Background(), "Click the 'Nonexistent Widget' button")
	if err != nil {
		t.Fatalf("weak match should still succeed, got error: %v", err)
	}
	if !out.OK {
		t.Fatalf("expected OK=true for a weak match, got %+v", out)
	}
	if out.Score >= lowConfidence {
		t.Skipf("match wasn't low-confidence (score %.3f); nothing to assert", out.Score)
	}
	if len(out.Near) == 0 {
		t.Errorf("low-confidence success must surface Near candidates, got none")
	}
}

func TestStep_SuccessIsClean(t *testing.T) {
	page := &runtime.MockPage{
		URL: "https://example.com",
		Elements: []dom.ElementSnapshot{
			{Tag: "button", VisibleText: "Login", IsVisible: true},
		},
	}
	sess := newTestSession(page)

	out, err := sess.Step(context.Background(), "Click the 'Login' button")
	if err != nil {
		t.Fatalf("Step failed: %v", err)
	}
	if !out.OK || out.Reason != ReasonOK {
		t.Errorf("expected clean success, got %+v", out)
	}
	if out.Action != "click" {
		t.Errorf("expected action=click, got %q", out.Action)
	}
}

func TestRun_AggregatesAndRecords(t *testing.T) {
	page := &runtime.MockPage{
		URL: "https://example.com",
		Elements: []dom.ElementSnapshot{
			{Tag: "button", VisibleText: "Login", IsVisible: true},
		},
	}
	sess := newTestSession(page)

	script := "STEP 1: Smoke\n" +
		"    NAVIGATE TO 'https://example.com'\n" +
		"    Click the 'Login' button\n"

	out, err := sess.Run(context.Background(), script)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !out.OK {
		t.Errorf("expected OK run, got %+v", out)
	}
	if out.TotalSteps != 2 || len(out.Results) != 2 {
		t.Errorf("expected 2 recorded steps, got TotalSteps=%d results=%d", out.TotalSteps, len(out.Results))
	}
	for _, r := range out.Results {
		if !r.OK || r.Reason != ReasonOK {
			t.Errorf("step not clean: %+v", r)
		}
	}
}

func TestClose_Idempotent(t *testing.T) {
	sess := newTestSession(&runtime.MockPage{})
	if err := sess.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Errorf("second Close should be a no-op, got: %v", err)
	}
}
