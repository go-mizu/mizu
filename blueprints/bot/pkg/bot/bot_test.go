package bot

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// capturingLLM is a test LLM that captures requests and delegates to Echo.
type capturingLLM struct {
	onChat func(req *types.LLMRequest)
}

func (c *capturingLLM) Chat(ctx context.Context, req *types.LLMRequest) (*types.LLMResponse, error) {
	if c.onChat != nil {
		c.onChat(req)
	}
	return (&llm.Echo{}).Chat(ctx, req)
}

// testConfig returns a Config for testing with an isolated temp dir.
func testConfig(t *testing.T) *config.Config {
	t.Helper()
	ws := setupTestWorkspace(t)
	dataDir := t.TempDir()
	return &config.Config{
		Workspace: ws,
		DataDir:   dataDir,
		Telegram: config.TelegramConfig{
			Enabled:  true,
			DMPolicy: "allowlist",
		},
	}
}

// testInboundMessage returns a basic inbound message for testing.
func testInboundMessage(content string) *types.InboundMessage {
	return &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-test",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     content,
		Origin:      "dm",
	}
}

func TestBot_HandleMessage_BasicFlow(t *testing.T) {
	cfg := testConfig(t)
	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	resp, err := b.HandleMessage(ctx, testInboundMessage("Hello, bot!"))
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// Echo provider returns "[Echo] You said: <last message>".
	if !strings.Contains(resp, "[Echo]") {
		t.Errorf("expected echo response, got: %s", resp)
	}
	if !strings.Contains(resp, "Hello, bot!") {
		t.Errorf("expected echo of user message, got: %s", resp)
	}
}

func TestBot_HandleMessage_Allowlist_Allowed(t *testing.T) {
	cfg := testConfig(t)
	cfg.Telegram.AllowFrom = []string{"user-1"}

	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	resp, err := b.HandleMessage(ctx, testInboundMessage("Hello from allowed user"))
	if err != nil {
		t.Fatalf("HandleMessage should succeed for allowed user: %v", err)
	}

	if !strings.Contains(resp, "[Echo]") {
		t.Errorf("expected echo response, got: %s", resp)
	}
}

func TestBot_HandleMessage_Allowlist_Blocked(t *testing.T) {
	cfg := testConfig(t)
	cfg.Telegram.AllowFrom = []string{"other-user"}

	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	_, err = b.HandleMessage(ctx, testInboundMessage("Hello from blocked user"))
	if err == nil {
		t.Fatal("expected error for blocked user")
	}

	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("error should mention 'not allowed', got: %v", err)
	}
}

func TestBot_HandleMessage_SessionPersistence(t *testing.T) {
	cfg := testConfig(t)
	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()

	// Send first message.
	_, err = b.HandleMessage(ctx, testInboundMessage("First message"))
	if err != nil {
		t.Fatalf("first message: %v", err)
	}

	// Send second message from the same user.
	resp, err := b.HandleMessage(ctx, testInboundMessage("Second message"))
	if err != nil {
		t.Fatalf("second message: %v", err)
	}

	// The second response should echo the second message.
	if !strings.Contains(resp, "Second message") {
		t.Errorf("expected echo of second message, got: %s", resp)
	}

	// Verify only one session was created by listing sessions.
	sessions, err := b.store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	activeCount := 0
	for _, s := range sessions {
		if s.Status == "active" && s.PeerID == "user-1" {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("expected 1 active session, got %d", activeCount)
	}
}

func TestBot_HandleMessage_SlashCommand(t *testing.T) {
	cfg := testConfig(t)
	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	resp, err := b.HandleMessage(ctx, testInboundMessage("/help"))
	if err != nil {
		t.Fatalf("HandleMessage /help: %v", err)
	}

	if !strings.Contains(resp, "Available commands") {
		t.Errorf("expected commands list, got: %s", resp)
	}
	if !strings.Contains(resp, "/new") {
		t.Error("help should list /new command")
	}
	if !strings.Contains(resp, "/help") {
		t.Error("help should list /help command")
	}
}

func TestBot_FullRoundTrip_WithMemoryAndContext(t *testing.T) {
	ws := setupTestWorkspace(t)
	dataDir := t.TempDir()

	// Add skills directory with a SKILL.md.
	skillDir := filepath.Join(ws, "skills", "test-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	skillContent := "---\nname: test-skill\ndescription: A test skill\n---\n# Test Skill\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	// Add an indexable file for memory search.
	goContent := "package main\n\n// GreetUser prints a greeting message to the console.\nfunc GreetUser(name string) {\n\tfmt.Println(\"Hello, \" + name)\n}\n"
	if err := os.WriteFile(filepath.Join(ws, "main.go"), []byte(goContent), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	// Add a notes file for broader memory coverage.
	mdContent := "# Project Notes\n\nThis project implements a greeting service.\nThe primary function is GreetUser which takes a name parameter.\n"
	if err := os.WriteFile(filepath.Join(ws, "NOTES.md"), []byte(mdContent), 0o644); err != nil {
		t.Fatalf("write NOTES.md: %v", err)
	}

	cfg := &config.Config{
		Workspace: ws,
		DataDir:   dataDir,
		Telegram: config.TelegramConfig{
			Enabled:  true,
			DMPolicy: "allowlist",
		},
	}

	var capturedPrompt string
	provider := &capturingLLM{
		onChat: func(req *types.LLMRequest) {
			capturedPrompt = req.SystemPrompt
		},
	}

	b, err := New(cfg, provider)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	// Use "GreetUser" as the message so the FTS query matches indexed content.
	resp, err := b.HandleMessage(ctx, testInboundMessage("GreetUser"))
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// Verify the LLM response was returned.
	if !strings.Contains(resp, "[Echo]") {
		t.Errorf("expected echo response, got: %s", resp)
	}

	// Verify the system prompt contains workspace context.
	if !strings.Contains(capturedPrompt, "You are a personal assistant") {
		t.Error("system prompt should contain identity section")
	}

	// Verify skills section is in the prompt.
	if !strings.Contains(capturedPrompt, "test-skill") {
		t.Error("system prompt should contain skills section")
	}

	// Verify memory results are in the prompt (if memory was indexed).
	if b.memory != nil {
		if !strings.Contains(capturedPrompt, "# Relevant Context") {
			t.Errorf("system prompt should contain memory context section; prompt: %s", capturedPrompt)
		}
		if !strings.Contains(capturedPrompt, "GreetUser") {
			t.Errorf("system prompt should contain GreetUser from memory search; prompt: %s", capturedPrompt)
		}
	}
}
