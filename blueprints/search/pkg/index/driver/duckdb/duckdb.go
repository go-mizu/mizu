//go:build !chdb

package duckdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	duckdbdrv "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	// Register 4 variants of the DuckDB engine so they can be selected via --engine:
	//   duckdb           — original: one fmt.Sprintf INSERT per doc (baseline)
	//   duckdb-prepared  — prepared statement per doc; no string building/escaping
	//   duckdb-multirow  — single INSERT with N-row VALUES per batch; 1 SQL parse per batch
	//   duckdb-appender  — DuckDB Appender API; bypasses SQL entirely
	for _, mode := range []string{"naive", "prepared", "multirow", "appender"} {
		m := mode
		name := "duckdb"
		if m != "naive" {
			name = "duckdb-" + m
		}
		index.Register(name, func() index.Engine { return &Engine{name: name, insertMode: m} })
	}
}

// Engine is a DuckDB FTS engine. insertMode controls the bulk-insert strategy.
type Engine struct {
	db         *sql.DB
	dbPath     string
	dir        string
	name       string
	insertMode string // "naive" | "prepared" | "multirow" | "appender"

	// appender mode: a dedicated connection kept alive while the Appender is open.
	appenderConn *sql.Conn
	appender     *duckdbdrv.Appender

	// cumulative time spent in Index() calls (excludes FTS build).
	insertTime time.Duration
}

func (e *Engine) Name() string { return e.name }

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

	// Tune DuckDB: use all CPUs, raise checkpoint threshold to avoid mid-import
	// WAL flushes, and grant generous temp storage.
	nCPU := runtime.NumCPU()
	tuning := fmt.Sprintf(
		"SET threads = %d; SET checkpoint_threshold = '4GB'; SET preserve_insertion_order = false;",
		nCPU,
	)
	if _, err := db.ExecContext(ctx, tuning); err != nil {
		// non-fatal: some settings may not be available on older DuckDB versions
		fmt.Fprintf(os.Stderr, "duckdb tuning warning: %v\n", err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS documents (
		doc_id VARCHAR PRIMARY KEY,
		text   VARCHAR
	)`)
	return err
}

func (e *Engine) Close() error {
	if e.appender != nil {
		e.appender.Close() //nolint:errcheck
		e.appender = nil
	}
	if e.appenderConn != nil {
		e.appenderConn.Close() //nolint:errcheck
		e.appenderConn = nil
	}
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

// Index routes to the configured insert implementation and tracks cumulative insert time.
func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	t0 := time.Now()
	var err error
	switch e.insertMode {
	case "prepared":
		err = e.indexPrepared(ctx, docs)
	case "multirow":
		err = e.indexMultirow(ctx, docs)
	case "appender":
		err = e.indexAppender(ctx, docs)
	default: // "naive"
		err = e.indexNaive(ctx, docs)
	}
	e.insertTime += time.Since(t0)
	return err
}

// ── Insert implementations ────────────────────────────────────────────────

// indexNaive: original approach — one fmt.Sprintf INSERT per doc with manual ' escaping.
func (e *Engine) indexNaive(ctx context.Context, docs []index.Document) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

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

// indexPrepared: prepare one statement per batch; execute once per doc.
// Eliminates SQL string building and manual escaping.
func (e *Engine) indexPrepared(ctx context.Context, docs []index.Document) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO documents (doc_id, text) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, doc := range docs {
		if _, err := stmt.ExecContext(ctx, doc.DocID, string(doc.Text)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// indexMultirow: one INSERT per batch with N-row VALUES clause.
// Single SQL parse and plan per batch instead of N.
func (e *Engine) indexMultirow(ctx context.Context, docs []index.Document) error {
	// Build: INSERT OR IGNORE INTO documents (doc_id, text) VALUES (?,?),(?,?), ...
	var sb strings.Builder
	sb.WriteString("INSERT OR IGNORE INTO documents (doc_id, text) VALUES ")
	args := make([]any, 0, len(docs)*2)
	for i, doc := range docs {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("(?,?)")
		args = append(args, doc.DocID, string(doc.Text))
	}
	_, err := e.db.ExecContext(ctx, sb.String(), args...)
	return err
}

// indexAppender: use the DuckDB Appender API — bypasses SQL parsing entirely.
// The Appender is created lazily on the first call and reused across batches.
func (e *Engine) indexAppender(ctx context.Context, docs []index.Document) error {
	if e.appender == nil {
		// Allocate a dedicated connection for the Appender and keep it alive.
		sqlConn, err := e.db.Conn(ctx)
		if err != nil {
			return fmt.Errorf("appender: get conn: %w", err)
		}
		e.appenderConn = sqlConn

		var appErr error
		if err := sqlConn.Raw(func(c any) error {
			dConn, ok := c.(driver.Conn)
			if !ok {
				return fmt.Errorf("expected duckdb driver.Conn, got %T", c)
			}
			e.appender, appErr = duckdbdrv.NewAppenderFromConn(dConn, "", "documents")
			return appErr
		}); err != nil {
			return fmt.Errorf("appender: create: %w", err)
		}
	}

	for _, doc := range docs {
		if err := e.appender.AppendRow(doc.DocID, string(doc.Text)); err != nil {
			return fmt.Errorf("appender: append row: %w", err)
		}
	}
	return nil
}

// ── Search ────────────────────────────────────────────────────────────────

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	rows, err := e.searchFTS(ctx, q.Text, limit, q.Offset)
	if err != nil {
		return e.searchLike(ctx, q.Text, limit, q.Offset)
	}
	return rows, nil
}

func (e *Engine) searchFTS(ctx context.Context, query string, limit, offset int) (index.Results, error) {
	if _, err := e.db.ExecContext(ctx, "INSTALL fts; LOAD fts"); err != nil {
		return index.Results{}, err
	}
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

// ── FTS index creation ────────────────────────────────────────────────────

// CreateFTSIndex builds the BM25 full-text index after all documents are inserted.
// It flushes any pending Appender data, prints INSERT vs FTS-build timing,
// and runs PRAGMA create_fts_index.
func (e *Engine) CreateFTSIndex(ctx context.Context) error {
	// Flush and close the appender before running DDL on the same table.
	if e.appender != nil {
		if err := e.appender.Close(); err != nil {
			return fmt.Errorf("flush appender: %w", err)
		}
		e.appender = nil
	}
	if e.appenderConn != nil {
		if err := e.appenderConn.Close(); err != nil {
			return fmt.Errorf("close appender conn: %w", err)
		}
		e.appenderConn = nil
	}

	fmt.Fprintf(os.Stderr, "\n  insert time: %s (mode=%s)\n",
		e.insertTime.Round(100*time.Millisecond), e.insertMode)

	if _, err := e.db.ExecContext(ctx, "INSTALL fts; LOAD fts"); err != nil {
		return fmt.Errorf("load fts: %w", err)
	}

	t0 := time.Now()
	fmt.Fprintf(os.Stderr, "  building FTS index (PRAGMA create_fts_index)...\n")
	_, err := e.db.ExecContext(ctx,
		`PRAGMA create_fts_index('documents', 'doc_id', 'text',
		 stemmer='english', stopwords='english', lower=1, strip_accents=1, overwrite=1)`)
	if err != nil {
		return fmt.Errorf("create fts index: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  fts build time: %s\n", time.Since(t0).Round(100*time.Millisecond))
	return nil
}

var _ index.Engine = (*Engine)(nil)
