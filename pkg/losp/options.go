// Package losp provides the public API for the losp interpreter.
package losp

import (
	"io"
	"time"

	"nickandperla.net/losp/internal/eval"
	"nickandperla.net/losp/internal/provider"
	"nickandperla.net/losp/internal/store"
)

// Option configures a Runtime.
type Option func(*Runtime)

// WithSQLiteStore configures SQLite persistence at the given path.
func WithSQLiteStore(path string) Option {
	return func(r *Runtime) {
		s, err := store.NewSQLite(path)
		if err == nil {
			r.store = s
		}
	}
}

// WithMemoryStore configures an in-memory store (for testing).
func WithMemoryStore() Option {
	return func(r *Runtime) {
		r.store = store.NewMemory()
	}
}

// WithMockProvider configures a mock LLM provider (for testing).
func WithMockProvider(response string) Option {
	return func(r *Runtime) {
		r.provider = provider.NewMock(response)
	}
}

// WithMockProviderFunc configures a mock LLM provider with a custom handler (for testing).
// The handler receives the system and user prompts and returns the response.
func WithMockProviderFunc(handler func(system, user string) string) Option {
	return func(r *Runtime) {
		r.provider = provider.NewMockHandler(handler)
	}
}

// WithStreamCallback sets the streaming callback for LLM output.
func WithStreamCallback(cb func(token string)) Option {
	return func(r *Runtime) {
		r.streamCb = cb
	}
}

// WithInputReader sets the input reader for READ builtin.
func WithInputReader(reader func(prompt string) (string, error)) Option {
	return func(r *Runtime) {
		r.inputReader = reader
	}
}

// WithOutputWriter sets the output writer for SAY builtin.
func WithOutputWriter(writer func(text string) error) Option {
	return func(r *Runtime) {
		r.outputWriter = writer
	}
}

// WithOutput sets the io.Writer for output.
func WithOutput(w io.Writer) Option {
	return func(r *Runtime) {
		r.outputWriter = func(text string) error {
			_, err := w.Write([]byte(text))
			return err
		}
	}
}

// WithTimeout sets the timeout for LLM requests.
func WithTimeout(timeout time.Duration) Option {
	return func(r *Runtime) {
		r.timeout = timeout
	}
}

// WithNoPrompt disables LLM prompts (for testing).
func WithNoPrompt() Option {
	return func(r *Runtime) {
		r.provider = nil
	}
}

// WithPrelude sets a custom prelude source to be loaded on startup.
// If not set, DefaultPrelude is used.
func WithPrelude(source string) Option {
	return func(r *Runtime) {
		r.prelude = source
	}
}

// WithNoStdlib disables loading the standard library prelude.
func WithNoStdlib() Option {
	return func(r *Runtime) {
		r.noStdlib = true
	}
}

// Store interface for custom stores.
type Store = eval.Store

// Provider interface for custom providers.
type Provider = eval.Provider

// PersistMode controls when expressions are persisted.
type PersistMode = eval.PersistMode

// Persist mode constants.
const (
	PersistOnDemand = eval.PersistOnDemand
	PersistAlways   = eval.PersistAlways
	PersistNever    = eval.PersistNever
)

// ParsePersistMode parses a string into a PersistMode.
func ParsePersistMode(s string) (PersistMode, bool) {
	return eval.ParsePersistMode(s)
}

// WithPersistMode sets the persistence mode.
func WithPersistMode(mode PersistMode) Option {
	return func(r *Runtime) {
		r.persistMode = mode
	}
}

// ProviderFactory creates a new provider with the given stream callback.
type ProviderFactory = eval.ProviderFactory

// StreamCallback is called with streaming LLM output.
type StreamCallback = eval.StreamCallback

// WithProviderFactory registers a provider factory by name.
func WithProviderFactory(name string, f ProviderFactory) Option {
	return func(r *Runtime) {
		r.providerFactories[name] = f
	}
}

// WithOllama configures the Ollama LLM provider.
func WithOllama(url, model string) Option {
	return func(r *Runtime) {
		opts := []provider.OllamaOption{}
		if url != "" {
			opts = append(opts, provider.WithOllamaURL(url))
		}
		if model != "" {
			opts = append(opts, provider.WithOllamaModel(model))
		}
		r.provider = provider.NewOllama(opts...)
		// Register factory for runtime switching
		r.providerFactories["OLLAMA"] = func(streamCb eval.StreamCallback) eval.Provider {
			fOpts := []provider.OllamaOption{}
			if url != "" {
				fOpts = append(fOpts, provider.WithOllamaURL(url))
			}
			if streamCb != nil {
				fOpts = append(fOpts, provider.WithOllamaStreamCallback(provider.StreamCallback(streamCb)))
			}
			return provider.NewOllama(fOpts...)
		}
	}
}

// WithOpenRouter configures the OpenRouter LLM provider.
func WithOpenRouter(model string) Option {
	return func(r *Runtime) {
		opts := []provider.OpenRouterOption{}
		if model != "" {
			opts = append(opts, provider.WithOpenRouterModel(model))
		}
		r.provider = provider.NewOpenRouter(opts...)
		// Register factory for runtime switching
		r.providerFactories["OPENROUTER"] = func(streamCb eval.StreamCallback) eval.Provider {
			fOpts := []provider.OpenRouterOption{}
			if streamCb != nil {
				fOpts = append(fOpts, provider.WithOpenRouterStreamCallback(provider.StreamCallback(streamCb)))
			}
			return provider.NewOpenRouter(fOpts...)
		}
	}
}

// WithAnthropic configures the Anthropic Claude LLM provider.
func WithAnthropic(model string) Option {
	return func(r *Runtime) {
		opts := []provider.AnthropicOption{}
		if model != "" {
			opts = append(opts, provider.WithAnthropicModel(model))
		}
		r.provider = provider.NewAnthropic(opts...)
		// Register factory for runtime switching
		r.providerFactories["ANTHROPIC"] = func(streamCb eval.StreamCallback) eval.Provider {
			fOpts := []provider.AnthropicOption{}
			if streamCb != nil {
				fOpts = append(fOpts, provider.WithAnthropicStreamCallback(provider.StreamCallback(streamCb)))
			}
			return provider.NewAnthropic(fOpts...)
		}
	}
}
