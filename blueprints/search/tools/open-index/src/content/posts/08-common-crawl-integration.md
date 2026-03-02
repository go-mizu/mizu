---
slug: common-crawl-integration
title: "Common Crawl Integration: Building on Giants"
date: 2026-02-23
summary: "3.2 billion pages per crawl, nine CLI commands, and zero downloads required to start querying."
tags: [engineering, data]
---

3.2 billion pages. That's a single monthly crawl from Common Crawl. The WARC archives for one crawl weigh 80-100 TB. The columnar index alone is hundreds of gigabytes of Parquet files. Going back to 2008, the total archive is measured in petabytes. All of it public, all of it on S3, none of it requiring authentication.

I don't try to replace any of that. What I build is the tooling that makes CC data accessible, queryable, and useful as a foundation for live recrawling. Along the way I discovered that 97% of the domains in some CC files are dead -- and that's not a bug, it's geography.

## What does Common Crawl actually give you?

Three distinct index types, each serving a different purpose:

<table>
  <thead>
    <tr>
      <th>Index Type</th>
      <th>Format</th>
      <th>Use Case</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>CDX</strong></td>
      <td>JSON API</td>
      <td>Look up specific URLs across all crawls. Zero download, query by URL or domain.</td>
    </tr>
    <tr>
      <td><strong>Columnar</strong></td>
      <td>Apache Parquet</td>
      <td>Analytics over crawl metadata. 300+ files per crawl, queryable via DuckDB.</td>
    </tr>
    <tr>
      <td><strong>WARC</strong></td>
      <td>Compressed archives</td>
      <td>Raw page content. HTTP headers + response body. Byte-range addressable on S3.</td>
    </tr>
  </tbody>
</table>

The scale is hard to internalize. Billions of pages per crawl. Hundreds of gigabytes of index files. Petabytes of raw content. And it's just... sitting there on S3, waiting for someone to ask it a question.

## Nine commands to talk to all of it

The `search cc` command group is my primary interface to Common Crawl. Nine subcommands cover the full lifecycle from discovery to data extraction:

<table>
  <thead>
    <tr>
      <th>Command</th>
      <th>Description</th>
      <th>Disk Usage</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>search cc crawls</strong></td>
      <td>List available crawls from CC API</td>
      <td>Zero</td>
    </tr>
    <tr>
      <td><strong>search cc index</strong></td>
      <td>Download columnar index (Parquet files)</td>
      <td>Varies (sampled)</td>
    </tr>
    <tr>
      <td><strong>search cc stats</strong></td>
      <td>Analytics over downloaded index data</td>
      <td>Zero (reads existing)</td>
    </tr>
    <tr>
      <td><strong>search cc query</strong></td>
      <td>SQL queries against CC data via DuckDB</td>
      <td>Zero (reads existing)</td>
    </tr>
    <tr>
      <td><strong>search cc fetch</strong></td>
      <td>Download specific WARC records</td>
      <td>Per-record</td>
    </tr>
    <tr>
      <td><strong>search cc warc</strong></td>
      <td>View WARC file contents</td>
      <td>Zero (reads existing)</td>
    </tr>
    <tr>
      <td><strong>search cc url</strong></td>
      <td>Look up a URL via CDX API</td>
      <td>Zero</td>
    </tr>
    <tr>
      <td><strong>search cc site</strong></td>
      <td>Extract all pages for a domain</td>
      <td>DuckDB (site.duckdb)</td>
    </tr>
    <tr>
      <td><strong>search cc recrawl</strong></td>
      <td>CC index to recrawler pipeline</td>
      <td>16-shard DuckDB</td>
    </tr>
  </tbody>
</table>

### Finding out what's available

The starting point is always listing crawls. CC publishes a JSON manifest at `index.commoncrawl.org/collinfo.json` with every crawl, its date range, page count, and index endpoints:

<pre><code><span style="color:#888">$</span> <span style="color:#e0e0e0">search cc crawls</span>

<span style="color:#4ade80">CC-MAIN-2026-04</span>  Jan 2026   3.2B pages   cdx-api ready
<span style="color:#4ade80">CC-MAIN-2025-51</span>  Dec 2025   3.1B pages   cdx-api ready
<span style="color:#4ade80">CC-MAIN-2025-47</span>  Nov 2025   3.0B pages   cdx-api ready
<span style="color:#888">... 98 more crawls</span></code></pre>

### Looking up a URL without downloading anything

The fastest way to check if a URL exists in Common Crawl is the CDX API. No downloads, no imports -- just ask:

<pre><code><span style="color:#888">$</span> <span style="color:#e0e0e0">search cc url https://example.com/page</span>

<span style="color:#60a5fa">Querying CDX API...</span>
  Crawl: <span style="color:#4ade80">CC-MAIN-2026-04</span>  Status: <span style="color:#4ade80">200</span>  Type: text/html  Size: 14,328
  Crawl: <span style="color:#4ade80">CC-MAIN-2025-51</span>  Status: <span style="color:#4ade80">200</span>  Type: text/html  Size: 14,102
  Crawl: <span style="color:#fbbf24">CC-MAIN-2025-22</span>  Status: <span style="color:#fbbf24">301</span>  Type: text/html  Size: 412
<span style="color:#888">3 captures found across all crawls</span></code></pre>

## How do you avoid downloading 300 GB of index files?

Downloading the entire columnar index for a single crawl means pulling 300+ Parquet files -- hundreds of gigabytes. For most analysis, that's absurd. I use two strategies.

**Sampling.** The `--sample N` flag downloads N evenly-spaced Parquet files out of the full set. Default is 5, which gives a representative cross-section of the crawl without downloading everything:

<pre><code><span style="color:#888">$</span> <span style="color:#e0e0e0">search cc index --sample 5</span>

<span style="color:#60a5fa">Downloading 5 of 300 parquet files (evenly spaced)</span>
  <span style="color:#4ade80">&#10003;</span> cdx-00000.parquet   <span style="color:#888">192 MB</span>
  <span style="color:#4ade80">&#10003;</span> cdx-00060.parquet   <span style="color:#888">187 MB</span>
  <span style="color:#4ade80">&#10003;</span> cdx-00120.parquet   <span style="color:#888">201 MB</span>
  <span style="color:#4ade80">&#10003;</span> cdx-00180.parquet   <span style="color:#888">194 MB</span>
  <span style="color:#4ade80">&#10003;</span> cdx-00240.parquet   <span style="color:#888">189 MB</span>
<span style="color:#888">Total: 963 MB (0.3% of full index)</span></code></pre>

**Cache manifest.** A local `cache.json` stores the crawl list and file manifests with a 24-hour TTL. Repeated runs reuse cached metadata instead of hitting the CC API again. Unchanged files are skipped entirely.

## Or just query S3 directly and download nothing at all

For exploration, even downloading 5 files is overkill. The `--remote` flag tells DuckDB to query S3 Parquet files directly via httpfs. Zero disk. Zero download. It still feels slightly magical:

<pre><code><span style="color:#888">$</span> <span style="color:#e0e0e0">search cc query --remote</span>

<span style="color:#60a5fa">Query (DuckDB httpfs, reading from S3 directly):</span>
<span style="color:#fbbf24">SELECT</span> url_host_tld, <span style="color:#fbbf24">COUNT</span>(*) <span style="color:#fbbf24">AS</span> pages
<span style="color:#fbbf24">FROM</span> <span style="color:#4ade80">read_parquet</span>(<span style="color:#e0e0e0">'s3://commoncrawl/cc-index/table/cc-main/warc/crawl=CC-MAIN-2026-04/subset=warc/*.parquet'</span>)
<span style="color:#fbbf24">GROUP BY</span> url_host_tld
<span style="color:#fbbf24">ORDER BY</span> pages <span style="color:#fbbf24">DESC</span>
<span style="color:#fbbf24">LIMIT</span> 10;

<span style="color:#e0e0e0">url_host_tld</span>  <span style="color:#e0e0e0">pages</span>
<span style="color:#888">────────────  ──────────</span>
com           1,847,293,041
org             198,420,112
de              142,837,209
net              98,201,443
ru               87,492,310
<span style="color:#888">... 5 more rows</span></code></pre>

<div class="note">
  <strong>Trade-off:</strong> Remote queries are great for ad-hoc exploration but slow for batch processing. Each query scans S3 over the network. For repeated analytics, download the sample first with <code>search cc index --sample 5</code> and query locally.
</div>

## The bridge from archived data to live recrawling

The most powerful integration is the pipeline from CC index data into the live recrawler. The idea: use CC as a seed list of known URLs, then recrawl them for fresh content.

### How the pipeline connects

<pre><code>  Common Crawl S3
  (Parquet Index)
        |
        v
  +-------------------------------+
  |  Seed URL Extraction          |
  |  read_parquet() -- zero       |
  |  DuckDB import, direct read   |
  +---------------+---------------+
                  |
        +---------+---------+
        |                   |
  --last N            --file N
  (last N files)      (specific file)
        |                   |
        +---------+---------+
                  |
                  v
  +-------------------------------+
  |  Batch DNS Resolution         |
  |  20K workers, multi-server    |
  |  CF -> Google -> stdlib       |
  +---------------+---------------+
                  |
        +---------+---------+
        |                   |
   resolved IPs        dead domains
        |              (skipped)
        v
  +-------------------------------+
  |  Streaming Probe + Feed       |
  |  5K workers, immediate URL    |
  |  feed as probes succeed       |
  +---------------+---------------+
                  |
                  v
  +-------------------------------+
  |  HTTP Workers (50K default)   |
  |  8 max conns/domain           |
  |  round-robin interleaving     |
  +---------------+---------------+
                  |
                  v
  +-------------------------------+
  |  Sharded ResultDB (DuckDB)   |
  |  16 shards, 500 rows/stmt    |
  |  ~/data/common-crawl/        |
  |    {CrawlID}/recrawl/        |
  +-------------------------------+</code></pre>

### Three ways to pick your seed data

<pre><code><span style="color:#888"># Use the last N parquet files (direct, no import step)</span>
<span style="color:#888">$</span> <span style="color:#e0e0e0">search cc recrawl --last</span>

<span style="color:#888"># Use a specific parquet file by number</span>
<span style="color:#888">$</span> <span style="color:#e0e0e0">search cc recrawl --file 50</span>

<span style="color:#888"># Use sampled files (legacy mode)</span>
<span style="color:#888">$</span> <span style="color:#e0e0e0">search cc recrawl --sample 5</span></code></pre>

Seeds are extracted via `ExtractSeedURLsFromParquet()`, which calls `read_parquet()` directly. No intermediate DuckDB import step -- Parquet files are read in-place and URLs stream into the recrawler pipeline.

<pre><code><span style="color:#888">$</span> <span style="color:#e0e0e0">search cc recrawl --last --max-conns-per-domain 8 --domain-fail-threshold 2</span>

<span style="color:#60a5fa">Extracting seed URLs from CC-MAIN-2026-04...</span>
  Seeds: <span style="color:#4ade80">2,481,093 URLs</span> from 73,291 domains

<span style="color:#60a5fa">Batch DNS resolution (20K workers)...</span>
  Resolved: <span style="color:#4ade80">42,891</span>  Dead: <span style="color:#fbbf24">28,104</span>  Timeout: <span style="color:#888">2,296</span>

<span style="color:#60a5fa">Streaming probe + feed...</span>
  Alive: <span style="color:#4ade80">38,210</span>  Refused: <span style="color:#fbbf24">4,681</span>  <span style="color:#888">65s elapsed</span>

<span style="color:#60a5fa">HTTP fetch (50K workers, 8 max conns/domain)...</span>
  Fetched: <span style="color:#4ade80">891,204</span>  Failed: <span style="color:#fbbf24">1,589,889</span>  <span style="color:#888">57.5% live</span>
  Bandwidth: <span style="color:#4ade80">142 MB/s</span>  Duration: <span style="color:#888">4m12s</span>

<span style="color:#888">Results: ~/data/common-crawl/CC-MAIN-2026-04/recrawl/ (16 shards)</span>
<span style="color:#888">DNS cache: ~/data/common-crawl/CC-MAIN-2026-04/dns.duckdb</span></code></pre>

The DNS cache persists between runs. Subsequent recrawls of the same crawl ID skip DNS resolution for already-resolved and known-dead domains.

## Extracting an entire site from the archive

The `search cc site` command pulls all pages for a single domain from CC archives. Three modes depending on how much data you want:

<table>
  <thead>
    <tr>
      <th>Mode</th>
      <th>Data Retrieved</th>
      <th>Speed</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>urls</strong></td>
      <td>URL list from CDX API only</td>
      <td>Fast (API-only, no WARC)</td>
    </tr>
    <tr>
      <td><strong>links</strong></td>
      <td>URLs + outgoing links from WARC</td>
      <td>Medium (WARC byte-range fetches)</td>
    </tr>
    <tr>
      <td><strong>full</strong></td>
      <td>URLs + links + full page body</td>
      <td>Slower (full WARC content)</td>
    </tr>
  </tbody>
</table>

<pre><code><span style="color:#888">$</span> <span style="color:#e0e0e0">search cc site duckdb.org --mode links</span>

<span style="color:#60a5fa">CDX query (matchType=domain)...</span>
  Pages: <span style="color:#4ade80">1,110</span> across all subdomains

<span style="color:#60a5fa">WARC fetch (byte-range from data.commoncrawl.org)...</span>
  Fetched: <span style="color:#4ade80">1,110</span>  Peak: <span style="color:#4ade80">193/s</span>  Failed: <span style="color:#4ade80">0</span>
  Links extracted: <span style="color:#4ade80">110,482</span>
  Duration: <span style="color:#888">7s</span>

<span style="color:#888">Results: ~/data/common-crawl/site/duckdb.org/site.duckdb</span>
<span style="color:#888">Tables: pages (1,110 rows), links (110,482 rows), meta</span></code></pre>

The CDX API is queried with `matchType=domain`, which returns all subdomains automatically. WARC records come via byte-range HTTP requests against `data.commoncrawl.org` -- no need to download entire WARC files. One byte-range GET per page, no wasted bandwidth. Results go into a single DuckDB database with pages, links, and meta tables.

## Browse it in a browser

For times when the CLI is too much ceremony, there's a Cloudflare Worker at **cc-viewer.go-mizu.workers.dev**. Hono, server-side rendered HTML, KV caching:

<div class="cards">
  <div class="card">
    <div class="card-ic"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg> <span>URL Lookup</span></div>
    <h3>/url/*</h3>
    <p>Enter any URL to see all CC captures across crawls. Same data as the CDX API, rendered in a browsable table.</p>
  </div>
  <div class="card">
    <div class="card-ic"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/></svg> <span>Domain Browse</span></div>
    <h3>/domain/:domain</h3>
    <p>Browse all pages captured for a domain. Paginated listing with status codes, content types, and timestamps.</p>
  </div>
  <div class="card">
    <div class="card-ic"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/></svg> <span>WARC Viewer</span></div>
    <h3>/view</h3>
    <p>View raw WARC records directly in the browser. Decompressed in-worker using DecompressionStream('gzip').</p>
  </div>
  <div class="card">
    <div class="card-ic"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83Z"/><path d="m22 17.65-9.17 4.16a2 2 0 0 1-1.66 0L2 17.65"/><path d="m22 12.65-9.17 4.16a2 2 0 0 1-1.66 0L2 12.65"/></svg> <span>Crawl List</span></div>
    <h3>/crawls</h3>
    <p>List all available CC crawls with dates, page counts, and quick links to the CDX API endpoints.</p>
  </div>
</div>

## The things I learned the hard way

Working with Common Crawl data at scale taught me several things that aren't in any documentation.

<div class="note">
  <strong>97% of those domains are dead.</strong> CC Parquet files are TLD-partitioned. File 299 is almost entirely .cn domains. File 0 is mostly .ru. File 50 is predominantly .fi. When I recrawled from outside those regions, approximately 97% of the domains came back dead. Many country-code TLD sites only resolve within their region's DNS infrastructure, or they've simply gone offline since CC archived them. This isn't a failure -- it's the reality of the internet. If you're benchmarking a recrawler, pick file 50 (mixed TLDs) for realistic numbers. File 299 will give you 97%+ dead regardless of how good your code is.
</div>

<div class="note">
  <strong>The CDX API returns JSON, not plain text.</strong> When using <code>showNumPages=true</code>, the response is <code>{"pages": N, ...}</code> -- a JSON object, not a bare integer. My early code assumed a plain number and silently failed. Small detail, easy to miss, wasted an hour.
</div>

<div class="note">
  <strong>Byte-range WARC requests are the right approach.</strong> Downloading full WARC files to extract a few pages is wasteful. The CC CDN at <code>data.commoncrawl.org</code> supports standard HTTP Range headers. The CDX index gives the exact offset and length for each record. One byte-range GET per page, zero wasted bandwidth.
</div>

## What this makes possible

Common Crawl gives a starting point -- billions of known URLs with metadata. With efficient tooling on top, several workflows open up:

- **Freshness tracking:** Recrawl CC URLs to see what's changed, what's died, and what's new.
- **Domain analysis:** Extract complete site structures from CC archives without crawling live.
- **Seed generation:** Use CC as a URL source for focused recrawls of specific TLDs, languages, or content types.
- **Historical analysis:** Query CC Parquet data to understand how the web has changed over time.
- **Link graph construction:** Extract outgoing links from WARC records to build web graphs.

None of this replaces Common Crawl. It builds on top of it. CC provides the petabytes of raw data. OpenIndex provides the CLI, the caching layer, the recrawler bridge, and the analytics tooling to make that data useful for specific tasks.

All of this is open source at [github.com/nicholasgasior/gopher-crawl](https://github.com/nicholasgasior/gopher-crawl). The Go implementation lives in `pkg/cc/`. The CC Viewer worker is in `tools/cc-viewer/`. Contributions welcome.
