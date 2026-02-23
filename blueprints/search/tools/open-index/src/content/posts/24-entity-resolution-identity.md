---
slug: entity-resolution-identity
title: "Entity Resolution — Who Is \"Apple\"?"
date: 2026-03-12
summary: "The same entity appears 10,000 times across 10,000 pages with 10,000 slightly different names. Merging them is the hardest problem in knowledge graphs."
tags: [ontology, ai, data]
---

You've extracted entities from a billion pages. Your NER pipeline says "Apple" appears on 3.2 million of them. Great. How many distinct entities is that?

At least two (the company, the fruit). Probably three (Apple Records). Maybe four (Apple Bank). The string "Apple" tells you nothing about identity. Context does. And context is expensive.

This is entity resolution -- the problem that sits between "we found some names" and "we have a knowledge graph." It's three problems wearing a trenchcoat.

## Three problems that look like one

People say "entity resolution" and mean different things. There are actually three distinct subproblems, each with different techniques:

**Disambiguation.** "Apple" on TechCrunch is Apple Inc. "Apple" on a recipe blog is the fruit. Same string, different entities. The surrounding text determines which one. A sentence containing "CEO," "stock price," or "launched" points to the company. A sentence containing "peel," "bake," or "orchard" points to the fruit. Context determines identity.

**Deduplication.** "Apple Inc.", "Apple", "Apple Computer", "AAPL", "Apple, Inc" across 10,000 pages all refer to the same entity. Different surface forms, same real-world thing. Merge them into one canonical record.

**Entity linking.** Match "Apple" to Wikidata Q312 (Apple Inc.) or Q89 (the fruit). Connect your local entities to external knowledge bases so you can inherit their structured data -- founding date, headquarters, CEO -- without extracting it yourself.

Three problems. Different techniques. Usually solved together in a single pipeline.

## Canonical URI scheme

Every resolved entity gets a canonical URI. This is the primary key -- the thing that unifies all surface forms, all mentions, all linked identifiers into one record.

<pre><code><span style="color:#60a5fa">https://openindex.org/entity/</span><span style="color:#fbbf24">{type}</span><span style="color:#60a5fa">/</span><span style="color:#fbbf24">{slug}</span>

<span style="color:#888">-- Examples:</span>
<span style="color:#4ade80">https://openindex.org/entity/organization/apple-inc</span>
<span style="color:#4ade80">https://openindex.org/entity/person/tim-cook</span>
<span style="color:#4ade80">https://openindex.org/entity/place/cupertino-california</span>
<span style="color:#4ade80">https://openindex.org/entity/topic/machine-learning</span></code></pre>

The slug is derived from the canonical name via lowercasing and hyphenation. Multiple surface forms map to one URI. "Apple Inc.", "Apple Computer", and "AAPL" all resolve to `organization/apple-inc`. The URI is the foreign key in the mentions table, the join key in the triples table, and the lookup key in the API.

This isn't a novel design -- it's how Wikidata, DBpedia, and every other serious knowledge base works. The novelty is building it from crawled web data rather than manual curation.

## sameAs -- the identity bridge

Once we have canonical URIs, we need to connect them to external knowledge bases. That's what `schema:sameAs` does. An OpenIndex entity can declare "I'm the same thing as Wikidata Q312 and DBpedia Apple_Inc."

<pre><code>{
  <span style="color:#60a5fa">"entity_id"</span>: <span style="color:#4ade80">"https://openindex.org/entity/organization/apple-inc"</span>,
  <span style="color:#60a5fa">"canonical_name"</span>: <span style="color:#4ade80">"Apple Inc."</span>,
  <span style="color:#60a5fa">"sameAs"</span>: [
    <span style="color:#4ade80">"https://www.wikidata.org/entity/Q312"</span>,
    <span style="color:#4ade80">"https://dbpedia.org/resource/Apple_Inc."</span>,
    <span style="color:#4ade80">"https://www.freebase.com/m/0k8z"</span>
  ]
}</code></pre>

Critical detail: `sameAs` is **not transitive** by default. If A sameAs B and B sameAs C, that doesn't automatically mean A sameAs C. This is deliberate.

Here's why. Suppose:
- OpenIndex says `apple-inc` sameAs Wikidata Q312
- Wikidata Q312 sameAs some DBpedia resource
- That DBpedia resource sameAs a Freebase entry
- The Freebase entry has a sameAs link to a YAGO entity that's actually "Apple Records"

Blindly chaining sameAs links across knowledge bases leads to **entity collapse** -- distinct entities get merged into one mega-entity because some intermediate KB made a bad identity judgment. The Beatles' record label becomes Apple Inc. becomes the fruit. Transitive closure on sameAs is a well-known disaster in the linked data community.

<div class="note">
  <strong>This isn't hypothetical.</strong> The "sameAs problem" has been extensively documented. A 2019 study found that naive transitive closure on owl:sameAs across the Linked Open Data cloud produces identity sets containing thousands of clearly distinct entities. We treat sameAs as a directional assertion, not an equivalence relation.
</div>

## The resolution pipeline

Four stages. Each one narrows the candidates and increases confidence.

### Stage 1: Candidate generation

Given a mention "Apple" with surrounding context, find candidate entities using fuzzy string matching. This is the high-recall, low-precision stage -- cast a wide net.

<pre><code><span style="color:#888">-- Find candidate entities for a mention using Jaro-Winkler similarity</span>
<span style="color:#60a5fa">SELECT</span>
  e.entity_id,
  e.canonical_name,
  e.entity_type,
  <span style="color:#e0e0e0">jaro_winkler_similarity</span>(m.surface_form, e.canonical_name) <span style="color:#60a5fa">AS</span> name_sim
<span style="color:#60a5fa">FROM</span> mentions m
<span style="color:#60a5fa">CROSS JOIN</span> entities e
<span style="color:#60a5fa">WHERE</span> <span style="color:#e0e0e0">jaro_winkler_similarity</span>(m.surface_form, e.canonical_name) > <span style="color:#fbbf24">0.85</span>
   <span style="color:#60a5fa">OR</span> m.surface_form = <span style="color:#60a5fa">ANY</span>(e.alternate_names)
<span style="color:#60a5fa">ORDER BY</span> name_sim <span style="color:#60a5fa">DESC</span>
<span style="color:#60a5fa">LIMIT</span> <span style="color:#fbbf24">10</span>;</code></pre>

Jaro-Winkler works well for entity names because it gives extra weight to prefix matches. "Apple Inc" vs "Apple Inc." scores 0.98. "Apple" vs "Apple Computer" scores 0.87. "Apple" vs "Application" scores 0.78 -- below our threshold. Fast, and it eliminates most of the entity catalog without expensive comparison.

### Stage 2: Context scoring

For each candidate, compute a feature vector from the surrounding text. This is where disambiguation actually happens.

<table>
  <thead>
    <tr>
      <th>Feature</th>
      <th>Type</th>
      <th>Example (Apple Inc.)</th>
      <th>Example (apple fruit)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>name_similarity</strong></td>
      <td>float</td>
      <td>0.95</td>
      <td>0.95</td>
    </tr>
    <tr>
      <td><strong>type_match</strong></td>
      <td>bool</td>
      <td>true (NER: ORG)</td>
      <td>false (NER: ORG)</td>
    </tr>
    <tr>
      <td><strong>domain_context</strong></td>
      <td>category</td>
      <td>tech_news</td>
      <td>tech_news</td>
    </tr>
    <tr>
      <td><strong>co_occurring_entities</strong></td>
      <td>list</td>
      <td>[Tim Cook, iPhone, WWDC]</td>
      <td>[Tim Cook, iPhone, WWDC]</td>
    </tr>
    <tr>
      <td><strong>keyword_signals</strong></td>
      <td>list</td>
      <td>[CEO, revenue, launched]</td>
      <td>[CEO, revenue, launched]</td>
    </tr>
    <tr>
      <td><strong>combined_score</strong></td>
      <td>float</td>
      <td><strong>0.94</strong></td>
      <td><strong>0.12</strong></td>
    </tr>
  </tbody>
</table>

Name similarity alone can't distinguish -- both candidates score 0.95 against "Apple." But the NER model tagged it as an Organization. The page domain is TechCrunch (tech_news category). Co-occurring entities include "Tim Cook" and "iPhone." Keyword signals include "CEO" and "revenue." Every feature except name_similarity points overwhelmingly to Apple Inc. The fruit never had a chance.

### Stage 3: KB linking

For high-confidence candidates, optionally confirm against Wikidata. This step isn't required, but it dramatically improves precision for well-known entities.

<pre><code><span style="color:#888">-- Wikidata search API call</span>
<span style="color:#e0e0e0">GET</span> <span style="color:#4ade80">https://www.wikidata.org/w/api.php</span>
  <span style="color:#60a5fa">?action=</span>wbsearchentities
  <span style="color:#60a5fa">&search=</span>Apple Inc
  <span style="color:#60a5fa">&language=</span>en
  <span style="color:#60a5fa">&type=</span>item
  <span style="color:#60a5fa">&limit=</span>5

<span style="color:#888">-- Response (truncated)</span>
{
  <span style="color:#60a5fa">"search"</span>: [
    {
      <span style="color:#60a5fa">"id"</span>: <span style="color:#4ade80">"Q312"</span>,
      <span style="color:#60a5fa">"label"</span>: <span style="color:#4ade80">"Apple Inc."</span>,
      <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"American multinational technology company"</span>
    },
    {
      <span style="color:#60a5fa">"id"</span>: <span style="color:#4ade80">"Q213710"</span>,
      <span style="color:#60a5fa">"label"</span>: <span style="color:#4ade80">"Apple Records"</span>,
      <span style="color:#60a5fa">"description"</span>: <span style="color:#4ade80">"record label founded by the Beatles"</span>
    }
  ]
}
</code></pre>

The Wikidata description gives us another signal. "American multinational technology company" matches the tech_news context far better than "record label founded by the Beatles." We now have both a Wikidata ID (Q312) and a sameAs link to store.

### Stage 4: Deduplication

After resolution, cluster remaining mentions by entity. All mentions that resolved to the same canonical URI get merged.

<pre><code><span style="color:#888">-- Merge mentions that resolved to the same entity</span>
<span style="color:#60a5fa">UPDATE</span> mentions
<span style="color:#60a5fa">SET</span> resolved_entity_id = <span style="color:#4ade80">'https://openindex.org/entity/organization/apple-inc'</span>
<span style="color:#60a5fa">WHERE</span> surface_form <span style="color:#60a5fa">IN</span> (<span style="color:#4ade80">'Apple'</span>, <span style="color:#4ade80">'Apple Inc.'</span>, <span style="color:#4ade80">'Apple Inc'</span>, <span style="color:#4ade80">'Apple Computer'</span>, <span style="color:#4ade80">'AAPL'</span>)
  <span style="color:#60a5fa">AND</span> entity_type = <span style="color:#4ade80">'Organization'</span>
  <span style="color:#60a5fa">AND</span> confidence > <span style="color:#fbbf24">0.7</span>;

<span style="color:#888">-- Update the entities table with discovered alternate names</span>
<span style="color:#60a5fa">UPDATE</span> entities
<span style="color:#60a5fa">SET</span> alternate_names = <span style="color:#e0e0e0">list_distinct</span>(
  <span style="color:#e0e0e0">list_concat</span>(alternate_names, [<span style="color:#4ade80">'Apple Computer'</span>, <span style="color:#4ade80">'AAPL'</span>])
)
<span style="color:#60a5fa">WHERE</span> entity_id = <span style="color:#4ade80">'https://openindex.org/entity/organization/apple-inc'</span>;</code></pre>

Each time we resolve a new surface form to an existing entity, the entity's alternate_names list grows. The next time we encounter that surface form, candidate generation finds it immediately via the exact match path -- no fuzzy matching needed. The system gets faster as it resolves more mentions.

## How scale changes everything

The resolution strategy that works at 10M pages fails catastrophically at 1B. Here's why:

<table>
  <thead>
    <tr>
      <th>Corpus Size</th>
      <th>Est. Mentions</th>
      <th>Pairwise Comparisons</th>
      <th>Strategy</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>10M pages</strong></td>
      <td>~50M</td>
      <td>~1.25 x 10<sup>15</sup></td>
      <td>Brute-force feasible with blocking by type. Simple SQL. Minutes.</td>
    </tr>
    <tr>
      <td><strong>100M pages</strong></td>
      <td>~500M</td>
      <td>~1.25 x 10<sup>17</sup></td>
      <td>Need LSH for candidate generation. Partition by entity type + first letter. Hours.</td>
    </tr>
    <tr>
      <td><strong>1B pages</strong></td>
      <td>~5B</td>
      <td>~1.25 x 10<sup>19</sup></td>
      <td>Incremental resolution only. Resolve new mentions against existing entities. Never re-resolve the full set. Days.</td>
    </tr>
  </tbody>
</table>

5 billion mentions. Compare every pair? That's 12.5 quintillion comparisons. At 1 million comparisons per second, that takes **396 years**. Obviously you don't do pairwise comparison at this scale.

The trick is **blocking** -- partition mentions into groups that might match and only compare within groups. Block by entity type (don't compare People against Places), by first character, by domain category. A good blocking strategy reduces 12.5 quintillion comparisons to maybe 50 billion. That's still a lot, but it fits in a weekend.

At 1B pages, even blocked comparison is too expensive to run from scratch every crawl cycle. So you switch to incremental resolution: new mentions get resolved against the existing entity catalog. Only genuinely ambiguous cases trigger full re-comparison. The entity catalog grows monotonically -- once resolved, an entity stays resolved unless evidence contradicts it.

## DuckDB implementation

Two tables. One for mentions (raw extracted entity references), one for entities (resolved canonical records).

<pre><code><span style="color:#888">-- Raw entity mentions from NER + JSON-LD extraction</span>
<span style="color:#60a5fa">CREATE TABLE</span> mentions (
  mention_id      <span style="color:#fbbf24">INTEGER</span> <span style="color:#60a5fa">PRIMARY KEY</span>,
  surface_form    <span style="color:#fbbf24">VARCHAR</span>,          <span style="color:#888">-- "Apple Inc.", "AAPL", etc.</span>
  entity_type     <span style="color:#fbbf24">VARCHAR</span>,          <span style="color:#888">-- Person, Organization, Place, ...</span>
  source_url      <span style="color:#fbbf24">VARCHAR</span>,          <span style="color:#888">-- page where mention was found</span>
  confidence      <span style="color:#fbbf24">FLOAT</span>,            <span style="color:#888">-- extraction confidence (0.0 - 1.0)</span>
  context_snippet <span style="color:#fbbf24">VARCHAR</span>,          <span style="color:#888">-- surrounding text (±50 chars)</span>
  resolved_entity_id <span style="color:#fbbf24">VARCHAR</span>       <span style="color:#888">-- NULL until resolved, then canonical URI</span>
);

<span style="color:#888">-- Resolved canonical entities</span>
<span style="color:#60a5fa">CREATE TABLE</span> entities (
  entity_id       <span style="color:#fbbf24">VARCHAR</span> <span style="color:#60a5fa">PRIMARY KEY</span>,  <span style="color:#888">-- canonical URI</span>
  canonical_name  <span style="color:#fbbf24">VARCHAR</span>,            <span style="color:#888">-- "Apple Inc."</span>
  entity_type     <span style="color:#fbbf24">VARCHAR</span>,            <span style="color:#888">-- Organization</span>
  alternate_names <span style="color:#fbbf24">VARCHAR[]</span>,          <span style="color:#888">-- ["Apple", "Apple Computer", "AAPL"]</span>
  same_as         <span style="color:#fbbf24">VARCHAR[]</span>,          <span style="color:#888">-- ["https://wikidata.org/entity/Q312"]</span>
  mention_count   <span style="color:#fbbf24">INTEGER</span>,            <span style="color:#888">-- total mentions across all pages</span>
  first_seen      <span style="color:#fbbf24">TIMESTAMP</span>,          <span style="color:#888">-- earliest mention</span>
  last_seen       <span style="color:#fbbf24">TIMESTAMP</span>           <span style="color:#888">-- most recent mention</span>
);</code></pre>

The resolution query chains candidate generation and scoring:

<pre><code><span style="color:#888">-- Find best entity match for unresolved mentions</span>
<span style="color:#60a5fa">WITH</span> candidates <span style="color:#60a5fa">AS</span> (
  <span style="color:#60a5fa">SELECT</span>
    m.mention_id,
    m.surface_form,
    e.entity_id,
    e.canonical_name,
    <span style="color:#e0e0e0">jaro_winkler_similarity</span>(m.surface_form, e.canonical_name) <span style="color:#60a5fa">AS</span> name_sim,
    <span style="color:#60a5fa">CASE WHEN</span> m.entity_type = e.entity_type <span style="color:#60a5fa">THEN</span> <span style="color:#fbbf24">0.3</span> <span style="color:#60a5fa">ELSE</span> <span style="color:#fbbf24">0.0</span> <span style="color:#60a5fa">END</span> <span style="color:#60a5fa">AS</span> type_bonus,
    <span style="color:#60a5fa">CASE WHEN</span> m.surface_form = <span style="color:#60a5fa">ANY</span>(e.alternate_names)
         <span style="color:#60a5fa">THEN</span> <span style="color:#fbbf24">1.0</span> <span style="color:#60a5fa">ELSE</span> <span style="color:#fbbf24">0.0</span> <span style="color:#60a5fa">END</span> <span style="color:#60a5fa">AS</span> exact_alt
  <span style="color:#60a5fa">FROM</span> mentions m
  <span style="color:#60a5fa">CROSS JOIN</span> entities e
  <span style="color:#60a5fa">WHERE</span> m.resolved_entity_id <span style="color:#60a5fa">IS NULL</span>
    <span style="color:#60a5fa">AND</span> (
      <span style="color:#e0e0e0">jaro_winkler_similarity</span>(m.surface_form, e.canonical_name) > <span style="color:#fbbf24">0.85</span>
      <span style="color:#60a5fa">OR</span> m.surface_form = <span style="color:#60a5fa">ANY</span>(e.alternate_names)
    )
),
scored <span style="color:#60a5fa">AS</span> (
  <span style="color:#60a5fa">SELECT</span> *,
    <span style="color:#888">-- Exact alternate name match trumps everything</span>
    <span style="color:#60a5fa">CASE WHEN</span> exact_alt = <span style="color:#fbbf24">1.0</span> <span style="color:#60a5fa">THEN</span> <span style="color:#fbbf24">1.0</span>
         <span style="color:#60a5fa">ELSE</span> (name_sim * <span style="color:#fbbf24">0.6</span>) + type_bonus
    <span style="color:#60a5fa">END</span> <span style="color:#60a5fa">AS</span> final_score,
    <span style="color:#60a5fa">ROW_NUMBER</span>() <span style="color:#60a5fa">OVER</span> (
      <span style="color:#60a5fa">PARTITION BY</span> mention_id <span style="color:#60a5fa">ORDER BY</span>
        exact_alt <span style="color:#60a5fa">DESC</span>, name_sim + type_bonus <span style="color:#60a5fa">DESC</span>
    ) <span style="color:#60a5fa">AS</span> rank
  <span style="color:#60a5fa">FROM</span> candidates
)
<span style="color:#60a5fa">SELECT</span> mention_id, entity_id, canonical_name, final_score
<span style="color:#60a5fa">FROM</span> scored
<span style="color:#60a5fa">WHERE</span> rank = <span style="color:#fbbf24">1</span> <span style="color:#60a5fa">AND</span> final_score > <span style="color:#fbbf24">0.7</span>;</code></pre>

Mentions that score below 0.7 stay unresolved. They'll either get resolved when we encounter a better surface form later, or they'll remain as unlinked mentions in the graph. Unresolved is better than wrong.

## Confidence propagation

Confidence isn't a single number. It's a chain, and every link weakens it.

<pre><code><span style="color:#e0e0e0">Extraction confidence</span> × <span style="color:#e0e0e0">Resolution confidence</span> = <span style="color:#e0e0e0">Triple confidence</span>

<span style="color:#888">-- JSON-LD extraction: the page author declared it</span>
<span style="color:#4ade80">extraction = 1.0</span>    <span style="color:#888">(deterministic)</span>
<span style="color:#4ade80">resolution = 0.95</span>   <span style="color:#888">(high — name matched exactly)</span>
<span style="color:#4ade80">triple     = 0.95</span>   <span style="color:#888">(strong — use this)</span>

<span style="color:#888">-- NER extraction: model found it in running text</span>
<span style="color:#fbbf24">extraction = 0.92</span>   <span style="color:#888">(GLiNER confidence)</span>
<span style="color:#fbbf24">resolution = 0.85</span>   <span style="color:#888">(fuzzy match, type confirmed)</span>
<span style="color:#fbbf24">triple     = 0.78</span>   <span style="color:#888">(weaker — flag for review above threshold)</span>

<span style="color:#888">-- Low-confidence NER + ambiguous resolution</span>
<span style="color:#e0e0e0">extraction = 0.71</span>   <span style="color:#888">(uncertain extraction)</span>
<span style="color:#e0e0e0">resolution = 0.62</span>   <span style="color:#888">(multiple candidates scored close)</span>
<span style="color:#e0e0e0">triple     = 0.44</span>   <span style="color:#888">(below threshold — discard)</span></code></pre>

The rule is simple: confidence inherits from the weakest link. A perfectly extracted entity that resolves ambiguously produces a low-confidence triple. A poorly extracted entity that happens to match exactly still gets dragged down by the extraction score.

We set a minimum triple confidence of 0.5 for inclusion in the knowledge graph. Below that, the mention is stored but doesn't generate triples. This means roughly 15-20% of NER-extracted mentions won't produce knowledge graph edges -- and that's fine. Precision matters more than recall when you're building a graph that downstream queries will trust.

<div class="note">
  <strong>Why multiply instead of min?</strong> Multiplication penalizes double uncertainty more harshly. If both extraction and resolution are at 0.7, min gives 0.7 (looks okay), but multiply gives 0.49 (below threshold, discarded). When two stages are both uncertain, the combined result should reflect that compounded risk. Multiplication does this naturally.
</div>

## Where this sits in the architecture

Entity resolution runs as a batch job after NER extraction. It doesn't need to be real-time -- new crawl results get resolved in bulk once per crawl cycle.

<pre><code>  <span style="color:#4ade80">Crawled HTML</span>
       |
       v
  <span style="color:#fbbf24">JSON-LD + NER Extraction</span>        <span style="color:#888">── produces raw mentions</span>
       |
       v
  <span style="color:#60a5fa">Entity Resolution (batch)</span>       <span style="color:#888">── this post</span>
       |
       +── Candidate generation    <span style="color:#888">── DuckDB jaro_winkler_similarity</span>
       +── Context scoring         <span style="color:#888">── feature vectors from page context</span>
       +── KB linking (optional)   <span style="color:#888">── Wikidata API, disabled by default</span>
       +── Deduplication           <span style="color:#888">── merge surface forms → canonical URI</span>
       |
       v
  <span style="color:#4ade80">Knowledge Graph Triples</span>         <span style="color:#888">── (subject, predicate, object) in DuckDB</span></code></pre>

DuckDB's string functions -- `jaro_winkler_similarity`, `levenshtein`, `list_contains` -- handle candidate generation entirely in SQL. No external services, no Python scripts, no Spark clusters. The whole pipeline is SQL-native, which means it runs anywhere DuckDB runs: your laptop, a CI server, a single beefy machine.

Wikidata linking is optional and disabled by default. For most crawls, the combination of fuzzy matching + type constraints + co-occurrence signals is enough. Enable it per crawl when you need high-precision entities for a specific domain.

<div class="note">
  <strong>Honest status:</strong> The schema is designed. The candidate generation queries work in DuckDB today. Context scoring and Wikidata linking are prototyped but not integrated into the pipeline. Full batch resolution targeting mid-2026, alongside the knowledge graph triple store.
</div>

Posts 11 and 14 called entity resolution "genuinely hard" and "an open research problem." They weren't wrong. But "hard" doesn't mean "impossible" -- it means you need a pipeline with multiple stages, honest confidence scores, and the discipline to leave ambiguous mentions unresolved rather than guessing wrong. That's what we're building.
