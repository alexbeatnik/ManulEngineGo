// Package browser defines the abstract interfaces between the engine core and
// browser backends. CDP is the first implementation, but the interface is
// designed to accommodate future backends (Firefox, WebKit, native desktop).
//
// IMPORTANT: the Page interface deliberately does not include element-resolution
// or targeting logic. The browser backend is responsible only for:
//   - raw page access and navigation
//   - JS evaluation and DOM probing
//   - input dispatch (click, type, scroll)
//   - session lifecycle
//
// All DOM intelligence (candidate extraction, heuristics, scoring, target
// resolution) lives in pkg/runtime, pkg/heuristics, and pkg/scorer.
package browser

import (
	"context"
	"time"
)

// Page is the abstract interface for a single browser tab/page.
// Implementations are intended for use from a single goroutine.
type Page interface {
	// Navigate loads the given URL and waits for the initial load event.
	Navigate(ctx context.Context, url string) error

	// EvalJS evaluates a JavaScript expression in the page context and returns
	// the JSON-encoded result value.
	EvalJS(ctx context.Context, expr string) ([]byte, error)

	// CallProbe calls a serialized JS arrow-function expression with a JSON argument.
	// This is the primary mechanism for running in-page heuristic probes.
	// fn is the complete JS function expression (not a function name).
	// arg is the argument, marshaled to JSON before invocation.
	CallProbe(ctx context.Context, fn string, arg any) ([]byte, error)

	// Click performs a left-click at the given viewport coordinates.
	Click(ctx context.Context, x, y float64) error

	// Focus focuses the element resolved by ID or XPath.
	Focus(ctx context.Context, id int, xpath string) error

	// SetInputValue sets the value of an input element at the given ID or XPath,
	// dispatching the appropriate input/change events.
	SetInputValue(ctx context.Context, id int, xpath, value string) error

	// SetChecked sets the checked state of a checkbox or radio element.
	SetChecked(ctx context.Context, id int, xpath string, checked bool) error

	// ScrollIntoView scrolls the element at the given ID or XPath into the viewport.
	ScrollIntoView(ctx context.Context, id int, xpath string) error

	// ScrollPage scrolls the page or a container by the viewport height.
	// direction is "down" or "up". container is an optional locator string
	// understood by the backend (CDP accepts XPath and CSS selectors; empty = window).
	ScrollPage(ctx context.Context, direction, container string) error

	// DoubleClick performs a double-click at the given viewport coordinates.
	DoubleClick(ctx context.Context, x, y float64) error

	// RightClick performs a right-click (context menu) at the given viewport coordinates.
	RightClick(ctx context.Context, x, y float64) error

	// Hover moves the mouse to the given viewport coordinates without clicking.
	Hover(ctx context.Context, x, y float64) error

	// DragAndDrop simulates a drag from (fromX, fromY) to (toX, toY).
	DragAndDrop(ctx context.Context, fromX, fromY, toX, toY float64) error

	// SetFileInput sets file paths on a file input element at the given ID or XPath.
	SetFileInput(ctx context.Context, id int, xpath string, filePaths []string) error

	// Screenshot captures a PNG screenshot of the current page.
	Screenshot(ctx context.Context) ([]byte, error)

	// WaitForResponse waits for a network response matching the URL pattern.
	WaitForResponse(ctx context.Context, urlPattern string, timeout time.Duration) error

	// HighlightElement injects a temporary border highlight for debugging.
	HighlightElement(ctx context.Context, id int, xpath string, durationMS int) error

	// ClearHighlight immediately removes any active highlight (flash or debug)
	// from the page. It must not panic if the page has navigated or the element
	// has been destroyed.
	ClearHighlight(ctx context.Context) error

	// GetElementCenter returns the center viewport coordinates of an element.
	GetElementCenter(ctx context.Context, id int, xpath string) (float64, float64, error)

	// DispatchKey dispatches a keyboard event. key is the key name (e.g. "Enter").
	// modifiers is a bitmask: 1=Alt, 2=Ctrl, 4=Meta, 8=Shift.
	DispatchKey(ctx context.Context, key string, modifiers int) error

	// CurrentURL returns the current page URL.
	CurrentURL(ctx context.Context) (string, error)

	// WaitForLoad waits for the page's load event to fire. Used after a click
	// that triggers navigation, to ensure the new page is ready for interaction.
	WaitForLoad(ctx context.Context) error

	// Wait pauses execution on the page side (not just sleep) — e.g. waits for
	// navigation or network idle. Use context deadline for hard timeout.
	Wait(ctx context.Context, duration time.Duration) error

	// Frames returns the page's frames (main + iframes) with their per-frame
	// JavaScript execution contexts. Index 0 is the main frame.
	Frames(ctx context.Context) ([]FrameRef, error)

	// EvalJSInFrame evaluates an expression in the execution context of the
	// frame at frameIndex (0 == main frame). It is the per-frame counterpart
	// of EvalJS and is what makes iframe content reachable.
	EvalJSInFrame(ctx context.Context, frameIndex int, expr string) ([]byte, error)

	// CallProbeInFrame calls a serialized JS arrow-function with a JSON arg in
	// the frame at frameIndex (0 == main frame). Per-frame counterpart of CallProbe.
	CallProbeInFrame(ctx context.Context, frameIndex int, fn string, arg any) ([]byte, error)

	// Close closes the page/tab (does not close the browser).
	Close() error
}

// FrameRef identifies one frame (main document or iframe) and the index used to
// route per-frame evaluation. Index 0 is always the main frame.
type FrameRef struct {
	Index int
	URL   string
	Name  string
}

// Browser is the abstract interface for a browser connection.
type Browser interface {
	// FirstPage returns the first available page target.
	FirstPage(ctx context.Context) (Page, error)

	// NewPage opens a new browser tab/page.
	NewPage(ctx context.Context) (Page, error)

	// Close terminates the browser connection (does not kill the browser process).
	Close() error
}
