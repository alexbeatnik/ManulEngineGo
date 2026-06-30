// Package cdp provides a Chrome DevTools Protocol client for ManulEngine (Go).
//
// This package implements the low-level WebSocket messenger and the
// command-level CDP calls (Navigate, Evaluate, Click, etc.) used by
// pkg/browser/cdp_backend.go.
package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexbeatnik/ManulEngineGo/pkg/core"
)

// ── Types ──────────────────────────────────────────────────────────────────────

// Target represents a single debuggable browser target (page, worker, etc.).
type Target struct {
	// ID is the unique target ID assigned by Chrome.
	ID string `json:"id"`
	// Type is the target type ("page", "background_page", "worker", etc.)
	Type string `json:"type"`
	// Title is the page title.
	Title string `json:"title"`
	// URL is the current URL of the target.
	URL string `json:"url"`
	// WSURL is the WebSocket debugger URL for this target.
	WSURL string `json:"webSocketDebuggerUrl"`
}

// KeyEventParams holds the browser input event parameters for keyboard dispatch.
type KeyEventParams struct {
	Type                  string `json:"type"`
	Key                   string `json:"key"`
	Code                  string `json:"code,omitempty"`
	WindowsVirtualKeyCode int    `json:"windowsVirtualKeyCode,omitempty"`
	Modifiers             int    `json:"modifiers,omitempty"`
	// Text/UnmodifiedText carry the character a keyDown produces. Chrome
	// treats a keyDown WITHOUT text as a rawKeyDown: no keypress event fires,
	// so Enter does not submit forms and printable keys type nothing. Senders
	// must set both for character-producing keys (Enter → "\r") and clear them
	// on the matching keyUp.
	Text           string `json:"text,omitempty"`
	UnmodifiedText string `json:"unmodifiedText,omitempty"`
}

// ── CDP Commands ───────────────────────────────────────────────────────────────

// Navigate instructs the browser to navigate to the given URL.
func Navigate(ctx context.Context, c *Conn, url string) error {
	_, err := c.Call(ctx, "Page.navigate", map[string]interface{}{"url": url})
	return err
}

// Evaluate runs JavaScript in the page context and returns the result.
func Evaluate(ctx context.Context, c *Conn, expression string) (interface{}, error) {
	res, err := c.Call(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err != nil {
		return nil, err
	}

	// {"result": {"type": "...", "value": ...}}
	var wrap struct {
		Result struct {
			Value interface{} `json:"value"`
			Type  string      `json:"type"`
		} `json:"result"`
		ExceptionDetails interface{} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return nil, fmt.Errorf("unmarshal evaluate result: %w", err)
	}
	if wrap.ExceptionDetails != nil {
		return nil, fmt.Errorf("js exception: %v", wrap.ExceptionDetails)
	}
	return wrap.Result.Value, nil
}

// EvaluateInContext runs JavaScript in a specific execution context (frame).
// contextID == 0 falls back to the default/main context (plain Evaluate), so
// callers can pass a frame's context id unconditionally.
func EvaluateInContext(ctx context.Context, c *Conn, contextID int, expression string) (interface{}, error) {
	if contextID == 0 {
		return Evaluate(ctx, c, expression)
	}
	res, err := c.Call(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
		"contextId":     contextID,
	})
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Result struct {
			Value interface{} `json:"value"`
		} `json:"result"`
		ExceptionDetails interface{} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return nil, fmt.Errorf("unmarshal evaluate result: %w", err)
	}
	if wrap.ExceptionDetails != nil {
		return nil, fmt.Errorf("js exception: %v", wrap.ExceptionDetails)
	}
	return wrap.Result.Value, nil
}

// CallFunctionOn calls a JS function string with a JSON-serialized argument.
func CallFunctionOn(ctx context.Context, c *Conn, objectId string, arg interface{}) (interface{}, error) {
	// objectId is the function source; it is evaluated (and, with an arg,
	// invoked) in the default execution context.
	var expr string
	if arg == nil {
		expr = objectId
	} else {
		expr = fmt.Sprintf("(%s)(%s)", objectId, MustMarshalString(arg))
	}
	res, err := c.Call(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    expr,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err == nil {
		// Just parse Evaluate wrapper
		var wrap struct {
			Result struct {
				Value interface{} `json:"value"`
			} `json:"result"`
			ExceptionDetails interface{} `json:"exceptionDetails"`
		}
		if json.Unmarshal(res, &wrap) == nil {
			if wrap.ExceptionDetails != nil {
				return nil, fmt.Errorf("js exception: %v", wrap.ExceptionDetails)
			}
			return wrap.Result.Value, nil
		}
	}

	return nil, err
}

func MustMarshalString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return "undefined"
	}
	return string(b)
}

func evaluateObjectID(ctx context.Context, c *Conn, expression string) (string, error) {
	res, err := c.Call(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    expression,
		"returnByValue": false,
		"awaitPromise":  true,
	})
	if err != nil {
		return "", err
	}

	var wrap struct {
		Result struct {
			ObjectID string `json:"objectId"`
		} `json:"result"`
		ExceptionDetails interface{} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return "", fmt.Errorf("unmarshal evaluate handle result: %w", err)
	}
	if wrap.ExceptionDetails != nil {
		return "", fmt.Errorf("js exception: %v", wrap.ExceptionDetails)
	}
	if wrap.Result.ObjectID == "" {
		return "", fmt.Errorf("expression did not resolve to a remote object")
	}
	return wrap.Result.ObjectID, nil
}

func looksLikeXPath(locator string) bool {
	locator = strings.TrimSpace(locator)
	if locator == "" {
		return false
	}
	lower := strings.ToLower(locator)
	if strings.HasPrefix(lower, "xpath=") {
		return true
	}
	return strings.HasPrefix(locator, "/") ||
		strings.HasPrefix(locator, "//") ||
		strings.HasPrefix(locator, "./") ||
		strings.HasPrefix(locator, "../") ||
		strings.HasPrefix(locator, "(")
}

func locatorExpression(locator string) string {
	trimmed := strings.TrimSpace(locator)
	if strings.HasPrefix(strings.ToLower(trimmed), "xpath=") {
		trimmed = strings.TrimSpace(trimmed[6:])
	}
	if looksLikeXPath(trimmed) {
		return fmt.Sprintf("document.evaluate(%q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue", trimmed)
	}
	return fmt.Sprintf("document.querySelector(%q)", trimmed)
}

// Click dispatches a mouse click at the given page coordinates.
func Click(ctx context.Context, c *Conn, x, y float64) error {
	// MousePressed
	_, err := c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mousePressed",
		"button":     "left",
		"x":          x,
		"y":          y,
		"clickCount": 1,
	})
	if err != nil {
		return err
	}
	// MouseReleased
	_, err = c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mouseReleased",
		"button":     "left",
		"x":          x,
		"y":          y,
		"clickCount": 1,
	})
	return err
}

// DoubleClick dispatches a double-click at the given page coordinates.
func DoubleClick(ctx context.Context, c *Conn, x, y float64) error {
	// Press 1
	_, err := c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mousePressed", "button": "left", "x": x, "y": y, "clickCount": 1,
	})
	if err != nil {
		return err
	}
	_, err = c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseReleased", "button": "left", "x": x, "y": y, "clickCount": 1,
	})
	if err != nil {
		return err
	}

	// Press 2
	_, err = c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mousePressed", "button": "left", "x": x, "y": y, "clickCount": 2,
	})
	if err != nil {
		return err
	}
	_, err = c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseReleased", "button": "left", "x": x, "y": y, "clickCount": 2,
	})
	return err
}

// RightClick dispatches a right-click (contextmenu) at the given page coordinates.
func RightClick(ctx context.Context, c *Conn, x, y float64) error {
	_, err := c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mousePressed",
		"button":     "right",
		"x":          x,
		"y":          y,
		"clickCount": 1,
	})
	if err != nil {
		return err
	}
	_, err = c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type":       "mouseReleased",
		"button":     "right",
		"x":          x,
		"y":          y,
		"clickCount": 1,
	})
	return err
}

// Hover dispatches a mousemove event at the given page coordinates.
func Hover(ctx context.Context, c *Conn, x, y float64) error {
	_, err := c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	})
	return err
}

// DragAndDrop dispatches a drag-and-drop sequence.
func DragAndDrop(ctx context.Context, c *Conn, fromX, fromY, toX, toY float64) error {
	// Mouse move to start
	if err := Hover(ctx, c, fromX, fromY); err != nil {
		return err
	}
	// Press
	_, err := c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mousePressed", "button": "left", "x": fromX, "y": fromY, "clickCount": 1,
	})
	if err != nil {
		return err
	}

	// Move slowly (simulate multiple steps?) Simple 1-step move:
	_, err = c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseMoved", "button": "left", "x": toX, "y": toY,
	})
	if err != nil {
		return err
	}

	// Release
	_, err = c.Call(ctx, "Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseReleased", "button": "left", "x": toX, "y": toY, "clickCount": 1,
	})
	return err
}

// Focus focuses the element resolved by ID or XPath.
func (c *Conn) Focus(ctx context.Context, id int, xpath string) error {
	js := fmt.Sprintf(`
		var el = (window.__manulReg && window.__manulReg[%d]) || document.evaluate(%q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (el) el.focus();
	`, id, xpath)
	_, err := Evaluate(ctx, c, js)
	return err
}

// SetInputValue sets the value of an input element resolved by ID or XPath.
// Uses the native HTMLInputElement/HTMLTextAreaElement value setter to
// bypass framework-level overrides (React, Vue, etc.) that intercept
// the value property on individual elements.
func (c *Conn) SetInputValue(ctx context.Context, id int, xpath, value string) error {
	js := fmt.Sprintf(`
		var el = (window.__manulReg && window.__manulReg[%[1]d]) || document.evaluate(%[2]q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (el) {
			var targetEl = el;
			if (el.tagName !== 'INPUT' && el.tagName !== 'TEXTAREA' && el.tagName !== 'SELECT' && el.getAttribute('contenteditable') !== 'true') {
				if (el.tagName === 'LABEL' && el.htmlFor) {
					targetEl = document.getElementById(el.htmlFor) || targetEl;
				} else {
					var child = el.querySelector('input, textarea, select');
					if (child) {
						targetEl = child;
					} else if (el.nextElementSibling) {
						var next = el.nextElementSibling.matches('input, textarea, select') ? el.nextElementSibling : el.nextElementSibling.querySelector('input, textarea, select');
						if (next) targetEl = next;
					} else if (el.parentElement && el.parentElement.nextElementSibling) {
						var nextParent = el.parentElement.nextElementSibling;
						var pChild = nextParent.matches('input, textarea, select') ? nextParent : nextParent.querySelector('input, textarea, select');
						if (pChild) targetEl = pChild;
					}
				}
			}
			el = targetEl;

			// Use the native value setter so React/Vue/Angular state updates fire.
			var proto = Object.getPrototypeOf(el);
			var nativeSetter = null;
			while (proto && proto !== Object.prototype) {
				var desc = Object.getOwnPropertyDescriptor(proto, 'value');
				if (desc && desc.set) {
					nativeSetter = desc.set;
					break;
				}
				proto = Object.getPrototypeOf(proto);
			}

			if (nativeSetter) {
				nativeSetter.call(el, %[3]q);
			} else {
				el.value = %[3]q;
			}
			el.dispatchEvent(new Event('input', { bubbles: true }));
			el.dispatchEvent(new Event('change', { bubbles: true }));
			// Focus the element so a subsequent PRESS Enter / Tab / etc.
			// reaches it as document.activeElement. Without this, SPA
			// sites like YouTube swallow the key on document.body and
			// the form is never submitted.
			if (typeof el.focus === 'function') {
				try { el.focus({ preventScroll: true }); } catch (_) { el.focus(); }
			}
		}
	`, id, xpath, value)
	_, err := Evaluate(ctx, c, js)
	return err
}

func (c *Conn) SetChecked(ctx context.Context, id int, xpath string, checked bool) error {
	js := fmt.Sprintf(`
		var el = (window.__manulReg && window.__manulReg[%[1]d]) || document.evaluate(%[2]q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (el) {
			var targetEl = el;
			var desiredAria = %[3]v ? 'true' : 'false';
			// Refinement: find checkbox/radio if el is not one
			if (el.tagName !== 'INPUT' || (el.type !== 'checkbox' && el.type !== 'radio')) {
				if (el.tagName === 'LABEL' && el.htmlFor) {
					targetEl = document.getElementById(el.htmlFor) || targetEl;
				} else {
					var child = el.querySelector('input[type=checkbox], input[type=radio]');
					if (child) {
						targetEl = child;
					} else {
						// Look in nearby siblings or parents (common in tables)
						var cell = el.closest('td, th, div');
						if (cell) {
							var cb = cell.querySelector('input[type=checkbox], input[type=radio]') || 
							         cell.parentElement.querySelector('input[type=checkbox], input[type=radio]');
							if (cb) targetEl = cb;
						}
					}
				}
			}
			el = targetEl;

			if (el.tagName === 'INPUT' && (el.type === 'checkbox' || el.type === 'radio')) {
				if (el.checked !== %[3]v) {
					if (typeof el.scrollIntoView === 'function') {
						el.scrollIntoView({ block: 'center', inline: 'center' });
					}
					if (typeof el.focus === 'function') {
						try { el.focus({ preventScroll: true }); } catch (_) { el.focus(); }
					}

					// Let the browser perform the native toggle first.
					el.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true, view: window }));
					el.dispatchEvent(new MouseEvent('mouseup', { bubbles: true, cancelable: true, view: window }));
					if (typeof el.click === 'function') {
						el.click();
					} else {
						el.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
					}

					// If the native click did not produce the requested state,
					// fall back to the native checked setter and emit value events.
					if (el.checked !== %[3]v) {
						var proto = Object.getPrototypeOf(el);
						var nativeSetter = null;
						while (proto && proto !== Object.prototype) {
							var desc = Object.getOwnPropertyDescriptor(proto, 'checked');
							if (desc && desc.set) {
								nativeSetter = desc.set;
								break;
							}
							proto = Object.getPrototypeOf(proto);
						}
						if (nativeSetter) {
							nativeSetter.call(el, %[3]v);
						} else {
							el.checked = %[3]v;
						}
						el.dispatchEvent(new Event('input', { bubbles: true }));
						el.dispatchEvent(new Event('change', { bubbles: true }));
					}
				}
			} else {
				var role = (el.getAttribute('role') || '').toLowerCase();
				if (role === 'checkbox' || role === 'radio' || role === 'switch') {
					var current = el.getAttribute('aria-checked');
					if (current !== desiredAria) {
						if (typeof el.scrollIntoView === 'function') {
							el.scrollIntoView({ block: 'center', inline: 'center' });
						}
						if (typeof el.focus === 'function') {
							try { el.focus({ preventScroll: true }); } catch (_) { el.focus(); }
						}

						el.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true, view: window }));
						el.dispatchEvent(new MouseEvent('mouseup', { bubbles: true, cancelable: true, view: window }));
						if (typeof el.click === 'function') {
							el.click();
						} else {
							el.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
						}

						if (el.getAttribute('aria-checked') !== desiredAria) {
							el.setAttribute('aria-checked', desiredAria);
							el.dispatchEvent(new Event('input', { bubbles: true }));
							el.dispatchEvent(new Event('change', { bubbles: true }));
						}
					}
				}
			}
		}
	`, id, xpath, checked)
	_, err := Evaluate(ctx, c, js)
	return err
}

// ScrollIntoView scrolls the element resolved by ID or XPath into the viewport.
func (c *Conn) ScrollIntoView(ctx context.Context, id int, xpath string) error {
	js := fmt.Sprintf(`
		var el = (window.__manulReg && window.__manulReg[%d]) || document.evaluate(%q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (el) el.scrollIntoView({block: "center", inline: "center"});
	`, id, xpath)
	_, err := Evaluate(ctx, c, js)
	return err
}

// ScrollPage scrolls the page or a container element in the given direction.
// Currently only direction="up" or "down" are well supported.
func ScrollPage(ctx context.Context, c *Conn, direction, container string) error {
	// A basic implementation. In prod this would use Input.synthesizeScrollGesture or JS.
	var amount = 500
	if direction == "up" {
		amount = -500
	}
	containerLower := strings.ToLower(strings.TrimSpace(container))
	if containerLower == string(core.ScrollStrategyGenericList) {
		js := fmt.Sprintf(`(() => {
			const target = document.querySelector('#dropdown') ||
				document.querySelector('[role="listbox"]') ||
				document.querySelector('[class*="dropdown"]');
			if (!target) return false;
			target.scrollBy({ top: %d, behavior: 'auto' });
			return true;
		})()`, amount)
		_, err := Evaluate(ctx, c, js)
		return err
	}
	js := fmt.Sprintf(`window.scrollBy(0, %d);`, amount)
	if container != "" {
		nodeExpr := locatorExpression(container)
		// A more robust selection: find the element, and if it's not scrollable, look for a scrollable child.
		js = fmt.Sprintf(`
			(() => {
				var el = %s;
				if (!el) return;
				
				const isScrollable = (node) => {
					const cs = window.getComputedStyle(node);
					const hasOverflow = (cs.overflowY === 'auto' || cs.overflowY === 'scroll');
					return hasOverflow && node.scrollHeight > node.clientHeight;
				};

				// Find the scrollable element (itself or deeply nested)
				var target = el;
				if (!isScrollable(el)) {
					var all = el.querySelectorAll('*');
					for(var i=0; i<all.length; i++) {
						if (isScrollable(all[i])) {
							target = all[i];
							// Don't break! Find the MOST deeply nested scrollable child (usually the one people mean)
						}
					}
				}
				target.scrollBy({ top: %d, behavior: 'auto' });
			})()
		`, nodeExpr, amount)
	}
	_, err := Evaluate(ctx, c, js)
	return err
}

// SetFileInput sets the file paths on a file input element resolved by ID or XPath.
func (c *Conn) SetFileInput(ctx context.Context, id int, xpath string, filePaths []string) error {
	js := fmt.Sprintf(`(() => {
		var el = (window.__manulReg && window.__manulReg[%[1]d]) || document.evaluate(%[2]q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (!el) return null;

		var targetEl = el;
		if (el.tagName !== 'INPUT' || el.type !== 'file') {
			if (el.tagName === 'LABEL') {
				if (el.control && el.control.tagName === 'INPUT' && el.control.type === 'file') {
					targetEl = el.control;
				} else if (el.htmlFor) {
					targetEl = document.getElementById(el.htmlFor) || targetEl;
				}
			}
			if ((targetEl.tagName !== 'INPUT' || targetEl.type !== 'file') && el.querySelector) {
				var child = el.querySelector('input[type=file]');
				if (child) targetEl = child;
			}
			if ((targetEl.tagName !== 'INPUT' || targetEl.type !== 'file') && el.nextElementSibling) {
				var next = el.nextElementSibling.matches('input[type=file]')
					? el.nextElementSibling
					: el.nextElementSibling.querySelector && el.nextElementSibling.querySelector('input[type=file]');
				if (next) targetEl = next;
			}
			if ((targetEl.tagName !== 'INPUT' || targetEl.type !== 'file') && el.parentElement) {
				var nearby = el.parentElement.querySelector && el.parentElement.querySelector('input[type=file]');
				if (nearby) targetEl = nearby;
			}
		}

		if (!targetEl || targetEl.tagName !== 'INPUT' || targetEl.type !== 'file') {
			return null;
		}
		return targetEl;
	})()`, id, xpath)

	objectID, err := evaluateObjectID(ctx, c, js)
	if err != nil {
		return fmt.Errorf("SetFileInput: resolve file input: %w", err)
	}

	// Get the backend node ID
	rawRes, err := c.Call(ctx, "DOM.requestNode", map[string]interface{}{
		"objectId": objectID,
	})
	if err != nil {
		return err
	}

	var res struct {
		NodeId int `json:"nodeId"`
	}
	if err := json.Unmarshal(rawRes, &res); err != nil {
		return fmt.Errorf("SetFileInput: unmarshal requestNode: %w", err)
	}

	_, err = c.Call(ctx, "DOM.setFileInputFiles", map[string]interface{}{
		"nodeId": res.NodeId,
		"files":  filePaths,
	})
	return err
}

// Screenshot captures a PNG screenshot of the current viewport.
func Screenshot(ctx context.Context, c *Conn) ([]byte, error) {
	res, err := c.Call(ctx, "Page.captureScreenshot", map[string]interface{}{
		"format": "png",
	})
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Data []byte `json:"data"` // base64 encoded by chrome, auto-decoded by Go's []byte unmarshal!
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return nil, fmt.Errorf("unmarshal screenshot: %w", err)
	}
	return wrap.Data, nil
}

// WaitForResponse waits for a network response whose URL matches the given pattern.
func WaitForResponse(ctx context.Context, c *Conn, urlPattern string, timeout time.Duration) error {
	// Enable network tracking first
	_, err := c.Call(ctx, "Network.enable", nil)
	if err != nil {
		return fmt.Errorf("Network.enable: %w", err)
	}

	sub := c.Subscribe()
	defer sub.Close()
	defer func() {
		// Bound the cleanup call so a dead/stuck socket cannot hang the worker forever.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = c.Call(ctx, "Network.disable", nil)
	}()

	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctxTimeout.Done():
			return fmt.Errorf("timeout waiting for response pattern %q", urlPattern)
		case event, ok := <-sub.C():
			if !ok {
				return fmt.Errorf("cdp connection closed while waiting for response pattern %q", urlPattern)
			}
			if event.Method == "Network.responseReceived" {
				var received struct {
					Response struct {
						URL string `json:"url"`
					} `json:"response"`
				}
				if err := json.Unmarshal(event.Params, &received); err == nil {
					// Extremely simple suffix/substring match
					if len(received.Response.URL) >= len(urlPattern) &&
						received.Response.URL[len(received.Response.URL)-len(urlPattern):] == urlPattern {
						return nil
					}
				}
			}
		}
	}
}

// HighlightElement injects a temporary border highlight for debugging.
// Matches Python manul-engine _highlight: 4px solid red border + #ffeb3b background.
func (c *Conn) HighlightElement(ctx context.Context, id int, xpath string, durationMS int) error {
	js := fmt.Sprintf(`
		var el = (window.__manulReg && window.__manulReg[%d]) || document.evaluate(%q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (el) {
			el.setAttribute('data-manul-flash-old-border', el.style.border || '');
			el.setAttribute('data-manul-flash-old-bg', el.style.backgroundColor || '');
			el.style.border = '4px solid red';
			el.style.backgroundColor = '#ffeb3b';
			setTimeout(() => {
				var oldBorder = el.getAttribute('data-manul-flash-old-border');
				if (oldBorder !== null) {
					el.style.border = oldBorder;
					el.removeAttribute('data-manul-flash-old-border');
				}
				var oldBg = el.getAttribute('data-manul-flash-old-bg');
				if (oldBg !== null) {
					el.style.backgroundColor = oldBg;
					el.removeAttribute('data-manul-flash-old-bg');
				}
			}, %d);
		}
	`, id, xpath, durationMS)
	_, err := Evaluate(ctx, c, js)
	return err
}

// ClearHighlight immediately removes any active highlight from the page.
// It restores original inline styles for flash highlights and removes debug
// highlight attributes. Errors are swallowed because the page may have
// navigated and destroyed the execution context.
func (c *Conn) ClearHighlight(ctx context.Context) error {
	js := `(() => {
		document.querySelectorAll('[data-manul-debug-highlight]').forEach(
			el => el.removeAttribute('data-manul-debug-highlight')
		);
		const s = document.getElementById('manul-debug-style');
		if (s) s.remove();
		document.querySelectorAll('[data-manul-flash-old-border]').forEach(el => {
			el.style.border = el.getAttribute('data-manul-flash-old-border') || '';
			el.removeAttribute('data-manul-flash-old-border');
		});
		document.querySelectorAll('[data-manul-flash-old-bg]').forEach(el => {
			el.style.backgroundColor = el.getAttribute('data-manul-flash-old-bg') || '';
			el.removeAttribute('data-manul-flash-old-bg');
		});
	})();`
	_, err := Evaluate(ctx, c, js)
	return err
}

// GetElementCenter returns the centre coordinates of an element resolved by ID or XPath.
func (c *Conn) GetElementCenter(ctx context.Context, id int, xpath string) (x, y float64, err error) {
	js := fmt.Sprintf(`
		var el = (window.__manulReg && window.__manulReg[%d]) || document.evaluate(%q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (!el) {
			throw new Error("Element not found");
		}
		el.scrollIntoView({behavior: 'instant', block: 'center', inline: 'center'});
		var rect = el.getBoundingClientRect();
		// If it's still outside, we might need a small delay, but instant scroll usually is synchronous.
		JSON.stringify({x: rect.x + rect.width/2, y: rect.y + rect.height/2});
	`, id, xpath)

	val, err := Evaluate(ctx, c, js)
	if err != nil {
		return 0, 0, err
	}

	var coords struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}

	// val should be a string containing JSON due to JSON.stringify
	str, ok := val.(string)
	if !ok {
		return 0, 0, fmt.Errorf("unexpected evaluate result format: %T", val)
	}
	if err := json.Unmarshal([]byte(str), &coords); err != nil {
		return 0, 0, err
	}
	return coords.X, coords.Y, nil
}

// DispatchKeyEvent sends a keyboard event to the currently focused element.
// eventType is one of "keyDown" / "keyUp" / "rawKeyDown" / "char" — it
// overrides params.Type so callers don't have to set it on both ends of
// a key-press pair.
func DispatchKeyEvent(ctx context.Context, c *Conn, eventType string, params KeyEventParams) error {
	params.Type = eventType
	_, err := c.Call(ctx, "Input.dispatchKeyEvent", params)
	return err
}

// GetCurrentURL returns the current URL of the page.
func GetCurrentURL(ctx context.Context, c *Conn) (string, error) {
	val, err := Evaluate(ctx, c, "window.location.href")
	if err != nil {
		return "", err
	}
	if s, ok := val.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("unexpected evaluation result for URL: %v", val)
}

// WaitForLoad is available but ManulEngine (Go) prefers JS-polling WaitForLoad
// in cdp_backend.go to avoid race conditions on cached pages.
func WaitForLoad(ctx context.Context, c *Conn) error {
	return nil // Handled in cdp_backend.go
}

// ── JSON helpers ───────────────────────────────────────────────────────────────

// MustMarshal marshals v to JSON, panicking on error (used in tests only).
func MustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
