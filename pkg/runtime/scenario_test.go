package runtime

import (
	"context"
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/config"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
	"github.com/alexbeatnik/ManulEngineGo/pkg/utils"
)

func TestRuntime_ConditionalScenario(t *testing.T) {
	// 1. Setup Mock Page with specific elements
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{
				ID:          1,
				XPath:       "/button[1]",
				Tag:         "button",
				VisibleText: "Save",
				IsVisible:   true,
			},
		},
	}
	mock.Elements[0].Normalize()

	cfg := config.Config{}
	logger := utils.NewLogger(nil)
	rt := New(cfg, mock, logger)
	ctx := context.Background()

	// 2. Scenario: IF 'Save' exists Click it, ELSE click 'Cancel'
	hunt := &dsl.Hunt{
		Commands: []dsl.Command{
			{
				Type: dsl.CmdIf,
				Condition: "button 'Save' exists",
				Branches: []dsl.Branch{
					{
						Kind: "if",
						Condition: "button 'Save' exists",
						Body: []dsl.Command{
							{Type: dsl.CmdClick, Target: "Save", TypeHint: "button"},
						},
					},
					{
						Kind: "else",
						Body: []dsl.Command{
							{Type: dsl.CmdClick, Target: "Cancel", TypeHint: "button"},
						},
					},
				},
			},
		},
	}

	// 3. Run
	_, err := rt.RunHunt(ctx, hunt)
	if err != nil {
		t.Fatalf("RunHunt failed: %v", err)
	}

	// 4. Verify
	if len(mock.Clicks) != 1 {
		t.Errorf("expected 1 click, got %d", len(mock.Clicks))
	}
}

func TestRuntime_VariableExtractionScenario(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{
				ID:          1,
				XPath:       "/div[1]",
				Tag:         "div",
				VisibleText: "$99.99",
				IsVisible:   true,
			},
		},
	}
	mock.Elements[0].Normalize()

	cfg := config.Config{}
	logger := utils.NewLogger(nil)
	rt := New(cfg, mock, logger)
	ctx := context.Background()

	hunt := &dsl.Hunt{
		Commands: []dsl.Command{
			{Type: dsl.CmdExtract, Target: "div", ExtractVar: "price"},
			{Type: dsl.CmdPrint, PrintText: "The price is {price}"},
		},
	}

	_, err := rt.RunHunt(ctx, hunt)
	if err != nil {
		t.Fatalf("RunHunt failed: %v", err)
	}

	val, _ := rt.vars.Resolve("price")
	if val != "$99.99" {
		t.Errorf("expected $99.99, got %q", val)
	}
}
