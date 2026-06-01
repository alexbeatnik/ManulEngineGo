package runtime

import (
	"testing"

	"github.com/alexbeatnik/ManulHeart/pkg/config"
	"github.com/alexbeatnik/ManulHeart/pkg/dsl"
	"github.com/alexbeatnik/ManulHeart/pkg/utils"
)

// ---- scoreToConfidence -------------------------------------------------------

func TestScoreToConfidence(t *testing.T) {
	cases := []struct {
		score float64
		want  int
	}{
		{0.0, 0},
		{-1.0, 0},
		{0.001, 1},  // > 0 but < 0.01
		{0.009, 1},
		{0.01, 3},   // >= 0.01
		{0.049, 3},
		{0.05, 5},   // >= 0.05
		{0.099, 5},
		{0.1, 7},    // >= 0.1
		{0.499, 7},
		{0.5, 9},    // >= 0.5
		{0.999, 9},
		{1.0, 10},   // >= 1.0
		{1.5, 10},   // over 1.0 still 10
	}
	for _, tc := range cases {
		got := scoreToConfidence(tc.score)
		if got != tc.want {
			t.Errorf("scoreToConfidence(%v) = %d want %d", tc.score, got, tc.want)
		}
	}
}

// ---- shouldPause ------------------------------------------------------------

func newTestRuntime(cfg config.Config) *Runtime {
	var buf noopWriter
	logger := utils.NewLogger(&buf)
	return New(cfg, &MockPage{}, logger)
}

// noopWriter satisfies io.Writer for constructing a Logger without real output.
type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestShouldPause_DebugContinue(t *testing.T) {
	rt := newTestRuntime(config.Default())
	rt.debugContinue = true

	// Even with empty breakLines (pause-every-step mode) debugContinue wins.
	cmd := dsl.Command{LineNum: 5}
	if rt.shouldPause(cmd, 0) {
		t.Error("shouldPause should return false when debugContinue=true")
	}
}

func TestShouldPause_EmptyBreakLines_PausesEveryStep(t *testing.T) {
	cfg := config.Default()
	// No BreakLines → empty map → pause on every step.
	rt := newTestRuntime(cfg)

	for _, lineNum := range []int{1, 10, 99, 0} {
		cmd := dsl.Command{LineNum: lineNum}
		if !rt.shouldPause(cmd, 0) {
			t.Errorf("shouldPause should return true for line %d when breakLines is empty", lineNum)
		}
	}
}

func TestShouldPause_SpecificBreakLines(t *testing.T) {
	cfg := config.Default()
	cfg.BreakLines = []int{10, 20}
	rt := newTestRuntime(cfg)

	cases := []struct {
		lineNum int
		want    bool
	}{
		{10, true},
		{20, true},
		{5, false},
		{15, false},
		{0, false},
	}
	for _, tc := range cases {
		cmd := dsl.Command{LineNum: tc.lineNum}
		got := rt.shouldPause(cmd, 0)
		if got != tc.want {
			t.Errorf("shouldPause(line=%d) = %v want %v", tc.lineNum, got, tc.want)
		}
	}
}

func TestShouldPause_DebugContinue_StopsAtBreakLine(t *testing.T) {
	cfg := config.Default()
	cfg.BreakLines = []int{10, 20}
	rt := newTestRuntime(cfg)
	rt.debugContinue = true

	// debugContinue free-runs until it hits a user-set breakpoint line.
	if !rt.shouldPause(dsl.Command{LineNum: 10}, 0) {
		t.Error("shouldPause should return true at breakLine 10 even when debugContinue=true")
	}
	// shouldPause resets debugContinue so the next normal pause cycle works.
	if rt.debugContinue {
		t.Error("shouldPause should reset debugContinue after hitting a breakLine")
	}
}

func TestShouldPause_DebugContinue_SkipsNonBreakLines(t *testing.T) {
	cfg := config.Default()
	cfg.BreakLines = []int{10, 20}
	rt := newTestRuntime(cfg)
	rt.debugContinue = true

	// Lines not in breakLines must not pause.
	for _, lineNum := range []int{1, 5, 15, 99} {
		cmd := dsl.Command{LineNum: lineNum}
		if rt.shouldPause(cmd, 0) {
			t.Errorf("shouldPause(line=%d) should be false during free-run", lineNum)
		}
	}
}

func TestShouldPause_BreakSteps(t *testing.T) {
	cfg := config.Default()
	cfg.BreakLines = []int{10} // non-empty so we don't enter pause-every-step mode
	rt := newTestRuntime(cfg)
	rt.breakSteps = map[int]bool{3: true}

	// Index 3 must pause; indices that aren't in breakSteps or breakLines must not.
	if !rt.shouldPause(dsl.Command{LineNum: 99}, 3) {
		t.Error("shouldPause should return true for breakSteps index 3")
	}
	if rt.shouldPause(dsl.Command{LineNum: 99}, 2) {
		t.Error("shouldPause should return false for index 2 (not in breakSteps or breakLines)")
	}
}

// ---- token state mutations ---------------------------------------------------
//
// The "next", "continue", and "debug-stop" debug tokens each mutate Runtime
// state before returning control to the execution loop. These tests verify
// that state directly, without needing to pipe stdin.

// nextTokenState replicates the state mutation that "next" performs.
func nextTokenState(rt *Runtime, idx int) {
	if rt.breakSteps == nil {
		rt.breakSteps = make(map[int]bool)
	}
	rt.breakSteps[idx+1] = true
}

// continueTokenState replicates the state mutation that "continue" performs.
func continueTokenState(rt *Runtime) {
	rt.debugContinue = true
	rt.breakSteps = make(map[int]bool)
}

// debugStopTokenState replicates the state mutation that "debug-stop" performs.
func debugStopTokenState(rt *Runtime) {
	rt.debugContinue = true
	rt.breakLines = make(map[int]bool)
	rt.breakSteps = make(map[int]bool)
}

func TestNextToken_PausesAtNextStep(t *testing.T) {
	cfg := config.Default()
	cfg.BreakLines = []int{99} // non-empty; idx+1 is not in breakLines
	rt := newTestRuntime(cfg)

	nextTokenState(rt, 2) // simulates "next" while paused at step 2

	// Step 3 (idx 3) must now trigger a pause.
	if !rt.shouldPause(dsl.Command{LineNum: 1}, 3) {
		t.Error("shouldPause should return true at idx 3 after 'next' from idx 2")
	}
	// Step 4 must not.
	if rt.shouldPause(dsl.Command{LineNum: 1}, 4) {
		t.Error("shouldPause should return false at idx 4 after 'next' from idx 2")
	}
}

func TestContinueToken_PreservesBreakLines(t *testing.T) {
	cfg := config.Default()
	cfg.BreakLines = []int{20}
	rt := newTestRuntime(cfg)
	// Simulate a one-shot breakStep that "continue" must clear.
	rt.breakSteps = map[int]bool{5: true}

	continueTokenState(rt)

	// breakLines must still cause a pause — continue only clears breakSteps.
	if !rt.shouldPause(dsl.Command{LineNum: 20}, 99) {
		t.Error("shouldPause should return true at breakLine 20 after 'continue'")
	}
	// The one-shot step advance must be gone.
	if rt.shouldPause(dsl.Command{LineNum: 1}, 5) {
		t.Error("shouldPause should return false at old breakStep 5 after 'continue'")
	}
}

func TestContinueToken_SetsDebugContinue(t *testing.T) {
	rt := newTestRuntime(config.Default())

	continueTokenState(rt)

	if !rt.debugContinue {
		t.Error("'continue' must set debugContinue so execution free-runs to the next breakpoint")
	}
}

func TestDebugStopToken_SuppressesAllPauses(t *testing.T) {
	cfg := config.Default()
	cfg.BreakLines = []int{10, 20}
	rt := newTestRuntime(cfg)
	rt.breakSteps = map[int]bool{3: true}

	debugStopTokenState(rt)

	// Neither a breakLine nor a breakStep must pause after debug-stop.
	for _, lineNum := range []int{10, 20, 0, 99} {
		cmd := dsl.Command{LineNum: lineNum}
		if rt.shouldPause(cmd, 3) {
			t.Errorf("shouldPause(line=%d,idx=3) should be false after 'debug-stop'", lineNum)
		}
	}
}

func TestConfidenceLabel(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{0.0, "none"},
		{-1.0, "none"},
		{0.001, "low"},
		{0.099, "low"},
		{0.1, "medium"},
		{0.499, "medium"},
		{0.5, "high"},
		{1.0, "high"},
	}
	for _, tc := range cases {
		got := confidenceLabel(tc.score)
		if got != tc.want {
			t.Errorf("confidenceLabel(%v) = %q want %q", tc.score, got, tc.want)
		}
	}
}
