package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/table/feature/workspaces"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// WorkspacesStore provides PostgreSQL-based workspace storage.
type WorkspacesStore struct {
	db *sql.DB
}

// NewWorkspacesStore creates a new workspaces store.
func NewWorkspacesStore(db *sql.DB) *WorkspacesStore {
	return &WorkspacesStore{db: db}
}

// Create creates a new workspace.
func (s *WorkspacesStore) Create(ctx context.Context, ws *workspaces.Workspace) error {
	now := time.Now()
	ws.CreatedAt = now
	ws.UpdatedAt = now
	if ws.Plan == "" {
		ws.Plan = "free"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, slug, icon, plan, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, ws.ID, ws.Name, ws.Slug, nullString(ws.Icon), ws.Plan, ws.OwnerID, ws.CreatedAt, ws.UpdatedAt)
	return err
}

// GetByID retrieves a workspace by ID.
func (s *WorkspacesStore) GetByID(ctx context.Context, id string) (*workspaces.Workspace, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, icon, plan, owner_id, created_at, updated_at
		FROM workspaces WHERE id = $1
	`, id)
	return s.scanWorkspace(row)
}

// GetBySlug retrieves a workspace by slug.
func (s *WorkspacesStore) GetBySlug(ctx context.Context, slug string) (*workspaces.Workspace, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, icon, plan, owner_id, created_at, updated_at
		FROM workspaces WHERE slug = $1
	`, slug)
	return s.scanWorkspace(row)
}

// Update updates a workspace.
func (s *WorkspacesStore) Update(ctx context.Context, ws *workspaces.Workspace) error {
	ws.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		UPDATE workspaces SET
			name = $1, slug = $2, icon = $3, plan = $4, updated_at = $5
		WHERE id = $6
	`, ws.Name, ws.Slug, nullString(ws.Icon), ws.Plan, ws.UpdatedAt, ws.ID)
	return err
}

// Delete deletes a workspace.
func (s *WorkspacesStore) Delete(ctx context.Context, id string) error {
	_, _ = s.db.ExecContext(ctx, `DELETE FROM workspace_members WHERE workspace_id = $1`, id)

	baseIDs, err := s.fetchBaseIDs(ctx, id)
	if err != nil {
		return err
	}
	baseStore := NewBasesStore(s.db)
	for _, baseID := range baseIDs {
		if err := baseStore.Delete(ctx, baseID); err != nil {
			return err
		}
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = $1`, id)
	return err
}

// AddMember adds a member to a workspace.
func (s *WorkspacesStore) AddMember(ctx context.Context, member *workspaces.Member) error {
	if member.ID == "" {
		member.ID = ulid.New()
	}
	member.JoinedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspace_members (id, workspace_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`, member.ID, member.WorkspaceID, member.UserID, member.Role, member.JoinedAt)
	return err
}

// RemoveMember removes a member from a workspace.
func (s *WorkspacesStore) RemoveMember(ctx context.Context, workspaceID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID)
	return err
}

// UpdateMemberRole updates a member's role.
func (s *WorkspacesStore) UpdateMemberRole(ctx context.Context, workspaceID, userID, role string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE workspace_members SET role = $1 WHERE workspace_id = $2 AND user_id = $3
	`, role, workspaceID, userID)
	return err
}

// GetMember retrieves a specific member.
func (s *WorkspacesStore) GetMember(ctx context.Context, workspaceID, userID string) (*workspaces.Member, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, user_id, role, joined_at
		FROM workspace_members WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID)

	member := &workspaces.Member{}
	err := row.Scan(&member.ID, &member.WorkspaceID, &member.UserID, &member.Role, &member.JoinedAt)
	if err == sql.ErrNoRows {
		return nil, workspaces.ErrNotMember
	}
	if err != nil {
		return nil, err
	}
	return member, nil
}

// ListMembers lists all members of a workspace.
func (s *WorkspacesStore) ListMembers(ctx context.Context, workspaceID string) ([]*workspaces.Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, user_id, role, joined_at
		FROM workspace_members WHERE workspace_id = $1
		ORDER BY joined_at ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*workspaces.Member
	for rows.Next() {
		member := &workspaces.Member{}
		if err := rows.Scan(&member.ID, &member.WorkspaceID, &member.UserID, &member.Role, &member.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

// ListByUser lists all workspaces a user belongs to.
func (s *WorkspacesStore) ListByUser(ctx context.Context, userID string) ([]*workspaces.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT w.id, w.name, w.slug, w.icon, w.plan, w.owner_id, w.created_at, w.updated_at
		FROM workspaces w
		LEFT JOIN workspace_members wm ON w.id = wm.workspace_id
		WHERE w.owner_id = $1 OR wm.user_id = $1
		ORDER BY w.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaceList []*workspaces.Workspace
	for rows.Next() {
		ws, err := s.scanWorkspaceRows(rows)
		if err != nil {
			return nil, err
		}
		workspaceList = append(workspaceList, ws)
	}
	return workspaceList, rows.Err()
}

func (s *WorkspacesStore) scanWorkspace(row *sql.Row) (*workspaces.Workspace, error) {
	ws := &workspaces.Workspace{}
	var icon sql.NullString

	err := row.Scan(&ws.ID, &ws.Name, &ws.Slug, &icon, &ws.Plan, &ws.OwnerID, &ws.CreatedAt, &ws.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, workspaces.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if icon.Valid {
		ws.Icon = icon.String
	}
	return ws, nil
}

func (s *WorkspacesStore) scanWorkspaceRows(rows *sql.Rows) (*workspaces.Workspace, error) {
	ws := &workspaces.Workspace{}
	var icon sql.NullString

	err := rows.Scan(&ws.ID, &ws.Name, &ws.Slug, &icon, &ws.Plan, &ws.OwnerID, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if icon.Valid {
		ws.Icon = icon.String
	}
	return ws, nil
}

func (s *WorkspacesStore) fetchBaseIDs(ctx context.Context, workspaceID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM bases WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var baseIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		baseIDs = append(baseIDs, id)
	}
	return baseIDs, rows.Err()
}
