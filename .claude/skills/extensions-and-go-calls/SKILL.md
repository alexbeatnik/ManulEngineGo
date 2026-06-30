---
name: extensions-and-go-calls
description: Add or debug Go-level extensions — RegisterCustomControl and RegisterGoCall. Use when wiring a custom element handler, a CALL GO target, or when writing tests that involve the extension registry.
---

# Extensions and Go Calls

ManulEngine (Go) has two escape hatches from pure DSL: **custom controls** and **Go call handlers**.
Both are package-global registries defined in `pkg/runtime/extensions.go`.

## When to use each

| Mechanism | Use case |
|-----------|---------|
| `RegisterCustomControl` | A specific page + element combination requires non-standard browser interaction. The DSL command is still `CLICK`, `FILL`, etc. — the custom handler intercepts it before heuristic resolution. |
| `RegisterGoCall` | A `.hunt` file needs to invoke arbitrary Go code via `CALL GO handler_name`. Results flow back into hunt variables. |

## Custom control handlers

### Type signatures

```go
type CustomControlInvocation struct {
    Page       string            // normalized page label at invocation time
    Target     string            // normalized target label from the DSL command
    ActionType string            // "click", "input", "hover", etc. — see table below
    Value      string            // only for input-style commands (FILL, TYPE)
    Variables  map[string]string // snapshot of runtime variables at invocation time
    Command    dsl.Command       // full original command (read-only)
}

type CustomControlHandler func(context.Context, browser.Page, CustomControlInvocation) error
```

### Registration

```go
// Register at process init, BEFORE spawning any worker pool.
err := runtime.RegisterCustomControl("Login Page", "Remember me", myHandler)
```

Registration validates that both `page` and `target` are non-empty (after normalization).
Returns an error if either is empty; existing registrations are silently overwritten.

### Wildcard page (`"*"`)

Pass `"*"` as the page to match the target on **any** page:

```go
runtime.RegisterCustomControl("*", "Cookie Consent", dismissCookieBanner)
```

`GetCustomControl` checks the exact page match first, then falls back to `"*"`. The
wildcard does not participate in other registry operations — it is purely a lookup fallback.

### How the page label is resolved at runtime

The page label is derived from the live browser at the moment the command executes:

1. Reads `document.title` via JS (`page.ExecuteScript`).
2. If that fails or is empty, falls back to the URL: parses the host, drops `www.`, then takes the first non-empty path segment.

The label is then normalized with `normalizeRegistryLabel` (lowercase, trim, collapse
internal whitespace). Registration keys undergo the same normalization — so `"Login Page"`,
`"login page"`, and `" Login  Page "` all resolve to the same key.

> **Pitfall:** Never assume the page label matches the URL exactly. If your handler isn't
> firing, print `document.title` in the browser to check the exact label in use.

### ActionType mapping

Which DSL command triggers which `ActionType` in `CustomControlInvocation`:

| DSL command(s) | `ActionType` |
|----------------|-------------|
| `FILL`, `TYPE` | `"input"` |
| `CLICK` | `"click"` |
| `DOUBLE CLICK` | `"double_click"` |
| `RIGHT CLICK` | `"right_click"` |
| `HOVER` | `"hover"` |
| `SELECT` | `"select"` |
| `CHECK` | `"check"` |
| `UNCHECK` | `"uncheck"` |
| `UPLOAD` | `"upload"` |

Commands that have no `ActionType` mapping (`NAVIGATE`, `WAIT`, `PRESS`, `VERIFY`, etc.)
do not consult the custom control registry at all.

### Interception point

`tryExecuteCustomControl` is called at the top of `executeCommand`, **before** the
heuristic targeting pipeline (`resolveTarget`). If a registered handler matches, the
entire scoring + DOM probe is skipped. The handler receives the raw `browser.Page` and
is responsible for all browser interaction directly.

## CALL GO handlers

### Type signatures

```go
type GoCallInvocation struct {
    Name      string            // handler name as registered
    Args      []string          // positional args from DSL, after variable substitution
    Variables map[string]string // snapshot of runtime variables at invocation time
    Page      browser.Page      // the live page (read-only recommended)
    Command   dsl.Command       // full original command (read-only)
}

type GoCallHandler func(context.Context, GoCallInvocation) (any, error)
```

### Registration

```go
runtime.RegisterGoCall("fetch_token", func(ctx context.Context, inv runtime.GoCallInvocation) (any, error) {
    return getAuthToken(inv.Args[0]), nil
})
```

### DSL syntax

```
CALL GO fetch_token '{api_key}' INTO {token}
```

- Args are space-separated quoted or unquoted tokens after the handler name.
- Variable substitution (`{key}`) happens before the handler is called.
- The return value is stored via `fmt.Sprint(result)` into `GoCallResultVar` as a string
  variable at `LevelRow` scope. If the handler returns `nil`, an empty string is stored.

### Error handling

If the handler returns a non-nil error, execution stops and the error propagates as a
hunt failure — same as any other runtime error.

## Lifecycle invariants

1. **Register at process init.** Call `RegisterCustomControl` / `RegisterGoCall` before
   `worker.NewPool` or any `worker.AdoptWorker`. The maps are protected by a
   `sync.RWMutex`, but late registration races with workers already reading the map.
2. **Handlers must be goroutine-safe.** Every worker in the pool may invoke the same
   handler simultaneously. Handlers must not share unsynchronized mutable state.
3. **Do NOT hold the registry lock in a handler.** Handlers are called with the registry
   read-lock released — re-entrant registration from inside a handler will deadlock.

## Testing with the extension registry

Use `ResetRuntimeRegistries()` to clear both maps between test cases:

```go
func TestMyHandler(t *testing.T) {
    runtime.ResetRuntimeRegistries() // exported test helper
    defer runtime.ResetRuntimeRegistries()

    runtime.RegisterCustomControl("*", "Accept", func(...) error { ... })
    // ... run hunt with AdoptWorker ...
}
```

> **Warning:** `ResetRuntimeRegistries()` must NOT be called while any worker is active.
> All adopted workers must be closed (via `w.Close()`) before resetting.

### Full test pattern

```go
func TestCustomControl_Click(t *testing.T) {
    runtime.ResetRuntimeRegistries()
    defer runtime.ResetRuntimeRegistries()

    called := false
    runtime.RegisterCustomControl("*", "Accept Cookies", func(
        ctx context.Context, p browser.Page, inv runtime.CustomControlInvocation,
    ) error {
        called = true
        if inv.ActionType != "click" {
            t.Errorf("expected click, got %q", inv.ActionType)
        }
        return nil
    })

    page := &runtime.MockPage{
        Title: "My Page",
        Elements: []dom.ElementSnapshot{
            {ID: 1, Tag: "button", VisibleText: "Accept Cookies",
             IsVisible: true, Rect: dom.Rect{Left: 10, Top: 10, Width: 80, Height: 30}},
        },
    }
    w := worker.AdoptWorker(1, config.Default(), page, nil)
    defer w.Close()

    h, _ := dsl.Parse(strings.NewReader(`
STEP 1: Accept
    CLICK the 'Accept Cookies' button
DONE.
`))
    res, err := w.Run(context.Background(), h)
    if err != nil {
        t.Fatal(err)
    }
    if !res.Passed {
        t.Fatalf("hunt failed: %v", res.FailReason)
    }
    if !called {
        t.Fatal("custom handler was not invoked")
    }
}
```

## Key files

- [`pkg/runtime/extensions.go`](../../../pkg/runtime/extensions.go) — registry maps, types, `RegisterCustomControl`, `RegisterGoCall`, `GetCustomControl`, `GetGoCall`, `ResetRuntimeRegistries`, `normalizeRegistryLabel`, `customControlActionType`.
- [`pkg/runtime/runtime.go`](../../../pkg/runtime/runtime.go) — `tryExecuteCustomControl` (interception), `executeCallGo` (CALL GO dispatch).
- [`pkg/runtime/mock.go`](../../../pkg/runtime/mock.go) — `MockPage` for tests.

## Common pitfalls

| Mistake | Consequence |
|---------|-------------|
| Registering after pool spawns | Race condition — handler may not be seen by workers already running |
| Non-goroutine-safe handler state | Data race under `go test -race` |
| Assuming page label matches URL hostname exactly | Handler never fires; check `document.title` first |
| Returning a non-string from `GoCallHandler` | Value stored as `fmt.Sprint(result)` — floats and structs stringify in unexpected formats |
| Calling `ResetRuntimeRegistries()` with active workers | Panics or data race — always close all workers first |
