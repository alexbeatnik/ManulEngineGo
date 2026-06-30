package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// VERIFY CHECKED / NOT CHECKED — Checkbox & Radio State Test Suite
//
// Port of ManulEngine test_34_verify_checked.py — 20 state assertions.
//
// The Python version uses browser + ManulEngine._handle_verify().
// The Go version tests element resolution via scorer.Rank(): given a set of
// checkboxes and radios with their labels, the scorer must rank the correct
// element first. The "checked" state itself is a Playwright runtime check,
// so we test the targeting side: can the scorer find the right checkbox/radio
// by label, aria-label, or data-qa?
//
// Validates:
// 1. Simple checkboxes found by label text
// 2. Radio buttons found by label text
// 3. Aria-label-only checkboxes
// 4. Data-QA identified checkboxes (via label)
// 5. Checkbox inside forms with other elements
// 6. Multiple radio groups
// 7. Fieldset groups with many checkboxes
// ─────────────────────────────────────────────────────────────────────────────

import (
	"github.com/alexbeatnik/ManulEngineGo/pkg/scorer"
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func allCheckboxElements() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Group 1: simple checkboxes
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Newsletter"), withID("chk_on")),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Promotions"), withID("chk_off")),
		// Group 2: radios
		makeEl(withTag("input"), withInputType("radio"), withLabel("Pro Plan"), withID("rad_sel")),
		makeEl(withTag("input"), withInputType("radio"), withLabel("Free Plan"), withID("rad_unsel")),
		// Group 3: aria-label only
		makeEl(withTag("input"), withInputType("checkbox"), withAriaLabel("Accept Terms"), withID("chk_aria_on")),
		makeEl(withTag("input"), withInputType("checkbox"), withAriaLabel("Subscribe Updates"), withID("chk_aria_off")),
		// Group 4: data-qa with label
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Agree to TOS"), withID("chk_dqa_on"),
			func(e *dom.ElementSnapshot) { e.DataQA = "agree-tos" }),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Opt-in Marketing"), withID("chk_dqa_off"),
			func(e *dom.ElementSnapshot) { e.DataQA = "opt-marketing" }),
		// Group 5: inside form
		makeEl(withTag("input"), withInputType("text"), withPlaceholder("Username"), withID("form_user")),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Remember Me"), withID("chk_form_on")),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Save Card"), withID("chk_form_off")),
		makeEl(withTag("button"), withText("Login"), withID("form_submit")),
		// Group 6: second radio group
		makeEl(withTag("input"), withInputType("radio"), withLabel("Express Shipping"), withID("rad2_sel")),
		makeEl(withTag("input"), withInputType("radio"), withLabel("Standard Shipping"), withID("rad2_unsel")),
		// Group 7: fieldset preferences
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Email Alerts"), withID("pref_on1")),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("SMS Alerts"), withID("pref_off1")),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Push Notifications"), withID("pref_on2")),
		makeEl(withTag("input"), withInputType("checkbox"), withLabel("Weekly Digest"), withID("pref_off2")),
	}
}

// verifyTargetFound ensures the scorer ranks the expected element first.
func verifyTargetFound(t *testing.T, query, expectedID string, elements []dom.ElementSnapshot) {
	t.Helper()
	ranked := scorer.Rank(query, "", "checkbox", elements, 10, nil)
	if len(ranked) == 0 {
		t.Fatalf("Rank returned 0 for query=%q", query)
	}
	if ranked[0].Element.HTMLId != expectedID {
		t.Errorf("expected %s for query=%q, got %s", expectedID, query, ranked[0].Element.HTMLId)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 1: Simple checkboxes
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyChecked_Newsletter(t *testing.T) {
	verifyTargetFound(t, "newsletter", "chk_on", allCheckboxElements())
}

func TestVerifyChecked_Promotions(t *testing.T) {
	verifyTargetFound(t, "promotions", "chk_off", allCheckboxElements())
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 2: Radio buttons
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyChecked_ProPlan(t *testing.T) {
	verifyTargetFound(t, "pro plan", "rad_sel", allCheckboxElements())
}

func TestVerifyChecked_FreePlan(t *testing.T) {
	verifyTargetFound(t, "free plan", "rad_unsel", allCheckboxElements())
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 3: Aria-label only
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyChecked_AcceptTerms(t *testing.T) {
	verifyTargetFound(t, "accept terms", "chk_aria_on", allCheckboxElements())
}

func TestVerifyChecked_SubscribeUpdates(t *testing.T) {
	verifyTargetFound(t, "subscribe updates", "chk_aria_off", allCheckboxElements())
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 4: Data-QA identified
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyChecked_AgreeToTOS(t *testing.T) {
	verifyTargetFound(t, "agree to tos", "chk_dqa_on", allCheckboxElements())
}

func TestVerifyChecked_OptInMarketing(t *testing.T) {
	verifyTargetFound(t, "opt-in marketing", "chk_dqa_off", allCheckboxElements())
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 5: Inside form
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyChecked_RememberMe(t *testing.T) {
	verifyTargetFound(t, "remember me", "chk_form_on", allCheckboxElements())
}

func TestVerifyChecked_SaveCard(t *testing.T) {
	verifyTargetFound(t, "save card", "chk_form_off", allCheckboxElements())
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 6: Second radio group
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyChecked_ExpressShipping(t *testing.T) {
	verifyTargetFound(t, "express shipping", "rad2_sel", allCheckboxElements())
}

func TestVerifyChecked_StandardShipping(t *testing.T) {
	verifyTargetFound(t, "standard shipping", "rad2_unsel", allCheckboxElements())
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 7: Fieldset preferences
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyChecked_EmailAlerts(t *testing.T) {
	verifyTargetFound(t, "email alerts", "pref_on1", allCheckboxElements())
}

func TestVerifyChecked_SMSAlerts(t *testing.T) {
	verifyTargetFound(t, "sms alerts", "pref_off1", allCheckboxElements())
}

func TestVerifyChecked_PushNotifications(t *testing.T) {
	verifyTargetFound(t, "push notifications", "pref_on2", allCheckboxElements())
}

func TestVerifyChecked_WeeklyDigest(t *testing.T) {
	verifyTargetFound(t, "weekly digest", "pref_off2", allCheckboxElements())
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cross-validation (same elements, reconfirm)
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyChecked_CrossVal_Newsletter(t *testing.T) {
	verifyTargetFound(t, "newsletter", "chk_on", allCheckboxElements())
}

func TestVerifyChecked_CrossVal_ProPlan(t *testing.T) {
	verifyTargetFound(t, "pro plan", "rad_sel", allCheckboxElements())
}

func TestVerifyChecked_CrossVal_AcceptTerms(t *testing.T) {
	verifyTargetFound(t, "accept terms", "chk_aria_on", allCheckboxElements())
}

func TestVerifyChecked_CrossVal_AgreeToTOS(t *testing.T) {
	verifyTargetFound(t, "agree to tos", "chk_dqa_on", allCheckboxElements())
}
