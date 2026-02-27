package recrawler

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	_ "github.com/duckdb/duckdb-go/v2"
)

const defaultShardCount = 8

// ResultDB writes recrawl results to sharded DuckDB files in a directory.
// Each shard has its own async flusher goroutine, eliminating cross-shard contention.
// Uses batch multi-row VALUES inserts for maximum write throughput.
type ResultDB struct {
	dir     string
	shards  []*resultShard
	flushed atomic.Int64
}

// resultShard is one DuckDB file with its own buffer and flusher.
type resultShard struct {
	db      *sql.DB
	mu      sync.Mutex
	batch   []Result
	batchSz int
	flushCh chan []Result
	done    chan struct{}
}

// NewResultDB creates a sharded result DB in the given directory.
// Creates dir/results_000.duckdb through dir/results_NNN.duckdb.
func NewResultDB(dir string, shardCount, batchSize int) (*ResultDB, error) {
	if shardCount <= 0 {
		shardCount = defaultShardCount
	}
	if batchSize <= 0 {
		batchSize = 5000
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
			db:      db,
			batchSz: batchSize,
			flushCh: make(chan []Result, 2),
			done:    make(chan struct{}),
		}

		if err := initResultSchema(db); err != nil {
			db.Close()
			rdb.closeOpenShards(i)
			return nil, fmt.Errorf("init shard %d schema: %w", i, err)
		}

		go s.flusher(&rdb.flushed)
		rdb.shards[i] = s
	}

	return rdb, nil
}

func (rdb *ResultDB) closeOpenShards(n int) {
	for i := range n {
		if rdb.shards[i] != nil {
			rdb.shards[i].db.Close()
		}
	}
}

func initResultSchema(db *sql.DB) error {
	// Cap DuckDB buffer pool at 128 MB per shard (16 shards × 128 MB = 2 GB total).
	// Without this, DuckDB aggressively buffers body data and exhausts server RAM.
	// DuckDB spills excess pages to a temp file, so this does not affect correctness.
	if _, err := db.Exec("SET memory_limit='128MB'"); err != nil {
		return fmt.Errorf("set memory_limit: %w", err)
	}
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS results (
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
			status VARCHAR DEFAULT 'done'
		)
	`)
	return err
}

// shardFor returns the shard index for a URL using FNV-1a hash.
func (rdb *ResultDB) shardFor(url string) int {
	h := uint32(2166136261)
	for i := 0; i < len(url); i++ {
		h ^= uint32(url[i])
		h *= 16777619
	}
	return int(h % uint32(len(rdb.shards)))
}

// Add queues a result for batch writing. Never blocks on DB I/O.
func (rdb *ResultDB) Add(r Result) {
	s := rdb.shards[rdb.shardFor(r.URL)]
	s.mu.Lock()
	s.batch = append(s.batch, r)
	if len(s.batch) >= s.batchSz {
		batch := s.batch
		s.batch = make([]Result, 0, s.batchSz)
		s.mu.Unlock()
		s.flushCh <- batch
		return
	}
	s.mu.Unlock()
}

// Flush sends all pending results across all shards to their async flushers.
func (rdb *ResultDB) Flush(_ context.Context) error {
	for _, s := range rdb.shards {
		s.mu.Lock()
		if len(s.batch) > 0 {
			batch := s.batch
			s.batch = make([]Result, 0, s.batchSz)
			s.mu.Unlock()
			s.flushCh <- batch
		} else {
			s.mu.Unlock()
		}
	}
	return nil
}

func (s *resultShard) flusher(flushed *atomic.Int64) {
	defer close(s.done)
	for batch := range s.flushCh {
		n := writeBatchValues(s.db, batch)
		flushed.Add(int64(n))
	}
}

// sanitizeStr ensures s is safe for DuckDB VARCHAR: no null bytes and valid UTF-8.
// Russian/CJK pages are often Windows-1251/Shift-JIS; the raw bytes cast to string
// produce invalid UTF-8 sequences that cause duckdb_bind_varchar to fail.
func sanitizeStr(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\x00", "")
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	return s
}

// writeBatchValues uses multi-row VALUES for high-throughput inserts.
// ~100x faster than row-by-row prepared statements.
// Returns the number of rows successfully written.
func writeBatchValues(db *sql.DB, batch []Result) int {
	const cols = 14
	const maxPerStmt = 500 // DuckDB param limit / cols

	written := 0
	for i := 0; i < len(batch); i += maxPerStmt {
		end := min(i+maxPerStmt, len(batch))
		chunk := batch[i:end]

		var b strings.Builder
		b.WriteString("INSERT OR REPLACE INTO results (url, status_code, content_type, content_length, body, title, description, language, domain, redirect_url, fetch_time_ms, crawled_at, error, status) VALUES ")
		args := make([]any, 0, len(chunk)*cols)

		for j, r := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			status := "done"
			if r.Error != "" {
				status = "failed"
			}
			args = append(args, sanitizeStr(r.URL), r.StatusCode, sanitizeStr(r.ContentType), r.ContentLength,
				sanitizeStr(r.Body), sanitizeStr(r.Title), sanitizeStr(r.Description), sanitizeStr(r.Language),
				sanitizeStr(r.Domain), sanitizeStr(r.RedirectURL), r.FetchTimeMs, r.CrawledAt,
				sanitizeStr(r.Error), status)
		}

		if _, err := db.Exec(b.String(), args...); err != nil {
			fmt.Fprintf(os.Stderr, "[resultdb] INSERT error (batch %d rows): %v\n", len(chunk), err)
		} else {
			written += len(chunk)
		}
	}
	return written
}

// SetMeta stores a key-value pair in shard 0's meta table.
func (rdb *ResultDB) SetMeta(_ context.Context, key, value string) error {
	db := rdb.shards[0].db
	db.Exec(`CREATE TABLE IF NOT EXISTS meta (key VARCHAR PRIMARY KEY, value VARCHAR)`)
	_, err := db.Exec("INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)", key, value)
	return err
}

// FlushedCount returns the total number of results written across all shards.
func (rdb *ResultDB) FlushedCount() int64 {
	return rdb.flushed.Load()
}

// PendingCount returns the total number of results not yet flushed.
func (rdb *ResultDB) PendingCount() int {
	total := 0
	for _, s := range rdb.shards {
		s.mu.Lock()
		total += len(s.batch)
		s.mu.Unlock()
	}
	return total
}

// Dir returns the result directory path.
func (rdb *ResultDB) Dir() string {
	return rdb.dir
}

// Close flushes remaining results, waits for all flushers, and closes databases.
func (rdb *ResultDB) Close() error {
	rdb.Flush(context.Background())
	for _, s := range rdb.shards {
		close(s.flushCh)
		<-s.done
		s.db.Close()
	}
	return nil
}
