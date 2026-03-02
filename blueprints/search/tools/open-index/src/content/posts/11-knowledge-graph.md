---
slug: knowledge-graph
title: "The Knowledge Graph Nobody Asked For"
date: 2026-02-27
summary: "2.5 million pages indexed by URL. Zero understanding of what's on them. That changes with entity extraction."
tags: [roadmap, data]
---

We have 2.5 million crawled pages in DuckDB. We can tell you the URL, the status code, the content type, the domain. We can tell you which pages link to which other pages. What we can't tell you is who wrote them, what companies they mention, or how those companies relate to each other.

That's the gap. A page index answers "where." A knowledge graph answers "what" and "who" and "how they connect." We're building one.

## Pages tell you where. Entities tell you what.

A page index says "this URL contains the string 'OpenAI'." A knowledge graph says "OpenAI is an Organization, founded by Sam Altman (Person), headquartered in San Francisco (Place), which created GPT-4 (CreativeWork)." The first is a text match. The second is structured understanding -- typed entities with named relationships between them.

The difference matters the moment you try to answer a question like "which AI companies are headquartered in the Bay Area?" A text search requires you to know every company name in advance. A knowledge graph lets you query the structure: find all entities of type Organization, where industry = AI, where location = Bay Area. The graph already knows what you'd otherwise have to enumerate by hand.

## Schema.org is already doing half the work

Here's the thing most people don't realize: millions of web pages already have structured entity data embedded in their HTML. Google requires it for rich search snippets, so every e-commerce site, news publisher, and recipe blog has JSON-LD blocks describing their content in Schema.org vocabulary.

<pre><code><span style="color:#888">&lt;!-- Embedded in the &lt;head&gt; of a typical news article --&gt;</span>
&lt;script type=<span style="color:#4ade80">"application/ld+json"</span>&gt;
{
  <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"NewsArticle"</span>,
  <span style="color:#60a5fa">"headline"</span>: <span style="color:#4ade80">"OpenAI Announces GPT-5"</span>,
  <span style="color:#60a5fa">"author"</span>: {
    <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Person"</span>,
    <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Jane Smith"</span>
  },
  <span style="color:#60a5fa">"publisher"</span>: {
    <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Organization"</span>,
    <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"TechCrunch"</span>
  },
  <span style="color:#60a5fa">"about"</span>: {
    <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Organization"</span>,
    <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"OpenAI"</span>
  }
}
&lt;/script&gt;</code></pre>

From that single block, we extract four entities (a NewsArticle, a Person, two Organizations) and three relationships (authored_by, published_by, about). No ML model. No NER pipeline. Just JSON parsing.

This isn't a small dataset. Web Data Commons estimates that 44% of pages on the Common Crawl corpus contain some form of structured data. That's hundreds of millions of pages with pre-annotated entities, waiting to be parsed.

## When structured data isn't enough

The other 56% of pages have no JSON-LD, no microdata, no RDFa. And even pages with structured data often annotate only the top-level entity -- the article itself -- while the body text mentions dozens of other people, companies, and places with no markup at all.

That's where NER (Named Entity Recognition) comes in. Run a model over the page text, extract entity mentions with types -- Person, Organization, Location, Product -- and you get structured data from unstructured prose.

The catch is entity resolution. "Apple" on a tech blog is Apple Inc. "Apple" on a recipe site is the fruit. "Washington" could be a state, a city, a president, or a football team. Resolving ambiguous mentions to canonical entities isn't a parsing problem. It's a disambiguation problem, and it's genuinely hard at web scale.

<div class="note">
  <strong>Honest status:</strong> The JSON-LD extraction pipeline is designed and ready to build. The NER pipeline isn't built yet. We're starting with structured data because it's fast, precise, and covers a surprising amount of the web. NER comes later, incrementally, once the entity store exists.
</div>

## The extraction pipeline

The plan, from crawled page to stored knowledge:

<pre>
  Crawled HTML
       |
       v
  <span style="color:#4ade80">Parse JSON-LD / Microdata / RDFa</span>    <span style="color:#888">(cheap, deterministic)</span>
       |
       +---> Structured entities + relationships
       |
       v
  <span style="color:#fbbf24">Run NER on body text</span>                <span style="color:#888">(expensive, probabilistic)</span>
       |
       +---> Raw entity mentions
       |
       v
  <span style="color:#60a5fa">Entity Resolution</span>                   <span style="color:#888">(match to canonical entities)</span>
       |
       +---> Resolved entities
       |
       v
  <span style="color:#60a5fa">Relationship Extraction</span>             <span style="color:#888">(co-occurrence, explicit predicates)</span>
       |
       v
  <span style="color:#4ade80">Triple Store (DuckDB)</span>               <span style="color:#888">(subject, predicate, object)</span>
</pre>

The first stage -- JSON-LD parsing -- is effectively free. It's string search for `<script type="application/ld+json">`, then `JSON.parse`. The NER stage is where the compute cost lives, and where most of the engineering complexity sits. By splitting the pipeline, we can run the cheap stage over the entire corpus now and add the expensive stage selectively.

## Triples: the atomic unit of knowledge

Every fact in a knowledge graph is a triple: subject, predicate, object.

- **"OpenAI"** -> `founded_by` -> **"Sam Altman"**
- **"Sam Altman"** -> `is_a` -> **"Person"**
- **"OpenAI"** -> `headquartered_in` -> **"San Francisco"**
- **"GPT-4"** -> `created_by` -> **"OpenAI"**

Triples compose into a graph. Follow the edges from OpenAI and you reach Sam Altman, San Francisco, GPT-4. Follow them further and you reach every entity connected to those. The graph is queryable: "find all entities connected to OpenAI within 2 hops" returns a subgraph of people, places, and products.

## Why we're (probably) not using Neo4j

The obvious choice for graph storage is a graph database -- Neo4j, DGraph, something purpose-built. But we already have DuckDB everywhere. The recrawler writes to it. The CC site extractor writes to it. Analytics runs on it. Adding a separate graph database means a new dependency, a new query language, a new operational concern.

A triple is three strings and two floats:

<table>
  <thead>
    <tr>
      <th>Column</th>
      <th>Type</th>
      <th>Example</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>subject</strong></td>
      <td>VARCHAR</td>
      <td><code>openai</code></td>
    </tr>
    <tr>
      <td><strong>predicate</strong></td>
      <td>VARCHAR</td>
      <td><code>founded_by</code></td>
    </tr>
    <tr>
      <td><strong>object</strong></td>
      <td>VARCHAR</td>
      <td><code>sam_altman</code></td>
    </tr>
    <tr>
      <td><strong>source_url</strong></td>
      <td>VARCHAR</td>
      <td><code>https://techcrunch.com/...</code></td>
    </tr>
    <tr>
      <td><strong>confidence</strong></td>
      <td>FLOAT</td>
      <td><code>1.0</code> (JSON-LD) / <code>0.82</code> (NER)</td>
    </tr>
  </tbody>
</table>

DuckDB handles billions of rows. Graph traversal via SQL is ugly -- recursive CTEs aren't pretty -- but for the early version, "ugly but working with zero new dependencies" beats "elegant but requires deploying and maintaining a graph database." If we outgrow DuckDB for this, we'll know exactly why, and the migration path is clear because the data model is trivial.

## What we already have vs. what we're building

The CC site extractor already builds a graph -- a link graph. Page A links to page B. That's structure, but it's web topology, not semantics. The knowledge graph adds meaning on top.

<table>
  <thead>
    <tr>
      <th>Property</th>
      <th>Link Graph (existing)</th>
      <th>Knowledge Graph (planned)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Nodes</strong></td>
      <td>URLs / pages</td>
      <td>Entities (Person, Org, Place, ...)</td>
    </tr>
    <tr>
      <td><strong>Edges</strong></td>
      <td>"links to"</td>
      <td>Named relationships (founded_by, located_in, ...)</td>
    </tr>
    <tr>
      <td><strong>Source</strong></td>
      <td>HTML <code>&lt;a&gt;</code> tags</td>
      <td>JSON-LD + NER + relationship extraction</td>
    </tr>
    <tr>
      <td><strong>Query example</strong></td>
      <td>"What pages link to openai.com?"</td>
      <td>"Who founded OpenAI?"</td>
    </tr>
    <tr>
      <td><strong>Storage</strong></td>
      <td>DuckDB (pages + links tables)</td>
      <td>DuckDB (triples table)</td>
    </tr>
    <tr>
      <td><strong>Status</strong></td>
      <td>Built, running</td>
      <td>Designed, not yet built</td>
    </tr>
  </tbody>
</table>

The two graphs are complementary. The link graph tells you which pages are important (PageRank). The knowledge graph tells you what those pages are about. Eventually, they merge: a page's importance (from links) weights the confidence of entities extracted from it.

## This is the hard one

Let's be direct about where this sits on the difficulty spectrum. The recrawler was hard because of concurrency and throughput. The knowledge graph is hard because of ambiguity and scale.

Entity resolution at web scale is an open research problem. Google has the Knowledge Graph. They also have thousands of engineers and two decades of data. We have DuckDB and a crawl corpus.

But the JSON-LD path gives us a real head start. Structured data extraction is fast, precise, and covers a larger slice of the web than most people expect. Start there. Build the triple store. Get queries working over structured entities. Then add NER incrementally -- one entity type at a time, one domain at a time -- and measure whether the added noise is worth the added coverage.

<div class="note">
  <strong>Where this fits on the roadmap:</strong> Full-text search (Tantivy) and vector search (Vald) come first -- they're closer to production-ready. The knowledge graph is the next layer after that. JSON-LD extraction will likely ship alongside or shortly after Tantivy integration. NER is further out.
</div>

The web already has more structure than we give it credit for. Millions of pages are annotated with typed entities and relationships, sitting in `<script>` tags that most crawlers ignore. We're going to stop ignoring them.
