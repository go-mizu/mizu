package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/favorites"
	"github.com/go-mizu/blueprints/workspace/feature/history"
	"github.com/go-mizu/blueprints/workspace/feature/notifications"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/search"
	"github.com/go-mizu/blueprints/workspace/feature/sharing"
	"github.com/go-mizu/blueprints/workspace/feature/templates"
)

// SharesStore implements sharing.Store.
type SharesStore struct {
	db *sql.DB
}

func NewSharesStore(db *sql.DB) *SharesStore { return &SharesStore{db: db} }

func (s *SharesStore) Create(ctx context.Context, sh *sharing.Share) error {
	// Convert empty strings to nil for token/password to avoid unique constraint violations
	var token, password interface{}
	if sh.Token != "" {
		token = sh.Token
	}
	if sh.Password != "" {
		password = sh.Password
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO shares (id, page_id, type, permission, user_id, token, password, expires_at, domain, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sh.ID, sh.PageID, sh.Type, sh.Permission, sh.UserID, token, password, sh.ExpiresAt, sh.Domain, sh.CreatedBy, sh.CreatedAt)
	return err
}

func (s *SharesStore) GetByID(ctx context.Context, id string) (*sharing.Share, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, page_id, type, permission, user_id, token, password, expires_at, domain, created_by, created_at FROM shares WHERE id = ?`, id)
	var sh sharing.Share
	err := row.Scan(&sh.ID, &sh.PageID, &sh.Type, &sh.Permission, &sh.UserID, &sh.Token, &sh.Password, &sh.ExpiresAt, &sh.Domain, &sh.CreatedBy, &sh.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

func (s *SharesStore) GetByToken(ctx context.Context, token string) (*sharing.Share, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, page_id, type, permission, user_id, token, password, expires_at, domain, created_by, created_at FROM shares WHERE token = ?`, token)
	var sh sharing.Share
	err := row.Scan(&sh.ID, &sh.PageID, &sh.Type, &sh.Permission, &sh.UserID, &sh.Token, &sh.Password, &sh.ExpiresAt, &sh.Domain, &sh.CreatedBy, &sh.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

func (s *SharesStore) GetByPageAndUser(ctx context.Context, pageID, userID string) (*sharing.Share, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, page_id, type, permission, user_id, token, password, expires_at, domain, created_by, created_at FROM shares WHERE page_id = ? AND user_id = ?`, pageID, userID)
	var sh sharing.Share
	err := row.Scan(&sh.ID, &sh.PageID, &sh.Type, &sh.Permission, &sh.UserID, &sh.Token, &sh.Password, &sh.ExpiresAt, &sh.Domain, &sh.CreatedBy, &sh.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

func (s *SharesStore) GetPublicByPage(ctx context.Context, pageID string) (*sharing.Share, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, page_id, type, permission, user_id, token, password, expires_at, domain, created_by, created_at FROM shares WHERE page_id = ? AND type = 'public'`, pageID)
	var sh sharing.Share
	err := row.Scan(&sh.ID, &sh.PageID, &sh.Type, &sh.Permission, &sh.UserID, &sh.Token, &sh.Password, &sh.ExpiresAt, &sh.Domain, &sh.CreatedBy, &sh.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

func (s *SharesStore) Update(ctx context.Context, id string, perm sharing.Permission) error {
	_, err := s.db.ExecContext(ctx, "UPDATE shares SET permission = ? WHERE id = ?", perm, id)
	return err
}

func (s *SharesStore) UpdateLink(ctx context.Context, id string, opts sharing.LinkOpts) error {
	_, err := s.db.ExecContext(ctx, "UPDATE shares SET permission = ?, password = ?, expires_at = ? WHERE id = ?", opts.Permission, opts.Password, opts.ExpiresAt, id)
	return err
}

func (s *SharesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM shares WHERE id = ?", id)
	return err
}

func (s *SharesStore) ListByPage(ctx context.Context, pageID string) ([]*sharing.Share, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, page_id, type, permission, user_id, token, password, expires_at, domain, created_by, created_at FROM shares WHERE page_id = ?`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*sharing.Share
	for rows.Next() {
		var sh sharing.Share
		rows.Scan(&sh.ID, &sh.PageID, &sh.Type, &sh.Permission, &sh.UserID, &sh.Token, &sh.Password, &sh.ExpiresAt, &sh.Domain, &sh.CreatedBy, &sh.CreatedAt)
		result = append(result, &sh)
	}
	return result, rows.Err()
}

// FavoritesStore implements favorites.Store.
type FavoritesStore struct {
	db *sql.DB
}

func NewFavoritesStore(db *sql.DB) *FavoritesStore { return &FavoritesStore{db: db} }

func (s *FavoritesStore) Create(ctx context.Context, f *favorites.Favorite) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO favorites (id, user_id, page_id, workspace_id, created_at) VALUES (?, ?, ?, ?, ?)`, f.ID, f.UserID, f.PageID, f.WorkspaceID, f.CreatedAt)
	return err
}

func (s *FavoritesStore) Delete(ctx context.Context, userID, pageID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM favorites WHERE user_id = ? AND page_id = ?", userID, pageID)
	return err
}

func (s *FavoritesStore) List(ctx context.Context, userID, workspaceID string) ([]*favorites.Favorite, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, page_id, workspace_id, created_at FROM favorites WHERE user_id = ? AND workspace_id = ? ORDER BY created_at DESC`, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*favorites.Favorite
	for rows.Next() {
		var f favorites.Favorite
		rows.Scan(&f.ID, &f.UserID, &f.PageID, &f.WorkspaceID, &f.CreatedAt)
		result = append(result, &f)
	}
	return result, rows.Err()
}

func (s *FavoritesStore) Exists(ctx context.Context, userID, pageID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM favorites WHERE user_id = ? AND page_id = ?", userID, pageID).Scan(&count)
	return count > 0, err
}

// HistoryStore implements history.Store.
type HistoryStore struct {
	db *sql.DB
}

func NewHistoryStore(db *sql.DB) *HistoryStore { return &HistoryStore{db: db} }

func (s *HistoryStore) CreateRevision(ctx context.Context, r *history.Revision) error {
	blocksJSON, _ := json.Marshal(r.Blocks)
	propsJSON, _ := json.Marshal(r.Properties)
	_, err := s.db.ExecContext(ctx, `INSERT INTO revisions (id, page_id, version, title, blocks, properties, author_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, r.ID, r.PageID, r.Version, r.Title, string(blocksJSON), string(propsJSON), r.AuthorID, r.CreatedAt)
	return err
}

func (s *HistoryStore) GetRevision(ctx context.Context, id string) (*history.Revision, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, page_id, version, title, CAST(blocks AS VARCHAR), CAST(properties AS VARCHAR), author_id, created_at FROM revisions WHERE id = ?`, id)
	var r history.Revision
	var blocksJSON, propsJSON string
	err := row.Scan(&r.ID, &r.PageID, &r.Version, &r.Title, &blocksJSON, &propsJSON, &r.AuthorID, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(blocksJSON), &r.Blocks)
	json.Unmarshal([]byte(propsJSON), &r.Properties)
	return &r, nil
}

func (s *HistoryStore) ListRevisions(ctx context.Context, pageID string, limit int) ([]*history.Revision, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, page_id, version, title, author_id, created_at FROM revisions WHERE page_id = ? ORDER BY version DESC LIMIT ?`, pageID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*history.Revision
	for rows.Next() {
		var r history.Revision
		rows.Scan(&r.ID, &r.PageID, &r.Version, &r.Title, &r.AuthorID, &r.CreatedAt)
		result = append(result, &r)
	}
	return result, rows.Err()
}

func (s *HistoryStore) GetLatestVersion(ctx context.Context, pageID string) (int, error) {
	var version int
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM revisions WHERE page_id = ?", pageID).Scan(&version)
	return version, err
}

func (s *HistoryStore) CreateActivity(ctx context.Context, a *history.Activity) error {
	detailsJSON, _ := json.Marshal(a.Details)
	_, err := s.db.ExecContext(ctx, `INSERT INTO activities (id, workspace_id, page_id, block_id, actor_id, action, details, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, a.ID, a.WorkspaceID, a.PageID, a.BlockID, a.ActorID, a.Action, string(detailsJSON), a.CreatedAt)
	return err
}

func (s *HistoryStore) ListByWorkspace(ctx context.Context, workspaceID string, opts history.ActivityOpts) ([]*history.Activity, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, page_id, block_id, actor_id, action, CAST(details AS VARCHAR), created_at FROM activities WHERE workspace_id = ? ORDER BY created_at DESC LIMIT ?`, workspaceID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanActivities(rows)
}

func (s *HistoryStore) ListByPage(ctx context.Context, pageID string, opts history.ActivityOpts) ([]*history.Activity, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, page_id, block_id, actor_id, action, CAST(details AS VARCHAR), created_at FROM activities WHERE page_id = ? ORDER BY created_at DESC LIMIT ?`, pageID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanActivities(rows)
}

func (s *HistoryStore) ListByUser(ctx context.Context, userID string, opts history.ActivityOpts) ([]*history.Activity, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, page_id, block_id, actor_id, action, CAST(details AS VARCHAR), created_at FROM activities WHERE actor_id = ? ORDER BY created_at DESC LIMIT ?`, userID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanActivities(rows)
}

func (s *HistoryStore) scanActivities(rows *sql.Rows) ([]*history.Activity, error) {
	var result []*history.Activity
	for rows.Next() {
		var a history.Activity
		var detailsJSON string
		rows.Scan(&a.ID, &a.WorkspaceID, &a.PageID, &a.BlockID, &a.ActorID, &a.Action, &detailsJSON, &a.CreatedAt)
		json.Unmarshal([]byte(detailsJSON), &a.Details)
		result = append(result, &a)
	}
	return result, rows.Err()
}

// NotificationsStore implements notifications.Store.
type NotificationsStore struct {
	db *sql.DB
}

func NewNotificationsStore(db *sql.DB) *NotificationsStore { return &NotificationsStore{db: db} }

func (s *NotificationsStore) Create(ctx context.Context, n *notifications.Notification) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO notifications (id, user_id, type, title, body, page_id, actor_id, is_read, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, n.ID, n.UserID, n.Type, n.Title, n.Body, n.PageID, n.ActorID, n.IsRead, n.CreatedAt)
	return err
}

func (s *NotificationsStore) GetByID(ctx context.Context, id string) (*notifications.Notification, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, type, title, body, page_id, actor_id, is_read, created_at FROM notifications WHERE id = ?`, id)
	var n notifications.Notification
	err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.PageID, &n.ActorID, &n.IsRead, &n.CreatedAt)
	return &n, err
}

func (s *NotificationsStore) List(ctx context.Context, userID string, opts notifications.ListOpts) ([]*notifications.Notification, error) {
	query := "SELECT id, user_id, type, title, body, page_id, actor_id, is_read, created_at FROM notifications WHERE user_id = ?"
	if opts.UnreadOnly {
		query += " AND is_read = FALSE"
	}
	query += " ORDER BY created_at DESC"
	if opts.Limit > 0 {
		query += " LIMIT " + string(rune(opts.Limit))
	}
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*notifications.Notification
	for rows.Next() {
		var n notifications.Notification
		rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.PageID, &n.ActorID, &n.IsRead, &n.CreatedAt)
		result = append(result, &n)
	}
	return result, rows.Err()
}

func (s *NotificationsStore) MarkRead(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE notifications SET is_read = TRUE WHERE id = ?", id)
	return err
}

func (s *NotificationsStore) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE notifications SET is_read = TRUE WHERE user_id = ?", userID)
	return err
}

func (s *NotificationsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM notifications WHERE id = ?", id)
	return err
}

func (s *NotificationsStore) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = FALSE", userID).Scan(&count)
	return count, err
}

// TemplatesStore implements templates.Store.
type TemplatesStore struct {
	db *sql.DB
}

func NewTemplatesStore(db *sql.DB) *TemplatesStore { return &TemplatesStore{db: db} }

func (s *TemplatesStore) Create(ctx context.Context, t *templates.Template) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO templates (id, name, description, category, preview, page_id, is_system, workspace_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, t.ID, t.Name, t.Description, t.Category, t.Preview, t.PageID, t.IsSystem, t.WorkspaceID, t.CreatedAt)
	return err
}

func (s *TemplatesStore) GetByID(ctx context.Context, id string) (*templates.Template, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, description, category, preview, page_id, is_system, workspace_id, created_at FROM templates WHERE id = ?`, id)
	var t templates.Template
	err := row.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.Preview, &t.PageID, &t.IsSystem, &t.WorkspaceID, &t.CreatedAt)
	return &t, err
}

func (s *TemplatesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM templates WHERE id = ?", id)
	return err
}

func (s *TemplatesStore) ListSystem(ctx context.Context, category string) ([]*templates.Template, error) {
	query := "SELECT id, name, description, category, preview, page_id, is_system, workspace_id, created_at FROM templates WHERE is_system = TRUE"
	if category != "" {
		query += " AND category = '" + category + "'"
	}
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*templates.Template
	for rows.Next() {
		var t templates.Template
		rows.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.Preview, &t.PageID, &t.IsSystem, &t.WorkspaceID, &t.CreatedAt)
		result = append(result, &t)
	}
	return result, rows.Err()
}

func (s *TemplatesStore) ListByWorkspace(ctx context.Context, workspaceID, category string) ([]*templates.Template, error) {
	query := "SELECT id, name, description, category, preview, page_id, is_system, workspace_id, created_at FROM templates WHERE workspace_id = ?"
	args := []interface{}{workspaceID}
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*templates.Template
	for rows.Next() {
		var t templates.Template
		rows.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.Preview, &t.PageID, &t.IsSystem, &t.WorkspaceID, &t.CreatedAt)
		result = append(result, &t)
	}
	return result, rows.Err()
}

// SearchStore implements search.Store.
type SearchStore struct {
	db *sql.DB
}

func NewSearchStore(db *sql.DB) *SearchStore { return &SearchStore{db: db} }

func (s *SearchStore) RecordAccess(ctx context.Context, userID, pageID string, accessedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO page_access (user_id, page_id, accessed_at) VALUES (?, ?, ?) ON CONFLICT (user_id, page_id) DO UPDATE SET accessed_at = EXCLUDED.accessed_at`, userID, pageID, accessedAt)
	return err
}

func (s *SearchStore) Search(ctx context.Context, workspaceID, query string, opts search.SearchOpts) ([]*pages.Page, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, parent_type, parent_id, title, icon, cover, is_archived, created_by, created_at, updated_by, updated_at
		FROM pages
		WHERE workspace_id = ? AND is_archived = FALSE AND title ILIKE ?
		ORDER BY updated_at DESC
		LIMIT ?
	`, workspaceID, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*pages.Page
	for rows.Next() {
		var p pages.Page
		rows.Scan(&p.ID, &p.WorkspaceID, &p.ParentType, &p.ParentID, &p.Title, &p.Icon, &p.Cover, &p.IsArchived, &p.CreatedBy, &p.CreatedAt, &p.UpdatedBy, &p.UpdatedAt)
		result = append(result, &p)
	}
	return result, rows.Err()
}

func (s *SearchStore) QuickSearch(ctx context.Context, workspaceID, query string, limit int) ([]*pages.PageRef, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, icon
		FROM pages
		WHERE workspace_id = ? AND is_archived = FALSE AND title ILIKE ?
		ORDER BY updated_at DESC
		LIMIT ?
	`, workspaceID, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*pages.PageRef
	for rows.Next() {
		var p pages.PageRef
		rows.Scan(&p.ID, &p.Title, &p.Icon)
		result = append(result, &p)
	}
	return result, rows.Err()
}

func (s *SearchStore) GetRecent(ctx context.Context, userID, workspaceID string, limit int) ([]*pages.Page, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.workspace_id, p.parent_type, p.parent_id, p.title, p.icon, p.cover, p.is_archived, p.created_by, p.created_at, p.updated_by, p.updated_at
		FROM pages p
		JOIN page_access a ON p.id = a.page_id
		WHERE p.workspace_id = ? AND a.user_id = ? AND p.is_archived = FALSE
		ORDER BY a.accessed_at DESC
		LIMIT ?
	`, workspaceID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*pages.Page
	for rows.Next() {
		var p pages.Page
		rows.Scan(&p.ID, &p.WorkspaceID, &p.ParentType, &p.ParentID, &p.Title, &p.Icon, &p.Cover, &p.IsArchived, &p.CreatedBy, &p.CreatedAt, &p.UpdatedBy, &p.UpdatedAt)
		result = append(result, &p)
	}
	return result, rows.Err()
}
