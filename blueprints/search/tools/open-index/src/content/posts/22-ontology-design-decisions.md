---
slug: ontology-design-decisions
title: "6 Entity Types, 7 Relationships, and a Thousand Arguments"
date: 2026-03-10
summary: "Every ontology is an opinion about what matters. Here's how we picked our entity types, why Topic isn't in Schema.org, and what we're deliberately leaving out."
tags: [ontology, roadmap]
---

Every ontology is an opinion disguised as a data model.

Every type you include is a bet that it matters. Every type you exclude is a bet that it doesn't. There's no objectively correct ontology for the web -- there's only one that's useful for your specific queries, your specific corpus, your specific pipeline. Schema.org has 800+ types. Wikidata has 100M+ items. DBpedia maps all of Wikipedia. We looked at all of them and picked 6 entity types and 7 relationships.

That sounds absurdly small. It is. That's the point.

## The selection filter

We didn't start with "what exists in the world." We started with "what can we reliably extract from web pages at scale, and what forms useful graph structure once extracted." Three questions acted as a filter:

1. **Extractable?** Can we pull this entity type from crawled HTML -- via JSON-LD, NER, or heuristics -- with acceptable precision?
2. **Frequent?** Does it appear on more than ~1% of pages? Rare entity types add schema complexity without adding query value.
3. **Connected?** Does it form meaningful relationships with other types? Isolated nodes are just metadata. Graph value comes from edges.

If an entity type doesn't pass all three, it doesn't earn a spot.

<table>
  <thead>
    <tr>
      <th>Entity Type</th>
      <th>Extractable?</th>
      <th>Frequent?</th>
      <th>Connected?</th>
      <th>In Schema.org?</th>
      <th>Decision</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>WebPage</strong></td>
      <td style="color:#4ade80">Always</td>
      <td style="color:#4ade80">100%</td>
      <td style="color:#4ade80">Anchor node</td>
      <td style="color:#4ade80">Yes</td>
      <td style="color:#4ade80"><strong>IN</strong></td>
    </tr>
    <tr>
      <td><strong>Person</strong></td>
      <td style="color:#4ade80">JSON-LD + NER</td>
      <td style="color:#4ade80">~35%</td>
      <td style="color:#4ade80">High</td>
      <td style="color:#4ade80">Yes</td>
      <td style="color:#4ade80"><strong>IN</strong></td>
    </tr>
    <tr>
      <td><strong>Organization</strong></td>
      <td style="color:#4ade80">JSON-LD + NER</td>
      <td style="color:#4ade80">~40%</td>
      <td style="color:#4ade80">High</td>
      <td style="color:#4ade80">Yes</td>
      <td style="color:#4ade80"><strong>IN</strong></td>
    </tr>
    <tr>
      <td><strong>Place</strong></td>
      <td style="color:#4ade80">NER + geocoding</td>
      <td style="color:#4ade80">~20%</td>
      <td style="color:#4ade80">Medium</td>
      <td style="color:#4ade80">Yes</td>
      <td style="color:#4ade80"><strong>IN</strong></td>
    </tr>
    <tr>
      <td><strong>Topic</strong></td>
      <td style="color:#fbbf24">Classification</td>
      <td style="color:#4ade80">~80%</td>
      <td style="color:#4ade80">Very high</td>
      <td style="color:#fbbf24">No</td>
      <td style="color:#4ade80"><strong>IN</strong></td>
    </tr>
    <tr>
      <td><strong>Product</strong></td>
      <td style="color:#4ade80">JSON-LD + NER</td>
      <td style="color:#4ade80">~15%</td>
      <td style="color:#4ade80">Medium</td>
      <td style="color:#4ade80">Yes</td>
      <td style="color:#4ade80"><strong>IN</strong></td>
    </tr>
    <tr>
      <td>Event</td>
      <td style="color:#4ade80">JSON-LD</td>
      <td style="color:#fbbf24">~5%</td>
      <td style="color:#fbbf24">Temporal</td>
      <td style="color:#4ade80">Yes</td>
      <td style="color:#e0e0e0">OUT</td>
    </tr>
    <tr>
      <td>CreativeWork</td>
      <td style="color:#4ade80">JSON-LD</td>
      <td style="color:#4ade80">~25%</td>
      <td style="color:#fbbf24">Overlaps WebPage</td>
      <td style="color:#4ade80">Yes</td>
      <td style="color:#e0e0e0">OUT</td>
    </tr>
    <tr>
      <td>MedicalEntity</td>
      <td style="color:#fbbf24">Domain-specific NER</td>
      <td style="color:#e0e0e0">~0.3%</td>
      <td style="color:#fbbf24">Domain-specific</td>
      <td style="color:#4ade80">Yes</td>
      <td style="color:#e0e0e0">OUT</td>
    </tr>
    <tr>
      <td>FinancialProduct</td>
      <td style="color:#fbbf24">Domain-specific</td>
      <td style="color:#e0e0e0">~0.5%</td>
      <td style="color:#fbbf24">Domain-specific</td>
      <td style="color:#4ade80">Yes</td>
      <td style="color:#e0e0e0">OUT</td>
    </tr>
  </tbody>
</table>

CreativeWork is the interesting rejection. It's common and extractable, but it overlaps almost entirely with WebPage. A news article is a WebPage. A blog post is a WebPage. Treating them as separate types splits the graph into two parallel structures describing the same underlying content. We chose to keep WebPage as the single document node and use Topic and relationship edges to capture what a CreativeWork subtype would have added.

## Deep-dive: each type and why it's shaped the way it is

### WebPage -- the provenance anchor

Every entity in the graph connects back to at least one WebPage. Without it, you have assertions floating in space -- "Elon Musk works at Tesla" with no source. WebPage is the "where did we learn this" node. It grounds every triple in a crawled, timestamped document.

The `oi:fetchTime` and `oi:crawlId` properties aren't in Schema.org. We added them because provenance matters: the same page crawled six months apart might make contradictory claims, and downstream consumers need to know which version they're looking at.

### Person -- the alias problem

A Person entity isn't a name. It's a cluster of names that all refer to the same human. The `alternateName[]` array does the heavy lifting:

<pre><code>{
  <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Person"</span>,
  <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Elon Musk"</span>,
  <span style="color:#60a5fa">"alternateName"</span>: [<span style="color:#4ade80">"Musk"</span>, <span style="color:#4ade80">"@elonmusk"</span>, <span style="color:#4ade80">"Elon Reeve Musk"</span>],
  <span style="color:#60a5fa">"sameAs"</span>: [<span style="color:#4ade80">"https://www.wikidata.org/wiki/Q317521"</span>]
}</code></pre>

NER extracts "Musk" from one page and "Elon Musk" from another. Entity resolution merges them into a single node. The `sameAs` link to Wikidata provides a global identifier that survives name changes, typos, and transliterations. Schema.org gives us `alternateName` for free -- one of the few cases where the spec anticipated exactly the problem we're solving.

### Organization -- flat on purpose

Schema.org offers `Corporation`, `NGO`, `EducationalOrganization`, `GovernmentOrganization`, and a dozen more subtypes of Organization. We deliberately flattened them all into a single Organization type.

Why? Subtype classification is a separate problem from entity extraction. Deciding whether "OpenAI" is a Corporation, a NonProfit, or a ResearchOrganization requires domain knowledge that changes over time (OpenAI famously switched from nonprofit to capped-profit). The parent type Organization is stable. The subtype isn't. Storing the subtype means maintaining classification logic, handling ambiguity, and supporting reclassification. That's real engineering cost for marginal query value -- most queries don't filter by organization subtype.

### Place -- coordinates are optional

Most Place mentions on the web are text: "San Francisco," "Tokyo," "the European Union." Not coordinates. The `geo` property (GeoCoordinates: latitude/longitude) is optional because geocoding is a separate pipeline with its own error modes. "Cambridge" could be Massachusetts or England. Resolving that requires context we may not have at extraction time.

The `containedInPlace` hierarchy -- San Francisco -> California -> United States -- is where the real structure lives. It lets you query "all organizations located in California" without knowing every city in the state.

### Topic -- the rebel type

Topic is the only entity type we invented. It doesn't exist in Schema.org. The closest thing in the Semantic Web ecosystem is SKOS (Simple Knowledge Organization System), which defines `skos:Concept` with `broader` and `narrower` hierarchy.

We need Topic because "machine learning" isn't a Person, Organization, Place, or Product -- but it's one of the most connected concepts on the web. Thousands of pages are *about* machine learning. Hundreds of people work *in* machine learning. Dozens of products *use* machine learning. Without a Topic type, all those connections collapse into string matches on page text.

<pre><code><span style="color:#888">// Topic hierarchy, SKOS-style</span>
{
  <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"oi:Topic"</span>,
  <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Machine Learning"</span>,
  <span style="color:#60a5fa">"broader"</span>: { <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Artificial Intelligence"</span> },
  <span style="color:#60a5fa">"narrower"</span>: [
    { <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Deep Learning"</span> },
    { <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Reinforcement Learning"</span> },
    { <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Natural Language Processing"</span> }
  ],
  <span style="color:#60a5fa">"sameAs"</span>: [<span style="color:#4ade80">"https://www.wikidata.org/wiki/Q2539"</span>]
}</code></pre>

The `broader`/`narrower` hierarchy means a query for pages about "Artificial Intelligence" can optionally include pages about all child topics. Wikidata's `sameAs` link grounds the concept in an external knowledge base, which helps with disambiguation -- "Python" the topic (programming language) vs. "Python" the topic (snake) resolve to different Wikidata IDs.

### Product -- dual Schema.org mapping

"React" is software. "iPhone" is hardware. Both are products. Schema.org has separate types for these: `SoftwareApplication` and `Product`. We use a single Product type that maps to whichever Schema.org superclass applies.

The `manufacturer` property points to an Organization, connecting the product graph to the company graph. That edge -- `oi:createdBy` -- turns out to be one of the most useful in the entire ontology. "Find all products created by Google" is a trivial graph traversal when the edge exists, and nearly impossible from text search alone.

## Relationship design: why explicit inverses

Seven relationships connect the six entity types. Every relationship has a named inverse:

<table>
  <thead>
    <tr>
      <th>Relationship</th>
      <th>Domain</th>
      <th>Range</th>
      <th>Inverse</th>
      <th>Why Stored</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>oi:mentions</code></td>
      <td>WebPage</td>
      <td>Any</td>
      <td><code>oi:mentionedIn</code></td>
      <td>Page -> Entity traversal</td>
    </tr>
    <tr>
      <td><code>oi:affiliatedWith</code></td>
      <td>Person</td>
      <td>Organization</td>
      <td><code>oi:hasAffiliate</code></td>
      <td>"Who works at X?"</td>
    </tr>
    <tr>
      <td><code>oi:locatedIn</code></td>
      <td>Org / Event</td>
      <td>Place</td>
      <td><code>oi:locationOf</code></td>
      <td>"What's in SF?"</td>
    </tr>
    <tr>
      <td><code>oi:createdBy</code></td>
      <td>Product</td>
      <td>Person / Org</td>
      <td><code>oi:created</code></td>
      <td>"What did Google build?"</td>
    </tr>
    <tr>
      <td><code>oi:about</code></td>
      <td>WebPage</td>
      <td>Topic</td>
      <td><code>oi:topicOf</code></td>
      <td>"Pages about ML"</td>
    </tr>
    <tr>
      <td><code>oi:linksTo</code></td>
      <td>WebPage</td>
      <td>WebPage</td>
      <td><code>oi:linkedFrom</code></td>
      <td>Web topology</td>
    </tr>
    <tr>
      <td><code>schema:sameAs</code></td>
      <td>Any</td>
      <td>Any</td>
      <td><em>symmetric</em></td>
      <td>Entity deduplication</td>
    </tr>
  </tbody>
</table>

Why store inverses explicitly instead of computing them at query time? Query performance. "Find all pages that mention OpenAI" requires scanning every `oi:mentions` triple where the object is OpenAI. With an explicit `oi:mentionedIn` triple pointing from OpenAI back to each page, it's a direct lookup. At tens of millions of triples, that difference is the gap between sub-second and multi-second queries.

`schema:sameAs` is the exception -- it's symmetric, so it's its own inverse. If entity A `sameAs` entity B, then B `sameAs` A. We store it once and treat it as bidirectional.

## What we deliberately left out

The rejected types and properties are as important as the accepted ones. Each represents a deliberate decision, not an oversight.

**Event.** Tempting. Schema.org has a rich Event type. But events are inherently temporal: "PyCon 2025" expires. "WWDC 2024" is over. Our index is cumulative, not temporal -- we don't have lifecycle management, archival policies, or "this entity is no longer current" semantics. Adding Event means building a temporal reasoning layer we don't need yet.

**Temporal properties.** No `startDate`, `endDate`, `foundingDate` on any type. These change. They conflict across sources. Wikipedia says one founding date, Crunchbase says another, the company's own About page says a third. Resolving conflicts requires provenance-weighted voting, which is a research project, not a schema decision. We'll add temporal properties when we have the resolution logic to back them up.

**Numeric properties.** No revenue, population, stock price, employee count. These are facts, not entities. They change daily. They belong in a time-series database or a facts table, not a knowledge graph. The knowledge graph answers "what is connected to what." A facts table answers "what is the current value of X."

**Fine-grained subtypes.** No Corporation vs. NGO. No City vs. Country vs. Region. No SoftwareApplication vs. PhysicalProduct. Flat types, rich relationships. The subtype can always be added as a property later if it turns out to matter. Removing a subtype from the schema is much harder than adding one.

## JSON Schema: entity validation

Every entity entering the knowledge graph gets validated against a JSON Schema. The `@type` field acts as a discriminator -- it determines which properties are valid:

<pre><code>{
  <span style="color:#60a5fa">"$schema"</span>: <span style="color:#4ade80">"https://json-schema.org/draft/2020-12/schema"</span>,
  <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"object"</span>,
  <span style="color:#60a5fa">"required"</span>: [<span style="color:#4ade80">"@type"</span>, <span style="color:#4ade80">"name"</span>],
  <span style="color:#60a5fa">"discriminator"</span>: { <span style="color:#60a5fa">"propertyName"</span>: <span style="color:#4ade80">"@type"</span> },
  <span style="color:#60a5fa">"oneOf"</span>: [
    {
      <span style="color:#60a5fa">"properties"</span>: {
        <span style="color:#60a5fa">"@type"</span>: { <span style="color:#60a5fa">"const"</span>: <span style="color:#4ade80">"WebPage"</span> },
        <span style="color:#60a5fa">"url"</span>:   { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>, <span style="color:#60a5fa">"format"</span>: <span style="color:#4ade80">"uri"</span> },
        <span style="color:#60a5fa">"name"</span>:  { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
        <span style="color:#60a5fa">"inLanguage"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
        <span style="color:#60a5fa">"oi:fetchTime"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>, <span style="color:#60a5fa">"format"</span>: <span style="color:#4ade80">"date-time"</span> },
        <span style="color:#60a5fa">"oi:crawlId"</span>:  { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> }
      }
    },
    {
      <span style="color:#60a5fa">"properties"</span>: {
        <span style="color:#60a5fa">"@type"</span>: { <span style="color:#60a5fa">"const"</span>: <span style="color:#4ade80">"Person"</span> },
        <span style="color:#60a5fa">"name"</span>:  { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
        <span style="color:#60a5fa">"alternateName"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"array"</span>, <span style="color:#60a5fa">"items"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> } },
        <span style="color:#60a5fa">"affiliation"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
        <span style="color:#60a5fa">"sameAs"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"array"</span>, <span style="color:#60a5fa">"items"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>, <span style="color:#60a5fa">"format"</span>: <span style="color:#4ade80">"uri"</span> } }
      }
    },
    {
      <span style="color:#60a5fa">"properties"</span>: {
        <span style="color:#60a5fa">"@type"</span>: { <span style="color:#60a5fa">"const"</span>: <span style="color:#4ade80">"oi:Topic"</span> },
        <span style="color:#60a5fa">"name"</span>:  { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
        <span style="color:#60a5fa">"broader"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
        <span style="color:#60a5fa">"narrower"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"array"</span>, <span style="color:#60a5fa">"items"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> } },
        <span style="color:#60a5fa">"sameAs"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"array"</span>, <span style="color:#60a5fa">"items"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>, <span style="color:#60a5fa">"format"</span>: <span style="color:#4ade80">"uri"</span> } }
      }
    }
    <span style="color:#888">// ... Organization, Place, Product omitted for brevity</span>
  ]
}</code></pre>

The discriminated union pattern means a single schema validates all six types. `@type` is the switch. If you send a Person with a `geo` field, validation fails. If you send a Topic without a `name`, validation fails. The schema enforces the ontology at the API boundary before anything reaches the triple store.

## DuckDB storage: entities + triples

Two tables. Entities hold properties. Triples hold relationships. This is a relational encoding of a graph -- not the prettiest thing in the world, but it works with the DuckDB infrastructure we already have everywhere.

<pre><code><span style="color:#888">-- Entity table: one row per entity</span>
<span style="color:#60a5fa">CREATE TABLE</span> entities (
  id            <span style="color:#fbbf24">VARCHAR PRIMARY KEY</span>,  <span style="color:#888">-- deterministic hash of (type, name)</span>
  type          <span style="color:#fbbf24">VARCHAR NOT NULL</span>,      <span style="color:#888">-- WebPage, Person, Organization, ...</span>
  name          <span style="color:#fbbf24">VARCHAR NOT NULL</span>,
  properties    <span style="color:#fbbf24">JSON</span>,                  <span style="color:#888">-- type-specific fields as JSON</span>
  same_as       <span style="color:#fbbf24">VARCHAR[]</span>,              <span style="color:#888">-- external URIs (Wikidata, etc.)</span>
  confidence    <span style="color:#fbbf24">FLOAT DEFAULT 1.0</span>,     <span style="color:#888">-- 1.0 for JSON-LD, lower for NER</span>
  source_url    <span style="color:#fbbf24">VARCHAR</span>,               <span style="color:#888">-- page that sourced this entity</span>
  created_at    <span style="color:#fbbf24">TIMESTAMP DEFAULT now()</span>
);

<span style="color:#888">-- Triple table: one row per relationship</span>
<span style="color:#60a5fa">CREATE TABLE</span> triples (
  subject_id    <span style="color:#fbbf24">VARCHAR NOT NULL</span>,      <span style="color:#888">-- FK to entities.id</span>
  predicate     <span style="color:#fbbf24">VARCHAR NOT NULL</span>,      <span style="color:#888">-- oi:mentions, oi:affiliatedWith, ...</span>
  object_id     <span style="color:#fbbf24">VARCHAR NOT NULL</span>,      <span style="color:#888">-- FK to entities.id</span>
  source_url    <span style="color:#fbbf24">VARCHAR</span>,               <span style="color:#888">-- provenance</span>
  confidence    <span style="color:#fbbf24">FLOAT DEFAULT 1.0</span>,
  created_at    <span style="color:#fbbf24">TIMESTAMP DEFAULT now()</span>
);

<span style="color:#888">-- Index for both traversal directions</span>
<span style="color:#60a5fa">CREATE INDEX</span> idx_triples_subject <span style="color:#60a5fa">ON</span> triples(subject_id, predicate);
<span style="color:#60a5fa">CREATE INDEX</span> idx_triples_object  <span style="color:#60a5fa">ON</span> triples(object_id, predicate);</code></pre>

Why two tables instead of encoding everything as triples? Properties are attributes of an entity (name, url, geo coordinates). Relationships connect two entities. Mixing them in a single triple table means "the name of entity X is 'OpenAI'" sits next to "entity X is located in San Francisco." The first is a property lookup, the second is a graph traversal. They have different access patterns and different indexing needs.

The `properties` column uses DuckDB's native JSON type. Type-specific fields -- `alternateName[]` for Person, `geo` for Place, `broader` for Topic -- live there instead of as top-level columns. This avoids a sparse table with 20 nullable columns where most rows only use 3.

<div class="note">
  <strong>Explicit inverse storage.</strong> When we insert <code>oi:mentions(page_123, openai)</code>, we also insert <code>oi:mentionedIn(openai, page_123)</code>. Two rows per relationship. It doubles the triple count but makes reverse traversals a single index lookup instead of a full scan. At web scale, that trade-off isn't even close.
</div>

## The thousand arguments

We had (and continue to have) arguments about every decision on this page. Should CreativeWork be a separate type? Should we add Event for the 5% of pages that have them? Is flat Organization going to bite us when someone needs to distinguish universities from companies?

Maybe. Probably. The ontology isn't frozen -- it's version 1. But the cost of adding a type later is low (new rows in the entities table, new predicates in triples). The cost of removing a type is high (migration, breaking queries, orphaned triples). So we're starting small and opinionated, with a clear path to expand.

Six types. Seven relationships. Everything connects back to a WebPage. Everything has provenance. Every edge has an inverse. It's less than what the world contains, and exactly enough to be useful.
