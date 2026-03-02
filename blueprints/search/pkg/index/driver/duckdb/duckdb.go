//go:build !chdb

package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("duckdb", func() index.Engine { return &Engine{} })
}

type Engine struct {
	db     *sql.DB
	dbPath string
	dir    string
}

func (e *Engine) Name() string { return "duckdb" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	e.dbPath = filepath.Join(dir, "fts.duckdb")

	db, err := sql.Open("duckdb", e.dbPath)
	if err != nil {
		return fmt.Errorf("duckdb open %s: %w", e.dbPath, err)
	}
	e.db = db

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS documents (
		doc_id VARCHAR PRIMARY KEY,
		text   VARCHAR
	)`)
	return err
}

func (e *Engine) Close() error {
	if e.db == nil {
		return nil
	}
	return e.db.Close()
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	var count int64
	if err := e.db.QueryRowContext(ctx, "SELECT count(*) FROM documents").Scan(&count); err != nil {
		return index.EngineStats{}, err
	}
	disk := index.DirSizeBytes(e.dir)
	return index.EngineStats{DocCount: count, DiskBytes: disk}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, doc := range docs {
		id := strings.ReplaceAll(doc.DocID, "'", "''")
		text := strings.ReplaceAll(string(doc.Text), "'", "''")
		sqlStr := fmt.Sprintf("INSERT OR IGNORE INTO documents (doc_id, text) VALUES ('%s', '%s')", id, text)
		if _, err := tx.ExecContext(ctx, sqlStr); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	// Try FTS search first
	rows, err := e.searchFTS(ctx, q.Text, limit, q.Offset)
	if err != nil {
		// Fallback to LIKE
		return e.searchLike(ctx, q.Text, limit, q.Offset)
	}
	return rows, nil
}

func (e *Engine) searchFTS(ctx context.Context, query string, limit, offset int) (index.Results, error) {
	// Load FTS extension
	if _, err := e.db.ExecContext(ctx, "INSTALL fts; LOAD fts"); err != nil {
		return index.Results{}, err
	}

	// Check if FTS index exists
	var n int
	err := e.db.QueryRowContext(ctx,
		"SELECT count(*) FROM information_schema.schemata WHERE schema_name = 'fts_main_documents'").Scan(&n)
	if err != nil || n == 0 {
		return index.Results{}, fmt.Errorf("no FTS index")
	}

	sqlStr := `SELECT doc_id, substring(text, 1, 200) AS snippet,
	        fts_main_documents.match_bm25(doc_id, ?, fields := 'text') AS score
	        FROM documents WHERE score IS NOT NULL
	        ORDER BY score DESC LIMIT ? OFFSET ?`

	rows, err := e.db.QueryContext(ctx, sqlStr, query, limit, offset)
	if err != nil {
		return index.Results{}, err
	}
	defer rows.Close()

	var results index.Results
	for rows.Next() {
		var h index.Hit
		if err := rows.Scan(&h.DocID, &h.Snippet, &h.Score); err != nil {
			return results, err
		}
		results.Hits = append(results.Hits, h)
	}
	results.Total = len(results.Hits)
	return results, rows.Err()
}

func (e *Engine) searchLike(ctx context.Context, query string, limit, offset int) (index.Results, error) {
	sqlStr := `SELECT doc_id, substring(text, 1, 200) AS snippet, 1.0 AS score
	        FROM documents WHERE text ILIKE '%' || ? || '%'
	        LIMIT ? OFFSET ?`

	rows, err := e.db.QueryContext(ctx, sqlStr, query, limit, offset)
	if err != nil {
		return index.Results{}, err
	}
	defer rows.Close()

	var results index.Results
	for rows.Next() {
		var h index.Hit
		if err := rows.Scan(&h.DocID, &h.Snippet, &h.Score); err != nil {
			return results, err
		}
		results.Hits = append(results.Hits, h)
	}
	results.Total = len(results.Hits)
	return results, rows.Err()
}

// CreateFTSIndex builds the DuckDB FTS index. Call after all documents are indexed.
func (e *Engine) CreateFTSIndex(ctx context.Context) error {
	if _, err := e.db.ExecContext(ctx, "INSTALL fts; LOAD fts"); err != nil {
		return fmt.Errorf("load fts: %w", err)
	}
	_, err := e.db.ExecContext(ctx,
		`PRAGMA create_fts_index('documents', 'doc_id', 'text',
		 stemmer='english', stopwords='english', lower=1, strip_accents=1, overwrite=1)`)
	if err != nil {
		return fmt.Errorf("create fts index: %w", err)
	}
	return nil
}

var _ index.Engine = (*Engine)(nil)
