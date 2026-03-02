---
slug: dead-urls
title: "97% of These URLs Are Dead"
date: 2026-02-14
summary: "Common Crawl parquet file 299 taught us that TLD partitioning and regional DNS make most seed URLs unreachable. The fix changed everything."
tags: [data, engineering]
---

I downloaded Common Crawl parquet file 299, extracted 180,000 seed URLs, and fed them to the recrawler. The DNS resolution phase finished in 12 seconds. Then the numbers came in: 97% of the domains were dead. NXDOMAIN. Gone. As if they'd never existed.

My first thought was that the recrawler was broken. My second thought was that my DNS resolver was misconfigured. My third thought -- the correct one -- was that I'd picked the worst possible test file.

## What's in file 299?

Common Crawl publishes a columnar index as Apache Parquet files on S3. Each monthly crawl produces about 300 parquet files. I'd been grabbing files at random for testing. File 299 seemed as good as any.

It wasn't. File 299 is almost entirely `.cn` domains. Chinese country-code TLD. Tens of thousands of domains ending in `.cn`, `.com.cn`, `.net.cn`. From my server in the US, almost none of them resolve. The DNS servers that know about these domains are in China. The domains themselves are often behind the Great Firewall or have simply gone offline since Common Crawl archived them years ago.

This isn't random. CC parquet files are TLD-partitioned. The files aren't shuffled by domain -- they're grouped. And the grouping happens to concentrate specific country-code TLDs into specific file numbers.

<table>
  <thead>
    <tr>
      <th>File Number</th>
      <th>Dominant TLD</th>
      <th>Dead Rate (from US)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>File 299</strong></td>
      <td>.cn (China)</td>
      <td>~97%</td>
    </tr>
    <tr>
      <td><strong>File 0</strong></td>
      <td>.ru (Russia)</td>
      <td>~90%</td>
    </tr>
    <tr>
      <td><strong>File 50</strong></td>
      <td>.fi (Finland) + mixed</td>
      <td>~40-50%</td>
    </tr>
    <tr>
      <td><strong>Sampled (5 files)</strong></td>
      <td>Mixed TLDs</td>
      <td>~42%</td>
    </tr>
  </tbody>
</table>

File 50 turned out to be predominantly Finnish domains, but with enough mix from other TLDs to give representative numbers. That's the one I use for benchmarking now. If you're building anything that processes CC data and want realistic results, avoid the edges of the file range.

<div class="note">
  <strong>Practical advice:</strong> The <code>--sample N</code> flag in our CLI downloads N evenly-spaced parquet files across the full range. With 5 files spread across 300, you get a cross-section of TLDs that approximates the full distribution. Never benchmark against a single CC parquet file unless you know its TLD composition.
</div>

## The false NXDOMAIN problem

Even after switching to file 50, I noticed something odd. I'd resolve a domain against Cloudflare's 1.1.1.1 and get NXDOMAIN -- no such domain. Then I'd check the same domain against Google's 8.8.8.8 and get a valid IP. The domain was alive. Cloudflare just didn't know about it.

This happens more often than you'd expect. DNS is not a single source of truth. It's a distributed system where different resolvers can have different answers at any given moment. Reasons include geographic anycast routing, resolver caching policies, rate limiting on high-volume lookups, and temporary outages at specific nameservers.

In my benchmarks, 4-7% of domains that Cloudflare reported as NXDOMAIN were actually alive according to Google. At the scale we operate -- 75,000+ unique domains per parquet file -- that's 3,000 to 5,000 domains that a single-resolver approach would incorrectly skip. Thousands of valid URLs, silently discarded.

## Three resolvers, one answer

The fix is multi-server DNS confirmation. I don't trust any single resolver's NXDOMAIN. Instead, I run a three-tier chain: Cloudflare first (it's fast), Google second (it's thorough), Go stdlib as a final fallback (it uses the OS resolver, which may have different routing).

<pre class="showcase-visual">
<span class="dim">// Multi-server DNS: try each resolver sequentially</span>
<span class="dim">// Success on ANY server = domain is alive</span>
<span class="dim">// Dead only if ALL three agree: NXDOMAIN</span>

<span class="blue">func</span> <span class="green">NewDNSResolver</span>(timeout <span class="blue">time.Duration</span>) *<span class="hl">DNSResolver</span> {
    <span class="blue">return</span> &amp;<span class="hl">DNSResolver</span>{
        resolvers: []*<span class="blue">net.Resolver</span>{
            <span class="green">makeResolver</span>(<span class="amber">""</span>, timeout),              <span class="dim">// system DNS</span>
            <span class="green">makeResolver</span>(<span class="amber">"8.8.8.8:53"</span>, timeout),   <span class="dim">// Google</span>
            <span class="green">makeResolver</span>(<span class="amber">"1.1.1.1:53"</span>, timeout),   <span class="dim">// Cloudflare</span>
        },
    }
}
</pre>

The resolution logic for each domain is simple: try resolvers in order, stop at the first success. If a resolver returns NXDOMAIN definitively (not a timeout, not a temporary error -- a real "this domain does not exist"), skip to the next resolver. Only mark the domain dead if every resolver in the chain agrees.

<pre class="showcase-visual">
<span class="blue">func</span> (d *<span class="hl">DNSResolver</span>) <span class="green">ResolveOne</span>(ctx context.Context, domain <span class="blue">string</span>) ([]<span class="blue">string</span>, <span class="blue">bool</span>, <span class="blue">error</span>) {
    <span class="dim">// Check cache first (resolved, dead, or timeout)</span>
    <span class="dim">// ... cache lookup omitted ...</span>

    <span class="blue">var</span> lastErr <span class="blue">error</span>
    <span class="blue">for</span> _, resolver := <span class="blue">range</span> d.resolvers {
        lookupCtx, cancel := context.<span class="green">WithTimeout</span>(ctx, perTimeout)
        addrs, lookupErr := resolver.<span class="green">LookupHost</span>(lookupCtx, domain)
        <span class="green">cancel</span>()

        <span class="blue">if</span> lookupErr == <span class="blue">nil</span> &amp;&amp; <span class="green">len</span>(addrs) > <span class="hl">0</span> {
            <span class="dim">// Success -- cache IPs and return</span>
            <span class="blue">return</span> addrs, <span class="blue">false</span>, <span class="blue">nil</span>
        }

        <span class="blue">if</span> <span class="green">isDefinitelyDead</span>(lookupErr) {
            <span class="dim">// NXDOMAIN from this resolver -- but try the next one</span>
            lastErr = lookupErr
            <span class="blue">continue</span>
        }

        <span class="dim">// Timeout or temporary error -- try next resolver</span>
        lastErr = lookupErr
    }

    <span class="dim">// All three failed</span>
    <span class="blue">return nil</span>, <span class="blue">true</span>, lastErr
}
</pre>

The key function is `isDefinitelyDead`. It checks whether the error is a confirmed NXDOMAIN -- not a timeout, not a temporary failure, but a definitive "this domain does not exist in DNS." Go's `net.DNSError` has an `IsNotFound` field for exactly this purpose.

## At scale: batch resolution with fastdns

For single domain lookups, Go's standard `net.Resolver` is fine. For resolving 75,000 domains in parallel, it's too slow. The OS resolver (mDNSResponder on macOS, systemd-resolved on Linux) can't handle thousands of concurrent lookups. It serializes them internally and becomes a bottleneck.

For batch resolution, I bypass the system resolver entirely and use direct UDP to Cloudflare and Google via the `phuslu/fastdns` library. 256 connection-pooled UDP sockets per server, 20,000 concurrent worker goroutines. Each worker pulls a domain, tries Cloudflare, tries Google if Cloudflare fails, falls back to the stdlib resolver as a last resort.

<pre class="showcase-visual">
<span class="dim">// Batch DNS: 20K workers, direct UDP to CF + Google</span>

<span class="blue">func</span> <span class="green">makeFastDNSClients</span>(timeout <span class="blue">time.Duration</span>) []<span class="blue">*fastdns.Client</span> {
    servers := []<span class="blue">string</span>{<span class="amber">"1.1.1.1:53"</span>, <span class="amber">"8.8.8.8:53"</span>}
    clients := <span class="green">make</span>([]<span class="blue">*fastdns.Client</span>, <span class="green">len</span>(servers))
    <span class="blue">for</span> i, addr := <span class="blue">range</span> servers {
        udpAddr, _ := net.<span class="green">ResolveUDPAddr</span>(<span class="amber">"udp"</span>, addr)
        clients[i] = &amp;<span class="blue">fastdns.Client</span>{
            Addr:    addr,
            Timeout: timeout,
            Dialer: &amp;<span class="blue">fastdns.UDPDialer</span>{
                Addr:     udpAddr,
                MaxConns: <span class="hl">256</span>,
            },
        }
    }
    <span class="blue">return</span> clients
}
</pre>

At 20K workers and 256 connections per server, this achieves 1,500+ queries per second with 97% accuracy. The 3% gap is mostly timeout noise -- domains where UDP packets were lost or the response came back after the deadline. Those domains get cached as timeouts and skipped for the current run, not falsely killed.

## The MergeHTTPDead disaster

Early in development, I had what seemed like a clever idea. After the HTTP fetch phase, I'd look at all the domains where every request failed -- timeouts, 503s, connection refused -- and merge them back into the DNS cache as "dead." The reasoning: if we can't reach the domain via HTTP, why bother resolving it again on the next run?

This was catastrophic.

A 503 Service Unavailable during a maintenance window would mark the domain as DNS-dead. A timeout because the server was temporarily overloaded would mark it as DNS-dead. A connection refused because we'd hit a rate limit would mark it as DNS-dead. Every transient HTTP failure became a permanent DNS blackhole.

I discovered this when I noticed that each successive crawl run was finding fewer and fewer live domains -- even when the underlying data hadn't changed. Domains that were perfectly healthy were being permanently excluded because they'd had one bad moment during a previous run.

<pre class="showcase-visual">
<span class="dim">// MergeHTTPDead is a no-op. HTTP failures should NOT</span>
<span class="dim">// contaminate the DNS cache. The probe phase handles</span>
<span class="dim">// HTTP reachability separately.</span>
<span class="dim">// Kept as a method stub so callers don't break.</span>
<span class="blue">func</span> (d *<span class="hl">DNSResolver</span>) <span class="green">MergeHTTPDead</span>(httpDead <span class="blue">map[string]bool</span>) <span class="blue">int</span> {
    <span class="blue">return</span> <span class="hl">0</span>
}
</pre>

The function still exists in the codebase. It's a no-op that returns zero. I kept it instead of deleting it because the method signature documents the mistake. Anyone who finds it and reads the comment will understand why HTTP evidence does not belong in DNS storage.

<div class="note">
  <strong>The principle:</strong> DNS cache contains DNS evidence. HTTP cache contains HTTP evidence. Never cross-contaminate. A domain that returns 503 at the HTTP layer is not DNS-dead -- it resolved to an IP, the IP accepted a connection, and the server returned a response. That's the opposite of dead. It's a live server having a bad day.
</div>

## What counts as dead?

After the MergeHTTPDead disaster, I got much more careful about what "dead" means at each layer of the stack. The recrawler has three stages, each with its own definition of dead.

<table>
  <thead>
    <tr>
      <th>Stage</th>
      <th>Marked Dead When</th>
      <th>NOT Dead When</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>DNS Resolution</strong></td>
      <td>NXDOMAIN on all 3 resolvers</td>
      <td>Timeout (might be slow, not dead)</td>
    </tr>
    <tr>
      <td><strong>Probe (TCP+TLS)</strong></td>
      <td>Connection refused, connection reset, DNS error</td>
      <td>Timeout (server might be slow but alive)</td>
    </tr>
    <tr>
      <td><strong>HTTP Fetch</strong></td>
      <td>Connection refused, DNS error (immediate kill). N consecutive timeouts with zero successes (threshold kill)</td>
      <td>503, 429, timeout with previous success</td>
    </tr>
  </tbody>
</table>

The probe stage deserves special attention. It sits between DNS and HTTP -- a lightweight TCP connect plus TLS handshake to check if the server is reachable at all. The classification is deliberately conservative: if the probe times out, the domain is classified as **alive**. The reasoning is the same as DNS: a timeout could mean slow, not dead. Better to waste a few seconds trying the HTTP fetch and failing fast than to permanently skip a domain that might have content.

Only definitive failure signals -- connection refused (port not open), connection reset (server actively rejected), DNS error during connect (can't even find the IP) -- cause a domain to be marked dead at probe time.

## The DNS cache: what goes in and what doesn't

After resolution, the DNS cache has three categories. Each one is persisted to DuckDB between runs.

<pre class="showcase-visual">
<span class="dim">~/data/common-crawl/CC-MAIN-2026-04/dns.duckdb</span>

<span class="green">resolved</span>   <span class="hl">42,891</span> domains   <span class="dim">domain -> []IP addresses</span>
<span class="amber">dead</span>       <span class="hl">31,204</span> domains   <span class="dim">NXDOMAIN, confirmed by multiple servers</span>
<span class="blue">timeout</span>       <span class="hl">876</span> domains   <span class="dim">all resolvers timed out, saved for retry</span>

<span class="dim">What goes IN the DNS cache:</span>
  <span class="green">+</span> Resolved IPs (any resolver succeeded)
  <span class="green">+</span> NXDOMAIN (all resolvers confirmed dead)
  <span class="green">+</span> DNS timeout (all resolvers timed out)

<span class="dim">What does NOT go in the DNS cache:</span>
  <span class="amber">x</span> HTTP 503 (server maintenance)
  <span class="amber">x</span> HTTP 429 (rate limited)
  <span class="amber">x</span> HTTP timeout (server slow)
  <span class="amber">x</span> Connection refused at HTTP stage
  <span class="amber">x</span> TLS handshake failure
</pre>

On the next run, the cache is loaded first. Already-resolved domains skip DNS entirely -- their cached IPs are used directly in the HTTP transport's dial function. Dead domains are pre-populated into the dead domain set and skipped in all phases. Timeout domains are treated as dead for the current run but will be re-resolved if the cache is cleared.

One filter that took me a while to get right: when loading the cache, entries marked as `http_dead` (from the old MergeHTTPDead days) are silently skipped. They get re-resolved instead of being trusted. This retroactive cleanup ensures that domains poisoned by the old logic eventually get a fair trial.

<pre class="showcase-visual">
<span class="dim">// Loading DNS cache -- skip http_dead entries</span>
<span class="blue">if</span> dead {
    <span class="blue">if</span> errMsg == <span class="amber">"http_dead"</span> {
        <span class="blue">continue</span>  <span class="dim">// Re-resolve; HTTP failure != DNS dead</span>
    }
    s.dead[domain] = errMsg
}
</pre>

## This is just the internet

I spent a day convinced file 299 had exposed a bug. It hadn't. It exposed reality.

The internet is not uniform. A domain that's alive in Beijing might be dead in New York. A DNS resolver that's authoritative for one TLD might return garbage for another. A server that was healthy when Common Crawl archived it six months ago might be a parking page today.

97% dead domains wasn't a bug in my recrawler. It was a lesson in data selection. Common Crawl's parquet files are TLD-partitioned, and country-code TLDs have dramatically different reachability depending on where you're crawling from. Once I understood that, the numbers made perfect sense.

The real engineering lesson is about layered defense. No single mechanism handles all the failure modes: multi-server DNS catches false NXDOMAINs, conservative probe classification catches false dead-on-timeout, cache isolation prevents cross-layer contamination, and the no-op MergeHTTPDead is a scar from learning the hard way that HTTP evidence and DNS evidence have to stay in separate boxes.

Pick your parquet files carefully. Trust no single resolver. And never, ever let a 503 kill a domain permanently.
