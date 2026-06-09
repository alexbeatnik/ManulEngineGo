# ManulHeart

> **Deterministic Web Automation Runtime in Go**

[![Alpha](https://img.shields.io/badge/status-alpha-orange)]() [![Go](https://img.shields.io/badge/go-%3E%3D1.26-blue)]() [![License](https://img.shields.io/badge/license-Apache%202.0-green)]()

ManulHeart executes `.hunt` files using plain-English commands, DOM intelligence, heuristic element resolution, and structured explainability. It connects to system-installed Chrome via the **Chrome DevTools Protocol (CDP)** using pure Go WebSockets.

**No Playwright. No Node.js. No CSS/XPath selectors as a public API. No LLM in the loop.**

Single dependency: `gorilla/websocket`. Pure Go. Single static binary. True goroutine-level parallelism. Embeddable in-process via the `pkg/agent` API.

---

## 📖 Documentation

- **[Overview](docs/overview.md)** — Why ManulHeart exists and how it differs from Playwright/Node.js stacks
- **[Getting Started](docs/getting-started.md)** — Build, install, and run your first hunt
- **[DSL Syntax](docs/dsl-syntax.md)** — Complete `.hunt` language reference
- **[DSL for LLMs](docs/dsl-for-llms.md)** — Compact cheat-sheet + agent JSON shapes (`manul schema` is the machine-readable mirror)
- **[Reports & Explainability](docs/overview.md#explainability)** — Scoring breakdowns and HTML reports
- **[Extensions](docs/extensions.md)** — `CALL GO`, custom controls, and the Go extension API
- **[Loops & Page Objects](docs/loops-and-pages.md)** — `REPEAT`, `FOR EACH`, `WHILE`, and the `pages/` registry

---

## Syntax First

A `.hunt` file is plain English. No selectors. No fragility.

```hunt
@context: E-commerce checkout smoke test
@title: swag-labs-checkout
@var: {username} = standard_user
@var: {password} = secret_sauce
@tags: smoke, checkout

STEP 1: Login
    NAVIGATE to https://www.saucedemo.com/
    FILL 'Username' field with '{username}'
    FILL 'Password' field with '{password}'
    CLICK 'Login' button

STEP 2: Add item to cart
    VERIFY that 'Products' is present
    CLICK 'Add to cart' button NEAR 'Sauce Labs Fleece Jacket'

STEP 3: Complete purchase
    CLICK the 'Shopping cart' link
    CLICK 'Checkout' button
    FILL 'First Name' field with 'Alice'
    FILL 'Last Name' field with 'Smith'
    FILL 'Zip/Postal Code' field with '49000'
    CLICK 'Continue' button
    CLICK 'Finish' button
    VERIFY that 'Thank you for your order!' is present

DONE.
```

The engine resolves `Sauce Labs Fleece Jacket` using a 4-channel deterministic scorer (text, id, semantic, proximity), finds the nearest `Add to cart` button via Euclidean pixel distance blended with DOM ancestry, clicks it, and reports the full candidate ranking if you pass `--explain`.

---

## Quick Start

### 1. Build

```bash
make build
```

Or with Go directly:

```bash
go build -o manul ./cmd/manul
```

### 2. Install

User-local (`~/.local/bin`):

```bash
make install
```

System-wide (`/usr/local/bin`):

```bash
make install-system
```

### 3. Run a hunt

Auto-launches Chrome, executes, cleans up:

```bash
manul examples/saucedemo.hunt
```

Headless:

```bash
manul examples/saucedemo.hunt --headless
```

Connect to an existing Chrome with `--remote-debugging-port=9222`:

```bash
manul examples/saucedemo.hunt --cdp http://127.0.0.1:9222
```

Run every `.hunt` in a directory:

```bash
manul examples/
```

Run a single step against a live browser:

```bash
manul run-step "Click the 'Login' button" --cdp http://127.0.0.1:9222
```

For agent/LLM drivers, `--compact` emits a small, machine-readable `StepOutcome`
(`ok`, `action`, `reason`, `score`, and on failure/low-confidence the top
candidates in `near`) instead of the full scorer breakdown — so the caller can
branch on a typed `reason` and exit code rather than parse error strings:

```bash
manul run-step "Click 'Checkout'" --compact --cdp http://127.0.0.1:9222
```

Read one value off an already-open page without paying for a full scan
(`manul read`). With a human label it resolves the element and returns its
value; with `--selector` it returns the sanitized visible text of that CSS
region:

```bash
manul read "Order total" --cdp http://127.0.0.1:9222
manul read --selector "#answer" --max-chars 2000 --cdp http://127.0.0.1:9222
```

`read` returns `{value, found, reason, near}` — on a miss it carries the typed
`reason` and the top `near` candidates so a driver retargets without a
follow-up scan. `--max-chars` budgets `--selector` region text so a read can't
flood a prompt.

For an LLM that needs the structure of the open page, `manul map` emits a
compact, landmark-grouped JSON map (label + role only, deduped, per-group
capped) — the cheap, prompt-ready alternative to a full `scan` draft. And
`manul schema` emits the engine's self-describing contract (DSL verbs + agent
JSON shapes + failure-reason enum) so a consumer pins it instead of stuffing
full prose docs into every prompt:

```bash
manul map --cdp http://127.0.0.1:9222 --max-per-group 8
manul schema   # → docs/dsl-for-llms.md is the human mirror of this contract
```

These (`read`, `run-step --compact`, `run`, `map`, `schema`) are the CLI face of
the embeddable `pkg/agent` API (`agent.Session` — `Read` / `ReadText` / `Step` /
`Run` / `Map`); the CLI and in-process consumers share one code path.

Pipe a hunt script from stdin — useful for one-off scripts, editor integrations, and CI generators that build hunts on the fly:

```bash
cat examples/saucedemo.hunt | manul -
echo 'STEP 1. Open the page
    NAVIGATE to "https://example.com"' | manul - --headless
```

When `target` is `-`, ManulHeart parses stdin as a single hunt (source path `<stdin>`), resolves `@import:` relative to the current working directory, and emits a partial `*HuntResult` even if a step fails so downstream tools (e.g. the OS-Manul dispatcher) can read per-step errors instead of being limited to `exit 1`.

### 4. Scan a page to generate a draft hunt

Basic scan — discovers interactive elements and writes a flat draft `.hunt`:

```bash
manul scan https://example.com
manul scan https://example.com --output tests/draft.hunt --headless
```

Full-page scan — groups elements by semantic region (forms, navigation, main content, Shadow DOM roots) and produces an annotated draft:

```bash
manul scan https://example.com --full
manul scan https://example.com --full --output tests/draft.hunt --headless
```

The `--full` mode mirrors ManulEngine's `FULL_SCAN_JS` — it traverses the entire DOM including Shadow DOM, resolves landmark containers (`<form>`, `<nav>`, `<main>`, `<dialog>`, ARIA roles), and groups elements under human-readable section headers so the draft is easier to read and trim.

---

## Philosophy

### Zero Dependencies (Pure CDP)

ManulHeart speaks to Chrome directly over a WebSocket. There is no Playwright, no Selenium, no Node.js runtime, and no heavy dependency tree. The only external package is `gorilla/websocket` for the transport. Everything else — the CDP protocol, the heuristic probes, the scorer, the DSL parser — is pure Go standard library.

### True Concurrency (Goroutines)

Because there is no GIL and no single-threaded browser driver process, ManulHeart can run dozens of hunts in parallel using native goroutines. The `pkg/worker` package provides a `WorkerPool` with per-worker Chrome isolation, `PortAllocator` for debug-port management, and race-detector-safe CDP transport. Each worker owns its own `Runtime`, `Page`, and `ChromeProcess`.

### Determinism

The same `.hunt` file against the same page produces the same resolution path every time. Element targeting is a deterministic pipeline: one JS probe → 37-field `ElementSnapshot` → 4-channel scorer → threshold check → action. No randomness, no LLM fallback, no "ask the model to guess."

---

## Key Features

| Feature | What it means |
|---------|--------------|
| **DSL-first automation** | `.hunt` files are the primary artifact. Plain English commands, not selectors. |
| **Deterministic targeting** | 4-channel heuristic scorer ranks every candidate with explicit signal breakdowns. |
| **Contextual qualifiers** | `NEAR`, `ON HEADER/FOOTER`, `INSIDE` restrict candidates spatially and structurally. |
| **Control flow** | `IF`/`ELIF`/`ELSE`, `WHILE`, `REPEAT N TIMES`, `FOR EACH` with full block nesting. `WHILE` is capped at 100 iterations; `REPEAT` exposes a `{i}` 0-based counter. |
| **Page-name registry** | `pages/<site>.json` maps URLs to human-readable labels for reports and `RegisterCustomControl`. Unknown URLs are auto-populated as `Auto: domain/path` placeholders. See [docs/loops-and-pages.md](docs/loops-and-pages.md). |
| **Hooks** | `[SETUP]` and `[TEARDOWN]` blocks run before and after the mission body. Teardown always executes. |
| **Script aliases** | `@script: {alias} = dotted.go.path` lets you alias `CALL GO` handlers. |
| **Parallel execution** | Native `WorkerPool` runs hunts concurrently with microscopic memory overhead. |
| **Explainability** | `--explain` prints top-5 candidate rankings with per-channel score breakdowns. |
| **Interactive debugger** | `--debug` pauses before every step; `PAUSE` command; breakpoint lines; browser modal UI. |
| **Extension API** | `CALL GO` invokes registered Go functions. `RegisterCustomControl` intercepts actions by page+target. |
| **HTML reports** | Per-hunt styled reports + aggregate `index.html` for parallel runs. |
| **Page scanner** | `manul scan <URL>` generates a draft `.hunt`; `--full` groups elements by semantic region (form, nav, main, shadow) including Shadow DOM. |
| **Stdin hunts** | `manul -` reads a hunt script from stdin and always emits a partial result on failure so dispatchers can read per-step errors. |
| **Embeddable agent API** | `pkg/agent.Session` owns Chrome and exposes compact, agent-friendly calls: `Read` (zero-scan), `ReadText`, `Step`/`Run` (typed `Reason` + `Near` candidates), `Map` (budgeted scan). CLI `read` / `run-step --compact` are thin wrappers over it. |
| **LLM contract commands** | `manul map` emits a compact landmark-grouped page map; `manul schema` emits the DSL grammar + agent JSON shapes as a token-lean, version-stamped contract. See [docs/dsl-for-llms.md](docs/dsl-for-llms.md). |
| **Zero external deps** | Only `gorilla/websocket`. No Playwright, no Node.js, no Python. |

---

## CLI & Configuration

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--cdp` | *(auto-launch)* | Connect to existing Chrome (skip auto-launch) |
| `--user-data-dir` | `/tmp/manulheart-chrome` | Chrome profile directory |
| `--headless` | `false` | Run Chrome without a visible window |
| `--verbose` | `false` | Enable debug-level logging |
| `--json` | `false` | Print JSON execution result to stdout |
| `--timeout` | `30s` | Default per-command timeout |
| `--debug` | `false` | Interactive step-by-step execution with breakpoints |
| `--explain` | `false` | Show targeting candidate rankings and scores |
| `--screenshot` | `none` | Capture screenshots: `none`, `on-fail`, `always` |
| `--html-report` | `false` | Generate a styled HTML report |
| `--tags` | *(none)* | Comma-separated tag filter (match `@tags:` in hunt files) |
| `--retries` | `0` | Retry failed steps N times |
| `--disable-cache` | `false` | Disable DOM snapshot caching |
| `--workers` | `1` | Number of parallel workers for directory runs |

### Environment Variables

ManulHeart respects `MANUL_*` prefix environment variables (CLI flags always take precedence):

| Variable | Type | Description |
|----------|------|-------------|
| `MANUL_HEADLESS` | `bool` | Run Chrome in headless mode |
| `MANUL_TIMEOUT` | `duration` | Default per-command timeout |
| `MANUL_EXPLAIN` | `bool` | Enable explain mode |
| `MANUL_SCREENSHOT` | `string` | Screenshot mode: `none`, `on-fail`, `always` |

### Configuration Priority

```
CLI Flags  >  MANUL_* env vars  >  manul_engine_configuration.json  >  config.Default()
```

---

## Architecture

```
cmd/manul           CLI entry point → produces `manul` binary
pkg/agent           Embedding facade: agent.Session over runtime/cdp/scorer —
                    Launch/Attach (owns Chrome), Read/ReadText/Step/Run/Map
pkg/cdp             Low-level CDP WebSocket transport and domain wrappers
pkg/browser         Abstract browser/page interfaces + CDP backend + Chrome lifecycle
pkg/runtime         Targeting pipeline: probe → filter → score → resolve;
                    DSL execution, control flow, variable management
pkg/worker          Worker / WorkerPool / PortAllocator for parallel execution
pkg/dom             Normalized DOM element model (ElementSnapshot with 37 fields)
pkg/heuristics      In-page JS probes (SnapshotProbe, VisibleTextProbe, ExtractDataProbe)
pkg/scorer          Deterministic 4-channel [0.0–1.0] scoring and ranking
pkg/dsl             .hunt file parser, import resolver, command AST with block nesting
pkg/explain         Structured execution results and explainability types
pkg/report          Styled HTML report generation + aggregate index.html
pkg/config          Runtime configuration (20 fields)
pkg/core            Shared enums (e.g. ScrollStrategy)
pkg/pages           Page-name registry: URL → human-readable label, lean/wrapped JSON,
                    auto-populate, longest-prefix site matching
pkg/scan            DOM scanner → draft .hunt; ScanPage (flat) + ScanPageFull (grouped)
pkg/utils           Semantic logging (Block/Action/Detail), ANSI stripping, error types
examples/           Sample .hunt files
docs/               Documentation
```

See [docs/overview.md](docs/overview.md) for the deep-dive architecture walkthrough.

---

## Embedding (Agent API)

To drive a browser from a Go program — an assistant, an agent, a custom tool —
embed `pkg/agent`. ManulHeart owns the entire browser lifecycle; the consumer
just calls a small, compact API and never touches CDP or the runtime directly:

```go
import "github.com/alexbeatnik/ManulHeart/pkg/agent"

sess, err := agent.Launch(ctx, agent.Options{Headless: true}) // ManulHeart spawns & owns Chrome
// or: agent.Attach(ctx, "http://127.0.0.1:9222", "", agent.Options{}) to use a running Chrome
defer sess.Close()

out, _ := sess.Step(ctx, "Click the 'Login' button")  // compact: ok, reason, score, near[]
total, _ := sess.Read(ctx, "Order total")             // zero-scan targeted text
text, _ := sess.ReadText(ctx, "#answer")              // sanitized region/page text
pm, _ := sess.Map(ctx, agent.MapBudget{MaxPerGroup: 8}) // budgeted landmark map
res, _ := sess.Run(ctx, huntScript)                   // whole .hunt, compact aggregate
```

Failures carry a machine-readable `Reason` (`not_found` / `ambiguous` /
`timeout` / `verify_failed` / `action_failed`) and, for resolution problems,
the top candidates in `Near` — so a caller branches on a typed reason instead
of parsing error strings, and corrects a weak match without a follow-up scan.
A `Session` owns one single-goroutine `Runtime`; use one Session per goroutine
for parallel work (or `pkg/worker` below). The CLI's `read` and
`run-step --compact` are thin wrappers over this same code path.

## Parallel Execution (Go API)

The `manul` CLI runs single-threaded by default. For true parallelism, embed the worker pool directly:

```go
import (
    "context"
    "github.com/alexbeatnik/ManulHeart/pkg/config"
    "github.com/alexbeatnik/ManulHeart/pkg/dsl"
    "github.com/alexbeatnik/ManulHeart/pkg/report"
    "github.com/alexbeatnik/ManulHeart/pkg/worker"
)

func runSuite(ctx context.Context, hunts []*dsl.Hunt) error {
    cfg := config.Default()
    results, err := worker.RunHuntsInParallel(ctx, cfg, hunts, 4, logger)
    if err != nil {
        return err
    }

    summaries := make([]report.RunSummary, len(results))
    for i, r := range results {
        summaries[i] = report.RunSummary{Result: r.Result, WorkerID: r.WorkerID}
    }
    _, _ = report.GenerateIndex(summaries, "reports")
    return nil
}
```

**Concurrency contract:** One `Runtime`, `Page`, and `ChromeProcess` per worker. Sharing them across goroutines is a data race — verified by `go test -race` in CI. Register all `CALL GO` handlers and custom controls **before** spawning the pool.

---

## Development Guides

- [**Scoring & Heuristics**](.claude/skills/scoring-heuristics/SKILL.md)
- [**Concurrency Rules**](.claude/skills/concurrency-rules/SKILL.md)
- [**Adding DSL Commands**](.claude/skills/adding-dsl-commands/SKILL.md)
- [**Go Calls & Extensions**](.claude/skills/extensions-and-go-calls/SKILL.md)
- [**Testing ManulHeart**](.claude/skills/testing-manulheart/SKILL.md)
- [**Hunt Authoring**](.claude/skills/hunt-authoring/SKILL.md)

---

## Project Status

**Alpha.** The core engine covers:

- 32+ DSL commands, full control flow (IF/ELIF/ELSE, WHILE, REPEAT N TIMES, FOR EACH)
- Page-name registry (`pages/<site>.json`) with auto-populate and longest-prefix matching
- `[SETUP]` / `[TEARDOWN]` hook blocks with fail-fast setup and guaranteed teardown
- `@script:` aliases for `CALL GO` handler paths
- Import system (`@import:`, `USE`/`CALL` expansion)
- 4-channel deterministic scoring with contextual qualifiers (NEAR, ON HEADER/FOOTER, INSIDE)
- Shadow DOM support, 3-pass proximity resolution, anti-phantom guards
- HTML reporting, screenshots, debug mode, explain mode
- Native `WorkerPool` for parallel execution with per-worker Chrome isolation
- Embeddable `pkg/agent` API (`Read`/`ReadText`/`Step`/`Run`/`Map`) with typed failure reasons
- Strongly-typed extension API (`CALL GO`, `RegisterCustomControl`)
- Race-detector-safe CDP transport and concurrent handler registries

**Version:** `v0.0.6` — one semver scheme everywhere: the git module tag, `manul --version`, and `go get github.com/alexbeatnik/ManulHeart@v0.0.6` all agree.

**Recommended install target:** expose the binary as a PATH command named `manul` for editor extensions and automation tooling.

> **VS Code Extension:** The [Manul Engine Extension](https://marketplace.visualstudio.com/items?itemName=manul-engine.manul-engine) supports ManulHeart out of the box. Open a workspace containing `go.mod` and the extension auto-detects the Go runtime, surfaces `CALL GO` snippets, and validates `CALL GO` steps inside hook blocks.

---

## License

Apache 2.0
