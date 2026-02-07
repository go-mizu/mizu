package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
	"github.com/spf13/cobra"
)

// NewRecrawl creates the top-level recrawl command.
func NewRecrawl() *cobra.Command {
	var (
		dbPath          string
		dirPath         string
		latest          bool
		workers         int
		dnsWorkers      int
		timeout         int
		headOnly        bool
		statusOnly      bool
		batchSize       int
		resume          bool
		userAgent       string
		dnsPrefetch     bool
		transportShards int
		twoPass         bool
	)

	cmd := &cobra.Command{
		Use:   "recrawl",
		Short: "High-throughput recrawl of URLs from a DuckDB seed database",
		Long: `Recrawl URLs from a DuckDB seed database at maximum throughput.

Seeds URLs from a DuckDB file (e.g. test.duckdb with a 'docs' table),
stores state in <name>.state.duckdb and results in <name>.result.duckdb.

Performance flags:
  --status-only       Only check HTTP status, close body immediately (fastest mode)
  --transport-shards  Shard HTTP transport pools to reduce lock contention
  --workers           Concurrent workers (default: 100000)

Directory mode:
  --dir + --latest    Auto-select the highest-index DuckDB file in a directory

Examples:
  search recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb
  search recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb --status-only --transport-shards 16
  search recrawl --dir ~/data/fineweb-1/CC-MAIN-2024-51/ --latest --status-only
  search recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb --resume`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve --dir + --latest to --db
			if dirPath != "" && latest {
				resolved, err := findLatestDuckDB(dirPath)
				if err != nil {
					return err
				}
				dbPath = resolved
			}
			if dbPath == "" {
				return fmt.Errorf("--db is required (or use --dir with --latest)")
			}
			return runRecrawl(cmd.Context(), dbPath, recrawler.Config{
				Workers:         workers,
				DNSWorkers:      dnsWorkers,
				Timeout:         time.Duration(timeout) * time.Millisecond,
				UserAgent:       userAgent,
				HeadOnly:        headOnly,
				StatusOnly:      statusOnly,
				BatchSize:       batchSize,
				Resume:          resume,
				DNSPrefetch:     dnsPrefetch,
				TransportShards: transportShards,
				TwoPass:        twoPass,
			})
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to seed DuckDB file")
	cmd.Flags().StringVar(&dirPath, "dir", "", "Directory containing DuckDB files (use with --latest)")
	cmd.Flags().BoolVar(&latest, "latest", false, "Auto-select highest-index DuckDB in --dir")
	cmd.Flags().IntVar(&workers, "workers", 100000, "Number of concurrent HTTP fetch workers")
	cmd.Flags().IntVar(&dnsWorkers, "dns-workers", 5000, "Number of concurrent DNS pipeline workers")
	cmd.Flags().IntVar(&timeout, "timeout", 1000, "Per-request timeout in milliseconds")
	cmd.Flags().BoolVar(&headOnly, "head-only", false, "Only fetch headers, skip body")
	cmd.Flags().BoolVar(&statusOnly, "status-only", false, "Only check HTTP status, close body immediately (fastest)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "DB write batch size")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip already-crawled URLs")
	cmd.Flags().StringVar(&userAgent, "user-agent", "MizuCrawler/1.0", "User-Agent header")
	cmd.Flags().BoolVar(&dnsPrefetch, "dns-prefetch", true, "Pre-resolve DNS for all domains")
	cmd.Flags().IntVar(&transportShards, "transport-shards", 16, "Number of HTTP transport shards")
	cmd.Flags().BoolVar(&twoPass, "two-pass", false, "Two-pass mode: probe domains before full fetch")

	return cmd
}

// findLatestDuckDB finds the DuckDB file with the highest filename index in a directory.
// Looks for *.parquet.duckdb or *.duckdb (excluding state/result/dns files).
func findLatestDuckDB(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var candidates []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".parquet.duckdb") {
			candidates = append(candidates, name)
		} else if strings.HasSuffix(name, ".duckdb") &&
			!strings.HasSuffix(name, ".state.duckdb") &&
			!strings.HasSuffix(name, ".result.duckdb") &&
			!strings.HasSuffix(name, ".dns.duckdb") &&
			name != "dns.duckdb" {
			candidates = append(candidates, name)
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no DuckDB files found in %s", dir)
	}

	sort.Strings(candidates)
	return filepath.Join(dir, candidates[len(candidates)-1]), nil
}

func runRecrawl(ctx context.Context, dbPath string, cfg recrawler.Config) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("High-Throughput Recrawl"))
	fmt.Println()

	// Load seed stats first
	fmt.Println(infoStyle.Render("Loading seed database..."))
	seedStats, err := recrawler.LoadSeedStats(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("loading seed stats: %w", err)
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  %d URLs across %d domains",
		seedStats.TotalURLs, seedStats.UniqueDomains)))

	// Load seed URLs (only url + domain, no sorting)
	fmt.Println(infoStyle.Render("Loading URLs..."))
	seeds, err := recrawler.LoadSeedURLs(ctx, dbPath, seedStats.TotalURLs)
	if err != nil {
		return fmt.Errorf("loading seed URLs: %w", err)
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  Loaded %d URLs", len(seeds))))

	// Derive result dir and DNS path from seed DB path
	base := strings.TrimSuffix(dbPath, ".duckdb")
	base = strings.TrimSuffix(base, ".parquet")
	resultDir := base + ".results"

	// DNS cache: use directory-level cache for sharing across per-parquet files
	dnsPath := filepath.Join(filepath.Dir(dbPath), "dns.duckdb")

	// Check for resume (scan existing result shards)
	var skip map[string]bool
	if cfg.Resume {
		fmt.Println(infoStyle.Render("Checking for previous crawl state..."))
		skip, err = recrawler.LoadAlreadyCrawledFromDir(ctx, resultDir)
		if err != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Could not load state: %v", err)))
		} else if len(skip) > 0 {
			fmt.Println(successStyle.Render(fmt.Sprintf("  Resuming: skipping %d already-crawled URLs", len(skip))))
		}
	}

	// DNS resolver (if enabled, pipelined with fetch — no separate prefetch step)
	var dnsResolver *recrawler.DNSResolver
	if cfg.DNSPrefetch {
		dnsResolver = recrawler.NewDNSResolver(2 * time.Second)
		cached, _ := dnsResolver.LoadCache(dnsPath)
		if cached > 0 {
			fmt.Println(successStyle.Render(fmt.Sprintf("  DNS cache: loaded %d entries from %s", cached, filepath.Base(dnsPath))))
		}
	}

	// Open sharded result databases
	fmt.Println(infoStyle.Render("Opening result databases..."))
	rdb, err := recrawler.NewResultDB(resultDir, 8, cfg.BatchSize)
	if err != nil {
		return fmt.Errorf("opening result db: %w", err)
	}
	defer rdb.Close()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Results → %s/ (8 shards)", resultDir)))

	// Store metadata
	rdb.SetMeta(ctx, "seed_db", dbPath)
	rdb.SetMeta(ctx, "started_at", time.Now().Format(time.RFC3339))
	rdb.SetMeta(ctx, "workers", fmt.Sprintf("%d", cfg.Workers))

	// Create stats tracker
	label := filepath.Base(strings.TrimSuffix(dbPath, ".duckdb"))
	stats := recrawler.NewStats(seedStats.TotalURLs, seedStats.UniqueDomains, label)

	// Print config summary
	mode := "full"
	if cfg.StatusOnly {
		mode = "status-only"
	} else if cfg.HeadOnly {
		mode = "head-only"
	}
	pipelineMode := "direct"
	if dnsResolver != nil {
		pipelineMode = "dns-pipeline"
	}
	if cfg.TwoPass {
		pipelineMode = "two-pass"
	}
	fmt.Println()
	fmt.Println(infoStyle.Render(fmt.Sprintf("Starting recrawl: %d workers, %v timeout, mode=%s, shards=%d, pipeline=%s",
		cfg.Workers, cfg.Timeout, mode, cfg.TransportShards, pipelineMode)))
	fmt.Println()

	// Create and run recrawler with live display
	r := recrawler.New(cfg, stats, rdb)

	// Enable pipelined DNS+fetch mode (resolve domain → immediately fetch URLs)
	if dnsResolver != nil {
		r.SetDNSResolver(dnsResolver)
	}

	err = recrawler.RunWithDisplay(ctx, r, seeds, skip, stats)

	// Final flush
	rdb.Flush(ctx)
	rdb.SetMeta(ctx, "finished_at", time.Now().Format(time.RFC3339))

	// Save DNS cache for next run
	if dnsResolver != nil {
		fmt.Print(infoStyle.Render("  Saving DNS cache..."))
		saveStart := time.Now()
		if saveErr := dnsResolver.SaveCache(dnsPath); saveErr != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf(" failed: %v", saveErr)))
		} else {
			fmt.Println(successStyle.Render(fmt.Sprintf(" saved in %s → %s",
				time.Since(saveStart).Truncate(time.Millisecond), filepath.Base(dnsPath))))
		}
	}

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("Recrawl finished with error: %v", err)))
	} else {
		fmt.Println(successStyle.Render("Recrawl complete!"))
	}
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Results: %s/", resultDir)))
	fmt.Println()

	return err
}
