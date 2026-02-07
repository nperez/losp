//go:build !(js && wasm)

package provider

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Anthropic is a provider for Anthropic's Claude API.
type Anthropic struct {
	APIKey   string
	Model    string
	Timeout  time.Duration
	StreamCb StreamCallback
}

// AnthropicOption configures the Anthropic provider.
type AnthropicOption func(*Anthropic)

// WithAnthropicAPIKey sets the API key.
func WithAnthropicAPIKey(key string) AnthropicOption {
	return func(a *Anthropic) { a.APIKey = key }
}

// WithAnthropicModel sets the model name.
func WithAnthropicModel(model string) AnthropicOption {
	return func(a *Anthropic) { a.Model = model }
}

// WithAnthropicTimeout sets the request timeout.
func WithAnthropicTimeout(timeout time.Duration) AnthropicOption {
	return func(a *Anthropic) { a.Timeout = timeout }
}

// WithAnthropicStreamCallback sets the streaming callback.
func WithAnthropicStreamCallback(cb StreamCallback) AnthropicOption {
	return func(a *Anthropic) { a.StreamCb = cb }
}

// NewAnthropic creates a new Anthropic provider.
func NewAnthropic(opts ...AnthropicOption) *Anthropic {
	a := &Anthropic{
		APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		Model:   "claude-sonnet-4-20250514",
		Timeout: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	ContentBlock *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content_block"`
}

// Prompt sends a prompt to Anthropic and returns the response.
func (a *Anthropic) Prompt(system, user string) (string, error) {
	if a.APIKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	messages := []anthropicMessage{
		{Role: "user", Content: user},
	}

	reqBody := anthropicRequest{
		Model:     a.Model,
		MaxTokens: 4096,
		System:    system,
		Messages:  messages,
		Stream:    a.StreamCb != nil,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: a.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic error (%d): %s", resp.StatusCode, string(body))
	}

	if a.StreamCb != nil {
		return a.readStream(resp.Body)
	}

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("anthropic error: %s", result.Error.Message)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("anthropic: no content in response")
	}

	// Concatenate all text content blocks
	var sb strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}

	return sb.String(), nil
}

func (a *Anthropic) readStream(body io.Reader) (string, error) {
	scanner := bufio.NewScanner(body)
	var fullResponse strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "" {
			continue
		}

		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// Handle content_block_delta events
		if event.Type == "content_block_delta" && event.Delta != nil && event.Delta.Type == "text_delta" {
			text := event.Delta.Text
			fullResponse.WriteString(text)

			if a.StreamCb != nil {
				a.StreamCb(text)
			}
		}
	}

	return fullResponse.String(), scanner.Err()
}
