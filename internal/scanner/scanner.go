// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

// Package scanner provides a streaming Unicode-aware lexer for losp.
package scanner

import (
	"bufio"
	"io"
	"strings"
	"unicode"

	"nickandperla.net/losp/internal/token"
)

// Scanner tokenizes losp input rune-by-rune.
type Scanner struct {
	reader *bufio.Reader
	buf    strings.Builder
	peeked *Item
	line   int // Current line number (1-based)
}

// Item represents a scanned token with its value.
type Item struct {
	Token token.Token
	Value string
	Line  int // Line number where this token started
}

// New creates a new Scanner from an io.Reader.
func New(r io.Reader) *Scanner {
	return &Scanner{
		reader: bufio.NewReader(r),
		line:   1,
	}
}

// Line returns the current line number (1-based).
func (s *Scanner) Line() int {
	return s.line
}

// NewFromString creates a new Scanner from a string.
func NewFromString(s string) *Scanner {
	return New(strings.NewReader(s))
}

// Peek returns the next item without consuming it.
func (s *Scanner) Peek() (*Item, error) {
	if s.peeked != nil {
		return s.peeked, nil
	}
	item, err := s.Next()
	if err != nil {
		return nil, err
	}
	s.peeked = item
	return item, nil
}

// Next returns the next token from the input.
func (s *Scanner) Next() (*Item, error) {
	if s.peeked != nil {
		item := s.peeked
		s.peeked = nil
		return item, nil
	}

	s.buf.Reset()
	startLine := s.line

	for {
		r, _, err := s.reader.ReadRune()
		if err == io.EOF {
			if s.buf.Len() > 0 {
				return &Item{Token: token.TEXT, Value: s.buf.String(), Line: startLine}, nil
			}
			return &Item{Token: token.EOF, Line: s.line}, nil
		}
		if err != nil {
			return nil, err
		}

		// Track newlines
		if r == '\n' {
			s.line++
		}

		if token.IsOperator(r) {
			// If we have accumulated text, return it first
			if s.buf.Len() > 0 {
				// Put the operator back (and undo line count if it was newline)
				s.reader.UnreadRune()
				if r == '\n' {
					s.line--
				}
				return &Item{Token: token.TEXT, Value: s.buf.String(), Line: startLine}, nil
			}
			// Return the operator
			return &Item{Token: token.TokenFromRune(r), Value: string(r), Line: s.line}, nil
		}

		s.buf.WriteRune(r)
	}
}

// isIdentChar returns true if the rune is valid in an identifier (letter, digit, underscore).
func isIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// ScanName scans the next identifier name from the input.
// Skips leading whitespace, returns the identifier.
// Identifiers consist of letters, digits, and underscores.
func (s *Scanner) ScanName() (string, error) {
	var name strings.Builder

	// Skip leading whitespace
	for {
		r, _, err := s.reader.ReadRune()
		if err == io.EOF {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		if r == '\n' {
			s.line++
		}
		if token.IsOperator(r) {
			s.reader.UnreadRune()
			if r == '\n' {
				s.line--
			}
			return "", nil
		}
		if !unicode.IsSpace(r) {
			if isIdentChar(r) {
				name.WriteRune(r)
			} else {
				s.reader.UnreadRune()
				return "", nil
			}
			break
		}
	}

	// Read identifier characters
	for {
		r, _, err := s.reader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if !isIdentChar(r) {
			s.reader.UnreadRune()
			break
		}
		name.WriteRune(r)
	}

	return name.String(), nil
}

// ScanUntilTerminator scans all content until a terminator is found.
// Returns the content and respects nested operators with terminators.
// The terminator is consumed but not included in the result.
func (s *Scanner) ScanUntilTerminator() (string, error) {
	var content strings.Builder
	depth := 1 // We start inside one operator

	for {
		r, _, err := s.reader.ReadRune()
		if err == io.EOF {
			// Unterminated - return what we have
			return content.String(), nil
		}
		if err != nil {
			return "", err
		}

		// Track newlines for accurate line numbers
		if r == '\n' {
			s.line++
		}

		if r == token.RuneTerminator {
			depth--
			if depth == 0 {
				return content.String(), nil
			}
		} else if r == token.RuneStore || r == token.RuneImmStore ||
			r == token.RuneExecute || r == token.RuneImmExecute {
			depth++
		}

		content.WriteRune(r)
	}
}

// SkipWhitespace consumes and discards whitespace.
func (s *Scanner) SkipWhitespace() error {
	for {
		r, _, err := s.reader.ReadRune()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if r == '\n' {
			s.line++
		}
		if !unicode.IsSpace(r) {
			s.reader.UnreadRune()
			if r == '\n' {
				s.line--
			}
			return nil
		}
	}
}

// PeekRune returns the next non-whitespace rune without consuming it.
// Returns 0 on EOF.
func (s *Scanner) PeekRune() (rune, error) {
	// Skip whitespace first
	if err := s.SkipWhitespace(); err != nil {
		return 0, err
	}

	r, _, err := s.reader.ReadRune()
	if err == io.EOF {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	// Put it back
	s.reader.UnreadRune()
	return r, nil
}
