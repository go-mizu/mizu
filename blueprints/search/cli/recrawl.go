package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
	"github.com/spf13/cobra"
	_ "github.com/duckdb/duckdb-go/v2"
)

// NewRecrawl creates the top-level recrawl command.
func NewRecrawl() *cobra.Command {
	var (
		dbPath          string
		dirPath         string
		latest          bool
		workers         int
		dnsWorkers      int
		dnsTimeout      int
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
  --dns-workers       Concurrent DNS workers (default: 2000)
  --dns-timeout       DNS lookup timeout in ms (default: 2000)

Directory mode:
  --dir + --latest    Auto-select the highest-index DuckDB file in a directory

Examples:
  search recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb
  search recrawl --db ~/data/fineweb-2/vie_Latn/test.duckdb --status-only --transport-shards 64
  search recrawl --dir ~/data/fineweb-1/CC-MAIN-2025-26/ --latest --status-only
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
				DNSTimeout:      time.Duration(dnsTimeout) * time.Millisecond,
				Timeout:         time.Duration(timeout) * time.Millisecond,
				UserAgent:       userAgent,
				HeadOnly:        headOnly,
				StatusOnly:      statusOnly,
				BatchSize:       batchSize,
				Resume:          resume,
				DNSPrefetch:     dnsPrefetch,
				TransportShards: transportShards,
				TwoPass:         twoPass,
			})
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to seed DuckDB file")
	cmd.Flags().StringVar(&dirPath, "dir", "", "Directory containing DuckDB files (use with --latest)")
	cmd.Flags().BoolVar(&latest, "latest", false, "Auto-select highest-index DuckDB in --dir")
	cmd.Flags().IntVar(&workers, "workers", 200, "Number of concurrent HTTP fetch workers")
	cmd.Flags().IntVar(&dnsWorkers, "dns-workers", 2000, "Number of concurrent DNS workers")
	cmd.Flags().IntVar(&dnsTimeout, "dns-timeout", 2000, "DNS lookup timeout in milliseconds")
	cmd.Flags().IntVar(&timeout, "timeout", 5000, "Per-request HTTP timeout in milliseconds")
	cmd.Flags().BoolVar(&headOnly, "head-only", false, "Only fetch headers, skip body")
	cmd.Flags().BoolVar(&statusOnly, "status-only", false, "Only check HTTP status, close body immediately (fastest)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "DB write batch size")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip already-crawled URLs")
	cmd.Flags().StringVar(&userAgent, "user-agent", "MizuCrawler/1.0", "User-Agent header")
	cmd.Flags().BoolVar(&dnsPrefetch, "dns-prefetch", true, "Batch DNS pre-resolution for all domains")
	cmd.Flags().IntVar(&transportShards, "transport-shards", 64, "Number of HTTP transport shards")
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
	if err := ctx.Err(); err != nil {
		return err
	}

	// Derive result dir and DNS path from seed DB path
	base := strings.TrimSuffix(dbPath, ".duckdb")
	base = strings.TrimSuffix(base, ".parquet")
	resultDir := base + ".results"
	mergedResultsPath := base + ".results.duckdb"
	failedDBPath := base + ".failed.duckdb"

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
	if err := ctx.Err(); err != nil {
		return err
	}

	// DNS resolver
	var dnsResolver *recrawler.DNSResolver
	saveDNSCache := func(prefix string) {
		if dnsResolver == nil {
			return
		}
		fmt.Print(infoStyle.Render(prefix))
		saveStart := time.Now()
		if saveErr := dnsResolver.SaveCache(dnsPath); saveErr != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf(" failed: %v", saveErr)))
		} else {
			fmt.Println(successStyle.Render(fmt.Sprintf(" saved in %s → %s (live=%d, dead=%d, timeout=%d)",
				time.Since(saveStart).Truncate(time.Millisecond), filepath.Base(dnsPath),
				dnsResolver.LiveCount(), dnsResolver.DeadCount(), dnsResolver.TimeoutCount())))
		}
	}
	if cfg.DNSPrefetch {
		dnsResolver = recrawler.NewDNSResolver(cfg.DNSTimeout)
		cached, _ := dnsResolver.LoadCache(dnsPath)
		if cached > 0 {
			fmt.Println(successStyle.Render(fmt.Sprintf("  DNS cache: loaded %d entries (live=%d, dead=%d, timeout=%d)",
				cached, dnsResolver.LiveCount(), dnsResolver.DeadCount(), dnsResolver.TimeoutCount())))
		}

		// Batch DNS pre-resolution: resolve all uncached domains upfront
		// This is much faster than per-domain resolution in the pipeline
		allDomains := make(map[string]bool, len(seeds))
		for _, s := range seeds {
			if skip == nil || !skip[s.URL] {
				allDomains[s.Domain] = true
			}
		}
		domainList := make([]string, 0, len(allDomains))
		for d := range allDomains {
			domainList = append(domainList, d)
		}

		fmt.Println(infoStyle.Render(fmt.Sprintf("  Batch DNS: resolving %d domains (%d workers, %v timeout)...",
			len(domainList), cfg.DNSWorkers, cfg.DNSTimeout)))

		var dnsDisplayLines int
		var checkpointMu sync.Mutex
		checkpointSaving := false
		checkpointDoneAt := int64(0)
		lastCheckpointAt := time.Now()
		checkpointEveryDone := int64(50000)
		if len(domainList) < 50000 {
			checkpointEveryDone = 5000
		}
		if len(domainList) < 5000 {
			checkpointEveryDone = 1000
		}
		checkpointEveryDur := 30 * time.Second
		saveDNSCheckpointAsync := func(reason string, force bool) {
			checkpointMu.Lock()
			if checkpointSaving && !force {
				checkpointMu.Unlock()
				return
			}
			checkpointSaving = true
			checkpointMu.Unlock()
			go func() {
				saveStart := time.Now()
				if saveErr := dnsResolver.SaveCache(dnsPath); saveErr != nil {
					fmt.Println(warningStyle.Render(fmt.Sprintf("  DNS cache checkpoint save failed (%s): %v", reason, saveErr)))
				} else {
					fmt.Println(infoStyle.Render(fmt.Sprintf("  DNS cache checkpoint saved (%s) in %s → %s (live=%d, dead=%d, timeout=%d)",
						reason,
						time.Since(saveStart).Truncate(time.Millisecond),
						filepath.Base(dnsPath),
						dnsResolver.LiveCount(), dnsResolver.DeadCount(), dnsResolver.TimeoutCount(),
					)))
				}
				checkpointMu.Lock()
				checkpointSaving = false
				checkpointMu.Unlock()
			}()
		}
		live, dead, timeout := dnsResolver.ResolveBatch(ctx, domainList, cfg.DNSWorkers, cfg.DNSTimeout, func(p recrawler.DNSProgress) {
			if dnsDisplayLines > 0 {
				fmt.Printf("\033[%dA\033[J", dnsDisplayLines)
			}
			etaStr := "---"
			if p.Done < int64(p.Total) && p.AvgSpeed > 0 {
				remaining := float64(p.Total) - float64(p.Done)
				if remaining > 0 {
					eta := time.Duration((remaining / p.AvgSpeed) * float64(time.Second))
					etaStr = formatDuration(eta)
				}
			} else if p.Done >= int64(p.Total) {
				etaStr = "0s"
			}
			output := fmt.Sprintf("  DNS  %d/%d  │  %d live  │  %d dead  │  %d timeout  │  %.0f/s  │  ETA %s  │  %s\n",
				p.Done, p.Total, p.Live, p.Dead, p.Timeout, p.Speed, etaStr, p.Elapsed.Truncate(time.Millisecond))
			fmt.Print(output)
			dnsDisplayLines = 1

			// Periodically checkpoint DNS cache during long batch DNS runs so progress survives
			// interrupts and large in-memory caches are persisted incrementally.
			if p.Total > 0 && p.Done < int64(p.Total) {
				needDoneCheckpoint := p.Done-checkpointDoneAt >= checkpointEveryDone
				needTimeCheckpoint := time.Since(lastCheckpointAt) >= checkpointEveryDur && p.Done > checkpointDoneAt
				if needDoneCheckpoint || needTimeCheckpoint {
					checkpointDoneAt = p.Done
					lastCheckpointAt = time.Now()
					saveDNSCheckpointAsync(fmt.Sprintf("%d/%d", p.Done, p.Total), false)
				}
			}
		})
		checkpointMu.Lock()
		inFlight := checkpointSaving
		checkpointMu.Unlock()
		if inFlight {
			fmt.Println(infoStyle.Render("  Waiting for DNS cache checkpoint save to finish..."))
		waitCheckpoint:
			for {
				checkpointMu.Lock()
				doneSaving := !checkpointSaving
				checkpointMu.Unlock()
				if doneSaving {
					break
				}
				select {
				case <-ctx.Done():
					break waitCheckpoint
				case <-time.After(200 * time.Millisecond):
				}
			}
		}
		if dnsDisplayLines > 0 {
			fmt.Printf("\033[%dA\033[J", dnsDisplayLines)
		}
		fmt.Println(successStyle.Render(fmt.Sprintf("  DNS: %d live, %d dead, %d timeout (%s)",
			live, dead, timeout, dnsResolver.Duration().Truncate(time.Millisecond))))
		saveDNSCache("  Saving DNS cache (post-DNS prefetch)...")
		if err := ctx.Err(); err != nil {
			return err
		}
	}

	// Open sharded result databases
	fmt.Println(infoStyle.Render("Opening result databases..."))
	rdb, err := recrawler.NewResultDB(resultDir, 8, cfg.BatchSize)
	if err != nil {
		return fmt.Errorf("opening result db: %w", err)
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  Results → %s/ (8 shards)", resultDir)))
	fmt.Printf("  %s %s\n", labelStyle.Render("Merged output:"), labelStyle.Render(mergedResultsPath))

	fmt.Println(infoStyle.Render("Opening failed database..."))
	fdb, err := recrawler.NewFailedDB(failedDBPath)
	if err != nil {
		_ = rdb.Close()
		return fmt.Errorf("opening failed db: %w", err)
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  Failed → %s", failedDBPath)))

	// Store metadata
	rdb.SetMeta(ctx, "seed_db", dbPath)
	rdb.SetMeta(ctx, "started_at", time.Now().Format(time.RFC3339))
	rdb.SetMeta(ctx, "workers", fmt.Sprintf("%d", cfg.Workers))
	fdb.SetMeta("seed_db", dbPath)
	fdb.SetMeta("started_at", time.Now().Format(time.RFC3339))

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
		pipelineMode = "batch-dns → direct"
	}
	if cfg.TwoPass {
		pipelineMode = "two-pass"
	}
	fmt.Println()
	fmt.Println(infoStyle.Render(fmt.Sprintf("Starting recrawl: %d workers, %v timeout, mode=%s, shards=%d, pipeline=%s",
		cfg.Workers, cfg.Timeout, mode, cfg.TransportShards, pipelineMode)))
	fmt.Println()

	// Create and run recrawler
	r := recrawler.New(cfg, stats, rdb)
	r.SetFailedDB(fdb)

	// Pre-populate DNS cache and dead domains from batch resolution.
	// Use SetDNSCache + SetDeadDomains (NOT SetDNSResolver) so that Run()
	// uses directFeed instead of spawning another DNS pipeline.
	if dnsResolver != nil {
		r.SetDNSCache(dnsResolver.ResolvedIPs())
		r.SetDeadDomains(dnsResolver.DeadOrTimeoutDomainsWithReasons())
	}

	err = recrawler.RunWithDisplay(ctx, r, seeds, skip, stats)

	// Final flush
	rdb.Flush(ctx)
	rdb.SetMeta(ctx, "finished_at", time.Now().Format(time.RFC3339))
	fdb.SetMeta("finished_at", time.Now().Format(time.RFC3339))
	if closeErr := fdb.Close(); closeErr != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("Failed DB close error: %v", closeErr)))
	}
	failedDomains := fdb.DomainCount()
	failedURLs := fdb.URLCount()
	if closeErr := rdb.Close(); closeErr != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("Result DB close error: %v", closeErr)))
	}

	if dnsResolver != nil {
		saveDNSCache("  Saving DNS cache...")
	}

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("Recrawl finished with error: %v", err)))
	} else {
		fmt.Println(successStyle.Render("Recrawl complete!"))
	}
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Results: %s/", resultDir)))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Failed:  %s", failedDBPath)))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Failed summary: domains=%s urls=%s",
		formatLargeNumber(failedDomains), formatLargeNumber(failedURLs))))
	if err == nil {
		fmt.Println(infoStyle.Render("Consolidating result shards..."))
		mergeStart := time.Now()
		rows, shardCount, mergeErr := mergeResultShardsToDuckDB(ctx, resultDir, mergedResultsPath)
		if mergeErr != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Consolidation failed: %v", mergeErr)))
		} else {
			fmt.Println(successStyle.Render(fmt.Sprintf("  Merged %d shard(s) → %s (%s rows, %s)",
				shardCount, mergedResultsPath, formatLargeNumber(rows), time.Since(mergeStart).Truncate(time.Millisecond))))
		}
	} else {
		fmt.Println(labelStyle.Render("  Consolidation: skipped (recrawl not complete)"))
	}
	fmt.Println()

	return err
}

func mergeResultShardsToDuckDB(ctx context.Context, resultDir, outPath string) (rows int64, shardCount int, err error) {
	entries, err := os.ReadDir(resultDir)
	if err != nil {
		return 0, 0, fmt.Errorf("reading result dir: %w", err)
	}
	var shards []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "results_") && strings.HasSuffix(name, ".duckdb") {
			shards = append(shards, filepath.Join(resultDir, name))
		}
	}
	if len(shards) == 0 {
		return 0, 0, fmt.Errorf("no result shards found in %s", resultDir)
	}
	sort.Strings(shards)

	tmpPath := outPath + ".tmp"
	_ = os.Remove(tmpPath)
	db, err := sql.Open("duckdb", tmpPath)
	if err != nil {
		return 0, 0, fmt.Errorf("opening merged db: %w", err)
	}
	defer func() {
		db.Close()
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err = db.ExecContext(ctx, `
		CREATE TABLE results (
			url VARCHAR PRIMARY KEY,
			status_code INTEGER,
			content_type VARCHAR,
			content_length BIGINT,
			body VARCHAR,
			title VARCHAR,
			description VARCHAR,
			language VARCHAR,
			domain VARCHAR,
			redirect_url VARCHAR,
			fetch_time_ms BIGINT,
			crawled_at TIMESTAMP,
			error VARCHAR,
			status VARCHAR
		);
		CREATE TABLE IF NOT EXISTS meta (key VARCHAR PRIMARY KEY, value VARCHAR);
	`); err != nil {
		return 0, 0, fmt.Errorf("creating merged schema: %w", err)
	}

	for i, shardPath := range shards {
		alias := fmt.Sprintf("s%d", i)
		if _, err = db.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS %s (READ_ONLY);", strings.ReplaceAll(shardPath, "'", "''"), alias)); err != nil {
			return 0, len(shards), fmt.Errorf("attach shard %s: %w", shardPath, err)
		}
		if _, err = db.ExecContext(ctx, fmt.Sprintf("INSERT OR REPLACE INTO results SELECT * FROM %s.results", alias)); err != nil {
			return 0, len(shards), fmt.Errorf("merge shard %s: %w", shardPath, err)
		}
		_, _ = db.ExecContext(ctx, fmt.Sprintf("INSERT OR REPLACE INTO meta SELECT * FROM %s.meta", alias))
		if _, err = db.ExecContext(ctx, fmt.Sprintf("DETACH %s", alias)); err != nil {
			return 0, len(shards), fmt.Errorf("detach shard %s: %w", shardPath, err)
		}
	}
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM results").Scan(&rows)

	if _, err = db.ExecContext(ctx, "INSERT OR REPLACE INTO meta (key, value) VALUES ('merged_from_dir', ?), ('merged_at', ?)", resultDir, time.Now().Format(time.RFC3339)); err != nil {
		return 0, len(shards), fmt.Errorf("writing merged meta: %w", err)
	}
	if err = db.Close(); err != nil {
		return 0, len(shards), fmt.Errorf("closing merged db: %w", err)
	}
	if rmErr := os.Remove(outPath); rmErr != nil && !os.IsNotExist(rmErr) {
		return 0, len(shards), fmt.Errorf("removing old merged db: %w", rmErr)
	}
	if err = os.Rename(tmpPath, outPath); err != nil {
		return 0, len(shards), fmt.Errorf("installing merged db: %w", err)
	}
	return rows, len(shards), nil
}
