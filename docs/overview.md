# ManulHeart — Architecture Overview

> *Why we built a pure Go/CDP runtime instead of wrapping Playwright.*

---

## The Problem with the Status Quo

Most modern browser automation stacks look like this:

```
Your Test  →  Playwright/Selenium Library  →  Browser Driver (Node.js/Python)  →  WebSocket  →  Chrome
```

This architecture carries inherent costs:

1. **Massive dependency footprint.** Playwright downloads hundreds of megabytes of browser binaries, Node.js runtimes, and native bindings. A "simple" `npm install` or `pip install` can pull in 500+ transitive dependencies.
2. **The GIL bottleneck.** Python's Global Interpreter Lock means true CPU parallelism is impossible. You scale by spawning processes, not threads — each with its own memory overhead.
3. **Indirection and opacity.** When an element fails to resolve, you are debugging through three layers of abstraction: your test code → the driver library → the CDP protocol. Error messages are often unhelpful concatenations of framework + browser internals.
4. **Selector fragility.** CSS selectors and XPath break on the slightest DOM refactor. They encode implementation detail, not user intent.

ManulHeart removes every layer between your intent and the browser except the CDP wire itself.

---

## The ManulHeart Stack

```
Your .hunt file
      │
      ▼
  pkg/dsl  ──►  Parse into Hunt{Commands[]Command}
      │
      ▼
  pkg/runtime  ──►  Execute commands, resolve targets, manage state
      │
      ├──►  pkg/heuristics  ──►  In-page JS probe (single TreeWalker pass)
      │
      ├──►  pkg/scorer  ──►  Deterministic 4-channel ranking
      │
      ├──►  pkg/dom  ──►  37-field ElementSnapshot
      │
      └──►  pkg/browser.Page  ──►  Action dispatch
                │
                ▼
          pkg/cdp.Conn  ──►  WebSocket to Chrome
```

There is no Node.js process. There is no Python interpreter in the hot path. There is no browser driver binary. ManulHeart is a single compiled Go binary that opens a WebSocket to Chrome and speaks CDP directly.

---

## Execution Model

### 1. Parse (pkg/dsl)

The `.hunt` parser is a single-pass line scanner with stack-based indentation tracking. It produces:

- `Hunt.Commands` — the main mission body
- `Hunt.SetupCommands` — commands inside `[SETUP]` … `[END SETUP]`
- `Hunt.TeardownCommands` — commands inside `[TEARDOWN]` … `[END TEARDOWN]`
- `Hunt.Vars` — `@var:` declarations (stored at `LevelMission` scope)
- `Hunt.ScriptAliases` — `@script:` alias map for `CALL GO` rewrite
- `Hunt.Imports` / `Hunt.Blueprints` — reusable step blocks from other files

Control flow (IF, WHILE, REPEAT, FOR EACH) is parsed into nested `Command.Body` and `Branch.Body` slices — no bytecode, no AST visitor pattern, just plain Go structs.

### 2. Target Resolution (pkg/runtime) — THE ENGINE CORE

For every action command (Click, Fill, Select, etc.), the runtime runs this exact pipeline:

```
1. CallProbe(SnapshotProbe)
      → pkg/heuristics.BuildSnapshotProbe() returns a self-contained JS arrow function
      → The function runs ONE TreeWalker pass over the live DOM (Shadow-DOM aware)
      → For every interactive element, it collects:
          visibleText, ariaLabel, placeholder, labelText, dataQA, dataTestId,
          htmlId, role, tag, rect, isVisible, isHidden, isDisabled, isChecked,
          isSelected, isEditable, xpath, ancestors, value, nameAttr, className
      → This is the FIRST and ONLY DOM query for targeting

2. deserializeSnapshot()
      → heuristics.ParseProbeResult(raw) → []dom.ElementSnapshot
      → Each snapshot is a normalized 37-field struct

3. scorer.Rank(query, typeHint, mode, elements, limit, anchor)
      → pkg/scorer computes per-channel scores:
          text:      exact/substr text, aria, placeholder, label, dataQA (weight 0.45)
          id:        html id with space→dash/underscore variants (weight 0.25)
          semantic:  tag/role alignment with interaction mode (weight 0.60)
          penalty:   disabled ×0.0, hidden ×0.1 (multiplier)
          proximity: Euclidean distance to anchor (NEAR/INSIDE) weight 1.50,
                     or XPath-depth DOM ancestry weight 0.10 (base)
      → Returns []RankedCandidate sorted by Total score descending

4. Threshold check
      → ThresholdAmbiguous  = 0.03  (minimum for any match)
      → ThresholdHighConfidence = 0.15 (strong match)
      → If the best score is below ambiguous → "target not found"
      → If the best score is above ambiguous but runner-up gap < 0.02 → "too ambiguous"
      → Otherwise → ResolvedTarget{Element, Score, RankedCandidates}
```

Nothing in `pkg/browser` returns "the element to click." That is exclusively `pkg/runtime`'s job. The `Page` interface exposes only raw actions: `Click(x, y)`, `SetInputValue(id, xpath, value)`, `EvalJS(expr)`.

### 3. Action Execution

Once resolved, the runtime:

1. Scrolls the element into view
2. Flashes a red border + yellow background highlight for 2 seconds (Python parity)
3. Performs the action (click, fill, select, etc.)
4. **Immediately clears the highlight** via `ClearHighlight()` — even if the action caused navigation

For restrictive modes (input, checkbox, select), a 3-pass fallback pipeline runs:
- Pass 1: Direct restrictive scoring
- Pass 2: Anchor search in unrestricted mode, then proximity boost
- Pass 3: Row-scoped checkbox search or generic nearby control search

### 4. Explainability (pkg/explain)

Every command execution produces an `ExecutionResult` containing:

- `Step` — raw DSL text
- `CommandType` — the verb
- `PageURL` — current page after execution
- `CandidatesConsidered` — how many elements the probe found
- `RankedCandidates` — top-N candidates with full `ScoreBreakdown`
- `WinnerXPath`, `WinnerScore` — the chosen element
- `ActionPerformed`, `ActionValue` — what was actually done
- `Success`, `Error`, `DurationMS`

Pass `--explain` to see candidate rankings in the terminal. Pass `--json` to get the full structured result for every step.

---

## The 37-Field ElementSnapshot

ManulHeart does not query the DOM incrementally. It queries once, exhaustively, and deserializes everything into a flat struct:

| Category | Fields |
|----------|--------|
| **Identity** | `ID`, `Tag`, `HTMLId`, `ClassName`, `NameAttr`, `XPath` |
| **Text** | `VisibleText`, `AriaLabel`, `Placeholder`, `LabelText`, `Title`, `DataQA`, `DataTestId` |
| **State** | `IsVisible`, `IsHidden`, `IsDisabled`, `IsChecked`, `IsSelected`, `IsEditable`, `IsInteractive` |
| **Geometry** | `Rect` (Left, Top, Width, Height, Right, Bottom) |
| **DOM** | `Ancestors` (semantic tags), `FrameIndex`, `ShadowHost` |
| **Value** | `Value`, `TextContent`, `InnerHTML` |

All 37 fields are populated in a single JS round-trip. The Go scorer then operates on this snapshot without ever touching the page again.

---

## WorkerPool & True Concurrency

Because ManulHeart is pure Go, it can run hunts in parallel using native goroutines — not processes, not threads fighting a GIL.

```
WorkerPool (4 workers)
  ├── Worker 0 → Chrome (port 9222) → Page → Runtime → Hunt A
  ├── Worker 1 → Chrome (port 9223) → Page → Runtime → Hunt B
  ├── Worker 2 → Chrome (port 9224) → Page → Runtime → Hunt C
  └── Worker 3 → Chrome (port 9225) → Page → Runtime → Hunt D
```

Each `Worker` owns exactly one `ChromeProcess`, one `cdp.Conn`, one `browser.Page`, and one `runtime.Runtime`. This isolation is enforced by design:

- `Runtime` is **single-goroutine** (unguarded cache, variable store, checkbox state)
- `cdp.Conn` is **goroutine-safe** (`writeMu`, atomic IDs, idempotent `Close`)
- `RegisterGoCall` / `RegisterCustomControl` are **package-global** — call at process init, before the pool spawns

The `PortAllocator` round-robins CDP debug ports with an OS-level free check. `WorkerPool` uses a bounded jobs channel with first-error tracking and optional `FailFast`. No `golang.org/x/sync/errgroup` — implemented inline with standard library primitives.

```go
results, err := worker.RunHuntsInParallel(ctx, cfg, hunts, 4, logger)
```

Returns per-hunt results in input order. Use `worker.NewPool` directly when you need `FailFast` or custom `ChromeOptions`.

---

## CDP Transport (pkg/cdp)

ManulHeart's CDP layer is a thin, race-safe WebSocket wrapper:

- **Request/response pipelining** — per-call channels keyed by atomic JSON-RPC ID
- **Event subscriptions** — `Subscribe()` returns a `*Subscription` with `C()` and `Close()`
- **Context-aware cancellation** — every `Call()` respects `ctx.Done()` and connection teardown
- **No background goroutines in Runtime** — modal polling in debug mode runs on the caller's goroutine via `time.Ticker`

The transport is ~300 lines. The domain helpers (`Navigate`, `Evaluate`, `Click`, `SetInputValue`, `HighlightElement`, etc.) are another ~400 lines. Compare that to Playwright's ~150,000 lines of TypeScript.

---

## Extensibility Points

| Feature | Where to add |
|---------|-------------|
| More browsers | New `browser.Page` implementation in `pkg/browser/` |
| Custom DSL commands | `pkg/dsl/parser.go` + `pkg/runtime/runtime.go` |
| Setup/teardown hooks | Already supported: `[SETUP]` / `[TEARDOWN]` blocks |
| Script aliases | Already supported: `@script: {alias} = path` |
| Custom controls | `pkg/runtime` — `RegisterCustomControl(page, target, handler)` |
| Go function calls | `pkg/runtime` — `RegisterGoCall(name, handler)` |
| Screenshots | `pkg/cdp` — `Page.captureScreenshot` |
| Scan-page | `manul scan <URL>` — `pkg/scan` with basic and `--full` modes (`0.0.1.2`+) |
| Stdin input | `manul -` reads a hunt script from stdin; `@import:` resolves against CWD (`0.0.1.3`+) |
| Semantic cache | `pkg/runtime` — XPath reuse from previous steps |
| Shadow DOM | `pkg/heuristics` — TreeWalker crossing shadow-root boundaries |
| Proximity targeting | `pkg/runtime` — 3-pass resolution for restrictive inputs |

---

## Why Pure Go / CDP Wins

| Dimension | Playwright/Node.js | ManulHeart |
|-----------|-------------------|------------|
| **Binary size** | ~180 MB (browsers + Node + bindings) | ~15 MB single static binary |
| **Dependencies** | 500+ npm/pip packages | 1 (`gorilla/websocket`) |
| **Startup time** | Seconds (Node boot + browser launch) | Milliseconds (Go binary + Chrome) |
| **Parallelism** | Process-based (memory-heavy) | Goroutine-based (lightweight) |
| **Debugging** | Framework stack traces | Direct CDP errors + structured explainability |
| **Selectors** | CSS/XPath (fragile) | Plain English + heuristics (robust) |
| **Extensibility** | JavaScript/Python plugins | Native Go functions + registries |

ManulHeart is not a wrapper around a wrapper. It is the automation engine, speaking directly to the browser, in a language designed for systems programming.
