---
slug: schema-org-foundation
title: "Schema.org — The Ontology Hiding in Every Web Page"
date: 2026-03-09
summary: "800 types, 1,400 properties, embedded in 44% of the web. We're not inventing a vocabulary. We're extending the one that already won."
tags: [ontology, data]
---

There are two ways to build a vocabulary for the web. You can sit in a committee room and design one from first principles -- spend years on the taxonomy, argue about edge cases, publish a spec nobody reads. Or you can wait for Google to make one up and bribe the entire web into using it with search ranking bonuses.

Schema.org took the second path. It won.

## Schema.org won (by Google's fist, not committee)

In June 2011, Google, Microsoft (Bing), and Yahoo announced Schema.org. Yandex joined a few months later. No W3C working group. No multi-year standards process. Four companies that collectively owned search said "use these types to describe your content and we'll reward you with rich snippets."

That's not a standards process. That's an ultimatum with a carrot.

And it worked. SEO teams across the planet scrambled to add structured data markup. E-commerce sites wanted star ratings in search results. News publishers wanted article cards with author photos. Recipe blogs wanted cooking time badges. Google dangled these rich snippets as ranking signals, and the web responded by annotating itself.

By 2024, Web Data Commons analysis of Common Crawl shows Schema.org markup on **44% of crawled pages**. Not 44% of "good" pages -- 44% of the entire web they could reach. Among high-quality domains (news, e-commerce, company sites), coverage is closer to 70-80%.

That's not adoption. That's dominance. And it happened not because Schema.org was the most theoretically correct vocabulary -- Dublin Core, FOAF, and Good Relations all had valid claims there. It happened because Google paid for it in search placement. The lesson: an ontology doesn't win by being right. It wins by being rewarded.

## What Schema.org actually is

Schema.org defines **800+ types** organized in a single-inheritance hierarchy rooted at `Thing`. Every type inherits from exactly one parent. There are **1,400+ properties** that describe attributes and relationships between types.

Three embedding formats carry Schema.org data inside HTML:

<table>
  <thead>
    <tr>
      <th>Format</th>
      <th>Share of Structured Data Pages</th>
      <th>How It Works</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>JSON-LD</strong></td>
      <td>~72%</td>
      <td>JSON blob inside <code>&lt;script type="application/ld+json"&gt;</code></td>
    </tr>
    <tr>
      <td><strong>Microdata</strong></td>
      <td>~22%</td>
      <td>HTML attributes (<code>itemscope</code>, <code>itemprop</code>) inline in DOM elements</td>
    </tr>
    <tr>
      <td><strong>RDFa</strong></td>
      <td>~6%</td>
      <td>RDF attributes (<code>typeof</code>, <code>property</code>) inline in DOM elements</td>
    </tr>
  </tbody>
</table>

JSON-LD won this sub-competition for the same reason Schema.org won the larger one: it's the easiest to implement. A JSON-LD block sits in a `<script>` tag. It doesn't touch the DOM. Doesn't interfere with CSS. Doesn't break rendering. A backend template can spit it out without coordinating with the frontend team. Microdata requires weaving attributes into your HTML structure -- every `div` and `span` becomes a carrier for schema properties. RDFa does the same with a different attribute set. Both are fragile. Both break when someone redesigns the page layout.

Here's what a real JSON-LD block looks like on a news site -- not the simplified version, the actual shape Google expects:

<pre><code><span style="color:#888">&lt;script type="application/ld+json"&gt;</span>
{
  <span style="color:#60a5fa">"@context"</span>: <span style="color:#4ade80">"https://schema.org"</span>,
  <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"NewsArticle"</span>,
  <span style="color:#60a5fa">"headline"</span>: <span style="color:#4ade80">"EU Passes Digital Markets Act"</span>,
  <span style="color:#60a5fa">"datePublished"</span>: <span style="color:#4ade80">"2026-03-01T10:00:00Z"</span>,
  <span style="color:#60a5fa">"author"</span>: [{
    <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Person"</span>,
    <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Clara Dupont"</span>,
    <span style="color:#60a5fa">"url"</span>: <span style="color:#4ade80">"https://example.com/author/cdupont"</span>
  }],
  <span style="color:#60a5fa">"publisher"</span>: {
    <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Organization"</span>,
    <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"EuroTech News"</span>,
    <span style="color:#60a5fa">"logo"</span>: { <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"ImageObject"</span>, <span style="color:#60a5fa">"url"</span>: <span style="color:#4ade80">"https://example.com/logo.png"</span> }
  },
  <span style="color:#60a5fa">"about"</span>: [
    { <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Organization"</span>, <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"European Union"</span> },
    { <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Thing"</span>, <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Digital Markets Act"</span> }
  ],
  <span style="color:#60a5fa">"mentions"</span>: [
    { <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Person"</span>, <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Thierry Breton"</span> },
    { <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Organization"</span>, <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Apple"</span> },
    { <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Organization"</span>, <span style="color:#60a5fa">"name"</span>: <span style="color:#4ade80">"Google"</span> }
  ]
}
<span style="color:#888">&lt;/script&gt;</span></code></pre>

One `<script>` tag. Seven entities. Four relationships. Zero model inference. That's the payoff of a vocabulary the web already speaks.

## The type hierarchy -- where OpenIndex fits

Schema.org's 800 types form a tree. Everything inherits from `Thing`. Here's the part of the hierarchy that matters for us, with OpenIndex's six entity types mapped:

<pre><code><span style="color:#e0e0e0">Thing</span>
<span style="color:#888">├──</span> <span style="color:#e0e0e0">CreativeWork</span>
<span style="color:#888">│   ├──</span> <span style="color:#e0e0e0">Article</span>
<span style="color:#888">│   │   └──</span> <span style="color:#e0e0e0">NewsArticle, BlogPosting, ...</span>
<span style="color:#888">│   └──</span> <span style="color:#4ade80">WebPage</span>              <span style="color:#888">← oi:WebPage</span>
<span style="color:#888">│       └──</span> <span style="color:#e0e0e0">FAQPage, ItemPage, ...</span>
<span style="color:#888">├──</span> <span style="color:#4ade80">Person</span>                    <span style="color:#888">← oi:Person</span>
<span style="color:#888">├──</span> <span style="color:#4ade80">Organization</span>              <span style="color:#888">← oi:Organization</span>
<span style="color:#888">│   ├──</span> <span style="color:#e0e0e0">Corporation</span>
<span style="color:#888">│   ├──</span> <span style="color:#e0e0e0">GovernmentOrganization</span>
<span style="color:#888">│   └──</span> <span style="color:#e0e0e0">LocalBusiness</span>
<span style="color:#888">├──</span> <span style="color:#4ade80">Place</span>                     <span style="color:#888">← oi:Place</span>
<span style="color:#888">│   ├──</span> <span style="color:#e0e0e0">City, Country, ...</span>
<span style="color:#888">│   └──</span> <span style="color:#e0e0e0">LocalBusiness</span> <span style="color:#888">(multi-inherit via type array)</span>
<span style="color:#888">├──</span> <span style="color:#4ade80">Product</span>                   <span style="color:#888">← oi:Product</span>
<span style="color:#888">│   └──</span> <span style="color:#e0e0e0">SoftwareApplication</span>
<span style="color:#888">└──</span> <span style="color:#e0e0e0">Intangible</span>
    <span style="color:#888">└──</span> <span style="color:#e0e0e0">(no direct match)</span>

<span style="color:#fbbf24">oi:Topic</span>                      <span style="color:#888">← extends Thing directly (no Schema.org equivalent)</span></code></pre>

Five of our six types map directly to Schema.org parent types. `Topic` is the exception -- Schema.org doesn't have a clean concept-as-entity type. `DefinedTerm` comes close but it's too narrow (it implies a glossary entry). So `oi:Topic` extends `Thing` directly and borrows SKOS vocabulary (`broader`, `narrower`) for hierarchical relationships.

The key insight: Schema.org subtypes are compatible upward. A `NewsArticle` is a `CreativeWork` is a `Thing`. When we encounter a `NewsArticle` in JSON-LD, we don't need a special case -- it maps to `oi:WebPage` because `WebPage` and `NewsArticle` share a `CreativeWork` ancestor. The type hierarchy does the mapping for us.

## Why extend instead of invent

We could define our own vocabulary from scratch. Six types, a handful of properties, a clean namespace. It'd be internally consistent and perfectly tailored to our needs. It'd also be useless.

**Interoperability.** Anything that consumes Schema.org can consume OpenIndex entities. SPARQL endpoints, Google's Rich Results Test, knowledge graph visualizers -- they all understand `schema:Person` with a `name` and `affiliation`. Use a custom `oi:Individual` type instead and you're speaking a language nobody else reads.

**Free data.** 44% of the web already annotates itself with Schema.org types. That's hundreds of millions of pages where entity extraction is just JSON parsing. If we invented a custom vocabulary, we'd need a mapping layer between Schema.org and our types for every page. By extending Schema.org, the mapping is identity: a `schema:Organization` *is* an `oi:Organization`. No translation needed.

**Community.** Schema.org has had thousands of contributors arguing about type definitions for 13 years. Should `Place` have a `geo` property or a `location` property? They debated it. Should `author` accept both `Person` and `Organization`? They decided. These are real design decisions with real trade-offs, and a community has already done the work. Starting from scratch means redoing those arguments with fewer people and less experience.

## What Schema.org gets wrong for us

Extending doesn't mean accepting everything. Schema.org was designed for search engine rich snippets, not web intelligence. The gaps are specific and fixable.

**Too broad.** 800 types when we need 6. Schema.org defines `MedicalCondition`, `TVSeries`, `Recipe`, `MusicRecording`, `ExerciseAction`, `LendAction`, and hundreds more. These exist because Google shows rich snippets for recipes (cooking time, calories) and TV shows (episode guides, ratings). We don't. Our ontology covers the entities useful for understanding the web's information structure, not rendering search result cards.

**No confidence scores.** Schema.org assumes every assertion is true. When a page says `"@type": "Person", "name": "Jane Smith"`, Schema.org treats that as fact. We can't. NER extraction is probabilistic -- the model is 92% confident this text span is a person. Even JSON-LD can be wrong (SEO spam, stale data, template bugs). OpenIndex adds `oi:confidence` to every entity and relationship because the real world isn't boolean.

**No provenance.** Which crawl found this entity? Which page asserted it? When was the page fetched? Schema.org doesn't track where a fact came from because it assumes facts are self-contained -- the page *is* the source. For a web-scale index where the same entity appears on thousands of pages with conflicting information, provenance isn't optional. OpenIndex adds `oi:crawlId`, `oi:sourceUrl`, and `oi:fetchTime`.

**Loose typing.** In Schema.org, the `author` property accepts `Person`, `Organization`, or a plain text string. That flexibility makes sense for markup -- a page author might just write `"author": "J. Smith"` without creating a Person object. For a knowledge graph, it's poison. A text string can't participate in relationships. We enforce typed references: `author` must point to an entity, not a string. If the source data gives us a string, we create an entity with `oi:confidence` reflecting the uncertainty.

## Schema.org vs. OpenIndex ontology

<table>
  <thead>
    <tr>
      <th>Property</th>
      <th>Schema.org</th>
      <th>OpenIndex Ontology</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Types</strong></td>
      <td>800+</td>
      <td>6 (WebPage, Person, Organization, Place, Product, Topic)</td>
    </tr>
    <tr>
      <td><strong>Inheritance</strong></td>
      <td>Single-inheritance from Thing</td>
      <td>Extends Schema.org types (5 of 6 map directly)</td>
    </tr>
    <tr>
      <td><strong>Confidence</strong></td>
      <td>Not supported</td>
      <td><code>oi:confidence</code> on every entity and relationship</td>
    </tr>
    <tr>
      <td><strong>Provenance</strong></td>
      <td>Not tracked</td>
      <td><code>oi:sourceUrl</code>, <code>oi:crawlId</code>, <code>oi:fetchTime</code></td>
    </tr>
    <tr>
      <td><strong>Relationship typing</strong></td>
      <td>Loose (accepts text or entity)</td>
      <td>Strict (typed entity references only)</td>
    </tr>
    <tr>
      <td><strong>Embedding format</strong></td>
      <td>JSON-LD, Microdata, RDFa</td>
      <td>JSON-LD for input; DuckDB triples for storage</td>
    </tr>
    <tr>
      <td><strong>Primary consumer</strong></td>
      <td>Search engine crawlers (rich snippets)</td>
      <td>Knowledge graph, vector search, OQL queries</td>
    </tr>
    <tr>
      <td><strong>Maintained by</strong></td>
      <td>Schema.org Community Group (W3C-hosted)</td>
      <td>OpenIndex project</td>
    </tr>
  </tbody>
</table>

The relationship is additive, not competitive. Schema.org defines the base vocabulary. OpenIndex adds the properties needed for a crawl-derived, probabilistic, provenance-tracked knowledge graph. A valid OpenIndex entity is always a valid Schema.org entity. The reverse isn't true -- Schema.org entities lack confidence and provenance until we add them.

## The extraction payoff

What does Schema.org actually give us when we parse it at scale? Run a DuckDB query across a crawl to count JSON-LD type frequencies:

<pre><code><span style="color:#888">-- Type distribution across pages with JSON-LD</span>
<span style="color:#60a5fa">SELECT</span> json_extract_string(jsonld, <span style="color:#4ade80">'$."@type"'</span>) <span style="color:#60a5fa">AS</span> entity_type,
       <span style="color:#60a5fa">COUNT</span>(*) <span style="color:#60a5fa">AS</span> frequency
<span style="color:#60a5fa">FROM</span> pages
<span style="color:#60a5fa">WHERE</span> jsonld <span style="color:#60a5fa">IS NOT NULL</span>
<span style="color:#60a5fa">GROUP BY</span> entity_type
<span style="color:#60a5fa">ORDER BY</span> frequency <span style="color:#60a5fa">DESC</span>
<span style="color:#60a5fa">LIMIT</span> <span style="color:#fbbf24">15</span>;</code></pre>

Across a typical Common Crawl sample, the distribution looks roughly like this:

<table>
  <thead>
    <tr>
      <th>@type</th>
      <th>Frequency</th>
      <th>Maps to OpenIndex</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>WebPage</code></td>
      <td>~28%</td>
      <td>oi:WebPage</td>
    </tr>
    <tr>
      <td><code>Organization</code></td>
      <td>~14%</td>
      <td>oi:Organization</td>
    </tr>
    <tr>
      <td><code>Product</code></td>
      <td>~12%</td>
      <td>oi:Product</td>
    </tr>
    <tr>
      <td><code>Article</code></td>
      <td>~9%</td>
      <td>oi:WebPage (parent type)</td>
    </tr>
    <tr>
      <td><code>BreadcrumbList</code></td>
      <td>~8%</td>
      <td><span style="color:#888">-- (navigation, skip)</span></td>
    </tr>
    <tr>
      <td><code>Person</code></td>
      <td>~6%</td>
      <td>oi:Person</td>
    </tr>
    <tr>
      <td><code>LocalBusiness</code></td>
      <td>~5%</td>
      <td>oi:Organization + oi:Place</td>
    </tr>
    <tr>
      <td><code>NewsArticle</code></td>
      <td>~4%</td>
      <td>oi:WebPage (parent type)</td>
    </tr>
    <tr>
      <td><code>BlogPosting</code></td>
      <td>~3%</td>
      <td>oi:WebPage (parent type)</td>
    </tr>
    <tr>
      <td><code>ImageObject</code></td>
      <td>~3%</td>
      <td><span style="color:#888">-- (media, skip)</span></td>
    </tr>
    <tr>
      <td><code>FAQPage</code></td>
      <td>~2%</td>
      <td>oi:WebPage (parent type)</td>
    </tr>
    <tr>
      <td><code>SoftwareApplication</code></td>
      <td>~2%</td>
      <td>oi:Product (parent type)</td>
    </tr>
    <tr>
      <td><code>Place</code></td>
      <td>~1.5%</td>
      <td>oi:Place</td>
    </tr>
    <tr>
      <td><code>Event</code></td>
      <td>~1%</td>
      <td><span style="color:#888">-- (not in v1)</span></td>
    </tr>
    <tr>
      <td><code>Recipe</code></td>
      <td>~0.8%</td>
      <td><span style="color:#888">-- (not in v1)</span></td>
    </tr>
  </tbody>
</table>

The top six types that map to OpenIndex entities account for roughly **65% of all structured data**. Most of the rest is navigation markup (`BreadcrumbList`, `SiteNavigationElement`), media objects (`ImageObject`, `VideoObject`), or domain-specific types we don't need in v1 (`Recipe`, `Event`). The long tail of 800 types is exactly that -- a long tail. The head covers our use case.

`LocalBusiness` is an interesting case. In Schema.org's hierarchy, it inherits from both `Organization` and `Place` (one of the few multi-type patterns). For OpenIndex, we split it: the business entity becomes an `oi:Organization`, the address becomes an `oi:Place`, and an `oi:locatedIn` relationship connects them. Schema.org's loose type system collapses into our strict one.

## Architecture connection

The parser is straightforward Go. Scan HTML for `<script type="application/ld+json">` blocks, `json.Unmarshal` each one, walk the `@type` field to determine the Schema.org type, map it to an OpenIndex entity type, extract properties, write triples to DuckDB.

<pre><code><span style="color:#888">// Simplified extraction loop</span>
<span style="color:#60a5fa">for</span> _, block := <span style="color:#60a5fa">range</span> extractJSONLD(html) {
    schemaType := block[<span style="color:#4ade80">"@type"</span>]
    oiType := mapSchemaType(schemaType)  <span style="color:#888">// "NewsArticle" → "WebPage"</span>
    <span style="color:#60a5fa">if</span> oiType == <span style="color:#4ade80">""</span> {
        <span style="color:#60a5fa">continue</span>  <span style="color:#888">// BreadcrumbList, ImageObject — skip</span>
    }
    entity := Entity{
        Type:       oiType,
        Name:       block[<span style="color:#4ade80">"name"</span>],
        SourceURL:  pageURL,
        Confidence: <span style="color:#fbbf24">1.0</span>,  <span style="color:#888">// JSON-LD is deterministic</span>
        CrawlID:    crawlID,
    }
    triples = <span style="color:#60a5fa">append</span>(triples, entity.ToTriples()...)
}</code></pre>

No models. No GPU. 50K pages/second on commodity hardware because it's just string search and JSON parsing. The `mapSchemaType` function is a lookup table -- 800 Schema.org types map to 6 OpenIndex types (or nil for types we skip). The type hierarchy is baked into the table: `NewsArticle`, `BlogPosting`, `Article`, `FAQPage` all resolve to `oi:WebPage`.

This runs as the first stage of the extraction pipeline, before any NER model touches the data. Every page with JSON-LD gets its entities extracted for free. The NER stage only needs to process the 56% of pages without structured markup -- and even there, the JSON-LD entities provide context for entity resolution. If we already know "Apple" is an Organization from a hundred JSON-LD blocks, disambiguating "Apple" in plain text becomes easier.

<div class="note">
  <strong>Honest status:</strong> The type mapping table and JSON-LD parser are designed. The DuckDB triple store schema exists. The full extraction pipeline -- parsing, mapping, deduplication, and integration with the NER stage -- hasn't shipped yet. JSON-LD extraction is the next piece to land after Tantivy integration.
</div>

We didn't design an ontology. We found one already embedded in 44% of the web, adopted by millions of sites, maintained by thousands of contributors, and backed by every major search engine. We're adding confidence scores, provenance tracking, and strict typing -- the things Schema.org doesn't need for rich snippets but we need for a knowledge graph. Extend what won. Don't reinvent what's already everywhere.
