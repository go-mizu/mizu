package bot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// scenarioProvider implements llm.ToolProvider with dynamic per-call handlers.
// Each handler function receives the request and can inspect previous tool
// results before returning a response.
type scenarioProvider struct {
	handlers []func(req *types.LLMToolRequest) *types.LLMToolResponse
	calls    int
}

func (s *scenarioProvider) Chat(ctx context.Context, req *types.LLMRequest) (*types.LLMResponse, error) {
	return &types.LLMResponse{Content: "[fallback]", Model: "test"}, nil
}

func (s *scenarioProvider) ChatWithTools(ctx context.Context, req *types.LLMToolRequest) (*types.LLMToolResponse, error) {
	if s.calls >= len(s.handlers) {
		return &types.LLMToolResponse{
			Content:    []types.ContentBlock{{Type: "text", Text: "done"}},
			StopReason: "end_turn",
		}, nil
	}
	handler := s.handlers[s.calls]
	s.calls++
	return handler(req), nil
}

// createTempFiles creates count files with the given extension in dir.
// Returns a slice of created filenames (base names only).
func createTempFiles(t *testing.T, dir string, ext string, count int) []string {
	t.Helper()
	names := make([]string, count)
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("file%d%s", i+1, ext)
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(fmt.Sprintf("content of %s", name)), 0o644); err != nil {
			t.Fatalf("create temp file %s: %v", path, err)
		}
		names[i] = name
	}
	return names
}

// TestScenario_ListPDFs simulates the LLM requesting a list_files tool call
// to find PDF files in a directory, then producing a summary response.
func TestScenario_ListPDFs(t *testing.T) {
	tmpDir := t.TempDir()
	pdfNames := createTempFiles(t, tmpDir, ".pdf", 3)
	createTempFiles(t, tmpDir, ".txt", 2) // noise files

	provider := &scenarioProvider{
		handlers: []func(req *types.LLMToolRequest) *types.LLMToolResponse{
			// First call: LLM requests list_files tool.
			func(req *types.LLMToolRequest) *types.LLMToolResponse {
				return &types.LLMToolResponse{
					Content: []types.ContentBlock{
						{
							Type:  "tool_use",
							ID:    "call_list",
							Name:  "list_files",
							Input: map[string]any{"path": tmpDir, "pattern": "*.pdf"},
						},
					},
					StopReason: "tool_use",
				}
			},
			// Second call: LLM receives tool result and produces final text.
			func(req *types.LLMToolRequest) *types.LLMToolResponse {
				return &types.LLMToolResponse{
					Content: []types.ContentBlock{
						{
							Type: "text",
							Text: fmt.Sprintf("Found 3 PDFs:\n- %s\n- %s\n- %s",
								pdfNames[0], pdfNames[1], pdfNames[2]),
						},
					},
					StopReason: "end_turn",
				}
			},
		},
	}

	cfg := testConfig(t)
	b, err := New(cfg, provider)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	resp, err := b.HandleMessage(ctx, &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-test",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     fmt.Sprintf("List all PDF files in %s", tmpDir),
		Origin:      "dm",
	})
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// Verify the response mentions 3 PDFs.
	if !strings.Contains(resp, "3 PDF") {
		t.Errorf("expected response to mention '3 PDF', got: %s", resp)
	}

	// Verify all PDF filenames are present.
	for _, name := range pdfNames {
		if !strings.Contains(resp, name) {
			t.Errorf("expected response to contain %q, got: %s", name, resp)
		}
	}

	// Verify exactly 2 ChatWithTools calls were made.
	if provider.calls != 2 {
		t.Errorf("expected 2 ChatWithTools calls, got %d", provider.calls)
	}
}

// TestScenario_RunCommand_FindPDFs simulates the LLM using run_command
// to execute a find command for PDF files.
func TestScenario_RunCommand_FindPDFs(t *testing.T) {
	tmpDir := t.TempDir()
	pdfNames := createTempFiles(t, tmpDir, ".pdf", 2)

	provider := &scenarioProvider{
		handlers: []func(req *types.LLMToolRequest) *types.LLMToolResponse{
			// First call: LLM requests run_command.
			func(req *types.LLMToolRequest) *types.LLMToolResponse {
				return &types.LLMToolResponse{
					Content: []types.ContentBlock{
						{
							Type: "tool_use",
							ID:   "call_find",
							Name: "run_command",
							Input: map[string]any{
								"command": fmt.Sprintf("find %s -name '*.pdf' | sort", tmpDir),
							},
						},
					},
					StopReason: "tool_use",
				}
			},
			// Second call: LLM receives command output and summarises.
			func(req *types.LLMToolRequest) *types.LLMToolResponse {
				// Verify that the tool result from the previous iteration
				// actually contains the PDF file paths.
				lastMsg, ok := req.Messages[len(req.Messages)-1].(map[string]any)
				if !ok {
					return &types.LLMToolResponse{
						Content:    []types.ContentBlock{{Type: "text", Text: "error: unexpected message format"}},
						StopReason: "end_turn",
					}
				}
				content, _ := lastMsg["content"].([]any)
				var toolOutput string
				for _, c := range content {
					if block, ok := c.(types.ToolResultBlock); ok {
						toolOutput = block.Content
					}
				}

				// Build response confirming find results.
				return &types.LLMToolResponse{
					Content: []types.ContentBlock{
						{
							Type: "text",
							Text: fmt.Sprintf("Found PDFs using find command:\n%s", toolOutput),
						},
					},
					StopReason: "end_turn",
				}
			},
		},
	}

	cfg := testConfig(t)
	b, err := New(cfg, provider)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	resp, err := b.HandleMessage(ctx, &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-test",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     fmt.Sprintf("Find all PDFs in %s", tmpDir),
		Origin:      "dm",
	})
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// Verify that the actual find command was executed and results are present.
	for _, name := range pdfNames {
		if !strings.Contains(resp, name) {
			t.Errorf("expected response to contain %q, got: %s", name, resp)
		}
	}

	if provider.calls != 2 {
		t.Errorf("expected 2 ChatWithTools calls, got %d", provider.calls)
	}
}

// TestScenario_ReadFile simulates the LLM requesting to read a file
// and then producing a summary of its contents.
func TestScenario_ReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	fileContent := "Hello, this is the secret content of the test file.\nLine two.\nLine three.\n"
	filePath := filepath.Join(tmpDir, "testdata.txt")
	if err := os.WriteFile(filePath, []byte(fileContent), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	provider := &scenarioProvider{
		handlers: []func(req *types.LLMToolRequest) *types.LLMToolResponse{
			// First call: LLM requests read_file.
			func(req *types.LLMToolRequest) *types.LLMToolResponse {
				return &types.LLMToolResponse{
					Content: []types.ContentBlock{
						{
							Type:  "tool_use",
							ID:    "call_read",
							Name:  "read_file",
							Input: map[string]any{"path": filePath},
						},
					},
					StopReason: "tool_use",
				}
			},
			// Second call: LLM receives file content and produces summary.
			func(req *types.LLMToolRequest) *types.LLMToolResponse {
				// Extract the tool result to verify it contains the file content.
				lastMsg, ok := req.Messages[len(req.Messages)-1].(map[string]any)
				if !ok {
					return &types.LLMToolResponse{
						Content:    []types.ContentBlock{{Type: "text", Text: "error: bad message"}},
						StopReason: "end_turn",
					}
				}
				content, _ := lastMsg["content"].([]any)
				var toolOutput string
				for _, c := range content {
					if block, ok := c.(types.ToolResultBlock); ok {
						toolOutput = block.Content
					}
				}

				summary := fmt.Sprintf("File summary: The file contains %d lines. Content: %s",
					strings.Count(toolOutput, "\n"), toolOutput)
				return &types.LLMToolResponse{
					Content: []types.ContentBlock{
						{Type: "text", Text: summary},
					},
					StopReason: "end_turn",
				}
			},
		},
	}

	cfg := testConfig(t)
	b, err := New(cfg, provider)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	resp, err := b.HandleMessage(ctx, &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-test",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     fmt.Sprintf("Read the file at %s and summarise it", filePath),
		Origin:      "dm",
	})
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// Verify the response contains the actual file content.
	if !strings.Contains(resp, "secret content") {
		t.Errorf("expected response to contain file content 'secret content', got: %s", resp)
	}
	if !strings.Contains(resp, "Line two") {
		t.Errorf("expected response to contain 'Line two', got: %s", resp)
	}

	if provider.calls != 2 {
		t.Errorf("expected 2 ChatWithTools calls, got %d", provider.calls)
	}
}

// TestScenario_MultiToolConversation simulates a multi-step conversation
// where the LLM first lists files, then reads one of them, then produces
// a summary. This tests three iterations of the tool loop.
func TestScenario_MultiToolConversation(t *testing.T) {
	tmpDir := t.TempDir()
	// Create files with known content.
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Project Readme\nThis is the project readme.\n"), 0o644); err != nil {
		t.Fatalf("write readme.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte("key: value\nport: 8080\n"), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("Important notes here.\n"), 0o644); err != nil {
		t.Fatalf("write notes.txt: %v", err)
	}

	readmePath := filepath.Join(tmpDir, "readme.md")

	provider := &scenarioProvider{
		handlers: []func(req *types.LLMToolRequest) *types.LLMToolResponse{
			// Step 1: LLM requests list_files.
			func(req *types.LLMToolRequest) *types.LLMToolResponse {
				return &types.LLMToolResponse{
					Content: []types.ContentBlock{
						{
							Type:  "tool_use",
							ID:    "call_list",
							Name:  "list_files",
							Input: map[string]any{"path": tmpDir},
						},
					},
					StopReason: "tool_use",
				}
			},
			// Step 2: LLM sees the file listing and requests read_file on readme.md.
			func(req *types.LLMToolRequest) *types.LLMToolResponse {
				// Verify that the tool result from step 1 contains the file listing.
				lastMsg, ok := req.Messages[len(req.Messages)-1].(map[string]any)
				if ok {
					content, _ := lastMsg["content"].([]any)
					for _, c := range content {
						if block, ok := c.(types.ToolResultBlock); ok {
							if !strings.Contains(block.Content, "readme.md") {
								t.Errorf("step 2: tool result should contain 'readme.md', got: %s", block.Content)
							}
						}
					}
				}

				return &types.LLMToolResponse{
					Content: []types.ContentBlock{
						{
							Type:  "tool_use",
							ID:    "call_read",
							Name:  "read_file",
							Input: map[string]any{"path": readmePath},
						},
					},
					StopReason: "tool_use",
				}
			},
			// Step 3: LLM receives file content and produces final summary.
			func(req *types.LLMToolRequest) *types.LLMToolResponse {
				// Verify the read_file result contains the readme content.
				lastMsg, ok := req.Messages[len(req.Messages)-1].(map[string]any)
				if ok {
					content, _ := lastMsg["content"].([]any)
					for _, c := range content {
						if block, ok := c.(types.ToolResultBlock); ok {
							if !strings.Contains(block.Content, "Project Readme") {
								t.Errorf("step 3: tool result should contain 'Project Readme', got: %s", block.Content)
							}
						}
					}
				}

				return &types.LLMToolResponse{
					Content: []types.ContentBlock{
						{
							Type: "text",
							Text: "I found 3 files in the directory. The readme.md contains: # Project Readme - This is the project readme.",
						},
					},
					StopReason: "end_turn",
				}
			},
		},
	}

	cfg := testConfig(t)
	b, err := New(cfg, provider)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	resp, err := b.HandleMessage(ctx, &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-test",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     fmt.Sprintf("List files in %s and read the readme", tmpDir),
		Origin:      "dm",
	})
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// Verify the final response.
	if !strings.Contains(resp, "3 files") {
		t.Errorf("expected response to mention '3 files', got: %s", resp)
	}
	if !strings.Contains(resp, "Project Readme") {
		t.Errorf("expected response to contain 'Project Readme', got: %s", resp)
	}

	// Verify exactly 3 ChatWithTools calls were made (list, read, summarise).
	if provider.calls != 3 {
		t.Errorf("expected 3 ChatWithTools calls, got %d", provider.calls)
	}
}
