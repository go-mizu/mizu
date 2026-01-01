package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// File represents a file record.
type File struct {
	ID          string
	UserID      string
	ParentID    sql.NullString
	Name        string
	MimeType    string
	Size        int64
	StorageKey  string
	Checksum    sql.NullString
	Description sql.NullString
	IsStarred   bool
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	TrashedAt   sql.NullTime
}

// FileVersion represents a file version record.
type FileVersion struct {
	ID         string
	FileID     string
	Version    int
	Size       int64
	StorageKey string
	Checksum   sql.NullString
	CreatedBy  string
	CreatedAt  time.Time
}

// CreateFile inserts a new file.
func (s *Store) CreateFile(ctx context.Context, f *File) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO files (id, user_id, parent_id, name, mime_type, size, storage_key, checksum, description, is_starred, version, created_at, updated_at, trashed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, f.ID, f.UserID, f.ParentID, f.Name, f.MimeType, f.Size, f.StorageKey, f.Checksum, f.Description, f.IsStarred, f.Version, f.CreatedAt, f.UpdatedAt, f.TrashedAt)
	return err
}

// GetFileByID retrieves a file by ID.
func (s *Store) GetFileByID(ctx context.Context, id string) (*File, error) {
	f := &File{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, parent_id, name, mime_type, size, storage_key, checksum, description, is_starred, version, created_at, updated_at, trashed_at
		FROM files WHERE id = ?
	`, id).Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.MimeType, &f.Size, &f.StorageKey, &f.Checksum, &f.Description, &f.IsStarred, &f.Version, &f.CreatedAt, &f.UpdatedAt, &f.TrashedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return f, err
}

// UpdateFile updates a file.
func (s *Store) UpdateFile(ctx context.Context, f *File) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE files SET user_id = ?, parent_id = ?, name = ?, mime_type = ?, size = ?, storage_key = ?, checksum = ?, description = ?, is_starred = ?, version = ?, updated_at = ?, trashed_at = ?
		WHERE id = ?
	`, f.UserID, f.ParentID, f.Name, f.MimeType, f.Size, f.StorageKey, f.Checksum, f.Description, f.IsStarred, f.Version, f.UpdatedAt, f.TrashedAt, f.ID)
	return err
}

// DeleteFile permanently deletes a file.
func (s *Store) DeleteFile(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM files WHERE id = ?`, id)
	return err
}

// ListFilesByUser lists all files for a user in a specific folder.
func (s *Store) ListFilesByUser(ctx context.Context, userID, parentID string) ([]*File, error) {
	var rows *sql.Rows
	var err error

	if parentID == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, user_id, parent_id, name, mime_type, size, storage_key, checksum, description, is_starred, version, created_at, updated_at, trashed_at
			FROM files WHERE user_id = ? AND parent_id IS NULL AND trashed_at IS NULL ORDER BY name
		`, userID)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, user_id, parent_id, name, mime_type, size, storage_key, checksum, description, is_starred, version, created_at, updated_at, trashed_at
			FROM files WHERE user_id = ? AND parent_id = ? AND trashed_at IS NULL ORDER BY name
		`, userID, parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFiles(rows)
}

// ListStarredFiles lists all starred files for a user.
func (s *Store) ListStarredFiles(ctx context.Context, userID string) ([]*File, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, parent_id, name, mime_type, size, storage_key, checksum, description, is_starred, version, created_at, updated_at, trashed_at
		FROM files WHERE user_id = ? AND is_starred = TRUE AND trashed_at IS NULL ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFiles(rows)
}

// ListRecentFiles lists recently modified files for a user.
func (s *Store) ListRecentFiles(ctx context.Context, userID string, limit int) ([]*File, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, parent_id, name, mime_type, size, storage_key, checksum, description, is_starred, version, created_at, updated_at, trashed_at
		FROM files WHERE user_id = ? AND trashed_at IS NULL ORDER BY updated_at DESC LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFiles(rows)
}

// ListTrashedFiles lists trashed files for a user.
func (s *Store) ListTrashedFiles(ctx context.Context, userID string) ([]*File, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, parent_id, name, mime_type, size, storage_key, checksum, description, is_starred, version, created_at, updated_at, trashed_at
		FROM files WHERE user_id = ? AND trashed_at IS NOT NULL ORDER BY trashed_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFiles(rows)
}

// SearchFiles searches files by name.
func (s *Store) SearchFiles(ctx context.Context, userID, query string) ([]*File, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, parent_id, name, mime_type, size, storage_key, checksum, description, is_starred, version, created_at, updated_at, trashed_at
		FROM files WHERE user_id = ? AND trashed_at IS NULL AND LOWER(name) LIKE LOWER(?) ORDER BY name
	`, userID, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFiles(rows)
}

// TrashFile moves a file to trash.
func (s *Store) TrashFile(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE files SET trashed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// RestoreFile restores a file from trash.
func (s *Store) RestoreFile(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE files SET trashed_at = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// StarFile stars a file.
func (s *Store) StarFile(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE files SET is_starred = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// UnstarFile unstars a file.
func (s *Store) UnstarFile(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE files SET is_starred = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// CreateFileVersion inserts a new file version.
func (s *Store) CreateFileVersion(ctx context.Context, v *FileVersion) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO file_versions (id, file_id, version, size, storage_key, checksum, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, v.ID, v.FileID, v.Version, v.Size, v.StorageKey, v.Checksum, v.CreatedBy, v.CreatedAt)
	return err
}

// ListFileVersions lists all versions of a file.
func (s *Store) ListFileVersions(ctx context.Context, fileID string) ([]*FileVersion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_id, version, size, storage_key, checksum, created_by, created_at
		FROM file_versions WHERE file_id = ? ORDER BY version DESC
	`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*FileVersion
	for rows.Next() {
		v := &FileVersion{}
		if err := rows.Scan(&v.ID, &v.FileID, &v.Version, &v.Size, &v.StorageKey, &v.Checksum, &v.CreatedBy, &v.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

// GetFileVersion retrieves a specific file version.
func (s *Store) GetFileVersion(ctx context.Context, fileID string, version int) (*FileVersion, error) {
	v := &FileVersion{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, file_id, version, size, storage_key, checksum, created_by, created_at
		FROM file_versions WHERE file_id = ? AND version = ?
	`, fileID, version).Scan(&v.ID, &v.FileID, &v.Version, &v.Size, &v.StorageKey, &v.Checksum, &v.CreatedBy, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return v, err
}

// DeleteFileVersions deletes all versions of a file.
func (s *Store) DeleteFileVersions(ctx context.Context, fileID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM file_versions WHERE file_id = ?`, fileID)
	return err
}

// GetUserStorageUsed calculates total storage used by a user.
func (s *Store) GetUserStorageUsed(ctx context.Context, userID string) (int64, error) {
	var total sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT SUM(size) FROM files WHERE user_id = ?`, userID).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total.Int64, nil
}

func scanFiles(rows *sql.Rows) ([]*File, error) {
	var files []*File
	for rows.Next() {
		f := &File{}
		if err := rows.Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.MimeType, &f.Size, &f.StorageKey, &f.Checksum, &f.Description, &f.IsStarred, &f.Version, &f.CreatedAt, &f.UpdatedAt, &f.TrashedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}
