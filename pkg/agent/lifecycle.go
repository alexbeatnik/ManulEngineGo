package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Connect is the one-call entry point for an embedding app: it attaches to an
// already-running Chrome when one is reachable, and otherwise launches (and
// owns) a fresh one. This frees a consumer from reimplementing the
// "is the debug port up? if not, spawn Chrome, wait for it, then attach"
// dance — the engine owns the whole browser lifecycle.
//
// Reachability is probed at opts.CDPURL (preferred) or http://127.0.0.1:<Port>
// (Port 0 → 9222). When the probe succeeds the returned Session does NOT own
// Chrome (Close leaves it running); when Connect launches, Close reaps it.
func Connect(ctx context.Context, opts Options) (*Session, error) {
	endpoint := opts.CDPURL
	if endpoint == "" {
		port := opts.Port
		if port == 0 {
			port = 9222
		}
		endpoint = fmt.Sprintf("http://127.0.0.1:%d", port)
	}

	if cdpReachable(ctx, endpoint) {
		return Attach(ctx, endpoint, "", opts)
	}
	return Launch(ctx, opts)
}

// cdpReachable reports whether a Chrome DevTools HTTP endpoint answers at
// <endpoint>/json/version within a short timeout. This is the same signal the
// CDP transport relies on, checked without opening a WebSocket.
func cdpReachable(ctx context.Context, endpoint string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	url := strings.TrimRight(endpoint, "/") + "/json/version"
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// PageState is a lightweight (title, URL) snapshot of the page currently loaded
// in the session — enough for an app to verify a navigation-class action did
// what it claimed ("did the URL change to a /watch page, or are we still on
// /results?") and to label the page for the user.
type PageState struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// PageState reads the current page's title and URL in one call. Errors reading
// either field are tolerated individually — a missing title or URL comes back
// empty rather than failing the whole snapshot, mirroring how an app treats
// "unknown" page state as a soft condition.
func (s *Session) PageState(ctx context.Context) (PageState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return PageState{}, fmt.Errorf("agent: session closed")
	}

	var ps PageState
	if url, err := s.page.CurrentURL(ctx); err == nil {
		ps.URL = url
	}
	if raw, err := s.page.EvalJS(ctx, "document.title"); err == nil {
		var title string
		if json.Unmarshal(raw, &title) == nil {
			ps.Title = strings.TrimSpace(title)
		} else {
			ps.Title = strings.TrimSpace(string(raw))
		}
	}
	return ps, nil
}

// DiffPageState renders a short before/after report an LLM can use to judge
// whether a navigation-class action landed where intended. It returns "" when
// nothing observable changed (same title and URL), so a caller never feeds the
// model a confusing "before == after" block.
func DiffPageState(before, after PageState) string {
	if before.Title == after.Title && before.URL == after.URL {
		return ""
	}
	var b strings.Builder
	b.WriteString("Page change:\n")
	if before.Title != after.Title {
		fmt.Fprintf(&b, "  title: %q → %q\n", before.Title, after.Title)
	}
	if before.URL != after.URL {
		fmt.Fprintf(&b, "  url: %s → %s\n", before.URL, after.URL)
	}
	return strings.TrimRight(b.String(), "\n")
}
