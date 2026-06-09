package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexbeatnik/ManulHeart/pkg/browser"
	"github.com/alexbeatnik/ManulHeart/pkg/heuristics"
)

// Lookup opens url in a NEW BACKGROUND tab (the session's active page is never
// switched away from), waits settle for client-side content to render, reads
// the page text, then closes the tab. It returns the sanitized result.
//
// This is the unobtrusive "read a page without disturbing the user" primitive:
// because it runs in the same browser/profile the session owns, logged-in
// state and anti-bot reputation carry over. An embedding app gets web lookups
// without writing any CDP or tab-management code.
//
// extractJS, when non-empty, is a JS expression (e.g. an IIFE) returning the
// string to extract — use it to prefer a specific region (an answer panel, a
// widget) over whole-page text. When empty, Lookup reads the sanitized visible
// body text via the standard page-text probe.
func (s *Session) Lookup(ctx context.Context, url string, settle time.Duration, extractJS string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", fmt.Errorf("agent: session closed")
	}
	if s.endpoint == "" {
		return "", fmt.Errorf("agent: Lookup needs a CDP endpoint (use Launch/Attach/Connect)")
	}

	b := browser.NewCDPBrowser(s.endpoint)
	page, targetID, err := b.OpenTarget(ctx, url)
	if err != nil {
		return "", fmt.Errorf("agent: lookup open tab: %w", err)
	}
	defer func() {
		_ = page.Close()
		// Reap the tab even if ctx is already done — use a fresh context so a
		// cancelled lookup doesn't leak a background tab.
		_ = b.CloseTarget(context.Background(), targetID)
	}()

	if settle > 0 {
		select {
		case <-time.After(settle):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	var raw []byte
	if strings.TrimSpace(extractJS) != "" {
		raw, err = page.EvalJS(ctx, extractJS)
	} else {
		raw, err = page.CallProbe(ctx, heuristics.BuildPageTextProbe(), "")
	}
	if err != nil {
		return "", fmt.Errorf("agent: lookup extract: %w", err)
	}

	var text string
	if json.Unmarshal(raw, &text) != nil {
		text = string(raw)
	}
	return sanitizeText(text), nil
}
