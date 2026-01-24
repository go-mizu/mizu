// Package llm provides a provider-agnostic interface for LLM backends.
// The interface is designed to be OpenAI-compatible for easy integration.
package llm

import (
	"context"
	"errors"
	"sync"
)

// Provider is the main interface for LLM backends.
type Provider interface {
	// ChatCompletion performs a chat completion request.
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatCompletionStream performs a streaming chat completion request.
	ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)

	// Embedding generates embeddings for the given input texts.
	Embedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error)

	// Models returns the list of available models.
	Models(ctx context.Context) ([]Model, error)

	// Ping checks if the provider is healthy.
	Ping(ctx context.Context) error
}

// ChatRequest represents a chat completion request (OpenAI compatible).
type ChatRequest struct {
	Model       string    `json:"model,omitempty"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamEvent represents a streaming event.
type StreamEvent struct {
	ID    string `json:"id,omitempty"`
	Delta string `json:"delta"`
	Done  bool   `json:"done"`
	Error error  `json:"error,omitempty"`
}

// EmbeddingRequest represents an embedding request.
type EmbeddingRequest struct {
	Model string   `json:"model,omitempty"`
	Input []string `json:"input"`
}

// EmbeddingResponse represents an embedding response.
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

// EmbeddingData represents a single embedding.
type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

// Model represents an available model.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// Config holds common configuration for providers.
type Config struct {
	BaseURL string
	APIKey  string
	Timeout int // seconds
}

// Common errors.
var (
	ErrProviderNotFound = errors.New("llm: provider not found")
	ErrInvalidRequest   = errors.New("llm: invalid request")
	ErrContextCanceled  = errors.New("llm: context canceled")
	ErrStreamClosed     = errors.New("llm: stream closed")
)

// ProviderFactory creates a new Provider instance.
type ProviderFactory func(Config) (Provider, error)

var (
	providersMu sync.RWMutex
	providers   = make(map[string]ProviderFactory)
)

// Register registers a provider factory.
func Register(name string, factory ProviderFactory) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers[name] = factory
}

// New creates a new Provider instance by name.
func New(name string, cfg Config) (Provider, error) {
	providersMu.RLock()
	factory, ok := providers[name]
	providersMu.RUnlock()
	if !ok {
		return nil, ErrProviderNotFound
	}
	return factory(cfg)
}

// Providers returns the list of registered provider names.
func Providers() []string {
	providersMu.RLock()
	defer providersMu.RUnlock()
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}
