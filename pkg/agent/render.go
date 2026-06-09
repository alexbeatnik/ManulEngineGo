package agent

import (
	"fmt"
	"strings"
)

// RenderForLLM turns a PageMap into a compact, prompt-ready block: one section
// per landmark group, each element on its own line as "- <label> [<role>]".
// Groups are already deduped, ranked and budgeted by Map; this only handles
// presentation, so any embedding app gets LLM-ready text without re-rendering.
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
			fmt.Fprintf(&b, "  - %s [%s]\n", label, role)
		}
		if more := hiddenByDisplay + g.Truncated; more > 0 {
			fmt.Fprintf(&b, "  … +%d more\n", more)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
