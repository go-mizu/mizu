package cli

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/amazon"
	"github.com/spf13/cobra"
)

// NewAmazon creates the amazon CLI command.
func NewAmazon() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "amazon",
		Short: "Scrape Amazon products, brands, reviews, Q&A, and more",
		Long: `Scrape public Amazon data into a local DuckDB database.

Supports products, brands, authors, categories, bestseller lists, reviews, and Q&A.
Data is stored in $HOME/data/amazon/amazon.duckdb.

Examples:
  search amazon product B08N5WRWNW         # Fetch a single product (Echo Dot)
  search amazon reviews B08N5WRWNW         # Fetch all reviews for a product
  search amazon bestsellers --category electronics
  search amazon search "wireless headphones" --max-results 50
  search amazon crawl --workers 2          # Bulk crawl the queue`,
	}
	cmd.AddCommand(newAmazonProduct())
	cmd.AddCommand(newAmazonBrand())
	cmd.AddCommand(newAmazonAuthor())
	cmd.AddCommand(newAmazonCategory())
	cmd.AddCommand(newAmazonSeller())
	cmd.AddCommand(newAmazonSearch())
	cmd.AddCommand(newAmazonBestsellers())
	cmd.AddCommand(newAmazonReviews())
	cmd.AddCommand(newAmazonQA())
	cmd.AddCommand(newAmazonSeed())
	cmd.AddCommand(newAmazonCrawl())
	cmd.AddCommand(newAmazonInfo())
	cmd.AddCommand(newAmazonJobs())
	cmd.AddCommand(newAmazonQueue())
	return cmd
}

// ── Shared flag helpers ───────────────────────────────────────────────────────

func addAmazonFlags(cmd *cobra.Command, dbPath, statePath *string, delay, maxPages *int) {
	cfg := amazon.DefaultConfig()
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to amazon.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
	cmd.Flags().IntVar(maxPages, "max-pages", cfg.MaxPages, "Max pages per entity (0 = unlimited)")
}

func openAmazonDBs(dbPath, statePath string, delay, maxPages int) (*amazon.DB, *amazon.State, *amazon.Client, error) {
	cfg := amazon.DefaultConfig()
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond
	cfg.MaxPages = maxPages
	db, err := amazon.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}
	stateDB, err := amazon.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("open state: %w", err)
	}
	client := amazon.NewClient(cfg)
	return db, stateDB, client, nil
}

// ── URL normalization helpers ─────────────────────────────────────────────────

func normalizeProductURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return amazon.BaseURL + "/dp/" + s
}

func normalizeSellerURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return amazon.BaseURL + "/sp?seller=" + s
}

func normalizeCategoryURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return amazon.BaseURL + "/b?node=" + s
}

// ── product ───────────────────────────────────────────────────────────────────

func newAmazonProduct() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "product <ASIN|url>",
		Short: "Fetch a single Amazon product",
		Args:  cobra.ExactArgs(1),
		Example: `  search amazon product B08N5WRWNW
  search amazon product https://www.amazon.com/dp/B08N5WRWNW`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			productURL := normalizeProductURL(args[0])
			fmt.Printf("Fetching %s ...\n", productURL)

			task := &amazon.ProductTask{
				URL:      productURL,
				Client:   client,
				DB:       db,
				StateDB:  stateDB,
				MaxPages: maxPages,
			}
			m, err := task.Run(cmd.Context(), func(s *amazon.ProductState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

// ── brand ─────────────────────────────────────────────────────────────────────

func newAmazonBrand() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "brand <slug|url>",
		Short: "Fetch an Amazon brand/store page",
		Args:  cobra.ExactArgs(1),
		Example: `  search amazon brand Apple
  search amazon brand https://www.amazon.com/stores/Apple`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			brandURL := args[0]
			if !strings.HasPrefix(brandURL, "http") {
				brandURL = amazon.BaseURL + "/stores/" + brandURL
			}
			fmt.Printf("Fetching %s ...\n", brandURL)

			task := &amazon.BrandTask{
				URL:     brandURL,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *amazon.BrandState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

// ── author ────────────────────────────────────────────────────────────────────

func newAmazonAuthor() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "author <slug|url>",
		Short: "Fetch an Amazon Author Central page",
		Args:  cobra.ExactArgs(1),
		Example: `  search amazon author stephenking
  search amazon author https://www.amazon.com/author/stephenking`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			authorURL := args[0]
			if !strings.HasPrefix(authorURL, "http") {
				authorURL = amazon.BaseURL + "/author/" + authorURL
			}
			fmt.Printf("Fetching %s ...\n", authorURL)

			task := &amazon.AuthorTask{
				URL:     authorURL,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *amazon.AuthorState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

// ── category ──────────────────────────────────────────────────────────────────

func newAmazonCategory() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "category <node_id|url>",
		Short: "Fetch an Amazon browse node / category page",
		Args:  cobra.ExactArgs(1),
		Example: `  search amazon category 172282
  search amazon category https://www.amazon.com/b?node=172282`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			categoryURL := normalizeCategoryURL(args[0])
			fmt.Printf("Fetching %s ...\n", categoryURL)

			task := &amazon.CategoryTask{
				URL:     categoryURL,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *amazon.CategoryState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

// ── seller ────────────────────────────────────────────────────────────────────

func newAmazonSeller() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "seller <seller_id|url>",
		Short: "Fetch an Amazon third-party seller profile",
		Args:  cobra.ExactArgs(1),
		Example: `  search amazon seller A2L77EE7U53NWQ
  search amazon seller https://www.amazon.com/sp?seller=A2L77EE7U53NWQ`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			sellerURL := normalizeSellerURL(args[0])
			fmt.Printf("Fetching %s ...\n", sellerURL)

			task := &amazon.SellerTask{
				URL:     sellerURL,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *amazon.SellerState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

// ── search ────────────────────────────────────────────────────────────────────

func newAmazonSearch() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages, maxResults, page int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Amazon and enqueue results",
		Args:  cobra.ExactArgs(1),
		Example: `  search amazon search "wireless headphones" --max-results 50
  search amazon search "kindle" --page 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			query := args[0]
			searchURL := amazon.BaseURL + "/s?k=" + url.QueryEscape(query) + "&page=" + fmt.Sprintf("%d", page)
			fmt.Printf("Searching: %s\n", searchURL)

			task := &amazon.SearchTask{
				URL:        searchURL,
				Query:      query,
				Page:       page,
				Client:     client,
				DB:         db,
				StateDB:    stateDB,
				MaxPages:   maxPages,
				MaxResults: maxResults,
			}
			m, err := task.Run(cmd.Context(), func(s *amazon.SearchState) {
				fmt.Printf("  [%s] %s (results=%d)\n", s.Status, s.URL, s.ResultsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: enqueued=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().IntVar(&maxResults, "max-results", 100, "Maximum results to enqueue")
	cmd.Flags().IntVar(&page, "page", 1, "Starting page number")
	return cmd
}

// ── bestsellers ───────────────────────────────────────────────────────────────

func newAmazonBestsellers() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int
	var category, listType string

	cmd := &cobra.Command{
		Use:   "bestsellers",
		Short: "Fetch an Amazon bestseller list",
		Args:  cobra.NoArgs,
		Example: `  search amazon bestsellers --category electronics
  search amazon bestsellers --type new-releases --category books
  search amazon bestsellers --type most-wished-for`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			var bestsellerURL string
			if category == "" {
				bestsellerURL = amazon.BaseURL + "/" + listType
			} else {
				bestsellerURL = amazon.BaseURL + "/" + listType + "/" + category
			}
			fmt.Printf("Fetching %s ...\n", bestsellerURL)

			task := &amazon.BestsellerTask{
				URL:      bestsellerURL,
				ListType: listType,
				Category: category,
				NodeID:   "",
				Client:   client,
				DB:       db,
				StateDB:  stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *amazon.BestsellerState) {
				fmt.Printf("  [%s] %s (entries=%d)\n", s.Status, s.URL, s.EntriesFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().StringVar(&category, "category", "", "Category slug (e.g. electronics, books)")
	cmd.Flags().StringVar(&listType, "type", "bestsellers", "List type: bestsellers, new-releases, most-wished-for, movers-and-shakers")
	return cmd
}

// ── reviews ───────────────────────────────────────────────────────────────────

func newAmazonReviews() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "reviews <ASIN>",
		Short: "Fetch all reviews for an Amazon product",
		Args:  cobra.ExactArgs(1),
		Example: `  search amazon reviews B08N5WRWNW
  search amazon reviews B08N5WRWNW --max-pages 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			asin := args[0]
			reviewURL := amazon.BaseURL + "/product-reviews/" + asin
			fmt.Printf("Fetching reviews for ASIN %s ...\n", asin)

			task := &amazon.ReviewTask{
				URL:      reviewURL,
				ASIN:     asin,
				Client:   client,
				DB:       db,
				StateDB:  stateDB,
				MaxPages: maxPages,
			}
			m, err := task.Run(cmd.Context(), func(s *amazon.ReviewState) {
				fmt.Printf("  [%s] page=%d reviews=%d\n", s.Status, s.Pages, s.ReviewsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d pages=%d skipped=%d failed=%d\n", m.Fetched, m.Pages, m.Skipped, m.Failed)
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

// ── qa ────────────────────────────────────────────────────────────────────────

func newAmazonQA() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "qa <ASIN>",
		Short: "Fetch all Q&A for an Amazon product",
		Args:  cobra.ExactArgs(1),
		Example: `  search amazon qa B08N5WRWNW
  search amazon qa B08N5WRWNW --max-pages 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			asin := args[0]
			qaURL := amazon.BaseURL + "/ask/" + asin
			fmt.Printf("Fetching Q&A for ASIN %s ...\n", asin)

			task := &amazon.QATask{
				URL:      qaURL,
				ASIN:     asin,
				Client:   client,
				DB:       db,
				StateDB:  stateDB,
				MaxPages: maxPages,
			}
			m, err := task.Run(cmd.Context(), func(s *amazon.QAState) {
				fmt.Printf("  [%s] page=%d qa=%d\n", s.Status, s.Pages, s.QAsFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d pages=%d skipped=%d failed=%d\n", m.Fetched, m.Pages, m.Skipped, m.Failed)
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

// ── seed ──────────────────────────────────────────────────────────────────────

func newAmazonSeed() *cobra.Command {
	var statePath string
	var filePath, entityType string
	var priority int

	cfg := amazon.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the crawl queue from a file (one ASIN or URL per line)",
		Args:  cobra.NoArgs,
		Example: `  search amazon seed --file asins.txt
  search amazon seed --file urls.txt --entity product --priority 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}

			stateDB, err := amazon.OpenState(statePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer stateDB.Close()

			f, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("open file: %w", err)
			}
			defer f.Close()

			total := 0
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				var enqURL string
				if entityType == amazon.EntityProduct {
					enqURL = normalizeProductURL(line)
				} else {
					enqURL = line
				}
				if err := stateDB.Enqueue(enqURL, entityType, priority); err == nil {
					total++
				}
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			fmt.Printf("Enqueued %d URLs (entity=%s priority=%d)\n", total, entityType, priority)
			return nil
		},
	}

	cmd.Flags().StringVar(&statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().StringVar(&filePath, "file", "", "File with one ASIN or URL per line (required)")
	cmd.Flags().StringVar(&entityType, "entity", amazon.EntityProduct, "Entity type: product, brand, author, category, search, bestseller, review, qa, seller")
	cmd.Flags().IntVar(&priority, "priority", 10, "Queue priority (higher = fetched first)")
	return cmd
}

// ── crawl ─────────────────────────────────────────────────────────────────────

func newAmazonCrawl() *cobra.Command {
	var dbPath, statePath string
	var delay, workers, maxPages int

	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Bulk crawl from the queue (use seed or search to seed first)",
		Args:  cobra.NoArgs,
		Example: `  search amazon seed --file asins.txt && search amazon crawl --workers 2
  search amazon crawl --workers 1 --delay 5000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := amazon.DefaultConfig()
			cfg.DBPath = dbPath
			cfg.StatePath = statePath
			cfg.Workers = workers
			cfg.Delay = time.Duration(delay) * time.Millisecond
			cfg.MaxPages = maxPages

			db, err := amazon.OpenDB(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			stateDB, err := amazon.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			client := amazon.NewClient(cfg)

			pending, _, done, failed := stateDB.QueueStats()
			fmt.Printf("Queue: pending=%d done=%d failed=%d\n", pending, done, failed)
			if pending == 0 {
				fmt.Println("Queue is empty. Run 'search amazon seed' or 'search amazon search' first.")
				return nil
			}

			fmt.Printf("Starting crawl with %d workers, delay=%s ...\n", workers, cfg.Delay)
			jobID := "crawl-" + fmt.Sprintf("%d", time.Now().Unix())
			stateDB.CreateJob(jobID, "bulk-crawl", "crawl")

			task := &amazon.CrawlTask{
				Config:  cfg,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *amazon.CrawlState) {
				amazon.PrintCrawlProgress(s)
			})
			if err != nil {
				_ = stateDB.UpdateJob(jobID, "failed", err.Error())
				return err
			}
			_ = stateDB.UpdateJob(jobID, "done", fmt.Sprintf("done=%d failed=%d duration=%s", m.Done, m.Failed, m.Duration.Round(time.Second)))

			fmt.Printf("\nCrawl complete: done=%d failed=%d duration=%s\n",
				m.Done, m.Failed, m.Duration.Round(time.Second))
			return nil
		},
	}

	cfg := amazon.DefaultConfig()
	cmd.Flags().StringVar(&dbPath, "db", cfg.DBPath, "Path to amazon.duckdb")
	cmd.Flags().StringVar(&statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(&workers, "workers", cfg.Workers, "Concurrent fetch workers")
	cmd.Flags().IntVar(&delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
	cmd.Flags().IntVar(&maxPages, "max-pages", 0, "Max pages per entity (0 = unlimited)")
	return cmd
}

// ── info ─────────────────────────────────────────────────────────────────────

func newAmazonInfo() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show Amazon database stats and queue depth",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, _, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			return amazon.PrintStats(db, stateDB)
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	return cmd
}

// ── jobs ─────────────────────────────────────────────────────────────────────

func newAmazonJobs() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages, limit int

	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent crawl jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, stateDB, _, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			jobs, err := stateDB.ListJobs(limit)
			if err != nil {
				return err
			}

			if len(jobs) == 0 {
				fmt.Println("No jobs found.")
				return nil
			}

			fmt.Println("── Recent Jobs ──")
			for _, j := range jobs {
				dur := ""
				if !j.CompletedAt.IsZero() {
					dur = j.CompletedAt.Sub(j.StartedAt).Round(time.Second).String()
				}
				fmt.Printf("  [%s] %s  %s  started=%s  duration=%s\n",
					j.Status, j.JobID, j.Name,
					j.StartedAt.Format("2006-01-02 15:04"), dur)
			}
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of jobs to show")
	return cmd
}

// ── queue ─────────────────────────────────────────────────────────────────────

func newAmazonQueue() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages, limit int
	var status string

	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Inspect queue items",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, stateDB, _, err := openAmazonDBs(dbPath, statePath, delay, maxPages)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			items, err := stateDB.ListQueue(status, limit)
			if err != nil {
				return err
			}

			if len(items) == 0 {
				fmt.Printf("No %s items in queue.\n", status)
				return nil
			}

			fmt.Printf("── Queue items (status=%s) ──\n", status)
			for _, it := range items {
				fmt.Printf("  [%s] %s\n", it.EntityType, it.URL)
			}
			return nil
		},
	}

	addAmazonFlags(cmd, &dbPath, &statePath, &delay, &maxPages)
	cmd.Flags().StringVar(&status, "status", "pending", "Filter by status: pending, failed, done, in_progress")
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of items to show")
	return cmd
}
