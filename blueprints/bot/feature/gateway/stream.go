package gateway

import (
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// deltaInterval is the throttle period between delta emissions.
const deltaInterval = 150 * time.Millisecond

// DeltaAccumulator accumulates streaming text and thinking content from the LLM
// and emits throttled delta events every 150ms via a Broadcaster.
// It is thread-safe; all accumulated content is guarded by a mutex.
type DeltaAccumulator struct {
	runID      string
	sessionKey string
	bc         Broadcaster

	mu       sync.Mutex
	text     string
	thinking string
	dirty    bool
	seq      int

	done chan struct{}
}

// NewDeltaAccumulator creates a new accumulator that will broadcast delta events
// for the given run and session via the provided broadcaster.
func NewDeltaAccumulator(runID, sessionKey string, broadcaster Broadcaster) *DeltaAccumulator {
	return &DeltaAccumulator{
		runID:      runID,
		sessionKey: sessionKey,
		bc:         broadcaster,
		done:       make(chan struct{}),
	}
}

// OnTextDelta appends new text content from the LLM stream.
func (d *DeltaAccumulator) OnTextDelta(text string) {
	d.mu.Lock()
	d.text += text
	d.dirty = true
	d.mu.Unlock()
}

// OnThinkingDelta appends new thinking content from the LLM stream.
func (d *DeltaAccumulator) OnThinkingDelta(thinking string) {
	d.mu.Lock()
	d.thinking += thinking
	d.dirty = true
	d.mu.Unlock()
}

// Text returns the accumulated text content so far.
func (d *DeltaAccumulator) Text() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.text
}

// Thinking returns the accumulated thinking content so far.
func (d *DeltaAccumulator) Thinking() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.thinking
}

// Start begins the ticker goroutine that emits delta events every 150ms
// when new content has been accumulated. Must be called before streaming begins.
func (d *DeltaAccumulator) Start() {
	go d.tickerLoop()
}

// tickerLoop runs the 150ms ticker, emitting a delta only when the dirty flag is set.
func (d *DeltaAccumulator) tickerLoop() {
	ticker := time.NewTicker(deltaInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.done:
			return
		case <-ticker.C:
			d.emitIfDirty()
		}
	}
}

// Stop stops the ticker goroutine and emits any remaining accumulated content
// as a final delta. It blocks until the ticker goroutine has exited.
func (d *DeltaAccumulator) Stop() {
	close(d.done)
	// Drain any remaining content as a last delta.
	d.emitIfDirty()
}

// emitIfDirty checks whether new content has arrived since the last emission
// and broadcasts a delta event if so.
func (d *DeltaAccumulator) emitIfDirty() {
	d.mu.Lock()
	if !d.dirty {
		d.mu.Unlock()
		return
	}
	d.seq++
	seq := d.seq
	text := d.text
	thinking := d.thinking
	d.dirty = false
	d.mu.Unlock()

	d.broadcastDelta(seq, text, thinking)
}

// broadcastDelta sends a single delta event with the current accumulated content.
func (d *DeltaAccumulator) broadcastDelta(seq int, text, thinking string) {
	content := buildDeltaContent(text, thinking)

	d.bc.Broadcast("chat", map[string]any{
		"runId":      d.runID,
		"sessionKey": d.sessionKey,
		"seq":        seq,
		"state":      "delta",
		"message": map[string]any{
			"role":      "assistant",
			"content":   content,
			"timestamp": time.Now().UnixMilli(),
		},
	})
}

// buildDeltaContent constructs the content block slice for a delta event.
// If thinking is present it is included as the first block.
func buildDeltaContent(text, thinking string) []map[string]any {
	var content []map[string]any
	if thinking != "" {
		content = append(content, map[string]any{
			"type":     "thinking",
			"thinking": thinking,
		})
	}
	content = append(content, map[string]any{
		"type": "text",
		"text": text,
	})
	return content
}

// EmitFinal broadcasts the final event with the complete content blocks,
// stop reason, and token usage. This is called explicitly after the LLM
// completes, not from the ticker goroutine.
func (d *DeltaAccumulator) EmitFinal(content []types.ContentBlock, stopReason string, usage *TokenUsage) {
	d.mu.Lock()
	d.seq++
	seq := d.seq
	d.mu.Unlock()

	// Convert content blocks to the OpenClaw wire format.
	blocks := make([]map[string]any, 0, len(content))
	for _, b := range content {
		block := map[string]any{"type": b.Type}
		switch b.Type {
		case "text":
			block["text"] = b.Text
		case "thinking":
			block["thinking"] = b.Text
		case "tool_use":
			block["id"] = b.ID
			block["name"] = b.Name
			block["input"] = b.Input
		default:
			block["text"] = b.Text
		}
		blocks = append(blocks, block)
	}

	msg := map[string]any{
		"role":       "assistant",
		"content":    blocks,
		"timestamp":  time.Now().UnixMilli(),
		"stopReason": stopReason,
	}
	if usage != nil {
		msg["usage"] = map[string]any{
			"input":       usage.Input,
			"output":      usage.Output,
			"totalTokens": usage.TotalTokens,
		}
	}

	d.bc.Broadcast("chat", map[string]any{
		"runId":      d.runID,
		"sessionKey": d.sessionKey,
		"seq":        seq,
		"state":      "final",
		"message":    msg,
	})
}

// EmitError broadcasts an error event for the current run.
func (d *DeltaAccumulator) EmitError(err error) {
	d.mu.Lock()
	d.seq++
	seq := d.seq
	d.mu.Unlock()

	d.bc.Broadcast("chat", map[string]any{
		"runId":        d.runID,
		"sessionKey":   d.sessionKey,
		"seq":          seq,
		"state":        "error",
		"errorMessage": err.Error(),
	})
}

// EmitAborted broadcasts an abort event for the current run.
func (d *DeltaAccumulator) EmitAborted(reason string) {
	d.mu.Lock()
	d.seq++
	seq := d.seq
	d.mu.Unlock()

	d.bc.Broadcast("chat", map[string]any{
		"runId":      d.runID,
		"sessionKey": d.sessionKey,
		"seq":        seq,
		"state":      "aborted",
		"stopReason": reason,
	})
}
