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
  state.scrapeStatusFilter = '';

  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4 flex items-center gap-3">
        <a href="#/scrape" class="ui-btn text-xs px-2 py-1">\u2190 Back</a>
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
    <div class="surface p-3">
      <div class="flex flex-wrap gap-2 items-center">
        <input id="scrape-domain-input" class="ui-input text-sm px-3 py-1.5 w-56"
          placeholder="domain.com" type="text"
          onkeydown="if(event.key==='Enter') startScrape()">
        <select id="scrape-mode" class="ui-select text-sm px-2 py-1.5" onchange="toggleScrapeAdvanced()">
          <option value="http">HTTP</option>
          <option value="browser">Browser</option>
          <option value="worker">Worker</option>
        </select>
        <input id="scrape-max-pages" class="ui-input text-sm px-3 py-1.5 w-24"
          placeholder="Pages" type="number" min="0" value="0">
        <input id="scrape-max-depth" class="ui-input text-sm px-3 py-1.5 w-20"
          placeholder="Depth" type="number" min="0" value="0">
        <input id="scrape-workers" class="ui-input text-sm px-3 py-1.5 w-20"
          placeholder="Workers" type="number" min="0" value="0">
        <label class="flex items-center gap-1.5 text-xs ui-subtle cursor-pointer">
          <input id="scrape-store-body" type="checkbox" checked class="w-4 h-4 cursor-pointer">
          Store HTML
        </label>
        <button onclick="startScrape()" class="ui-btn ui-btn-primary text-sm px-4 py-1.5">
          Scrape
        </button>
        <button onclick="toggleScrapeAdvancedPanel()" class="ui-btn text-xs px-2 py-1.5 ui-subtle">Advanced \u25BC</button>
      </div>
      <div id="scrape-advanced" class="hidden mt-3 pt-3 border-t border-[var(--border)]">
        <div class="flex flex-wrap gap-x-4 gap-y-2 items-center text-xs">
          <div class="flex items-center gap-1.5">
            <label class="ui-subtle">Timeout(s)</label>
            <input id="scrape-timeout" class="ui-input text-xs px-2 py-1 w-16" type="number" min="0" value="0" placeholder="10">
          </div>
          <label class="flex items-center gap-1.5 cursor-pointer">
            <input id="scrape-no-robots" type="checkbox" class="w-3.5 h-3.5 cursor-pointer">
            <span class="ui-subtle">No robots.txt</span>
          </label>
          <label class="flex items-center gap-1.5 cursor-pointer">
            <input id="scrape-no-sitemap" type="checkbox" class="w-3.5 h-3.5 cursor-pointer">
            <span class="ui-subtle">No sitemap</span>
          </label>
          <label class="flex items-center gap-1.5 cursor-pointer">
            <input id="scrape-subdomain" type="checkbox" class="w-3.5 h-3.5 cursor-pointer">
            <span class="ui-subtle">Include subdomains</span>
          </label>
          <label class="flex items-center gap-1.5 cursor-pointer">
            <input id="scrape-continuous" type="checkbox" class="w-3.5 h-3.5 cursor-pointer">
            <span class="ui-subtle">Continuous</span>
          </label>
          <div id="scrape-scroll-wrap" class="flex items-center gap-1.5 hidden">
            <label class="ui-subtle">Scroll</label>
            <input id="scrape-scroll" class="ui-input text-xs px-2 py-1 w-14" type="number" min="0" value="0" placeholder="0">
          </div>
          <div class="flex items-center gap-1.5">
            <label class="ui-subtle">Stale(h)</label>
            <input id="scrape-stale" class="ui-input text-xs px-2 py-1 w-14" type="number" min="0" value="0" placeholder="0">
          </div>
          <div class="flex items-center gap-1.5">
            <label class="ui-subtle">Seed URL</label>
            <input id="scrape-seed-url" class="ui-input text-xs px-2 py-1 w-48" type="text" placeholder="https://...">
          </div>
          <div id="scrape-worker-opts" class="flex items-center gap-3 hidden">
            <div class="flex items-center gap-1.5">
              <label class="ui-subtle">Token</label>
              <input id="scrape-worker-token" class="ui-input text-xs px-2 py-1 w-40" type="password" placeholder="from env if empty">
            </div>
            <div class="flex items-center gap-1.5">
              <label class="ui-subtle">Worker URL</label>
              <input id="scrape-worker-url" class="ui-input text-xs px-2 py-1 w-48" type="text" placeholder="https://crawler.go-mizu.workers.dev">
            </div>
            <label class="flex items-center gap-1.5 cursor-pointer">
              <input id="scrape-worker-browser" type="checkbox" class="w-3.5 h-3.5 cursor-pointer">
              <span class="ui-subtle">CF Browser</span>
            </label>
          </div>
        </div>
      </div>
      <div id="scrape-start-error" class="text-xs mt-2 hidden" style="color:var(--error)"></div>
    </div>`;
}

function toggleScrapeAdvancedPanel() {
  const el = $('scrape-advanced');
  if (el) el.classList.toggle('hidden');
}

function toggleScrapeAdvanced() {
  const mode = $('scrape-mode')?.value;
  const scrollWrap = $('scrape-scroll-wrap');
  if (scrollWrap) {
    if (mode === 'browser') scrollWrap.classList.remove('hidden');
    else scrollWrap.classList.add('hidden');
  }
  const workerOpts = $('scrape-worker-opts');
  if (workerOpts) {
    if (mode === 'worker') { workerOpts.classList.remove('hidden'); $('scrape-advanced')?.classList.remove('hidden'); }
    else workerOpts.classList.add('hidden');
  }
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
    timeout_s: parseInt($('scrape-timeout')?.value || '0', 10) || 0,
    store_body: !!$('scrape-store-body')?.checked,
    resume: false,
    no_robots: !!$('scrape-no-robots')?.checked,
    no_sitemap: !!$('scrape-no-sitemap')?.checked,
    include_subdomain: !!$('scrape-subdomain')?.checked,
    scroll_count: parseInt($('scrape-scroll')?.value || '0', 10) || 0,
    continuous: !!$('scrape-continuous')?.checked,
    stale_hours: parseInt($('scrape-stale')?.value || '0', 10) || 0,
    seed_url: ($('scrape-seed-url')?.value || '').trim(),
    worker_token: ($('scrape-worker-token')?.value || '').trim(),
    worker_url: ($('scrape-worker-url')?.value || '').trim(),
    worker_browser: !!$('scrape-worker-browser')?.checked,
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

  const fmtDate = (v) => {
    if (!v) return '\u2014';
    const dt = new Date(v);
    if (dt.getFullYear() <= 1970) return '\u2014';
    return dt.toLocaleDateString();
  };

  const fmtSizeCol = (n) => (n > 0 ? fmtBytes(n) : '\u2014');

  const rows = domains.map(d => {
    // Action buttons
    let actions = `<button onclick="event.stopPropagation();startScrapeDomain('${esc(d.domain)}')" class="ui-btn text-xs px-2 py-0.5">Scrape</button>`;
    if (d.pages > 0) {
      actions += ` <button onclick="event.stopPropagation();triggerScrapePipeline('${esc(d.domain)}')" class="ui-btn text-xs px-2 py-0.5">\u2192 MD</button>`;
    }
    if (d.has_markdown) {
      actions += ` <button onclick="event.stopPropagation();triggerScrapeIndex('${esc(d.domain)}')" class="ui-btn text-xs px-2 py-0.5">\u2192 Index</button>`;
    }

    return `
    <tr class="border-t border-[var(--border)] hover:bg-[var(--surface-hover)] cursor-pointer"
        onclick="navigateTo('#/scrape/${encodeURIComponent(d.domain)}')">
      <td class="px-4 py-2.5 text-sm font-mono font-medium">${esc(d.domain)}</td>
      <td class="px-4 py-2.5 text-sm text-right">${fmtNum(d.pages)}</td>
      <td class="px-4 py-2.5 text-sm text-right" style="color:var(--success)">${fmtNum(d.success)}</td>
      <td class="px-4 py-2.5 text-sm text-right" style="color:var(--error)">${fmtNum(d.failed)}</td>
      <td class="px-4 py-2.5 text-sm text-right">${fmtSizeCol(d.html_bytes)}</td>
      <td class="px-4 py-2.5 text-sm text-right">${fmtSizeCol(d.md_bytes)}</td>
      <td class="px-4 py-2.5 text-sm text-right">${fmtSizeCol(d.index_bytes)}</td>
      <td class="px-4 py-2.5 text-sm text-right">${fmtDate(d.last_crawl)}</td>
      <td class="px-4 py-2.5 text-sm text-right whitespace-nowrap">${actions}</td>
    </tr>`;
  }).join('');

  el.innerHTML = `
    <div class="surface">
      <table class="w-full">
        <thead>
          <tr class="text-xs ui-subtle">
            <th class="px-4 py-2.5 text-left font-medium">Domain</th>
            <th class="px-4 py-2.5 text-right font-medium">Pages</th>
            <th class="px-4 py-2.5 text-right font-medium">OK</th>
            <th class="px-4 py-2.5 text-right font-medium">Failed</th>
            <th class="px-4 py-2.5 text-right font-medium">HTML Size</th>
            <th class="px-4 py-2.5 text-right font-medium">MD Size</th>
            <th class="px-4 py-2.5 text-right font-medium">Index Size</th>
            <th class="px-4 py-2.5 text-right font-medium">Last Crawl</th>
            <th class="px-4 py-2.5 text-right font-medium">Actions</th>
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
  const isRunning = active && (active.status === 'running' || active.status === 'queued');

  // Parse live ScrapeState from JSON job message
  let live = null;
  if (active && active.message) {
    try { live = JSON.parse(active.message); } catch {}
  }

  // ── DB Stats Panel ──────────────────────────────────────────────────
  const fmtDate = (v) => {
    if (!v) return '\u2014';
    const dt = new Date(v);
    if (dt.getFullYear() <= 1970) return '\u2014';
    return dt.toLocaleString();
  };
  const fmtSz = (n) => (n > 0 ? fmtBytes(n) : '\u2014');

  const dbHTML = stats ? `
    <div class="mb-3">
      <div class="text-xs font-medium ui-subtle mb-2 uppercase tracking-wide">DB Stats</div>
      <div class="flex flex-wrap gap-x-6 gap-y-2 text-sm">
        <div><span class="ui-subtle text-xs">Pages</span><br><strong>${fmtNum(stats.pages)}</strong></div>
        <div><span class="ui-subtle text-xs">Success</span><br><strong style="color:var(--success)">${fmtNum(stats.success)}</strong></div>
        <div><span class="ui-subtle text-xs">Failed</span><br><strong style="color:var(--error)">${fmtNum(stats.failed)}</strong></div>
        <div><span class="ui-subtle text-xs">Links</span><br><strong>${fmtNum(stats.links)}</strong></div>
        <div><span class="ui-subtle text-xs">HTML</span><br><strong>${fmtSz(stats.html_bytes)}</strong></div>
        ${stats.md_bytes > 0 ? `<div><span class="ui-subtle text-xs">MD</span><br><strong>${fmtSz(stats.md_bytes)}</strong></div>` : ''}
        ${stats.index_bytes > 0 ? `<div><span class="ui-subtle text-xs">Index</span><br><strong>${fmtSz(stats.index_bytes)}</strong></div>` : ''}
        <div><span class="ui-subtle text-xs">Last Crawl</span><br><strong>${fmtDate(stats.last_crawl)}</strong></div>
      </div>
    </div>` : (!isRunning ? `<div class="text-sm ui-subtle mb-3">No crawl data yet.</div>` : '');

  // ── Live Stats Panel ────────────────────────────────────────────────
  let liveHTML = '';
  if (isRunning) {
    const fmtElapsed = (ms) => {
      if (!ms) return '\u2014';
      const s = Math.floor(ms / 1000);
      if (s < 60) return s + 's';
      const m = Math.floor(s / 60), rem = s % 60;
      return m + 'm ' + rem + 's';
    };

    const fmtRate = (bps) => {
      if (!bps || bps <= 0) return '\u2014';
      if (bps < 1024) return bps.toFixed(0) + ' B/s';
      if (bps < 1024 * 1024) return (bps / 1024).toFixed(1) + ' KB/s';
      return (bps / (1024 * 1024)).toFixed(1) + ' MB/s';
    };

    const pct = Math.round((active.progress || 0) * 100);
    const rps = live ? live.pages_per_sec : active.rate;

    // Build extra metrics row (timeout, blocked, skipped, bytes/sec, peak, retry, avg fetch)
    let extraMetrics = '';
    if (live) {
      const extras = [];
      if (live.timeout > 0) extras.push(`<div><span class="ui-subtle text-xs">Timeout</span><br><strong style="color:var(--warning)">${fmtNum(live.timeout)}</strong></div>`);
      if (live.blocked > 0) extras.push(`<div><span class="ui-subtle text-xs">Blocked</span><br><strong style="color:var(--warning)">${fmtNum(live.blocked)}</strong></div>`);
      if (live.skipped > 0) extras.push(`<div><span class="ui-subtle text-xs">Skipped</span><br><strong style="color:var(--muted)">${fmtNum(live.skipped)}</strong></div>`);
      if (live.bytes_per_sec > 0) extras.push(`<div><span class="ui-subtle text-xs">Speed</span><br><strong>${fmtRate(live.bytes_per_sec)}</strong></div>`);
      if (live.peak_speed > 0) extras.push(`<div><span class="ui-subtle text-xs">Peak</span><br><strong>${live.peak_speed.toFixed(1)}/s</strong></div>`);
      if (live.avg_fetch_ms > 0) extras.push(`<div><span class="ui-subtle text-xs">Avg Fetch</span><br><strong>${live.avg_fetch_ms < 1000 ? live.avg_fetch_ms.toFixed(0) + 'ms' : (live.avg_fetch_ms / 1000).toFixed(1) + 's'}</strong></div>`);
      if (live.retry_queue > 0) extras.push(`<div><span class="ui-subtle text-xs">Retry Q</span><br><strong>${fmtNum(live.retry_queue)}</strong></div>`);
      if (extras.length > 0) {
        extraMetrics = `<div class="flex flex-wrap gap-x-5 gap-y-1.5 text-sm mt-1.5 pt-1.5 border-t border-[var(--border)]">${extras.join('')}</div>`;
      }
    }

    liveHTML = `
      <div class="mb-3 p-3 rounded" style="background:var(--surface-2)">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium" style="color:var(--accent)">Live \u2014 ${esc(active.status)}</span>
          ${rps > 0 ? `<span class="text-xs font-mono ui-subtle">${rps.toFixed(1)} pages/s</span>` : ''}
        </div>
        ${live ? `
        <div class="flex flex-wrap gap-x-5 gap-y-1.5 text-sm mb-2">
          <div><span class="ui-subtle text-xs">Crawled</span><br><strong>${fmtNum(live.pages)}</strong></div>
          <div><span class="ui-subtle text-xs">OK</span><br><strong style="color:var(--success)">${fmtNum(live.success)}</strong></div>
          <div><span class="ui-subtle text-xs">Failed</span><br><strong style="color:var(--error)">${fmtNum(live.failed)}</strong></div>
          <div><span class="ui-subtle text-xs">Frontier</span><br><strong>${fmtNum(live.frontier)}</strong></div>
          <div><span class="ui-subtle text-xs">In-flight</span><br><strong>${fmtNum(live.in_flight)}</strong></div>
          <div><span class="ui-subtle text-xs">Links</span><br><strong>${fmtNum(live.links_found)}</strong></div>
          <div><span class="ui-subtle text-xs">Recv</span><br><strong>${fmtBytes(live.bytes_recv)}</strong></div>
          <div><span class="ui-subtle text-xs">Elapsed</span><br><strong>${fmtElapsed(live.elapsed_ms)}</strong></div>
        </div>
        ${extraMetrics}` : ''}
        <div class="h-1.5 rounded-full overflow-hidden mt-2" style="background:var(--border)">
          <div class="h-full rounded-full transition-all" style="background:var(--accent);width:${pct}%"></div>
        </div>
      </div>`;

    // Auto-refresh status + pages while running
    setTimeout(() => {
      loadScrapeDomainStatus(domain);
      loadScrapePages(domain);
    }, 3000);
  }

  // ── Controls ────────────────────────────────────────────────────────
  const totalPages = (live && live.pages > 0) ? live.pages : (stats ? stats.pages : 0);
  const controlsHTML = isRunning ? `
    <div class="flex flex-wrap gap-2">
      <button onclick="stopScrape('${esc(domain)}')" class="ui-btn text-xs px-3 py-1.5" style="border-color:var(--error);color:var(--error)">Stop</button>
    </div>` : `
    <div class="flex flex-wrap gap-2 items-center">
      <select id="sd-mode" class="ui-select text-xs px-2 py-1.5" onchange="toggleDomainWorkerOpts()">
        <option value="http">HTTP</option>
        <option value="browser">Browser</option>
        <option value="worker">Worker</option>
      </select>
      <input id="sd-max-pages" class="ui-input text-xs px-2 py-1.5 w-20" placeholder="Pages" type="number" min="0" value="0">
      <input id="sd-workers" class="ui-input text-xs px-2 py-1.5 w-20" placeholder="Workers" type="number" min="0" value="0">
      <span id="sd-worker-opts" class="hidden flex items-center gap-2">
        <input id="sd-worker-token" class="ui-input text-xs px-2 py-1 w-36" type="password" placeholder="from env if empty">
        <label class="flex items-center gap-1 cursor-pointer"><input id="sd-worker-browser" type="checkbox" class="w-3.5 h-3.5 cursor-pointer"><span class="ui-subtle text-xs">CF Browser</span></label>
      </span>
      <button onclick="startScrapeDomainFull('${esc(domain)}')" class="ui-btn ui-btn-primary text-xs px-3 py-1.5">New Crawl</button>
      <button onclick="resumeScrape('${esc(domain)}')" class="ui-btn text-xs px-3 py-1.5">Resume</button>
      ${totalPages > 0 ? `<button onclick="triggerScrapePipeline('${esc(domain)}')" class="ui-btn text-xs px-3 py-1.5">\u2192 Markdown</button>` : ''}
      ${stats && stats.has_markdown ? `<button onclick="triggerScrapeIndex('${esc(domain)}')" class="ui-btn text-xs px-3 py-1.5">\u2192 Index</button>` : ''}
    </div>`;

  el.innerHTML = `
    <div class="surface p-4">
      ${dbHTML}
      ${liveHTML}
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
      status: state.scrapeStatusFilter || '',
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
  const activeFilter = state.scrapeStatusFilter || '';

  const tabs = [
    ['', 'All'],
    ['2xx', '2xx'],
    ['3xx', '3xx'],
    ['4xx', '4xx'],
    ['5xx', '5xx'],
    ['error', 'Error'],
  ].map(([v, l]) => {
    const active = v === activeFilter;
    return `<button class="text-xs px-3 py-1.5 border-b-2 transition-colors ${active ? 'font-medium' : 'ui-subtle'}"
      style="${active ? 'border-color:var(--accent);color:var(--accent)' : 'border-color:transparent'}"
      onclick="state.scrapeStatusFilter='${v}';state.scrapePage=1;loadScrapePages('${esc(domain)}')">${l}</button>`;
  }).join('');

  const sortOpts = [
    ['crawled_at', 'Date'],
    ['url', 'URL'],
    ['status', 'Status'],
    ['size', 'Size'],
    ['fetch_time', 'Fetch Time'],
  ].map(([v, l]) => `<option value="${v}"${state.scrapeSort===v?' selected':''}>${l}</option>`).join('');

  const fmtFetch = (ms) => {
    if (!ms || ms <= 0) return '\u2014';
    if (ms < 1000) return ms + 'ms';
    return (ms / 1000).toFixed(1) + 's';
  };

  const fmtRelTime = (v) => {
    if (!v) return '\u2014';
    const dt = new Date(v);
    if (isNaN(dt.getTime()) || dt.getFullYear() <= 1970) return '\u2014';
    const diff = Math.floor((Date.now() - dt.getTime()) / 1000);
    if (diff < 60) return diff + 's ago';
    if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
    if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
    if (diff < 86400 * 30) return Math.floor(diff / 86400) + 'd ago';
    return dt.toLocaleDateString();
  };

  const rows = pages.length === 0
    ? `<tr><td colspan="7" class="px-4 py-8 text-center text-sm ui-subtle">No pages found</td></tr>`
    : pages.map(p => {
        const sc = p.status_code;
        const isBlocked = p.error && p.error.startsWith('blocked:');
        const statusStyle = sc >= 500 ? 'color:var(--error)' : sc >= 400 ? 'color:var(--warning)' : sc >= 300 ? 'color:var(--accent)' : sc >= 200 ? (isBlocked ? 'color:var(--warning)' : 'color:var(--success)') : (p.error ? 'color:var(--error)' : 'color:var(--muted)');
        const statusLabel = sc > 0 ? String(sc) : (p.error ? 'ERR' : '\u2014');
        // For blocked pages, show a short tag instead of full error in title column
        const titleDisplay = p.title && !isBlocked ? p.title : (isBlocked ? '' : (p.error || ''));
        const blockedTag = isBlocked ? `<span class="text-xs px-1.5 py-0.5 rounded" style="background:var(--warning-bg,rgba(234,179,8,0.15));color:var(--warning)">blocked</span>` : '';
        return `<tr class="border-t border-[var(--border)] hover:bg-[var(--surface-hover)]">
          <td class="px-4 py-2 text-xs font-mono" style="${statusStyle}">${esc(statusLabel)}</td>
          <td class="px-4 py-2 text-sm max-w-xs truncate" title="${esc(p.url)}">
            <a href="${esc(p.url)}" target="_blank" rel="noopener" class="hover:underline" style="color:var(--link)">${esc(p.url)}</a>
          </td>
          <td class="px-4 py-2 text-sm max-w-xs truncate" title="${esc(p.error || p.title || '')}">${blockedTag}${titleDisplay ? ' ' + esc(titleDisplay) : (blockedTag ? '' : '\u2014')}</td>
          <td class="px-4 py-2 text-xs ui-subtle">${esc(p.content_type ? p.content_type.split(';')[0] : '\u2014')}</td>
          <td class="px-4 py-2 text-xs ui-subtle text-right">${p.content_length > 0 ? fmtBytes(p.content_length) : '\u2014'}</td>
          <td class="px-4 py-2 text-xs ui-subtle text-right">${fmtFetch(p.fetch_time_ms)}</td>
          <td class="px-4 py-2 text-xs ui-subtle text-right" title="${esc(p.crawled_at || '')}">${fmtRelTime(p.crawled_at)}</td>
        </tr>`;
      }).join('');

  el.innerHTML = `
    <div class="surface">
      <div class="flex border-b border-[var(--border)] px-3 pt-1">${tabs}</div>
      <div class="p-3 border-b border-[var(--border)] flex flex-wrap items-center gap-3">
        <span class="text-xs ui-subtle">${fmtNum(total)} pages</span>
        <input class="ui-input text-xs px-2 py-1 flex-1 min-w-40 max-w-64" placeholder="Filter URL or title\u2026"
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
            <th class="px-4 py-2 text-right font-medium">Crawled</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>
      ${totalPages > 1 ? `
        <div class="p-3 border-t border-[var(--border)] flex items-center gap-2 text-xs">
          ${page > 1 ? `<button class="ui-btn px-2 py-1" onclick="loadScrapePages('${esc(domain)}',${page-1})">\u2190 Prev</button>` : ''}
          <span class="ui-subtle">Page ${page} / ${totalPages}</span>
          ${page < totalPages ? `<button class="ui-btn px-2 py-1" onclick="loadScrapePages('${esc(domain)}',${page+1})">Next \u2192</button>` : ''}
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

async function startScrapeDomainFull(domain) {
  try {
    const mode = $('sd-mode')?.value || 'http';
    const maxPages = parseInt($('sd-max-pages')?.value || '0', 10) || 0;
    const workers = parseInt($('sd-workers')?.value || '0', 10) || 0;
    const payload = { domain, mode, max_pages: maxPages, workers, store_body: true, resume: false };
    if (mode === 'worker') {
      payload.worker_token = ($('sd-worker-token')?.value || '').trim();
      payload.worker_browser = !!$('sd-worker-browser')?.checked;
    }
    await apiScrapeStart(payload);
    await loadScrapeDomainStatus(domain);
  } catch (e) {
    showScrapeActionMsg(e.message, 'error');
  }
}

function toggleDomainWorkerOpts() {
  const mode = $('sd-mode')?.value;
  const el = $('sd-worker-opts');
  if (el) {
    if (mode === 'worker') el.classList.remove('hidden');
    else el.classList.add('hidden');
  }
}

async function triggerScrapePipeline(domain) {
  try {
    const res = await apiScrapePipeline(domain);
    showScrapeActionMsg(`Started job ${res.job_id} \u2014 converting pages to markdown`, 'ok');
  } catch (e) {
    showScrapeActionMsg(e.message, 'error');
  }
}

async function triggerScrapeIndex(domain) {
  try {
    const res = await apiScrapeIndex(domain);
    showScrapeActionMsg(`Started index job ${res.job_id}`, 'ok');
    // Reload list after brief delay
    setTimeout(() => loadScrapeList(), 1000);
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
