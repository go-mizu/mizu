import { icons, cardIcon } from '../icons'

export const researchPage = `
<h2>Research</h2>
<p>OpenIndex is a new project (started February 2026). There are no published papers or academic citations yet.</p>

<div class="note">
  If you are a researcher interested in using OpenIndex tools or data for your work, we would like to hear from you. Open an issue on <a href="https://github.com/nicholasgasior/gopher-crawl">GitHub</a> to start a conversation.
</div>

<hr>

<h2>Available Data for Research</h2>
<p>The following data and tools are available today for research use:</p>

<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('database')} Common Crawl Index</div>
    <p>Query CC's columnar Parquet index with DuckDB. Billions of URLs with metadata: domain, TLD, status, language, WARC location.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('globe')} Recrawled Data</div>
    <p>Recrawl results stored in sharded DuckDB. URL reachability, status codes, response times, redirect chains.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('layers')} Domain Crawls</div>
    <p>Full single-domain crawl data: pages, links, content types. Stored in DuckDB with bloom filter deduplication.</p>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('search')} Site Extraction</div>
    <p>All pages for a domain from CC archives. URLs, extracted links, and optionally full page content.</p>
  </div>
</div>

<hr>

<h2>Potential Research Areas</h2>
<p>Areas where OpenIndex data could be useful:</p>

<ul>
  <li><strong>Web archival studies</strong> -- URL decay rates, domain lifespans, content drift over time.</li>
  <li><strong>Web infrastructure</strong> -- TLS adoption, HTTP/2 prevalence, server distribution, CDN usage.</li>
  <li><strong>Link analysis</strong> -- Domain-level and page-level link graphs from CC WAT data and site extraction.</li>
  <li><strong>Language coverage</strong> -- Distribution of languages on the web, underrepresented language content.</li>
  <li><strong>Crawl engineering</strong> -- DNS resolution patterns, connection management, rate limiting strategies.</li>
  <li><strong>NLP training data</strong> -- Filtered web text via CC WET files, deduplicated with Parquet digests.</li>
</ul>

<hr>

<h2>How to Cite</h2>
<p>If you use OpenIndex tools in your research, you can reference the GitHub repository:</p>

<pre><code>@software{openindex2026,
  author    = {Gasior, Nicholas},
  title     = {OpenIndex: Open-Source Web Intelligence Tools},
  url       = {https://github.com/nicholasgasior/gopher-crawl},
  year      = {2026},
  license   = {Apache-2.0}
}</code></pre>

<p>There is no published paper yet. If and when one exists, it will be listed here.</p>

<hr>

<h2>License</h2>
<p>All OpenIndex software is released under the <strong>Apache License 2.0</strong>. You can use, modify, and distribute it for any purpose, including commercial and academic use.</p>

<p>Common Crawl data is available under its own terms at <a href="https://commoncrawl.org/terms-of-use">commoncrawl.org/terms-of-use</a>.</p>
`
