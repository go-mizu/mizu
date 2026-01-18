package storetest

import (
	"context"
	"os"
	"testing"

	"github.com/go-mizu/mizu/blueprints/lingo/store"
	"github.com/go-mizu/mizu/blueprints/lingo/store/postgres"
	"github.com/go-mizu/mizu/blueprints/lingo/store/sqlite"
)

// testStores holds the test stores for both drivers
type testStores struct {
	stores map[string]store.Store
}

// setupTestStores creates test stores for both PostgreSQL and SQLite
func setupTestStores(t *testing.T) *testStores {
	t.Helper()

	ctx := context.Background()
	stores := make(map[string]store.Store)

	// Setup SQLite (always available, in-memory)
	sqliteStore, err := sqlite.New(ctx, ":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	if err := sqliteStore.Ensure(ctx); err != nil {
		t.Fatalf("failed to ensure sqlite schema: %v", err)
	}
	stores["sqlite"] = sqliteStore

	// Setup PostgreSQL (only if TEST_POSTGRES_URL is set)
	if pgURL := os.Getenv("TEST_POSTGRES_URL"); pgURL != "" {
		pgStore, err := postgres.New(ctx, pgURL)
		if err != nil {
			t.Logf("PostgreSQL not available: %v", err)
		} else {
			if err := pgStore.CreateExtensions(ctx); err != nil {
				t.Logf("failed to create postgres extensions: %v", err)
			}
			if err := pgStore.Ensure(ctx); err != nil {
				t.Fatalf("failed to ensure postgres schema: %v", err)
			}
			stores["postgres"] = pgStore
		}
	}

	t.Cleanup(func() {
		for _, s := range stores {
			s.Close()
		}
	})

	return &testStores{stores: stores}
}

// forEachStore runs a test for each available store
func (ts *testStores) forEachStore(t *testing.T, testFunc func(t *testing.T, s store.Store)) {
	for name, s := range ts.stores {
		t.Run(name, func(t *testing.T) {
			testFunc(t, s)
		})
	}
}

// seedTestData adds consistent test data to a store
func seedTestData(t *testing.T, s store.Store) {
	t.Helper()
	ctx := context.Background()

	// Seed in order: languages, courses, achievements, leagues, users
	if seeder, ok := s.(interface{ SeedLanguages(context.Context) error }); ok {
		if err := seeder.SeedLanguages(ctx); err != nil {
			t.Fatalf("failed to seed languages: %v", err)
		}
	}

	if seeder, ok := s.(interface{ SeedCourses(context.Context) error }); ok {
		if err := seeder.SeedCourses(ctx); err != nil {
			t.Fatalf("failed to seed courses: %v", err)
		}
	}

	if seeder, ok := s.(interface{ SeedAchievements(context.Context) error }); ok {
		if err := seeder.SeedAchievements(ctx); err != nil {
			t.Fatalf("failed to seed achievements: %v", err)
		}
	}

	if seeder, ok := s.(interface{ SeedLeagues(context.Context) error }); ok {
		if err := seeder.SeedLeagues(ctx); err != nil {
			t.Fatalf("failed to seed leagues: %v", err)
		}
	}
}

// assertNoError verifies no error occurred
func assertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", msg, err)
	}
}

// assertError verifies an error occurred
func assertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error but got none", msg)
	}
}

// assertEqual verifies two values are equal
func assertEqual[T comparable](t *testing.T, expected, actual T, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: expected %v but got %v", msg, expected, actual)
	}
}

// assertNotNil verifies a value is not nil
func assertNotNil(t *testing.T, v any, msg string) {
	t.Helper()
	if v == nil {
		t.Fatalf("%s: expected non-nil value", msg)
	}
}
