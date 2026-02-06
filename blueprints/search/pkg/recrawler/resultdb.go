package recrawler

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ResultDB writes recrawl results and state to DuckDB.
type ResultDB struct {
	resultDB *sql.DB
	stateDB  *sql.DB

	mu      sync.Mutex
	batch   []Result
	batchSz int
	flushed int64
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
	}

	if err := rdb.initSchema(); err != nil {
		resultDB.Close()
		stateDB.Close()
		return nil, err
	}

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

// Add queues a result for batch writing.
func (rdb *ResultDB) Add(r Result) {
	rdb.mu.Lock()
	rdb.batch = append(rdb.batch, r)
	shouldFlush := len(rdb.batch) >= rdb.batchSz
	rdb.mu.Unlock()

	if shouldFlush {
		rdb.Flush(context.Background())
	}
}

// Flush writes all pending results to the database.
func (rdb *ResultDB) Flush(ctx context.Context) error {
	rdb.mu.Lock()
	if len(rdb.batch) == 0 {
		rdb.mu.Unlock()
		return nil
	}
	batch := rdb.batch
	rdb.batch = make([]Result, 0, rdb.batchSz)
	rdb.mu.Unlock()

	// Write results
	if err := rdb.writeBatch(ctx, batch); err != nil {
		return err
	}

	// Write state
	if err := rdb.writeState(ctx, batch); err != nil {
		return err
	}

	rdb.mu.Lock()
	rdb.flushed += int64(len(batch))
	rdb.mu.Unlock()

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
	rdb.mu.Lock()
	defer rdb.mu.Unlock()
	return rdb.flushed
}

// PendingCount returns the number of results not yet flushed.
func (rdb *ResultDB) PendingCount() int {
	rdb.mu.Lock()
	defer rdb.mu.Unlock()
	return len(rdb.batch)
}

// Close flushes remaining results and closes databases.
func (rdb *ResultDB) Close() error {
	rdb.Flush(context.Background())
	rdb.resultDB.Close()
	rdb.stateDB.Close()
	return nil
}
