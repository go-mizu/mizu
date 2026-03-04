package weaviate

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	"github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/internal/httpx"
)

const defaultAddr = "http://localhost:8080"

var nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func init() {
	vector.Register("weaviate", func(cfg vector.Config) (vector.Store, error) {
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
	if token := cfg.Options["api_key"]; token != "" {
		s.headers = map[string]string{"Authorization": "Bearer " + token}
	}
	return s
}

func (s *Store) Collection(name string) vector.Collection {
	return &collection{store: s, className: className(name)}
}

func (s *Store) endpoint(path string) string {
	return strings.TrimRight(s.EffectiveAddr(defaultAddr), "/") + path
}

type collection struct {
	store     *Store
	className string

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
	if err := c.createClass(ctx); err != nil {
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
			return fmt.Errorf("weaviate: dimension mismatch at item %q: got %d want %d", it.ID, len(it.Vector), dim)
		}
	}
	if !inited {
		if err := c.createClass(ctx); err != nil {
			return err
		}
		c.mu.Lock()
		c.inited = true
		c.mu.Unlock()
	}

	type obj struct {
		Class      string         `json:"class"`
		ID         string         `json:"id"`
		Properties map[string]any `json:"properties,omitempty"`
		Vector     []float32      `json:"vector,omitempty"`
	}
	payload := struct {
		Objects []obj `json:"objects"`
	}{Objects: make([]obj, 0, len(items))}
	for _, it := range items {
		props := map[string]any{}
		for k, v := range it.Metadata {
			props[k] = v
		}
		payload.Objects = append(payload.Objects, obj{
			Class:      c.className,
			ID:         uuid.NewSHA1(uuid.Nil, []byte(it.ID)).String(),
			Properties: props,
			Vector:     it.Vector,
		})
	}

	return httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/v1/batch/objects"), c.store.headers, payload, nil)
}

func (c *collection) Search(ctx context.Context, q vector.Query) (vector.Results, error) {
	k := httpx.NormalizeK(q.K)
	query := fmt.Sprintf(`{Get{%s(nearVector:{vector:%s}, limit:%d%s){_additional{id distance certainty score}}}}`, c.className, floatArrayLiteral(q.Vector), k, buildWhereClause(q.Filter))
	payload := map[string]any{"query": query}

	var resp struct {
		Data map[string]map[string][]struct {
			Additional struct {
				ID        string  `json:"id"`
				Distance  float64 `json:"distance"`
				Certainty float64 `json:"certainty"`
				Score     any     `json:"score"`
			} `json:"_additional"`
		} `json:"data"`
		Errors []map[string]any `json:"errors"`
	}
	if err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/v1/graphql"), c.store.headers, payload, &resp); err != nil {
		return vector.Results{}, err
	}
	if len(resp.Errors) > 0 {
		return vector.Results{}, fmt.Errorf("weaviate graphql returned errors")
	}
	rows := resp.Data["Get"][c.className]
	hits := make([]vector.Hit, 0, len(rows))
	for _, r := range rows {
		score := asFloat64(r.Additional.Score)
		if score == 0 {
			score = r.Additional.Certainty
		}
		hits = append(hits, vector.Hit{ID: r.Additional.ID, Score: score, Distance: r.Additional.Distance})
	}
	return vector.Results{Hits: hits, Total: len(hits)}, nil
}

func asFloat64(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	default:
		return 0
	}
}

func (c *collection) createClass(ctx context.Context) error {
	err := httpx.DoJSON(ctx, c.store.client, http.MethodPost, c.store.endpoint("/v1/schema"), c.store.headers,
		map[string]any{"class": c.className, "vectorizer": "none"}, nil)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "already exists") || strings.Contains(lower, "422") {
			return nil
		}
	}
	return err
}

func className(name string) string {
	n := nonAlphaNum.ReplaceAllString(name, " ")
	parts := strings.Fields(strings.ToLower(n))
	if len(parts) == 0 {
		return "Collection"
	}
	for i := range parts {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func floatArrayLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i := range v {
		parts[i] = fmt.Sprintf("%g", v[i])
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func buildWhereClause(filter map[string]string) string {
	if len(filter) == 0 {
		return ""
	}
	ops := make([]string, 0, len(filter))
	for k, v := range filter {
		ops = append(ops, fmt.Sprintf(`{path:["%s"],operator:Equal,valueText:"%s"}`, escapeGraphQL(k), escapeGraphQL(v)))
	}
	return fmt.Sprintf(", where:{operator:And, operands:[%s]}", strings.Join(ops, ","))
}

func escapeGraphQL(s string) string {
	s = strings.ReplaceAll(s, `\\`, `\\\\`)
	return strings.ReplaceAll(s, `"`, `\\"`)
}

var _ vector.Store = (*Store)(nil)
var _ vector.Collection = (*collection)(nil)
var _ vector.AddrSetter = (*Store)(nil)
