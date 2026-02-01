package compat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/session"
)

// TestSessionEntryStructure creates a session via the session package
// and verifies it has all fields that OpenClaw's sessions.json has.
func TestSessionEntryStructure(t *testing.T) {
	dir := t.TempDir()
	store, err := session.NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	key := "agent:main:test-peer"
	entry, isNew, err := store.GetOrCreate(key, "Test User", "direct", "telegram")
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if !isNew {
		t.Fatal("expected new session")
	}

	// Populate all OpenClaw-compatible fields.
	entry.SystemSent = true
	entry.Model = "claude-opus-4-5"
	entry.ModelProvider = "anthropic"
	entry.ContextTokens = 200000
	entry.DeliveryContext = &session.DeliveryCtx{
		Channel:   "telegram",
		To:        "telegram:12345",
		AccountId: "default",
	}
	entry.LastChannel = "telegram"
	entry.LastTo = "telegram:12345"
	entry.LastAccountId = "default"
	entry.AuthProfileOverride = "anthropic:default"
	entry.AuthProfileOverrideSource = "auto"
	entry.Origin = &session.SessionOrigin{
		Label:     "Test User id:12345",
		Provider:  "telegram",
		Surface:   "telegram",
		ChatType:  "direct",
		From:      "telegram:12345",
		To:        "telegram:12345",
		AccountId: "default",
	}
	entry.SessionFile = filepath.Join(dir, entry.SessionID+".jsonl")
	entry.InputTokens = 100
	entry.OutputTokens = 200
	entry.TotalTokens = 300
	entry.SkillsSnapshot = &session.SkillsSnap{
		Version: 1,
		Skills: []session.SkillRef{
			{Name: "weather"},
			{Name: "github"},
		},
	}
	entry.SystemPromptReport = &session.SystemPromptReport{
		Source:       "run",
		GeneratedAt: entry.UpdatedAt,
		SessionID:   entry.SessionID,
		SessionKey:  key,
		Provider:    "anthropic",
		Model:       "claude-opus-4-5",
	}

	if err := store.UpdateEntry(key, entry); err != nil {
		t.Fatalf("UpdateEntry: %v", err)
	}

	// Read back and verify JSON has all expected fields.
	data, err := os.ReadFile(filepath.Join(dir, "sessions.json"))
	if err != nil {
		t.Fatalf("read sessions.json: %v", err)
	}

	var index map[string]map[string]any
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatalf("parse sessions.json: %v", err)
	}

	entryMap, ok := index[key]
	if !ok {
		t.Fatalf("key %q not found in sessions.json", key)
	}

	// All OpenClaw session entry fields that must be present.
	requiredFields := []string{
		"sessionId",
		"updatedAt",
		"systemSent",
		"chatType",
		"deliveryContext",
		"lastChannel",
		"origin",
		"sessionFile",
		"skillsSnapshot",
		"authProfileOverride",
		"authProfileOverrideSource",
		"lastTo",
		"lastAccountId",
		"inputTokens",
		"outputTokens",
		"totalTokens",
		"modelProvider",
		"model",
		"contextTokens",
		"systemPromptReport",
	}

	for _, field := range requiredFields {
		t.Run("field/"+field, func(t *testing.T) {
			if _, ok := entryMap[field]; !ok {
				t.Errorf("session entry missing field %q", field)
			}
		})
	}

	// Verify deliveryContext structure.
	t.Run("deliveryContext/structure", func(t *testing.T) {
		dc, ok := entryMap["deliveryContext"].(map[string]any)
		if !ok {
			t.Fatal("deliveryContext is not an object")
		}
		for _, f := range []string{"channel", "to", "accountId"} {
			if _, ok := dc[f]; !ok {
				t.Errorf("deliveryContext missing field %q", f)
			}
		}
	})

	// Verify origin structure.
	t.Run("origin/structure", func(t *testing.T) {
		orig, ok := entryMap["origin"].(map[string]any)
		if !ok {
			t.Fatal("origin is not an object")
		}
		for _, f := range []string{"label", "provider", "surface", "chatType", "from", "to", "accountId"} {
			if _, ok := orig[f]; !ok {
				t.Errorf("origin missing field %q", f)
			}
		}
	})
}

// TestSessionTranscriptFormat creates a transcript and verifies JSONL format.
func TestSessionTranscriptFormat(t *testing.T) {
	dir := t.TempDir()
	store, err := session.NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	key := "agent:main:test-peer"
	entry, _, err := store.GetOrCreate(key, "Test", "direct", "telegram")
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}

	// Append messages.
	store.AppendTranscript(entry.SessionID, &session.TranscriptEntry{
		Type: "message",
		Message: &session.TranscriptMessage{
			Role:    "user",
			Content: "Hello",
		},
	})
	store.AppendTranscript(entry.SessionID, &session.TranscriptEntry{
		Type: "message",
		Message: &session.TranscriptMessage{
			Role:    "assistant",
			Content: "Hi there!",
		},
		Usage: &session.TokenUsage{Input: 5, Output: 10},
	})

	// Read back.
	entries, err := store.ReadTranscript(entry.SessionID)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	// Should have 3 entries: session header + 2 messages.
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// First should be session header.
	if entries[0].Type != "session" {
		t.Errorf("first entry type = %q; want %q", entries[0].Type, "session")
	}
	if entries[0].Version != 2 {
		t.Errorf("session version = %d; want 2", entries[0].Version)
	}

	// Second should be user message.
	if entries[1].Type != "message" {
		t.Errorf("second entry type = %q; want %q", entries[1].Type, "message")
	}
	if entries[1].Message.Role != "user" {
		t.Errorf("second entry role = %q; want %q", entries[1].Message.Role, "user")
	}

	// Third should be assistant message with usage.
	if entries[2].Type != "message" {
		t.Errorf("third entry type = %q; want %q", entries[2].Type, "message")
	}
	if entries[2].Usage == nil {
		t.Error("third entry missing usage")
	} else if entries[2].Usage.Input != 5 {
		t.Errorf("usage input = %d; want 5", entries[2].Usage.Input)
	}
}

// TestSessionKeyFormat verifies session key format matches OpenClaw convention.
func TestSessionKeyFormat(t *testing.T) {
	tests := []struct {
		name     string
		agentID  string
		channel  string
		peerID   string
		groupID  string
		expected string
	}{
		{"DM", "main", "telegram", "12345", "", "agent:main:12345"},
		{"Group", "main", "telegram", "12345", "group99", "agent:main:telegram:group:group99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := session.SessionKey(tt.agentID, tt.channel, tt.peerID, tt.groupID)
			if got != tt.expected {
				t.Errorf("SessionKey() = %q; want %q", got, tt.expected)
			}
		})
	}
}

// TestLiveSessionComparison compares actual sessions.json between openclaw and openbot.
func TestLiveSessionComparison(t *testing.T) {
	openclawSess := filepath.Join(os.Getenv("HOME"), ".openclaw", "agents", "main", "sessions", "sessions.json")
	openbotSess := filepath.Join(os.Getenv("HOME"), ".openbot", "agents", "main", "sessions", "sessions.json")

	if _, err := os.Stat(openclawSess); os.IsNotExist(err) {
		t.Skip("~/.openclaw sessions.json does not exist")
	}
	if _, err := os.Stat(openbotSess); os.IsNotExist(err) {
		t.Skip("~/.openbot sessions.json does not exist")
	}

	ocData, err := os.ReadFile(openclawSess)
	if err != nil {
		t.Fatalf("read openclaw sessions: %v", err)
	}
	obData, err := os.ReadFile(openbotSess)
	if err != nil {
		t.Fatalf("read openbot sessions: %v", err)
	}

	var ocIndex, obIndex map[string]map[string]any
	if err := json.Unmarshal(ocData, &ocIndex); err != nil {
		t.Fatalf("parse openclaw sessions: %v", err)
	}
	if err := json.Unmarshal(obData, &obIndex); err != nil {
		t.Fatalf("parse openbot sessions: %v", err)
	}

	// Get first entry from each and compare fields.
	var ocEntry map[string]any
	for _, v := range ocIndex {
		ocEntry = v
		break
	}
	var obEntry map[string]any
	for _, v := range obIndex {
		obEntry = v
		break
	}

	if ocEntry == nil {
		t.Skip("no openclaw sessions")
	}
	if obEntry == nil {
		t.Skip("no openbot sessions")
	}

	// Check all OpenClaw fields exist in OpenBot entry.
	for key := range ocEntry {
		t.Run("field/"+key, func(t *testing.T) {
			if _, ok := obEntry[key]; !ok {
				t.Errorf("openbot session missing field %q", key)
			}
		})
	}

	// Check type consistency for shared fields.
	for key, ocVal := range ocEntry {
		obVal, ok := obEntry[key]
		if !ok {
			continue
		}
		t.Run("type/"+key, func(t *testing.T) {
			ocType := typeOf(ocVal)
			obType := typeOf(obVal)
			if ocType != obType {
				t.Errorf("type mismatch for %q: openclaw=%s openbot=%s", key, ocType, obType)
			}
		})
	}
}

func typeOf(v any) string {
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}
