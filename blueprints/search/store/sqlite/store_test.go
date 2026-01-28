package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// testStore creates a temporary store for testing.
func testStore(t *testing.T) (*Store, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "search-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	s, err := New(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := s.Ensure(ctx); err != nil {
		s.Close()
		os.RemoveAll(dir)
		t.Fatalf("failed to ensure schema: %v", err)
	}

	cleanup := func() {
		s.Close()
		os.RemoveAll(dir)
	}

	return s, cleanup
}

func TestNew(t *testing.T) {
	dir, err := os.MkdirTemp("", "search-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	if s.db == nil {
		t.Error("expected db to be set")
	}
	if s.search == nil {
		t.Error("expected search store to be set")
	}
	if s.index == nil {
		t.Error("expected index store to be set")
	}
	if s.suggest == nil {
		t.Error("expected suggest store to be set")
	}
	if s.knowledge == nil {
		t.Error("expected knowledge store to be set")
	}
	if s.history == nil {
		t.Error("expected history store to be set")
	}
	if s.preference == nil {
		t.Error("expected preference store to be set")
	}
}

func TestNew_CreateDirectory(t *testing.T) {
	dir, err := os.MkdirTemp("", "search-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Path with non-existent subdirectory
	dbPath := filepath.Join(dir, "subdir", "another", "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}
}

func TestStore_Ensure(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Ensure is idempotent
	if err := s.Ensure(ctx); err != nil {
		t.Errorf("second Ensure() error = %v", err)
	}
}

func TestStore_CreateExtensions(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Should be no-op for SQLite
	if err := s.CreateExtensions(ctx); err != nil {
		t.Errorf("CreateExtensions() error = %v", err)
	}
}

func TestStore_FeatureStores(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	if s.Search() == nil {
		t.Error("Search() returned nil")
	}
	if s.Index() == nil {
		t.Error("Index() returned nil")
	}
	if s.Suggest() == nil {
		t.Error("Suggest() returned nil")
	}
	if s.Knowledge() == nil {
		t.Error("Knowledge() returned nil")
	}
	if s.History() == nil {
		t.Error("History() returned nil")
	}
	if s.Preference() == nil {
		t.Error("Preference() returned nil")
	}
}

func TestStore_SeedDocuments(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	if err := s.SeedDocuments(ctx); err != nil {
		t.Errorf("SeedDocuments() error = %v", err)
	}

	// Verify documents were created
	stats, err := s.Index().GetIndexStats(ctx)
	if err != nil {
		t.Fatalf("GetIndexStats() error = %v", err)
	}

	if stats.TotalDocuments == 0 {
		t.Error("expected documents to be seeded")
	}
}

func TestStore_SeedKnowledge(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	if err := s.SeedKnowledge(ctx); err != nil {
		t.Errorf("SeedKnowledge() error = %v", err)
	}

	// Verify entities were created
	entities, err := s.Knowledge().ListEntities(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("ListEntities() error = %v", err)
	}

	if len(entities) == 0 {
		t.Error("expected entities to be seeded")
	}
}
