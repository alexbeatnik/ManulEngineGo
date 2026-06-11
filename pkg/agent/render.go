package agent

import (
	"fmt"
	"strings"
)

// RenderForLLM turns a PageMap into a compact, prompt-ready block: one section
// per landmark group, each element on its own line. Groups are already
// deduped, ranked and budgeted by Map; this only handles presentation, so any
// embedding app gets LLM-ready text without re-rendering.
//
// The format is tuned for small local models:
//   - a one-line header tells the model HOW to use the list (exact quoted
//     text as targets) — without it, weak models paraphrase labels;
//   - every label is single-quoted, so the model can copy it verbatim
//     (including inner spaces) straight into a CLICK/FILL command;
//   - every role is paired with the DSL verb that works on it
//     ("link — CLICK", "textbox — FILL"), so the model never has to infer
//     which command fits which element.
//
// maxPerGroup further caps how many elements are PRINTED per group (0 → all
// that Map kept). A trailing "… +N more" line combines elements hidden by this
// display cap with those Map already dropped (MapGroup.Truncated), so the count
// reflects the real page, not just what survived rendering.
func (pm PageMap) RenderForLLM(maxPerGroup int) string {
	if len(pm.Groups) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Interactive elements on the page. To act on one, copy the EXACT text in quotes as your target.\n")
	if pm.URL != "" {
		fmt.Fprintf(&b, "Current page: %s\n", pm.URL)
	}
	for _, g := range pm.Groups {
		if len(g.Elements) == 0 {
			continue
		}
		shown := g.Elements
		hiddenByDisplay := 0
		if maxPerGroup > 0 && len(shown) > maxPerGroup {
			hiddenByDisplay = len(shown) - maxPerGroup
			shown = shown[:maxPerGroup]
		}

		fmt.Fprintf(&b, "[%s] (%d items)\n", g.Name, len(g.Elements)+g.Truncated)
		for _, e := range shown {
			label := strings.TrimSpace(e.Label)
			role := e.Role
			if role == "" {
				role = "element"
			}
			fmt.Fprintf(&b, "  - '%s' [%s — %s]\n", label, role, roleActionVerb(role))
		}
		if more := hiddenByDisplay + g.Truncated; more > 0 {
			fmt.Fprintf(&b, "  … +%d more\n", more)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// roleActionVerb maps an element role to the DSL verb an agent should use on
// it, so a model reading the page map never has to guess which command fits.
func roleActionVerb(role string) string {
	switch strings.ToLower(role) {
	case "textbox", "searchbox", "spinbutton", "textarea", "input":
		return "FILL"
	case "combobox", "listbox", "select":
		return "SELECT"
	case "checkbox", "switch":
		return "CHECK"
	default:
		// link, button, menuitem, tab, option, radio, headings, generic tags.
		return "CLICK"
	}
}
