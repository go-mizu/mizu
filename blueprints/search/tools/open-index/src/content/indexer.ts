import { icons, cardIcon } from '../icons'

export const indexerPage = `
<h2>Multi-Layer Indexing</h2>
<p>OpenIndex is designed around four complementary index types. Today, the columnar index (Parquet + DuckDB) is built and operational. The others are in various stages of planning.</p>

<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('database')} <span>Columnar Index</span></div>
    <h3>Built</h3>
    <p>Apache Parquet files with structured metadata. Queryable with DuckDB, including remote S3 queries via httpfs. 16-shard DuckDB for write throughput.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('fileText')} <span>CDX Index</span></div>
    <h3>Planned</h3>
    <p>CDXJ-format index for URL-to-record lookup. Compatible with Common Crawl / Wayback Machine format. Currently accessed via CC's CDX API.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('search')} <span>Full-Text Index</span></div>
    <h3>Planned</h3>
    <p>Tantivy-based inverted index for keyword search. BM25 ranking, phrase queries, field-specific search. Not yet implemented.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('sparkles')} <span>Vector Index</span></div>
    <h3>Planned</h3>
    <p>Dense embeddings in Vald distributed vector DB. Semantic similarity search. Architecture designed, not yet deployed.</p>
  </div>
</div>

<hr>

<h2>Columnar Index (Parquet + DuckDB)</h2>
<p>This is the primary index today. Crawl results are stored in 16-shard DuckDB databases with batch-VALUES inserts. Data can be exported to Parquet for analytics and sharing.</p>

<h3>How It Works</h3>
<p>The recrawler and domain crawler write results directly to sharded DuckDB:</p>

<pre><code># 16 DuckDB shards for concurrent write throughput
# Batch-VALUES inserts: 500 rows per INSERT statement
# Async flush via dedicated flusher goroutines
# URLs distributed across shards by hash

~/data/common-crawl/CC-MAIN-2026-04/recrawl/
  shard_00.duckdb
  shard_01.duckdb
  ...
  shard_15.duckdb</code></pre>

<h3>Querying with DuckDB</h3>
<p>DuckDB can query Parquet files directly from S3 without downloading, using the httpfs extension:</p>

<pre><code>-- Remote query: zero disk, zero download
-- Uses DuckDB httpfs extension
INSTALL httpfs;
LOAD httpfs;

SELECT url_host_tld, count(*) as pages
FROM read_parquet('s3://commoncrawl/cc-index/table/cc-main/warc/crawl=CC-MAIN-2026-04/*.parquet')
GROUP BY url_host_tld
ORDER BY pages DESC
LIMIT 20;

-- Query local results from a recrawl
SELECT url, status_code, content_type
FROM read_parquet('~/data/common-crawl/CC-MAIN-2026-04/recrawl/*.parquet')
WHERE status_code = 200
LIMIT 100;</code></pre>

<h3>Compatible Tools</h3>
<table>
  <thead>
    <tr>
      <th>Tool</th>
      <th>Usage</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>DuckDB</strong></td>
      <td><code>SELECT * FROM read_parquet('path/*.parquet')</code></td>
    </tr>
    <tr>
      <td><strong>Polars</strong></td>
      <td><code>pl.scan_parquet("path/*.parquet")</code></td>
    </tr>
    <tr>
      <td><strong>pandas</strong></td>
      <td><code>pd.read_parquet("path/")</code></td>
    </tr>
    <tr>
      <td><strong>Apache Spark</strong></td>
      <td><code>spark.read.parquet("s3://...")</code></td>
    </tr>
    <tr>
      <td><strong>ClickHouse</strong></td>
      <td><code>SELECT * FROM s3('s3://...', 'Parquet')</code></td>
    </tr>
  </tbody>
</table>

<hr>

<h2>CDX Index (Planned)</h2>
<p>OpenIndex plans to produce its own CDX index for URL-level lookups. Currently, URL lookups use Common Crawl's CDX API directly.</p>

<h3>Current Access via CC</h3>
<pre><code># Query CC CDX API directly (zero disk, zero download)
search cc url https://example.com

# Or via curl
curl "https://index.commoncrawl.org/CC-MAIN-2026-04-index?url=example.com&output=json"</code></pre>

<h3>Planned Format</h3>
<p>When OpenIndex produces its own crawl data, CDX records will follow the standard CDXJ format:</p>
<pre><code>com,example)/path 20260215083000 {"url":"https://example.com/path","mime":"text/html","status":"200","digest":"sha1:ABC123...","length":"12453","offset":"3245678","filename":"warc/00042.warc.gz"}</code></pre>

<hr>

<h2>Full-Text Index (Planned)</h2>
<p>Keyword search across crawled content is planned using <a href="https://github.com/quickwit-oss/tantivy">Tantivy</a>, a full-text search engine written in Rust.</p>

<h3>Planned Capabilities</h3>
<ul>
  <li>BM25 ranking</li>
  <li>Phrase queries and Boolean operators</li>
  <li>Field-specific search (title, body, domain)</li>
  <li>Language filtering</li>
  <li>Faceted results</li>
</ul>

<div class="note">
  Full-text indexing is not yet implemented. The design is based on Tantivy's architecture but no code has been written for this component yet.
</div>

<hr>

<h2>Vector Index (Planned)</h2>
<p>Semantic search via dense embeddings is planned using <a href="https://vald.vdaas.org/">Vald</a>, a distributed approximate nearest neighbor search engine.</p>

<h3>Planned Design</h3>
<table>
  <thead>
    <tr>
      <th>Property</th>
      <th>Planned Value</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Vector DB</strong></td>
      <td>Vald (distributed ANN)</td>
    </tr>
    <tr>
      <td><strong>Embedding model</strong></td>
      <td>TBD (likely multilingual-e5-large, 1024 dims)</td>
    </tr>
    <tr>
      <td><strong>Granularity</strong></td>
      <td>Per-page (title + first 512 tokens)</td>
    </tr>
    <tr>
      <td><strong>Similarity metric</strong></td>
      <td>Cosine similarity</td>
    </tr>
  </tbody>
</table>

<p>See the <a href="/vector-search">Vector Search</a> page for the full vision.</p>

<div class="note">
  Vector indexing is in the design phase. No embeddings have been generated yet. The architecture has been planned but deployment is pending the completion of the crawl pipeline.
</div>

<hr>

<h2>Index Status Summary</h2>
<table>
  <thead>
    <tr>
      <th>Index Type</th>
      <th>Status</th>
      <th>Technology</th>
      <th>Access</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Columnar (Parquet)</strong></td>
      <td>Built</td>
      <td>DuckDB + Apache Parquet</td>
      <td>Local + S3 remote queries</td>
    </tr>
    <tr>
      <td><strong>CDX</strong></td>
      <td>Planned (using CC's CDX API now)</td>
      <td>CDXJ format</td>
      <td>Via CC CDX API</td>
    </tr>
    <tr>
      <td><strong>Full-text</strong></td>
      <td>Planned</td>
      <td>Tantivy (Rust)</td>
      <td>Not yet available</td>
    </tr>
    <tr>
      <td><strong>Vector</strong></td>
      <td>Planned</td>
      <td>Vald (distributed ANN)</td>
      <td>Not yet available</td>
    </tr>
  </tbody>
</table>
`
