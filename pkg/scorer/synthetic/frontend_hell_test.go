package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// FRONTEND HELL & ANTI-PATTERN GAUNTLET DOM SCORING TEST SUITE
//
// Port of ManulEngine test_17_frontend_hell.py — ~39-element page with split
// text, hidden/fake duplicates, CSS uppercase, deeply nested text,
// contenteditable, whitespace-padded placeholders, web components, aria tricks.
// All 30 tests are clickable/input mode (no select).
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

func frontendHellDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Split text "Confirm Action"
		el(1, "/html/body/div[1]", withTag("div"), withRole("button"), withID("t1"), withText("Confirm Action")),
		// Fake hidden Settings vs real Settings
		el(2, "/html/body/button[1]", withTag("button"), withID("fake2"), withHidden(), withText("Settings")),
		el(3, "/html/body/button[2]", withTag("button"), withID("t2"), withText("Settings")),
		// Notifications (SVG icon + aria)
		el(4, "/html/body/button[3]", withTag("button"), withID("t3"), withAriaLabel("Notifications")),
		// Delivery Address (sibling label)
		el(5, "/html/body/input[1]", withTag("input"), withInputType("text"), withID("t4"), withLabel("Delivery Address")),
		// CSS uppercase "proceed" → "PROCEED"
		el(6, "/html/body/button[4]", withTag("button"), withID("t5"), withClassName("uppercase"), withText("proceed")),
		// Promo Code
		el(7, "/html/body/input[2]", withTag("input"), withInputType("text"), withID("t6"), withPlaceholder("Promo Code")),
		// Credit Card Number
		el(8, "/html/body/input[3]", withTag("input"), withInputType("text"), withID("t7"), withAriaLabel("Credit Card Number")),
		// Pay Now submit
		el(9, "/html/body/input[4]", withTag("input"), withInputType("submit"), withID("t8"), withValue("Pay Now")),
		// Finalize Order (deeply nested em)
		el(10, "/html/body/div[2]", withTag("div"), withRole("button"), withID("t9"), withText("Finalize Order")),
		// Agree to Terms checkbox
		el(11, "/html/body/input[5]", withTag("input"), withInputType("checkbox"), withID("t10"), withLabel("Agree to Terms")),
		// Download Invoice (title-based)
		el(12, "/html/body/div[3]", withTag("div"), withID("t11"), withAriaLabel("Download Invoice"), withText("⬇️ PDF")),
		// First Name (whitespace-padded placeholder)
		el(13, "/html/body/input[6]", withTag("input"), withInputType("text"), withID("t12"), withPlaceholder("First Name")),
		// Fake "Update Profile Info" vs real "Update Profile"
		el(14, "/html/body/button[5]", withTag("button"), withID("fake13"), withText("Update Profile Info")),
		el(15, "/html/body/button[6]", withTag("button"), withID("t13"), withText("Update Profile")),
		// Fake hidden Logout vs real
		el(16, "/html/body/button[7]", withTag("button"), withID("fake14"), withHidden(), withText("Logout")),
		el(17, "/html/body/button[8]", withTag("button"), withID("t14"), withText("Logout")),
		// Fake hidden Delete Account vs real
		el(18, "/html/body/button[9]", withTag("button"), withID("fake15"), withHidden(), withText("Delete Account")),
		el(19, "/html/body/button[10]", withTag("button"), withID("t15"), withText("Delete Account")),
		// Search Items (with SVG icon)
		el(20, "/html/body/button[11]", withTag("button"), withID("t16"), withText("Search Items")),
		// Upload Avatar (br inside)
		el(21, "/html/body/button[12]", withTag("button"), withID("t17"), withText("Upload Avatar")),
		// Date of Birth (label from preceding bold text)
		el(22, "/html/body/input[7]", withTag("input"), withInputType("date"), withID("t18"), withLabel("Date of Birth")),
		// Biography (contenteditable)
		el(23, "/html/body/div[4]", withTag("div"), withRole("textbox"), withID("t19"), withEditable(), withAriaLabel("Biography")),
		// Fake offscreen Subscribe vs real
		el(24, "/html/body/button[13]", withTag("button"), withID("fake20"), withHidden(), withText("Subscribe")),
		el(25, "/html/body/button[14]", withTag("button"), withID("t20"), withText("Subscribe")),
		// Fake aria-hidden Connect Wallet vs real
		el(26, "/html/body/button[15]", withTag("button"), withID("fake21"), withHidden(), withText("Connect Wallet")),
		el(27, "/html/body/button[16]", withTag("button"), withID("t21"), withText("Connect Wallet")),
		// Security pin
		el(28, "/html/body/input[8]", withTag("input"), withInputType("password"), withID("t22"), withNameAttr("security_pin")),
		// Go to Checkout link
		el(29, "/html/body/a[1]", withTag("a"), withID("t23"), withClassName("btn-primary"), withText("Go to Checkout")),
		// Fake zero-size Refresh vs real
		el(30, "/html/body/button[17]", withTag("button"), withID("fake24"), withHidden(), withText("Refresh Page")),
		el(31, "/html/body/button[18]", withTag("button"), withID("t24"), withText("Refresh Page")),
		// Scan QR Code image
		el(32, "/html/body/img[1]", withTag("img"), withID("t25"), withAriaLabel("Scan QR Code")),
		// Secret Token (label via aria-labelledby)
		el(33, "/html/body/span[1]", withTag("span"), withID("lbl26"), withText("Secret Token")),
		el(34, "/html/body/input[9]", withTag("input"), withInputType("text"), withID("t26"), withLabel("Secret Token")),
		// Dark Theme menuitem
		el(35, "/html/body/div[5]", withTag("div"), withRole("menuitem"), withID("t27"), withText("Dark Theme")),
		// Send Message (extra whitespace/nbsp)
		el(36, "/html/body/button[19]", withTag("button"), withID("t28"), withText("Send Message")),
		// Fake "Phone Number (Optional)" vs real "Phone Number"
		el(37, "/html/body/input[10]", withTag("input"), withInputType("text"), withID("fake29"), withPlaceholder("Phone Number (Optional)")),
		el(38, "/html/body/input[11]", withTag("input"), withInputType("text"), withID("t29"), withPlaceholder("Phone Number")),
		// Web component
		el(39, "/html/body/custom-btn[1]", withTag("custom-btn"), withID("t30"), withText("Launch Rocket")),
	}
}

func TestFrontendHell(t *testing.T) {
	elements := frontendHellDOM()

	tests := []struct {
		name       string
		query      string
		mode       string
		expectedID string
	}{
		{"Click Confirm Action", "Confirm Action", "clickable", "t1"},
		{"Click Settings", "Settings", "clickable", "t2"},
		{"Click Notifications", "Notifications", "clickable", "t3"},
		{"Fill Delivery Address", "Delivery Address", "input", "t4"},
		{"Click Proceed", "Proceed", "clickable", "t5"},
		{"Fill Promo Code", "Promo Code", "input", "t6"},
		{"Fill Credit Card Number", "Credit Card Number", "input", "t7"},
		{"Click Pay Now", "Pay Now", "clickable", "t8"},
		{"Click Finalize Order", "Finalize Order", "clickable", "t9"},
		{"Check Agree to Terms", "Agree to Terms", "clickable", "t10"},
		{"Click Download Invoice", "Download Invoice", "clickable", "t11"},
		{"Fill First Name", "First Name", "input", "t12"},
		{"Click Update Profile", "Update Profile", "clickable", "t13"},
		{"Click Logout", "Logout", "clickable", "t14"},
		{"Click Delete Account", "Delete Account", "clickable", "t15"},
		{"Click Search Items", "Search Items", "clickable", "t16"},
		{"Click Upload Avatar", "Upload Avatar", "clickable", "t17"},
		{"Fill Date of Birth", "Date of Birth", "input", "t18"},
		{"Fill Biography", "Biography", "input", "t19"},
		{"Click Subscribe", "Subscribe", "clickable", "t20"},
		{"Click Connect Wallet", "Connect Wallet", "clickable", "t21"},
		{"Fill security_pin", "security_pin", "input", "t22"},
		{"Click Go to Checkout", "Go to Checkout", "clickable", "t23"},
		{"Click Refresh Page", "Refresh Page", "clickable", "t24"},
		{"Click Scan QR Code", "Scan QR Code", "clickable", "t25"},
		{"Fill Secret Token", "Secret Token", "input", "t26"},
		{"Click Dark Theme", "Dark Theme", "clickable", "t27"},
		{"Click Send Message", "Send Message", "clickable", "t28"},
		{"Fill Phone Number", "Phone Number", "input", "t29"},
		{"Click Launch Rocket", "Launch Rocket", "clickable", "t30"},
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
