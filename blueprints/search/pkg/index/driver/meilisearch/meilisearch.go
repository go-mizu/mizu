package meilisearch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ms "github.com/meilisearch/meilisearch-go"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

const (
	defaultAddr = "http://localhost:7700"
	indexUID    = "fts_docs"
	primaryKey  = "doc_id"
)

func init() {
	index.Register("meilisearch", func() index.Engine { return &Engine{} })
}

// Engine is an external FTS engine backed by a Meilisearch HTTP service.
type Engine struct {
	index.BaseExternal
	client ms.ServiceManager
	idx    ms.IndexManager
}

func (e *Engine) Name() string { return "meilisearch" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	addr := e.EffectiveAddr(defaultAddr)
	e.client = ms.New(addr)

	// Create or get the index.
	if task, cerr := e.client.CreateIndexWithContext(ctx, &ms.IndexConfig{
		Uid:        indexUID,
		PrimaryKey: primaryKey,
	}); cerr != nil {
		// If creation failed, verify the index already exists (it may have been created previously).
		if _, getErr := e.client.GetIndexWithContext(ctx, indexUID); getErr != nil {
			return fmt.Errorf("meilisearch create index: %w", cerr)
		}
		// Index already existed — continue.
	} else {
		if _, err := e.client.WaitForTask(task.TaskUID, 50*time.Millisecond); err != nil {
			return fmt.Errorf("meilisearch: wait for create index: %w", err)
		}
	}

	e.idx = e.client.Index(indexUID)

	// Set searchable attributes to only the "text" field.
	attrs := []string{"text"}
	attrTask, err := e.idx.UpdateSearchableAttributesWithContext(ctx, &attrs)
	if err != nil {
		return fmt.Errorf("meilisearch: update searchable attributes: %w", err)
	}
	if _, err := e.client.WaitForTask(attrTask.TaskUID, 50*time.Millisecond); err != nil {
		return fmt.Errorf("meilisearch: wait for searchable attributes: %w", err)
	}

	return nil
}

func (e *Engine) Close() error {
	// HTTP client — nothing to close.
	return nil
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	if e.idx == nil {
		return index.EngineStats{}, nil
	}
	stats, err := e.idx.GetStatsWithContext(ctx)
	if err != nil {
		return index.EngineStats{}, fmt.Errorf("meilisearch: stats: %w", err)
	}
	return index.EngineStats{
		DocCount: stats.NumberOfDocuments,
	}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}
	if e.idx == nil {
		return fmt.Errorf("meilisearch: engine not opened")
	}
	payload := make([]map[string]any, len(docs))
	for i, d := range docs {
		payload[i] = map[string]any{
			"doc_id": d.DocID,
			"text":   string(d.Text),
		}
	}
	pk := primaryKey
	task, err := e.idx.AddDocumentsWithContext(ctx, payload, &ms.DocumentOptions{PrimaryKey: &pk})
	if err != nil {
		return fmt.Errorf("meilisearch: add documents: %w", err)
	}
	// Wait for indexing to complete so documents are immediately searchable.
	if _, err := e.client.WaitForTask(task.TaskUID, 50*time.Millisecond); err != nil {
		return fmt.Errorf("meilisearch: wait for index task: %w", err)
	}
	return nil
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	if e.idx == nil {
		return index.Results{}, nil
	}
	limit := int64(q.Limit)
	if limit <= 0 {
		limit = 10
	}
	req := &ms.SearchRequest{
		Limit:  limit,
		Offset: int64(q.Offset),
	}
	res, err := e.idx.SearchWithContext(ctx, q.Text, req)
	if err != nil {
		return index.Results{}, fmt.Errorf("meilisearch: search: %w", err)
	}

	results := index.Results{
		Total: int(res.EstimatedTotalHits),
	}
	for _, hit := range res.Hits {
		docID := stringField(hit, "doc_id")
		text := stringField(hit, "text")
		snippet := text
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		results.Hits = append(results.Hits, index.Hit{
			DocID:   docID,
			Score:   0,
			Snippet: snippet,
		})
	}
	return results, nil
}

// stringField extracts a string value from a meilisearch Hit (map[string]json.RawMessage).
func stringField(hit ms.Hit, key string) string {
	raw, ok := hit[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

var _ index.Engine = (*Engine)(nil)
var _ index.AddrSetter = (*Engine)(nil)
