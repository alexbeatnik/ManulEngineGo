package runtime

import (
	"context"
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/config"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
	"github.com/alexbeatnik/ManulEngineGo/pkg/utils"
)

func TestRuntime_Variables(t *testing.T) {
	mock := &MockPage{}
	cfg := config.Config{}
	logger := utils.NewLogger(nil)
	rt := New(cfg, mock, logger)
	ctx := context.Background()

	// 1. Static variable initialization from Hunt
	hunt := &dsl.Hunt{
		Vars: map[string]string{
			"user_email": "admin@test.com",
			"password":   "secret123",
		},
		Commands: []dsl.Command{
			{Type: dsl.CmdPrint, PrintText: "Email: {user_email}"},
		},
	}

	_, err := rt.RunHunt(ctx, hunt)
	if err != nil {
		t.Fatalf("RunHunt failed: %v", err)
	}

	val, _ := rt.vars.Resolve("user_email")
	if val != "admin@test.com" {
		t.Errorf("expected admin@test.com, got %q", val)
	}

	// 2. Dynamic assignment via SET
	huntSet := &dsl.Hunt{
		Commands: []dsl.Command{
			{Type: dsl.CmdSet, SetVar: "new_var", SetValue: "hello"},
			{Type: dsl.CmdPrint, PrintText: "{new_var} world"},
		},
	}
	_, err = rt.RunHunt(ctx, huntSet)
	if err != nil {
		t.Fatalf("SET hunt failed: %v", err)
	}
	valNew, _ := rt.vars.Resolve("new_var")
	if valNew != "hello" {
		t.Errorf("expected hello, got %q", valNew)
	}

	// 3. Interpolation in resolution (resolveVariables)
	rt.vars.Set("target", "Login", LevelGlobal)
	resolved := rt.resolveVariables("Click {target} button")
	expected := "Click Login button"
	if resolved != expected {
		t.Errorf("expected %q, got %q", expected, resolved)
	}
}
