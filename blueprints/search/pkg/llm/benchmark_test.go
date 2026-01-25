package llm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// ModelEndpoint represents a model server endpoint for testing.
type ModelEndpoint struct {
	Name    string
	URL     string
	Enabled bool
}

// DefaultEndpoints returns the standard model endpoints for comparison.
func DefaultEndpoints() []ModelEndpoint {
	return []ModelEndpoint{
		{Name: "gemma-270m-quick", URL: "http://localhost:8082", Enabled: true},
		{Name: "gemma-1b-deep", URL: "http://localhost:8083", Enabled: true},
		{Name: "gemma-4b-research", URL: "http://localhost:8084", Enabled: true},
		{Name: "gpt-oss-20b", URL: "http://localhost:8085", Enabled: true},
	}
}

// BenchmarkResult holds the results of a model benchmark.
type BenchmarkResult struct {
	Model           string
	URL             string
	Latency         time.Duration
	TokensGenerated int
	TokensPerSecond float64
	ResponseQuality float64 // 0-1 score based on response criteria
	Error           error
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
			provider, err := createTestProvider(ep.URL)
			if err != nil {
				t.Skipf("Could not create provider for %s: %v", ep.Name, err)
				return
			}

			// Check connectivity
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := provider.Ping(ctx); err != nil {
				cancel()
				t.Skipf("Server %s not available: %v", ep.Name, err)
				return
			}
			cancel()

			for _, p := range prompts {
				t.Run(p.name, func(t *testing.T) {
					result := benchmarkPrompt(provider, ep.Name, ep.URL, p.prompt, p.criteria)
					results[p.name] = append(results[p.name], result)

					if result.Error != nil {
						t.Errorf("Error: %v", result.Error)
						return
					}

					t.Logf("Model: %s, Latency: %v, Tokens/sec: %.2f, Quality: %.2f",
						result.Model, result.Latency, result.TokensPerSecond, result.ResponseQuality)
				})
			}
		})
	}

	// Print summary
	printComparisonSummary(t, results)
}

func createTestProvider(url string) (Provider, error) {
	// Try to find a registered provider
	names := Providers()
	if len(names) == 0 {
		return nil, fmt.Errorf("no providers registered")
	}

	return New(names[0], Config{
		BaseURL: url,
		Timeout: 120,
	})
}

func benchmarkPrompt(provider Provider, name, url, prompt string, criteria []string) BenchmarkResult {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	resp, err := provider.ChatCompletion(ctx, ChatRequest{
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   256,
		Temperature: 0.3,
	})

	result := BenchmarkResult{
		Model: name,
		URL:   url,
	}

	if err != nil {
		result.Error = err
		return result
	}

	result.Latency = time.Since(start)
	result.TokensGenerated = resp.Usage.CompletionTokens
	if result.Latency > 0 {
		result.TokensPerSecond = float64(result.TokensGenerated) / result.Latency.Seconds()
	}

	// Evaluate quality based on criteria
	if len(resp.Choices) > 0 && len(criteria) > 0 {
		content := strings.ToLower(resp.Choices[0].Message.Content)
		matches := 0
		for _, c := range criteria {
			if strings.Contains(content, strings.ToLower(c)) {
				matches++
			}
		}
		result.ResponseQuality = float64(matches) / float64(len(criteria))
	} else if len(resp.Choices) > 0 {
		// For creative prompts, just check we got a response
		result.ResponseQuality = 1.0
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
			provider, err := createTestProvider(ep.URL)
			if err != nil {
				t.Skipf("Could not create provider: %v", err)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := provider.Ping(ctx); err != nil {
				cancel()
				t.Skipf("Server not available: %v", err)
				return
			}
			cancel()

			var totalLatency time.Duration
			var totalTokens int

			for i := 0; i < iterations; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				start := time.Now()

				resp, err := provider.ChatCompletion(ctx, ChatRequest{
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

	// Only test embedding-capable endpoints
	endpoints := []ModelEndpoint{
		{Name: "gemma-270m-quick", URL: "http://localhost:8082", Enabled: true},
		{Name: "gemma-1b-deep", URL: "http://localhost:8083", Enabled: true},
		{Name: "gemma-4b-research", URL: "http://localhost:8084", Enabled: true},
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
			provider, err := createTestProvider(ep.URL)
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
