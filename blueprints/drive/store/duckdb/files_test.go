package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// ============================================================
// File CRUD Tests
// ============================================================

func TestCreateFile_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	file.MimeType = "application/pdf"
	file.Size = 2048
	file.Checksum = sql.NullString{String: "sha256:abc123", Valid: true}
	file.Description = sql.NullString{String: "Important document", Valid: true}

	if err := store.CreateFile(ctx, file); err != nil {
		t.Fatalf("create file failed: %v", err)
	}

	got, err := store.GetFileByID(ctx, "file1")
	if err != nil {
		t.Fatalf("get file failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected file, got nil")
	}
	if got.Name != "document.pdf" {
		t.Errorf("expected name document.pdf, got %s", got.Name)
	}
	if got.MimeType != "application/pdf" {
		t.Errorf("expected mime_type application/pdf, got %s", got.MimeType)
	}
	if got.Size != 2048 {
		t.Errorf("expected size 2048, got %d", got.Size)
	}
}

func TestCreateFile_WithParent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	file := newTestFile("file1", "user1", "readme.txt")
	file.ParentID = sql.NullString{String: "folder1", Valid: true}

	if err := store.CreateFile(ctx, file); err != nil {
		t.Fatalf("create file failed: %v", err)
	}

	got, _ := store.GetFileByID(ctx, "file1")
	if !got.ParentID.Valid || got.ParentID.String != "folder1" {
		t.Errorf("expected parent_id folder1, got %v", got.ParentID)
	}
}

func TestGetFileByID_Exists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	store.CreateFile(ctx, file)

	got, err := store.GetFileByID(ctx, "file1")
	if err != nil {
		t.Fatalf("get file failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected file, got nil")
	}
	if got.ID != "file1" {
		t.Errorf("expected ID file1, got %s", got.ID)
	}
}

func TestGetFileByID_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	got, err := store.GetFileByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get file failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestUpdateFile_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "original.txt")
	store.CreateFile(ctx, file)

	file.Name = "renamed.txt"
	file.Size = 4096
	file.Version = 2
	file.IsStarred = true
	file.UpdatedAt = time.Now().Truncate(time.Microsecond)

	if err := store.UpdateFile(ctx, file); err != nil {
		t.Fatalf("update file failed: %v", err)
	}

	got, _ := store.GetFileByID(ctx, "file1")
	if got.Name != "renamed.txt" {
		t.Errorf("expected name renamed.txt, got %s", got.Name)
	}
	if got.Size != 4096 {
		t.Errorf("expected size 4096, got %d", got.Size)
	}
	if got.Version != 2 {
		t.Errorf("expected version 2, got %d", got.Version)
	}
	if !got.IsStarred {
		t.Error("expected is_starred true")
	}
}

func TestDeleteFile_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	store.CreateFile(ctx, file)

	if err := store.DeleteFile(ctx, "file1"); err != nil {
		t.Fatalf("delete file failed: %v", err)
	}

	got, _ := store.GetFileByID(ctx, "file1")
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

// ============================================================
// File Listing Tests
// ============================================================

func TestListFilesByUser_RootLevel(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file1 := newTestFile("file1", "user1", "aaa.txt")
	file2 := newTestFile("file2", "user1", "bbb.txt")
	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)

	files, err := store.ListFilesByUser(ctx, "user1", "")
	if err != nil {
		t.Fatalf("list files failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// Should be sorted by name
	if files[0].Name != "aaa.txt" {
		t.Errorf("expected first file aaa.txt, got %s", files[0].Name)
	}
}

func TestListFilesByUser_InFolder(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	file1 := newTestFile("file1", "user1", "root.txt")
	file2 := newTestFile("file2", "user1", "in_folder.txt")
	file2.ParentID = sql.NullString{String: "folder1", Valid: true}

	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)

	// List root files
	rootFiles, _ := store.ListFilesByUser(ctx, "user1", "")
	if len(rootFiles) != 1 {
		t.Errorf("expected 1 root file, got %d", len(rootFiles))
	}

	// List folder files
	folderFiles, _ := store.ListFilesByUser(ctx, "user1", "folder1")
	if len(folderFiles) != 1 {
		t.Errorf("expected 1 folder file, got %d", len(folderFiles))
	}
	if folderFiles[0].Name != "in_folder.txt" {
		t.Errorf("expected in_folder.txt, got %s", folderFiles[0].Name)
	}
}

func TestListFilesByUser_ExcludesTrashed(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file1 := newTestFile("file1", "user1", "normal.txt")
	file2 := newTestFile("file2", "user1", "trashed.txt")
	file2.TrashedAt = sql.NullTime{Time: time.Now(), Valid: true}

	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)

	files, _ := store.ListFilesByUser(ctx, "user1", "")
	if len(files) != 1 {
		t.Errorf("expected 1 file (excluding trashed), got %d", len(files))
	}
	if files[0].Name != "normal.txt" {
		t.Errorf("expected normal.txt, got %s", files[0].Name)
	}
}

func TestListFilesByUser_SortedByName(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file1 := newTestFile("file1", "user1", "zebra.txt")
	file2 := newTestFile("file2", "user1", "apple.txt")
	file3 := newTestFile("file3", "user1", "banana.txt")

	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)
	store.CreateFile(ctx, file3)

	files, _ := store.ListFilesByUser(ctx, "user1", "")
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	if files[0].Name != "apple.txt" || files[1].Name != "banana.txt" || files[2].Name != "zebra.txt" {
		t.Errorf("files not sorted by name: %s, %s, %s", files[0].Name, files[1].Name, files[2].Name)
	}
}

// ============================================================
// File Features Tests
// ============================================================

func TestListStarredFiles(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file1 := newTestFile("file1", "user1", "normal.txt")
	file2 := newTestFile("file2", "user1", "starred.txt")
	file2.IsStarred = true

	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)

	files, err := store.ListStarredFiles(ctx, "user1")
	if err != nil {
		t.Fatalf("list starred files failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 starred file, got %d", len(files))
	}
	if files[0].Name != "starred.txt" {
		t.Errorf("expected starred.txt, got %s", files[0].Name)
	}
}

func TestListRecentFiles(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file1 := newTestFile("file1", "user1", "old.txt")
	file1.UpdatedAt = time.Now().Add(-24 * time.Hour).Truncate(time.Microsecond)

	file2 := newTestFile("file2", "user1", "recent.txt")
	file2.UpdatedAt = time.Now().Truncate(time.Microsecond)

	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)

	files, err := store.ListRecentFiles(ctx, "user1", 10)
	if err != nil {
		t.Fatalf("list recent files failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	// Should be ordered by updated_at DESC
	if files[0].Name != "recent.txt" {
		t.Errorf("expected recent.txt first, got %s", files[0].Name)
	}
}

func TestListRecentFiles_Limit(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	for i := 1; i <= 10; i++ {
		file := newTestFile("file"+string(rune('0'+i)), "user1", "file"+string(rune('0'+i))+".txt")
		store.CreateFile(ctx, file)
	}

	files, _ := store.ListRecentFiles(ctx, "user1", 5)
	if len(files) != 5 {
		t.Errorf("expected 5 files with limit, got %d", len(files))
	}
}

func TestListTrashedFiles(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file1 := newTestFile("file1", "user1", "normal.txt")
	file2 := newTestFile("file2", "user1", "trashed.txt")
	file2.TrashedAt = sql.NullTime{Time: time.Now(), Valid: true}

	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)

	files, err := store.ListTrashedFiles(ctx, "user1")
	if err != nil {
		t.Fatalf("list trashed files failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 trashed file, got %d", len(files))
	}
	if files[0].Name != "trashed.txt" {
		t.Errorf("expected trashed.txt, got %s", files[0].Name)
	}
}

func TestSearchFiles_NameMatch(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file1 := newTestFile("file1", "user1", "document.pdf")
	file2 := newTestFile("file2", "user1", "photo.jpg")
	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)

	files, err := store.SearchFiles(ctx, "user1", "document")
	if err != nil {
		t.Fatalf("search files failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 match, got %d", len(files))
	}
	if files[0].Name != "document.pdf" {
		t.Errorf("expected document.pdf, got %s", files[0].Name)
	}
}

func TestSearchFiles_CaseInsensitive(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "Document.PDF")
	store.CreateFile(ctx, file)

	files, _ := store.SearchFiles(ctx, "user1", "DOCUMENT")
	if len(files) != 1 {
		t.Errorf("expected case-insensitive match, got %d", len(files))
	}

	files, _ = store.SearchFiles(ctx, "user1", "document")
	if len(files) != 1 {
		t.Errorf("expected case-insensitive match, got %d", len(files))
	}
}

func TestSearchFiles_PartialMatch(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "important_document_2024.pdf")
	store.CreateFile(ctx, file)

	files, _ := store.SearchFiles(ctx, "user1", "document")
	if len(files) != 1 {
		t.Errorf("expected partial match, got %d", len(files))
	}

	files, _ = store.SearchFiles(ctx, "user1", "2024")
	if len(files) != 1 {
		t.Errorf("expected partial match, got %d", len(files))
	}
}

func TestSearchFiles_ExcludesTrashed(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	file.TrashedAt = sql.NullTime{Time: time.Now(), Valid: true}
	store.CreateFile(ctx, file)

	files, _ := store.SearchFiles(ctx, "user1", "document")
	if len(files) != 0 {
		t.Errorf("expected 0 matches (trashed excluded), got %d", len(files))
	}
}

// ============================================================
// Trash Operations Tests
// ============================================================

func TestTrashFile(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	store.CreateFile(ctx, file)

	if err := store.TrashFile(ctx, "file1"); err != nil {
		t.Fatalf("trash file failed: %v", err)
	}

	got, _ := store.GetFileByID(ctx, "file1")
	if !got.TrashedAt.Valid {
		t.Error("expected trashed_at to be set")
	}
}

func TestRestoreFile(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	file.TrashedAt = sql.NullTime{Time: time.Now(), Valid: true}
	store.CreateFile(ctx, file)

	if err := store.RestoreFile(ctx, "file1"); err != nil {
		t.Fatalf("restore file failed: %v", err)
	}

	got, _ := store.GetFileByID(ctx, "file1")
	if got.TrashedAt.Valid {
		t.Error("expected trashed_at to be cleared")
	}
}

func TestTrashRestore_Roundtrip(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	store.CreateFile(ctx, file)

	// Initially not trashed
	got, _ := store.GetFileByID(ctx, "file1")
	if got.TrashedAt.Valid {
		t.Error("file should not be trashed initially")
	}

	// Trash it
	store.TrashFile(ctx, "file1")
	got, _ = store.GetFileByID(ctx, "file1")
	if !got.TrashedAt.Valid {
		t.Error("file should be trashed")
	}

	// Restore it
	store.RestoreFile(ctx, "file1")
	got, _ = store.GetFileByID(ctx, "file1")
	if got.TrashedAt.Valid {
		t.Error("file should not be trashed after restore")
	}
}

// ============================================================
// Star Operations Tests
// ============================================================

func TestStarFile(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	store.CreateFile(ctx, file)

	if err := store.StarFile(ctx, "file1"); err != nil {
		t.Fatalf("star file failed: %v", err)
	}

	got, _ := store.GetFileByID(ctx, "file1")
	if !got.IsStarred {
		t.Error("expected is_starred true")
	}
}

func TestUnstarFile(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	file.IsStarred = true
	store.CreateFile(ctx, file)

	if err := store.UnstarFile(ctx, "file1"); err != nil {
		t.Fatalf("unstar file failed: %v", err)
	}

	got, _ := store.GetFileByID(ctx, "file1")
	if got.IsStarred {
		t.Error("expected is_starred false")
	}
}

func TestStarUnstar_Roundtrip(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	store.CreateFile(ctx, file)

	// Initially not starred
	got, _ := store.GetFileByID(ctx, "file1")
	if got.IsStarred {
		t.Error("file should not be starred initially")
	}

	// Star it
	store.StarFile(ctx, "file1")
	got, _ = store.GetFileByID(ctx, "file1")
	if !got.IsStarred {
		t.Error("file should be starred")
	}

	// Unstar it
	store.UnstarFile(ctx, "file1")
	got, _ = store.GetFileByID(ctx, "file1")
	if got.IsStarred {
		t.Error("file should not be starred after unstar")
	}
}

// ============================================================
// File Version Tests
// ============================================================

func TestCreateFileVersion_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	store.CreateFile(ctx, file)

	version := &FileVersion{
		ID:         "ver1",
		FileID:     "file1",
		Version:    1,
		Size:       1024,
		StorageKey: "storage/file1/v1",
		Checksum:   sql.NullString{String: "sha256:abc", Valid: true},
		CreatedBy:  "user1",
		CreatedAt:  time.Now().Truncate(time.Microsecond),
	}

	if err := store.CreateFileVersion(ctx, version); err != nil {
		t.Fatalf("create file version failed: %v", err)
	}

	got, err := store.GetFileVersion(ctx, "file1", 1)
	if err != nil {
		t.Fatalf("get file version failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected version, got nil")
	}
	if got.Size != 1024 {
		t.Errorf("expected size 1024, got %d", got.Size)
	}
}

func TestListFileVersions(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	store.CreateFile(ctx, file)

	for i := 1; i <= 3; i++ {
		version := &FileVersion{
			ID:         "ver" + string(rune('0'+i)),
			FileID:     "file1",
			Version:    i,
			Size:       int64(i * 1024),
			StorageKey: "storage/file1/v" + string(rune('0'+i)),
			CreatedBy:  "user1",
			CreatedAt:  time.Now().Truncate(time.Microsecond),
		}
		store.CreateFileVersion(ctx, version)
	}

	versions, err := store.ListFileVersions(ctx, "file1")
	if err != nil {
		t.Fatalf("list file versions failed: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}

	// Should be ordered by version DESC
	if versions[0].Version != 3 {
		t.Errorf("expected first version to be 3, got %d", versions[0].Version)
	}
	if versions[2].Version != 1 {
		t.Errorf("expected last version to be 1, got %d", versions[2].Version)
	}
}

func TestGetFileVersion(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	store.CreateFile(ctx, file)

	version := &FileVersion{
		ID:         "ver1",
		FileID:     "file1",
		Version:    1,
		Size:       1024,
		StorageKey: "storage/file1/v1",
		CreatedBy:  "user1",
		CreatedAt:  time.Now().Truncate(time.Microsecond),
	}
	store.CreateFileVersion(ctx, version)

	got, err := store.GetFileVersion(ctx, "file1", 1)
	if err != nil {
		t.Fatalf("get file version failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected version, got nil")
	}
	if got.Version != 1 {
		t.Errorf("expected version 1, got %d", got.Version)
	}
}

func TestGetFileVersion_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	got, err := store.GetFileVersion(ctx, "nonexistent", 1)
	if err != nil {
		t.Fatalf("get file version failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestDeleteFileVersions(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "test.txt")
	store.CreateFile(ctx, file)

	for i := 1; i <= 3; i++ {
		version := &FileVersion{
			ID:         "ver" + string(rune('0'+i)),
			FileID:     "file1",
			Version:    i,
			Size:       int64(i * 1024),
			StorageKey: "storage/file1/v" + string(rune('0'+i)),
			CreatedBy:  "user1",
			CreatedAt:  time.Now().Truncate(time.Microsecond),
		}
		store.CreateFileVersion(ctx, version)
	}

	if err := store.DeleteFileVersions(ctx, "file1"); err != nil {
		t.Fatalf("delete file versions failed: %v", err)
	}

	versions, _ := store.ListFileVersions(ctx, "file1")
	if len(versions) != 0 {
		t.Errorf("expected 0 versions after delete, got %d", len(versions))
	}
}

// ============================================================
// Storage Calculations Tests
// ============================================================

func TestGetUserStorageUsed_Empty(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	used, err := store.GetUserStorageUsed(ctx, "user1")
	if err != nil {
		t.Fatalf("get storage used failed: %v", err)
	}
	if used != 0 {
		t.Errorf("expected 0, got %d", used)
	}
}

func TestGetUserStorageUsed_MultipleFiles(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file1 := newTestFile("file1", "user1", "file1.txt")
	file1.Size = 1000
	file2 := newTestFile("file2", "user1", "file2.txt")
	file2.Size = 2000
	file3 := newTestFile("file3", "user1", "file3.txt")
	file3.Size = 3000

	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)
	store.CreateFile(ctx, file3)

	used, err := store.GetUserStorageUsed(ctx, "user1")
	if err != nil {
		t.Fatalf("get storage used failed: %v", err)
	}
	if used != 6000 {
		t.Errorf("expected 6000, got %d", used)
	}
}

// ============================================================
// Business Use Cases
// ============================================================

func TestFileLifecycle_UploadEditTrashDelete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Upload
	file := newTestFile("file1", "user1", "document.docx")
	file.Size = 5000
	if err := store.CreateFile(ctx, file); err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	// Edit (update metadata)
	file.Name = "important_document.docx"
	file.Version = 2
	if err := store.UpdateFile(ctx, file); err != nil {
		t.Fatalf("edit failed: %v", err)
	}

	// Verify edit
	got, _ := store.GetFileByID(ctx, "file1")
	if got.Name != "important_document.docx" {
		t.Errorf("expected renamed file, got %s", got.Name)
	}

	// Trash
	if err := store.TrashFile(ctx, "file1"); err != nil {
		t.Fatalf("trash failed: %v", err)
	}

	// Verify trashed (not in normal listing)
	files, _ := store.ListFilesByUser(ctx, "user1", "")
	if len(files) != 0 {
		t.Error("trashed file should not appear in listing")
	}

	// Verify in trash listing
	trashed, _ := store.ListTrashedFiles(ctx, "user1")
	if len(trashed) != 1 {
		t.Error("file should appear in trash listing")
	}

	// Permanently delete
	if err := store.DeleteFile(ctx, "file1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify deleted
	got, _ = store.GetFileByID(ctx, "file1")
	if got != nil {
		t.Error("file should be permanently deleted")
	}
}

func TestFileVersioning_MultipleEdits(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.txt")
	store.CreateFile(ctx, file)

	// Simulate multiple edits with version history
	for i := 1; i <= 5; i++ {
		version := &FileVersion{
			ID:         "ver" + string(rune('0'+i)),
			FileID:     "file1",
			Version:    i,
			Size:       int64(i * 100),
			StorageKey: "storage/file1/v" + string(rune('0'+i)),
			CreatedBy:  "user1",
			CreatedAt:  time.Now().Add(time.Duration(i) * time.Minute).Truncate(time.Microsecond),
		}
		store.CreateFileVersion(ctx, version)
	}

	// Verify version history
	versions, _ := store.ListFileVersions(ctx, "file1")
	if len(versions) != 5 {
		t.Errorf("expected 5 versions, got %d", len(versions))
	}

	// Can retrieve specific version
	v3, _ := store.GetFileVersion(ctx, "file1", 3)
	if v3 == nil || v3.Size != 300 {
		t.Error("should be able to retrieve specific version")
	}
}

func TestFileOrganization_MoveToFolder(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	file := newTestFile("file1", "user1", "report.pdf")
	store.CreateFile(ctx, file)

	// Initially at root
	rootFiles, _ := store.ListFilesByUser(ctx, "user1", "")
	if len(rootFiles) != 1 {
		t.Error("file should be at root")
	}

	// Move to folder
	file.ParentID = sql.NullString{String: "folder1", Valid: true}
	store.UpdateFile(ctx, file)

	// Verify moved
	rootFiles, _ = store.ListFilesByUser(ctx, "user1", "")
	if len(rootFiles) != 0 {
		t.Error("file should no longer be at root")
	}

	folderFiles, _ := store.ListFilesByUser(ctx, "user1", "folder1")
	if len(folderFiles) != 1 {
		t.Error("file should be in folder")
	}
}

func TestFileDuplication_SameFolder(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Same name but different IDs is allowed (simulating "Copy of" behavior)
	file1 := newTestFile("file1", "user1", "document.pdf")
	file2 := newTestFile("file2", "user1", "document.pdf")

	if err := store.CreateFile(ctx, file1); err != nil {
		t.Fatalf("create file1 failed: %v", err)
	}
	if err := store.CreateFile(ctx, file2); err != nil {
		t.Fatalf("create file2 failed: %v", err)
	}

	files, _ := store.ListFilesByUser(ctx, "user1", "")
	if len(files) != 2 {
		t.Errorf("expected 2 files with same name, got %d", len(files))
	}
}
