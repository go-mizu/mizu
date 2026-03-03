package elasticsearch

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
	defaultAddr = "http://localhost:9201"
	indexName   = "fts_docs"
)

func init() {
	index.Register("elasticsearch", func() index.Engine { return NewEngine() })
}

// Engine is an external FTS driver backed by Elasticsearch via the REST API.
type Engine struct {
	index.BaseExternal
	client *http.Client
	base   string
}

func NewEngine() *Engine {
	return &Engine{client: &http.Client{Timeout: 120 * time.Second}}
}

func (e *Engine) Name() string { return "elasticsearch" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.base = e.EffectiveAddr(defaultAddr)

	mapping := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"doc_id": map[string]any{"type": "keyword"},
				"text":   map[string]any{"type": "text", "analyzer": "english"},
			},
		},
	}
	body, _ := json.Marshal(mapping)
	req, _ := http.NewRequestWithContext(ctx, "PUT", e.base+"/"+indexName, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("elasticsearch create index: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch create index HTTP %d: %s", resp.StatusCode, b)
	}
	return nil
}

func (e *Engine) Close() error { return nil }

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	var stats index.EngineStats

	req, _ := http.NewRequestWithContext(ctx, "GET", e.base+"/"+indexName+"/_count", nil)
	resp, err := e.client.Do(req)
	if err != nil {
		return stats, fmt.Errorf("elasticsearch count: %w", err)
	}
	defer resp.Body.Close()
	var countResp struct {
		Count int64 `json:"count"`
	}
	json.NewDecoder(resp.Body).Decode(&countResp) //nolint:errcheck
	stats.DocCount = countResp.Count

	req2, _ := http.NewRequestWithContext(ctx, "GET", e.base+"/"+indexName+"/_stats/store", nil)
	resp2, err := e.client.Do(req2)
	if err != nil {
		return stats, nil
	}
	defer resp2.Body.Close()
	var storeResp struct {
		All struct {
			Total struct {
				Store struct {
					SizeInBytes int64 `json:"size_in_bytes"`
				} `json:"store"`
			} `json:"total"`
		} `json:"_all"`
	}
	json.NewDecoder(resp2.Body).Decode(&storeResp) //nolint:errcheck
	stats.DiskBytes = storeResp.All.Total.Store.SizeInBytes

	return stats, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	for _, d := range docs {
		enc.Encode(map[string]any{"index": map[string]any{"_index": indexName, "_id": d.DocID}}) //nolint:errcheck
		enc.Encode(map[string]string{"doc_id": d.DocID, "text": string(d.Text)})                 //nolint:errcheck
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/_bulk", strings.NewReader(sb.String()))
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("elasticsearch bulk: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch bulk HTTP %d: %s", resp.StatusCode, b)
	}
	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Error *struct{ Reason string `json:"reason"` } `json:"error"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		return fmt.Errorf("elasticsearch bulk decode: %w", err)
	}
	if bulkResp.Errors {
		for _, item := range bulkResp.Items {
			for _, op := range item {
				if op.Error != nil {
					return fmt.Errorf("elasticsearch bulk item error: %s", op.Error.Reason)
				}
			}
		}
	}
	return nil
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	payload := map[string]any{
		"query":     map[string]any{"match": map[string]any{"text": q.Text}},
		"highlight": map[string]any{"fields": map[string]any{"text": map[string]any{"fragment_size": 200, "number_of_fragments": 1}}},
		"size":      limit,
		"from":      q.Offset,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", e.base+"/"+indexName+"/_search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return index.Results{}, fmt.Errorf("elasticsearch search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return index.Results{}, fmt.Errorf("elasticsearch search HTTP %d: %s", resp.StatusCode, b)
	}
	var sr struct {
		Hits struct {
			Total struct{ Value int `json:"value"` } `json:"total"`
			Hits  []struct {
				ID        string              `json:"_id"`
				Score     float64             `json:"_score"`
				Highlight map[string][]string `json:"highlight"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return index.Results{}, fmt.Errorf("elasticsearch search decode: %w", err)
	}
	results := index.Results{Total: sr.Hits.Total.Value}
	for _, hit := range sr.Hits.Hits {
		h := index.Hit{DocID: hit.ID, Score: hit.Score}
		if frags := hit.Highlight["text"]; len(frags) > 0 {
			h.Snippet = frags[0]
		}
		results.Hits = append(results.Hits, h)
	}
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
