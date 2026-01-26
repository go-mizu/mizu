package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// ModelEndpoint represents a model server endpoint for testing.
type ModelEndpoint struct {
	Name     string
	URL      string
	Provider string // "llamacpp" or "claude"
	Model    string // Model ID for Claude
	Enabled  bool
}

// DefaultEndpoints returns the standard model endpoints for comparison.
func DefaultEndpoints() []ModelEndpoint {
	endpoints := []ModelEndpoint{
		{Name: "gemma-270m-quick", URL: "http://localhost:8082", Provider: "llamacpp", Enabled: true},
		{Name: "gemma-1b-deep", URL: "http://localhost:8083", Provider: "llamacpp", Enabled: true},
		{Name: "gemma-4b-research", URL: "http://localhost:8084", Provider: "llamacpp", Enabled: true},
		{Name: "gpt-oss-20b", URL: "http://localhost:8085", Provider: "llamacpp", Enabled: true},
	}

	// Add Claude endpoints if API key is available
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		endpoints = append(endpoints,
			ModelEndpoint{Name: "claude-haiku", Provider: "claude", Model: "claude-3.5-haiku", Enabled: true},
			ModelEndpoint{Name: "claude-sonnet", Provider: "claude", Model: "claude-sonnet-4", Enabled: true},
		)
	}

	return endpoints
}

// BenchmarkResult holds the results of a model benchmark.
type BenchmarkResult struct {
	Model           string        `json:"model"`
	Provider        string        `json:"provider"`
	URL             string        `json:"url,omitempty"`
	Latency         time.Duration `json:"latency"`
	InputTokens     int           `json:"input_tokens"`
	TokensGenerated int           `json:"tokens_generated"`
	TokensPerSecond float64       `json:"tokens_per_second"`
	CostUSD         float64       `json:"cost_usd,omitempty"`
	ResponseQuality float64       `json:"response_quality"` // 0-1 score based on response criteria
	Response        string        `json:"response,omitempty"`
	Error           error         `json:"error,omitempty"`
}

// TestModelsComparison runs a comparison test across all available models.
// This test is skipped unless LLM_INTEGRATION_TEST=1 is set.
func TestModelsComparison(t *testing.T) {
	if os.Getenv("LLM_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test (set LLM_INTEGRATION_TEST=1 to run)")
	}

	endpoints := DefaultEndpoints()
	prompts := []struct {
		name     string
		prompt   string
		criteria []string
	}{
		{
			name:     "factual",
			prompt:   "What is the capital of France? Answer in one sentence.",
			criteria: []string{"Paris"},
		},
		{
			name:     "reasoning",
			prompt:   "If a train travels 60 miles in 1 hour, how far will it travel in 2.5 hours? Show your calculation.",
			criteria: []string{"150", "miles"},
		},
		{
			name:     "creative",
			prompt:   "Write a haiku about programming.",
			criteria: []string{}, // No specific criteria for creative
		},
	}

	results := make(map[string][]BenchmarkResult)

	for _, ep := range endpoints {
		if !ep.Enabled {
			continue
		}

		t.Run(ep.Name, func(t *testing.T) {
			provider, err := createTestProvider(ep)
			if err != nil {
				t.Skipf("Could not create provider for %s: %v", ep.Name, err)
				return
			}

			// Check connectivity (skip for Claude as Ping is expensive)
			if ep.Provider != "claude" {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := provider.Ping(ctx); err != nil {
					cancel()
					t.Skipf("Server %s not available: %v", ep.Name, err)
					return
				}
				cancel()
			}

			for _, p := range prompts {
				t.Run(p.name, func(t *testing.T) {
					result := benchmarkPrompt(provider, ep, p.prompt, p.criteria)
					results[p.name] = append(results[p.name], result)

					if result.Error != nil {
						t.Errorf("Error: %v", result.Error)
						return
					}

					logMsg := fmt.Sprintf("Model: %s, Latency: %v, Tokens/sec: %.2f, Quality: %.2f",
						result.Model, result.Latency, result.TokensPerSecond, result.ResponseQuality)
					if result.CostUSD > 0 {
						logMsg += fmt.Sprintf(", Cost: $%.6f", result.CostUSD)
					}
					t.Log(logMsg)
				})
			}
		})
	}

	// Print summary
	printComparisonSummary(t, results)
}

func createTestProvider(ep ModelEndpoint) (Provider, error) {
	switch ep.Provider {
	case "claude":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		return New("claude", Config{
			APIKey:  apiKey,
			Timeout: 120,
		})
	default:
		// Default to llamacpp
		return New("llamacpp", Config{
			BaseURL: ep.URL,
			Timeout: 120,
		})
	}
}

func benchmarkPrompt(provider Provider, ep ModelEndpoint, prompt string, criteria []string) BenchmarkResult {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	resp, err := provider.ChatCompletion(ctx, ChatRequest{
		Model: ep.Model,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   256,
		Temperature: 0.3,
	})

	result := BenchmarkResult{
		Model:    ep.Name,
		Provider: ep.Provider,
		URL:      ep.URL,
	}

	if err != nil {
		result.Error = err
		return result
	}

	result.Latency = time.Since(start)
	result.InputTokens = resp.Usage.PromptTokens
	result.TokensGenerated = resp.Usage.CompletionTokens
	if result.Latency > 0 {
		result.TokensPerSecond = float64(result.TokensGenerated) / result.Latency.Seconds()
	}

	// Calculate cost for Claude
	if ep.Provider == "claude" {
		// Cost calculation (simplified - actual rates may vary)
		// Haiku: $0.25/1M input, $1.25/1M output
		// Sonnet: $3/1M input, $15/1M output
		inputCost := float64(result.InputTokens) / 1_000_000
		outputCost := float64(result.TokensGenerated) / 1_000_000
		switch ep.Model {
		case "claude-3.5-haiku":
			result.CostUSD = inputCost*0.80 + outputCost*4.00
		case "claude-sonnet-4":
			result.CostUSD = inputCost*3.00 + outputCost*15.00
		case "claude-opus-4":
			result.CostUSD = inputCost*15.00 + outputCost*75.00
		}
	}

	// Evaluate quality based on criteria
	if len(resp.Choices) > 0 {
		result.Response = resp.Choices[0].Message.Content
		if len(criteria) > 0 {
			content := strings.ToLower(result.Response)
			matches := 0
			for _, c := range criteria {
				if strings.Contains(content, strings.ToLower(c)) {
					matches++
				}
			}
			result.ResponseQuality = float64(matches) / float64(len(criteria))
		} else {
			// For creative prompts, just check we got a response
			result.ResponseQuality = 1.0
		}
	}

	return result
}

func printComparisonSummary(t *testing.T, results map[string][]BenchmarkResult) {
	t.Log("\n=== Model Comparison Summary ===")

	for promptName, benchmarks := range results {
		t.Logf("\nPrompt: %s", promptName)
		t.Log("----------------------------------------")

		var fastestLatency time.Duration
		var fastestModel string
		var bestQuality float64
		var bestQualityModel string

		for _, b := range benchmarks {
			if b.Error != nil {
				continue
			}

			if fastestLatency == 0 || b.Latency < fastestLatency {
				fastestLatency = b.Latency
				fastestModel = b.Model
			}

			if b.ResponseQuality > bestQuality {
				bestQuality = b.ResponseQuality
				bestQualityModel = b.Model
			}

			t.Logf("  %s: %v (%.2f tok/s, quality: %.2f)",
				b.Model, b.Latency.Round(time.Millisecond), b.TokensPerSecond, b.ResponseQuality)
		}

		if fastestModel != "" {
			t.Logf("  → Fastest: %s (%v)", fastestModel, fastestLatency.Round(time.Millisecond))
		}
		if bestQualityModel != "" && bestQuality > 0 {
			t.Logf("  → Best quality: %s (%.2f)", bestQualityModel, bestQuality)
		}
	}
}

// TestModelLatencyBenchmark runs latency benchmarks.
func TestModelLatencyBenchmark(t *testing.T) {
	if os.Getenv("LLM_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test (set LLM_INTEGRATION_TEST=1 to run)")
	}

	endpoints := DefaultEndpoints()
	iterations := 3

	for _, ep := range endpoints {
		if !ep.Enabled {
			continue
		}

		t.Run(ep.Name+"_latency", func(t *testing.T) {
			provider, err := createTestProvider(ep)
			if err != nil {
				t.Skipf("Could not create provider: %v", err)
				return
			}

			// Skip ping for Claude as it's expensive
			if ep.Provider != "claude" {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := provider.Ping(ctx); err != nil {
					cancel()
					t.Skipf("Server not available: %v", err)
					return
				}
				cancel()
			}

			var totalLatency time.Duration
			var totalTokens int

			for i := 0; i < iterations; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				start := time.Now()

				resp, err := provider.ChatCompletion(ctx, ChatRequest{
					Model: ep.Model,
					Messages: []Message{
						{Role: "user", Content: "Say hello in one word."},
					},
					MaxTokens:   10,
					Temperature: 0,
				})
				cancel()

				if err != nil {
					t.Errorf("Iteration %d failed: %v", i+1, err)
					continue
				}

				latency := time.Since(start)
				totalLatency += latency
				totalTokens += resp.Usage.CompletionTokens
			}

			avgLatency := totalLatency / time.Duration(iterations)
			t.Logf("Average latency: %v", avgLatency.Round(time.Millisecond))
			t.Logf("Average tokens/sec: %.2f", float64(totalTokens)/totalLatency.Seconds())
		})
	}
}

// TestEmbeddingComparison compares embedding generation across models.
func TestEmbeddingComparison(t *testing.T) {
	if os.Getenv("LLM_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test (set LLM_INTEGRATION_TEST=1 to run)")
	}

	// Only test embedding-capable endpoints (Claude doesn't support embeddings)
	endpoints := []ModelEndpoint{
		{Name: "gemma-270m-quick", URL: "http://localhost:8082", Provider: "llamacpp", Enabled: true},
		{Name: "gemma-1b-deep", URL: "http://localhost:8083", Provider: "llamacpp", Enabled: true},
		{Name: "gemma-4b-research", URL: "http://localhost:8084", Provider: "llamacpp", Enabled: true},
	}

	testInputs := []string{
		"The quick brown fox jumps over the lazy dog.",
		"Machine learning is transforming how we build software.",
	}

	for _, ep := range endpoints {
		if !ep.Enabled {
			continue
		}

		t.Run(ep.Name+"_embedding", func(t *testing.T) {
			provider, err := createTestProvider(ep)
			if err != nil {
				t.Skipf("Could not create provider: %v", err)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			start := time.Now()
			resp, err := provider.Embedding(ctx, EmbeddingRequest{
				Input: testInputs,
			})
			if err != nil {
				t.Skipf("Embedding not supported or failed: %v", err)
				return
			}

			latency := time.Since(start)

			if len(resp.Data) != len(testInputs) {
				t.Errorf("Expected %d embeddings, got %d", len(testInputs), len(resp.Data))
				return
			}

			t.Logf("Latency: %v", latency.Round(time.Millisecond))
			for i, data := range resp.Data {
				t.Logf("Input %d: %d dimensions", i+1, len(data.Embedding))
			}
		})
	}
}

// TestProviderCostComparison compares costs between providers.
func TestProviderCostComparison(t *testing.T) {
	if os.Getenv("LLM_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test (set LLM_INTEGRATION_TEST=1 to run)")
	}

	claudeKey := os.Getenv("ANTHROPIC_API_KEY")
	if claudeKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set - skipping cost comparison")
	}

	// Compare costs for same prompt across tiers
	endpoints := []ModelEndpoint{
		{Name: "claude-haiku", Provider: "claude", Model: "claude-3.5-haiku", Enabled: true},
		{Name: "claude-sonnet", Provider: "claude", Model: "claude-sonnet-4", Enabled: true},
	}

	prompt := "Explain the theory of relativity in 100 words."
	var results []BenchmarkResult

	for _, ep := range endpoints {
		t.Run(ep.Name, func(t *testing.T) {
			provider, err := createTestProvider(ep)
			if err != nil {
				t.Skipf("Could not create provider: %v", err)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			start := time.Now()
			resp, err := provider.ChatCompletion(ctx, ChatRequest{
				Model: ep.Model,
				Messages: []Message{
					{Role: "user", Content: prompt},
				},
				MaxTokens:   256,
				Temperature: 0,
			})

			if err != nil {
				t.Errorf("Error: %v", err)
				return
			}

			latency := time.Since(start)
			inputTokens := resp.Usage.PromptTokens
			outputTokens := resp.Usage.CompletionTokens

			// Calculate cost
			var cost float64
			switch ep.Model {
			case "claude-3.5-haiku":
				cost = float64(inputTokens)/1_000_000*0.80 + float64(outputTokens)/1_000_000*4.00
			case "claude-sonnet-4":
				cost = float64(inputTokens)/1_000_000*3.00 + float64(outputTokens)/1_000_000*15.00
			}

			result := BenchmarkResult{
				Model:           ep.Name,
				Provider:        ep.Provider,
				Latency:         latency,
				InputTokens:     inputTokens,
				TokensGenerated: outputTokens,
				CostUSD:         cost,
			}
			results = append(results, result)

			t.Logf("Model: %s", ep.Name)
			t.Logf("  Tokens: %d in / %d out", inputTokens, outputTokens)
			t.Logf("  Latency: %v", latency.Round(time.Millisecond))
			t.Logf("  Cost: $%.6f", cost)
		})
	}

	// Output results as JSON if requested
	if os.Getenv("LLM_BENCHMARK_JSON") == "1" {
		data, _ := json.MarshalIndent(results, "", "  ")
		t.Logf("\nJSON Results:\n%s", string(data))
	}
}

// TestStreamingComparison compares streaming performance between providers.
func TestStreamingComparison(t *testing.T) {
	if os.Getenv("LLM_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test (set LLM_INTEGRATION_TEST=1 to run)")
	}

	endpoints := DefaultEndpoints()
	prompt := "Count from 1 to 5, with each number on a new line."

	for _, ep := range endpoints {
		if !ep.Enabled {
			continue
		}

		t.Run(ep.Name+"_streaming", func(t *testing.T) {
			provider, err := createTestProvider(ep)
			if err != nil {
				t.Skipf("Could not create provider: %v", err)
				return
			}

			// Skip ping for Claude
			if ep.Provider != "claude" {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := provider.Ping(ctx); err != nil {
					cancel()
					t.Skipf("Server not available: %v", err)
					return
				}
				cancel()
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			start := time.Now()
			var timeToFirstToken time.Duration
			var response strings.Builder

			ch, err := provider.ChatCompletionStream(ctx, ChatRequest{
				Model: ep.Model,
				Messages: []Message{
					{Role: "user", Content: prompt},
				},
				MaxTokens:   64,
				Temperature: 0,
			})
			if err != nil {
				t.Fatalf("Stream failed: %v", err)
			}

			for event := range ch {
				if event.Error != nil {
					t.Fatalf("Stream error: %v", event.Error)
				}
				if event.Delta != "" && timeToFirstToken == 0 {
					timeToFirstToken = time.Since(start)
				}
				response.WriteString(event.Delta)
			}

			totalTime := time.Since(start)

			t.Logf("Provider: %s (%s)", ep.Name, ep.Provider)
			t.Logf("  Time to first token: %v", timeToFirstToken.Round(time.Millisecond))
			t.Logf("  Total time: %v", totalTime.Round(time.Millisecond))
			t.Logf("  Response length: %d chars", response.Len())
		})
	}
}
