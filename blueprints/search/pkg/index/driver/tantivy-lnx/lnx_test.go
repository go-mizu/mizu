package lnx_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/tantivy-lnx"
)

func skipIfLnxDown(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:8000/api/v1/indexes", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("lnx not available at localhost:8000: %v", err)
	}
	resp.Body.Close()
}

func TestLnxEngine_AddrSetter(t *testing.T) {
	eng, err := index.NewEngine("tantivy-lnx")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	setter, ok := eng.(index.AddrSetter)
	if !ok {
		t.Fatal("tantivy-lnx Engine does not implement AddrSetter")
	}
	setter.SetAddr("http://my-server:8000")
}

func TestLnxEngine_Roundtrip(t *testing.T) {
	skipIfLnxDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := index.NewEngine("tantivy-lnx")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	eng.(index.AddrSetter).SetAddr("http://localhost:8000")

	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "lnx-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "lnx-doc2", Text: []byte("climate change global warming renewable energy")},
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
}
