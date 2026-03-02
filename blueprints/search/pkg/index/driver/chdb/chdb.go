//go:build chdb

package chdb

import (
	"context"
	"fmt"
	"os"
	"strings"

	chdb_api "github.com/chdb-io/chdb-go/chdb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("chdb", func() index.Engine { return &Engine{} })
}

type Engine struct {
	session *chdb_api.Session
	dir     string
}

func (e *Engine) Name() string { return "chdb" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir

	session, err := chdb_api.NewSession(dir)
	if err != nil {
		return fmt.Errorf("chdb open %s: %w", dir, err)
	}
	e.session = session

	_, err = session.Query(`CREATE TABLE IF NOT EXISTS documents (
		doc_id String,
		text   String,
		INDEX text_idx text TYPE inverted()
	) ENGINE = MergeTree() ORDER BY doc_id
	SETTINGS index_granularity = 8192`)
	if err != nil {
		return fmt.Errorf("chdb create table: %w", err)
	}
	return nil
}

func (e *Engine) Close() error {
	if e.session != nil {
		e.session.Cleanup()
	}
	return nil
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	result, err := e.session.Query("SELECT count(*) FROM documents")
	if err != nil {
		return index.EngineStats{}, err
	}
	var count int64
	fmt.Sscan(strings.TrimSpace(result.String()), &count)
	disk := index.DirSizeBytes(e.dir)
	return index.EngineStats{DocCount: count, DiskBytes: disk}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO documents (doc_id, text) VALUES ")
	for i, doc := range docs {
		if i > 0 {
			sb.WriteString(", ")
		}
		// Escape single quotes
		id := strings.ReplaceAll(doc.DocID, "'", "''")
		text := strings.ReplaceAll(string(doc.Text), "'", "''")
		// Escape backslashes for ClickHouse
		text = strings.ReplaceAll(text, `\`, `\\`)
		fmt.Fprintf(&sb, "('%s', '%s')", id, text)
	}
	_, err := e.session.Query(sb.String())
	return err
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	// Escape query for safety
	query := strings.ReplaceAll(q.Text, "'", "''")
	query = strings.ReplaceAll(query, `\`, `\\`)

	sqlStr := fmt.Sprintf(`SELECT doc_id, substring(text, 1, 200) AS snippet
		FROM documents
		WHERE hasAllTokens(lower(text), lower('%s'))
		ORDER BY length(text) ASC
		LIMIT %d OFFSET %d FORMAT TabSeparated`, query, limit, q.Offset)

	result, err := e.session.Query(sqlStr)
	if err != nil {
		return index.Results{}, fmt.Errorf("chdb search: %w", err)
	}

	var results index.Results
	lines := strings.Split(strings.TrimSpace(result.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		results.Hits = append(results.Hits, index.Hit{
			DocID:   parts[0],
			Score:   1.0,
			Snippet: parts[1],
		})
	}
	results.Total = len(results.Hits)
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
