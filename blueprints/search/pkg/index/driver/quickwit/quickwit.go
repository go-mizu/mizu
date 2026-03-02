package quickwit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const (
	defaultAddr = "http://localhost:7280"
	indexID     = "fts_docs"
)

func init() {
	index.Register("quickwit", func() index.Engine { return NewEngine() })
}

type Engine struct {
	index.BaseExternal
	client *http.Client
	base   string
	dir    string
}

func NewEngine() *Engine {
	return &Engine{client: &http.Client{Timeout: 120 * time.Second}}
}

func (e *Engine) Name() string { return "quickwit" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	e.base = e.EffectiveAddr(defaultAddr)

	// Create index — 200 = created, 400 = already exists, both OK
	schema := map[string]any{
		"index_id": indexID,
		"doc_mapping": map[string]any{
			"field_mappings": []any{
				map[string]any{"name": "doc_id", "type": "text", "tokenizer": "raw", "stored": true},
				map[string]any{
					"name": "text", "type": "text", "tokenizer": "default",
					"stored": true, "record": "position", "fieldnorms": true,
				},
			},
		},
		"search_settings": map[string]any{
			"default_search_fields": []string{"text"},
		},
	}
	body, _ := json.Marshal(schema)
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/api/v1/indexes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("quickwit create index: %w", err)
	}
	defer resp.Body.Close()
	// 200 (created) or 400 (already exists) are both acceptable
	if resp.StatusCode >= 500 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("quickwit create index HTTP %d: %s", resp.StatusCode, b)
	}
	return nil
}

func (e *Engine) Close() error { return nil }

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	if e.client == nil {
		return index.EngineStats{}, nil
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", e.base+"/api/v1/indexes/"+indexID, nil)
	resp, err := e.client.Do(req)
	if err != nil {
		return index.EngineStats{}, fmt.Errorf("quickwit stats: %w", err)
	}
	defer resp.Body.Close()
	var info struct {
		IndexConfig struct{} `json:"index_config"`
		Splits      []struct {
			NumDocs int64 `json:"num_docs"`
		} `json:"splits"`
	}
	json.NewDecoder(resp.Body).Decode(&info) //nolint:errcheck
	var total int64
	for _, s := range info.Splits {
		total += s.NumDocs
	}
	return index.EngineStats{DocCount: total}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	if e.client == nil {
		return fmt.Errorf("quickwit: engine not opened")
	}
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	for _, d := range docs {
		if err := enc.Encode(map[string]string{"doc_id": d.DocID, "text": string(d.Text)}); err != nil {
			return fmt.Errorf("quickwit encode doc: %w", err)
		}
	}
	url := fmt.Sprintf("%s/api/v1/%s/ingest?commit=force", e.base, indexID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(sb.String()))
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("quickwit ingest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("quickwit ingest HTTP %d: %s", resp.StatusCode, b)
	}
	return nil
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	if e.client == nil {
		return index.Results{}, fmt.Errorf("quickwit: engine not opened")
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	payload := map[string]any{
		"query":          q.Text,
		"max_hits":       limit,
		"start_offset":   q.Offset,
		"snippet_fields": []string{"text"},
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/%s/search", e.base, indexID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return index.Results{}, fmt.Errorf("quickwit search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return index.Results{}, fmt.Errorf("quickwit search HTTP %d: %s", resp.StatusCode, b)
	}
	var sr struct {
		Hits    []map[string]any `json:"hits"`
		NumHits int              `json:"num_hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return index.Results{}, fmt.Errorf("quickwit decode: %w", err)
	}
	results := index.Results{Total: sr.NumHits}
	for _, hit := range sr.Hits {
		h := index.Hit{Score: 1.0}
		if v, ok := hit["doc_id"].(string); ok {
			h.DocID = v
		}
		// Try to extract snippet from _snippets.text[0]
		if snips, ok := hit["_snippets"].(map[string]any); ok {
			if texts, ok := snips["text"].([]any); ok && len(texts) > 0 {
				h.Snippet, _ = texts[0].(string)
			}
		}
		results.Hits = append(results.Hits, h)
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
