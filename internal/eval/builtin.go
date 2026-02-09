package eval

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"nickandperla.net/losp/internal/expr"
	"nickandperla.net/losp/internal/stdlib"
	"nickandperla.net/losp/internal/token"
)

// BuiltinFunc is the signature for builtin functions.
type BuiltinFunc func(e *Evaluator, argsRaw string) (expr.Expr, error)

// getBuiltin returns the builtin function for the given name, or nil if not found.
func getBuiltin(name string) BuiltinFunc {
	switch name {
	case "TRUE":
		return builtinTrue
	case "FALSE":
		return builtinFalse
	case "EMPTY":
		return builtinEmpty
	case "IF":
		return builtinIf
	case "COMPARE":
		return builtinCompare
	case "FOREACH":
		return builtinForeach
	case "SAY":
		return builtinSay
	case "READ":
		return builtinRead
	case "COUNT":
		return builtinCount
	case "APPEND":
		return builtinAppend
	case "PERSIST":
		return builtinPersist
	case "LOAD":
		return builtinLoad
	case "PROMPT":
		return builtinPrompt
	case "EXTRACT":
		return builtinExtract
	case "SYSTEM":
		return builtinSystem
	case "UPPER":
		return builtinUpper
	case "LOWER":
		return builtinLower
	case "TRIM":
		return builtinTrim
	case "GENERATE":
		return builtinGenerate
	case "ASYNC":
		return builtinAsync
	case "AWAIT":
		return builtinAwait
	case "CHECK":
		return builtinCheck
	case "TIMER":
		return builtinTimer
	case "TICKS":
		return builtinTicks
	case "SLEEP":
		return builtinSleep
	case "CORPUS":
		return builtinCorpus
	case "ADD":
		return builtinAdd
	case "INDEX":
		return builtinIndex
	case "SEARCH":
		return builtinSearch
	case "EMBED":
		return builtinEmbed
	case "SIMILAR":
		return builtinSimilar
	case "HISTORY":
		return builtinHistory
	}
	return nil
}

func builtinTrue(e *Evaluator, argsRaw string) (expr.Expr, error) {
	return expr.Text{Value: "TRUE"}, nil
}

func builtinFalse(e *Evaluator, argsRaw string) (expr.Expr, error) {
	return expr.Text{Value: "FALSE"}, nil
}

func builtinEmpty(e *Evaluator, argsRaw string) (expr.Expr, error) {
	return expr.Empty{}, nil
}

func builtinIf(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// IF condition then-expr [else-expr]
	// Parse: first arg is condition, second is then, third is else
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if len(args) < 2 {
		return expr.Empty{}, nil
	}

	condition := args[0]
	thenExpr := args[1]
	elseExpr := ""
	if len(args) >= 3 {
		elseExpr = args[2]
	}

	// Evaluate condition
	condResult := strings.TrimSpace(condition)

	if condResult == "TRUE" {
		// Return then-expr as text (use dynamic execute to evaluate if needed)
		return expr.Text{Value: thenExpr}, nil
	} else {
		// Return else-expr as text
		if elseExpr == "" {
			return expr.Empty{}, nil
		}
		return expr.Text{Value: elseExpr}, nil
	}
}

func builtinCompare(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// COMPARE expects exactly two arguments (expressions)
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if len(args) < 2 {
		return expr.Text{Value: "FALSE"}, nil
	}

	if args[0] == args[1] {
		return expr.Text{Value: "TRUE"}, nil
	}
	return expr.Text{Value: "FALSE"}, nil
}

func builtinForeach(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// FOREACH items-expr body-name
	// Two expression arguments:
	//   1. items-expr - evaluates to text containing expressions (one per line or operator)
	//   2. body-name - text name of the expression to execute per item
	// The items text is re-parsed as expressions; each result is passed to body.
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	if len(args) < 2 {
		return expr.Empty{}, nil
	}

	// Re-parse the items text to expand embedded expressions
	itemsText := args[0]
	items, err := e.parseArgs(itemsText)
	if err != nil {
		return expr.Empty{}, nil
	}
	if len(items) == 0 {
		return expr.Empty{}, nil
	}

	// Second arg is the body expression name
	bodyName := args[1]

	// Get the body expression
	stored := e.namespace.Get(bodyName)
	if stored.IsEmpty() {
		return expr.Empty{}, nil
	}

	var results []string
	for _, item := range items {
		if s, ok := stored.(expr.Stored); ok {
			// Bind item to first parameter
			if len(s.Params) > 0 {
				e.namespace.Set(s.Params[0], expr.Text{Value: item})
			}
			result := mustEval(e, s.Body.String())
			results = append(results, result)
		}
	}

	return expr.Text{Value: strings.Join(results, "\n")}, nil
}

func builtinSay(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// Evaluate args
	result, err := e.Eval(argsRaw)
	if err != nil {
		return nil, err
	}

	text := strings.TrimSpace(result)
	if e.outputWriter != nil {
		e.outputWriter(text + "\n")
	}

	// Return empty - output already happened via outputWriter
	return expr.Empty{}, nil
}

func builtinRead(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// Parse args as expressions
	evaluated, err := e.Eval(argsRaw)
	if err != nil {
		return nil, err
	}
	prompt := strings.TrimSpace(evaluated)

	if e.inputReader == nil {
		return expr.Empty{}, nil
	}

	input, err := e.inputReader(prompt)
	if err != nil {
		return nil, err
	}

	return expr.Text{Value: strings.TrimSpace(input)}, nil
}

func builtinCount(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// Evaluate the expression
	result, err := e.Eval(argsRaw)
	if err != nil {
		return nil, err
	}

	text := strings.TrimSpace(result)
	if text == "" {
		return expr.Text{Value: "0"}, nil
	}

	lines := strings.Split(text, "\n")
	return expr.Text{Value: fmt.Sprintf("%d", len(lines))}, nil
}

func builtinAppend(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if len(args) < 2 {
		return expr.Empty{}, nil
	}

	name := args[0]
	content := strings.Join(args[1:], " ")

	// Get existing value
	existing := e.namespace.Get(name)
	var newValue string
	if !existing.IsEmpty() {
		newValue = existing.String() + "\n" + content
	} else {
		newValue = content
	}

	e.namespace.Set(name, expr.Text{Value: newValue})

	// Auto-persist in ALWAYS mode
	if e.persistMode == PersistAlways && e.store != nil {
		e.autoPersist(name)
	}

	return expr.Empty{}, nil
}

// formatAsDefinition generates the full losp source for an expression.
// For Stored expressions: ▼name □param1 □param2 body ◆
// For Text expressions: just the text value
// For other expressions: their String() representation
func formatAsDefinition(name string, val expr.Expr) string {
	if val == nil || val.IsEmpty() {
		return ""
	}

	stored, ok := val.(expr.Stored)
	if !ok {
		// Not a stored expression, just return the value
		return val.String()
	}

	// Build the full definition: ▼name □param1 □param2 body ◆
	var sb strings.Builder
	sb.WriteRune(token.RuneStore) // ▼
	sb.WriteString(name)
	sb.WriteString(" ")

	// Add placeholders
	for _, param := range stored.Params {
		sb.WriteRune(token.RunePlaceholder) // □
		sb.WriteString(param)
		sb.WriteString(" ")
	}

	// Add body
	if stored.Body != nil {
		sb.WriteString(stored.Body.String())
	}

	sb.WriteString(" ")
	sb.WriteRune(token.RuneTerminator) // ◆

	return sb.String()
}

func builtinPersist(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// In NEVER or ALWAYS mode, PERSIST is a no-op
	if e.PersistMode() == PersistNever || e.PersistMode() == PersistAlways {
		return expr.Empty{}, nil
	}

	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if len(args) < 1 {
		return expr.Empty{}, nil
	}

	name := args[0]

	if e.store == nil {
		return expr.Empty{}, nil
	}

	val := e.namespace.Get(name)

	// Format as full definition so we can reconstruct on LOAD
	fullDef := formatAsDefinition(name, val)
	if err := e.store.Put(name, expr.Text{Value: fullDef}); err != nil {
		return nil, err
	}

	return expr.Empty{}, nil
}

func builtinLoad(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// LOAD name [default]
	// Loads name from store. If not found/empty and default provided, uses default.
	// If the stored value is a full definition (starts with ▼), re-evals it to
	// reconstruct the Stored expression with its parameters.
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if len(args) < 1 {
		return expr.Empty{}, nil
	}

	name := args[0]
	var defaultVal string
	if len(args) >= 2 {
		defaultVal = args[1]
	}

	// Try loading from store
	var val expr.Expr
	if e.store != nil {
		val, err = e.store.Get(name)
		if err != nil {
			return nil, err
		}
	}

	// If we got a value from store, process it
	if val != nil && !val.IsEmpty() {
		text := val.String()

		// Check if it's a full definition (starts with ▼)
		// If so, re-eval to reconstruct the Stored expression
		trimmed := strings.TrimSpace(text)
		runes := []rune(trimmed)
		if len(runes) > 0 && runes[0] == token.RuneStore {
			// Re-eval the definition - this will store it in namespace
			_, err := e.Eval(text)
			if err != nil {
				return nil, err
			}
		} else {
			// Plain text value, just set it directly
			e.namespace.Set(name, expr.Text{Value: text})
		}
		return expr.Empty{}, nil
	}

	// Otherwise use default if provided
	if defaultVal != "" {
		e.namespace.Set(name, expr.Text{Value: defaultVal})
	}

	return expr.Empty{}, nil
}

func builtinExtract(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// EXTRACT label source
	// Parses source for "LABEL: value" format and returns the value
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if len(args) < 2 {
		return expr.Empty{}, nil
	}

	label := strings.ToUpper(strings.TrimSpace(args[0]))
	source := args[1]

	// Parse line by line looking for LABEL: value
	lines := strings.Split(source, "\n")
	var result strings.Builder
	capturing := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this line starts a new label
		if colonIdx := strings.Index(trimmed, ":"); colonIdx > 0 {
			potentialLabel := strings.ToUpper(strings.TrimSpace(trimmed[:colonIdx]))
			// Check if it looks like a label (alphanumeric/underscore only)
			isLabel := true
			for _, r := range potentialLabel {
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
					isLabel = false
					break
				}
			}

			if isLabel {
				if potentialLabel == label {
					// Found our label, start capturing
					capturing = true
					value := strings.TrimSpace(trimmed[colonIdx+1:])
					if value != "" {
						result.WriteString(value)
					}
					continue
				} else if capturing {
					// Hit a different label, stop capturing
					break
				}
			}
		}

		// If we're capturing, append this line (it's a continuation)
		if capturing {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(trimmed)
		}
	}

	extracted := strings.TrimSpace(result.String())
	if extracted == "" {
		return expr.Empty{}, nil
	}
	return expr.Text{Value: extracted}, nil
}

func builtinPrompt(e *Evaluator, argsRaw string) (expr.Expr, error) {
	if e.provider == nil {
		return expr.Empty{}, nil
	}

	// Evaluate args to resolve any operators (like ▲)
	evaluated, err := e.Eval(argsRaw)
	if err != nil {
		return nil, err
	}

	// Parse args: system-prompt user-prompt
	// Split on double newline or use first line as system
	text := strings.TrimSpace(evaluated)

	parts := strings.SplitN(text, "\n", 2)

	var system, user string
	if len(parts) == 1 {
		user = parts[0]
	} else {
		system = strings.TrimSpace(parts[0])
		user = strings.TrimSpace(parts[1])
	}

	response, err := e.provider.Prompt(system, user)
	if err != nil {
		return nil, err
	}

	return expr.Text{Value: response}, nil
}

func builtinSystem(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// SYSTEM setting [value]
	// With one arg: returns current value
	// With two args: sets new value
	// Arguments are expressions — newlines or operators separate them
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return expr.Empty{}, nil
	}

	setting := strings.ToUpper(strings.TrimSpace(args[0]))
	var value string
	if len(args) >= 2 {
		value = strings.TrimSpace(args[1])
	}

	switch setting {
	case "PERSIST_MODE":
		if value != "" {
			mode, ok := ParsePersistMode(value)
			if !ok {
				return expr.Text{Value: "UNKNOWN"}, nil
			}
			e.SetPersistMode(mode)
			return expr.Empty{}, nil
		}
		return expr.Text{Value: e.PersistMode().String()}, nil

	case "MODEL":
		if cfg, ok := e.provider.(Configurable); ok {
			if value != "" {
				cfg.SetModel(value)
				return expr.Empty{}, nil
			}
			return expr.Text{Value: cfg.GetModel()}, nil
		}
		return expr.Empty{}, nil

	case "PROVIDER":
		if value != "" {
			name := strings.ToUpper(value)
			factory, ok := e.providerFactories[name]
			if !ok {
				return expr.Text{Value: "UNKNOWN_PROVIDER"}, nil
			}
			// Copy inference params from old provider to new one
			var oldParams map[string]string
			if cfg, ok := e.provider.(Configurable); ok {
				for _, key := range []string{"TEMPERATURE", "NUM_CTX", "TOP_K", "TOP_P", "MAX_TOKENS"} {
					if v := cfg.GetParam(key); v != "" {
						if oldParams == nil {
							oldParams = make(map[string]string)
						}
						oldParams[key] = v
					}
				}
			}
			newProvider := factory(e.streamCb)
			if cfg, ok := newProvider.(Configurable); ok && oldParams != nil {
				for k, v := range oldParams {
					cfg.SetParam(k, v)
				}
			}
			e.provider = newProvider
			return expr.Empty{}, nil
		}
		// Get current provider name
		if cfg, ok := e.provider.(Configurable); ok {
			return expr.Text{Value: cfg.ProviderName()}, nil
		}
		return expr.Empty{}, nil

	case "TEMPERATURE", "NUM_CTX", "TOP_K", "TOP_P", "MAX_TOKENS", "EMBED_MODEL":
		if cfg, ok := e.provider.(Configurable); ok {
			if value != "" {
				cfg.SetParam(setting, value)
				return expr.Empty{}, nil
			}
			return expr.Text{Value: cfg.GetParam(setting)}, nil
		}
		return expr.Empty{}, nil

	case "SEARCH_LIMIT":
		if value != "" {
			e.SetSetting("SEARCH_LIMIT", value)
			return expr.Empty{}, nil
		}
		return expr.Text{Value: e.GetSetting("SEARCH_LIMIT", "10")}, nil

	case "HISTORY_LIMIT":
		if value != "" {
			n, err := strconv.Atoi(value)
			if err != nil {
				return expr.Text{Value: "INVALID"}, nil
			}
			e.historyLimit = n
			return expr.Empty{}, nil
		}
		return expr.Text{Value: strconv.Itoa(e.historyLimit)}, nil

	default:
		return expr.Text{Value: "UNKNOWN_SETTING"}, nil
	}
}

func builtinUpper(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if len(args) == 0 {
		return expr.Empty{}, nil
	}

	var results []string
	for _, arg := range args {
		results = append(results, strings.ToUpper(arg))
	}

	return expr.Text{Value: strings.Join(results, "\n")}, nil
}

func builtinLower(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if len(args) == 0 {
		return expr.Empty{}, nil
	}

	var results []string
	for _, arg := range args {
		results = append(results, strings.ToLower(arg))
	}

	return expr.Text{Value: strings.Join(results, "\n")}, nil
}

func builtinTrim(e *Evaluator, argsRaw string) (expr.Expr, error) {
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if len(args) == 0 {
		return expr.Empty{}, nil
	}

	var results []string
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed != "" {
			results = append(results, trimmed)
		}
	}

	if len(results) == 0 {
		return expr.Empty{}, nil
	}

	return expr.Text{Value: strings.Join(results, "\n")}, nil
}

func builtinGenerate(e *Evaluator, argsRaw string) (expr.Expr, error) {
	if e.provider == nil {
		return expr.Empty{}, nil
	}

	evaluated, err := e.Eval(argsRaw)
	if err != nil {
		return nil, err
	}
	request := strings.TrimSpace(evaluated)
	if request == "" {
		return expr.Empty{}, nil
	}

	// Use compact primer to fit within model context limits
	system := stdlib.PrimerCompact
	user := request + "\n\nOutput ONLY losp code. No markdown. No explanation."

	response, err := e.provider.Prompt(system, user)
	if err != nil {
		return nil, err
	}

	return expr.Text{Value: strings.TrimSpace(response)}, nil
}
