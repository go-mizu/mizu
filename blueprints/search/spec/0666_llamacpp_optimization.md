# spec/0666: llamacpp Embedding Throughput Optimization

## Goal

Optimize the Go llamacpp embedding pipeline from 358 vec/s baseline toward
maximum throughput on M4 Mac with all-MiniLM-L6-v2 Q8_0.

## Results

### Pipeline Throughput (100 files, 2783 chunks → 5181 chunks with 256-char)

| Config | vec/s | Errors | Notes |
|--------|-------|--------|-------|
| Baseline (pre-optimization) | 358 | many | 500-char, parallel=1, no flash-attn |
| +server flags, +conn pool | 280 | 46 | parallel=16, ctx=4096 (256 tok/slot!) |
| +ctx fix (8192, 512 tok/slot) | 226 | 0 | Zero errors but slower per-embedding |
| +256-char chunks, batch=16 | **576** | 0 | **Best 100-file result** |
| 500 files (46K chunks, sustained) | **466** | 0 | Sustained over 99 seconds |

### Raw Server Ceiling (no pipeline overhead)

| Text Length | vec/s | Config |
|-------------|-------|--------|
| ~250 chars | 794 | 8 workers, batch=16 |
| ~500 chars | 375 | 8 workers, batch=16 |
| Short text, 8 concurrent curl | 1,442 | Lower bound |

### Multi-Server (shared M4 GPU)

| Servers | vec/s | Scaling |
|---------|-------|---------|
| 1 | 375 | baseline |
| 2 | ~300 | 0.8x (GPU contention) |
| 3 | 594 | 1.6x (from earlier session) |

Multi-server on shared Metal GPU shows diminishing returns due to GPU contention.

### Quantization Comparison (Q8 vs Q4)

| Quant | Short text | Long text | Notes |
|-------|-----------|-----------|-------|
| Q8_0 (23 MB) | 794 | 375 | Better for long text |
| Q4_K_M (20 MB) | 870 | 338 | ~10% faster short, slower long |

Model already fits L2 cache at 23MB; Q4 saves little compute for 22M params.

## Root Causes Found

### 1. Per-Slot Context Too Small (Critical)

`--ctx-size 4096 / --parallel 16 = 256 tokens per slot`

500-char chunks tokenize to 258-310 tokens → HTTP 400 errors:
```
"input (260 tokens) is larger than the max context size (256 tokens)"
```

Each batch error triggers 16 individual retry requests, blocking an embed
worker. Fix: `--ctx-size 8192` gives 512 tokens/slot (model max).

### 2. HTTP Connection Pool Starvation

Default `http.DefaultTransport` has `MaxIdleConnsPerHost: 2`. With 8 embed
workers making concurrent requests, connections weren't reused.

Fix: Custom transport with `MaxIdleConnsPerHost: 64`.

### 3. Chunk Size Dominates Throughput

Shorter chunks = fewer tokens = less compute per embedding:
- 500-char → ~150-200 tokens → 375 raw vec/s
- 250-char → ~80-100 tokens → 794 raw vec/s (2.1x)

Trade-off: more chunks per document but each embeds faster.

### 4. Embed Worker Count Sweet Spot

| Workers | vec/s (256-char) | Notes |
|---------|-----------------|-------|
| 4 | 249 | Under-saturated |
| 8 | **576** | Optimal |
| 16 | 444 | GPU contention |

## Changes Made

### Server Config (`docker/llamacpp/docker-compose.yml`)

- Model: `nomic-embed-text-v1.5.Q8_0.gguf` (768d, 137MB) → `all-MiniLM-L6-v2.Q8_0.gguf` (384d, 23MB)
- `--ctx-size`: 2048 → 8192
- Added: `--batch-size 512`, `--ubatch-size 512`, `--parallel 16`
- Added: `--flash-attn on`, `--threads 8`, `--threads-batch 8`

### llamacpp Driver (`pkg/embed/driver/llamacpp/llamacpp.go`)

- Multi-server round-robin: comma-separated `cfg.Addr` ("http://h1:8087,http://h2:8088")
- `atomic.Uint64` counter for lock-free round-robin
- HTTP Transport: `MaxIdleConns/PerHost/PerConn: 64`
- `defaultBatchSize`: 64 → 16
- JSON decode: `json.NewDecoder().Decode()` → `io.ReadAll()` + `json.Unmarshal()`
- Health checks all servers on Open()
- Error messages include server address

### ONNX Driver (`pkg/embed/driver/onnx/onnx.go`)

- Session reuse: create once in `Open()`, reuse across `Embed()` calls
- `sync.Mutex` for thread safety (ONNX `session.Run()` not thread-safe)
- Pad last batch to session capacity instead of recreating session
- CoreML EP support via `cfg.Addr = "coreml"`

## Optimal Server Launch

```bash
llama-server --model ~/data/models/all-MiniLM-L6-v2.Q8_0.gguf \
  --embedding --pooling mean \
  --ctx-size 8192 --batch-size 512 --ubatch-size 512 \
  --parallel 16 --cont-batching --flash-attn on \
  --threads 8 --threads-batch 8 --port 8086
```

## Optimal Pipeline Flags

```bash
search cc fts embed run --input <dir> --driver llamacpp \
  --embed-workers 8 --batch-size 16 --max-chars 256 --overlap 100
```

## Why 1K vec/s Isn't Achievable on Single M4

The M4 Metal GPU tops out at ~800 raw vec/s with 250-char text. Pipeline
overhead (HTTP serialization, JSON encode/decode, batching, file I/O)
consumes 25-40% leaving 466-576 effective. Multi-server on the same GPU
doesn't scale due to shared Metal compute. Reaching 1K would require
multiple GPUs, a faster model architecture, or a non-HTTP inference path.
