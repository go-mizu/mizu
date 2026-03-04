# spec/0660: Embedding Pipeline

## Overview

Vector embedding pipeline for the CC search infrastructure. Two embedding
drivers — **llamacpp** (HTTP client to llama.cpp server) and **onnx** (local
ONNX Runtime inference) — implement the `embed.Driver` interface. A new CLI
command `search cc fts embed` reads markdown files, chunks text, generates
embeddings, and writes binary vector files.

## Package Structure

```
pkg/embed/
  embed.go              ← existing: Model/Input/Vector interfaces
  registry.go           ← Driver/Config + Register/New/List registry
  chunker.go            ← ChunkText(text, maxChars, overlap)
  models.go             ← ModelInfo registry, download, progress

pkg/embed/tokenizer/
  tokenizer.go          ← Pure Go BERT WordPiece tokenizer
  tokenizer_test.go

pkg/embed/driver/llamacpp/
  llamacpp.go           ← HTTP client to /v1/embeddings
  llamacpp_test.go      ← Tests with httptest mock server

pkg/embed/driver/onnx/           (build tag: onnx)
  onnx.go               ← ONNX Runtime driver + mean pooling
  download.go           ← Auto-download model from HuggingFace
  onnx_test.go

cli/
  cc_fts_embed.go       ← CLI: embed run/download/models + pipeline
  cc_fts_embed_onnx.go  ← ONNX driver import (build tag: onnx)
```

## Interfaces

### embed.Driver

Extends the existing `embed.Model` with lifecycle:

```go
type Config struct {
    Dir       string // model cache directory
    Addr      string // server address (llamacpp)
    Model     string // model name override
    BatchSize int    // max batch size
}

type Driver interface {
    Model
    Open(ctx context.Context, cfg Config) error
    Close() error
}
```

### Registry

Same pattern as `pkg/index/engine.go`:

```go
embed.Register("llamacpp", factory)
embed.New("llamacpp") → Driver
embed.List() → []string
```

## Driver: llamacpp

- Calls `/v1/embeddings` (OpenAI-compatible) on a llama.cpp server
- Default addr: `http://localhost:8086`
- Health check + probe dimension on Open() via `/health` + dummy embedding
- Batches inputs at configurable batch size (default 64)
- No external Go dependencies; pure HTTP client
- Server processes requests through n_parallel slots (not true batch inference)

## Driver: onnx

- Uses `github.com/yalue/onnxruntime_go` v1.24.0 for local inference
- Default model: `sentence-transformers/all-MiniLM-L6-v2` (384-dim, 128 max seq len)
- Auto-downloads model.onnx + vocab.txt from HuggingFace to `~/data/models/onnx/`
- Includes pure Go BERT WordPiece tokenizer
- Pipeline: tokenize → ONNX batch inference → mean pooling → L2 normalize
- True batch inference (single forward pass for entire batch)
- Build tag: `//go:build onnx` (requires ONNX Runtime shared library)
- ORT version mapping: Go binding v1.24.0 → requires ORT 1.24.x

## CLI: `search cc fts embed`

Three subcommands: `run`, `download`, `models`.

```
search cc fts embed run [flags]

Flags:
  --input          Input markdown directory (bypasses CC pipeline)
  --output         Output directory (default: auto)
  --crawl          Crawl ID (default: latest)
  --file           WARC file index or range (0-9)
  --driver         Embedding driver: llamacpp, onnx
  --addr           Server address (for llamacpp, default http://localhost:8086)
  --model          Model name (default: auto per driver)
  --batch-size     Inputs per embedding batch (default 64)
  --embed-workers  Concurrent embedding workers (default 4)
  --file-workers   Parallel file readers (default NumCPU)
  --max-chars      Max characters per text chunk (default 500)
  --overlap        Chunk overlap in characters (default 200)
  --model-dir      Model storage directory (default ~/data/models)
  --download       Download model before embedding if missing

search cc fts embed download [flags]   # download model files
search cc fts embed models  [flags]    # list models with status
```

### Pipeline Architecture (4-stage)

```
Stage 1: File readers (errgroup, N=file-workers)
    → chunkCh (chan chunkItem)

Stage 2: Batcher (single goroutine)
    → batchCh (chan []chunkItem, sized to batch-size)

Stage 3: Embed workers (errgroup, N=embed-workers)
    → resultCh (chan embedResult)
    Error recovery: if batch fails, retry one-by-one to isolate bad inputs

Stage 4: Writer (single goroutine, non-blocking)
    → vectors.bin + meta.jsonl (sequential I/O, never blocks embedders)
```

### Data Flow

```
~/data/common-crawl/{crawl}/markdown/{warcIdx}/**/*.md
    → read files (plain .md or .md.gz)
    → ChunkText(text, maxChars=500, overlap=200)
    → batch Embed via driver (parallel workers)
    → vectors.bin + meta.jsonl + stats.json
```

### Output Format

```
~/data/common-crawl/{crawl}/embed/{driver}/{warcIdx}/
  vectors.bin    N × dim × 4 bytes (raw float32, little-endian)
  meta.jsonl     one JSON line per vector: {id, file, chunk_idx, text_len, dim}
  stats.json     {files, chunks, vectors, errors, dim, driver, embed_workers,
                  batch_size, elapsed_ms, vec_per_sec}
```

- `vectors.bin` is a flat concatenation of float32 vectors. Vector i starts at
  byte offset `i × dim × 4`.
- `meta.jsonl` line i corresponds to vector i. The `id` field is `filename:chunk_idx`.

## Chunking

`embed.ChunkText(text, maxChars, overlap)`:

- Split on `\n\n` paragraph boundaries
- Hard-split paragraphs exceeding maxChars
- Adjacent chunks overlap by `overlap` characters for context continuity
- Default: maxChars=500, overlap=200
- **KEY LESSON**: maxChars=2000 overflows 512-token model context; 500 chars is safe

## Model Registry

`pkg/embed/models.go` provides model discovery, download, and status checking:

| Driver   | Model                    | Dim | Size  | Description                              |
|----------|--------------------------|-----|-------|------------------------------------------|
| llamacpp | nomic-embed-text-v1.5    | 768 | 137MB | Best quality, 8K context                 |
| llamacpp | bge-small-en-v1.5        | 384 | 67MB  | Fast, compact                            |
| llamacpp | all-MiniLM-L6-v2         | 384 | 46MB  | Smallest, fastest (default for benchmarks)|
| onnx     | all-MiniLM-L6-v2         | 384 | 90MB  | CPU-optimized, default ONNX model        |

## Docker

`docker/llamacpp/docker-compose.yml` — `llamacpp-embed` service:

```yaml
llamacpp-embed:
  image: ghcr.io/ggml-org/llama.cpp:server
  ports: ["8086:8080"]
  command: >
    --model /models/nomic-embed-text-v1.5.Q8_0.gguf
    --embedding --pooling mean --ctx-size 2048 --cont-batching
```

## Server Setup

### llama-server (native, recommended)

```bash
# Download pre-built binary
curl -fsSL "https://github.com/ggml-org/llama.cpp/releases/latest/..." -o llama.tar.gz

# Start with Metal GPU (macOS)
llama-server --model ~/data/models/all-MiniLM-L6-v2.Q8_0.gguf \
  --embedding --pooling mean --port 8086 --cont-batching -ngl 99

# Start CPU-only (Linux)
llama-server --model ~/data/models/all-MiniLM-L6-v2.Q8_0.gguf \
  --embedding --pooling mean --port 8086 --cont-batching --threads $(nproc)
```

### ONNX Runtime Install

```bash
# macOS
brew install onnxruntime  # v1.24.2

# Linux (Ubuntu 24.04)
ONNX_VERSION=1.23.0  # latest available for Linux
curl -fsSL "https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-linux-x64-${ONNX_VERSION}.tgz" -o /tmp/onnxruntime.tgz
tar -xzf /tmp/onnxruntime.tgz -C /tmp/
cp /tmp/onnxruntime-linux-x64-${ONNX_VERSION}/lib/libonnxruntime.so.${ONNX_VERSION} /usr/local/lib/
ln -sf /usr/local/lib/libonnxruntime.so.${ONNX_VERSION} /usr/local/lib/libonnxruntime.so
ldconfig
```

**KEY LESSON (ORT version mismatch)**: Go binding version must match ORT major.minor.
`yalue/onnxruntime_go v1.24.0` requires ORT 1.24.x (API v24). ORT 1.23.0 (API v23) fails
with "requested API version [24] is not available". Mac Homebrew has ORT 1.24.2; Linux
only has ORT 1.23.0 as of 2026-03. Use Go binding v1.23.0 for Linux, v1.24.0 for Mac —
or wait for ORT 1.24 Linux release.

**Current solution**: Go binding v1.24.0 + ORT 1.24.2 (Mac Homebrew) works. For Linux,
install ORT 1.24 from pre-release or build from source, or downgrade Go binding to v1.23.0.

## Build & Test

```bash
# Test llamacpp driver (always works)
go test ./pkg/embed/driver/llamacpp/...

# Test tokenizer (always works)
go test ./pkg/embed/tokenizer/...

# Test ONNX driver (needs: brew install onnxruntime)
go test -tags onnx ./pkg/embed/driver/onnx/...

# Build without ONNX
GOWORK=off go build ./cmd/search/

# Build with ONNX support
GOWORK=off go build -tags onnx ./cmd/search/

# Deploy to server2 (native build, no QEMU)
make build-on-server SERVER=2
# Then SSH and rebuild with -tags onnx:
ssh root@server2 'cd ~/.search-build/... && go build -tags onnx ...'
```

## Benchmarks

### Test Data

- WARC 00000 from CC-MAIN-2026-08
- Markdown files: 148 (subdir 00/) or 12,845-21,184 (full WARC 00000)
- Model: all-MiniLM-L6-v2 (384-dim), max-chars=500, batch-size=64

### Mac M4 (Apple Silicon, 10-core CPU, 10-core GPU)

148 files, 2,262 chunks:

| Driver   | GPU   | embed-workers | elapsed | vec/s | notes                    |
|----------|-------|:---:|---------|:---:|-----------------------------------|
| llamacpp | CPU   | 1   | 6.3s    | 358 | **fastest** — M4 CPU very strong  |
| llamacpp | CPU   | 2   | 7.5s    | 303 | contention, same server           |
| llamacpp | CPU   | 4   | 7.2s    | 314 | contention, same server           |
| llamacpp | CPU   | 8   | 8.3s    | 272 | worst — too much contention       |
| llamacpp | Metal | 1   | 7.1s    | 317 | GPU overhead > benefit for 23MB model |
| llamacpp | Metal | 2   | 8.2s    | 276 | Metal + contention                |
| llamacpp | Metal | 4   | 8.7s    | 261 |                                   |
| onnx     | CPU   | 1   | 31.5s   | 72  | true batch inference              |
| onnx     | CPU   | 2   | 19.4s   | 117 | **scales well** — parallel sessions |
| onnx     | CPU   | 4   | 21.1s   | 107 | diminishing returns               |

**KEY LESSON (Metal vs CPU for small models)**: Metal GPU provides no speedup for the
23MB all-MiniLM-L6-v2 model — data transfer overhead exceeds compute savings.
CPU-only llamacpp at 358 vec/s is the fastest option on M4.

**KEY LESSON (ONNX multi-worker scaling)**: ONNX creates independent inference sessions
per embed-worker, achieving true parallel CPU inference. 2 workers = 1.6x speedup (117 vs 72).
4 workers shows diminishing returns due to CPU core contention.

**KEY LESSON (llamacpp vs ONNX)**: llamacpp processes through n_parallel slots (default 4),
not true batch forward pass. ONNX constructs a single [batch, seq_len, dim] tensor per call.
For this model, llamacpp is still 5x faster (358 vs 72) because the HTTP server's compiled
C++ is faster than ONNX Runtime's generic inference engine.

### Server2 (AMD EPYC, 6 vCPUs, 12GB RAM, Ubuntu 24.04)

82 files, 1,475 chunks (subdir 00/):

| Driver   | Mode    | embed-workers | elapsed | vec/s | notes                 |
|----------|---------|:---:|---------|:---:|-------------------------------|
| llamacpp | Docker  | 1   | 60.5s   | 24  | Docker container, n_parallel=4 |
| llamacpp | Docker  | 2   | 62.5s   | 24  | no improvement                |
| llamacpp | Docker  | 4   | 58.8s   | 25  | marginal                      |
| llamacpp | native  | 1   | 66.6s   | 22  | native binary, same perf      |
| llamacpp | native  | 1   | 92.3s   | 16  | batch=128 — **slower**        |
| onnx     | native  | 1   | 68.9s   | 21  | true batch, comparable        |

Full WARC 00000 (21,184 files, in progress):

| Driver   | vectors | elapsed   | vec/s | projected total |
|----------|---------|-----------|:---:|-----------------|
| onnx     | 29,632+ | ~25min    | 19-20 | ~35K vecs, ~30min |
| llamacpp | 5,632+  | ~25min    | ~5    | ~35K vecs, ~3hrs |

**KEY LESSON (Docker vs native for llamacpp)**: Zero performance difference — the
bottleneck is CPU compute, not container overhead.

**KEY LESSON (batch size)**: batch=128 is **slower** than batch=64 (16 vs 22 vec/s)
because larger batches increase per-batch token processing overhead in the server.

**KEY LESSON (ONNX dominates on server2)**: ONNX true batch inference is 4x faster
than llamacpp slot-based processing on the same CPU (19 vs 5 vec/s for full run).
llamacpp's n_parallel=4 slot system adds overhead for embedding workloads.

### Summary

| Environment | Best Driver | Best Config | vec/s |
|-------------|------------|-------------|:---:|
| Mac M4      | llamacpp (CPU) | embed-workers=1, batch=64 | **358** |
| Mac M4      | onnx       | embed-workers=2 | **117** |
| Server2     | onnx       | embed-workers=1, batch=64 | **19-20** |
| Server2     | llamacpp   | Docker, embed-workers=1 | **24** (small), **5** (full) |

## Future Work

- **CoreML EP for ONNX**: Build ORT from source with `--use_coreml` for ANE acceleration on Mac
- **llama.cpp CGO binding**: Eliminate HTTP overhead for in-process embedding
- Vector store integration (pkg/vector drivers: DuckDB vss, sqlite-vec, hnswlib)
- Hybrid search: FTS score + vector similarity reranking
- Incremental embedding (skip already-embedded files via meta.jsonl)
