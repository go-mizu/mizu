---
slug: tantivy-full-text-search
title: "Full-Text Search for a Billion Pages"
date: 2026-02-25
summary: "BM25 has powered web search for 30 years. We're bringing it to OpenIndex with a Rust library, not a Java cluster."
tags: [roadmap, search]
---

BM25 has been the backbone of web search for 30 years. It predates Google. It predates the web as most people know it. Vector embeddings and neural retrieval get the headlines now, but when someone types `"connection refused" golang net/http` into a search box, they want the page that contains those exact words. BM25 finds it. A vector model might find a page about "networking errors in Go" -- related, but wrong.

We need both. This post is about the keyword side.

---

## Why Tantivy instead of Elasticsearch?

Elasticsearch is a fine piece of software wrapped in a terrible operational experience. You need a JVM. You need a cluster -- master nodes, data nodes, coordinating nodes. You need YAML configuration files that are 200 lines long before you've indexed a single document. You need 4GB of heap minimum, and the garbage collector will pause your queries at the worst possible moment. You need to worry about split-brain, shard rebalancing, and that one node that decided to leave the cluster at 3 AM.

Tantivy is a Rust library. You call a function. It builds an inverted index. You call another function. It searches. There's no cluster, no JVM, no YAML, no heap tuning. It runs in your process, uses your memory, and exits when you're done.

<table>
  <thead>
    <tr>
      <th>Concern</th>
      <th>Elasticsearch</th>
      <th>Tantivy</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Runtime</strong></td>
      <td>JVM (4GB+ heap)</td>
      <td>Native Rust library</td>
    </tr>
    <tr>
      <td><strong>Deployment</strong></td>
      <td>Cluster of nodes</td>
      <td>Embedded in your binary</td>
    </tr>
    <tr>
      <td><strong>Configuration</strong></td>
      <td>YAML files, index templates, ILM policies</td>
      <td>Rust structs, schema builder</td>
    </tr>
    <tr>
      <td><strong>GC pauses</strong></td>
      <td>Yes (G1GC, tunable but present)</td>
      <td>None</td>
    </tr>
    <tr>
      <td><strong>Scaling model</strong></td>
      <td>Cluster auto-rebalancing</td>
      <td>You control shard boundaries</td>
    </tr>
    <tr>
      <td><strong>Query capability</strong></td>
      <td>BM25, filters, aggregations, scripting</td>
      <td>BM25, boolean, phrase, faceted, range</td>
    </tr>
  </tbody>
</table>

Elasticsearch gives you more out of the box -- aggregations, scripting, a REST API, a plugin ecosystem. But we already have DuckDB for analytics and aggregations. We don't need another analytics engine. We need an inverted index that does BM25 scoring and gets out of the way.

## What Tantivy actually gives us

The core feature set: BM25 scoring with configurable k1/b parameters, inverted index construction with automatic segment merging, phrase queries, boolean queries, faceted search, and range queries on numeric fields. It handles compaction internally -- small segments merge into larger ones in the background. And it ships a query parser that handles `AND`, `OR`, `NOT`, phrase matching with quotes, and field-scoped queries like `title:"climate change"`.

That's everything we need for keyword search. Nothing we don't.

## The indexing pipeline

Crawled pages arrive as raw HTML with metadata. They need to become searchable documents in an inverted index. The pipeline looks like this:

<pre><code>  Crawl Output (recrawler / domain crawler)
         |
         v
  +------------------+
  |  Text Extraction  |  strip HTML, decode entities,
  |                    |  handle charset, remove boilerplate
  +--------+---------+
           |
           v
  +------------------+
  |    Tokenization   |  language-aware: whitespace for EN,
  |                    |  n-grams for CJK, ICU for Arabic
  +--------+---------+
           |
           v
  +------------------+
  |  Tantivy Indexer  |  build inverted index, BM25 stats,
  |                    |  segment merging, commit
  +--------+---------+
           |
           v
  +------------------+
  |   Index Segments  |  immutable, merge in background,
  |   (.tantivy/)     |  one directory per crawl batch
  +------------------+</code></pre>

In Rust, the indexing looks roughly like this:

<pre><code><span style="color:#60a5fa">use</span> tantivy::{schema::*, Index, <span style="color:#e0e0e0">IndexWriter</span>};

<span style="color:#555">// Define the schema</span>
<span style="color:#60a5fa">let mut</span> <span style="color:#e0e0e0">builder</span> = Schema::builder();
<span style="color:#60a5fa">let</span> <span style="color:#e0e0e0">url</span>   = builder.add_text_field(<span style="color:#4ade80">"url"</span>, <span style="color:#fbbf24">STORED</span>);
<span style="color:#60a5fa">let</span> <span style="color:#e0e0e0">title</span> = builder.add_text_field(<span style="color:#4ade80">"title"</span>, <span style="color:#fbbf24">TEXT</span> | <span style="color:#fbbf24">STORED</span>);
<span style="color:#60a5fa">let</span> <span style="color:#e0e0e0">body</span>  = builder.add_text_field(<span style="color:#4ade80">"body"</span>, <span style="color:#fbbf24">TEXT</span>);
<span style="color:#60a5fa">let</span> <span style="color:#e0e0e0">domain</span> = builder.add_text_field(<span style="color:#4ade80">"domain"</span>, <span style="color:#fbbf24">STRING</span> | <span style="color:#fbbf24">STORED</span>);
<span style="color:#60a5fa">let</span> <span style="color:#e0e0e0">schema</span> = builder.build();

<span style="color:#555">// Create the index and writer</span>
<span style="color:#60a5fa">let</span> <span style="color:#e0e0e0">index</span> = Index::create_in_dir(<span style="color:#4ade80">"./index_shard_00"</span>, schema.clone())?;
<span style="color:#60a5fa">let mut</span> <span style="color:#e0e0e0">writer</span>: <span style="color:#fbbf24">IndexWriter</span> = index.writer(<span style="color:#fbbf24">256_000_000</span>)?; <span style="color:#555">// 256MB heap</span>

<span style="color:#555">// Index a document</span>
writer.add_document(doc!(
    url   => <span style="color:#4ade80">"https://example.com/page"</span>,
    title => <span style="color:#4ade80">"Example Page Title"</span>,
    body  => <span style="color:#e0e0e0">extracted_text</span>,
    domain => <span style="color:#4ade80">"example.com"</span>,
))?;

<span style="color:#555">// Commit flushes to a new segment</span>
writer.commit()?;</code></pre>

256MB of writer heap. One function call to add a document. One to commit. Tantivy handles the rest -- building the posting lists, computing term frequencies, writing the segment files. Compare that to crafting an Elasticsearch bulk indexing request and hoping the cluster's ingest pipeline doesn't reject it.

## Language-aware tokenization isn't optional

English tokenization is whitespace plus lowercasing. Maybe stemming. Maybe stop words. It's a solved problem.

CJK (Chinese, Japanese, Korean) has no whitespace between words. Arabic runs right-to-left with complex affix morphology. Thai has no spaces. You can't treat these like English and expect useful search results.

<table>
  <thead>
    <tr>
      <th>Language Family</th>
      <th>Tokenization Strategy</th>
      <th>Example</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>English / European</strong></td>
      <td>Whitespace + lowercase + stemming</td>
      <td>"running" &rarr; "run"</td>
    </tr>
    <tr>
      <td><strong>CJK</strong></td>
      <td>Character bi-grams or dictionary segmentation (Lindera/jieba)</td>
      <td>"東京都" &rarr; "東京" + "京都" + "都"</td>
    </tr>
    <tr>
      <td><strong>Arabic / Hebrew</strong></td>
      <td>ICU tokenizer with affix stripping</td>
      <td>"والكتب" &rarr; "كتب" (books)</td>
    </tr>
    <tr>
      <td><strong>Thai / Khmer</strong></td>
      <td>ICU word break rules (no whitespace)</td>
      <td>"กรุงเทพ" &rarr; "กรุง" + "เทพ"</td>
    </tr>
  </tbody>
</table>

Tantivy supports custom tokenizers per field. The plan is to detect language at indexing time (via HTTP headers, HTML lang attribute, or CLD3 detection) and route to the appropriate tokenizer. English gets the default tokenizer with stemming. CJK gets bi-gram or Lindera. Everything else gets ICU as a fallback.

<div class="note">
  <strong>Why bi-grams for CJK?</strong> Dictionary-based segmentation (Lindera, jieba) gives cleaner tokens but requires maintaining word lists. Bi-grams are language-agnostic and index everything. The trade-off is larger index size and fuzzier matching. We're starting with bi-grams and will benchmark dictionary segmentation once the pipeline is stable.
</div>

## Sharding at scale

One Tantivy index per crawl batch. The recrawler already shards DuckDB output by domain hash across 16 files. The search index follows the same pattern: per-batch segments, merged incrementally as new crawls complete.

A query fans out across shards, each shard returns its top-K results with BM25 scores, and a coordinator merges them. This is the same architecture Elasticsearch uses internally -- every Elasticsearch "index" is a collection of shards, every shard is a Lucene index. The difference: we decide where the shard boundaries fall instead of letting a cluster manager guess.

The shard boundary matters more than people think. Shard by domain hash and queries for a single domain only hit one shard. Shard by crawl date and time-range queries are efficient. We're starting with domain-hash sharding to match the existing DuckDB layout, which means `site:example.com` queries touch exactly one shard.

## BM25 vs. vector search -- what each actually does well

<table>
  <thead>
    <tr>
      <th>Query Type</th>
      <th>BM25 (Tantivy)</th>
      <th>Vector Search</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Exact phrase: <code>"segfault in malloc"</code></td>
      <td>Exact match, fast, deterministic</td>
      <td>Might find related memory errors</td>
    </tr>
    <tr>
      <td>Rare term: <code>ECONNREFUSED</code></td>
      <td>High IDF makes this trivial</td>
      <td>Rare tokens often poorly embedded</td>
    </tr>
    <tr>
      <td>Known-item: <code>RFC 7231</code></td>
      <td>Finds the exact document</td>
      <td>May find "HTTP specification" pages</td>
    </tr>
    <tr>
      <td>Conceptual: "how do computers learn"</td>
      <td>Matches pages with those words</td>
      <td>Finds ML/AI content even without keyword overlap</td>
    </tr>
    <tr>
      <td>Multilingual: English query, Japanese pages</td>
      <td>Fails (different token space)</td>
      <td>Multilingual models handle this</td>
    </tr>
    <tr>
      <td>Typo tolerance</td>
      <td>None (edit distance is separate)</td>
      <td>Embeddings absorb minor typos</td>
    </tr>
  </tbody>
</table>

Neither wins everywhere. BM25 is better for precision -- when the user knows what they're looking for. Vector search is better for recall -- when the user knows what they mean but not how it's phrased. A production search engine needs both, with score fusion to blend the results.

OQL (the OpenIndex query language, coming in a future post) will handle the fusion. For now, Tantivy handles the keyword path.

## Integration with DuckDB

Tantivy handles text. DuckDB handles everything else -- domain, date, HTTP status, content type, language. A query like `climate change site:edu after:2025` splits into two paths:

1. **Tantivy** resolves `"climate change"` against the inverted index and returns matching document IDs with BM25 scores
2. **DuckDB** filters those IDs by `domain LIKE '%.edu'` and `fetched_at > '2025-01-01'`
3. A **score fusion** step merges the results, re-ranks, and returns the top K

This split plays to each engine's strength. Tantivy doesn't need to know about dates or domains. DuckDB doesn't need an inverted index. The document ID is the join key between them.

<div class="note">
  <strong>Why not put everything in Tantivy?</strong> Tantivy can store and filter on numeric/date fields, and for simple cases that works fine. But DuckDB already holds all our crawl metadata with full SQL -- window functions, CTEs, aggregations across crawl runs. Duplicating that into Tantivy's schema would mean maintaining two sources of truth. Better to let each engine do what it's best at.
</div>

## What's built vs. what's planned

Honest status:

- **Built and battle-tested**: The DuckDB pipeline -- 16-shard ResultDB, batch-VALUES inserts, 200K rows/s sustained. The recrawler producing crawl data. The domain crawler. Common Crawl integration with Parquet. All of this works today and has processed hundreds of millions of URLs.
- **Designed, prototyped**: The Tantivy indexing pipeline architecture. The schema. The query fanout across shards. The DuckDB-Tantivy integration layer with document ID joins.
- **Planned**: Production deployment of full-text search. Language-aware tokenization beyond English. Score fusion between BM25 and vector search. OQL query parsing.

Target: mid-2026 for the first production index covering a full Common Crawl snapshot.

<div class="note">
  <strong>This is a roadmap post.</strong> The pipeline architecture is defined and the individual pieces are proven. But the end-to-end Tantivy integration hasn't shipped yet. We'll publish benchmarks -- real numbers, not projections -- once we've indexed a full crawl batch and run queries against it.
</div>

The crawl pipeline can already produce hundreds of millions of pages per run. The storage layer handles the throughput. The missing piece is the search layer that makes all that data queryable by keyword. Tantivy fills that gap -- no JVM, no cluster, no operational overhead. Just an inverted index and BM25.
