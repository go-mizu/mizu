package driver_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/chroma"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/elasticsearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/meilisearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/milvus"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/opensearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/pgvector"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/qdrant"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/solr"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/typesense"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/weaviate"
)

type backendCase struct {
	name  string
	addr  string
	probe string
	isTCP bool
	opts  map[string]string
}

func TestVectorDriversRoundTrip(t *testing.T) {
	cases := []backendCase{
		{name: "qdrant", addr: "http://localhost:6333", probe: "http://localhost:6333/healthz"},
		{name: "weaviate", addr: "http://localhost:8080", probe: "http://localhost:8080/v1/.well-known/ready"},
		{name: "milvus", addr: "http://localhost:19530", probe: "http://localhost:19530/v2/vectordb/collections/list"},
		{name: "chroma", addr: "http://localhost:8000", probe: "http://localhost:8000/api/v1/heartbeat"},
		{name: "elasticsearch", addr: "http://localhost:9201", probe: "http://localhost:9201"},
		{name: "opensearch", addr: "http://localhost:9200", probe: "http://localhost:9200"},
		{name: "meilisearch", addr: "http://localhost:7700", probe: "http://localhost:7700/health"},
		{name: "typesense", addr: "http://localhost:8108", probe: "http://localhost:8108/health", opts: map[string]string{"api_key": "mizu-typesense-key"}},
		{name: "pgvector", addr: "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable", probe: "localhost:5432", isTCP: true},
		{name: "solr", addr: "http://localhost:8983", probe: "http://localhost:8983/solr/admin/info/system?wt=json"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !isAvailable(tc) {
				t.Skipf("%s is not available", tc.name)
			}

			store, err := vector.Open(tc.name, vector.Config{Addr: tc.addr, Options: tc.opts})
			if err != nil {
				t.Skipf("%s open failed (likely not fully ready): %v", tc.name, err)
			}
			if closer, ok := store.(vector.Closer); ok {
				defer func() { _ = closer.Close() }()
			}

			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			col := store.Collection(fmt.Sprintf("mizu_vector_test_%s_%d", tc.name, time.Now().UnixNano()))
			if err := col.Init(ctx); err != nil {
				t.Fatalf("Init: %v", err)
			}

			items := []vector.Item{
				{ID: "v1", Vector: []float32{0.10, 0.20, 0.30}, Metadata: map[string]string{"topic": "ai"}},
				{ID: "v2", Vector: []float32{0.11, 0.19, 0.31}, Metadata: map[string]string{"topic": "ai"}},
				{ID: "v3", Vector: []float32{0.90, 0.10, 0.10}, Metadata: map[string]string{"topic": "db"}},
			}
			if err := col.Index(ctx, items); err != nil {
				t.Fatalf("Index: %v", err)
			}

			res, err := col.Search(ctx, vector.Query{Vector: []float32{0.10, 0.21, 0.29}, K: 2})
			if err != nil {
				t.Fatalf("Search: %v", err)
			}
			if len(res.Hits) == 0 {
				t.Fatalf("expected hits, got none")
			}
		})
	}
}

func TestVectorDriversRegistered(t *testing.T) {
	want := []string{
		"chroma",
		"elasticsearch",
		"meilisearch",
		"milvus",
		"opensearch",
		"pgvector",
		"qdrant",
		"solr",
		"typesense",
		"weaviate",
	}
	got := map[string]bool{}
	for _, name := range vector.List() {
		got[name] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Fatalf("driver %q not registered; got=%v", name, vector.List())
		}
	}
}

func isAvailable(tc backendCase) bool {
	if tc.isTCP {
		conn, err := net.DialTimeout("tcp", tc.probe, 1500*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, tc.probe, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 500
}
