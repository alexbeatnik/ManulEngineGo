# ManulEngine (Go) — Go Embedding API Contract

> **Machine-readable contract for the `pkg/agent` Go embedding API — the programmatic way to drive ManulEngine (Go) from a Go application.**
> Consumed by Go programs embedding the engine, agent harnesses, and the `manul` agent subcommands (`map`/`read`/`run-step`), which are thin CLI wrappers over this API.
>
> **Runtime-specific surface.** This is the ManulEngine (Go) analog of ManulEngine's
> Python `ManulSession` async context-manager API. The two are **not** wire- or
> source-compatible (Go package vs Python class), but they expose the **same
> capabilities** and the **same result shapes**: `StepOutcome`, `PageMap`, and the
> `Reason` enum here are byte-identical to the agent JSON in `MANUL_CLI_CONTRACT.md`
> (the `map`/`read`/`run-step` payloads) and to the failure-reason enum shared
> across the DSL/CLI contracts.

```json
{
  "version": "0.1.0",
  "generatedFrom": "pkg/agent/agent.go :: Options, Session, Launch(), Attach(), (*Session).Close/Read/ReadText/Step/Run/Map, Value, Reason, Cand, StepOutcome, RunOutcome, MapBudget, MapElement, MapGroup, PageMap, TruncateText(), DefaultMaxPerGroup",
  "importPath": "github.com/alexbeatnik/ManulEngineGo/pkg/agent",

  "overview": {
    "model": "A Session is a live, owned browser connection plus the targeting runtime. Create one with Launch (ManulEngine (Go) spawns Chrome) or Attach (connect to an already-running Chrome). Always Close it.",
    "lifecycleOwnership": "Launch owns the Chrome process (and any temp profile) — Close terminates it. Attach does NOT own Chrome — Close leaves it running.",
    "concurrency": "Every Session method is guarded by an internal mutex; a single Session drives a single page and serializes calls. Use one Session per page/goroutine."
  },

  "options": {
    "type": "Options",
    "fields": [
      { "name": "Headless",       "type": "bool",            "default": false,            "description": "Run Chrome without a visible window (Launch only)." },
      { "name": "Port",           "type": "int",             "default": "0 → 9222",       "description": "CDP debug port for a Launch-managed Chrome. Ignored by Attach." },
      { "name": "CDPURL",         "type": "string",          "default": "",               "description": "Explicit CDP HTTP endpoint. Ignored by Launch." },
      { "name": "ExecutablePath", "type": "string",          "default": "",               "description": "Override the Chrome binary location (Launch only)." },
      { "name": "UserDataDir",    "type": "string",          "default": "",               "description": "Override the Chrome profile dir (Launch only). Empty → unique temp profile, removed on Close." },
      { "name": "Config",         "type": "*config.Config",  "default": "nil → config.Default()", "description": "Engine configuration (see MANUL_CONFIG_CONTRACT.md)." },
      { "name": "Logger",         "type": "*utils.Logger",   "default": "nil → discard",  "description": "Log sink. Nil discards output so an embedding app's stdout/stderr stays clean (and structured agent JSON is not corrupted)." }
    ]
  },

  "constructors": [
    {
      "name": "Launch",
      "signature": "Launch(ctx context.Context, opts Options) (*Session, error)",
      "description": "Spawns a Chrome process owned by ManulEngine (Go), attaches to its first page, and returns a ready Session. Close tears the process (and temp profile) down."
    },
    {
      "name": "Attach",
      "signature": "Attach(ctx context.Context, cdpURL, urlSubstr string, opts Options) (*Session, error)",
      "description": "Connects to an already-running Chrome at cdpURL (e.g. http://127.0.0.1:9222). When urlSubstr is non-empty, selects the most-recently-active page whose URL contains it; otherwise the first page. Close leaves Chrome running. Requires a non-empty cdpURL."
    }
  ],

  "sessionMethods": [
    {
      "name": "Close",
      "signature": "(*Session) Close() error",
      "description": "Releases the page connection and, for Launch sessions, terminates Chrome and removes any temp profile. Safe to call multiple times (idempotent)."
    },
    {
      "name": "Read",
      "signature": "(*Session) Read(ctx context.Context, target string) (Value, error)",
      "description": "Extracts the text of the element matching a human label, using only the cheap extraction probe (one CDP round-trip, no full snapshot). A target that doesn't resolve (or resolves empty) returns Value{Found:false, Reason:ReasonNotFound} with a nil error — 'nothing there' is a normal answer."
    },
    {
      "name": "ReadText",
      "signature": "(*Session) ReadText(ctx context.Context, selector string) (string, error)",
      "description": "Returns the case-preserved, shadow-DOM-aware visible text of the page, sanitized of markup noise (base64/data-*/SVG path data). Pass a CSS selector to scope to a region; pass \"\" for the whole body. Single probe round-trip."
    },
    {
      "name": "Step",
      "signature": "(*Session) Step(ctx context.Context, instruction string) (StepOutcome, error)",
      "description": "Runs a single plain-English instruction (one DSL line). Failures carry a machine-readable Reason and, for target-resolution problems or low-confidence (<0.35) matches, the top candidates in Near — so an agent can correct course without scanning."
    },
    {
      "name": "Run",
      "signature": "(*Session) Run(ctx context.Context, huntScript string) (RunOutcome, error)",
      "description": "Executes a full .hunt script (multiple lines, STEP blocks, loops, conditionals) against the session's page and returns a compact aggregate with per-step outcomes."
    },
    {
      "name": "Map",
      "signature": "(*Session) Map(ctx context.Context, budget MapBudget) (PageMap, error)",
      "description": "Returns a landmark-grouped, budgeted scan of the current page (one full-scan JS probe; no navigation). The bounded alternative to dumping the DOM — deduped, ranked, per-group capped."
    }
  ],

  "types": {
    "Value": {
      "description": "Result of a Read.",
      "fields": [
        { "name": "Text",   "type": "string", "description": "Extracted, trimmed text. Empty when Found is false." },
        { "name": "Found",  "type": "bool",   "description": "Whether the target resolved to a non-empty value." },
        { "name": "Reason", "type": "Reason", "description": "ReasonOK when Found, ReasonNotFound otherwise. (Read uses the extraction probe, not the scorer, so it offers no Near candidates — use Step/Map to retarget.)" }
      ]
    },
    "Cand": {
      "description": "Compact candidate surfaced on a failed/low-confidence Step.",
      "json": { "text": "string — human-visible label", "score": "float64 — candidate score" }
    },
    "StepOutcome": {
      "description": "Compact result of one Step. Byte-identical to the run-step agent JSON in MANUL_CLI_CONTRACT.md.",
      "json": {
        "ok":     "bool — step succeeded",
        "step":   "string — raw DSL line (omitempty)",
        "action": "string — lowercase command kind (click, fill, navigate…)",
        "value":  "string — value used/extracted (omitempty)",
        "url":    "string — page URL after the step (omitempty)",
        "reason": "Reason — classifies the outcome; ReasonOK on success",
        "error":  "string — raw error message when ok is false (omitempty)",
        "score":  "float64 — winning candidate score (omitempty when 0)",
        "near":   "[]Cand — top candidates on failure or low-confidence (<0.35) match (omitempty)"
      }
    },
    "RunOutcome": {
      "description": "Compact aggregate of running a whole hunt script.",
      "json": {
        "ok":          "bool",
        "url":         "string (omitempty)",
        "total_steps": "int",
        "passed":      "int",
        "failed":      "int",
        "results":     "[]StepOutcome (omitempty) — per-step compact outcomes, in order",
        "duration_ms": "int64"
      }
    },
    "MapBudget": {
      "description": "Bounds a Map call so the result stays cheap for an LLM prompt.",
      "fields": [
        { "name": "MaxPerGroup",      "type": "int",  "default": "0 → DefaultMaxPerGroup (8)", "description": "Cap on elements returned per landmark group." },
        { "name": "IncludeUnlabeled", "type": "bool", "default": false, "description": "Keep elements with no human-visible label (rarely useful targets)." }
      ]
    },
    "MapElement": { "description": "One interactive element in a PageMap group.", "json": { "label": "string", "role": "string" } },
    "MapGroup": {
      "description": "A landmark region and its budgeted elements.",
      "json": { "name": "string — landmark label (Page, Main, Nav: …, … [shadow])", "elements": "[]MapElement", "truncated": "int — elements dropped by the cap (omitempty)" }
    },
    "PageMap": {
      "description": "Landmark-grouped, budgeted view of the current page. Identical shape to the `map` agent JSON. Groups ordered: Page → useful landmarks (Main/forms/results) → chrome (header/nav/footer) → rest alphabetically.",
      "json": { "url": "string (omitempty)", "groups": "[]MapGroup" }
    },
    "Reason": {
      "type": "string enum",
      "description": "Machine-readable outcome classifier. Identical to the failure_reasons enum shared across the DSL/CLI contracts.",
      "values": ["ok", "not_found", "ambiguous", "timeout", "verify_failed", "action_failed"],
      "constants": ["ReasonOK", "ReasonNotFound", "ReasonAmbiguous", "ReasonTimeout", "ReasonVerifyFailed", "ReasonActionFailed"]
    }
  },

  "helpers": {
    "TruncateText": { "signature": "TruncateText(s string, maxChars int) string", "description": "Truncate read output to a character budget for prompt safety." },
    "DefaultMaxPerGroup": { "type": "const int", "value": 8, "description": "Per-group element cap when MapBudget.MaxPerGroup is 0." }
  }
}
```
