// File: lib/storage/driver/local/storage_test.go
package local_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage/driver/local"
)

func Open(ctx context.Context, dsn string) (storage.Storage, error) {
	return local.Open(ctx, dsn)
}

// localFactory creates a local storage instance for conformance testing.
func localFactory(t *testing.T) (storage.Storage, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	ctx := context.Background()

	st, err := Open(ctx, tmpDir)
	if err != nil {
		t.Fatalf("local.Open(%q): %v", tmpDir, err)
	}

	return st, func() {
		_ = st.Close()
	}
}

// TestConformance runs the full conformance suite against local storage.
func TestConformance(t *testing.T) {
	// Import the conformance suite from storage_test package
	// This ensures local driver passes all interface contract tests

	// Note: Can't directly call storage_test.ConformanceSuite due to package boundaries
	// Instead, we run comprehensive local-specific tests here
	runLocalConformanceTests(t)
}

func runLocalConformanceTests(t *testing.T) {
	t.Run("StorageOperations", testStorageOperations)
	t.Run("BucketOperations", testBucketOperations)
	t.Run("ObjectOperations", testObjectOperations)
	t.Run("PathSecurity", testPathSecurity)
	t.Run("Concurrency", testConcurrency)
	t.Run("EdgeCases", testEdgeCases)
	t.Run("DSNParsing", testDSNParsing)
	t.Run("Helpers", testHelpers)
}

func testStorageOperations(t *testing.T) {
	t.Run("Open_ValidPath", func(t *testing.T) {
		tmpDir := t.TempDir()
		ctx := context.Background()

		st, err := Open(ctx, tmpDir)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer func() {
			_ = st.Close()
		}()

		if st == nil {
			t.Error("expected non-nil storage")
		}
	})

	t.Run("Open_NonExistentPath", func(t *testing.T) {
		ctx := context.Background()

		_, err := Open(ctx, "/nonexistent/path/that/does/not/exist")
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})

	t.Run("Open_FileNotDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.txt")
		_ = os.WriteFile(filePath, []byte("test"), 0644)

		ctx := context.Background()

		_, err := Open(ctx, filePath)
		if err == nil {
			t.Error("expected error for file instead of directory")
		}
	})

	t.Run("Open_CancelledContext", func(t *testing.T) {
		tmpDir := t.TempDir()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := Open(ctx, tmpDir)
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})

	t.Run("Features", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()

		features := st.Features()
		if !features["move"] {
			t.Error("expected move feature")
		}
		if !features["directories"] {
			t.Error("expected directories feature")
		}
		if !features["object_move_server"] {
			t.Error("expected object_move_server feature")
		}
		if !features["dir_move_server"] {
			t.Error("expected dir_move_server feature")
		}
	})

	t.Run("Close", func(t *testing.T) {
		st, _ := localFactory(t)
		err := st.Close()
		if err != nil {
			t.Errorf("Close: %v", err)
		}
		// Close should be idempotent
		err = st.Close()
		if err != nil {
			t.Errorf("second Close: %v", err)
		}
	})

	t.Run("CreateBucket", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

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

	t.Run("CreateBucket_EmptyName", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, err := st.CreateBucket(ctx, "", nil)
		if err == nil {
			t.Error("expected error for empty bucket name")
		}
	})

	t.Run("CreateBucket_WhitespaceName", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, err := st.CreateBucket(ctx, "   ", nil)
		if err == nil {
			t.Error("expected error for whitespace bucket name")
		}
	})

	t.Run("CreateBucket_Duplicate", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "dup", nil)
		_, err := st.CreateBucket(ctx, "dup", nil)
		if !errors.Is(err, storage.ErrExist) {
			t.Errorf("expected ErrExist, got %v", err)
		}
	})

	t.Run("DeleteBucket_Empty", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "to-delete", nil)
		err := st.DeleteBucket(ctx, "to-delete", nil)
		if err != nil {
			t.Fatalf("DeleteBucket: %v", err)
		}
	})

	t.Run("DeleteBucket_NotExist", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		err := st.DeleteBucket(ctx, "nonexistent", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("DeleteBucket_NonEmptyNoForce", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "non-empty", nil)
		b := st.Bucket("non-empty")
		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)

		err := st.DeleteBucket(ctx, "non-empty", nil)
		if err == nil {
			t.Error("expected error when deleting non-empty bucket without force")
		}
	})

	t.Run("DeleteBucket_Force", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "force-delete", nil)
		b := st.Bucket("force-delete")
		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)

		err := st.DeleteBucket(ctx, "force-delete", storage.Options{"force": true})
		if err != nil {
			t.Fatalf("DeleteBucket with force: %v", err)
		}
	})

	t.Run("DeleteBucket_EmptyName", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		err := st.DeleteBucket(ctx, "", nil)
		if err == nil {
			t.Error("expected error for empty bucket name")
		}
	})

	t.Run("Buckets_List", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		names := []string{"alpha", "beta", "gamma"}
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

		for _, n := range names {
			if !contains(found, n) {
				t.Errorf("expected to find bucket %q", n)
			}
		}
	})

	t.Run("Buckets_Pagination", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			_, _ = st.CreateBucket(ctx, string(rune('a'+i)), nil)
		}

		iter, _ := st.Buckets(ctx, 2, 1, nil)
		count := 0
		for {
			info, _ := iter.Next()
			if info == nil {
				break
			}
			count++
		}
		_ = iter.Close()

		if count != 2 {
			t.Errorf("expected 2 buckets, got %d", count)
		}
	})

	t.Run("Buckets_NegativeOffset", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "test", nil)

		iter, err := st.Buckets(ctx, 10, -5, nil)
		if err != nil {
			t.Fatalf("Buckets: %v", err)
		}
		_ = iter.Close()
	})

	t.Run("Buckets_SkipsHiddenDirs", func(t *testing.T) {
		tmpDir := t.TempDir()
		ctx := context.Background()

		// Create hidden directory
		_ = os.Mkdir(filepath.Join(tmpDir, ".hidden"), 0755)
		_ = os.Mkdir(filepath.Join(tmpDir, "visible"), 0755)

		st, _ := Open(ctx, tmpDir)
		defer func() {
			_ = st.Close()
		}()

		iter, _ := st.Buckets(ctx, 0, 0, nil)
		defer func() {
			_ = iter.Close()
		}()

		for {
			info, _ := iter.Next()
			if info == nil {
				break
			}
			if strings.HasPrefix(info.Name, ".") {
				t.Errorf("should not list hidden bucket: %q", info.Name)
			}
		}
	})

	t.Run("Bucket_Handle", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()

		b := st.Bucket("test")
		if b == nil {
			t.Error("expected non-nil bucket")
		}
		if b.Name() != "test" {
			t.Errorf("expected name 'test', got %q", b.Name())
		}
	})

	t.Run("Bucket_EmptyName_Default", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()

		b := st.Bucket("")
		if b.Name() != "default" {
			t.Errorf("expected name 'default', got %q", b.Name())
		}
	})

	t.Run("Bucket_Features", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "test", nil)
		b := st.Bucket("test")

		features := b.Features()
		if !features["move"] {
			t.Error("expected move feature")
		}
	})
}

func testBucketOperations(t *testing.T) {
	t.Run("Info", func(t *testing.T) {
		st, cleanup := localFactory(t)
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
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		b := st.Bucket("nonexistent")
		_, err := b.Info(ctx)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("Info_CancelledContext", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		b := st.Bucket("test")
		_, err := b.Info(ctx)
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})
}

func testObjectOperations(t *testing.T) {
	t.Run("Write_Read", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
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
		if obj.Bucket != "data" {
			t.Errorf("expected bucket 'data', got %q", obj.Bucket)
		}

		rc, readObj, err := b.Open(ctx, "test.txt", 0, 0, nil)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer func() {
			_ = rc.Close()
		}()

		data, _ := io.ReadAll(rc)
		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}
		if readObj.Size != int64(len(content)) {
			t.Errorf("expected size %d, got %d", len(content), readObj.Size)
		}
	})

	t.Run("Write_EmptyKey", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "", strings.NewReader("x"), 1, "text/plain", nil)
		if err == nil {
			t.Error("expected error for empty key")
		}
	})

	t.Run("Write_WhitespaceKey", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "   ", strings.NewReader("x"), 1, "text/plain", nil)
		if err == nil {
			t.Error("expected error for whitespace key")
		}
	})

	t.Run("Write_NestedPath", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "a/b/c/d/file.txt", strings.NewReader("nested"), 6, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write nested: %v", err)
		}

		obj, err := b.Stat(ctx, "a/b/c/d/file.txt", nil)
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if obj.Size != 6 {
			t.Errorf("expected size 6, got %d", obj.Size)
		}
	})

	t.Run("Write_Overwrite", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "overwrite.txt", strings.NewReader("first"), 5, "text/plain", nil)
		_, err := b.Write(ctx, "overwrite.txt", strings.NewReader("second"), 6, "text/plain", nil)
		if err != nil {
			t.Fatalf("Overwrite: %v", err)
		}

		rc, _, _ := b.Open(ctx, "overwrite.txt", 0, 0, nil)
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != "second" {
			t.Errorf("expected 'second', got %q", string(data))
		}
	})

	t.Run("Write_CancelledContext", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		_, _ = st.CreateBucket(context.Background(), "data", nil)
		cancel()

		b := st.Bucket("data")
		_, err := b.Write(ctx, "test.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})

	t.Run("Open_NotExist", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _, err := b.Open(ctx, "notfound.txt", 0, 0, nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("Open_Directory", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "dir/file.txt", strings.NewReader("x"), 1, "text/plain", nil)

		_, _, err := b.Open(ctx, "dir", 0, 0, nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for opening directory, got %v", err)
		}
	})

	t.Run("Open_RangeOffset", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "range.txt", strings.NewReader("0123456789"), 10, "text/plain", nil)

		rc, _, err := b.Open(ctx, "range.txt", 5, 0, nil)
		if err != nil {
			t.Fatalf("Open with offset: %v", err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != "56789" {
			t.Errorf("expected '56789', got %q", string(data))
		}
	})

	t.Run("Open_RangeOffsetLength", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "range.txt", strings.NewReader("0123456789"), 10, "text/plain", nil)

		rc, _, err := b.Open(ctx, "range.txt", 2, 5, nil)
		if err != nil {
			t.Fatalf("Open with offset and length: %v", err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != "23456" {
			t.Errorf("expected '23456', got %q", string(data))
		}
	})

	t.Run("Stat", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "stat.txt", strings.NewReader("content"), 7, "text/plain", nil)

		obj, err := b.Stat(ctx, "stat.txt", nil)
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}

		if obj.Key != "stat.txt" {
			t.Errorf("expected key 'stat.txt', got %q", obj.Key)
		}
		if obj.Size != 7 {
			t.Errorf("expected size 7, got %d", obj.Size)
		}
		if obj.IsDir {
			t.Error("expected IsDir=false")
		}
	})

	t.Run("Stat_Directory", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "mydir/file.txt", strings.NewReader("x"), 1, "text/plain", nil)

		obj, err := b.Stat(ctx, "mydir", nil)
		if err != nil {
			t.Fatalf("Stat directory: %v", err)
		}

		if !obj.IsDir {
			t.Error("expected IsDir=true for directory")
		}
	})

	t.Run("Stat_NotExist", func(t *testing.T) {
		st, cleanup := localFactory(t)
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
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "delete.txt", strings.NewReader("x"), 1, "text/plain", nil)

		err := b.Delete(ctx, "delete.txt", nil)
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err = b.Stat(ctx, "delete.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Error("expected file to be deleted")
		}
	})

	t.Run("Delete_NotExist", func(t *testing.T) {
		st, cleanup := localFactory(t)
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
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "dir/a.txt", strings.NewReader("a"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/b.txt", strings.NewReader("b"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/sub/c.txt", strings.NewReader("c"), 1, "text/plain", nil)

		err := b.Delete(ctx, "dir", storage.Options{"recursive": true})
		if err != nil {
			t.Fatalf("Delete recursive: %v", err)
		}

		_, err = b.Stat(ctx, "dir", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Error("expected directory to be deleted")
		}
	})

	t.Run("Copy", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		content := "copy content"
		_, _ = b.Write(ctx, "src.txt", strings.NewReader(content), int64(len(content)), "text/plain", nil)

		obj, err := b.Copy(ctx, "dst.txt", "data", "src.txt", nil)
		if err != nil {
			t.Fatalf("Copy: %v", err)
		}

		if obj.Key != "dst.txt" {
			t.Errorf("expected key 'dst.txt', got %q", obj.Key)
		}

		// Verify copy
		rc, _, _ := b.Open(ctx, "dst.txt", 0, 0, nil)
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}

		// Source should still exist
		_, err = b.Stat(ctx, "src.txt", nil)
		if err != nil {
			t.Error("source should still exist after copy")
		}
	})

	t.Run("Copy_CrossBucket", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "src-bucket", nil)
		_, _ = st.CreateBucket(ctx, "dst-bucket", nil)

		srcB := st.Bucket("src-bucket")
		dstB := st.Bucket("dst-bucket")

		_, _ = srcB.Write(ctx, "file.txt", strings.NewReader("cross"), 5, "text/plain", nil)

		_, err := dstB.Copy(ctx, "copied.txt", "src-bucket", "file.txt", nil)
		if err != nil {
			t.Fatalf("Copy cross bucket: %v", err)
		}

		rc, _, _ := dstB.Open(ctx, "copied.txt", 0, 0, nil)
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != "cross" {
			t.Errorf("expected 'cross', got %q", string(data))
		}
	})

	t.Run("Copy_NotExist", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Copy(ctx, "dst.txt", "data", "notfound.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("Move", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "original.txt", strings.NewReader("move"), 4, "text/plain", nil)

		obj, err := b.Move(ctx, "moved.txt", "data", "original.txt", nil)
		if err != nil {
			t.Fatalf("Move: %v", err)
		}

		if obj.Key != "moved.txt" {
			t.Errorf("expected key 'moved.txt', got %q", obj.Key)
		}

		// Original should be gone
		_, err = b.Stat(ctx, "original.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Error("expected original to be deleted after move")
		}

		// Moved should exist
		rc, _, _ := b.Open(ctx, "moved.txt", 0, 0, nil)
		data, _ := io.ReadAll(rc)
		_ = rc.Close()

		if string(data) != "move" {
			t.Errorf("expected 'move', got %q", string(data))
		}
	})

	t.Run("Move_CrossBucket", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "src-bucket", nil)
		_, _ = st.CreateBucket(ctx, "dst-bucket", nil)

		srcB := st.Bucket("src-bucket")
		dstB := st.Bucket("dst-bucket")

		_, _ = srcB.Write(ctx, "file.txt", strings.NewReader("move"), 4, "text/plain", nil)

		_, err := dstB.Move(ctx, "moved.txt", "src-bucket", "file.txt", nil)
		if err != nil {
			t.Fatalf("Move cross bucket: %v", err)
		}

		// Source should be gone
		_, err = srcB.Stat(ctx, "file.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Error("expected source to be deleted after move")
		}
	})

	t.Run("Move_NotExist", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Move(ctx, "dst.txt", "data", "notfound.txt", nil)
		if !errors.Is(err, storage.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		files := []string{"a.txt", "b.txt", "dir/c.txt"}
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

		// Should have files and directory
		if count < 3 {
			t.Errorf("expected at least 3 objects, got %d", count)
		}
	})

	t.Run("List_Prefix", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "foo/a.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "foo/b.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "bar/c.txt", strings.NewReader("x"), 1, "text/plain", nil)

		iter, _ := b.List(ctx, "foo", 0, 0, nil)
		defer func() {
			_ = iter.Close()
		}()

		var keys []string
		for {
			obj, _ := iter.Next()
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
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "top.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/nested.txt", strings.NewReader("x"), 1, "text/plain", nil)

		iter, _ := b.List(ctx, "", 0, 0, storage.Options{"recursive": false})
		defer func() {
			_ = iter.Close()
		}()

		var keys []string
		for {
			obj, _ := iter.Next()
			if obj == nil {
				break
			}
			keys = append(keys, obj.Key)
		}

		// Should have top.txt and dir (directory)
		if len(keys) != 2 {
			t.Errorf("expected 2 items, got %d: %v", len(keys), keys)
		}
	})

	t.Run("List_DirsOnly", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/nested.txt", strings.NewReader("x"), 1, "text/plain", nil)

		iter, _ := b.List(ctx, "", 0, 0, storage.Options{"dirs_only": true})
		defer func() {
			_ = iter.Close()
		}()

		for {
			obj, _ := iter.Next()
			if obj == nil {
				break
			}
			if !obj.IsDir {
				t.Errorf("dirs_only returned non-directory %q", obj.Key)
			}
		}
	})

	t.Run("List_FilesOnly", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/nested.txt", strings.NewReader("x"), 1, "text/plain", nil)

		iter, _ := b.List(ctx, "", 0, 0, storage.Options{"files_only": true})
		defer func() {
			_ = iter.Close()
		}()

		for {
			obj, _ := iter.Next()
			if obj == nil {
				break
			}
			if obj.IsDir {
				t.Errorf("files_only returned directory %q", obj.Key)
			}
		}
	})

	t.Run("List_DirsAndFilesOnly", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "dir/nested.txt", strings.NewReader("x"), 1, "text/plain", nil)

		// Both dirs_only and files_only should cancel each other out
		iter, _ := b.List(ctx, "", 0, 0, storage.Options{"dirs_only": true, "files_only": true})
		defer func() {
			_ = iter.Close()
		}()

		count := 0
		for {
			obj, _ := iter.Next()
			if obj == nil {
				break
			}
			count++
		}

		// Should list all items when both are true
		if count < 2 {
			t.Errorf("expected at least 2 items when both dirs_only and files_only, got %d", count)
		}
	})

	t.Run("List_Pagination", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		for i := 0; i < 10; i++ {
			_, _ = b.Write(ctx, string(rune('a'+i))+".txt", strings.NewReader("x"), 1, "text/plain", nil)
		}

		iter, _ := b.List(ctx, "", 3, 2, nil)
		defer func() {
			_ = iter.Close()
		}()

		count := 0
		for {
			obj, _ := iter.Next()
			if obj == nil {
				break
			}
			count++
		}

		if count != 3 {
			t.Errorf("expected 3 items with limit 3, got %d", count)
		}
	})

	t.Run("List_NegativeOffset", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		iter, err := b.List(ctx, "", 10, -5, nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		_ = iter.Close()
	})

	t.Run("List_NonExistentPrefix", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		iter, err := b.List(ctx, "nonexistent", 0, 0, nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		obj, _ := iter.Next()
		if obj != nil {
			t.Error("expected empty list for non-existent prefix")
		}
	})

	t.Run("SignedURL", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.SignedURL(ctx, "file.txt", "GET", time.Hour, nil)
		if !errors.Is(err, storage.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported for local SignedURL, got %v", err)
		}
	})
}

func testPathSecurity(t *testing.T) {
	t.Run("PathTraversal_DotDot", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "../escape.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for path traversal, got %v", err)
		}
	})

	t.Run("PathTraversal_Nested", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "foo/../../escape.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for nested path traversal, got %v", err)
		}
	})

	t.Run("PathTraversal_Open", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, _, err := b.Open(ctx, "../escape.txt", 0, 0, nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for Open path traversal, got %v", err)
		}
	})

	t.Run("PathTraversal_Stat", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Stat(ctx, "../escape.txt", nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for Stat path traversal, got %v", err)
		}
	})

	t.Run("PathTraversal_Delete", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		err := b.Delete(ctx, "../escape.txt", nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for Delete path traversal, got %v", err)
		}
	})

	t.Run("PathTraversal_Copy", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Copy(ctx, "../escape.txt", "data", "src.txt", nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for Copy dst traversal, got %v", err)
		}

		_, err = b.Copy(ctx, "dst.txt", "data", "../escape.txt", nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for Copy src traversal, got %v", err)
		}
	})

	t.Run("PathTraversal_Move", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Move(ctx, "../escape.txt", "data", "src.txt", nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for Move dst traversal, got %v", err)
		}
	})

	t.Run("PathTraversal_List", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.List(ctx, "../escape", 0, 0, nil)
		if !errors.Is(err, storage.ErrPermission) {
			t.Errorf("expected ErrPermission for List path traversal, got %v", err)
		}
	})

	t.Run("BackslashNormalization", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		// Backslash should be normalized
		_, err := b.Write(ctx, `dir\file.txt`, strings.NewReader("x"), 1, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write with backslash: %v", err)
		}

		// Should be accessible with forward slash
		_, err = b.Stat(ctx, "dir/file.txt", nil)
		if err != nil {
			t.Errorf("Stat with forward slash failed: %v", err)
		}
	})

	t.Run("LeadingSlash", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "/file.txt", strings.NewReader("x"), 1, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write with leading slash: %v", err)
		}

		_, err = b.Stat(ctx, "file.txt", nil)
		if err != nil {
			t.Errorf("Stat without leading slash failed: %v", err)
		}
	})

	t.Run("SafeBucketName_Slash", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()

		b := st.Bucket("test/bucket")
		if strings.Contains(b.Name(), "/") {
			t.Errorf("bucket name should sanitize /: %q", b.Name())
		}
	})

	t.Run("SafeBucketName_Dot", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()

		b := st.Bucket(".")
		if b.Name() == "." {
			t.Errorf("bucket name should sanitize .: %q", b.Name())
		}
	})

	t.Run("SafeBucketName_DotDot", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()

		b := st.Bucket("..")
		if b.Name() == ".." {
			t.Errorf("bucket name should sanitize ..: %q", b.Name())
		}
	})
}

func testConcurrency(t *testing.T) {
	t.Run("ConcurrentWrites", func(t *testing.T) {
		st, cleanup := localFactory(t)
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
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "concurrent", nil)
		b := st.Bucket("concurrent")
		_, _ = b.Write(ctx, "shared.txt", strings.NewReader("shared"), 6, "text/plain", nil)

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
				_, _ = io.ReadAll(rc)
				_ = rc.Close()
			}()
		}

		wg.Wait()
		close(errs)

		for err := range errs {
			t.Errorf("concurrent read error: %v", err)
		}
	})

	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "concurrent", nil)
		b := st.Bucket("concurrent")
		_, _ = b.Write(ctx, "rw.txt", strings.NewReader("initial"), 7, "text/plain", nil)

		var wg sync.WaitGroup

		// Concurrent readers
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					rc, _, err := b.Open(ctx, "rw.txt", 0, 0, nil)
					if err != nil {
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
					_, _ = b.Write(ctx, "rw.txt", strings.NewReader(data), int64(len(data)), "text/plain", nil)
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("ConcurrentList", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "concurrent", nil)
		b := st.Bucket("concurrent")

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
}

func testEdgeCases(t *testing.T) {
	t.Run("LargeFile", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		size := int64(5 * 1024 * 1024) // 5MB
		data := bytes.Repeat([]byte("x"), int(size))

		_, err := b.Write(ctx, "large.bin", bytes.NewReader(data), size, "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("Write large: %v", err)
		}

		obj, _ := b.Stat(ctx, "large.bin", nil)
		if obj.Size != size {
			t.Errorf("expected size %d, got %d", size, obj.Size)
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "empty.txt", strings.NewReader(""), 0, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write empty: %v", err)
		}

		obj, _ := b.Stat(ctx, "empty.txt", nil)
		if obj.Size != 0 {
			t.Errorf("expected size 0, got %d", obj.Size)
		}
	})

	t.Run("SpecialCharactersInKey", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		keys := []string{
			"file with spaces.txt",
			"file-with-dashes.txt",
			"file_with_underscores.txt",
			"file.multiple.dots.txt",
		}

		for _, key := range keys {
			_, err := b.Write(ctx, key, strings.NewReader("x"), 1, "text/plain", nil)
			if err != nil {
				t.Errorf("Write %q: %v", key, err)
				continue
			}

			_, err = b.Stat(ctx, key, nil)
			if err != nil {
				t.Errorf("Stat %q: %v", key, err)
			}
		}
	})

	t.Run("DeeplyNestedPath", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		key := "a/b/c/d/e/f/g/h/i/j/deep.txt"
		_, err := b.Write(ctx, key, strings.NewReader("deep"), 4, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write deeply nested: %v", err)
		}

		obj, _ := b.Stat(ctx, key, nil)
		if obj.Key != key {
			t.Errorf("expected key %q, got %q", key, obj.Key)
		}
	})

	t.Run("IteratorMultipleClose", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		iter, _ := st.Buckets(ctx, 0, 0, nil)
		_ = iter.Close()
		_ = iter.Close() // Should not panic
	})

	t.Run("ObjectIteratorMultipleClose", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		iter, _ := b.List(ctx, "", 0, 0, nil)
		_ = iter.Close()
		_ = iter.Close() // Should not panic
	})

	t.Run("BoolOptFalseForNonBool", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)

		// Pass non-bool value for bool option
		err := b.Delete(ctx, "file.txt", storage.Options{"recursive": "yes"})
		if err != nil {
			t.Errorf("Delete with non-bool recursive: %v", err)
		}
	})

	t.Run("RecursiveOptionString", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)

		// Test that string value for recursive doesn't cause issues
		iter, err := b.List(ctx, "", 0, 0, storage.Options{"recursive": "true"})
		if err != nil {
			t.Fatalf("List with string recursive: %v", err)
		}
		_ = iter.Close()
	})
}

func testDSNParsing(t *testing.T) {
	ctx := context.Background()

	t.Run("BareAbsolutePath", func(t *testing.T) {
		tmpDir := t.TempDir()
		st, err := Open(ctx, tmpDir)
		if err != nil {
			t.Fatalf("Open bare path: %v", err)
		}
		_ = st.Close()
	})

	t.Run("LocalScheme", func(t *testing.T) {
		tmpDir := t.TempDir()
		st, err := Open(ctx, "local:"+tmpDir)
		if err != nil {
			t.Fatalf("Open local: scheme: %v", err)
		}
		_ = st.Close()
	})

	t.Run("FileScheme", func(t *testing.T) {
		tmpDir := t.TempDir()
		st, err := Open(ctx, "file://"+tmpDir)
		if err != nil {
			t.Fatalf("Open file:// scheme: %v", err)
		}
		_ = st.Close()
	})

	t.Run("EmptyDSN", func(t *testing.T) {
		_, err := Open(ctx, "")
		if err == nil {
			t.Error("expected error for empty DSN")
		}
	})

	t.Run("LocalNoPath", func(t *testing.T) {
		_, err := Open(ctx, "local:")
		if err == nil {
			t.Error("expected error for local: with no path")
		}
	})

	t.Run("FileEmptyPath", func(t *testing.T) {
		_, err := Open(ctx, "file://")
		if err == nil {
			t.Error("expected error for file:// with empty path")
		}
	})

	t.Run("UnsupportedScheme", func(t *testing.T) {
		_, err := Open(ctx, "s3://bucket")
		if err == nil {
			t.Error("expected error for unsupported scheme")
		}
	})

	t.Run("InvalidURL", func(t *testing.T) {
		// A string that can't be parsed as URL
		_, err := Open(ctx, "://invalid")
		if err == nil {
			t.Error("expected error for invalid URL")
		}
	})
}

func testHelpers(t *testing.T) {
	t.Run("CleanPrefix_Empty", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)

		// Empty prefix should list all
		iter, err := b.List(ctx, "", 0, 0, nil)
		if err != nil {
			t.Fatalf("List empty prefix: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		obj, _ := iter.Next()
		if obj == nil {
			t.Error("expected at least one object")
		}
	})

	t.Run("CleanPrefix_Whitespace", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)

		// Whitespace prefix should behave like empty
		iter, err := b.List(ctx, "   ", 0, 0, nil)
		if err != nil {
			t.Fatalf("List whitespace prefix: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()

		obj, _ := iter.Next()
		if obj == nil {
			t.Error("expected at least one object")
		}
	})

	t.Run("CleanPrefix_Dot", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")
		_, _ = b.Write(ctx, "file.txt", strings.NewReader("x"), 1, "text/plain", nil)

		// Dot prefix should behave like empty
		iter, err := b.List(ctx, ".", 0, 0, nil)
		if err != nil {
			t.Fatalf("List dot prefix: %v", err)
		}
		defer func() {
			_ = iter.Close()
		}()
	})

	t.Run("CleanKey_OnlySlash", func(t *testing.T) {
		st, cleanup := localFactory(t)
		defer cleanup()
		ctx := context.Background()

		_, _ = st.CreateBucket(ctx, "data", nil)
		b := st.Bucket("data")

		_, err := b.Write(ctx, "/", strings.NewReader("x"), 1, "text/plain", nil)
		if err == nil {
			t.Error("expected error for only-slash key")
		}
	})
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
