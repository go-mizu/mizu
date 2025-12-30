package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultAnthropicURL   = "https://api.anthropic.com/v1"
	defaultAnthropicModel = "claude-3-5-sonnet-20241022"
	anthropicVersion      = "2023-06-01"
)

// AnthropicClient implements the Client interface for Anthropic.
type AnthropicClient struct {
	config *Config
	client *http.Client
}

// NewAnthropicClient creates a new Anthropic client.
func NewAnthropicClient(config *Config) *AnthropicClient {
	if config.BaseURL == "" {
		config.BaseURL = defaultAnthropicURL
	}
	if config.Model == "" {
		config.Model = defaultAnthropicModel
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	return &AnthropicClient{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

// Provider returns the provider name.
func (c *AnthropicClient) Provider() string {
	return "anthropic"
}

// Complete generates a completion.
func (c *AnthropicClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = c.config.Model
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.config.Temperature
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// Convert messages to Anthropic format
	messages, system := convertToAnthropicMessages(req.Messages)

	body := map[string]any{
		"model":       model,
		"messages":    messages,
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}

	if system != "" {
		body["system"] = system
	}
	if len(req.Stop) > 0 {
		body["stop_sequences"] = req.Stop
	}
	if len(req.Tools) > 0 {
		body["tools"] = convertToAnthropicTools(req.Tools)
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/messages", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	response := &CompletionResponse{
		ID:           result.ID,
		Model:        result.Model,
		FinishReason: result.StopReason,
		Usage: &Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}

	// Extract content and tool calls
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			response.Content = block.Text
		case "tool_use":
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}

	return response, nil
}

// CompleteStream generates a streaming completion.
func (c *AnthropicClient) CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = c.config.Model
	}

	messages, system := convertToAnthropicMessages(req.Messages)

	body := map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": req.MaxTokens,
		"stream":     true,
	}

	if system != "" {
		body["system"] = system
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/messages", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	chunks := make(chan StreamChunk, 100)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			select {
			case <-ctx.Done():
				chunks <- StreamChunk{Error: ctx.Err()}
				return
			default:
			}

			var event anthropicStreamEvent
			if err := decoder.Decode(&event); err != nil {
				if err != io.EOF {
					chunks <- StreamChunk{Error: err}
				}
				return
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Type == "text_delta" {
					chunks <- StreamChunk{Content: event.Delta.Text}
				}
			case "message_stop":
				chunks <- StreamChunk{FinishReason: "stop"}
				return
			}
		}
	}()

	return chunks, nil
}

// Embed generates embeddings - Anthropic doesn't have embeddings, so we return an error.
func (c *AnthropicClient) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not supported by Anthropic provider")
}

// Models returns available models.
func (c *AnthropicClient) Models(ctx context.Context) ([]ModelInfo, error) {
	// Anthropic doesn't have a models endpoint, return known models
	return []ModelInfo{
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Provider: "anthropic", MaxTokens: 8192},
		{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Provider: "anthropic", MaxTokens: 8192},
		{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Provider: "anthropic", MaxTokens: 4096},
	}, nil
}

// Helper types for Anthropic API

type anthropicResponse struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type  string         `json:"type"`
		Text  string         `json:"text,omitempty"`
		ID    string         `json:"id,omitempty"`
		Name  string         `json:"name,omitempty"`
		Input map[string]any `json:"input,omitempty"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

// Helper functions

func convertToAnthropicMessages(msgs []Message) ([]map[string]string, string) {
	var system string
	var result []map[string]string

	for _, m := range msgs {
		if m.Role == RoleSystem {
			system = m.Content
			continue
		}
		role := string(m.Role)
		if role == "assistant" {
			role = "assistant"
		}
		result = append(result, map[string]string{
			"role":    role,
			"content": m.Content,
		})
	}

	return result, system
}

func convertToAnthropicTools(tools []Tool) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, t := range tools {
		result[i] = map[string]any{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": t.Parameters,
		}
	}
	return result
}
