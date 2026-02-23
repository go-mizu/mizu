---
slug: distributed-consistency
title: "When Shards Disagree"
date: 2026-03-07
summary: "What happens when shards disagree, agents go down, and segments get written at the same time."
tags: [distributed, engineering]
---

Pick two: Consistency, Availability, Partition tolerance. The CAP theorem says you can't have all three in a distributed system. A bank picks CP -- if the network splits, stop accepting writes rather than risk showing the wrong balance. A search engine picks AP -- a slightly stale result is better than no result at all.

This choice shapes every design decision in OpenIndex.

## What does "consistent" mean when you have 16 databases?

DuckDB shards use a single-writer, multi-reader model. One goroutine writes to each shard file. Unlimited readers can query concurrently. Within a single shard, writes are strongly consistent -- a query always sees the latest committed data.

But "within a single shard" is doing heavy lifting. The index has 16 shards. A cross-shard query might hit shard 0 (freshly flushed) and shard 15 (still buffering). For a brief window, different shards have different views. That's eventual consistency across the index.

In practice, it rarely matters. The shard key is the domain. All URLs from `example.com` go to one shard. Within-domain queries are always consistent. Cross-domain aggregations -- "how many total pages?" -- might be off by a few hundred rows during a flush cycle. Nobody notices.

<table>
  <thead>
    <tr>
      <th>Query Type</th>
      <th>Shards Touched</th>
      <th>Consistency</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>site:example.com</code></td>
      <td>1</td>
      <td><span style="color:#4ade80"><strong>Strong</strong></span> (single writer)</td>
    </tr>
    <tr>
      <td><code>status:200 domain:*.edu</code></td>
      <td>16</td>
      <td><span style="color:#fbbf24"><strong>Eventual</strong></span> (cross-shard)</td>
    </tr>
    <tr>
      <td><code>COUNT(*) total pages</code></td>
      <td>16</td>
      <td><span style="color:#fbbf24"><strong>Eventual</strong></span> (off by flush lag)</td>
    </tr>
  </tbody>
</table>

## What happens when a Vald agent dies?

Vald replicates vectors across multiple agents. Three replicas per vector, spread across different pods. When an agent dies, queries route to survivors. Availability stays up.

Writes are trickier. A new vector goes to the primary agent, then replicates asynchronously. During that lag, different agents have different data. Query agent 0 and you see the new page. Query agent 1 and you don't. The vector index disagrees with itself for a few seconds.

For search, this is fine. The web changes faster than replication lag -- by the time a vector propagates, dozens of new pages have been crawled.

<div class="note">
  <strong>Replication lag vs. crawl lag.</strong> Vald's async replication adds 1-5 seconds of delay. The crawl pipeline adds hours or days of delay between a page changing on the web and OpenIndex re-fetching it. Worrying about seconds of replication lag on top of days of crawl lag is optimizing the wrong thing.
</div>

## Immutable segments give you free snapshot isolation

Tantivy's consistency model is the most elegant of the three. Segments are immutable once written. New documents go into new segments. Readers see a consistent snapshot of all committed segments. Writers append without blocking readers.

This is MVCC -- Multi-Version Concurrency Control -- without the complexity. No rollback log, no row-level locking, no isolation levels. The immutability *is* the isolation mechanism.

<pre><code>  <span style="color:#888">Time ──────────────────────────────────►</span>

  <span style="color:#60a5fa">Segment A</span>  ████████████████████  <span style="color:#888">(committed, immutable)</span>
  <span style="color:#60a5fa">Segment B</span>       ██████████████████████  <span style="color:#888">(committed, immutable)</span>
  <span style="color:#fbbf24">Segment C</span>                    ██████████  <span style="color:#888">(writing...)</span>

  <span style="color:#4ade80">Reader 1</span>  opens here ──┐
                         │  sees: A, B
                         │  doesn't see: C (uncommitted)

  <span style="color:#4ade80">Reader 2</span>         opens here ──┐
                              │  sees: A, B, C (now committed)

  <span style="color:#e0e0e0">Neither reader blocks. Neither sees partial writes.</span></code></pre>

In a distributed setup, different nodes create different segments. A query fans out to all nodes, each searches its local segments, and results merge by BM25 score. No node returns half-written documents.

Background merging compacts old segments into larger ones without affecting readers. An active reader keeps references to the segments it opened with. Old segments get deleted only after all readers release them. Garbage collection for index files.

## The consistency spectrum

<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Consistency Model</th>
      <th>During Partition</th>
      <th>Search Impact</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>DuckDB</strong></td>
      <td>Per-shard strong, cross-shard eventual</td>
      <td>Affected shards unavailable</td>
      <td>Domain queries unaffected</td>
    </tr>
    <tr>
      <td><strong>Vald</strong></td>
      <td>Eventual (async replication)</td>
      <td>Replicas serve reads, writes queue</td>
      <td>New vectors delayed 1-5s</td>
    </tr>
    <tr>
      <td><strong>Tantivy</strong></td>
      <td>Snapshot isolation (immutable segments)</td>
      <td>Nodes serve committed segments</td>
      <td>Latest segment may be missing</td>
    </tr>
    <tr>
      <td><strong>DNS cache</strong></td>
      <td>Best-effort (gossip)</td>
      <td>Stale IPs served until TTL</td>
      <td>Crawl may hit wrong server briefly</td>
    </tr>
  </tbody>
</table>

Every component sits somewhere different on the spectrum. None need strong consistency across the entire cluster. That's AP paying off.

## Why staleness is OK

A bank account showing $50 when the real balance is $47 -- that's a lawsuit. A search index returning 1,000,003 results instead of 1,000,007 -- nobody notices and nobody cares.

Search results are approximate by nature. BM25 scoring is a heuristic. Vector similarity is approximate nearest neighbor, not exact. The web changes faster than any crawler can keep up with. By the time results render in a browser, the underlying pages may have already changed.

Adding seconds of replication lag on top of that inherent approximation changes nothing meaningful. A bit more staleness in a best-effort view of a constantly-changing web is rounding error.

## When staleness isn't OK

There's a line between "stale" and "broken." These cross it:

- **Index corruption**: Two writers to the same DuckDB file produces data loss. The single-writer-per-shard model exists to prevent this.
- **Duplicate entries**: Same URL indexed by two different nodes. Now `site:example.com` returns the same page twice.
- **Missing entries**: URL assigned to a node that crashed before flushing. The URL disappears from the index entirely.

These aren't consistency trade-offs. They're bugs. The distributed design needs to prevent them, not tolerate them.

<div class="note">
  <strong>Staleness vs. correctness.</strong> A stale result is an old result. A corrupt result is a wrong result. An eventual consistency model accepts the first and must prevent the second. Every component in OpenIndex has write-path guarantees that prevent corruption, even when read-path consistency is relaxed.
</div>

## The write path matters more

Read consistency is negotiable in a search system. Write consistency isn't.

If two nodes both index the same URL, you get duplicates. If a node crashes mid-write, you get partial data. These failure modes compound -- duplicates accumulate, partial writes corrupt aggregations, lost appends erode coverage.

The solution: **domain-partitioned writes**. Same domain, same node, same shard. One writer per shard, always. Write-ahead logging handles crash recovery. Tantivy's immutable segments mean a failed write just produces an incomplete segment that gets discarded on restart. Vald's agent assignment gives each vector exactly one primary owner.

Idempotent inserts handle the rest. If a URL arrives twice (crawl pipeline retry), the second insert is a no-op. No duplicates, no corruption, no special recovery logic.

## Where this stands

DuckDB sharding is battle-tested on a single machine. Hundreds of millions of URLs across 16 shards. Cross-shard eventual consistency is invisible in practice. Vald replication architecture is designed -- agent topology, async replication, failover routing. Kubernetes deployment is next. Tantivy distributed segments are in design. The open question is cross-node segment merging without centralized coordination.

The CAP decisions are made. We picked AP. Every component optimizes for availability and partition tolerance, accepts bounded staleness on reads, and enforces strict correctness on writes. For a search engine indexing a web that changes billions of times a day, that's the right trade-off.
