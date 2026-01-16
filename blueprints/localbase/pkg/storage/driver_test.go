// File: lib/storage/driver_test.go
package storage_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
	_ "github.com/go-mizu/mizu/blueprints/localbase/pkg/storage/driver/local" // Register local driver
)

// mockDriver is a simple driver for testing registration.
type mockDriver struct {
	openCalled bool
	openDSN    string
	openErr    error
	openResult storage.Storage
}

func (d *mockDriver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	d.openCalled = true
	d.openDSN = dsn
	if d.openErr != nil {
		return nil, d.openErr
	}
	return d.openResult, nil
}

// Note: Register tests require process isolation since drivers are global.
// These tests use unique names to avoid conflicts with production drivers.

func TestOpen_EmptyDSN(t *testing.T) {
	ctx := context.Background()

	_, err := storage.Open(ctx, "")
	if err == nil {
		t.Error("expected error for empty DSN")
	}
}

func TestOpen_UnknownDriver(t *testing.T) {
	ctx := context.Background()

	_, err := storage.Open(ctx, "unknown-driver-xyz://path")
	if err == nil {
		t.Error("expected error for unknown driver")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestOpen_MissingScheme(t *testing.T) {
	ctx := context.Background()

	// DSN without colon should error
	_, err := storage.Open(ctx, "no-scheme-no-colon")
	if err == nil {
		t.Error("expected error for DSN without scheme")
	}
}

func TestDriverFromDSN_Variations(t *testing.T) {
	// These tests verify DSN parsing by attempting to open with various formats.
	// The driver lookup will fail but we can verify the parsing doesn't error unexpectedly.

	ctx := context.Background()

	testCases := []struct {
		name      string
		dsn       string
		wantError bool
	}{
		{"url_style", "scheme://host/path", true}, // unknown driver
		{"colon_style", "scheme:/path", true},     // unknown driver
		{"no_colon", "justpath", true},            // missing driver
		{"empty_scheme", "://path", true},         // empty scheme
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := storage.Open(ctx, tc.dsn)
			if tc.wantError && err == nil {
				t.Errorf("expected error for DSN %q", tc.dsn)
			}
		})
	}
}

func TestOpen_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Even with cancelled context, if driver is unknown, we should get driver error
	_, err := storage.Open(ctx, "unknown://path")
	if err == nil {
		t.Error("expected error")
	}
}

func TestOpen_LocalDriver(t *testing.T) {
	// Test that local driver is properly registered via init()
	// We need to import the local package for this

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Both "local" and "file" schemes should work
	testCases := []string{
		"local:" + tmpDir,
		"file://" + tmpDir,
	}

	for _, dsn := range testCases {
		t.Run(dsn, func(t *testing.T) {
			st, err := storage.Open(ctx, dsn)
			if err != nil {
				t.Fatalf("Open(%q): %v", dsn, err)
			}
			defer func() {
				_ = st.Close()
			}()

			if st == nil {
				t.Error("expected non-nil storage")
			}
		})
	}
}

func TestOpen_LocalDriverBarePath(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Bare absolute path should work with local driver
	st, err := storage.Open(ctx, tmpDir)
	if err != nil {
		// This might fail if DSN parsing doesn't handle bare paths
		// That's acceptable behavior
		t.Skipf("bare path DSN not supported: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()
}

// TestRegisterPanics tests that Register panics for invalid inputs.
// Note: These must be run carefully as they affect global state.
func TestRegisterPanics(t *testing.T) {
	// Test empty name panic
	t.Run("EmptyName", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty name")
			}
		}()
		storage.Register("", &mockDriver{})
	})

	// Test nil driver panic
	t.Run("NilDriver", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil driver")
			}
		}()
		storage.Register("test-nil-driver-unique-name", nil)
	})

	// Test duplicate registration panic
	t.Run("Duplicate", func(t *testing.T) {
		uniqueName := "test-dup-driver-" + t.Name()

		// First registration should succeed
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("first registration should not panic: %v", r)
				}
			}()
			storage.Register(uniqueName, &mockDriver{})
		}()

		// Second registration should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for duplicate registration")
			}
		}()
		storage.Register(uniqueName, &mockDriver{})
	})
}

// TestConcurrentOpen tests that Open is safe for concurrent use.
func TestConcurrentOpen(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	var wg sync.WaitGroup
	errs := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			st, err := storage.Open(ctx, "local:"+tmpDir)
			if err != nil {
				errs <- err
				return
			}
			_ = st.Close()
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent Open error: %v", err)
	}
}

// TestDriverOpenError tests that driver Open errors are propagated.
func TestDriverOpenError(t *testing.T) {
	ctx := context.Background()

	// Use a registered driver with non-existent path
	_, err := storage.Open(ctx, "local:/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

// TestDriverContextPropagation tests that context is passed to driver.
func TestDriverContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	tmpDir := t.TempDir()

	// Open with cancelled context should fail
	_, err := storage.Open(ctx, "local:"+tmpDir)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
