// ===================================================================
// Tab: Scrape
// ===================================================================

let scrapeFilterTimer = null;

async function renderScrape() {
  state.currentPage = 'scrape';
  if (!state.scrapePage) state.scrapePage = 1;
  if (!state.scrapeSort) state.scrapeSort = 'crawled_at';
  if (state.scrapeQ === undefined) state.scrapeQ = '';

  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <h1 class="page-title">Scrape</h1>
        <p class="page-subtitle ui-subtle text-sm mt-1">Crawl any domain and browse scraped pages</p>
      </div>
      <div id="scrape-start-pane" class="mb-4"></div>
      <div id="scrape-list-pane"></div>
    </div>`;

  renderScrapeStartForm();
  await loadScrapeList();
}

async function renderScrapeDomain(domain) {
  state.currentPage = 'scrape-domain';
  state.scrapeDomain = domain;
  state.scrapePage = 1;
  state.scrapeQ = '';
  state.scrapeSort = 'crawled_at';

  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4 flex items-center gap-3">
        <a href="#/scrape" class="ui-btn text-xs px-2 py-1">← Back</a>
        <h1 class="page-title">${esc(domain)}</h1>
      </div>
      <div id="scrape-status-pane" class="mb-4"></div>
      <div id="scrape-pages-pane"></div>
    </div>`;

  await loadScrapeDomainStatus(domain);
  await loadScrapePages(domain);
}

// ── Start Form ───────────────────────────────────────────────────────────

function renderScrapeStartForm() {
  const el = $('scrape-start-pane');
  if (!el) return;
  el.innerHTML = `
    <div class="surface p-4">
      <h2 class="text-sm font-semibold mb-3">Start New Scrape</h2>
      <div class="flex flex-wrap gap-3 items-end">
        <div class="flex flex-col gap-1">
          <label class="text-xs ui-subtle">Domain</label>
          <input id="scrape-domain-input" class="ui-input text-sm px-3 py-1.5 w-56"
            placeholder="example.com" type="text"
            onkeydown="if(event.key==='Enter') startScrape()">
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-xs ui-subtle">Mode</label>
          <select id="scrape-mode" class="ui-select text-sm px-2 py-1.5">
            <option value="http">HTTP (fast)</option>
            <option value="browser">Browser (JS)</option>
          </select>
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-xs ui-subtle">Max Pages</label>
          <input id="scrape-max-pages" class="ui-input text-sm px-3 py-1.5 w-28"
            placeholder="0 = unlimited" type="number" min="0" value="0">
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-xs ui-subtle">Max Depth</label>
          <input id="scrape-max-depth" class="ui-input text-sm px-3 py-1.5 w-24"
            placeholder="0 = unlimited" type="number" min="0" value="0">
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-xs ui-subtle">Workers</label>
          <input id="scrape-workers" class="ui-input text-sm px-3 py-1.5 w-24"
            placeholder="default" type="number" min="0" value="0">
        </div>
        <div class="flex items-center gap-2 pb-1">
          <input id="scrape-store-body" type="checkbox" checked class="w-4 h-4 cursor-pointer">
          <label for="scrape-store-body" class="text-xs ui-subtle cursor-pointer">Store HTML</label>
        </div>
        <button onclick="startScrape()" class="ui-btn ui-btn-primary text-sm px-4 py-1.5 pb-1">
          Start Scrape
        </button>
      </div>
      <div id="scrape-start-error" class="text-xs mt-2 hidden" style="color:var(--error)"></div>
    </div>`;
}

async function startScrape() {
  const domain = ($('scrape-domain-input')?.value || '').trim();
  const errEl = $('scrape-start-error');
  if (!domain) {
    if (errEl) { errEl.textContent = 'Enter a domain'; errEl.classList.remove('hidden'); }
    return;
  }
  if (errEl) errEl.classList.add('hidden');

  const payload = {
    domain,
    mode: $('scrape-mode')?.value || 'http',
    max_pages: parseInt($('scrape-max-pages')?.value || '0', 10) || 0,
    max_depth: parseInt($('scrape-max-depth')?.value || '0', 10) || 0,
    workers: parseInt($('scrape-workers')?.value || '0', 10) || 0,
    store_body: !!$('scrape-store-body')?.checked,
    resume: false,
  };

  try {
    const res = await apiScrapeStart(payload);
    navigateTo(`#/scrape/${encodeURIComponent(res.domain)}`);
  } catch (e) {
    if (errEl) { errEl.textContent = e.message; errEl.classList.remove('hidden'); }
  }
}

// ── Domain List ──────────────────────────────────────────────────────────

async function loadScrapeList() {
  const el = $('scrape-list-pane');
  if (!el) return;
  try {
    const data = await apiScrapeList();
    if (state.currentPage !== 'scrape') return;
    renderScrapeList(data);
  } catch (e) {
    if (el) el.innerHTML = `<div class="surface p-4 text-sm" style="color:var(--error)">${esc(e.message)}</div>`;
  }
}

function renderScrapeList(data) {
  const el = $('scrape-list-pane');
  if (!el) return;
  const domains = (data && data.domains) || [];
  if (domains.length === 0) {
    el.innerHTML = `<div class="surface p-4 text-sm ui-subtle">No scraped domains yet. Start a scrape above.</div>`;
    return;
  }
  const rows = domains.map(d => `
    <tr class="border-t border-[var(--border)] hover:bg-[var(--surface-hover)] cursor-pointer"
        onclick="navigateTo('#/scrape/${encodeURIComponent(d.domain)}')">
      <td class="px-4 py-2.5 text-sm font-mono font-medium">${esc(d.domain)}</td>
      <td class="px-4 py-2.5 text-sm text-right">${fmtNum(d.pages)}</td>
      <td class="px-4 py-2.5 text-sm text-right" style="color:var(--success)">${fmtNum(d.success)}</td>
      <td class="px-4 py-2.5 text-sm text-right" style="color:var(--error)">${fmtNum(d.failed)}</td>
      <td class="px-4 py-2.5 text-sm text-right">${fmtNum(d.links)}</td>
      <td class="px-4 py-2.5 text-sm text-right">${d.last_crawl ? new Date(d.last_crawl).toLocaleDateString() : '—'}</td>
      <td class="px-4 py-2.5 text-sm text-center">${d.has_markdown ? '<span style="color:var(--success)">✓</span>' : '<span class="ui-subtle">—</span>'}</td>
      <td class="px-4 py-2.5 text-sm text-center">${d.has_index ? '<span style="color:var(--success)">✓</span>' : '<span class="ui-subtle">—</span>'}</td>
    </tr>`).join('');

  el.innerHTML = `
    <div class="surface">
      <table class="w-full">
        <thead>
          <tr class="text-xs ui-subtle">
            <th class="px-4 py-2.5 text-left font-medium">Domain</th>
            <th class="px-4 py-2.5 text-right font-medium">Pages</th>
            <th class="px-4 py-2.5 text-right font-medium">OK</th>
            <th class="px-4 py-2.5 text-right font-medium">Failed</th>
            <th class="px-4 py-2.5 text-right font-medium">Links</th>
            <th class="px-4 py-2.5 text-right font-medium">Last Crawl</th>
            <th class="px-4 py-2.5 text-center font-medium">MD</th>
            <th class="px-4 py-2.5 text-center font-medium">Index</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>
    </div>`;
}

// ── Domain Detail: Status ─────────────────────────────────────────────────

async function loadScrapeDomainStatus(domain) {
  const el = $('scrape-status-pane');
  if (!el) return;
  try {
    const data = await apiScrapeStatus(domain);
    if (state.currentPage !== 'scrape-domain') return;
    renderScrapeDomainStatus(domain, data);
  } catch (e) {
    if (el) el.innerHTML = `<div class="surface p-4 text-sm" style="color:var(--error)">${esc(e.message)}</div>`;
  }
}

function renderScrapeDomainStatus(domain, data) {
  const el = $('scrape-status-pane');
  if (!el) return;

  const stats = data.stats;
  const active = data.active_job;

  // Stats row
  const statsHTML = stats ? `
    <div class="flex flex-wrap gap-6 text-sm">
      <div><span class="ui-subtle text-xs">Pages</span><br><strong>${fmtNum(stats.pages)}</strong></div>
      <div><span class="ui-subtle text-xs">Success</span><br><strong style="color:var(--success)">${fmtNum(stats.success)}</strong></div>
      <div><span class="ui-subtle text-xs">Failed</span><br><strong style="color:var(--error)">${fmtNum(stats.failed)}</strong></div>
      <div><span class="ui-subtle text-xs">Links</span><br><strong>${fmtNum(stats.links)}</strong></div>
      <div><span class="ui-subtle text-xs">Last Crawl</span><br><strong>${stats.last_crawl ? new Date(stats.last_crawl).toLocaleString() : '—'}</strong></div>
    </div>` : `<div class="text-sm ui-subtle">No crawl data yet.</div>`;

  // Active job progress
  let progressHTML = '';
  if (active && (active.status === 'running' || active.status === 'queued')) {
    progressHTML = `
      <div class="mt-3 p-3 rounded" style="background:var(--surface-2)">
        <div class="flex items-center justify-between mb-1">
          <span class="text-xs font-mono" style="color:var(--accent)">${esc(active.status)}</span>
          <span class="text-xs ui-subtle">${esc(active.message)}</span>
          <span class="text-xs font-mono">${active.rate > 0 ? fmtNum(Math.round(active.rate)) + '/s' : ''}</span>
        </div>
        <div class="h-1.5 rounded-full overflow-hidden" style="background:var(--border)">
          <div class="h-full rounded-full transition-all" style="background:var(--accent);width:${Math.round((active.progress||0)*100)}%"></div>
        </div>
      </div>`;

    // Auto-refresh status while running
    setTimeout(() => loadScrapeDomainStatus(domain), 2000);
  }

  // Controls
  const hasActive = active && (active.status === 'running' || active.status === 'queued');
  const controlsHTML = `
    <div class="flex flex-wrap gap-2 mt-3">
      ${hasActive
        ? `<button onclick="stopScrape('${esc(domain)}')" class="ui-btn text-xs px-3 py-1.5" style="border-color:var(--error);color:var(--error)">Stop</button>`
        : `<button onclick="resumeScrape('${esc(domain)}')" class="ui-btn text-xs px-3 py-1.5">Resume</button>
           <button onclick="startScrapeDomain('${esc(domain)}')" class="ui-btn ui-btn-primary text-xs px-3 py-1.5">New Crawl</button>`}
      ${!hasActive && stats && stats.pages > 0
        ? `<button onclick="triggerScrapePipeline('${esc(domain)}')" class="ui-btn text-xs px-3 py-1.5">Convert to Markdown</button>`
        : ''}
    </div>`;

  el.innerHTML = `
    <div class="surface p-4">
      ${statsHTML}
      ${progressHTML}
      ${controlsHTML}
      <div id="scrape-action-msg" class="text-xs mt-2 hidden"></div>
    </div>`;
}

// ── Domain Detail: Pages Table ────────────────────────────────────────────

async function loadScrapePages(domain, page) {
  if (page !== undefined) state.scrapePage = page;
  const el = $('scrape-pages-pane');
  if (!el) return;

  try {
    const data = await apiScrapePages(domain, {
      page: state.scrapePage,
      pageSize: 50,
      q: state.scrapeQ,
      sort: state.scrapeSort,
    });
    if (state.currentPage !== 'scrape-domain') return;
    renderScrapePagesTable(domain, data);
  } catch (e) {
    if (el) el.innerHTML = `<div class="surface p-4 text-sm" style="color:var(--error)">${esc(e.message)}</div>`;
  }
}

function renderScrapePagesTable(domain, data) {
  const el = $('scrape-pages-pane');
  if (!el) return;
  const pages = (data && data.pages) || [];
  const total = (data && data.total) || 0;
  const page = (data && data.page) || 1;
  const pageSize = (data && data.page_size) || 50;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const sortOpts = [
    ['crawled_at', 'Date'],
    ['url', 'URL'],
    ['status', 'Status'],
    ['size', 'Size'],
    ['fetch_time', 'Fetch Time'],
  ].map(([v, l]) => `<option value="${v}"${state.scrapeSort===v?' selected':''}>${l}</option>`).join('');

  const rows = pages.length === 0
    ? `<tr><td colspan="6" class="px-4 py-8 text-center text-sm ui-subtle">No pages found</td></tr>`
    : pages.map(p => {
        const statusClass = p.status_code >= 400 ? 'color:var(--error)' : p.status_code >= 300 ? 'color:var(--warning)' : 'color:var(--success)';
        return `<tr class="border-t border-[var(--border)] hover:bg-[var(--surface-hover)]">
          <td class="px-4 py-2 text-xs font-mono" style="${statusClass}">${p.status_code || '—'}</td>
          <td class="px-4 py-2 text-sm max-w-xs truncate" title="${esc(p.url)}">
            <a href="${esc(p.url)}" target="_blank" rel="noopener" class="hover:underline" style="color:var(--link)">${esc(p.url)}</a>
          </td>
          <td class="px-4 py-2 text-sm max-w-xs truncate">${esc(p.title || '—')}</td>
          <td class="px-4 py-2 text-xs ui-subtle">${esc(p.content_type ? p.content_type.split(';')[0] : '—')}</td>
          <td class="px-4 py-2 text-xs ui-subtle text-right">${p.content_length > 0 ? fmtBytes(p.content_length) : '—'}</td>
          <td class="px-4 py-2 text-xs ui-subtle text-right">${p.fetch_time_ms > 0 ? p.fetch_time_ms + 'ms' : '—'}</td>
        </tr>`;
      }).join('');

  el.innerHTML = `
    <div class="surface">
      <div class="p-3 border-b border-[var(--border)] flex flex-wrap items-center gap-3">
        <span class="text-xs ui-subtle">${fmtNum(total)} pages</span>
        <input class="ui-input text-xs px-2 py-1 flex-1 min-w-40 max-w-64" placeholder="Filter URL or title…"
          value="${esc(state.scrapeQ || '')}"
          oninput="clearTimeout(scrapeFilterTimer); scrapeFilterTimer=setTimeout(()=>{state.scrapeQ=this.value;state.scrapePage=1;loadScrapePages('${esc(domain)}')},300)">
        <select class="ui-select text-xs px-2 py-1" onchange="state.scrapeSort=this.value;state.scrapePage=1;loadScrapePages('${esc(domain)}')">
          ${sortOpts}
        </select>
      </div>
      <table class="w-full">
        <thead>
          <tr class="text-xs ui-subtle">
            <th class="px-4 py-2 text-left font-medium w-16">Status</th>
            <th class="px-4 py-2 text-left font-medium">URL</th>
            <th class="px-4 py-2 text-left font-medium">Title</th>
            <th class="px-4 py-2 text-left font-medium">Type</th>
            <th class="px-4 py-2 text-right font-medium">Size</th>
            <th class="px-4 py-2 text-right font-medium">Fetch</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>
      ${totalPages > 1 ? `
        <div class="p-3 border-t border-[var(--border)] flex items-center gap-2 text-xs">
          ${page > 1 ? `<button class="ui-btn px-2 py-1" onclick="loadScrapePages('${esc(domain)}',${page-1})">← Prev</button>` : ''}
          <span class="ui-subtle">Page ${page} / ${totalPages}</span>
          ${page < totalPages ? `<button class="ui-btn px-2 py-1" onclick="loadScrapePages('${esc(domain)}',${page+1})">Next →</button>` : ''}
        </div>` : ''}
    </div>`;
}

// ── Actions ───────────────────────────────────────────────────────────────

async function stopScrape(domain) {
  try {
    await apiScrapeStop(domain);
    await loadScrapeDomainStatus(domain);
  } catch (e) {
    showScrapeActionMsg(e.message, 'error');
  }
}

async function resumeScrape(domain) {
  try {
    await apiScrapeResume(domain);
    await loadScrapeDomainStatus(domain);
  } catch (e) {
    showScrapeActionMsg(e.message, 'error');
  }
}

async function startScrapeDomain(domain) {
  try {
    await apiScrapeStart({ domain, mode: 'http', store_body: true, resume: false });
    await loadScrapeDomainStatus(domain);
  } catch (e) {
    showScrapeActionMsg(e.message, 'error');
  }
}

async function triggerScrapePipeline(domain) {
  try {
    const res = await apiScrapePipeline(domain);
    showScrapeActionMsg(`Started job ${res.job_id} — converting pages to markdown`, 'ok');
  } catch (e) {
    showScrapeActionMsg(e.message, 'error');
  }
}

function showScrapeActionMsg(msg, type) {
  const el = $('scrape-action-msg');
  if (!el) return;
  el.textContent = msg;
  el.style.color = type === 'error' ? 'var(--error)' : 'var(--success)';
  el.classList.remove('hidden');
  setTimeout(() => el.classList.add('hidden'), 5000);
}

// ── Helpers ───────────────────────────────────────────────────────────────

function fmtBytes(n) {
  if (n < 1024) return n + 'B';
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + 'KB';
  return (n / (1024 * 1024)).toFixed(1) + 'MB';
}
