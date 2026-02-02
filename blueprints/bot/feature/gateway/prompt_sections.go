package gateway

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// SystemPromptParams contains all inputs for building a system prompt.
// Mirrors OpenClaw's SystemPromptParams interface for 100% compatibility.
type SystemPromptParams struct {
	Agent           *types.Agent
	WorkspaceDir    string
	Origin          string   // "dm", "group", "subagent", "cron", "webhook"
	Query           string   // current user message (for memory search)
	Channel         string   // "telegram", "discord", etc.
	Capabilities    []string // channel capabilities
	OwnerNumbers    []string // phone numbers for user identity
	UserTimezone    string   // e.g. "Asia/Saigon"
	ToolNames       []string // registered LLM tool names
	DocsPath        string   // path to local docs
	SkillsPrompt    string   // pre-built XML skills block
	AlwaysPrompt    string   // always-skills body content
	MemoryPrompt    string   // memory recall content
	HeartbeatPrompt string   // periodic task instructions
	GroupContext     string   // group-specific context
	ThinkingLevel   string   // reasoning level
	SessionID       string   // for report generation
}

// buildIdentitySection returns the identity paragraph.
// Uses the agent's system prompt as identity, or a default.
func buildIdentitySection(agent *types.Agent) string {
	if agent != nil && agent.SystemPrompt != "" {
		return agent.SystemPrompt
	}
	return `You are a personal assistant running inside OpenBot.
You help your human through messaging channels.
Be genuinely helpful, not performatively helpful.
Have opinions. Be resourceful before asking.`
}

// buildUserIdentitySection returns the user identity section with owner numbers.
func buildUserIdentitySection(params *SystemPromptParams) string {
	if len(params.OwnerNumbers) == 0 {
		return ""
	}
	nums := strings.Join(params.OwnerNumbers, ", ")
	return fmt.Sprintf("## User Identity\nOwner numbers: %s. Treat messages from these numbers as the user.", nums)
}

// buildMessagingSection returns messaging guidance when the message tool is available.
func buildMessagingSection(params *SystemPromptParams) string {
	return `## Messaging
- message: Send messages and channel actions

### message tool
Use the message tool to send messages to channels. When a message does not
require a response (greetings, acknowledgments), respond with ONLY: NO_REPLY

## Silent Replies
When NO_REPLY is appropriate, output exactly "NO_REPLY" with no other text.`
}

// hasMessageTool checks if the message tool is in the tool list.
func hasMessageTool(toolNames []string) bool {
	for _, t := range toolNames {
		if strings.EqualFold(t, "message") {
			return true
		}
	}
	return false
}

// buildSkillsGuidanceSection wraps the skills prompt with usage guidance.
func buildSkillsGuidanceSection(params *SystemPromptParams) string {
	readTool := "read"
	for _, t := range params.ToolNames {
		if strings.EqualFold(t, "Read") {
			readTool = "Read"
			break
		}
	}

	var b strings.Builder
	b.WriteString("## Skills\n\n")
	b.WriteString(fmt.Sprintf(
		"- If exactly one skill clearly applies: read its SKILL.md at <location> with `%s`, then follow it.\n", readTool))
	b.WriteString("- If multiple skills might apply, pick the most specific one.\n")
	b.WriteString("- If no skill applies, proceed without one.\n")
	b.WriteString(params.SkillsPrompt)

	// Inject always-skills content after the XML block.
	if params.AlwaysPrompt != "" {
		b.WriteString("\n\n")
		b.WriteString(params.AlwaysPrompt)
	}

	return b.String()
}

// buildToolAvailabilitySection lists available tools with descriptions.
func buildToolAvailabilitySection(toolNames []string) string {
	toolDescs := map[string]string{
		"read":             "Read file contents",
		"edit":             "Edit file contents",
		"write":            "Write file contents",
		"exec":             "Run shell commands",
		"message":          "Send messages and channel actions",
		"process":          "Process management (background sessions)",
		"memory_search":    "Search memory index",
		"memory_get":       "Get memory entries",
		"web_search":       "Search the web",
		"web_fetch":        "Fetch a URL",
		"sessions_list":    "List sessions",
		"sessions_history": "Get session history",
		"sessions_send":    "Send to a session",
		"sessions_spawn":   "Spawn a new session",
		"agents_list":      "List agents",
		"cron":             "Cron job management",
		"gateway":          "Gateway control",
		"tts":              "Text-to-speech",
		"image":            "Image generation",
		"canvas":           "Canvas/HTML rendering",
		"nodes":            "Node management",
		"browser":          "Browser control",
	}

	var b strings.Builder
	b.WriteString("Tool availability (filtered by policy):\n")
	for _, name := range toolNames {
		desc := toolDescs[strings.ToLower(name)]
		if desc == "" {
			desc = name
		}
		b.WriteString(fmt.Sprintf("- %s: %s\n", name, desc))
	}
	return b.String()
}

// buildCLIReferenceSection returns the CLI quick reference.
func buildCLIReferenceSection() string {
	return `## OpenBot CLI Quick Reference
Common commands:
  openbot status              System status
  openbot gateway restart     Restart gateway
  openbot config get <path>   Read config value
  openbot config set <path>   Write config value
  openbot skills list         List available skills
  openbot memory search <q>   Search memory
  openbot sessions            List sessions
  openbot cron list           List cron jobs
Do not invent commands — only use documented ones.`
}

// buildDateTimeSection returns the current date/time section.
func buildDateTimeSection(userTimezone string) string {
	now := time.Now()
	dateStr := now.Format("Monday, January 2, 2006 3:04 PM MST")

	if userTimezone != "" {
		if loc, err := time.LoadLocation(userTimezone); err == nil {
			dateStr = now.In(loc).Format("Monday, January 2, 2006 3:04 PM MST")
		}
		return fmt.Sprintf("## Current Date & Time\n%s\nUser timezone: %s", dateStr, userTimezone)
	}
	return fmt.Sprintf("## Current Date & Time\n%s", dateStr)
}

// buildRuntimeSection returns the runtime info line.
func buildRuntimeSection(params *SystemPromptParams) string {
	hostname, _ := os.Hostname()
	parts := []string{
		fmt.Sprintf("host=%s", hostname),
		fmt.Sprintf("os=%s (%s)", runtime.GOOS, runtime.GOARCH),
	}
	if params.Agent != nil {
		parts = append(parts, fmt.Sprintf("agent=%s", params.Agent.ID))
		if params.Agent.Model != "" {
			parts = append(parts, fmt.Sprintf("model=%s", params.Agent.Model))
		}
	}
	if params.Channel != "" {
		parts = append(parts, fmt.Sprintf("channel=%s", params.Channel))
	}
	if len(params.Capabilities) > 0 {
		parts = append(parts, fmt.Sprintf("capabilities=%s", strings.Join(params.Capabilities, ",")))
	}
	if params.ThinkingLevel != "" {
		parts = append(parts, fmt.Sprintf("thinking=%s", params.ThinkingLevel))
	}
	return fmt.Sprintf("## Runtime\n%s", strings.Join(parts, " "))
}
