package elasticsearch_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/elasticsearch"
)

func skipIfElasticsearchDown(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:9201/", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("elasticsearch not available at localhost:9201: %v", err)
	}
	resp.Body.Close()
}

func TestElasticsearchEngine_AddrSetter(t *testing.T) {
	eng, err := index.NewEngine("elasticsearch")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	setter, ok := eng.(index.AddrSetter)
	if !ok {
		t.Fatal("elasticsearch Engine does not implement AddrSetter")
	}
	setter.SetAddr("http://my-server:9200")
}

func TestElasticsearchEngine_Roundtrip(t *testing.T) {
	skipIfElasticsearchDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := index.NewEngine("elasticsearch")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	eng.(index.AddrSetter).SetAddr("http://localhost:9201")

	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "es-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "es-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "es-doc3", Text: []byte("open source software development programming")},
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
