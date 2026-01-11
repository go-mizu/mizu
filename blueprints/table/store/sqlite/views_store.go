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
// Pre-allocates slice with expected capacity to avoid multiple allocations.
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

	// Pre-allocate with typical capacity (most tables have 3-10 views)
	viewList := make([]*views.View, 0, 8)
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
	view.Filters = unmarshalJSONSlice[views.Filter](filters.String)
	view.Sorts = unmarshalJSONSlice[views.SortSpec](sorts.String)
	view.Groups = unmarshalJSONSlice[views.GroupSpec](groups.String)
	view.FieldConfig = unmarshalJSONSlice[views.FieldViewConfig](fieldConfig.String)

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
	view.Filters = unmarshalJSONSlice[views.Filter](filters.String)
	view.Sorts = unmarshalJSONSlice[views.SortSpec](sorts.String)
	view.Groups = unmarshalJSONSlice[views.GroupSpec](groups.String)
	view.FieldConfig = unmarshalJSONSlice[views.FieldViewConfig](fieldConfig.String)

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
func unmarshalJSONSlice[T any](jsonStr string) []T {
	if jsonStr == "" {
		return nil
	}
	var result []T
	json.Unmarshal([]byte(jsonStr), &result)
	return result
}

// UpdatePositionsBatch updates multiple view positions in a single query.
// This is more efficient than calling Update multiple times for view reordering.
func (s *ViewsStore) UpdatePositionsBatch(ctx context.Context, viewPositions map[string]int) error {
	if len(viewPositions) == 0 {
		return nil
	}

	now := time.Now()

	// Use a transaction for atomicity and better performance
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare statement for reuse
	stmt, err := tx.PrepareContext(ctx, `UPDATE views SET position = ?, updated_at = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for viewID, position := range viewPositions {
		if _, err := stmt.ExecContext(ctx, position, now, viewID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateFilters updates only the filters column without fetching the full view.
// This avoids the read-modify-write pattern for better performance.
func (s *ViewsStore) UpdateFilters(ctx context.Context, viewID string, filters []views.Filter) error {
	now := time.Now()
	filtersStr := marshalJSON(filters)
	_, err := s.db.ExecContext(ctx, `UPDATE views SET filters = ?, updated_at = ? WHERE id = ?`, filtersStr, now, viewID)
	return err
}

// UpdateSorts updates only the sorts column without fetching the full view.
func (s *ViewsStore) UpdateSorts(ctx context.Context, viewID string, sorts []views.SortSpec) error {
	now := time.Now()
	sortsStr := marshalJSON(sorts)
	_, err := s.db.ExecContext(ctx, `UPDATE views SET sorts = ?, updated_at = ? WHERE id = ?`, sortsStr, now, viewID)
	return err
}

// UpdateGroups updates only the groups column without fetching the full view.
func (s *ViewsStore) UpdateGroups(ctx context.Context, viewID string, groups []views.GroupSpec) error {
	now := time.Now()
	groupsStr := marshalJSON(groups)
	_, err := s.db.ExecContext(ctx, `UPDATE views SET groups = ?, updated_at = ? WHERE id = ?`, groupsStr, now, viewID)
	return err
}

// UpdateFieldConfig updates only the field_config column without fetching the full view.
func (s *ViewsStore) UpdateFieldConfig(ctx context.Context, viewID string, fieldConfig []views.FieldViewConfig) error {
	now := time.Now()
	fieldConfigStr := marshalJSON(fieldConfig)
	_, err := s.db.ExecContext(ctx, `UPDATE views SET field_config = ?, updated_at = ? WHERE id = ?`, fieldConfigStr, now, viewID)
	return err
}
