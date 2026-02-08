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
	"time"
)

// SitePage is a page extracted from Common Crawl for a specific site.
type SitePage struct {
	URL           string
	StatusCode    int
	ContentType   string
	ContentLength int64
	Title         string
	Description   string
	Language      string
	CrawlID       string
	CrawledAt     time.Time
	WARCFilename  string
	WARCOffset    int64
	WARCLength    int64
	Body          string
	FetchTimeMs   int64
	Error         string
}

// SiteDB stores site extraction results in a single DuckDB file.
type SiteDB struct {
	db   *sql.DB
	path string

	pageMu  sync.Mutex
	pageBuf []SitePage
	linkMu  sync.Mutex
	linkBuf []SiteLink

	batchSz   int
	pageFlush chan []SitePage
	linkFlush chan []SiteLink
	done      chan struct{}

	pagesFlushed atomic.Int64
	linksFlushed atomic.Int64
}

// NewSiteDB creates a site database in the given directory.
func NewSiteDB(dir string, batchSize int) (*SiteDB, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating site dir: %w", err)
	}

	path := filepath.Join(dir, "site.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("opening site db: %w", err)
	}

	if err := initSiteSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	sdb := &SiteDB{
		db:        db,
		path:      path,
		batchSz:   batchSize,
		pageFlush: make(chan []SitePage, 16),
		linkFlush: make(chan []SiteLink, 16),
		done:      make(chan struct{}),
	}

	go sdb.flusher()

	return sdb, nil
}

func initSiteSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS pages (
			url             VARCHAR PRIMARY KEY,
			status_code     INTEGER,
			content_type    VARCHAR,
			content_length  BIGINT,
			title           VARCHAR,
			description     VARCHAR,
			language        VARCHAR,
			crawl_id        VARCHAR,
			crawled_at      TIMESTAMP,
			warc_filename   VARCHAR,
			warc_offset     BIGINT,
			warc_length     BIGINT,
			body            VARCHAR,
			fetch_time_ms   BIGINT,
			error           VARCHAR
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS links (
			source_url  VARCHAR,
			target_url  VARCHAR,
			anchor_text VARCHAR,
			rel         VARCHAR,
			is_internal BOOLEAN
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS meta (key VARCHAR PRIMARY KEY, value VARCHAR)`)
	return err
}

// AddPage queues a page for batch writing.
func (sdb *SiteDB) AddPage(p SitePage) {
	sdb.pageMu.Lock()
	sdb.pageBuf = append(sdb.pageBuf, p)
	if len(sdb.pageBuf) >= sdb.batchSz {
		batch := sdb.pageBuf
		sdb.pageBuf = make([]SitePage, 0, sdb.batchSz)
		sdb.pageMu.Unlock()
		sdb.pageFlush <- batch
		return
	}
	sdb.pageMu.Unlock()
}

// AddLinks queues links for batch writing.
func (sdb *SiteDB) AddLinks(links []SiteLink) {
	if len(links) == 0 {
		return
	}
	sdb.linkMu.Lock()
	sdb.linkBuf = append(sdb.linkBuf, links...)
	if len(sdb.linkBuf) >= sdb.batchSz {
		batch := sdb.linkBuf
		sdb.linkBuf = make([]SiteLink, 0, sdb.batchSz)
		sdb.linkMu.Unlock()
		sdb.linkFlush <- batch
		return
	}
	sdb.linkMu.Unlock()
}

func (sdb *SiteDB) flusher() {
	defer close(sdb.done)

	pf := sdb.pageFlush
	lf := sdb.linkFlush

	for pf != nil || lf != nil {
		select {
		case batch, ok := <-pf:
			if !ok {
				pf = nil
				continue
			}
			sdb.writePageBatch(batch)
			sdb.pagesFlushed.Add(int64(len(batch)))
		case batch, ok := <-lf:
			if !ok {
				lf = nil
				continue
			}
			sdb.writeLinkBatch(batch)
			sdb.linksFlushed.Add(int64(len(batch)))
		}
	}
}

func (sdb *SiteDB) writePageBatch(batch []SitePage) {
	const maxPerStmt = 500

	for i := 0; i < len(batch); i += maxPerStmt {
		end := min(i+maxPerStmt, len(batch))
		chunk := batch[i:end]

		var b strings.Builder
		b.WriteString("INSERT OR REPLACE INTO pages (url, status_code, content_type, content_length, title, description, language, crawl_id, crawled_at, warc_filename, warc_offset, warc_length, body, fetch_time_ms, error) VALUES ")
		args := make([]any, 0, len(chunk)*15)

		for j, p := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args, p.URL, p.StatusCode, p.ContentType, p.ContentLength,
				p.Title, p.Description, p.Language, p.CrawlID,
				p.CrawledAt, p.WARCFilename, p.WARCOffset, p.WARCLength,
				p.Body, p.FetchTimeMs, p.Error)
		}

		sdb.db.Exec(b.String(), args...)
	}
}

func (sdb *SiteDB) writeLinkBatch(batch []SiteLink) {
	const maxPerStmt = 500

	for i := 0; i < len(batch); i += maxPerStmt {
		end := min(i+maxPerStmt, len(batch))
		chunk := batch[i:end]

		var b strings.Builder
		b.WriteString("INSERT INTO links (source_url, target_url, anchor_text, rel, is_internal) VALUES ")
		args := make([]any, 0, len(chunk)*5)

		for j, l := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?,?)")
			args = append(args, l.SourceURL, l.TargetURL, l.AnchorText, l.Rel, l.IsInternal)
		}

		sdb.db.Exec(b.String(), args...)
	}
}

// Flush sends all pending batches to the flusher.
func (sdb *SiteDB) Flush(_ context.Context) error {
	sdb.pageMu.Lock()
	if len(sdb.pageBuf) > 0 {
		batch := sdb.pageBuf
		sdb.pageBuf = nil
		sdb.pageMu.Unlock()
		sdb.pageFlush <- batch
	} else {
		sdb.pageMu.Unlock()
	}

	sdb.linkMu.Lock()
	if len(sdb.linkBuf) > 0 {
		batch := sdb.linkBuf
		sdb.linkBuf = nil
		sdb.linkMu.Unlock()
		sdb.linkFlush <- batch
	} else {
		sdb.linkMu.Unlock()
	}

	return nil
}

// SetMeta stores a key-value pair in the meta table.
func (sdb *SiteDB) SetMeta(key, value string) {
	sdb.db.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)`, key, value)
}

// PageCount returns the number of pages flushed.
func (sdb *SiteDB) PageCount() int64 {
	return sdb.pagesFlushed.Load()
}

// LinkCount returns the number of links flushed.
func (sdb *SiteDB) LinkCount() int64 {
	return sdb.linksFlushed.Load()
}

// Path returns the database file path.
func (sdb *SiteDB) Path() string {
	return sdb.path
}

// Close flushes remaining data and closes the database.
func (sdb *SiteDB) Close() error {
	sdb.Flush(context.Background())
	close(sdb.pageFlush)
	close(sdb.linkFlush)
	<-sdb.done
	return sdb.db.Close()
}

// LoadAlreadyExtracted returns URLs already in the site database (for resume).
func LoadAlreadyExtracted(dir string) (map[string]bool, error) {
	path := filepath.Join(dir, "site.duckdb")
	if _, err := os.Stat(path); err != nil {
		return make(map[string]bool), nil
	}

	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		return make(map[string]bool), nil
	}
	defer db.Close()

	var tableName string
	if err := db.QueryRow("SELECT table_name FROM information_schema.tables WHERE table_name='pages'").Scan(&tableName); err != nil {
		return make(map[string]bool), nil
	}

	rows, err := db.Query("SELECT url FROM pages WHERE error IS NULL OR error = ''")
	if err != nil {
		return make(map[string]bool), nil
	}
	defer rows.Close()

	done := make(map[string]bool)
	for rows.Next() {
		var u string
		rows.Scan(&u)
		done[u] = true
	}
	return done, nil
}
