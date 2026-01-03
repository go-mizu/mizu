package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/workspace/feature/workspaces"
)

// WorkspacesStore implements workspaces.Store.
type WorkspacesStore struct {
	db *sql.DB
}

// NewWorkspacesStore creates a new WorkspacesStore.
func NewWorkspacesStore(db *sql.DB) *WorkspacesStore {
	return &WorkspacesStore{db: db}
}

func (s *WorkspacesStore) Create(ctx context.Context, ws *workspaces.Workspace) error {
	settingsJSON, _ := json.Marshal(ws.Settings)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, slug, icon, domain, plan, settings, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ws.ID, ws.Name, ws.Slug, ws.Icon, ws.Domain, ws.Plan, string(settingsJSON), ws.OwnerID, ws.CreatedAt, ws.UpdatedAt)
	return err
}

func (s *WorkspacesStore) GetByID(ctx context.Context, id string) (*workspaces.Workspace, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, icon, domain, plan, CAST(settings AS VARCHAR), owner_id, created_at, updated_at
		FROM workspaces WHERE id = ?
	`, id)
	return s.scanWorkspace(row)
}

func (s *WorkspacesStore) GetBySlug(ctx context.Context, slug string) (*workspaces.Workspace, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, icon, domain, plan, CAST(settings AS VARCHAR), owner_id, created_at, updated_at
		FROM workspaces WHERE slug = ?
	`, slug)
	return s.scanWorkspace(row)
}

func (s *WorkspacesStore) ListByUser(ctx context.Context, userID string) ([]*workspaces.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.name, w.slug, w.icon, w.domain, w.plan, CAST(w.settings AS VARCHAR), w.owner_id, w.created_at, w.updated_at
		FROM workspaces w
		JOIN members m ON w.id = m.workspace_id
		WHERE m.user_id = ?
		ORDER BY w.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*workspaces.Workspace
	for rows.Next() {
		ws, err := s.scanWorkspaceFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, ws)
	}
	return result, rows.Err()
}

func (s *WorkspacesStore) Update(ctx context.Context, id string, in *workspaces.UpdateIn) error {
	sets := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []interface{}{}

	if in.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *in.Name)
	}
	if in.Icon != nil {
		sets = append(sets, "icon = ?")
		args = append(args, *in.Icon)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE workspaces SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *WorkspacesStore) UpdateSettings(ctx context.Context, id string, settings workspaces.Settings) error {
	settingsJSON, _ := json.Marshal(settings)
	_, err := s.db.ExecContext(ctx, `
		UPDATE workspaces SET settings = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, string(settingsJSON), id)
	return err
}

func (s *WorkspacesStore) UpdateOwner(ctx context.Context, id, ownerID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE workspaces SET owner_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, ownerID, id)
	return err
}

func (s *WorkspacesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM workspaces WHERE id = ?", id)
	return err
}

func (s *WorkspacesStore) scanWorkspace(row *sql.Row) (*workspaces.Workspace, error) {
	var ws workspaces.Workspace
	var settingsJSON string
	err := row.Scan(&ws.ID, &ws.Name, &ws.Slug, &ws.Icon, &ws.Domain, &ws.Plan, &settingsJSON, &ws.OwnerID, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(settingsJSON), &ws.Settings)
	return &ws, nil
}

func (s *WorkspacesStore) scanWorkspaceFromRows(rows *sql.Rows) (*workspaces.Workspace, error) {
	var ws workspaces.Workspace
	var settingsJSON string
	err := rows.Scan(&ws.ID, &ws.Name, &ws.Slug, &ws.Icon, &ws.Domain, &ws.Plan, &settingsJSON, &ws.OwnerID, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(settingsJSON), &ws.Settings)
	return &ws, nil
}
