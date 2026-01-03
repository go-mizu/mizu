package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"

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
	if in.Name != nil {
		_, err := s.db.ExecContext(ctx, "UPDATE views SET name = ? WHERE id = ?", *in.Name, id)
		if err != nil {
			return err
		}
	}
	if in.Filter != nil {
		filterJSON, _ := json.Marshal(in.Filter)
		_, err := s.db.ExecContext(ctx, "UPDATE views SET filter = ? WHERE id = ?", string(filterJSON), id)
		if err != nil {
			return err
		}
	}
	if len(in.Sorts) > 0 {
		sortsJSON, _ := json.Marshal(in.Sorts)
		_, err := s.db.ExecContext(ctx, "UPDATE views SET sorts = ? WHERE id = ?", string(sortsJSON), id)
		if err != nil {
			return err
		}
	}
	if len(in.Properties) > 0 {
		propsJSON, _ := json.Marshal(in.Properties)
		_, err := s.db.ExecContext(ctx, "UPDATE views SET properties = ? WHERE id = ?", string(propsJSON), id)
		if err != nil {
			return err
		}
	}
	if in.GroupBy != nil {
		_, err := s.db.ExecContext(ctx, "UPDATE views SET group_by = ? WHERE id = ?", *in.GroupBy, id)
		if err != nil {
			return err
		}
	}
	if in.CalendarBy != nil {
		_, err := s.db.ExecContext(ctx, "UPDATE views SET calendar_by = ? WHERE id = ?", *in.CalendarBy, id)
		if err != nil {
			return err
		}
	}
	return nil
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
	for i, id := range viewIDs {
		_, err := s.db.ExecContext(ctx, "UPDATE views SET position = ? WHERE id = ?", i, id)
		if err != nil {
			return err
		}
	}
	return nil
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
