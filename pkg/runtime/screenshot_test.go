package runtime

import "testing"

// TestScreenshotSlug verifies the SCREENSHOT label → filesystem-safe base name
// (mirrors ManulEngine's extract_screenshot_name).
func TestScreenshotSlug(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"after login", "after_login"},
		{"cart.png", "cart"},
		{"  Step 2: checkout!  ", "Step_2_checkout"},
		{"already_ok-1", "already_ok-1"},
	}
	for _, c := range cases {
		if got := screenshotSlug(c.in); got != c.want {
			t.Errorf("screenshotSlug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
