import { cssURL } from './asset'
import type { CDXEntry, Crawl, WARCRecord, DomainStats, URLGroup, CrawlStats, DomainSummary, ClusterEntry } from './types'
import { formatTimestamp, crawlToDate, statusClass } from './cc'

// ---- Icons (Lucide-style inline SVG) ----

const svg = {
  logo: `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`,
  logoBig: `<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`,
  search: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>`,
  globe: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/></svg>`,
  file: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>`,
  database: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5V19a9 3 0 0 0 18 0V5"/><path d="M3 12a9 3 0 0 0 18 0"/></svg>`,
  calendar: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 2v4"/><path d="M16 2v4"/><rect width="18" height="18" x="3" y="4" rx="2"/><path d="M3 10h18"/></svg>`,
  eye: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7z"/><circle cx="12" cy="12" r="3"/></svg>`,
  code: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>`,
  layers: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83z"/><path d="m22 17.65-9.17 4.16a2 2 0 0 1-1.66 0L2 17.65"/><path d="m22 12.65-9.17 4.16a2 2 0 0 1-1.66 0L2 12.65"/></svg>`,
  external: `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 3h6v6"/><path d="M10 14 21 3"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/></svg>`,
  arrow: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m9 18 6-6-6-6"/></svg>`,
  moon: `<svg class="icon-moon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/></svg>`,
  sun: `<svg class="icon-sun" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>`,
  hash: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="4" x2="20" y1="9" y2="9"/><line x1="4" x2="20" y1="15" y2="15"/><line x1="10" x2="8" y1="3" y2="21"/><line x1="16" x2="14" y1="3" y2="21"/></svg>`,
  hardDrive: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="22" x2="2" y1="12" y2="12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/><line x1="6" x2="6.01" y1="16" y2="16"/><line x1="10" x2="10.01" y1="16" y2="16"/></svg>`,
  download: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/></svg>`,
  clock: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>`,
}

// ---- Helpers ----

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

function fmtNum(n: number): string {
  if (n >= 1_000_000_000) return (n / 1_000_000_000).toFixed(1).replace(/\.0$/, '') + 'B'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1).replace(/\.0$/, '') + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1).replace(/\.0$/, '') + 'K'
  return n.toLocaleString('en-US')
}

function fmtBytes(n: number): string {
  if (n >= 1_099_511_627_776) return (n / 1_099_511_627_776).toFixed(1) + ' TB'
  if (n >= 1_073_741_824) return (n / 1_073_741_824).toFixed(1) + ' GB'
  if (n >= 1_048_576) return (n / 1_048_576).toFixed(1) + ' MB'
  if (n >= 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}

function truncURL(url: string, maxLen = 80): string {
  if (url.length <= maxLen) return url
  return url.substring(0, maxLen - 3) + '...'
}

/** Format CDX timestamp (20260115123456) to human-friendly relative or absolute */
function fmtDate(ts: string): string {
  if (!ts || ts.length < 8) return ts
  const y = ts.substring(0, 4)
  const m = ts.substring(4, 6)
  const d = ts.substring(6, 8)
  return `${monthName(parseInt(m))} ${parseInt(d)}, ${y}`
}

function monthName(m: number): string {
  return ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'][m - 1] || ''
}

function statusBadge(status: string): string {
  return `<span class="badge ${statusClass(status)}">${esc(status)}</span>`
}

function mimeBadge(mime: string): string {
  const short = mime.replace('text/', '').replace('application/', '').replace('image/', 'img/')
  return `<span class="badge st-mime">${esc(short)}</span>`
}

function simpleHash(s: string): string {
  let h = 0
  for (let i = 0; i < s.length; i++) h = ((h << 5) - h + s.charCodeAt(i)) | 0
  return (h >>> 0).toString(36)
}

function fmtCrawlDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  return `${months[d.getMonth()]} ${d.getDate()}`
}

// ---- Home Page ----

export function renderHomePage(): string {
  return `<div class="home">
<div class="home-logo">${svg.logoBig}</div>
<h1 class="home-title">Common Crawl Viewer</h1>
<p class="home-sub">Browse billions of web pages from the open Common Crawl archive</p>
<div class="home-search">
<form action="/search" method="get" class="search-form">
<div class="search-box">
${svg.search}
<input class="search-input" type="text" name="q" placeholder="Search by URL or domain..." autocomplete="off" autofocus>
</div>
</form>
</div>
<div class="home-chips">
<a href="/crawls" class="chip">${svg.layers} Browse Crawls</a>
<a href="/domains" class="chip">${svg.globe} Top Domains</a>
</div>
<div class="home-examples">
<span class="home-examples-label">Try:</span>
<a href="/domain/wikipedia.org">wikipedia.org</a>
<a href="/domain/github.com">github.com</a>
<a href="/domain/news.ycombinator.com">news.ycombinator.com</a>
<a href="/url/https://example.com/">example.com</a>
</div>
<div class="home-theme">
<button class="theme-toggle" onclick="T()" title="Toggle theme">${svg.moon}${svg.sun}</button>
</div>
</div>`
}

// ---- URL Lookup Page ----

export function renderURLPage(url: string, entries: CDXEntry[], crawl: string): string {
  let h = `<div class="page-header">
<div class="breadcrumb"><a href="/">Home</a> ${svg.arrow} <span>URL Lookup</span></div>
<h1 class="page-title">${svg.globe} ${esc(truncURL(url, 60))}</h1>
<div class="page-meta">
<span class="meta-item">${svg.database} ${crawl}</span>
<span class="meta-item">${svg.file} ${fmtNum(entries.length)} capture${entries.length !== 1 ? 's' : ''}</span>
</div>
</div>`

  if (entries.length === 0) {
    h += `<div class="empty-state"><h2>No captures found</h2><p>This URL wasn't found in ${esc(crawl)}. Try a different crawl or check the URL.</p></div>`
    return h
  }

  // Card list layout
  h += `<div class="card-list">`
  for (const e of entries) {
    const viewURL = `/view?file=${encodeURIComponent(e.filename)}&offset=${e.offset}&length=${e.length}&url=${encodeURIComponent(e.url)}`
    h += `<a href="${viewURL}" class="card-item card-item-link">
<div class="card-item-icon">${svg.file}</div>
<div class="card-item-body">
<div class="card-item-title">${fmtDate(e.timestamp)}</div>
<div class="card-item-sub">${statusBadge(e.status)} <span class="sep">&middot;</span> ${mimeBadge(e.mime)} <span class="sep">&middot;</span> ${fmtBytes(parseInt(e.length) || 0)}</div>
</div>
<div class="card-item-stats">
<span class="card-item-stat">${svg.eye} View</span>
</div>
</a>`
  }
  h += `</div>`
  return h
}

// ---- Domain Browse Page ----

export function renderDomainPage(domain: string, groups: URLGroup[], crawl: string, page: number, totalPages: number, stats: DomainStats): string {
  let h = `<div class="page-header">
<div class="breadcrumb"><a href="/">Home</a> ${svg.arrow} <a href="/domains">Domains</a> ${svg.arrow} <span>${esc(domain)}</span></div>
<h1 class="page-title">${svg.globe} ${esc(domain)}</h1>
<div class="page-meta">
<span class="meta-item">${svg.database} ${esc(crawl)}</span>
</div>
</div>`

  // KPI stats
  const ok = stats.statusCounts['200'] || 0
  const redirects = Object.entries(stats.statusCounts).filter(([s]) => s.startsWith('3')).reduce((a, [, c]) => a + c, 0)
  const errors = Object.entries(stats.statusCounts).filter(([s]) => parseInt(s) >= 400).reduce((a, [, c]) => a + c, 0)

  h += `<div class="kpi-grid">
<div class="kpi-card">
<div class="kpi-icon kpi-icon-blue">${svg.file}</div>
<div class="kpi-body">
<div class="kpi-value">${fmtNum(stats.totalPages)}</div>
<div class="kpi-label">Captures</div>
<div class="kpi-sub">${fmtNum(stats.uniquePaths)} unique URLs</div>
</div>
</div>
<div class="kpi-card">
<div class="kpi-icon kpi-icon-orange">${svg.database}</div>
<div class="kpi-body">
<div class="kpi-value">${fmtBytes(stats.totalSize)}</div>
<div class="kpi-label">Total Size</div>
<div class="kpi-sub">compressed</div>
</div>
</div>
<div class="kpi-card">
<div class="kpi-icon kpi-icon-green">${svg.eye}</div>
<div class="kpi-body">
<div class="kpi-value">${fmtNum(ok)}</div>
<div class="kpi-label">Success (2xx)</div>
<div class="kpi-sub">${stats.totalPages > 0 ? Math.round(ok / stats.totalPages * 100) : 0}%</div>
</div>
</div>
<div class="kpi-card">
<div class="kpi-icon kpi-icon-purple">${svg.layers}</div>
<div class="kpi-body">
<div class="kpi-value">${fmtNum(redirects + errors)}</div>
<div class="kpi-label">Redirects + Errors</div>
<div class="kpi-sub">${fmtNum(redirects)} 3xx, ${fmtNum(errors)} 4xx/5xx</div>
</div>
</div>
</div>`

  // Status + MIME breakdown
  const topStatuses = Object.entries(stats.statusCounts).sort((a, b) => b[1] - a[1])
  const topMimes = Object.entries(stats.mimeCounts).sort((a, b) => b[1] - a[1]).slice(0, 6)

  h += `<div class="breakdown-row">
<div class="breakdown-card">
<div class="breakdown-title">HTTP Status</div>`
  for (const [status, count] of topStatuses) {
    const pct = stats.totalPages > 0 ? Math.round(count / stats.totalPages * 100) : 0
    h += `<div class="breakdown-item"><span>${statusBadge(status)}</span><span class="breakdown-bar-wrap"><span class="breakdown-bar breakdown-bar-${statusClass(status)}" style="width:${Math.max(pct, 1)}%"></span></span><span class="breakdown-val">${fmtNum(count)}</span></div>`
  }
  h += `</div>
<div class="breakdown-card">
<div class="breakdown-title">Content Types</div>`
  for (const [mime, count] of topMimes) {
    const pct = stats.totalPages > 0 ? Math.round(count / stats.totalPages * 100) : 0
    h += `<div class="breakdown-item"><span>${mimeBadge(mime)}</span><span class="breakdown-bar-wrap"><span class="breakdown-bar" style="width:${Math.max(pct, 1)}%;background:var(--accent)"></span></span><span class="breakdown-val">${fmtNum(count)}</span></div>`
  }
  h += `</div>
</div>`

  if (groups.length === 0) {
    h += `<div class="empty-state"><h2>No pages found</h2><p>This domain wasn't found in ${esc(crawl)}.</p></div>`
    return h
  }

  // URL groups as card list
  h += `<div class="section-header">Pages</div>`
  h += `<div class="table-wrap"><table class="data-table">
<thead><tr>
<th>URL</th>
<th>Captures</th>
<th>Last Captured</th>
<th>Status</th>
<th>Type</th>
<th>Size</th>
</tr></thead>
<tbody>`

  for (const g of groups) {
    const latest = g.entries[0]
    const latestViewURL = `/view?file=${encodeURIComponent(latest.filename)}&offset=${latest.offset}&length=${latest.length}&url=${encodeURIComponent(latest.url)}`
    const hasMultiple = g.count > 1
    const rowId = 'g' + simpleHash(g.url)
    const toggleAttr = hasMultiple ? ` class="group-row expandable" onclick="toggleGroup('${rowId}')"` : ''

    h += `<tr${toggleAttr}>
<td class="url-cell">${hasMultiple ? `<span class="expand-icon" id="icon-${rowId}">+</span>` : '<span class="expand-spacer"></span>'}<a href="${latestViewURL}" title="${esc(g.url)}" onclick="event.stopPropagation()">${esc(truncURL(g.path, 55))}</a></td>
<td class="num">${hasMultiple ? `<span class="capture-count">${g.count}</span>` : '1'}</td>
<td>${fmtDate(g.latestTimestamp)}</td>
<td>${statusBadge(g.latestStatus)}</td>
<td>${mimeBadge(g.latestMime)}</td>
<td class="num">${fmtBytes(parseInt(g.latestLength) || 0)}</td>
</tr>`

    if (hasMultiple) {
      for (const e of g.entries) {
        const viewURL = `/view?file=${encodeURIComponent(e.filename)}&offset=${e.offset}&length=${e.length}&url=${encodeURIComponent(e.url)}`
        h += `<tr class="sub-row sub-${rowId}" style="display:none">
<td class="url-cell sub-indent"><a href="${viewURL}">${fmtDate(e.timestamp)}</a></td>
<td></td>
<td>${fmtDate(e.timestamp)}</td>
<td>${statusBadge(e.status)}</td>
<td>${mimeBadge(e.mime)}</td>
<td class="num">${fmtBytes(parseInt(e.length) || 0)}</td>
</tr>`
      }
    }
  }

  h += `</tbody></table></div>`

  if (totalPages > 1) {
    h += renderPagination(`/domain/${domain}?crawl=${encodeURIComponent(crawl)}`, page, totalPages, true)
  }

  return h
}

// ---- Domains Listing Page ----

export function renderDomainsPage(crawlID: string, domains: DomainSummary[], page: number, totalPages: number, totalDomains: number, totalEntries: number): string {
  let h = `<div class="page-header">
<div class="breadcrumb"><a href="/">Home</a> ${svg.arrow} <span>Domains</span></div>
<h1 class="page-title">${svg.globe} Top Domains</h1>
<div class="page-meta">
<span class="meta-item">${svg.database} ${esc(crawlID)}</span>
<span class="meta-item">${svg.layers} Page ${page + 1} of ${fmtNum(totalPages)}</span>
</div>
</div>`

  if (totalDomains > 0) {
    h += `<div class="kpi-grid">
<div class="kpi-card">
<div class="kpi-icon kpi-icon-blue">${svg.globe}</div>
<div class="kpi-body">
<div class="kpi-value">${fmtNum(totalDomains)}</div>
<div class="kpi-label">Unique Domains</div>
<div class="kpi-sub">from cluster.idx</div>
</div>
</div>
<div class="kpi-card">
<div class="kpi-icon kpi-icon-green">${svg.file}</div>
<div class="kpi-body">
<div class="kpi-value">${fmtNum(totalEntries)}</div>
<div class="kpi-label">Index Pages</div>
<div class="kpi-sub">~${fmtNum(totalEntries * 3000)} est. captures</div>
</div>
</div>
</div>`
  }

  if (domains.length === 0) {
    h += `<div class="empty-state"><h2>No domains found</h2><p>Cluster index data not loaded for this crawl. Run the processing script first.</p></div>`
    return h
  }

  // Kaggle-style card list for domains
  h += `<div class="card-list">`
  for (const d of domains) {
    const estCaptures = d.pages * 3000
    h += `<a href="/domain/${esc(d.domain)}?crawl=${esc(crawlID)}" class="card-item card-item-link">
<div class="card-item-icon">${svg.globe}</div>
<div class="card-item-body">
<div class="card-item-title">${esc(d.domain)}</div>
<div class="card-item-sub">${fmtNum(d.pages)} index pages <span class="sep">&middot;</span> ~${fmtNum(estCaptures)} captures</div>
</div>
<div class="card-item-stats">
<span class="card-item-stat">${svg.file} <strong>${fmtNum(d.pages)}</strong></span>
</div>
</a>`
  }
  h += `</div>`

  if (totalPages > 1) {
    h += renderPagination(`/domains?crawl=${encodeURIComponent(crawlID)}`, page, totalPages, true)
  }

  return h
}

// ---- WARC Content Viewer ----

export function renderViewPage(record: WARCRecord, url: string, filename: string): string {
  const waybackURL = `https://web.archive.org/web/${url}`
  const isHTML = record.contentType.includes('text/html') || record.body.trimStart().startsWith('<')

  let h = `<div class="page-header">
<div class="breadcrumb"><a href="/">Home</a> ${svg.arrow} <span>View</span></div>
<h1 class="page-title">${svg.eye} Page Capture</h1>
<div class="page-meta">
<span class="meta-item">${svg.globe} <a href="${esc(url)}" target="_blank" rel="noopener">${esc(truncURL(url, 50))} ${svg.external}</a></span>
</div>
</div>`

  const isTruncated = record.httpHeaders['X-CC-Viewer-Truncated']
  if (isTruncated) {
    h += `<div class="notice notice-warn">Body truncated to 512 KB (original: ${fmtBytes(record.contentLength)}). <a href="${esc(waybackURL)}" target="_blank" rel="noopener">View full page on Wayback Machine ${svg.external}</a></div>`
  }

  h += `<div class="meta-grid">
<div class="meta-card"><div class="meta-label">Status</div><div class="meta-val">${statusBadge(String(record.httpStatus))}</div></div>
<div class="meta-card"><div class="meta-label">Content Type</div><div class="meta-val">${esc(record.contentType || 'unknown')}</div></div>
<div class="meta-card"><div class="meta-label">Size</div><div class="meta-val">${fmtBytes(record.contentLength)}</div></div>
<div class="meta-card"><div class="meta-label">Captured</div><div class="meta-val">${esc(record.date || 'unknown')}</div></div>
<div class="meta-card"><div class="meta-label">WARC Type</div><div class="meta-val">${esc(record.warcType)}</div></div>
<div class="meta-card"><div class="meta-label">Wayback</div><div class="meta-val"><a href="${esc(waybackURL)}" target="_blank" rel="noopener">View ${svg.external}</a></div></div>
</div>`

  h += `<div class="view-tabs">
<button class="view-tab active" onclick="showTab('preview')">Preview</button>
<button class="view-tab" onclick="showTab('source')">${svg.code} Source</button>
<button class="view-tab" onclick="showTab('headers')">${svg.hash} Headers</button>
</div>`

  if (isHTML) {
    const sanitized = record.body
      .replace(/<script[\s\S]*?<\/script>/gi, '')
      .replace(/\son\w+="[^"]*"/gi, '')
      .replace(/\son\w+='[^']*'/gi, '')
    h += `<div class="tab-content" id="tab-preview"><iframe class="preview-frame" sandbox="allow-same-origin" srcdoc="${esc(sanitized)}"></iframe></div>`
  } else {
    h += `<div class="tab-content" id="tab-preview"><pre class="code-block">${esc(record.body.substring(0, 50000))}</pre></div>`
  }

  h += `<div class="tab-content hidden" id="tab-source"><pre class="code-block">${esc(record.body.substring(0, 100000))}</pre></div>`

  let headersHTML = ''
  for (const [k, v] of Object.entries(record.httpHeaders)) {
    headersHTML += `<tr><td>${esc(k)}</td><td>${esc(v)}</td></tr>`
  }
  h += `<div class="tab-content hidden" id="tab-headers"><table class="data-table headers-table"><thead><tr><th>Header</th><th>Value</th></tr></thead><tbody>${headersHTML}</tbody></table></div>`

  return h
}

// ---- Crawl List Page ----

export function renderCrawlsPage(crawls: (Crawl & { stats?: CrawlStats })[]): string {
  let h = `<div class="page-header">
<div class="breadcrumb"><a href="/">Home</a> ${svg.arrow} <span>Crawls</span></div>
<h1 class="page-title">${svg.layers} Available Crawls</h1>
<div class="page-meta"><span class="meta-item">${svg.database} ${crawls.length} crawls</span></div>
</div>
<div class="crawl-grid">`

  for (const c of crawls) {
    const s = c.stats
    const dateRange = c.from && c.to
      ? `${fmtCrawlDate(c.from)} – ${fmtCrawlDate(c.to)}`
      : crawlToDate(c.id)

    h += `<a href="/crawl/${esc(c.id)}" class="crawl-card">
<div class="crawl-card-head">
<div class="crawl-id">${esc(c.id)}</div>
${s ? `<span class="crawl-stat-size">${fmtBytes(s.estimatedSizeBytes)}</span>` : ''}
</div>
<div class="crawl-name">${esc(c.name)}</div>
<div class="crawl-date">${svg.calendar} ${esc(dateRange)}</div>`

    if (s) {
      h += `<div class="crawl-stats-row">
<span class="crawl-stat">${svg.hardDrive} <span class="crawl-stat-val">${fmtNum(s.warcFiles)}</span> WARCs</span>
<span class="crawl-stat">${svg.layers} <span class="crawl-stat-val">${fmtNum(s.segments)}</span> segments</span>
<span class="crawl-stat">${svg.file} <span class="crawl-stat-val">${fmtNum(s.indexFiles)}</span> index</span>
</div>`
    }

    h += `</a>`
  }

  h += `</div>`
  return h
}

// ---- Crawl Detail Page ----

export function renderCrawlDetailPage(crawl: Crawl, stats: CrawlStats): string {
  const dateRange = crawl.from && crawl.to
    ? `${fmtCrawlDate(crawl.from)} – ${fmtCrawlDate(crawl.to)}`
    : crawlToDate(crawl.id)

  let h = `<div class="page-header">
<div class="breadcrumb"><a href="/">Home</a> ${svg.arrow} <a href="/crawls">Crawls</a> ${svg.arrow} <span>${esc(crawl.id)}</span></div>
<h1 class="page-title">${svg.database} ${esc(crawl.id)}</h1>
<div class="page-meta">
<span class="meta-item">${svg.calendar} ${esc(dateRange)}</span>
<span class="meta-item">${esc(crawl.name)}</span>
</div>
</div>`

  h += `<div class="kpi-grid">
<div class="kpi-card">
<div class="kpi-icon kpi-icon-blue">${svg.hardDrive}</div>
<div class="kpi-body">
<div class="kpi-value">${fmtNum(stats.warcFiles)}</div>
<div class="kpi-label">WARC Files</div>
<div class="kpi-sub">~${fmtBytes(stats.avgWARCSize)} avg</div>
</div>
</div>
<div class="kpi-card">
<div class="kpi-icon kpi-icon-green">${svg.layers}</div>
<div class="kpi-body">
<div class="kpi-value">${fmtNum(stats.segments)}</div>
<div class="kpi-label">Segments</div>
<div class="kpi-sub">${fmtNum(Math.round(stats.warcFiles / stats.segments))}/seg</div>
</div>
</div>
<div class="kpi-card">
<div class="kpi-icon kpi-icon-purple">${svg.file}</div>
<div class="kpi-body">
<div class="kpi-value">${fmtNum(stats.indexFiles)}</div>
<div class="kpi-label">Index Files</div>
<div class="kpi-sub">CDX index</div>
</div>
</div>
<div class="kpi-card">
<div class="kpi-icon kpi-icon-orange">${svg.database}</div>
<div class="kpi-body">
<div class="kpi-value">${fmtBytes(stats.estimatedSizeBytes)}</div>
<div class="kpi-label">Total Size</div>
<div class="kpi-sub">compressed</div>
</div>
</div>
</div>`

  h += `<div class="nav-grid">
<a href="/crawl/${esc(crawl.id)}/files" class="nav-card">
<div class="nav-card-icon">${svg.hardDrive}</div>
<div class="nav-card-body">
<div class="nav-card-title">Browse WARC Files</div>
<div class="nav-card-desc">${fmtNum(stats.warcFiles)} archive files across ${fmtNum(stats.segments)} segments</div>
</div>
${svg.arrow}
</a>
<a href="/crawl/${esc(crawl.id)}/cdx" class="nav-card">
<div class="nav-card-icon">${svg.code}</div>
<div class="nav-card-body">
<div class="nav-card-title">Browse CDX Index</div>
<div class="nav-card-desc">Search the CDX index by URL prefix</div>
</div>
${svg.arrow}
</a>
</div>`

  h += `<div class="section-header">Search this crawl</div>
<form action="/search" method="get">
<input type="hidden" name="crawl" value="${esc(crawl.id)}">
<div class="search-box" style="max-width:520px">
${svg.search}
<input class="search-input" type="text" name="q" placeholder="Search by URL or domain..." autocomplete="off">
</div>
</form>`

  return h
}

// ---- Crawl Files Page ----

export function renderCrawlFilesPage(crawlID: string, files: string[], page: number, totalPages: number, totalFiles: number): string {
  let h = `<div class="page-header">
<div class="breadcrumb"><a href="/">Home</a> ${svg.arrow} <a href="/crawls">Crawls</a> ${svg.arrow} <a href="/crawl/${esc(crawlID)}">${esc(crawlID)}</a> ${svg.arrow} <span>Files</span></div>
<h1 class="page-title">${svg.hardDrive} WARC Files</h1>
<div class="page-meta">
<span class="meta-item">${svg.file} ${fmtNum(totalFiles)} files</span>
<span class="meta-item">${svg.layers} Page ${page + 1} of ${totalPages}</span>
</div>
</div>`

  h += `<div class="card-list">`
  for (let i = 0; i < files.length; i++) {
    const f = files[i]
    const idx = page * 100 + i + 1
    const parts = f.split('/')
    const fname = parts[parts.length - 1]
    const segPart = parts.length >= 4 ? parts[3] : ''

    h += `<div class="card-item">
<div class="card-item-icon">${svg.hardDrive}</div>
<div class="card-item-body">
<div class="card-item-title" title="${esc(f)}">${esc(fname)}</div>
<div class="card-item-sub">Segment ${esc(segPart)}</div>
</div>
<div class="card-item-stats">
<span class="card-item-stat">#${idx}</span>
</div>
</div>`
  }
  h += `</div>`

  if (totalPages > 1) {
    h += renderPagination(`/crawl/${crawlID}/files`, page, totalPages)
  }

  return h
}

// ---- CDX Browse Page ----

export function renderCrawlCDXPage(crawlID: string, entries: CDXEntry[], page: number, totalPages: number, prefix: string, clusterEntries: ClusterEntry[] = [], clusterPage = 0, clusterTotalPages = 0): string {
  let h = `<div class="page-header">
<div class="breadcrumb"><a href="/">Home</a> ${svg.arrow} <a href="/crawls">Crawls</a> ${svg.arrow} <a href="/crawl/${esc(crawlID)}">${esc(crawlID)}</a> ${svg.arrow} <span>CDX</span></div>
<h1 class="page-title">${svg.code} CDX Index</h1>
<div class="page-meta">
<span class="meta-item">${svg.database} ${esc(crawlID)}</span>
${prefix ? `<span class="meta-item">${svg.search} ${fmtNum(entries.length)} results</span>` : ''}
</div>
</div>`

  h += `<form action="/crawl/${esc(crawlID)}/cdx" method="get" style="margin-bottom:28px">
<div class="search-box" style="max-width:520px">
${svg.search}
<input class="search-input" type="text" name="prefix" value="${esc(prefix)}" placeholder="Search by URL prefix (e.g. github.com)..." autocomplete="off" autofocus>
</div>
</form>`

  if (!prefix) {
    if (clusterEntries.length > 0) {
      h += `<div class="section-header">Index Page Directory</div>`
      h += `<div class="card-list">`
      for (const e of clusterEntries) {
        h += `<a href="/domain/${esc(e.domain)}?crawl=${esc(crawlID)}" class="card-item card-item-link">
<div class="card-item-icon">${svg.code}</div>
<div class="card-item-body">
<div class="card-item-title">${esc(e.domain)}</div>
<div class="card-item-sub">${esc(truncURL(e.surtKey, 50))} <span class="sep">&middot;</span> ${esc(e.cdxFile)}</div>
</div>
<div class="card-item-stats">
<span class="card-item-stat">${svg.hash} <strong>${fmtNum(e.pageNum)}</strong></span>
</div>
</a>`
      }
      h += `</div>`
      if (clusterTotalPages > 1) {
        h += renderPagination(`/crawl/${crawlID}/cdx`, clusterPage, clusterTotalPages)
      }
    } else {
      h += `<div class="empty-state"><h2>Browse the CDX Index</h2><p>Enter a URL prefix to search through the CDX index entries for this crawl.</p>
<p style="font-size:13px;margin-top:16px">Try: <a href="/crawl/${esc(crawlID)}/cdx?prefix=github.com" style="color:var(--accent-fg)">github.com</a> · <a href="/crawl/${esc(crawlID)}/cdx?prefix=wikipedia.org" style="color:var(--accent-fg)">wikipedia.org</a> · <a href="/crawl/${esc(crawlID)}/cdx?prefix=example.com" style="color:var(--accent-fg)">example.com</a></p></div>`
    }
    return h
  }

  if (entries.length === 0) {
    h += `<div class="empty-state"><h2>No results</h2><p>No entries found for prefix "${esc(prefix)}" in ${esc(crawlID)}.</p></div>`
    return h
  }

  // CDX results as card list
  h += `<div class="card-list">`
  for (const e of entries) {
    const viewURL = `/view?file=${encodeURIComponent(e.filename)}&offset=${e.offset}&length=${e.length}&url=${encodeURIComponent(e.url)}`
    let shortURL = e.url
    try {
      const u = new URL(e.url)
      shortURL = u.host + u.pathname + (u.search || '')
    } catch { /* keep full */ }

    h += `<a href="${viewURL}" class="card-item card-item-link">
<div class="card-item-body">
<div class="card-item-title" title="${esc(e.url)}">${esc(truncURL(shortURL, 65))}</div>
<div class="card-item-sub">${statusBadge(e.status)} <span class="sep">&middot;</span> ${mimeBadge(e.mime)} <span class="sep">&middot;</span> ${fmtDate(e.timestamp)} <span class="sep">&middot;</span> ${fmtBytes(parseInt(e.length) || 0)}</div>
</div>
<div class="card-item-stats">
<span class="card-item-stat">${svg.eye} View</span>
</div>
</a>`
  }
  h += `</div>`

  if (totalPages > 1) {
    h += renderPagination(`/crawl/${crawlID}/cdx?prefix=${encodeURIComponent(prefix)}`, page, totalPages, true)
  }

  return h
}

// ---- Shared Pagination ----

function renderPagination(basePath: string, page: number, totalPages: number, hasQueryParams = false): string {
  const sep = hasQueryParams ? '&' : '?'
  let h = `<div class="pagination">`
  if (page > 0) {
    h += `<a href="${basePath}${sep}page=${page - 1}" class="pag-btn">Previous</a>`
  }
  h += `<span class="pag-info">Page ${page + 1} of ${fmtNum(totalPages)}</span>`
  if (page < totalPages - 1) {
    h += `<a href="${basePath}${sep}page=${page + 1}" class="pag-btn">Next</a>`
  }
  h += `</div>`
  return h
}

// ---- Layout ----

const themeScript = `<script>(function(){var t=localStorage.getItem('t');if(!t)t=matchMedia('(prefers-color-scheme:dark)').matches?'d':'l';document.documentElement.dataset.t=t})();function T(){var h=document.documentElement,n=h.dataset.t==='d'?'l':'d';h.dataset.t=n;localStorage.setItem('t',n)}</script>`

const tabScript = `<script>function showTab(id){document.querySelectorAll('.tab-content').forEach(function(el){el.classList.add('hidden')});document.querySelectorAll('.view-tab').forEach(function(el){el.classList.remove('active')});var tab=document.getElementById('tab-'+id);if(tab)tab.classList.remove('hidden');event.target.classList.add('active')}
function toggleGroup(id){var rows=document.querySelectorAll('.sub-'+id);var icon=document.getElementById('icon-'+id);var show=rows[0]&&rows[0].style.display==='none';rows.forEach(function(r){r.style.display=show?'':'none'});if(icon)icon.textContent=show?'\u2212':'+'}</script>`

function renderTopBar(query?: string): string {
  return `<div class="topbar">
<a href="/" class="topbar-logo">${svg.logo}<span>CC Viewer</span></a>
<form action="/search" method="get" class="topbar-search">
<div class="search-box compact">
${svg.search}
<input class="search-input" type="text" name="q" placeholder="Search URL or domain..." value="${query ? esc(query) : ''}" autocomplete="off">
</div>
</form>
<div class="topbar-actions">
<a href="/domains" class="topbar-link">Domains</a>
<a href="/crawls" class="topbar-link">Crawls</a>
<button class="theme-toggle" onclick="T()" title="Toggle theme">${svg.moon}${svg.sun}</button>
</div>
</div>`
}

export function renderLayout(title: string, content: string, opts: { isHome?: boolean; query?: string } = {}): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>${esc(title)}</title>
<meta name="description" content="Browse billions of web pages from Common Crawl archives">
<link rel="stylesheet" href="${cssURL}">
<link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%231a73e8' stroke-width='2'><circle cx='12' cy='12' r='10'/><path d='M2 12h20'/><path d='M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z'/></svg>">
${themeScript}
</head>
<body>
<div class="wrap">${opts.isHome ? '' : renderTopBar(opts.query)}${content}</div>
${tabScript}
</body>
</html>`
}

export function renderError(title: string, message: string): string {
  return renderLayout(title, `<div class="empty-state"><h2>${esc(title)}</h2><p>${esc(message)}</p><a href="/" class="btn-primary">Go Home</a></div>`)
}
