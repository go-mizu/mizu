package gateway

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/feature/command"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/compact"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/memory"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Broadcaster sends real-time events to connected clients.
type Broadcaster interface {
	Broadcast(event string, payload any)
}

// ProcessMessageResult contains the full result of processing a message.
type ProcessMessageResult struct {
	SessionID string `json:"sessionId"`
	AgentID   string `json:"agentId"`
	Content   string `json:"content"`
	Model     string `json:"model"`
	MessageID string `json:"messageId"`
}

// DefaultContextWindow is the default context window size for Claude models.
const DefaultContextWindow = 200000

// Service is the core message routing engine.
// It receives inbound messages, resolves the target agent via bindings,
// manages sessions, invokes the LLM, and stores conversation history.
// It integrates workspace bootstrap, skills, memory search, and context
// pruning/compaction matching OpenClaw's behavior.
type Service struct {
	store       store.Store
	llm         llm.Provider
	commands    *command.Service
	memReg      *memoryRegistry
	ctxBuilder  *contextBuilder
	broadcaster Broadcaster
	startAt     time.Time

	mu       sync.Mutex
	inflight map[string]context.CancelFunc // session ID → cancel
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
		inflight:   make(map[string]context.CancelFunc),
	}
}

// SetBroadcaster sets the event broadcaster for real-time dashboard updates.
func (g *Service) SetBroadcaster(b Broadcaster) {
	g.broadcaster = b
}

// Abort cancels an in-flight LLM request for the given session.
// Returns true if a request was actually cancelled.
func (g *Service) Abort(sessionID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if cancel, ok := g.inflight[sessionID]; ok {
		cancel()
		delete(g.inflight, sessionID)
		return true
	}
	return false
}

// Close releases resources held by the gateway service.
func (g *Service) Close() {
	if g.memReg != nil {
		g.memReg.closeAll()
	}
}

// SearchMemory searches indexed memory for the given workspace directory.
// Returns formatted search results for the dashboard. If workspaceDir is empty,
// it tries to find the default agent's workspace.
func (g *Service) SearchMemory(ctx context.Context, workspaceDir, query string, limit int) ([]memory.SearchResult, error) {
	if g.memReg == nil {
		return nil, nil
	}
	mgr, err := g.memReg.get(workspaceDir)
	if err != nil || mgr == nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}
	return mgr.Search(ctx, query, limit, 0)
}

// MemoryStats returns the stats for a workspace's memory index.
func (g *Service) MemoryStats(workspaceDir string) (fileCount int, chunkCount int, err error) {
	if g.memReg == nil {
		return 0, 0, nil
	}
	mgr, err := g.memReg.get(workspaceDir)
	if err != nil || mgr == nil {
		return 0, 0, err
	}
	return mgr.Stats()
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
func (g *Service) ProcessMessage(ctx context.Context, msg *types.InboundMessage) (*ProcessMessageResult, error) {
	channelType := string(msg.ChannelType)
	channelID := msg.ChannelID

	// 1. Resolve agent
	agent, err := g.store.ResolveAgent(ctx, channelType, channelID, msg.PeerID)
	if err != nil {
		return nil, fmt.Errorf("resolve agent: %w", err)
	}

	// 2. Get or create session
	session, err := g.store.GetOrCreateSession(ctx, agent.ID, channelID, channelType, msg.PeerID, msg.PeerName, msg.Origin)
	if err != nil {
		return nil, fmt.Errorf("get/create session: %w", err)
	}

	// 3. Check for slash commands
	cmd, cmdArgs, isCommand := g.commands.Parse(msg.Content)
	if isCommand {
		// Load skills for skill command dispatch.
		if agent.Workspace != "" {
			earlySkills, _ := skill.LoadAllSkills(agent.Workspace, skill.BundledSkillsDir())
			g.commands.SetSkills(earlySkills)
		}

		// Check if it's a skill command first.
		if matchedSkill, ok := g.commands.IsSkillCommand(cmd); ok {
			// Skill commands flow through the LLM with skill content injected.
			// Rewrite the message content: use args as the user query,
			// and the skill content will be injected as additional context.
			msg.Content = cmdArgs
			if msg.Content == "" {
				msg.Content = matchedSkill.Description
			}
			msg.SkillContext = matchedSkill.Content
			msg.SkillName = matchedSkill.Name
		} else {
			response := g.handleCommand(ctx, cmd, cmdArgs, agent, session)
			return &ProcessMessageResult{
				SessionID: session.ID,
				AgentID:   agent.ID,
				Content:   response,
				Model:     agent.Model,
			}, nil
		}
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
		return nil, fmt.Errorf("store user message: %w", err)
	}

	// 4a. Broadcast user message
	if g.broadcaster != nil {
		g.broadcaster.Broadcast("chat.message", map[string]any{
			"sessionId": session.ID,
			"message": map[string]any{
				"id":        userMsg.ID,
				"role":      userMsg.Role,
				"content":   userMsg.Content,
				"createdAt": userMsg.CreatedAt.Format(time.RFC3339),
			},
		})
	}

	// 5. Build conversation history
	history, err := g.store.ListMessages(ctx, session.ID, 50)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
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
	// 6a. Load skills and build prompts.
	var skillsPrompt, alwaysPrompt string
	var loadedSkills []*skill.Skill
	if agent.Workspace != "" {
		skillsPrompt, loadedSkills = g.ctxBuilder.buildSkillsSection(agent.Workspace)
		if len(loadedSkills) > 0 {
			alwaysPrompt = skill.BuildAlwaysSkillsPrompt(loadedSkills)
			// Update command service with loaded skills for command dispatch.
			g.commands.SetSkills(loadedSkills)
		}
	}

	// 6a-ii. Apply skill env overrides (apiKey → primaryEnv, custom env vars).
	// Scoped to this request; cleanup restores original values.
	cleanupEnv := applySkillEnvOverrides(loadedSkills, agent.Workspace)
	defer cleanupEnv()

	// 6b. Build system prompt with all OpenClaw-compatible sections.
	promptParams := &SystemPromptParams{
		Agent:        agent,
		WorkspaceDir: agent.Workspace,
		Origin:       msg.Origin,
		Query:        msg.Content,
		Channel:      channelType,
		SkillsPrompt: skillsPrompt,
		AlwaysPrompt: alwaysPrompt,
		SessionID:    session.ID,
	}
	buildResult := g.ctxBuilder.buildSystemPrompt(ctx, promptParams)
	systemPrompt := buildResult.Prompt

	// 6b-ii. Inject skill command context if a skill was triggered via /command.
	if msg.SkillContext != "" {
		systemPrompt += fmt.Sprintf("\n\n<skill-context name=%q>\n%s\n</skill-context>", msg.SkillName, msg.SkillContext)
	}

	// 6c. Collect skill names for reporting.
	var skillNames []string
	for _, s := range loadedSkills {
		if s.Ready {
			skillNames = append(skillNames, s.Name)
		}
	}
	buildResult.SkillNames = skillNames

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

	// 8. Broadcast typing indicator
	if g.broadcaster != nil {
		g.broadcaster.Broadcast("chat.typing", map[string]any{
			"sessionId": session.ID,
			"agentId":   agent.ID,
		})
	}

	// 8a. Register cancellable context for abort support
	llmCtx, cancel := context.WithCancel(ctx)
	g.mu.Lock()
	g.inflight[session.ID] = cancel
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		delete(g.inflight, session.ID)
		g.mu.Unlock()
		cancel()
	}()

	// 8b. Call LLM
	llmReq := &types.LLMRequest{
		Model:        agent.Model,
		SystemPrompt: systemPrompt,
		Messages:     llmMessages,
		MaxTokens:    agent.MaxTokens,
		Temperature:  agent.Temperature,
	}

	llmResp, err := g.llm.Chat(llmCtx, llmReq)
	if err != nil {
		log.Printf("LLM error for agent %s: %v", agent.ID, err)
		if g.broadcaster != nil {
			g.broadcaster.Broadcast("chat.done", map[string]any{
				"sessionId": session.ID,
				"agentId":   agent.ID,
			})
		}
		return nil, fmt.Errorf("LLM chat: %w", err)
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
		return nil, fmt.Errorf("store assistant message: %w", err)
	}

	// 9a. Broadcast assistant message and done
	if g.broadcaster != nil {
		g.broadcaster.Broadcast("chat.message", map[string]any{
			"sessionId": session.ID,
			"message": map[string]any{
				"id":        assistantMsg.ID,
				"role":      assistantMsg.Role,
				"content":   assistantMsg.Content,
				"createdAt": assistantMsg.CreatedAt.Format(time.RFC3339),
			},
		})
		g.broadcaster.Broadcast("chat.done", map[string]any{
			"sessionId": session.ID,
			"agentId":   agent.ID,
		})
	}

	return &ProcessMessageResult{
		SessionID: session.ID,
		AgentID:   agent.ID,
		Content:   llmResp.Content,
		Model:     agent.Model,
		MessageID: assistantMsg.ID,
	}, nil
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
		params := &SystemPromptParams{
			Agent:        agent,
			WorkspaceDir: agent.Workspace,
			Origin:       session.Origin,
			Channel:      session.ChannelType,
			SessionID:    session.ID,
		}
		// Load skills for context display.
		if agent.Workspace != "" {
			sp, _ := g.ctxBuilder.buildSkillsSection(agent.Workspace)
			params.SkillsPrompt = sp
		}
		result := g.ctxBuilder.buildSystemPrompt(context.Background(), params)
		if result.Prompt == "" {
			return "No system prompt configured."
		}
		return fmt.Sprintf("System prompt:\n%s", result.Prompt)
	case "/memory":
		if args == "" {
			return "Usage: /memory <query>\nSearches the agent's memory index for relevant context."
		}
		result, err := g.MemorySearch(ctx, agent.Workspace, args)
		if err != nil {
			return fmt.Sprintf("Memory search error: %v", err)
		}
		return result
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

// ReIndex triggers a re-index of the memory for the given workspace directory.
// This is called after session compaction to ensure memory search stays current.
func (g *Service) ReIndex(workspaceDir string) error {
	if g.memReg == nil || workspaceDir == "" {
		return nil
	}
	mgr, err := g.memReg.get(workspaceDir)
	if err != nil || mgr == nil {
		return err
	}
	return mgr.ReIndex()
}

// Commands returns available slash commands.
func (g *Service) Commands() []types.Command {
	return g.commands.Commands()
}
