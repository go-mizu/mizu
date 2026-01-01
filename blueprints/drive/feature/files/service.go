package files

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/drive/pkg/ulid"
	"github.com/go-mizu/blueprints/drive/store/duckdb"
)

var (
	ErrNotFound     = errors.New("file not found")
	ErrMissingName  = errors.New("name is required")
	ErrUnauthorized = errors.New("unauthorized")
)

// Service implements the files API.
type Service struct {
	store *duckdb.Store
}

// NewService creates a new files service.
func NewService(store *duckdb.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, userID string, in *CreateIn) (*File, error) {
	if in.Name == "" {
		return nil, ErrMissingName
	}

	now := time.Now()
	storageKey := in.StorageKey
	if storageKey == "" {
		storageKey = ulid.New()
	}

	dbFile := &duckdb.File{
		ID:          ulid.New(),
		UserID:      userID,
		ParentID:    sql.NullString{String: in.ParentID, Valid: in.ParentID != ""},
		Name:        in.Name,
		MimeType:    in.MimeType,
		Size:        in.Size,
		StorageKey:  storageKey,
		Checksum:    sql.NullString{String: in.Checksum, Valid: in.Checksum != ""},
		Description: sql.NullString{String: in.Description, Valid: in.Description != ""},
		IsStarred:   false,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.CreateFile(ctx, dbFile); err != nil {
		return nil, err
	}

	// Update user storage used
	_ = s.store.UpdateUserStorageUsed(ctx, userID, in.Size)

	return dbFileToFile(dbFile), nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*File, error) {
	dbFile, err := s.store.GetFileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbFile == nil {
		return nil, ErrNotFound
	}
	return dbFileToFile(dbFile), nil
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*File, error) {
	dbFile, err := s.store.GetFileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbFile == nil {
		return nil, ErrNotFound
	}

	if in.Name != nil {
		dbFile.Name = *in.Name
	}
	if in.Description != nil {
		dbFile.Description = sql.NullString{String: *in.Description, Valid: true}
	}
	dbFile.UpdatedAt = time.Now()

	if err := s.store.UpdateFile(ctx, dbFile); err != nil {
		return nil, err
	}

	return dbFileToFile(dbFile), nil
}

func (s *Service) Move(ctx context.Context, id string, in *MoveIn) (*File, error) {
	dbFile, err := s.store.GetFileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbFile == nil {
		return nil, ErrNotFound
	}

	dbFile.ParentID = sql.NullString{String: in.ParentID, Valid: in.ParentID != ""}
	dbFile.UpdatedAt = time.Now()

	if err := s.store.UpdateFile(ctx, dbFile); err != nil {
		return nil, err
	}

	return dbFileToFile(dbFile), nil
}

func (s *Service) Copy(ctx context.Context, id, userID string, in *CopyIn) (*File, error) {
	original, err := s.store.GetFileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if original == nil {
		return nil, ErrNotFound
	}

	name := in.Name
	if name == "" {
		name = "Copy of " + original.Name
	}

	now := time.Now()
	dbFile := &duckdb.File{
		ID:          ulid.New(),
		UserID:      userID,
		ParentID:    sql.NullString{String: in.ParentID, Valid: in.ParentID != ""},
		Name:        name,
		MimeType:    original.MimeType,
		Size:        original.Size,
		StorageKey:  original.StorageKey, // Share the same storage key
		Checksum:    original.Checksum,
		Description: original.Description,
		IsStarred:   false,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.CreateFile(ctx, dbFile); err != nil {
		return nil, err
	}

	return dbFileToFile(dbFile), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	dbFile, err := s.store.GetFileByID(ctx, id)
	if err != nil {
		return err
	}
	if dbFile == nil {
		return ErrNotFound
	}

	// Delete versions
	_ = s.store.DeleteFileVersions(ctx, id)

	// Update user storage used
	_ = s.store.UpdateUserStorageUsed(ctx, dbFile.UserID, -dbFile.Size)

	return s.store.DeleteFile(ctx, id)
}

func (s *Service) Trash(ctx context.Context, id string) error {
	dbFile, err := s.store.GetFileByID(ctx, id)
	if err != nil {
		return err
	}
	if dbFile == nil {
		return ErrNotFound
	}
	return s.store.TrashFile(ctx, id)
}

func (s *Service) Restore(ctx context.Context, id string) error {
	return s.store.RestoreFile(ctx, id)
}

func (s *Service) Star(ctx context.Context, id, userID string) error {
	dbFile, err := s.store.GetFileByID(ctx, id)
	if err != nil {
		return err
	}
	if dbFile == nil {
		return ErrNotFound
	}
	return s.store.StarFile(ctx, id)
}

func (s *Service) Unstar(ctx context.Context, id, userID string) error {
	return s.store.UnstarFile(ctx, id)
}

func (s *Service) ListByUser(ctx context.Context, userID, parentID string) ([]*File, error) {
	dbFiles, err := s.store.ListFilesByUser(ctx, userID, parentID)
	if err != nil {
		return nil, err
	}
	return dbFilesToFiles(dbFiles), nil
}

func (s *Service) ListStarred(ctx context.Context, userID string) ([]*File, error) {
	dbFiles, err := s.store.ListStarredFiles(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dbFilesToFiles(dbFiles), nil
}

func (s *Service) ListRecent(ctx context.Context, userID string, limit int) ([]*File, error) {
	if limit <= 0 {
		limit = 20
	}
	dbFiles, err := s.store.ListRecentFiles(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return dbFilesToFiles(dbFiles), nil
}

func (s *Service) ListTrashed(ctx context.Context, userID string) ([]*File, error) {
	dbFiles, err := s.store.ListTrashedFiles(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dbFilesToFiles(dbFiles), nil
}

func (s *Service) Search(ctx context.Context, userID, query string) ([]*File, error) {
	dbFiles, err := s.store.SearchFiles(ctx, userID, query)
	if err != nil {
		return nil, err
	}
	return dbFilesToFiles(dbFiles), nil
}

func (s *Service) ListVersions(ctx context.Context, fileID string) ([]*FileVersion, error) {
	dbVersions, err := s.store.ListFileVersions(ctx, fileID)
	if err != nil {
		return nil, err
	}
	versions := make([]*FileVersion, len(dbVersions))
	for i, v := range dbVersions {
		versions[i] = dbVersionToVersion(v)
	}
	return versions, nil
}

func (s *Service) GetVersion(ctx context.Context, fileID string, version int) (*FileVersion, error) {
	dbVersion, err := s.store.GetFileVersion(ctx, fileID, version)
	if err != nil {
		return nil, err
	}
	if dbVersion == nil {
		return nil, ErrNotFound
	}
	return dbVersionToVersion(dbVersion), nil
}

func (s *Service) RestoreVersion(ctx context.Context, fileID string, version int, userID string) (*File, error) {
	dbVersion, err := s.store.GetFileVersion(ctx, fileID, version)
	if err != nil {
		return nil, err
	}
	if dbVersion == nil {
		return nil, ErrNotFound
	}

	dbFile, err := s.store.GetFileByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if dbFile == nil {
		return nil, ErrNotFound
	}

	// Create a version of the current state
	now := time.Now()
	currentVersion := &duckdb.FileVersion{
		ID:         ulid.New(),
		FileID:     fileID,
		Version:    dbFile.Version,
		Size:       dbFile.Size,
		StorageKey: dbFile.StorageKey,
		Checksum:   dbFile.Checksum,
		CreatedBy:  userID,
		CreatedAt:  now,
	}
	_ = s.store.CreateFileVersion(ctx, currentVersion)

	// Restore the old version
	dbFile.Size = dbVersion.Size
	dbFile.StorageKey = dbVersion.StorageKey
	dbFile.Checksum = dbVersion.Checksum
	dbFile.Version++
	dbFile.UpdatedAt = now

	if err := s.store.UpdateFile(ctx, dbFile); err != nil {
		return nil, err
	}

	return dbFileToFile(dbFile), nil
}

func dbFileToFile(f *duckdb.File) *File {
	file := &File{
		ID:          f.ID,
		UserID:      f.UserID,
		Name:        f.Name,
		MimeType:    f.MimeType,
		Size:        f.Size,
		StorageKey:  f.StorageKey,
		Checksum:    f.Checksum.String,
		Description: f.Description.String,
		IsStarred:   f.IsStarred,
		Version:     f.Version,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
	if f.ParentID.Valid {
		file.ParentID = f.ParentID.String
	}
	if f.TrashedAt.Valid {
		file.TrashedAt = f.TrashedAt.Time
	}
	return file
}

func dbFilesToFiles(dbFiles []*duckdb.File) []*File {
	files := make([]*File, len(dbFiles))
	for i, f := range dbFiles {
		files[i] = dbFileToFile(f)
	}
	return files
}

func dbVersionToVersion(v *duckdb.FileVersion) *FileVersion {
	return &FileVersion{
		ID:         v.ID,
		FileID:     v.FileID,
		Version:    v.Version,
		Size:       v.Size,
		StorageKey: v.StorageKey,
		Checksum:   v.Checksum.String,
		CreatedBy:  v.CreatedBy,
		CreatedAt:  v.CreatedAt,
	}
}
