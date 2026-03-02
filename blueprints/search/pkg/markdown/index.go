package markdown

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/duckdb/duckdb-go/v2" // registers "duckdb" driver
)

// IndexDB tracks conversion results in DuckDB.
type IndexDB struct {
	db        *sql.DB
	mu        sync.Mutex
	batch     []IndexRecord
	batchSize int
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

// OpenIndex opens or creates the index DuckDB at the given path.
func OpenIndex(path string, batchSize int) (*IndexDB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("markdown index: mkdir: %w", err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("markdown index: open %s: %w", path, err)
	}
	// Tune for bulk insert
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
	return &IndexDB{db: db, batchSize: batchSize}, nil
}

// Add enqueues a record for batch insert. Call Flush to write remaining.
func (idx *IndexDB) Add(rec IndexRecord) error {
	idx.mu.Lock()
	idx.batch = append(idx.batch, rec)
	if len(idx.batch) >= idx.batchSize {
		batch := idx.batch
		idx.batch = nil
		idx.mu.Unlock()
		return idx.writeBatch(batch)
	}
	idx.mu.Unlock()
	return nil
}

// Flush writes any remaining buffered records.
func (idx *IndexDB) Flush() error {
	idx.mu.Lock()
	batch := idx.batch
	idx.batch = nil
	idx.mu.Unlock()
	if len(batch) == 0 {
		return nil
	}
	return idx.writeBatch(batch)
}

// Close flushes and closes the database.
func (idx *IndexDB) Close() error {
	if err := idx.Flush(); err != nil {
		idx.db.Close()
		return err
	}
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

func (idx *IndexDB) writeBatch(batch []IndexRecord) error {
	if len(batch) == 0 {
		return nil
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

	_, err := idx.db.Exec(sb.String(), args...)
	if err != nil {
		return fmt.Errorf("markdown index: write batch (%d rows): %w", len(batch), err)
	}
	return nil
}
