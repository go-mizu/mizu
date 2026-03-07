package qdrant

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	"github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/internal/httpx"
)

const defaultAddr = "http://localhost:6333"

func init() {
	vector.Register("qdrant", func(cfg vector.Config) (vector.Store, error) {
		return New(cfg), nil
	})
}

type Store struct {
	vector.BaseExternal
	client  *http.Client
	headers map[string]string
}

func New(cfg vector.Config) *Store {
	s := &Store{client: &http.Client{Timeout: 30 * time.Second}}
	s.SetAddr(cfg.Addr)
	if key := cfg.Options["api_key"]; key != "" {
		s.headers = map[string]string{"api-key": key}
	}
	return s
}

func (s *Store) Collection(name string) vector.Collection { return &collection{store: s, name: name} }

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
	if err := c.createCollection(ctx, dim); err != nil {
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

	for i := range items {
		if len(items[i].Vector) != dim {
			return fmt.Errorf("qdrant: dimension mismatch at item %q: got %d want %d", items[i].ID, len(items[i].Vector), dim)
		}
	}
	if !inited {
		if err := c.createCollection(ctx, dim); err != nil {
			return err
		}
		c.mu.Lock()
		c.inited = true
		c.mu.Unlock()
	}

	type point struct {
		ID      any            `json:"id"`
		Vector  []float32      `json:"vector"`
		Payload map[string]any `json:"payload,omitempty"`
	}
	payload := struct {
		Points []point `json:"points"`
	}{Points: make([]point, 0, len(items))}
	for _, it := range items {
		payload.Points = append(payload.Points, point{
			ID:      toQdrantID(it.ID),
			Vector:  it.Vector,
			Payload: httpx.ToAnyMetadata(it.Metadata),
		})
	}
	url := c.store.endpoint("/collections/" + c.name + "/points?wait=true")
	return httpx.DoJSON(ctx, c.store.client, http.MethodPut, url, c.store.headers, payload, nil)
}

func toQdrantID(id string) any {
	if n, err := strconv.ParseUint(id, 10, 64); err == nil {
		return n
	}
	return uuid.NewSHA1(uuid.Nil, []byte(id)).String()
}

func (c *collection) Search(ctx context.Context, q vector.Query) (vector.Results, error) {
	k := httpx.NormalizeK(q.K)
	filter := map[string]any{}
	if len(q.Filter) > 0 {
		must := make([]map[string]any, 0, len(q.Filter))
		for key, val := range q.Filter {
			must = append(must, map[string]any{"key": key, "match": map[string]any{"value": val}})
		}
		filter["must"] = must
	}
	payload := map[string]any{"query": q.Vector, "limit": k, "with_payload": true}
	if len(filter) > 0 {
		payload["filter"] = filter
	}

	var resp struct {
		Result struct {
			Points []struct {
				ID       any            `json:"id"`
				Score    float64        `json:"score"`
				Distance float64        `json:"distance"`
				Payload  map[string]any `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}
	url := c.store.endpoint("/collections/" + c.name + "/points/query")
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, url, c.store.headers, payload, &resp); err != nil {
		return vector.Results{}, err
	}

	hits := make([]vector.Hit, 0, len(resp.Result.Points))
	for _, p := range resp.Result.Points {
		hit := vector.Hit{ID: fmt.Sprint(p.ID), Score: p.Score, Distance: p.Distance}
		if len(p.Payload) > 0 {
			hit.Metadata = httpx.ToStringMetadata(p.Payload)
		}
		hits = append(hits, hit)
	}
	return vector.Results{Hits: hits, Total: len(hits)}, nil
}

func (c *collection) createCollection(ctx context.Context, dim int) error {
	payload := map[string]any{"vectors": map[string]any{"size": dim, "distance": "Cosine"}}
	url := c.store.endpoint("/collections/" + c.name)
	err := httpx.DoJSON(ctx, c.store.client, http.MethodPut, url, c.store.headers, payload, nil)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return nil
	}
	return err
}

var _ vector.Store = (*Store)(nil)
var _ vector.Collection = (*collection)(nil)
var _ vector.AddrSetter = (*Store)(nil)
