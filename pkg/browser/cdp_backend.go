// Package browser — CDP backend implementation.
package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexbeatnik/ManulHeart/pkg/cdp"
)

// CDPBrowser is the Chrome DevTools Protocol implementation of Browser.
type CDPBrowser struct {
	endpoint string
}

// NewCDPBrowser creates a CDPBrowser connected to the given HTTP endpoint.
// Example endpoint: "http://127.0.0.1:9222"
func NewCDPBrowser(endpoint string) *CDPBrowser {
	return &CDPBrowser{endpoint: endpoint}
}

// FirstPage attaches to the first available page target in the running Chrome instance.
func (b *CDPBrowser) FirstPage(ctx context.Context) (Page, error) {
	targets, err := cdp.ListTargets(ctx, b.endpoint)
	if err != nil {
		return nil, fmt.Errorf("browser: list targets: %w", err)
	}
	target, err := cdp.FindPageTarget(targets)
	if err != nil {
		return nil, fmt.Errorf("browser: find page target: %w", err)
	}
	conn, err := cdp.DialTarget(ctx, target.WSURL)
	if err != nil {
		return nil, fmt.Errorf("browser: dial target %q: %w", target.WSURL, err)
	}
	return &CDPPage{conn: conn}, nil
}

// PageMatching attaches to the first page-type target whose URL contains
// the given substring (case-insensitive). Returns an error when no target
// matches; callers should typically fall back to FirstPage or surface a
// clear error to the user.
//
// Chrome's /json/list orders targets most-recently-active first, so when
// the user has several tabs of the same site open, this picks the one
// they've used most recently. That matters for multi-tab CDP sessions
// where "the first page" would otherwise be a lottery.
func (b *CDPBrowser) PageMatching(ctx context.Context, urlSubstr string) (Page, error) {
	if urlSubstr == "" {
		return b.FirstPage(ctx)
	}
	targets, err := cdp.ListTargets(ctx, b.endpoint)
	if err != nil {
		return nil, fmt.Errorf("browser: list targets: %w", err)
	}
	needle := strings.ToLower(urlSubstr)
	for _, t := range targets {
		if t.Type != "page" {
			continue
		}
		if !strings.Contains(strings.ToLower(t.URL), needle) {
			continue
		}
		conn, err := cdp.DialTarget(ctx, t.WSURL)
		if err != nil {
			return nil, fmt.Errorf("browser: dial target %q: %w", t.WSURL, err)
		}
		return &CDPPage{conn: conn}, nil
	}
	return nil, fmt.Errorf("browser: no page target with URL containing %q (found %d targets)", urlSubstr, len(targets))
}

// NewPage is not yet implemented for CDP (requires Target.createTarget).
func (b *CDPBrowser) NewPage(ctx context.Context) (Page, error) {
	return nil, fmt.Errorf("browser: NewPage not yet implemented for CDP backend")
}

// Close is a no-op for CDPBrowser — we don't own the browser process.
func (b *CDPBrowser) Close() error { return nil }

// ── CDPPage ───────────────────────────────────────────────────────────────────

// CDPPage is the CDP implementation of Page.
type CDPPage struct {
	conn *cdp.Conn
}

func (p *CDPPage) Navigate(ctx context.Context, url string) error {
	if err := cdp.Navigate(ctx, p.conn, url); err != nil {
		return err
	}

	// Two-phase wait to avoid the "stale readyState" race condition:
	// Phase 1 — wait for readyState to leave "complete", indicating
	//   the browser has begun the new navigation. We cap this at 500ms
	//   so same-page navigations (anchors, cached pages) don't hang.
	phase1Deadline := time.Now().Add(500 * time.Millisecond)
	const poll = 50 * time.Millisecond
	for time.Now().Before(phase1Deadline) {
		raw, err := cdp.Evaluate(ctx, p.conn, "document.readyState")
		if err != nil {
			// JS eval can fail briefly during navigation — that itself
			// means the navigation has started.
			break
		}
		if stateStr, ok := raw.(string); ok && stateStr != "complete" {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(poll):
		}
	}

	// Phase 2 — wait for the new page to reach readyState=="complete".
	return p.WaitForLoad(ctx)
}

func (p *CDPPage) EvalJS(ctx context.Context, expr string) ([]byte, error) {
	// Callers commonly pass an arrow-function literal like `() => {...}` —
	// intending to run it, not just evaluate the function object. Plain
	// Runtime.evaluate on such an expression returns the function (which
	// serializes to `{}` under returnByValue) and produces zero scan
	// results. Auto-invoke it so the JS body actually runs.
	wrappedExpr := expr
	trimmed := strings.TrimSpace(expr)
	if strings.HasPrefix(trimmed, "() =>") || strings.HasPrefix(trimmed, "()=>") ||
		strings.HasPrefix(trimmed, "function") {
		wrappedExpr = "(" + trimmed + ")()"
	}

	raw, err := cdp.Evaluate(ctx, p.conn, wrappedExpr)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		b, e := json.Marshal(v)
		return b, e
	}
}

func (p *CDPPage) CallProbe(ctx context.Context, fn string, arg any) ([]byte, error) {
	raw, err := cdp.CallFunctionOn(ctx, p.conn, fn, arg)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		b, e := json.Marshal(v)
		return b, e
	}
}

func (p *CDPPage) Click(ctx context.Context, x, y float64) error {
	return cdp.Click(ctx, p.conn, x, y)
}

func (p *CDPPage) Focus(ctx context.Context, id int, xpath string) error {
	return p.conn.Focus(ctx, id, xpath)
}

func (p *CDPPage) SetInputValue(ctx context.Context, id int, xpath, value string) error {
	return p.conn.SetInputValue(ctx, id, xpath, value)
}

func (p *CDPPage) ScrollIntoView(ctx context.Context, id int, xpath string) error {
	return p.conn.ScrollIntoView(ctx, id, xpath)
}
func (p *CDPPage) SetChecked(ctx context.Context, id int, xpath string, checked bool) error {
	return p.conn.SetChecked(ctx, id, xpath, checked)
}


func (p *CDPPage) ScrollPage(ctx context.Context, direction, container string) error {
	return cdp.ScrollPage(ctx, p.conn, direction, container)
}

func (p *CDPPage) DoubleClick(ctx context.Context, x, y float64) error {
	return cdp.DoubleClick(ctx, p.conn, x, y)
}

func (p *CDPPage) RightClick(ctx context.Context, x, y float64) error {
	return cdp.RightClick(ctx, p.conn, x, y)
}

func (p *CDPPage) Hover(ctx context.Context, x, y float64) error {
	return cdp.Hover(ctx, p.conn, x, y)
}

func (p *CDPPage) DragAndDrop(ctx context.Context, fromX, fromY, toX, toY float64) error {
	return cdp.DragAndDrop(ctx, p.conn, fromX, fromY, toX, toY)
}

func (p *CDPPage) SetFileInput(ctx context.Context, id int, xpath string, filePaths []string) error {
	return p.conn.SetFileInput(ctx, id, xpath, filePaths)
}

func (p *CDPPage) Screenshot(ctx context.Context) ([]byte, error) {
	return cdp.Screenshot(ctx, p.conn)
}

func (p *CDPPage) WaitForResponse(ctx context.Context, urlPattern string, timeout time.Duration) error {
	return cdp.WaitForResponse(ctx, p.conn, urlPattern, timeout)
}

func (p *CDPPage) HighlightElement(ctx context.Context, id int, xpath string, durationMS int) error {
	return p.conn.HighlightElement(ctx, id, xpath, durationMS)
}

func (p *CDPPage) ClearHighlight(ctx context.Context) error {
	return p.conn.ClearHighlight(ctx)
}

func (p *CDPPage) GetElementCenter(ctx context.Context, id int, xpath string) (float64, float64, error) {
	return p.conn.GetElementCenter(ctx, id, xpath)
}

func (p *CDPPage) DispatchKey(ctx context.Context, key string, modifiers int) error {
	params := cdp.KeyEventParams{
		Key:                   key,
		WindowsVirtualKeyCode: keyToVirtualCode(key),
		Modifiers:             modifiers,
	}
	if err := cdp.DispatchKeyEvent(ctx, p.conn, "keyDown", params); err != nil {
		return err
	}
	return cdp.DispatchKeyEvent(ctx, p.conn, "keyUp", params)
}

// keyToVirtualCode maps common key names to Windows virtual key codes.
func keyToVirtualCode(key string) int {
	codes := map[string]int{
		"Enter": 13, "Tab": 9, "Escape": 27, "Backspace": 8,
		"Delete": 46, "ArrowUp": 38, "ArrowDown": 40,
		"ArrowLeft": 37, "ArrowRight": 39, "Home": 36, "End": 35,
		"PageUp": 33, "PageDown": 34, "Space": 32, "F1": 112,
		"F2": 113, "F3": 114, "F4": 115, "F5": 116, "F6": 117,
		"F7": 118, "F8": 119, "F9": 120, "F10": 121, "F11": 122, "F12": 123,
		"a": 65, "b": 66, "c": 67, "v": 86, "x": 88, "z": 90,
	}
	if code, ok := codes[key]; ok {
		return code
	}
	// Default: try first character as uppercase ASCII
	if len(key) == 1 && key[0] >= 'a' && key[0] <= 'z' {
		return int(key[0]) - 32
	}
	return 0
}

func (p *CDPPage) CurrentURL(ctx context.Context) (string, error) {
	return cdp.GetCurrentURL(ctx, p.conn)
}

func (p *CDPPage) WaitForLoad(ctx context.Context) error {
	// Poll document.readyState via JS instead of relying on CDP event registration.
	// Event-based approach misses loadEventFired when the page loads before we
	// register the handler. JS polling is always accurate.
	const pollInterval = 150 * time.Millisecond
	for {
		raw, err := cdp.Evaluate(ctx, p.conn, "document.readyState")
		if err == nil && raw != nil {
			if stateStr, ok := raw.(string); ok && stateStr == "complete" {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func (p *CDPPage) Wait(ctx context.Context, duration time.Duration) error {
	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *CDPPage) Close() error {
	return p.conn.Close()
}
