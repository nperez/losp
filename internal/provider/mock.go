// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package provider

// Mock is a mock provider for testing.
type Mock struct {
	Response string
	Handler  func(system, user string) string
	model    string
	params   map[string]string
}

// NewMock creates a new mock provider with a fixed response.
func NewMock(response string) *Mock {
	return &Mock{Response: response, model: "mock-model", params: make(map[string]string)}
}

// NewMockHandler creates a mock provider with a custom handler.
func NewMockHandler(handler func(system, user string) string) *Mock {
	return &Mock{Handler: handler, model: "mock-model", params: make(map[string]string)}
}

// Prompt returns the mock response or calls the handler.
func (m *Mock) Prompt(system, user string) (string, error) {
	if m.Handler != nil {
		return m.Handler(system, user), nil
	}
	return m.Response, nil
}

// GetParam returns an inference parameter value.
func (m *Mock) GetParam(key string) string { return m.params[key] }

// SetParam sets an inference parameter value.
func (m *Mock) SetParam(key, value string) { m.params[key] = value }

// GetModel returns the current model name.
func (m *Mock) GetModel() string { return m.model }

// SetModel sets the model name.
func (m *Mock) SetModel(model string) { m.model = model }

// ProviderName returns "MOCK".
func (m *Mock) ProviderName() string { return "MOCK" }
