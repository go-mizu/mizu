package pgvector

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	"github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/internal/httpx"
)

const defaultDSN = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

var invalidTable = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func init() {
	vector.Register("pgvector", func(cfg vector.Config) (vector.Store, error) {
		return New(cfg)
	})
}

type Store struct {
	vector.BaseExternal
	pool *pgxpool.Pool
}

func New(cfg vector.Config) (*Store, error) {
	s := &Store{}
	s.SetAddr(cfg.Addr)
	dsn := s.EffectiveAddr(defaultDSN)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("pgvector open pool: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgvector ping: %w", err)
	}
	s.pool = pool
	return s, nil
}

func (s *Store) Collection(name string) vector.Collection {
	return &collection{store: s, name: safeTable(name)}
}

func (s *Store) Close() error {
	if s.pool != nil {
		s.pool.Close()
		s.pool = nil
	}
	return nil
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
	if err := c.createSchema(ctx, dim); err != nil {
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
			return fmt.Errorf("pgvector: dimension mismatch at item %q: got %d want %d", it.ID, len(it.Vector), dim)
		}
	}
	if !inited {
		if err := c.createSchema(ctx, dim); err != nil {
			return err
		}
		c.mu.Lock()
		c.inited = true
		c.mu.Unlock()
	}

	for _, it := range items {
		meta, _ := json.Marshal(httpx.ToAnyMetadata(it.Metadata))
		vec := vectorLiteral(it.Vector)
		q := fmt.Sprintf(`
INSERT INTO %s (id, embedding, metadata)
VALUES ($1, $2::vector, $3::jsonb)
ON CONFLICT (id) DO UPDATE SET embedding = EXCLUDED.embedding, metadata = EXCLUDED.metadata`, c.name)
		if _, err := c.store.pool.Exec(ctx, q, it.ID, vec, string(meta)); err != nil {
			return fmt.Errorf("pgvector upsert %s: %w", it.ID, err)
		}
	}
	return nil
}

func (c *collection) Search(ctx context.Context, q vector.Query) (vector.Results, error) {
	k := httpx.NormalizeK(q.K)
	where := ""
	args := []any{vectorLiteral(q.Vector), k}
	if len(q.Filter) > 0 {
		b, _ := json.Marshal(httpx.ToAnyMetadata(q.Filter))
		where = "WHERE metadata @> $3::jsonb"
		args = append(args, string(b))
	}

	sql := fmt.Sprintf(`
SELECT id, (1 - (embedding <=> $1::vector)) AS score, (embedding <=> $1::vector) AS distance
FROM %s
%s
ORDER BY embedding <=> $1::vector
LIMIT $2`, c.name, where)

	rows, err := c.store.pool.Query(ctx, sql, args...)
	if err != nil {
		return vector.Results{}, fmt.Errorf("pgvector search: %w", err)
	}
	defer rows.Close()

	hits := make([]vector.Hit, 0, k)
	for rows.Next() {
		var h vector.Hit
		if err := rows.Scan(&h.ID, &h.Score, &h.Distance); err != nil {
			return vector.Results{}, fmt.Errorf("pgvector scan: %w", err)
		}
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return vector.Results{}, fmt.Errorf("pgvector rows: %w", err)
	}
	return vector.Results{Hits: hits, Total: len(hits)}, nil
}

func (c *collection) createSchema(ctx context.Context, dim int) error {
	if _, err := c.store.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`); err != nil {
		return fmt.Errorf("pgvector create extension: %w", err)
	}
	q := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id TEXT PRIMARY KEY,
	embedding VECTOR(%d) NOT NULL,
	metadata JSONB DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS %s_embedding_idx
ON %s USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
`, c.name, dim, c.name, c.name)
	_, err := c.store.pool.Exec(ctx, q)
	if err != nil {
		return fmt.Errorf("pgvector create schema: %w", err)
	}
	return nil
}

func safeTable(name string) string {
	t := strings.ToLower(name)
	t = invalidTable.ReplaceAllString(t, "_")
	t = strings.Trim(t, "_")
	if t == "" {
		return "vectors"
	}
	if t[0] >= '0' && t[0] <= '9' {
		return "v_" + t
	}
	return t
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
var _ vector.Closer = (*Store)(nil)
var _ vector.AddrSetter = (*Store)(nil)
