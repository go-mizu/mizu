package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/table/feature/bases"
)

// BasesStore provides PostgreSQL-based base storage.
type BasesStore struct {
	db *sql.DB
}

// NewBasesStore creates a new bases store.
func NewBasesStore(db *sql.DB) *BasesStore {
	return &BasesStore{db: db}
}

// Create creates a new base.
func (s *BasesStore) Create(ctx context.Context, base *bases.Base) error {
	now := time.Now()
	base.CreatedAt = now
	base.UpdatedAt = now
	if base.Color == "" {
		base.Color = "#2563EB"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bases (id, workspace_id, name, description, icon, color, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, base.ID, base.WorkspaceID, base.Name, nullString(base.Description), nullString(base.Icon), base.Color, base.CreatedBy, base.CreatedAt, base.UpdatedAt)
	return err
}

// GetByID retrieves a base by ID.
func (s *BasesStore) GetByID(ctx context.Context, id string) (*bases.Base, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, icon, color, created_by, created_at, updated_at
		FROM bases WHERE id = $1
	`, id)
	return s.scanBase(row)
}

// Update updates a base.
func (s *BasesStore) Update(ctx context.Context, base *bases.Base) error {
	base.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		UPDATE bases SET
			name = $1, description = $2, icon = $3, color = $4, updated_at = $5
		WHERE id = $6
	`, base.Name, nullString(base.Description), nullString(base.Icon), base.Color, base.UpdatedAt, base.ID)
	return err
}

// Delete deletes a base and all related data.
func (s *BasesStore) Delete(ctx context.Context, id string) error {
	// Delete related webhooks and deliveries
	_, _ = s.db.ExecContext(ctx, `
		DELETE FROM webhook_deliveries
		WHERE webhook_id IN (SELECT id FROM webhooks WHERE base_id = $1)
	`, id)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE base_id = $1`, id)
	// Delete shares for the base
	_, _ = s.db.ExecContext(ctx, `DELETE FROM shares WHERE base_id = $1`, id)

	// Delete tables and their related data using the tables store
	tableIDs, err := s.fetchTableIDs(ctx, id)
	if err != nil {
		return err
	}
	tableStore := NewTablesStore(s.db)
	for _, tableID := range tableIDs {
		if err := tableStore.Delete(ctx, tableID); err != nil {
			return err
		}
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM bases WHERE id = $1`, id)
	return err
}

// ListByWorkspace lists all bases in a workspace.
func (s *BasesStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]*bases.Base, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, name, description, icon, color, created_by, created_at, updated_at
		FROM bases WHERE workspace_id = $1
		ORDER BY name ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var baseList []*bases.Base
	for rows.Next() {
		base, err := s.scanBaseRows(rows)
		if err != nil {
			return nil, err
		}
		baseList = append(baseList, base)
	}
	return baseList, rows.Err()
}

func (s *BasesStore) scanBase(row *sql.Row) (*bases.Base, error) {
	base := &bases.Base{}
	var description, icon sql.NullString

	err := row.Scan(&base.ID, &base.WorkspaceID, &base.Name, &description, &icon, &base.Color, &base.CreatedBy, &base.CreatedAt, &base.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, bases.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if description.Valid {
		base.Description = description.String
	}
	if icon.Valid {
		base.Icon = icon.String
	}
	return base, nil
}

func (s *BasesStore) scanBaseRows(rows *sql.Rows) (*bases.Base, error) {
	base := &bases.Base{}
	var description, icon sql.NullString

	err := rows.Scan(&base.ID, &base.WorkspaceID, &base.Name, &description, &icon, &base.Color, &base.CreatedBy, &base.CreatedAt, &base.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		base.Description = description.String
	}
	if icon.Valid {
		base.Icon = icon.String
	}
	return base, nil
}

func (s *BasesStore) fetchTableIDs(ctx context.Context, baseID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM tables WHERE base_id = $1`, baseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		tableIDs = append(tableIDs, id)
	}
	return tableIDs, rows.Err()
}
