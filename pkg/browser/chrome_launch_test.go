package browser

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLaunchChrome_SurvivesContextCancel guards the launch-context decoupling:
// the context passed to LaunchChrome scopes only the startup wait, NOT the
// Chrome process. Regression: exec.CommandContext tied Chrome's lifetime to
// the launching task's context, so the browser was killed seconds after every
// action that had launched it (the embedding app cancels that context when
// the task's pipeline ends).
//
// Skipped unless MANUL_TEST_LAUNCH=1, so plain CI never spawns a Chrome:
//
//	MANUL_TEST_LAUNCH=1 go test -run TestLaunchChrome ./pkg/browser/
func TestLaunchChrome_SurvivesContextCancel(t *testing.T) {
	if os.Getenv("MANUL_TEST_LAUNCH") == "" {
		t.Skip("set MANUL_TEST_LAUNCH=1 to run (spawns a real headless Chrome)")
	}
	if _, err := findChrome(""); err != nil {
		t.Skipf("no chrome binary: %v", err)
	}

	// A dedicated port: 9222 may be busy with the user's real session.
	ctx, cancel := context.WithCancel(context.Background())
	cp, err := LaunchChrome(ctx, ChromeOptions{Port: 9777, Headless: true, DisableGPU: true})
	if err != nil {
		cancel()
		t.Fatalf("launch: %v", err)
	}
	defer cp.Close()

	// The launching task ends. Give a context-tied kill time to land so a
	// regression actually fails the probe below.
	cancel()
	time.Sleep(500 * time.Millisecond)

	resp, err := http.Get(cp.Endpoint() + "/json/version")
	if err != nil {
		t.Fatalf("chrome died after launch-context cancel: %v", err)
	}
	resp.Body.Close()

	// Close still owns teardown: after it, the endpoint must stop answering.
	if err := cp.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if resp, err := http.Get(cp.Endpoint() + "/json/version"); err == nil {
		resp.Body.Close()
		t.Fatalf("chrome still answering after Close")
	}
}
