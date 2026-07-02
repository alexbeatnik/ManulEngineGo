// Package browser — Chrome process lifecycle management.
package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ChromeProcess manages a Chrome browser process spawned for automation.
type ChromeProcess struct {
	cmd            *exec.Cmd
	port           int
	userDataDir    string
	ownsDataDir    bool // true when we created the dir and should clean it up
}

// ChromeOptions configures the Chrome process to spawn.
type ChromeOptions struct {
	// Port for Chrome's remote debugging protocol. Default: 9222.
	Port int
	// UserDataDir is the Chrome profile directory.
	// If empty, a unique temp directory is created per run and cleaned up on Close.
	UserDataDir string
	// DisableGPU disables GPU acceleration. Default: true.
	DisableGPU bool
	// Headless runs Chrome without a visible window.
	Headless bool
	// ExecutablePath overrides the Chrome binary location.
	ExecutablePath string
	// Channel selects a system Chrome/Chromium binary by name (chrome,
	// chrome-beta, chrome-dev, chromium, msedge). Empty = platform defaults.
	// Mirrors ManulEngine's `channel` / MANUL_CHANNEL.
	Channel string
}

// channelBinaries maps a channel name to the concrete binaries to probe (in
// order). Mirrors ManulEngine's _CHANNEL_BINARIES.
var channelBinaries = map[string][]string{
	"chrome":      {"google-chrome-stable", "google-chrome"},
	"chrome-beta": {"google-chrome-beta"},
	"chrome-dev":  {"google-chrome-unstable"},
	"chromium":    {"chromium", "chromium-browser"},
	"msedge":      {"microsoft-edge-stable", "microsoft-edge"},
}

// DefaultChromeOptions returns sensible defaults for automation.
// UserDataDir is left empty so LaunchChrome creates a unique temp directory.
func DefaultChromeOptions() ChromeOptions {
	return ChromeOptions{
		Port:       9222,
		DisableGPU: true,
		Headless:   false,
	}
}

// LaunchChrome starts a Chrome process with remote debugging enabled.
// It blocks until Chrome's CDP endpoint is reachable (or context expires).
// If opts.UserDataDir is empty, a unique temp directory is created and owned
// by the returned ChromeProcess (removed when Close is called).
func LaunchChrome(ctx context.Context, opts ChromeOptions) (*ChromeProcess, error) {
	chromePath := opts.ExecutablePath
	if chromePath == "" {
		var err error
		chromePath, err = findChrome(opts.Channel)
		if err != nil {
			return nil, err
		}
	}

	ownsDir := false
	if opts.UserDataDir == "" {
		dir, err := os.MkdirTemp("", "manulengine-chrome-*")
		if err != nil {
			return nil, fmt.Errorf("create chrome temp dir: %w", err)
		}
		opts.UserDataDir = dir
		ownsDir = true
	}

	// Write Chrome preferences to disable password manager at profile level.
	if err := writeAutomationPrefs(opts.UserDataDir); err != nil {
		if ownsDir {
			_ = os.RemoveAll(opts.UserDataDir)
		}
		return nil, fmt.Errorf("write chrome prefs: %w", err)
	}

	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", opts.Port),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-client-side-phishing-detection",
		"--disable-default-apps",
		"--disable-extensions",
		"--disable-hang-monitor",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-sync",
		"--disable-translate",
		"--disable-search-engine-choice-screen",
		"--disable-features=PasswordLeakDetection,PasswordManagerOnboarding,PasswordCheck,ChromePasswordManagerUI,CredentialManager,AutofillServerCommunication,IdentityStatusDialog,GlobalMediaControls,MediaRouter,Translate,OptimizationHints",
		"--no-service-autorun",
		"--password-store=basic",
		"--disable-save-password-bubble",
		"--disable-component-update",
		"--disable-infobars",
		fmt.Sprintf("--user-data-dir=%s", opts.UserDataDir),
	}
	if opts.DisableGPU {
		args = append(args, "--disable-gpu")
	}
	if opts.Headless {
		args = append(args, "--headless=new")
	}

	// Chrome must outlive ctx: the caller's context typically scopes one task
	// (a single prompt/pipeline turn in an embedding app), while the browser is
	// a long-lived resource shared across tasks. exec.CommandContext would kill
	// Chrome the moment that context is cancelled — the user watched the browser
	// close seconds after every action that had launched it. ctx still bounds
	// the startup phase (waitForCDP below); after that only Close kills Chrome.
	cmd := exec.Command(chromePath, args...)
	// Detach stdout/stderr — Chrome is noisy by default.
	cmd.Stdout = nil
	cmd.Stderr = nil
	// Inherit environment (required for DISPLAY on Linux).
	cmd.Env = os.Environ()
	// Platform-specific process group setup (implemented in chrome_unix.go / chrome_windows.go).
	setProcGroup(cmd)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start chrome: %w", err)
	}

	cp := &ChromeProcess{
		cmd:         cmd,
		port:        opts.Port,
		userDataDir: opts.UserDataDir,
		ownsDataDir: ownsDir,
	}

	// Wait for CDP endpoint to become reachable.
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", opts.Port)
	if err := waitForCDP(ctx, endpoint, 15*time.Second); err != nil {
		// Chrome started but CDP never became reachable — kill it.
		_ = cp.Close()
		return nil, fmt.Errorf("chrome started but CDP not reachable at %s: %w", endpoint, err)
	}

	return cp, nil
}

// Endpoint returns the HTTP CDP endpoint URL.
func (cp *ChromeProcess) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d", cp.port)
}

// findChrome searches for a Chrome binary in common locations.
func findChrome(channel string) (string, error) {
	var candidates []string
	// Channel-selected binaries take precedence over platform defaults.
	if channel != "" {
		if bins, ok := channelBinaries[strings.ToLower(channel)]; ok {
			candidates = append(candidates, bins...)
		} else {
			candidates = append(candidates, channel) // treat as a bare binary name
		}
	}
	switch runtime.GOOS {
	case "linux":
		candidates = append(candidates,
			"google-chrome-stable",
			"google-chrome",
			"chromium-browser",
			"chromium",
		)
	case "darwin":
		candidates = append(candidates,
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"google-chrome",
			"chromium",
		)
	case "windows":
		candidates = append(candidates,
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		)
	default:
		candidates = append(candidates, "google-chrome", "chromium")
	}

	for _, c := range candidates {
		// Absolute path — check existence directly.
		if strings.Contains(c, "/") || strings.Contains(c, `\`) {
			if _, err := os.Stat(c); err == nil {
				return c, nil
			}
			continue
		}
		// Short name — look up in PATH.
		if p, err := exec.LookPath(c); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("chrome not found; install Google Chrome or set it in PATH")
}

// waitForCDP polls the CDP /json endpoint until it responds or timeout expires.
// Uses context.WithTimeout so both the outer context deadline and the
// internal timeout are respected — whichever fires first wins.
func waitForCDP(ctx context.Context, endpoint string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := endpoint + "/json"
	client := &http.Client{Timeout: 2 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for CDP at %s: %w", endpoint, ctx.Err())
		default:
		}

		// Quick TCP check first (cheaper than full HTTP).
		u, _ := neturl.Parse(endpoint)
		conn, err := net.DialTimeout("tcp", u.Host, 500*time.Millisecond)
		if err != nil {
			select {
			case <-ctx.Done():
				return fmt.Errorf("timeout waiting for CDP at %s: %w", endpoint, ctx.Err())
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}
		conn.Close()

		// TCP is open — verify CDP responds.
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("create CDP probe request: %w", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for CDP at %s: %w", endpoint, ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// writeAutomationPrefs writes a Chrome Preferences file that disables the
// password manager, autofill, credential prompts, and other dialogs that
// interfere with automation. This is necessary because CLI flags alone do not
// suppress all Chrome-managed modals (e.g. password breach warnings).
func writeAutomationPrefs(userDataDir string) error {
	defaultDir := filepath.Join(userDataDir, "Default")
	if err := os.MkdirAll(defaultDir, 0o755); err != nil {
		return err
	}

	prefs := map[string]any{
		"credentials_enable_service":                  false,
		"credentials_enable_autosignin":               false,
		"profile": map[string]any{
			"password_manager_enabled":              false,
			"password_manager_leak_detection":       false,
			"default_content_setting_values": map[string]any{
				"notifications": 2, // block
			},
		},
		"autofill": map[string]any{
			"profile_enabled":    false,
			"credit_card_enabled": false,
		},
		"savefile": map[string]any{
			"default_directory": os.TempDir(),
		},
		"download": map[string]any{
			"prompt_for_download": false,
		},
		"password_manager": map[string]any{
			"enabled": false,
		},
	}

	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(defaultDir, "Preferences"), data, 0o644)
}
