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

func TestClose_Idempotent(t *testing.T) {
	sess := newTestSession(&runtime.MockPage{})
	if err := sess.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Errorf("second Close should be a no-op, got: %v", err)
	}
}
