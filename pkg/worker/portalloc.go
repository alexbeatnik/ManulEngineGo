// Package worker provides goroutine-safe primitives for executing ManulEngine (Go)
// hunts in parallel: per-worker Runtime/Page/Chrome ownership, a bounded
// worker pool, and CDP debug-port allocation.
package worker

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// ErrPortsExhausted is returned by PortAllocator.Acquire when no port in
// the configured range is available.
var ErrPortsExhausted = errors.New("worker: port allocator exhausted")

// PortAllocator hands out CDP debug ports from a fixed range and tracks
// in-use ports so two workers never collide. Released ports return to the
// pool. Safe for concurrent use.
type PortAllocator struct {
	mu        sync.Mutex
	start     int
	end       int // inclusive
	cursor    int // round-robin starting point
	inUse     map[int]bool
	checkPort func(int) bool // nil → portFree (OS-level bind check)
}

// NewPortAllocator creates an allocator covering [start, end] inclusive.
// A typical value: NewPortAllocator(9222, 9321) for 100 concurrent workers.
func NewPortAllocator(start, end int) *PortAllocator {
	if end < start {
		start, end = end, start
	}
	return &PortAllocator{
		start:  start,
		end:    end,
		cursor: start,
		inUse:  make(map[int]bool, end-start+1),
	}
}

// Acquire returns a free port and marks it in-use. The caller MUST call
// Release(port) when done. Returns ErrPortsExhausted if the range is full
// or every port is currently bound by another OS process.
func (a *PortAllocator) Acquire() (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	total := a.end - a.start + 1
	for tried := 0; tried < total; tried++ {
		port := a.cursor
		a.cursor++
		if a.cursor > a.end {
			a.cursor = a.start
		}
		if a.inUse[port] {
			continue
		}
		// Avoid handing out a port some other process is already bound to.
		// This is a best-effort check; the port could be claimed between
		// here and Chrome's bind() call. Chrome will fail to start in that
		// case, and the caller can simply retry Acquire.
		check := a.checkPort
		if check == nil {
			check = portFree
		}
		if !check(port) {
			continue
		}
		a.inUse[port] = true
		return port, nil
	}
	return 0, ErrPortsExhausted
}

// Release returns a port to the pool. Safe to call with a port not owned
// by this allocator (no-op).
func (a *PortAllocator) Release(port int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.inUse, port)
}

// portFree returns true if the port can be bound on the IPv4 loopback
// address (127.0.0.1). Failure to bind there means the port is treated
// as unavailable.
func portFree(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}
