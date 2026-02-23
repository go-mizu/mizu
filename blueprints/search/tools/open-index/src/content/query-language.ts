import { icons, cardIcon } from '../icons'

export const queryLanguagePage = `
<div class="note note-warn">
  <strong>Coming Soon.</strong> OQL (OpenIndex Query Language) is planned but not yet implemented. This page describes the vision. Nothing here is usable today.
</div>

<h2>OpenIndex Query Language (OQL)</h2>
<p>OQL is a planned SQL-like query language for searching and analyzing the OpenIndex web corpus. The goal is to provide a single interface for querying across the columnar index, full-text search, and (eventually) vector embeddings.</p>

<hr>

<h2>Design Goals</h2>
<ul>
  <li><strong>Familiar syntax</strong> -- SQL-like, so anyone who knows SQL can use it immediately.</li>
  <li><strong>Built on DuckDB</strong> -- Leverage DuckDB's SQL engine for columnar queries over Parquet data.</li>
  <li><strong>Full-text extensions</strong> -- Add <code>CONTAINS()</code> for keyword search once a full-text index exists.</li>
  <li><strong>CLI-first</strong> -- Execute queries from the terminal. Results in table, JSON, or CSV format.</li>
</ul>

<hr>

<h2>Example Queries (Vision)</h2>
<p>These illustrate what OQL might look like. None of these work yet.</p>

<h3>Columnar Index Queries</h3>
<p>These would translate to DuckDB SQL against Parquet files:</p>

<pre><code>-- Count pages by TLD
SELECT url_host_tld, COUNT(*) as pages
FROM index
GROUP BY url_host_tld
ORDER BY pages DESC
LIMIT 20

-- Find all English pages on a domain
SELECT url, fetch_status, content_languages
FROM index
WHERE url_host_registered_domain = 'example.com'
  AND content_languages LIKE '%eng%'
  AND fetch_status = 200
LIMIT 100

-- Domain size distribution
SELECT url_host_registered_domain as domain,
       COUNT(*) as pages
FROM index
WHERE fetch_status = 200
GROUP BY domain
HAVING pages > 1000
ORDER BY pages DESC
LIMIT 50</code></pre>

<h3>Full-Text Search (Future)</h3>
<p>Would require building a full-text index (not yet started):</p>

<pre><code>-- Keyword search (future)
SELECT url, title
FROM fulltext
WHERE CONTAINS('machine learning tutorial')
LIMIT 20

-- Phrase matching (future)
SELECT url, title
FROM fulltext
WHERE CONTAINS('"web crawling" AND go')
LIMIT 20</code></pre>

<h3>Combined Queries (Future)</h3>
<p>Join columnar metadata with full-text results:</p>

<pre><code>-- Filter full-text results by metadata (future)
SELECT f.url, f.title, i.content_languages
FROM fulltext f
JOIN index i ON f.url = i.url
WHERE CONTAINS('distributed systems')
  AND i.url_host_tld = 'edu'
LIMIT 50</code></pre>

<hr>

<h2>What You Can Do Today</h2>
<p>While OQL is not implemented, you can run equivalent queries directly with DuckDB:</p>

<pre><code># Remote query against CC parquet index (works now)
duckdb -c "
INSTALL httpfs; LOAD httpfs;
SELECT url_host_tld, COUNT(*) as pages
FROM read_parquet('s3://commoncrawl/cc-index/table/cc-main/warc/crawl=CC-MAIN-2026-04/subset=warc/*.parquet')
GROUP BY url_host_tld
ORDER BY pages DESC
LIMIT 20;
"

# Or use the CLI
search cc query --remote --sql "SELECT url_host_tld, COUNT(*) as pages FROM ... LIMIT 20"</code></pre>

<div class="note">
  If you are interested in contributing to the OQL design or implementation, see the <a href="https://github.com/nicholasgasior/gopher-crawl">GitHub repository</a>. Spec discussions happen in the issues.
</div>
`
