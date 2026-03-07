package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	"github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/internal/httpx"
)

const defaultAddr = "http://localhost:9200"

func init() {
	vector.Register("elasticsearch", func(cfg vector.Config) (vector.Store, error) {
		return New(cfg), nil
	})
}

type Store struct {
	vector.BaseExternal
	client *http.Client
}

func New(cfg vector.Config) *Store {
	s := &Store{client: &http.Client{Timeout: 45 * time.Second}}
	s.SetAddr(cfg.Addr)
	return s
}

func (s *Store) Collection(name string) vector.Collection {
	return &collection{store: s, name: name}
}

func (s *Store) endpoint(path string) string {
	return strings.TrimRight(s.EffectiveAddr(defaultAddr), "/") + path
}

type collection struct {
	store *Store
	name  string

	mu        sync.Mutex
	dimension int
	inited    bool
}

func (c *collection) Init(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.inited {
		return nil
	}
	dim := c.dimension
	if dim == 0 {
		dim = 3
	}
	if err := c.createIndex(ctx, dim); err != nil {
		return err
	}
	c.dimension = dim
	c.inited = true
	return nil
}

func (c *collection) Index(ctx context.Context, items []vector.Item) error {
	if len(items) == 0 {
		return nil
	}
	c.mu.Lock()
	if c.dimension == 0 {
		c.dimension = len(items[0].Vector)
	}
	dim := c.dimension
	inited := c.inited
	c.mu.Unlock()

	for _, it := range items {
		if len(it.Vector) != dim {
			return fmt.Errorf("elasticsearch: dimension mismatch at item %q: got %d want %d", it.ID, len(it.Vector), dim)
		}
	}
	if !inited {
		if err := c.createIndex(ctx, dim); err != nil {
			return err
		}
		c.mu.Lock()
		c.inited = true
		c.mu.Unlock()
	}

	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	for _, it := range items {
		_ = enc.Encode(map[string]any{"index": map[string]any{"_index": c.name, "_id": it.ID}})
		_ = enc.Encode(map[string]any{"vector": it.Vector, "metadata": httpx.ToAnyMetadata(it.Metadata)})
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.store.endpoint("/_bulk?refresh=true"), strings.NewReader(buf.String()))
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := c.store.client.Do(req)
	if err != nil {
		return fmt.Errorf("elasticsearch bulk: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("elasticsearch bulk: status %d", resp.StatusCode)
	}
	return nil
}

func (c *collection) Search(ctx context.Context, q vector.Query) (vector.Results, error) {
	k := httpx.NormalizeK(q.K)
	must := make([]map[string]any, 0, len(q.Filter))
	for key, val := range q.Filter {
		must = append(must, map[string]any{"term": map[string]any{"metadata." + key: val}})
	}

	payload := map[string]any{
		"knn": map[string]any{
			"field":          "vector",
			"query_vector":   q.Vector,
			"k":              k,
			"num_candidates": max(k*4, 50),
		},
	}
	if len(must) > 0 {
		payload["query"] = map[string]any{"bool": map[string]any{"filter": must}}
	}

	var resp struct {
		Hits struct {
			Total any `json:"total"`
			Hits  []struct {
				ID    string  `json:"_id"`
				Score float64 `json:"_score"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/"+c.name+"/_search"), nil, payload, &resp); err != nil {
		return vector.Results{}, err
	}

	hits := make([]vector.Hit, 0, len(resp.Hits.Hits))
	for _, h := range resp.Hits.Hits {
		hits = append(hits, vector.Hit{ID: h.ID, Score: h.Score})
	}
	return vector.Results{Hits: hits, Total: len(hits)}, nil
}

func (c *collection) createIndex(ctx context.Context, dim int) error {
	payload := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"vector": map[string]any{
					"type":       "dense_vector",
					"dims":       dim,
					"index":      true,
					"similarity": "cosine",
				},
				"metadata": map[string]any{"type": "object", "dynamic": true},
			},
		},
	}
	err := httpx.DoJSON(ctx, c.store.client, http.MethodPut, c.store.endpoint("/"+c.name), nil, payload, nil)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "resource_already_exists_exception") || strings.Contains(lower, "already") {
			return nil
		}
	}
	return err
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var _ vector.Store = (*Store)(nil)
var _ vector.Collection = (*collection)(nil)
var _ vector.AddrSetter = (*Store)(nil)
