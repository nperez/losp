package store

import (
	"sync"

	"nickandperla.net/losp/internal/expr"
)

// Memory is an in-memory store for testing.
type Memory struct {
	mu       sync.RWMutex
	data     map[string]expr.Expr
	metadata map[string]string
}

// NewMemory creates a new in-memory store.
func NewMemory() *Memory {
	return &Memory{
		data:     make(map[string]expr.Expr),
		metadata: make(map[string]string),
	}
}

// Get retrieves an expression by name.
func (m *Memory) Get(name string) (expr.Expr, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.data[name]; ok {
		return e, nil
	}
	return nil, nil
}

// Put stores an expression by name.
func (m *Memory) Put(name string, e expr.Expr) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[name] = e
	return nil
}

// Delete removes an expression by name.
func (m *Memory) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, name)
	return nil
}

// Close is a no-op for memory store.
func (m *Memory) Close() error {
	return nil
}

// GetMetadata retrieves a metadata value by key.
func (m *Memory) GetMetadata(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metadata[key], nil
}

// SetMetadata stores a metadata value by key.
func (m *Memory) SetMetadata(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata[key] = value
	return nil
}
