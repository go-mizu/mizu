package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
)

func TestNew(t *testing.T) {
	t.Run("requires API key", func(t *testing.T) {
		_, err := New(Config{})
		if err == nil {
			t.Error("expected error for missing API key")
		}
	})

	t.Run("creates client with API key", func(t *testing.T) {
		client, err := New(Config{APIKey: "test-key"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		if client.apiKey != "test-key" {
			t.Errorf("expected API key 'test-key', got %q", client.apiKey)
		}
		if client.baseURL != defaultBaseURL {
			t.Errorf("expected base URL %q, got %q", defaultBaseURL, client.baseURL)
		}
	})

	t.Run("uses custom base URL", func(t *testing.T) {
		client, err := New(Config{
			APIKey:  "test-key",
			BaseURL: "https://custom.api.com",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.baseURL != "https://custom.api.com" {
			t.Errorf("expected base URL 'https://custom.api.com', got %q", client.baseURL)
		}
	})
}

func TestProviderRegistration(t *testing.T) {
	// Set API key for registration test
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	providers := llm.Providers()
	found := false
	for _, p := range providers {
		if p == "claude" {
			found = true
			break
		}
	}
	if !found {
		t.Error("claude provider not registered")
	}
}

func TestModels(t *testing.T) {
	client, _ := New(Config{APIKey: "test-key"})

	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(models) != len(Models) {
		t.Errorf("expected %d models, got %d", len(Models), len(models))
	}

	// Check that all models have correct owner
	for _, m := range models {
		if m.OwnedBy != "anthropic" {
			t.Errorf("expected owned_by 'anthropic', got %q", m.OwnedBy)
		}
	}
}

func TestConvertRequest(t *testing.T) {
	client, _ := New(Config{APIKey: "test-key"})

	t.Run("extracts system prompt", func(t *testing.T) {
		req := llm.ChatRequest{
			Model: "claude-sonnet-4",
			Messages: []llm.Message{
				{Role: "system", Content: "You are helpful"},
				{Role: "user", Content: "Hello"},
			},
			MaxTokens: 100,
		}

		claudeReq, sysPrompt := client.convertRequest(req)

		if sysPrompt != "You are helpful" {
			t.Errorf("expected system prompt 'You are helpful', got %q", sysPrompt)
		}
		if claudeReq.System != "You are helpful" {
			t.Errorf("expected System field 'You are helpful', got %q", claudeReq.System)
		}
		if len(claudeReq.Messages) != 1 {
			t.Errorf("expected 1 message (system filtered out), got %d", len(claudeReq.Messages))
		}
	})

	t.Run("resolves model name", func(t *testing.T) {
		req := llm.ChatRequest{
			Model:    "claude-sonnet-4",
			Messages: []llm.Message{{Role: "user", Content: "Hi"}},
		}

		claudeReq, _ := client.convertRequest(req)

		expected := Models["claude-sonnet-4"].ID
		if claudeReq.Model != expected {
			t.Errorf("expected model %q, got %q", expected, claudeReq.Model)
		}
	})

	t.Run("converts tools", func(t *testing.T) {
		req := llm.ChatRequest{
			Model:    "claude-sonnet-4",
			Messages: []llm.Message{{Role: "user", Content: "Search for Go"}},
			Tools: []llm.Tool{{
				Name:        "web_search",
				Description: "Search the web",
				InputSchema: llm.JSONSchema{
					Type: "object",
					Properties: map[string]llm.Property{
						"query": {Type: "string", Description: "Search query"},
					},
					Required: []string{"query"},
				},
			}},
			ToolChoice: "auto",
		}

		claudeReq, _ := client.convertRequest(req)

		if len(claudeReq.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(claudeReq.Tools))
		}
		if claudeReq.Tools[0].Name != "web_search" {
			t.Errorf("expected tool name 'web_search', got %q", claudeReq.Tools[0].Name)
		}
		if claudeReq.ToolChoice == nil || claudeReq.ToolChoice.Type != "auto" {
			t.Error("expected tool_choice type 'auto'")
		}
	})
}

func TestChatCompletion(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("missing or incorrect API key header")
		}
		if r.Header.Get("anthropic-version") != anthropicVersion {
			t.Error("missing or incorrect anthropic-version header")
		}

		// Return mock response
		resp := claudeResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{{
				Type: "text",
				Text: "Hello! How can I help you?",
			}},
			Model:      "claude-sonnet-4-20250514",
			StopReason: "end_turn",
			Usage: claudeUsage{
				InputTokens:  10,
				OutputTokens: 8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, _ := New(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	resp, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:     "claude-sonnet-4",
		Messages:  []llm.Message{{Role: "user", Content: "Hello"}},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != "msg_123" {
		t.Errorf("expected ID 'msg_123', got %q", resp.ID)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello! How can I help you?" {
		t.Errorf("unexpected content: %q", resp.Choices[0].Message.Content)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 8 {
		t.Errorf("expected 8 completion tokens, got %d", resp.Usage.CompletionTokens)
	}
}

func TestChatCompletionWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := claudeResponse{
			ID:   "msg_456",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "Let me search for that."},
				{
					Type:  "tool_use",
					ID:    "toolu_123",
					Name:  "web_search",
					Input: json.RawMessage(`{"query":"Go programming language"}`),
				},
			},
			Model:      "claude-sonnet-4-20250514",
			StopReason: "tool_use",
			Usage: claudeUsage{
				InputTokens:  15,
				OutputTokens: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, _ := New(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	resp, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:    "claude-sonnet-4",
		Messages: []llm.Message{{Role: "user", Content: "Search for Go"}},
		Tools:    ToolDefinitions[:1], // Just web_search
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Choices[0].FinishReason != "tool_calls" {
		t.Errorf("expected finish reason 'tool_calls', got %q", resp.Choices[0].FinishReason)
	}
	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Choices[0].Message.ToolCalls))
	}

	tc := resp.Choices[0].Message.ToolCalls[0]
	if tc.ID != "toolu_123" {
		t.Errorf("expected tool call ID 'toolu_123', got %q", tc.ID)
	}
	if tc.Name != "web_search" {
		t.Errorf("expected tool name 'web_search', got %q", tc.Name)
	}
}

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		model        string
		inputTokens  int
		outputTokens int
		expectedMin  float64
		expectedMax  float64
	}{
		{"claude-haiku-4.5", 1000, 500, 0.0035, 0.0036}, // Low cost: (1000/1M * 1.00) + (500/1M * 5.00) = 0.0035
		{"claude-sonnet-4", 1000, 500, 0.0105, 0.0106},  // Medium cost: (1000/1M * 3.00) + (500/1M * 15.00) = 0.0105
		{"claude-opus-4", 1000, 500, 0.0525, 0.0526},    // High cost: (1000/1M * 15.00) + (500/1M * 75.00) = 0.0525
		{"unknown-model", 1000, 500, 0, 0},              // Unknown model
	}

	for _, tc := range tests {
		t.Run(tc.model, func(t *testing.T) {
			cost := CalculateCost(tc.model, tc.inputTokens, tc.outputTokens)
			if cost < tc.expectedMin || cost > tc.expectedMax {
				t.Errorf("cost %.6f outside expected range [%.6f, %.6f]", cost, tc.expectedMin, tc.expectedMax)
			}
		})
	}
}

func TestToolDefinitions(t *testing.T) {
	tools := GetAllTools()

	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}

	expectedNames := []string{"web_search", "fetch_url", "execute_code"}
	for i, name := range expectedNames {
		if tools[i].Name != name {
			t.Errorf("expected tool %d to be %q, got %q", i, name, tools[i].Name)
		}
	}

	// Test GetToolByName
	tool := GetToolByName("web_search")
	if tool == nil {
		t.Fatal("expected to find web_search tool")
	}
	if tool.Name != "web_search" {
		t.Errorf("expected name 'web_search', got %q", tool.Name)
	}

	// Test non-existent tool
	if GetToolByName("nonexistent") != nil {
		t.Error("expected nil for nonexistent tool")
	}
}

func TestGetSystemPrompt(t *testing.T) {
	tests := []struct {
		mode                string
		includeToolGuidance bool
		contains            string
	}{
		{"quick", false, "concise"},
		{"deep", false, "research assistant"},
		{"research", false, "comprehensive analysis"},
		{"deepsearch", false, "comprehensive report"},
		{"deep", true, "Tool Usage Guidelines"},
	}

	for _, tc := range tests {
		t.Run(tc.mode, func(t *testing.T) {
			prompt := GetSystemPrompt(tc.mode, tc.includeToolGuidance)
			if prompt == "" {
				t.Error("expected non-empty prompt")
			}
			// Just verify we get some content back
			if len(prompt) < 50 {
				t.Error("prompt seems too short")
			}
		})
	}
}

// Integration test - requires ANTHROPIC_API_KEY and LLM_INTEGRATION_TEST=1
func TestChatCompletionIntegration(t *testing.T) {
	if os.Getenv("LLM_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test (set LLM_INTEGRATION_TEST=1 to run)")
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test (ANTHROPIC_API_KEY not set)")
	}

	client, err := New(Config{APIKey: apiKey})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
		Model:       "claude-3.5-haiku",
		Messages:    []llm.Message{{Role: "user", Content: "Say 'hello' and nothing else."}},
		MaxTokens:   10,
		Temperature: 0,
	})
	if err != nil {
		t.Fatalf("chat completion failed: %v", err)
	}

	t.Logf("Response: %s", resp.Choices[0].Message.Content)
	t.Logf("Usage: %d input, %d output tokens", resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	if resp.Usage.PromptTokens == 0 {
		t.Error("expected non-zero prompt tokens")
	}
}
