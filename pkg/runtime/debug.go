package runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alexbeatnik/ManulHeart/pkg/dsl"
	"github.com/alexbeatnik/ManulHeart/pkg/scorer"
)

var ErrDebugStop = errors.New("debug: stop requested")

func (rt *Runtime) shouldPause(cmd dsl.Command, idx int) bool {
	if rt.debugContinue {
		// Free-run mode: only stop at user-set line breakpoints.
		if rt.breakLines[cmd.LineNum] {
			rt.debugContinue = false // re-arm for the next continue
			return true
		}
		return false
	}
	if len(rt.breakLines) == 0 && len(rt.breakSteps) == 0 {
		return true
	}
	if rt.breakLines[cmd.LineNum] {
		return true
	}
	if rt.breakSteps != nil && rt.breakSteps[idx] {
		return true
	}
	return false
}

func isTTY() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func (rt *Runtime) injectDebugModal(ctx context.Context, step string) error {
	stepJSON, _ := json.Marshal(step)
	js := `(function(stepText){
		const old = document.getElementById('manul-debug-modal');
		if (old) old.remove();
		window.__manul_debug_action = null;

		const modal = document.createElement('div');
		modal.id = 'manul-debug-modal';
		modal.setAttribute('data-manul-debug', 'true');
		modal.style.cssText = [
			'position:fixed', 'top:12px', 'right:12px', 'z-index:2147483647',
			'background:#1e1e2e', 'color:#cdd6f4',
			'border:2px solid #89b4fa', 'border-radius:8px',
			'padding:14px 40px 14px 16px',
			'font-family:monospace', 'font-size:13px',
			'max-width:420px', 'word-break:break-all',
			'box-shadow:0 4px 24px rgba(0,0,0,.55)',
			'pointer-events:all', 'user-select:none',
		].join(';');

		const label = document.createElement('div');
		label.style.cssText = 'font-weight:bold;color:#89b4fa;margin-bottom:6px;font-size:11px;letter-spacing:.06em;';
		label.textContent = '\uD83D\uDC3E MANUL DEBUG PAUSE';

		const text = document.createElement('div');
		text.style.cssText = 'line-height:1.5;';
		text.textContent = stepText;

		const btn = document.createElement('button');
		btn.id = 'manul-debug-abort';
		btn.textContent = '\u2715';
		btn.title = 'Abort test run';
		btn.style.cssText = [
			'position:absolute', 'top:8px', 'right:8px',
			'background:transparent', 'border:none',
			'color:#a6adc8', 'font-size:16px', 'font-weight:bold',
			'cursor:pointer', 'line-height:1', 'padding:2px 6px',
			'border-radius:4px', 'transition:background .15s,color .15s',
		].join(';');
		btn.onmouseover = function(){ btn.style.background='#f38ba8'; btn.style.color='#1e1e2e'; };
		btn.onmouseout  = function(){ btn.style.background='transparent'; btn.style.color='#a6adc8'; };
		btn.addEventListener('click', function(){ window.__manul_debug_action = 'ABORT'; });

		modal.appendChild(label);
		modal.appendChild(text);
		modal.appendChild(btn);
		document.body.appendChild(modal);
	})(` + string(stepJSON) + `)`
	_, err := rt.page.EvalJS(ctx, js)
	return err
}

func (rt *Runtime) removeDebugModal(ctx context.Context) error {
	js := `(() => {
		const m = document.getElementById('manul-debug-modal');
		if (m) m.remove();
		window.__manul_debug_action = null;
	})();`
	_, err := rt.page.EvalJS(ctx, js)
	return err
}

func (rt *Runtime) debugHighlight(ctx context.Context, xpath string) error {
	xpathJSON, _ := json.Marshal(xpath)
	js := `(function(){
		const styleId = 'manul-debug-style';
		const styleCss = "[data-manul-debug-highlight='true']{outline:4px solid #ff00ff !important;box-shadow:0 0 15px #ff00ff !important;background:rgba(255,0,255,.12) !important;z-index:999999 !important;}";
		if (!document.getElementById(styleId)) {
			const s = document.createElement('style');
			s.id = styleId;
			s.textContent = styleCss;
			document.head.appendChild(s);
		}
		const r = document.evaluate(` + string(xpathJSON) + `, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null);
		const el = r.singleNodeValue;
		if (el) {
			el.setAttribute('data-manul-debug-highlight', 'true');
			el.scrollIntoView({behavior:'smooth',block:'center'});
		}
	})();`
	_, err := rt.page.EvalJS(ctx, js)
	return err
}

func (rt *Runtime) clearDebugHighlight(ctx context.Context) error {
	js := `(() => {
		document.querySelectorAll('[data-manul-debug-highlight]').forEach(
			el => el.removeAttribute('data-manul-debug-highlight')
		);
		const s = document.getElementById('manul-debug-style');
		if (s) s.remove();
	})();`
	_, err := rt.page.EvalJS(ctx, js)
	return err
}

func scoreToConfidence(s float64) int {
	switch {
	case s >= 1.0:
		return 10
	case s >= 0.5:
		return 9
	case s >= 0.1:
		return 7
	case s >= 0.05:
		return 5
	case s >= 0.01:
		return 3
	case s > 0:
		return 1
	default:
		return 0
	}
}

func (rt *Runtime) explainStep(ctx context.Context, cmd dsl.Command) string {
	elements, err := rt.loadSnapshot(ctx)
	if err != nil {
		return fmt.Sprintf("explain: snapshot failed: %v", err)
	}

	query := cmd.Target
	if query == "" {
		query = cmd.Raw
	}
	mode := string(cmd.InteractionMode)
	if mode == "" {
		mode = string(dsl.ModeNone)
	}

	ranked := scorer.Rank(query, cmd.TypeHint, mode, elements, 5, nil)
	rt.lastExplainData = ranked

	if len(ranked) == 0 {
		return fmt.Sprintf("explain: no candidates found for %q", query)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "explain: top %d for %q\n", len(ranked), query)
	for i, c := range ranked {
		conf := scoreToConfidence(c.Explain.Score.Total)
		text := c.Element.VisibleText
		if len(text) > 60 {
			text = text[:57] + "..."
		}
		fmt.Fprintf(&sb, "  #%d score=%.3f conf=%d/10 <%s> %q\n      xpath=%s\n",
			i+1, c.Explain.Score.Total, conf, c.Element.Tag, text, c.Element.XPath)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (rt *Runtime) debugPrompt(ctx context.Context, cmd dsl.Command, idx int) error {
	if isTTY() {
		return rt.debugPromptTTY(ctx, cmd, idx)
	}
	return rt.debugPromptExtension(ctx, cmd, idx)
}

// debugPromptTTY drives the interactive readline prompt when stdin is a TTY.
// CONCURRENCY: no background goroutines — all page access stays on the caller's goroutine.
func (rt *Runtime) debugPromptTTY(ctx context.Context, cmd dsl.Command, idx int) error {
	if err := rt.injectDebugModal(ctx, cmd.Raw); err != nil {
		rt.logger.Warn("debug: modal inject failed: %v", err)
	}
	defer rt.removeDebugModal(ctx)

	sc := bufio.NewScanner(os.Stdin)
	// The extension line-buffer safety cap is 1 MB; match it so long tokens don't trigger ErrTooLong.
	sc.Buffer(make([]byte, 1024), 1024*1024)
	inputCh := make(chan string, 1)

	readNext := func() {
		go func() {
			rt.logger.Info("\n[DEBUG] paused at: %s", cmd.Raw)
			rt.logger.Info("  Commands: next | continue | debug-stop | highlight <xpath> | explain | abort")
			fmt.Fprint(os.Stdout, "  > ")
			os.Stdout.Sync()
			if sc.Scan() {
				inputCh <- strings.TrimSpace(sc.Text())
			} else {
				inputCh <- "next"
			}
		}()
	}
	readNext()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Poll the in-browser modal for an abort click — runs on the single Runtime goroutine.
			raw, err := rt.page.EvalJS(ctx, `window.__manul_debug_action||""`)
			if err != nil {
				continue
			}
			var action string
			if json.Unmarshal(raw, &action) == nil && action == "abort" {
				rt.logger.Warn("debug: abort from browser")
				return ErrDebugStop
			}
		case token := <-inputCh:
			switch {
			case token == "" || token == "next":
				rt.clearDebugHighlight(ctx)
				return nil
			case token == "continue":
				rt.debugContinue = true
				rt.breakSteps = make(map[int]bool)
				rt.clearDebugHighlight(ctx)
				return nil
			case token == "debug-stop":
				rt.debugContinue = true
				rt.breakLines = make(map[int]bool)
				rt.breakSteps = make(map[int]bool)
				rt.clearDebugHighlight(ctx)
				return nil
			case token == "abort":
				return ErrDebugStop
			case strings.HasPrefix(token, "highlight "):
				xpath := strings.TrimPrefix(token, "highlight ")
				if err := rt.debugHighlight(ctx, xpath); err != nil {
					rt.logger.Warn("debug: highlight failed: %v", err)
				}
				readNext()
			case token == "explain":
				rt.logger.Info(rt.explainStep(ctx, cmd))
				readNext()
			default:
				rt.logger.Warn("debug: unknown command %q — try: next, continue, debug-stop, highlight <xpath>, explain, abort", token)
				readNext()
			}
		}
	}
}

func confidenceLabel(score float64) string {
	switch {
	case score >= 0.5:
		return "high"
	case score >= 0.1:
		return "medium"
	case score > 0:
		return "low"
	default:
		return "none"
	}
}

// explainNextPayload implements the VS Code extension's ExplainNextResult
// TypeScript interface (contracts/EXTENSION_ENGINE_CONTRACT.md §3.5).
// All 10 fields are serialized on every emission; null-typed fields use
// pointer types so `encoding/json` can write JSON null.
type explainNextPayload struct {
	Step            string   `json:"step"`
	Score           float64  `json:"score"`
	ConfidenceLabel string   `json:"confidence_label"`
	TargetFound     bool     `json:"target_found"`
	TargetElement   *string  `json:"target_element"`
	Explanation     string   `json:"explanation"`
	Risk            string   `json:"risk"`
	Suggestion      *string  `json:"suggestion"`
	HeuristicScore  *float64 `json:"heuristic_score"`
	HeuristicMatch  *string  `json:"heuristic_match"`
}

func (rt *Runtime) buildExplainNextResult(ctx context.Context, stepText string, cmd dsl.Command) explainNextPayload {
	elements, err := rt.loadSnapshot(ctx)
	if err != nil {
		return explainNextPayload{
			Step:            stepText,
			Score:           0,
			ConfidenceLabel: "none",
			TargetFound:     false,
			Explanation:     fmt.Sprintf("snapshot failed: %v", err),
		}
	}

	query := cmd.Target
	if query == "" {
		query = stepText
	}
	mode := string(cmd.InteractionMode)
	if mode == "" {
		mode = string(dsl.ModeNone)
	}
	ranked := scorer.Rank(query, cmd.TypeHint, mode, elements, 5, nil)
	rt.lastExplainData = ranked

	if len(ranked) == 0 {
		return explainNextPayload{
			Step:            stepText,
			Score:           0,
			ConfidenceLabel: "none",
			TargetFound:     false,
			Explanation:     fmt.Sprintf("no candidates found for %q", query),
		}
	}

	top := ranked[0]
	topXPath := top.Element.XPath
	topScore := top.Explain.Score.Total

	label := confidenceLabel(topScore)
	sb := top.Explain.Score
	textCh := sb.ExactTextMatch + sb.NormalizedTextMatch + sb.LabelMatch + sb.PlaceholderMatch + sb.AriaMatch + sb.DataQAMatch
	semanticCh := sb.TagSemantics + sb.TypeHintAlignment
	penaltyCh := sb.VisibilityScore * sb.InteractabilityScore
	explanation := fmt.Sprintf(
		"top candidate <%s> score=%.3f (text=%.3f id=%.3f semantic=%.3f penalty=%.3f)",
		top.Element.Tag,
		topScore,
		textCh,
		sb.IDMatch,
		semanticCh,
		penaltyCh,
	)

	risk := ""
	var suggestion *string
	if topScore < 0.1 {
		risk = "low confidence — target may be ambiguous or missing"
		if len(ranked) > 1 {
			s := fmt.Sprintf("next candidate <%s> xpath=%s score=%.3f",
				ranked[1].Element.Tag, ranked[1].Element.XPath, ranked[1].Explain.Score.Total)
			suggestion = &s
		}
	}

	match := top.Element.VisibleText
	if match == "" {
		match = top.Element.Tag
	}

	return explainNextPayload{
		Step:            stepText,
		Score:           topScore,
		ConfidenceLabel: label,
		TargetFound:     topScore > 0,
		TargetElement:   &topXPath,
		Explanation:     explanation,
		Risk:            risk,
		Suggestion:      suggestion,
		HeuristicScore:  &topScore,
		HeuristicMatch:  &match,
	}
}

func (rt *Runtime) debugPromptExtension(ctx context.Context, cmd dsl.Command, idx int) error {
	// Contract §3.4: payload idx is 1-based.
	pausePayload := fmt.Sprintf(`{"step":%q,"idx":%d}`, cmd.Raw, idx+1)

	emitPauseMarker := func() {
		fmt.Fprintf(os.Stdout, "\x00MANUL_DEBUG_PAUSE\x00%s\n", pausePayload)
		os.Stdout.Sync()
	}
	emitPauseMarker()

	if err := rt.injectDebugModal(ctx, cmd.Raw); err != nil {
		rt.logger.Warn("debug: modal inject failed: %v", err)
	}
	defer rt.removeDebugModal(ctx)

	sc := bufio.NewScanner(os.Stdin)
	sc.Split(bufio.ScanLines)
	// The extension line-buffer safety cap is 1 MB; match it so long tokens don't trigger ErrTooLong.
	sc.Buffer(make([]byte, 1024), 1024*1024)

	inputCh := make(chan string, 1)
	readNext := func() {
		go func() {
			if sc.Scan() {
				inputCh <- sc.Text()
			} else {
				inputCh <- "next"
			}
		}()
	}
	readNext()

	emitExplain := func(stepText string) {
		payload := rt.buildExplainNextResult(ctx, stepText, cmd)
		// Clear previous explain-next highlight and highlight the new best candidate.
		_ = rt.clearDebugHighlight(ctx)
		if payload.TargetElement != nil {
			_ = rt.debugHighlight(ctx, *payload.TargetElement)
		}
		ep, _ := json.Marshal(payload)
		fmt.Fprintf(os.Stdout, "\x00MANUL_EXPLAIN_NEXT\x00%s\n", ep)
		os.Stdout.Sync()
	}

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			raw, err := rt.page.EvalJS(ctx, `window.__manul_debug_action||""`)
			if err != nil {
				continue
			}
			var action string
			if json.Unmarshal(raw, &action) == nil && action == "ABORT" {
				rt.logger.Warn("debug: abort from browser")
				return ErrDebugStop
			}
		case token := <-inputCh:
			raw := strings.TrimRight(token, "\r\n")
			trimmed := strings.TrimSpace(raw)
			lower := strings.ToLower(trimmed)

			switch {
			case lower == "" || lower == "next":
				// Pause at the next step. Preserve existing breakLines; append
				// an index-based one-shot breakpoint at idx+1 (0-based internal).
				if rt.breakSteps == nil {
					rt.breakSteps = make(map[int]bool)
				}
				rt.breakSteps[idx+1] = true
				rt.clearDebugHighlight(ctx)
				return nil

			case lower == "continue":
				rt.debugContinue = true
				rt.breakSteps = make(map[int]bool)
				rt.clearDebugHighlight(ctx)
				return nil

			case lower == "debug-stop":
				// Contract §4.3: clear breakpoints, continue the run.
				rt.debugContinue = true
				rt.breakLines = make(map[int]bool)
				rt.breakSteps = make(map[int]bool)
				rt.clearDebugHighlight(ctx)
				return nil

			case lower == "abort":
				return ErrDebugStop

			case lower == "highlight":
				js := `(function(){var el=document.querySelector('[data-manul-debug-highlight="true"]');if(el)el.scrollIntoView({behavior:'smooth',block:'center'});})();`
				rt.page.EvalJS(ctx, js)
				emitPauseMarker()
				readNext()

			case strings.HasPrefix(lower, "highlight "):
				xpath := strings.TrimPrefix(trimmed, "highlight ")
				xpath = strings.TrimPrefix(xpath, "HIGHLIGHT ")
				if err := rt.debugHighlight(ctx, xpath); err != nil {
					rt.logger.Warn("debug: highlight failed: %v", err)
				}
				emitPauseMarker()
				readNext()

			case lower == "explain-next" || lower == "explain":
				emitExplain(cmd.Raw)
				emitPauseMarker()
				readNext()

			case strings.HasPrefix(lower, "explain-next "):
				// Contract §4.3: `explain-next {"step":"<override>"}\n`.
				jsonPart := strings.TrimSpace(trimmed[len("explain-next"):])
				stepText := cmd.Raw
				var ov struct {
					Step string `json:"step"`
				}
				if err := json.Unmarshal([]byte(jsonPart), &ov); err == nil && ov.Step != "" {
					stepText = ov.Step
				}
				emitExplain(stepText)
				emitPauseMarker()
				readNext()

			default:
				emitPauseMarker()
				readNext()
			}
		}
	}
}