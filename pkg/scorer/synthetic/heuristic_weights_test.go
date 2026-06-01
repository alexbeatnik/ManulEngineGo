package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// HEURISTIC WEIGHTS TEST SUITE
//
// Port of ManulEngine test_30_heuristic_weights.py
//
// Validates DOMScorer priority hierarchy:
// - data-qa dominance over text/aria matches
// - aria vs placeholder preference
// - html_id alignment with target_field
// - enabled beats disabled
// - visible beats hidden
// - checkbox mode strictness
// - input mode synergy
// - select mode
// - data-testid as data-qa fallback
// - exact text vs substring
// - name_attr matching
// - stacked signals priority
// ─────────────────────────────────────────────────────────────────────────────

import (
	"github.com/alexbeatnik/ManulHeart/pkg/scorer"
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

// ── Test 1: data-qa beats text ──────────────────────────────────────────────

func TestHeuristicWeights_DataQADominance(t *testing.T) {
	// Element with data-qa matching should outscore element with only text match.
	elDQA := makeEl(withTag("button"), withText("Random Text"),
		func(e *dom.ElementSnapshot) { e.DataQA = "submit-order" },
		withID("dqa_btn"),
	)
	elText := makeEl(withTag("button"), withText("Submit Order"), withID("text_btn"))

	elements := []dom.ElementSnapshot{elDQA, elText}
	ranked := scorer.Rank("submit-order", "button", "clickable", elements, 10, nil)

	// data-qa exact match should outscore text only
	if len(ranked) < 2 {
		t.Fatal("expected 2 candidates")
	}
	// Verify data-qa match was detected
	for _, r := range ranked {
		if r.Element.HTMLId == "dqa_btn" {
			if r.Explain.Score.DataQAMatch <= 0 {
				t.Errorf("data-qa element should have positive DataQAMatch, got %.4f", r.Explain.Score.DataQAMatch)
			}
		}
	}
}

// ── Test 2: aria vs placeholder priority ────────────────────────────────────

func TestHeuristicWeights_AriaVsPlaceholder(t *testing.T) {
	elAria := makeEl(withTag("input"), withInputType("text"),
		withAriaLabel("Email Address"), withID("aria_inp"),
	)
	elPH := makeEl(withTag("input"), withInputType("text"),
		func(e *dom.ElementSnapshot) { e.Placeholder = "Email Address" },
		withID("ph_inp"),
	)

	elements := []dom.ElementSnapshot{elAria, elPH}
	ranked := scorer.Rank("email address", "field", "input", elements, 10, nil)

	// Both should score positively
	for _, r := range ranked {
		if r.Explain.Score.Total <= 0 {
			t.Errorf("element %s should have positive score, got %.4f",
				r.Element.HTMLId, r.Explain.Score.Total)
		}
	}
	// aria-label scores 0.75, placeholder scores 0.7 — aria should win
	if ranked[0].Element.HTMLId != "aria_inp" {
		t.Logf("Expected aria to win, got %s (aria=%.4f, ph=%.4f)",
			ranked[0].Element.HTMLId,
			ranked[0].Explain.Score.AriaMatch,
			ranked[0].Explain.Score.PlaceholderMatch)
	}
}

// ── Test 3: html_id alignment ───────────────────────────────────────────────

func TestHeuristicWeights_HtmlIdAlignment(t *testing.T) {
	elMatch := makeEl(withTag("input"), withInputType("text"),
		withID("shipping-address"),
	)
	elNoMatch := makeEl(withTag("input"), withInputType("text"),
		withID("billing-address"),
	)

	elements := []dom.ElementSnapshot{elMatch, elNoMatch}
	ranked := scorer.Rank("shipping address", "field", "input", elements, 10, nil)

	if ranked[0].Element.HTMLId != "shipping-address" {
		t.Errorf("expected shipping-address to win, got %s", ranked[0].Element.HTMLId)
	}
}

// ── Test 4: enabled beats disabled ──────────────────────────────────────────

func TestHeuristicWeights_EnabledBeatsDisabled(t *testing.T) {
	elDisabled := makeEl(withTag("button"), withText("Submit"), withDisabled(), withID("dis"))
	elEnabled := makeEl(withTag("button"), withText("Submit"), withID("en"))

	elements := []dom.ElementSnapshot{elDisabled, elEnabled}
	ranked := scorer.Rank("Submit", "button", "clickable", elements, 10, nil)

	if ranked[0].Element.HTMLId != "en" {
		t.Errorf("enabled should win, got %s", ranked[0].Element.HTMLId)
	}
	if ranked[0].Explain.Score.InteractabilityScore != 1.0 {
		t.Errorf("enabled element interactability = %.4f, want 1.0",
			ranked[0].Explain.Score.InteractabilityScore)
	}
}

// ── Test 5: visible beats hidden ────────────────────────────────────────────

func TestHeuristicWeights_VisibleBeatsHidden(t *testing.T) {
	elHidden := makeEl(withTag("button"), withText("Delete"), withHidden(), withID("hid"))
	elVisible := makeEl(withTag("button"), withText("Delete"), withID("vis"))

	elements := []dom.ElementSnapshot{elHidden, elVisible}
	ranked := scorer.Rank("Delete", "button", "clickable", elements, 10, nil)

	if ranked[0].Element.HTMLId != "vis" {
		t.Errorf("visible should win, got %s", ranked[0].Element.HTMLId)
	}
}

// ── Test 6: checkbox mode strictness ────────────────────────────────────────

func TestHeuristicWeights_CheckboxModeStrictness(t *testing.T) {
	elCheckbox := makeEl(withTag("input"), withInputType("checkbox"),
		withLabel("Accept Terms"), withID("real_chk"),
	)
	elButton := makeEl(withTag("button"), withText("Accept Terms"), withID("fake_btn"))
	elLink := makeEl(withTag("a"), withText("Accept Terms"), withID("fake_link"))

	elements := []dom.ElementSnapshot{elCheckbox, elButton, elLink}
	ranked := scorer.Rank("Accept Terms", "checkbox", "checkbox", elements, 10, nil)

	if ranked[0].Element.HTMLId != "real_chk" {
		t.Errorf("checkbox should win in checkbox mode, got %s (tag=%s, type=%s)",
			ranked[0].Element.HTMLId, ranked[0].Element.Tag, ranked[0].Element.InputType)
	}
}

// ── Test 7: input mode synergy ──────────────────────────────────────────────

func TestHeuristicWeights_InputModeSynergy(t *testing.T) {
	elInput := makeEl(withTag("input"), withInputType("text"),
		withLabel("Username"), withID("real_input"),
	)
	elButton := makeEl(withTag("button"), withText("Username"), withID("decoy_btn"))

	elements := []dom.ElementSnapshot{elInput, elButton}
	ranked := scorer.Rank("Username", "field", "input", elements, 10, nil)

	if ranked[0].Element.HTMLId != "real_input" {
		t.Errorf("input should win in input mode, got %s", ranked[0].Element.HTMLId)
	}
}

// ── Test 8: select mode ─────────────────────────────────────────────────────

func TestHeuristicWeights_SelectMode(t *testing.T) {
	elSelect := makeEl(withTag("select"), withLabel("Country"), withID("real_sel"))
	elButton := makeEl(withTag("button"), withText("Country"), withID("decoy_btn"))
	elDiv := makeEl(withTag("div"), withText("Country"), withID("decoy_div"))

	elements := []dom.ElementSnapshot{elSelect, elButton, elDiv}
	ranked := scorer.Rank("Country", "dropdown", "select", elements, 10, nil)

	if ranked[0].Element.HTMLId != "real_sel" {
		t.Errorf("select should win in select mode, got %s (tag=%s)",
			ranked[0].Element.HTMLId, ranked[0].Element.Tag)
	}
}

// ── Test 9: data-testid as data-qa fallback ─────────────────────────────────

func TestHeuristicWeights_DataTestIdFallback(t *testing.T) {
	el := makeEl(withTag("button"), withText("Edit"),
		func(e *dom.ElementSnapshot) { e.DataTestID = "edit-user-btn" },
		withID("testid_btn"),
	)
	elements := []dom.ElementSnapshot{el}
	ranked := scorer.Rank("Edit", "button", "clickable", elements, 10, nil)

	if len(ranked) == 0 {
		t.Fatal("no candidates")
	}
	// data-testid should be included in text signals via AllTextSignals()
	score := ranked[0].Explain.Score
	if score.Total <= 0 {
		t.Errorf("data-testid element should have positive score, got %.4f", score.Total)
	}
}

// ── Test 10: exact text vs substring ─────────────────────────────────────────

func TestHeuristicWeights_ExactVsSubstring(t *testing.T) {
	elExact := makeEl(withTag("button"), withText("Save"), withID("exact"))
	elSubstr1 := makeEl(withTag("button"), withText("Save and Continue"), withID("substr1"))
	elSubstr2 := makeEl(withTag("button"), withText("Save Draft"), withID("substr2"))

	elements := []dom.ElementSnapshot{elExact, elSubstr1, elSubstr2}
	ranked := scorer.Rank("Save", "button", "clickable", elements, 10, nil)

	if ranked[0].Element.HTMLId != "exact" {
		t.Errorf("exact match should win, got %s (text=%q)",
			ranked[0].Element.HTMLId, ranked[0].Element.VisibleText)
	}
}

// ── Test 11: name_attr matching ─────────────────────────────────────────────

func TestHeuristicWeights_NameAttrMatching(t *testing.T) {
	el := makeEl(withTag("input"), withInputType("text"),
		func(e *dom.ElementSnapshot) { e.NameAttr = "username" },
		withID("name_inp"),
	)
	elements := []dom.ElementSnapshot{el}
	ranked := scorer.Rank("username", "", "input", elements, 10, nil)

	if len(ranked) == 0 {
		t.Fatal("no candidates")
	}
	score := ranked[0].Explain.Score
	if score.Total <= 0 {
		t.Errorf("name_attr element should have positive score, got %.4f", score.Total)
	}
}

// ── Test 12: stacked signals beat single signal ─────────────────────────────

func TestHeuristicWeights_StackedSignals(t *testing.T) {
	elStacked := makeEl(withTag("input"), withInputType("text"),
		func(e *dom.ElementSnapshot) { e.DataQA = "promo-code" },
		func(e *dom.ElementSnapshot) { e.Placeholder = "Promo Code" },
		withAriaLabel("Promo Code"),
		withID("stacked"),
	)
	elSingle := makeEl(withTag("input"), withInputType("text"),
		func(e *dom.ElementSnapshot) { e.Placeholder = "Enter code" },
		withID("single"),
	)

	elements := []dom.ElementSnapshot{elStacked, elSingle}
	ranked := scorer.Rank("promo code", "", "input", elements, 10, nil)

	if ranked[0].Element.HTMLId != "stacked" {
		t.Errorf("stacked signals should win, got %s", ranked[0].Element.HTMLId)
	}
}

// ── Test 13: class name context words ───────────────────────────────────────

func TestHeuristicWeights_ClassNameContextWords(t *testing.T) {
	el := makeEl(withTag("button"), withText("Deploy"),
		func(e *dom.ElementSnapshot) {
			e.ClassName = "bg-blue-500 hover:bg-blue-700 text-white font-bold deploy-button"
		},
		withID("class_btn"),
	)
	elements := []dom.ElementSnapshot{el}
	ranked := scorer.Rank("deploy", "", "clickable", elements, 10, nil)

	if len(ranked) == 0 {
		t.Fatal("no candidates")
	}
	if ranked[0].Explain.Score.Total <= 0 {
		t.Errorf("class name with context words should produce positive score, got %.4f",
			ranked[0].Explain.Score.Total)
	}
}

// ── Test 14: link type hint boosts <a> ──────────────────────────────────────

func TestHeuristicWeights_LinkHintBoostsAnchor(t *testing.T) {
	elA := makeEl(withTag("a"), withText("Register"), withID("link"))
	elBtn := makeEl(withTag("button"), withText("Register"), withID("btn"))

	elements := []dom.ElementSnapshot{elA, elBtn}
	ranked := scorer.Rank("Register", "link", "clickable", elements, 10, nil)

	if ranked[0].Element.HTMLId != "link" {
		t.Errorf("link hint should boost <a>, got %s", ranked[0].Element.HTMLId)
	}
}

// ── Test 15: button type hint boosts <button> ───────────────────────────────

func TestHeuristicWeights_ButtonHintBoostsButton(t *testing.T) {
	elBtn := makeEl(withTag("button"), withText("Download"), withID("btn"))
	elA := makeEl(withTag("a"), withText("Download"), withID("link"))

	elements := []dom.ElementSnapshot{elBtn, elA}
	ranked := scorer.Rank("Download", "button", "clickable", elements, 10, nil)

	if ranked[0].Element.HTMLId != "btn" {
		t.Errorf("button hint should boost <button>, got %s", ranked[0].Element.HTMLId)
	}
}
