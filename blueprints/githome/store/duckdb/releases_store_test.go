package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/releases"
	"github.com/oklog/ulid/v2"
)

func createTestRelease(t *testing.T, store *ReleasesStore, repoID, authorID, tagName string) *releases.Release {
	t.Helper()
	id := ulid.Make().String()
	r := &releases.Release{
		ID:              id,
		RepoID:          repoID,
		TagName:         tagName,
		TargetCommitish: "main",
		Name:            "Release " + tagName,
		Body:            "Release notes for " + tagName,
		IsDraft:         false,
		IsPrerelease:    false,
		AuthorID:        authorID,
		CreatedAt:       time.Now(),
	}
	if err := store.Create(context.Background(), r); err != nil {
		t.Fatalf("failed to create test release: %v", err)
	}
	return r
}

// =============================================================================
// Release CRUD Tests
// =============================================================================

func TestReleasesStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	publishedAt := time.Now()
	r := &releases.Release{
		ID:              ulid.Make().String(),
		RepoID:          repoID,
		TagName:         "v1.0.0",
		TargetCommitish: "main",
		Name:            "Version 1.0.0",
		Body:            "Initial release",
		IsDraft:         false,
		IsPrerelease:    false,
		AuthorID:        userID,
		CreatedAt:       time.Now(),
		PublishedAt:     &publishedAt,
	}

	err := releasesStore.Create(context.Background(), r)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := releasesStore.GetByID(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected release to be created")
	}
	if got.TagName != "v1.0.0" {
		t.Errorf("got tag_name %q, want %q", got.TagName, "v1.0.0")
	}
	if got.Name != "Version 1.0.0" {
		t.Errorf("got name %q, want %q", got.Name, "Version 1.0.0")
	}
}

func TestReleasesStore_Create_Draft(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := &releases.Release{
		ID:              ulid.Make().String(),
		RepoID:          repoID,
		TagName:         "v2.0.0",
		TargetCommitish: "develop",
		Name:            "Version 2.0.0",
		Body:            "Draft release",
		IsDraft:         true,
		IsPrerelease:    false,
		AuthorID:        userID,
		CreatedAt:       time.Now(),
	}

	releasesStore.Create(context.Background(), r)

	got, _ := releasesStore.GetByID(context.Background(), r.ID)
	if !got.IsDraft {
		t.Error("expected release to be draft")
	}
	if got.PublishedAt != nil {
		t.Error("expected published_at to be nil for draft")
	}
}

func TestReleasesStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := createTestRelease(t, releasesStore, repoID, userID, "v1.0.0")

	got, err := releasesStore.GetByID(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected release")
	}
	if got.ID != r.ID {
		t.Errorf("got ID %q, want %q", got.ID, r.ID)
	}
}

func TestReleasesStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	releasesStore := NewReleasesStore(store.DB())

	got, err := releasesStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent release")
	}
}

func TestReleasesStore_GetByTag(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := createTestRelease(t, releasesStore, repoID, userID, "v1.2.3")

	got, err := releasesStore.GetByTag(context.Background(), repoID, "v1.2.3")
	if err != nil {
		t.Fatalf("GetByTag failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected release")
	}
	if got.ID != r.ID {
		t.Errorf("got ID %q, want %q", got.ID, r.ID)
	}
}

func TestReleasesStore_GetByTag_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	got, err := releasesStore.GetByTag(context.Background(), repoID, "v999.999.999")
	if err != nil {
		t.Fatalf("GetByTag failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent tag")
	}
}

func TestReleasesStore_GetLatest(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	// Create releases with published_at
	for i := 1; i <= 3; i++ {
		publishedAt := time.Now().Add(time.Duration(i) * time.Hour)
		r := &releases.Release{
			ID:              ulid.Make().String(),
			RepoID:          repoID,
			TagName:         "v1." + string(rune('0'+i)) + ".0",
			TargetCommitish: "main",
			Name:            "Release",
			IsDraft:         false,
			IsPrerelease:    false,
			AuthorID:        userID,
			CreatedAt:       time.Now(),
			PublishedAt:     &publishedAt,
		}
		releasesStore.Create(context.Background(), r)
	}

	got, err := releasesStore.GetLatest(context.Background(), repoID)
	if err != nil {
		t.Fatalf("GetLatest failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected release")
	}
	// Should be the latest published release
	if got.TagName != "v1.3.0" {
		t.Errorf("got tag_name %q, want %q", got.TagName, "v1.3.0")
	}
}

func TestReleasesStore_GetLatest_SkipsDrafts(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	// Create a published release
	publishedAt := time.Now()
	r1 := &releases.Release{
		ID:              ulid.Make().String(),
		RepoID:          repoID,
		TagName:         "v1.0.0",
		TargetCommitish: "main",
		Name:            "Published Release",
		IsDraft:         false,
		IsPrerelease:    false,
		AuthorID:        userID,
		CreatedAt:       time.Now(),
		PublishedAt:     &publishedAt,
	}
	releasesStore.Create(context.Background(), r1)

	// Create a draft release (should be skipped)
	r2 := &releases.Release{
		ID:              ulid.Make().String(),
		RepoID:          repoID,
		TagName:         "v2.0.0",
		TargetCommitish: "main",
		Name:            "Draft Release",
		IsDraft:         true,
		IsPrerelease:    false,
		AuthorID:        userID,
		CreatedAt:       time.Now(),
	}
	releasesStore.Create(context.Background(), r2)

	got, _ := releasesStore.GetLatest(context.Background(), repoID)
	if got.TagName != "v1.0.0" {
		t.Errorf("expected published release, got %q", got.TagName)
	}
}

func TestReleasesStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := createTestRelease(t, releasesStore, repoID, userID, "v1.0.0")

	r.Name = "Updated Name"
	r.Body = "Updated body"
	publishedAt := time.Now()
	r.PublishedAt = &publishedAt

	err := releasesStore.Update(context.Background(), r)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := releasesStore.GetByID(context.Background(), r.ID)
	if got.Name != "Updated Name" {
		t.Errorf("got name %q, want %q", got.Name, "Updated Name")
	}
	if got.PublishedAt == nil {
		t.Error("expected published_at to be set")
	}
}

func TestReleasesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := createTestRelease(t, releasesStore, repoID, userID, "v1.0.0")

	err := releasesStore.Delete(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := releasesStore.GetByID(context.Background(), r.ID)
	if got != nil {
		t.Error("expected release to be deleted")
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestReleasesStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	for i := 1; i <= 5; i++ {
		createTestRelease(t, releasesStore, repoID, userID, "v1."+string(rune('0'+i))+".0")
	}

	list, err := releasesStore.List(context.Background(), repoID, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d releases, want 5", len(list))
	}
}

func TestReleasesStore_List_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	for i := 1; i <= 10; i++ {
		createTestRelease(t, releasesStore, repoID, userID, "v"+string(rune('0'+i))+".0.0")
	}

	page1, _ := releasesStore.List(context.Background(), repoID, 3, 0)
	page2, _ := releasesStore.List(context.Background(), repoID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d releases on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d releases on page 2, want 3", len(page2))
	}
}

// =============================================================================
// Asset Tests
// =============================================================================

func TestReleasesStore_CreateAsset(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := createTestRelease(t, releasesStore, repoID, userID, "v1.0.0")

	asset := &releases.Asset{
		ID:          ulid.Make().String(),
		ReleaseID:   r.ID,
		Name:        "app-v1.0.0.zip",
		Label:       "Application Binary",
		ContentType: "application/zip",
		SizeBytes:   1024 * 1024,
		UploaderID:  userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := releasesStore.CreateAsset(context.Background(), asset)
	if err != nil {
		t.Fatalf("CreateAsset failed: %v", err)
	}

	got, _ := releasesStore.GetAsset(context.Background(), asset.ID)
	if got == nil {
		t.Fatal("expected asset")
	}
	if got.Name != "app-v1.0.0.zip" {
		t.Errorf("got name %q, want %q", got.Name, "app-v1.0.0.zip")
	}
}

func TestReleasesStore_GetAsset(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := createTestRelease(t, releasesStore, repoID, userID, "v1.0.0")

	asset := &releases.Asset{
		ID:          ulid.Make().String(),
		ReleaseID:   r.ID,
		Name:        "app.zip",
		ContentType: "application/zip",
		SizeBytes:   1024,
		UploaderID:  userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	releasesStore.CreateAsset(context.Background(), asset)

	got, err := releasesStore.GetAsset(context.Background(), asset.ID)
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected asset")
	}
}

func TestReleasesStore_UpdateAsset(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := createTestRelease(t, releasesStore, repoID, userID, "v1.0.0")

	asset := &releases.Asset{
		ID:          ulid.Make().String(),
		ReleaseID:   r.ID,
		Name:        "old-name.zip",
		ContentType: "application/zip",
		SizeBytes:   1024,
		UploaderID:  userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	releasesStore.CreateAsset(context.Background(), asset)

	asset.Name = "new-name.zip"
	asset.Label = "Updated Label"

	err := releasesStore.UpdateAsset(context.Background(), asset)
	if err != nil {
		t.Fatalf("UpdateAsset failed: %v", err)
	}

	got, _ := releasesStore.GetAsset(context.Background(), asset.ID)
	if got.Name != "new-name.zip" {
		t.Errorf("got name %q, want %q", got.Name, "new-name.zip")
	}
	if got.Label != "Updated Label" {
		t.Errorf("got label %q, want %q", got.Label, "Updated Label")
	}
}

func TestReleasesStore_DeleteAsset(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := createTestRelease(t, releasesStore, repoID, userID, "v1.0.0")

	asset := &releases.Asset{
		ID:          ulid.Make().String(),
		ReleaseID:   r.ID,
		Name:        "app.zip",
		ContentType: "application/zip",
		SizeBytes:   1024,
		UploaderID:  userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	releasesStore.CreateAsset(context.Background(), asset)

	err := releasesStore.DeleteAsset(context.Background(), asset.ID)
	if err != nil {
		t.Fatalf("DeleteAsset failed: %v", err)
	}

	got, _ := releasesStore.GetAsset(context.Background(), asset.ID)
	if got != nil {
		t.Error("expected asset to be deleted")
	}
}

func TestReleasesStore_ListAssets(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := createTestRelease(t, releasesStore, repoID, userID, "v1.0.0")

	for i := 0; i < 3; i++ {
		asset := &releases.Asset{
			ID:          ulid.Make().String(),
			ReleaseID:   r.ID,
			Name:        "asset-" + string(rune('a'+i)) + ".zip",
			ContentType: "application/zip",
			SizeBytes:   1024,
			UploaderID:  userID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		releasesStore.CreateAsset(context.Background(), asset)
	}

	list, err := releasesStore.ListAssets(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d assets, want 3", len(list))
	}
}

func TestReleasesStore_IncrementDownload(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	releasesStore := NewReleasesStore(store.DB())

	r := createTestRelease(t, releasesStore, repoID, userID, "v1.0.0")

	asset := &releases.Asset{
		ID:            ulid.Make().String(),
		ReleaseID:     r.ID,
		Name:          "app.zip",
		ContentType:   "application/zip",
		SizeBytes:     1024,
		DownloadCount: 0,
		UploaderID:    userID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	releasesStore.CreateAsset(context.Background(), asset)

	// Increment download count 5 times
	for i := 0; i < 5; i++ {
		err := releasesStore.IncrementDownload(context.Background(), asset.ID)
		if err != nil {
			t.Fatalf("IncrementDownload failed: %v", err)
		}
	}

	got, _ := releasesStore.GetAsset(context.Background(), asset.ID)
	if got.DownloadCount != 5 {
		t.Errorf("got download_count %d, want 5", got.DownloadCount)
	}
}

// Verify interface compliance
var _ releases.Store = (*ReleasesStore)(nil)
