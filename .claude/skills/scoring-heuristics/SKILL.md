---
name: scoring-heuristics
description: Add, tune, or debug ManulEngine (Go)'s element resolution scoring. Use when modifying pkg/scorer, adding a new signal to ElementSnapshot (pkg/dom), extending the JS probe (pkg/heuristics), or wiring a new score component into pkg/explain for debuggability.
---

# Working on the Scorer

ManulEngine (Go) resolves DSL targets to DOM elements by scoring every candidate
on a `[0.0, 1.0]` scale across weighted channels, then ranking. The scorer
is PURE and STATELESS — same inputs always produce the same output. No
randomness, no LLM calls, no hidden caches.

## Architecture

```
.hunt command ──► runtime.loadSnapshot ──► heuristics.ParseProbeResult
                                                    │
                                                    ▼
                                   []dom.ElementSnapshot (37 fields)
                                                    │
                                                    ▼
                            scorer.Rank(query, typeHint, mode, elements, topN, anchor)
                                                    │
                                                    ▼
                                   []RankedCandidate → winner = [0]
                                                    │
                                                    ▼
                                   explain.ExecutionResult (top-5, breakdowns)
```

## Current channels + weights

From [pkg/scorer/scorer.go](../../../pkg/scorer/scorer.go):

```go
var Weights = WeightsConfig{
    Cache:      2.00,  // semantic cache and blind context reuse (not yet implemented in Go)
    Semantic:   0.60,  // type alignment, mode synergy, cross-mode penalties
    Text:       0.45,  // visible text, aria-label, placeholder, label, data-qa
    Attributes: 0.25,  // html id, class-name, data-qa, data-testid, anchor-attr-affinity
    Proximity:  0.10,  // base weight (Euclidean / DOM ancestry); boosted to 1.5 for NEAR/INSIDE
}
```

The 5 categories are independent and combined linearly after category
aggregation. `Total` is the clamped confidence; `RawScore` is the
unclamped weighted sum used for ranking. Every category is printed by
`--explain` and the breakdown is the project's core differentiator.

> **Proximity has two modes.** The `0.10` base weight applies unconditionally
> using XPath-depth DOM ancestry closeness — a structural signal that is
> always present. When a `NEAR` or `INSIDE` qualifier is active, the runtime
> overrides the proximity weight to `1.5` and switches to Euclidean pixel
> distance from the resolved anchor element. These are fundamentally different
> signals, not a "bump" of the same one. Never assume proximity is negligible
> just because the base weight is small — in contextual mode it dominates.

## Mode synergy & cross-mode penalties (v0.0.1.0+)

The semantics channel includes two Python-aligned modifiers computed in
`scorer.Score()` after the raw tag-semantics and type-hint signals:

- **Mode synergy (+0.5):** When any text signal is a *perfect* match
  (exact text, exact aria, exact data-qa, exact label, or exact placeholder),
  the semantics score receives a +0.5 bonus IF the element type aligns with
  the interaction mode:
  - `clickable`/`hover` → real button or real link
  - `input` → real input (not checkbox/radio/button)
  - `select` → native `<select>`, `<option>`, or combobox/listbox role
- **Cross-mode penalty (−1.0):** Elements of the wrong type for the mode
  receive a −1.0 raw semantics penalty:
  - `select` mode + checkbox/radio (unless explicitly wanted)
  - `input` mode + checkbox/radio
  - `clickable` mode + real input (when typeHint is "button")

These modifiers are applied *before* the semantic weight (0.60) so their
weighted impact is ±0.3 to ±0.6.

## Adding a new scoring signal

1. **Where does the signal live?**
   - DOM attribute you can read in a probe → extend
     [pkg/heuristics/snapshot_probe.js](../../../pkg/heuristics/) and
     [pkg/dom/ElementSnapshot](../../../pkg/dom/).
   - Derived from existing fields → add a pure helper in `pkg/scorer`.
2. **Which channel does it belong to?** Pick one of the 4. Do NOT invent a
   fifth channel without a design discussion — more channels = more
   interaction effects, harder to tune.
3. **Additive or multiplicative?** Default additive (sum within channel).
   Multiplicative only for HARD GATES (e.g. `IsDisabled → Total = 0`).
4. **Write the synthetic test FIRST.** Add a new case to
   `pkg/scorer/synthetic/*` that would fail without your change.
5. **Run the full synthetic suite.** All 476+ cases must still pass.
   A new heuristic that helps one case while breaking two is not a net win.

## Invariants the scorer must preserve

- **Disabled elements score 0.** `if el.IsDisabled { return ScoreBreakdown{Total: 0.0, InteractabilityScore: 0.0} }`.
- **Hidden elements receive a ×0.1 penalty multiplier** applied to the final
  weighted sum (not filtered out).
- **Pure function.** `Score(query, typeHint, mode, el, anchor)` takes all
  inputs; no global lookups, no randomness.
- **Deterministic ordering.** `Rank()` sorts by unclamped `RawScore` descending;
  ties are broken by stable DOM order — never by `map` iteration.
- **Bounded to `[0.0, 1.0]` per channel** after weighting. Clip if your
  math overflows.

## Debugging a resolution failure

Run the hunt with `--explain`:

```bash
manul path/to/file.hunt --explain
```

Output shows, per targeting command:

```
Target: 'Login' (mode=clickable)
  [1] ID=42 tag=button text="Sign in"     total=0.71  sem=0.60 text=0.08 id=0.00 prox=0.03
  [2] ID=17 tag=a      text="Sign in"     total=0.58  sem=0.40 text=0.08 id=0.00 prox=0.10
  [3] ID=88 tag=button text="Login help"  total=0.52  sem=0.60 text=0.02 id=0.00 prox=-0.10  (penalty)
```

Reading this:
- `total` < `ThresholdHighConfidence` (0.15) → ambiguous, may fail.
- Gap between [1] and [2] < `ThresholdRunnerUpGap` (0.02) → scorer is
  confused; tighten signals or add a qualifier.
- Channel that's suspiciously flat → your new signal isn't firing; check
  the probe output.

## Wiring into explain

New scoring components MUST appear in `explain.ScoreBreakdown`. If you add
a sub-component, either:
- Fold it into an existing channel's display, OR
- Add a named field and update the HTML renderer in
  [pkg/report/html.go](../../../pkg/report/html.go).

Never emit a score that isn't attributable — if `--explain` can't justify
the ranking, the resolution becomes unreviewable.

## Thresholds worth knowing

From [pkg/runtime/runtime.go](../../../pkg/runtime/runtime.go):

```go
ThresholdHighConfidence = 0.15 // strong heuristic match
ThresholdAmbiguous      = 0.03 // minimum for heuristic choice
ThresholdRunnerUpGap    = 0.02 // winner must beat runner-up by this
ThresholdPass3Total     = 0.12 // Pass 3 min total score to accept a refined target
ThresholdPass3Proximity = 0.18 // Pass 3 min proximity score (Euclidean to anchor)
ThresholdPass3Gap       = 0.04 // Pass 3 winner must beat runner-up by this
```

The `Pass3` constants govern the **3-pass targeting strategy** used for
restrictive interaction modes (checkboxes, radios, selects, hidden inputs):

- **Pass 1 (strict match):** finds elements of the requested type matching
  the label directly. If a high-confidence winner is found, stops here.
- **Pass 2 (anchor search):** finds a non-interactive element (e.g. a `<td>`
  containing the text) to use as a spatial anchor.
- **Pass 3 (refined target):** searches for the actual interactive element
  near the Pass 2 anchor. Accepts it only if the total score exceeds
  `ThresholdPass3Total`, proximity exceeds `ThresholdPass3Proximity`, and
  the winner beats the runner-up by `ThresholdPass3Gap`. Otherwise, targets
  the anchor itself and lets the action handler do local refinement.

Changing a threshold requires running the full synthetic suite and
checking that no previously-resolved case silently becomes "ambiguous".

## Common anti-patterns

- **Adding substring-match hacks** ("if text contains 'login', boost 0.5").
  Instead: normalize the text, tokenize, compare token overlap.
- **Reading DOM state in the scorer.** The scorer sees only
  `[]ElementSnapshot`. If you need new data, extend the probe.
- **Global mutable weights** tuned per-hunt. `scorer.Weights` is global and
  effectively immutable after init. Resist the urge to pass per-hunt
  weight overrides.
- **Using `map` iteration for tie-breaking.** Non-deterministic — sort first.

## Key files

- [pkg/scorer/scorer.go](../../../pkg/scorer/scorer.go) — `Score`, `Rank`, `Weights`.
- [pkg/scorer/synthetic/](../../../pkg/scorer/synthetic/) — 35 test files.
- [pkg/dom/](../../../pkg/dom/) — `ElementSnapshot` struct.
- [pkg/heuristics/](../../../pkg/heuristics/) — JS probes, probe result parsing.
- [pkg/explain/explain.go](../../../pkg/explain/explain.go) — `ScoreBreakdown`.
