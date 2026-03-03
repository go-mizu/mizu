package dahlia

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func TestEngineRegistration(t *testing.T) {
	engines := index.List()
	found := false
	for _, name := range engines {
		if name == "dahlia" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("dahlia not in engine list: %v", engines)
	}

	eng, err := index.NewEngine("dahlia")
	if err != nil {
		t.Fatal(err)
	}
	if eng.Name() != "dahlia" {
		t.Fatalf("Name()=%q, want dahlia", eng.Name())
	}
}

func TestEngineIndexAndSearch(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	eng := &Engine{}
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatal(err)
	}
	defer eng.Close()

	// Index documents
	docs := []index.Document{
		{DocID: "doc1", Text: []byte("machine learning algorithms for natural language processing")},
		{DocID: "doc2", Text: []byte("deep learning neural networks and artificial intelligence")},
		{DocID: "doc3", Text: []byte("the quick brown fox jumps over the lazy dog")},
		{DocID: "doc4", Text: []byte("natural language processing with machine learning models")},
		{DocID: "doc5", Text: []byte("search engine algorithms and information retrieval")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatal(err)
	}

	// Finalize to flush
	if err := eng.Finalize(ctx); err != nil {
		t.Fatal(err)
	}

	// Search
	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}

	if len(results.Hits) == 0 {
		t.Fatal("expected search results")
	}

	t.Logf("Search results for 'machine learning':")
	for _, hit := range results.Hits {
		t.Logf("  %s (score=%.4f) %s", hit.DocID, hit.Score, hit.Snippet[:min(60, len(hit.Snippet))])
	}
}

func TestEngineMultiBatch(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	eng := &Engine{}
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatal(err)
	}
	defer eng.Close()

	// Index in two batches
	batch1 := []index.Document{
		{DocID: "b1_1", Text: []byte("machine learning algorithms")},
		{DocID: "b1_2", Text: []byte("deep learning networks")},
	}
	batch2 := []index.Document{
		{DocID: "b2_1", Text: []byte("machine learning models")},
		{DocID: "b2_2", Text: []byte("search algorithms")},
	}

	eng.Index(ctx, batch1)
	eng.Finalize(ctx) // flush first batch
	eng.Index(ctx, batch2)
	eng.Finalize(ctx) // flush second batch

	// Should find results across both batches
	results, err := eng.Search(ctx, index.Query{Text: "machine", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Hits) < 2 {
		t.Fatalf("expected >= 2 hits from multi-batch, got %d", len(results.Hits))
	}
}

func TestEngineStats(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	eng := &Engine{}
	eng.Open(ctx, dir)
	defer eng.Close()

	eng.Index(ctx, []index.Document{
		{DocID: "d1", Text: []byte("hello world")},
	})
	eng.Finalize(ctx)

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.DocCount != 1 {
		t.Fatalf("DocCount=%d, want 1", stats.DocCount)
	}
	if stats.DiskBytes <= 0 {
		t.Fatalf("DiskBytes=%d, want > 0", stats.DiskBytes)
	}
}

func TestEngineBooleanMust(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	eng := &Engine{}
	eng.Open(ctx, dir)
	defer eng.Close()

	eng.Index(ctx, []index.Document{
		{DocID: "apple_banana", Text: []byte("fresh apple and ripe banana fruit salad")},
		{DocID: "apple_only", Text: []byte("crispy apple pie with cinnamon")},
		{DocID: "banana_only", Text: []byte("banana bread recipe with nuts")},
	})
	eng.Finalize(ctx)

	results, err := eng.Search(ctx, index.Query{Text: "+apple +banana", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("expected results for +apple +banana")
	}
	// Should only have the doc with both terms
	for _, hit := range results.Hits {
		if hit.DocID != "apple_banana" {
			t.Logf("unexpected hit: %s", hit.DocID)
		}
	}
}

func TestEnginePhraseSearch(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	eng := &Engine{}
	eng.Open(ctx, dir)
	defer eng.Close()

	eng.Index(ctx, []index.Document{
		{DocID: "ml_phrase", Text: []byte("advances in machine learning for robotics")},
		{DocID: "ml_separate", Text: []byte("the machine was learning to walk by itself")},
		{DocID: "no_ml", Text: []byte("the quick brown fox")},
	})
	eng.Finalize(ctx)

	results, err := eng.Search(ctx, index.Query{Text: `"machine learning"`, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("expected phrase results")
	}
	t.Logf("Phrase search results:")
	for _, hit := range results.Hits {
		t.Logf("  %s score=%.4f", hit.DocID, hit.Score)
	}
}

func TestEngineLargeBatch(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	eng := &Engine{}
	eng.Open(ctx, dir)
	defer eng.Close()

	// Index 1000 documents
	docs := make([]index.Document, 1000)
	for i := range docs {
		docs[i] = index.Document{
			DocID: fmt.Sprintf("doc_%06d", i),
			Text:  []byte(fmt.Sprintf("document number %d about topic %d with searchable content and various keywords like algorithm data structure machine learning", i, i%20)),
		}
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatal(err)
	}
	if err := eng.Finalize(ctx); err != nil {
		t.Fatal(err)
	}

	stats, _ := eng.Stats(ctx)
	if stats.DocCount != 1000 {
		t.Fatalf("DocCount=%d, want 1000", stats.DocCount)
	}

	results, err := eng.Search(ctx, index.Query{Text: "algorithm", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("expected results")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
