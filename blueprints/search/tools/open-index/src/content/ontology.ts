import { icons, cardIcon } from '../icons'

export const ontologyPage = `
<h2>What is an Ontology?</h2>
<p>An ontology is a formal schema that defines the types of entities, their properties, and the relationships between them. For web intelligence, an ontology provides a shared vocabulary for describing what web content is <em>about</em> -- not just what it contains.</p>

<p>The OpenIndex Ontology will define the entity types, properties, and relationships used across the platform: in the knowledge graph, search index, and API responses.</p>

<div class="note note-warn">
  <strong>Status: Early design phase.</strong> The ontology is being designed. No formal specification has been published yet. This page describes the vision and planned approach. Contributions and feedback are welcome.
</div>

<h2>Why an Ontology Matters</h2>
<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('layers')} <span>Shared Vocabulary</span></div>
    <p>Without an ontology, "Mozilla" could be a person's name, an organization, or a product. The ontology defines what entity types exist and how to tell them apart.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('gitFork')} <span>Typed Relationships</span></div>
    <p>"Person works at Organization" and "Product created by Organization" are different relationships. The ontology defines which connections are valid between which types.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('code')} <span>Interoperability</span></div>
    <p>A well-defined ontology means other tools and systems can consume OpenIndex data correctly. Export to JSON-LD, RDF, or OWL and it works with existing semantic web infrastructure.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('shield')} <span>Data Quality</span></div>
    <p>Schema validation ensures every entity has the right properties and every relationship connects the right types. Catches errors before they enter the knowledge graph.</p>
  </div>
</div>

<h2>Design Principles</h2>

<h3>Schema.org Compatibility</h3>
<p>The ontology will extend <a href="https://schema.org/">Schema.org</a> rather than replace it. Every OpenIndex entity class will map to a Schema.org type or be a subclass of one. This means:</p>
<ul>
  <li>Pages with JSON-LD / Microdata / RDFa using Schema.org types map automatically to OpenIndex entities.</li>
  <li>Entity exports use Schema.org properties where applicable.</li>
  <li>Custom properties use the <code>oi:</code> namespace prefix.</li>
</ul>

<pre><code>// Planned namespace structure
@prefix oi:     &lt;https://openindex.org/ontology/&gt; .
@prefix schema: &lt;https://schema.org/&gt; .
@prefix rdfs:   &lt;http://www.w3.org/2000/01/rdf-schema#&gt; .

// OpenIndex Person extends schema:Person
oi:Person rdfs:subClassOf schema:Person .
oi:Person rdfs:label "Person" .
oi:Person rdfs:comment "A named individual identified in web content." .</code></pre>

<h3>Extensible</h3>
<p>New entity types, properties, and relationships can be proposed and added. The ontology is meant to grow with the project and community needs.</p>

<h3>Practical</h3>
<p>Focused on entities and relationships commonly found on the web and useful for search, analytics, and knowledge extraction. Not an academic exercise -- a working vocabulary.</p>

<h2>Planned Entity Types</h2>
<p>The following entity types are planned for the initial version:</p>

<details>
  <summary>WebPage (schema:WebPage)</summary>
  <div class="details-body">
    <p>The fundamental unit: a crawled web page.</p>
    <table>
      <thead>
        <tr><th>Property</th><th>Type</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr><td><code>url</code></td><td>URL</td><td>Canonical URL</td></tr>
        <tr><td><code>name</code></td><td>Text</td><td>Page title</td></tr>
        <tr><td><code>description</code></td><td>Text</td><td>Meta description</td></tr>
        <tr><td><code>inLanguage</code></td><td>Language</td><td>Primary language</td></tr>
        <tr><td><code>oi:fetchTime</code></td><td>DateTime</td><td>When crawled</td></tr>
        <tr><td><code>oi:crawlId</code></td><td>Text</td><td>Crawl identifier</td></tr>
      </tbody>
    </table>
  </div>
</details>

<details>
  <summary>Person (schema:Person)</summary>
  <div class="details-body">
    <p>A named individual identified through NER, structured data, or link analysis.</p>
    <table>
      <thead>
        <tr><th>Property</th><th>Type</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr><td><code>name</code></td><td>Text</td><td>Full name</td></tr>
        <tr><td><code>alternateName</code></td><td>Text[]</td><td>Aliases</td></tr>
        <tr><td><code>affiliation</code></td><td>Organization</td><td>Organizational affiliation</td></tr>
        <tr><td><code>sameAs</code></td><td>URL[]</td><td>External identifiers (Wikidata, etc.)</td></tr>
      </tbody>
    </table>
  </div>
</details>

<details>
  <summary>Organization (schema:Organization)</summary>
  <div class="details-body">
    <p>A company, institution, government agency, or NGO.</p>
    <table>
      <thead>
        <tr><th>Property</th><th>Type</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr><td><code>name</code></td><td>Text</td><td>Official name</td></tr>
        <tr><td><code>url</code></td><td>URL</td><td>Official website</td></tr>
        <tr><td><code>location</code></td><td>Place</td><td>Headquarters</td></tr>
        <tr><td><code>sameAs</code></td><td>URL[]</td><td>External identifiers</td></tr>
      </tbody>
    </table>
  </div>
</details>

<details>
  <summary>Place (schema:Place)</summary>
  <div class="details-body">
    <p>A geographic location: country, city, address, region.</p>
    <table>
      <thead>
        <tr><th>Property</th><th>Type</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr><td><code>name</code></td><td>Text</td><td>Place name</td></tr>
        <tr><td><code>geo</code></td><td>GeoCoordinates</td><td>Latitude/longitude</td></tr>
        <tr><td><code>containedInPlace</code></td><td>Place</td><td>Parent location</td></tr>
      </tbody>
    </table>
  </div>
</details>

<details>
  <summary>Topic (oi:Topic, extends schema:Thing)</summary>
  <div class="details-body">
    <p>A subject, field of study, or concept. Used to classify content and connect entities.</p>
    <table>
      <thead>
        <tr><th>Property</th><th>Type</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr><td><code>name</code></td><td>Text</td><td>Topic name</td></tr>
        <tr><td><code>broader</code></td><td>Topic</td><td>Parent topic</td></tr>
        <tr><td><code>narrower</code></td><td>Topic[]</td><td>Child topics</td></tr>
        <tr><td><code>sameAs</code></td><td>URL[]</td><td>Wikidata, LCSH, etc.</td></tr>
      </tbody>
    </table>
  </div>
</details>

<details>
  <summary>Product (schema:Product / schema:SoftwareApplication)</summary>
  <div class="details-body">
    <p>Software, hardware, or commercial product.</p>
    <table>
      <thead>
        <tr><th>Property</th><th>Type</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr><td><code>name</code></td><td>Text</td><td>Product name</td></tr>
        <tr><td><code>manufacturer</code></td><td>Organization</td><td>Developer or manufacturer</td></tr>
        <tr><td><code>url</code></td><td>URL</td><td>Official product page</td></tr>
      </tbody>
    </table>
  </div>
</details>

<h2>Planned Relationship Types</h2>
<table>
  <thead>
    <tr>
      <th>Relationship</th>
      <th>Domain</th>
      <th>Range</th>
      <th>Inverse</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>oi:mentions</code></td>
      <td>WebPage</td>
      <td>Any</td>
      <td><code>oi:mentionedIn</code></td>
    </tr>
    <tr>
      <td><code>oi:affiliatedWith</code></td>
      <td>Person</td>
      <td>Organization</td>
      <td><code>oi:hasAffiliate</code></td>
    </tr>
    <tr>
      <td><code>oi:locatedIn</code></td>
      <td>Org / Event</td>
      <td>Place</td>
      <td><code>oi:locationOf</code></td>
    </tr>
    <tr>
      <td><code>oi:createdBy</code></td>
      <td>Product</td>
      <td>Person / Org</td>
      <td><code>oi:created</code></td>
    </tr>
    <tr>
      <td><code>oi:about</code></td>
      <td>WebPage</td>
      <td>Topic</td>
      <td><code>oi:topicOf</code></td>
    </tr>
    <tr>
      <td><code>oi:linksTo</code></td>
      <td>WebPage</td>
      <td>WebPage</td>
      <td><code>oi:linkedFrom</code></td>
    </tr>
    <tr>
      <td><code>schema:sameAs</code></td>
      <td>Any</td>
      <td>Any</td>
      <td>symmetric</td>
    </tr>
  </tbody>
</table>

<h2>Planned Output Formats</h2>
<p>The ontology definition will be available in standard semantic web formats:</p>
<table>
  <thead>
    <tr>
      <th>Format</th>
      <th>Use Case</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>JSON-LD Context</strong></td>
      <td>Web-native linked data, API responses</td>
    </tr>
    <tr>
      <td><strong>RDF/Turtle</strong></td>
      <td>Compact RDF for triple stores</td>
    </tr>
    <tr>
      <td><strong>OWL</strong></td>
      <td>Formal ontology editors, reasoning engines</td>
    </tr>
    <tr>
      <td><strong>JSON Schema</strong></td>
      <td>Validation of entity documents</td>
    </tr>
  </tbody>
</table>

<h2>Example Entity (Planned Format)</h2>
<pre><code>{
  "@context": [
    "https://schema.org",
    "https://openindex.org/ontology/context.jsonld"
  ],
  "@type": "Organization",
  "name": "Mozilla Foundation",
  "alternateName": ["Mozilla", "MoFo"],
  "url": "https://mozilla.org",
  "foundingDate": "2003-07-15",
  "location": {
    "@type": "Place",
    "name": "San Francisco, California"
  },
  "sameAs": [
    "https://www.wikidata.org/wiki/Q55672"
  ],
  "oi:crawlId": "OI-2026-02",
  "oi:confidence": 0.99
}</code></pre>

<h2>Contributing</h2>
<p>The ontology is in early design. This is a good time to contribute -- the schema is not yet locked down. Feedback on entity types, properties, and relationships is especially useful.</p>
<ul>
  <li>Open an issue on <a href="https://github.com/nicholasgasior/gopher-crawl">GitHub</a> with the <code>ontology</code> label</li>
  <li>Describe the proposed type/property, its Schema.org alignment, and example use cases</li>
  <li>See the <a href="/contributing">Contributing</a> page for the general contribution process</li>
</ul>
`
