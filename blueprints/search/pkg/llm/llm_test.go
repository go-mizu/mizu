package llm

import (
	"testing"
)

func TestRegisterAndProviders(t *testing.T) {
	// Clear providers for test isolation
	providersMu.Lock()
	original := providers
	providers = make(map[string]ProviderFactory)
	providersMu.Unlock()

	defer func() {
		providersMu.Lock()
		providers = original
		providersMu.Unlock()
	}()

	// Register a mock provider
	Register("mock", func(cfg Config) (Provider, error) {
		return nil, nil
	})

	names := Providers()
	if len(names) != 1 {
		t.Errorf("expected 1 provider, got %d", len(names))
	}
	if names[0] != "mock" {
		t.Errorf("expected 'mock', got %q", names[0])
	}
}

func TestNewProviderNotFound(t *testing.T) {
	_, err := New("nonexistent", Config{})
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestChatRequestJSON(t *testing.T) {
	req := ChatRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	if req.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", req.Model)
	}
	if len(req.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(req.Messages))
	}
}

func TestMessageTypes(t *testing.T) {
	tests := []struct {
		role    string
		content string
	}{
		{"system", "You are a helpful assistant"},
		{"user", "Hello"},
		{"assistant", "Hi there!"},
	}

	for _, tt := range tests {
		msg := Message{Role: tt.role, Content: tt.content}
		if msg.Role != tt.role {
			t.Errorf("expected role %q, got %q", tt.role, msg.Role)
		}
		if msg.Content != tt.content {
			t.Errorf("expected content %q, got %q", tt.content, msg.Content)
		}
	}
}

func TestUsageCalculation(t *testing.T) {
	usage := Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}

	expected := usage.PromptTokens + usage.CompletionTokens
	if usage.TotalTokens != expected {
		t.Errorf("expected total %d, got %d", expected, usage.TotalTokens)
	}
}

func TestStreamEventDone(t *testing.T) {
	event := StreamEvent{Done: true}
	if !event.Done {
		t.Error("expected Done to be true")
	}

	event = StreamEvent{Delta: "hello"}
	if event.Done {
		t.Error("expected Done to be false")
	}
}

func TestEmbeddingRequestInput(t *testing.T) {
	req := EmbeddingRequest{
		Model: "text-embedding",
		Input: []string{"hello", "world"},
	}

	if len(req.Input) != 2 {
		t.Errorf("expected 2 inputs, got %d", len(req.Input))
	}
}

func TestModelStruct(t *testing.T) {
	m := Model{
		ID:      "gpt-oss-20b",
		Object:  "model",
		Created: 1234567890,
		OwnedBy: "openai",
	}

	if m.ID != "gpt-oss-20b" {
		t.Errorf("expected ID 'gpt-oss-20b', got %q", m.ID)
	}
	if m.OwnedBy != "openai" {
		t.Errorf("expected OwnedBy 'openai', got %q", m.OwnedBy)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{}
	if cfg.BaseURL != "" {
		t.Error("expected empty BaseURL by default")
	}
	if cfg.Timeout != 0 {
		t.Error("expected zero Timeout by default")
	}
}
