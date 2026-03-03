package clickhouse

import (
	"context"
	"fmt"
	"strings"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const (
	defaultAddr = "localhost:9000"
	defaultDB   = "fts"
	defaultUser = "fts"
	defaultPass = "fts"
)

func init() {
	index.Register("clickhouse", func() index.Engine { return &Engine{} })
}

// Engine is an external FTS engine backed by ClickHouse via the native TCP protocol.
type Engine struct {
	index.BaseExternal
	conn driver.Conn
}

func (e *Engine) Name() string { return "clickhouse" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	addr := e.EffectiveAddr(defaultAddr)
	conn, err := ch.Open(&ch.Options{
		Addr: []string{addr},
		Auth: ch.Auth{
			Database: defaultDB,
			Username: defaultUser,
			Password: defaultPass,
		},
		MaxOpenConns: 4,
		MaxIdleConns: 2,
	})
	if err != nil {
		return fmt.Errorf("clickhouse: open: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		return fmt.Errorf("clickhouse: ping: %w", err)
	}
	e.conn = conn
	return e.CreateTable(ctx)
}

// CreateTable creates the fts_docs table if it does not already exist.
func (e *Engine) CreateTable(ctx context.Context) error {
	if e.conn == nil {
		return fmt.Errorf("clickhouse: not connected")
	}
	// text index type is GA in ClickHouse 26.2; replaces tokenbf_v1 for FTS.
	const ddl = `
CREATE TABLE IF NOT EXISTS fts_docs (
    doc_id String,
    text   String,
    INDEX  text_idx text TYPE text(tokenizer = 'splitByNonAlpha')
) ENGINE = MergeTree()
ORDER BY doc_id
SETTINGS index_granularity = 8192`
	if err := e.conn.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("clickhouse: create table: %w", err)
	}
	return nil
}

// DropTable drops the fts_docs table. Useful for test cleanup.
func (e *Engine) DropTable(ctx context.Context) error {
	if e.conn == nil {
		return fmt.Errorf("clickhouse: not connected")
	}
	if err := e.conn.Exec(ctx, "DROP TABLE IF EXISTS fts_docs"); err != nil {
		return fmt.Errorf("clickhouse: drop table: %w", err)
	}
	return nil
}

func (e *Engine) Close() error {
	if e.conn == nil {
		return nil
	}
	return e.conn.Close()
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if e.conn == nil {
		return fmt.Errorf("clickhouse: not connected")
	}
	if len(docs) == 0 {
		return nil
	}
	batch, err := e.conn.PrepareBatch(ctx, "INSERT INTO fts_docs (doc_id, text)")
	if err != nil {
		return fmt.Errorf("clickhouse: prepare batch: %w", err)
	}
	for _, d := range docs {
		if err := batch.Append(d.DocID, string(d.Text)); err != nil {
			return fmt.Errorf("clickhouse: batch append: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("clickhouse: batch send: %w", err)
	}
	return nil
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	if e.conn == nil {
		return index.Results{}, fmt.Errorf("clickhouse: not connected")
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	if strings.TrimSpace(q.Text) == "" {
		return index.Results{}, nil
	}
	// hasAllTokens supports multi-word queries and leverages the text index (ClickHouse 26.2+).
	// lower() on both sides gives case-insensitive matching.
	args := []any{strings.ToLower(q.Text), limit, q.Offset}
	const query = `
SELECT doc_id, substring(text, 1, 200) AS snippet, 1.0 AS score
FROM fts_docs
WHERE hasAllTokens(lower(text), ?)
ORDER BY doc_id
LIMIT ? OFFSET ?`

	rows, err := e.conn.Query(ctx, query, args...)
	if err != nil {
		return index.Results{}, fmt.Errorf("clickhouse: search query: %w", err)
	}
	defer rows.Close()

	var hits []index.Hit
	for rows.Next() {
		var docID, snippet string
		var score float64
		if err := rows.Scan(&docID, &snippet, &score); err != nil {
			return index.Results{}, fmt.Errorf("clickhouse: scan row: %w", err)
		}
		hits = append(hits, index.Hit{
			DocID:   docID,
			Score:   score,
			Snippet: snippet,
		})
	}
	if err := rows.Err(); err != nil {
		return index.Results{}, fmt.Errorf("clickhouse: rows error: %w", err)
	}

	return index.Results{
		Hits:  hits,
		Total: len(hits),
	}, nil
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	if e.conn == nil {
		return index.EngineStats{}, fmt.Errorf("clickhouse: not connected")
	}

	var docCount uint64
	row := e.conn.QueryRow(ctx, "SELECT count() FROM fts_docs")
	if err := row.Scan(&docCount); err != nil {
		return index.EngineStats{}, fmt.Errorf("clickhouse: stats count: %w", err)
	}

	var diskBytes uint64
	row = e.conn.QueryRow(ctx,
		"SELECT sum(data_compressed_bytes) FROM system.parts WHERE table='fts_docs' AND active")
	if err := row.Scan(&diskBytes); err != nil {
		return index.EngineStats{}, fmt.Errorf("clickhouse: stats disk: %w", err)
	}

	return index.EngineStats{
		DocCount:  int64(docCount),
		DiskBytes: int64(diskBytes),
	}, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
