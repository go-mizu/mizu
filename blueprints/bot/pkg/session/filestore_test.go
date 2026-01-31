package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionKey(t *testing.T) {
	tests := []struct {
		name        string
		agentID     string
		channelType string
		peerID      string
		groupID     string
		want        string
	}{
		{
			name:        "DM key",
			agentID:     "bot1",
			channelType: "telegram",
			peerID:      "user123",
			groupID:     "",
			want:        "agent:bot1:user123",
		},
		{
			name:        "Group key",
			agentID:     "bot1",
			channelType: "telegram",
			peerID:      "user123",
			groupID:     "group456",
			want:        "agent:bot1:telegram:group:group456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SessionKey(tt.agentID, tt.channelType, tt.peerID, tt.groupID)
			if got != tt.want {
				t.Errorf("SessionKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileStore_GetOrCreate(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	key := "agent:bot1:user123"

	// Create new session.
	entry, created, err := fs.GetOrCreate(key, "Alice", "direct", "telegram")
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Error("expected session to be newly created")
	}
	if entry.SessionID == "" {
		t.Error("expected non-empty session ID")
	}
	if entry.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want %q", entry.DisplayName, "Alice")
	}
	if entry.ChatType != "direct" {
		t.Errorf("ChatType = %q, want %q", entry.ChatType, "direct")
	}
	if entry.Channel != "telegram" {
		t.Errorf("Channel = %q, want %q", entry.Channel, "telegram")
	}
	if entry.Status != "active" {
		t.Errorf("Status = %q, want %q", entry.Status, "active")
	}
	if entry.UpdatedAt == 0 {
		t.Error("expected non-zero updatedAt")
	}

	origSessionID := entry.SessionID
	origUpdatedAt := entry.UpdatedAt

	// Get existing session.
	entry2, created2, err := fs.GetOrCreate(key, "Alice", "direct", "telegram")
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Error("expected session to already exist")
	}
	if entry2.SessionID != origSessionID {
		t.Errorf("SessionID changed: got %q, want %q", entry2.SessionID, origSessionID)
	}
	if entry2.UpdatedAt < origUpdatedAt {
		t.Error("updatedAt should be >= original")
	}

	// Verify sessions.json was written.
	data, err := os.ReadFile(filepath.Join(dir, "sessions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index map[string]*Entry
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatal(err)
	}
	if _, ok := index[key]; !ok {
		t.Error("expected key in sessions.json")
	}
}

func TestFileStore_AppendAndReadTranscript(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	sessionID := "test-session-001"

	// Append a message.
	msg := &TranscriptEntry{
		Type: "message",
		Message: &TranscriptMessage{
			Role:    "user",
			Content: "Hello, bot!",
		},
	}
	if err := fs.AppendTranscript(sessionID, msg); err != nil {
		t.Fatal(err)
	}

	// Append another message.
	reply := &TranscriptEntry{
		Type: "message",
		Message: &TranscriptMessage{
			Role:    "assistant",
			Content: "Hello! How can I help?",
		},
		Usage: &TokenUsage{Input: 10, Output: 15},
	}
	if err := fs.AppendTranscript(sessionID, reply); err != nil {
		t.Fatal(err)
	}

	// Read transcript.
	entries, err := fs.ReadTranscript(sessionID)
	if err != nil {
		t.Fatal(err)
	}

	// Should have 3 entries: session header + 2 messages.
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// First entry is session header.
	if entries[0].Type != "session" {
		t.Errorf("entry[0].Type = %q, want %q", entries[0].Type, "session")
	}
	if entries[0].Version != 2 {
		t.Errorf("entry[0].Version = %d, want 2", entries[0].Version)
	}
	if entries[0].ID != sessionID {
		t.Errorf("entry[0].ID = %q, want %q", entries[0].ID, sessionID)
	}
	if entries[0].Timestamp == "" {
		t.Error("expected non-empty timestamp on session header")
	}

	// Second entry is user message.
	if entries[1].Type != "message" {
		t.Errorf("entry[1].Type = %q, want %q", entries[1].Type, "message")
	}
	if entries[1].Message == nil {
		t.Fatal("entry[1].Message is nil")
	}
	if entries[1].Message.Role != "user" {
		t.Errorf("entry[1].Message.Role = %q, want %q", entries[1].Message.Role, "user")
	}

	// Third entry is assistant message with usage.
	if entries[2].Message == nil {
		t.Fatal("entry[2].Message is nil")
	}
	if entries[2].Message.Role != "assistant" {
		t.Errorf("entry[2].Message.Role = %q, want %q", entries[2].Message.Role, "assistant")
	}
	if entries[2].Usage == nil {
		t.Fatal("entry[2].Usage is nil")
	}
	if entries[2].Usage.Input != 10 {
		t.Errorf("entry[2].Usage.Input = %d, want 10", entries[2].Usage.Input)
	}
	if entries[2].Usage.Output != 15 {
		t.Errorf("entry[2].Usage.Output = %d, want 15", entries[2].Usage.Output)
	}

	// Verify JSONL file exists.
	jsonlPath := filepath.Join(dir, sessionID+".jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		t.Errorf("expected JSONL file to exist: %v", err)
	}
}

func TestFileStore_ReadTranscript_NonExistent(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := fs.ReadTranscript("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for nonexistent transcript, got %d", len(entries))
	}
}

func TestFileStore_ListSessions(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create sessions with different keys.
	_, _, err = fs.GetOrCreate("agent:bot1:user1", "Alice", "direct", "telegram")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = fs.GetOrCreate("agent:bot1:user2", "Bob", "direct", "telegram")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = fs.GetOrCreate("agent:bot1:telegram:group:g1", "Team", "group", "telegram")
	if err != nil {
		t.Fatal(err)
	}

	sessions, err := fs.ListSessions()
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}

	// Should be sorted by updatedAt descending; last created has highest updatedAt.
	if sessions[0].Entry.DisplayName != "Team" {
		t.Errorf("first session should be Team (most recent), got %q", sessions[0].Entry.DisplayName)
	}

	// Verify all entries have session IDs.
	for i, s := range sessions {
		if s.Entry.SessionID == "" {
			t.Errorf("sessions[%d] has empty SessionID", i)
		}
		if s.Key == "" {
			t.Errorf("sessions[%d] has empty Key", i)
		}
	}
}

func TestFileStore_ResetSession(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	key := "agent:bot1:user1"
	entry, _, err := fs.GetOrCreate(key, "Alice", "direct", "telegram")
	if err != nil {
		t.Fatal(err)
	}
	origID := entry.SessionID

	// Update token usage first.
	if err := fs.UpdateTokenUsage(key, 100, 200); err != nil {
		t.Fatal(err)
	}

	// Reset session.
	reset, err := fs.ResetSession(key)
	if err != nil {
		t.Fatal(err)
	}

	if reset.SessionID == origID {
		t.Error("expected new session ID after reset")
	}
	if reset.SessionID == "" {
		t.Error("expected non-empty session ID after reset")
	}
	if reset.InputTokens != 0 {
		t.Errorf("InputTokens = %d, want 0", reset.InputTokens)
	}
	if reset.OutputTokens != 0 {
		t.Errorf("OutputTokens = %d, want 0", reset.OutputTokens)
	}
	if reset.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0", reset.TotalTokens)
	}
	if reset.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want %q", reset.DisplayName, "Alice")
	}

	// Verify persisted.
	index, err := fs.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if index[key].SessionID != reset.SessionID {
		t.Error("persisted session ID doesn't match reset ID")
	}
}

func TestFileStore_ResetSession_NotFound(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.ResetSession("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestFileStore_UpdateTokenUsage(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	key := "agent:bot1:user1"
	_, _, err = fs.GetOrCreate(key, "Alice", "direct", "telegram")
	if err != nil {
		t.Fatal(err)
	}

	// First update.
	if err := fs.UpdateTokenUsage(key, 50, 100); err != nil {
		t.Fatal(err)
	}

	index, err := fs.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	entry := index[key]
	if entry.InputTokens != 50 {
		t.Errorf("InputTokens = %d, want 50", entry.InputTokens)
	}
	if entry.OutputTokens != 100 {
		t.Errorf("OutputTokens = %d, want 100", entry.OutputTokens)
	}
	if entry.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", entry.TotalTokens)
	}

	// Second update (incremental).
	if err := fs.UpdateTokenUsage(key, 25, 75); err != nil {
		t.Fatal(err)
	}

	index, err = fs.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	entry = index[key]
	if entry.InputTokens != 75 {
		t.Errorf("InputTokens = %d, want 75", entry.InputTokens)
	}
	if entry.OutputTokens != 175 {
		t.Errorf("OutputTokens = %d, want 175", entry.OutputTokens)
	}
	if entry.TotalTokens != 250 {
		t.Errorf("TotalTokens = %d, want 250", entry.TotalTokens)
	}
}

func TestFileStore_UpdateTokenUsage_NotFound(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.UpdateTokenUsage("nonexistent", 10, 20)
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestFileStore_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	key := "agent:bot1:user1"
	_, _, err = fs.GetOrCreate(key, "Alice", "direct", "telegram")
	if err != nil {
		t.Fatal(err)
	}

	// Verify sessions.json exists with correct permissions.
	path := filepath.Join(dir, "sessions.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}

	// Verify no temp file left behind.
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after atomic write")
	}

	// Verify content is valid JSON.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var index map[string]*Entry
	if err := json.Unmarshal(data, &index); err != nil {
		t.Errorf("sessions.json is not valid JSON: %v", err)
	}
	if _, ok := index[key]; !ok {
		t.Error("expected key in sessions.json")
	}

	// Verify it's pretty-printed (indented).
	if !strings.Contains(string(data), "\n") {
		t.Error("expected indented JSON output")
	}
}

func TestFileStore_DeleteEntry(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	key1 := "agent:bot1:user1"
	key2 := "agent:bot1:user2"
	_, _, err = fs.GetOrCreate(key1, "Alice", "direct", "telegram")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = fs.GetOrCreate(key2, "Bob", "direct", "telegram")
	if err != nil {
		t.Fatal(err)
	}

	// Delete one entry.
	if err := fs.DeleteEntry(key1); err != nil {
		t.Fatal(err)
	}

	// Verify only key2 remains.
	index, err := fs.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := index[key1]; ok {
		t.Error("key1 should have been deleted")
	}
	if _, ok := index[key2]; !ok {
		t.Error("key2 should still exist")
	}

	// Delete nonexistent key should not error (idempotent).
	if err := fs.DeleteEntry("nonexistent"); err != nil {
		t.Errorf("deleting nonexistent key should not error: %v", err)
	}
}

func TestFileStore_UpdateEntry(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	key := "agent:bot1:user1"
	entry, _, err := fs.GetOrCreate(key, "Alice", "direct", "telegram")
	if err != nil {
		t.Fatal(err)
	}

	// Update model info.
	entry.Model = "claude-3-opus"
	entry.ModelProvider = "anthropic"
	entry.Label = "test session"
	if err := fs.UpdateEntry(key, entry); err != nil {
		t.Fatal(err)
	}

	// Read back.
	index, err := fs.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	updated := index[key]
	if updated.Model != "claude-3-opus" {
		t.Errorf("Model = %q, want %q", updated.Model, "claude-3-opus")
	}
	if updated.ModelProvider != "anthropic" {
		t.Errorf("ModelProvider = %q, want %q", updated.ModelProvider, "anthropic")
	}
	if updated.Label != "test session" {
		t.Errorf("Label = %q, want %q", updated.Label, "test session")
	}
}

func TestGenerateUUID(t *testing.T) {
	uuid := generateUUID()

	// Check format: 8-4-4-4-12 hex chars.
	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts, got %d: %q", len(parts), uuid)
	}

	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			t.Errorf("part[%d] length = %d, want %d: %q", i, len(part), expectedLengths[i], part)
		}
	}

	// Version 4: third group starts with '4'.
	if parts[2][0] != '4' {
		t.Errorf("version nibble = %c, want '4'", parts[2][0])
	}

	// Variant: fourth group starts with 8, 9, a, or b.
	first := parts[3][0]
	if first != '8' && first != '9' && first != 'a' && first != 'b' {
		t.Errorf("variant nibble = %c, want 8/9/a/b", first)
	}

	// Two UUIDs should differ.
	uuid2 := generateUUID()
	if uuid == uuid2 {
		t.Error("two UUIDs should not be identical")
	}
}

func TestFileStore_TranscriptCustomEntry(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	sessionID := "custom-test"

	// Append custom entry type.
	custom := &TranscriptEntry{
		Type:  "custom",
		Key:   "compaction",
		Value: map[string]any{"before": 100, "after": 20},
	}
	if err := fs.AppendTranscript(sessionID, custom); err != nil {
		t.Fatal(err)
	}

	// Append model_change entry.
	mc := &TranscriptEntry{
		Type:  "model_change",
		Model: "claude-3-sonnet",
	}
	if err := fs.AppendTranscript(sessionID, mc); err != nil {
		t.Fatal(err)
	}

	entries, err := fs.ReadTranscript(sessionID)
	if err != nil {
		t.Fatal(err)
	}

	// Should have 3: header + custom + model_change.
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[1].Type != "custom" {
		t.Errorf("entry[1].Type = %q, want %q", entries[1].Type, "custom")
	}
	if entries[1].Key != "compaction" {
		t.Errorf("entry[1].Key = %q, want %q", entries[1].Key, "compaction")
	}

	if entries[2].Type != "model_change" {
		t.Errorf("entry[2].Type = %q, want %q", entries[2].Type, "model_change")
	}
	if entries[2].Model != "claude-3-sonnet" {
		t.Errorf("entry[2].Model = %q, want %q", entries[2].Model, "claude-3-sonnet")
	}
}
