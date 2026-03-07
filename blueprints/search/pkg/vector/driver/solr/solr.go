package solr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	"github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/internal/httpx"
)

const (
	defaultAddr = "http://localhost:8983"
	defaultCore = "gettingstarted"
)

func init() {
	vector.Register("solr", func(cfg vector.Config) (vector.Store, error) {
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
	return &collection{store: s, core: defaultCore}
}

func (s *Store) endpoint(path string) string {
	return strings.TrimRight(s.EffectiveAddr(defaultAddr), "/") + path
}

type collection struct {
	store *Store
	core  string

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
	if err := c.ensureSchema(ctx, dim); err != nil {
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
			return fmt.Errorf("solr: dimension mismatch at item %q: got %d want %d", it.ID, len(it.Vector), dim)
		}
	}
	if !inited {
		if err := c.ensureSchema(ctx, dim); err != nil {
			return err
		}
		c.mu.Lock()
		c.inited = true
		c.mu.Unlock()
	}

	docs := make([]map[string]any, 0, len(items))
	for _, it := range items {
		doc := map[string]any{"id": it.ID, "text": it.ID, "embedding": it.Vector}
		for k, v := range it.Metadata {
			doc[k] = v
		}
		docs = append(docs, doc)
	}
	return httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/solr/"+c.core+"/update?commit=true"), nil, docs, nil)
}

func (c *collection) Search(ctx context.Context, q vector.Query) (vector.Results, error) {
	k := httpx.NormalizeK(q.K)
	vals := url.Values{}
	vals.Set("q", fmt.Sprintf("{!knn f=embedding topK=%d}%s", k, vectorLiteral(q.Vector)))
	vals.Set("fl", "id,score")
	vals.Set("rows", fmt.Sprintf("%d", k))
	vals.Set("wt", "json")
	if len(q.Filter) > 0 {
		parts := make([]string, 0, len(q.Filter))
		for key, val := range q.Filter {
			parts = append(parts, fmt.Sprintf("%s:\"%s\"", key, strings.ReplaceAll(val, `"`, `\\"`)))
		}
		vals.Set("fq", strings.Join(parts, " AND "))
	}

	var resp struct {
		Response struct {
			NumFound int `json:"numFound"`
			Docs     []struct {
				ID    string  `json:"id"`
				Score float64 `json:"score"`
			} `json:"docs"`
		} `json:"response"`
	}
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodGet, c.store.endpoint("/solr/"+c.core+"/select?"+vals.Encode()), nil, nil, &resp); err != nil {
		return vector.Results{}, err
	}
	hits := make([]vector.Hit, 0, len(resp.Response.Docs))
	for _, d := range resp.Response.Docs {
		hits = append(hits, vector.Hit{ID: d.ID, Score: d.Score})
	}
	return vector.Results{Hits: hits, Total: resp.Response.NumFound}, nil
}

func (c *collection) ensureSchema(ctx context.Context, dim int) error {
	schemaURL := c.store.endpoint("/solr/" + c.core + "/schema")
	typeReq := map[string]any{"add-field-type": map[string]any{
		"name":               "knn_vector",
		"class":              "solr.DenseVectorField",
		"vectorDimension":    dim,
		"similarityFunction": "cosine",
	}}
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, schemaURL, nil, typeReq, nil); err != nil {
		lower := strings.ToLower(err.Error())
		if !strings.Contains(lower, "already exists") {
			return err
		}
	}
	for _, req := range []map[string]any{
		{"add-field": map[string]any{"name": "embedding", "type": "knn_vector", "stored": true, "indexed": true}},
		{"add-field": map[string]any{"name": "text", "type": "text_general", "stored": true, "indexed": true}},
	} {
		if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, schemaURL, nil, req, nil); err != nil {
			lower := strings.ToLower(err.Error())
			if !strings.Contains(lower, "already exists") {
				return err
			}
		}
	}
	return nil
}

func vectorLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i := range v {
		parts[i] = fmt.Sprintf("%g", v[i])
	}
	return "[" + strings.Join(parts, ",") + "]"
}

var _ vector.Store = (*Store)(nil)
var _ vector.Collection = (*collection)(nil)
var _ vector.AddrSetter = (*Store)(nil)
