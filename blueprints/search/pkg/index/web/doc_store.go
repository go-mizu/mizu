package web

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"

	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// DocRecord is per-document metadata derived from a .md.warc.gz WARC record header.
type DocRecord struct {
	DocID        string    `json:"doc_id"`
	Shard        string    `json:"shard"`
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	CrawlDate    time.Time `json:"crawl_date,omitempty"`
	SizeBytes    int64     `json:"size_bytes"`
	WordCount    int       `json:"word_count"`
	WARCRecordID string    `json:"warc_record_id,omitempty"`
	RefersTo     string    `json:"refers_to,omitempty"`
}

// DocShardMeta holds per-shard scan statistics.
type DocShardMeta struct {
	Shard          string
	TotalDocs      int64
	TotalSizeBytes int64
	LastDocDate    time.Time
	LastScannedAt  time.Time
}

// shardCache holds all DocRecords for one shard loaded into memory.
type shardCache struct {
	docs     map[string]DocRecord // docID → record
	meta     DocShardMeta
	loadedAt time.Time
}

// DocStore manages per-shard DuckDB metadata files with an in-memory cache.
// Each shard gets its own {shard}.meta.duckdb file inside warcMdBase.
// Records are loaded into memory on first access and cached for fast lookups.
type DocStore struct {
	warcMdBase string
	mu         sync.RWMutex
	cache      map[string]*shardCache // shard → in-memory cache
	scanning   map[string]bool        // shard → scan-in-progress guard
}

// NewDocStore creates a DocStore rooted at warcMdBase.
// DuckDB files are created at {warcMdBase}/{shard}.meta.duckdb as shards are scanned.
func NewDocStore(warcMdBase string) (*DocStore, error) {
	if err := os.MkdirAll(warcMdBase, 0o755); err != nil {
		return nil, fmt.Errorf("doc_store: mkdir: %w", err)
	}
	return &DocStore{
		warcMdBase: warcMdBase,
		cache:      make(map[string]*shardCache),
		scanning:   make(map[string]bool),
	}, nil
}

// Init is a no-op; schema is created per-shard when ScanShard is called.
func (ds *DocStore) Init(_ context.Context) error { return nil }

// Close is a no-op; connections are opened/closed per operation.
func (ds *DocStore) Close() error { return nil }

// ── Internal helpers ──────────────────────────────────────────────────────────

func (ds *DocStore) shardDBPath(shard string) string {
	return filepath.Join(ds.warcMdBase, shard+".meta.duckdb")
}

func openShardDB(path string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

const createDocRecords = `
CREATE TABLE IF NOT EXISTS doc_records (
	doc_id         TEXT PRIMARY KEY,
	url            TEXT NOT NULL DEFAULT '',
	title          TEXT NOT NULL DEFAULT '',
	crawl_date     TEXT NOT NULL DEFAULT '',
	size_bytes     BIGINT DEFAULT 0,
	word_count     INTEGER DEFAULT 0,
	warc_record_id TEXT NOT NULL DEFAULT '',
	refers_to      TEXT NOT NULL DEFAULT '',
	scanned_at     TEXT NOT NULL DEFAULT ''
)`

const createDocScanMeta = `
CREATE TABLE IF NOT EXISTS doc_scan_meta (
	id               INTEGER PRIMARY KEY,
	total_docs       BIGINT DEFAULT 0,
	total_size_bytes BIGINT DEFAULT 0,
	last_doc_date    TEXT NOT NULL DEFAULT '',
	last_scanned_at  TEXT NOT NULL DEFAULT ''
)`

func initShardSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, createDocRecords); err != nil {
		return fmt.Errorf("create doc_records: %w", err)
	}
	if _, err := db.ExecContext(ctx, createDocScanMeta); err != nil {
		return fmt.Errorf("create doc_scan_meta: %w", err)
	}
	return nil
}

// ── Scan guards ───────────────────────────────────────────────────────────────

func (ds *DocStore) acquireScan(shard string) bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.scanning[shard] {
		return false
	}
	ds.scanning[shard] = true
	return true
}

func (ds *DocStore) releaseScan(shard string) {
	ds.mu.Lock()
	delete(ds.scanning, shard)
	ds.mu.Unlock()
}

// IsScanning returns true if a scan is currently running for the given shard.
func (ds *DocStore) IsScanning(_, shard string) bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.scanning[shard]
}

// ── Cache helpers ─────────────────────────────────────────────────────────────

// invalidateCache removes a shard from the in-memory cache so the next access
// re-reads from DuckDB.
func (ds *DocStore) invalidateCache(shard string) {
	ds.mu.Lock()
	delete(ds.cache, shard)
	ds.mu.Unlock()
}

// warmShard loads all records for a shard from its DuckDB file into memory.
// Returns nil if the DuckDB file doesn't exist yet.
func (ds *DocStore) warmShard(ctx context.Context, shard string) (*shardCache, error) {
	dbPath := ds.shardDBPath(shard)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil
	}

	db, err := openShardDB(dbPath + "?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("warm open %s: %w", shard, err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT doc_id, url, title, crawl_date, size_bytes, word_count, warc_record_id, refers_to
		FROM doc_records ORDER BY crawl_date DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("warm query %s: %w", shard, err)
	}
	defer rows.Close()

	sc := &shardCache{
		docs:     make(map[string]DocRecord),
		loadedAt: time.Now(),
	}
	var totalSize int64
	var lastDate time.Time
	for rows.Next() {
		var d DocRecord
		var crawlDate string
		if err := rows.Scan(&d.DocID, &d.URL, &d.Title, &crawlDate,
			&d.SizeBytes, &d.WordCount, &d.WARCRecordID, &d.RefersTo); err != nil {
			return nil, fmt.Errorf("warm scan %s: %w", shard, err)
		}
		d.Shard = shard
		if t, err := time.Parse(time.RFC3339, crawlDate); err == nil {
			d.CrawlDate = t
			if t.After(lastDate) {
				lastDate = t
			}
		}
		totalSize += d.SizeBytes
		sc.docs[d.DocID] = d
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Read scan meta.
	var totalDocs int64
	var lastDocDate, lastScannedAt string
	db.QueryRowContext(ctx, `
		SELECT total_docs, total_size_bytes, last_doc_date, last_scanned_at
		FROM doc_scan_meta WHERE id=1
	`).Scan(&totalDocs, &totalSize, &lastDocDate, &lastScannedAt)

	sc.meta = DocShardMeta{Shard: shard, TotalDocs: int64(len(sc.docs)), TotalSizeBytes: totalSize}
	if t, err := time.Parse(time.RFC3339, lastDocDate); err == nil {
		sc.meta.LastDocDate = t
	}
	if t, err := time.Parse(time.RFC3339Nano, lastScannedAt); err == nil {
		sc.meta.LastScannedAt = t
	}

	return sc, nil
}

// getOrWarm returns the cache for shard, warming from DuckDB if not loaded.
func (ds *DocStore) getOrWarm(ctx context.Context, shard string) (*shardCache, error) {
	ds.mu.RLock()
	sc := ds.cache[shard]
	ds.mu.RUnlock()
	if sc != nil {
		return sc, nil
	}

	sc, err := ds.warmShard(ctx, shard)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, nil // no DB file yet
	}

	ds.mu.Lock()
	ds.cache[shard] = sc
	ds.mu.Unlock()
	return sc, nil
}

// ── ScanShard ─────────────────────────────────────────────────────────────────

// ScanShard scans a single .md.warc.gz file and writes DocRecords to its DuckDB.
// Returns (0, nil) if a scan for this shard is already in progress (idempotent guard).
func (ds *DocStore) ScanShard(ctx context.Context, _, shard, warcMdPath string) (int64, error) {
	if !ds.acquireScan(shard) {
		logInfof("doc_store: scan already in progress shard=%s, skipping", shard)
		return 0, nil
	}
	defer ds.releaseScan(shard)

	f, err := os.Open(warcMdPath)
	if err != nil {
		return 0, fmt.Errorf("doc_store scan open: %w", err)
	}
	defer f.Close()

	// Open (or create) shard DuckDB.
	dbPath := ds.shardDBPath(shard)
	db, err := openShardDB(dbPath)
	if err != nil {
		return 0, fmt.Errorf("doc_store open db %s: %w", shard, err)
	}
	defer db.Close()

	if err := initShardSchema(ctx, db); err != nil {
		return 0, err
	}

	// Full rescan: delete existing records.
	if _, err := db.ExecContext(ctx, `DELETE FROM doc_records`); err != nil {
		return 0, fmt.Errorf("doc_store delete: %w", err)
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)

	var totalDocs int64
	var totalSizeBytes int64
	var lastDocDate time.Time

	// Use DuckDB Appender for bulk insert (avoids SQL parameter binding issues).
	sqlConn, err := db.Conn(ctx)
	if err != nil {
		return 0, fmt.Errorf("doc_store conn: %w", err)
	}

	err = sqlConn.Raw(func(anyConn any) error {
		driverConn, ok := anyConn.(driver.Conn)
		if !ok {
			return fmt.Errorf("doc_store: expected driver.Conn, got %T", anyConn)
		}
		appender, err := duckdb.NewAppenderFromConn(driverConn, "", "doc_records")
		if err != nil {
			return fmt.Errorf("doc_store appender: %w", err)
		}

		r := warcpkg.NewReader(f)
		for r.Next() {
			if ctx.Err() != nil {
				appender.Close()
				return ctx.Err()
			}
			rec := r.Record()
			if rec.Header.Type() != warcpkg.TypeConversion {
				io.Copy(io.Discard, rec.Body)
				continue
			}

			recordID := rec.Header.RecordID()
			docID := warcRecordIDtoDocID(recordID)
			if docID == "" {
				io.Copy(io.Discard, rec.Body)
				continue
			}

			targetURI := rec.Header.TargetURI()
			dateStr := rec.Header.Get("WARC-Date")
			refersTo := rec.Header.RefersTo()
			sizeBytes := rec.Header.ContentLength()

			head := make([]byte, 256)
			n, _ := rec.Body.Read(head)
			io.Copy(io.Discard, rec.Body)
			head = head[:n]

			title := extractDocTitle(head, targetURI)

			if t, err := time.Parse(time.RFC3339, dateStr); err == nil && t.After(lastDocDate) {
				lastDocDate = t
			}
			totalSizeBytes += sizeBytes
			totalDocs++

			if err := appender.AppendRow(
				docID, targetURI, title, dateStr,
				sizeBytes, int32(sizeBytes/5),
				recordID, refersTo, nowStr,
			); err != nil {
				appender.Close()
				return fmt.Errorf("doc_store append: %w", err)
			}
		}
		if err := r.Err(); err != nil {
			appender.Close()
			return fmt.Errorf("doc_store scan: %w", err)
		}
		return appender.Close()
	})
	// Release the sql.Conn back to the pool before using db.ExecContext (MaxOpenConns=1).
	sqlConn.Close()
	if err != nil {
		return 0, err
	}

	lastDocDateStr := ""
	if !lastDocDate.IsZero() {
		lastDocDateStr = lastDocDate.UTC().Format(time.RFC3339)
	}
	db.ExecContext(ctx, `DELETE FROM doc_scan_meta`)
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO doc_scan_meta (id, total_docs, total_size_bytes, last_doc_date, last_scanned_at)
		VALUES (1, %d, %d, '%s', '%s')
	`, totalDocs, totalSizeBytes, lastDocDateStr, nowStr)); err != nil {
		logErrorf("doc_store: update scan_meta shard=%s: %v", shard, err)
	}

	// Invalidate cache so next read re-warms from DuckDB.
	ds.invalidateCache(shard)

	logInfof("doc_store: scanned shard=%s docs=%d size=%d", shard, totalDocs, totalSizeBytes)
	return totalDocs, nil
}

// ScanAll scans all .md.warc.gz files in warcMdBase.
func (ds *DocStore) ScanAll(ctx context.Context, _, warcMdBase string) (int64, error) {
	entries, err := os.ReadDir(warcMdBase)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	var total int64
	for _, e := range entries {
		if ctx.Err() != nil {
			return total, ctx.Err()
		}
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md.warc.gz") {
			continue
		}
		shard := strings.TrimSuffix(e.Name(), ".md.warc.gz")
		n, err := ds.ScanShard(ctx, "", shard, filepath.Join(warcMdBase, e.Name()))
		if err != nil {
			logErrorf("doc_store: scan shard=%s err=%v", shard, err)
			continue
		}
		total += n
	}
	return total, nil
}

// ── Read operations (from cache) ──────────────────────────────────────────────

// ListShardMetas returns scan metadata for all shards that have a .meta.duckdb file.
func (ds *DocStore) ListShardMetas(ctx context.Context, _ string) ([]DocShardMeta, error) {
	entries, err := os.ReadDir(ds.warcMdBase)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []DocShardMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".meta.duckdb") {
			continue
		}
		shard := strings.TrimSuffix(e.Name(), ".meta.duckdb")
		sc, err := ds.getOrWarm(ctx, shard)
		if err != nil || sc == nil {
			continue
		}
		out = append(out, sc.meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Shard < out[j].Shard })
	return out, nil
}

// GetShardMeta returns scan metadata for one shard from the cache.
func (ds *DocStore) GetShardMeta(ctx context.Context, _, shard string) (DocShardMeta, bool, error) {
	sc, err := ds.getOrWarm(ctx, shard)
	if err != nil {
		return DocShardMeta{}, false, err
	}
	if sc == nil {
		return DocShardMeta{}, false, nil
	}
	return sc.meta, true, nil
}

// ListDocs returns paginated DocRecords for a shard, served from the in-memory cache.
func (ds *DocStore) ListDocs(ctx context.Context, _, shard string, page, pageSize int, q, sortBy string) ([]DocRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}

	sc, err := ds.getOrWarm(ctx, shard)
	if err != nil {
		return nil, 0, err
	}
	if sc == nil {
		return nil, 0, nil
	}

	// Collect and filter.
	qLower := strings.ToLower(q)
	docs := make([]DocRecord, 0, len(sc.docs))
	for _, d := range sc.docs {
		if qLower != "" {
			if !strings.Contains(strings.ToLower(d.URL), qLower) &&
				!strings.Contains(strings.ToLower(d.Title), qLower) {
				continue
			}
		}
		docs = append(docs, d)
	}

	// Sort.
	switch sortBy {
	case "size":
		sort.Slice(docs, func(i, j int) bool { return docs[i].SizeBytes > docs[j].SizeBytes })
	case "words":
		sort.Slice(docs, func(i, j int) bool { return docs[i].WordCount > docs[j].WordCount })
	case "title":
		sort.Slice(docs, func(i, j int) bool { return docs[i].Title < docs[j].Title })
	case "url":
		sort.Slice(docs, func(i, j int) bool { return docs[i].URL < docs[j].URL })
	default: // date desc
		sort.Slice(docs, func(i, j int) bool { return docs[i].CrawlDate.After(docs[j].CrawlDate) })
	}

	total := int64(len(docs))
	start := (page - 1) * pageSize
	if start >= len(docs) {
		return nil, total, nil
	}
	end := start + pageSize
	if end > len(docs) {
		end = len(docs)
	}
	return docs[start:end], total, nil
}

// GetDoc returns metadata for a single doc, served from the in-memory cache.
func (ds *DocStore) GetDoc(ctx context.Context, _, shard, docID string) (DocRecord, bool, error) {
	sc, err := ds.getOrWarm(ctx, shard)
	if err != nil {
		return DocRecord{}, false, err
	}
	if sc == nil {
		return DocRecord{}, false, nil
	}
	d, ok := sc.docs[docID]
	return d, ok, nil
}

// ── ShardStats (DuckDB aggregation query, not cached) ────────────────────────

// ShardStatsResponse holds aggregated statistics for a shard.
type ShardStatsResponse struct {
	Shard         string      `json:"shard"`
	TotalDocs     int64       `json:"total_docs"`
	TotalSize     int64       `json:"total_size"`
	AvgSize       int64       `json:"avg_size"`
	MinSize       int64       `json:"min_size"`
	MaxSize       int64       `json:"max_size"`
	DateFrom      string      `json:"date_from"`
	DateTo        string      `json:"date_to"`
	TopDomains    []domainRow `json:"top_domains"`
	SizeBuckets   []sizeRow   `json:"size_buckets"`
	DateHistogram []dateRow   `json:"date_histogram"`
}

type domainRow struct {
	Domain string `json:"domain"`
	Count  int64  `json:"count"`
}

type sizeRow struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type dateRow struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// ShardStats returns aggregated statistics for a shard.
func (ds *DocStore) ShardStats(ctx context.Context, _, shard string) (ShardStatsResponse, error) {
	dbPath := ds.shardDBPath(shard)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return ShardStatsResponse{Shard: shard, TotalDocs: 0}, nil
	}

	db, err := openShardDB(dbPath + "?access_mode=read_only")
	if err != nil {
		return ShardStatsResponse{Shard: shard}, err
	}
	defer db.Close()

	// Top domains by doc count.
	drows, err := db.QueryContext(ctx, `
		SELECT
			regexp_extract(url, 'https?://(?:www\.)?([^/:?#]+)', 1) AS domain,
			COUNT(*) AS cnt
		FROM doc_records
		WHERE url != ''
		GROUP BY 1
		ORDER BY 2 DESC
		LIMIT 20
	`)
	if err != nil {
		return ShardStatsResponse{Shard: shard}, fmt.Errorf("domain stats: %w", err)
	}
	defer drows.Close()
	var domains []domainRow
	for drows.Next() {
		var d domainRow
		drows.Scan(&d.Domain, &d.Count)
		if d.Domain == "" {
			continue
		}
		domains = append(domains, d)
	}

	// Size distribution buckets.
	srows, err := db.QueryContext(ctx, `
		SELECT
			CASE
				WHEN size_bytes < 1024        THEN '<1 KB'
				WHEN size_bytes < 5120        THEN '1–5 KB'
				WHEN size_bytes < 20480       THEN '5–20 KB'
				WHEN size_bytes < 102400      THEN '20–100 KB'
				ELSE '>100 KB'
			END AS bucket,
			COUNT(*) AS cnt
		FROM doc_records
		GROUP BY 1
		ORDER BY MIN(size_bytes)
	`)
	if err != nil {
		return ShardStatsResponse{}, fmt.Errorf("size buckets: %w", err)
	}
	defer srows.Close()
	var sizeBuckets []sizeRow
	for srows.Next() {
		var b sizeRow
		srows.Scan(&b.Label, &b.Count)
		sizeBuckets = append(sizeBuckets, b)
	}

	// Date histogram: docs per day (last 60 days).
	hrows, err := db.QueryContext(ctx, `
		SELECT LEFT(crawl_date, 10) AS day, COUNT(*) AS cnt
		FROM doc_records
		WHERE crawl_date != ''
		GROUP BY 1
		ORDER BY 1
		LIMIT 60
	`)
	if err != nil {
		return ShardStatsResponse{}, fmt.Errorf("date histogram: %w", err)
	}
	defer hrows.Close()
	var histogram []dateRow
	for hrows.Next() {
		var d dateRow
		hrows.Scan(&d.Date, &d.Count)
		histogram = append(histogram, d)
	}

	// Summary totals.
	var totalDocs, totalSize, minSize, maxSize int64
	var minDate, maxDate string
	db.QueryRowContext(ctx, `
		SELECT COUNT(*), SUM(size_bytes), MIN(size_bytes), MAX(size_bytes),
		       MIN(crawl_date), MAX(crawl_date)
		FROM doc_records
	`).Scan(&totalDocs, &totalSize, &minSize, &maxSize, &minDate, &maxDate)

	var avgSize int64
	if totalDocs > 0 {
		avgSize = totalSize / totalDocs
	}

	return ShardStatsResponse{
		Shard:         shard,
		TotalDocs:     totalDocs,
		TotalSize:     totalSize,
		AvgSize:       avgSize,
		MinSize:       minSize,
		MaxSize:       maxSize,
		DateFrom:      minDate,
		DateTo:        maxDate,
		TopDomains:    domains,
		SizeBuckets:   sizeBuckets,
		DateHistogram: histogram,
	}, nil
}

// ── Helper functions ──────────────────────────────────────────────────────────

func warcRecordIDtoDocID(recordID string) string {
	s := strings.TrimPrefix(recordID, "<urn:uuid:")
	s = strings.TrimSuffix(s, ">")
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, ":<>") {
		return ""
	}
	return s
}

func extractDocTitle(head []byte, fallbackURL string) string {
	for _, line := range bytes.Split(head, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if bytes.HasPrefix(line, []byte("# ")) {
			return string(bytes.TrimPrefix(line, []byte("# ")))
		}
		if bytes.HasPrefix(line, []byte("## ")) {
			return string(bytes.TrimPrefix(line, []byte("## ")))
		}
	}
	if fallbackURL != "" {
		if u, err := url.Parse(fallbackURL); err == nil && u.Hostname() != "" {
			return u.Hostname()
		}
	}
	return fallbackURL
}

// readDocFromWARCMd scans warcMdPath for a record matching docID and returns the body.
func readDocFromWARCMd(warcMdPath, docID string) ([]byte, bool, error) {
	f, err := os.Open(warcMdPath)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()

	target := "<urn:uuid:" + docID + ">"
	r := warcpkg.NewReader(f)
	for r.Next() {
		rec := r.Record()
		if rec.Header.RecordID() == target {
			body, err := io.ReadAll(rec.Body)
			if err != nil {
				return nil, false, err
			}
			return body, true, nil
		}
		io.Copy(io.Discard, rec.Body)
	}
	if err := r.Err(); err != nil {
		return nil, false, err
	}
	return nil, false, nil
}

func listWARCMdShards(warcMdBase string) []string {
	entries, err := os.ReadDir(warcMdBase)
	if err != nil {
		return nil
	}
	var shards []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md.warc.gz") {
			shards = append(shards, strings.TrimSuffix(e.Name(), ".md.warc.gz"))
		}
	}
	sort.Strings(shards)
	return shards
}
