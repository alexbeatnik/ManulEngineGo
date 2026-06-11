package agent

import (
	"fmt"
	"strings"
)

// This file renders execution outcomes as plain language for an LLM.
//
// The raw engine errors ("target resolution too ambiguous (confidence 0.033,
// runner-up 0.033)") are precise but opaque to small local models — they keep
// retrying the same broken label or apologize instead of correcting course.
// DescribeForLLM translates every outcome into three things even the weakest
// model can act on: WHAT ran, WHY it failed (in words, not scores), and WHAT
// TO DO NEXT (with exact copy-pastable labels when we have them).

// DescribeForLLM renders one step outcome as plain language with corrective
// advice. Successful steps come back as one short line; failures explain the
// cause and, when the scorer saw close-enough candidates, list their exact
// labels so the model can retarget without another page scan.
func (o StepOutcome) DescribeForLLM() string {
	step := strings.TrimSpace(o.Step)
	if step == "" {
		step = strings.ToUpper(o.Action)
	}

	var b strings.Builder
	if o.OK {
		fmt.Fprintf(&b, "OK: %s", step)
		if o.URL != "" {
			fmt.Fprintf(&b, " — the page is now %s", o.URL)
		}
		// A "success" with Near populated means the match was weak: the click
		// landed on the best fuzzy guess, which may be the wrong element.
		if len(o.Near) > 0 {
			fmt.Fprintf(&b,
				"\n  WARNING: the target matched only weakly (score %.2f), so this may have hit the WRONG element. "+
					"If the result looks wrong, do NOT repeat the same step — click one of these real elements instead: %s.",
				o.Score, nearLabelList(o.Near))
		}
		return b.String()
	}

	fmt.Fprintf(&b, "FAILED: %s", step)
	switch o.Reason {
	case ReasonNotFound:
		b.WriteString("\n  Why: NOTHING on the page matches that target text — the element does not exist (the label was probably made up). " +
			"Do NOT retry the same text.")
	case ReasonAmbiguous:
		b.WriteString("\n  Why: the target text is too vague — no single element matches it clearly. " +
			"Descriptions like 'first video' or 'the result' never work. Use the EXACT visible text of ONE element, copied verbatim.")
	case ReasonTimeout:
		b.WriteString("\n  Why: the page took too long to respond. It may still have loaded — check what page is open now before retrying.")
	case ReasonVerifyFailed:
		b.WriteString("\n  Why: the expected text or state is not on the page.")
	default:
		if o.Error != "" {
			fmt.Fprintf(&b, "\n  Why: %s", o.Error)
		}
	}
	if len(o.Near) > 0 {
		fmt.Fprintf(&b,
			"\n  Closest REAL elements on the page: %s. To act on one, copy its quoted text EXACTLY into your next command.",
			nearLabelList(o.Near))
	}
	return b.String()
}

// nearLabelList renders Near candidates as quoted, copy-pastable labels with
// their match scores: 'Login' (0.42), 'Log out' (0.18).
func nearLabelList(near []Cand) string {
	parts := make([]string, 0, len(near))
	for _, c := range near {
		parts = append(parts, fmt.Sprintf("'%s' (%.2f)", c.Text, c.Score))
	}
	return strings.Join(parts, ", ")
}

// RenderForLLM renders a whole run as a plain-language report: a one-line
// verdict, the final page, then each step via DescribeForLLM. When the run
// failed it states explicitly that later steps did NOT execute — small models
// otherwise assume the whole batch ran and report success.
func (r RunOutcome) RenderForLLM() string {
	var b strings.Builder
	verdict := "SUCCESS"
	if !r.OK {
		verdict = "FAILED"
	}
	fmt.Fprintf(&b, "%s: ran %d step(s) — %d passed, %d failed.", verdict, r.TotalSteps, r.Passed, r.Failed)
	if r.URL != "" {
		fmt.Fprintf(&b, " The page now open is: %s", r.URL)
	}
	for _, s := range r.Results {
		b.WriteString("\n")
		b.WriteString(s.DescribeForLLM())
	}
	if r.Failed > 0 {
		b.WriteString("\nThe run STOPPED at the first failed step — the steps after it did NOT run. " +
			"Fix only that step; do not re-run the steps that already passed.")
	}
	return b.String()
}

// DescribePageChange renders a before/after page comparison that ALWAYS says
// something — unlike DiffPageState, which returns "" when nothing changed.
// An explicit "nothing changed" line is exactly the signal a small model
// needs to realize its click was swallowed instead of declaring victory.
func DescribePageChange(before, after PageState) string {
	if before.Title == after.Title && before.URL == after.URL {
		where := after.URL
		if where == "" {
			where = "the same page"
		}
		return fmt.Sprintf("The page did NOT change — still %q at %s. The last action most likely had NO effect (the click was swallowed or the form never submitted).",
			after.Title, where)
	}
	var b strings.Builder
	b.WriteString("The page CHANGED:\n")
	fmt.Fprintf(&b, "  before: %q (%s)\n", before.Title, before.URL)
	fmt.Fprintf(&b, "  after:  %q (%s)", after.Title, after.URL)
	return b.String()
}
