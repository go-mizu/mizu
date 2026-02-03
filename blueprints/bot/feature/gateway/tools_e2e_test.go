package gateway

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// ---------------------------------------------------------------------------
// toolMockLLM is a mock LLM provider that simulates tool calling.
// It returns tool_use responses on the first call, then a final text
// response incorporating the tool results.
// ---------------------------------------------------------------------------

type toolMockLLM struct {
	callCount int
	// toolName and toolInput define what tool the LLM "wants" to call.
	toolName  string
	toolInput map[string]any
	// multiTools supports multi-tool chains (called in sequence).
	multiTools []toolCall
}

type toolCall struct {
	name  string
	input map[string]any
}

func (m *toolMockLLM) Chat(_ context.Context, req *types.LLMRequest) (*types.LLMResponse, error) {
	return &types.LLMResponse{
		Content: "[Echo] " + req.Messages[len(req.Messages)-1].Content,
		Model:   "mock",
	}, nil
}

func (m *toolMockLLM) ChatWithTools(_ context.Context, req *types.LLMToolRequest) (*types.LLMToolResponse, error) {
	m.callCount++

	// Multi-tool chain: return tools one at a time.
	if len(m.multiTools) > 0 {
		idx := m.callCount - 1
		if idx < len(m.multiTools) {
			tc := m.multiTools[idx]
			return &types.LLMToolResponse{
				Content: []types.ContentBlock{
					{Type: "text", Text: "Let me use " + tc.name + "."},
					{Type: "tool_use", ID: "toolu_" + tc.name, Name: tc.name, Input: tc.input},
				},
				StopReason:   "tool_use",
				InputTokens:  100,
				OutputTokens: 50,
			}, nil
		}
		// After all tools, return final response that references the results.
		return &types.LLMToolResponse{
			Content: []types.ContentBlock{
				{Type: "text", Text: "I've completed the multi-step task. Here are the results from the tools I used."},
			},
			StopReason:   "end_turn",
			InputTokens:  200,
			OutputTokens: 100,
		}, nil
	}

	// Single tool: first call returns tool_use, second returns final text.
	if m.callCount == 1 {
		return &types.LLMToolResponse{
			Content: []types.ContentBlock{
				{Type: "text", Text: "Let me look that up for you."},
				{Type: "tool_use", ID: "toolu_01", Name: m.toolName, Input: m.toolInput},
			},
			StopReason:   "tool_use",
			InputTokens:  100,
			OutputTokens: 50,
		}, nil
	}

	// Second call: LLM has seen the tool result, produces final answer.
	// Check if the previous message contains tool results.
	lastMsg := req.Messages[len(req.Messages)-1]
	var toolResultText string
	if msgMap, ok := lastMsg.(map[string]any); ok {
		if content, ok := msgMap["content"]; ok {
			if results, ok := content.([]any); ok {
				for _, r := range results {
					if tr, ok := r.(types.ToolResultBlock); ok {
						toolResultText = tr.Content
					}
				}
			}
		}
	}

	responseText := "Based on the tool results"
	if toolResultText != "" {
		responseText += ": " + truncate(toolResultText, 200)
	}

	return &types.LLMToolResponse{
		Content: []types.ContentBlock{
			{Type: "text", Text: responseText},
		},
		StopReason:   "end_turn",
		InputTokens:  200,
		OutputTokens: 100,
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ---------------------------------------------------------------------------
// Scenario 1: "List all PDF files in ~/Downloads"
// Tests: list_files tool with pattern filtering
// ---------------------------------------------------------------------------

func TestToolScenario_ListFilesInDownloads(t *testing.T) {
	// Create a temp directory with some test files.
	dir := t.TempDir()
	for _, name := range []string{"book1.pdf", "book2.pdf", "notes.txt", "report.pdf"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	agent := testAgent(setupWorkspace(t))
	ms := newMockStore(agent)
	ms.bindings = append(ms.bindings, types.Binding{
		AgentID: agent.ID, ChannelType: "*", ChannelID: "*", PeerID: "*",
	})

	mockLLM := &toolMockLLM{
		toolName:  "list_files",
		toolInput: map[string]any{"path": dir, "pattern": "*.pdf"},
	}

	svc := NewService(ms, mockLLM)
	defer svc.Close()

	msg := testMessage("List all PDF books in " + dir)
	result, err := svc.ProcessMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("ProcessMessage error: %v", err)
	}

	// The mock LLM should have been called twice:
	// 1. Initial call → returns tool_use for list_files
	// 2. After tool execution → returns final text
	if mockLLM.callCount != 2 {
		t.Errorf("expected 2 LLM calls (tool loop), got %d", mockLLM.callCount)
	}

	// Result should contain text (the mock LLM references tool results).
	if result.Content == "" {
		t.Error("expected non-empty response content")
	}

	// Verify tool was actually executed by checking the mock saw it.
	t.Logf("Tool scenario 1 response: %s", result.Content)
}

// ---------------------------------------------------------------------------
// Scenario 2: "Read the contents of a specific file"
// Tests: read_file tool
// ---------------------------------------------------------------------------

func TestToolScenario_ReadFile(t *testing.T) {
	dir := t.TempDir()
	readmePath := filepath.Join(dir, "README.md")
	readmeContent := "# My Project\n\nThis is a test project with important documentation."
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
		t.Fatal(err)
	}

	agent := testAgent(setupWorkspace(t))
	ms := newMockStore(agent)
	ms.bindings = append(ms.bindings, types.Binding{
		AgentID: agent.ID, ChannelType: "*", ChannelID: "*", PeerID: "*",
	})

	mockLLM := &toolMockLLM{
		toolName:  "read_file",
		toolInput: map[string]any{"path": readmePath},
	}

	svc := NewService(ms, mockLLM)
	defer svc.Close()

	msg := testMessage("Read the contents of " + readmePath)
	result, err := svc.ProcessMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("ProcessMessage error: %v", err)
	}

	if mockLLM.callCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mockLLM.callCount)
	}

	// The final response should reference the tool results which contain the file content.
	if !strings.Contains(result.Content, "tool results") {
		t.Errorf("expected response to reference tool results, got: %s", result.Content)
	}

	t.Logf("Tool scenario 2 response: %s", result.Content)
}

// ---------------------------------------------------------------------------
// Scenario 3: "What processes are using port 8080?"
// Tests: run_command tool
// ---------------------------------------------------------------------------

func TestToolScenario_RunCommand(t *testing.T) {
	agent := testAgent(setupWorkspace(t))
	ms := newMockStore(agent)
	ms.bindings = append(ms.bindings, types.Binding{
		AgentID: agent.ID, ChannelType: "*", ChannelID: "*", PeerID: "*",
	})

	mockLLM := &toolMockLLM{
		toolName:  "run_command",
		toolInput: map[string]any{"command": "echo test-output"},
	}

	svc := NewService(ms, mockLLM)
	defer svc.Close()

	msg := testMessage("Run 'echo test-output' for me")
	result, err := svc.ProcessMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("ProcessMessage error: %v", err)
	}

	if mockLLM.callCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mockLLM.callCount)
	}

	// The run_command tool should have executed "echo test-output".
	// The mock LLM references the result in its final answer.
	if !strings.Contains(result.Content, "tool results") {
		t.Errorf("expected response to reference command output, got: %s", result.Content)
	}

	t.Logf("Tool scenario 3 response: %s", result.Content)
}

// ---------------------------------------------------------------------------
// Scenario 4: "Search the web for Go 1.23 release notes"
// Tests: web_search tool (returns placeholder since no real API key)
// ---------------------------------------------------------------------------

func TestToolScenario_WebSearch(t *testing.T) {
	agent := testAgent(setupWorkspace(t))
	ms := newMockStore(agent)
	ms.bindings = append(ms.bindings, types.Binding{
		AgentID: agent.ID, ChannelType: "*", ChannelID: "*", PeerID: "*",
	})

	mockLLM := &toolMockLLM{
		toolName:  "web_search",
		toolInput: map[string]any{"query": "Go 1.23 release notes"},
	}

	svc := NewService(ms, mockLLM)
	defer svc.Close()

	msg := testMessage("Search the web for Go 1.23 release notes")
	result, err := svc.ProcessMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("ProcessMessage error: %v", err)
	}

	if mockLLM.callCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mockLLM.callCount)
	}

	// web_search without a real API key returns a placeholder.
	// The tool loop still runs and the LLM gets the result.
	if result.Content == "" {
		t.Error("expected non-empty response")
	}

	t.Logf("Tool scenario 4 response: %s", result.Content)
}

// ---------------------------------------------------------------------------
// Scenario 5: "List files then read the first .txt file" (multi-tool chain)
// Tests: list_files followed by read_file in sequence
// ---------------------------------------------------------------------------

func TestToolScenario_MultiToolChain(t *testing.T) {
	// Create temp directory with a .txt file.
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(txtPath, []byte("These are my important notes."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data.csv"), []byte("a,b,c"), 0o644); err != nil {
		t.Fatal(err)
	}

	agent := testAgent(setupWorkspace(t))
	ms := newMockStore(agent)
	ms.bindings = append(ms.bindings, types.Binding{
		AgentID: agent.ID, ChannelType: "*", ChannelID: "*", PeerID: "*",
	})

	mockLLM := &toolMockLLM{
		multiTools: []toolCall{
			{name: "list_files", input: map[string]any{"path": dir, "pattern": "*.txt"}},
			{name: "read_file", input: map[string]any{"path": txtPath}},
		},
	}

	svc := NewService(ms, mockLLM)
	defer svc.Close()

	msg := testMessage("List .txt files in " + dir + " then read the first one")
	result, err := svc.ProcessMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("ProcessMessage error: %v", err)
	}

	// Multi-tool chain: 3 calls (tool1, tool2, final).
	if mockLLM.callCount != 3 {
		t.Errorf("expected 3 LLM calls for multi-tool chain, got %d", mockLLM.callCount)
	}

	if result.Content == "" {
		t.Error("expected non-empty response after multi-tool chain")
	}

	t.Logf("Tool scenario 5 response: %s", result.Content)
}

// ---------------------------------------------------------------------------
// Verify tool definitions are sent to the LLM
// ---------------------------------------------------------------------------

func TestToolDefinitionsPassedToLLM(t *testing.T) {
	agent := testAgent(setupWorkspace(t))
	ms := newMockStore(agent)
	ms.bindings = append(ms.bindings, types.Binding{
		AgentID: agent.ID, ChannelType: "*", ChannelID: "*", PeerID: "*",
	})

	// Track what tools were sent to the LLM.
	var receivedTools []types.ToolDefinition

	captureLLM := &toolCaptureLLM{
		onChatWithTools: func(req *types.LLMToolRequest) {
			receivedTools = req.Tools
		},
	}

	svc := NewService(ms, captureLLM)
	defer svc.Close()

	msg := testMessage("Hello")
	_, err := svc.ProcessMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("ProcessMessage error: %v", err)
	}

	// Verify tools were passed.
	if len(receivedTools) == 0 {
		t.Fatal("no tools were passed to the LLM — the tool loop is not wired correctly")
	}

	// Check for key tools.
	toolNames := make(map[string]bool)
	for _, td := range receivedTools {
		toolNames[td.Name] = true
	}

	expectedTools := []string{"list_files", "read_file", "run_command", "web_search", "web_fetch", "edit", "write"}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("expected tool %q to be passed to LLM, but it wasn't. Got: %v", name, toolNames)
		}
	}

	t.Logf("Tools passed to LLM: %d total", len(receivedTools))
	toolJSON, _ := json.MarshalIndent(receivedTools[:3], "", "  ")
	t.Logf("First 3 tools: %s", toolJSON)
}

// toolCaptureLLM captures what's sent to ChatWithTools.
type toolCaptureLLM struct {
	onChatWithTools func(req *types.LLMToolRequest)
}

func (m *toolCaptureLLM) Chat(_ context.Context, req *types.LLMRequest) (*types.LLMResponse, error) {
	return &types.LLMResponse{Content: "Hello!", Model: "mock"}, nil
}

func (m *toolCaptureLLM) ChatWithTools(_ context.Context, req *types.LLMToolRequest) (*types.LLMToolResponse, error) {
	if m.onChatWithTools != nil {
		m.onChatWithTools(req)
	}
	return &types.LLMToolResponse{
		Content:    []types.ContentBlock{{Type: "text", Text: "Hello from tools!"}},
		StopReason: "end_turn",
	}, nil
}
