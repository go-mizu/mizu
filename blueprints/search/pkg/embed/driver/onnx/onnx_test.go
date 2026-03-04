//go:build onnx

package onnx

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
)

func TestMeanPool(t *testing.T) {
	// batch=1, seq=3, dim=2
	hidden := []float32{
		1, 2, // token 0
		3, 4, // token 1
		5, 6, // token 2
	}
	mask := []int64{1, 1, 0} // only first 2 tokens

	result := meanPool(hidden, mask, 1, 3, 2)
	if len(result) != 1 {
		t.Fatalf("expected 1 vector, got %d", len(result))
	}

	// mean of [1,2] and [3,4] = [2, 3]
	want := []float32{2.0, 3.0}
	for i, v := range result[0] {
		if math.Abs(float64(v-want[i])) > 1e-6 {
			t.Errorf("result[0][%d] = %f, want %f", i, v, want[i])
		}
	}
}

func TestL2Normalize(t *testing.T) {
	vec := []float32{3, 4}
	l2Normalize(vec)

	// norm = 5, so [3/5, 4/5] = [0.6, 0.8]
	if math.Abs(float64(vec[0])-0.6) > 1e-6 {
		t.Errorf("vec[0] = %f, want 0.6", vec[0])
	}
	if math.Abs(float64(vec[1])-0.8) > 1e-6 {
		t.Errorf("vec[1] = %f, want 0.8", vec[1])
	}
}

func TestDriverIntegration(t *testing.T) {
	// Skip if ONNX Runtime is not available.
	if findONNXRuntimeLib() == "" {
		t.Skip("ONNX Runtime not found; set ONNXRUNTIME_LIB or install via: brew install onnxruntime")
	}

	home, _ := os.UserHomeDir()
	modelDir := filepath.Join(home, "data", "models", "onnx", "all-MiniLM-L6-v2")

	// Skip if model is not downloaded.
	if _, err := os.Stat(filepath.Join(modelDir, "model.onnx")); os.IsNotExist(err) {
		t.Skip("model not downloaded; run: EnsureModel() first or download manually")
	}

	ctx := context.Background()
	d := &Driver{}
	err := d.Open(ctx, embed.Config{Dir: modelDir})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	if d.Dimension() != 384 {
		t.Fatalf("Dimension() = %d, want 384", d.Dimension())
	}

	vecs, err := d.Embed(ctx, []embed.Input{
		{Text: "The quick brown fox jumps over the lazy dog."},
		{Text: "Machine learning is fascinating."},
	})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("got %d vectors, want 2", len(vecs))
	}
	for i, v := range vecs {
		if len(v.Values) != 384 {
			t.Errorf("vec[%d] has %d dims, want 384", i, len(v.Values))
		}
		// Check L2 normalized (norm ≈ 1).
		var norm float64
		for _, val := range v.Values {
			norm += float64(val) * float64(val)
		}
		norm = math.Sqrt(norm)
		if math.Abs(norm-1.0) > 0.01 {
			t.Errorf("vec[%d] norm = %f, want ~1.0", i, norm)
		}
	}
}
