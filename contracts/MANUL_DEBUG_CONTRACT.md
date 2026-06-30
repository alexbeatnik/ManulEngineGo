# ManulEngine (Go) — Debug & Explain Contract

> **Machine-readable contract for interactive debugging, the pause/explain wire protocol, and explain-next scoring previews.**
> Consumed by the VS Code extension debug driver, CI/CD diagnostics, and downstream tooling.
>
> **Shared surface.** ManulEngine (Go) copy of a contract shared with ManulEngine
> (Python). The **stdin/stdout debug wire protocol** (pause marker, explain-next
> marker, command tokens, 1-based step index) is **identical** across both
> runtimes — it is the same protocol the VS Code extension speaks; see
> `EXTENSION_ENGINE_CONTRACT.md` for the authoritative byte-level spec. The one
> behavioral difference: ManulEngine (Go)'s `explain-next` is a **read-only** scoring
> preview (optionally with a step override); it does **not** support ManulEngine's
> `!execute` What-If step *injection* (replacing and running the current action).

```json
{
  "version": "0.1.0",
  "generatedFrom": "pkg/runtime/debug.go :: shouldPause(), debugPrompt(), debugPromptTTY(), debugPromptExtension(), injectDebugModal(), debugHighlight(), explainStep(), buildExplainNextResult(), explainNextPayload; pkg/config/config.go :: DebugMode, BreakLines, ExplainMode",

  "config": {
    "DebugMode": {
      "jsonKey": "debug_mode",
      "flag": "--debug",
      "env": "MANUL_DEBUG",
      "type": "bool",
      "default": false,
      "description": "Pause execution for interactive stepping. With no breakpoints set, pauses before every DSL command."
    },
    "BreakLines": {
      "jsonKey": "break_lines",
      "flag": "--break-lines",
      "type": "[]int",
      "default": [],
      "description": "1-based .hunt line numbers that act as breakpoints. Empty means pause on every step (when DebugMode is on). CLI accepts a comma-separated list."
    },
    "ExplainMode": {
      "jsonKey": "explain_mode",
      "flag": "--explain",
      "type": "bool",
      "default": false,
      "description": "Print a full scoring breakdown for every targeted element (non-interactive)."
    }
  },

  "pauseDecision": {
    "function": "shouldPause(cmd, idx) bool",
    "state": [
      { "name": "debugContinue", "type": "bool", "description": "Set by 'continue'/'debug-stop'. Suppresses pauses until a breakLine is hit (then re-arms) or for the rest of the run." },
      { "name": "breakLines",    "type": "map[int]bool", "description": "Active line-number breakpoints (from --break-lines)." },
      { "name": "breakSteps",    "type": "map[int]bool", "description": "One-shot index breakpoints appended by the 'next' command." }
    ],
    "rules": [
      "If debugContinue is set: pause only when breakLines[cmd.LineNum] matches (which re-arms debugContinue=false); otherwise skip.",
      "If both breakLines and breakSteps are empty: pause on every step (plain DebugMode).",
      "Otherwise: pause when breakLines[cmd.LineNum] OR breakSteps[idx] matches."
    ]
  },

  "promptModes": {
    "selection": "debugPrompt() routes to TTY mode when stdin is a TTY, otherwise to extension (line-protocol) mode.",
    "tty": {
      "function": "debugPromptTTY()",
      "ui": "Readline prompt ('  > '). Injects the in-browser modal and prints the command list.",
      "commands": ["next", "continue", "debug-stop", "highlight <xpath>", "explain", "abort"]
    },
    "extension": {
      "function": "debugPromptExtension()",
      "transport": "Reads command tokens as stdin lines (1 MB line cap). Emits NUL-delimited markers on stdout.",
      "commands": ["next", "continue", "debug-stop", "abort", "highlight", "highlight <xpath>", "explain-next", "explain", "explain-next <json-override>"]
    }
  },

  "wireProtocol": {
    "note": "Identical to ManulEngine; authoritative spec lives in EXTENSION_ENGINE_CONTRACT.md.",
    "pauseMarker": {
      "format": "\\x00MANUL_DEBUG_PAUSE\\x00<json>\\n",
      "payload": { "step": "string — the raw step line", "idx": "int — 1-based step index" },
      "emittedOn": "Entering a pause, and re-emitted after each non-terminal command (highlight/explain) to re-prompt."
    },
    "explainMarker": {
      "format": "\\x00MANUL_EXPLAIN_NEXT\\x00<json>\\n",
      "payload": "explainNextPayload (see explainNext.fields)"
    },
    "commandSemantics": {
      "next":         "Pause again at the next step (appends a one-shot breakStep at idx+1). Empty line == next.",
      "continue":     "Clear step breakpoints and resume (debugContinue).",
      "debug-stop":   "Clear ALL breakpoints (lines + steps) and resume to completion.",
      "abort":        "Stop the mission (ErrDebugStop).",
      "highlight":    "Scroll the current highlight into view (no arg) or highlight the given xpath.",
      "explain-next": "Emit an explain marker for the current step, or for a step override via 'explain-next {\"step\":\"...\"}'. Read-only."
    }
  },

  "explainNext": {
    "function": "buildExplainNextResult(stepText, cmd) explainNextPayload",
    "description": "Read-only scoring preview for the current (or overridden) step. Ranks the live snapshot via scorer.Rank and projects the top candidate. Highlights the best candidate in the page.",
    "fields": [
      { "name": "step",             "json": "step",             "type": "string" },
      { "name": "score",            "json": "score",            "type": "float64", "description": "Top candidate total score." },
      { "name": "confidence_label", "json": "confidence_label", "type": "string",  "description": "high (>=0.5) | medium (>=0.1) | low (>0) | none (0)." },
      { "name": "target_found",     "json": "target_found",     "type": "bool",    "description": "True when top score > 0." },
      { "name": "target_element",   "json": "target_element",   "type": "string?", "description": "XPath of the top candidate (null when none)." },
      { "name": "explanation",      "json": "explanation",      "type": "string",  "description": "Channel breakdown: text / id / semantic / penalty." },
      { "name": "risk",             "json": "risk",             "type": "string",  "description": "Set to a low-confidence warning when top score < 0.1." },
      { "name": "suggestion",       "json": "suggestion",       "type": "string?", "description": "Next-best candidate when the top is low-confidence." },
      { "name": "heuristic_score",  "json": "heuristic_score",  "type": "float64?" },
      { "name": "heuristic_match",  "json": "heuristic_match",  "type": "string?" }
    ],
    "confidenceThresholds": { "high": ">= 0.5", "medium": ">= 0.1", "low": "> 0", "none": "0" },
    "ttyExplain": "explainStep() prints the top 5 candidates (score, conf/10, tag, truncated text, xpath) to the log instead of emitting a marker."
  },

  "inBrowserModal": {
    "inject": "injectDebugModal(step) renders a draggable overlay showing the paused step with an Abort button.",
    "abortSignal": "The Abort button sets window.__manul_debug_action; the prompt loop polls it every 200ms ('abort' in TTY, 'ABORT' in extension mode) and returns ErrDebugStop.",
    "highlight": "debugHighlight(xpath) marks the resolved element (data-manul-debug-highlight) and scrolls it into view; clearDebugHighlight() removes it."
  },

  "notInHeart": "ManulEngine (Go)'s explain-next is read-only. It does NOT implement ManulEngine's What-If step *injection* (the `!execute <step>` REPL command / _what_if_execute_step that replaces and runs the current action). Use 'explain-next {step override}' for a scoring preview instead."
}
```
