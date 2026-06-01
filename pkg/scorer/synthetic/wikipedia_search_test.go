package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// WIKIPEDIA SEARCH INPUT SCORING TEST SUITE
//
// Port of ManulEngine test_26_wikipedia_search.py — heuristic scoring of
// Wikipedia Vector 2022-style search inputs.
//
// Tests call Score/Rank directly with synthetic dom.ElementSnapshot arrays.
// Validates:
// 1. Search input ranked above submit button
// 2. Exact aria-label match produces high score
// 3. Exact placeholder match produces high score
// 4. Partial aria < exact aria
// 5. type="search" is not penalized in input mode
// 6. role="searchbox" treated as real input
// 7. Disabled search input is penalized
// 8. Full Wikipedia DOM disambiguation scenario
// ─────────────────────────────────────────────────────────────────────────────

import (
	"github.com/alexbeatnik/ManulHeart/pkg/scorer"
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

// withNameAttr sets the NameAttr field.
func withNameAttr(n string) func(*dom.ElementSnapshot) {
	return func(e *dom.ElementSnapshot) { e.NameAttr = n }
}

// ── Section 1: Search input beats button ─────────────────────────────────────

func TestWiki_SearchInputWinsOverButton(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("input"), withInputType("search"), withAriaLabel("Search Wikipedia"),
			withPlaceholder("search wikipedia"), withClassName("cdx-text-input__input"),
			withRole("searchbox"), withNameAttr("search"), withID("searchInput"),
			func(e *dom.ElementSnapshot) { e.ID = 0 }),
		makeEl(withTag("button"), withInputType("submit"), withAriaLabel("Search"),
			withID("searchButton"),
			func(e *dom.ElementSnapshot) { e.ID = 1 }),
		makeEl(withTag("a"), withText("Search results"),
			func(e *dom.ElementSnapshot) { e.ID = 2 }),
	}

	ranked := scorer.Rank("Search Wikipedia", "", "input", elements, 10, nil)
	if len(ranked) == 0 {
		t.Fatal("Rank returned 0 candidates")
	}
	if ranked[0].Element.ID != 0 {
		t.Errorf("expected search input (#0) at rank 1, got #%d", ranked[0].Element.ID)
	}
	// Search input should outscore button
	inputScore := ranked[0].Explain.Score.Total
	var btnScore float64
	for _, r := range ranked {
		if r.Element.ID == 1 {
			btnScore = r.Explain.Score.Total
		}
	}
	if inputScore <= btnScore {
		t.Errorf("search input (%.4f) should outscore button (%.4f)", inputScore, btnScore)
	}
}

// ── Section 2: Exact aria-label match ────────────────────────────────────────

func TestWiki_ExactAriaMatchHighScore(t *testing.T) {
	target := makeEl(withTag("input"), withInputType("search"), withAriaLabel("Search Wikipedia"),
		func(e *dom.ElementSnapshot) { e.ID = 0 })
	decoy := makeEl(withTag("input"), withInputType("text"), withAriaLabel("Username"),
		func(e *dom.ElementSnapshot) { e.ID = 1 })

	tScore := scorer.Score("Search Wikipedia", "", "input", &target, nil).Total
	dScore := scorer.Score("Search Wikipedia", "", "input", &decoy, nil).Total

	if tScore <= 0.35 {
		t.Errorf("exact aria match should produce high score, got %.4f", tScore)
	}
	if tScore <= dScore {
		t.Errorf("aria match (%.4f) should outscore unrelated input (%.4f)", tScore, dScore)
	}
}

// ── Section 3: Exact placeholder match ───────────────────────────────────────

func TestWiki_ExactPlaceholderMatch(t *testing.T) {
	el := makeEl(withTag("input"), withInputType("search"), withPlaceholder("search wikipedia"),
		func(e *dom.ElementSnapshot) { e.ID = 0 })

	score := scorer.Score("Search Wikipedia", "", "input", &el, nil).Total
	if score <= 0.3 {
		t.Errorf("exact placeholder match should produce good score, got %.4f", score)
	}
}

// ── Section 4: Partial aria < exact aria ─────────────────────────────────────

func TestWiki_PartialAriaLessThanExact(t *testing.T) {
	exact := makeEl(withTag("input"), withInputType("search"), withAriaLabel("Search Wikipedia"),
		func(e *dom.ElementSnapshot) { e.ID = 0 })
	partial := makeEl(withTag("input"), withInputType("search"), withAriaLabel("Wikipedia Help"),
		func(e *dom.ElementSnapshot) { e.ID = 1 })

	eScore := scorer.Score("Search Wikipedia", "", "input", &exact, nil).Total
	pScore := scorer.Score("Search Wikipedia", "", "input", &partial, nil).Total

	if eScore <= pScore {
		t.Errorf("exact aria (%.4f) should beat partial aria (%.4f)", eScore, pScore)
	}
}

// ── Section 5: type="search" not penalized in input mode ─────────────────────

func TestWiki_SearchTypeNotPenalized(t *testing.T) {
	searchEl := makeEl(withTag("input"), withInputType("search"), withAriaLabel("My Search"),
		func(e *dom.ElementSnapshot) { e.ID = 0 })
	textEl := makeEl(withTag("input"), withInputType("text"), withAriaLabel("My Search"),
		func(e *dom.ElementSnapshot) { e.ID = 1 })

	sScore := scorer.Score("My Search", "", "input", &searchEl, nil).Total
	tScore := scorer.Score("My Search", "", "input", &textEl, nil).Total

	if sScore < tScore {
		t.Errorf("type=search (%.4f) should not be penalized vs type=text (%.4f)", sScore, tScore)
	}
}

// ── Section 6: role="searchbox" treated as real input ────────────────────────

func TestWiki_RoleSearchbox(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("input"), withInputType("search"), withRole("searchbox"),
			withAriaLabel("City search"),
			func(e *dom.ElementSnapshot) { e.ID = 0 }),
		makeEl(withTag("button"), withAriaLabel("City search"),
			func(e *dom.ElementSnapshot) { e.ID = 1 }),
	}

	winner := rankFirst(t, "City search", "", "input", elements)
	if winner.Element.ID != 0 {
		t.Errorf("role=searchbox should rank #1, got ID=%d", winner.Element.ID)
	}
}

// ── Section 7: Full Wikipedia disambiguation ─────────────────────────────────

func TestWiki_FullDisambiguation(t *testing.T) {
	elements := []dom.ElementSnapshot{
		makeEl(withTag("input"), withInputType("search"), withAriaLabel("Search Wikipedia"),
			withPlaceholder("search wikipedia"), withClassName("cdx-text-input__input"),
			withRole("searchbox"), withNameAttr("search"),
			func(e *dom.ElementSnapshot) { e.ID = 0 }),
		makeEl(withTag("button"), withAriaLabel("Search"), withID("searchButton"),
			func(e *dom.ElementSnapshot) { e.ID = 1 }),
		makeEl(withTag("a"), withText("Create account"),
			func(e *dom.ElementSnapshot) { e.ID = 2 }),
		makeEl(withTag("a"), withText("Log in"),
			func(e *dom.ElementSnapshot) { e.ID = 3 }),
		makeEl(withTag("input"), withInputType("text"), withAriaLabel("Username"),
			withID("wpName1"),
			func(e *dom.ElementSnapshot) { e.ID = 4 }),
	}

	ranked := scorer.Rank("Search Wikipedia", "", "input", elements, 10, nil)
	if ranked[0].Element.ID != 0 {
		t.Errorf("search input should be #1, got #%d", ranked[0].Element.ID)
	}

	bestScore := ranked[0].Explain.Score.Total
	var btnScore, userScore float64
	for _, r := range ranked {
		if r.Element.ID == 1 {
			btnScore = r.Explain.Score.Total
		}
		if r.Element.ID == 4 {
			userScore = r.Explain.Score.Total
		}
	}
	if bestScore <= btnScore {
		t.Errorf("search input (%.4f) should dominate button (%.4f)", bestScore, btnScore)
	}
	if bestScore <= userScore {
		t.Errorf("search input (%.4f) should outscore Username (%.4f)", bestScore, userScore)
	}
}

// ── Section 8: Disabled search input penalized ───────────────────────────────

func TestWiki_DisabledPenalized(t *testing.T) {
	enabled := makeEl(withTag("input"), withInputType("search"), withAriaLabel("Search Wikipedia"),
		func(e *dom.ElementSnapshot) { e.ID = 0 })
	disabled := makeEl(withTag("input"), withInputType("search"), withAriaLabel("Search Wikipedia"),
		withDisabled(),
		func(e *dom.ElementSnapshot) { e.ID = 1 })

	eScore := scorer.Score("Search Wikipedia", "", "input", &enabled, nil).Total
	dScore := scorer.Score("Search Wikipedia", "", "input", &disabled, nil).Total

	if eScore <= dScore {
		t.Errorf("enabled (%.4f) should outscore disabled (%.4f)", eScore, dScore)
	}
}
