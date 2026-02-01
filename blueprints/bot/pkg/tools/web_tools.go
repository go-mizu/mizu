package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WebSearchTool returns a stub tool for web search.
func WebSearchTool() *Tool {
	return &Tool{
		Name:        "web_search",
		Description: "Search the web using a query string. Returns search result snippets.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query string",
				},
				"max_results": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results to return (default 5)",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "Web search not available in embedded mode. Use the gateway for web search capabilities.", nil
		},
	}
}

// WebFetchTool returns a tool that fetches content from a URL.
func WebFetchTool() *Tool {
	return &Tool{
		Name:        "web_fetch",
		Description: "Fetch content from a URL and return it as text.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "The URL to fetch content from",
				},
				"max_bytes": map[string]any{
					"type":        "integer",
					"description": "Maximum number of bytes to read (default 100KB)",
				},
			},
			"required": []string{"url"},
		},
		Execute: func(ctx context.Context, input map[string]any) (string, error) {
			rawURL := getStringParam(input, "url")
			maxBytes := getIntParam(input, "max_bytes")
			if maxBytes <= 0 {
				maxBytes = 100 * 1024
			}

			if rawURL == "" {
				return "url is required", nil
			}

			client := &http.Client{Timeout: 30 * time.Second}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
			if err != nil {
				return fmt.Sprintf("invalid request: %v", err), nil
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Sprintf("fetch failed: %v", err), nil
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
			if err != nil {
				return fmt.Sprintf("read failed: %v", err), nil
			}

			if resp.StatusCode >= 400 {
				return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), nil
			}

			return string(body), nil
		},
	}
}
