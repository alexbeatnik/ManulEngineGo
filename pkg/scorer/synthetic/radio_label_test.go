package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// RADIO/CHECKBOX LABEL DISAMBIGUATION — regression test for the
// rahulshettyacademy.com bug where <label for="radio1"> wraps an <input> that
// has no matching id="radio1". Clicking the label does nothing (the `for`
// takes precedence over the implicit wrapping association), so the scorer
// must pick the <input type=radio>, not the label.
//
// Two failure modes are covered:
//   1. typeHint="radio" from the parser ("radio button" compound hint) must
//      boost the input over the label.
//   2. typeHint="button" from a plain "radio button" — even without the
//      radio-specific boost, the input should still win because the probe no
//      longer flags radios as IsEditable (which previously triggered a -1.0
//      cross-mode penalty via isRealInput).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func rahulRadioDOM() []dom.ElementSnapshot {
	// Label+input pairs mirroring the real page markup. Labels carry the
	// visible "Radio1/2/3" text and a for= that points to a nonexistent id;
	// inputs are nested inside and only carry value="radio{n}" + className.
	return []dom.ElementSnapshot{
		// Radio1
		el(1, "/html/body/fieldset/label[1]",
			withTag("label"),
			withText("Radio1"),
			withLabel("Radio1"),
		),
		el(2, "/html/body/fieldset/label[1]/input",
			withTag("input"), withInputType("radio"),
			withClassName("radioButton"),
			withNameAttr("radioButton"),
			withValue("radio1"),
			withLabel("Radio1"),
			withAccessibleName("Radio1"),
		),
		// Radio2
		el(3, "/html/body/fieldset/label[2]",
			withTag("label"),
			withText("Radio2"),
			withLabel("Radio2"),
		),
		el(4, "/html/body/fieldset/label[2]/input",
			withTag("input"), withInputType("radio"),
			withClassName("radioButton"),
			withNameAttr("radioButton"),
			withValue("radio2"),
			withLabel("Radio2"),
			withAccessibleName("Radio2"),
		),
		// Radio3
		el(5, "/html/body/fieldset/label[3]",
			withTag("label"),
			withText("Radio3"),
			withLabel("Radio3"),
		),
		el(6, "/html/body/fieldset/label[3]/input",
			withTag("input"), withInputType("radio"),
			withClassName("radioButton"),
			withNameAttr("radioButton"),
			withValue("radio3"),
			withLabel("Radio3"),
			withAccessibleName("Radio3"),
		),
	}
}

// Winner must be the <input type=radio>, not the wrapping <label>.
// With typeHint="radio" (what the fixed parser produces for "radio button for 'Radio1'").
func TestRadioLabel_InputWinsWithRadioHint(t *testing.T) {
	elements := rahulRadioDOM()
	cases := []struct {
		query    string
		wantTag  string
		wantType string
		wantVal  string
	}{
		{"Radio1", "input", "radio", "radio1"},
		{"Radio2", "input", "radio", "radio2"},
		{"Radio3", "input", "radio", "radio3"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			got := rankFirst(t, tc.query, "radio", "clickable", elements)
			if got.Element.Tag != tc.wantTag || got.Element.InputType != tc.wantType {
				t.Errorf("%s: picked tag=%s type=%s, want tag=%s type=%s (label stole the click)",
					tc.query, got.Element.Tag, got.Element.InputType, tc.wantTag, tc.wantType)
			}
			if got.Element.Value != tc.wantVal {
				t.Errorf("%s: picked value=%q, want %q", tc.query, got.Element.Value, tc.wantVal)
			}
		})
	}
}

// Even with typeHint="button" (the wrong hint the old parser produced), the
// input should still win once the probe stops mislabelling radios as
// IsEditable. This locks in the is_editable fix.
func TestRadioLabel_InputWinsWithButtonHint(t *testing.T) {
	elements := rahulRadioDOM()
	// Sanity-check: if IsEditable were mistakenly true on a radio input (the
	// original probe bug), the scorer would apply a -1.0 cross-mode penalty
	// and the label would win. We assert it's NOT set on radios here so the
	// test stays meaningful if the synthetic fixture is ever changed.
	for _, el := range elements {
		if el.Tag == "input" && el.InputType == "radio" && el.IsEditable {
			t.Fatalf("synthetic radio input %d was flagged IsEditable — that's the bug this test protects against", el.ID)
		}
	}
	got := rankFirst(t, "Radio1", "button", "clickable", elements)
	if got.Element.Tag != "input" || got.Element.InputType != "radio" {
		t.Errorf("picked tag=%s type=%s, want input[type=radio] — label should not win",
			got.Element.Tag, got.Element.InputType)
	}
}
