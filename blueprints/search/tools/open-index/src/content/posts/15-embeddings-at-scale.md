---
slug: embeddings-at-scale
title: "A Million Embeddings Before Breakfast"
date: 2026-03-03
summary: "1024-dimensional vectors for a billion pages. The math, the GPUs, and why it takes 70 days on one card."
tags: [ai, engineering]
---

1 billion pages. 1024 dimensions. 4 bytes per float.

Multiply it out: 1,000,000,000 x 1,024 x 4 = **4 TB**. That's the raw vector data. No metadata, no index overhead, no duplicate chunks. Just the embeddings themselves, stored as float32 arrays.

But you can't store what you haven't generated. multilingual-e5-large processes about 500 chunks per second on an A100. A typical web page splits into 3 chunks. A billion pages means 3 billion chunks. At 500 chunks/s, that's 6,000,000 seconds. Divide by 86,400. You get **70 days** on a single GPU.

The numbers are humbling before you've written a line of pipeline code.

## Why this model and not another?

multilingual-e5-large hits a specific sweet spot: high enough quality to be useful, small enough to be practical, and it works across languages. The web isn't English-only, and neither is the index.

<table>
  <thead>
    <tr>
      <th>Model</th>
      <th>Dimensions</th>
      <th>Languages</th>
      <th>MTEB Avg</th>
      <th>Trade-off</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>all-MiniLM-L6-v2</strong></td>
      <td>384</td>
      <td>English</td>
      <td>~56</td>
      <td>Fast, but lower quality and English-only</td>
    </tr>
    <tr>
      <td><strong>BGE-large-en-v1.5</strong></td>
      <td>1024</td>
      <td>English</td>
      <td>~64</td>
      <td>Strong, but no multilingual support</td>
    </tr>
    <tr>
      <td><strong>multilingual-e5-large</strong></td>
      <td style="color:#4ade80"><strong>1024</strong></td>
      <td style="color:#4ade80"><strong>100+</strong></td>
      <td style="color:#4ade80"><strong>~61</strong></td>
      <td>Best balance for a multilingual web index</td>
    </tr>
    <tr>
      <td><strong>Cohere embed-v3</strong></td>
      <td>1024</td>
      <td>100+</td>
      <td>~65</td>
      <td>API-only, pay per call, data leaves your infra</td>
    </tr>
  </tbody>
</table>

Cohere scores higher on benchmarks. But it's an API. At 3 billion chunks, the bill would be staggering, and every embedding request ships crawl data to someone else's servers. Open weights mean the model runs on our hardware and the marginal cost per embedding is electricity.

1024 dimensions: 384 is too small for multilingual nuance, 2048+ doubles storage for marginal gains. At 1024, the vector separates "car repair" from "car rental" in Japanese, Portuguese, and Finnish.

## Chunks, not pages

You don't embed a whole page. A typical web page is 5-50 KB of text. The model's context window is 512 tokens -- roughly 400 words. Feed it a 10,000-word page and it truncates silently, embedding only the first 400 words. That's useless.

So you chunk. Split the page into paragraphs or fixed-size windows of ~256 tokens with overlap. Embed each chunk independently. Store chunk-level vectors with a back-pointer to the source page.

A search query matches against individual chunks, not entire documents. Someone searching "how to replace brake pads" hits the specific paragraph about brake replacement, not a 50-page automotive manual. Better precision. More vectors -- a billion pages become 3 billion chunks -- but that's the price of useful search.

## The GPU pipeline

The bottleneck isn't the model. It's keeping the GPU fed.

<pre><code>  <span style="color:#4ade80">Raw text</span> (from DuckDB, 16 shards)
       |
       | extract, clean, strip HTML
       v
  <span style="color:#fbbf24">Chunker</span> (CPU)
       |  split into ~256-token windows
       v
  <span style="color:#60a5fa">Tokenizer</span> (CPU)
       |  WordPiece → token IDs + attention masks
       v
  <span style="color:#fbbf24">Batch Padder</span> (CPU)
       |  pad to max length in batch, collate tensors
       v
  <span style="color:#e0e0e0">══════════════════════════════════</span>  <span style="color:#888">PCIe boundary</span>
       v
  <span style="color:#60a5fa">Model Inference</span> (GPU, A100)
       |  forward pass → 1024-dim raw embeddings
       v
  <span style="color:#4ade80">L2 Normalize</span> (GPU)
       |  unit vectors for cosine similarity
       v
  <span style="color:#e0e0e0">══════════════════════════════════</span>  <span style="color:#888">PCIe boundary</span>
       v
  <span style="color:#4ade80">Vald</span> (gRPC insert, vector + chunk ID)</code></pre>

The key technique: double-buffering. While batch N runs on the GPU, batch N+1 is being tokenized and padded on the CPU. When the GPU finishes, the next batch is already waiting. No idle cycles. The CPU work (tokenizing, padding, collating) takes roughly 30% of the GPU inference time, so a single prefetch thread keeps the pipeline saturated.

<div class="note">
  <strong>Batch size matters.</strong> Too small (8-16) and GPU utilization drops below 60% -- the overhead of launching kernels dominates. Too large (512+) and you waste memory on padding since shorter sequences get padded to match the longest in the batch. The sweet spot for e5-large on A100: <strong>64-128 chunks per batch</strong>. That gives ~92% GPU utilization with acceptable padding overhead.
</div>

## Product quantization: 4 TB is too much RAM

Vald keeps vectors in memory for fast ANN search. 4 TB of float32 vectors means 4 TB of RAM. That's 64 machines with 64 GB each, just for the vectors.

Product quantization (PQ) compresses vectors by splitting each 1024-dim vector into 128 subspaces of 8 dimensions, then replacing each subspace with the index of its nearest centroid from a learned codebook. One byte per subspace. 1024 floats become 128 bytes.

<table>
  <thead>
    <tr>
      <th>Compression</th>
      <th>Bytes/Vector</th>
      <th>1B Vectors</th>
      <th>Recall@10</th>
      <th>Trade-off</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>None (float32)</td>
      <td>4,096</td>
      <td><strong>4 TB</strong></td>
      <td>100%</td>
      <td>Exact, but enormous</td>
    </tr>
    <tr>
      <td>PQ (128 subspaces)</td>
      <td>128</td>
      <td style="color:#4ade80"><strong>~128 GB</strong></td>
      <td>~95%</td>
      <td>32x smaller, slight recall loss</td>
    </tr>
    <tr>
      <td>PQ + reranking</td>
      <td>128 + partial</td>
      <td><strong>~250 GB</strong></td>
      <td>~98%</td>
      <td>Rerank top-100 with full vectors</td>
    </tr>
    <tr>
      <td>float16</td>
      <td>2,048</td>
      <td><strong>2 TB</strong></td>
      <td>~99.9%</td>
      <td>Half the memory, near-lossless</td>
    </tr>
  </tbody>
</table>

The PQ + reranking approach is the plan. Store compressed vectors in Vald for the initial ANN search, keep full-precision vectors on disk (DuckDB/Parquet), rerank the top-100 candidates with exact cosine similarity. 250 GB fits on 4 nodes with 64 GB each. That's manageable.

<div class="note">
  <strong>5% recall loss sounds bad. It isn't.</strong> At top-10, losing 5% recall means that in 1 out of 20 queries, one result that should have been in the top 10 gets replaced by one that's slightly further away in vector space. For web search, where you're combining vector results with BM25 results from Tantivy anyway, this is invisible. The hybrid ranking absorbs the noise.
</div>

## Incremental embedding: don't redo the work

New crawl data arrives daily. Re-embedding a billion pages every time would be absurd. The pipeline needs to be incremental.

Each crawl batch gets its own embedding segment. New pages get embedded and appended to Vald. Changed pages (detected by content hash) get re-embedded. Unchanged pages keep their existing vectors. The segment structure means old embeddings can be re-generated with a better model later -- rebuild the segment, swap it in.

DuckDB tracks what's been embedded. A `WHERE embedded_at IS NULL` query produces the work queue. The pipeline processes only the delta.

## Multi-GPU scaling

The math is straightforward because the work is embarrassingly parallel. Each chunk is independent. No attention across chunks, no cross-document context, no gradient synchronization. Pure data parallelism.

<table>
  <thead>
    <tr>
      <th>GPUs</th>
      <th>Chunks/sec</th>
      <th>Time (3B chunks)</th>
      <th>Hardware</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>1x A100</td>
      <td>~500</td>
      <td><strong>70 days</strong></td>
      <td>Single node</td>
    </tr>
    <tr>
      <td>4x A100</td>
      <td>~2,000</td>
      <td><strong>17 days</strong></td>
      <td>1 node (4-GPU)</td>
    </tr>
    <tr>
      <td>8x A100</td>
      <td>~4,000</td>
      <td><strong>9 days</strong></td>
      <td>1 node (8-GPU) or 2 nodes</td>
    </tr>
    <tr>
      <td>32x A100</td>
      <td>~16,000</td>
      <td><strong>~2 days</strong></td>
      <td>4 nodes (8-GPU each)</td>
    </tr>
  </tbody>
</table>

Each GPU runs an independent copy of the model, pulls chunks from a shared queue, writes vectors to Vald. No cross-GPU communication. No NCCL. No distributed training framework. Just a `for` loop on each card.

A100 spot instances run about $1/hr. 70 GPU-days = $1,680 for the full billion-page run. Not cheap, but it's a one-time batch cost, not a recurring per-query fee.

## Where this stands

Model selected. Chunking pipeline prototyped and tested on sample data. GPU inference benchmarked -- the 500 chunks/s number comes from real runs on A100, not spec sheets. Vald integration is in design, following the same split architecture described in the vector search post: Vald holds vectors and IDs, DuckDB holds everything else.

The full embedding pipeline is post-Tantivy. Full-text search ships first because it doesn't need GPUs or a Kubernetes cluster. Vector search layers on top. The initial target is 10-50M pages -- enough to validate chunking quality, PQ recall, and the Vald integration end-to-end before committing GPU time to the full run.

70 days on one card. 9 days on eight. 2 days on thirty-two. The math is simple. The engineering is making sure every one of those 6 million seconds of GPU time produces a useful vector.
