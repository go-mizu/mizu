package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/feature/command"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/compact"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/memory"
	filesession "github.com/go-mizu/mizu/blueprints/bot/pkg/session"
	bottools "github.com/go-mizu/mizu/blueprints/bot/pkg/tools"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Bot is the core engine that manages sessions, messages, and LLM interactions.
// It wires together the store, LLM provider, command service, prompt builder,
// and memory manager to process inbound messages end-to-end.
type Bot struct {
	cfg       *config.Config
	store     store.Store
	llm       llm.Provider
	commands  *command.Service
	prompt    *PromptBuilder
	memory    *memory.MemoryManager
	tools     *bottools.Registry
	allowSet  map[string]bool
	fileStore *filesession.FileStore // OpenClaw-compatible file session store
}

// New creates a new Bot, opening the SQLite database and initialising all
// subsystems. It creates a default agent and wildcard binding if none exist.
func New(cfg *config.Config, provider llm.Provider) (*Bot, error) {
	// Ensure data directory exists.
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Open SQLite store.
	dbPath := filepath.Join(cfg.DataDir, "bot.db")
	s, err := sqlite.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	// Ensure schema.
	ctx := context.Background()
	if err := s.Ensure(ctx); err != nil {
		s.Close()
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Build allowlist set from config.
	allowSet := make(map[string]bool, len(cfg.Telegram.AllowFrom))
	for _, u := range cfg.Telegram.AllowFrom {
		allowSet[u] = true
	}

	b := &Bot{
		cfg:      cfg,
		store:    s,
		llm:      provider,
		commands: command.NewService(),
		prompt:   NewPromptBuilder(cfg.Workspace),
		allowSet: allowSet,
	}

	b.tools = bottools.NewRegistry()
	bottools.RegisterBuiltins(b.tools)

	// Create default agent and binding.
	if err := b.ensureDefaultAgent(ctx); err != nil {
		s.Close()
		return nil, fmt.Errorf("ensure default agent: %w", err)
	}

	// Initialise memory manager (non-fatal on failure).
	if cfg.Workspace != "" {
		memDBPath := filepath.Join(cfg.DataDir, "memory.db")
		memCfg := memory.DefaultMemoryConfig()
		memCfg.WorkspaceDir = cfg.Workspace
		mgr, err := memory.NewMemoryManager(memDBPath, cfg.Workspace, memCfg)
		if err == nil {
			b.memory = mgr
			// Index workspace files on startup.
			_ = mgr.IndexAll()
		}
		// If memory init fails, just proceed without it.
	}

	// Initialize file-based session store (OpenClaw-compatible).
	sessDir := filepath.Join(cfg.DataDir, "agents", "default", "sessions")
	fs, err := filesession.NewFileStore(sessDir)
	if err != nil {
		// Non-fatal: log and continue without file store.
		log.Printf("File session store init: %v", err)
	} else {
		b.fileStore = fs
	}

	return b, nil
}

// Close releases all resources held by the Bot.
func (b *Bot) Close() {
	if b.memory != nil {
		b.memory.Close()
	}
	if b.store != nil {
		b.store.Close()
	}
}

// FileStore returns the file-based session store (may be nil).
func (b *Bot) FileStore() *filesession.FileStore {
	return b.fileStore
}

// HandleMessage processes an inbound message through the full pipeline:
// policy check -> resolve agent -> get/create session -> check commands ->
// store message -> build history -> prune -> build prompt -> memory search ->
// call LLM -> store response -> return.
func (b *Bot) HandleMessage(ctx context.Context, msg *types.InboundMessage) (string, error) {
	// 1. Check DM allowlist policy.
	if err := b.checkPolicy(msg); err != nil {
		return "", err
	}

	// 2. Resolve agent via bindings.
	agent, err := b.store.ResolveAgent(ctx, string(msg.ChannelType), msg.ChannelID, msg.PeerID)
	if err != nil {
		return "", fmt.Errorf("resolve agent: %w", err)
	}

	// 3. Get or create session.
	session, err := b.store.GetOrCreateSession(
		ctx,
		agent.ID,
		msg.ChannelID,
		string(msg.ChannelType),
		msg.PeerID,
		msg.PeerName,
		msg.Origin,
	)
	if err != nil {
		return "", fmt.Errorf("get session: %w", err)
	}

	// 3b. Sync to file-based session store.
	var fsKey string
	if b.fileStore != nil {
		chatType := "direct"
		if msg.Origin == "group" {
			chatType = "group"
		}
		fsKey = filesession.SessionKey(agent.ID, string(msg.ChannelType), msg.PeerID, msg.GroupID)
		fsEntry, isNew, fsErr := b.fileStore.GetOrCreate(fsKey, msg.PeerName, chatType, string(msg.ChannelType))
		if fsErr != nil {
			log.Printf("File store sync: %v", fsErr)
		} else if isNew {
			// Write model info on new session.
			fsEntry.Model = agent.Model
			fsEntry.ModelProvider = "anthropic"
			fsEntry.ContextTokens = 200000
			b.fileStore.UpdateEntry(fsKey, fsEntry)
		}
	}

	// 4. Check for slash commands.
	cmd, args, isCommand := b.commands.Parse(msg.Content)
	if isCommand {
		return b.handleCommand(ctx, cmd, args, agent, session), nil
	}

	// 5. Store the user message.
	userMsg := &types.Message{
		SessionID: session.ID,
		AgentID:   agent.ID,
		ChannelID: msg.ChannelID,
		PeerID:    msg.PeerID,
		Role:      types.RoleUser,
		Content:   msg.Content,
	}
	if err := b.store.CreateMessage(ctx, userMsg); err != nil {
		return "", fmt.Errorf("store user message: %w", err)
	}

	// 5b. Append user message to JSONL transcript.
	if b.fileStore != nil {
		fsEntry, _, _ := b.fileStore.GetOrCreate(fsKey, msg.PeerName, "", string(msg.ChannelType))
		if fsEntry != nil {
			b.fileStore.AppendTranscript(fsEntry.SessionID, &filesession.TranscriptEntry{
				Type:      "message",
				ID:        userMsg.ID,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Message: &filesession.TranscriptMessage{
					Role:    "user",
					Content: msg.Content,
				},
			})
		}
	}

	// 6. Build conversation history from stored messages.
	history, err := b.store.ListMessages(ctx, session.ID, 50)
	if err != nil {
		return "", fmt.Errorf("list messages: %w", err)
	}

	llmMsgs := make([]types.LLMMsg, 0, len(history))
	for _, m := range history {
		if m.Role == types.RoleSystem {
			continue
		}
		llmMsgs = append(llmMsgs, types.LLMMsg{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// 7. Apply context pruning.
	contextWindow := 200000 // Claude default context window
	pruneCfg := compact.DefaultPruneConfig()
	totalTokens := compact.EstimateMessagesTokens(llmMsgs)
	llmMsgs = compact.PruneMessages(llmMsgs, totalTokens, contextWindow, pruneCfg)

	// Also prune history share to leave room for system prompt + response.
	pruneResult := compact.PruneHistoryForContextShare(llmMsgs, contextWindow, 0.7)
	llmMsgs = pruneResult.Messages

	// 8. Build system prompt with workspace + memory.
	systemPrompt := b.prompt.Build(msg.Origin, msg.Content)

	// 9. Search memory for relevant context.
	if b.memory != nil && msg.Content != "" {
		results, err := b.memory.Search(ctx, msg.Content, 6, 0)
		if err == nil && len(results) > 0 {
			memSection := formatMemoryResults(results)
			systemPrompt = systemPrompt + "\n\n" + memSection
		}
	}

	// 10. Call the LLM provider (with tool loop if supported).
	var responseText string

	toolProvider, hasTools := b.llm.(llm.ToolProvider)
	if hasTools && len(b.tools.All()) > 0 {
		// Convert history to []any for the tool request.
		msgs := make([]any, len(llmMsgs))
		for i, m := range llmMsgs {
			msgs[i] = map[string]any{
				"role":    m.Role,
				"content": m.Content,
			}
		}

		// Build tool definitions.
		toolDefs := make([]types.ToolDefinition, 0, len(b.tools.All()))
		for _, t := range b.tools.All() {
			toolDefs = append(toolDefs, types.ToolDefinition{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		}

		toolReq := &types.LLMToolRequest{
			Model:        agent.Model,
			SystemPrompt: systemPrompt,
			Messages:     msgs,
			MaxTokens:    agent.MaxTokens,
			Temperature:  agent.Temperature,
			Tools:        toolDefs,
		}

		toolResp, err := runToolLoop(ctx, toolProvider, b.tools, toolReq)
		if err != nil {
			return "", fmt.Errorf("tool loop: %w", err)
		}
		responseText = toolResp.TextContent()
	} else {
		// Fallback to simple chat (no tools).
		llmReq := &types.LLMRequest{
			Model:        agent.Model,
			SystemPrompt: systemPrompt,
			Messages:     llmMsgs,
			MaxTokens:    agent.MaxTokens,
			Temperature:  agent.Temperature,
		}
		llmResp, err := b.llm.Chat(ctx, llmReq)
		if err != nil {
			return "", fmt.Errorf("llm chat: %w", err)
		}
		responseText = llmResp.Content
	}

	// 11. Store the assistant response.
	assistantMsg := &types.Message{
		SessionID: session.ID,
		AgentID:   agent.ID,
		ChannelID: msg.ChannelID,
		PeerID:    msg.PeerID,
		Role:      types.RoleAssistant,
		Content:   responseText,
	}
	if err := b.store.CreateMessage(ctx, assistantMsg); err != nil {
		return "", fmt.Errorf("store assistant message: %w", err)
	}

	// 11b. Append assistant response to JSONL transcript and update tokens.
	if b.fileStore != nil {
		fsEntry, _, _ := b.fileStore.GetOrCreate(fsKey, msg.PeerName, "", string(msg.ChannelType))
		if fsEntry != nil {
			b.fileStore.AppendTranscript(fsEntry.SessionID, &filesession.TranscriptEntry{
				Type:      "message",
				ID:        assistantMsg.ID,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Message: &filesession.TranscriptMessage{
					Role:    "assistant",
					Content: responseText,
				},
			})
		}
	}

	return responseText, nil
}

// checkPolicy enforces the DM allowlist policy. An empty allowlist means
// all users are allowed (matching OpenClaw behavior).
func (b *Bot) checkPolicy(msg *types.InboundMessage) error {
	// Only enforce for DM messages with allowlist policy.
	if msg.Origin != "dm" {
		return nil
	}
	if b.cfg.Telegram.DMPolicy != "allowlist" {
		return nil
	}

	// Empty allowlist means allow all.
	if len(b.allowSet) == 0 {
		return nil
	}

	if !b.allowSet[msg.PeerID] {
		return fmt.Errorf("user %s is not allowed to send DMs", msg.PeerID)
	}

	return nil
}

// handleCommand processes a slash command and returns the response text.
// Some commands (like /context and /memory) are handled specially by the bot.
func (b *Bot) handleCommand(ctx context.Context, cmd, args string, agent *types.Agent, session *types.Session) string {
	switch cmd {
	case "/context":
		// Build the enriched system prompt and return it.
		systemPrompt := b.prompt.Build(session.Origin, "")
		if systemPrompt != "" {
			return fmt.Sprintf("System prompt:\n%s", systemPrompt)
		}
		return "No system prompt configured."

	case "/new", "/reset":
		// Expire the current session.
		session.Status = "expired"
		b.store.UpdateSession(ctx, session)
		// Reset file-based session too.
		if b.fileStore != nil {
			key := filesession.SessionKey(agent.ID, session.ChannelType, session.PeerID, "")
			if _, err := b.fileStore.ResetSession(key); err != nil {
				log.Printf("File store reset: %v", err)
			}
		}
		return b.commands.Execute(cmd, args, agent)

	case "/memory":
		if b.memory == nil {
			return "No memory index available."
		}
		if args == "" {
			return "Usage: /memory <query>"
		}
		results, err := b.memory.Search(ctx, args, 6, 0)
		if err != nil {
			return fmt.Sprintf("Memory search error: %v", err)
		}
		if len(results) == 0 {
			return "No relevant results found."
		}
		return formatMemoryResults(results)

	default:
		return b.commands.Execute(cmd, args, agent)
	}
}

// ensureDefaultAgent creates the default agent and wildcard binding if they
// do not already exist.
func (b *Bot) ensureDefaultAgent(ctx context.Context) error {
	// Check if default agent already exists.
	_, err := b.store.GetAgent(ctx, "default")
	if err == nil {
		return nil // Already exists.
	}

	// Create the default agent.
	agent := &types.Agent{
		ID:          "default",
		Name:        "Default Assistant",
		Model:       "claude-sonnet-4-20250514",
		Workspace:   b.cfg.Workspace,
		MaxTokens:   4096,
		Temperature: 0.7,
		Status:      "active",
	}
	if err := b.store.CreateAgent(ctx, agent); err != nil {
		return fmt.Errorf("create default agent: %w", err)
	}

	// Create wildcard binding so all messages route to the default agent.
	binding := &types.Binding{
		AgentID:     "default",
		ChannelType: "*",
		ChannelID:   "*",
		PeerID:      "*",
		Priority:    0,
	}
	if err := b.store.CreateBinding(ctx, binding); err != nil {
		return fmt.Errorf("create default binding: %w", err)
	}

	return nil
}

// formatMemoryResults formats memory search results into a readable string
// suitable for inclusion in the system prompt.
func formatMemoryResults(results []memory.SearchResult) string {
	var sb strings.Builder
	sb.WriteString("# Relevant Context\n")

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("\n## Result %d: %s (lines %d-%d, score: %.2f)\n",
			i+1, r.Path, r.StartLine, r.EndLine, r.Score))
		sb.WriteString(r.Snippet)
		if !strings.HasSuffix(r.Snippet, "\n") {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
