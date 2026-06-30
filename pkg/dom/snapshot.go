// Package dom provides normalized DOM element modeling for ManulEngine (Go).
//
// ElementSnapshot is the canonical in-memory representation of a single DOM
// element as returned by the in-page JS probe. All engine-side logic
// (scoring, normalization, target resolution) operates on this type.
package dom

import "strings"

// Rect is a bounding box in viewport coordinates (pixels).
type Rect struct {
	Top    float64 `json:"top"`
	Left   float64 `json:"left"`
	Bottom float64 `json:"bottom"`
	Right  float64 `json:"right"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// ElementSnapshot is the normalized representation of a single page element.
// It is populated by the JS probe in pkg/heuristics and enriched by the
// engine-side normalization step in pkg/runtime.
type ElementSnapshot struct {
	// -- Identity -------------------------------------------------------

	// ID is the engine-assigned numeric identifier stored on el.__manulId,
	// allocated from window.__manulIdCounter by the in-page JS probe.
	ID int `json:"id"`
	// Name is the display name (e.g. "Login button [SHADOW_DOM]").
	Name string `json:"name,omitempty"`
	// XPath is the deterministic full XPath computed in-page.
	XPath string `json:"xpath"`
	// Tag is the lowercase HTML tag name.
	Tag string `json:"tag"`
	// InputType is the value of the `type` attribute for input elements.
	InputType string `json:"input_type,omitempty"`
	// FrameIndex is the zero-based frame index this element belongs to.
	// 0 is the main document; positive values refer to embedded frames.
	FrameIndex int `json:"frame_index,omitempty"`

	// -- Text signals ---------------------------------------------------

	// VisibleText is the innerText of the element (trimmed).
	VisibleText string `json:"visible_text,omitempty"`
	// AriaLabel is the aria-label attribute value.
	AriaLabel string `json:"aria_label,omitempty"`
	// AccessibleName is the computed accessible name (via browser heuristics).
	AccessibleName string `json:"accessible_name,omitempty"`
	// Placeholder is the placeholder attribute (inputs/textareas).
	Placeholder string `json:"placeholder,omitempty"`
	// Title is the title attribute.
	Title string `json:"title,omitempty"`
	// DataQA is the data-qa attribute value.
	DataQA string `json:"data_qa,omitempty"`
	// DataTestID is the data-testid attribute value.
	DataTestID string `json:"data_testid,omitempty"`
	// LabelText is the text of a linked <label> element (via for= or wrapping).
	LabelText string `json:"label_text,omitempty"`
	// NameAttr is the name attribute.
	NameAttr string `json:"name_attr,omitempty"`
	// HTMLId is the id attribute.
	HTMLId string `json:"html_id,omitempty"`
	// ClassName is the class attribute string.
	ClassName string `json:"class_name,omitempty"`
	// IconClasses is a space-separated list of classes from icon children.
	IconClasses string `json:"icon_classes,omitempty"`
	// Role is the explicit or inferred ARIA role.
	Role string `json:"role,omitempty"`
	// Value is the current value for input/select elements.
	Value string `json:"value,omitempty"`
	// Ancestors holds the tag names of top 8 parents.
	Ancestors []string `json:"ancestors,omitempty"`

	// -- State ----------------------------------------------------------

	// IsVisible reports whether the element was visible when the snapshot was taken.
	IsVisible bool `json:"is_visible"`
	// IsDisabled reports whether the element has a disabled attribute.
	IsDisabled bool `json:"is_disabled"`
	// IsHidden reports whether the element is hidden via CSS or display:none.
	IsHidden bool `json:"is_hidden"`
	// IsEditable reports whether the element accepts text input.
	IsEditable bool `json:"is_editable"`
	// IsSelect reports whether the element is a native <select>.
	IsSelect bool `json:"is_select"`
	// IsContentEditable reports whether the element has contenteditable="true".
	IsContentEditable bool `json:"is_contenteditable"`
	// IsChecked reports whether the element (checkbox/radio) is checked.
	IsChecked bool `json:"is_checked"`
	// IsSelected reports whether the element is selected (option/aria-selected).
	IsSelected bool `json:"is_selected"`
	// IsInShadow reports whether the element is inside a shadow DOM.
	IsInShadow bool `json:"is_in_shadow,omitempty"`

	// -- Geometry -------------------------------------------------------

	// Rect is the bounding box in viewport coordinates.
	Rect Rect `json:"rect"`

	// -- Normalized fields (populated by engine-side normalization) -----

	// NormText is the lowercase-trimmed visible text, used for scoring.
	NormText string `json:"-"`
	// NormAriaLabel is the lowercase-trimmed aria-label.
	NormAriaLabel string `json:"-"`
	// NormPlaceholder is the lowercase-trimmed placeholder.
	NormPlaceholder string `json:"-"`
	// NormLabelText is the lowercase-trimmed label text.
	NormLabelText string `json:"-"`
	// NormDataQA is the lowercase-trimmed data-qa value.
	NormDataQA string `json:"-"`
	// NormHTMLId is the lowercase-trimmed html id.
	NormHTMLId string `json:"-"`
}

// Normalize populates all Norm* fields from the raw attribute values.
// Must be called before scoring.
func (e *ElementSnapshot) Normalize() {
	e.NormText = norm(e.VisibleText)
	e.NormAriaLabel = norm(e.AriaLabel)
	e.NormPlaceholder = norm(e.Placeholder)
	e.NormLabelText = norm(e.LabelText)
	e.NormDataQA = norm(e.DataQA)
	e.NormHTMLId = norm(e.HTMLId)
}

// IsInteractive reports whether this element is interactable in the given mode.
func (e *ElementSnapshot) IsInteractive(mode string) bool {
	if e.IsDisabled {
		return false
	}
	switch mode {
	case "input":
		if e.Tag == "input" && e.InputType == "file" {
			return false
		}
		return e.IsEditable || e.Tag == "input" || e.Tag == "textarea" ||
			e.Role == "textbox" || e.Role == "spinbutton" || e.Role == "slider"
	case "checkbox":
		return (e.Tag == "input" && (e.InputType == "checkbox" || e.InputType == "radio")) ||
			e.Role == "checkbox" || e.Role == "radio"
	case "select":
		return e.Tag == "select" || e.Role == "listbox" || e.Role == "combobox"
	default: // clickable
		return true
	}
}

// AllTextSignals returns a deduplicated list of all text signals for this element,
// used by the scorer to check for matches across all text-bearing attributes.
func (e *ElementSnapshot) AllTextSignals() []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range []string{
		e.NormText,
		e.NormAriaLabel,
		e.NormPlaceholder,
		e.NormLabelText,
		e.NormDataQA,
		norm(e.Title),
		norm(e.DataTestID),
		norm(e.NameAttr),
		norm(e.Value),
		norm(e.IconClasses),
	} {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// norm lowercases and collapses internal whitespace, identical to the scorer's
// normalize function. Using the same normalization everywhere ensures that
// "First  Name" (double space) compares equal to query "first name".
func norm(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

// PageSnapshot is the result of a full DOM probe: all candidate elements plus
// the page-level visible text used for VERIFY commands.
type PageSnapshot struct {
	// URL is the page URL at snapshot time.
	URL string
	// Title is the <title> element text.
	Title string
	// VisibleText is the full visible text content of the page (lowercased),
	// used for VERIFY presence checks.
	VisibleText string
	// Elements is the ordered list of candidate elements.
	Elements []ElementSnapshot
}
