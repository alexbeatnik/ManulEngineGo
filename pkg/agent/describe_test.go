package agent

import (
	"strings"
	"testing"
)

func TestDescribeForLLM_Success(t *testing.T) {
	o := StepOutcome{OK: true, Step: "CLICK 'Login'", Reason: ReasonOK, URL: "https://x/account"}
	out := o.DescribeForLLM()
	if !strings.HasPrefix(out, "OK: CLICK 'Login'") {
		t.Errorf("success line wrong: %q", out)
	}
	if !strings.Contains(out, "https://x/account") {
		t.Errorf("resulting URL missing: %q", out)
	}
	if strings.Contains(out, "WARNING") {
		t.Errorf("confident success must not warn: %q", out)
	}
}

func TestDescribeForLLM_WeakSuccessWarns(t *testing.T) {
	o := StepOutcome{
		OK: true, Step: "CLICK 'More information'", Reason: ReasonOK, Score: 0.26,
		Near: []Cand{{Text: "Learn more", Score: 0.26}},
	}
	out := o.DescribeForLLM()
	if !strings.Contains(out, "WARNING") || !strings.Contains(out, "WRONG element") {
		t.Errorf("weak match must warn: %q", out)
	}
	if !strings.Contains(out, "'Learn more' (0.26)") {
		t.Errorf("near candidates must be quoted with scores: %q", out)
	}
}

func TestDescribeForLLM_AmbiguousFailureExplains(t *testing.T) {
	o := StepOutcome{
		OK: false, Step: "CLICK 'перше відео'", Reason: ReasonAmbiguous,
		Error: "target resolution too ambiguous (confidence 0.033, runner-up 0.033)",
		Near:  []Cand{{Text: "Хімера — Love Will Find You", Score: 0.18}},
	}
	out := o.DescribeForLLM()
	if !strings.HasPrefix(out, "FAILED: CLICK 'перше відео'") {
		t.Errorf("failure must echo the step: %q", out)
	}
	if !strings.Contains(out, "too vague") || !strings.Contains(out, "EXACT visible text") {
		t.Errorf("ambiguous advice missing: %q", out)
	}
	if !strings.Contains(out, "'Хімера — Love Will Find You' (0.18)") {
		t.Errorf("real candidates must be listed: %q", out)
	}
}

func TestDescribeForLLM_NotFound(t *testing.T) {
	o := StepOutcome{OK: false, Step: "CLICK 'Nonexistent'", Reason: ReasonNotFound}
	out := o.DescribeForLLM()
	if !strings.Contains(out, "does not exist") || !strings.Contains(out, "Do NOT retry") {
		t.Errorf("not-found advice missing: %q", out)
	}
}

func TestRunOutcomeRenderForLLM(t *testing.T) {
	r := RunOutcome{
		OK: false, URL: "https://x/results", TotalSteps: 3, Passed: 2, Failed: 1,
		Results: []StepOutcome{
			{OK: true, Step: "NAVIGATE to https://x", Reason: ReasonOK},
			{OK: true, Step: "FILL 'Search' field with 'cats'", Reason: ReasonOK},
			{OK: false, Step: "CLICK 'first result'", Reason: ReasonAmbiguous},
		},
	}
	out := r.RenderForLLM()
	if !strings.HasPrefix(out, "FAILED: ran 3 step(s) — 2 passed, 1 failed.") {
		t.Errorf("verdict line wrong: %q", out)
	}
	if !strings.Contains(out, "https://x/results") {
		t.Errorf("final URL missing: %q", out)
	}
	if !strings.Contains(out, "STOPPED at the first failed step") {
		t.Errorf("stop-explanation missing: %q", out)
	}
	if !strings.Contains(out, "OK: NAVIGATE to https://x") ||
		!strings.Contains(out, "FAILED: CLICK 'first result'") {
		t.Errorf("per-step lines missing: %q", out)
	}
}

func TestDescribePageChange(t *testing.T) {
	same := PageState{Title: "Results", URL: "https://x/results"}
	out := DescribePageChange(same, same)
	if !strings.Contains(out, "did NOT change") || !strings.Contains(out, "NO effect") {
		t.Errorf("no-change case must be explicit: %q", out)
	}

	out = DescribePageChange(same, PageState{Title: "Video", URL: "https://x/watch"})
	if !strings.Contains(out, "CHANGED") ||
		!strings.Contains(out, "https://x/results") || !strings.Contains(out, "https://x/watch") {
		t.Errorf("change case must show before/after: %q", out)
	}
}
