package runtime

import (
	"context"
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/config"
	"github.com/alexbeatnik/ManulHeart/pkg/dom"
	"github.com/alexbeatnik/ManulHeart/pkg/dsl"
	"github.com/alexbeatnik/ManulHeart/pkg/explain"
	"github.com/alexbeatnik/ManulHeart/pkg/utils"
)

func TestRuntime_Conditionals(t *testing.T) {
	mock := &MockPage{
		Elements: []dom.ElementSnapshot{
			{ID: 1, Tag: "button", VisibleText: "Save", IsVisible: true, Rect: dom.Rect{Top: 10, Left: 10, Width: 100, Height: 30}},
		},
	}
	mock.Elements[0].Normalize()

	cfg := config.Config{}
	logger := utils.NewLogger(nil)
	rt := New(cfg, mock, logger)
	ctx := context.Background()

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"ExistsTrue", "button 'Save' exists", true},
		{"ExistsFalse", "button 'Cancel' exists", false},
		{"NotExistsTrue", "button 'Cancel' not exists", true},
		{"NotExistsFalse", "button 'Save' not exists", false},
		{"TextPresentTrue", "text 'Save' is present", true},
		{"TextPresentFalse", "text 'Cancel' is present", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := rt.evaluateCondition(ctx, tc.condition)
			if err != nil {
				t.Fatalf("evaluateCondition failed: %v", err)
			}
			if got != tc.expected {
				t.Errorf("condition %q: got %v, want %v", tc.condition, got, tc.expected)
			}
		})
	}
}

func TestRuntime_Loops(t *testing.T) {
	mock := &MockPage{}
	cfg := config.Config{}
	logger := utils.NewLogger(nil)
	rt := New(cfg, mock, logger)
	ctx := context.Background()

	// 1. REPEAT loop
	hunt := &dsl.Hunt{
		Commands: []dsl.Command{
			{
				Type: dsl.CmdRepeat,
				RepeatCount: 3,
				RepeatVar: "i",
				Body: []dsl.Command{
					{Type: dsl.CmdPrint, PrintText: "Iteration {i}"},
				},
			},
		},
	}

	_, err := rt.RunHunt(ctx, hunt)
	if err != nil {
		t.Fatalf("REPEAT hunt failed: %v", err)
	}
	// Verify i was 2 after the loop
	val, _ := rt.vars.Resolve("i")
	if val != "2" {
		t.Errorf("expected i=2, got %q", val)
	}

	// 2. WHILE loop
	rt.vars.Set("counter", "0", LevelGlobal)
	huntWhile := &dsl.Hunt{
		Commands: []dsl.Command{
			{
				Type: dsl.CmdWhile,
				WhileCondition: "{counter} != '3'",
				Body: []dsl.Command{
					{Type: dsl.CmdPrint, PrintText: "While {counter}"},
					{Type: dsl.CmdSet, SetVar: "counter", SetValue: "3"}, // Break manually for now as we don't have math
				},
			},
		},
	}
	_, err = rt.RunHunt(ctx, huntWhile)
	if err != nil {
		t.Fatalf("WHILE hunt failed: %v", err)
	}

	// 3. FOR EACH loop
	rt.vars.Set("products", "Laptop, Headphones, Mouse", LevelGlobal)
	huntForEach := &dsl.Hunt{
		Commands: []dsl.Command{
			{
				Type:              dsl.CmdForEach,
				ForEachVar:        "product",
				ForEachCollection: "products",
				Body: []dsl.Command{
					{Type: dsl.CmdPrint, PrintText: "Product: {product}"},
				},
			},
		},
	}
	_, err = rt.RunHunt(ctx, huntForEach)
	if err != nil {
		t.Fatalf("FOR EACH hunt failed: %v", err)
	}
	// Verify last product was Mouse
	val, _ = rt.vars.Resolve("product")
	if val != "Mouse" {
		t.Errorf("expected product=Mouse, got %q", val)
	}

	// 4. Nested loops
	huntNested := &dsl.Hunt{
		Commands: []dsl.Command{
			{
				Type:        dsl.CmdRepeat,
				RepeatCount: 2,
				RepeatVar:   "i",
				Body: []dsl.Command{
					{
						Type:              dsl.CmdForEach,
						ForEachVar:        "item",
						ForEachCollection: "products",
						Body: []dsl.Command{
							{Type: dsl.CmdPrint, PrintText: "i={i} item={item}"},
						},
					},
				},
			},
		},
	}
	_, err = rt.RunHunt(ctx, huntNested)
	if err != nil {
		t.Fatalf("Nested loop hunt failed: %v", err)
	}
}

// TestRuntime_LoopBodyStepsRecorded guards against loop/IF bodies executing
// their steps without recording them in the HuntResult — which previously
// dropped every body step from reports and the onStep callback.
func TestRuntime_LoopBodyStepsRecorded(t *testing.T) {
	mock := &MockPage{}
	cfg := config.Config{}
	logger := utils.NewLogger(nil)
	rt := New(cfg, mock, logger)
	ctx := context.Background()

	var streamed int
	rt.SetOnStep(func(explain.ExecutionResult) { streamed++ })

	hunt := &dsl.Hunt{
		Commands: []dsl.Command{
			{
				Type:        dsl.CmdRepeat,
				RepeatCount: 3,
				RepeatVar:   "i",
				Body: []dsl.Command{
					{Type: dsl.CmdPrint, PrintText: "Iteration {i}"},
				},
			},
		},
	}

	res, err := rt.RunHunt(ctx, hunt)
	if err != nil {
		t.Fatalf("hunt failed: %v", err)
	}

	// 3 body PRINTs + 1 REPEAT container step.
	const wantSteps = 4
	if len(res.Results) != wantSteps {
		t.Errorf("expected %d recorded results, got %d", wantSteps, len(res.Results))
	}
	if res.TotalSteps != wantSteps {
		t.Errorf("expected TotalSteps=%d, got %d", wantSteps, res.TotalSteps)
	}
	if streamed != wantSteps {
		t.Errorf("expected onStep to fire %d times, got %d", wantSteps, streamed)
	}
}
