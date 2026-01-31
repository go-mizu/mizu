//go:build integration

package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	filesession "github.com/go-mizu/mizu/blueprints/bot/pkg/session"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// TestIntegration_AnthropicToolCall verifies the full LLM + tool call pipeline
// using the real Anthropic API. It creates a workspace with test files, sends a
// message asking the bot to list them, and verifies the response mentions the
// files and that a file-based session was persisted.
func TestIntegration_AnthropicToolCall(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	// Create a temp workspace with test files.
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	workDir := filepath.Join(tmpDir, "workspace")
	os.MkdirAll(workDir, 0o755)

	// Write test files for the tool to find.
	os.WriteFile(filepath.Join(workDir, "hello.txt"), []byte("Hello, world!"), 0o644)
	os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Test Project"), 0o644)

	cfg := &config.Config{
		Workspace:    workDir,
		DataDir:      dataDir,
		AnthropicKey: apiKey,
		Telegram: config.TelegramConfig{
			Enabled:  true,
			DMPolicy: "open",
		},
	}

	provider := llm.NewClaude()
	b, err := New(cfg, provider)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Send a message asking to list files, providing the explicit path so the
	// LLM uses the correct directory with the list_files tool.
	resp, err := b.HandleMessage(ctx, &types.InboundMessage{
		ChannelType: types.ChannelWebhook,
		ChannelID:   "test-cli",
		PeerID:      "integration-test",
		PeerName:    "IntegrationTest",
		Content:     fmt.Sprintf("Use the list_files tool to list all files in the directory %s. Just show the filenames.", workDir),
		Origin:      "dm",
	})
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	t.Logf("Response: %s", resp)

	// The response should mention the files we created.
	if !strings.Contains(resp, "hello.txt") && !strings.Contains(resp, "README") {
		t.Errorf("response should mention workspace files, got: %s", resp)
	}

	// Verify file-based session was created.
	fs := b.FileStore()
	if fs == nil {
		t.Fatal("FileStore should not be nil")
	}

	sessions, err := fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}

	if len(sessions) == 0 {
		t.Fatal("expected at least 1 session in file store")
	}

	// Verify transcript has entries.
	entry := sessions[0].Entry
	entries, err := fs.ReadTranscript(entry.SessionID)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	if len(entries) < 3 { // session header + user + assistant
		t.Errorf("expected at least 3 transcript entries, got %d", len(entries))
	}

	t.Logf("Session ID: %s", entry.SessionID)
	t.Logf("Transcript entries: %d", len(entries))
	for i, e := range entries {
		if e.Message != nil {
			t.Logf("  [%d] %s: %v", i, e.Message.Role, truncate(fmt.Sprintf("%v", e.Message.Content), 100))
		} else {
			t.Logf("  [%d] type=%s", i, e.Type)
		}
	}
}

// TestIntegration_TelegramSend verifies that a message can be sent via the
// real Telegram Bot API. It uses the sendMessage endpoint directly.
func TestIntegration_TelegramSend(t *testing.T) {
	botToken := os.Getenv("TELEGRAM_API_KEY")
	if botToken == "" {
		t.Skip("TELEGRAM_API_KEY not set")
	}

	// Send a test message to a known chat.
	chatID := os.Getenv("TELEGRAM_TEST_CHAT_ID")
	if chatID == "" {
		chatID = "1994676962" // Default test user
	}

	// Use Telegram API directly to send.
	payload := map[string]any{
		"chat_id": chatID,
		"text":    fmt.Sprintf("[Integration Test] OpenBot test at %s", time.Now().Format(time.RFC3339)),
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("sendMessage: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sendMessage returned %d", resp.StatusCode)
	}

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.OK {
		t.Fatal("Telegram API returned ok=false")
	}

	t.Logf("Sent message ID: %d to chat %s", result.Result.MessageID, chatID)
}

// TestIntegration_FullPipeline is a full end-to-end test combining the Anthropic
// tool call pipeline, file-based session persistence, and Telegram message delivery.
func TestIntegration_FullPipeline(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	botToken := os.Getenv("TELEGRAM_API_KEY")
	if apiKey == "" || botToken == "" {
		t.Skip("ANTHROPIC_API_KEY and TELEGRAM_API_KEY required")
	}

	chatID := os.Getenv("TELEGRAM_TEST_CHAT_ID")
	if chatID == "" {
		chatID = "1994676962"
	}

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	workDir := filepath.Join(tmpDir, "workspace")
	os.MkdirAll(workDir, 0o755)

	// Create test files.
	for _, name := range []string{"report.pdf", "invoice.pdf", "photo.jpg", "notes.txt"} {
		os.WriteFile(filepath.Join(workDir, name), []byte("test content"), 0o644)
	}

	cfg := &config.Config{
		Workspace:    workDir,
		DataDir:      dataDir,
		AnthropicKey: apiKey,
		Telegram: config.TelegramConfig{
			Enabled:  true,
			BotToken: botToken,
			DMPolicy: "open",
		},
	}

	provider := llm.NewClaude()
	b, err := New(cfg, provider)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Step 1: Send message through bot, providing the explicit path so the
	// LLM uses the correct directory with the list_files tool.
	resp, err := b.HandleMessage(ctx, &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   chatID,
		PeerID:      chatID,
		PeerName:    "IntegrationTest",
		Content:     fmt.Sprintf("Use the list_files tool to list all PDF files in the directory %s. Just give me the filenames.", workDir),
		Origin:      "dm",
	})
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	t.Logf("Bot response: %s", resp)

	// Step 2: Verify tool was called and PDFs found.
	if !strings.Contains(strings.ToLower(resp), "pdf") {
		t.Errorf("response should mention PDF files, got: %s", resp)
	}

	// Step 3: Send the response via Telegram.
	telegramPayload := map[string]any{
		"chat_id": chatID,
		"text":    fmt.Sprintf("OpenBot Integration Test Result:\n\n%s", resp),
	}
	telegramBody, _ := json.Marshal(telegramPayload)
	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	tgResp, err := http.Post(telegramURL, "application/json", strings.NewReader(string(telegramBody)))
	if err != nil {
		t.Fatalf("telegram send: %v", err)
	}
	defer tgResp.Body.Close()

	if tgResp.StatusCode != http.StatusOK {
		t.Fatalf("telegram send returned %d", tgResp.StatusCode)
	}
	t.Logf("Sent result to Telegram chat %s", chatID)

	// Step 4: Verify file-based session.
	fs := b.FileStore()
	if fs == nil {
		t.Fatal("FileStore should not be nil")
	}

	sessions, err := fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}

	if len(sessions) == 0 {
		t.Fatal("expected at least 1 session")
	}

	// Verify session metadata.
	session := sessions[0]
	t.Logf("Session key: %s", session.Key)
	t.Logf("Session ID: %s", session.Entry.SessionID)
	t.Logf("Model: %s", session.Entry.Model)

	// Verify transcript.
	transcriptEntries, err := fs.ReadTranscript(session.Entry.SessionID)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	t.Logf("Transcript entries: %d", len(transcriptEntries))

	// Should have at least: session header, user message, assistant message.
	if len(transcriptEntries) < 3 {
		t.Errorf("expected at least 3 transcript entries, got %d", len(transcriptEntries))
	}

	// Verify sessions.json exists on disk.
	sessionsPath := filepath.Join(dataDir, "agents", "default", "sessions", "sessions.json")
	if _, err := os.Stat(sessionsPath); os.IsNotExist(err) {
		t.Error("sessions.json should exist on disk")
	} else {
		data, _ := os.ReadFile(sessionsPath)
		t.Logf("sessions.json content: %s", string(data))
	}
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Ensure unused imports are referenced (build tag means these are only compiled
// with -tags=integration, so the compiler sees them only when the tag is set).
var _ filesession.FileStore
