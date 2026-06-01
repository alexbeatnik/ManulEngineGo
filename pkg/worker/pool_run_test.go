package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexbeatnik/ManulHeart/pkg/config"
	"github.com/alexbeatnik/ManulHeart/pkg/dsl"
	"github.com/alexbeatnik/ManulHeart/pkg/runtime"
)

func navigateHunt(t *testing.T, url string) *dsl.Hunt {
	t.Helper()
	h, err := dsl.Parse(strings.NewReader(
		fmt.Sprintf("STEP 1: nav\n    NAVIGATE to '%s'\nDONE.\n", url)))
	if err != nil {
		t.Fatalf("parse hunt: %v", err)
	}
	return h
}

// perWorkerFactory returns a WorkerFactory where each worker invocation
// receives its own fresh MockPage, preventing cross-goroutine data races.
func perWorkerFactory() WorkerFactory {
	var mu sync.Mutex
	cursor := 0
	return func(ctx context.Context, opts Options) (*Worker, error) {
		mu.Lock()
		i := cursor
		cursor++
		mu.Unlock()
		page := &runtime.MockPage{
			URL:   fmt.Sprintf("https://example.test/w%d", i),
			Title: fmt.Sprintf("worker-%d", i),
		}
		return AdoptWorker(opts.ID, config.Default(), page, nil), nil
	}
}

// errFactory returns a WorkerFactory that always fails with the given message.
func errFactory(msg string) WorkerFactory {
	return func(_ context.Context, _ Options) (*Worker, error) {
		return nil, errors.New(msg)
	}
}

// TestPool_Run_OrderPreserved verifies that Pool.Run returns one result per
// input hunt, in the same order as the input slice, regardless of which
// worker processed it.
func TestPool_Run_OrderPreserved(t *testing.T) {
	const N = 7
	hunts := make([]*dsl.Hunt, N)
	for i := range hunts {
		hunts[i] = navigateHunt(t, fmt.Sprintf("https://example.test/%d", i))
	}

	pool, err := NewPool(PoolOptions{
		Concurrency: 3,
		Allocator:   NewPortAllocator(1, 10),
		Factory:     perWorkerFactory(),
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	results, runErr := pool.Run(ctx, hunts)
	if runErr != nil {
		t.Fatalf("Run: %v", runErr)
	}
	if len(results) != N {
		t.Fatalf("expected %d results, got %d", N, len(results))
	}
	for i, r := range results {
		if r.Hunt != hunts[i] {
			t.Fatalf("results[%d].Hunt mismatch: result contains wrong hunt pointer", i)
		}
		if r.Err != nil {
			t.Fatalf("results[%d].Err = %v", i, r.Err)
		}
	}
}

// TestPool_Run_AllSpawnFail verifies that when every worker fails to spawn,
// Run returns a non-nil error and every result carries a non-nil error.
// This exercises the post-wg.Wait backfill for hunts that were never processed.
func TestPool_Run_AllSpawnFail(t *testing.T) {
	hunts := []*dsl.Hunt{
		navigateHunt(t, "https://a.test"),
		navigateHunt(t, "https://b.test"),
		navigateHunt(t, "https://c.test"),
	}
	pool, err := NewPool(PoolOptions{
		Concurrency: 2,
		Allocator:   NewPortAllocator(1, 10),
		Factory:     errFactory("chrome failed to start"),
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	results, runErr := pool.Run(ctx, hunts)
	if runErr == nil {
		t.Fatal("expected non-nil error when all workers fail to spawn")
	}
	if len(results) != len(hunts) {
		t.Fatalf("expected %d results, got %d", len(hunts), len(results))
	}
	for i, r := range results {
		if r.Err == nil {
			t.Fatalf("results[%d].Err should be non-nil after all-spawn-failure", i)
		}
		if r.Hunt != hunts[i] {
			t.Fatalf("results[%d].Hunt mismatch after backfill", i)
		}
	}
}

// TestPool_Run_EmptyInput verifies that Run with no hunts returns immediately
// without error.
func TestPool_Run_EmptyInput(t *testing.T) {
	pool, _ := NewPool(PoolOptions{
		Concurrency: 2,
		Allocator:   NewPortAllocator(1, 10),
		Factory:     errFactory("should not be called"),
	})
	results, err := pool.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("empty run: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

// TestPool_Run_FailFast verifies that FailFast cancels in-flight hunts on the
// first failure.
func TestPool_Run_FailFast(t *testing.T) {
	// Create a hunt that will fail (bad command type)
	badHunt := &dsl.Hunt{
		Commands: []dsl.Command{
			{Type: dsl.CommandType("INVALID_COMMAND"), Raw: "INVALID"},
		},
	}
	goodHunt := navigateHunt(t, "https://example.test/good")

	hunts := []*dsl.Hunt{badHunt, goodHunt, goodHunt}

	pool, err := NewPool(PoolOptions{
		Concurrency: 3,
		Allocator:   NewPortAllocator(1, 10),
		Factory:     perWorkerFactory(),
		FailFast:    true,
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	results, runErr := pool.Run(ctx, hunts)
	if runErr == nil {
		t.Fatal("expected non-nil error with FailFast")
	}
	if len(results) != len(hunts) {
		t.Fatalf("expected %d results, got %d", len(hunts), len(results))
	}
	// At least the bad hunt should have an error
	if results[0].Err == nil {
		t.Fatal("expected bad hunt to fail")
	}
}

// TestPool_Run_PartialSpawnFail verifies that when some workers fail to spawn
// but others succeed, the successful workers still process their hunts.
func TestPool_Run_PartialSpawnFail(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	factory := func(_ context.Context, _ Options) (*Worker, error) {
		mu.Lock()
		calls++
		c := calls
		mu.Unlock()
		if c == 1 {
			return nil, errors.New("first worker fails")
		}
		page := &runtime.MockPage{URL: "https://example.test", Title: "ok"}
		return AdoptWorker(c, config.Default(), page, nil), nil
	}

	hunts := []*dsl.Hunt{
		navigateHunt(t, "https://a.test"),
		navigateHunt(t, "https://b.test"),
	}
	pool, err := NewPool(PoolOptions{
		Concurrency: 2,
		Allocator:   NewPortAllocator(1, 10),
		Factory:     factory,
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	results, runErr := pool.Run(ctx, hunts)
	// Should have at least one success and one error (either spawn error or context cancelled)
	if runErr == nil {
		t.Fatal("expected non-nil error with partial spawn failure")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// At least one result should have succeeded
	foundSuccess := false
	for _, r := range results {
		if r.Err == nil {
			foundSuccess = true
			break
		}
	}
	if !foundSuccess {
		t.Fatal("expected at least one successful result with partial spawn failure")
	}
}
