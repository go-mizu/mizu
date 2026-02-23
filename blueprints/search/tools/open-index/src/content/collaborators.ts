import { icons, cardIcon } from '../icons'

export const collaboratorsPage = `
<h2>Technologies We Build On</h2>
<p>OpenIndex is built on the shoulders of exceptional open-source projects and open data initiatives. These are the technologies that make the platform possible.</p>

<div class="collab-grid">
  <div class="collab-card">
    <div class="collab-card-name">${cardIcon('code')} Go</div>
    <p>Core language for the CLI, crawler, recrawler, and domain crawler. Go 1.22+ with net/http ServeMux patterns.</p>
  </div>
  <div class="collab-card">
    <div class="collab-card-name">${cardIcon('database')} DuckDB</div>
    <p>Analytical database for sharded storage, Parquet queries, and crawl analytics. Runs embedded, no server needed.</p>
  </div>
  <div class="collab-card">
    <div class="collab-card-name">${cardIcon('package')} Apache Parquet</div>
    <p>Columnar file format for the crawl index. Compact, fast to query, and compatible with every data tool.</p>
  </div>
  <div class="collab-card">
    <div class="collab-card-name">${cardIcon('search')} Vald</div>
    <p>Distributed vector search engine for semantic similarity. Planned integration for dense embedding search across the corpus.</p>
  </div>
  <div class="collab-card">
    <div class="collab-card-name">${cardIcon('zap')} Hono</div>
    <p>Lightweight TypeScript web framework for the search worker. Runs on Cloudflare Workers at the edge.</p>
  </div>
  <div class="collab-card">
    <div class="collab-card-name">${cardIcon('globe')} Cloudflare Workers</div>
    <p>Edge runtime for the search worker and URL fetcher. Low latency, globally distributed, zero cold starts.</p>
  </div>
  <div class="collab-card">
    <div class="collab-card-name">${cardIcon('layers')} Common Crawl</div>
    <p>Open web crawl archive. OpenIndex integrates directly with CC's CDX API, WARC files, and Parquet columnar index.</p>
  </div>
  <div class="collab-card">
    <div class="collab-card-name">${cardIcon('boxes')} Apache Arrow</div>
    <p>In-memory columnar format underlying Parquet. Powers fast analytical processing across the pipeline.</p>
  </div>
  <div class="collab-card">
    <div class="collab-card-name">${cardIcon('terminal')} Zig</div>
    <p>Systems language for the high-performance recrawler. Raw TCP + TLS for maximum throughput at minimal memory.</p>
  </div>
  <div class="collab-card">
    <div class="collab-card-name">${cardIcon('cpu')} Mizu</div>
    <p>Lightweight Go web framework that OpenIndex is part of. Thin context layer, middleware support, and app lifecycle management.</p>
  </div>
</div>

<hr>

<h3>We're Looking for Collaborators</h3>
<p>OpenIndex is a solo project today, but it does not have to stay that way. If you work with any of these technologies -- or bring expertise in areas like NLP, information retrieval, distributed systems, or web standards -- there is room to contribute.</p>

<p>What collaboration looks like:</p>
<ul>
  <li><strong>Code contributions</strong> -- Pick up an issue, submit a PR, improve a package.</li>
  <li><strong>Domain expertise</strong> -- Help design the ontology, improve crawl quality heuristics, or refine the search pipeline.</li>
  <li><strong>Data contributions</strong> -- Share seed URL lists, language-specific resources, or quality signals.</li>
  <li><strong>Testing and feedback</strong> -- Run the tools, report bugs, suggest improvements.</li>
</ul>

<div class="note">
  <strong>Interested?</strong> Open an issue on <a href="https://github.com/nicholasgasior/gopher-crawl">GitHub</a> or reach out at <a href="mailto:hello@openindex.org">hello@openindex.org</a>. No formal process -- just start a conversation.
</div>
`
