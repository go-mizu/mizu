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
		timeout         int
		headOnly        bool
		statusOnly      bool
		batchSize       int
		resume          bool
		userAgent       string
		dnsPrefetch     bool
		transportShards int
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
				Timeout:         time.Duration(timeout) * time.Millisecond,
				UserAgent:       userAgent,
				HeadOnly:        headOnly,
				StatusOnly:      statusOnly,
				BatchSize:       batchSize,
				Resume:          resume,
				DNSPrefetch:     dnsPrefetch,
				TransportShards: transportShards,
			})
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to seed DuckDB file")
	cmd.Flags().StringVar(&dirPath, "dir", "", "Directory containing DuckDB files (use with --latest)")
	cmd.Flags().BoolVar(&latest, "latest", false, "Auto-select highest-index DuckDB in --dir")
	cmd.Flags().IntVar(&workers, "workers", 100000, "Number of concurrent workers")
	cmd.Flags().IntVar(&timeout, "timeout", 1000, "Per-request timeout in milliseconds")
	cmd.Flags().BoolVar(&headOnly, "head-only", false, "Only fetch headers, skip body")
	cmd.Flags().BoolVar(&statusOnly, "status-only", false, "Only check HTTP status, close body immediately (fastest)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "DB write batch size")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip already-crawled URLs")
	cmd.Flags().StringVar(&userAgent, "user-agent", "MizuCrawler/1.0", "User-Agent header")
	cmd.Flags().BoolVar(&dnsPrefetch, "dns-prefetch", true, "Pre-resolve DNS for all domains")
	cmd.Flags().IntVar(&transportShards, "transport-shards", 16, "Number of HTTP transport shards")

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

	// Derive state/result/dns paths from seed DB path
	base := strings.TrimSuffix(dbPath, ".duckdb")
	// Also strip .parquet suffix if present (e.g. foo.parquet.duckdb → foo.parquet)
	base = strings.TrimSuffix(base, ".parquet")
	statePath := base + ".state.duckdb"
	resultPath := base + ".result.duckdb"

	// DNS cache: use directory-level cache for sharing across per-parquet files
	dnsPath := filepath.Join(filepath.Dir(dbPath), "dns.duckdb")

	// Check for resume
	var skip map[string]bool
	if cfg.Resume {
		fmt.Println(infoStyle.Render("Checking for previous crawl state..."))
		skip, err = recrawler.LoadAlreadyCrawled(ctx, statePath)
		if err != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Could not load state: %v", err)))
		} else if len(skip) > 0 {
			fmt.Println(successStyle.Render(fmt.Sprintf("  Resuming: skipping %d already-crawled URLs", len(skip))))
		}
	}

	// DNS pre-resolution with cache
	var dnsResolver *recrawler.DNSResolver
	if cfg.DNSPrefetch {
		dnsResolver = recrawler.NewDNSResolver(2 * time.Second)

		// Load DNS cache
		cached, _ := dnsResolver.LoadCache(dnsPath)
		if cached > 0 {
			fmt.Println(successStyle.Render(fmt.Sprintf("  DNS cache: loaded %d entries from %s", cached, filepath.Base(dnsPath))))
		}

		// Extract unique domains
		domainSet := make(map[string]bool, seedStats.UniqueDomains)
		for _, s := range seeds {
			if s.Domain != "" {
				domainSet[s.Domain] = true
			}
		}
		domains := make([]string, 0, len(domainSet))
		for d := range domainSet {
			domains = append(domains, d)
		}

		fmt.Println(infoStyle.Render(fmt.Sprintf("Pre-resolving DNS for %d domains...", len(domains))))
		dnsWorkers := min(len(domains), 10000)
		live, dead := dnsResolver.Resolve(ctx, domains, dnsWorkers)
		fmt.Println(successStyle.Render(fmt.Sprintf("  DNS: %d live, %d dead (%s)",
			live, dead, dnsResolver.Duration().Truncate(time.Millisecond))))

		// Save DNS cache for next run
		newEntries := live + dead - int(dnsResolver.CachedCount())
		if newEntries > 0 {
			fmt.Print(infoStyle.Render("  Saving DNS cache..."))
			saveStart := time.Now()
			if err := dnsResolver.SaveCache(dnsPath); err != nil {
				fmt.Println(warningStyle.Render(fmt.Sprintf(" failed: %v", err)))
			} else {
				fmt.Println(successStyle.Render(fmt.Sprintf(" saved %d entries in %s → %s",
					live+dead, time.Since(saveStart).Truncate(time.Millisecond), filepath.Base(dnsPath))))
			}
		}

		// Count URLs that will be skipped due to dead domains
		deadDomains := dnsResolver.DeadDomains()
		urlsSkipped := 0
		for _, s := range seeds {
			if deadDomains[s.Domain] {
				urlsSkipped++
			}
		}
		if urlsSkipped > 0 {
			fmt.Println(successStyle.Render(fmt.Sprintf("  Skipping %d URLs on dead domains (%.1f%%)",
				urlsSkipped, float64(urlsSkipped)/float64(len(seeds))*100)))
		}
	}

	// Open result/state databases
	fmt.Println(infoStyle.Render("Opening result databases..."))
	rdb, err := recrawler.NewResultDB(resultPath, statePath, cfg.BatchSize)
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
	stats := recrawler.NewStats(seedStats.TotalURLs, seedStats.UniqueDomains, label)

	// Print config summary
	mode := "full"
	if cfg.StatusOnly {
		mode = "status-only"
	} else if cfg.HeadOnly {
		mode = "head-only"
	}
	fmt.Println()
	fmt.Println(infoStyle.Render(fmt.Sprintf("Starting recrawl: %d workers, %v timeout, mode=%s, shards=%d",
		cfg.Workers, cfg.Timeout, mode, cfg.TransportShards)))
	fmt.Println()

	// Create and run recrawler with live display
	r := recrawler.New(cfg, stats, rdb)

	// Apply DNS-dead domains and cached IPs for direct dialing
	if dnsResolver != nil {
		r.SetDeadDomains(dnsResolver.DeadDomains())
		r.SetDNSCache(dnsResolver.ResolvedIPs())
	}

	err = recrawler.RunWithDisplay(ctx, r, seeds, skip, stats)

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
