package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/config"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
	"github.com/alexbeatnik/ManulEngineGo/pkg/utils"
)

func TestRuntime_SetVar(t *testing.T) {
	mock := &MockPage{}
	cfg := config.Config{}
	logger := utils.NewLogger(nil)
	rt := New(cfg, mock, logger)
	ctx := context.Background()

	// Test case: SET {discount} = '10%'
	cmd := dsl.Command{
		Type:     dsl.CmdSet,
		SetVar:   "discount",
		SetValue: "10%",
	}

	_, err := rt.executeCommand(ctx, cmd)
	if err != nil {
		t.Fatalf("executeCommand failed: %v", err)
	}

	val, ok := rt.vars.Resolve("discount")
	if !ok || val != "10%" {
		t.Errorf("expected 10%%, got %s (ok=%v)", val, ok)
	}

	// Test case: SET {total} = '$100'
	cmd2 := dsl.Command{
		Type:     dsl.CmdSet,
		SetVar:   "total",
		SetValue: "$100",
	}
	_, err = rt.executeCommand(ctx, cmd2)
	if err != nil {
		t.Fatalf("executeCommand cmd2 failed: %v", err)
	}

	// Verify interpolation of previous variable
	sv := dsl.Command{
		Type:      dsl.CmdPrint,
		PrintText: "Total with discount: {total} (Applied {discount})",
	}
	text := rt.resolveVariables(sv.PrintText)
	expected := "Total with discount: $100 (Applied 10%)"
	if text != expected {
		t.Errorf("expected %q, got %q", expected, text)
	}
}

func TestRuntime_Indentation(t *testing.T) {
	// This tests the parser's ability to handle indented lines
	input := `
        NAVIGATE to https://example.com
        CLICK the "Login" button
        SET {user} = "Alex"
    `
	hunt, err := dsl.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(hunt.Commands) != 3 {
		t.Errorf("expected 3 commands, got %d", len(hunt.Commands))
	}

	if hunt.Commands[0].Type != dsl.CmdNavigate {
		t.Errorf("expected NAVIGATE, got %v", hunt.Commands[0].Type)
	}
	if hunt.Commands[1].Type != dsl.CmdClick {
		t.Errorf("expected CLICK, got %v", hunt.Commands[1].Type)
	}
	if hunt.Commands[2].Type != dsl.CmdSet {
		t.Errorf("expected SET, got %v", hunt.Commands[2].Type)
	}
}

func TestRuntime_VarScoping_MissionLevel(t *testing.T) {
	mock := &MockPage{}
	cfg := config.Config{}
	logger := utils.NewLogger(nil)
	rt := New(cfg, mock, logger)
	ctx := context.Background()

	hunt := &dsl.Hunt{
		Vars: map[string]string{
			"base_url": "https://example.com",
		},
		Commands: []dsl.Command{
			{Type: dsl.CmdNavigate, URL: "https://example.com"},
		},
	}

	_, err := rt.RunHunt(ctx, hunt)
	if err != nil {
		t.Fatalf("RunHunt failed: %v", err)
	}

	// Verify the variable is stored at LevelMission (3), not LevelGlobal (4).
	val, level, ok := rt.vars.ResolveLevel("base_url")
	if !ok {
		t.Fatal("expected base_url to be resolved")
	}
	if val != "https://example.com" {
		t.Errorf("base_url = %q, want https://example.com", val)
	}
	if level != LevelMission {
		t.Errorf("base_url scope level = %d, want LevelMission (%d)", level, LevelMission)
	}
}
