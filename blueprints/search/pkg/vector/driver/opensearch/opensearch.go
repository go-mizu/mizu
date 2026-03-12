package opensearch

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	"github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/internal/httpx"
)

const defaultAddr = "http://localhost:9201"

func init() {
	vector.Register("opensearch", func(cfg vector.Config) (vector.Store, error) {
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
			return fmt.Errorf("opensearch: dimension mismatch at item %q: got %d want %d", it.ID, len(it.Vector), dim)
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

	for _, it := range items {
		payload := map[string]any{"vector": it.Vector, "metadata": httpx.ToAnyMetadata(it.Metadata)}
		url := c.store.endpoint("/" + c.name + "/_doc/" + it.ID + "?refresh=true")
		if err := httpx.DoJSON(ctx, c.store.client, http.MethodPut, url, nil, payload, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *collection) Search(ctx context.Context, q vector.Query) (vector.Results, error) {
	k := httpx.NormalizeK(q.K)
	payload := map[string]any{
		"size": k,
		"query": map[string]any{
			"knn": map[string]any{
				"vector": map[string]any{
					"vector": q.Vector,
					"k":      k,
				},
			},
		},
	}
	if len(q.Filter) > 0 {
		filters := make([]map[string]any, 0, len(q.Filter))
		for key, val := range q.Filter {
			filters = append(filters, map[string]any{"term": map[string]any{"metadata." + key: val}})
		}
		payload["query"] = map[string]any{
			"bool": map[string]any{
				"must": map[string]any{
					"knn": map[string]any{
						"vector": map[string]any{
							"vector": q.Vector,
							"k":      k,
						},
					},
				},
				"filter": filters,
			},
		}
	}

	var resp struct {
		Hits struct {
			Hits []struct {
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
		"settings": map[string]any{"index": map[string]any{"knn": true}},
		"mappings": map[string]any{
			"properties": map[string]any{
				"vector": map[string]any{
					"type":      "knn_vector",
					"dimension": dim,
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

var _ vector.Store = (*Store)(nil)
var _ vector.Collection = (*collection)(nil)
var _ vector.AddrSetter = (*Store)(nil)
