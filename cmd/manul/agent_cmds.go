// Agent-facing CLI commands that exist to feed an LLM a compact, authoritative
// view of the engine: `map` (a budgeted page map) and `schema` (the DSL grammar
// + agent JSON shapes). Both keep stdout to the JSON payload so a driver can
// pipe them straight into a prompt.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/alexbeatnik/ManulEngineGo/pkg/agent"
	"github.com/alexbeatnik/ManulEngineGo/pkg/scan"
)

// scanElement is the LLM-facing projection of a scanned element: just the
// label, role and editability. The raw scan carries a CSS `locator` and a
// `tag` too, but those are dead weight for a consumer that targets by human
// label — and a CSS selector is not a public targeting API — so we drop them
// from the JSON an agent reads.
type scanElement struct {
	Label    string `json:"label"`
	Role     string `json:"role"`
	Editable bool   `json:"editable,omitempty"`
}

// compactScanGroups projects raw scan groups into the token-lean agent shape,
// dropping unlabeled elements and the locator/tag fields.
func compactScanGroups(groups map[string][]scan.FullElement) map[string][]scanElement {
	out := make(map[string][]scanElement, len(groups))
	for name, els := range groups {
		kept := make([]scanElement, 0, len(els))
		for _, e := range els {
			label := e.Label
			if label == "" {
				continue
			}
			role := e.Role
			if role == "" {
				role = e.Tag
			}
			kept = append(kept, scanElement{Label: label, Role: role, Editable: e.Editable})
		}
		if len(kept) > 0 {
			out[name] = kept
		}
	}
	return out
}

// ── map subcommand ──────────────────────────────────────────────────────────

// cmdMap implements `manul map [flags]` — a landmark-grouped, budgeted scan of
// the page an agent already has open, emitted as compact JSON (label+role only,
// deduped, per-group capped). It is the cheap, prompt-ready alternative to
// dumping a full `scan` draft: the caller bounds the cost via --max-per-group.
func cmdMap(args []string) error {
	fs := flag.NewFlagSet("map", flag.ExitOnError)
	cdpEndpoint := fs.String("cdp", "http://127.0.0.1:9222", "CDP endpoint URL of an already-running Chrome")
	urlSubstr := fs.String("tab", "", "attach to the page whose URL contains this substring")
	maxPerGroup := fs.Int("max-per-group", agent.DefaultMaxPerGroup, "cap elements returned per landmark group")
	includeUnlabeled := fs.Bool("include-unlabeled", false, "keep elements with no human-visible label")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: manul map [flags]\n\n"+
			"  Emits a compact, landmark-grouped JSON map of the open page —\n"+
			"  the structural view an agent uses to pick a target by its label.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	sess, err := agent.Attach(ctx, *cdpEndpoint, *urlSubstr, agent.Options{})
	if err != nil {
		return err
	}
	defer sess.Close()

	pm, err := sess.Map(ctx, agent.MapBudget{
		MaxPerGroup:      *maxPerGroup,
		IncludeUnlabeled: *includeUnlabeled,
	})
	if err != nil {
		return err
	}
	return emitJSON(pm)
}

// ── schema subcommand ───────────────────────────────────────────────────────

// cmdSchema implements `manul schema [--json]` — the engine's self-describing
// contract for LLM consumers: the DSL verbs (with one-line syntax), the agent
// JSON result shapes, and the failure-reason enum. A driver pins this instead
// of stuffing full prose docs into every prompt.
func cmdSchema(args []string) error {
	fs := flag.NewFlagSet("schema", flag.ExitOnError)
	// --json is accepted for symmetry; schema is always JSON. A non-JSON human
	// summary would just duplicate docs/dsl-for-llms.md.
	_ = fs.Bool("json", true, "emit the schema as JSON (always on)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: manul schema [--json]\n\n"+
			"  Emits the DSL grammar + agent JSON shapes as a compact contract\n"+
			"  for LLM consumers.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	return emitJSON(engineSchema())
}

// schemaVerb is one DSL verb in the contract: the keyword, its one-line syntax,
// and a terse note on when to use it.
type schemaVerb struct {
	Verb   string `json:"verb"`
	Syntax string `json:"syntax"`
	Note   string `json:"note,omitempty"`
}

// engineSchema returns the static, version-stamped contract document. Keep this
// in sync with pkg/dsl (verbs) and pkg/agent (shapes); docs/dsl-for-llms.md is
// the human mirror of the same facts.
func engineSchema() map[string]any {
	return map[string]any{
		"engine":  "manul",
		"version": version,
		"targeting": "Elements are resolved by their human-visible label via a deterministic " +
			"scorer — never CSS/XPath. Always quote labels: Click the 'Login' button.",
		"hunt_rules": []string{
			"STEP headers are numbered; action lines are not.",
			"4-space indent under each STEP.",
			"Never hardcode data: @var: {key} = value, reference as {key}.",
			"Follow FILL/TYPE with VERIFY ... has value \"...\".",
		},
		"verbs": []schemaVerb{
			{Verb: "NAVIGATE", Syntax: "NAVIGATE to <url>", Note: "load a page"},
			{Verb: "CLICK", Syntax: "Click the '<label>' button|link", Note: "click an element by label"},
			{Verb: "DOUBLE CLICK", Syntax: "Double-click the '<label>'"},
			{Verb: "RIGHT CLICK", Syntax: "Right-click the '<label>'"},
			{Verb: "FILL", Syntax: "Fill '<label>' with '<value>'", Note: "set an input; follow with VERIFY"},
			{Verb: "TYPE", Syntax: "Type '<value>' into '<label>'", Note: "keystroke-by-keystroke"},
			{Verb: "SELECT", Syntax: "Select '<option>' from the '<label>' dropdown"},
			{Verb: "CHECK", Syntax: "Check the checkbox for '<label>'"},
			{Verb: "UNCHECK", Syntax: "Uncheck the checkbox for '<label>'"},
			{Verb: "HOVER", Syntax: "Hover over the '<label>'"},
			{Verb: "DRAG", Syntax: "Drag '<label>' to '<target>'"},
			{Verb: "PRESS", Syntax: "Press <key>", Note: "e.g. Press Enter"},
			{Verb: "SCROLL", Syntax: "Scroll down|up|to '<label>'"},
			{Verb: "UPLOAD", Syntax: "Upload '<path>' to '<label>'"},
			{Verb: "VERIFY", Syntax: "VERIFY '<label>' has value|text \"<expected>\"", Note: "hard assertion"},
			{Verb: "VERIFY SOFTLY", Syntax: "VERIFY SOFTLY that '<label>' is present", Note: "non-fatal assertion"},
			{Verb: "EXTRACT", Syntax: "EXTRACT '<label>' into {var}", Note: "read text into a variable"},
			{Verb: "WAIT", Syntax: "WAIT <seconds>"},
			{Verb: "WAIT FOR", Syntax: "Wait for '<label>' to be visible|hidden", Note: "wait until present"},
			{Verb: "WAIT FOR RESPONSE", Syntax: "WAIT FOR RESPONSE <url-substr>"},
			{Verb: "SET", Syntax: "SET {var} = <value>"},
			{Verb: "MOCK", Syntax: "MOCK <METHOD> '<path>' with '<file>'", Note: "intercept a request with a fixture"},
			{Verb: "PRINT", Syntax: "PRINT <text|{var}>"},
			{Verb: "SCREENSHOT", Syntax: "SCREENSHOT"},
			{Verb: "REPEAT", Syntax: "REPEAT N TIMES:", Note: "{i} is a 0-based counter; ends at END REPEAT"},
			{Verb: "FOR EACH", Syntax: "FOR EACH {x} IN {list}:", Note: "comma-separated list; ends at END FOR"},
			{Verb: "WHILE", Syntax: "WHILE <condition>:", Note: "capped at 100 iterations; ends at END WHILE"},
			{Verb: "IF", Syntax: "IF <condition>:", Note: "with optional ELIF/ELSE; ends at END IF"},
			{Verb: "CALL GO", Syntax: "CALL GO <package.function>", Note: "invoke a registered Go function"},
			{Verb: "USE", Syntax: "USE <blueprint>", Note: "expand a blueprint"},
		},
		"step_outcome": map[string]string{
			"ok":     "bool — step succeeded",
			"action": "string — lowercase verb (click, fill, navigate…)",
			"value":  "string — value used/extracted (omitted when empty)",
			"url":    "string — page URL after the step (omitted when unchanged)",
			"reason": "string — one of the failure_reasons enum; 'ok' on success",
			"error":  "string — raw error message (omitted on success)",
			"score":  "number — winning candidate score [0..1]",
			"near":   "array of {text, score} — top candidates on failure or low-confidence match",
		},
		"page_map": map[string]string{
			"url":      "string — current page URL",
			"groups":   "array of {name, elements[], truncated}",
			"element":  "{label, role, editable?}",
			"ordering": "Page first, then content landmarks, then chrome (header/nav/footer).",
		},
		"failure_reasons": []string{
			string(agent.ReasonOK),
			string(agent.ReasonNotFound),
			string(agent.ReasonAmbiguous),
			string(agent.ReasonTimeout),
			string(agent.ReasonVerifyFailed),
			string(agent.ReasonActionFailed),
		},
		"agent_commands": map[string]string{
			"run-step --compact": "run one instruction → StepOutcome JSON",
			"read":               "read one labelled value (zero-scan) → {value, found, reason}",
			"read --selector":    "read sanitized region text → {text, selector}",
			"map":                "compact landmark-grouped page map → page_map JSON",
			"schema":             "this contract",
		},
	}
}
