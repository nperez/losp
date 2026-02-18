// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"fmt"
	"strings"

	"nickandperla.net/losp/internal/expr"
	"nickandperla.net/losp/internal/store"
	"nickandperla.net/losp/internal/token"
)

func builtinHistory(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return expr.Empty{}, nil
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return expr.Empty{}, nil
	}

	hs := historyStore(e)
	if hs == nil {
		return expr.Empty{}, nil
	}

	entries, err := hs.GetHistory(name, e.historyLimit)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return expr.Empty{}, nil
	}

	// Create ephemeral expressions for each version
	var names []string
	for _, ve := range entries {
		vName := fmt.Sprintf("_%s_%d", name, ve.Version)
		names = append(names, vName)

		// The DB value is already in formatAsDefinition format.
		// If it starts with ▼name (a Stored expression), use it as-is:
		// executing ▶_X_1 ◆ will re-evaluate the ▼X ... ◆ definition, redefining X.
		// If the DB value is plain text, wrap it as ▼X text ◆ so executing
		// ▶_X_1 ◆ redefines X to that text value.
		body := ve.Value
		trimmed := strings.TrimSpace(body)
		runes := []rune(trimmed)
		if len(runes) == 0 || runes[0] != token.RuneStore {
			// Plain text: wrap as ▼name text ◆
			body = string(token.RuneStore) + name + " " + body + " " + string(token.RuneTerminator)
		}

		// Store as a Stored expression in the namespace (NOT persisted)
		e.namespace.Set(vName, expr.Stored{Body: expr.Text{Value: body}})
	}

	return expr.Text{Value: strings.Join(names, "\n")}, nil
}

// historyStore type-asserts the evaluator's store to HistoryStore.
func historyStore(e *Evaluator) store.HistoryStore {
	if e.store == nil {
		return nil
	}
	hs, _ := e.store.(store.HistoryStore)
	return hs
}
