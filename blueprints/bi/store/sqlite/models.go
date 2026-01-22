package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// ModelStore implements store.ModelStore.
type ModelStore struct {
	db *sql.DB
}

func (s *ModelStore) Create(ctx context.Context, m *store.Model) error {
	if m.ID == "" {
		m.ID = generateID()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	var collID interface{}
	if m.CollectionID != "" {
		collID = m.CollectionID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO models (id, name, description, collection_id, datasource_id, query, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.Name, m.Description, collID, m.DataSourceID, toJSON(m.Query), m.CreatedBy, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *ModelStore) GetByID(ctx context.Context, id string) (*store.Model, error) {
	var m store.Model
	var query string
	var collID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, collection_id, datasource_id, query, created_by, created_at, updated_at
		FROM models WHERE id = ?
	`, id).Scan(&m.ID, &m.Name, &m.Description, &collID, &m.DataSourceID, &query, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.CollectionID = collID.String
	fromJSON(query, &m.Query)
	return &m, nil
}

func (s *ModelStore) List(ctx context.Context) ([]*store.Model, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, collection_id, datasource_id, query, created_by, created_at, updated_at
		FROM models ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Model
	for rows.Next() {
		var m store.Model
		var query string
		var collID sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &collID, &m.DataSourceID, &query, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		m.CollectionID = collID.String
		fromJSON(query, &m.Query)
		result = append(result, &m)
	}
	return result, rows.Err()
}

func (s *ModelStore) Update(ctx context.Context, m *store.Model) error {
	m.UpdatedAt = time.Now()

	var collID interface{}
	if m.CollectionID != "" {
		collID = m.CollectionID
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE models SET name=?, description=?, collection_id=?, query=?, updated_at=?
		WHERE id=?
	`, m.Name, m.Description, collID, toJSON(m.Query), m.UpdatedAt, m.ID)
	return err
}

func (s *ModelStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM models WHERE id=?`, id)
	return err
}

func (s *ModelStore) CreateColumn(ctx context.Context, col *store.ModelColumn) error {
	if col.ID == "" {
		col.ID = generateID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO model_columns (id, model_id, name, display_name, description, semantic, visible)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, col.ID, col.ModelID, col.Name, col.DisplayName, col.Description, col.Semantic, col.Visible)
	return err
}

func (s *ModelStore) ListColumns(ctx context.Context, modelID string) ([]*store.ModelColumn, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, model_id, name, display_name, description, semantic, visible
		FROM model_columns WHERE model_id = ?
	`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.ModelColumn
	for rows.Next() {
		var c store.ModelColumn
		if err := rows.Scan(&c.ID, &c.ModelID, &c.Name, &c.DisplayName, &c.Description, &c.Semantic, &c.Visible); err != nil {
			return nil, err
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *ModelStore) UpdateColumn(ctx context.Context, col *store.ModelColumn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE model_columns SET display_name=?, description=?, semantic=?, visible=?
		WHERE id=?
	`, col.DisplayName, col.Description, col.Semantic, col.Visible, col.ID)
	return err
}
