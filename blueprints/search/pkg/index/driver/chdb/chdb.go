//go:build chdb

package chdb

import (
	"bytes"
	"context"
	"encoding/json"
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

	if _, err = session.Query(`SET allow_experimental_full_text_index = 1`); err != nil {
		return fmt.Errorf("chdb set fts: %w", err)
	}
	_, err = session.Query(`CREATE TABLE IF NOT EXISTS documents (
		doc_id String,
		text   String,
		INDEX text_idx text TYPE text(tokenizer='default') GRANULARITY 1
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
	// INSERT … FORMAT JSONEachRow embeds NDJSON after the SQL prefix.
	// json.Encoder escapes all special characters (newlines → \n, etc.) so no
	// multi-line field ambiguity.  No manual escaping required.
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO documents FORMAT JSONEachRow\n")
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	type row struct {
		DocID string `json:"doc_id"`
		Text  string `json:"text"`
	}
	for _, doc := range docs {
		if err := enc.Encode(row{DocID: doc.DocID, Text: string(doc.Text)}); err != nil {
			return err
		}
	}
	_, err := e.session.Query(buf.String())
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
		WHERE hasToken(lower(text), lower('%s'))
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
