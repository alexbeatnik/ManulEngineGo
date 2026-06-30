package main

import (
	"context"
	"strings"
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/browser"
	"github.com/alexbeatnik/ManulEngineGo/pkg/config"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
	"github.com/alexbeatnik/ManulEngineGo/pkg/runtime"
	"github.com/alexbeatnik/ManulEngineGo/pkg/utils"
	"github.com/alexbeatnik/ManulEngineGo/pkg/worker"
)

// TestCustomControlAndCallGo demonstrates the full SDET workflow:
//   1. Register a Custom Control that intercepts a standard DSL step.
//   2. Register a CALL GO helper that returns a map (flattened into variables).
//   3. Run a hunt that exercises both extensions end-to-end.
func TestCustomControlAndCallGo(t *testing.T) {
	// Reset global registries so tests don't interfere with each other.
	runtime.ResetRuntimeRegistries()
	defer runtime.ResetRuntimeRegistries()

	// ── 1. Register Custom Control ───────────────────────────────────────────
	// Intercepts "FILL 'React Datepicker'" on the Checkout Page, bypassing
	// the heuristic DOM pipeline entirely.
	err := runtime.RegisterCustomControl("Checkout Page", "React Datepicker",
		func(ctx context.Context, page browser.Page, inv runtime.CustomControlInvocation) error {
			if inv.ActionType != "input" {
				t.Errorf("expected ActionType=input, got %q", inv.ActionType)
			}
			if inv.Value != "2026-12-25" {
				t.Errorf("expected Value=2026-12-25, got %q", inv.Value)
			}
			// In a real scenario we would interact with the live page.
			// For this test we just assert the invocation was routed correctly.
			return nil
		})
	if err != nil {
		t.Fatalf("RegisterCustomControl failed: %v", err)
	}

	// ── 2. Register CALL GO helper ──────────────────────────────────────────
	// A helper that simulates a DB setup call and returns a map.
	// Python manul-engine flattens dict returns into shared variables;
	// ManulEngine (Go) does the same for map[string]string / map[string]any.
	err = runtime.RegisterGoCall("db.setup",
		func(ctx context.Context, inv runtime.GoCallInvocation) (any, error) {
			return map[string]string{
				"admin_token": "tok_12345",
				"base_url":    "https://api.example.com",
			}, nil
		})
	if err != nil {
		t.Fatalf("RegisterGoCall failed: %v", err)
	}

	// ── 3. Build a mock page ────────────────────────────────────────────────
	// The page title is "Checkout Page" so the custom-control lookup matches.
	mockPage := &runtime.MockPage{
		URL:   "https://example-shop.com/checkout",
		Title: "Checkout Page",
		Elements: []dom.ElementSnapshot{
			{ID: 1, Tag: "input", Name: "React Datepicker", XPath: "//input[@id='custom-calendar-input']"},
		},
	}

	// ── 4. Adopt a Worker and run the hunt ──────────────────────────────────
	cfg := config.Default()
	w := worker.AdoptWorker(1, cfg, mockPage, utils.NewLogger(nil))
	defer w.Close()

	hunt, parseErr := dsl.Parse(strings.NewReader(`
STEP 1: Setup
    CALL GO db.setup into {setup_result}

STEP 2: Fill Date
    FILL 'React Datepicker' field with '2026-12-25'

DONE.
`))
	if parseErr != nil {
		t.Fatalf("parse hunt: %v", parseErr)
	}

	result, runErr := w.Run(context.Background(), hunt)
	if runErr != nil {
		t.Fatalf("run hunt: %v", runErr)
	}
	if !result.Success {
		t.Fatalf("hunt failed: %d passed, %d failed", result.Passed, result.Failed)
	}

	// ── 5. Verify flattened variables from CALL GO map return ───────────────
	rt := w.Runtime()
	if tok, ok := rt.ResolveVariable("admin_token"); !ok || tok != "tok_12345" {
		t.Fatalf("admin_token = %q (ok=%v), want tok_12345", tok, ok)
	}
	if url, ok := rt.ResolveVariable("base_url"); !ok || url != "https://api.example.com" {
		t.Fatalf("base_url = %q (ok=%v), want https://api.example.com", url, ok)
	}
	if setup, ok := rt.ResolveVariable("setup_result"); !ok || setup == "" {
		t.Fatalf("setup_result should be non-empty stringified map, got %q (ok=%v)", setup, ok)
	}
}

// TestCmdRead_RequiresTargetOrSelector verifies the read subcommand rejects an
// invocation with neither a target label nor a --selector before it tries to
// touch a browser (so the check is browser-free and fast).
func TestCmdRead_RequiresTargetOrSelector(t *testing.T) {
	err := cmdRead([]string{"--cdp", "http://127.0.0.1:65535"})
	if err == nil {
		t.Fatal("expected an error when no target and no --selector given")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("error should explain the requirement, got: %v", err)
	}
}
