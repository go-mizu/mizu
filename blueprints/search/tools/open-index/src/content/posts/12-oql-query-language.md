---
slug: oql-query-language
title: "OQL — One Query, Four Backends"
date: 2026-02-28
summary: "Four search backends, one query language. Full-text, vectors, graph traversal, and metadata in a single expression."
tags: [roadmap, search]
---

Searching the open web right now means four different tools. Want keyword search? That's Tantivy. Semantic similarity? Vald. Entity relationships? Graph queries. Filter by domain, date, HTTP status? DuckDB SQL. Four APIs, four query syntaxes, four result sets you have to stitch together by hand.

Nobody wants to do that. So we're building OQL.

## What does a query actually look like?

OQL is SQL-like on purpose. Here's the simplest possible query:

<pre><code><span style="color:#60a5fa">SEARCH</span> <span style="color:#4ade80">"climate change"</span></code></pre>

That's a full-text search via Tantivy. BM25 scoring. Returns ranked documents. Now add a filter:

<pre><code><span style="color:#60a5fa">SEARCH</span> <span style="color:#4ade80">"climate change"</span>
<span style="color:#60a5fa">WHERE</span> <span style="color:#e0e0e0">domain</span> <span style="color:#60a5fa">LIKE</span> <span style="color:#4ade80">'%.edu'</span>
  <span style="color:#60a5fa">AND</span> <span style="color:#e0e0e0">date</span> > <span style="color:#4ade80">'2025-01-01'</span></code></pre>

Tantivy handles the keyword search, DuckDB handles the metadata filter. Two backends, one query. Now add semantic search:

<pre><code><span style="color:#60a5fa">SEARCH</span> <span style="color:#4ade80">"climate change"</span>
<span style="color:#60a5fa">SIMILAR TO</span> <span style="color:#4ade80">"global warming policy proposals"</span>
<span style="color:#60a5fa">WHERE</span> <span style="color:#e0e0e0">domain</span> <span style="color:#60a5fa">LIKE</span> <span style="color:#4ade80">'%.edu'</span>
<span style="color:#60a5fa">ORDER BY</span> <span style="color:#e0e0e0">score</span> <span style="color:#60a5fa">DESC</span>
<span style="color:#60a5fa">LIMIT</span> <span style="color:#fbbf24">50</span></code></pre>

Three backends now. Tantivy finds keyword matches. Vald finds semantically similar documents -- pages about carbon tax legislation that never mention "climate change" by name. DuckDB filters to `.edu` domains. Score fusion merges the rankings. And the full version, with graph traversal:

<pre><code><span style="color:#60a5fa">SEARCH</span> <span style="color:#4ade80">"climate change"</span>
<span style="color:#60a5fa">SIMILAR TO</span> <span style="color:#4ade80">"environmental policy"</span>
<span style="color:#60a5fa">GRAPH</span> <span style="color:#e0e0e0">Organization</span> <span style="color:#60a5fa">LINKED_TO</span> <span style="color:#e0e0e0">Person</span>
<span style="color:#60a5fa">WHERE</span> <span style="color:#e0e0e0">domain</span> <span style="color:#60a5fa">LIKE</span> <span style="color:#4ade80">'%.gov'</span>
  <span style="color:#60a5fa">AND</span> <span style="color:#e0e0e0">content_type</span> = <span style="color:#4ade80">'text/html'</span>
<span style="color:#60a5fa">ORDER BY</span> <span style="color:#e0e0e0">score</span> <span style="color:#60a5fa">DESC</span>
<span style="color:#60a5fa">LIMIT</span> <span style="color:#fbbf24">20</span></code></pre>

All four backends. One query. The keyword hits, semantic neighbors, entity relationships, and metadata filters all merge into a single ranked result set.

## Why not invent something new?

SQL has been around for 50 years. Every engineer can read it or learn it in an afternoon. GraphQL is too deeply nested for search workloads. Custom DSLs mean documentation nobody reads and syntax nobody remembers.

We took the parts of SQL that work -- `SELECT`, `WHERE`, `ORDER BY`, `LIMIT` -- and added three new clauses: `SEARCH`, `SIMILAR TO`, and `GRAPH`. If you can read SQL, you can read OQL. That's the whole design philosophy.

<div class="note">
  <strong>Familiarity beats novelty.</strong> Every new syntax you invent is a syntax someone has to learn. OQL extends SQL rather than replacing it. The goal is that anyone who's written a WHERE clause can start querying OpenIndex in minutes.
</div>

## The four clauses

### SEARCH -- full-text via Tantivy

Maps directly to Tantivy's inverted index. BM25 scoring. Supports phrases, boolean operators, and field-specific search:

<pre><code><span style="color:#888">-- Phrase search</span>
<span style="color:#60a5fa">SEARCH</span> <span style="color:#4ade80">"machine learning"</span>

<span style="color:#888">-- Boolean operators</span>
<span style="color:#60a5fa">SEARCH</span> <span style="color:#4ade80">"rust AND webassembly NOT javascript"</span>

<span style="color:#888">-- Field-specific</span>
<span style="color:#60a5fa">SEARCH</span> <span style="color:#e0e0e0">title:</span><span style="color:#4ade80">"OpenIndex"</span> <span style="color:#60a5fa">AND</span> <span style="color:#e0e0e0">body:</span><span style="color:#4ade80">"web crawling"</span></code></pre>

Returns document IDs with BM25 scores. Scores typically range from 0 to ~25 depending on term frequency and document length.

### SIMILAR TO -- vector search via Vald

Takes a text string, runs it through multilingual-e5-large embeddings, and finds nearest vectors in Vald:

<pre><code><span style="color:#888">-- Find pages semantically similar to a concept</span>
<span style="color:#60a5fa">SIMILAR TO</span> <span style="color:#4ade80">"renewable energy infrastructure investment"</span>

<span style="color:#888">-- Combine with keyword search for hybrid results</span>
<span style="color:#60a5fa">SEARCH</span> <span style="color:#4ade80">"solar panel"</span>
<span style="color:#60a5fa">SIMILAR TO</span> <span style="color:#4ade80">"photovoltaic energy generation systems"</span></code></pre>

Returns document IDs with cosine similarity scores (0 to 1). When combined with `SEARCH`, both score sets feed into fusion.

### GRAPH -- knowledge graph traversal

Queries the entity graph built from NER extraction and Schema.org structured data:

<pre><code><span style="color:#888">-- Find organizations linked to people</span>
<span style="color:#60a5fa">GRAPH</span> <span style="color:#e0e0e0">Organization</span> <span style="color:#60a5fa">LINKED_TO</span> <span style="color:#e0e0e0">Person</span>

<span style="color:#888">-- Multi-hop traversal</span>
<span style="color:#60a5fa">GRAPH</span> <span style="color:#e0e0e0">Person</span> <span style="color:#60a5fa">FOUNDED_BY</span> <span style="color:#e0e0e0">Organization</span> <span style="color:#60a5fa">LOCATED_IN</span> <span style="color:#e0e0e0">Place</span></code></pre>

Entity types map to the OpenIndex ontology: `Person`, `Organization`, `Place`, `Event`, `Product`. Relationship predicates (`LINKED_TO`, `FOUNDED_BY`, `LOCATED_IN`) correspond to edges in the knowledge graph.

### WHERE -- metadata filtering via DuckDB

Standard SQL predicates against the crawl metadata columns:

<pre><code><span style="color:#60a5fa">WHERE</span> <span style="color:#e0e0e0">domain</span> <span style="color:#60a5fa">LIKE</span> <span style="color:#4ade80">'%.edu'</span>
  <span style="color:#60a5fa">AND</span> <span style="color:#e0e0e0">date</span> > <span style="color:#4ade80">'2025-06-01'</span>
  <span style="color:#60a5fa">AND</span> <span style="color:#e0e0e0">status</span> = <span style="color:#fbbf24">200</span>
  <span style="color:#60a5fa">AND</span> <span style="color:#e0e0e0">content_type</span> = <span style="color:#4ade80">'text/html'</span>
  <span style="color:#60a5fa">AND</span> <span style="color:#e0e0e0">content_length</span> > <span style="color:#fbbf24">5000</span></code></pre>

This is the narrowing layer. Every other clause produces candidate documents; `WHERE` filters them down.

## Score fusion is the hard part

`SEARCH` returns BM25 scores (0 to ~25). `SIMILAR TO` returns cosine similarity (0 to 1). `GRAPH` returns binary matches -- an entity relationship either exists or it doesn't. These scales are completely incompatible. You can't just add them together.

Three options:

<table>
  <thead>
    <tr>
      <th>Method</th>
      <th>How It Works</th>
      <th>Trade-off</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Reciprocal Rank Fusion (RRF)</strong></td>
      <td>Merges by rank position, ignoring raw scores</td>
      <td>Simple, no tuning required, surprisingly effective</td>
    </tr>
    <tr>
      <td><strong>Weighted Linear</strong></td>
      <td>Normalize scores to [0,1], apply weights</td>
      <td>Tunable but requires choosing weights per query type</td>
    </tr>
    <tr>
      <td><strong>Learn-to-Rank</strong></td>
      <td>ML model trained on relevance judgments</td>
      <td>Best quality, requires training data we don't have yet</td>
    </tr>
  </tbody>
</table>

We're starting with RRF. The formula is straightforward:

<pre><code><span style="color:#888">-- Reciprocal Rank Fusion</span>
<span style="color:#888">-- For each document d appearing in any result set:</span>
<span style="color:#e0e0e0">RRF(d)</span> = <span style="color:#e0e0e0">SUM</span>( <span style="color:#fbbf24">1</span> / (<span style="color:#e0e0e0">k</span> + <span style="color:#e0e0e0">rank(d, result_set)</span>) )

<span style="color:#888">-- k = 60 (standard constant)</span>
<span style="color:#888">-- rank = position in each result set (1-indexed)</span>
<span style="color:#888">-- Documents appearing in multiple sets score higher</span>

<span style="color:#888">-- Example: document appears at rank 3 in SEARCH, rank 7 in SIMILAR TO</span>
<span style="color:#e0e0e0">RRF</span> = <span style="color:#fbbf24">1</span>/(<span style="color:#fbbf24">60</span>+<span style="color:#fbbf24">3</span>) + <span style="color:#fbbf24">1</span>/(<span style="color:#fbbf24">60</span>+<span style="color:#fbbf24">7</span>) = <span style="color:#fbbf24">0.01587</span> + <span style="color:#fbbf24">0.01493</span> = <span style="color:#fbbf24">0.03080</span></code></pre>

RRF has a nice property: it doesn't care about score magnitude. A BM25 score of 22.7 and a cosine similarity of 0.89 are treated the same way -- only their rank positions matter. This makes it parameter-free across different scoring systems.

<div class="note">
  <strong>Why not weighted linear from the start?</strong> Because picking weights is a research problem. Should BM25 count for 60% and cosine similarity for 40%? Or 50/50? It depends on the query, the corpus, and what the user actually wants. RRF sidesteps this entirely by ignoring raw scores. We'll add configurable fusion strategies later, but RRF is the right default.
</div>

## Parse, plan, execute

An OQL query goes through three stages:

<pre><code>  <span style="color:#e0e0e0">OQL Query Text</span>
       |
       v
  <span style="color:#60a5fa">┌─────────────────────────────────┐</span>
  <span style="color:#60a5fa">│</span>  <strong>Parser</strong>                         <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  OQL text → AST                 <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  Validates syntax, extracts     <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  clauses into typed nodes        <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">└───────────────┬─────────────────┘</span>
                |
                v
  <span style="color:#60a5fa">┌─────────────────────────────────┐</span>
  <span style="color:#60a5fa">│</span>  <strong>Planner</strong>                        <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  Routes clauses to backends:    <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  SEARCH    → <span style="color:#4ade80">Tantivy</span>            <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  SIMILAR   → <span style="color:#4ade80">Vald</span>               <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  GRAPH     → <span style="color:#4ade80">Knowledge Graph</span>    <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  WHERE     → <span style="color:#4ade80">DuckDB</span>             <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">└───────────────┬─────────────────┘</span>
                |
                v
  <span style="color:#60a5fa">┌─────────────────────────────────┐</span>
  <span style="color:#60a5fa">│</span>  <strong>Executor</strong>                       <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  Runs backend queries in        <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  parallel, collects result sets  <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">└───────────────┬─────────────────┘</span>
                |
                v
  <span style="color:#60a5fa">┌─────────────────────────────────┐</span>
  <span style="color:#60a5fa">│</span>  <strong>Fusion</strong>                         <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  Reciprocal Rank Fusion (RRF)   <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">│</span>  Merge + rank → final results   <span style="color:#60a5fa">│</span>
  <span style="color:#60a5fa">└─────────────────────────────────┘</span></code></pre>

The key decision is parallelism. When a query has `SEARCH`, `SIMILAR TO`, and `GRAPH` clauses, those three backend calls are independent -- they can run simultaneously. Only after all three return does fusion begin. The `WHERE` clause can run in parallel too (as a DuckDB query), or it can be applied as a post-filter on the fused results. The planner decides based on selectivity estimates.

## What each clause maps to

<table>
  <thead>
    <tr>
      <th>OQL Clause</th>
      <th>Backend</th>
      <th>Returns</th>
      <th>Score Type</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>SEARCH</code></td>
      <td>Tantivy</td>
      <td>Document IDs</td>
      <td>BM25 (0 to ~25)</td>
    </tr>
    <tr>
      <td><code>SIMILAR TO</code></td>
      <td>Vald</td>
      <td>Document IDs</td>
      <td>Cosine similarity (0 to 1)</td>
    </tr>
    <tr>
      <td><code>GRAPH</code></td>
      <td>Knowledge Graph</td>
      <td>Entity + Document IDs</td>
      <td>Binary (match or no match)</td>
    </tr>
    <tr>
      <td><code>WHERE</code></td>
      <td>DuckDB</td>
      <td>Document IDs</td>
      <td>Filter (no score)</td>
    </tr>
    <tr>
      <td><code>ORDER BY</code></td>
      <td>Fusion layer</td>
      <td>Sorted results</td>
      <td>Fused RRF score</td>
    </tr>
    <tr>
      <td><code>LIMIT</code></td>
      <td>Fusion layer</td>
      <td>Top-K results</td>
      <td>--</td>
    </tr>
  </tbody>
</table>

## Where this stands

Let's be direct: OQL doesn't exist yet. The grammar is being defined. There isn't a parser. What does exist is each backend independently:

- **Tantivy** -- planned for full-text BM25 search over the crawl corpus
- **Vald** -- planned for vector search with multilingual-e5-large embeddings
- **Knowledge Graph** -- planned entity extraction from Schema.org + NER
- **DuckDB** -- working today, 16-shard storage with full SQL

OQL is the glue. It turns four separate tools into one interface. The language design comes first -- get the syntax right, define the AST, build the parser. Then wire up each clause to its backend as those backends come online.

Target: parser and basic queries (SEARCH + WHERE) by end of 2026. SIMILAR TO and GRAPH follow as Vald and the knowledge graph mature.

<div class="note">
  <strong>This is a design document, not a release announcement.</strong> We're sharing the OQL design early because we want feedback on the syntax before writing a parser. If you've built query languages or have opinions about what works and what doesn't, <a href="https://github.com/nicholasgasior/gopher-crawl">open an issue</a>.
</div>

The goal isn't to invent a new query paradigm. It's to make four powerful backends feel like one. Write a query, get results. That's it.
