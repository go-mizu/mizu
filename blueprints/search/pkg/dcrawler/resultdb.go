package dcrawler

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ResultDB writes crawl results to sharded DuckDB files.
type ResultDB struct {
	dir     string
	shards  []*resultShard
	flushed atomic.Int64
}

type resultShard struct {
	db        *sql.DB
	mu        sync.Mutex
	pageBatch []Result
	linkBatch []Link
	batchSz   int
	pageFlush chan []Result
	linkFlush chan []Link
	done      chan struct{}
}

// NewResultDB creates a sharded result DB in the given directory.
func NewResultDB(dir string, shardCount, batchSize int) (*ResultDB, error) {
	if shardCount <= 0 {
		shardCount = 8
	}
	if batchSize <= 0 {
		batchSize = 500
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating result dir: %w", err)
	}

	rdb := &ResultDB{
		dir:    dir,
		shards: make([]*resultShard, shardCount),
	}

	for i := range shardCount {
		path := filepath.Join(dir, fmt.Sprintf("results_%03d.duckdb", i))
		db, err := sql.Open("duckdb", path)
		if err != nil {
			rdb.closeOpenShards(i)
			return nil, fmt.Errorf("opening shard %d: %w", i, err)
		}
		s := &resultShard{
			db:        db,
			batchSz:   batchSize,
			pageFlush: make(chan []Result, 16),
			linkFlush: make(chan []Link, 16),
			done:      make(chan struct{}),
		}
		if err := initPageSchema(db); err != nil {
			db.Close()
			rdb.closeOpenShards(i)
			return nil, fmt.Errorf("init shard %d schema: %w", i, err)
		}
		go s.flusher(&rdb.flushed)
		rdb.shards[i] = s
	}
	return rdb, nil
}

func initPageSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS pages (
			url              VARCHAR PRIMARY KEY,
			url_hash         UBIGINT NOT NULL,
			depth            INTEGER DEFAULT 0,
			status_code      SMALLINT,
			content_type     VARCHAR,
			content_length   BIGINT,
			body_hash        UBIGINT,
			body             BLOB,
			title            VARCHAR,
			description      VARCHAR,
			language         VARCHAR,
			canonical        VARCHAR,
			etag             VARCHAR,
			last_modified    VARCHAR,
			server           VARCHAR,
			redirect_url     VARCHAR,
			link_count       INTEGER DEFAULT 0,
			fetch_time_ms    BIGINT,
			crawled_at       TIMESTAMP NOT NULL,
			error            VARCHAR
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS links (
			source_hash  UBIGINT NOT NULL,
			target_url   VARCHAR NOT NULL,
			anchor_text  VARCHAR,
			rel          VARCHAR,
			is_internal  BOOLEAN
		)
	`)
	return err
}

func (rdb *ResultDB) closeOpenShards(n int) {
	for i := range n {
		if rdb.shards[i] != nil {
			rdb.shards[i].db.Close()
		}
	}
}

func (rdb *ResultDB) shardFor(url string) int {
	h := uint32(2166136261)
	for i := 0; i < len(url); i++ {
		h ^= uint32(url[i])
		h *= 16777619
	}
	return int(h % uint32(len(rdb.shards)))
}

// AddPage queues a page result for batch writing.
func (rdb *ResultDB) AddPage(r Result) {
	s := rdb.shards[rdb.shardFor(r.URL)]
	s.mu.Lock()
	s.pageBatch = append(s.pageBatch, r)
	if len(s.pageBatch) >= s.batchSz {
		batch := s.pageBatch
		s.pageBatch = make([]Result, 0, s.batchSz)
		s.mu.Unlock()
		s.pageFlush <- batch
		return
	}
	s.mu.Unlock()
}

// AddLinks queues links for batch writing (sharded by source hash).
func (rdb *ResultDB) AddLinks(sourceHash uint64, links []Link) {
	groups := make(map[int][]Link)
	for _, l := range links {
		l.SourceHash = sourceHash
		idx := rdb.shardFor(l.TargetURL)
		groups[idx] = append(groups[idx], l)
	}
	for idx, batch := range groups {
		s := rdb.shards[idx]
		s.mu.Lock()
		s.linkBatch = append(s.linkBatch, batch...)
		if len(s.linkBatch) >= s.batchSz*5 {
			b := s.linkBatch
			s.linkBatch = make([]Link, 0, s.batchSz*5)
			s.mu.Unlock()
			s.linkFlush <- b
			continue
		}
		s.mu.Unlock()
	}
}

// Flush sends all pending data to async flushers.
func (rdb *ResultDB) Flush() {
	for _, s := range rdb.shards {
		s.mu.Lock()
		if len(s.pageBatch) > 0 {
			batch := s.pageBatch
			s.pageBatch = nil
			s.mu.Unlock()
			s.pageFlush <- batch
		} else {
			s.mu.Unlock()
		}
		s.mu.Lock()
		if len(s.linkBatch) > 0 {
			batch := s.linkBatch
			s.linkBatch = nil
			s.mu.Unlock()
			s.linkFlush <- batch
		} else {
			s.mu.Unlock()
		}
	}
}

// flusher drains page and link channels, writing batches to DuckDB.
// Exits when BOTH channels are closed.
func (s *resultShard) flusher(flushed *atomic.Int64) {
	defer close(s.done)
	pf := s.pageFlush
	lf := s.linkFlush
	for pf != nil || lf != nil {
		select {
		case batch, ok := <-pf:
			if !ok {
				pf = nil
				continue
			}
			writePageBatch(s.db, batch)
			flushed.Add(int64(len(batch)))
		case batch, ok := <-lf:
			if !ok {
				lf = nil
				continue
			}
			writeLinkBatch(s.db, batch)
		}
	}
}

func writePageBatch(db *sql.DB, batch []Result) {
	const cols = 20
	const maxPerStmt = 250

	for i := 0; i < len(batch); i += maxPerStmt {
		end := min(i+maxPerStmt, len(batch))
		chunk := batch[i:end]

		var b strings.Builder
		b.WriteString("INSERT OR REPLACE INTO pages (url, url_hash, depth, status_code, content_type, content_length, body_hash, body, title, description, language, canonical, etag, last_modified, server, redirect_url, link_count, fetch_time_ms, crawled_at, error) VALUES ")
		args := make([]any, 0, len(chunk)*cols)

		for j, r := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args, r.URL, r.URLHash, r.Depth, r.StatusCode,
				r.ContentType, r.ContentLength, r.BodyHash, r.BodyCompressed,
				r.Title, r.Description, r.Language, r.Canonical,
				r.ETag, r.LastModified, r.Server, r.RedirectURL,
				r.LinkCount, r.FetchTimeMs, r.CrawledAt, r.Error)
		}
		db.Exec(b.String(), args...)
	}
}

func writeLinkBatch(db *sql.DB, batch []Link) {
	const cols = 5
	const maxPerStmt = 500

	for i := 0; i < len(batch); i += maxPerStmt {
		end := min(i+maxPerStmt, len(batch))
		chunk := batch[i:end]

		var b strings.Builder
		b.WriteString("INSERT INTO links (source_hash, target_url, anchor_text, rel, is_internal) VALUES ")
		args := make([]any, 0, len(chunk)*cols)

		for j, l := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?,?)")
			args = append(args, l.SourceHash, l.TargetURL, l.AnchorText, l.Rel, l.IsInternal)
		}
		db.Exec(b.String(), args...)
	}
}

// SetMeta stores a key-value pair in shard 0's meta table.
func (rdb *ResultDB) SetMeta(key, value string) error {
	db := rdb.shards[0].db
	db.Exec(`CREATE TABLE IF NOT EXISTS meta (key VARCHAR PRIMARY KEY, value VARCHAR)`)
	_, err := db.Exec("INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)", key, value)
	return err
}

// FlushedCount returns the total number of pages written across all shards.
func (rdb *ResultDB) FlushedCount() int64 {
	return rdb.flushed.Load()
}

// Dir returns the result directory path.
func (rdb *ResultDB) Dir() string {
	return rdb.dir
}

// LoadExistingURLs reads all URLs from existing shard files for resume mode.
func (rdb *ResultDB) LoadExistingURLs(markSeen func(string)) (int, error) {
	count := 0
	for i := range len(rdb.shards) {
		path := filepath.Join(rdb.dir, fmt.Sprintf("results_%03d.duckdb", i))
		if _, err := os.Stat(path); err != nil {
			continue
		}
		db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
		if err != nil {
			continue
		}
		rows, err := db.Query("SELECT url FROM pages")
		if err != nil {
			db.Close()
			continue
		}
		for rows.Next() {
			var u string
			if err := rows.Scan(&u); err == nil {
				markSeen(u)
				count++
			}
		}
		rows.Close()
		db.Close()
	}
	return count, nil
}

// Close flushes remaining data and closes all shards.
func (rdb *ResultDB) Close() error {
	// Flush any remaining partial batches into the channels
	rdb.Flush()
	// Close both channels so flushers drain and exit
	for _, s := range rdb.shards {
		close(s.pageFlush)
		close(s.linkFlush)
	}
	// Wait for all flushers to finish writing
	for _, s := range rdb.shards {
		<-s.done
		s.db.Close()
	}
	return nil
}
