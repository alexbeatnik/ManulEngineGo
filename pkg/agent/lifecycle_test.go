package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/runtime"
)

func TestPageState_ReadsTitleAndURL(t *testing.T) {
	sess := newTestSession(&runtime.MockPage{URL: "https://example.com/watch", Title: "A Video — Example"})
	ps, err := sess.PageState(context.Background())
	if err != nil {
		t.Fatalf("PageState failed: %v", err)
	}
	if ps.URL != "https://example.com/watch" {
		t.Errorf("URL not read: %q", ps.URL)
	}
	if ps.Title != "A Video — Example" {
		t.Errorf("Title not read: %q", ps.Title)
	}
}

func TestPageState_OnClosedSession(t *testing.T) {
	sess := newTestSession(&runtime.MockPage{URL: "https://example.com"})
	_ = sess.Close()
	if _, err := sess.PageState(context.Background()); err == nil {
		t.Errorf("expected error on closed session")
	}
}

func TestDiffPageState(t *testing.T) {
	// No change → empty report.
	same := PageState{Title: "T", URL: "u"}
	if got := DiffPageState(same, same); got != "" {
		t.Errorf("expected empty diff for identical state, got %q", got)
	}

	// URL change is surfaced.
	got := DiffPageState(
		PageState{Title: "Results", URL: "https://x/results"},
		PageState{Title: "Video", URL: "https://x/watch"},
	)
	if !strings.Contains(got, "results") || !strings.Contains(got, "watch") {
		t.Errorf("url change not reported: %q", got)
	}
	if !strings.Contains(got, "Results") || !strings.Contains(got, "Video") {
		t.Errorf("title change not reported: %q", got)
	}
}

// TestAlive covers the dead-Chrome detection an embedding app relies on to
// know when its cached Session must be reconnected: alive while the endpoint
// answers, dead once Chrome is gone (user closed the window), and never alive
// without an endpoint or after Close.
func TestAlive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/version" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	sess := newTestSession(&runtime.MockPage{URL: "https://example.com"})
	if sess.Alive(context.Background()) {
		t.Errorf("session without an endpoint must not be alive")
	}

	sess.endpoint = srv.URL
	if !sess.Alive(context.Background()) {
		t.Errorf("expected alive while endpoint %s answers", srv.URL)
	}

	srv.Close() // Chrome "exits": the endpoint stops answering
	if sess.Alive(context.Background()) {
		t.Errorf("expected dead once the endpoint stops answering")
	}
}

func TestAlive_ClosedSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sess := newTestSession(&runtime.MockPage{URL: "https://example.com"})
	sess.endpoint = srv.URL
	_ = sess.Close()
	if sess.Alive(context.Background()) {
		t.Errorf("closed session must not be alive even with a live endpoint")
	}
}

func TestCDPReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/version" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if !cdpReachable(context.Background(), srv.URL) {
		t.Errorf("expected reachable for live server %s", srv.URL)
	}
	if cdpReachable(context.Background(), "http://127.0.0.1:1") {
		t.Errorf("expected unreachable for dead endpoint")
	}
}

func TestRenderForLLM(t *testing.T) {
	pm := PageMap{
		URL: "https://example.com",
		Groups: []MapGroup{
			{
				Name: "Main",
				Elements: []MapElement{
					{Label: "First video", Role: "link"},
					{Label: "Second video", Role: "link"},
					{Label: "Third video", Role: "link"},
				},
				Truncated: 2, // Map already dropped 2
			},
			{Name: "Empty", Elements: nil},
		},
	}

	out := pm.RenderForLLM(2) // display cap of 2

	if !strings.Contains(out, "[Main] (5 items)") {
		t.Errorf("group count should reflect kept+truncated (3+2=5): %q", out)
	}
	// Labels are quoted (copy-pastable) and carry the DSL verb for the role.
	if !strings.Contains(out, "- 'First video' [link — CLICK]") {
		t.Errorf("missing quoted element line with action verb: %q", out)
	}
	if strings.Contains(out, "Third video") {
		t.Errorf("display cap of 2 not applied: %q", out)
	}
	// 1 hidden by display + 2 truncated by Map = 3 more.
	if !strings.Contains(out, "… +3 more") {
		t.Errorf("combined 'more' count wrong: %q", out)
	}
	if strings.Contains(out, "[Empty]") {
		t.Errorf("empty group should be skipped: %q", out)
	}
	// The header tells the model how to use the list, and the URL grounds it.
	if !strings.Contains(out, "EXACT text in quotes") {
		t.Errorf("usage header missing: %q", out)
	}
	if !strings.Contains(out, "Current page: https://example.com") {
		t.Errorf("page URL missing: %q", out)
	}
}

func TestRenderForLLM_RoleVerbs(t *testing.T) {
	pm := PageMap{Groups: []MapGroup{{
		Name: "Form",
		Elements: []MapElement{
			{Label: "Search", Role: "textbox"},
			{Label: "Country", Role: "combobox"},
			{Label: "Subscribe", Role: "checkbox"},
			{Label: "Submit", Role: "button"},
		},
	}}}
	out := pm.RenderForLLM(0)
	for _, want := range []string{
		"'Search' [textbox — FILL]",
		"'Country' [combobox — SELECT]",
		"'Subscribe' [checkbox — CHECK]",
		"'Submit' [button — CLICK]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
