package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/cc"
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

Subcommands:
  crawls   List available Common Crawl datasets
  index    Download + import columnar index to DuckDB
  stats    Show index statistics
  query    Query index for matching URLs
  fetch    High-throughput page extraction from WARC files
  warc     Fetch and display a single WARC record
  url      Lookup a URL via CDX API

Examples:
  search cc crawls
  search cc index --crawl CC-MAIN-2026-04
  search cc stats --crawl CC-MAIN-2026-04
  search cc query --crawl CC-MAIN-2026-04 --lang eng --status 200 --limit 100
  search cc fetch --crawl CC-MAIN-2026-04 --lang eng --mime text/html --limit 1000000
  search cc warc --file crawl-data/CC-MAIN-2026-04/... --offset 12345 --length 6789
  search cc url --crawl CC-MAIN-2026-04 --url https://example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCCCrawls())
	cmd.AddCommand(newCCIndex())
	cmd.AddCommand(newCCStats())
	cmd.AddCommand(newCCQuery())
	cmd.AddCommand(newCCFetch())
	cmd.AddCommand(newCCWarc())
	cmd.AddCommand(newCCURL())

	return cmd
}

// ── cc crawls ──────────────────────────────────────────────

func newCCCrawls() *cobra.Command {
	var (
		search string
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "crawls",
		Short: "List available Common Crawl datasets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCCrawls(cmd.Context(), search, limit)
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "Filter crawls by ID")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max crawls to display")

	return cmd
}

func runCCCrawls(ctx context.Context, search string, limit int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl Datasets"))
	fmt.Println()

	client := cc.NewClient("", 4)
	crawls, err := client.ListCrawls(ctx)
	if err != nil {
		return fmt.Errorf("listing crawls: %w", err)
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

	if limit > 0 && len(crawls) > limit {
		crawls = crawls[:limit]
	}

	// Check local data
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "common-crawl")

	fmt.Printf("  %-20s %-30s %-12s %-12s %s\n",
		"ID", "Name", "From", "To", "Local")
	fmt.Println(strings.Repeat("─", 100))

	for _, c := range crawls {
		fromStr := "---"
		toStr := "---"
		if !c.From.IsZero() {
			fromStr = c.From.Format("2006-01-02")
		}
		if !c.To.IsZero() {
			toStr = c.To.Format("2006-01-02")
		}

		localStatus := labelStyle.Render("---")
		crawlDir := filepath.Join(dataDir, c.ID)
		if fi, err := os.Stat(crawlDir); err == nil && fi.IsDir() {
			// Check for index
			if _, err := os.Stat(filepath.Join(crawlDir, "index.duckdb")); err == nil {
				localStatus = successStyle.Render("indexed")
			} else {
				entries, _ := os.ReadDir(filepath.Join(crawlDir, "index"))
				if len(entries) > 0 {
					localStatus = infoStyle.Render(fmt.Sprintf("%d parquet", len(entries)))
				} else {
					localStatus = warningStyle.Render("dir only")
				}
			}
		}

		fmt.Printf("  %-20s %-30s %-12s %-12s %s\n",
			c.ID, c.Name, fromStr, toStr, localStatus)
	}

	fmt.Printf("\n  %s\n", labelStyle.Render(fmt.Sprintf("Showing %d crawls", len(crawls))))
	return nil
}

// ── cc index ──────────────────────────────────────────────

func newCCIndex() *cobra.Command {
	var (
		crawlID   string
		importOnly bool
		workers   int
	)

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Download + import columnar index to DuckDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCIndex(cmd.Context(), crawlID, importOnly, workers)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "CC-MAIN-2026-04", "Crawl ID")
	cmd.Flags().BoolVar(&importOnly, "import-only", false, "Skip download, import existing parquet files")
	cmd.Flags().IntVar(&workers, "workers", 10, "Concurrent download workers")

	return cmd
}

func runCCIndex(ctx context.Context, crawlID string, importOnly bool, workers int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl Index"))
	fmt.Println()

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	cfg.IndexWorkers = workers

	client := cc.NewClient(cfg.BaseURL, cfg.TransportShards)

	if !importOnly {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Downloading columnar index for %s...", crawlID)))
		fmt.Println(labelStyle.Render(fmt.Sprintf("  → %s", cfg.IndexDir())))

		start := time.Now()
		err := cc.DownloadIndex(ctx, client, cfg, func(p cc.DownloadProgress) {
			if p.Error != nil {
				fmt.Println(warningStyle.Render(fmt.Sprintf("  [%d/%d] %s — error: %v",
					p.FileIndex, p.TotalFiles, p.File, p.Error)))
			} else if p.Done {
				fmt.Printf("  [%d/%d] %s\n", p.FileIndex, p.TotalFiles, p.File)
			}
		})
		if err != nil {
			return fmt.Errorf("downloading index: %w", err)
		}
		fmt.Println(successStyle.Render(fmt.Sprintf("  Download complete in %s", time.Since(start).Truncate(time.Second))))
	}

	// Import to DuckDB
	fmt.Println(infoStyle.Render("Importing to DuckDB..."))
	fmt.Println(labelStyle.Render(fmt.Sprintf("  → %s", cfg.IndexDBPath())))

	importStart := time.Now()
	rowCount, err := cc.ImportIndex(ctx, cfg)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCStats(cmd.Context(), crawlID)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "CC-MAIN-2026-04", "Crawl ID")

	return cmd
}

func runCCStats(ctx context.Context, crawlID string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl Index Statistics"))
	fmt.Println()

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

	fmt.Println(infoStyle.Render(fmt.Sprintf("  Crawl: %s", crawlID)))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Index: %s", dbPath)))
	fmt.Println()
	fmt.Println(fmt.Sprintf("  Total records:   %s", ccFmtInt64(summary.TotalRecords)))
	fmt.Println(fmt.Sprintf("  Unique hosts:    %s", ccFmtInt64(summary.UniqueHosts)))
	fmt.Println(fmt.Sprintf("  Unique domains:  %s", ccFmtInt64(summary.UniqueDomains)))

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
	)

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query index for matching URLs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCQuery(cmd.Context(), crawlID, lang, mime, status, domain, tld, limit, count)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "CC-MAIN-2026-04", "Crawl ID")
	cmd.Flags().StringVar(&lang, "lang", "", "Language filter (e.g. eng)")
	cmd.Flags().StringVar(&mime, "mime", "", "MIME type filter (e.g. text/html)")
	cmd.Flags().IntVar(&status, "status", 0, "HTTP status filter (e.g. 200)")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain filter")
	cmd.Flags().StringVar(&tld, "tld", "", "TLD filter (e.g. com)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Max results")
	cmd.Flags().BoolVar(&count, "count", false, "Show count only")

	return cmd
}

func runCCQuery(ctx context.Context, crawlID, lang, mime string, status int, domain, tld string, limit int, countOnly bool) error {
	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	dbPath := cfg.IndexDBPath()

	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index not found — run 'cc index' first")
	}

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

	fmt.Printf("  %-80s %-6s %-20s %s\n", "URL", "Status", "Content-Type", "Language")
	fmt.Println(strings.Repeat("─", 140))

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
	)

	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "High-throughput page extraction from WARC files",
		Long: `Fetch pages from Common Crawl WARC files via byte-range requests.

First queries the columnar index for matching URLs, then fetches WARC records
in parallel from the CDN. Extracted pages are stored in sharded DuckDB files.

Examples:
  search cc fetch --crawl CC-MAIN-2026-04 --lang eng --mime text/html --limit 10000
  search cc fetch --crawl CC-MAIN-2026-04 --status 200 --workers 5000 --limit 1000000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFetch(cmd.Context(), crawlID, lang, mime, status, domain, tld, limit, workers, timeout, resume)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "CC-MAIN-2026-04", "Crawl ID")
	cmd.Flags().StringVar(&lang, "lang", "", "Language filter (e.g. eng)")
	cmd.Flags().StringVar(&mime, "mime", "text/html", "MIME type filter")
	cmd.Flags().IntVar(&status, "status", 200, "HTTP status filter")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain filter")
	cmd.Flags().StringVar(&tld, "tld", "", "TLD filter (e.g. com)")
	cmd.Flags().IntVar(&limit, "limit", 10000, "Max records to fetch")
	cmd.Flags().IntVar(&workers, "workers", 5000, "Concurrent fetch workers")
	cmd.Flags().IntVar(&timeout, "timeout", 30000, "Per-request timeout in ms")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip already-fetched records")

	return cmd
}

func runCCFetch(ctx context.Context, crawlID, lang, mime string, status int, domain, tld string,
	limit, workers, timeout int, resume bool) error {

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Common Crawl WARC Page Extraction"))
	fmt.Println()

	cfg := cc.DefaultConfig()
	cfg.CrawlID = crawlID
	cfg.Workers = workers
	cfg.Timeout = time.Duration(timeout) * time.Millisecond
	cfg.Resume = resume

	dbPath := cfg.IndexDBPath()
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index not found — run 'cc index' first")
	}

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

	// Count matching records
	fmt.Println(infoStyle.Render("Querying index..."))
	totalCount, err := cc.QueryIndexCount(ctx, dbPath, filter)
	if err != nil {
		return fmt.Errorf("counting records: %w", err)
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("  %s matching records (fetching up to %s)",
		ccFmtInt64(totalCount), ccFmtInt64(int64(limit)))))

	// Query WARC pointers
	fmt.Println(infoStyle.Render("Loading WARC pointers..."))
	pointers, err := cc.QueryIndex(ctx, dbPath, filter)
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
	defer rdb.Close()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Results → %s/ (8 shards)", cfg.ResultDir())))

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

// ── cc warc ──────────────────────────────────────────────

func newCCWarc() *cobra.Command {
	var (
		file   string
		offset int64
		length int64
	)

	cmd := &cobra.Command{
		Use:   "warc",
		Short: "Fetch and display a single WARC record",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCWarc(cmd.Context(), file, offset, length)
		},
	}

	cmd.Flags().StringVar(&file, "file", "", "WARC file path (relative to CC base URL)")
	cmd.Flags().Int64Var(&offset, "offset", 0, "Byte offset")
	cmd.Flags().Int64Var(&length, "length", 0, "Byte length")
	cmd.MarkFlagRequired("file")
	cmd.MarkFlagRequired("offset")
	cmd.MarkFlagRequired("length")

	return cmd
}

func runCCWarc(ctx context.Context, file string, offset, length int64) error {
	client := cc.NewClient("", 4)

	ptr := cc.WARCPointer{
		WARCFilename: file,
		RecordOffset: offset,
		RecordLength: length,
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("Fetching WARC record from %s [%d-%d]...", file, offset, offset+length-1)))

	data, err := client.FetchWARCRecord(ctx, 0, ptr)
	if err != nil {
		return fmt.Errorf("fetching record: %w", err)
	}

	resp, err := cc.ParseWARCRecord(data)
	if err != nil {
		return fmt.Errorf("parsing record: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("WARC Record:"))
	fmt.Printf("  Type:       %s\n", resp.WARCType)
	fmt.Printf("  Target URI: %s\n", resp.TargetURI)
	fmt.Printf("  Date:       %s\n", resp.Date.Format(time.RFC3339))
	fmt.Printf("  Record ID:  %s\n", resp.RecordID)
	fmt.Printf("  HTTP Status: %d\n", resp.HTTPStatus)

	fmt.Println()
	fmt.Println(infoStyle.Render("HTTP Headers:"))
	for k, v := range resp.HTTPHeaders {
		fmt.Printf("  %s: %s\n", k, v)
	}

	fmt.Println()
	fmt.Println(infoStyle.Render(fmt.Sprintf("Body (%d bytes):", len(resp.Body))))
	body := string(resp.Body)
	if len(body) > 2000 {
		body = body[:2000] + "\n... (truncated)"
	}
	fmt.Println(body)

	return nil
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
		Short: "Lookup a URL via CDX API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCURL(cmd.Context(), crawlID, targetURL, domain, limit)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "CC-MAIN-2026-04", "Crawl ID")
	cmd.Flags().StringVar(&targetURL, "url", "", "URL to lookup")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain to lookup (all URLs)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results for domain lookup")

	return cmd
}

func runCCURL(ctx context.Context, crawlID, targetURL, domain string, limit int) error {
	if targetURL == "" && domain == "" {
		return fmt.Errorf("--url or --domain is required")
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
	fmt.Println(strings.Repeat("─", 130))

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

// ── helpers ──────────────────────────────────────────────

func ccFmtInt64(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 1000 {
		return s
	}

	// Add comma separators
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
