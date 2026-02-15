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
		Short: "Perplexity AI search scraper & API client",
		Long: `Search Perplexity AI and extract structured results.

Supports four modes:
  SSE search (anonymous, scrapes web UI)
  Labs search (Socket.IO, anonymous, open-source models)
  API search (official REST API, requires API key)
  Pro search (requires account registration)

Multi-account support: register multiple accounts and rotate them.
All data and errors stored in DuckDB.

Data: $HOME/data/perplexity/

Examples:
  search perplexity search "go webframework"
  search perplexity search "quantum computing" --stream
  search perplexity labs "explain golang channels" --model sonar-pro
  search perplexity api "AI trends 2026" --model sonar-pro
  search perplexity search "machine learning" --pro
  search perplexity register
  search perplexity register --count 3
  search perplexity accounts
  search perplexity accounts add-key --key "pplx-xxx" --name "prod"
  search perplexity errors
  search perplexity info`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newPerplexitySearch())
	cmd.AddCommand(newPerplexityLabs())
	cmd.AddCommand(newPerplexityAPI())
	cmd.AddCommand(newPerplexityRegister())
	cmd.AddCommand(newPerplexityAccounts())
	cmd.AddCommand(newPerplexityErrors())
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

			// Open DB for error logging and account rotation
			db, dbErr := perplexity.OpenDB(cfg.DBPath())
			if dbErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not open DB: %v\n", dbErr)
			}
			if db != nil {
				defer db.Close()
			}

			// For pro modes, try account rotation
			if pro || reasoning || deep {
				if db != nil {
					am := perplexity.NewAccountManager(db)
					account, err := am.NextAccount()
					if err != nil {
						// Fallback to session file
						if loadErr := client.LoadSession(); loadErr != nil {
							return fmt.Errorf("pro mode requires a session; run 'search perplexity register' first: %w", loadErr)
						}
					} else {
						if err := am.LoadAccountSession(client, account); err != nil {
							// Fallback to session file
							if loadErr := client.LoadSession(); loadErr != nil {
								return fmt.Errorf("could not load account %d or session file: %w", account.ID, loadErr)
							}
						}
					}
				} else {
					if err := client.LoadSession(); err != nil {
						return fmt.Errorf("pro mode requires a session; run 'search perplexity register' first: %w", err)
					}
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

			start := time.Now()
			var result *perplexity.SearchResult
			if stream {
				result, err = client.SearchStream(ctx, query, opts, func(data map[string]any) {
					if answer, ok := data["answer"].(string); ok && answer != "" {
						fmt.Print("\r" + answer[:min(80, len(answer))] + "...")
					}
				})
				fmt.Println()
			} else {
				result, err = client.Search(ctx, query, opts)
			}
			if err != nil {
				// Log error to DB
				if db != nil {
					db.LogError(&perplexity.ErrorLog{
						Source:    "sse",
						Operation: "search",
						Query:    query,
						ErrorType: classifyError(err),
						ErrorMsg: err.Error(),
					})
				}
				return fmt.Errorf("search: %w", err)
			}

			result.DurationMs = time.Since(start).Milliseconds()

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Print(perplexity.FormatAnswer(result))

			// Save to DB
			if db != nil {
				if err := db.SaveSearch(result); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not save search: %v\n", err)
				} else {
					count, _ := db.Count()
					fmt.Printf("\nSaved to %s (%d searches total)\n", cfg.DBPath(), count)
				}
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

			cfg := perplexity.DefaultConfig()
			db, _ := perplexity.OpenDB(cfg.DBPath())
			if db != nil {
				defer db.Close()
			}

			fmt.Printf("Connecting to Perplexity Labs...\n")

			start := time.Now()
			lc, err := perplexity.NewLabsClient(ctx)
			if err != nil {
				if db != nil {
					db.LogError(&perplexity.ErrorLog{
						Source:    "labs",
						Operation: "connect",
						Query:    query,
						ErrorType: classifyError(err),
						ErrorMsg: err.Error(),
					})
				}
				return fmt.Errorf("connect labs: %w", err)
			}
			defer lc.Close()

			fmt.Printf("Querying (%s): %s\n\n", model, query)

			result, err := lc.Ask(ctx, query, model)
			if err != nil {
				if db != nil {
					db.LogError(&perplexity.ErrorLog{
						Source:    "labs",
						Operation: "labs_query",
						Query:    query,
						ErrorType: classifyError(err),
						ErrorMsg: err.Error(),
					})
				}
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
			if db != nil {
				sr := &perplexity.SearchResult{
					Query:      query,
					Answer:     result.Output,
					Mode:       "labs",
					Model:      result.Model,
					Source:     "labs",
					SearchedAt: time.Now(),
					DurationMs: time.Since(start).Milliseconds(),
				}
				if err := db.SaveSearch(sr); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not save: %v\n", err)
				} else {
					count, _ := db.Count()
					fmt.Printf("\nSaved to %s (%d searches total)\n", cfg.DBPath(), count)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&model, "model", perplexity.ModelR1, "Labs model to use")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output raw JSON")

	return cmd
}

func newPerplexityAPI() *cobra.Command {
	var (
		model      string
		mode       string
		stream     bool
		jsonOut    bool
		ctxSize    string
		recency    string
		domains    string
		images     bool
		related    bool
		systemMsg  string
		maxTokens  int
		noSearch   bool
		searchOnly bool
	)

	cmd := &cobra.Command{
		Use:   "api <query>",
		Short: "Search via official Perplexity API",
		Long: `Search using the official Perplexity REST API (requires API key).

Available models:
  sonar (default, lightweight)
  sonar-pro (advanced, 2x more results)
  sonar-reasoning-pro (chain-of-thought reasoning)
  sonar-deep-research (multi-step research)

Add API keys: search perplexity accounts add-key --key "pplx-xxx"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			cfg := perplexity.DefaultConfig()
			db, err := perplexity.OpenDB(cfg.DBPath())
			if err != nil {
				return fmt.Errorf("open DB: %w", err)
			}
			defer db.Close()

			am := perplexity.NewAccountManager(db)
			apiKey, err := am.NextAPIKey()
			if err != nil {
				return fmt.Errorf("no API key: %w", err)
			}

			client := perplexity.NewAPIClient(apiKey.Key, apiKey.ID)

			// Handle search-only mode
			if searchOnly {
				return doSearchOnly(cmd.Context(), client, db, apiKey, query, mode, recency, domains)
			}

			// Build chat request
			req := &perplexity.ChatRequest{
				Model:    model,
				Messages: []perplexity.ChatMessage{},
			}

			if systemMsg != "" {
				req.Messages = append(req.Messages, perplexity.ChatMessage{
					Role:    "system",
					Content: systemMsg,
				})
			}
			req.Messages = append(req.Messages, perplexity.ChatMessage{
				Role:    "user",
				Content: query,
			})

			if maxTokens > 0 {
				req.MaxTokens = maxTokens
			}
			if mode != "" {
				req.SearchMode = mode
			}
			if recency != "" {
				req.SearchRecencyFilter = recency
			}
			if domains != "" {
				req.SearchDomainFilter = strings.Split(domains, ",")
			}
			if images {
				req.ReturnImages = true
			}
			if related {
				req.ReturnRelated = true
			}
			if noSearch {
				req.DisableSearch = true
			}
			if ctxSize != "" {
				req.WebSearchOptions = &perplexity.WebSearchOptions{
					SearchContextSize: ctxSize,
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			if !jsonOut {
				fmt.Printf("API query (%s): %s\n\n", model, query)
			}

			start := time.Now()
			var resp *perplexity.ChatResponse

			if stream {
				resp, err = client.ChatStream(ctx, req, func(content string) {
					if !jsonOut {
						preview := content
						if len(preview) > 80 {
							preview = preview[:80]
						}
						fmt.Print("\r" + preview + "...")
					}
				})
				if !jsonOut {
					fmt.Println()
				}
			} else {
				resp, err = client.Chat(ctx, req)
			}

			if err != nil {
				// Detect error type and update key status
				errType := classifyError(err)
				if apiErr, ok := err.(*perplexity.APIError); ok {
					if apiErr.IsAuth() {
						am.MarkKeyFailed(apiKey.ID, err.Error())
						errType = perplexity.ErrAuth
					} else if apiErr.IsRateLimit() {
						am.MarkKeyRateLimited(apiKey.ID, err.Error())
						errType = perplexity.ErrRateLimit
					}
					db.LogError(&perplexity.ErrorLog{
						APIKeyID:     apiKey.ID,
						Source:       "api",
						Operation:    "api_chat",
						Query:        query,
						ErrorType:    errType,
						ErrorMsg:     err.Error(),
						HTTPStatus:   apiErr.StatusCode,
						ResponseBody: apiErr.Body,
					})
				} else {
					db.LogError(&perplexity.ErrorLog{
						APIKeyID:  apiKey.ID,
						Source:    "api",
						Operation: "api_chat",
						Query:    query,
						ErrorType: errType,
						ErrorMsg: err.Error(),
					})
				}
				return fmt.Errorf("api: %w", err)
			}

			durationMs := time.Since(start).Milliseconds()

			// Convert to SearchResult for storage
			result := client.ToSearchResult(resp, query, durationMs)
			result.Mode = model

			// Track API key usage
			if resp.Usage != nil {
				am.RecordKeyUsage(apiKey.ID, resp.Usage.TotalTokens)
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			// Print formatted result
			fmt.Print(perplexity.FormatAnswer(result))

			if resp.Usage != nil {
				fmt.Printf("\nTokens: %d (prompt: %d, completion: %d)\n",
					resp.Usage.TotalTokens, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
			}
			fmt.Printf("Duration: %dms\n", durationMs)

			// Save to DB
			if err := db.SaveSearch(result); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save: %v\n", err)
			} else {
				count, _ := db.Count()
				fmt.Printf("Saved (%d total)\n", count)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&model, "model", perplexity.APISonar, "API model")
	cmd.Flags().StringVar(&mode, "mode", "", "Search mode: web, academic, sec")
	cmd.Flags().BoolVar(&stream, "stream", false, "Stream output")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output raw JSON")
	cmd.Flags().StringVar(&ctxSize, "context", "", "Search context size: low, medium, high")
	cmd.Flags().StringVar(&recency, "recency", "", "Recency filter: hour, day, week, month, year")
	cmd.Flags().StringVar(&domains, "domains", "", "Domain filter (comma-separated, prefix - to exclude)")
	cmd.Flags().BoolVar(&images, "images", false, "Return images")
	cmd.Flags().BoolVar(&related, "related", false, "Return related questions")
	cmd.Flags().StringVar(&systemMsg, "system", "", "System prompt")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 0, "Max output tokens")
	cmd.Flags().BoolVar(&noSearch, "no-search", false, "Disable web search (pure LLM)")
	cmd.Flags().BoolVar(&searchOnly, "search-only", false, "Search API only (raw results, no LLM)")

	return cmd
}

func doSearchOnly(ctx context.Context, client *perplexity.APIClient, db *perplexity.DB, apiKey *perplexity.APIKey, query, mode, recency, domains string) error {
	req := &perplexity.SearchAPIRequest{
		Query: query,
	}
	if mode != "" {
		req.SearchMode = mode
	}
	if recency != "" {
		req.SearchRecencyFilter = recency
	}
	if domains != "" {
		req.SearchDomainFilter = strings.Split(domains, ",")
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	start := time.Now()
	resp, err := client.Search(ctx, req)
	if err != nil {
		db.LogError(&perplexity.ErrorLog{
			APIKeyID:  apiKey.ID,
			Source:    "api",
			Operation: "api_search",
			Query:    query,
			ErrorType: classifyError(err),
			ErrorMsg: err.Error(),
		})
		return fmt.Errorf("search: %w", err)
	}

	durationMs := time.Since(start).Milliseconds()

	fmt.Printf("Search results for: %s (%dms)\n\n", query, durationMs)
	for i, r := range resp.Results {
		fmt.Printf("  [%d] %s\n      %s\n", i+1, r.Title, r.URL)
		if r.Snippet != "" {
			snippet := r.Snippet
			if len(snippet) > 120 {
				snippet = snippet[:120] + "..."
			}
			fmt.Printf("      %s\n", snippet)
		}
		fmt.Println()
	}

	// Save to DB
	sr := &perplexity.SearchResult{
		Query:      query,
		Source:     "api",
		Mode:       "search",
		Model:      "search-api",
		SearchedAt: time.Now(),
		APIKeyID:   apiKey.ID,
		DurationMs: durationMs,
	}
	for _, r := range resp.Results {
		sr.WebResults = append(sr.WebResults, perplexity.WebResult{
			Name:    r.Title,
			URL:     r.URL,
			Snippet: r.Snippet,
		})
	}
	db.SaveSearch(sr)

	return nil
}

func newPerplexityRegister() *cobra.Command {
	var (
		xsrf    string
		laravel string
		count   int
	)

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register new Perplexity account(s) via emailnator",
		Long: `Register new Perplexity account(s) using a disposable email.

Fully automated by default - fetches emailnator cookies automatically.
Optionally provide cookies manually with --xsrf and --laravel.

Use --count N to register multiple accounts.

Example:
  search perplexity register
  search perplexity register --count 3
  search perplexity register --xsrf "TOKEN" --laravel "SESSION"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := perplexity.DefaultConfig()
			db, err := perplexity.OpenDB(cfg.DBPath())
			if err != nil {
				return fmt.Errorf("open DB: %w", err)
			}
			defer db.Close()

			manualCookies := xsrf != "" && laravel != ""

			for i := 0; i < count; i++ {
				if count > 1 {
					fmt.Printf("\n--- Registering account %d/%d ---\n", i+1, count)
				}

				client, err := perplexity.NewClient(cfg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

				if manualCookies {
					cookies := perplexity.EmailnatorCookies{
						XSRFToken:      xsrf,
						LaravelSession: laravel,
					}
					err = client.RegisterWithDB(ctx, cookies, db)
				} else {
					err = client.AutoRegister(ctx, db)
				}
				cancel()

				if err != nil {
					db.LogError(&perplexity.ErrorLog{
						Source:    "register",
						Operation: "register",
						ErrorType: classifyError(err),
						ErrorMsg: err.Error(),
					})
					fmt.Fprintf(os.Stderr, "Registration %d failed: %v\n", i+1, err)
					continue
				}
			}

			// Show account summary
			totalAccounts, activeAccounts, _ := db.CountAccounts()
			fmt.Printf("\nAccounts: %d total, %d active\n", totalAccounts, activeAccounts)

			return nil
		},
	}

	cmd.Flags().StringVar(&xsrf, "xsrf", "", "Emailnator XSRF-TOKEN cookie (optional, auto-fetched if omitted)")
	cmd.Flags().StringVar(&laravel, "laravel", "", "Emailnator laravel_session cookie (optional, auto-fetched if omitted)")
	cmd.Flags().IntVar(&count, "count", 1, "Number of accounts to register")

	return cmd
}

func newPerplexityAccounts() *cobra.Command {
	var status string

	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Manage Perplexity accounts and API keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := perplexity.DefaultConfig()
			db, err := perplexity.OpenDB(cfg.DBPath())
			if err != nil {
				return fmt.Errorf("open DB: %w", err)
			}
			defer db.Close()

			// List accounts
			accounts, err := db.ListAccounts(status)
			if err != nil {
				return fmt.Errorf("list accounts: %w", err)
			}

			fmt.Printf("Scraped Accounts:\n")
			if len(accounts) == 0 {
				fmt.Printf("  (none)\n")
			}
			for _, a := range accounts {
				statusIcon := "+"
				switch a.Status {
				case perplexity.AccountExhausted:
					statusIcon = "~"
				case perplexity.AccountFailed, perplexity.AccountBanned:
					statusIcon = "x"
				}
				fmt.Printf("  [%s] #%d %s  status=%s  pro=%d  used=%d  last=%s\n",
					statusIcon, a.ID, a.Email, a.Status, a.ProQueries, a.UseCount,
					a.LastUsedAt.Format("2006-01-02 15:04"))
				if a.ErrorMsg != "" {
					errPreview := a.ErrorMsg
					if len(errPreview) > 60 {
						errPreview = errPreview[:60] + "..."
					}
					fmt.Printf("       error: %s\n", errPreview)
				}
			}

			// List API keys
			keys, err := db.ListAPIKeys()
			if err != nil {
				return fmt.Errorf("list api keys: %w", err)
			}

			fmt.Printf("\nAPI Keys:\n")
			if len(keys) == 0 {
				fmt.Printf("  (none)\n")
			}
			for _, k := range keys {
				statusIcon := "+"
				switch k.Status {
				case perplexity.KeyRateLimited:
					statusIcon = "~"
				case perplexity.KeyInvalid, perplexity.KeyExhausted:
					statusIcon = "x"
				}
				keyPreview := k.Key
				if len(keyPreview) > 12 {
					keyPreview = keyPreview[:8] + "..." + keyPreview[len(keyPreview)-4:]
				}
				name := k.Name
				if name == "" {
					name = "(unnamed)"
				}
				fmt.Printf("  [%s] #%d %s  %s  status=%s  used=%d  tokens=%d\n",
					statusIcon, k.ID, name, keyPreview, k.Status, k.UseCount, k.TotalTokens)
				if k.ErrorMsg != "" {
					errPreview := k.ErrorMsg
					if len(errPreview) > 60 {
						errPreview = errPreview[:60] + "..."
					}
					fmt.Printf("       error: %s\n", errPreview)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter accounts by status (active, exhausted, failed, banned)")

	// Subcommands
	cmd.AddCommand(newPerplexityAddKey())
	cmd.AddCommand(newPerplexityRemoveAccount())

	return cmd
}

func newPerplexityAddKey() *cobra.Command {
	var (
		key  string
		name string
	)

	cmd := &cobra.Command{
		Use:   "add-key",
		Short: "Add a Perplexity API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if key == "" {
				return fmt.Errorf("--key is required")
			}

			cfg := perplexity.DefaultConfig()
			db, err := perplexity.OpenDB(cfg.DBPath())
			if err != nil {
				return fmt.Errorf("open DB: %w", err)
			}
			defer db.Close()

			am := perplexity.NewAccountManager(db)
			if err := am.AddAPIKey(key, name); err != nil {
				return fmt.Errorf("add key: %w", err)
			}

			fmt.Printf("API key added: %s\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&key, "key", "", "API key (pplx-xxx)")
	cmd.Flags().StringVar(&name, "name", "", "Friendly label")

	return cmd
}

func newPerplexityRemoveAccount() *cobra.Command {
	var (
		accountID int
		keyID     int
	)

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove an account or API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if accountID == 0 && keyID == 0 {
				return fmt.Errorf("specify --id (account) or --key-id (API key)")
			}

			cfg := perplexity.DefaultConfig()
			db, err := perplexity.OpenDB(cfg.DBPath())
			if err != nil {
				return fmt.Errorf("open DB: %w", err)
			}
			defer db.Close()

			if accountID > 0 {
				if err := db.DeleteAccount(accountID); err != nil {
					return fmt.Errorf("delete account: %w", err)
				}
				fmt.Printf("Account #%d removed\n", accountID)
			}

			if keyID > 0 {
				if err := db.DeleteAPIKey(keyID); err != nil {
					return fmt.Errorf("delete key: %w", err)
				}
				fmt.Printf("API key #%d removed\n", keyID)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&accountID, "id", 0, "Account ID to remove")
	cmd.Flags().IntVar(&keyID, "key-id", 0, "API key ID to remove")

	return cmd
}

func newPerplexityErrors() *cobra.Command {
	var (
		limit   int
		source  string
		errType string
	)

	cmd := &cobra.Command{
		Use:   "errors",
		Short: "Show recent errors",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := perplexity.DefaultConfig()
			db, err := perplexity.OpenDB(cfg.DBPath())
			if err != nil {
				return fmt.Errorf("open DB: %w", err)
			}
			defer db.Close()

			errors, err := db.RecentErrors(limit, source, errType)
			if err != nil {
				return fmt.Errorf("list errors: %w", err)
			}

			totalErrors, _ := db.CountErrors()
			fmt.Printf("Errors: %d total (showing last %d)\n\n", totalErrors, limit)

			if len(errors) == 0 {
				fmt.Println("  No errors found")
				return nil
			}

			for _, e := range errors {
				fmt.Printf("  [%s] %s/%s  type=%s\n",
					e.CreatedAt.Format("2006-01-02 15:04:05"),
					e.Source, e.Operation, e.ErrorType)
				if e.Query != "" {
					queryPreview := e.Query
					if len(queryPreview) > 60 {
						queryPreview = queryPreview[:60] + "..."
					}
					fmt.Printf("    query: %s\n", queryPreview)
				}
				errPreview := e.ErrorMsg
				if len(errPreview) > 100 {
					errPreview = errPreview[:100] + "..."
				}
				fmt.Printf("    error: %s\n", errPreview)
				if e.HTTPStatus > 0 {
					fmt.Printf("    http: %d\n", e.HTTPStatus)
				}
				if e.AccountID > 0 {
					fmt.Printf("    account: #%d\n", e.AccountID)
				}
				if e.APIKeyID > 0 {
					fmt.Printf("    api_key: #%d\n", e.APIKeyID)
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Number of errors to show")
	cmd.Flags().StringVar(&source, "source", "", "Filter by source: sse, labs, api, register")
	cmd.Flags().StringVar(&errType, "type", "", "Filter by error type")

	return cmd
}

func newPerplexityInfo() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show stored search statistics and account summary",
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

			// Account summary
			totalAccounts, activeAccounts, _ := db.CountAccounts()
			totalKeys, activeKeys, _ := db.CountAPIKeys()
			totalErrors, _ := db.CountErrors()

			fmt.Printf("\nAccounts: %d total, %d active\n", totalAccounts, activeAccounts)
			fmt.Printf("API keys: %d total, %d active\n", totalKeys, activeKeys)
			fmt.Printf("Errors: %d logged\n", totalErrors)

			// Show session info (legacy)
			client, err := perplexity.NewClient(cfg)
			if err == nil {
				if err := client.LoadSession(); err == nil {
					fmt.Printf("\nLegacy session: active\n")
					fmt.Printf("Pro queries remaining: %d\n", client.CopilotQueries())
				}
			}

			return nil
		},
	}
}

// classifyError determines the error type from an error.
func classifyError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "rate limit") || strings.Contains(msg, "429"):
		return perplexity.ErrRateLimit
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
		return perplexity.ErrTimeout
	case strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "auth"):
		return perplexity.ErrAuth
	case strings.Contains(msg, "cloudflare") || strings.Contains(msg, "cf_clearance"):
		return perplexity.ErrCloudflare
	case strings.Contains(msg, "parse") || strings.Contains(msg, "unmarshal") || strings.Contains(msg, "json"):
		return perplexity.ErrParse
	default:
		return perplexity.ErrHTTP
	}
}
