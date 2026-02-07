package cc

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	_ "github.com/duckdb/duckdb-go/v2"
)

// IndexManifest returns the list of parquet file paths for a crawl's columnar index.
// Uses cache to avoid re-fetching the manifest.
func IndexManifest(ctx context.Context, client *Client, cfg Config) ([]string, error) {
	// Check cache first
	cache := NewCache(cfg.DataDir)
	cd := cache.Load()
	if cd != nil {
		if paths := cache.GetManifest(cd, cfg.CrawlID, "cc-index-table.paths.gz"); paths != nil {
			return paths, nil
		}
	}

	paths, err := client.DownloadManifest(ctx, cfg.CrawlID, "cc-index-table.paths.gz")
	if err != nil {
		return nil, fmt.Errorf("downloading index manifest: %w", err)
	}

	// Filter to subset=warc only
	var warcPaths []string
	for _, p := range paths {
		if strings.Contains(p, "subset=warc") {
			warcPaths = append(warcPaths, p)
		}
	}

	// Cache manifest
	if cd == nil {
		cd = &CacheData{}
	}
	cache.SetManifest(cd, cfg.CrawlID, "cc-index-table.paths.gz", warcPaths)
	cache.Save(cd)

	return warcPaths, nil
}

// DownloadIndex downloads columnar index parquet files for a crawl.
// Uses the cc-index-table.paths.gz manifest to discover files.
// If sampleSize > 0, only downloads that many files (evenly spaced for representative sample).
// This is the key disk/network optimization: 1 file ≈ 220MB → ~2.5M records, enough for most queries.
func DownloadIndex(ctx context.Context, client *Client, cfg Config, sampleSize int, progress ProgressFn) error {
	warcPaths, err := IndexManifest(ctx, client, cfg)
	if err != nil {
		return err
	}
	if len(warcPaths) == 0 {
		return fmt.Errorf("no warc subset parquet files found in manifest")
	}

	// Sample mode: pick evenly spaced files for representative coverage
	if sampleSize > 0 && sampleSize < len(warcPaths) {
		sampled := make([]string, 0, sampleSize)
		step := float64(len(warcPaths)) / float64(sampleSize)
		for i := 0; i < sampleSize; i++ {
			idx := int(float64(i) * step)
			if idx >= len(warcPaths) {
				idx = len(warcPaths) - 1
			}
			sampled = append(sampled, warcPaths[idx])
		}
		warcPaths = sampled
	}

	indexDir := cfg.IndexDir()
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return fmt.Errorf("creating index dir: %w", err)
	}

	workers := cfg.IndexWorkers
	if workers <= 0 {
		workers = 10
	}

	var completed atomic.Int64
	total := len(warcPaths)
	sem := make(chan struct{}, workers)
	var mu sync.Mutex
	var firstErr error

	for _, remotePath := range warcPaths {
		if ctx.Err() != nil {
			break
		}

		localName := filepath.Base(remotePath)
		localPath := filepath.Join(indexDir, localName)

		// Skip if already downloaded
		if fi, err := os.Stat(localPath); err == nil && fi.Size() > 0 {
			done := completed.Add(1)
			if progress != nil {
				progress(DownloadProgress{
					File:       localName,
					FileIndex:  int(done),
					TotalFiles: total,
					Done:       true,
				})
			}
			continue
		}

		path := remotePath
		lPath := localPath
		lName := localName

		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()

			err := client.DownloadFile(ctx, path, lPath, nil)
			done := completed.Add(1)

			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("downloading %s: %w", path, err)
				}
				mu.Unlock()
			}

			if progress != nil {
				progress(DownloadProgress{
					File:       lName,
					FileIndex:  int(done),
					TotalFiles: total,
					Done:       err == nil,
					Error:      err,
				})
			}
		}()
	}

	// Wait for all workers
	for range workers {
		sem <- struct{}{}
	}

	if firstErr != nil {
		return firstErr
	}
	return nil
}

// DownloadOneIndexFile downloads a single parquet file from the CC index manifest.
// fileIndex == -1 downloads the last (latest) file; fileIndex >= 0 downloads that specific file.
// Returns the local path to the downloaded parquet file.
func DownloadOneIndexFile(ctx context.Context, client *Client, cfg Config, fileIndex int, progress ProgressFn) (string, error) {
	warcPaths, err := IndexManifest(ctx, client, cfg)
	if err != nil {
		return "", err
	}
	if len(warcPaths) == 0 {
		return "", fmt.Errorf("no warc subset parquet files found in manifest")
	}

	// Resolve file index
	idx := fileIndex
	if idx < 0 {
		idx = len(warcPaths) - 1
	}
	if idx >= len(warcPaths) {
		return "", fmt.Errorf("file index %d out of range (manifest has %d files)", idx, len(warcPaths))
	}

	remotePath := warcPaths[idx]
	localName := filepath.Base(remotePath)

	indexDir := cfg.IndexDir()
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return "", fmt.Errorf("creating index dir: %w", err)
	}
	localPath := filepath.Join(indexDir, localName)

	// Skip if already downloaded
	if fi, err := os.Stat(localPath); err == nil && fi.Size() > 0 {
		if progress != nil {
			progress(DownloadProgress{
				File:       localName,
				FileIndex:  1,
				TotalFiles: 1,
				Done:       true,
			})
		}
		return localPath, nil
	}

	if progress != nil {
		progress(DownloadProgress{
			File:       localName,
			FileIndex:  0,
			TotalFiles: 1,
		})
	}

	if err := client.DownloadFile(ctx, remotePath, localPath, nil); err != nil {
		return "", fmt.Errorf("downloading %s: %w", remotePath, err)
	}

	if progress != nil {
		progress(DownloadProgress{
			File:       localName,
			FileIndex:  1,
			TotalFiles: 1,
			Done:       true,
		})
	}

	return localPath, nil
}

// QueryRemoteParquet queries parquet files directly from the CC S3 bucket via DuckDB's httpfs.
// This avoids downloading any parquet files locally — ideal for quick lookups.
// Note: slower than local queries but uses zero disk space.
func QueryRemoteParquet(ctx context.Context, cfg Config, filter IndexFilter) ([]WARCPointer, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("opening duckdb: %w", err)
	}
	defer db.Close()

	// Install and load httpfs for remote parquet access
	db.ExecContext(ctx, "INSTALL httpfs")
	db.ExecContext(ctx, "LOAD httpfs")
	db.ExecContext(ctx, "SET s3_region='us-east-1'")

	// Build remote glob URL
	remoteGlob := fmt.Sprintf("s3://commoncrawl/cc-index/table/cc-main/warc/crawl=%s/subset=warc/*.parquet", cfg.CrawlID)

	// Build query with filter
	query, args := buildRemoteQuery(remoteGlob, filter)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying remote parquet: %w", err)
	}
	defer rows.Close()

	var pointers []WARCPointer
	for rows.Next() {
		var p WARCPointer
		var offset, length int64
		if err := rows.Scan(
			&p.URL, &p.WARCFilename, &offset, &length,
			&p.ContentType, &p.Language, &p.FetchStatus, &p.Domain,
		); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		p.RecordOffset = offset
		p.RecordLength = length
		pointers = append(pointers, p)
	}
	return pointers, rows.Err()
}

func buildRemoteQuery(parquetGlob string, f IndexFilter) (string, []any) {
	var b strings.Builder
	var args []any
	var conditions []string

	b.WriteString(fmt.Sprintf(`SELECT url, warc_filename, warc_record_offset, warc_record_length,
		COALESCE(content_mime_detected, ''), COALESCE(content_languages, ''),
		fetch_status, COALESCE(url_host_registered_domain, '')
		FROM read_parquet('%s', hive_partitioning=true)`, parquetGlob))

	if len(f.StatusCodes) > 0 {
		placeholders := makeIntPlaceholders(f.StatusCodes)
		conditions = append(conditions, fmt.Sprintf("fetch_status IN (%s)", placeholders))
		for _, s := range f.StatusCodes {
			args = append(args, s)
		}
	}

	if len(f.MimeTypes) > 0 {
		placeholders := makeStringPlaceholders(len(f.MimeTypes))
		conditions = append(conditions, fmt.Sprintf("content_mime_detected IN (%s)", placeholders))
		for _, m := range f.MimeTypes {
			args = append(args, m)
		}
	}

	if len(f.TLDs) > 0 {
		placeholders := makeStringPlaceholders(len(f.TLDs))
		conditions = append(conditions, fmt.Sprintf("url_host_tld IN (%s)", placeholders))
		for _, t := range f.TLDs {
			args = append(args, t)
		}
	}

	if len(f.Domains) > 0 {
		placeholders := makeStringPlaceholders(len(f.Domains))
		conditions = append(conditions, fmt.Sprintf("url_host_registered_domain IN (%s)", placeholders))
		for _, d := range f.Domains {
			args = append(args, d)
		}
	}

	if len(f.ExcludeDomains) > 0 {
		placeholders := makeStringPlaceholders(len(f.ExcludeDomains))
		conditions = append(conditions, fmt.Sprintf("url_host_registered_domain NOT IN (%s)", placeholders))
		for _, d := range f.ExcludeDomains {
			args = append(args, d)
		}
	}

	for _, lang := range f.Languages {
		conditions = append(conditions, "content_languages LIKE ?")
		args = append(args, "%"+lang+"%")
	}

	conditions = append(conditions, "warc_filename IS NOT NULL")

	if len(conditions) > 0 {
		b.WriteString(" WHERE ")
		b.WriteString(strings.Join(conditions, " AND "))
	}

	if f.Limit > 0 {
		b.WriteString(fmt.Sprintf(" LIMIT %d", f.Limit))
	}
	if f.Offset > 0 {
		b.WriteString(fmt.Sprintf(" OFFSET %d", f.Offset))
	}

	return b.String(), args
}

// ImportIndex imports downloaded parquet files into a DuckDB database.
// Creates the ccindex table with all columns and useful indexes.
func ImportIndex(ctx context.Context, cfg Config) (int64, error) {
	indexDir := cfg.IndexDir()
	dbPath := cfg.IndexDBPath()

	// Check that parquet files exist
	entries, err := os.ReadDir(indexDir)
	if err != nil {
		return 0, fmt.Errorf("reading index dir: %w", err)
	}
	var parquetCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".parquet") {
			parquetCount++
		}
	}
	if parquetCount == 0 {
		return 0, fmt.Errorf("no parquet files found in %s", indexDir)
	}

	// Remove existing DB for clean import
	os.Remove(dbPath)
	os.Remove(dbPath + ".wal")

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return 0, fmt.Errorf("opening duckdb: %w", err)
	}
	defer db.Close()

	// Import all parquet files using DuckDB's glob
	glob := filepath.Join(indexDir, "*.parquet")
	query := fmt.Sprintf(`CREATE TABLE ccindex AS SELECT * FROM read_parquet('%s')`, glob)
	if _, err := db.ExecContext(ctx, query); err != nil {
		return 0, fmt.Errorf("importing parquet: %w", err)
	}

	// Create indexes for common query patterns
	indexes := []string{
		"CREATE INDEX idx_domain ON ccindex(url_host_registered_domain)",
		"CREATE INDEX idx_tld ON ccindex(url_host_tld)",
		"CREATE INDEX idx_status ON ccindex(fetch_status)",
		"CREATE INDEX idx_mime ON ccindex(content_mime_detected)",
	}
	for _, idx := range indexes {
		if _, err := db.ExecContext(ctx, idx); err != nil {
			// Non-fatal: index creation may fail on large datasets
			continue
		}
	}

	// Count rows
	var rowCount int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ccindex").Scan(&rowCount); err != nil {
		return 0, fmt.Errorf("counting rows: %w", err)
	}

	return rowCount, nil
}

// QueryIndex queries the columnar index with the given filter and returns WARC pointers.
func QueryIndex(ctx context.Context, dbPath string, filter IndexFilter) ([]WARCPointer, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("opening index db: %w", err)
	}
	defer db.Close()

	query, args := buildIndexQuery(filter)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying index: %w", err)
	}
	defer rows.Close()

	var pointers []WARCPointer
	for rows.Next() {
		var p WARCPointer
		var offset, length int64
		if err := rows.Scan(
			&p.URL, &p.WARCFilename, &offset, &length,
			&p.ContentType, &p.Language, &p.FetchStatus, &p.Domain,
		); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		p.RecordOffset = offset
		p.RecordLength = length
		pointers = append(pointers, p)
	}
	return pointers, rows.Err()
}

// QueryIndexCount returns the count of matching records.
func QueryIndexCount(ctx context.Context, dbPath string, filter IndexFilter) (int64, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return 0, fmt.Errorf("opening index db: %w", err)
	}
	defer db.Close()

	query, args := buildCountQuery(filter)
	var count int64
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting: %w", err)
	}
	return count, nil
}

// IndexStats returns summary statistics about the imported index.
func IndexStats(ctx context.Context, dbPath string) (*IndexSummary, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("opening index db: %w", err)
	}
	defer db.Close()

	summary := &IndexSummary{
		StatusDist: make(map[int]int64),
		MimeDist:   make(map[string]int64),
		TLDDist:    make(map[string]int64),
		LangDist:   make(map[string]int64),
	}

	// Total records
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ccindex").Scan(&summary.TotalRecords)

	// Unique hosts and domains
	db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT url_host_name) FROM ccindex").Scan(&summary.UniqueHosts)
	db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT url_host_registered_domain) FROM ccindex").Scan(&summary.UniqueDomains)

	// Status distribution (top 10)
	rows, err := db.QueryContext(ctx, "SELECT fetch_status, COUNT(*) as cnt FROM ccindex GROUP BY fetch_status ORDER BY cnt DESC LIMIT 10")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var status int
			var count int64
			rows.Scan(&status, &count)
			summary.StatusDist[status] = count
		}
	}

	// MIME distribution (top 10)
	rows2, err := db.QueryContext(ctx, "SELECT content_mime_detected, COUNT(*) as cnt FROM ccindex WHERE content_mime_detected IS NOT NULL GROUP BY content_mime_detected ORDER BY cnt DESC LIMIT 10")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var mime string
			var count int64
			rows2.Scan(&mime, &count)
			summary.MimeDist[mime] = count
		}
	}

	// TLD distribution (top 20)
	rows3, err := db.QueryContext(ctx, "SELECT url_host_tld, COUNT(*) as cnt FROM ccindex WHERE url_host_tld IS NOT NULL GROUP BY url_host_tld ORDER BY cnt DESC LIMIT 20")
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var tld string
			var count int64
			rows3.Scan(&tld, &count)
			summary.TLDDist[tld] = count
		}
	}

	return summary, nil
}

// buildIndexQuery constructs a SELECT query from an IndexFilter.
func buildIndexQuery(f IndexFilter) (string, []any) {
	var b strings.Builder
	var args []any
	var conditions []string

	b.WriteString(`SELECT url, warc_filename, warc_record_offset, warc_record_length,
		COALESCE(content_mime_detected, ''), COALESCE(content_languages, ''),
		fetch_status, COALESCE(url_host_registered_domain, '')
		FROM ccindex`)

	if len(f.StatusCodes) > 0 {
		placeholders := makeIntPlaceholders(f.StatusCodes)
		conditions = append(conditions, fmt.Sprintf("fetch_status IN (%s)", placeholders))
		for _, s := range f.StatusCodes {
			args = append(args, s)
		}
	}

	if len(f.MimeTypes) > 0 {
		placeholders := makeStringPlaceholders(len(f.MimeTypes))
		conditions = append(conditions, fmt.Sprintf("content_mime_detected IN (%s)", placeholders))
		for _, m := range f.MimeTypes {
			args = append(args, m)
		}
	}

	if len(f.TLDs) > 0 {
		placeholders := makeStringPlaceholders(len(f.TLDs))
		conditions = append(conditions, fmt.Sprintf("url_host_tld IN (%s)", placeholders))
		for _, t := range f.TLDs {
			args = append(args, t)
		}
	}

	if len(f.Domains) > 0 {
		placeholders := makeStringPlaceholders(len(f.Domains))
		conditions = append(conditions, fmt.Sprintf("url_host_registered_domain IN (%s)", placeholders))
		for _, d := range f.Domains {
			args = append(args, d)
		}
	}

	if len(f.ExcludeDomains) > 0 {
		placeholders := makeStringPlaceholders(len(f.ExcludeDomains))
		conditions = append(conditions, fmt.Sprintf("url_host_registered_domain NOT IN (%s)", placeholders))
		for _, d := range f.ExcludeDomains {
			args = append(args, d)
		}
	}

	for _, lang := range f.Languages {
		conditions = append(conditions, "content_languages LIKE ?")
		args = append(args, "%"+lang+"%")
	}

	// Filter out NULL warc_filename (needed for WARC fetching)
	conditions = append(conditions, "warc_filename IS NOT NULL")

	if len(conditions) > 0 {
		b.WriteString(" WHERE ")
		b.WriteString(strings.Join(conditions, " AND "))
	}

	if f.Limit > 0 {
		b.WriteString(fmt.Sprintf(" LIMIT %d", f.Limit))
	}
	if f.Offset > 0 {
		b.WriteString(fmt.Sprintf(" OFFSET %d", f.Offset))
	}

	return b.String(), args
}

// buildCountQuery constructs a COUNT query from an IndexFilter.
func buildCountQuery(f IndexFilter) (string, []any) {
	selectQuery, args := buildIndexQuery(f)
	// Replace SELECT ... FROM with SELECT COUNT(*) FROM
	fromIdx := strings.Index(selectQuery, "FROM ccindex")
	if fromIdx < 0 {
		return "SELECT 0", nil
	}
	return "SELECT COUNT(*) " + selectQuery[fromIdx:], args
}

func makeStringPlaceholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func makeIntPlaceholders(vals []int) string {
	parts := make([]string, len(vals))
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}
