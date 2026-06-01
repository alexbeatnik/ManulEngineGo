package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// EXPLAIN MODE TEST SUITE
//
// Port of ManulEngine test_44_explain_mode.py
//
// In Go the scorer.ScoreBreakdown is always populated (no explain flag).
// Tests validate:
// 1. All breakdown fields are populated for a matching element
// 2. Field values are in [0.0, 1.0]
// 3. Disabled penalty zeros total and raw score
// 4. Hidden penalty reduces total
// 5. Multiple elements all get scored via scorer.Rank()
// 6. Signals list is populated in Candidate
// 7. Channel sum consistency (text+id+semantic+proximity ≈ raw / penalty)
// 8. Rank order is reflected in Candidate.Rank field
// ─────────────────────────────────────────────────────────────────────────────

import (
	"github.com/alexbeatnik/ManulHeart/pkg/scorer"
	"math"
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Section 1: Breakdown fields populated
// ═══════════════════════════════════════════════════════════════════════════════

func TestExplain_BreakdownFieldsPopulated(t *testing.T) {
	el := makeEl(withTag("button"), withText("Login"))
	score := scorer.Score("login", "button", "clickable", &el, nil)

	if score.ExactTextMatch <= 0 {
		t.Errorf("ExactTextMatch should be > 0 for exact match, got %.4f", score.ExactTextMatch)
	}
	if score.TagSemantics <= 0 {
		t.Errorf("TagSemantics should be > 0 for button in clickable mode, got %.4f", score.TagSemantics)
	}
	if score.VisibilityScore <= 0 {
		t.Errorf("VisibilityScore should be > 0 for visible element, got %.4f", score.VisibilityScore)
	}
	if score.InteractabilityScore <= 0 {
		t.Errorf("InteractabilityScore should be > 0 for enabled element, got %.4f", score.InteractabilityScore)
	}
	if score.Total <= 0 {
		t.Errorf("Total should be > 0 for matching element, got %.4f", score.Total)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section 2: All values in [0, 1]
// ═══════════════════════════════════════════════════════════════════════════════

func TestExplain_AllValuesInRange(t *testing.T) {
	el := makeEl(
		withTag("input"), withText(""), withInputType("text"),
		withLabel("Username"), withAriaLabel("username field"),
		func(e *dom.ElementSnapshot) {
			e.Placeholder = "Enter username"
			e.DataQA = "user-input"
			e.HTMLId = "username"
		},
	)
	score := scorer.Score("username", "", "input", &el, nil)

	fields := map[string]float64{
		"ExactTextMatch":      score.ExactTextMatch,
		"NormalizedTextMatch":  score.NormalizedTextMatch,
		"LabelMatch":          score.LabelMatch,
		"PlaceholderMatch":    score.PlaceholderMatch,
		"AriaMatch":           score.AriaMatch,
		"DataQAMatch":         score.DataQAMatch,
		"IDMatch":             score.IDMatch,
		"TagSemantics":        score.TagSemantics,
		"TypeHintAlignment":   score.TypeHintAlignment,
		"VisibilityScore":     score.VisibilityScore,
		"InteractabilityScore": score.InteractabilityScore,
		"ProximityScore":      score.ProximityScore,
		"Total":               score.Total,
	}

	for name, val := range fields {
		if val < 0.0 || val > 1.0 {
			t.Errorf("%s = %.4f, expected in [0.0, 1.0]", name, val)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section 3: Disabled penalty zeros total
// ═══════════════════════════════════════════════════════════════════════════════

func TestExplain_DisabledPenaltyZero(t *testing.T) {
	el := makeEl(withTag("button"), withText("Login"), withDisabled())
	score := scorer.Score("login", "button", "clickable", &el, nil)

	if score.Total != 0.0 {
		t.Errorf("disabled element Total should be 0.0, got %.4f", score.Total)
	}
	if score.RawScore != 0.0 {
		t.Errorf("disabled element RawScore should be 0.0, got %.4f", score.RawScore)
	}
	if score.InteractabilityScore != 0.0 {
		t.Errorf("disabled InteractabilityScore should be 0.0, got %.4f", score.InteractabilityScore)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section 4: Hidden penalty reduces total
// ═══════════════════════════════════════════════════════════════════════════════

func TestExplain_HiddenPenaltyReduced(t *testing.T) {
	visible := makeEl(withTag("button"), withText("Login"))
	hidden := makeEl(withTag("button"), withText("Login"), withHidden())

	visScore := scorer.Score("login", "button", "clickable", &visible, nil)
	hidScore := scorer.Score("login", "button", "clickable", &hidden, nil)

	if hidScore.Total >= visScore.Total {
		t.Errorf("hidden total (%.4f) should be < visible total (%.4f)",
			hidScore.Total, visScore.Total)
	}
	if hidScore.VisibilityScore >= visScore.VisibilityScore {
		t.Errorf("hidden visibility (%.4f) should be < visible visibility (%.4f)",
			hidScore.VisibilityScore, visScore.VisibilityScore)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section 5: Multiple elements — all get scored
// ═══════════════════════════════════════════════════════════════════════════════

func TestExplain_MultipleElementsAllScored(t *testing.T) {
	els := []dom.ElementSnapshot{
		makeEl(withTag("button"), withText("Login"), withID("btn1")),
		makeEl(withTag("button"), withText("Logout"), withID("btn2")),
		makeEl(withTag("button"), withText("Submit"), withID("btn3")),
	}

	ranked := scorer.Rank("login", "button", "clickable", els, 10, nil)

	if len(ranked) != 3 {
		t.Fatalf("expected 3 ranked candidates, got %d", len(ranked))
	}

	for i, rc := range ranked {
		if rc.Explain.Score.Total < 0 {
			t.Errorf("candidate %d has negative total: %.4f", i, rc.Explain.Score.Total)
		}
		if rc.Explain.Rank != i+1 {
			t.Errorf("candidate %d has rank %d, expected %d", i, rc.Explain.Rank, i+1)
		}
	}

	// First should have the best score (Login matches "login")
	if ranked[0].Explain.Score.Total < ranked[1].Explain.Score.Total {
		t.Errorf("best scoring element should rank first")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section 6: Signals list populated
// ═══════════════════════════════════════════════════════════════════════════════

func TestExplain_SignalsPopulated(t *testing.T) {
	el := makeEl(withTag("button"), withText("Login"))
	ranked := scorer.Rank("login", "button", "clickable",
		[]dom.ElementSnapshot{el}, 1, nil)

	if len(ranked) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(ranked))
	}
	if len(ranked[0].Explain.Signals) == 0 {
		t.Error("expected non-empty Signals list for matching element")
	}

	// Check that at least one signal mentions text
	hasTextSignal := false
	for _, sig := range ranked[0].Explain.Signals {
		if sig.Signal == "exact_text" || sig.Signal == "norm_text" {
			hasTextSignal = true
			break
		}
	}
	if !hasTextSignal {
		t.Error("expected a text-related signal for exact match 'Login'")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section 7: Chosen flag
// ═══════════════════════════════════════════════════════════════════════════════

func TestExplain_ChosenFlag(t *testing.T) {
	els := []dom.ElementSnapshot{
		makeEl(withTag("button"), withText("Login"), withID("btn1")),
		makeEl(withTag("button"), withText("Cancel"), withID("btn2")),
	}

	ranked := scorer.Rank("login", "button", "clickable", els, 10, nil)

	if !ranked[0].Explain.Chosen {
		t.Error("first-ranked candidate should have Chosen=true")
	}
	if ranked[1].Explain.Chosen {
		t.Error("second-ranked candidate should have Chosen=false")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Section 8: Channel score consistency
// ═══════════════════════════════════════════════════════════════════════════════

func TestExplain_ChannelScoreConsistency(t *testing.T) {
	// For a non-penalized element, raw ≈ text*wt + attributes*wa + semantic*ws + proximity*wp
	el := makeEl(withTag("button"), withText("Login"))
	score := scorer.Score("login", "button", "clickable", &el, nil)

	textCat := score.ExactTextMatch + score.NormalizedTextMatch + score.LabelMatch + score.PlaceholderMatch + score.AriaMatch + score.DataQAMatch
	attrCat := score.IDMatch
	// Mode synergy (+0.5) applies for perfect text match + clickable mode + real button
	semCat := score.TagSemantics + score.TypeHintAlignment + 0.5
	proxCat := score.ProximityScore

	recomputed := textCat*scorer.Weights.Text +
		attrCat*scorer.Weights.Attributes +
		semCat*scorer.Weights.Semantic +
		proxCat*scorer.Weights.Proximity

	if math.Abs(score.RawScore-recomputed) > 0.001 {
		t.Errorf("recomputed (%.4f) too far from RawScore (%.4f); channel sum inconsistent",
			recomputed, score.RawScore)
	}
}
