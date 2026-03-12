package web

import (
	"bufio"
	"bytes"
	"compress/gzip"
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

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/api"
	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// DocRecord is per-document metadata derived from a .md.warc.gz WARC record header.
type DocRecord = api.DocRecord

// DocShardMeta holds per-shard scan statistics.
type DocShardMeta = api.DocShardMeta

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
	host           TEXT NOT NULL DEFAULT '',
	title          TEXT NOT NULL DEFAULT '',
	crawl_date     TEXT NOT NULL DEFAULT '',
	size_bytes     BIGINT DEFAULT 0,
	word_count     INTEGER DEFAULT 0,
	warc_record_id TEXT NOT NULL DEFAULT '',
	refers_to      TEXT NOT NULL DEFAULT '',
	gzip_offset    BIGINT DEFAULT 0,
	gzip_size      BIGINT DEFAULT 0,
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
	// Migrate: if the table exists but lacks new columns (host, gzip_offset),
	// drop and recreate. Full rescan will repopulate.
	var hasHost int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name='doc_records' AND column_name='host'
	`).Scan(&hasHost)
	if err == nil && hasHost == 0 {
		// Old schema — check if table exists at all.
		var tableExists int
		db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM information_schema.tables
			WHERE table_name='doc_records'
		`).Scan(&tableExists)
		if tableExists > 0 {
			db.ExecContext(ctx, `DROP TABLE doc_records`)
		}
	}

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
		SELECT doc_id, url, host, title, crawl_date, size_bytes, word_count,
		       warc_record_id, refers_to, gzip_offset, gzip_size
		FROM doc_records ORDER BY crawl_date DESC
	`)
	if err != nil {
		// Old schema without new columns — return empty cache so rescan is triggered.
		return nil, nil
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
		if err := rows.Scan(&d.DocID, &d.URL, &d.Host, &d.Title, &crawlDate,
			&d.SizeBytes, &d.WordCount, &d.WARCRecordID, &d.RefersTo,
			&d.GzipOffset, &d.GzipSize); err != nil {
			return nil, fmt.Errorf("warm scan %s: %w", shard, err)
		}
		d.Shard = shard
		if t, ok := parseDocTime(crawlDate); ok {
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

// ScanShard scans a .md.warc.gz file and writes DocRecords (with gzip offsets)
// to the shard's DuckDB. Returns (0, nil) if a scan is already in progress.
func (ds *DocStore) ScanShard(ctx context.Context, _, shard, warcMdPath string) (int64, error) {
	if !ds.acquireScan(shard) {
		logInfof("doc_store: scan already in progress shard=%s, skipping", shard)
		return 0, nil
	}
	defer ds.releaseScan(shard)
	return ds.scanWARCMd(ctx, shard, warcMdPath)
}

// countingReader wraps an io.Reader and tracks the total bytes read.
type countingReader struct {
	r io.Reader
	n int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.n += int64(n)
	return n, err
}

// scanWARCMd scans a .md.warc.gz file. For each conversion record it extracts
// URL, host, date, title, and records the gzip member byte offset/size for
// fast random-access retrieval later.
//
// Gzip member tracking: we wrap the file in a countingReader, then a bufio.Reader.
// The gzip reader reads from the bufio. After each member is fully consumed,
// (cr.n - br.Buffered()) gives the exact byte offset in the file.
func (ds *DocStore) scanWARCMd(ctx context.Context, shard, warcMdPath string) (int64, error) {
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

	// Use DuckDB Appender for bulk insert.
	sqlConn, err := db.Conn(ctx)
	if err != nil {
		return 0, fmt.Errorf("doc_store conn: %w", err)
	}

	// Wrap the file in countingReader → bufio.Reader.
	// gzip.NewReader(br) won't re-wrap because bufio implements io.ByteReader.
	// After each member: filePos = cr.n - int64(br.Buffered()).
	cr := &countingReader{r: f}
	br := bufio.NewReaderSize(cr, 64*1024)

	err = sqlConn.Raw(func(anyConn any) error {
		driverConn, ok := anyConn.(driver.Conn)
		if !ok {
			return fmt.Errorf("doc_store: expected driver.Conn, got %T", anyConn)
		}
		appender, err := duckdb.NewAppenderFromConn(driverConn, "", "doc_records")
		if err != nil {
			return fmt.Errorf("doc_store appender: %w", err)
		}

		first := true
		var gz *gzip.Reader

		for {
			if ctx.Err() != nil {
				appender.Close()
				return ctx.Err()
			}

			// Record file offset of this gzip member's start.
			memberStart := cr.n - int64(br.Buffered())

			// Check for gzip magic bytes (EOF detection).
			peek, peekErr := br.Peek(2)
			if peekErr != nil {
				break // EOF or error
			}
			if peek[0] != 0x1f || peek[1] != 0x8b {
				break // not a gzip member
			}

			// Open gzip member.
			if first {
				gz, err = gzip.NewReader(br)
				if err != nil {
					break
				}
				first = false
			} else {
				if err := gz.Reset(br); err != nil {
					break
				}
			}
			gz.Multistream(false)

			// Parse the WARC record from the decompressed gzip member.
			// NewReader on a non-gzip stream parses it as plain WARC text.
			wr := warcpkg.NewReader(gz)
			if !wr.Next() {
				// Drain and continue to next member.
				io.Copy(io.Discard, gz)
				continue
			}
			rec := wr.Record()
			if rec.Header.Type() != warcpkg.TypeConversion {
				io.Copy(io.Discard, rec.Body)
				io.Copy(io.Discard, gz)
				continue
			}

			recordID := rec.Header.RecordID()
			docID := warcRecordIDtoDocID(recordID)
			if docID == "" {
				io.Copy(io.Discard, rec.Body)
				io.Copy(io.Discard, gz)
				continue
			}

			targetURI := rec.Header.TargetURI()
			dateStr := rec.Header.Get("WARC-Date")
			refersTo := rec.Header.RefersTo()
			sizeBytes := rec.Header.ContentLength()

			// Read first chunk for metadata extraction from markdown content.
			head := make([]byte, 8192)
			n, _ := rec.Body.Read(head)
			io.Copy(io.Discard, rec.Body)
			io.Copy(io.Discard, gz) // drain any trailing bytes in the member
			head = head[:n]

			meta := extractDocMetadata(head)
			if strings.TrimSpace(targetURI) == "" {
				targetURI = meta.URL
			}
			host := extractHost(targetURI)
			if host == "" {
				host = meta.Host
			}
			title := extractDocTitle(head, targetURI)
			if title == "" {
				title = meta.Title
			}
			if strings.TrimSpace(dateStr) == "" {
				dateStr = meta.Date
			}
			if t, ok := parseDocTime(dateStr); ok {
				dateStr = t.UTC().Format(time.RFC3339)
				if t.After(lastDocDate) {
					lastDocDate = t
				}
			} else if strings.TrimSpace(meta.Date) != "" && meta.Date != dateStr {
				if t, ok := parseDocTime(meta.Date); ok {
					dateStr = t.UTC().Format(time.RFC3339)
					if t.After(lastDocDate) {
						lastDocDate = t
					}
				}
			}

			// Compute gzip member size after fully draining the member.
			memberEnd := cr.n - int64(br.Buffered())
			gzipSize := memberEnd - memberStart

			totalSizeBytes += sizeBytes
			totalDocs++

			if err := appender.AppendRow(
				docID, targetURI, host, title, dateStr,
				sizeBytes, int32(sizeBytes/5),
				recordID, refersTo,
				memberStart, gzipSize,
				nowStr,
			); err != nil {
				appender.Close()
				return fmt.Errorf("doc_store append: %w", err)
			}
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

	logInfof("doc_store: scanned shard=%s docs=%d size=%d (warc_md)", shard, totalDocs, totalSizeBytes)
	return totalDocs, nil
}

// ScanAll scans all .md.warc.gz files in the warc_md/ directory.
// crawlBase should be the crawl directory (e.g. ~/data/common-crawl/{crawlID}).
func (ds *DocStore) ScanAll(ctx context.Context, _, crawlBase string) (int64, error) {
	var total int64

	warcMdDir := crawlBase
	if !strings.HasSuffix(filepath.Base(crawlBase), "warc_md") {
		warcMdDir = filepath.Join(crawlBase, "warc_md")
	}
	entries, err := os.ReadDir(warcMdDir)
	if err != nil {
		return 0, nil // no warc_md dir yet
	}
	for _, e := range entries {
		if ctx.Err() != nil {
			return total, ctx.Err()
		}
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md.warc.gz") {
			continue
		}
		shard := strings.TrimSuffix(e.Name(), ".md.warc.gz")
		n, err := ds.ScanShard(ctx, "", shard, filepath.Join(warcMdDir, e.Name()))
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
				!strings.Contains(strings.ToLower(d.Title), qLower) &&
				!strings.Contains(strings.ToLower(d.Host), qLower) {
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
type ShardStatsResponse = api.ShardStatsResponse

type domainRow = api.DomainRow

type sizeRow = api.SizeBucket

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

	// Top domains by doc count (uses pre-extracted host column).
	drows, err := db.QueryContext(ctx, `
		SELECT host, COUNT(*) AS cnt
		FROM doc_records
		WHERE host != ''
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

	// Domain count.
	var totalDomains int64
	db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT host) FROM doc_records WHERE host != ''`).Scan(&totalDomains)

	// Pages-per-domain distribution (how many domains have 1 page, 2-5 pages, etc.)
	dprows, err := db.QueryContext(ctx, `
		WITH domain_counts AS (
			SELECT host, COUNT(*) AS cnt
			FROM doc_records
			WHERE host != ''
			GROUP BY host
		)
		SELECT
			CASE
				WHEN cnt = 1           THEN '1 page'
				WHEN cnt BETWEEN 2 AND 5   THEN '2–5'
				WHEN cnt BETWEEN 6 AND 20  THEN '6–20'
				WHEN cnt BETWEEN 21 AND 100 THEN '21–100'
				ELSE '>100'
			END AS bucket,
			COUNT(*) AS domains
		FROM domain_counts
		GROUP BY 1
		ORDER BY MIN(cnt)
	`)
	if err != nil {
		return ShardStatsResponse{}, fmt.Errorf("domain size buckets: %w", err)
	}
	defer dprows.Close()
	var domainSizeBuckets []sizeRow
	for dprows.Next() {
		var b sizeRow
		dprows.Scan(&b.Label, &b.Count)
		domainSizeBuckets = append(domainSizeBuckets, b)
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
		Shard:             shard,
		TotalDocs:         totalDocs,
		TotalSize:         totalSize,
		AvgSize:           avgSize,
		MinSize:           minSize,
		MaxSize:           maxSize,
		DateFrom:          minDate,
		DateTo:            maxDate,
		TotalDomains:      totalDomains,
		TopDomains:        domains,
		SizeBuckets:       sizeBuckets,
		DomainSizeBuckets: domainSizeBuckets,
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
	meta := extractDocMetadata(head)
	if meta.Title != "" {
		return meta.Title
	}

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

type docMetaExtract struct {
	Title string
	URL   string
	Host  string
	Date  string
}

// extractDocMetadata extracts best-effort metadata from markdown text.
// Supports simple "Key: Value" lines and link-first documents.
func extractDocMetadata(head []byte) docMetaExtract {
	var out docMetaExtract
	lines := bytes.Split(head, []byte("\n"))

	for _, rawLine := range lines {
		line := strings.TrimSpace(string(rawLine))
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "title:"):
			v := cleanInlineValue(line[len("title:"):])
			if v != "" && out.Title == "" {
				out.Title = v
			}
		case strings.HasPrefix(lower, "date:"):
			v := cleanInlineValue(line[len("date:"):])
			if v != "" && out.Date == "" {
				out.Date = v
			}
		case strings.HasPrefix(lower, "url:"):
			v := cleanInlineValue(line[len("url:"):])
			if v != "" && out.URL == "" {
				out.URL = strings.Trim(v, "<>")
			}
		case strings.HasPrefix(lower, "host:"):
			v := cleanInlineValue(line[len("host:"):])
			if v != "" && out.Host == "" {
				out.Host = strings.Trim(strings.Trim(v, "<>"), "/")
			}
		}
		if out.URL == "" {
			if title, linkURL, ok := parseMarkdownLinkLine(line); ok {
				out.URL = linkURL
				if out.Title == "" {
					out.Title = title
				}
			}
		}
		if out.Title != "" && out.URL != "" && out.Date != "" && out.Host != "" {
			break
		}
	}

	if out.Host == "" && out.URL != "" {
		out.Host = extractHost(out.URL)
	}
	return out
}

func cleanInlineValue(v string) string {
	s := strings.TrimSpace(v)
	s = strings.Trim(s, `"'`)
	return strings.TrimSpace(s)
}

func parseMarkdownLinkLine(line string) (title, linkURL string, ok bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "[") {
		return "", "", false
	}
	endText := strings.Index(line, "](")
	if endText <= 1 {
		return "", "", false
	}
	endURL := strings.Index(line[endText+2:], ")")
	if endURL <= 0 {
		return "", "", false
	}
	title = strings.TrimSpace(line[1:endText])
	linkURL = strings.TrimSpace(line[endText+2 : endText+2+endURL])
	linkURL = strings.Trim(linkURL, "<>")
	fields := strings.Fields(linkURL)
	if len(fields) == 0 {
		return "", "", false
	}
	linkURL = fields[0]
	if title == "" || linkURL == "" {
		return "", "", false
	}
	u, err := url.Parse(linkURL)
	if err != nil {
		return "", "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", false
	}
	return title, linkURL, true
}

// extractHost returns the hostname from a URL, stripping the www. prefix.
func extractHost(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	h := u.Hostname()
	h = strings.TrimPrefix(h, "www.")
	return h
}

func parseDocTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z0700",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"20060102150405",
		"20060102",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// ReadDocByOffset reads a single WARC record from warcMdPath at the given
// gzip member offset/size. Returns the markdown body.
// This is O(1) random access vs O(N) sequential scan.
func ReadDocByOffset(warcMdPath string, offset, size int64) ([]byte, error) {
	f, err := os.Open(warcMdPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to offset %d: %w", offset, err)
	}

	var r io.Reader = f
	if size > 0 {
		r = io.LimitReader(f, size)
	}

	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip open at offset %d: %w", offset, err)
	}
	defer gz.Close()

	wr := warcpkg.NewReader(gz)
	if !wr.Next() {
		if err := wr.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no WARC record at offset %d", offset)
	}
	return io.ReadAll(wr.Record().Body)
}

// readDocFromWARCMd scans warcMdPath sequentially for a record matching docID.
// Fallback for records without stored offsets.
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
