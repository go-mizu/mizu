package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/go-mizu/blueprints/workspace/feature/views"
)

// ViewsStore implements views.Store.
type ViewsStore struct {
	db *sql.DB
}

// NewViewsStore creates a new ViewsStore.
func NewViewsStore(db *sql.DB) *ViewsStore {
	return &ViewsStore{db: db}
}

func (s *ViewsStore) Create(ctx context.Context, v *views.View) error {
	filterJSON, _ := json.Marshal(v.Filter)
	sortsJSON, _ := json.Marshal(v.Sorts)
	propsJSON, _ := json.Marshal(v.Properties)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO views (id, database_id, name, type, filter, sorts, properties, group_by, calendar_by, position, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, v.ID, v.DatabaseID, v.Name, v.Type, string(filterJSON), string(sortsJSON), string(propsJSON), v.GroupBy, v.CalendarBy, v.Position, v.CreatedBy, v.CreatedAt)
	return err
}

func (s *ViewsStore) GetByID(ctx context.Context, id string) (*views.View, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, database_id, name, type, CAST(filter AS VARCHAR), CAST(sorts AS VARCHAR), CAST(properties AS VARCHAR), group_by, calendar_by, position, created_by, created_at
		FROM views WHERE id = ?
	`, id)
	return s.scanView(row)
}

func (s *ViewsStore) Update(ctx context.Context, id string, in *views.UpdateIn) error {
	// Build a single UPDATE with all fields to avoid multiple round-trips
	sets := []string{}
	args := []interface{}{}

	if in.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *in.Name)
	}
	if in.Filter != nil {
		filterJSON, _ := json.Marshal(in.Filter)
		sets = append(sets, "filter = ?")
		args = append(args, string(filterJSON))
	}
	if len(in.Sorts) > 0 {
		sortsJSON, _ := json.Marshal(in.Sorts)
		sets = append(sets, "sorts = ?")
		args = append(args, string(sortsJSON))
	}
	if len(in.Properties) > 0 {
		propsJSON, _ := json.Marshal(in.Properties)
		sets = append(sets, "properties = ?")
		args = append(args, string(propsJSON))
	}
	if in.GroupBy != nil {
		sets = append(sets, "group_by = ?")
		args = append(args, *in.GroupBy)
	}
	if in.CalendarBy != nil {
		sets = append(sets, "calendar_by = ?")
		args = append(args, *in.CalendarBy)
	}

	// Only execute if there are actual changes
	if len(sets) == 0 {
		return nil
	}

	args = append(args, id)
	query := "UPDATE views SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *ViewsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM views WHERE id = ?", id)
	return err
}

func (s *ViewsStore) ListByDatabase(ctx context.Context, databaseID string) ([]*views.View, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, database_id, name, type, CAST(filter AS VARCHAR), CAST(sorts AS VARCHAR), CAST(properties AS VARCHAR), group_by, calendar_by, position, created_by, created_at
		FROM views WHERE database_id = ?
		ORDER BY position
	`, databaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanViews(rows)
}

func (s *ViewsStore) Reorder(ctx context.Context, databaseID string, viewIDs []string) error {
	if len(viewIDs) == 0 {
		return nil
	}

	// Build batch UPDATE with CASE statement to avoid N individual updates
	var caseBuilder strings.Builder
	args := make([]interface{}, 0, len(viewIDs)*2+len(viewIDs))

	caseBuilder.WriteString("UPDATE views SET position = CASE id ")
	for i, id := range viewIDs {
		caseBuilder.WriteString("WHEN ? THEN ? ")
		args = append(args, id, i)
	}
	caseBuilder.WriteString("END WHERE id IN (")

	placeholders := make([]string, len(viewIDs))
	for i, id := range viewIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	caseBuilder.WriteString(strings.Join(placeholders, ", "))
	caseBuilder.WriteString(")")

	_, err := s.db.ExecContext(ctx, caseBuilder.String(), args...)
	return err
}

func (s *ViewsStore) scanView(row *sql.Row) (*views.View, error) {
	var v views.View
	var filterJSON, sortsJSON, propsJSON sql.NullString
	err := row.Scan(&v.ID, &v.DatabaseID, &v.Name, &v.Type, &filterJSON, &sortsJSON, &propsJSON, &v.GroupBy, &v.CalendarBy, &v.Position, &v.CreatedBy, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	if filterJSON.Valid {
		json.Unmarshal([]byte(filterJSON.String), &v.Filter)
	}
	if sortsJSON.Valid {
		json.Unmarshal([]byte(sortsJSON.String), &v.Sorts)
	}
	if propsJSON.Valid {
		json.Unmarshal([]byte(propsJSON.String), &v.Properties)
	}
	return &v, nil
}

func (s *ViewsStore) scanViews(rows *sql.Rows) ([]*views.View, error) {
	var result []*views.View
	for rows.Next() {
		var v views.View
		var filterJSON, sortsJSON, propsJSON sql.NullString
		err := rows.Scan(&v.ID, &v.DatabaseID, &v.Name, &v.Type, &filterJSON, &sortsJSON, &propsJSON, &v.GroupBy, &v.CalendarBy, &v.Position, &v.CreatedBy, &v.CreatedAt)
		if err != nil {
			return nil, err
		}
		if filterJSON.Valid {
			json.Unmarshal([]byte(filterJSON.String), &v.Filter)
		}
		if sortsJSON.Valid {
			json.Unmarshal([]byte(sortsJSON.String), &v.Sorts)
		}
		if propsJSON.Valid {
			json.Unmarshal([]byte(propsJSON.String), &v.Properties)
		}
		result = append(result, &v)
	}
	return result, rows.Err()
}
