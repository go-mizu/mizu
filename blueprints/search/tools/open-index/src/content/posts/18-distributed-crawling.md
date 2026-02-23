---
slug: distributed-crawling
title: "Crawling from Ten Machines at Once"
date: 2026-03-06
summary: "From one machine to ten. How to distribute 100K workers across a fleet without fetching the same URL twice."
tags: [distributed, engineering]
---

The single-machine recrawler tops out around 100K concurrent connections. It works -- 2.5M URLs in 65 seconds, per-domain semaphores keeping every server happy, 64 transport shards preventing mutex contention, 16-shard DuckDB absorbing the writes. But something always bottlenecks. Network bandwidth saturates. File descriptors run low. CPU spends too much time on TLS handshakes. To crawl more of the web, we need more machines.

The naive approach -- "just run ten copies" -- creates an immediate problem. Ten machines, each with 8 max connections per domain, means 80 simultaneous connections hitting example.com. We're back to the flooding problem that took success rates from 57.5% to 0.8%. Except now it's distributed across a fleet, which makes it harder to diagnose and impossible to fix with a single semaphore.

## Why domain-based partitioning?

Each domain gets assigned to exactly one machine. Hash the domain name, take the modulo by node count. If `hash("example.com") % 10 = 3`, node 3 handles every URL on example.com. No other node touches it.

This preserves the single-machine invariant: one semaphore, one machine, one connection budget per domain. The same 8-connection limit that works locally works across ten nodes -- because from any given server's perspective, traffic still comes from one source.

<pre class="showcase-visual">
<span class="dim">// Domain → node assignment</span>
<span class="blue">func</span> <span class="green">assignNode</span>(domain <span class="blue">string</span>, nodeCount <span class="blue">int</span>) <span class="blue">int</span> {
    h := fnv.New32a()
    h.Write([]<span class="blue">byte</span>(domain))
    <span class="blue">return int</span>(h.Sum32()) % nodeCount
}

<span class="dim">// Partition seed URLs into per-node batches</span>
batches := <span class="green">make</span>([][]SeedURL, nodeCount)
<span class="blue">for</span> _, url := <span class="blue">range</span> seeds {
    node := <span class="green">assignNode</span>(url.Domain, nodeCount)
    batches[node] = <span class="green">append</span>(batches[node], url)
}
<span class="dim">// Ship each batch to its assigned node</span>
</pre>

Simple. Each node gets its URL batch upfront and works independently. No coordination during the crawl.

## What happens when a node dies?

Modulo hashing has a brittle failure mode. If node 7 crashes and the fleet shrinks from 10 to 9, every domain assignment changes -- `hash % 10` becomes `hash % 9`. Domains shuffle across the entire fleet. Nodes that were halfway through their batches suddenly get reassigned domains they've never seen, while their in-progress domains move elsewhere.

Consistent hashing fixes this. Instead of modulo, domains and nodes are placed on a hash ring. Each domain maps to the nearest node clockwise on the ring. When a node leaves, only its domains move -- roughly 1/N of the total -- and they shift to the next node on the ring. Everything else stays put.

<pre class="showcase-visual">
<span class="dim">           Node 2</span>
<span class="dim">          ╱      ╲</span>
<span class="green">    ● ● ●</span>          <span class="blue">● ●</span>
<span class="dim">   ╱</span>                    <span class="dim">╲</span>
<span class="green"> Node 1</span>               <span class="blue">Node 3</span>
<span class="dim">   ╲</span>                    <span class="dim">╱</span>
<span class="amber">    ● ● ●</span>          <span class="hl">● ● ●</span>
<span class="dim">          ╲      ╱</span>
<span class="dim">           Node 4</span>

<span class="dim">● = domains assigned to each node</span>
<span class="dim">If Node 3 dies, only its <span class="blue">● ●</span> domains</span>
<span class="dim">move to Node 4. Nodes 1 and 2 are unaffected.</span>
</pre>

With virtual nodes (multiple hash positions per physical node), the load distribution evens out further. We use 150 virtual nodes per physical machine -- enough to keep domain counts within 15% of the mean across all nodes.

<table>
  <thead>
    <tr>
      <th>Strategy</th>
      <th>Domains moved on node failure</th>
      <th>Complexity</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Modulo hashing</strong></td>
      <td>~100% (all reshuffled)</td>
      <td>Trivial</td>
    </tr>
    <tr>
      <td><strong>Consistent hashing</strong></td>
      <td>~1/N (only dead node's domains)</td>
      <td>Moderate</td>
    </tr>
    <tr>
      <td><strong>Consistent + virtual nodes</strong></td>
      <td>~1/N, evenly distributed</td>
      <td>Moderate</td>
    </tr>
  </tbody>
</table>

## URL dedup without cross-node chatter

On one machine, a simple `sync.Map` tracks fetched URLs. Across ten machines, that map needs to be distributed. Three options:

1. **Shared Redis set** -- every node checks before fetching. Simple, but every URL lookup adds a network round-trip. At 200K URLs/second across the fleet, that's 200K Redis queries per second just for dedup.
2. **Bloom filter per node** -- fast, probabilistic, but false positives mean skipped URLs. Acceptable for some workloads, not for ours.
3. **Domain-partitioned dedup** -- if node 3 owns example.com, it tracks all example.com URLs locally. No cross-node communication. Zero overhead.

We chose option 3. It falls out of domain-based partitioning for free. Since each domain lives on exactly one node, dedup is purely local. The same `sync.Map` that works on one machine works identically in the distributed version. No Redis. No bloom filters. No network overhead.

<div class="note">
  <strong>The pattern:</strong> Domain-based partitioning solves three problems at once -- connection limiting, URL dedup, and result locality. Each node is a self-contained single-machine recrawler for its assigned domains. The distribution layer just decides which domains go where.
</div>

## Sharing the DNS cache

The DNS cache -- resolved IPs, dead domains, timeouts -- should be shared. A domain resolved by node 1 shouldn't trigger another DNS lookup when node 3 needs it for routing decisions. But DuckDB doesn't handle concurrent writes from multiple machines, and a centralized DNS service adds a dependency we'd rather avoid.

Gossip protocol fits here. Each node maintains its local DNS cache and periodically exchanges state with peers. Eventual consistency is fine for DNS data -- a few redundant lookups during convergence won't hurt throughput. The cache converges within seconds, and after that, every node has the full picture.

<table>
  <thead>
    <tr>
      <th>DNS sharing strategy</th>
      <th>Consistency</th>
      <th>Overhead</th>
      <th>Verdict</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Shared DuckDB file</td>
      <td>Strong</td>
      <td>Write contention</td>
      <td style="color:#fbbf24">Too slow</td>
    </tr>
    <tr>
      <td>Redis</td>
      <td>Strong</td>
      <td>Network per lookup</td>
      <td style="color:#4ade80">Works, adds dependency</td>
    </tr>
    <tr>
      <td>Gossip protocol</td>
      <td>Eventual</td>
      <td>Minimal</td>
      <td style="color:#4ade80">Best fit</td>
    </tr>
  </tbody>
</table>

## Merging 160 shard files

Each node produces 16 DuckDB shards. Ten nodes produce 160 files. After the crawl, these need to merge into a queryable result set. DuckDB handles this natively -- no custom merge logic required.

<pre class="showcase-visual">
<span class="dim">-- Merge all shards from all nodes into a single view</span>
<span class="blue">CREATE VIEW</span> results <span class="blue">AS</span>
<span class="blue">SELECT</span> * <span class="blue">FROM</span> <span class="green">read_parquet</span>(<span class="amber">'nodes/*/shard_*.parquet'</span>);

<span class="dim">-- Or export each node's shards to parquet first</span>
<span class="dim">-- node_01: 16 shards → 1 parquet</span>
<span class="dim">-- node_02: 16 shards → 1 parquet</span>
<span class="dim">-- ...</span>
<span class="dim">-- Then query across all 10 parquet files with glob</span>

<span class="blue">SELECT</span> domain, <span class="green">count</span>(*) <span class="blue">AS</span> pages,
       <span class="green">avg</span>(status_code) <span class="blue">AS</span> avg_status
<span class="blue">FROM</span> results
<span class="blue">GROUP BY</span> domain
<span class="blue">ORDER BY</span> pages <span class="blue">DESC</span>
<span class="blue">LIMIT</span> <span class="hl">50</span>;
</pre>

Export each node's DuckDB shards to Parquet (columnar, compressed, portable), then query the merged dataset with a glob pattern. DuckDB reads Parquet files in parallel across cores. The merge step adds minutes, not hours.

## The hard parts nobody's solved yet

Domain-based partitioning handles the steady state cleanly. The edges are where things get complicated.

**Exactly-once fetch.** The coordinator assigns a URL batch to node 3. Node 3 dies mid-fetch. Some URLs were fetched, some weren't, and the coordinator doesn't know which. Reassigning the entire batch to node 4 means some URLs get fetched twice. The pragmatic answer: tolerate duplicates and dedup in the merge step. Exactly-once in a distributed system requires consensus protocols, and that's more complexity than duplicating a few HTTP requests.

**Network partitions.** Two nodes both think they own a domain because they can't see each other but can still reach the target servers. From the server's perspective, the per-domain connection limit just doubled. Partition-tolerant assignment requires a coordination service (etcd, Consul) that adds operational overhead.

**Stragglers.** If node 3 gets assigned domains that are disproportionately slow (Finnish domains, anyone?), the entire crawl waits for it. Work stealing -- letting fast nodes take domains from slow ones -- helps, but conflicts with the "one domain, one node" invariant.

<div class="note">
  <strong>Each of these has known solutions.</strong> Exactly-once has idempotent writes. Partitions have lease-based ownership. Stragglers have work stealing with domain handoff. The solutions exist. The implementation cost isn't zero.
</div>

## Why not a central job queue?

Systems like Scrapy Cluster and Celery-based crawlers use a central job queue -- Redis, RabbitMQ, Kafka. Workers pull URLs one at a time, fetch, report back. It works, but the queue becomes a bottleneck and a single point of failure.

Domain-based partitioning is push-based. The coordinator partitions URLs upfront and ships batches to nodes. During the crawl, there's no central component to fail. Each node is autonomous. If the coordinator dies after distribution, the crawl continues. If a worker node dies, only its domains are affected.

The tradeoff: a job queue gives you fine-grained load balancing (any worker can take any URL). Domain partitioning gives you locality (all URLs for a domain are colocated, enabling local dedup and local semaphores). For web crawling, locality wins.

## Where we are

<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Single-machine recrawler</td>
      <td><span style="color:#4ade80">Battle-tested</span></td>
    </tr>
    <tr>
      <td>Distribution design</td>
      <td><span style="color:#4ade80">Complete</span></td>
    </tr>
    <tr>
      <td>Consistent hashing</td>
      <td><span style="color:#fbbf24">Prototyped</span></td>
    </tr>
    <tr>
      <td>Gossip DNS cache</td>
      <td><span style="color:#fbbf24">Evaluating options</span></td>
    </tr>
    <tr>
      <td>Multi-node deployment</td>
      <td><span style="color:#888">After search layer ships</span></td>
    </tr>
  </tbody>
</table>

The single-machine recrawler already handles the workloads we've thrown at it. Distribution isn't about replacing it -- it's about scaling past the point where one machine's NIC becomes the ceiling. The architecture is designed so that each node runs the exact same recrawler binary with the exact same flags. The only difference is which domains it receives. Ten machines, ten independent crawlers, one coordinated assignment. No shared state during the crawl. Merge afterward.

That's the goal, anyway. We'll see what breaks when we actually try it.
