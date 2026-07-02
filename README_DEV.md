<p align="center">
    <img src="images/manul.png" alt="ManulEngine (Go) mascot" width="160" />
</p>

# 😼 ManulEngine (Go) 0.1.0 — Deterministic Web & Desktop Automation Runtime

> **Developer README.** The user-facing tour lives in [README.md](README.md); this file is the
> engineering manual: project structure, runtime architecture, extension points, configuration,
> testing, and release mechanics. The Python sibling keeps the same manual in
> [ManulEngine/README_DEV.md](https://github.com/alexbeatnik/ManulEngine/blob/main/README_DEV.md).

**Status: Alpha.** Solo-developed, battle-tested against synthetic DOM suites and real sites.
No stability promises; the core claim is transparency — every resolution is explainable.

---

## 📁 Project Structure

```
cmd/manul           CLI entry point → produces the `manul` binary
pkg/agent           Embedding facade: agent.Session (Launch/Attach/Connect,
                    Read/ReadText/Step/Run/Map, Lookup, PageState) + LLM renderers
pkg/cdp             CDP WebSocket transport + domain wrappers (per-frame contexts)
pkg/browser         Browser/Page interfaces, CDP backend, Chrome process lifecycle
pkg/runtime         DSL execution: probe → filter → score → resolve → act;
                    control flow, ScopedVariables, [SETUP]/[TEARDOWN], registries
pkg/worker          Worker / WorkerPool / PortAllocator (parallel directory runs)
pkg/dom             ElementSnapshot (37 normalized fields)
pkg/heuristics      In-page JS probes (snapshot, visible-text, xpath, extract, page-text)
pkg/scorer          Deterministic 4-channel scorer + contextual qualifiers
pkg/dsl             .hunt parser, imports/@script aliases, command AST
pkg/explain         ExecutionResult / HuntResult / candidate explainability types
pkg/report          Per-hunt HTML report + aggregate index.html + run_history.json
pkg/config          Config struct, JSON file + MANUL_* env + defaults
pkg/pages           Page-name registry (pages/<site>.json, auto-populate)
pkg/scan            manul scan: flat + --full landmark-grouped drafts
pkg/daemon          @schedule: watcher (manul daemon)
pkg/record          manul record: interaction recorder
pkg/utils           Semantic logger (Block/Action/Detail), error types
contracts/          Frozen public-surface contracts (MANUL_*_CONTRACT.md ×8 + extension)
docs/               User documentation
examples/           Sample .hunt files
.claude/skills/     Deep-dive engineering guides (scoring, concurrency, DSL, testing)
```

`AGENTS.md` is the long-form internals map for both humans and AI assistants working in this repo.

---

## 🏛️ Architecture — the engine as a runtime

One deterministic pipeline serves every consumer (CLI runs, agent commands, embedding API):

```
.hunt line ──► pkg/dsl (parse) ──► pkg/runtime (dispatch)
                                        │
                          snapshot probe (pkg/heuristics, JS)
                                        │
                          37-field ElementSnapshot (pkg/dom)
                                        │
                          4-channel scorer (pkg/scorer)
                                        │
                     threshold check ──► CDP action (pkg/cdp Input.*)
                                        │
                     ExecutionResult (pkg/explain) ──► report / StepOutcome
```

- **No LLM in the loop.** Resolution is 100% the deterministic scorer; same page + same step ⇒ same result.
- **Native CDP.** One external dependency (`gorilla/websocket`); trusted `Input.*` events at real
  coordinates; per-frame execution contexts for iframes/OOPIF.
- **True concurrency.** No GIL: `pkg/worker` runs whole hunts in parallel goroutines, one
  `Runtime`+`Page`+`ChromeProcess` per worker (`go test -race`-verified).

---

## ✨ Key Features (dev view)

- Full `.hunt` DSL: control flow (`IF/ELIF/ELSE`, `WHILE`≤100, `REPEAT` with `{i}`, `FOR EACH`),
  contextual qualifiers (`NEAR`, `ON HEADER/FOOTER`, `INSIDE`), waits, strict assertions
  (`VERIFY … has value|text|placeholder`), `MOCK`, `PRINT`, `SCREENSHOT`, `OPEN APP`.
- Agent surface: `manul schema | map | read | run-step` emit compact JSON on stdout (logs → stderr),
  attach to a running Chrome over `--cdp`. Identical JSON shapes with the Python engine
  (`failure_reasons`: `ok, not_found, ambiguous, timeout, verify_failed, action_failed`).
- Embedding API `pkg/agent`: `Connect/Launch/Attach → Session.{Step,Run,Read,ReadText,Map,Lookup,PageState}` +
  prompt-ready renderers (`RenderForLLM`, `DescribeForLLM`, `DescribePageChange`).
- Explainability: `--explain` prints top-5 per-channel rankings; debug wire protocol
  (`MANUL_DEBUG_PAUSE`/`MANUL_EXPLAIN_NEXT` NUL-markers) shared with the VS Code extension.

### 🧹 [SETUP] / [TEARDOWN] hooks and inline `CALL GO`

```hunt
@script: {db} = testdata.Seed

[SETUP]
    CALL GO {db}.CreateUser "{email}" into {user_id}
[END SETUP]

STEP 1: …
    CALL GO api.FetchOTP "{email}" into {otp}

[TEARDOWN]
    CALL GO {db}.Cleanup "{email}"
[END TEARDOWN]
```

Handlers are **registered in-process before the run** (no filesystem imports):

```go
runtime.RegisterGoCall("api.FetchOTP", func(ctx context.Context, inv runtime.GoCallInvocation) (any, error) {
    return fetchOTP(inv.Args[0])
})
```

`[SETUP]` failure marks the mission `broken` and skips browser steps; `[TEARDOWN]` always runs
after a successful setup. Returned scalars land in the `into {var}` target; returned maps set
multiple variables.

### 📋 `@var:` declarations and `@script:` aliases

Five-level `ScopedVariables` precedence: **row > step > mission > global > import**.
`@script: {alias} = package.Func` aliases a registered `CALL GO` handler path.

### 🏷️ `@tags:` + `--tags` filter

`@tags: smoke, auth` in the header; `manul --tags smoke dir/` runs only matching hunts
(env: `MANUL_TAGS`).

### 🎛️ Custom Controls & the page registry

```go
runtime.RegisterCustomControl("Checkout Page", "React Datepicker",
    func(ctx context.Context, page browser.Page, inv runtime.CustomControlInvocation) error {
        // inv.ActionType / inv.Value / inv.Variables; drive the widget via page.EvalJS(...)
        return nil
    })
```

Page names come from `pages/<site>.json` (auto-populated `Auto: domain/path` placeholders;
longest-prefix site matching). `"*"` registers an any-page control. Registries are
`sync.RWMutex`-guarded package globals — register at process init, **never** while workers run
(`ResetRuntimeRegistries()` is test-only).

### 🐹 Public Go API (`pkg/agent`)

```go
sess, _ := agent.Connect(ctx, agent.Options{Port: 9222}) // attach or launch+own
defer sess.Close()
out, _ := sess.Step(ctx, "Click the 'Login' button")     // StepOutcome{OK, Reason, Score, Near}
res, _ := sess.Run(ctx, huntScript)                      // RunOutcome + per-step results
```

Failures carry a typed `Reason` and `Near` candidates — branch on values, not error strings.
One `Session` = one page = one goroutine; use `pkg/worker` for parallel suites.

### 🧠 Deterministic resolution — no LLM in the loop

Weights are shared with the Python engine (`cache 2.0 · semantics 0.60 · text 0.45 ·
attributes 0.25 · proximity 0.10`) and frozen by `contracts/MANUL_SCORING_CONTRACT.md` +
golden-number tests. Don't touch weights without bumping the contract.

---

## 💻 System Requirements

- **Go ≥ 1.26** (build only; the artifact is a single static binary)
- **Google Chrome / Chromium** on `PATH` (or `--executable-path`); CDP is Chromium-only by design
- Linux / macOS / Windows

---

## 🛠️ Installation

### From source (dev mode)

```bash
git clone https://github.com/alexbeatnik/ManulEngineGo.git
cd ManulEngineGo
make build            # → ./manul
make install          # → ~/.local/bin/manul
make install-system   # → /usr/local/bin/manul
```

### From the module

```bash
go install github.com/alexbeatnik/ManulEngineGo/cmd/manul@latest
```

---

## ⚙️ Configuration (`manul_engine_configuration.json`)

Read from the CWD; layering (highest → lowest): **CLI flags → `MANUL_*` env → JSON file → `config.Default()`**.

| Key | Default | Env | Description |
|---|---|---|---|
| `headless` | `false` | `MANUL_HEADLESS` | Hide the browser window. |
| `browser` | `"chromium"` | `MANUL_BROWSER` | `chromium` (launch) or `electron` (attach over CDP). |
| `browser_args` | `[]` | `MANUL_BROWSER_ARGS` | Extra Chrome launch flags. |
| `channel` | — | `MANUL_CHANNEL` | Binary channel: `chrome`, `msedge`, `chromium`, … |
| `executable_path` | — | `MANUL_EXECUTABLE_PATH` | Explicit Chrome/Electron binary. |
| `cdp_endpoint` | — | `MANUL_CDP_ENDPOINT` | Attach to a running Chrome instead of launching. |
| `timeout` | `5000` | `MANUL_TIMEOUT` | Action timeout (ms). |
| `nav_timeout` | `30000` | `MANUL_NAV_TIMEOUT` | Navigation timeout (ms). |
| `disable_cache` | `false` | `MANUL_DISABLE_CACHE` (inverse alias: `MANUL_SEMANTIC_CACHE_ENABLED`) | Disable in-session DOM snapshot cache. |
| `workers` | `1` | `MANUL_WORKERS` | Parallel hunt files (worker pool). |
| `retries` | `0` | `MANUL_RETRIES` | Retry failed steps N times. |
| `verify_max_retries` | `15` | `MANUL_VERIFY_MAX_RETRIES` | VERIFY re-poll budget. |
| `screenshot` | `"on-fail"` | `MANUL_SCREENSHOT` | `none` / `on-fail` / `always`. |
| `html_report` | `false` | `MANUL_HTML_REPORT` | Generate HTML report. |
| `explain_mode` | `false` | `MANUL_EXPLAIN` | Per-channel scoring output. |
| `debug_mode` | `false` | `MANUL_DEBUG` | Pause before every step. |
| `break_lines` | `[]` | — | 1-based breakpoint lines (with debug). |
| `tags` | `[]` | `MANUL_TAGS` | Tag filter. |
| `tests_home` | `"tests"` | `MANUL_TESTS_HOME` | Default output dir for new hunts/scans. |

---

## 🖥️ CLI Usage

```bash
manul file.hunt | dir/ | .            # run (implicit `run` subcommand); `-` reads stdin
manul --headless --html-report dir/   # CI mode
manul --tags smoke --retries 2 dir/   # filter + retry
manul --workers 4 dir/                # parallel pool
manul --debug --break-lines 12,20 f.hunt
manul --explain f.hunt
manul --cdp http://127.0.0.1:9222 --target 'url=app.local' f.hunt

manul scan <URL> [--full] [--output draft.hunt]
manul record <URL> [output.hunt]
manul daemon dir/ --headless          # @schedule: watcher
manul pages [list|migrate]
manul controls list

# agent commands (JSON on stdout, logs on stderr)
manul schema
manul map        [--cdp …] [--tab s] [--max-per-group 8] [--include-unlabeled]
manul read 'Lbl' [--cdp …] [--selector css] [--max-chars n]
manul run-step "Click the 'Login' button" [--cdp …] [--compact]
```

Flags may appear before or after positionals (interleaved parsing, same as the Python CLI).
Exit codes: `0` success, non-zero on any failed hunt/step.

---

## 🧪 Tests

```bash
go test ./...                 # full unit + synthetic suite (21 packages)
go test -race ./pkg/worker/   # concurrency contract
go vet ./...
```

Scoring golden numbers live in `pkg/scorer` tests and must stay identical to the Python
engine's (`scoring_math`). Deep-dive guides: [.claude/skills/](.claude/skills/) —
scoring-heuristics, concurrency-rules, adding-dsl-commands, extensions-and-go-calls,
testing-manulengine-go, hunt-authoring.

**Adding a DSL command:** parser case in `pkg/dsl/parser.go` (+`CommandType`), dispatch in
`pkg/runtime/runtime.go:executeCommand`, tests in both packages, and — if it's public surface —
the DSL contract + `manul schema` verbs list.

---

## 🖱️ VS Code Extension

The [Manul Engine Extension](https://marketplace.visualstudio.com/items?itemName=manul-engine.manul-engine)
auto-detects the Go runtime via `go.mod`: `CALL GO` snippets, hook-block scaffolds, gutter
breakpoints (`--break-lines`), the debug QuickPick overlay, and Test Explorer integration all
work against this binary. The wire contract is `contracts/EXTENSION_ENGINE_CONTRACT.md`
(byte-identical across the three repos).

---

## 🔖 Version Bump

`const version` in `cmd/manul/main.go` is the single source of truth (reported by
`manul --version` and the agent schema, **no `v` prefix**). Bump it together with:
the git tag (`v0.1.0` — Go needs the prefix), README badges/notes, and the
`"version"` field in every `contracts/MANUL_*_CONTRACT.md`. Keep it in lockstep with the
Python engine's `pyproject.toml` version — the two runtimes release the same surface.

---

## 📜 Release Notes: 0.1.0

- **Rebrand:** ManulHeart → **ManulEngine (Go)**; module path `github.com/alexbeatnik/ManulEngineGo`.
- **Cross-runtime parity** with ManulEngine (Python): shared DSL surface (`PRINT`, `SCREENSHOT`,
  `OPEN APP`, END-terminators), identical agent JSON (`schema`/`map`/`read`/`run-step`,
  `failure_reasons`, `step_outcome.score`, `editable` in `map`), identical CLI flags
  (`--workers`, `--channel`, `--html-report` default off, `--disable-cache`, `--target`),
  `run_history.json` byte-shape, full contracts set (9 files) vendored in-repo.
- `VERIFY '<label>' has value|text|placeholder` attribute form implemented at runtime.
- Interleaved CLI flag parsing (flags before or after positionals) across all subcommands.
- Chrome password-manager/leak-detection UI disabled at launch (clean automation runs).

Apache-2.0.
