# Extensions — `CALL GO` & Custom Controls

> *How to extend ManulEngine (Go) with native Go code.*

ManulEngine (Go) does not use Python decorators, JavaScript plugins, or dynamic module loading. Extensions are **compiled Go functions** registered at process startup via a strongly-typed API. Because they are native Go, they are:

- **Type-safe** — the compiler checks your handler signatures
- **Zero-overhead** — no interpreter, no serialization across language boundaries
- **Goroutine-safe** — register once, invoke from any worker concurrently

There are two extension mechanisms:

1. **`CALL GO`** — invoke a registered Go function by name from a `.hunt` file
2. **`RegisterCustomControl`** — intercept a specific action (e.g., "Click 'React Datepicker'") and handle it entirely in Go, bypassing DOM resolution

---

## `CALL GO`

From a `.hunt` file:

```hunt
@script: {helpers} = mypackage.helpers

STEP 1:
    CALL GO {helpers}.generate_token "admin" into {token}
    CALL GO {helpers}.verify_token {token}
    SET {headers} = Authorization:Bearer {token}
```

The `@script:` directive maps `{helpers}` to `mypackage.helpers`. The parser rewrites `CALL GO {helpers}.generate_token` to `CALL GO mypackage.helpers.generate_token` before execution.

### Writing a Handler

In your Go code (e.g., `main.go` or a plugin package):

```go
package main

import (
    "context"
    "fmt"

    "github.com/alexbeatnik/ManulEngineGo/pkg/runtime"
)

func init() {
    // Register at process init — before any WorkerPool spawns.
    runtime.RegisterGoCall("mypackage.helpers.generate_token", func(
        ctx context.Context,
        invocation runtime.GoCallInvocation,
    ) (any, error) {
        // invocation.Args contains the positional arguments from the DSL.
        // invocation.Variables contains the flattened variable map.
        // invocation.Page gives you direct browser.Page access.
        role := invocation.Args[0]
        token := fmt.Sprintf("tok_%s_%d", role, time.Now().Unix())

        // Return a scalar → bound to {token} via "into {token}"
        return token, nil
    })
}
```

### Handler Signature

```go
type GoCallHandler func(context.Context, GoCallInvocation) (any, error)

type GoCallInvocation struct {
    Name      string            // dotted handler name
    Args      []string          // positional args, already variable-interpolated
    Variables map[string]string // full flattened variable scope
    Page      browser.Page      // direct page access for probes / JS eval
    Command   dsl.Command       // the raw parsed command
}
```

### Return Values

| Return type | Behavior |
|-------------|----------|
| `string` / `int` / scalar | Bound to the variable declared in `into {var}` |
| `map[string]string` | Each key→value pair is written to `LevelRow` scope |
| `map[string]any` | Each key→value pair is written to `LevelRow` scope (values stringified) |
| `error` | Step fails; error message logged |

### Argument Parsing

The DSL parser supports several calling conventions:

```hunt
CALL GO mypackage.helpers.echo "hello world" plain into {message}
CALL GO mypackage.helpers.concat with args: 'a' 'b c' to {joined}
CALL GO mypackage.helpers.noop
```

Arguments are shell-tokenized (respecting single and double quotes) and variable-interpolated at runtime. The `with args:` and `into` / `to` keywords are syntactic sugar recognized by the parser.

---

## Custom Controls

Custom controls intercept a specific **page + target** combination before DOM resolution runs. Use them for complex UI components (date pickers, rich text editors, custom dropdowns) where coordinate-based clicking is insufficient.

### Writing a Control Handler

```go
package main

import (
    "context"

    "github.com/alexbeatnik/ManulEngineGo/pkg/browser"
    "github.com/alexbeatnik/ManulEngineGo/pkg/runtime"
)

func init() {
    runtime.RegisterCustomControl("Checkout Page", "React Datepicker", func(
        ctx context.Context,
        page browser.Page,
        invocation runtime.CustomControlInvocation,
    ) error {
        // invocation.ActionType is "input", "click", "select", etc.
        // invocation.Value is the fill value (e.g., "2026-12-25")
        // invocation.Target is the resolved target label
        // invocation.Variables is the full flattened scope

        js := fmt.Sprintf(`
            (() => {
                const input = document.querySelector('[data-testid="datepicker"]');
                if (!input) throw new Error('datepicker not found');
                input.value = %q;
                input.dispatchEvent(new Event('input', {bubbles: true}));
                input.dispatchEvent(new Event('change', {bubbles: true}));
            })()
        `, invocation.Value)
        _, err := page.EvalJS(ctx, js)
        return err
    })
}
```

### Handler Signature

```go
type CustomControlHandler func(context.Context, browser.Page, CustomControlInvocation) error

type CustomControlInvocation struct {
    Page       string            // current page label (title or URL-derived)
    Target     string            // the target label from the DSL
    ActionType string            // "input" | "click" | "double_click" | "right_click" | "hover" | "select" | "check" | "uncheck" | "upload"
    Value      string            // fill/select value, if any
    Variables  map[string]string // full flattened variable scope
    Command    dsl.Command       // raw parsed command
}
```

### Registration & Lookup Semantics

- Registration is **case-insensitive** after whitespace normalization.
- A wildcard page `"*"` matches any page where a specific page handler is not found.
- Lookups happen at runtime before DOM probing. If a custom control is found, the engine skips the entire probe → score → resolve pipeline.

```go
// Specific page match
RegisterCustomControl("Checkout Page", "React Datepicker", handler)

// Fallback for any page
RegisterCustomControl("*", "React Datepicker", handler)
```

---

## Concurrency & Lifecycle Rules

### Register Before Spawn

All `RegisterGoCall` and `RegisterCustomControl` calls must happen **before** the worker pool starts:

```go
func main() {
    // 1. Register all extensions
    registerAllExtensions()

    // 2. THEN spawn workers
    pool, _ := worker.NewPool(worker.PoolOptions{...})
    results, _ := pool.Run(ctx, hunts)
}
```

The registries are mutex-guarded for safe concurrent reads during execution, but registering while workers are running is strongly discouraged — visibility becomes timing-dependent.

### Handler Safety

Handlers may be invoked by **every worker simultaneously**. They must be safe for concurrent use:

```go
// BAD: shared mutable state without synchronization
var globalCounter int

// GOOD: stateless, or use sync.Map / atomic
func handler(ctx context.Context, invocation runtime.GoCallInvocation) (any, error) {
    // Read-only access to invocation.Variables
    // Local variables only
    return computeResult(invocation.Args), nil
}
```

### Reset in Tests

Use `runtime.ResetRuntimeRegistries()` in tests to clear state between test cases:

```go
func TestSomething(t *testing.T) {
    runtime.ResetRuntimeRegistries()
    t.Cleanup(runtime.ResetRuntimeRegistries)

    runtime.RegisterGoCall("test.double", func(ctx context.Context, inv runtime.GoCallInvocation) (any, error) {
        n, _ := strconv.Atoi(inv.Args[0])
        return n * 2, nil
    })

    // ... run hunt ...
}
```

---

## Variable Scoping

Extensions read and write variables through the runtime's 5-level scope hierarchy:

| Level | Precedence | How to access |
|-------|------------|---------------|
| `LevelRow` | Highest (1) | Set by `EXTRACT`, `CALL GO` map returns, `SET` inside loops |
| `LevelStep` | 2 | Set by `SET` at top-level mission scope |
| `LevelMission` | 3 | Set by `@var:` declarations |
| `LevelGlobal` | 4 | Set by CLI/env/lifecycle hooks |
| `LevelImport` | Lowest (5) | Set by imported hunt variables |

From a handler, `invocation.Variables` gives you the **flattened** view (highest precedence wins). To write back, return a `map[string]string` or `map[string]any` — each key is written to `LevelRow`.

```go
// Writes api_key and endpoint into LevelRow
return map[string]string{
    "api_key":  "secret123",
    "endpoint": "https://api.example.com",
}, nil
```

---

## Full Example: Auth + API Extension

```go
package main

import (
    "context"
    "fmt"
    "net/http"

    "github.com/alexbeatnik/ManulEngineGo/pkg/runtime"
)

func init() {
    runtime.RegisterGoCall("auth.fetch_token", func(ctx context.Context, inv runtime.GoCallInvocation) (any, error) {
        clientID := inv.Args[0]
        secret := inv.Args[1]

        req, _ := http.NewRequestWithContext(ctx, "POST", "https://auth.example.com/token", nil)
        req.SetBasicAuth(clientID, secret)

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()

        var result struct{ Token string }
        json.NewDecoder(resp.Body).Decode(&result)
        return result.Token, nil
    })

    runtime.RegisterGoCall("api.get_user", func(ctx context.Context, inv runtime.GoCallInvocation) (any, error) {
        token := inv.Args[0]
        userID := inv.Args[1]

        req, _ := http.NewRequestWithContext(ctx, "GET",
            fmt.Sprintf("https://api.example.com/users/%s", userID), nil)
        req.Header.Set("Authorization", "Bearer "+token)

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()

        var user map[string]any
        json.NewDecoder(resp.Body).Decode(&user)
        return user, nil // map[string]any → flattened into LevelRow variables
    })
}
```

```hunt
@script: {api} = auth

[SETUP]
    CALL GO {api}.fetch_token "my_client" "my_secret" into {token}
[END SETUP]

STEP 1:
    CALL GO api.get_user {token} "123" into {user}
    PRINT "User email: {user_email}"
    NAVIGATE to https://app.example.com/login
    FILL 'Email' field with '{user_email}'
```

---

## Comparison with Python ManulEngine

| Capability | Python (`CALL PYTHON`) | Go (`CALL GO`) |
|------------|----------------------|----------------|
| Language | Python modules | Go functions |
| Registration | `@custom_control` decorator | `RegisterCustomControl()` / `RegisterGoCall()` |
| Type safety | Runtime duck typing | Compile-time checked |
| Concurrency | GIL-limited | Goroutine-safe by design |
| Arguments | `shlex.split` + placeholder resolution | Shell-tokenized + `ScopedVariables.Interpolate()` |
| Return handling | Dict flattened into variables | `map[string]string` / `map[string]any` → `LevelRow` |
| Aliases | `@script:` | `@script:` (same syntax, Go paths) |

ManulEngine (Go)'s extension model trades dynamic reloading for safety, speed, and parallelism. You compile your extensions into the binary once; they run at native speed forever.
