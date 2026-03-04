# spec/0661: Embedding Models Report

## Current Default

**all-MiniLM-L6-v2** (ONNX driver)
- Dimensions: 384
- Parameters: 22.7M
- ONNX size: ~90MB (FP32), ~23MB (INT8 quantized)
- Max tokens: 256
- Architecture: BERT-based, 6 layers
- License: Apache 2.0
- Quality: Strong general-purpose English embedding model
- Speed: ~3,000 sentences/sec on CPU (batch=32)
- Use case: Default model, good balance of quality and speed

## Recommended Models by Use Case

### Tier 1: Small & Fast (< 100MB, < 512 dim)

| Model | Dim | Params | Size | Speed | Quality | Notes |
|-------|-----|--------|------|-------|---------|-------|
| **all-MiniLM-L6-v2** | 384 | 22.7M | 90MB | ★★★★★ | ★★★☆☆ | Default. Best speed/quality ratio |
| all-MiniLM-L12-v2 | 384 | 33.4M | 134MB | ★★★★☆ | ★★★★☆ | 12 layers, better quality |
| bge-small-en-v1.5 | 384 | 33.4M | 134MB | ★★★★☆ | ★★★★☆ | BAAI, strong benchmark scores |
| snowflake-arctic-embed-s | 384 | 33.4M | 134MB | ★★★★☆ | ★★★★☆ | Snowflake, good for retrieval |
| gte-small | 384 | 33.4M | 134MB | ★★★★☆ | ★★★☆☆ | Alibaba DAMO, multilingual |

### Tier 2: Medium (100-500MB, 768 dim)

| Model | Dim | Params | Size | Speed | Quality | Notes |
|-------|-----|--------|------|-------|---------|-------|
| **nomic-embed-text-v1.5** | 768 | 137M | 274MB | ★★★☆☆ | ★★★★☆ | Best GGUF support, llama.cpp native |
| bge-base-en-v1.5 | 768 | 109M | 438MB | ★★★☆☆ | ★★★★☆ | BAAI, strong retrieval |
| gte-base-en-v1.5 | 768 | 137M | 548MB | ★★★☆☆ | ★★★★☆ | Alibaba, good multilingual |
| e5-base-v2 | 768 | 109M | 438MB | ★★★☆☆ | ★★★★☆ | Microsoft, instruction-tuned |
| jina-embeddings-v2-base-en | 768 | 137M | 548MB | ★★★☆☆ | ★★★★☆ | 8K context, long document support |

### Tier 3: Large & High Quality (> 500MB, 1024+ dim)

| Model | Dim | Params | Size | Speed | Quality | Notes |
|-------|-----|--------|------|-------|---------|-------|
| bge-large-en-v1.5 | 1024 | 335M | 1.3GB | ★★☆☆☆ | ★★★★★ | Top MTEB scores |
| gte-large-en-v1.5 | 1024 | 434M | 1.7GB | ★★☆☆☆ | ★★★★★ | Alibaba, best in class |
| nomic-embed-text-v1.5 (768) | 768 | 137M | 274MB | ★★★☆☆ | ★★★★☆ | Matryoshka: can truncate to 256/512 dim |
| e5-large-v2 | 1024 | 335M | 1.3GB | ★★☆☆☆ | ★★★★★ | Microsoft, top retrieval |
| mxbai-embed-large-v1 | 1024 | 335M | 1.3GB | ★★☆☆☆ | ★★★★★ | Mixedbread, strong MTEB |

### Tier 4: Multilingual

| Model | Dim | Params | Size | Speed | Quality | Notes |
|-------|-----|--------|------|-------|---------|-------|
| multilingual-e5-small | 384 | 118M | 472MB | ★★★★☆ | ★★★☆☆ | 100+ languages |
| multilingual-e5-base | 768 | 278M | 1.1GB | ★★★☆☆ | ★★★★☆ | Best multilingual quality |
| paraphrase-multilingual-MiniLM-L12-v2 | 384 | 118M | 472MB | ★★★★☆ | ★★★☆☆ | 50+ languages, fast |
| bge-m3 | 1024 | 568M | 2.3GB | ★☆☆☆☆ | ★★★★★ | Dense+sparse+colbert, multilingual |

## GGUF Availability (for llama.cpp driver)

Models with well-tested GGUF conversions for llama.cpp:

| Model | GGUF Source | Q8_0 Size | Tested |
|-------|------------|-----------|--------|
| nomic-embed-text-v1.5 | nomic-ai/nomic-embed-text-v1.5-GGUF | ~137MB | ✓ native support |
| bge-small-en-v1.5 | CompendiumLabs/bge-small-en-v1.5-gguf | ~67MB | ✓ |
| bge-base-en-v1.5 | CompendiumLabs/bge-base-en-v1.5-gguf | ~220MB | ✓ |
| bge-large-en-v1.5 | CompendiumLabs/bge-large-en-v1.5-gguf | ~670MB | ✓ |
| all-MiniLM-L6-v2 | leliuga/all-MiniLM-L6-v2-GGUF | ~46MB | ✓ |
| e5-small-v2 | ChristianAzinn/e5-small-v2-gguf | ~67MB | experimental |

## ONNX Availability (for ONNX driver)

All sentence-transformers models export to ONNX via `optimum`:

```bash
optimum-cli export onnx --model sentence-transformers/all-MiniLM-L6-v2 ./onnx/
```

Pre-exported ONNX models are available on HuggingFace for most popular models
in the `onnx/` subdirectory of the model repo.

## Recommendations

### For CC markdown embedding (current use case)

**Primary: all-MiniLM-L6-v2 via ONNX** (384-dim)
- Fastest inference, smallest footprint
- Good quality for English web content
- ~3,000 sentences/sec on CPU

**Secondary: nomic-embed-text-v1.5 via llamacpp** (768-dim)
- Best llama.cpp integration
- Matryoshka support (truncate to 256-dim for speed)
- Longer context (8192 tokens vs 256)

### For production search

**bge-large-en-v1.5** or **gte-large-en-v1.5** (1024-dim)
- Top MTEB retrieval benchmarks
- Significant quality improvement for reranking
- ~500 sentences/sec on CPU

### For multilingual content

**multilingual-e5-base** (768-dim)
- 100+ languages
- Good quality across languages
- ~800 sentences/sec on CPU

## Storage Estimates

For 1M markdown documents, average 3 chunks per document = 3M vectors:

| Dim | vectors.bin size | meta.jsonl size | Total |
|-----|-----------------|-----------------|-------|
| 384 | 4.3 GB | ~300 MB | ~4.6 GB |
| 768 | 8.6 GB | ~300 MB | ~8.9 GB |
| 1024 | 11.5 GB | ~300 MB | ~11.8 GB |

## Quantization Options

For reducing storage and improving search speed:

- **Float16**: 50% size reduction, minimal quality loss
- **INT8**: 75% size reduction, ~1-2% quality loss
- **Binary**: 32× size reduction, significant quality loss but fast hamming distance

Current implementation stores float32. INT8 quantization can be added as a
post-processing step or as a `--quantize` flag to `search cc fts embed`.
