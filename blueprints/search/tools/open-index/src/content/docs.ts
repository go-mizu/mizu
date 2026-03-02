import { icons, cardIcon } from '../icons'

export const docsPage = `
<h2>Documentation</h2>
<p>OpenIndex documentation, guides, and references. The project is new -- documentation will grow as features are built.</p>

<div class="cards">
  <a href="/get-started" class="card" style="text-decoration:none;color:inherit">
    <div class="card-ic">${cardIcon('terminal')} Get Started</div>
    <p>Install the CLI, run your first crawl, and query Common Crawl data.</p>
    <span class="card-lk">Read guide ${icons.arrowRight}</span>
  </a>
  <a href="/api" class="card" style="text-decoration:none;color:inherit">
    <div class="card-ic">${cardIcon('code')} API Reference</div>
    <p>HTTP endpoints for the CC Viewer worker. Alpha status -- endpoints may change.</p>
    <span class="card-lk">View reference ${icons.arrowRight}</span>
  </a>
  <a href="/data-formats" class="card" style="text-decoration:none;color:inherit">
    <div class="card-ic">${cardIcon('fileText')} Data Formats</div>
    <p>WARC, WAT, WET, Parquet, and DuckDB schemas used by the project.</p>
    <span class="card-lk">Learn formats ${icons.arrowRight}</span>
  </a>
  <a href="/query-language" class="card" style="text-decoration:none;color:inherit">
    <div class="card-ic">${cardIcon('search')} Query Language</div>
    <p>OQL -- a planned SQL-like language for querying the web index. Coming soon.</p>
    <span class="card-lk">View spec ${icons.arrowRight}</span>
  </a>
  <a href="/errata" class="card" style="text-decoration:none;color:inherit">
    <div class="card-ic">${cardIcon('alertTriangle')} Errata</div>
    <p>Known data quality issues and caveats. Check here before reporting bugs.</p>
    <span class="card-lk">View errata ${icons.arrowRight}</span>
  </a>
  <a href="/status" class="card" style="text-decoration:none;color:inherit">
    <div class="card-ic">${cardIcon('activity')} Status</div>
    <p>Current status of each component: what works, what is planned, what is in progress.</p>
    <span class="card-lk">View status ${icons.arrowRight}</span>
  </a>
</div>

<hr>

<h2>Source Code</h2>
<p>All code is open source under Apache 2.0.</p>

<div class="cards">
  <a href="https://github.com/nicholasgasior/gopher-crawl" class="card" style="text-decoration:none;color:inherit">
    <div class="card-ic">${cardIcon('github')} Main Repository</div>
    <p>Go CLI, crawler, recrawler, Common Crawl tools, domain crawler. The core of the project.</p>
    <span class="card-lk">View on GitHub ${icons.externalLink}</span>
  </a>
  <div class="card">
    <div class="card-ic">${cardIcon('globe')} CC Viewer Worker</div>
    <p>Cloudflare Worker for browsing Common Crawl data via web UI and API. Hono + TypeScript.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('cpu')} Zig Recrawler</div>
    <p>Experimental recrawler in Zig. Raw TCP + TLS for maximum throughput and control.</p>
  </div>
</div>

<hr>

<h2>Specs</h2>
<p>Design specs are checked into the repository under <code>spec/</code>. Key documents:</p>

<table>
  <thead><tr><th>Spec</th><th>Description</th></tr></thead>
  <tbody>
    <tr><td><code>0504_commoncrawl.md</code></td><td>Common Crawl package design (CC index, CDX, WARC fetch)</td></tr>
    <tr><td><code>0505_*.md</code></td><td>CC recrawl pipeline (CC index to recrawler bridge)</td></tr>
    <tr><td><code>0506_enhance_throughput.md</code></td><td>Recrawler v2.0 (DNS, probe, feed, bandwidth display)</td></tr>
    <tr><td><code>0507_zig_recrawler.md</code></td><td>Zig recrawler design (raw TCP + TLS)</td></tr>
    <tr><td><code>0510_domain_crawler.md</code></td><td>Single-domain high-throughput crawler</td></tr>
    <tr><td><code>0511_extract_cc.md</code></td><td>CC site extraction (CDX + WARC + links)</td></tr>
    <tr><td><code>0523_cc_viewer.md</code></td><td>CC Viewer Cloudflare Worker</td></tr>
  </tbody>
</table>

<hr>

<h2>CLI Commands Reference</h2>
<table>
  <thead><tr><th>Command</th><th>Description</th></tr></thead>
  <tbody>
    <tr><td><code>search cc crawls</code></td><td>List available Common Crawl crawls</td></tr>
    <tr><td><code>search cc index</code></td><td>Download CC parquet index files</td></tr>
    <tr><td><code>search cc stats</code></td><td>Show CC index statistics</td></tr>
    <tr><td><code>search cc query</code></td><td>Query CC index with SQL</td></tr>
    <tr><td><code>search cc fetch</code></td><td>Fetch WARC records</td></tr>
    <tr><td><code>search cc url</code></td><td>CDX API URL lookup</td></tr>
    <tr><td><code>search cc site</code></td><td>Extract all pages for a domain</td></tr>
    <tr><td><code>search cc recrawl</code></td><td>Recrawl URLs from CC index</td></tr>
    <tr><td><code>search crawl-domain</code></td><td>Crawl a single domain</td></tr>
    <tr><td><code>search download</code></td><td>Download FineWeb data</td></tr>
    <tr><td><code>search analytics</code></td><td>Run analytics dashboard</td></tr>
  </tbody>
</table>
`
