package meilisearch

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

const defaultAddr = "http://localhost:7700"

func init() {
	vector.Register("meilisearch", func(cfg vector.Config) (vector.Store, error) {
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
		s.headers = map[string]string{"Authorization": "Bearer " + key}
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
	if err := c.createIndexAndEmbedder(ctx, 3); err != nil {
		return err
	}
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
			return fmt.Errorf("meilisearch: dimension mismatch at item %q: got %d want %d", it.ID, len(it.Vector), dim)
		}
	}
	if !inited {
		if err := c.createIndexAndEmbedder(ctx, dim); err != nil {
			return err
		}
		c.mu.Lock()
		c.inited = true
		c.mu.Unlock()
	}

	docs := make([]map[string]any, 0, len(items))
	for _, it := range items {
		doc := map[string]any{"id": it.ID, "_vectors": map[string]any{"default": it.Vector}}
		for k, v := range it.Metadata {
			doc[k] = v
		}
		docs = append(docs, doc)
	}

	var task struct {
		TaskUID int64 `json:"taskUid"`
	}
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/indexes/"+c.name+"/documents"), c.store.headers, docs, &task); err != nil {
		return err
	}
	return c.waitTask(ctx, task.TaskUID)
}

func (c *collection) Search(ctx context.Context, q vector.Query) (vector.Results, error) {
	payload := map[string]any{
		"vector": q.Vector,
		"hybrid": map[string]any{"embedder": "default", "semanticRatio": 1.0},
		"limit":  httpx.NormalizeK(q.K),
	}
	if len(q.Filter) > 0 {
		parts := make([]string, 0, len(q.Filter))
		for k, v := range q.Filter {
			parts = append(parts, fmt.Sprintf("%s = '%s'", k, strings.ReplaceAll(v, "'", "\\'")))
		}
		payload["filter"] = strings.Join(parts, " AND ")
	}

	var resp struct {
		Hits               []map[string]any `json:"hits"`
		EstimatedTotalHits int              `json:"estimatedTotalHits"`
	}
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/indexes/"+c.name+"/search"), c.store.headers, payload, &resp); err != nil {
		return vector.Results{}, err
	}
	hits := make([]vector.Hit, 0, len(resp.Hits))
	for _, h := range resp.Hits {
		hits = append(hits, vector.Hit{ID: fmt.Sprint(h["id"])})
	}
	total := resp.EstimatedTotalHits
	if total == 0 {
		total = len(hits)
	}
	return vector.Results{Hits: hits, Total: total}, nil
}

func (c *collection) createIndexAndEmbedder(ctx context.Context, dim int) error {
	var task struct {
		TaskUID int64 `json:"taskUid"`
	}
	err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/indexes"), c.store.headers, map[string]any{"uid": c.name, "primaryKey": "id"}, &task)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if !strings.Contains(lower, "already") && !strings.Contains(lower, "exists") {
			return err
		}
	}
	if task.TaskUID > 0 {
		if err := c.waitTask(ctx, task.TaskUID); err != nil {
			return err
		}
	}

	task = struct {
		TaskUID int64 `json:"taskUid"`
	}{}
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPatch, c.store.endpoint("/indexes/"+c.name+"/settings/embedders"), c.store.headers,
		map[string]any{"default": map[string]any{"source": "userProvided", "dimensions": dim}}, &task); err != nil {
		return err
	}
	if task.TaskUID > 0 {
		if err := c.waitTask(ctx, task.TaskUID); err != nil {
			return err
		}
	}
	return nil
}

func (c *collection) waitTask(ctx context.Context, taskID int64) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		var resp struct {
			Status string `json:"status"`
			Error  any    `json:"error"`
		}
		if err := httpx.DoJSON(ctx, c.store.client, http.MethodGet, c.store.endpoint(fmt.Sprintf("/tasks/%d", taskID)), c.store.headers, nil, &resp); err != nil {
			return err
		}
		switch resp.Status {
		case "succeeded":
			return nil
		case "failed", "canceled":
			return fmt.Errorf("meilisearch task %d %s: %v", taskID, resp.Status, resp.Error)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("meilisearch task %d timeout", taskID)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

var _ vector.Store = (*Store)(nil)
var _ vector.Collection = (*collection)(nil)
var _ vector.AddrSetter = (*Store)(nil)
