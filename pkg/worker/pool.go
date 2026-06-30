package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/alexbeatnik/ManulEngineGo/pkg/browser"
	"github.com/alexbeatnik/ManulEngineGo/pkg/config"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
	"github.com/alexbeatnik/ManulEngineGo/pkg/explain"
	"github.com/alexbeatnik/ManulEngineGo/pkg/utils"
)

// WorkerFactory constructs a Worker given a context and Options. The default
// factory (used when PoolOptions.Factory is nil) is NewWorker, which launches
// a real Chrome process. Override in tests to inject mock pages via AdoptWorker
// and avoid Chrome launches.
type WorkerFactory func(ctx context.Context, opts Options) (*Worker, error)

// PoolOptions configures a WorkerPool.
type PoolOptions struct {
	// Concurrency is the maximum number of Workers (and Chrome processes)
	// active at once. Required, must be >= 1.
	Concurrency int

	// Config is the engine-wide config inherited by every Worker.
	Config config.Config

	// Logger is the parent logger. Each Worker derives its own prefixed
	// child via utils.WithPrefix.
	Logger *utils.Logger

	// Allocator hands out CDP debug ports. Required.
	Allocator *PortAllocator

	// ChromeOptions are passed to each Worker's Chrome launch (Port is
	// always overridden by the Allocator).
	ChromeOptions browser.ChromeOptions

	// FailFast cancels the shared context as soon as any hunt errors,
	// causing other in-flight workers to abort their current step.
	// When false, all hunts run to completion regardless of failures.
	FailFast bool

	// Factory constructs Workers for the pool. If nil, NewWorker is used.
	// Override in tests to inject mock pages via AdoptWorker.
	Factory WorkerFactory
}

// PoolResult bundles a hunt's outcome with the worker that ran it.
type PoolResult struct {
	WorkerID int
	Hunt     *dsl.Hunt
	Result   *explain.HuntResult
	Err      error
}

// WorkerPool dispatches hunts to a bounded set of Workers running in parallel.
type WorkerPool struct {
	opts PoolOptions
}

// NewPool returns a WorkerPool with the given options.
func NewPool(opts PoolOptions) (*WorkerPool, error) {
	if opts.Concurrency < 1 {
		return nil, errors.New("worker: PoolOptions.Concurrency must be >= 1")
	}
	if opts.Allocator == nil {
		return nil, errors.New("worker: PoolOptions.Allocator is required")
	}
	return &WorkerPool{opts: opts}, nil
}

// Run executes every hunt across the worker pool. It returns one PoolResult
// per input hunt, in input order. The error return is non-nil if any hunt
// failed (the first error encountered, errors-style); per-hunt errors are
// also embedded in their PoolResult.
//
// If PoolOptions.FailFast is true, the context shared with all workers is
// cancelled on the first failure, so in-flight hunts will abort.
func (p *WorkerPool) Run(ctx context.Context, hunts []*dsl.Hunt) ([]PoolResult, error) {
	if len(hunts) == 0 {
		return nil, nil
	}

	results := make([]PoolResult, len(hunts))
	jobs := make(chan int, len(hunts))
	for i := range hunts {
		jobs <- i
	}
	close(jobs)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg       sync.WaitGroup
		errMu    sync.Mutex
		firstErr error
	)
	recordErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
		if p.opts.FailFast {
			cancel()
		}
	}

	factory := p.opts.Factory
	if factory == nil {
		factory = NewWorker
	}

	concurrency := p.opts.Concurrency
	if concurrency > len(hunts) {
		concurrency = len(hunts)
	}

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		workerSlot := w + 1
		go func() {
			defer wg.Done()
			worker, err := factory(runCtx, Options{
				ID:            workerSlot,
				Config:        p.opts.Config,
				Logger:        p.opts.Logger,
				Allocator:     p.opts.Allocator,
				ChromeOptions: p.opts.ChromeOptions,
			})
			if err != nil {
				// Don't drain the jobs channel — let other successfully-spawned
				// workers continue at reduced concurrency. Draining would race
				// with healthy goroutines and prevent them from processing hunts.
				err = fmt.Errorf("pool: spawn worker %d: %w", workerSlot, err)
				recordErr(err)
				return
			}
			defer worker.Close()

			for idx := range jobs {
				if runCtx.Err() != nil {
					results[idx] = PoolResult{
						WorkerID: worker.ID(),
						Hunt:     hunts[idx],
						Err:      runCtx.Err(),
					}
					recordErr(runCtx.Err())
					continue
				}
				res, runErr := worker.Run(runCtx, hunts[idx])
				results[idx] = PoolResult{
					WorkerID: worker.ID(),
					Hunt:     hunts[idx],
					Result:   res,
					Err:      runErr,
				}
				recordErr(runErr)
			}
		}()
	}

	wg.Wait()
	// Backfill any hunts that were never processed because all workers
	// failed to spawn before draining the jobs channel.
	if firstErr != nil {
		for i := range results {
			if results[i].Hunt == nil {
				results[i] = PoolResult{WorkerID: 0, Hunt: hunts[i], Err: firstErr}
			}
		}
	}
	return results, firstErr
}

// RunHuntsInParallel is a zero-config convenience wrapper around WorkerPool.Run.
// It constructs an internal PortAllocator over the inclusive range
// [9222, 9222+concurrency*2] and runs all hunts. For FailFast, custom port
// ranges, or a mock WorkerFactory use NewPool directly.
func RunHuntsInParallel(ctx context.Context, cfg config.Config, hunts []*dsl.Hunt, concurrency int, baseLogger *utils.Logger) ([]PoolResult, error) {
	alloc := NewPortAllocator(9222, 9222+concurrency*2)
	chromeOpts := browser.DefaultChromeOptions()
	chromeOpts.Headless = cfg.Headless
	if cfg.Channel != nil && *cfg.Channel != "" {
		chromeOpts.Channel = *cfg.Channel
	}
	if cfg.ExecutablePath != nil && *cfg.ExecutablePath != "" {
		chromeOpts.ExecutablePath = *cfg.ExecutablePath
	}
	pool, err := NewPool(PoolOptions{
		Concurrency:   concurrency,
		Config:        cfg,
		Logger:        baseLogger,
		Allocator:     alloc,
		ChromeOptions: chromeOpts,
	})
	if err != nil {
		return nil, err
	}
	return pool.Run(ctx, hunts)
}
