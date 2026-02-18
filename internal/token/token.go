// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

// Package token defines losp token types and Unicode operator constants.
package token

// Token represents a losp token type.
type Token int

const (
	EOF Token = iota
	TEXT

	// Operators (Unicode)
	STORE        // ▼ U+25BC - Store expression (deferred)
	IMM_STORE    // ▽ U+25BD - Evaluate now, store result
	RETRIEVE     // ▲ U+25B2 - Retrieve stored expression
	IMM_RETRIEVE // △ U+25B3 - Retrieve now, substitute into stream
	EXECUTE      // ▶ U+25B6 - Execute named expression or builtin
	IMM_EXECUTE  // ▷ U+25B7 - Execute now, substitute result
	PLACEHOLDER  // □ U+25A1 - Declare argument slot
	DEFER        // ◯ U+25EF - Prevent parse-time resolution
	TERMINATOR   // ◆ U+25C6 - End current operator's scope
)

// Unicode runes for each operator.
const (
	RuneStore       = '▼' // U+25BC
	RuneImmStore    = '▽' // U+25BD
	RuneRetrieve    = '▲' // U+25B2
	RuneImmRetrieve = '△' // U+25B3
	RuneExecute     = '▶' // U+25B6
	RuneImmExecute  = '▷' // U+25B7
	RunePlaceholder = '□' // U+25A1
	RuneDefer       = '◯' // U+25EF
	RuneTerminator  = '◆' // U+25C6
)

// IsOperator returns true if the rune is a losp operator.
func IsOperator(r rune) bool {
	switch r {
	case RuneStore, RuneImmStore, RuneRetrieve, RuneImmRetrieve,
		RuneExecute, RuneImmExecute, RunePlaceholder, RuneDefer, RuneTerminator:
		return true
	}
	return false
}

// TokenFromRune returns the token type for an operator rune.
func TokenFromRune(r rune) Token {
	switch r {
	case RuneStore:
		return STORE
	case RuneImmStore:
		return IMM_STORE
	case RuneRetrieve:
		return RETRIEVE
	case RuneImmRetrieve:
		return IMM_RETRIEVE
	case RuneExecute:
		return EXECUTE
	case RuneImmExecute:
		return IMM_EXECUTE
	case RunePlaceholder:
		return PLACEHOLDER
	case RuneDefer:
		return DEFER
	case RuneTerminator:
		return TERMINATOR
	}
	return TEXT
}

// String returns the string representation of a token.
func (t Token) String() string {
	switch t {
	case EOF:
		return "EOF"
	case TEXT:
		return "TEXT"
	case STORE:
		return "STORE"
	case IMM_STORE:
		return "IMM_STORE"
	case RETRIEVE:
		return "RETRIEVE"
	case IMM_RETRIEVE:
		return "IMM_RETRIEVE"
	case EXECUTE:
		return "EXECUTE"
	case IMM_EXECUTE:
		return "IMM_EXECUTE"
	case PLACEHOLDER:
		return "PLACEHOLDER"
	case DEFER:
		return "DEFER"
	case TERMINATOR:
		return "TERMINATOR"
	}
	return "UNKNOWN"
}

// IsImmediate returns true if the token is a parse-time (immediate) operator.
func (t Token) IsImmediate() bool {
	switch t {
	case IMM_STORE, IMM_RETRIEVE, IMM_EXECUTE:
		return true
	}
	return false
}

// IsDeferred returns true if the token is an execution-time (deferred) operator.
func (t Token) IsDeferred() bool {
	switch t {
	case STORE, RETRIEVE, EXECUTE:
		return true
	}
	return false
}

// NeedsTerminator returns true if the operator requires a terminator.
func (t Token) NeedsTerminator() bool {
	switch t {
	case STORE, IMM_STORE, EXECUTE, IMM_EXECUTE:
		return true
	}
	return false
}
