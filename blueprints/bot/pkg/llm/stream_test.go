package llm

import (
	"context"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests: SSE line-protocol parser
// ---------------------------------------------------------------------------

func TestParseSSEStream_TextDeltas(t *testing.T) {
	// Simulate an Anthropic streaming response with text deltas.
	ssePayload := strings.Join([]string{
		"event: message_start",
		`data: {"type":"message_start","message":{"id":"msg_01","type":"message","role":"assistant","content":[],"model":"claude-test","usage":{"input_tokens":25,"output_tokens":0}}}`,
		"",
		"event: content_block_start",
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":", world!"}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":0}`,
		"",
		"event: message_delta",
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":10}}`,
		"",
		"event: message_stop",
		`data: {"type":"message_stop"}`,
		"",
	}, "\n")

	var events []struct {
		eventType string
		data      string
	}

	err := ParseSSEStream(context.Background(), strings.NewReader(ssePayload), func(eventType string, data []byte) error {
		events = append(events, struct {
			eventType string
			data      string
		}{eventType, string(data)})
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSEStream: %v", err)
	}

	// We expect 6 events: message_start, content_block_start, 2x content_block_delta,
	// content_block_stop, message_delta, message_stop.
	if len(events) != 7 {
		t.Fatalf("expected 7 events, got %d", len(events))
	}

	// Verify event types in order.
	expectedTypes := []string{
		"message_start",
		"content_block_start",
		"content_block_delta",
		"content_block_delta",
		"content_block_stop",
		"message_delta",
		"message_stop",
	}
	for i, expected := range expectedTypes {
		if events[i].eventType != expected {
			t.Errorf("event[%d]: expected type %q, got %q", i, expected, events[i].eventType)
		}
	}

	// Verify text delta payloads contain the expected text.
	if !strings.Contains(events[2].data, `"text":"Hello"`) {
		t.Errorf("first text delta should contain Hello, got: %s", events[2].data)
	}
	if !strings.Contains(events[3].data, `"text":", world!"`) {
		t.Errorf("second text delta should contain ', world!', got: %s", events[3].data)
	}
}

func TestParseSSEStream_EmptyLines(t *testing.T) {
	// SSE with extra blank lines between events and comments.
	ssePayload := strings.Join([]string{
		": this is a comment",
		"",
		"",
		"event: ping",
		`data: {"type":"ping"}`,
		"",
		"",
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}`,
		"",
		"",
	}, "\n")

	var eventTypes []string

	err := ParseSSEStream(context.Background(), strings.NewReader(ssePayload), func(eventType string, data []byte) error {
		eventTypes = append(eventTypes, eventType)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSEStream: %v", err)
	}

	// Extra blank lines without data should not produce events.
	// Only the two actual frames (ping and content_block_delta) should be dispatched.
	if len(eventTypes) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(eventTypes), eventTypes)
	}
	if eventTypes[0] != "ping" {
		t.Errorf("event[0]: expected ping, got %s", eventTypes[0])
	}
	if eventTypes[1] != "content_block_delta" {
		t.Errorf("event[1]: expected content_block_delta, got %s", eventTypes[1])
	}
}

func TestParseSSEStream_ContextCancel(t *testing.T) {
	// Create a very long stream that would block.
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, "event: content_block_delta")
		lines = append(lines, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"chunk"}}`)
		lines = append(lines, "")
	}
	ssePayload := strings.Join(lines, "\n")

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	err := ParseSSEStream(ctx, strings.NewReader(ssePayload), func(eventType string, data []byte) error {
		callCount++
		if callCount >= 3 {
			// Cancel context after receiving a few events.
			cancel()
		}
		return nil
	})

	// The parser should return the context cancellation error.
	if err == nil {
		t.Fatal("expected error from context cancellation, got nil")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}

	// Should have stopped processing after a few events, not all 1000.
	if callCount > 10 {
		t.Errorf("expected parser to stop early, but processed %d events", callCount)
	}
}

func TestParseSSEStream_MultipleBlocks(t *testing.T) {
	// Stream with thinking block, text block, and tool_use block.
	ssePayload := strings.Join([]string{
		"event: message_start",
		`data: {"type":"message_start","message":{"id":"msg_02","type":"message","role":"assistant","content":[],"model":"claude-test","usage":{"input_tokens":50,"output_tokens":0}}}`,
		"",
		// Thinking block.
		"event: content_block_start",
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Let me think..."}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":" about this."}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":0}`,
		"",
		// Text block.
		"event: content_block_start",
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Here is my answer."}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":1}`,
		"",
		// Tool use block.
		"event: content_block_start",
		`data: {"type":"content_block_start","index":2,"content_block":{"type":"tool_use","id":"toolu_01","name":"read"}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"/tmp/test.txt\"}"}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":2}`,
		"",
		"event: message_delta",
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":30}}`,
		"",
		"event: message_stop",
		`data: {"type":"message_stop"}`,
		"",
	}, "\n")

	var thinkingDeltas []string
	var textDeltas []string
	var jsonDeltas []string
	var blockTypes []string

	err := ParseSSEStream(context.Background(), strings.NewReader(ssePayload), func(eventType string, data []byte) error {
		// Track content_block_start types.
		if eventType == "content_block_start" {
			if strings.Contains(string(data), `"type":"thinking"`) {
				blockTypes = append(blockTypes, "thinking")
			} else if strings.Contains(string(data), `"type":"text"`) {
				blockTypes = append(blockTypes, "text")
			} else if strings.Contains(string(data), `"type":"tool_use"`) {
				blockTypes = append(blockTypes, "tool_use")
			}
		}

		// Track deltas by type.
		if eventType == "content_block_delta" {
			d := string(data)
			if strings.Contains(d, "thinking_delta") {
				thinkingDeltas = append(thinkingDeltas, d)
			} else if strings.Contains(d, "text_delta") {
				textDeltas = append(textDeltas, d)
			} else if strings.Contains(d, "input_json_delta") {
				jsonDeltas = append(jsonDeltas, d)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSEStream: %v", err)
	}

	// Verify we saw all three block types.
	if len(blockTypes) != 3 {
		t.Fatalf("expected 3 block starts, got %d: %v", len(blockTypes), blockTypes)
	}
	expectedBlocks := []string{"thinking", "text", "tool_use"}
	for i, expected := range expectedBlocks {
		if blockTypes[i] != expected {
			t.Errorf("block[%d]: expected %s, got %s", i, expected, blockTypes[i])
		}
	}

	// Verify delta counts.
	if len(thinkingDeltas) != 2 {
		t.Errorf("expected 2 thinking deltas, got %d", len(thinkingDeltas))
	}
	if len(textDeltas) != 1 {
		t.Errorf("expected 1 text delta, got %d", len(textDeltas))
	}
	if len(jsonDeltas) != 1 {
		t.Errorf("expected 1 json delta, got %d", len(jsonDeltas))
	}

	// Verify thinking content.
	if !strings.Contains(thinkingDeltas[0], "Let me think...") {
		t.Errorf("first thinking delta should contain 'Let me think...', got: %s", thinkingDeltas[0])
	}
	if !strings.Contains(thinkingDeltas[1], " about this.") {
		t.Errorf("second thinking delta should contain ' about this.', got: %s", thinkingDeltas[1])
	}

	// Verify text content.
	if !strings.Contains(textDeltas[0], "Here is my answer.") {
		t.Errorf("text delta should contain 'Here is my answer.', got: %s", textDeltas[0])
	}

	// Verify tool use JSON content.
	if !strings.Contains(jsonDeltas[0], "/tmp/test.txt") {
		t.Errorf("json delta should contain tool input, got: %s", jsonDeltas[0])
	}
}

func TestParseSSEStream_DoneSentinel(t *testing.T) {
	// Some SSE streams use [DONE] as end-of-stream marker.
	ssePayload := strings.Join([]string{
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"done"}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")

	callCount := 0
	err := ParseSSEStream(context.Background(), strings.NewReader(ssePayload), func(eventType string, data []byte) error {
		callCount++
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSEStream: %v", err)
	}

	// Only the first event should be dispatched; [DONE] terminates the stream.
	if callCount != 1 {
		t.Errorf("expected 1 event before [DONE], got %d", callCount)
	}
}

func TestParseSSEStream_HandlerError(t *testing.T) {
	ssePayload := strings.Join([]string{
		"event: ping",
		`data: {"type":"ping"}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}`,
		"",
	}, "\n")

	handlerErr := context.DeadlineExceeded // arbitrary sentinel error
	err := ParseSSEStream(context.Background(), strings.NewReader(ssePayload), func(eventType string, data []byte) error {
		return handlerErr
	})

	if err != handlerErr {
		t.Errorf("expected handler error %v, got: %v", handlerErr, err)
	}
}

func TestParseSSEStream_TrailingFrameNoBlankLine(t *testing.T) {
	// A stream that ends without a trailing blank line should still flush.
	ssePayload := "event: ping\ndata: {\"type\":\"ping\"}"

	callCount := 0
	err := ParseSSEStream(context.Background(), strings.NewReader(ssePayload), func(eventType string, data []byte) error {
		callCount++
		if eventType != "ping" {
			t.Errorf("expected event type ping, got %s", eventType)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSEStream: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 trailing event, got %d", callCount)
	}
}
