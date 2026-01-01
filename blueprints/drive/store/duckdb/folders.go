package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// Folder represents a folder record.
type Folder struct {
	ID          string
	UserID      string
	ParentID    sql.NullString
	Name        string
	Description sql.NullString
	Color       sql.NullString
	IsStarred   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	TrashedAt   sql.NullTime
}

// CreateFolder inserts a new folder.
func (s *Store) CreateFolder(ctx context.Context, f *Folder) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO folders (id, user_id, parent_id, name, description, color, is_starred, created_at, updated_at, trashed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, f.ID, f.UserID, f.ParentID, f.Name, f.Description, f.Color, f.IsStarred, f.CreatedAt, f.UpdatedAt, f.TrashedAt)
	return err
}

// GetFolderByID retrieves a folder by ID.
func (s *Store) GetFolderByID(ctx context.Context, id string) (*Folder, error) {
	f := &Folder{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, parent_id, name, description, color, is_starred, created_at, updated_at, trashed_at
		FROM folders WHERE id = ?
	`, id).Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.Description, &f.Color, &f.IsStarred, &f.CreatedAt, &f.UpdatedAt, &f.TrashedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return f, err
}

// UpdateFolder updates a folder.
func (s *Store) UpdateFolder(ctx context.Context, f *Folder) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE folders SET user_id = ?, parent_id = ?, name = ?, description = ?, color = ?, is_starred = ?, updated_at = ?, trashed_at = ?
		WHERE id = ?
	`, f.UserID, f.ParentID, f.Name, f.Description, f.Color, f.IsStarred, f.UpdatedAt, f.TrashedAt, f.ID)
	return err
}

// DeleteFolder permanently deletes a folder.
func (s *Store) DeleteFolder(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM folders WHERE id = ?`, id)
	return err
}

// ListFoldersByUser lists all folders for a user in a specific parent folder.
func (s *Store) ListFoldersByUser(ctx context.Context, userID, parentID string) ([]*Folder, error) {
	var rows *sql.Rows
	var err error

	if parentID == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, user_id, parent_id, name, description, color, is_starred, created_at, updated_at, trashed_at
			FROM folders WHERE user_id = ? AND parent_id IS NULL AND trashed_at IS NULL ORDER BY name
		`, userID)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, user_id, parent_id, name, description, color, is_starred, created_at, updated_at, trashed_at
			FROM folders WHERE user_id = ? AND parent_id = ? AND trashed_at IS NULL ORDER BY name
		`, userID, parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}

// ListAllFoldersByUser lists all folders for a user.
func (s *Store) ListAllFoldersByUser(ctx context.Context, userID string) ([]*Folder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, parent_id, name, description, color, is_starred, created_at, updated_at, trashed_at
		FROM folders WHERE user_id = ? AND trashed_at IS NULL ORDER BY name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}

// ListStarredFolders lists all starred folders for a user.
func (s *Store) ListStarredFolders(ctx context.Context, userID string) ([]*Folder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, parent_id, name, description, color, is_starred, created_at, updated_at, trashed_at
		FROM folders WHERE user_id = ? AND is_starred = TRUE AND trashed_at IS NULL ORDER BY name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}

// ListTrashedFolders lists trashed folders for a user.
func (s *Store) ListTrashedFolders(ctx context.Context, userID string) ([]*Folder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, parent_id, name, description, color, is_starred, created_at, updated_at, trashed_at
		FROM folders WHERE user_id = ? AND trashed_at IS NOT NULL ORDER BY trashed_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}

// SearchFolders searches folders by name.
func (s *Store) SearchFolders(ctx context.Context, userID, query string) ([]*Folder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, parent_id, name, description, color, is_starred, created_at, updated_at, trashed_at
		FROM folders WHERE user_id = ? AND trashed_at IS NULL AND LOWER(name) LIKE LOWER(?) ORDER BY name
	`, userID, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}

// TrashFolder moves a folder to trash.
func (s *Store) TrashFolder(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE folders SET trashed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// RestoreFolder restores a folder from trash.
func (s *Store) RestoreFolder(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE folders SET trashed_at = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// StarFolder stars a folder.
func (s *Store) StarFolder(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE folders SET is_starred = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// UnstarFolder unstars a folder.
func (s *Store) UnstarFolder(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE folders SET is_starred = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// GetFolderPath returns the path from root to folder.
func (s *Store) GetFolderPath(ctx context.Context, id string) ([]*Folder, error) {
	var path []*Folder
	currentID := id

	for currentID != "" {
		folder, err := s.GetFolderByID(ctx, currentID)
		if err != nil {
			return nil, err
		}
		if folder == nil {
			break
		}
		path = append([]*Folder{folder}, path...)
		currentID = folder.ParentID.String
		if !folder.ParentID.Valid {
			break
		}
	}

	return path, nil
}

// ListChildFolderIDs returns all descendant folder IDs (for recursive operations).
func (s *Store) ListChildFolderIDs(ctx context.Context, id string) ([]string, error) {
	var ids []string
	toProcess := []string{id}

	for len(toProcess) > 0 {
		currentID := toProcess[0]
		toProcess = toProcess[1:]

		rows, err := s.db.QueryContext(ctx, `SELECT id FROM folders WHERE parent_id = ?`, currentID)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var childID string
			if err := rows.Scan(&childID); err != nil {
				rows.Close()
				return nil, err
			}
			ids = append(ids, childID)
			toProcess = append(toProcess, childID)
		}
		rows.Close()
	}

	return ids, nil
}

func scanFolders(rows *sql.Rows) ([]*Folder, error) {
	var folders []*Folder
	for rows.Next() {
		f := &Folder{}
		if err := rows.Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.Description, &f.Color, &f.IsStarred, &f.CreatedAt, &f.UpdatedAt, &f.TrashedAt); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}
