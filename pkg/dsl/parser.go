// Package dsl implements the ManulEngine (Go) .hunt DSL parser.
//
// A .hunt file is a sequence of natural-language-style automation commands
// with optional @header directives, STEP blocks, and control-flow constructs.
//
// The grammar is intentionally flexible — the parser uses keyword matching
// rather than strict BNF, prioritising human-readability over rigidity.
package dsl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ── Command types ──────────────────────────────────────────────────────────────

// CommandType is the DSL verb for a command.
type CommandType string

const (
	CmdNavigate        CommandType = "NAVIGATE"
	CmdOpenApp         CommandType = "OPEN_APP"
	CmdClick           CommandType = "CLICK"
	CmdDoubleClick     CommandType = "DOUBLE_CLICK"
	CmdRightClick      CommandType = "RIGHT_CLICK"
	CmdFill            CommandType = "FILL"
	CmdType            CommandType = "TYPE"
	CmdSelect          CommandType = "SELECT"
	CmdCheck           CommandType = "CHECK"
	CmdUncheck         CommandType = "UNCHECK"
	CmdVerify          CommandType = "VERIFY"
	CmdVerifySoft      CommandType = "VERIFY_SOFT"
	CmdVerifyField     CommandType = "VERIFY_FIELD"
	CmdExtract         CommandType = "EXTRACT"
	CmdScroll          CommandType = "SCROLL"
	CmdPress           CommandType = "PRESS"
	CmdWait            CommandType = "WAIT"
	CmdWaitFor         CommandType = "WAIT_FOR"
	CmdWaitForResponse CommandType = "WAIT_FOR_RESPONSE"
	CmdHover           CommandType = "HOVER"
	CmdDrag            CommandType = "DRAG"
	CmdSet             CommandType = "SET"
	CmdPrint           CommandType = "PRINT"
	CmdScreenshot      CommandType = "SCREENSHOT"
	CmdHighlight       CommandType = "HIGHLIGHT"
	CmdRepeat          CommandType = "REPEAT"
	CmdForEach         CommandType = "FOR_EACH"
	CmdWhile           CommandType = "WHILE"
	CmdIf              CommandType = "IF"
	CmdElIf            CommandType = "ELIF"
	CmdElse            CommandType = "ELSE"
	CmdEndIf           CommandType = "END_IF"
	CmdEndFor          CommandType = "END_FOR"
	CmdEndWhile        CommandType = "END_WHILE"
	CmdEndRepeat       CommandType = "END_REPEAT"
	CmdCallGo          CommandType = "CALL_GO"
	CmdCallStep        CommandType = "CALL"
	CmdUse             CommandType = "USE"
	CmdUploadFile      CommandType = "UPLOAD_FILE"
	CmdUpload          CommandType = "UPLOAD_FILE" // alias for backward compatibility
	CmdPause           CommandType = "PAUSE"
	CmdDebugVars       CommandType = "DEBUG_VARS"
	CmdMock            CommandType = "MOCK"
	CmdUnknown         CommandType = "UNKNOWN"
)

// InteractionMode controls which elements are eligible for targeting.
type InteractionMode string

const (
	ModeClickable InteractionMode = "clickable"
	ModeInput     InteractionMode = "input"
	ModeCheckbox  InteractionMode = "checkbox"
	ModeSelect    InteractionMode = "select"
	ModeNone      InteractionMode = "none"
)

// ── Structures ─────────────────────────────────────────────────────────────────

// ImportSpec represents a single @import directive.
type ImportSpec struct {
	// Source is the .hunt file path (relative to the importing file).
	Source string
	// Names is the list of STEP block names to import. ["*"] means all.
	Names []string
	// Aliases maps importing name → local alias.
	Aliases map[string]string
}

// Command is a single parsed DSL instruction.
type Command struct {
	// Type is the DSL verb.
	Type CommandType
	// Raw is the original unparsed DSL text.
	Raw string
	// Verb is the first word of the raw text (normalised lowercase).
	Verb string
	// LineNum is the 1-based source line number in the .hunt file.
	LineNum int

	// StepBlock is the STEP label this command belongs to (if any).
	StepBlock string
	// Tags filters execution to specific @tag values.
	Tags []string

	// Target is the human-readable label of the element to interact with.
	Target string
	// TypeHint is the element-type keyword (button, link, field, …).
	TypeHint string
	// InteractionMode is the scoring mode for element disambiguation.
	InteractionMode InteractionMode

	// URL is the navigation destination for NAVIGATE commands.
	URL string
	// Value is the fill/select value.
	Value string

	// NearAnchor is the NEAR qualifier anchor label.
	NearAnchor string
	// OnRegion is the ON <region> qualifier (header, footer, sidebar, …).
	OnRegion string
	// InsideContainer is the INSIDE qualifier container label.
	InsideContainer string
	// InsideRowText is the "with <text>" clarifier for INSIDE.
	InsideRowText string

	// VerifyFieldKind is the attribute to verify for CmdVerifyField: "text", "value", "placeholder".
	VerifyFieldKind string

	// VerifyText is the text to verify for VERIFY commands.
	VerifyText string
	// VerifyNegated is true for "VERIFY that X is NOT present".
	VerifyNegated bool
	// VerifyState is the element state to verify (checked, visible, …).
	VerifyState string

	// ExtractVar is the variable name to store EXTRACT results into.
	ExtractVar string

	// ScrollDirection is "up" or "down" for SCROLL commands.
	ScrollDirection string
	// ScrollContainer is the element label to scroll within.
	ScrollContainer string

	// PressKey is the key combination for PRESS commands.
	PressKey string

	// WaitSeconds is the duration in seconds for WAIT commands.
	WaitSeconds float64
	// WaitForState is the element state for WAIT FOR commands.
	WaitForState string
	// WaitResponseURL is the URL pattern for WAIT FOR RESPONSE.
	WaitResponseURL string

	// SetVar is the variable name for SET commands.
	SetVar string
	// SetValue is the value for SET commands.
	SetValue string

	// DragSource is the source element label for DRAG commands.
	DragSource string
	// DragTarget is the drop target element label for DRAG commands.
	DragTarget string

	// RepeatCount is the iteration count for REPEAT commands.
	RepeatCount int
	// RepeatVar is the loop variable for REPEAT commands (default "i").
	RepeatVar string
	// ForEachVar is the loop variable for FOR EACH commands.
	ForEachVar string
	// ForEachCollection is the collection variable for FOR EACH commands.
	ForEachCollection string
	// WhileCondition is the condition expression for WHILE commands.
	WhileCondition string
	// IfCondition is the condition expression for IF/ELIF commands.
	IfCondition string

	// Condition is the generic condition string (used by IF/ELIF/WHILE).
	Condition string

	// PressTarget is the optional target element to focus before pressing the key.
	PressTarget string

	// UploadFile is the file path for UPLOAD commands (alias for UploadFilePath).
	UploadFile string

	// PrintText is the text to print for PRINT commands.
	PrintText string

	// ScreenshotName is the optional file label for SCREENSHOT commands.
	ScreenshotName string

	// CallStepName is the step block name to call for CALL_STEP commands.
	CallStepName string
	// GoCallName is the registered handler name for CALL GO commands.
	GoCallName string
	// GoCallArgs is the positional argument list for CALL GO commands.
	GoCallArgs []string
	// GoCallResultVar stores CALL GO results into {var} when provided.
	GoCallResultVar string

	// UploadFilePath is the file path for UPLOAD_FILE commands.
	UploadFilePath string

	// MockMethod is the HTTP method for MOCK commands (GET, POST, PUT, PATCH, DELETE).
	MockMethod string
	// MockPattern is the URL pattern for MOCK commands.
	MockPattern string
	// MockFile is the response file path for MOCK commands.
	MockFile string

	// Body is the block body for control-flow commands (REPEAT, FOR EACH, WHILE).
	Body []Command
	// Branches holds IF/ELIF/ELSE conditional branches.
	Branches []Branch
}

// Branch represents a single conditional path (IF, ELIF, ELSE).
type Branch struct {
	Kind      string
	Condition string
	Body      []Command
}

// Hunt is the fully parsed representation of a .hunt file.
type Hunt struct {
	// SourcePath is the absolute filesystem path to the .hunt file.
	SourcePath string
	// Title is the @title directive value.
	Title string
	// Context is the @context directive value.
	Context string
	// Tags are the file-level @tags.
	Tags []string
	// DataFile is the attached test data file (e.g., from @data_file).
	DataFile string
	// Schedule defines execution frequency (e.g., from @schedule).
	Schedule string
	// Exports holds names of variables exported across hunts (e.g., from @export).
	Exports []string
	// Vars holds @var declarations and runtime SET values.
	Vars map[string]string
	// ScriptAliases maps @script: aliases to their full dotted paths.
	ScriptAliases map[string]string
	// Imports contains parsed @import directives.
	Imports []ImportSpec
	// Blueprints holds imported STEP blocks keyed by name.
	Blueprints map[string][]Command
	// Commands is the ordered list of top-level parsed commands.
	Commands []Command
	// SetupCommands holds commands inside [SETUP] ... [END SETUP].
	SetupCommands []Command
	// TeardownCommands holds commands inside [TEARDOWN] ... [END TEARDOWN].
	TeardownCommands []Command
}

// Expand replaces all CmdUse and CmdCallStep placeholders in the Hunt's
// top-level commands and nested blocks with the actual commands from
// hunt.Blueprints. It should be called after ResolveImports.
func (h *Hunt) Expand() error {
	expanded, err := expandCommands(h.Commands, h.Blueprints, make(map[string]bool))
	if err != nil {
		return err
	}
	h.Commands = expanded

	if len(h.SetupCommands) > 0 {
		expanded, err = expandCommands(h.SetupCommands, h.Blueprints, make(map[string]bool))
		if err != nil {
			return err
		}
		h.SetupCommands = expanded
	}

	if len(h.TeardownCommands) > 0 {
		expanded, err = expandCommands(h.TeardownCommands, h.Blueprints, make(map[string]bool))
		if err != nil {
			return err
		}
		h.TeardownCommands = expanded
	}

	return nil
}

func expandCommands(cmds []Command, blueprints map[string][]Command, visited map[string]bool) ([]Command, error) {
	var out []Command
	for _, cmd := range cmds {
		if cmd.Type == CmdUse || cmd.Type == CmdCallStep {
			name := strings.ToLower(cmd.CallStepName)
			if visited[name] {
				return nil, fmt.Errorf("recursion detected in USE/CALL expansion: %s", name)
			}
			block, found := blueprints[name]
			if !found {
				return nil, fmt.Errorf("USE/CALL: block %q not found in blueprints", name)
			}
			// Recursive expansion of the block itself.
			visited[name] = true
			expandedBody, err := expandCommands(block, blueprints, visited)
			delete(visited, name) // clean up for sister branches
			if err != nil {
				return nil, err
			}
			out = append(out, expandedBody...)
		} else {
			// Expand nested blocks (IF, REPEAT, etc.)
			if len(cmd.Body) > 0 {
				eb, err := expandCommands(cmd.Body, blueprints, visited)
				if err != nil {
					return nil, err
				}
				cmd.Body = eb
			}
			for i := range cmd.Branches {
				eb, err := expandCommands(cmd.Branches[i].Body, blueprints, visited)
				if err != nil {
					return nil, err
				}
				cmd.Branches[i].Body = eb
			}
			out = append(out, cmd)
		}
	}
	return out, nil
}

// ── Parser ─────────────────────────────────────────────────────────────────────

// Parse reads a .hunt file from r and returns the parsed Hunt.
// Variable substitution of @var declarations is applied during parsing.
func Parse(r io.Reader) (*Hunt, error) {
	hunt := &Hunt{Vars: map[string]string{}}
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := parseLines(hunt, lines); err != nil {
		return nil, err
	}
	return hunt, nil
}

// ParseFile reads a .hunt file from the filesystem and returns the parsed Hunt.
func ParseFile(path string) (*Hunt, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}
	defer f.Close()
	hunt, err := Parse(f)
	if err != nil {
		return nil, err
	}
	hunt.SourcePath = path
	return hunt, nil
}

// parseLines is the main parsing loop, implementing stack-based block AST building via indentation.
func parseLines(hunt *Hunt, lines []string) error {
	var currentStep string
	var currentTags []string
	var inSetup, inTeardown bool

	type stackFrame struct {
		cmd    *Command
		branch *Branch // For IF commands, points to the active branch
		indent int
	}
	var stack []stackFrame

	appendCmd := func(cmd Command) *Command {
		if len(stack) == 0 {
			if inSetup {
				hunt.SetupCommands = append(hunt.SetupCommands, cmd)
				return &hunt.SetupCommands[len(hunt.SetupCommands)-1]
			}
			if inTeardown {
				hunt.TeardownCommands = append(hunt.TeardownCommands, cmd)
				return &hunt.TeardownCommands[len(hunt.TeardownCommands)-1]
			}
			hunt.Commands = append(hunt.Commands, cmd)
			return &hunt.Commands[len(hunt.Commands)-1]
		}
		top := stack[len(stack)-1]
		if top.branch != nil {
			top.branch.Body = append(top.branch.Body, cmd)
			return &top.branch.Body[len(top.branch.Body)-1]
		}
		top.cmd.Body = append(top.cmd.Body, cmd)
		return &top.cmd.Body[len(top.cmd.Body)-1]
	}

	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := 0
		for _, ch := range raw {
			if ch == ' ' {
				indent++
			} else if ch == '\t' {
				indent += 4
			} else {
				break
			}
		}

		upper := strings.ToUpper(trimmed)

		// SETUP / TEARDOWN block markers
		if upper == "[SETUP]" {
			inSetup = true
			inTeardown = false
			continue
		}
		if upper == "[END SETUP]" || upper == "[END_SETUP]" {
			inSetup = false
			continue
		}
		if upper == "[TEARDOWN]" {
			inTeardown = true
			inSetup = false
			continue
		}
		if upper == "[END TEARDOWN]" || upper == "[END_TEARDOWN]" {
			inTeardown = false
			continue
		}

		// In-line tags (apply to the NEXT command) - Case sensitive (@TAG vs @tags)
		if strings.HasPrefix(trimmed, "@TAG:") || strings.HasPrefix(trimmed, "@TAGS:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				for _, t := range strings.Split(parts[1], ",") {
					currentTags = append(currentTags, strings.TrimSpace(t))
				}
			}
			continue
		}

		// Metadata headers (@context, @title, @var, @tags)
		if strings.HasPrefix(trimmed, "@") {
			if err := parseDirective(hunt, trimmed); err != nil {
				return fmt.Errorf("line %d: %w", i+1, err)
			}
			continue
		}

		// Logical steps (optionally numbered: "1. STEP 1: ..." or "STEP: Login")
		if (strings.Contains(upper, "STEP ") || strings.HasPrefix(upper, "STEP:")) && !strings.Contains(upper, "'") && !strings.Contains(upper, "\"") {
			currentStep = strings.TrimSuffix(trimmed, ":")
			continue
		}
		if upper == "DONE." || upper == "DONE" {
			break
		}

		for len(stack) > 0 && indent <= stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
		}

		expanded := applyVars(hunt, trimmed)

		if strings.HasPrefix(upper, "ELIF ") || strings.HasPrefix(upper, "ELSE IF ") || upper == "ELSE:" || upper == "ELSE" {
			var lastCmd *Command
			if len(stack) == 0 {
				if len(hunt.Commands) > 0 {
					lastCmd = &hunt.Commands[len(hunt.Commands)-1]
				}
			} else {
				top := stack[len(stack)-1]
				if top.branch != nil {
					if len(top.branch.Body) > 0 {
						lastCmd = &top.branch.Body[len(top.branch.Body)-1]
					}
				} else {
					if len(top.cmd.Body) > 0 {
						lastCmd = &top.cmd.Body[len(top.cmd.Body)-1]
					}
				}
			}

			if lastCmd != nil && lastCmd.Type == CmdIf {
				kind := "elif"
				cond := ""
				if strings.HasPrefix(upper, "ELIF ") || strings.HasPrefix(upper, "ELSE IF ") {
					rest := stripPrefix(trimmed, "ELSE IF ", "ELIF ")
					cond = strings.TrimSuffix(strings.TrimSpace(rest), ":")
				} else {
					kind = "else"
				}

				lastCmd.Branches = append(lastCmd.Branches, Branch{
					Kind:      kind,
					Condition: cond,
					Body:      []Command{},
				})

				stack = append(stack, stackFrame{
					cmd:    lastCmd,
					branch: &lastCmd.Branches[len(lastCmd.Branches)-1],
					indent: indent,
				})
				continue
			}
		}

		cmd := parseCommandLine(expanded)
		cmd.Raw = trimmed
		cmd.LineNum = i + 1
		cmd.StepBlock = currentStep
		cmd.Tags = append([]string{}, currentTags...)
		currentTags = nil

		rewriteScriptAlias(&cmd, hunt.ScriptAliases)

		ptr := appendCmd(cmd)

		if ptr.Type == CmdIf {
			ptr.Branches = append(ptr.Branches, Branch{
				Kind:      "if",
				Condition: ptr.Condition,
				Body:      []Command{},
			})
			stack = append(stack, stackFrame{
				cmd:    ptr,
				branch: &ptr.Branches[len(ptr.Branches)-1],
				indent: indent,
			})
		} else if ptr.Type == CmdRepeat || ptr.Type == CmdForEach || ptr.Type == CmdWhile {
			stack = append(stack, stackFrame{
				cmd:    ptr,
				branch: nil,
				indent: indent,
			})
		}
	}
	return nil
}

// parseDirective handles @key: value header lines.
func parseDirective(hunt *Hunt, line string) error {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return nil
	}
	key := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(parts[0]), "@"))
	value := strings.TrimSpace(parts[1])

	switch key {
	case "title":
		hunt.Title = value
	case "context":
		hunt.Context = value
	case "tags", "tag":
		if value != "" {
			for _, t := range strings.Split(value, ",") {
				hunt.Tags = append(hunt.Tags, strings.TrimSpace(t))
			}
		}
	case "data", "data_file", "datafile":
		hunt.DataFile = strings.Trim(value, "'\"")
	case "schedule":
		hunt.Schedule = value
	case "export", "exports":
		for _, e := range strings.Split(value, ",") {
			hunt.Exports = append(hunt.Exports, strings.TrimSpace(e))
		}
	case "var":
		// @var: {name} = value
		eqIdx := strings.Index(value, "=")
		if eqIdx < 0 {
			return nil
		}
		varName := strings.Trim(strings.TrimSpace(value[:eqIdx]), "{}")
		varValue := strings.TrimSpace(value[eqIdx+1:])
		hunt.Vars[varName] = varValue
	case "script":
		// @script: {alias} = path
		eqIdx := strings.Index(value, "=")
		if eqIdx < 0 {
			return nil
		}
		alias := strings.Trim(strings.TrimSpace(value[:eqIdx]), "{}")
		path := strings.TrimSpace(value[eqIdx+1:])
		if hunt.ScriptAliases == nil {
			hunt.ScriptAliases = make(map[string]string)
		}
		hunt.ScriptAliases[alias] = path
	case "import":
		imp, err := parseImportLine(value)
		if err != nil {
			return err
		}
		hunt.Imports = append(hunt.Imports, imp)
	}
	return nil
}

// parseImportLine parses "@import: Login, Register from 'auth.hunt'" variants.
func parseImportLine(value string) (ImportSpec, error) {
	imp := ImportSpec{Aliases: map[string]string{}}
	// Pattern: <names> from '<file>'
	fromIdx := strings.LastIndex(strings.ToLower(value), " from ")
	if fromIdx < 0 {
		// Bare: @import: 'file.hunt'
		imp.Source = strings.Trim(strings.TrimSpace(value), "'\"")
		imp.Names = []string{"*"}
		return imp, nil
	}
	namesPart := strings.TrimSpace(value[:fromIdx])
	filePart := strings.Trim(strings.TrimSpace(value[fromIdx+6:]), "'\"")
	imp.Source = filePart

	for _, n := range strings.Split(namesPart, ",") {
		n = strings.TrimSpace(n)
		// Check for "OldName as NewName" alias syntax.
		asIdx := strings.Index(strings.ToLower(n), " as ")
		if asIdx >= 0 {
			orig := strings.TrimSpace(n[:asIdx])
			alias := strings.TrimSpace(n[asIdx+4:])
			imp.Names = append(imp.Names, orig)
			imp.Aliases[orig] = alias
		} else {
			imp.Names = append(imp.Names, n)
		}
	}
	return imp, nil
}

// applyVars replaces {var} references with their current values.
func applyVars(hunt *Hunt, s string) string {
	for name, val := range hunt.Vars {
		s = strings.ReplaceAll(s, "{"+name+"}", val)
	}
	return s
}

// parseCommand parses a single DSL line at the given line number.
// This is the exported form used by tests.
func parseCommand(line string, _ int) (Command, error) {
	return parseCommandLine(line), nil
}

// parseCommandLine is the internal parser.
func parseCommandLine(line string) Command {
	upper := strings.ToUpper(line)
	fields := strings.Fields(strings.ToLower(line))
	firstWord := ""
	if len(fields) > 0 {
		firstWord = fields[0]
	}
	cmd := Command{Verb: firstWord}

	switch {
	// ── NAVIGATE ──────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "NAVIGATE TO ") || strings.HasPrefix(upper, "NAVIGATE "):
		cmd.Type = CmdNavigate
		raw := stripPrefix(line, "NAVIGATE TO ", "NAVIGATE ")
		cmd.URL = unquote(raw)

	// ── OPEN APP ──────────────────────────────────────────────────────────────
	// Desktop/Electron entry point. ManulEngine launches+attaches; in ManulEngine (Go)
	// the app window is already attached at launch (--executable-path / --cdp),
	// so OPEN APP is a readiness checkpoint on the current window.
	case upper == "OPEN APP" || strings.HasPrefix(upper, "OPEN APP "):
		cmd.Type = CmdOpenApp

	// ── DOUBLE CLICK ──────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "DOUBLE CLICK "), strings.HasPrefix(upper, "DOUBLECLICK "):
		cmd.Type = CmdDoubleClick
		rest := stripPrefix(line, "DOUBLE CLICK ", "DOUBLECLICK ")
		cmd.Target, cmd.TypeHint, cmd.InteractionMode = parseTarget(rest)
		cmd = parseQualifiers(cmd, rest)

	// ── RIGHT CLICK ───────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "RIGHT CLICK "), strings.HasPrefix(upper, "RIGHTCLICK "):
		cmd.Type = CmdRightClick
		rest := stripPrefix(line, "RIGHT CLICK ", "RIGHTCLICK ")
		cmd.Target, cmd.TypeHint, cmd.InteractionMode = parseTarget(rest)
		cmd = parseQualifiers(cmd, rest)

	// ── CLICK ─────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "CLICK "):
		cmd.Type = CmdClick
		rest := stripPrefix(line, "CLICK ")
		cmd.Target, cmd.TypeHint, cmd.InteractionMode = parseTarget(rest)
		cmd = parseQualifiers(cmd, rest)

	// ── FILL ──────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "FILL ") || strings.HasPrefix(upper, "FILL THE "):
		cmd.Type = CmdFill
		cmd.InteractionMode = ModeInput
		rest := stripPrefix(line, "FILL THE ", "FILL ")
		// "FILL 'Target' field with 'Value'"
		firstQ := extractFirstQuoted(rest)
		cmd.Target = firstQ
		withIdx := strings.Index(strings.ToUpper(rest), " WITH ")
		if withIdx >= 0 {
			cmd.Value = unquote(strings.TrimSpace(rest[withIdx+6:]))
		}

	// ── TYPE ──────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "TYPE "):
		cmd.Type = CmdType
		cmd.InteractionMode = ModeInput
		rest := stripPrefix(line, "TYPE ")
		// "TYPE 'value' into the 'target' field"
		firstQ := extractFirstQuoted(rest)
		cmd.Value = firstQ
		intoIdx := strings.Index(strings.ToUpper(rest), " INTO ")
		if intoIdx >= 0 {
			cmd.Target, _, _ = parseTarget(strings.TrimSpace(rest[intoIdx+6:]))
		}

	// ── SELECT ────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "SELECT "):
		cmd.Type = CmdSelect
		cmd.InteractionMode = ModeSelect
		rest := stripPrefix(line, "SELECT ")
		// "SELECT 'value' from the 'target' dropdown"
		firstQ := extractFirstQuoted(rest)
		cmd.Value = firstQ
		fromIdx := strings.Index(strings.ToUpper(rest), " FROM ")
		if fromIdx >= 0 {
			cmd.Target, cmd.TypeHint, _ = parseTarget(strings.TrimSpace(rest[fromIdx+6:]))
		}

	// ── CHECK ─────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "CHECK "):
		cmd.Type = CmdCheck
		cmd.InteractionMode = ModeCheckbox
		rest := stripPrefix(line, "CHECK ")
		// "CHECK the checkbox for 'target'"
		forIdx := strings.Index(strings.ToUpper(rest), " FOR ")
		if forIdx >= 0 {
			cmd.Target = unquote(strings.TrimSpace(rest[forIdx+5:]))
		} else {
			cmd.Target, cmd.TypeHint, _ = parseTarget(rest)
		}

	// ── UNCHECK ───────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "UNCHECK "):
		cmd.Type = CmdUncheck
		cmd.InteractionMode = ModeCheckbox
		rest := stripPrefix(line, "UNCHECK ")
		forIdx := strings.Index(strings.ToUpper(rest), " FOR ")
		if forIdx >= 0 {
			cmd.Target = unquote(strings.TrimSpace(rest[forIdx+5:]))
		} else {
			cmd.Target, cmd.TypeHint, _ = parseTarget(rest)
		}

	// ── VERIFY FIELD (has text/value/placeholder) ─────────────────────────────
	// Must come BEFORE the general VERIFY cases.
	case strings.HasPrefix(upper, "VERIFY ") &&
		(strings.Contains(upper, " HAS TEXT ") ||
			strings.Contains(upper, " HAS VALUE ") ||
			strings.Contains(upper, " HAS PLACEHOLDER ")):
		cmd.Type = CmdVerifyField
		rest := stripPrefix(line, "VERIFY ")
		// Extract target (first quoted string).
		cmd.Target = extractFirstQuoted(rest)
		// Detect kind and expected value.
		restUp := strings.ToUpper(rest)
		for _, kind := range []string{"PLACEHOLDER", "VALUE", "TEXT"} {
			hasKind := " HAS " + kind + " "
			hasIdx := strings.Index(restUp, hasKind)
			if hasIdx >= 0 {
				cmd.VerifyFieldKind = strings.ToLower(kind)
				after := strings.TrimSpace(rest[hasIdx+len(hasKind):])
				cmd.Value = unquote(after)
				break
			}
		}

	// ── VERIFY SOFTLY ─────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "VERIFY SOFTLY "):
		cmd.Type = CmdVerifySoft
		rest := stripPrefix(line, "VERIFY SOFTLY ")
		cmd.VerifyText = extractVerifyText(rest, &cmd.VerifyNegated, &cmd.VerifyState)

	// ── VERIFY ────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "VERIFY "):
		rest := stripPrefix(line, "VERIFY ")
		cmd.VerifyText = extractVerifyText(rest, &cmd.VerifyNegated, &cmd.VerifyState)
		if cmd.VerifyState != "" {
			cmd.Type = CmdVerifyField
		} else {
			cmd.Type = CmdVerify
		}

	// ── EXTRACT ───────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "EXTRACT "):
		cmd.Type = CmdExtract
		cmd.InteractionMode = ModeNone
		rest := stripPrefix(line, "EXTRACT ")
		// "EXTRACT the 'Target' into {var}"
		rest = stripPrefix(rest, "THE ", "the ")
		intoIdx := strings.Index(strings.ToUpper(rest), " INTO ")
		if intoIdx >= 0 {
			cmd.Target = unquote(strings.TrimSpace(rest[:intoIdx]))
			varStr := strings.TrimSpace(rest[intoIdx+6:])
			cmd.ExtractVar = strings.Trim(varStr, "{}")
		} else {
			cmd.Target = unquote(rest)
		}

	// ── SCROLL ────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "SCROLL "):
		cmd.Type = CmdScroll
		rest := stripPrefix(line, "SCROLL ")
		upR := strings.ToUpper(rest)
		if strings.HasPrefix(upR, "UP") {
			cmd.ScrollDirection = "up"
			rest = strings.TrimSpace(rest[2:])
		} else {
			cmd.ScrollDirection = "down"
			rest = strings.TrimSpace(strings.TrimPrefix(rest, strings.Fields(rest)[0]))
		}
		insideIdx := strings.Index(" "+strings.ToUpper(rest), " INSIDE ")
		if insideIdx >= 0 {
			rawContainer := strings.TrimSpace(rest[insideIdx+7:])
			container := extractFirstQuoted(rawContainer)
			if container == "" {
				container = unquote(stripPrefix(rawContainer, "THE ", "the ", "the"))
			}
			cmd.ScrollContainer = container
		}

	// ── PRESS ─────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "PRESS "):
		cmd.Type = CmdPress
		rest := strings.TrimSpace(line[6:])
		// Handle: PRESS Key ON 'Target'
		onIdx := strings.Index(strings.ToUpper(rest), " ON ")
		if onIdx >= 0 {
			cmd.PressKey = strings.TrimSpace(rest[:onIdx])
			cmd.PressTarget, _, _ = parseTarget(strings.TrimSpace(rest[onIdx+4:]))
			if cmd.PressTarget == "" {
				cmd.PressTarget = unquote(strings.TrimSpace(rest[onIdx+4:]))
			}
		} else {
			cmd.PressKey = rest
		}

	// ── WAIT FOR RESPONSE ─────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "WAIT FOR RESPONSE "):
		cmd.Type = CmdWaitForResponse
		cmd.WaitResponseURL = unquote(strings.TrimSpace(line[18:]))

	// ── WAIT FOR ──────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "WAIT FOR ") || strings.HasPrefix(upper, "WAIT UNTIL "):
		cmd.Type = CmdWaitFor
		rest := stripPrefix(line, "WAIT FOR ", "WAIT UNTIL ")
		// "WAIT FOR 'Element' to be visible/hidden/enabled/disappear…"
		firstQ := extractFirstQuoted(rest)
		cmd.Target = firstQ
		toBeIdx := strings.Index(strings.ToUpper(rest), " TO BE ")
		if toBeIdx >= 0 {
			cmd.WaitForState = strings.ToLower(strings.TrimSpace(rest[toBeIdx+7:]))
		} else if toDisappearIdx := strings.Index(strings.ToUpper(rest), " TO DISAPPEAR"); toDisappearIdx >= 0 {
			cmd.WaitForState = "disappear"
		}

	// ── WAIT N ────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "WAIT "):
		cmd.Type = CmdWait
		rest := strings.TrimSpace(line[5:])
		// Strip optional "SECONDS" suffix.
		rest = strings.TrimSuffix(strings.TrimSuffix(strings.ToUpper(rest), " SECONDS"), " SECOND")
		rest = strings.ToLower(rest)
		if waitFields := strings.Fields(rest); len(waitFields) > 0 {
			if n, err := strconv.ParseFloat(waitFields[0], 64); err == nil {
				cmd.WaitSeconds = n
			}
		}

	// ── HOVER ─────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "HOVER "):
		cmd.Type = CmdHover
		rest := stripPrefix(line, "HOVER OVER THE ", "HOVER OVER ", "HOVER THE ", "HOVER ")
		cmd.Target, cmd.TypeHint, cmd.InteractionMode = parseTarget(rest)

	// ── DRAG ──────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "DRAG "):
		cmd.Type = CmdDrag
		rest := stripPrefix(line, "DRAG THE ELEMENT ", "DRAG THE ", "DRAG ")
		// "DRAG '<source>' and drop it into '<target>'"
		andIdx := strings.Index(strings.ToUpper(rest), " AND ")
		if andIdx >= 0 {
			cmd.DragSource = unquote(strings.TrimSpace(rest[:andIdx]))
			dropPart := strings.ToUpper(rest[andIdx:])
			intoIdx := strings.Index(dropPart, " INTO ")
			if intoIdx >= 0 {
				cmd.DragTarget = unquote(strings.TrimSpace(rest[andIdx+intoIdx+6:]))
			}
		}

	// ── SET ───────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "SET "):
		cmd.Type = CmdSet
		rest := strings.TrimSpace(line[4:])
		eqIdx := strings.Index(rest, "=")
		if eqIdx >= 0 {
			cmd.SetVar = strings.Trim(strings.TrimSpace(rest[:eqIdx]), "{}")
			cmd.SetValue = unquote(strings.TrimSpace(rest[eqIdx+1:]))
		}

	// ── PRINT ─────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "PRINT "):
		cmd.Type = CmdPrint
		cmd.PrintText = unquote(strings.TrimSpace(line[6:]))

	// ── SCREENSHOT ────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "SCREENSHOT"):
		cmd.Type = CmdScreenshot
		cmd.ScreenshotName = unquote(strings.TrimSpace(line[len("SCREENSHOT"):]))

	// ── HIGHLIGHT ─────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "HIGHLIGHT "):
		cmd.Type = CmdHighlight
		rest := stripPrefix(line, "HIGHLIGHT THE ", "HIGHLIGHT ")
		cmd.Target, cmd.TypeHint, cmd.InteractionMode = parseTarget(rest)

	// ── UPLOAD FILE ───────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "UPLOAD ") || strings.HasPrefix(upper, "UPLOAD_FILE "):
		cmd.Type = CmdUploadFile
		rest := stripPrefix(line, "UPLOAD FILE ", "UPLOAD_FILE ", "UPLOAD ")
		toIdx := strings.Index(strings.ToUpper(rest), " TO ")
		if toIdx >= 0 {
			cmd.UploadFilePath = unquote(strings.TrimSpace(rest[:toIdx]))
			cmd.UploadFile = cmd.UploadFilePath
			cmd.Target, cmd.TypeHint, cmd.InteractionMode = parseTarget(strings.TrimSpace(rest[toIdx+4:]))
		} else {
			cmd.UploadFilePath = unquote(rest)
			cmd.UploadFile = cmd.UploadFilePath
		}

	// ── FOR EACH ──────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "FOR EACH "):
		cmd.Type = CmdForEach
		rest := stripPrefix(line, "FOR EACH ")
		inIdx := strings.Index(strings.ToUpper(rest), " IN ")
		if inIdx >= 0 {
			cmd.ForEachVar = strings.Trim(strings.TrimSpace(rest[:inIdx]), "{}")
			cmd.ForEachCollection = strings.Trim(strings.TrimSpace(strings.TrimSuffix(rest[inIdx+4:], ":")), "{}")
		}

	// ── WHILE ─────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "WHILE "):
		cmd.Type = CmdWhile
		rest := stripPrefix(line, "WHILE ")
		cmd.WhileCondition = strings.TrimSuffix(strings.TrimSpace(rest), ":")
		cmd.Condition = cmd.WhileCondition

	// ── REPEAT ────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "REPEAT "):
		cmd.Type = CmdRepeat
		rest := stripPrefix(line, "REPEAT ")
		cmd.RepeatVar = "i" // default
		// "REPEAT N TIMES:" or "REPEAT N TIME:"
		fields := strings.Fields(rest)
		if len(fields) >= 1 {
			if n, err := strconv.Atoi(fields[0]); err == nil {
				cmd.RepeatCount = n
			}
		}

	// ── IF ────────────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "IF "):
		cmd.Type = CmdIf
		rest := stripPrefix(line, "IF ")
		cmd.IfCondition = strings.TrimSuffix(strings.TrimSpace(rest), ":")
		cmd.Condition = cmd.IfCondition

	// ── ELIF / ELSE IF ────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "ELIF ") || strings.HasPrefix(upper, "ELSE IF "):
		cmd.Type = CmdElIf
		rest := stripPrefix(line, "ELSE IF ", "ELIF ")
		cmd.IfCondition = strings.TrimSuffix(strings.TrimSpace(rest), ":")
		cmd.Condition = cmd.IfCondition

	// ── ELSE ──────────────────────────────────────────────────────────────────
	case upper == "ELSE:" || upper == "ELSE":
		cmd.Type = CmdElse

	// ── END markers ───────────────────────────────────────────────────────────
	case upper == "END IF" || upper == "END IF:" || upper == "ENDIF":
		cmd.Type = CmdEndIf
	case upper == "END FOR" || upper == "END FOR:" || upper == "ENDFOR" || upper == "END EACH":
		cmd.Type = CmdEndFor
	case upper == "END WHILE" || upper == "END WHILE:" || upper == "ENDWHILE":
		cmd.Type = CmdEndWhile
	case upper == "END REPEAT" || upper == "END REPEAT:" || upper == "ENDREPEAT":
		cmd.Type = CmdEndRepeat

	// ── CALL GO ─────────────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "CALL GO "):
		cmd.Type = CmdCallGo
		rest := strings.TrimSpace(stripPrefix(line, "CALL GO "))
		cmd.GoCallName, cmd.GoCallArgs, cmd.GoCallResultVar = parseCallGo(rest)

	// ── USE / CALL STEP ──────────────────────────────────────────────────────
	case strings.HasPrefix(upper, "USE ") || strings.HasPrefix(upper, "CALL ") || strings.HasPrefix(upper, "RUN STEP "):
		if strings.HasPrefix(upper, "USE ") {
			cmd.Type = CmdUse
			cmd.CallStepName = stripPrefix(line, "USE ")
		} else {
			cmd.Type = CmdCallStep
			cmd.CallStepName = stripPrefix(line, "RUN STEP ", "CALL ")
		}
		cmd.CallStepName = strings.TrimSpace(cmd.CallStepName)
 
	// ── DEBUGGING ─────────────────────────────────────────────────────────────
	case upper == "PAUSE":
		cmd.Type = CmdPause
	case upper == "DEBUG VARS":
		cmd.Type = CmdDebugVars

	// ── MOCK GET/POST/PUT/PATCH/DELETE ────────────────────────────────────────
	case strings.HasPrefix(upper, "MOCK "):
		cmd.Type = CmdMock
		rest := strings.TrimSpace(stripPrefix(line, "MOCK "))
		cmd.MockMethod, cmd.MockPattern, cmd.MockFile = parseMock(rest)

	default:
		cmd.Type = CmdUnknown
	}

	return cmd
}

func parseCallGo(rest string) (name string, args []string, resultVar string) {
	tokens := splitShellTokens(rest)
	if len(tokens) == 0 {
		return "", nil, ""
	}

	name = tokens[0]
	args = append([]string(nil), tokens[1:]...)

	if len(args) >= 2 && strings.EqualFold(args[0], "with") {
		argsKeyword := strings.TrimSuffix(strings.ToLower(args[1]), ":")
		if argsKeyword == "args" {
			args = args[2:]
		}
	}

	if len(args) >= 2 {
		assignKeyword := strings.ToLower(args[len(args)-2])
		if (assignKeyword == "into" || assignKeyword == "to") && isVariableToken(args[len(args)-1]) {
			resultVar = strings.Trim(args[len(args)-1], "{}")
			args = args[:len(args)-2]
		}
	}

	return name, args, resultVar
}

// parseMock extracts method, URL pattern, and response file from a MOCK command.
// Syntax: MOCK GET "/api/users" with 'mocks/users.json'
func parseMock(s string) (method, pattern, file string) {
	fields := strings.Fields(s)
	if len(fields) < 1 {
		return "", "", ""
	}
	method = strings.ToUpper(fields[0])
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		// ok
	default:
		return "", "", ""
	}
	// Remainder after method: "/api/users" with 'mocks/users.json'
	rest := strings.TrimSpace(s[len(fields[0]):])
	pattern = extractFirstQuoted(rest)
	if pattern == "" {
		return method, "", ""
	}
	// Find "with" keyword
	withIdx := strings.Index(strings.ToUpper(rest), " WITH ")
	if withIdx >= 0 {
		file = extractFirstQuoted(rest[withIdx+6:])
	}
	return method, pattern, file
}

// ── Target parsing helpers ─────────────────────────────────────────────────────

// parseTarget extracts the quoted label, type-hint, and interaction mode
// from a command tail like: "the 'Submit' button", "'Login' link", "on 'Save'".
func parseTarget(s string) (target, typeHint string, mode InteractionMode) {
	mode = ModeClickable
	upper := strings.ToUpper(s)

	// Strip leading "THE ", "ON THE ", "ON ", "FOR THE ", "FOR ".
	s = stripPrefix(s, "THE ", "ON THE ", "ON ", "FOR THE ", "FOR ", "OVER THE ", "OVER ")

	// Extract first quoted string.
	target = extractFirstQuoted(s)
	if target == "" {
		// Unquoted: take text up to a keyword.
		fields := strings.Fields(s)
		if len(fields) > 0 {
			target = fields[0]
		}
	}

	// Detect type-hint keyword.
	_ = upper
	hintsClickable := []string{"button", "link", "tab", "menuitem", "option", "element", "icon", "image"}
	hintsInput := []string{"field", "input", "textbox", "textarea", "text field", "search", "email", "password"}
	hintsSelect := []string{"dropdown", "select", "combobox", "listbox"}
	hintsCheckbox := []string{"checkbox", "radio", "toggle", "switch"}

	sLower := strings.ToLower(s)
	// matchHint returns true when h appears as a whole word in sLower —
	// at the start, after a space, or at the very end.
	matchHint := func(h string) bool {
		return strings.HasPrefix(sLower, h+" ") ||
			strings.Contains(sLower, " "+h+" ") ||
			strings.HasSuffix(sLower, " "+h) ||
			sLower == h
	}
	// Checkbox/radio hints are checked first: phrases like "radio button" or
	// "toggle switch" must resolve to the stateful control, not the generic
	// "button"/"switch" clickable. CLICK commands still override mode to
	// ModeClickable at dispatch time — only the typeHint differs.
	for _, h := range hintsCheckbox {
		if matchHint(h) {
			typeHint = h
			mode = ModeCheckbox
			return
		}
	}
	for _, h := range hintsClickable {
		if matchHint(h) {
			typeHint = h
			mode = ModeClickable
			return
		}
	}
	for _, h := range hintsInput {
		if matchHint(h) {
			typeHint = h
			mode = ModeInput
			return
		}
	}
	for _, h := range hintsSelect {
		if matchHint(h) {
			typeHint = h
			mode = ModeSelect
			return
		}
	}
	return
}

// parseQualifiers extracts NEAR / ON <region> / INSIDE qualifiers from a rest string.
func parseQualifiers(cmd Command, rest string) Command {
	upper := strings.ToUpper(rest)
	if nearIdx := strings.Index(upper, " NEAR "); nearIdx >= 0 {
		cmd.NearAnchor = unquote(strings.TrimSpace(rest[nearIdx+6:]))
	}
	if onIdx := strings.Index(upper, " ON "); onIdx >= 0 {
		after := strings.TrimSpace(rest[onIdx+4:])
		if !strings.HasPrefix(strings.ToUpper(after), "THE ") {
			cmd.OnRegion = strings.ToLower(unquote(after))
		}
	}
	if insideIdx := strings.Index(upper, " INSIDE "); insideIdx >= 0 {
		after := strings.TrimSpace(rest[insideIdx+8:])
		after = stripPrefix(after, "THE ")
		withIdx := strings.Index(strings.ToUpper(after), " ROW WITH ")
		if withIdx >= 0 {
			cmd.InsideContainer = unquote(strings.TrimSpace(after[:withIdx]))
			cmd.InsideRowText = unquote(strings.TrimSpace(after[withIdx+10:]))
		} else {
			withIdx = strings.Index(strings.ToUpper(after), " WITH ")
			if withIdx >= 0 {
				cmd.InsideContainer = unquote(strings.TrimSpace(after[:withIdx]))
				cmd.InsideRowText = unquote(strings.TrimSpace(after[withIdx+6:]))
			} else {
				cmd.InsideContainer = unquote(after)
			}
		}
	}
	return cmd
}

// extractVerifyText parses "that 'X' is [NOT] present/checked/visible/…".
func extractVerifyText(rest string, negated *bool, state *string) string {
	rest = stripPrefix(rest, "THAT ", "that ")
	q := extractFirstQuoted(rest)
	upper := strings.ToUpper(rest)
	if strings.Contains(upper, " IS NOT ") || strings.Contains(upper, " ARE NOT ") {
		*negated = true
	}
	for _, st := range []string{"checked", "unchecked", "visible", "hidden", "enabled", "disabled", "selected", "disappear"} {
		stateToken := strings.ToUpper(st)
		if strings.Contains(upper, " IS "+stateToken) ||
			strings.Contains(upper, " IS NOT "+stateToken) ||
			strings.Contains(upper, " ARE "+stateToken) ||
			strings.Contains(upper, " ARE NOT "+stateToken) ||
			strings.Contains(upper, " TO "+strings.ToUpper(st)) {
			*state = st
			break
		}
	}
	return q
}

// rewriteScriptAlias rewrites CALL GO commands that use a @script: alias
// to their full dotted path. It handles both {alias} and {alias}.func forms.
func rewriteScriptAlias(cmd *Command, aliases map[string]string) {
	if cmd.Type != CmdCallGo || len(aliases) == 0 {
		return
	}
	name := cmd.GoCallName
	if !strings.HasPrefix(name, "{") {
		return
	}
	closeIdx := strings.Index(name, "}")
	if closeIdx < 0 {
		return
	}
	alias := name[1:closeIdx]
	rest := name[closeIdx+1:]
	if path, ok := aliases[alias]; ok {
		cmd.GoCallName = path + rest
	}
}

// ── String utilities ───────────────────────────────────────────────────────────

// stripPrefix removes the first matching prefix (case-insensitive) from s.
func stripPrefix(s string, prefixes ...string) string {
	upper := strings.ToUpper(s)
	for _, p := range prefixes {
		if strings.HasPrefix(upper, strings.ToUpper(p)) {
			return s[len(p):]
		}
	}
	return s
}

// unquote removes surrounding single or double quotes from s.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') ||
			(s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// extractFirstQuoted returns the content of the first single- or double-quoted
// substring in s, or empty string if none.
func extractFirstQuoted(s string) string {
	startSingle := strings.IndexByte(s, '\'')
	startDouble := strings.IndexByte(s, '"')
	var start int
	var q byte
	if startSingle < 0 && startDouble < 0 {
		return ""
	} else if startSingle < 0 {
		start = startDouble
		q = '"'
	} else if startDouble < 0 {
		start = startSingle
		q = '\''
	} else if startSingle < startDouble {
		start = startSingle
		q = '\''
	} else {
		start = startDouble
		q = '"'
	}
	end := strings.IndexByte(s[start+1:], q)
	if end < 0 {
		return ""
	}
	return s[start+1 : start+1+end]
}

func splitShellTokens(s string) []string {
	var tokens []string
	var current strings.Builder
	var quote rune

	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
	}

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			current.WriteRune(r)
		}
	}
	flush()
	return tokens
}

func isVariableToken(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) >= 3 && strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
}
