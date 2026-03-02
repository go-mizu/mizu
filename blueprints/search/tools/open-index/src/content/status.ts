import { icons, cardIcon } from '../icons'

export const statusPage = `
<h2>Project Status</h2>
<p>Current state of each OpenIndex component. Updated February 2026.</p>

<div class="note">
  OpenIndex is a new project. Most components are in early development or planning stages. This page reflects the honest current state.
</div>

<hr>

<h2>Tools and Infrastructure</h2>

<div class="status-item">
  <span class="status-name">${icons.terminal} Go CLI</span>
  <span class="status-badge status-operational">Available</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.globe} CC Viewer Worker</span>
  <span class="status-badge status-operational">Deployed</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.zap} Recrawler (Go)</span>
  <span class="status-badge status-operational">Active</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.cpu} Recrawler (Zig)</span>
  <span class="status-badge" style="background:#fffbeb;color:var(--amber)">Experimental</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.search} Domain Crawler</span>
  <span class="status-badge status-operational">Active</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.layers} CC Site Extraction</span>
  <span class="status-badge status-operational">Active</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.database} CC Index Tools</span>
  <span class="status-badge status-operational">Active</span>
</div>

<hr>

<h2>Indexes and Data Layers</h2>

<div class="status-item">
  <span class="status-name">${icons.database} CDX Index (via CC API)</span>
  <span class="status-badge status-operational">Available</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.package} Columnar Index (CC Parquet)</span>
  <span class="status-badge status-operational">Available</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.search} Full-Text Search Index</span>
  <span class="status-badge" style="background:#f1f5f9;color:var(--fg-3)">Planned</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.sparkles} Vector Embeddings</span>
  <span class="status-badge" style="background:#f1f5f9;color:var(--fg-3)">Planned</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.gitFork} Knowledge Graph</span>
  <span class="status-badge" style="background:#f1f5f9;color:var(--fg-3)">Planned</span>
</div>

<hr>

<h2>API and Services</h2>

<div class="status-item">
  <span class="status-name">${icons.code} CC Viewer API</span>
  <span class="status-badge" style="background:#fffbeb;color:var(--amber)">Alpha</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.zap} URL Fetcher Worker</span>
  <span class="status-badge" style="background:#fffbeb;color:var(--amber)">Internal</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.shield} Auth / Rate Limiting</span>
  <span class="status-badge" style="background:#f1f5f9;color:var(--fg-3)">Planned</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.fileText} OQL Query Language</span>
  <span class="status-badge" style="background:#f1f5f9;color:var(--fg-3)">Planned</span>
</div>

<hr>

<h2>Frontend and UI</h2>

<div class="status-item">
  <span class="status-name">${icons.globe} Search Frontend (React + Vite)</span>
  <span class="status-badge" style="background:#fffbeb;color:var(--amber)">In Progress</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.globe} CC Viewer Web UI</span>
  <span class="status-badge status-operational">Deployed</span>
</div>
<div class="status-item">
  <span class="status-name">${icons.activity} Analytics Dashboard (Gradio)</span>
  <span class="status-badge status-operational">Available</span>
</div>

<hr>

<h2>Component Details</h2>

<details>
  <summary>Go CLI -- Available</summary>
  <div class="details-body">
    <p>The primary interface. Install with <code>go install github.com/nicholasgasior/gopher-crawl/cmd/search@latest</code>. Includes commands for CC index, crawling, recrawling, site extraction, and data downloads.</p>
  </div>
</details>

<details>
  <summary>Recrawler (Go) -- Active</summary>
  <div class="details-body">
    <p>High-throughput URL recrawler. 50K HTTP workers, 20K DNS workers, sharded DuckDB output. Batch DNS with streaming probe-to-feed pipeline. Per-domain connection limits (default: 8). DNS cache persistence across runs.</p>
  </div>
</details>

<details>
  <summary>Recrawler (Zig) -- Experimental</summary>
  <div class="details-body">
    <p>Experimental alternative using raw TCP sockets and <code>std.crypto.tls.Client</code> for TLS. Non-blocking connect with poll(). Under active development.</p>
  </div>
</details>

<details>
  <summary>Full-Text Search -- Planned</summary>
  <div class="details-body">
    <p>No implementation started. When built, this would provide keyword search across crawled page content. Likely candidates: Tantivy, Bleve, or a custom inverted index backed by DuckDB.</p>
  </div>
</details>

<details>
  <summary>Vector Embeddings -- Planned</summary>
  <div class="details-body">
    <p>No implementation started. Would generate dense embeddings for crawled pages to enable semantic similarity search. Requires significant compute for web-scale embedding generation.</p>
  </div>
</details>

<details>
  <summary>Knowledge Graph -- Planned</summary>
  <div class="details-body">
    <p>No implementation started. Would extract entities and relationships from crawled content. Requires NER, entity linking, and relationship extraction pipelines.</p>
  </div>
</details>
`
