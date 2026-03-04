package typesense

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	"github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/internal/httpx"
)

const defaultAddr = "http://localhost:8108"

func init() {
	vector.Register("typesense", func(cfg vector.Config) (vector.Store, error) {
		return New(cfg), nil
	})
}

type Store struct {
	vector.BaseExternal
	client  *http.Client
	headers map[string]string
}

func New(cfg vector.Config) *Store {
	apiKey := cfg.Options["api_key"]
	if apiKey == "" {
		apiKey = "mizu-typesense-key"
	}
	s := &Store{client: &http.Client{Timeout: 30 * time.Second}, headers: map[string]string{"X-TYPESENSE-API-KEY": apiKey}}
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

	for _, it := range items {
		if len(it.Vector) != dim {
			return fmt.Errorf("typesense: dimension mismatch at item %q: got %d want %d", it.ID, len(it.Vector), dim)
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

	var ndjson strings.Builder
	enc := json.NewEncoder(&ndjson)
	for _, it := range items {
		doc := map[string]any{"id": it.ID, "text": it.ID, "embedding": it.Vector}
		for k, v := range it.Metadata {
			doc[k] = v
		}
		_ = enc.Encode(doc)
	}
	endpoint := c.store.endpoint("/collections/" + c.name + "/documents/import?action=upsert")
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(ndjson.String()))
	req.Header.Set("Content-Type", "text/plain")
	for k, v := range c.store.headers {
		req.Header.Set(k, v)
	}
	resp, err := c.store.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("typesense import status %d", resp.StatusCode)
	}
	return nil
}

func (c *collection) Search(ctx context.Context, q vector.Query) (vector.Results, error) {
	k := httpx.NormalizeK(q.K)
	parts := make([]string, len(q.Vector))
	for i := range q.Vector {
		parts[i] = fmt.Sprintf("%g", q.Vector[i])
	}
	vals := url.Values{}
	vals.Set("q", "*")
	vals.Set("query_by", "text")
	vals.Set("vector_query", fmt.Sprintf("embedding:([%s], k:%d)", strings.Join(parts, ","), k))
	if len(q.Filter) > 0 {
		f := make([]string, 0, len(q.Filter))
		for key, val := range q.Filter {
			f = append(f, fmt.Sprintf("%s:=%s", key, val))
		}
		vals.Set("filter_by", strings.Join(f, " && "))
	}

	var resp struct {
		Found int `json:"found"`
		Hits  []struct {
			Document       map[string]any `json:"document"`
			VectorDistance float64        `json:"vector_distance"`
		} `json:"hits"`
	}
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodGet, c.store.endpoint("/collections/"+c.name+"/documents/search?"+vals.Encode()), c.store.headers, nil, &resp); err != nil {
		return vector.Results{}, err
	}

	hits := make([]vector.Hit, 0, len(resp.Hits))
	for _, h := range resp.Hits {
		id := fmt.Sprint(h.Document["id"])
		hits = append(hits, vector.Hit{ID: id, Distance: h.VectorDistance, Score: 1 / (1 + h.VectorDistance)})
	}
	total := resp.Found
	if total == 0 {
		total = len(hits)
	}
	return vector.Results{Hits: hits, Total: total}, nil
}

func (c *collection) createCollection(ctx context.Context, dim int) error {
	payload := map[string]any{
		"name": c.name,
		"fields": []map[string]any{
			{"name": "id", "type": "string"},
			{"name": "text", "type": "string"},
			{"name": "embedding", "type": "float[]", "num_dim": dim},
		},
	}
	err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/collections"), c.store.headers, payload, nil)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "already") || strings.Contains(lower, "exists") {
			return nil
		}
	}
	return err
}

var _ vector.Store = (*Store)(nil)
var _ vector.Collection = (*collection)(nil)
var _ vector.AddrSetter = (*Store)(nil)
