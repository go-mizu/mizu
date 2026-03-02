---
slug: self-hosting-openindex
title: "Running Your Own Search Engine"
date: 2026-03-01
summary: "What it takes to run a search engine on a single server. Spoiler: less than you think."
tags: [roadmap, infrastructure]
---

"Search engine" conjures images of massive data centers and thousands of servers. Running one at Google scale requires that. Running one at "answer questions about 50 million pages" scale? A decent server and some patience.

This post walks through what it takes to run OpenIndex on your own hardware today -- seeding data from Common Crawl, running crawls, querying results. What works, what doesn't yet, and what's coming.

## What does "self-hosting a search engine" mean?

Not Google. Not even Bing. A focused search engine over a specific corpus: your crawl data, Common Crawl subsets, domain-specific collections. Think "search engine for .edu sites" or "search engine for everything Common Crawl captured about climate research."

That's achievable. That's useful. And it doesn't require a data center.

You pick a slice of the web, crawl it (or grab it from Common Crawl for free), index it, and query it. The entire pipeline runs on a single machine. Scale up when you need to, but start small.

## What's in the stack?

Here's what exists today and what's planned:

<pre>
  <span style="color:#4ade80">Working today</span>                        <span style="color:#fbbf24">Coming soon</span>
  +-----------------+                  +-----------------+
  | <span style="color:#4ade80">Go CLI</span>          |                  | <span style="color:#fbbf24">Tantivy</span>         |
  |  recrawler      |                  |  full-text      |
  |  domain crawler |                  |  search index   |
  |  CC integration |                  +-----------------+
  |  data import    |                  +-----------------+
  +-----------------+                  | <span style="color:#fbbf24">Vald</span>            |
  +-----------------+                  |  vector search  |
  | <span style="color:#4ade80">DuckDB</span>          |                  |  embeddings     |
  |  16-shard store |                  +-----------------+
  |  SQL analytics  |                  +-----------------+
  |  Parquet I/O    |                  | <span style="color:#fbbf24">Knowledge Graph</span> |
  +-----------------+                  |  entities       |
  +-----------------+                  |  relationships  |
  | <span style="color:#4ade80">CF Workers</span>      |                  +-----------------+
  |  CC Viewer      |                  +-----------------+
  |  search API     |                  | <span style="color:#fbbf24">OQL</span>             |
  +-----------------+                  |  query language  |
                                       +-----------------+
</pre>

The Go CLI handles all crawling and data management. DuckDB stores everything in 16 sharded files with SQL access. Cloudflare Workers run the CC Viewer and API layer. Future pieces -- Tantivy for text search, Vald for vectors, a knowledge graph -- will plug into the same data pipeline.

## Docker Compose: one command to start

The target deployment is `docker compose up`. Here's what that will look like:

<pre><code><span style="color:#888"># docker-compose.yml (planned)</span>
<span style="color:#60a5fa">services</span>:
  <span style="color:#4ade80">cli</span>:
    build: .
    volumes:
      - ./data:/data
    ports:
      - <span style="color:#fbbf24">"8080:8080"</span>
    command: search serve

  <span style="color:#4ade80">tantivy</span>:  <span style="color:#888"># coming soon</span>
    image: openindex/tantivy
    volumes:
      - ./data/index:/index
    ports:
      - <span style="color:#fbbf24">"8081:8081"</span>

  <span style="color:#4ade80">vald</span>:  <span style="color:#888"># coming soon</span>
    image: vdaas/vald
    ports:
      - <span style="color:#fbbf24">"8082:8082"</span></code></pre>

<div class="note">
  <strong>Docker deployment isn't ready yet.</strong> The Go CLI runs natively today. Docker packaging is on the roadmap. The compose file above shows the planned architecture -- you can run everything except the sidecar services right now by building the CLI directly.
</div>

Today you build and run the CLI natively:

<pre><code><span style="color:#e0e0e0">$ make install</span>
<span style="color:#888"># Binary lands at $HOME/bin/mizu</span>

<span style="color:#e0e0e0">$ search --help</span></code></pre>

## Seeding with Common Crawl

Don't start with an empty index. Common Crawl gives you billions of URLs with metadata for free. Two commands get you from zero to a million pages in DuckDB:

<pre>
<span style="color:#e0e0e0">$ search cc index --sample 5</span>

<span style="color:#888">Downloading 5 evenly-spaced parquet files from CC-MAIN-2026-04...</span>
  file 0/300   <span style="color:#4ade80">done</span>  <span style="color:#888">42 MB</span>
  file 60/300  <span style="color:#4ade80">done</span>  <span style="color:#888">38 MB</span>
  file 120/300 <span style="color:#4ade80">done</span>  <span style="color:#888">41 MB</span>
  file 180/300 <span style="color:#4ade80">done</span>  <span style="color:#888">39 MB</span>
  file 240/300 <span style="color:#4ade80">done</span>  <span style="color:#888">44 MB</span>

<span style="color:#888">Stored at ~/data/common-crawl/CC-MAIN-2026-04/</span>

<span style="color:#e0e0e0">$ search cc recrawl --last 5 --workers 50000</span>

<span style="color:#888">DNS resolution</span>   <span style="color:#60a5fa">████████████████████</span>  <span style="color:#e0e0e0">20K workers</span>
  resolved: <span style="color:#4ade80">42,891</span>  dead: <span style="color:#fbbf24">31,204</span>  timeout: 876

<span style="color:#888">Streaming probe</span>  <span style="color:#60a5fa">████████████████████</span>  <span style="color:#e0e0e0">5K workers</span>
  alive: <span style="color:#4ade80">28,445</span>  refused: <span style="color:#fbbf24">14,446</span>

<span style="color:#888">HTTP fetch</span>       <span style="color:#60a5fa">████████████████████</span>  <span style="color:#e0e0e0">50K workers</span>
  <span style="color:#4ade80">147,231 fetched</span>  <span style="color:#fbbf24">23,109 failed</span>
  <span style="color:#e0e0e0">275 pages/s</span> peak  8 max conns/domain
  rolling bandwidth: <span style="color:#4ade80">48.2 MB/s</span>

<span style="color:#888">Results</span> → 16-shard DuckDB  <span style="color:#e0e0e0">1.2 GB</span> total
  ~/data/common-crawl/CC-MAIN-2026-04/recrawl/
</pre>

The `--sample 5` flag downloads five evenly-spaced parquet files from the CC index, giving a representative TLD distribution. The recrawler then takes those seed URLs and fetches them live -- batch DNS, streaming probes, 50K HTTP workers with per-domain rate limiting.

Alternatively, skip the download entirely. The `--remote` flag queries CC's S3 parquet files directly via DuckDB httpfs -- zero disk, zero download:

<pre><code><span style="color:#e0e0e0">$ search cc stats --remote</span>
<span style="color:#888"># Runs SQL over S3 parquet via HTTP range requests</span></code></pre>

## Running your first crawl

Two options, depending on whether you want breadth or depth.

**Breadth: recrawl thousands of domains quickly.** The recrawler pulls seed URLs from CC data and fetches them across thousands of domains simultaneously. Good for building a broad index fast.

**Depth: crawl one site thoroughly.** The domain crawler uses HTTP/2 multiplexing to crawl a single domain at high speed. Bloom filter frontier, resumable state, sharded results.

<pre>
<span style="color:#888"># Depth: crawl a single domain</span>
<span style="color:#e0e0e0">$ search crawl-domain duckdb.org --max-pages 5000 --workers 500</span>

<span style="color:#888">Protocol:</span>  <span style="color:#4ade80">HTTP/2</span> (multiplexed)
<span style="color:#888">Frontier:</span>  bloom filter + channel queue
<span style="color:#888">Workers:</span>   500 active, 100 max connections

  pages: <span style="color:#4ade80">1,110</span>  links: <span style="color:#60a5fa">110,442</span>  errors: <span style="color:#fbbf24">0</span>
  peak: <span style="color:#e0e0e0">193 pages/s</span>  elapsed: 7.1s

<span style="color:#888">Results</span> → ~/data/crawler/duckdb.org/
  results/shard_00.duckdb ... shard_15.duckdb
  state.duckdb <span style="color:#888">(resumable)</span>
</pre>

Both approaches produce the same output format: 16-shard DuckDB files with full SQL queryability.

## Querying what you've got

Today, queries go through DuckDB SQL via the CLI. The `search cc query` and `search cc stats` commands run analytics over your crawl data:

<pre><code><span style="color:#888"># Top domains by page count in your recrawl data</span>
<span style="color:#e0e0e0">$ search cc stats</span>

<span style="color:#888">  Domain                  Pages    Avg Size    Success%</span>
  <span style="color:#4ade80">en.wikipedia.org</span>        12,841   48.2 KB     94.1%
  <span style="color:#4ade80">stackoverflow.com</span>        8,293   31.7 KB     91.3%
  <span style="color:#4ade80">github.com</span>               6,104   22.4 KB     88.7%
  <span style="color:#888">...</span>

<span style="color:#888"># Or run arbitrary SQL</span>
<span style="color:#e0e0e0">$ search cc query "SELECT domain, COUNT(*) as pages \</span>
<span style="color:#e0e0e0">    FROM pages WHERE status = 200 \</span>
<span style="color:#e0e0e0">    GROUP BY domain ORDER BY pages DESC LIMIT 10"</span></code></pre>

The CC Viewer at [cc-viewer.go-mizu.workers.dev](https://cc-viewer.go-mizu.workers.dev) adds a web UI for browsing CC data -- URL lookups, domain browsing, WARC viewing. That's already live.

What's missing: full-text search. You can't yet type "climate change policy" and get ranked results. That's the Tantivy integration on the roadmap. For now, SQL gives you everything you need for analytics and data exploration, but keyword search requires building the inverted index first.

## Hardware requirements

Be realistic about what each tier can handle:

<table>
  <thead>
    <tr>
      <th>Tier</th>
      <th>RAM</th>
      <th>Disk</th>
      <th>Pages</th>
      <th>Good for</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Laptop</strong></td>
      <td>8 GB</td>
      <td>50 GB SSD</td>
      <td>~1M</td>
      <td>Testing, single-domain crawls, learning the CLI</td>
    </tr>
    <tr>
      <td><strong>Server</strong></td>
      <td>32 GB</td>
      <td>500 GB SSD</td>
      <td>~50M</td>
      <td>Production recrawls, multi-domain indexes, analytics</td>
    </tr>
    <tr>
      <td><strong>Cluster</strong></td>
      <td>128 GB+</td>
      <td>Multi-TB NVMe</td>
      <td>1B+</td>
      <td>Full CC subsets, vector search (Vald), knowledge graph</td>
    </tr>
  </tbody>
</table>

Where the resources go:

<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Bottleneck</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>DuckDB</strong></td>
      <td>CPU + disk I/O</td>
      <td>SSD strongly recommended. Analytical queries are CPU-bound.</td>
    </tr>
    <tr>
      <td><strong>Recrawler</strong></td>
      <td>Network + file descriptors</td>
      <td>50K workers need ~50K open sockets. Bump <code>ulimit -n</code>.</td>
    </tr>
    <tr>
      <td><strong>Tantivy</strong> <span style="color:#888">(planned)</span></td>
      <td>Disk + CPU</td>
      <td>Inverted index is disk-heavy. Merges are CPU-heavy.</td>
    </tr>
    <tr>
      <td><strong>Vald</strong> <span style="color:#888">(planned)</span></td>
      <td>RAM</td>
      <td>Vector indexes live in memory. Budget ~1 KB per vector.</td>
    </tr>
  </tbody>
</table>

<div class="note">
  <strong>Start small.</strong> A laptop with 8 GB of RAM can run a meaningful crawl of a few thousand domains. You don't need the cluster tier to get value -- that's for when you want to index a significant chunk of Common Crawl or run vector search over millions of embeddings.
</div>

## What works today vs what's coming

<table>
  <thead>
    <tr>
      <th>Capability</th>
      <th style="color:#4ade80">Today</th>
      <th style="color:#fbbf24">Coming</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Crawling</strong></td>
      <td>Recrawler (100K workers), domain crawler (HTTP/2)</td>
      <td>Distributed crawling across nodes</td>
    </tr>
    <tr>
      <td><strong>Data source</strong></td>
      <td>Common Crawl (CDX, Parquet, WARC, site extractor)</td>
      <td>Custom seed lists, sitemaps</td>
    </tr>
    <tr>
      <td><strong>Storage</strong></td>
      <td>16-shard DuckDB, Parquet export/import</td>
      <td>S3-backed archival, incremental updates</td>
    </tr>
    <tr>
      <td><strong>Query</strong></td>
      <td>SQL via CLI, CC Viewer web UI</td>
      <td>OQL (unified query language)</td>
    </tr>
    <tr>
      <td><strong>Text search</strong></td>
      <td>--</td>
      <td>Tantivy inverted index</td>
    </tr>
    <tr>
      <td><strong>Vector search</strong></td>
      <td>--</td>
      <td>Vald (ANN), per-page embeddings</td>
    </tr>
    <tr>
      <td><strong>Knowledge graph</strong></td>
      <td>--</td>
      <td>Entity extraction, Schema.org, link graph</td>
    </tr>
    <tr>
      <td><strong>Deployment</strong></td>
      <td>Native binary, CF Workers</td>
      <td>Docker Compose, Helm charts</td>
    </tr>
  </tbody>
</table>

The crawling and storage layers are solid -- built on real benchmarks, not estimates. The search and understanding layers are what's next.

## Why self-host?

**Control.** Your data stays on your hardware. No third-party indexing your crawl results. No terms of service governing what you can analyze.

**No rate limits.** Commercial search APIs charge per query and cap throughput. Your own index has no artificial limits -- query it a million times if you want.

**Reproducibility.** Research requires reproducible results. A local index with versioned crawl data gives you that. A cloud API that updates weekly doesn't.

**Customization.** Crawl exactly what you care about. Skip what you don't. Weight domains differently. Build a search engine that answers *your* questions, not everyone's.

<div class="note">
  <strong>How does this compare to alternatives?</strong> Common Crawl gives you the raw data but no search layer. Commercial APIs (Google Custom Search, Bing Web Search) give you search but no control, and they get expensive at scale. SearXNG aggregates other engines' results but doesn't maintain its own index. OpenIndex is the missing piece: your data, your index, your queries, your hardware.
</div>

The code is at [github.com/nicholasgasior/gopher-crawl](https://github.com/nicholasgasior/gopher-crawl). Clone it, run `search cc index --sample 5`, and you'll have Common Crawl data in DuckDB in under five minutes. Run `search cc recrawl --last 5` and you'll have live-fetched pages in under an hour. That's a search engine -- a small one, but yours.
