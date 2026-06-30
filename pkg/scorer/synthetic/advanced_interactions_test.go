package synthetic

import (
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func advancedDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		el(1, "/body/div[1]", withID("image-context"), withTag("img"), withAriaLabel("Image")),
		el(2, "/body/input[1]", withID("search-input"), withTag("input"), withPlaceholder("Search Input")),
		el(3, "/body/button[1]", withID("shadow-btn"), withTag("button"), withText("Shadow Button"),
			func(e *dom.ElementSnapshot) { e.IsInShadow = true }),
	}
}

func TestAdvancedResolution(t *testing.T) {
	elements := advancedDOM()

	tests := []struct {
		name       string
		query      string
		mode       string
		expectedID string
	}{
		{"RightClickImage", "Image", "clickable", "image-context"},
		{"PressOnSearch", "Search Input", "clickable", "search-input"},
		{"RightClickShadow", "Shadow Button", "clickable", "shadow-btn"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rankFirstID(t, tc.query, "", tc.mode, elements)
			if got != tc.expectedID {
				t.Errorf("got %s, want %s", got, tc.expectedID)
			}
		})
	}
}
