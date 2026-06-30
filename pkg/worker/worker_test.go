package worker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexbeatnik/ManulEngineGo/pkg/config"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
	"github.com/alexbeatnik/ManulEngineGo/pkg/runtime"
	"github.com/alexbeatnik/ManulEngineGo/pkg/utils"
)

func mockHunt(t *testing.T, body string) *dsl.Hunt {
	t.Helper()
	hunt, err := dsl.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("parse hunt: %v", err)
	}
	return hunt
}

func mockPageWithButton(label string) *runtime.MockPage {
	return &runtime.MockPage{
		URL:   "https://example.test/" + label,
		Title: label,
		Elements: []dom.ElementSnapshot{
			{
				ID:          1,
				Tag:         "button",
				XPath:       "/html/body/button",
				VisibleText: label,
				Rect:        dom.Rect{Left: 10, Top: 10, Width: 100, Height: 30},
				IsVisible:   true,
			},
		},
	}
}

func TestAdoptWorker_RunsHunt(t *testing.T) {
	page := mockPageWithButton("Login")
	w := AdoptWorker(7, config.Default(), page, utils.NewLogger(nil))
	defer w.Close()

	if w.ID() != 7 {
		t.Fatalf("expected id 7, got %d", w.ID())
	}

	body := "STEP 1: smoke\n    Click the 'Login' button\nDONE.\n"
	hunt := mockHunt(t, body)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := w.Run(ctx, hunt)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success; failed=%d soft=%v", res.Failed, res.SoftErrors)
	}
	if len(page.Clicks) != 1 {
		t.Fatalf("expected 1 click on mock page, got %d", len(page.Clicks))
	}
}

// TestPool_ParallelExecutionNoStateBleed runs many hunts across many adopted
// workers in parallel and asserts that each worker's MockPage saw only its
// own clicks. Any state bleed (e.g. shared snapshot cache, shared variable
// store) would cause cross-page click counts or wrong elements clicked.
func TestPool_ParallelExecutionNoStateBleed(t *testing.T) {
	const nHunts = 16

	type job struct {
		hunt *dsl.Hunt
		page *runtime.MockPage
		want string
	}
	jobs := make([]job, nHunts)
	for i := 0; i < nHunts; i++ {
		label := fmt.Sprintf("Btn%d", i)
		jobs[i] = job{
			hunt: mockHunt(t, fmt.Sprintf(
				"STEP 1: smoke\n    Click the '%s' button\nDONE.\n", label)),
			page: mockPageWithButton(label),
			want: label,
		}
	}

	// Drive workers manually instead of using Pool.Run to avoid launching
	// Chrome. Pool.Run dispatch semantics are covered in pool_run_test.go
	// via the injected WorkerFactory.
	var wg sync.WaitGroup
	errs := make(chan error, nHunts)
	for i := 0; i < nHunts; i++ {
		wg.Add(1)
		go func(j job, id int) {
			defer wg.Done()
			w := AdoptWorker(id, config.Default(), j.page, nil)
			defer w.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			res, err := w.Run(ctx, j.hunt)
			if err != nil {
				errs <- fmt.Errorf("worker %d: %w", id, err)
				return
			}
			if !res.Success {
				errs <- fmt.Errorf("worker %d: hunt failed", id)
				return
			}
			if len(j.page.Clicks) != 1 {
				errs <- fmt.Errorf("worker %d: expected 1 click on %q, got %d",
					id, j.want, len(j.page.Clicks))
				return
			}
		}(jobs[i], i+1)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("%v", err)
	}
}

// TestPool_NewPoolValidation verifies constructor validation for PoolOptions.
// It asserts that NewPool rejects zero concurrency, rejects a missing
// allocator, and succeeds when given a valid configuration.
func TestPool_NewPoolValidation(t *testing.T) {
	if _, err := NewPool(PoolOptions{Concurrency: 0, Allocator: NewPortAllocator(1, 2)}); err == nil {
		t.Fatalf("expected error on zero concurrency")
	}
	if _, err := NewPool(PoolOptions{Concurrency: 1}); err == nil {
		t.Fatalf("expected error on missing allocator")
	}
	if _, err := NewPool(PoolOptions{Concurrency: 4, Allocator: NewPortAllocator(1, 2)}); err != nil {
		t.Fatalf("expected pool to construct: %v", err)
	}
}

// TestWorker_LogPrefix verifies per-worker log lines carry the [wN] prefix.
func TestWorker_LogPrefix(t *testing.T) {
	var sb strings.Builder
	parent := utils.NewLogger(&sb)
	child := utils.WithPrefix(parent, "[w42] ")
	child.Info("hello")
	if !strings.Contains(sb.String(), "[w42] hello") {
		t.Fatalf("expected log line to contain [w42] hello, got: %q", sb.String())
	}
}
