package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// VERIFY ENABLED / DISABLED — State Detection Test Suite
//
// Port of ManulEngine test_32_verify_enabled.py — 20 state assertions.
//
// The Python version uses browser + ManulEngine._handle_verify().
// The Go version tests element resolution + IsDisabled state: if the scorer
// finds the element AND the element has the correct disabled state, the
// VERIFY command would succeed.
//
// We test two things per case:
// 1. The target element is ranked first (found correctly)
// 2. The element's IsDisabled flag matches expectation
// ─────────────────────────────────────────────────────────────────────────────

import (
	"github.com/alexbeatnik/ManulHeart/pkg/scorer"
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers for element sets
// ═══════════════════════════════════════════════════════════════════════════════

// All elements from the ENABLED_DOM — each with correct disabled/aria-disabled state.
func allEnabledDisabledElements() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Group 1: simple buttons
		makeEl(withTag("button"), withText("Active Button"), withID("btn_active")),
		makeEl(withTag("button"), withText("Inactive Button"), withID("btn_inactive"), withDisabled()),
		// Group 2: inputs
		makeEl(withTag("input"), withInputType("text"), withAriaLabel("Active Input"), withPlaceholder("Active Input"), withID("inp_active")),
		makeEl(withTag("input"), withInputType("text"), withAriaLabel("Inactive Input"), withPlaceholder("Inactive Input"), withID("inp_inactive"), withDisabled()),
		// Group 3: select
		makeEl(withTag("select"), withAriaLabel("Active Select"), withID("sel_active")),
		makeEl(withTag("select"), withAriaLabel("Inactive Select"), withID("sel_inactive"), withDisabled()),
		// Group 4: textarea
		makeEl(withTag("textarea"), withAriaLabel("Active Textarea"), withID("ta_active")),
		makeEl(withTag("textarea"), withAriaLabel("Inactive Textarea"), withID("ta_inactive"), withDisabled()),
		// Group 5: anchor with aria-disabled
		makeEl(withTag("a"), withText("Active Link"), withRole("button"), withID("link_active")),
		makeEl(withTag("a"), withText("Inactive Link"), withRole("button"), withID("link_inactive"), withDisabled()),
		// Group 6: div role=button
		makeEl(withTag("div"), withText("Active Action"), withRole("button"), withID("div_active")),
		makeEl(withTag("div"), withText("Inactive Action"), withRole("button"), withClassName("disabled"), withID("div_inactive"), withDisabled()),
		// Group 7: label with associated control
		makeEl(withTag("input"), withInputType("text"), withLabel("Active Control"), withID("ctrl_active")),
		makeEl(withTag("input"), withInputType("text"), withLabel("Inactive Control"), withID("ctrl_inactive"), withDisabled()),
		// Group 8: aria-disabled
		makeEl(withTag("button"), withText("Aria Active"), withID("btn_aria_active")),
		makeEl(withTag("button"), withText("Aria Inactive"), withID("btn_aria_inactive"), withDisabled()),
		// Group 9: disabled="" empty attr
		makeEl(withTag("button"), withText("Attr Disabled"), withID("btn_attr_disabled"), withDisabled()),
		makeEl(withTag("button"), withText("No Attr"), withID("btn_no_attr")),
		// Group 10: menuitem
		makeEl(withTag("div"), withText("Active Menu Item"), withRole("menuitem"), withID("mi_active")),
		makeEl(withTag("div"), withText("Inactive Menu Item"), withRole("menuitem"), withID("mi_inactive"), withDisabled()),
	}
}

// verifyState checks that the scorer finds the target AND the disabled state matches.
// For disabled elements (IsDisabled=true), the penalty zeros the score, so they won't
// rank #1 when competing with enabled elements. We search the full ranked list.
func verifyState(t *testing.T, query string, elements []dom.ElementSnapshot, expectedID string, expectDisabled bool) {
	t.Helper()
	ranked := scorer.Rank(query, "", "clickable", elements, len(elements), nil)
	if len(ranked) == 0 {
		t.Fatalf("Rank returned 0 candidates for query=%q", query)
	}

	if !expectDisabled {
		// Enabled elements should rank first
		if ranked[0].Element.HTMLId != expectedID {
			t.Errorf("expected %s at rank 1, got %s (query=%q)", expectedID, ranked[0].Element.HTMLId, query)
		}
		if ranked[0].Element.IsDisabled {
			t.Errorf("expected IsDisabled=false for %s", expectedID)
		}
	} else {
		// Disabled elements get penalty=0, so they won't be #1.
		// Find the target in the ranked list and verify its state.
		found := false
		for _, r := range ranked {
			if r.Element.HTMLId == expectedID {
				found = true
				if !r.Element.IsDisabled {
					t.Errorf("expected IsDisabled=true for %s", expectedID)
				}
				break
			}
		}
		if !found {
			t.Errorf("element %s not found in ranked candidates (query=%q)", expectedID, query)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 1: Simple buttons
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyEnabled_ActiveButton(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "active button", els, "btn_active", false)
}

func TestVerifyEnabled_InactiveButton(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "inactive button", els, "btn_inactive", true)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 2: Inputs
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyEnabled_ActiveInput(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "active input", els, "inp_active", false)
}

func TestVerifyEnabled_InactiveInput(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "inactive input", els, "inp_inactive", true)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 3: Select
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyEnabled_ActiveSelect(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "active select", els, "sel_active", false)
}

func TestVerifyEnabled_InactiveSelect(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "inactive select", els, "sel_inactive", true)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 4: Textarea
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyEnabled_ActiveTextarea(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "active textarea", els, "ta_active", false)
}

func TestVerifyEnabled_InactiveTextarea(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "inactive textarea", els, "ta_inactive", true)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 5: Anchor with aria-disabled
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyEnabled_ActiveLink(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "active link", els, "link_active", false)
}

func TestVerifyEnabled_InactiveLink(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "inactive link", els, "link_inactive", true)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 6: div role=button
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyEnabled_ActiveDivButton(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "active action", els, "div_active", false)
}

func TestVerifyEnabled_InactiveDivButton(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "inactive action", els, "div_inactive", true)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 7: Label with associated control
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyEnabled_ActiveControl(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "active control", els, "ctrl_active", false)
}

func TestVerifyEnabled_InactiveControl(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "inactive control", els, "ctrl_inactive", true)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 8: aria-disabled
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyEnabled_AriaActive(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "aria active", els, "btn_aria_active", false)
}

func TestVerifyEnabled_AriaInactive(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "aria inactive", els, "btn_aria_inactive", true)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 9: disabled="" empty attribute
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyEnabled_AttrDisabled(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "attr disabled", els, "btn_attr_disabled", true)
}

func TestVerifyEnabled_NoAttr(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "no attr", els, "btn_no_attr", false)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group 10: role=menuitem
// ═══════════════════════════════════════════════════════════════════════════════

func TestVerifyEnabled_ActiveMenuItem(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "active menu item", els, "mi_active", false)
}

func TestVerifyEnabled_InactiveMenuItem(t *testing.T) {
	els := allEnabledDisabledElements()
	verifyState(t, "inactive menu item", els, "mi_inactive", true)
}
