package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/cms/feature/media"
)

func TestMediaStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	m := &media.Media{
		ID:               "media-001",
		UploaderID:       "user-001",
		Filename:         "image-abc123.jpg",
		OriginalFilename: "my-photo.jpg",
		MimeType:         "image/jpeg",
		FileSize:         1024000,
		StoragePath:      "/uploads/2024/01/image-abc123.jpg",
		StorageProvider:  "local",
		URL:              "/uploads/2024/01/image-abc123.jpg",
		AltText:          "A beautiful photo",
		Caption:          "Photo caption",
		Title:            "My Photo",
		Description:      "A detailed description",
		Width:            1920,
		Height:           1080,
		Meta:             `{"exif":"data"}`,
		CreatedAt:        testTime,
		UpdatedAt:        testTime,
	}

	err := store.Create(ctx, m)
	assertNoError(t, err)

	got, err := store.GetByID(ctx, m.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, m.ID)
	assertEqual(t, "Filename", got.Filename, m.Filename)
	assertEqual(t, "OriginalFilename", got.OriginalFilename, m.OriginalFilename)
	assertEqual(t, "MimeType", got.MimeType, m.MimeType)
	assertEqual(t, "FileSize", got.FileSize, m.FileSize)
	assertEqual(t, "Width", got.Width, m.Width)
	assertEqual(t, "Height", got.Height, m.Height)
}

func TestMediaStore_Create_WithDimensions(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	m := &media.Media{
		ID:               "media-dim",
		UploaderID:       "user-001",
		Filename:         "image-dim.jpg",
		OriginalFilename: "dim.jpg",
		MimeType:         "image/jpeg",
		FileSize:         500000,
		StoragePath:      "/uploads/dim.jpg",
		StorageProvider:  "local",
		URL:              "/uploads/dim.jpg",
		Width:            800,
		Height:           600,
		CreatedAt:        testTime,
		UpdatedAt:        testTime,
	}

	err := store.Create(ctx, m)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, m.ID)
	assertEqual(t, "Width", got.Width, 800)
	assertEqual(t, "Height", got.Height, 600)
}

func TestMediaStore_Create_WithDuration(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	m := &media.Media{
		ID:               "media-video",
		UploaderID:       "user-001",
		Filename:         "video.mp4",
		OriginalFilename: "my-video.mp4",
		MimeType:         "video/mp4",
		FileSize:         10000000,
		StoragePath:      "/uploads/video.mp4",
		StorageProvider:  "local",
		URL:              "/uploads/video.mp4",
		Width:            1920,
		Height:           1080,
		Duration:         300, // 5 minutes in seconds
		CreatedAt:        testTime,
		UpdatedAt:        testTime,
	}

	err := store.Create(ctx, m)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, m.ID)
	assertEqual(t, "Duration", got.Duration, 300)
}

func TestMediaStore_Create_MinimalFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	m := &media.Media{
		ID:               "media-min",
		UploaderID:       "user-001",
		Filename:         "file.txt",
		OriginalFilename: "document.txt",
		MimeType:         "text/plain",
		FileSize:         100,
		StoragePath:      "/uploads/file.txt",
		StorageProvider:  "local",
		URL:              "/uploads/file.txt",
		CreatedAt:        testTime,
		UpdatedAt:        testTime,
	}

	err := store.Create(ctx, m)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, m.ID)
	assertEqual(t, "AltText", got.AltText, "")
	assertEqual(t, "Caption", got.Caption, "")
	assertEqual(t, "Width", got.Width, 0)
	assertEqual(t, "Height", got.Height, 0)
}

func TestMediaStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	m := &media.Media{
		ID:               "media-get",
		UploaderID:       "user-001",
		Filename:         "get.jpg",
		OriginalFilename: "get.jpg",
		MimeType:         "image/jpeg",
		FileSize:         1000,
		StoragePath:      "/uploads/get.jpg",
		StorageProvider:  "local",
		URL:              "/uploads/get.jpg",
		CreatedAt:        testTime,
		UpdatedAt:        testTime,
	}
	assertNoError(t, store.Create(ctx, m))

	got, err := store.GetByID(ctx, m.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, m.ID)
}

func TestMediaStore_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	got, err := store.GetByID(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent media")
	}
}

func TestMediaStore_GetByFilename(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	m := &media.Media{
		ID:               "media-filename",
		UploaderID:       "user-001",
		Filename:         "unique-filename-abc123.jpg",
		OriginalFilename: "photo.jpg",
		MimeType:         "image/jpeg",
		FileSize:         1000,
		StoragePath:      "/uploads/unique-filename-abc123.jpg",
		StorageProvider:  "local",
		URL:              "/uploads/unique-filename-abc123.jpg",
		CreatedAt:        testTime,
		UpdatedAt:        testTime,
	}
	assertNoError(t, store.Create(ctx, m))

	got, err := store.GetByFilename(ctx, "unique-filename-abc123.jpg")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, m.ID)
}

func TestMediaStore_List(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		m := &media.Media{
			ID:               "media-list-" + string(rune('a'+i)),
			UploaderID:       "user-001",
			Filename:         "list-" + string(rune('a'+i)) + ".jpg",
			OriginalFilename: "file.jpg",
			MimeType:         "image/jpeg",
			FileSize:         1000,
			StoragePath:      "/uploads/list-" + string(rune('a'+i)) + ".jpg",
			StorageProvider:  "local",
			URL:              "/uploads/list-" + string(rune('a'+i)) + ".jpg",
			CreatedAt:        testTime,
			UpdatedAt:        testTime,
		}
		assertNoError(t, store.Create(ctx, m))
	}

	list, total, err := store.List(ctx, &media.ListIn{Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 5)
	assertLen(t, list, 5)
}

func TestMediaStore_List_FilterByUploader(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	uploaders := []string{"user-a", "user-a", "user-b"}
	for i, uploader := range uploaders {
		m := &media.Media{
			ID:               "media-uploader-" + string(rune('a'+i)),
			UploaderID:       uploader,
			Filename:         "uploader-" + string(rune('a'+i)) + ".jpg",
			OriginalFilename: "file.jpg",
			MimeType:         "image/jpeg",
			FileSize:         1000,
			StoragePath:      "/uploads/uploader.jpg",
			StorageProvider:  "local",
			URL:              "/uploads/uploader.jpg",
			CreatedAt:        testTime,
			UpdatedAt:        testTime,
		}
		assertNoError(t, store.Create(ctx, m))
	}

	list, total, err := store.List(ctx, &media.ListIn{UploaderID: "user-a", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestMediaStore_List_FilterByMimeType(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	mimeTypes := []string{"image/jpeg", "image/png", "video/mp4", "image/gif"}
	for i, mt := range mimeTypes {
		m := &media.Media{
			ID:               "media-mime-" + string(rune('a'+i)),
			UploaderID:       "user-001",
			Filename:         "mime-" + string(rune('a'+i)),
			OriginalFilename: "file",
			MimeType:         mt,
			FileSize:         1000,
			StoragePath:      "/uploads/mime",
			StorageProvider:  "local",
			URL:              "/uploads/mime",
			CreatedAt:        testTime,
			UpdatedAt:        testTime,
		}
		assertNoError(t, store.Create(ctx, m))
	}

	// Filter by image/* (prefix match)
	list, total, err := store.List(ctx, &media.ListIn{MimeType: "image", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 3) // jpeg, png, gif
	assertLen(t, list, 3)
}

func TestMediaStore_List_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	mediaData := []struct {
		filename string
		title    string
	}{
		{"photo-abc.jpg", "Vacation Photo"},
		{"document.pdf", "Report"},
		{"photo-xyz.jpg", "Another Photo"},
	}
	for i, md := range mediaData {
		m := &media.Media{
			ID:               "media-search-" + string(rune('a'+i)),
			UploaderID:       "user-001",
			Filename:         md.filename,
			OriginalFilename: md.filename,
			MimeType:         "application/octet-stream",
			FileSize:         1000,
			StoragePath:      "/uploads/" + md.filename,
			StorageProvider:  "local",
			URL:              "/uploads/" + md.filename,
			Title:            md.title,
			CreatedAt:        testTime,
			UpdatedAt:        testTime,
		}
		assertNoError(t, store.Create(ctx, m))
	}

	// Search by filename or title containing "photo"
	list, total, err := store.List(ctx, &media.ListIn{Search: "photo", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2) // 2 files with "photo" in filename or title
	assertLen(t, list, 2)
}

func TestMediaStore_List_OrderBy(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		m := &media.Media{
			ID:               "media-order-" + string(rune('a'+i)),
			UploaderID:       "user-001",
			Filename:         "order-" + string(rune('c'-i)) + ".jpg", // c, b, a
			OriginalFilename: "file.jpg",
			MimeType:         "image/jpeg",
			FileSize:         int64((i + 1) * 1000), // 1000, 2000, 3000
			StoragePath:      "/uploads/order.jpg",
			StorageProvider:  "local",
			URL:              "/uploads/order.jpg",
			CreatedAt:        testTime,
			UpdatedAt:        testTime,
		}
		assertNoError(t, store.Create(ctx, m))
	}

	// Order by file_size DESC
	list, _, err := store.List(ctx, &media.ListIn{OrderBy: "file_size", Order: "DESC", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "FileSize[0]", list[0].FileSize, int64(3000))
	assertEqual(t, "FileSize[2]", list[2].FileSize, int64(1000))
}

func TestMediaStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	m := &media.Media{
		ID:               "media-update",
		UploaderID:       "user-001",
		Filename:         "update.jpg",
		OriginalFilename: "update.jpg",
		MimeType:         "image/jpeg",
		FileSize:         1000,
		StoragePath:      "/uploads/update.jpg",
		StorageProvider:  "local",
		URL:              "/uploads/update.jpg",
		CreatedAt:        testTime,
		UpdatedAt:        testTime,
	}
	assertNoError(t, store.Create(ctx, m))

	err := store.Update(ctx, m.ID, &media.UpdateIn{
		AltText:     ptr("New alt text"),
		Caption:     ptr("New caption"),
		Title:       ptr("New title"),
		Description: ptr("New description"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, m.ID)
	assertEqual(t, "AltText", got.AltText, "New alt text")
	assertEqual(t, "Caption", got.Caption, "New caption")
	assertEqual(t, "Title", got.Title, "New title")
	assertEqual(t, "Description", got.Description, "New description")
}

func TestMediaStore_Update_PartialFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	m := &media.Media{
		ID:               "media-partial",
		UploaderID:       "user-001",
		Filename:         "partial.jpg",
		OriginalFilename: "partial.jpg",
		MimeType:         "image/jpeg",
		FileSize:         1000,
		StoragePath:      "/uploads/partial.jpg",
		StorageProvider:  "local",
		URL:              "/uploads/partial.jpg",
		AltText:          "Original alt",
		Caption:          "Original caption",
		CreatedAt:        testTime,
		UpdatedAt:        testTime,
	}
	assertNoError(t, store.Create(ctx, m))

	// Only update alt text
	err := store.Update(ctx, m.ID, &media.UpdateIn{
		AltText: ptr("New alt"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, m.ID)
	assertEqual(t, "AltText", got.AltText, "New alt")
	assertEqual(t, "Caption", got.Caption, "Original caption") // Unchanged
}

func TestMediaStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewMediaStore(db)
	ctx := context.Background()

	m := &media.Media{
		ID:               "media-delete",
		UploaderID:       "user-001",
		Filename:         "delete.jpg",
		OriginalFilename: "delete.jpg",
		MimeType:         "image/jpeg",
		FileSize:         1000,
		StoragePath:      "/uploads/delete.jpg",
		StorageProvider:  "local",
		URL:              "/uploads/delete.jpg",
		CreatedAt:        testTime,
		UpdatedAt:        testTime,
	}
	assertNoError(t, store.Create(ctx, m))

	err := store.Delete(ctx, m.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, m.ID)
	if got != nil {
		t.Error("expected media to be deleted")
	}
}
