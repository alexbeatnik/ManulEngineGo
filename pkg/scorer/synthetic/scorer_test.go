package synthetic

import (
	"github.com/alexbeatnik/ManulEngineGo/pkg/scorer"
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

// ── Test: Pagination links ────────────────────────────────────────────────────
// Scenario: a table has rows with numbers, and below it is a pagination nav
// with <a> links: 1, 2, 3, 4, 5. "CLICK on the '2' link" must pick the <a>
// with text "2", not a <td> cell or other element.

func TestPaginationLink_ClickOn2(t *testing.T) {
	elements := []dom.ElementSnapshot{
		// Table cell with "2" as row content
		makeEl(withTag("td"), withText("2"), withXPath("/html/body/table/tr[2]/td[1]")),
		// Pagination link "1"
		makeEl(withTag("a"), withText("1"), withID("page_1"), withXPath("/html/body/nav/a[1]")),
		// Pagination link "2" — this should win
		makeEl(withTag("a"), withText("2"), withID("page_2"), withXPath("/html/body/nav/a[2]")),
		// Pagination link "3"
		makeEl(withTag("a"), withText("3"), withID("page_3"), withXPath("/html/body/nav/a[3]")),
		// A button elsewhere
		makeEl(withTag("button"), withText("Page 2 settings")),
	}

	winner := rankFirst(t, "2", "link", "clickable", elements)
	if winner.Element.Tag != "a" || winner.Element.VisibleText != "2" {
		t.Errorf("expected <a> with text '2', got <%s> with text %q (xpath=%s, score=%.4f)",
			winner.Element.Tag, winner.Element.VisibleText, winner.Element.XPath, winner.Explain.Score.Total)

		// Print all candidates for debugging
		ranked := scorer.Rank("2", "link", "clickable", elements, 10, nil)
		for _, r := range ranked {
			t.Logf("  rank=%d tag=%-8s text=%-20q hint=%.4f semantic=%.4f text_exact=%.4f total=%.4f",
				r.Explain.Rank, r.Element.Tag, r.Element.VisibleText,
				r.Explain.Score.TypeHintAlignment, r.Explain.Score.TagSemantics,
				r.Explain.Score.ExactTextMatch, r.Explain.Score.Total)
		}
	}
}

func TestPaginationLink_ClickOn4(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("td"), withText("4"), withXPath("/html/body/table/tr[4]/td[1]")),
		makeEl(withTag("a"), withText("3"), withID("page_3"), withXPath("/html/body/nav/a[3]")),
		makeEl(withTag("a"), withText("4"), withID("page_4"), withXPath("/html/body/nav/a[4]")),
		makeEl(withTag("a"), withText("5"), withID("page_5"), withXPath("/html/body/nav/a[5]")),
	}

	winner := rankFirst(t, "4", "link", "clickable", elements)
	if winner.Element.Tag != "a" || winner.Element.VisibleText != "4" {
		t.Errorf("expected <a> with text '4', got <%s> text=%q", winner.Element.Tag, winner.Element.VisibleText)
	}
}

// ── Test: Table checkbox by row number ────────────────────────────────────────
// Scenario: a paginated table has rows like:
//   <tr><td><input type="checkbox"></td><td>7</td><td>John</td></tr>
// "CHECK the checkbox for '7'" must pick the checkbox whose label/adjacent
// text contains "7".

func TestTableCheckbox_CheckFor7(t *testing.T) {
	elements := []dom.ElementSnapshot{
		// Checkbox in row 7 — label text comes from adjacent <td>
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("7"),
			withXPath("/html/body/table/tr[7]/td[1]/input[1]")),
		// Checkbox in row 8
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("8"),
			withXPath("/html/body/table/tr[8]/td[1]/input[1]")),
		// A link with text "7" (pagination)
		makeEl(withTag("a"), withText("7"), withXPath("/html/body/nav/a[7]")),
		// A table cell with text "7"
		makeEl(withTag("td"), withText("7"), withXPath("/html/body/table/tr[7]/td[2]")),
	}

	winner := rankFirst(t, "7", "checkbox", "checkbox", elements)
	if winner.Element.Tag != "input" || winner.Element.InputType != "checkbox" {
		t.Errorf("expected <input type=checkbox>, got <%s type=%s> text=%q label=%q",
			winner.Element.Tag, winner.Element.InputType,
			winner.Element.VisibleText, winner.Element.LabelText)
		ranked := scorer.Rank("7", "checkbox", "checkbox", elements, 10, nil)
		for _, r := range ranked {
			t.Logf("  rank=%d tag=%-8s type=%-10s text=%-10q label=%-10q semantic=%.4f total=%.4f",
				r.Explain.Rank, r.Element.Tag, r.Element.InputType,
				r.Element.VisibleText, r.Element.LabelText,
				r.Explain.Score.TagSemantics, r.Explain.Score.Total)
		}
	}
}

func TestTableCheckbox_CheckFor17(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("17"),
			withXPath("/html/body/table/tr[2]/td[1]/input[1]")),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("18"),
			withXPath("/html/body/table/tr[3]/td[1]/input[1]")),
		makeEl(withTag("td"), withText("17"), withXPath("/html/body/table/tr[2]/td[2]")),
	}

	winner := rankFirst(t, "17", "checkbox", "checkbox", elements)
	if winner.Element.Tag != "input" || winner.Element.InputType != "checkbox" {
		t.Errorf("expected <input type=checkbox> with label '17', got <%s> label=%q",
			winner.Element.Tag, winner.Element.LabelText)
	}
}

// ── Test: Select dropdown ────────────────────────────────────────────────────
// Scenario: SELECT 'Japan' from 'Country' dropdown — should pick <select>
// over other elements containing "Country".

func TestSelectDropdown_Country(t *testing.T) {
	elements := []dom.ElementSnapshot{
		// Label text "Country"
		makeEl(withTag("label"), withText("Country"), withXPath("/html/body/label[1]")),
		// Select element with label "Country"
		makeEl(withTag("select"), withLabel("Country"), withID("country"),
			withXPath("/html/body/select[1]")),
		// A div with text "Country"
		makeEl(withTag("div"), withText("Select your Country"), withXPath("/html/body/div[5]")),
	}

	winner := rankFirst(t, "Country", "dropdown", "select", elements)
	if winner.Element.Tag != "select" {
		t.Errorf("expected <select>, got <%s> text=%q label=%q",
			winner.Element.Tag, winner.Element.VisibleText, winner.Element.LabelText)
	}
}

// ── Test: Disabled element penalty ───────────────────────────────────────────

func TestDisabledElementGetsZero(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("button"), withText("Submit"), withDisabled()),
		makeEl(withTag("button"), withText("Submit")),
	}

	winner := rankFirst(t, "Submit", "button", "clickable", elements)
	if winner.Element.IsDisabled {
		t.Error("disabled element should not win")
	}
}

// ── Test: Hidden element penalty ─────────────────────────────────────────────

func TestHiddenElementGetsPenalty(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("button"), withText("Submit"), withHidden()),
		makeEl(withTag("button"), withText("Submit")),
	}

	winner := rankFirst(t, "Submit", "button", "clickable", elements)
	if winner.Element.IsHidden {
		t.Error("hidden element should not outrank visible one")
	}
}

// ── Test: Exact text match beats substring ───────────────────────────────────

func TestExactTextBeatSubstring(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("button"), withText("Save and Continue")),
		makeEl(withTag("button"), withText("Save")),
		makeEl(withTag("button"), withText("Save Draft")),
	}

	winner := rankFirst(t, "Save", "", "clickable", elements)
	if winner.Element.VisibleText != "Save" {
		t.Errorf("expected exact 'Save', got %q", winner.Element.VisibleText)
	}
}

// ── Test: Type hint "link" boosts <a> over other tags ────────────────────────

func TestTypeHintLinkBoostsAnchor(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("button"), withText("Download")),
		makeEl(withTag("a"), withText("Download")),
	}

	winner := rankFirst(t, "Download", "link", "clickable", elements)
	if winner.Element.Tag != "a" {
		t.Errorf("expected <a>, got <%s>", winner.Element.Tag)
	}
}

// ── Test: Checkbox mode rejects non-checkbox elements ────────────────────────

func TestCheckboxModePenalizesNonCheckbox(t *testing.T) {
	elements := []dom.ElementSnapshot{
		// Correct: input[type=checkbox] with label "Monday"
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Monday"),
			withXPath("/html/body/div/input[1]")),
		// Decoy: a button with text "Monday"
		makeEl(withTag("button"), withText("Monday"),
			withXPath("/html/body/div/button[1]")),
		// Decoy: a link with text "Monday"
		makeEl(withTag("a"), withText("Monday"),
			withXPath("/html/body/nav/a[1]")),
	}

	winner := rankFirst(t, "Monday", "checkbox", "checkbox", elements)
	if winner.Element.Tag != "input" || winner.Element.InputType != "checkbox" {
		t.Errorf("expected <input type=checkbox>, got <%s type=%s>",
			winner.Element.Tag, winner.Element.InputType)
	}
}

// ── Test: Short text ambiguity — "2" as link vs td ───────────────────────────
// This is the core pagination bug scenario. A <td> with text "2" and an <a>
// with text "2" both have exact text match, but the type hint "link" must
// disambiguate.

func TestShortTextAmbiguity_TdVsLink(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("td"), withText("2"), withXPath("/html/body/table/tr[2]/td[1]")),
		makeEl(withTag("a"), withText("2"), withXPath("/html/body/nav/a[2]")),
	}

	winner := rankFirst(t, "2", "link", "clickable", elements)
	if winner.Element.Tag != "a" {
		t.Errorf("expected <a>, got <%s> xpath=%s", winner.Element.Tag, winner.Element.XPath)
		ranked := scorer.Rank("2", "link", "clickable", elements, 10, nil)
		for _, r := range ranked {
			t.Logf("  tag=%-5s text=%-5q exact=%.3f norm=%.3f semantic=%.3f hint=%.3f total=%.4f",
				r.Element.Tag, r.Element.VisibleText,
				r.Explain.Score.ExactTextMatch, r.Explain.Score.NormalizedTextMatch,
				r.Explain.Score.TagSemantics, r.Explain.Score.TypeHintAlignment,
				r.Explain.Score.Total)
		}
	}
}

// ── Test: Checkbox finds label from adjacent table cell ──────────────────────
// The real page scenario: the checkbox has no visibleText because
// checkboxes don't render text. The label comes from an adjacent
// <td> — the heuristic probe must capture this as labelText.
// If labelText is properly captured, the scorer should find the match.

func TestCheckboxWithLabelOnly(t *testing.T) {
	// Checkbox with label "7" but no visible text
	chk := makeEl(withTag("input"), withInputType("checkbox"), withLabel("7"))
	if chk.NormLabelText != "7" {
		t.Fatalf("expected NormLabelText='7', got %q", chk.NormLabelText)
	}

	elements := []dom.ElementSnapshot{chk}
	winner := rankFirst(t, "7", "checkbox", "checkbox", elements)
	if winner.Explain.Score.Total <= 0 {
		t.Errorf("checkbox with label '7' should have positive score, got %.4f", winner.Explain.Score.Total)
	}
	if winner.Explain.Score.LabelMatch <= 0 {
		t.Errorf("expected positive LabelMatch, got %.4f", winner.Explain.Score.LabelMatch)
	}
}

// ── Test: Checkbox in real table DOM scenario ────────────────────────────────
// CRITICAL: This simulates what actually happens on testautomationpractice.
// The checkbox is in a table row. Both the checkbox <input> and the <td>
// with number "7" are in the same row. The probe exposes the <td>'s text
// only if the label resolver connects them.

func TestCheckboxInTableRow_RealisticScoring(t *testing.T) {
	elements := []dom.ElementSnapshot{
		// Checkbox: no visible text, but label = row number
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("7"),
			withXPath("//table/tr[2]/td[1]/input[1]")),
		// The <td> that contains the number text
		makeEl(withTag("td"), withText("7"),
			withXPath("//table/tr[2]/td[2]")),
		// Another checkbox in another row
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("8"),
			withXPath("//table/tr[3]/td[1]/input[1]")),
		// Pagination "7" link
		makeEl(withTag("a"), withText("7"),
			withXPath("//nav/a[7]")),
	}

	winner := rankFirst(t, "7", "checkbox", "checkbox", elements)

	// Must pick the checkbox, not the <td> or <a>
	if winner.Element.Tag != "input" || winner.Element.InputType != "checkbox" {
		t.Errorf("expected <input type=checkbox>, got <%s type=%s> text=%q label=%q score=%.4f",
			winner.Element.Tag, winner.Element.InputType,
			winner.Element.VisibleText, winner.Element.LabelText,
			winner.Explain.Score.Total)
	}
	// Must pick the one with label "7", not "8"
	if winner.Element.LabelText != "7" {
		t.Errorf("expected label '7', got %q", winner.Element.LabelText)
	}
}

// ── Individual scoring function tests ─────────────────────────────────────────

func TestScoreExactText(t *testing.T) {
	el := makeEl(withTag("button"), withText("Submit"))
	s := scorer.Score("submit", "", "clickable", &el, nil)
	if s.ExactTextMatch != 1.0 {
		t.Errorf("ExactTextMatch = %.3f, want 1.0", s.ExactTextMatch)
	}
}

func TestScoreExactText_NoMatch(t *testing.T) {
	el := makeEl(withTag("button"), withText("Cancel"))
	s := scorer.Score("submit", "", "clickable", &el, nil)
	if s.ExactTextMatch != 0.0 {
		t.Errorf("ExactTextMatch = %.3f, want 0.0", s.ExactTextMatch)
	}
}

func TestScoreNormText_Substring(t *testing.T) {
	el := makeEl(withTag("button"), withText("Save and Continue"))
	s := scorer.Score("save", "", "clickable", &el, nil)
	if s.NormalizedTextMatch <= 0 {
		t.Errorf("NormTextMatch = %.3f, want > 0", s.NormalizedTextMatch)
	}
}

func TestScoreLabel_Exact(t *testing.T) {
	el := makeEl(withTag("input"), withInputType("checkbox"), withLabel("Monday"))
	s := scorer.Score("monday", "checkbox", "checkbox", &el, nil)
	if s.LabelMatch < 0.5 {
		t.Errorf("LabelMatch = %.3f, want >= 0.5", s.LabelMatch)
	}
}

func TestScoreLabel_Contains(t *testing.T) {
	el := makeEl(withTag("input"), withInputType("text"), withLabel("Full Name"))
	s := scorer.Score("name", "", "input", &el, nil)
	if s.LabelMatch <= 0 {
		t.Errorf("LabelMatch = %.3f, want > 0", s.LabelMatch)
	}
}

func TestScoreAriaLabel(t *testing.T) {
	el := makeEl(withTag("button"), withAriaLabel("Close dialog"))
	s := scorer.Score("close dialog", "", "clickable", &el, nil)
	if s.AriaMatch < 0.5 {
		t.Errorf("AriaMatch = %.3f, want >= 0.5", s.AriaMatch)
	}
}

func TestScoreID(t *testing.T) {
	el := makeEl(withTag("button"), withID("submit-btn"))
	s := scorer.Score("submit btn", "", "clickable", &el, nil)
	if s.IDMatch <= 0 {
		t.Errorf("IDMatch = %.3f, want > 0", s.IDMatch)
	}
}

func TestScoreTagSemantics_ButtonInClickable(t *testing.T) {
	el := makeEl(withTag("button"))
	s := scorer.Score("x", "", "clickable", &el, nil)
	if s.TagSemantics < 0.05 {
		t.Errorf("TagSemantics = %.3f, want >= 0.05 for button in clickable", s.TagSemantics)
	}
}

func TestScoreTagSemantics_CheckboxInCheckboxMode(t *testing.T) {
	el := makeEl(withTag("input"), withInputType("checkbox"))
	s := scorer.Score("x", "", "checkbox", &el, nil)
	if s.TagSemantics < 0.05 {
		t.Errorf("TagSemantics = %.3f, want >= 0.05 for checkbox in checkbox mode", s.TagSemantics)
	}
}

func TestScoreTagSemantics_ButtonPenaltyInCheckboxMode(t *testing.T) {
	el := makeEl(withTag("button"))
	s := scorer.Score("x", "", "checkbox", &el, nil)
	if s.TagSemantics >= 0 {
		t.Errorf("TagSemantics = %.3f, want < 0 for button in checkbox mode", s.TagSemantics)
	}
}

func TestScoreTagSemantics_SelectInSelectMode(t *testing.T) {
	el := makeEl(withTag("select"))
	s := scorer.Score("x", "", "select", &el, nil)
	if s.TagSemantics < 0.05 {
		t.Errorf("TagSemantics = %.3f, want >= 0.05 for select in select mode", s.TagSemantics)
	}
}

func TestScoreTypeHint_LinkBoostsAnchor(t *testing.T) {
	el := makeEl(withTag("a"))
	s := scorer.Score("x", "link", "clickable", &el, nil)
	if s.TypeHintAlignment < 0.3 {
		t.Errorf("TypeHintAlignment = %.3f, want >= 0.3 for <a> with hint=link", s.TypeHintAlignment)
	}
}

func TestScoreTypeHint_LinkDoesNotBoostTd(t *testing.T) {
	el := makeEl(withTag("td"))
	s := scorer.Score("x", "link", "clickable", &el, nil)
	if s.TypeHintAlignment > 0 {
		t.Errorf("TypeHintAlignment = %.3f for <td> with hint=link, want 0", s.TypeHintAlignment)
	}
}

func TestScorePenalty_DisabledZero(t *testing.T) {
	el := makeEl(withTag("button"), withText("Submit"), withDisabled())
	s := scorer.Score("submit", "", "clickable", &el, nil)
	if s.Total != 0.0 {
		t.Errorf("Total = %.3f, want 0.0 for disabled", s.Total)
	}
}

func TestScorePenalty_HiddenReduced(t *testing.T) {
	visible := makeEl(withTag("button"), withText("Submit"))
	hidden := makeEl(withTag("button"), withText("Submit"), withHidden())
	sv := scorer.Score("submit", "", "clickable", &visible, nil)
	sh := scorer.Score("submit", "", "clickable", &hidden, nil)
	if sh.Total >= sv.Total {
		t.Errorf("hidden (%.3f) should be less than visible (%.3f)", sh.Total, sv.Total)
	}
}

// ── Rank ordering tests ──────────────────────────────────────────────────────

func TestRank_TopN(t *testing.T) {
	elements := make([]dom.ElementSnapshot, 20)
	for i := range elements {
		elements[i] = makeEl(withTag("button"), withText("btn"))
	}
	ranked := scorer.Rank("btn", "", "clickable", elements, 5, nil)
	if len(ranked) != 5 {
		t.Errorf("Rank(topN=5) returned %d, want 5", len(ranked))
	}
}

func TestRank_OnlyFirstIsChosen(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("button"), withText("Submit")),
		makeEl(withTag("button"), withText("Submit")),
	}
	ranked := scorer.Rank("submit", "", "clickable", elements, 10, nil)
	if !ranked[0].Explain.Chosen {
		t.Error("first should be chosen")
	}
	if ranked[1].Explain.Chosen {
		t.Error("second should not be chosen")
	}
}

// ── NEAR proximity tests ─────────────────────────────────────────────────────

func TestNearProximity_CloserWins(t *testing.T) {
	closeBtn := makeEl(withTag("button"), withText("Add to Cart"),
		withXPath("/html/body/div[1]/button[1]"))
	closeBtn.Rect = dom.Rect{Left: 100, Top: 100, Width: 80, Height: 30}
	
	farBtn := makeEl(withTag("button"), withText("Add to Cart"),
		withXPath("/html/body/div[2]/button[1]"))
	farBtn.Rect = dom.Rect{Left: 100, Top: 600, Width: 80, Height: 30}

	anchor := &scorer.AnchorContext{
		Rect:  dom.Rect{Left: 100, Top: 80, Width: 200, Height: 30},
		XPath: "/html/body/div[1]/h2[1]",
		Words: []string{"product"},
	}

	elements := []dom.ElementSnapshot{farBtn, closeBtn}
	ranked := scorer.Rank("Add to Cart", "", "clickable", elements, 10, anchor)
	if len(ranked) < 2 {
		t.Fatal("expected at least 2 ranked candidates")
	}
	// The one closer to anchor should be first
	if ranked[0].Element.Rect.Top != 100 {
		t.Errorf("expected closer button (top=100) to win, got top=%.0f", ranked[0].Element.Rect.Top)
	}
}

// ── Real scenario: "CLICK on the '2' link" with full page context ────────────

func TestRealScenario_PaginationWithDecoys(t *testing.T) {
	// Simulate the real testautomationpractice page:
	// - Table cells with numbers 1-5
	// - Pagination <a> links with numbers 1-5
	// - Various buttons and inputs
	elements := []dom.ElementSnapshot{
		// Table cells
		makeEl(withTag("td"), withText("1"), withXPath("//table/tr[1]/td[1]")),
		makeEl(withTag("td"), withText("2"), withXPath("//table/tr[2]/td[1]")),
		makeEl(withTag("td"), withText("3"), withXPath("//table/tr[3]/td[1]")),
		makeEl(withTag("td"), withText("12"), withXPath("//table/tr[12]/td[1]")),
		// Pagination links
		makeEl(withTag("a"), withText("1"), withXPath("//ul[@id='pagination']/li[1]/a[1]")),
		makeEl(withTag("a"), withText("2"), withXPath("//ul[@id='pagination']/li[2]/a[1]")),
		makeEl(withTag("a"), withText("3"), withXPath("//ul[@id='pagination']/li[3]/a[1]")),
		makeEl(withTag("a"), withText("4"), withXPath("//ul[@id='pagination']/li[4]/a[1]")),
		makeEl(withTag("a"), withText("5"), withXPath("//ul[@id='pagination']/li[5]/a[1]")),
		// Other elements that might contain "2"
		makeEl(withTag("span"), withText("Page 2 of 5")),
		makeEl(withTag("button"), withText("Show 20 per page")),
	}

	winner := rankFirst(t, "2", "link", "clickable", elements)
	if winner.Element.Tag != "a" || winner.Element.VisibleText != "2" {
		t.Errorf("expected <a>'2', got <%s>%q", winner.Element.Tag, winner.Element.VisibleText)
	}

	winner4 := rankFirst(t, "4", "link", "clickable", elements)
	if winner4.Element.Tag != "a" || winner4.Element.VisibleText != "4" {
		t.Errorf("expected <a>'4', got <%s>%q", winner4.Element.Tag, winner4.Element.VisibleText)
	}
}

// ── Real scenario: "CHECK the checkbox for '7'" on table page ────────────────

func TestRealScenario_TableCheckboxFullContext(t *testing.T) {
	elements := []dom.ElementSnapshot{
		// Day-of-week checkboxes (these have label via <label for=>)
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Sunday"),
			withID("sunday"), withXPath("//input[@id='sunday']")),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Monday"),
			withID("monday"), withXPath("//input[@id='monday']")),
		// Table row checkboxes — label from adjacent <td> text
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("6"),
			withXPath("//table[@id='productTable']/tbody/tr[1]/td[1]/input[1]")),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("7"),
			withXPath("//table[@id='productTable']/tbody/tr[2]/td[1]/input[1]")),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("8"),
			withXPath("//table[@id='productTable']/tbody/tr[3]/td[1]/input[1]")),
		// The <td> cells with the numbers (these are in clickable mode but not checkbox mode)
		makeEl(withTag("td"), withText("6"), withXPath("//table/tbody/tr[1]/td[2]")),
		makeEl(withTag("td"), withText("7"), withXPath("//table/tbody/tr[2]/td[2]")),
		makeEl(withTag("td"), withText("8"), withXPath("//table/tbody/tr[3]/td[2]")),
	}

	winner := rankFirst(t, "7", "checkbox", "checkbox", elements)
	if winner.Element.Tag != "input" || winner.Element.InputType != "checkbox" || winner.Element.LabelText != "7" {
		t.Errorf("expected checkbox with label '7', got <%s type=%s> label=%q xpath=%s score=%.4f",
			winner.Element.Tag, winner.Element.InputType, winner.Element.LabelText,
			winner.Element.XPath, winner.Explain.Score.Total)
		ranked := scorer.Rank("7", "checkbox", "checkbox", elements, 10, nil)
		for _, r := range ranked {
			t.Logf("  rank=%d <%s type=%s> label=%q text=%q semantic=%.3f label=%.3f total=%.4f",
				r.Explain.Rank, r.Element.Tag, r.Element.InputType,
				r.Element.LabelText, r.Element.VisibleText,
				r.Explain.Score.TagSemantics, r.Explain.Score.LabelMatch,
				r.Explain.Score.Total)
		}
	}
}

// ── Word-overlap scoring tests ───────────────────────────────────────────────

func TestSignificantWords(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"cpu of chrome", 2},           // "cpu", "chrome"; "of" is stop word
		{"price of master in selenium", 3}, // "price", "master", "selenium"
		{"a", 0},                       // single letter, too short
		{"submit", 1},
		{"the quick brown fox", 3},     // "quick", "brown", "fox"; "the" is stop word
	}
	for _, tc := range tests {
		got := scorer.SignificantWords(tc.input)
		if len(got) != tc.want {
			t.Errorf("scorer.SignificantWords(%q) = %v (len=%d), want len=%d", tc.input, got, len(got), tc.want)
		}
	}
}

func TestScoreNormText_WordOverlap(t *testing.T) {
	// A <td> cell whose label text includes both "cpu" and "chrome"
	el := makeEl(withTag("td"), withText("3.2 GHz"), withLabel("CPU Chrome"))
	s := scorer.Score("cpu of chrome", "", "none", &el, nil)
	if s.NormalizedTextMatch < 0.5 {
		t.Errorf("NormalizedTextMatch = %.3f, want >= 0.5 for word-overlap 'cpu of chrome' vs label 'CPU Chrome'", s.NormalizedTextMatch)
	}
}

func TestScoreLabelText_WordOverlap(t *testing.T) {
	el := makeEl(withTag("td"), withLabel("CPU Chrome 3000"))
	s := scorer.Score("cpu of chrome", "", "none", &el, nil)
	if s.LabelMatch < 0.5 {
		t.Errorf("LabelMatch = %.3f, want >= 0.5 for word-overlap", s.LabelMatch)
	}
}

func TestScoreLabelText_PartialWordOverlap(t *testing.T) {
	// Only 1 of 2 significant words matches
	el := makeEl(withTag("td"), withLabel("Firefox 2.5GHz"))
	s := scorer.Score("cpu of chrome", "", "none", &el, nil)
	// "chrome" not in label, only "cpu" would need to match... wait, neither "cpu" nor "chrome" is in "firefox 2.5ghz"
	if s.LabelMatch > 0.1 {
		t.Errorf("LabelMatch = %.3f, want < 0.1 for no word overlap", s.LabelMatch)
	}
}

// ── Tag semantics: none/locate mode ──────────────────────────────────────────

func TestScoreTagSemantics_TdInLocateMode(t *testing.T) {
	el := makeEl(withTag("td"), withText("Chrome"))
	s := scorer.Score("chrome", "", "none", &el, nil)
	if s.TagSemantics < 0.02 {
		t.Errorf("TagSemantics = %.3f, want >= 0.02 for <td> in locate mode", s.TagSemantics)
	}
}

func TestScoreTagSemantics_ButtonInLocateMode(t *testing.T) {
	el := makeEl(withTag("button"), withText("Popup"))
	s := scorer.Score("popup", "", "none", &el, nil)
	if s.TagSemantics > 0.02 {
		t.Errorf("TagSemantics = %.3f, want <= 0.02 for <button> in locate mode (should not be preferred)", s.TagSemantics)
	}
}

func TestScoreTagSemantics_HeadingInLocateMode(t *testing.T) {
	el := makeEl(withTag("h2"), withText("Products"))
	s := scorer.Score("products", "", "none", &el, nil)
	if s.TagSemantics < 0.025 {
		t.Errorf("TagSemantics = %.3f, want >= 0.025 for <h2> in locate mode", s.TagSemantics)
	}
}

// ── Real scenario: EXTRACT 'CPU of Chrome' from table ────────────────────────

func TestRealScenario_ExtractCPUofChrome(t *testing.T) {
	// Simulate the web table on testautomationpractice.blogspot.com:
	// | Browser | CPU     | Price |
	// | Chrome  | 3.2 GHz | 35    |
	// | Firefox | 2.9 GHz | 28    |
	// The <td> for Chrome's CPU has labelText from column header "CPU" + sibling "Chrome"
	elements := []dom.ElementSnapshot{
		// Table cells (in locate mode these are candidates)
		makeEl(withTag("td"), withText("Chrome"),
			withLabel("Browser"), withXPath("//table/tr[2]/td[1]")),
		makeEl(withTag("td"), withText("3.2 GHz"),
			withLabel("CPU Chrome 35"), withXPath("//table/tr[2]/td[2]")),
		makeEl(withTag("td"), withText("35"),
			withLabel("Price Chrome 3.2 GHz"), withXPath("//table/tr[2]/td[3]")),
		// Firefox row
		makeEl(withTag("td"), withText("Firefox"),
			withLabel("Browser"), withXPath("//table/tr[3]/td[1]")),
		makeEl(withTag("td"), withText("2.9 GHz"),
			withLabel("CPU Firefox 28"), withXPath("//table/tr[3]/td[2]")),
		makeEl(withTag("td"), withText("28"),
			withLabel("Price Firefox 2.9 GHz"), withXPath("//table/tr[3]/td[3]")),
		// Decoy: button on the page
		makeEl(withTag("button"), withText("Popup Windows"),
			withID("PopUp"), withXPath("//button[@id='PopUp']")),
		// Decoy: another button
		makeEl(withTag("button"), withText("Double Click"),
			withXPath("//button[@id='dblClkBtn']")),
	}

	// EXTRACT 'CPU of Chrome' — should find the Chrome CPU cell, not "Popup Windows" button
	winner := rankFirst(t, "cpu of chrome", "element", "none", elements)
	if winner.Element.VisibleText != "3.2 GHz" {
		t.Errorf("EXTRACT 'CPU of Chrome': expected '3.2 GHz', got %q (tag=%s, xpath=%s, score=%.4f)",
			winner.Element.VisibleText, winner.Element.Tag, winner.Element.XPath, winner.Explain.Score.Total)
		ranked := scorer.Rank("cpu of chrome", "element", "none", elements, 10, nil)
		for _, r := range ranked {
			t.Logf("  rank=%d <%s> text=%-15q label=%-25q norm=%.3f label=%.3f sem=%.3f total=%.4f",
				r.Explain.Rank, r.Element.Tag, r.Element.VisibleText, r.Element.LabelText,
				r.Explain.Score.NormalizedTextMatch, r.Explain.Score.LabelMatch,
				r.Explain.Score.TagSemantics, r.Explain.Score.Total)
		}
	}

	// EXTRACT 'CPU of Firefox' — should find the Firefox CPU cell
	winnerFF := rankFirst(t, "cpu of firefox", "element", "none", elements)
	if winnerFF.Element.VisibleText != "2.9 GHz" {
		t.Errorf("EXTRACT 'CPU of Firefox': expected '2.9 GHz', got %q", winnerFF.Element.VisibleText)
	}

	// EXTRACT 'Price of Chrome' — should find Chrome's price
	winnerPrice := rankFirst(t, "price of chrome", "element", "none", elements)
	if winnerPrice.Element.VisibleText != "35" {
		t.Errorf("EXTRACT 'Price of Chrome': expected '35', got %q", winnerPrice.Element.VisibleText)
	}
}

// ── Real scenario: CLICK 'Item 100' in dropdown (vs pagination "1") ──────────

func TestRealScenario_DropdownItem100VsPagination(t *testing.T) {
	elements := []dom.ElementSnapshot{
		// Pagination links
		makeEl(withTag("a"), withText("1"), withXPath("//ul[@id='pagination']/li[1]/a[1]")),
		makeEl(withTag("a"), withText("2"), withXPath("//ul[@id='pagination']/li[2]/a[1]")),
		makeEl(withTag("a"), withText("3"), withXPath("//ul[@id='pagination']/li[3]/a[1]")),
		// Dropdown items (typically <li> without IDs)
		makeEl(withTag("li"), withText("Item 98"), withXPath("//div[@class='dropdown']/ul/li[98]")),
		makeEl(withTag("li"), withText("Item 99"), withXPath("//div[@class='dropdown']/ul/li[99]")),
		makeEl(withTag("li"), withText("Item 100"), withXPath("//div[@class='dropdown']/ul/li[100]")),
		// <span> decoy
		makeEl(withTag("span"), withText("100 items total")),
	}

	winner := rankFirst(t, "item 100", "element", "clickable", elements)
	if winner.Element.VisibleText != "Item 100" {
		t.Errorf("CLICK 'Item 100': expected 'Item 100', got %q (tag=%s, xpath=%s, score=%.4f)",
			winner.Element.VisibleText, winner.Element.Tag, winner.Element.XPath, winner.Explain.Score.Total)
		ranked := scorer.Rank("item 100", "element", "clickable", elements, 10, nil)
		for _, r := range ranked {
			t.Logf("  rank=%d <%s> text=%-20q exact=%.3f norm=%.3f sem=%.3f total=%.4f",
				r.Explain.Rank, r.Element.Tag, r.Element.VisibleText,
				r.Explain.Score.ExactTextMatch, r.Explain.Score.NormalizedTextMatch,
				r.Explain.Score.TagSemantics, r.Explain.Score.Total)
		}
	}
}

// ── Test: EXTRACT 'Price of Master In Selenium' disambiguation ───────────────

func TestRealScenario_ExtractPriceOfMasterInSelenium(t *testing.T) {
	elements := []dom.ElementSnapshot{
		// Book table:
		// | Title                 | Price |
		// | Selenium              | 300   |
		// | Master In Selenium    | 400   |
		makeEl(withTag("td"), withText("Selenium"),
			withLabel("Title"), withXPath("//table/tr[2]/td[1]")),
		makeEl(withTag("td"), withText("300"),
			withLabel("Price Selenium"), withXPath("//table/tr[2]/td[2]")),
		makeEl(withTag("td"), withText("Master In Selenium"),
			withLabel("Title"), withXPath("//table/tr[3]/td[1]")),
		makeEl(withTag("td"), withText("400"),
			withLabel("Price Master In Selenium"), withXPath("//table/tr[3]/td[2]")),
		// Decoy button
		makeEl(withTag("button"), withText("Popup Windows"),
			withID("PopUp"), withXPath("//button[@id='PopUp']")),
	}

	winner := rankFirst(t, "price of master in selenium", "element", "none", elements)
	if winner.Element.VisibleText != "400" {
		t.Errorf("EXTRACT 'Price of Master In Selenium': expected '400', got %q (tag=%s, label=%q, score=%.4f)",
			winner.Element.VisibleText, winner.Element.Tag, winner.Element.LabelText, winner.Explain.Score.Total)
	}
}
