// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

// Package provider defines LLM provider interfaces and implementations.
package provider

// Provider is the interface for LLM providers.
type Provider interface {
	// Prompt sends a prompt to the LLM and returns the response.
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

// EmbeddingProvider generates vector embeddings from text.
type EmbeddingProvider interface {
	Embed(texts []string) ([][]float32, error)
}

// StreamCallback is called with each token during streaming.
type StreamCallback func(token string)
