// Package testutil provides shared test utilities for vector database drivers.
package testutil

import (
	"context"
	"math/rand"
	"testing"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
)

// RunDriverTests runs a comprehensive test suite against a vector database.
func RunDriverTests(t *testing.T, db vectorize.DB) {
	t.Helper()

	ctx := context.Background()
	indexName := "test_index_" + randomString(8)
	dimensions := 128

	// Clean up after tests
	defer func() {
		db.DeleteIndex(ctx, indexName)
	}()

	// Test Ping
	t.Run("Ping", func(t *testing.T) {
		if err := db.Ping(ctx); err != nil {
			t.Fatalf("Ping failed: %v", err)
		}
	})

	// Test CreateIndex
	t.Run("CreateIndex", func(t *testing.T) {
		idx := &vectorize.Index{
			Name:        indexName,
			Dimensions:  dimensions,
			Metric:      vectorize.Cosine,
			Description: "Test index",
		}
		if err := db.CreateIndex(ctx, idx); err != nil {
			t.Fatalf("CreateIndex failed: %v", err)
		}

		// Try to create duplicate
		if err := db.CreateIndex(ctx, idx); err != vectorize.ErrIndexExists {
			t.Fatalf("Expected ErrIndexExists, got: %v", err)
		}
	})

	// Test GetIndex
	t.Run("GetIndex", func(t *testing.T) {
		idx, err := db.GetIndex(ctx, indexName)
		if err != nil {
			t.Fatalf("GetIndex failed: %v", err)
		}
		if idx.Name != indexName {
			t.Errorf("Expected name %s, got %s", indexName, idx.Name)
		}
		if idx.Dimensions != dimensions {
			t.Errorf("Expected dimensions %d, got %d", dimensions, idx.Dimensions)
		}
	})

	// Test ListIndexes
	t.Run("ListIndexes", func(t *testing.T) {
		indexes, err := db.ListIndexes(ctx)
		if err != nil {
			t.Fatalf("ListIndexes failed: %v", err)
		}
		found := false
		for _, idx := range indexes {
			if idx.Name == indexName {
				found = true
				break
			}
		}
		if !found {
			t.Error("Created index not found in list")
		}
	})

	// Test Insert
	t.Run("Insert", func(t *testing.T) {
		vectors := GenerateVectors(10, dimensions)
		if err := db.Insert(ctx, indexName, vectors); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	})

	// Test Search
	t.Run("Search", func(t *testing.T) {
		queryVector := randomVector(dimensions)
		opts := &vectorize.SearchOptions{
			TopK:           5,
			ReturnMetadata: true,
		}
		matches, err := db.Search(ctx, indexName, queryVector, opts)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(matches) == 0 {
			t.Error("Expected matches, got none")
		}
		if len(matches) > opts.TopK {
			t.Errorf("Expected max %d matches, got %d", opts.TopK, len(matches))
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		ids := []string{"vec_0", "vec_1", "vec_2"}
		vectors, err := db.Get(ctx, indexName, ids)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(vectors) != len(ids) {
			t.Errorf("Expected %d vectors, got %d", len(ids), len(vectors))
		}
	})

	// Test Upsert
	t.Run("Upsert", func(t *testing.T) {
		vectors := []*vectorize.Vector{
			{
				ID:       "vec_0",
				Values:   randomVector(dimensions),
				Metadata: map[string]any{"updated": true},
			},
		}
		if err := db.Upsert(ctx, indexName, vectors); err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		ids := []string{"vec_9"}
		if err := db.Delete(ctx, indexName, ids); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify deletion
		vectors, err := db.Get(ctx, indexName, ids)
		if err != nil {
			t.Fatalf("Get after delete failed: %v", err)
		}
		if len(vectors) != 0 {
			t.Error("Deleted vector still exists")
		}
	})

	// Test DeleteIndex
	t.Run("DeleteIndex", func(t *testing.T) {
		if err := db.DeleteIndex(ctx, indexName); err != nil {
			t.Fatalf("DeleteIndex failed: %v", err)
		}

		// Verify deletion
		_, err := db.GetIndex(ctx, indexName)
		if err != vectorize.ErrIndexNotFound {
			t.Fatalf("Expected ErrIndexNotFound after delete, got: %v", err)
		}
	})
}

// GenerateVectors creates test vectors with sequential IDs.
func GenerateVectors(count, dimensions int) []*vectorize.Vector {
	vectors := make([]*vectorize.Vector, count)
	for i := 0; i < count; i++ {
		vectors[i] = &vectorize.Vector{
			ID:        generateID(i),
			Values:    randomVector(dimensions),
			Namespace: "",
			Metadata: map[string]any{
				"index":    i,
				"category": []string{"A", "B", "C"}[i%3],
			},
		}
	}
	return vectors
}

// GenerateVectorsWithNamespace creates test vectors with a specific namespace.
func GenerateVectorsWithNamespace(count, dimensions int, namespace string) []*vectorize.Vector {
	vectors := GenerateVectors(count, dimensions)
	for _, v := range vectors {
		v.Namespace = namespace
	}
	return vectors
}

func generateID(i int) string {
	return "vec_" + randomString(4) + "_" + itoa(i)
}

func randomVector(dimensions int) []float32 {
	v := make([]float32, dimensions)
	for i := range v {
		v[i] = rand.Float32()*2 - 1 // Range [-1, 1]
	}
	return v
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func init() {
	// rand.Seed is deprecated in Go 1.20+, global rand is auto-seeded
}
