//go:build tantivy

package tantivy_test

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-go"
)

func TestTantivyEngine_Roundtrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := index.NewEngine("tantivy")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "tv-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "tv-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "tv-doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("Search: expected hits, got none")
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount != 3 {
		t.Errorf("DocCount: got %d, want 3", stats.DocCount)
	}
}
