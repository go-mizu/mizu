package milvus

import (
	"context"
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	"github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/internal/httpx"
)

const defaultAddr = "http://localhost:19530"

func init() {
	vector.Register("milvus", func(cfg vector.Config) (vector.Store, error) {
		return New(cfg), nil
	})
}

type Store struct {
	vector.BaseExternal
	client  *http.Client
	headers map[string]string
}

func New(cfg vector.Config) *Store {
	token := cfg.Options["token"]
	if token == "" {
		token = "root:Milvus"
	}
	s := &Store{
		client:  &http.Client{Timeout: 45 * time.Second},
		headers: map[string]string{"Authorization": "Bearer " + token},
	}
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
	if err := c.loadCollection(ctx); err != nil {
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
			return fmt.Errorf("milvus: dimension mismatch at item %q: got %d want %d", it.ID, len(it.Vector), dim)
		}
	}
	if !inited {
		if err := c.createCollection(ctx, dim); err != nil {
			return err
		}
		if err := c.loadCollection(ctx); err != nil {
			return err
		}
		c.mu.Lock()
		c.inited = true
		c.mu.Unlock()
	}

	rows := make([]map[string]any, 0, len(items))
	for _, it := range items {
		rows = append(rows, map[string]any{
			"id":       toMilvusID(it.ID),
			"vector":   it.Vector,
			"metadata": httpx.ToAnyMetadata(it.Metadata),
		})
	}
	payload := map[string]any{"collectionName": c.name, "data": rows}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
	}
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/v2/vectordb/entities/insert"), c.store.headers, payload, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("milvus insert code=%d: %s", resp.Code, resp.Msg)
	}
	_ = c.loadCollection(ctx)
	return nil
}

func toMilvusID(id string) int64 {
	if n, err := strconv.ParseInt(id, 10, 64); err == nil {
		return n
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(id))
	return int64(h.Sum64() & 0x7fffffffffffffff)
}

func (c *collection) Search(ctx context.Context, q vector.Query) (vector.Results, error) {
	k := httpx.NormalizeK(q.K)
	payload := map[string]any{
		"collectionName": c.name,
		"annsField":      "vector",
		"data":           [][]float32{q.Vector},
		"limit":          k,
	}
	if len(q.Filter) > 0 {
		exprs := make([]string, 0, len(q.Filter))
		for key, val := range q.Filter {
			exprs = append(exprs, fmt.Sprintf("metadata['%s'] == \"%s\"", key, strings.ReplaceAll(val, `"`, `\\"`)))
		}
		payload["filter"] = strings.Join(exprs, " and ")
	}
	for attempt := 0; attempt < 12; attempt++ {
		_ = c.loadCollection(ctx)
		var resp struct {
			Code int `json:"code"`
			Data []struct {
				ID       any     `json:"id"`
				Distance float64 `json:"distance"`
				Score    float64 `json:"score"`
			} `json:"data"`
		}
		if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/v2/vectordb/entities/search"), c.store.headers, payload, &resp); err != nil {
			return vector.Results{}, err
		}
		if resp.Code != 0 {
			return vector.Results{}, fmt.Errorf("milvus search failed: code=%d", resp.Code)
		}
		if len(resp.Data) == 0 && attempt < 11 {
			time.Sleep(250 * time.Millisecond)
			continue
		}
		hits := make([]vector.Hit, 0, len(resp.Data))
		for _, r := range resp.Data {
			score := r.Score
			if score == 0 {
				score = r.Distance
			}
			hits = append(hits, vector.Hit{ID: fmt.Sprint(r.ID), Score: score, Distance: r.Distance})
		}
		return vector.Results{Hits: hits, Total: len(hits)}, nil
	}
	return vector.Results{}, nil
}

func (c *collection) createCollection(ctx context.Context, dim int) error {
	payload := map[string]any{"collectionName": c.name, "dimension": dim}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
	}
	err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/v2/vectordb/collections/create"), c.store.headers, payload, &resp)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "already") || strings.Contains(lower, "exist") {
			return nil
		}
		return err
	}
	if resp.Code != 0 && !strings.Contains(strings.ToLower(resp.Msg), "exist") {
		return fmt.Errorf("milvus create collection failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

func (c *collection) loadCollection(ctx context.Context) error {
	payload := map[string]any{"collectionName": c.name}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
	}
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/v2/vectordb/collections/load"), c.store.headers, payload, &resp); err != nil {
		return err
	}
	if resp.Code != 0 && !strings.Contains(strings.ToLower(resp.Msg), "already") {
		return fmt.Errorf("milvus load collection failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

var _ vector.Store = (*Store)(nil)
var _ vector.Collection = (*collection)(nil)
var _ vector.AddrSetter = (*Store)(nil)
