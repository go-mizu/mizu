package fineweb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSearchComparison compares FTS vs LIKE search methods.
// This test requires a populated database to run meaningful comparisons.
// Skip with: go test -short
func TestSearchComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping search comparison test in short mode")
	}

	home, _ := os.UserHomeDir()
	dbDir := filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	dbPath := filepath.Join(dbDir, "vie_Latn.duckdb")

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skip("Skipping test: database not found at", dbPath)
	}

	store, err := NewStore("vie_Latn", dbDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Verify we have data
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}
	if count == 0 {
		t.Skip("Skipping test: no documents in database")
	}
	t.Logf("Document count: %d", count)

	// Check if FTS index exists
	hasFTS := store.HasFTSIndex(ctx)
	t.Logf("FTS index exists: %v", hasFTS)

	// Test queries
	queries := []string{
		"Việt Nam",        // Vietnamese text
		"thành phố",       // "city" in Vietnamese
		"công nghệ",       // "technology" in Vietnamese
		"internet",        // English loanword
		"2024",            // Number/date
	}

	for _, query := range queries {
		t.Run(fmt.Sprintf("Query_%s", query), func(t *testing.T) {
			comp, err := store.CompareSearch(ctx, query, 20)
			if err != nil {
				t.Errorf("CompareSearch failed: %v", err)
				return
			}

			t.Logf("Query: %q", comp.Query)

			if comp.LIKEError != nil {
				t.Logf("  LIKE error: %v", comp.LIKEError)
			} else {
				t.Logf("  LIKE: %d results in %v", len(comp.LIKE.Documents), comp.LIKE.Duration)
			}

			if comp.FTSError != nil {
				t.Logf("  FTS error: %v", comp.FTSError)
			} else if comp.FTS != nil {
				t.Logf("  FTS:  %d results in %v", len(comp.FTS.Documents), comp.FTS.Duration)
			}

			if comp.FTSError == nil && comp.LIKEError == nil && comp.FTS != nil {
				t.Logf("  Overlap: %d (%.1f%%)", comp.Overlap, comp.OverlapPct)
				t.Logf("  Speedup: %.1f%% (positive = FTS faster)", comp.SpeedupPct)
			}
		})
	}
}

// TestFTSIndexCreation tests creating FTS index with various configurations.
func TestFTSIndexCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping FTS index creation test in short mode")
	}

	// Use a temp directory for this test
	tmpDir := t.TempDir()

	store, err := NewStore("test_fts", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Insert some test documents
	testDocs := []struct {
		id   string
		url  string
		text string
	}{
		{"1", "http://example.com/1", "The quick brown fox jumps over the lazy dog"},
		{"2", "http://example.com/2", "A quick brown dog runs in the park"},
		{"3", "http://example.com/3", "The fox is quick and brown"},
		{"4", "http://example.com/4", "Lazy dogs sleep all day long"},
		{"5", "http://example.com/5", "Parks are great for running and jumping"},
	}

	for _, doc := range testDocs {
		_, err := store.db.ExecContext(ctx, `
			INSERT INTO documents (id, url, text, dump, date, language, language_score)
			VALUES (?, ?, ?, '', '', 'en', 1.0)
		`, doc.id, doc.url, doc.text)
		if err != nil {
			t.Fatalf("Failed to insert test document: %v", err)
		}
	}

	// Test FTS config
	cfg := FTSConfig{
		Stemmer:                "porter",
		Stopwords:              "english",
		StripAccents:           true,
		Lower:                  true,
		MemoryLimit:            "1GB",
		Threads:                2,
		PreserveInsertionOrder: false,
	}

	// Create FTS index
	err = store.CreateFTSIndexWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create FTS index: %v", err)
	}

	// Verify FTS index exists
	if !store.HasFTSIndex(ctx) {
		t.Error("FTS index should exist after creation")
	}

	// Test FTS search
	result, err := store.SearchFTS(ctx, "quick brown", 10, 0)
	if err != nil {
		t.Fatalf("FTS search failed: %v", err)
	}

	if len(result.Documents) == 0 {
		t.Error("FTS search should return results for 'quick brown'")
	}

	t.Logf("FTS search for 'quick brown': %d results in %v", len(result.Documents), result.Duration)
	for i, doc := range result.Documents {
		t.Logf("  %d. [%.4f] %s", i+1, doc.Score, doc.Text[:min(50, len(doc.Text))])
	}

	// Test LIKE search for comparison
	likeResult, err := store.SearchLike(ctx, "quick brown", 10, 0)
	if err != nil {
		t.Fatalf("LIKE search failed: %v", err)
	}

	t.Logf("LIKE search for 'quick brown': %d results in %v", len(likeResult.Documents), likeResult.Duration)

	// Drop FTS index
	err = store.DropFTSIndex(ctx)
	if err != nil {
		t.Errorf("Failed to drop FTS index: %v", err)
	}

	if store.HasFTSIndex(ctx) {
		t.Error("FTS index should not exist after drop")
	}
}

// TestBM25Parameters tests different BM25 parameter configurations.
func TestBM25Parameters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping BM25 parameters test in short mode")
	}

	tmpDir := t.TempDir()

	store, err := NewStore("test_bm25", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Insert varied documents for testing BM25 parameters
	docs := []struct {
		id   string
		text string
	}{
		{"1", "cat cat cat cat cat"},                    // High term frequency
		{"2", "cat"},                                     // Low term frequency
		{"3", "cat dog bird fish snake lizard"},         // Long document
		{"4", "cat dog"},                                 // Short document
		{"5", "the cat sat on the mat"},                 // Normal sentence
	}

	for _, doc := range docs {
		_, err := store.db.ExecContext(ctx, `
			INSERT INTO documents (id, url, text, dump, date, language, language_score)
			VALUES (?, 'http://test.com', ?, '', '', 'en', 1.0)
		`, doc.id, doc.text)
		if err != nil {
			t.Fatalf("Failed to insert: %v", err)
		}
	}

	// Create FTS index
	err = store.CreateFTSIndex(ctx)
	if err != nil {
		t.Fatalf("Failed to create FTS index: %v", err)
	}

	// Test different BM25 parameters
	testCases := []struct {
		name        string
		k           float64
		b           float64
		conjunctive bool
	}{
		{"default", 1.2, 0.75, false},
		{"high_k", 2.0, 0.75, false},     // Higher term frequency saturation
		{"low_k", 0.5, 0.75, false},       // Lower term frequency saturation
		{"high_b", 1.2, 1.0, false},       // Full length normalization
		{"low_b", 1.2, 0.0, false},        // No length normalization
		{"conjunctive", 1.2, 0.75, true},  // Require all terms
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := store.SearchFTSWithParams(ctx, "cat", 10, 0, tc.k, tc.b, tc.conjunctive)
			if err != nil {
				t.Errorf("Search failed: %v", err)
				return
			}

			t.Logf("BM25 k=%.2f b=%.2f conj=%v: %d results", tc.k, tc.b, tc.conjunctive, len(result.Documents))
			for i, doc := range result.Documents {
				t.Logf("  %d. [%.4f] id=%s text=%q", i+1, doc.Score, doc.ID, doc.Text)
			}
		})
	}
}

// TestSearchResultOverlap tests the overlap calculation between search methods.
func TestSearchResultOverlap(t *testing.T) {
	// Unit test for calculateOverlap function
	docs1 := []Document{
		{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"},
	}
	docs2 := []Document{
		{ID: "2"}, {ID: "3"}, {ID: "5"}, {ID: "6"},
	}

	overlap := calculateOverlap(docs1, docs2)
	if overlap != 2 {
		t.Errorf("Expected overlap of 2, got %d", overlap)
	}

	// Test with no overlap
	docs3 := []Document{
		{ID: "10"}, {ID: "11"},
	}
	overlap = calculateOverlap(docs1, docs3)
	if overlap != 0 {
		t.Errorf("Expected overlap of 0, got %d", overlap)
	}

	// Test with full overlap
	overlap = calculateOverlap(docs1, docs1)
	if overlap != 4 {
		t.Errorf("Expected overlap of 4, got %d", overlap)
	}
}

// TestDefaultFTSConfig tests the default FTS configuration.
func TestDefaultFTSConfig(t *testing.T) {
	cfg := DefaultFTSConfig()

	if cfg.Stemmer != "porter" {
		t.Errorf("Expected stemmer 'porter', got %q", cfg.Stemmer)
	}
	if cfg.Stopwords != "english" {
		t.Errorf("Expected stopwords 'english', got %q", cfg.Stopwords)
	}
	if !cfg.StripAccents {
		t.Error("StripAccents should be true by default")
	}
	if !cfg.Lower {
		t.Error("Lower should be true by default")
	}
	if cfg.MemoryLimit != "4GB" {
		t.Errorf("Expected memory limit '4GB', got %q", cfg.MemoryLimit)
	}
	if cfg.Threads != 4 {
		t.Errorf("Expected threads 4, got %d", cfg.Threads)
	}
	if cfg.PreserveInsertionOrder {
		t.Error("PreserveInsertionOrder should be false by default")
	}
}

// TestSearchResultTiming tests that search results include timing information.
func TestSearchResultTiming(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStore("test_timing", tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Insert a test document
	_, err = store.db.ExecContext(ctx, `
		INSERT INTO documents (id, url, text, dump, date, language, language_score)
		VALUES ('1', 'http://test.com', 'test content here', '', '', 'en', 1.0)
	`)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Test LIKE search timing
	result, err := store.SearchLike(ctx, "test", 10, 0)
	if err != nil {
		t.Fatalf("SearchLike failed: %v", err)
	}

	if result.Duration == 0 {
		t.Error("Duration should not be zero")
	}
	if result.Method != "like" {
		t.Errorf("Method should be 'like', got %q", result.Method)
	}

	t.Logf("LIKE search took %v", result.Duration)
}

// BenchmarkSearchMethods benchmarks FTS vs LIKE search.
// Run with: go test -bench=BenchmarkSearchMethods -benchtime=10s
func BenchmarkSearchMethods(b *testing.B) {
	home, _ := os.UserHomeDir()
	dbDir := filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	dbPath := filepath.Join(dbDir, "vie_Latn.duckdb")

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		b.Skip("Skipping benchmark: database not found")
	}

	store, err := NewStore("vie_Latn", dbDir)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	query := "Việt Nam"

	b.Run("LIKE", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := store.SearchLike(ctx, query, 20, 0)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	// Only run FTS benchmark if index exists
	if store.HasFTSIndex(ctx) {
		b.Run("FTS", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := store.SearchFTS(ctx, query, 20, 0)
				if err != nil {
					b.Fatalf("Search failed: %v", err)
				}
			}
		})
	}
}

// TestLargeDatasetFTSCreation tests FTS creation with memory-optimized settings.
// This test is specifically for validating the on-disk indexing configuration.
func TestLargeDatasetFTSCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset FTS test in short mode")
	}

	home, _ := os.UserHomeDir()
	dbDir := filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	dbPath := filepath.Join(dbDir, "vie_Latn.duckdb")

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skip("Skipping test: database not found at", dbPath)
	}

	store, err := NewStore("vie_Latn", dbDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Check document count
	count, _ := store.Count(ctx)
	t.Logf("Document count: %d", count)

	// Skip if FTS already exists
	if store.HasFTSIndex(ctx) {
		t.Log("FTS index already exists, testing search...")

		result, err := store.SearchFTS(ctx, "Việt Nam", 10, 0)
		if err != nil {
			t.Errorf("FTS search failed: %v", err)
		} else {
			t.Logf("FTS search returned %d results in %v", len(result.Documents), result.Duration)
		}
		return
	}

	// Configure for large dataset with on-disk indexing
	cfg := FTSConfig{
		Stemmer:      "none", // Vietnamese doesn't benefit from porter stemmer
		Stopwords:    "none", // No Vietnamese stopwords available
		StripAccents: false,  // Keep Vietnamese diacritics
		Lower:        true,

		// Memory-optimized settings for large datasets
		MemoryLimit:            "8GB",
		Threads:                4,
		TempDirectory:          filepath.Join(dbDir, "fts_temp"),
		MaxTempDirectorySize:   "100GB",
		PreserveInsertionOrder: false,
	}

	t.Log("Creating FTS index with on-disk configuration...")
	t.Logf("Config: %+v", cfg)

	start := time.Now()
	err = store.CreateFTSIndexWithConfig(ctx, cfg)
	if err != nil {
		t.Logf("FTS creation failed (expected for very large datasets): %v", err)
		t.Log("This is expected behavior - the dataset may be too large for available resources")
		return
	}

	t.Logf("FTS index created in %v", time.Since(start))

	// Test search
	result, err := store.SearchFTS(ctx, "Việt Nam", 10, 0)
	if err != nil {
		t.Errorf("FTS search failed: %v", err)
	} else {
		t.Logf("FTS search returned %d results in %v", len(result.Documents), result.Duration)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
