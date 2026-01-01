package files

import (
	"bytes"
	"context"
	"io"
	"math"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/drive/feature/accounts"
	"github.com/go-mizu/blueprints/drive/feature/folders"
	"github.com/go-mizu/blueprints/drive/pkg/hash"
	mimepkg "github.com/go-mizu/blueprints/drive/pkg/mime"
	"github.com/go-mizu/blueprints/drive/pkg/ulid"
	"github.com/go-mizu/blueprints/drive/storage"
)

const (
	DefaultChunkSize = 5 * 1024 * 1024   // 5 MB
	MinChunkSize     = 1 * 1024 * 1024   // 1 MB
	MaxChunkSize     = 100 * 1024 * 1024 // 100 MB
	UploadExpiry     = 24 * time.Hour
	DefaultLockDur   = 1 * time.Hour
)

// Service implements the files API.
type Service struct {
	store        Store
	versionStore VersionStore
	uploadStore  ChunkedUploadStore
	folderSvc    folders.API
	accountSvc   accounts.API
	storage      storage.Storage
}

// NewService creates a new files service.
func NewService(store Store, versionStore VersionStore, uploadStore ChunkedUploadStore, folderSvc folders.API, accountSvc accounts.API, storage storage.Storage) *Service {
	return &Service{
		store:        store,
		versionStore: versionStore,
		uploadStore:  uploadStore,
		folderSvc:    folderSvc,
		accountSvc:   accountSvc,
		storage:      storage,
	}
}

// Upload uploads a single file.
func (s *Service) Upload(ctx context.Context, ownerID string, in *UploadIn) (*File, error) {
	// Check quota
	if err := s.checkQuota(ctx, ownerID, in.Size); err != nil {
		return nil, err
	}

	// Get folder
	var folderPath string
	folderID := in.FolderID
	if folderID == "" {
		root, err := s.folderSvc.EnsureRoot(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		folderID = root.ID
		folderPath = root.Path
	} else {
		folder, err := s.folderSvc.GetByID(ctx, folderID)
		if err != nil {
			return nil, err
		}
		folderPath = folder.Path
	}

	// Detect MIME type
	var buf bytes.Buffer
	tee := io.TeeReader(in.Reader, &buf)

	// Read first 512 bytes for detection
	header := make([]byte, 512)
	n, _ := io.ReadFull(tee, header)
	header = header[:n]

	mimeType := mimepkg.DetectFromBytes(header)
	if in.MimeType != "" {
		mimeType = in.MimeType
	}

	// Create multi-reader with header bytes
	fullReader := io.MultiReader(&buf, in.Reader)

	// Generate file ID
	fileID := ulid.New()

	// Save to storage
	storagePath, err := s.storage.Save(ctx, ownerID, fileID, fullReader, in.Size)
	if err != nil {
		return nil, err
	}

	// Get extension
	ext := filepath.Ext(in.Filename)
	if ext != "" {
		ext = strings.TrimPrefix(ext, ".")
	}

	now := time.Now()
	file := &File{
		ID:             fileID,
		OwnerID:        ownerID,
		FolderID:       folderID,
		Name:           in.Filename,
		Path:           path.Join(folderPath, in.Filename),
		Size:           in.Size,
		MimeType:       mimeType,
		Extension:      ext,
		StoragePath:    storagePath,
		HasThumbnail:   false,
		Starred:        false,
		Trashed:        false,
		Locked:         false,
		VersionCount:   1,
		CurrentVersion: 1,
		Description:    in.Description,
		CreatedAt:      now,
		UpdatedAt:      now,
		AccessedAt:     now,
	}

	if err := s.store.Create(ctx, file); err != nil {
		s.storage.Delete(ctx, storagePath)
		return nil, err
	}

	// Update storage usage
	s.accountSvc.UpdateStorageUsed(ctx, ownerID, in.Size)

	return file, nil
}

// GetByID retrieves a file by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*File, error) {
	return s.store.GetByID(ctx, id)
}

// List lists files.
func (s *Service) List(ctx context.Context, ownerID string, in *ListIn) ([]*File, error) {
	return s.store.List(ctx, ownerID, in)
}

// ListRecent lists recently accessed files.
func (s *Service) ListRecent(ctx context.Context, ownerID string, limit int) ([]*File, error) {
	return s.store.ListRecent(ctx, ownerID, limit)
}

// Update updates file metadata.
func (s *Service) Update(ctx context.Context, id, ownerID string, in *UpdateIn) (*File, error) {
	file, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if file.OwnerID != ownerID {
		return nil, ErrNotOwner
	}

	if file.Locked && file.LockedBy != ownerID {
		return nil, ErrFileLocked
	}

	// Check name collision
	if in.Name != nil && *in.Name != file.Name {
		if existing, _ := s.store.GetByOwnerAndFolderAndName(ctx, ownerID, file.FolderID, *in.Name); existing != nil {
			return nil, ErrNameTaken
		}
	}

	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}

	return s.store.GetByID(ctx, id)
}

// Move moves a file to a different folder.
func (s *Service) Move(ctx context.Context, id, ownerID, newFolderID string) (*File, error) {
	file, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if file.OwnerID != ownerID {
		return nil, ErrNotOwner
	}

	// Get new folder
	folder, err := s.folderSvc.GetByID(ctx, newFolderID)
	if err != nil {
		return nil, err
	}

	// Check name collision
	if existing, _ := s.store.GetByOwnerAndFolderAndName(ctx, ownerID, newFolderID, file.Name); existing != nil {
		return nil, ErrNameTaken
	}

	newPath := path.Join(folder.Path, file.Name)
	if err := s.store.UpdateFolder(ctx, id, newFolderID, newPath); err != nil {
		return nil, err
	}

	return s.store.GetByID(ctx, id)
}

// Copy copies a file.
func (s *Service) Copy(ctx context.Context, id, ownerID, destFolderID string, newName string) (*File, error) {
	file, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Open source file
	src, err := s.storage.Open(ctx, file.StoragePath)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	if newName == "" {
		newName = file.Name
	}

	return s.Upload(ctx, ownerID, &UploadIn{
		Filename:    newName,
		FolderID:    destFolderID,
		Description: file.Description,
		MimeType:    file.MimeType,
		Size:        file.Size,
		Reader:      src,
	})
}

// Delete moves a file to trash.
func (s *Service) Delete(ctx context.Context, id, ownerID string) error {
	file, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if file.OwnerID != ownerID {
		return ErrNotOwner
	}

	return s.store.UpdateTrashed(ctx, id, true)
}

// Star stars or unstars a file.
func (s *Service) Star(ctx context.Context, id, ownerID string, starred bool) error {
	file, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if file.OwnerID != ownerID {
		return ErrNotOwner
	}

	return s.store.UpdateStarred(ctx, id, starred)
}

// Lock locks a file.
func (s *Service) Lock(ctx context.Context, id, ownerID string, duration time.Duration) error {
	file, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if file.OwnerID != ownerID {
		return ErrNotOwner
	}

	if file.Locked && file.LockedBy != ownerID {
		return ErrFileLocked
	}

	if duration == 0 {
		duration = DefaultLockDur
	}

	expiresAt := time.Now().Add(duration)
	return s.store.UpdateLock(ctx, id, true, ownerID, &expiresAt)
}

// Unlock unlocks a file.
func (s *Service) Unlock(ctx context.Context, id, ownerID string) error {
	file, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Owner can always unlock, or lock holder
	if file.OwnerID != ownerID && file.LockedBy != ownerID {
		return ErrNotOwner
	}

	return s.store.UpdateLock(ctx, id, false, "", nil)
}

// Open opens a file for reading.
func (s *Service) Open(ctx context.Context, id string) (io.ReadCloser, *File, error) {
	file, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.storage.Open(ctx, file.StoragePath)
	if err != nil {
		return nil, nil, err
	}

	return reader, file, nil
}

// UpdateAccessed updates the accessed timestamp.
func (s *Service) UpdateAccessed(ctx context.Context, id string) error {
	return s.store.UpdateAccessed(ctx, id)
}

// CreateChunkedUpload creates a chunked upload session.
func (s *Service) CreateChunkedUpload(ctx context.Context, ownerID string, in *CreateChunkedUploadIn) (*ChunkedUpload, error) {
	// Check quota
	if err := s.checkQuota(ctx, ownerID, in.Size); err != nil {
		return nil, err
	}

	chunkSize := DefaultChunkSize
	if in.ChunkSize > 0 {
		chunkSize = clamp(in.ChunkSize, MinChunkSize, MaxChunkSize)
	}

	totalChunks := int(math.Ceil(float64(in.Size) / float64(chunkSize)))

	uploadID := ulid.New()
	tempPath, err := s.storage.CreateChunkDir(ctx, uploadID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	upload := &ChunkedUpload{
		ID:          uploadID,
		AccountID:   ownerID,
		FolderID:    in.FolderID,
		Filename:    in.Filename,
		TotalSize:   in.Size,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		MimeType:    in.MimeType,
		Status:      "pending",
		TempPath:    tempPath,
		ExpiresAt:   now.Add(UploadExpiry),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.uploadStore.Create(ctx, upload); err != nil {
		s.storage.CleanupChunks(ctx, uploadID)
		return nil, err
	}

	return upload, nil
}

// UploadChunk uploads a single chunk.
func (s *Service) UploadChunk(ctx context.Context, uploadID string, index int, r io.Reader, size int64, checksum string) error {
	upload, err := s.uploadStore.GetByID(ctx, uploadID)
	if err != nil {
		return ErrUploadNotFound
	}

	if time.Now().After(upload.ExpiresAt) {
		return ErrUploadExpired
	}

	if index < 0 || index >= upload.TotalChunks {
		return ErrInvalidUpload
	}

	// Save chunk
	if err := s.storage.SaveChunk(ctx, uploadID, index, r, size); err != nil {
		return err
	}

	// Record chunk
	chunk := &UploadChunk{
		UploadID:   uploadID,
		ChunkIndex: index,
		Size:       int(size),
		Checksum:   checksum,
		CreatedAt:  time.Now(),
	}

	if err := s.uploadStore.CreateChunk(ctx, chunk); err != nil {
		return err
	}

	// Update status
	s.uploadStore.UpdateStatus(ctx, uploadID, "uploading")

	return nil
}

// GetUploadProgress returns upload progress.
func (s *Service) GetUploadProgress(ctx context.Context, uploadID string) (*UploadProgress, error) {
	upload, err := s.uploadStore.GetByID(ctx, uploadID)
	if err != nil {
		return nil, ErrUploadNotFound
	}

	chunks, err := s.uploadStore.GetChunks(ctx, uploadID)
	if err != nil {
		return nil, err
	}

	uploadedSet := make(map[int]bool)
	var uploaded []int
	for _, c := range chunks {
		uploadedSet[c.ChunkIndex] = true
		uploaded = append(uploaded, c.ChunkIndex)
	}

	var missing []int
	for i := 0; i < upload.TotalChunks; i++ {
		if !uploadedSet[i] {
			missing = append(missing, i)
		}
	}

	progress := float64(len(uploaded)) / float64(upload.TotalChunks) * 100

	return &UploadProgress{
		UploadID:       uploadID,
		Status:         upload.Status,
		TotalChunks:    upload.TotalChunks,
		UploadedChunks: uploaded,
		MissingChunks:  missing,
		ProgressPct:    progress,
		ExpiresAt:      upload.ExpiresAt,
	}, nil
}

// CompleteUpload completes a chunked upload.
func (s *Service) CompleteUpload(ctx context.Context, uploadID, ownerID string, checksum string) (*File, error) {
	upload, err := s.uploadStore.GetByID(ctx, uploadID)
	if err != nil {
		return nil, ErrUploadNotFound
	}

	if upload.AccountID != ownerID {
		return nil, ErrNotOwner
	}

	// Verify all chunks present
	chunks, err := s.uploadStore.GetChunks(ctx, uploadID)
	if err != nil {
		return nil, err
	}

	if len(chunks) != upload.TotalChunks {
		return nil, ErrChunkMissing
	}

	// Generate file ID
	fileID := ulid.New()

	// Assemble chunks
	storagePath, err := s.storage.AssembleChunks(ctx, uploadID, upload.TotalChunks, ownerID, fileID)
	if err != nil {
		return nil, err
	}

	// Compute checksum
	reader, err := s.storage.Open(ctx, storagePath)
	if err != nil {
		return nil, err
	}
	computedHash, _ := hash.SHA256Reader(reader)
	reader.Close()

	// Get folder
	var folderPath string
	folderID := upload.FolderID
	if folderID == "" {
		root, err := s.folderSvc.EnsureRoot(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		folderID = root.ID
		folderPath = root.Path
	} else {
		folder, err := s.folderSvc.GetByID(ctx, folderID)
		if err != nil {
			return nil, err
		}
		folderPath = folder.Path
	}

	// Detect MIME type
	mimeType := upload.MimeType
	if mimeType == "" {
		mimeType = mimepkg.DetectFromFilename(upload.Filename)
	}

	ext := filepath.Ext(upload.Filename)
	if ext != "" {
		ext = strings.TrimPrefix(ext, ".")
	}

	now := time.Now()
	file := &File{
		ID:             fileID,
		OwnerID:        ownerID,
		FolderID:       folderID,
		Name:           upload.Filename,
		Path:           path.Join(folderPath, upload.Filename),
		Size:           upload.TotalSize,
		MimeType:       mimeType,
		Extension:      ext,
		StoragePath:    storagePath,
		ChecksumSHA256: computedHash,
		HasThumbnail:   false,
		Starred:        false,
		Trashed:        false,
		Locked:         false,
		VersionCount:   1,
		CurrentVersion: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
		AccessedAt:     now,
	}

	if err := s.store.Create(ctx, file); err != nil {
		s.storage.Delete(ctx, storagePath)
		return nil, err
	}

	// Cleanup
	s.storage.CleanupChunks(ctx, uploadID)
	s.uploadStore.DeleteChunks(ctx, uploadID)
	s.uploadStore.Delete(ctx, uploadID)

	// Update storage usage
	s.accountSvc.UpdateStorageUsed(ctx, ownerID, upload.TotalSize)

	return file, nil
}

// CancelUpload cancels a chunked upload.
func (s *Service) CancelUpload(ctx context.Context, uploadID, ownerID string) error {
	upload, err := s.uploadStore.GetByID(ctx, uploadID)
	if err != nil {
		return ErrUploadNotFound
	}

	if upload.AccountID != ownerID {
		return ErrNotOwner
	}

	s.storage.CleanupChunks(ctx, uploadID)
	s.uploadStore.DeleteChunks(ctx, uploadID)
	return s.uploadStore.Delete(ctx, uploadID)
}

// ListVersions lists file versions.
func (s *Service) ListVersions(ctx context.Context, fileID string) ([]*FileVersion, error) {
	return s.versionStore.ListByFile(ctx, fileID)
}

// UploadVersion uploads a new version.
func (s *Service) UploadVersion(ctx context.Context, fileID, ownerID string, r io.Reader, size int64, comment string) (*FileVersion, error) {
	file, err := s.store.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if file.OwnerID != ownerID {
		return nil, ErrNotOwner
	}

	// Check quota
	if err := s.checkQuota(ctx, ownerID, size); err != nil {
		return nil, err
	}

	newVersion := file.CurrentVersion + 1

	// Save version
	storagePath, err := s.storage.SaveVersion(ctx, ownerID, fileID, newVersion, r, size)
	if err != nil {
		return nil, err
	}

	version := &FileVersion{
		ID:            ulid.New(),
		FileID:        fileID,
		VersionNumber: newVersion,
		Size:          size,
		StoragePath:   storagePath,
		UploadedBy:    ownerID,
		Comment:       comment,
		CreatedAt:     time.Now(),
	}

	if err := s.versionStore.Create(ctx, version); err != nil {
		s.storage.Delete(ctx, storagePath)
		return nil, err
	}

	// Update file
	if err := s.store.UpdateVersion(ctx, fileID, file.VersionCount+1, newVersion); err != nil {
		return nil, err
	}

	// Update storage
	s.accountSvc.UpdateStorageUsed(ctx, ownerID, size)

	return version, nil
}

// OpenVersion opens a specific version.
func (s *Service) OpenVersion(ctx context.Context, fileID string, version int) (io.ReadCloser, *FileVersion, error) {
	v, err := s.versionStore.GetByFileAndVersion(ctx, fileID, version)
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.storage.Open(ctx, v.StoragePath)
	if err != nil {
		return nil, nil, err
	}

	return reader, v, nil
}

// RestoreVersion restores a previous version as current.
func (s *Service) RestoreVersion(ctx context.Context, fileID, ownerID string, version int) (*File, error) {
	file, err := s.store.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if file.OwnerID != ownerID {
		return nil, ErrNotOwner
	}

	v, err := s.versionStore.GetByFileAndVersion(ctx, fileID, version)
	if err != nil {
		return nil, err
	}

	// Open version
	reader, err := s.storage.Open(ctx, v.StoragePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Upload as new version
	_, err = s.UploadVersion(ctx, fileID, ownerID, reader, v.Size, "Restored from version "+string(rune('0'+version)))
	if err != nil {
		return nil, err
	}

	return s.store.GetByID(ctx, fileID)
}

func (s *Service) checkQuota(ctx context.Context, ownerID string, additionalBytes int64) error {
	usage, err := s.accountSvc.GetStorageUsage(ctx, ownerID)
	if err != nil {
		return err
	}

	if usage.Used+additionalBytes > usage.Quota {
		return ErrQuotaExceeded
	}

	return nil
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
