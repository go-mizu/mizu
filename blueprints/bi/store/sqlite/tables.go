package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// TableStore implements store.TableStore.
type TableStore struct {
	db *sql.DB
}

func (s *TableStore) Create(ctx context.Context, t *store.Table) error {
	if t.ID == "" {
		t.ID = generateID()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tables (id, datasource_id, schema_name, name, display_name, description, row_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.DataSourceID, t.Schema, t.Name, t.DisplayName, t.Description, t.RowCount, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *TableStore) GetByID(ctx context.Context, id string) (*store.Table, error) {
	var t store.Table
	err := s.db.QueryRowContext(ctx, `
		SELECT id, datasource_id, schema_name, name, display_name, description, row_count, created_at, updated_at
		FROM tables WHERE id = ?
	`, id).Scan(&t.ID, &t.DataSourceID, &t.Schema, &t.Name, &t.DisplayName, &t.Description, &t.RowCount, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *TableStore) ListByDataSource(ctx context.Context, dsID string) ([]*store.Table, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, datasource_id, schema_name, name, display_name, description, row_count, created_at, updated_at
		FROM tables WHERE datasource_id = ? ORDER BY name
	`, dsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Table
	for rows.Next() {
		var t store.Table
		if err := rows.Scan(&t.ID, &t.DataSourceID, &t.Schema, &t.Name, &t.DisplayName, &t.Description, &t.RowCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &t)
	}
	return result, rows.Err()
}

func (s *TableStore) Update(ctx context.Context, t *store.Table) error {
	t.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE tables SET display_name=?, description=?, row_count=?, updated_at=?
		WHERE id=?
	`, t.DisplayName, t.Description, t.RowCount, t.UpdatedAt, t.ID)
	return err
}

func (s *TableStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tables WHERE id=?`, id)
	return err
}

func (s *TableStore) CreateColumn(ctx context.Context, col *store.Column) error {
	if col.ID == "" {
		col.ID = generateID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO columns (id, table_id, name, display_name, type, semantic, description, position)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, col.ID, col.TableID, col.Name, col.DisplayName, col.Type, col.Semantic, col.Description, col.Position)
	return err
}

func (s *TableStore) ListColumns(ctx context.Context, tableID string) ([]*store.Column, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, table_id, name, display_name, type, semantic, description, position
		FROM columns WHERE table_id = ? ORDER BY position
	`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Column
	for rows.Next() {
		var c store.Column
		if err := rows.Scan(&c.ID, &c.TableID, &c.Name, &c.DisplayName, &c.Type, &c.Semantic, &c.Description, &c.Position); err != nil {
			return nil, err
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *TableStore) DeleteColumnsByTable(ctx context.Context, tableID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM columns WHERE table_id=?`, tableID)
	return err
}

func (s *TableStore) GetColumn(ctx context.Context, id string) (*store.Column, error) {
	var c store.Column
	err := s.db.QueryRowContext(ctx, `
		SELECT id, table_id, name, display_name, type, semantic, description, position
		FROM columns WHERE id = ?
	`, id).Scan(&c.ID, &c.TableID, &c.Name, &c.DisplayName, &c.Type, &c.Semantic, &c.Description, &c.Position)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *TableStore) UpdateColumn(ctx context.Context, col *store.Column) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE columns SET display_name=?, description=?, semantic=?
		WHERE id=?
	`, col.DisplayName, col.Description, col.Semantic, col.ID)
	return err
}
