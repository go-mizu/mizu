import { icons, cardIcon } from '../icons'

export const architecturePage = `
<h2>System Architecture</h2>
<p>OpenIndex is built as a pipeline: Common Crawl data and live recrawls flow through Go services into sharded DuckDB storage, exposed via a Hono API on Cloudflare Workers. The architecture reflects what is actually built and running today.</p>

<h3>Pipeline Overview</h3>
<pre><code>  CC Index (Parquet)          Seed Lists / Sitemaps
       |                              |
       v                              v
  +------------------------------------------+
  |        Seed URL Extraction (Go)          |
  |   read_parquet() / CDX API / manual      |
  +--------------------+---------------------+
                       |
                       v
  +------------------------------------------+
  |         Batch DNS Resolution             |
  |   20K workers, multi-server confirm      |
  |   CF -> Google -> stdlib fallback        |
  +--------------------+---------------------+
                       |
           +-----------+-----------+
           |                       |
     resolved IPs            dead domains
           |                  (skip)
           v
  +------------------------------------------+
  |        Streaming Probe + Feed            |
  |   5K probers, immediate URL feed         |
  |   round-robin domain interleaving        |
  +--------------------+---------------------+
                       |
                       v
  +------------------------------------------+
  |          HTTP Workers (100K)             |
  |   8 max conns/domain, 3s timeout         |
  |   per-domain semaphores                  |
  +--------------------+---------------------+
                       |
                       v
  +------------------------------------------+
  |        Sharded ResultDB (DuckDB)         |
  |   16 shards, batch-VALUES inserts        |
  |   500 rows/stmt, async flush             |
  +--------------------+---------------------+
                       |
                       v
  +------------------------------------------+
  |          Parquet Export + API             |
  |   Columnar analytics, CF Worker API      |
  +------------------------------------------+</code></pre>

<h2>Core Components</h2>
<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('globe')} <span>Recrawler</span></div>
    <h3>pkg/recrawler</h3>
    <p>High-throughput URL recrawler. Batch DNS with 20K workers, streaming probe with 5K workers that immediately feed resolved URLs to 100K HTTP workers. Per-domain connection limiting prevents flooding. sync.Map for dead domains, 64 transport shards, 500ms TLS timeout.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('cpu')} <span>Domain Crawler</span></div>
    <h3>pkg/dcrawler</h3>
    <p>Single-domain crawler using HTTP/2 multiplexing. errgroup workers + coordinator goroutine. Bloom filter frontier for URL deduplication. Sharded DuckDB for results, resumable state via state.duckdb. 275 pages/s peak.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('database')} <span>Common Crawl</span></div>
    <h3>pkg/cc</h3>
    <p>Downloads CC columnar index (parquet), CDX index, WARC files. Smart caching with 24h TTL. Remote S3 queries via DuckDB httpfs (zero disk). CDX API for direct URL lookup. Bridge to recrawler via ExtractSeedURLsFromParquet().</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('zap')} <span>API Layer</span></div>
    <h3>tools/cc-viewer</h3>
    <p>Hono + Cloudflare Worker. SSR HTML with KV cache. Routes: URL lookup, domain browsing, WARC viewer, crawl listing. Deployed at cc-viewer.go-mizu.workers.dev.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('terminal')} <span>CLI</span></div>
    <h3>cmd/search</h3>
    <p>Go CLI built on Cobra + Fang. Subcommands: serve, init, seed, crawl, crawl-domain, cc, download, recrawl, reddit, analytics. Full pipeline control from the terminal.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('package')} <span>URL Fetcher</span></div>
    <h3>tools/url-fetcher</h3>
    <p>Cloudflare Worker for batch URL fetching from edge. POST /fetch with max 500 URLs, Bearer auth, Promise.allSettled. 386 URLs/s peak throughput.</p>
  </div>
</div>

<h2>Technology Stack</h2>
<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Technology</th>
      <th>Purpose</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Crawler</strong></td>
      <td>Go</td>
      <td>Recrawler (100K workers) + domain crawler (HTTP/2)</td>
    </tr>
    <tr>
      <td><strong>Analytics DB</strong></td>
      <td>DuckDB</td>
      <td>16-shard storage, SQL queries over Parquet, batch inserts</td>
    </tr>
    <tr>
      <td><strong>Columnar Index</strong></td>
      <td>Apache Parquet</td>
      <td>Columnar storage for high-throughput analytics</td>
    </tr>
    <tr>
      <td><strong>API</strong></td>
      <td>Hono (TypeScript)</td>
      <td>Edge API on Cloudflare Workers, SSR, KV cache</td>
    </tr>
    <tr>
      <td><strong>Web Framework</strong></td>
      <td>Mizu (Go)</td>
      <td>HTTP server lifecycle, routing, middleware</td>
    </tr>
    <tr>
      <td><strong>CLI</strong></td>
      <td>Cobra + Fang (Go)</td>
      <td>Command-line interface for all operations</td>
    </tr>
    <tr>
      <td><strong>DNS</strong></td>
      <td>Custom Go resolver</td>
      <td>Multi-server confirmation (CF, Google, stdlib)</td>
    </tr>
    <tr>
      <td><strong>Object Storage</strong></td>
      <td>S3-compatible</td>
      <td>WARC files, Parquet files via httpfs</td>
    </tr>
  </tbody>
</table>

<h2>Data Flow: CC Recrawl Pipeline</h2>
<p>The primary data pipeline today is the CC recrawl: extract seed URLs from Common Crawl, then recrawl them live.</p>

<pre><code># 1. Download CC parquet index (or query remote)
search cc index --sample 5

# 2. Extract seed URLs directly from parquet
#    Uses read_parquet() -- zero DuckDB import
search cc recrawl --last

# 3. Pipeline runs automatically:
#    Batch DNS (20K workers)
#    -> Streaming probe (5K workers, immediate feed)
#    -> HTTP workers (50K, 8 max conns/domain)
#    -> Sharded ResultDB (16 DuckDB files)

# Results stored at:
# ~/data/common-crawl/{CrawlID}/recrawl/  (16-shard DuckDB)
# ~/data/common-crawl/{CrawlID}/dns.duckdb (shared DNS cache)</code></pre>

<h2>Storage Architecture</h2>
<h3>Sharded DuckDB</h3>
<p>ResultDB uses 16 DuckDB shards for concurrent write throughput. Each shard receives batch-VALUES inserts of 500 rows per statement. URLs are distributed across shards by hash.</p>

<pre><code>~/data/common-crawl/CC-MAIN-2026-04/recrawl/
  shard_00.duckdb
  shard_01.duckdb
  ...
  shard_15.duckdb

~/data/crawler/{domain}/
  results/
    shard_00.duckdb ... shard_15.duckdb
  state.duckdb</code></pre>

<h3>DNS Cache</h3>
<p>Three-category DNS cache persisted in DuckDB: resolved (IPs), dead (NXDOMAIN), timeout (saved for reuse). Multi-server confirmation prevents false positives. HTTP failures never contaminate the DNS cache.</p>

<h2>Key Design Decisions</h2>
<div class="cards">
  <div class="card">
    <h3>Per-domain connection limiting</h3>
    <p>50K workers / 73 domains = 685 conns/domain, yielding 0.8% success. Adding 8 max conns/domain raised success to 57.5% -- a 69x improvement. Pre-created domain semaphores distribute load.</p>
  </div>
  <div class="card">
    <h3>Streaming probe-to-feed</h3>
    <p>Old: probe ALL domains, collect, shuffle, feed (sequential). New: probe in parallel, push URLs immediately, workers start instantly. 3x faster total time (65s vs 185s for 2.5M URLs).</p>
  </div>
  <div class="card">
    <h3>URL interleaving</h3>
    <p>Round-robin across domains instead of sequential per-domain. Prevents hammering a single domain when workers are fast. Distributes load evenly across the domain pool.</p>
  </div>
  <div class="card">
    <h3>Batch DNS first, then crawl</h3>
    <p>Running DNS pipeline and HTTP workers simultaneously causes goroutine explosion. Batch DNS resolves all domains upfront, sets cache + dead domains, then feeds URLs to workers.</p>
  </div>
</div>

<div class="note">
  <strong>Built on real benchmarks.</strong> Every architectural decision above came from running actual crawls and measuring results. The numbers in this document are from real test runs, not projections.
</div>
`
