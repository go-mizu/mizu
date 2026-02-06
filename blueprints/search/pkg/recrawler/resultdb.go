package recrawler

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ResultDB writes recrawl results and state to DuckDB.
// Uses an async flush goroutine so workers never block on DB I/O.
type ResultDB struct {
	resultDB *sql.DB
	stateDB  *sql.DB

	mu      sync.Mutex
	batch   []Result
	batchSz int
	flushed atomic.Int64

	// Async flush channel
	flushCh chan []Result
	done    chan struct{}
}

// NewResultDB opens (or creates) the result and state DuckDB files.
func NewResultDB(resultPath, statePath string, batchSize int) (*ResultDB, error) {
	resultDB, err := sql.Open("duckdb", resultPath)
	if err != nil {
		return nil, fmt.Errorf("opening result db: %w", err)
	}

	stateDB, err := sql.Open("duckdb", statePath)
	if err != nil {
		resultDB.Close()
		return nil, fmt.Errorf("opening state db: %w", err)
	}

	rdb := &ResultDB{
		resultDB: resultDB,
		stateDB:  stateDB,
		batchSz:  batchSize,
		flushCh:  make(chan []Result, 16), // buffer up to 16 pending batches
		done:     make(chan struct{}),
	}

	if err := rdb.initSchema(); err != nil {
		resultDB.Close()
		stateDB.Close()
		return nil, err
	}

	// Start async flusher goroutine
	go rdb.flusher()

	return rdb, nil
}

func (rdb *ResultDB) initSchema() error {
	_, err := rdb.resultDB.Exec(`
		CREATE TABLE IF NOT EXISTS results (
			url VARCHAR PRIMARY KEY,
			status_code INTEGER,
			content_type VARCHAR,
			content_length BIGINT,
			title VARCHAR,
			description VARCHAR,
			language VARCHAR,
			domain VARCHAR,
			redirect_url VARCHAR,
			fetch_time_ms BIGINT,
			crawled_at TIMESTAMP,
			error VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("creating results table: %w", err)
	}

	_, err = rdb.stateDB.Exec(`
		CREATE TABLE IF NOT EXISTS state (
			url VARCHAR PRIMARY KEY,
			status VARCHAR DEFAULT 'pending',
			status_code INTEGER,
			error VARCHAR,
			fetched_at TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("creating state table: %w", err)
	}

	_, err = rdb.stateDB.Exec(`
		CREATE TABLE IF NOT EXISTS meta (
			key VARCHAR PRIMARY KEY,
			value VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("creating meta table: %w", err)
	}

	return nil
}

// flusher runs as a goroutine, draining batches from flushCh and writing to DB.
func (rdb *ResultDB) flusher() {
	defer close(rdb.done)
	for batch := range rdb.flushCh {
		rdb.writeBatch(context.Background(), batch)
		rdb.writeState(context.Background(), batch)
		rdb.flushed.Add(int64(len(batch)))
	}
}

// Add queues a result for batch writing. Never blocks on DB I/O.
func (rdb *ResultDB) Add(r Result) {
	rdb.mu.Lock()
	rdb.batch = append(rdb.batch, r)
	if len(rdb.batch) >= rdb.batchSz {
		batch := rdb.batch
		rdb.batch = make([]Result, 0, rdb.batchSz)
		rdb.mu.Unlock()
		rdb.flushCh <- batch // send to async flusher
		return
	}
	rdb.mu.Unlock()
}

// Flush sends all pending results to the async flusher.
func (rdb *ResultDB) Flush(_ context.Context) error {
	rdb.mu.Lock()
	if len(rdb.batch) == 0 {
		rdb.mu.Unlock()
		return nil
	}
	batch := rdb.batch
	rdb.batch = make([]Result, 0, rdb.batchSz)
	rdb.mu.Unlock()

	rdb.flushCh <- batch
	return nil
}

func (rdb *ResultDB) writeBatch(ctx context.Context, batch []Result) error {
	tx, err := rdb.resultDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin result tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO results
			(url, status_code, content_type, content_length, title, description,
			 language, domain, redirect_url, fetch_time_ms, crawled_at, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare result stmt: %w", err)
	}
	defer stmt.Close()

	for _, r := range batch {
		_, err := stmt.ExecContext(ctx,
			r.URL, r.StatusCode, r.ContentType, r.ContentLength,
			r.Title, r.Description, r.Language, r.Domain,
			r.RedirectURL, r.FetchTimeMs, r.CrawledAt, r.Error)
		if err != nil {
			return fmt.Errorf("insert result: %w", err)
		}
	}

	return tx.Commit()
}

func (rdb *ResultDB) writeState(ctx context.Context, batch []Result) error {
	tx, err := rdb.stateDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin state tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO state (url, status, status_code, error, fetched_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare state stmt: %w", err)
	}
	defer stmt.Close()

	for _, r := range batch {
		status := "done"
		if r.Error != "" {
			status = "failed"
		}
		_, err := stmt.ExecContext(ctx, r.URL, status, r.StatusCode, r.Error, r.CrawledAt)
		if err != nil {
			return fmt.Errorf("insert state: %w", err)
		}
	}

	return tx.Commit()
}

// SetMeta stores a key-value pair in the state meta table.
func (rdb *ResultDB) SetMeta(ctx context.Context, key, value string) error {
	_, err := rdb.stateDB.ExecContext(ctx,
		"INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)", key, value)
	return err
}

// FlushedCount returns the number of results written to disk.
func (rdb *ResultDB) FlushedCount() int64 {
	return rdb.flushed.Load()
}

// PendingCount returns the number of results not yet flushed.
func (rdb *ResultDB) PendingCount() int {
	rdb.mu.Lock()
	defer rdb.mu.Unlock()
	return len(rdb.batch)
}

// Close flushes remaining results, waits for async flusher to finish, and closes databases.
func (rdb *ResultDB) Close() error {
	rdb.Flush(context.Background())
	close(rdb.flushCh) // signal flusher to stop
	<-rdb.done         // wait for all writes to complete
	rdb.resultDB.Close()
	rdb.stateDB.Close()
	return nil
}
