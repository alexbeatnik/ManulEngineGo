package browser

import "testing"

func TestNormalizeKeyName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Enter", "Enter"},
		{"enter", "Enter"},
		{"RETURN", "Enter"},
		{"esc", "Escape"},
		{"Escape", "Escape"},
		{"tab", "Tab"},
		{"space", " "},
		{"Spacebar", " "},
		{"down", "ArrowDown"},
		{"ArrowUp", "ArrowUp"},
		{"del", "Delete"},
		{" Enter ", "Enter"},
		// Unknown names pass through (trimmed).
		{"F5", "F5"},
		{"a", "a"},
	}
	for _, c := range cases {
		if got := normalizeKeyName(c.in); got != c.want {
			t.Errorf("normalizeKeyName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestKeyText_CharProducingKeysCarryText(t *testing.T) {
	// Enter MUST produce "\r": a CDP keyDown without text is a rawKeyDown —
	// no keypress fires and forms are never submitted.
	if got := keyText("Enter"); got != "\r" {
		t.Errorf("keyText(Enter) = %q, want \\r", got)
	}
	if got := keyText(" "); got != " " {
		t.Errorf("keyText(space) = %q, want a single space", got)
	}
	if got := keyText("a"); got != "a" {
		t.Errorf("keyText(a) = %q, want %q", got, "a")
	}
	// Non-printing keys must NOT carry text.
	for _, key := range []string{"Escape", "Tab", "Backspace", "ArrowDown", "F5"} {
		if got := keyText(key); got != "" {
			t.Errorf("keyText(%q) = %q, want empty", key, got)
		}
	}
}

func TestKeyToCode(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Enter", "Enter"},
		{"Escape", "Escape"},
		{" ", "Space"},
		{"ArrowDown", "ArrowDown"},
		{"a", "KeyA"},
		{"Z", "KeyZ"},
		{"7", "Digit7"},
		{"F5", ""}, // unmapped is tolerated
	}
	for _, c := range cases {
		if got := keyToCode(c.in); got != c.want {
			t.Errorf("keyToCode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestKeyToVirtualCode(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"Enter", 13},
		{"Escape", 27},
		{"Tab", 9},
		{" ", 32},
		{"a", 65},
		{"A", 65},
		{"5", 53},
		{"NoSuchKey", 0},
	}
	for _, c := range cases {
		if got := keyToVirtualCode(c.in); got != c.want {
			t.Errorf("keyToVirtualCode(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
