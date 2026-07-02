<p align="center">
    <img src="images/manul.png" alt="ManulEngine (Go) mascot" width="180" />
</p>

# ManulEngine (Go)

[![Status: Alpha](https://img.shields.io/badge/status-alpha-d97706)](#project-status)
[![Go](https://img.shields.io/badge/go-%3E%3D1.26-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-green)](LICENSE)
[![Manul Engine Extension](https://img.shields.io/visual-studio-marketplace/v/manul-engine.manul-engine?label=VS%20Code%20Extension&logo=visualstudiocode)](https://marketplace.visualstudio.com/items?itemName=manul-engine.manul-engine)

**A deterministic automation runtime for both humans and LLM agents — the Go implementation of ManulEngine.** Write (or generate) `.hunt` files in plain English; ManulEngine resolves every element with deterministic DOM heuristics and drives Chrome directly over the Chrome DevTools Protocol (CDP) — no Playwright, no selectors, no cloud APIs, no AI required.

The same runtime serves two drivers from one artifact:

- **Humans** author readable `.hunt` steps (`Click the 'Login' button`) — QA tests, RPA, synthetic monitors. No selectors to maintain.
- **LLM agents** drive it through JSON CLI commands (`manul map` / `run-step` / `read` / `schema`) that target elements by human label, never CSS/XPath.

This is the Go sibling of [ManulEngine (Python)](https://github.com/alexbeatnik/ManulEngine): the same `.hunt` DSL, the same deterministic scorer, and the same agent JSON shapes — built on a single dependency (`gorilla/websocket`), shipping as one static binary, with true goroutine-level parallelism and an embeddable in-process API (`pkg/agent`).

### Built for agents — and it's measurably cheaper on tokens

An agent has to *see* a page before it can act. A browser driver like Playwright or Selenium doesn't help here — it gives *code* access to the page, not the model. An LLM agent built on one still has to serialize the page into the prompt, and the usual ways are raw HTML or the accessibility snapshot — both expensive. `manul map` instead emits a compact, landmark-grouped view of just the labelled, interactive elements. Measured with the GPT-4 tokenizer (`cl100k_base`) on representative pages — the same compact `manul map` JSON both runtimes emit:

| What an agent feeds the model to perceive a page | Tokens | vs `manul map` |
| --- | --- | --- |
| Raw HTML (page source) | 2,216 – 2,241 | **4–8× more** |
| Accessibility tree (role + name) | 1,384 – 1,912 | **3.6–5× more** |
| **`manul map` (compact JSON)** | **278 – 528** | **1×** |

So the perception step that every browser-agent pays on *every* turn costs **~4–8× fewer tokens** than dumping HTML, and **~3.6–5× fewer** than the a11y tree. These are clean synthetic pages — real-world HTML is far more bloated, so the gap widens in practice.

Authoring is also leaner and far more durable: the same flow written as a `.hunt` file carries **zero CSS/XPath selectors** to break when the markup shifts.

> **Status: Alpha.** Solo-developed, actively battle-tested. Bugs are expected, APIs may evolve, and there are no promises about stability or production readiness. The core claim is transparency: when a step works, you can see exactly why; when it fails, you get the scoring breakdown to diagnose it.

> **📖 Full Documentation:** [Overview](docs/overview.md) · [Installation](docs/installation.md) · [Getting Started](docs/getting-started.md) · [DSL Syntax](docs/dsl-syntax.md) · [DSL for LLMs](docs/dsl-for-llms.md) · [Reports & Explainability](docs/reports.md) · [Integration](docs/integration.md) · [Extensions](docs/extensions.md) · [Loops & Page Objects](docs/loops-and-pages.md)

---

## Syntax First

ManulEngine runs `.hunt` files — plain-English automation scripts that read like manual QA steps. Here is the DSL in action.

### A complete flow

```text
@context: Smoke test for a login page
@title: Login Smoke
@var: {email} = admin@example.com
@var: {password} = secret123

STEP 1: Open the app
    NAVIGATE to https://example.com/login
    VERIFY that 'Sign In' is present

STEP 2: Authenticate
    FILL 'Email' field with '{email}'
    VERIFY "Email" field has value "{email}"
    FILL 'Password' field with '{password}'
    CLICK the 'Sign In' button
    VERIFY that 'Dashboard' is present

DONE.
```

Run it:

```bash
manul path/to/login.hunt
```

Every `@var:` is declared up front — never hardcode test data inside steps. `VERIFY` confirms state after every significant action. `DONE.` closes the mission.

> **Case-insensitive keywords.** All DSL keywords are case-insensitive at runtime — `CLICK`, `Click`, and `click` all work. The canonical form used in documentation and generated files is ALL UPPERCASE.
>
> **Element type hints are optional.** Words like `button`, `link`, `field`, `dropdown` placed after the target outside quotes are not required, but they provide a strong heuristic signal that boosts scoring accuracy. `CLICK the 'Login' button` and `CLICK the 'Login'` both work — the former is more precise.

### Conditional branching

Branch test logic with `IF` / `ELIF` / `ELSE` based on what the page actually contains. Nesting is supported.

```text
STEP 1: Adaptive login
    IF button 'SSO Login' exists:
        CLICK the 'SSO Login' button
        VERIFY that 'SSO Portal' is present
    ELIF text 'Sign In' is present:
        FILL 'Username' field with '{username}'
        CLICK the 'Sign In' button
    ELSE:
        CLICK the 'Create Account' link
```

Conditions can check element existence, visible text, variable equality, substring containment, or simple truthiness — all evaluated against the live page.

### Loops

Repeat actions with `REPEAT`, iterate data with `FOR EACH`, or poll dynamic state with `WHILE`. Loops nest freely with conditionals.

```text
@var: {products} = Laptop, Headphones, Mouse

STEP 1: Add products to cart
    FOR EACH {product} IN {products}:
        FILL 'Search' field with '{product}'
        PRESS Enter
        CLICK the 'Add to Cart' button NEAR '{product}'
        VERIFY that 'Added to cart' is present

STEP 2: Load all reviews
    WHILE button 'Load More' exists:
        CLICK the 'Load More' button
        WAIT 2

STEP 3: Retry checkout
    REPEAT 3 TIMES:
        CLICK the 'Place Order' button
        IF text 'Success' is present:
            VERIFY that 'Order confirmed' is present
```

`REPEAT N TIMES:` runs a fixed count. `FOR EACH {var} IN {collection}:` iterates comma-separated values. `WHILE <condition>:` repeats until the condition is false (safety limit: 100 iterations). `{i}` counter is auto-set on every iteration.

### Contextual navigation

When a page has repeating controls — multiple "Delete" buttons, "Edit" links in every row — use contextual qualifiers instead of brittle selectors.

```text
CLICK the 'Edit' button NEAR 'John Doe'
CLICK the 'Login' button ON HEADER
CLICK the 'Privacy Policy' link ON FOOTER
CLICK the 'Delete' button INSIDE 'Actions' row with 'John Doe'
```

`NEAR` ranks by pixel distance. `ON HEADER` / `ON FOOTER` scopes to viewport zones. `INSIDE` restricts scoring to a resolved row or container subtree.

### Variables, data-driven runs, and backend hooks

```text
@var: {email} = admin@example.com
@var: {password} = secret123
@script: {db} = db.Helpers
@data: users.csv
@tags: smoke, auth

[SETUP]
    CALL GO {db}.SeedUser "{email}" "{password}"
[END SETUP]

STEP 1: Login
    NAVIGATE to https://example.com/login
    FILL 'Email' field with '{email}'
    FILL 'Password' field with '{password}'
    CLICK the 'Sign In' button
    VERIFY that 'Dashboard' is present

STEP 2: Fetch and use an OTP
    CLICK the 'Send OTP' button
    CALL GO api.FetchOTP "{email}" into {otp}
    FILL 'OTP' field with '{otp}'
    CLICK the 'Verify' button
    VERIFY that 'Welcome' is present

[TEARDOWN]
    CALL GO {db}.CleanDatabase "{email}"
[END TEARDOWN]
```

`@data:` loops the entire mission over each row in a CSV or JSON file. `@tags:` lets you filter runs with `manul --tags smoke`. `[SETUP]` / `[TEARDOWN]` run Go outside the browser for data seeding and cleanup. `CALL GO ... into {var}` captures return values mid-test — ideal for OTPs, tokens, and backend state. `CALL GO` invokes Go functions registered in-process via `RegisterGoCall` (see [Embedding](#embedding-go-api)); `@script:` aliases a handler path.

### Explicit waits and strict assertions

```text
WAIT FOR 'Submit' to be visible
WAIT FOR 'Loading...' to disappear
WAIT FOR RESPONSE "/api/checkout"

VERIFY "Email" field has value "{email}"
VERIFY "Save" button has text "Save Changes"
VERIFY "Search" input has placeholder "Type to search..."
```

Explicit waits poll element visibility over CDP instead of hardcoded sleeps. `WAIT FOR RESPONSE` blocks until a matching network response arrives. Strict assertions resolve the element through heuristics and compare exact text, value, or placeholder with `==`.

### Page scanning for LLM agents

A full-page scan groups every interactive control on the page by its nearest semantic landmark (form, nav, header, dialog, section …) including Shadow DOM, and writes an annotated draft `.hunt`:

```bash
manul scan https://example.com --full
```

Designed for LLM-driven authoring — the grouped output is easy for a model (or a human) to read and trim into a real hunt. For live agentic perception, prefer `manul map` (below), which emits the same structure as token-lean JSON without writing a file.

### Agent commands — drive the engine from an external LLM

For agentic use, ManulEngine exposes a small set of **JSON-emitting CLI commands** (the CLI face of the embeddable [`pkg/agent`](#embedding-go-api) API). They attach to an **already-running Chrome over CDP**, so an external model keeps one browser open and issues stateless calls against it. The JSON payload goes to **stdout**; all engine logs go to **stderr**, so a driver can pipe the output straight into a prompt.

```bash
# 1. start Chrome once with remote debugging
google-chrome --remote-debugging-port=9222 &

# 2. let the model see the page, act, and read — by human label, never CSS/XPath
manul schema                                       # DSL grammar + agent JSON shapes (no browser)
manul map --cdp http://127.0.0.1:9222              # compact landmark-grouped page map → JSON
manul run-step "Click the 'Login' button" --compact # run one instruction → step-outcome JSON
manul read 'Order total' --cdp http://127.0.0.1:9222 # read one labelled value → {value, found, reason}
manul read --selector '#cart' --cdp ...            # sanitized region text → {text, selector}
```

Shared flags: `--cdp <url>` (default `http://127.0.0.1:9222`) and `--tab <url-substr>` to pick a tab. `run-step --compact` returns a non-zero exit code when the step fails, and surfaces `near` candidates (with `0.0–1.0` scores) on a failed or low-confidence match so the agent can retarget without a re-scan. `manul schema` is the machine-readable contract a driver pins instead of stuffing full docs into every prompt.

### Shared libraries and scheduling

```text
@import: Login, Logout from lib/auth.hunt
@export: Checkout
@schedule: every 5 minutes

STEP 1: Setup
    USE Login

STEP 2: Purchase flow
    CLICK the 'Buy Now' button
    VERIFY that 'Order Confirmed' is present

STEP 3: Cleanup
    USE Logout

DONE.
```

`@import:` / `USE` reuses named STEP blocks across files. `@export:` controls visibility. `@schedule:` plus `manul daemon` turns any hunt into a recurring monitor or RPA job.

### CLI

```bash
manul path/to/hunts/                             # run all .hunt files in a directory
manul --headless path/to/file.hunt               # headless single file
manul --tags smoke path/to/hunts/                # filter by tags
manul --html-report --screenshot on-fail path/   # reports + failure screenshots
manul --explain path/to/file.hunt                # per-step scoring breakdown
manul --debug path/to/file.hunt                  # pause before every step
manul --workers 4 path/to/hunts/                 # run files in parallel
manul scan https://example.com                   # scan a page → draft.hunt
manul daemon path/to/hunts/ --headless           # run scheduled hunts
```

---

## Philosophy

### Determinism, not prompt variance

The primary resolver is not an LLM. It is a weighted heuristic scorer (`pkg/scorer`) backed by an in-page JavaScript probe. Scores are normalized on a `0.0–1.0` confidence scale across four channels: `text`, `id`, `semantics`, and `proximity` (with an in-session cache signal). The result is repeatable: same page state plus same step text equals same resolution — no prompt variance, no cloud dependency.

When `--explain` is enabled, every resolved step prints the top-5 candidates with a per-channel breakdown so you can see exactly which signals drove the decision and which lost.

### Native CDP, no Playwright

ManulEngine talks to the browser through its **own** Chrome DevTools Protocol client — a thin WebSocket transport in [`pkg/cdp`](pkg/cdp) with a single external dependency (`gorilla/websocket`). There is no Playwright, no Selenium, no Node.js, and no bundled browser download: the engine launches the Chrome/Chromium you already have on `PATH` (or attaches to a running one) and drives it directly.

Why own the protocol layer:

- **One small dependency, fully inspectable.** The whole browser driver is a handful of readable Go packages (`pkg/cdp` transport, `pkg/browser` launcher, `pkg/dom` element model) rather than a large vendored toolchain. What the engine sends to Chrome is exactly what you can read.
- **Trusted input by default.** Clicks and keystrokes are dispatched via CDP `Input.*` events at real coordinates, and form values go through the native value setter so React/Vue/Angular state updates fire — no `force` hacks needed. `PRESS Enter` sends a real character event (`text:"\r"`) so forms actually submit.
- **Per-frame execution contexts.** A selector is resolved once inside the owning frame's execution context, then every operation runs against that handle — so same-origin iframes (and OOPIF child targets) are first-class, not an afterthought.
- **CDP is Chromium-only by design.** Because the protocol is Chrome's, the engine drives Chrome/Chromium only. Firefox/WebKit are intentionally not supported; pick the concrete binary with `--channel` (`chrome`, `msedge`, `chromium`, …) or `--executable-path`.

### True concurrency (goroutines)

Because there is no GIL and no single-threaded driver process, ManulEngine runs dozens of hunts in parallel using native goroutines. The `pkg/worker` package provides a `WorkerPool` with per-worker Chrome isolation, a `PortAllocator` for debug-port management, and race-detector-safe CDP transport — each worker owns its own `Runtime`, `Page`, and `ChromeProcess`. Run directory suites concurrently with `--workers N`, or embed the pool directly (see [Parallel Execution](#parallel-execution-go-api)).

### Dual-persona workflow

Manual QA writes plain-English `.hunt` steps — no code required. SDETs extend the same files with Go hooks (`[SETUP]` / `[TEARDOWN]`, `CALL GO`) and `RegisterCustomControl` handlers for complex widgets. Both personas work on the same artifact; the Go extensions are registered in-process before a run.

### No AI in the loop — fully deterministic

ManulEngine has **no LLM inside it**. Element resolution is 100% the deterministic scorer — same page state + same step ⇒ same result, every run, with no model to install, no temperature to pin, and no network calls. The *intelligence* lives in the external agent that drives the engine via the [agent commands](#agent-commands--drive-the-engine-from-an-external-llm) (`map` / `run-step` / `read` / `schema`); the runtime itself stays a predictable execution layer.

---

## Four Automation Pillars

The same runtime and the same DSL serve four use cases:

| Pillar | How |
|---|---|
| **QA / E2E testing** | Write plain-English flows, verify outcomes, attach HTML reports and screenshots. No selectors in the test source. |
| **RPA workflows** | Log into portals, fill forms, extract values, hand off to Go for backend or filesystem steps. |
| **Synthetic monitoring** | Pair `.hunt` files with `@schedule:` and `manul daemon` for recurring health checks. |
| **AI agent targets** | Constrained DSL execution is safer than raw CDP/scripting for external agents — the runtime still owns scoring, retries, and validation. |

---

## Key Features

- **Conditional branching & loops** — `IF` / `ELIF` / `ELSE` for adaptive flows; `REPEAT`, `FOR EACH`, `WHILE` for iterating data, retrying actions, and polling dynamic state. Full nesting support (`WHILE` capped at 100 iterations; `REPEAT` exposes a `{i}` 0-based counter).
- **Deterministic targeting** — 4-channel heuristic scorer ranks every candidate with explicit signal breakdowns; contextual qualifiers (`NEAR`, `ON HEADER/FOOTER`, `INSIDE`) restrict candidates spatially and structurally. Shadow DOM, 3-pass proximity resolution, and anti-phantom guards included.
- **Explainability** — `--explain` prints top-5 candidate rankings with per-channel score breakdowns. The VS Code extension shows hover tooltips and an "Explain Current Step" action during debug pauses.
- **Interactive debugger** — `--debug` pauses before every step with a browser modal UI; breakpoint lines (`--break-lines`); `explain-next` previews the scoring for the upcoming (or an overridden) step without executing it.
- **Embeddable agent API (`pkg/agent`)** — `agent.Session` owns Chrome and exposes compact, agent-friendly calls: `Read` (zero-scan), `ReadText`, `Step`/`Run` (typed `Reason` + `Near` candidates), `Map` (budgeted scan). CLI `read` / `run-step --compact` are thin wrappers over it.
  ```go
  import "github.com/alexbeatnik/ManulEngineGo/pkg/agent"

  sess, _ := agent.Launch(ctx, agent.Options{Headless: true})
  defer sess.Close()
  sess.Step(ctx, "Click the 'Login' button")
  sess.Read(ctx, "Order total")
  ```
- **LLM-friendly rendering** — `PageMap.RenderForLLM` lists quoted, copy-pastable labels with the DSL verb per role (`'Search' [textbox — FILL]`); `DescribeForLLM` / `RenderForLLM` translate failures into plain-language advice; `DescribePageChange` says explicitly when the page did NOT change. Tuned so even small local models stay grounded.
- **Desktop / Electron** — Set `--executable-path` and use `OPEN APP` instead of `NAVIGATE` to drive Electron apps with the same DSL.
- **Custom controls** — `RegisterCustomControl(page, target, handler)` lets you handle complex widgets (datepickers, virtual tables, canvas elements) with raw CDP while the hunt file keeps a single readable step.
- **Page-name registry** — `pages/<site>.json` maps URLs to human-readable labels for reports; unknown URLs auto-populate as `Auto: domain/path` placeholders. See [docs/loops-and-pages.md](docs/loops-and-pages.md).
- **Reliable keyboard input** — `PRESS` dispatches real character events (Enter carries `text:"\r"` so forms submit); key names are normalized (`enter`/`esc`/`space`/`down` → DOM key values).
- **HTML reports** — `--html-report` generates per-hunt styled reports plus an aggregate `index.html` for parallel runs.
- **Single static binary** — `go build` produces one self-contained `manul` executable. No runtime, no interpreter, no `node_modules`; ideal for CI images and `scratch`/distroless containers.

---

## Architecture

```
cmd/manul           CLI entry point → produces `manul` binary
pkg/agent           Embedding facade: agent.Session over runtime/cdp/scorer —
                    Launch/Attach/Connect (owns Chrome), Read/ReadText/Step/Run/Map
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
pkg/config          Runtime configuration
pkg/core            Shared enums (e.g. ScrollStrategy)
pkg/pages           Page-name registry: URL → human-readable label
pkg/scan            DOM scanner → draft .hunt; ScanPage (flat) + ScanPageFull (grouped)
pkg/utils           Semantic logging, ANSI stripping, error types
examples/           Sample .hunt files
docs/               Documentation
```

See [docs/overview.md](docs/overview.md) for the deep-dive architecture walkthrough.

---

## Quickstart

### Install

Requires Go ≥ 1.26 and a system-installed Google Chrome / Chromium on `PATH`.

```bash
# build a local binary
make build                  # or: go build -o manul ./cmd/manul

# install on PATH
make install                # user-local (~/.local/bin)
make install-system         # system-wide (/usr/local/bin)

# or straight from the module
go install github.com/alexbeatnik/ManulEngineGo/cmd/manul@latest
```

### Configure

Create `manul_engine_configuration.json` in the workspace root. All keys are optional:

```json
{
  "browser": "chromium",
  "headless": false
}
```

This is the minimal recommended config — fully heuristics-only, no AI dependency.

### Run

```bash
manul examples/login.hunt                        # single file
manul examples/                                  # all hunts in a directory
manul --headless --html-report examples/         # CI mode with reports
```

### Configuration reference

| Key | Default | Description |
|---|---|---|
| `headless` | `false` | Hide the browser window. |
| `browser` | `"chromium"` | `chromium` (launch system Chrome) or `electron` (attach to a running Chrome/Electron over CDP). |
| `browser_args` | `[]` | Extra browser launch flags. |
| `disable_cache` | `false` | Disable the in-session DOM snapshot cache (env: `MANUL_DISABLE_CACHE`, or the inverse `MANUL_SEMANTIC_CACHE_ENABLED`). |
| `timeout` | `5000` | Action timeout, milliseconds. |
| `nav_timeout` | `30000` | Navigation timeout, milliseconds. |
| `workers` | `1` | Max parallel hunt files. |
| `channel` | `null` | Chrome/Chromium binary to launch (`chrome`, `msedge`, `chromium`, …). |
| `executable_path` | `null` | Explicit path to a Chrome/Chromium (or Electron) executable. |
| `retries` | `0` | Retry failed steps N times. |
| `screenshot` | `"on-fail"` | `none`, `on-fail`, or `always`. |
| `html_report` | `false` | Generate an HTML report. |
| `explain_mode` | `false` | Per-channel scoring breakdown in output. |
| `debug_mode` | `false` | Pause before every step for interactive stepping. |

Environment variables (`MANUL_HEADLESS`, `MANUL_BROWSER`, `MANUL_CHANNEL`, `MANUL_WORKERS`, etc.) override JSON config; CLI flags override everything:

```
CLI flags  >  MANUL_* env vars  >  manul_engine_configuration.json  >  config.Default()
```

### Binary distribution

`go build` yields a single static `manul` binary — no interpreter, no `node_modules`, no bundled browser. Drop it into a CI image (or a `scratch`/distroless container with system Chrome) and run headless:

```bash
go build -o manul ./cmd/manul
./manul --headless --html-report --screenshot on-fail hunts/
```

---

## Embedding (Go API)

To drive a browser from a Go program — an assistant, an agent, a custom tool — embed `pkg/agent`. ManulEngine owns the entire browser lifecycle; the consumer just calls a small, compact API and never touches CDP or the runtime directly:

```go
import "github.com/alexbeatnik/ManulEngineGo/pkg/agent"

sess, err := agent.Connect(ctx, agent.Options{Port: 9222}) // attach if Chrome is up, else launch & own it
// or: agent.Launch(ctx, agent.Options{Headless: true})    // always spawn & own Chrome
// or: agent.Attach(ctx, "http://127.0.0.1:9222", "", ...) // use a specific running Chrome
defer sess.Close()                                          // reaps Chrome when this session launched it

out, _ := sess.Step(ctx, "Click the 'Login' button")  // compact: ok, reason, score, near[]
total, _ := sess.Read(ctx, "Order total")             // zero-scan targeted text
text, _ := sess.ReadText(ctx, "#answer")              // sanitized region/page text
pm, _ := sess.Map(ctx, agent.MapBudget{MaxPerGroup: 8}) // budgeted landmark map
res, _ := sess.Run(ctx, huntScript)                   // whole .hunt, compact aggregate
ps, _ := sess.PageState(ctx)                          // {Title, URL} snapshot
ans, _ := sess.Lookup(ctx, url, 3*time.Second, "")    // background-tab read (no UI disruption)

// Prompt-ready presentation — an embedding app needs zero browser code:
prompt := pm.RenderForLLM(5)            // page map: quoted labels + per-role DSL verb (CLICK/FILL/SELECT/CHECK)
report := res.RenderForLLM()            // whole run in plain language, with corrective advice per failed step
line := out.DescribeForLLM()            // one step outcome: what ran, why it failed, what to do next
diff := agent.DescribePageChange(before, ps) // ALWAYS speaks — incl. explicit "the page did NOT change"
```

Failures carry a machine-readable `Reason` (`not_found` / `ambiguous` / `timeout` / `verify_failed` / `action_failed`) and, for resolution problems, the top candidates in `Near` — so a caller branches on a typed reason instead of parsing error strings, and corrects a weak match without a follow-up scan.

`Session.Lookup` opens a URL in a **background tab** (the active page is never switched away from), waits for it to settle, runs an optional extractor JS, then closes the tab — so a consumer needs no CDP code to do unobtrusive web lookups in the user's logged-in profile. `agent.Connect` is the one-call lifecycle entry point: it attaches to a Chrome already listening on the debug port, or launches (and owns) one otherwise.

### Parallel Execution (Go API)

The `manul` CLI runs single-threaded unless you pass `--workers`. For full control over parallelism, embed the worker pool directly:

```go
import (
    "context"
    "github.com/alexbeatnik/ManulEngineGo/pkg/config"
    "github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
    "github.com/alexbeatnik/ManulEngineGo/pkg/report"
    "github.com/alexbeatnik/ManulEngineGo/pkg/worker"
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

**Concurrency contract:** one `Runtime`, `Page`, and `ChromeProcess` per worker. Sharing them across goroutines is a data race — verified by `go test -race` in CI. Register all `CALL GO` handlers and custom controls **before** spawning the pool.

---

## Ecosystem

| Component | Role | Links |
|-----------|------|-------|
| **ManulEngine (Python)** | Deterministic automation runtime (Python). The reference sibling — same DSL, scorer, and agent JSON. | [PyPI](https://pypi.org/project/manul-engine/) · [GitHub](https://github.com/alexbeatnik/ManulEngine) |
| **ManulEngine (Go)** | This project — the Go runtime. Single static binary, goroutine parallelism, embeddable `pkg/agent`. | [GitHub](https://github.com/alexbeatnik/ManulEngineGo) |
| **Manul Engine Extension** | VS Code extension with debug panel, explain mode, and Test Explorer integration. Auto-detects the Go runtime from `go.mod`. | [Marketplace](https://marketplace.visualstudio.com/items?itemName=manul-engine.manul-engine) · [Open VSX](https://open-vsx.org/extension/manul-engine/manul-engine) · [GitHub](https://github.com/alexbeatnik/ManulEngineExtension) |
| **ManulMcpServer** | MCP bridge that gives Copilot Chat and other agents access to the engine. | [Marketplace](https://marketplace.visualstudio.com/items?itemName=manul-engine.manul-mcp-server) · [GitHub](https://github.com/alexbeatnik/ManulMcpServer) |

### Contributing and running tests

```bash
git clone https://github.com/alexbeatnik/ManulEngineGo.git
cd ManulEngineGo
# Requires Go ≥ 1.26 and a system-installed Google Chrome / Chromium on PATH.

go build ./...                                   # build everything
go test ./...                                    # full unit + synthetic suite
go test -race ./pkg/worker/...                   # concurrency safety
```

### Development guides

- [**Scoring & Heuristics**](.claude/skills/scoring-heuristics/SKILL.md)
- [**Concurrency Rules**](.claude/skills/concurrency-rules/SKILL.md)
- [**Adding DSL Commands**](.claude/skills/adding-dsl-commands/SKILL.md)
- [**Go Calls & Extensions**](.claude/skills/extensions-and-go-calls/SKILL.md)
- [**Hunt Authoring**](.claude/skills/hunt-authoring/SKILL.md)

---

## Get Involved

ManulEngine (Go) is alpha-stage and solo-developed. If deterministic, explainable browser automation interests you:

- Build it: `go install github.com/alexbeatnik/ManulEngineGo/cmd/manul@latest` (needs system Chrome/Chromium)
- File issues: [github.com/alexbeatnik/ManulEngineGo/issues](https://github.com/alexbeatnik/ManulEngineGo/issues)

---

## Project Status

**Alpha.** The core engine covers:

- 32+ DSL commands, full control flow (`IF`/`ELIF`/`ELSE`, `WHILE`, `REPEAT N TIMES`, `FOR EACH`)
- 4-channel deterministic scoring with contextual qualifiers (`NEAR`, `ON HEADER/FOOTER`, `INSIDE`)
- Shadow DOM support, 3-pass proximity resolution, anti-phantom guards
- `[SETUP]` / `[TEARDOWN]` hook blocks with fail-fast setup and guaranteed teardown; `@script:` aliases for `CALL GO` handlers
- Import system (`@import:`, `USE`/`CALL` expansion) and the page-name registry (`pages/<site>.json`)
- HTML reporting, screenshots, debug mode, explain mode
- Native `WorkerPool` for parallel execution with per-worker Chrome isolation
- Embeddable `pkg/agent` API (`Read`/`ReadText`/`Step`/`Run`/`Map`) with typed failure reasons and plain-language rendering for LLM drivers
- Strongly-typed extension API (`CALL GO`, `RegisterCustomControl`); race-detector-safe CDP transport

**Version:** `0.1.0` — `manul --version` and the contracts report `0.1.0` (no prefix); the git module tag carries the `v` prefix Go requires: `go get github.com/alexbeatnik/ManulEngineGo@v0.1.0`.

> **VS Code Extension:** The [Manul Engine Extension](https://marketplace.visualstudio.com/items?itemName=manul-engine.manul-engine) supports ManulEngine (Go) out of the box. Open a workspace containing `go.mod` and the extension auto-detects the Go runtime, surfaces `CALL GO` snippets, and validates `CALL GO` steps inside hook blocks.

---

## License

Apache-2.0.
