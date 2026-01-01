package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/drive/feature/folders"
)

// FoldersStore handles folder persistence.
type FoldersStore struct {
	db *sql.DB
}

// Create inserts a new folder.
func (s *FoldersStore) Create(ctx context.Context, f *folders.Folder) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO folders (id, owner_id, parent_id, name, path, depth, color, is_root, starred, trashed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.OwnerID, nullString(f.ParentID), f.Name, f.Path, f.Depth, f.Color, f.IsRoot, f.Starred, f.Trashed, f.CreatedAt, f.UpdatedAt)
	return err
}

// GetByID retrieves a folder by ID.
func (s *FoldersStore) GetByID(ctx context.Context, id string) (*folders.Folder, error) {
	f := &folders.Folder{}
	var parentID, color sql.NullString
	var trashedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, owner_id, parent_id, name, path, depth, color, is_root, starred, trashed, trashed_at, created_at, updated_at
		FROM folders WHERE id = ?`, id).Scan(
		&f.ID, &f.OwnerID, &parentID, &f.Name, &f.Path, &f.Depth, &color, &f.IsRoot, &f.Starred, &f.Trashed, &trashedAt, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, folders.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	f.ParentID = parentID.String
	f.Color = color.String
	if trashedAt.Valid {
		f.TrashedAt = &trashedAt.Time
	}

	return f, nil
}

// GetByOwnerAndParentAndName retrieves a folder by owner, parent, and name.
func (s *FoldersStore) GetByOwnerAndParentAndName(ctx context.Context, ownerID, parentID, name string) (*folders.Folder, error) {
	f := &folders.Folder{}
	var pID, color sql.NullString
	var trashedAt sql.NullTime

	var err error
	if parentID == "" {
		err = s.db.QueryRowContext(ctx, `
			SELECT id, owner_id, parent_id, name, path, depth, color, is_root, starred, trashed, trashed_at, created_at, updated_at
			FROM folders WHERE owner_id = ? AND parent_id IS NULL AND name = ?`, ownerID, name).Scan(
			&f.ID, &f.OwnerID, &pID, &f.Name, &f.Path, &f.Depth, &color, &f.IsRoot, &f.Starred, &f.Trashed, &trashedAt, &f.CreatedAt, &f.UpdatedAt)
	} else {
		err = s.db.QueryRowContext(ctx, `
			SELECT id, owner_id, parent_id, name, path, depth, color, is_root, starred, trashed, trashed_at, created_at, updated_at
			FROM folders WHERE owner_id = ? AND parent_id = ? AND name = ?`, ownerID, parentID, name).Scan(
			&f.ID, &f.OwnerID, &pID, &f.Name, &f.Path, &f.Depth, &color, &f.IsRoot, &f.Starred, &f.Trashed, &trashedAt, &f.CreatedAt, &f.UpdatedAt)
	}

	if err == sql.ErrNoRows {
		return nil, folders.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	f.ParentID = pID.String
	f.Color = color.String
	if trashedAt.Valid {
		f.TrashedAt = &trashedAt.Time
	}

	return f, nil
}

// GetRoot retrieves the root folder for a user.
func (s *FoldersStore) GetRoot(ctx context.Context, ownerID string) (*folders.Folder, error) {
	f := &folders.Folder{}
	var parentID, color sql.NullString
	var trashedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, owner_id, parent_id, name, path, depth, color, is_root, starred, trashed, trashed_at, created_at, updated_at
		FROM folders WHERE owner_id = ? AND is_root = TRUE`, ownerID).Scan(
		&f.ID, &f.OwnerID, &parentID, &f.Name, &f.Path, &f.Depth, &color, &f.IsRoot, &f.Starred, &f.Trashed, &trashedAt, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, folders.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	f.ParentID = parentID.String
	f.Color = color.String
	if trashedAt.Valid {
		f.TrashedAt = &trashedAt.Time
	}

	return f, nil
}

// List lists folders.
func (s *FoldersStore) List(ctx context.Context, ownerID string, in *folders.ListIn) ([]*folders.Folder, error) {
	query := `SELECT id, owner_id, parent_id, name, path, depth, color, is_root, starred, trashed, trashed_at, created_at, updated_at
		FROM folders WHERE owner_id = ? AND trashed = ?`
	args := []any{ownerID, in.Trashed}

	if in.ParentID != "" {
		query += " AND parent_id = ?"
		args = append(args, in.ParentID)
	}

	if in.Starred != nil {
		query += " AND starred = ?"
		args = append(args, *in.Starred)
	}

	// Order
	orderBy := "name"
	if in.OrderBy != "" {
		orderBy = in.OrderBy
	}
	order := "ASC"
	if in.Order == "desc" {
		order = "DESC"
	}
	query += " ORDER BY " + orderBy + " " + order

	// Pagination
	if in.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, in.Limit)
	}
	if in.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, in.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}

// ListByParent lists folders by parent.
func (s *FoldersStore) ListByParent(ctx context.Context, ownerID, parentID string) ([]*folders.Folder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_id, parent_id, name, path, depth, color, is_root, starred, trashed, trashed_at, created_at, updated_at
		FROM folders WHERE owner_id = ? AND parent_id = ? ORDER BY name`,
		ownerID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}

// ListDescendants lists all descendant folders.
func (s *FoldersStore) ListDescendants(ctx context.Context, id string) ([]*folders.Folder, error) {
	folder, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_id, parent_id, name, path, depth, color, is_root, starred, trashed, trashed_at, created_at, updated_at
		FROM folders WHERE path LIKE ? AND id != ?`,
		folder.Path+"/%", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}

// Update updates a folder.
func (s *FoldersStore) Update(ctx context.Context, id string, in *folders.UpdateIn) error {
	if in.Name != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE folders SET name = ?, updated_at = ? WHERE id = ?`, *in.Name, time.Now(), id); err != nil {
			return err
		}
	}
	if in.Color != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE folders SET color = ?, updated_at = ? WHERE id = ?`, *in.Color, time.Now(), id); err != nil {
			return err
		}
	}
	return nil
}

// UpdatePath updates folder path and depth.
func (s *FoldersStore) UpdatePath(ctx context.Context, id, path string, depth int) error {
	_, err := s.db.ExecContext(ctx, `UPDATE folders SET path = ?, depth = ?, updated_at = ? WHERE id = ?`, path, depth, time.Now(), id)
	return err
}

// UpdateParent updates folder parent.
func (s *FoldersStore) UpdateParent(ctx context.Context, id, parentID, path string, depth int) error {
	_, err := s.db.ExecContext(ctx, `UPDATE folders SET parent_id = ?, path = ?, depth = ?, updated_at = ? WHERE id = ?`, parentID, path, depth, time.Now(), id)
	return err
}

// UpdateTrashed updates trashed status.
func (s *FoldersStore) UpdateTrashed(ctx context.Context, id string, trashed bool) error {
	var trashedAt any
	if trashed {
		trashedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `UPDATE folders SET trashed = ?, trashed_at = ?, updated_at = ? WHERE id = ?`, trashed, trashedAt, time.Now(), id)
	return err
}

// UpdateStarred updates starred status.
func (s *FoldersStore) UpdateStarred(ctx context.Context, id string, starred bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE folders SET starred = ?, updated_at = ? WHERE id = ?`, starred, time.Now(), id)
	return err
}

// Delete deletes a folder.
func (s *FoldersStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM folders WHERE id = ?`, id)
	return err
}

func scanFolders(rows *sql.Rows) ([]*folders.Folder, error) {
	var result []*folders.Folder
	for rows.Next() {
		f := &folders.Folder{}
		var parentID, color sql.NullString
		var trashedAt sql.NullTime

		if err := rows.Scan(&f.ID, &f.OwnerID, &parentID, &f.Name, &f.Path, &f.Depth, &color, &f.IsRoot, &f.Starred, &f.Trashed, &trashedAt, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}

		f.ParentID = parentID.String
		f.Color = color.String
		if trashedAt.Valid {
			f.TrashedAt = &trashedAt.Time
		}

		result = append(result, f)
	}
	return result, rows.Err()
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
