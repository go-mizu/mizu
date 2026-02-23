---
slug: rdf-turtle-owl
title: "RDF, Turtle, OWL — Making the Ontology Portable"
date: 2026-03-11
summary: "Four serialization formats for one ontology. JSON-LD for APIs, Turtle for humans, OWL for reasoners, JSON Schema for validation."
tags: [ontology, data]
---

Internally, everything lives in DuckDB. The knowledge graph is triples in a sharded database. The entity store is rows. The crawl results are rows. DuckDB is great for us -- fast, embedded, SQL-native.

It's useless for everyone else.

If someone wants to load OpenIndex entities into Protege, hook up a SPARQL endpoint, or merge our data into their own knowledge graph, they need standard formats. Not "here's a DuckDB file, good luck." Standard, published, decades-old formats that every semantic web tool on the planet already understands.

Four of them, specifically.

## The Semantic Web layer cake (the useful parts)

The Semantic Web vision was grandiose. Tim Berners-Lee described a web where machines reason over interlinked data, autonomous agents negotiate on your behalf, and ontologies provide universal meaning. Most of that didn't happen. But the technology stack it produced has genuinely useful layers if you're selective about what you adopt.

**RDF** is the data model. Everything is triples: subject, predicate, object. A page mentions an organization. A person is affiliated with a company. That's it -- three things connected by a relationship. The model doesn't care how you serialize it.

**Serialization formats** are how you write triples down. Turtle is human-readable and compact. JSON-LD is web-native -- it's JSON with a `@context` that maps keys to URIs. RDF/XML is legacy (avoid). N-Triples is line-oriented and good for streaming.

**RDFS** adds schema. Subclass relationships, domain/range constraints. "Person is a subclass of Thing." "The domain of `affiliatedWith` is Person." Light, practical, not much to argue with.

**OWL** is the logic layer. Formal axioms, inverse properties, class restrictions, reasoning. Powerful but heavy. You can declare that `oi:mentions` has an inverse `oi:mentionedIn`, and a reasoner will infer the reverse triple automatically. Whether you actually *need* a reasoner depends on your use case.

Here's where we stand: we use the bottom layers (RDF, Turtle, JSON-LD) heavily and the top layers (OWL reasoning) lightly. The ontology is defined with enough OWL to be useful in Protege and compatible with reasoners, but we aren't building an inference engine. The DuckDB triple store handles queries directly.

## Full Turtle definition of the OpenIndex ontology

This is the canonical definition. Everything else -- JSON-LD context, OWL axioms, JSON Schema -- derives from this.

<pre><code><span style="color:#888"># ── OpenIndex Ontology v0.1 ──────────────────────────────</span>
<span style="color:#888"># Canonical source: https://openindex.org/ontology/</span>

<span style="color:#60a5fa">@prefix</span> oi:     <span style="color:#4ade80">&lt;https://openindex.org/ontology/&gt;</span> .
<span style="color:#60a5fa">@prefix</span> schema: <span style="color:#4ade80">&lt;https://schema.org/&gt;</span> .
<span style="color:#60a5fa">@prefix</span> rdfs:   <span style="color:#4ade80">&lt;http://www.w3.org/2000/01/rdf-schema#&gt;</span> .
<span style="color:#60a5fa">@prefix</span> owl:    <span style="color:#4ade80">&lt;http://www.w3.org/2002/07/owl#&gt;</span> .
<span style="color:#60a5fa">@prefix</span> xsd:    <span style="color:#4ade80">&lt;http://www.w3.org/2001/XMLSchema#&gt;</span> .
<span style="color:#60a5fa">@prefix</span> skos:   <span style="color:#4ade80">&lt;http://www.w3.org/2004/02/skos/core#&gt;</span> .

<span style="color:#888"># ── Ontology metadata ────────────────────────────────────</span>
<span style="color:#4ade80">&lt;https://openindex.org/ontology/&gt;</span>
    <span style="color:#60a5fa">a</span>               owl:Ontology ;
    rdfs:label      <span style="color:#fbbf24">"OpenIndex Ontology"</span> ;
    rdfs:comment    <span style="color:#fbbf24">"Entity types and relationships for web knowledge graphs."</span> ;
    owl:versionInfo <span style="color:#fbbf24">"0.1"</span> .

<span style="color:#888"># ── Entity classes ────────────────────────────────────────</span>
oi:WebPage <span style="color:#60a5fa">a</span> owl:Class ;
    rdfs:subClassOf schema:WebPage ;
    rdfs:label      <span style="color:#fbbf24">"Web Page"</span> ;
    rdfs:comment    <span style="color:#fbbf24">"A crawled web page with metadata."</span> .

oi:Person <span style="color:#60a5fa">a</span> owl:Class ;
    rdfs:subClassOf schema:Person ;
    rdfs:label      <span style="color:#fbbf24">"Person"</span> ;
    rdfs:comment    <span style="color:#fbbf24">"A named individual identified in web content."</span> .

oi:Organization <span style="color:#60a5fa">a</span> owl:Class ;
    rdfs:subClassOf schema:Organization ;
    rdfs:label      <span style="color:#fbbf24">"Organization"</span> ;
    rdfs:comment    <span style="color:#fbbf24">"A company, institution, or NGO."</span> .

oi:Place <span style="color:#60a5fa">a</span> owl:Class ;
    rdfs:subClassOf schema:Place ;
    rdfs:label      <span style="color:#fbbf24">"Place"</span> ;
    rdfs:comment    <span style="color:#fbbf24">"A geographic location."</span> .

oi:Topic <span style="color:#60a5fa">a</span> owl:Class ;
    rdfs:subClassOf schema:Thing ;
    rdfs:subClassOf skos:Concept ;
    rdfs:label      <span style="color:#fbbf24">"Topic"</span> ;
    rdfs:comment    <span style="color:#fbbf24">"A subject or field of study."</span> .

oi:Product <span style="color:#60a5fa">a</span> owl:Class ;
    rdfs:subClassOf schema:Product ;
    rdfs:label      <span style="color:#fbbf24">"Product"</span> ;
    rdfs:comment    <span style="color:#fbbf24">"Software, hardware, or commercial product."</span> .

<span style="color:#888"># ── Relationship properties ───────────────────────────────</span>
oi:mentions <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:domain     oi:WebPage ;
    rdfs:range      owl:Thing ;
    rdfs:label      <span style="color:#fbbf24">"mentions"</span> ;
    owl:inverseOf   oi:mentionedIn .

oi:mentionedIn <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:label      <span style="color:#fbbf24">"mentioned in"</span> .

oi:affiliatedWith <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:domain     oi:Person ;
    rdfs:range      oi:Organization ;
    rdfs:label      <span style="color:#fbbf24">"affiliated with"</span> ;
    owl:inverseOf   oi:hasAffiliate .

oi:hasAffiliate <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:label      <span style="color:#fbbf24">"has affiliate"</span> .

oi:locatedIn <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:domain     owl:Thing ;
    rdfs:range      oi:Place ;
    rdfs:label      <span style="color:#fbbf24">"located in"</span> ;
    owl:inverseOf   oi:locationOf .

oi:locationOf <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:label      <span style="color:#fbbf24">"location of"</span> .

oi:createdBy <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:domain     oi:Product ;
    rdfs:range      owl:Thing ;
    rdfs:label      <span style="color:#fbbf24">"created by"</span> ;
    owl:inverseOf   oi:created .

oi:created <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:label      <span style="color:#fbbf24">"created"</span> .

oi:about <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:domain     oi:WebPage ;
    rdfs:range      oi:Topic ;
    rdfs:label      <span style="color:#fbbf24">"about"</span> ;
    owl:inverseOf   oi:topicOf .

oi:topicOf <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:label      <span style="color:#fbbf24">"topic of"</span> .

oi:linksTo <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:domain     oi:WebPage ;
    rdfs:range      oi:WebPage ;
    rdfs:label      <span style="color:#fbbf24">"links to"</span> ;
    owl:inverseOf   oi:linkedFrom .

oi:linkedFrom <span style="color:#60a5fa">a</span> owl:ObjectProperty ;
    rdfs:label      <span style="color:#fbbf24">"linked from"</span> .

<span style="color:#888"># ── Custom data properties ────────────────────────────────</span>
oi:confidence <span style="color:#60a5fa">a</span> owl:DatatypeProperty ;
    rdfs:domain     owl:Thing ;
    rdfs:range      xsd:float ;
    rdfs:label      <span style="color:#fbbf24">"confidence"</span> ;
    rdfs:comment    <span style="color:#fbbf24">"Extraction confidence score, 0.0 to 1.0."</span> .

oi:fetchTime <span style="color:#60a5fa">a</span> owl:DatatypeProperty ;
    rdfs:domain     oi:WebPage ;
    rdfs:range      xsd:dateTime ;
    rdfs:label      <span style="color:#fbbf24">"fetch time"</span> ;
    rdfs:comment    <span style="color:#fbbf24">"When the page was crawled."</span> .

oi:crawlId <span style="color:#60a5fa">a</span> owl:DatatypeProperty ;
    rdfs:domain     oi:WebPage ;
    rdfs:range      xsd:string ;
    rdfs:label      <span style="color:#fbbf24">"crawl ID"</span> ;
    rdfs:comment    <span style="color:#fbbf24">"Identifier for the crawl run."</span> .</code></pre>

That's roughly 100 triples defining six entity classes, seven relationship pairs (each with an inverse), and three custom data properties. Load this file into Protege and you get a visual class hierarchy. Load it into a SPARQL endpoint and every relationship is queryable. It's the same ontology that lives in our DuckDB triple store, just serialized for the rest of the world.

## JSON-LD context document

JSON-LD is how the ontology travels over HTTP. When an API response includes `"@context": "https://openindex.org/ontology/context.jsonld"`, every key maps to a full URI without cluttering the payload.

Here's the complete context:

<pre><code>{
  <span style="color:#60a5fa">"@context"</span>: {
    <span style="color:#60a5fa">"oi"</span>:          <span style="color:#4ade80">"https://openindex.org/ontology/"</span>,
    <span style="color:#60a5fa">"schema"</span>:      <span style="color:#4ade80">"https://schema.org/"</span>,
    <span style="color:#60a5fa">"skos"</span>:        <span style="color:#4ade80">"http://www.w3.org/2004/02/skos/core#"</span>,

    <span style="color:#60a5fa">"WebPage"</span>:     <span style="color:#4ade80">"oi:WebPage"</span>,
    <span style="color:#60a5fa">"Person"</span>:      <span style="color:#4ade80">"oi:Person"</span>,
    <span style="color:#60a5fa">"Organization"</span>: <span style="color:#4ade80">"oi:Organization"</span>,
    <span style="color:#60a5fa">"Place"</span>:       <span style="color:#4ade80">"oi:Place"</span>,
    <span style="color:#60a5fa">"Topic"</span>:       <span style="color:#4ade80">"oi:Topic"</span>,
    <span style="color:#60a5fa">"Product"</span>:     <span style="color:#4ade80">"oi:Product"</span>,

    <span style="color:#60a5fa">"mentions"</span>:       { <span style="color:#60a5fa">"@id"</span>: <span style="color:#4ade80">"oi:mentions"</span>,       <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"@id"</span> },
    <span style="color:#60a5fa">"affiliatedWith"</span>: { <span style="color:#60a5fa">"@id"</span>: <span style="color:#4ade80">"oi:affiliatedWith"</span>, <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"@id"</span> },
    <span style="color:#60a5fa">"locatedIn"</span>:      { <span style="color:#60a5fa">"@id"</span>: <span style="color:#4ade80">"oi:locatedIn"</span>,      <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"@id"</span> },
    <span style="color:#60a5fa">"createdBy"</span>:      { <span style="color:#60a5fa">"@id"</span>: <span style="color:#4ade80">"oi:createdBy"</span>,      <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"@id"</span> },
    <span style="color:#60a5fa">"about"</span>:          { <span style="color:#60a5fa">"@id"</span>: <span style="color:#4ade80">"oi:about"</span>,          <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"@id"</span> },
    <span style="color:#60a5fa">"linksTo"</span>:        { <span style="color:#60a5fa">"@id"</span>: <span style="color:#4ade80">"oi:linksTo"</span>,        <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"@id"</span> },
    <span style="color:#60a5fa">"sameAs"</span>:         { <span style="color:#60a5fa">"@id"</span>: <span style="color:#4ade80">"schema:sameAs"</span>,     <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"@id"</span> },

    <span style="color:#60a5fa">"confidence"</span>:     { <span style="color:#60a5fa">"@id"</span>: <span style="color:#4ade80">"oi:confidence"</span>,  <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"xsd:float"</span> },
    <span style="color:#60a5fa">"fetchTime"</span>:      { <span style="color:#60a5fa">"@id"</span>: <span style="color:#4ade80">"oi:fetchTime"</span>,   <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"xsd:dateTime"</span> },
    <span style="color:#60a5fa">"crawlId"</span>:        { <span style="color:#60a5fa">"@id"</span>: <span style="color:#4ade80">"oi:crawlId"</span>,     <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"xsd:string"</span> },

    <span style="color:#60a5fa">"name"</span>:           <span style="color:#4ade80">"schema:name"</span>,
    <span style="color:#60a5fa">"url"</span>:            <span style="color:#4ade80">"schema:url"</span>,
    <span style="color:#60a5fa">"description"</span>:    <span style="color:#4ade80">"schema:description"</span>,
    <span style="color:#60a5fa">"inLanguage"</span>:     <span style="color:#4ade80">"schema:inLanguage"</span>,
    <span style="color:#60a5fa">"alternateName"</span>:  <span style="color:#4ade80">"schema:alternateName"</span>,
    <span style="color:#60a5fa">"geo"</span>:            <span style="color:#4ade80">"schema:geo"</span>,
    <span style="color:#60a5fa">"broader"</span>:        <span style="color:#4ade80">"skos:broader"</span>,
    <span style="color:#60a5fa">"narrower"</span>:       <span style="color:#4ade80">"skos:narrower"</span>
  }
}</code></pre>

With that context in place, an API response looks like clean JSON while being fully dereferenceable as linked data:

<pre><code>{
  <span style="color:#60a5fa">"@context"</span>: <span style="color:#4ade80">"https://openindex.org/ontology/context.jsonld"</span>,
  <span style="color:#60a5fa">"@type"</span>:    <span style="color:#4ade80">"Person"</span>,
  <span style="color:#60a5fa">"@id"</span>:      <span style="color:#4ade80">"https://openindex.org/entity/linus-torvalds"</span>,
  <span style="color:#60a5fa">"name"</span>:     <span style="color:#4ade80">"Linus Torvalds"</span>,
  <span style="color:#60a5fa">"affiliatedWith"</span>: {
    <span style="color:#60a5fa">"@type"</span>: <span style="color:#4ade80">"Organization"</span>,
    <span style="color:#60a5fa">"@id"</span>:   <span style="color:#4ade80">"https://openindex.org/entity/linux-foundation"</span>,
    <span style="color:#60a5fa">"name"</span>:  <span style="color:#4ade80">"The Linux Foundation"</span>
  },
  <span style="color:#60a5fa">"sameAs"</span>: [
    <span style="color:#4ade80">"https://www.wikidata.org/wiki/Q34253"</span>,
    <span style="color:#4ade80">"https://dbpedia.org/resource/Linus_Torvalds"</span>
  ],
  <span style="color:#60a5fa">"confidence"</span>: <span style="color:#fbbf24">0.99</span>,
  <span style="color:#60a5fa">"crawlId"</span>:    <span style="color:#4ade80">"OI-2026-03"</span>
}</code></pre>

A regular JSON consumer reads it as a flat object. A linked data consumer resolves `"affiliatedWith"` to `https://openindex.org/ontology/affiliatedWith`, follows the `@id` links, and integrates it into their graph. Same payload, two audiences.

## OWL reasoning -- what you get for free

Most of the Turtle definition above is RDFS -- class hierarchies, labels, domain/range. The OWL bits are the `owl:inverseOf` declarations and the `owl:ObjectProperty` / `owl:DatatypeProperty` typing. That's enough to enable three useful inferences.

### 1. Subclass inference

Every OpenIndex class is a subclass of a Schema.org type:

<pre><code>oi:Person  <span style="color:#60a5fa">rdfs:subClassOf</span>  schema:Person .</code></pre>

A SPARQL query for `?x a schema:Person` automatically includes every `oi:Person`. Anyone querying against Schema.org types gets OpenIndex data for free. They don't need to know about our namespace.

This works transitively. If someone else defines `ex:Researcher rdfs:subClassOf oi:Person`, then researchers are also `schema:Person`. The subclass chain flows upward without anyone coordinating.

### 2. Inverse properties

Every relationship has an explicit inverse:

<pre><code>oi:mentions  <span style="color:#60a5fa">owl:inverseOf</span>  oi:mentionedIn .</code></pre>

Assert one triple -- "Page A `oi:mentions` Entity X" -- and a reasoner infers the reverse: "Entity X `oi:mentionedIn` Page A." You store one direction, query both. This cuts storage in half for bidirectional relationships without any application logic.

In practice, we store both directions in DuckDB because SQL doesn't have a reasoner. But anyone loading our Turtle export into a triple store with OWL reasoning enabled gets the inverses materialized automatically.

### 3. Domain and range constraints

<pre><code>oi:affiliatedWith
    <span style="color:#60a5fa">rdfs:domain</span>  oi:Person ;
    <span style="color:#60a5fa">rdfs:range</span>   oi:Organization .</code></pre>

If you assert `X oi:affiliatedWith Y`, a reasoner infers that X must be a Person and Y must be an Organization. It's free type checking. An entity with `affiliatedWith` edges is automatically classified as a Person even if nobody explicitly typed it.

This is useful for data quality. If your NER pipeline extracts "MIT affiliatedWith Cambridge" (wrong -- MIT is an Organization, not a Person), the domain constraint flags it. The triple is valid RDF, but it contradicts the ontology. Validation tools catch this.

<div class="note">
  <strong>We're pragmatic about reasoning.</strong> We don't run a reasoner in production. DuckDB handles queries directly with explicit SQL. But the OWL axioms mean anyone who imports our data into Protege, Stardog, or GraphDB gets inference for free. The ontology enables it; we don't mandate it.
</div>

## JSON Schema -- validation without philosophy

OWL validates *semantics*. JSON Schema validates *structure*. Most API consumers want the latter: "does this JSON object have the right keys with the right types?" They don't care about subclass inference.

Here's a compact JSON Schema for an OpenIndex entity:

<pre><code>{
  <span style="color:#60a5fa">"$schema"</span>: <span style="color:#4ade80">"https://json-schema.org/draft/2020-12/schema"</span>,
  <span style="color:#60a5fa">"$id"</span>:     <span style="color:#4ade80">"https://openindex.org/schema/entity.json"</span>,
  <span style="color:#60a5fa">"title"</span>:   <span style="color:#4ade80">"OpenIndex Entity"</span>,
  <span style="color:#60a5fa">"type"</span>:    <span style="color:#4ade80">"object"</span>,
  <span style="color:#60a5fa">"required"</span>: [<span style="color:#4ade80">"@type"</span>, <span style="color:#4ade80">"name"</span>],
  <span style="color:#60a5fa">"properties"</span>: {
    <span style="color:#60a5fa">"@type"</span>:    { <span style="color:#60a5fa">"enum"</span>: [<span style="color:#4ade80">"WebPage"</span>,<span style="color:#4ade80">"Person"</span>,<span style="color:#4ade80">"Organization"</span>,<span style="color:#4ade80">"Place"</span>,<span style="color:#4ade80">"Topic"</span>,<span style="color:#4ade80">"Product"</span>] },
    <span style="color:#60a5fa">"@id"</span>:      { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>, <span style="color:#60a5fa">"format"</span>: <span style="color:#4ade80">"uri"</span> },
    <span style="color:#60a5fa">"name"</span>:     { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
    <span style="color:#60a5fa">"url"</span>:      { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>, <span style="color:#60a5fa">"format"</span>: <span style="color:#4ade80">"uri"</span> },
    <span style="color:#60a5fa">"description"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
    <span style="color:#60a5fa">"sameAs"</span>:   { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"array"</span>,  <span style="color:#60a5fa">"items"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>, <span style="color:#60a5fa">"format"</span>: <span style="color:#4ade80">"uri"</span> } },
    <span style="color:#60a5fa">"confidence"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"number"</span>, <span style="color:#60a5fa">"minimum"</span>: <span style="color:#fbbf24">0</span>, <span style="color:#60a5fa">"maximum"</span>: <span style="color:#fbbf24">1</span> },
    <span style="color:#60a5fa">"crawlId"</span>:  { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span> },
    <span style="color:#60a5fa">"fetchTime"</span>: { <span style="color:#60a5fa">"type"</span>: <span style="color:#4ade80">"string"</span>, <span style="color:#60a5fa">"format"</span>: <span style="color:#4ade80">"date-time"</span> }
  },
  <span style="color:#60a5fa">"additionalProperties"</span>: <span style="color:#fbbf24">true</span>
}</code></pre>

The contrast is clear. OWL says: "if X `affiliatedWith` Y, then X is a Person and Y is an Organization." JSON Schema says: "the `@type` field must be one of these six strings, and `confidence` must be a number between 0 and 1." One reasons about meaning. The other checks shape.

Most API consumers validate with JSON Schema at the edge and never think about OWL. That's fine. The ontology supports both, and they aren't competing -- they validate different things.

## Format comparison

<table>
  <thead>
    <tr>
      <th>Format</th>
      <th>Audience</th>
      <th>Strengths</th>
      <th>Weaknesses</th>
      <th>Extension</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>JSON-LD</strong></td>
      <td>Web developers, APIs</td>
      <td>Native JSON; works with any HTTP client; <code>@context</code> provides linked data semantics without changing the payload shape</td>
      <td>Verbose for large graphs; context resolution adds a network request</td>
      <td><code>.jsonld</code></td>
    </tr>
    <tr>
      <td><strong>Turtle</strong></td>
      <td>Ontology authors, SPARQL users</td>
      <td>Human-readable; compact prefix notation; easy to diff and review in version control</td>
      <td>Not JSON; web tools can't consume it directly</td>
      <td><code>.ttl</code></td>
    </tr>
    <tr>
      <td><strong>OWL/RDF-XML</strong></td>
      <td>Reasoners, Protege</td>
      <td>Full OWL expressiveness; tool support in Protege, Stardog, HermiT</td>
      <td>XML; unreadable by humans; painful to edit by hand</td>
      <td><code>.owl</code> / <code>.rdf</code></td>
    </tr>
    <tr>
      <td><strong>JSON Schema</strong></td>
      <td>API consumers, CI pipelines</td>
      <td>Validates structure; native to every language; works in CI/CD without semantic web tooling</td>
      <td>No reasoning; can't express subclass or inverse relationships</td>
      <td><code>.json</code></td>
    </tr>
  </tbody>
</table>

## Interoperability -- sameAs linking

The `schema:sameAs` property bridges OpenIndex entities to the rest of the linked data universe. Every entity can carry URIs pointing to the same thing in external knowledge bases:

<pre><code><span style="color:#888"># OpenIndex entity linked to external KBs</span>
<span style="color:#4ade80">&lt;https://openindex.org/entity/mozilla-foundation&gt;</span>
    <span style="color:#60a5fa">a</span>            oi:Organization ;
    schema:name  <span style="color:#fbbf24">"Mozilla Foundation"</span> ;
    schema:sameAs
        <span style="color:#4ade80">&lt;https://www.wikidata.org/wiki/Q55672&gt;</span> ,
        <span style="color:#4ade80">&lt;https://dbpedia.org/resource/Mozilla_Foundation&gt;</span> ,
        <span style="color:#4ade80">&lt;https://www.crunchbase.com/organization/mozilla&gt;</span> .</code></pre>

These links enable SPARQL federation -- querying across multiple knowledge bases in a single query. Want to enrich OpenIndex organizations with founding dates from Wikidata?

<pre><code><span style="color:#60a5fa">SELECT</span> ?name ?founded <span style="color:#60a5fa">WHERE</span> {
  <span style="color:#888"># From OpenIndex</span>
  ?org  <span style="color:#60a5fa">a</span>             oi:Organization ;
        schema:name   ?name ;
        schema:sameAs ?wikidata .

  <span style="color:#888"># From Wikidata (via federation)</span>
  <span style="color:#60a5fa">SERVICE</span> <span style="color:#4ade80">&lt;https://query.wikidata.org/sparql&gt;</span> {
    ?wikidata <span style="color:#4ade80">&lt;http://www.wikidata.org/prop/direct/P571&gt;</span> ?founded .
  }
}</code></pre>

<div class="note note-warn">
  <strong>Honest status:</strong> We don't run a SPARQL endpoint yet. The data model fully supports it -- every triple in DuckDB can be exported as valid RDF and loaded into any triple store. Running a public SPARQL endpoint is an operational decision (uptime, query limits, abuse prevention) that comes later. The sameAs links work today for anyone who exports our data into their own endpoint.
</div>

## Write once, publish four ways

The architecture is straightforward. The ontology is defined once, in Turtle, as the canonical source. Everything else is derived:

<pre><code>  <span style="color:#4ade80">ontology.ttl</span>  <span style="color:#888">(canonical source, version-controlled)</span>
       |
       +──> <span style="color:#60a5fa">context.jsonld</span>     <span style="color:#888">JSON-LD context for APIs</span>
       |
       +──> <span style="color:#60a5fa">ontology.owl</span>       <span style="color:#888">OWL/XML for Protege and reasoners</span>
       |
       +──> <span style="color:#60a5fa">entity.schema.json</span> <span style="color:#888">JSON Schema for validation</span>
       |
       +──> <span style="color:#fbbf24">DuckDB triple store</span> <span style="color:#888">internal query engine</span></code></pre>

One change to `ontology.ttl` propagates to all four outputs. The JSON-LD context is generated by extracting prefixes and property mappings. The OWL file is a serialization format change (Turtle to RDF/XML) with no semantic difference. The JSON Schema is extracted from class definitions and property ranges. The DuckDB schema mirrors the entity classes and properties as table columns.

No manual synchronization. No drift between formats. Define once in the most readable format (Turtle), derive the rest mechanically.

That's the bet: the Semantic Web produced an absurdly tall layer cake of specifications. Most of them aren't worth adopting. But the bottom layers -- RDF as a data model, Turtle for human readability, JSON-LD for web transport, and just enough OWL for subclass inference and inverse properties -- give you interoperability with decades of tooling while keeping the complexity manageable.

We'll publish the Turtle file, the JSON-LD context, and the JSON Schema alongside the next ontology revision. The OWL/XML export is automated. Four formats, one truth, zero philosophy tax.
