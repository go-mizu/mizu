import { icons, cardIcon } from '../icons'

export const vectorSearchPage = `
<h2>What is Vector Search?</h2>
<p>Vector search finds web pages by meaning rather than exact keyword matches. Instead of looking for pages containing specific words, vector search finds pages whose content is semantically similar to a query -- even when they use completely different terminology.</p>

<div class="note note-warn">
  <strong>Status: Planned.</strong> Vector search is designed but not yet deployed. No embeddings have been generated yet. This page describes the vision and planned architecture.
</div>

<div class="cards">
  <div class="card">
    <h3>Keyword Search</h3>
    <p>Query: <code>"automobile safety regulations"</code></p>
    <p>Finds pages containing those exact terms. Misses pages about "car crash standards" or "vehicle protection laws" that use different words.</p>
  </div>
  <div class="card">
    <h3>Vector Search</h3>
    <p>Query: <code>"automobile safety regulations"</code></p>
    <p>Finds pages about the concept of vehicle safety, regardless of specific words. Also surfaces "NHTSA requirements" and "EU vehicle safety directives".</p>
  </div>
</div>

<h2>How It Will Work</h2>
<p>Every crawled page will be processed through an embedding model that converts text content into a dense numerical vector -- a list of floating-point numbers encoding the page's semantic meaning.</p>

<h3>Planned Embedding Pipeline</h3>
<ol>
  <li><strong>Text extraction</strong> -- Clean plaintext extracted from HTML, removing boilerplate, navigation, and ads.</li>
  <li><strong>Input construction</strong> -- Page title + first 512 tokens of body text concatenated.</li>
  <li><strong>Encoding</strong> -- Passed through an embedding model (likely multilingual-e5-large), producing a 1024-dimensional vector.</li>
  <li><strong>Normalization</strong> -- L2-normalized so cosine similarity equals dot product.</li>
  <li><strong>Indexing</strong> -- Inserted into Vald distributed vector database.</li>
</ol>

<h2>Planned Architecture: Vald</h2>
<p><a href="https://vald.vdaas.org/">Vald</a> is a distributed approximate nearest neighbor (ANN) search engine developed by Yahoo Japan. It was selected for OpenIndex because of its distributed architecture, high throughput, and Kubernetes-native deployment.</p>

<h3>Vald Components</h3>
<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Role</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Vald Agent</strong></td>
      <td>Stores vector data, runs ANN search using NGT algorithm</td>
    </tr>
    <tr>
      <td><strong>Vald LB Gateway</strong></td>
      <td>Distributes queries across agent nodes</td>
    </tr>
    <tr>
      <td><strong>Vald Discoverer</strong></td>
      <td>Service discovery for agents in Kubernetes</td>
    </tr>
    <tr>
      <td><strong>Vald Index Manager</strong></td>
      <td>Coordinates index creation and rebalancing</td>
    </tr>
  </tbody>
</table>

<h3>Planned Configuration</h3>
<table>
  <thead>
    <tr>
      <th>Property</th>
      <th>Planned Value</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Embedding model</strong></td>
      <td>multilingual-e5-large (evaluating)</td>
    </tr>
    <tr>
      <td><strong>Dimensions</strong></td>
      <td>1024</td>
    </tr>
    <tr>
      <td><strong>Algorithm</strong></td>
      <td>NGT-ANNg (graph-based ANN)</td>
    </tr>
    <tr>
      <td><strong>Similarity metric</strong></td>
      <td>Cosine similarity (default)</td>
    </tr>
    <tr>
      <td><strong>Granularity</strong></td>
      <td>Per-page (title + first 512 tokens)</td>
    </tr>
    <tr>
      <td><strong>Max tokens</strong></td>
      <td>512</td>
    </tr>
    <tr>
      <td><strong>Languages</strong></td>
      <td>100+ (multilingual model)</td>
    </tr>
  </tbody>
</table>

<h2>Similarity Metrics</h2>
<table>
  <thead>
    <tr>
      <th>Metric</th>
      <th>Range</th>
      <th>Best For</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Cosine Similarity</strong></td>
      <td>[-1, 1]</td>
      <td>Semantic similarity. 1.0 = identical meaning. Invariant to vector magnitude.</td>
    </tr>
    <tr>
      <td><strong>L2 Distance</strong></td>
      <td>[0, inf)</td>
      <td>Exact content matching. 0.0 = identical. Sensitive to magnitude.</td>
    </tr>
  </tbody>
</table>

<h2>Planned API</h2>

<div class="endpoint">
  <div class="endpoint-header">
    <span class="endpoint-method method-post">POST</span>
    <span class="endpoint-path">/v1/vector/search</span>
  </div>
  <div class="endpoint-body">
    <p>Find pages semantically similar to a query. Pass text (auto-embedded) or a raw vector.</p>
    <p><strong>Status:</strong> Not yet implemented.</p>
  </div>
</div>

<pre><code># Planned API usage (not yet available)
curl -X POST "https://api.openindex.org/v1/vector/search" \\
  -H "Content-Type: application/json" \\
  -d '{
    "query": "recent advances in quantum computing",
    "k": 10,
    "metric": "cosine"
  }'

# Planned response format
{
  "results": [
    {
      "url": "https://example.com/quantum-computing",
      "title": "Quantum Computing Breakthroughs",
      "similarity": 0.951,
      "language": "en"
    }
  ]
}</code></pre>

<h2>Use Cases</h2>
<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('search')} <span>Semantic Search</span></div>
    <p>Find content by meaning across the entire web index. Query in natural language, get results that match the concept regardless of exact wording.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('layers')} <span>Content Clustering</span></div>
    <p>Group pages by semantic similarity. Discover topic clusters, identify emerging trends, map knowledge domains.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('sparkles')} <span>RAG</span></div>
    <p>Retrieval-augmented generation: retrieve relevant web pages as context for LLMs. Build grounded AI systems backed by real web content.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('database')} <span>Deduplication</span></div>
    <p>Find near-duplicate pages with different URLs but semantically identical content. Clean datasets, detect content farms.</p>
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
      <td><strong>1</strong></td>
      <td>Evaluate embedding models (multilingual-e5-large, BGE, etc.)</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>2</strong></td>
      <td>Generate embeddings for a sample crawl</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>3</strong></td>
      <td>Deploy Vald cluster, load sample vectors</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>4</strong></td>
      <td>API endpoint for vector search</td>
      <td>Planned</td>
    </tr>
    <tr>
      <td><strong>5</strong></td>
      <td>Scale to full crawl data</td>
      <td>Planned</td>
    </tr>
  </tbody>
</table>

<p>Interested in helping build vector search? See the <a href="/contributing">Contributing</a> page or check the <a href="https://github.com/nicholasgasior/gopher-crawl">GitHub repo</a>.</p>
`
