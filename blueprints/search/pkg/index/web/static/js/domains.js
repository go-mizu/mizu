// ===================================================================
// Tab: Domains
// ===================================================================

let domainsFilterTimer = null;

async function renderDomains() {
  state.currentPage = 'domains';
  if (!state.domainPage) state.domainPage = 1;
  if (!state.domainSort) state.domainSort = 'count';
  if (state.domainQ === undefined) state.domainQ = '';

  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <h1 class="page-title">Domains</h1>
      </div>
      <div class="surface p-4">
        <div id="domains-content">${domainsSkeleton()}</div>
      </div>
    </div>`;

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
    renderDomainsTable(data);
  } catch(e) {
    const el2 = $('domains-content');
    if (el2) el2.innerHTML = `<div class="text-xs text-red-400 py-4">${esc(e.message)}</div>`;
  }
}

function renderDomainsTable(data) {
  const el = $('domains-content');
  if (!el) return;

  const domains = data.domains || [];
  const total = data.total || 0;
  const page = data.page || 1;
  const pageSize = data.page_size || 100;
  const totalPages = Math.ceil(total / pageSize);
  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, total);
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
    <div class="overflow-x-auto">
    <table class="w-full text-xs ui-table">
      <thead>
        <tr class="text-left">
          <th class="pb-2 pr-3 font-medium">Domain</th>
          <th class="pb-2 font-medium text-right">URLs</th>
        </tr>
      </thead>
      <tbody>
        ${domains.map((d, i) => `
          <tr class="file-row anim-fade-up" style="animation-delay:${Math.min(i,20)*10}ms">
            <td class="py-2 pr-3">
              <a href="#/domains/${encodeURIComponent(d.domain)}"
                class="ui-link font-mono font-medium">${esc(d.domain)}</a>
              <div class="mt-1 progress-track" style="height:3px">
                <div class="progress-fill"
                  style="width:${Math.max(2,(d.count/maxCount)*100).toFixed(1)}%"></div>
              </div>
            </td>
            <td class="py-2 text-right font-mono ui-subtle whitespace-nowrap">
              ${(d.count||0).toLocaleString()}
            </td>
          </tr>`).join('')}
      </tbody>
    </table>
    </div>
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

// ── Domain Detail ─────────────────────────────────────────────────────────────

async function renderDomainDetail(domain) {
  state.currentPage = 'domain-detail';
  state.domainDetailDomain = domain;
  state.domainDetailPage = 1;
  if (!state.domainDetailSort) state.domainDetailSort = 'url';

  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <div class="flex items-center gap-2 text-xs font-mono ui-subtle mb-1">
          <a href="#/domains" class="ui-link">Domains</a>
          <span>/</span>
          <span class="font-medium" style="color:var(--text)">${esc(domain)}</span>
        </div>
        <h1 class="page-title">${esc(domain)}</h1>
      </div>
      <div class="surface p-4">
        <div id="domain-detail-content">${domainsSkeleton()}</div>
      </div>
    </div>`;

  await loadDomainDetail();
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
    });
    if (state.currentPage !== 'domain-detail') return;
    renderDomainDetailTable(data);
  } catch(e) {
    const el2 = $('domain-detail-content');
    if (el2) el2.innerHTML = `<div class="text-xs text-red-400 py-4">${esc(e.message)}</div>`;
  }
}

function renderDomainDetailTable(data) {
  const el = $('domain-detail-content');
  if (!el) return;

  const docs = data.docs || [];
  const total = data.total || 0;
  const page = data.page || 1;
  const pageSize = data.page_size || 100;
  const totalPages = Math.ceil(total / pageSize);
  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, total);

  el.innerHTML = `
    <div class="flex items-center gap-3 mb-4 flex-wrap">
      <span class="meta-line">${start}\u2013${end} of ${total.toLocaleString()}</span>
      <select class="ui-input text-xs px-2 py-1 ml-auto"
        onchange="state.domainDetailSort=this.value;loadDomainDetail(1)">
        <option value="url"    ${(state.domainDetailSort||'url')==='url'?'selected':''}>URL A\u2013Z</option>
        <option value="status" ${state.domainDetailSort==='status'?'selected':''}>Status</option>
      </select>
    </div>
    ${docs.length === 0 ? `<div class="ui-empty">No URLs found for this domain.</div>` : `
    <div class="overflow-x-auto">
    <table class="w-full text-xs ui-table">
      <thead>
        <tr class="text-left">
          <th class="pb-2 pr-3 font-medium">URL</th>
          <th class="pb-2 pr-3 font-medium text-right hidden sm:table-cell">Status</th>
          <th class="pb-2 font-medium text-right hidden md:table-cell">Date</th>
        </tr>
      </thead>
      <tbody>
        ${docs.map((d, i) => `
          <tr class="file-row anim-fade-up" style="animation-delay:${Math.min(i,20)*10}ms">
            <td class="py-2 pr-3 max-w-[320px] sm:max-w-none">
              ${d.url
                ? `<a href="${esc(d.url)}" target="_blank" rel="noopener noreferrer"
                    class="ui-link font-mono truncate block" title="${esc(d.url)}">${truncateURL(d.url, 70)}</a>`
                : '<span class="ui-subtle">\u2014</span>'}
            </td>
            <td class="py-2 pr-3 text-right font-mono ui-subtle whitespace-nowrap hidden sm:table-cell">
              ${d.fetch_status ? statusChip(d.fetch_status) : ''}
            </td>
            <td class="py-2 text-right font-mono ui-subtle whitespace-nowrap hidden md:table-cell">
              ${d.crawl_date ? fmtDate(d.crawl_date) : ''}
            </td>
          </tr>`).join('')}
      </tbody>
    </table>
    </div>
    ${totalPages > 1 ? `
    <div class="flex items-center justify-between mt-4 text-xs">
      <button onclick="loadDomainDetail(${page-1})" ${page<=1?'disabled':''} class="ui-btn px-3 py-1.5">&larr; Prev</button>
      <span class="ui-subtle">Page ${page} of ${totalPages}</span>
      <button onclick="loadDomainDetail(${page+1})" ${page>=totalPages?'disabled':''} class="ui-btn px-3 py-1.5">Next &rarr;</button>
    </div>` : ''}
    `}`;
}

function statusChip(code) {
  if (!code) return '';
  let color = 'ui-subtle';
  if (code >= 200 && code < 300) color = 'status-completed';
  else if (code >= 300 && code < 400) color = 'text-yellow-400';
  else if (code >= 400) color = 'text-red-400';
  return `<span class="font-mono ${color}">${code}</span>`;
}

function domainsSkeleton() {
  return `<div class="space-y-2">` +
    Array.from({length: 8}, () => `
      <div class="flex gap-3 py-2 border-b border-[var(--border)]">
        <div class="h-3 w-48 ui-skeleton"></div>
        <div class="h-3 w-12 ui-skeleton ml-auto"></div>
      </div>`).join('') +
    `</div>`;
}
