package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawler"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/store/sqlite"
	"github.com/spf13/cobra"
)

// NewCrawl creates the crawl command
func NewCrawl() *cobra.Command {
	var (
		depth     int
		limit     int
		delay     int
		sitemap   string
		workers   int
		userAgent string
		include   []string
		exclude   []string
		noRobots  bool
		scope     string
		batchSize int
		resume    bool
	)

	cmd := &cobra.Command{
		Use:   "crawl [url]",
		Short: "Crawl and index web pages",
		Long: `Crawl and index web pages starting from a URL.

Examples:
  search crawl https://golang.org
  search crawl https://example.com --depth 3 --limit 100 --workers 4
  search crawl --sitemap https://example.com/sitemap.xml --limit 50
  search crawl --resume
  search crawl status`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var startURL string
			if len(args) > 0 {
				startURL = args[0]
			}

			scopePolicy := crawler.ScopeSameDomain
			switch strings.ToLower(scope) {
			case "host":
				scopePolicy = crawler.ScopeSameHost
			case "subpath":
				scopePolicy = crawler.ScopeSubpath
			}

			cfg := crawler.Config{
				Workers:       workers,
				MaxDepth:      depth,
				MaxPages:      limit,
				Delay:         time.Duration(delay) * time.Millisecond,
				UserAgent:     userAgent,
				Timeout:       30 * time.Second,
				Scope:         scopePolicy,
				IncludeGlobs:  include,
				ExcludeGlobs:  exclude,
				RespectRobots: !noRobots,
				BatchSize:     batchSize,
			}

			stateFile := filepath.Join(GetDataDir(), "crawl_state.json")
			if resume {
				cfg.StateFile = stateFile
			}

			return runCrawl(cmd.Context(), startURL, sitemap, cfg, stateFile, resume)
		},
	}

	cmd.Flags().IntVar(&depth, "depth", 2, "Maximum crawl depth")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum pages to crawl")
	cmd.Flags().IntVar(&delay, "delay", 1000, "Delay between requests in milliseconds")
	cmd.Flags().StringVar(&sitemap, "sitemap", "", "Sitemap URL to crawl")
	cmd.Flags().IntVar(&workers, "workers", 4, "Number of concurrent workers")
	cmd.Flags().StringVar(&userAgent, "user-agent", "MizuCrawler/1.0", "User-Agent header")
	cmd.Flags().StringSliceVar(&include, "include", nil, "URL path patterns to include (glob)")
	cmd.Flags().StringSliceVar(&exclude, "exclude", nil, "URL path patterns to exclude (glob)")
	cmd.Flags().BoolVar(&noRobots, "no-robots", false, "Ignore robots.txt")
	cmd.Flags().StringVar(&scope, "scope", "domain", "Scope policy: domain, host, or subpath")
	cmd.Flags().IntVar(&batchSize, "batch-size", 10, "Batch size for indexing")
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume an interrupted crawl")

	// Add subcommands
	cmd.AddCommand(newCrawlStatus())
	cmd.AddCommand(newRecrawl())

	return cmd
}

func newCrawlStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show crawl state and statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateFile := filepath.Join(GetDataDir(), "crawl_state.json")
			state, err := crawler.GetStateInfo(stateFile)
			if err != nil {
				return fmt.Errorf("reading crawl state: %w", err)
			}
			if state == nil {
				fmt.Println(infoStyle.Render("No crawl state found"))
				return nil
			}

			fmt.Println(Banner())
			fmt.Println(subtitleStyle.Render("Crawl Status"))
			fmt.Println()
			fmt.Println(labelStyle.Render("  Start URL:    ") + state.StartURL)
			fmt.Println(labelStyle.Render("  Started:      ") + state.StartedAt.Format(time.RFC3339))
			fmt.Println(labelStyle.Render("  Last Updated: ") + state.UpdatedAt.Format(time.RFC3339))
			fmt.Println()
			fmt.Println(labelStyle.Render("  Pages crawled: ") + fmt.Sprintf("%d", state.Stats.PagesSuccess))
			fmt.Println(labelStyle.Render("  Pages failed:  ") + fmt.Sprintf("%d", state.Stats.PagesFailed))
			fmt.Println(labelStyle.Render("  Pages skipped: ") + fmt.Sprintf("%d", state.Stats.PagesSkipped))
			fmt.Println(labelStyle.Render("  URLs visited:  ") + fmt.Sprintf("%d", len(state.Visited)))
			fmt.Println(labelStyle.Render("  URLs pending:  ") + fmt.Sprintf("%d", len(state.Pending)))
			fmt.Println()

			return nil
		},
	}
}

func runCrawl(ctx context.Context, startURL, sitemapURL string, cfg crawler.Config, stateFile string, resume bool) error {
	if startURL == "" && sitemapURL == "" && !resume {
		return fmt.Errorf("either a URL, --sitemap, or --resume is required")
	}

	// If resuming without a URL, load it from state
	if resume && startURL == "" && sitemapURL == "" {
		state, err := crawler.GetStateInfo(stateFile)
		if err != nil || state == nil {
			return fmt.Errorf("no crawl state to resume from")
		}
		startURL = state.StartURL
		cfg.StateFile = stateFile
	}

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Crawling web pages..."))
	fmt.Println()

	// Connect to database
	fmt.Println(infoStyle.Render("Opening SQLite database..."))
	s, err := sqlite.New(GetDatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer s.Close()
	fmt.Println(successStyle.Render("  Database opened"))

	// Create crawler
	c, err := crawler.New(cfg)
	if err != nil {
		return fmt.Errorf("creating crawler: %w", err)
	}

	// Batch results for bulk indexing
	var batch []*store.Document
	c.OnResult(func(r crawler.CrawlResult) {
		doc := resultToDocument(r)
		batch = append(batch, doc)

		fmt.Println(successStyle.Render(fmt.Sprintf("  [%d] %s", len(batch), r.Title)))

		if len(batch) >= cfg.BatchSize {
			if err := s.Index().BulkIndex(ctx, batch); err != nil {
				fmt.Println(warningStyle.Render(fmt.Sprintf("  Failed to index batch: %v", err)))
			}
			batch = batch[:0]
		}
	})

	c.OnProgress(func(stats crawler.CrawlStats) {
		// Progress is printed via OnResult
	})

	// Run crawl
	var stats crawler.CrawlStats
	if sitemapURL != "" {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Crawling sitemap: %s", sitemapURL)))
		stats, err = c.CrawlSitemap(ctx, sitemapURL)
	} else {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Crawling URL: %s (depth: %d, workers: %d)", startURL, cfg.MaxDepth, cfg.Workers)))
		stats, err = c.Crawl(ctx, startURL)
	}

	// Index remaining batch
	if len(batch) > 0 {
		if indexErr := s.Index().BulkIndex(ctx, batch); indexErr != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Failed to index final batch: %v", indexErr)))
		}
	}

	if err != nil {
		return fmt.Errorf("crawl failed: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("Crawl complete: %d pages (%d success, %d failed, %d skipped) in %s",
		stats.PagesTotal, stats.PagesSuccess, stats.PagesFailed, stats.PagesSkipped,
		stats.Duration.Truncate(time.Millisecond))))
	if stats.PagesPerSecond > 0 {
		fmt.Println(labelStyle.Render(fmt.Sprintf("  %.1f pages/sec, %d bytes total", stats.PagesPerSecond, stats.BytesTotal)))
	}
	fmt.Println()

	return nil
}

func newRecrawl() *cobra.Command {
	var (
		dbPath    string
		workers   int
		timeout   int
		headOnly  bool
		batchSize int
		resume    bool
		userAgent string
	)

	cmd := &cobra.Command{
		Use:   "recrawl",
		Short: "High-throughput recrawl of URLs from a DuckDB seed database",
		Long: `Recrawl URLs from a DuckDB seed database at maximum throughput.

Seeds URLs from a DuckDB file (e.g. test.duckdb with a 'docs' table),
stores state in <name>.state.duckdb and results in <name>.result.duckdb.

Examples:
  search crawl recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb
  search crawl recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb --workers 1000 --head-only
  search crawl recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb --resume`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				return fmt.Errorf("--db is required")
			}
			return runRecrawl(cmd.Context(), dbPath, crawler.RecrawlConfig{
				Workers:   workers,
				Timeout:   time.Duration(timeout) * time.Second,
				UserAgent: userAgent,
				HeadOnly:  headOnly,
				BatchSize: batchSize,
				Resume:    resume,
			})
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to seed DuckDB file (required)")
	cmd.Flags().IntVar(&workers, "workers", 500, "Number of concurrent workers")
	cmd.Flags().IntVar(&timeout, "timeout", 10, "Per-request timeout in seconds")
	cmd.Flags().BoolVar(&headOnly, "head-only", false, "Only fetch headers, skip body")
	cmd.Flags().IntVar(&batchSize, "batch-size", 1000, "DB write batch size")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip already-crawled URLs")
	cmd.Flags().StringVar(&userAgent, "user-agent", "MizuCrawler/1.0", "User-Agent header")

	return cmd
}

func runRecrawl(ctx context.Context, dbPath string, cfg crawler.RecrawlConfig) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("High-Throughput Recrawl"))
	fmt.Println()

	// Load seed stats first
	fmt.Println(infoStyle.Render("Loading seed database..."))
	seedStats, err := crawler.LoadSeedStats(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("loading seed stats: %w", err)
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  %d URLs across %d domains",
		seedStats.TotalURLs, seedStats.UniqueDomains)))

	// Load seed URLs
	fmt.Println(infoStyle.Render("Loading URLs..."))
	seeds, err := crawler.LoadSeedURLs(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("loading seed URLs: %w", err)
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  Loaded %d URLs", len(seeds))))

	// Check for resume
	var skip map[string]bool
	// Derive state/result paths from seed DB path
	// e.g. test.duckdb → test.state.duckdb, test.result.duckdb
	base := strings.TrimSuffix(dbPath, ".duckdb")
	statePath := base + ".state.duckdb"
	resultPath := base + ".result.duckdb"

	if cfg.Resume {
		fmt.Println(infoStyle.Render("Checking for previous crawl state..."))
		skip, err = crawler.LoadAlreadyCrawled(ctx, statePath)
		if err != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Could not load state: %v", err)))
		} else if len(skip) > 0 {
			fmt.Println(successStyle.Render(fmt.Sprintf("  Resuming: skipping %d already-crawled URLs", len(skip))))
		}
	}

	// Open result/state databases
	fmt.Println(infoStyle.Render("Opening result databases..."))
	rdb, err := crawler.NewResultDB(resultPath, statePath, cfg.BatchSize)
	if err != nil {
		return fmt.Errorf("opening result db: %w", err)
	}
	defer rdb.Close()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Results → %s", resultPath)))
	fmt.Println(successStyle.Render(fmt.Sprintf("  State   → %s", statePath)))

	// Store metadata
	rdb.SetMeta(ctx, "seed_db", dbPath)
	rdb.SetMeta(ctx, "started_at", time.Now().Format(time.RFC3339))
	rdb.SetMeta(ctx, "workers", fmt.Sprintf("%d", cfg.Workers))

	// Create stats tracker
	label := filepath.Base(strings.TrimSuffix(dbPath, ".duckdb"))
	stats := crawler.NewRecrawlStats(seedStats.TotalURLs, seedStats.UniqueDomains, label)

	// Print config summary
	fmt.Println()
	fmt.Println(infoStyle.Render(fmt.Sprintf("Starting recrawl: %d workers, %v timeout, head-only=%v",
		cfg.Workers, cfg.Timeout, cfg.HeadOnly)))
	fmt.Println()

	// Create and run recrawler with live display
	r := crawler.NewRecrawler(cfg, stats, rdb)
	err = crawler.RunWithDisplay(ctx, r, seeds, skip, stats)

	// Final flush
	rdb.Flush(ctx)
	rdb.SetMeta(ctx, "finished_at", time.Now().Format(time.RFC3339))

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("Recrawl finished with error: %v", err)))
	} else {
		fmt.Println(successStyle.Render("Recrawl complete!"))
	}
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Results: %s", resultPath)))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  State:   %s", statePath)))
	fmt.Println()

	return err
}

// resultToDocument converts a CrawlResult to a store.Document.
func resultToDocument(r crawler.CrawlResult) *store.Document {
	wordCount := len(strings.Fields(r.Content))

	metadata := make(map[string]any, len(r.Metadata))
	for k, v := range r.Metadata {
		metadata[k] = v
	}

	return &store.Document{
		URL:         r.URL,
		Title:       r.Title,
		Description: r.Description,
		Content:     r.Content,
		Domain:      r.Domain,
		Language:    r.Language,
		ContentType: r.ContentType,
		WordCount:   wordCount,
		CrawledAt:   r.CrawledAt,
		Metadata:    metadata,
	}
}
