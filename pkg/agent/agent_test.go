package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/config"
	"github.com/alexbeatnik/ManulHeart/pkg/dom"
	"github.com/alexbeatnik/ManulHeart/pkg/runtime"
	"github.com/alexbeatnik/ManulHeart/pkg/utils"
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

// scanMockPage embeds MockPage and overrides EvalJS so the full-scan probe
// returns canned landmark-grouped JSON — letting us exercise Map's budgeting
// without a real browser. All other Page methods are promoted from MockPage.
type scanMockPage struct {
	*runtime.MockPage
	fullScanJSON string
}

func (p *scanMockPage) EvalJS(ctx context.Context, expr string) ([]byte, error) {
	if strings.Contains(expr, "document.title") {
		return p.MockPage.EvalJS(ctx, expr)
	}
	return []byte(p.fullScanJSON), nil
}

func newScanSession(json string) *Session {
	page := &scanMockPage{MockPage: &runtime.MockPage{URL: "https://example.com"}, fullScanJSON: json}
	return &Session{
		rt:   runtime.New(config.Default(), page, utils.NewLogger(nil)),
		page: page,
	}
}

func TestMap_BudgetsAndRanks(t *testing.T) {
	// Footer declared before Main/Page; budgeting + ranking must reorder and
	// cap. "Apply" appears twice in Main → deduped to one.
	json := `{
		"Footer": [{"label":"Privacy","role":"link","tag":"a"}],
		"Main":   [{"label":"Apply","role":"button","tag":"button"},
		           {"label":"Apply","role":"button","tag":"button"},
		           {"label":"Cancel","role":"button","tag":"button"},
		           {"label":"Help","role":"link","tag":"a"}],
		"Page":   [{"label":"Home","role":"link","tag":"a"}]
	}`
	sess := newScanSession(json)

	pm, err := sess.Map(context.Background(), MapBudget{MaxPerGroup: 2})
	if err != nil {
		t.Fatalf("Map failed: %v", err)
	}

	if len(pm.Groups) != 3 {
		t.Fatalf("expected 3 groups, got %d: %+v", len(pm.Groups), pm.Groups)
	}
	// Ranking: Page (0) → Main (1) → Footer (10).
	if pm.Groups[0].Name != "Page" || pm.Groups[1].Name != "Main" || pm.Groups[2].Name != "Footer" {
		t.Errorf("groups not ranked as Page/Main/Footer: %v %v %v",
			pm.Groups[0].Name, pm.Groups[1].Name, pm.Groups[2].Name)
	}
	// Main had 4 (one dup) → 3 unique, capped at 2 → Truncated=1.
	main := pm.Groups[1]
	if len(main.Elements) != 2 || main.Truncated != 1 {
		t.Errorf("Main budget wrong: %d elements, truncated=%d", len(main.Elements), main.Truncated)
	}
	if main.Elements[0].Label != "Apply" {
		t.Errorf("expected deduped Apply first in Main, got %q", main.Elements[0].Label)
	}
}

func TestMap_DropsUnlabeledByDefault(t *testing.T) {
	json := `{"Main":[{"label":"","role":"button","tag":"button"},{"label":"OK","role":"button","tag":"button"}]}`
	sess := newScanSession(json)

	pm, err := sess.Map(context.Background(), MapBudget{})
	if err != nil {
		t.Fatalf("Map failed: %v", err)
	}
	if len(pm.Groups) != 1 || len(pm.Groups[0].Elements) != 1 {
		t.Fatalf("expected 1 labeled element, got %+v", pm.Groups)
	}
	if pm.Groups[0].Elements[0].Label != "OK" {
		t.Errorf("expected only 'OK', got %q", pm.Groups[0].Elements[0].Label)
	}
}

func TestMap_OnClosedSession(t *testing.T) {
	sess := newScanSession(`{}`)
	_ = sess.Close()
	if _, err := sess.Map(context.Background(), MapBudget{}); err == nil {
		t.Errorf("expected error mapping a closed session")
	}
}

// textMockPage overrides CallProbe to return canned page text, so ReadText
// can be exercised without a browser. The selector argument is recorded.
type textMockPage struct {
	*runtime.MockPage
	text     string
	gotArg   any
	probeFn  string
}

func (p *textMockPage) CallProbe(ctx context.Context, fn string, arg any) ([]byte, error) {
	p.probeFn = fn
	p.gotArg = arg
	b, _ := json.Marshal(p.text)
	return b, nil
}

func newTextSession(text string) (*Session, *textMockPage) {
	page := &textMockPage{MockPage: &runtime.MockPage{URL: "https://example.com"}, text: text}
	s := &Session{
		rt:   runtime.New(config.Default(), page, utils.NewLogger(nil)),
		page: page,
	}
	return s, page
}

func TestReadText_ReturnsCasePreservedText(t *testing.T) {
	sess, _ := newTextSession("The Weather in Kyiv is 18°C and Sunny")
	got, err := sess.ReadText(context.Background(), "")
	if err != nil {
		t.Fatalf("ReadText failed: %v", err)
	}
	if got != "The Weather in Kyiv is 18°C and Sunny" {
		t.Errorf("text not preserved verbatim: %q", got)
	}
}

func TestReadText_Sanitizes(t *testing.T) {
	raw := "Real answer line\n" +
		"data:image/png;base64,AAAABBBBCCCC\n" +
		"data-ved=somethinglong\n" +
		"   \n" +
		"Second real line"
	sess, _ := newTextSession(raw)
	got, err := sess.ReadText(context.Background(), "")
	if err != nil {
		t.Fatalf("ReadText failed: %v", err)
	}
	if strings.Contains(got, "base64") || strings.Contains(got, "data-ved") {
		t.Errorf("noise survived sanitization: %q", got)
	}
	if !strings.Contains(got, "Real answer line") || !strings.Contains(got, "Second real line") {
		t.Errorf("real content dropped: %q", got)
	}
}

func TestReadText_PassesSelector(t *testing.T) {
	sess, page := newTextSession("scoped")
	if _, err := sess.ReadText(context.Background(), "#answer"); err != nil {
		t.Fatalf("ReadText failed: %v", err)
	}
	if page.gotArg != "#answer" {
		t.Errorf("selector not passed to probe: got %v", page.gotArg)
	}
}

func TestReadText_OnClosedSession(t *testing.T) {
	sess, _ := newTextSession("x")
	_ = sess.Close()
	if _, err := sess.ReadText(context.Background(), ""); err == nil {
		t.Errorf("expected error on closed session")
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
