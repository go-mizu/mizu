# spec/0664: Rust Embedder — Three-Way Benchmark

## Goal

Replace the Go embedding pipeline (358 vec/s llamacpp, 117 vec/s ONNX on M4) with
a Rust embedder targeting **18,000+ vec/s on M4 Metal**. Benchmark three approaches,
pick the winner, then build full HTTP server + CLI.

## Current Baseline (Go)

| Environment | Driver | Throughput |
|-------------|--------|------------|
| M4 Mac | llamacpp (CPU) | 358 vec/s |
| M4 Mac | ONNX | 117 vec/s |
| Server2 (EPYC 6 vCPU) | ONNX | 20 vec/s |
| Server2 | llamacpp (Docker) | 5-24 vec/s |

## Three Approaches

### 1. Pure Candle (`candle-core` + `candle-transformers`)

- Native Metal GPU kernels, Accelerate (AMX) for CPU
- BERT implementation in `candle-transformers::models::bert`
- Zero-copy safetensors via mmap, HF Hub model download
- Tokenization via `tokenizers` crate (HuggingFace Rust tokenizer)
- **Expected**: 18K-22K vec/s on M4 Metal (per metal-candle benchmarks)

### 2. metal-candle wrapper

- Pre-built embedding API: `EmbeddingModel::from_pretrained().encode()`
- Wraps Candle with pooling + normalization built-in
- Proven 22K docs/s on Apple Silicon
- **Risk**: third-party crate, Apple-only focus

### 3. fastembed-rs (ONNX Runtime backend)

- 30+ models out-of-box, battle-tested
- ONNX Runtime linked (~50MB), CPU only on Mac
- **Expected**: ~400 vec/s CPU (no GPU acceleration)
- **Risk**: can't achieve 50x target without GPU

## Benchmark Protocol

- **Model**: sentence-transformers/all-MiniLM-L6-v2 (384-dim, 22MB)
- **Corpus**: 10,000 synthetic chunks (500 chars each)
- **Batch sizes**: 1, 16, 32, 64, 128, 256
- **Warmup**: 100 vectors discarded
- **Metrics**: vec/s, P50/P99 batch latency (ms), peak RSS (MB)

### M4 Mac test matrix

| Approach | CPU | Metal | Accelerate |
|----------|-----|-------|------------|
| candle | ✓ | ✓ | ✓ |
| metal-candle | ✓ | ✓ | — |
| fastembed | ✓ | — | — |

### Server2 test matrix

| Approach | CPU (pure Rust) | CPU (OpenBLAS) |
|----------|-----------------|----------------|
| candle | ✓ | ✓ |
| fastembed | ✓ | — |

## Directory Structure

```
tools/embedder/
├── candle/              # Approach 1: Pure Candle
│   ├── Cargo.toml
│   └── src/main.rs
├── metalcandle/         # Approach 2: metal-candle
│   ├── Cargo.toml
│   └── src/main.rs
├── fastembed/           # Approach 3: fastembed-rs
│   ├── Cargo.toml
│   └── src/main.rs
├── bench.sh             # Run all benchmarks, collect results
└── testdata/
    └── chunks.jsonl     # 10K test chunks
```

## Benchmark CLI Interface

Each binary implements the same interface:

```bash
# Benchmark mode
embedder bench --input chunks.jsonl --batch-size 64

# Output (JSON):
{
  "approach": "candle",
  "backend": "metal",
  "model": "all-MiniLM-L6-v2",
  "batch_size": 64,
  "total_vecs": 10000,
  "vecs_per_sec": 18500.0,
  "p50_ms": 3.4,
  "p99_ms": 4.1,
  "peak_rss_mb": 142,
  "elapsed_sec": 0.54
}
```

## After Benchmark: Winner Gets Full Build

The winning approach gets:

1. **HTTP server** — Axum, OpenAI-compatible `/v1/embeddings` endpoint
   - Drop-in replacement for llamacpp driver in Go pipeline
   - Batching, health check, model info endpoints
2. **CLI mode** — `embedder embed` reads stdin JSONL, writes binary vectors
3. **Go driver** — New `embed.Driver` registered as "candle" in Go
4. **Cross-compile** — Docker build for Linux (server2), native for macOS
5. **Feature flags** — `--features metal,accelerate,cuda,openblas`

## Benchmark Results (March 2026)

### M4 Mac — All Approaches (10K real Go/markdown chunks)

| Approach | Backend | Batch | vec/s | P50 ms | RSS MB |
|----------|---------|-------|-------|--------|--------|
| **ORT (Rust)** | **CPU** | **16** | **146** | **116** | **366** |
| ORT (Rust) | CoreML | 16 | 55 | — | 2,871 |
| Candle 0.9.2 | Accelerate | 32 | 60 | 1,902 | 535 |
| fastembed 5.x | ONNX CPU | 64 | 52 | 1,160 | 2,850 |
| Go llamacpp | GGML CPU | 64 | **358** | — | — |

### Key Lessons

1. **Candle Metal is broken** for BERT (no LayerNorm kernel, issue #2832)
2. **CoreML adds massive overhead** for small models (22M params) — XPC dispatch dominates
3. **ORT CPU is the best Rust approach** (146 vec/s, 366 MB RSS)
4. **llamacpp still wins overall** because GGML quantization (Q4/Q8) reduces matmul 4-8x
5. **18K+ target is unrealistic** without working Metal GPU or quantized inference
6. **Batch=16 optimal for ORT** — less padding waste than larger batches

### Conclusion

The Rust ORT embedder (146 vec/s) beats candle (60), fastembed (52), and Go ONNX (117),
but cannot match Go llamacpp (358 vec/s) because llamacpp uses quantized GGML weights.

**Next steps to close the gap:**
- Use ONNX quantized model (model_qint8_arm64.onnx, 23 MB) with ORT
- Explore GGUF/GGML loading from Rust (llama.cpp C FFI)
- Wait for candle Metal LayerNorm support
- Multi-session parallelism (N ORT sessions on N threads)

## Implementation Order

1. ~~Write spec (this file)~~
2. ~~Build all benchmark binaries~~
3. ~~Generate test corpus~~
4. ~~Benchmark on M4 (all backends)~~
5. ~~Pick winner~~ → ORT CPU
6. Build full HTTP server + CLI for ORT
7. Cross-compile and benchmark on server2
8. Integrate as Go embedding driver
