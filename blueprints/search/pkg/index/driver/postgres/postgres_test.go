package postgres_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/postgres"
)

func skipIfPostgresDown(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "localhost:5432", 2*time.Second)
	if err != nil {
		t.Skipf("postgres not available at localhost:5432: %v", err)
	}
	conn.Close()
}

func TestPostgresEngine_AddrSetter(t *testing.T) {
	eng, err := index.NewEngine("postgres")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	setter, ok := eng.(index.AddrSetter)
	if !ok {
		t.Fatal("postgres Engine does not implement AddrSetter")
	}
	setter.SetAddr("postgres://user:pass@host:5432/db")
}

func TestPostgresEngine_Roundtrip(t *testing.T) {
	skipIfPostgresDown(t)
	ctx := context.Background()
	dir := t.TempDir()

	eng, err := index.NewEngine("postgres")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	eng.(index.AddrSetter).SetAddr("postgres://fineweb:fineweb@localhost:5432/fts")

	if err := eng.Open(ctx, dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	docs := []index.Document{
		{DocID: "pg-doc1", Text: []byte("machine learning algorithms deep neural networks")},
		{DocID: "pg-doc2", Text: []byte("climate change global warming renewable energy")},
		{DocID: "pg-doc3", Text: []byte("open source software development programming")},
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
