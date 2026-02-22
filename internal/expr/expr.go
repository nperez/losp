// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

// Package expr defines losp expression types.
package expr

import (
	"strings"

	"nickandperla.net/losp/internal/token"
)

// Expr is the interface all expression types implement.
type Expr interface {
	// String returns the serializable representation of the expression.
	String() string
	// IsEmpty returns true if this is an empty expression.
	IsEmpty() bool
}

// Empty represents an empty/absent value.
type Empty struct{}

func (e Empty) String() string { return "" }
func (e Empty) IsEmpty() bool  { return true }

// Placeholder represents an argument slot (□name).
type Placeholder struct {
	Name string
}

func (p Placeholder) String() string {
	return string(token.RunePlaceholder) + p.Name
}
func (p Placeholder) IsEmpty() bool { return false }

// Operator represents an operator expression (▼name body ◆).
type Operator struct {
	Op   token.Token
	Name string
	Body Expr // Body content (for STORE/IMM_STORE/EXECUTE/IMM_EXECUTE)
}

func (o Operator) String() string {
	var sb strings.Builder
	switch o.Op {
	case token.STORE:
		sb.WriteRune(token.RuneStore)
	case token.IMM_STORE:
		sb.WriteRune(token.RuneImmStore)
	case token.RETRIEVE:
		sb.WriteRune(token.RuneRetrieve)
	case token.IMM_RETRIEVE:
		sb.WriteRune(token.RuneImmRetrieve)
	case token.EXECUTE:
		sb.WriteRune(token.RuneExecute)
	case token.IMM_EXECUTE:
		sb.WriteRune(token.RuneImmExecute)
	case token.DEFER:
		sb.WriteRune(token.RuneDefer)
	}
	sb.WriteString(o.Name)
	if o.Body != nil && !o.Body.IsEmpty() {
		sb.WriteString(" ")
		sb.WriteString(o.Body.String())
	}
	if o.Op.NeedsTerminator() {
		sb.WriteRune(token.RuneTerminator)
	}
	return sb.String()
}
func (o Operator) IsEmpty() bool { return false }

// Stored represents a stored expression with optional parameters.
// This is the universal value type for the namespace — all values
// (plain text, parameterized expressions, APPEND results) are Stored.
type Stored struct {
	Params []string // Placeholder names in order (nil for plain values)
	Body   string   // The expression body text
}

func (s Stored) String() string { return s.Body }
func (s Stored) IsEmpty() bool  { return s.Body == "" }

// Compound represents a sequence of expressions.
type Compound struct {
	Exprs []Expr
}

func (c Compound) String() string {
	var sb strings.Builder
	for _, e := range c.Exprs {
		sb.WriteString(e.String())
	}
	return sb.String()
}
func (c Compound) IsEmpty() bool {
	for _, e := range c.Exprs {
		if !e.IsEmpty() {
			return false
		}
	}
	return true
}

// NewText creates a new Stored expression, returning Empty if the value is empty.
func NewText(value string) Expr {
	if value == "" {
		return Empty{}
	}
	return Stored{Body: value}
}

// NewCompound creates a new Compound from expressions, simplifying if possible.
func NewCompound(exprs ...Expr) Expr {
	// Filter out empty expressions
	var nonEmpty []Expr
	for _, e := range exprs {
		if !e.IsEmpty() {
			nonEmpty = append(nonEmpty, e)
		}
	}

	switch len(nonEmpty) {
	case 0:
		return Empty{}
	case 1:
		return nonEmpty[0]
	default:
		return Compound{Exprs: nonEmpty}
	}
}

// Flatten returns a flat slice of expressions from potentially nested compounds.
func Flatten(e Expr) []Expr {
	if e == nil || e.IsEmpty() {
		return nil
	}
	if c, ok := e.(Compound); ok {
		var result []Expr
		for _, sub := range c.Exprs {
			result = append(result, Flatten(sub)...)
		}
		return result
	}
	return []Expr{e}
}
