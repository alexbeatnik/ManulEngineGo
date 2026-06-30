package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/config"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
	"github.com/alexbeatnik/ManulEngineGo/pkg/utils"
)

func TestRuntime_SetupRunsBeforeMission(t *testing.T) {
	ResetRuntimeRegistries()
	t.Cleanup(ResetRuntimeRegistries)

	var executionOrder []string
	err := RegisterGoCall("test.setup", func(ctx context.Context, invocation GoCallInvocation) (any, error) {
		executionOrder = append(executionOrder, "setup")
		return "setup-value", nil
	})
	if err != nil {
		t.Fatalf("RegisterGoCall failed: %v", err)
	}

	err = RegisterGoCall("test.mission", func(ctx context.Context, invocation GoCallInvocation) (any, error) {
		executionOrder = append(executionOrder, "mission")
		return "mission-value", nil
	})
	if err != nil {
		t.Fatalf("RegisterGoCall failed: %v", err)
	}

	src := `
[SETUP]
    CALL GO test.setup into {setup_var}
[END SETUP]

STEP 1:
    CALL GO test.mission into {mission_var}
`
	hunt, parseErr := dsl.Parse(strings.NewReader(src))
	if parseErr != nil {
		t.Fatalf("Parse failed: %v", parseErr)
	}

	mock := &MockPage{}
	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	_, runErr := rt.RunHunt(context.Background(), hunt)
	if runErr != nil {
		t.Fatalf("RunHunt failed: %v", runErr)
	}

	if len(executionOrder) != 2 {
		t.Fatalf("expected 2 executions, got %d: %v", len(executionOrder), executionOrder)
	}
	if executionOrder[0] != "setup" {
		t.Errorf("first execution = %q, want setup", executionOrder[0])
	}
	if executionOrder[1] != "mission" {
		t.Errorf("second execution = %q, want mission", executionOrder[1])
	}

	// Variables set during setup must persist into mission scope.
	if val, ok := rt.vars.Resolve("setup_var"); !ok || val != "setup-value" {
		t.Errorf("setup_var = %q, ok=%v, want setup-value", val, ok)
	}
	if val, ok := rt.vars.Resolve("mission_var"); !ok || val != "mission-value" {
		t.Errorf("mission_var = %q, ok=%v, want mission-value", val, ok)
	}
}

func TestRuntime_TeardownAlwaysRuns(t *testing.T) {
	ResetRuntimeRegistries()
	t.Cleanup(ResetRuntimeRegistries)

	var teardownRan bool
	err := RegisterGoCall("test.fail", func(ctx context.Context, invocation GoCallInvocation) (any, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("RegisterGoCall failed: %v", err)
	}

	err = RegisterGoCall("test.teardown", func(ctx context.Context, invocation GoCallInvocation) (any, error) {
		teardownRan = true
		return nil, nil
	})
	if err != nil {
		t.Fatalf("RegisterGoCall failed: %v", err)
	}

	src := `
[SETUP]
    CALL GO test.fail
[END SETUP]

STEP 1:
    CALL GO test.fail

[TEARDOWN]
    CALL GO test.teardown
[END TEARDOWN]
`
	hunt, parseErr := dsl.Parse(strings.NewReader(src))
	if parseErr != nil {
		t.Fatalf("Parse failed: %v", parseErr)
	}

	mock := &MockPage{}
	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	_, _ = rt.RunHunt(context.Background(), hunt)

	if !teardownRan {
		t.Error("expected teardown to run even though setup and mission succeeded")
	}
}

func TestRuntime_SetupFailureAbortsMission(t *testing.T) {
	ResetRuntimeRegistries()
	t.Cleanup(ResetRuntimeRegistries)

	var missionRan bool
	err := RegisterGoCall("test.fail", func(ctx context.Context, invocation GoCallInvocation) (any, error) {
		return nil, context.Canceled
	})
	if err != nil {
		t.Fatalf("RegisterGoCall failed: %v", err)
	}

	err = RegisterGoCall("test.mission", func(ctx context.Context, invocation GoCallInvocation) (any, error) {
		missionRan = true
		return nil, nil
	})
	if err != nil {
		t.Fatalf("RegisterGoCall failed: %v", err)
	}

	src := `
[SETUP]
    CALL GO test.fail
[END SETUP]

STEP 1:
    CALL GO test.mission
`
	hunt, parseErr := dsl.Parse(strings.NewReader(src))
	if parseErr != nil {
		t.Fatalf("Parse failed: %v", parseErr)
	}

	mock := &MockPage{}
	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	_, runErr := rt.RunHunt(context.Background(), hunt)
	if runErr == nil {
		t.Fatal("expected RunHunt to fail because setup failed")
	}

	if missionRan {
		t.Error("expected mission to be skipped when setup fails")
	}
}

func TestRuntime_TeardownRunsAfterSetupFailure(t *testing.T) {
	ResetRuntimeRegistries()
	t.Cleanup(ResetRuntimeRegistries)

	var teardownRan bool
	err := RegisterGoCall("test.fail", func(ctx context.Context, invocation GoCallInvocation) (any, error) {
		return nil, context.Canceled
	})
	if err != nil {
		t.Fatalf("RegisterGoCall failed: %v", err)
	}

	err = RegisterGoCall("test.teardown", func(ctx context.Context, invocation GoCallInvocation) (any, error) {
		teardownRan = true
		return nil, nil
	})
	if err != nil {
		t.Fatalf("RegisterGoCall failed: %v", err)
	}

	src := `
[SETUP]
    CALL GO test.fail
[END SETUP]

[TEARDOWN]
    CALL GO test.teardown
[END TEARDOWN]
`
	hunt, parseErr := dsl.Parse(strings.NewReader(src))
	if parseErr != nil {
		t.Fatalf("Parse failed: %v", parseErr)
	}

	mock := &MockPage{}
	rt := New(config.Config{}, mock, utils.NewLogger(nil))
	_, _ = rt.RunHunt(context.Background(), hunt)

	if !teardownRan {
		t.Error("expected teardown to run even when setup fails")
	}
}
