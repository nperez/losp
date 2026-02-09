// Package store provides persistence for losp expressions.
package store

import "nickandperla.net/losp/internal/expr"

// Store is the interface for expression persistence.
type Store interface {
	// Get retrieves an expression by name. Returns nil if not found.
	Get(name string) (expr.Expr, error)
	// Put stores an expression by name, overwriting if it exists.
	Put(name string, e expr.Expr) error
	// Delete removes an expression by name.
	Delete(name string) error
	// Close releases resources.
	Close() error
}

// VersionEntry represents a single version of a persisted expression.
type VersionEntry struct {
	Version int
	Value   string
	Ts      string
}

// HistoryStore extends Store with version history queries.
type HistoryStore interface {
	GetHistory(name string, limit int) ([]VersionEntry, error)
}
