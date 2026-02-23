import { icons, cardIcon } from '../icons'

export const roadmapPage = `
<h2>Roadmap</h2>
<p>What has been built, what is in progress, and what comes next. This reflects the actual state of the project.</p>

<hr>

<h3>Done</h3>

<div class="timeline">
  <div class="timeline-item done">
    <div class="timeline-date">2026</div>
    <h3>Go CLI Framework</h3>
    <p>Cobra + Fang CLI with commands for crawling, downloading, indexing, and analytics. Entry point for all OpenIndex operations.</p>
  </div>
  <div class="timeline-item done">
    <div class="timeline-date">2026</div>
    <h3>Common Crawl Integration</h3>
    <p>Full <code>pkg/cc</code> package: columnar index download, CDX API queries, WARC fetching, smart caching, remote S3 Parquet queries via DuckDB httpfs.</p>
  </div>
  <div class="timeline-item done">
    <div class="timeline-date">2026</div>
    <h3>High-Throughput Recrawler</h3>
    <p><code>pkg/recrawler</code>: 100K HTTP workers, 20K DNS workers, batch DNS resolution, streaming probe-and-feed pipeline, per-domain rate limiting.</p>
  </div>
  <div class="timeline-item done">
    <div class="timeline-date">2026</div>
    <h3>Domain Crawler</h3>
    <p><code>pkg/dcrawler</code>: single-domain crawler with HTTP/2 multiplexing, bloom filter frontier, errgroup workers, and coordinator pattern.</p>
  </div>
  <div class="timeline-item done">
    <div class="timeline-date">2026</div>
    <h3>Sharded DuckDB Storage</h3>
    <p>16-shard DuckDB result storage with batch-VALUES inserts. Shared pattern across recrawler, domain crawler, and CC site extractor.</p>
  </div>
  <div class="timeline-item done">
    <div class="timeline-date">2026</div>
    <h3>Parquet Columnar Index</h3>
    <p>Parquet files for URL metadata, content hashes, and crawl data. Queryable with DuckDB, zero-copy reads via Apache Arrow.</p>
  </div>
  <div class="timeline-item done">
    <div class="timeline-date">2026</div>
    <h3>Search Worker</h3>
    <p>Hono-based Cloudflare Worker with meta-search capabilities. Deployed at the edge with low latency worldwide.</p>
  </div>
  <div class="timeline-item done">
    <div class="timeline-date">2026</div>
    <h3>70+ Search Engine Adapters</h3>
    <p>Engine adapters extending <code>BaseEngine</code> with <code>buildRequest</code> + <code>parseResponse</code> pattern. Each adapter tested with vitest.</p>
  </div>
</div>

<hr>

<h3>In Progress</h3>

<div class="timeline">
  <div class="timeline-item">
    <div class="timeline-date">2026</div>
    <h3>Knowledge Graph Pipeline</h3>
    <p>Entity extraction, relationship mapping, and graph construction from crawled web content. Link graph analysis is working; NER pipeline is in design.</p>
  </div>
  <div class="timeline-item">
    <div class="timeline-date">2026</div>
    <h3>Vector Search (Vald)</h3>
    <p>Dense embedding generation and semantic similarity search via Vald distributed vector database. Integration architecture defined.</p>
  </div>
  <div class="timeline-item">
    <div class="timeline-date">2026</div>
    <h3>Open Ontology Design</h3>
    <p>Community-maintainable entity type schema. Schema.org compatible. Defining core types: Person, Organization, Place, CreativeWork.</p>
  </div>
  <div class="timeline-item">
    <div class="timeline-date">2026</div>
    <h3>CDX Index</h3>
    <p>URL-to-record lookup index for crawled pages. Compatible with the standard CDX format used by web archives.</p>
  </div>
</div>

<hr>

<h3>Next</h3>

<div class="timeline">
  <div class="timeline-item next">
    <div class="timeline-date">Future</div>
    <h3>Full-Text Search (Tantivy)</h3>
    <p>BM25-based full-text search index across the crawled corpus. Tantivy for Rust-native indexing with language-aware tokenization.</p>
  </div>
  <div class="timeline-item next">
    <div class="timeline-date">Future</div>
    <h3>OQL Query Language</h3>
    <p>A SQL-like query language for combining full-text search, vector similarity, graph traversals, and metadata filters in a single query.</p>
  </div>
  <div class="timeline-item next">
    <div class="timeline-date">Future</div>
    <h3>Public API</h3>
    <p>REST API for search, URL lookup, domain browsing, and data access. Free and open.</p>
  </div>
  <div class="timeline-item next">
    <div class="timeline-date">Future</div>
    <h3>Self-Hosted Deployment</h3>
    <p>Run the complete OpenIndex stack on your own infrastructure. Docker Compose and documentation for local and cloud deployment.</p>
  </div>
</div>

<hr>

<p>This roadmap is honest about where things stand. Items move from "Next" to "In Progress" to "Done" as work happens. If you want to accelerate something, <a href="/contributing">contribute</a>.</p>
`
