---
slug: cc-viewer
title: "Building a Common Crawl Viewer on Cloudflare Workers"
date: 2026-02-16
summary: "A shareable URL beats a hundred CLI invocations. SSR HTML on the edge, WARC decompression, KV caching -- no React required."
tags: [engineering, cloudflare]
---

Last month I spent a week in my CLI, running `search cc url` and `search cc site` and piping outputs through jq and less. The data was all there -- every Common Crawl capture for any URL, full WARC records, domain-level page listings. But every time someone on the team asked "can you check if CC has this page?", the answer was "sure, let me SSH in and run a command." That got old fast.

A URL you can share is worth a hundred CLI invocations. So I built a web viewer -- a Cloudflare Worker that queries Common Crawl's APIs, decompresses WARC records on the edge, and serves server-rendered HTML. No React, no build step, no client-side JavaScript framework. Just Hono, template literals, and KV caching.

It is live at [cc-viewer.go-mizu.workers.dev](https://cc-viewer.go-mizu.workers.dev).

## Why Not Just Link to Common Crawl?

Common Crawl's own interface is built for machines. The CDX API returns newline-delimited JSON. The WARC files are gzipped binary blobs on S3. The crawl list is a JSON array at an endpoint most people will never find. All of this is excellent for programmatic access and terrible for "hey, look at this page from 2019."

I wanted five things:

- Paste a URL, see every CC capture of it across all crawls
- Browse all captured pages for a domain
- View the actual WARC content -- rendered, in the browser
- List available crawls with dates and page counts
- Share any of the above as a link

A CLI can do the first four. Only a web app can do the fifth.

## Hono + SSR: The Same Trick Twice

The OpenIndex site itself -- the one you are reading right now -- runs on Cloudflare Workers with Hono and server-side HTML via template literals. No JSX, no virtual DOM, no hydration step. The CC Viewer uses the exact same approach.

<pre><code><span style="color:#60a5fa">// A route in the CC Viewer</span>
app.get(<span style="color:#4ade80">'/url/*'</span>, async (c) => {
  const targetUrl = c.req.path.replace(<span style="color:#4ade80">'/url/'</span>, <span style="color:#4ade80">''</span>)
  const captures = await queryCDX(targetUrl)

  return c.html(layout(<span style="color:#4ade80">\`
    &lt;h2&gt;Captures for \${escapeHtml(targetUrl)}&lt;/h2&gt;
    &lt;table&gt;
      &lt;thead&gt;&lt;tr&gt;
        &lt;th&gt;Crawl&lt;/th&gt;&lt;th&gt;Status&lt;/th&gt;
        &lt;th&gt;Type&lt;/th&gt;&lt;th&gt;Size&lt;/th&gt;
      &lt;/tr&gt;&lt;/thead&gt;
      &lt;tbody&gt;
        \${captures.map(renderCaptureRow).join(<span style="color:#4ade80">''</span>)}
      &lt;/tbody&gt;
    &lt;/table&gt;
  \`</span>))
})</code></pre>

This is the entire rendering model. A function that returns an HTML string. The `layout()` wrapper adds the shell -- nav, head, footer. Each route builds its own content as a template literal. The Worker responds with `Content-Type: text/html` and the browser does what browsers do.

No bundle. No tree-shaking. No code-splitting. The deployed Worker is a single JavaScript file under 50 KB.

## Five Routes, Five Problems

The viewer has five main routes, and each one taught me something about Common Crawl's API surface.

### / -- Home

A search box. That is it. Type a URL, get redirected to `/url/{whatever}`. Type a domain, get redirected to `/domain/{whatever}`. The routing logic is four lines of string checking.

### /url/* -- URL Lookup

This queries the CDX API at `index.commoncrawl.org/{crawl}-index` for every crawl that has a capture of the given URL. The response is a table of captures -- crawl ID, HTTP status, content type, response size, and a link to view the WARC record.

The tricky part: you need to know which crawls exist before you can query them. That comes from the crawl list endpoint. More on that in a moment.

### /domain/:domain -- Domain Browse

Shows all pages captured for a domain across a specific crawl. The CDX API supports `matchType=domain` which returns all subdomains automatically -- so querying `example.com` also returns pages from `www.example.com`, `blog.example.com`, and so on.

Pagination is where this gets interesting. The CDX API supports a `page` parameter, and you can ask how many pages exist with `showNumPages=true`.

### The showNumPages Gotcha

<div class="note">
  <strong>This one cost me an hour.</strong> When you query the CDX API with <code>showNumPages=true</code>, the response is <code>{"pages": 42, "pageSize": 15, "blocks": 3}</code> -- a JSON object. My early code parsed it as a plain integer. <code>parseInt("{\"pages\": 42...")</code> returns <code>NaN</code>, which JavaScript happily coerces to <code>0</code> in most comparisons. The pagination silently broke -- every domain appeared to have zero additional pages.
</div>

The fix was obvious once I found it. Parse JSON, read the `pages` field. But "silently returns a wrong value instead of throwing" is the worst kind of bug. The viewer worked. It just showed one page of results for every domain, and I assumed some domains only had a few captures.

### /view -- WARC Viewer

This is the fun one. Given a WARC record location (filename, offset, length), the viewer fetches the compressed record from `data.commoncrawl.org` via a byte-range HTTP request, decompresses it in the Worker, and renders the content in the browser.

### /crawls -- Crawl List

Lists all available Common Crawl crawls with dates, page counts, and links to browse each one. Simple table, sourced from one API call.

## Where Does the Crawl List Come From?

<div class="note">
  <strong>The endpoint is <code>index.commoncrawl.org/collinfo.json</code>.</strong> Not <code>commoncrawl.org/collinfo.json</code>. Not <code>data.commoncrawl.org/collinfo.json</code>. The <code>index</code> subdomain. I burned 30 minutes hitting the wrong host and getting 404s before finding this in a GitHub issue comment.
</div>

The response is a JSON array of objects, each describing a crawl:

<pre><code>[
  {
    <span style="color:#4ade80">"id"</span>: <span style="color:#4ade80">"CC-MAIN-2026-04"</span>,
    <span style="color:#4ade80">"name"</span>: <span style="color:#4ade80">"January 2026 Index"</span>,
    <span style="color:#4ade80">"timegate"</span>: <span style="color:#4ade80">"https://index.commoncrawl.org/CC-MAIN-2026-04/"</span>,
    <span style="color:#4ade80">"cdx-api"</span>: <span style="color:#4ade80">"https://index.commoncrawl.org/CC-MAIN-2026-04-index"</span>
  },
  <span style="color:#888">// ... 100+ crawls going back to 2013</span>
]</code></pre>

Each crawl has a `cdx-api` field that gives you the query endpoint for that specific crawl's CDX index. You can also construct it yourself: `index.commoncrawl.org/{crawlId}-index`. I cache this list in KV because it changes at most once a month.

## Decompressing WARC Records on the Edge

WARC records in Common Crawl are individually gzip-compressed. When you fetch a byte range from a WARC file on `data.commoncrawl.org`, you get back a gzipped chunk containing the HTTP response headers and body for that capture.

Cloudflare Workers have `DecompressionStream` available in the runtime. This is a Web Streams API that does streaming gzip decompression -- no external libraries, no wasm modules, no buffer-the-whole-thing-and-decompress.

<pre><code><span style="color:#60a5fa">async function</span> decompressWarc(compressed: ReadableStream): Promise&lt;string&gt; {
  const ds = <span style="color:#4ade80">new</span> DecompressionStream(<span style="color:#4ade80">'gzip'</span>)
  const decompressed = compressed.pipeThrough(ds)
  const reader = decompressed.getReader()
  const chunks: Uint8Array[] = []

  <span style="color:#60a5fa">while</span> (<span style="color:#4ade80">true</span>) {
    const { done, value } = <span style="color:#60a5fa">await</span> reader.read()
    <span style="color:#60a5fa">if</span> (done) <span style="color:#60a5fa">break</span>
    chunks.push(value)
  }

  <span style="color:#60a5fa">return new</span> TextDecoder().decode(concat(chunks))
}</code></pre>

The decompressed content is a WARC record -- it starts with WARC headers, then the HTTP response headers, then the body. I split on the double CRLF boundaries to extract the HTML content, then render it in an iframe or a `<pre>` block depending on the content type.

This streams. The Worker does not buffer the entire compressed record before decompressing. For a 200 KB WARC record (typical for a web page), the overhead is negligible. For multi-megabyte records, the streaming matters -- you are not allocating a single huge buffer.

## KV Caching Strategy

Cloudflare KV is a global key-value store with edge caching. Reads are fast (single-digit milliseconds from the nearest edge). Writes propagate globally in under 60 seconds. The pricing is generous for read-heavy workloads.

I cache three things:

<table>
  <thead>
    <tr>
      <th>What</th>
      <th>Key Pattern</th>
      <th>TTL</th>
      <th>Why</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Crawl list</strong></td>
      <td><code>crawls</code></td>
      <td>24 hours</td>
      <td>Changes monthly at most. One API call saved per request.</td>
    </tr>
    <tr>
      <td><strong>CDX responses</strong></td>
      <td><code>cdx:{crawl}:{url}</code></td>
      <td>1 hour</td>
      <td>CC data is immutable once published. Cache aggressively.</td>
    </tr>
    <tr>
      <td><strong>Domain listings</strong></td>
      <td><code>domain:{crawl}:{domain}:{page}</code></td>
      <td>1 hour</td>
      <td>Same rationale -- immutable crawl data.</td>
    </tr>
  </tbody>
</table>

WARC records are not cached in KV. They are too large (up to several MB) and too numerous. The byte-range fetch from `data.commoncrawl.org` is fast enough that caching would add complexity without meaningful latency improvement. If I ever need WARC caching, I would use R2 (Cloudflare's object store) rather than KV.

## The API Layer

Every route has a corresponding `/api/` endpoint that returns JSON instead of HTML. Same data, different content type.

<pre><code><span style="color:#888">GET /api/health</span>        <span style="color:#60a5fa">→</span> { "status": "ok" }
<span style="color:#888">GET /api/crawls</span>        <span style="color:#60a5fa">→</span> [{ "id": "CC-MAIN-2026-04", ... }]
<span style="color:#888">GET /api/url/{url}</span>     <span style="color:#60a5fa">→</span> [{ "crawl": "...", "status": 200, ... }]
<span style="color:#888">GET /api/domain/{d}</span>    <span style="color:#60a5fa">→</span> [{ "url": "...", "timestamp": "...", ... }]
<span style="color:#888">GET /api/view?...</span>      <span style="color:#60a5fa">→</span> { "warc": "...", "headers": "...", "body": "..." }</code></pre>

The implementation is straightforward -- the API route calls the same data-fetching function as the HTML route, but returns `c.json(data)` instead of `c.html(layout(...))`. No duplication of business logic. I could have used content negotiation (check the `Accept` header), but separate route prefixes are simpler to reason about and easier to curl.

## What I Would Change

The viewer works. It has been live for a few weeks and handles a few hundred requests a day. But if I were starting over:

**I would add search.** Right now you need to know the exact URL or domain. A full-text search over the CDX data -- even just prefix matching -- would make the viewer dramatically more useful for exploration. The CDX API supports prefix queries, but I have not wired that up yet.

**I would handle large domains better.** A domain like `en.wikipedia.org` has millions of captures in CC. The current pagination works, but the UI is not designed for browsing millions of rows. Some kind of filtering (by status code, content type, date range) would help.

**I would not change the rendering approach.** SSR with template literals is the right call for this kind of tool. The pages are mostly tables and text. There is no interactivity that requires a JavaScript framework. The Worker responds in under 50ms for cached data and under 500ms for uncached CDX queries. React would add kilobytes of client JavaScript and a hydration step for zero user-facing benefit.

## The Stack

<table>
  <thead>
    <tr>
      <th>Component</th>
      <th>Choice</th>
      <th>Why</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Runtime</strong></td>
      <td>Cloudflare Workers</td>
      <td>Edge deployment, sub-millisecond cold starts, global distribution</td>
    </tr>
    <tr>
      <td><strong>Framework</strong></td>
      <td>Hono</td>
      <td>Lightweight, Workers-native, good routing</td>
    </tr>
    <tr>
      <td><strong>Rendering</strong></td>
      <td>SSR template literals</td>
      <td>Zero client JS, fast responses, simple to maintain</td>
    </tr>
    <tr>
      <td><strong>Cache</strong></td>
      <td>Cloudflare KV</td>
      <td>Edge-cached reads, simple API, generous free tier</td>
    </tr>
    <tr>
      <td><strong>Data source</strong></td>
      <td>Common Crawl CDX + WARC</td>
      <td>Public APIs, no authentication, immutable data</td>
    </tr>
    <tr>
      <td><strong>Decompression</strong></td>
      <td>DecompressionStream</td>
      <td>Native to Workers runtime, streaming, zero dependencies</td>
    </tr>
  </tbody>
</table>

The source is at [github.com/nicholasgasior/gopher-crawl](https://github.com/nicholasgasior/gopher-crawl) in `tools/cc-viewer/`. The KV namespace ID is `a412dc6f75e245e09c90944e156c5cf6` if you want to deploy your own instance.
