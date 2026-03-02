---
slug: recrawler-architecture
title: "Building a 100K-Worker Recrawler in Go"
date: 2026-02-18
summary: "50,000 goroutines, 73 domains, 0.8% success rate. One per-domain semaphore later: 57.5%. Every phase exists because something broke first."
tags: [engineering, go]
---

0.8%. That was my success rate the first time I pointed 50,000 goroutines at 73 domains. Fifty thousand workers, seventy-three targets -- roughly 685 simultaneous connections per domain. Servers took one look at that and said absolutely not.

I added a single constraint -- a per-domain semaphore capped at 8 connections -- and the success rate jumped to 57.5%. A 69x improvement from one number. The rest of this post is about every other mistake I made getting here, because every phase of this recrawler exists because something broke first.

## What is this thing supposed to do?

Common Crawl publishes petabytes of web data, but the index is a snapshot in time. URLs go stale. Domains die. Content changes. If you want fresh data, you need to recrawl -- millions of URLs across tens of thousands of domains without hammering any single server.

The recrawler I built processes 2.5 million URLs in 65 seconds. Here's the architecture, told through the failures that shaped it.

## Three phases, and why they can't overlap

My first prototype ran DNS resolution and HTTP fetching in parallel. The thinking was obvious: start fetching the moment the first domains resolve. Instead, it caused goroutine explosion. The DNS pipeline generated resolved domains faster than HTTP workers could process them, and each resolved domain spawned more work. Memory hit 20+ GB and the process crashed.

So: strict phase separation. DNS finishes completely before probing starts. Probing streams into fetching (but that's the only overlap allowed). This is simpler to reason about, simpler to debug, and prevents the resource explosion that comes from unbounded pipeline parallelism.

<pre class="showcase-visual">
<span class="dim">                    Seed URLs (from CC parquet)</span>
<span class="dim">                              |</span>
<span class="dim">                              v</span>
<span class="blue">  ┌──────────────────────────────────────────────┐</span>
<span class="blue">  │</span>  <span class="hl">Phase 1: Batch DNS Resolution</span>              <span class="blue">│</span>
<span class="blue">  │</span>  20K workers, multi-server confirmation      <span class="blue">│</span>
<span class="blue">  │</span>  CF 1.1.1.1 → Google 8.8.8.8 → stdlib      <span class="blue">│</span>
<span class="blue">  │</span>  sync.Map: resolved / dead / timeout         <span class="blue">│</span>
<span class="blue">  └────────────────────┬─────────────────────────┘</span>
<span class="dim">              ┌────────┴────────┐</span>
<span class="dim">              v                 v</span>
<span class="green">        resolved IPs</span>       <span class="amber">dead domains</span>
<span class="dim">              |              (skip)</span>
<span class="dim">              v</span>
<span class="blue">  ┌──────────────────────────────────────────────┐</span>
<span class="blue">  │</span>  <span class="hl">Phase 2: Streaming Probe + Feed</span>            <span class="blue">│</span>
<span class="blue">  │</span>  5K workers, TCP connect + TLS handshake     <span class="blue">│</span>
<span class="blue">  │</span>  timeout = alive, refused/reset = dead        <span class="blue">│</span>
<span class="blue">  │</span>  URLs streamed immediately to feed channel    <span class="blue">│</span>
<span class="blue">  └────────────────────┬─────────────────────────┘</span>
<span class="dim">              ┌────────┴────────┐</span>
<span class="dim">              v                 v</span>
<span class="green">        alive domains</span>      <span class="amber">dead domains</span>
<span class="dim">              |              (skip)</span>
<span class="dim">              v</span>
<span class="blue">  ┌──────────────────────────────────────────────┐</span>
<span class="blue">  │</span>  <span class="hl">Phase 3: Direct HTTP Workers</span>                <span class="blue">│</span>
<span class="blue">  │</span>  50K workers, 8 max conns/domain              <span class="blue">│</span>
<span class="blue">  │</span>  64 transport shards, 500ms TLS timeout       <span class="blue">│</span>
<span class="blue">  │</span>  Round-robin URL interleaving                 <span class="blue">│</span>
<span class="blue">  └────────────────────┬─────────────────────────┘</span>
<span class="dim">                       v</span>
<span class="blue">  ┌──────────────────────────────────────────────┐</span>
<span class="blue">  │</span>  <span class="hl">16-Shard ResultDB (DuckDB)</span>                  <span class="blue">│</span>
<span class="blue">  │</span>  batch-VALUES inserts, 500 rows/stmt          <span class="blue">│</span>
<span class="blue">  └──────────────────────────────────────────────┘</span>
</pre>

## Phase 1: Why I resolve every domain before fetching anything

A single Common Crawl parquet file can contain 100K+ unique domains. Most of them are dead. Before fetching a single URL, I resolve every unique domain with 20,000 concurrent DNS workers. Each worker pulls a domain from a channel, resolves it, and writes the result to one of three categories using `sync.Map` for lock-free concurrent access:

- **Resolved** -- at least one IP address returned. IPs cached for later phases.
- **Dead** -- NXDOMAIN confirmed by multiple servers. Permanently excluded.
- **Timeout** -- DNS servers didn't respond. Saved to the DNS cache for retry next run.

### The single-resolver trap

Early runs used only Cloudflare's resolver. Then I noticed 5-7% of "dead" domains were actually alive when checked with Google. Geographic DNS anycast, rate limiting, temporary outages -- a single resolver lies to you in a dozen different ways.

Now I use a three-tier confirmation chain: Cloudflare (1.1.1.1), then Google (8.8.8.8), then Go's stdlib resolver. A domain is only marked dead when all three agree. Without this, I'd skip thousands of valid domains per run.

### Don't let HTTP failures touch the DNS cache

<div class="note">
  <strong>Key design rule:</strong> HTTP failures never contaminate the DNS cache. <code>MergeHTTPDead</code> is a no-op. If an HTTP fetch fails (timeout, 503, connection refused), the domain stays in the DNS cache as resolved. Only DNS-level evidence can mark a domain as dead.
</div>

I learned this the hard way. An early optimization tried to merge HTTP-level failures back into the DNS cache as "dead" domains. The idea was to save future DNS lookups. In practice, it was catastrophic: a server returning 503 during a maintenance window would be marked as DNS-dead, and all future crawls would skip it forever. The DNS cache is a DNS cache -- only DNS evidence goes in.

The cache persists to DuckDB between runs. Next crawl, I load it first and skip re-resolving known domains. Saves minutes on subsequent runs against the same data.

## Phase 2: Why probing and feeding happen simultaneously

DNS resolution tells me an IP address exists. It doesn't tell me whether the server is accepting connections. Phase 2 sends 5,000 concurrent probers to perform TCP connect + TLS handshake on every resolved domain.

### Being conservative about what "dead" means

The probe logic is deliberately conservative. A domain is only marked dead on definitive failure signals: connection refused, connection reset, or DNS error during connect. If the probe times out, the domain is classified as **alive**. A timeout could mean the server is slow, overloaded, or rate-limiting me. I'd rather try the fetch and fail fast than skip a domain that might have content.

### The 3x speedup I should have built on day one

The original design was sequential: probe all domains, collect the results into a list, shuffle, then feed URLs to HTTP workers. This meant HTTP workers sat completely idle until every single probe completed -- even if 90% of probes finished in the first few seconds.

The fix was obvious in hindsight. Push URLs to the feed channel the moment a probe succeeds. Workers start fetching immediately. Probing and fetching overlap in time, but they're operating on *different domains* -- the probe is checking domains not yet confirmed, while workers fetch URLs from already-confirmed domains.

<pre class="showcase-visual">
<span class="dim">// Old approach: sequential probe → collect → shuffle → feed</span>
<span class="amber">probeAll(domains)</span>           <span class="dim">// 120s -- wait for ALL probes</span>
<span class="amber">shuffle(aliveDomains)</span>       <span class="dim">//   5s -- reorganize</span>
<span class="amber">feedToWorkers(urls)</span>         <span class="dim">//  60s -- now workers can start</span>
<span class="dim">// Total: 185s</span>

<span class="dim">// New approach: streaming probe → immediate feed</span>
<span class="green">probeStreaming(domains, func(domain) {</span>
<span class="green">    feedCh &lt;- urlsForDomain(domain)</span>  <span class="dim">// workers start instantly</span>
<span class="green">})</span>
<span class="dim">// Total: 65s (probing + fetching overlap)</span>
</pre>

65 seconds instead of 185 seconds for 2.5M URLs. Nearly 3x faster, with zero change to the HTTP worker logic. If I were starting over, I'd build the streaming pipeline from day one.

## Phase 3: 50,000 workers and the constraint that makes them work

This is the core of the recrawler: 50,000 concurrent goroutines pulling URLs from a shared feed channel. The number 50K isn't arbitrary -- it's the sweet spot where throughput is maximized without overwhelming Go's scheduler or exhausting file descriptors.

### 0.8% -- the number that changed everything

I already mentioned this, but it's worth dwelling on. My first version had no per-domain limits. 50K workers. 73 domains. ~685 concurrent connections per domain. The result: **0.8% success rate**. Servers refused connections, rate-limited me, or simply collapsed under the load.

The fix was per-domain semaphores. Each domain gets a buffered channel with capacity 8. Before fetching, a worker acquires a slot. If all 8 are taken, the worker blocks until one frees up:

<pre class="showcase-visual">
<span class="dim">// Per-domain semaphore: limit concurrent connections per domain</span>
<span class="blue">type</span> <span class="hl">DomainSemaphore</span> <span class="blue">struct</span> {
    sems <span class="blue">sync.Map</span>  <span class="dim">// domain -> chan struct{}</span>
    max  <span class="blue">int</span>
}

<span class="blue">func</span> (ds *<span class="hl">DomainSemaphore</span>) <span class="green">Acquire</span>(domain <span class="blue">string</span>) {
    v, _ := ds.sems.LoadOrStore(domain, <span class="green">make</span>(<span class="blue">chan struct{}</span>, ds.max))
    sem := v.(<span class="blue">chan struct{}</span>)
    sem <span class="amber">&lt;-</span> <span class="blue">struct{}{}</span>  <span class="dim">// blocks if all slots taken</span>
}

<span class="blue">func</span> (ds *<span class="hl">DomainSemaphore</span>) <span class="green">Release</span>(domain <span class="blue">string</span>) {
    v, ok := ds.sems.Load(domain)
    <span class="blue">if</span> ok {
        <span class="amber">&lt;-</span>v.(<span class="blue">chan struct{}</span>)  <span class="dim">// free one slot</span>
    }
}

<span class="dim">// Worker loop</span>
<span class="blue">func</span> <span class="green">worker</span>(feedCh <span class="blue">&lt;-chan</span> *URL, sem *<span class="hl">DomainSemaphore</span>, results *ResultDB) {
    <span class="blue">for</span> url := <span class="blue">range</span> feedCh {
        sem.<span class="green">Acquire</span>(url.Domain)
        resp, err := <span class="green">fetch</span>(url)
        sem.<span class="green">Release</span>(url.Domain)

        <span class="blue">if</span> err != <span class="blue">nil</span> {
            <span class="green">handleFailure</span>(url, err)
            <span class="blue">continue</span>
        }
        results.<span class="green">Write</span>(url, resp)
    }
}
</pre>

The numbers tell the story:

<table>
  <thead>
    <tr>
      <th>Configuration</th>
      <th>Success Rate</th>
      <th>Improvement</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>50K workers, no per-domain limit</td>
      <td><strong>0.8%</strong></td>
      <td>Baseline</td>
    </tr>
    <tr>
      <td>50K workers, 8 max conns/domain</td>
      <td><strong>57.5%</strong></td>
      <td>69x</td>
    </tr>
  </tbody>
</table>

<div class="note">
  <strong>The lesson:</strong> Raw concurrency is useless if you flood individual targets. The constraint -- 8 connections per domain -- is what makes 50K total workers viable. Without it, even 1K workers would fail against a concentrated domain set.
</div>

### 64 transport shards because Go's http.Transport has a mutex

Go's `http.Transport` maintains internal connection pools and mutexes. At 50K goroutines, a single transport becomes a bottleneck. I shard across 64 transports, assigning each URL to a transport by hash of its domain. Each transport gets its own TLS config with a 500ms handshake timeout, its own connection pool, its own idle connection limits.

### Cutting losses with domain fail thresholds

If a domain returns 2 consecutive failures (timeouts, connection errors, HTTP 429/503), it's marked "failed" for the remainder of the run. All remaining URLs for that domain get skipped. This prevents a single broken domain from consuming worker time and slots. The threshold of 2 balances between tolerating a single transient error and cutting losses quickly.

## Why feed order matters as much as fetch speed

A naive feed sends all URLs for domain A, then all for domain B, and so on. This means the first N thousand requests all hit the same domain -- exactly the flooding pattern I'm trying to avoid, even with per-domain semaphores.

The fix is URL interleaving: round-robin across domains. Three domains with 100, 200, and 50 URLs? Feed order is A, B, C, A, B, C, A, B, ... until C runs out, then A, B, A, B, ... until A runs out, then the remaining B URLs. Load is distributed from the very first URL.

<pre class="showcase-visual">
<span class="dim">// Round-robin URL interleaving across domains</span>
<span class="blue">func</span> <span class="green">interleave</span>(domainURLs <span class="blue">map[string]</span>[]*URL) <span class="blue">&lt;-chan</span> *URL {
    ch := <span class="green">make</span>(<span class="blue">chan</span> *URL, <span class="hl">8192</span>)
    <span class="blue">go func</span>() {
        <span class="blue">defer</span> <span class="green">close</span>(ch)
        <span class="dim">// Build domain iterators</span>
        type iter <span class="blue">struct</span> {
            urls []*URL
            idx  <span class="blue">int</span>
        }
        iters := <span class="green">make</span>([]iter, <span class="hl">0</span>, <span class="green">len</span>(domainURLs))
        <span class="blue">for</span> _, urls := <span class="blue">range</span> domainURLs {
            iters = <span class="green">append</span>(iters, iter{urls: urls})
        }
        <span class="dim">// Round-robin until all exhausted</span>
        <span class="blue">for</span> <span class="green">len</span>(iters) > <span class="hl">0</span> {
            alive := iters[:<span class="hl">0</span>]
            <span class="blue">for</span> _, it := <span class="blue">range</span> iters {
                ch <span class="amber">&lt;-</span> it.urls[it.idx]
                it.idx++
                <span class="blue">if</span> it.idx < <span class="green">len</span>(it.urls) {
                    alive = <span class="green">append</span>(alive, it)
                }
            }
            iters = alive
        }
    }()
    <span class="blue">return</span> ch
}
</pre>

## Where the results go: 16-shard DuckDB

At thousands of writes per second, a single DuckDB file becomes a bottleneck. I shard across 16 DuckDB files, assigning each URL by hashing its domain. Each shard has a dedicated flusher goroutine that accumulates rows and writes them in batch-VALUES inserts of 500 rows per statement.

500 rows per batch was chosen empirically -- larger batches increase latency between fetch and write confirmation, while smaller batches don't fully amortize statement preparation cost.

<pre class="showcase-visual">
<span class="dim">~/data/common-crawl/CC-MAIN-2026-04/recrawl/</span>
  shard_00.duckdb   <span class="dim">~75 MB each</span>
  shard_01.duckdb
  <span class="dim">...</span>
  shard_15.duckdb
  <span class="dim">total: ~1.2 GB</span>

<span class="dim">~/data/common-crawl/CC-MAIN-2026-04/dns.duckdb</span>
  <span class="green">42,891</span> resolved    <span class="dim">(domain → []IP)</span>
  <span class="amber">31,204</span> dead        <span class="dim">(NXDOMAIN, multi-server confirmed)</span>
     <span class="hl">876</span> timeout     <span class="dim">(saved for retry on next run)</span>
</pre>

## 97% of those domains are dead, and that's not a bug

One thing that genuinely surprised me: Common Crawl parquet files are TLD-partitioned. They're not randomly distributed. File 299 is almost entirely `.cn` domains. File 0 is mostly `.ru`. File 50 is predominantly `.fi`.

When you recrawl from outside those geographic regions, about 97% of the domains are dead. Many country-code TLD sites only resolve within their region's DNS infrastructure, or they've simply gone offline since Common Crawl archived them. This isn't a failure of the recrawler -- it's the reality of the internet.

<div class="note">
  <strong>Practical implication:</strong> If you're benchmarking a recrawler, pick your CC parquet file carefully. File 50 (mixed TLDs) gives realistic numbers. File 299 (.cn only) will show 97%+ dead domains regardless of your architecture.
</div>

## The actual numbers

All numbers below are from a real crawl run, not projections. Seed data extracted directly from Common Crawl parquet files using `read_parquet()` with zero DuckDB import.

<table>
  <thead>
    <tr>
      <th>Metric</th>
      <th>Value</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Total URLs processed</strong></td>
      <td>2.5M</td>
    </tr>
    <tr>
      <td><strong>Total time (streaming)</strong></td>
      <td>65s</td>
    </tr>
    <tr>
      <td><strong>Peak throughput</strong></td>
      <td>275 pages/s</td>
    </tr>
    <tr>
      <td><strong>DNS resolved</strong></td>
      <td>42,891 domains</td>
    </tr>
    <tr>
      <td><strong>DNS dead</strong></td>
      <td>31,204 domains</td>
    </tr>
    <tr>
      <td><strong>DNS timeout</strong></td>
      <td>876 domains</td>
    </tr>
    <tr>
      <td><strong>Domains alive after probe</strong></td>
      <td>28,445</td>
    </tr>
    <tr>
      <td><strong>Pages fetched successfully</strong></td>
      <td>147,231</td>
    </tr>
    <tr>
      <td><strong>ResultDB size</strong></td>
      <td>1.2 GB (16 shards)</td>
    </tr>
    <tr>
      <td><strong>HTTP workers</strong></td>
      <td>50,000</td>
    </tr>
    <tr>
      <td><strong>Max conns per domain</strong></td>
      <td>8</td>
    </tr>
    <tr>
      <td><strong>Transport shards</strong></td>
      <td>64</td>
    </tr>
  </tbody>
</table>

### How streaming changed the timeline

<table>
  <thead>
    <tr>
      <th>Phase</th>
      <th>Sequential (v1)</th>
      <th>Streaming (v2)</th>
      <th>Speedup</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>DNS resolution</strong></td>
      <td>12s</td>
      <td>12s</td>
      <td>1x (unchanged)</td>
    </tr>
    <tr>
      <td><strong>Probe + feed</strong></td>
      <td>120s + 5s + 60s = 185s</td>
      <td>65s (overlapped)</td>
      <td>2.8x</td>
    </tr>
    <tr>
      <td><strong>Total pipeline</strong></td>
      <td>~200s</td>
      <td>~77s</td>
      <td>2.6x</td>
    </tr>
  </tbody>
</table>

## What I'd do differently

If starting over, I'd build the streaming probe-to-feed pipeline from day one. The sequential design was never fast enough, and retrofitting the streaming approach meant rethinking the entire data flow between probe and feed.

I'd also start with per-domain semaphores immediately. Running it wide open and watching 0.8% success rate was informative for benchmarking but wasted significant development time debugging connection failures that were entirely self-inflicted.

The three-phase architecture -- DNS, probe, fetch -- is the right design. Strict phase separation is simpler, debuggable, and prevents resource explosion. The constraint that no phases overlap except probe-to-feed streaming isn't a limitation. It's a feature.

## Running it yourself

```
# Recrawl the latest CC parquet file
search cc recrawl --last

# Recrawl a specific file (e.g., file 50 for mixed TLDs)
search cc recrawl --file 50

# Tune connection limits
search cc recrawl --last --max-conns-per-domain 4 --domain-fail-threshold 3

# Use sample mode (N evenly-spaced files)
search cc recrawl --sample 5
```

Results land at `~/data/common-crawl/{CrawlID}/recrawl/` (16-shard DuckDB) with the DNS cache at `~/data/common-crawl/{CrawlID}/dns.duckdb`.
