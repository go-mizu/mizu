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

const defaultShardCount = 8

// ResultDB writes page extraction results to sharded DuckDB files.
type ResultDB struct {
	dir     string
	shards  []*resultShard
	flushed atomic.Int64
}

type resultShard struct {
	db      *sql.DB
	mu      sync.Mutex
	batch   []PageResult
	batchSz int
	flushCh chan []PageResult
	done    chan struct{}
}

// NewResultDB creates a sharded result DB in the given directory.
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
			flushCh: make(chan []PageResult, 16),
			done:    make(chan struct{}),
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

func (rdb *ResultDB) closeOpenShards(n int) {
	for i := range n {
		if rdb.shards[i] != nil {
			rdb.shards[i].db.Close()
		}
	}
}

func initPageSchema(db *sql.DB) error {
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
			warc_filename VARCHAR,
			fetch_time_ms BIGINT,
			crawled_at TIMESTAMP,
			error VARCHAR,
			status VARCHAR DEFAULT 'done'
		)
	`)
	return err
}

func (rdb *ResultDB) shardFor(url string) int {
	h := uint32(2166136261)
	for i := 0; i < len(url); i++ {
		h ^= uint32(url[i])
		h *= 16777619
	}
	return int(h % uint32(len(rdb.shards)))
}

// Add queues a result for batch writing.
func (rdb *ResultDB) Add(r PageResult) {
	s := rdb.shards[rdb.shardFor(r.URL)]
	s.mu.Lock()
	s.batch = append(s.batch, r)
	if len(s.batch) >= s.batchSz {
		batch := s.batch
		s.batch = make([]PageResult, 0, s.batchSz)
		s.mu.Unlock()
		s.flushCh <- batch
		return
	}
	s.mu.Unlock()
}

// Flush sends all pending results to their async flushers.
func (rdb *ResultDB) Flush(_ context.Context) error {
	for _, s := range rdb.shards {
		s.mu.Lock()
		if len(s.batch) > 0 {
			batch := s.batch
			s.batch = make([]PageResult, 0, s.batchSz)
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
		writePageBatch(s.db, batch)
		flushed.Add(int64(len(batch)))
	}
}

func writePageBatch(db *sql.DB, batch []PageResult) {
	const cols = 14
	const maxPerStmt = 500

	for i := 0; i < len(batch); i += maxPerStmt {
		end := min(i+maxPerStmt, len(batch))
		chunk := batch[i:end]

		var b strings.Builder
		b.WriteString("INSERT OR REPLACE INTO results (url, status_code, content_type, content_length, body, title, description, language, domain, warc_filename, fetch_time_ms, crawled_at, error, status) VALUES ")
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
			args = append(args, r.URL, r.StatusCode, r.ContentType, r.ContentLength,
				r.Body, r.Title, r.Description, r.Language, r.Domain,
				r.WARCFilename, r.FetchTimeMs, r.CrawledAt, r.Error, status)
		}

		db.Exec(b.String(), args...)
	}
}

// SetMeta stores a key-value pair in shard 0's meta table.
func (rdb *ResultDB) SetMeta(_ context.Context, key, value string) error {
	db := rdb.shards[0].db
	db.Exec(`CREATE TABLE IF NOT EXISTS meta (key VARCHAR PRIMARY KEY, value VARCHAR)`)
	_, err := db.Exec("INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)", key, value)
	return err
}

// FlushedCount returns the total number of results written.
func (rdb *ResultDB) FlushedCount() int64 {
	return rdb.flushed.Load()
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

// LoadAlreadyFetched scans result shard files for URLs already fetched.
func LoadAlreadyFetched(ctx context.Context, dir string) (map[string]bool, error) {
	done := make(map[string]bool)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return done, nil
		}
		return nil, err
	}

	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "results_") || !strings.HasSuffix(e.Name(), ".duckdb") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		db, err := sql.Open("duckdb", path+"?access_mode=read_only")
		if err != nil {
			continue
		}

		// Check if results table exists
		var tableName string
		if err := db.QueryRowContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_name='results'").Scan(&tableName); err != nil {
			db.Close()
			continue
		}

		rows, err := db.QueryContext(ctx, "SELECT url FROM results")
		if err != nil {
			db.Close()
			continue
		}
		for rows.Next() {
			var url string
			rows.Scan(&url)
			done[url] = true
		}
		rows.Close()
		db.Close()
	}

	return done, nil
}
