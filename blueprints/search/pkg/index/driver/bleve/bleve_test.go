package bleve_test

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/bleve"
)

func TestBleveEngine_Roundtrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := index.NewEngine("bleve")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("Search: expected at least one hit, got none")
	}
	// doc1 should be in results
	found := false
	for _, h := range results.Hits {
		if h.DocID == "doc1" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected doc1 in results, got: %v", results.Hits)
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount != 3 {
		t.Errorf("DocCount: got %d, want 3", stats.DocCount)
	}
	if stats.DiskBytes == 0 {
		t.Error("DiskBytes should be > 0 after indexing")
	}
}

func TestBleveEngine_Name(t *testing.T) {
	eng, _ := index.NewEngine("bleve")
	if eng.Name() != "bleve" {
		t.Errorf("Name: got %q, want %q", eng.Name(), "bleve")
	}
}

func TestBleveEngine_EmptySearch(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	eng, _ := index.NewEngine("bleve")
	eng.Open(ctx, dir)
	defer eng.Close()

	// Empty index — search should not error
	results, err := eng.Search(ctx, index.Query{Text: "anything", Limit: 5})
	if err != nil {
		t.Fatalf("Search on empty index: %v", err)
	}
	if len(results.Hits) != 0 {
		t.Errorf("expected 0 hits on empty index, got %d", len(results.Hits))
	}
}
