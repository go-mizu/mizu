//go:build onnx

// Package onnx implements an embed.Driver using ONNX Runtime for local inference.
//
// It loads a sentence-transformer ONNX model (default: all-MiniLM-L6-v2) and
// runs inference using the yalue/onnxruntime_go bindings. The ONNX Runtime
// shared library must be installed on the system.
//
// Supports CoreML execution provider on macOS for GPU/ANE acceleration.
// Enable via Config.Addr = "coreml".
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
	"strings"
	"sync"

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
	coreml    bool

	// Mutex protects session — ONNX Runtime session.Run() is not thread-safe.
	mu sync.Mutex

	// Reusable session components (created once in Open, reused in Embed).
	session      *ort.AdvancedSession
	inputIDs     *ort.Tensor[int64]
	attMask      *ort.Tensor[int64]
	typeIDs      *ort.Tensor[int64]
	outputTensor *ort.Tensor[float32]
	sessionBatch int // batch size the session was created for
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
//
// Set cfg.Addr = "coreml" to enable CoreML execution provider on macOS.
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
	d.coreml = strings.EqualFold(cfg.Addr, "coreml")

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

	// Create a reusable session for the default batch size.
	if err := d.createSession(d.batchSize); err != nil {
		return fmt.Errorf("onnx: create session: %w", err)
	}

	return nil
}

// createSession creates or recreates the ONNX session for the given batch size.
func (d *Driver) createSession(batchSize int) error {
	// Destroy previous session if any.
	d.destroySession()

	seqLen := int64(d.maxSeqLen)
	bs := int64(batchSize)
	shape := ort.NewShape(bs, seqLen)

	var err error
	d.inputIDs, err = ort.NewEmptyTensor[int64](shape)
	if err != nil {
		return fmt.Errorf("create input_ids tensor: %w", err)
	}
	d.attMask, err = ort.NewEmptyTensor[int64](shape)
	if err != nil {
		return fmt.Errorf("create attention_mask tensor: %w", err)
	}
	d.typeIDs, err = ort.NewEmptyTensor[int64](shape)
	if err != nil {
		return fmt.Errorf("create token_type_ids tensor: %w", err)
	}

	outputShape := ort.NewShape(bs, seqLen, int64(d.dim))
	d.outputTensor, err = ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return fmt.Errorf("create output tensor: %w", err)
	}

	// Session options.
	var opts *ort.SessionOptions
	if d.coreml {
		opts, err = ort.NewSessionOptions()
		if err != nil {
			return fmt.Errorf("create session options: %w", err)
		}
		defer opts.Destroy()
		if err := opts.AppendExecutionProviderCoreMLV2(map[string]string{
			"EnableOnSubgraphs": "1",
		}); err != nil {
			return fmt.Errorf("append CoreML EP: %w", err)
		}
	}

	d.session, err = ort.NewAdvancedSession(
		d.modelPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		[]ort.ArbitraryTensor{d.inputIDs, d.attMask, d.typeIDs},
		[]ort.ArbitraryTensor{d.outputTensor},
		opts,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	d.sessionBatch = batchSize
	return nil
}

func (d *Driver) destroySession() {
	if d.session != nil {
		d.session.Destroy()
		d.session = nil
	}
	if d.inputIDs != nil {
		d.inputIDs.Destroy()
		d.inputIDs = nil
	}
	if d.attMask != nil {
		d.attMask.Destroy()
		d.attMask = nil
	}
	if d.typeIDs != nil {
		d.typeIDs.Destroy()
		d.typeIDs = nil
	}
	if d.outputTensor != nil {
		d.outputTensor.Destroy()
		d.outputTensor = nil
	}
}

func (d *Driver) Close() error {
	d.destroySession()
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
	d.mu.Lock()
	defer d.mu.Unlock()

	actualBatch := len(texts)
	seqLen := d.maxSeqLen

	// Pad texts to session batch size (avoid resizing session which is not thread-safe).
	paddedTexts := texts
	if actualBatch < d.sessionBatch {
		paddedTexts = make([]string, d.sessionBatch)
		copy(paddedTexts, texts)
		// Remaining entries are empty strings — tokenizer handles them fine.
	} else if actualBatch > d.sessionBatch {
		// Should not happen if batchSize is set correctly in Embed(), but handle gracefully.
		return nil, fmt.Errorf("onnx: batch %d exceeds session capacity %d", actualBatch, d.sessionBatch)
	}

	// Tokenize.
	encoded := d.tok.EncodeBatch(paddedTexts)

	// Fill pre-allocated tensor data.
	ids := d.inputIDs.GetData()
	mask := d.attMask.GetData()
	types := d.typeIDs.GetData()

	for b, enc := range encoded {
		off := b * seqLen
		copy(ids[off:off+seqLen], enc.InputIDs)
		copy(mask[off:off+seqLen], enc.AttentionMask)
		copy(types[off:off+seqLen], enc.TokenTypeIDs)
	}

	if err := d.session.Run(); err != nil {
		return nil, fmt.Errorf("onnx: run: %w", err)
	}

	// Mean pooling + L2 normalization — only for actual (non-padded) inputs.
	hidden := d.outputTensor.GetData()
	maskData := d.attMask.GetData()
	vecs := meanPool(hidden, maskData, actualBatch, seqLen, d.dim)

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
