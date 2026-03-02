import { icons, cardIcon } from '../icons'

export const overviewPage = `
<h2>What is OpenIndex?</h2>
<p>OpenIndex is an open-source web intelligence platform. It combines a high-throughput web crawler, columnar indexing, and (planned) knowledge graph and vector search into a single, composable stack.</p>

<p>The project started in 2026 as part of the <a href="https://github.com/nicholasgasior/gopher-crawl">Mizu ecosystem</a> -- a Go web framework. It is currently a solo-developer project, open for contributions.</p>

<div class="note">
  <strong>Honest status:</strong> OpenIndex is early-stage. The crawler pipeline, sharded storage, and columnar index are built and working. Knowledge graph, vector search, full-text search, and the ontology are in design or planning. This page reflects what exists today and what is planned.
</div>

<h2>What Is Built Today</h2>
<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('globe')} <span>Recrawler</span></div>
    <p>Go-based recrawler with 100K HTTP workers, 20K DNS workers, per-domain connection limiting (8 max), multi-server DNS confirmation, and streaming probe-to-feed pipeline. Tested at 275+ pages/s peak throughput.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('cpu')} <span>Domain Crawler</span></div>
    <p>Single-domain high-throughput crawler using HTTP/2 multiplexing. Bloom filter frontier, sharded DuckDB results, resumable state. 275 pages/s peak on real sites.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('database')} <span>Sharded Storage</span></div>
    <p>16-shard DuckDB with batch-VALUES inserts (500 rows/stmt). Parquet columnar files for analytics. Zero-copy S3 queries via DuckDB httpfs extension.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('terminal')} <span>CLI Tool</span></div>
    <p>Go CLI built on Cobra + Fang. Commands: serve, crawl, crawl-domain, cc (Common Crawl), download, analytics, recrawl, reddit. Full pipeline from terminal.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('code')} <span>Common Crawl Integration</span></div>
    <p>Downloads columnar index (parquet), CDX index, WARC files from CC. Smart caching, remote S3 queries, CDX API access. Bridge to recrawler for seed URLs.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('zap')} <span>API Layer</span></div>
    <p>Hono + TypeScript on Cloudflare Workers. CC Viewer deployed at cc-viewer.go-mizu.workers.dev. URL lookup, domain browsing, WARC viewing.</p>
  </div>
</div>

<h2>What Is Planned</h2>
<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('search')} <span>Full-text Search</span></div>
    <p>Tantivy-based inverted index for keyword search across crawled content. BM25 ranking, phrase queries, field-specific filtering.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('sparkles')} <span>Vector Search</span></div>
    <p>Vald distributed vector DB with dense embeddings per page. Semantic similarity search, content clustering, RAG support.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('gitFork')} <span>Knowledge Graph</span></div>
    <p>Entity extraction via NER, Schema.org parsing. Web graph from link analysis. Entity resolution and relationship mapping.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('layers')} <span>Open Ontology</span></div>
    <p>Community-maintained schema for web entities. Schema.org compatible, available in JSON-LD, RDF, and OWL formats.</p>
  </div>
</div>

<h2>Tech Stack</h2>
<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Technology</th>
      <th>Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Crawler</strong></td>
      <td>Go (net/http, custom DNS)</td>
      <td>Built</td>
    </tr>
    <tr>
      <td><strong>Domain Crawler</strong></td>
      <td>Go (HTTP/2, bloom filter)</td>
      <td>Built</td>
    </tr>
    <tr>
      <td><strong>Storage</strong></td>
      <td>DuckDB (16-shard) + Parquet</td>
      <td>Built</td>
    </tr>
    <tr>
      <td><strong>CC Integration</strong></td>
      <td>Go (S3, CDX API, httpfs)</td>
      <td>Built</td>
    </tr>
    <tr>
      <td><strong>API</strong></td>
      <td>Hono + TypeScript (CF Workers)</td>
      <td>Built</td>
    </tr>
    <tr>
      <td><strong>CLI</strong></td>
      <td>Go (Cobra + Fang)</td>
      <td>Built</td>
    </tr>
    <tr>
      <td><strong>Web Framework</strong></td>
      <td>Mizu (Go, net/http 1.22+)</td>
      <td>Built</td>
    </tr>
    <tr>
      <td><strong>Full-text Index</strong></td>
      <td>Tantivy (Rust)</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>Vector DB</strong></td>
      <td>Vald (distributed ANN)</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>Knowledge Graph</strong></td>
      <td>TBD (DuckDB or dedicated graph DB)</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>Ontology</strong></td>
      <td>JSON-LD / RDF / OWL</td>
      <td>Designing</td>
    </tr>
  </tbody>
</table>

<h2>How It Differs from Common Crawl</h2>
<p>OpenIndex builds <strong>on top of</strong> Common Crawl rather than replacing it. CC provides petabytes of raw crawl data. OpenIndex adds:</p>
<ul>
  <li><strong>An open-source crawler</strong> -- the Go recrawler and domain crawler are fully open, auditable, and configurable.</li>
  <li><strong>Recrawl pipeline</strong> -- seed URLs from CC index, then recrawl live to get fresh content and verify liveness.</li>
  <li><strong>Sharded analytics DB</strong> -- DuckDB + Parquet for SQL analytics without downloading terabytes of WARC files.</li>
  <li><strong>Intelligence layers</strong> -- planned: full-text search, vector embeddings, entity extraction, knowledge graph.</li>
  <li><strong>Edge API</strong> -- Hono on Cloudflare Workers for low-latency access worldwide.</li>
</ul>

<h2>Key Packages</h2>
<table>
  <thead>
    <tr>
      <th>Package</th>
      <th>Path</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Recrawler</strong></td>
      <td><code>pkg/recrawler/</code></td>
      <td>Batch DNS, streaming probe, 100K HTTP workers, sharded ResultDB</td>
    </tr>
    <tr>
      <td><strong>Domain Crawler</strong></td>
      <td><code>pkg/dcrawler/</code></td>
      <td>Single-domain crawler, HTTP/2, bloom filter frontier</td>
    </tr>
    <tr>
      <td><strong>Common Crawl</strong></td>
      <td><code>pkg/cc/</code></td>
      <td>CC index, CDX API, WARC fetcher, seed URL extraction</td>
    </tr>
    <tr>
      <td><strong>CC Viewer</strong></td>
      <td><code>tools/cc-viewer/</code></td>
      <td>Hono CF Worker for browsing CC data</td>
    </tr>
    <tr>
      <td><strong>URL Fetcher</strong></td>
      <td><code>tools/url-fetcher/</code></td>
      <td>CF Worker for batch URL fetching from edge</td>
    </tr>
  </tbody>
</table>

<p>See the <a href="/architecture">Architecture</a> page for the full pipeline diagram, or the <a href="/latest-build">Latest Build</a> page for current data status.</p>
`
