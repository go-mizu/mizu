package duckdb

import (
	"context"
	"testing"
	"time"
)

func TestUploadsStore_Create(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		upload := &Upload{
			Filename:         "test-image.jpg",
			OriginalFilename: "My Test Image.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 500, // 500KB
		}

		err := store.Uploads.Create(ctx, upload)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if upload.ID == "" {
			t.Error("Expected ID to be set")
		}
		if upload.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if upload.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}
	})

	t.Run("WithDimensions", func(t *testing.T) {
		width, height := 1920, 1080
		upload := &Upload{
			Filename:         "photo-dimensions.jpg",
			OriginalFilename: "Photo.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 1000,
			Width:            &width,
			Height:           &height,
		}

		err := store.Uploads.Create(ctx, upload)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Verify by fetching
		fetched, _ := store.Uploads.GetByID(ctx, upload.ID)
		if fetched.Width == nil || *fetched.Width != 1920 {
			t.Errorf("Expected Width=1920, got %v", fetched.Width)
		}
		if fetched.Height == nil || *fetched.Height != 1080 {
			t.Errorf("Expected Height=1080, got %v", fetched.Height)
		}
	})

	t.Run("WithFocalPoint", func(t *testing.T) {
		focalX, focalY := 0.3, 0.7
		upload := &Upload{
			Filename:         "photo-focal.jpg",
			OriginalFilename: "Photo.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 800,
			FocalX:           &focalX,
			FocalY:           &focalY,
		}

		err := store.Uploads.Create(ctx, upload)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		fetched, _ := store.Uploads.GetByID(ctx, upload.ID)
		if fetched.FocalX == nil || *fetched.FocalX != 0.3 {
			t.Errorf("Expected FocalX=0.3, got %v", fetched.FocalX)
		}
		if fetched.FocalY == nil || *fetched.FocalY != 0.7 {
			t.Errorf("Expected FocalY=0.7, got %v", fetched.FocalY)
		}
	})

	t.Run("WithSizes", func(t *testing.T) {
		upload := &Upload{
			Filename:         "photo-sizes.jpg",
			OriginalFilename: "Photo.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 1500,
			Sizes: map[string]ImageSizeInfo{
				"thumbnail": {
					Width:    150,
					Height:   150,
					Filename: "photo-sizes-thumb.jpg",
					MimeType: "image/jpeg",
					Filesize: 1024 * 10,
				},
				"medium": {
					Width:    800,
					Height:   600,
					Filename: "photo-sizes-medium.jpg",
					MimeType: "image/jpeg",
					Filesize: 1024 * 100,
				},
			},
		}

		err := store.Uploads.Create(ctx, upload)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		fetched, _ := store.Uploads.GetByID(ctx, upload.ID)
		if len(fetched.Sizes) != 2 {
			t.Errorf("Expected 2 sizes, got %d", len(fetched.Sizes))
		}
		thumb, ok := fetched.Sizes["thumbnail"]
		if !ok {
			t.Error("Expected thumbnail size")
		} else if thumb.Width != 150 {
			t.Errorf("Expected thumbnail width=150, got %d", thumb.Width)
		}
	})

	t.Run("WithMetadata", func(t *testing.T) {
		upload := &Upload{
			Filename:         "photo-meta.jpg",
			OriginalFilename: "Photo.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 600,
			Alt:              "A beautiful landscape",
			Caption:          "Sunset over the mountains",
		}

		err := store.Uploads.Create(ctx, upload)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		fetched, _ := store.Uploads.GetByID(ctx, upload.ID)
		if fetched.Alt != "A beautiful landscape" {
			t.Errorf("Expected Alt='A beautiful landscape', got %s", fetched.Alt)
		}
		if fetched.Caption != "Sunset over the mountains" {
			t.Errorf("Expected Caption='Sunset over the mountains', got %s", fetched.Caption)
		}
	})

	t.Run("MinimalUpload", func(t *testing.T) {
		upload := &Upload{
			Filename:         "document.pdf",
			OriginalFilename: "Document.pdf",
			MimeType:         "application/pdf",
			Filesize:         1024 * 200,
		}

		err := store.Uploads.Create(ctx, upload)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if upload.ID == "" {
			t.Error("Expected ID to be set")
		}
	})
}

func TestUploadsStore_GetByID(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Exists", func(t *testing.T) {
		upload := &Upload{
			Filename:         "find-me.jpg",
			OriginalFilename: "Find Me.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 300,
		}
		store.Uploads.Create(ctx, upload)

		fetched, err := store.Uploads.GetByID(ctx, upload.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		if fetched == nil {
			t.Fatal("Expected upload to be found")
		}
		if fetched.Filename != "find-me.jpg" {
			t.Errorf("Expected Filename='find-me.jpg', got %s", fetched.Filename)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		fetched, err := store.Uploads.GetByID(ctx, "nonexistent12345678901234")
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		if fetched != nil {
			t.Error("Expected nil for non-existent upload")
		}
	})

	t.Run("NullableFields", func(t *testing.T) {
		upload := &Upload{
			Filename:         "nullable-test.pdf",
			OriginalFilename: "Nullable.pdf",
			MimeType:         "application/pdf",
			Filesize:         1024 * 100,
		}
		store.Uploads.Create(ctx, upload)

		fetched, err := store.Uploads.GetByID(ctx, upload.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		if fetched.Width != nil {
			t.Errorf("Expected Width=nil, got %v", fetched.Width)
		}
		if fetched.Height != nil {
			t.Errorf("Expected Height=nil, got %v", fetched.Height)
		}
		if fetched.FocalX != nil {
			t.Errorf("Expected FocalX=nil, got %v", fetched.FocalX)
		}
	})

	t.Run("SizesDeserialization", func(t *testing.T) {
		upload := &Upload{
			Filename:         "sizes-deser.jpg",
			OriginalFilename: "Sizes.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 400,
			Sizes: map[string]ImageSizeInfo{
				"small": {Width: 300, Height: 200, Filename: "small.jpg", MimeType: "image/jpeg", Filesize: 5000},
			},
		}
		store.Uploads.Create(ctx, upload)

		fetched, err := store.Uploads.GetByID(ctx, upload.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		if fetched.Sizes == nil {
			t.Fatal("Expected Sizes to be populated")
		}
		small, ok := fetched.Sizes["small"]
		if !ok {
			t.Error("Expected 'small' size")
		} else {
			if small.Width != 300 {
				t.Errorf("Expected small.Width=300, got %d", small.Width)
			}
		}
	})
}

func TestUploadsStore_Update(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("AllFields", func(t *testing.T) {
		upload := &Upload{
			Filename:         "update-all.jpg",
			OriginalFilename: "Update.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 500,
		}
		store.Uploads.Create(ctx, upload)

		// Update
		width, height := 800, 600
		focalX, focalY := 0.5, 0.5
		upload.Filename = "updated-filename.jpg"
		upload.Width = &width
		upload.Height = &height
		upload.FocalX = &focalX
		upload.FocalY = &focalY
		upload.Alt = "Updated alt"
		upload.Caption = "Updated caption"

		err := store.Uploads.Update(ctx, upload)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		fetched, _ := store.Uploads.GetByID(ctx, upload.ID)
		if fetched.Filename != "updated-filename.jpg" {
			t.Errorf("Expected updated filename, got %s", fetched.Filename)
		}
		if fetched.Alt != "Updated alt" {
			t.Errorf("Expected updated alt, got %s", fetched.Alt)
		}
	})

	t.Run("UpdatesTimestamp", func(t *testing.T) {
		upload := &Upload{
			Filename:         "timestamp-update.jpg",
			OriginalFilename: "Timestamp.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 300,
		}
		store.Uploads.Create(ctx, upload)
		originalUpdated := upload.UpdatedAt

		time.Sleep(10 * time.Millisecond)

		upload.Alt = "Updated"
		err := store.Uploads.Update(ctx, upload)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if !upload.UpdatedAt.After(originalUpdated) {
			t.Error("Expected UpdatedAt to be updated")
		}
	})

	t.Run("AddSizes", func(t *testing.T) {
		upload := &Upload{
			Filename:         "add-sizes.jpg",
			OriginalFilename: "AddSizes.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 800,
		}
		store.Uploads.Create(ctx, upload)

		upload.Sizes = map[string]ImageSizeInfo{
			"large": {Width: 1200, Height: 800, Filename: "large.jpg", MimeType: "image/jpeg", Filesize: 100000},
		}

		err := store.Uploads.Update(ctx, upload)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		fetched, _ := store.Uploads.GetByID(ctx, upload.ID)
		if len(fetched.Sizes) != 1 {
			t.Errorf("Expected 1 size, got %d", len(fetched.Sizes))
		}
	})

	t.Run("UpdateAlt", func(t *testing.T) {
		upload := &Upload{
			Filename:         "update-alt.jpg",
			OriginalFilename: "UpdateAlt.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 400,
			Alt:              "Original alt text",
		}
		store.Uploads.Create(ctx, upload)

		upload.Alt = "New alt text"
		err := store.Uploads.Update(ctx, upload)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		fetched, _ := store.Uploads.GetByID(ctx, upload.ID)
		if fetched.Alt != "New alt text" {
			t.Errorf("Expected Alt='New alt text', got %s", fetched.Alt)
		}
	})
}

func TestUploadsStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		upload := &Upload{
			Filename:         "delete-me.jpg",
			OriginalFilename: "Delete.jpg",
			MimeType:         "image/jpeg",
			Filesize:         1024 * 200,
		}
		store.Uploads.Create(ctx, upload)

		err := store.Uploads.Delete(ctx, upload.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		fetched, _ := store.Uploads.GetByID(ctx, upload.ID)
		if fetched != nil {
			t.Error("Expected upload to be deleted")
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		err := store.Uploads.Delete(ctx, "nonexistent12345678901234")
		if err != nil {
			t.Fatalf("Delete failed for non-existent: %v", err)
		}
	})
}

func TestUploadsStore_Find(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create test data
	uploads := []struct {
		filename string
		mimeType string
		filesize int64
	}{
		{"image1.jpg", "image/jpeg", 1024 * 100},
		{"image2.png", "image/png", 1024 * 200},
		{"image3.jpg", "image/jpeg", 1024 * 300},
		{"document.pdf", "application/pdf", 1024 * 400},
		{"video.mp4", "video/mp4", 1024 * 5000},
	}

	for _, u := range uploads {
		upload := &Upload{
			Filename:         u.filename,
			OriginalFilename: u.filename,
			MimeType:         u.mimeType,
			Filesize:         u.filesize,
		}
		store.Uploads.Create(ctx, upload)
	}

	t.Run("All", func(t *testing.T) {
		result, err := store.Uploads.Find(ctx, nil)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 5 {
			t.Errorf("Expected TotalDocs=5, got %d", result.TotalDocs)
		}
	})

	t.Run("ByMimeType", func(t *testing.T) {
		result, err := store.Uploads.Find(ctx, &FindOptions{
			Where: map[string]any{"mimeType": "image/jpeg"},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 2 {
			t.Errorf("Expected 2 jpeg images, got %d", result.TotalDocs)
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		result, err := store.Uploads.Find(ctx, &FindOptions{
			Limit: 2,
			Page:  2,
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.Limit != 2 {
			t.Errorf("Expected Limit=2, got %d", result.Limit)
		}
		if result.Page != 2 {
			t.Errorf("Expected Page=2, got %d", result.Page)
		}
		if len(result.Docs) != 2 {
			t.Errorf("Expected 2 docs, got %d", len(result.Docs))
		}
	})

	t.Run("Sorting", func(t *testing.T) {
		result, err := store.Uploads.Find(ctx, &FindOptions{
			Sort: []SortField{{Field: "filesize", Desc: true}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if len(result.Docs) > 0 {
			firstFilesize, ok := result.Docs[0].Data["filesize"].(int64)
			if ok && firstFilesize != 1024*5000 {
				t.Errorf("Expected largest file first, got filesize=%d", firstFilesize)
			}
		}
	})
}
