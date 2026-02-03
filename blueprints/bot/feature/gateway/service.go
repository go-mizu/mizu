package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/feature/command"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/compact"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/memory"
	filesession "github.com/go-mizu/mizu/blueprints/bot/pkg/session"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	bottools "github.com/go-mizu/mizu/blueprints/bot/pkg/tools"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Broadcaster sends real-time events to connected clients.
type Broadcaster interface {
	Broadcast(event string, payload any)
}

// ProcessMessageResult contains the full result of processing a message.
type ProcessMessageResult struct {
	SessionID  string      `json:"sessionId"`
	SessionKey string      `json:"sessionKey,omitempty"`
	AgentID    string      `json:"agentId"`
	Content    string      `json:"content"`
	Model      string      `json:"model"`
	MessageID  string      `json:"messageId"`
	RunID      string      `json:"runId,omitempty"`
	Usage      *TokenUsage `json:"usage,omitempty"`
}

// TokenUsage tracks LLM token consumption for a request.
type TokenUsage struct {
	Input       int `json:"input"`
	Output      int `json:"output"`
	TotalTokens int `json:"totalTokens"`
}

// DefaultContextWindow is the default context window size for Claude models.
const DefaultContextWindow = 200000

// dedupeTTL is the idempotency cache TTL matching OpenClaw's behavior.
const dedupeTTL = 5 * time.Minute

// chatAbortEntry tracks an in-flight chat run for abort support.
type chatAbortEntry struct {
	cancel     context.CancelFunc
	sessionID  string
	sessionKey string
	startedAt  time.Time
}

// dedupeEntry caches a chat.send response for idempotency.
type dedupeEntry struct {
	ts      time.Time
	payload any
	err     error
}

// ChannelTyper provides Send and typing actions for a channel driver.
type ChannelTyper interface {
	SendTypingAction(ctx context.Context, chatID string) error
}

// Service is the core message routing engine.
// It receives inbound messages, resolves the target agent via bindings,
// manages sessions, invokes the LLM, and stores conversation history.
// It integrates workspace bootstrap, skills, memory search, and context
// pruning/compaction matching OpenClaw's behavior.
type Service struct {
	store       store.Store
	llm         llm.Provider
	tools       *bottools.Registry
	commands    *command.Service
	memReg      *memoryRegistry
	ctxBuilder  *contextBuilder
	broadcaster Broadcaster
	startAt     time.Time

	// File-based session store (OpenClaw-compatible JSONL transcripts).
	fileStore *filesession.FileStore

	// Channel drivers for deliver routing and typing indicators.
	channelDrivers map[string]channel.Driver

	mu          sync.Mutex
	inflight    map[string]*chatAbortEntry // runId → entry
	dedupeCache map[string]*dedupeEntry    // idempotencyKey → cached response
}

// NewService creates a gateway service with memory and context management.
func NewService(s store.Store, provider llm.Provider) *Service {
	memReg := newMemoryRegistry()

	// Initialize tool registry with all built-in tools.
	toolRegistry := bottools.NewRegistry()
	bottools.RegisterBuiltins(toolRegistry)

	return &Service{
		store:          s,
		llm:            provider,
		tools:          toolRegistry,
		commands:       command.NewService(),
		memReg:         memReg,
		ctxBuilder:     newContextBuilder(memReg),
		startAt:        time.Now(),
		inflight:       make(map[string]*chatAbortEntry),
		dedupeCache:    make(map[string]*dedupeEntry),
		channelDrivers: make(map[string]channel.Driver),
	}
}

// SetBroadcaster sets the event broadcaster for real-time dashboard updates.
func (g *Service) SetBroadcaster(b Broadcaster) {
	g.broadcaster = b
}

// RegisterChannelDriver registers a channel driver for deliver routing.
func (g *Service) RegisterChannelDriver(name string, driver channel.Driver) {
	g.channelDrivers[name] = driver
}

// SetFileStore sets the OpenClaw-compatible file session store for JSONL transcripts.
func (g *Service) SetFileStore(fs *filesession.FileStore) {
	g.fileStore = fs
}

// Abort cancels an in-flight LLM request for the given session (legacy).
// Returns true if a request was actually cancelled.
func (g *Service) Abort(sessionID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for runID, entry := range g.inflight {
		if entry.sessionID == sessionID {
			entry.cancel()
			delete(g.inflight, runID)
			return true
		}
	}
	return false
}

// AbortByRunID aborts a specific run, verifying the sessionKey matches.
func (g *Service) AbortByRunID(runID, sessionKey string) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, ok := g.inflight[runID]
	if !ok {
		return false, nil
	}
	if sessionKey != "" && entry.sessionKey != sessionKey {
		return false, fmt.Errorf("runId does not match sessionKey")
	}
	entry.cancel()
	delete(g.inflight, runID)
	if g.broadcaster != nil {
		g.broadcaster.Broadcast("chat", map[string]any{
			"runId": runID, "sessionKey": entry.sessionKey,
			"seq": 1, "state": "aborted", "stopReason": "rpc",
		})
	}
	return true, nil
}

// AbortBySessionKey aborts all runs for a session key.
func (g *Service) AbortBySessionKey(sessionKey string) ([]string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var aborted []string
	for runID, entry := range g.inflight {
		if entry.sessionKey == sessionKey {
			entry.cancel()
			delete(g.inflight, runID)
			aborted = append(aborted, runID)
			if g.broadcaster != nil {
				g.broadcaster.Broadcast("chat", map[string]any{
					"runId": runID, "sessionKey": sessionKey,
					"seq": 1, "state": "aborted", "stopReason": "rpc",
				})
			}
		}
	}
	return aborted, len(aborted) > 0
}

// CheckDedupe checks the idempotency cache for a previous response.
// Returns the cached payload and true if found, nil and false otherwise.
func (g *Service) CheckDedupe(key string) (any, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cleanDedupe()
	entry, ok := g.dedupeCache[key]
	if !ok {
		return nil, false
	}
	return entry.payload, true
}

// SetDedupe stores a response in the idempotency cache.
func (g *Service) SetDedupe(key string, payload any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.dedupeCache[key] = &dedupeEntry{ts: time.Now(), payload: payload}
}

// cleanDedupe removes expired entries. Must be called with mu held.
func (g *Service) cleanDedupe() {
	cutoff := time.Now().Add(-dedupeTTL)
	for k, v := range g.dedupeCache {
		if v.ts.Before(cutoff) {
			delete(g.dedupeCache, k)
		}
	}
}

// SessionKeyToQuery resolves an OpenClaw session key to store query params.
func SessionKeyToQuery(key string) (agentID, channelType, peerID string) {
	parts := strings.Split(key, ":")
	if len(parts) < 3 || parts[0] != "agent" {
		return "main", "webhook", "dashboard-user"
	}
	agentID = parts[1]
	if len(parts) == 3 {
		return agentID, "webhook", "dashboard-user"
	}
	return agentID, parts[2], parts[3]
}

// BuildSessionKey constructs a session key from components.
func BuildSessionKey(agentID, channelType, peerID string) string {
	if channelType == "webhook" && peerID == "dashboard-user" {
		return "agent:" + agentID + ":main"
	}
	return "agent:" + agentID + ":" + channelType + ":" + peerID
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

// ProcessMessage handles an inbound message end-to-end.
// When msg.Async is true, returns immediately with {runId, status} and
// processes in a background goroutine with streaming delta events.
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

	sessionKey := BuildSessionKey(agent.ID, channelType, msg.PeerID)
	runID := msg.RunID
	if runID == "" {
		runID = fmt.Sprintf("run-%d", time.Now().UnixNano())
	}

	// 3. Check for slash commands
	cmd, cmdArgs, isCommand := g.commands.Parse(msg.Content)
	if isCommand {
		if agent.Workspace != "" {
			earlySkills, _ := skill.LoadAllSkills(agent.Workspace, skill.BundledSkillsDir())
			g.commands.SetSkills(earlySkills)
		}
		if matchedSkill, ok := g.commands.IsSkillCommand(cmd); ok {
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

	// 4a. Append to JSONL transcript
	g.appendTranscript(sessionKey, session.ID, "user", msg.Content, nil)

	// 4b. Broadcast user message
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

	// For async mode, dispatch to goroutine and return immediately.
	if msg.Async {
		go g.processLLM(context.Background(), msg, agent, session, sessionKey, runID)
		return &ProcessMessageResult{
			SessionID:  session.ID,
			SessionKey: sessionKey,
			AgentID:    agent.ID,
			RunID:      runID,
		}, nil
	}

	// Synchronous processing (legacy / channel drivers).
	return g.processLLM(ctx, msg, agent, session, sessionKey, runID)
}

// processLLM runs the LLM pipeline: history, prompt, streaming call, store, broadcast.
func (g *Service) processLLM(ctx context.Context, msg *types.InboundMessage, agent *types.Agent, session *types.Session, sessionKey, runID string) (*ProcessMessageResult, error) {
	channelType := string(msg.ChannelType)
	channelID := msg.ChannelID

	// Apply timeout.
	timeout := 120 * time.Second
	if msg.TimeoutMs > 0 {
		timeout = time.Duration(msg.TimeoutMs) * time.Millisecond
	}
	ctx, timeoutCancel := context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	// 5. Build conversation history
	history, err := g.store.ListMessages(ctx, session.ID, 50)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	llmMessages := make([]types.LLMMsg, len(history))
	for i, m := range history {
		llmMessages[i] = types.LLMMsg{Role: m.Role, Content: m.Content}
	}

	// 5a. Context pruning.
	contextWindow := DefaultContextWindow
	totalTokens := compact.EstimateMessagesTokens(llmMessages)
	llmMessages = compact.PruneMessages(llmMessages, totalTokens, contextWindow, compact.DefaultPruneConfig())

	totalTokens = compact.EstimateMessagesTokens(llmMessages)
	historyBudget := float64(contextWindow-compact.DefaultReserveTokensFloor) / float64(contextWindow)
	pruneResult := compact.PruneHistoryForContextShare(llmMessages, contextWindow, historyBudget)
	llmMessages = pruneResult.Messages

	// 6. Build enriched system prompt.
	var skillsPrompt, alwaysPrompt string
	var loadedSkills []*skill.Skill
	if agent.Workspace != "" {
		skillsPrompt, loadedSkills = g.ctxBuilder.buildSkillsSection(agent.Workspace)
		if len(loadedSkills) > 0 {
			alwaysPrompt = skill.BuildAlwaysSkillsPrompt(loadedSkills)
			g.commands.SetSkills(loadedSkills)
		}
	}

	cleanupEnv := applySkillEnvOverrides(loadedSkills, agent.Workspace)
	defer cleanupEnv()

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

	if msg.SkillContext != "" {
		systemPrompt += fmt.Sprintf("\n\n<skill-context name=%q>\n%s\n</skill-context>", msg.SkillName, msg.SkillContext)
	}

	// 7. Memory flush check.
	totalTokens = compact.EstimateMessagesTokens(llmMessages)
	flushCfg := compact.DefaultFlushConfig()
	if compact.ShouldRunMemoryFlush(totalTokens, contextWindow, flushCfg) {
		llmMessages = append(llmMessages, types.LLMMsg{
			Role:    types.RoleUser,
			Content: compact.BuildFlushPrompt(flushCfg),
		})
	}

	// 8. Broadcast typing + start typing indicator for Telegram.
	if g.broadcaster != nil {
		g.broadcaster.Broadcast("chat.typing", map[string]any{
			"sessionId": session.ID,
			"agentId":   agent.ID,
		})
	}
	typingCancel := g.startTypingIndicator(ctx, msg)
	defer typingCancel()

	// 8a. Register abort context.
	llmCtx, cancel := context.WithCancel(ctx)
	g.mu.Lock()
	g.inflight[runID] = &chatAbortEntry{
		cancel: cancel, sessionID: session.ID,
		sessionKey: sessionKey, startedAt: time.Now(),
	}
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		delete(g.inflight, runID)
		g.mu.Unlock()
		cancel()
	}()

	// Resolve thinking level: message > session > default.
	thinkingLevel := msg.ThinkingLevel

	// 8b. Call LLM with streaming if available.
	var responseText string
	var contentBlocks []types.ContentBlock
	var inputTokens, outputTokens int
	var stopReason string

	// Create delta accumulator for streaming.
	delta := NewDeltaAccumulator(runID, sessionKey, g.broadcaster)

	toolProvider, hasTools := g.llm.(llm.ToolProvider)
	if hasTools && g.tools != nil && len(g.tools.All()) > 0 {
		msgs := make([]any, len(llmMessages))
		for i, m := range llmMessages {
			msgs[i] = map[string]any{"role": m.Role, "content": m.Content}
		}

		toolDefs := make([]types.ToolDefinition, 0, len(g.tools.All()))
		for _, t := range g.tools.All() {
			toolDefs = append(toolDefs, types.ToolDefinition{
				Name: t.Name, Description: t.Description, InputSchema: t.InputSchema,
			})
		}

		toolReq := &types.LLMToolRequest{
			Model: agent.Model, SystemPrompt: systemPrompt,
			Messages: msgs, MaxTokens: agent.MaxTokens,
			Temperature: agent.Temperature, Tools: toolDefs,
			ThinkingLevel: thinkingLevel,
		}

		// Try streaming tool loop first.
		var toolResp *types.LLMToolResponse
		if streamProvider, ok := g.llm.(llm.ToolStreamProvider); ok {
			delta.Start()
			streamCB := func(event *llm.StreamEvent) error {
				if event.Delta != nil {
					if event.Delta.Text != "" {
						delta.OnTextDelta(event.Delta.Text)
					}
					if event.Delta.Thinking != "" {
						delta.OnThinkingDelta(event.Delta.Thinking)
					}
				}
				return nil
			}
			toolResp, err = bottools.RunToolLoopStream(llmCtx, streamProvider, g.tools, toolReq, streamCB, g.broadcaster, runID, sessionKey)
			delta.Stop()
		} else {
			toolResp, err = bottools.RunToolLoop(llmCtx, toolProvider, g.tools, toolReq)
		}

		if err != nil {
			log.Printf("LLM tool loop error for agent %s: %v", agent.ID, err)
			delta.EmitError(err)
			if g.broadcaster != nil {
				g.broadcaster.Broadcast("chat.done", map[string]any{
					"sessionId": session.ID, "agentId": agent.ID,
				})
			}
			return nil, fmt.Errorf("LLM tool loop: %w", err)
		}
		responseText = toolResp.TextContent()
		contentBlocks = toolResp.Content
		inputTokens = toolResp.InputTokens
		outputTokens = toolResp.OutputTokens
		stopReason = toolResp.StopReason
	} else {
		// No tools -- try streaming simple chat.
		llmReq := &types.LLMRequest{
			Model: agent.Model, SystemPrompt: systemPrompt,
			Messages: llmMessages, MaxTokens: agent.MaxTokens,
			Temperature: agent.Temperature,
		}

		if streamProvider, ok := g.llm.(llm.StreamProvider); ok {
			delta.Start()
			streamCB := func(event *llm.StreamEvent) error {
				if event.Delta != nil && event.Delta.Text != "" {
					delta.OnTextDelta(event.Delta.Text)
				}
				return nil
			}
			llmResp, llmErr := streamProvider.ChatStream(llmCtx, llmReq, streamCB)
			delta.Stop()
			if llmErr != nil {
				log.Printf("LLM stream error for agent %s: %v", agent.ID, llmErr)
				delta.EmitError(llmErr)
				return nil, fmt.Errorf("LLM stream: %w", llmErr)
			}
			responseText = llmResp.Content
			inputTokens = llmResp.InputTokens
			outputTokens = llmResp.OutputTokens
		} else {
			llmResp, llmErr := g.llm.Chat(llmCtx, llmReq)
			if llmErr != nil {
				log.Printf("LLM error for agent %s: %v", agent.ID, llmErr)
				delta.EmitError(llmErr)
				return nil, fmt.Errorf("LLM chat: %w", llmErr)
			}
			responseText = llmResp.Content
			inputTokens = llmResp.InputTokens
			outputTokens = llmResp.OutputTokens
		}
		contentBlocks = []types.ContentBlock{{Type: "text", Text: responseText}}
	}

	if stopReason == "" {
		stopReason = "end_turn"
	}

	// 9. Store assistant response with metadata.
	usage := &TokenUsage{
		Input: inputTokens, Output: outputTokens,
		TotalTokens: inputTokens + outputTokens,
	}
	metadata := map[string]any{
		"stopReason": stopReason,
		"usage":      usage,
		"timestamp":  time.Now().UnixMilli(),
		"model":      agent.Model,
	}
	metadataJSON, _ := json.Marshal(metadata)

	assistantMsg := &types.Message{
		SessionID: session.ID,
		AgentID:   agent.ID,
		ChannelID: channelID,
		PeerID:    msg.PeerID,
		Role:      types.RoleAssistant,
		Content:   responseText,
		Metadata:  string(metadataJSON),
	}
	if err := g.store.CreateMessage(ctx, assistantMsg); err != nil {
		log.Printf("Error storing assistant message: %v", err)
	}

	// 9a. Append to JSONL transcript.
	g.appendTranscript(sessionKey, session.ID, "assistant", contentBlocks, &filesession.TokenUsage{
		Input: inputTokens, Output: outputTokens,
	})

	// 9b. Update file session token usage.
	if g.fileStore != nil {
		_ = g.fileStore.UpdateTokenUsage(sessionKey, inputTokens, outputTokens)
	}

	// 9c. Broadcast OpenClaw-format final event.
	wireContent := contentBlocksToWire(contentBlocks)
	if g.broadcaster != nil {
		delta.EmitFinal(contentBlocks, stopReason, usage)
		// Legacy events
		g.broadcaster.Broadcast("chat.message", map[string]any{
			"sessionId": session.ID,
			"message": map[string]any{
				"id": assistantMsg.ID, "role": assistantMsg.Role,
				"content":   responseText,
				"createdAt": assistantMsg.CreatedAt.Format(time.RFC3339),
			},
		})
		g.broadcaster.Broadcast("chat.done", map[string]any{
			"sessionId": session.ID, "agentId": agent.ID,
		})
	}
	_ = wireContent // used by EmitFinal internally

	// 10. Deliver to channel if requested.
	if msg.Deliver && session.ChannelType != "" && session.ChannelType != string(types.ChannelWebhook) {
		g.deliverToChannel(ctx, session, responseText)
	}

	return &ProcessMessageResult{
		SessionID: session.ID, SessionKey: sessionKey,
		AgentID: agent.ID, Content: responseText,
		Model: agent.Model, MessageID: assistantMsg.ID,
		RunID: runID, Usage: usage,
	}, nil
}

// appendTranscript writes a message to the JSONL transcript file.
func (g *Service) appendTranscript(sessionKey, sessionID, role string, content any, usage *filesession.TokenUsage) {
	if g.fileStore == nil {
		return
	}
	// Ensure session exists in file store.
	_, _, _ = g.fileStore.GetOrCreate(sessionKey, "", "", "")

	entry := &filesession.TranscriptEntry{
		Type: "message",
		ID:   fmt.Sprintf("%d", time.Now().UnixNano()),
		Message: &filesession.TranscriptMessage{
			Role:    role,
			Content: content,
		},
		Usage: usage,
	}
	_ = g.fileStore.AppendTranscript(sessionID, entry)
}

// startTypingIndicator sends periodic typing actions for Telegram channels.
// Returns a cancel function to stop the indicator.
func (g *Service) startTypingIndicator(ctx context.Context, msg *types.InboundMessage) context.CancelFunc {
	if msg.ChannelType != types.ChannelTelegram {
		return func() {}
	}
	driver, ok := g.channelDrivers["telegram"]
	if !ok {
		return func() {}
	}
	typer, ok := driver.(ChannelTyper)
	if !ok {
		return func() {}
	}

	typingCtx, cancel := context.WithCancel(ctx)
	chatID := msg.PeerID
	if msg.GroupID != "" {
		chatID = msg.GroupID
	}

	// Send initial typing action.
	go typer.SendTypingAction(typingCtx, chatID)

	// Repeat every 5 seconds.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				typer.SendTypingAction(typingCtx, chatID)
			}
		}
	}()

	return cancel
}

// deliverToChannel routes a response to the session's originating channel.
func (g *Service) deliverToChannel(ctx context.Context, session *types.Session, content string) {
	driver, ok := g.channelDrivers[session.ChannelType]
	if !ok {
		log.Printf("Deliver: no driver for channel type %s", session.ChannelType)
		return
	}
	outMsg := &types.OutboundMessage{
		ChannelType: types.ChannelType(session.ChannelType),
		ChannelID:   session.ChannelID,
		PeerID:      session.PeerID,
		Content:     content,
		ParseMode:   "markdown",
	}
	if err := driver.Send(ctx, outMsg); err != nil {
		log.Printf("Deliver error to %s/%s: %v", session.ChannelType, session.PeerID, err)
	}
}

// contentBlocksToWire converts ContentBlock slice to wire format for events.
func contentBlocksToWire(blocks []types.ContentBlock) []any {
	wire := make([]any, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case "thinking":
			wire = append(wire, map[string]any{"type": "thinking", "thinking": b.Thinking})
		case "tool_use":
			wire = append(wire, map[string]any{
				"type": "tool_use", "id": b.ID, "name": b.Name, "input": b.Input,
			})
		default:
			wire = append(wire, map[string]any{"type": "text", "text": b.Text})
		}
	}
	return wire
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
