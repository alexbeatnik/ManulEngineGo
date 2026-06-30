---
name: configuration-system
description: Add or modify runtime configuration fields in ManulEngine (Go). Use when introducing a new tunable engine parameter, wiring CLI flags, or env-var overrides. Covers the 3-layer priority chain and pointer-based JSON overlay.
---

# Configuration System

ManulEngine (Go)'s configuration is resolved from four sources in **strict priority order**:

```
CLI Flags  >  MANUL_* env vars  >  manul_engine_configuration.json  >  config.Default()
```

All code that needs config receives a `config.Config` struct. Never construct a `Config` literal from scratch — always start from `config.Default()` and apply layers.

## Architecture

```
config.Default()
      │
      ▼
applyJSONFile()  ← manul_engine_configuration.json (if present in CWD)
      │
      ▼
overrideFromEnv()  ← MANUL_* environment variables
      │
      ▼
CLI flag overrides  ← cmd/manul/main.go
```

## The `Config` struct

Defined in [`pkg/config/config.go`](../../../pkg/config/config.go). Key fields:

| Field | Type | Default | Env var | JSON key |
|-------|------|---------|---------|----------|
| `CDPEndpoint` | `string` | `""` | `MANUL_CDP_ENDPOINT` | `cdp_endpoint` |
| `Browser` | `string` | `"chromium"` | `MANUL_BROWSER` | `browser` |
| `Headless` | `bool` | `false` | `MANUL_HEADLESS` | `headless` |
| `Verbose` | `bool` | `false` | `MANUL_VERBOSE` | `verbose` |
| `DebugMode` | `bool` | `false` | `MANUL_DEBUG` | `debug_mode` |
| `ExplainMode` | `bool` | `false` | `MANUL_EXPLAIN` | `explain_mode` |
| `DefaultTimeout` | `time.Duration` | `5000ms` | `MANUL_TIMEOUT` | `timeout` (ms int) |
| `NavTimeout` | `time.Duration` | `30000ms` | `MANUL_NAV_TIMEOUT` | `nav_timeout` (ms int) |
| `Screenshot` | `string` | `"on-fail"` | `MANUL_SCREENSHOT` | `screenshot` |
| `HTMLReport` | `bool` | `false` | `MANUL_HTML_REPORT` | `html_report` |
| `Retries` | `int` | `0` | `MANUL_RETRIES` | `retries` |
| `VerifyMaxRetries` | `int` | `15` | `MANUL_VERIFY_MAX_RETRIES` | `verify_max_retries` |
| `Workers` | `int` | `1` | `MANUL_WORKERS` | `workers` |
| `DisableCache` | `bool` | `false` | `MANUL_DISABLE_CACHE` | `disable_cache` |
| `Tags` | `[]string` | `[]` | `MANUL_TAGS` | `tags` |
| `TestsHome` | `string` | `"tests"` | `MANUL_TESTS_HOME` | `tests_home` |
| `AutoAnnotate` | `bool` | `false` | `MANUL_AUTO_ANNOTATE` | `auto_annotate` |
| `BrowserArgs` | `[]string` | `nil` | `MANUL_BROWSER_ARGS` | `browser_args` |
| `ExecutablePath` | `*string` | `nil` | `MANUL_EXECUTABLE_PATH` | `executable_path` |
| `BreakLines` | `[]int` | `nil` | — | `break_lines` |

## Adding a new config field — the checklist

### 1. Update `Config` struct

Add the field to the `Config` struct in `pkg/config/config.go` with a `json` tag.

### 2. Update `jsonConfig` intermediate struct

**Critical:** For `bool`, `int`, and `time.Duration` fields, use **pointer types** in `jsonConfig`:

```go
type jsonConfig struct {
    // ... existing fields ...
    MyFeature    *bool   `json:"my_feature"`      // pointer!
    MyThreshold  *int    `json:"my_threshold"`    // pointer!
}
```

Why pointers? `jsonConfig` is unmarshaled from JSON. A `nil` pointer means "field absent — don't override default". A non-nil pointer means "explicitly set, even to `false` or `0`". Without pointers, `false` in JSON is indistinguishable from a missing field.

### 3. Wire `applyJSONFile` overlay

In `applyJSONFile`, add a guarded assignment:

```go
if raw.MyFeature != nil {
    cfg.MyFeature = *raw.MyFeature
}
if raw.MyThreshold != nil {
    cfg.MyThreshold = *raw.MyThreshold
}
```

### 4. Wire env-var override

In `overrideFromEnv`, add:

```go
if v := os.Getenv("MANUL_MY_FEATURE"); v != "" {
    if b, err := strconv.ParseBool(v); err == nil {
        cfg.MyFeature = b
    }
}
if v := os.Getenv("MANUL_MY_THRESHOLD"); v != "" {
    if n, err := strconv.Atoi(v); err == nil {
        cfg.MyThreshold = n
    }
}
```

**Naming convention:** `MANUL_SNAKE_CASE` matching the JSON key.

### 5. Wire CLI flag (if applicable)

In `cmd/manul/main.go`, add a flag and apply it after `config.Load()`:

```go
cfg := config.Default() // or config.Load() if you want JSON/env
flag.BoolVar(&cfg.MyFeature, "my-feature", cfg.MyFeature, "Enable my feature")
flag.Parse()
```

CLI flags have highest priority and are applied last.

### 6. Duration fields — special JSON format

Durations in JSON are **integer milliseconds** (not Go duration strings):

```json
{
  "timeout": 10000,
  "nav_timeout": 30000
}
```

Env vars accept both milliseconds and `time.ParseDuration` formats:
- `MANUL_TIMEOUT=5000` → 5 seconds
- `MANUL_TIMEOUT=5s` → 5 seconds

## String-slice env vars

Comma-separated values are split by `splitCSV`:
- `MANUL_BROWSER_ARGS="--disable-gpu,--no-sandbox"`
- `MANUL_TAGS="smoke,login"`

## Common pitfalls

| Mistake | Consequence |
|---------|-------------|
| Non-pointer `bool`/`int` in `jsonConfig` | `"my_feature": false` in JSON silently ignored — can't distinguish from missing field |
| Forgetting env-var wiring | Feature works via JSON but not via `MANUL_*` — inconsistent UX |
| Using `time.Duration` directly in JSON | `json.Unmarshal` fails on `"5s"` because we use `*int` for milliseconds |
| Constructing `Config{}` literal instead of `Default()` | Zero values override sensible defaults (e.g. `Timeout: 0`) |
| Env var name not matching `MANUL_SNAKE_CASE` | Users can't discover it; breaks convention |

## Key files

- [`pkg/config/config.go`](../../../pkg/config/config.go) — `Config`, `jsonConfig`, `Load()`, `applyJSONFile()`, `overrideFromEnv()`
- [`cmd/manul/main.go`](../../../cmd/manul/main.go) — CLI flag definitions
