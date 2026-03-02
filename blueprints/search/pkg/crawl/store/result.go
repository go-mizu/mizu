package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	_ "github.com/duckdb/duckdb-go/v2"
	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
)

const defaultShardCount = 8

// ResultDB writes recrawl results to sharded DuckDB files in a directory.
// Each shard has its own async flusher goroutine, eliminating cross-shard contention.
// Uses batch multi-row VALUES inserts for maximum write throughput.
type ResultDB struct {
	dir           string
	shards        []*resultShard
	flushed       atomic.Int64
	memPerShardMB int // DuckDB memory_limit per shard; 0 treated as defaultDuckMemPerShardMB
}

// resultShard is one DuckDB file with its own buffer and flusher.
type resultShard struct {
	db      *sql.DB
	mu      sync.Mutex
	batch   []crawl.Result
	batchSz int
	flushCh chan []crawl.Result
	done    chan struct{}
}

// NewResultDB creates a sharded result DB in the given directory.
// Creates dir/results_000.duckdb through dir/results_NNN.duckdb.
// duckMemPerShardMB sets DuckDB memory_limit per shard (0 = use default 256 MB).
func NewResultDB(dir string, shardCount, batchSize, duckMemPerShardMB int) (*ResultDB, error) {
	if shardCount <= 0 {
		shardCount = defaultShardCount
	}
	if batchSize <= 0 {
		batchSize = 5000
	}
	if duckMemPerShardMB <= 0 {
		duckMemPerShardMB = 256
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating result dir: %w", err)
	}

	rdb := &ResultDB{
		dir:           dir,
		shards:        make([]*resultShard, shardCount),
		memPerShardMB: duckMemPerShardMB,
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
			flushCh: make(chan []crawl.Result, 16), // 16 batches × batchSz headroom before blocking
			done:    make(chan struct{}),
		}

		if err := initResultSchema(db, rdb.memPerShardMB); err != nil {
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

func initResultSchema(db *sql.DB, memMB int) error {
	// Cap DuckDB buffer pool per shard. checkpoint_threshold is scaled proportionally:
	// at memMB/40 we keep dirty pages << pool, so LRU eviction succeeds when checkpoint
	// pins a new 256 KB block. Both values are passed in from the caller for adaptive sizing.
	// Note: calling CHECKPOINT explicitly from Go causes DuckDB SIGSEGV — use threshold only.
	ckptMB := max(memMB/40, 4)
	if _, err := db.Exec(fmt.Sprintf("SET memory_limit='%dMB'", memMB)); err != nil {
		return fmt.Errorf("set memory_limit: %w", err)
	}
	// Limit DuckDB to 1 thread per shard (default = all CPU cores).
	// With 8 shards on a 4-core server, the default would spawn 32 threads.
	// Each thread has its own execution context that allocates memory outside the buffer pool.
	// Reducing to 1 thread cuts thread-local overhead by ~4× and avoids OOM.
	if _, err := db.Exec("SET threads=1"); err != nil {
		return fmt.Errorf("set threads: %w", err)
	}
	// Disabling insertion-order preservation reduces memory usage for large INSERTs
	// (DuckDB official suggestion for OOM scenarios).
	if _, err := db.Exec("SET preserve_insertion_order=false"); err != nil {
		return fmt.Errorf("set preserve_insertion_order: %w", err)
	}
	if _, err := db.Exec(fmt.Sprintf("SET checkpoint_threshold='%dMB'", ckptMB)); err != nil {
		slog.Default().Warn("resultdb: set checkpoint threshold", "err", err)
	}
	// No PRIMARY KEY: avoids ART index I/O on every INSERT (which keeps index pages
	// resident in RSS, causing OOM on the 5.9 GB server). Fresh crawl runs have
	// unique URLs from the parquet seed file; duplicates can be deduped post-hoc.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS results (
			url VARCHAR,
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
			status VARCHAR DEFAULT 'done',
			body_cid VARCHAR DEFAULT ''
		)
	`); err != nil {
		return err
	}
	// Add body_cid to existing DBs created before this schema version.
	_, _ = db.Exec(`ALTER TABLE results ADD COLUMN IF NOT EXISTS body_cid VARCHAR DEFAULT ''`)
	return nil
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
// Implements crawl.ResultWriter.
func (rdb *ResultDB) Add(r crawl.Result) {
	s := rdb.shards[rdb.shardFor(r.URL)]
	s.mu.Lock()
	s.batch = append(s.batch, r)
	if len(s.batch) >= s.batchSz {
		batch := s.batch
		s.batch = make([]crawl.Result, 0, s.batchSz)
		s.mu.Unlock()
		s.flushCh <- batch
		return
	}
	s.mu.Unlock()
}

// Flush sends all pending results across all shards to their async flushers.
// Implements crawl.ResultWriter.
func (rdb *ResultDB) Flush(_ context.Context) error {
	for _, s := range rdb.shards {
		s.mu.Lock()
		if len(s.batch) > 0 {
			batch := s.batch
			s.batch = make([]crawl.Result, 0, s.batchSz)
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
func writeBatchValues(db *sql.DB, batch []crawl.Result) int {
	const cols = 15
	const maxPerStmt = 400 // DuckDB param limit / cols; 400×15=6000, well within 32767

	written := 0
	for i := 0; i < len(batch); i += maxPerStmt {
		end := min(i+maxPerStmt, len(batch))
		chunk := batch[i:end]

		var b strings.Builder
		b.WriteString("INSERT INTO results (url, status_code, content_type, content_length, body, title, description, language, domain, redirect_url, fetch_time_ms, crawled_at, error, status, body_cid) VALUES ")
		args := make([]any, 0, len(chunk)*cols)

		for j, r := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			status := "done"
			if r.Error != "" {
				status = "failed"
			}
			args = append(args, sanitizeStr(r.URL), r.StatusCode, sanitizeStr(r.ContentType), r.ContentLength,
				"", sanitizeStr(r.Title), sanitizeStr(r.Description), sanitizeStr(r.Language),
				sanitizeStr(r.Domain), sanitizeStr(r.RedirectURL), r.FetchTimeMs, r.CrawledAt,
				sanitizeStr(r.Error), status, sanitizeStr(r.BodyCID))
		}

		if _, err := db.Exec(b.String(), args...); err != nil {
			slog.Default().Warn("resultdb: INSERT error", "rows", len(chunk), "err", err)
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

// ReopenShards closes and reopens each DuckDB shard connection, releasing the
// CGO buffer pool (~256 MB per shard × 8 shards = ~2 GB) which is invisible
// to Go's GC. Call between domain batches in batch/pipeline chunk modes.
// Implements crawl.ShardReopener.
func (rdb *ResultDB) ReopenShards() error {
	for i, s := range rdb.shards {
		// Flush any pending batch into the channel, then drain the flusher
		// goroutine by closing the channel and waiting for it to finish.
		// This avoids the race where len(flushCh)==0 but the goroutine hasn't
		// finished calling writeBatchValues yet.
		s.mu.Lock()
		if len(s.batch) > 0 {
			batch := s.batch
			s.batch = make([]crawl.Result, 0, s.batchSz)
			s.mu.Unlock()
			s.flushCh <- batch
		} else {
			s.mu.Unlock()
		}
		close(s.flushCh)
		<-s.done // wait for flusher to finish all writes

		// Close and reopen to release DuckDB's CGO buffer pool.
		path := filepath.Join(rdb.dir, fmt.Sprintf("results_%03d.duckdb", i))
		s.db.Close()
		db, err := sql.Open("duckdb", path)
		if err != nil {
			return fmt.Errorf("reopen shard %d: %w", i, err)
		}
		if err := applyShardSettings(db, rdb.memPerShardMB); err != nil {
			db.Close()
			return fmt.Errorf("reopen shard %d settings: %w", i, err)
		}
		s.db = db

		// Start a fresh flusher goroutine for the next batch.
		s.flushCh = make(chan []crawl.Result, 16)
		s.done = make(chan struct{})
		go s.flusher(&rdb.flushed)
	}
	return nil
}

// applyShardSettings applies DuckDB performance settings to a freshly opened connection.
func applyShardSettings(db *sql.DB, memMB int) error {
	ckptMB := max(memMB/40, 4)
	for _, stmt := range []string{
		fmt.Sprintf("SET memory_limit='%dMB'", memMB),
		"SET threads=1",
		"SET preserve_insertion_order=false",
		fmt.Sprintf("SET checkpoint_threshold='%dMB'", ckptMB),
	} {
		if _, err := db.Exec(stmt); err != nil {
			slog.Default().Warn("resultdb: apply shard settings", "stmt", stmt, "err", err)
		}
	}
	return nil
}

// Close flushes remaining results, waits for all flushers, and closes databases.
// Implements crawl.ResultWriter.
func (rdb *ResultDB) Close() error {
	rdb.Flush(context.Background())
	for _, s := range rdb.shards {
		close(s.flushCh)
		<-s.done
		s.db.Close()
	}
	return nil
}
