// Package llm provides a provider-agnostic interface for LLM backends.
// The interface is designed to be OpenAI-compatible for easy integration.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// Provider is the main interface for LLM backends.
type Provider interface {
	// Name returns the provider name (e.g., "llamacpp", "claude").
	Name() string

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
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"` // "auto", "any", "none"
}

// Message represents a chat message.
type Message struct {
	Role       string       `json:"role"` // system, user, assistant, tool
	Content    string       `json:"content,omitempty"`
	ToolCalls  []ToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"` // For tool result messages
}

// JSONSchema represents a JSON Schema for tool input validation.
type JSONSchema struct {
	Type        string              `json:"type"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Required    []string            `json:"required,omitempty"`
	Description string              `json:"description,omitempty"`
}

// Property represents a property in a JSON Schema.
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// Tool represents a function that can be called by the LLM.
type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema JSONSchema `json:"input_schema"`
}

// ToolCall represents a tool invocation from the LLM.
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
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
	CacheReadTokens  int `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
}

// StreamEvent represents a streaming event.
type StreamEvent struct {
	ID           string     `json:"id,omitempty"`
	Delta        string     `json:"delta"`
	Done         bool       `json:"done"`
	Error        error      `json:"error,omitempty"`
	ToolCall     *ToolCall  `json:"tool_call,omitempty"`
	Usage        *Usage     `json:"usage,omitempty"`
	InputTokens  int        `json:"input_tokens,omitempty"`
	OutputTokens int        `json:"output_tokens,omitempty"`
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
	ErrToolNotFound     = errors.New("llm: tool not found")
	ErrToolExecution    = errors.New("llm: tool execution failed")
)

// RequestMetrics captures observability data for an LLM request.
type RequestMetrics struct {
	Provider         string        `json:"provider"`
	Model            string        `json:"model"`
	RequestID        string        `json:"request_id"`
	StartTime        time.Time     `json:"start_time"`
	TimeToFirstToken time.Duration `json:"time_to_first_token"`
	TotalDuration    time.Duration `json:"total_duration"`
	TokensPerSecond  float64       `json:"tokens_per_second"`
	InputTokens      int           `json:"input_tokens"`
	OutputTokens     int           `json:"output_tokens"`
	CacheReadTokens  int           `json:"cache_read_tokens"`
	CacheWriteTokens int           `json:"cache_write_tokens"`
	CostUSD          float64       `json:"cost_usd"`
	ToolCalls        int           `json:"tool_calls"`
	Success          bool          `json:"success"`
	Error            string        `json:"error,omitempty"`
}

// MetricsCollector is the interface for collecting LLM metrics.
type MetricsCollector interface {
	RecordRequest(ctx context.Context, m RequestMetrics)
	GetSessionTotals(sessionID string) (inputTokens, outputTokens int, costUSD float64)
}

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
