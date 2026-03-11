package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
)

// mockServer creates a test HTTP server that returns dim-dimensional embeddings.
// It returns one embedding per request entry in the batch.
func mockServer(t *testing.T, dim int, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if statusCode != http.StatusOK {
			http.Error(w, "api error", statusCode)
			return
		}

		var req batchEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := batchEmbedResponse{
			Embeddings: make([]embeddingValue, len(req.Requests)),
		}
		for i := range req.Requests {
			vec := make([]float32, dim)
			for j := range vec {
				vec[j] = float32(i+1) * 0.1 * float32(j+1)
			}
			resp.Embeddings[i] = embeddingValue{Values: vec}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestDriverEmbed(t *testing.T) {
	dim := 4
	srv := mockServer(t, dim, http.StatusOK)
	defer srv.Close()

	ctx := context.Background()
	d := &Driver{baseURL: srv.URL}
	err := d.Open(ctx, embed.Config{
		Addr:      "fake-key",
		BatchSize: 2,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	if d.Dimension() != dim {
		t.Fatalf("Dimension() = %d, want %d", d.Dimension(), dim)
	}

	vecs, err := d.Embed(ctx, []embed.Input{
		{Text: "hello"},
		{Text: "world"},
		{Text: "foo"},
	})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("got %d vectors, want 3", len(vecs))
	}
	for i, v := range vecs {
		if len(v.Values) != dim {
			t.Errorf("vec[%d] has %d dims, want %d", i, len(v.Values), dim)
		}
	}
}

func TestDriverBatching(t *testing.T) {
	dim := 2
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		var req batchEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := batchEmbedResponse{
			Embeddings: make([]embeddingValue, len(req.Requests)),
		}
		for i := range req.Requests {
			resp.Embeddings[i] = embeddingValue{Values: make([]float32, dim)}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ctx := context.Background()
	d := &Driver{baseURL: srv.URL}
	err := d.Open(ctx, embed.Config{
		Addr:      "fake-key",
		BatchSize: 2,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// Reset after Open's probe call.
	requestCount.Store(0)

	inputs := make([]embed.Input, 5)
	for i := range inputs {
		inputs[i] = embed.Input{Text: "text"}
	}

	vecs, err := d.Embed(ctx, inputs)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 5 {
		t.Fatalf("got %d vectors, want 5", len(vecs))
	}

	// 5 inputs with batch size 2 → 3 requests (2+2+1)
	if got := requestCount.Load(); got != 3 {
		t.Errorf("expected 3 batch requests, got %d", got)
	}
}

func TestDriverAPIError(t *testing.T) {
	srv := mockServer(t, 4, http.StatusUnauthorized)
	defer srv.Close()

	ctx := context.Background()
	d := &Driver{baseURL: srv.URL}
	err := d.Open(ctx, embed.Config{Addr: "bad-key"})
	if err == nil {
		d.Close()
		t.Fatal("expected Open to fail when server returns 401")
	}
}

func TestDriverName(t *testing.T) {
	d := &Driver{}
	if d.Name() != "gemini" {
		t.Errorf("Name() = %q, want %q", d.Name(), "gemini")
	}
	d.model = "text-embedding-004"
	if d.Name() != "gemini/text-embedding-004" {
		t.Errorf("Name() = %q, want %q", d.Name(), "gemini/text-embedding-004")
	}
}

func TestLoadLocalEnv(t *testing.T) {
	dir := t.TempDir()

	// Test with "export KEY=VALUE" format (quoted).
	exportFile := filepath.Join(dir, "export.env")
	content := `# comment
OTHER_KEY=ignored
export GEMINI_API_KEY="test-key-123"
export ANOTHER_KEY='other-value'
`
	if err := os.WriteFile(exportFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got := loadKeyFromFile(exportFile, "GEMINI_API_KEY")
	if got != "test-key-123" {
		t.Errorf("loadKeyFromFile (export quoted) = %q, want %q", got, "test-key-123")
	}

	// Test plain KEY=VALUE format (no export, no quotes).
	plainFile := filepath.Join(dir, "plain.env")
	plainContent := `GEMINI_API_KEY=plain-key-456
OTHER=something
`
	if err := os.WriteFile(plainFile, []byte(plainContent), 0600); err != nil {
		t.Fatal(err)
	}

	got = loadKeyFromFile(plainFile, "GEMINI_API_KEY")
	if got != "plain-key-456" {
		t.Errorf("loadKeyFromFile (plain) = %q, want %q", got, "plain-key-456")
	}

	// Test missing key returns empty string.
	got = loadKeyFromFile(exportFile, "NONEXISTENT_KEY")
	if got != "" {
		t.Errorf("loadKeyFromFile (missing) = %q, want empty string", got)
	}

	// Test non-existent file returns empty string.
	got = loadKeyFromFile(filepath.Join(dir, "nonexistent.env"), "GEMINI_API_KEY")
	if got != "" {
		t.Errorf("loadKeyFromFile (no file) = %q, want empty string", got)
	}
}

func TestDriverEmbedCountMismatch(t *testing.T) {
	// Server returns n-1 embeddings for an n-text request.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req batchEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Return one fewer embedding than requested.
		count := len(req.Requests) - 1
		if count < 0 {
			count = 0
		}
		resp := batchEmbedResponse{
			Embeddings: make([]embeddingValue, count),
		}
		for i := range resp.Embeddings {
			resp.Embeddings[i] = embeddingValue{Values: []float32{0.1, 0.2, 0.3, 0.4}}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ctx := context.Background()
	// Open with a batch size larger than 3 so the probe (1 text → 0 returned) would fail.
	// Use batch size 10 so Open's probe sends 1 text and gets 0 back → Open itself fails.
	// Instead, open with a server that returns correct count for the probe (1 text → 1 embedding),
	// then switch to the mismatch server for Embed. We achieve this by using a counter.
	var callCount atomic.Int32
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req batchEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		n := callCount.Add(1)
		var count int
		if n == 1 {
			// First call is the probe — return correct count so Open succeeds.
			count = len(req.Requests)
		} else {
			// Subsequent calls return n-1 embeddings.
			count = len(req.Requests) - 1
			if count < 0 {
				count = 0
			}
		}
		resp := batchEmbedResponse{
			Embeddings: make([]embeddingValue, count),
		}
		for i := range resp.Embeddings {
			resp.Embeddings[i] = embeddingValue{Values: []float32{0.1, 0.2, 0.3, 0.4}}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv2.Close()
	_ = srv // unused but kept to avoid lint noise on the first server definition above

	d := &Driver{baseURL: srv2.URL}
	if err := d.Open(ctx, embed.Config{Addr: "fake-key"}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	_, err := d.Embed(ctx, []embed.Input{
		{Text: "a"},
		{Text: "b"},
		{Text: "c"},
	})
	if err == nil {
		t.Fatal("expected error for count mismatch, got nil")
	}
}

func TestParseModelName(t *testing.T) {
	tests := []struct {
		input    string
		wantModel string
		wantDim  int
	}{
		{"text-embedding-004", "text-embedding-004", 0},
		{"gemini-embedding-exp-03-07", "gemini-embedding-exp-03-07", 0},
		{"gemini-embedding-exp-03-07:768", "gemini-embedding-exp-03-07", 768},
		{"gemini-embedding-exp-03-07:1536", "gemini-embedding-exp-03-07", 1536},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			gotModel, gotDim := parseModelName(tc.input)
			if gotModel != tc.wantModel {
				t.Errorf("parseModelName(%q) model = %q, want %q", tc.input, gotModel, tc.wantModel)
			}
			if gotDim != tc.wantDim {
				t.Errorf("parseModelName(%q) dim = %d, want %d", tc.input, gotDim, tc.wantDim)
			}
		})
	}
}
