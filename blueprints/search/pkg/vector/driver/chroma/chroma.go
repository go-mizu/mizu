package chroma

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

const defaultAddr = "http://localhost:8000"

func init() {
	vector.Register("chroma", func(cfg vector.Config) (vector.Store, error) {
		return New(cfg), nil
	})
}

type Store struct {
	vector.BaseExternal
	client *http.Client
}

func New(cfg vector.Config) *Store {
	s := &Store{client: &http.Client{Timeout: 30 * time.Second}}
	s.SetAddr(cfg.Addr)
	return s
}

func (s *Store) Collection(name string) vector.Collection { return &collection{store: s, name: name} }

func (s *Store) endpoint(path string) string {
	return strings.TrimRight(s.EffectiveAddr(defaultAddr), "/") + path
}

type collection struct {
	store *Store
	name  string

	mu           sync.Mutex
	dimension    int
	inited       bool
	collectionID string
}

func (c *collection) Init(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.inited {
		return nil
	}
	id, err := c.createCollection(ctx)
	if err != nil {
		return err
	}
	c.collectionID = id
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
	cid := c.collectionID
	c.mu.Unlock()

	for _, it := range items {
		if len(it.Vector) != dim {
			return fmt.Errorf("chroma: dimension mismatch at item %q: got %d want %d", it.ID, len(it.Vector), dim)
		}
	}
	if !inited || cid == "" {
		id, err := c.createCollection(ctx)
		if err != nil {
			return err
		}
		c.mu.Lock()
		c.inited = true
		c.collectionID = id
		c.mu.Unlock()
		cid = id
	}

	ids := make([]string, 0, len(items))
	embeddings := make([][]float32, 0, len(items))
	metas := make([]map[string]any, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
		embeddings = append(embeddings, it.Vector)
		metas = append(metas, httpx.ToAnyMetadata(it.Metadata))
	}
	payload := map[string]any{"ids": ids, "embeddings": embeddings, "metadatas": metas}
	url := c.store.endpoint("/api/v2/tenants/default_tenant/databases/default_database/collections/" + cid + "/add")
	return httpx.DoJSON(ctx, c.store.client, http.MethodPost, url, nil, payload, nil)
}

func (c *collection) Search(ctx context.Context, q vector.Query) (vector.Results, error) {
	c.mu.Lock()
	cid := c.collectionID
	c.mu.Unlock()
	if cid == "" {
		id, err := c.createCollection(ctx)
		if err != nil {
			return vector.Results{}, err
		}
		cid = id
		c.mu.Lock()
		c.collectionID = id
		c.inited = true
		c.mu.Unlock()
	}

	payload := map[string]any{
		"query_embeddings": [][]float32{q.Vector},
		"n_results":        httpx.NormalizeK(q.K),
	}
	if len(q.Filter) > 0 {
		where := make(map[string]any, len(q.Filter))
		for k, v := range q.Filter {
			where[k] = v
		}
		payload["where"] = where
	}

	var resp struct {
		IDs       [][]string  `json:"ids"`
		Distances [][]float64 `json:"distances"`
	}
	url := c.store.endpoint("/api/v2/tenants/default_tenant/databases/default_database/collections/" + cid + "/query")
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, url, nil, payload, &resp); err != nil {
		return vector.Results{}, err
	}
	if len(resp.IDs) == 0 {
		return vector.Results{}, nil
	}
	ids := resp.IDs[0]
	dists := []float64{}
	if len(resp.Distances) > 0 {
		dists = resp.Distances[0]
	}
	hits := make([]vector.Hit, 0, len(ids))
	for i, id := range ids {
		d := 0.0
		if i < len(dists) {
			d = dists[i]
		}
		hits = append(hits, vector.Hit{ID: id, Distance: d, Score: 1 / (1 + d)})
	}
	return vector.Results{Hits: hits, Total: len(hits)}, nil
}

func (c *collection) createCollection(ctx context.Context) (string, error) {
	payload := map[string]any{"name": c.name, "get_or_create": true}
	var resp struct {
		ID string `json:"id"`
	}
	url := c.store.endpoint("/api/v2/tenants/default_tenant/databases/default_database/collections")
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, url, nil, payload, &resp); err != nil {
		return "", err
	}
	if resp.ID == "" {
		return "", fmt.Errorf("chroma: empty collection id")
	}
	return resp.ID, nil
}

var _ vector.Store = (*Store)(nil)
var _ vector.Collection = (*collection)(nil)
var _ vector.AddrSetter = (*Store)(nil)
