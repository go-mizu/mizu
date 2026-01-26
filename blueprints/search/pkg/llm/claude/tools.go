package claude

import (
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
)

// Tool input types for type-safe parsing.

// WebSearchInput represents the input for web search tool.
type WebSearchInput struct {
	Query string `json:"query"`
}

// URLFetchInput represents the input for URL fetch tool.
type URLFetchInput struct {
	URL string `json:"url"`
}

// CodeExecutionInput represents the input for code execution tool.
type CodeExecutionInput struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

// ToolDefinitions contains all available tools for Claude.
var ToolDefinitions = []llm.Tool{
	{
		Name:        "web_search",
		Description: "Search the web for current information. Use for facts, news, recent events, documentation, or any information that may have changed since your knowledge cutoff.",
		InputSchema: llm.JSONSchema{
			Type: "object",
			Properties: map[string]llm.Property{
				"query": {
					Type:        "string",
					Description: "The search query. Be specific and include relevant keywords.",
				},
			},
			Required: []string{"query"},
		},
	},
	{
		Name:        "fetch_url",
		Description: "Fetch and read the content from a specific URL. Use when you need to read the full content of a webpage, article, or documentation page.",
		InputSchema: llm.JSONSchema{
			Type: "object",
			Properties: map[string]llm.Property{
				"url": {
					Type:        "string",
					Description: "The complete URL to fetch (must start with http:// or https://).",
				},
			},
			Required: []string{"url"},
		},
	},
	{
		Name:        "execute_code",
		Description: "Execute Python code for calculations, data processing, analysis, or generating output. The code runs in a sandboxed environment. Returns stdout and stderr.",
		InputSchema: llm.JSONSchema{
			Type: "object",
			Properties: map[string]llm.Property{
				"code": {
					Type:        "string",
					Description: "The Python code to execute. Use print() to output results.",
				},
				"language": {
					Type:        "string",
					Description: "The programming language (currently only 'python' is supported).",
					Enum:        []string{"python"},
				},
			},
			Required: []string{"code"},
		},
	},
}

// GetToolByName returns a tool definition by name.
func GetToolByName(name string) *llm.Tool {
	for i := range ToolDefinitions {
		if ToolDefinitions[i].Name == name {
			return &ToolDefinitions[i]
		}
	}
	return nil
}

// GetSearchTools returns tools appropriate for search/research mode.
func GetSearchTools() []llm.Tool {
	return []llm.Tool{
		ToolDefinitions[0], // web_search
		ToolDefinitions[1], // fetch_url
	}
}

// GetAllTools returns all available tools.
func GetAllTools() []llm.Tool {
	return ToolDefinitions
}
