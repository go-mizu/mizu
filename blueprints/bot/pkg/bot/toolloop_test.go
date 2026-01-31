package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/tools"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// mockToolProvider implements llm.ToolProvider for testing.
type mockToolProvider struct {
	responses []*types.LLMToolResponse
	calls     int
	chatCalls int
}

func (m *mockToolProvider) Chat(ctx context.Context, req *types.LLMRequest) (*types.LLMResponse, error) {
	m.chatCalls++
	return &types.LLMResponse{Content: "[Echo]", Model: "test"}, nil
}

func (m *mockToolProvider) ChatWithTools(ctx context.Context, req *types.LLMToolRequest) (*types.LLMToolResponse, error) {
	if m.calls >= len(m.responses) {
		return &types.LLMToolResponse{
			Content:    []types.ContentBlock{{Type: "text", Text: "done"}},
			StopReason: "end_turn",
		}, nil
	}
	resp := m.responses[m.calls]
	m.calls++
	return resp, nil
}

func testRegistry() *tools.Registry {
	r := tools.NewRegistry()
	r.Register(&tools.Tool{
		Name:        "test_tool",
		Description: "Returns a fixed result",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Execute: func(ctx context.Context, input map[string]any) (string, error) {
			return "test_result", nil
		},
	})
	return r
}

func TestToolLoop_SingleToolCall(t *testing.T) {
	registry := testRegistry()

	provider := &mockToolProvider{
		responses: []*types.LLMToolResponse{
			{
				Content: []types.ContentBlock{
					{Type: "tool_use", ID: "call_1", Name: "test_tool", Input: map[string]any{}},
				},
				StopReason: "tool_use",
			},
			{
				Content: []types.ContentBlock{
					{Type: "text", Text: "Here is the result"},
				},
				StopReason: "end_turn",
			},
		},
	}

	req := &types.LLMToolRequest{
		Model:    "test",
		Messages: []any{map[string]any{"role": "user", "content": "run test_tool"}},
		Tools: []types.ToolDefinition{
			{Name: "test_tool", Description: "test", InputSchema: map[string]any{"type": "object"}},
		},
	}

	resp, err := runToolLoop(context.Background(), provider, registry, req)
	if err != nil {
		t.Fatalf("runToolLoop: %v", err)
	}

	if resp.TextContent() != "Here is the result" {
		t.Errorf("expected 'Here is the result', got: %s", resp.TextContent())
	}

	if provider.calls != 2 {
		t.Errorf("expected 2 ChatWithTools calls, got %d", provider.calls)
	}

	// Verify the tool result was appended as a user message.
	// After iteration: original user msg + assistant msg + tool result msg = 3.
	if len(req.Messages) != 3 {
		t.Errorf("expected 3 messages in request, got %d", len(req.Messages))
	}
}

func TestToolLoop_MultipleIterations(t *testing.T) {
	registry := testRegistry()

	// Also register a second tool.
	registry.Register(&tools.Tool{
		Name:        "second_tool",
		Description: "Returns another result",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Execute: func(ctx context.Context, input map[string]any) (string, error) {
			return "second_result", nil
		},
	})

	provider := &mockToolProvider{
		responses: []*types.LLMToolResponse{
			{
				Content: []types.ContentBlock{
					{Type: "tool_use", ID: "call_1", Name: "test_tool", Input: map[string]any{}},
				},
				StopReason: "tool_use",
			},
			{
				Content: []types.ContentBlock{
					{Type: "tool_use", ID: "call_2", Name: "second_tool", Input: map[string]any{}},
				},
				StopReason: "tool_use",
			},
			{
				Content: []types.ContentBlock{
					{Type: "text", Text: "All tools executed"},
				},
				StopReason: "end_turn",
			},
		},
	}

	req := &types.LLMToolRequest{
		Model:    "test",
		Messages: []any{map[string]any{"role": "user", "content": "run both tools"}},
	}

	resp, err := runToolLoop(context.Background(), provider, registry, req)
	if err != nil {
		t.Fatalf("runToolLoop: %v", err)
	}

	if resp.TextContent() != "All tools executed" {
		t.Errorf("expected 'All tools executed', got: %s", resp.TextContent())
	}

	if provider.calls != 3 {
		t.Errorf("expected 3 ChatWithTools calls, got %d", provider.calls)
	}

	// 1 original + 2*(assistant+tool_result) = 5
	if len(req.Messages) != 5 {
		t.Errorf("expected 5 messages in request, got %d", len(req.Messages))
	}
}

func TestToolLoop_MaxIterations(t *testing.T) {
	registry := testRegistry()

	// Provider always returns tool_use, never finishes.
	provider := &mockToolProvider{
		responses: func() []*types.LLMToolResponse {
			responses := make([]*types.LLMToolResponse, maxToolIterations+5)
			for i := range responses {
				responses[i] = &types.LLMToolResponse{
					Content: []types.ContentBlock{
						{Type: "tool_use", ID: "call_inf", Name: "test_tool", Input: map[string]any{}},
					},
					StopReason: "tool_use",
				}
			}
			return responses
		}(),
	}

	req := &types.LLMToolRequest{
		Model:    "test",
		Messages: []any{map[string]any{"role": "user", "content": "loop forever"}},
	}

	_, err := runToolLoop(context.Background(), provider, registry, req)
	if err == nil {
		t.Fatal("expected error for exceeding max iterations")
	}

	if !strings.Contains(err.Error(), "exceeded") {
		t.Errorf("error should mention 'exceeded', got: %v", err)
	}

	if provider.calls != maxToolIterations {
		t.Errorf("expected %d calls, got %d", maxToolIterations, provider.calls)
	}
}

func TestToolLoop_UnknownTool(t *testing.T) {
	registry := testRegistry() // Only has "test_tool".

	provider := &mockToolProvider{
		responses: []*types.LLMToolResponse{
			{
				Content: []types.ContentBlock{
					{Type: "tool_use", ID: "call_unk", Name: "nonexistent_tool", Input: map[string]any{}},
				},
				StopReason: "tool_use",
			},
			{
				Content: []types.ContentBlock{
					{Type: "text", Text: "Understood, tool not found"},
				},
				StopReason: "end_turn",
			},
		},
	}

	req := &types.LLMToolRequest{
		Model:    "test",
		Messages: []any{map[string]any{"role": "user", "content": "use unknown tool"}},
	}

	resp, err := runToolLoop(context.Background(), provider, registry, req)
	if err != nil {
		t.Fatalf("runToolLoop: %v", err)
	}

	if resp.TextContent() != "Understood, tool not found" {
		t.Errorf("unexpected response: %s", resp.TextContent())
	}

	// Verify the tool result message contains an error.
	// Messages: original user + assistant + tool_result = 3.
	if len(req.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(req.Messages))
	}

	// The last message is the tool_result user message.
	toolResultMsg, ok := req.Messages[2].(map[string]any)
	if !ok {
		t.Fatal("expected tool result message to be map[string]any")
	}
	content, ok := toolResultMsg["content"].([]any)
	if !ok {
		t.Fatal("expected tool result content to be []any")
	}
	if len(content) != 1 {
		t.Fatalf("expected 1 tool result block, got %d", len(content))
	}
	block, ok := content[0].(types.ToolResultBlock)
	if !ok {
		t.Fatal("expected ToolResultBlock")
	}
	if !block.IsError {
		t.Error("expected IsError to be true for unknown tool")
	}
	if !strings.Contains(block.Content, "Unknown tool") {
		t.Errorf("expected 'Unknown tool' in content, got: %s", block.Content)
	}
}

func TestBot_HandleMessage_WithTools(t *testing.T) {
	cfg := testConfig(t)

	provider := &mockToolProvider{
		responses: []*types.LLMToolResponse{
			{
				Content: []types.ContentBlock{
					{Type: "tool_use", ID: "call_1", Name: "test_tool", Input: map[string]any{}},
				},
				StopReason: "tool_use",
			},
			{
				Content: []types.ContentBlock{
					{Type: "text", Text: "Tool result processed"},
				},
				StopReason: "end_turn",
			},
		},
	}

	b, err := New(cfg, provider)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	// Register a test tool in the bot's registry.
	b.tools.Register(&tools.Tool{
		Name:        "test_tool",
		Description: "Returns a fixed result",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Execute: func(ctx context.Context, input map[string]any) (string, error) {
			return "test_result", nil
		},
	})

	ctx := context.Background()
	resp, err := b.HandleMessage(ctx, &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-test",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     "Please use the tool",
		Origin:      "dm",
	})
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	if resp != "Tool result processed" {
		t.Errorf("expected 'Tool result processed', got: %s", resp)
	}

	// Verify ChatWithTools was called (not Chat).
	if provider.calls != 2 {
		t.Errorf("expected 2 ChatWithTools calls, got %d", provider.calls)
	}
	if provider.chatCalls != 0 {
		t.Errorf("expected 0 Chat calls, got %d", provider.chatCalls)
	}
}

func TestBot_HandleMessage_FallbackWithoutToolProvider(t *testing.T) {
	// Use a provider that does NOT implement ToolProvider (Echo).
	cfg := testConfig(t)

	// capturingLLM does not implement ToolProvider, so it should fallback.
	var chatCalled bool
	provider := &capturingLLM{
		onChat: func(req *types.LLMRequest) {
			chatCalled = true
		},
	}

	b, err := New(cfg, provider)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	resp, err := b.HandleMessage(ctx, testInboundMessage("Hello without tools"))
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	if !chatCalled {
		t.Error("expected Chat to be called for non-ToolProvider")
	}

	if !strings.Contains(resp, "[Echo]") {
		t.Errorf("expected echo response, got: %s", resp)
	}
}
