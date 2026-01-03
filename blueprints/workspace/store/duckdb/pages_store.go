package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// PagesStore implements pages.Store.
type PagesStore struct {
	db *sql.DB
}

// NewPagesStore creates a new PagesStore.
func NewPagesStore(db *sql.DB) *PagesStore {
	return &PagesStore{db: db}
}

func (s *PagesStore) Create(ctx context.Context, p *pages.Page) error {
	propsJSON, _ := json.Marshal(p.Properties)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pages (id, workspace_id, parent_id, parent_type, database_id, row_position, title, icon, cover, cover_y, properties, is_template, is_archived, created_by, created_at, updated_by, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.WorkspaceID, p.ParentID, p.ParentType, nullString(p.DatabaseID), p.RowPosition, p.Title, p.Icon, p.Cover, p.CoverY, string(propsJSON), p.IsTemplate, p.IsArchived, p.CreatedBy, p.CreatedAt, p.UpdatedBy, p.UpdatedAt)
	return err
}

func (s *PagesStore) GetByID(ctx context.Context, id string) (*pages.Page, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, parent_id, parent_type, database_id, row_position, title, icon, cover, cover_y, CAST(properties AS VARCHAR), is_template, is_archived, created_by, created_at, updated_by, updated_at
		FROM pages WHERE id = ?
	`, id)
	return s.scanPage(row)
}

func (s *PagesStore) Update(ctx context.Context, id string, in *pages.UpdateIn) error {
	sets := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []interface{}{}

	if in.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *in.Title)
	}
	if in.Icon != nil {
		sets = append(sets, "icon = ?")
		args = append(args, *in.Icon)
	}
	if in.Cover != nil {
		sets = append(sets, "cover = ?")
		args = append(args, *in.Cover)
	}
	if in.CoverY != nil {
		sets = append(sets, "cover_y = ?")
		args = append(args, *in.CoverY)
	}
	if in.Properties != nil {
		propsJSON, _ := json.Marshal(*in.Properties)
		sets = append(sets, "properties = ?")
		args = append(args, string(propsJSON))
	}
	if in.UpdatedBy != "" {
		sets = append(sets, "updated_by = ?")
		args = append(args, in.UpdatedBy)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE pages SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *PagesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM pages WHERE id = ?", id)
	return err
}

func (s *PagesStore) ListByWorkspace(ctx context.Context, workspaceID string, opts pages.ListOpts) ([]*pages.Page, error) {
	query := `
		SELECT id, workspace_id, parent_id, parent_type, database_id, row_position, title, icon, cover, cover_y, CAST(properties AS VARCHAR), is_template, is_archived, created_by, created_at, updated_by, updated_at
		FROM pages
		WHERE workspace_id = ? AND parent_type = 'workspace' AND database_id IS NULL
	`
	if !opts.IncludeArchived {
		query += " AND is_archived = FALSE"
	}
	query += " ORDER BY title"
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPages(rows)
}

func (s *PagesStore) ListByParent(ctx context.Context, parentID string, parentType pages.ParentType) ([]*pages.Page, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, parent_id, parent_type, database_id, row_position, title, icon, cover, cover_y, CAST(properties AS VARCHAR), is_template, is_archived, created_by, created_at, updated_by, updated_at
		FROM pages
		WHERE parent_id = ? AND parent_type = ? AND is_archived = FALSE
		ORDER BY title
	`, parentID, parentType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPages(rows)
}

func (s *PagesStore) ListByParentIDs(ctx context.Context, parentIDs []string, parentType pages.ParentType) (map[string][]*pages.Page, error) {
	if len(parentIDs) == 0 {
		return make(map[string][]*pages.Page), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(parentIDs))
	args := make([]interface{}, len(parentIDs)+1)
	for i, id := range parentIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	args[len(parentIDs)] = parentType

	query := fmt.Sprintf(`
		SELECT id, workspace_id, parent_id, parent_type, database_id, row_position, title, icon, cover, cover_y, CAST(properties AS VARCHAR), is_template, is_archived, created_by, created_at, updated_by, updated_at
		FROM pages
		WHERE parent_id IN (%s) AND parent_type = ? AND is_archived = FALSE
		ORDER BY title
	`, strings.Join(placeholders, ", "))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allPages, err := s.scanPages(rows)
	if err != nil {
		return nil, err
	}

	// Group by parent ID
	result := make(map[string][]*pages.Page)
	for _, p := range allPages {
		result[p.ParentID] = append(result[p.ParentID], p)
	}

	return result, nil
}

func (s *PagesStore) Archive(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE pages SET is_archived = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (s *PagesStore) Restore(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE pages SET is_archived = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (s *PagesStore) ListArchived(ctx context.Context, workspaceID string) ([]*pages.Page, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, parent_id, parent_type, database_id, row_position, title, icon, cover, cover_y, CAST(properties AS VARCHAR), is_template, is_archived, created_by, created_at, updated_by, updated_at
		FROM pages
		WHERE workspace_id = ? AND is_archived = TRUE
		ORDER BY updated_at DESC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPages(rows)
}

func (s *PagesStore) Move(ctx context.Context, id, newParentID string, newParentType pages.ParentType) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pages SET parent_id = ?, parent_type = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, newParentID, newParentType, id)
	return err
}

func (s *PagesStore) Search(ctx context.Context, workspaceID, query string, opts pages.SearchOpts) ([]*pages.Page, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, parent_id, parent_type, database_id, row_position, title, icon, cover, cover_y, CAST(properties AS VARCHAR), is_template, is_archived, created_by, created_at, updated_by, updated_at
		FROM pages
		WHERE workspace_id = ? AND is_archived = FALSE AND LOWER(title) LIKE ?
		ORDER BY updated_at DESC
		LIMIT ?
	`, workspaceID, "%"+strings.ToLower(query)+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPages(rows)
}

func (s *PagesStore) GetRecent(ctx context.Context, userID, workspaceID string, limit int) ([]*pages.Page, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.workspace_id, p.parent_id, p.parent_type, p.database_id, p.row_position, p.title, p.icon, p.cover, p.cover_y, CAST(p.properties AS VARCHAR), p.is_template, p.is_archived, p.created_by, p.created_at, p.updated_by, p.updated_at
		FROM pages p
		JOIN page_access pa ON p.id = pa.page_id
		WHERE pa.user_id = ? AND p.workspace_id = ? AND p.is_archived = FALSE
		ORDER BY pa.accessed_at DESC
		LIMIT ?
	`, userID, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPages(rows)
}

func (s *PagesStore) scanPage(row *sql.Row) (*pages.Page, error) {
	var p pages.Page
	var propsJSON string
	var databaseID sql.NullString
	err := row.Scan(&p.ID, &p.WorkspaceID, &p.ParentID, &p.ParentType, &databaseID, &p.RowPosition, &p.Title, &p.Icon, &p.Cover, &p.CoverY, &propsJSON, &p.IsTemplate, &p.IsArchived, &p.CreatedBy, &p.CreatedAt, &p.UpdatedBy, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if databaseID.Valid {
		p.DatabaseID = databaseID.String
	}
	json.Unmarshal([]byte(propsJSON), &p.Properties)
	return &p, nil
}

func (s *PagesStore) scanPages(rows *sql.Rows) ([]*pages.Page, error) {
	var result []*pages.Page
	for rows.Next() {
		var p pages.Page
		var propsJSON string
		var databaseID sql.NullString
		err := rows.Scan(&p.ID, &p.WorkspaceID, &p.ParentID, &p.ParentType, &databaseID, &p.RowPosition, &p.Title, &p.Icon, &p.Cover, &p.CoverY, &propsJSON, &p.IsTemplate, &p.IsArchived, &p.CreatedBy, &p.CreatedAt, &p.UpdatedBy, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if databaseID.Valid {
			p.DatabaseID = databaseID.String
		}
		json.Unmarshal([]byte(propsJSON), &p.Properties)
		result = append(result, &p)
	}
	return result, rows.Err()
}

// Database row operations (pages with database_id set)

func (s *PagesStore) ListByDatabase(ctx context.Context, databaseID string, limit int, cursor string) ([]*pages.Page, error) {
	query := `
		SELECT id, workspace_id, parent_id, parent_type, database_id, row_position, title, icon, cover, cover_y, CAST(properties AS VARCHAR), is_template, is_archived, created_by, created_at, updated_by, updated_at
		FROM pages
		WHERE database_id = ? AND is_archived = FALSE
	`
	args := []interface{}{databaseID}

	if cursor != "" {
		query += " AND id > ?"
		args = append(args, cursor)
	}

	query += " ORDER BY row_position ASC, created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPages(rows)
}

func (s *PagesStore) CountByDatabase(ctx context.Context, databaseID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pages WHERE database_id = ? AND is_archived = FALSE", databaseID).Scan(&count)
	return count, err
}

func (s *PagesStore) DeleteByDatabase(ctx context.Context, databaseID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM pages WHERE database_id = ?", databaseID)
	return err
}

func (s *PagesStore) GetMaxRowPosition(ctx context.Context, databaseID string) (int64, error) {
	var maxPos sql.NullInt64
	err := s.db.QueryRowContext(ctx, "SELECT MAX(row_position) FROM pages WHERE database_id = ?", databaseID).Scan(&maxPos)
	if err != nil {
		return 0, err
	}
	if !maxPos.Valid {
		return 0, nil
	}
	return maxPos.Int64, nil
}

// nullString returns a sql.NullString for optional string values.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
