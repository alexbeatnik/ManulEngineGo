package cdp

import (
	"context"
	"encoding/json"
	"sync"
)

// FrameInfo describes one frame (main document or iframe) and the JavaScript
// execution context Chrome created for it. Index 0 is always the main frame;
// positive indices are embedded frames in attach order.
type FrameInfo struct {
	FrameID   string
	URL       string
	Name      string
	ContextID int
	Index     int
}

// FrameTracker maintains the live frameId → execution-context-id mapping for a
// page session by listening to Runtime/Page CDP events. It is what lets
// ManulEngine (Go) evaluate JavaScript inside iframes (per-frame execution contexts)
// instead of only the default/main context.
//
// Mirrors the per-frame routing in the Python manul-engine CDP backend
// (manul_engine/cdp/page.py).
type FrameTracker struct {
	conn *Conn

	mu          sync.Mutex
	mainFrameID string
	frames      map[string]*FrameInfo // frameId -> info
	order       []string              // frame insertion order (main first)

	sub *Subscription
}

// NewFrameTracker enables the Page and Runtime domains, seeds the frame tree,
// and starts listening for execution-context lifecycle events. The returned
// tracker keeps a background goroutine alive until the connection closes.
func NewFrameTracker(ctx context.Context, conn *Conn) (*FrameTracker, error) {
	ft := &FrameTracker{
		conn:   conn,
		frames: make(map[string]*FrameInfo),
	}

	// Enabling Runtime makes Chrome emit executionContextCreated for every
	// frame; enabling Page makes it emit frameNavigated/frameAttached.
	if _, err := conn.Call(ctx, "Page.enable", nil); err != nil {
		return nil, err
	}
	if _, err := conn.Call(ctx, "Runtime.enable", nil); err != nil {
		return nil, err
	}

	ft.sub = conn.Subscribe()
	go ft.loop()

	ft.seedFrameTree(ctx)
	return ft, nil
}

// Frames returns the current frames, main frame first, then child frames in
// attach order. Frames whose execution context is not yet known are included
// with ContextID == 0 so callers can still see the frame.
func (ft *FrameTracker) Frames() []FrameInfo {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	out := make([]FrameInfo, 0, len(ft.order))
	idx := 0
	appendFrame := func(id string) {
		f, ok := ft.frames[id]
		if !ok {
			return
		}
		copyF := *f
		copyF.Index = idx
		idx++
		out = append(out, copyF)
	}
	if ft.mainFrameID != "" {
		appendFrame(ft.mainFrameID)
	}
	for _, id := range ft.order {
		if id == ft.mainFrameID {
			continue
		}
		appendFrame(id)
	}
	return out
}

// ContextForIndex resolves a frame index (as returned by Frames) to its
// execution context id. Index 0 / unknown returns 0 (the default context).
func (ft *FrameTracker) ContextForIndex(index int) int {
	if index <= 0 {
		return 0
	}
	frames := ft.Frames()
	for _, f := range frames {
		if f.Index == index {
			return f.ContextID
		}
	}
	return 0
}

func (ft *FrameTracker) seedFrameTree(ctx context.Context) {
	raw, err := ft.conn.Call(ctx, "Page.getFrameTree", nil)
	if err != nil {
		return
	}
	var resp struct {
		FrameTree frameTreeNode `json:"frameTree"`
	}
	if json.Unmarshal(raw, &resp) != nil {
		return
	}
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.ingest(&resp.FrameTree, true)
}

type frameTreeNode struct {
	Frame struct {
		ID       string `json:"id"`
		ParentID string `json:"parentId"`
		URL      string `json:"url"`
		Name     string `json:"name"`
	} `json:"frame"`
	ChildFrames []frameTreeNode `json:"childFrames"`
}

// ingest walks the frame tree; caller holds ft.mu.
func (ft *FrameTracker) ingest(node *frameTreeNode, isMain bool) {
	id := node.Frame.ID
	if id == "" {
		return
	}
	if isMain {
		ft.mainFrameID = id
	}
	ft.upsertLocked(id, node.Frame.URL, node.Frame.Name)
	for i := range node.ChildFrames {
		ft.ingest(&node.ChildFrames[i], false)
	}
}

// upsertLocked creates/updates a frame record; caller holds ft.mu.
func (ft *FrameTracker) upsertLocked(id, url, name string) *FrameInfo {
	f, ok := ft.frames[id]
	if !ok {
		f = &FrameInfo{FrameID: id}
		ft.frames[id] = f
		ft.order = append(ft.order, id)
	}
	if url != "" {
		f.URL = url
	}
	if name != "" {
		f.Name = name
	}
	return f
}

func (ft *FrameTracker) loop() {
	ch := ft.sub.C()
	for msg := range ch {
		switch msg.Method {
		case "Runtime.executionContextCreated":
			ft.onContextCreated(msg.Params)
		case "Runtime.executionContextDestroyed":
			ft.onContextDestroyed(msg.Params)
		case "Runtime.executionContextsCleared":
			ft.onContextsCleared()
		case "Page.frameNavigated":
			ft.onFrameNavigated(msg.Params)
		case "Page.frameAttached":
			ft.onFrameAttached(msg.Params)
		case "Page.frameDetached":
			ft.onFrameDetached(msg.Params)
		}
	}
}

func (ft *FrameTracker) onContextCreated(params json.RawMessage) {
	var p struct {
		Context struct {
			ID      int `json:"id"`
			AuxData struct {
				FrameID   string `json:"frameId"`
				IsDefault bool   `json:"isDefault"`
			} `json:"auxData"`
		} `json:"context"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	fid := p.Context.AuxData.FrameID
	if fid == "" || !p.Context.AuxData.IsDefault {
		return
	}
	ft.mu.Lock()
	f := ft.upsertLocked(fid, "", "")
	f.ContextID = p.Context.ID
	ft.mu.Unlock()
}

func (ft *FrameTracker) onContextDestroyed(params json.RawMessage) {
	var p struct {
		ExecutionContextID int `json:"executionContextId"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	ft.mu.Lock()
	for _, f := range ft.frames {
		if f.ContextID == p.ExecutionContextID {
			f.ContextID = 0
		}
	}
	ft.mu.Unlock()
}

func (ft *FrameTracker) onContextsCleared() {
	ft.mu.Lock()
	for _, f := range ft.frames {
		f.ContextID = 0
	}
	ft.mu.Unlock()
}

func (ft *FrameTracker) onFrameNavigated(params json.RawMessage) {
	var p struct {
		Frame struct {
			ID       string `json:"id"`
			ParentID string `json:"parentId"`
			URL      string `json:"url"`
			Name     string `json:"name"`
		} `json:"frame"`
	}
	if json.Unmarshal(params, &p) != nil || p.Frame.ID == "" {
		return
	}
	ft.mu.Lock()
	if p.Frame.ParentID == "" {
		ft.mainFrameID = p.Frame.ID
	}
	ft.upsertLocked(p.Frame.ID, p.Frame.URL, p.Frame.Name)
	ft.mu.Unlock()
}

func (ft *FrameTracker) onFrameAttached(params json.RawMessage) {
	var p struct {
		FrameID string `json:"frameId"`
	}
	if json.Unmarshal(params, &p) != nil || p.FrameID == "" {
		return
	}
	ft.mu.Lock()
	ft.upsertLocked(p.FrameID, "", "")
	ft.mu.Unlock()
}

func (ft *FrameTracker) onFrameDetached(params json.RawMessage) {
	var p struct {
		FrameID string `json:"frameId"`
	}
	if json.Unmarshal(params, &p) != nil || p.FrameID == "" {
		return
	}
	ft.mu.Lock()
	delete(ft.frames, p.FrameID)
	ft.order = removeString(ft.order, p.FrameID)
	ft.mu.Unlock()
}

func removeString(s []string, v string) []string {
	out := s[:0]
	for _, x := range s {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}
