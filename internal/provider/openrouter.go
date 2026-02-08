package provider

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// OpenRouter is a provider for OpenRouter API.
type OpenRouter struct {
	APIKey   string
	Model    string
	Timeout  time.Duration
	StreamCb StreamCallback
	params   map[string]string
}

// OpenRouterOption configures the OpenRouter provider.
type OpenRouterOption func(*OpenRouter)

// WithOpenRouterAPIKey sets the API key.
func WithOpenRouterAPIKey(key string) OpenRouterOption {
	return func(o *OpenRouter) { o.APIKey = key }
}

// WithOpenRouterModel sets the model name.
func WithOpenRouterModel(model string) OpenRouterOption {
	return func(o *OpenRouter) { o.Model = model }
}

// WithOpenRouterTimeout sets the request timeout.
func WithOpenRouterTimeout(timeout time.Duration) OpenRouterOption {
	return func(o *OpenRouter) { o.Timeout = timeout }
}

// WithOpenRouterStreamCallback sets the streaming callback.
func WithOpenRouterStreamCallback(cb StreamCallback) OpenRouterOption {
	return func(o *OpenRouter) { o.StreamCb = cb }
}

// NewOpenRouter creates a new OpenRouter provider.
func NewOpenRouter(opts ...OpenRouterOption) *OpenRouter {
	o := &OpenRouter{
		APIKey:  os.Getenv("OPEN_ROUTER_API_KEY"),
		Model:   "z-ai/glm-4.5-air:free",
		Timeout: 5 * time.Minute,
		params:  make(map[string]string),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// GetParam returns an inference parameter value.
func (o *OpenRouter) GetParam(key string) string { return o.params[key] }

// SetParam sets an inference parameter value.
func (o *OpenRouter) SetParam(key, value string) { o.params[key] = value }

// GetModel returns the current model name.
func (o *OpenRouter) GetModel() string { return o.Model }

// SetModel sets the model name.
func (o *OpenRouter) SetModel(model string) { o.Model = model }

// ProviderName returns "OPENROUTER".
func (o *OpenRouter) ProviderName() string { return "OPENROUTER" }

type openRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []openRouterMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature *float64            `json:"temperature,omitempty"`
	TopP        *float64            `json:"top_p,omitempty"`
	TopK        *int                `json:"top_k,omitempty"`
	MaxTokens   *int                `json:"max_tokens,omitempty"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []struct {
		Message openRouterMessage `json:"message"`
		Delta   openRouterMessage `json:"delta"`
	} `json:"choices"`
}

// Prompt sends a prompt to OpenRouter and returns the response.
func (o *OpenRouter) Prompt(system, user string) (string, error) {
	if o.APIKey == "" {
		return "", fmt.Errorf("OPEN_ROUTER_API_KEY not set")
	}

	// Retry up to 3 times on empty responses (free tier rate limiting)
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		result, err := o.promptOnce(system, user)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		if result != "" {
			return result, nil
		}
		lastErr = fmt.Errorf("empty response")
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	return "", fmt.Errorf("openrouter: failed after 3 attempts: %v", lastErr)
}

func (o *OpenRouter) promptOnce(system, user string) (string, error) {
	// Combine system and user into single user message
	// Many free models don't support system prompts
	combinedUser := user
	if system != "" {
		combinedUser = system + "\n\n" + user
	}

	messages := []openRouterMessage{}
	messages = append(messages, openRouterMessage{Role: "user", Content: combinedUser})

	reqBody := openRouterRequest{
		Model:    o.Model,
		Messages: messages,
		Stream:   o.StreamCb != nil,
	}
	if v, ok := o.params["TEMPERATURE"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			reqBody.Temperature = &f
		}
	}
	if v, ok := o.params["TOP_P"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			reqBody.TopP = &f
		}
	}
	if v, ok := o.params["TOP_K"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			reqBody.TopK = &n
		}
	}
	if v, ok := o.params["MAX_TOKENS"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			reqBody.MaxTokens = &n
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	client := &http.Client{Timeout: o.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openrouter error: %s", string(body))
	}

	if o.StreamCb != nil {
		return o.readStream(resp.Body)
	}

	var result openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("openrouter: no response choices returned (possible rate limit or content filter)")
	}

	return result.Choices[0].Message.Content, nil
}

type openRouterEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openRouterEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// Embed generates embeddings via OpenRouter's /v1/embeddings endpoint (OpenAI-compatible).
func (o *OpenRouter) Embed(texts []string) ([][]float32, error) {
	if o.APIKey == "" {
		return nil, fmt.Errorf("OPEN_ROUTER_API_KEY not set")
	}

	model := o.params["EMBED_MODEL"]
	if model == "" {
		model = o.Model
	}
	reqBody := openRouterEmbedRequest{
		Model: model,
		Input: texts,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	client := &http.Client{Timeout: o.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter embed error: %s", string(body))
	}

	var result openRouterEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Sort by index to maintain order
	embeddings := make([][]float32, len(result.Data))
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}

	return embeddings, nil
}

func (o *OpenRouter) readStream(body io.Reader) (string, error) {
	scanner := bufio.NewScanner(body)
	var fullResponse strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openRouterResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			fullResponse.WriteString(content)

			if o.StreamCb != nil {
				o.StreamCb(content)
			}
		}
	}

	return fullResponse.String(), scanner.Err()
}
