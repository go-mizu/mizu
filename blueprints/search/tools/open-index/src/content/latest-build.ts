import { icons, cardIcon } from '../icons'

export const latestBuildPage = `
<h2>Current Status</h2>
<p>OpenIndex is in its initial data collection phase. The first crawls are underway, seeded from Common Crawl data and processed through the Go recrawler pipeline.</p>

<div class="note">
  <strong>Early stage.</strong> OpenIndex does not yet have its own complete web crawl. The data available today comes from recrawling Common Crawl seed URLs and targeted domain crawls. Numbers will grow as the pipeline matures.
</div>

<div class="stats" style="max-width:100%">
  <div class="stat">
    <div class="stat-v">Active</div>
    <div class="stat-l">Crawl Status</div>
  </div>
  <div class="stat">
    <div class="stat-v">CC-MAIN-2026-04</div>
    <div class="stat-l">Seed Source</div>
  </div>
  <div class="stat">
    <div class="stat-v">Go + DuckDB</div>
    <div class="stat-l">Pipeline</div>
  </div>
  <div class="stat">
    <div class="stat-v">Parquet</div>
    <div class="stat-l">Index Format</div>
  </div>
</div>

<h2>Data Sources</h2>

<h3>Common Crawl Seed Data</h3>
<p>OpenIndex builds on <a href="https://commoncrawl.org">Common Crawl</a>, the largest open web archive. The current seed source is <strong>CC-MAIN-2026-04</strong>.</p>

<table>
  <thead>
    <tr>
      <th>Source</th>
      <th>Description</th>
      <th>Access</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>CC Columnar Index</strong></td>
      <td>Parquet files with URL metadata for billions of pages</td>
      <td>S3 remote query via DuckDB httpfs</td>
    </tr>
    <tr>
      <td><strong>CC CDX Index</strong></td>
      <td>URL-to-WARC record lookup</td>
      <td>CDX API (zero disk)</td>
    </tr>
    <tr>
      <td><strong>CC WARC Files</strong></td>
      <td>Raw HTTP responses</td>
      <td>S3 byte-range requests</td>
    </tr>
    <tr>
      <td><strong>CC Web Graphs</strong></td>
      <td>Host-level and domain-level link graphs</td>
      <td>S3 download</td>
    </tr>
  </tbody>
</table>

<h3>Recrawl Data</h3>
<p>Seed URLs extracted from CC index are recrawled live through the Go pipeline to verify liveness and capture fresh content.</p>

<pre><code># Extract seed URLs from CC parquet and recrawl
search cc recrawl --last

# Or target a specific CC parquet file
search cc recrawl --file 50

# Results stored locally:
~/data/common-crawl/CC-MAIN-2026-04/recrawl/
  shard_00.duckdb ... shard_15.duckdb

# DNS cache shared across runs:
~/data/common-crawl/CC-MAIN-2026-04/dns.duckdb</code></pre>

<h3>Domain Crawl Data</h3>
<p>Targeted domain crawls produce deep coverage of specific sites.</p>

<pre><code># Crawl a domain
search crawl-domain example.com --max-pages 5000

# Results:
~/data/crawler/example.com/
  results/shard_00.duckdb ... shard_15.duckdb
  state.duckdb  # resumable state</code></pre>

<h2>Data Formats Available</h2>

<details>
  <summary>DuckDB Sharded Results</summary>
  <div class="details-body">
    <p>Primary storage format. 16-shard DuckDB with batch-VALUES inserts.</p>
    <table>
      <thead>
        <tr><th>Property</th><th>Value</th></tr>
      </thead>
      <tbody>
        <tr><td>Shards</td><td>16 DuckDB files</td></tr>
        <tr><td>Insert strategy</td><td>Batch-VALUES, 500 rows/stmt</td></tr>
        <tr><td>Distribution</td><td>URL hash</td></tr>
        <tr><td>Location</td><td><code>~/data/common-crawl/{CrawlID}/recrawl/</code></td></tr>
      </tbody>
    </table>
    <pre><code># Query results with DuckDB CLI
duckdb ~/data/common-crawl/CC-MAIN-2026-04/recrawl/shard_00.duckdb \\
  "SELECT url, status_code, content_type FROM results LIMIT 10"</code></pre>
  </div>
</details>

<details>
  <summary>Parquet (via CC Index)</summary>
  <div class="details-body">
    <p>Common Crawl's columnar index is queryable directly from S3 without downloading.</p>
    <pre><code># Remote query -- zero disk, zero download
duckdb -c "
  INSTALL httpfs; LOAD httpfs;
  SELECT url_host_tld, count(*) as pages
  FROM read_parquet(
    's3://commoncrawl/cc-index/table/cc-main/warc/crawl=CC-MAIN-2026-04/subset=warc/*.parquet'
  )
  GROUP BY url_host_tld
  ORDER BY pages DESC
  LIMIT 10;
"</code></pre>
  </div>
</details>

<details>
  <summary>DNS Cache</summary>
  <div class="details-body">
    <p>Three-category DNS cache persisted in DuckDB. Shared across crawl runs.</p>
    <table>
      <thead>
        <tr><th>Category</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr><td><strong>Resolved</strong></td><td>Domain resolved to IP addresses</td></tr>
        <tr><td><strong>Dead</strong></td><td>NXDOMAIN confirmed by multiple DNS servers</td></tr>
        <tr><td><strong>Timeout</strong></td><td>DNS timeout, saved for retry</td></tr>
      </tbody>
    </table>
    <pre><code># DNS cache location
~/data/common-crawl/CC-MAIN-2026-04/dns.duckdb</code></pre>
  </div>
</details>

<details>
  <summary>WARC Files (via CC)</summary>
  <div class="details-body">
    <p>Raw HTTP responses accessible via Common Crawl's CDN using byte-range requests.</p>
    <pre><code># Fetch a specific WARC record by byte range
search cc fetch --url https://example.com

# Or look up via CDX API
search cc url https://example.com</code></pre>
  </div>
</details>

<h2>Accessing the Data</h2>

<h3>CLI Tool</h3>
<pre><code># Install
go install github.com/nicholasgasior/gopher-crawl/cmd/search@latest

# List available CC crawls
search cc crawls

# Query CC index remotely
search cc stats --remote

# Download CC parquet sample
search cc index --sample 5

# Recrawl from CC seeds
search cc recrawl --last

# Crawl a single domain
search crawl-domain example.com</code></pre>

<h3>CC Viewer (Web)</h3>
<p>Browse Common Crawl data through the web viewer:</p>
<pre><code># Deployed at:
https://cc-viewer.go-mizu.workers.dev

# Routes:
/url/*         -- Look up a URL in CC
/domain/:name  -- Browse a domain
/view          -- WARC record viewer
/crawls        -- List available crawls</code></pre>

<h3>DuckDB Direct</h3>
<pre><code># Query local recrawl results
duckdb ~/data/common-crawl/CC-MAIN-2026-04/recrawl/shard_00.duckdb

# Query CC index remotely (requires httpfs)
duckdb -c "INSTALL httpfs; LOAD httpfs;
  SELECT count(*) FROM read_parquet('s3://commoncrawl/cc-index/...');"</code></pre>

<h2>What Is Next</h2>
<table>
  <thead>
    <tr>
      <th>Milestone</th>
      <th>Description</th>
      <th>Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>CC seed recrawl</strong></td>
      <td>Recrawl seed URLs from CC-MAIN-2026-04 parquet index</td>
      <td>In progress</td>
    </tr>
    <tr>
      <td><strong>Domain crawl coverage</strong></td>
      <td>Deep crawls of selected domains via domain crawler</td>
      <td>In progress</td>
    </tr>
    <tr>
      <td><strong>Parquet export</strong></td>
      <td>Export recrawl results to Parquet for sharing</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>Own CDX index</strong></td>
      <td>Produce OpenIndex CDX from own crawl data</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>Full-text index</strong></td>
      <td>Tantivy-based keyword search</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>Vector embeddings</strong></td>
      <td>Generate embeddings, deploy Vald</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>Knowledge graph</strong></td>
      <td>NER pipeline, entity extraction</td>
      <td>Planned</td>
    </tr>
  </tbody>
</table>

<div class="note">
  <strong>Growing dataset.</strong> OpenIndex is a new project. The dataset is small and growing. If you need large-scale web data today, <a href="https://commoncrawl.org">Common Crawl</a> is the established source -- and OpenIndex integrates with it directly.
</div>
`
