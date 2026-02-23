---
slug: graph-queries-sql
title: "Graph Queries in SQL — Recursive CTEs Instead of SPARQL"
date: 2026-03-13
summary: "You don't need a graph database to query a graph. DuckDB's recursive CTEs do path traversal, and they're already installed."
tags: [ontology, search, data]
---

Graph databases exist for a reason. Cypher is elegant. SPARQL is expressive. Both are purpose-built for walking edges and matching patterns across connected data. If you're starting from scratch and graph queries are your primary workload, you should probably use one.

We aren't starting from scratch. We already run DuckDB. It handles billions of rows. The recrawler writes to it, the CC extractor writes to it, analytics runs on it, the triple store lives in it. Adding Neo4j or a SPARQL endpoint means another database to deploy, another query language to learn, another service to keep alive at 3 AM.

Recursive CTEs aren't pretty. The queries are verbose, sometimes painful to read, and nobody's putting them on a conference slide for aesthetic reasons. They also work, require zero additional infrastructure, and don't add another operational dependency. Ugly + working + zero ops beats elegant + another database to maintain.

## The triples table with two indexes

The triple store from [post 11](/blog/knowledge-graph) stores every fact as a row:

<pre><code><span style="color:#60a5fa">CREATE TABLE</span> triples (
  subject    <span style="color:#fbbf24">VARCHAR</span>,
  predicate  <span style="color:#fbbf24">VARCHAR</span>,
  object     <span style="color:#fbbf24">VARCHAR</span>,
  source_url <span style="color:#fbbf24">VARCHAR</span>,
  confidence <span style="color:#fbbf24">FLOAT</span>
);</code></pre>

One table. Five columns. Every entity relationship in the knowledge graph lives here. But without indexes, even simple lookups degrade to full table scans. Two indexes make graph traversal viable:

<pre><code><span style="color:#888">-- SPO: "What does entity X do?"</span>
<span style="color:#888">-- Forward traversal: given a subject, find its predicates and objects</span>
<span style="color:#60a5fa">CREATE INDEX</span> idx_spo <span style="color:#60a5fa">ON</span> triples (subject, predicate, object);

<span style="color:#888">-- OPS: "Who connects to entity X?"</span>
<span style="color:#888">-- Backward traversal: given an object, find what points to it</span>
<span style="color:#60a5fa">CREATE INDEX</span> idx_ops <span style="color:#60a5fa">ON</span> triples (object, predicate, subject);</code></pre>

Why both? Forward traversal uses SPO. "What organizations did Sam Altman found?" starts with `subject = 'sam_altman'` and walks outward. The SPO index handles this instantly.

Backward traversal uses OPS. "Find all pages that mention Apple" starts with `object = 'apple'` and walks inward. Without the OPS index, that query scans every row looking for `object = 'apple'` in the third column -- the index on `(subject, predicate, object)` doesn't help because the query doesn't constrain `subject`. The OPS index flips the column order so backward lookups hit an index prefix.

<div class="note">
  <strong>Two indexes, not three.</strong> Some triple stores add a third index (PSO) for predicate-first queries like "find all <code>oi:locatedIn</code> relationships." We skip it. Predicate-first queries are rare in practice -- you almost always know the entity and want its connections, not the other way around. If predicate-only queries become common, adding the third index is a one-liner.
</div>

## Basic patterns -- SPARQL vs SQL side-by-side

The translation from SPARQL to SQL is mechanical. Pattern matching becomes JOINs. Triple patterns become WHERE clauses. The SQL is longer but structurally identical.

### Example 1: Type query

*Find all organizations.*

<pre><code><span style="color:#888">-- SPARQL</span>
<span style="color:#60a5fa">SELECT</span> <span style="color:#e0e0e0">?org</span> <span style="color:#60a5fa">WHERE</span> {
  <span style="color:#e0e0e0">?org</span> <span style="color:#4ade80">rdf:type</span> <span style="color:#4ade80">oi:Organization</span>
}</code></pre>

<pre><code><span style="color:#888">-- DuckDB SQL</span>
<span style="color:#60a5fa">SELECT</span> subject <span style="color:#60a5fa">AS</span> org
<span style="color:#60a5fa">FROM</span> triples
<span style="color:#60a5fa">WHERE</span> predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> object = <span style="color:#4ade80">'oi:Organization'</span>;</code></pre>

One SPARQL triple pattern, one SQL WHERE clause. The SPO index covers this -- it's a prefix match on `(subject, predicate)` with a filter on `object`.

### Example 2: Relationship query

*Find all people affiliated with MIT.*

<pre><code><span style="color:#888">-- SPARQL</span>
<span style="color:#60a5fa">SELECT</span> <span style="color:#e0e0e0">?person</span> <span style="color:#60a5fa">WHERE</span> {
  <span style="color:#e0e0e0">?person</span> <span style="color:#4ade80">oi:affiliatedWith</span> <span style="color:#4ade80">&lt;entity/mit&gt;</span>
}</code></pre>

<pre><code><span style="color:#888">-- DuckDB SQL</span>
<span style="color:#60a5fa">SELECT</span> subject <span style="color:#60a5fa">AS</span> person
<span style="color:#60a5fa">FROM</span> triples
<span style="color:#60a5fa">WHERE</span> predicate = <span style="color:#4ade80">'oi:affiliatedWith'</span>
  <span style="color:#60a5fa">AND</span> object = <span style="color:#4ade80">'entity/mit'</span>;</code></pre>

Same structure. OPS index this time -- we're searching by object.

### Example 3: Multi-condition join

*Find products created by organizations located in San Francisco.*

<pre><code><span style="color:#888">-- SPARQL</span>
<span style="color:#60a5fa">SELECT</span> <span style="color:#e0e0e0">?product</span> <span style="color:#e0e0e0">?org</span> <span style="color:#60a5fa">WHERE</span> {
  <span style="color:#e0e0e0">?product</span> <span style="color:#4ade80">oi:createdBy</span>  <span style="color:#e0e0e0">?org</span> .
  <span style="color:#e0e0e0">?org</span>     <span style="color:#4ade80">rdf:type</span>     <span style="color:#4ade80">oi:Organization</span> .
  <span style="color:#e0e0e0">?org</span>     <span style="color:#4ade80">oi:locatedIn</span> <span style="color:#4ade80">&lt;entity/san_francisco&gt;</span>
}</code></pre>

<pre><code><span style="color:#888">-- DuckDB SQL</span>
<span style="color:#60a5fa">SELECT</span> t1.subject <span style="color:#60a5fa">AS</span> product,
       t1.object  <span style="color:#60a5fa">AS</span> org
<span style="color:#60a5fa">FROM</span> triples t1
<span style="color:#60a5fa">JOIN</span> triples t2
  <span style="color:#60a5fa">ON</span> t2.subject = t1.object
  <span style="color:#60a5fa">AND</span> t2.predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> t2.object = <span style="color:#4ade80">'oi:Organization'</span>
<span style="color:#60a5fa">JOIN</span> triples t3
  <span style="color:#60a5fa">ON</span> t3.subject = t1.object
  <span style="color:#60a5fa">AND</span> t3.predicate = <span style="color:#4ade80">'oi:locatedIn'</span>
  <span style="color:#60a5fa">AND</span> t3.object = <span style="color:#4ade80">'entity/san_francisco'</span>
<span style="color:#60a5fa">WHERE</span> t1.predicate = <span style="color:#4ade80">'oi:createdBy'</span>;</code></pre>

Three SPARQL triple patterns become three table references joined together. Each SPARQL variable (`?product`, `?org`) becomes a join condition. The SQL is longer. It's also doing the exact same thing -- matching a graph pattern by finding rows that satisfy all three constraints simultaneously.

## Multi-hop traversal -- recursive CTE walkthrough

This is the main event. Given an entity, find everything within 2 hops.

<pre><code><span style="color:#888">-- Find all entities within 2 hops of 'openai'</span>
<span style="color:#60a5fa">WITH RECURSIVE</span> neighbors <span style="color:#60a5fa">AS</span> (
  <span style="color:#888">-- Base case: direct neighbors (1 hop)</span>
  <span style="color:#60a5fa">SELECT</span>
    object        <span style="color:#60a5fa">AS</span> entity,
    predicate     <span style="color:#60a5fa">AS</span> via,
    <span style="color:#fbbf24">1</span>             <span style="color:#60a5fa">AS</span> depth,
    subject       <span style="color:#60a5fa">AS</span> reached_from
  <span style="color:#60a5fa">FROM</span> triples
  <span style="color:#60a5fa">WHERE</span> subject = <span style="color:#4ade80">'openai'</span>

  <span style="color:#60a5fa">UNION ALL</span>

  <span style="color:#888">-- Recursive case: neighbors of neighbors</span>
  <span style="color:#60a5fa">SELECT</span>
    t.object      <span style="color:#60a5fa">AS</span> entity,
    t.predicate   <span style="color:#60a5fa">AS</span> via,
    n.depth + <span style="color:#fbbf24">1</span>   <span style="color:#60a5fa">AS</span> depth,
    n.entity      <span style="color:#60a5fa">AS</span> reached_from
  <span style="color:#60a5fa">FROM</span> triples t
  <span style="color:#60a5fa">JOIN</span> neighbors n
    <span style="color:#60a5fa">ON</span> t.subject = n.entity
  <span style="color:#60a5fa">WHERE</span> n.depth < <span style="color:#fbbf24">2</span>           <span style="color:#888">-- depth limit</span>
    <span style="color:#60a5fa">AND</span> t.object != <span style="color:#4ade80">'openai'</span>  <span style="color:#888">-- prevent cycling back to start</span>
)
<span style="color:#60a5fa">SELECT DISTINCT</span> entity, via, depth, reached_from
<span style="color:#60a5fa">FROM</span> neighbors
<span style="color:#60a5fa">ORDER BY</span> depth, entity;</code></pre>

Walk through it:

1. **Base case** selects all direct neighbors of `'openai'` -- every row where `subject = 'openai'`. That gives us `sam_altman`, `san_francisco`, `gpt-4`, etc. Each gets `depth = 1`.

2. **Recursive case** takes those results and finds *their* neighbors. For each entity at depth 1, it joins back to the triples table to find entities at depth 2. The `WHERE n.depth < 2` clause stops the recursion -- without it, the CTE runs until it exhausts the graph or DuckDB's memory.

3. **Cycle prevention** -- `t.object != 'openai'` is a minimal guard against cycling back to the start node. For a production query, you'd want a proper visited set (more on that next).

Example output:

<table>
  <thead>
    <tr>
      <th>entity</th>
      <th>via</th>
      <th>depth</th>
      <th>reached_from</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>sam_altman</td>
      <td>oi:createdBy</td>
      <td>1</td>
      <td>openai</td>
    </tr>
    <tr>
      <td>san_francisco</td>
      <td>oi:locatedIn</td>
      <td>1</td>
      <td>openai</td>
    </tr>
    <tr>
      <td>gpt-4</td>
      <td>oi:createdBy</td>
      <td>1</td>
      <td>openai</td>
    </tr>
    <tr>
      <td>stanford</td>
      <td>oi:affiliatedWith</td>
      <td>2</td>
      <td>sam_altman</td>
    </tr>
    <tr>
      <td>california</td>
      <td>oi:locatedIn</td>
      <td>2</td>
      <td>san_francisco</td>
    </tr>
    <tr>
      <td>chatgpt</td>
      <td>schema:sameAs</td>
      <td>2</td>
      <td>gpt-4</td>
    </tr>
  </tbody>
</table>

From one entity, two hops, six connected nodes. The query pattern is always the same: base case seeds the starting set, recursive case expands outward, depth limit stops the explosion.

## Path queries -- "how are X and Y connected?"

The hardest graph query in SQL: given two entities, find all paths between them. This is where DuckDB's array support earns its keep.

<pre><code><span style="color:#888">-- Find paths from 'sam_altman' to 'san_francisco' (max depth 3)</span>
<span style="color:#60a5fa">WITH RECURSIVE</span> paths <span style="color:#60a5fa">AS</span> (
  <span style="color:#888">-- Base case: start at source entity</span>
  <span style="color:#60a5fa">SELECT</span>
    subject                          <span style="color:#60a5fa">AS</span> current,
    [subject]::VARCHAR[]             <span style="color:#60a5fa">AS</span> visited,
    [subject || <span style="color:#4ade80">' -['</span> || predicate || <span style="color:#4ade80">']-> '</span> || object]::VARCHAR[]
                                     <span style="color:#60a5fa">AS</span> path,
    <span style="color:#fbbf24">1</span>                                <span style="color:#60a5fa">AS</span> depth
  <span style="color:#60a5fa">FROM</span> triples
  <span style="color:#60a5fa">WHERE</span> subject = <span style="color:#4ade80">'sam_altman'</span>

  <span style="color:#60a5fa">UNION ALL</span>

  <span style="color:#888">-- Recursive case: extend paths by one hop</span>
  <span style="color:#60a5fa">SELECT</span>
    t.object                         <span style="color:#60a5fa">AS</span> current,
    list_append(p.visited, t.object) <span style="color:#60a5fa">AS</span> visited,
    list_append(p.path,
      t.subject || <span style="color:#4ade80">' -['</span> || t.predicate || <span style="color:#4ade80">']-> '</span> || t.object
    )                                <span style="color:#60a5fa">AS</span> path,
    p.depth + <span style="color:#fbbf24">1</span>                      <span style="color:#60a5fa">AS</span> depth
  <span style="color:#60a5fa">FROM</span> triples t
  <span style="color:#60a5fa">JOIN</span> paths p
    <span style="color:#60a5fa">ON</span> t.subject = p.current
  <span style="color:#60a5fa">WHERE</span> p.depth < <span style="color:#fbbf24">3</span>                  <span style="color:#888">-- depth limit</span>
    <span style="color:#60a5fa">AND</span> <span style="color:#60a5fa">NOT</span> list_contains(p.visited, t.object)  <span style="color:#888">-- cycle prevention</span>
)
<span style="color:#60a5fa">SELECT</span> path, depth
<span style="color:#60a5fa">FROM</span> paths
<span style="color:#60a5fa">WHERE</span> current = <span style="color:#4ade80">'san_francisco'</span>
<span style="color:#60a5fa">ORDER BY</span> depth;</code></pre>

The cycle detection is critical. Without `list_contains(p.visited, t.object)`, the CTE happily traverses A -> B -> A -> B -> A forever until the depth limit saves you -- or doesn't, if you forgot to set one. The `visited` array accumulates every node on the current path. If the next hop would revisit a node, the row gets filtered out.

DuckDB's `list_append` and `list_contains` make this feasible. In databases without array support, you'd have to encode the visited set as a delimited string and use `LIKE '%,node,%'` for containment checks. It works, but it's the kind of query that makes you question your life choices.

<div class="note note-warn">
  <strong>Performance warning:</strong> Path queries explore exponentially more rows per hop. At depth 1, you might scan 50 rows. At depth 2, 2,500. At depth 3, 125,000. The <code>visited</code> array grows with each path, and <code>list_contains</code> is O(n) per check. For production use, constrain entity types at each hop (see Mitigations below).
</div>

## Aggregation queries

Not every graph question needs recursion. The three most useful aggregates are one-liners.

### Degree centrality -- which entities have the most connections?

<pre><code><span style="color:#60a5fa">SELECT</span> entity, <span style="color:#60a5fa">SUM</span>(connections) <span style="color:#60a5fa">AS</span> degree
<span style="color:#60a5fa">FROM</span> (
  <span style="color:#60a5fa">SELECT</span> subject <span style="color:#60a5fa">AS</span> entity, <span style="color:#60a5fa">COUNT</span>(*) <span style="color:#60a5fa">AS</span> connections <span style="color:#60a5fa">FROM</span> triples <span style="color:#60a5fa">GROUP BY</span> subject
  <span style="color:#60a5fa">UNION ALL</span>
  <span style="color:#60a5fa">SELECT</span> object <span style="color:#60a5fa">AS</span> entity, <span style="color:#60a5fa">COUNT</span>(*) <span style="color:#60a5fa">AS</span> connections <span style="color:#60a5fa">FROM</span> triples <span style="color:#60a5fa">GROUP BY</span> object
)
<span style="color:#60a5fa">GROUP BY</span> entity
<span style="color:#60a5fa">ORDER BY</span> degree <span style="color:#60a5fa">DESC</span>
<span style="color:#60a5fa">LIMIT</span> <span style="color:#fbbf24">10</span>;</code></pre>

Counts both inbound and outbound edges. Entities mentioned everywhere -- "google", "united_states", "english" -- float to the top. Useful for spotting hub entities and potential noise.

### Type distribution -- how many entities of each type?

<pre><code><span style="color:#60a5fa">SELECT</span> object <span style="color:#60a5fa">AS</span> entity_type, <span style="color:#60a5fa">COUNT</span>(*) <span style="color:#60a5fa">AS</span> count
<span style="color:#60a5fa">FROM</span> triples
<span style="color:#60a5fa">WHERE</span> predicate = <span style="color:#4ade80">'rdf:type'</span>
<span style="color:#60a5fa">GROUP BY</span> object
<span style="color:#60a5fa">ORDER BY</span> count <span style="color:#60a5fa">DESC</span>;</code></pre>

### Relationship frequency -- which predicates are most common?

<pre><code><span style="color:#60a5fa">SELECT</span> predicate, <span style="color:#60a5fa">COUNT</span>(*) <span style="color:#60a5fa">AS</span> frequency
<span style="color:#60a5fa">FROM</span> triples
<span style="color:#60a5fa">GROUP BY</span> predicate
<span style="color:#60a5fa">ORDER BY</span> frequency <span style="color:#60a5fa">DESC</span>;</code></pre>

Expect `oi:mentions` to dominate (it's the most common extraction from NER), followed by `rdf:type`, then `oi:locatedIn` and `oi:affiliatedWith`. If `schema:sameAs` is suspiciously high, you probably have a deduplication problem.

## Performance at scale

All numbers assume the SPO and OPS indexes exist. Without them, multiply everything by 100x or more.

<table>
  <thead>
    <tr>
      <th>Query Type</th>
      <th>100K triples</th>
      <th>10M triples</th>
      <th>1B triples</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Type lookup</strong></td>
      <td>&lt;1 ms</td>
      <td>~5 ms</td>
      <td>~50 ms</td>
    </tr>
    <tr>
      <td><strong>1-hop neighbors</strong></td>
      <td>&lt;1 ms</td>
      <td>~10 ms</td>
      <td>~100 ms</td>
    </tr>
    <tr>
      <td><strong>2-hop traversal</strong></td>
      <td>~5 ms</td>
      <td>~200 ms</td>
      <td>~3 s</td>
    </tr>
    <tr>
      <td><strong>Path query (depth 3)</strong></td>
      <td>~20 ms</td>
      <td>~2 s</td>
      <td>~30 s</td>
    </tr>
    <tr>
      <td><strong>Full-graph aggregation</strong></td>
      <td>~10 ms</td>
      <td>~500 ms</td>
      <td>~8 s</td>
    </tr>
  </tbody>
</table>

At 10M triples, everything's interactive. At 1B, point lookups and 1-hop queries are still fast, but multi-hop traversal starts hurting. Path queries at depth 3 across a billion triples can take 30 seconds because the search space explodes exponentially -- each hop multiplies the candidate set by the average degree of the graph.

<div class="note">
  <strong>Honest about the wall.</strong> 1B triples with a depth-3 path query is where SQL graph queries stop being fun. The recursive CTE materializes every intermediate path in memory. DuckDB doesn't have graph-specific optimizations like Neo4j's traversal engine or Dgraph's distributed graph execution. At this scale, the mitigations below aren't optional -- they're survival.
</div>

## Mitigations

Four techniques keep graph queries in SQL from falling off a cliff.

**Depth limits.** Never traverse more than 3 hops without a very good reason. Each hop multiplies the work. 2 hops covers "friends-of-friends" style queries. 3 hops covers most real questions. 4+ hops usually means you're exploring, not querying -- and you should export to a graph library for that.

**Type filtering.** Constrain entity types at each hop. Instead of "find all entities within 2 hops," query "find all Organizations connected to Persons within 2 hops." Adding `AND t2.predicate = 'rdf:type' AND t2.object = 'oi:Person'` at each recursive step prunes branches early.

<pre><code><span style="color:#888">-- Type-constrained 2-hop: only follow Organization → Person edges</span>
<span style="color:#60a5fa">WITH RECURSIVE</span> typed_neighbors <span style="color:#60a5fa">AS</span> (
  <span style="color:#60a5fa">SELECT</span> object <span style="color:#60a5fa">AS</span> entity, <span style="color:#fbbf24">1</span> <span style="color:#60a5fa">AS</span> depth
  <span style="color:#60a5fa">FROM</span> triples
  <span style="color:#60a5fa">WHERE</span> subject = <span style="color:#4ade80">'openai'</span>
    <span style="color:#60a5fa">AND</span> predicate = <span style="color:#4ade80">'oi:affiliatedWith'</span>
  <span style="color:#60a5fa">UNION ALL</span>
  <span style="color:#60a5fa">SELECT</span> t.object, n.depth + <span style="color:#fbbf24">1</span>
  <span style="color:#60a5fa">FROM</span> triples t
  <span style="color:#60a5fa">JOIN</span> typed_neighbors n <span style="color:#60a5fa">ON</span> t.subject = n.entity
  <span style="color:#60a5fa">WHERE</span> n.depth < <span style="color:#fbbf24">2</span>
    <span style="color:#60a5fa">AND</span> t.predicate = <span style="color:#4ade80">'oi:locatedIn'</span>  <span style="color:#888">-- only follow locatedIn edges</span>
)
<span style="color:#60a5fa">SELECT DISTINCT</span> entity, depth <span style="color:#60a5fa">FROM</span> typed_neighbors;</code></pre>

**Confidence thresholds.** `WHERE confidence > 0.8` isn't just about quality -- it's about performance. NER-extracted triples with confidence 0.3 are probably noise. Filtering them out eliminates rows *and* reduces the branching factor at each hop. Fewer edges to follow means exponentially less work in recursive queries.

**Materialized subgraphs.** For frequently queried neighborhoods, pre-compute the result as a materialized view:

<pre><code><span style="color:#888">-- Pre-compute the 2-hop neighborhood for high-degree entities</span>
<span style="color:#60a5fa">CREATE TABLE</span> neighborhood_cache <span style="color:#60a5fa">AS</span>
<span style="color:#60a5fa">WITH RECURSIVE</span> ... <span style="color:#888">-- (the 2-hop CTE from above)</span>
<span style="color:#60a5fa">SELECT</span> * <span style="color:#60a5fa">FROM</span> neighbors;</code></pre>

Rebuild nightly or on crawl completion. Turns a 3-second recursive query into a 50ms indexed lookup.

## OQL GRAPH clause compilation

The OQL `GRAPH` clause from [post 12](/blog/oql-query-language) exists specifically so users don't have to write these queries by hand. Here's how it compiles.

Single-hop:

<pre><code><span style="color:#888">-- OQL</span>
<span style="color:#60a5fa">GRAPH</span> <span style="color:#e0e0e0">Organization</span> <span style="color:#60a5fa">LINKED_TO</span> <span style="color:#e0e0e0">Person</span></code></pre>

<pre><code><span style="color:#888">-- Compiled SQL</span>
<span style="color:#60a5fa">SELECT</span> t1.subject <span style="color:#60a5fa">AS</span> organization,
       t1.object  <span style="color:#60a5fa">AS</span> person
<span style="color:#60a5fa">FROM</span> triples t1
<span style="color:#60a5fa">JOIN</span> triples t_type1
  <span style="color:#60a5fa">ON</span> t_type1.subject = t1.subject
  <span style="color:#60a5fa">AND</span> t_type1.predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> t_type1.object = <span style="color:#4ade80">'oi:Organization'</span>
<span style="color:#60a5fa">JOIN</span> triples t_type2
  <span style="color:#60a5fa">ON</span> t_type2.subject = t1.object
  <span style="color:#60a5fa">AND</span> t_type2.predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> t_type2.object = <span style="color:#4ade80">'oi:Person'</span>
<span style="color:#60a5fa">WHERE</span> t1.predicate = <span style="color:#4ade80">'oi:linkedTo'</span>;</code></pre>

One OQL clause, three SQL JOINs. The compiler adds type checks for both endpoints automatically.

Multi-hop:

<pre><code><span style="color:#888">-- OQL</span>
<span style="color:#60a5fa">GRAPH</span> <span style="color:#e0e0e0">Person</span> <span style="color:#60a5fa">FOUNDED_BY</span> <span style="color:#e0e0e0">Organization</span> <span style="color:#60a5fa">LOCATED_IN</span> <span style="color:#e0e0e0">Place</span></code></pre>

<pre><code><span style="color:#888">-- Compiled SQL</span>
<span style="color:#60a5fa">SELECT</span> t1.subject  <span style="color:#60a5fa">AS</span> person,
       t1.object   <span style="color:#60a5fa">AS</span> organization,
       t2.object   <span style="color:#60a5fa">AS</span> place
<span style="color:#60a5fa">FROM</span> triples t1
<span style="color:#60a5fa">JOIN</span> triples t2
  <span style="color:#60a5fa">ON</span> t2.subject = t1.object
  <span style="color:#60a5fa">AND</span> t2.predicate = <span style="color:#4ade80">'oi:locatedIn'</span>
<span style="color:#60a5fa">JOIN</span> triples t_type1
  <span style="color:#60a5fa">ON</span> t_type1.subject = t1.subject
  <span style="color:#60a5fa">AND</span> t_type1.predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> t_type1.object = <span style="color:#4ade80">'oi:Person'</span>
<span style="color:#60a5fa">JOIN</span> triples t_type2
  <span style="color:#60a5fa">ON</span> t_type2.subject = t1.object
  <span style="color:#60a5fa">AND</span> t_type2.predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> t_type2.object = <span style="color:#4ade80">'oi:Organization'</span>
<span style="color:#60a5fa">JOIN</span> triples t_type3
  <span style="color:#60a5fa">ON</span> t_type3.subject = t2.object
  <span style="color:#60a5fa">AND</span> t_type3.predicate = <span style="color:#4ade80">'rdf:type'</span>
  <span style="color:#60a5fa">AND</span> t_type3.object = <span style="color:#4ade80">'oi:Place'</span>
<span style="color:#60a5fa">WHERE</span> t1.predicate = <span style="color:#4ade80">'oi:createdBy'</span>;</code></pre>

Two edge traversals, three type checks, five JOINs total. The OQL compiler generates this mechanically: each node in the GRAPH chain adds one edge JOIN and one type-check JOIN. The pattern is completely regular, which is exactly why it's worth automating -- writing this SQL by hand every time would be miserable.

## When to give up on SQL

Being honest about where this breaks:

**5+ hops.** Recursive CTE performance degrades exponentially with depth. Each hop adds a JOIN in the materialized intermediate result. At 5 hops across a moderately connected graph, DuckDB is doing more work than most purpose-built graph engines would at 10 hops. If your question needs 5+ hops, you need a graph engine.

**Real-time graph algorithms.** PageRank, community detection, connected components -- these iterate over the entire graph multiple times. You *can* implement PageRank as a recursive CTE. You shouldn't. Use NetworkX (Python), igraph (C/Python), or a dedicated graph compute engine. These algorithms need adjacency list traversal, not SQL joins.

**Variable-length path matching with constraints.** "Find all paths of length 2-5 between X and Y where every intermediate node is a Person" is one Cypher pattern: `MATCH (x)-[:KNOWS*2..5]->(y) WHERE ALL(n IN nodes(path) WHERE n:Person)`. In SQL, that's a recursive CTE with depth bounds, type checks at every hop, and array-based cycle detection. It works. It's also a tax on your sanity.

**When you hit these walls**, the export path is trivial. Triples are three columns. Load them into NetworkX with `nx.from_pandas_edgelist(df, 'subject', 'object', 'predicate')`. Load them into Neo4j with `LOAD CSV`. The data model is deliberately simple because we knew from the start that some queries would outgrow SQL.

<div class="note">
  <strong>The honest split:</strong> SQL handles 90% of graph queries we actually run -- type lookups, 1-2 hop traversals, aggregations. The remaining 10% (deep traversals, graph algorithms, complex pattern matching) gets exported to specialized tools. This is a reasonable trade-off. The alternative -- running Neo4j for everything -- means operating a graph database for the 90% of queries that DuckDB handles in milliseconds.
</div>

Recursive CTEs won't win any beauty contests. The queries are verbose, the cycle detection is manual, and anything past 3 hops starts sweating. But they run on infrastructure we already have, against data that's already there, without deploying anything new. For a project that already stores triples in DuckDB, that's not a compromise. That's a feature.
