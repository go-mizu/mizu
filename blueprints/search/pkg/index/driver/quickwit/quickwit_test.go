package quickwit_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/quickwit"
)

func skipIfQuickwitDown(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:7280/api/v1/version", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("quickwit not available at localhost:7280: %v", err)
	}
	resp.Body.Close()
}

func TestQuickwitEngine_AddrSetter(t *testing.T) {
	eng, err := index.NewEngine("quickwit")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	setter, ok := eng.(index.AddrSetter)
	if !ok {
		t.Fatal("quickwit Engine does not implement AddrSetter")
	}
	setter.SetAddr("http://my-server:7280")
}

func TestQuickwitEngine_Roundtrip(t *testing.T) {
	skipIfQuickwitDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := index.NewEngine("quickwit")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	eng.(index.AddrSetter).SetAddr("http://localhost:7280")

	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "qw-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "qw-doc2", Text: []byte("climate change global warming renewable energy")},
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
