package command

import (
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Service handles in-chat slash commands.
type Service struct{}

// NewService creates a command service.
func NewService() *Service {
	return &Service{}
}

// Commands returns the list of available slash commands.
func (s *Service) Commands() []types.Command {
	return []types.Command{
		{Name: "/new", Description: "Start a new conversation session", Usage: "/new"},
		{Name: "/reset", Description: "Reset the current session", Usage: "/reset"},
		{Name: "/status", Description: "Show agent and session status", Usage: "/status"},
		{Name: "/model", Description: "Switch the AI model", Usage: "/model <model-name>"},
		{Name: "/help", Description: "Show available commands", Usage: "/help"},
		{Name: "/compact", Description: "Summarize older context to free space", Usage: "/compact"},
		{Name: "/context", Description: "Show enriched system prompt (workspace + skills + memory)", Usage: "/context"},
		{Name: "/memory", Description: "Search the agent's memory index", Usage: "/memory <query>"},
	}
}

// Parse checks if a message is a slash command and returns the command name and args.
func (s *Service) Parse(content string) (cmd string, args string, isCommand bool) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "/") {
		return "", "", false
	}

	parts := strings.SplitN(content, " ", 2)
	cmd = strings.ToLower(parts[0])
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}
	return cmd, args, true
}

// Execute handles a slash command and returns the response text.
func (s *Service) Execute(cmd, args string, agent *types.Agent) string {
	switch cmd {
	case "/help":
		var sb strings.Builder
		sb.WriteString("Available commands:\n\n")
		for _, c := range s.Commands() {
			sb.WriteString(fmt.Sprintf("  %s - %s\n", c.Usage, c.Description))
		}
		return sb.String()

	case "/status":
		return fmt.Sprintf("Agent: %s (%s)\nModel: %s\nStatus: %s\nMax Tokens: %d\nTemperature: %.1f",
			agent.Name, agent.ID, agent.Model, agent.Status, agent.MaxTokens, agent.Temperature)

	case "/model":
		if args == "" {
			return fmt.Sprintf("Current model: %s\nUsage: /model <model-name>", agent.Model)
		}
		return fmt.Sprintf("Model switched to: %s (takes effect on next message)", args)

	case "/context":
		if agent.SystemPrompt != "" {
			return fmt.Sprintf("System prompt:\n%s", agent.SystemPrompt)
		}
		return "No system prompt configured."

	case "/new":
		return "New session started. Previous context has been cleared."

	case "/reset":
		return "Session reset. Starting fresh."

	case "/compact":
		return "Context compacted. Older messages have been summarized."

	default:
		return fmt.Sprintf("Unknown command: %s\nType /help for available commands.", cmd)
	}
}
