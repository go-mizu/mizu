package tools

import "context"

// getIntParam extracts an integer parameter from the input map.
func getIntParam(input map[string]any, key string) int {
	v, ok := input[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

// MemorySearchTool returns a tool that searches the memory index for relevant content.
func MemorySearchTool() *Tool {
	return &Tool{
		Name:        "memory_search",
		Description: "Search the memory index for relevant content.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query",
				},
				"max_results": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results to return",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "memory_search is not available in embedded mode. Use the gateway for memory search.", nil
		},
	}
}

// MemoryGetTool returns a tool that gets a specific memory entry by path.
func MemoryGetTool() *Tool {
	return &Tool{
		Name:        "memory_get",
		Description: "Get a specific memory entry by path.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The path of the memory entry to retrieve",
				},
				"start_line": map[string]any{
					"type":        "integer",
					"description": "Optional start line number",
				},
				"end_line": map[string]any{
					"type":        "integer",
					"description": "Optional end line number",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "memory_get is not available in embedded mode. Use the gateway for memory retrieval.", nil
		},
	}
}
