package cli

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
	"github.com/go-mizu/mizu/blueprints/search/pkg/archived/recrawler"
	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/spf13/cobra"
)

// NewCC creates the cc command with subcommands for Common Crawl operations.
func NewCC() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cc",
		Short: "Common Crawl index and WARC page extraction",
		Long: `Download, index, and extract pages from Common Crawl archives.

Supports the columnar index (parquet), CDXJ index, and WARC file extraction
via byte-range requests for high-throughput page retrieval.

Smart caching:
  --sample N    Download only N parquet files (evenly spaced) instead of all ~900
  --remote      Query parquet directly from S3 (zero disk, slower)
  Manifests, crawl lists, and latest crawl ID are cached for 24h in $HOME/data/common-crawl/cache.json

Subcommands:
  crawls   List available Common Crawl datasets
  parquet  List/download/import columnar-index parquet files
  index    Download + import columnar index to DuckDB
  stats    Show index statistics
  query    Query index for matching URLs (local or remote)
  fetch    High-throughput page extraction from WARC files
  recrawl  CC index → URL extraction → recrawl from origin servers
  warc     Work with WARC files (get/list/download/extract/import/query)
  url      Lookup a URL via CDX API

`,
		Example: `  search cc crawls
  search cc parquet list                           # latest crawl + subset=warc
  search cc parquet list --subset all --limit 20
  search cc parquet download --file m:600          # manifest selector
  search cc index --sample 5
  search cc stats
  search cc query --lang eng --status 200 --limit 100
  search cc query --remote --domain example.com --limit 10
  search cc fetch --remote --domain example.com --limit 100
  search cc recrawl --file 0 --status-only --limit 1000
  search cc warc list
  search cc warc get --file crawl-data/CC-MAIN-2026-08/... --offset 12345 --length 6789
  search cc url --url https://example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCCCrawls())
	cmd.AddCommand(newCCParquet())
	cmd.AddCommand(newCCIndex())
	cmd.AddCommand(newCCStats())
	cmd.AddCommand(newCCQuery())
	cmd.AddCommand(newCCFetch())
	cmd.AddCommand(newCCWarcParent())
	cmd.AddCommand(newCCURL())
	cmd.AddCommand(newCCRecrawl())
	cmd.AddCommand(newCCRecrawlDrone())
	cmd.AddCommand(newCCVerify())
	cmd.AddCommand(newCCSite())

	return cmd
}

// ── cc crawls ──────────────────────────────────────────────

func newCCCrawls() *cobra.Command {
	var (
		search  string
		limit   int
		noCache bool
	)

	cmd := &cobra.Command{
		Use:   "crawls",
		Short: "List available Common Crawl datasets",
		Long: `List available Common Crawl datasets from the cached crawl catalog or the CC API.

Shows local status for each crawl (indexed/parquet/cached directory presence) so you
can quickly choose a crawl for ` + "`cc parquet`" + `, ` + "`cc index`" + `, or ` + "`cc recrawl`" + `.`,
		Example: `  search cc crawls
  search cc crawls --limit 10
  search cc crawls --search 2026-08
  search cc crawls --no-cache`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCCrawls(cmd.Context(), search, limit, noCache)
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "Filter crawls by ID")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max crawls to display")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Bypass cache")

	return cmd
}

func runCCCrawls(ctx context.Context, search string, limit int, noCache bool) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl Datasets"))
	fmt.Println()

	cfg := cc.DefaultConfig()
	cache := cc.NewCache(cfg.DataDir)
	cacheSource := "api"

	// Try cache first
	var crawls []cc.Crawl
	if !noCache {
		if cd := cache.Load(); cache.IsFresh(cd) && len(cd.Crawls) > 0 {
			crawls = cd.Crawls
			cacheSource = "cache"
			fmt.Println(labelStyle.Render("  Using cached crawl catalog"))
		}
	}

	if len(crawls) == 0 {
		client := cc.NewClient("", 4)
		var err error
		crawls, err = client.ListCrawls(ctx)
		if err != nil {
			return fmt.Errorf("listing crawls: %w", err)
		}
		cacheSource = "api"

		// Update cache
		cd := cache.Load()
		if cd == nil {
			cd = &cc.CacheData{}
		}
		cd.Crawls = crawls
		for _, c := range crawls {
			if c.ID > cd.LatestCrawlID {
				cd.LatestCrawlID = c.ID
			}
		}
		cd.FetchedAt = time.Now()
		cache.Save(cd)
	}

	// Filter
	if search != "" {
		var filtered []cc.Crawl
		for _, c := range crawls {
			if strings.Contains(strings.ToLower(c.ID), strings.ToLower(search)) ||
				strings.Contains(strings.ToLower(c.Name), strings.ToLower(search)) {
				filtered = append(filtered, c)
			}
		}
		crawls = filtered
	}
	matchedCount := len(crawls)

	if limit > 0 && len(crawls) > limit {
		crawls = crawls[:limit]
	}

	// Check local data
	dataDir := cfg.DataDir

	summary := ccRenderKVCard("Crawl Catalog", [][2]string{
		{"Source", ccStatusChip("info", cacheSource)},
		{"Filter", func() string {
			if strings.TrimSpace(search) == "" {
				return labelStyle.Render("(none)")
			}
			return infoStyle.Render(search)
		}()},
		{"Showing", ccFmtInt64(int64(len(crawls)))},
		{"Matched", ccFmtInt64(int64(matchedCount))},
	})
	fmt.Println(summary)
	fmt.Println()

	latestID := ""
	for _, c := range crawls {
		if c.ID > latestID {
			latestID = c.ID
		}
	}

	var rows [][]string
	for _, c := range crawls {
		fromStr := "---"
		toStr := "---"
		if !c.From.IsZero() {
			fromStr = c.From.Format("2006-01-02")
		}
		if !c.To.IsZero() {
			toStr = c.To.Format("2006-01-02")
		}

		localStatus := ccStatusChip("muted", "---")
		crawlDir := filepath.Join(dataDir, c.ID)
		if fi, err := os.Stat(crawlDir); err == nil && fi.IsDir() {
			if _, err := os.Stat(filepath.Join(crawlDir, "index.duckdb")); err == nil {
				localStatus = ccStatusChip("ok", "indexed")
			} else {
				tmpCfg := cc.DefaultConfig()
				tmpCfg.CrawlID = c.ID
				if files, ferr := cc.LocalParquetFiles(tmpCfg); ferr == nil && len(files) > 0 {
					localStatus = ccStatusChip("info", fmt.Sprintf("%d parquet", len(files)))
				} else {
					entries, _ := os.ReadDir(filepath.Join(crawlDir, "index"))
					if len(entries) > 0 {
						localStatus = ccStatusChip("info", fmt.Sprintf("%d entries", len(entries)))
					} else {
						localStatus = ccStatusChip("warn", "dir only")
					}
				}
			}
		}

		idCell := c.ID
		if c.ID == latestID {
			idCell = c.ID + " " + ccStatusChip("info", "latest")
		}
		rows = append(rows, []string{
			idCell,
			trimMiddle(c.Name, 36),
			fromStr,
			toStr,
			localStatus,
		})
	}

	fmt.Println(infoStyle.Render("Available Crawls"))
	fmt.Println(ccRenderTable([]string{"ID", "Name", "From", "To", "Local"}, rows, ccTableOptions{
		RightAlignCols: map[int]bool{},
	}))
	if len(crawls) < matchedCount {
		fmt.Println()
		fmt.Println(labelStyle.Render(fmt.Sprintf("Showing %d of %d matched crawls (use --limit 0 for all)", len(crawls), matchedCount)))
	} else {
		fmt.Println()
		fmt.Println(labelStyle.Render(fmt.Sprintf("Showing %d crawls", len(crawls))))
	}
	return nil
}

// ── cc index ──────────────────────────────────────────────

func newCCIndex() *cobra.Command {
	var (
		crawlID    string
		importOnly bool
		workers    int
		sample     int
	)

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Download + import columnar index to DuckDB",
		Long: `Download columnar index parquet files and import to DuckDB.

Use --sample N to download only N evenly-spaced parquet files instead of all ~900.
Each file is ~220MB and contains ~2.5M records. For most queries, 1-10 files suffice.

  --sample 1   ~220MB disk, ~2.5M records  (quick exploration)
  --sample 5   ~1.1GB disk, ~12.5M records (representative sample)
  --sample 20  ~4.4GB disk, ~50M records   (substantial coverage)
  --sample 0   ~200GB disk, ~2.3B records  (full index)`,
		Example: `  search cc index --sample 1
  search cc index --sample 5
  search cc index --crawl CC-MAIN-2026-08 --sample 20
  search cc index --import-only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCIndex(cmd.Context(), crawlID, importOnly, workers, sample)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest cached/latest available)")
	cmd.Flags().BoolVar(&importOnly, "import-only", false, "Skip download, import existing parquet files")
	cmd.Flags().IntVar(&workers, "workers", 10, "Concurrent download workers")
	cmd.Flags().IntVar(&sample, "sample", 5, "Download only N parquet files (0 = all ~900)")

	return cmd
}

func runCCIndex(ctx context.Context, crawlID string, importOnly bool, workers, sample int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl Index"))
	fmt.Println()

	resolvedCrawlID, crawlNote, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedCrawlID
	if crawlNote != "" {
		fmt.Println(labelStyle.Render("Using defaults"))
		ccPrintDefaultCrawlResolution(crawlID, crawlNote)
		fmt.Println()
	}

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	cfg.IndexWorkers = workers

	client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)

	if !importOnly {
		if sample > 0 {
			fmt.Printf("  %s\n", infoStyle.Render(fmt.Sprintf(
				"Downloading %d sampled parquet files for %s (~%dMB)...",
				sample, crawlID, sample*220)))
		} else {
			fmt.Printf("  %s\n", infoStyle.Render(fmt.Sprintf(
				"Downloading full columnar index for %s (~200GB)...", crawlID)))
		}
		fmt.Println(labelStyle.Render(fmt.Sprintf("  -> %s", cfg.IndexDir())))

		start := time.Now()
		dlReporter := newCCDownloadReporter()
		err = cc.DownloadIndex(ctx, client, cfg, sample, dlReporter.Callback)
		if err != nil {
			return fmt.Errorf("downloading index: %w", err)
		}
		fmt.Println(successStyle.Render(fmt.Sprintf("  Download complete in %s", time.Since(start).Truncate(time.Second))))
	}

	// Import to DuckDB
	fmt.Println(infoStyle.Render("Importing parquet to per-file DuckDB + catalog..."))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Parquet root: %s", cfg.IndexDir())))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Shards:      %s", cfg.IndexShardDir())))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Catalog:     %s", cfg.IndexDBPath())))

	importStart := time.Now()
	importReporter := newCCImportReporter()
	rowCount, err := cc.ImportIndexWithProgress(ctx, cfg, importReporter.Callback)
	if err != nil {
		return fmt.Errorf("importing index: %w", err)
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  Imported %s rows in %s",
		ccFmtInt64(rowCount), time.Since(importStart).Truncate(time.Second))))
	return nil
}

// ── cc stats ──────────────────────────────────────────────

func newCCStats() *cobra.Command {
	var crawlID string

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show index statistics",
		Long:  "Show summary statistics for the local Common Crawl index DuckDB (defaults to the latest crawl).",
		Example: `  search cc stats
  search cc stats --crawl CC-MAIN-2026-08`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCStats(cmd.Context(), crawlID)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest cached/latest available)")

	return cmd
}

func runCCStats(ctx context.Context, crawlID string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl Index Statistics"))
	fmt.Println()

	resolvedCrawlID, crawlNote, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedCrawlID
	if crawlNote != "" {
		fmt.Println(labelStyle.Render("Using defaults"))
		ccPrintDefaultCrawlResolution(crawlID, crawlNote)
		fmt.Println()
	}

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID

	dbPath := cfg.IndexDBPath()
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index not found at %s — run 'cc index' first", dbPath)
	}

	summary, err := cc.IndexStats(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("loading stats: %w", err)
	}

	fmt.Printf("  Crawl:           %s\n", infoStyle.Render(crawlID))
	fmt.Printf("  Index:           %s\n", labelStyle.Render(dbPath))
	fmt.Println()
	fmt.Printf("  Total records:   %s\n", ccFmtInt64(summary.TotalRecords))
	fmt.Printf("  Unique hosts:    %s\n", ccFmtInt64(summary.UniqueHosts))
	fmt.Printf("  Unique domains:  %s\n", ccFmtInt64(summary.UniqueDomains))

	// Status distribution
	fmt.Println()
	fmt.Println(infoStyle.Render("  HTTP Status Distribution:"))
	for status, count := range summary.StatusDist {
		fmt.Printf("    %d: %s\n", status, ccFmtInt64(count))
	}

	// MIME distribution
	fmt.Println()
	fmt.Println(infoStyle.Render("  Content Type Distribution:"))
	for mime, count := range summary.MimeDist {
		fmt.Printf("    %-40s %s\n", mime, ccFmtInt64(count))
	}

	// TLD distribution
	fmt.Println()
	fmt.Println(infoStyle.Render("  Top TLDs:"))
	type tldCount struct {
		tld   string
		count int64
	}
	var tlds []tldCount
	for t, c := range summary.TLDDist {
		tlds = append(tlds, tldCount{t, c})
	}
	sort.Slice(tlds, func(i, j int) bool { return tlds[i].count > tlds[j].count })
	for _, t := range tlds {
		fmt.Printf("    %-10s %s\n", t.tld, ccFmtInt64(t.count))
	}

	return nil
}

// ── cc query ──────────────────────────────────────────────

func newCCQuery() *cobra.Command {
	var (
		crawlID string
		lang    string
		mime    string
		status  int
		domain  string
		tld     string
		limit   int
		count   bool
		remote  bool
	)

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query index for matching URLs",
		Long: `Query the columnar index for URLs matching your criteria.

By default queries the local DuckDB index (requires 'cc index' first).
Use --remote to query parquet files directly from S3 — no local download needed,
but slower (network-bound). Ideal for quick lookups on limited disk.

`,
		Example: `  search cc query --lang eng --status 200 --limit 100
  search cc query --remote --domain example.com --limit 10
  search cc query --tld com --mime text/html --count
  search cc query --crawl CC-MAIN-2026-08 --domain openai.com --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCQuery(cmd.Context(), crawlID, lang, mime, status, domain, tld, limit, count, remote)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest cached/latest available)")
	cmd.Flags().StringVar(&lang, "lang", "", "Language filter (e.g. eng)")
	cmd.Flags().StringVar(&mime, "mime", "", "MIME type filter (e.g. text/html)")
	cmd.Flags().IntVar(&status, "status", 0, "HTTP status filter (e.g. 200)")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain filter")
	cmd.Flags().StringVar(&tld, "tld", "", "TLD filter (e.g. com)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Max results")
	cmd.Flags().BoolVar(&count, "count", false, "Show count only")
	cmd.Flags().BoolVar(&remote, "remote", false, "Query S3 parquet directly (no local download needed)")

	return cmd
}

func runCCQuery(ctx context.Context, crawlID, lang, mime string, status int, domain, tld string, limit int, countOnly, remote bool) error {
	if resolvedCrawlID, _, err := ccResolveCrawlID(ctx, crawlID); err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	} else {
		crawlID = resolvedCrawlID
	}

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID

	filter := cc.IndexFilter{Limit: limit}
	if lang != "" {
		filter.Languages = strings.Split(lang, ",")
	}
	if mime != "" {
		filter.MimeTypes = strings.Split(mime, ",")
	}
	if status > 0 {
		filter.StatusCodes = []int{status}
	}
	if domain != "" {
		filter.Domains = strings.Split(domain, ",")
	}
	if tld != "" {
		filter.TLDs = strings.Split(tld, ",")
	}

	if remote {
		fmt.Println(infoStyle.Render("Querying S3 parquet directly (no local index needed)..."))
		pointers, err := cc.QueryRemoteParquet(ctx, cfg, filter)
		if err != nil {
			return fmt.Errorf("remote query: %w", err)
		}
		if countOnly {
			fmt.Printf("  Matching records: %s\n", ccFmtInt64(int64(len(pointers))))
			return nil
		}
		return displayQueryResults(pointers)
	}

	// Local query
	dbPath := cfg.IndexDBPath()
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index not found — run 'cc index' first (or use --remote)")
	}

	if countOnly {
		n, err := cc.QueryIndexCount(ctx, dbPath, filter)
		if err != nil {
			return err
		}
		fmt.Printf("  Matching records: %s\n", ccFmtInt64(n))
		return nil
	}

	pointers, err := cc.QueryIndex(ctx, dbPath, filter)
	if err != nil {
		return err
	}
	return displayQueryResults(pointers)
}

func displayQueryResults(pointers []cc.WARCPointer) error {
	fmt.Printf("  %-80s %-6s %-20s %s\n", "URL", "Status", "Content-Type", "Language")
	fmt.Println(strings.Repeat("-", 140))

	for _, p := range pointers {
		url := p.URL
		if len(url) > 80 {
			url = url[:77] + "..."
		}
		ct := p.ContentType
		if len(ct) > 20 {
			ct = ct[:17] + "..."
		}
		lang := p.Language
		if len(lang) > 10 {
			lang = lang[:10]
		}
		fmt.Printf("  %-80s %-6d %-20s %s\n", url, p.FetchStatus, ct, lang)
	}

	fmt.Printf("\n  %s\n", labelStyle.Render(fmt.Sprintf("Showing %d results", len(pointers))))
	return nil
}

// ── cc fetch ──────────────────────────────────────────────

func newCCFetch() *cobra.Command {
	var (
		crawlID string
		lang    string
		mime    string
		status  int
		domain  string
		tld     string
		limit   int
		workers int
		timeout int
		resume  bool
		remote  bool
	)

	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "High-throughput page extraction from WARC files",
		Long: `Fetch pages from Common Crawl WARC files via byte-range requests.

First queries the index for matching URLs, then fetches WARC records
in parallel from the CDN. Extracted pages are stored in sharded DuckDB files.

Use --remote to query the index from S3 directly (no local parquet needed).

`,
		Example: `  search cc fetch --lang eng --mime text/html --limit 10000
  search cc fetch --remote --domain example.com --limit 100
  search cc fetch --status 200 --workers 5000 --limit 1000000
  search cc fetch --crawl CC-MAIN-2026-08 --domain openai.com --limit 500`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFetch(cmd.Context(), crawlID, lang, mime, status, domain, tld, limit, workers, timeout, resume, remote)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest cached/latest available)")
	cmd.Flags().StringVar(&lang, "lang", "", "Language filter (e.g. eng)")
	cmd.Flags().StringVar(&mime, "mime", "text/html", "MIME type filter")
	cmd.Flags().IntVar(&status, "status", 200, "HTTP status filter")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain filter")
	cmd.Flags().StringVar(&tld, "tld", "", "TLD filter (e.g. com)")
	cmd.Flags().IntVar(&limit, "limit", 10000, "Max records to fetch")
	cmd.Flags().IntVar(&workers, "workers", 5000, "Concurrent fetch workers")
	cmd.Flags().IntVar(&timeout, "timeout", 30000, "Per-request timeout in ms")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip already-fetched records")
	cmd.Flags().BoolVar(&remote, "remote", false, "Query S3 index directly (no local parquet needed)")

	return cmd
}

func runCCFetch(ctx context.Context, crawlID, lang, mime string, status int, domain, tld string,
	limit, workers, timeout int, resume, remote bool) error {

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl WARC Page Extraction"))
	fmt.Println()

	resolvedCrawlID, crawlNote, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedCrawlID
	if crawlNote != "" {
		fmt.Println(labelStyle.Render("Using defaults"))
		ccPrintDefaultCrawlResolution(crawlID, crawlNote)
		fmt.Println()
	}

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	cfg.Workers = workers
	cfg.Timeout = time.Duration(timeout) * time.Millisecond
	cfg.Resume = resume

	// Build filter
	filter := cc.IndexFilter{Limit: limit}
	if lang != "" {
		filter.Languages = strings.Split(lang, ",")
	}
	if mime != "" {
		filter.MimeTypes = strings.Split(mime, ",")
	}
	if status > 0 {
		filter.StatusCodes = []int{status}
	}
	if domain != "" {
		filter.Domains = strings.Split(domain, ",")
	}
	if tld != "" {
		filter.TLDs = strings.Split(tld, ",")
	}

	// Query WARC pointers (local or remote)
	var pointers []cc.WARCPointer

	if remote {
		fmt.Println(infoStyle.Render("Querying S3 parquet index directly..."))
		pointers, err = cc.QueryRemoteParquet(ctx, cfg, filter)
	} else {
		dbPath := cfg.IndexDBPath()
		if _, err := os.Stat(dbPath); err != nil {
			return fmt.Errorf("index not found — run 'cc index' first (or use --remote)")
		}

		fmt.Println(infoStyle.Render("Querying local index..."))
		totalCount, err2 := cc.QueryIndexCount(ctx, dbPath, filter)
		if err2 != nil {
			return fmt.Errorf("counting records: %w", err2)
		}
		fmt.Println(successStyle.Render(fmt.Sprintf("  %s matching records (fetching up to %s)",
			ccFmtInt64(totalCount), ccFmtInt64(int64(limit)))))

		pointers, err = cc.QueryIndex(ctx, dbPath, filter)
	}

	if err != nil {
		return fmt.Errorf("querying index: %w", err)
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  Loaded %s pointers", ccFmtInt64(int64(len(pointers))))))

	if len(pointers) == 0 {
		fmt.Println(warningStyle.Render("  No matching records found"))
		return nil
	}

	// Check for resume
	var skip map[string]bool
	if resume {
		fmt.Println(infoStyle.Render("Checking for previous results..."))
		skip, err = cc.LoadAlreadyFetched(ctx, cfg.ResultDir())
		if err != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Could not load state: %v", err)))
		} else if len(skip) > 0 {
			fmt.Println(successStyle.Render(fmt.Sprintf("  Resuming: skipping %d already-fetched records", len(skip))))
		}
	}

	// Open result DB
	fmt.Println(infoStyle.Render("Opening result databases..."))
	rdb, err := cc.NewResultDB(cfg.ResultDir(), 8, cfg.BatchSize)
	if err != nil {
		return fmt.Errorf("opening result db: %w", err)
	}
	defer func() {
		if rdb != nil {
			_ = rdb.Close()
		}
	}()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Results -> %s/ (8 shards)", cfg.ResultDir())))

	rdb.SetMeta(ctx, "crawl_id", crawlID)
	rdb.SetMeta(ctx, "started_at", time.Now().Format(time.RFC3339))
	rdb.SetMeta(ctx, "workers", fmt.Sprintf("%d", workers))

	// Create client and fetcher
	client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)
	stats := cc.NewFetchStats(len(pointers), crawlID)
	fetcher := cc.NewFetcher(cfg, client, stats, rdb)

	fmt.Println()
	fmt.Println(infoStyle.Render(fmt.Sprintf("Starting fetch: %d workers, %v timeout, %d records",
		workers, cfg.Timeout, len(pointers))))
	fmt.Println()

	// Run with display
	err = cc.RunWithDisplay(ctx, fetcher, pointers, skip, stats)

	// Final flush
	rdb.Flush(ctx)
	rdb.SetMeta(ctx, "finished_at", time.Now().Format(time.RFC3339))

	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("Fetch finished with error: %v", err)))
	} else {
		fmt.Println(successStyle.Render("Fetch complete!"))
	}
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Results: %s/", cfg.ResultDir())))
	fmt.Println()

	return err
}

// ── cc url ──────────────────────────────────────────────

func newCCURL() *cobra.Command {
	var (
		crawlID   string
		targetURL string
		domain    string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "url",
		Short: "Lookup a URL via CDX API (zero disk, network-only)",
		Long: `Lookup URLs in Common Crawl via the CDX API.
This is the lightest-weight option: uses zero disk space, queries the CC API directly.

`,
		Example: `  search cc url --url https://example.com
  search cc url --domain example.com --limit 50
  search cc url --crawl CC-MAIN-2026-08 --domain openai.com --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCURL(cmd.Context(), crawlID, targetURL, domain, limit)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest cached/latest available)")
	cmd.Flags().StringVar(&targetURL, "url", "", "URL to lookup")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain to lookup (all URLs)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results for domain lookup")

	return cmd
}

func runCCURL(ctx context.Context, crawlID, targetURL, domain string, limit int) error {
	if targetURL == "" && domain == "" {
		return fmt.Errorf("--url or --domain is required")
	}
	if resolvedCrawlID, _, err := ccResolveCrawlID(ctx, crawlID); err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	} else {
		crawlID = resolvedCrawlID
	}

	var entries []cc.CDXJEntry
	var err error

	if targetURL != "" {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Looking up %s in %s...", targetURL, crawlID)))
		entries, err = cc.LookupURL(ctx, crawlID, targetURL)
	} else {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Looking up %s/* in %s...", domain, crawlID)))
		entries, err = cc.LookupDomain(ctx, crawlID, domain, limit)
	}

	if err != nil {
		return fmt.Errorf("CDX lookup: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println(warningStyle.Render("  No results found"))
		return nil
	}

	fmt.Println()
	fmt.Printf("  %-80s %-6s %-20s %s\n", "URL", "Status", "MIME", "Timestamp")
	fmt.Println(strings.Repeat("-", 130))

	for _, e := range entries {
		url := e.URL
		if len(url) > 80 {
			url = url[:77] + "..."
		}
		fmt.Printf("  %-80s %-6s %-20s %s\n", url, e.Status, e.Mime, e.Timestamp)
	}

	fmt.Printf("\n  %s\n", labelStyle.Render(fmt.Sprintf("Found %d results", len(entries))))
	return nil
}

// ── cc recrawl ──────────────────────────────────────────────

func newCCRecrawl() *cobra.Command {
	var (
		crawlID             string
		sample              int
		last                bool
		file                string
		importOnly          bool
		workers             int
		dnsWorkers          int
		dnsTimeout          int
		timeout             int
		statusOnly          bool
		headOnly            bool
		transportShards     int
		maxConnsPerDomain   int
		dnsPrefetch         bool
		reloadDNSCache      bool
		resume              bool
		lang                string
		mime                string
		status              int
		domain              string
		tld                 string
		limit               int
		batchSize           int
		engine              string
		domainTimeoutMs      int
		domainFailThreshold  int
		domainDeadProbe      int
		domainStallRatio     int
		domainTimeoutMaxMs   int
		retryTimeoutMs       int
		noRetry              bool
		dbMemMB              int
		dbShards             int
		chunkMode            string
	)

	cmd := &cobra.Command{
		Use:   "recrawl",
		Short: "Download CC index parquet, extract URLs, recrawl from origin servers",
		Long: `Combined pipeline: CC index → URL extraction → high-throughput recrawl.

Three modes for loading the CC index:

  --last         Download the LAST (latest) parquet file, query directly via
                 read_parquet() — zero DuckDB import, fastest startup (recommended)
  --file X       Use parquet selector or local path, query directly via read_parquet()
                 selectors: N (warc/part index), p:N, w:N, m:N (manifest index)
  --sample N     Download N evenly-spaced parquet files, import to DuckDB (legacy)

Pipeline:
  1. Download parquet file(s) from CC columnar index (~220MB each, ~2.5M URLs)
  2. Extract URLs matching your filters (direct parquet query or DuckDB)
  3. Batch DNS pre-resolution
  4. HTTP recrawl from origin servers (throughput depends on timeout/concurrency tuning)

This fetches FRESH content from origin servers (not cached WARC data).
Use 'cc fetch' instead if you want pre-crawled content from WARC files.

Examples:
  search cc recrawl --last --status-only
  search cc recrawl --last --status-only --workers 2000
  search cc recrawl --file 0 --status-only --limit 1000          # warc index, latest crawl by default
  search cc recrawl --file p:0 --status-only --limit 1000        # part index (part-00000) in warc subset
  search cc recrawl --file m:600 --status-only --limit 1000      # global manifest index
  search cc recrawl --file /path/to/local.parquet --status-only
  search cc recrawl --sample 1 --status-only --workers 2000
  search cc recrawl --sample 1 --lang eng --mime text/html --workers 200
  search cc recrawl --import-only --resume --workers 2000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := cc.IndexFilter{}
			if lang != "" {
				filter.Languages = strings.Split(lang, ",")
			}
			if mime != "" {
				filter.MimeTypes = strings.Split(mime, ",")
			}
			if status > 0 {
				filter.StatusCodes = []int{status}
			}
			if domain != "" {
				filter.Domains = strings.Split(domain, ",")
			}
			if tld != "" {
				filter.TLDs = strings.Split(tld, ",")
			}
			if limit > 0 {
				filter.Limit = limit
			}

			return runCCRecrawl(cmd.Context(), ccRecrawlOpts{
				crawlID:            crawlID,
				sample:             sample,
				last:               last,
				file:               file,
				importOnly:         importOnly,
				filter:             filter,
				workers:            workers,
				workersExplicit:    cmd.Flags().Changed("workers"),
				dnsWorkers:         dnsWorkers,
				dnsTimeout:         dnsTimeout,
				dnsWorkersExplicit: cmd.Flags().Changed("dns-workers"),
				dnsTimeoutExplicit: cmd.Flags().Changed("dns-timeout"),
				timeout:            timeout,
				statusOnly:         statusOnly,
				headOnly:           headOnly,
				transportShards:    transportShards,
				maxConnsPerDomain:  maxConnsPerDomain,
				dnsPrefetch:        dnsPrefetch,
				reloadDNSCache:     reloadDNSCache,
				resume:             resume,
				batchSize:          batchSize,
				engine:              engine,
			domainTimeoutMs:     domainTimeoutMs,
			domainFailThreshold: domainFailThreshold,
			domainDeadProbe:     domainDeadProbe,
			domainStallRatio:    domainStallRatio,
			domainTimeoutMaxMs:  domainTimeoutMaxMs,
			retryTimeoutMs:      retryTimeoutMs,
			noRetry:             noRetry,
			dbMemMB:             dbMemMB,
			dbShards:            dbShards,
			chunkMode:           chunkMode,
		})
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest cached/latest available)")
	cmd.Flags().BoolVar(&last, "last", false, "Download last (latest) parquet file, query directly (recommended)")
	cmd.Flags().StringVar(&file, "file", "", "Parquet selector/path: N (warc/part index), p:N, w:N, m:N, or local path")
	cmd.Flags().IntVar(&sample, "sample", 1, "Number of parquet files to download (0=all, legacy mode)")
	cmd.Flags().BoolVar(&importOnly, "import-only", false, "Skip parquet download, use existing DuckDB index")
	cmd.Flags().IntVar(&workers, "workers", -1, "HTTP fetch workers (-1=auto from hardware)")
	cmd.Flags().IntVar(&dnsWorkers, "dns-workers", 2000, "DNS resolution workers (use 0 for auto)")
	cmd.Flags().IntVar(&dnsTimeout, "dns-timeout", 2000, "DNS timeout in ms (use 0 for auto)")
	cmd.Flags().IntVar(&timeout, "timeout", 5000, "HTTP timeout in ms")
	cmd.Flags().BoolVar(&statusOnly, "status-only", false, "Only check HTTP status (fastest)")
	cmd.Flags().BoolVar(&headOnly, "head-only", false, "HEAD requests only")
	cmd.Flags().IntVar(&transportShards, "transport-shards", 64, "HTTP transport pool shards")
	cmd.Flags().IntVar(&maxConnsPerDomain, "max-conns-per-domain", 8, "Max concurrent connections per domain (prevents server flooding)")
	cmd.Flags().BoolVar(&dnsPrefetch, "dns-prefetch", true, "Batch DNS pre-resolution")
	cmd.Flags().BoolVar(&reloadDNSCache, "reload-dns-cache", false, "Ignore cached dns.duckdb and resolve DNS again from network")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip already-crawled URLs")
	cmd.Flags().StringVar(&lang, "lang", "", "Language filter (e.g. eng)")
	cmd.Flags().StringVar(&mime, "mime", "", "MIME type filter (e.g. text/html)")
	cmd.Flags().IntVar(&status, "status", 200, "HTTP status filter from CC index")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain filter")
	cmd.Flags().StringVar(&tld, "tld", "", "TLD filter (e.g. com)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max URLs to recrawl (0=all from index)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "DB write batch size")
	cmd.Flags().StringVar(&engine, "engine", "keepalive", "Crawl engine: keepalive|epoll|swarm|rawhttp")
	cmd.Flags().IntVar(&domainTimeoutMs, "domain-timeout", -1, "Per-domain context deadline in ms; 0=disabled, -1=adaptive (2×sweep time, clamped [30s,10min])")
	cmd.Flags().IntVar(&domainFailThreshold, "domain-fail-threshold", -1, "Abandon domain after this many timeout rounds (×conns); -1=engine default (3)")
	cmd.Flags().IntVar(&domainDeadProbe, "domain-dead-probe", 10, "Abandon domain after this many consecutive timeouts with 0 successes (0=disabled)")
	cmd.Flags().IntVar(&domainStallRatio, "domain-stall-ratio", 20, "Abandon domain when timeouts ≥ successes×ratio after dead-probe window (0=disabled; 20 = >95% timeout rate)")
	cmd.Flags().IntVar(&domainTimeoutMaxMs, "domain-timeout-max", 0, "Upper bound for adaptive per-domain timeout in ms (0=default 10min); lower values free workers faster from slow domains")
	cmd.Flags().IntVar(&retryTimeoutMs, "retry-timeout", 10000, "Pass-2 timeout in ms for retrying timeout URLs (0=disabled)")
	cmd.Flags().BoolVar(&noRetry, "no-retry", false, "Skip pass-2 retry of timeout URLs")
	cmd.Flags().IntVar(&dbMemMB, "db-mem-mb", 0, "DuckDB memory per shard in MB (0=auto: 15% avail RAM / shards)")
	cmd.Flags().IntVar(&dbShards, "db-shards", 0, "ResultDB shard count (0=auto: clamp(CPUs×2, 4, 16))")
	cmd.Flags().StringVar(&chunkMode, "chunk-mode", "stream", "Seed delivery mode: stream|pipeline|batch (stream: sort-then-stream; pipeline: low-memory cursor from seed DB, use for >1M seeds to prevent OOM; batch: N-domain chunks)")

	return cmd
}

type ccRecrawlOpts struct {
	crawlID             string
	sample              int
	last                bool
	file                string
	importOnly          bool
	filter              cc.IndexFilter
	workers             int
	workersExplicit     bool
	dnsWorkers          int
	dnsTimeout          int
	dnsWorkersExplicit  bool
	dnsTimeoutExplicit  bool
	timeout             int
	statusOnly          bool
	headOnly            bool
	transportShards     int
	maxConnsPerDomain   int
	dnsPrefetch         bool
	reloadDNSCache      bool
	resume              bool
	batchSize           int
	engine              string
	domainTimeoutMs     int
	domainFailThreshold int
	domainDeadProbe      int
	domainStallRatio     int
	domainTimeoutMaxMs   int
	retryTimeoutMs       int
	noRetry              bool
	dbMemMB              int
	dbShards             int
	chunkMode            string
	seedDBPath           string // set when seeds are pre-materialized for pipeline mode
	totalSeeds           int64  // pre-extraction count for pipeline mode coverage display
}

func runCCRecrawl(ctx context.Context, opts ccRecrawlOpts) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("CC Index → Recrawl Pipeline"))
	fmt.Println()

	resolvedCrawlID, crawlNote, err := ccResolveCrawlID(ctx, opts.crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	opts.crawlID = resolvedCrawlID

	ccCfg := cc.DefaultConfig()
	ccCfg.CrawlID = opts.crawlID

	if !opts.workersExplicit && opts.workers > 10000 {
		fmt.Println(warningStyle.Render(fmt.Sprintf("Clamping workers from %d to 10000 to avoid local socket exhaustion", opts.workers)))
		fmt.Println(labelStyle.Render("  Use --workers to override if you intentionally tuned OS limits for higher concurrency"))
		fmt.Println()
		opts.workers = 10000
	}

	if crawlNote != "" {
		fmt.Println(labelStyle.Render(fmt.Sprintf("Using defaults")))
		ccPrintDefaultCrawlResolution(opts.crawlID, crawlNote)
		fmt.Println()
	}

	// Determine mode: --last, --file, or --sample (legacy)
	mode := "sample" // default legacy mode
	if opts.last {
		mode = "last"
	} else if opts.file != "" {
		mode = "file"
	} else if opts.importOnly {
		mode = "sample" // import-only always uses DuckDB path
	}

	var seeds []recrawler.SeedURL
	var uniqueDomains int
	var sourceParquetPath string

	switch mode {
	case "last", "file":
		// ── Direct parquet mode (--last or --file) ────────────────
		var parquetPath string

		if mode == "last" {
			fmt.Println(infoStyle.Render(fmt.Sprintf("Step 1: Downloading LAST parquet file for %s (~220MB)...", opts.crawlID)))
			client := cc.NewClient(ccCfg.BaseURL, ccCfg.TransportShards)
			start := time.Now()
			dlReporter := newCCDownloadReporter()
			parquetPath, err = cc.DownloadOneIndexFile(ctx, client, ccCfg, -1, dlReporter.Callback)
			if err != nil {
				return fmt.Errorf("downloading last parquet: %w", err)
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("  Download complete (%s)", time.Since(start).Truncate(time.Second))))
		} else {
			// --file: local path or selector (N, p:N, w:N, m:N)
			fmt.Println(infoStyle.Render(fmt.Sprintf("Step 1: Resolving parquet selector for %s...", opts.crawlID)))
			start := time.Now()
			dlReporter := newCCDownloadReporter()
			resolved, resolveErr := ccResolveRecrawlParquetFile(ctx, ccCfg, opts.file, dlReporter.Callback)
			if resolveErr != nil {
				return resolveErr
			}
			parquetPath = resolved.Path
			switch {
			case resolved.Source == "local":
				fmt.Println(labelStyle.Render(fmt.Sprintf("  Local file: %s", parquetPath)))
			case resolved.Cached:
				fmt.Println(successStyle.Render(fmt.Sprintf("  Using cached parquet (%s)", ccFmtBytes(resolved.SizeBytes))))
				fmt.Println(labelStyle.Render(fmt.Sprintf("  Selector: %s → %s", opts.file, resolved.Detail)))
				fmt.Println(labelStyle.Render(fmt.Sprintf("  Local:    %s", parquetPath)))
			default:
				fmt.Println(successStyle.Render(fmt.Sprintf("  Download complete (%s)", time.Since(start).Truncate(time.Second))))
				fmt.Println(labelStyle.Render(fmt.Sprintf("  Selector: %s → %s", opts.file, resolved.Detail)))
				fmt.Println(labelStyle.Render(fmt.Sprintf("  Local:    %s", parquetPath)))
			}
		}
		fmt.Println()
		sourceParquetPath = parquetPath

		// ── Step 2: Extract URLs from parquet ──
		// pipeline mode: materialize to DuckDB (no Go heap) — use for >1M seeds to prevent OOM.
		// stream mode (default): load all seeds into memory (faster for small-medium runs).
		printFilterSummary(opts.filter)
		extractStart := time.Now()

		if opts.chunkMode == "pipeline" {
			// Compute run dir early so we can place seeds.duckdb next to result/failed DBs.
			runDir := ccRecrawlParquetRunDir(ccCfg, sourceParquetPath)
			seedDBPath := filepath.Join(runDir, "seeds.duckdb")
			if err := os.MkdirAll(runDir, 0o755); err != nil {
				return fmt.Errorf("creating run dir: %w", err)
			}
			fmt.Println(infoStyle.Render("Step 2: Materializing seeds to DuckDB (pipeline mode, no Go heap)..."))
			var seedCount int
			seedCount, uniqueDomains, err = cc.WriteSeedURLsFromParquet(ctx, parquetPath, opts.filter, seedDBPath)
			if err != nil {
				return fmt.Errorf("materializing seeds: %w", err)
			}
			if seedCount == 0 {
				fmt.Println(warningStyle.Render("  No matching URLs found in parquet"))
				return nil
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("  %s URLs across %s domains → %s (%s)",
				ccFmtInt64(int64(seedCount)), ccFmtInt64(int64(uniqueDomains)),
				filepath.Base(seedDBPath),
				time.Since(extractStart).Truncate(time.Millisecond))))
			opts.seedDBPath = seedDBPath
			opts.totalSeeds = int64(seedCount)
		} else {
			fmt.Println(infoStyle.Render("Step 2: Extracting URLs directly from parquet (zero DuckDB import)..."))
			seeds, uniqueDomains, err = cc.ExtractSeedURLsFromParquet(ctx, parquetPath, opts.filter)
			if err != nil {
				return fmt.Errorf("extracting seeds from parquet: %w", err)
			}
			if len(seeds) == 0 {
				fmt.Println(warningStyle.Render("  No matching URLs found in parquet"))
				return nil
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("  %s URLs across %s domains (%s)",
				ccFmtInt64(int64(len(seeds))), ccFmtInt64(int64(uniqueDomains)),
				time.Since(extractStart).Truncate(time.Millisecond))))
		}
		fmt.Println()

	default:
		// ── Legacy sample mode (download + import + query DuckDB) ──

		// Step 1: Download parquet file(s)
		if !opts.importOnly {
			client := cc.NewClient(ccCfg.BaseURL, ccCfg.TransportShards)

			if opts.sample > 0 {
				fmt.Println(infoStyle.Render(fmt.Sprintf("Step 1: Downloading %d parquet file(s) for %s (~%dMB)...",
					opts.sample, opts.crawlID, opts.sample*220)))
			} else {
				fmt.Println(infoStyle.Render(fmt.Sprintf("Step 1: Downloading full index for %s (~200GB)...", opts.crawlID)))
			}
			fmt.Println(labelStyle.Render(fmt.Sprintf("  → %s", ccCfg.IndexDir())))

			start := time.Now()
			dlReporter := newCCDownloadReporter()
			err := cc.DownloadIndex(ctx, client, ccCfg, opts.sample, dlReporter.Callback)
			if err != nil {
				return fmt.Errorf("downloading index: %w", err)
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("  Download complete (%s)", time.Since(start).Truncate(time.Second))))
			fmt.Println()
		}

		// Step 2: Import to DuckDB
		dbPath := ccCfg.IndexDBPath()

		needsImport := true
		if _, err := os.Stat(dbPath); err == nil {
			fmt.Println(infoStyle.Render("Step 2: Index already imported, skipping..."))
			fmt.Println(labelStyle.Render(fmt.Sprintf("  → %s", dbPath)))
			needsImport = false
		}

		if needsImport {
			fmt.Println(infoStyle.Render("Step 2: Importing parquet to per-file DuckDB + catalog..."))
			fmt.Println(labelStyle.Render(fmt.Sprintf("  Parquet root: %s", ccCfg.IndexDir())))
			fmt.Println(labelStyle.Render(fmt.Sprintf("  Shards:      %s", ccCfg.IndexShardDir())))
			fmt.Println(labelStyle.Render(fmt.Sprintf("  Catalog:     %s", dbPath)))

			importStart := time.Now()
			importReporter := newCCImportReporter()
			rowCount, importErr := cc.ImportIndexWithProgress(ctx, ccCfg, importReporter.Callback)
			if importErr != nil {
				return fmt.Errorf("importing index: %w", importErr)
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("  Imported %s rows (%s)",
				ccFmtInt64(rowCount), time.Since(importStart).Truncate(time.Second))))
		}
		fmt.Println()

		// Step 3: Extract URLs
		fmt.Println(infoStyle.Render("Step 3: Extracting URLs from CC index..."))
		printFilterSummary(opts.filter)

		seeds, uniqueDomains, err = cc.ExtractSeedURLs(ctx, dbPath, opts.filter)
		if err != nil {
			return fmt.Errorf("extracting seeds: %w", err)
		}
		if len(seeds) == 0 {
			fmt.Println(warningStyle.Render("  No matching URLs found in index"))
			return nil
		}
		fmt.Println(successStyle.Render(fmt.Sprintf("  %s URLs across %s domains",
			ccFmtInt64(int64(len(seeds))), ccFmtInt64(int64(uniqueDomains)))))
		fmt.Println()
	}

	// ── Step 4: Batch DNS pre-resolution ────────────────────────
	dnsTimeout := time.Duration(opts.dnsTimeout) * time.Millisecond
	dnsWorkers := opts.dnsWorkers

	resultDir := ccCfg.RecrawlDir()
	dnsPath := ccCfg.DNSCachePath()
	failedDBPath := ccCfg.FailedDBPath()

	if sourceParquetPath != "" {
		runDir := ccRecrawlParquetRunDir(ccCfg, sourceParquetPath)
		resultDir = filepath.Join(runDir, "results")
		dnsPath = filepath.Join(runDir, "dns.duckdb")
		failedDBPath = filepath.Join(runDir, "failed.duckdb")
		fmt.Println(infoStyle.Render("Per-parquet storage enabled"))
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Parquet: %s", sourceParquetPath)))
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Run dir: %s", runDir)))
		fmt.Println()
	}

	var dnsResolver *recrawler.DNSResolver
	if opts.dnsPrefetch && opts.seedDBPath != "" {
		// Pipeline mode: seeds are in a DB file, not in Go memory.
		// DNS prefetch needs to iterate hosts — load cache only; new hosts resolved at crawl time via DomainDeadProbe.
		dnsResolver = recrawler.NewDNSResolver(dnsTimeout)
		if cached, _ := dnsResolver.LoadCache(dnsPath); cached > 0 {
			fmt.Println(infoStyle.Render(fmt.Sprintf("DNS cache (pipeline mode): %d cached entries loaded", cached)))
		} else {
			fmt.Println(labelStyle.Render("DNS prefetch skipped (pipeline mode — seeds in DB; dead domains handled by DomainDeadProbe=10)"))
		}
		fmt.Println()
	} else if opts.dnsPrefetch {
		fmt.Println(infoStyle.Render("Batch DNS pre-resolution..."))

		dnsResolver = recrawler.NewDNSResolver(dnsTimeout)
		if opts.reloadDNSCache {
			fmt.Println(labelStyle.Render("  DNS cache: reload requested (ignoring existing cache file for this run)"))
			fmt.Println(labelStyle.Render("  Tip: omit --reload-dns-cache to reuse previous DNS results"))
		} else {
			cached, _ := dnsResolver.LoadCache(dnsPath)
			if cached > 0 {
				fmt.Println(successStyle.Render(fmt.Sprintf("  DNS cache loaded: %d entries (live=%d, dead=%d, timeout=%d)",
					cached, dnsResolver.LiveCount(), dnsResolver.DeadCount(), dnsResolver.TimeoutCount())))
			} else {
				fmt.Println(labelStyle.Render("  DNS cache: no reusable entries found (first run or empty cache)"))
			}
		}

		// Collect unique hosts to resolve (not just registered domains).
		// DNS cache is keyed by URL hostname used in HTTP dialing.
		allHosts := make(map[string]bool, len(seeds))
		for _, s := range seeds {
			host := strings.TrimSpace(s.Host)
			if host == "" {
				host = strings.TrimSpace(s.Domain)
			}
			if host != "" {
				allHosts[host] = true
			}
		}
		hostList := make([]string, 0, len(allHosts))
		for h := range allHosts {
			hostList = append(hostList, h)
		}

		cov := ccDNSCacheCoverage(dnsResolver, hostList)
		if cov.CachedTotal > 0 {
			fmt.Println(labelStyle.Render(fmt.Sprintf("  DNS cache coverage: %s/%s hosts (live=%s, dead=%s, timeout=%s)",
				ccFmtInt64(int64(cov.CachedTotal)), ccFmtInt64(int64(cov.Total)),
				ccFmtInt64(int64(cov.CachedResolved)), ccFmtInt64(int64(cov.CachedDead)), ccFmtInt64(int64(cov.CachedTimeout)))))
		}

		effDNSWorkers, effDNSTimeout, dnsTuneNotes := ccResolveEffectiveDNSTuning(opts, cov.Pending)
		dnsTimeout = effDNSTimeout
		if effDNSWorkers > 0 {
			dnsWorkers = effDNSWorkers
		}
		for _, note := range dnsTuneNotes {
			fmt.Println(labelStyle.Render("  " + note))
		}

		if cov.Pending == 0 {
			fmt.Println(successStyle.Render("  All hosts already covered by DNS cache; skipping network DNS lookups"))
			fmt.Println(labelStyle.Render("  Use --reload-dns-cache to force a fresh DNS pass"))
			fmt.Println()
		} else {
			baseLive := dnsResolver.LiveCount()
			baseDead := dnsResolver.DeadCount()
			baseTimeout := dnsResolver.TimeoutCount()
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Resolving %s uncached hosts (%d workers, %v timeout)...",
				ccFmtInt64(int64(cov.Pending)), dnsWorkers, dnsTimeout)))

			var dnsDisplayLines int
			live, dead, timedout := dnsResolver.ResolveBatch(ctx, hostList, dnsWorkers, dnsTimeout,
				func(p recrawler.DNSProgress) {
					if dnsDisplayLines > 0 {
						fmt.Printf("\033[%dA\033[J", dnsDisplayLines)
					}
					eta := ccDNSProgressETA(p)
					output := fmt.Sprintf("  DNS  %d/%d  │  %d live  │  %d dead  │  %d timeout  │  %.0f/s  │  ETA %s  │  %s\n",
						p.Done, p.Total, p.Live, p.Dead, p.Timeout, p.Speed, eta, p.Elapsed.Truncate(time.Millisecond))
					fmt.Print(output)
					dnsDisplayLines = 1
				})
			if dnsDisplayLines > 0 {
				fmt.Printf("\033[%dA\033[J", dnsDisplayLines)
			}
			newLive := int64(live) - baseLive
			newDead := int64(dead) - baseDead
			newTimeout := int64(timedout) - baseTimeout
			if newLive < 0 {
				newLive = 0
			}
			if newDead < 0 {
				newDead = 0
			}
			if newTimeout < 0 {
				newTimeout = 0
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("  DNS (uncached): %s live, %s dead, %s timeout (%s)",
				ccFmtInt64(newLive), ccFmtInt64(newDead), ccFmtInt64(newTimeout), dnsResolver.Duration().Truncate(time.Millisecond))))
			fmt.Println(labelStyle.Render(fmt.Sprintf("  DNS totals: live=%s, dead=%s, timeout=%s (cached=%s)",
				ccFmtInt64(int64(live)), ccFmtInt64(int64(dead)), ccFmtInt64(int64(timedout)), ccFmtInt64(dnsResolver.CachedCount()))))
			fmt.Println()
		}

		timeoutListDir := filepath.Dir(dnsPath)
		timeoutListPath, timeoutListErr := ccWriteDNSTimeoutList(timeoutListDir, dnsResolver.TimeoutDomainsWithErrors())
		if timeoutListErr != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  DNS timeout list export failed: %v", timeoutListErr)))
		} else {
			fmt.Println(labelStyle.Render(fmt.Sprintf("  DNS timeout list: %s domains → %s",
				ccFmtInt64(int64(dnsResolver.TimeoutCount())), timeoutListPath)))
			fmt.Println(labelStyle.Render("  Re-run with --reload-dns-cache to compare timeout domains after tuning"))
			fmt.Println()
		}
	}

	// ── Engine dispatch: always use keepalive (v3) ──────────────────────────
	if opts.engine == "" {
		opts.engine = "keepalive"
	}
	if err := os.MkdirAll(filepath.Dir(failedDBPath), 0755); err != nil {
		return fmt.Errorf("creating recrawl data dir: %w", err)
	}
	return runCCRecrawlV3(ctx, opts, seeds, dnsResolver, resultDir, failedDBPath)
}

// runCCRecrawlV3 delegates crawling to a recrawl_v3 engine selected by opts.engine.
// It is called from runCCRecrawl when --engine is set. failedDBPath dir must already exist.
func runCCRecrawlV3(ctx context.Context, opts ccRecrawlOpts,
	seeds []recrawler.SeedURL, dnsResolver *recrawler.DNSResolver,
	resultDir, failedDBPath string) error {

	homeDir, _ := os.UserHomeDir()
	siCache := filepath.Join(homeDir, ".cache", "search", "sysinfo.json")
	si := crawl.LoadOrGatherSysInfo(siCache, 30*time.Minute)

	dbShards := opts.dbShards
	if dbShards <= 0 {
		dbShards = crawl.AutoShardCount(si.CPUCount)
	}
	dbMemMB := opts.dbMemMB
	if dbMemMB <= 0 {
		dbMemMB = crawl.AutoDuckMemPerShard(int(si.MemAvailableMB), dbShards)
	}

	var dnsCache crawl.DNSCache
	if dnsResolver != nil {
		dnsCache = crawl.WrapDNSResolver(dnsResolver)
	} else {
		dnsCache = &crawl.NoopDNS{}
	}

	chunkMode := opts.chunkMode
	seedPath := opts.seedDBPath
	// When seeds were pre-materialized to a DB (pipeline mode), force ChunkMode=pipeline
	// so runWithChunkMode uses the SeedCursor path even if opts.chunkMode is "stream".
	if seedPath != "" {
		chunkMode = "pipeline"
	}

	jcfg := crawl.JobConfig{
		Engine:              opts.engine,
		Workers:             opts.workers,
		MaxConnsPerDomain:   opts.maxConnsPerDomain,
		Timeout:             time.Duration(opts.timeout) * time.Millisecond,
		RetryTimeout:        time.Duration(opts.retryTimeoutMs) * time.Millisecond,
		NoRetry:             opts.noRetry,
		StatusOnly:          opts.statusOnly,
		InsecureTLS:         true,
		DomainFailThreshold: opts.domainFailThreshold,
		DomainDeadProbe:     opts.domainDeadProbe,
		DomainStallRatio:    opts.domainStallRatio,
		BatchSize:           opts.batchSize,
		ChunkMode:           chunkMode,
		SeedPath:            seedPath,
	}
	if opts.domainTimeoutMs != 0 {
		jcfg.DomainTimeout = time.Duration(opts.domainTimeoutMs) * time.Millisecond // -1ms = adaptive
	}
	if opts.domainTimeoutMaxMs > 0 {
		jcfg.AdaptiveTimeoutMax = time.Duration(opts.domainTimeoutMaxMs) * time.Millisecond
	}

	// Seed count for coverage display: use pre-computed totalSeeds for pipeline mode,
	// or len(seeds) for stream mode.
	totalSeeds := opts.totalSeeds
	if totalSeeds == 0 {
		totalSeeds = int64(len(seeds))
	}

	// recrawler.SeedURL is a type alias for crawl.SeedURL — no conversion needed.
	runtime.GC()
	debug.FreeOSMemory()

	return runRecrawlJob(ctx, recrawlJobArgs{
		Seeds:        seeds,
		DNSCache:     dnsCache,
		JobCfg:       jcfg,
		ResultDir:    resultDir,
		FailedDBPath: failedDBPath,
		WriterMode:   "bin",
		SlowDomainMs: 30_000,
		DBShards:     dbShards,
		DBMemMB:      dbMemMB,
		SysInfo:      si,
		TotalSeeds:   totalSeeds,
		SeedCount:    opts.totalSeeds,
	})
}

// newCCRecrawlDrone is the hidden subcommand spawned by SwarmEngine queen processes.
// It reads seed URLs as JSON lines from stdin and crawls them using KeepAliveEngine.
func newCCRecrawlDrone() *cobra.Command {
	var (
		droneID             int
		workers             int
		timeoutMs           int
		maxConnsPerDomain   int
		statusOnly          bool
		batchSize           int
		resultDir           string
		failedDB            string
		domainFailThreshold int
		domainTimeoutMs     int
	)
	cmd := &cobra.Command{
		Use:    "recrawl-drone",
		Hidden: true,
		Short:  "Internal: drone worker for swarm engine",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = droneID
			cfg := crawl.DefaultConfig()
			cfg.Workers = workers
			cfg.Timeout = time.Duration(timeoutMs) * time.Millisecond
			cfg.MaxConnsPerDomain = maxConnsPerDomain
			cfg.StatusOnly = statusOnly
			cfg.InsecureTLS = true
			cfg.BatchSize = batchSize
			cfg.SwarmResultDir = resultDir
			cfg.SwarmFailedDB = failedDB
			cfg.DomainFailThreshold = domainFailThreshold
			if domainTimeoutMs != 0 {
				cfg.DomainTimeout = time.Duration(domainTimeoutMs) * time.Millisecond
			}
			return crawl.RunDrone(cmd.Context(), cfg)
		},
	}
	cmd.Flags().IntVar(&droneID, "drone-id", 0, "Drone index (used for log prefix)")
	cmd.Flags().IntVar(&workers, "workers", 500, "Fetch worker count")
	cmd.Flags().IntVar(&timeoutMs, "timeout", 5000, "Per-request timeout (ms)")
	cmd.Flags().IntVar(&maxConnsPerDomain, "max-conns-per-domain", 4, "Max conns per domain")
	cmd.Flags().BoolVar(&statusOnly, "status-only", false, "Status-only mode (discard body)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "DB write batch size")
	cmd.Flags().StringVar(&resultDir, "result-dir", "", "Result DB directory (required)")
	cmd.Flags().StringVar(&failedDB, "failed-db", "", "Failed URL DB path")
	cmd.Flags().IntVar(&domainFailThreshold, "domain-fail-threshold", 3, "Domain abandonment threshold")
	cmd.Flags().IntVar(&domainTimeoutMs, "domain-timeout", 0, "Per-domain timeout ms (0=disabled)")
	return cmd
}

// ── cc verify ──────────────────────────────────────────────

func newCCVerify() *cobra.Command {
	var (
		crawlID     string
		workers     int
		dnsTimeout  int
		httpTimeout int
		limit       int
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify failed domains from recrawl (slow, thorough)",
		Long: `Slowly and thoroughly verify domains marked as dead during recrawl.

For each failed domain:
  1. DNS: tries system, Google 8.8.8.8, Cloudflare 1.1.1.1 (10s timeout each)
  2. HTTP: tries https:// and http:// with GET (30s timeout each)
  3. Verdict: truly dead (all fail) or false positive (any succeed)

Use few workers (default 10) to avoid network rate-limiting.

`,
		Example: `  search cc verify
  search cc verify --workers 5 --dns-timeout 15000
  search cc verify --crawl CC-MAIN-2026-08 --limit 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCVerify(cmd.Context(), crawlID, workers, dnsTimeout, httpTimeout, limit)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest cached/latest available)")
	cmd.Flags().IntVar(&workers, "workers", 10, "Verification workers (keep low for accuracy)")
	cmd.Flags().IntVar(&dnsTimeout, "dns-timeout", 10000, "DNS timeout per resolver (ms)")
	cmd.Flags().IntVar(&httpTimeout, "http-timeout", 30000, "HTTP timeout (ms)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max domains to verify (0=all)")

	return cmd
}

func runCCVerify(ctx context.Context, crawlID string, workers, dnsTimeout, httpTimeout, limit int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Domain Verification (Slow, Thorough)"))
	fmt.Println()

	resolvedCrawlID, crawlNote, err := ccResolveCrawlID(ctx, crawlID)
	if err != nil {
		return fmt.Errorf("resolving crawl: %w", err)
	}
	crawlID = resolvedCrawlID
	if crawlNote != "" {
		fmt.Println(labelStyle.Render("Using defaults"))
		ccPrintDefaultCrawlResolution(crawlID, crawlNote)
		fmt.Println()
	}

	ccCfg := cc.DefaultConfig()
	ccCfg.CrawlID = crawlID

	failedPath := ccCfg.FailedDBPath()
	if _, err := os.Stat(failedPath); err != nil {
		return fmt.Errorf("failed DB not found: %s — run 'cc recrawl' first", failedPath)
	}

	// Show failure summary
	summary, total, err := recrawler.FailedDomainSummary(failedPath)
	if err != nil {
		return fmt.Errorf("reading failed DB: %w", err)
	}

	fmt.Printf("  Failed domains: %s\n", ccFmtInt64(int64(total)))
	for reason, count := range summary {
		fmt.Printf("    %-25s %s\n", reason, ccFmtInt64(int64(count)))
	}
	fmt.Println()

	if limit > 0 && limit < total {
		fmt.Println(infoStyle.Render(fmt.Sprintf("  Verifying top %d domains (by URL count)...", limit)))
	}

	outputPath := ccCfg.VerifyDBPath()
	fmt.Printf("  Output: %s\n", outputPath)
	fmt.Printf("  Workers: %d  DNS timeout: %ds  HTTP timeout: %ds\n",
		workers, dnsTimeout/1000, httpTimeout/1000)
	fmt.Println()

	cfg := recrawler.VerifyConfig{
		Workers:     workers,
		DNSTimeout:  time.Duration(dnsTimeout) * time.Millisecond,
		HTTPTimeout: time.Duration(httpTimeout) * time.Millisecond,
	}

	var displayLines int
	err = recrawler.VerifyFailedDomains(ctx, failedPath, outputPath, cfg, limit, func(p recrawler.VerifyProgress) {
		if displayLines > 0 {
			fmt.Printf("\033[%dA\033[J", displayLines)
		}
		pct := float64(0)
		if p.Total > 0 {
			pct = float64(p.Done) / float64(p.Total) * 100
		}
		output := fmt.Sprintf("  Verify %d/%d (%.1f%%)  │  %d alive  │  %d dead  │  %d false+  │  %.1f/s  │  %s\n",
			p.Done, p.Total, pct, p.Alive, p.Dead, p.FalsePos, p.Speed, p.Elapsed.Truncate(time.Second))
		fmt.Print(output)
		displayLines = 1
	})

	if err != nil {
		return fmt.Errorf("verification: %w", err)
	}

	// Print final results
	fmt.Println()
	vTotal, vAlive, vDead, vFP, fpRate, _ := recrawler.VerifySummary(outputPath)
	fmt.Println(successStyle.Render("Verification complete!"))
	fmt.Printf("  Total:           %s domains\n", ccFmtInt64(int64(vTotal)))
	fmt.Printf("  Truly dead:      %s\n", ccFmtInt64(int64(vDead)))
	fmt.Printf("  Actually alive:  %s (false positives)\n", ccFmtInt64(int64(vAlive)))
	fmt.Printf("  False positive rate: %.2f%%\n", fpRate)
	if vFP > 0 {
		fmt.Println()
		fmt.Println(warningStyle.Render(fmt.Sprintf("  %d domains were incorrectly marked dead!", vFP)))
		// Show some examples
		fps, _ := recrawler.VerifyFalsePositives(outputPath, 10)
		for _, fp := range fps {
			fmt.Printf("    %-40s %s  DNS=%s  HTTP=%d  HTTPS=%d\n",
				fp.Domain, fp.OriginalReason,
				fp.DNSSystemIPs, fp.HTTPStatus, fp.HTTPSStatus)
		}
	}
	fmt.Println()
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Results: %s", outputPath)))
	fmt.Println()

	return nil
}

// ── helpers ──────────────────────────────────────────────

func printFilterSummary(filter cc.IndexFilter) {
	var parts []string
	if len(filter.StatusCodes) > 0 {
		parts = append(parts, fmt.Sprintf("status=%v", filter.StatusCodes))
	}
	if len(filter.MimeTypes) > 0 {
		parts = append(parts, fmt.Sprintf("mime=%v", filter.MimeTypes))
	}
	if len(filter.Languages) > 0 {
		parts = append(parts, fmt.Sprintf("lang=%v", filter.Languages))
	}
	if len(filter.Domains) > 0 {
		parts = append(parts, fmt.Sprintf("domain=%v", filter.Domains))
	}
	if len(filter.TLDs) > 0 {
		parts = append(parts, fmt.Sprintf("tld=%v", filter.TLDs))
	}
	if filter.Limit > 0 {
		parts = append(parts, fmt.Sprintf("limit=%d", filter.Limit))
	}
	if len(parts) > 0 {
		fmt.Println(labelStyle.Render(fmt.Sprintf("  Filter: %s", strings.Join(parts, ", "))))
	}
}

func ccFmtInt64(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 1000 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

type ccDNSCacheCoverageSummary struct {
	Total          int
	CachedTotal    int
	CachedResolved int
	CachedDead     int
	CachedTimeout  int
	Pending        int
}

func ccDNSCacheCoverage(dns interface {
	IsResolved(domain string) bool
	IsDead(domain string) bool
	IsTimeout(domain string) bool
}, domains []string) ccDNSCacheCoverageSummary {
	out := ccDNSCacheCoverageSummary{Total: len(domains)}
	for _, domain := range domains {
		switch {
		case dns.IsResolved(domain):
			out.CachedResolved++
		case dns.IsDead(domain):
			out.CachedDead++
		case dns.IsTimeout(domain):
			out.CachedTimeout++
		default:
			out.Pending++
		}
	}
	out.CachedTotal = out.CachedResolved + out.CachedDead + out.CachedTimeout
	return out
}

func ccResolveEffectiveDNSTuning(opts ccRecrawlOpts, pendingDomains int) (workers int, timeout time.Duration, notes []string) {
	// Timeout: preserve explicit values. Auto mode reduces long-tail stalls on large batches.
	// BUG FIX: was `if timeoutMs <= 0` — the default flag value is 2000, not 0, so
	// auto-tune never fired. Now checks !dnsTimeoutExplicit instead.
	timeoutMs := opts.dnsTimeout
	if !opts.dnsTimeoutExplicit {
		timeoutMs = 2000
		if pendingDomains >= 20000 {
			timeoutMs = 1200
		}
		if pendingDomains >= 50000 {
			// Keep 2000ms at large scale: ~5% of valid domains need 1-2s to resolve
			// from Cloudflare/Google (slow NS delegation, DNSSEC validation).
			// The retry that previously recovered these is skipped (see ResolveBatch),
			// so the first-pass timeout must be long enough to catch slow-but-live.
			// 4K workers at 2s = 2,048/s dead throughput → ~40s for 80K hosts
			// (vs 350s at 640 workers with retry, or 60s at 4K workers with retry).
			timeoutMs = 2000
		}
		notes = append(notes, fmt.Sprintf("dns-timeout auto=%dms (use --dns-timeout to override)", timeoutMs))
	}
	timeout = time.Duration(timeoutMs) * time.Millisecond

	if pendingDomains <= 0 {
		return 0, timeout, notes
	}

	// Workers: auto-tune unless explicitly provided. Cap below the fastdns UDP pool capacity
	// to avoid oversubscription and timeout spikes.
	if opts.dnsWorkersExplicit && opts.dnsWorkers > 0 {
		return opts.dnsWorkers, timeout, notes
	}
	if opts.dnsWorkersExplicit && opts.dnsWorkers <= 0 {
		notes = append(notes, "dns-workers auto requested via --dns-workers 0")
	}
	if !opts.dnsWorkersExplicit && opts.dnsWorkers == 2000 {
		notes = append(notes, "dns-workers auto-tuned (default 2000 oversubscribes fastdns UDP pool)")
	}

	cpu := runtime.NumCPU()
	auto := cpu * 64
	if pendingDomains >= 20000 {
		auto = max(auto, 1024)
	}
	if pendingDomains >= 50000 {
		// For CC recrawl batches (80K+ hosts, ~95% dead):
		// need ~4K workers at 1s timeout to finish dead-domain scanning in ~20s.
		// fastDNSConnsPerServer=4096 → pool=8192 → stableCap=6144 allows this.
		auto = max(auto, 4096)
	}
	auto = max(auto, 128)
	if stableCap := recrawler.DNSBatchRecommendedWorkerCap(); stableCap > 0 {
		if auto > stableCap {
			notes = append(notes, fmt.Sprintf("dns-workers capped to %d for fastdns pool stability (pool=%d)",
				stableCap, recrawler.DNSBatchPoolCapacity()))
		}
		auto = min(auto, stableCap)
	}
	auto = min(auto, pendingDomains)
	if auto < 1 {
		auto = 1
	}
	notes = append(notes, fmt.Sprintf("dns-workers auto=%d (cpu=%d, pending=%s)", auto, cpu, ccFmtInt64(int64(pendingDomains))))
	return auto, timeout, notes
}

func ccDNSProgressETA(p recrawler.DNSProgress) string {
	remain := int64(p.Total) - p.Done
	if remain <= 0 {
		return "0s"
	}
	speed := p.Speed
	if p.AvgSpeed > 0 && (speed <= 0 || speed < p.AvgSpeed*0.35) {
		speed = p.AvgSpeed
	}
	if speed <= 0 {
		return "0s"
	}
	eta := time.Duration(float64(time.Second) * float64(remain) / speed)
	if eta < 0 {
		return "0s"
	}
	return eta.Truncate(time.Second).String()
}

func ccWriteDNSTimeoutList(dir string, timeouts map[string]string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating dns timeout list dir: %w", err)
	}
	tsvPath := filepath.Join(dir, "dns_timeout_domains.tsv")
	txtPath := filepath.Join(dir, "dns_timeout_domains.txt")

	domains := make([]string, 0, len(timeouts))
	for domain := range timeouts {
		domains = append(domains, domain)
	}
	sort.Strings(domains)

	tsvFile, err := os.Create(tsvPath)
	if err != nil {
		return "", fmt.Errorf("creating %s: %w", filepath.Base(tsvPath), err)
	}
	defer tsvFile.Close()
	tsvW := bufio.NewWriterSize(tsvFile, 256*1024)
	if _, err := tsvW.WriteString("domain\terror\n"); err != nil {
		return "", err
	}
	for _, domain := range domains {
		errMsg := strings.ReplaceAll(timeouts[domain], "\t", " ")
		errMsg = strings.ReplaceAll(errMsg, "\n", " ")
		errMsg = strings.ReplaceAll(errMsg, "\r", " ")
		if _, err := fmt.Fprintf(tsvW, "%s\t%s\n", domain, errMsg); err != nil {
			return "", err
		}
	}
	if err := tsvW.Flush(); err != nil {
		return "", err
	}

	txtFile, err := os.Create(txtPath)
	if err != nil {
		return "", fmt.Errorf("creating %s: %w", filepath.Base(txtPath), err)
	}
	defer txtFile.Close()
	txtW := bufio.NewWriterSize(txtFile, 256*1024)
	for _, domain := range domains {
		if _, err := txtW.WriteString(domain + "\n"); err != nil {
			return "", err
		}
	}
	if err := txtW.Flush(); err != nil {
		return "", err
	}
	return tsvPath, nil
}

func ccRecrawlParquetRunDir(cfg cc.Config, parquetPath string) string {
	subset := cc.ParquetSubsetFromPath(parquetPath)
	if subset == "" {
		subset = "unknown"
	}
	base := filepath.Base(parquetPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = sanitizePathToken(base)
	return filepath.Join(cfg.RecrawlDir(), "parquet", subset, base)
}

type ccRecrawlParquetResolution struct {
	Path      string
	Detail    string
	Source    string // local, warc-index, manifest-index
	Cached    bool
	SizeBytes int64
}

func ccResolveRecrawlParquetFile(ctx context.Context, cfg cc.Config, selector string, progress cc.ProgressFn) (ccRecrawlParquetResolution, error) {
	// First treat as a local path (supports absolute and relative paths).
	if fi, err := os.Stat(selector); err == nil && fi.Size() > 0 {
		return ccRecrawlParquetResolution{
			Path:      selector,
			Detail:    "local path",
			Source:    "local",
			Cached:    true,
			SizeBytes: fi.Size(),
		}, nil
	}

	kind, idx, err := ccParseParquetSelector(selector)
	if err != nil {
		return ccRecrawlParquetResolution{}, fmt.Errorf("invalid --file %q: %w", selector, err)
	}

	client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)
	switch kind {
	case "warc":
		files, listErr := cc.ListParquetFiles(ctx, client, cfg, cc.ParquetListOptions{Subset: "warc"})
		if listErr != nil {
			return ccRecrawlParquetResolution{}, fmt.Errorf("listing warc parquet files: %w", listErr)
		}
		if len(files) == 0 {
			return ccRecrawlParquetResolution{}, fmt.Errorf("no warc parquet files found in manifest")
		}
		if idx < 0 || idx >= len(files) {
			return ccRecrawlParquetResolution{}, fmt.Errorf("warc parquet index %d out of range (warc subset has %d files)", idx, len(files))
		}
		f := files[idx]
		localPath := cc.LocalParquetPathForRemote(cfg, f.RemotePath)
		if fi, statErr := os.Stat(localPath); statErr == nil && fi.Size() > 0 {
			return ccRecrawlParquetResolution{
				Path:      localPath,
				Detail:    fmt.Sprintf("warc[%d] (manifest=%d, %s)", idx, f.ManifestIndex, f.Filename),
				Source:    "warc-index",
				Cached:    true,
				SizeBytes: fi.Size(),
			}, nil
		}
		if progress != nil {
			progress(cc.DownloadProgress{
				File:       f.Filename,
				RemotePath: f.RemotePath,
				FileIndex:  1,
				TotalFiles: 1,
				Started:    true,
			})
		}
		if err := cc.DownloadParquetFiles(ctx, client, cfg, []cc.ParquetFile{f}, 1, progress); err != nil {
			return ccRecrawlParquetResolution{}, fmt.Errorf("downloading warc parquet index %d: %w", idx, err)
		}
		fi, _ := os.Stat(localPath)
		var size int64
		if fi != nil {
			size = fi.Size()
		}
		return ccRecrawlParquetResolution{
			Path:      localPath,
			Detail:    fmt.Sprintf("warc[%d] (manifest=%d, %s)", idx, f.ManifestIndex, f.Filename),
			Source:    "warc-index",
			SizeBytes: size,
		}, nil

	case "manifest":
		paths, manErr := cc.ParquetManifest(ctx, client, cfg)
		if manErr != nil {
			return ccRecrawlParquetResolution{}, fmt.Errorf("loading parquet manifest: %w", manErr)
		}
		if idx < 0 || idx >= len(paths) {
			return ccRecrawlParquetResolution{}, fmt.Errorf("manifest parquet index %d out of range (manifest has %d files)", idx, len(paths))
		}
		remotePath := paths[idx]
		subset := cc.ParquetSubsetFromPath(remotePath)
		if subset != "warc" {
			return ccRecrawlParquetResolution{}, fmt.Errorf("manifest index %d is subset=%s (recrawl requires subset=warc); use `search cc parquet list --crawl %s --subset warc` then `--file <warc-index>` or choose a warc manifest index", idx, subset, cfg.CrawlID)
		}
		localPath := cc.LocalParquetPathForRemote(cfg, remotePath)
		if fi, statErr := os.Stat(localPath); statErr == nil && fi.Size() > 0 {
			return ccRecrawlParquetResolution{
				Path:      localPath,
				Detail:    fmt.Sprintf("manifest[%d] (%s)", idx, filepath.Base(remotePath)),
				Source:    "manifest-index",
				Cached:    true,
				SizeBytes: fi.Size(),
			}, nil
		}
		gotPath, dlErr := cc.DownloadManifestParquetFile(ctx, client, cfg, idx, progress)
		if dlErr != nil {
			return ccRecrawlParquetResolution{}, fmt.Errorf("downloading manifest parquet index %d: %w", idx, dlErr)
		}
		fi, _ := os.Stat(gotPath)
		var size int64
		if fi != nil {
			size = fi.Size()
		}
		return ccRecrawlParquetResolution{
			Path:      gotPath,
			Detail:    fmt.Sprintf("manifest[%d] (%s)", idx, filepath.Base(remotePath)),
			Source:    "manifest-index",
			SizeBytes: size,
		}, nil
	}

	return ccRecrawlParquetResolution{}, fmt.Errorf("unsupported parquet selector kind: %s", kind)
}

func ccParseParquetSelector(s string) (kind string, idx int, err error) {
	if n, convErr := strconv.Atoi(s); convErr == nil {
		return "warc", n, nil
	}
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("expected local path or selector N / p:N / w:N / m:N")
	}
	n, convErr := strconv.Atoi(parts[1])
	if convErr != nil {
		return "", 0, fmt.Errorf("selector index must be numeric: %q", parts[1])
	}
	switch strings.ToLower(strings.TrimSpace(parts[0])) {
	case "p", "part":
		return "warc", n, nil
	case "w", "warc":
		return "warc", n, nil
	case "m", "manifest":
		return "manifest", n, nil
	default:
		return "", 0, fmt.Errorf("unknown selector prefix %q (use p:, w:, or m:)", parts[0])
	}
}

func sanitizePathToken(s string) string {
	if s == "" {
		return "run"
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

type ccTimeoutReplaySummary struct {
	Candidates             int
	ReplaySeeds            int
	MissingSeeds           int
	Recovered              int
	FatalReclassified      int
	RecheckFailed          int
	RemainingTimeoutKilled int
}

func ccLoadFailedDomainsByReason(dbPath, reason string) ([]recrawler.FailedDomain, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening failed db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT domain, reason, COALESCE(error_msg, ''), COALESCE(ips, ''),
		       COALESCE(url_count, 0), COALESCE(stage, ''), detected_at
		FROM failed_domains
		WHERE reason = ?
		ORDER BY url_count DESC, domain ASC
	`, reason)
	if err != nil {
		return nil, fmt.Errorf("querying failed domains by reason: %w", err)
	}
	defer rows.Close()

	var out []recrawler.FailedDomain
	for rows.Next() {
		var d recrawler.FailedDomain
		if err := rows.Scan(&d.Domain, &d.Reason, &d.Error, &d.IPs, &d.URLCount, &d.Stage, &d.DetectedAt); err != nil {
			return nil, fmt.Errorf("scanning failed domain: %w", err)
		}
		out = append(out, d)
	}
	return out, nil
}

func ccFailedDomainReasonCount(dbPath, reason string) (int, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return 0, fmt.Errorf("opening failed db: %w", err)
	}
	defer db.Close()

	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM failed_domains WHERE reason = ?", reason).Scan(&n); err != nil {
		return 0, fmt.Errorf("counting failed domains by reason: %w", err)
	}
	return n, nil
}

func ccFailedDBSetMeta(dbPath, key, value string) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("opening failed db: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS meta (key VARCHAR PRIMARY KEY, value VARCHAR)`); err != nil {
		return err
	}
	_, err = db.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)`, key, value)
	return err
}

func ccBuildTimeoutReplaySeeds(seeds []recrawler.SeedURL, skip map[string]bool, domains []recrawler.FailedDomain) ([]recrawler.SeedURL, []string) {
	want := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		want[d.Domain] = struct{}{}
	}

	// Candidates may be keyed by registered domain (legacy) or URL host (current).
	// Prefer a URL that was not skipped by --resume, fall back to any URL for the key.
	preferred := make(map[string]recrawler.SeedURL, len(domains))
	fallback := make(map[string]recrawler.SeedURL, len(domains))

	for _, s := range seeds {
		keys := []string{}
		if h := strings.TrimSpace(s.Host); h != "" {
			keys = append(keys, h)
		}
		if d := strings.TrimSpace(s.Domain); d != "" && (len(keys) == 0 || keys[0] != d) {
			keys = append(keys, d)
		}
		if len(keys) == 0 {
			continue
		}
		for _, key := range keys {
			if _, ok := want[key]; !ok {
				continue
			}
			if _, ok := fallback[key]; !ok {
				fallback[key] = s
			}
			if skip != nil && skip[s.URL] {
				continue
			}
			if _, ok := preferred[key]; !ok {
				preferred[key] = s
			}
		}
	}

	replaySeeds := make([]recrawler.SeedURL, 0, len(domains))
	missing := make([]string, 0)
	for _, d := range domains {
		if s, ok := preferred[d.Domain]; ok {
			replaySeeds = append(replaySeeds, s)
			continue
		}
		if s, ok := fallback[d.Domain]; ok {
			replaySeeds = append(replaySeeds, s)
			continue
		}
		missing = append(missing, d.Domain)
	}
	return replaySeeds, missing
}

func ccReclassifyTimeoutKilledDomains(dbPath string, candidates []recrawler.FailedDomain, recovered map[string]bool, replayDead map[string]string) (ccTimeoutReplaySummary, error) {
	summary := ccTimeoutReplaySummary{Candidates: len(candidates)}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return summary, fmt.Errorf("opening failed db for reclassify: %w", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return summary, fmt.Errorf("begin failed db tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE failed_domains SET reason = ?, error_msg = ?, stage = ? WHERE domain = ? AND reason = 'http_timeout_killed'`)
	if err != nil {
		return summary, fmt.Errorf("prepare failed db update: %w", err)
	}
	defer stmt.Close()

	for _, d := range candidates {
		newReason := "http_timeout_recheck_failed"
		errMsg := "still unavailable after timeout replay"
		stage := "http_timeout_replay"

		switch {
		case recovered[d.Domain]:
			newReason = "http_timeout_recovered"
			errMsg = "recovered in timeout replay"
			summary.Recovered++
		default:
			switch replayDead[d.Domain] {
			case "http_refused", "http_dns_error":
				newReason = replayDead[d.Domain]
				errMsg = "reclassified by timeout replay"
				summary.FatalReclassified++
			default:
				summary.RecheckFailed++
			}
		}

		if _, err := stmt.Exec(newReason, errMsg, stage, d.Domain); err != nil {
			return summary, fmt.Errorf("updating failed domain %s: %w", d.Domain, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return summary, fmt.Errorf("commit failed db reclassify: %w", err)
	}

	remaining, err := ccFailedDomainReasonCount(dbPath, "http_timeout_killed")
	if err != nil {
		return summary, err
	}
	summary.RemainingTimeoutKilled = remaining
	return summary, nil
}

func ccRunTimeoutKilledReplay(
	ctx context.Context,
	failedDBPath string,
	rdb *recrawler.ResultDB,
	seeds []recrawler.SeedURL,
	skip map[string]bool,
	dnsResolver *recrawler.DNSResolver,
	baseCfg recrawler.Config,
	label string,
) (ccTimeoutReplaySummary, error) {
	candidates, err := ccLoadFailedDomainsByReason(failedDBPath, "http_timeout_killed")
	if err != nil {
		return ccTimeoutReplaySummary{}, err
	}
	if len(candidates) == 0 {
		return ccTimeoutReplaySummary{}, nil
	}

	replaySeeds, missingSeeds := ccBuildTimeoutReplaySeeds(seeds, skip, candidates)
	summary := ccTimeoutReplaySummary{
		Candidates:   len(candidates),
		ReplaySeeds:  len(replaySeeds),
		MissingSeeds: len(missingSeeds),
	}

	fmt.Println(infoStyle.Render("Timeout replay pass (requeue timeout-killed hosts/domains)..."))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Candidates: %s keys", ccFmtInt64(int64(len(candidates))))))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Replay seeds: %s URLs (1 per key)", ccFmtInt64(int64(len(replaySeeds))))))
	if len(missingSeeds) > 0 {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Missing seeds for %d key(s); will classify as recheck_failed", len(missingSeeds))))
	}

	if len(replaySeeds) == 0 {
		reclass, err := ccReclassifyTimeoutKilledDomains(failedDBPath, candidates, map[string]bool{}, map[string]string{})
		if err != nil {
			return summary, err
		}
		reclass.MissingSeeds = len(missingSeeds)
		return reclass, nil
	}

	replayCfg := baseCfg
	replayCfg.Workers = min(len(replaySeeds), 1000)
	if replayCfg.Workers < 1 {
		replayCfg.Workers = 1
	}
	replayCfg.MaxConnsPerDomain = 1
	replayCfg.DomainFailThreshold = 1000 // disable timeout-based domain kill during replay
	if replayCfg.Timeout < 10*time.Second {
		replayCfg.Timeout = 10 * time.Second
	}

	fmt.Println(labelStyle.Render(fmt.Sprintf("  Replay config: %d workers, %v timeout, max-conns-per-domain=%d, domain-fail-threshold=%d",
		replayCfg.Workers, replayCfg.Timeout, replayCfg.MaxConnsPerDomain, replayCfg.DomainFailThreshold)))
	fmt.Println()

	failedDB, err := recrawler.OpenFailedDB(failedDBPath)
	if err != nil {
		return summary, fmt.Errorf("opening failed db for timeout replay: %w", err)
	}
	defer func() {
		if failedDB != nil {
			failedDB.Close()
		}
	}()
	failedDB.SetMeta("timeout_replay_started_at", time.Now().Format(time.RFC3339))

	replayStats := recrawler.NewStats(len(replaySeeds), len(replaySeeds), label+"-timeout-replay")
	r := recrawler.New(replayCfg, replayStats, rdb)
	r.SetFailedDB(failedDB)
	if dnsResolver != nil {
		r.SetDNSCache(dnsResolver.ResolvedIPs())
		r.SetDeadHosts(dnsResolver.DeadOrTimeoutDomainsWithReasons())
	}

	if err := recrawler.RunWithDisplay(ctx, r, replaySeeds, nil, replayStats); err != nil {
		return summary, fmt.Errorf("timeout replay recrawl: %w", err)
	}
	rdb.Flush(ctx)

	failedDB.SetMeta("timeout_replay_finished_at", time.Now().Format(time.RFC3339))
	if err := failedDB.Close(); err != nil {
		return summary, fmt.Errorf("closing failed db after timeout replay: %w", err)
	}
	failedDB = nil

	recovered := r.SucceededDomains()
	for h := range r.SucceededHosts() {
		recovered[h] = true
	}
	replayDead := r.DeadDomainReasons()
	for h, reason := range r.DeadHostReasons() {
		replayDead[h] = reason
	}
	reclass, err := ccReclassifyTimeoutKilledDomains(failedDBPath, candidates, recovered, replayDead)
	if err != nil {
		return summary, err
	}
	reclass.ReplaySeeds = len(replaySeeds)
	reclass.MissingSeeds = len(missingSeeds)

	fmt.Println(successStyle.Render(fmt.Sprintf("  Timeout replay complete: %s recovered, %s fatal reclassified, %s still failed",
		ccFmtInt64(int64(reclass.Recovered)), ccFmtInt64(int64(reclass.FatalReclassified)), ccFmtInt64(int64(reclass.RecheckFailed)))))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  Remaining http_timeout_killed: %d", reclass.RemainingTimeoutKilled)))
	fmt.Println()

	return reclass, nil
}
