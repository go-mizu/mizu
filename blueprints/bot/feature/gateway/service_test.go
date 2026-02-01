package gateway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/workspace"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// ---------------------------------------------------------------------------
// mockStore implements store.Store with in-memory maps.
// ---------------------------------------------------------------------------

type mockStore struct {
	mu       sync.Mutex
	agent    *types.Agent
	sessions map[string]*types.Session
	messages []types.Message
	channels []types.Channel
	bindings []types.Binding
	seqID    int
}

func newMockStore(agent *types.Agent) *mockStore {
	return &mockStore{
		agent:    agent,
		sessions: make(map[string]*types.Session),
	}
}

func (m *mockStore) nextID() string {
	m.seqID++
	return fmt.Sprintf("id-%d", m.seqID)
}

// --- AgentStore ---

func (m *mockStore) ListAgents(_ context.Context) ([]types.Agent, error) {
	if m.agent == nil {
		return nil, nil
	}
	return []types.Agent{*m.agent}, nil
}

func (m *mockStore) GetAgent(_ context.Context, id string) (*types.Agent, error) {
	if m.agent != nil && m.agent.ID == id {
		return m.agent, nil
	}
	return nil, fmt.Errorf("agent not found: %s", id)
}

func (m *mockStore) CreateAgent(_ context.Context, a *types.Agent) error {
	m.agent = a
	return nil
}

func (m *mockStore) UpdateAgent(_ context.Context, a *types.Agent) error {
	m.agent = a
	return nil
}

func (m *mockStore) DeleteAgent(_ context.Context, _ string) error {
	m.agent = nil
	return nil
}

// --- ChannelStore ---

func (m *mockStore) ListChannels(_ context.Context) ([]types.Channel, error) {
	return m.channels, nil
}

func (m *mockStore) GetChannel(_ context.Context, id string) (*types.Channel, error) {
	for _, c := range m.channels {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("channel not found: %s", id)
}

func (m *mockStore) CreateChannel(_ context.Context, c *types.Channel) error {
	m.channels = append(m.channels, *c)
	return nil
}

func (m *mockStore) UpdateChannel(_ context.Context, _ *types.Channel) error { return nil }
func (m *mockStore) DeleteChannel(_ context.Context, _ string) error         { return nil }

// --- SessionStore ---

func (m *mockStore) ListSessions(_ context.Context) ([]types.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []types.Session
	for _, s := range m.sessions {
		out = append(out, *s)
	}
	return out, nil
}

func (m *mockStore) GetSession(_ context.Context, id string) (*types.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found: %s", id)
}

func (m *mockStore) GetOrCreateSession(_ context.Context, agentID, channelID, channelType, peerID, displayName, origin string) (*types.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Look for an existing active session matching the key.
	for _, s := range m.sessions {
		if s.AgentID == agentID && s.ChannelType == channelType && s.PeerID == peerID && s.Status == "active" {
			return s, nil
		}
	}

	id := m.nextID()
	now := time.Now().UTC()
	s := &types.Session{
		ID:          id,
		AgentID:     agentID,
		ChannelID:   channelID,
		ChannelType: channelType,
		PeerID:      peerID,
		DisplayName: displayName,
		Origin:      origin,
		Status:      "active",
		Metadata:    "{}",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.sessions[id] = s
	return s, nil
}

func (m *mockStore) UpdateSession(_ context.Context, s *types.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[s.ID]; ok {
		m.sessions[s.ID] = s
	}
	return nil
}

func (m *mockStore) DeleteSession(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *mockStore) ExpireSessions(_ context.Context, _ string, _ int) (int, error) {
	return 0, nil
}

// --- MessageStore ---

func (m *mockStore) ListMessages(_ context.Context, sessionID string, limit int) ([]types.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []types.Message
	for _, msg := range m.messages {
		if msg.SessionID == sessionID {
			out = append(out, msg)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func (m *mockStore) CreateMessage(_ context.Context, msg *types.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if msg.ID == "" {
		msg.ID = m.nextID()
	}
	msg.CreatedAt = time.Now().UTC()
	m.messages = append(m.messages, *msg)
	return nil
}

func (m *mockStore) CountMessages(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages), nil
}

// --- BindingStore ---

func (m *mockStore) ListBindings(_ context.Context) ([]types.Binding, error) {
	return m.bindings, nil
}

func (m *mockStore) CreateBinding(_ context.Context, b *types.Binding) error {
	m.bindings = append(m.bindings, *b)
	return nil
}

func (m *mockStore) DeleteBinding(_ context.Context, _ string) error { return nil }

func (m *mockStore) ResolveAgent(_ context.Context, _, _, _ string) (*types.Agent, error) {
	if m.agent == nil {
		return nil, fmt.Errorf("no agent configured")
	}
	return m.agent, nil
}

// --- Store-level ---

func (m *mockStore) Ensure(_ context.Context) error   { return nil }
func (m *mockStore) SeedData(_ context.Context) error  { return nil }
func (m *mockStore) Close() error                      { return nil }
func (m *mockStore) Stats(_ context.Context) (*store.Stats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &store.Stats{
		Agents:   1,
		Channels: len(m.channels),
		Sessions: len(m.sessions),
		Messages: len(m.messages),
	}, nil
}

// Compile-time interface check.
var _ store.Store = (*mockStore)(nil)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupWorkspace creates a workspace directory with bootstrap files and
// optional skill and indexable content files.
func setupWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create bootstrap files via workspace.EnsureWorkspace.
	if err := workspace.EnsureWorkspace(dir); err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}

	// Create a skill directory with a SKILL.md.
	skillDir := filepath.Join(dir, "skills", "test-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	skillContent := `---
name: test-skill
description: A test skill for integration tests
---
# Test Skill

This skill is used for integration testing purposes.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	// Create indexable files for memory testing.
	goContent := `package main

// GreetUser prints a greeting message to the console.
func GreetUser(name string) {
	fmt.Println("Hello, " + name)
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goContent), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	mdContent := `# Project Notes

This project implements a greeting service.
The primary function is GreetUser which takes a name parameter.
`
	if err := os.WriteFile(filepath.Join(dir, "NOTES.md"), []byte(mdContent), 0o644); err != nil {
		t.Fatalf("write NOTES.md: %v", err)
	}

	return dir
}

// testAgent returns a pre-configured agent for testing.
func testAgent(workspaceDir string) *types.Agent {
	return &types.Agent{
		ID:           "agent-1",
		Name:         "TestBot",
		Model:        "echo",
		SystemPrompt: "You are a helpful test assistant.",
		Workspace:    workspaceDir,
		MaxTokens:    1024,
		Temperature:  0.7,
		Status:       "active",
	}
}

// testMessage returns a basic inbound message for testing.
func testMessage(content string) *types.InboundMessage {
	return &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chan-1",
		PeerID:      "user-1",
		PeerName:    "TestUser",
		Content:     content,
		Origin:      "dm",
	}
}

// ---------------------------------------------------------------------------
// Tests: Context builder
// ---------------------------------------------------------------------------

func TestBuildSystemPrompt_BasePromptOnly(t *testing.T) {
	agent := &types.Agent{
		SystemPrompt: "You are a helpful assistant.",
	}

	memReg := newMemoryRegistry()
	defer memReg.closeAll()
	cb := newContextBuilder(memReg)

	prompt := cb.buildSystemPrompt(context.Background(), agent, "dm", "")
	if prompt != "You are a helpful assistant." {
		t.Errorf("expected base prompt only, got: %s", prompt)
	}
}

func TestBuildSystemPrompt_WithWorkspaceBootstrap(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)

	memReg := newMemoryRegistry()
	defer memReg.closeAll()
	cb := newContextBuilder(memReg)

	prompt := cb.buildSystemPrompt(context.Background(), agent, "dm", "")

	// Should contain base prompt.
	if !strings.Contains(prompt, "You are a helpful test assistant.") {
		t.Error("prompt should contain base system prompt")
	}

	// Should contain bootstrap content (EnsureWorkspace creates AGENTS.md, SOUL.md, etc.).
	if !strings.Contains(prompt, "# Project Context") {
		t.Error("prompt should contain workspace project context section")
	}
	if !strings.Contains(prompt, "AGENTS.md") {
		t.Error("prompt should reference AGENTS.md")
	}
	if !strings.Contains(prompt, "SOUL.md") {
		t.Error("prompt should reference SOUL.md")
	}
}

func TestBuildSystemPrompt_WithSkills(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)

	memReg := newMemoryRegistry()
	defer memReg.closeAll()
	cb := newContextBuilder(memReg)

	prompt := cb.buildSystemPrompt(context.Background(), agent, "dm", "")

	// Should contain skills section (XML format).
	if !strings.Contains(prompt, "<available_skills>") {
		t.Error("prompt should contain available skills section")
	}
	if !strings.Contains(prompt, "<name>test-skill</name>") {
		t.Error("prompt should contain test-skill name")
	}
	if !strings.Contains(prompt, "<description>A test skill for integration tests</description>") {
		t.Error("prompt should contain test skill description")
	}
}

func TestBuildSystemPrompt_WithMemorySearch(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)

	memReg := newMemoryRegistry()
	defer memReg.closeAll()
	cb := newContextBuilder(memReg)

	// Use a query that matches the Go file content.
	prompt := cb.buildSystemPrompt(context.Background(), agent, "dm", "GreetUser function")

	// Should contain memory results because the workspace has indexable files.
	if !strings.Contains(prompt, "# Relevant Context") {
		t.Error("prompt should contain relevant context section from memory search")
	}
	if !strings.Contains(prompt, "GreetUser") {
		t.Error("prompt should contain matching memory result for GreetUser")
	}
}

func TestBuildSystemPrompt_NoWorkspace(t *testing.T) {
	agent := &types.Agent{
		SystemPrompt: "Base only.",
	}

	memReg := newMemoryRegistry()
	defer memReg.closeAll()
	cb := newContextBuilder(memReg)

	prompt := cb.buildSystemPrompt(context.Background(), agent, "dm", "query")

	// With no workspace, should only have the base prompt.
	if prompt != "Base only." {
		t.Errorf("expected only base prompt, got: %s", prompt)
	}
}

func TestBuildSystemPrompt_SubagentOrigin(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)

	memReg := newMemoryRegistry()
	defer memReg.closeAll()
	cb := newContextBuilder(memReg)

	prompt := cb.buildSystemPrompt(context.Background(), agent, "subagent", "")

	// Subagent sessions only load AGENTS.md and TOOLS.md.
	if !strings.Contains(prompt, "AGENTS.md") {
		t.Error("subagent prompt should contain AGENTS.md")
	}
	if !strings.Contains(prompt, "TOOLS.md") {
		t.Error("subagent prompt should contain TOOLS.md")
	}
	// Should NOT contain SOUL.md or USER.md for subagent.
	if strings.Contains(prompt, "## SOUL.md") {
		t.Error("subagent prompt should not contain SOUL.md section")
	}
	if strings.Contains(prompt, "## USER.md") {
		t.Error("subagent prompt should not contain USER.md section")
	}
}

// ---------------------------------------------------------------------------
// Tests: Memory registry
// ---------------------------------------------------------------------------

func TestMemoryRegistry_EmptyWorkspace(t *testing.T) {
	reg := newMemoryRegistry()
	defer reg.closeAll()

	mgr, err := reg.get("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgr != nil {
		t.Error("expected nil manager for empty workspace dir")
	}
}

func TestMemoryRegistry_CachesManager(t *testing.T) {
	dir := setupWorkspace(t)
	reg := newMemoryRegistry()
	defer reg.closeAll()

	mgr1, err := reg.get(dir)
	if err != nil {
		t.Fatalf("first get: %v", err)
	}
	if mgr1 == nil {
		t.Fatal("expected non-nil manager")
	}

	mgr2, err := reg.get(dir)
	if err != nil {
		t.Fatalf("second get: %v", err)
	}

	// Should return the same cached instance.
	if mgr1 != mgr2 {
		t.Error("expected same cached manager instance on second get")
	}
}

func TestMemoryRegistry_IndexesFiles(t *testing.T) {
	dir := setupWorkspace(t)
	reg := newMemoryRegistry()
	defer reg.closeAll()

	mgr, err := reg.get(dir)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	// The workspace contains main.go and NOTES.md, so searching for content
	// from those files should return results.
	results, err := mgr.Search(context.Background(), "GreetUser", 0, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected search results for indexed workspace files")
	}
}

func TestMemoryRegistry_CloseAll(t *testing.T) {
	dir := setupWorkspace(t)
	reg := newMemoryRegistry()

	_, err := reg.get(dir)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if len(reg.managers) != 1 {
		t.Fatalf("expected 1 manager, got %d", len(reg.managers))
	}

	reg.closeAll()

	if len(reg.managers) != 0 {
		t.Errorf("expected 0 managers after closeAll, got %d", len(reg.managers))
	}
}

func TestMemoryRegistry_NonexistentDir(t *testing.T) {
	reg := newMemoryRegistry()
	defer reg.closeAll()

	mgr, err := reg.get("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgr != nil {
		t.Error("expected nil manager for nonexistent directory")
	}
}

// ---------------------------------------------------------------------------
// Tests: ProcessMessage flow
// ---------------------------------------------------------------------------

func TestProcessMessage_BasicFlow(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()
	msg := testMessage("Hello, bot!")

	resp, err := svc.ProcessMessage(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Echo provider returns "[Echo] You said: <last message>".
	if !strings.Contains(resp, "[Echo]") {
		t.Errorf("expected echo response, got: %s", resp)
	}
	if !strings.Contains(resp, "Hello, bot!") {
		t.Errorf("expected echo of user message, got: %s", resp)
	}

	// Verify messages were stored: 1 user + 1 assistant = 2.
	ms.mu.Lock()
	msgCount := len(ms.messages)
	ms.mu.Unlock()
	if msgCount != 2 {
		t.Errorf("expected 2 stored messages (user + assistant), got %d", msgCount)
	}
}

func TestProcessMessage_CreatesSession(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()
	msg := testMessage("Hi")

	_, err := svc.ProcessMessage(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	ms.mu.Lock()
	sessionCount := len(ms.sessions)
	ms.mu.Unlock()

	if sessionCount != 1 {
		t.Errorf("expected 1 session, got %d", sessionCount)
	}
}

func TestProcessMessage_ReusesSameSession(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()

	// Send two messages from same peer.
	_, err := svc.ProcessMessage(ctx, testMessage("First"))
	if err != nil {
		t.Fatalf("first message: %v", err)
	}
	_, err = svc.ProcessMessage(ctx, testMessage("Second"))
	if err != nil {
		t.Fatalf("second message: %v", err)
	}

	ms.mu.Lock()
	sessionCount := len(ms.sessions)
	ms.mu.Unlock()

	if sessionCount != 1 {
		t.Errorf("expected 1 session (reused), got %d", sessionCount)
	}
}

func TestProcessMessage_StoresUserAndAssistantMessages(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()
	_, err := svc.ProcessMessage(ctx, testMessage("What is Go?"))
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	if len(ms.messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(ms.messages))
	}

	userMsg := ms.messages[0]
	if userMsg.Role != types.RoleUser {
		t.Errorf("first message should be user role, got %s", userMsg.Role)
	}
	if userMsg.Content != "What is Go?" {
		t.Errorf("user message content mismatch: %s", userMsg.Content)
	}

	assistantMsg := ms.messages[1]
	if assistantMsg.Role != types.RoleAssistant {
		t.Errorf("second message should be assistant role, got %s", assistantMsg.Role)
	}
	if !strings.Contains(assistantMsg.Content, "[Echo]") {
		t.Errorf("assistant message should be echo response, got: %s", assistantMsg.Content)
	}
}

func TestProcessMessage_EnrichedSystemPrompt(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)

	// Use a custom LLM that captures the system prompt.
	var capturedSystemPrompt string
	captureLLM := &capturingLLM{
		onChat: func(req *types.LLMRequest) {
			capturedSystemPrompt = req.SystemPrompt
		},
	}

	svc := NewService(ms, captureLLM)
	defer svc.Close()

	ctx := context.Background()
	_, err := svc.ProcessMessage(ctx, testMessage("Tell me about GreetUser"))
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// The system prompt should contain the base prompt, workspace context, and skills.
	if !strings.Contains(capturedSystemPrompt, "You are a helpful test assistant.") {
		t.Error("system prompt should contain agent base prompt")
	}
	if !strings.Contains(capturedSystemPrompt, "# Project Context") {
		t.Error("system prompt should contain workspace context")
	}
	if !strings.Contains(capturedSystemPrompt, "<available_skills>") {
		t.Error("system prompt should contain skills section")
	}
}

// capturingLLM is a test LLM that captures requests and delegates to Echo.
type capturingLLM struct {
	onChat func(req *types.LLMRequest)
}

func (c *capturingLLM) Chat(ctx context.Context, req *types.LLMRequest) (*types.LLMResponse, error) {
	if c.onChat != nil {
		c.onChat(req)
	}
	return (&llm.Echo{}).Chat(ctx, req)
}

func TestProcessMessage_MultiTurnConversation(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()

	// Send multiple messages to build up conversation history.
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("Message %d", i+1)
		_, err := svc.ProcessMessage(ctx, testMessage(content))
		if err != nil {
			t.Fatalf("ProcessMessage %d: %v", i+1, err)
		}
	}

	// Should have 5 user + 5 assistant = 10 messages.
	ms.mu.Lock()
	msgCount := len(ms.messages)
	ms.mu.Unlock()

	if msgCount != 10 {
		t.Errorf("expected 10 messages after 5 turns, got %d", msgCount)
	}
}

// ---------------------------------------------------------------------------
// Tests: Command handling
// ---------------------------------------------------------------------------

func TestCommand_Context(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()
	msg := testMessage("/context")

	resp, err := svc.ProcessMessage(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// /context should return the enriched system prompt.
	if !strings.HasPrefix(resp, "System prompt:") {
		t.Errorf("expected system prompt response, got: %s", resp)
	}
	if !strings.Contains(resp, "You are a helpful test assistant.") {
		t.Error("/context response should contain base system prompt")
	}
	if !strings.Contains(resp, "# Project Context") {
		t.Error("/context response should contain workspace context")
	}
}

func TestCommand_New(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()

	// Send a regular message first to create a session.
	_, err := svc.ProcessMessage(ctx, testMessage("Hello"))
	if err != nil {
		t.Fatalf("first message: %v", err)
	}

	// Now send /new to expire the session.
	resp, err := svc.ProcessMessage(ctx, testMessage("/new"))
	if err != nil {
		t.Fatalf("/new: %v", err)
	}

	if !strings.Contains(resp, "New session started") {
		t.Errorf("expected new session response, got: %s", resp)
	}

	// The session should be expired.
	ms.mu.Lock()
	expiredCount := 0
	for _, s := range ms.sessions {
		if s.Status == "expired" {
			expiredCount++
		}
	}
	ms.mu.Unlock()

	if expiredCount == 0 {
		t.Error("expected at least one expired session after /new")
	}
}

func TestCommand_Reset(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()

	// Create a session first.
	_, err := svc.ProcessMessage(ctx, testMessage("Setup"))
	if err != nil {
		t.Fatalf("setup message: %v", err)
	}

	// Send /reset.
	resp, err := svc.ProcessMessage(ctx, testMessage("/reset"))
	if err != nil {
		t.Fatalf("/reset: %v", err)
	}

	if !strings.Contains(resp, "Session reset") {
		t.Errorf("expected reset response, got: %s", resp)
	}

	// Verify session was expired.
	ms.mu.Lock()
	expiredCount := 0
	for _, s := range ms.sessions {
		if s.Status == "expired" {
			expiredCount++
		}
	}
	ms.mu.Unlock()

	if expiredCount == 0 {
		t.Error("expected at least one expired session after /reset")
	}
}

func TestCommand_Help(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()
	resp, err := svc.ProcessMessage(ctx, testMessage("/help"))
	if err != nil {
		t.Fatalf("/help: %v", err)
	}

	if !strings.Contains(resp, "Available commands") {
		t.Errorf("expected help listing, got: %s", resp)
	}
	if !strings.Contains(resp, "/new") {
		t.Error("help should list /new command")
	}
	if !strings.Contains(resp, "/context") {
		t.Error("help should list /context command")
	}
}

func TestCommand_DoesNotStoreMessages(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()
	_, err := svc.ProcessMessage(ctx, testMessage("/help"))
	if err != nil {
		t.Fatalf("/help: %v", err)
	}

	// Commands should not store user/assistant messages.
	ms.mu.Lock()
	msgCount := len(ms.messages)
	ms.mu.Unlock()

	if msgCount != 0 {
		t.Errorf("commands should not store messages, got %d", msgCount)
	}
}

// ---------------------------------------------------------------------------
// Tests: MemorySearch
// ---------------------------------------------------------------------------

func TestMemorySearch_WithIndexedFiles(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	// Trigger memory indexing by first processing a message (which initializes
	// the memoryRegistry for the workspace).
	ctx := context.Background()
	_, err := svc.ProcessMessage(ctx, testMessage("init"))
	if err != nil {
		t.Fatalf("init message: %v", err)
	}

	// Now search for content in the indexed files.
	result, err := svc.MemorySearch(ctx, dir, "GreetUser")
	if err != nil {
		t.Fatalf("MemorySearch: %v", err)
	}

	if strings.Contains(result, "No memory index") || strings.Contains(result, "No relevant results") {
		t.Errorf("expected search results, got: %s", result)
	}
	if !strings.Contains(result, "GreetUser") {
		t.Errorf("search results should contain GreetUser, got: %s", result)
	}
}

func TestMemorySearch_EmptyWorkspace(t *testing.T) {
	ms := newMockStore(testAgent(""))
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	result, err := svc.MemorySearch(context.Background(), "", "anything")
	if err != nil {
		t.Fatalf("MemorySearch: %v", err)
	}
	if result != "No memory index available." {
		t.Errorf("expected no index message, got: %s", result)
	}
}

func TestMemorySearch_EmptyQuery(t *testing.T) {
	dir := setupWorkspace(t)
	ms := newMockStore(testAgent(dir))
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	result, err := svc.MemorySearch(context.Background(), dir, "")
	if err != nil {
		t.Fatalf("MemorySearch: %v", err)
	}
	if result != "No memory index available." {
		t.Errorf("expected no index message for empty query, got: %s", result)
	}
}

func TestMemorySearch_NoMatch(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	ctx := context.Background()
	// Initialize memory index.
	_, _ = svc.ProcessMessage(ctx, testMessage("init"))

	result, err := svc.MemorySearch(ctx, dir, "xyznonexistentquery123456")
	if err != nil {
		t.Fatalf("MemorySearch: %v", err)
	}
	if !strings.Contains(result, "No relevant results") {
		t.Errorf("expected no results message, got: %s", result)
	}
}

// ---------------------------------------------------------------------------
// Tests: Status and Commands
// ---------------------------------------------------------------------------

func TestStatus(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	status, err := svc.Status(context.Background(), 8080)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if status.Status != "running" {
		t.Errorf("expected running status, got: %s", status.Status)
	}
	if status.Port != 8080 {
		t.Errorf("expected port 8080, got: %d", status.Port)
	}
	if status.ActiveAgents != 1 {
		t.Errorf("expected 1 active agent, got: %d", status.ActiveAgents)
	}
}

func TestCommands(t *testing.T) {
	ms := newMockStore(testAgent(""))
	svc := NewService(ms, &llm.Echo{})
	defer svc.Close()

	cmds := svc.Commands()
	if len(cmds) == 0 {
		t.Fatal("expected non-empty commands list")
	}

	cmdNames := make(map[string]bool)
	for _, c := range cmds {
		cmdNames[c.Name] = true
	}

	expected := []string{"/new", "/reset", "/help", "/context", "/memory", "/compact"}
	for _, name := range expected {
		if !cmdNames[name] {
			t.Errorf("missing expected command: %s", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Service lifecycle
// ---------------------------------------------------------------------------

func TestServiceClose(t *testing.T) {
	dir := setupWorkspace(t)
	agent := testAgent(dir)
	ms := newMockStore(agent)
	svc := NewService(ms, &llm.Echo{})

	// Initialize memory registry.
	ctx := context.Background()
	_, _ = svc.ProcessMessage(ctx, testMessage("init"))

	// Close should not panic and should clean up memory managers.
	svc.Close()

	if len(svc.memReg.managers) != 0 {
		t.Error("expected all memory managers to be cleaned up after Close")
	}
}
