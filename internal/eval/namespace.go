// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

// Package eval implements the losp evaluator.
package eval

import (
	"sync"

	"nickandperla.net/losp/internal/expr"
)

// Namespace is a thread-safe global namespace for losp variables.
type Namespace struct {
	mu    sync.RWMutex
	store map[string]expr.Expr
}

// NewNamespace creates a new empty namespace.
func NewNamespace() *Namespace {
	return &Namespace{
		store: make(map[string]expr.Expr),
	}
}

// Get retrieves an expression by name. Returns Empty if not found.
func (n *Namespace) Get(name string) expr.Expr {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if e, ok := n.store[name]; ok {
		return e
	}
	return expr.Empty{}
}

// Set stores an expression by name.
func (n *Namespace) Set(name string, e expr.Expr) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.store[name] = e
}

// Has returns true if the name exists in the namespace.
func (n *Namespace) Has(name string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	_, ok := n.store[name]
	return ok
}

// Delete removes an expression from the namespace.
func (n *Namespace) Delete(name string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.store, name)
}

// Clone creates a shallow copy of the namespace.
func (n *Namespace) Clone() *Namespace {
	n.mu.RLock()
	defer n.mu.RUnlock()
	clone := NewNamespace()
	for k, v := range n.store {
		clone.store[k] = v
	}
	return clone
}
