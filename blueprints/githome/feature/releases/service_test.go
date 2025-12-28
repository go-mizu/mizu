//go:build ignore
// +build ignore

// This test file is excluded from build until releases store is implemented in duckdb store.

package releases_test

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/releases"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*releases.Service, *duckdb.Store, string, func()) {
	t.Helper()
	t.Skip("releases store not yet implemented in duckdb store")

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}

	store, err := duckdb.New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		store.Close()
		t.Fatalf("failed to ensure schema: %v", err)
	}

	// Create temp storage directory
	storagePath, err := os.MkdirTemp("", "releases-test-*")
	if err != nil {
		store.Close()
		t.Fatalf("failed to create temp dir: %v", err)
	}

	service := releases.NewService(store.Releases(), store.Repos(), store.Users(), "https://api.example.com", storagePath)

	cleanup := func() {
		os.RemoveAll(storagePath)
		store.Close()
	}

	return service, store, storagePath, cleanup
}

func createTestUser(t *testing.T, store *duckdb.Store, login, email string) *users.User {
	t.Helper()
	user := &users.User{
		Login:        login,
		Email:        email,
		Name:         "Test User",
		PasswordHash: "hash",
		Type:         "User",
	}
	if err := store.Users().Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestRepo(t *testing.T, store *duckdb.Store, owner *users.User, name string) *repos.Repository {
	t.Helper()
	repo := &repos.Repository{
		Name:          name,
		FullName:      owner.Login + "/" + name,
		OwnerID:       owner.ID,
		OwnerType:     "User",
		Visibility:    "public",
		DefaultBranch: "main",
	}
	if err := store.Repos().Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

func createTestRelease(t *testing.T, service *releases.Service, owner, repo string, authorID int64, tag string) *releases.Release {
	t.Helper()
	rel, err := service.Create(context.Background(), owner, repo, authorID, &releases.CreateIn{
		TagName: tag,
		Name:    tag,
		Body:    "Release notes for " + tag,
	})
	if err != nil {
		t.Fatalf("failed to create test release: %v", err)
	}
	return rel
}

// Release Creation Tests

func TestService_Create_Success(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	rel, err := service.Create(context.Background(), "owner", "testrepo", owner.ID, &releases.CreateIn{
		TagName: "v1.0.0",
		Name:    "Version 1.0.0",
		Body:    "Initial release",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if rel.TagName != "v1.0.0" {
		t.Errorf("got tag_name %q, want v1.0.0", rel.TagName)
	}
	if rel.Name != "Version 1.0.0" {
		t.Errorf("got name %q, want Version 1.0.0", rel.Name)
	}
	if rel.Body != "Initial release" {
		t.Errorf("got body %q, want Initial release", rel.Body)
	}
	if rel.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if rel.Draft {
		t.Error("expected draft to be false by default")
	}
	if rel.Prerelease {
		t.Error("expected prerelease to be false by default")
	}
	if rel.TargetCommitish != "main" {
		t.Errorf("expected target_commitish to be main, got %q", rel.TargetCommitish)
	}
	if rel.Author == nil {
		t.Error("expected author to be set")
	}
	if rel.PublishedAt == nil {
		t.Error("expected published_at to be set for non-draft")
	}
	if rel.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_Create_Draft(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	rel, err := service.Create(context.Background(), "owner", "testrepo", owner.ID, &releases.CreateIn{
		TagName: "v1.0.0",
		Draft:   true,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !rel.Draft {
		t.Error("expected draft to be true")
	}
	if rel.PublishedAt != nil {
		t.Error("expected published_at to be nil for draft")
	}
}

func TestService_Create_Prerelease(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	rel, err := service.Create(context.Background(), "owner", "testrepo", owner.ID, &releases.CreateIn{
		TagName:    "v1.0.0-beta",
		Prerelease: true,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !rel.Prerelease {
		t.Error("expected prerelease to be true")
	}
}

func TestService_Create_DuplicateTag(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	_, err := service.Create(context.Background(), "owner", "testrepo", owner.ID, &releases.CreateIn{
		TagName: "v1.0.0",
	})
	if err != releases.ErrReleaseExists {
		t.Errorf("expected ErrReleaseExists, got %v", err)
	}
}

func TestService_Create_RepoNotFound(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")

	_, err := service.Create(context.Background(), "owner", "unknown", owner.ID, &releases.CreateIn{
		TagName: "v1.0.0",
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// Release Retrieval Tests

func TestService_Get_Success(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	rel, err := service.Get(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if rel.ID != created.ID {
		t.Errorf("got ID %d, want %d", rel.ID, created.ID)
	}
	if rel.TagName != "v1.0.0" {
		t.Errorf("got tag_name %q, want v1.0.0", rel.TagName)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_, err := service.Get(context.Background(), "owner", "testrepo", 99999)
	if err != releases.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetByTag_Success(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	rel, err := service.GetByTag(context.Background(), "owner", "testrepo", "v1.0.0")
	if err != nil {
		t.Fatalf("GetByTag failed: %v", err)
	}

	if rel.TagName != "v1.0.0" {
		t.Errorf("got tag_name %q, want v1.0.0", rel.TagName)
	}
}

func TestService_GetByTag_NotFound(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_, err := service.GetByTag(context.Background(), "owner", "testrepo", "nonexistent")
	if err != releases.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetLatest(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")
	createTestRelease(t, service, "owner", "testrepo", owner.ID, "v2.0.0")

	rel, err := service.GetLatest(context.Background(), "owner", "testrepo")
	if err != nil {
		t.Fatalf("GetLatest failed: %v", err)
	}

	// Should return v2.0.0 (latest)
	if rel.TagName != "v2.0.0" {
		t.Errorf("expected latest release v2.0.0, got %q", rel.TagName)
	}
}

func TestService_GetLatest_NoReleases(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_, err := service.GetLatest(context.Background(), "owner", "testrepo")
	if err != releases.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_List(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")
	createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.1.0")
	createTestRelease(t, service, "owner", "testrepo", owner.ID, "v2.0.0")

	list, err := service.List(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 releases, got %d", len(list))
	}
}

func TestService_List_Pagination(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	for i := 0; i < 5; i++ {
		createTestRelease(t, service, "owner", "testrepo", owner.ID, "v"+string(rune('a'+i)))
	}

	list, err := service.List(context.Background(), "owner", "testrepo", &releases.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 releases, got %d", len(list))
	}
}

// Release Update Tests

func TestService_Update_Name(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	newName := "Updated Release Name"
	updated, err := service.Update(context.Background(), "owner", "testrepo", created.ID, &releases.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Name != "Updated Release Name" {
		t.Errorf("got name %q, want Updated Release Name", updated.Name)
	}
}

func TestService_Update_Body(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	newBody := "Updated release body"
	updated, err := service.Update(context.Background(), "owner", "testrepo", created.ID, &releases.UpdateIn{
		Body: &newBody,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Body != "Updated release body" {
		t.Errorf("got body %q, want Updated release body", updated.Body)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	newName := "newname"
	_, err := service.Update(context.Background(), "owner", "testrepo", 99999, &releases.UpdateIn{
		Name: &newName,
	})
	if err != releases.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Release Delete Tests

func TestService_Delete_Success(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	err := service.Delete(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = service.Get(context.Background(), "owner", "testrepo", created.ID)
	if err != releases.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	err := service.Delete(context.Background(), "owner", "testrepo", 99999)
	if err != releases.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Asset Tests

func TestService_UploadAsset_Success(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	rel := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	content := []byte("binary content")
	reader := bytes.NewReader(content)

	asset, err := service.UploadAsset(context.Background(), "owner", "testrepo", rel.ID, owner.ID, "app.zip", "application/zip", reader)
	if err != nil {
		t.Fatalf("UploadAsset failed: %v", err)
	}

	if asset.Name != "app.zip" {
		t.Errorf("got name %q, want app.zip", asset.Name)
	}
	if asset.ContentType != "application/zip" {
		t.Errorf("got content_type %q, want application/zip", asset.ContentType)
	}
	if asset.Size != len(content) {
		t.Errorf("got size %d, want %d", asset.Size, len(content))
	}
	if asset.State != "uploaded" {
		t.Errorf("got state %q, want uploaded", asset.State)
	}
	if asset.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if asset.Uploader == nil {
		t.Error("expected uploader to be set")
	}
	if asset.URL == "" {
		t.Error("expected URL to be populated")
	}
	if asset.BrowserDownloadURL == "" {
		t.Error("expected BrowserDownloadURL to be populated")
	}
}

func TestService_UploadAsset_ReleaseNotFound(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	reader := bytes.NewReader([]byte("content"))

	_, err := service.UploadAsset(context.Background(), "owner", "testrepo", 99999, owner.ID, "app.zip", "application/zip", reader)
	if err != releases.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetAsset_Success(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	rel := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	reader := bytes.NewReader([]byte("content"))
	created, _ := service.UploadAsset(context.Background(), "owner", "testrepo", rel.ID, owner.ID, "app.zip", "application/zip", reader)

	asset, err := service.GetAsset(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}

	if asset.ID != created.ID {
		t.Errorf("got ID %d, want %d", asset.ID, created.ID)
	}
}

func TestService_GetAsset_NotFound(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_, err := service.GetAsset(context.Background(), "owner", "testrepo", 99999)
	if err != releases.ErrAssetNotFound {
		t.Errorf("expected ErrAssetNotFound, got %v", err)
	}
}

func TestService_ListAssets(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	rel := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	_, _ = service.UploadAsset(context.Background(), "owner", "testrepo", rel.ID, owner.ID, "app1.zip", "application/zip", bytes.NewReader([]byte("1")))
	_, _ = service.UploadAsset(context.Background(), "owner", "testrepo", rel.ID, owner.ID, "app2.zip", "application/zip", bytes.NewReader([]byte("2")))

	assets, err := service.ListAssets(context.Background(), "owner", "testrepo", rel.ID, nil)
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}

	if len(assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(assets))
	}
}

func TestService_UpdateAsset(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	rel := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	reader := bytes.NewReader([]byte("content"))
	created, _ := service.UploadAsset(context.Background(), "owner", "testrepo", rel.ID, owner.ID, "app.zip", "application/zip", reader)

	newName := "renamed.zip"
	newLabel := "Main application"
	updated, err := service.UpdateAsset(context.Background(), "owner", "testrepo", created.ID, &releases.UpdateAssetIn{
		Name:  &newName,
		Label: &newLabel,
	})
	if err != nil {
		t.Fatalf("UpdateAsset failed: %v", err)
	}

	if updated.Name != "renamed.zip" {
		t.Errorf("got name %q, want renamed.zip", updated.Name)
	}
	if updated.Label != "Main application" {
		t.Errorf("got label %q, want Main application", updated.Label)
	}
}

func TestService_DeleteAsset_Success(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	rel := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	reader := bytes.NewReader([]byte("content"))
	created, _ := service.UploadAsset(context.Background(), "owner", "testrepo", rel.ID, owner.ID, "app.zip", "application/zip", reader)

	err := service.DeleteAsset(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("DeleteAsset failed: %v", err)
	}

	// Verify deleted
	_, err = service.GetAsset(context.Background(), "owner", "testrepo", created.ID)
	if err != releases.ErrAssetNotFound {
		t.Errorf("expected ErrAssetNotFound after delete, got %v", err)
	}
}

func TestService_DownloadAsset(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	rel := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	content := []byte("binary content here")
	reader := bytes.NewReader(content)
	created, _ := service.UploadAsset(context.Background(), "owner", "testrepo", rel.ID, owner.ID, "app.zip", "application/zip", reader)

	rc, contentType, err := service.DownloadAsset(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("DownloadAsset failed: %v", err)
	}
	defer rc.Close()

	if contentType != "application/zip" {
		t.Errorf("got content_type %q, want application/zip", contentType)
	}

	// Read content
	buf := new(bytes.Buffer)
	buf.ReadFrom(rc)
	if buf.String() != string(content) {
		t.Errorf("downloaded content doesn't match")
	}
}

// Generate Notes Tests

func TestService_GenerateNotes(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	notes, err := service.GenerateNotes(context.Background(), "owner", "testrepo", &releases.GenerateNotesIn{
		TagName:         "v2.0.0",
		PreviousTagName: "v1.0.0",
	})
	if err != nil {
		t.Fatalf("GenerateNotes failed: %v", err)
	}

	if notes.Name != "v2.0.0" {
		t.Errorf("got name %q, want v2.0.0", notes.Name)
	}
	if notes.Body == "" {
		t.Error("expected body to be populated")
	}
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	rel := createTestRelease(t, service, "owner", "testrepo", owner.ID, "v1.0.0")

	if rel.URL == "" {
		t.Error("expected URL to be set")
	}
	if rel.HTMLURL == "" {
		t.Error("expected HTMLURL to be set")
	}
	if rel.AssetsURL == "" {
		t.Error("expected AssetsURL to be set")
	}
	if rel.UploadURL == "" {
		t.Error("expected UploadURL to be set")
	}
	if rel.TarballURL == "" {
		t.Error("expected TarballURL to be set")
	}
	if rel.ZipballURL == "" {
		t.Error("expected ZipballURL to be set")
	}
	if rel.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

// Integration Test - Releases Across Repos

func TestService_ReleasesAcrossRepos(t *testing.T) {
	service, store, _, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "repo1")
	createTestRepo(t, store, owner, "repo2")

	// Same tag in different repos should work
	rel1 := createTestRelease(t, service, "owner", "repo1", owner.ID, "v1.0.0")
	rel2 := createTestRelease(t, service, "owner", "repo2", owner.ID, "v1.0.0")

	if rel1.ID == rel2.ID {
		t.Error("releases in different repos should have different IDs")
	}

	// Each repo should have its own releases
	list1, _ := service.List(context.Background(), "owner", "repo1", nil)
	list2, _ := service.List(context.Background(), "owner", "repo2", nil)

	if len(list1) != 1 {
		t.Errorf("repo1 should have 1 release, got %d", len(list1))
	}
	if len(list2) != 1 {
		t.Errorf("repo2 should have 1 release, got %d", len(list2))
	}
}
