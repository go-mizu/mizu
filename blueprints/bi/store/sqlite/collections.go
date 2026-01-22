package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// CollectionStore implements store.CollectionStore.
type CollectionStore struct {
	db *sql.DB
}

func (s *CollectionStore) Create(ctx context.Context, c *store.Collection) error {
	if c.ID == "" {
		c.ID = generateID()
	}
	c.CreatedAt = time.Now()

	var parentID interface{}
	if c.ParentID != "" {
		parentID = c.ParentID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO collections (id, name, description, parent_id, color, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.Name, c.Description, parentID, c.Color, c.CreatedBy, c.CreatedAt)
	return err
}

func (s *CollectionStore) GetByID(ctx context.Context, id string) (*store.Collection, error) {
	var c store.Collection
	var parentID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, parent_id, color, created_by, created_at
		FROM collections WHERE id = ?
	`, id).Scan(&c.ID, &c.Name, &c.Description, &parentID, &c.Color, &c.CreatedBy, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ParentID = parentID.String
	return &c, nil
}

func (s *CollectionStore) List(ctx context.Context) ([]*store.Collection, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, parent_id, color, created_by, created_at
		FROM collections ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Collection
	for rows.Next() {
		var c store.Collection
		var parentID sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &parentID, &c.Color, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.ParentID = parentID.String
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *CollectionStore) ListByParent(ctx context.Context, parentID string) ([]*store.Collection, error) {
	var rows *sql.Rows
	var err error
	if parentID == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, description, parent_id, color, created_by, created_at
			FROM collections WHERE parent_id IS NULL ORDER BY name
		`)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, description, parent_id, color, created_by, created_at
			FROM collections WHERE parent_id = ? ORDER BY name
		`, parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Collection
	for rows.Next() {
		var c store.Collection
		var pID sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &pID, &c.Color, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.ParentID = pID.String
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *CollectionStore) Update(ctx context.Context, c *store.Collection) error {
	var parentID interface{}
	if c.ParentID != "" {
		parentID = c.ParentID
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE collections SET name=?, description=?, parent_id=?, color=?
		WHERE id=?
	`, c.Name, c.Description, parentID, c.Color, c.ID)
	return err
}

func (s *CollectionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE id=?`, id)
	return err
}
