package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// MetricStore implements store.MetricStore.
type MetricStore struct {
	db *sql.DB
}

func (s *MetricStore) Create(ctx context.Context, m *store.Metric) error {
	if m.ID == "" {
		m.ID = generateID()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO metrics (id, name, description, table_id, definition, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.Name, m.Description, m.TableID, toJSON(m.Definition), m.CreatedBy, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *MetricStore) GetByID(ctx context.Context, id string) (*store.Metric, error) {
	var m store.Metric
	var def string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, table_id, definition, created_by, created_at, updated_at
		FROM metrics WHERE id = ?
	`, id).Scan(&m.ID, &m.Name, &m.Description, &m.TableID, &def, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fromJSON(def, &m.Definition)
	return &m, nil
}

func (s *MetricStore) List(ctx context.Context) ([]*store.Metric, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, table_id, definition, created_by, created_at, updated_at
		FROM metrics ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Metric
	for rows.Next() {
		var m store.Metric
		var def string
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.TableID, &def, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		fromJSON(def, &m.Definition)
		result = append(result, &m)
	}
	return result, rows.Err()
}

func (s *MetricStore) ListByTable(ctx context.Context, tableID string) ([]*store.Metric, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, table_id, definition, created_by, created_at, updated_at
		FROM metrics WHERE table_id = ? ORDER BY name
	`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Metric
	for rows.Next() {
		var m store.Metric
		var def string
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.TableID, &def, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		fromJSON(def, &m.Definition)
		result = append(result, &m)
	}
	return result, rows.Err()
}

func (s *MetricStore) Update(ctx context.Context, m *store.Metric) error {
	m.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE metrics SET name=?, description=?, definition=?, updated_at=?
		WHERE id=?
	`, m.Name, m.Description, toJSON(m.Definition), m.UpdatedAt, m.ID)
	return err
}

func (s *MetricStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM metrics WHERE id=?`, id)
	return err
}
