package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/table/feature/views"
)

// ViewsStore provides SQLite-based view storage.
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
	s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM views WHERE table_id = ?`, view.TableID).Scan(&maxPos)
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, view.ID, view.TableID, view.Name, view.Type, configStr, filtersStr, sortsStr, groupsStr, fieldConfigStr, view.Position, view.IsDefault, view.IsLocked, view.CreatedBy, view.CreatedAt, view.UpdatedAt)
	return err
}

// GetByID retrieves a view by ID.
func (s *ViewsStore) GetByID(ctx context.Context, id string) (*views.View, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, table_id, name, type, config, filters, sorts, groups, field_config, position, is_default, is_locked, created_by, created_at, updated_at
		FROM views WHERE id = ?
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
			name = ?, type = ?, config = ?, filters = ?, sorts = ?, groups = ?, field_config = ?, position = ?, is_default = ?, is_locked = ?, updated_at = ?
		WHERE id = ?
	`, view.Name, view.Type, configStr, filtersStr, sortsStr, groupsStr, fieldConfigStr, view.Position, view.IsDefault, view.IsLocked, view.UpdatedAt, view.ID)
	return err
}

// Delete deletes a view.
func (s *ViewsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM views WHERE id = ?`, id)
	return err
}

// ListByTable lists all views in a table.
func (s *ViewsStore) ListByTable(ctx context.Context, tableID string) ([]*views.View, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, table_id, name, type, config, filters, sorts, groups, field_config, position, is_default, is_locked, created_by, created_at, updated_at
		FROM views WHERE table_id = ?
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
	var config, filters, sorts, groups, fieldConfig sql.NullString

	err := row.Scan(&view.ID, &view.TableID, &view.Name, &view.Type, &config, &filters, &sorts, &groups, &fieldConfig, &view.Position, &view.IsDefault, &view.IsLocked, &view.CreatedBy, &view.CreatedAt, &view.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, views.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if config.Valid {
		view.Config = json.RawMessage(config.String)
	}
	view.Filters = unmarshalJSONSlice[views.Filter](config.String, filters.String)
	view.Sorts = unmarshalJSONSlice[views.SortSpec](config.String, sorts.String)
	view.Groups = unmarshalJSONSlice[views.GroupSpec](config.String, groups.String)
	view.FieldConfig = unmarshalJSONSlice[views.FieldViewConfig](config.String, fieldConfig.String)

	return view, nil
}

func (s *ViewsStore) scanViewRows(rows *sql.Rows) (*views.View, error) {
	view := &views.View{}
	var config, filters, sorts, groups, fieldConfig sql.NullString

	err := rows.Scan(&view.ID, &view.TableID, &view.Name, &view.Type, &config, &filters, &sorts, &groups, &fieldConfig, &view.Position, &view.IsDefault, &view.IsLocked, &view.CreatedBy, &view.CreatedAt, &view.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if config.Valid {
		view.Config = json.RawMessage(config.String)
	}
	view.Filters = unmarshalJSONSlice[views.Filter](config.String, filters.String)
	view.Sorts = unmarshalJSONSlice[views.SortSpec](config.String, sorts.String)
	view.Groups = unmarshalJSONSlice[views.GroupSpec](config.String, groups.String)
	view.FieldConfig = unmarshalJSONSlice[views.FieldViewConfig](config.String, fieldConfig.String)

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

// unmarshalJSONSlice handles SQLite TEXT columns storing JSON arrays
func unmarshalJSONSlice[T any](_, jsonStr string) []T {
	if jsonStr == "" {
		return nil
	}
	var result []T
	json.Unmarshal([]byte(jsonStr), &result)
	return result
}
