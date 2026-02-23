---
slug: per-domain-flooding
title: "685 Connections Per Domain (and Why That Broke Everything)"
date: 2026-02-12
summary: "50,000 goroutines, 73 domains, and a 0.8% success rate that rewrote the entire architecture."
tags: [engineering, go]
---

0.8%.

I stared at the terminal for a good thirty seconds, convinced I'd misread it. My recrawler had just finished its first real benchmark -- 50,000 concurrent HTTP workers pulling URLs from a Common Crawl parquet file -- and the success rate was zero point eight percent. Out of roughly 170,000 fetch attempts, fewer than 1,400 got a response. The rest were connection refused, connection reset, or just silence.

I checked the logs. No crashes, no panics, no OOM kills. The code was doing exactly what I'd told it to do. Which, it turned out, was the problem.

## What does 50,000 / 73 equal?

The parquet file I'd picked for testing contained URLs from 73 domains. Seventy-three. I had 50,000 goroutines racing to fetch URLs, and they were distributed across those 73 domains by whatever order the channel happened to serve them. Quick napkin math: 50,000 / 73 = ~685 concurrent connections per domain.

Six hundred and eighty-five simultaneous TCP connections hitting the same server. From a single IP address. Most web servers are configured to handle maybe 50-100 concurrent connections from a single client before they start refusing. Some are more generous, some less. But 685? Nobody is going to tolerate that. The servers were doing the rational thing: slamming the door.

This wasn't a bug in my code. It was a bug in my mental model. I'd been thinking about total concurrency -- 50K workers, that sounds fast! -- when I should have been thinking about concurrency *per target*. The distinction matters enormously, and it took a 0.8% success rate to teach me.

## A buffered channel is a semaphore

The fix is a per-domain semaphore. Before any worker fetches a URL, it must acquire a slot on that domain's semaphore. If all slots are taken, the worker blocks. When the fetch completes, the slot is released.

Go gives you this for free with buffered channels. A channel of capacity N is a counting semaphore: writing to it acquires a slot (blocks when full), reading from it releases one. No mutexes, no condition variables, no third-party libraries.

<pre class="showcase-visual">
<span class="blue">type</span> <span class="hl">Recrawler</span> <span class="blue">struct</span> {
    <span class="dim">// ... other fields ...</span>

    <span class="dim">// Per-domain connection limiter: prevents flooding individual servers.</span>
    <span class="dim">// Pre-created in Run() to avoid mutex contention during fetch.</span>
    domainSems   <span class="blue">map[string]chan struct{}</span>
    domainSemsMu <span class="blue">sync.RWMutex</span>
}
</pre>

The semaphore map is pre-populated before any worker starts. During `Run()`, I iterate over every domain in the seed data and create a buffered channel with capacity equal to `MaxConnsPerDomain` (default: 8). This pre-creation step is critical -- if semaphores were created lazily during fetch, the `sync.RWMutex` on the map would become a contention point with 50K goroutines hitting it simultaneously.

<pre class="showcase-visual">
<span class="dim">// Pre-create domain semaphores before launching workers</span>
<span class="blue">for</span> d := <span class="blue">range</span> domainURLs {
    r.domainSems[d] = <span class="green">make</span>(<span class="blue">chan struct{}</span>, r.config.MaxConnsPerDomain)
}
</pre>

The worker loop is the interesting part. Each worker pulls a URL from the shared feed channel, acquires the domain semaphore, does the fetch, then releases. The `select` on `ctx.Done()` ensures clean shutdown -- if the context is cancelled while a worker is blocked waiting for a semaphore slot, it unblocks immediately instead of hanging forever.

<pre class="showcase-visual">
<span class="blue">func</span> (r *<span class="hl">Recrawler</span>) <span class="green">worker</span>(ctx context.Context, client *http.Client, urls <span class="blue">&lt;-chan</span> SeedURL) <span class="blue">error</span> {
    <span class="blue">for</span> {
        <span class="blue">select</span> {
        <span class="blue">case</span> <span class="amber">&lt;-</span>ctx.Done():
            <span class="blue">return</span> ctx.Err()
        <span class="blue">case</span> seed, ok := <span class="amber">&lt;-</span>urls:
            <span class="blue">if</span> !ok {
                <span class="blue">return nil</span>
            }
            <span class="blue">if</span> r.<span class="green">isDomainDead</span>(seed.Domain) {
                <span class="blue">continue</span>
            }
            <span class="dim">// Acquire per-domain slot (blocks if all 8 taken)</span>
            sem := r.<span class="green">domainSem</span>(seed.Domain)
            <span class="blue">select</span> {
            <span class="blue">case</span> sem <span class="amber">&lt;-</span> <span class="blue">struct{}{}</span>:
            <span class="blue">case</span> <span class="amber">&lt;-</span>ctx.Done():
                <span class="blue">return</span> ctx.Err()
            }
            r.<span class="green">fetchOne</span>(ctx, client, seed)
            <span class="amber">&lt;-</span>sem  <span class="dim">// release slot</span>
        }
    }
}
</pre>

Notice the double `select`. The outer one pulls from the URL channel or exits on cancellation. The inner one acquires the semaphore or exits on cancellation. Without that inner select, a worker could block on a saturated semaphore even after the context has been cancelled, preventing graceful shutdown.

## What 8 connections buys you

I re-ran the benchmark with `MaxConnsPerDomain` set to 8. Same 50K workers. Same 73 domains. Same parquet file.

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

57.5%. A 69x improvement. From a single integer parameter.

The remaining 42.5% failure is not from flooding -- it's from domains that are genuinely dead, timing out, or returning errors for legitimate reasons. When you're recrawling Common Crawl data that can be months old, a large chunk of URLs simply don't exist anymore. 57.5% is a realistic live rate for aged CC seed data with mixed TLDs.

<div class="note">
  <strong>Why 8?</strong> I tested values from 2 to 64. Below 4, throughput drops because workers spend too much time blocked on semaphores. Above 16, some servers start rate-limiting again. 8 was the sweet spot: high enough to keep workers busy, low enough that no single server feels overwhelmed. It's configurable via <code>--max-conns-per-domain</code>.
</div>

## When a domain keeps failing

Even with semaphores, some domains are black holes. They accept the TCP connection, maybe even complete the TLS handshake, then hang for the full timeout on every request. With 8 connections per domain and a 3-second timeout, that's 8 worker-slots burned for 3 seconds each, over and over, for a domain that will never return useful content.

The fix is a domain fail threshold. If a domain accumulates N consecutive failures without a single success, it's marked dead for the rest of the run. All remaining URLs for that domain are skipped.

<pre class="showcase-visual">
<span class="dim">// Per-domain timeout tracking: sync.Map + atomic for 50K workers</span>
domainFailCounts <span class="blue">sync.Map</span>  <span class="dim">// domain -> *atomic.Int32</span>
domainSucceeded  <span class="blue">sync.Map</span>  <span class="dim">// domain -> true</span>

<span class="blue">func</span> (r *<span class="hl">Recrawler</span>) <span class="green">recordDomainTimeout</span>(domain <span class="blue">string</span>) {
    <span class="dim">// Domain has succeeded before? Immune to timeout-kill.</span>
    <span class="blue">if</span> _, ok := r.domainSucceeded.Load(domain); ok {
        <span class="blue">return</span>
    }
    counter, _ := r.domainFailCounts.LoadOrStore(domain, &amp;<span class="blue">atomic.Int32</span>{})
    fails := counter.(*<span class="blue">atomic.Int32</span>).Add(<span class="hl">1</span>)
    <span class="blue">if</span> <span class="green">int</span>(fails) >= r.config.DomainFailThreshold {
        r.<span class="green">markDomainDead</span>(domain, <span class="amber">"http_timeout_killed"</span>)
    }
}

<span class="blue">func</span> (r *<span class="hl">Recrawler</span>) <span class="green">recordDomainSuccess</span>(domain <span class="blue">string</span>) {
    r.domainSucceeded.Store(domain, <span class="blue">true</span>)
}
</pre>

Two details worth calling out. First, the success immunity: once a domain returns even one valid response, it can never be killed by the fail threshold. This prevents a transient blip (one slow request between many fast ones) from poisoning a healthy domain. Second, the data structures: `sync.Map` for lock-free reads in the hot path, `atomic.Int32` for the counter. With 50K goroutines, a regular mutex here would be a disaster.

The default threshold is 2. That means two consecutive failures with zero successes. It's aggressive, but in practice the domains that fail twice in a row without ever succeeding are genuinely unreachable. I tested with thresholds of 3, 5, and 10 -- higher values barely changed the success rate but significantly increased wasted fetch time on dead domains.

## Why one http.Transport isn't enough

Go's `http.Transport` is goroutine-safe. The documentation says so. You can share one across your entire application and it'll work fine. But "goroutine-safe" and "performs well under extreme concurrency" are different claims.

Internally, `http.Transport` manages connection pools, TLS session caches, and idle connection lists -- all behind mutexes. At 50K goroutines, those mutexes become hot. Workers spend measurable time waiting for pool locks instead of doing actual work.

The fix is transport sharding. I create 64 independent `http.Transport` instances, each with its own connection pool, TLS config, and idle connection limits. Each worker is assigned to a shard by hashing its worker ID. Each URL is assigned to a shard by hashing its domain. This means all requests to the same domain go through the same transport (good for connection reuse) while spreading lock contention across 64 independent pools.

<pre class="showcase-visual">
<span class="dim">// Create 64 sharded HTTP clients at startup</span>
r.clients = <span class="green">make</span>([]<span class="blue">*http.Client</span>, cfg.TransportShards)
<span class="blue">for</span> i := <span class="blue">range</span> cfg.TransportShards {
    r.clients[i] = r.<span class="green">buildClient</span>(i)
}

<span class="dim">// Each transport shard has independent configuration</span>
transport := &amp;<span class="blue">http.Transport</span>{
    DialContext:           dialFunc,  <span class="dim">// uses cached DNS IPs</span>
    MaxIdleConns:          maxIdlePerShard,
    MaxIdleConnsPerHost:   <span class="hl">50</span>,
    IdleConnTimeout:       <span class="hl">30 * time.Second</span>,
    TLSHandshakeTimeout:   <span class="hl">500 * time.Millisecond</span>,
    ResponseHeaderTimeout: cfg.Timeout,
    ForceAttemptHTTP2:     <span class="blue">false</span>,  <span class="dim">// HTTP/1.1 for one-shot multi-host</span>
    WriteBufferSize:       <span class="hl">4 * 1024</span>,
    ReadBufferSize:        <span class="hl">8 * 1024</span>,
}
</pre>

A subtle detail: `ForceAttemptHTTP2` is set to `false`. For the recrawler -- which hits thousands of different servers once or twice each -- HTTP/1.1 is faster than HTTP/2. HTTP/2 multiplexing shines when you're sending many requests to the *same* server (the domain crawler uses it). For one-shot multi-host fetching, the HTTP/2 connection setup overhead outweighs the multiplexing benefit.

Another detail: the custom `dialFunc` inside each transport uses cached DNS IPs. After the batch DNS resolution phase, every resolved domain has its IP addresses stored in memory. The dial function checks this cache before calling the system resolver. This eliminates runtime DNS lookups entirely during the fetch phase -- the OS resolver never sees any of these domains.

## Why feed order matters

Per-domain semaphores prevent you from having 685 connections to one server. But there's a subtler problem: if you feed all URLs for domain A, then all URLs for domain B, then all URLs for domain C, the first thousand requests all target the same domain. Even with a semaphore capping at 8 concurrent connections, all 50K workers are contending for those 8 slots on domain A. The other 72 domains sit idle.

The fix is URL interleaving -- round-robin across domains. Instead of AAAA...BBBB...CCCC, the feed order becomes ABCABCABC. Each worker pulls the next URL from the channel and it's from a different domain than the last one. Load distributes across the full domain pool from the very first fetch.

<pre class="showcase-visual">
<span class="dim">// Round-robin URL interleaving across alive domains</span>
cursors := <span class="green">make</span>([]<span class="blue">int</span>, <span class="green">len</span>(aliveList))
remaining := <span class="green">len</span>(aliveList)

<span class="blue">for</span> remaining > <span class="hl">0</span> {
    remaining = <span class="hl">0</span>
    <span class="blue">for</span> i, ad := <span class="blue">range</span> aliveList {
        <span class="blue">if</span> cursors[i] < <span class="green">len</span>(ad.urls) {
            <span class="blue">select</span> {
            <span class="blue">case</span> urlCh <span class="amber">&lt;-</span> ad.urls[cursors[i]]:
                cursors[i]++
            <span class="blue">case</span> <span class="amber">&lt;-</span>ctx.Done():
                <span class="blue">return</span>
            }
            <span class="blue">if</span> cursors[i] < <span class="green">len</span>(ad.urls) {
                remaining++
            }
        }
    }
}
</pre>

This is a simple cursor-based round-robin. Each domain has a cursor tracking how far into its URL list we've fed. On each pass, we emit one URL from each domain that still has remaining URLs. When a domain is exhausted, it drops out. The loop continues until all domains are drained.

Think of it like dealing cards. You don't give all 13 cards to player one, then all 13 to player two. You deal one card to each player in turn. Same principle, applied to 73 domains and 170,000 URLs.

## Putting the numbers together

Here's the full picture of what each constraint contributes:

<table>
  <thead>
    <tr>
      <th>Constraint</th>
      <th>What it prevents</th>
      <th>Implementation</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Per-domain semaphore (8)</strong></td>
      <td>Connection flooding individual servers</td>
      <td>Buffered channel per domain, pre-created</td>
    </tr>
    <tr>
      <td><strong>Domain fail threshold (2)</strong></td>
      <td>Wasting workers on dead domains</td>
      <td>sync.Map + atomic.Int32, success immunity</td>
    </tr>
    <tr>
      <td><strong>Transport sharding (64)</strong></td>
      <td>Mutex contention on connection pools</td>
      <td>64 independent http.Transport instances</td>
    </tr>
    <tr>
      <td><strong>URL interleaving</strong></td>
      <td>Sequential domain flooding in feed order</td>
      <td>Round-robin cursor across domain lists</td>
    </tr>
  </tbody>
</table>

Remove any one of these and performance degrades significantly. Remove the per-domain semaphore and you're back to 0.8%. Remove the fail threshold and dead domains burn 30-40% of your worker capacity. Remove transport sharding and mutex contention drops throughput by 20-30% at 50K workers. Remove interleaving and the first few seconds of every run are wasted on a single domain.

## The deeper lesson

I spent two days after that 0.8% benchmark thinking I had a networking problem. Maybe my TCP stack wasn't tuned right. Maybe I needed to increase file descriptors. Maybe I needed a faster DNS resolver. I was looking at the infrastructure when the problem was the application logic.

Raw concurrency is not throughput. 50,000 goroutines pointed at 73 servers is a denial-of-service attack, not a crawler. The constraint -- 8 connections per domain, not 685 -- is what makes the system *work*. It's what transforms 50K goroutines from a liability into an asset.

There's an analogy to highway traffic here. A highway with 50,000 cars and no speed limit doesn't move faster than one with a 60 mph limit. It moves slower, because without constraints the system jams. Speed limits, lane discipline, on-ramp metering -- these are constraints that enable throughput. Per-domain semaphores are on-ramp metering for web crawlers.

Every subsequent optimization -- transport sharding, fail thresholds, URL interleaving -- followed from the same insight. The system got faster by adding rules, not by removing them. By the time I'd added all four constraints, 50K workers were a real asset: enough concurrency to saturate available bandwidth across thousands of domains, without overwhelming any single one.

0.8% to 57.5%. Sixty-nine times better. From constraints.
