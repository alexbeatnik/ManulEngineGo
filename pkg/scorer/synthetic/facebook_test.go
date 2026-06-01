package synthetic

// ─────────────────────────────────────────────────────────────────────────────
// FACEBOOK COMET UI DOM SCORING TEST SUITE
//
// Port of ManulEngine test_15_facebook_final_boss.py — 12-element Facebook
// login & registration page with comet-style dynamic IDs.
// Validates: email/password login, registration form with first/last name,
// birthday comboboxes, gender selector, mobile/email + password fields, submit.
// Note: Original uses prefix-matching on IDs; here we use exact IDs.
// ─────────────────────────────────────────────────────────────────────────────

import (
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/dom"
)

func facebookDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		// Login
		el(1, "/html/body/input[1]", withTag("input"), withInputType("text"), withID("_R_1h6kqsqppb6amH1_"), withNameAttr("email"), withLabel("Email or mobile number")),
		el(2, "/html/body/input[2]", withTag("input"), withInputType("password"), withID("_R_1hmkqsqppb6amH1_"), withNameAttr("pass"), withLabel("Password")),
		el(3, "/html/body/div[1]", withTag("div"), withRole("button"), withID("login_btn_click"), withAriaLabel("Log In"), withText("Log in")),
		// Registration
		el(4, "/html/body/input[3]", withTag("input"), withInputType("text"), withID("_R_1cl2p4jikacppb6amH1_"), withLabel("First name")),
		el(5, "/html/body/input[4]", withTag("input"), withInputType("text"), withID("_R_1kl2p4jikacppb6amH1_"), withLabel("Last name")),
		el(6, "/html/body/div[2]", withTag("div"), withID("_r_3_"), withRole("combobox"), withAriaLabel("Select Month"), withText("Month")),
		el(7, "/html/body/div[3]", withTag("div"), withID("_r_9_"), withRole("combobox"), withAriaLabel("Select Day"), withText("Day")),
		el(8, "/html/body/div[4]", withTag("div"), withID("_r_f_"), withRole("combobox"), withAriaLabel("Select Year"), withText("Year")),
		el(9, "/html/body/div[5]", withTag("div"), withID("_R_mad6p4jikacppb6amH2_"), withRole("combobox"), withAriaLabel("Select your gender"), withText("Select your gender")),
		el(10, "/html/body/input[5]", withTag("input"), withInputType("text"), withID("_R_6ad8p4jikacppb6amH1_"), withLabel("Mobile number or email")),
		el(11, "/html/body/input[6]", withTag("input"), withInputType("password"), withID("_R_clap4jikacppb6amH1_"), withLabel("Password")),
		el(12, "/html/body/div[6]", withTag("div"), withRole("button"), withID("reg_submit_click"), withText("Submit")),
	}
}

func TestFacebook(t *testing.T) {
	elements := facebookDOM()

	tests := []struct {
		name       string
		query      string
		mode       string
		expectedID string
	}{
		{"Fill Email or mobile number", "Email or mobile number", "input", "_R_1h6kqsqppb6amH1_"},
		{"Click Log in", "Log in", "clickable", "login_btn_click"},
		{"Fill First name", "First name", "input", "_R_1cl2p4jikacppb6amH1_"},
		{"Fill Last name", "Last name", "input", "_R_1kl2p4jikacppb6amH1_"},
		{"Fill Mobile number or email", "Mobile number or email", "input", "_R_6ad8p4jikacppb6amH1_"},
		{"Click Submit", "Submit", "clickable", "reg_submit_click"},
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

func TestFacebook_Select(t *testing.T) {
	elements := facebookDOM()

	tests := []struct {
		name       string
		query      string
		expectedID string
	}{
		{"Select Month", "Month", "_r_3_"},
		{"Select Day", "Day", "_r_9_"},
		{"Select Year", "Year", "_r_f_"},
		{"Select your gender", "Select your gender", "_R_mad6p4jikacppb6amH2_"},
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
