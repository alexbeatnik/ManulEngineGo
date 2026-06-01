package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// THE UNHOLY MESS & FINAL BOSS DOM SCORING TEST SUITE
//
// Port of ManulEngine test_10_mess.py — 100-element chaotic page with cookie
// banners, CAPTCHA, dark patterns, upsells, floating ads, rich-text editors,
// exotic inputs, tooltips, popovers, social icons, contenteditable, shadow DOM,
// traps, and deceptive patterns.
// Skipped: extract (28,81-86), verify (3,10,19,26,34,64,69,79,98),
//          optional/exp=None (27,93,94,99), execute_step (55,61).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

func messDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Cookie banner (x1-x10)
		el(1, "/html/body/button[1]", withTag("button"), withID("x1"), withText("ACCEPT ALL COOKIES")),
		el(2, "/html/body/button[2]", withTag("button"), withID("x2"), withClassName("dark-pattern-btn"), withText("Manage Preferences")),
		el(3, "/html/body/button[3]", withTag("button"), withID("x3"), withClassName("dark-pattern-btn"), withText("Reject All (Takes 5 minutes)")),
		el(4, "/html/body/input[1]", withTag("input"), withInputType("checkbox"), withID("x4"), withLabel("Strictly Necessary"), withDisabled()),
		el(5, "/html/body/input[2]", withTag("input"), withInputType("checkbox"), withID("x5"), withLabel("Marketing Cookies")),
		el(6, "/html/body/input[3]", withTag("input"), withInputType("checkbox"), withID("x6"), withLabel("Analytics Cookies")),
		el(7, "/html/body/button[4]", withTag("button"), withID("x7"), withDataQA("save-pref"), withText("Save Preferences")),
		el(8, "/html/body/a[1]", withTag("a"), withID("x8"), withText("Read our 50-page Privacy Policy")),
		el(9, "/html/body/button[5]", withTag("button"), withID("x9"), withAriaLabel("Close Cookie Banner"), withText("X")),
		el(10, "/html/body/div[1]", withTag("div"), withID("x10"), withRole("alert"), withText("Consent saved.")),

		// CAPTCHA (x11-x20)
		el(11, "/html/body/input[4]", withTag("input"), withInputType("checkbox"), withID("x11"), withLabel("I am human")),
		el(12, "/html/body/button[6]", withTag("button"), withID("x12"), withAriaLabel("Reload Captcha Image"), withText("🔄")),
		el(13, "/html/body/button[7]", withTag("button"), withID("x13"), withAriaLabel("Play Audio Challenge"), withText("🔊")),
		el(14, "/html/body/input[5]", withTag("input"), withInputType("text"), withID("x14"), withPlaceholder("Type the distorted text")),
		el(15, "/html/body/input[6]", withTag("input"), withInputType("text"), withID("x15_honeypot"), withClassName("honeypot"), withPlaceholder("Do not fill this")),
		el(16, "/html/body/button[8]", withTag("button"), withID("x16_honeypot"), withText("Admin bypass")),
		el(17, "/html/body/img[1]", withTag("img"), withID("x17"), withRole("button"), withAriaLabel("Traffic Light")),
		el(18, "/html/body/button[9]", withTag("button"), withID("x18"), withText("Verify Images")),
		el(19, "/html/body/div[2]", withTag("div"), withID("x19"), withText("Error: Please try again.")),
		el(20, "/html/body/button[10]", withTag("button"), withID("x20"), withText("Report CAPTCHA issue")),

		// Dark upsell patterns (x21-x30)
		el(21, "/html/body/button[11]", withTag("button"), withID("x21"), withText("Yes, upgrade me now!")),
		el(22, "/html/body/button[12]", withTag("button"), withID("x22"), withClassName("dark-pattern-btn"), withText("No thanks, I hate saving money")),
		el(23, "/html/body/input[7]", withTag("input"), withInputType("checkbox"), withID("x23"), withLabel("Subscribe to spam newsletter")),
		el(24, "/html/body/input[8]", withTag("input"), withInputType("checkbox"), withID("x24"), withLabel("sell your soul")),
		el(25, "/html/body/button[13]", withTag("button"), withID("x25"), withAriaLabel("Continue without upgrading"), withText("Continue")),
		el(26, "/html/body/div[3]", withTag("div"), withID("x26"), withHidden(), withText("Wait, don't leave! Take 50% off!")),
		el(27, "/html/body/button[14]", withTag("button"), withID("x27"), withHidden(), withText("Claim 50% Discount")),
		el(28, "/html/body/span[1]", withTag("span"), withID("x28"), withDataQA("sale-timer"), withText("Sale ends in 00:59")),
		el(29, "/html/body/button[15]", withTag("button"), withID("x29"), withText("Read Terms")),
		el(30, "/html/body/button[16]", withTag("button"), withID("x30"), withText("Close Upsell")),

		// Floating ad & chat (x31-x38)
		el(31, "/html/body/button[17]", withTag("button"), withID("x31"), withAriaLabel("Close Ad"), withText("✖")),
		el(32, "/html/body/a[2]", withTag("a"), withID("x32"), withText("Click here for free iPad")),
		el(33, "/html/body/button[18]", withTag("button"), withID("x33"), withAriaLabel("Chat Support"), withText("💬")),
		el(34, "/html/body/div[4]", withTag("div"), withID("x34"), withRole("dialog")),
		el(35, "/html/body/input[9]", withTag("input"), withInputType("text"), withID("x35"), withPlaceholder("Type message...")),
		el(36, "/html/body/button[19]", withTag("button"), withID("x36"), withAriaLabel("Send Message"), withText("➤")),
		el(37, "/html/body/button[20]", withTag("button"), withID("x37"), withText("Minimize Chat")),
		el(38, "/html/body/button[21]", withTag("button"), withID("x38"), withText("End Chat Session")),

		// Sticky nav (x39-x40)
		el(39, "/html/body/button[22]", withTag("button"), withID("x39"), withText("Sticky Header Menu")),
		el(40, "/html/body/input[10]", withTag("input"), withInputType("text"), withID("x40"), withPlaceholder("Global Search")),

		// Rich text editor (x41-x50)
		el(41, "/html/body/button[23]", withTag("button"), withID("x41"), withAriaLabel("Bold (Ctrl+B)"), withText("B")),
		el(42, "/html/body/button[24]", withTag("button"), withID("x42"), withAriaLabel("Italic (Ctrl+I)"), withText("I")),
		el(43, "/html/body/button[25]", withTag("button"), withID("x43"), withAriaLabel("Underline (Ctrl+U)"), withText("U")),
		el(44, "/html/body/button[26]", withTag("button"), withID("x44"), withAriaLabel("Insert Link"), withText("🔗")),
		el(45, "/html/body/button[27]", withTag("button"), withID("x45"), withAriaLabel("Insert Image"), withText("🖼️")),
		el(46, "/html/body/select[1]", withTag("select"), withID("x46"), withAriaLabel("Font Size"), withText("12pt")),
		el(47, "/html/body/button[28]", withTag("button"), withID("x47"), withText("View Source (HTML)")),
		el(48, "/html/body/div[5]", withTag("div"), withID("x48"), withAriaLabel("Rich Text Area"), withEditable(), withText("Start typing...")),
		el(49, "/html/body/button[29]", withTag("button"), withID("x49"), withText("Publish Post")),
		el(50, "/html/body/button[30]", withTag("button"), withID("x50"), withText("Save as Draft")),

		// Exotic inputs (x51-x60)
		el(51, "/html/body/input[11]", withTag("input"), withInputType("color"), withID("x51"), withLabel("Pick a color:"), withValue("#ff0000")),
		el(52, "/html/body/input[12]", withTag("input"), withInputType("time"), withID("x52"), withLabel("Alarm Time:")),
		el(53, "/html/body/input[13]", withTag("input"), withInputType("month"), withID("x53"), withLabel("Expiration Month:")),
		el(54, "/html/body/input[14]", withTag("input"), withInputType("range"), withID("x54"), withLabel("Intensity:")),
		el(55, "/html/body/input[15]", withTag("input"), withInputType("password"), withID("x55"), withLabel("Secret Key:"), withValue("hidden_key")),
		el(56, "/html/body/button[31]", withTag("button"), withID("x56"), withText("Unlock Key")),
		el(57, "/html/body/input[16]", withTag("input"), withInputType("file"), withID("x57"), withAriaLabel("Avatar Upload")),
		el(58, "/html/body/button[32]", withTag("button"), withID("x58"), withText("Clear File")),
		el(59, "/html/body/input[17]", withTag("input"), withInputType("range"), withID("x59"), withAriaLabel("Volume Knob"), withLabel("Volume Knob"), withValue("75")),
		el(60, "/html/body/button[33]", withTag("button"), withID("x60"), withText("Reset Defaults")),

		// Tooltips & popovers (x61-x68)
		el(61, "/html/body/span[2]", withTag("span"), withID("x61"), withText("Hover me")),
		el(62, "/html/body/button[34]", withTag("button"), withID("x62"), withText("Submit")),
		el(63, "/html/body/div[6]", withTag("div"), withID("x63"), withRole("button"), withText("Click for more info")),
		el(64, "/html/body/div[7]", withTag("div"), withID("x64"), withHidden(), withText("Here is the secret info.")),
		el(65, "/html/body/button[35]", withTag("button"), withID("x65"), withText("Close Popover")),
		el(66, "/html/body/span[3]", withTag("span"), withID("x66"), withClassName("info-icon"), withAriaLabel("Help"), withText("?")),
		el(67, "/html/body/input[18]", withTag("input"), withInputType("text"), withID("x67"), withPlaceholder("Hover to reveal")),
		el(68, "/html/body/button[36]", withTag("button"), withID("x68"), withClassName("ghost-btn"), withText("Ghost Action")),

		// Dynamic content (x69-x70)
		el(69, "/html/body/div[8]", withTag("div"), withID("x69"), withText("Dynamic Text: Loading...")),
		el(70, "/html/body/button[37]", withTag("button"), withID("x70"), withText("Load Data")),

		// Social SVG icons (x71-x73)
		el(71, "/html/body/a[3]", withTag("a"), withID("x71"), withAriaLabel("Facebook")),
		el(72, "/html/body/a[4]", withTag("a"), withID("x72"), withAriaLabel("Twitter")),
		el(73, "/html/body/a[5]", withTag("a"), withID("x73"), withAriaLabel("LinkedIn")),

		// Tricky aria vs visible text (x74-x78)
		el(74, "/html/body/button[38]", withTag("button"), withID("x74"), withAriaLabel("Actual Action"), withText("Wrong Visible Text")),
		el(75, "/html/body/div[9]", withTag("div"), withID("x75"), withRole("button"), withDisabled(), withText("Looks clickable but isn't")),
		el(76, "/html/body/button[39]", withTag("button"), withID("x76"), withText("Go to Target")),
		el(77, "/html/body/div[10]", withTag("div"), withID("x77"), withRole("textbox"), withEditable(), withText("Fake Input")),
		el(78, "/html/body/button[40]", withTag("button"), withID("x78"), withText("Clear Fake Input")),

		// Alert dialog (x79-x80)
		el(79, "/html/body/div[11]", withTag("div"), withID("x79"), withRole("alertdialog")),
		el(80, "/html/body/button[41]", withTag("button"), withID("x80"), withText("Dismiss Warning")),

		// Data extraction elements (x81-x90)
		el(81, "/html/body/strong[1]", withTag("strong"), withID("x81"), withText("Manul")),
		el(82, "/html/body/span[4]", withTag("span"), withID("x82"), withText("3 years")),
		el(83, "/html/body/i[1]", withTag("i"), withID("x83"), withText("QA Automation")),
		el(84, "/html/body/span[5]", withTag("span"), withID("x84"), withRole("cell"), withText("Operational")),
		el(85, "/html/body/span[6]", withTag("span"), withID("x85"), withRole("cell"), withText("42ms")),
		el(86, "/html/body/span[7]", withTag("span"), withID("x86"), withText("Deep Text")),
		el(87, "/html/body/button[42]", withTag("button"), withID("x87"), withText("Extract Everything")),
		el(88, "/html/body/input[19]", withTag("input"), withInputType("text"), withID("x88"), withValue("Pre-filled data")),
		el(89, "/html/body/textarea[1]", withTag("textarea"), withID("x89"), withText("Pre-filled textarea"), withEditable()),
		el(90, "/html/body/button[43]", withTag("button"), withID("x90"), withText("Wipe Data")),

		// Shadow DOM (x91-x92)
		el(91, "/html/body/button[44]", withTag("button"), withID("x91"), withText("Shadow Strike")),
		el(92, "/html/body/input[20]", withTag("input"), withInputType("text"), withID("x92"), withPlaceholder("Shadow Input")),

		// Traps & deceptive patterns (x93-x97)
		el(93, "/html/body/button[45]", withTag("button"), withID("x93"), withHidden(), withText("Zero Pixel Trap")),
		el(94, "/html/body/div[12]", withTag("div"), withID("x94"), withRole("button"), withHidden(), withText("Clipped Trap")),
		el(95, "/html/body/span[8]", withTag("span"), withID("x95"), withRole("checkbox"), withText("Custom Span Checkbox")),
		el(96, "/html/body/button[46]", withTag("button"), withID("x96"), withClassName("generic"), withText("Submit")),
		el(97, "/html/body/button[47]", withTag("button"), withID("x97"), withClassName("generic"), withDataQA("final"), withText("Submit Final")),

		// Finish (x98-x100)
		el(98, "/html/body/div[13]", withTag("div"), withID("x98"), withText("Test Complete")),
		el(99, "/html/body/button[48]", withTag("button"), withID("x99"), withHidden(), withText("Hidden Hover Button")),
		el(100, "/html/body/button[49]", withTag("button"), withID("x100"), withClassName("celebrate-btn"), withText("🎉 FINISH LAB 🎉")),
	}
}

func TestMess(t *testing.T) {
	elements := messDOM()

	tests := []struct {
		name       string
		query      string
		mode       string
		expectedID string
	}{
		// Cookie banner
		{"Click ACCEPT ALL COOKIES", "ACCEPT ALL COOKIES", "clickable", "x1"},
		{"Click Manage Preferences", "Manage Preferences", "clickable", "x2"},
		{"Click Reject All", "Reject All", "clickable", "x3"},
		{"Check Marketing Cookies", "Marketing Cookies", "clickable", "x5"},
		{"Check Analytics Cookies", "Analytics Cookies", "clickable", "x6"},
		{"Click Save Preferences", "Save Preferences", "clickable", "x7"},
		{"Click Privacy Policy link", "Privacy Policy", "clickable", "x8"},
		{"Click Close Cookie Banner", "Close Cookie Banner", "clickable", "x9"},
		// CAPTCHA
		{"Check I am human", "I am human", "clickable", "x11"},
		{"Click Reload Captcha Image", "Reload Captcha Image", "clickable", "x12"},
		{"Click Play Audio Challenge", "Play Audio Challenge", "clickable", "x13"},
		{"Fill Type the distorted text", "Type the distorted text", "input", "x14"},
		{"Fill Do not fill this", "Do not fill this", "input", "x15_honeypot"},
		{"Click Admin bypass", "Admin bypass", "clickable", "x16_honeypot"},
		{"Click Traffic Light image", "Traffic Light", "clickable", "x17"},
		{"Click Verify Images", "Verify Images", "clickable", "x18"},
		{"Click Report CAPTCHA issue", "Report CAPTCHA issue", "clickable", "x20"},
		// Dark upsell patterns
		{"Click Yes upgrade me now", "Yes, upgrade me now!", "clickable", "x21"},
		{"Click I hate saving money", "I hate saving money", "clickable", "x22"},
		{"Uncheck Subscribe to spam newsletter", "Subscribe to spam newsletter", "clickable", "x23"},
		{"Uncheck sell your soul", "sell your soul", "clickable", "x24"},
		{"Click Continue without upgrading", "Continue without upgrading", "clickable", "x25"},
		{"Click Read Terms", "Read Terms", "clickable", "x29"},
		{"Click Close Upsell", "Close Upsell", "clickable", "x30"},
		// Floating ad & chat
		{"Click Close Ad", "Close Ad", "clickable", "x31"},
		{"Click free iPad link", "free iPad", "clickable", "x32"},
		{"Click Chat Support", "Chat Support", "clickable", "x33"},
		{"Fill Type message", "Type message...", "input", "x35"},
		{"Click Send Message", "Send Message", "clickable", "x36"},
		{"Click Minimize Chat", "Minimize Chat", "clickable", "x37"},
		{"Click End Chat Session", "End Chat Session", "clickable", "x38"},
		// Sticky nav
		{"Click Sticky Header Menu", "Sticky Header Menu", "clickable", "x39"},
		{"Fill Global Search", "Global Search", "input", "x40"},
		// Rich text editor
		{"Click Bold", "Bold (Ctrl+B)", "clickable", "x41"},
		{"Click Italic", "Italic (Ctrl+I)", "clickable", "x42"},
		{"Click Underline", "Underline", "clickable", "x43"},
		{"Click Insert Link", "Insert Link", "clickable", "x44"},
		{"Click Insert Image", "Insert Image", "clickable", "x45"},
		{"Click View Source HTML", "View Source (HTML)", "clickable", "x47"},
		{"Fill Rich Text Area", "Rich Text Area", "input", "x48"},
		{"Click Publish Post", "Publish Post", "clickable", "x49"},
		{"Click Save as Draft", "Save as Draft", "clickable", "x50"},
		// Exotic inputs
		{"Fill Pick a color", "Pick a color", "input", "x51"},
		{"Fill Alarm Time", "Alarm Time", "input", "x52"},
		{"Fill Expiration Month", "Expiration Month", "input", "x53"},
		{"Fill Intensity", "Intensity", "input", "x54"},
		{"Click Unlock Key", "Unlock Key", "clickable", "x56"},
		{"Click Avatar Upload", "Avatar Upload", "clickable", "x57"},
		{"Click Clear File", "Clear File", "clickable", "x58"},
		{"Fill Volume Knob", "Volume Knob", "input", "x59"},
		{"Click Reset Defaults", "Reset Defaults", "clickable", "x60"},
		// Tooltips & popovers
		{"Click Submit button (x62)", "Submit", "clickable", "x62"},
		{"Click Click for more info", "Click for more info", "clickable", "x63"},
		{"Click Close Popover", "Close Popover", "clickable", "x65"},
		{"Click Help icon", "Help", "clickable", "x66"},
		{"Fill Hover to reveal", "Hover to reveal", "input", "x67"},
		{"Click Ghost Action", "Ghost Action", "clickable", "x68"},
		// Dynamic content
		{"Click Load Data", "Load Data", "clickable", "x70"},
		// Social SVG icons
		{"Click Facebook icon", "Facebook", "clickable", "x71"},
		{"Click Twitter icon", "Twitter", "clickable", "x72"},
		{"Click LinkedIn icon", "LinkedIn", "clickable", "x73"},
		// Tricky aria vs visible text
		{"Click Actual Action", "Actual Action", "clickable", "x74"},
		// x75 is disabled (aria-disabled=true) — scorer correctly skips it in clickable mode
		{"Click Go to Target", "Go to Target", "clickable", "x76"},
		{"Fill Fake Input", "Fake Input", "input", "x77"},
		{"Click Clear Fake Input", "Clear Fake Input", "clickable", "x78"},
		// Alert dialog
		{"Click Dismiss Warning", "Dismiss Warning", "clickable", "x80"},
		// Data extraction & pre-filled
		{"Click Extract Everything", "Extract Everything", "clickable", "x87"},
		{"Fill Pre-filled data", "Pre-filled data", "input", "x88"},
		{"Fill Pre-filled textarea", "Pre-filled textarea", "input", "x89"},
		{"Click Wipe Data", "Wipe Data", "clickable", "x90"},
		// Shadow DOM
		{"Click Shadow Strike", "Shadow Strike", "clickable", "x91"},
		{"Fill Shadow Input", "Shadow Input", "input", "x92"},
		// Custom checkbox
		{"Check Custom Span Checkbox", "Custom Span Checkbox", "clickable", "x95"},
		// Ambiguous submits
		{"Click Submit Final", "Submit Final", "clickable", "x97"},
		// Finish
		{"Click FINISH LAB", "FINISH LAB", "clickable", "x100"},
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

func TestMess_Select(t *testing.T) {
	elements := messDOM()

	tests := []struct {
		name       string
		query      string
		expectedID string
	}{
		{"Select Font Size", "Font Size", "x46"},
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
