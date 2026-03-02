import { icons, cardIcon } from '../icons'

export const getStartedPage = `
<h2>Get Started with OpenIndex</h2>
<p>OpenIndex provides tools for crawling, indexing, and querying web data. There are three ways to get started: the CLI, the API (alpha), or direct data access.</p>

<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('terminal')} CLI Tool</div>
    <p>Install the Go CLI and start crawling, querying Common Crawl, and building local indexes.</p>
    <a href="#cli" class="card-lk">Jump to CLI ${icons.arrowRight}</a>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('code')} API (Alpha)</div>
    <p>HTTP endpoints for querying crawl data. Currently in early alpha -- expect breaking changes.</p>
    <a href="#api" class="card-lk">Jump to API ${icons.arrowRight}</a>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('database')} Raw Data</div>
    <p>Access Common Crawl archives and query Parquet indexes directly with DuckDB.</p>
    <a href="#data" class="card-lk">Jump to Data ${icons.arrowRight}</a>
  </div>
</div>

<hr>

<h2 id="cli">CLI Installation</h2>
<p>The CLI is written in Go. Install it directly:</p>

<pre><code>go install github.com/nicholasgasior/gopher-crawl/cmd/search@latest</code></pre>

<p>Verify the installation:</p>

<pre><code>search --help</code></pre>

<h3>Common Crawl Commands</h3>
<p>Query and download Common Crawl index data:</p>

<pre><code># List available Common Crawl crawls
search cc crawls

# Query the CC columnar index (remote, zero disk)
search cc query --remote --sql "SELECT url, status FROM read_parquet(...) LIMIT 10"

# Download CC index parquet files (sample of 5 evenly-spaced files)
search cc index --sample 5

# Look up a URL in the CC CDX API (zero download)
search cc url https://example.com

# View CC index statistics
search cc stats

# Fetch WARC records by byte-range
search cc fetch</code></pre>

<h3>Domain Crawling</h3>
<p>Crawl a single domain with HTTP/2 multiplexing:</p>

<pre><code># Crawl a domain (default: 500 workers, 100 max connections)
search crawl-domain example.com --max-pages 1000

# Resume a previous crawl
search crawl-domain example.com --resume

# Adjust concurrency
search crawl-domain example.com --workers 200 --max-conns 50</code></pre>

<p>Results are stored in sharded DuckDB files at <code>$HOME/data/crawler/{domain}/</code>.</p>

<h3>CC Site Extraction</h3>
<p>Extract all pages for a domain from Common Crawl archives:</p>

<pre><code># Get URLs only (CDX API, fast)
search cc site example.com --mode urls

# Get URLs + extracted links (WARC parsing)
search cc site example.com --mode links

# Get full content + links
search cc site example.com --mode full</code></pre>

<h3>Recrawling</h3>
<p>Recrawl URLs from Common Crawl index with high throughput:</p>

<pre><code># Recrawl from the latest CC parquet file
search cc recrawl --last

# Recrawl from a specific file
search cc recrawl --file 50

# Tune concurrency (default: 50K workers, 8 max conns/domain)
search cc recrawl --last --max-conns-per-domain 8 --domain-fail-threshold 2</code></pre>

<hr>

<h2 id="api">API (Alpha)</h2>

<div class="note note-warn">
  The API is in early alpha. Endpoints may change without notice. It is deployed as a Cloudflare Worker and is not yet publicly documented with stable URLs.
</div>

<p>The CC Viewer worker provides basic web access to Common Crawl data:</p>

<pre><code># Look up a URL in Common Crawl
curl "https://cc-viewer.go-mizu.workers.dev/api/url?url=https://example.com"

# List available crawls
curl "https://cc-viewer.go-mizu.workers.dev/api/crawls"

# Browse a domain
curl "https://cc-viewer.go-mizu.workers.dev/api/domain/example.com"

# View a WARC record
curl "https://cc-viewer.go-mizu.workers.dev/api/view?url=https://example.com"</code></pre>

<p>A full search API with query, filtering, and pagination is planned but not yet implemented. See the <a href="/api">API Reference</a> for what is available.</p>

<hr>

<h2 id="data">Direct Data Access</h2>
<p>OpenIndex builds on top of <a href="https://commoncrawl.org">Common Crawl</a> data. You can query their archives directly.</p>

<h3>Query CC Parquet Index with DuckDB</h3>
<pre><code># Install DuckDB
# https://duckdb.org/docs/installation

# Query CC columnar index remotely (no download)
duckdb -c "
INSTALL httpfs; LOAD httpfs;
SELECT url_host_name, COUNT(*) as pages
FROM read_parquet('s3://commoncrawl/cc-index/table/cc-main/warc/crawl=CC-MAIN-2026-04/subset=warc/*.parquet')
GROUP BY url_host_name
ORDER BY pages DESC
LIMIT 20;
"</code></pre>

<h3>Download and Query Locally</h3>
<pre><code># Use the CLI to download sample parquet files
search cc index --sample 5

# Then query locally
duckdb ~/data/common-crawl/CC-MAIN-2026-04/cc-index.duckdb -c "
SELECT url, status, content_languages
FROM cc_index
WHERE url_host_name = 'example.com'
LIMIT 50;
"</code></pre>

<h3>WARC Record Retrieval</h3>
<p>Once you have a WARC filename and byte offset from the index, fetch the raw record:</p>

<pre><code># Byte-range request for a specific WARC record
curl -r 53847234-53892481 \\
  "https://data.commoncrawl.org/crawl-data/CC-MAIN-2026-04/segments/.../warc/00000.warc.gz" \\
  | zcat</code></pre>

<div class="note">
  All Common Crawl data is freely available on S3 at <code>s3://commoncrawl/</code> and via HTTPS at <code>data.commoncrawl.org</code>. No authentication required.
</div>
`
