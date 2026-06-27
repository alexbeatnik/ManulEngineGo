package browser

import "testing"

// TestChannelBinaries verifies the channel→binary map matches ManulEngine's
// _CHANNEL_BINARIES so MANUL_CHANNEL resolves to the same Chrome across runtimes.
func TestChannelBinaries(t *testing.T) {
	want := map[string][]string{
		"chrome":      {"google-chrome-stable", "google-chrome"},
		"chrome-beta": {"google-chrome-beta"},
		"chrome-dev":  {"google-chrome-unstable"},
		"chromium":    {"chromium", "chromium-browser"},
		"msedge":      {"microsoft-edge-stable", "microsoft-edge"},
	}
	for ch, bins := range want {
		got, ok := channelBinaries[ch]
		if !ok {
			t.Errorf("channel %q missing from channelBinaries", ch)
			continue
		}
		if len(got) != len(bins) {
			t.Errorf("channel %q: got %v, want %v", ch, got, bins)
			continue
		}
		for i := range bins {
			if got[i] != bins[i] {
				t.Errorf("channel %q[%d]: got %q, want %q", ch, i, got[i], bins[i])
			}
		}
	}
}
