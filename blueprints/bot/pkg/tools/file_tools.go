package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditFileTool returns a tool that edits a file by replacing old text with new text.
func EditFileTool() *Tool {
	return &Tool{
		Name:        "edit",
		Description: "Edit a file by replacing old text with new text. The old_string must be unique in the file.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to edit",
				},
				"old_string": map[string]any{
					"type":        "string",
					"description": "The exact text to find and replace (must be unique in the file)",
				},
				"new_string": map[string]any{
					"type":        "string",
					"description": "The text to replace old_string with",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
		Execute: func(_ context.Context, input map[string]any) (string, error) {
			path := getStringParam(input, "path")
			oldStr := getStringParam(input, "old_string")
			newStr := getStringParam(input, "new_string")

			if oldStr == "" {
				return "old_string must not be empty", nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err.Error(), nil
			}

			content := string(data)
			count := strings.Count(content, oldStr)

			if count == 0 {
				return "old_string not found in file", nil
			}
			if count > 1 {
				return fmt.Sprintf("old_string is not unique in the file (found %d occurrences)", count), nil
			}

			updated := strings.Replace(content, oldStr, newStr, 1)
			if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
				return err.Error(), nil
			}

			return fmt.Sprintf("edited %s", path), nil
		},
	}
}

// WriteFileTool returns a tool that writes content to a file, creating it if needed.
func WriteFileTool() *Tool {
	return &Tool{
		Name:        "write",
		Description: "Write content to a file, creating it if needed or overwriting existing content.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to write",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
		Execute: func(_ context.Context, input map[string]any) (string, error) {
			path := getStringParam(input, "path")
			content := getStringParam(input, "content")

			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err.Error(), nil
			}

			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err.Error(), nil
			}

			return fmt.Sprintf("wrote %d bytes to %s", len(content), path), nil
		},
	}
}
