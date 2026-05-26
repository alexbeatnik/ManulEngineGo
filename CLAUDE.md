# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# Manul Ecosystem Agent Instructions

You are a Principal Go Systems Engineer and QA Architect working on **ManulHeart**, a deterministic, zero-dependency browser automation engine.

## Core Philosophy

1. **Zero External Dependencies:** Only the Go standard library and `gorilla/websocket`. No Playwright, no Selenium, no heavy logging libraries. Do NOT add new third-party deps (including `golang.org/x/sync`); implement equivalents inline.
2. **Deterministic Targeting:** Elements are resolved via the 4-channel heuristic scorer — never CSS/XPath selectors as a public API.
3. **LLM-Free Runtime:** Execution engine is 100% deterministic. LLMs are never used during test execution.
4. **Visual Semantic Logging:** Always use the custom methods in `pkg/utils/logger.go` (`BlockStart`, `ActionPass`, `HeuristicDetail`, etc.) instead of `fmt.Print`.
5. **Alpha Status:** APIs may change. Never imply guarantees of stability or production-readiness.

## Build & Test Commands

```bash
# Build
go build -o manul ./cmd/manul
make build               # same via Makefile

# Run all tests (always with race detector)
go test -race ./...
make test

# Run a single package
go test -race ./pkg/scorer/...

# Run a specific test
go test -race -run TestScorer ./pkg/scorer/...

# Install locally (~/.local/bin/manul)
make install

# Run a hunt file
./manul examples/saucedemo.hunt
./manul examples/saucedemo.hunt --headless --verbose
./manul examples/saucedemo.hunt --explain   # show scorer candidate rankings
./manul examples/ --html-report             # all hunts in dir + aggregate index.html

# Run a single step against a running Chrome
./manul run-step "Click the 'Login' button" --cdp http://127.0.0.1:9222
```

## Skill Navigation (Crucial)

Read the relevant skill file **before** making changes to related systems.

| Area | Skill file |
|------|-----------|
| Concurrency, `pkg/worker`, `pkg/runtime`, `pkg/cdp`, any `go` routine | `.claude/skills/concurrency-rules/SKILL.md` |
| `pkg/scorer`, `pkg/dom`, JS probes (`pkg/heuristics`) | `.claude/skills/scoring-heuristics/SKILL.md` |
| Writing or reviewing `.hunt` files | `.claude/skills/hunt-authoring/SKILL.md` |
| Writing or debugging tests | `.claude/skills/testing-manulheart/SKILL.md` |
| Adding/modifying DSL commands in `pkg/dsl` + `pkg/runtime` | `.claude/skills/adding-dsl-commands/SKILL.md` |
| `RegisterCustomControl` / `RegisterGoCall` extension registries | `.claude/skills/extensions-and-go-calls/SKILL.md` |

## Architecture

```
cmd/manul/          CLI entry point → produces `manul` binary
pkg/dsl/            .hunt parser → Hunt{Commands[]Command}; no browser access
pkg/runtime/        Targeting pipeline + hunt execution (SINGLE-GOROUTINE per worker)
pkg/heuristics/     In-page JS probes (SnapshotProbe, VisibleTextProbe, ExtractDataProbe)
pkg/scorer/         Deterministic stateless 4-channel scoring [0.0–1.0]
pkg/dom/            ElementSnapshot (37 fields): normalized Go structs from probe output
pkg/browser/        Page/Browser interfaces + CDP backend + Chrome lifecycle
pkg/cdp/            Raw CDP WebSocket transport; goroutine-safe Conn
pkg/worker/         Worker / WorkerPool / PortAllocator / RunHuntsInParallel — parallel execution substrate
pkg/explain/        Pure data types: ExecutionResult, HuntResult, ScoreBreakdown
pkg/report/         Per-hunt HTML report + aggregate index.html
pkg/config/         Runtime configuration (20 fields); config.Default() + JSON + env-var loading
pkg/core/           Shared enums (e.g. ScrollStrategy: window vs generic-list containers)
pkg/pages/          URL → human-readable page label registry (lean/wrapped JSON,
                    auto-populate, longest-prefix site match)
pkg/utils/          Dual-output semantic logger + ANSI stripping + error types
examples/           Reference .hunt files
docs/overview.md    Detailed architecture walkthrough
```

### Targeting Pipeline (the engine core)

Every target-based command goes through this pipeline inside `pkg/runtime`:

1. **`CallProbe(SnapshotProbe)`** — single JS TreeWalker pass over the live DOM (Shadow DOM–aware). Collects all signals per candidate in one round-trip. This is the *only* DOM query for targeting.
2. **`deserializeSnapshot()`** — produces `[]dom.ElementSnapshot`.
3. **`scorer.Rank()`** — scores each candidate across 4 channels:

   | Channel | Weight | Signals |
   |---------|--------|---------|
   | text | 0.45 | innerText, aria-label, placeholder, label, data-qa |
   | id | 0.25 | html `id` (with space→dash/underscore variants) |
   | semantic | 0.60 | tag/role alignment with interaction mode, type hint |
   | penalty | ×mult | disabled ×0.0, hidden ×0.1 |
   | proximity | 1.5 (contextual) | Euclidean distance to anchor (NEAR/INSIDE) |

4. **Threshold check** → `ResolvedTarget{Element, Score, RankedCandidates}`.
5. **Action execution** — `click`, `fill`, `select`, `verify`, etc. via `pkg/browser`.

Nothing in `pkg/browser.Page` returns "the element to act on" — that is exclusively `pkg/runtime`'s job.

### Concurrency Contract

`runtime.Runtime` is **single-goroutine by design** (unguarded cache, variable store, checkbox state). Sharing it across goroutines is a data race.

- Use `pkg/worker.Worker` for parallel execution — one Worker owns one Chrome process + Page + Runtime.
- `cdp.Conn` is safe for concurrent use (`writeMu`, `atomic.Int64` IDs, idempotent `Close`).
- `cdp.Conn.Subscribe()` returns `*Subscription`; callers **must** `defer sub.Close()`.
- `RegisterCustomControl` / `RegisterGoCall` are package-global — call at process init, before the pool spawns.
- CI runs `go test -race ./...` on every package. Any new goroutine or shared map must pass the race detector.

### Parallel Execution (Go API)

The CLI is single-threaded; the worker pool is a Go API:

```go
alloc := worker.NewPortAllocator(9222, 9321)
pool, _ := worker.NewPool(worker.PoolOptions{
    Concurrency: 4,
    Config:      config.Default(),
    Allocator:   alloc,
    FailFast:    false,
})
results, firstErr := pool.Run(ctx, hunts)
// results[i] corresponds to hunts[i]
report.GenerateIndex(summaries, "reports")  // aggregate index.html
```

For quick fan-out without `FailFast` or custom `ChromeOptions`, use the convenience
wrapper: `results, err := worker.RunHuntsInParallel(ctx, cfg, hunts, n, logger)` —
returns per-hunt results in input order.

### Configuration Priority Chain (`0.0.1.0`+)

`pkg/config` resolves a 20-field `Config` struct from four sources in strict priority order:

```
CLI Flags  >  MANUL_* env vars  >  manul_engine_configuration.json  >  config.Default()
```

- Always start from `config.Default()` and apply layers on top — never construct a `Config` literal from scratch.
- `MANUL_HEADLESS`, `MANUL_TIMEOUT`, `MANUL_EXPLAIN`, `MANUL_SCREENSHOT` are the primary env overrides.
- If `manul_engine_configuration.json` exists in the working directory it is merged before env vars.

## Loops & Page Registry (`0.0.1.1`+)

ManulHeart matches Python ManulEngine's loop and page-naming semantics:

- **`REPEAT N TIMES:`** — body executes N times; `{i}` is auto-bound as a 0-based counter.
- **`FOR EACH {var} IN {collection}:`** — iterates a comma-separated list stored in a variable.
- **`WHILE <condition>:`** — hard-capped at 100 iterations to prevent infinite loops; predicate uses the same grammar as `IF`.
- Loops nest in any combination with `IF`/`ELIF`/`ELSE`. See `examples/loops_demo.hunt` and `docs/loops-and-pages.md`.

Page labels are resolved through `pkg/pages` (`pages/<safe-netloc>.json` next to the hunt files): `document.title` → longest-prefix site match → exact/regex/substring → `"Domain"` fallback → URL-derived fallback. Unknown URLs are auto-populated as `Auto: domain/path` placeholders. Both lean and wrapped JSON forms are accepted; registry is reloaded per lookup so edits are picked up live.

## Full-Page Scanner (`0.0.1.2`+)

`pkg/scan` exposes two scan modes via `manul scan <URL>`:

- **Basic** (`ScanPage`): flat list of interactive elements → draft `.hunt` in document order. Mirrors ManulEngine's `SCAN_JS`.
- **Full** (`ScanPageFull`, flag `--full`): traverses the DOM with Shadow DOM recursion, groups elements by semantic landmark (`<form>`, `<nav>`, `<main>`, `<dialog>`, ARIA roles). Shadow DOM roots appear as `GroupName [shadow]`. Mirrors ManulEngine's `FULL_SCAN_JS`.

`BuildHuntFull` emits `# ── GroupName ──` comment headers; `Page` group is always first, rest are sorted alphabetically. ARIA roles (`textbox`, `combobox`, `switch`) are mapped to the correct DSL verbs (`Fill`, `Select`, `Check`).

## Visual Parity for Element Highlighting (`0.0.1.0`+)

ManulHeart now matches Python ManulEngine's visual feedback exactly:

- **Normal action flash highlight:** Every resolved action (click, fill, hover, etc.) triggers a
  2-second red border + yellow background flash via `Page.HighlightElement()`. This mirrors
  Python's `_highlight()` method.
- **Debug persistent highlight:** In debug/break mode, a **magenta outline + glow**
  (`outline:4px solid #ff00ff; box-shadow:0 0 15px #ff00ff`) is applied via
  `data-manul-debug-highlight` before the pause prompt and cleared before the action executes.
  This mirrors Python's `_debug_highlight()` / `_clear_debug_highlight()`.
- **Explain-next highlight:** When `explain-next` is requested (extension protocol), the best
  heuristic candidate is highlighted with the same persistent magenta style, replacing any
  previous highlight. This mirrors Python's `ExplainNextDebugger._highlight_match()`.

The CSS payload, attribute name (`data-manul-debug-highlight`), style ID (`manul-debug-style`),
and scroll behavior (`scrollIntoView({behavior:'smooth',block:'center'})`) are all byte-for-byte
identical to the Python implementation.

## Testing Patterns

- **Always run with `-race`:** `go test -race ./...`.
- **Worker tests** use `worker.AdoptWorker` with `MockPage` — canonical pattern for verifying no state bleed between workers (`pkg/worker/worker_test.go`).
- **CDP tests** use an in-process `httptest` WebSocket echo server — see `pkg/cdp/conn_test.go`. New transport features must be tested there first.
- **Scorer tests** are pure unit tests — no browser, no goroutines.
- Do **not** introduce `time.Sleep` in production paths. Use `select { case <-ctx.Done(): ... }` or readiness checks. `time.Sleep` is acceptable in tests only.

## `.hunt` File Rules

- 4-space indentation under `STEP` headers.
- Never hardcode test data — use `@var: {key} = value` and reference via `{key}`.
- Always follow `FILL`/`TYPE` with a `VERIFY ... has value "..."` assertion.
- Action lines are never numbered (numbers only on `STEP` headers).
- Use quoted strings for target labels: `Click the 'Login' button`.

## Doc Sync Rule

When updating a feature's documentation, keep **both** `README.md` and `.github/copilot-instructions.md` in sync. A feature in one but not the other is a documentation bug.

## Token Optimization

This environment has `CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING=1`. Keep responses concise and go straight to implementation.
