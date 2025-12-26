package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
)

// WorkspacesStore handles workspace data access.
type WorkspacesStore struct {
	db *sql.DB
}

// NewWorkspacesStore creates a new workspaces store.
func NewWorkspacesStore(db *sql.DB) *WorkspacesStore {
	return &WorkspacesStore{db: db}
}

func (s *WorkspacesStore) Create(ctx context.Context, w *workspaces.Workspace) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspaces (id, slug, name, description, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, w.ID, w.Slug, w.Name, w.Description, w.AvatarURL, w.CreatedAt, w.UpdatedAt)
	return err
}

func (s *WorkspacesStore) GetByID(ctx context.Context, id string) (*workspaces.Workspace, error) {
	w := &workspaces.Workspace{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, slug, name, description, avatar_url, created_at, updated_at
		FROM workspaces WHERE id = $1
	`, id).Scan(&w.ID, &w.Slug, &w.Name, &w.Description, &w.AvatarURL, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

func (s *WorkspacesStore) GetBySlug(ctx context.Context, slug string) (*workspaces.Workspace, error) {
	w := &workspaces.Workspace{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, slug, name, description, avatar_url, created_at, updated_at
		FROM workspaces WHERE slug = $1
	`, slug).Scan(&w.ID, &w.Slug, &w.Name, &w.Description, &w.AvatarURL, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

func (s *WorkspacesStore) ListByUser(ctx context.Context, userID string) ([]*workspaces.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.slug, w.name, w.description, w.avatar_url, w.created_at, w.updated_at
		FROM workspaces w
		INNER JOIN workspace_members wm ON w.id = wm.workspace_id
		WHERE wm.user_id = $1
		ORDER BY w.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*workspaces.Workspace
	for rows.Next() {
		w := &workspaces.Workspace{}
		if err := rows.Scan(&w.ID, &w.Slug, &w.Name, &w.Description, &w.AvatarURL, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (s *WorkspacesStore) Update(ctx context.Context, id string, in *workspaces.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE workspaces SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			avatar_url = COALESCE($4, avatar_url),
			updated_at = $5
		WHERE id = $1
	`, id, in.Name, in.Description, in.AvatarURL, time.Now())
	return err
}

func (s *WorkspacesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = $1`, id)
	return err
}

// Member operations

func (s *WorkspacesStore) AddMember(ctx context.Context, m *workspaces.Member) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspace_members (id, workspace_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)
	`, m.ID, m.WorkspaceID, m.UserID, m.Role, m.JoinedAt)
	return err
}

func (s *WorkspacesStore) GetMember(ctx context.Context, workspaceID, userID string) (*workspaces.Member, error) {
	m := &workspaces.Member{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, user_id, role, joined_at
		FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (s *WorkspacesStore) ListMembers(ctx context.Context, workspaceID string) ([]*workspaces.Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, user_id, role, joined_at
		FROM workspace_members
		WHERE workspace_id = $1
		ORDER BY joined_at
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*workspaces.Member
	for rows.Next() {
		m := &workspaces.Member{}
		if err := rows.Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *WorkspacesStore) UpdateMemberRole(ctx context.Context, id, role string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE workspace_members SET role = $2 WHERE id = $1
	`, id, role)
	return err
}

func (s *WorkspacesStore) RemoveMember(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workspace_members WHERE id = $1`, id)
	return err
}
