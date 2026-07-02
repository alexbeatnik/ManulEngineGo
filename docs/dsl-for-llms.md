# ManulEngine (Go) DSL — LLM Cheat-Sheet

A compact, copy-pasteable contract for an LLM that **authors `.hunt` files** or
**drives the engine over the agent API**. This is the human mirror of
`manul schema` (which emits the same facts as JSON). Pin one of these in a
prompt instead of dumping the full docs.

## Targeting model

Elements are resolved by their **human-visible label** via a deterministic
scorer — **never** CSS/XPath. Always quote labels: `Click the 'Login' button`.

## `.hunt` rules

1. `STEP` headers are numbered; action lines under them are **not**.
2. 4-space indent under each `STEP`.
3. Never hardcode data — declare `@var: {key} = value`, reference as `{key}`.
4. Always follow `FILL`/`TYPE` with `VERIFY '<label>' has value "<expected>"`.

## Verbs

| Verb | Syntax |
|------|--------|
| `NAVIGATE` | `NAVIGATE to <url>` |
| `CLICK` | `Click the '<label>' button` / `link` |
| `DOUBLE_CLICK` | `Double-click the '<label>'` |
| `RIGHT_CLICK` | `Right-click the '<label>'` |
| `FILL` | `Fill '<label>' with '<value>'` |
| `TYPE` | `Type '<value>' into '<label>'` |
| `SELECT` | `Select '<option>' from the '<label>' dropdown` |
| `CHECK` / `UNCHECK` | `Check the checkbox for '<label>'` |
| `HOVER` | `Hover over the '<label>'` |
| `DRAG` | `Drag '<label>' to '<target>'` |
| `PRESS` | `Press <key>` (e.g. `Press Enter`) |
| `SCROLL` | `Scroll down` / `up` / `to '<label>'` |
| `UPLOAD_FILE` | `Upload '<path>' to '<label>'` |
| `VERIFY` | `VERIFY '<label>' has value\|text "<expected>"` |
| `VERIFY_SOFT` | non-fatal `VERIFY` |
| `EXTRACT` | `EXTRACT '<label>' into {var}` |
| `WAIT` | `WAIT <seconds>` |
| `WAIT_FOR` | `WAIT_FOR '<label>'` |
| `WAIT_FOR_RESPONSE` | `WAIT_FOR_RESPONSE <url-substr>` |
| `SET` | `SET {var} = <value>` |
| `PRINT` | `PRINT <text\|{var}>` |
| `SCREENSHOT` | `SCREENSHOT` |

### Control flow

| Block | Syntax | End | Notes |
|-------|--------|-----|-------|
| `REPEAT` | `REPEAT N TIMES:` | `END REPEAT` | `{i}` is a 0-based counter |
| `FOR EACH` | `FOR EACH {x} IN {list}:` | `END FOR` | `{list}` is comma-separated |
| `WHILE` | `WHILE <condition>:` | `END WHILE` | capped at 100 iterations |
| `IF` | `IF <condition>:` + `ELIF`/`ELSE` | `END IF` | same grammar as `WHILE` |
| `CALL` | `CALL <step-block>` | — | invoke a reusable `@script` block |
| `USE` | `USE <blueprint>` | — | expand a blueprint |

## Agent API result shapes

`run-step --compact` / `agent.Step` → **StepOutcome**:

```json
{ "ok": true, "action": "click", "value": "", "url": "https://…",
  "reason": "ok", "score": 0.82,
  "near": [{ "text": "Log In", "score": 0.18 }] }
```

- `url` is **omitted when unchanged** from the previous step.
- `near` (top candidates) appears only on **failure** or a **low-confidence**
  match (`score < 0.35`) — use it to retarget without a follow-up scan.
- `reason` ∈ `ok` · `not_found` · `ambiguous` · `timeout` · `verify_failed` ·
  `action_failed`.

`map` / `agent.Map` → **PageMap** (compact, budgeted):

```json
{ "url": "https://…",
  "groups": [ { "name": "Page",
                "elements": [ { "label": "Email", "role": "textbox", "editable": true },
                              { "label": "Login", "role": "button" } ],
                "truncated": 3 } ] }
```

`editable` marks inputs an agent can `FILL` (omitted when false).

Groups are ordered for an agent: `Page` first, then content landmarks
(main / forms / results), then chrome (header / nav / footer). Bound the size
with `--max-per-group`.

`read` / `agent.Read` → `{ "value": "...", "found": true, "reason": "ok" }`.
Read uses a dedicated extraction probe (zero-scan), so it carries no `near`
candidates — use `map` or `run-step` to retarget after a miss. `read --selector
<css> --max-chars N` returns sanitized region text, truncated to `N` characters
with a `[+K chars truncated]` marker.

## Get the machine-readable version

```bash
manul schema   # same contract as JSON, version-stamped
```
