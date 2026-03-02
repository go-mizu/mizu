package markdown

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2" // registers "duckdb" driver
)

// IndexDB tracks conversion results in DuckDB.
// Records are written by a background drainer goroutine — Add is non-blocking.
type IndexDB struct {
	db   *sql.DB
	ch   chan IndexRecord
	done chan struct{}
}

// IndexRecord is one row in the files table.
type IndexRecord struct {
	CID              string
	HTMLSize         int
	MarkdownSize     int
	HTMLTokens       int
	MarkdownTokens   int
	CompressionRatio float64
	Title            string
	Language         string
	HasContent       bool
	ConvertMs        int
	Error            string
}

const createTableSQL = `
CREATE TABLE IF NOT EXISTS files (
    cid VARCHAR PRIMARY KEY,
    html_size INTEGER,
    markdown_size INTEGER,
    html_tokens INTEGER,
    markdown_tokens INTEGER,
    compression_ratio FLOAT,
    title VARCHAR,
    language VARCHAR,
    has_content BOOLEAN,
    convert_ms INTEGER,
    created_at TIMESTAMP DEFAULT current_timestamp,
    error VARCHAR
);
`

// OpenIndex opens or creates the index DuckDB at the given path and starts the
// background drainer goroutine.
func OpenIndex(path string, batchSize int) (*IndexDB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("markdown index: mkdir: %w", err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("markdown index: open %s: %w", path, err)
	}
	for _, q := range []string{
		"SET threads=1",
		"SET preserve_insertion_order=false",
	} {
		if _, err := db.Exec(q); err != nil {
			db.Close()
			return nil, fmt.Errorf("markdown index: %s: %w", q, err)
		}
	}
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("markdown index: create table: %w", err)
	}
	if batchSize <= 0 {
		batchSize = 1000
	}
	idx := &IndexDB{
		db:   db,
		ch:   make(chan IndexRecord, 10_000),
		done: make(chan struct{}),
	}
	go idx.drain(batchSize)
	return idx, nil
}

// Add enqueues a record. The background drainer writes it to DuckDB in batches.
// Blocks only if the internal buffer (10 000 records) is full.
func (idx *IndexDB) Add(rec IndexRecord) {
	idx.ch <- rec
}

// Close closes the input channel, waits for the drainer to flush all pending
// records to DuckDB, then closes the database.
func (idx *IndexDB) Close() error {
	close(idx.ch)
	<-idx.done
	return idx.db.Close()
}

// Count returns the total number of records in the index.
func (idx *IndexDB) Count() (int, error) {
	var count int
	err := idx.db.QueryRow("SELECT COUNT(*) FROM files").Scan(&count)
	return count, err
}

// ErrorCategory is a grouped error type with its count.
type ErrorCategory struct {
	Category string
	Count    int
}

// QueryErrors opens the index at path and returns grouped error categories sorted by count desc.
func QueryErrors(path string) ([]ErrorCategory, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("query errors: open %s: %w", path, err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT
			CASE
				WHEN error = 'no content extracted' THEN 'no content extracted'
				WHEN error LIKE 'read:%'            THEN 'read error'
				WHEN error LIKE 'write:%'           THEN 'write error'
				WHEN error LIKE 'html render:%'     THEN 'html render error'
				WHEN error LIKE 'md convert:%'      THEN 'markdown convert error'
				WHEN error IS NOT NULL AND error != '' THEN 'other: ' || error
			END AS category,
			COUNT(*) AS cnt
		FROM files
		WHERE error IS NOT NULL AND error != ''
		GROUP BY category
		ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query errors: %w", err)
	}
	defer rows.Close()

	var cats []ErrorCategory
	for rows.Next() {
		var c ErrorCategory
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// drain reads from ch and writes to DuckDB in batches. Runs as a background
// goroutine; signals done when ch is closed and all pending records are flushed.
func (idx *IndexDB) drain(batchSize int) {
	defer close(idx.done)

	batch := make([]IndexRecord, 0, batchSize)
	for rec := range idx.ch {
		batch = append(batch, rec)
		if len(batch) >= batchSize {
			idx.writeBatch(batch)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		idx.writeBatch(batch)
	}
}

func (idx *IndexDB) writeBatch(batch []IndexRecord) {
	if len(batch) == 0 {
		return
	}
	var sb strings.Builder
	sb.WriteString("INSERT OR REPLACE INTO files (cid, html_size, markdown_size, html_tokens, markdown_tokens, compression_ratio, title, language, has_content, convert_ms, error) VALUES ")

	args := make([]any, 0, len(batch)*11)
	for i, rec := range batch {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("(?,?,?,?,?,?,?,?,?,?,?)")
		args = append(args,
			rec.CID,
			rec.HTMLSize,
			rec.MarkdownSize,
			rec.HTMLTokens,
			rec.MarkdownTokens,
			rec.CompressionRatio,
			rec.Title,
			rec.Language,
			rec.HasContent,
			rec.ConvertMs,
			rec.Error,
		)
	}

	// Errors from individual batch writes are best-effort; the index is advisory.
	idx.db.Exec(sb.String(), args...)
}
