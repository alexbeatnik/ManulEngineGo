---
name: dsl-imports-variables
description: Debug or extend the ManulEngine (Go) DSL preprocessor. Use when working with @import resolution, USE/CALL block expansion, variable interpolation, or scoped variable behavior. Covers the 5-level precedence hierarchy and circular-import detection.
---

# DSL Imports and Variables

The ManulEngine (Go) DSL has a two-phase preprocessor:

1. **`Parse()`** — tokenizes the raw `.hunt` text into `Hunt{Commands[], Imports[], Vars{}}`
2. **`Expand()`** — resolves `USE` / `CALL` blocks by inlining blueprints, then applies variable substitution

After parsing, callers typically run:

```go
h, _ := dsl.Parse(reader)
h.ResolveImports(baseDir)  // load @import files
h.Expand()                 // inline USE/CALL + interpolate variables
```

## Import resolution (`pkg/dsl/imports.go`)

### `@import` syntax

```hunt
@import: Login from 'shared/auth.hunt'
@import: * from 'shared/setup.hunt'
```

- Named imports: only the listed STEP blocks are available via `USE`
- Wildcard (`*`): all STEP blocks + file-level `@var:` declarations are merged
- Aliases: `@import: Login as Auth from 'shared/auth.hunt'` — blueprint keyed as `"auth"`

### Resolution algorithm

`ResolveImports(hunt)` performs a DFS with a `visited` map:

1. Compute absolute path of each import relative to `hunt.SourcePath`
2. Detect cycles — circular import → parse error
3. Parse the imported file recursively
4. Extract STEP blocks via `extractStepBlocks()`
5. Merge into `hunt.Blueprints` (named) or `hunt.Vars` (wildcard)

### STEP block normalization

`normalizeStepName()` strips prefixes to create blueprint keys:
- `"STEP 1: Login"` → `"login"`
- `"1. STEP 2: Checkout"` → `"checkout"`
- `"STEP: Finalize"` → `"finalize"`

Lookup is case-insensitive. When `USE Login` is encountered, the runtime searches blueprints for `"login"` (lowercase).

### Circular import example

```
A.hunt imports B.hunt
B.hunt imports C.hunt
C.hunt imports A.hunt  → error: "circular import detected"
```

## Variable scoping (`pkg/runtime/variables.go`)

### 5-level precedence hierarchy

Highest priority first:

| Level | Name | Set by | Lifetime |
|-------|------|--------|----------|
| 1 | `LevelRow` | `FOR EACH` / `REPEAT` loop bodies | One loop iteration |
| 2 | `LevelStep` | `EXTRACT INTO`, `CALL GO INTO`, command results | One STEP block |
| 3 | `LevelMission` | `@var:` declarations in the hunt file | Entire hunt |
| 4 | `LevelGlobal` | `SET` command or runtime globals | Entire hunt |
| 5 | `LevelImport` | `@var:` from wildcard-imported files | Entire hunt |

`Resolve(name)` walks levels 1→5 and returns the first match.

### Interpolation rules

`ScopedVariables.Interpolate(s)` replaces placeholders in a string:
- Forms: `$var`, `${var}`, `{var}`
- **Longest-first substitution** — keys are sorted by length descending before replacement, so `$username` is replaced before `$user`
- Brace forms are replaced before bare `$` to avoid partial matches

Example:
```hunt
@var: {user_id} = 42
@var: {user} = alice

FILL 'Name' field with '{user}'        → "alice"
PRINT 'ID: {user_id}'                  → "ID: 42"
```

### Variable lifecycle in control flow

```hunt
FOR EACH {item} IN {items}:
    EXTRACT the 'Price' into {price}   → LevelRow
    PRINT '{item}: {price}'
```

`LevelRow` is cleared after each loop iteration. `LevelStep` is cleared when entering a new STEP block.

## `Expand()` mechanics (`pkg/dsl/parser.go`)

`Hunt.Expand()` does two things:

1. **Blueprint inlining** — replaces `USE BlockName` commands with the commands from `hunt.Blueprints["blockname"]`
2. **Variable interpolation** — runs `applyVars()` on every string field in every command

Expansion is **recursive** for nested block bodies (`IF`, `WHILE`, `REPEAT`, `FOR EACH`). The `expandCommands()` helper descends into `Body` and `Branches`.

## Common debugging scenarios

| Symptom | Cause | Fix |
|---------|-------|-----|
| "block not found in blueprints" | `USE` name doesn't match normalized STEP name | Check `normalizeStepName()` output; ensure case-insensitive match |
| Circular import error | Import cycle via transitive files | Break the cycle by extracting shared code into a third file |
| Variable value is wrong | Shadowing at a higher-precedence level | Use `DEBUG VARS` to dump the 5-level hierarchy |
| `{user_id}` partially replaced as `{user}` | Substitution order | `Interpolate` uses longest-first; check if keys overlap |
| `EXTRACT` value not available in next step | Stored at `LevelRow` but row changed | Ensure you're still in the same iteration/scope |

## Extending the preprocessor

If you add a new `@header` directive (e.g. `@script:`):

1. Add the field to `Hunt` struct in `pkg/dsl/parser.go`
2. Parse it in the top-level parser loop (like `@title`, `@context`, `@var`)
3. Ensure it survives `ResolveImports()` — wildcard imports may need to merge the new field
4. Ensure it survives `Expand()` — if it's a string, run `applyVars()` on it

## Key files

- [`pkg/dsl/parser.go`](../../../pkg/dsl/parser.go) — `Hunt`, `Command`, `Expand()`, `applyVars()`, `expandCommands()`
- [`pkg/dsl/imports.go`](../../../pkg/dsl/imports.go) — `ResolveImports()`, `extractStepBlocks()`, `normalizeStepName()`
- [`pkg/runtime/variables.go`](../../../pkg/runtime/variables.go) — `ScopedVariables`, `Level`, `Interpolate()`
- [`pkg/runtime/runtime.go`](../../../pkg/runtime/runtime.go) — where `Set`, `EXTRACT INTO`, and `CALL GO INTO` store values
