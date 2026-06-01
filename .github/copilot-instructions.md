# Copilot Instructions — ManulHeart

## GLOBAL DOC SYNC RULE

> **CRITICAL — Read this first.**
> Whenever the user asks to update documentation, a README, or a feature's description, you **MUST** automatically update the repo-local core files to keep the project's positioning and syntax rules perfectly synchronised:
> 1. `README.md` — public-facing feature docs and version footer
> 2. `.github/copilot-instructions.md` — AI training context and syntax rules
>
> A feature that appears in one file but not the other is a documentation bug.
> This `.github/copilot-instructions.md` file is the single canonical source of Copilot/LLM instructions for this repository.

## SOLO DEV ALPHA POSITIONING

> **CRITICAL — Read this first.**
> ManulHeart is a Go-based high-performance port of the ManulEngine interaction logic. It is currently in **active alpha**. 
> Never imply guarantees of stability, maturity, or production-readiness.
> Bugs are expected, APIs may change, and the project is meant for technical exploration.

## CLI INSTALL + VERSION

> **CRITICAL — Read this first.**
> Current documented ManulHeart CLI version is **0.0.1.4**.
> When documenting install or usage, prefer the Go binary as a PATH-visible system command named `manul`
> (for example `~/.local/bin/manul` or `/usr/local/bin/manul`) so editor extensions can invoke it directly.
> Do not document the repo-local binary as the only intended integration path when the request is about running from tools or extensions.

## AI Identity Directive

**CRITICAL — Read this first.**
ManulHeart is a **deterministic, DSL-first Web Automation Runtime** written in Go. It is NOT an AI-first tool.

1. **Prioritise deterministic actions.** Always default to the plain-English DSL (`CLICK`, `FILL`, `VERIFY`, `NAVIGATE`, `EXTRACT`, `SELECT`, `CHECK`, etc.) and the built-in `Scorer` heuristics.
2. **Three-Pass Targeting Strategy.** ManulHeart uses a robust multi-pass resolution for restrictive interaction modes (checkboxes, radios, selects):
   - **Pass 1 (Strict Match):** Finds elements of the requested type (e.g., input[type=checkbox]) matching the text directly.
   - **Pass 2 (Anchor Anchor):** Finds a non-interactive element (e.g., a <td> containing "7") to use as a proximity anchor.
   - **Pass 3 (Refined Target):** Searches for the desired interactive element near the identified anchor. If found within proximity limits, it targets that; otherwise, it targets the anchor and lets the action handler perform local refinement.
3. **Execution Robustness.** Interactions with checkboxes and radios use a full sequence of simulated mouse events (`mousedown`, `mouseup`, `click`) and native `el.click()` to ensure compatibility with modern frameworks (React, Vue, jQuery).
4. **Smart Scrolling.** The `SCROLL` command intelligently detects scrollable containers. If the primary target isn't scrollable, it recursively searches for the most deeply nested scrollable child with vertical overflow.

## What is this project?

ManulHeart is a high-performance Go port of the Manul interaction engine.
It acts as a standalone interpreter for the `.hunt` DSL, driving Chromium via CDP.
It resolves DOM elements using a weighted heuristic `Scorer` and a JavaScript `TreeWalker` snapshot probe that handles Shadow DOM boundaries.
It is specifically designed to handle "Frontend Hell": zero-size inputs, hidden labels, custom div-based dropdowns, and paginated dynamic tables.

**Stack:** Go 1.21+ · Chrome DevTools Protocol (CDP) · JavaScript (TreeWalker)
**Dependencies:** exactly one — `github.com/gorilla/websocket`. Do NOT add new
third-party deps (including `golang.org/x/sync`); implement equivalents inline.

## Repository layout

```text
cmd/manul/                 CLI entry point (main.go)
pkg/
  cdp/                     CDP WebSocket transport, Conn lifecycle, Subscription handles
  browser/                 Abstract browser/page interfaces + CDP backend + Chrome lifecycle
  dom/                     Element snapshotting and XPath resolution
  heuristics/              Scoring logic (Scorer), keyword analysis, and embedded JS probes
    snapshot_probe.js      TreeWalker DOM traversal (Shadow DOM aware)
    extract_data.js        Data extraction JS logic
    visible_text_probe.js  Deep text collection
  runtime/                 Interpretation of .hunt files, execution state, variable memory
                           (SINGLE-GOROUTINE — see "Concurrency contract" below)
  worker/                  Worker, WorkerPool, PortAllocator (parallel execution substrate)
  explain/                 Score breakdown and debugging visualization
  report/                  Per-hunt HTML report + aggregate index.html
  config/                  Runtime configuration (20 fields); config.Default() + JSON + env-var loading
  utils/                   Logger (dual-output: stdout+ANSI, file+stripped) + error types
  pages/                   URL → human-readable page label registry; lean/wrapped JSON
                           forms, longest-prefix site match, auto-populate on first hit
  scan/                    `manul scan <URL>` — DOM scanner that generates a draft .hunt file;
                           `ScanPage` (basic, flat) and `ScanPageFull` (grouped by semantic region,
                           Shadow DOM aware) — mirrors ManulEngine's SCAN_JS / FULL_SCAN_JS
  agent/                   Batteries-included embedding facade: agent.Session over
                           runtime/cdp/scorer. Launch/Attach own Chrome; Read (zero-scan),
                           ReadText (region text), Step/Run (compact StepOutcome + typed
                           Reason + Near), Map (budgeted landmark scan). CLI `read` /
                           `run-step --compact` are thin wrappers over this.
examples/                  Reference .hunt files (mega.hunt, sampler.hunt, loops_demo.hunt)
```

## Agent API (`pkg/agent`)

For embedding ManulHeart in agent/LLM applications, `pkg/agent` is the stable,
compact facade — consumers get browser control "out of the box" without
spawning Chrome, speaking CDP, or assembling the runtime themselves:

- `agent.Launch(ctx, Options)` — ManulHeart spawns and owns Chrome; `Close` reaps it.
- `agent.Attach(ctx, cdpURL, urlSubstr, Options)` — connect to an existing Chrome (Close leaves it running).
- `Session.Read(target)` — zero-scan targeted text extraction (one probe, no snapshot).
- `Session.ReadText(selector)` — sanitized visible text of a region (or whole body).
- `Session.Step(instruction)` / `Session.Run(huntScript)` — compact `StepOutcome` / `RunOutcome` with a typed `Reason` (`ok`/`not_found`/`ambiguous`/`timeout`/`verify_failed`/`action_failed`) and top-N `Near` candidates on failure/low-confidence — no scorer breakdown, no error-string parsing.
- `Session.Map(MapBudget)` — landmark-grouped, deduped, ranked, capped page map.

A `Session` owns one single-goroutine `Runtime`; it serializes its own calls
but is not parallel — one Session per goroutine. The CLI (`manul read`,
`manul run-step --compact`) routes through this same code path. The typed
`explain.FailureReason` on `ExecutionResult` is the source the agent `Reason`
mirrors.

## Concurrency contract (`0.0.0.5`+)

> **CRITICAL — Read this before writing any code that touches `Runtime`, `Page`, or CDP.**

1. **`runtime.Runtime` is single-goroutine.** The DOM snapshot cache, variable
   store, and sticky checkbox states are unguarded by design. Sharing a
   `Runtime` between goroutines is a data race, caught by `go test -race`.
2. **To run in parallel, use `pkg/worker`.** A `Worker` owns exactly one
   Chrome process, one `Page`, and one `Runtime`. Use `worker.NewWorker` for
   real Chrome; `worker.AdoptWorker` for tests/embedding with a pre-built
   `Page`.
3. **`WorkerPool` dispatches hunts over a bounded jobs channel.** Options:
   `Concurrency`, `Config`, `Allocator` (required), `ChromeOptions`,
   `FailFast`. Result ordering matches input order; per-hunt errors live on
   `PoolResult.Err`; the outer error is the first failure seen.
4. **`PortAllocator` hands out CDP debug ports.** Call `Acquire()` / `Release()`
   per worker. Two workers must never share a port — the allocator also
   best-effort-checks the port is free at the OS level.
5. **`cdp.Conn` is safe for concurrent use.** Writes are serialised by
   `writeMu`; request IDs use `atomic.Int64`; `Close()` is idempotent via
   `sync.Once`.
6. **`cdp.Conn.Subscribe()` returns a `*Subscription`.** Callers MUST invoke
   `sub.Close()` (typically `defer`). Do NOT use the legacy raw-channel form
   — it no longer exists.
7. **Extension registries (`RegisterCustomControl`, `RegisterGoCall`) are
   package-global.** Register at process init, BEFORE spawning the pool.
   Handlers must themselves be safe for concurrent invocation — every worker
   may call the same handler simultaneously.
8. **CI runs `go test -race` on every package.** Any new goroutine spawn,
   shared map access, or channel plumbing must pass the race detector.

## Step format

Steps are atomic browser instructions. **STEP-grouped (unnumbered) is the standard format.**

**Canonical format:**

```text
STEP 1: Navigate to the page
    NAVIGATE to https://example.com
    VERIFY that 'Login' is present

STEP 2: Enter credentials
    FILL 'Username' with 'admin'
    FILL 'Password' with 'secret'
    CLICK the 'Login' button
    VERIFY that 'Welcome' is present
```

**ABSOLUTE RULES for `.hunt` files:**
1. **Unnumbered DSL Syntax:** NEVER prepend numbers (`1.`, `2.`) to action lines.
2. **Logical `STEP` Grouping:** Use `STEP [Optional Number]: [Description]` for structure.
3. **4-space Indentation:** Action lines under `STEP` headers MUST be indented by 4 spaces.
4. **Static Data (@var):** NEVER hardcode test data. Use `@var: {key} = value` at the top and reference via `{key}`.
5. **Post-Input Guard:** Always follow a `FILL` or `TYPE` step with a `VERIFY ... has value "..."` assertion.

## System Keywords (parser-detected)

* `NAVIGATE to [url]`
* `WAIT [seconds]`
* `PRESS [Key]` (e.g., `PRESS Enter`)
* `CLICK [Target]`
* `DOUBLE CLICK [Target]`
* `RIGHT CLICK [Target]`
* `SELECT [Value] from [Dropdown]`
* `CHECK the checkbox for [Target]`
* `UNCHECK the checkbox for [Target]`
* `SCROLL DOWN` or `SCROLL DOWN inside the list`
* `EXTRACT [target] into {variable_name}`
* `VERIFY that [target] is [present|not present|checked|enabled|disabled]`
* `VERIFY [Target] has [text|placeholder|value] "[Expected]"`
* `USE [BlockName]` (Inlines imported step block)
* `CALL [BlockName]` (Functional alias for USE)
* `DONE.`

## Heuristic Scoring (Normalised 0.0–1.0)

The `Scorer` ranks candidates using weighted categories (unified with Python
ManulEngine as of v0.0.1.0):

1. **Cache (2.00):** Semantic cache and blind context reuse (placeholder in Go).
2. **Text (0.45):** Direct innerText, aria-label, placeholder, label, and data-qa matches.
3. **Semantics (0.60):** Element type alignment, mode synergy (+0.5 on perfect match), and cross-mode penalties (−1.0 for type mismatches).
4. **Attributes (0.25):** Matches on `id`, `class`, `name`, `data-qa`, and anchor-attribute affinity.
5. **Proximity (0.10 base, 1.5 when contextual):** Euclidean distance to an anchor (used in `NEAR`, `INSIDE`).

`Rank()` sorts by unclamped `RawScore` so small differences survive even when
`Total` confidence is capped at 1.0.

## Common Pitfalls & Learnings

* **Shadow DOM:** Standard XPaths fail inside shadow roots. The Go engine uses a custom `ShadowHostPath` and JS-based resolution to bridge shadow boundaries.
* **Invisible Inputs:** React/Vue often hide the real `<input type="checkbox">` behind a styled `<div>`. The engine collects these hidden inputs and uses `Pass 2` anchors to find them.
* **Scroll Lag:** Dynamic dropdowns and lists often need a `WAIT 1` after clicking to allow the DOM to populate.
* **Pagination:** After clicking table pagination links, use `WAIT 1` in the `.hunt` file to let AJAX updates settle before the next targeting probe. Do not use `time.Sleep` in Go production code for this.

## Interaction Robustness

When generating automation logic:
* Use **quoted strings** for target labels (`'Login'`) to ensure high scoring priority.
* For tables, use **text identifiers** (`CHECK the checkbox for 'Item ID'`) – let the 3-pass targeting handle the proximity to the actual checkbox input.
* For custom dropdowns, the engine automatically falls back from `select_option` to `click()` on the resolved target.

## Page Scanner (`0.0.1.2`+)

`pkg/scan` implements the `manul scan <URL>` subcommand. Two modes:

| Mode | Command | Output |
|------|---------|--------|
| Basic | `manul scan <URL>` | Flat draft `.hunt` — all interactive elements in document order |
| Full | `manul scan <URL> --full` | Grouped draft — elements organised by semantic region (form, nav, main, dialog, shadow roots) |

**Shadow DOM:** both modes recurse into `el.shadowRoot` — elements from Web Components are included and annotated with `[shadow]` in the full-page group name.

**Go API:**
- `scan.ScanPage(ctx, url, headless)` → `[]scan.Element`
- `scan.ScanPageFull(ctx, url, headless)` → `map[string][]scan.FullElement`
- `scan.BuildHunt(url, elements)` → `.hunt` string
- `scan.BuildHuntFull(url, groups)` → annotated `.hunt` string with `# ── GroupName ──` section headers

This mirrors ManulEngine's `SCAN_JS` / `FULL_SCAN_JS` behaviour — same JS logic ported to Go-embedded JS strings.

## Stdin Hunt Input (`0.0.1.4`+)

`manul -` (or `manul run - …`) reads a single hunt script from stdin instead of a `.hunt` file. Semantics:

- Parsed via `dsl.Parse(os.Stdin)` and tagged with `SourcePath = "<stdin>"`.
- `@import:` is resolved against the current working directory (there is no source path to anchor against).
- Failure now emits a partial `*HuntResult` even when `RunHunt` returns an error, so downstream consumers (e.g. the OS-Manul dispatcher) can read per-step errors instead of being limited to `exit 1`.
- The failure summary prints as `N/N hunt file(s) failed` instead of `N/0` (counted against `len(hunts)`, not `len(huntFiles)`).

Use cases: editor integrations, CI generators, ad-hoc one-liner runs, dispatcher orchestration. Do not use stdin mode for multi-hunt suites — pass a directory instead.

## Parallel execution (Go API)

The CLI is still single-threaded; the worker pool is a Go API. Typical use:

```go
alloc := worker.NewPortAllocator(9222, 9321)
pool, _ := worker.NewPool(worker.PoolOptions{
    Concurrency: 4,
    Config:      config.Default(),
    Allocator:   alloc,
    FailFast:    false,
})
results, firstErr := pool.Run(ctx, hunts)
```

- Order preserved: `results[i]` corresponds to `hunts[i]`.
- Use `report.GenerateIndex(summaries, outDir)` for an aggregate `index.html`.
- Per-worker logs are prefixed `[wN] ` via `utils.WithPrefix(parent, "[wN] ")`.
- Logger API: `utils.NewLogger(logFile)` (stdout + optional ANSI-stripped file); `l.WithLevel(level)` for verbose mode; semantic methods `BlockStart/Pass/Fail`, `ActionStart/Pass/Fail/Warn`, `HeuristicDetail`, `ActionDetail`.
- Per-hunt report filenames carry an atomic sequence counter — never collide.

## Configuration priority chain (`0.0.1.0`+)

`pkg/config` resolves a 20-field `Config` struct from four sources in strict priority order:

```
CLI Flags  >  MANUL_* env vars  >  manul_engine_configuration.json  >  config.Default()
```

- `config.Default()` always returns a safe zero-configuration baseline — no file required.
- If `manul_engine_configuration.json` exists in the working directory it is merged next.
- `MANUL_HEADLESS`, `MANUL_TIMEOUT`, `MANUL_EXPLAIN`, `MANUL_SCREENSHOT` override the JSON.
- CLI flag parsing applies last and wins unconditionally.

When generating code that reads configuration, always start from `config.Default()` and apply layers on top — never construct a `Config` literal from scratch.

## VS Code Debug Protocol (`0.0.1.0`+)

`pkg/runtime/debug.go` exposes an interactive step debugger driven over stdin/stdout pipes.

### Pause marker

When the engine pauses it writes a NUL-delimited sentinel line to stdout:

```
\x00MANUL_DEBUG_PAUSE\x00{"step":"Click the 'Login' button","idx":3}\n
```

- `step` — raw DSL text of the command about to execute.
- `idx` — **1-based** command index within the hunt (matches line display in the extension).

### Stdin tokens

The extension sends one token per line:

| Token | Effect |
|-------|--------|
| `next` (or empty Enter) | Execute current step, pause at next |
| `continue` | Free-run to next `--break-lines` breakpoint |
| `debug-stop` | Suppress all future pauses (clears all breakpoints, free-runs to end) |
| `abort` | Halt execution immediately with error |
| `explain-next` | Score candidates for current step, emit `MANUL_EXPLAIN_NEXT` payload, then re-emit pause marker |
| `explain-next {"step":"<override>"}` | Score candidates for the overridden step text instead |

### Explain-next payload

After `explain-next` the engine emits a second sentinel then re-pauses:

```
\x00MANUL_EXPLAIN_NEXT\x00<json>\n
\x00MANUL_DEBUG_PAUSE\x00{"step":"...","idx":N}\n
```

The JSON is a 10-field `ExplainNextResult` (matches `explainNextPayload` in `pkg/runtime/debug.go`):

```json
{
  "step":             "Click the 'Login' button",
  "score":            0.87,
  "confidence_label": "high",
  "target_found":     true,
  "target_element":   "<button> Login",
  "explanation":      "top candidate <button> score=0.870 (text=0.450 id=0.000 semantic=0.600 penalty=1.000)",
  "risk":             "",
  "suggestion":       null,
  "heuristic_score":  null,
  "heuristic_match":  null
}
```

`confidence_label` is derived from `score`:

| Score range | Confidence | Label |
|-------------|-----------|-------|
| ≤ 0 | 0 | none |
| > 0, < 0.01 | 1 | low |
| 0.01–0.049 | 3 | low |
| 0.05–0.099 | 5 | medium |
| 0.10–0.499 | 7 | medium |
| 0.50–0.999 | 9 | high |
| ≥ 1.0 | 10 | high |

### shouldPause

`shouldPause(cmd, idx)` returns true when `breakLines` is empty (pause-every-step mode) or the command's line number appears in `breakLines`, UNLESS `debugContinue` is set — in which case all pauses are suppressed. The `idx` parameter enables future index-based breakpoint matching without changing the line-number logic.

## Filesystem artifacts (`0.0.1.0`+)

After every hunt run the engine appends one JSONL record to `<cwd>/reports/run_history.json` (directory created automatically):

```json
{"file":"/abs/path/to/login.hunt","name":"login.hunt","timestamp":"2026-04-21T10:00:00Z","status":"pass","duration_ms":4250}
```

Fields: `file` (absolute path), `name` (`filepath.Base`), `timestamp` (RFC3339 UTC), `status` (`"pass"` or `"fail"`), `duration_ms` (float64). The file is append-only; each record ends with `\n`. Implemented in `pkg/report/run_history.go` via `AppendRunHistory(reportsDir, *explain.HuntResult)`.

## Loop constructs (`0.0.1.1`+)

ManulHeart's parser/runtime supports the same loop forms as Python ManulEngine. All bodies follow the standard 4-space indentation rule.

| Form | Syntax | Notes |
|------|--------|-------|
| Counted | `REPEAT N TIMES:` | `{i}` is auto-set inside the body as a 0-based counter. |
| Iterating | `FOR EACH {var} IN {collection}:` | Iterates a comma-separated list stored in a variable (typically `@var`). |
| Conditional | `WHILE <condition>:` | Hard cap of **100 iterations** to prevent infinite loops. Condition mirrors the `IF` predicate grammar. |

Loops nest inside each other and inside `IF`/`ELIF`/`ELSE`. See `examples/loops_demo.hunt` and [docs/loops-and-pages.md](../docs/loops-and-pages.md) for runnable examples.

## Page-name registry (`0.0.1.1`+)

`pkg/pages` resolves URL → human-readable page label for reports, debug output, and `RegisterCustomControl(pageLabel, target, handler)` lookups. The registry lives in a `pages/` directory next to the hunt files (or the current working directory).

Resolution order on `NAVIGATE`:

1. `document.title` (when available).
2. Longest-prefix site match in `pages/<safe-netloc>.json`, then exact URL → regex → substring → `"Domain"` fallback.
3. URL-derived fallback (`scheme://host/path`).

If no match is found the engine **auto-populates** the file with `Auto: domain/path` placeholders so the registry grows naturally during development. Both lean and wrapped JSON forms are accepted — see [docs/loops-and-pages.md](../docs/loops-and-pages.md). Implemented in `pkg/pages/pages.go`; reload-per-lookup mirrors the Python `_auto_populate_registry()` semantics.

## Testing expectations

- **Default to `-race`:** `go test -race ./...`. CI runs race on every package.
- **Worker tests use `AdoptWorker`:** `pkg/worker/worker_test.go` dispatches 16
  hunts across 16 adopted workers with `MockPage`. That is the canonical
  pattern for verifying "no state bleed" when adding new Runtime state.
- **CDP tests use an in-process `httptest` WebSocket echo server:** see
  `pkg/cdp/conn_test.go`. Any new CDP transport feature must be tested there
  before shipping.
- **Do not introduce `time.Sleep` in production paths.** Prefer context-aware
  waits such as `select { case <-ctx.Done(): ... case <-time.After(...): ... }`
  or explicit readiness checks. Test-only `time.Sleep` is acceptable where
  necessary, but runtime/production code must not depend on fixed sleeps.
