package fineweb_test

import (
	"context"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"

	// Import drivers for testing
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/duckdb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/sqlite"
	// Note: Other drivers commented out as they require additional dependencies
	// _ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/bleve"
	// _ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/bluge"
	// _ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/porter"
)

// TestDriverRegistry tests driver registration and lookup.
func TestDriverRegistry(t *testing.T) {
	drivers := fineweb.List()
	if len(drivers) == 0 {
		t.Fatal("No drivers registered")
	}

	t.Logf("Registered drivers: %v", drivers)

	// Check that duckdb and sqlite are registered
	if !fineweb.IsRegistered("duckdb") {
		t.Error("DuckDB driver not registered")
	}
	if !fineweb.IsRegistered("sqlite") {
		t.Error("SQLite driver not registered")
	}

	// Check unknown driver
	if fineweb.IsRegistered("nonexistent") {
		t.Error("Nonexistent driver should not be registered")
	}
}

// TestDriverConfig tests driver configuration helpers.
func TestDriverConfig(t *testing.T) {
	cfg := fineweb.DriverConfig{
		DataDir:  "/test/data",
		Language: "en",
		Options: map[string]any{
			"string_opt": "value",
			"int_opt":    42,
			"float_opt":  3.14,
			"bool_opt":   true,
		},
	}

	// Test GetString
	if v := cfg.GetString("string_opt", ""); v != "value" {
		t.Errorf("GetString: expected 'value', got %q", v)
	}
	if v := cfg.GetString("missing", "default"); v != "default" {
		t.Errorf("GetString default: expected 'default', got %q", v)
	}

	// Test GetInt
	if v := cfg.GetInt("int_opt", 0); v != 42 {
		t.Errorf("GetInt: expected 42, got %d", v)
	}
	if v := cfg.GetInt("missing", 99); v != 99 {
		t.Errorf("GetInt default: expected 99, got %d", v)
	}

	// Test GetFloat64
	if v := cfg.GetFloat64("float_opt", 0); v != 3.14 {
		t.Errorf("GetFloat64: expected 3.14, got %f", v)
	}

	// Test GetBool
	if v := cfg.GetBool("bool_opt", false); !v {
		t.Error("GetBool: expected true")
	}

	// Test With
	cfg2 := cfg.With("new_opt", "new_value")
	if v := cfg2.GetString("new_opt", ""); v != "new_value" {
		t.Errorf("With: expected 'new_value', got %q", v)
	}
	// Original should be unchanged
	if v := cfg.GetString("new_opt", "missing"); v != "missing" {
		t.Error("With should not modify original")
	}
}

// TestEmbeddedDrivers tests basic operations on embedded drivers.
func TestEmbeddedDrivers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping driver tests in short mode")
	}

	tmpDir := t.TempDir()

	// Test each embedded driver
	embeddedDrivers := []string{"duckdb", "sqlite"}

	for _, driverName := range embeddedDrivers {
		if !fineweb.IsRegistered(driverName) {
			t.Logf("Skipping %s (not registered)", driverName)
			continue
		}

		t.Run(driverName, func(t *testing.T) {
			testDriver(t, driverName, tmpDir)
		})
	}
}

func testDriver(t *testing.T, driverName, baseDir string) {
	ctx := context.Background()

	cfg := fineweb.DriverConfig{
		DataDir:  filepath.Join(baseDir, driverName),
		Language: "test",
	}

	// Open driver
	driver, err := fineweb.Open(driverName, cfg)
	if err != nil {
		t.Fatalf("Failed to open driver: %v", err)
	}
	defer driver.Close()

	// Check name
	if driver.Name() != driverName {
		t.Errorf("Name: expected %q, got %q", driverName, driver.Name())
	}

	// Check interfaces
	indexer, hasIndexer := fineweb.AsIndexer(driver)
	stats, hasStats := fineweb.AsStats(driver)

	t.Logf("Driver %s: Indexer=%v Stats=%v", driverName, hasIndexer, hasStats)

	// Test indexing if supported
	if hasIndexer {
		testDocs := []fineweb.Document{
			{ID: "1", URL: "http://example.com/1", Text: "The quick brown fox jumps over the lazy dog"},
			{ID: "2", URL: "http://example.com/2", Text: "A quick brown dog runs in the park"},
			{ID: "3", URL: "http://example.com/3", Text: "Vietnamese text: Xin chào Việt Nam"},
			{ID: "4", URL: "http://example.com/4", Text: "Technology and innovation in 2024"},
			{ID: "5", URL: "http://example.com/5", Text: "Parks are great for running"},
		}

		docs := docIterator(testDocs)
		err = indexer.Import(ctx, docs, func(imported, total int64) {
			t.Logf("  Imported %d documents", imported)
		})
		if err != nil {
			t.Fatalf("Import failed: %v", err)
		}
	}

	// Test count if supported
	if hasStats {
		count, err := stats.Count(ctx)
		if err != nil {
			t.Errorf("Count failed: %v", err)
		} else {
			t.Logf("Document count: %d", count)
			if hasIndexer && count != 5 {
				t.Errorf("Expected 5 documents, got %d", count)
			}
		}
	}

	// Test search
	result, err := driver.Search(ctx, "quick", 10, 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	t.Logf("Search results: %d documents in %v", len(result.Documents), result.Duration)

	if hasIndexer && len(result.Documents) == 0 {
		t.Error("Search should return results after indexing")
	}

	for i, doc := range result.Documents {
		t.Logf("  %d. [%.4f] %s: %s", i+1, doc.Score, doc.ID, truncate(doc.Text, 50))
	}

	// Test offset
	if len(result.Documents) > 1 {
		result2, err := driver.Search(ctx, "quick", 10, 1)
		if err != nil {
			t.Errorf("Search with offset failed: %v", err)
		} else if len(result2.Documents) >= len(result.Documents) {
			t.Error("Offset should reduce results")
		}
	}

	// Test Vietnamese search
	vietResult, err := driver.Search(ctx, "Việt Nam", 10, 0)
	if err != nil {
		t.Errorf("Vietnamese search failed: %v", err)
	} else {
		t.Logf("Vietnamese search: %d results", len(vietResult.Documents))
	}
}

// docIterator creates an iterator from a slice of documents.
func docIterator(docs []fineweb.Document) iter.Seq2[fineweb.Document, error] {
	return func(yield func(fineweb.Document, error) bool) {
		for _, doc := range docs {
			if !yield(doc, nil) {
				return
			}
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TestOpenUnknownDriver tests opening an unknown driver.
func TestOpenUnknownDriver(t *testing.T) {
	_, err := fineweb.Open("nonexistent", fineweb.DriverConfig{})
	if err == nil {
		t.Error("Opening unknown driver should fail")
	}
}

// TestMustOpen tests panic on error.
func TestMustOpen(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustOpen should panic on error")
		}
	}()

	_ = fineweb.MustOpen("nonexistent", fineweb.DriverConfig{})
}

// TestDriverInfo tests getting driver metadata.
func TestDriverInfo(t *testing.T) {
	if !fineweb.IsRegistered("duckdb") {
		t.Skip("DuckDB not registered")
	}

	tmpDir := t.TempDir()
	driver, err := fineweb.Open("duckdb", fineweb.DriverConfig{DataDir: tmpDir})
	if err != nil {
		t.Fatalf("Failed to open driver: %v", err)
	}
	defer driver.Close()

	info := fineweb.GetDriverInfo(driver)
	if info == nil {
		t.Fatal("GetDriverInfo returned nil")
	}

	t.Logf("Driver info: %+v", info)

	if info.Name != "duckdb" {
		t.Errorf("Expected name 'duckdb', got %q", info.Name)
	}
}

// TestParquetReader tests the parquet reader (if data exists).
func TestParquetReader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parquet test in short mode")
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	reader := fineweb.NewParquetReader(parquetDir)

	ctx := context.Background()

	// List files
	files, err := reader.ListParquetFiles()
	if err != nil {
		t.Fatalf("ListParquetFiles failed: %v", err)
	}
	t.Logf("Found %d parquet files", len(files))

	// Count documents (this can be slow for large files)
	count, err := reader.CountDocuments(ctx)
	if err != nil {
		t.Fatalf("CountDocuments failed: %v", err)
	}
	t.Logf("Total documents: %d", count)

	// Read first few documents
	docs := reader.ReadAll(ctx)
	var read int
	for doc, err := range docs {
		if err != nil {
			t.Fatalf("Error reading document: %v", err)
		}
		if read == 0 {
			t.Logf("First document: ID=%s URL=%s Text=%s...", doc.ID, doc.URL, truncate(doc.Text, 100))
		}
		read++
		if read >= 10 {
			break
		}
	}
	t.Logf("Read %d documents", read)
}

// BenchmarkDriverSearch benchmarks search across drivers.
func BenchmarkDriverSearch(b *testing.B) {
	tmpDir := b.TempDir()

	drivers := []string{"duckdb", "sqlite"}
	queries := []string{"quick", "brown", "fox"}

	for _, driverName := range drivers {
		if !fineweb.IsRegistered(driverName) {
			continue
		}

		b.Run(driverName, func(b *testing.B) {
			ctx := context.Background()

			driver, err := fineweb.Open(driverName, fineweb.DriverConfig{
				DataDir:  filepath.Join(tmpDir, driverName),
				Language: "bench",
			})
			if err != nil {
				b.Fatalf("Failed to open driver: %v", err)
			}
			defer driver.Close()

			// Index test data
			if indexer, ok := fineweb.AsIndexer(driver); ok {
				testDocs := generateTestDocs(1000)
				docs := docIterator(testDocs)
				_ = indexer.Import(ctx, docs, nil)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				query := queries[i%len(queries)]
				_, err := driver.Search(ctx, query, 20, 0)
				if err != nil {
					b.Fatalf("Search failed: %v", err)
				}
			}
		})
	}
}

func generateTestDocs(n int) []fineweb.Document {
	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"A quick brown dog runs in the park",
		"Technology and innovation drive progress",
		"Vietnamese cuisine is delicious and diverse",
		"Parks are great for running and exercise",
		"Software engineering requires problem solving",
		"Data science combines statistics and programming",
		"Machine learning enables intelligent systems",
		"Cloud computing provides scalable infrastructure",
		"Cybersecurity protects digital assets",
	}

	docs := make([]fineweb.Document, n)
	for i := 0; i < n; i++ {
		docs[i] = fineweb.Document{
			ID:   fmt.Sprintf("doc-%d", i),
			URL:  fmt.Sprintf("http://example.com/%d", i),
			Text: texts[i%len(texts)],
		}
	}
	return docs
}
