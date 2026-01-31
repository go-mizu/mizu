package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func TestClaude_ChatWithTools_EndTurn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != "/v1/messages" {
			t.Errorf("expected path /v1/messages, got %s", r.URL.Path)
		}
		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key test-key, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version 2023-06-01, got %s", r.Header.Get("anthropic-version"))
		}

		// Verify the request has tools
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := body["tools"]; !ok {
			t.Error("expected tools in request")
		}

		// Return end_turn response
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "The answer is 42."},
			},
			"model":       "claude-test",
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 5},
		})
	}))
	defer server.Close()

	c := NewClaudeWithURL("test-key", server.URL)
	resp, err := c.ChatWithTools(context.Background(), &types.LLMToolRequest{
		Model:        "claude-test",
		SystemPrompt: "You are helpful.",
		Messages: []any{
			map[string]any{"role": "user", "content": "What is the answer?"},
		},
		MaxTokens: 1024,
		Tools: []types.ToolDefinition{
			{
				Name:        "get_answer",
				Description: "Gets the answer to everything",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ChatWithTools: %v", err)
	}

	if resp.StopReason != "end_turn" {
		t.Errorf("expected stop_reason end_turn, got %s", resp.StopReason)
	}
	if resp.Model != "claude-test" {
		t.Errorf("expected model claude-test, got %s", resp.Model)
	}
	if resp.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 5 {
		t.Errorf("expected 5 output tokens, got %d", resp.OutputTokens)
	}

	text := resp.TextContent()
	if text != "The answer is 42." {
		t.Errorf("expected text 'The answer is 42.', got %q", text)
	}

	uses := resp.ToolUses()
	if len(uses) != 0 {
		t.Errorf("expected 0 tool uses, got %d", len(uses))
	}
}

func TestClaude_ChatWithTools_ToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Let me look that up."},
				{
					"type":  "tool_use",
					"id":    "toolu_01A",
					"name":  "get_weather",
					"input": map[string]any{"location": "San Francisco"},
				},
			},
			"model":       "claude-test",
			"stop_reason": "tool_use",
			"usage":       map[string]any{"input_tokens": 20, "output_tokens": 15},
		})
	}))
	defer server.Close()

	c := NewClaudeWithURL("test-key", server.URL)
	resp, err := c.ChatWithTools(context.Background(), &types.LLMToolRequest{
		Messages: []any{
			map[string]any{"role": "user", "content": "What is the weather in SF?"},
		},
		Tools: []types.ToolDefinition{
			{
				Name:        "get_weather",
				Description: "Get the current weather",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string"},
					},
					"required": []string{"location"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ChatWithTools: %v", err)
	}

	if resp.StopReason != "tool_use" {
		t.Errorf("expected stop_reason tool_use, got %s", resp.StopReason)
	}

	text := resp.TextContent()
	if text != "Let me look that up." {
		t.Errorf("expected text 'Let me look that up.', got %q", text)
	}

	uses := resp.ToolUses()
	if len(uses) != 1 {
		t.Fatalf("expected 1 tool use, got %d", len(uses))
	}
	if uses[0].Name != "get_weather" {
		t.Errorf("expected tool name get_weather, got %s", uses[0].Name)
	}
	if uses[0].ID != "toolu_01A" {
		t.Errorf("expected tool ID toolu_01A, got %s", uses[0].ID)
	}
	loc, ok := uses[0].Input["location"]
	if !ok || loc != "San Francisco" {
		t.Errorf("expected input location 'San Francisco', got %v", loc)
	}
}

func TestClaude_ChatWithTools_NoAPIKey(t *testing.T) {
	c := NewClaudeWithURL("", "http://should-not-be-called")
	resp, err := c.ChatWithTools(context.Background(), &types.LLMToolRequest{
		Model: "test-model",
		Messages: []any{
			map[string]any{"role": "user", "content": "hello"},
		},
	})
	if err != nil {
		t.Fatalf("ChatWithTools: %v", err)
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("expected stop_reason end_turn, got %s", resp.StopReason)
	}
	if resp.TextContent() != "No API key configured." {
		t.Errorf("expected fallback text, got %q", resp.TextContent())
	}
}

func TestClaude_ChatWithTools_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	defer server.Close()

	c := NewClaudeWithURL("test-key", server.URL)
	_, err := c.ChatWithTools(context.Background(), &types.LLMToolRequest{
		Messages: []any{
			map[string]any{"role": "user", "content": "hello"},
		},
	})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestClaude_Chat_WithBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("expected path /v1/messages, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Hello from mock!"},
			},
			"model": "claude-test",
			"usage": map[string]any{"input_tokens": 5, "output_tokens": 3},
		})
	}))
	defer server.Close()

	c := NewClaudeWithURL("test-key", server.URL)
	resp, err := c.Chat(context.Background(), &types.LLMRequest{
		Messages: []types.LLMMsg{
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello from mock!" {
		t.Errorf("expected 'Hello from mock!', got %q", resp.Content)
	}
}
