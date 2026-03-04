# spec/0663: Mac-Optimized Embedding

## Overview

Research and benchmarks for computing text embeddings on Apple Silicon M4,
evaluating six approaches for maximum throughput from Go.

## Hardware

- Apple M4 (10-core CPU, 10-core GPU, 16-core Neural Engine)
- Neural Engine: ~19 TFLOPS FP16 at 2.8W (~80x efficiency/FLOP vs A100)
- Unified memory: CPU, GPU, and ANE share the same memory pool (zero-copy)

## Approaches Evaluated

### 1. llama.cpp (HTTP server) — IMPLEMENTED, FASTEST

**How it works**: Run `llama-server` with `--embedding --pooling mean`. Our existing
llamacpp driver calls `/v1/embeddings`. With `-ngl 99`, all layers offloaded to Metal GPU.

**Benchmark (M4, all-MiniLM-L6-v2, 148 files, 2,262 chunks)**:
- CPU-only, w=1: **358 vec/s** (6.3s) ← best
- Metal GPU, w=1: 317 vec/s (7.1s) — no improvement for 23MB model

**Verdict**: Best option today. CPU beats GPU for small models (data transfer > compute).
For larger models (nomic-embed-text-v1.5 at 137MB), Metal should show gains.

### 2. ONNX Runtime (CPU) — IMPLEMENTED

**How it works**: `yalue/onnxruntime_go` runs true batch inference locally. Install via
`brew install onnxruntime` (v1.24.2 on Mac). Build with `-tags onnx`.

**Benchmark (M4, all-MiniLM-L6-v2, 148 files, 2,262 chunks)**:
- w=1: 72 vec/s (31.5s)
- w=2: **117 vec/s** (19.4s) — 1.6x scaling, parallel sessions
- w=4: 107 vec/s (21.1s) — diminishing returns

**Verdict**: Slower than llamacpp but no server dependency. Good for offline/CI use.
Multi-worker mode scales well (independent ORT sessions).

### 3. ONNX Runtime + CoreML EP — NOT YET IMPLEMENTED

**How it works**: ONNX Runtime's CoreML execution provider delegates ops to CoreML,
which dispatches to ANE + GPU + CPU.

**Go API** (already supported in `yalue/onnxruntime_go`):
```go
options, _ := ort.NewSessionOptions()
options.AppendExecutionProviderCoreMLV2(map[string]string{
    "MLComputeUnits": "ALL",       // ANE + GPU + CPU
    "ModelFormat":    "MLProgram", // supports LayerNorm, Gelu
})
session, _ := ort.NewAdvancedSession("model.onnx", ..., options)
```

**Blocker**: The official `onnxruntime-osx-arm64` from GitHub releases is **CPU-only**.
CoreML EP is NOT included. Must build ORT from source with `--use_coreml` to get a
`libonnxruntime.dylib` that includes the CoreML provider.

**Expected**: 2-5x over CPU-only ONNX (144-360 vec/s). Would match or beat llamacpp.

**Implementation plan** (driver/mac or onnx CoreML path):
1. Build ORT from source: `./build.sh --config Release --use_coreml --parallel`
2. Replace Homebrew dylib: `cp build/Release/libonnxruntime.dylib /opt/homebrew/lib/`
3. Add CoreML session options in onnx.go `Open()` when on darwin
4. Benchmark with `MLComputeUnits: ALL` vs `CPUAndNeuralEngine` vs `CPUOnly`

### 4. Core ML (native Obj-C bridge) — COMPLEX, DEFERRED

**How it works**: Convert PyTorch model to `.mlmodelc` via `coremltools`, load in Obj-C,
call from Go via CGO.

**Go bindings**: `gomlx/go-coreml` exists but is alpha — model builder only, not a
pre-trained model loader. Would need custom Obj-C wrapper (~100 lines).

**Expected**: 3-10x for ANE-optimized models. Apple claims 10x for distilBERT on ANE.

**Verdict**: High complexity (Python conversion + Obj-C bridge). CoreML EP via ONNX Runtime
is a better path — same ANE target, much simpler integration.

### 5. MLX — IMPRACTICAL FROM GO

Apple's ML framework for research. Python-first; Go bindings (`luxfi/mlx`) are
low-level tensor ops only — no model loading, no transformer layers.

**Verdict**: Not viable for Go. Would require Python subprocess or HTTP server.

### 6. Candle (Rust + Metal) — HIGH EFFORT

Hugging Face Rust ML framework with Metal backend. `metal-candle` claims 22K docs/sec
for embeddings on Apple Silicon. Go access via CGO + Rust FFI.

**Verdict**: High effort (Rust toolchain + C-ABI wrapper). Performance claims need
independent verification. Not worth the complexity vs. llamacpp or CoreML EP.

## Recommendation

### Short term (done)

Use **llamacpp CPU** (358 vec/s on M4, zero build complexity). Our existing driver
already achieves the best throughput without any Mac-specific code.

### Medium term (next step)

Build **ONNX Runtime with CoreML EP** from source. This is the only option that targets
the Neural Engine for maximum efficiency. Steps:

```bash
# 1. Clone and build ORT with CoreML
git clone --recursive https://github.com/microsoft/onnxruntime
cd onnxruntime
./build.sh --config Release --use_coreml --parallel --build_shared_lib

# 2. Install
cp build/Release/libonnxruntime.1.24.0.dylib /opt/homebrew/lib/
ln -sf libonnxruntime.1.24.0.dylib /opt/homebrew/lib/libonnxruntime.dylib

# 3. Test (no code changes needed — driver detects CoreML at runtime)
go test -tags onnx ./pkg/embed/driver/onnx/...
```

Then add CoreML session options to `pkg/embed/driver/onnx/onnx.go`:

```go
func (d *Driver) Open(ctx context.Context, cfg embed.Config) error {
    // ... existing init ...

    opts, _ := ort.NewSessionOptions()
    if runtime.GOOS == "darwin" {
        opts.AppendExecutionProviderCoreMLV2(map[string]string{
            "MLComputeUnits": "ALL",
        })
    }
    // ... create session with opts ...
}
```

### Long term

- Profile ANE utilization with Instruments.app
- Evaluate nomic-embed-text-v1.5 (768-dim) where GPU/ANE speedup is more significant
- Consider CGO binding to llama.cpp for in-process embedding (no HTTP server)

## Key Findings

1. **M4 CPU is remarkably fast** for small embedding models. llamacpp at 358 vec/s
   means 2,262 vectors in 6.3s — fast enough for most use cases without GPU.

2. **Metal GPU doesn't help for 23MB models**. The data transfer overhead exceeds
   compute savings. GPU/ANE acceleration matters for larger models (>100MB).

3. **ONNX multi-worker scales linearly** on M4 thanks to independent inference
   sessions. 2 workers = 1.6x, approaching CPU core count as upper bound.

4. **CoreML EP is the path to ANE** — simpler than native CoreML (no Python
   conversion, no Obj-C bridge) and `onnxruntime_go` already has the API.
