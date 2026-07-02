# Reports & Explainability

> **ManulEngine (Go) 0.1.0**

ManulEngine (Go) provides multiple layers of observability: HTML reports with screenshots, per-channel scoring breakdowns, and an interactive debugger with a read-only explain-next preview.

## HTML Reports

### Generating a report

```bash
manul --html-report examples/
manul --html-report --screenshot on-fail examples/   # screenshots on failure
manul --html-report --screenshot always examples/    # screenshot every step
```

Each hunt gets a styled per-hunt HTML report under `reports/`; parallel/directory runs additionally get an aggregate `reports/index.html` with one row per hunt and worker attribution.

### Screenshot modes

| Mode | CLI flag | Behavior |
|------|----------|----------|
| **on-fail** (default) | `--screenshot on-fail` | Captures a screenshot only when a step fails |
| **always** | `--screenshot always` | Captures a screenshot after every step (forensic) |
| **none** | `--screenshot none` | No screenshots |

The `SCREENSHOT ["<name>"]` DSL step additionally saves an on-demand PNG under `screenshots/`.

### Run history

Each test run appends a JSON Lines entry to `reports/run_history.json`:

```json
{"file": "examples/login.hunt", "name": "login.hunt", "timestamp": "2026-07-02T10:30:00+00:00", "status": "pass", "duration_ms": 3400}
```

The shape is **identical to ManulEngine (Python)** — `{file, name, timestamp, status, duration_ms}` — and the VS Code extension's Scheduler Dashboard reads it to display sparkline history per hunt file.

> **Difference from the Python engine:** report-session merging (`manul_report_state.json`) is an Engine (Python) feature; the Go engine writes one report per run plus the aggregate index.

---

## Explain Mode

Explain mode reveals exactly how the engine scored and selected each element. This is the primary debugging tool when a step resolves to the wrong element or fails unexpectedly.

### Enabling explain mode

```bash
manul --explain examples/login.hunt
```

Or set `"explain_mode": true` in `manul_engine_configuration.json` (env: `MANUL_EXPLAIN`).

### Reading the output

For each targeted step, explain mode prints the top-5 candidate ranking with per-channel signals:

```text
explain: top 5 for "Login"
  #1 score=0.593 conf=9/10 <button> "Login"
      xpath=/html/body/main/form/button[1]
  #2 score=0.112 conf=7/10 <a> "Login help"
      xpath=/html/body/footer/a[2]
```

The channels behind the total (text · id · semantics · proximity, plus visibility/interactability penalties) are defined by `contracts/MANUL_SCORING_CONTRACT.md` and are identical to the Python engine's.

### Machine-readable outcomes

For drivers and CI, prefer the structured outputs over parsing logs:

```bash
manul run --json file.hunt        # full HuntResult JSON on stdout
manul run --jsonl file.hunt       # per-step JSON Lines + final HuntResult
manul run-step "Click 'Login'"    # compact StepOutcome JSON (default)
```

`StepOutcome.reason` ∈ `ok · not_found · ambiguous · timeout · verify_failed · action_failed`; failed/low-confidence steps carry `near` candidates with scores.

---

## Interactive Debugger

```bash
manul --debug examples/login.hunt              # pause before every step
manul --debug --break-lines 12,20 file.hunt    # pause only at those lines
```

At a pause (TTY): `next` · `continue` · `debug-stop` · `highlight <xpath>` · `explain` · `abort`. An in-browser modal shows the paused step with an Abort button; the resolved target is highlighted on the page.

`explain-next` (used by the VS Code extension, also available as `explain` in the TTY) is a **read-only scoring preview** of the upcoming step — it ranks the live DOM without executing anything, so you can edit the step text and re-preview until the confidence is right.

> **Difference from the Python engine:** Engine (Python) additionally has a What-If REPL command (`!execute`) that *replaces and runs* the current step; the Go debugger's preview is read-only by design.

The pause/explain wire protocol (`MANUL_DEBUG_PAUSE` / `MANUL_EXPLAIN_NEXT` NUL-markers on stdout) is shared byte-for-byte with the Python engine — see `contracts/EXTENSION_ENGINE_CONTRACT.md`.
