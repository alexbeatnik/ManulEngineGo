package runtime

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alexbeatnik/ManulEngineGo/pkg/config"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
	"github.com/alexbeatnik/ManulEngineGo/pkg/utils"
)

func TestRuntime_ExplainMetadataForClick(t *testing.T) {
	mock := &MockPage{
		URL: "https://example.com/start",
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/button[1]", Tag: "button", VisibleText: "Save", IsVisible: true, Rect: dom.Rect{Top: 10, Left: 20, Width: 100, Height: 30}},
		},
	}
	mock.Elements[0].Normalize()

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	result, err := rt.RunHunt(context.Background(), &dsl.Hunt{
		Commands: []dsl.Command{{Type: dsl.CmdClick, Raw: "CLICK the 'Save' button", Target: "Save", TypeHint: "button"}},
	})
	if err != nil {
		t.Fatalf("RunHunt failed: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	step := result.Results[0]
	if !step.TargetRequired {
		t.Fatal("expected click to require target resolution")
	}
	if step.TargetQuery != "Save" {
		t.Fatalf("TargetQuery = %q, want Save", step.TargetQuery)
	}
	if step.CandidatesConsidered != 1 {
		t.Fatalf("CandidatesConsidered = %d, want 1", step.CandidatesConsidered)
	}
	if step.WinnerXPath != "/button[1]" {
		t.Fatalf("WinnerXPath = %q, want /button[1]", step.WinnerXPath)
	}
	if step.WinnerScore <= 0 {
		t.Fatalf("WinnerScore = %.3f, want > 0", step.WinnerScore)
	}
	if step.ActionPerformed != "click" {
		t.Fatalf("ActionPerformed = %q, want click", step.ActionPerformed)
	}
	if len(step.RankedCandidates) == 0 {
		t.Fatal("expected ranked candidates to be captured")
	}
	if step.ProbeMetadata == nil || step.ProbeMetadata["resolution_strategy"] == nil {
		t.Fatal("expected resolution strategy metadata")
	}
}

func TestRuntime_UploadFile(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 7, XPath: "/input[1]", Tag: "input", InputType: "file", AriaLabel: "Profile Picture", IsVisible: true, Rect: dom.Rect{Top: 10, Left: 20, Width: 120, Height: 30}},
		},
	}
	mock.Elements[0].Normalize()

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:           dsl.CmdUploadFile,
		Raw:            "UPLOAD 'avatar.png' to 'Profile Picture'",
		Target:         "Profile Picture",
		UploadFilePath: "avatar.png",
	})
	if err != nil {
		t.Fatalf("executeCommand failed: %v", err)
	}
	files := mock.FileInputs["/input[1]"]
	if len(files) != 1 || files[0] != "avatar.png" {
		t.Fatalf("uploaded files = %v, want [avatar.png]", files)
	}
	if res.ActionValue != "avatar.png" {
		t.Fatalf("ActionValue = %q, want avatar.png", res.ActionValue)
	}
	if res.WinnerXPath != "/input[1]" {
		t.Fatalf("WinnerXPath = %q, want /input[1]", res.WinnerXPath)
	}
	if !res.TargetRequired {
		t.Fatal("expected upload to require target resolution")
	}
}

func TestRuntime_SnapshotCacheUsedForRepeatedReadOnlyLookups(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/button[1]", Tag: "button", VisibleText: "Save", IsVisible: true},
		},
	}
	mock.Elements[0].Normalize()

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	ctx := context.Background()

	matched, err := rt.evaluateCondition(ctx, "button 'Save' exists")
	if err != nil {
		t.Fatalf("first evaluateCondition failed: %v", err)
	}
	if !matched {
		t.Fatal("expected first condition to match")
	}
	matched, err = rt.evaluateCondition(ctx, "text 'Save' is present")
	if err != nil {
		t.Fatalf("second evaluateCondition failed: %v", err)
	}
	if !matched {
		t.Fatal("expected second condition to match")
	}
	if mock.ProbeCalls != 1 {
		t.Fatalf("ProbeCalls = %d, want 1 with cache enabled", mock.ProbeCalls)
	}
}

func TestRuntime_DisableCacheForcesFreshSnapshotProbe(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/button[1]", Tag: "button", VisibleText: "Save", IsVisible: true},
		},
	}
	mock.Elements[0].Normalize()

	rt := New(config.Config{DisableCache: true}, mock, utils.NewLogger(nil))
	ctx := context.Background()

	matched, err := rt.evaluateCondition(ctx, "button 'Save' exists")
	if err != nil {
		t.Fatalf("first evaluateCondition failed: %v", err)
	}
	if !matched {
		t.Fatal("expected first condition to match")
	}
	matched, err = rt.evaluateCondition(ctx, "text 'Save' is present")
	if err != nil {
		t.Fatalf("second evaluateCondition failed: %v", err)
	}
	if !matched {
		t.Fatal("expected second condition to match")
	}
	if mock.ProbeCalls != 2 {
		t.Fatalf("ProbeCalls = %d, want 2 with cache disabled", mock.ProbeCalls)
	}
}

func TestRuntime_NavigateInvalidatesSnapshotCache(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/button[1]", Tag: "button", VisibleText: "Save", IsVisible: true},
		},
	}
	mock.Elements[0].Normalize()

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	ctx := context.Background()

	matched, err := rt.evaluateCondition(ctx, "button 'Save' exists")
	if err != nil {
		t.Fatalf("initial evaluateCondition failed: %v", err)
	}
	if !matched {
		t.Fatal("expected initial condition to match")
	}
	if mock.ProbeCalls != 1 {
		t.Fatalf("ProbeCalls after first read = %d, want 1", mock.ProbeCalls)
	}

	_, err = rt.executeCommand(ctx, dsl.Command{Type: dsl.CmdNavigate, Raw: "NAVIGATE to https://example.com/dashboard", URL: "https://example.com/dashboard"})
	if err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	matched, err = rt.evaluateCondition(ctx, "button 'Save' exists")
	if err != nil {
		t.Fatalf("post-navigate evaluateCondition failed: %v", err)
	}
	if !matched {
		t.Fatal("expected post-navigate condition to match")
	}
	if mock.ProbeCalls != 2 {
		t.Fatalf("ProbeCalls after navigate = %d, want 2 because navigation should invalidate cache", mock.ProbeCalls)
	}
	if mock.LastNavigate != "https://example.com/dashboard" {
		t.Fatalf("LastNavigate = %q, want https://example.com/dashboard", mock.LastNavigate)
	}
}

func TestRuntime_ClickInvalidatesSnapshotCache(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/button[1]", Tag: "button", VisibleText: "Save", IsVisible: true, Rect: dom.Rect{Top: 10, Left: 20, Width: 100, Height: 30}},
		},
	}
	mock.Elements[0].Normalize()

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	ctx := context.Background()

	matched, err := rt.evaluateCondition(ctx, "button 'Save' exists")
	if err != nil {
		t.Fatalf("initial evaluateCondition failed: %v", err)
	}
	if !matched {
		t.Fatal("expected initial condition to match")
	}
	if mock.ProbeCalls != 1 {
		t.Fatalf("ProbeCalls after first read = %d, want 1", mock.ProbeCalls)
	}

	_, err = rt.executeCommand(ctx, dsl.Command{Type: dsl.CmdClick, Raw: "CLICK the 'Save' button", Target: "Save", TypeHint: "button"})
	if err != nil {
		t.Fatalf("click failed: %v", err)
	}

	matched, err = rt.evaluateCondition(ctx, "button 'Save' exists")
	if err != nil {
		t.Fatalf("post-click evaluateCondition failed: %v", err)
	}
	if !matched {
		t.Fatal("expected post-click condition to match")
	}
	if mock.ProbeCalls != 2 {
		t.Fatalf("ProbeCalls after click = %d, want 2 because click should invalidate cache", mock.ProbeCalls)
	}
	if len(mock.Clicks) != 1 {
		t.Fatalf("expected 1 click, got %d", len(mock.Clicks))
	}
}

func TestResolveRestrictiveCandidatesPrefersAnchorWithNearbyControl(t *testing.T) {
	elements := []dom.ElementSnapshot{
		{ID: 1, XPath: "/html/body/div[1]/table[1]/tbody[1]/tr[1]/td[1]", Tag: "td", VisibleText: "7", IsVisible: true, Rect: dom.Rect{Top: 100, Left: 100, Width: 30, Height: 20}},
		{ID: 2, XPath: "/html/body/div[1]/table[2]/tbody[1]/tr[1]/td[1]", Tag: "td", VisibleText: "7", IsVisible: true, Rect: dom.Rect{Top: 200, Left: 100, Width: 30, Height: 20}},
		{ID: 3, XPath: "/html/body/div[1]/table[2]/tbody[1]/tr[1]/td[3]/input[1]", Tag: "input", InputType: "checkbox", IsVisible: true, Rect: dom.Rect{Top: 200, Left: 180, Width: 20, Height: 20}},
	}
	for i := range elements {
		elements[i].Normalize()
	}

	ranked, strategy := resolveRestrictiveCandidates("7", "checkbox", dsl.ModeCheckbox, elements, nil, nil)
	if strategy != "restrictive-pass1" && strategy != "restrictive-pass3" && strategy != "restrictive-pass3-row" {
		t.Fatalf("strategy = %q, want restrictive-pass1, restrictive-pass3 or restrictive-pass3-row", strategy)
	}
	if len(ranked) == 0 || ranked[0].Element.ID != 3 {
		t.Fatalf("winner = %+v, want checkbox ID 3", ranked)
	}
}

func TestResolveRestrictiveCandidatesPrefersSameRowCheckbox(t *testing.T) {
	elements := []dom.ElementSnapshot{
		{ID: 1, XPath: "/html/body/table[1]/tbody[1]/tr[8]/td[1]", Tag: "td", VisibleText: "7", IsVisible: true, Rect: dom.Rect{Top: 1200, Left: -9000, Width: 30, Height: 20}},
		{ID: 2, XPath: "/html/body/table[2]/tbody[1]/tr[2]/td[1]", Tag: "td", VisibleText: "7", IsVisible: true, Rect: dom.Rect{Top: 200, Left: 100, Width: 30, Height: 20}},
		{ID: 3, XPath: "/html/body/table[2]/tbody[1]/tr[2]/td[4]/input[1]", Tag: "input", InputType: "checkbox", IsVisible: true, Rect: dom.Rect{Top: 200, Left: 180, Width: 20, Height: 20}},
		{ID: 4, XPath: "/html/body/div[1]/input[1]", Tag: "input", InputType: "checkbox", IsVisible: true, Rect: dom.Rect{Top: 1210, Left: -8960, Width: 20, Height: 20}},
	}
	for i := range elements {
		elements[i].Normalize()
	}

	ranked, _ := resolveRestrictiveCandidates("7", "checkbox", dsl.ModeCheckbox, elements, nil, nil)
	if len(ranked) == 0 || ranked[0].Element.ID != 3 {
		t.Fatalf("winner = %+v, want same-row checkbox ID 3", ranked)
	}
}

func TestIsGenericListContainer(t *testing.T) {
	for _, input := range []string{"list", "the list", "dropdown", "dropdown list", "listbox"} {
		if !isGenericListContainer(input) {
			t.Fatalf("expected %q to be treated as a generic list container", input)
		}
	}
	if isGenericListContainer("Country") {
		t.Fatal("did not expect Country to be treated as a generic list container")
	}
}

func TestRuntime_VerifyCheckedFailsForUncheckedControl(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/table[1]/tr[1]/td[1]", Tag: "td", VisibleText: "7", IsVisible: true, Rect: dom.Rect{Top: 100, Left: 100, Width: 30, Height: 20}},
			{ID: 2, XPath: "/html/body/table[1]/tr[1]/td[4]/input[1]", Tag: "input", InputType: "checkbox", LabelText: "Select", IsVisible: true, IsChecked: false, Rect: dom.Rect{Top: 100, Left: 180, Width: 20, Height: 20}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	_, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:        dsl.CmdVerifyField,
		Raw:         "VERIFY that '7' is checked",
		VerifyText:  "7",
		VerifyState: "checked",
	})
	if err == nil {
		t.Fatal("expected verify checked to fail for unchecked control")
	}
}

func TestRuntime_CheckThenVerifyCheckedPasses(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/table[1]/tr[1]/td[1]", Tag: "td", VisibleText: "7", IsVisible: true, Rect: dom.Rect{Top: 100, Left: 100, Width: 30, Height: 20}},
			{ID: 2, XPath: "/html/body/table[1]/tr[1]/td[4]/input[1]", Tag: "input", InputType: "checkbox", AriaLabel: "7", LabelText: "7", IsVisible: true, IsChecked: false, Rect: dom.Rect{Top: 100, Left: 180, Width: 20, Height: 20}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	_, err := rt.RunHunt(context.Background(), &dsl.Hunt{
		Commands: []dsl.Command{
			{Type: dsl.CmdCheck, Raw: "CHECK the checkbox for '7'", Target: "7", TypeHint: "checkbox"},
			{Type: dsl.CmdVerifyField, Raw: "VERIFY that '7' is checked", VerifyText: "7", VerifyState: "checked"},
		},
	})
	if err != nil {
		t.Fatalf("expected hunt to pass after checking control, got %v", err)
	}
	if !mock.Elements[1].IsChecked {
		t.Fatal("expected mock checkbox state to be updated by CHECK")
	}
}

func TestRuntime_ReconcileStickyCheckboxStateReappliesVisibleRow(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/table[1]/tr[1]/td[1]", Tag: "td", VisibleText: "7", IsVisible: true, Rect: dom.Rect{Top: 100, Left: 100, Width: 30, Height: 20}},
			{ID: 2, XPath: "/html/body/table[1]/tr[1]/td[4]/input[1]", Tag: "input", InputType: "checkbox", AriaLabel: "7", LabelText: "7", IsVisible: true, IsChecked: false, Rect: dom.Rect{Top: 100, Left: 180, Width: 20, Height: 20}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	rt.rememberStickyCheckboxState("7", true)
	if err := rt.reconcileStickyCheckboxStates(context.Background()); err != nil {
		t.Fatalf("reconcileStickyCheckboxStates failed: %v", err)
	}
	if !mock.Elements[1].IsChecked {
		t.Fatal("expected reconcile to reapply checked state for visible row")
	}
}

func TestRuntime_ClickPrefersExactAriaLabel(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/button[1]", Tag: "button", AriaLabel: "Expand all", IsVisible: true, Rect: dom.Rect{Top: 20, Left: 20, Width: 24, Height: 24}},
			{ID: 2, XPath: "/html/body/div[1]/button[2]", Tag: "button", AriaLabel: "Collapse all", IsVisible: true, Rect: dom.Rect{Top: 20, Left: 70, Width: 24, Height: 24}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	_, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdClick,
		Raw:      "CLICK the 'Expand all' button",
		Target:   "Expand all",
		TypeHint: "button",
	})
	if err != nil {
		t.Fatalf("click failed: %v", err)
	}
	if len(mock.Clicks) != 1 {
		t.Fatalf("expected 1 click, got %d", len(mock.Clicks))
	}
	if mock.Clicks[0].X != 32 || mock.Clicks[0].Y != 32 {
		t.Fatalf("clicked (%v,%v), want center of Expand all button (32,32)", mock.Clicks[0].X, mock.Clicks[0].Y)
	}
}

func TestRuntime_ClickNearAnchorChoosesClosestCandidate(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/span[1]", Tag: "span", VisibleText: "Quantity", IsVisible: true, Rect: dom.Rect{Top: 100, Left: 100, Width: 80, Height: 20}},
			{ID: 2, XPath: "/html/body/div[1]/button[1]", Tag: "button", VisibleText: "Increase", AriaLabel: "Increase Quantity", IsVisible: true, Rect: dom.Rect{Top: 100, Left: 200, Width: 24, Height: 24}},
			{ID: 3, XPath: "/html/body/div[2]/button[1]", Tag: "button", VisibleText: "Increase", AriaLabel: "Increase Adults", IsVisible: true, Rect: dom.Rect{Top: 100, Left: 500, Width: 24, Height: 24}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	_, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:       dsl.CmdClick,
		Raw:        "CLICK the 'Increase' button NEAR 'Quantity'",
		Target:     "Increase",
		TypeHint:   "button",
		NearAnchor: "Quantity",
	})
	if err != nil {
		t.Fatalf("click with near anchor failed: %v", err)
	}
	if len(mock.Clicks) != 1 {
		t.Fatalf("expected 1 click, got %d", len(mock.Clicks))
	}
	if mock.Clicks[0].X != 212 || mock.Clicks[0].Y != 112 {
		t.Fatalf("clicked (%v,%v), want center of near candidate (212,112)", mock.Clicks[0].X, mock.Clicks[0].Y)
	}
}

func TestRuntime_ClickNearAnchorSameCardBeatsCloserNeighbor(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/div[4]/div[1]/a/div[1]", Tag: "div", VisibleText: "Sauce Labs Fleece Jacket", IsVisible: true, Rect: dom.Rect{Top: 100, Left: 420, Width: 140, Height: 30}},
			{ID: 2, XPath: "/html/body/div[1]/div[4]/div[2]/button[1]", Tag: "button", VisibleText: "Add to cart", HTMLId: "add-to-cart-sauce-labs-fleece-jacket", IsVisible: true, Rect: dom.Rect{Top: 112, Left: 565, Width: 105, Height: 30}},
			{ID: 3, XPath: "/html/body/div[1]/div[3]/div[2]/button[1]", Tag: "button", VisibleText: "Add to cart", HTMLId: "add-to-cart-sauce-labs-backpack", IsVisible: true, Rect: dom.Rect{Top: 106, Left: 360, Width: 105, Height: 30}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:       dsl.CmdClick,
		Raw:        "CLICK the 'Add to cart' button NEAR 'Sauce Labs Fleece Jacket'",
		Target:     "Add to cart",
		TypeHint:   "button",
		NearAnchor: "Sauce Labs Fleece Jacket",
	})
	if err != nil {
		t.Fatalf("click with near anchor failed: %v", err)
	}
	if len(mock.Clicks) != 1 {
		t.Fatalf("expected 1 click, got %d", len(mock.Clicks))
	}
	if res.WinnerXPath != "/html/body/div[1]/div[4]/div[2]/button[1]" {
		t.Fatalf("WinnerXPath = %q, want same-card button xpath", res.WinnerXPath)
	}
	if mock.Clicks[0].X != 617.5 || mock.Clicks[0].Y != 127 {
		t.Fatalf("clicked (%v,%v), want center of same-card button (617.5,127)", mock.Clicks[0].X, mock.Clicks[0].Y)
	}
	if res.ProbeMetadata["near_anchor"] != "Sauce Labs Fleece Jacket" {
		t.Fatalf("near_anchor metadata = %v, want Sauce Labs Fleece Jacket", res.ProbeMetadata["near_anchor"])
	}
}

func TestRuntime_ClickNearAnchorPrefersCandidateInSameFrame(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/span[1]", Tag: "span", VisibleText: "Widget Controls", IsVisible: true, FrameIndex: 1, Rect: dom.Rect{Top: 100, Left: 100, Width: 120, Height: 20}},
			{ID: 2, XPath: "/html/body/button[1]", Tag: "button", VisibleText: "Save", HTMLId: "main_save", IsVisible: true, FrameIndex: 0, Rect: dom.Rect{Top: 105, Left: 230, Width: 80, Height: 24}},
			{ID: 3, XPath: "/html/body/button[2]", Tag: "button", VisibleText: "Save", HTMLId: "iframe_save", IsVisible: true, FrameIndex: 1, Rect: dom.Rect{Top: 105, Left: 230, Width: 80, Height: 24}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:       dsl.CmdClick,
		Raw:        "CLICK the 'Save' button NEAR 'Widget Controls'",
		Target:     "Save",
		TypeHint:   "button",
		NearAnchor: "Widget Controls",
	})
	if err != nil {
		t.Fatalf("click with iframe near anchor failed: %v", err)
	}
	if res.WinnerXPath != "/html/body/button[2]" {
		t.Fatalf("WinnerXPath = %q, want same-frame iframe button xpath", res.WinnerXPath)
	}
	if len(mock.Clicks) != 1 {
		t.Fatalf("expected 1 click, got %d", len(mock.Clicks))
	}
	if mock.Clicks[0].X != 270 || mock.Clicks[0].Y != 117 {
		t.Fatalf("clicked (%v,%v), want iframe button center (270,117)", mock.Clicks[0].X, mock.Clicks[0].Y)
	}
}

func TestRuntime_ClickNearAnchorFailsWhenAnchorMissing(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/button[1]", Tag: "button", VisibleText: "Save", IsVisible: true, Rect: dom.Rect{Top: 40, Left: 40, Width: 90, Height: 30}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	_, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:       dsl.CmdClick,
		Raw:        "CLICK the 'Save' button NEAR 'Missing Anchor'",
		Target:     "Save",
		TypeHint:   "button",
		NearAnchor: "Missing Anchor",
	})
	if err == nil {
		t.Fatal("expected click with missing near anchor to fail")
	}
	if got := err.Error(); !strings.Contains(got, `near anchor not found: "Missing Anchor"`) {
		t.Fatalf("error = %q, want missing near anchor message", got)
	}
	if len(mock.Clicks) != 0 {
		t.Fatalf("expected no click when near anchor is missing, got %d", len(mock.Clicks))
	}
}

func TestRuntime_ClickOnHeaderPrefersHeaderCandidate(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/header[1]/a[1]", Tag: "a", VisibleText: "Login", Ancestors: []string{"header", "body", "html"}, IsVisible: true, Rect: dom.Rect{Top: 20, Left: 20, Width: 80, Height: 24}},
			{ID: 2, XPath: "/html/body/main[1]/a[1]", Tag: "a", VisibleText: "Login", Ancestors: []string{"main", "body", "html"}, IsVisible: true, Rect: dom.Rect{Top: 420, Left: 20, Width: 80, Height: 24}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdClick,
		Raw:      "CLICK the 'Login' link ON HEADER",
		Target:   "Login",
		TypeHint: "link",
		OnRegion: "HEADER",
	})
	if err != nil {
		t.Fatalf("click on header failed: %v", err)
	}
	if res.WinnerXPath != "/html/body/header[1]/a[1]" {
		t.Fatalf("WinnerXPath = %q, want header link xpath", res.WinnerXPath)
	}
	if res.ProbeMetadata["on_region"] != "HEADER" {
		t.Fatalf("on_region metadata = %v, want HEADER", res.ProbeMetadata["on_region"])
	}
	if len(mock.Clicks) != 1 {
		t.Fatalf("expected 1 click, got %d", len(mock.Clicks))
	}
	if mock.Clicks[0].X != 60 || mock.Clicks[0].Y != 32 {
		t.Fatalf("clicked (%v,%v), want header link center (60,32)", mock.Clicks[0].X, mock.Clicks[0].Y)
	}
}

func TestRuntime_ClickOnFooterPrefersFooterCandidate(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/footer[1]/a[1]", Tag: "a", VisibleText: "Privacy Policy", Ancestors: []string{"footer", "body", "html"}, IsVisible: true, Rect: dom.Rect{Top: 930, Left: 20, Width: 120, Height: 24}},
			{ID: 2, XPath: "/html/body/main[1]/a[1]", Tag: "a", VisibleText: "Privacy Policy", Ancestors: []string{"main", "body", "html"}, IsVisible: true, Rect: dom.Rect{Top: 80, Left: 20, Width: 120, Height: 24}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdClick,
		Raw:      "CLICK the 'Privacy Policy' link ON FOOTER",
		Target:   "Privacy Policy",
		TypeHint: "link",
		OnRegion: "FOOTER",
	})
	if err != nil {
		t.Fatalf("click on footer failed: %v", err)
	}
	if res.WinnerXPath != "/html/body/footer[1]/a[1]" {
		t.Fatalf("WinnerXPath = %q, want footer link xpath", res.WinnerXPath)
	}
	if len(mock.Clicks) != 1 {
		t.Fatalf("expected 1 click, got %d", len(mock.Clicks))
	}
	if mock.Clicks[0].X != 80 || mock.Clicks[0].Y != 942 {
		t.Fatalf("clicked (%v,%v), want footer link center (80,942)", mock.Clicks[0].X, mock.Clicks[0].Y)
	}
}

func TestRuntime_ClickInsideRowChoosesCandidateFromMatchingRow(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/table[1]/tbody[1]/tr[1]/td[1]", Tag: "td", VisibleText: "Jane", IsVisible: true, Rect: dom.Rect{Top: 120, Left: 40, Width: 80, Height: 24}},
			{ID: 2, XPath: "/html/body/table[1]/tbody[1]/tr[1]/td[4]/button[1]", Tag: "button", VisibleText: "Delete", IsVisible: true, Rect: dom.Rect{Top: 120, Left: 260, Width: 80, Height: 24}},
			{ID: 3, XPath: "/html/body/table[1]/tbody[1]/tr[2]/td[1]", Tag: "td", VisibleText: "John", IsVisible: true, Rect: dom.Rect{Top: 180, Left: 40, Width: 80, Height: 24}},
			{ID: 4, XPath: "/html/body/table[1]/tbody[1]/tr[2]/td[4]/button[1]", Tag: "button", VisibleText: "Delete", IsVisible: true, Rect: dom.Rect{Top: 180, Left: 260, Width: 80, Height: 24}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:          dsl.CmdClick,
		Raw:           "CLICK the 'Delete' button INSIDE 'Actions' row with 'John'",
		Target:        "Delete",
		TypeHint:      "button",
		InsideRowText: "John",
	})
	if err != nil {
		t.Fatalf("click inside row failed: %v", err)
	}
	if res.WinnerXPath != "/html/body/table[1]/tbody[1]/tr[2]/td[4]/button[1]" {
		t.Fatalf("WinnerXPath = %q, want John row button xpath", res.WinnerXPath)
	}
	if res.ProbeMetadata["inside_row_text"] != "John" {
		t.Fatalf("inside_row_text metadata = %v, want John", res.ProbeMetadata["inside_row_text"])
	}
}

func TestRuntime_ClickInsideContainerChoosesDescendant(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/section[1]", Tag: "section", VisibleText: "Checkout Summary", IsVisible: true, Rect: dom.Rect{Top: 100, Left: 40, Width: 320, Height: 180}},
			{ID: 2, XPath: "/html/body/div[1]/section[1]/button[1]", Tag: "button", VisibleText: "Edit", IsVisible: true, Rect: dom.Rect{Top: 220, Left: 60, Width: 80, Height: 24}},
			{ID: 3, XPath: "/html/body/div[1]/section[2]", Tag: "section", VisibleText: "Shipping", IsVisible: true, Rect: dom.Rect{Top: 320, Left: 40, Width: 320, Height: 180}},
			{ID: 4, XPath: "/html/body/div[1]/section[2]/button[1]", Tag: "button", VisibleText: "Edit", IsVisible: true, Rect: dom.Rect{Top: 440, Left: 60, Width: 80, Height: 24}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:            dsl.CmdClick,
		Raw:             "CLICK the 'Edit' button INSIDE 'Checkout Summary'",
		Target:          "Edit",
		TypeHint:        "button",
		InsideContainer: "Checkout Summary",
	})
	if err != nil {
		t.Fatalf("click inside container failed: %v", err)
	}
	if res.WinnerXPath != "/html/body/div[1]/section[1]/button[1]" {
		t.Fatalf("WinnerXPath = %q, want checkout summary button xpath", res.WinnerXPath)
	}
	if res.ProbeMetadata["inside_container"] != "Checkout Summary" {
		t.Fatalf("inside_container metadata = %v, want Checkout Summary", res.ProbeMetadata["inside_container"])
	}
}

func TestRuntime_FillPrefersInputByLabelOverMatchingButtonText(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/button[1]", Tag: "button", VisibleText: "Email Address", IsVisible: true, Rect: dom.Rect{Top: 20, Left: 20, Width: 140, Height: 32}},
			{ID: 2, XPath: "/html/body/div[1]/input[1]", Tag: "input", InputType: "text", LabelText: "Email Address", IsVisible: true, IsEditable: true, Rect: dom.Rect{Top: 70, Left: 20, Width: 220, Height: 32}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:   dsl.CmdFill,
		Raw:    "FILL 'Email Address' field with 'alex@manul.dev'",
		Target: "Email Address",
		Value:  "alex@manul.dev",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	if got := mock.Inputs["/html/body/div[1]/input[1]"]; got != "alex@manul.dev" {
		t.Fatalf("input value = %q, want alex@manul.dev", got)
	}
	if res.WinnerXPath != "/html/body/div[1]/input[1]" {
		t.Fatalf("WinnerXPath = %q, want input xpath", res.WinnerXPath)
	}
	if res.ProbeMetadata["interaction_mode"] != string(dsl.ModeInput) {
		t.Fatalf("interaction_mode = %v, want %s", res.ProbeMetadata["interaction_mode"], dsl.ModeInput)
	}
	if _, clickedButton := mock.Inputs["/html/body/div[1]/button[1]"]; clickedButton {
		t.Fatal("did not expect non-input candidate to be used for fill")
	}
}

func TestRuntime_FillUsesPlaceholderWhenNoLabelExists(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/button[1]", Tag: "button", VisibleText: "Search Products", IsVisible: true, Rect: dom.Rect{Top: 20, Left: 20, Width: 150, Height: 32}},
			{ID: 2, XPath: "/html/body/div[1]/input[1]", Tag: "input", InputType: "text", Placeholder: "Search Products", IsVisible: true, IsEditable: true, Rect: dom.Rect{Top: 70, Left: 20, Width: 240, Height: 32}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:   dsl.CmdFill,
		Raw:    "FILL 'Search Products' field with 'backpack'",
		Target: "Search Products",
		Value:  "backpack",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	if got := mock.Inputs["/html/body/div[1]/input[1]"]; got != "backpack" {
		t.Fatalf("input value = %q, want backpack", got)
	}
	if res.WinnerXPath != "/html/body/div[1]/input[1]" {
		t.Fatalf("WinnerXPath = %q, want placeholder-matched input", res.WinnerXPath)
	}
}

func TestRuntime_FillUsesFieldsetLegendAsLabel(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/input[1]", Tag: "input", InputType: "text", LabelText: "Suggession Class Example", IsVisible: true, IsEditable: true, Rect: dom.Rect{Top: 70, Left: 20, Width: 240, Height: 32}},
			{ID: 2, XPath: "/html/body/div[1]/input[2]", Tag: "input", InputType: "text", LabelText: "Enter Your Name", IsVisible: true, IsEditable: true, Rect: dom.Rect{Top: 130, Left: 20, Width: 240, Height: 32}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:   dsl.CmdFill,
		Raw:    "FILL 'Suggession Class' field with 'Ukra'",
		Target: "Suggession Class",
		Value:  "Ukra",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	if got := mock.Inputs["/html/body/div[1]/input[1]"]; got != "Ukra" {
		t.Fatalf("fieldset input value = %q, want Ukra", got)
	}
	if _, otherUsed := mock.Inputs["/html/body/div[1]/input[2]"]; otherUsed {
		t.Fatal("did not expect unrelated input to be used")
	}
	if res.WinnerXPath != "/html/body/div[1]/input[1]" {
		t.Fatalf("WinnerXPath = %q, want fieldset legend matched input", res.WinnerXPath)
	}
	if res.ProbeMetadata["interaction_mode"] != string(dsl.ModeInput) {
		t.Fatalf("interaction_mode = %v, want %s", res.ProbeMetadata["interaction_mode"], dsl.ModeInput)
	}
}

type delayedAutocompletePage struct {
	*MockPage
	pendingSuggestion bool
}

func (p *delayedAutocompletePage) SetInputValue(ctx context.Context, id int, xpath, value string) error {
	if err := p.MockPage.SetInputValue(ctx, id, xpath, value); err != nil {
		return err
	}
	p.pendingSuggestion = true
	return nil
}

func (p *delayedAutocompletePage) Wait(ctx context.Context, d time.Duration) error {
	if p.pendingSuggestion {
		p.pendingSuggestion = false
		p.Elements = append(p.Elements, dom.ElementSnapshot{
			ID:          2,
			XPath:       "/html/body/ul[1]/li[1]/div[1]",
			Tag:         "div",
			VisibleText: "Ukraine",
			IsVisible:   true,
			Rect:        dom.Rect{Top: 120, Left: 20, Bottom: 150, Right: 220, Width: 200, Height: 30},
		})
		p.Elements[len(p.Elements)-1].Normalize()
	}
	return nil
}

func TestRuntime_FillWaitsForReactiveAutocompleteSuggestions(t *testing.T) {
	page := &delayedAutocompletePage{MockPage: &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/input[1]", Tag: "input", InputType: "text", LabelText: "Suggession Class Example", IsVisible: true, IsEditable: true, Rect: dom.Rect{Top: 70, Left: 20, Bottom: 102, Right: 260, Width: 240, Height: 32}},
		},
	}}
	for i := range page.Elements {
		page.Elements[i].Normalize()
	}

	rt := New(config.Config{}, page, utils.NewLogger(nil))
	if _, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:   dsl.CmdFill,
		Raw:    "FILL 'Suggession Class' field with 'Ukra'",
		Target: "Suggession Class",
		Value:  "Ukra",
	}); err != nil {
		t.Fatalf("fill failed: %v", err)
	}

	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdClick,
		Raw:      "CLICK the 'Ukraine' element",
		Target:   "Ukraine",
		TypeHint: "element",
	})
	if err != nil {
		t.Fatalf("click failed: %v", err)
	}
	if len(page.Clicks) != 1 {
		t.Fatalf("expected 1 click, got %d", len(page.Clicks))
	}
	if res.WinnerXPath != "/html/body/ul[1]/li[1]/div[1]" {
		t.Fatalf("WinnerXPath = %q, want revealed autocomplete suggestion", res.WinnerXPath)
	}
}

type delayedHoverMenuPage struct {
	*MockPage
	pendingReveal bool
}

func (p *delayedHoverMenuPage) Hover(ctx context.Context, x, y float64) error {
	p.Clicks = append(p.Clicks, Point{X: x, Y: y})
	p.pendingReveal = true
	return nil
}

func (p *delayedHoverMenuPage) Wait(ctx context.Context, d time.Duration) error {
	if p.pendingReveal {
		p.pendingReveal = false
		p.Elements = append(p.Elements, dom.ElementSnapshot{
			ID:          2,
			XPath:       "/html/body/div[1]/a[1]",
			Tag:         "a",
			VisibleText: "Top",
			IsVisible:   true,
			Rect:        dom.Rect{Top: 120, Left: 20, Bottom: 165, Right: 180, Width: 160, Height: 45},
		})
		p.Elements[len(p.Elements)-1].Normalize()
	}
	return nil
}

func TestRuntime_HoverWaitsForReactiveMenuReveal(t *testing.T) {
	page := &delayedHoverMenuPage{MockPage: &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/button[1]", Tag: "button", VisibleText: "Mouse Hover", IsVisible: true, Rect: dom.Rect{Top: 70, Left: 20, Bottom: 102, Right: 160, Width: 140, Height: 32}},
		},
	}}
	for i := range page.Elements {
		page.Elements[i].Normalize()
	}

	rt := New(config.Config{}, page, utils.NewLogger(nil))
	if _, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdHover,
		Raw:      "HOVER over the 'Mouse Hover' button",
		Target:   "Mouse Hover",
		TypeHint: "button",
	}); err != nil {
		t.Fatalf("hover failed: %v", err)
	}

	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdClick,
		Raw:      "CLICK on the 'Top' link",
		Target:   "Top",
		TypeHint: "link",
	})
	if err != nil {
		t.Fatalf("click failed: %v", err)
	}
	if res.WinnerXPath != "/html/body/div[1]/a[1]" {
		t.Fatalf("WinnerXPath = %q, want revealed hover-menu link", res.WinnerXPath)
	}
}

func TestRuntime_ClickCollapsesNestedAutocompleteWrappers(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/ul[1]/li[1]", Tag: "li", VisibleText: "Ukraine", IsVisible: true, Rect: dom.Rect{Top: 121, Left: 21, Bottom: 151, Right: 219, Width: 198, Height: 30}},
			{ID: 2, XPath: "/html/body/ul[1]/li[1]/div[1]", Tag: "div", VisibleText: "Ukraine", IsVisible: true, Rect: dom.Rect{Top: 121, Left: 21, Bottom: 151, Right: 219, Width: 198, Height: 30}},
			{ID: 3, XPath: "/html/body/div[2]/a[1]", Tag: "a", VisibleText: "Discount Coupons", IsVisible: true, Rect: dom.Rect{Top: 300, Left: 21, Bottom: 331, Right: 249, Width: 228, Height: 31}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdClick,
		Raw:      "CLICK the 'Ukraine' element",
		Target:   "Ukraine",
		TypeHint: "element",
	})
	if err != nil {
		t.Fatalf("click failed: %v", err)
	}
	if len(mock.Clicks) != 1 {
		t.Fatalf("expected 1 click, got %d", len(mock.Clicks))
	}
	if res.WinnerXPath != "/html/body/ul[1]/li[1]/div[1]" {
		t.Fatalf("WinnerXPath = %q, want deepest autocomplete candidate", res.WinnerXPath)
	}
}

func TestRuntime_FillPrefersVisibleInputOverHiddenExactMatch(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/input[1]", Tag: "input", InputType: "text", Placeholder: "Search", IsVisible: false, IsHidden: true, IsEditable: true, Rect: dom.Rect{Top: 20, Left: 20, Width: 200, Height: 32}},
			{ID: 2, XPath: "/html/body/div[1]/input[2]", Tag: "input", InputType: "text", Placeholder: "Search", IsVisible: true, IsEditable: true, Rect: dom.Rect{Top: 80, Left: 20, Width: 200, Height: 32}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:   dsl.CmdFill,
		Raw:    "FILL 'Search' field with 'golang'",
		Target: "Search",
		Value:  "golang",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	if got := mock.Inputs["/html/body/div[1]/input[2]"]; got != "golang" {
		t.Fatalf("visible input value = %q, want golang", got)
	}
	if _, hiddenUsed := mock.Inputs["/html/body/div[1]/input[1]"]; hiddenUsed {
		t.Fatal("did not expect hidden exact-match input to be used")
	}
	if res.WinnerXPath != "/html/body/div[1]/input[2]" {
		t.Fatalf("WinnerXPath = %q, want visible input xpath", res.WinnerXPath)
	}
}

func TestRuntime_SelectPrefersNativeSelectOverMatchingButton(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/button[1]", Tag: "button", VisibleText: "Country", IsVisible: true, Rect: dom.Rect{Top: 20, Left: 20, Width: 120, Height: 32}},
			{ID: 2, XPath: "/html/body/div[1]/select[1]", Tag: "select", IsSelect: true, LabelText: "Country", IsVisible: true, Rect: dom.Rect{Top: 80, Left: 20, Width: 180, Height: 32}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdSelect,
		Raw:      "SELECT 'Japan' from the 'Country' dropdown",
		Target:   "Country",
		TypeHint: "dropdown",
		Value:    "Japan",
	})
	if err != nil {
		t.Fatalf("select failed: %v", err)
	}
	if res.WinnerXPath != "/html/body/div[1]/select[1]" {
		t.Fatalf("WinnerXPath = %q, want native select xpath", res.WinnerXPath)
	}
	if res.ProbeMetadata["interaction_mode"] != string(dsl.ModeSelect) {
		t.Fatalf("interaction_mode = %v, want %s", res.ProbeMetadata["interaction_mode"], dsl.ModeSelect)
	}
	if len(mock.Clicks) != 0 {
		t.Fatalf("expected native select path to avoid click fallback, got %d clicks", len(mock.Clicks))
	}
	if res.ActionValue != "Japan" {
		t.Fatalf("ActionValue = %q, want Japan", res.ActionValue)
	}
}

func TestRuntime_CheckPrefersRoleCheckboxOverMatchingButtonText(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/button[1]", Tag: "button", VisibleText: "Email Alerts", IsVisible: true, Rect: dom.Rect{Top: 20, Left: 20, Width: 140, Height: 32}},
			{ID: 2, XPath: "/html/body/div[1]/div[1]", Tag: "div", Role: "checkbox", AriaLabel: "Email Alerts", IsVisible: true, Rect: dom.Rect{Top: 70, Left: 20, Width: 28, Height: 28}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdCheck,
		Raw:      "CHECK the checkbox for 'Email Alerts'",
		Target:   "Email Alerts",
		TypeHint: "checkbox",
	})
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !mock.Elements[1].IsChecked {
		t.Fatal("expected role=checkbox candidate to be checked")
	}
	if res.WinnerXPath != "/html/body/div[1]/div[1]" {
		t.Fatalf("WinnerXPath = %q, want role checkbox xpath", res.WinnerXPath)
	}
	if res.ProbeMetadata["interaction_mode"] != string(dsl.ModeCheckbox) {
		t.Fatalf("interaction_mode = %v, want %s", res.ProbeMetadata["interaction_mode"], dsl.ModeCheckbox)
	}
}

func TestRuntime_CheckPassesForLowConfidenceRoleCheckbox(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/span[1]", Tag: "span", Role: "checkbox", AriaLabel: "Select Home", IsVisible: true, Rect: dom.Rect{Top: 70, Left: 20, Width: 16, Height: 16}},
			{ID: 2, XPath: "/html/body/div[1]/span[2]", Tag: "span", VisibleText: "Home", IsVisible: true, Rect: dom.Rect{Top: 68, Left: 44, Width: 44, Height: 20}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdCheck,
		Raw:      "CHECK the checkbox for 'Home'",
		Target:   "Home",
		TypeHint: "checkbox",
	})
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !mock.Elements[0].IsChecked {
		t.Fatal("expected low-confidence role checkbox candidate to be checked")
	}
	if res.WinnerXPath != "/html/body/div[1]/span[1]" {
		t.Fatalf("WinnerXPath = %q, want role checkbox xpath", res.WinnerXPath)
	}
	if res.WinnerScore <= ThresholdHighConfidence {
		t.Fatalf("WinnerScore = %.3f, want high-confidence checkbox score", res.WinnerScore)
	}
	if res.ProbeMetadata["interaction_mode"] != string(dsl.ModeCheckbox) {
		t.Fatalf("interaction_mode = %v, want %s", res.ProbeMetadata["interaction_mode"], dsl.ModeCheckbox)
	}
	if res.ProbeMetadata["resolution_strategy"] != "restrictive-pass1" {
		t.Fatalf("resolution_strategy = %v, want restrictive-pass1", res.ProbeMetadata["resolution_strategy"])
	}
}

func TestRuntime_VerifyCheckedPassesForLowConfidenceRoleCheckbox(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/span[1]", Tag: "span", Role: "checkbox", AriaLabel: "Select Home", IsVisible: true, IsChecked: true, Rect: dom.Rect{Top: 70, Left: 20, Width: 16, Height: 16}},
			{ID: 2, XPath: "/html/body/div[1]/span[2]", Tag: "span", VisibleText: "Home", IsVisible: true, Rect: dom.Rect{Top: 68, Left: 44, Width: 44, Height: 20}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	res, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:        dsl.CmdVerifyField,
		Raw:         "VERIFY that 'Home' is checked",
		VerifyText:  "Home",
		VerifyState: "checked",
	})
	if err != nil {
		t.Fatalf("verify checked failed: %v", err)
	}
	if res.WinnerXPath != "/html/body/div[1]/span[1]" {
		t.Fatalf("WinnerXPath = %q, want role checkbox xpath", res.WinnerXPath)
	}
	if res.WinnerScore <= ThresholdHighConfidence {
		t.Fatalf("WinnerScore = %.3f, want high-confidence checkbox score", res.WinnerScore)
	}
}

func TestRuntime_ClickFailsForAmbiguousLowConfidenceCandidates(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/div[1]/button[1]", Tag: "button", ClassName: "alpha-trigger", IsVisible: true, Rect: dom.Rect{Top: 20, Left: 20, Width: 120, Height: 32}},
			{ID: 2, XPath: "/html/body/div[1]/button[2]", Tag: "button", ClassName: "alpha-launch", IsVisible: true, Rect: dom.Rect{Top: 20, Left: 180, Width: 120, Height: 32}},
		},
	}
	for i := range mock.Elements {
		mock.Elements[i].Normalize()
	}

	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	_, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:   dsl.CmdClick,
		Raw:    "CLICK 'alpha'",
		Target: "alpha",
	})
	if err == nil {
		t.Fatal("expected low-confidence ambiguous resolution to fail")
	}
	if got := err.Error(); !strings.Contains(got, "target resolution too ambiguous") {
		t.Fatalf("error = %q, want ambiguous resolution message", got)
	}
	if len(mock.Clicks) != 0 {
		t.Fatalf("expected no click on ambiguous resolution, got %d", len(mock.Clicks))
	}
}

type noOpCheckPage struct {
	*MockPage
}

func (p *noOpCheckPage) SetChecked(ctx context.Context, id int, xpath string, checked bool) error {
	return nil
}

func TestRuntime_CheckFailsWhenCheckboxStateDoesNotChange(t *testing.T) {
	base := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, XPath: "/html/body/table[1]/tr[1]/td[1]", Tag: "td", VisibleText: "7", IsVisible: true, Rect: dom.Rect{Top: 100, Left: 100, Width: 30, Height: 20}},
			{ID: 2, XPath: "/html/body/table[1]/tr[1]/td[4]/input[1]", Tag: "input", InputType: "checkbox", LabelText: "Select", IsVisible: true, IsChecked: false, Rect: dom.Rect{Top: 100, Left: 180, Width: 20, Height: 20}},
		},
	}
	for i := range base.Elements {
		base.Elements[i].Normalize()
	}

	rt := New(config.Config{}, &noOpCheckPage{MockPage: base}, utils.NewLogger(nil))
	_, err := rt.executeCommand(context.Background(), dsl.Command{
		Type:     dsl.CmdCheck,
		Raw:      "CHECK the checkbox for '7'",
		Target:   "7",
		TypeHint: "checkbox",
	})
	if err == nil {
		t.Fatal("expected CHECK to fail when checkbox state does not actually change")
	}
}

// VERIFY SOFTLY must never fail the step: missing text produces a warning
// + Success=true so the hunt continues. Regression for the sampler.hunt bug
// where VERIFY SOFTLY was unimplemented and raised "command VERIFY_SOFT not
// yet implemented".
func TestRuntime_VerifySoftly(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		negated bool
		// pageText is what the visible-text probe returns; a VERIFY SOFTLY
		// failure must still leave Success=true.
		pageText string
	}{
		{"missing text, expected present", "ghost-text", false, "welcome to the site"},
		{"present text, expected present", "welcome", false, "welcome to the site"},
		{"present text, expected NOT present", "welcome", true, "welcome to the site"},
		{"missing text, expected NOT present", "ghost-text", true, "welcome to the site"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockPage{URL: "https://example.com", Title: tc.pageText, Elements: []dom.ElementSnapshot{
				{ID: 1, XPath: "/body", Tag: "body", VisibleText: tc.pageText, IsVisible: true},
			}}
			mock.Elements[0].Normalize()
			rt := New(config.Config{}, mock, utils.NewLogger(nil))
			result, err := rt.RunHunt(context.Background(), &dsl.Hunt{
				Commands: []dsl.Command{{
					Type:          dsl.CmdVerifySoft,
					Raw:           "VERIFY SOFTLY that '" + tc.text + "' is present",
					VerifyText:    tc.text,
					VerifyNegated: tc.negated,
				}},
			})
			if err != nil {
				t.Fatalf("RunHunt returned error for soft verify: %v", err)
			}
			if len(result.Results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(result.Results))
			}
			if !result.Results[0].Success {
				t.Fatalf("VERIFY SOFTLY must always mark the step Success=true (got %+v)", result.Results[0])
			}
			if result.Results[0].Error != "" {
				t.Fatalf("VERIFY SOFTLY must not set Error (got %q)", result.Results[0].Error)
			}
			if result.Results[0].TargetQuery != tc.text {
				t.Fatalf("TargetQuery = %q, want %q", result.Results[0].TargetQuery, tc.text)
			}
		})
	}
}
