package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexbeatnik/ManulEngineGo/pkg/browser"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dom"
)

// MockPage is a test-only browser.Page implementation that operates on
// pre-defined DOM snapshots instead of a real browser.
type MockPage struct {
	URL      string
	Title    string
	Elements []dom.ElementSnapshot

	// Record of interaction calls
	Clicks       []Point
	Inputs       map[string]string // xpath -> value
	FileInputs   map[string][]string
	LastNavigate string
	ProbeCalls   int
}

type Point struct {
	X, Y float64
}

func (m *MockPage) Navigate(ctx context.Context, url string) error {
	m.URL = url
	m.LastNavigate = url
	return nil
}

func (m *MockPage) EvalJS(ctx context.Context, expr string) ([]byte, error) {
	if strings.Contains(expr, "document.title") {
		return json.Marshal(m.Title)
	}
	return nil, nil
}

func (m *MockPage) CallProbe(ctx context.Context, fn string, arg any) ([]byte, error) {
	m.ProbeCalls++
	// If it's a data extraction probe, we simulate the logic:
	// Find element matching the target text from arg and return its value/text.
	if strings.Contains(fn, "classified") && strings.Contains(fn, "allTables") { // Simple detection for extract_data.js
		params, _ := arg.([]string)
		if len(params) > 0 {
			target := strings.ToLower(params[0])
			for _, el := range m.Elements {
				if strings.Contains(strings.ToLower(el.VisibleText), target) || strings.ToLower(el.Tag) == target {
					if el.Value != "" {
						return []byte(el.Value), nil
					}
					return []byte(el.VisibleText), nil
				}
			}
		}
		return []byte("null"), nil
	}

	// Default: return the all-elements snapshot
	res := dom.PageSnapshot{
		URL:         m.URL,
		VisibleText: "Mock Page Content",
		Elements:    m.Elements,
	}
	return json.Marshal(res)
}

func (m *MockPage) Click(ctx context.Context, x, y float64) error {
	m.Clicks = append(m.Clicks, Point{x, y})
	return nil
}

func (m *MockPage) Focus(ctx context.Context, id int, xpath string) error { return nil }

func (m *MockPage) SetInputValue(ctx context.Context, id int, xpath, value string) error {
	if m.Inputs == nil {
		m.Inputs = make(map[string]string)
	}
	m.Inputs[xpath] = value
	return nil
}

func (m *MockPage) SetChecked(ctx context.Context, id int, xpath string, checked bool) error {
	for i := range m.Elements {
		if m.Elements[i].XPath == xpath || m.Elements[i].ID == id {
			m.Elements[i].IsChecked = checked
			return nil
		}
	}
	return nil
}

func (m *MockPage) ScrollIntoView(ctx context.Context, id int, xpath string) error { return nil }

func (m *MockPage) ScrollPage(ctx context.Context, direction, container string) error { return nil }

func (m *MockPage) DoubleClick(ctx context.Context, x, y float64) error { return nil }

func (m *MockPage) RightClick(ctx context.Context, x, y float64) error { return nil }

func (m *MockPage) Hover(ctx context.Context, x, y float64) error { return nil }

func (m *MockPage) DragAndDrop(ctx context.Context, fX, fY, tX, tY float64) error { return nil }

func (m *MockPage) SetFileInput(ctx context.Context, id int, xpath string, paths []string) error {
	if m.FileInputs == nil {
		m.FileInputs = make(map[string][]string)
	}
	copyPaths := append([]string(nil), paths...)
	m.FileInputs[xpath] = copyPaths
	return nil
}

func (m *MockPage) Screenshot(ctx context.Context) ([]byte, error) { return nil, nil }

func (m *MockPage) WaitForResponse(ctx context.Context, pattern string, timeout time.Duration) error {
	return nil
}

func (m *MockPage) HighlightElement(ctx context.Context, id int, xpath string, duration int) error {
	return nil
}

func (m *MockPage) ClearHighlight(ctx context.Context) error {
	return nil
}

func (m *MockPage) GetElementCenter(ctx context.Context, id int, xpath string) (float64, float64, error) {
	for _, el := range m.Elements {
		if el.XPath == xpath || el.ID == id {
			return el.Rect.Left + el.Rect.Width/2, el.Rect.Top + el.Rect.Height/2, nil
		}
	}
	return 0, 0, fmt.Errorf("element not found: %s (ID=%d)", xpath, id)
}

func (m *MockPage) DispatchKey(ctx context.Context, key string, mods int) error { return nil }

func (m *MockPage) CurrentURL(ctx context.Context) (string, error) { return m.URL, nil }

func (m *MockPage) WaitForLoad(ctx context.Context) error { return nil }

func (m *MockPage) Wait(ctx context.Context, d time.Duration) error { return nil }

// Frames reports a single main frame (the mock has no real iframes).
func (m *MockPage) Frames(ctx context.Context) ([]browser.FrameRef, error) {
	return []browser.FrameRef{{Index: 0}}, nil
}

// EvalJSInFrame ignores the frame index (single-frame mock) and delegates.
func (m *MockPage) EvalJSInFrame(ctx context.Context, frameIndex int, expr string) ([]byte, error) {
	return m.EvalJS(ctx, expr)
}

// CallProbeInFrame ignores the frame index (single-frame mock) and delegates.
func (m *MockPage) CallProbeInFrame(ctx context.Context, frameIndex int, fn string, arg any) ([]byte, error) {
	return m.CallProbe(ctx, fn, arg)
}

func (m *MockPage) Close() error { return nil }
