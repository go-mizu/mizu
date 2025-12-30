package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

// GlobalsStore handles global document operations.
type GlobalsStore struct {
	db *sql.DB
}

// NewGlobalsStore creates a new GlobalsStore.
func NewGlobalsStore(db *sql.DB) *GlobalsStore {
	return &GlobalsStore{db: db}
}

// Global represents a global document.
type Global struct {
	ID        string
	Slug      string
	Data      map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Get retrieves a global by slug.
func (s *GlobalsStore) Get(ctx context.Context, slug string) (*Global, error) {
	query := `SELECT id, slug, data, created_at, updated_at FROM _globals WHERE slug = ?`

	var g Global
	var dataJSON string

	err := s.db.QueryRowContext(ctx, query, slug).Scan(
		&g.ID, &g.Slug, &dataJSON, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get global: %w", err)
	}

	if err := json.Unmarshal([]byte(dataJSON), &g.Data); err != nil {
		return nil, fmt.Errorf("unmarshal global data: %w", err)
	}

	return &g, nil
}

// Update updates or creates a global.
func (s *GlobalsStore) Update(ctx context.Context, slug string, data map[string]any) (*Global, error) {
	now := time.Now()

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal global data: %w", err)
	}

	// Check if exists
	existing, err := s.Get(ctx, slug)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// Update
		query := `UPDATE _globals SET data = ?, updated_at = ? WHERE slug = ?`
		_, err = s.db.ExecContext(ctx, query, string(dataJSON), now, slug)
		if err != nil {
			return nil, fmt.Errorf("update global: %w", err)
		}
		return &Global{
			ID:        existing.ID,
			Slug:      slug,
			Data:      data,
			CreatedAt: existing.CreatedAt,
			UpdatedAt: now,
		}, nil
	}

	// Create
	id := ulid.New()
	query := `INSERT INTO _globals (id, slug, data, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	_, err = s.db.ExecContext(ctx, query, id, slug, string(dataJSON), now, now)
	if err != nil {
		return nil, fmt.Errorf("create global: %w", err)
	}

	return &Global{
		ID:        id,
		Slug:      slug,
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// List lists all globals.
func (s *GlobalsStore) List(ctx context.Context) ([]*Global, error) {
	query := `SELECT id, slug, data, created_at, updated_at FROM _globals ORDER BY slug`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list globals: %w", err)
	}
	defer rows.Close()

	var globals []*Global
	for rows.Next() {
		var g Global
		var dataJSON string

		if err := rows.Scan(&g.ID, &g.Slug, &dataJSON, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan global: %w", err)
		}

		if err := json.Unmarshal([]byte(dataJSON), &g.Data); err != nil {
			return nil, fmt.Errorf("unmarshal global data: %w", err)
		}

		globals = append(globals, &g)
	}

	return globals, nil
}
