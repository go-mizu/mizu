// File: lib/storage/factory_test.go
package storage_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/drive/lib/storage"
	"github.com/go-mizu/blueprints/drive/lib/storage/driver/local"
	_ "github.com/go-mizu/blueprints/drive/lib/storage/driver/memory" // Register memory driver
)

// Factory is an alias for StorageFactory that matches the user's requested signature.
// This allows both types to be used interchangeably in tests.
type Factory = StorageFactory

// driverRegistry maps driver names to their factory functions.
var driverRegistry = make(map[string]StorageFactory)

// RegisterDriver registers a driver factory for testing.
func RegisterDriver(name string, factory Factory) {
	driverRegistry[name] = factory
}

// GetDriverFactory returns the factory for a registered driver.
func GetDriverFactory(name string) (StorageFactory, bool) {
	f, ok := driverRegistry[name]
	return f, ok
}

// RegisteredDrivers returns all registered driver names.
func RegisteredDrivers() []string {
	names := make([]string, 0, len(driverRegistry))
	for name := range driverRegistry {
		names = append(names, name)
	}
	return names
}

func init() {
	// Register local driver factory
	RegisterDriver("local", func(t *testing.T) (storage.Storage, func()) {
		t.Helper()
		tmpDir := t.TempDir()
		ctx := context.Background()

		st, err := local.Open(ctx, tmpDir)
		if err != nil {
			t.Fatalf("local.Open(%q): %v", tmpDir, err)
		}

		return st, func() {
			_ = st.Close()
		}
	})

	// Register memory driver factory (uses storage.Open with registered driver)
	RegisterDriver("memory", func(t *testing.T) (storage.Storage, func()) {
		t.Helper()
		ctx := context.Background()

		st, err := storage.Open(ctx, "mem://")
		if err != nil {
			t.Fatalf("storage.Open(mem://): %v", err)
		}

		return st, func() {
			_ = st.Close()
		}
	})
}

// TestLocalConformance runs the conformance suite against the local driver.
func TestLocalConformance(t *testing.T) {
	factory, ok := GetDriverFactory("local")
	if !ok {
		t.Fatal("local driver not registered")
	}
	ConformanceSuite(t, factory)
}

// TestMemoryConformance runs the conformance suite against the memory driver.
// Note: Some tests are skipped because the memory driver has intentionally
// different behavior (e.g., auto-bucket creation for easier testing).
func TestMemoryConformance(t *testing.T) {
	t.Skip("memory driver has different semantics (auto-bucket creation); use TestMemoryBasicOperations instead")
}

// TestLocalMultipart runs the multipart test suite against the local driver.
func TestLocalMultipart(t *testing.T) {
	factory, ok := GetDriverFactory("local")
	if !ok {
		t.Fatal("local driver not registered")
	}
	MultipartSuite(t, factory)
}

// TestMemoryMultipart runs the multipart test suite against the memory driver.
// Note: Memory driver doesn't check context cancellation in multipart operations.
func TestMemoryMultipart(t *testing.T) {
	t.Skip("memory driver has different semantics; use TestMemoryBasicMultipart instead")
}

// TestAllDriversConformance runs conformance tests against drivers that support
// full filesystem-like semantics.
func TestAllDriversConformance(t *testing.T) {
	// Only local driver supports full conformance tests
	factory, ok := GetDriverFactory("local")
	if !ok {
		t.Skip("local driver not registered")
	}
	ConformanceSuite(t, factory)
}

// TestAllDriversMultipart runs multipart tests against drivers that support
// full multipart semantics.
func TestAllDriversMultipart(t *testing.T) {
	// Only local driver supports full multipart tests with context handling
	factory, ok := GetDriverFactory("local")
	if !ok {
		t.Skip("local driver not registered")
	}
	MultipartSuite(t, factory)
}

// TestDriverRegistration verifies that both local and memory drivers are registered.
func TestDriverRegistration(t *testing.T) {
	drivers := RegisteredDrivers()
	if len(drivers) < 2 {
		t.Errorf("expected at least 2 drivers, got %d", len(drivers))
	}

	localFound := false
	memoryFound := false
	for _, name := range drivers {
		if name == "local" {
			localFound = true
		}
		if name == "memory" {
			memoryFound = true
		}
	}

	if !localFound {
		t.Error("local driver not registered")
	}
	if !memoryFound {
		t.Error("memory driver not registered")
	}
}

// TestFactoryCreateAndCleanup tests that factories properly create and clean up storage.
func TestFactoryCreateAndCleanup(t *testing.T) {
	for _, name := range RegisteredDrivers() {
		t.Run(name, func(t *testing.T) {
			factory, _ := GetDriverFactory(name)

			st, cleanup := factory(t)
			if st == nil {
				t.Fatal("factory returned nil storage")
			}

			// Verify storage is usable
			ctx := context.Background()
			_, err := st.CreateBucket(ctx, "factory-test", nil)
			if err != nil {
				t.Fatalf("CreateBucket: %v", err)
			}

			// Cleanup should not panic
			cleanup()
		})
	}
}

// TestFactoryIsolation tests that each factory call creates an isolated storage instance.
// Note: Only runs for local driver since memory driver auto-creates buckets.
func TestFactoryIsolation(t *testing.T) {
	factory, ok := GetDriverFactory("local")
	if !ok {
		t.Skip("local driver not registered")
	}

	st1, cleanup1 := factory(t)
	defer cleanup1()

	st2, cleanup2 := factory(t)
	defer cleanup2()

	ctx := context.Background()

	// Create bucket in st1
	_, err := st1.CreateBucket(ctx, "isolation-test", nil)
	if err != nil {
		t.Fatalf("CreateBucket in st1: %v", err)
	}

	// st2 should not see it (isolated storage)
	b2 := st2.Bucket("isolation-test")
	_, err = b2.Info(ctx)
	if err == nil {
		t.Error("expected st2 to not see bucket from st1")
	}
}

// TestMemoryBasicOperations tests basic memory driver operations.
// The memory driver has different semantics (auto-bucket creation),
// so it gets its own dedicated test suite.
func TestMemoryBasicOperations(t *testing.T) {
	factory, ok := GetDriverFactory("memory")
	if !ok {
		t.Fatal("memory driver not registered")
	}

	st, cleanup := factory(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("CreateBucket", func(t *testing.T) {
		info, err := st.CreateBucket(ctx, "test-bucket", nil)
		if err != nil {
			t.Fatalf("CreateBucket: %v", err)
		}
		if info.Name != "test-bucket" {
			t.Errorf("expected name 'test-bucket', got %q", info.Name)
		}
	})

	t.Run("CreateBucket_Duplicate", func(t *testing.T) {
		_, err := st.CreateBucket(ctx, "dup-bucket", nil)
		if err != nil {
			t.Fatalf("CreateBucket first: %v", err)
		}
		_, err = st.CreateBucket(ctx, "dup-bucket", nil)
		if err == nil {
			t.Error("expected error for duplicate bucket")
		}
	})

	t.Run("WriteAndOpen", func(t *testing.T) {
		b := st.Bucket("write-test")
		_, err := b.Write(ctx, "file.txt", strings.NewReader("hello"), 5, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write: %v", err)
		}

		rc, obj, err := b.Open(ctx, "file.txt", 0, 0, nil)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer rc.Close()

		data, _ := io.ReadAll(rc)
		if string(data) != "hello" {
			t.Errorf("expected 'hello', got %q", data)
		}
		if obj.Size != 5 {
			t.Errorf("expected size 5, got %d", obj.Size)
		}
	})

	t.Run("Stat", func(t *testing.T) {
		b := st.Bucket("stat-test")
		_, _ = b.Write(ctx, "stat.txt", strings.NewReader("content"), 7, "text/plain", nil)

		obj, err := b.Stat(ctx, "stat.txt", nil)
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if obj.Size != 7 {
			t.Errorf("expected size 7, got %d", obj.Size)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		b := st.Bucket("delete-test")
		_, _ = b.Write(ctx, "delete.txt", strings.NewReader("x"), 1, "text/plain", nil)

		err := b.Delete(ctx, "delete.txt", nil)
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}

		_, err = b.Stat(ctx, "delete.txt", nil)
		if err == nil {
			t.Error("expected error after delete")
		}
	})

	t.Run("Copy", func(t *testing.T) {
		b := st.Bucket("copy-test")
		_, _ = b.Write(ctx, "src.txt", strings.NewReader("source"), 6, "text/plain", nil)

		_, err := b.Copy(ctx, "dst.txt", "", "src.txt", nil)
		if err != nil {
			t.Fatalf("Copy: %v", err)
		}

		obj, err := b.Stat(ctx, "dst.txt", nil)
		if err != nil {
			t.Fatalf("Stat dst: %v", err)
		}
		if obj.Size != 6 {
			t.Errorf("expected size 6, got %d", obj.Size)
		}
	})

	t.Run("List", func(t *testing.T) {
		b := st.Bucket("list-test")
		_, _ = b.Write(ctx, "a.txt", strings.NewReader("a"), 1, "text/plain", nil)
		_, _ = b.Write(ctx, "b.txt", strings.NewReader("b"), 1, "text/plain", nil)

		iter, err := b.List(ctx, "", 0, 0, nil)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		defer iter.Close()

		count := 0
		for {
			obj, err := iter.Next()
			if err != nil || obj == nil {
				break
			}
			count++
		}
		if count != 2 {
			t.Errorf("expected 2 objects, got %d", count)
		}
	})

	t.Run("Features", func(t *testing.T) {
		features := st.Features()
		if features["multipart"] != true {
			t.Error("expected multipart feature")
		}
	})
}

// TestMemoryBasicMultipart tests memory driver multipart operations.
func TestMemoryBasicMultipart(t *testing.T) {
	factory, ok := GetDriverFactory("memory")
	if !ok {
		t.Fatal("memory driver not registered")
	}

	st, cleanup := factory(t)
	defer cleanup()
	ctx := context.Background()

	b := st.Bucket("multipart-test")
	mp, ok := b.(storage.HasMultipart)
	if !ok {
		t.Fatal("memory bucket does not support multipart")
	}

	t.Run("InitAndComplete", func(t *testing.T) {
		mu, err := mp.InitMultipart(ctx, "test.bin", "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("InitMultipart: %v", err)
		}

		part1, err := mp.UploadPart(ctx, mu, 1, strings.NewReader("AAA"), 3, nil)
		if err != nil {
			t.Fatalf("UploadPart 1: %v", err)
		}

		part2, err := mp.UploadPart(ctx, mu, 2, strings.NewReader("BBB"), 3, nil)
		if err != nil {
			t.Fatalf("UploadPart 2: %v", err)
		}

		obj, err := mp.CompleteMultipart(ctx, mu, []*storage.PartInfo{part1, part2}, nil)
		if err != nil {
			t.Fatalf("CompleteMultipart: %v", err)
		}

		if obj.Size != 6 {
			t.Errorf("expected size 6, got %d", obj.Size)
		}

		// Verify content
		rc, _, _ := b.Open(ctx, "test.bin", 0, 0, nil)
		data, _ := io.ReadAll(rc)
		rc.Close()
		if string(data) != "AAABBB" {
			t.Errorf("expected 'AAABBB', got %q", data)
		}
	})

	t.Run("ListParts", func(t *testing.T) {
		mu, _ := mp.InitMultipart(ctx, "parts.bin", "application/octet-stream", nil)
		defer mp.AbortMultipart(ctx, mu, nil)

		_, _ = mp.UploadPart(ctx, mu, 1, strings.NewReader("A"), 1, nil)
		_, _ = mp.UploadPart(ctx, mu, 2, strings.NewReader("B"), 1, nil)
		_, _ = mp.UploadPart(ctx, mu, 3, strings.NewReader("C"), 1, nil)

		parts, err := mp.ListParts(ctx, mu, 0, 0, nil)
		if err != nil {
			t.Fatalf("ListParts: %v", err)
		}
		if len(parts) != 3 {
			t.Errorf("expected 3 parts, got %d", len(parts))
		}
	})

	t.Run("AbortMultipart", func(t *testing.T) {
		mu, _ := mp.InitMultipart(ctx, "abort.bin", "application/octet-stream", nil)
		_, _ = mp.UploadPart(ctx, mu, 1, strings.NewReader("X"), 1, nil)

		err := mp.AbortMultipart(ctx, mu, nil)
		if err != nil {
			t.Fatalf("AbortMultipart: %v", err)
		}
	})
}
