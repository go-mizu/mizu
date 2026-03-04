package llamacpp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
)

func TestDriverEmbed(t *testing.T) {
	dim := 4

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path != "/v1/embeddings" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var req embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := embeddingResponse{
			Data: make([]embeddingData, len(req.Input)),
		}
		for i := range req.Input {
			vec := make([]float32, dim)
			for j := range vec {
				vec[j] = float32(i+1) * 0.1 * float32(j+1)
			}
			resp.Data[i] = embeddingData{
				Embedding: vec,
				Index:     i,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ctx := context.Background()
	d := &Driver{}
	err := d.Open(ctx, embed.Config{
		Addr:      srv.URL,
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
	var requestCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		requestCount++
		var req embeddingRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp := embeddingResponse{Data: make([]embeddingData, len(req.Input))}
		for i := range req.Input {
			resp.Data[i] = embeddingData{
				Embedding: make([]float32, dim),
				Index:     i,
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ctx := context.Background()
	d := &Driver{}
	err := d.Open(ctx, embed.Config{
		Addr:      srv.URL,
		BatchSize: 2,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	requestCount = 0 // reset after probe

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
	if requestCount != 3 {
		t.Errorf("expected 3 batch requests, got %d", requestCount)
	}
}

func TestDriverServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not loaded", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx := context.Background()
	d := &Driver{}
	err := d.Open(ctx, embed.Config{Addr: srv.URL})
	if err == nil {
		d.Close()
		t.Fatal("expected Open to fail when server returns error")
	}
}

func TestDriverName(t *testing.T) {
	d := &Driver{}
	if d.Name() != "llamacpp" {
		t.Errorf("Name() = %q, want %q", d.Name(), "llamacpp")
	}
	d.model = "nomic-embed"
	if d.Name() != "llamacpp/nomic-embed" {
		t.Errorf("Name() = %q, want %q", d.Name(), "llamacpp/nomic-embed")
	}
}
