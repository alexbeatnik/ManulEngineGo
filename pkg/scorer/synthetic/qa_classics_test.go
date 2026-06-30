package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// QA CLASSICS DOM SCORING TEST SUITE
//
// Port of ManulEngine test_14_qa_classics.py — 30-element Rahul Shetty QA
// practice page + blogspot controls.
// Validates: radio buttons, autocomplete, dropdowns, checkboxes, alert/confirm
// buttons, hide/show, tables, hover menu, Wikipedia search, date picker,
// speed/file selects, double-click copy, drag-drop.
// Skipped: extract (11-13,27-29), verify (30), hover/execute_step (14).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func qaClassicsDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Radio buttons
		el(1, "/html/body/input[1]", withTag("input"), withInputType("radio"), withID("rs_rad1"), withNameAttr("radioButton"), withClassName("radioButton"), withLabel("Radio1"), withValue("radio1")),
		el(2, "/html/body/input[2]", withTag("input"), withInputType("radio"), withID("rs_rad2"), withNameAttr("radioButton"), withClassName("radioButton"), withLabel("Radio2"), withValue("radio2")),
		el(3, "/html/body/input[3]", withTag("input"), withInputType("radio"), withID("rs_rad3"), withNameAttr("radioButton"), withClassName("radioButton"), withLabel("Radio3"), withValue("radio3")),
		// Autocomplete
		el(4, "/html/body/input[4]", withTag("input"), withInputType("text"), withID("autocomplete"), withClassName("inputs"), withPlaceholder("Type to Select Countries"), withLabel("Suggession Class Example")),
		// Dropdown
		el(5, "/html/body/select[1]", withTag("select"), withID("dropdown-class-example"), withNameAttr("dropdown-class-example"), withText("Select")),
		// Checkboxes
		el(6, "/html/body/input[5]", withTag("input"), withInputType("checkbox"), withID("checkBoxOption1"), withNameAttr("checkBoxOption1"), withLabel("Option1"), withValue("option1")),
		el(7, "/html/body/input[6]", withTag("input"), withInputType("checkbox"), withID("checkBoxOption2"), withNameAttr("checkBoxOption2"), withLabel("Option2"), withValue("option2")),
		el(8, "/html/body/input[7]", withTag("input"), withInputType("checkbox"), withID("checkBoxOption3"), withNameAttr("checkBoxOption3"), withLabel("Option3"), withValue("option3")),
		// Name input
		el(9, "/html/body/input[8]", withTag("input"), withInputType("text"), withID("name"), withNameAttr("enter-name"), withClassName("inputs"), withPlaceholder("Enter Your Name")),
		// Alert / Confirm buttons
		el(10, "/html/body/input[9]", withTag("input"), withInputType("button"), withID("alertbtn"), withClassName("btn-style"), withValue("Alert")),
		el(11, "/html/body/input[10]", withTag("input"), withInputType("button"), withID("confirmbtn"), withClassName("btn-style"), withValue("Confirm")),
		// Hide / Show
		el(12, "/html/body/input[11]", withTag("input"), withInputType("button"), withID("hide-textbox"), withClassName("btn-style class2"), withValue("Hide")),
		el(13, "/html/body/input[12]", withTag("input"), withInputType("button"), withID("show-textbox"), withClassName("btn-style class2"), withValue("Show")),
		el(14, "/html/body/input[13]", withTag("input"), withInputType("text"), withID("displayed-text"), withNameAttr("show-hide"), withClassName("inputs"), withPlaceholder("Hide/Show Example")),
		// Course table
		el(15, "/html/body/table[1]", withTag("table"), withID("product"), withNameAttr("courses")),
		// Mouse hover
		el(16, "/html/body/button[1]", withTag("button"), withID("mousehover"), withClassName("btn btn-primary"), withText("Mouse Hover")),
		el(17, "/html/body/a[1]", withTag("a"), withID("rs_top"), withText("Top")),
		el(18, "/html/body/a[2]", withTag("a"), withID("rs_reload"), withText("Reload")),
		// Wikipedia search
		el(19, "/html/body/input[14]", withTag("input"), withInputType("text"), withID("Wikipedia1_wikipedia-search-input"), withClassName("wikipedia-search-input"), withLabel("Wikipedia Search:")),
		el(20, "/html/body/input[15]", withTag("input"), withInputType("button"), withID("bs_wiki_btn"), withClassName("wikipedia-search-button"), withValue("🔍")),
		// New window
		el(21, "/html/body/button[2]", withTag("button"), withID("bs_new_window"), withText("New Browser Window")),
		// Date picker
		el(22, "/html/body/input[16]", withTag("input"), withInputType("text"), withID("datepicker"), withLabel("Date Picker:")),
		// Select speed & file
		el(23, "/html/body/select[2]", withTag("select"), withID("speed"), withNameAttr("speed"), withLabel("Select Speed"), withText("Medium")),
		el(24, "/html/body/select[3]", withTag("select"), withID("files"), withNameAttr("files"), withLabel("Select a file"), withText("jQuery.js")),
		// Double click / copy
		el(25, "/html/body/button[3]", withTag("button"), withID("bs_dbl_click"), withText("Copy Text")),
		el(26, "/html/body/input[17]", withTag("input"), withInputType("text"), withID("bs_field1"), withValue("Hello World")),
		el(27, "/html/body/input[18]", withTag("input"), withInputType("text"), withID("bs_field2")),
		// Drag & drop
		el(28, "/html/body/div[1]", withTag("div"), withID("draggable"), withText("Drag me")),
		el(29, "/html/body/div[2]", withTag("div"), withID("droppable"), withText("Drop here")),
		// Book table
		el(30, "/html/body/table[2]", withTag("table"), withID("BookTable"), withNameAttr("BookTable")),
	}
}

func TestQAClassics(t *testing.T) {
	elements := qaClassicsDOM()

	tests := []struct {
		name       string
		query      string
		mode       string
		expectedID string
	}{
		{"Click Radio2", "Radio2", "clickable", "rs_rad2"},
		{"Fill Suggession Class Example", "Suggession Class Example", "input", "autocomplete"},
		{"Check Option1 checkbox", "Option1", "clickable", "checkBoxOption1"},
		{"Check Option3 checkbox", "Option3", "clickable", "checkBoxOption3"},
		{"Fill Enter Your Name", "Enter Your Name", "input", "name"},
		{"Click Alert button", "Alert", "clickable", "alertbtn"},
		{"Click Confirm button", "Confirm", "clickable", "confirmbtn"},
		{"Fill Hide/Show Example", "Hide/Show Example", "input", "displayed-text"},
		{"Click Hide button", "Hide", "clickable", "hide-textbox"},
		{"Click Top from hover menu", "Top", "clickable", "rs_top"},
		{"Click Reload from hover menu", "Reload", "clickable", "rs_reload"},
		{"Fill Wikipedia Search", "Wikipedia Search", "input", "Wikipedia1_wikipedia-search-input"},
		{"Click Wikipedia search button", "🔍", "clickable", "bs_wiki_btn"},
		{"Click New Browser Window", "New Browser Window", "clickable", "bs_new_window"},
		{"Fill Date Picker", "Date Picker", "input", "datepicker"},
		{"Double click Copy Text", "Copy Text", "clickable", "bs_dbl_click"},
		{"Fill bs_field1 Hello World", "Hello World", "input", "bs_field1"},
		{"Click Drag me", "Drag me", "clickable", "draggable"},
		{"Click Drop here", "Drop here", "clickable", "droppable"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rankFirstID(t, tc.query, "", tc.mode, elements)
			if got != tc.expectedID {
				t.Errorf("query=%q mode=%s → got %s, want %s", tc.query, tc.mode, got, tc.expectedID)
			}
		})
	}
}

func TestQAClassics_Select(t *testing.T) {
	elements := qaClassicsDOM()

	tests := []struct {
		name       string
		query      string
		expectedID string
	}{
		{"Select Dropdown Example", "Dropdown Example", "dropdown-class-example"},
		{"Select Speed", "Select Speed", "speed"},
		{"Select a file", "Select a file", "files"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rankFirstID(t, tc.query, "", "select", elements)
			if got != tc.expectedID {
				t.Errorf("query=%q mode=select → got %s, want %s", tc.query, got, tc.expectedID)
			}
		})
	}
}
