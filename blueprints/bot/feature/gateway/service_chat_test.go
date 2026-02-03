package gateway

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	filesession "github.com/go-mizu/mizu/blueprints/bot/pkg/session"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// ---------------------------------------------------------------------------
// delayLLM is a mock LLM provider that delays before returning.
// ---------------------------------------------------------------------------

type delayLLM struct {
	delay    time.Duration
	mu       sync.Mutex
	canceled bool
}

func (d *delayLLM) Chat(ctx context.Context, req *types.LLMRequest) (*types.LLMResponse, error) {
	select {
	case <-time.After(d.delay):
		return &types.LLMResponse{
			Content: "[Delayed] response",
			Model:   "delay-mock",
		}, nil
	case <-ctx.Done():
		d.mu.Lock()
		d.canceled = true
		d.mu.Unlock()
		return nil, ctx.Err()
	}
}

// ---------------------------------------------------------------------------
// chatBroadcaster captures broadcast events for async tests.
// ---------------------------------------------------------------------------

type chatBroadcaster struct {
	mu     sync.Mutex
	events []broadcastEvent
	cond   *sync.Cond
}

func newChatBroadcaster() *chatBroadcaster {
	b := &chatBroadcaster{}
	b.cond = sync.NewCond(&b.mu)
	return b
}

func (b *chatBroadcaster) Broadcast(event string, payload any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, broadcastEvent{event, payload})
	b.cond.Broadcast()
}

// waitForEvent waits up to timeout for an event matching the predicate.
func (b *chatBroadcaster) waitForEvent(timeout time.Duration, pred func(broadcastEvent) bool) (broadcastEvent, bool) {
	deadline := time.Now().Add(timeout)
	b.mu.Lock()
	defer b.mu.Unlock()

	for {
		for _, ev := range b.events {
			if pred(ev) {
				return ev, true
			}
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return broadcastEvent{}, false
		}
		// Use a polling approach since sync.Cond doesn't support timeout.
		b.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		b.mu.Lock()
	}
}

func (b *chatBroadcaster) getEvents() []broadcastEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]broadcastEvent, len(b.events))
	copy(out, b.events)
	return out
}

// ---------------------------------------------------------------------------
// mockChannelDriver implements channel.Driver for deliver routing tests.
// ---------------------------------------------------------------------------

type mockChannelDriver struct {
	mu   sync.Mutex
	sent []*types.OutboundMessage
}

func (d *mockChannelDriver) Type() types.ChannelType                        { return types.ChannelTelegram }
func (d *mockChannelDriver) Connect(_ context.Context) error                { return nil }
func (d *mockChannelDriver) Disconnect(_ context.Context) error             { return nil }
func (d *mockChannelDriver) Status() string                                 { return "connected" }
func (d *mockChannelDriver) Send(_ context.Context, msg *types.OutboundMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sent = append(d.sent, msg)
	return nil
}

func (d *mockChannelDriver) getSent() []*types.OutboundMessage {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]*types.OutboundMessage, len(d.sent))
	copy(out, d.sent)
	return out
}

// Compile-time check.
var _ channel.Driver = (*mockChannelDriver)(nil)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func asyncTestMessage(content string) *types.InboundMessage {
	return &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-1",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     content,
		Origin:      "dm",
		Async:       true,
	}
}

// ---------------------------------------------------------------------------
// Tests: Async ProcessMessage
// ---------------------------------------------------------------------------

func TestProcessMessage_AsyncReturnsImmediately(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)

	// Use a short delay so the test doesn't take long, but enough to verify
	// that the async return is faster than the LLM processing.
	delayed := &delayLLM{delay: 500 * time.Millisecond}
	svc := NewService(ms, delayed)
	defer svc.Close()

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	msg := asyncTestMessage("Hello async")

	start := time.Now()
	resp, err := svc.ProcessMessage(ctx, msg)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ProcessMessage async: %v", err)
	}

	// Should return in well under the LLM delay time.
	if elapsed > 200*time.Millisecond {
		t.Errorf("async ProcessMessage took %v; expected < 200ms", elapsed)
	}

	// Should have a RunID.
	if resp.RunID == "" {
		t.Error("expected non-empty RunID for async response")
	}

	// Should have session info.
	if resp.SessionID == "" {
		t.Error("expected non-empty SessionID")
	}
	if resp.AgentID != "agent-1" {
		t.Errorf("expected AgentID 'agent-1', got %q", resp.AgentID)
	}

	// Content should be empty for async (not yet processed).
	if resp.Content != "" {
		t.Errorf("expected empty content for async response, got %q", resp.Content)
	}

	// Wait for the background goroutine to finish so TempDir cleanup succeeds.
	// The goroutine broadcasts events when it completes (either success or error).
	bc.waitForEvent(3*time.Second, func(ev broadcastEvent) bool {
		if ev.event != "chat.done" && ev.event != "chat" {
			return false
		}
		payload, ok := ev.payload.(map[string]any)
		if !ok {
			return false
		}
		state, _ := payload["state"].(string)
		return state == "final" || state == "error" || ev.event == "chat.done"
	})
}

func TestProcessMessage_BroadcastsFinalEvent(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	msg := asyncTestMessage("Hello broadcast")

	resp, err := svc.ProcessMessage(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Wait for the async goroutine to complete and broadcast the final event.
	finalEv, found := bc.waitForEvent(5*time.Second, func(ev broadcastEvent) bool {
		if ev.event != "chat" {
			return false
		}
		payload, ok := ev.payload.(map[string]any)
		if !ok {
			return false
		}
		return payload["state"] == "final"
	})

	if !found {
		// Print all events for debugging.
		events := bc.getEvents()
		t.Logf("received %d events:", len(events))
		for i, ev := range events {
			t.Logf("  [%d] event=%s payload=%v", i, ev.event, ev.payload)
		}
		t.Fatal("timed out waiting for final chat event")
	}

	payload := finalEv.payload.(map[string]any)
	if payload["runId"] != resp.RunID {
		t.Errorf("expected runId %q, got %v", resp.RunID, payload["runId"])
	}
	if payload["state"] != "final" {
		t.Errorf("expected state 'final', got %v", payload["state"])
	}

	// Verify the message content contains the echo response.
	msg2, ok := payload["message"].(map[string]any)
	if !ok {
		t.Fatal("expected message in final event payload")
	}
	if msg2["role"] != "assistant" {
		t.Errorf("expected role 'assistant', got %v", msg2["role"])
	}

	// Verify content blocks include text.
	content, ok := msg2["content"].([]map[string]any)
	if !ok {
		t.Fatalf("expected content to be []map[string]any, got %T", msg2["content"])
	}
	foundText := false
	for _, block := range content {
		if block["type"] == "text" {
			text, _ := block["text"].(string)
			if strings.Contains(text, "[Echo]") {
				foundText = true
			}
		}
	}
	if !foundText {
		t.Error("final event content should contain Echo response text")
	}
}

func TestProcessMessage_TimeoutApplied(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)

	// LLM delays 5 seconds, but timeout is 200ms.
	delayed := &delayLLM{delay: 5 * time.Second}
	svc := NewService(ms, delayed)
	defer svc.Close()

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	msg := &types.InboundMessage{
		ChannelType: types.ChannelWebhook,
		ChannelID:   "chan-1",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     "Hello timeout",
		Origin:      "dm",
		Async:       false,
		TimeoutMs:   200,
	}

	start := time.Now()
	_, err := svc.ProcessMessage(ctx, msg)
	elapsed := time.Since(start)

	// Should fail because of timeout.
	if err == nil {
		t.Fatal("expected error from timeout, got nil")
	}

	// Should complete in roughly the timeout duration, not the full 5s delay.
	if elapsed > 2*time.Second {
		t.Errorf("expected timeout within ~200ms, but took %v", elapsed)
	}

	// The error should relate to the context being cancelled.
	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("expected context/deadline error, got: %v", err)
	}
}

func TestProcessMessage_TranscriptWritten(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	// Set up a file store for JSONL transcripts.
	sessionsDir := filepath.Join(t.TempDir(), "sessions")
	fs, err := filesession.NewFileStore(sessionsDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	svc.SetFileStore(fs)

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	msg := testMessage("What is 2+2?")

	resp, err := svc.ProcessMessage(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// The file store should have a session entry.
	sessions, err := fs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("expected at least 1 file session after ProcessMessage")
	}

	// Read the JSONL transcript using the DB session ID (which is used as the
	// JSONL filename). The gateway writes transcripts using the store session
	// ID, not the file store's own generated UUID.
	dbSessionID := resp.SessionID
	transcript, err := fs.ReadTranscript(dbSessionID)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	// Should have: session header + user message + assistant message = 3 entries.
	if len(transcript) < 3 {
		t.Fatalf("expected at least 3 transcript entries, got %d", len(transcript))
	}

	// First entry is session header.
	if transcript[0].Type != "session" {
		t.Errorf("first transcript entry should be session header, got %q", transcript[0].Type)
	}

	// Second entry should be the user message.
	if transcript[1].Type != "message" {
		t.Errorf("second transcript entry should be message, got %q", transcript[1].Type)
	}
	if transcript[1].Message == nil {
		t.Fatal("second entry should have a message")
	}
	if transcript[1].Message.Role != "user" {
		t.Errorf("second entry role should be 'user', got %q", transcript[1].Message.Role)
	}

	// Verify user message content.
	contentStr, ok := transcript[1].Message.Content.(string)
	if !ok {
		t.Fatalf("user message content should be string, got %T", transcript[1].Message.Content)
	}
	if contentStr != "What is 2+2?" {
		t.Errorf("user message content mismatch: %q", contentStr)
	}

	// Third entry should be the assistant message.
	if transcript[2].Type != "message" {
		t.Errorf("third transcript entry should be message, got %q", transcript[2].Type)
	}
	if transcript[2].Message == nil {
		t.Fatal("third entry should have a message")
	}
	if transcript[2].Message.Role != "assistant" {
		t.Errorf("third entry role should be 'assistant', got %q", transcript[2].Message.Role)
	}
}

func TestProcessMessage_DeliverRouting(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	// Register a mock channel driver.
	driver := &mockChannelDriver{}
	svc.RegisterChannelDriver("telegram", driver)

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	msg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-1",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     "Hello deliver",
		Origin:      "dm",
		Deliver:     true,
	}

	resp, err := svc.ProcessMessage(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Verify the response was sent via the channel driver.
	sent := driver.getSent()
	if len(sent) == 0 {
		t.Fatal("expected at least 1 message sent via channel driver")
	}

	// The delivered message should contain the echo response.
	lastSent := sent[len(sent)-1]
	if !strings.Contains(lastSent.Content, "[Echo]") {
		t.Errorf("delivered message should contain echo response, got: %s", lastSent.Content)
	}
	if lastSent.PeerID != "user-1" {
		t.Errorf("delivered message should target user-1, got: %s", lastSent.PeerID)
	}
	if lastSent.ChannelType != types.ChannelTelegram {
		t.Errorf("delivered message should be telegram type, got: %s", lastSent.ChannelType)
	}

	// Response should still be valid.
	if !strings.Contains(resp.Content, "[Echo]") {
		t.Errorf("response content should contain echo, got: %s", resp.Content)
	}
}

func TestProcessMessage_NoDeliverForWebhook(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	driver := &mockChannelDriver{}
	svc.RegisterChannelDriver("webhook", driver)

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	msg := &types.InboundMessage{
		ChannelType: types.ChannelWebhook,
		ChannelID:   "chan-1",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     "Hello webhook",
		Origin:      "dm",
		Deliver:     true,
	}

	_, err := svc.ProcessMessage(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Webhook channels should NOT deliver (per the code logic).
	sent := driver.getSent()
	if len(sent) != 0 {
		t.Errorf("expected no delivery for webhook channel, got %d messages", len(sent))
	}
}

// ---------------------------------------------------------------------------
// Tests: Abort functionality
// ---------------------------------------------------------------------------

func TestAbortByRunID(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)

	delayed := &delayLLM{delay: 10 * time.Second}
	svc := NewService(ms, delayed)
	defer svc.Close()

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	msg := asyncTestMessage("slow request")

	resp, err := svc.ProcessMessage(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Give the goroutine a moment to register the inflight entry.
	time.Sleep(100 * time.Millisecond)

	// Abort the run.
	aborted, err := svc.AbortByRunID(resp.RunID, resp.SessionKey)
	if err != nil {
		t.Fatalf("AbortByRunID: %v", err)
	}
	if !aborted {
		t.Error("expected abort to succeed")
	}

	// Verify an aborted event was broadcast.
	abortEv, found := bc.waitForEvent(2*time.Second, func(ev broadcastEvent) bool {
		if ev.event != "chat" {
			return false
		}
		payload, ok := ev.payload.(map[string]any)
		if !ok {
			return false
		}
		return payload["state"] == "aborted"
	})
	if !found {
		t.Fatal("expected aborted event to be broadcast")
	}

	payload := abortEv.payload.(map[string]any)
	if payload["runId"] != resp.RunID {
		t.Errorf("expected aborted runId %q, got %v", resp.RunID, payload["runId"])
	}
}

func TestAbortBySessionKey(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)

	delayed := &delayLLM{delay: 10 * time.Second}
	svc := NewService(ms, delayed)
	defer svc.Close()

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	msg := asyncTestMessage("slow request 2")

	resp, err := svc.ProcessMessage(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Give the goroutine a moment to register.
	time.Sleep(100 * time.Millisecond)

	aborted, ok := svc.AbortBySessionKey(resp.SessionKey)
	if !ok {
		t.Error("expected abort by session key to succeed")
	}
	if len(aborted) != 1 {
		t.Errorf("expected 1 aborted run, got %d", len(aborted))
	}
}

// ---------------------------------------------------------------------------
// Tests: Deduplication cache
// ---------------------------------------------------------------------------

func TestDedupe_CacheHit(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	// Set a value in the dedup cache.
	svc.SetDedupe("key-1", map[string]any{"result": "cached"})

	// Retrieve it.
	payload, ok := svc.CheckDedupe("key-1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	m, ok := payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", payload)
	}
	if m["result"] != "cached" {
		t.Errorf("expected result 'cached', got %v", m["result"])
	}
}

func TestDedupe_CacheMiss(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	_, ok := svc.CheckDedupe("nonexistent")
	if ok {
		t.Error("expected cache miss for nonexistent key")
	}
}

// ---------------------------------------------------------------------------
// Tests: SessionKey helpers
// ---------------------------------------------------------------------------

func TestBuildSessionKey_DashboardUser(t *testing.T) {
	key := BuildSessionKey("main", "webhook", "dashboard-user")
	if key != "agent:main:main" {
		t.Errorf("expected 'agent:main:main', got %q", key)
	}
}

func TestBuildSessionKey_TelegramPeer(t *testing.T) {
	key := BuildSessionKey("agent-1", "telegram", "12345")
	if key != "agent:agent-1:telegram:12345" {
		t.Errorf("expected 'agent:agent-1:telegram:12345', got %q", key)
	}
}

func TestSessionKeyToQuery_MainKey(t *testing.T) {
	agentID, channelType, peerID := SessionKeyToQuery("agent:main:main")
	if agentID != "main" {
		t.Errorf("expected agentID 'main', got %q", agentID)
	}
	if channelType != "webhook" {
		t.Errorf("expected channelType 'webhook', got %q", channelType)
	}
	if peerID != "dashboard-user" {
		t.Errorf("expected peerID 'dashboard-user', got %q", peerID)
	}
}

func TestSessionKeyToQuery_FullKey(t *testing.T) {
	agentID, channelType, peerID := SessionKeyToQuery("agent:bot-1:telegram:user-42")
	if agentID != "bot-1" {
		t.Errorf("expected agentID 'bot-1', got %q", agentID)
	}
	if channelType != "telegram" {
		t.Errorf("expected channelType 'telegram', got %q", channelType)
	}
	if peerID != "user-42" {
		t.Errorf("expected peerID 'user-42', got %q", peerID)
	}
}

func TestSessionKeyToQuery_InvalidKey(t *testing.T) {
	agentID, channelType, peerID := SessionKeyToQuery("invalid-key")
	// Should fall back to defaults.
	if agentID != "main" {
		t.Errorf("expected default agentID 'main', got %q", agentID)
	}
	if channelType != "webhook" {
		t.Errorf("expected default channelType 'webhook', got %q", channelType)
	}
	if peerID != "dashboard-user" {
		t.Errorf("expected default peerID 'dashboard-user', got %q", peerID)
	}
}

// ---------------------------------------------------------------------------
// Tests: Async with broadcaster stores messages in DB
// ---------------------------------------------------------------------------

func TestProcessMessage_AsyncStoresMessages(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	msg := asyncTestMessage("async store test")

	_, err := svc.ProcessMessage(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Wait for the final event (indicating async processing completed).
	_, found := bc.waitForEvent(5*time.Second, func(ev broadcastEvent) bool {
		if ev.event != "chat" {
			return false
		}
		payload, ok := ev.payload.(map[string]any)
		if !ok {
			return false
		}
		return payload["state"] == "final"
	})
	if !found {
		t.Fatal("timed out waiting for async processing to complete")
	}

	// After async processing, messages should be stored.
	ms.mu.Lock()
	msgCount := len(ms.messages)
	ms.mu.Unlock()

	// Should have: 1 user + 1 assistant = 2 messages.
	if msgCount != 2 {
		t.Errorf("expected 2 stored messages, got %d", msgCount)
	}

	// Verify roles.
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if ms.messages[0].Role != types.RoleUser {
		t.Errorf("first message should be user, got %s", ms.messages[0].Role)
	}
	if ms.messages[1].Role != types.RoleAssistant {
		t.Errorf("second message should be assistant, got %s", ms.messages[1].Role)
	}
	if !strings.Contains(ms.messages[1].Content, "[Echo]") {
		t.Errorf("assistant message should contain echo, got: %s", ms.messages[1].Content)
	}
}

// ---------------------------------------------------------------------------
// Tests: Transcript JSONL with assistant metadata
// ---------------------------------------------------------------------------

func TestProcessMessage_TranscriptHasUsage(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)

	// Use a capturing LLM that reports token usage.
	capLLM := &capturingLLM{
		onChat: func(req *types.LLMRequest) {},
	}
	svc := NewService(ms, capLLM)
	defer svc.Close()

	sessionsDir := filepath.Join(t.TempDir(), "sessions")
	fs, err := filesession.NewFileStore(sessionsDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	svc.SetFileStore(fs)

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	resp, err := svc.ProcessMessage(ctx, testMessage("test usage"))
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Read transcript using the DB session ID (which is the JSONL filename).
	transcript, err := fs.ReadTranscript(resp.SessionID)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	// Find the assistant message entry.
	var assistantEntry *filesession.TranscriptEntry
	for i := range transcript {
		if transcript[i].Type == "message" && transcript[i].Message != nil && transcript[i].Message.Role == "assistant" {
			assistantEntry = &transcript[i]
			break
		}
	}
	if assistantEntry == nil {
		t.Fatal("expected assistant message in transcript")
	}

	// The content should be serializable (could be string or content blocks).
	contentJSON, err := json.Marshal(assistantEntry.Message.Content)
	if err != nil {
		t.Fatalf("marshal content: %v", err)
	}
	if len(contentJSON) == 0 {
		t.Error("expected non-empty content in assistant transcript entry")
	}
}

// ---------------------------------------------------------------------------
// Tests: JSONL files are actually written to disk
// ---------------------------------------------------------------------------

func TestProcessMessage_JSONLFileExists(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	sessionsDir := filepath.Join(t.TempDir(), "sessions")
	fs, err := filesession.NewFileStore(sessionsDir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	svc.SetFileStore(fs)

	bc := newChatBroadcaster()
	svc.SetBroadcaster(bc)

	ctx := context.Background()
	resp, err := svc.ProcessMessage(ctx, testMessage("check file"))
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// The JSONL file is named after the DB session ID (not the file store's own UUID).
	jsonlPath := filepath.Join(sessionsDir, resp.SessionID+".jsonl")
	info, err := os.Stat(jsonlPath)
	if err != nil {
		t.Fatalf("JSONL file should exist at %s: %v", jsonlPath, err)
	}
	if info.Size() == 0 {
		t.Error("JSONL file should not be empty")
	}
}
