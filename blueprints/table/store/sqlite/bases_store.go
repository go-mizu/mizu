package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/table/feature/bases"
)

// BasesStore provides SQLite-based base storage.
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, base.ID, base.WorkspaceID, base.Name, base.Description, base.Icon, base.Color, base.CreatedBy, base.CreatedAt, base.UpdatedAt)
	return err
}

// GetByID retrieves a base by ID.
func (s *BasesStore) GetByID(ctx context.Context, id string) (*bases.Base, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, icon, color, created_by, created_at, updated_at
		FROM bases WHERE id = ?
	`, id)
	return s.scanBase(row)
}

// Update updates a base.
func (s *BasesStore) Update(ctx context.Context, base *bases.Base) error {
	base.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		UPDATE bases SET
			name = ?, description = ?, icon = ?, color = ?, updated_at = ?
		WHERE id = ?
	`, base.Name, base.Description, base.Icon, base.Color, base.UpdatedAt, base.ID)
	return err
}

// Delete deletes a base and all related data.
func (s *BasesStore) Delete(ctx context.Context, id string) error {
	// Delete related webhooks and deliveries
	_, _ = s.db.ExecContext(ctx, `
		DELETE FROM webhook_deliveries
		WHERE webhook_id IN (SELECT id FROM webhooks WHERE base_id = ?)
	`, id)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE base_id = ?`, id)
	// Delete shares for the base
	_, _ = s.db.ExecContext(ctx, `DELETE FROM shares WHERE base_id = ?`, id)

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

	_, err = s.db.ExecContext(ctx, `DELETE FROM bases WHERE id = ?`, id)
	return err
}

// ListByWorkspace lists all bases in a workspace.
func (s *BasesStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]*bases.Base, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, name, description, icon, color, created_by, created_at, updated_at
		FROM bases WHERE workspace_id = ?
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
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM tables WHERE base_id = ?`, baseID)
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
