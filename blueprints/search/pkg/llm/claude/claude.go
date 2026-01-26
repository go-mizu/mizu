// Package claude provides an Anthropic Claude driver for the llm package.
package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
)

const (
	defaultBaseURL   = "https://api.anthropic.com/v1"
	defaultTimeout   = 120 * time.Second
	anthropicVersion = "2023-06-01"
)

func init() {
	llm.Register("claude", func(cfg llm.Config) (llm.Provider, error) {
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		return New(Config{
			APIKey:  apiKey,
			BaseURL: cfg.BaseURL,
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		})
	})
}

// ModelConfig holds configuration for a Claude model.
type ModelConfig struct {
	ID           string
	ContextSize  int
	InputCost    float64 // per 1M tokens
	OutputCost   float64 // per 1M tokens
	Capabilities []string
}

// Models maps friendly names to Claude model configurations.
var Models = map[string]ModelConfig{
	// Claude 4.5 Series (Latest - 2025/2026)
	"claude-opus-4.5": {
		ID:           "claude-opus-4-5-20251101",
		ContextSize:  200000,
		InputCost:    5.00,
		OutputCost:   25.00,
		Capabilities: []string{"text", "vision"},
	},
	"claude-sonnet-4.5": {
		ID:           "claude-sonnet-4-5-20250929",
		ContextSize:  200000, // 1M with beta header
		InputCost:    3.00,
		OutputCost:   15.00,
		Capabilities: []string{"text", "vision"},
	},
	"claude-haiku-4.5": {
		ID:           "claude-haiku-4-5-20251001",
		ContextSize:  200000,
		InputCost:    1.00,
		OutputCost:   5.00,
		Capabilities: []string{"text", "vision"},
	},
	// Claude 4 Series (Legacy)
	"claude-opus-4": {
		ID:           "claude-opus-4-20250514",
		ContextSize:  200000,
		InputCost:    15.00,
		OutputCost:   75.00,
		Capabilities: []string{"text", "vision"},
	},
	"claude-sonnet-4": {
		ID:           "claude-sonnet-4-20250514",
		ContextSize:  200000,
		InputCost:    3.00,
		OutputCost:   15.00,
		Capabilities: []string{"text", "vision"},
	},
	// Claude 3 Series (Legacy)
	"claude-3-haiku": {
		ID:           "claude-3-haiku-20240307",
		ContextSize:  200000,
		InputCost:    0.25,
		OutputCost:   1.25,
		Capabilities: []string{"text"},
	},
}

// TierModels maps tiers to default Claude models.
var TierModels = map[string]string{
	"quick":    "claude-haiku-4.5",
	"deep":     "claude-sonnet-4.5",
	"research": "claude-opus-4.5",
}

// Config holds configuration for the Claude client.
type Config struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
	Model   string // Default model to use (e.g., "claude-haiku-4.5")
}

// Client is an Anthropic Claude API client.
type Client struct {
	apiKey       string
	baseURL      string
	defaultModel string
	httpClient   *http.Client
}

// New creates a new Claude client.
func New(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("claude: API key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.Model == "" {
		cfg.Model = "claude-sonnet-4.5" // Default to Sonnet 4.5
	}

	return &Client{
		apiKey:       cfg.APIKey,
		baseURL:      strings.TrimSuffix(cfg.BaseURL, "/"),
		defaultModel: cfg.Model,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

// Name returns the provider name.
func (c *Client) Name() string {
	return "claude"
}

// claudeRequest is the request format for Claude API.
type claudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    []claudeMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []claudeTool    `json:"tools,omitempty"`
	ToolChoice  *toolChoice     `json:"tool_choice,omitempty"`
}

type claudeMessage struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type claudeTool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema JSONSchema `json:"input_schema"`
}

type JSONSchema struct {
	Type        string              `json:"type"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Required    []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type toolChoice struct {
	Type string `json:"type"` // "auto", "any", "tool"
	Name string `json:"name,omitempty"`
}

// claudeResponse is the response format from Claude API.
type claudeResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []contentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence string         `json:"stop_sequence,omitempty"`
	Usage        claudeUsage    `json:"usage"`
}

type claudeUsage struct {
	InputTokens        int `json:"input_tokens"`
	OutputTokens       int `json:"output_tokens"`
	CacheCreationInput int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInput     int `json:"cache_read_input_tokens,omitempty"`
}

// ChatCompletion performs a chat completion request.
func (c *Client) ChatCompletion(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	claudeReq, systemPrompt := c.convertRequest(req)

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("claude: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("claude: create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("claude: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claude: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("claude: decode response: %w", err)
	}

	return c.convertResponse(claudeResp, systemPrompt), nil
}

// ChatCompletionStream performs a streaming chat completion request.
func (c *Client) ChatCompletionStream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	req.Stream = true
	claudeReq, _ := c.convertRequest(req)
	claudeReq.Stream = true

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("claude: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("claude: create request: %w", err)
	}
	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("claude: do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("claude: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan llm.StreamEvent, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		var inputTokens, outputTokens int
		var currentToolCall *llm.ToolCall

		for {
			select {
			case <-ctx.Done():
				ch <- llm.StreamEvent{Error: ctx.Err(), Done: true}
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					ch <- llm.StreamEvent{
						Done:         true,
						InputTokens:  inputTokens,
						OutputTokens: outputTokens,
					}
					return
				}
				ch <- llm.StreamEvent{Error: err, Done: true}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- llm.StreamEvent{
					Done:         true,
					InputTokens:  inputTokens,
					OutputTokens: outputTokens,
				}
				return
			}

			var event streamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "message_start":
				if event.Message != nil && event.Message.Usage != nil {
					inputTokens = event.Message.Usage.InputTokens
				}

			case "content_block_start":
				if event.ContentBlock != nil {
					if event.ContentBlock.Type == "tool_use" {
						currentToolCall = &llm.ToolCall{
							ID:   event.ContentBlock.ID,
							Name: event.ContentBlock.Name,
						}
					}
				}

			case "content_block_delta":
				if event.Delta != nil {
					switch event.Delta.Type {
					case "text_delta":
						if event.Delta.Text != "" {
							ch <- llm.StreamEvent{Delta: event.Delta.Text}
						}
					case "input_json_delta":
						if currentToolCall != nil && event.Delta.PartialJSON != "" {
							// Accumulate partial JSON for tool call
							currentToolCall.Input = append(currentToolCall.Input, []byte(event.Delta.PartialJSON)...)
						}
					}
				}

			case "content_block_stop":
				if currentToolCall != nil {
					ch <- llm.StreamEvent{ToolCall: currentToolCall}
					currentToolCall = nil
				}

			case "message_delta":
				if event.Usage != nil {
					outputTokens = event.Usage.OutputTokens
				}

			case "message_stop":
				ch <- llm.StreamEvent{
					Done:         true,
					InputTokens:  inputTokens,
					OutputTokens: outputTokens,
				}
				return

			case "error":
				ch <- llm.StreamEvent{
					Error: fmt.Errorf("claude: %s", event.Error.Message),
					Done:  true,
				}
				return
			}
		}
	}()

	return ch, nil
}

type streamEvent struct {
	Type         string              `json:"type"`
	Message      *streamMessage      `json:"message,omitempty"`
	ContentBlock *streamContentBlock `json:"content_block,omitempty"`
	Delta        *streamDelta        `json:"delta,omitempty"`
	Usage        *streamUsage        `json:"usage,omitempty"`
	Error        *streamError        `json:"error,omitempty"`
	Index        int                 `json:"index,omitempty"`
}

type streamMessage struct {
	ID    string       `json:"id"`
	Usage *streamUsage `json:"usage,omitempty"`
}

type streamContentBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type streamDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

type streamUsage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
}

type streamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Embedding generates embeddings for the given input texts.
// Note: Claude doesn't have a native embedding API, so this returns an error.
func (c *Client) Embedding(ctx context.Context, req llm.EmbeddingRequest) (*llm.EmbeddingResponse, error) {
	return nil, fmt.Errorf("claude: embedding not supported - use a dedicated embedding model")
}

// Models returns the list of available models.
func (c *Client) Models(ctx context.Context) ([]llm.Model, error) {
	models := make([]llm.Model, 0, len(Models))
	for name, cfg := range Models {
		models = append(models, llm.Model{
			ID:      name,
			Object:  "model",
			OwnedBy: "anthropic",
			Created: time.Now().Unix(),
		})
		_ = cfg // Use cfg for additional metadata if needed
	}
	return models, nil
}

// Ping checks if the API is accessible.
// For Claude, we verify connectivity by checking the API key format
// and making a lightweight HEAD request to the API endpoint.
func (c *Client) Ping(ctx context.Context) error {
	// Check if API key is configured
	if c.apiKey == "" {
		return fmt.Errorf("claude: API key not configured")
	}

	// Make a lightweight request to check connectivity
	// We use a GET to the models endpoint which doesn't cost anything
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("claude: create ping request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("claude: ping request failed: %w", err)
	}
	defer resp.Body.Close()

	// 200 = success, 401 = bad key, other errors indicate issues
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("claude: invalid API key")
	}

	// For other status codes (like 404 for endpoint not found),
	// still consider the API reachable if we got a response
	return nil
}

// setHeaders sets the required headers for Claude API requests.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
}

// convertRequest converts an llm.ChatRequest to Claude's format.
func (c *Client) convertRequest(req llm.ChatRequest) (claudeRequest, string) {
	// Extract system prompt from messages
	var systemPrompt string
	var messages []claudeMessage

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}

		claudeMsg := claudeMessage{Role: msg.Role}

		// Handle tool results
		if msg.Role == "tool" && msg.ToolCallID != "" {
			claudeMsg.Role = "user"
			claudeMsg.Content = []contentBlock{{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   msg.Content,
			}}
			messages = append(messages, claudeMsg)
			continue
		}

		// Handle assistant messages with tool calls
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			var content []contentBlock
			if msg.Content != "" {
				content = append(content, contentBlock{Type: "text", Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				content = append(content, contentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: tc.Input,
				})
			}
			claudeMsg.Content = content
			messages = append(messages, claudeMsg)
			continue
		}

		// Regular message
		claudeMsg.Content = []contentBlock{{Type: "text", Text: msg.Content}}
		messages = append(messages, claudeMsg)
	}

	// Resolve model name
	modelID := req.Model
	if modelID == "" {
		modelID = c.defaultModel
	}
	if cfg, ok := Models[modelID]; ok {
		modelID = cfg.ID
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	claudeReq := claudeRequest{
		Model:       modelID,
		MaxTokens:   maxTokens,
		Messages:    messages,
		System:      systemPrompt,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}

	// Convert tools
	if len(req.Tools) > 0 {
		claudeReq.Tools = make([]claudeTool, len(req.Tools))
		for i, t := range req.Tools {
			claudeReq.Tools[i] = claudeTool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: JSONSchema{
					Type:       t.InputSchema.Type,
					Properties: convertProperties(t.InputSchema.Properties),
					Required:   t.InputSchema.Required,
				},
			}
		}

		// Set tool choice
		switch req.ToolChoice {
		case "auto", "":
			claudeReq.ToolChoice = &toolChoice{Type: "auto"}
		case "any":
			claudeReq.ToolChoice = &toolChoice{Type: "any"}
		case "none":
			claudeReq.ToolChoice = nil
		default:
			// Specific tool name
			claudeReq.ToolChoice = &toolChoice{Type: "tool", Name: req.ToolChoice}
		}
	}

	return claudeReq, systemPrompt
}

func convertProperties(props map[string]llm.Property) map[string]Property {
	if props == nil {
		return nil
	}
	result := make(map[string]Property, len(props))
	for k, v := range props {
		result[k] = Property{
			Type:        v.Type,
			Description: v.Description,
			Enum:        v.Enum,
		}
	}
	return result
}

// convertResponse converts Claude's response to llm.ChatResponse.
func (c *Client) convertResponse(resp claudeResponse, _ string) *llm.ChatResponse {
	var content string
	var toolCalls []llm.ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			toolCalls = append(toolCalls, llm.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}

	finishReason := resp.StopReason
	if finishReason == "end_turn" {
		finishReason = "stop"
	} else if finishReason == "tool_use" {
		finishReason = "tool_calls"
	}

	return &llm.ChatResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []llm.Choice{{
			Index: 0,
			Message: llm.Message{
				Role:      "assistant",
				Content:   content,
				ToolCalls: toolCalls,
			},
			FinishReason: finishReason,
		}},
		Usage: llm.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
			CacheReadTokens:  resp.Usage.CacheReadInput,
			CacheWriteTokens: resp.Usage.CacheCreationInput,
		},
	}
}

// CalculateCost calculates the cost for a given model and token usage.
func CalculateCost(model string, inputTokens, outputTokens int) float64 {
	cfg, ok := Models[model]
	if !ok {
		// Try to find by ID
		for _, c := range Models {
			if c.ID == model {
				cfg = c
				ok = true
				break
			}
		}
	}
	if !ok {
		return 0
	}

	inputCost := float64(inputTokens) / 1_000_000 * cfg.InputCost
	outputCost := float64(outputTokens) / 1_000_000 * cfg.OutputCost
	return inputCost + outputCost
}
