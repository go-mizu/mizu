// File: lib/storage/multipart_test.go
package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/go-mizu/blueprints/drive/lib/storage"
)

// MultipartSuite runs the full multipart test suite against any storage implementation
// that supports multipart uploads.
func MultipartSuite(t *testing.T, factory StorageFactory) {
	t.Helper()
	t.Run("Multipart", func(t *testing.T) {
		multipartTests(t, factory)
	})
}

// multipartTests tests HasMultipart operations.
func multipartTests(t *testing.T, factory StorageFactory) {
	t.Run("InitMultipart", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, err := st.CreateBucket(ctx, "multipart", nil)
		if err != nil {
			t.Fatalf("CreateBucket: %v", err)
		}

		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "test.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}

		if mu.Bucket != "multipart" {
			t.Errorf("expected bucket 'multipart', got %q", mu.Bucket)
		}
		if mu.Key != "test.bin" {
			t.Errorf("expected key 'test.bin', got %q", mu.Key)
		}
		if mu.UploadID == "" {
			t.Error("expected non-empty UploadID")
		}

		// Clean up
		_ = mp.AbortMultipart(ctx, mu, nil)
	})

	t.Run("InitMultipart_EmptyKey", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		_, err := mp.InitMultipart(ctx, "", "application/octet-stream", nil)
		if err == nil {
			t.Error("expected error for empty key")
		}
	})

	t.Run("InitMultipart_WithMetadata", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		meta := map[string]string{"custom": "value", "another": "meta"}
		mu, err := mp.InitMultipart(ctx, "test.bin", "application/json", storage.Options{"metadata": meta})
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		// Verify metadata is stored
		if mu.Metadata == nil {
			t.Error("expected metadata to be stored")
		} else {
			if mu.Metadata["custom"] != "value" {
				t.Errorf("expected custom=value, got %q", mu.Metadata["custom"])
			}
		}
	})

	t.Run("UploadPart", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "test.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		data := []byte("hello part 1")
		part, err := mp.UploadPart(ctx, mu, 1, bytes.NewReader(data), int64(len(data)), nil)
		if err != nil {
			t.Fatalf("UploadPart: %v", err)
		}

		if part.Number != 1 {
			t.Errorf("expected part number 1, got %d", part.Number)
		}
		if part.Size != int64(len(data)) {
			t.Errorf("expected size %d, got %d", len(data), part.Size)
		}
		if part.ETag == "" {
			t.Error("expected non-empty ETag")
		}
	})

	t.Run("UploadPart_InvalidPartNumber", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "test.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		// Part number 0 is invalid
		_, err = mp.UploadPart(ctx, mu, 0, strings.NewReader("data"), 4, nil)
		if err == nil {
			t.Error("expected error for part number 0")
		}

		// Part number > 10000 is invalid
		_, err = mp.UploadPart(ctx, mu, 10001, strings.NewReader("data"), 4, nil)
		if err == nil {
			t.Error("expected error for part number > 10000")
		}

		// Negative part number
		_, err = mp.UploadPart(ctx, mu, -1, strings.NewReader("data"), 4, nil)
		if err == nil {
			t.Error("expected error for negative part number")
		}
	})

	t.Run("UploadPart_InvalidUploadID", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		invalidMU := &storage.MultipartUpload{
			Bucket:   "multipart",
			Key:      "test.bin",
			UploadID: "nonexistent-upload-id",
		}

		_, err := mp.UploadPart(ctx, invalidMU, 1, strings.NewReader("data"), 4, nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist for invalid upload, got %v", err)
		}
	})

	t.Run("UploadPart_MultipleParts", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "test.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		parts := []struct {
			number int
			data   string
		}{
			{1, "part one data"},
			{2, "part two data here"},
			{3, "part three"},
		}

		for _, p := range parts {
			_, err := mp.UploadPart(ctx, mu, p.number, strings.NewReader(p.data), int64(len(p.data)), nil)
			if err != nil {
				t.Errorf("UploadPart %d: %v", p.number, err)
			}
		}
	})

	t.Run("UploadPart_OverwritePart", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "test.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		// Upload part 1
		_, err = mp.UploadPart(ctx, mu, 1, strings.NewReader("first"), 5, nil)
		if err != nil {
			t.Fatalf("UploadPart first: %v", err)
		}

		// Overwrite part 1
		part, err := mp.UploadPart(ctx, mu, 1, strings.NewReader("second data"), 11, nil)
		if err != nil {
			t.Fatalf("UploadPart overwrite: %v", err)
		}

		if part.Size != 11 {
			t.Errorf("expected overwritten part size 11, got %d", part.Size)
		}
	})

	t.Run("ListParts", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "test.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		// Upload a few parts
		for i := 1; i <= 3; i++ {
			data := strings.Repeat("x", i*100)
			_, err := mp.UploadPart(ctx, mu, i, strings.NewReader(data), int64(len(data)), nil)
			if err != nil {
				t.Fatalf("UploadPart %d: %v", i, err)
			}
		}

		parts, err := mp.ListParts(ctx, mu, 0, 0, nil)
		if err != nil {
			t.Fatalf("ListParts: %v", err)
		}

		if len(parts) != 3 {
			t.Errorf("expected 3 parts, got %d", len(parts))
		}

		// Verify parts are sorted by number
		for i, p := range parts {
			if p.Number != i+1 {
				t.Errorf("expected part %d at index %d, got part %d", i+1, i, p.Number)
			}
		}
	})

	t.Run("ListParts_Empty", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "test.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		parts, err := mp.ListParts(ctx, mu, 0, 0, nil)
		if err != nil {
			t.Fatalf("ListParts: %v", err)
		}

		if len(parts) != 0 {
			t.Errorf("expected 0 parts, got %d", len(parts))
		}
	})

	t.Run("ListParts_Pagination", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "test.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		// Upload 5 parts
		for i := 1; i <= 5; i++ {
			_, _ = mp.UploadPart(ctx, mu, i, strings.NewReader("data"), 4, nil)
		}

		// Get first 2
		parts, err := mp.ListParts(ctx, mu, 2, 0, nil)
		if err != nil {
			t.Fatalf("ListParts: %v", err)
		}
		if len(parts) != 2 {
			t.Errorf("expected 2 parts with limit 2, got %d", len(parts))
		}

		// Get with offset
		parts, err = mp.ListParts(ctx, mu, 10, 3, nil)
		if err != nil {
			t.Fatalf("ListParts with offset: %v", err)
		}
		if len(parts) != 2 {
			t.Errorf("expected 2 parts with offset 3, got %d", len(parts))
		}
	})

	t.Run("ListParts_InvalidUploadID", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		invalidMU := &storage.MultipartUpload{
			Bucket:   "multipart",
			Key:      "test.bin",
			UploadID: "nonexistent",
		}

		_, err := mp.ListParts(ctx, invalidMU, 0, 0, nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("CompleteMultipart", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "complete.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}

		// Upload parts
		part1Data := "AAAAAAAAAA" // 10 bytes
		part2Data := "BBBBBBBBBB" // 10 bytes
		part3Data := "CCCCCCCCCC" // 10 bytes

		p1, _ := mp.UploadPart(ctx, mu, 1, strings.NewReader(part1Data), int64(len(part1Data)), nil)
		p2, _ := mp.UploadPart(ctx, mu, 2, strings.NewReader(part2Data), int64(len(part2Data)), nil)
		p3, _ := mp.UploadPart(ctx, mu, 3, strings.NewReader(part3Data), int64(len(part3Data)), nil)

		// Complete
		obj, err := mp.CompleteMultipart(ctx, mu, []*storage.PartInfo{p1, p2, p3}, nil)
		if err != nil {
			t.Fatalf("CompleteMultipart: %v", err)
		}

		if obj.Key != "complete.bin" {
			t.Errorf("expected key 'complete.bin', got %q", obj.Key)
		}
		if obj.Size != 30 {
			t.Errorf("expected size 30, got %d", obj.Size)
		}

		// Verify content by reading back
		rc, _, err := b.Open(ctx, "complete.bin", 0, 0, nil)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		expected := part1Data + part2Data + part3Data
		if string(data) != expected {
			t.Errorf("expected %q, got %q", expected, string(data))
		}
	})

	t.Run("CompleteMultipart_OutOfOrder", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "outoforder.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}

		// Upload parts out of order
		p3, _ := mp.UploadPart(ctx, mu, 3, strings.NewReader("CCC"), 3, nil)
		p1, _ := mp.UploadPart(ctx, mu, 1, strings.NewReader("AAA"), 3, nil)
		p2, _ := mp.UploadPart(ctx, mu, 2, strings.NewReader("BBB"), 3, nil)

		// Complete with parts in wrong order (implementation should sort)
		obj, err := mp.CompleteMultipart(ctx, mu, []*storage.PartInfo{p3, p1, p2}, nil)
		if err != nil {
			t.Fatalf("CompleteMultipart: %v", err)
		}

		// Verify content is in correct order
		rc, _, _ := b.Open(ctx, "outoforder.bin", 0, 0, nil)
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != "AAABBBCCC" {
			t.Errorf("expected 'AAABBBCCC', got %q", string(data))
		}

		if obj.Size != 9 {
			t.Errorf("expected size 9, got %d", obj.Size)
		}
	})

	t.Run("CompleteMultipart_EmptyParts", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "empty.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		// Try to complete with no parts
		_, err = mp.CompleteMultipart(ctx, mu, []*storage.PartInfo{}, nil)
		if err == nil {
			t.Error("expected error for empty parts list")
		}
	})

	t.Run("CompleteMultipart_MissingPart", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "missing.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		// Upload only part 1
		p1, _ := mp.UploadPart(ctx, mu, 1, strings.NewReader("AAA"), 3, nil)

		// Try to complete with a part that wasn't uploaded
		fakePart := &storage.PartInfo{Number: 99, Size: 10, ETag: "fake"}
		_, err = mp.CompleteMultipart(ctx, mu, []*storage.PartInfo{p1, fakePart}, nil)
		if err == nil {
			t.Error("expected error for missing part")
		}
	})

	t.Run("CompleteMultipart_InvalidUploadID", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		invalidMU := &storage.MultipartUpload{
			Bucket:   "multipart",
			Key:      "test.bin",
			UploadID: "nonexistent",
		}

		parts := []*storage.PartInfo{{Number: 1, Size: 10, ETag: "fake"}}
		_, err := mp.CompleteMultipart(ctx, invalidMU, parts, nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("AbortMultipart", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "abort.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}

		// Upload some parts
		_, _ = mp.UploadPart(ctx, mu, 1, strings.NewReader("data"), 4, nil)
		_, _ = mp.UploadPart(ctx, mu, 2, strings.NewReader("more"), 4, nil)

		// Abort
		err = mp.AbortMultipart(ctx, mu, nil)
		if err != nil {
			t.Fatalf("AbortMultipart: %v", err)
		}

		// Verify upload is gone
		_, err = mp.ListParts(ctx, mu, 0, 0, nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist after abort, got %v", err)
		}
	})

	t.Run("AbortMultipart_AlreadyAborted", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "double-abort.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}

		// Abort first time
		err = mp.AbortMultipart(ctx, mu, nil)
		if err != nil {
			t.Fatalf("AbortMultipart first: %v", err)
		}

		// Abort second time should not error (idempotent)
		// Note: Some implementations may return ErrNotExist, which is also acceptable
		_ = mp.AbortMultipart(ctx, mu, nil)
	})

	t.Run("CopyPart_Unsupported", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "copypart.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		// CopyPart is typically unsupported for local/memory drivers
		_, err = mp.CopyPart(ctx, mu, 1, nil)
		if !errors.Is(err, storage.ErrUnsupported) {
			t.Logf("CopyPart returned: %v (may be supported or unsupported)", err)
		}
	})

	t.Run("LargeMultipartUpload", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "large.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}

		// Upload multiple 1MB parts
		partSize := 1024 * 1024 // 1MB
		numParts := 3
		var parts []*storage.PartInfo

		for i := 1; i <= numParts; i++ {
			data := bytes.Repeat([]byte{byte(i)}, partSize)
			part, err := mp.UploadPart(ctx, mu, i, bytes.NewReader(data), int64(len(data)), nil)
			if err != nil {
				t.Fatalf("UploadPart %d: %v", i, err)
			}
			parts = append(parts, part)
		}

		obj, err := mp.CompleteMultipart(ctx, mu, parts, nil)
		if err != nil {
			t.Fatalf("CompleteMultipart: %v", err)
		}

		expectedSize := int64(partSize * numParts)
		if obj.Size != expectedSize {
			t.Errorf("expected size %d, got %d", expectedSize, obj.Size)
		}

		// Verify stat
		statObj, err := b.Stat(ctx, "large.bin", nil)
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if statObj.Size != expectedSize {
			t.Errorf("stat size mismatch: expected %d, got %d", expectedSize, statObj.Size)
		}
	})

	t.Run("ConcurrentPartUploads", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "concurrent.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}
		defer func() {
			_ = mp.AbortMultipart(ctx, mu, nil)
		}()

		numParts := 10
		var wg sync.WaitGroup
		errs := make(chan error, numParts)
		parts := make([]*storage.PartInfo, numParts)
		var partsMu sync.Mutex

		for i := 1; i <= numParts; i++ {
			wg.Add(1)
			go func(partNum int) {
				defer wg.Done()
				data := strings.Repeat(string(rune('A'+partNum-1)), 100)
				part, err := mp.UploadPart(ctx, mu, partNum, strings.NewReader(data), int64(len(data)), nil)
				if err != nil {
					errs <- err
					return
				}
				partsMu.Lock()
				parts[partNum-1] = part
				partsMu.Unlock()
			}(i)
		}

		wg.Wait()
		close(errs)

		for err := range errs {
			t.Errorf("concurrent upload error: %v", err)
		}

		// Verify all parts were uploaded
		listedParts, err := mp.ListParts(ctx, mu, 0, 0, nil)
		if err != nil {
			t.Fatalf("ListParts: %v", err)
		}
		if len(listedParts) != numParts {
			t.Errorf("expected %d parts, got %d", numParts, len(listedParts))
		}
	})

	t.Run("MultipleMultipartUploads", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		// Start multiple uploads for different keys
		mu1, _ := mp.InitMultipart(ctx, "file1.bin", "application/octet-stream", nil)
		mu2, _ := mp.InitMultipart(ctx, "file2.bin", "application/octet-stream", nil)
		defer func() {
			_ = mp.AbortMultipart(ctx, mu1, nil)
			_ = mp.AbortMultipart(ctx, mu2, nil)
		}()

		// Upload parts to both
		_, err := mp.UploadPart(ctx, mu1, 1, strings.NewReader("file1"), 5, nil)
		if err != nil {
			t.Errorf("UploadPart to mu1: %v", err)
		}

		_, err = mp.UploadPart(ctx, mu2, 1, strings.NewReader("file2"), 5, nil)
		if err != nil {
			t.Errorf("UploadPart to mu2: %v", err)
		}

		// Verify parts are separate
		parts1, _ := mp.ListParts(ctx, mu1, 0, 0, nil)
		parts2, _ := mp.ListParts(ctx, mu2, 0, 0, nil)

		if len(parts1) != 1 || len(parts2) != 1 {
			t.Errorf("expected 1 part each, got %d and %d", len(parts1), len(parts2))
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		_, _ = st.CreateBucket(context.Background(), "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		cancel() // Cancel immediately

		_, err := mp.InitMultipart(ctx, "cancelled.bin", "application/octet-stream", nil)
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})

	t.Run("NestedKey", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "multipart", nil)
		b := st.Bucket("multipart")
		mp, ok := b.(storage.HasMultipart)
		if !ok {
			t.Skip("bucket does not support multipart uploads")
		}

		mu, err := mp.InitMultipart(ctx, "path/to/nested/file.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}

		part, _ := mp.UploadPart(ctx, mu, 1, strings.NewReader("nested"), 6, nil)
		obj, err := mp.CompleteMultipart(ctx, mu, []*storage.PartInfo{part}, nil)
		if err != nil {
			t.Fatalf("CompleteMultipart: %v", err)
		}

		if obj.Key != "path/to/nested/file.bin" {
			t.Errorf("expected key 'path/to/nested/file.bin', got %q", obj.Key)
		}

		// Verify accessible
		_, err = b.Stat(ctx, "path/to/nested/file.bin", nil)
		if err != nil {
			t.Errorf("Stat nested: %v", err)
		}
	})
}

// MultipartTestHelper provides helper functions for multipart testing.
type MultipartTestHelper struct {
	t       *testing.T
	storage storage.Storage
	bucket  storage.Bucket
	mp      storage.HasMultipart
}

// NewMultipartTestHelper creates a new helper for multipart testing.
func NewMultipartTestHelper(t *testing.T, st storage.Storage, bucketName string) *MultipartTestHelper {
	t.Helper()
	ctx := context.Background()

	_, _ = st.CreateBucket(ctx, bucketName, nil)
	b := st.Bucket(bucketName)
	mp, ok := b.(storage.HasMultipart)
	if !ok {
		return nil
	}

	return &MultipartTestHelper{
		t:       t,
		storage: st,
		bucket:  b,
		mp:      mp,
	}
}

// CreateAndComplete creates a multipart upload with the given parts and completes it.
func (h *MultipartTestHelper) CreateAndComplete(ctx context.Context, key string, partData []string) (*storage.Object, error) {
	mu, err := h.mp.InitMultipart(ctx, key, "application/octet-stream", nil)
	if err != nil {
		return nil, err
	}

	var parts []*storage.PartInfo
	for i, data := range partData {
		part, err := h.mp.UploadPart(ctx, mu, i+1, strings.NewReader(data), int64(len(data)), nil)
		if err != nil {
			_ = h.mp.AbortMultipart(ctx, mu, nil)
			return nil, err
		}
		parts = append(parts, part)
	}

	// Sort parts by number
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Number < parts[j].Number
	})

	return h.mp.CompleteMultipart(ctx, mu, parts, nil)
}
