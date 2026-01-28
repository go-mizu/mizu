package llamacpp

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
)

// skipIfNoServer skips the test if the llama.cpp server is not available.
func skipIfNoServer(t *testing.T, client *Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Skipping test: llama.cpp server not available: %v", err)
	}
}

func getTestURL() string {
	url := os.Getenv("LLAMACPP_QUICK_URL")
	if url == "" {
		url = "http://localhost:8082"
	}
	return url
}

func TestNew(t *testing.T) {
	client, err := New(Config{
		BaseURL: getTestURL(),
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if client == nil {
		t.Fatal("New() returned nil client")
	}
}

func TestNew_DefaultURL(t *testing.T) {
	client, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if client.baseURL != "http://localhost:8080" {
		t.Errorf("New() baseURL = %v, want http://localhost:8080", client.baseURL)
	}
}

func TestClient_Ping(t *testing.T) {
	client, err := New(Config{
		BaseURL: getTestURL(),
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Skipf("Skipping: llama.cpp server not available: %v", err)
	}
}

func TestClient_Models(t *testing.T) {
	client, err := New(Config{
		BaseURL: getTestURL(),
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	skipIfNoServer(t, client)

	ctx := context.Background()
	models, err := client.Models(ctx)
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	if len(models) == 0 {
		t.Error("Models() returned empty list")
	}

	// Log models for debugging
	for _, m := range models {
		t.Logf("Model: %s", m.ID)
	}
}

func TestClient_ChatCompletion(t *testing.T) {
	client, err := New(Config{
		BaseURL: getTestURL(),
		Timeout: 120 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	skipIfNoServer(t, client)

	ctx := context.Background()

	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "Say hello in one word."},
		},
		MaxTokens:   50,
		Temperature: 0.1,
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("ChatCompletion() returned no choices")
	}

	content := resp.Choices[0].Message.Content
	if content == "" {
		t.Error("ChatCompletion() returned empty content")
	}

	t.Logf("Response: %s", content)
}

func TestClient_ChatCompletion_SystemMessage(t *testing.T) {
	client, err := New(Config{
		BaseURL: getTestURL(),
		Timeout: 120 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	skipIfNoServer(t, client)

	ctx := context.Background()

	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a pirate. Respond only in pirate speak."},
			{Role: "user", Content: "How are you?"},
		},
		MaxTokens:   100,
		Temperature: 0.3,
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("ChatCompletion() returned no choices")
	}

	t.Logf("Pirate response: %s", resp.Choices[0].Message.Content)
}

func TestClient_ChatCompletionStream(t *testing.T) {
	client, err := New(Config{
		BaseURL: getTestURL(),
		Timeout: 120 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	skipIfNoServer(t, client)

	ctx := context.Background()

	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "Count from 1 to 5."},
		},
		MaxTokens:   100,
		Temperature: 0.1,
		Stream:      true,
	}

	stream, err := client.ChatCompletionStream(ctx, req)
	if err != nil {
		t.Fatalf("ChatCompletionStream() error = %v", err)
	}

	var tokens []string
	for event := range stream {
		if event.Error != nil {
			t.Fatalf("Stream error: %v", event.Error)
		}
		if event.Delta != "" {
			tokens = append(tokens, event.Delta)
		}
		if event.Done {
			break
		}
	}

	if len(tokens) == 0 {
		t.Error("ChatCompletionStream() returned no tokens")
	}

	fullResponse := strings.Join(tokens, "")
	t.Logf("Streamed response: %s", fullResponse)
}

func TestClient_Embedding(t *testing.T) {
	client, err := New(Config{
		BaseURL: getTestURL(),
		Timeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	skipIfNoServer(t, client)

	ctx := context.Background()

	req := llm.EmbeddingRequest{
		Input: []string{"Hello world"},
	}

	resp, err := client.Embedding(ctx, req)
	if err != nil {
		t.Fatalf("Embedding() error = %v", err)
	}

	if len(resp.Data) == 0 {
		t.Fatal("Embedding() returned no data")
	}

	if len(resp.Data[0].Embedding) == 0 {
		t.Error("Embedding() returned empty embedding vector")
	}

	t.Logf("Embedding dimensions: %d", len(resp.Data[0].Embedding))
}

func TestClient_Embedding_Multiple(t *testing.T) {
	client, err := New(Config{
		BaseURL: getTestURL(),
		Timeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	skipIfNoServer(t, client)

	ctx := context.Background()

	req := llm.EmbeddingRequest{
		Input: []string{
			"The quick brown fox",
			"jumps over the lazy dog",
		},
	}

	resp, err := client.Embedding(ctx, req)
	if err != nil {
		t.Fatalf("Embedding() error = %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("Embedding() returned %d embeddings, want 2", len(resp.Data))
	}
}

func TestRegistry(t *testing.T) {
	// Test that llamacpp is registered in the provider registry
	providers := llm.Providers()
	found := false
	for _, p := range providers {
		if p == "llamacpp" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("llamacpp provider not registered")
	}

	provider, err := llm.New("llamacpp", llm.Config{
		BaseURL: getTestURL(),
		Timeout: 30,
	})
	if err != nil {
		t.Fatalf("llm.New() error = %v", err)
	}

	if provider == nil {
		t.Fatal("llm.New() returned nil provider")
	}
}

func TestRegistry_Integration(t *testing.T) {
	// Full integration test using the registry
	provider, err := llm.New("llamacpp", llm.Config{
		BaseURL: getTestURL(),
		Timeout: 120,
	})
	if err != nil {
		t.Fatalf("llm.New() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Ping(ctx); err != nil {
		t.Skipf("Skipping: llama.cpp server not available: %v", err)
	}

	// Test chat completion through the provider interface
	resp, err := provider.ChatCompletion(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "Say yes or no."},
		},
		MaxTokens:   10,
		Temperature: 0.1,
	})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("ChatCompletion() returned no choices")
	}

	t.Logf("Provider response: %s", resp.Choices[0].Message.Content)
}

// TestGPTOSS20B tests the gpt-oss-20b model specifically.
func TestGPTOSS20B(t *testing.T) {
	url := os.Getenv("LLAMACPP_GPTOSS_URL")
	if url == "" {
		url = "http://localhost:8085"
	}

	client, err := New(Config{
		BaseURL: url,
		Timeout: 180 * time.Second, // Longer timeout for larger model
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	skipIfNoServer(t, client)

	ctx := context.Background()

	// Test reasoning capability
	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "If I have 3 apples and give away 1, then buy 5 more, how many do I have? Think step by step."},
		},
		MaxTokens:   200,
		Temperature: 0.3,
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("ChatCompletion() returned no choices")
	}

	content := resp.Choices[0].Message.Content
	t.Logf("GPT-OSS-20B reasoning response: %s", content)

	// Check if the answer contains "7"
	if !strings.Contains(content, "7") {
		t.Errorf("Expected answer to contain '7', got: %s", content)
	}
}

// TestAllModelsComparison runs a comparison test across all available models.
func TestAllModelsComparison(t *testing.T) {
	if os.Getenv("LLM_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test (set LLM_INTEGRATION_TEST=1 to run)")
	}

	models := []struct {
		name string
		url  string
	}{
		{"gemma-270m-quick", "http://localhost:8082"},
		{"gemma-1b-deep", "http://localhost:8083"},
		{"gemma-4b-research", "http://localhost:8084"},
		{"gpt-oss-20b", "http://localhost:8085"},
	}

	prompt := "What is 2 + 2? Answer with just the number."

	for _, m := range models {
		t.Run(m.name, func(t *testing.T) {
			client, err := New(Config{
				BaseURL: m.url,
				Timeout: 120 * time.Second,
			})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := client.Ping(ctx); err != nil {
				cancel()
				t.Skipf("Server not available at %s: %v", m.url, err)
				return
			}
			cancel()

			start := time.Now()
			resp, err := client.ChatCompletion(context.Background(), llm.ChatRequest{
				Messages: []llm.Message{
					{Role: "user", Content: prompt},
				},
				MaxTokens:   20,
				Temperature: 0,
			})
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("ChatCompletion() error = %v", err)
			}

			if len(resp.Choices) == 0 {
				t.Fatal("ChatCompletion() returned no choices")
			}

			content := resp.Choices[0].Message.Content
			correct := strings.Contains(content, "4")

			t.Logf("Model: %s", m.name)
			t.Logf("  Latency: %v", elapsed.Round(time.Millisecond))
			t.Logf("  Response: %s", strings.TrimSpace(content))
			t.Logf("  Correct: %v", correct)
			t.Logf("  Tokens: %d", resp.Usage.CompletionTokens)
		})
	}
}
