package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/table/feature/views"
)

// ViewsStore provides PostgreSQL-based view storage.
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

	configJSON := marshalJSONB(view.Config)
	filtersJSON := marshalJSONB(view.Filters)
	sortsJSON := marshalJSONB(view.Sorts)
	groupsJSON := marshalJSONB(view.Groups)
	fieldConfigJSON := marshalJSONB(view.FieldConfig)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO views (id, table_id, name, type, config, filters, sorts, groups, field_config, position, is_default, is_locked, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, view.ID, view.TableID, view.Name, view.Type, configJSON, filtersJSON, sortsJSON, groupsJSON, fieldConfigJSON, view.Position, view.IsDefault, view.IsLocked, view.CreatedBy, view.CreatedAt, view.UpdatedAt)
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

	configJSON := marshalJSONB(view.Config)
	filtersJSON := marshalJSONB(view.Filters)
	sortsJSON := marshalJSONB(view.Sorts)
	groupsJSON := marshalJSONB(view.Groups)
	fieldConfigJSON := marshalJSONB(view.FieldConfig)

	_, err := s.db.ExecContext(ctx, `
		UPDATE views SET
			name = $1, type = $2, config = $3, filters = $4, sorts = $5, groups = $6, field_config = $7, position = $8, is_default = $9, is_locked = $10, updated_at = $11
		WHERE id = $12
	`, view.Name, view.Type, configJSON, filtersJSON, sortsJSON, groupsJSON, fieldConfigJSON, view.Position, view.IsDefault, view.IsLocked, view.UpdatedAt, view.ID)
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
	var configJSON, filtersJSON, sortsJSON, groupsJSON, fieldConfigJSON []byte

	err := row.Scan(&view.ID, &view.TableID, &view.Name, &view.Type, &configJSON, &filtersJSON, &sortsJSON, &groupsJSON, &fieldConfigJSON, &view.Position, &view.IsDefault, &view.IsLocked, &view.CreatedBy, &view.CreatedAt, &view.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, views.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	view.Config = unmarshalJSONBRaw(configJSON)
	view.Filters = unmarshalJSONBSlice[views.Filter](filtersJSON)
	view.Sorts = unmarshalJSONBSlice[views.SortSpec](sortsJSON)
	view.Groups = unmarshalJSONBSlice[views.GroupSpec](groupsJSON)
	view.FieldConfig = unmarshalJSONBSlice[views.FieldViewConfig](fieldConfigJSON)

	return view, nil
}

func (s *ViewsStore) scanViewRows(rows *sql.Rows) (*views.View, error) {
	view := &views.View{}
	var configJSON, filtersJSON, sortsJSON, groupsJSON, fieldConfigJSON []byte

	err := rows.Scan(&view.ID, &view.TableID, &view.Name, &view.Type, &configJSON, &filtersJSON, &sortsJSON, &groupsJSON, &fieldConfigJSON, &view.Position, &view.IsDefault, &view.IsLocked, &view.CreatedBy, &view.CreatedAt, &view.UpdatedAt)
	if err != nil {
		return nil, err
	}

	view.Config = unmarshalJSONBRaw(configJSON)
	view.Filters = unmarshalJSONBSlice[views.Filter](filtersJSON)
	view.Sorts = unmarshalJSONBSlice[views.SortSpec](sortsJSON)
	view.Groups = unmarshalJSONBSlice[views.GroupSpec](groupsJSON)
	view.FieldConfig = unmarshalJSONBSlice[views.FieldViewConfig](fieldConfigJSON)

	return view, nil
}

// marshalJSONB marshals a value to JSONB bytes for PostgreSQL.
func marshalJSONB(v any) []byte {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

// unmarshalJSONBRaw returns the raw JSON bytes as json.RawMessage.
func unmarshalJSONBRaw(b []byte) json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	return json.RawMessage(b)
}

// unmarshalJSONBSlice unmarshals JSONB bytes to a typed slice.
func unmarshalJSONBSlice[T any](b []byte) []T {
	if len(b) == 0 {
		return nil
	}
	var result []T
	json.Unmarshal(b, &result)
	return result
}

// unmarshalJSONBMap unmarshals JSONB bytes to a typed map.
func unmarshalJSONBMap[T any](b []byte) map[string]T {
	if len(b) == 0 {
		return nil
	}
	var result map[string]T
	json.Unmarshal(b, &result)
	return result
}
