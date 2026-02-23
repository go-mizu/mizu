import { icons, cardIcon } from '../icons'

export const apiReferencePage = `
<div class="note note-warn">
  The OpenIndex API is in <strong>early alpha</strong>. Endpoints are subject to change. There is no authentication, no rate limiting, and no SLA. Use at your own risk.
</div>

<h2>Current Deployment</h2>
<p>The API is currently deployed as a Cloudflare Worker (CC Viewer). It provides read-only access to Common Crawl data.</p>

<pre><code>Base URL: https://cc-viewer.go-mizu.workers.dev</code></pre>

<hr>

<h2>Implemented Endpoints</h2>
<p>These endpoints are live and functional today.</p>

<h3>Health Check</h3>
<div class="endpoint">
  <div class="endpoint-header">
    <span class="endpoint-method method-get">GET</span>
    <span class="endpoint-path">/api/health</span>
  </div>
  <div class="endpoint-body">
    <p>Returns service health status.</p>
<pre><code>curl https://cc-viewer.go-mizu.workers.dev/api/health</code></pre>
  </div>
</div>

<h3>List Crawls</h3>
<div class="endpoint">
  <div class="endpoint-header">
    <span class="endpoint-method method-get">GET</span>
    <span class="endpoint-path">/api/crawls</span>
  </div>
  <div class="endpoint-body">
    <p>Lists available Common Crawl crawl IDs.</p>
<pre><code>curl https://cc-viewer.go-mizu.workers.dev/api/crawls</code></pre>
  </div>
</div>

<h3>URL Lookup</h3>
<div class="endpoint">
  <div class="endpoint-header">
    <span class="endpoint-method method-get">GET</span>
    <span class="endpoint-path">/api/url?url={url}</span>
  </div>
  <div class="endpoint-body">
    <p>Looks up a URL in the Common Crawl CDX index. Returns metadata and WARC location.</p>
    <h4>Parameters</h4>
    <table>
      <thead><tr><th>Param</th><th>Type</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td><code>url</code></td><td>string</td><td>URL to look up (required)</td></tr>
      </tbody>
    </table>
<pre><code>curl "https://cc-viewer.go-mizu.workers.dev/api/url?url=https://example.com"</code></pre>
  </div>
</div>

<h3>Domain Browse</h3>
<div class="endpoint">
  <div class="endpoint-header">
    <span class="endpoint-method method-get">GET</span>
    <span class="endpoint-path">/api/domain/{domain}</span>
  </div>
  <div class="endpoint-body">
    <p>Lists crawled URLs for a domain from the CC CDX index.</p>
<pre><code>curl https://cc-viewer.go-mizu.workers.dev/api/domain/example.com</code></pre>
  </div>
</div>

<h3>View WARC Record</h3>
<div class="endpoint">
  <div class="endpoint-header">
    <span class="endpoint-method method-get">GET</span>
    <span class="endpoint-path">/api/view?url={url}</span>
  </div>
  <div class="endpoint-body">
    <p>Fetches and renders the WARC record for a URL. Decompresses gzip on the edge.</p>
<pre><code>curl "https://cc-viewer.go-mizu.workers.dev/api/view?url=https://example.com"</code></pre>
  </div>
</div>

<hr>

<h2>Planned Endpoints</h2>
<p>The following endpoints are planned but <strong>not yet implemented</strong>.</p>

<h3>Full-Text Search</h3>
<div class="endpoint">
  <div class="endpoint-header">
    <span class="endpoint-method method-get">GET</span>
    <span class="endpoint-path">/api/search?q={query}</span>
  </div>
  <div class="endpoint-body">
    <p><strong>Status: Planned.</strong> Full-text search across indexed content. Will support filtering by domain, language, and date range.</p>
<pre><code># Not yet available
curl "https://api.openindex.org/v1/search?q=machine+learning&limit=10"</code></pre>
  </div>
</div>

<h3>Parquet Index Query</h3>
<div class="endpoint">
  <div class="endpoint-header">
    <span class="endpoint-method method-post">POST</span>
    <span class="endpoint-path">/api/query</span>
  </div>
  <div class="endpoint-body">
    <p><strong>Status: Planned.</strong> Execute SQL queries against the columnar index. Will use DuckDB on the backend.</p>
<pre><code># Not yet available
curl -X POST "https://api.openindex.org/v1/query" \\
  -H "Content-Type: application/json" \\
  -d '{"sql": "SELECT url, status FROM index WHERE domain = '\\''example.com'\\'' LIMIT 10"}'</code></pre>
  </div>
</div>

<h3>Batch URL Fetch</h3>
<div class="endpoint">
  <div class="endpoint-header">
    <span class="endpoint-method method-post">POST</span>
    <span class="endpoint-path">/api/fetch</span>
  </div>
  <div class="endpoint-body">
    <p><strong>Status: Internal.</strong> A batch URL fetcher deployed as a separate worker. Currently used internally for recrawling. May be exposed publicly in the future.</p>
<pre><code># Internal endpoint (requires auth token)
curl -X POST "https://url-fetcher.go-mizu.workers.dev/fetch" \\
  -H "Authorization: Bearer {token}" \\
  -H "Content-Type: application/json" \\
  -d '{"urls": ["https://example.com", "https://example.org"]}'</code></pre>
  </div>
</div>

<hr>

<h2>Error Handling</h2>
<p>The API returns standard HTTP status codes. Error responses include a JSON body:</p>

<pre><code>{
  "error": "URL parameter is required"
}</code></pre>

<table>
  <thead><tr><th>Status</th><th>Meaning</th></tr></thead>
  <tbody>
    <tr><td>200</td><td>Success</td></tr>
    <tr><td>400</td><td>Bad request (missing or invalid parameters)</td></tr>
    <tr><td>404</td><td>URL not found in index</td></tr>
    <tr><td>500</td><td>Server error</td></tr>
  </tbody>
</table>

<div class="note">
  The API is a Cloudflare Worker with a 30-second execution limit. Large WARC record fetches may time out. If this happens, use the CLI for direct byte-range requests instead.
</div>
`
