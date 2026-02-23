import { icons, cardIcon } from '../icons'

export const knowledgeGraphPage = `
<h2>Knowledge Graph</h2>
<p>The OpenIndex Knowledge Graph aims to go beyond raw web data to model the semantic meaning of web content -- people, organizations, places, topics, and the relationships between them.</p>

<div class="note note-warn">
  <strong>Status: Early design.</strong> The knowledge graph is in planning. Web graph data is accessible through Common Crawl's existing web graph datasets. Entity extraction and the entity graph are not yet built.
</div>

<h2>What Exists Today</h2>
<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('globe')} <span>Web Graph (via CC)</span></div>
    <h3>Accessible Now</h3>
    <p>Common Crawl publishes host-level and domain-level web graphs. OpenIndex can access these through the CC integration package. These capture hyperlink structure between pages.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('gitFork')} <span>Link Data from Crawls</span></div>
    <h3>Available</h3>
    <p>The domain crawler (pkg/dcrawler) extracts outgoing links from every crawled page and stores them in the sharded DuckDB. The CC site extractor captures page+link relationships.</p>
  </div>
</div>

<h2>What Is Planned</h2>

<h3>Entity Graph</h3>
<p>A semantic graph of named entities extracted from web content. Unlike the web graph (page-to-page links), the entity graph models real-world relationships.</p>

<p>Planned entity types:</p>
<table>
  <thead>
    <tr>
      <th>Entity Type</th>
      <th>Description</th>
      <th>Example</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Person</strong></td>
      <td>Named individuals</td>
      <td>Linus Torvalds, Ada Lovelace</td>
    </tr>
    <tr>
      <td><strong>Organization</strong></td>
      <td>Companies, institutions, agencies</td>
      <td>Mozilla Foundation, CERN</td>
    </tr>
    <tr>
      <td><strong>Location</strong></td>
      <td>Geographic places</td>
      <td>Geneva, Switzerland</td>
    </tr>
    <tr>
      <td><strong>Topic</strong></td>
      <td>Subjects and fields of study</td>
      <td>Machine Learning, Climate Science</td>
    </tr>
    <tr>
      <td><strong>Event</strong></td>
      <td>Conferences, incidents</td>
      <td>NeurIPS, World Cup</td>
    </tr>
    <tr>
      <td><strong>Product</strong></td>
      <td>Software, hardware</td>
      <td>PostgreSQL, Linux</td>
    </tr>
  </tbody>
</table>

<h3>Planned Extraction Pipeline</h3>
<p>Entity extraction will use multiple signal sources:</p>

<details>
  <summary>Named Entity Recognition (NER)</summary>
  <div class="details-body">
    <p>Multilingual NER model to identify entity mentions in page text. Entity mentions linked to canonical IDs through entity resolution. Technology TBD -- evaluating spaCy, Stanza, and transformer-based models.</p>
  </div>
</details>

<details>
  <summary>Schema.org Parsing</summary>
  <div class="details-body">
    <p>Parse structured data markup (JSON-LD, Microdata, RDFa) to extract typed entities. Schema.org types map to OpenIndex entity types. This provides high-confidence data with explicit properties.</p>
  </div>
</details>

<details>
  <summary>Link Analysis</summary>
  <div class="details-body">
    <p>Anchor text analysis to extract relationships from hyperlinks. Outgoing links from entity pages (e.g., Wikipedia) used to discover entity relationships. The domain crawler already captures this data.</p>
  </div>
</details>

<details>
  <summary>Entity Resolution</summary>
  <div class="details-body">
    <p>Merge multiple mentions of the same entity using string similarity, context matching, and link-based signals. Link to external identifiers (Wikidata QIDs, DBpedia URIs).</p>
  </div>
</details>

<h3>Planned Relationship Types</h3>
<table>
  <thead>
    <tr>
      <th>Relationship</th>
      <th>Source</th>
      <th>Target</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>mentions</code></td>
      <td>WebPage</td>
      <td>Any Entity</td>
    </tr>
    <tr>
      <td><code>affiliatedWith</code></td>
      <td>Person</td>
      <td>Organization</td>
    </tr>
    <tr>
      <td><code>locatedIn</code></td>
      <td>Organization / Event</td>
      <td>Location</td>
    </tr>
    <tr>
      <td><code>createdBy</code></td>
      <td>Product</td>
      <td>Person / Organization</td>
    </tr>
    <tr>
      <td><code>linksTo</code></td>
      <td>WebPage</td>
      <td>WebPage</td>
    </tr>
    <tr>
      <td><code>sameAs</code></td>
      <td>Any Entity</td>
      <td>Any Entity</td>
    </tr>
  </tbody>
</table>

<h2>Web Graph Access Today</h2>
<p>Common Crawl's web graph data is available through the CC integration:</p>

<pre><code># Access CC web graph data
# Host-level and domain-level graphs available per crawl
# See: https://commoncrawl.org/web-graphs

# The CC site extractor captures links for specific domains:
search cc site example.com --mode links

# Domain crawler also captures all outgoing links:
search crawl-domain example.com --max-pages 1000
# Links stored in results DB: pages + links tables</code></pre>

<h2>Why Build a Knowledge Graph</h2>
<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('search')} <span>Better Search</span></div>
    <p>Find content by entity, not just keyword. "Pages mentioning Mozilla" instead of pages containing the string "mozilla".</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('gitFork')} <span>Relationship Discovery</span></div>
    <p>Map connections between entities across millions of pages. Who works where, what organizations are related, how topics connect.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('trendingUp')} <span>Web Analysis</span></div>
    <p>Study information diffusion, track topic spread, analyze organizational networks. The web graph already enables link analysis and authority metrics.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('layers')} <span>Structured Data</span></div>
    <p>Convert unstructured web content into queryable structured knowledge. Foundation for downstream applications and research.</p>
  </div>
</div>

<h2>Roadmap</h2>
<table>
  <thead>
    <tr>
      <th>Phase</th>
      <th>Description</th>
      <th>Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Phase 0</strong></td>
      <td>Web graph via CC integration + domain crawler link extraction</td>
      <td>Available</td>
    </tr>
    <tr>
      <td><strong>Phase 1</strong></td>
      <td>Schema.org structured data parsing from crawled pages</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>Phase 2</strong></td>
      <td>NER pipeline for entity extraction</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>Phase 3</strong></td>
      <td>Entity resolution and relationship mapping</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>Phase 4</strong></td>
      <td>Graph query API</td>
      <td>Planned</td>
    </tr>
  </tbody>
</table>

<p>Interested in contributing to the knowledge graph? See the <a href="/contributing">Contributing</a> page or open an issue on <a href="https://github.com/nicholasgasior/gopher-crawl">GitHub</a>.</p>
`
