// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// AsyncHandle represents a forked async computation or timer.
type AsyncHandle struct {
	id       string
	done     chan struct{}
	result   string
	err      error
	isTimer  bool
	fireAt   time.Time
	duration time.Duration
	timer    *time.Timer
}

// AsyncRegistry manages async handles across evaluators.
type AsyncRegistry struct {
	mu      sync.Mutex
	handles map[string]*AsyncHandle
	counter atomic.Int64
	wg      sync.WaitGroup
}

// NewAsyncRegistry creates a new async registry.
func NewAsyncRegistry() *AsyncRegistry {
	return &AsyncRegistry{
		handles: make(map[string]*AsyncHandle),
	}
}

// Register creates a new handle and registers it.
func (r *AsyncRegistry) Register(isTimer bool, duration time.Duration) *AsyncHandle {
	id := fmt.Sprintf("_async_%d", r.counter.Add(1))
	h := &AsyncHandle{
		id:       id,
		done:     make(chan struct{}),
		isTimer:  isTimer,
		duration: duration,
	}
	if isTimer {
		h.fireAt = time.Now().Add(duration)
	}
	r.mu.Lock()
	r.handles[id] = h
	r.mu.Unlock()
	return h
}

// Get retrieves a handle by ID.
func (r *AsyncRegistry) Get(id string) *AsyncHandle {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.handles[id]
}

// Shutdown stops pending timers and waits for running goroutines.
func (r *AsyncRegistry) Shutdown() {
	r.mu.Lock()
	for _, h := range r.handles {
		if h.timer != nil {
			h.timer.Stop()
		}
	}
	r.mu.Unlock()

	// Wait for running goroutines with a timeout
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}
