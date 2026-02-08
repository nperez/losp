package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Ollama is a provider for local Ollama LLM.
type Ollama struct {
	URL      string
	Model    string
	Timeout  time.Duration
	StreamCb StreamCallback
	params   map[string]string
}

// OllamaOption configures the Ollama provider.
type OllamaOption func(*Ollama)

// WithOllamaURL sets the Ollama API URL.
func WithOllamaURL(url string) OllamaOption {
	return func(o *Ollama) { o.URL = url }
}

// WithOllamaModel sets the model name.
func WithOllamaModel(model string) OllamaOption {
	return func(o *Ollama) { o.Model = model }
}

// WithOllamaTimeout sets the request timeout.
func WithOllamaTimeout(timeout time.Duration) OllamaOption {
	return func(o *Ollama) { o.Timeout = timeout }
}

// WithOllamaStreamCallback sets the streaming callback.
func WithOllamaStreamCallback(cb StreamCallback) OllamaOption {
	return func(o *Ollama) { o.StreamCb = cb }
}

// NewOllama creates a new Ollama provider.
func NewOllama(opts ...OllamaOption) *Ollama {
	o := &Ollama{
		URL:     "http://localhost:11434",
		Model:   "qwen3:30b-a3b-instruct-2507-q4_K_M",
		Timeout: 5 * time.Minute,
		params:  make(map[string]string),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// GetParam returns an inference parameter value.
func (o *Ollama) GetParam(key string) string { return o.params[key] }

// SetParam sets an inference parameter value.
func (o *Ollama) SetParam(key, value string) { o.params[key] = value }

// GetModel returns the current model name.
func (o *Ollama) GetModel() string { return o.Model }

// SetModel sets the model name.
func (o *Ollama) SetModel(model string) { o.Model = model }

// ProviderName returns "OLLAMA".
func (o *Ollama) ProviderName() string { return "OLLAMA" }

type ollamaRequest struct {
	Model     string                 `json:"model"`
	Messages  []ollamaMessage        `json:"messages"`
	Stream    bool                   `json:"stream"`
	Options   map[string]interface{} `json:"options,omitempty"`
	KeepAlive string                 `json:"keep_alive,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

// Prompt sends a prompt to Ollama and returns the response.
func (o *Ollama) Prompt(system, user string) (string, error) {
	messages := []ollamaMessage{}
	if system != "" {
		messages = append(messages, ollamaMessage{Role: "system", Content: system})
	}
	messages = append(messages, ollamaMessage{Role: "user", Content: user})

	options := map[string]interface{}{"num_ctx": 16384}
	if v, ok := o.params["NUM_CTX"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			options["num_ctx"] = n
		}
	}
	if v, ok := o.params["TEMPERATURE"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			options["temperature"] = f
		}
	}
	if v, ok := o.params["TOP_K"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			options["top_k"] = n
		}
	}
	if v, ok := o.params["TOP_P"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			options["top_p"] = f
		}
	}

	reqBody := ollamaRequest{
		Model:     o.Model,
		Messages:  messages,
		Stream:    o.StreamCb != nil,
		Options:   options,
		KeepAlive: "5m",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: o.Timeout}
	resp, err := client.Post(o.URL+"/api/chat", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama error: %s", string(body))
	}

	if o.StreamCb != nil {
		return o.readStream(resp.Body)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Message.Content, nil
}

func (o *Ollama) readStream(body io.Reader) (string, error) {
	decoder := json.NewDecoder(body)
	var fullResponse bytes.Buffer

	for {
		var chunk ollamaResponse
		if err := decoder.Decode(&chunk); err == io.EOF {
			break
		} else if err != nil {
			return fullResponse.String(), err
		}

		content := chunk.Message.Content
		fullResponse.WriteString(content)

		if o.StreamCb != nil {
			o.StreamCb(content)
		}

		if chunk.Done {
			break
		}
	}

	return fullResponse.String(), nil
}
