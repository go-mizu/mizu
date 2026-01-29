package fts_highthroughput_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb/drivers/fts_highthroughput"
)

// TestSearchCorrectness verifies that fts_highthroughput returns correct results.
func TestSearchCorrectness(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	home, _ := os.UserHomeDir()
	parquetDir := filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")

	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		t.Skipf("Parquet directory not found: %s", parquetDir)
	}

	tmpDir := t.TempDir()
	ctx := context.Background()

	// Open driver
	driver, err := fineweb.Open("fts_highthroughput", fineweb.DriverConfig{
		DataDir:  tmpDir,
		Language: "vie_Latn",
	})
	if err != nil {
		t.Fatalf("Failed to open driver: %v", err)
	}
	defer driver.Close()

	// Index some documents
	indexer, ok := fineweb.AsIndexer(driver)
	if !ok {
		t.Fatal("Driver does not support indexing")
	}

	reader := fineweb.NewParquetReader(parquetDir).WithBatchSize(1000)

	// Index first 10k docs for quick test
	var indexed int64
	for batch, err := range reader.ReadTextsOnlyParallel(ctx, 4) {
		if err != nil {
			t.Fatal(err)
		}

		// Convert to docs iterator
		docs := make(chan fineweb.Document, len(batch))
		for _, doc := range batch {
			docs <- fineweb.Document{ID: doc.ID, Text: doc.Text}
		}
		close(docs)

		docsIter := func(yield func(fineweb.Document, error) bool) {
			for doc := range docs {
				if !yield(doc, nil) {
					return
				}
			}
		}

		if err := indexer.Import(ctx, docsIter, nil); err != nil {
			t.Fatal(err)
		}

		indexed += int64(len(batch))
		if indexed >= 10000 {
			break
		}
	}
	t.Logf("Indexed %d documents", indexed)

	// Get doc count
	stats, ok := fineweb.AsStats(driver)
	if ok {
		count, _ := stats.Count(ctx)
		t.Logf("Index reports %d documents", count)
	}

	// Test queries - these should return results if the search works correctly
	testQueries := []string{
		"vietnam",
		"hanoi",
		"saigon",
		"the",
		"and",
		"của",      // Vietnamese word "of"
		"là",       // Vietnamese word "is"
		"trong",    // Vietnamese word "in"
		"việt nam", // Vietnam in Vietnamese
	}

	t.Log("Testing search queries...")
	foundAny := false
	for _, query := range testQueries {
		result, err := driver.Search(ctx, query, 10, 0)
		if err != nil {
			t.Errorf("Search error for %q: %v", query, err)
			continue
		}

		if len(result.Documents) > 0 {
			foundAny = true
			t.Logf("Query %q: %d results, top score %.4f, took %v",
				query, len(result.Documents), result.Documents[0].Score, result.Duration)

			// Verify scores are reasonable
			for i, doc := range result.Documents {
				if doc.Score <= 0 {
					t.Errorf("Query %q: doc %d has non-positive score %f", query, i, doc.Score)
				}
				if i > 0 && doc.Score > result.Documents[i-1].Score {
					t.Errorf("Query %q: results not sorted by score descending", query)
				}
			}
		} else {
			t.Logf("Query %q: 0 results (took %v)", query, result.Duration)
		}
	}

	if !foundAny {
		t.Error("CRITICAL: No search query returned any results! Search may be broken.")
	}
}

// TestSearchWithKnownContent indexes specific content and verifies search finds it.
func TestSearchWithKnownContent(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Open driver
	driver, err := fineweb.Open("fts_highthroughput", fineweb.DriverConfig{
		DataDir:  tmpDir,
		Language: "test",
	})
	if err != nil {
		t.Fatalf("Failed to open driver: %v", err)
	}
	defer driver.Close()

	indexer, ok := fineweb.AsIndexer(driver)
	if !ok {
		t.Fatal("Driver does not support indexing")
	}

	// Create test documents with known content
	testDocs := []fineweb.Document{
		{ID: "doc1", Text: "The quick brown fox jumps over the lazy dog"},
		{ID: "doc2", Text: "Python is a programming language that is easy to learn"},
		{ID: "doc3", Text: "Machine learning and artificial intelligence are transforming technology"},
		{ID: "doc4", Text: "The fox is quick and brown while the dog is lazy"},
		{ID: "doc5", Text: "Go programming language is fast and efficient"},
	}

	// Index documents
	docsIter := func(yield func(fineweb.Document, error) bool) {
		for _, doc := range testDocs {
			if !yield(doc, nil) {
				return
			}
		}
	}

	if err := indexer.Import(ctx, docsIter, nil); err != nil {
		t.Fatal(err)
	}

	// Get doc count
	stats, ok := fineweb.AsStats(driver)
	if ok {
		count, _ := stats.Count(ctx)
		t.Logf("Index reports %d documents", count)
		if count != 5 {
			t.Errorf("Expected 5 documents, got %d", count)
		}
	}

	// Test searches that should return results
	tests := []struct {
		query       string
		expectDocs  bool
		expectFirst string // Expected doc ID for first result (if known)
	}{
		{"fox", true, ""},         // Should find doc1 and doc4
		{"quick brown", true, ""}, // Should find doc1 and doc4
		{"programming", true, ""}, // Should find doc2 and doc5
		{"python", true, "doc2"},  // Should find doc2 first
		{"machine learning", true, "doc3"},
		{"lazy dog", true, ""},          // Should find doc1 and doc4
		{"nonexistent xyz123", false, ""}, // Should not find anything
	}

	for _, tc := range tests {
		result, err := driver.Search(ctx, tc.query, 10, 0)
		if err != nil {
			t.Errorf("Search error for %q: %v", tc.query, err)
			continue
		}

		if tc.expectDocs && len(result.Documents) == 0 {
			t.Errorf("Query %q: expected results but got none", tc.query)
		} else if !tc.expectDocs && len(result.Documents) > 0 {
			t.Errorf("Query %q: expected no results but got %d", tc.query, len(result.Documents))
		}

		if tc.expectFirst != "" && len(result.Documents) > 0 {
			if result.Documents[0].ID != tc.expectFirst {
				t.Logf("Query %q: expected first result %q, got %q (score: %f)",
					tc.query, tc.expectFirst, result.Documents[0].ID, result.Documents[0].Score)
			}
		}

		if len(result.Documents) > 0 {
			t.Logf("Query %q: %d results, top=%s (%.4f)",
				tc.query, len(result.Documents), result.Documents[0].ID, result.Documents[0].Score)
		} else {
			t.Logf("Query %q: 0 results", tc.query)
		}
	}
}
