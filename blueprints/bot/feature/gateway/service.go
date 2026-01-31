package gateway

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/feature/command"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/compact"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// DefaultContextWindow is the default context window size for Claude models.
const DefaultContextWindow = 200000

// Service is the core message routing engine.
// It receives inbound messages, resolves the target agent via bindings,
// manages sessions, invokes the LLM, and stores conversation history.
// It integrates workspace bootstrap, skills, memory search, and context
// pruning/compaction matching OpenClaw's behavior.
type Service struct {
	store      store.Store
	llm        llm.Provider
	commands   *command.Service
	memReg     *memoryRegistry
	ctxBuilder *contextBuilder
	startAt    time.Time
}

// NewService creates a gateway service with memory and context management.
func NewService(s store.Store, provider llm.Provider) *Service {
	memReg := newMemoryRegistry()
	return &Service{
		store:      s,
		llm:        provider,
		commands:   command.NewService(),
		memReg:     memReg,
		ctxBuilder: newContextBuilder(memReg),
		startAt:    time.Now(),
	}
}

// Close releases resources held by the gateway service.
func (g *Service) Close() {
	if g.memReg != nil {
		g.memReg.closeAll()
	}
}

// ProcessMessage handles an inbound message end-to-end:
//  1. Resolve agent via bindings
//  2. Get or create session
//  3. Check for slash commands
//  4. Store user message
//  5. Build conversation history with pruning
//  6. Build enriched system prompt (workspace + skills + memory)
//  7. Check for memory flush trigger
//  8. Call LLM
//  9. Store assistant response
//  10. Return response
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

	// 5a. Apply context pruning to keep conversation within budget.
	contextWindow := DefaultContextWindow
	totalTokens := compact.EstimateMessagesTokens(llmMessages)

	llmMessages = compact.PruneMessages(
		llmMessages,
		totalTokens,
		contextWindow,
		compact.DefaultPruneConfig(),
	)

	// 5b. If still over budget, drop oldest messages.
	totalTokens = compact.EstimateMessagesTokens(llmMessages)
	historyBudget := float64(contextWindow-compact.DefaultReserveTokensFloor) / float64(contextWindow)
	pruneResult := compact.PruneHistoryForContextShare(llmMessages, contextWindow, historyBudget)
	llmMessages = pruneResult.Messages

	// 6. Build enriched system prompt (workspace + skills + memory search).
	systemPrompt := g.ctxBuilder.buildSystemPrompt(ctx, agent, msg.Origin, msg.Content)

	// 7. Check for memory flush trigger.
	totalTokens = compact.EstimateMessagesTokens(llmMessages)
	flushCfg := compact.DefaultFlushConfig()
	if compact.ShouldRunMemoryFlush(totalTokens, contextWindow, flushCfg) {
		flushPrompt := compact.BuildFlushPrompt(flushCfg)
		llmMessages = append(llmMessages, types.LLMMsg{
			Role:    types.RoleUser,
			Content: flushPrompt,
		})
	}

	// 8. Call LLM
	llmReq := &types.LLMRequest{
		Model:        agent.Model,
		SystemPrompt: systemPrompt,
		Messages:     llmMessages,
		MaxTokens:    agent.MaxTokens,
		Temperature:  agent.Temperature,
	}

	llmResp, err := g.llm.Chat(ctx, llmReq)
	if err != nil {
		log.Printf("LLM error for agent %s: %v", agent.ID, err)
		return "", fmt.Errorf("LLM chat: %w", err)
	}

	// 9. Store assistant response
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
		// Expire current session so a new one is created next message.
		session.Status = "expired"
		g.store.UpdateSession(ctx, session)
		return g.commands.Execute(cmd, args, agent)
	case "/context":
		// Show enriched system prompt.
		prompt := g.ctxBuilder.buildSystemPrompt(context.Background(), agent, session.Origin, "")
		if prompt == "" {
			return "No system prompt configured."
		}
		return fmt.Sprintf("System prompt:\n%s", prompt)
	default:
		return g.commands.Execute(cmd, args, agent)
	}
}

// MemorySearch exposes memory search for the /memory command.
func (g *Service) MemorySearch(ctx context.Context, workspaceDir, query string) (string, error) {
	if g.memReg == nil || workspaceDir == "" || query == "" {
		return "No memory index available.", nil
	}

	mgr, err := g.memReg.get(workspaceDir)
	if err != nil || mgr == nil {
		return "No memory index available.", nil
	}

	results, err := mgr.Search(ctx, query, 0, 0)
	if err != nil {
		return "", fmt.Errorf("memory search: %w", err)
	}
	if len(results) == 0 {
		return "No relevant results found.", nil
	}

	return formatMemoryResults(results), nil
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
