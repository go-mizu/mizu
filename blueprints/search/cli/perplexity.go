package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler/perplexity"
	"github.com/spf13/cobra"
)

// NewPerplexity creates the perplexity command with subcommands.
func NewPerplexity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perplexity",
		Short: "Perplexity AI search scraper",
		Long: `Search Perplexity AI and extract structured results.

Supports three modes:
  SSE search (anonymous, no account needed)
  Labs search (Socket.IO, anonymous, open-source models)
  Pro search (requires account registration)

Data: $HOME/data/perplexity/

Examples:
  search perplexity search "go webframework"
  search perplexity search "quantum computing" --stream
  search perplexity search "machine learning" --json
  search perplexity labs "explain golang channels" --model sonar-pro
  search perplexity search "AI trends 2026" --pro
  search perplexity register
  search perplexity info`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newPerplexitySearch())
	cmd.AddCommand(newPerplexityLabs())
	cmd.AddCommand(newPerplexityRegister())
	cmd.AddCommand(newPerplexityInfo())

	return cmd
}

func newPerplexitySearch() *cobra.Command {
	var (
		pro       bool
		reasoning bool
		deep      bool
		sources   string
		language  string
		followUp  string
		stream    bool
		jsonOut   bool
		incognito bool
		model     string
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Perplexity AI via SSE",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			cfg := perplexity.DefaultConfig()
			client, err := perplexity.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			// Try loading existing session for pro modes
			if pro || reasoning || deep {
				if err := client.LoadSession(); err != nil {
					return fmt.Errorf("pro mode requires a session; run 'search perplexity register' first: %w", err)
				}
			}

			opts := perplexity.DefaultSearchOptions()
			if pro {
				opts.Mode = perplexity.ModePro
			} else if reasoning {
				opts.Mode = perplexity.ModeReasoning
			} else if deep {
				opts.Mode = perplexity.ModeDeepResearch
			}

			if model != "" {
				opts.Model = model
			}

			if sources != "" {
				opts.Sources = strings.Split(sources, ",")
			}
			if language != "" {
				opts.Language = language
			}
			if followUp != "" {
				opts.FollowUpUUID = followUp
			}
			opts.Incognito = incognito

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			if !jsonOut {
				fmt.Printf("Searching: %s\n\n", query)
			}

			var result *perplexity.SearchResult
			if stream {
				result, err = client.SearchStream(ctx, query, opts, func(data map[string]any) {
					// Print streaming indicator
					if answer, ok := data["answer"].(string); ok && answer != "" {
						fmt.Print("\r" + answer[:min(80, len(answer))] + "...")
					}
				})
				fmt.Println() // newline after stream
			} else {
				result, err = client.Search(ctx, query, opts)
			}
			if err != nil {
				return fmt.Errorf("search: %w", err)
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			// Print formatted result
			fmt.Print(perplexity.FormatAnswer(result))

			// Save to DB
			db, err := perplexity.OpenDB(cfg.DBPath())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not open DB: %v\n", err)
				return nil
			}
			defer db.Close()

			if err := db.SaveSearch(result); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save search: %v\n", err)
			} else {
				count, _ := db.Count()
				fmt.Printf("\nSaved to %s (%d searches total)\n", cfg.DBPath(), count)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&pro, "pro", false, "Use pro search mode (requires account)")
	cmd.Flags().BoolVar(&reasoning, "reasoning", false, "Use reasoning mode (requires account)")
	cmd.Flags().BoolVar(&deep, "deep", false, "Use deep research mode (requires account)")
	cmd.Flags().StringVar(&model, "model", "", "Specific model to use")
	cmd.Flags().StringVar(&sources, "sources", "web", "Comma-separated: web,scholar,social")
	cmd.Flags().StringVar(&language, "language", "en-US", "Language code")
	cmd.Flags().StringVar(&followUp, "follow-up", "", "Backend UUID for follow-up query")
	cmd.Flags().BoolVar(&stream, "stream", false, "Stream output to terminal")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output raw JSON")
	cmd.Flags().BoolVar(&incognito, "incognito", false, "Incognito mode")

	return cmd
}

func newPerplexityLabs() *cobra.Command {
	var (
		model   string
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "labs <query>",
		Short: "Search via Perplexity Labs (Socket.IO, anonymous)",
		Long: `Search using Perplexity Labs with open-source models.
No account required. Available models:
  r1-1776 (default)
  sonar-pro
  sonar
  sonar-reasoning-pro
  sonar-reasoning`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			fmt.Printf("Connecting to Perplexity Labs...\n")

			lc, err := perplexity.NewLabsClient(ctx)
			if err != nil {
				return fmt.Errorf("connect labs: %w", err)
			}
			defer lc.Close()

			fmt.Printf("Querying (%s): %s\n\n", model, query)

			result, err := lc.Ask(ctx, query, model)
			if err != nil {
				return fmt.Errorf("labs query: %w", err)
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Printf("Model: %s\n\n", result.Model)
			fmt.Println(result.Output)

			// Save to DB
			cfg := perplexity.DefaultConfig()
			db, err := perplexity.OpenDB(cfg.DBPath())
			if err != nil {
				return nil
			}
			defer db.Close()

			sr := &perplexity.SearchResult{
				Query:      query,
				Answer:     result.Output,
				Mode:       "labs",
				Model:      result.Model,
				Source:     "labs",
				SearchedAt: time.Now(),
			}
			if err := db.SaveSearch(sr); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save: %v\n", err)
			} else {
				count, _ := db.Count()
				fmt.Printf("\nSaved to %s (%d searches total)\n", cfg.DBPath(), count)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&model, "model", perplexity.ModelR1, "Labs model to use")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output raw JSON")

	return cmd
}

func newPerplexityRegister() *cobra.Command {
	var xsrf, laravel string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new Perplexity account via emailnator",
		Long: `Register a new Perplexity account using a disposable email.

Requires emailnator.com cookies. Get them from your browser:
  1. Go to https://www.emailnator.com/
  2. Open DevTools → Application → Cookies
  3. Copy XSRF-TOKEN and laravel_session values

Example:
  search perplexity register --xsrf "TOKEN" --laravel "SESSION"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if xsrf == "" || laravel == "" {
				return fmt.Errorf("both --xsrf and --laravel cookies are required")
			}

			cfg := perplexity.DefaultConfig()
			client, err := perplexity.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			cookies := perplexity.EmailnatorCookies{
				XSRFToken:      xsrf,
				LaravelSession: laravel,
			}

			return client.Register(ctx, cookies)
		},
	}

	cmd.Flags().StringVar(&xsrf, "xsrf", "", "Emailnator XSRF-TOKEN cookie")
	cmd.Flags().StringVar(&laravel, "laravel", "", "Emailnator laravel_session cookie")

	return cmd
}

func newPerplexityInfo() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show stored search statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := perplexity.DefaultConfig()

			db, err := perplexity.OpenDB(cfg.DBPath())
			if err != nil {
				return fmt.Errorf("open DB: %w", err)
			}
			defer db.Close()

			count, _ := db.Count()
			fmt.Printf("Database: %s\n", cfg.DBPath())
			fmt.Printf("Total searches: %d\n", count)

			// Show recent searches
			recent, err := db.RecentSearches(5)
			if err != nil {
				return nil
			}

			if len(recent) > 0 {
				fmt.Printf("\nRecent searches:\n")
				for _, r := range recent {
					answerPreview := r.Answer
					if len(answerPreview) > 80 {
						answerPreview = answerPreview[:80] + "..."
					}
					fmt.Printf("  [%s] %s (%s/%s)\n    %s\n",
						r.SearchedAt.Format("2006-01-02 15:04"),
						r.Query, r.Source, r.Mode,
						answerPreview,
					)
				}
			}

			// Show session info
			client, err := perplexity.NewClient(cfg)
			if err == nil {
				if err := client.LoadSession(); err == nil {
					fmt.Printf("\nSession: active\n")
					fmt.Printf("Pro queries remaining: %d\n", client.CopilotQueries())
				} else {
					fmt.Printf("\nSession: none (use 'search perplexity register' to create)\n")
				}
			}

			return nil
		},
	}
}
