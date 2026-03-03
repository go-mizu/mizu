package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("sqlite", func() index.Engine { return &Engine{} })
}

type Engine struct {
	db     *sql.DB
	dbPath string
	dir    string
}

func (e *Engine) Name() string { return "sqlite" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	e.dbPath = filepath.Join(dir, "fts.db")
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", e.dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("sqlite open %s: %w", e.dbPath, err)
	}
	e.db = db

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS documents (
			doc_id TEXT PRIMARY KEY,
			text   TEXT
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
			text,
			content='documents',
			content_rowid='rowid',
			tokenize='unicode61 remove_diacritics 0'
		)`,
		`CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
			INSERT INTO documents_fts(rowid, text) VALUES (new.rowid, new.text);
		END`,
		`CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
			INSERT INTO documents_fts(documents_fts, rowid, text) VALUES ('delete', old.rowid, old.text);
		END`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("sqlite init: %w", err)
		}
	}
	return nil
}

func (e *Engine) Close() error {
	if e.db == nil {
		return nil
	}
	// Optimize FTS before closing
	e.db.Exec("INSERT INTO documents_fts(documents_fts) VALUES ('optimize')")
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
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

// luceneToFTS5 translates benchmark query shapes to SQLite FTS5 syntax:
//   "+term1 +term2"        → "term1 term2"        (AND; FTS5 default)
//   "+term1 -term2"        → "term1 NOT term2"    (AND NOT)
//   `"phrase query"`       → `"phrase query"`     (phrase; unchanged)
//   "term1 term2"          → "term1 OR term2"     (union)
func luceneToFTS5(q string) string {
	hasBoolOp := strings.Contains(q, "+") ||
		strings.HasPrefix(q, "-") ||
		strings.Contains(q, " -")
	if hasBoolOp {
		parts := strings.Fields(q)
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			switch {
			case strings.HasPrefix(p, "+"):
				out = append(out, p[1:])
			case strings.HasPrefix(p, "-"):
				out = append(out, "NOT "+p[1:])
			default:
				out = append(out, p)
			}
		}
		return strings.Join(out, " ")
	}
	if strings.HasPrefix(q, `"`) {
		return q
	}
	tokens := strings.Fields(q)
	if len(tokens) > 1 {
		return strings.Join(tokens, " OR ")
	}
	return q
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	sqlStr := `SELECT d.doc_id, snippet(documents_fts, 0, '', '', '...', 40) AS snippet,
	        -bm25(documents_fts) AS score
	        FROM documents_fts
	        JOIN documents d ON d.rowid = documents_fts.rowid
	        WHERE documents_fts.text MATCH ?
	        ORDER BY bm25(documents_fts)
	        LIMIT ? OFFSET ?`

	rows, err := e.db.QueryContext(ctx, sqlStr, luceneToFTS5(q.Text), limit, q.Offset)
	if err != nil {
		return index.Results{}, fmt.Errorf("sqlite search: %w", err)
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

var _ index.Engine = (*Engine)(nil)
