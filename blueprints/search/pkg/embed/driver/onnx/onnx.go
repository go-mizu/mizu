//go:build onnx

// Package onnx implements an embed.Driver using ONNX Runtime for local inference.
//
// It loads a sentence-transformer ONNX model (default: all-MiniLM-L6-v2) and
// runs inference using the yalue/onnxruntime_go bindings. The ONNX Runtime
// shared library must be installed on the system.
//
// Build with: go build -tags onnx
package onnx

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/go-mizu/mizu/blueprints/search/pkg/embed"
	"github.com/go-mizu/mizu/blueprints/search/pkg/embed/tokenizer"
)

const (
	defaultModelName = "all-MiniLM-L6-v2"
	defaultDim       = 384
	defaultMaxSeqLen = 128
	defaultBatchSize = 32
)

func init() {
	embed.Register("onnx", func() embed.Driver { return &Driver{} })
}

// Driver runs ONNX model inference locally.
type Driver struct {
	tok       *tokenizer.Tokenizer
	modelPath string
	dim       int
	maxSeqLen int
	batchSize int
	modelName string
}

func (d *Driver) Name() string {
	if d.modelName != "" {
		return "onnx/" + d.modelName
	}
	return "onnx"
}

func (d *Driver) Dimension() int { return d.dim }

// Open initializes the ONNX runtime, downloads the model if needed, and
// creates the inference session.
func (d *Driver) Open(ctx context.Context, cfg embed.Config) error {
	d.dim = defaultDim
	d.maxSeqLen = defaultMaxSeqLen
	d.batchSize = cfg.BatchSize
	if d.batchSize <= 0 {
		d.batchSize = defaultBatchSize
	}
	d.modelName = cfg.Model
	if d.modelName == "" {
		d.modelName = defaultModelName
	}

	// Determine model directory.
	dir := cfg.Dir
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "data", "models", "onnx", d.modelName)
	}

	// Download model + vocab if needed.
	modelPath, vocabPath, err := EnsureModel(dir)
	if err != nil {
		return err
	}
	d.modelPath = modelPath

	// Initialize tokenizer.
	d.tok, err = tokenizer.New(vocabPath, d.maxSeqLen)
	if err != nil {
		return fmt.Errorf("onnx: tokenizer: %w", err)
	}

	// Initialize ONNX Runtime.
	libPath := findONNXRuntimeLib()
	if libPath == "" {
		return fmt.Errorf("onnx: ONNX Runtime shared library not found. Install with: brew install onnxruntime")
	}
	ort.SetSharedLibraryPath(libPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("onnx: init runtime: %w", err)
	}

	return nil
}

func (d *Driver) Close() error {
	if err := ort.DestroyEnvironment(); err != nil {
		return fmt.Errorf("onnx: destroy env: %w", err)
	}
	return nil
}

// Embed generates embeddings using the ONNX model.
func (d *Driver) Embed(ctx context.Context, inputs []embed.Input) ([]embed.Vector, error) {
	texts := make([]string, len(inputs))
	for i, inp := range inputs {
		texts[i] = inp.Text
	}

	var result []embed.Vector
	for i := 0; i < len(texts); i += d.batchSize {
		end := i + d.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := d.embedBatch(texts[i:end])
		if err != nil {
			return nil, err
		}
		result = append(result, vecs...)
	}
	return result, nil
}

func (d *Driver) embedBatch(texts []string) ([]embed.Vector, error) {
	batchSize := int64(len(texts))
	seqLen := int64(d.maxSeqLen)

	// Tokenize.
	encoded := d.tok.EncodeBatch(texts)

	// Flatten into contiguous arrays.
	inputIDs := make([]int64, batchSize*seqLen)
	attentionMask := make([]int64, batchSize*seqLen)
	tokenTypeIDs := make([]int64, batchSize*seqLen)

	for b, enc := range encoded {
		off := int64(b) * seqLen
		copy(inputIDs[off:off+seqLen], enc.InputIDs)
		copy(attentionMask[off:off+seqLen], enc.AttentionMask)
		copy(tokenTypeIDs[off:off+seqLen], enc.TokenTypeIDs)
	}

	shape := ort.NewShape(batchSize, seqLen)

	inputIDsTensor, err := ort.NewTensor(shape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("onnx: create input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	attMaskTensor, err := ort.NewTensor(shape, attentionMask)
	if err != nil {
		return nil, fmt.Errorf("onnx: create attention_mask tensor: %w", err)
	}
	defer attMaskTensor.Destroy()

	typeIDsTensor, err := ort.NewTensor(shape, tokenTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("onnx: create token_type_ids tensor: %w", err)
	}
	defer typeIDsTensor.Destroy()

	// Output tensor: [batch, seq_len, dim]
	outputShape := ort.NewShape(batchSize, seqLen, int64(d.dim))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, fmt.Errorf("onnx: create output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	// Create session and run.
	session, err := ort.NewAdvancedSession(
		d.modelPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		[]ort.ArbitraryTensor{inputIDsTensor, attMaskTensor, typeIDsTensor},
		[]ort.ArbitraryTensor{outputTensor},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("onnx: create session: %w", err)
	}
	defer session.Destroy()

	if err := session.Run(); err != nil {
		return nil, fmt.Errorf("onnx: run: %w", err)
	}

	// Mean pooling + L2 normalization.
	hidden := outputTensor.GetData()
	vecs := meanPool(hidden, attentionMask, int(batchSize), int(seqLen), d.dim)

	result := make([]embed.Vector, len(vecs))
	for i, v := range vecs {
		l2Normalize(v)
		result[i] = embed.Vector{Values: v}
	}
	return result, nil
}

// meanPool applies attention-masked mean pooling over the sequence dimension.
func meanPool(hidden []float32, mask []int64, batchSize, seqLen, dim int) [][]float32 {
	result := make([][]float32, batchSize)
	for b := 0; b < batchSize; b++ {
		vec := make([]float32, dim)
		var count float32
		for s := 0; s < seqLen; s++ {
			if mask[b*seqLen+s] == 0 {
				continue
			}
			count++
			off := b*seqLen*dim + s*dim
			for d := 0; d < dim; d++ {
				vec[d] += hidden[off+d]
			}
		}
		if count > 0 {
			for d := 0; d < dim; d++ {
				vec[d] /= count
			}
		}
		result[b] = vec
	}
	return result
}

// l2Normalize normalizes a vector to unit length.
func l2Normalize(vec []float32) {
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		invNorm := float32(1.0 / norm)
		for i := range vec {
			vec[i] *= invNorm
		}
	}
}

// findONNXRuntimeLib searches common locations for the ONNX Runtime shared library.
func findONNXRuntimeLib() string {
	candidates := []string{
		// macOS Homebrew
		"/opt/homebrew/lib/libonnxruntime.dylib",
		"/usr/local/lib/libonnxruntime.dylib",
		// Linux
		"/usr/lib/libonnxruntime.so",
		"/usr/local/lib/libonnxruntime.so",
		"/usr/lib/x86_64-linux-gnu/libonnxruntime.so",
	}

	// Check env override first.
	if p := os.Getenv("ONNXRUNTIME_LIB"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	arch := runtime.GOARCH
	if arch == "arm64" && runtime.GOOS == "darwin" {
		// Homebrew on Apple Silicon.
		candidates = append([]string{"/opt/homebrew/lib/libonnxruntime.dylib"}, candidates...)
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
