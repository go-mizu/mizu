package ai_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/feature/ai"
	"github.com/go-mizu/mizu/blueprints/search/feature/session"
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
)

// mockProvider implements llm.Provider for testing
type mockProvider struct {
	chatResponse   *llm.ChatResponse
	streamResponse []llm.StreamEvent
	embedResponse  *llm.EmbeddingResponse
	models         []llm.Model
	pingErr        error
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) ChatCompletion(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.chatResponse != nil {
		return m.chatResponse, nil
	}
	return &llm.ChatResponse{
		ID:      "test-id",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "test-model",
		Choices: []llm.Choice{
			{
				Index: 0,
				Message: llm.Message{
					Role:    "assistant",
					Content: "This is a test response about the topic. It includes citations [1] and [2] for verification.",
				},
				FinishReason: "stop",
			},
		},
		Usage: llm.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *mockProvider) ChatCompletionStream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	ch := make(chan llm.StreamEvent, 10)
	go func() {
		defer close(ch)
		if len(m.streamResponse) > 0 {
			for _, event := range m.streamResponse {
				ch <- event
			}
			return
		}
		// Default stream response
		tokens := []string{"This ", "is ", "a ", "test ", "response ", "[1]", "."}
		for _, token := range tokens {
			ch <- llm.StreamEvent{Delta: token}
		}
		ch <- llm.StreamEvent{Done: true}
	}()
	return ch, nil
}

func (m *mockProvider) Embedding(ctx context.Context, req llm.EmbeddingRequest) (*llm.EmbeddingResponse, error) {
	if m.embedResponse != nil {
		return m.embedResponse, nil
	}
	// Return mock embeddings
	data := make([]llm.EmbeddingData, len(req.Input))
	for i := range req.Input {
		data[i] = llm.EmbeddingData{
			Object:    "embedding",
			Index:     i,
			Embedding: make([]float32, 384), // Mock embedding
		}
	}
	return &llm.EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  "test-embed",
	}, nil
}

func (m *mockProvider) Models(ctx context.Context) ([]llm.Model, error) {
	if m.models != nil {
		return m.models, nil
	}
	return []llm.Model{
		{ID: "test-model", Object: "model", Created: time.Now().Unix(), OwnedBy: "test"},
	}, nil
}

func (m *mockProvider) Ping(ctx context.Context) error {
	return m.pingErr
}

// mockSearchService implements a minimal search interface
type mockSearchService struct{}

func (m *mockSearchService) Search(ctx context.Context, query string, opts interface{}) (interface{}, error) {
	// Return mock search results
	return map[string]interface{}{
		"results": []map[string]interface{}{
			{"url": "https://example.com/1", "title": "Result 1", "snippet": "This is result 1"},
			{"url": "https://example.com/2", "title": "Result 2", "snippet": "This is result 2"},
		},
		"total_results": 2,
	}, nil
}

// mockSessionStore implements session.Store for testing
type mockSessionStore struct {
	sessions map[string]*session.Session
	messages map[string][]*session.Message
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{
		sessions: make(map[string]*session.Session),
		messages: make(map[string][]*session.Message),
	}
}

func (m *mockSessionStore) Create(ctx context.Context, title string) (*session.Session, error) {
	sess := &session.Session{
		ID:        "test-session-id",
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.sessions[sess.ID] = sess
	return sess, nil
}

func (m *mockSessionStore) Get(ctx context.Context, id string) (*session.Session, error) {
	if sess, ok := m.sessions[id]; ok {
		return sess, nil
	}
	return nil, nil
}

func (m *mockSessionStore) List(ctx context.Context, limit, offset int) ([]*session.Session, int, error) {
	var sessions []*session.Session
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions, len(sessions), nil
}

func (m *mockSessionStore) Delete(ctx context.Context, id string) error {
	delete(m.sessions, id)
	delete(m.messages, id)
	return nil
}

func (m *mockSessionStore) AddMessage(ctx context.Context, sessionID, role, content, mode string, citations []session.Citation) (*session.Message, error) {
	msg := &session.Message{
		ID:        "test-message-id",
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Mode:      mode,
		Citations: citations,
		CreatedAt: time.Now(),
	}
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return msg, nil
}

func (m *mockSessionStore) GetMessages(ctx context.Context, sessionID string) ([]*session.Message, error) {
	return m.messages[sessionID], nil
}

func (m *mockSessionStore) GetConversationContext(ctx context.Context, sessionID string) ([]map[string]string, error) {
	var context []map[string]string
	for _, msg := range m.messages[sessionID] {
		context = append(context, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	return context, nil
}

func TestGetModes(t *testing.T) {
	modes := ai.GetModes()

	if len(modes) != 4 {
		t.Errorf("Expected 4 modes, got %d", len(modes))
	}

	expectedModes := map[string]bool{
		"quick":      false,
		"deep":       false,
		"research":   false,
		"deepsearch": false,
	}

	for _, mode := range modes {
		if _, ok := expectedModes[mode.ID]; !ok {
			t.Errorf("Unexpected mode: %s", mode.ID)
		}
		expectedModes[mode.ID] = true

		if mode.Name == "" {
			t.Errorf("Mode %s has empty name", mode.ID)
		}
		if mode.Description == "" {
			t.Errorf("Mode %s has empty description", mode.ID)
		}
	}

	for id, found := range expectedModes {
		if !found {
			t.Errorf("Missing mode: %s", id)
		}
	}
}

func TestModelRegistry(t *testing.T) {
	registry := ai.NewModelRegistry()

	// Test empty registry
	models := registry.ListModels()
	if len(models) != 0 {
		t.Errorf("Expected empty registry, got %d models", len(models))
	}

	// Register a model
	provider := &mockProvider{}
	registry.RegisterModel(ai.ModelInfo{
		ID:           "test-model",
		Provider:     "test",
		Name:         "Test Model",
		Capabilities: []ai.Capability{ai.CapabilityText},
		ContextSize:  4096,
		Speed:        "fast",
		Available:    true,
	}, provider)

	// Test model retrieval
	info, ok := registry.GetModel("test-model")
	if !ok {
		t.Error("Model not found after registration")
	}
	if info.Name != "Test Model" {
		t.Errorf("Expected 'Test Model', got '%s'", info.Name)
	}

	// Test provider retrieval
	p, ok := registry.GetProvider("test-model")
	if !ok {
		t.Error("Provider not found")
	}
	if p == nil {
		t.Error("Provider is nil")
	}

	// Test listing by capability
	textModels := registry.ListModelsByCapability(ai.CapabilityText)
	if len(textModels) != 1 {
		t.Errorf("Expected 1 text model, got %d", len(textModels))
	}

	visionModels := registry.ListModelsByCapability(ai.CapabilityVision)
	if len(visionModels) != 0 {
		t.Errorf("Expected 0 vision models, got %d", len(visionModels))
	}

	// Test default model
	info, p, ok = registry.GetDefaultModel(ai.CapabilityText)
	if !ok {
		t.Error("Default text model not found")
	}
	if info.ID != "test-model" {
		t.Errorf("Expected 'test-model' as default, got '%s'", info.ID)
	}
}

func TestModelRegistryHealth(t *testing.T) {
	registry := ai.NewModelRegistry()

	healthyProvider := &mockProvider{pingErr: nil}
	registry.RegisterModel(ai.ModelInfo{
		ID:           "healthy-model",
		Provider:     "test",
		Name:         "Healthy Model",
		Capabilities: []ai.Capability{ai.CapabilityText},
		ContextSize:  4096,
		Speed:        "fast",
		Available:    true,
	}, healthyProvider)

	ctx := context.Background()
	isHealthy := registry.CheckHealth(ctx, "healthy-model")
	if !isHealthy {
		t.Error("Expected healthy model to be healthy")
	}

	isHealthy = registry.CheckHealth(ctx, "nonexistent-model")
	if isHealthy {
		t.Error("Expected nonexistent model to be unhealthy")
	}
}

func TestModelInfoCapabilities(t *testing.T) {
	info := ai.ModelInfo{
		ID:           "multi-cap-model",
		Provider:     "test",
		Name:         "Multi-Capability Model",
		Capabilities: []ai.Capability{ai.CapabilityText, ai.CapabilityVision, ai.CapabilityEmbeddings},
		ContextSize:  8192,
		Speed:        "balanced",
		Available:    true,
	}

	if len(info.Capabilities) != 3 {
		t.Errorf("Expected 3 capabilities, got %d", len(info.Capabilities))
	}

	// Test capability constants
	if ai.CapabilityText != "text" {
		t.Errorf("CapabilityText should be 'text', got '%s'", ai.CapabilityText)
	}
	if ai.CapabilityVision != "vision" {
		t.Errorf("CapabilityVision should be 'vision', got '%s'", ai.CapabilityVision)
	}
	if ai.CapabilityEmbeddings != "embeddings" {
		t.Errorf("CapabilityEmbeddings should be 'embeddings', got '%s'", ai.CapabilityEmbeddings)
	}
	if ai.CapabilityVoice != "voice" {
		t.Errorf("CapabilityVoice should be 'voice', got '%s'", ai.CapabilityVoice)
	}
}

func TestModeConstants(t *testing.T) {
	if ai.ModeQuick != "quick" {
		t.Errorf("ModeQuick should be 'quick', got '%s'", ai.ModeQuick)
	}
	if ai.ModeDeep != "deep" {
		t.Errorf("ModeDeep should be 'deep', got '%s'", ai.ModeDeep)
	}
	if ai.ModeResearch != "research" {
		t.Errorf("ModeResearch should be 'research', got '%s'", ai.ModeResearch)
	}
	if ai.ModeDeepSearch != "deepsearch" {
		t.Errorf("ModeDeepSearch should be 'deepsearch', got '%s'", ai.ModeDeepSearch)
	}
}

func TestQueryStruct(t *testing.T) {
	query := ai.Query{
		Text:      "test query",
		Mode:      ai.ModeQuick,
		SessionID: "session-123",
	}

	if query.Text != "test query" {
		t.Errorf("Expected 'test query', got '%s'", query.Text)
	}
	if query.Mode != ai.ModeQuick {
		t.Errorf("Expected 'quick', got '%s'", query.Mode)
	}
	if query.SessionID != "session-123" {
		t.Errorf("Expected 'session-123', got '%s'", query.SessionID)
	}
}

func TestResponseStruct(t *testing.T) {
	resp := ai.Response{
		Answer: "Test answer [1]",
		Citations: []session.Citation{
			{Index: 1, URL: "https://example.com", Title: "Example", Snippet: "Test snippet"},
		},
		FollowUps: []string{"Follow up 1", "Follow up 2"},
		Sources: []ai.Source{
			{URL: "https://example.com", Title: "Example"},
		},
		SessionID: "session-123",
		Mode:      ai.ModeQuick,
	}

	if resp.Answer != "Test answer [1]" {
		t.Errorf("Unexpected answer: %s", resp.Answer)
	}
	if len(resp.Citations) != 1 {
		t.Errorf("Expected 1 citation, got %d", len(resp.Citations))
	}
	if len(resp.FollowUps) != 2 {
		t.Errorf("Expected 2 follow ups, got %d", len(resp.FollowUps))
	}
	if len(resp.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(resp.Sources))
	}
}

func TestStreamEventStruct(t *testing.T) {
	// Test token event
	tokenEvent := ai.StreamEvent{
		Type:    "token",
		Content: "Hello",
	}
	if tokenEvent.Type != "token" {
		t.Errorf("Expected type 'token', got '%s'", tokenEvent.Type)
	}

	// Test citation event
	cit := session.Citation{Index: 1, URL: "https://example.com", Title: "Example"}
	citEvent := ai.StreamEvent{
		Type:     "citation",
		Citation: &cit,
	}
	if citEvent.Citation.Index != 1 {
		t.Errorf("Expected citation index 1, got %d", citEvent.Citation.Index)
	}

	// Test done event with response
	doneEvent := ai.StreamEvent{
		Type: "done",
		Response: &ai.StreamResponse{
			FollowUps: []string{"Question 1", "Question 2"},
			SessionID: "session-123",
		},
	}
	if len(doneEvent.Response.FollowUps) != 2 {
		t.Errorf("Expected 2 follow ups, got %d", len(doneEvent.Response.FollowUps))
	}

	// Test error event
	errorEvent := ai.StreamEvent{
		Type:  "error",
		Error: "Something went wrong",
	}
	if errorEvent.Error != "Something went wrong" {
		t.Errorf("Expected error message, got '%s'", errorEvent.Error)
	}
}

func TestReasoningStepStruct(t *testing.T) {
	step := ai.ReasoningStep{
		Type:   "search",
		Input:  "query",
		Output: "Found 5 results",
	}

	if step.Type != "search" {
		t.Errorf("Expected type 'search', got '%s'", step.Type)
	}
	if step.Input != "query" {
		t.Errorf("Expected input 'query', got '%s'", step.Input)
	}
	if step.Output != "Found 5 results" {
		t.Errorf("Expected output 'Found 5 results', got '%s'", step.Output)
	}
}

func TestSourceStruct(t *testing.T) {
	now := time.Now()
	source := ai.Source{
		URL:       "https://example.com",
		Title:     "Example Site",
		FetchedAt: now,
	}

	if source.URL != "https://example.com" {
		t.Errorf("Unexpected URL: %s", source.URL)
	}
	if source.Title != "Example Site" {
		t.Errorf("Unexpected title: %s", source.Title)
	}
	if source.FetchedAt != now {
		t.Error("FetchedAt mismatch")
	}
}

func TestDeepSearchProgressStruct(t *testing.T) {
	progress := ai.DeepSearchProgress{
		Phase:       "fetching",
		Current:     5,
		Total:       50,
		Message:     "Fetching 5/50 sources",
		SectionName: "",
	}

	if progress.Phase != "fetching" {
		t.Errorf("Expected phase 'fetching', got '%s'", progress.Phase)
	}
	if progress.Current != 5 {
		t.Errorf("Expected current 5, got %d", progress.Current)
	}
	if progress.Total != 50 {
		t.Errorf("Expected total 50, got %d", progress.Total)
	}
}

func TestDeepSearchConfigDefaults(t *testing.T) {
	cfg := ai.DeepSearchConfig{
		MaxSources:     50,
		WorkerPool:     10,
		FetchTimeout:   30 * time.Second,
		AnalysisDepth:  5,
		ReportSections: 5,
	}

	if cfg.MaxSources != 50 {
		t.Errorf("Expected MaxSources 50, got %d", cfg.MaxSources)
	}
	if cfg.WorkerPool != 10 {
		t.Errorf("Expected WorkerPool 10, got %d", cfg.WorkerPool)
	}
	if cfg.FetchTimeout != 30*time.Second {
		t.Errorf("Expected FetchTimeout 30s, got %v", cfg.FetchTimeout)
	}
}

func TestReportSectionStruct(t *testing.T) {
	section := ai.ReportSection{
		Title:   "Key Findings",
		Content: "This section contains the key findings [1][2].",
		Order:   0,
	}

	if section.Title != "Key Findings" {
		t.Errorf("Unexpected title: %s", section.Title)
	}
	if section.Order != 0 {
		t.Errorf("Expected order 0, got %d", section.Order)
	}
}

func TestDeepSearchResponseStruct(t *testing.T) {
	resp := ai.DeepSearchResponse{
		Query:       "test query",
		Overview:    "This is an overview",
		KeyFindings: []string{"Finding 1", "Finding 2"},
		Sections: []ai.ReportSection{
			{Title: "Section 1", Content: "Content 1", Order: 0},
		},
		Methodology: "Research methodology",
		Citations:   []session.Citation{{Index: 1, URL: "https://example.com"}},
		Sources:     []ai.Source{{URL: "https://example.com"}},
		FollowUps:   []string{"Follow up 1"},
		SessionID:   "session-123",
		Mode:        ai.ModeDeepSearch,
		Duration:    5 * time.Second,
	}

	if resp.Query != "test query" {
		t.Errorf("Unexpected query: %s", resp.Query)
	}
	if len(resp.KeyFindings) != 2 {
		t.Errorf("Expected 2 key findings, got %d", len(resp.KeyFindings))
	}
	if len(resp.Sections) != 1 {
		t.Errorf("Expected 1 section, got %d", len(resp.Sections))
	}
	if resp.Mode != ai.ModeDeepSearch {
		t.Errorf("Expected mode 'deepsearch', got '%s'", resp.Mode)
	}
}

func TestModelErrorStruct(t *testing.T) {
	err := ai.ErrNoModelAvailable
	if err.Error() != "no model available" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestTokenUsageStruct(t *testing.T) {
	usage := ai.TokenUsage{
		InputTokens:     100,
		OutputTokens:    50,
		TotalTokens:     150,
		CacheReadTokens: 10,
		CostUSD:         0.001,
		TokensPerSecond: 25.5,
	}

	if usage.InputTokens != 100 {
		t.Errorf("Expected InputTokens 100, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 50 {
		t.Errorf("Expected OutputTokens 50, got %d", usage.OutputTokens)
	}
	if usage.TotalTokens != 150 {
		t.Errorf("Expected TotalTokens 150, got %d", usage.TotalTokens)
	}
	if usage.TokensPerSecond != 25.5 {
		t.Errorf("Expected TokensPerSecond 25.5, got %f", usage.TokensPerSecond)
	}
}

func TestStreamResponseWithUsage(t *testing.T) {
	// Test StreamResponse includes usage
	resp := ai.StreamResponse{
		Text:        "Test response",
		Mode:        ai.ModeQuick,
		Citations:   []session.Citation{{Index: 1, URL: "https://example.com", Title: "Test"}},
		FollowUps:   []string{"Follow up 1", "Follow up 2"},
		SessionID:   "test-session",
		SourcesUsed: 1,
		Usage: &ai.TokenUsage{
			InputTokens:    100,
			OutputTokens:   50,
			TotalTokens:    150,
			TokensPerSecond: 30.0,
		},
	}

	if resp.Usage == nil {
		t.Fatal("Expected Usage to be non-nil")
	}
	if resp.Usage.TotalTokens != 150 {
		t.Errorf("Expected TotalTokens 150, got %d", resp.Usage.TotalTokens)
	}
	if resp.Usage.TokensPerSecond != 30.0 {
		t.Errorf("Expected TokensPerSecond 30.0, got %f", resp.Usage.TokensPerSecond)
	}
}

func TestRelatedQuestionStruct(t *testing.T) {
	questions := []ai.RelatedQuestion{
		{Text: "What is the history?", Category: "deeper"},
		{Text: "How does it compare?", Category: "comparison"},
		{Text: "What are current trends?", Category: "current"},
	}

	if len(questions) != 3 {
		t.Errorf("Expected 3 questions, got %d", len(questions))
	}
	if questions[0].Text != "What is the history?" {
		t.Errorf("Unexpected question text: %s", questions[0].Text)
	}
	if questions[0].Category != "deeper" {
		t.Errorf("Unexpected category: %s", questions[0].Category)
	}
}

func TestResponseWithFollowUps(t *testing.T) {
	resp := ai.Response{
		Answer:   "Test answer",
		Mode:     ai.ModeQuick,
		FollowUps: []string{"Follow up 1", "Follow up 2", "Follow up 3"},
		RelatedQuestions: []ai.RelatedQuestion{
			{Text: "Related question 1", Category: "related"},
			{Text: "Related question 2", Category: "practical"},
		},
		Usage: &ai.TokenUsage{
			InputTokens:  200,
			OutputTokens: 100,
			TotalTokens:  300,
		},
	}

	if len(resp.FollowUps) != 3 {
		t.Errorf("Expected 3 follow-ups, got %d", len(resp.FollowUps))
	}
	if len(resp.RelatedQuestions) != 2 {
		t.Errorf("Expected 2 related questions, got %d", len(resp.RelatedQuestions))
	}
	if resp.Usage == nil {
		t.Fatal("Expected Usage to be non-nil")
	}
	if resp.Usage.TotalTokens != 300 {
		t.Errorf("Expected TotalTokens 300, got %d", resp.Usage.TotalTokens)
	}
}

func TestStreamEventWithTokenUsage(t *testing.T) {
	// Mock provider with token usage in stream events
	provider := &mockProvider{
		streamResponse: []llm.StreamEvent{
			{Delta: "Hello "},
			{Delta: "world"},
			{Done: true, InputTokens: 10, OutputTokens: 5},
		},
	}

	ctx := context.Background()
	stream, err := provider.ChatCompletionStream(ctx, llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var inputTokens, outputTokens int
	var content string
	for event := range stream {
		content += event.Delta
		if event.InputTokens > 0 {
			inputTokens = event.InputTokens
		}
		if event.OutputTokens > 0 {
			outputTokens = event.OutputTokens
		}
	}

	if content != "Hello world" {
		t.Errorf("Expected 'Hello world', got '%s'", content)
	}
	if inputTokens != 10 {
		t.Errorf("Expected inputTokens 10, got %d", inputTokens)
	}
	if outputTokens != 5 {
		t.Errorf("Expected outputTokens 5, got %d", outputTokens)
	}
}
