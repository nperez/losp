package main

import (
	"fmt"
	"os"

	"nickandperla.net/losp/internal/provider"
	"nickandperla.net/losp/pkg/losp"
)

func configureProvider(opts *[]losp.Option, providerF, ollamaURL, model string) {
	switch providerF {
	case "ollama":
		*opts = append(*opts, losp.WithOllama(ollamaURL, model))
	case "openrouter":
		*opts = append(*opts, losp.WithOpenRouter(model))
	case "anthropic":
		*opts = append(*opts, losp.WithAnthropic(model))
	case "":
		// Default to ollama if available
		*opts = append(*opts, losp.WithOllama(ollamaURL, model))
	default:
		fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", providerF)
		os.Exit(1)
	}

	// Register additional provider factories for runtime switching.
	// WithOllama/WithOpenRouter/WithAnthropic already register their own factory,
	// so we only register factories for providers NOT already selected.
	if providerF != "ollama" && providerF != "" {
		*opts = append(*opts, losp.WithProviderFactory("OLLAMA", func(streamCb losp.StreamCallback) losp.Provider {
			fOpts := []provider.OllamaOption{}
			if ollamaURL != "" {
				fOpts = append(fOpts, provider.WithOllamaURL(ollamaURL))
			}
			if streamCb != nil {
				fOpts = append(fOpts, provider.WithOllamaStreamCallback(provider.StreamCallback(streamCb)))
			}
			return provider.NewOllama(fOpts...)
		}))
	}
	if providerF != "openrouter" {
		if os.Getenv("OPEN_ROUTER_API_KEY") != "" {
			*opts = append(*opts, losp.WithProviderFactory("OPENROUTER", func(streamCb losp.StreamCallback) losp.Provider {
				fOpts := []provider.OpenRouterOption{}
				if streamCb != nil {
					fOpts = append(fOpts, provider.WithOpenRouterStreamCallback(provider.StreamCallback(streamCb)))
				}
				return provider.NewOpenRouter(fOpts...)
			}))
		}
	}
	if providerF != "anthropic" {
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			*opts = append(*opts, losp.WithProviderFactory("ANTHROPIC", func(streamCb losp.StreamCallback) losp.Provider {
				fOpts := []provider.AnthropicOption{}
				if streamCb != nil {
					fOpts = append(fOpts, provider.WithAnthropicStreamCallback(provider.StreamCallback(streamCb)))
				}
				return provider.NewAnthropic(fOpts...)
			}))
		}
	}
}
