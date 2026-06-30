---
name: hunt-authoring
description: Author or review ManulEngine (Go) .hunt DSL files. Use when the user asks to write, modify, or review a .hunt file, or when generating automation for a web flow that will be run by the `manul` CLI.
---

# Authoring `.hunt` files

`.hunt` is a plain-English, deterministic DSL. It is NOT a general scripting
language — it is a sequence of browser intents resolved by the heuristic
scorer in `pkg/scorer`. Write for the scorer, not for yourself.

## Five absolute rules

1. **No numbering on action lines.** `1.`, `2.` prefixes are a parse error.
2. **4-space indent under `STEP`.** Tabs or 2-space indent break the parser.
3. **Quote every target label.** `Click the 'Login' button`, not `Click Login`.
   Quoted strings score higher in the Text channel.
4. **Never hardcode test data.** Use `@var: {key} = value` at file top, then
   reference `{key}`. Variables resolve at command execution (supports
   `${k}`, `{k}`, `$k` forms, longest-first).
5. **Post-input guard is mandatory.** After every `FILL` / `TYPE`, emit a
   `VERIFY 'Field' has value "..."` so a silently failed input fails loudly.

## Canonical skeleton

```hunt
@title: short suite name
@context: one-sentence description for reports
@var: {username} = standard_user
@var: {password} = secret_sauce
@tags: smoke, login

STEP 1: Land on the login page
    NAVIGATE to 'https://example.com/login'
    VERIFY that 'Sign in' is present

STEP 2: Sign in
    FILL 'Username' field with '{username}'
    VERIFY 'Username' has value '{username}'
    FILL 'Password' field with '{password}'
    CLICK the 'Sign in' button
    VERIFY that 'Dashboard' is present

DONE.
```

## Contextual qualifiers — prefer these over CSS/XPath

- `NEAR 'Anchor Text'` — spatial + DOM-ancestry proximity to an anchor.
- `INSIDE 'Container' row with 'Row Text'` — scopes to a table row containing
  the given text. Standard pattern for per-row buttons/checkboxes.
- `ON HEADER` / `ON FOOTER` — region-restricted candidates.

Example — the right way to click a per-row delete button:

```hunt
CLICK the 'Delete' button INSIDE 'Users' row with 'alice@example.com'
```

NEVER write `CLICK the 4th 'Delete' button` — positional indexing is fragile
and unsupported.

## Control flow (when you need it)

```hunt
IF 'Banner: cookies' is present:
    CLICK the 'Accept' button

REPEAT 3 TIMES as {i}:
    CLICK the 'Next' button

FOR EACH {item} IN {items}:
    FILL 'Search' field with '{item}'
    VERIFY that '{item}' is present

WHILE 'Next page' is present:
    CLICK the 'Next page' button
```

`WHILE` is capped at 100 iterations. Conditions support element
present/absent, `{var} == 'value'`, `{var} != 'value'`,
`{var} contains 'text'`, and truthy checks.

## Imports and blocks

```hunt
@import: Login from 'shared/auth.hunt'
@import: * from 'shared/setup.hunt'

STEP 1: Session
    USE Login
```

`USE` inlines a named `STEP` block from an imported file at parse time.
`CALL` is a functional alias. Imports are cycle-safe (DFS with a visited
set); a circular import is a parse error.

## Assertions

- `VERIFY that 'X' is present` / `is NOT present`
- `VERIFY that 'Checkbox' is checked`
- `VERIFY 'Email' field has value '...'` — for inputs
- `VERIFY 'Heading' field has text '...'` — for text content
- `VERIFY SOFTLY that '...' is present` — warning-only, hunt continues

## Advanced / less common commands

### Keyboard input

```hunt
PRESS Enter
PRESS Escape
PRESS Control+A
PRESS Tab ON 'Username'    # fires key event on a specific element
```

### Network wait

```hunt
WAIT FOR RESPONSE "api/users"
```

Blocks until a network response URL matching the substring is received.
Use after form submits that trigger XHR/fetch requests to avoid polling
with `WAIT N` fixed delays.

### Drag-and-drop and file upload

```hunt
DRAG 'Card A' and drop onto 'Column B'
UPLOAD '/path/to/file.png' to 'Avatar Upload'
```

`DRAG` uses CDP mouse events with an HTML5 `DragEvent` fallback.
`UPLOAD` resolves the file-input element via the scorer and sets its
value directly — no OS file-picker involved.

### Data extraction

```hunt
EXTRACT the 'Price' into {total}
PRINT 'Current price: {total}'
```

`EXTRACT` is table-aware: if the target resolves inside a `<table>`, it
uses the column header as context. The extracted value is stored as a
string variable reachable via `{total}` for the rest of the hunt.

### Debugging inside a hunt

```hunt
DEBUG VARS    # dumps all current runtime variables to the console
PAUSE         # enters interactive breakpoint; no-op unless --debug is active
```

Run with `--debug` for interactive step-by-step execution with breakpoints:

```bash
manul file.hunt --debug
```

`PAUSE` halts execution and prompts for input — useful when developing a
hunt against a live browser. Remove `PAUSE` lines before committing.

## Common pitfalls

| Symptom | Cause | Fix |
|---|---|---|
| "cannot resolve element" on a custom widget | The real `<input>` is hidden behind a styled `<div>` | Use `CHECK the checkbox for 'Label'` — engine runs the 3-pass anchor search automatically |
| Intermittent pagination failures | Click fires before AJAX resolves | Add a `WAIT FOR 'Row N+1' to be visible` after the click |
| Wrong row clicked in a table | Missing row scope | Use `INSIDE 'Table' row with 'unique text'` |
| Dropdown doesn't open with `SELECT` | Non-native custom dropdown | Engine falls back to `click()` automatically; verify with `VERIFY 'Field' has text '...'` |

## When asked to write a hunt

1. Ask or infer which URL / flow.
2. Draft the `@title`/`@context`/`@var` header.
3. Split into logical `STEP` blocks — one user intent per STEP.
4. Quote every target; pick the MOST VISIBLE text the user would read.
5. Add post-input VERIFY after every FILL/TYPE.
6. Close with `DONE.`.

Test by running: `manul path/to/file.hunt --headless --explain` — `--explain`
prints the top-5 scored candidates per step, which is the fastest way to
diagnose a bad target.
