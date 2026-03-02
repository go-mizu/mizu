---
slug: vector-search-without-cloud
title: "Vector Search Without a Cloud Bill"
date: 2026-02-26
summary: "Semantic search that runs on your hardware. No managed service, no per-query pricing, no vendor lock-in."
tags: [roadmap, search]
---

Search for "car repair." You want pages about automobile maintenance, vehicle servicing, mechanic shops. Keyword search won't find them. The words are different. The meaning is the same.

This is the gap between lexical and semantic search. Tantivy handles the lexical side -- exact terms, BM25 ranking, inverted indexes. But web search that only matches on exact words misses too much. We need something that understands meaning.

## What happens when you turn text into numbers?

Dense embeddings transform text into a 1024-dimensional vector -- a list of 1024 floating-point numbers. Similar meanings cluster together in this vector space. "Car" and "automobile" end up near each other. "Car" and "banana" don't.

The model we're planning to use: **multilingual-e5-large**. Three reasons:

- **Multilingual coverage** -- the web isn't English-only, and neither is the index
- **1024 dimensions** -- large enough to capture nuance, small enough to store at scale
- **Strong MTEB performance** -- consistently near the top of embedding benchmarks without being a 7B-parameter monster

The embedding process is straightforward. Take a crawled page, chunk it into paragraphs, run each chunk through the model, get back a 1024-element float32 vector. Store the vector alongside a document ID. That's it.

## Why we aren't paying Pinecone

The obvious question: why self-host? Managed vector databases exist. They handle scaling, replication, and backups. They also charge per query, lock you into their API, and store your data on their infrastructure.

<table>
  <thead>
    <tr>
      <th>Property</th>
      <th>Pinecone</th>
      <th>Weaviate</th>
      <th>Milvus</th>
      <th>Vald</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>License</strong></td>
      <td>Proprietary</td>
      <td>BSD-3</td>
      <td>Apache 2.0</td>
      <td>Apache 2.0</td>
    </tr>
    <tr>
      <td><strong>Deployment</strong></td>
      <td>Managed only</td>
      <td>Self-host or cloud</td>
      <td>Self-host or cloud</td>
      <td>Self-host (K8s)</td>
    </tr>
    <tr>
      <td><strong>Dependencies</strong></td>
      <td>N/A</td>
      <td>Go + modules</td>
      <td>etcd + MinIO + Pulsar</td>
      <td>Kubernetes only</td>
    </tr>
    <tr>
      <td><strong>ANN algorithm</strong></td>
      <td>Proprietary</td>
      <td>HNSW</td>
      <td>Multiple (IVF, HNSW)</td>
      <td>NGT</td>
    </tr>
    <tr>
      <td><strong>Data residency</strong></td>
      <td>Their servers</td>
      <td>Your choice</td>
      <td>Your choice</td>
      <td>Your choice</td>
    </tr>
    <tr>
      <td><strong>Pricing model</strong></td>
      <td>Per-query + storage</td>
      <td>Free (self-host)</td>
      <td>Free (self-host)</td>
      <td>Free (self-host)</td>
    </tr>
  </tbody>
</table>

Weaviate and Milvus are both open-source, but they're heavy. Milvus needs etcd for metadata, MinIO for object storage, and Pulsar for message streaming -- three distributed systems just to run the vector database. Weaviate is simpler but still requires managing Go modules and schema definitions.

Vald is different. It was built by Yahoo Japan for production-scale similarity search. The architecture is distributed from day one: lightweight agents each hold a shard of the vector space, gRPC for communication, Kubernetes for orchestration. No external dependencies beyond K8s itself.

## How Vald actually works

Vald uses NGT -- Neighborhood Graph and Tree -- for approximate nearest neighbor search. NGT builds a graph where nodes are vectors and edges connect nearby neighbors. To find the nearest vectors to a query, you walk the graph, hopping from neighbor to neighbor, converging toward the closest cluster.

<pre><code>  <span style="color:#60a5fa">Query vector</span>
       |
       v
  <span style="color:#e0e0e0">┌─────────────────────────────────────────┐</span>
  <span style="color:#e0e0e0">│</span>  <span style="color:#fbbf24">Vald Gateway (gRPC)</span>                    <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Fan-out query to all agents             <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">└──────────┬──────────┬──────────┬────────┘</span>
           |          |          |
           v          v          v
  <span style="color:#4ade80">┌─────────┐  ┌─────────┐  ┌─────────┐</span>
  <span style="color:#4ade80">│ Agent 0 │  │ Agent 1 │  │ Agent 2 │</span>  <span style="color:#888">... Agent N</span>
  <span style="color:#4ade80">│ NGT     │  │ NGT     │  │ NGT     │</span>
  <span style="color:#4ade80">│ shard   │  │ shard   │  │ shard   │</span>
  <span style="color:#4ade80">└────┬────┘  └────┬────┘  └────┬────┘</span>
       |          |          |
       v          v          v
  <span style="color:#e0e0e0">┌─────────────────────────────────────────┐</span>
  <span style="color:#e0e0e0">│</span>  <span style="color:#fbbf24">Merge results by distance</span>              <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">│</span>  Return top-K nearest doc IDs           <span style="color:#e0e0e0">│</span>
  <span style="color:#e0e0e0">└─────────────────────────────────────────┘</span></code></pre>

Each agent pod holds a partition of the index in memory. Queries fan out to every agent, each returns its local top-K, and the gateway merges results by distance. Agents auto-balance on Kubernetes -- add more pods, the index redistributes.

## The split architecture

Here's the key design decision: Vald stores only vectors and document IDs. Everything else lives in DuckDB.

A search query flows like this: embed the query text, search Vald for the nearest vectors, get back document IDs, join those IDs against DuckDB for metadata (URL, domain, title, crawl date, content type), apply any filters, return results.

This split keeps Vald small. It only needs to hold float32 vectors and integer IDs. DuckDB handles the rich metadata, filtering, and aggregation -- the same 16-shard architecture we already use for crawl results. No duplication. No sync headaches. Each system does what it's good at.

## Approximate means approximate

NGT doesn't return exact nearest neighbors. It returns *approximate* nearest neighbors. The trade-off: speed for accuracy. In practice, the accuracy is high where it matters.

<table>
  <thead>
    <tr>
      <th>K (results)</th>
      <th>Recall</th>
      <th>Latency (est.)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Top-10</td>
      <td><strong>95%+</strong></td>
      <td>~5ms</td>
    </tr>
    <tr>
      <td>Top-100</td>
      <td>~92%</td>
      <td>~12ms</td>
    </tr>
    <tr>
      <td>Top-1000</td>
      <td>~85%</td>
      <td>~40ms</td>
    </tr>
  </tbody>
</table>

For web search, top-10 is what users see. 95% recall at 5ms latency is the right trade-off. Nobody scrolls to result #847 and complains it should have been #842.

<div class="note">
  <strong>Recall here means:</strong> of the true K nearest neighbors (found by brute-force exact search), what percentage does the ANN algorithm actually return? At top-10, NGT misses maybe one result out of twenty runs. Good enough.
</div>

## The storage math

Time for napkin math. One billion pages, each producing a 1024-dimensional float32 vector:

1B vectors x 1024 dims x 4 bytes = **~4 TB** of raw vector data.

That's a lot of RAM. Vald keeps vectors in memory for fast search. Four terabytes of RAM isn't something you run on a laptop.

<table>
  <thead>
    <tr>
      <th>Index Size</th>
      <th>Raw Vectors</th>
      <th>With PQ (8x compression)</th>
      <th>Realistic Hardware</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>10M pages</td>
      <td>~40 GB</td>
      <td>~5 GB</td>
      <td>1 node, 64 GB RAM</td>
    </tr>
    <tr>
      <td>100M pages</td>
      <td>~400 GB</td>
      <td>~50 GB</td>
      <td>4 nodes, 64 GB each</td>
    </tr>
    <tr>
      <td>1B pages</td>
      <td>~4 TB</td>
      <td>~500 GB</td>
      <td>8-16 nodes, 64 GB each</td>
    </tr>
  </tbody>
</table>

Options for reducing the footprint:

- **Sample the index** -- you don't need every page vectorized. A representative subset still gives good semantic search.
- **Lower dimensions** -- 384 dims instead of 1024 cuts storage by 63%, with some accuracy loss on multilingual queries.
- **Product quantization (PQ)** -- compress vectors 4-8x by quantizing subspaces. Recall drops maybe 2-3% at top-10. Worth it at scale.

<div class="note">
  <strong>"Without a cloud bill" doesn't mean free.</strong> Self-hosting costs hardware. But the cost is predictable -- buy or rent the machines, and the marginal cost per query is zero. No surprise bills when traffic spikes. No vendor deciding to raise prices 3x next quarter.
</div>

## The embedding pipeline

Text flows in from the crawler. The pipeline:

<pre><code>  <span style="color:#4ade80">Crawled pages</span> (DuckDB, 16 shards)
       |
       | extract text, strip HTML
       v
  <span style="color:#fbbf24">Chunking</span> (split into paragraphs, ~256 tokens each)
       |
       | batch chunks (32-64 per batch)
       v
  <span style="color:#60a5fa">multilingual-e5-large</span> (GPU inference)
       |
       | 1024-dim float32 per chunk
       v
  <span style="color:#4ade80">Vald</span> (gRPC insert, vector + doc ID)
       +
  <span style="color:#4ade80">DuckDB</span> (metadata: URL, domain, title, chunk offset)</code></pre>

Estimated throughput on a single GPU: ~500 chunks/second. A billion pages with an average of 3 chunks each means roughly 3 billion embeddings. At 500/s, that's about 70 days on one GPU. Parallelizable across multiple GPUs, obviously. This isn't fast, but it's a batch job that runs once per crawl, not a latency-sensitive path.

## Where this stands

Honest status: Vald integration is in architecture design. The embedding pipeline is prototyped but hasn't run at scale. The DuckDB metadata join works -- it's the same pattern we use for Tantivy's full-text search integration, where the search engine returns doc IDs and DuckDB fills in the rest.

What's built:
- DuckDB metadata store (16-shard, production)
- Doc ID join pattern (working, same as Tantivy path)
- Embedding model selection (multilingual-e5-large, benchmarked)

What's planned:
- Vald cluster deployment on K8s
- GPU embedding pipeline at scale
- Product quantization for the initial subset
- Query API that combines Vald + Tantivy + DuckDB

The target is an initial deployment with a subset of the index -- probably 10-50M pages. Enough to validate the architecture end-to-end before committing to the full billion-page run. We'll write about how that goes.
