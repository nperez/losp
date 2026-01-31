package provider

// Mock is a mock provider for testing.
type Mock struct {
	Response string
	Handler  func(system, user string) string
}

// NewMock creates a new mock provider with a fixed response.
func NewMock(response string) *Mock {
	return &Mock{Response: response}
}

// NewMockHandler creates a mock provider with a custom handler.
func NewMockHandler(handler func(system, user string) string) *Mock {
	return &Mock{Handler: handler}
}

// Prompt returns the mock response or calls the handler.
func (m *Mock) Prompt(system, user string) (string, error) {
	if m.Handler != nil {
		return m.Handler(system, user), nil
	}
	return m.Response, nil
}
