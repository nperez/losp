package eval

import (
	"fmt"
	"io"
	"strings"

	"nickandperla.net/losp/internal/expr"
	"nickandperla.net/losp/internal/scanner"
	"nickandperla.net/losp/internal/token"
)

// Store is the interface for expression persistence.
type Store interface {
	Get(name string) (expr.Expr, error)
	Put(name string, e expr.Expr) error
	Delete(name string) error
	Close() error
}

// MetadataStore extends Store with metadata operations.
type MetadataStore interface {
	Store
	GetMetadata(key string) (string, error)
	SetMetadata(key, value string) error
}

// PersistMode controls when expressions are persisted.
type PersistMode int

const (
	// PersistOnDemand is the default - explicit PERSIST/LOAD calls only.
	PersistOnDemand PersistMode = iota
	// PersistAlways auto-persists on every store, auto-loads on every retrieve.
	PersistAlways
	// PersistNever makes PERSIST a no-op (memory-only mode).
	PersistNever
)

// String returns the string representation of a PersistMode.
func (m PersistMode) String() string {
	switch m {
	case PersistOnDemand:
		return "ON_DEMAND"
	case PersistAlways:
		return "ALWAYS"
	case PersistNever:
		return "NEVER"
	default:
		return "UNKNOWN"
	}
}

// ParsePersistMode parses a string into a PersistMode.
func ParsePersistMode(s string) (PersistMode, bool) {
	switch strings.ToUpper(s) {
	case "ON_DEMAND":
		return PersistOnDemand, true
	case "ALWAYS":
		return PersistAlways, true
	case "NEVER":
		return PersistNever, true
	default:
		return PersistOnDemand, false
	}
}

// Provider is the interface for LLM providers.
type Provider interface {
	Prompt(system, user string) (string, error)
}

// Configurable allows getting/setting inference parameters at runtime.
type Configurable interface {
	GetParam(key string) string
	SetParam(key string, value string)
	GetModel() string
	SetModel(model string)
	ProviderName() string
}

// ProviderFactory creates a new provider with the given stream callback.
type ProviderFactory func(streamCb StreamCallback) Provider

// StreamCallback is called with streaming LLM output.
type StreamCallback func(token string)

// InputReader reads user input.
type InputReader func(prompt string) (string, error)

// OutputWriter writes output (for SAY builtin).
type OutputWriter func(text string) error

// Evaluator interprets losp expressions.
type Evaluator struct {
	namespace         *Namespace
	store             Store
	provider          Provider
	streamCb          StreamCallback
	inputReader       InputReader
	outputWriter      OutputWriter
	deferDepth        int            // Tracks ◯ defer operator depth
	persistMode       PersistMode    // Controls persistence behavior
	loadOnly          bool
	asyncRegistry     *AsyncRegistry
	corpusRegistry    *CorpusRegistry
	providerFactories map[string]ProviderFactory
	settings          map[string]string // Runtime settings (SEARCH_LIMIT, etc.)
	historyLimit      int               // Limit for HISTORY queries (0 = all)
}

// Option configures an Evaluator.
type Option func(*Evaluator)

// WithStore sets the persistence store.
func WithStore(s Store) Option {
	return func(e *Evaluator) { e.store = s }
}

// WithProvider sets the LLM provider.
func WithProvider(p Provider) Option {
	return func(e *Evaluator) { e.provider = p }
}

// WithStreamCallback sets the streaming callback.
func WithStreamCallback(cb StreamCallback) Option {
	return func(e *Evaluator) { e.streamCb = cb }
}

// WithInputReader sets the input reader for READ builtin.
func WithInputReader(r InputReader) Option {
	return func(e *Evaluator) { e.inputReader = r }
}

// WithOutputWriter sets the output writer for SAY builtin.
func WithOutputWriter(w OutputWriter) Option {
	return func(e *Evaluator) { e.outputWriter = w }
}

// WithPersistMode sets the persistence mode.
func WithPersistMode(mode PersistMode) Option {
	return func(e *Evaluator) { e.persistMode = mode }
}

// SetInputReader changes the input reader for READ builtin.
func (e *Evaluator) SetInputReader(r InputReader) {
	e.inputReader = r
}

// New creates a new Evaluator with the given options.
func New(opts ...Option) *Evaluator {
	e := &Evaluator{
		namespace:         NewNamespace(),
		asyncRegistry:     NewAsyncRegistry(),
		corpusRegistry:    NewCorpusRegistry(),
		providerFactories: make(map[string]ProviderFactory),
		settings:          make(map[string]string),
		outputWriter: func(text string) error {
			fmt.Print(text)
			return nil
		},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// SetProvider sets the LLM provider at runtime.
func (e *Evaluator) SetProvider(p Provider) {
	e.provider = p
}

// RegisterProviderFactory registers a factory for creating providers by name.
func (e *Evaluator) RegisterProviderFactory(name string, f ProviderFactory) {
	e.providerFactories[name] = f
}

// forkForAsync creates a new Evaluator for async execution.
// The forked evaluator has a cloned namespace (snapshot isolation),
// shared store, provider, and async registry, but nil I/O.
func (e *Evaluator) forkForAsync() *Evaluator {
	return &Evaluator{
		namespace:         e.namespace.Clone(),
		store:             e.store,
		provider:          e.provider,
		asyncRegistry:     e.asyncRegistry,
		corpusRegistry:    e.corpusRegistry,
		persistMode:       e.persistMode,
		providerFactories: e.providerFactories,
		settings:          e.settings,
		historyLimit:      e.historyLimit,
		// inputReader, outputWriter, streamCb are nil (SAY silenced, READ returns EMPTY)
	}
}

// Eval evaluates a losp string and returns the result.
func (e *Evaluator) Eval(input string) (string, error) {
	return e.EvalReader(strings.NewReader(input))
}

// EvalReader evaluates losp from a reader.
func (e *Evaluator) EvalReader(r io.Reader) (string, error) {
	scan := scanner.New(r)
	result, err := e.evalStream(scan, false)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.String()), nil
}

// LoadReader loads definitions from a reader without executing top-level code.
// Only ▼ (store) operators are processed; ▶ (execute) at top level is ignored.
func (e *Evaluator) LoadReader(r io.Reader) error {
	e.loadOnly = true
	defer func() { e.loadOnly = false }()
	scan := scanner.New(r)
	_, err := e.evalStream(scan, false)
	return err
}

// evalStream processes the input stream, returning the last non-empty result.
func (e *Evaluator) evalStream(scan *scanner.Scanner, stopAtTerminator bool) (expr.Expr, error) {
	var results []expr.Expr

	for {
		item, err := scan.Next()
		if err != nil {
			return nil, err
		}

		switch item.Token {
		case token.EOF:
			return e.concatResults(results), nil

		case token.TERMINATOR:
			if stopAtTerminator {
				return e.concatResults(results), nil
			}
			// Stray terminator, ignore

		case token.TEXT:
			if item.Value != "" {
				results = append(results, expr.Text{Value: item.Value})
			}

		case token.DEFER:
			if e.deferDepth > 0 {
				// Already inside a ◯: preserve this ◯ as text for later consumption
				e.deferDepth++
				result, err := e.evalStream(scan, true)
				e.deferDepth--
				if err != nil {
					return nil, err
				}
				// Reconstruct the ◯ wrapper so it survives to the next parse
				results = append(results, expr.Text{Value: string(token.RuneDefer) + result.String() + string(token.RuneTerminator)})
			} else {
				// At top level: ◯ is CONSUMED - only the deferred content is returned
				e.deferDepth++
				result, err := e.evalStream(scan, true)
				e.deferDepth--
				if err != nil {
					return nil, err
				}
				results = append(results, result)
			}

		case token.PLACEHOLDER:
			name, err := scan.ScanName()
			if err != nil {
				return nil, err
			}
			results = append(results, expr.Placeholder{Name: name})

		case token.STORE, token.IMM_STORE:
			// Check for dynamic naming (△name or ▷expr resolves to name)
			name, err := e.scanNameOrDynamic(scan)
			if err != nil {
				return nil, err
			}

			if item.Token == token.IMM_STORE {
				// ▽ - Immediate store: scan body preserving ◯ for Eval to handle
				body, err := e.scanBodyPreservingDefer(scan)
				if err != nil {
					return nil, err
				}

				if e.deferDepth == 0 {
					// Evaluate body now, store result
					evaluated, err := e.Eval(body)
					if err != nil {
						return nil, err
					}
					e.namespace.Set(name, expr.Text{Value: evaluated})
				} else {
					// Inside ◯: store as deferred expression
					bodyExpr, params, err := e.parseBody(body)
					if err != nil {
						return nil, err
					}
					e.namespace.Set(name, expr.Stored{Params: params, Body: bodyExpr})
				}
			} else {
				// ▼ - Deferred store: process body, evaluating immediate operators
				// CRITICAL: Immediate operators (△, ▷, ▽) fire NOW during body collection
				body, params, err := e.evalBodyForDeferredStore(scan, "▼"+name, item.Line)
				if err != nil {
					return nil, err
				}
				stored := expr.Stored{Params: params, Body: expr.Text{Value: body}}
				e.namespace.Set(name, stored)
			}

			// Auto-persist in ALWAYS mode
			if e.persistMode == PersistAlways && e.store != nil {
				e.autoPersist(name)
			}
			results = append(results, expr.Empty{})

		case token.RETRIEVE, token.IMM_RETRIEVE:
			// Support dynamic naming (▲varname or △varname resolves to actual variable name)
			name, err := e.scanNameOrDynamic(scan)
			if err != nil {
				return nil, err
			}

			if item.Token == token.RETRIEVE {
				// ▲ - DEFERRED retrieve: operates at EXECUTE time
				// Only immediate operators fire; deferred operators are preserved
				val := e.namespace.Get(name)
				result, err := e.parseBodyImmediateOnly(val.String())
				if err != nil {
					return nil, err
				}

				// Update stored body with parsed result (ephemeral semantic)
				if s, ok := val.(expr.Stored); ok {
					e.namespace.Set(name, expr.Stored{Params: s.Params, Body: expr.Text{Value: result}})
				} else {
					e.namespace.Set(name, expr.Text{Value: result})
				}

				results = append(results, expr.Text{Value: result})
			} else if e.deferDepth == 0 {
				// △ - IMMEDIATE retrieve at parse time: only immediate ops fire
				val := e.namespace.Get(name)
				result, err := e.parseBodyImmediateOnly(val.String())
				if err != nil {
					return nil, err
				}

				// Update stored body - any immediate ops that fired are now replaced
				if s, ok := val.(expr.Stored); ok {
					e.namespace.Set(name, expr.Stored{Params: s.Params, Body: expr.Text{Value: result}})
				} else {
					e.namespace.Set(name, expr.Text{Value: result})
				}

				results = append(results, expr.Text{Value: result})
			} else {
				// △ inside ◯ - return the operator itself
				results = append(results, expr.Operator{
					Op:   item.Token,
					Name: name,
				})
			}

		case token.EXECUTE, token.IMM_EXECUTE:
			name, err := e.scanNameOrDynamic(scan)
			if err != nil {
				return nil, err
			}
			argsRaw, err := scan.ScanUntilTerminator()
			if err != nil {
				return nil, err
			}

			if (item.Token == token.IMM_EXECUTE && e.deferDepth == 0) || item.Token == token.EXECUTE {
				result, err := e.execute(name, argsRaw)
				if err != nil {
					return nil, err
				}
				// For immediate execute (▷), re-evaluate result to splice losp operators into stream
				if item.Token == token.IMM_EXECUTE {
					evaluated, err := e.Eval(result.String())
					if err != nil {
						return nil, err
					}
					results = append(results, expr.Text{Value: evaluated})
				} else {
					results = append(results, result)
				}
			} else {
				// Deferred - return the operator itself
				results = append(results, expr.Operator{
					Op:   item.Token,
					Name: name,
					Body: expr.Text{Value: argsRaw},
				})
			}
		}
	}
}

// parseBody parses a body string for immediate store (▽), extracting placeholders.
// All operators are preserved as text since the body will be evaluated immediately.
func (e *Evaluator) parseBody(body string) (expr.Expr, []string, error) {
	scan := scanner.NewFromString(body)
	var exprs []expr.Expr
	var params []string

	for {
		item, err := scan.Next()
		if err != nil {
			return nil, nil, err
		}
		if item.Token == token.EOF {
			break
		}

		if item.Token == token.PLACEHOLDER {
			name, err := scan.ScanName()
			if err != nil {
				return nil, nil, err
			}
			params = append(params, name)
			// Don't add placeholder to body - it's just a parameter declaration
		} else if item.Token == token.TEXT {
			exprs = append(exprs, expr.Text{Value: item.Value})
		} else if item.Token.NeedsTerminator() {
			// Operators with terminators (▼, ▽, ▶, ▷): preserve full syntax
			// including name, body, and terminator
			name, err := scan.ScanName()
			if err != nil {
				return nil, nil, err
			}
			body, err := scan.ScanUntilTerminator()
			if err != nil {
				return nil, nil, err
			}
			// Reconstruct the full operator syntax as text
			fullOp := item.Value + name + body + string(token.RuneTerminator)
			exprs = append(exprs, expr.Text{Value: fullOp})
		} else if item.Token == token.RETRIEVE || item.Token == token.IMM_RETRIEVE {
			// Retrieve operators have a name but no terminator
			name, err := scan.ScanName()
			if err != nil {
				return nil, nil, err
			}
			fullOp := item.Value + name
			exprs = append(exprs, expr.Text{Value: fullOp})
		} else {
			// For other operators (DEFER, TERMINATOR), include just the rune
			exprs = append(exprs, expr.Text{Value: item.Value})
		}
	}

	return expr.NewCompound(exprs...), params, nil
}

// evalBodyForDeferredStore processes the body of a ▼ (deferred store) operation.
// CRITICAL: Immediate operators (△, ▷, ▽) are evaluated immediately as they are encountered.
// Deferred operators (▲, ▶, ▼) are preserved as text for later execution.
// This implements the core losp semantic: immediate operators ALWAYS evaluate immediately.
// The opName and startLine parameters are used for error reporting.
func (e *Evaluator) evalBodyForDeferredStore(scan *scanner.Scanner, opName string, startLine int) (string, []string, error) {
	var parts []string
	var params []string

	for {
		item, err := scan.Next()
		if err != nil {
			return "", nil, err
		}

		switch item.Token {
		case token.EOF:
			return "", nil, fmt.Errorf("unexpected EOF at line %d: unterminated %s starting at line %d", scan.Line(), opName, startLine)

		case token.TERMINATOR:
			return strings.Join(parts, ""), params, nil

		case token.TEXT:
			parts = append(parts, item.Value)

		case token.PLACEHOLDER:
			name, err := scan.ScanName()
			if err != nil {
				return "", nil, err
			}
			params = append(params, name)
			// Placeholder is just a declaration, not included in body

		case token.DEFER:
			// ◯ - defer ALL immediate operators until its own terminating ◆
			if e.deferDepth == 0 {
				// At top level: ◯ is CONSUMED - only the deferred content is stored
				e.deferDepth++
				deferredPart, deferredParams, err := e.evalBodyForDeferredStore(scan, "◯ (defer)", item.Line)
				e.deferDepth--
				if err != nil {
					return "", nil, fmt.Errorf("in %s starting at line %d: %w", opName, startLine, err)
				}
				params = append(params, deferredParams...)
				parts = append(parts, deferredPart)
			} else {
				// Already inside a ◯: preserve this ◯ as text for later consumption
				e.deferDepth++
				deferredPart, deferredParams, err := e.evalBodyForDeferredStore(scan, "◯ (defer)", item.Line)
				e.deferDepth--
				if err != nil {
					return "", nil, fmt.Errorf("in %s starting at line %d: %w", opName, startLine, err)
				}
				params = append(params, deferredParams...)
				// Preserve ◯ and its terminator
				parts = append(parts, string(token.RuneDefer)+deferredPart+string(token.RuneTerminator))
			}

		case token.IMM_RETRIEVE:
			// △ - immediate retrieve
			if e.deferDepth == 0 {
				// Evaluate immediately - this is the key fix!
				name, err := e.scanNameOrDynamic(scan)
				if err != nil {
					return "", nil, err
				}
				val := e.namespace.Get(name)
				result, err := e.Eval(val.String())
				if err != nil {
					return "", nil, err
				}
				parts = append(parts, result)
			} else {
				// Inside ◯, preserve as text (including any dynamic name operators)
				nameText, err := e.scanNamePreservingOperators(scan)
				if err != nil {
					return "", nil, err
				}
				parts = append(parts, string(token.RuneImmRetrieve)+nameText)
			}

		case token.RETRIEVE:
			// ▲ - deferred retrieve, always preserve as text
			name, err := scan.ScanName()
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, string(token.RuneRetrieve)+name)

		case token.IMM_EXECUTE:
			// ▷ - immediate execute
			if e.deferDepth == 0 {
				// Execute immediately - this is the key fix!
				name, err := e.scanNameOrDynamic(scan)
				if err != nil {
					return "", nil, err
				}
				argsRaw, err := scan.ScanUntilTerminator()
				if err != nil {
					return "", nil, err
				}
				result, err := e.execute(name, argsRaw)
				if err != nil {
					return "", nil, err
				}
				// Re-evaluate result to splice any losp operators into the stream
				evaluated, err := e.Eval(result.String())
				if err != nil {
					return "", nil, err
				}
				parts = append(parts, evaluated)
			} else {
				// Inside ◯, preserve as text (including any dynamic name operators)
				nameText, err := e.scanNamePreservingOperators(scan)
				if err != nil {
					return "", nil, err
				}
				argsRaw, err := scan.ScanUntilTerminator()
				if err != nil {
					return "", nil, err
				}
				parts = append(parts, string(token.RuneImmExecute)+nameText+argsRaw+string(token.RuneTerminator))
			}

		case token.EXECUTE:
			// ▶ - deferred execute, always preserve as text
			// BUT we need to properly track nested terminators (e.g., inside ◯ or other operators)
			// Use scanNamePreservingOperators to preserve any dynamic naming for later
			nameText, err := e.scanNamePreservingOperators(scan)
			if err != nil {
				return "", nil, err
			}
			body, err := e.scanBodyWithNestedTerminators(scan)
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, string(token.RuneExecute)+nameText+body+string(token.RuneTerminator))

		case token.IMM_STORE:
			// ▽ - immediate store
			if e.deferDepth == 0 {
				// Evaluate and store immediately - this is the key fix!
				name, err := e.scanNameOrDynamic(scan)
				if err != nil {
					return "", nil, err
				}
				body, err := scan.ScanUntilTerminator()
				if err != nil {
					return "", nil, err
				}
				evaluated, err := e.Eval(body)
				if err != nil {
					return "", nil, err
				}
				e.namespace.Set(name, expr.Text{Value: evaluated})
				// Auto-persist in ALWAYS mode
				if e.persistMode == PersistAlways && e.store != nil {
					e.autoPersist(name)
				}
				// Immediate store produces no output in the body
			} else {
				// Inside ◯, preserve as text (including any dynamic name operators)
				nameText, err := e.scanNamePreservingOperators(scan)
				if err != nil {
					return "", nil, err
				}
				// Must use scanBodyWithNestedTerminators to properly track nested operators
				body, err := e.scanBodyWithNestedTerminators(scan)
				if err != nil {
					return "", nil, err
				}
				parts = append(parts, string(token.RuneImmStore)+nameText+body+string(token.RuneTerminator))
			}

		case token.STORE:
			// ▼ - deferred store (nested), preserve as text including dynamic name
			// Dynamic naming (▼▲name value ◆) should be resolved at execution time,
			// not at definition time, so we preserve the ▲ operator.
			nestedLine := item.Line
			nameText, err := e.scanNamePreservingOperators(scan)
			if err != nil {
				return "", nil, err
			}
			nestedBody, nestedParams, err := e.evalBodyForDeferredStore(scan, "▼"+nameText, nestedLine)
			if err != nil {
				return "", nil, fmt.Errorf("in %s starting at line %d: %w", opName, startLine, err)
			}
			// Reconstruct placeholder declarations so they survive re-parsing
			var placeholders string
			for _, p := range nestedParams {
				placeholders += string(token.RunePlaceholder) + p + " "
			}
			parts = append(parts, string(token.RuneStore)+nameText+" "+placeholders+nestedBody+string(token.RuneTerminator))
		}
	}
}

// scanBodyPreservingDefer scans until terminator at depth 0, preserving ◯ in the output.
// Use this for ▽ body scanning where the body will be passed to Eval (which handles ◯).
func (e *Evaluator) scanBodyPreservingDefer(scan *scanner.Scanner) (string, error) {
	var result strings.Builder
	depth := 0

	for {
		item, err := scan.Next()
		if err != nil {
			return "", err
		}

		switch item.Token {
		case token.EOF:
			return "", fmt.Errorf("unexpected EOF while scanning body")

		case token.TERMINATOR:
			if depth == 0 {
				return result.String(), nil
			}
			result.WriteRune(token.RuneTerminator)
			depth--

		case token.DEFER:
			// ◯ is PRESERVED - include it and track its terminator
			result.WriteString(item.Value)
			depth++

		case token.STORE, token.IMM_STORE, token.EXECUTE, token.IMM_EXECUTE:
			result.WriteString(item.Value)
			depth++

		default:
			result.WriteString(item.Value)
		}
	}
}

// scanBodyWithNestedTerminators scans until we find a terminator at depth 0.
// It tracks operators that introduce terminators: ▼, ▽, ▶, ▷, ◯
// All operators are PRESERVED as text - consumption is handled by the parsing functions.
func (e *Evaluator) scanBodyWithNestedTerminators(scan *scanner.Scanner) (string, error) {
	var result strings.Builder
	depth := 0

	for {
		item, err := scan.Next()
		if err != nil {
			return "", err
		}

		switch item.Token {
		case token.EOF:
			return "", fmt.Errorf("unexpected EOF while scanning body")

		case token.TERMINATOR:
			if depth == 0 {
				// This is our terminator
				return result.String(), nil
			}
			// Nested terminator - include it and decrease depth
			result.WriteRune(token.RuneTerminator)
			depth--

		case token.DEFER:
			// ◯ is PRESERVED as text - track its terminator scope
			result.WriteString(item.Value)
			depth++

		case token.STORE, token.IMM_STORE, token.EXECUTE, token.IMM_EXECUTE:
			// These operators introduce their own terminator scope
			result.WriteString(item.Value)
			depth++

		default:
			// Everything else is just added to the result
			result.WriteString(item.Value)
		}
	}
}

// execute runs a builtin or stored expression.
// Per PRIMER.md, execution follows four phases:
// 1. LOAD - body text is retrieved from the namespace
// 2. PARSE - immediate operators fire (parse-time effects)
// 3. POPULATE - placeholders are bound to arguments
// 4. EXECUTE - deferred expressions run
func (e *Evaluator) execute(name string, argsRaw string) (expr.Expr, error) {
	// Check for builtin first (exact case match — builtins are ALL CAPS)
	if builtin := getBuiltin(name); builtin != nil {
		return builtin(e, argsRaw)
	}

	// 1. LOAD - look up stored expression
	stored := e.namespace.Get(name)
	if stored.IsEmpty() {
		return expr.Empty{}, nil
	}

	// Parse arguments (needed for POPULATE phase)
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if s, ok := stored.(expr.Stored); ok {
		// 2. PARSE - fire immediate operators BEFORE binding placeholders
		// This is critical: per PRIMER.md, immediate operators fire at PARSE time,
		// which is BEFORE placeholders are bound at POPULATE time.
		parsedBody, err := e.parseBodyImmediateOnly(s.Body.String())
		if err != nil {
			return nil, err
		}

		// EPHEMERAL: Update stored body - immediate operators are consumed
		// The body now contains only what remains after immediate operators fired
		e.namespace.Set(name, expr.Stored{Params: s.Params, Body: expr.Text{Value: parsedBody}})

		// 3. POPULATE - bind arguments to placeholders
		for i, param := range s.Params {
			if i < len(args) {
				e.namespace.Set(param, expr.Text{Value: args[i]})
			}
		}

		// 4. EXECUTE - evaluate the body (deferred operators run now)
		return expr.Text{Value: mustEval(e, parsedBody)}, nil
	}

	// If it's just text, return it
	return stored, nil
}

// parseBodyImmediateOnly processes a body string, firing immediate operators
// but preserving deferred operators as text.
// This implements the PARSE phase per PRIMER.md, where immediate operators
// fire BEFORE placeholders are bound.
func (e *Evaluator) parseBodyImmediateOnly(body string) (string, error) {
	scan := scanner.NewFromString(body)
	var parts []string

	for {
		item, err := scan.Next()
		if err != nil {
			return "", err
		}

		switch item.Token {
		case token.EOF:
			return strings.Join(parts, ""), nil

		case token.TERMINATOR:
			// Stray terminator - shouldn't happen in well-formed body
			parts = append(parts, string(token.RuneTerminator))

		case token.TEXT:
			parts = append(parts, item.Value)

		case token.PLACEHOLDER:
			// Placeholder declarations shouldn't be in body (extracted during store)
			// but if present, preserve as text
			name, _ := scan.ScanName()
			parts = append(parts, string(token.RunePlaceholder)+name)

		case token.DEFER:
			// ◯ - defer operator: consume this ◯, increment deferDepth so immediate ops inside are preserved
			// Per CLAUDE.md: when ◯ is consumed, "Immediate operators inside are NOT fired (deferDepth > 0)"
			bodyContent, err := e.scanBodyWithNestedTerminators(scan)
			if err != nil {
				return "", err
			}
			// Increment deferDepth so immediate operators inside are preserved, not fired
			e.deferDepth++
			parsedContent, err := e.parseBodyImmediateOnly(bodyContent)
			e.deferDepth--
			if err != nil {
				return "", err
			}
			parts = append(parts, parsedContent)

		case token.IMM_RETRIEVE:
			// △ - immediate retrieve: fire NOW (before placeholders bound)
			if e.deferDepth == 0 {
				name, err := e.scanNameOrDynamic(scan)
				if err != nil {
					return "", err
				}
				val := e.namespace.Get(name)
				result, err := e.Eval(val.String())
				if err != nil {
					return "", err
				}
				parts = append(parts, result)
			} else {
				// Inside ◯, preserve as text
				nameText, _ := e.scanNamePreservingOperators(scan)
				parts = append(parts, string(token.RuneImmRetrieve)+nameText)
			}

		case token.RETRIEVE:
			// ▲ - deferred retrieve: preserve as text for EXECUTE phase
			name, _ := scan.ScanName()
			parts = append(parts, string(token.RuneRetrieve)+name)

		case token.IMM_EXECUTE:
			// ▷ - immediate execute: fire NOW (before placeholders bound)
			if e.deferDepth == 0 {
				name, err := e.scanNameOrDynamic(scan)
				if err != nil {
					return "", err
				}
				argsRaw, _ := scan.ScanUntilTerminator()
				result, err := e.execute(name, argsRaw)
				if err != nil {
					return "", err
				}
				// Re-evaluate result to splice any losp operators into the stream
				evaluated, err := e.Eval(result.String())
				if err != nil {
					return "", err
				}
				parts = append(parts, evaluated)
			} else {
				// Inside ◯, preserve as text (including any dynamic name operators)
				nameText, _ := e.scanNamePreservingOperators(scan)
				argsRaw, _ := scan.ScanUntilTerminator()
				parts = append(parts, string(token.RuneImmExecute)+nameText+argsRaw+string(token.RuneTerminator))
			}

		case token.EXECUTE:
			// ▶ - deferred execute: preserve as text for EXECUTE phase
			// Use scanNamePreservingOperators to preserve any dynamic naming for later
			nameText, _ := e.scanNamePreservingOperators(scan)
			bodyText, _ := e.scanBodyWithNestedTerminators(scan)
			parts = append(parts, string(token.RuneExecute)+nameText+bodyText+string(token.RuneTerminator))

		case token.IMM_STORE:
			// ▽ - immediate store: fire NOW
			if e.deferDepth == 0 {
				name, err := e.scanNameOrDynamic(scan)
				if err != nil {
					return "", err
				}
				bodyText, _ := scan.ScanUntilTerminator()
				evaluated, err := e.Eval(bodyText)
				if err != nil {
					return "", err
				}
				e.namespace.Set(name, expr.Text{Value: evaluated})
				// Auto-persist in ALWAYS mode
				if e.persistMode == PersistAlways && e.store != nil {
					e.autoPersist(name)
				}
				// Immediate store produces no output
			} else {
				// Inside ◯, preserve as text
				nameText, _ := e.scanNamePreservingOperators(scan)
				bodyText, _ := e.scanBodyWithNestedTerminators(scan)
				parts = append(parts, string(token.RuneImmStore)+nameText+bodyText+string(token.RuneTerminator))
			}

		case token.STORE:
			// ▼ - deferred store: preserve as text for EXECUTE phase
			// This allows dynamic naming (▼▲name value ◆) to work with placeholders,
			// since placeholders are bound in POPULATE phase which comes after PARSE.
			nameText, _ := e.scanNamePreservingOperators(scan)
			bodyText, _ := e.scanBodyWithNestedTerminators(scan)
			parts = append(parts, string(token.RuneStore)+nameText+bodyText+string(token.RuneTerminator))
		}
	}
}

func mustEval(e *Evaluator, s string) string {
	result, _ := e.Eval(s)
	return result
}

// parseArgs parses the argument string into individual arguments.
// Each expression is one argument. Text expressions are separated by newlines.
// Operators evaluate to single arguments (preserving multi-word content).
func (e *Evaluator) parseArgs(argsRaw string) ([]string, error) {
	scan := scanner.NewFromString(argsRaw)
	var args []string

	for {
		item, err := scan.Next()
		if err != nil {
			return nil, err
		}
		if item.Token == token.EOF {
			break
		}

		switch item.Token {
		case token.TEXT:
			// Text is split by newlines - each line is a separate argument
			// Empty lines are skipped (they're formatting, not arguments)
			lines := strings.Split(item.Value, "\n")
			for _, line := range lines {
				if s := strings.TrimSpace(line); s != "" {
					args = append(args, s)
				}
			}
		case token.IMM_RETRIEVE:
			// Operators always produce an argument, even if empty
			// Use scanNameOrDynamic to support dynamic naming (e.g., △▲ref)
			name, err := e.scanNameOrDynamic(scan)
			if err != nil {
				return nil, err
			}
			val := e.namespace.Get(name)
			args = append(args, strings.TrimSpace(val.String()))
		case token.IMM_EXECUTE:
			// Operators always produce an argument, even if empty
			// Use scanNameOrDynamic to support dynamic naming (e.g., ▷▲ref ◆)
			name, err := e.scanNameOrDynamic(scan)
			if err != nil {
				return nil, err
			}
			body, _ := scan.ScanUntilTerminator()
			res, err := e.execute(name, body)
			if err != nil {
				return nil, err
			}
			if res != nil {
				args = append(args, strings.TrimSpace(res.String()))
			} else {
				args = append(args, "")
			}
		case token.RETRIEVE:
			// Operators always produce an argument, even if empty
			// Use scanNameOrDynamic to support dynamic naming (e.g., ▲▲ref)
			name, err := e.scanNameOrDynamic(scan)
			if err != nil {
				return nil, err
			}
			val := e.namespace.Get(name)
			var result string
			if s, ok := val.(expr.Stored); ok {
				result, _ = e.parseBodyImmediateOnly(s.Body.String())
			} else {
				result, _ = e.parseBodyImmediateOnly(val.String())
			}
			args = append(args, strings.TrimSpace(result))
		case token.EXECUTE:
			// Operators always produce an argument, even if empty
			// Use scanNameOrDynamic to support dynamic naming (e.g., ▶▲ref ◆)
			name, err := e.scanNameOrDynamic(scan)
			if err != nil {
				return nil, err
			}
			body, _ := scan.ScanUntilTerminator()
			res, err := e.execute(name, body)
			if err != nil {
				return nil, err
			}
			if res != nil {
				args = append(args, strings.TrimSpace(res.String()))
			} else {
				args = append(args, "")
			}
		}
	}

	return args, nil
}

// concatResults concatenates all non-empty expressions into a single result.
// Whitespace-only results containing newlines (source formatting between statements)
// are collapsed into a single newline separator. Other whitespace (spaces on same
// line) is preserved as-is.
func (e *Evaluator) concatResults(exprs []expr.Expr) expr.Expr {
	var parts []string
	needsNewline := false

	for _, ex := range exprs {
		if !ex.IsEmpty() {
			s := ex.String()
			if strings.TrimSpace(s) == "" {
				// Whitespace-only: check if it contains newlines
				if strings.Contains(s, "\n") {
					// Newline-containing whitespace: mark for newline separator
					if len(parts) > 0 {
						needsNewline = true
					}
				} else {
					// Space-only whitespace (same line): preserve as-is
					if needsNewline {
						parts = append(parts, "\n")
						needsNewline = false
					}
					parts = append(parts, s)
				}
			} else {
				// Content: add newline separator if needed, then content
				if needsNewline {
					parts = append(parts, "\n")
					needsNewline = false
				}
				parts = append(parts, s)
			}
		}
	}

	if len(parts) == 0 {
		return expr.Empty{}
	}
	return expr.Text{Value: strings.Join(parts, "")}
}

// scanNameOrDynamic scans a name, supporting dynamic naming with operators.
// If the next character is a retrieve or execute operator, it evaluates it to get the name.
// Both immediate (△, ▷) and deferred (▲, ▶) operators are supported - in the name
// position they behave the same because we need the name NOW.
// Otherwise, it uses ScanName as usual.
func (e *Evaluator) scanNameOrDynamic(scan *scanner.Scanner) (string, error) {
	// Peek at next non-whitespace rune
	r, err := scan.PeekRune()
	if err != nil {
		return "", err
	}

	switch r {
	case token.RuneImmRetrieve, token.RuneRetrieve:
		// Consume the operator
		scan.Next()
		// Get the name to retrieve
		refName, err := scan.ScanName()
		if err != nil {
			return "", err
		}
		// Get the value from namespace and re-parse to resolve any operators
		val := e.namespace.Get(refName)
		result, err := e.Eval(val.String())
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(result), nil

	case token.RuneImmExecute, token.RuneExecute:
		// Consume the operator
		scan.Next()
		// Get the expression name and args
		exprName, err := scan.ScanName()
		if err != nil {
			return "", err
		}
		argsRaw, err := scan.ScanUntilTerminator()
		if err != nil {
			return "", err
		}
		// Execute and use result as name
		result, err := e.execute(exprName, argsRaw)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(result.String()), nil

	default:
		// Regular name
		return scan.ScanName()
	}
}

// scanNamePreservingOperators scans a name, preserving any dynamic naming operators as text.
// Used when we're inside a defer (◯) and need to preserve operators for later evaluation.
func (e *Evaluator) scanNamePreservingOperators(scan *scanner.Scanner) (string, error) {
	// Peek at next non-whitespace rune
	r, err := scan.PeekRune()
	if err != nil {
		return "", err
	}

	switch r {
	case token.RuneImmRetrieve:
		// △ - preserve as text
		scan.Next()
		refName, err := scan.ScanName()
		if err != nil {
			return "", err
		}
		return string(token.RuneImmRetrieve) + refName, nil

	case token.RuneRetrieve:
		// ▲ - preserve as text
		scan.Next()
		refName, err := scan.ScanName()
		if err != nil {
			return "", err
		}
		return string(token.RuneRetrieve) + refName, nil

	case token.RuneImmExecute:
		// ▷ - preserve as text including body
		scan.Next()
		exprName, err := scan.ScanName()
		if err != nil {
			return "", err
		}
		argsRaw, err := scan.ScanUntilTerminator()
		if err != nil {
			return "", err
		}
		return string(token.RuneImmExecute) + exprName + argsRaw + string(token.RuneTerminator), nil

	case token.RuneExecute:
		// ▶ - preserve as text including body
		scan.Next()
		exprName, err := scan.ScanName()
		if err != nil {
			return "", err
		}
		argsRaw, err := scan.ScanUntilTerminator()
		if err != nil {
			return "", err
		}
		return string(token.RuneExecute) + exprName + argsRaw + string(token.RuneTerminator), nil

	default:
		// Regular name
		return scan.ScanName()
	}
}

// Namespace returns the evaluator's namespace.
func (e *Evaluator) Namespace() *Namespace {
	return e.namespace
}

// Store returns the evaluator's persistence store.
func (e *Evaluator) Store() Store {
	return e.store
}

// Provider returns the evaluator's LLM provider.
func (e *Evaluator) Provider() Provider {
	return e.provider
}

// AsyncRegistry returns the evaluator's async registry.
func (e *Evaluator) AsyncRegistry() *AsyncRegistry {
	return e.asyncRegistry
}

// CorpusRegistry returns the evaluator's corpus registry.
func (e *Evaluator) CorpusRegistry() *CorpusRegistry {
	return e.corpusRegistry
}

// GetSetting returns a runtime setting value, or the default if unset.
func (e *Evaluator) GetSetting(key, defaultVal string) string {
	if v, ok := e.settings[key]; ok {
		return v
	}
	return defaultVal
}

// SetSetting sets a runtime setting value.
func (e *Evaluator) SetSetting(key, value string) {
	e.settings[key] = value
}

// PersistMode returns the current persistence mode.
func (e *Evaluator) PersistMode() PersistMode {
	return e.persistMode
}

// SetPersistMode sets the persistence mode.
func (e *Evaluator) SetPersistMode(mode PersistMode) {
	e.persistMode = mode
}

// autoPersist persists a value to the store (used in ALWAYS mode).
func (e *Evaluator) autoPersist(name string) {
	val := e.namespace.Get(name)
	fullDef := formatAsDefinition(name, val)
	e.store.Put(name, expr.Text{Value: fullDef})
}
