---
name: testing-manulengine-go
description: Run or write tests for ManulEngine (Go) the right way. Use when adding a new test, debugging a flaky test, updating CI, or when the user asks "are tests passing?". Covers race-detector expectations, mock patterns, and the synthetic scorer suite.
---

# Testing ManulEngine (Go)

## Default command

```bash
go test -race -count=1 -timeout 240s ./...
```

`-race` is mandatory — CI runs it on every package. `-count=1` defeats the
test cache (important when mocking time-sensitive code). 240s covers the
full synthetic scorer suite plus the 16-way parallel worker test.

## Per-package quick commands

| Package | Command | Notes |
|---|---|---|
| Scorer (synthetic DOMs) | `go test -race -v ./pkg/scorer/synthetic/...` | 35 files, 15 domain DOMs |
| Runtime (flow + vars) | `go test -race -v ./pkg/runtime/...` | Uses `MockPage` |
| CDP transport | `go test -race -v ./pkg/cdp/...` | Uses `httptest` WebSocket server |
| Worker pool | `go test -race -v ./pkg/worker/...` | Adopted workers, port allocator |
| DSL parser | `go test -race -v ./pkg/dsl/...` | Pure parser, no I/O |

Never drop `-race` in CI. If a test is too slow under `-race`, split it —
don't exempt it.

## Mock patterns

### `runtime.MockPage` — fake browser.Page

For runtime / worker tests that shouldn't launch Chrome. See
[pkg/runtime/mock.go](../../../pkg/runtime/mock.go). Typical shape:

```go
page := &runtime.MockPage{
    URL: "https://example.test/login",
    Title: "Login",
    Elements: []dom.ElementSnapshot{
        {ID: 1, Tag: "button", XPath: "/html/body/button",
         VisibleText: "Sign in", IsVisible: true,
         Rect: dom.Rect{Left: 10, Top: 10, Width: 100, Height: 30}},
    },
}
w := worker.AdoptWorker(1, config.Default(), page, nil)
defer w.Close()
res, err := w.Run(ctx, hunt)
```

> **Essential `MockPage` element fields — do not omit:**
> - `IsVisible: true` — the visibility pre-filter drops hidden elements before
>   scoring. An element without this field defaults to `false` and is silently
>   excluded, causing "cannot resolve element" with no obvious cause.
> - `Rect: dom.Rect{...}` — required by the proximity scorer and the
>   `NEAR`/`INSIDE` Euclidean-distance path. A zero `Rect` causes the scorer
>   to produce a flat zero for the proximity channel and can change rankings
>   unexpectedly.
>
> If your test's target is never resolved but the element is clearly in
> `Elements`, check these two fields first.

### `httptest` + `gorilla/websocket` — fake CDP

For `pkg/cdp` tests. See `startMockCDP` in
[pkg/cdp/conn_test.go](../../../pkg/cdp/conn_test.go). The server echoes
JSON-RPC requests and lets the test inject events over a channel. Use
this — do NOT launch real Chrome in unit tests.

## Parallel-safety tests

When you add state to `Runtime` or change how workers share resources, add
a parallel adoption test to
[pkg/worker/worker_test.go](../../../pkg/worker/worker_test.go). The
canonical shape (≥ 8 workers, per-worker mock page, assert no cross-talk):

```go
func TestPool_ParallelExecutionNoStateBleed(t *testing.T) {
    var wg sync.WaitGroup
    errs := make(chan error, N)
    for i := 0; i < N; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            w := worker.AdoptWorker(id, cfg, page, nil)
            defer w.Close()
            // ... run hunt, assert only THIS worker's page saw the effect
        }(i+1)
    }
    wg.Wait()
    close(errs)
    for err := range errs { t.Fatalf("%v", err) }
}
```

The existing version is 16-way — go higher if the state you added is
contentious.

## Writing scorer tests

Synthetic tests live in `pkg/scorer/synthetic/`. Each file models a domain
(e.g. `ecommerce_dom_test.go`, `media_test.go`) as a fixed
`[]ElementSnapshot` plus a table of `{query, mode, expected_id}` rows. The
test walks the table and asserts `scorer.Rank(...)` returns the expected
winner.

When adding a new scoring heuristic:
1. Add or extend a domain DOM to include the shape you want covered.
2. Add rows for each disambiguation the heuristic must handle correctly.
3. Also add rows that must NOT regress — e.g. confirm "Subscribe" still
   beats "Subscribed" after your change.
4. Run the full synthetic suite; a change is only green if all 476+ cases
   pass.

## CI (`.github/workflows/synthetic-tests.yml`)

- Triggers on push to `main` / `feat/*`, plus PRs to `main`.
- Each step isolates a package group and runs with `-race`.
- The final step is `go test -race ./...` — a catch-all.

If a new package should be surfaced (e.g. a named step for visibility), add
a dedicated job step next to the CDP and worker steps.

## Common failures

| Failure | Likely cause |
|---|---|
| `DATA RACE` in `runtime.ScopedVariables` | Shared Runtime across goroutines — use per-worker Runtime |
| `DATA RACE` in snapshot cache | Same — isolate per Worker |
| CDP test hangs for 240s | Mock server not closed; check `defer stop()` |
| Synthetic scorer regression on one case | Your heuristic over-weights a signal. Inspect score breakdown via `scorer.Score(...)` directly |
| `port X already in use` under parallel test | `PortAllocator` range too small, or the OS-level check racing — widen the range |

## What NOT to test

- Real Chrome launches in unit tests — too flaky, too slow. Gate them
  behind `//go:build integration` or env vars if truly needed.
- Time-of-day behaviour (e.g. "report filename format"). Assert structure,
  not literal timestamps.
- Deep JS semantics in `snapshot_probe.js` — tested indirectly via the
  synthetic scorer suite.
