package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/memory"
)

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

// MemorySearchTool returns a stub tool registered when no MemoryManager is available.
func MemorySearchTool() *Tool {
	return &Tool{
		Name:        "memory_search",
		Description: "Search your indexed memory (workspace files + session transcripts) for relevant context. Returns text snippets with source paths and relevance scores. Use when you need to recall past conversations, decisions, or stored knowledge.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results (default 6)",
				},
				"source": map[string]any{
					"type":        "string",
					"enum":        []string{"memory", "sessions", "all"},
					"description": "Filter by source type: 'memory' for workspace files, 'sessions' for conversation transcripts, 'all' for both (default: all)",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "Memory search is not available — no memory manager configured.", nil
		},
	}
}

// MemoryGetTool returns a stub tool registered when no MemoryManager is available.
func MemoryGetTool() *Tool {
	return &Tool{
		Name:        "memory_get",
		Description: "Read specific lines from a file in the memory index. Use after memory_search to get full context around a snippet.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File path (relative to workspace)",
				},
				"from": map[string]any{
					"type":        "integer",
					"description": "Start line number (1-based, default 1)",
				},
				"count": map[string]any{
					"type":        "integer",
					"description": "Number of lines to read (default 20)",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "Memory get is not available — no memory manager configured.", nil
		},
	}
}

// RegisterMemoryTools replaces the stub memory tools with real implementations
// backed by the given MemoryManager.
func RegisterMemoryTools(r *Registry, mgr *memory.MemoryManager) {
	if mgr == nil {
		return
	}

	// Override memory_search with real implementation.
	r.Register(&Tool{
		Name:        "memory_search",
		Description: "Search your indexed memory (workspace files + session transcripts) for relevant context. Returns text snippets with source paths and relevance scores. Use when you need to recall past conversations, decisions, or stored knowledge.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results (default 6)",
				},
				"source": map[string]any{
					"type":        "string",
					"enum":        []string{"memory", "sessions", "all"},
					"description": "Filter by source type: 'memory' for workspace files, 'sessions' for conversation transcripts, 'all' for both (default: all)",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(ctx context.Context, input map[string]any) (string, error) {
			query := getStringParam(input, "query")
			if query == "" {
				return "Error: query is required", nil
			}
			limit := getIntParam(input, "limit")
			if limit <= 0 {
				limit = 6
			}
			source := getStringParam(input, "source")

			results, err := mgr.Search(ctx, query, limit, 0)
			if err != nil {
				return fmt.Sprintf("Search error: %v", err), nil
			}

			// Filter by source if specified.
			if source != "" && source != "all" {
				var filtered []memory.SearchResult
				for _, r := range results {
					if r.Source == source {
						filtered = append(filtered, r)
					}
				}
				results = filtered
			}

			if len(results) == 0 {
				return "No results found.", nil
			}

			var b strings.Builder
			fmt.Fprintf(&b, "Found %d results:\n\n", len(results))
			for i, r := range results {
				src := r.Source
				if src == "" {
					src = "memory"
				}
				fmt.Fprintf(&b, "[%d] %s (lines %d-%d, score: %.2f, source: %s)\n",
					i+1, r.Path, r.StartLine, r.EndLine, r.Score, src)
				b.WriteString(r.Snippet)
				if !strings.HasSuffix(r.Snippet, "\n") {
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}
			return b.String(), nil
		},
	})

	// Override memory_get with real implementation.
	r.Register(&Tool{
		Name:        "memory_get",
		Description: "Read specific lines from a file in the memory index. Use after memory_search to get full context around a snippet.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File path (relative to workspace)",
				},
				"from": map[string]any{
					"type":        "integer",
					"description": "Start line number (1-based, default 1)",
				},
				"count": map[string]any{
					"type":        "integer",
					"description": "Number of lines to read (default 20)",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(_ context.Context, input map[string]any) (string, error) {
			path := getStringParam(input, "path")
			if path == "" {
				return "Error: path is required", nil
			}
			from := getIntParam(input, "from")
			if from <= 0 {
				from = 1
			}
			count := getIntParam(input, "count")
			if count <= 0 {
				count = 20
			}

			content, err := mgr.GetLines(path, from, count)
			if err != nil {
				return fmt.Sprintf("Error reading %s: %v", path, err), nil
			}
			if content == "" {
				return fmt.Sprintf("No content at %s lines %d-%d", path, from, from+count-1), nil
			}
			return content, nil
		},
	})
}
