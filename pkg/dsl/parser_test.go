package dsl

import (
	"strings"
	"testing"
)

// ── Parse helpers ─────────────────────────────────────────────────────────────

func mustParse(t *testing.T, src string) *Hunt {
	t.Helper()
	hunt, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	return hunt
}

func mustParseLine(t *testing.T, line string) Command {
	t.Helper()
	hunt := mustParse(t, line)
	if len(hunt.Commands) == 0 {
		t.Fatalf("no commands parsed from %q", line)
	}
	return hunt.Commands[0]
}

// ── Headers ───────────────────────────────────────────────────────────────────

func TestParseHeaders(t *testing.T) {
	src := `@context: testing context
@title: my_suite
@var: {base} = https://example.com
`
	hunt := mustParse(t, src)
	if hunt.Context != "testing context" {
		t.Errorf("context = %q", hunt.Context)
	}
	if hunt.Title != "my_suite" {
		t.Errorf("title = %q", hunt.Title)
	}
	if hunt.Vars["base"] != "https://example.com" {
		t.Errorf("var base = %q", hunt.Vars["base"])
	}
}

func TestParseTags(t *testing.T) {
	src := `@tags: smoke, regression, ui`
	hunt := mustParse(t, src)
	if len(hunt.Tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(hunt.Tags))
	}
	if hunt.Tags[0] != "smoke" || hunt.Tags[1] != "regression" || hunt.Tags[2] != "ui" {
		t.Errorf("tags = %v", hunt.Tags)
	}
}

// ── NAVIGATE ──────────────────────────────────────────────────────────────────

func TestParseNavigate(t *testing.T) {
	cmd := mustParseLine(t, "NAVIGATE to https://example.com/page")
	if cmd.Type != CmdNavigate {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.URL != "https://example.com/page" {
		t.Errorf("url = %q", cmd.URL)
	}
}

func TestParseNavigateQuoted(t *testing.T) {
	cmd := mustParseLine(t, "NAVIGATE to 'https://example.com/page'")
	if cmd.URL != "https://example.com/page" {
		t.Errorf("url = %q", cmd.URL)
	}
}

// ── CLICK ─────────────────────────────────────────────────────────────────────

func TestParseClickButton(t *testing.T) {
	cmd := mustParseLine(t, "Click the 'Submit' button")
	if cmd.Type != CmdClick {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "Submit" {
		t.Errorf("target = %q", cmd.Target)
	}
	if cmd.TypeHint != "button" {
		t.Errorf("hint = %q", cmd.TypeHint)
	}
	if cmd.InteractionMode != ModeClickable {
		t.Errorf("mode = %s", cmd.InteractionMode)
	}
}

func TestParseClickLink(t *testing.T) {
	cmd := mustParseLine(t, "CLICK on the '2' link")
	if cmd.Type != CmdClick {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "2" {
		t.Errorf("target = %q", cmd.Target)
	}
	if cmd.TypeHint != "link" {
		t.Errorf("hint = %q, want 'link'", cmd.TypeHint)
	}
}

func TestParseClickElement(t *testing.T) {
	cmd := mustParseLine(t, "CLICK the 'Scrolling DropDown' element")
	if cmd.Target != "Scrolling DropDown" {
		t.Errorf("target = %q", cmd.Target)
	}
	if cmd.TypeHint != "element" {
		t.Errorf("hint = %q", cmd.TypeHint)
	}
}

// Compound hints like "radio button", "toggle switch" must resolve to the
// more-specific stateful control, not the generic "button"/"switch" clickable.
// Regression for the rahulshettyacademy radio-label bug where the parser
// incorrectly returned typeHint="button" for "radio button", stripping the
// radio-specific semantic boost and letting the wrapping <label> win.
func TestParseClickRadioButtonCompoundHint(t *testing.T) {
	cases := []struct {
		line     string
		hint     string
		mode     InteractionMode
		target   string
	}{
		{"CLICK the radio button for 'Radio1'", "radio", ModeCheckbox, "Radio1"},
		{"CLICK the 'Radio1' radio button", "radio", ModeCheckbox, "Radio1"},
		{"CLICK the checkbox for 'Agree'", "checkbox", ModeCheckbox, "Agree"},
		{"CLICK the toggle switch for 'Dark mode'", "toggle", ModeCheckbox, "Dark mode"},
	}
	for _, tc := range cases {
		cmd := mustParseLine(t, tc.line)
		if cmd.Target != tc.target {
			t.Errorf("%q: target = %q, want %q", tc.line, cmd.Target, tc.target)
		}
		if cmd.TypeHint != tc.hint {
			t.Errorf("%q: hint = %q, want %q", tc.line, cmd.TypeHint, tc.hint)
		}
		if cmd.InteractionMode != tc.mode {
			t.Errorf("%q: mode = %s, want %s", tc.line, cmd.InteractionMode, tc.mode)
		}
	}
}

// ── DOUBLE CLICK ──────────────────────────────────────────────────────────────

func TestParseDoubleClick(t *testing.T) {
	cmd := mustParseLine(t, "DOUBLE CLICK the 'Copy Text' button")
	if cmd.Type != CmdDoubleClick {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "Copy Text" {
		t.Errorf("target = %q", cmd.Target)
	}
	if cmd.TypeHint != "button" {
		t.Errorf("hint = %q", cmd.TypeHint)
	}
}

// ── FILL / TYPE ───────────────────────────────────────────────────────────────

func TestParseFill(t *testing.T) {
	cmd := mustParseLine(t, "FILL 'Name' field with 'Mega Manul'")
	if cmd.Type != CmdFill {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "Name" {
		t.Errorf("target = %q", cmd.Target)
	}
	if cmd.Value != "Mega Manul" {
		t.Errorf("value = %q", cmd.Value)
	}
	if cmd.InteractionMode != ModeInput {
		t.Errorf("mode = %s", cmd.InteractionMode)
	}
}

func TestParseType(t *testing.T) {
	cmd := mustParseLine(t, "TYPE 'Shadow Boss' into the 'Shadow Root' field")
	if cmd.Type != CmdType {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Value != "Shadow Boss" {
		t.Errorf("value = %q", cmd.Value)
	}
	if cmd.Target != "Shadow Root" {
		t.Errorf("target = %q", cmd.Target)
	}
}

// ── SELECT ────────────────────────────────────────────────────────────────────

func TestParseSelect(t *testing.T) {
	cmd := mustParseLine(t, "SELECT 'Japan' from the 'Country' dropdown")
	if cmd.Type != CmdSelect {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Value != "Japan" {
		t.Errorf("value = %q", cmd.Value)
	}
	if cmd.Target != "Country" {
		t.Errorf("target = %q", cmd.Target)
	}
	if cmd.InteractionMode != ModeSelect {
		t.Errorf("mode = %s", cmd.InteractionMode)
	}
}

// ── CHECK / UNCHECK ───────────────────────────────────────────────────────────

func TestParseCheck(t *testing.T) {
	cmd := mustParseLine(t, "CHECK the checkbox for 'Monday'")
	if cmd.Type != CmdCheck {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "Monday" {
		t.Errorf("target = %q", cmd.Target)
	}
	if cmd.InteractionMode != ModeCheckbox {
		t.Errorf("mode = %s", cmd.InteractionMode)
	}
}

func TestParseCheckNumericTarget(t *testing.T) {
	cmd := mustParseLine(t, "CHECK the checkbox for '7'")
	if cmd.Type != CmdCheck {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "7" {
		t.Errorf("target = %q", cmd.Target)
	}
	if cmd.InteractionMode != ModeCheckbox {
		t.Errorf("mode = %s", cmd.InteractionMode)
	}
}

func TestParseUncheck(t *testing.T) {
	cmd := mustParseLine(t, "Uncheck the checkbox for 'Auto-Renew'")
	if cmd.Type != CmdUncheck {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "Auto-Renew" {
		t.Errorf("target = %q", cmd.Target)
	}
}

func TestParseCallGoWithArgsAndInto(t *testing.T) {
	cmd := mustParseLine(t, `CALL GO helpers.echo "hello world" {email} plain into {message}`)
	if cmd.Type != CmdCallGo {
		t.Fatalf("type = %s, want %s", cmd.Type, CmdCallGo)
	}
	if cmd.GoCallName != "helpers.echo" {
		t.Fatalf("GoCallName = %q, want helpers.echo", cmd.GoCallName)
	}
	wantArgs := []string{"hello world", "{email}", "plain"}
	if len(cmd.GoCallArgs) != len(wantArgs) {
		t.Fatalf("GoCallArgs len = %d, want %d (%v)", len(cmd.GoCallArgs), len(wantArgs), cmd.GoCallArgs)
	}
	for i := range wantArgs {
		if cmd.GoCallArgs[i] != wantArgs[i] {
			t.Fatalf("GoCallArgs[%d] = %q, want %q", i, cmd.GoCallArgs[i], wantArgs[i])
		}
	}
	if cmd.GoCallResultVar != "message" {
		t.Fatalf("GoCallResultVar = %q, want message", cmd.GoCallResultVar)
	}
}

func TestParseCallGoWithArgsPrefixAndToAlias(t *testing.T) {
	cmd := mustParseLine(t, `CALL GO math.concat with args: 'a' 'b c' to {joined}`)
	if cmd.Type != CmdCallGo {
		t.Fatalf("type = %s, want %s", cmd.Type, CmdCallGo)
	}
	if cmd.GoCallName != "math.concat" {
		t.Fatalf("GoCallName = %q, want math.concat", cmd.GoCallName)
	}
	wantArgs := []string{"a", "b c"}
	if len(cmd.GoCallArgs) != len(wantArgs) {
		t.Fatalf("GoCallArgs len = %d, want %d (%v)", len(cmd.GoCallArgs), len(wantArgs), cmd.GoCallArgs)
	}
	for i := range wantArgs {
		if cmd.GoCallArgs[i] != wantArgs[i] {
			t.Fatalf("GoCallArgs[%d] = %q, want %q", i, cmd.GoCallArgs[i], wantArgs[i])
		}
	}
	if cmd.GoCallResultVar != "joined" {
		t.Fatalf("GoCallResultVar = %q, want joined", cmd.GoCallResultVar)
	}
}

func TestParseCallGoNoArgs(t *testing.T) {
	cmd := mustParseLine(t, `CALL GO helpers.noop`)
	if cmd.Type != CmdCallGo {
		t.Fatalf("type = %s, want %s", cmd.Type, CmdCallGo)
	}
	if cmd.GoCallName != "helpers.noop" {
		t.Fatalf("GoCallName = %q, want helpers.noop", cmd.GoCallName)
	}
	if len(cmd.GoCallArgs) != 0 {
		t.Fatalf("GoCallArgs = %v, want none", cmd.GoCallArgs)
	}
	if cmd.GoCallResultVar != "" {
		t.Fatalf("GoCallResultVar = %q, want empty", cmd.GoCallResultVar)
	}
}

func TestParseCallGoNoArgsWithInto(t *testing.T) {
	cmd := mustParseLine(t, `CALL GO helpers.get_value into {result}`)
	if cmd.Type != CmdCallGo {
		t.Fatalf("type = %s, want %s", cmd.Type, CmdCallGo)
	}
	if cmd.GoCallName != "helpers.get_value" {
		t.Fatalf("GoCallName = %q, want helpers.get_value", cmd.GoCallName)
	}
	if len(cmd.GoCallArgs) != 0 {
		t.Fatalf("GoCallArgs = %v, want none", cmd.GoCallArgs)
	}
	if cmd.GoCallResultVar != "result" {
		t.Fatalf("GoCallResultVar = %q, want result", cmd.GoCallResultVar)
	}
}

func TestParseCallStepStillUsesExistingSemantics(t *testing.T) {
	cmd := mustParseLine(t, "CALL Login Flow")
	if cmd.Type != CmdCallStep {
		t.Fatalf("type = %s, want %s", cmd.Type, CmdCallStep)
	}
	if cmd.CallStepName != "Login Flow" {
		t.Fatalf("CallStepName = %q, want Login Flow", cmd.CallStepName)
	}
}

// ── VERIFY ────────────────────────────────────────────────────────────────────

func TestParseVerifyPresent(t *testing.T) {
	cmd := mustParseLine(t, "VERIFY that 'Mega Manul' is present")
	if cmd.Type != CmdVerify {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.VerifyText != "Mega Manul" {
		t.Errorf("verifyText = %q", cmd.VerifyText)
	}
	if cmd.VerifyNegated {
		t.Error("should not be negated")
	}
}

func TestParseVerifyNotPresent(t *testing.T) {
	cmd := mustParseLine(t, "VERIFY that 'Error' is NOT present")
	if cmd.Type != CmdVerify {
		t.Errorf("type = %s", cmd.Type)
	}
	if !cmd.VerifyNegated {
		t.Error("should be negated")
	}
}

func TestParseVerifySoftly(t *testing.T) {
	cmd := mustParseLine(t, "VERIFY SOFTLY that 'Warning' is present")
	if cmd.Type != CmdVerifySoft {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.VerifyText != "Warning" {
		t.Errorf("verifyText = %q", cmd.VerifyText)
	}
}

func TestParseVerifyChecked(t *testing.T) {
	cmd := mustParseLine(t, "VERIFY that 'Monday' is checked")
	if cmd.Type != CmdVerifyField {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.VerifyText != "Monday" {
		t.Errorf("verifyText = %q", cmd.VerifyText)
	}
	if cmd.VerifyState != "checked" {
		t.Errorf("verifyState = %q, want checked", cmd.VerifyState)
	}
	if cmd.VerifyNegated {
		t.Error("VerifyNegated = true, want false")
	}
}

func TestParseVerifyNotChecked(t *testing.T) {
	cmd := mustParseLine(t, "VERIFY that 'Accept' is NOT checked")
	if cmd.Type != CmdVerifyField {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.VerifyText != "Accept" {
		t.Errorf("verifyText = %q", cmd.VerifyText)
	}
	if cmd.VerifyState != "checked" {
		t.Errorf("verifyState = %q, want checked", cmd.VerifyState)
	}
	if !cmd.VerifyNegated {
		t.Error("VerifyNegated = false, want true")
	}
}

// ── SCROLL ────────────────────────────────────────────────────────────────────

func TestParseScrollDown(t *testing.T) {
	cmd := mustParseLine(t, "SCROLL DOWN")
	if cmd.Type != CmdScroll {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.ScrollDirection != "down" {
		t.Errorf("direction = %q", cmd.ScrollDirection)
	}
}

func TestParseScrollInsideContainer(t *testing.T) {
	cmd := mustParseLine(t, "SCROLL DOWN inside the 'list'")
	if cmd.Type != CmdScroll {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.ScrollContainer != "list" {
		t.Errorf("container = %q", cmd.ScrollContainer)
	}
}

// ── PRESS ─────────────────────────────────────────────────────────────────────

func TestParsePressEnter(t *testing.T) {
	cmd := mustParseLine(t, "PRESS ENTER")
	if cmd.Type != CmdPress {
		t.Errorf("type = %s", cmd.Type)
	}
	if !strings.EqualFold(cmd.PressKey, "ENTER") {
		t.Errorf("key = %q", cmd.PressKey)
	}
}

func TestParsePressCombo(t *testing.T) {
	cmd := mustParseLine(t, "PRESS Control+A")
	if cmd.Type != CmdPress {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.PressKey != "Control+A" {
		t.Errorf("key = %q", cmd.PressKey)
	}
}

// ── EXTRACT ───────────────────────────────────────────────────────────────────

func TestParseExtract(t *testing.T) {
	cmd := mustParseLine(t, "EXTRACT the 'CPU of Chrome' into {chrome_cpu}")
	if cmd.Type != CmdExtract {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "CPU of Chrome" {
		t.Errorf("target = %q", cmd.Target)
	}
	if cmd.ExtractVar != "chrome_cpu" {
		t.Errorf("var = %q", cmd.ExtractVar)
	}
}

// ── SET ───────────────────────────────────────────────────────────────────────

func TestParseSet(t *testing.T) {
	cmd := mustParseLine(t, "SET {greeting} = Hello World")
	if cmd.Type != CmdSet {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.SetVar != "greeting" {
		t.Errorf("var = %q", cmd.SetVar)
	}
	if cmd.SetValue != "Hello World" {
		t.Errorf("value = %q", cmd.SetValue)
	}
}

// ── WAIT ──────────────────────────────────────────────────────────────────────

func TestParseWait(t *testing.T) {
	cmd := mustParseLine(t, "WAIT 2")
	if cmd.Type != CmdWait {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.WaitSeconds != 2 {
		t.Errorf("seconds = %f", cmd.WaitSeconds)
	}
}

func TestParseWaitForVisible(t *testing.T) {
	cmd := mustParseLine(t, "Wait for 'Loading' to be visible")
	if cmd.Type != CmdWaitFor {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "Loading" {
		t.Errorf("target = %q", cmd.Target)
	}
	if cmd.WaitForState != "visible" {
		t.Errorf("state = %q", cmd.WaitForState)
	}
}

// ── HOVER ─────────────────────────────────────────────────────────────────────

func TestParseHover(t *testing.T) {
	cmd := mustParseLine(t, "HOVER over the 'Profile' element")
	if cmd.Type != CmdHover {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "Profile" {
		t.Errorf("target = %q", cmd.Target)
	}
}

// ── RIGHT CLICK ───────────────────────────────────────────────────────────────

func TestParseRightClick(t *testing.T) {
	cmd := mustParseLine(t, "RIGHT CLICK the 'File' element")
	if cmd.Type != CmdRightClick {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.Target != "File" {
		t.Errorf("target = %q", cmd.Target)
	}
}

// ── DRAG ──────────────────────────────────────────────────────────────────────

func TestParseDrag(t *testing.T) {
	cmd := mustParseLine(t, "DRAG the element 'Drag me to my target' and drop it into 'Drop here'")
	if cmd.Type != CmdDrag {
		t.Errorf("type = %s", cmd.Type)
	}
	if cmd.DragSource != "Drag me to my target" {
		t.Errorf("source = %q", cmd.DragSource)
	}
	if cmd.DragTarget != "Drop here" {
		t.Errorf("dropTarget = %q", cmd.DragTarget)
	}
}

// ── NEAR / ON / INSIDE qualifiers ─────────────────────────────────────────────

func TestParseNearQualifier(t *testing.T) {
	cmd := mustParseLine(t, "Click the 'Edit' button NEAR 'John Doe'")
	if cmd.NearAnchor != "John Doe" {
		t.Errorf("near = %q", cmd.NearAnchor)
	}
}

func TestParseOnRegion(t *testing.T) {
	cmd := mustParseLine(t, "Click the 'Logo' link ON HEADER")
	if cmd.OnRegion != "header" {
		t.Errorf("region = %q", cmd.OnRegion)
	}
}

func TestParseInsideQualifier(t *testing.T) {
	cmd := mustParseLine(t, "Click the 'Delete' button INSIDE 'Actions' row with 'John'")
	if cmd.InsideContainer != "Actions" {
		t.Errorf("inside = %q", cmd.InsideContainer)
	}
	if cmd.InsideRowText != "John" {
		t.Errorf("rowText = %q", cmd.InsideRowText)
	}
}

// ── STEP blocks and DONE ─────────────────────────────────────────────────────

func TestParseStepBlocks(t *testing.T) {
	src := `
STEP 1: First step
    NAVIGATE to https://example.com

STEP 2: Second step
    CLICK the 'Login' button
DONE.
`
	hunt := mustParse(t, src)
	if len(hunt.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(hunt.Commands))
	}
	if hunt.Commands[0].StepBlock != "STEP 1: First step" {
		t.Errorf("step[0] block = %q", hunt.Commands[0].StepBlock)
	}
	if hunt.Commands[1].StepBlock != "STEP 2: Second step" {
		t.Errorf("step[1] block = %q", hunt.Commands[1].StepBlock)
	}
}

// ── Comments and blank lines ─────────────────────────────────────────────────

func TestParseIgnoresComments(t *testing.T) {
	src := `# This is a comment
    # Indented comment
NAVIGATE to https://example.com
`
	hunt := mustParse(t, src)
	if len(hunt.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(hunt.Commands))
	}
}

// ── PRINT ─────────────────────────────────────────────────────────────────────

func TestParsePrint(t *testing.T) {
	cmd := mustParseLine(t, "PRINT 'Hello World'")
	if cmd.Type != CmdPrint {
		t.Errorf("type = %s", cmd.Type)
	}
}

func TestParseOpenApp(t *testing.T) {
	for _, line := range []string{"OPEN APP", "OPEN APP", "open app"} {
		cmd := mustParseLine(t, line)
		if cmd.Type != CmdOpenApp {
			t.Errorf("parse %q: type = %s, want OPEN_APP", line, cmd.Type)
		}
	}
}

func TestParseScreenshot(t *testing.T) {
	bare := mustParseLine(t, "SCREENSHOT")
	if bare.Type != CmdScreenshot {
		t.Errorf("type = %s, want SCREENSHOT", bare.Type)
	}
	if bare.ScreenshotName != "" {
		t.Errorf("bare ScreenshotName = %q, want empty", bare.ScreenshotName)
	}
	named := mustParseLine(t, `SCREENSHOT "after login"`)
	if named.Type != CmdScreenshot {
		t.Errorf("type = %s, want SCREENSHOT", named.Type)
	}
	if named.ScreenshotName != "after login" {
		t.Errorf("ScreenshotName = %q, want %q", named.ScreenshotName, "after login")
	}
}

// ── Variable substitution ─────────────────────────────────────────────────────

func TestParseVariableSubstitution(t *testing.T) {
	src := `@var: {url} = https://example.com
NAVIGATE to {url}
`
	hunt := mustParse(t, src)
	if len(hunt.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(hunt.Commands))
	}
	if hunt.Commands[0].URL != "https://example.com" {
		t.Errorf("url = %q, expected substituted value", hunt.Commands[0].URL)
	}
}

// ── SETUP / TEARDOWN ──────────────────────────────────────────────────────────

func TestParseSetupTeardownBlocks(t *testing.T) {
	src := `
[SETUP]
    PRINT "setup runs"
    SET {setup_var} = hello
[END SETUP]

STEP 1: Mission
    NAVIGATE to https://example.com
    CLICK the 'Login' button

[TEARDOWN]
    PRINT "teardown runs"
[END TEARDOWN]
`
	hunt := mustParse(t, src)

	if len(hunt.SetupCommands) != 2 {
		t.Fatalf("expected 2 setup commands, got %d", len(hunt.SetupCommands))
	}
	if hunt.SetupCommands[0].Type != CmdPrint {
		t.Errorf("setup[0] type = %s, want PRINT", hunt.SetupCommands[0].Type)
	}
	if hunt.SetupCommands[1].Type != CmdSet {
		t.Errorf("setup[1] type = %s, want SET", hunt.SetupCommands[1].Type)
	}

	if len(hunt.Commands) != 2 {
		t.Fatalf("expected 2 mission commands, got %d", len(hunt.Commands))
	}
	if hunt.Commands[0].Type != CmdNavigate {
		t.Errorf("mission[0] type = %s, want NAVIGATE", hunt.Commands[0].Type)
	}
	if hunt.Commands[1].Type != CmdClick {
		t.Errorf("mission[1] type = %s, want CLICK", hunt.Commands[1].Type)
	}

	if len(hunt.TeardownCommands) != 1 {
		t.Fatalf("expected 1 teardown command, got %d", len(hunt.TeardownCommands))
	}
	if hunt.TeardownCommands[0].Type != CmdPrint {
		t.Errorf("teardown[0] type = %s, want PRINT", hunt.TeardownCommands[0].Type)
	}
}

func TestParseSetupTeardownWithNestedBlocks(t *testing.T) {
	src := `
[SETUP]
    IF {x} == '1':
        PRINT "conditional setup"
[END SETUP]

STEP 1:
    NAVIGATE to https://example.com
`
	hunt := mustParse(t, src)
	if len(hunt.SetupCommands) != 1 {
		t.Fatalf("expected 1 setup command, got %d", len(hunt.SetupCommands))
	}
	ifCmd := hunt.SetupCommands[0]
	if ifCmd.Type != CmdIf {
		t.Fatalf("setup[0] type = %s, want IF", ifCmd.Type)
	}
	if len(ifCmd.Branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(ifCmd.Branches))
	}
	if len(ifCmd.Branches[0].Body) != 1 {
		t.Fatalf("expected 1 body command, got %d", len(ifCmd.Branches[0].Body))
	}
	if ifCmd.Branches[0].Body[0].Type != CmdPrint {
		t.Errorf("body[0] type = %s, want PRINT", ifCmd.Branches[0].Body[0].Type)
	}
}

// ── @script: aliases ──────────────────────────────────────────────────────────

func TestParseScriptAliasRewrite(t *testing.T) {
	src := `@script: {helpers} = mypackage.helpers
CALL GO {helpers}.echo "hello" into {result}
`
	hunt := mustParse(t, src)
	if len(hunt.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(hunt.Commands))
	}
	cmd := hunt.Commands[0]
	if cmd.Type != CmdCallGo {
		t.Fatalf("type = %s, want CALL_GO", cmd.Type)
	}
	if cmd.GoCallName != "mypackage.helpers.echo" {
		t.Errorf("GoCallName = %q, want mypackage.helpers.echo", cmd.GoCallName)
	}
}

func TestParseScriptAliasNoBraces(t *testing.T) {
	src := `@script: {helpers} = mypackage.helpers
CALL GO helpers.echo "hello"
`
	hunt := mustParse(t, src)
	if len(hunt.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(hunt.Commands))
	}
	cmd := hunt.Commands[0]
	if cmd.GoCallName != "helpers.echo" {
		t.Errorf("GoCallName = %q, want helpers.echo (no rewrite expected)", cmd.GoCallName)
	}
}

// ── Full .hunt parse ──────────────────────────────────────────────────────────

func TestParseMegaHuntStructure(t *testing.T) {
	src := `@context: Test context
@title: test_suite

STEP 1: Navigate
    NAVIGATE to https://example.com

STEP 2: Inputs
    FILL 'Name' field with 'John'
    FILL 'Email' field with 'john@test.com'
    VERIFY that 'John' is present

STEP 3: Interactions
    CLICK the 'Submit' button
    SELECT 'Option A' from the 'Dropdown' dropdown
    CHECK the checkbox for 'Terms'

STEP 4: Advanced
    DOUBLE CLICK the 'Copy' button
    DRAG the element 'Source' and drop it into 'Target'
    EXTRACT the 'Price' into {price}
    SCROLL DOWN
    SCROLL DOWN inside the 'list'

DONE.
`
	hunt := mustParse(t, src)
	if hunt.Title != "test_suite" {
		t.Errorf("title = %q", hunt.Title)
	}
	if hunt.Context != "Test context" {
		t.Errorf("context = %q", hunt.Context)
	}

	// Count by type
	counts := map[CommandType]int{}
	for _, cmd := range hunt.Commands {
		counts[cmd.Type]++
	}

	expect := map[CommandType]int{
		CmdNavigate:    1,
		CmdFill:        2,
		CmdVerify:      1,
		CmdClick:       1,
		CmdSelect:      1,
		CmdCheck:       1,
		CmdDoubleClick: 1,
		CmdDrag:        1,
		CmdExtract:     1,
		CmdScroll:      2,
	}
	for typ, want := range expect {
		if counts[typ] != want {
			t.Errorf("count(%s) = %d, want %d", typ, counts[typ], want)
		}
	}
}
