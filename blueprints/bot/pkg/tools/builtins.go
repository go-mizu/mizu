package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RegisterBuiltins adds all built-in tools to the registry.
// Tool names match OpenClaw's tool naming convention.
func RegisterBuiltins(r *Registry) {
	// File tools (OpenClaw: read, edit, write).
	r.Register(ReadFileTool())   // "read_file" — also covers OpenClaw's "read"
	r.Register(ListFilesTool())  // "list_files"
	r.Register(EditFileTool())   // "edit"
	r.Register(WriteFileTool())  // "write"

	// System tools (OpenClaw: exec, process).
	r.Register(RunCommandTool()) // "run_command" — covers OpenClaw's "exec"
	r.Register(ProcessTool())    // "process"

	// Web tools (OpenClaw: web_search, web_fetch).
	r.Register(WebSearchTool()) // "web_search"
	r.Register(WebFetchTool())  // "web_fetch"

	// Session tools (OpenClaw: sessions_list, sessions_history, session_status, sessions_send, sessions_spawn).
	r.Register(SessionsListTool())    // "sessions_list"
	r.Register(SessionsHistoryTool()) // "sessions_history"
	r.Register(SessionStatusTool())   // "session_status"
	r.Register(SessionsSendTool())    // "sessions_send"
	r.Register(SessionsSpawnTool())   // "sessions_spawn"

	// Memory tools (OpenClaw: memory_search, memory_get).
	r.Register(MemorySearchTool()) // "memory_search"
	r.Register(MemoryGetTool())    // "memory_get"

	// Communication tools (OpenClaw: message, tts).
	r.Register(MessageTool()) // "message"
	r.Register(TTSTool())     // "tts"

	// Agent management (OpenClaw: agents_list).
	r.Register(AgentsListTool()) // "agents_list"

	// Infrastructure tools (OpenClaw: gateway, cron, browser, canvas, nodes).
	r.Register(GatewayTool()) // "gateway"
	r.Register(CronTool())    // "cron"
	r.Register(BrowserTool()) // "browser"
	r.Register(CanvasTool())  // "canvas"
	r.Register(NodesTool())   // "nodes"

	// Media tools (OpenClaw: image).
	r.Register(ImageTool()) // "image"
}

// getStringParam extracts a string parameter from the input map.
func getStringParam(input map[string]any, key string) string {
	v, ok := input[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// getBoolParam extracts a boolean parameter from the input map.
func getBoolParam(input map[string]any, key string) bool {
	v, ok := input[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}

// ListFilesTool returns a tool that lists files in a directory.
func ListFilesTool() *Tool {
	return &Tool{
		Name:        "list_files",
		Description: "List files in a directory. Returns filename, size, and modification date for each file.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Directory path to list",
				},
				"pattern": map[string]any{
					"type":        "string",
					"description": "Optional glob pattern to filter files (e.g. *.pdf)",
				},
				"recursive": map[string]any{
					"type":        "boolean",
					"description": "Whether to search recursively in subdirectories",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(_ context.Context, input map[string]any) (string, error) {
			dir := getStringParam(input, "path")
			pattern := getStringParam(input, "pattern")
			recursive := getBoolParam(input, "recursive")

			var result string

			if recursive {
				err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.IsDir() {
						return nil
					}
					if pattern != "" {
						matched, matchErr := filepath.Match(pattern, d.Name())
						if matchErr != nil {
							return matchErr
						}
						if !matched {
							return nil
						}
					}
					info, infoErr := d.Info()
					if infoErr != nil {
						return infoErr
					}
					result += fmt.Sprintf("  %d  %s  %s\n", info.Size(), info.ModTime().Format(time.DateOnly), path)
					return nil
				})
				if err != nil {
					return err.Error(), nil
				}
			} else {
				entries, err := os.ReadDir(dir)
				if err != nil {
					return err.Error(), nil
				}
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					if pattern != "" {
						matched, matchErr := filepath.Match(pattern, entry.Name())
						if matchErr != nil {
							return matchErr.Error(), nil
						}
						if !matched {
							continue
						}
					}
					info, infoErr := entry.Info()
					if infoErr != nil {
						return infoErr.Error(), nil
					}
					result += fmt.Sprintf("  %d  %s  %s\n", info.Size(), info.ModTime().Format(time.DateOnly), entry.Name())
				}
			}

			if result == "" {
				return "no files found", nil
			}
			return result, nil
		},
	}
}

// ReadFileTool returns a tool that reads the contents of a text file.
func ReadFileTool() *Tool {
	return &Tool{
		Name:        "read_file",
		Description: "Read the contents of a text file. Large files are truncated at 100KB.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to read",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(_ context.Context, input map[string]any) (string, error) {
			path := getStringParam(input, "path")

			data, err := os.ReadFile(path)
			if err != nil {
				return err.Error(), nil
			}

			const maxSize = 100 * 1024
			if len(data) > maxSize {
				return string(data[:maxSize]) + fmt.Sprintf("\n... [truncated, file is %d bytes total]", len(data)), nil
			}
			return string(data), nil
		},
	}
}

// RunCommandTool returns a tool that executes a shell command.
func RunCommandTool() *Tool {
	return &Tool{
		Name:        "run_command",
		Description: "Execute a shell command and return its output. Commands have a 30-second timeout and output is limited to 50KB.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Shell command to execute",
				},
				"workdir": map[string]any{
					"type":        "string",
					"description": "Working directory (optional, defaults to home directory)",
				},
			},
			"required": []string{"command"},
		},
		Execute: func(ctx context.Context, input map[string]any) (string, error) {
			command := getStringParam(input, "command")
			workdir := getStringParam(input, "workdir")

			if workdir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return err.Error(), nil
				}
				workdir = home
			}

			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, "sh", "-c", command)
			cmd.Dir = workdir

			output, err := cmd.CombinedOutput()
			if ctx.Err() == context.DeadlineExceeded {
				return "command timed out after 30s", nil
			}

			const maxOutput = 50 * 1024
			if len(output) > maxOutput {
				return string(output[:maxOutput]) + fmt.Sprintf("\n... [truncated, %d bytes total]", len(output)), nil
			}

			if err != nil {
				return string(output) + "\n" + err.Error(), nil
			}
			return string(output), nil
		},
	}
}
