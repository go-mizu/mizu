//go:build integration

package gemini

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
)

func loadTestAPIKey(t *testing.T) string {
	t.Helper()
	home, _ := os.UserHomeDir()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = loadKeyFromFile(filepath.Join(home, "data", ".local.env"), "GEMINI_API_KEY")
	}
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set — skipping integration test")
	}
	return apiKey
}

// TestIntegrationEmbed tests text embedding with three model variants.
func TestIntegrationEmbed(t *testing.T) {
	apiKey := loadTestAPIKey(t)
	ctx := context.Background()

	tests := []struct {
		model   string
		wantDim int
	}{
		{"gemini-embedding-2-preview", 3072},
		{"gemini-embedding-2-preview:768", 768},
		{"gemini-embedding-2-preview:1536", 1536},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			d := &Driver{}
			if err := d.Open(ctx, embed.Config{
				Addr:  apiKey,
				Model: tt.model,
			}); err != nil {
				t.Fatalf("Open(%s): %v", tt.model, err)
			}
			defer d.Close()

			if d.Dimension() != tt.wantDim {
				t.Errorf("Dimension() = %d, want %d", d.Dimension(), tt.wantDim)
			}

			vecs, err := d.Embed(ctx, []embed.Input{
				{Text: "The quick brown fox jumps over the lazy dog."},
				{Text: "Go is an open-source programming language designed at Google."},
			})
			if err != nil {
				t.Fatalf("Embed: %v", err)
			}
			if len(vecs) != 2 {
				t.Fatalf("got %d vectors, want 2", len(vecs))
			}
			for i, v := range vecs {
				if len(v.Values) != tt.wantDim {
					t.Errorf("vec[%d]: got %d dims, want %d", i, len(v.Values), tt.wantDim)
				}
				nonzero := 0
				for _, f := range v.Values {
					if f != 0 {
						nonzero++
					}
				}
				if nonzero == 0 {
					t.Errorf("vec[%d]: all zeros — API issue", i)
				}
			}
			t.Logf("model=%s dim=%d ok", tt.model, tt.wantDim)
		})
	}
}

// TestIntegrationBatch verifies batching works correctly with the live API.
func TestIntegrationBatch(t *testing.T) {
	apiKey := loadTestAPIKey(t)
	ctx := context.Background()

	d := &Driver{}
	if err := d.Open(ctx, embed.Config{
		Addr:      apiKey,
		BatchSize: 3, // small batch to force multiple API calls
	}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// 7 inputs with batchSize=3 → 3 API calls (3+3+1)
	inputs := make([]embed.Input, 7)
	for i := range inputs {
		inputs[i] = embed.Input{Text: fmt.Sprintf("test sentence number %d for embedding", i+1)}
	}

	vecs, err := d.Embed(ctx, inputs)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 7 {
		t.Fatalf("got %d vectors, want 7", len(vecs))
	}
	t.Logf("batched 7 inputs → %d vectors, dim=%d", len(vecs), len(vecs[0].Values))
}
