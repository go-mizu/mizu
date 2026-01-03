package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-mizu/blueprints/workspace/feature/databases"
)

// DatabasesStore implements databases.Store.
type DatabasesStore struct {
	db *sql.DB
}

// NewDatabasesStore creates a new DatabasesStore.
func NewDatabasesStore(db *sql.DB) *DatabasesStore {
	return &DatabasesStore{db: db}
}

func (s *DatabasesStore) Create(ctx context.Context, d *databases.Database) error {
	descJSON, _ := json.Marshal(d.Description)
	propsJSON, _ := json.Marshal(d.Properties)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO databases (id, workspace_id, page_id, title, description, icon, cover, is_inline, properties, created_by, created_at, updated_by, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, d.ID, d.WorkspaceID, d.PageID, d.Title, string(descJSON), d.Icon, d.Cover, d.IsInline, string(propsJSON), d.CreatedBy, d.CreatedAt, d.UpdatedBy, d.UpdatedAt)
	return err
}

func (s *DatabasesStore) GetByID(ctx context.Context, id string) (*databases.Database, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, page_id, title, CAST(description AS VARCHAR), icon, cover, is_inline, CAST(properties AS VARCHAR), created_by, created_at, updated_by, updated_at
		FROM databases WHERE id = ?
	`, id)
	return s.scanDatabase(row)
}

func (s *DatabasesStore) Update(ctx context.Context, id string, in *databases.UpdateIn) error {
	if in.Title != nil {
		_, err := s.db.ExecContext(ctx, "UPDATE databases SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", *in.Title, id)
		if err != nil {
			return err
		}
	}
	if in.Icon != nil {
		_, err := s.db.ExecContext(ctx, "UPDATE databases SET icon = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", *in.Icon, id)
		if err != nil {
			return err
		}
	}
	if in.Cover != nil {
		_, err := s.db.ExecContext(ctx, "UPDATE databases SET cover = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", *in.Cover, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *DatabasesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM databases WHERE id = ?", id)
	return err
}

func (s *DatabasesStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]*databases.Database, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, page_id, title, CAST(description AS VARCHAR), icon, cover, is_inline, CAST(properties AS VARCHAR), created_by, created_at, updated_by, updated_at
		FROM databases WHERE workspace_id = ?
		ORDER BY title
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanDatabases(rows)
}

func (s *DatabasesStore) ListByPage(ctx context.Context, pageID string) ([]*databases.Database, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, page_id, title, CAST(description AS VARCHAR), icon, cover, is_inline, CAST(properties AS VARCHAR), created_by, created_at, updated_by, updated_at
		FROM databases WHERE page_id = ?
		ORDER BY title
	`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanDatabases(rows)
}

func (s *DatabasesStore) UpdateProperties(ctx context.Context, id string, props []databases.Property) error {
	propsJSON, _ := json.Marshal(props)
	_, err := s.db.ExecContext(ctx, "UPDATE databases SET properties = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", string(propsJSON), id)
	return err
}

func (s *DatabasesStore) scanDatabase(row *sql.Row) (*databases.Database, error) {
	var d databases.Database
	var descJSON, propsJSON string
	err := row.Scan(&d.ID, &d.WorkspaceID, &d.PageID, &d.Title, &descJSON, &d.Icon, &d.Cover, &d.IsInline, &propsJSON, &d.CreatedBy, &d.CreatedAt, &d.UpdatedBy, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(descJSON), &d.Description)
	json.Unmarshal([]byte(propsJSON), &d.Properties)
	return &d, nil
}

func (s *DatabasesStore) scanDatabases(rows *sql.Rows) ([]*databases.Database, error) {
	var result []*databases.Database
	for rows.Next() {
		var d databases.Database
		var descJSON, propsJSON string
		err := rows.Scan(&d.ID, &d.WorkspaceID, &d.PageID, &d.Title, &descJSON, &d.Icon, &d.Cover, &d.IsInline, &propsJSON, &d.CreatedBy, &d.CreatedAt, &d.UpdatedBy, &d.UpdatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(descJSON), &d.Description)
		json.Unmarshal([]byte(propsJSON), &d.Properties)
		result = append(result, &d)
	}
	return result, rows.Err()
}
