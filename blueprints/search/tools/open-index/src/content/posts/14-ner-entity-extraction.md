---
slug: ner-entity-extraction
title: "Teaching Machines to Read the Web"
date: 2026-03-02
summary: "Transformer models reading web pages and finding every person, company, and place mentioned. At a billion pages."
tags: [ai, data]
---

We have a billion crawled pages. Tantivy can find pages containing the word "EPA." Vald can find pages semantically related to "environmental regulation." But neither tells us *who* or *what* a page is about.

"This page mentions climate change" vs. "This page is about the **EPA** (Organization), reports written by **Dr. Jane Smith** (Person), covering policies in **Washington D.C.** (Place)."

That gap -- between keyword occurrence and structured understanding -- is what Named Entity Recognition closes.

## Two passes, not one

Here's the trick: you don't need to run an ML model on every page. A huge chunk of the web already has structured entity data sitting in `<script>` tags.

**Pass 1: Parse JSON-LD and Schema.org.** Web Data Commons estimates 44% of crawled pages contain structured data. That's free entities -- typed, named, with relationships -- extracted by JSON parsing. Zero model inference. Zero GPU cost.

**Pass 2: Run transformer NER on the rest.** The other 56% of pages have no structured markup. For these, a model reads the text and extracts entity mentions with types and confidence scores. Expensive, probabilistic, but it's the only way to get entities from raw prose.

Why do JSON-LD first? It's exact. It's fast. And it's already there. No model will ever beat the precision of `"@type": "Person", "name": "Jane Smith"` -- that's the page author *telling you* who they are.

<div class="note">
  <strong>The 44% number is conservative.</strong> It counts pages with any Schema.org markup. Among high-quality domains -- news sites, e-commerce, company pages -- the coverage is closer to 70-80%. The pages most worth extracting entities from are also the most likely to have JSON-LD already.
</div>

## What NER actually extracts

Six entity types cover most of the useful ground: Person, Organization, Location, CreativeWork, Event, Product.

Given raw text, the model outputs entity spans with types and confidence:

<pre><code><span style="color:#e0e0e0">Input:</span>  <span style="color:#888">"Dr. Sarah Chen at MIT published a study on GPT-4's impact</span>
        <span style="color:#888"> on drug discovery in Boston last September."</span>

<span style="color:#e0e0e0">Output:</span>
  <span style="color:#60a5fa">Dr. Sarah Chen</span>    <span style="color:#fbbf24">Person</span>         <span style="color:#4ade80">0.97</span>
  <span style="color:#60a5fa">MIT</span>               <span style="color:#fbbf24">Organization</span>   <span style="color:#4ade80">0.99</span>
  <span style="color:#60a5fa">GPT-4</span>             <span style="color:#fbbf24">Product</span>        <span style="color:#4ade80">0.91</span>
  <span style="color:#60a5fa">Boston</span>            <span style="color:#fbbf24">Location</span>       <span style="color:#4ade80">0.95</span>
  <span style="color:#60a5fa">last September</span>    <span style="color:#fbbf24">Event/Date</span>     <span style="color:#4ade80">0.88</span></code></pre>

Each entity gets a character span (start/end offset), a type label, and a confidence score. The spans let you trace back to the exact source text. The confidence score lets downstream systems decide what to keep and what to discard.

## Why small specialized models, not GPT-4

GPT-4 can do NER. It can also write poetry about NER. We don't need poetry. We need entity labels at a billion pages.

<table>
  <thead>
    <tr>
      <th>Approach</th>
      <th>Model Size</th>
      <th>Cost per 1B Pages</th>
      <th>Throughput</th>
      <th>F1 Score (NER)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>GPT-4 API</strong></td>
      <td>~1.8T params</td>
      <td>~$150,000</td>
      <td>~10 pages/s</td>
      <td>~92%</td>
    </tr>
    <tr>
      <td><strong>GLiNER-large</strong></td>
      <td>350M params</td>
      <td>~$800 (GPU rental)</td>
      <td>~1,000 pages/s</td>
      <td>~90%</td>
    </tr>
    <tr>
      <td><strong>spaCy transformer</strong></td>
      <td>110M params</td>
      <td>~$400 (GPU rental)</td>
      <td>~2,500 pages/s</td>
      <td>~88%</td>
    </tr>
    <tr>
      <td><strong>JSON-LD parsing</strong></td>
      <td>0 params</td>
      <td>~$0</td>
      <td>~50,000 pages/s</td>
      <td>100%</td>
    </tr>
  </tbody>
</table>

At 5KB average per page, one billion pages is about 5TB of text, roughly 1.5 trillion tokens. GPT-4 at $30/M input tokens: **$150,000** just for extraction. A fine-tuned 350M-parameter model on a rented A100 does the same job for the cost of two weeks of GPU time. The F1 difference is 2 points. The cost difference is 200x.

<div class="note">
  <strong>GLiNER is particularly interesting here.</strong> Unlike traditional NER models with fixed entity types, GLiNER accepts entity labels as input -- you can ask it to find "Person, Organization, Location" on one page and "Drug, Gene, Disease" on another. Same model, different extraction targets. That flexibility matters when the corpus spans every domain on the web.
</div>

## Entity resolution -- the genuinely hard part

The model says "Apple" is an entity. Is it Apple Inc. or the fruit?

On TechCrunch, it's a company. On a recipe blog, it's a fruit. On a page about the Beatles' record label, it's both. Context matters, and context is expensive to reason about.

Three approaches, in order of difficulty:

1. **Type constraints from surrounding text.** If the sentence contains "CEO," "stock price," or "launched," the entity is probably an organization. Cheap heuristics that work surprisingly often.
2. **Coreference resolution.** "Apple announced... The company said..." -- "the company" refers to Apple, confirming it's an organization. Requires a separate model pass.
3. **Entity linking to Wikidata.** Match "Apple" against a knowledge base of 100M+ entities. Use context vectors to pick the right one. The most accurate approach, but requires maintaining a massive entity database.

This is an open research problem at web scale. Google has the Knowledge Graph and thousands of engineers. We're building toward it incrementally -- start with high-confidence entities from JSON-LD, add NER with type constraints, and layer in entity linking as the knowledge graph grows.

## The extraction pipeline

End to end, crawled HTML to knowledge graph triples:

<pre><code>  <span style="color:#4ade80">Crawled HTML</span>
       |
       v
  <span style="color:#fbbf24">JSON-LD / Schema.org Parse</span>     <span style="color:#888">── 44% of pages, exact, free</span>
       |
       +──> <span style="color:#4ade80">Structured entities</span>
       |
       v
  <span style="color:#60a5fa">Transformer NER Model</span>          <span style="color:#888">── 56% of pages, probabilistic</span>
       |
       +──> <span style="color:#fbbf24">Raw entity mentions</span>
       |
       v
  <span style="color:#60a5fa">Entity Resolution</span>              <span style="color:#888">── disambiguate, link to canonical IDs</span>
       |
       v
  <span style="color:#fbbf24">Deduplication</span>                  <span style="color:#888">── merge mentions across pages</span>
       |
       v
  <span style="color:#4ade80">Knowledge Graph Triples</span>        <span style="color:#888">── (subject, predicate, object) in DuckDB</span></code></pre>

The JSON-LD stage runs on CPU at 50K pages/second. The NER stage requires GPU but only runs on pages that lack structured data. Entity resolution is the bottleneck -- it's where the ambiguity lives and where the most engineering time will go.

## Batch inference on GPU

The NER model runs in batches. Feed it 32-128 text chunks at once and the GPU stays saturated. The trick is pipelining: while the current batch is running inference, the next batch is being tokenized on CPU.

<table>
  <thead>
    <tr>
      <th>Hardware</th>
      <th>Batch Size</th>
      <th>Pages/Second</th>
      <th>Time for 560M Pages</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>1x A100 (80GB)</td>
      <td>64</td>
      <td>~1,000</td>
      <td>~6.5 days</td>
    </tr>
    <tr>
      <td>4x A100</td>
      <td>64</td>
      <td>~3,800</td>
      <td>~1.7 days</td>
    </tr>
    <tr>
      <td>8x A100</td>
      <td>128</td>
      <td>~7,200</td>
      <td>~21 hours</td>
    </tr>
  </tbody>
</table>

560M pages is the 56% without structured data. At 1,000 pages/second on one A100, that's under a week. With a multi-GPU setup, under two days. This isn't a latency-sensitive path -- it's a batch job that runs once per crawl cycle.

GPU utilization is what matters. A half-empty batch wastes money. Pre-tokenize aggressively, keep the inference queue full, and the per-page cost drops to fractions of a cent.

## Integration with the knowledge graph

Extracted entities become nodes. Relationships become edges. Everything is stored as triples -- `(subject, predicate, object)` -- in DuckDB's existing 16-shard architecture.

A page about Dr. Sarah Chen's MIT study produces:

- `sarah_chen` -> `affiliated_with` -> `mit`
- `sarah_chen` -> `is_a` -> `Person`
- `mit` -> `is_a` -> `Organization`
- `sarah_chen` -> `published` -> `gpt4_drug_discovery_study`
- `gpt4_drug_discovery_study` -> `mentions` -> `gpt-4`

Queryable via OQL's planned `GRAPH` clause: `GRAPH MATCH (p:Person)-[:affiliated_with]->(o:Organization) WHERE o.name = "MIT"` returns every researcher linked to MIT across the entire corpus.

## Where this stands

Honest status:

- **JSON-LD parsing**: Prototyped. The extraction code works. Integration with the triple store is designed.
- **NER model selection**: Evaluating GLiNER (flexible entity types, strong zero-shot), spaCy transformers (battle-tested, fast), and custom fine-tuned options.
- **Entity resolution**: Research phase. High-confidence heuristics first, Wikidata linking later.
- **Full pipeline**: Targeting mid-2026.

<div class="note">
  <strong>Sequencing matters.</strong> JSON-LD extraction ships first -- it's cheap, precise, and covers the highest-quality pages. NER adds coverage incrementally. Entity resolution improves accuracy over time as the knowledge graph accumulates more context for disambiguation. Each layer makes the next one better.
</div>

The web is full of structure that most crawlers throw away. JSON-LD blocks, Schema.org annotations, typed entity markup -- all sitting in HTML that gets stripped down to plain text. We're going to read it properly.
