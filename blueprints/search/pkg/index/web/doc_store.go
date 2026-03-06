package web

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// DocRecord is per-document metadata derived from a .md.warc.gz WARC record header.
// Body is NOT stored — only headers + first 256 bytes for title extraction.
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

// DocStore is a lightweight SQLite-backed store for per-document browse metadata.
type DocStore struct {
	db *sql.DB
}

// NewDocStore opens (or creates) the doc SQLite store at dbPath.
// Assumes the "sqlite" database/sql driver is already registered.
func NewDocStore(dbPath string) (*DocStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("doc_store: mkdir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("doc_store: open: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("doc_store: wal: %w", err)
	}
	db.Exec("PRAGMA busy_timeout = 5000")
	return &DocStore{db: db}, nil
}

// Init creates required tables if they don't exist.
func (ds *DocStore) Init(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS doc_records (
			doc_id         TEXT PRIMARY KEY,
			shard          TEXT NOT NULL,
			crawl_id       TEXT NOT NULL,
			url            TEXT NOT NULL DEFAULT '',
			title          TEXT NOT NULL DEFAULT '',
			crawl_date     TEXT,
			size_bytes     INTEGER DEFAULT 0,
			word_count     INTEGER DEFAULT 0,
			warc_record_id TEXT,
			refers_to      TEXT,
			scanned_at     TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_doc_records_shard ON doc_records(crawl_id, shard)`,
		`CREATE INDEX IF NOT EXISTS idx_doc_records_url ON doc_records(crawl_id, url)`,
		`CREATE TABLE IF NOT EXISTS doc_scan_meta (
			crawl_id         TEXT NOT NULL,
			shard            TEXT NOT NULL,
			total_docs       INTEGER DEFAULT 0,
			total_size_bytes INTEGER DEFAULT 0,
			last_doc_date    TEXT,
			last_scanned_at  TEXT NOT NULL,
			PRIMARY KEY (crawl_id, shard)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := ds.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("doc_store: init schema: %w", err)
		}
	}
	return nil
}

// Close closes the underlying database.
func (ds *DocStore) Close() error {
	if ds == nil || ds.db == nil {
		return nil
	}
	return ds.db.Close()
}

// ScanShard scans a single .md.warc.gz file and upserts DocRecords for crawlID/shard.
// Returns the total doc count for the shard after scanning.
func (ds *DocStore) ScanShard(ctx context.Context, crawlID, shard, warcMdPath string) (int64, error) {
	f, err := os.Open(warcMdPath)
	if err != nil {
		return 0, fmt.Errorf("doc_store scan open: %w", err)
	}
	defer f.Close()

	now := time.Now().UTC()
	r := warcpkg.NewReader(f)

	type scanRow struct {
		docID        string
		url          string
		title        string
		crawlDate    string
		sizeBytes    int64
		wordCount    int
		warcRecordID string
		refersTo     string
	}

	batch := make([]scanRow, 0, 256)
	var totalSizeBytes int64
	var lastDocDate time.Time

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		tx, err := ds.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO doc_records
				(doc_id, shard, crawl_id, url, title, crawl_date, size_bytes, word_count, warc_record_id, refers_to, scanned_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(doc_id) DO UPDATE SET
				url=excluded.url, title=excluded.title, crawl_date=excluded.crawl_date,
				size_bytes=excluded.size_bytes, word_count=excluded.word_count,
				warc_record_id=excluded.warc_record_id, refers_to=excluded.refers_to,
				scanned_at=excluded.scanned_at
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		nowStr := now.Format(time.RFC3339Nano)
		for _, row := range batch {
			if _, err := stmt.ExecContext(ctx,
				row.docID, shard, crawlID, row.url, row.title, row.crawlDate,
				row.sizeBytes, row.wordCount, row.warcRecordID, row.refersTo, nowStr,
			); err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	for r.Next() {
		if ctx.Err() != nil {
			return 0, ctx.Err()
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

		// Read first 256 bytes for title extraction; discard rest of body.
		head := make([]byte, 256)
		n, _ := rec.Body.Read(head)
		io.Copy(io.Discard, rec.Body)
		head = head[:n]

		title := extractDocTitle(head, targetURI)
		wordCount := int(sizeBytes / 5) // rough estimate: ~5 chars/word

		if t, err := time.Parse(time.RFC3339, dateStr); err == nil && t.After(lastDocDate) {
			lastDocDate = t
		}
		totalSizeBytes += sizeBytes

		batch = append(batch, scanRow{
			docID:        docID,
			url:          targetURI,
			title:        title,
			crawlDate:    dateStr,
			sizeBytes:    sizeBytes,
			wordCount:    wordCount,
			warcRecordID: recordID,
			refersTo:     refersTo,
		})

		if len(batch) >= 500 {
			if err := flushBatch(); err != nil {
				return 0, fmt.Errorf("doc_store flush: %w", err)
			}
		}
	}
	if err := r.Err(); err != nil {
		return 0, fmt.Errorf("doc_store scan: %w", err)
	}
	if err := flushBatch(); err != nil {
		return 0, fmt.Errorf("doc_store flush final: %w", err)
	}

	// Count total for this shard after upsert.
	var totalDocs int64
	ds.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM doc_records WHERE crawl_id=? AND shard=?`,
		crawlID, shard,
	).Scan(&totalDocs)

	lastDocDateStr := ""
	if !lastDocDate.IsZero() {
		lastDocDateStr = lastDocDate.UTC().Format(time.RFC3339)
	}
	if _, err := ds.db.ExecContext(ctx, `
		INSERT INTO doc_scan_meta (crawl_id, shard, total_docs, total_size_bytes, last_doc_date, last_scanned_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(crawl_id, shard) DO UPDATE SET
			total_docs=excluded.total_docs,
			total_size_bytes=excluded.total_size_bytes,
			last_doc_date=excluded.last_doc_date,
			last_scanned_at=excluded.last_scanned_at
	`, crawlID, shard, totalDocs, totalSizeBytes, lastDocDateStr, now.Format(time.RFC3339Nano)); err != nil {
		logErrorf("doc_store: update scan_meta shard=%s: %v", shard, err)
	}

	return totalDocs, nil
}

// ScanAll scans all .md.warc.gz files in warcMdBase for the given crawlID.
// Returns the total doc count across all shards.
func (ds *DocStore) ScanAll(ctx context.Context, crawlID, warcMdBase string) (int64, error) {
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
		n, err := ds.ScanShard(ctx, crawlID, shard, filepath.Join(warcMdBase, e.Name()))
		if err != nil {
			logErrorf("doc_store: scan shard=%s err=%v", shard, err)
			continue
		}
		total += n
		logInfof("doc_store: scanned shard=%s docs=%d", shard, n)
	}
	return total, nil
}

// ListShardMetas returns scan metadata for all scanned shards of a crawl.
func (ds *DocStore) ListShardMetas(ctx context.Context, crawlID string) ([]DocShardMeta, error) {
	rows, err := ds.db.QueryContext(ctx, `
		SELECT shard, total_docs, total_size_bytes, last_doc_date, last_scanned_at
		FROM doc_scan_meta WHERE crawl_id=? ORDER BY shard
	`, crawlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DocShardMeta
	for rows.Next() {
		var m DocShardMeta
		var lastDocDate, lastScannedAt string
		if err := rows.Scan(&m.Shard, &m.TotalDocs, &m.TotalSizeBytes, &lastDocDate, &lastScannedAt); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339, lastDocDate); err == nil {
			m.LastDocDate = t
		}
		if t, err := time.Parse(time.RFC3339Nano, lastScannedAt); err == nil {
			m.LastScannedAt = t
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetShardMeta returns scan metadata for one shard.
func (ds *DocStore) GetShardMeta(ctx context.Context, crawlID, shard string) (DocShardMeta, bool, error) {
	var m DocShardMeta
	m.Shard = shard
	var lastDocDate, lastScannedAt string
	err := ds.db.QueryRowContext(ctx, `
		SELECT total_docs, total_size_bytes, last_doc_date, last_scanned_at
		FROM doc_scan_meta WHERE crawl_id=? AND shard=?
	`, crawlID, shard).Scan(&m.TotalDocs, &m.TotalSizeBytes, &lastDocDate, &lastScannedAt)
	if err == sql.ErrNoRows {
		return m, false, nil
	}
	if err != nil {
		return m, false, err
	}
	if t, err := time.Parse(time.RFC3339, lastDocDate); err == nil {
		m.LastDocDate = t
	}
	if t, err := time.Parse(time.RFC3339Nano, lastScannedAt); err == nil {
		m.LastScannedAt = t
	}
	return m, true, nil
}

// ListDocs returns paginated DocRecords for a shard.
// q filters by URL or title (case-insensitive LIKE). sortBy: "date","size","words","title","url".
func (ds *DocStore) ListDocs(ctx context.Context, crawlID, shard string, page, pageSize int, q, sortBy string) ([]DocRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}

	orderClause := "ORDER BY crawl_date DESC"
	switch sortBy {
	case "size":
		orderClause = "ORDER BY size_bytes DESC"
	case "words":
		orderClause = "ORDER BY word_count DESC"
	case "title":
		orderClause = "ORDER BY title ASC"
	case "url":
		orderClause = "ORDER BY url ASC"
	}

	var whereExtra string
	baseArgs := []any{crawlID, shard}
	if q != "" {
		whereExtra = " AND (url LIKE ? OR title LIKE ?)"
		like := "%" + q + "%"
		baseArgs = append(baseArgs, like, like)
	}

	var total int64
	countArgs := append([]any{}, baseArgs...)
	ds.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM doc_records WHERE crawl_id=? AND shard=?`+whereExtra,
		countArgs...,
	).Scan(&total)

	queryArgs := append(append([]any{}, baseArgs...), pageSize, (page-1)*pageSize)
	rows, err := ds.db.QueryContext(ctx,
		`SELECT doc_id, url, title, crawl_date, size_bytes, word_count
		 FROM doc_records WHERE crawl_id=? AND shard=?`+whereExtra+`
		 `+orderClause+` LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []DocRecord
	for rows.Next() {
		var d DocRecord
		var crawlDateStr string
		if err := rows.Scan(&d.DocID, &d.URL, &d.Title, &crawlDateStr, &d.SizeBytes, &d.WordCount); err != nil {
			return nil, 0, err
		}
		d.Shard = shard
		if t, err := time.Parse(time.RFC3339, crawlDateStr); err == nil {
			d.CrawlDate = t
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

// GetDoc returns metadata for a single doc.
func (ds *DocStore) GetDoc(ctx context.Context, crawlID, shard, docID string) (DocRecord, bool, error) {
	var d DocRecord
	var crawlDateStr string
	err := ds.db.QueryRowContext(ctx, `
		SELECT doc_id, url, title, crawl_date, size_bytes, word_count, warc_record_id, refers_to
		FROM doc_records WHERE crawl_id=? AND shard=? AND doc_id=?
	`, crawlID, shard, docID).Scan(
		&d.DocID, &d.URL, &d.Title, &crawlDateStr,
		&d.SizeBytes, &d.WordCount, &d.WARCRecordID, &d.RefersTo,
	)
	if err == sql.ErrNoRows {
		return d, false, nil
	}
	if err != nil {
		return d, false, err
	}
	d.Shard = shard
	if t, err := time.Parse(time.RFC3339, crawlDateStr); err == nil {
		d.CrawlDate = t
	}
	return d, true, nil
}

// ── Helper functions ──────────────────────────────────────────────────────────

// warcRecordIDtoDocID extracts the UUID from a WARC-Record-ID like "<urn:uuid:...>".
func warcRecordIDtoDocID(recordID string) string {
	s := strings.TrimPrefix(recordID, "<urn:uuid:")
	s = strings.TrimSuffix(s, ">")
	s = strings.TrimSpace(s)
	// Sanity: UUIDs don't contain ':' or '<'
	if strings.ContainsAny(s, ":<>") {
		return ""
	}
	return s
}

// extractDocTitle extracts the first Markdown H1 or H2 title from head bytes.
// Falls back to URL hostname, then raw URL.
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

// readDocFromWARCMd scans warcMdPath for a record whose WARC-Record-ID matches docID
// and returns the full markdown body.
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

// listWARCMdShards returns sorted shard names (without .md.warc.gz) from warcMdBase.
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
