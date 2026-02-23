import { icons, cardIcon } from '../icons'

export const homePage = `
<div class="hero">
  <h1>Index the open web.</h1>
  <p class="hero-sub">An open-source web intelligence platform. Crawl, index, and build knowledge from the open web — with columnar storage, vector search, and graph analysis.</p>
  <div class="hero-actions">
    <a href="/get-started" class="btn btn-p">Get Started</a>
    <a href="/overview" class="btn btn-s">How it Works</a>
  </div>
  <div class="stats">
    <div class="stat">
      <div class="stat-v">Growing</div>
      <div class="stat-l">Pages indexed</div>
    </div>
    <div class="stat">
      <div class="stat-v">Go + TS</div>
      <div class="stat-l">Core stack</div>
    </div>
    <div class="stat">
      <div class="stat-v">2026</div>
      <div class="stat-l">First crawls</div>
    </div>
    <div class="stat">
      <div class="stat-v">Open</div>
      <div class="stat-l">Source &amp; data</div>
    </div>
  </div>
</div>

<div class="feature-grid-wrap">
  <div class="feature-grid">
    <a href="/overview" class="feature-cell feature-cell-hero" style="text-decoration:none;color:inherit">
      <h2>The open<br>web stack.</h2>
      <p>Not just crawl data. A platform for understanding the web — from raw HTML to semantic knowledge.</p>
    </a>
    <a href="/crawler" class="feature-cell" style="text-decoration:none;color:inherit">
      <div class="cell-icon">${cardIcon('globe')}</div>
      <h3>Open Crawler</h3>
      <p>100K concurrent HTTP workers with per-domain rate limiting and full robots.txt compliance.</p>
      <span class="arrow">${icons.arrowRight}</span>
    </a>
    <a href="/indexer" class="feature-cell" style="text-decoration:none;color:inherit">
      <div class="cell-icon">${cardIcon('database')}</div>
      <h3>Multi-layer Indexer</h3>
      <p>Parquet columnar index for analytics via DuckDB. CDX for URL lookup. Full-text and vector planned.</p>
      <span class="arrow">${icons.arrowRight}</span>
    </a>
    <a href="/knowledge-graph" class="feature-cell" style="text-decoration:none;color:inherit">
      <div class="cell-icon">${cardIcon('gitFork')}</div>
      <h3>Knowledge Graph</h3>
      <p>Entity extraction and Schema.org parsing. Web graph from link analysis. Relationship mapping.</p>
      <span class="arrow">${icons.arrowRight}</span>
    </a>
    <a href="/vector-search" class="feature-cell" style="text-decoration:none;color:inherit">
      <div class="cell-icon">${cardIcon('search')}</div>
      <h3>Vector Search</h3>
      <p>Find content by meaning, not just keywords. Dense embeddings per page via Vald distributed ANN.</p>
      <span class="arrow">${icons.arrowRight}</span>
    </a>
    <a href="/architecture" class="feature-cell" style="text-decoration:none;color:inherit">
      <div class="cell-icon">${cardIcon('cpu')}</div>
      <h3>Pipeline &amp; API</h3>
      <p>Go pipeline, DuckDB sharded storage, Hono API on Cloudflare Workers. Full stack from CLI to edge.</p>
      <span class="arrow">${icons.arrowRight}</span>
    </a>
  </div>
</div>

<div class="showcase">
  <div>
    <div class="showcase-tag">${cardIcon('globe')} Open Crawler</div>
    <h2><strong>Built for throughput.</strong> <span class="muted">100K HTTP workers, per-domain rate limiting, multi-server DNS confirmation, and streaming probe-to-feed pipeline.</span></h2>
    <a href="/crawler" class="btn btn-s">Learn more</a>
  </div>
  <div class="showcase-visual">
<span class="hl">$ search recrawl --last 5 --workers 50000</span>

<span class="dim">DNS resolution</span>  <span class="blue">████████████████</span>  <span class="hl">20K workers</span>
  resolved: <span class="green">42,891</span>  dead: <span class="amber">31,204</span>  timeout: 876

<span class="dim">Probing domains</span> <span class="blue">████████████████</span>  <span class="hl">5K workers</span>
  alive: <span class="green">28,445</span>  dead: <span class="amber">14,446</span>

<span class="dim">Fetching URLs</span>   <span class="blue">████████████████</span>  <span class="hl">50K workers</span>
  <span class="green">✓ 147,231</span> fetched  <span class="hl">275 pages/s</span> peak
  → 16-shard ResultDB  1.2 GB total
  </div>
</div>

<div class="section-alt">
  <div class="showcase showcase-rev">
    <div>
      <div class="showcase-tag">${cardIcon('database')} Sharded Storage</div>
      <h2><strong>SQL analytics at web scale.</strong> <span class="muted">16-shard DuckDB with batch inserts, Parquet columnar files, and zero-copy S3 queries via DuckDB httpfs.</span></h2>
      <a href="/indexer" class="btn btn-s">Learn more</a>
    </div>
    <div class="showcase-visual">
<span class="dim">-- Zero-copy S3 query via DuckDB httpfs</span>
<span class="blue">SELECT</span> <span class="hl">url_host_name</span>,
       <span class="blue">COUNT</span>(*) <span class="blue">as</span> pages
<span class="blue">FROM</span> <span class="green">read_parquet</span>(
  <span class="amber">'s3://commoncrawl/cc-index/...'</span>
)
<span class="blue">WHERE</span> crawl = <span class="amber">'CC-MAIN-2026-04'</span>
<span class="blue">GROUP BY</span> url_host_name
<span class="blue">ORDER BY</span> pages <span class="blue">DESC</span>
<span class="blue">LIMIT</span> <span class="hl">10</span>;
    </div>
  </div>
</div>

<div class="section-alt">
<section class="section section-narrow">
  <div class="sh">
    <h2>Latest</h2>
  </div>
  <div class="blog-grid">
    <a href="/blog/per-domain-flooding" class="blog-card">
      <div class="blog-card-body">
        <span class="blog-card-tag">Engineering</span>
        <h3>685 Connections Per Domain</h3>
        <p>50K workers, 73 domains, 0.8% success rate. One constraint turned it into 57.5%.</p>
      </div>
    </a>
    <a href="/blog/dead-urls" class="blog-card">
      <div class="blog-card-body">
        <span class="blog-card-tag">Data</span>
        <h3>97% of These URLs Are Dead</h3>
        <p>CC parquet files are TLD-partitioned. File 299 is .cn domains. Most of them are dead from here.</p>
      </div>
    </a>
    <a href="/blog/zig-recrawler" class="blog-card">
      <div class="blog-card-body">
        <span class="blog-card-tag">Engineering</span>
        <h3>Raw TCP and TLS in Zig</h3>
        <p>No std.http.Client. Raw sockets, manual TLS, 64KB per worker. An education in how networks work.</p>
      </div>
    </a>
  </div>
</section>
</div>

<section class="section" style="text-align:center">
  <h2 style="margin-bottom:0.75rem;font-size:1.25rem">Help build the open web index</h2>
  <p style="color:var(--fg-2);margin-bottom:2rem;font-size:0.9375rem;max-width:480px;margin-left:auto;margin-right:auto">OpenIndex is open source. Contributions, feedback, and ideas are welcome.</p>
  <div class="hero-actions">
    <a href="https://github.com/nicholasgasior/gopher-crawl" class="btn btn-p">${icons.github} View on GitHub</a>
    <a href="/contributing" class="btn btn-s">Contributing Guide</a>
  </div>
</section>
`
