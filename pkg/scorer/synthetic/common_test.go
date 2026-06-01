package synthetic

import (
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
	"github.com/alexbeatnik/ManulHeart/pkg/scorer"
)

// ── Helpers for Synthetic Tests ──────────────────────────────────────────────

func makeEl(opts ...func(*dom.ElementSnapshot)) dom.ElementSnapshot {
	el := dom.ElementSnapshot{
		ID:        1,
		XPath:     "/html/body/div[1]",
		Tag:       "div",
		IsVisible: true,
	}
	for _, o := range opts {
		o(&el)
	}
	el.Normalize()
	return el
}

func withTag(tag string) func(*dom.ElementSnapshot)     { return func(e *dom.ElementSnapshot) { e.Tag = tag } }
func withInputType(t string) func(*dom.ElementSnapshot) { return func(e *dom.ElementSnapshot) { e.InputType = t } }
func withText(text string) func(*dom.ElementSnapshot)   { return func(e *dom.ElementSnapshot) { e.VisibleText = text } }
func withLabel(label string) func(*dom.ElementSnapshot) { return func(e *dom.ElementSnapshot) { e.LabelText = label } }
func withID(id string) func(*dom.ElementSnapshot)       { return func(e *dom.ElementSnapshot) { e.HTMLId = id } }
func withRole(role string) func(*dom.ElementSnapshot)     { return func(e *dom.ElementSnapshot) { e.Role = role } }
func withAriaLabel(a string) func(*dom.ElementSnapshot)   { return func(e *dom.ElementSnapshot) { e.AriaLabel = a } }
func withXPath(x string) func(*dom.ElementSnapshot)       { return func(e *dom.ElementSnapshot) { e.XPath = x } }
func withDisabled() func(*dom.ElementSnapshot)            { return func(e *dom.ElementSnapshot) { e.IsDisabled = true } }
func withHidden() func(*dom.ElementSnapshot)              { return func(e *dom.ElementSnapshot) { e.IsHidden = true; e.IsVisible = false } }
func withAccessibleName(n string) func(*dom.ElementSnapshot) { return func(e *dom.ElementSnapshot) { e.AccessibleName = n } }
func withValue(v string) func(*dom.ElementSnapshot)          { return func(e *dom.ElementSnapshot) { e.Value = v } }

func rankFirst(t *testing.T, query, typeHint, mode string, elements []dom.ElementSnapshot) scorer.RankedCandidate {
	t.Helper()
	ranked := scorer.Rank(query, typeHint, mode, elements, 10, nil)
	if len(ranked) == 0 {
		t.Fatalf("Rank returned 0 candidates for query=%q mode=%s", query, mode)
	}
	return ranked[0]
}
