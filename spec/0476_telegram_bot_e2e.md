# 0476: Telegram Bot E2E - Tool Execution & CLI

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enhance openbot with a tool execution loop so Claude can read files, list directories, and run commands — then add CLI subcommands for session management and message sending.

**Architecture:** Add an Anthropic tool-use loop to the bot engine. When the LLM returns `stop_reason: tool_use`, execute the requested tool locally and feed the result back until the LLM produces a final text response. Expose CLI subcommands (`sessions`, `history`, `send`, `status`) that interact with the SQLite store directly.

**Tech Stack:** Go, Anthropic Messages API (tool_use), SQLite, Cobra CLI

---

## Research: How OpenClaw Handles This

### OpenClaw Session Format

OpenClaw stores sessions as JSONL files in `~/.openclaw/agents/main/sessions/`. Each line is a JSON object with a `type` field:

- `type: "session"` — Session header (version, ID, timestamp, cwd)
- `type: "model_change"` — Model switches
- `type: "message"` — Conversation entries with nested `message.role` (user/assistant/toolResult) and `message.content` (list of blocks: text, thinking, toolCall)

### OpenClaw Tool Architecture

OpenClaw provides 22+ tools through a pi-agent-core framework:
- **read** — Read files (path param, returns content or base64 for images)
- **write** — Write files (path + content)
- **edit** — Edit files (path + oldText + newText)
- **exec** — Execute shell commands (command, workdir, env, timeout, pty, background)
- **process** — Background process management
- **browser** — Browser automation
- **memory_search/memory_get** — Memory tools
- **web_search/web_fetch** — Web tools
- **message** — Send messages to channels
- **sessions_list/sessions_history/sessions_send** — Session management

### OpenClaw PDF Scenario (from live session)

In the observed session at `~/.openclaw/agents/main/sessions/08ef14de-*.jsonl`:

1. User asks "How many PDFs in my Downloads folder?"
2. Agent uses `exec` tool: `find ~/Downloads -name "*.pdf" | wc -l` → 181
3. Agent responds: "You've got 181 PDF files"
4. User asks "List the biggest ones"
5. Agent uses `exec` tool: `du -sh ~/Downloads/*.pdf | sort -rh | head -20`
6. Agent formats results as a markdown table
7. User asks "Organize them into folders"
8. Agent uses `exec` tool multiple times: creates folders, moves files by category

Key pattern: The LLM decides which tool to call, the system executes it, returns results, and the LLM formulates its final response.

### Our Implementation Strategy

We implement a subset of OpenClaw's tools using the **Anthropic Messages API tool_use protocol**:

1. Define tools as Anthropic-compatible JSON schemas
2. Send tools in the API request payload
3. When `stop_reason: "tool_use"`, extract tool calls from response content blocks
4. Execute each tool locally
5. Append tool results as `role: "user"` messages with `tool_result` content blocks
6. Loop until `stop_reason: "end_turn"`

---

## Implementation Plan

### Task 1: Tool Types and Registry

**Files:**
- Create: `pkg/tools/tools.go`
- Create: `pkg/tools/tools_test.go`

Define the tool abstraction:

```go
// Tool defines a tool that the LLM can invoke.
type Tool struct {
    Name        string
    Description string
    InputSchema map[string]any // JSON Schema for parameters
    Execute     func(ctx context.Context, input map[string]any) (string, error)
}

// Registry holds all available tools.
type Registry struct {
    tools map[string]*Tool
}

func NewRegistry() *Registry
func (r *Registry) Register(t *Tool)
func (r *Registry) Get(name string) *Tool
func (r *Registry) All() []*Tool
func (r *Registry) Definitions() []map[string]any // Anthropic API format
```

Built-in tools to register:
- `list_files` — List files in a directory, optional glob pattern
- `read_file` — Read a file's content (text only, size limit 100KB)
- `run_command` — Execute a shell command (timeout 30s, output limit 50KB)

**Tests:**
- TestRegistry_RegisterAndGet
- TestRegistry_Definitions (verify Anthropic JSON schema format)

---

### Task 2: Built-in Tool Implementations

**Files:**
- Create: `pkg/tools/builtins.go`
- Create: `pkg/tools/builtins_test.go`

Implement three tools:

**list_files:**
```
Input: { "path": "/some/dir", "pattern": "*.pdf" (optional), "recursive": false (optional) }
Output: One filename per line, with size and modification date
```

**read_file:**
```
Input: { "path": "/some/file.txt" }
Output: File content as string (truncated at 100KB with notice)
```

**run_command:**
```
Input: { "command": "find ~/Downloads -name '*.pdf' | wc -l", "workdir": "" (optional) }
Output: Command stdout+stderr (truncated at 50KB, timeout 30s)
```

Safety: All tools validate paths are under allowed directories (home dir tree). `run_command` has a 30-second timeout and output size limit.

**Tests:**
- TestListFiles_BasicDirectory
- TestListFiles_WithPattern
- TestListFiles_Recursive
- TestReadFile_SmallFile
- TestReadFile_LargeFileTruncation
- TestReadFile_NonExistent
- TestRunCommand_SimpleCommand
- TestRunCommand_Timeout
- TestRunCommand_OutputTruncation

---

### Task 3: Anthropic Tool-Use API Support

**Files:**
- Modify: `pkg/llm/llm.go` (add ChatWithTools method)
- Modify: `types/types.go` (add tool-related types)
- Create: `pkg/llm/llm_test.go`

Add tool-use types to `types/types.go`:

```go
// ToolDefinition is the Anthropic API tool definition format.
type ToolDefinition struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    InputSchema map[string]any `json:"input_schema"`
}

// ContentBlock is a content block in an Anthropic response.
type ContentBlock struct {
    Type  string         `json:"type"`            // "text" or "tool_use"
    Text  string         `json:"text,omitempty"`
    ID    string         `json:"id,omitempty"`    // tool_use ID
    Name  string         `json:"name,omitempty"`  // tool name
    Input map[string]any `json:"input,omitempty"` // tool input
}

// ToolResult is a tool execution result sent back to the API.
type ToolResult struct {
    Type      string `json:"type"`        // "tool_result"
    ToolUseID string `json:"tool_use_id"`
    Content   string `json:"content"`
    IsError   bool   `json:"is_error,omitempty"`
}

// LLMToolRequest extends LLMRequest with tool definitions.
type LLMToolRequest struct {
    LLMRequest
    Tools []ToolDefinition `json:"tools,omitempty"`
}

// LLMToolResponse extends LLMResponse with content blocks and stop reason.
type LLMToolResponse struct {
    Content      []ContentBlock `json:"content"`
    Model        string         `json:"model"`
    StopReason   string         `json:"stopReason"` // "end_turn" or "tool_use"
    InputTokens  int            `json:"inputTokens"`
    OutputTokens int            `json:"outputTokens"`
}
```

Add `ChatWithTools` to the Claude provider that:
1. Sends tools in the request payload
2. Returns the full content blocks and stop_reason
3. Handles both `end_turn` and `tool_use` stop reasons

**Tests:**
- TestClaude_ChatWithTools_MockServer (use httptest to mock Anthropic API)

---

### Task 4: Tool Execution Loop in Bot Engine

**Files:**
- Modify: `pkg/bot/bot.go` (add tool loop to HandleMessage)
- Create: `pkg/bot/toolloop.go` (tool loop logic)
- Create: `pkg/bot/toolloop_test.go`

Add a `ToolProvider` interface to the LLM provider:

```go
type ToolProvider interface {
    ChatWithTools(ctx context.Context, req *types.LLMToolRequest) (*types.LLMToolResponse, error)
}
```

The tool loop in `HandleMessage`:
1. After building the prompt and history, check if the LLM provider supports tools
2. If it does, call `ChatWithTools` with tool definitions
3. If stop_reason is `tool_use`, execute each tool call via the registry
4. Append assistant message (with tool_use blocks) and user message (with tool_result blocks) to the conversation
5. Call `ChatWithTools` again with the updated conversation
6. Repeat until `stop_reason` is `end_turn` or max iterations (10) reached
7. Extract the final text from content blocks

The Bot struct gets a `tools *tools.Registry` field, initialized with the three built-in tools.

**Tests:**
- TestToolLoop_SingleToolCall
- TestToolLoop_MultipleToolCalls
- TestToolLoop_MaxIterations
- TestToolLoop_NoTools_FallsBackToChat
- TestBot_HandleMessage_WithTools (integration test)

---

### Task 5: CLI Subcommands for openbot

**Files:**
- Create: `cmd/openbot/cli.go` (CLI framework)
- Modify: `cmd/openbot/main.go` (add subcommand dispatch)

Add subcommands to openbot:

**`openbot` (no args)** — Run the bot (existing behavior)

**`openbot sessions`** — List active sessions from SQLite:
```
ID          Peer        Channel   Messages  Last Active
a1b2c3d4    @tamnd87    telegram  42        2m ago
```

**`openbot history [session-id]`** — Show messages for a session:
```
[14:12] user: How many PDFs in my Downloads?
[14:12] assistant: You've got 181 PDF files...
```
If no session-id given, uses most recent active session.

**`openbot send <message>`** — Send a message through the bot engine (processes through LLM + tools, prints response):
```
$ openbot send "List all PDFs in ~/Downloads"
Found 181 PDF files in ~/Downloads:
...
```

**`openbot status`** — Show bot status:
```
OpenBot Status:
  Config:    ~/.openbot/openbot.json
  Database:  ~/.openbot/data/bot.db
  Sessions:  3 active
  Messages:  127 total
  Memory:    42 chunks indexed
```

**Tests (in separate test file):**
- TestCLI_Sessions_ListsFromStore
- TestCLI_History_ShowsMessages
- TestCLI_Send_ProcessesMessage
- TestCLI_Status_ShowsStats

---

### Task 6: Integration Test — PDF Listing Scenario

**Files:**
- Create: `pkg/bot/scenario_test.go`

End-to-end test that simulates the PDF listing scenario:

1. Create a temp directory with fake PDF files
2. Configure the bot with a mock LLM that simulates tool use:
   - First call: returns `tool_use` for `list_files` with `{"path": "<tempdir>", "pattern": "*.pdf"}`
   - Second call: returns `end_turn` with formatted text listing the PDFs
3. Send a message "List all PDF files"
4. Verify the response contains the expected PDF filenames
5. Verify tool execution was logged

Also test the `run_command` tool with a simple command.

**Tests:**
- TestScenario_ListPDFs
- TestScenario_RunCommand_FindPDFs
- TestScenario_MultiToolConversation
