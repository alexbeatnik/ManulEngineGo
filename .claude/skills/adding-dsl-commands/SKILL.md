---
name: adding-dsl-commands
description: Add a new verb/command to the ManulEngine (Go) .hunt DSL. Use when extending the parser with a new CommandType, wiring runtime dispatch, and writing parser + runtime tests. Covers the full multi-file change checklist.
---

# Adding a New DSL Command

Adding a new `.hunt` verb touches exactly four areas in a fixed order:
parser → runtime dispatch → test. Never skip a step.

## File checklist (in order)

| File | What changes |
|------|-------------|
| `pkg/dsl/parser.go` | New `CommandType` constant + `parseCommand` branch |
| `pkg/dsl/parser_test.go` | Parse-only round-trip test |
| `pkg/runtime/runtime.go` | `executeCommand` switch case |
| `pkg/runtime/runtime_test.go` | End-to-end runtime test with `MockPage` |

## Step 1 — declare the CommandType

`CommandType` is an `int` iota at the top of `pkg/dsl/parser.go`. Add your
constant in alphabetical order among peers of the same interaction category:

```go
const (
    CmdClick CommandType = iota
    CmdFill
    CmdMyNewVerb  // ← add here
    ...
)
```

The numeric values are not stable across versions — never store them
externally.

## Step 2 — teach the parser

`parseCommand(line string, indent int) (Command, error)` uses a chain of
`strings.HasPrefix` / regex checks. Match the longest possible prefix so
partial matches don't shadow later verbs.

Key `Command` fields you will populate — read `pkg/dsl/parser.go` for the
full struct, but the most commonly needed ones are:

```go
type Command struct {
    Type            CommandType
    Target          string   // primary quoted label, e.g. 'Login'
    TypeHint        string   // element type hint: "button", "link", ...
    InteractionMode string   // "clickable", "fillable", "selectable", ...
    Value           string   // secondary argument (FILL value, PRESS key, ...)
    NearAnchor      string   // NEAR 'Anchor' qualifier
    OnRegion        string   // ON HEADER / ON FOOTER
    InsideContainer string   // INSIDE 'Container' qualifier
    InsideRowText   string   // row with 'Row text'
    URL             string   // only for NAVIGATE
    GoCallArgs      []string // only for CALL GO
    GoCallResultVar string   // only for CALL GO result variable
    Body            []Command  // block body (IF/WHILE/REPEAT etc.)
    Branches        []Branch   // IF/ELIF/ELSE branches
}
```

The parser is **indentation-driven** — do NOT try to parse block bodies
yourself. Add your verb to `parseCommand` only; the stack-based outer loop
handles indented children.

### Extracting the quoted target

Use the existing helper pattern:

```go
if m := reQuoted.FindStringSubmatch(line); m != nil {
    cmd.Target = m[1]
}
```

`reQuoted` is a package-level `regexp.MustCompile(`` `'([^']+)'` ``)`.

## Step 3 — wire runtime dispatch

`pkg/runtime/runtime.go` has a large `switch cmd.Type` inside
`executeCommand`. Add a case:

```go
case dsl.CmdMyNewVerb:
    return rt.executeMyNewVerb(ctx, cmd)
```

Then implement `executeMyNewVerb` in the same file (or a sibling file if it
is substantial). Follow the existing pattern:

```go
func (rt *Runtime) executeMyNewVerb(ctx context.Context, cmd dsl.Command) error {
    rt.logger.ActionStart("MY NEW VERB", cmd.Target)

    // 1. Resolve the target element (if it targets a DOM element)
    resolved, err := rt.resolveTarget(ctx, cmd)
    if err != nil {
        rt.logger.ActionFail("MY NEW VERB", err.Error())
        return err
    }

    // 2. Perform browser action
    if err := rt.page.DoSomething(ctx, resolved.Element.XPath); err != nil {
        rt.logger.ActionFail("MY NEW VERB", err.Error())
        return err
    }

    rt.logger.ActionPass("MY NEW VERB", cmd.Target)
    return nil
}
```

**Logging conventions** (from `pkg/utils/logger.go`):

| Call | When |
|------|------|
| `logger.ActionStart(verb, target)` | Before attempting the action |
| `logger.ActionPass(verb, detail)` | On success |
| `logger.ActionFail(verb, detail)` | On error (before returning) |
| `logger.ActionWarn(verb, detail)` | Non-fatal warning |
| `logger.HeuristicDetail(detail)` | Per-candidate scoring detail |

Do **not** use `fmt.Print` or `log.Printf` anywhere.

### If the command does NOT target a DOM element

Commands like `WAIT`, `PRESS`, `NAVIGATE`, `SET` skip `resolveTarget`. Model
after `executePressKey` or `executeNavigate` in `runtime.go`.

## Step 4 — add to `browser.Page` interface (if needed)

New low-level browser actions live in `pkg/browser/page.go` as interface
methods, then implemented in `pkg/browser/cdp_page.go`. Both must be updated
together. If your verb reuses an existing method (`Click`, `Fill`,
`ExecuteScript`, etc.) you can skip this step.

`MockPage` in `pkg/runtime/mock.go` must also get the new method stub — the
compiler will remind you if you forget, because `MockPage` implements
`browser.Page`.

## Step 5 — tests

### Parser test (pure, no browser)

```go
func TestParse_MyNewVerb(t *testing.T) {
    src := `
STEP 1: My step
    MY NEW VERB 'Target Label'
`
    h, err := dsl.Parse(strings.NewReader(src))
    if err != nil {
        t.Fatal(err)
    }
    if len(h.Commands) == 0 {
        t.Fatal("no commands parsed")
    }
    cmd := h.Commands[0].Body[0]
    if cmd.Type != dsl.CmdMyNewVerb {
        t.Errorf("wrong type: %v", cmd.Type)
    }
    if cmd.Target != "Target Label" {
        t.Errorf("wrong target: %q", cmd.Target)
    }
}
```

### Runtime test (uses MockPage)

```go
func TestRuntime_MyNewVerb(t *testing.T) {
    page := &runtime.MockPage{
        Elements: []dom.ElementSnapshot{
            {ID: 1, Tag: "button", VisibleText: "Target Label",
             IsVisible: true, Rect: dom.Rect{Left: 10, Top: 10, Width: 80, Height: 30}},
        },
    }
    w := worker.AdoptWorker(1, config.Default(), page, nil)
    defer w.Close()

    h, _ := dsl.Parse(strings.NewReader(`
STEP 1: Test
    MY NEW VERB 'Target Label'
DONE.
`))
    res, err := w.Run(context.Background(), h)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !res.Passed {
        t.Fatalf("hunt failed: %v", res.FailReason)
    }
}
```

Always run with `-race`:

```bash
go test -race ./pkg/dsl/... ./pkg/runtime/...
```

## Common pitfalls

| Mistake | Consequence |
|---------|-------------|
| Adding the CommandType at the wrong iota position | Silently renumbers all subsequent constants — breaks existing marshalled hunts |
| Parsing with a too-short prefix | Shadows another verb (e.g. `CLICK` can shadow `CLICK THE CHECKBOX`) |
| Forgetting `MockPage` stub for new `browser.Page` method | Compiler error in tests, not caught until CI |
| Using `rt.page` directly without `resolveTarget` | Bypasses the scorer — inconsistent with every other targeting command |
| `fmt.Print` in runtime | Output bypasses the logger, invisible in HTML reports |

## Parse → Expand pipeline

After `Parse()`, callers usually call `h.ResolveImports(baseDir)` and then
`h.Expand()`. `Expand()` inlines `USE`/`CALL` blocks and does NOT re-run
`parseCommand`. Your new verb just needs to survive `Expand()` unmodified —
it will, unless it contains a `Body` that needs recursive expansion (in which
case, follow the pattern of `CmdIf` or `CmdRepeat`).
