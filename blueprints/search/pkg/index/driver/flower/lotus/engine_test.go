package lotus

import (
	"context"
	"os"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func TestEngine_IndexAndSearch(t *testing.T) {
	dir := t.TempDir()
	e := &Engine{}
	ctx := context.Background()

	if err := e.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer e.Close()

	// Index some documents
	docs := []index.Document{
		{DocID: "doc1", Text: []byte("The quick brown fox jumps over the lazy dog")},
		{DocID: "doc2", Text: []byte("A fast red fox leaps across the sleeping hound")},
		{DocID: "doc3", Text: []byte("The lazy cat sits on the warm windowsill")},
		{DocID: "doc4", Text: []byte("Quick brown rabbits hop through the garden fence")},
		{DocID: "doc5", Text: []byte("The fox and the hound are natural enemies in the wild")},
	}
	if err := e.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// Search for "fox"
	results, err := e.Search(ctx, index.Query{Text: "fox", Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("expected search hits for 'fox', got none")
	}
	t.Logf("Search 'fox': %d hits", len(results.Hits))
	for _, h := range results.Hits {
		t.Logf("  %s score=%.4f", h.DocID, h.Score)
	}

	// Search for "quick brown"
	results2, err := e.Search(ctx, index.Query{Text: "quick brown", Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results2.Hits) == 0 {
		t.Fatal("expected search hits for 'quick brown', got none")
	}
	t.Logf("Search 'quick brown': %d hits", len(results2.Hits))
	for _, h := range results2.Hits {
		t.Logf("  %s score=%.4f", h.DocID, h.Score)
	}

	// Verify stats
	stats, err := e.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount != 5 {
		t.Errorf("expected DocCount=5, got %d", stats.DocCount)
	}
	t.Logf("Stats: docs=%d disk=%d bytes", stats.DocCount, stats.DiskBytes)

	// Verify lotus.meta file exists
	if _, err := os.Stat(dir + "/lotus.meta"); os.IsNotExist(err) {
		t.Error("lotus.meta not created")
	}
}

func TestEngine_MultiBatch(t *testing.T) {
	dir := t.TempDir()
	e := &Engine{}
	ctx := context.Background()

	if err := e.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer e.Close()

	// Index two separate batches (creates two segments)
	batch1 := []index.Document{
		{DocID: "a1", Text: []byte("machine learning algorithms for natural language processing")},
		{DocID: "a2", Text: []byte("deep neural networks revolutionize computer vision tasks")},
	}
	batch2 := []index.Document{
		{DocID: "b1", Text: []byte("natural language understanding with transformer models")},
		{DocID: "b2", Text: []byte("reinforcement learning for robotic control systems")},
	}

	if err := e.Index(ctx, batch1); err != nil {
		t.Fatalf("Index batch1: %v", err)
	}
	if err := e.Index(ctx, batch2); err != nil {
		t.Fatalf("Index batch2: %v", err)
	}

	// Search should find results from both segments
	results, err := e.Search(ctx, index.Query{Text: "natural language", Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) < 2 {
		t.Errorf("expected >=2 hits for 'natural language', got %d", len(results.Hits))
	}
	t.Logf("Search 'natural language': %d hits", len(results.Hits))
	for _, h := range results.Hits {
		t.Logf("  %s score=%.4f", h.DocID, h.Score)
	}
}

func TestEngine_BooleanMust(t *testing.T) {
	dir := t.TempDir()
	e := &Engine{}
	ctx := context.Background()

	if err := e.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer e.Close()

	docs := []index.Document{
		{DocID: "d1", Text: []byte("apple banana cherry")},
		{DocID: "d2", Text: []byte("apple cherry date")},
		{DocID: "d3", Text: []byte("banana cherry elderberry")},
		{DocID: "d4", Text: []byte("apple banana date")},
	}
	if err := e.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// +apple +banana = must contain both
	results, err := e.Search(ctx, index.Query{Text: "+apple +banana", Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	t.Logf("Search '+apple +banana': %d hits", len(results.Hits))
	for _, h := range results.Hits {
		t.Logf("  %s score=%.4f", h.DocID, h.Score)
	}
	// Should find d1 and d4 (both have apple AND banana)
	if len(results.Hits) != 2 {
		t.Errorf("expected 2 hits for '+apple +banana', got %d", len(results.Hits))
	}
}

func TestEngine_Registration(t *testing.T) {
	eng, err := index.NewEngine("lotus")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if eng.Name() != "lotus" {
		t.Errorf("expected name 'lotus', got %q", eng.Name())
	}
}
