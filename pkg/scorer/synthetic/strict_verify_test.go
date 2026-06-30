package synthetic

import (
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

func strictDOM() []dom.ElementSnapshot {
	return []dom.ElementSnapshot{
		el(1, "/body/button[1]", withID("strict_save_btn"), withTag("button"), withText("Save me")),
		el(2, "/body/input[1]", withID("strict_login_field"), withTag("input"), withPlaceholder("Login/Email"), withLabel("Login")),
		el(3, "/body/input[2]", withID("strict_email_value"), withTag("input"), withValue("captain@manul.com"), withLabel("Profile Email")),
	}
}

func TestStrictVerify(t *testing.T) {
	elements := strictDOM()

	tests := []struct {
		name       string
		query      string
		mode       string
		expectedID string
	}{
		{"TextMatch", "Save me", "clickable", "strict_save_btn"},
		{"PlaceholderMatch", "Login/Email", "input", "strict_login_field"},
		{"ValueMatch", "captain@manul.com", "input", "strict_email_value"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rankFirstID(t, tc.query, "", tc.mode, elements)
			if got != tc.expectedID {
				t.Errorf("%s: got %s, want %s", tc.name, got, tc.expectedID)
			}
		})
	}
}
