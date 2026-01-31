package memory

import (
	"math"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *MemoryStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewMemoryStore(dbPath)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}
	if err := store.EnsureSchema(); err != nil {
		store.Close()
		t.Fatalf("EnsureSchema: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestNewMemoryStore_EnsureSchema(t *testing.T) {
	store := newTestStore(t)

	// Verify the four key tables exist by querying sqlite_master.
	tables := map[string]bool{
		"files":           false,
		"chunks":          false,
		"chunks_fts":      false,
		"embedding_cache": false,
	}

	rows, err := store.db.Query(`SELECT name FROM sqlite_master WHERE type IN ('table','view')`)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if _, ok := tables[name]; ok {
			tables[name] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}

	for tbl, found := range tables {
		if !found {
			t.Errorf("table %q not found after EnsureSchema", tbl)
		}
	}
}

func TestUpsertFile_GetFileHash_Roundtrip(t *testing.T) {
	store := newTestStore(t)

	// Initially no hash.
	hash, err := store.GetFileHash("foo.go")
	if err != nil {
		t.Fatalf("GetFileHash: %v", err)
	}
	if hash != "" {
		t.Fatalf("expected empty hash for unknown file, got %q", hash)
	}

	// Upsert and read back.
	if err := store.UpsertFile("foo.go", "file", "abc123", 1000, 512); err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}

	hash, err = store.GetFileHash("foo.go")
	if err != nil {
		t.Fatalf("GetFileHash after upsert: %v", err)
	}
	if hash != "abc123" {
		t.Errorf("hash = %q, want %q", hash, "abc123")
	}

	// Update the hash.
	if err := store.UpsertFile("foo.go", "file", "def456", 2000, 1024); err != nil {
		t.Fatalf("UpsertFile update: %v", err)
	}
	hash, err = store.GetFileHash("foo.go")
	if err != nil {
		t.Fatalf("GetFileHash after update: %v", err)
	}
	if hash != "def456" {
		t.Errorf("hash = %q, want %q", hash, "def456")
	}
}

func TestUpsertChunk_DeleteChunksByPath(t *testing.T) {
	store := newTestStore(t)

	// Insert two chunks for the same path.
	if err := store.UpsertChunk("f:1", "f.go", "file", 1, 10, "h1", "", "text1", nil); err != nil {
		t.Fatalf("UpsertChunk 1: %v", err)
	}
	if err := store.UpsertChunk("f:11", "f.go", "file", 11, 20, "h2", "", "text2", nil); err != nil {
		t.Fatalf("UpsertChunk 2: %v", err)
	}

	// Verify chunks exist.
	var count int
	if err := store.db.QueryRow(`SELECT count(*) FROM chunks WHERE path = ?`, "f.go").Scan(&count); err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 chunks, got %d", count)
	}

	// Delete by path.
	if err := store.DeleteChunksByPath("f.go"); err != nil {
		t.Fatalf("DeleteChunksByPath: %v", err)
	}

	if err := store.db.QueryRow(`SELECT count(*) FROM chunks WHERE path = ?`, "f.go").Scan(&count); err != nil {
		t.Fatalf("count chunks after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 chunks after delete, got %d", count)
	}
}

func TestSearchFTS(t *testing.T) {
	store := newTestStore(t)

	// Insert chunks with searchable text.
	if err := store.UpsertChunk("a:1", "a.go", "file", 1, 5, "h1", "", "the quick brown fox jumps over the lazy dog", nil); err != nil {
		t.Fatalf("UpsertChunk: %v", err)
	}
	if err := store.UpsertChunk("b:1", "b.go", "file", 1, 5, "h2", "", "hello world greetings from the server", nil); err != nil {
		t.Fatalf("UpsertChunk: %v", err)
	}

	// Search for "fox".
	results, err := store.SearchFTS("fox", 10)
	if err != nil {
		t.Fatalf("SearchFTS: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 FTS result for 'fox'")
	}
	if results[0].Path != "a.go" {
		t.Errorf("expected path 'a.go', got %q", results[0].Path)
	}

	// Search for "hello" should find b.go.
	results, err = store.SearchFTS("hello", 10)
	if err != nil {
		t.Fatalf("SearchFTS hello: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 FTS result for 'hello'")
	}
	if results[0].Path != "b.go" {
		t.Errorf("expected path 'b.go', got %q", results[0].Path)
	}

	// Search for something not present.
	results, err = store.SearchFTS("xyzzy", 10)
	if err != nil {
		t.Fatalf("SearchFTS xyzzy: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for 'xyzzy', got %d", len(results))
	}

	// Empty query returns nil.
	results, err = store.SearchFTS("", 10)
	if err != nil {
		t.Fatalf("SearchFTS empty: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty query, got %d results", len(results))
	}
}

func TestSearchVector(t *testing.T) {
	store := newTestStore(t)

	// Insert chunks with embeddings (simple 3-dimensional vectors).
	emb1 := []float64{1.0, 0.0, 0.0}
	emb2 := []float64{0.0, 1.0, 0.0}
	emb3 := []float64{0.7, 0.7, 0.0} // closer to emb1 than emb2

	if err := store.UpsertChunk("v:1", "v1.go", "file", 1, 5, "h1", "model", "vector one", emb1); err != nil {
		t.Fatalf("UpsertChunk: %v", err)
	}
	if err := store.UpsertChunk("v:2", "v2.go", "file", 1, 5, "h2", "model", "vector two", emb2); err != nil {
		t.Fatalf("UpsertChunk: %v", err)
	}
	if err := store.UpsertChunk("v:3", "v3.go", "file", 1, 5, "h3", "model", "vector three", emb3); err != nil {
		t.Fatalf("UpsertChunk: %v", err)
	}

	// Query with a vector close to emb1.
	queryVec := []float64{0.9, 0.1, 0.0}
	results, err := store.SearchVector(queryVec, 10)
	if err != nil {
		t.Fatalf("SearchVector: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 vector result")
	}

	// First result should be v1.go (most similar to query).
	if results[0].Path != "v1.go" {
		t.Errorf("expected first result 'v1.go', got %q", results[0].Path)
	}

	// Empty query returns nil.
	results, err = store.SearchVector(nil, 10)
	if err != nil {
		t.Fatalf("SearchVector nil: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty query vec, got %d results", len(results))
	}
}

func TestGetCachedEmbedding_SetCachedEmbedding(t *testing.T) {
	store := newTestStore(t)

	// Initially not cached.
	emb, err := store.GetCachedEmbedding("hash1", "model-a")
	if err != nil {
		t.Fatalf("GetCachedEmbedding: %v", err)
	}
	if emb != nil {
		t.Fatalf("expected nil for uncached embedding, got %v", emb)
	}

	// Cache an embedding.
	want := []float64{0.1, 0.2, 0.3, 0.4}
	if err := store.SetCachedEmbedding("hash1", "model-a", want); err != nil {
		t.Fatalf("SetCachedEmbedding: %v", err)
	}

	// Read back.
	emb, err = store.GetCachedEmbedding("hash1", "model-a")
	if err != nil {
		t.Fatalf("GetCachedEmbedding after set: %v", err)
	}
	if len(emb) != len(want) {
		t.Fatalf("embedding length = %d, want %d", len(emb), len(want))
	}
	for i := range want {
		if math.Abs(emb[i]-want[i]) > 1e-12 {
			t.Errorf("emb[%d] = %f, want %f", i, emb[i], want[i])
		}
	}

	// Different model should not return the same cached embedding.
	emb, err = store.GetCachedEmbedding("hash1", "model-b")
	if err != nil {
		t.Fatalf("GetCachedEmbedding different model: %v", err)
	}
	if emb != nil {
		t.Errorf("expected nil for different model, got %v", emb)
	}
}

func TestDeleteFile(t *testing.T) {
	store := newTestStore(t)

	if err := store.UpsertFile("del.go", "file", "hash", 100, 50); err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}

	hash, err := store.GetFileHash("del.go")
	if err != nil {
		t.Fatalf("GetFileHash: %v", err)
	}
	if hash != "hash" {
		t.Fatalf("hash = %q, want %q", hash, "hash")
	}

	if err := store.DeleteFile("del.go"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}

	hash, err = store.GetFileHash("del.go")
	if err != nil {
		t.Fatalf("GetFileHash after delete: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty hash after delete, got %q", hash)
	}
}

func TestEncodeDecodeEmbedding_Roundtrip(t *testing.T) {
	// Normal vector.
	vec := []float64{1.1, 2.2, 3.3}
	encoded, err := encodeEmbedding(vec)
	if err != nil {
		t.Fatalf("encodeEmbedding: %v", err)
	}
	if encoded == "" {
		t.Fatal("encoded should not be empty for non-empty vector")
	}

	decoded, err := decodeEmbedding(encoded)
	if err != nil {
		t.Fatalf("decodeEmbedding: %v", err)
	}
	if len(decoded) != len(vec) {
		t.Fatalf("decoded length = %d, want %d", len(decoded), len(vec))
	}
	for i := range vec {
		if math.Abs(decoded[i]-vec[i]) > 1e-12 {
			t.Errorf("decoded[%d] = %f, want %f", i, decoded[i], vec[i])
		}
	}
}

func TestEncodeDecodeEmbedding_Empty(t *testing.T) {
	// Empty vector => empty string.
	encoded, err := encodeEmbedding(nil)
	if err != nil {
		t.Fatalf("encodeEmbedding(nil): %v", err)
	}
	if encoded != "" {
		t.Errorf("encodeEmbedding(nil) = %q, want \"\"", encoded)
	}

	encoded, err = encodeEmbedding([]float64{})
	if err != nil {
		t.Fatalf("encodeEmbedding([]): %v", err)
	}
	if encoded != "" {
		t.Errorf("encodeEmbedding([]) = %q, want \"\"", encoded)
	}

	// Empty string => nil.
	decoded, err := decodeEmbedding("")
	if err != nil {
		t.Fatalf("decodeEmbedding(\"\"): %v", err)
	}
	if decoded != nil {
		t.Errorf("decodeEmbedding(\"\") = %v, want nil", decoded)
	}
}

func TestCosineSimilarity_Identical(t *testing.T) {
	a := []float64{1.0, 2.0, 3.0}
	got := cosineSimilarity(a, a)
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("cosineSimilarity(identical) = %f, want 1.0", got)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float64{1.0, 0.0, 0.0}
	b := []float64{0.0, 1.0, 0.0}
	got := cosineSimilarity(a, b)
	if math.Abs(got) > 1e-9 {
		t.Errorf("cosineSimilarity(orthogonal) = %f, want 0.0", got)
	}
}

func TestCosineSimilarity_DifferentLength(t *testing.T) {
	a := []float64{1.0, 2.0}
	b := []float64{1.0, 2.0, 3.0}
	got := cosineSimilarity(a, b)
	if got != 0.0 {
		t.Errorf("cosineSimilarity(different length) = %f, want 0.0", got)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float64{0.0, 0.0, 0.0}
	b := []float64{1.0, 2.0, 3.0}
	got := cosineSimilarity(a, b)
	if got != 0.0 {
		t.Errorf("cosineSimilarity(zero vec) = %f, want 0.0", got)
	}
}

func TestEnsureSchema_Idempotent(t *testing.T) {
	store := newTestStore(t)

	// Calling EnsureSchema again should not fail.
	if err := store.EnsureSchema(); err != nil {
		t.Fatalf("second EnsureSchema: %v", err)
	}
}
