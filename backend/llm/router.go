// Package llm provides multi-provider LLM routing for OpenAI, Anthropic, Groq, and other providers.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Provider represents an LLM provider type.
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGroq      Provider = "groq"
	ProviderOllama    Provider = "ollama"
	ProviderLMStudio  Provider = "lmstudio"
)

// Model represents a configured model with provider info.
type Model struct {
	ID       string   `json:"id"`
	Provider Provider `json:"provider"`
	Name     string   `json:"name"`
}

// Router handles routing requests to different LLM providers.
type Router struct {
	mu       sync.RWMutex
	models   map[string]Model
	apiKeys  map[Provider]string
	baseURLs map[Provider]string
	client   *http.Client
}

// Config holds router configuration.
type Config struct {
	APIKeys  map[Provider]string
	BaseURLs map[Provider]string
}

// NewRouter creates a new LLM router.
func NewRouter(cfg Config) *Router {
	return &Router{
		models:   make(map[string]Model),
		apiKeys:  cfg.APIKeys,
		baseURLs: cfg.BaseURLs,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// RegisterModel adds a model to the router's registry.
func (r *Router) RegisterModel(model Model) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[model.ID] = model
}

// GetModel returns a model by ID.
func (r *Router) GetModel(id string) (Model, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.models[id]
	return model, ok
}

// ListModels returns all registered models.
func (r *Router) ListModels() []Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	models := make([]Model, 0, len(r.models))
	for _, m := range r.models {
		models = append(models, m)
	}
	return models
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	Model    string          `json:"model"`
	Messages []ChatMessage   `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
	MaxTokens int            `json:"max_tokens,omitempty"`
	Temperature float64      `json:"temperature,omitempty"`
}

// ChatMessage represents a single message in a chat.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Chat streams a chat completion request.
func (r *Router) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	model, ok := r.GetModel(req.Model)
	if !ok {
		return nil, fmt.Errorf("model %s not found", req.Model)
	}

	switch model.Provider {
	case ProviderOpenAI:
		return r.chatOpenAI(ctx, req)
	case ProviderAnthropic:
		return r.chatAnthropic(ctx, req)
	case ProviderGroq:
		return r.chatGroq(ctx, req)
	case ProviderOllama:
		return r.chatOllama(ctx, req)
	case ProviderLMStudio:
		return r.chatLMStudio(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", model.Provider)
	}
}

// StreamChat streams a chat completion with callback for each chunk.
func (r *Router) StreamChat(ctx context.Context, req ChatRequest, onChunk func(string) error) error {
	model, ok := r.GetModel(req.Model)
	if !ok {
		return fmt.Errorf("model %s not found", req.Model)
	}

	switch model.Provider {
	case ProviderOpenAI:
		return r.streamOpenAI(ctx, req, onChunk)
	case ProviderAnthropic:
		return r.streamAnthropic(ctx, req, onChunk)
	case ProviderGroq:
		return r.streamGroq(ctx, req, onChunk)
	case ProviderOllama:
		return r.streamOllama(ctx, req, onChunk)
	default:
		return fmt.Errorf("streaming not supported for provider: %s", model.Provider)
	}
}

// chatOpenAI handles OpenAI-compatible API requests.
func (r *Router) chatOpenAI(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	baseURL := r.baseURLs[ProviderOpenAI]
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if key := r.apiKeys[ProviderOpenAI]; key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// chatAnthropic handles Anthropic Claude API requests.
func (r *Router) chatAnthropic(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	baseURL := r.baseURLs[ProviderAnthropic]
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	url := fmt.Sprintf("%s/messages", baseURL)

	// Convert messages format for Anthropic
	anthropicReq := map[string]interface{}{
		"model": req.Model,
		"messages": req.Messages,
		"max_tokens": req.MaxTokens,
	}
	if req.Temperature > 0 {
		anthropicReq["temperature"] = req.Temperature
	}

	payload, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	if key := r.apiKeys[ProviderAnthropic]; key != "" {
		httpReq.Header.Set("x-api-key", key)
	}

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert Anthropic response to our format
	content := ""
	if c, ok := result["content"].([]interface{}); ok {
		for _, item := range c {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					content = text
					break
				}
			}
		}
	}

	return &ChatResponse{
		ID:    "",
		Model: req.Model,
		Choices: []Choice{{
			Index: 0,
			Message: ChatMessage{
				Role:    "assistant",
				Content: content,
			},
			FinishReason: "stop",
		}},
		Usage: Usage{},
	}, nil
}

// chatGroq handles Groq API requests.
func (r *Router) chatGroq(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	baseURL := r.baseURLs[ProviderGroq]
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if key := r.apiKeys[ProviderGroq]; key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// chatOllama handles Ollama API requests.
func (r *Router) chatOllama(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	baseURL := r.baseURLs[ProviderOllama]
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	url := fmt.Sprintf("%s/api/chat", baseURL)

	// Convert to Ollama format
	ollamaReq := map[string]interface{}{
		"model": req.Model,
		"messages": req.Messages,
		"stream": false,
	}
	if req.Temperature > 0 {
		ollamaReq["options"] = map[string]float64{"temperature": req.Temperature}
	}

	payload, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert Ollama response to our format
	content := ""
	if msg, ok := result["message"].(map[string]interface{}); ok {
		if c, ok := msg["content"].(string); ok {
			content = c
		}
	}

	return &ChatResponse{
		ID:    "",
		Model: req.Model,
		Choices: []Choice{{
			Index: 0,
			Message: ChatMessage{
				Role:    "assistant",
				Content: content,
			},
			FinishReason: "stop",
		}},
		Usage: Usage{},
	}, nil
}

// chatLMStudio handles LM Studio API requests.
func (r *Router) chatLMStudio(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	baseURL := r.baseURLs[ProviderLMStudio]
	if baseURL == "" {
		baseURL = "http://localhost:1234/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// Streaming implementations

func (r *Router) streamOpenAI(ctx context.Context, req ChatRequest, onChunk func(string) error) error {
	baseURL := r.baseURLs[ProviderOpenAI]
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if key := r.apiKeys[ProviderOpenAI]; key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	// Read SSE stream using bufio.Scanner
	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer size for potential long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if strings.TrimSpace(data) == "[DONE]" {
				break
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if json.Unmarshal([]byte(data), &chunk) == nil && len(chunk.Choices) > 0 {
				if err := onChunk(chunk.Choices[0].Delta.Content); err != nil {
					return err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}
	return nil
}

func (r *Router) streamAnthropic(ctx context.Context, req ChatRequest, onChunk func(string) error) error {
	baseURL := r.baseURLs[ProviderAnthropic]
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	url := fmt.Sprintf("%s/messages", baseURL)

	anthropicReq := map[string]interface{}{
		"model":           req.Model,
		"messages":        req.Messages,
		"max_tokens":      req.MaxTokens,
		"stream":          true,
	}
	if req.Temperature > 0 {
		anthropicReq["temperature"] = req.Temperature
	}

	payload, err := json.Marshal(anthropicReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")
	if key := r.apiKeys[ProviderAnthropic]; key != "" {
		httpReq.Header.Set("x-api-key", key)
	}

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	// Read SSE stream using bufio.Scanner
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if strings.TrimSpace(data) == "" {
				continue
			}
			var chunk struct {
				Content []struct {
					Type     string `json:"type"`
					Text     string `json:"text"`
				} `json:"content"`
			}
			if json.Unmarshal([]byte(data), &chunk) == nil && len(chunk.Content) > 0 {
				for _, c := range chunk.Content {
					if c.Type == "text" && c.Text != "" {
						if err := onChunk(c.Text); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}
	return nil
}

func (r *Router) streamGroq(ctx context.Context, req ChatRequest, onChunk func(string) error) error {
	// Groq uses OpenAI-compatible streaming
	return r.streamOpenAI(ctx, req, onChunk)
}

func (r *Router) streamOllama(ctx context.Context, req ChatRequest, onChunk func(string) error) error {
	baseURL := r.baseURLs[ProviderOllama]
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	url := fmt.Sprintf("%s/api/chat", baseURL)

	ollamaReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}
	if req.Temperature > 0 {
		ollamaReq["options"] = map[string]float64{"temperature": req.Temperature}
	}

	payload, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	dec := json.NewDecoder(resp.Body)
	for dec.More() {
		var chunk map[string]interface{}
		if err := dec.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("stream decode error: %w", err)
		}
		if msg, ok := chunk["message"].(map[string]interface{}); ok {
			if content, ok := msg["content"].(string); ok && content != "" {
				if err := onChunk(content); err != nil {
					return err
				}
			}
		}
	}
	return nil
}