// Package llm provides a unified interface for LLM providers.
package llm

import (
	"context"
	"time"
)

// Role represents a message role.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a chat message.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest represents a completion request.
type CompletionRequest struct {
	Model       string            // Model to use (optional, uses default)
	Messages    []Message         // Conversation messages
	MaxTokens   int               // Max tokens to generate
	Temperature float64           // Temperature (0-2)
	TopP        float64           // Top-p sampling
	Stop        []string          // Stop sequences
	Tools       []Tool            // Available tools
	ToolChoice  string            // "auto", "none", or tool name
	Metadata    map[string]string // Additional metadata
}

// Tool represents a callable tool.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema
}

// ToolCall represents an LLM's request to call a tool.
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// CompletionResponse represents a completion response.
type CompletionResponse struct {
	ID           string     // Response ID
	Model        string     // Model used
	Content      string     // Generated text
	ToolCalls    []ToolCall // Tool calls requested
	FinishReason string     // "stop", "length", "tool_calls"
	Usage        *Usage     // Token usage
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// EmbeddingRequest represents an embedding request.
type EmbeddingRequest struct {
	Model string   // Model to use
	Input []string // Texts to embed
}

// EmbeddingResponse represents an embedding response.
type EmbeddingResponse struct {
	Model      string      // Model used
	Embeddings [][]float32 // Generated embeddings
	Usage      *Usage      // Token usage
}

// Client defines the LLM client interface.
type Client interface {
	// Complete generates a completion.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// CompleteStream generates a streaming completion.
	CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)

	// Embed generates embeddings.
	Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// Models returns available models.
	Models(ctx context.Context) ([]ModelInfo, error)

	// Provider returns the provider name.
	Provider() string
}

// StreamChunk represents a streaming chunk.
type StreamChunk struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
	Error        error
}

// ModelInfo represents model information.
type ModelInfo struct {
	ID          string
	Name        string
	Provider    string
	MaxTokens   int
	InputCost   float64 // Per 1M tokens
	OutputCost  float64 // Per 1M tokens
}

// Config holds LLM configuration.
type Config struct {
	Provider    string  // "openai", "anthropic", "local"
	APIKey      string  // API key
	BaseURL     string  // Custom base URL
	Model       string  // Default model
	MaxRetries  int     // Max retries
	Timeout     time.Duration
	Temperature float64 // Default temperature
}
