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
			"fields": map[string]any{
				"doc_id": map[string]any{"type": "text", "stored": true},
				"text":   map[string]any{"type": "text", "stored": false},
			},
		},
	}
	body, _ := json.Marshal(schema)
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/api/v1/indexes", bytes.NewReader(body))
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
	if e.client == nil {
		return index.EngineStats{}, nil
	}
	url := fmt.Sprintf("%s/api/v1/indexes/%s/summary", e.base, lnxIndex)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := e.client.Do(req)
	if err != nil {
		return index.EngineStats{}, fmt.Errorf("lnx stats: %w", err)
	}
	defer resp.Body.Close()
	var info struct {
		NumDocs int64 `json:"num_docs"`
	}
	json.NewDecoder(resp.Body).Decode(&info) //nolint:errcheck
	return index.EngineStats{DocCount: info.NumDocs}, nil
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
	url := fmt.Sprintf("%s/api/v1/indexes/%s/documents", e.base, lnxIndex)
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
	url := fmt.Sprintf("%s/api/v1/indexes/%s/commit", e.base, lnxIndex)
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
	payload := map[string]any{
		"query":  q.Text,
		"limit":  limit,
		"offset": q.Offset,
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/v1/indexes/%s/search", e.base, lnxIndex)
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

	// lnx response shape: {"data": [...], "count": N} or {"hits": [...], "total": N}
	// Try to handle both shapes
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return index.Results{}, fmt.Errorf("lnx decode: %w", err)
	}

	var results index.Results

	// Try "data" key first (lnx v0.x)
	if dataRaw, ok := raw["data"]; ok {
		var hits []map[string]any
		if err := json.Unmarshal(dataRaw, &hits); err == nil {
			for _, hit := range hits {
				h := index.Hit{Score: 1.0}
				// lnx stores doc fields under "doc" key or directly
				if doc, ok := hit["doc"].(map[string]any); ok {
					h.DocID, _ = doc["doc_id"].(string)
				} else {
					h.DocID, _ = hit["doc_id"].(string)
				}
				if s, ok := hit["_score"].(float64); ok {
					h.Score = s
				}
				results.Hits = append(results.Hits, h)
			}
		}
	}

	// Try to get count
	if countRaw, ok := raw["count"]; ok {
		json.Unmarshal(countRaw, &results.Total) //nolint:errcheck
	}
	if results.Total == 0 {
		results.Total = len(results.Hits)
	}

	return results, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
