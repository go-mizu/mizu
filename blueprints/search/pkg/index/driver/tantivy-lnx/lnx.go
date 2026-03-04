package lnx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const (
	defaultAddr = "http://localhost:8000"
	lnxIndex    = "fts_docs"
)

func init() {
	index.Register("tantivy-lnx", func() index.Engine { return NewEngine() })
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

func (e *Engine) Name() string { return "tantivy-lnx" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	e.base = e.EffectiveAddr(defaultAddr)

	// lnx v0.9 index creation schema
	schema := map[string]any{
		"override_if_exists": false,
		"index": map[string]any{
			"name":                   lnxIndex,
			"writer_threads":         4,
			"writer_heap_size_bytes": 67108864,
			"reader_threads":         4,
			"max_concurrency":        10,
			"search_fields":          []string{"text"},
			"store_records":          true,
			"storage_type":           "filesystem",
			"fields": map[string]any{
				"doc_id": map[string]any{"type": "text", "stored": true},
				"text":   map[string]any{"type": "text", "stored": false},
			},
		},
	}
	body, _ := json.Marshal(schema)
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/indexes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("lnx create index: %w", err)
	}
	defer resp.Body.Close()
	// 200 = created, 400/409 = already exists — both acceptable
	if resp.StatusCode >= 500 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lnx create index HTTP %d: %s", resp.StatusCode, b)
	}
	return nil
}

func (e *Engine) Close() error { return nil }

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	return index.EngineStats{}, nil // lnx v0.9 has no doc-count summary endpoint
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	if e.client == nil {
		return fmt.Errorf("lnx: engine not opened")
	}
	payload := make([]map[string]string, len(docs))
	for i, d := range docs {
		payload[i] = map[string]string{"doc_id": d.DocID, "text": string(d.Text)}
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/indexes/%s/documents", e.base, lnxIndex)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("lnx add docs: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lnx add docs HTTP %d: %s", resp.StatusCode, b)
	}
	return e.commit(ctx)
}

func (e *Engine) commit(ctx context.Context) error {
	url := fmt.Sprintf("%s/indexes/%s/commit", e.base, lnxIndex)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("lnx commit: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lnx commit HTTP %d: %s", resp.StatusCode, b)
	}
	return nil
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	if e.client == nil {
		return index.Results{}, fmt.Errorf("lnx: engine not opened")
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	// lnx v0.9 simple string query: {"query": "...", "limit": N, "offset": N}
	payload := map[string]any{
		"query":  q.Text,
		"limit":  limit,
		"offset": q.Offset,
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/indexes/%s/search", e.base, lnxIndex)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return index.Results{}, fmt.Errorf("lnx search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return index.Results{}, fmt.Errorf("lnx search HTTP %d: %s", resp.StatusCode, b)
	}

	// lnx v0.9 response: {"status":200,"data":{"hits":[{"doc":{"doc_id":"..."},"score":N}],"count":N}}
	var outer struct {
		Data struct {
			Hits []struct {
				Doc   map[string]string `json:"doc"`
				Score float64           `json:"score"`
			} `json:"hits"`
			Count int `json:"count"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&outer); err != nil {
		return index.Results{}, fmt.Errorf("lnx decode: %w", err)
	}

	results := index.Results{Total: outer.Data.Count}
	for _, hit := range outer.Data.Hits {
		results.Hits = append(results.Hits, index.Hit{
			DocID: hit.Doc["doc_id"],
			Score: hit.Score,
		})
	}
	if results.Total == 0 {
		results.Total = len(results.Hits)
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
