---
name: concurrency-rules
description: Enforce ManulEngine (Go)'s concurrency contract when editing runtime/, cdp/, worker/, or anything that spawns goroutines. Use when adding shared state, introducing a goroutine, modifying the Worker/Pool API, or touching CDP transport internals.
---

# ManulEngine (Go) concurrency contract

Established in `0.0.0.2`, extended in `0.0.0.3` with `RunHuntsInParallel`
and per-worker log prefixes, extended in `0.0.0.5` with the configuration
system (`pkg/config`) and VS Code debug protocol (`pkg/runtime/debug.go`),
refined in `0.0.0.6` (20-field `Config`, 37-field `ElementSnapshot`, `pkg/core` enums),
hardened in `0.0.0.7` (removed internal goroutine from `Runtime`, bounded cleanup contexts,
connection-leak fix in CLI).
Every rule here has a test under `-race`; violations trip CI.

## Hard invariants

1. **`runtime.Runtime` is single-goroutine.** Fields are unguarded by design
   (`cachedElements`, `ScopedVariables.levels`, `stickyCheckboxStates`).
   Every parallel unit owns its own `Runtime`. See
   [pkg/runtime/runtime.go:41-58](../../../pkg/runtime/runtime.go) for the
   doc comment.

2. **Parallelism goes through `pkg/worker`.** Never spin up goroutines that
   share a `Runtime`. Use:
   - `worker.NewWorker` — owns a real Chrome + Page + Runtime.
   - `worker.AdoptWorker` — wraps an existing `browser.Page` (tests / embed).
   - `worker.NewPool` — bounded concurrency, jobs channel, first-error tracking.
   - `worker.RunHuntsInParallel(ctx, cfg, hunts, n, logger)` — zero-config
     convenience wrapper that creates a pool, runs hunts, and returns results
     in input order. Use this for quick fan-out; use `NewPool` directly when
     you need `FailFast` or custom `ChromeOptions`.

3. **Ports go through `worker.PortAllocator`.** No hardcoded 9222, no
   parallel-safe assumption without `Acquire()` / `Release()`.

4. **`cdp.Conn` is safe for concurrent use.** Writes serialized by `writeMu`,
   request IDs via `atomic.Int64`, `Close()` via `sync.Once`.

5. **Subscriptions require `Close()`.** `c.Subscribe()` returns `*Subscription`.
   Always `defer sub.Close()`. The channel is closed by the publisher on
   `Conn.Close()`, so receivers must handle the `ok == false` case.

6. **Extension registries freeze at init.** `RegisterCustomControl` /
   `RegisterGoCall` must be called before `pool.Run(...)`. Handlers
   themselves must be concurrent-safe — every worker may invoke the same
   handler simultaneously.

7. **No new external dependencies.** The README brags about exactly one
   (`gorilla/websocket`). Implement `errgroup`-equivalent semantics inline
   — see `pkg/worker/pool.go` for the template.

8. **No `time.Sleep` in production code.** Zero calls today; every wait
   uses `select { case <-ctx.Done(): ... case <-time.After(...): ... }`.

## Adding state to `Runtime` — the checklist

If you add a new field to `Runtime`:

- [ ] It is owned by the single goroutine that owns the Runtime. Never
      expose it via a getter that other goroutines might call.
- [ ] `go test -race ./pkg/runtime/...` still passes.
- [ ] If the state survives between hunts (rare), document why.
- [ ] Add a worker-pool test in `pkg/worker/worker_test.go` that exercises
      the new state across ≥ 8 parallel adopted workers and asserts no
      bleed.

## Adding a goroutine — the checklist

Every new `go` statement needs answers to:

- [ ] **Lifetime:** when does it exit? Is it tied to a `context.Context`?
- [ ] **Cleanup:** is there a `defer wg.Done()` / `defer close(ch)` / `defer cancel()`?
- [ ] **Panics:** is it guarded by `defer recover()` if it runs arbitrary
      user code?
- [ ] **Channels:** does every send partner with a matching receive even
      under shutdown?
- [ ] **Tests:** does the race detector pass, including under cancellation?

Example — the parent-ctx watchdog in `cdp.Conn`:

```go
go func() {
    select {
    case <-ctx.Done():       // parent cancelled → tear down
        _ = c.Close()
    case <-connCtx.Done():   // we closed normally → exit
    }
}()
```

## Adding a package-level `var` — the checklist

- [ ] Is it actually needed, or can it live on a struct?
- [ ] If shared: is it mutex-guarded, atomic, or immutable?
- [ ] Is the intended lifecycle documented (e.g. "register at init, freeze
      before pool spawns")?
- [ ] Are tests isolated? (`resetRuntimeRegistries()` is the pattern for
      test cleanup.)

## When reviewing a PR that touches concurrent code

Quick smell test:

- Unbounded slice of channels → leak.
- `defer c.Unsubscribe(ch)` style → outdated; should be `defer sub.Close()`.
- `rt.vars.Something(...)` from two goroutines → data race.
- `for { ... time.Sleep(...) ... }` → blocking; refactor to select on ctx.
- `sync.Map` → almost always the wrong choice here; prefer `RWMutex + map`
  for the few genuinely shared structures we have.

## Per-worker logging

Derive a child logger for each worker using `utils.WithPrefix`. It shares the
parent's writer and level but prepends a `[wN]` tag to every line:

```go
workerLog := utils.WithPrefix(parentLogger, fmt.Sprintf("[w%d] ", id))
```

All `pkg/worker` code routes through this prefix — do not construct a fresh
`NewLogger` per worker (that would split the output stream).

## Key files

- [pkg/cdp/conn.go](../../../pkg/cdp/conn.go) — transport; `Conn`, `Subscription`.
- [pkg/worker/worker.go](../../../pkg/worker/worker.go) — `Worker` lifecycle.
- [pkg/worker/pool.go](../../../pkg/worker/pool.go) — `WorkerPool` dispatch.
- [pkg/worker/portalloc.go](../../../pkg/worker/portalloc.go) — `PortAllocator`.
- [pkg/runtime/extensions.go](../../../pkg/runtime/extensions.go) — registry freeze policy.
- [pkg/utils/logger.go](../../../pkg/utils/logger.go) — `WithPrefix(parent, "[wN] ")` for
  per-worker log prefixes; `NewLogger(logFile)` for dual stdout+ANSI-stripped file output.
