// File: lib/storage/conformance_test.go
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
	"time"

	"github.com/go-mizu/blueprints/drive/lib/storage"
)

// StorageFactory creates a new Storage instance for testing.
// The returned cleanup function should remove any temporary resources.
type StorageFactory func(t *testing.T) (storage.Storage, func())

// ConformanceSuite runs the full conformance test suite against any storage implementation.
func ConformanceSuite(t *testing.T, factory StorageFactory) {
	t.Helper()
	t.Run("Storage", func(t *testing.T) {
		storageTests(t, factory)
	})
	t.Run("Bucket", func(t *testing.T) {
		bucketTests(t, factory)
	})
	t.Run("Iterators", func(t *testing.T) {
		iteratorTests(t, factory)
	})
	t.Run("EdgeCases", func(t *testing.T) {
		edgeCaseTests(t, factory)
	})
	t.Run("Concurrency", func(t *testing.T) {
		concurrencyTests(t, factory)
	})
}

// storageTests tests Storage-level operations.
func storageTests(t *testing.T, factory StorageFactory) {
	t.Run("CreateBucket", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		// Create bucket
		info, err := st.CreateBucket(ctx, "test-bucket", nil)
		if err != nil {
			t.Fatalf("CreateBucket: %v", err)
		}
		if info.Name != "test-bucket" {
			t.Errorf("expected name 'test-bucket', got %q", info.Name)
		}
		if info.CreatedAt.IsZero() {
			t.Error("expected non-zero CreatedAt")
		}
	})

	t.Run("CreateBucket_AlreadyExists", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, err := st.CreateBucket(ctx, "dup-bucket", nil)
		if err != nil {
			t.Fatalf("first CreateBucket: %v", err)
		}

		_, err = st.CreateBucket(ctx, "dup-bucket", nil)
		if !errors.Is(err, storage.ErrExist) {
			t.Errorf("expected ErrExist, got %v", err)
		}
	})

	t.Run("DeleteBucket_Empty", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, err := st.CreateBucket(ctx, "to-delete", nil)
		if err != nil {
			t.Fatalf("CreateBucket: %v", err)
		}

		err = st.DeleteBucket(ctx, "to-delete", nil)
		if err != nil {
			t.Fatalf("DeleteBucket: %v", err)
		}

		// Verify deleted
		_, err = st.Bucket("to-delete").Info(ctx)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist after delete, got %v", err)
		}
	})

	t.Run("DeleteBucket_NonEmpty", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, err := st.CreateBucket(ctx, "non-empty", nil)
		if err != nil {
			t.Fatalf("CreateBucket: %v", err)
		}

		// Write an object
		b := st.Bucket("non-empty")
		_, err = b.Write(ctx, "file.txt", strings.NewReader("content"), 7, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		// Delete without force should fail
		err = st.DeleteBucket(ctx, "non-empty", nil)
		if err == nil {
			t.Error("expected error when deleting non-empty bucket without force")
		}

		// Delete with force should succeed
		err = st.DeleteBucket(ctx, "non-empty", storage.Options{"force": true})
		if err != nil {
			t.Errorf("DeleteBucket with force: %v", err)
		}
	})

	t.Run("DeleteBucket_NotExist", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		err := st.DeleteBucket(ctx, "nonexistent", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("Buckets_List", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		// Create buckets
		names := []string{"alpha", "beta", "gamma"}
		for _, name := range names {
			_, err := st.CreateBucket(ctx, name, nil)
			if err != nil {
				t.Fatalf("CreateBucket %s: %v", name, err)
			}
		}

		// List all
		iter, err := st.Buckets(ctx, 0, 0, nil)
		if err != nil {
			t.Fatalf("Buckets: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		var found []string
		for {
			info, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if info == nil {
				break
			}
			found = append(found, info.Name)
		}

		sort.Strings(found)
		sort.Strings(names)

		for _, name := range names {
			if !contains(found, name) {
				t.Errorf("expected to find bucket %q", name)
			}
		}
	})

	t.Run("Buckets_Pagination", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		// Create 5 buckets
		for i := 0; i < 5; i++ {
			_, err := st.CreateBucket(ctx, string(rune('a'+i)), nil)
			if err != nil {
				t.Fatalf("CreateBucket: %v", err)
			}
		}

		// Get first 2
		iter, err := st.Buckets(ctx, 2, 0, nil)
		if err != nil {
			t.Fatalf("Buckets: %v", err)
		}

		count := 0
		for {
			info, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if info == nil {
				break
			}
			count++
		}
		_ = iter.Close()

		if count != 2 {
			t.Errorf("expected 2 buckets, got %d", count)
		}

		// Get with offset
		iter, err = st.Buckets(ctx, 10, 3, nil)
		if err != nil {
			t.Fatalf("Buckets with offset: %v", err)
		}

		count = 0
		for {
			info, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if info == nil {
				break
			}
			count++
		}
		_ = iter.Close()

		if count != 2 {
			t.Errorf("expected 2 buckets with offset 3, got %d", count)
		}
	})

	t.Run("Features", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()

		features := st.Features()
		if features == nil {
			t.Error("Features returned nil")
		}
		// Features map can vary by implementation, just verify it returns
	})

	t.Run("Close", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()

		err := st.Close()
		if err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	t.Run("BucketHandle_NoIO", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()

		// Getting a bucket handle should not error even if bucket doesn't exist
		b := st.Bucket("nonexistent")
		if b == nil {
			t.Error("Bucket returned nil")
		}
		if b.Name() != "nonexistent" {
			t.Errorf("expected name 'nonexistent', got %q", b.Name())
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := st.CreateBucket(ctx, "cancelled", nil)
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})
}

// bucketTests tests Bucket-level operations.
func bucketTests(t *testing.T, factory StorageFactory) {
	t.Run("WriteAndOpen", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, err := st.CreateBucket(ctx, "data", nil)
		if err != nil {
			t.Fatalf("CreateBucket: %v", err)
		}

		b := st.Bucket("data")
		content := "hello world"

		obj, err := b.Write(ctx, "test.txt", strings.NewReader(content), int64(len(content)), "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		if obj.Key != "test.txt" {
			t.Errorf("expected key 'test.txt', got %q", obj.Key)
		}
		if obj.Size != int64(len(content)) {
			t.Errorf("expected size %d, got %d", len(content), obj.Size)
		}
		if obj.ContentType != "text/plain" {
			t.Errorf("expected content type 'text/plain', got %q", obj.ContentType)
		}

		// Read back
		rc, readObj, err := b.Open(ctx, "test.txt", 0, 0, nil)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer func() {
			_ = rc.Close()
		}()

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}

		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}

		if readObj.Size != int64(len(content)) {
			t.Errorf("expected readObj.Size %d, got %d", len(content), readObj.Size)
		}
	})

	t.Run("WriteOverwrite", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		// Write first
		_, err := b.Write(ctx, "overwrite.txt", strings.NewReader("first"), 5, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write first: %v", err)
		}

		// Overwrite
		_, err = b.Write(ctx, "overwrite.txt", strings.NewReader("second"), 6, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write second: %v", err)
		}

		// Read back
		rc, _, err := b.Open(ctx, "overwrite.txt", 0, 0, nil)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != "second" {
			t.Errorf("expected 'second', got %q", string(data))
		}
	})

	t.Run("WriteNestedPath", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "a/b/c/nested.txt", strings.NewReader("nested"), 6, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write nested: %v", err)
		}

		rc, _, err := b.Open(ctx, "a/b/c/nested.txt", 0, 0, nil)
		if err != nil {
			t.Fatalf("Open nested: %v", err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != "nested" {
			t.Errorf("expected 'nested', got %q", string(data))
		}
	})

	t.Run("Open_RangeRead", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		content := "0123456789"
		_, err := b.Write(ctx, "range.txt", strings.NewReader(content), 10, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		// Read with offset
		rc, _, err := b.Open(ctx, "range.txt", 3, 0, nil)
		if err != nil {
			t.Fatalf("Open with offset: %v", err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != "3456789" {
			t.Errorf("expected '3456789', got %q", string(data))
		}

		// Read with offset and length
		rc, _, err = b.Open(ctx, "range.txt", 2, 4, nil)
		if err != nil {
			t.Fatalf("Open with offset and length: %v", err)
		}
		data, _ = io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != "2345" {
			t.Errorf("expected '2345', got %q", string(data))
		}
	})

	t.Run("Open_NotExist", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _, err := b.Open(ctx, "notfound.txt", 0, 0, nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("Stat", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "stat.txt", strings.NewReader("data"), 4, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		obj, err := b.Stat(ctx, "stat.txt", nil)
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}

		if obj.Key != "stat.txt" {
			t.Errorf("expected key 'stat.txt', got %q", obj.Key)
		}
		if obj.Size != 4 {
			t.Errorf("expected size 4, got %d", obj.Size)
		}
		if obj.IsDir {
			t.Error("expected IsDir=false")
		}
	})

	t.Run("Stat_Directory", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		// Write file in subdirectory
		_, err := b.Write(ctx, "dir/file.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		// Stat the directory
		obj, err := b.Stat(ctx, "dir", nil)
		if err != nil {
			t.Fatalf("Stat dir: %v", err)
		}

		if !obj.IsDir {
			t.Error("expected IsDir=true for directory")
		}
	})

	t.Run("Stat_NotExist", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Stat(ctx, "notfound.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "delete.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		err = b.Delete(ctx, "delete.txt", nil)
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err = b.Stat(ctx, "delete.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist after delete, got %v", err)
		}
	})

	t.Run("Delete_NotExist", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		err := b.Delete(ctx, "notfound.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("Delete_Recursive", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		// Create nested files
		_, _ = b.Write(ctx, "dir/a.txt", strings.NewReader("a"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/b.txt", strings.NewReader("b"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/sub/c.txt", strings.NewReader("c"), 1, "text/plain", nil)

		err := b.Delete(ctx, "dir", storage.Options{"recursive": true})
		if err != nil {
			t.Fatalf("Delete recursive: %v", err)
		}

		_, err = b.Stat(ctx, "dir", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected dir deleted, got %v", err)
		}
	})

	t.Run("Copy", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		content := "copy me"
		_, err := b.Write(ctx, "source.txt", strings.NewReader(content), int64(len(content)), "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		obj, err := b.Copy(ctx, "dest.txt", "data", "source.txt", nil)
		if err != nil {
			t.Fatalf("Copy: %v", err)
		}

		if obj.Key != "dest.txt" {
			t.Errorf("expected key 'dest.txt', got %q", obj.Key)
		}

		// Verify copy
		rc, _, err := b.Open(ctx, "dest.txt", 0, 0, nil)
		if err != nil {
			t.Fatalf("Open copy: %v", err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}

		// Source should still exist
		_, err = b.Stat(ctx, "source.txt", nil)
		if err != nil {
			t.Errorf("source should still exist: %v", err)
		}
	})

	t.Run("Copy_CrossBucket", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "src-bucket", nil)
		_, _ = st.CreateBucket(ctx, "dst-bucket", nil)

		srcB := st.Bucket("src-bucket")
		dstB := st.Bucket("dst-bucket")

		content := "cross bucket"
		_, _ = srcB.Write(ctx, "file.txt", strings.NewReader(content), int64(len(content)), "text/plain", nil)

		_, err := dstB.Copy(ctx, "copied.txt", "src-bucket", "file.txt", nil)
		if err != nil {
			t.Fatalf("Copy cross bucket: %v", err)
		}

		rc, _, err := dstB.Open(ctx, "copied.txt", 0, 0, nil)
		if err != nil {
			t.Fatalf("Open copied: %v", err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}
	})

	t.Run("Copy_NotExist", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Copy(ctx, "dest.txt", "data", "notfound.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("Move", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		content := "move me"
		_, err := b.Write(ctx, "original.txt", strings.NewReader(content), int64(len(content)), "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		obj, err := b.Move(ctx, "moved.txt", "data", "original.txt", nil)
		if err != nil {
			t.Fatalf("Move: %v", err)
		}

		if obj.Key != "moved.txt" {
			t.Errorf("expected key 'moved.txt', got %q", obj.Key)
		}

		// Verify moved
		rc, _, err := b.Open(ctx, "moved.txt", 0, 0, nil)
		if err != nil {
			t.Fatalf("Open moved: %v", err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}

		// Original should be gone
		_, err = b.Stat(ctx, "original.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected original deleted, got %v", err)
		}
	})

	t.Run("Move_CrossBucket", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "src-bucket", nil)
		_, _ = st.CreateBucket(ctx, "dst-bucket", nil)

		srcB := st.Bucket("src-bucket")
		dstB := st.Bucket("dst-bucket")

		content := "move cross"
		_, _ = srcB.Write(ctx, "file.txt", strings.NewReader(content), int64(len(content)), "text/plain", nil)

		_, err := dstB.Move(ctx, "moved.txt", "src-bucket", "file.txt", nil)
		if err != nil {
			t.Fatalf("Move cross bucket: %v", err)
		}

		// Verify in destination
		rc, _, _ := dstB.Open(ctx, "moved.txt", 0, 0, nil)
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}

		// Source should be gone
		_, err = srcB.Stat(ctx, "file.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected source deleted, got %v", err)
		}
	})

	t.Run("Move_NotExist", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Move(ctx, "dest.txt", "data", "notfound.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("List_All", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		files := []string{"a.txt", "b.txt", "c.txt", "dir/d.txt", "dir/e.txt"}
		for _, f := range files {
			_, _ = b.Write(ctx, f, strings.NewReader("x"), 1, "text/plain", nil)
		}

		iter, err := b.List(ctx, "", 0, 0, nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		var keys []string
		for {
			obj, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if obj == nil {
				break
			}
			keys = append(keys, obj.Key)
		}

		// Should have files + dir directory
		if len(keys) < len(files) {
			t.Errorf("expected at least %d objects, got %d", len(files), len(keys))
		}
	})

	t.Run("List_Prefix", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "foo/a.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "foo/b.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "bar/c.txt", strings.NewReader("x"), 1, "text/plain", nil)

		iter, err := b.List(ctx, "foo", 0, 0, nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		var keys []string
		for {
			obj, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if obj == nil {
				break
			}
			keys = append(keys, obj.Key)
		}

		for _, k := range keys {
			if !strings.HasPrefix(k, "foo") {
				t.Errorf("unexpected key %q with prefix 'foo'", k)
			}
		}
	})

	t.Run("List_NonRecursive", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "top.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/nested.txt", strings.NewReader("x"), 1, "text/plain", nil)

		iter, err := b.List(ctx, "", 0, 0, storage.Options{"recursive": false})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		var keys []string
		for {
			obj, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if obj == nil {
				break
			}
			keys = append(keys, obj.Key)
		}

		// Should only have top.txt and dir (as directory)
		for _, k := range keys {
			if strings.Contains(k, "/") && !strings.HasSuffix(k, "/") {
				// Non-recursive shouldn't show nested files directly
				parts := strings.Split(k, "/")
				if len(parts) > 2 {
					t.Errorf("non-recursive list shouldn't show deeply nested %q", k)
				}
			}
		}
	})

	t.Run("List_Pagination", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		for i := 0; i < 10; i++ {
			_, _ = b.Write(ctx, string(rune('a'+i))+".txt", strings.NewReader("x"), 1, "text/plain", nil)
		}

		// Get first 3
		iter, err := b.List(ctx, "", 3, 0, nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}

		count := 0
		for {
			obj, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if obj == nil {
				break
			}
			count++
		}
		_ = iter.Close()

		if count != 3 {
			t.Errorf("expected 3 objects, got %d", count)
		}

		// Get with offset
		iter, err = b.List(ctx, "", 5, 5, nil)
		if err != nil {
			t.Fatalf("List with offset: %v", err)
		}

		count = 0
		for {
			obj, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if obj == nil {
				break
			}
			count++
		}
		_ = iter.Close()

		if count != 5 {
			t.Errorf("expected 5 objects with offset 5, got %d", count)
		}
	})

	t.Run("List_DirsOnly", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/nested.txt", strings.NewReader("x"), 1, "text/plain", nil)

		iter, err := b.List(ctx, "", 0, 0, storage.Options{"dirs_only": true})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		for {
			obj, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if obj == nil {
				break
			}
			if !obj.IsDir {
				t.Errorf("dirs_only returned non-directory %q", obj.Key)
			}
		}
	})

	t.Run("List_FilesOnly", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/nested.txt", strings.NewReader("x"), 1, "text/plain", nil)

		iter, err := b.List(ctx, "", 0, 0, storage.Options{"files_only": true})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		for {
			obj, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if obj == nil {
				break
			}
			if obj.IsDir {
				t.Errorf("files_only returned directory %q", obj.Key)
			}
		}
	})

	t.Run("List_Empty", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "empty", nil)
		b := st.Bucket("empty")

		iter, err := b.List(ctx, "", 0, 0, nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		obj, err := iter.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if obj != nil {
			t.Error("expected empty list")
		}
	})

	t.Run("SignedURL", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		// SignedURL may not be supported by all backends
		_, err := b.SignedURL(ctx, "file.txt", "GET", time.Hour, nil)
		// Just verify it doesn't panic; error is expected for local storage
		_ = err
	})

	t.Run("Info", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "info-test", nil)
		b := st.Bucket("info-test")

		info, err := b.Info(ctx)
		if err != nil {
			t.Fatalf("Info: %v", err)
		}

		if info.Name != "info-test" {
			t.Errorf("expected name 'info-test', got %q", info.Name)
		}
	})

	t.Run("Info_NotExist", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		b := st.Bucket("nonexistent")
		_, err := b.Info(ctx)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()

		_, _ = st.CreateBucket(context.Background(), "data", nil)
		b := st.Bucket("data")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := b.Write(ctx, "test.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if err == nil {
			t.Error("expected error for cancelled context")
		}

		_, _, err = b.Open(ctx, "test.txt", 0, 0, nil)
		if err == nil {
			t.Error("expected error for cancelled context")
		}

		_, err = b.Stat(ctx, "test.txt", nil)
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})
}

// iteratorTests tests iterator behavior.
func iteratorTests(t *testing.T, factory StorageFactory) {
	t.Run("BucketIter_MultipleClose", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		iter, err := st.Buckets(ctx, 0, 0, nil)
		if err != nil {
			t.Fatalf("Buckets: %v", err)
		}

		// Close multiple times should not panic
		if err := iter.Close(); err != nil {
			t.Errorf("first Close: %v", err)
		}
		if err := iter.Close(); err != nil {
			t.Errorf("second Close: %v", err)
		}
	})

	t.Run("ObjectIter_MultipleClose", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		iter, err := b.List(ctx, "", 0, 0, nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}

		if err := iter.Close(); err != nil {
			t.Errorf("first Close: %v", err)
		}
		if err := iter.Close(); err != nil {
			t.Errorf("second Close: %v", err)
		}
	})

	t.Run("BucketIter_Sorted", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		names := []string{"zebra", "alpha", "mango"}
		for _, n := range names {
			_, _ = st.CreateBucket(ctx, n, nil)
		}

		iter, err := st.Buckets(ctx, 0, 0, nil)
		if err != nil {
			t.Fatalf("Buckets: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		var found []string
		for {
			info, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if info == nil {
				break
			}
			found = append(found, info.Name)
		}

		// Filter only our buckets
		var ours []string
		for _, f := range found {
			for _, n := range names {
				if f == n {
					ours = append(ours, f)
					break
				}
			}
		}

		// Check sorted
		if !sort.StringsAreSorted(ours) {
			t.Errorf("bucket list not sorted: %v", ours)
		}
	})

	t.Run("ObjectIter_Sorted", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		names := []string{"z.txt", "a.txt", "m.txt"}
		for _, n := range names {
			_, _ = b.Write(ctx, n, strings.NewReader("x"), 1, "text/plain", nil)
		}

		iter, err := b.List(ctx, "", 0, 0, nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		var keys []string
		for {
			obj, err := iter.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if obj == nil {
				break
			}
			keys = append(keys, obj.Key)
		}

		if !sort.StringsAreSorted(keys) {
			t.Errorf("object list not sorted: %v", keys)
		}
	})
}

// edgeCaseTests tests edge cases and error conditions.
func edgeCaseTests(t *testing.T, factory StorageFactory) {
	t.Run("EmptyBucketName", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()

		// Empty bucket name should use default or error
		b := st.Bucket("")
		// Just verify it doesn't panic
		_ = b.Name()
	})

	t.Run("EmptyKey", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "", strings.NewReader("x"), 1, "text/plain", nil)
		if err == nil {
			t.Error("expected error for empty key")
		}

		_, _, err = b.Open(ctx, "", 0, 0, nil)
		if err == nil {
			t.Error("expected error for empty key")
		}
	})

	t.Run("PathTraversal", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		// Try path traversal
		_, err := b.Write(ctx, "../escape.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if err == nil {
			t.Error("expected error for path traversal")
		}

		_, err = b.Write(ctx, "foo/../../escape.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if err == nil {
			t.Error("expected error for path traversal")
		}

		_, _, err = b.Open(ctx, "../escape.txt", 0, 0, nil)
		if err == nil {
			t.Error("expected error for path traversal in Open")
		}
	})

	t.Run("BackslashNormalization", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		// Backslashes should be normalized to forward slashes
		_, err := b.Write(ctx, `dir\file.txt`, strings.NewReader("x"), 1, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write with backslash: %v", err)
		}

		// Should be accessible with forward slash
		_, err = b.Stat(ctx, "dir/file.txt", nil)
		if err != nil {
			t.Errorf("Stat with forward slash: %v", err)
		}
	})

	t.Run("LeadingSlash", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		// Leading slash should be stripped
		_, err := b.Write(ctx, "/file.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write with leading slash: %v", err)
		}

		_, err = b.Stat(ctx, "file.txt", nil)
		if err != nil {
			t.Errorf("Stat without leading slash: %v", err)
		}
	})

	t.Run("WhitespaceKey", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "   ", strings.NewReader("x"), 1, "text/plain", nil)
		if err == nil {
			t.Error("expected error for whitespace-only key")
		}
	})

	t.Run("SpecialBucketNames", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()

		// Test that special names are sanitized
		b := st.Bucket("test/bucket")
		if strings.Contains(b.Name(), "/") {
			t.Errorf("bucket name should sanitize /: %q", b.Name())
		}
	})

	t.Run("LargeFile", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		// Write 1MB file
		size := int64(1024 * 1024)
		data := bytes.Repeat([]byte("x"), int(size))

		_, err := b.Write(ctx, "large.bin", bytes.NewReader(data), size, "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("Write large: %v", err)
		}

		obj, err := b.Stat(ctx, "large.bin", nil)
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}

		if obj.Size != size {
			t.Errorf("expected size %d, got %d", size, obj.Size)
		}
	})

	t.Run("NilOptions", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		// All operations should work with nil options
		_, err := st.CreateBucket(ctx, "nil-opts", nil)
		if err != nil {
			t.Fatalf("CreateBucket: %v", err)
		}

		b := st.Bucket("nil-opts")
		_, err = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		_, _, err = b.Open(ctx, "file.txt", 0, 0, nil)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}

		_, err = b.List(ctx, "", 0, 0, nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
	})

	t.Run("OpenDirectory", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "dir/file.txt", strings.NewReader("x"), 1, "text/plain", nil)

		// Opening a directory should fail
		_, _, err := b.Open(ctx, "dir", 0, 0, nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for opening directory, got %v", err)
		}
	})

	t.Run("NegativeOffset", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		_, _ = st.Buckets(ctx, 10, -5, nil)                 // Should handle gracefully
		_, _ = st.Bucket("data").List(ctx, "", 10, -5, nil) // Should handle gracefully
	})
}

// concurrencyTests tests concurrent access patterns.
func concurrencyTests(t *testing.T, factory StorageFactory) {
	t.Run("ConcurrentWrites", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "concurrent", nil)
		b := st.Bucket("concurrent")

		var wg sync.WaitGroup
		errs := make(chan error, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				key := string(rune('a'+(n%26))) + "_" + string(rune('0'+(n/26))) + ".txt"
				_, err := b.Write(ctx, key, strings.NewReader("data"), 4, "text/plain", nil)
				if err != nil {
					errs <- err
				}
			}(i)
		}

		wg.Wait()
		close(errs)

		for err := range errs {
			t.Errorf("concurrent write error: %v", err)
		}
	})

	t.Run("ConcurrentReads", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "concurrent", nil)
		b := st.Bucket("concurrent")

		// Write a file first
		_, err := b.Write(ctx, "shared.txt", strings.NewReader("shared data"), 11, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		var wg sync.WaitGroup
		errs := make(chan error, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rc, _, err := b.Open(ctx, "shared.txt", 0, 0, nil)
				if err != nil {
					errs <- err
					return
				}
				_, err = io.ReadAll(rc)
				_ = rc.Close()
				if err != nil {
					errs <- err
				}
			}()
		}

		wg.Wait()
		close(errs)

		for err := range errs {
			t.Errorf("concurrent read error: %v", err)
		}
	})

	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "concurrent", nil)
		b := st.Bucket("concurrent")

		// Initial write
		_, _ = b.Write(ctx, "rw.txt", strings.NewReader("initial"), 7, "text/plain", nil)

		var wg sync.WaitGroup
		errs := make(chan error, 200)

		// Concurrent readers
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					rc, _, err := b.Open(ctx, "rw.txt", 0, 0, nil)
					if err != nil {
						// File might not exist during write, that's ok
						if !errors.Is(err, storage.ErrNotExist) {
							errs <- err
						}
						continue
					}
					_, _ = io.ReadAll(rc)
					_ = rc.Close()
				}
			}()
		}

		// Concurrent writers
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					data := strings.Repeat("x", n%100+1)
					_, err := b.Write(ctx, "rw.txt", strings.NewReader(data), int64(len(data)), "text/plain", nil)
					if err != nil {
						errs <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errs)

		errCount := 0
		for err := range errs {
			t.Logf("concurrent rw error: %v", err)
			errCount++
		}

		// Some errors might be acceptable due to race conditions
		// The important thing is no panics or data corruption
		if errCount > 10 {
			t.Errorf("too many concurrent rw errors: %d", errCount)
		}
	})

	t.Run("ConcurrentList", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "concurrent", nil)
		b := st.Bucket("concurrent")

		// Write some files
		for i := 0; i < 50; i++ {
			_, _ = b.Write(ctx, string(rune('a'+i%26))+string(rune('0'+i/26))+".txt", strings.NewReader("x"), 1, "text/plain", nil)
		}

		var wg sync.WaitGroup
		errs := make(chan error, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				iter, err := b.List(ctx, "", 0, 0, nil)
				if err != nil {
					errs <- err
					return
				}
				for {
					obj, err := iter.Next()
					if err != nil {
						errs <- err
						break
					}
					if obj == nil {
						break
					}
				}
				_ = iter.Close()
			}()
		}

		wg.Wait()
		close(errs)

		for err := range errs {
			t.Errorf("concurrent list error: %v", err)
		}
	})

	t.Run("ConcurrentBucketOps", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		var wg sync.WaitGroup
		errs := make(chan error, 100)

		// Concurrent bucket creation/deletion
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				name := "concurrent-" + string(rune('a'+n%26))

				_, err := st.CreateBucket(ctx, name, nil)
				if err != nil && !errors.Is(err, storage.ErrExist) {
					errs <- err
				}

				_, err = st.Buckets(ctx, 0, 0, nil)
				if err != nil {
					errs <- err
				}

				// Try to delete (might fail if others created)
				_ = st.DeleteBucket(ctx, name, storage.Options{"force": true})
			}(i)
		}

		wg.Wait()
		close(errs)

		for err := range errs {
			t.Errorf("concurrent bucket ops error: %v", err)
		}
	})

	t.Run("ConcurrentCopyMove", func(t *testing.T) {
		st, cleanup := factory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "concurrent", nil)
		b := st.Bucket("concurrent")

		// Create source files
		for i := 0; i < 20; i++ {
			_, _ = b.Write(ctx, "src"+string(rune('a'+i))+".txt", strings.NewReader("data"), 4, "text/plain", nil)
		}

		var wg sync.WaitGroup
		errs := make(chan error, 100)

		// Concurrent copies
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				src := "src" + string(rune('a'+n)) + ".txt"
				dst := "copy" + string(rune('a'+n)) + ".txt"
				_, err := b.Copy(ctx, dst, "concurrent", src, nil)
				if err != nil && !errors.Is(err, storage.ErrNotExist) {
					errs <- err
				}
			}(i)
		}

		wg.Wait()
		close(errs)

		for err := range errs {
			t.Errorf("concurrent copy error: %v", err)
		}
	})
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
