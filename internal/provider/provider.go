// Package provider defines LLM provider interfaces and implementations.
package provider

// Provider is the interface for LLM providers.
type Provider interface {
	// Prompt sends a prompt to the LLM and returns the response.
	Prompt(system, user string) (string, error)
}

// StreamCallback is called with each token during streaming.
type StreamCallback func(token string)
