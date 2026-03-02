---
slug: duckdb-parquet
title: "Why DuckDB + Parquet for Web-Scale Analytics"
date: 2026-02-20
summary: "200K inserts per second into an embedded database. No server, no cluster, no cloud bill."
tags: [engineering, storage]
---

200,000 inserts per second. That's what happens when 100K HTTP workers are fetching pages concurrently and every result needs to land somewhere with full SQL queryability. I needed a database that could eat those writes, answer analytical queries with window functions and CTEs, and also query remote Parquet files on S3 without downloading anything. Oh, and it couldn't be a server -- no Postgres, no cloud data warehouse, nothing I'd have to deploy or keep running.

That sounds like an impossible shopping list. Turns out DuckDB checks every box.

---

## What exactly do I need SQL over?

Common Crawl publishes billions of web pages per crawl. The raw WARC files for a single crawl weigh 80-100 TB. I don't need the full HTML of every page to answer questions like "how many .com pages returned a 200?" or "what are the top 50 domains by page count?" I need a columnar index and a query engine that can read it without downloading the whole thing.

Three requirements, specifically:

- **Ingest crawl results at high throughput** -- the recrawler produces 50K-200K rows per second across 100K concurrent HTTP workers
- **Query the data with full SQL** -- window functions, CTEs, aggregations, joins across crawl runs
- **Query remote data without downloading it** -- Common Crawl publishes their columnar index as Parquet on S3, and I want to query it in place

Postgres can't sustain 200K inserts/second without significant infrastructure. BigQuery and Snowflake add cost and latency. I wanted something embedded, fast, and columnar. Something that's just a file.

## So what makes DuckDB the answer?

DuckDB is an embedded columnar analytics database. No server process, no network protocol, no configuration. You open a file and run SQL. But it's not a toy -- the query engine supports window functions, CTEs, lateral joins, and parallel execution across cores. Think of it as "SQLite, but for analytics" -- that analogy does most of the work.

<table>
  <thead>
    <tr>
      <th>Property</th>
      <th>Why It Matters</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Embedded</strong></td>
      <td>No server to deploy, manage, or keep running. The database is a file.</td>
    </tr>
    <tr>
      <td><strong>Columnar engine</strong></td>
      <td>Analytics queries (COUNT, GROUP BY, aggregations) scan only the columns they need.</td>
    </tr>
    <tr>
      <td><strong>Native Parquet support</strong></td>
      <td>read_parquet() operates directly on Parquet files -- local or remote via S3.</td>
    </tr>
    <tr>
      <td><strong>httpfs extension</strong></td>
      <td>Zero-copy S3 queries. DuckDB reads Parquet column chunks via HTTP range requests.</td>
    </tr>
    <tr>
      <td><strong>Full SQL</strong></td>
      <td>Window functions, CTEs, subqueries, UNION ALL -- everything you expect from a real database.</td>
    </tr>
    <tr>
      <td><strong>Fast concurrent reads</strong></td>
      <td>Multiple goroutines can read from the same DuckDB file without coordination.</td>
    </tr>
  </tbody>
</table>

The critical thing I discovered: DuckDB treats Parquet as a first-class data source. You don't import data into DuckDB to query it. `read_parquet()` pushes predicates down into the Parquet metadata, skipping row groups that can't match. This makes remote S3 queries viable -- DuckDB only fetches the column chunks it needs.

## One writer per file is a problem. Sixteen files is the solution.

DuckDB has a single-writer limitation. Only one connection can write at a time. With 100K HTTP workers producing results concurrently, a single database file becomes a bottleneck immediately.

So I shard writes across 16 independent DuckDB files.

### How sharding works

<pre><code>  HTTP Workers (100K concurrent)
       |         |         |         |
       v         v         v         v
  +--------+ +--------+ +--------+ +--------+
  | hash() | | hash() | | hash() | | hash() |
  +--------+ +--------+ +--------+ +--------+
       |         |         |         |
       v         v         v         v
  +-------------------------------------------------+
  |           Shard Router: hash(domain) % 16       |
  +-------------------------------------------------+
    |    |    |    |    |    |         |    |    |
    v    v    v    v    v    v         v    v    v
  +--+ +--+ +--+ +--+ +--+ +--+   +--+ +--+ +--+
  |00| |01| |02| |03| |04| |05|...|13| |14| |15|
  +--+ +--+ +--+ +--+ +--+ +--+   +--+ +--+ +--+
   shard_00.duckdb  ...              shard_15.duckdb

  Each shard: independent DuckDB file
  Each shard: own flusher goroutine
  Each shard: batch-VALUES inserts (500 rows/stmt)
  No cross-shard coordination needed</code></pre>

The shard key is the domain. All URLs from the same domain go to the same shard. This means domain-level queries -- "how many pages did I fetch from example.com?" -- only need to read one shard. That wasn't obvious at first, but it turned out to be one of the better accidental decisions.

### The schema is intentionally simple

Each shard contains a single `pages` table:

<pre><code><span style="color:#60a5fa">CREATE TABLE</span> pages (
  url           <span style="color:#fbbf24">TEXT</span>,
  status        <span style="color:#fbbf24">INTEGER</span>,
  content_type  <span style="color:#fbbf24">TEXT</span>,
  content_length <span style="color:#fbbf24">INTEGER</span>,
  title         <span style="color:#fbbf24">TEXT</span>,
  elapsed_ms    <span style="color:#fbbf24">INTEGER</span>,
  fetched_at    <span style="color:#fbbf24">TIMESTAMP</span>,
  domain        <span style="color:#fbbf24">TEXT</span>,
  error         <span style="color:#fbbf24">TEXT</span>
);</code></pre>

### Why 500 rows per batch, and not 100 or 1000?

Individual INSERT statements are slow. Each one triggers a write-ahead log flush and metadata update. So I batch 500 rows into a single VALUES clause:

<pre><code><span style="color:#60a5fa">INSERT INTO</span> pages <span style="color:#60a5fa">VALUES</span>
  (<span style="color:#4ade80">'https://a.com/1'</span>, <span style="color:#fbbf24">200</span>, <span style="color:#4ade80">'text/html'</span>, ...),
  (<span style="color:#4ade80">'https://a.com/2'</span>, <span style="color:#fbbf24">301</span>, <span style="color:#4ade80">'text/html'</span>, ...),
  <span style="color:#555">-- ... 498 more rows ...</span>
  (<span style="color:#4ade80">'https://b.com/99'</span>, <span style="color:#fbbf24">200</span>, <span style="color:#4ade80">'text/html'</span>, ...);</code></pre>

The flusher goroutine for each shard accumulates rows in a buffer. When the buffer reaches 500 (or a timeout fires), it constructs the multi-row VALUES statement and executes it in a single DuckDB call:

<pre><code><span style="color:#60a5fa">func</span> (s *Shard) <span style="color:#e0e0e0">flush</span>(rows []Row) <span style="color:#fbbf24">error</span> {
    <span style="color:#60a5fa">if</span> <span style="color:#e0e0e0">len</span>(rows) == <span style="color:#fbbf24">0</span> {
        <span style="color:#60a5fa">return</span> <span style="color:#fbbf24">nil</span>
    }
    <span style="color:#555">// Build: INSERT INTO pages VALUES (?,?,...), (?,?,...), ...</span>
    buf := strings.Builder{}
    buf.WriteString(<span style="color:#4ade80">"INSERT INTO pages VALUES "</span>)
    args := <span style="color:#60a5fa">make</span>([]<span style="color:#60a5fa">any</span>, <span style="color:#fbbf24">0</span>, <span style="color:#e0e0e0">len</span>(rows)*<span style="color:#fbbf24">9</span>)
    <span style="color:#60a5fa">for</span> i, r := <span style="color:#60a5fa">range</span> rows {
        <span style="color:#60a5fa">if</span> i > <span style="color:#fbbf24">0</span> { buf.WriteByte(<span style="color:#4ade80">','</span>) }
        buf.WriteString(<span style="color:#4ade80">"(?,?,?,?,?,?,?,?,?)"</span>)
        args = <span style="color:#60a5fa">append</span>(args, r.URL, r.Status, r.ContentType,
            r.ContentLength, r.Title, r.ElapsedMs,
            r.FetchedAt, r.Domain, r.Error)
    }
    _, err := s.db.Exec(buf.String(), args...)
    <span style="color:#60a5fa">return</span> err
}</code></pre>

<div class="note">
  <strong>Why 500 rows?</strong> I tested this empirically. At 100 rows/batch: ~20K rows/s. At 500: throughput jumped to ~50K rows/s per shard. At 1000: marginal improvement, more memory. With 16 shards at 50K rows/s each, peak write throughput reaches ~200K rows/s. That's enough to keep up with 100K HTTP workers without breaking a sweat.
</div>

## DuckDB for writes, Parquet for everything else

DuckDB is the write-path and local query engine. Parquet is the archival and remote query format. They complement each other nicely:

<table>
  <thead>
    <tr>
      <th>Concern</th>
      <th>DuckDB</th>
      <th>Parquet</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Write path</strong></td>
      <td>Primary (batch inserts)</td>
      <td>Export target</td>
    </tr>
    <tr>
      <td><strong>Storage</strong></td>
      <td>Local files</td>
      <td>Local or S3</td>
    </tr>
    <tr>
      <td><strong>Compression</strong></td>
      <td>Internal</td>
      <td>Snappy, Zstd, Gzip</td>
    </tr>
    <tr>
      <td><strong>Remote query</strong></td>
      <td>Via httpfs (reads Parquet)</td>
      <td>Native (column chunks via HTTP range requests)</td>
    </tr>
    <tr>
      <td><strong>Concurrent writes</strong></td>
      <td>Single writer per file</td>
      <td>Immutable (write once)</td>
    </tr>
    <tr>
      <td><strong>Best for</strong></td>
      <td>Active crawl ingestion</td>
      <td>Published indexes, archival</td>
    </tr>
  </tbody>
</table>

Parquet's columnar layout means a query touching only `url_host_name` and `fetch_status` reads only those two columns from disk or network. For a file with 20 columns, that's a 10x reduction in I/O. Combined with row group metadata (min/max statistics), DuckDB can skip entire row groups that don't match a WHERE clause. It's predicate pushdown all the way down.

## Querying petabytes on S3 without downloading a byte

Common Crawl publishes a columnar index for every crawl as Parquet files on S3. DuckDB can query these files directly. This is the `--remote` flag in the CLI, and it still feels a little like magic every time I use it:

<pre><code><span style="color:#555">-- Top 10 .com domains by page count in CC-MAIN-2026-04</span>
<span style="color:#60a5fa">SELECT</span> url_host_name, <span style="color:#60a5fa">COUNT</span>(*) <span style="color:#60a5fa">AS</span> pages
<span style="color:#60a5fa">FROM</span> <span style="color:#e0e0e0">read_parquet</span>(
  <span style="color:#4ade80">'s3://commoncrawl/cc-index/table/cc-main/warc/crawl=CC-MAIN-2026-04/subset=warc/*.parquet'</span>
)
<span style="color:#60a5fa">WHERE</span> url_host_tld = <span style="color:#4ade80">'com'</span>
<span style="color:#60a5fa">GROUP BY</span> url_host_name
<span style="color:#60a5fa">ORDER BY</span> pages <span style="color:#60a5fa">DESC</span>
<span style="color:#60a5fa">LIMIT</span> <span style="color:#fbbf24">10</span>;</code></pre>

Under the hood, DuckDB issues HTTP range requests to S3. It reads the Parquet footer (schema + row group metadata), determines which row groups might match, then fetches only the relevant column chunks. For a query filtering on `url_host_tld`, most row groups are skipped entirely because the footer's min/max statistics eliminate them before any data is transferred.

<div class="note">
  <strong>Performance.</strong> Remote S3 queries typically complete in 2-5 seconds for single-file analytics. Local Parquet queries are sub-second for most aggregations. The bottleneck for remote queries is network latency, not DuckDB.
</div>

## Three ways to access Common Crawl data

The CC package (`pkg/cc`) provides three access modes, all built on DuckDB + Parquet:

### Mode 1: Remote Query (Zero Disk)

<pre><code><span style="color:#555"># Query CC index on S3 directly -- nothing downloaded</span>
<span style="color:#e0e0e0">$ search cc stats --remote</span>

<span style="color:#555"># DuckDB loads httpfs, issues range requests to S3</span>
<span style="color:#555"># Only column chunks matching the query are fetched</span></code></pre>

### Mode 2: Sample Download

<pre><code><span style="color:#555"># Download N evenly-spaced parquet files from the CC index</span>
<span style="color:#e0e0e0">$ search cc index --sample 5</span>

<span style="color:#555"># Downloads files 0, 60, 120, 180, 240 (out of 300)</span>
<span style="color:#555"># Stored at ~/data/common-crawl/CC-MAIN-2026-04/</span></code></pre>

### Mode 3: Direct Parquet Extraction

<pre><code><span style="color:#555"># Extract seed URLs directly from parquet via read_parquet()</span>
<span style="color:#555"># Zero DuckDB import -- reads parquet natively</span>
<span style="color:#e0e0e0">$ search cc recrawl --last</span></code></pre>

A fun discovery: CC parquet files are TLD-partitioned. File 299 is almost entirely `.cn` domains, file 0 is mostly `.ru`, file 50 is predominantly `.fi`. About 97% of CC index domains are dead when recrawled from outside their geographic region. The `--sample` flag spaces files evenly to get a representative TLD distribution.

## The full data flow

Here's how it all connects:

<pre><code>  CC Parquet Index (S3)
         |
         | read_parquet() / --remote
         v
  +------------------+
  |  Seed Extraction |  zero import, native parquet read
  +--------+---------+
           |
           v
  +------------------+
  |    Recrawler     |  100K HTTP workers
  +--------+---------+
           |
           | hash(domain) % 16
           v
  +------------------+     +------------------+
  | ResultDB (16x)   | --> | Parquet Export    |
  | shard_00..15.db  |     | archived.parquet |
  +------------------+     +------------------+
           |                        |
           v                        v
  +------------------+     +------------------+
  |  Local Queries   |     |  S3 / Remote     |
  |  DuckDB CLI/API  |     |  read_parquet()  |
  +------------------+     +------------------+</code></pre>

## What the benchmarks actually say

<table>
  <thead>
    <tr>
      <th>Operation</th>
      <th>Throughput</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Single-shard write</strong></td>
      <td>~50K rows/s</td>
      <td>500 rows/batch, sustained</td>
    </tr>
    <tr>
      <td><strong>16-shard write</strong></td>
      <td>~200K rows/s peak</td>
      <td>No cross-shard coordination</td>
    </tr>
    <tr>
      <td><strong>Local Parquet query</strong></td>
      <td>Sub-second</td>
      <td>Most aggregation queries</td>
    </tr>
    <tr>
      <td><strong>Remote S3 query</strong></td>
      <td>2-5 seconds</td>
      <td>Single-file, depends on network</td>
    </tr>
    <tr>
      <td><strong>Individual INSERT</strong></td>
      <td>~2K rows/s</td>
      <td>25x slower than batched (avoid)</td>
    </tr>
  </tbody>
</table>

<div class="note">
  <strong>All numbers from real runs.</strong> These benchmarks come from actual crawl pipelines running against Common Crawl data, not synthetic tests. Write throughput numbers are sustained over millions of rows. Query times measured against CC-MAIN-2026-04 Parquet files on S3.
</div>

## Where it breaks down

DuckDB + Parquet isn't the right choice for every workload. I want to be honest about where it doesn't work:

**Works well for:**

- Analytical queries over crawl data (aggregations, grouping, filtering)
- High-throughput batch ingestion from concurrent workers
- Remote queries against S3-hosted Parquet indexes
- Embedded use cases where you don't want a database server

**Doesn't work well for:**

- Point lookups by primary key (use a key-value store or B-tree index)
- Concurrent writes to the same file (single-writer limitation -- hence the sharding)
- Real-time streaming ingestion (batch-oriented by design)
- Transactional workloads (no row-level locking)

The sharding adds complexity but eliminates the single-writer bottleneck. With 16 shards, each handles roughly 6K workers (100K / 16), well within DuckDB's single-writer throughput. The domain-based shard key means most queries only need one shard, keeping read performance high.

For my use case -- crawling billions of URLs and analyzing the results with SQL -- DuckDB and Parquet give me the query power of a data warehouse with the operational simplicity of SQLite. No servers to manage. No cloud bills to optimize. No ETL pipelines to maintain. Just files and SQL.
