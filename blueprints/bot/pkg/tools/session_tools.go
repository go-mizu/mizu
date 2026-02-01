package tools

import "context"

// SessionsListTool returns a tool that lists stored conversation sessions.
func SessionsListTool() *Tool {
	return &Tool{
		Name:        "sessions_list",
		Description: "List stored conversation sessions.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "No session store available in embedded mode.", nil
		},
	}
}

// SessionsHistoryTool returns a tool that gets conversation history for a session.
func SessionsHistoryTool() *Tool {
	return &Tool{
		Name:        "sessions_history",
		Description: "Get conversation history for a session.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":        "string",
					"description": "The session ID to retrieve history for",
				},
			},
			"required": []string{"session_id"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "Session history not available in embedded mode.", nil
		},
	}
}

// SessionStatusTool returns a tool that gets the status of the current session.
func SessionStatusTool() *Tool {
	return &Tool{
		Name:        "session_status",
		Description: "Get the status of the current session.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":        "string",
					"description": "Optional session ID to check status for",
				},
			},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "Session status not available in embedded mode.", nil
		},
	}
}

// SessionsSendTool returns a tool that sends a message to an existing session.
func SessionsSendTool() *Tool {
	return &Tool{
		Name:        "sessions_send",
		Description: "Send a message to an existing session.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":        "string",
					"description": "The session ID to send the message to",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "The message to send",
				},
			},
			"required": []string{"session_id", "message"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "sessions_send is not available in embedded mode. Use the gateway for session messaging.", nil
		},
	}
}

// SessionsSpawnTool returns a tool that spawns a new agent session.
func SessionsSpawnTool() *Tool {
	return &Tool{
		Name:        "sessions_spawn",
		Description: "Spawn a new agent session.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent_id": map[string]any{
					"type":        "string",
					"description": "Optional agent ID to spawn the session for",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "The initial message for the new session",
				},
			},
			"required": []string{"message"},
		},
		Execute: func(_ context.Context, _ map[string]any) (string, error) {
			return "sessions_spawn is not available in embedded mode. Use the gateway for session spawning.", nil
		},
	}
}
