// Package main provides a CLI tool for working with LLM backends.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm/claude"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/llm/claude"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/llm/llamacpp"
	"github.com/spf13/cobra"
)

var (
	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	userStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))

	assistantStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46"))

	// Flags
	provider    string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	timeout     int
	stream      bool
	showUsage   bool

	// Styles for token usage
	usageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("238")).
			PaddingLeft(1)

	costStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "llm",
		Short: "CLI tool for working with LLM backends",
		Long:  titleStyle.Render("LLM CLI") + "\n\nA command-line tool for interacting with various LLM backends.",
	}

	rootCmd.PersistentFlags().StringVarP(&provider, "provider", "P", "llamacpp", "LLM provider (llamacpp, claude)")
	rootCmd.PersistentFlags().StringVarP(&baseURL, "url", "u", "", "LLM server base URL (default depends on provider)")
	rootCmd.PersistentFlags().StringVarP(&model, "model", "m", "", "Model name (optional)")
	rootCmd.PersistentFlags().IntVarP(&maxTokens, "max-tokens", "n", 2048, "Maximum tokens to generate")
	rootCmd.PersistentFlags().Float64VarP(&temperature, "temperature", "t", 0.7, "Sampling temperature")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 120, "Request timeout in seconds")
	rootCmd.PersistentFlags().BoolVarP(&stream, "stream", "s", true, "Enable streaming output")
	rootCmd.PersistentFlags().BoolVar(&showUsage, "usage", true, "Show token usage after responses")

	rootCmd.AddCommand(chatCmd())
	rootCmd.AddCommand(askCmd())
	rootCmd.AddCommand(modelsCmd())
	rootCmd.AddCommand(embedCmd())
	rootCmd.AddCommand(benchCmd())
	rootCmd.AddCommand(pingCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, errorStyle.Render("Error: "+err.Error()))
		os.Exit(1)
	}
}

func getProvider() (llm.Provider, error) {
	// Set default URL based on provider
	url := baseURL
	if url == "" {
		switch provider {
		case "claude":
			url = "https://api.anthropic.com/v1"
		default:
			url = "http://localhost:8080"
		}
	}

	return llm.New(provider, llm.Config{
		BaseURL: url,
		Timeout: timeout,
	})
}

// displayUsage shows token usage and cost information.
func displayUsage(usage llm.Usage, providerName, modelName string) {
	if !showUsage {
		return
	}

	var lines []string

	// Token counts
	lines = append(lines, fmt.Sprintf("Tokens: %d in / %d out / %d total",
		usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))

	// Cache tokens (if present)
	if usage.CacheReadTokens > 0 || usage.CacheWriteTokens > 0 {
		lines = append(lines, fmt.Sprintf("Cache: %d read / %d write",
			usage.CacheReadTokens, usage.CacheWriteTokens))
	}

	// Cost for Claude
	if providerName == "claude" {
		cost := claude.CalculateCost(modelName, usage.PromptTokens, usage.CompletionTokens)
		if cost > 0 {
			lines = append(lines, costStyle.Render(fmt.Sprintf("Cost: $%.6f", cost)))
		}
	}

	// Display with usage style
	for _, line := range lines {
		fmt.Println(usageStyle.Render(line))
	}
}

func chatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chat",
		Short: "Interactive chat with the LLM",
		RunE: func(cmd *cobra.Command, args []string) error {
			llmProvider, err := getProvider()
			if err != nil {
				return err
			}

			// Determine display URL
			displayURL := baseURL
			if displayURL == "" {
				if provider == "claude" {
					displayURL = "api.anthropic.com"
				} else {
					displayURL = "localhost:8080"
				}
			}

			fmt.Println(titleStyle.Render("LLM Chat"))
			fmt.Println(infoStyle.Render(fmt.Sprintf("Provider: %s (%s)", provider, displayURL)))
			fmt.Println(infoStyle.Render("Type 'exit' or 'quit' to end the session"))
			fmt.Println()

			messages := []llm.Message{}
			reader := bufio.NewReader(os.Stdin)

			for {
				fmt.Print(userStyle.Render("You: "))
				input, err := reader.ReadString('\n')
				if err != nil {
					return err
				}

				input = strings.TrimSpace(input)
				if input == "" {
					continue
				}
				if input == "exit" || input == "quit" {
					fmt.Println(infoStyle.Render("Goodbye!"))
					return nil
				}
				if input == "/clear" {
					messages = []llm.Message{}
					fmt.Println(infoStyle.Render("Conversation cleared"))
					continue
				}

				messages = append(messages, llm.Message{Role: "user", Content: input})

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)

				fmt.Print(assistantStyle.Render("Assistant: "))

				if stream {
					ch, err := llmProvider.ChatCompletionStream(ctx, llm.ChatRequest{
						Model:       model,
						Messages:    messages,
						MaxTokens:   maxTokens,
						Temperature: temperature,
					})
					if err != nil {
						cancel()
						fmt.Println(errorStyle.Render(err.Error()))
						messages = messages[:len(messages)-1]
						continue
					}

					var response strings.Builder
					var lastUsage llm.Usage
					for event := range ch {
						if event.Error != nil {
							fmt.Println(errorStyle.Render(event.Error.Error()))
							break
						}
						fmt.Print(event.Delta)
						response.WriteString(event.Delta)
						// Capture final usage from stream
						if event.InputTokens > 0 || event.OutputTokens > 0 {
							lastUsage.PromptTokens = event.InputTokens
							lastUsage.CompletionTokens = event.OutputTokens
							lastUsage.TotalTokens = event.InputTokens + event.OutputTokens
						}
						if event.Usage != nil {
							lastUsage = *event.Usage
						}
					}
					fmt.Println()
					messages = append(messages, llm.Message{Role: "assistant", Content: response.String()})
					displayUsage(lastUsage, provider, model)
				} else {
					resp, err := llmProvider.ChatCompletion(ctx, llm.ChatRequest{
						Model:       model,
						Messages:    messages,
						MaxTokens:   maxTokens,
						Temperature: temperature,
					})
					if err != nil {
						cancel()
						fmt.Println(errorStyle.Render(err.Error()))
						messages = messages[:len(messages)-1]
						continue
					}

					if len(resp.Choices) > 0 {
						content := resp.Choices[0].Message.Content
						fmt.Println(content)
						messages = append(messages, llm.Message{Role: "assistant", Content: content})
					}
					displayUsage(resp.Usage, provider, model)
				}
				cancel()
				fmt.Println()
			}
		},
	}
}

func askCmd() *cobra.Command {
	var system string

	cmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask a single question",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			llmProvider, err := getProvider()
			if err != nil {
				return err
			}

			question := strings.Join(args, " ")
			messages := []llm.Message{}

			if system != "" {
				messages = append(messages, llm.Message{Role: "system", Content: system})
			}
			messages = append(messages, llm.Message{Role: "user", Content: question})

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
			defer cancel()

			if stream {
				ch, err := llmProvider.ChatCompletionStream(ctx, llm.ChatRequest{
					Model:       model,
					Messages:    messages,
					MaxTokens:   maxTokens,
					Temperature: temperature,
				})
				if err != nil {
					return err
				}

				var lastUsage llm.Usage
				for event := range ch {
					if event.Error != nil {
						return event.Error
					}
					fmt.Print(event.Delta)
					if event.InputTokens > 0 || event.OutputTokens > 0 {
						lastUsage.PromptTokens = event.InputTokens
						lastUsage.CompletionTokens = event.OutputTokens
						lastUsage.TotalTokens = event.InputTokens + event.OutputTokens
					}
					if event.Usage != nil {
						lastUsage = *event.Usage
					}
				}
				fmt.Println()
				displayUsage(lastUsage, provider, model)
			} else {
				resp, err := llmProvider.ChatCompletion(ctx, llm.ChatRequest{
					Model:       model,
					Messages:    messages,
					MaxTokens:   maxTokens,
					Temperature: temperature,
				})
				if err != nil {
					return err
				}

				if len(resp.Choices) > 0 {
					fmt.Println(resp.Choices[0].Message.Content)
				}
				displayUsage(resp.Usage, provider, model)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&system, "system", "", "System prompt")
	return cmd
}

func modelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List available models",
		RunE: func(cmd *cobra.Command, args []string) error {
			llmProvider, err := getProvider()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
			defer cancel()

			models, err := llmProvider.Models(ctx)
			if err != nil {
				return err
			}

			fmt.Println(titleStyle.Render("Available Models"))
			fmt.Println(infoStyle.Render(fmt.Sprintf("Provider: %s", provider)))
			fmt.Println()

			for _, m := range models {
				fmt.Printf("  %s %s\n",
					successStyle.Render("•"),
					m.ID,
				)
				if m.OwnedBy != "" {
					fmt.Printf("    %s\n", infoStyle.Render("Owner: "+m.OwnedBy))
				}
			}

			if len(models) == 0 {
				fmt.Println(infoStyle.Render("  No models found"))
			}

			return nil
		},
	}
}

func embedCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "embed [text...]",
		Short: "Generate embeddings for text",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			llmProvider, err := getProvider()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
			defer cancel()

			resp, err := llmProvider.Embedding(ctx, llm.EmbeddingRequest{
				Model: model,
				Input: args,
			})
			if err != nil {
				return err
			}

			if output == "json" {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			fmt.Println(titleStyle.Render("Embeddings"))
			fmt.Println()

			for i, data := range resp.Data {
				fmt.Printf("%s Input %d: %s\n",
					successStyle.Render("•"),
					i+1,
					infoStyle.Render(fmt.Sprintf("%d dimensions", len(data.Embedding))),
				)
				if output == "full" {
					fmt.Printf("  [%.4f, %.4f, %.4f, ... %.4f]\n",
						data.Embedding[0], data.Embedding[1], data.Embedding[2],
						data.Embedding[len(data.Embedding)-1])
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "summary", "Output format: summary, full, json")
	return cmd
}

func benchCmd() *cobra.Command {
	var iterations int
	var prompt string

	cmd := &cobra.Command{
		Use:   "bench",
		Short: "Benchmark LLM performance",
		RunE: func(cmd *cobra.Command, args []string) error {
			llmProvider, err := getProvider()
			if err != nil {
				return err
			}

			fmt.Println(titleStyle.Render("LLM Benchmark"))
			fmt.Println(infoStyle.Render(fmt.Sprintf("Provider: %s", provider)))
			fmt.Println(infoStyle.Render(fmt.Sprintf("Iterations: %d", iterations)))
			fmt.Println()

			// Ping test
			fmt.Print("Testing connectivity... ")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := llmProvider.Ping(ctx); err != nil {
				cancel()
				fmt.Println(errorStyle.Render("FAILED"))
				return err
			}
			cancel()
			fmt.Println(successStyle.Render("OK"))

			// Generation benchmark
			var totalTime time.Duration
			var totalTokens int
			var totalInputTokens int
			var totalCost float64

			messages := []llm.Message{
				{Role: "user", Content: prompt},
			}

			for i := 0; i < iterations; i++ {
				fmt.Printf("Iteration %d/%d... ", i+1, iterations)

				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
				start := time.Now()

				resp, err := llmProvider.ChatCompletion(ctx, llm.ChatRequest{
					Model:       model,
					Messages:    messages,
					MaxTokens:   maxTokens,
					Temperature: temperature,
				})
				cancel()

				if err != nil {
					fmt.Println(errorStyle.Render("FAILED: " + err.Error()))
					continue
				}

				elapsed := time.Since(start)
				totalTime += elapsed
				totalTokens += resp.Usage.CompletionTokens
				totalInputTokens += resp.Usage.PromptTokens

				// Calculate cost for Claude
				if provider == "claude" {
					totalCost += claude.CalculateCost(model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
				}

				fmt.Printf("%s (%d tokens in %v)\n",
					successStyle.Render("OK"),
					resp.Usage.CompletionTokens,
					elapsed.Round(time.Millisecond))
			}

			fmt.Println()
			fmt.Println(titleStyle.Render("Results"))
			fmt.Printf("  Total time:     %v\n", totalTime.Round(time.Millisecond))
			fmt.Printf("  Input tokens:   %d\n", totalInputTokens)
			fmt.Printf("  Output tokens:  %d\n", totalTokens)
			if iterations > 0 {
				avgTime := totalTime / time.Duration(iterations)
				fmt.Printf("  Avg time:       %v\n", avgTime.Round(time.Millisecond))
				if totalTokens > 0 {
					tokensPerSec := float64(totalTokens) / totalTime.Seconds()
					fmt.Printf("  Tokens/sec:     %.2f\n", tokensPerSec)
				}
			}
			if totalCost > 0 {
				fmt.Printf("  %s\n", costStyle.Render(fmt.Sprintf("Total cost:   $%.6f", totalCost)))
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&iterations, "iterations", "i", 3, "Number of iterations")
	cmd.Flags().StringVarP(&prompt, "prompt", "p", "Explain what a neural network is in one paragraph.", "Benchmark prompt")
	return cmd
}

func pingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Check if the LLM server is healthy",
		RunE: func(cmd *cobra.Command, args []string) error {
			llmProvider, err := getProvider()
			if err != nil {
				return err
			}

			fmt.Printf("Pinging %s... ", provider)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			start := time.Now()
			if err := llmProvider.Ping(ctx); err != nil {
				fmt.Println(errorStyle.Render("FAILED"))
				return err
			}

			elapsed := time.Since(start)
			fmt.Println(successStyle.Render(fmt.Sprintf("OK (%v)", elapsed.Round(time.Millisecond))))
			return nil
		},
	}
}
