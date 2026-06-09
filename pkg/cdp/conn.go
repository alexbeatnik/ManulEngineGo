package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// msgReq represents a JSON-RPC 2.0 request.
type msgReq struct {
	ID     int64       `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// msgResp represents a JSON-RPC 2.0 response or event.
type msgResp struct {
	ID     int64           `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *msgError       `json:"error,omitempty"`
}

type msgError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// eventChanCap is the buffer size of every event subscription channel.
// A slow subscriber that fails to drain will lose events past this depth
// (publisher uses non-blocking send), but will never block the read loop.
const eventChanCap = 64

// Conn is a live WebSocket connection to a Chrome DevTools Protocol target.
// It is safe for concurrent use from multiple goroutines.
type Conn struct {
	wsURL string
	ws    *websocket.Conn

	// idSeq is the JSON-RPC request ID counter. Atomic to avoid mutex
	// contention on the hot Call() path.
	idSeq atomic.Int64

	// writeMu serializes WriteJSON calls — gorilla/websocket requires
	// external synchronization on writes.
	writeMu sync.Mutex

	// pendingMu guards the pending map.
	pendingMu sync.Mutex
	pending   map[int64]chan *msgResp

	// subsMu guards the eventSubs map.
	subsMu    sync.Mutex
	subsSeq   uint64
	eventSubs map[uint64]chan *msgResp

	// closeOnce guarantees Close() is idempotent and triggers cleanup
	// (cancel context, close socket) exactly once.
	closeOnce sync.Once

	ctx    context.Context
	cancel context.CancelFunc

	// parentCtx is the caller's Dial-time context. connCtx is deliberately
	// detached from it (so a short dial deadline doesn't kill a live conn),
	// but Call still honors parentCtx cancellation directly: a goroutine
	// bridges parentCtx.Done → Close asynchronously, so without this a Call
	// issued right after the parent is cancelled could race the teardown and
	// succeed. Checking parentCtx.Err() up front makes that deterministic.
	parentCtx context.Context
}

// Subscription is a handle to a CDP event subscription. The caller must
// invoke Close() when done — typically via `defer sub.Close()` — to release
// the underlying channel slot. Forgetting Close() leaks one channel per
// orphaned subscription.
type Subscription struct {
	id   uint64
	ch   chan *msgResp
	conn *Conn
	once sync.Once
}

// C returns the receive channel for this subscription. The channel is closed
// by the publisher when the connection terminates or Close() is called.
func (s *Subscription) C() <-chan *msgResp {
	if s == nil {
		return nil
	}
	return s.ch
}

// Close releases the subscription. Safe to call multiple times.
func (s *Subscription) Close() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		s.conn.unsubscribe(s.id)
	})
}

// DialTarget establishes a WebSocket connection to the given target URL.
func DialTarget(ctx context.Context, wsURL string) (*Conn, error) {
	ws, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial %s: %w", wsURL, err)
	}

	connCtx, cancel := context.WithCancel(context.Background())
	c := &Conn{
		wsURL:     wsURL,
		ws:        ws,
		pending:   make(map[int64]chan *msgResp),
		eventSubs: make(map[uint64]chan *msgResp),
		ctx:       connCtx,
		cancel:    cancel,
		parentCtx: ctx,
	}

	// Watch for parent ctx cancellation: tear down the connection.
	// This makes Dial-time deadlines propagate to an idle connection.
	go func() {
		select {
		case <-ctx.Done():
			_ = c.Close()
		case <-connCtx.Done():
		}
	}()

	go c.readLoop()
	return c, nil
}

// Close terminates the WebSocket connection. Safe to call multiple times.
// On Close, all in-flight Call() invocations return an error, and every
// active Subscription channel is closed.
func (c *Conn) Close() error {
	if c == nil {
		return nil
	}
	var closeErr error
	c.closeOnce.Do(func() {
		c.cancel()
		if c.ws != nil {
			closeErr = c.ws.Close()
		}
		// Drain pending request waiters: closing their channels would
		// race with readLoop's send, so instead let Call()'s
		// `<-c.ctx.Done()` branch fire after cancel() above.

		// Close all subscription channels so subscribers exit cleanly.
		c.subsMu.Lock()
		for id, ch := range c.eventSubs {
			close(ch)
			delete(c.eventSubs, id)
		}
		c.subsMu.Unlock()
	})
	return closeErr
}

func (c *Conn) readLoop() {
	// Calling Close() (idempotent via closeOnce) ensures subscription channels
	// are closed when the read loop exits for any reason — including unexpected
	// WebSocket errors from the remote end.
	defer c.Close()
	for {
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		// Bail early if the connection has been closed externally.
		select {
		case <-c.ctx.Done():
			return
		default:
		}
		var resp msgResp
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue
		}

		if resp.ID != 0 {
			c.pendingMu.Lock()
			ch, ok := c.pending[resp.ID]
			if ok {
				delete(c.pending, resp.ID)
			}
			c.pendingMu.Unlock()
			if ok {
				ch <- &resp
			}
			continue
		}
		if resp.Method != "" {
			c.subsMu.Lock()
			for _, sub := range c.eventSubs {
				select {
				case sub <- &resp:
				default:
				}
			}
			c.subsMu.Unlock()
		}
	}
}

// Subscribe returns a handle to a CDP event subscription. The caller MUST
// invoke Close() on the returned Subscription when done; failing to do so
// leaks one buffered channel per orphaned subscription.
//
// If the connection is already closed, Subscribe returns a Subscription whose
// channel is already closed, so any receive on sub.C() exits immediately.
func (c *Conn) Subscribe() *Subscription {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()
	// Check ctx.Done() while holding subsMu. Close() calls cancel() before
	// acquiring subsMu, so this check is the safe seam: either we see Done
	// and return a pre-closed channel, or Close() will find and close our
	// newly registered channel when it acquires the lock next.
	select {
	case <-c.ctx.Done():
		ch := make(chan *msgResp)
		close(ch)
		return &Subscription{ch: ch, conn: c}
	default:
	}
	c.subsSeq++
	id := c.subsSeq
	ch := make(chan *msgResp, eventChanCap)
	c.eventSubs[id] = ch
	return &Subscription{id: id, ch: ch, conn: c}
}

func (c *Conn) unsubscribe(id uint64) {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()
	if ch, ok := c.eventSubs[id]; ok {
		delete(c.eventSubs, id)
		// Close so any blocked receiver wakes up.
		// Safe because readLoop holds subsMu while sending.
		close(ch)
	}
}

// Call sends a JSON-RPC request and waits for the response.
func (c *Conn) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	// Honor the Dial-time parent context up front: once it's cancelled the
	// connection is being (or about to be) torn down, so fail deterministically
	// instead of racing the async teardown goroutine.
	if c.parentCtx != nil {
		if err := c.parentCtx.Err(); err != nil {
			return nil, fmt.Errorf("cdp connection closed: %w", err)
		}
	}

	id := c.idSeq.Add(1)
	ch := make(chan *msgResp, 1)

	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	req := msgReq{
		ID:     id,
		Method: method,
		Params: params,
	}

	c.writeMu.Lock()
	err := c.ws.WriteJSON(req)
	c.writeMu.Unlock()
	if err != nil {
		c.deletePending(id)
		return nil, fmt.Errorf("cdp write: %w", err)
	}

	select {
	case <-ctx.Done():
		c.deletePending(id)
		return nil, ctx.Err()
	case <-c.parentCtx.Done():
		// Parent cancelled while this Call was in flight — teardown is coming.
		c.deletePending(id)
		return nil, fmt.Errorf("cdp connection closed")
	case <-c.ctx.Done():
		c.deletePending(id)
		return nil, fmt.Errorf("cdp connection closed")
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("cdp error: code=%d message=%s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

func (c *Conn) deletePending(id int64) {
	c.pendingMu.Lock()
	delete(c.pending, id)
	c.pendingMu.Unlock()
}

// ListTargets queries the Chrome HTTP endpoint and returns all available targets.
func ListTargets(ctx context.Context, endpoint string) ([]Target, error) {
	url := endpoint
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = fmt.Sprintf("http://%s", endpoint)
	}
	url = strings.TrimSuffix(url, "/") + "/json/list"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
	}
	defer resp.Body.Close()

	var targets []Target
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("decode targets: %w", err)
	}
	return targets, nil
}

// FindPageTarget returns the first page-type target from a list.
func FindPageTarget(targets []Target) (Target, error) {
	for _, t := range targets {
		if t.Type == "page" {
			return t, nil
		}
	}
	return Target{}, fmt.Errorf("cdp.FindPageTarget: no page target found among %d targets", len(targets))
}
