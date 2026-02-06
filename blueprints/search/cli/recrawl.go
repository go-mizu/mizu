package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
	"github.com/spf13/cobra"
)

// NewRecrawl creates the top-level recrawl command.
func NewRecrawl() *cobra.Command {
	var (
		dbPath      string
		workers     int
		timeout     int
		headOnly    bool
		batchSize   int
		resume      bool
		userAgent   string
		dnsPrefetch bool
	)

	cmd := &cobra.Command{
		Use:   "recrawl",
		Short: "High-throughput recrawl of URLs from a DuckDB seed database",
		Long: `Recrawl URLs from a DuckDB seed database at maximum throughput.

Seeds URLs from a DuckDB file (e.g. test.duckdb with a 'docs' table),
stores state in <name>.state.duckdb and results in <name>.result.duckdb.

DNS prefetch resolves all domains upfront, skipping dead domains instantly.
Results are cached in <name>.dns.duckdb for instant reuse across runs.
Per-domain failure tracking skips remaining URLs for domains that fail.

Examples:
  search recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb
  search recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb --workers 50000 --timeout 2 --dns-prefetch
  search recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb --resume`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				return fmt.Errorf("--db is required")
			}
			return runRecrawl(cmd.Context(), dbPath, recrawler.Config{
				Workers:     workers,
				Timeout:     time.Duration(timeout) * time.Millisecond,
				UserAgent:   userAgent,
				HeadOnly:    headOnly,
				BatchSize:   batchSize,
				Resume:      resume,
				DNSPrefetch: dnsPrefetch,
			})
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to seed DuckDB file (required)")
	cmd.Flags().IntVar(&workers, "workers", 50000, "Number of concurrent workers")
	cmd.Flags().IntVar(&timeout, "timeout", 1000, "Per-request timeout in milliseconds")
	cmd.Flags().BoolVar(&headOnly, "head-only", false, "Only fetch headers, skip body")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "DB write batch size")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip already-crawled URLs")
	cmd.Flags().StringVar(&userAgent, "user-agent", "MizuCrawler/1.0", "User-Agent header")
	cmd.Flags().BoolVar(&dnsPrefetch, "dns-prefetch", true, "Pre-resolve DNS for all domains")

	return cmd
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
	statePath := base + ".state.duckdb"
	resultPath := base + ".result.duckdb"

	// DNS cache: use split-level cache (e.g., train/dns.duckdb) for sharing
	// across per-parquet files, fall back to file-specific cache
	dnsPath := filepath.Join(filepath.Dir(dbPath), "dns.duckdb")
	if filepath.Base(dbPath) == filepath.Base(filepath.Dir(dbPath))+".duckdb" {
		// This IS the split-level file (e.g., train.duckdb), use file-specific cache
		dnsPath = base + ".dns.duckdb"
	}

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

		// Save DNS cache for next run (skip if all entries came from cache)
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
	fmt.Println()
	fmt.Println(infoStyle.Render(fmt.Sprintf("Starting recrawl: %d workers, %v timeout, head-only=%v, dns-prefetch=%v",
		cfg.Workers, cfg.Timeout, cfg.HeadOnly, cfg.DNSPrefetch)))
	fmt.Println()

	// Create and run recrawler with live display
	r := recrawler.New(cfg, stats, rdb)

	// Apply DNS-dead domains
	if dnsResolver != nil {
		r.SetDeadDomains(dnsResolver.DeadDomains())
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
