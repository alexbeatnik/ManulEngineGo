// Package runtime implements the ManulHeart DSL execution engine.
//
// STATUS: Stub package. The full implementation will live here once the
// CDP backend (pkg/cdp) is production-ready.
package runtime

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexbeatnik/ManulHeart/pkg/browser"
	"github.com/alexbeatnik/ManulHeart/pkg/config"
	"github.com/alexbeatnik/ManulHeart/pkg/core"
	"github.com/alexbeatnik/ManulHeart/pkg/dom"
	"github.com/alexbeatnik/ManulHeart/pkg/dsl"
	"github.com/alexbeatnik/ManulHeart/pkg/explain"
	"github.com/alexbeatnik/ManulHeart/pkg/heuristics"
	"github.com/alexbeatnik/ManulHeart/pkg/pages"
	"github.com/alexbeatnik/ManulHeart/pkg/scorer"
	"github.com/alexbeatnik/ManulHeart/pkg/utils"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const (
	ThresholdHighConfidence = 0.15 // strong heuristic match
	ThresholdAmbiguous      = 0.03 // minimum for heuristic choice
	ThresholdRunnerUpGap    = 0.02
	ThresholdPass3Total     = 0.12
	ThresholdPass3Proximity = 0.18
	ThresholdPass3Gap       = 0.04
)

// Runtime executes ManulHeart DSL hunts against a live browser page.
//
// CONCURRENCY CONTRACT: A Runtime instance is NOT safe for concurrent use.
// Each goroutine executing hunts must own its own Runtime, Page, and
// (typically) ChromeProcess. The DOM snapshot cache, variable store, and
// sticky checkbox state are unguarded by design — sharing a Runtime across
// goroutines will cause data races detectable by `go test -race`.
//
// To run multiple hunts in parallel, construct one Runtime per worker via
// pkg/worker.NewWorker rather than sharing a single Runtime.
type Runtime struct {
	cfg    config.Config
	page   browser.Page
	logger *utils.Logger
	vars   *ScopedVariables
	pages  *pages.Registry

	// sourcePath is the .hunt file path, used for resolving relative mock/data files.
	sourcePath string

	cachedElements       []dom.ElementSnapshot
	stickyCheckboxStates map[string]bool

	// mockRules stores MOCK command rules keyed by "METHOD URL_PATTERN".
	mockRules map[string]mockRule

	// debug state — only populated when cfg.DebugMode is true
	breakLines       map[int]bool             // source line numbers that are breakpoints; empty = pause every step
	breakSteps       map[int]bool             // command indices queued by extension-mode "next" to pause at
	debugContinue    bool                     // when true, skip all future pauses
	lastExplainData  []scorer.RankedCandidate // cached ranking for the "explain" debug command
	pendingDebugPause bool                    // set in runCommands when an action step should pause after resolution
	pendingDebugIdx  int                      // command index for the pending debug pause

	// onStep, when non-nil, is invoked synchronously after each step
	// completes (pass or fail) with that step's ExecutionResult. Set via
	// SetOnStep. Intended for streaming consumers (e.g. the CLI's -jsonl
	// mode) that need per-step progress before the full HuntResult.
	onStep func(explain.ExecutionResult)

	// activeHuntRes is the HuntResult of the in-flight RunHunt call. Loop
	// (REPEAT/WHILE/FOR EACH) and IF bodies route through runCommands with
	// this so their steps are recorded in the report and surfaced via
	// onStep — not silently dropped. Nil outside RunHunt.
	activeHuntRes *explain.HuntResult
}

type mockRule struct {
	Method      string
	Pattern     string
	Body        string
	ContentType string
}

// New creates a new Runtime bound to the given Config, Page, and Logger.
//
// The returned Runtime is single-goroutine; see the type doc for the
// concurrency contract. For parallel execution, use pkg/worker.
func New(cfg config.Config, page browser.Page, logger *utils.Logger) *Runtime {
		rt := &Runtime{
			cfg:                  cfg,
			page:                 page,
			logger:               logger,
			vars:                 NewScopedVariables(),
			pages:                pages.NewRegistry(""),
			stickyCheckboxStates: make(map[string]bool),
			mockRules:            make(map[string]mockRule),
			breakLines:           make(map[int]bool),
		}
	for _, ln := range cfg.BreakLines {
		rt.breakLines[ln] = true
	}
	return rt
}

// ResolveVariable returns the value of a runtime variable, respecting scope
// precedence (Row > Step > Mission > Global > Import).
func (rt *Runtime) ResolveVariable(name string) (string, bool) {
	return rt.vars.Resolve(name)
}

// SetOnStep registers a callback that fires after every step (pass or
// fail) with that step's ExecutionResult. The callback runs on the
// runtime's own goroutine and MUST NOT block — long-running consumers
// should hand the value off to a buffered channel. Pass nil to clear.
//
// Single-goroutine contract: the callback inherits the Runtime's
// concurrency guarantees, so no locking is required when it reads the
// passed ExecutionResult.
func (rt *Runtime) SetOnStep(fn func(explain.ExecutionResult)) {
	rt.onStep = fn
}

// RunHunt executes all commands in hunt against the bound page.
// It returns an explain.HuntResult summarising the execution.
// Commands are grouped by their StepBlock label; each group emits
// BlockStart / BlockPass / BlockFail so the output mirrors the
// per-STEP structure of the .hunt file.
// Optional rowVars (data-driven testing) are applied at LevelRow scope
// so they override mission-level @var: declarations.
func (rt *Runtime) RunHunt(ctx context.Context, hunt *dsl.Hunt, rowVars ...map[string]string) (*explain.HuntResult, error) {
	if hunt == nil {
		return nil, fmt.Errorf("runtime: nil hunt")
	}

	for k, v := range hunt.Vars {
		rt.vars.Set(k, v, LevelMission)
	}

	if len(rowVars) > 0 && rowVars[0] != nil {
		for k, v := range rowVars[0] {
			rt.vars.Set(k, v, LevelRow)
		}
	}

	rt.sourcePath = hunt.SourcePath

	result := &explain.HuntResult{
		HuntFile: hunt.SourcePath,
		Title:    hunt.Title,
		Context:  hunt.Context,
	}
	// Expose the result so loop/IF bodies (executed inside executeCommand)
	// can record their steps through runCommands. Cleared on return.
	rt.activeHuntRes = result
	defer func() { rt.activeHuntRes = nil }()

	start := time.Now()
	passed, failed := 0, 0
	var firstErr error

	// Execute SETUP block before the mission body.
	if len(hunt.SetupCommands) > 0 {
		rt.logger.BlockStart("SETUP")
		p, f, err := rt.runCommands(ctx, hunt.SetupCommands, result, 0)
		passed += p
		failed += f
		if err != nil {
			rt.logger.BlockFail("SETUP")
			firstErr = err
		} else {
			rt.logger.BlockPass("SETUP")
		}
	}

	// Defer TEARDOWN so it always executes, even if setup or mission fails.
	defer func() {
		if len(hunt.TeardownCommands) > 0 {
			rt.logger.BlockStart("TEARDOWN")
			_, _, tdErr := rt.runCommands(ctx, hunt.TeardownCommands, result, 0)
			if tdErr != nil {
				rt.logger.BlockFail("TEARDOWN")
			} else {
				rt.logger.BlockPass("TEARDOWN")
			}
		}
		result.TotalDuration = time.Since(start)
		result.TotalDurationMS = result.TotalDuration.Milliseconds()
		// Derive the summary from the recorded results so that steps nested
		// inside loop/IF bodies (which runCommands appends but whose counters
		// are not threaded back to the outer pass/fail tally) are included.
		resPassed, resFailed := 0, 0
		for _, r := range result.Results {
			if r.Success {
				resPassed++
			} else {
				resFailed++
			}
		}
		result.TotalSteps = resPassed + resFailed
		result.Passed = resPassed
		result.Failed = resFailed
		result.Success = resFailed == 0
	}()

	if firstErr != nil {
		return result, firstErr
	}

	// Group consecutive commands by StepBlock, preserving order.
	type stepGroup struct {
		name     string
		commands []dsl.Command
	}
	var groups []stepGroup
	defaultLabel := hunt.Title
	if defaultLabel == "" {
		defaultLabel = "Mission"
	}
	for _, cmd := range hunt.Commands {
		label := cmd.StepBlock
		if label == "" {
			label = defaultLabel
		}
		if len(groups) == 0 || groups[len(groups)-1].name != label {
			groups = append(groups, stepGroup{name: label})
		}
		groups[len(groups)-1].commands = append(groups[len(groups)-1].commands, cmd)
	}

	var cmdOffset int
	for _, g := range groups {
		rt.logger.BlockStart(g.name)
		p, f, err := rt.runCommands(ctx, g.commands, result, cmdOffset)
		cmdOffset += len(g.commands)
		passed += p
		failed += f
		if err != nil || f > 0 {
			rt.logger.BlockFail(g.name)
			if firstErr == nil {
				firstErr = err
			}
			// Stop at first failed block (mirrors Python behaviour).
			break
		}
		rt.logger.BlockPass(g.name)
	}

	return result, firstErr
}

func isActionCommand(t dsl.CommandType) bool {
	switch t {
	case dsl.CmdClick, dsl.CmdFill, dsl.CmdType, dsl.CmdHover, dsl.CmdCheck,
		dsl.CmdUncheck, dsl.CmdSelect, dsl.CmdDoubleClick, dsl.CmdRightClick, dsl.CmdUploadFile:
		return true
	}
	return false
}

func (rt *Runtime) runCommands(ctx context.Context, commands []dsl.Command, huntRes *explain.HuntResult, offset int) (int, int, error) {
	passed, failed := 0, 0
	for i, cmd := range commands {
		if err := ctx.Err(); err != nil {
			return passed, failed, fmt.Errorf("runtime: context cancelled: %w", err)
		}

		globalIdx := offset + i
		rt.pendingDebugPause = false
		if rt.cfg.DebugMode && rt.shouldPause(cmd, globalIdx) {
			if isActionCommand(cmd.Type) {
				// Defer debug pause until after element resolution inside executeCommand.
				rt.pendingDebugPause = true
				rt.pendingDebugIdx = globalIdx
			} else {
				if dbgErr := rt.debugPrompt(ctx, cmd, globalIdx); dbgErr != nil {
					return passed, failed, dbgErr
				}
			}
		}

		rt.logger.ActionStart(cmd.Raw)
		stepStart := time.Now()
		stepResult, err := rt.executeCommand(ctx, cmd)
		durMs := time.Since(stepStart).Milliseconds()

		if huntRes != nil {
			huntRes.Results = append(huntRes.Results, stepResult)
		}
		if rt.onStep != nil {
			rt.onStep(stepResult)
		}
		if err != nil {
			failed++
			rt.logger.ActionFail(err)
			rt.logger.Error("step failed (%s): %v", cmd.Raw, err)
			return passed, failed, err
		}
		rt.logger.ActionPass(float64(durMs) / 1000)
		passed++
	}
	return passed, failed, nil
}

// StepResult is the result of a single DSL step run via RunStep.
type StepResult struct {
	// Command is the raw DSL text that was executed.
	Command string `json:"command"`
	// Success is true when the command succeeded.
	Success bool `json:"success"`
	// Error is the error message if Success is false.
	Error string `json:"error,omitempty"`
}

// RunStep executes a single raw DSL command string and returns its result.
func (rt *Runtime) RunStep(ctx context.Context, rawStep string) (*StepResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Parse the single-line command.
	hunt, err := dsl.Parse(strings.NewReader(rawStep))
	if err != nil {
		return &StepResult{Command: rawStep, Error: err.Error()}, err
	}
	if len(hunt.Commands) == 0 {
		return &StepResult{Command: rawStep, Success: true}, nil
	}
	stepResult, execErr := rt.executeCommand(ctx, hunt.Commands[0])
	return &StepResult{
		Command: rawStep,
		Success: execErr == nil,
		Error:   stepResult.Error,
	}, execErr
}

// RunCommand executes a single already-parsed DSL command and returns the
// full structured ExecutionResult — including the targeting decision chain
// (ranked candidates, winner score) and any extracted ActionValue.
//
// Unlike RunStep, which collapses the result into a thin pass/fail record,
// RunCommand preserves everything callers need to build their own compact
// views. It is the primitive that pkg/agent is built on. The command runs
// on the runtime's own goroutine and obeys the single-goroutine contract.
func (rt *Runtime) RunCommand(ctx context.Context, cmd dsl.Command) (explain.ExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return explain.ExecutionResult{}, err
	}
	return rt.executeCommand(ctx, cmd)
}

func (rt *Runtime) resolveAnchor(ctx context.Context, label string, elements []dom.ElementSnapshot) (*scorer.AnchorContext, error) {
	if label == "" {
		return nil, nil
	}
	winner, err := rt.resolveStructuralAnchor(label, elements)
	if err != nil {
		return nil, err
	}
	return &scorer.AnchorContext{
		Rect:       winner.Rect,
		XPath:      winner.XPath,
		FrameIndex: winner.FrameIndex,
		Words:      scorer.SignificantWords(winner.VisibleText),
	}, nil
}

func (rt *Runtime) resolveStructuralAnchor(label string, elements []dom.ElementSnapshot) (dom.ElementSnapshot, error) {
	if label == "" {
		return dom.ElementSnapshot{}, fmt.Errorf("anchor label is empty")
	}
	label = rt.resolveVariables(label)
	// Anchor resolution uses "none" mode to allow matching any structural element (div, span, etc.)
	ranked := scorer.Rank(label, "", string(dsl.ModeNone), elements, 1, nil)
	if len(ranked) == 0 || ranked[0].Explain.Score.Total < ThresholdAmbiguous {
		return dom.ElementSnapshot{}, fmt.Errorf("near anchor not found: %q", label)
	}
	return ranked[0].Element, nil
}

// executeCommand runs a single DSL command and returns its execution result.
func (rt *Runtime) executeCommand(ctx context.Context, cmd dsl.Command) (res explain.ExecutionResult, err error) {
	start := time.Now()
	res = explain.ExecutionResult{
		Step:            cmd.Raw,
		CommandType:     string(cmd.Type),
		ActionPerformed: strings.ToLower(string(cmd.Type)),
	}
	defer func() {
		if pageURL, urlErr := rt.page.CurrentURL(ctx); urlErr == nil {
			res.PageURL = pageURL
		}
		res.Duration = time.Since(start)
		res.DurationMS = res.Duration.Milliseconds()
	}()

	switch cmd.Type {
	case dsl.CmdNavigate:
		url := rt.resolveVariables(cmd.URL)
		res.ActionValue = url
		err = rt.page.Navigate(ctx, url)
		if err == nil {
			// Navigation started, wait for it to complete.
			// Brief pause helps CDP catch up before we check readyState.
			if waitErr := rt.page.Wait(ctx, 300*time.Millisecond); waitErr != nil {
				err = waitErr
				break
			}
			if waitErr := rt.page.WaitForLoad(ctx); waitErr != nil {
				err = waitErr
				break
			}
			rt.invalidateSnapshot()
			if rt.cfg.AutoAnnotate {
				rt.autoAnnotateNavigate(ctx, url)
			}
			// Re-apply mocks after navigation so they are available on the new page.
			if len(rt.mockRules) > 0 {
				_ = rt.applyMockJS(ctx)
			}
		}

	case dsl.CmdWait:
		err = rt.page.Wait(ctx, time.Duration(cmd.WaitSeconds*float64(time.Second)))

	case dsl.CmdPress:
		// PRESS <key> dispatches a keyboard event to whatever element
		// currently has focus. Typical use: FILL '<field>' '<text>'
		// followed by PRESS Enter to submit a form. The "PRESS Key ON
		// '<target>'" form is parsed but not yet supported — callers
		// should CLICK '<target>' first, then PRESS Key.
		key := strings.TrimSpace(cmd.PressKey)
		if key == "" {
			err = fmt.Errorf("PRESS: missing key name")
			break
		}
		res.ActionValue = key
		if strings.TrimSpace(cmd.PressTarget) != "" {
			rt.logger.Warn("PRESS ON '<target>' is not yet implemented — dispatching to focused element instead")
		}
		if dispatchErr := rt.page.DispatchKey(ctx, key, 0); dispatchErr != nil {
			err = fmt.Errorf("PRESS %s: %w", key, dispatchErr)
			break
		}
		rt.invalidateSnapshot()

	case dsl.CmdPrint:
		text := rt.resolveVariables(cmd.PrintText)
		res.ActionValue = text
		rt.logger.ActionDetail("📢", "PRINT: %s", text)

	case dsl.CmdWaitForResponse:
		pattern := rt.resolveVariables(cmd.WaitResponseURL)
		res.ActionValue = pattern
		err = rt.page.WaitForResponse(ctx, pattern, rt.cfg.DefaultTimeout)

	case dsl.CmdCallGo:
		res.ActionValue, res.ProbeMetadata, err = rt.executeCallGo(ctx, cmd)

	case dsl.CmdMock:
		err = rt.handleMock(ctx, cmd)
		if err == nil {
			res.ActionValue = fmt.Sprintf("%s %s → %s", cmd.MockMethod, cmd.MockPattern, cmd.MockFile)
		}

	case dsl.CmdPause:
		// Force an interactive debug prompt even if --debug is not set globally.
		// Contract §4.1: PAUSE only produces pauses in terminal (--debug) mode;
		// in pipe mode the extension does not respond, which would deadlock.
		if !isTTY() {
			rt.logger.Warn("PAUSE ignored in non-TTY mode")
			break
		}
		rt.logger.Info("PAUSE command encountered")
		err = rt.debugPrompt(ctx, cmd, -1)
 
	case dsl.CmdDebugVars:
		vars := rt.vars.String()
		rt.logger.Info(vars)
		res.ActionValue = vars

	case dsl.CmdSet:
		if cmd.SetVar != "" {
			val := rt.resolveVariables(cmd.SetValue)
			res.ActionValue = val
			rt.vars.Set(cmd.SetVar, val, LevelRow)
			break
		}
		fallthrough

	case dsl.CmdClick, dsl.CmdFill, dsl.CmdType, dsl.CmdHover, dsl.CmdCheck, dsl.CmdUncheck, dsl.CmdSelect, dsl.CmdDoubleClick, dsl.CmdRightClick, dsl.CmdUploadFile:
		handled, actionValue, metadata, customErr := rt.tryExecuteCustomControl(ctx, cmd)
		if handled {
			res.ActionValue = actionValue
			res.ProbeMetadata = metadata
			if customErr != nil {
				err = customErr
			} else {
				rt.invalidateSnapshot()
			}
			break
		}

		// Target resolution needed for interaction
		res.TargetRequired = true
		elements, errSnapshot := rt.loadSnapshot(ctx)
		if errSnapshot != nil {
			err = errSnapshot
			break
		}
		res.CandidatesConsidered = len(elements)

		// Figure out interaction mode
		mode := dsl.ModeNone
		if cmd.Type == dsl.CmdFill || cmd.Type == dsl.CmdSet || cmd.Type == dsl.CmdType {
			mode = dsl.ModeInput
		} else if cmd.Type == dsl.CmdClick {
			mode = dsl.ModeClickable
		} else if cmd.Type == dsl.CmdCheck || cmd.Type == dsl.CmdUncheck {
			mode = dsl.ModeCheckbox
		} else if cmd.Type == dsl.CmdSelect {
			mode = dsl.ModeSelect
		} else if cmd.Type == dsl.CmdUploadFile {
			mode = dsl.ModeClickable
		}

		targetPath := rt.resolveVariables(cmd.Target)
		if cmd.Type == dsl.CmdSet {
			targetPath = rt.resolveVariables(cmd.SetVar)
		}
		res.TargetQuery = targetPath
		res.TypeHint = cmd.TypeHint

		if (cmd.Type == dsl.CmdFill || cmd.Type == dsl.CmdType || cmd.Type == dsl.CmdSet) && isShadowLikeQuery(targetPath) {
			val := rt.resolveVariables(cmd.Value)
			if cmd.Type == dsl.CmdSet && cmd.SetValue != "" {
				val = rt.resolveVariables(cmd.SetValue)
			}
			handled, typeErr := rt.trySetShadowInputValue(ctx, targetPath, val)
			if typeErr != nil {
				err = typeErr
				break
			}
			if handled {
				res.ActionValue = val
				res.ProbeMetadata = map[string]any{"resolution_strategy": "shadow-input-direct"}
				rt.invalidateSnapshot()
				if waitErr := rt.page.Wait(ctx, 200*time.Millisecond); waitErr != nil {
					err = waitErr
				}
				break
			}
		}

		if cmd.Type == dsl.CmdClick && isDropdownLikeQuery(targetPath) {
			handled, clickErr := rt.tryClickDropdownTriggerByLabel(ctx, targetPath)
			if clickErr != nil {
				err = clickErr
				break
			}
			if handled {
				res.ActionValue = targetPath
				res.ProbeMetadata = map[string]any{"resolution_strategy": "dropdown-trigger-direct"}
				rt.invalidateSnapshot()
				if waitErr := rt.page.Wait(ctx, 300*time.Millisecond); waitErr != nil {
					err = waitErr
				}
				break
			}
		}

		if cmd.Type == dsl.CmdClick && isLikelyDropdownOptionQuery(targetPath) {
			handled, clickErr := rt.tryClickVisibleDropdownOption(ctx, targetPath)
			if clickErr != nil {
				err = clickErr
				break
			}
			if handled {
				res.ActionValue = targetPath
				res.ProbeMetadata = map[string]any{"resolution_strategy": "dropdown-option-direct"}
				rt.invalidateSnapshot()
				if waitErr := rt.page.Wait(ctx, 200*time.Millisecond); waitErr != nil {
					err = waitErr
				}
				break
			}
		}

		anchor, errAnchor := rt.resolveAnchor(ctx, cmd.NearAnchor, elements)
		if errAnchor != nil {
			err = errAnchor
			break
		}

		candidateElements, contextualStrategy, contextErr := rt.applyContextualFilters(cmd, elements)
		if contextErr != nil {
			err = contextErr
			break
		}

		// Restrictive modes (input, checkbox, select) need special handling
		// to support "Click/Check X" when X is a label or nearby table cell.
		isRestrictive := mode == dsl.ModeInput || mode == dsl.ModeCheckbox || mode == dsl.ModeSelect

		var ranked []scorer.RankedCandidate
		resolutionStrategy := "standard"
		if isRestrictive && targetPath != "" {
			ranked, resolutionStrategy = resolveRestrictiveCandidates(targetPath, cmd.TypeHint, mode, candidateElements, anchor, rt.logger)
		}

		// Standard resolution if not already handled by restrictive fallback
		if len(ranked) == 0 {
			ranked = scorer.Rank(targetPath, cmd.TypeHint, string(mode), candidateElements, 5, anchor)
			resolutionStrategy = "standard"
		}
		ranked = collapseNestedDuplicateRankedCandidates(ranked)
		if contextualStrategy != "" {
			resolutionStrategy = contextualStrategy + "+" + resolutionStrategy
		}

		if len(ranked) == 0 {
			err = fmt.Errorf("target not found: %q", targetPath)
			res.FailureReason = explain.ReasonNotFound
			break
		}

		best := ranked[0]
		if selectionIsAmbiguous(ranked) {
			runnerUp := 0.0
			if len(ranked) > 1 {
				runnerUp = ranked[1].Explain.Score.Total
			}
			err = fmt.Errorf("target resolution too ambiguous (confidence %.3f, runner-up %.3f)", best.Explain.Score.Total, runnerUp)
			res.FailureReason = explain.ReasonAmbiguous
			// Attach the top candidates so callers can surface "you almost
			// matched X (0.18)" without a follow-up scan.
			appendRankedCandidates(&res, ranked, 5)
			break
		}
		appendRankedCandidates(&res, ranked, 5)
		res.WinnerXPath = best.Element.XPath
		res.WinnerScore = best.Explain.Score.Total
		res.ProbeMetadata = map[string]any{
			"resolution_strategy": resolutionStrategy,
			"interaction_mode":    string(mode),
		}
		if cmd.NearAnchor != "" {
			res.ProbeMetadata["near_anchor"] = cmd.NearAnchor
		}
		if cmd.OnRegion != "" {
			res.ProbeMetadata["on_region"] = cmd.OnRegion
		}
		if cmd.InsideContainer != "" {
			res.ProbeMetadata["inside_container"] = cmd.InsideContainer
		}
		if cmd.InsideRowText != "" {
			res.ProbeMetadata["inside_row_text"] = cmd.InsideRowText
		}

		// Anti-phantom guard for inputs/selects (soft warning; logging is inside the helper).
		rt.passesAntiPhantomGuard(string(mode), targetPath, best.Element)

		winner := best.Element
		conf := best.Explain.Score.Total
		label := "Context reuse"
		if conf >= ThresholdHighConfidence {
			label = "High confidence match"
		} else if conf >= ThresholdAmbiguous {
			label = "Keyword match"
		}
		rt.logger.HeuristicDetail(conf, fmt.Sprintf("%s — '%s' → %s (ID=%d)", label, targetPath, winner.Name, winner.ID))

		if rt.cfg.ExplainMode {
			rt.logger.Info("  Breakdown: Text=%.2f, Attr=%.2f, Sem=%.2f, Prox=%.2f",
				best.Explain.Score.NormalizedTextMatch,
				best.Explain.Score.LabelMatch+best.Explain.Score.AriaMatch,
				best.Explain.Score.TagSemantics,
				best.Explain.Score.ProximityScore)
		}

		// Visual feedback: flash highlight for every action (matches Python _highlight).
		_ = rt.page.HighlightElement(ctx, winner.ID, winner.XPath, 2000)
		// Ensure the highlight is removed as soon as the interaction finishes,
		// even if the action triggers navigation that destroys the element.
		defer func() {
			_ = rt.page.ClearHighlight(ctx)
		}()

		// Debug mode: persistent magenta highlight + pause before action execution.
		if rt.pendingDebugPause {
			rt.pendingDebugPause = false
			_ = rt.debugHighlight(ctx, winner.XPath)
			if dbgErr := rt.debugPrompt(ctx, cmd, rt.pendingDebugIdx); dbgErr != nil {
				err = dbgErr
				break
			}
			_ = rt.clearDebugHighlight(ctx)
		}

		// Perform action
		switch cmd.Type {
		case dsl.CmdFill, dsl.CmdSet, dsl.CmdType:
			val := rt.resolveVariables(cmd.Value)
			if cmd.Type == dsl.CmdSet && cmd.SetValue != "" {
				// This case should be handled by the specialized CmdSet above,
				// but we keep it for robustness if fallthrough happened.
				val = rt.resolveVariables(cmd.SetValue)
			}
			res.ActionValue = val
			err = rt.page.SetInputValue(ctx, winner.ID, winner.XPath, val)
			if err == nil {
				rt.logger.ActionDetail("⌨️", "Typed %q → %q", val, winner.Name)
				if waitErr := rt.page.Wait(ctx, 300*time.Millisecond); waitErr != nil {
					err = waitErr
					break
				}
				rt.invalidateSnapshot()
			}
		case dsl.CmdClick:
			if isDropdownLikeQuery(targetPath) && !isDropdownLikeElement(winner) {
				handled, clickErr := rt.tryClickNearbyDropdownControl(ctx, winner)
				if clickErr != nil {
					err = clickErr
					break
				}
				if handled {
					rt.invalidateSnapshot()
					if waitErr := rt.page.Wait(ctx, 300*time.Millisecond); waitErr != nil {
						err = waitErr
					} else if stickyErr := rt.reconcileStickyCheckboxStates(ctx); stickyErr != nil {
						err = stickyErr
					}
					break
				}
			}
			x, y, e := rt.page.GetElementCenter(ctx, winner.ID, winner.XPath)
			if e != nil {
				err = fmt.Errorf("center calc: %w", e)
			} else {
				// Perform interaction
				_ = rt.page.ScrollIntoView(ctx, winner.ID, winner.XPath)
				err = rt.page.Click(ctx, x, y)
				if err == nil {
					// A click may trigger navigation or AJAX update.
					if waitErr := rt.page.Wait(ctx, 500*time.Millisecond); waitErr != nil {
						err = waitErr
						break
					}
					rt.invalidateSnapshot()
					_ = rt.page.WaitForLoad(ctx)
					if stickyErr := rt.reconcileStickyCheckboxStates(ctx); stickyErr != nil {
						err = stickyErr
					}
				}
			}
		case dsl.CmdDoubleClick:
			x, y, e := rt.page.GetElementCenter(ctx, winner.ID, winner.XPath)
			if e != nil {
				err = fmt.Errorf("center calc: %w", e)
			} else {
				err = rt.page.DoubleClick(ctx, x, y)
			}
		case dsl.CmdRightClick:
			x, y, e := rt.page.GetElementCenter(ctx, winner.ID, winner.XPath)
			if e != nil {
				err = fmt.Errorf("center calc: %w", e)
			} else {
				err = rt.page.RightClick(ctx, x, y)
			}
		case dsl.CmdHover:
			x, y, e := rt.page.GetElementCenter(ctx, winner.ID, winner.XPath)
			if e != nil {
				err = fmt.Errorf("center calc: %w", e)
			} else {
				_ = rt.page.ScrollIntoView(ctx, winner.ID, winner.XPath)
				err = rt.page.Hover(ctx, x, y)
				if err == nil {
					if waitErr := rt.page.Wait(ctx, 300*time.Millisecond); waitErr != nil {
						err = waitErr
						break
					}
					rt.invalidateSnapshot()
				}
			}
		case dsl.CmdCheck, dsl.CmdUncheck:
			checked := cmd.Type == dsl.CmdCheck
			err = rt.page.SetChecked(ctx, winner.ID, winner.XPath, checked)
			if err == nil {
				rt.rememberStickyCheckboxState(targetPath, checked)
				rt.invalidateSnapshot()
				if verifyErr := rt.ensureCheckboxTargetState(ctx, targetPath, checked, ranked); verifyErr != nil {
					err = verifyErr
				}
			}
		case dsl.CmdSelect:
			val := rt.resolveVariables(cmd.Value)
			res.ActionValue = val
			_ = rt.page.ScrollIntoView(ctx, winner.ID, winner.XPath)

			// Detect if it's a native select or custom dropdown
			if winner.Tag == "select" {
				js := fmt.Sprintf(`(() => {
					const el = document.evaluate("%s", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
					if (!el) return;
					for (let opt of el.options) {
						if (opt.text.trim() === "%s" || opt.value === "%s") {
							el.value = opt.value;
							el.dispatchEvent(new Event('change', {bubbles: true}));
							return;
						}
					}
				})()`, strings.ReplaceAll(winner.XPath, `"`, `\"`),
					strings.ReplaceAll(val, `"`, `\"`),
					strings.ReplaceAll(val, `"`, `\"`))
				_, err = rt.page.EvalJS(ctx, js)
				if err == nil {
					rt.invalidateSnapshot()
				}
			} else {
				// Custom dropdown: Click then search for the option text
				_ = rt.page.Click(ctx, winner.Rect.Left+winner.Rect.Width/2, winner.Rect.Top+winner.Rect.Height/2)
				rt.invalidateSnapshot()
				if waitErr := rt.page.Wait(ctx, 300*time.Millisecond); waitErr != nil {
					err = waitErr
					break
				}

				// Re-probe to find the option that appeared
				raw, _ := rt.page.CallProbe(ctx, heuristics.BuildSnapshotProbe(), nil)
				elements, _ := heuristics.ParseProbeResult(raw)

				// Exclude the container itself and hidden elements from option search
				var candidates []dom.ElementSnapshot
				for _, e := range elements {
					if e.ID != winner.ID && e.IsVisible && e.Tag != "input" {
						candidates = append(candidates, e)
					}
				}

				rankedOpt := scorer.Rank(val, "", "clickable", candidates, 1, nil)
				if len(rankedOpt) > 0 {
					opt := rankedOpt[0].Element
					rt.logger.Info("Selected option %q (Tag=%s ID=%d)", val, opt.Tag, opt.ID)
					_ = rt.page.ScrollIntoView(ctx, opt.ID, opt.XPath)
					cx, cy, _ := rt.page.GetElementCenter(ctx, opt.ID, opt.XPath)
					err = rt.page.Click(ctx, cx, cy)
					if err == nil {
						rt.invalidateSnapshot()
					}
				} else {
					err = fmt.Errorf("could not find option %q after clicking %q", val, winner.Tag)
				}
			}
		case dsl.CmdUploadFile:
			filePath := rt.resolveVariables(cmd.UploadFilePath)
			if filePath == "" {
				filePath = rt.resolveVariables(cmd.UploadFile)
			}
			res.ActionValue = filePath
			err = rt.page.SetFileInput(ctx, winner.ID, winner.XPath, []string{filePath})
			if err == nil {
				rt.invalidateSnapshot()
			}
		}

	case dsl.CmdDrag:
		elements, errSnapshot := rt.loadSnapshot(ctx)
		if errSnapshot != nil {
			err = errSnapshot
			break
		}

		sourcePath := rt.resolveVariables(cmd.DragSource)
		rankedSrc := scorer.Rank(sourcePath, cmd.TypeHint, string(dsl.ModeClickable), elements, 5, nil)
		if len(rankedSrc) == 0 {
			err = fmt.Errorf("drag source not found: %q", sourcePath)
			break
		}
		for _, r := range rankedSrc {
			res.RankedCandidates = append(res.RankedCandidates, r.Explain)
		}
		srcEl := rankedSrc[0].Element

		x1, y1, errSrc := rt.page.GetElementCenter(ctx, srcEl.ID, srcEl.XPath)
		if errSrc != nil {
			err = fmt.Errorf("source center calc failed: %w", errSrc)
			break
		}

		dropPath := rt.resolveVariables(cmd.DragTarget)
		rankedDest := scorer.Rank(dropPath, "", string(dsl.ModeClickable), elements, 5, nil)
		if len(rankedDest) == 0 {
			err = fmt.Errorf("drag destination not found: %q", dropPath)
			break
		}
		for _, r := range rankedDest {
			res.RankedCandidates = append(res.RankedCandidates, r.Explain)
		}
		destEl := rankedDest[0].Element

		x2, y2, errDest := rt.page.GetElementCenter(ctx, destEl.ID, destEl.XPath)
		if errDest != nil {
			err = fmt.Errorf("destination center calc failed: %w", errDest)
			break
		}

		rt.logger.Info("Target '%s' resolved to element: ID=%d Tag=%s XPath=%s", sourcePath, srcEl.ID, srcEl.Tag, srcEl.XPath)
		rt.logger.Info("Target '%s' resolved to element: ID=%d Tag=%s XPath=%s", dropPath, destEl.ID, destEl.Tag, destEl.XPath)

		err = rt.page.DragAndDrop(ctx, x1, y1, x2, y2)
		if err == nil {
			rt.invalidateSnapshot()
		}

	case dsl.CmdExtract:
		// Use dedicated extraction probe which handles tables/text nodes
		target := rt.resolveVariables(cmd.Target)
		res.TargetRequired = true
		res.TargetQuery = target
		hint := "" // we could extract hint from cmd if needed
		params := []string{target, hint}

		val, errProbe := rt.page.CallProbe(ctx, heuristics.BuildExtractProbe(), params)
		if errProbe != nil {
			err = errProbe
			break
		}

		extracted := strings.Trim(string(val), "\"") // Unquote JSON string if needed

		if extracted == "" || extracted == "null" {
			err = fmt.Errorf("extract target not found or empty: %q", target)
			break
		}
		rt.vars.Set(cmd.ExtractVar, extracted, LevelRow)
		res.ActionValue = extracted
		rt.logger.Info("Extracted '%s' into {%s}", extracted, cmd.ExtractVar)

	case dsl.CmdScroll:
		containerID := ""
		if cmd.ScrollContainer != "" {
			res.TargetRequired = true
			res.TargetQuery = cmd.ScrollContainer
			if isGenericListContainer(cmd.ScrollContainer) {
				containerID = string(core.ScrollStrategyGenericList)
				res.ProbeMetadata = map[string]any{"scroll_strategy": "dropdown-list"}
			} else {
				elements, _ := rt.loadSnapshot(ctx)
				res.CandidatesConsidered = len(elements)
				ranked := scorer.Rank(cmd.ScrollContainer, "", "none", elements, 1, nil)
				if len(ranked) > 0 {
					appendRankedCandidates(&res, ranked, 1)
					res.WinnerXPath = ranked[0].Element.XPath
					res.WinnerScore = ranked[0].Explain.Score.Total
					containerID = ranked[0].Element.XPath
				}
			}
		}

		if scroller, ok := rt.page.(interface {
			ScrollPage(context.Context, string, string) error
		}); ok {
			err = scroller.ScrollPage(ctx, cmd.ScrollDirection, containerID)
		} else {
			// fallback via EvalJS
			amount := 500
			if cmd.ScrollDirection == "up" {
				amount = -500
			}
			js := fmt.Sprintf("window.scrollBy(0, %d)", amount)
			if containerID != "" {
				js = fmt.Sprintf(`
					(document.evaluate("%s", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue).scrollBy(0, %d)
				`, strings.ReplaceAll(containerID, `"`, `\"`), amount)
			}
			_, err = rt.page.EvalJS(ctx, js)
		}
		if err == nil {
			rt.invalidateSnapshot()
		}

	case dsl.CmdVerifySoft:
		// Non-fatal text presence check. Unlike CmdVerify, failure is downgraded
		// to a warning and the hunt continues. Single-shot (no retry loop):
		// a soft assert is informational, and waiting the full DefaultTimeout
		// on something the caller already expects might not be there would
		// unnecessarily slow hunts. Use WAIT FOR ... or hard VERIFY when you
		// need to actually wait for an element.
		target := rt.resolveVariables(cmd.VerifyText)
		res.TargetQuery = target
		raw, errProbe := rt.page.CallProbe(ctx, heuristics.BuildVisibleTextProbe(), nil)
		if errProbe != nil {
			rt.logger.ActionWarn(fmt.Sprintf("VERIFY SOFTLY skipped: probe error: %v", errProbe))
			break
		}
		pageText := strings.ToLower(string(raw))
		present := strings.Contains(pageText, strings.ToLower(target))
		satisfied := present
		if cmd.VerifyNegated {
			satisfied = !present
		}
		if !satisfied {
			expect := "present"
			if cmd.VerifyNegated {
				expect = "NOT present"
			}
			rt.logger.ActionWarn(fmt.Sprintf("VERIFY SOFTLY failed (non-fatal): '%s' expected %s", target, expect))
		}

	case dsl.CmdVerify:
		// Lightweight text presence check via dedicated probe with a small retry loop
		target := rt.resolveVariables(cmd.VerifyText)
		res.TargetQuery = target
		var present bool
		var pageText string
		deadline := time.Now().Add(rt.cfg.DefaultTimeout)
		if dlat, ok := ctx.Deadline(); ok && dlat.Before(deadline) {
			deadline = dlat
		}

		for {
			raw, errProbe := rt.page.CallProbe(ctx, heuristics.BuildVisibleTextProbe(), nil)
			if errProbe == nil {
				pageText = strings.ToLower(string(raw))
				present = strings.Contains(pageText, strings.ToLower(target))
			}
			if present || time.Now().After(deadline) {
				break
			}
			if waitErr := rt.page.Wait(ctx, 200*time.Millisecond); waitErr != nil {
				err = waitErr
				break
			}
		}
		if err != nil {
			break
		}

		if cmd.VerifyNegated {
			if present {
				err = fmt.Errorf("verification failed: '%s' is present, but expected NOT to be", target)
			}
		} else {
			if !present {
				rt.logger.Error("VERIFY FAILED. pageText sample: %s", pageText[:min(500, len(pageText))])
				err = fmt.Errorf("verification failed: '%s' is not present", target)
			}
		}

	case dsl.CmdVerifyField:
		// Full element resolution for state-specific verification.
		res.TargetRequired = true
		target := rt.resolveVariables(cmd.VerifyText)
		res.TargetQuery = target
		verifyDeadline := time.Now().Add(rt.cfg.DefaultTimeout)
		if deadline, ok := ctx.Deadline(); ok && deadline.Before(verifyDeadline) {
			verifyDeadline = deadline
		}
		stateVerified := false
		lastFound := false
		lastStateValue := false
		var lastWinner dom.ElementSnapshot
		var ranked []scorer.RankedCandidate

		for {
			rt.invalidateSnapshot()
			elements, errSnapshot := rt.loadSnapshot(ctx)
			if errSnapshot != nil {
				err = errSnapshot
				break
			}
			res.CandidatesConsidered = len(elements)

			ranked = rankForVerifyState(target, cmd.VerifyState, elements, rt.logger)
			lastFound = verifyRankedCandidateAcceptable(cmd.VerifyState, ranked)
			if lastFound {
				lastWinner = ranked[0].Element
				lastStateValue = elementMatchesVerifyState(lastWinner, cmd.VerifyState)
				verifySatisfied := lastStateValue
				if cmd.VerifyNegated {
					verifySatisfied = !verifySatisfied
				}
				if verifySatisfied {
					stateVerified = true
					break
				}
			} else {
				verifySatisfied := missingElementSatisfiesVerifyState(cmd.VerifyState)
				if cmd.VerifyNegated {
					verifySatisfied = !verifySatisfied
				}
				if verifySatisfied {
					stateVerified = true
					break
				}
			}

			if time.Now().After(verifyDeadline) {
				break
			}
			if waitErr := rt.page.Wait(ctx, 200*time.Millisecond); waitErr != nil {
				err = waitErr
				break
			}
		}

		if len(ranked) > 0 {
			appendRankedCandidates(&res, ranked, 1)
			res.WinnerXPath = ranked[0].Element.XPath
			res.WinnerScore = ranked[0].Explain.Score.Total
		}
		if err != nil {
			break
		}
		if !stateVerified {
			if !lastFound {
				err = fmt.Errorf("verification failed: target field '%s' not found for state %q", target, cmd.VerifyState)
				break
			}
			err = fmt.Errorf("verification failed: target '%s' expected state %s, actual state %s", target, expectedVerifyStateDescription(cmd.VerifyState, cmd.VerifyNegated), actualVerifyStateDescription(cmd.VerifyState, lastStateValue))
		}

	case dsl.CmdIf:
		var bodyToRun []dsl.Command
		for _, b := range cmd.Branches {
			if b.Kind == "else" {
				bodyToRun = b.Body
				break
			}
			matched, cerr := rt.evaluateCondition(ctx, b.Condition)
			if cerr != nil {
				err = cerr
				break
			}
			if matched {
				bodyToRun = b.Body
				break
			}
		}
		if err == nil && len(bodyToRun) > 0 {
			_, _, err = rt.runCommands(ctx, bodyToRun, rt.activeHuntRes, 0)
		}

	case dsl.CmdRepeat:
		count := cmd.RepeatCount
		for i := 0; i < count; i++ {
			if cmd.RepeatVar != "" {
				rt.vars.Set(cmd.RepeatVar, fmt.Sprintf("%d", i), LevelRow)
			}
			_, _, err = rt.runCommands(ctx, cmd.Body, rt.activeHuntRes, 0)
			if err != nil {
				break
			}
		}

	case dsl.CmdWhile:
		limit := 100
		for i := 0; i < limit; i++ {
			matched, cerr := rt.evaluateCondition(ctx, cmd.WhileCondition)
			if cerr != nil {
				err = cerr
				break
			}
			if !matched {
				break
			}
			_, _, err = rt.runCommands(ctx, cmd.Body, rt.activeHuntRes, 0)
			if err != nil {
				break
			}
			if i == limit-1 {
				rt.logger.Warn("WHILE loop reached limit (100)")
			}
		}

	case dsl.CmdForEach:
		v, _ := rt.vars.Resolve(cmd.ForEachCollection)
		coll := v
		items := strings.Split(coll, ",")
		for _, val := range items {
			val = strings.TrimSpace(val)
			if val == "" {
				continue
			}
			rt.vars.Set(cmd.ForEachVar, val, LevelRow)
			_, _, err = rt.runCommands(ctx, cmd.Body, rt.activeHuntRes, 0)
			if err != nil {
				break
			}
		}

	default:
		err = fmt.Errorf("runtime: command %q not yet implemented", cmd.Type)
	}

	if err != nil {
		res.Error = err.Error()
		// Classify the failure for machine-readable consumption. Sites that
		// know the precise cause (target not found / ambiguous) set
		// res.FailureReason directly above; only fill in the rest here.
		if res.FailureReason == explain.ReasonNone {
			res.FailureReason = classifyFailure(ctx, cmd, err)
		}
	} else {
		res.Success = true
	}
	return res, err
}

// classifyFailure derives a machine-readable FailureReason from a failed
// command's error and context, for callers that branch on failure kind
// without parsing error strings. Used only as a fallback — the targeting
// pipeline sets the precise not_found/ambiguous reasons at their source.
func classifyFailure(ctx context.Context, cmd dsl.Command, err error) explain.FailureReason {
	if ctx.Err() != nil {
		return explain.ReasonTimeout
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "too ambiguous"):
		return explain.ReasonAmbiguous
	case strings.Contains(msg, "not found"):
		return explain.ReasonNotFound
	case strings.Contains(msg, "verification failed"):
		return explain.ReasonVerifyFailed
	case strings.Contains(msg, "context deadline") || strings.Contains(msg, "timeout"):
		return explain.ReasonTimeout
	}
	// Commands that resolved a target but failed during the action itself.
	if cmd.Type == dsl.CmdVerify || cmd.Type == dsl.CmdVerifyField {
		return explain.ReasonVerifyFailed
	}
	return explain.ReasonActionFailed
}

func (rt *Runtime) evaluateCondition(ctx context.Context, cond string) (bool, error) {
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return false, nil
	}
	if cond == "true" {
		return true, nil
	}
	if cond == "false" {
		return false, nil
	}

	// 1. Handle element existence: (button|link|field|element|checkbox) 'Target' [not] exists
	if strings.Contains(cond, "exists") {
		neg := strings.Contains(cond, "not exists")
		// Simple parsing for now, actual implementation should use regex
		parts := strings.Fields(cond)
		if len(parts) >= 2 {
			target := ""
			// Extract quoted target
			start := strings.Index(cond, "'")
			end := strings.LastIndex(cond, "'")
			if start != -1 && end != -1 && start < end {
				target = cond[start+1 : end]
			}

			elements, err := rt.loadSnapshot(ctx)
			if err != nil {
				return false, err
			}
			ranked := scorer.Rank(target, "", "clickable", elements, 1, nil)
			found := len(ranked) > 0 && ranked[0].Explain.Score.Total > 0.2
			if neg {
				return !found, nil
			}
			return found, nil
		}
	}

	// 2. Handle text presence: text 'Target' is [not] present
	if strings.Contains(cond, "is present") || strings.Contains(cond, "is not present") {
		neg := strings.Contains(cond, "is not present")
		start := strings.Index(cond, "'")
		end := strings.LastIndex(cond, "'")
		target := ""
		if start != -1 && end != -1 && start < end {
			target = cond[start+1 : end]
		}

		elements, err := rt.loadSnapshot(ctx)
		if err != nil {
			return false, err
		}
		ranked := scorer.Rank(target, "", "none", elements, 1, nil)
		found := len(ranked) > 0 && ranked[0].Explain.Score.Total > 0.2
		if neg {
			return !found, nil
		}
		return found, nil
	}

	// 3. Handle variable comparisons: {var} == 'val', $var != 'val'
	if strings.HasPrefix(cond, "{") || strings.HasPrefix(cond, "$") {
		// Resolve variables first
		resolved := rt.resolveVariables(cond)
		if strings.Contains(resolved, " == ") {
			parts := strings.SplitN(resolved, " == ", 2)
			v1 := strings.TrimSpace(parts[0])
			v2 := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
			return v1 == v2, nil
		}
		if strings.Contains(resolved, " != ") {
			parts := strings.SplitN(resolved, " != ", 2)
			v1 := strings.TrimSpace(parts[0])
			v2 := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
			return v1 != v2, nil
		}
		if strings.Contains(resolved, " contains ") {
			parts := strings.SplitN(resolved, " contains ", 2)
			v1 := strings.TrimSpace(parts[0])
			v2 := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
			return strings.Contains(v1, v2), nil
		}
		// Truthy check for {var}
		val := strings.TrimSpace(resolved)
		return val != "" && val != "false" && val != "0" && val != "null", nil
	}

	return false, fmt.Errorf("unknown condition format: %q", cond)
}

func (rt *Runtime) handleMock(ctx context.Context, cmd dsl.Command) error {
	method := rt.resolveVariables(cmd.MockMethod)
	pattern := rt.resolveVariables(cmd.MockPattern)
	mockFile := rt.resolveVariables(cmd.MockFile)
	if method == "" || pattern == "" || mockFile == "" {
		return fmt.Errorf("MOCK: invalid syntax — expected MOCK <METHOD> \"<pattern>\" with '<file>'")
	}

	// Resolve mock file path: hunt dir → CWD
	huntDir := ""
	if rt.sourcePath != "" {
		huntDir = filepath.Dir(rt.sourcePath)
	}
	candidates := []string{}
	if huntDir != "" {
		candidates = append(candidates, filepath.Join(huntDir, mockFile))
	}
	candidates = append(candidates, filepath.Join(huntDir, "..", mockFile))
	candidates = append(candidates, mockFile)

	var resolved string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			resolved = c
			break
		}
	}
	if resolved == "" {
		return fmt.Errorf("MOCK: file not found: %s", mockFile)
	}

	body, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("MOCK: failed to read mock file %s: %w", mockFile, err)
	}

	contentType := "text/plain"
	if strings.HasSuffix(resolved, ".json") {
		contentType = "application/json"
	}

	key := method + " " + pattern
	rt.mockRules[key] = mockRule{
		Method:      method,
		Pattern:     pattern,
		Body:        string(body),
		ContentType: contentType,
	}

	rt.logger.ActionDetail("🔀", "MOCK %s *%s → %s", method, pattern, mockFile)

	// Inject the mock override JS into the page.
	if err := rt.applyMockJS(ctx); err != nil {
		return fmt.Errorf("MOCK: failed to inject mock JS: %w", err)
	}
	return nil
}

func (rt *Runtime) applyMockJS(ctx context.Context) error {
	js := `(function(){
  if (window.__manulMockApplied) return;
  window.__manulMockApplied = true;
  window.__manulMocks = window.__manulMocks || {};
  const origFetch = window.fetch;
  window.fetch = function(url, opts) {
    var method = (opts && opts.method || 'GET').toUpperCase();
    var mock = window.__manulMocks[method + ' ' + url];
    if (mock) {
      return Promise.resolve(new Response(mock.body, {
        status: 200,
        headers: {'Content-Type': mock.contentType}
      }));
    }
    return origFetch(url, opts);
  };
})();`
	_, err := rt.page.EvalJS(ctx, js)
	if err != nil {
		return err
	}

	// Register each mock rule.
	for key, rule := range rt.mockRules {
		regJS := fmt.Sprintf(`window.__manulMocks[%q] = {body: %q, contentType: %q};`,
			key, rule.Body, rule.ContentType)
		_, err := rt.page.EvalJS(ctx, regJS)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rt *Runtime) resolveVariables(s string) string {
	return rt.vars.Interpolate(s)
}

func (rt *Runtime) passesAntiPhantomGuard(mode string, query string, el dom.ElementSnapshot) bool {
	if mode != string(dsl.ModeInput) && mode != "select" {
		return true
	}

	q := strings.ToLower(query)
	words := strings.Fields(q)
	if len(words) == 0 {
		return true
	}

	// Collected signals for this element
	signals := el.AllTextSignals()
	signals = append(signals, el.HTMLId, el.Tag)

	for _, s := range signals {
		s_l := strings.ToLower(s)
		for _, w := range words {
			if len(w) >= 2 && strings.Contains(s_l, w) {
				return true
			}
		}
	}
	rt.logger.ActionDetail("👻", "ANTI-PHANTOM GUARD: heuristic choice %q for target %q has low keyword correlation.", el.Tag, query)
	return false
}

func (rt *Runtime) autoAnnotateNavigate(ctx context.Context, url string) {
	pageName := rt.pages.LookupPageName(url)
	if pageName == "" {
		pageName = url
	}
	rt.logger.ActionDetail("📍", "Auto-Nav: %s", pageName)
}

func resolveRestrictiveCandidates(targetPath, typeHint string, mode dsl.InteractionMode, elements []dom.ElementSnapshot, anchor *scorer.AnchorContext, logger *utils.Logger) ([]scorer.RankedCandidate, string) {
	selfRanked := scorer.Rank(targetPath, typeHint, string(mode), elements, 5, anchor)
	if len(selfRanked) > 0 && selfRanked[0].Explain.Score.Total >= ThresholdHighConfidence {
		return selfRanked, "restrictive-pass1"
	}

	anchorRanked := scorer.Rank(targetPath, typeHint, string(dsl.ModeNone), elements, 8, anchor)
	if len(anchorRanked) == 0 {
		return nil, "restrictive-pass2"
	}

	bestScore := -1.0
	var bestRanked []scorer.RankedCandidate
	bestStrategy := "restrictive-anchor"

	for _, anchorCandidate := range anchorRanked {
		if anchorCandidate.Element.IsInteractive(string(mode)) {
			candidateScore := anchorCandidate.Explain.Score.Total + 0.05
			if candidateScore > bestScore {
				bestScore = candidateScore
				bestRanked = []scorer.RankedCandidate{anchorCandidate}
				bestStrategy = "restrictive-pass2"
			}
			continue
		}

		newAnchor := &scorer.AnchorContext{
			Rect:       anchorCandidate.Element.Rect,
			XPath:      anchorCandidate.Element.XPath,
			FrameIndex: anchorCandidate.Element.FrameIndex,
			Words:      scorer.SignificantWords(anchorCandidate.Element.VisibleText),
		}
		if mode == dsl.ModeCheckbox {
			rowScoped := checkboxCandidatesInSameRow(anchorCandidate.Element, elements)
			if len(rowScoped) > 0 {
				rowRanked := scorer.Rank("", typeHint, string(mode), rowScoped, 5, newAnchor)
				if pass3CandidateAcceptable(rowRanked) {
					candidateScore := rowRanked[0].Explain.Score.Total + anchorCandidate.Explain.Score.Total*0.35 + 0.15
					if candidateScore > bestScore {
						bestScore = candidateScore
						bestRanked = rowRanked
						bestStrategy = "restrictive-pass3-row"
					}
					continue
				}
			}
		}
		rankedFallback := scorer.Rank("", typeHint, string(mode), elements, 5, newAnchor)
		if !pass3CandidateAcceptable(rankedFallback) {
			continue
		}
		candidateScore := rankedFallback[0].Explain.Score.Total + anchorCandidate.Explain.Score.Total*0.25
		if candidateScore > bestScore {
			bestScore = candidateScore
			bestRanked = rankedFallback
			bestStrategy = "restrictive-pass3"
		}
	}

	if len(bestRanked) > 0 {
		if bestStrategy == "restrictive-pass3" && logger != nil {
			logger.Info("Resolved restrictive target %q via multi-anchor nearby control search.", targetPath)
		}
		return bestRanked, bestStrategy
	}

	return anchorRanked, "restrictive-anchor"
}

func (rt *Runtime) applyContextualFilters(cmd dsl.Command, elements []dom.ElementSnapshot) ([]dom.ElementSnapshot, string, error) {
	filtered := elements
	var strategies []string

	if cmd.OnRegion != "" {
		regionFiltered := filterRegionCandidates(cmd.OnRegion, filtered)
		if len(regionFiltered) == 0 {
			return nil, "", fmt.Errorf("no candidates found in region %q", cmd.OnRegion)
		}
		filtered = regionFiltered
		strategies = append(strategies, "on-"+normalizeContextLabel(cmd.OnRegion))
	}

	if cmd.InsideRowText != "" {
		rowAnchor, err := rt.resolveStructuralAnchor(cmd.InsideRowText, filtered)
		if err != nil {
			return nil, "", fmt.Errorf("inside row anchor not found: %q", cmd.InsideRowText)
		}
		rowFiltered := candidatesInSameRow(rowAnchor, filtered)
		if len(rowFiltered) == 0 {
			return nil, "", fmt.Errorf("no candidates found inside row %q", cmd.InsideRowText)
		}
		filtered = rowFiltered
		strategies = append(strategies, "inside-row")
	}

	if cmd.InsideContainer != "" {
		containerAnchor, err := rt.resolveStructuralAnchor(cmd.InsideContainer, filtered)
		if err != nil {
			return nil, "", fmt.Errorf("inside container not found: %q", cmd.InsideContainer)
		}
		containerFiltered := descendantsOf(containerAnchor, filtered)
		if len(containerFiltered) == 0 {
			return nil, "", fmt.Errorf("no candidates found inside container %q", cmd.InsideContainer)
		}
		filtered = containerFiltered
		strategies = append(strategies, "inside-container")
	}

	return filtered, strings.Join(strategies, "+"), nil
}

func checkboxCandidatesInSameRow(anchor dom.ElementSnapshot, elements []dom.ElementSnapshot) []dom.ElementSnapshot {
	rowPrefix := rowXPathPrefix(anchor.XPath)
	if rowPrefix == "" {
		return nil
	}
	var out []dom.ElementSnapshot
	for _, element := range elements {
		if !element.IsInteractive(string(dsl.ModeCheckbox)) {
			continue
		}
		if strings.HasPrefix(element.XPath, rowPrefix+"/") {
			out = append(out, element)
		}
	}
	return out
}

func candidatesInSameRow(anchor dom.ElementSnapshot, elements []dom.ElementSnapshot) []dom.ElementSnapshot {
	rowPrefix := rowXPathPrefix(anchor.XPath)
	if rowPrefix == "" {
		return nil
	}
	var out []dom.ElementSnapshot
	for _, element := range elements {
		if strings.HasPrefix(element.XPath, rowPrefix+"/") {
			out = append(out, element)
		}
	}
	return out
}

func descendantsOf(container dom.ElementSnapshot, elements []dom.ElementSnapshot) []dom.ElementSnapshot {
	prefix := strings.TrimRight(container.XPath, "/") + "/"
	var out []dom.ElementSnapshot
	for _, element := range elements {
		if element.XPath == container.XPath {
			continue
		}
		if strings.HasPrefix(element.XPath, prefix) {
			out = append(out, element)
		}
	}
	return out
}

func rowXPathPrefix(xpath string) string {
	parts := strings.Split(strings.Trim(xpath, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	prefix := make([]string, 0, len(parts))
	for _, part := range parts {
		prefix = append(prefix, part)
		if strings.HasPrefix(part, "tr[") {
			return "/" + strings.Join(prefix, "/")
		}
	}
	return ""
}

func filterRegionCandidates(region string, elements []dom.ElementSnapshot) []dom.ElementSnapshot {
	viewportHeight := inferredViewportHeight(elements)
	var out []dom.ElementSnapshot
	for _, element := range elements {
		if elementMatchesRegion(region, element, viewportHeight) {
			out = append(out, element)
		}
	}
	return out
}

func inferredViewportHeight(elements []dom.ElementSnapshot) float64 {
	maxBottom := 0.0
	for _, element := range elements {
		bottom := element.Rect.Top + element.Rect.Height
		if bottom > maxBottom {
			maxBottom = bottom
		}
	}
	if maxBottom <= 0 {
		return 1000
	}
	return maxBottom
}

func elementMatchesRegion(region string, el dom.ElementSnapshot, viewportHeight float64) bool {
	region = normalizeContextLabel(region)
	if viewportHeight <= 0 {
		viewportHeight = 1000
	}
	bottom := el.Rect.Top + el.Rect.Height
	switch region {
	case "header":
		for _, ancestor := range el.Ancestors {
			ancestor = normalizeContextLabel(ancestor)
			if ancestor == "header" || ancestor == "nav" {
				return true
			}
		}
		return el.Rect.Top >= 0 && el.Rect.Top <= viewportHeight*0.15
	case "footer":
		for _, ancestor := range el.Ancestors {
			ancestor = normalizeContextLabel(ancestor)
			if ancestor == "footer" {
				return true
			}
		}
		return bottom >= viewportHeight*0.85
	default:
		return true
	}
}

func normalizeContextLabel(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), "-")
}

func isGenericListContainer(query string) bool {
	lower := strings.ToLower(strings.TrimSpace(query))
	lower = strings.TrimPrefix(lower, "the ")
	return lower == "list" || lower == "dropdown" || lower == "dropdown list" || lower == "listbox"
}

func isDropdownLikeQuery(query string) bool {
	lower := strings.ToLower(query)
	return strings.Contains(lower, "dropdown") || strings.Contains(lower, "combo box") || strings.Contains(lower, "combobox")
}

func isLikelyDropdownOptionQuery(query string) bool {
	lower := strings.ToLower(strings.TrimSpace(query))
	return strings.HasPrefix(lower, "item ") || strings.HasPrefix(lower, "option ")
}

func isShadowLikeQuery(query string) bool {
	return strings.Contains(strings.ToLower(query), "shadow")
}

func isDropdownLikeElement(el dom.ElementSnapshot) bool {
	if el.Tag == "select" || strings.EqualFold(el.Role, "combobox") || strings.EqualFold(el.Role, "listbox") {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{el.Name, el.HTMLId, el.ClassName, el.Placeholder, el.AriaLabel}, " "))
	return strings.Contains(haystack, "dropdown") || strings.Contains(haystack, "combo") || strings.Contains(haystack, "select")
}

func rankForVerifyState(target, state string, elements []dom.ElementSnapshot, logger *utils.Logger) []scorer.RankedCandidate {
	mode := dsl.ModeNone
	typeHint := ""
	switch strings.ToLower(state) {
	case "checked", "unchecked":
		mode = dsl.ModeCheckbox
		typeHint = "checkbox"
	case "selected":
		mode = dsl.ModeSelect
	}
	if mode != dsl.ModeNone && target != "" {
		ranked, _ := resolveRestrictiveCandidates(target, typeHint, mode, elements, nil, nil)
		if len(ranked) > 0 {
			return ranked
		}
	}
	return scorer.Rank(target, "", "none", elements, 1, nil)
}

func elementMatchesVerifyState(el dom.ElementSnapshot, state string) bool {
	switch strings.ToLower(state) {
	case "checked":
		return el.IsChecked
	case "unchecked":
		return !el.IsChecked
	case "enabled":
		return !el.IsDisabled
	case "disabled":
		return el.IsDisabled
	case "visible":
		return el.IsVisible && !el.IsHidden
	case "hidden", "disappear":
		return !el.IsVisible || el.IsHidden
	case "selected":
		return el.IsSelected
	default:
		return false
	}
}

func missingElementSatisfiesVerifyState(state string) bool {
	switch strings.ToLower(state) {
	case "hidden", "disappear":
		return true
	default:
		return false
	}
}

func expectedVerifyStateDescription(state string, negated bool) string {
	base := strings.ToLower(strings.TrimSpace(state))
	if base == "" {
		base = "present"
	}
	if negated {
		return "NOT " + base
	}
	return base
}

func actualVerifyStateDescription(state string, matches bool) string {
	base := strings.ToLower(strings.TrimSpace(state))
	if base == "" {
		return "unknown"
	}
	if matches {
		return base
	}
	switch base {
	case "checked":
		return "unchecked"
	case "unchecked":
		return "checked"
	case "enabled":
		return "disabled"
	case "disabled":
		return "enabled"
	case "visible":
		return "hidden"
	case "hidden", "disappear":
		return "visible"
	case "selected":
		return "not selected"
	default:
		return "not " + base
	}
}

func verifyRankedCandidateAcceptable(state string, ranked []scorer.RankedCandidate) bool {
	if len(ranked) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "checked", "unchecked", "selected":
		return !selectionIsAmbiguous(ranked)
	default:
		return ranked[0].Explain.Score.Total > 0.3
	}
}

func (rt *Runtime) ensureCheckboxTargetState(ctx context.Context, target string, desired bool, initialRanked []scorer.RankedCandidate) error {
	if waitErr := rt.page.Wait(ctx, 150*time.Millisecond); waitErr != nil {
		return waitErr
	}
	rt.invalidateSnapshot()
	matched, err := rt.checkboxTargetHasState(ctx, target, desired)
	if err != nil {
		return err
	}
	if matched {
		return nil
	}

	retryCandidates, err := rt.collectCheckboxRetryCandidates(ctx, target, initialRanked)
	if err != nil {
		return err
	}
	tried := map[string]bool{}
	for _, candidate := range retryCandidates {
		key := candidate.Element.XPath
		if key == "" {
			key = fmt.Sprintf("id:%d", candidate.Element.ID)
		}
		if tried[key] {
			continue
		}
		tried[key] = true

		if err := rt.page.SetChecked(ctx, candidate.Element.ID, candidate.Element.XPath, desired); err != nil {
			continue
		}
		if waitErr := rt.page.Wait(ctx, 150*time.Millisecond); waitErr != nil {
			return waitErr
		}
		rt.invalidateSnapshot()
		matched, err = rt.checkboxTargetHasState(ctx, target, desired)
		if err != nil {
			return err
		}
		if matched {
			return nil
		}
	}

	return fmt.Errorf("checkbox target %q did not reach checked=%t", target, desired)
}

func (rt *Runtime) checkboxTargetHasState(ctx context.Context, target string, desired bool) (bool, error) {
	elements, err := rt.loadSnapshot(ctx)
	if err != nil {
		return false, err
	}
	ranked := rankForVerifyState(target, "checked", elements, nil)
	if !verifyRankedCandidateAcceptable("checked", ranked) {
		return false, nil
	}
	return ranked[0].Element.IsChecked == desired, nil
}

func (rt *Runtime) collectCheckboxRetryCandidates(ctx context.Context, target string, initialRanked []scorer.RankedCandidate) ([]scorer.RankedCandidate, error) {
	elements, err := rt.loadSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	var candidates []scorer.RankedCandidate
	candidates = append(candidates, initialRanked...)
	if restrictive, _ := resolveRestrictiveCandidates(target, "checkbox", dsl.ModeCheckbox, elements, nil, nil); len(restrictive) > 0 {
		candidates = append(candidates, restrictive...)
	}
	candidates = append(candidates, scorer.Rank(target, "checkbox", string(dsl.ModeCheckbox), elements, 5, nil)...)
	candidates = append(candidates, scorer.Rank(target, "", string(dsl.ModeNone), elements, 5, nil)...)
	return candidates, nil
}

func (rt *Runtime) rememberStickyCheckboxState(target string, checked bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return
	}
	if rt.stickyCheckboxStates == nil {
		rt.stickyCheckboxStates = make(map[string]bool)
	}
	rt.stickyCheckboxStates[target] = checked
}

func (rt *Runtime) reconcileStickyCheckboxStates(ctx context.Context) error {
	if len(rt.stickyCheckboxStates) == 0 {
		return nil
	}
	states := rt.stickyCheckboxStates
	rt.stickyCheckboxStates = nil
	elements, err := rt.loadSnapshot(ctx)
	if err != nil {
		rt.stickyCheckboxStates = states
		return err
	}
	for target, desired := range states {
		ranked := rankForVerifyState(target, "checked", elements, nil)
		if !verifyRankedCandidateAcceptable("checked", ranked) {
			continue
		}
		winner := ranked[0].Element
		if winner.IsChecked == desired {
			continue
		}
		if err := rt.page.SetChecked(ctx, winner.ID, winner.XPath, desired); err != nil {
			return err
		}
		rt.invalidateSnapshot()
		elements, err = rt.loadSnapshot(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rt *Runtime) tryClickNearbyDropdownControl(ctx context.Context, anchor dom.ElementSnapshot) (bool, error) {
	js := fmt.Sprintf(`(() => {
		const anchor = (window.__manulReg && window.__manulReg[%[1]d]) || document.evaluate(%[2]q, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
		if (!anchor) return false;

		const isVisible = (node) => {
			if (!node) return false;
			const rect = node.getBoundingClientRect();
			const cs = window.getComputedStyle(node);
			return cs.display !== 'none' && cs.visibility !== 'hidden' && parseFloat(cs.opacity || '1') > 0 && rect.width > 0 && rect.height > 0;
		};

		let root = anchor.closest('.widget, .form-group, section, article, aside, div') || anchor.parentElement || anchor;
		const preferredSelectors = [
			'#comboBox',
			'[role="combobox"]',
			'[id*="combo"]',
			'[class*="combo"]',
			'[id*="dropdown"]',
			'[class*="dropdown"]'
		];
		const selectors = [
			'#comboBox',
			'[role="combobox"]',
			'select',
			'input[list]',
			'input[type="text"]',
			'[id*="combo"]',
			'[class*="combo"]',
			'[id*="dropdown"]',
			'[class*="dropdown"]'
		];

		let target = null;
		for (const selector of preferredSelectors) {
			const matches = Array.from(document.querySelectorAll(selector)).filter(isVisible);
			const preferred = matches.find(node => node.id === 'comboBox' || node.getAttribute('role') === 'combobox' || node.tagName === 'INPUT');
			if (preferred) {
				target = preferred;
				break;
			}
		}

		for (const selector of selectors) {
			if (target) break;
			const matches = Array.from(root.querySelectorAll(selector)).filter(isVisible);
			const preferred = matches.find(node => node.id === 'comboBox' || node.getAttribute('role') === 'combobox' || node.tagName === 'SELECT' || node.tagName === 'INPUT');
			if (preferred) {
				target = preferred;
				break;
			}
			if (matches.length > 0) {
				target = matches[0];
				break;
			}
		}

		if (!target && root.parentElement) {
			root = root.parentElement;
			for (const selector of selectors) {
				const candidate = root.querySelector(selector);
				if (isVisible(candidate)) {
					target = candidate;
					break;
				}
			}
		}

		if (!target) return false;
		target.scrollIntoView({block: 'center', inline: 'center'});
		if (typeof target.focus === 'function') target.focus();
		['mousedown', 'mouseup', 'click'].forEach((evt) => {
			target.dispatchEvent(new MouseEvent(evt, { bubbles: true, cancelable: true, view: window }));
		});
		if (typeof target.click === 'function') target.click();
		return true;
	})()`, anchor.ID, anchor.XPath)

	raw, err := rt.page.EvalJS(ctx, js)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(raw)) == "true", nil
}

func (rt *Runtime) tryClickDropdownTriggerByLabel(ctx context.Context, targetPath string) (bool, error) {
	js := fmt.Sprintf(`(() => {
		const normalize = (value) => (value || '').replace(/\s+/g, ' ').trim().toLowerCase();
		const isVisible = (node) => {
			if (!node) return false;
			const rect = node.getBoundingClientRect();
			const cs = window.getComputedStyle(node);
			return cs.display !== 'none' && cs.visibility !== 'hidden' && parseFloat(cs.opacity || '1') > 0 && rect.width > 0 && rect.height > 0;
		};
		const clickNode = (node) => {
			if (!node) return false;
			node.scrollIntoView({ block: 'center', inline: 'center' });
			if (typeof node.focus === 'function') node.focus();
			['mousedown', 'mouseup', 'click'].forEach((evt) => {
				node.dispatchEvent(new MouseEvent(evt, { bubbles: true, cancelable: true, view: window }));
			});
			if (typeof node.click === 'function') node.click();
			return true;
		};

		const wanted = normalize(%q);
		const triggerSelectors = ['#comboBox', '[role="combobox"]', 'input[list]', '[id*="combo"]', '[class*="combo"]'];
		const all = Array.from(document.querySelectorAll('h1,h2,h3,h4,h5,h6,label,legend,span,div,p'));
		const labels = all.filter((node) => normalize(node.innerText) === wanted);
		for (const label of labels) {
			let scope = label;
			for (let depth = 0; depth < 6 && scope; depth += 1) {
				for (const selector of triggerSelectors) {
					const local = Array.from(scope.querySelectorAll(selector)).find(isVisible);
					if (local && clickNode(local)) return true;
				}
				let sibling = scope.nextElementSibling;
				while (sibling) {
					for (const selector of triggerSelectors) {
						const siblingMatch = sibling.matches(selector) ? sibling : sibling.querySelector(selector);
						if (isVisible(siblingMatch) && clickNode(siblingMatch)) return true;
					}
					sibling = sibling.nextElementSibling;
				}
				scope = scope.parentElement;
			}
		}

		for (const selector of triggerSelectors) {
			const fallback = Array.from(document.querySelectorAll(selector)).find(isVisible);
			if (fallback && clickNode(fallback)) return true;
		}
		return false;
	})()`, targetPath)

	raw, err := rt.page.EvalJS(ctx, js)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(raw)) == "true", nil
}

func (rt *Runtime) trySetShadowInputValue(ctx context.Context, targetPath, value string) (bool, error) {
	js := fmt.Sprintf(`(() => {
		const normalize = (text) => (text || '').replace(/\s+/g, ' ').trim().toLowerCase();
		const query = normalize(%q);
		const value = %q;
		const hosts = [];
		const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_ELEMENT);
		while (walker.nextNode()) {
			const node = walker.currentNode;
			if (node.shadowRoot) hosts.push(node);
		}
		for (const host of hosts) {
			const nearby = normalize((host.parentElement && host.parentElement.innerText) || host.innerText || host.id || host.className || '');
			if (query && !nearby.includes('shadow') && !query.includes('shadow')) continue;
			const control = host.shadowRoot.querySelector('input[type="text"], textarea, input:not([type]), input[type="search"], input[type="email"], input[type="tel"], input[type="url"]');
			if (!control) continue;
			host.scrollIntoView({ block: 'center', inline: 'center' });
			control.focus();
			control.value = value;
			control.dispatchEvent(new Event('input', { bubbles: true }));
			control.dispatchEvent(new Event('change', { bubbles: true }));
			return true;
		}
		return false;
	})()`, targetPath, value)

	raw, err := rt.page.EvalJS(ctx, js)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(raw)) == "true", nil
}

func (rt *Runtime) tryClickVisibleDropdownOption(ctx context.Context, targetPath string) (bool, error) {
	js := fmt.Sprintf(`(() => {
		const normalize = (value) => (value || '').replace(/\s+/g, ' ').trim().toLowerCase();
		const isVisible = (node) => {
			if (!node) return false;
			const rect = node.getBoundingClientRect();
			const cs = window.getComputedStyle(node);
			return cs.display !== 'none' && cs.visibility !== 'hidden' && parseFloat(cs.opacity || '1') > 0 && rect.width > 0 && rect.height > 0;
		};

		const wanted = normalize(%q);
		let list = document.querySelector('#dropdown') || document.querySelector('[role="listbox"]') || document.querySelector('[class*="dropdown"]');
		const combo = document.querySelector('#comboBox') || document.querySelector('[role="combobox"]') || document.querySelector('input[list]') || document.querySelector('[id*="combo"]');
		if ((!list || !isVisible(list)) && combo && isVisible(combo)) {
			combo.scrollIntoView({ block: 'center', inline: 'center' });
			if (typeof combo.focus === 'function') combo.focus();
			['mousedown', 'mouseup', 'click'].forEach((evt) => {
				combo.dispatchEvent(new MouseEvent(evt, { bubbles: true, cancelable: true, view: window }));
			});
			if (typeof combo.click === 'function') combo.click();
		}
		list = document.querySelector('#dropdown') || document.querySelector('[role="listbox"]') || document.querySelector('[class*="dropdown"]');
		if (!list || !isVisible(list)) return false;

		const candidates = Array.from(list.querySelectorAll('.option, [role="option"], div, li'))
			.filter(isVisible)
			.filter((node) => normalize(node.innerText) !== '');
		const target = candidates.find((node) => normalize(node.innerText) === wanted) ||
			candidates.find((node) => normalize(node.innerText).includes(wanted));
		if (!target) return false;

		target.scrollIntoView({ block: 'center', inline: 'nearest' });
		['mousedown', 'mouseup', 'click'].forEach((evt) => {
			target.dispatchEvent(new MouseEvent(evt, { bubbles: true, cancelable: true, view: window }));
		});
		if (typeof target.click === 'function') target.click();
		return true;
	})()`, targetPath)

	raw, err := rt.page.EvalJS(ctx, js)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(raw)) == "true", nil
}

func (rt *Runtime) loadSnapshot(ctx context.Context) ([]dom.ElementSnapshot, error) {
	if !rt.cfg.DisableCache && rt.cachedElements != nil {
		return rt.cachedElements, nil
	}
	raw, err := rt.page.CallProbe(ctx, heuristics.BuildSnapshotProbe(), nil)
	if err != nil {
		return nil, fmt.Errorf("probe failed: %w", err)
	}
	elements, err := heuristics.ParseProbeResult(raw)
	if err != nil {
		return nil, fmt.Errorf("parse probe failed: %w", err)
	}
	if !rt.cfg.DisableCache {
		rt.cachedElements = elements
	}
	return elements, nil
}

func (rt *Runtime) invalidateSnapshot() {
	rt.cachedElements = nil
}

func appendRankedCandidates(res *explain.ExecutionResult, ranked []scorer.RankedCandidate, limit int) {
	if limit <= 0 || len(ranked) < limit {
		limit = len(ranked)
	}
	for i := 0; i < limit; i++ {
		res.RankedCandidates = append(res.RankedCandidates, ranked[i].Explain)
	}
}

func collapseNestedDuplicateRankedCandidates(ranked []scorer.RankedCandidate) []scorer.RankedCandidate {
	if len(ranked) < 2 {
		return ranked
	}
	collapsed := make([]scorer.RankedCandidate, 0, len(ranked))
	for _, candidate := range ranked {
		merged := false
		for i := range collapsed {
			if !nestedDuplicateRankedCandidates(collapsed[i], candidate) {
				continue
			}
			if preferMoreSpecificRankedCandidate(candidate, collapsed[i]) {
				collapsed[i] = candidate
			}
			merged = true
			break
		}
		if !merged {
			collapsed = append(collapsed, candidate)
		}
	}
	for i := range collapsed {
		collapsed[i].Explain.Rank = i + 1
		collapsed[i].Explain.Chosen = i == 0
	}
	return collapsed
}

func nestedDuplicateRankedCandidates(existing, candidate scorer.RankedCandidate) bool {
	if math.Abs(existing.Explain.Score.Total-candidate.Explain.Score.Total) > 0.02 {
		return false
	}
	textA := normalizeCandidateText(existing.Element.VisibleText)
	textB := normalizeCandidateText(candidate.Element.VisibleText)
	if textA == "" || textA != textB {
		return false
	}
	if !(isXPathAncestor(existing.Element.XPath, candidate.Element.XPath) || isXPathAncestor(candidate.Element.XPath, existing.Element.XPath)) {
		return false
	}
	return rectIntersectionRatio(existing.Element.Rect, candidate.Element.Rect) >= 0.85
}

func preferMoreSpecificRankedCandidate(candidate, existing scorer.RankedCandidate) bool {
	candidateDepth := len(xpathParts(candidate.Element.XPath))
	existingDepth := len(xpathParts(existing.Element.XPath))
	if candidateDepth != existingDepth {
		return candidateDepth > existingDepth
	}
	candidateArea := candidate.Element.Rect.Width * candidate.Element.Rect.Height
	existingArea := existing.Element.Rect.Width * existing.Element.Rect.Height
	if candidateArea != existingArea {
		return candidateArea < existingArea
	}
	return candidate.Element.ID > existing.Element.ID
}

func normalizeCandidateText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func xpathParts(xpath string) []string {
	var parts []string
	for _, part := range strings.Split(xpath, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func isXPathAncestor(ancestor, descendant string) bool {
	if ancestor == "" || descendant == "" || ancestor == descendant {
		return false
	}
	ancestorParts := xpathParts(ancestor)
	descendantParts := xpathParts(descendant)
	if len(ancestorParts) >= len(descendantParts) {
		return false
	}
	for i := range ancestorParts {
		if ancestorParts[i] != descendantParts[i] {
			return false
		}
	}
	return true
}

func rectIntersectionRatio(a, b dom.Rect) float64 {
	left := math.Max(a.Left, b.Left)
	top := math.Max(a.Top, b.Top)
	right := math.Min(a.Right, b.Right)
	bottom := math.Min(a.Bottom, b.Bottom)
	if right <= left || bottom <= top {
		return 0.0
	}
	intersection := (right - left) * (bottom - top)
	areaA := a.Width * a.Height
	areaB := b.Width * b.Height
	if areaA <= 0 || areaB <= 0 {
		return 0.0
	}
	return intersection / math.Min(areaA, areaB)
}

func selectionIsAmbiguous(ranked []scorer.RankedCandidate) bool {
	if len(ranked) == 0 {
		return true
	}
	best := ranked[0].Explain.Score.Total
	if best < ThresholdAmbiguous {
		return true
	}
	if len(ranked) == 1 {
		return false
	}
	return best < ThresholdHighConfidence && best-ranked[1].Explain.Score.Total < ThresholdRunnerUpGap
}

func pass3CandidateAcceptable(ranked []scorer.RankedCandidate) bool {
	if len(ranked) == 0 {
		return false
	}
	best := ranked[0].Explain.Score
	if best.Total < ThresholdPass3Total || best.ProximityScore < ThresholdPass3Proximity {
		return false
	}
	if len(ranked) == 1 {
		return true
	}
	return best.Total-ranked[1].Explain.Score.Total >= ThresholdPass3Gap
}
