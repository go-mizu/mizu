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
	t.Visible = true // Default to visible

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tables (id, datasource_id, schema_name, name, display_name, description, visible, field_order, row_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.DataSourceID, t.Schema, t.Name, t.DisplayName, t.Description, t.Visible, t.FieldOrder, t.RowCount, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *TableStore) GetByID(ctx context.Context, id string) (*store.Table, error) {
	var t store.Table
	err := s.db.QueryRowContext(ctx, `
		SELECT id, datasource_id, schema_name, name, display_name, description,
			COALESCE(visible, 1), COALESCE(field_order, ''), row_count, created_at, updated_at
		FROM tables WHERE id = ?
	`, id).Scan(&t.ID, &t.DataSourceID, &t.Schema, &t.Name, &t.DisplayName, &t.Description,
		&t.Visible, &t.FieldOrder, &t.RowCount, &t.CreatedAt, &t.UpdatedAt)
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
		SELECT id, datasource_id, schema_name, name, display_name, description,
			COALESCE(visible, 1), COALESCE(field_order, ''), row_count, created_at, updated_at
		FROM tables WHERE datasource_id = ? ORDER BY name
	`, dsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Table
	for rows.Next() {
		var t store.Table
		if err := rows.Scan(&t.ID, &t.DataSourceID, &t.Schema, &t.Name, &t.DisplayName, &t.Description,
			&t.Visible, &t.FieldOrder, &t.RowCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &t)
	}
	return result, rows.Err()
}

func (s *TableStore) Update(ctx context.Context, t *store.Table) error {
	t.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE tables SET display_name=?, description=?, visible=?, field_order=?, row_count=?, updated_at=?
		WHERE id=?
	`, t.DisplayName, t.Description, t.Visible, t.FieldOrder, t.RowCount, t.UpdatedAt, t.ID)
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
	if col.Visibility == "" {
		col.Visibility = "everywhere"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO columns (
			id, table_id, name, display_name, type, mapped_type, semantic, description, position,
			visibility, nullable, primary_key, foreign_key, foreign_table, foreign_column,
			distinct_count, null_count, min_value, max_value, avg_length, cached_values, values_cached_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		col.ID, col.TableID, col.Name, col.DisplayName, col.Type, col.MappedType, col.Semantic, col.Description, col.Position,
		col.Visibility, col.Nullable, col.PrimaryKey, col.ForeignKey, col.ForeignTable, col.ForeignColumn,
		col.DistinctCount, col.NullCount, col.MinValue, col.MaxValue, col.AvgLength, toJSON(col.CachedValues), col.ValuesCachedAt,
	)
	return err
}

func (s *TableStore) ListColumns(ctx context.Context, tableID string) ([]*store.Column, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, table_id, name, COALESCE(display_name, name), type, COALESCE(mapped_type, ''), COALESCE(semantic, ''),
			COALESCE(description, ''), position, COALESCE(visibility, 'everywhere'),
			COALESCE(nullable, 1), COALESCE(primary_key, 0), COALESCE(foreign_key, 0),
			COALESCE(foreign_table, ''), COALESCE(foreign_column, ''),
			COALESCE(distinct_count, 0), COALESCE(null_count, 0), COALESCE(min_value, ''), COALESCE(max_value, ''),
			COALESCE(avg_length, 0), COALESCE(cached_values, '[]'), values_cached_at
		FROM columns WHERE table_id = ? ORDER BY position
	`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Column
	for rows.Next() {
		var c store.Column
		var cachedValues string
		var valuesCachedAt sql.NullTime
		if err := rows.Scan(
			&c.ID, &c.TableID, &c.Name, &c.DisplayName, &c.Type, &c.MappedType, &c.Semantic,
			&c.Description, &c.Position, &c.Visibility,
			&c.Nullable, &c.PrimaryKey, &c.ForeignKey, &c.ForeignTable, &c.ForeignColumn,
			&c.DistinctCount, &c.NullCount, &c.MinValue, &c.MaxValue, &c.AvgLength, &cachedValues, &valuesCachedAt,
		); err != nil {
			return nil, err
		}
		fromJSON(cachedValues, &c.CachedValues)
		if valuesCachedAt.Valid {
			c.ValuesCachedAt = &valuesCachedAt.Time
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
	var cachedValues string
	var valuesCachedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, table_id, name, COALESCE(display_name, name), type, COALESCE(mapped_type, ''), COALESCE(semantic, ''),
			COALESCE(description, ''), position, COALESCE(visibility, 'everywhere'),
			COALESCE(nullable, 1), COALESCE(primary_key, 0), COALESCE(foreign_key, 0),
			COALESCE(foreign_table, ''), COALESCE(foreign_column, ''),
			COALESCE(distinct_count, 0), COALESCE(null_count, 0), COALESCE(min_value, ''), COALESCE(max_value, ''),
			COALESCE(avg_length, 0), COALESCE(cached_values, '[]'), values_cached_at
		FROM columns WHERE id = ?
	`, id).Scan(
		&c.ID, &c.TableID, &c.Name, &c.DisplayName, &c.Type, &c.MappedType, &c.Semantic,
		&c.Description, &c.Position, &c.Visibility,
		&c.Nullable, &c.PrimaryKey, &c.ForeignKey, &c.ForeignTable, &c.ForeignColumn,
		&c.DistinctCount, &c.NullCount, &c.MinValue, &c.MaxValue, &c.AvgLength, &cachedValues, &valuesCachedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fromJSON(cachedValues, &c.CachedValues)
	if valuesCachedAt.Valid {
		c.ValuesCachedAt = &valuesCachedAt.Time
	}
	return &c, nil
}

func (s *TableStore) UpdateColumn(ctx context.Context, col *store.Column) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE columns SET
			display_name=?, description=?, semantic=?, visibility=?,
			distinct_count=?, null_count=?, min_value=?, max_value=?, avg_length=?,
			cached_values=?, values_cached_at=?
		WHERE id=?
	`,
		col.DisplayName, col.Description, col.Semantic, col.Visibility,
		col.DistinctCount, col.NullCount, col.MinValue, col.MaxValue, col.AvgLength,
		toJSON(col.CachedValues), col.ValuesCachedAt, col.ID,
	)
	return err
}
