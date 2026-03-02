import { icons, cardIcon } from '../icons'

export const blogPage = `
<h2>Blog</h2>
<p>Project updates and technical deep-dives from the OpenIndex project.</p>

<div class="blog-grid">
  <a href="/blog/announcing-openindex" class="blog-card">
    <div class="blog-card-body">
      <span class="blog-card-tag">Launch</span>
      <h3>Announcing OpenIndex</h3>
      <p>An open-source web intelligence platform built on Go, DuckDB, and the Mizu ecosystem. What exists today and where we are headed.</p>
    </div>
  </a>
  <a href="/blog/recrawler-architecture" class="blog-card">
    <div class="blog-card-body">
      <span class="blog-card-tag">Engineering</span>
      <h3>Building a 100K-Worker Recrawler in Go</h3>
      <p>Batch DNS, streaming probes, per-domain connection limiting. How we went from 0.8% to 57.5% success rate.</p>
    </div>
  </a>
  <a href="/blog/duckdb-parquet" class="blog-card">
    <div class="blog-card-body">
      <span class="blog-card-tag">Engineering</span>
      <h3>Why DuckDB + Parquet for Web-Scale Analytics</h3>
      <p>16-shard DuckDB, batch inserts, zero-copy S3 queries. How columnar storage replaced traditional databases.</p>
    </div>
  </a>
</div>

<div class="blog-grid" style="margin-top:0">
  <a href="/blog/domain-crawler" class="blog-card">
    <div class="blog-card-body">
      <span class="blog-card-tag">Engineering</span>
      <h3>Domain Crawling at 275 Pages/s with HTTP/2</h3>
      <p>HTTP/2 multiplexing, bloom filter frontier, coordinator-worker pattern. Exhaustive single-domain crawling.</p>
    </div>
  </a>
  <a href="/blog/common-crawl-integration" class="blog-card">
    <div class="blog-card-body">
      <span class="blog-card-tag">Data</span>
      <h3>Common Crawl Integration: Building on Giants</h3>
      <p>Parquet index, CDX API, WARC fetching, zero-disk S3 queries. How we bridge CC data to the recrawl pipeline.</p>
    </div>
  </a>
</div>
`
