---
slug: announcing-openindex
title: "Announcing OpenIndex"
date: 2026-02-15
summary: "An open-source toolkit for crawling, indexing, and querying the web. No company. No funding. Just code."
tags: [launch]
---

Last Tuesday at 2 AM I watched a recrawler hit 0.8% success rate across 50,000 workers and thought, "well, that's the entire architecture wrong." Three days later, after adding a single constraint -- 8 max connections per domain -- the success rate jumped to 57.5%. A 69x improvement from one number.

That kind of thing has been happening for months. I've been building OpenIndex since early 2026 -- a set of tools for crawling, indexing, and querying the open web. Today I'm putting all of it on [GitHub](https://github.com/nicholasgasior/gopher-crawl).

No company behind this. No funding round. No product. Just one developer with a bunch of working code and a belief that the web's largest public dataset shouldn't require a corporate badge to analyze.

## Why does this exist?

Common Crawl publishes petabytes of raw web archives. That's incredible. But raw WARC files are not useful if you can't query them, search them, or extract structure from them. OpenIndex builds the layers on top -- the indexes, the search, the knowledge extraction -- and keeps everything open.

<div class="note">
  <strong>Why open?</strong> Because the web was built in the open. A researcher, a student, a small team -- they should have the same access to web intelligence as an engineer at a search company. The largest public dataset in human history shouldn't require a corporate partnership to analyze.
</div>

Let me tell you what actually exists right now. Not a roadmap. Working code you can clone and run.

## What can you actually run today?

### The Recrawler

The recrawler takes seed URLs from Common Crawl's columnar index and fetches them live. Up to 100K concurrent HTTP workers across thousands of domains, 20K DNS workers for batch resolution, 5K probe workers that stream URLs to fetchers the instant a domain is confirmed alive.

I mentioned the 0.8% failure already. That benchmark shaped the entire design -- 50K workers against 73 domains meant roughly 685 simultaneous connections per domain. Servers just said no. Per-domain semaphores capped at 8 connections turned that disaster into something that actually works.

<pre>
<span class="hl">$ search cc recrawl --last 5 --workers 50000</span>

<span class="dim">DNS resolution</span>   <span class="blue">████████████████████</span>  <span class="hl">20K workers</span>
  resolved: <span class="green">42,891</span>  dead: <span class="amber">31,204</span>  timeout: 876

<span class="dim">Streaming probe</span>  <span class="blue">████████████████████</span>  <span class="hl">5K workers</span>
  alive: <span class="green">28,445</span>  refused: <span class="amber">14,446</span>
  <span class="dim">feeding URLs to HTTP workers as probes complete...</span>

<span class="dim">HTTP fetch</span>       <span class="blue">████████████████████</span>  <span class="hl">50K workers</span>
  <span class="green">147,231 fetched</span>  <span class="amber">23,109 failed</span>
  <span class="hl">275 pages/s</span> peak  8 max conns/domain
  rolling bandwidth: <span class="green">48.2 MB/s</span>

<span class="dim">Results</span> → 16-shard DuckDB  <span class="hl">1.2 GB</span> total
  ~/data/common-crawl/CC-MAIN-2026-04/recrawl/
</pre>

DNS uses multi-server confirmation -- Cloudflare, then Google, then stdlib -- to prevent a single flaky resolver from killing thousands of valid domains. The probe stage is conservative: a timeout means alive, only a connection refused or DNS error means dead. I'd rather try a fetch and fail fast than skip a server that's just slow.

The streaming probe-to-feed pipeline was a 3x speed improvement over the old batch approach. Previously: probe all domains, collect results, shuffle, then feed to workers. Now: push URLs immediately as probes succeed. Workers start fetching the moment the first domain is confirmed alive. Total time dropped from 185s to 65s for 2.5M URLs.

### The Domain Crawler

For deep crawling a single site, there's a separate domain crawler built around HTTP/2 multiplexing. Bloom filter frontier for URL deduplication, errgroup workers with a coordinator goroutine, resumable state in DuckDB. On kenh14.vn it peaked at 275 pages/s -- 1,380 pages in 7 seconds.

<pre>
<span class="hl">$ search crawl-domain kenh14.vn --max-pages 5000 --workers 500</span>

<span class="dim">Protocol:</span>  <span class="green">HTTP/2</span> (multiplexed)
<span class="dim">Frontier:</span>  bloom filter + channel queue
<span class="dim">Workers:</span>   500 active, 100 max connections

  pages: <span class="green">1,380</span>  links: <span class="blue">24,819</span>  errors: <span class="amber">3</span>
  peak: <span class="hl">275 pages/s</span>  elapsed: 7.1s

<span class="dim">Results</span> → ~/data/crawler/kenh14.vn/
  results/shard_00.duckdb ... shard_15.duckdb
  state.duckdb <span class="dim">(resumable)</span>
</pre>

### Sharded Storage

All crawl results land in 16-shard DuckDB databases. Batch-VALUES inserts of 500 rows per statement. Async flush. URLs distributed across shards by domain hash. For analytics, everything exports to Apache Parquet, and DuckDB's httpfs extension lets you run SQL over remote Parquet files on S3 without downloading a single byte.

### Common Crawl Integration

The CC package downloads and queries the columnar index (Parquet), CDX index, and WARC files. Smart caching with 24h TTL avoids redundant downloads. The `--remote` flag queries S3 Parquet directly via DuckDB httpfs -- zero disk. The CDX API provides direct URL lookups without any downloads at all. A bridge module extracts seed URLs from CC Parquet files and feeds them straight into the recrawler.

### The CLI

Everything runs through a Go CLI built on Cobra and Fang:

<pre>
<span class="hl">$ search --help</span>

<span class="dim">Commands:</span>
  <span class="green">serve</span>          Start the API server
  <span class="green">crawl-domain</span>   Deep-crawl a single domain
  <span class="green">cc index</span>       Download/query CC columnar index
  <span class="green">cc recrawl</span>     Recrawl URLs from CC index
  <span class="green">cc site</span>        Extract all CC pages for a domain
  <span class="green">cc url</span>         Look up a URL via CDX API
  <span class="green">download</span>       Download FineWeb datasets
  <span class="green">reddit</span>         Reddit archive tools
  <span class="green">analytics</span>      Run analytics dashboards

<span class="dim">Flags:</span>
  --workers          HTTP worker count <span class="dim">(default: 50000)</span>
  --max-conns-per-domain  <span class="dim">(default: 8)</span>
  --domain-fail-threshold <span class="dim">(default: 2)</span>
</pre>

### The API Layer

The API runs on Cloudflare Workers -- Hono and TypeScript. The CC Viewer is deployed and live: URL lookups, domain browsing, WARC viewing, crawl listing. SSR HTML, KV caching. It's the first read layer on top of the crawl data.

## How does it all fit together?

<pre>
  CC Index (Parquet)         Live Seed URLs
       |                          |
       v                          v
  +-----------------------------------------------+
  |        <span class="hl">Seed URL Extraction</span> (Go)              |
  |   read_parquet() / CDX API / manual           |
  +----------------------+------------------------+
                         |
                         v
  +-----------------------------------------------+
  |        <span class="blue">Batch DNS Resolution</span>                  |
  |   20K workers, CF → Google → stdlib           |
  +----------+-----------+------------------------+
             |           |
       resolved IPs   dead domains <span class="dim">(skip)</span>
             |
             v
  +-----------------------------------------------+
  |        <span class="blue">Streaming Probe + Feed</span>                |
  |   5K probers → immediate URL feed             |
  +----------------------+------------------------+
                         |
                         v
  +-----------------------------------------------+
  |        <span class="green">HTTP Workers</span> (up to 100K)              |
  |   8 max conns/domain, round-robin interleave  |
  +----------------------+------------------------+
                         |
                         v
  +-----------------------------------------------+
  |        <span class="amber">Sharded ResultDB</span> (DuckDB)              |
  |   16 shards, 500 rows/stmt, async flush       |
  +----------------------+------------------------+
                         |
              +----------+----------+
              v                     v
        Parquet Export       Hono API (CF Workers)
</pre>

## What's in the stack?

<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Technology</th>
      <th>Why</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Crawler</strong></td>
      <td>Go</td>
      <td>Goroutines make 100K concurrent workers trivial. No async/await ceremony.</td>
    </tr>
    <tr>
      <td><strong>Storage</strong></td>
      <td>DuckDB + Parquet</td>
      <td>Columnar analytics without a database server. Sharding for write throughput.</td>
    </tr>
    <tr>
      <td><strong>API</strong></td>
      <td>Hono + TypeScript</td>
      <td>Edge deployment on CF Workers. Sub-millisecond cold starts. KV caching.</td>
    </tr>
    <tr>
      <td><strong>Web Framework</strong></td>
      <td>Mizu (Go)</td>
      <td>Lightweight HTTP lifecycle. Built in the same ecosystem.</td>
    </tr>
    <tr>
      <td><strong>CLI</strong></td>
      <td>Cobra + Fang</td>
      <td>Composable command tree. Every pipeline stage accessible from terminal.</td>
    </tr>
    <tr>
      <td><strong>DNS</strong></td>
      <td>Custom Go resolver</td>
      <td>Multi-server confirmation prevents false positives from single resolver failures.</td>
    </tr>
    <tr>
      <td><strong>Remote Storage</strong></td>
      <td>S3-compatible + httpfs</td>
      <td>Zero-copy queries over remote Parquet. No data movement for analytics.</td>
    </tr>
  </tbody>
</table>

## Where is this going?

The crawling and storage layers work. What comes next is the hard part -- search and understanding.

**Full-text search** via Tantivy, a Rust inverted index. Fast, memory-efficient, battle-tested. This gives OpenIndex keyword search over the entire crawl corpus.

**Vector search** via Vald, a distributed approximate nearest neighbor engine. Dense embeddings per page for semantic search -- finding content by meaning, not keywords. Query "climate change policy proposals" and get results about carbon tax legislation even if those exact words don't appear.

**Knowledge graph**: entity extraction via NER, Schema.org parsing from structured data already embedded in web pages, web graph construction from link analysis. The goal is turning raw HTML into a queryable graph of entities, relationships, and connections.

**Open ontology**: a community-maintained schema for web entities. JSON-LD, RDF, OWL -- the formats exist. What's missing is a practical, evolving ontology that maps to what actually exists on the web. This is the longest-term piece, and the one that benefits most from other people's input.

## Want to help?

I built this alone but I don't want to keep building it alone. If you care about open web infrastructure:

- **Run crawls.** The more diverse the crawl data, the better. Different geographic regions surface different parts of the web.
- **Improve the pipeline.** The Go codebase is straightforward. If you see something that could be faster, more correct, or better tested -- open a PR.
- **Build on the data.** The crawl results are in DuckDB and Parquet. Build tools, analyses, or visualizations on top of them.
- **Shape the roadmap.** The knowledge graph and ontology work especially benefits from multiple perspectives. Open an issue with ideas.

The code is at [github.com/nicholasgasior/gopher-crawl](https://github.com/nicholasgasior/gopher-crawl). Clone it, run `search --help`, and see what it does. If something is confusing, that's a documentation bug -- file it.

<div class="note">
  <strong>This project is early.</strong> The crawling and storage layers are solid -- built on real benchmarks, not projections. But search, knowledge graph, and ontology are still ahead. If you've ever wanted to work on open web infrastructure from the ground up, this is the time.
</div>

The web is the largest public dataset in existence. I'd like to build the open tools to understand it, and I could use some help.
