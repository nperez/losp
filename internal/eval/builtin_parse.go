// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"fmt"
	"strings"
	"unicode"

	"nickandperla.net/losp/internal/expr"
	"nickandperla.net/losp/internal/token"
)

// ParseReasonName is the namespace entry PARSE writes its explanation into.
// It is set on every PARSE call: the reason on FALSE, empty on TRUE.
const ParseReasonName = "PARSE_REASON"

// openScope records an operator that is waiting for its terminator.
type openScope struct {
	op   rune
	line int
}

// ValidateSyntax checks losp source text for structural errors without evaluating
// any of it. Nothing is stored, executed, or fired - immediate operators included.
// It returns the empty string when the text is well formed, otherwise a one-line
// explanation of the first problem found.
func ValidateSyntax(src string) string {
	runes := []rune(src)
	var stack []openScope
	line := 1

	// nameSlot reports the problem with the name that follows an operator at
	// runes[i:], and advances i past it. Store and execute operators accept a
	// computed name (dynamic naming), in which case the operator supplying it
	// is left for the main loop; a placeholder takes a literal name only.
	nameSlot := func(i int, op rune, dynamic bool) (int, string) {
		j := i
		for j < len(runes) && unicode.IsSpace(runes[j]) {
			if runes[j] == '\n' {
				line++
			}
			j++
		}
		if j >= len(runes) {
			return j, fmt.Sprintf("%s on line %d has no name", opName(op), line)
		}
		if dynamic {
			switch runes[j] {
			case token.RuneRetrieve, token.RuneImmRetrieve, token.RuneExecute, token.RuneImmExecute:
				// Computed name - leave the operator for the main loop.
				return j, ""
			}
		}
		if !isNameRune(runes[j]) {
			return j, fmt.Sprintf("%s on line %d has no name", opName(op), line)
		}
		for j < len(runes) && isNameRune(runes[j]) {
			j++
		}
		return j, ""
	}

	i := 0
	for i < len(runes) {
		r := runes[i]
		if r == '\n' {
			line++
			i++
			continue
		}
		if !token.IsOperator(r) {
			i++
			continue
		}

		switch r {
		case token.RuneStore, token.RuneImmStore, token.RuneExecute, token.RuneImmExecute:
			stack = append(stack, openScope{op: r, line: line})
			next, problem := nameSlot(i+1, r, true)
			if problem != "" {
				return problem
			}
			i = next

		case token.RuneRetrieve, token.RuneImmRetrieve:
			next, problem := nameSlot(i+1, r, true)
			if problem != "" {
				return problem
			}
			i = next

		case token.RunePlaceholder:
			next, problem := nameSlot(i+1, r, false)
			if problem != "" {
				return problem
			}
			i = next

		case token.RuneDefer:
			stack = append(stack, openScope{op: r, line: line})
			i++

		case token.RuneTerminator:
			// A terminator with nothing open closes nothing, and the evaluator
			// simply passes over it: ▼A first ◆ ◆ defines A, and a definition
			// that follows one still lands. Reporting it here would make PARSE
			// stricter than the language it checks, and would reject generated
			// code that runs correctly.
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			i++

		default:
			i++
		}
	}

	if len(stack) > 0 {
		last := stack[len(stack)-1]
		if len(stack) == 1 {
			return fmt.Sprintf("%s on line %d is missing its END terminator", opName(last.op), last.line)
		}
		// Naming only the last one leaves the rest to be hunted for, and an
		// answer that closes one operator comes back just as unfinished. Each
		// unclosed operator is listed, innermost first, so every one of them can
		// be given its terminator in a single pass.
		parts := make([]string, 0, len(stack))
		for i := len(stack) - 1; i >= 0; i-- {
			parts = append(parts, fmt.Sprintf("%s on line %d", opName(stack[i].op), stack[i].line))
		}
		return fmt.Sprintf("%d operators are missing their END terminator: %s",
			len(stack), strings.Join(parts, ", "))
	}

	return ""
}

// DefinesOnlyEmptyBody reports whether the source is a single definition with
// nothing in it, as in ▼Name ◆. Such an answer is structurally valid and passes
// PARSE, so nothing downstream can tell it apart from working code - it simply
// does nothing wherever it is called. GENERATE uses this to recognise a repair
// that reached validity by discarding the code it was asked to correct.
func DefinesOnlyEmptyBody(src string) bool {
	runes := []rune(strings.TrimSpace(src))
	i := 0
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	if i >= len(runes) || (runes[i] != token.RuneStore && runes[i] != token.RuneImmStore) {
		return false
	}
	i++
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	start := i
	for i < len(runes) && isNameRune(runes[i]) {
		i++
	}
	if i == start {
		return false
	}
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	// Everything the definition was asked to hold would sit here.
	return i < len(runes) && runes[i] == token.RuneTerminator && i == len(runes)-1
}

// SentinelAnonymous is the name reported for code that defines no expression,
// or that names one dynamically. It keeps DESCRIBE's output shape unconditional.
const SentinelAnonymous = "ANONYMOUS"

// FirstDefinedName returns the name of the first expression the source defines,
// or ANONYMOUS if it defines none. A dynamically named definition (▼▶Slot ◆ ◆)
// is ANONYMOUS too: its name is not knowable without evaluating, and nothing
// here evaluates. Like ValidateSyntax, this only walks the text.
func FirstDefinedName(src string) string {
	runes := []rune(src)
	for i, r := range runes {
		if r != token.RuneStore && r != token.RuneImmStore {
			continue
		}
		j := i + 1
		for j < len(runes) && unicode.IsSpace(runes[j]) {
			j++
		}
		start := j
		for j < len(runes) && isNameRune(runes[j]) {
			j++
		}
		if j > start {
			return string(runes[start:j])
		}
		return SentinelAnonymous
	}
	return SentinelAnonymous
}

// opName gives an operator its ASCII shorthand name. Reasons are read back with
// ▲PARSE_REASON, which re-parses them, so a reason must never contain an
// operator rune of its own.
func opName(r rune) string {
	switch r {
	case token.RuneStore:
		return "DEF"
	case token.RuneImmStore:
		return "IDEF"
	case token.RuneRetrieve:
		return "GET"
	case token.RuneImmRetrieve:
		return "IGET"
	case token.RuneExecute:
		return "RUN"
	case token.RuneImmExecute:
		return "IRUN"
	case token.RunePlaceholder:
		return "ARG"
	case token.RuneDefer:
		return "DEFER"
	}
	return "operator"
}

// NamedTarget returns the text as an expression name, or the empty string if the
// text is not a single name. It is applied to an argument that has already been
// evaluated: a bare word is its own name, and a retrieval is expected to yield
// one. Anything else is code rather than a reference to one expression.
func NamedTarget(text string) string {
	runes := []rune(strings.TrimSpace(text))
	i := 0
	for i < len(runes) && isNameRune(runes[i]) {
		i++
	}
	if i == 0 || i != len(runes) {
		return ""
	}
	return string(runes[:i])
}

// isNameRune reports whether r may appear in an expression name.
func isNameRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// builtinParse checks whether the body stored under a NAME is well-formed losp.
// It takes the name rather than the text because a text argument is itself
// scanned as losp: the very terminators PARSE exists to find would close PARSE's
// own argument early. The stored body is read straight out of the namespace and
// never evaluated.
func builtinParse(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	name := ""
	if len(args) > 0 {
		name = strings.TrimSpace(args[0])
	}
	if name == "" {
		return parseResult(e, false, "PARSE needs the name of an expression to check")
	}

	e.autoLoad(name)
	if !e.namespace.Has(name) {
		return parseResult(e, false, "there is no expression named "+name)
	}

	body := e.namespace.Get(name).String()
	if strings.TrimSpace(body) == "" {
		return parseResult(e, false, name+" is empty - there is no code to check")
	}

	if problem := ValidateSyntax(body); problem != "" {
		return parseResult(e, false, problem)
	}

	return parseResult(e, true, "")
}

// parseResult records the explanation in PARSE_REASON and returns TRUE or FALSE.
func parseResult(e *Evaluator, ok bool, reason string) (expr.Expr, error) {
	e.namespace.Set(ParseReasonName, expr.Stored{Body: reason})
	if ok {
		return expr.Stored{Body: "TRUE"}, nil
	}
	return expr.Stored{Body: "FALSE"}, nil
}
