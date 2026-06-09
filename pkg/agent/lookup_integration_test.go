package agent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLookup_Integration exercises the real background-tab path against a live
// Chrome. It is skipped unless MANUL_TEST_CDP points at a running Chrome's CDP
// HTTP endpoint (e.g. http://127.0.0.1:9333), so it never runs in plain CI.
//
//	google-chrome --headless=new --remote-debugging-port=9333 --user-data-dir=/tmp/x &
//	MANUL_TEST_CDP=http://127.0.0.1:9333 go test -run TestLookup_Integration ./pkg/agent/
func TestLookup_Integration(t *testing.T) {
	endpoint := os.Getenv("MANUL_TEST_CDP")
	if endpoint == "" {
		t.Skip("set MANUL_TEST_CDP to a running Chrome CDP endpoint to run this test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	sess, err := Attach(ctx, endpoint, "", Options{})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	defer sess.Close()

	// A self-contained data: URL — no network needed.
	const page = "data:text/html,<title>LK</title><body><h1>Hello Lookup</h1><p>The background tab works.</p></body>"

	// Whole-body text (extractJS == "").
	text, err := sess.Lookup(ctx, page, 300*time.Millisecond, "")
	if err != nil {
		t.Fatalf("lookup body: %v", err)
	}
	if !strings.Contains(text, "Hello Lookup") || !strings.Contains(text, "background tab works") {
		t.Fatalf("body text missing expected content: %q", text)
	}

	// Custom extractor JS (prefer a specific node).
	got, err := sess.Lookup(ctx, page, 300*time.Millisecond, `(() => document.querySelector('h1').innerText)()`)
	if err != nil {
		t.Fatalf("lookup extractJS: %v", err)
	}
	if strings.TrimSpace(got) != "Hello Lookup" {
		t.Fatalf("extractJS result wrong: %q", got)
	}
}
