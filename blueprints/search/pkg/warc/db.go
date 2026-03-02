package warc

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

	_ "github.com/duckdb/duckdb-go/v2"
)

// WARCRecord is a flattened WARC record suitable for DuckDB storage.
type WARCRecord struct {
	WARCFile    string
	RecordID    string
	RecordType  string
	URL         string
	Domain      string
	CrawledAt   time.Time
	HTTPStatus  int
	MIMEType    string
	Language    string
	Title       string
	Description string
	Body        []byte
	BodyLength  int64
	HTTPHeaders string // JSON-encoded map[string]string
}

const warcShardCount = 8

// DBShardCount is the number of DuckDB shards created by OpenRecordDB.
const DBShardCount = warcShardCount

// RecordDB is an 8-shard DuckDB store for WARCRecord rows.
type RecordDB struct {
	dir     string
	shards  []*warcShard
	flushed atomic.Int64
	closeOnce sync.Once
}

type warcShard struct {
	db      *sql.DB
	mu      sync.Mutex
	batch   []WARCRecord
	batchSz int
	flushCh chan []WARCRecord
	done    chan struct{}
}

// OpenRecordDB opens (or creates) the sharded DuckDB store in dir.
func OpenRecordDB(dir string, batchSize int) (*RecordDB, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("warc: creating record db dir: %w", err)
	}

	rdb := &RecordDB{
		dir:    dir,
		shards: make([]*warcShard, warcShardCount),
	}

	for i := range warcShardCount {
		path := filepath.Join(dir, fmt.Sprintf("warc_%03d.duckdb", i))
		db, err := sql.Open("duckdb", path)
		if err != nil {
			rdb.closeShards(i)
			return nil, fmt.Errorf("warc: opening shard %d: %w", i, err)
		}
		if err := initWARCSchema(db); err != nil {
			db.Close()
			rdb.closeShards(i)
			return nil, fmt.Errorf("warc: init shard %d schema: %w", i, err)
		}
		s := &warcShard{
			db:      db,
			batchSz: batchSize,
			flushCh: make(chan []WARCRecord, 16),
			done:    make(chan struct{}),
		}
		go s.flusher(&rdb.flushed)
		rdb.shards[i] = s
	}
	return rdb, nil
}

func initWARCSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS records (
			warc_file    VARCHAR,
			record_id    VARCHAR,
			record_type  VARCHAR,
			url          VARCHAR,
			domain       VARCHAR,
			crawled_at   TIMESTAMP,
			http_status  INTEGER,
			mime_type    VARCHAR,
			language     VARCHAR,
			title        VARCHAR,
			description  VARCHAR,
			body         BLOB,
			body_length  BIGINT,
			http_headers VARCHAR
		)
	`)
	return err
}

func (rdb *RecordDB) shardFor(url string) int {
	h := uint32(2166136261)
	for i := 0; i < len(url); i++ {
		h ^= uint32(url[i])
		h *= 16777619
	}
	return int(h % warcShardCount)
}

// Insert queues WARCRecords for batch writing.
func (rdb *RecordDB) Insert(recs []WARCRecord) {
	for _, r := range recs {
		s := rdb.shards[rdb.shardFor(r.URL)]
		s.mu.Lock()
		s.batch = append(s.batch, r)
		if len(s.batch) >= s.batchSz {
			batch := s.batch
			s.batch = make([]WARCRecord, 0, s.batchSz)
			s.mu.Unlock()
			s.flushCh <- batch
			continue
		}
		s.mu.Unlock()
	}
}

// Flush sends all pending batches to flusher goroutines.
func (rdb *RecordDB) Flush(_ context.Context) {
	for _, s := range rdb.shards {
		s.mu.Lock()
		if len(s.batch) > 0 {
			batch := s.batch
			s.batch = make([]WARCRecord, 0, s.batchSz)
			s.mu.Unlock()
			s.flushCh <- batch
		} else {
			s.mu.Unlock()
		}
	}
}

// FlushedCount returns total records written.
func (rdb *RecordDB) FlushedCount() int64 { return rdb.flushed.Load() }

// Dir returns the database directory path.
func (rdb *RecordDB) Dir() string { return rdb.dir }

// Close flushes and shuts down all shards. Safe to call multiple times.
func (rdb *RecordDB) Close() error {
	var closeErr error
	rdb.closeOnce.Do(func() {
		rdb.Flush(context.Background())
		for _, s := range rdb.shards {
			close(s.flushCh)
			<-s.done
			s.db.Close()
		}
	})
	return closeErr
}

func (rdb *RecordDB) closeShards(n int) {
	for i := range n {
		if rdb.shards[i] != nil {
			rdb.shards[i].db.Close()
		}
	}
}

func (s *warcShard) flusher(flushed *atomic.Int64) {
	defer close(s.done)
	for batch := range s.flushCh {
		writeWARCBatch(s.db, batch)
		flushed.Add(int64(len(batch)))
	}
}

func writeWARCBatch(db *sql.DB, batch []WARCRecord) {
	const cols = 14
	const maxPer = 500
	for i := 0; i < len(batch); i += maxPer {
		end := min(i+maxPer, len(batch))
		chunk := batch[i:end]
		var b strings.Builder
		b.WriteString("INSERT INTO records VALUES ")
		args := make([]any, 0, len(chunk)*cols)
		for j, r := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args,
				r.WARCFile, r.RecordID, r.RecordType, r.URL, r.Domain,
				r.CrawledAt, r.HTTPStatus, r.MIMEType, r.Language,
				r.Title, r.Description, r.Body, r.BodyLength, r.HTTPHeaders)
		}
		db.Exec(b.String(), args...)
	}
}
