// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"strconv"
	"strings"

	"nickandperla.net/losp/internal/expr"
)

// Sentinels returned by the positional list builtins. They exist so that a
// failed lookup is distinguishable from a lookup that legitimately landed on an
// empty slot: an empty slot returns EMPTY, a broken lookup returns one of these.
const (
	sentinelInvalidIndex = "INVALID_INDEX"
	sentinelOutOfRange   = "OUT_OF_RANGE"
)

func builtinGrab(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// GRAB index list
	// Returns the item at index (0-based, negative counts from the end).
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	// The index is parsed before anything else so that a call site which wrote
	// its arguments on one line fails loudly. In "▶SLICE 0 2 ▲L ◆" there are
	// only two arguments: the text expression "0 2" and the list. Atoi fails
	// on "0 2", and without this the builtin would quietly return EMPTY.
	if len(args) < 1 {
		return expr.Stored{Body: sentinelInvalidIndex}, nil
	}
	idx, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil {
		return expr.Stored{Body: sentinelInvalidIndex}, nil
	}

	// A list is a list of expressions. A single argument is one list
	// expression, re-parsed into the expressions it holds; more than one means
	// the items arrived as separate expressions at the call site.
	items := args[1:]
	if len(items) == 1 {
		items, err = e.parseArgs(items[0])
		if err != nil {
			return nil, err
		}
	}

	// Negative indices count back from the end: -1 is the last item.
	if idx < 0 {
		idx += len(items)
	}
	if idx < 0 || idx >= len(items) {
		return expr.Stored{Body: sentinelOutOfRange}, nil
	}

	return expr.Stored{Body: items[idx]}, nil
}

func builtinFirst(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// FIRST list — sugar for GRAB 0
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	items := args
	if len(items) == 1 {
		items, err = e.parseArgs(items[0])
		if err != nil {
			return nil, err
		}
	}

	if len(items) == 0 {
		return expr.Stored{Body: sentinelOutOfRange}, nil
	}
	return expr.Stored{Body: items[0]}, nil
}

func builtinLast(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// LAST list — sugar for GRAB -1
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	items := args
	if len(items) == 1 {
		items, err = e.parseArgs(items[0])
		if err != nil {
			return nil, err
		}
	}

	if len(items) == 0 {
		return expr.Stored{Body: sentinelOutOfRange}, nil
	}
	return expr.Stored{Body: items[len(items)-1]}, nil
}

func builtinSlice(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// SLICE start end list
	// Half-open [start, end). Both bounds may be negative (counting from the
	// end) and both are clamped to the list, so a partially out-of-range slice
	// returns the overlap instead of failing. A blank bound (▲EMPTY) means
	// "from the beginning" / "to the end".
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	// Both bounds are parsed before anything else so that a call site which
	// wrote its arguments on one line fails loudly rather than returning
	// EMPTY. See the note in builtinGrab.
	if len(args) < 2 {
		return expr.Stored{Body: sentinelInvalidIndex}, nil
	}

	items := args[2:]
	if len(items) == 1 {
		items, err = e.parseArgs(items[0])
		if err != nil {
			return nil, err
		}
	}
	n := len(items)

	start := 0
	if s := strings.TrimSpace(args[0]); s != "" {
		start, err = strconv.Atoi(s)
		if err != nil {
			return expr.Stored{Body: sentinelInvalidIndex}, nil
		}
	}
	end := n
	if s := strings.TrimSpace(args[1]); s != "" {
		end, err = strconv.Atoi(s)
		if err != nil {
			return expr.Stored{Body: sentinelInvalidIndex}, nil
		}
	}

	if start < 0 {
		start += n
	}
	if start < 0 {
		start = 0
	}
	if start > n {
		start = n
	}
	if end < 0 {
		end += n
	}
	if end < 0 {
		end = 0
	}
	if end > n {
		end = n
	}

	if start >= end {
		return expr.Empty{}, nil
	}

	return expr.Stored{Body: strings.Join(items[start:end], "\n")}, nil
}
