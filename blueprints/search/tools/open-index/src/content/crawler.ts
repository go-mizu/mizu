import { icons, cardIcon } from '../icons'

export const crawlerPage = `
<h2>OpenIndexBot</h2>
<p>OpenIndexBot is the open-source web crawler powering OpenIndex. It is built in Go as part of the Mizu ecosystem and consists of two crawlers: a high-throughput recrawler for bulk URL processing, and a domain crawler for deep single-site crawling.</p>

<div class="note">
  <strong>Website owners:</strong> If you see requests from OpenIndexBot and want to control access, see the <a href="#blocking">Blocking OpenIndexBot</a> section below.
</div>

<h2>User-Agent String</h2>
<pre><code>OpenIndexBot/1.0 (+https://open-index.go-mizu.workers.dev/crawler)</code></pre>

<p>Full HTTP request headers:</p>
<pre><code>User-Agent: OpenIndexBot/1.0 (+https://open-index.go-mizu.workers.dev/crawler)
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
Accept-Encoding: gzip, br
Accept-Language: en-US,en;q=0.5
Connection: keep-alive</code></pre>

<h2 id="blocking">Robots.txt Compliance</h2>
<p>OpenIndexBot respects the <a href="https://www.rfc-editor.org/rfc/rfc9309">Robots Exclusion Protocol (RFC 9309)</a>. To block it:</p>

<pre><code># Block OpenIndexBot from all pages
User-agent: OpenIndexBot
Disallow: /</code></pre>

<p>Selective blocking:</p>
<pre><code># Block specific directories
User-agent: OpenIndexBot
Disallow: /private/
Disallow: /admin/
Disallow: /api/
Allow: /api/public/</code></pre>

<h2>Technical Architecture</h2>
<p>The crawler has two modes, each optimized for a different use case.</p>

<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('globe')} <span>Recrawler</span></div>
    <h3>Bulk URL Processing</h3>
    <p>Takes seed URLs (e.g., from Common Crawl index) and recrawls them in bulk. Designed for millions of URLs across thousands of domains.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('cpu')} <span>Domain Crawler</span></div>
    <h3>Deep Single-Site</h3>
    <p>Crawls a single domain using HTTP/2 multiplexing. Bloom filter frontier, link extraction, resumable state. Best for exhaustive site crawling.</p>
  </div>
</div>

<h3>Recrawler Specifications</h3>
<table>
  <thead>
    <tr>
      <th>Parameter</th>
      <th>Value</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>HTTP workers</strong></td>
      <td>Up to 100,000 concurrent</td>
    </tr>
    <tr>
      <td><strong>DNS workers</strong></td>
      <td>20,000 concurrent</td>
    </tr>
    <tr>
      <td><strong>Probe workers</strong></td>
      <td>5,000 concurrent</td>
    </tr>
    <tr>
      <td><strong>Max connections per domain</strong></td>
      <td>8 (configurable, default)</td>
    </tr>
    <tr>
      <td><strong>Domain fail threshold</strong></td>
      <td>2 consecutive failures before skipping</td>
    </tr>
    <tr>
      <td><strong>Probe timeout</strong></td>
      <td>3 seconds (conservative: timeout = alive)</td>
    </tr>
    <tr>
      <td><strong>TLS timeout</strong></td>
      <td>500ms</td>
    </tr>
    <tr>
      <td><strong>Transport shards</strong></td>
      <td>64</td>
    </tr>
    <tr>
      <td><strong>ResultDB shards</strong></td>
      <td>16 DuckDB files</td>
    </tr>
  </tbody>
</table>

<h3>Domain Crawler Specifications</h3>
<table>
  <thead>
    <tr>
      <th>Parameter</th>
      <th>Value</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Protocol</strong></td>
      <td>HTTP/2 with multiplexing</td>
    </tr>
    <tr>
      <td><strong>Frontier</strong></td>
      <td>Bloom filter (dedup) + channel-based queue</td>
    </tr>
    <tr>
      <td><strong>Concurrency</strong></td>
      <td>Configurable workers + max connections</td>
    </tr>
    <tr>
      <td><strong>State</strong></td>
      <td>Resumable via state.duckdb</td>
    </tr>
    <tr>
      <td><strong>Peak throughput</strong></td>
      <td>275 pages/s (measured on kenh14.vn)</td>
    </tr>
  </tbody>
</table>

<h2>Pipeline Stages</h2>
<p>The recrawler operates in three sequential stages:</p>

<h3>Stage 1: Batch DNS Resolution</h3>
<pre><code># 20K concurrent DNS workers
# Multi-server confirmation: Cloudflare -> Google -> stdlib
# Results: resolved (IPs), dead (NXDOMAIN), timeout (saved for reuse)
# Dead domains are skipped entirely in later stages</code></pre>

<h3>Stage 2: Streaming Probe</h3>
<pre><code># 5K concurrent probers check if resolved hosts accept connections
# Conservative: timeout = alive (only refused/reset/DNS error = dead)
# URLs are fed IMMEDIATELY to HTTP workers as probes succeed
# No waiting for all probes to complete -- streaming pipeline</code></pre>

<h3>Stage 3: HTTP Fetch</h3>
<pre><code># Up to 100K concurrent workers
# Per-domain semaphores (8 max conns/domain, pre-created)
# URL interleaving: round-robin across domains, not sequential
# Results written to 16-shard DuckDB (batch-VALUES, 500 rows/stmt)</code></pre>

<h2>Rate Limiting</h2>
<table>
  <thead>
    <tr>
      <th>Mechanism</th>
      <th>Behavior</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Per-domain connection cap</strong></td>
      <td>Maximum 8 concurrent connections per domain (configurable)</td>
    </tr>
    <tr>
      <td><strong>Domain fail threshold</strong></td>
      <td>2 consecutive failures marks domain as dead for this run</td>
    </tr>
    <tr>
      <td><strong>HTTP 429 / 503</strong></td>
      <td>Domain marked as failing, backed off</td>
    </tr>
    <tr>
      <td><strong>Connection errors</strong></td>
      <td>Counted toward domain fail threshold</td>
    </tr>
    <tr>
      <td><strong>DNS failure</strong></td>
      <td>Domain excluded after multi-server confirmation</td>
    </tr>
  </tbody>
</table>

<h2>Real Performance Data</h2>
<div class="note">
  These numbers are from actual crawl runs, not projections.
</div>

<table>
  <thead>
    <tr>
      <th>Metric</th>
      <th>Recrawler</th>
      <th>Domain Crawler</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Peak throughput</strong></td>
      <td>Thousands of URLs/s (depends on domain mix)</td>
      <td>275 pages/s (kenh14.vn)</td>
    </tr>
    <tr>
      <td><strong>Success rate</strong></td>
      <td>57.5% with 8 max conns/domain (CC data, 97% dead domains)</td>
      <td>~99% on healthy domains</td>
    </tr>
    <tr>
      <td><strong>DNS resolution</strong></td>
      <td>2.5M URLs DNS-resolved in seconds</td>
      <td>N/A (single domain)</td>
    </tr>
    <tr>
      <td><strong>Probe+feed time</strong></td>
      <td>65s for 2.5M URLs (streaming)</td>
      <td>N/A</td>
    </tr>
  </tbody>
</table>

<h2>CLI Usage</h2>
<pre><code># Recrawl from Common Crawl seed data
search cc recrawl --last
search cc recrawl --file 50
search cc recrawl --sample 5

# Crawl a single domain
search crawl-domain example.com --max-pages 1000 --workers 500

# Resume an interrupted crawl
search crawl-domain example.com --resume

# Configure connection limits
search cc recrawl --last --max-conns-per-domain 4 --domain-fail-threshold 3</code></pre>

<h2>Lessons Learned</h2>
<details>
  <summary>Per-domain connection flooding kills success rate</summary>
  <div class="details-body">
    <p>50K workers across 73 domains means ~685 connections per domain. This yielded 0.8% success rate. Adding per-domain semaphores with 8 max connections raised success to 57.5% -- a 69x improvement.</p>
  </div>
</details>

<details>
  <summary>Common Crawl parquet files are TLD-partitioned</summary>
  <div class="details-body">
    <p>CC file 299 is almost entirely .cn domains, file 0 is mostly .ru, file 50 is predominantly .fi. About 97% of CC index domains are dead when recrawled from outside their geographic region.</p>
  </div>
</details>

<details>
  <summary>Streaming probe-to-feed is 3x faster than batch</summary>
  <div class="details-body">
    <p>Old approach: probe ALL domains, collect results, shuffle, feed. New approach: probe in parallel, push URLs immediately as probes succeed. Result: 65s vs 185s for 2.5M URLs.</p>
  </div>
</details>

<details>
  <summary>Never run DNS pipeline and HTTP workers simultaneously</summary>
  <div class="details-body">
    <p>Running both causes goroutine explosion. The correct order: batch DNS first, set cache + dead domains, then directFeed to HTTP workers.</p>
  </div>
</details>

<h2>Contact</h2>
<p>Questions about OpenIndexBot or need help with robots.txt configuration:</p>
<ul>
  <li>GitHub: <a href="https://github.com/nicholasgasior/gopher-crawl/issues">Report an issue</a></li>
</ul>
`
