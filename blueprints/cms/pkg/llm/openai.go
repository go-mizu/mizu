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
	defaultOpenAIURL = "https://api.openai.com/v1"
	defaultModel     = "gpt-4o-mini"
)

// OpenAIClient implements the Client interface for OpenAI.
type OpenAIClient struct {
	config *Config
	client *http.Client
}

// NewOpenAIClient creates a new OpenAI client.
func NewOpenAIClient(config *Config) *OpenAIClient {
	if config.BaseURL == "" {
		config.BaseURL = defaultOpenAIURL
	}
	if config.Model == "" {
		config.Model = defaultModel
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &OpenAIClient{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

// Provider returns the provider name.
func (c *OpenAIClient) Provider() string {
	return "openai"
}

// Complete generates a completion.
func (c *OpenAIClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
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

	body := map[string]any{
		"model":       model,
		"messages":    convertMessages(req.Messages),
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}

	if req.TopP > 0 {
		body["top_p"] = req.TopP
	}
	if len(req.Stop) > 0 {
		body["stop"] = req.Stop
	}
	if len(req.Tools) > 0 {
		body["tools"] = convertTools(req.Tools)
		if req.ToolChoice != "" {
			body["tool_choice"] = req.ToolChoice
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := result.Choices[0]
	response := &CompletionResponse{
		ID:           result.ID,
		Model:        result.Model,
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: &Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}

	// Parse tool calls
	if len(choice.Message.ToolCalls) > 0 {
		for _, tc := range choice.Message.ToolCalls {
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				args = map[string]any{"raw": tc.Function.Arguments}
			}
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			})
		}
	}

	return response, nil
}

// CompleteStream generates a streaming completion.
func (c *OpenAIClient) CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = c.config.Model
	}

	body := map[string]any{
		"model":       model,
		"messages":    convertMessages(req.Messages),
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"stream":      true,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

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

			var chunk openAIStreamChunk
			if err := decoder.Decode(&chunk); err != nil {
				if err != io.EOF {
					chunks <- StreamChunk{Error: err}
				}
				return
			}

			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]
				chunks <- StreamChunk{
					Content:      choice.Delta.Content,
					FinishReason: choice.FinishReason,
				}
			}
		}
	}()

	return chunks, nil
}

// Embed generates embeddings.
func (c *OpenAIClient) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = "text-embedding-3-small"
	}

	body := map[string]any{
		"model": model,
		"input": req.Input,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/embeddings", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result openAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	response := &EmbeddingResponse{
		Model:      result.Model,
		Embeddings: make([][]float32, len(result.Data)),
		Usage: &Usage{
			PromptTokens: result.Usage.PromptTokens,
			TotalTokens:  result.Usage.TotalTokens,
		},
	}

	for i, emb := range result.Data {
		response.Embeddings[i] = emb.Embedding
	}

	return response, nil
}

// Models returns available models.
func (c *OpenAIClient) Models(ctx context.Context) ([]ModelInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.config.BaseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	models := make([]ModelInfo, len(result.Data))
	for i, m := range result.Data {
		models[i] = ModelInfo{
			ID:       m.ID,
			Name:     m.ID,
			Provider: "openai",
		}
	}

	return models, nil
}

// Helper types for OpenAI API responses

type openAIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type openAIEmbeddingResponse struct {
	Model string `json:"model"`
	Data  []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Helper functions

func convertMessages(msgs []Message) []map[string]string {
	result := make([]map[string]string, len(msgs))
	for i, m := range msgs {
		result[i] = map[string]string{
			"role":    string(m.Role),
			"content": m.Content,
		}
	}
	return result
}

func convertTools(tools []Tool) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, t := range tools {
		result[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		}
	}
	return result
}
