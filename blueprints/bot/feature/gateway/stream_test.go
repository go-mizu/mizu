package gateway

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// ---------------------------------------------------------------------------
// Mock broadcaster for stream tests
// ---------------------------------------------------------------------------

type broadcastEvent struct {
	event   string
	payload any
}

type mockBroadcaster struct {
	mu     sync.Mutex
	events []broadcastEvent
}

func (m *mockBroadcaster) Broadcast(event string, payload any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, broadcastEvent{event, payload})
}

func (m *mockBroadcaster) getEvents() []broadcastEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]broadcastEvent, len(m.events))
	copy(out, m.events)
	return out
}

// ---------------------------------------------------------------------------
// Tests: DeltaAccumulator text and thinking accumulation
// ---------------------------------------------------------------------------

func TestDeltaAccumulator_TextAccumulation(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-1", "agent:main:webhook:user-1", bc)

	// Add text deltas.
	acc.OnTextDelta("Hello")
	acc.OnTextDelta(", ")
	acc.OnTextDelta("world!")

	got := acc.Text()
	if got != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %q", got)
	}
}

func TestDeltaAccumulator_ThinkingAccumulation(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-1", "agent:main:webhook:user-1", bc)

	// Add thinking deltas.
	acc.OnThinkingDelta("Let me ")
	acc.OnThinkingDelta("think about this.")

	got := acc.Thinking()
	if got != "Let me think about this." {
		t.Errorf("expected 'Let me think about this.', got %q", got)
	}

	// Text should still be empty.
	if acc.Text() != "" {
		t.Errorf("expected empty text, got %q", acc.Text())
	}
}

func TestDeltaAccumulator_MixedAccumulation(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-1", "agent:main:webhook:user-1", bc)

	// Mix text and thinking.
	acc.OnThinkingDelta("Thinking...")
	acc.OnTextDelta("Answer: 42")

	if acc.Thinking() != "Thinking..." {
		t.Errorf("thinking mismatch: %q", acc.Thinking())
	}
	if acc.Text() != "Answer: 42" {
		t.Errorf("text mismatch: %q", acc.Text())
	}
}

// ---------------------------------------------------------------------------
// Tests: DeltaAccumulator emit methods
// ---------------------------------------------------------------------------

func TestDeltaAccumulator_EmitFinal(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-1", "agent:main:main", bc)

	contentBlocks := []types.ContentBlock{
		{Type: "thinking", Text: "deep thought"},
		{Type: "text", Text: "The answer is 42."},
	}
	usage := &TokenUsage{Input: 100, Output: 50, TotalTokens: 150}

	acc.EmitFinal(contentBlocks, "end_turn", usage)

	events := bc.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 broadcast event, got %d", len(events))
	}

	ev := events[0]
	if ev.event != "chat" {
		t.Errorf("expected event name 'chat', got %q", ev.event)
	}

	payload, ok := ev.payload.(map[string]any)
	if !ok {
		t.Fatalf("payload is not map[string]any: %T", ev.payload)
	}

	if payload["runId"] != "run-1" {
		t.Errorf("expected runId 'run-1', got %v", payload["runId"])
	}
	if payload["sessionKey"] != "agent:main:main" {
		t.Errorf("expected sessionKey 'agent:main:main', got %v", payload["sessionKey"])
	}
	if payload["state"] != "final" {
		t.Errorf("expected state 'final', got %v", payload["state"])
	}

	msg, ok := payload["message"].(map[string]any)
	if !ok {
		t.Fatalf("message is not map[string]any: %T", payload["message"])
	}
	if msg["role"] != "assistant" {
		t.Errorf("expected role 'assistant', got %v", msg["role"])
	}
	if msg["stopReason"] != "end_turn" {
		t.Errorf("expected stopReason 'end_turn', got %v", msg["stopReason"])
	}

	// Verify usage is included.
	usageMap, ok := msg["usage"].(map[string]any)
	if !ok {
		t.Fatalf("usage is not map[string]any: %T", msg["usage"])
	}
	if usageMap["input"] != 100 {
		t.Errorf("expected input 100, got %v", usageMap["input"])
	}
	if usageMap["output"] != 50 {
		t.Errorf("expected output 50, got %v", usageMap["output"])
	}
	if usageMap["totalTokens"] != 150 {
		t.Errorf("expected totalTokens 150, got %v", usageMap["totalTokens"])
	}

	// Verify content blocks.
	content, ok := msg["content"].([]map[string]any)
	if !ok {
		t.Fatalf("content is not []map[string]any: %T", msg["content"])
	}
	if len(content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(content))
	}
	if content[0]["type"] != "thinking" {
		t.Errorf("first block type should be 'thinking', got %v", content[0]["type"])
	}
	if content[0]["thinking"] != "deep thought" {
		t.Errorf("first block thinking should be 'deep thought', got %v", content[0]["thinking"])
	}
	if content[1]["type"] != "text" {
		t.Errorf("second block type should be 'text', got %v", content[1]["type"])
	}
	if content[1]["text"] != "The answer is 42." {
		t.Errorf("second block text should be 'The answer is 42.', got %v", content[1]["text"])
	}
}

func TestDeltaAccumulator_EmitFinal_NilUsage(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-2", "agent:main:main", bc)

	contentBlocks := []types.ContentBlock{
		{Type: "text", Text: "response"},
	}

	acc.EmitFinal(contentBlocks, "end_turn", nil)

	events := bc.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	payload := events[0].payload.(map[string]any)
	msg := payload["message"].(map[string]any)

	// usage should NOT be present when nil.
	if _, exists := msg["usage"]; exists {
		t.Error("expected no usage field when nil")
	}
}

func TestDeltaAccumulator_EmitError(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-err", "agent:main:main", bc)

	testErr := errors.New("LLM overloaded")
	acc.EmitError(testErr)

	events := bc.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 broadcast event, got %d", len(events))
	}

	ev := events[0]
	if ev.event != "chat" {
		t.Errorf("expected event name 'chat', got %q", ev.event)
	}

	payload, ok := ev.payload.(map[string]any)
	if !ok {
		t.Fatalf("payload is not map[string]any: %T", ev.payload)
	}

	if payload["runId"] != "run-err" {
		t.Errorf("expected runId 'run-err', got %v", payload["runId"])
	}
	if payload["state"] != "error" {
		t.Errorf("expected state 'error', got %v", payload["state"])
	}
	if payload["errorMessage"] != "LLM overloaded" {
		t.Errorf("expected errorMessage 'LLM overloaded', got %v", payload["errorMessage"])
	}
}

func TestDeltaAccumulator_EmitAborted(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-abort", "agent:main:main", bc)

	acc.EmitAborted("user_cancelled")

	events := bc.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 broadcast event, got %d", len(events))
	}

	ev := events[0]
	if ev.event != "chat" {
		t.Errorf("expected event name 'chat', got %q", ev.event)
	}

	payload, ok := ev.payload.(map[string]any)
	if !ok {
		t.Fatalf("payload is not map[string]any: %T", ev.payload)
	}

	if payload["runId"] != "run-abort" {
		t.Errorf("expected runId 'run-abort', got %v", payload["runId"])
	}
	if payload["state"] != "aborted" {
		t.Errorf("expected state 'aborted', got %v", payload["state"])
	}
	if payload["stopReason"] != "user_cancelled" {
		t.Errorf("expected stopReason 'user_cancelled', got %v", payload["stopReason"])
	}
}

// ---------------------------------------------------------------------------
// Tests: DeltaAccumulator sequence numbering
// ---------------------------------------------------------------------------

func TestDeltaAccumulator_SequenceIncrementing(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-seq", "agent:main:main", bc)

	// Emit multiple events and verify sequence increments.
	acc.EmitError(errors.New("e1"))
	acc.EmitError(errors.New("e2"))
	acc.EmitAborted("done")

	events := bc.getEvents()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	for i, ev := range events {
		payload := ev.payload.(map[string]any)
		seq := payload["seq"].(int)
		expectedSeq := i + 1
		if seq != expectedSeq {
			t.Errorf("event[%d]: expected seq %d, got %d", i, expectedSeq, seq)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: DeltaAccumulator throttled delta emission
// ---------------------------------------------------------------------------

func TestDeltaAccumulator_ThrottledDelta(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-throttle", "agent:main:main", bc)

	// Start the ticker goroutine.
	acc.Start()

	// Feed text deltas rapidly.
	acc.OnTextDelta("chunk1 ")
	acc.OnTextDelta("chunk2 ")
	acc.OnTextDelta("chunk3 ")

	// Wait for at least one tick (~150ms) plus some buffer.
	time.Sleep(250 * time.Millisecond)

	// Add more text after first tick.
	acc.OnTextDelta("chunk4 ")

	// Wait for another tick.
	time.Sleep(250 * time.Millisecond)

	// Stop the accumulator (this also drains any remaining dirty content).
	acc.Stop()

	events := bc.getEvents()

	// We should have received at least 1 delta event (from the ticker),
	// but not one event per chunk (they should be batched).
	if len(events) < 1 {
		t.Fatalf("expected at least 1 throttled delta event, got %d", len(events))
	}

	// Verify all events are "chat" events with state="delta".
	for i, ev := range events {
		if ev.event != "chat" {
			t.Errorf("event[%d]: expected event name 'chat', got %q", i, ev.event)
		}
		payload, ok := ev.payload.(map[string]any)
		if !ok {
			t.Fatalf("event[%d]: payload is not map[string]any", i)
		}
		if payload["state"] != "delta" {
			t.Errorf("event[%d]: expected state 'delta', got %v", i, payload["state"])
		}
		if payload["runId"] != "run-throttle" {
			t.Errorf("event[%d]: expected runId 'run-throttle', got %v", i, payload["runId"])
		}
	}

	// The last delta event should contain all accumulated text.
	lastPayload := events[len(events)-1].payload.(map[string]any)
	lastMsg := lastPayload["message"].(map[string]any)
	lastContent := lastMsg["content"].([]map[string]any)

	// Find the text block in the content.
	var foundText string
	for _, block := range lastContent {
		if block["type"] == "text" {
			foundText, _ = block["text"].(string)
		}
	}

	if foundText != "chunk1 chunk2 chunk3 chunk4 " {
		t.Errorf("expected all accumulated text, got %q", foundText)
	}
}

func TestDeltaAccumulator_NoDeltaWhenNotDirty(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-clean", "agent:main:main", bc)

	// Start the ticker but don't add any content.
	acc.Start()

	// Wait for a couple of ticks.
	time.Sleep(400 * time.Millisecond)

	acc.Stop()

	events := bc.getEvents()

	// No content was added, so no delta events should have been emitted.
	if len(events) != 0 {
		t.Errorf("expected 0 events when not dirty, got %d", len(events))
	}
}

func TestDeltaAccumulator_ThinkingInDelta(t *testing.T) {
	bc := &mockBroadcaster{}
	acc := NewDeltaAccumulator("run-think", "agent:main:main", bc)

	acc.Start()

	// Add thinking content.
	acc.OnThinkingDelta("I need to consider this...")
	acc.OnTextDelta("The answer is 42.")

	// Wait for a tick.
	time.Sleep(250 * time.Millisecond)

	acc.Stop()

	events := bc.getEvents()
	if len(events) < 1 {
		t.Fatalf("expected at least 1 delta event, got %d", len(events))
	}

	// Verify thinking is included in the delta content.
	lastPayload := events[len(events)-1].payload.(map[string]any)
	lastMsg := lastPayload["message"].(map[string]any)
	lastContent := lastMsg["content"].([]map[string]any)

	// When thinking is present, it should be the first block.
	if len(lastContent) < 2 {
		t.Fatalf("expected at least 2 content blocks (thinking + text), got %d", len(lastContent))
	}
	if lastContent[0]["type"] != "thinking" {
		t.Errorf("first block should be thinking, got %v", lastContent[0]["type"])
	}
	if lastContent[0]["thinking"] != "I need to consider this..." {
		t.Errorf("thinking content mismatch: %v", lastContent[0]["thinking"])
	}
	if lastContent[1]["type"] != "text" {
		t.Errorf("second block should be text, got %v", lastContent[1]["type"])
	}
}

// ---------------------------------------------------------------------------
// Tests: buildDeltaContent helper
// ---------------------------------------------------------------------------

func TestBuildDeltaContent_TextOnly(t *testing.T) {
	content := buildDeltaContent("hello", "")
	if len(content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(content))
	}
	if content[0]["type"] != "text" {
		t.Errorf("expected type 'text', got %v", content[0]["type"])
	}
	if content[0]["text"] != "hello" {
		t.Errorf("expected text 'hello', got %v", content[0]["text"])
	}
}

func TestBuildDeltaContent_WithThinking(t *testing.T) {
	content := buildDeltaContent("answer", "reasoning")
	if len(content) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(content))
	}
	if content[0]["type"] != "thinking" {
		t.Errorf("first block should be thinking, got %v", content[0]["type"])
	}
	if content[0]["thinking"] != "reasoning" {
		t.Errorf("thinking content mismatch: %v", content[0]["thinking"])
	}
	if content[1]["type"] != "text" {
		t.Errorf("second block should be text, got %v", content[1]["type"])
	}
	if content[1]["text"] != "answer" {
		t.Errorf("text content mismatch: %v", content[1]["text"])
	}
}
