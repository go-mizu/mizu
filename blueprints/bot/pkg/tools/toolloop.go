package tools

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Broadcaster sends real-time events to connected clients.
type Broadcaster interface {
	Broadcast(event string, payload any)
}

// MaxToolIterations is the maximum number of tool call rounds before giving up.
const MaxToolIterations = 10

// RunToolLoop runs the tool execution loop. It calls ChatWithTools, executes
// any tool calls, and repeats until the LLM produces a final text response
// or the iteration limit is reached.
func RunToolLoop(
	ctx context.Context,
	provider llm.ToolProvider,
	registry *Registry,
	req *types.LLMToolRequest,
) (*types.LLMToolResponse, error) {
	for i := 0; i < MaxToolIterations; i++ {
		resp, err := provider.ChatWithTools(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("tool loop iteration %d: %w", i, err)
		}

		// If the LLM is done (no more tool calls), return the final response.
		if resp.StopReason != "tool_use" {
			return resp, nil
		}

		toolUses := resp.ToolUses()
		if len(toolUses) == 0 {
			return resp, nil
		}

		// Append the assistant's response (with tool_use blocks) as an assistant message.
		// The Anthropic API expects the assistant message to contain the content blocks as-is.
		assistantContent := make([]any, len(resp.Content))
		for j, block := range resp.Content {
			assistantContent[j] = block
		}
		req.Messages = append(req.Messages, map[string]any{
			"role":    "assistant",
			"content": assistantContent,
		})

		// Execute each tool and build tool_result blocks.
		toolResults := make([]any, 0, len(toolUses))
		for _, use := range toolUses {
			tool := registry.Get(use.Name)
			var resultContent string
			var isError bool

			if tool == nil {
				resultContent = fmt.Sprintf("Unknown tool: %s", use.Name)
				isError = true
			} else {
				log.Printf("Tool call: %s(%v)", use.Name, use.Input)
				result, err := tool.Execute(ctx, use.Input)
				if err != nil {
					resultContent = fmt.Sprintf("Error: %v", err)
					isError = true
				} else {
					resultContent = result
				}
			}

			toolResults = append(toolResults, types.ToolResultBlock{
				Type:      "tool_result",
				ToolUseID: use.ID,
				Content:   resultContent,
				IsError:   isError,
			})
		}

		// Append tool results as a user message.
		req.Messages = append(req.Messages, map[string]any{
			"role":    "user",
			"content": toolResults,
		})
	}

	return nil, fmt.Errorf("tool loop exceeded %d iterations", MaxToolIterations)
}

// RunToolLoopStream runs the tool loop with streaming support.
// On each LLM turn it uses ChatWithToolsStream with the callback for deltas.
// Tool executions are broadcast as agent events via the broadcaster.
func RunToolLoopStream(
	ctx context.Context,
	provider llm.ToolStreamProvider,
	registry *Registry,
	req *types.LLMToolRequest,
	cb llm.StreamCallback,
	broadcaster Broadcaster,
	runID, sessionKey string,
) (*types.LLMToolResponse, error) {
	for i := 0; i < MaxToolIterations; i++ {
		var opts []llm.StreamOptions
		if req.ThinkingLevel != "" && req.ThinkingLevel != "off" {
			opts = append(opts, llm.StreamOptions{Thinking: llm.ThinkingLevel(req.ThinkingLevel)})
		}

		resp, err := provider.ChatWithToolsStream(ctx, req, cb, opts...)
		if err != nil {
			return nil, fmt.Errorf("tool stream loop iteration %d: %w", i, err)
		}

		if resp.StopReason != "tool_use" {
			return resp, nil
		}

		toolUses := resp.ToolUses()
		if len(toolUses) == 0 {
			return resp, nil
		}

		// Append assistant message.
		assistantContent := make([]any, len(resp.Content))
		for j, block := range resp.Content {
			assistantContent[j] = block
		}
		req.Messages = append(req.Messages, map[string]any{
			"role": "assistant", "content": assistantContent,
		})

		// Execute tools with event broadcasting.
		toolResults := make([]any, 0, len(toolUses))
		for _, use := range toolUses {
			// Broadcast tool start event.
			if broadcaster != nil {
				broadcaster.Broadcast("agent", map[string]any{
					"runId": runID, "sessionKey": sessionKey,
					"ts": time.Now().UnixMilli(),
					"data": map[string]any{
						"type": "tool_start", "toolName": use.Name,
						"toolCallId": use.ID,
					},
				})
			}

			tool := registry.Get(use.Name)
			var resultContent string
			var isError bool
			start := time.Now()

			if tool == nil {
				resultContent = fmt.Sprintf("Unknown tool: %s", use.Name)
				isError = true
			} else {
				log.Printf("Tool call: %s(%v)", use.Name, use.Input)
				result, toolErr := tool.Execute(ctx, use.Input)
				if toolErr != nil {
					resultContent = fmt.Sprintf("Error: %v", toolErr)
					isError = true
				} else {
					resultContent = result
				}
			}

			elapsed := time.Since(start)

			// Broadcast tool end event.
			if broadcaster != nil {
				broadcaster.Broadcast("agent", map[string]any{
					"runId": runID, "sessionKey": sessionKey,
					"ts": time.Now().UnixMilli(),
					"data": map[string]any{
						"type": "tool_end", "toolName": use.Name,
						"toolCallId": use.ID, "durationMs": elapsed.Milliseconds(),
						"isError": isError,
					},
				})
			}

			toolResults = append(toolResults, types.ToolResultBlock{
				Type: "tool_result", ToolUseID: use.ID,
				Content: resultContent, IsError: isError,
			})
		}

		req.Messages = append(req.Messages, map[string]any{
			"role": "user", "content": toolResults,
		})
	}

	return nil, fmt.Errorf("tool stream loop exceeded %d iterations", MaxToolIterations)
}
