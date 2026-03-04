# spec/0667: Qwen3-Embedding-0.6B vs all-MiniLM-L6-v2

## Summary

Added Qwen3-Embedding-0.6B as a llamacpp model option. It's a much larger model (600M vs 22M params)
with higher-dimensional embeddings (1024 vs 384) and 32K context (vs 512 tokens).

## Benchmark Results (M4 Mac, 100 files, Metal GPU)

| Model | Chunk Size | Vectors | Errors | vec/s | Time | Output |
|-------|-----------|---------|--------|-------|------|--------|
| all-MiniLM-L6-v2 (Q8, 23MB) | 256 chars | 2,999 | 0 | **532** | 5.6s | 4.4 MB |
| Qwen3-Embedding-0.6B (Q8, 639MB) | 2000 chars | 307 | 0 | **3** | 98s | 1.2 MB |

### Key Observations

1. **MiniLM is ~177x faster** in raw vec/s (532 vs 3)
2. **Qwen3 produces ~10x fewer vectors** for same content (307 vs 2999) due to larger chunks
3. **Qwen3 embeddings are 2.67x larger** per vector (1024 vs 384 floats)
4. **Net throughput**: MiniLM embeds 100 files in 5.6s vs Qwen3 in 98s — **17x faster end-to-end**
5. **MiniLM can't do 2000-char chunks** — its 512-token context is too small

### Quality Trade-offs

| Aspect | MiniLM | Qwen3 |
|--------|--------|-------|
| Dimension | 384 | 1024 |
| MTEB rank | Good (older) | Better (2025 SOTA for size) |
| Multilingual | English-focused | 100+ languages |
| Instruction-aware | No | Yes (1-5% boost) |
| Max context | 512 tokens (~250 chars) | 32K tokens (~16K chars) |
| Matryoshka (MRL) | No | Yes (32-1024 dim) |

## Server Configuration

### MiniLM (existing, port 8086)
```bash
llama-server --model all-MiniLM-L6-v2.Q8_0.gguf \
  --embedding --pooling mean \
  --ctx-size 8192 --batch-size 512 --ubatch-size 512 \
  --parallel 16 --flash-attn on --threads 8
```

### Qwen3 (new, port 8087)
```bash
llama-server --model Qwen3-Embedding-0.6B-Q8_0.gguf \
  --embedding --pooling last \
  --ctx-size 32768 --batch-size 2048 --ubatch-size 2048 \
  --parallel 4 --flash-attn on --threads 8
```

**Critical differences:**
- `--pooling last` (not mean!) — Qwen3 uses last-token pooling
- `--ctx-size 32768` — full 32K context
- `--parallel 4` — fewer slots, 8192 tokens each (vs 16 slots × 512 for MiniLM)

## Pipeline Flags

```bash
# MiniLM (fast, many small vectors)
search cc fts embed run --driver llamacpp --addr http://localhost:8086 \
  --model all-MiniLM-L6-v2 --embed-workers 8 --batch-size 16 \
  --max-chars 256 --overlap 100

# Qwen3 (slow, fewer high-quality vectors)
search cc fts embed run --driver llamacpp --addr http://localhost:8087 \
  --model qwen3-embedding-0.6b --embed-workers 4 --batch-size 4 \
  --max-chars 2000 --overlap 200
```

## When to Use Each

**MiniLM**: High-throughput batch embedding of large corpora. Speed matters more than quality.
Fine-grained retrieval (small chunks = precise matching).

**Qwen3**: Quality-sensitive applications, multilingual content, longer document passages.
Fewer vectors = smaller index. Instruction-aware queries for better retrieval.

## Changes Made

1. `pkg/embed/models.go` — added `qwen3-embedding-0.6b` model entry
2. `docker/llamacpp/docker-compose.yml` — added `llamacpp-embed-qwen3` service (port 8087)
