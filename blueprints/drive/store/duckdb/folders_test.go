package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// ============================================================
// Folder CRUD Tests
// ============================================================

func TestCreateFolder_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	folder.Description = sql.NullString{String: "My documents folder", Valid: true}
	folder.Color = sql.NullString{String: "#FF5733", Valid: true}

	if err := store.CreateFolder(ctx, folder); err != nil {
		t.Fatalf("create folder failed: %v", err)
	}

	got, err := store.GetFolderByID(ctx, "folder1")
	if err != nil {
		t.Fatalf("get folder failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected folder, got nil")
	}
	if got.Name != "Documents" {
		t.Errorf("expected name Documents, got %s", got.Name)
	}
	if got.Description.String != "My documents folder" {
		t.Errorf("expected description, got %s", got.Description.String)
	}
	if got.Color.String != "#FF5733" {
		t.Errorf("expected color #FF5733, got %s", got.Color.String)
	}
}

func TestCreateFolder_Nested(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	parent := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, parent)

	child := newTestFolder("folder2", "user1", "Work")
	child.ParentID = sql.NullString{String: "folder1", Valid: true}

	if err := store.CreateFolder(ctx, child); err != nil {
		t.Fatalf("create nested folder failed: %v", err)
	}

	got, _ := store.GetFolderByID(ctx, "folder2")
	if !got.ParentID.Valid || got.ParentID.String != "folder1" {
		t.Errorf("expected parent_id folder1, got %v", got.ParentID)
	}
}

func TestGetFolderByID_Exists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	got, err := store.GetFolderByID(ctx, "folder1")
	if err != nil {
		t.Fatalf("get folder failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected folder, got nil")
	}
	if got.ID != "folder1" {
		t.Errorf("expected ID folder1, got %s", got.ID)
	}
}

func TestGetFolderByID_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	got, err := store.GetFolderByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get folder failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestUpdateFolder_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	folder.Name = "Important Documents"
	folder.Color = sql.NullString{String: "#00FF00", Valid: true}
	folder.IsStarred = true
	folder.UpdatedAt = time.Now().Truncate(time.Microsecond)

	if err := store.UpdateFolder(ctx, folder); err != nil {
		t.Fatalf("update folder failed: %v", err)
	}

	got, _ := store.GetFolderByID(ctx, "folder1")
	if got.Name != "Important Documents" {
		t.Errorf("expected name Important Documents, got %s", got.Name)
	}
	if got.Color.String != "#00FF00" {
		t.Errorf("expected color #00FF00, got %s", got.Color.String)
	}
	if !got.IsStarred {
		t.Error("expected is_starred true")
	}
}

func TestDeleteFolder_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	if err := store.DeleteFolder(ctx, "folder1"); err != nil {
		t.Fatalf("delete folder failed: %v", err)
	}

	got, _ := store.GetFolderByID(ctx, "folder1")
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

// ============================================================
// Folder Listing Tests
// ============================================================

func TestListFoldersByUser_RootLevel(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder1 := newTestFolder("folder1", "user1", "AAA")
	folder2 := newTestFolder("folder2", "user1", "BBB")
	store.CreateFolder(ctx, folder1)
	store.CreateFolder(ctx, folder2)

	folders, err := store.ListFoldersByUser(ctx, "user1", "")
	if err != nil {
		t.Fatalf("list folders failed: %v", err)
	}
	if len(folders) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(folders))
	}

	// Should be sorted by name
	if folders[0].Name != "AAA" {
		t.Errorf("expected first folder AAA, got %s", folders[0].Name)
	}
}

func TestListFoldersByUser_InParent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	parent := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, parent)

	child := newTestFolder("folder2", "user1", "Work")
	child.ParentID = sql.NullString{String: "folder1", Valid: true}
	store.CreateFolder(ctx, child)

	rootFolders, _ := store.ListFoldersByUser(ctx, "user1", "")
	if len(rootFolders) != 1 {
		t.Errorf("expected 1 root folder, got %d", len(rootFolders))
	}

	childFolders, _ := store.ListFoldersByUser(ctx, "user1", "folder1")
	if len(childFolders) != 1 {
		t.Errorf("expected 1 child folder, got %d", len(childFolders))
	}
	if childFolders[0].Name != "Work" {
		t.Errorf("expected Work, got %s", childFolders[0].Name)
	}
}

func TestListFoldersByUser_ExcludesTrashed(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder1 := newTestFolder("folder1", "user1", "Normal")
	folder2 := newTestFolder("folder2", "user1", "Trashed")
	folder2.TrashedAt = sql.NullTime{Time: time.Now(), Valid: true}

	store.CreateFolder(ctx, folder1)
	store.CreateFolder(ctx, folder2)

	folders, _ := store.ListFoldersByUser(ctx, "user1", "")
	if len(folders) != 1 {
		t.Errorf("expected 1 folder (excluding trashed), got %d", len(folders))
	}
	if folders[0].Name != "Normal" {
		t.Errorf("expected Normal, got %s", folders[0].Name)
	}
}

func TestListAllFoldersByUser(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create nested structure
	folder1 := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder1)

	folder2 := newTestFolder("folder2", "user1", "Work")
	folder2.ParentID = sql.NullString{String: "folder1", Valid: true}
	store.CreateFolder(ctx, folder2)

	folder3 := newTestFolder("folder3", "user1", "Projects")
	folder3.ParentID = sql.NullString{String: "folder2", Valid: true}
	store.CreateFolder(ctx, folder3)

	// ListAllFoldersByUser should return all folders regardless of nesting
	folders, err := store.ListAllFoldersByUser(ctx, "user1")
	if err != nil {
		t.Fatalf("list all folders failed: %v", err)
	}
	if len(folders) != 3 {
		t.Errorf("expected 3 folders, got %d", len(folders))
	}
}

func TestListStarredFolders(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder1 := newTestFolder("folder1", "user1", "Normal")
	folder2 := newTestFolder("folder2", "user1", "Starred")
	folder2.IsStarred = true

	store.CreateFolder(ctx, folder1)
	store.CreateFolder(ctx, folder2)

	folders, err := store.ListStarredFolders(ctx, "user1")
	if err != nil {
		t.Fatalf("list starred folders failed: %v", err)
	}
	if len(folders) != 1 {
		t.Errorf("expected 1 starred folder, got %d", len(folders))
	}
	if folders[0].Name != "Starred" {
		t.Errorf("expected Starred, got %s", folders[0].Name)
	}
}

func TestListTrashedFolders(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder1 := newTestFolder("folder1", "user1", "Normal")
	folder2 := newTestFolder("folder2", "user1", "Trashed")
	folder2.TrashedAt = sql.NullTime{Time: time.Now(), Valid: true}

	store.CreateFolder(ctx, folder1)
	store.CreateFolder(ctx, folder2)

	folders, err := store.ListTrashedFolders(ctx, "user1")
	if err != nil {
		t.Fatalf("list trashed folders failed: %v", err)
	}
	if len(folders) != 1 {
		t.Errorf("expected 1 trashed folder, got %d", len(folders))
	}
	if folders[0].Name != "Trashed" {
		t.Errorf("expected Trashed, got %s", folders[0].Name)
	}
}

func TestSearchFolders_CaseInsensitive(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "My Documents")
	store.CreateFolder(ctx, folder)

	// Test case insensitive search
	folders, _ := store.SearchFolders(ctx, "user1", "DOCUMENTS")
	if len(folders) != 1 {
		t.Errorf("expected case-insensitive match, got %d", len(folders))
	}

	folders, _ = store.SearchFolders(ctx, "user1", "documents")
	if len(folders) != 1 {
		t.Errorf("expected case-insensitive match, got %d", len(folders))
	}

	folders, _ = store.SearchFolders(ctx, "user1", "my")
	if len(folders) != 1 {
		t.Errorf("expected partial match, got %d", len(folders))
	}
}

// ============================================================
// Folder Hierarchy Tests
// ============================================================

func TestGetFolderPath_SingleLevel(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	path, err := store.GetFolderPath(ctx, "folder1")
	if err != nil {
		t.Fatalf("get folder path failed: %v", err)
	}
	if len(path) != 1 {
		t.Errorf("expected path length 1, got %d", len(path))
	}
	if path[0].Name != "Documents" {
		t.Errorf("expected Documents, got %s", path[0].Name)
	}
}

func TestGetFolderPath_DeepNesting(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create deep hierarchy: Documents > Work > Projects > 2024
	folders := []struct {
		id     string
		name   string
		parent string
	}{
		{"f1", "Documents", ""},
		{"f2", "Work", "f1"},
		{"f3", "Projects", "f2"},
		{"f4", "2024", "f3"},
	}

	for _, f := range folders {
		folder := newTestFolder(f.id, "user1", f.name)
		if f.parent != "" {
			folder.ParentID = sql.NullString{String: f.parent, Valid: true}
		}
		store.CreateFolder(ctx, folder)
	}

	path, err := store.GetFolderPath(ctx, "f4")
	if err != nil {
		t.Fatalf("get folder path failed: %v", err)
	}
	if len(path) != 4 {
		t.Errorf("expected path length 4, got %d", len(path))
	}

	expectedPath := []string{"Documents", "Work", "Projects", "2024"}
	for i, name := range expectedPath {
		if path[i].Name != name {
			t.Errorf("expected path[%d]=%s, got %s", i, name, path[i].Name)
		}
	}
}

func TestListChildFolderIDs_NoChildren(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	ids, err := store.ListChildFolderIDs(ctx, "folder1")
	if err != nil {
		t.Fatalf("list child folder IDs failed: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 children, got %d", len(ids))
	}
}

func TestListChildFolderIDs_OneLevel(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	parent := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, parent)

	child1 := newTestFolder("folder2", "user1", "Work")
	child1.ParentID = sql.NullString{String: "folder1", Valid: true}
	child2 := newTestFolder("folder3", "user1", "Personal")
	child2.ParentID = sql.NullString{String: "folder1", Valid: true}

	store.CreateFolder(ctx, child1)
	store.CreateFolder(ctx, child2)

	ids, err := store.ListChildFolderIDs(ctx, "folder1")
	if err != nil {
		t.Fatalf("list child folder IDs failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 children, got %d", len(ids))
	}
}

func TestListChildFolderIDs_Recursive(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create: Root > Level1 > Level2 > Level3
	folders := []struct {
		id     string
		name   string
		parent string
	}{
		{"root", "Root", ""},
		{"level1a", "Level1A", "root"},
		{"level1b", "Level1B", "root"},
		{"level2a", "Level2A", "level1a"},
		{"level2b", "Level2B", "level1a"},
		{"level3a", "Level3A", "level2a"},
	}

	for _, f := range folders {
		folder := newTestFolder(f.id, "user1", f.name)
		if f.parent != "" {
			folder.ParentID = sql.NullString{String: f.parent, Valid: true}
		}
		store.CreateFolder(ctx, folder)
	}

	ids, err := store.ListChildFolderIDs(ctx, "root")
	if err != nil {
		t.Fatalf("list child folder IDs failed: %v", err)
	}
	// Should include: level1a, level1b, level2a, level2b, level3a
	if len(ids) != 5 {
		t.Errorf("expected 5 descendants, got %d", len(ids))
	}
}

// ============================================================
// Trash/Star Operations Tests
// ============================================================

func TestTrashFolder(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	if err := store.TrashFolder(ctx, "folder1"); err != nil {
		t.Fatalf("trash folder failed: %v", err)
	}

	got, _ := store.GetFolderByID(ctx, "folder1")
	if !got.TrashedAt.Valid {
		t.Error("expected trashed_at to be set")
	}
}

func TestRestoreFolder(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	folder.TrashedAt = sql.NullTime{Time: time.Now(), Valid: true}
	store.CreateFolder(ctx, folder)

	if err := store.RestoreFolder(ctx, "folder1"); err != nil {
		t.Fatalf("restore folder failed: %v", err)
	}

	got, _ := store.GetFolderByID(ctx, "folder1")
	if got.TrashedAt.Valid {
		t.Error("expected trashed_at to be cleared")
	}
}

func TestStarFolder(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	if err := store.StarFolder(ctx, "folder1"); err != nil {
		t.Fatalf("star folder failed: %v", err)
	}

	got, _ := store.GetFolderByID(ctx, "folder1")
	if !got.IsStarred {
		t.Error("expected is_starred true")
	}
}

func TestUnstarFolder(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	folder.IsStarred = true
	store.CreateFolder(ctx, folder)

	if err := store.UnstarFolder(ctx, "folder1"); err != nil {
		t.Fatalf("unstar folder failed: %v", err)
	}

	got, _ := store.GetFolderByID(ctx, "folder1")
	if got.IsStarred {
		t.Error("expected is_starred false")
	}
}

// ============================================================
// Business Use Cases
// ============================================================

func TestFolderHierarchy_TreeNavigation(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create a folder tree
	// Root
	// ├── Documents
	// │   ├── Work
	// │   └── Personal
	// └── Photos
	//     └── 2024

	folders := []struct {
		id     string
		name   string
		parent string
	}{
		{"documents", "Documents", ""},
		{"photos", "Photos", ""},
		{"work", "Work", "documents"},
		{"personal", "Personal", "documents"},
		{"photos2024", "2024", "photos"},
	}

	for _, f := range folders {
		folder := newTestFolder(f.id, "user1", f.name)
		if f.parent != "" {
			folder.ParentID = sql.NullString{String: f.parent, Valid: true}
		}
		store.CreateFolder(ctx, folder)
	}

	// Navigate: list root folders
	root, _ := store.ListFoldersByUser(ctx, "user1", "")
	if len(root) != 2 {
		t.Errorf("expected 2 root folders, got %d", len(root))
	}

	// Navigate: list Documents children
	docsChildren, _ := store.ListFoldersByUser(ctx, "user1", "documents")
	if len(docsChildren) != 2 {
		t.Errorf("expected 2 Documents children, got %d", len(docsChildren))
	}

	// Navigate: list Photos children
	photosChildren, _ := store.ListFoldersByUser(ctx, "user1", "photos")
	if len(photosChildren) != 1 {
		t.Errorf("expected 1 Photos child, got %d", len(photosChildren))
	}
}

func TestFolderWithContents(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	folder := newTestFolder("folder1", "user1", "Documents")
	store.CreateFolder(ctx, folder)

	// Add subfolders
	subfolder := newTestFolder("folder2", "user1", "Work")
	subfolder.ParentID = sql.NullString{String: "folder1", Valid: true}
	store.CreateFolder(ctx, subfolder)

	// Add files
	file := newTestFile("file1", "user1", "readme.txt")
	file.ParentID = sql.NullString{String: "folder1", Valid: true}
	store.CreateFile(ctx, file)

	// Verify folder contents
	subfolders, _ := store.ListFoldersByUser(ctx, "user1", "folder1")
	if len(subfolders) != 1 {
		t.Errorf("expected 1 subfolder, got %d", len(subfolders))
	}

	files, _ := store.ListFilesByUser(ctx, "user1", "folder1")
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
}

func TestFolderMove_ChangeParent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create two root folders
	folder1 := newTestFolder("folder1", "user1", "Documents")
	folder2 := newTestFolder("folder2", "user1", "Archive")
	store.CreateFolder(ctx, folder1)
	store.CreateFolder(ctx, folder2)

	// Create a child folder
	child := newTestFolder("child", "user1", "OldProject")
	child.ParentID = sql.NullString{String: "folder1", Valid: true}
	store.CreateFolder(ctx, child)

	// Verify initial location
	docs, _ := store.ListFoldersByUser(ctx, "user1", "folder1")
	if len(docs) != 1 {
		t.Error("child should be in Documents initially")
	}

	// Move child to Archive
	child.ParentID = sql.NullString{String: "folder2", Valid: true}
	store.UpdateFolder(ctx, child)

	// Verify new location
	docs, _ = store.ListFoldersByUser(ctx, "user1", "folder1")
	if len(docs) != 0 {
		t.Error("Documents should be empty after move")
	}

	archive, _ := store.ListFoldersByUser(ctx, "user1", "folder2")
	if len(archive) != 1 {
		t.Error("child should be in Archive after move")
	}
}

func TestFolderBreadcrumb_PathConstruction(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create nested folders
	folders := []struct {
		id     string
		name   string
		parent string
	}{
		{"home", "Home", ""},
		{"docs", "Documents", "home"},
		{"work", "Work", "docs"},
		{"project", "Project Alpha", "work"},
	}

	for _, f := range folders {
		folder := newTestFolder(f.id, "user1", f.name)
		if f.parent != "" {
			folder.ParentID = sql.NullString{String: f.parent, Valid: true}
		}
		store.CreateFolder(ctx, folder)
	}

	// Get path for deepest folder
	path, _ := store.GetFolderPath(ctx, "project")

	// Should be able to construct breadcrumb: Home > Documents > Work > Project Alpha
	if len(path) != 4 {
		t.Fatalf("expected 4 breadcrumb items, got %d", len(path))
	}

	breadcrumb := ""
	for i, folder := range path {
		if i > 0 {
			breadcrumb += " > "
		}
		breadcrumb += folder.Name
	}

	expected := "Home > Documents > Work > Project Alpha"
	if breadcrumb != expected {
		t.Errorf("expected %q, got %q", expected, breadcrumb)
	}
}
