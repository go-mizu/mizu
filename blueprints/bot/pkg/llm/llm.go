package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Provider generates AI responses from conversation history.
type Provider interface {
	Chat(ctx context.Context, req *types.LLMRequest) (*types.LLMResponse, error)
}

// Claude implements Provider using the Anthropic Messages API.
type Claude struct {
	apiKey string
	client *http.Client
}

// NewClaude creates a Claude provider. Uses ANTHROPIC_API_KEY env var.
func NewClaude() *Claude {
	return &Claude{
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *Claude) Chat(ctx context.Context, req *types.LLMRequest) (*types.LLMResponse, error) {
	if c.apiKey == "" {
		return &types.LLMResponse{
			Content: "I'm a bot assistant. To enable AI responses, set the ANTHROPIC_API_KEY environment variable.",
			Model:   req.Model,
		}, nil
	}

	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// Build Anthropic messages format
	messages := make([]map[string]string, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = map[string]string{
			"role":    m.Role,
			"content": m.Content,
		}
	}

	payload := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   messages,
	}
	if req.SystemPrompt != "" {
		payload["system"] = req.SystemPrompt
	}
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}

	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic API %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	content := ""
	if len(result.Content) > 0 {
		content = result.Content[0].Text
	}

	return &types.LLMResponse{
		Content:      content,
		Model:        result.Model,
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
	}, nil
}

// Echo is a simple provider that echoes back messages (for testing).
type Echo struct{}

func (e *Echo) Chat(_ context.Context, req *types.LLMRequest) (*types.LLMResponse, error) {
	last := "Hello!"
	if len(req.Messages) > 0 {
		last = req.Messages[len(req.Messages)-1].Content
	}
	return &types.LLMResponse{
		Content: fmt.Sprintf("[Echo] You said: %s", last),
		Model:   "echo",
	}, nil
}
