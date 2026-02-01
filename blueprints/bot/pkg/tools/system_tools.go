package tools

import "context"

// AgentsListTool returns a tool that lists configured agents.
func AgentsListTool() *Tool {
	return &Tool{
		Name:        "agents_list",
		Description: "List configured agents.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "agents_list is not available in embedded mode. Use the gateway for agent management.", nil
		},
	}
}

// ProcessTool returns a tool that manages background processes.
func ProcessTool() *Tool {
	return &Tool{
		Name:        "process",
		Description: "Manage background processes.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "Action to perform: list, kill, or status",
					"enum":        []string{"list", "kill", "status"},
				},
				"pid": map[string]any{
					"type":        "string",
					"description": "Process ID (required for kill and status actions)",
				},
			},
			"required": []string{"action"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "process is not available in embedded mode. Use the gateway for process management.", nil
		},
	}
}

// MessageTool returns a tool that sends a message via a channel.
func MessageTool() *Tool {
	return &Tool{
		Name:        "message",
		Description: "Send a message via a channel.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"channel": map[string]any{
					"type":        "string",
					"description": "The channel to send the message through",
				},
				"target": map[string]any{
					"type":        "string",
					"description": "The target recipient",
				},
				"text": map[string]any{
					"type":        "string",
					"description": "The message text to send",
				},
			},
			"required": []string{"channel", "target", "text"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "message is not available in embedded mode. Use the gateway for messaging.", nil
		},
	}
}

// CronTool returns a tool that manages cron jobs.
func CronTool() *Tool {
	return &Tool{
		Name:        "cron",
		Description: "Manage cron jobs.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "The cron action to perform",
				},
			},
			"required": []string{"action"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "cron is not available in embedded mode. Use the gateway for cron management.", nil
		},
	}
}

// GatewayTool returns a tool for gateway control.
func GatewayTool() *Tool {
	return &Tool{
		Name:        "gateway",
		Description: "Gateway control.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "The gateway action to perform",
				},
			},
			"required": []string{"action"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "gateway is not available in embedded mode. Use the gateway for gateway control.", nil
		},
	}
}

// TTSTool returns a tool for text-to-speech.
func TTSTool() *Tool {
	return &Tool{
		Name:        "tts",
		Description: "Text-to-speech.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{
					"type":        "string",
					"description": "The text to convert to speech",
				},
			},
			"required": []string{"text"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "tts is not available in embedded mode. Use the gateway for text-to-speech.", nil
		},
	}
}

// BrowserTool returns a tool for browser control.
func BrowserTool() *Tool {
	return &Tool{
		Name:        "browser",
		Description: "Browser control.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "The browser action to perform",
				},
				"url": map[string]any{
					"type":        "string",
					"description": "Optional URL for the browser action",
				},
			},
			"required": []string{"action"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "browser is not available in embedded mode. Use the gateway for browser control.", nil
		},
	}
}

// CanvasTool returns a tool for canvas/HTML rendering.
func CanvasTool() *Tool {
	return &Tool{
		Name:        "canvas",
		Description: "Canvas/HTML rendering.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "The canvas action to perform",
				},
			},
			"required": []string{"action"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "canvas is not available in embedded mode. Use the gateway for canvas rendering.", nil
		},
	}
}

// NodesTool returns a tool for node management.
func NodesTool() *Tool {
	return &Tool{
		Name:        "nodes",
		Description: "Node management.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "The node management action to perform",
				},
			},
			"required": []string{"action"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "nodes is not available in embedded mode. Use the gateway for node management.", nil
		},
	}
}

// ImageTool returns a tool for image generation and editing.
func ImageTool() *Tool {
	return &Tool{
		Name:        "image",
		Description: "Image generation and editing.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "The image action to perform",
				},
				"prompt": map[string]any{
					"type":        "string",
					"description": "Optional prompt for image generation",
				},
			},
			"required": []string{"action"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "image is not available in embedded mode. Use the gateway for image generation.", nil
		},
	}
}
