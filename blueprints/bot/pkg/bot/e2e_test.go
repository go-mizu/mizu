package bot

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	filesession "github.com/go-mizu/mizu/blueprints/bot/pkg/session"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// TestE2E_FileSessionCreation verifies that when a message is processed,
// both SQLite and file-based sessions are created.
func TestE2E_FileSessionCreation(t *testing.T) {
	cfg := testConfig(t)
	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	_, err = b.HandleMessage(ctx, testInboundMessage("Hello E2E"))
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// Verify SQLite session was created.
	sqlSessions, err := b.store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list SQLite sessions: %v", err)
	}
	var sqlSessionCount int
	for _, s := range sqlSessions {
		if s.PeerID == "user-1" && s.Status == "active" {
			sqlSessionCount++
		}
	}
	if sqlSessionCount != 1 {
		t.Errorf("expected 1 active SQLite session for user-1, got %d", sqlSessionCount)
	}

	// Verify file store was initialized.
	fs := b.FileStore()
	if fs == nil {
		t.Fatal("expected FileStore to be initialized, got nil")
	}

	// Verify sessions.json was created with the correct session key.
	sessionsPath := filepath.Join(cfg.DataDir, "agents", "default", "sessions", "sessions.json")
	if _, err := os.Stat(sessionsPath); os.IsNotExist(err) {
		t.Fatal("sessions.json was not created")
	}

	// Read and parse sessions.json to verify the key.
	data, err := os.ReadFile(sessionsPath)
	if err != nil {
		t.Fatalf("read sessions.json: %v", err)
	}
	var index map[string]*filesession.Entry
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatalf("parse sessions.json: %v", err)
	}

	// DM key should be "agent:default:<peerID>".
	expectedKey := "agent:default:user-1"
	entry, ok := index[expectedKey]
	if !ok {
		t.Fatalf("sessions.json missing key %q; keys: %v", expectedKey, keys(index))
	}

	if entry.SessionID == "" {
		t.Error("session entry has empty sessionId")
	}
	if entry.Status != "active" {
		t.Errorf("expected status 'active', got %q", entry.Status)
	}

	// Verify a JSONL transcript file was created.
	transcriptPath := filepath.Join(cfg.DataDir, "agents", "default", "sessions", entry.SessionID+".jsonl")
	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		t.Fatal("JSONL transcript file was not created")
	}

	// Verify the JSONL has a session header + user message + assistant message.
	entries, err := fs.ReadTranscript(entry.SessionID)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	// Expect at least 3 entries: session header, user message, assistant message.
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 transcript entries, got %d", len(entries))
	}

	if entries[0].Type != "session" {
		t.Errorf("first entry should be type 'session', got %q", entries[0].Type)
	}
	if entries[1].Type != "message" || entries[1].Message == nil || entries[1].Message.Role != "user" {
		t.Errorf("second entry should be user message, got type=%q", entries[1].Type)
	}
	if entries[2].Type != "message" || entries[2].Message == nil || entries[2].Message.Role != "assistant" {
		t.Errorf("third entry should be assistant message, got type=%q", entries[2].Type)
	}
}

// TestE2E_SessionKeyDerivation verifies correct session keys for DM vs group.
func TestE2E_SessionKeyDerivation(t *testing.T) {
	cfg := testConfig(t)
	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	fs := b.FileStore()
	if fs == nil {
		t.Fatal("expected FileStore to be initialized")
	}

	// Send a DM message (Origin: "dm", PeerID: "user-123").
	dmMsg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-dm",
		PeerID:      "user-123",
		PeerName:    "DMUser",
		Content:     "DM message",
		Origin:      "dm",
	}
	_, err = b.HandleMessage(ctx, dmMsg)
	if err != nil {
		t.Fatalf("HandleMessage DM: %v", err)
	}

	// Verify DM session key is "agent:default:user-123".
	sessions, err := fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}

	dmKeyFound := false
	for _, s := range sessions {
		if s.Key == "agent:default:user-123" {
			dmKeyFound = true
			break
		}
	}
	if !dmKeyFound {
		t.Errorf("expected DM session key 'agent:default:user-123', keys found: %v", sessionKeys(sessions))
	}

	// Send a group message (Origin: "group", GroupID: "group-456").
	groupMsg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-group",
		PeerID:      "user-789",
		PeerName:    "GroupUser",
		Content:     "Group message",
		Origin:      "group",
		GroupID:     "group-456",
	}
	_, err = b.HandleMessage(ctx, groupMsg)
	if err != nil {
		t.Fatalf("HandleMessage group: %v", err)
	}

	// Verify group session key is "agent:default:telegram:group:group-456".
	sessions, err = fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions after group: %v", err)
	}

	groupKeyFound := false
	for _, s := range sessions {
		if s.Key == "agent:default:telegram:group:group-456" {
			groupKeyFound = true
			break
		}
	}
	if !groupKeyFound {
		t.Errorf("expected group session key 'agent:default:telegram:group:group-456', keys found: %v", sessionKeys(sessions))
	}
}

// TestE2E_SessionTranscript verifies JSONL contains correct entries with matching content.
func TestE2E_SessionTranscript(t *testing.T) {
	cfg := testConfig(t)
	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	fs := b.FileStore()
	if fs == nil {
		t.Fatal("expected FileStore to be initialized")
	}

	userContent := "Tell me about Go testing"
	resp, err := b.HandleMessage(ctx, testInboundMessage(userContent))
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// Look up the session by the DM key.
	expectedKey := "agent:default:user-1"
	index, err := fs.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	entry, ok := index[expectedKey]
	if !ok {
		t.Fatalf("session key %q not found in index", expectedKey)
	}

	// Read the transcript.
	entries, err := fs.ReadTranscript(entry.SessionID)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	// Verify session header.
	if len(entries) < 1 || entries[0].Type != "session" {
		t.Fatal("missing session header in transcript")
	}
	if entries[0].Version != 2 {
		t.Errorf("expected session version 2, got %d", entries[0].Version)
	}

	// Find user and assistant message entries.
	var userEntry, assistantEntry *filesession.TranscriptEntry
	for i := range entries {
		if entries[i].Type == "message" && entries[i].Message != nil {
			if entries[i].Message.Role == "user" && userEntry == nil {
				userEntry = &entries[i]
			}
			if entries[i].Message.Role == "assistant" && assistantEntry == nil {
				assistantEntry = &entries[i]
			}
		}
	}

	if userEntry == nil {
		t.Fatal("no user message found in transcript")
	}
	if assistantEntry == nil {
		t.Fatal("no assistant message found in transcript")
	}

	// Verify user message content matches.
	userText, ok := userEntry.Message.Content.(string)
	if !ok {
		t.Fatalf("user message content is not a string: %T", userEntry.Message.Content)
	}
	if userText != userContent {
		t.Errorf("user message content = %q, want %q", userText, userContent)
	}

	// Verify assistant message content matches the response.
	assistantText, ok := assistantEntry.Message.Content.(string)
	if !ok {
		t.Fatalf("assistant message content is not a string: %T", assistantEntry.Message.Content)
	}
	if assistantText != resp {
		t.Errorf("assistant message content = %q, want %q", assistantText, resp)
	}

	// Verify timestamps are present.
	if userEntry.Timestamp == "" {
		t.Error("user message entry has empty timestamp")
	}
	if assistantEntry.Timestamp == "" {
		t.Error("assistant message entry has empty timestamp")
	}
}

// TestE2E_MultiMessageSession verifies messages accumulate in the same session.
func TestE2E_MultiMessageSession(t *testing.T) {
	cfg := testConfig(t)
	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	fs := b.FileStore()
	if fs == nil {
		t.Fatal("expected FileStore to be initialized")
	}

	messages := []string{"First message", "Second message", "Third message"}
	for _, msg := range messages {
		_, err := b.HandleMessage(ctx, testInboundMessage(msg))
		if err != nil {
			t.Fatalf("HandleMessage(%q): %v", msg, err)
		}
	}

	// Verify only one session was created in SQLite.
	sqlSessions, err := b.store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list SQLite sessions: %v", err)
	}
	activeCount := 0
	for _, s := range sqlSessions {
		if s.PeerID == "user-1" && s.Status == "active" {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("expected 1 active SQLite session, got %d", activeCount)
	}

	// Verify only one session in file store.
	fsSessions, err := fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}

	expectedKey := "agent:default:user-1"
	var matchingCount int
	for _, s := range fsSessions {
		if s.Key == expectedKey {
			matchingCount++
		}
	}
	if matchingCount != 1 {
		t.Errorf("expected 1 file store session for key %q, got %d", expectedKey, matchingCount)
	}

	// Read the transcript and count message entries.
	index, err := fs.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	entry := index[expectedKey]
	if entry == nil {
		t.Fatalf("session entry not found for key %q", expectedKey)
	}

	entries, err := fs.ReadTranscript(entry.SessionID)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	var userMsgCount, assistantMsgCount int
	for _, e := range entries {
		if e.Type == "message" && e.Message != nil {
			switch e.Message.Role {
			case "user":
				userMsgCount++
			case "assistant":
				assistantMsgCount++
			}
		}
	}

	if userMsgCount != 3 {
		t.Errorf("expected 3 user messages in transcript, got %d", userMsgCount)
	}
	if assistantMsgCount != 3 {
		t.Errorf("expected 3 assistant messages in transcript, got %d", assistantMsgCount)
	}

	// Verify total entries: 1 session header + 3 user + 3 assistant = 7.
	expectedTotal := 7
	if len(entries) != expectedTotal {
		t.Errorf("expected %d total transcript entries, got %d", expectedTotal, len(entries))
	}
}

// TestE2E_SessionReset verifies /new command creates a new session.
func TestE2E_SessionReset(t *testing.T) {
	cfg := testConfig(t)
	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	fs := b.FileStore()
	if fs == nil {
		t.Fatal("expected FileStore to be initialized")
	}

	// Send an initial message to create a session.
	_, err = b.HandleMessage(ctx, testInboundMessage("First session message"))
	if err != nil {
		t.Fatalf("HandleMessage first: %v", err)
	}

	// Capture the initial session ID from the file store.
	expectedKey := "agent:default:user-1"
	index, err := fs.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	firstEntry := index[expectedKey]
	if firstEntry == nil {
		t.Fatalf("session entry not found for key %q", expectedKey)
	}
	firstSessionID := firstEntry.SessionID

	// Send /new to reset the session.
	resetResp, err := b.HandleMessage(ctx, testInboundMessage("/new"))
	if err != nil {
		t.Fatalf("HandleMessage /new: %v", err)
	}
	if !strings.Contains(resetResp, "New session") {
		t.Logf("/new response: %s", resetResp) // Log but don't fail; response text may vary.
	}

	// Reload the index and verify the session ID changed.
	index, err = fs.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex after reset: %v", err)
	}
	resetEntry := index[expectedKey]
	if resetEntry == nil {
		t.Fatalf("session entry not found after reset for key %q", expectedKey)
	}
	secondSessionID := resetEntry.SessionID

	if firstSessionID == secondSessionID {
		t.Errorf("expected different session ID after /new, both are %q", firstSessionID)
	}

	// Send another message in the new session.
	_, err = b.HandleMessage(ctx, testInboundMessage("Second session message"))
	if err != nil {
		t.Fatalf("HandleMessage second: %v", err)
	}

	// Verify the old transcript file still exists.
	oldTranscript, err := fs.ReadTranscript(firstSessionID)
	if err != nil {
		t.Fatalf("ReadTranscript old: %v", err)
	}
	if len(oldTranscript) == 0 {
		t.Error("old transcript should not be empty")
	}

	// Verify the new transcript file was created with the new session.
	newTranscript, err := fs.ReadTranscript(secondSessionID)
	if err != nil {
		t.Fatalf("ReadTranscript new: %v", err)
	}
	// New transcript should have session header + user message + assistant message.
	if len(newTranscript) < 3 {
		t.Errorf("expected at least 3 entries in new transcript, got %d", len(newTranscript))
	}
}

// TestE2E_CLISessionsList verifies ListSessions reads from the file store correctly.
func TestE2E_CLISessionsList(t *testing.T) {
	// Create a standalone file store in a temp directory.
	tmpDir := t.TempDir()
	fs, err := filesession.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	// Create several sessions with different keys.
	keys := []struct {
		key         string
		displayName string
		chatType    string
		channel     string
	}{
		{"agent:default:alice", "Alice", "direct", "telegram"},
		{"agent:default:bob", "Bob", "direct", "telegram"},
		{"agent:default:telegram:group:devs", "DevGroup", "group", "telegram"},
	}

	for _, k := range keys {
		entry, isNew, err := fs.GetOrCreate(k.key, k.displayName, k.chatType, k.channel)
		if err != nil {
			t.Fatalf("GetOrCreate(%s): %v", k.key, err)
		}
		if !isNew {
			t.Errorf("expected new session for %q", k.key)
		}
		if entry.SessionID == "" {
			t.Errorf("empty sessionId for %q", k.key)
		}
	}

	// List and verify.
	sessions, err := fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}

	// Verify all expected keys are present.
	foundKeys := make(map[string]bool)
	for _, s := range sessions {
		foundKeys[s.Key] = true
	}
	for _, k := range keys {
		if !foundKeys[k.key] {
			t.Errorf("session key %q not found in listing", k.key)
		}
	}

	// Verify sessions are sorted by updatedAt descending (most recent first).
	for i := 1; i < len(sessions); i++ {
		if sessions[i].Entry.UpdatedAt > sessions[i-1].Entry.UpdatedAt {
			t.Errorf("sessions not sorted by updatedAt desc: index %d (%d) > index %d (%d)",
				i, sessions[i].Entry.UpdatedAt, i-1, sessions[i-1].Entry.UpdatedAt)
		}
	}

	// Verify display names.
	for _, s := range sessions {
		switch s.Key {
		case "agent:default:alice":
			if s.Entry.DisplayName != "Alice" {
				t.Errorf("expected displayName 'Alice', got %q", s.Entry.DisplayName)
			}
		case "agent:default:bob":
			if s.Entry.DisplayName != "Bob" {
				t.Errorf("expected displayName 'Bob', got %q", s.Entry.DisplayName)
			}
		case "agent:default:telegram:group:devs":
			if s.Entry.DisplayName != "DevGroup" {
				t.Errorf("expected displayName 'DevGroup', got %q", s.Entry.DisplayName)
			}
			if s.Entry.ChatType != "group" {
				t.Errorf("expected chatType 'group', got %q", s.Entry.ChatType)
			}
		}
	}
}

// TestE2E_TokenUsageTracking verifies token counts are updated in the file store.
func TestE2E_TokenUsageTracking(t *testing.T) {
	// Create a standalone file store to test token tracking directly.
	tmpDir := t.TempDir()
	fs, err := filesession.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	key := "agent:default:user-tokens"
	entry, isNew, err := fs.GetOrCreate(key, "TokenUser", "direct", "telegram")
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if !isNew {
		t.Fatal("expected new session")
	}

	// Verify initial token counts are zero.
	if entry.InputTokens != 0 || entry.OutputTokens != 0 || entry.TotalTokens != 0 {
		t.Errorf("expected initial token counts to be 0, got input=%d output=%d total=%d",
			entry.InputTokens, entry.OutputTokens, entry.TotalTokens)
	}

	// Update token usage: first exchange.
	err = fs.UpdateTokenUsage(key, 100, 50)
	if err != nil {
		t.Fatalf("UpdateTokenUsage first: %v", err)
	}

	// Verify after first update.
	index, err := fs.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	entry = index[key]
	if entry.InputTokens != 100 {
		t.Errorf("expected inputTokens=100, got %d", entry.InputTokens)
	}
	if entry.OutputTokens != 50 {
		t.Errorf("expected outputTokens=50, got %d", entry.OutputTokens)
	}
	if entry.TotalTokens != 150 {
		t.Errorf("expected totalTokens=150, got %d", entry.TotalTokens)
	}

	// Update token usage: second exchange (should accumulate).
	err = fs.UpdateTokenUsage(key, 200, 80)
	if err != nil {
		t.Fatalf("UpdateTokenUsage second: %v", err)
	}

	index, err = fs.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex after second: %v", err)
	}
	entry = index[key]
	if entry.InputTokens != 300 {
		t.Errorf("expected accumulated inputTokens=300, got %d", entry.InputTokens)
	}
	if entry.OutputTokens != 130 {
		t.Errorf("expected accumulated outputTokens=130, got %d", entry.OutputTokens)
	}
	if entry.TotalTokens != 430 {
		t.Errorf("expected accumulated totalTokens=430, got %d", entry.TotalTokens)
	}

	// Verify UpdateTokenUsage fails for nonexistent session.
	err = fs.UpdateTokenUsage("nonexistent:key", 10, 10)
	if err == nil {
		t.Error("expected error for nonexistent session key")
	}
}

// TestE2E_TelegramChatIDPropagation verifies chat_id (ChannelID) flows through the pipeline.
func TestE2E_TelegramChatIDPropagation(t *testing.T) {
	cfg := testConfig(t)
	b, err := New(cfg, &llm.Echo{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	ctx := context.Background()
	fs := b.FileStore()
	if fs == nil {
		t.Fatal("expected FileStore to be initialized")
	}

	// Create an InboundMessage with a specific ChannelID (simulating Telegram chat_id).
	chatID := "telegram-chat-12345"
	msg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   chatID,
		PeerID:      "tg-user-42",
		PeerName:    "TGUser",
		Content:     "Hello from Telegram",
		Origin:      "dm",
	}

	_, err = b.HandleMessage(ctx, msg)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// Verify the SQLite session was created with the correct channel ID.
	sqlSessions, err := b.store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list SQLite sessions: %v", err)
	}

	var found bool
	for _, s := range sqlSessions {
		if s.PeerID == "tg-user-42" && s.ChannelID == chatID {
			found = true
			if s.ChannelType != string(types.ChannelTelegram) {
				t.Errorf("expected channelType 'telegram', got %q", s.ChannelType)
			}
			break
		}
	}
	if !found {
		t.Error("SQLite session not found with expected channelID and peerID")
	}

	// Verify the file store session has the correct channel set.
	expectedKey := "agent:default:tg-user-42"
	index, err := fs.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	entry, ok := index[expectedKey]
	if !ok {
		t.Fatalf("file store entry not found for key %q; keys: %v", expectedKey, keys(index))
	}
	if entry.Channel != string(types.ChannelTelegram) {
		t.Errorf("expected file store channel 'telegram', got %q", entry.Channel)
	}
	if entry.ChatType != "direct" {
		t.Errorf("expected file store chatType 'direct', got %q", entry.ChatType)
	}
}

// --- Helpers ---

// keys extracts all keys from a map for diagnostic output.
func keys(m map[string]*filesession.Entry) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

// sessionKeys extracts all keys from a slice of SessionInfo for diagnostic output.
func sessionKeys(sessions []filesession.SessionInfo) []string {
	ks := make([]string, 0, len(sessions))
	for _, s := range sessions {
		ks = append(ks, s.Key)
	}
	return ks
}
