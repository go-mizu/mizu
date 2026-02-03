package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// ---------------------------------------------------------------------------
// Stream event types (Anthropic SSE streaming API)
// ---------------------------------------------------------------------------

// StreamEventType enumerates the event types emitted by Anthropic's streaming
// Messages API.
type StreamEventType string

const (
	EventMessageStart     StreamEventType = "message_start"
	EventContentBlockStart StreamEventType = "content_block_start"
	EventContentBlockDelta StreamEventType = "content_block_delta"
	EventContentBlockStop  StreamEventType = "content_block_stop"
	EventMessageDelta     StreamEventType = "message_delta"
	EventMessageStop      StreamEventType = "message_stop"
	EventPing             StreamEventType = "ping"
	EventError            StreamEventType = "error"
)

// StreamEvent is one parsed SSE event from the Anthropic streaming API.
type StreamEvent struct {
	Type StreamEventType `json:"type"`

	// message_start: the initial message envelope.
	Message *StreamMessage `json:"message,omitempty"`

	// content_block_start: the opening of a content block.
	Index        int               `json:"index,omitempty"`
	ContentBlock *ContentBlockStart `json:"content_block,omitempty"`

	// content_block_delta: an incremental update to the current block.
	Delta *DeltaContent `json:"delta,omitempty"`

	// message_delta: final metadata (stop_reason, usage).
	Usage *StreamUsage `json:"usage,omitempty"`

	// error: an API-level error inside the stream.
	Error *StreamError `json:"error,omitempty"`
}

// StreamMessage is the top-level message object received in message_start.
type StreamMessage struct {
	ID           string              `json:"id"`
	Type         string              `json:"type"`
	Role         string              `json:"role"`
	Content      []json.RawMessage   `json:"content"`
	Model        string              `json:"model"`
	StopReason   *string             `json:"stop_reason"`
	StopSequence *string             `json:"stop_sequence"`
	Usage        *StreamUsage        `json:"usage,omitempty"`
}

// ContentBlockStart describes the content block opened by content_block_start.
type ContentBlockStart struct {
	Type     string `json:"type"`               // "text", "thinking", "tool_use"
	Text     string `json:"text,omitempty"`      // initial text (usually "")
	Thinking string `json:"thinking,omitempty"`  // initial thinking (usually "")
	ID       string `json:"id,omitempty"`        // tool_use block ID
	Name     string `json:"name,omitempty"`      // tool name
}

// DeltaContent carries the incremental payload inside content_block_delta and
// message_delta events.
type DeltaContent struct {
	Type           string  `json:"type"`                      // text_delta, thinking_delta, input_json_delta
	Text           string  `json:"text,omitempty"`            // for text_delta
	Thinking       string  `json:"thinking,omitempty"`        // for thinking_delta
	PartialJSON    string  `json:"partial_json,omitempty"`    // for input_json_delta
	StopReason     string  `json:"stop_reason,omitempty"`     // for message_delta
	StopSequence   *string `json:"stop_sequence,omitempty"`   // for message_delta
}

// StreamUsage reports token counts inside stream events.
type StreamUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// StreamError is an error reported inside the SSE stream itself.
type StreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ---------------------------------------------------------------------------
// Callback and interfaces
// ---------------------------------------------------------------------------

// StreamCallback is invoked for every parsed SSE event during streaming.
// Return a non-nil error to abort the stream early.
type StreamCallback func(event *StreamEvent) error

// StreamProvider extends Provider with a streaming Chat method.
type StreamProvider interface {
	Provider
	ChatStream(ctx context.Context, req *types.LLMRequest, cb StreamCallback) (*types.LLMResponse, error)
}

// ToolStreamProvider extends ToolProvider with a streaming ChatWithTools method.
type ToolStreamProvider interface {
	ToolProvider
	ChatWithToolsStream(ctx context.Context, req *types.LLMToolRequest, cb StreamCallback, opts ...StreamOptions) (*types.LLMToolResponse, error)
}

// ---------------------------------------------------------------------------
// SSE line-protocol parser
// ---------------------------------------------------------------------------

// SSEHandler is called for each fully assembled SSE frame (event + data).
type SSEHandler func(eventType string, data []byte) error

// ParseSSEStream reads an io.Reader as an SSE byte stream, assembling
// multi-line data fields and dispatching each complete frame via handler.
// It respects context cancellation and treats a "[DONE]" data payload as an
// end-of-stream marker.
func ParseSSEStream(ctx context.Context, r io.Reader, handler SSEHandler) error {
	scanner := bufio.NewScanner(r)
	// Allow large SSE frames (1 MB lines).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventType string
	var dataParts []string

	for scanner.Scan() {
		// Check for context cancellation between lines.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		// Blank line terminates a frame.
		if line == "" {
			if len(dataParts) > 0 {
				data := strings.Join(dataParts, "\n")

				// "[DONE]" is an optional end-of-stream sentinel.
				if data == "[DONE]" {
					return nil
				}

				if err := handler(eventType, []byte(data)); err != nil {
					return err
				}
			}
			eventType = ""
			dataParts = dataParts[:0]
			continue
		}

		// Parse SSE field.
		if after, ok := strings.CutPrefix(line, "event:"); ok {
			eventType = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "data:"); ok {
			dataParts = append(dataParts, strings.TrimSpace(after))
		}
		// Lines starting with ":" are comments; other fields (id, retry) are ignored.
	}

	// Flush any trailing frame without a final blank line.
	if len(dataParts) > 0 {
		data := strings.Join(dataParts, "\n")
		if data != "[DONE]" {
			if err := handler(eventType, []byte(data)); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

// ---------------------------------------------------------------------------
// Payload helpers
// ---------------------------------------------------------------------------

// ThinkingLevel controls extended-thinking budget.
type ThinkingLevel string

const (
	ThinkingOff    ThinkingLevel = ""
	ThinkingLow    ThinkingLevel = "low"
	ThinkingMedium ThinkingLevel = "medium"
	ThinkingHigh   ThinkingLevel = "high"
)

// thinkingBudget returns the budget_tokens value for a given level.
func thinkingBudget(level ThinkingLevel) int {
	switch level {
	case ThinkingLow:
		return 1024
	case ThinkingMedium:
		return 4096
	case ThinkingHigh:
		return 16384
	default:
		return 0
	}
}

// StreamOptions controls optional behaviour for streaming requests.
type StreamOptions struct {
	Thinking ThinkingLevel
}

// buildStreamPayload takes an LLMToolRequest and returns the JSON payload with
// "stream": true added. If opts is non-nil and thinking is enabled the
// appropriate thinking configuration is merged in.
func buildStreamPayload(req *types.LLMToolRequest, opts *StreamOptions) ([]byte, error) {
	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	payload := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   req.Messages,
		"stream":     true,
	}
	if req.SystemPrompt != "" {
		payload["system"] = req.SystemPrompt
	}
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = map[string]any{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": t.InputSchema,
			}
		}
		payload["tools"] = tools
	}

	// Extended thinking support.
	if opts != nil && opts.Thinking != ThinkingOff {
		budget := thinkingBudget(opts.Thinking)
		if budget > 0 {
			payload["thinking"] = map[string]any{
				"type":          "enabled",
				"budget_tokens": budget,
			}
			// Anthropic requires max_tokens >= budget_tokens for thinking.
			if maxTokens < budget+1024 {
				payload["max_tokens"] = budget + 1024
			}
		}
	}

	return json.Marshal(payload)
}

// ---------------------------------------------------------------------------
// Claude streaming methods
// ---------------------------------------------------------------------------

// ChatStream sends a streaming request for a simple chat (no tools) and
// accumulates the full response. The callback is invoked for every SSE event.
func (c *Claude) ChatStream(ctx context.Context, req *types.LLMRequest, cb StreamCallback) (*types.LLMResponse, error) {
	// Convert LLMRequest into an LLMToolRequest (no tools) so we can reuse
	// the shared payload builder.
	messages := make([]any, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = map[string]string{
			"role":    m.Role,
			"content": m.Content,
		}
	}
	toolReq := &types.LLMToolRequest{
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		Messages:     messages,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
	}

	body, err := buildStreamPayload(toolReq, nil)
	if err != nil {
		return nil, fmt.Errorf("build stream payload: %w", err)
	}

	resp, err := c.doStreamRequest(ctx, body, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var (
		textBuf      strings.Builder
		model        string
		inputTokens  int
		outputTokens int
	)

	parseErr := ParseSSEStream(ctx, resp.Body, func(eventType string, data []byte) error {
		var ev StreamEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			return fmt.Errorf("unmarshal SSE data: %w", err)
		}
		ev.Type = StreamEventType(eventType)

		// Accumulate state.
		switch ev.Type {
		case EventMessageStart:
			if ev.Message != nil {
				model = ev.Message.Model
				if ev.Message.Usage != nil {
					inputTokens = ev.Message.Usage.InputTokens
				}
			}
		case EventContentBlockDelta:
			if ev.Delta != nil && ev.Delta.Type == "text_delta" {
				textBuf.WriteString(ev.Delta.Text)
			}
		case EventMessageDelta:
			if ev.Usage != nil {
				outputTokens = ev.Usage.OutputTokens
			}
		case EventError:
			if ev.Error != nil {
				return fmt.Errorf("stream error: [%s] %s", ev.Error.Type, ev.Error.Message)
			}
		}

		if cb != nil {
			return cb(&ev)
		}
		return nil
	})
	if parseErr != nil {
		return nil, parseErr
	}

	return &types.LLMResponse{
		Content:      textBuf.String(),
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}, nil
}

// ChatWithToolsStream sends a streaming request with tool definitions and
// accumulates the full response including all content blocks. The callback
// is invoked for every SSE event.
func (c *Claude) ChatWithToolsStream(ctx context.Context, req *types.LLMToolRequest, cb StreamCallback, opts ...StreamOptions) (*types.LLMToolResponse, error) {
	var so *StreamOptions
	if len(opts) > 0 {
		so = &opts[0]
	}

	body, err := buildStreamPayload(req, so)
	if err != nil {
		return nil, fmt.Errorf("build stream payload: %w", err)
	}

	resp, err := c.doStreamRequest(ctx, body, so)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// We accumulate content blocks as they open/close.
	type blockState struct {
		blockType   string
		text        strings.Builder
		thinking    strings.Builder
		toolID      string
		toolName    string
		inputJSON   strings.Builder
	}

	var (
		blocks       []blockState
		activeIndex  int
		model        string
		stopReason   string
		inputTokens  int
		outputTokens int
	)

	parseErr := ParseSSEStream(ctx, resp.Body, func(eventType string, data []byte) error {
		var ev StreamEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			return fmt.Errorf("unmarshal SSE data: %w", err)
		}
		ev.Type = StreamEventType(eventType)

		switch ev.Type {
		case EventMessageStart:
			if ev.Message != nil {
				model = ev.Message.Model
				if ev.Message.Usage != nil {
					inputTokens = ev.Message.Usage.InputTokens
				}
			}

		case EventContentBlockStart:
			activeIndex = ev.Index
			// Grow the slice to accommodate the new block.
			for len(blocks) <= activeIndex {
				blocks = append(blocks, blockState{})
			}
			if ev.ContentBlock != nil {
				blocks[activeIndex].blockType = ev.ContentBlock.Type
				if ev.ContentBlock.Text != "" {
					blocks[activeIndex].text.WriteString(ev.ContentBlock.Text)
				}
				if ev.ContentBlock.Thinking != "" {
					blocks[activeIndex].thinking.WriteString(ev.ContentBlock.Thinking)
				}
				blocks[activeIndex].toolID = ev.ContentBlock.ID
				blocks[activeIndex].toolName = ev.ContentBlock.Name
			}

		case EventContentBlockDelta:
			activeIndex = ev.Index
			if ev.Delta != nil && activeIndex < len(blocks) {
				switch ev.Delta.Type {
				case "text_delta":
					blocks[activeIndex].text.WriteString(ev.Delta.Text)
				case "thinking_delta":
					blocks[activeIndex].thinking.WriteString(ev.Delta.Thinking)
				case "input_json_delta":
					blocks[activeIndex].inputJSON.WriteString(ev.Delta.PartialJSON)
				}
			}

		case EventContentBlockStop:
			// Nothing to do; the block is already accumulated.

		case EventMessageDelta:
			if ev.Delta != nil {
				stopReason = ev.Delta.StopReason
			}
			if ev.Usage != nil {
				outputTokens = ev.Usage.OutputTokens
			}

		case EventError:
			if ev.Error != nil {
				return fmt.Errorf("stream error: [%s] %s", ev.Error.Type, ev.Error.Message)
			}
		}

		if cb != nil {
			return cb(&ev)
		}
		return nil
	})
	if parseErr != nil {
		return nil, parseErr
	}

	// Convert accumulated blocks into types.ContentBlock.
	contentBlocks := make([]types.ContentBlock, 0, len(blocks))
	for _, b := range blocks {
		switch b.blockType {
		case "text":
			contentBlocks = append(contentBlocks, types.ContentBlock{
				Type: "text",
				Text: b.text.String(),
			})
		case "thinking":
			contentBlocks = append(contentBlocks, types.ContentBlock{
				Type: "thinking",
				Text: b.thinking.String(),
			})
		case "tool_use":
			cb := types.ContentBlock{
				Type: "tool_use",
				ID:   b.toolID,
				Name: b.toolName,
			}
			// Parse accumulated JSON input.
			raw := b.inputJSON.String()
			if raw != "" {
				var input map[string]any
				if err := json.Unmarshal([]byte(raw), &input); err == nil {
					cb.Input = input
				}
			}
			contentBlocks = append(contentBlocks, cb)
		}
	}

	return &types.LLMToolResponse{
		Content:      contentBlocks,
		Model:        model,
		StopReason:   stopReason,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}, nil
}

// doStreamRequest builds and executes the HTTP request for a streaming call.
// It sets the appropriate headers including the thinking beta header when
// extended thinking is enabled.
func (c *Claude) doStreamRequest(ctx context.Context, body []byte, opts *StreamOptions) (*http.Response, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url()+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Extended thinking requires a beta header.
	if opts != nil && opts.Thinking != ThinkingOff {
		httpReq.Header.Set("anthropic-beta", "interleaved-thinking-2025-05-14")
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic API %d: %s", resp.StatusCode, respBody)
	}

	return resp, nil
}
