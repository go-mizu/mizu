// ===================================================================
// Tab: Domains
// ===================================================================

let domainsFilterTimer = null;
let domainsSyncPollTimer = null;

async function renderDomains() {
  state.currentPage = 'domains';
  if (!state.domainPage) state.domainPage = 1;
  if (!state.domainSort) state.domainSort = 'count';
  if (state.domainQ === undefined) state.domainQ = '';
  state.domainOverview = null;
  clearTimeout(domainsSyncPollTimer);

  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <h1 class="page-title">Domains</h1>
      </div>
      <div id="domains-cc-fetch-pane" class="mb-4"></div>
      <div id="domains-overview-pane" class="mb-4"></div>
      <div class="surface p-4">
        <div id="domains-content">${domainsSkeleton()}</div>
      </div>
    </div>`;

  renderCCDomainFetchForm();
  await loadDomains();
}

async function loadDomains(page) {
  if (page !== undefined) state.domainPage = page;
  const el = $('domains-content');
  if (!el) return;

  try {
    const data = await apiDomains({
      sort: state.domainSort,
      page: state.domainPage,
      q: state.domainQ,
    });
    if (state.currentPage !== 'domains') return;
    if (data.overview) state.domainOverview = data.overview;
    renderDomainsOverview(state.domainOverview, data.syncing, data.locked);
    if (data.locked) {
      scheduleDomainsSyncPoll(); // retry until lock is released
    } else {
      renderDomainsTable(data);
      if (data.syncing) scheduleDomainsSyncPoll();
    }
  } catch(e) {
    const el2 = $('domains-content');
    if (el2) el2.innerHTML = `<div class="text-xs text-red-400 py-4">${esc(e.message)}</div>`;
  }
}

// Poll only the overview stats while syncing — never re-renders the domain table.
function scheduleDomainsSyncPoll() {
  domainsSyncPollTimer = setTimeout(async () => {
    if (state.currentPage !== 'domains') return;
    try {
      const data = await apiDomainsOverview();
      if (state.currentPage !== 'domains') return;
      if (data.overview) state.domainOverview = data.overview;
      renderDomainsOverview(state.domainOverview, data.syncing, data.locked);
      if (data.locked || data.syncing) {
        scheduleDomainsSyncPoll();
      } else {
        // Sync finished (or lock released) — reload the full table once.
        loadDomains();
      }
    } catch(e) {
      scheduleDomainsSyncPoll(); // retry on transient errors
    }
  }, 4000);
}

// ── Overview pane (top of domain list) ───────────────────────────────────────

function renderDomainsOverview(ov, syncing, locked) {
  const el = $('domains-overview-pane');
  if (!el) return;
  if (!ov && !syncing && !locked) { el.innerHTML = ''; return; }

  if (locked) {
    el.innerHTML = `<div class="surface px-4 py-3 text-xs font-mono" style="border-color:rgba(248,113,113,0.4);color:#f87171;background:rgba(248,113,113,0.05)">
      \u26a0 domains.duckdb is locked by another search process. Waiting for it to exit\u2026
    </div>`;
    return;
  }

  if (syncing && !ov) {
    el.innerHTML = `<div class="surface px-4 py-3 text-xs font-mono" style="border-color:rgba(96,165,250,0.4);color:#93c5fd;background:rgba(96,165,250,0.05)">
      Indexing domain counts from parquet files\u2026 results will appear shortly.
    </div>`;
    return;
  }

  const totalURLs = ov.total_urls || 0;
  const totalDomains = ov.total_domains || 0;
  const buckets = ov.size_buckets || [];
  const bucketTotal = buckets.reduce((s, b) => s + (b.count || 0), 0) || 1;

  // Color palette for donut segments.
  const palette = ['#6366f1','#34d399','#fbbf24','#f87171','#c084fc'];

  // Build donut SVG.
  const donut = buildDonut(buckets.map((b,i) => ({
    label: b.label,
    value: b.count,
    color: palette[i % palette.length],
  })), 52);

  el.innerHTML = `
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <!-- Stat: total domains -->
      <div class="surface px-4 py-3 flex flex-col gap-1">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Domains</div>
        <div class="text-xl font-semibold tracking-tight">${totalDomains.toLocaleString()}</div>
      </div>
      <!-- Stat: total URLs -->
      <div class="surface px-4 py-3 flex flex-col gap-1">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Total URLs</div>
        <div class="text-xl font-semibold tracking-tight">${fmtBigNum(totalURLs)}</div>
      </div>
      <!-- Pages-per-domain donut -->
      <div class="surface px-4 py-3 col-span-2 flex items-center gap-4">
        <div class="shrink-0">${donut}</div>
        <div class="min-w-0">
          <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider mb-2">Pages per Domain</div>
          <div class="space-y-1">
            ${buckets.map((b, i) => `
              <div class="flex items-center gap-2 text-[11px]">
                <span class="inline-block w-2 h-2 rounded-full shrink-0" style="background:${palette[i%palette.length]}"></span>
                <span class="ui-subtle font-mono truncate">${esc(b.label)}</span>
                <span class="ml-auto font-mono ui-subtle whitespace-nowrap">${b.count.toLocaleString()}</span>
                <span class="font-mono ui-subtle w-8 text-right">${((b.count/bucketTotal)*100).toFixed(0)}%</span>
              </div>`).join('')}
          </div>
        </div>
      </div>
    </div>
    ${syncing ? `<div class="mt-2 text-[10px] font-mono" style="color:#60a5fa">\u25cf Syncing in background\u2026</div>` : ''}`;
}

// Build a simple SVG donut chart (no deps).
function buildDonut(segments, r) {
  const total = segments.reduce((s, x) => s + x.value, 0) || 1;
  const cx = r + 4, cy = r + 4, size = (r + 4) * 2;
  const stroke = r * 0.38;
  const ri = r - stroke / 2;

  let angle = -Math.PI / 2;
  const paths = segments.map(s => {
    const sweep = (s.value / total) * 2 * Math.PI;
    const x1 = cx + ri * Math.cos(angle);
    const y1 = cy + ri * Math.sin(angle);
    angle += sweep;
    const x2 = cx + ri * Math.cos(angle);
    const y2 = cy + ri * Math.sin(angle);
    const large = sweep > Math.PI ? 1 : 0;
    return `<path d="M${x1.toFixed(2)},${y1.toFixed(2)} A${ri},${ri} 0 ${large},1 ${x2.toFixed(2)},${y2.toFixed(2)}"
      fill="none" stroke="${s.color}" stroke-width="${stroke}" stroke-linecap="butt"/>`;
  }).join('');

  return `<svg width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">${paths}</svg>`;
}

function fmtBigNum(n) {
  if (n >= 1e9) return (n / 1e9).toFixed(1) + 'B';
  if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M';
  if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K';
  return n.toLocaleString();
}

// ── Domain list table ─────────────────────────────────────────────────────────

function renderDomainsTable(data) {
  const el = $('domains-content');
  if (!el) return;

  const domains = data.domains || [];
  const total = data.total || 0;
  const page = data.page || 1;
  const pageSize = data.page_size || 100;
  const totalPages = Math.ceil(total / pageSize);
  const maxCount = domains.reduce((m, d) => Math.max(m, d.count || 0), 0) || 1;

  el.innerHTML = `
    <div class="flex items-center gap-3 mb-4 flex-wrap">
      <span class="meta-line">${total.toLocaleString()} domain${total !== 1 ? 's' : ''}</span>
      <input id="domains-filter" type="search" placeholder="Filter domains\u2026"
        value="${esc(state.domainQ || '')}"
        class="ui-input text-xs px-2 py-1 w-40 sm:w-56"
        oninput="debounceDomainFilter(this.value)">
      <select class="ui-input text-xs px-2 py-1 ml-auto"
        onchange="state.domainSort=this.value;loadDomains(1)">
        <option value="count" ${(state.domainSort||'count')==='count'?'selected':''}>Count \u2193</option>
        <option value="alpha" ${state.domainSort==='alpha'?'selected':''}>Domain A\u2013Z</option>
      </select>
    </div>
    ${domains.length === 0 ? `
      <div class="ui-empty">${state.domainQ ? 'No domains match filter.' : 'No domain data yet \u2014 download parquet files first.'}</div>
    ` : `
    <table class="w-full text-xs ui-table">
      <thead>
        <tr class="text-left">
          <th class="pb-2 pr-3 font-medium">Domain</th>
          <th class="pb-2 font-medium text-right">URLs</th>
        </tr>
      </thead>
      <tbody>
        ${domains.map((d, i) => `
          <tr class="file-row anim-fade-up" style="animation-delay:${Math.min(i,20)*8}ms">
            <td class="py-2 pr-3">
              <div class="flex items-center gap-2">
                <a href="#/domains/${encodeURIComponent(d.domain)}"
                  class="ui-link font-mono font-medium shrink-0">${esc(d.domain)}</a>
              </div>
              <div class="mt-1.5 progress-track" style="height:2px">
                <div class="progress-fill" style="width:${Math.max(1,(d.count/maxCount)*100).toFixed(1)}%"></div>
              </div>
            </td>
            <td class="py-2 text-right font-mono ui-subtle whitespace-nowrap align-top">
              ${(d.count||0).toLocaleString()}
            </td>
          </tr>`).join('')}
      </tbody>
    </table>
    ${totalPages > 1 ? `
    <div class="flex items-center justify-between mt-4 text-xs">
      <button onclick="loadDomains(${page-1})" ${page<=1?'disabled':''} class="ui-btn px-3 py-1.5">&larr; Prev</button>
      <span class="ui-subtle">Page ${page} of ${totalPages}</span>
      <button onclick="loadDomains(${page+1})" ${page>=totalPages?'disabled':''} class="ui-btn px-3 py-1.5">Next &rarr;</button>
    </div>` : ''}
    `}`;
}

function debounceDomainFilter(val) {
  state.domainQ = val;
  clearTimeout(domainsFilterTimer);
  domainsFilterTimer = setTimeout(() => loadDomains(1), 300);
}

// ── Common Crawl Domain Fetch (separate code path + cache) ──────────────────

function renderCCDomainFetchForm() {
  const el = $('domains-cc-fetch-pane');
  if (!el) return;
  const domain = state.ccFetchDomain || '';
  const crawl = state.ccFetchCrawl || '';
  const maxURLs = state.ccFetchMaxURLs || 20000;
  el.innerHTML = `
    <div class="surface p-4">
      <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider mb-3">Common Crawl URL Lookup (CDX API)</div>
      <form class="grid grid-cols-1 sm:grid-cols-12 gap-2 items-end" onsubmit="submitCCDomainFetch(event)">
        <label class="sm:col-span-4 block">
          <div class="text-[10px] font-mono ui-subtle mb-1">Domain</div>
          <input id="cc-domain-input" type="text" class="ui-input text-xs px-2 py-2 w-full"
            placeholder="example.com" value="${esc(domain)}" required>
        </label>
        <label class="sm:col-span-3 block">
          <div class="text-[10px] font-mono ui-subtle mb-1">Crawl ID (optional)</div>
          <input id="cc-crawl-input" type="text" class="ui-input text-xs px-2 py-2 w-full"
            placeholder="CC-MAIN-2026-08" value="${esc(crawl)}">
        </label>
        <label class="sm:col-span-2 block">
          <div class="text-[10px] font-mono ui-subtle mb-1">Max URLs</div>
          <input id="cc-max-urls-input" type="number" min="100" max="200000"
            class="ui-input text-xs px-2 py-2 w-full" value="${String(maxURLs)}">
        </label>
        <div class="sm:col-span-3">
          <button id="cc-fetch-submit" type="submit" class="ui-btn px-3 py-2 w-full">Fetch & Open</button>
        </div>
      </form>
      <div id="cc-fetch-status" class="text-[11px] ui-subtle mt-2"></div>
    </div>`;
}

async function submitCCDomainFetch(ev) {
  ev.preventDefault();
  const domainInput = $('cc-domain-input');
  const crawlInput = $('cc-crawl-input');
  const maxURLsInput = $('cc-max-urls-input');
  const statusEl = $('cc-fetch-status');
  const btn = $('cc-fetch-submit');
  if (!domainInput || !statusEl || !btn) return;

  const domain = (domainInput.value || '').trim();
  const crawl = (crawlInput && crawlInput.value ? crawlInput.value : '').trim();
  const maxURLs = parseInt(maxURLsInput && maxURLsInput.value ? maxURLsInput.value : '20000', 10) || 20000;
  state.ccFetchDomain = domain;
  state.ccFetchCrawl = crawl;
  state.ccFetchMaxURLs = maxURLs;

  btn.disabled = true;
  statusEl.textContent = 'Fetching from Common Crawl CDX API...';
  try {
    const resp = await apiCCDomainFetch({ domain, crawl_id: crawl, max_urls: maxURLs });
    const crawlID = resp.crawl_id || crawl;
    statusEl.textContent = `Cached ${fmtBigNum(resp.total_urls || 0)} URLs from ${crawlID}${resp.truncated ? ' (truncated by max URLs)' : ''}. Opening detail...`;
    navigateTo(`/domains/cc/${encodeURIComponent(resp.domain || domain)}?crawl=${encodeURIComponent(crawlID || '')}`);
  } catch (e) {
    statusEl.textContent = e && e.message ? e.message : 'Failed to fetch domain URLs';
  } finally {
    btn.disabled = false;
  }
}

async function renderCCDomainDetail(domain, crawl) {
  state.currentPage = 'domain-cc-detail';
  state.ccDomain = domain;
  state.ccDomainCrawl = crawl || state.ccDomainCrawl || '';
  state.ccDomainPage = 1;
  if (!state.ccDomainSort) state.ccDomainSort = 'url';
  if (state.ccDomainQ === undefined) state.ccDomainQ = '';

  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <div class="flex items-center gap-2 text-xs font-mono ui-subtle mb-1">
          <a href="#/domains" class="ui-link">Domains</a>
          <span>/</span>
          <span>CommonCrawl</span>
          <span>/</span>
          <span class="font-medium" style="color:var(--text)">${esc(domain)}</span>
        </div>
        <h1 class="page-title font-mono">${esc(domain)}</h1>
      </div>
      <div class="surface p-4 mb-4">
        <div class="grid grid-cols-1 sm:grid-cols-12 gap-2 items-end">
          <label class="sm:col-span-4 block">
            <div class="text-[10px] font-mono ui-subtle mb-1">Domain</div>
            <input id="cc-detail-domain" type="text" class="ui-input text-xs px-2 py-2 w-full" value="${esc(domain)}">
          </label>
          <label class="sm:col-span-3 block">
            <div class="text-[10px] font-mono ui-subtle mb-1">Crawl</div>
            <input id="cc-detail-crawl" type="text" class="ui-input text-xs px-2 py-2 w-full" value="${esc(state.ccDomainCrawl || '')}" placeholder="latest cached crawl">
          </label>
          <label class="sm:col-span-3 block">
            <div class="text-[10px] font-mono ui-subtle mb-1">Filter URL</div>
            <input id="cc-detail-filter" type="search" class="ui-input text-xs px-2 py-2 w-full" value="${esc(state.ccDomainQ || '')}" placeholder="/path or query">
          </label>
          <div class="sm:col-span-2">
            <button class="ui-btn px-3 py-2 w-full" onclick="searchCCDomainFromDetail()">Search</button>
          </div>
        </div>
      </div>
      <div class="surface p-4">
        <div id="cc-domain-detail-content">${domainsSkeleton()}</div>
      </div>
    </div>`;

  await loadCCDomainDetail(1);
}

function searchCCDomainFromDetail() {
  const d = $('cc-detail-domain');
  const c = $('cc-detail-crawl');
  const q = $('cc-detail-filter');
  const domain = d && d.value ? d.value.trim() : '';
  const crawl = c && c.value ? c.value.trim() : '';
  state.ccDomainQ = q && q.value ? q.value.trim() : '';
  if (!domain) return;
  navigateTo(`/domains/cc/${encodeURIComponent(domain)}?crawl=${encodeURIComponent(crawl)}`);
}

async function loadCCDomainDetail(page) {
  if (page !== undefined) state.ccDomainPage = page;
  const el = $('cc-domain-detail-content');
  if (!el) return;
  try {
    const data = await apiCCDomainDetail(state.ccDomain, {
      crawl: state.ccDomainCrawl,
      page: state.ccDomainPage,
      sort: state.ccDomainSort || 'url',
      q: state.ccDomainQ || '',
    });
    if (state.currentPage !== 'domain-cc-detail') return;
    renderCCDomainDetailTable(data);
  } catch (e) {
    el.innerHTML = `<div class="text-xs text-red-400 py-4">${esc(e.message)}</div>`;
  }
}

function renderCCDomainDetailTable(data) {
  const el = $('cc-domain-detail-content');
  if (!el) return;
  const docs = data.docs || [];
  const total = data.total || 0;
  const page = data.page || 1;
  const pageSize = data.page_size || 100;
  const totalPages = Math.ceil(total / pageSize);
  const start = total > 0 ? (page - 1) * pageSize + 1 : 0;
  const end = Math.min(page * pageSize, total);
  const crawlID = data.crawl_id || state.ccDomainCrawl || '';

  el.innerHTML = `
    <div class="flex items-center gap-3 mb-4 flex-wrap">
      <span class="meta-line">${total > 0 ? `${start.toLocaleString()}\u2013${end.toLocaleString()} of ${total.toLocaleString()}` : '0 URLs'}</span>
      <span class="text-[10px] font-mono ui-subtle">crawl: ${esc(crawlID || 'unknown')}</span>
      ${data.cached_at ? `<span class="text-[10px] font-mono ui-subtle">cached: ${esc(data.cached_at)}</span>` : ''}
      ${data.truncated ? `<span class="text-[10px] font-mono" style="color:#fbbf24">truncated cache</span>` : ''}
      <select class="ui-input text-xs px-2 py-1 ml-auto" onchange="state.ccDomainSort=this.value;loadCCDomainDetail(1)">
        <option value="url" ${state.ccDomainSort==='url'?'selected':''}>URL A\u2013Z</option>
        <option value="status" ${state.ccDomainSort==='status'?'selected':''}>Status \u2191</option>
        <option value="newest" ${state.ccDomainSort==='newest'?'selected':''}>Newest</option>
      </select>
    </div>
    ${docs.length === 0 ? `<div class="ui-empty">No cached URLs for this query.</div>` : `
      <div class="space-y-0 divide-y" style="border-top:1px solid var(--border)">
        ${docs.map((d, i) => `
          <div class="flex items-start gap-3 py-2 anim-fade-up" style="animation-delay:${Math.min(i,20)*6}ms">
            <div class="shrink-0 pt-0.5">${statusBadge(d.fetch_status)}</div>
            <div class="min-w-0 flex-1">
              ${d.url ? `<a href="${esc(d.url)}" target="_blank" rel="noopener noreferrer" class="font-mono text-xs hover:text-[var(--accent)] transition-colors leading-relaxed break-all">${esc(d.url)}</a>` : '<span class="ui-subtle text-xs">\u2014</span>'}
              <div class="text-[10px] ui-subtle mt-0.5">${esc(d.mime || '')}${d.timestamp ? ` \u00b7 ${esc(d.timestamp)}` : ''}${d.filename ? ` \u00b7 ${esc(d.filename)}` : ''}</div>
            </div>
          </div>
        `).join('')}
      </div>
      ${totalPages > 1 ? `
      <div class="flex items-center justify-between mt-4 text-xs">
        <button onclick="loadCCDomainDetail(${page-1})" ${page<=1?'disabled':''} class="ui-btn px-3 py-1.5">&larr; Prev</button>
        <span class="ui-subtle">Page ${page} of ${totalPages}</span>
        <button onclick="loadCCDomainDetail(${page+1})" ${page>=totalPages?'disabled':''} class="ui-btn px-3 py-1.5">Next &rarr;</button>
      </div>` : ''}
    `}
  `;
}

// ── Domain Detail ─────────────────────────────────────────────────────────────

async function renderDomainDetail(domain) {
  state.currentPage = 'domain-detail';
  state.domainDetailDomain = domain;
  state.domainDetailPage = 1;
  state.domainDetailSort = 'status';
  state.domainDetailGroup = '';
  state.domainDetailStats = null;

  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <div class="flex items-center gap-2 text-xs font-mono ui-subtle mb-1">
          <a href="#/domains" class="ui-link">Domains</a>
          <span>/</span>
          <span class="font-medium" style="color:var(--text)">${esc(domain)}</span>
        </div>
        <h1 class="page-title font-mono">${esc(domain)}</h1>
      </div>
      <div id="domain-stats-pane" class="mb-4">${domainsSkeleton(3)}</div>
      <div class="surface p-4">
        <div id="domain-detail-content">${domainsSkeleton()}</div>
      </div>
    </div>`;

  await loadDomainDetail(1);
}

async function loadDomainDetail(page) {
  if (page !== undefined) state.domainDetailPage = page;
  const el = $('domain-detail-content');
  if (!el) return;
  const domain = state.domainDetailDomain;
  if (!domain) return;

  try {
    const data = await apiDomainDetail(domain, {
      sort: state.domainDetailSort,
      page: state.domainDetailPage,
      statusGroup: state.domainDetailGroup,
    });
    if (state.currentPage !== 'domain-detail') return;
    if (data.stats) state.domainDetailStats = data.stats;
    renderDomainStats(state.domainDetailStats, data.total, state.domainDetailGroup, data.docs || [], domain);
    renderDomainDetailTable(data);
  } catch(e) {
    const el2 = $('domain-detail-content');
    if (el2) el2.innerHTML = `<div class="text-xs text-red-400 py-4">${esc(e.message)}</div>`;
  }
}

// ── Stats pane ────────────────────────────────────────────────────────────────

const STATUS_GROUPS = [
  {key: '2xx', label: '2xx OK',       color: '#34d399', bg: 'rgba(52,211,153,0.12)'},
  {key: '3xx', label: '3xx Redirect', color: '#fbbf24', bg: 'rgba(251,191,36,0.12)'},
  {key: '4xx', label: '4xx Client',   color: '#f87171', bg: 'rgba(248,113,113,0.12)'},
  {key: '5xx', label: '5xx Server',   color: '#c084fc', bg: 'rgba(192,132,252,0.12)'},
  {key: 'other', label: 'Other',      color: '#6b7280', bg: 'rgba(107,114,128,0.12)'},
];

function renderDomainStats(stats, filteredTotal, activeGroup, docs, domain) {
  const el = $('domain-stats-pane');
  if (!el) return;
  if (!stats) { el.innerHTML = ''; return; }

  const buckets = stats.status_buckets || [];
  const total = stats.total || 0;

  // Aggregate raw codes → groups.
  const groupMap = {};
  for (const g of STATUS_GROUPS) groupMap[g.key] = 0;
  for (const b of buckets) {
    const c = b.code || 0;
    if (c >= 200 && c < 300) groupMap['2xx'] += b.count;
    else if (c >= 300 && c < 400) groupMap['3xx'] += b.count;
    else if (c >= 400 && c < 500) groupMap['4xx'] += b.count;
    else if (c >= 500 && c < 600) groupMap['5xx'] += b.count;
    else groupMap['other'] += b.count;
  }

  // Stacked bar.
  const barSegs = STATUS_GROUPS.filter(g => groupMap[g.key] > 0).map(g => {
    const pct = total > 0 ? (groupMap[g.key] / total * 100) : 0;
    const active = activeGroup === g.key;
    return `<div title="${g.label}: ${groupMap[g.key].toLocaleString()}"
      onclick="setDomainGroup('${g.key}')"
      style="width:${Math.max(0.4, pct).toFixed(2)}%;background:${g.color};height:100%;opacity:${active||!activeGroup?1:0.3};cursor:pointer;transition:opacity 0.15s"></div>`;
  }).join('');

  // Cards (All + each non-empty group).
  const allActive = !activeGroup;
  const allCard = `
    <button onclick="setDomainGroup('')" title="Show all URLs"
      class="flex flex-col gap-0.5 px-3 py-2 text-left rounded border transition-all"
      style="border-color:${allActive?'var(--accent)':'var(--border)'};background:${allActive?'rgba(99,102,241,0.1)':'var(--panel)'}">
      <span class="text-[10px] font-mono" style="color:${allActive?'var(--accent)':'var(--text-muted,#6b7280)'}">All</span>
      <span class="text-base font-semibold">${fmtBigNum(total)}</span>
      <span class="text-[10px] font-mono ui-subtle">100%</span>
    </button>`;

  const groupCards = STATUS_GROUPS.filter(g => groupMap[g.key] > 0).map(g => {
    const cnt = groupMap[g.key];
    const pct = total > 0 ? (cnt / total * 100).toFixed(0) : 0;
    const active = activeGroup === g.key;
    return `
    <button onclick="setDomainGroup('${g.key}')" title="Filter: ${g.label}"
      class="flex flex-col gap-0.5 px-3 py-2 text-left rounded border transition-all"
      style="border-color:${active?g.color:'var(--border)'};background:${active?g.bg:'var(--panel)'}">
      <span class="text-[10px] font-mono" style="color:${g.color}">${g.label}</span>
      <span class="text-base font-semibold">${fmtBigNum(cnt)}</span>
      <span class="text-[10px] font-mono ui-subtle">${pct}%</span>
    </button>`;
  }).join('');

  // Breakdown bar chart by raw status code (top-N visible codes).
  const topCodes = buckets.slice().sort((a,b) => b.count - a.count).slice(0, 8);
  const codeMax = topCodes.reduce((m, b) => Math.max(m, b.count), 0) || 1;
  const codeChart = topCodes.length > 0 ? `
    <div class="surface p-4">
      <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider mb-3">Top Status Codes</div>
      <div class="space-y-2">
        ${topCodes.map(b => {
          const g = STATUS_GROUPS.find(g => {
            const c = b.code || 0;
            if (g.key==='2xx') return c>=200&&c<300;
            if (g.key==='3xx') return c>=300&&c<400;
            if (g.key==='4xx') return c>=400&&c<500;
            if (g.key==='5xx') return c>=500&&c<600;
            return true;
          }) || STATUS_GROUPS[STATUS_GROUPS.length-1];
          const pct = (b.count / codeMax * 100).toFixed(1);
          return `
            <div class="flex items-center gap-2 text-[11px]">
              <span class="font-mono w-8 text-right" style="color:${g.color}">${b.code||'?'}</span>
              <div class="flex-1 h-1.5 rounded-full" style="background:var(--border)">
                <div class="h-full rounded-full" style="width:${pct}%;background:${g.color}"></div>
              </div>
              <span class="font-mono ui-subtle w-16 text-right">${fmtBigNum(b.count)}</span>
            </div>`;
        }).join('')}
      </div>
    </div>` : '';

  const urlTreePanel = renderURLTreePanel(docs || [], domain || state.domainDetailDomain || '');

  el.innerHTML = `
    <div class="space-y-3">
      <!-- Stacked status bar -->
      <div class="surface p-4">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider mb-3">Status Distribution
          ${activeGroup ? `<span class="ml-2 px-1.5 py-0.5 rounded text-[10px]" style="background:rgba(99,102,241,0.15);color:var(--accent)">filtered: ${activeGroup} &mdash; <a class="cursor-pointer" onclick="setDomainGroup('')">clear</a></span>` : ''}
        </div>
        <div class="flex overflow-hidden mb-4" style="height:10px;border-radius:5px;background:var(--border);gap:1px">
          ${total > 0 ? barSegs : ''}
        </div>
        <div class="flex gap-2 flex-wrap">
          ${allCard}
          ${groupCards}
        </div>
      </div>
      ${codeChart}
      ${urlTreePanel}
    </div>`;
}

function setDomainGroup(group) {
  state.domainDetailGroup = group;
  loadDomainDetail(1);
}

function renderURLTreePanel(docs, domain) {
  const root = buildURLPrefixTree(docs || []);
  const total = root.count || 0;
  const sortedByURL = state.domainDetailSort === 'url';

  return `
    <div class="surface p-4">
      <div class="flex items-center gap-2 mb-2 flex-wrap">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">URL Tree (Current Page)</div>
        <span class="text-[10px] px-1.5 py-0.5 rounded font-mono" style="background:var(--border);color:var(--text-muted,#6b7280)">
          ${fmtBigNum(total)} URL${total === 1 ? '' : 's'}
        </span>
        ${!sortedByURL ? `
          <button class="ui-btn text-[10px] px-2 py-1 ml-auto" onclick="state.domainDetailSort='url';loadDomainDetail(1)">
            Sort URL for better grouping
          </button>` : ''}
      </div>
      ${total === 0 ? `
        <div class="ui-empty">No URLs on this page.</div>
      ` : `
        <div class="text-[10px] ui-subtle mb-2">
          ${esc(domain || '')}${domain ? ' path hierarchy' : 'Path hierarchy'} from the currently loaded page.
        </div>
        <div class="space-y-1">
          ${renderURLTreeNodes(root.children || [], 0)}
        </div>
      `}
    </div>`;
}

function buildURLPrefixTree(docs) {
  const root = mkTreeNode('/');
  for (const d of docs || []) {
    if (!d || !d.url) continue;
    let path = '/';
    try {
      path = (new URL(d.url)).pathname || '/';
    } catch {
      path = '/';
    }
    const segments = path.split('/').filter(Boolean);
    const code = d.fetch_status || 0;
    root.count += 1;
    bumpStatus(root.statuses, code);

    let cur = root;
    for (const seg of segments) {
      let child = cur.childrenMap.get(seg);
      if (!child) {
        child = mkTreeNode(seg);
        cur.childrenMap.set(seg, child);
      }
      child.count += 1;
      bumpStatus(child.statuses, code);
      cur = child;
    }
  }
  finalizeTreeNode(root, true);
  return root;
}

function mkTreeNode(name) {
  return {
    name,
    count: 0,
    statuses: {},
    childrenMap: new Map(),
    children: []
  };
}

function bumpStatus(map, code) {
  const k = String(code || 0);
  map[k] = (map[k] || 0) + 1;
}

function finalizeTreeNode(node, isRoot) {
  const children = Array.from(node.childrenMap.values());
  for (const ch of children) finalizeTreeNode(ch, false);
  children.sort((a, b) => (b.count - a.count) || a.name.localeCompare(b.name));
  node.children = isRoot ? children : compressTreeNodes(children);
}

function compressTreeNodes(nodes) {
  return nodes.map((node) => {
    let cur = node;
    let name = node.name;
    // Compress one-child chains to reduce visual depth.
    while (cur.childrenMap && cur.childrenMap.size === 1) {
      const only = Array.from(cur.childrenMap.values())[0];
      name += '/' + only.name;
      cur = only;
    }
    return {
      name,
      count: cur.count,
      statuses: cur.statuses,
      children: compressTreeNodes(Array.from(cur.childrenMap ? cur.childrenMap.values() : []))
        .sort((a, b) => (b.count - a.count) || a.name.localeCompare(b.name))
    };
  });
}

function renderURLTreeNodes(nodes, depth) {
  if (!nodes || nodes.length === 0) return '';
  return nodes.map((n) => {
    const open = depth < 1 ? 'open' : '';
    const hasChildren = n.children && n.children.length > 0;
    const statusSummary = summarizeNodeStatuses(n.statuses);
    return `
      <details ${open} style="margin-left:${depth * 14}px" class="group">
        <summary class="flex items-center gap-2 py-0.5 cursor-pointer">
          <span class="font-mono text-[11px]" style="color:var(--text)">/${esc(n.name)}</span>
          <span class="text-[10px] px-1.5 py-0.5 rounded font-mono" style="background:var(--border);color:var(--text-muted,#6b7280)">${fmtBigNum(n.count)}</span>
          ${statusSummary}
          ${hasChildren ? '' : `<span class="text-[9px] ui-subtle">leaf</span>`}
        </summary>
        ${hasChildren ? `<div class="mt-0.5">${renderURLTreeNodes(n.children, depth + 1)}</div>` : ''}
      </details>`;
  }).join('');
}

function summarizeNodeStatuses(statuses) {
  const items = Object.entries(statuses || {})
    .sort((a, b) => (Number(b[1]) - Number(a[1])) || (Number(a[0]) - Number(b[0])))
    .slice(0, 3);
  if (items.length === 0) return '';
  return items.map(([code, cnt]) =>
    `<span class="font-mono text-[9px] px-1 py-0.5 rounded" style="background:${statusChipBg(Number(code))};color:${statusChipFg(Number(code))}">
      ${esc(code)}:${fmtBigNum(Number(cnt))}
    </span>`
  ).join('');
}

function statusChipBg(code) {
  if (code >= 200 && code < 300) return 'rgba(52,211,153,0.15)';
  if (code >= 300 && code < 400) return 'rgba(251,191,36,0.15)';
  if (code >= 400 && code < 500) return 'rgba(248,113,113,0.15)';
  if (code >= 500) return 'rgba(192,132,252,0.15)';
  return 'rgba(107,114,128,0.15)';
}

function statusChipFg(code) {
  if (code >= 200 && code < 300) return '#34d399';
  if (code >= 300 && code < 400) return '#fbbf24';
  if (code >= 400 && code < 500) return '#f87171';
  if (code >= 500) return '#c084fc';
  return '#6b7280';
}

// ── URL table ─────────────────────────────────────────────────────────────────

function renderDomainDetailTable(data) {
  const el = $('domain-detail-content');
  if (!el) return;

  const docs = data.docs || [];
  const total = data.total || 0;
  const page = data.page || 1;
  const pageSize = data.page_size || 100;
  const totalPages = Math.ceil(total / pageSize);
  const start = total > 0 ? (page - 1) * pageSize + 1 : 0;
  const end = Math.min(page * pageSize, total);
  const domain = state.domainDetailDomain || '';

  el.innerHTML = `
    <div class="flex items-center gap-3 mb-4 flex-wrap">
      <span class="meta-line">${total > 0 ? `${start.toLocaleString()}\u2013${end.toLocaleString()} of ${total.toLocaleString()}` : '0 URLs'}</span>
      <select class="ui-input text-xs px-2 py-1 ml-auto"
        onchange="state.domainDetailSort=this.value;loadDomainDetail(1)">
        <option value="status" ${(state.domainDetailSort||'status')==='status'?'selected':''}>Status \u2191</option>
        <option value="url"    ${state.domainDetailSort==='url'?'selected':''}>URL A\u2013Z</option>
      </select>
    </div>
    ${docs.length === 0 ? `<div class="ui-empty">No URLs found${state.domainDetailGroup ? ' for this filter' : ''}.</div>` : `
    <div class="space-y-0 divide-y" style="border-top:1px solid var(--border)">
      ${docs.map((d, i) => {
        const path = urlPath(d.url, domain);
        const scheme = urlScheme(d.url);
        return `
        <div class="flex items-start gap-3 py-2 anim-fade-up" style="animation-delay:${Math.min(i,20)*6}ms">
          <div class="shrink-0 pt-0.5">${statusBadge(d.fetch_status)}</div>
          <div class="min-w-0 flex-1">
            ${d.url
              ? `<a href="${esc(d.url)}" target="_blank" rel="noopener noreferrer"
                  class="font-mono text-xs hover:text-[var(--accent)] transition-colors leading-relaxed break-all"
                  title="${esc(d.url)}"><span class="ui-subtle">${esc(scheme)}${esc(domain)}</span><span style="color:var(--text)">${esc(path)}</span></a>`
              : '<span class="ui-subtle text-xs">\u2014</span>'}
          </div>
        </div>`;
      }).join('')}
    </div>
    ${totalPages > 1 ? `
    <div class="flex items-center justify-between mt-4 text-xs">
      <button onclick="loadDomainDetail(${page-1})" ${page<=1?'disabled':''} class="ui-btn px-3 py-1.5">&larr; Prev</button>
      <span class="ui-subtle">Page ${page} of ${totalPages}</span>
      <button onclick="loadDomainDetail(${page+1})" ${page>=totalPages?'disabled':''} class="ui-btn px-3 py-1.5">Next &rarr;</button>
    </div>` : ''}
    `}`;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function statusBadge(code) {
  const c = code || 0;
  let bg = 'rgba(107,114,128,0.15)', fg = '#6b7280';
  if (c >= 200 && c < 300) { bg = 'rgba(52,211,153,0.15)';  fg = '#34d399'; }
  else if (c >= 300 && c < 400) { bg = 'rgba(251,191,36,0.15)'; fg = '#fbbf24'; }
  else if (c >= 400 && c < 500) { bg = 'rgba(248,113,113,0.15)'; fg = '#f87171'; }
  else if (c >= 500) { bg = 'rgba(192,132,252,0.15)'; fg = '#c084fc'; }
  const label = c > 0 ? c : '???';
  return `<span class="inline-block font-mono text-[10px] px-1.5 py-0.5 rounded min-w-[32px] text-center"
    style="background:${bg};color:${fg}">${label}</span>`;
}

function urlPath(url, domain) {
  if (!url) return '';
  try {
    const u = new URL(url);
    return (u.pathname || '/') + u.search + u.hash;
  } catch { return url; }
}

function urlScheme(url) {
  if (!url) return '';
  try { return new URL(url).protocol + '//'; } catch { return ''; }
}

function domainsSkeleton(rows = 6) {
  return `<div class="space-y-2">` +
    Array.from({length: rows}, () => `
      <div class="flex gap-3 py-2 border-b border-[var(--border)]">
        <div class="h-3 w-48 ui-skeleton"></div>
        <div class="h-3 w-12 ui-skeleton ml-auto"></div>
      </div>`).join('') +
    `</div>`;
}
