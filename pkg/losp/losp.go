package losp

import (
	"io"
	"os"
	"time"

	"nickandperla.net/losp/internal/eval"
)

// Runtime is the losp interpreter runtime.
type Runtime struct {
	evaluator    *eval.Evaluator
	store        eval.Store
	provider     eval.Provider
	streamCb     func(token string)
	inputReader  func(prompt string) (string, error)
	outputWriter func(text string) error
	timeout      time.Duration
	prelude      string          // Custom prelude source (if empty, uses DefaultPrelude)
	noStdlib     bool            // If true, skip loading prelude
	persistMode  eval.PersistMode // Controls persistence behavior
}

// New creates a new losp runtime with the given options.
func New(opts ...Option) *Runtime {
	r := &Runtime{
		timeout: 5 * time.Minute,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Build evaluator options
	evalOpts := []eval.Option{}
	if r.store != nil {
		evalOpts = append(evalOpts, eval.WithStore(r.store))
	}
	if r.provider != nil {
		evalOpts = append(evalOpts, eval.WithProvider(r.provider))
	}
	if r.streamCb != nil {
		evalOpts = append(evalOpts, eval.WithStreamCallback(r.streamCb))
	}
	if r.inputReader != nil {
		evalOpts = append(evalOpts, eval.WithInputReader(r.inputReader))
	}
	if r.outputWriter != nil {
		evalOpts = append(evalOpts, eval.WithOutputWriter(r.outputWriter))
	}
	evalOpts = append(evalOpts, eval.WithPersistMode(r.persistMode))

	r.evaluator = eval.New(evalOpts...)

	// Load prelude unless disabled
	if !r.noStdlib {
		prelude := r.prelude
		if prelude == "" {
			prelude = DefaultPrelude
		}

		// Check for database override
		if r.store != nil {
			if stdlibExpr, err := r.store.Get("__stdlib__"); err == nil && stdlibExpr != nil && !stdlibExpr.IsEmpty() {
				prelude = stdlibExpr.String()
			}
		}

		// Evaluate prelude to populate namespace
		if prelude != "" {
			r.evaluator.Eval(prelude)
		}
	}

	return r
}

// Eval evaluates a losp string and returns the result.
func (r *Runtime) Eval(input string) (string, error) {
	return r.evaluator.Eval(input)
}

// EvalReader evaluates losp from a reader.
func (r *Runtime) EvalReader(reader io.Reader) (string, error) {
	return r.evaluator.EvalReader(reader)
}

// EvalFile evaluates a losp file.
func (r *Runtime) EvalFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return r.EvalReader(f)
}

// LoadReader loads definitions from a reader in load-only mode.
func (r *Runtime) LoadReader(reader io.Reader) error {
	return r.evaluator.LoadReader(reader)
}

// LoadFile loads definitions from a file in load-only mode.
func (r *Runtime) LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.LoadReader(f)
}

// Close releases resources.
func (r *Runtime) Close() error {
	if r.store != nil {
		return r.store.Close()
	}
	return nil
}
