package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/alexbeatnik/ManulEngineGo/pkg/browser"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
)

type CustomControlInvocation struct {
	Page       string
	Target     string
	ActionType string
	Value      string
	Variables  map[string]string
	Command    dsl.Command
}

type CustomControlHandler func(context.Context, browser.Page, CustomControlInvocation) error

type GoCallInvocation struct {
	Name      string
	Args      []string
	Variables map[string]string
	Page      browser.Page
	Command   dsl.Command
}

type GoCallHandler func(context.Context, GoCallInvocation) (any, error)

// REGISTRY POLICY (parallel execution):
//
// customControls and goCalls are package-global maps shared by every
// Runtime in the process. They are safe for concurrent reads (sync.RWMutex)
// and for concurrent registration, but the intended lifecycle is:
//
//   1. All Register* calls happen at process init (TestMain, main(),
//      or init() functions in plugin packages).
//   2. Worker pool spawns; goroutines call only Get* / handler invocation.
//   3. Process exits.
//
// Registering or unregistering handlers WHILE workers are executing is
// permitted by the type system (the mutex makes it data-race-free) but is
// strongly discouraged: the visibility of a handler to a particular
// in-flight hunt becomes timing-dependent. ResetRuntimeRegistries() exists
// for test fixtures and MUST NOT be called while any Worker is running.
//
// Handlers themselves MUST be safe for concurrent invocation across goroutines:
// the same handler may be invoked by every worker simultaneously.
var (
	customControlsMu sync.RWMutex
	customControls   = map[string]CustomControlHandler{}
	goCallsMu        sync.RWMutex
	goCalls          = map[string]GoCallHandler{}
)

func RegisterCustomControl(page, target string, handler CustomControlHandler) error {
	key := customControlKey(page, target)
	if key == "" {
		return fmt.Errorf("custom control registration requires non-empty page and target")
	}
	if handler == nil {
		return fmt.Errorf("custom control registration requires a handler")
	}
	customControlsMu.Lock()
	defer customControlsMu.Unlock()
	customControls[key] = handler
	return nil
}

func GetCustomControl(page, target string) (CustomControlHandler, bool) {
	customControlsMu.RLock()
	defer customControlsMu.RUnlock()

	if handler, ok := customControls[customControlKey(page, target)]; ok {
		return handler, true
	}
	if handler, ok := customControls[customControlKey("*", target)]; ok {
		return handler, true
	}
	return nil, false
}

func RegisterGoCall(name string, handler GoCallHandler) error {
	key := normalizeRegistryLabel(name)
	if key == "" {
		return fmt.Errorf("CALL GO registration requires a non-empty name")
	}
	if handler == nil {
		return fmt.Errorf("CALL GO registration requires a handler")
	}
	goCallsMu.Lock()
	defer goCallsMu.Unlock()
	goCalls[key] = handler
	return nil
}

func GetGoCall(name string) (GoCallHandler, bool) {
	goCallsMu.RLock()
	defer goCallsMu.RUnlock()
	handler, ok := goCalls[normalizeRegistryLabel(name)]
	return handler, ok
}

func ResetRuntimeRegistries() {
	customControlsMu.Lock()
	customControls = map[string]CustomControlHandler{}
	customControlsMu.Unlock()

	goCallsMu.Lock()
	goCalls = map[string]GoCallHandler{}
	goCallsMu.Unlock()
}

func (rt *Runtime) tryExecuteCustomControl(ctx context.Context, cmd dsl.Command) (bool, string, map[string]any, error) {
	actionType := customControlActionType(cmd)
	if actionType == "" {
		return false, "", nil, nil
	}

	target := rt.resolveVariables(cmd.Target)
	if strings.TrimSpace(target) == "" {
		return false, "", nil, nil
	}

	pageLabel := rt.currentPageLabel(ctx)
	handler, ok := GetCustomControl(pageLabel, target)
	if !ok {
		return false, "", nil, nil
	}

	value := customControlValue(rt, cmd)
	err := handler(ctx, rt.page, CustomControlInvocation{
		Page:       pageLabel,
		Target:     target,
		ActionType: actionType,
		Value:      value,
		Variables:  rt.vars.Flatten(),
		Command:    cmd,
	})
	metadata := map[string]any{
		"resolution_strategy": "custom-control",
		"custom_control_page": pageLabel,
	}
	return true, value, metadata, err
}

func (rt *Runtime) executeCallGo(ctx context.Context, cmd dsl.Command) (string, map[string]any, error) {
	handler, ok := GetGoCall(cmd.GoCallName)
	if !ok {
		return "", nil, fmt.Errorf("CALL GO handler not registered: %s", cmd.GoCallName)
	}

	args := make([]string, len(cmd.GoCallArgs))
	for i, arg := range cmd.GoCallArgs {
		args[i] = rt.resolveVariables(arg)
	}

	result, err := handler(ctx, GoCallInvocation{
		Name:      cmd.GoCallName,
		Args:      args,
		Variables: rt.vars.Flatten(),
		Page:      rt.page,
		Command:   cmd,
	})
	if err != nil {
		return "", nil, err
	}

	value := ""
	if result != nil {
		value = fmt.Sprint(result)
	}

	// Flatten map return values into runtime variables, mirroring Python's
	// bind_hook_result behaviour for dict returns.
	switch m := result.(type) {
	case map[string]string:
		for k, v := range m {
			if strings.TrimSpace(k) != "" {
				rt.vars.Set(k, v, LevelRow)
			}
		}
	case map[string]any:
		for k, v := range m {
			if strings.TrimSpace(k) != "" {
				rt.vars.Set(k, fmt.Sprint(v), LevelRow)
			}
		}
	}

	if cmd.GoCallResultVar != "" {
		rt.vars.Set(cmd.GoCallResultVar, value, LevelRow)
	}

	metadata := map[string]any{
		"resolution_strategy": "call-go",
		"go_call_name":        cmd.GoCallName,
		"go_call_args":        len(args),
	}
	return value, metadata, nil
}

func (rt *Runtime) currentPageLabel(ctx context.Context) string {
	rawTitle, err := rt.page.EvalJS(ctx, `(() => (document.title || "").trim())()`)
	if err == nil {
		var title string
		if jsonErr := json.Unmarshal(rawTitle, &title); jsonErr == nil {
			title = strings.TrimSpace(title)
			if title != "" {
				return title
			}
		}
		trimmed := strings.Trim(strings.TrimSpace(string(rawTitle)), `"`)
		if trimmed != "" {
			return trimmed
		}
	}

	currentURL, err := rt.page.CurrentURL(ctx)
	if err != nil {
		return ""
	}
	return pageLabelFromURL(currentURL)
}

func pageLabelFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	segment := strings.Trim(parsed.Path, "/")
	if idx := strings.LastIndex(segment, "/"); idx >= 0 {
		segment = segment[idx+1:]
	}
	segment = strings.TrimSpace(strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(segment))
	label := normalizeRegistryLabel(segment)
	if label == "" {
		label = normalizeRegistryLabel(strings.TrimPrefix(parsed.Hostname(), "www."))
	}
	if label == "" {
		return ""
	}
	if !strings.Contains(label, "page") {
		label += " page"
	}
	return label
}

func customControlActionType(cmd dsl.Command) string {
	switch cmd.Type {
	case dsl.CmdFill, dsl.CmdType:
		return "input"
	case dsl.CmdClick:
		return "click"
	case dsl.CmdDoubleClick:
		return "double_click"
	case dsl.CmdRightClick:
		return "right_click"
	case dsl.CmdHover:
		return "hover"
	case dsl.CmdSelect:
		return "select"
	case dsl.CmdCheck:
		return "check"
	case dsl.CmdUncheck:
		return "uncheck"
	case dsl.CmdUploadFile:
		return "upload"
	default:
		return ""
	}
}

func customControlValue(rt *Runtime, cmd dsl.Command) string {
	switch cmd.Type {
	case dsl.CmdFill, dsl.CmdType, dsl.CmdSelect:
		return rt.resolveVariables(cmd.Value)
	case dsl.CmdUploadFile:
		if cmd.UploadFilePath != "" {
			return rt.resolveVariables(cmd.UploadFilePath)
		}
		return rt.resolveVariables(cmd.UploadFile)
	case dsl.CmdCheck:
		return "true"
	case dsl.CmdUncheck:
		return "false"
	default:
		return ""
	}
}

func customControlKey(page, target string) string {
	pageKey := normalizeRegistryLabel(page)
	targetKey := normalizeRegistryLabel(target)
	if pageKey == "" || targetKey == "" {
		return ""
	}
	return pageKey + "\x00" + targetKey
}

func normalizeRegistryLabel(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

// ListCustomControls returns all registered custom controls as a sorted slice
// of "page → target" pairs.
func ListCustomControls() []struct{ Page, Target string } {
	customControlsMu.RLock()
	defer customControlsMu.RUnlock()

	out := make([]struct{ Page, Target string }, 0, len(customControls))
	for key := range customControls {
		parts := strings.SplitN(key, "\x00", 2)
		if len(parts) == 2 {
			out = append(out, struct{ Page, Target string }{Page: parts[0], Target: parts[1]})
		}
	}
	// Simple bubble sort for determinism (registry is small).
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[i].Page+out[i].Target > out[j].Page+out[j].Target {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
