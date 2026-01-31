package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/tools"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

const maxToolIterations = 10

// runToolLoop runs the tool execution loop. It calls ChatWithTools, executes
// any tool calls, and repeats until the LLM produces a final text response
// or the iteration limit is reached.
func runToolLoop(
	ctx context.Context,
	provider llm.ToolProvider,
	registry *tools.Registry,
	req *types.LLMToolRequest,
) (*types.LLMToolResponse, error) {
	for i := 0; i < maxToolIterations; i++ {
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

	return nil, fmt.Errorf("tool loop exceeded %d iterations", maxToolIterations)
}
