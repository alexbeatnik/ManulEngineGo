---
name: cdp-browser-backend
description: Add or modify low-level browser capabilities in ManulEngine (Go). Use when implementing a new Page interface method, touching Chrome lifecycle, CDP commands, input dispatch, or JS evaluation. Covers the interface-to-backend wiring pattern.
---

# CDP Browser Backend

ManulEngine (Go)'s browser abstraction has two layers:

1. **`pkg/browser/browser.go`** — Abstract `Page` / `Browser` interfaces. Runtime targets these.
2. **`pkg/browser/cdp_backend.go`** + **`pkg/cdp/cdp.go`** — CDP implementation. CDP is the only backend today.

The `concurrency-rules` skill covers goroutine safety of `cdp.Conn`; this skill covers the **API contract and wiring pattern**.

## Architecture

```
DSL command → runtime.resolveTarget → runtime.executeXxx
                                        │
                                        ▼
                              browser.Page interface
                                        │
                              pkg/browser/cdp_backend.go
                                        │
                              pkg/cdp/cdp.go  (command wrappers)
                                        │
                              pkg/cdp/conn.go (WebSocket transport)
```

**Key principle:** `pkg/browser.Page` deliberately contains **zero** element-resolution logic. It only performs raw actions at resolved coordinates or element IDs/XPaths. The scorer lives exclusively in `pkg/runtime`.

## Adding a new Page method — the 4-file checklist

If a new DSL command needs a browser capability that doesn't exist yet:

| File | What to change |
|------|---------------|
| `pkg/browser/browser.go` | Add method to `Page` interface |
| `pkg/cdp/cdp.go` | Add CDP command wrapper (or reuse existing) |
| `pkg/browser/cdp_backend.go` | Implement method on `CDPPage` |
| `pkg/runtime/mock.go` | Add stub to `MockPage` (compiler will remind you) |

### Example: adding `GetCookies()`

```go
// 1. pkg/browser/browser.go
type Page interface {
    // ... existing methods ...
    GetCookies(ctx context.Context) ([]Cookie, error)
}

// 2. pkg/cdp/cdp.go
func GetCookies(ctx context.Context, c *Conn) ([]Cookie, error) {
    res, err := c.Call(ctx, "Network.getAllCookies", nil)
    // ... unmarshal ...
}

// 3. pkg/browser/cdp_backend.go
func (p *CDPPage) GetCookies(ctx context.Context) ([]browser.Cookie, error) {
    return cdp.GetCookies(ctx, p.conn)
}

// 4. pkg/runtime/mock.go
func (m *MockPage) GetCookies(ctx context.Context) ([]browser.Cookie, error) {
    return nil, nil // or meaningful mock data
}
```

## Chrome process lifecycle (`pkg/browser/chrome.go`)

- **`LaunchChrome(ctx, opts)`** — spawns Chrome with automation flags, temp profile, and preference JSON. Blocks until CDP `/json` responds.
- **`ChromeOptions`** — `Port`, `UserDataDir`, `DisableGPU`, `Headless`.
- **`findChrome()`** — platform-specific binary discovery (Linux PATH → macOS .app → Windows Program Files).
- **Temp profile cleanup** — if `UserDataDir` is empty, `LaunchChrome` creates a temp dir and `ChromeProcess.Close()` removes it.
- **Automation prefs** — `writeAutomationPrefs()` disables password manager, autofill, notifications, and credential dialogs at the profile level. CLI flags alone cannot suppress all modals.

### Chrome CLI flags of note

Key flags hardcoded in `LaunchChrome`:
- `--remote-debugging-port={port}`
- `--headless=new` (not `--headless` — the new mode supports extensions and modern APIs)
- `--disable-gpu`
- `--user-data-dir={dir}`
- `--disable-features=PasswordLeakDetection,PasswordManagerOnboarding,...`

## JS evaluation patterns

Two CDP wrappers exist; understand the difference:

| Function | CDP method | Use case |
|----------|-----------|----------|
| `cdp.Evaluate(ctx, c, expr)` | `Runtime.evaluate` | Simple expression, returns `interface{}` |
| `cdp.CallFunctionOn(ctx, c, fn, arg)` | `Runtime.evaluate` (with serialized function call) | Probes — passes a JSON argument to a JS arrow function |

### EvalJS return-type handling

`CDPPage.EvalJS` (and `CallProbe`) normalize CDP's `interface{}` into `[]byte`:
- `[]byte` → passed through
- `string` → converted to `[]byte`
- anything else → `json.Marshal`

When unmarshaling inside runtime, always handle all three origins.

### The `window.__manulReg` registry

Several `cdp.go` methods resolve elements by **ID first**, falling back to XPath:

```js
var el = (window.__manulReg && window.__manulReg[ID]) ||
         document.evaluate(XPATH, ...).singleNodeValue;
```

`__manulReg` is populated by the snapshot probe (`pkg/heuristics/snapshot_probe.js`) so repeated actions on the same element avoid XPath re-evaluation. If you write a new element-targeting CDP method, follow this pattern.

## Input dispatch

### Mouse events

All mouse actions use `Input.dispatchMouseEvent`:
- `Click` — `mousePressed` + `mouseReleased`, `button: "left"`, `clickCount: 1`
- `DoubleClick` — press/release ×2, second pair with `clickCount: 2`
- `RightClick` — `button: "right"`
- `Hover` — `type: "mouseMoved"`
- `DragAndDrop` — move → press → move → release

### Keyboard events

`DispatchKey` sends `keyDown` + `keyUp` via `Input.dispatchKeyEvent` with Windows virtual-key codes. The `keyToVirtualCode` map covers common keys; single-character keys default to uppercase ASCII. Modifiers are a bitmask: `1=Alt, 2=Ctrl, 4=Meta, 8=Shift`.

## Navigation race-condition avoidance

`CDPPage.Navigate` uses a **two-phase wait**:

1. **Phase 1** — poll `document.readyState` until it **leaves** `"complete"` (capped at 500ms). This catches same-page navigations and cached pages that never fire `loadEventFired`.
2. **Phase 2** — poll until `readyState == "complete"` again.

Never rely solely on CDP `Page.loadEventFired` events — they are missed if the page loads before the listener attaches.

## Network wait (`WaitForResponse`)

1. Calls `Network.enable`
2. `Subscribe()`s to CDP events
3. Listens for `Network.responseReceived` and does a **suffix match** on the URL
4. `defer Network.disable` with a 5-second bounded cleanup context

Always `defer sub.Close()` on the subscription.

## Highlight system

Two visual feedback mechanisms:

| Mechanism | Duration | Style | Attribute |
|-----------|----------|-------|-----------|
| Flash highlight | 2 seconds | 4px red border + `#ffeb3b` background | `data-manul-flash-old-border` / `data-manul-flash-old-bg` |
| Debug/explain highlight | Persistent | 4px magenta outline + glow | `data-manul-debug-highlight` |

`ClearHighlight` cleans up both. It must not panic if the page navigated — errors are swallowed.

## Common pitfalls

| Mistake | Consequence |
|---------|-------------|
| Forgetting `MockPage` stub | Compiler error in tests, caught only at CI |
| Using `Page.loadEventFired` events for navigation | Misses cached pages; use JS polling |
| Adding element-resolution logic to `browser.Page` | Violates architecture — resolution belongs in `pkg/runtime` |
| Not handling `window.__manulReg` in new element methods | XPath fallback works but slower; inconsistent with existing methods |
| `cdp.Evaluate` with `returnByValue: false` and no objectId handling | Leaks remote objects or panics on unmarshaling |
| Missing `defer sub.Close()` in event subscribers | Goroutine leak |

## Key files

- [`pkg/browser/browser.go`](../../../pkg/browser/browser.go) — `Page` / `Browser` interfaces
- [`pkg/browser/cdp_backend.go`](../../../pkg/browser/cdp_backend.go) — `CDPPage` implementation
- [`pkg/browser/chrome.go`](../../../pkg/browser/chrome.go) — `LaunchChrome`, `ChromeProcess`, `findChrome`
- [`pkg/cdp/cdp.go`](../../../pkg/cdp/cdp.go) — CDP command wrappers
- [`pkg/cdp/conn.go`](../../../pkg/cdp/conn.go) — `Conn`, `Call`, `Subscribe`
- [`pkg/runtime/mock.go`](../../../pkg/runtime/mock.go) — `MockPage`
