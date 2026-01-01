package folders

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/drive/pkg/ulid"
	"github.com/go-mizu/blueprints/drive/store/duckdb"
)

var (
	ErrNotFound     = errors.New("folder not found")
	ErrMissingName  = errors.New("name is required")
	ErrUnauthorized = errors.New("unauthorized")
)

// Service implements the folders API.
type Service struct {
	store *duckdb.Store
}

// NewService creates a new folders service.
func NewService(store *duckdb.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, userID string, in *CreateIn) (*Folder, error) {
	if in.Name == "" {
		return nil, ErrMissingName
	}

	now := time.Now()
	dbFolder := &duckdb.Folder{
		ID:          ulid.New(),
		UserID:      userID,
		ParentID:    sql.NullString{String: in.ParentID, Valid: in.ParentID != ""},
		Name:        in.Name,
		Description: sql.NullString{String: in.Description, Valid: in.Description != ""},
		Color:       sql.NullString{String: in.Color, Valid: in.Color != ""},
		IsStarred:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.CreateFolder(ctx, dbFolder); err != nil {
		return nil, err
	}

	return dbFolderToFolder(dbFolder), nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Folder, error) {
	dbFolder, err := s.store.GetFolderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbFolder == nil {
		return nil, ErrNotFound
	}
	return dbFolderToFolder(dbFolder), nil
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Folder, error) {
	dbFolder, err := s.store.GetFolderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbFolder == nil {
		return nil, ErrNotFound
	}

	if in.Name != nil {
		dbFolder.Name = *in.Name
	}
	if in.Description != nil {
		dbFolder.Description = sql.NullString{String: *in.Description, Valid: true}
	}
	if in.Color != nil {
		dbFolder.Color = sql.NullString{String: *in.Color, Valid: true}
	}
	dbFolder.UpdatedAt = time.Now()

	if err := s.store.UpdateFolder(ctx, dbFolder); err != nil {
		return nil, err
	}

	return dbFolderToFolder(dbFolder), nil
}

func (s *Service) Move(ctx context.Context, id string, in *MoveIn) (*Folder, error) {
	dbFolder, err := s.store.GetFolderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbFolder == nil {
		return nil, ErrNotFound
	}

	dbFolder.ParentID = sql.NullString{String: in.ParentID, Valid: in.ParentID != ""}
	dbFolder.UpdatedAt = time.Now()

	if err := s.store.UpdateFolder(ctx, dbFolder); err != nil {
		return nil, err
	}

	return dbFolderToFolder(dbFolder), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	// Get all child folders
	childIDs, err := s.store.ListChildFolderIDs(ctx, id)
	if err != nil {
		return err
	}

	// Delete children first
	for _, childID := range childIDs {
		if err := s.store.DeleteFolder(ctx, childID); err != nil {
			return err
		}
	}

	// Delete the folder
	return s.store.DeleteFolder(ctx, id)
}

func (s *Service) Trash(ctx context.Context, id string) error {
	dbFolder, err := s.store.GetFolderByID(ctx, id)
	if err != nil {
		return err
	}
	if dbFolder == nil {
		return ErrNotFound
	}

	// Trash all child folders
	childIDs, err := s.store.ListChildFolderIDs(ctx, id)
	if err != nil {
		return err
	}
	for _, childID := range childIDs {
		_ = s.store.TrashFolder(ctx, childID)
	}

	return s.store.TrashFolder(ctx, id)
}

func (s *Service) Restore(ctx context.Context, id string) error {
	// Restore child folders
	childIDs, err := s.store.ListChildFolderIDs(ctx, id)
	if err != nil {
		return err
	}
	for _, childID := range childIDs {
		_ = s.store.RestoreFolder(ctx, childID)
	}

	return s.store.RestoreFolder(ctx, id)
}

func (s *Service) Star(ctx context.Context, id, userID string) error {
	dbFolder, err := s.store.GetFolderByID(ctx, id)
	if err != nil {
		return err
	}
	if dbFolder == nil {
		return ErrNotFound
	}
	return s.store.StarFolder(ctx, id)
}

func (s *Service) Unstar(ctx context.Context, id, userID string) error {
	return s.store.UnstarFolder(ctx, id)
}

func (s *Service) ListByUser(ctx context.Context, userID string) ([]*Folder, error) {
	dbFolders, err := s.store.ListAllFoldersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dbFoldersToFolders(dbFolders), nil
}

func (s *Service) ListByParent(ctx context.Context, userID, parentID string) ([]*Folder, error) {
	dbFolders, err := s.store.ListFoldersByUser(ctx, userID, parentID)
	if err != nil {
		return nil, err
	}
	return dbFoldersToFolders(dbFolders), nil
}

func (s *Service) ListStarred(ctx context.Context, userID string) ([]*Folder, error) {
	dbFolders, err := s.store.ListStarredFolders(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dbFoldersToFolders(dbFolders), nil
}

func (s *Service) ListTrashed(ctx context.Context, userID string) ([]*Folder, error) {
	dbFolders, err := s.store.ListTrashedFolders(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dbFoldersToFolders(dbFolders), nil
}

func (s *Service) Search(ctx context.Context, userID, query string) ([]*Folder, error) {
	dbFolders, err := s.store.SearchFolders(ctx, userID, query)
	if err != nil {
		return nil, err
	}
	return dbFoldersToFolders(dbFolders), nil
}

func (s *Service) GetPath(ctx context.Context, id string) ([]*Folder, error) {
	dbFolders, err := s.store.GetFolderPath(ctx, id)
	if err != nil {
		return nil, err
	}
	return dbFoldersToFolders(dbFolders), nil
}

func dbFolderToFolder(f *duckdb.Folder) *Folder {
	folder := &Folder{
		ID:          f.ID,
		UserID:      f.UserID,
		Name:        f.Name,
		Description: f.Description.String,
		Color:       f.Color.String,
		IsStarred:   f.IsStarred,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
	if f.ParentID.Valid {
		folder.ParentID = f.ParentID.String
	}
	if f.TrashedAt.Valid {
		folder.TrashedAt = f.TrashedAt.Time
	}
	return folder
}

func dbFoldersToFolders(dbFolders []*duckdb.Folder) []*Folder {
	folders := make([]*Folder, len(dbFolders))
	for i, f := range dbFolders {
		folders[i] = dbFolderToFolder(f)
	}
	return folders
}
