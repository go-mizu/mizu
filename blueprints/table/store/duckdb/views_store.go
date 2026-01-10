package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/table/feature/views"
)

// ViewsStore provides DuckDB-based view storage.
type ViewsStore struct {
	db *sql.DB
}

// NewViewsStore creates a new views store.
func NewViewsStore(db *sql.DB) *ViewsStore {
	return &ViewsStore{db: db}
}

// Create creates a new view.
func (s *ViewsStore) Create(ctx context.Context, view *views.View) error {
	now := time.Now()
	view.CreatedAt = now
	view.UpdatedAt = now

	if view.Type == "" {
		view.Type = views.TypeGrid
	}

	// Get max position
	var maxPos sql.NullInt64
	s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM views WHERE table_id = $1`, view.TableID).Scan(&maxPos)
	if maxPos.Valid {
		view.Position = int(maxPos.Int64) + 1
	}

	configStr := marshalJSON(view.Config)
	filtersStr := marshalJSON(view.Filters)
	sortsStr := marshalJSON(view.Sorts)
	groupsStr := marshalJSON(view.Groups)
	fieldConfigStr := marshalJSON(view.FieldConfig)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO views (id, table_id, name, type, config, filters, sorts, groups, field_config, position, is_default, is_locked, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, view.ID, view.TableID, view.Name, view.Type, configStr, filtersStr, sortsStr, groupsStr, fieldConfigStr, view.Position, view.IsDefault, view.IsLocked, view.CreatedBy, view.CreatedAt, view.UpdatedAt)
	return err
}

// GetByID retrieves a view by ID.
func (s *ViewsStore) GetByID(ctx context.Context, id string) (*views.View, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, table_id, name, type, config, filters, sorts, groups, field_config, position, is_default, is_locked, created_by, created_at, updated_at
		FROM views WHERE id = $1
	`, id)
	return s.scanView(row)
}

// Update updates a view.
func (s *ViewsStore) Update(ctx context.Context, view *views.View) error {
	view.UpdatedAt = time.Now()

	configStr := marshalJSON(view.Config)
	filtersStr := marshalJSON(view.Filters)
	sortsStr := marshalJSON(view.Sorts)
	groupsStr := marshalJSON(view.Groups)
	fieldConfigStr := marshalJSON(view.FieldConfig)

	_, err := s.db.ExecContext(ctx, `
		UPDATE views SET
			name = $1, type = $2, config = $3, filters = $4, sorts = $5, groups = $6, field_config = $7, position = $8, is_default = $9, is_locked = $10, updated_at = $11
		WHERE id = $12
	`, view.Name, view.Type, configStr, filtersStr, sortsStr, groupsStr, fieldConfigStr, view.Position, view.IsDefault, view.IsLocked, view.UpdatedAt, view.ID)
	return err
}

// Delete deletes a view.
func (s *ViewsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM views WHERE id = $1`, id)
	return err
}

// ListByTable lists all views in a table.
func (s *ViewsStore) ListByTable(ctx context.Context, tableID string) ([]*views.View, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, table_id, name, type, config, filters, sorts, groups, field_config, position, is_default, is_locked, created_by, created_at, updated_at
		FROM views WHERE table_id = $1
		ORDER BY position ASC
	`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var viewList []*views.View
	for rows.Next() {
		view, err := s.scanViewRows(rows)
		if err != nil {
			return nil, err
		}
		viewList = append(viewList, view)
	}
	return viewList, rows.Err()
}

func (s *ViewsStore) scanView(row *sql.Row) (*views.View, error) {
	view := &views.View{}
	var config, filters, sorts, groups, fieldConfig any

	err := row.Scan(&view.ID, &view.TableID, &view.Name, &view.Type, &config, &filters, &sorts, &groups, &fieldConfig, &view.Position, &view.IsDefault, &view.IsLocked, &view.CreatedBy, &view.CreatedAt, &view.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, views.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	view.Config = unmarshalJSONField(config)
	view.Filters = unmarshalJSONSlice[views.Filter](filters)
	view.Sorts = unmarshalJSONSlice[views.SortSpec](sorts)
	view.Groups = unmarshalJSONSlice[views.GroupSpec](groups)
	view.FieldConfig = unmarshalJSONSlice[views.FieldViewConfig](fieldConfig)

	return view, nil
}

func (s *ViewsStore) scanViewRows(rows *sql.Rows) (*views.View, error) {
	view := &views.View{}
	var config, filters, sorts, groups, fieldConfig any

	err := rows.Scan(&view.ID, &view.TableID, &view.Name, &view.Type, &config, &filters, &sorts, &groups, &fieldConfig, &view.Position, &view.IsDefault, &view.IsLocked, &view.CreatedBy, &view.CreatedAt, &view.UpdatedAt)
	if err != nil {
		return nil, err
	}

	view.Config = unmarshalJSONField(config)
	view.Filters = unmarshalJSONSlice[views.Filter](filters)
	view.Sorts = unmarshalJSONSlice[views.SortSpec](sorts)
	view.Groups = unmarshalJSONSlice[views.GroupSpec](groups)
	view.FieldConfig = unmarshalJSONSlice[views.FieldViewConfig](fieldConfig)

	return view, nil
}

func marshalJSON(v any) *string {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

// unmarshalJSONField handles DuckDB JSON columns that can be map, string, or nil
func unmarshalJSONField(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		return json.RawMessage(val)
	case map[string]any:
		b, _ := json.Marshal(val)
		return b
	case []any:
		b, _ := json.Marshal(val)
		return b
	default:
		return nil
	}
}

// unmarshalJSONSlice handles DuckDB JSON array columns
func unmarshalJSONSlice[T any](v any) []T {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		var result []T
		json.Unmarshal([]byte(val), &result)
		return result
	case []any:
		b, _ := json.Marshal(val)
		var result []T
		json.Unmarshal(b, &result)
		return result
	default:
		return nil
	}
}

// unmarshalJSONMap handles DuckDB JSON object columns
func unmarshalJSONMap[T any](v any) map[string]T {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		var result map[string]T
		json.Unmarshal([]byte(val), &result)
		return result
	case map[string]any:
		b, _ := json.Marshal(val)
		var result map[string]T
		json.Unmarshal(b, &result)
		return result
	default:
		return nil
	}
}
