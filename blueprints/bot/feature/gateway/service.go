package gateway

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/feature/command"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Service is the core message routing engine.
// It receives inbound messages, resolves the target agent via bindings,
// manages sessions, invokes the LLM, and stores conversation history.
type Service struct {
	store    store.Store
	llm      llm.Provider
	commands *command.Service
	startAt  time.Time
}

// NewService creates a gateway service.
func NewService(s store.Store, provider llm.Provider) *Service {
	return &Service{
		store:    s,
		llm:      provider,
		commands: command.NewService(),
		startAt:  time.Now(),
	}
}

// ProcessMessage handles an inbound message end-to-end:
// 1. Resolve agent via bindings
// 2. Get or create session
// 3. Check for slash commands
// 4. Store user message
// 5. Build conversation history
// 6. Call LLM
// 7. Store assistant response
// 8. Return response
func (g *Service) ProcessMessage(ctx context.Context, msg *types.InboundMessage) (string, error) {
	channelType := string(msg.ChannelType)
	channelID := msg.ChannelID

	// 1. Resolve agent
	agent, err := g.store.ResolveAgent(ctx, channelType, channelID, msg.PeerID)
	if err != nil {
		return "", fmt.Errorf("resolve agent: %w", err)
	}

	// 2. Get or create session
	session, err := g.store.GetOrCreateSession(ctx, agent.ID, channelID, channelType, msg.PeerID, msg.PeerName, msg.Origin)
	if err != nil {
		return "", fmt.Errorf("get/create session: %w", err)
	}

	// 3. Check for slash commands
	cmd, args, isCommand := g.commands.Parse(msg.Content)
	if isCommand {
		response := g.handleCommand(ctx, cmd, args, agent, session)
		return response, nil
	}

	// 4. Store user message
	userMsg := &types.Message{
		SessionID: session.ID,
		AgentID:   agent.ID,
		ChannelID: channelID,
		PeerID:    msg.PeerID,
		Role:      types.RoleUser,
		Content:   msg.Content,
	}
	if err := g.store.CreateMessage(ctx, userMsg); err != nil {
		return "", fmt.Errorf("store user message: %w", err)
	}

	// 5. Build conversation history
	history, err := g.store.ListMessages(ctx, session.ID, 50)
	if err != nil {
		return "", fmt.Errorf("list messages: %w", err)
	}

	llmMessages := make([]types.LLMMsg, len(history))
	for i, m := range history {
		llmMessages[i] = types.LLMMsg{Role: m.Role, Content: m.Content}
	}

	// 6. Call LLM
	llmReq := &types.LLMRequest{
		Model:        agent.Model,
		SystemPrompt: agent.SystemPrompt,
		Messages:     llmMessages,
		MaxTokens:    agent.MaxTokens,
		Temperature:  agent.Temperature,
	}

	llmResp, err := g.llm.Chat(ctx, llmReq)
	if err != nil {
		log.Printf("LLM error for agent %s: %v", agent.ID, err)
		return "", fmt.Errorf("LLM chat: %w", err)
	}

	// 7. Store assistant response
	assistantMsg := &types.Message{
		SessionID: session.ID,
		AgentID:   agent.ID,
		ChannelID: channelID,
		PeerID:    msg.PeerID,
		Role:      types.RoleAssistant,
		Content:   llmResp.Content,
	}
	if err := g.store.CreateMessage(ctx, assistantMsg); err != nil {
		return "", fmt.Errorf("store assistant message: %w", err)
	}

	return llmResp.Content, nil
}

func (g *Service) handleCommand(ctx context.Context, cmd, args string, agent *types.Agent, session *types.Session) string {
	switch cmd {
	case "/new", "/reset":
		// Expire current session so a new one is created next message
		session.Status = "expired"
		g.store.UpdateSession(ctx, session)
		return g.commands.Execute(cmd, args, agent)
	default:
		return g.commands.Execute(cmd, args, agent)
	}
}

// Status returns the current gateway status.
func (g *Service) Status(ctx context.Context, port int) (*types.GatewayStatus, error) {
	stats, err := g.store.Stats(ctx)
	if err != nil {
		return nil, err
	}

	channels, err := g.store.ListChannels(ctx)
	if err != nil {
		return nil, err
	}

	channelNames := make([]string, len(channels))
	for i, c := range channels {
		channelNames[i] = fmt.Sprintf("%s (%s: %s)", c.Name, c.Type, c.Status)
	}

	uptime := time.Since(g.startAt).Truncate(time.Second).String()

	return &types.GatewayStatus{
		Status:       "running",
		Port:         port,
		Uptime:       uptime,
		ActiveAgents: stats.Agents,
		Channels:     channelNames,
		Sessions:     stats.Sessions,
		Messages:     stats.Messages,
	}, nil
}

// Commands returns available slash commands.
func (g *Service) Commands() []types.Command {
	return g.commands.Commands()
}
