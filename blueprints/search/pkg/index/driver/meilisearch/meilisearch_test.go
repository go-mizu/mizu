package meilisearch_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/meilisearch"
)

func skipIfMeilisearchDown(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:7700/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("meilisearch not available at localhost:7700: %v", err)
	}
	resp.Body.Close()
}

func TestMeilisearchEngine_AddrSetter(t *testing.T) {
	eng, err := index.NewEngine("meilisearch")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	setter, ok := eng.(index.AddrSetter)
	if !ok {
		t.Fatal("meilisearch Engine does not implement AddrSetter")
	}
	setter.SetAddr("http://my-server:7700")
}

func TestMeilisearchEngine_Roundtrip(t *testing.T) {
	skipIfMeilisearchDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := index.NewEngine("meilisearch")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	setter, ok := eng.(index.AddrSetter)
	if !ok {
		t.Fatal("meilisearch Engine does not implement AddrSetter")
	}
	setter.SetAddr("http://localhost:7700")

	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "ms-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "ms-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "ms-doc3", Text: []byte("open source software development programming")},
	}
	if err := eng.Index(ctx, docs); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := eng.Search(ctx, index.Query{Text: "machine learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Hits) == 0 {
		t.Fatal("expected hits, got none")
	}

	stats, err := eng.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.DocCount < 3 {
		t.Errorf("DocCount: got %d, want >= 3", stats.DocCount)
	}
}
