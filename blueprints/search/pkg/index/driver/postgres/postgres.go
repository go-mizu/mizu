package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const defaultAddr = "postgres://fineweb:fineweb@localhost:5432/fts"

func init() {
	index.Register("postgres", func() index.Engine { return &Engine{} })
}

// Engine is an external FTS engine backed by PostgreSQL tsvector/GIN full-text search.
type Engine struct {
	index.BaseExternal
	pool *pgxpool.Pool
}

func (e *Engine) Name() string { return "postgres" }

// Open creates a connection pool, pings the server, and ensures the schema exists.
func (e *Engine) Open(ctx context.Context, dir string) error {
	addr := e.EffectiveAddr(defaultAddr)
	pool, err := pgxpool.New(ctx, addr)
	if err != nil {
		return fmt.Errorf("postgres: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("postgres: ping: %w", err)
	}
	e.pool = pool
	return e.CreateTable(ctx)
}

// CreateTable creates the fts_docs table and GIN index if they do not already exist.
func (e *Engine) CreateTable(ctx context.Context) error {
	if e.pool == nil {
		return fmt.Errorf("postgres: not connected")
	}
	const ddl = `
CREATE TABLE IF NOT EXISTS fts_docs (
    doc_id TEXT PRIMARY KEY,
    text   TEXT,
    tsv    TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', text)) STORED
);
CREATE INDEX IF NOT EXISTS fts_docs_tsv_idx ON fts_docs USING GIN(tsv);`
	if _, err := e.pool.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("postgres: create table: %w", err)
	}
	return nil
}

// DropTable drops the fts_docs table. Useful for test cleanup.
func (e *Engine) DropTable(ctx context.Context) error {
	if e.pool == nil {
		return fmt.Errorf("postgres: not connected")
	}
	if _, err := e.pool.Exec(ctx, "DROP TABLE IF EXISTS fts_docs"); err != nil {
		return fmt.Errorf("postgres: drop table: %w", err)
	}
	return nil
}

// Close shuts down the connection pool.
func (e *Engine) Close() error {
	if e.pool == nil {
		return nil
	}
	e.pool.Close()
	e.pool = nil
	return nil
}

// Index bulk-inserts documents using pgx CopyFrom (fastest batch insert).
// Duplicate doc_ids are handled via ON CONFLICT DO UPDATE.
func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if e.pool == nil {
		return fmt.Errorf("postgres: not connected")
	}
	if len(docs) == 0 {
		return nil
	}

	rows := make([][]any, len(docs))
	for i, d := range docs {
		rows[i] = []any{d.DocID, string(d.Text)}
	}

	// CopyFrom is the fastest bulk-insert path in pgx.
	// We use a temporary table + INSERT ... ON CONFLICT to support upserts,
	// because COPY itself does not handle conflicts.
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("postgres: acquire conn: %w", err)
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, `CREATE TEMP TABLE IF NOT EXISTS fts_docs_tmp (
		doc_id TEXT,
		text   TEXT
	)`)
	if err != nil {
		return fmt.Errorf("postgres: create temp table: %w", err)
	}

	_, err = conn.Exec(ctx, "TRUNCATE fts_docs_tmp")
	if err != nil {
		return fmt.Errorf("postgres: truncate temp table: %w", err)
	}

	_, err = conn.CopyFrom(
		ctx,
		pgx.Identifier{"fts_docs_tmp"},
		[]string{"doc_id", "text"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("postgres: copy from: %w", err)
	}

	_, err = conn.Exec(ctx, `
INSERT INTO fts_docs (doc_id, text)
SELECT doc_id, text FROM fts_docs_tmp
ON CONFLICT (doc_id) DO UPDATE SET text = EXCLUDED.text`)
	if err != nil {
		return fmt.Errorf("postgres: upsert from temp: %w", err)
	}

	return nil
}

// Search executes a full-text query using websearch_to_tsquery and ts_rank_cd.
func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	if e.pool == nil {
		return index.Results{}, fmt.Errorf("postgres: not connected")
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	const query = `
SELECT doc_id,
       ts_headline('english', text, websearch_to_tsquery('english', $1), 'MaxFragments=1,MaxWords=20') AS snippet,
       ts_rank_cd(tsv, websearch_to_tsquery('english', $1)) AS score
FROM fts_docs
WHERE tsv @@ websearch_to_tsquery('english', $1)
ORDER BY score DESC
LIMIT $2 OFFSET $3`

	rows, err := e.pool.Query(ctx, query, q.Text, limit, q.Offset)
	if err != nil {
		return index.Results{}, fmt.Errorf("postgres: search query: %w", err)
	}
	defer rows.Close()

	var hits []index.Hit
	for rows.Next() {
		var docID, snippet string
		var score float64
		if err := rows.Scan(&docID, &snippet, &score); err != nil {
			return index.Results{}, fmt.Errorf("postgres: scan row: %w", err)
		}
		hits = append(hits, index.Hit{
			DocID:   docID,
			Score:   score,
			Snippet: snippet,
		})
	}
	if err := rows.Err(); err != nil {
		return index.Results{}, fmt.Errorf("postgres: rows error: %w", err)
	}

	return index.Results{
		Hits:  hits,
		Total: len(hits),
	}, nil
}

// Stats returns the document count and total relation size for fts_docs.
func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	if e.pool == nil {
		return index.EngineStats{}, fmt.Errorf("postgres: not connected")
	}

	var docCount int64
	if err := e.pool.QueryRow(ctx, "SELECT count(*) FROM fts_docs").Scan(&docCount); err != nil {
		return index.EngineStats{}, fmt.Errorf("postgres: stats count: %w", err)
	}

	var diskBytes int64
	if err := e.pool.QueryRow(ctx, "SELECT pg_total_relation_size('fts_docs')").Scan(&diskBytes); err != nil {
		return index.EngineStats{}, fmt.Errorf("postgres: stats disk: %w", err)
	}

	return index.EngineStats{
		DocCount:  docCount,
		DiskBytes: diskBytes,
	}, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
