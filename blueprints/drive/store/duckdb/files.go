package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/drive/feature/files"
)

// FilesStore handles file persistence.
type FilesStore struct {
	db *sql.DB
}

// Create inserts a new file.
func (s *FilesStore) Create(ctx context.Context, f *files.File) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO files (id, owner_id, folder_id, name, path, size, mime_type, extension, storage_path,
			checksum_sha256, has_thumbnail, thumbnail_path, starred, trashed, locked, version_count,
			current_version, description, metadata, created_at, updated_at, accessed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.OwnerID, nullString(f.FolderID), f.Name, f.Path, f.Size, f.MimeType, f.Extension, f.StoragePath,
		f.ChecksumSHA256, f.HasThumbnail, f.ThumbnailPath, f.Starred, f.Trashed, f.Locked, f.VersionCount,
		f.CurrentVersion, f.Description, f.Metadata, f.CreatedAt, f.UpdatedAt, f.AccessedAt)
	return err
}

// GetByID retrieves a file by ID.
func (s *FilesStore) GetByID(ctx context.Context, id string) (*files.File, error) {
	f := &files.File{}
	var folderID, checksum, thumbnailPath, lockedBy, description, metadata sql.NullString
	var trashedAt, lockedAt, lockExpiresAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, owner_id, folder_id, name, path, size, mime_type, extension, storage_path,
			checksum_sha256, has_thumbnail, thumbnail_path, starred, trashed, trashed_at, locked, locked_by,
			locked_at, lock_expires_at, version_count, current_version, description, metadata,
			created_at, updated_at, accessed_at
		FROM files WHERE id = ?`, id).Scan(
		&f.ID, &f.OwnerID, &folderID, &f.Name, &f.Path, &f.Size, &f.MimeType, &f.Extension, &f.StoragePath,
		&checksum, &f.HasThumbnail, &thumbnailPath, &f.Starred, &f.Trashed, &trashedAt, &f.Locked, &lockedBy,
		&lockedAt, &lockExpiresAt, &f.VersionCount, &f.CurrentVersion, &description, &metadata,
		&f.CreatedAt, &f.UpdatedAt, &f.AccessedAt)

	if err == sql.ErrNoRows {
		return nil, files.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	f.FolderID = folderID.String
	f.ChecksumSHA256 = checksum.String
	f.ThumbnailPath = thumbnailPath.String
	f.LockedBy = lockedBy.String
	f.Description = description.String
	f.Metadata = metadata.String

	if trashedAt.Valid {
		f.TrashedAt = &trashedAt.Time
	}
	if lockedAt.Valid {
		f.LockedAt = &lockedAt.Time
	}
	if lockExpiresAt.Valid {
		f.LockExpiresAt = &lockExpiresAt.Time
	}

	return f, nil
}

// GetByOwnerAndFolderAndName retrieves a file by owner, folder, and name.
func (s *FilesStore) GetByOwnerAndFolderAndName(ctx context.Context, ownerID, folderID, name string) (*files.File, error) {
	var query string
	var args []any

	if folderID == "" {
		query = `SELECT id FROM files WHERE owner_id = ? AND folder_id IS NULL AND name = ?`
		args = []any{ownerID, name}
	} else {
		query = `SELECT id FROM files WHERE owner_id = ? AND folder_id = ? AND name = ?`
		args = []any{ownerID, folderID, name}
	}

	var id string
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, files.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}

// List lists files.
func (s *FilesStore) List(ctx context.Context, ownerID string, in *files.ListIn) ([]*files.File, error) {
	query := `SELECT id FROM files WHERE owner_id = ? AND trashed = ?`
	args := []any{ownerID, in.Trashed}

	if in.FolderID != "" {
		query += " AND folder_id = ?"
		args = append(args, in.FolderID)
	}

	if in.Starred != nil {
		query += " AND starred = ?"
		args = append(args, *in.Starred)
	}

	if in.MimeType != "" {
		query += " AND mime_type LIKE ?"
		args = append(args, in.MimeType+"%")
	}

	orderBy := "name"
	if in.OrderBy != "" {
		orderBy = in.OrderBy
	}
	order := "ASC"
	if in.Order == "desc" {
		order = "DESC"
	}
	query += " ORDER BY " + orderBy + " " + order

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

	var result []*files.File
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		f, err := s.GetByID(ctx, id)
		if err != nil {
			continue
		}
		result = append(result, f)
	}

	return result, rows.Err()
}

// ListRecent lists recently accessed files.
func (s *FilesStore) ListRecent(ctx context.Context, ownerID string, limit int) ([]*files.File, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM files WHERE owner_id = ? AND trashed = FALSE
		ORDER BY accessed_at DESC LIMIT ?`, ownerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*files.File
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		f, err := s.GetByID(ctx, id)
		if err != nil {
			continue
		}
		result = append(result, f)
	}

	return result, rows.Err()
}

// Update updates file metadata.
func (s *FilesStore) Update(ctx context.Context, id string, in *files.UpdateIn) error {
	if in.Name != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE files SET name = ?, updated_at = ? WHERE id = ?`, *in.Name, time.Now(), id); err != nil {
			return err
		}
	}
	if in.Description != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE files SET description = ?, updated_at = ? WHERE id = ?`, *in.Description, time.Now(), id); err != nil {
			return err
		}
	}
	return nil
}

// UpdateFolder updates file folder.
func (s *FilesStore) UpdateFolder(ctx context.Context, id, folderID, path string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE files SET folder_id = ?, path = ?, updated_at = ? WHERE id = ?`, folderID, path, time.Now(), id)
	return err
}

// UpdateTrashed updates trashed status.
func (s *FilesStore) UpdateTrashed(ctx context.Context, id string, trashed bool) error {
	var trashedAt any
	if trashed {
		trashedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `UPDATE files SET trashed = ?, trashed_at = ?, updated_at = ? WHERE id = ?`, trashed, trashedAt, time.Now(), id)
	return err
}

// UpdateStarred updates starred status.
func (s *FilesStore) UpdateStarred(ctx context.Context, id string, starred bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE files SET starred = ?, updated_at = ? WHERE id = ?`, starred, time.Now(), id)
	return err
}

// UpdateLock updates lock status.
func (s *FilesStore) UpdateLock(ctx context.Context, id string, locked bool, lockedBy string, expiresAt *time.Time) error {
	var lockedAt any
	if locked {
		lockedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `UPDATE files SET locked = ?, locked_by = ?, locked_at = ?, lock_expires_at = ?, updated_at = ? WHERE id = ?`,
		locked, nullString(lockedBy), lockedAt, expiresAt, time.Now(), id)
	return err
}

// UpdateVersion updates version info.
func (s *FilesStore) UpdateVersion(ctx context.Context, id string, versionCount, currentVersion int) error {
	_, err := s.db.ExecContext(ctx, `UPDATE files SET version_count = ?, current_version = ?, updated_at = ? WHERE id = ?`,
		versionCount, currentVersion, time.Now(), id)
	return err
}

// UpdateAccessed updates accessed timestamp.
func (s *FilesStore) UpdateAccessed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE files SET accessed_at = ? WHERE id = ?`, time.Now(), id)
	return err
}

// Delete deletes a file.
func (s *FilesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM files WHERE id = ?`, id)
	return err
}

// FileVersionsStore handles file version persistence.
type FileVersionsStore struct {
	db *sql.DB
}

// Create inserts a new version.
func (s *FileVersionsStore) Create(ctx context.Context, v *files.FileVersion) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO file_versions (id, file_id, version_number, size, storage_path, checksum_sha256, uploaded_by, comment, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.FileID, v.VersionNumber, v.Size, v.StoragePath, v.ChecksumSHA256, v.UploadedBy, v.Comment, v.CreatedAt)
	return err
}

// GetByFileAndVersion retrieves a version.
func (s *FileVersionsStore) GetByFileAndVersion(ctx context.Context, fileID string, version int) (*files.FileVersion, error) {
	v := &files.FileVersion{}
	var checksum, comment sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, file_id, version_number, size, storage_path, checksum_sha256, uploaded_by, comment, created_at
		FROM file_versions WHERE file_id = ? AND version_number = ?`, fileID, version).Scan(
		&v.ID, &v.FileID, &v.VersionNumber, &v.Size, &v.StoragePath, &checksum, &v.UploadedBy, &comment, &v.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, files.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	v.ChecksumSHA256 = checksum.String
	v.Comment = comment.String

	return v, nil
}

// ListByFile lists versions for a file.
func (s *FileVersionsStore) ListByFile(ctx context.Context, fileID string) ([]*files.FileVersion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_id, version_number, size, storage_path, checksum_sha256, uploaded_by, comment, created_at
		FROM file_versions WHERE file_id = ? ORDER BY version_number DESC`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*files.FileVersion
	for rows.Next() {
		v := &files.FileVersion{}
		var checksum, comment sql.NullString

		if err := rows.Scan(&v.ID, &v.FileID, &v.VersionNumber, &v.Size, &v.StoragePath, &checksum, &v.UploadedBy, &comment, &v.CreatedAt); err != nil {
			return nil, err
		}

		v.ChecksumSHA256 = checksum.String
		v.Comment = comment.String
		result = append(result, v)
	}

	return result, rows.Err()
}

// Delete deletes a version.
func (s *FileVersionsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM file_versions WHERE id = ?`, id)
	return err
}

// ChunkedUploadsStore handles chunked upload persistence.
type ChunkedUploadsStore struct {
	db *sql.DB
}

// Create inserts a new upload.
func (s *ChunkedUploadsStore) Create(ctx context.Context, u *files.ChunkedUpload) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO chunked_uploads (id, account_id, folder_id, filename, total_size, chunk_size, total_chunks,
			mime_type, status, temp_path, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.AccountID, nullString(u.FolderID), u.Filename, u.TotalSize, u.ChunkSize, u.TotalChunks,
		u.MimeType, u.Status, u.TempPath, u.ExpiresAt, u.CreatedAt, u.UpdatedAt)
	return err
}

// GetByID retrieves an upload.
func (s *ChunkedUploadsStore) GetByID(ctx context.Context, id string) (*files.ChunkedUpload, error) {
	u := &files.ChunkedUpload{}
	var folderID, mimeType, tempPath sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_id, folder_id, filename, total_size, chunk_size, total_chunks,
			mime_type, status, temp_path, expires_at, created_at, updated_at
		FROM chunked_uploads WHERE id = ?`, id).Scan(
		&u.ID, &u.AccountID, &folderID, &u.Filename, &u.TotalSize, &u.ChunkSize, &u.TotalChunks,
		&mimeType, &u.Status, &tempPath, &u.ExpiresAt, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, files.ErrUploadNotFound
	}
	if err != nil {
		return nil, err
	}

	u.FolderID = folderID.String
	u.MimeType = mimeType.String
	u.TempPath = tempPath.String

	return u, nil
}

// UpdateStatus updates upload status.
func (s *ChunkedUploadsStore) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE chunked_uploads SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now(), id)
	return err
}

// Delete deletes an upload.
func (s *ChunkedUploadsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM chunked_uploads WHERE id = ?`, id)
	return err
}

// CreateChunk inserts a chunk record.
func (s *ChunkedUploadsStore) CreateChunk(ctx context.Context, c *files.UploadChunk) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO upload_chunks (upload_id, chunk_index, size, checksum, storage_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (upload_id, chunk_index) DO UPDATE SET size = ?, checksum = ?, storage_path = ?`,
		c.UploadID, c.ChunkIndex, c.Size, c.Checksum, c.StoragePath, c.CreatedAt,
		c.Size, c.Checksum, c.StoragePath)
	return err
}

// GetChunks retrieves all chunks for an upload.
func (s *ChunkedUploadsStore) GetChunks(ctx context.Context, uploadID string) ([]*files.UploadChunk, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT upload_id, chunk_index, size, checksum, storage_path, created_at
		FROM upload_chunks WHERE upload_id = ? ORDER BY chunk_index`, uploadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*files.UploadChunk
	for rows.Next() {
		c := &files.UploadChunk{}
		var checksum sql.NullString

		if err := rows.Scan(&c.UploadID, &c.ChunkIndex, &c.Size, &checksum, &c.StoragePath, &c.CreatedAt); err != nil {
			return nil, err
		}

		c.Checksum = checksum.String
		result = append(result, c)
	}

	return result, rows.Err()
}

// DeleteChunks deletes all chunks for an upload.
func (s *ChunkedUploadsStore) DeleteChunks(ctx context.Context, uploadID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM upload_chunks WHERE upload_id = ?`, uploadID)
	return err
}
