package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXTUAL PROXIMITY TEST SUITE
//
// Port of ManulEngine test_47_contextual_proximity.py
//
// Validates:
// 1. NEAR: Euclidean distance boosting (closest candidate wins)
// 2. NEAR: DOM ancestry affinity (same-card beats closer neighbor)
// 3. NEAR: beyond 500px threshold → 0
// 4. NEAR: anchor dev-attribute affinity
// 5. Proximity weight boost (0.10 → 1.5 with NEAR)
// 6. XPath default proximity (no NEAR)
// 7. Edge cases: overlapping position, no XPath
// ─────────────────────────────────────────────────────────────────────────────

import (
	"github.com/alexbeatnik/ManulEngineGo/pkg/scorer"
	"math"
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func makeElWithRect(id string, tag string, text string, top, left, bottom, right float64, xpath string) dom.ElementSnapshot {
	return makeEl(
		withTag(tag), withText(text), withID(id),
		func(e *dom.ElementSnapshot) {
			e.XPath = xpath
			e.Rect = dom.Rect{
				Top:    top,
				Left:   left,
				Bottom: bottom,
				Right:  right,
				Width:  right - left,
				Height: bottom - top,
			}
		},
	)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section A: NEAR — closest element wins
// ═══════════════════════════════════════════════════════════════════════════════

func TestProximity_NearClosestWins(t *testing.T) {
	// Use non-matching text so text channel doesn't cap total at 1.0.
	// Both elements have the same weak signal — proximity breaks the tie.
	anchor := &scorer.AnchorContext{
		Rect: dom.Rect{Top: 90, Left: 190, Bottom: 120, Right: 250, Width: 60, Height: 30},
	}
	close := makeElWithRect("close", "button", "Action", 110, 240, 140, 340,
		"/html/body/div/button[1]")
	far := makeElWithRect("far", "button", "Action", 580, 780, 610, 880,
		"/html/body/div/button[2]")

	ranked := scorer.Rank("action", "button", "clickable",
		[]dom.ElementSnapshot{close, far}, 10, anchor)

	if ranked[0].Element.HTMLId != "close" {
		t.Errorf("closest element should win with NEAR, got %s", ranked[0].Element.HTMLId)
	}
	// scorer.Rank() sorts by RawScore (unclamped), so compare RawScore not Total
	if ranked[0].Explain.Score.RawScore <= ranked[1].Explain.Score.RawScore {
		t.Errorf("close raw score (%.4f) should beat far raw score (%.4f)",
			ranked[0].Explain.Score.RawScore, ranked[1].Explain.Score.RawScore)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section B: NEAR — distance scoring values
// ═══════════════════════════════════════════════════════════════════════════════

func TestProximity_NearOverlappingElement(t *testing.T) {
	// Element directly overlapping anchor → near score should be ~1.0
	anchor := &scorer.AnchorContext{
		Rect:  dom.Rect{Top: 100, Left: 100, Bottom: 130, Right: 200, Width: 100, Height: 30},
		XPath: "/html/body/div/span[1]",
	}
	el := makeElWithRect("onTop", "button", "Save", 100, 100, 130, 200,
		"/html/body/div/button[1]")

	score := scorer.Score("save", "button", "clickable", &el, anchor)
	if score.ProximityScore < 0.5 {
		t.Errorf("overlapping element proximity should be high, got %.4f", score.ProximityScore)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section C: NEAR — same container beats closer neighbor
// ═══════════════════════════════════════════════════════════════════════════════

func TestProximity_NearSameContainerBeatsCloserNeighbor(t *testing.T) {
	// Anchor in card [4]. Same-card button vs. neighbor-card button that's slightly closer.
	anchor := &scorer.AnchorContext{
		Rect:  dom.Rect{Top: 100, Left: 420, Bottom: 130, Right: 560, Width: 140, Height: 30},
		XPath: "/html/body/div/div[1]/div[4]/div[1]/a/div",
	}
	sameCard := makeElWithRect("same", "button", "Add to cart", 112, 565, 142, 670,
		"/html/body/div/div[1]/div[4]/div[2]/button")
	neighborCard := makeElWithRect("neighbor", "button", "Add to cart", 106, 360, 136, 465,
		"/html/body/div/div[1]/div[3]/div[2]/button")

	ranked := scorer.Rank("add to cart", "button", "clickable",
		[]dom.ElementSnapshot{neighborCard, sameCard}, 10, anchor)

	if ranked[0].Element.HTMLId != "same" {
		t.Errorf("same-card button should win via DOM affinity, got %s", ranked[0].Element.HTMLId)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section D: NEAR — beyond threshold gets 0
// ═══════════════════════════════════════════════════════════════════════════════

func TestProximity_NearBeyondThreshold(t *testing.T) {
	// Element very far away (>500px center distance)
	anchor := &scorer.AnchorContext{
		Rect: dom.Rect{Top: 0, Left: 0, Bottom: 30, Right: 100, Width: 100, Height: 30},
	}
	el := makeElWithRect("far", "button", "Save", 1400, 1400, 1430, 1500,
		"/html/body/footer/button[1]")

	score := scorer.Score("save", "button", "clickable", &el, anchor)
	if score.ProximityScore > 0.01 {
		t.Errorf("element beyond 500px threshold should get ~0 proximity, got %.4f", score.ProximityScore)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section E: NEAR — anchor dev-attribute affinity
// ═══════════════════════════════════════════════════════════════════════════════

func TestProximity_NearAnchorDevAttrAffinity(t *testing.T) {
	// Anchor text: "Sauce Labs Fleece Jacket"
	// Words: ["sauce", "labs", "fleece", "jacket"]
	// The correct button has id="add-to-cart-sauce-labs-fleece-jacket"
	anchor := &scorer.AnchorContext{
		Rect:  dom.Rect{Top: 427, Left: 916, Bottom: 447, Right: 1215, Width: 299, Height: 20},
		XPath: `//*[@id="item_5_title_link"]`,
		Words: []string{"sauce", "labs", "fleece", "jacket"},
	}
	bike := makeElWithRect("bike", "button", "Add to cart", 339, 1055, 373, 1215,
		`//*[@id="add-to-cart-sauce-labs-bike-light"]`)
	bike.HTMLId = "add-to-cart-sauce-labs-bike-light"
	bike.DataQA = "add-to-cart-sauce-labs-bike-light"
	bike.Normalize()

	fleece := makeElWithRect("fleece", "button", "Add to cart", 591, 1055, 625, 1215,
		`//*[@id="add-to-cart-sauce-labs-fleece-jacket"]`)
	fleece.HTMLId = "add-to-cart-sauce-labs-fleece-jacket"
	fleece.DataQA = "add-to-cart-sauce-labs-fleece-jacket"
	fleece.Normalize()

	ranked := scorer.Rank("add to cart", "button", "clickable",
		[]dom.ElementSnapshot{bike, fleece}, 10, anchor)

	if ranked[0].Element.HTMLId != "add-to-cart-sauce-labs-fleece-jacket" {
		t.Errorf("fleece jacket button should win via anchor attr affinity, got %s",
			ranked[0].Element.HTMLId)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section F: Proximity weight boost
// ═══════════════════════════════════════════════════════════════════════════════

func TestProximity_WeightBoostedWithNear(t *testing.T) {
	// Use RawScore instead of Total (which is clamped to 1.0).
	// With NEAR active, proximity weight is 1.5 (vs default 0.10).
	anchor := &scorer.AnchorContext{
		Rect: dom.Rect{Top: 100, Left: 100, Bottom: 130, Right: 200, Width: 100, Height: 30},
	}
	close := makeElWithRect("close", "button", "Action", 100, 210, 130, 310,
		"/html/body/div/button[1]")
	far := makeElWithRect("far", "button", "Action", 300, 400, 330, 500,
		"/html/body/div/button[2]")

	closeScore := scorer.Score("action", "button", "clickable", &close, anchor)
	farScore := scorer.Score("action", "button", "clickable", &far, anchor)

	diff := closeScore.RawScore - farScore.RawScore
	if diff <= 0 {
		t.Errorf("NEAR weight boost should make proximity matter more: close=%.4f far=%.4f",
			closeScore.RawScore, farScore.RawScore)
	}

	// Without NEAR, proximity difference should be much smaller
	closeNoNear := scorer.Score("action", "button", "clickable", &close, nil)
	farNoNear := scorer.Score("action", "button", "clickable", &far, nil)
	diffNoNear := math.Abs(closeNoNear.RawScore - farNoNear.RawScore)

	if diff <= diffNoNear {
		t.Errorf("NEAR proximity diff=%.4f should exceed no-NEAR diff=%.4f", diff, diffNoNear)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section G: Default XPath proximity (no NEAR)
// ═══════════════════════════════════════════════════════════════════════════════

func TestProximity_DefaultXPathProximity(t *testing.T) {
	// Without NEAR, scoreDepth is used (shallow elements score higher)
	shallow := makeEl(withTag("button"), withText("Submit"),
		func(e *dom.ElementSnapshot) { e.XPath = "/html/body/div/button" },
		withID("shallow"),
	)
	deep := makeEl(withTag("button"), withText("Submit"),
		func(e *dom.ElementSnapshot) { e.XPath = "/html/body/div/div/div/div/div/div/button" },
		withID("deep"),
	)

	shallowScore := scorer.Score("submit", "button", "clickable", &shallow, nil)
	deepScore := scorer.Score("submit", "button", "clickable", &deep, nil)

	// Both should have some proximity score
	if shallowScore.ProximityScore < 0 {
		t.Errorf("shallow proximity should be ≥ 0, got %.4f", shallowScore.ProximityScore)
	}
	if deepScore.ProximityScore < 0 {
		t.Errorf("deep proximity should be ≥ 0, got %.4f", deepScore.ProximityScore)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section H: Edge cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestProximity_IdenticalPositions(t *testing.T) {
	anchor := &scorer.AnchorContext{
		Rect:  dom.Rect{Top: 100, Left: 200, Bottom: 130, Right: 300, Width: 100, Height: 30},
		XPath: "/html/body/div/label",
	}
	el1 := makeElWithRect("el1", "button", "OK", 100, 200, 130, 300,
		"/html/body/div/button[1]")
	el2 := makeElWithRect("el2", "button", "OK", 100, 200, 130, 300,
		"/html/body/div/button[2]")

	score1 := scorer.Score("ok", "button", "clickable", &el1, anchor)
	score2 := scorer.Score("ok", "button", "clickable", &el2, anchor)

	// At identical positions, scores should be equal or very close
	if math.Abs(score1.ProximityScore-score2.ProximityScore) > 0.01 {
		t.Errorf("identical positions should give equal proximity: %.4f vs %.4f",
			score1.ProximityScore, score2.ProximityScore)
	}
}

func TestProximity_NearNoXPath(t *testing.T) {
	// When no XPath available, should fall back to pure spatial distance
	anchor := &scorer.AnchorContext{
		Rect:       dom.Rect{Top: 100, Left: 100, Bottom: 130, Right: 200, Width: 100, Height: 30},
		FrameIndex: 0,
		// No XPath
	}
	el := makeElWithRect("noXP", "button", "Save", 100, 210, 130, 310, "")

	score := scorer.Score("save", "button", "clickable", &el, anchor)
	// Should still get proximity from spatial distance
	if score.ProximityScore <= 0 {
		t.Errorf("NEAR without XPath should still use spatial distance, got %.4f", score.ProximityScore)
	}
}

func TestProximity_NearCrossFrameCandidateRejected(t *testing.T) {
	anchor := &scorer.AnchorContext{
		Rect:       dom.Rect{Top: 100, Left: 100, Bottom: 130, Right: 200, Width: 100, Height: 30},
		FrameIndex: 0,
	}
	el := makeElWithRect("iframe-btn", "button", "Save", 110, 120, 140, 220, "/html/body/button[1]")
	el.FrameIndex = 1

	score := scorer.Score("save", "button", "clickable", &el, anchor)
	if score.ProximityScore != 0.0 {
		t.Errorf("cross-frame NEAR should give 0 proximity, got %.4f", score.ProximityScore)
	}
}

func TestProximity_AnchorWordsEmpty(t *testing.T) {
	// Anchor with no words — attr affinity should be 0
	anchor := &scorer.AnchorContext{
		Rect:  dom.Rect{Top: 100, Left: 100, Bottom: 130, Right: 200, Width: 100, Height: 30},
		Words: nil,
	}
	el := makeElWithRect("el", "button", "Save", 100, 210, 130, 310,
		"/html/body/div/button[1]")

	score := scorer.Score("save", "button", "clickable", &el, anchor)
	// Should still have proximity from spatial distance, just no attr affinity bonus
	if score.ProximityScore < 0 {
		t.Errorf("proximity should be ≥ 0 even without anchor words, got %.4f", score.ProximityScore)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section I: Ranking with multiple NEAR candidates
// ═══════════════════════════════════════════════════════════════════════════════

func TestProximity_NearRankingFiveCandidates(t *testing.T) {
	anchor := &scorer.AnchorContext{
		Rect: dom.Rect{Top: 200, Left: 200, Bottom: 230, Right: 300, Width: 100, Height: 30},
	}
	// 5 candidates at increasing distances
	els := []dom.ElementSnapshot{
		makeElWithRect("d350", "button", "Save", 500, 400, 530, 500, "/html/body/div[5]/button"),
		makeElWithRect("d50", "button", "Save", 220, 260, 250, 360, "/html/body/div[1]/button"),
		makeElWithRect("d600", "button", "Save", 700, 600, 730, 700, "/html/body/div[6]/button"),
		makeElWithRect("d150", "button", "Save", 300, 350, 330, 450, "/html/body/div[3]/button"),
		makeElWithRect("d100", "button", "Save", 260, 310, 290, 410, "/html/body/div[2]/button"),
	}

	ranked := scorer.Rank("save", "button", "clickable", els, 5, anchor)

	// The closest (d50) should rank first
	if ranked[0].Element.HTMLId != "d50" {
		t.Errorf("closest candidate should rank first, got %s", ranked[0].Element.HTMLId)
	}
}
