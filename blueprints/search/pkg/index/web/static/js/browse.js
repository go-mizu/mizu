// ===================================================================
// Tab 6: Browse
// ===================================================================
async function renderBrowse(shard) {
  state.currentPage = 'browse';
  state.browseShard = shard;
  state.browsePage = 1;
  state.browseQ = '';
  state.browseSort = 'date';
  state.browseView = state.browseView || 'docs';
  $('main').innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <h1 class="page-title">Browse</h1>
        ${isDashboard ? `
        <div class="flex items-center gap-3">
          <span id="browse-refreshed-at" class="text-[11px] font-mono ui-subtle"></span>
          <button id="browse-refresh-btn" onclick="browseRefresh()" class="ui-btn px-3 py-2 text-xs font-mono">Refresh</button>
        </div>` : ''}
      </div>
      <div class="surface flex min-h-[calc(100vh-10rem)]">
        <aside class="w-56 shrink-0 p-3 ui-border-r overflow-y-auto">
          <div class="text-xs font-mono ui-subtle mb-3 uppercase tracking-wider">Shards</div>
          <div id="shard-list" class="space-y-0.5">
            <div class="ui-empty">loading\u2026</div>
          </div>
        </aside>
        <div class="flex-1 min-w-0 p-4" id="browse-content">
          <div class="ui-empty">loading\u2026</div>
        </div>
      </div>
    </div>`;

  try {
    if (isDashboard) {
      await refreshDashboardContext().catch(() => {});
    }
    const data = await apiBrowse();
    state.browseShards = data.shards || [];
    renderShardList(shard);
    updateBrowseRefreshedAt();
    if (shard) {
      loadShardView(shard);
    } else if (state.browseShards.length > 0) {
      navigateTo('/browse/' + state.browseShards[0].name);
    } else {
      $('browse-content').innerHTML = `<div class="ui-empty">No WARC shards found. Run the download pipeline steps first.</div>`;
    }
  } catch(e) {
    $('browse-content').innerHTML = `<div class="text-xs text-red-400 py-8">${esc(e.message)}</div>`;
  }
}

function updateBrowseRefreshedAt() {
  const el = $('browse-refreshed-at');
  if (!el) return;
  // Find the most recently scanned shard
  const shards = state.browseShards || [];
  let latest = '';
  for (const s of shards) {
    if (s.last_scanned_at && (!latest || s.last_scanned_at > latest)) latest = s.last_scanned_at;
  }
  el.textContent = latest ? `Refreshed ${fmtRelativeTime(latest)}` : '';
}

async function browseRefresh() {
  const btn = $('browse-refresh-btn');
  if (btn) { btn.disabled = true; btn.textContent = 'Refreshing…'; }
  try {
    await Promise.all([
      refreshDashboardMeta(true),
      apiMetaScanDocs(),
    ]);
  } catch(_) {}
  // Reload shard list
  try {
    const data = await apiBrowse();
    state.browseShards = data.shards || [];
    renderShardList(state.browseShard);
    updateBrowseRefreshedAt();
  } catch(_) {}
  if (btn) { btn.disabled = false; btn.textContent = 'Refresh'; }
}

function renderShardList(active) {
  const el = $('shard-list');
  if (!el || !state.browseShards) return;
  if (state.browseShards.length === 0) {
    el.innerHTML = `<div class="ui-empty">No shards</div>`;
    return;
  }
  el.innerHTML = state.browseShards.map(s => {
    const isActive = s.name === active;
    const hasPack = !!s.has_pack;
    const hasScan = !!s.has_scan;
    const scanning = !!s.scanning;
    const count = (s.file_count ?? 0).toLocaleString();

    const chips = [];
    if (hasPack) chips.push(`<span class="ui-chip ui-chip-ok">packed</span>`);
    else chips.push(`<span class="ui-chip ui-chip-off">raw</span>`);
    if (scanning) chips.push(`<span class="ui-chip" style="border-color:rgba(96,165,250,0.6);color:#93c5fd">scanning</span>`);
    else if (hasScan) chips.push(`<span class="ui-chip ui-chip-ok">indexed</span>`);

    const packBtn = !hasPack && isDashboard
      ? `<button onclick="event.preventDefault();triggerPackShard('${esc(s.name)}')" class="ml-1 text-[9px] font-mono px-1.5 py-0.5 ui-btn">Pack</button>`
      : '';

    return `<a href="#/browse/${s.name}"
       class="block py-1.5 px-2 text-xs font-mono cursor-pointer transition-colors ${isActive ? 'shard-active' : 'shard-item'}">
      <div class="flex items-center justify-between gap-1">
        <span class="truncate">${esc(s.name)}</span>
        <span class="ui-subtle shrink-0">${count}</span>
      </div>
      <div class="flex items-center gap-1 mt-1 flex-wrap">
        ${chips.join('')}${packBtn}
        ${s.total_size ? `<span class="text-[9px] font-mono ui-subtle ml-auto">${fmtBytes(s.total_size)}</span>` : ''}
      </div>
    </a>`;
  }).join('');
}

async function triggerPackShard(shard) {
  // shard is "00000", "00001", etc. — parseInt gives the manifest index.
  const fileIdx = parseInt(shard, 10);
  try {
    await apiPost('/api/jobs', {type: 'markdown', files: String(fileIdx)});
  } catch(e) {
    alert('Failed to start pack job: ' + e.message);
    return;
  }
  // Reload shard list so the new job shows up in the sidebar.
  try {
    const data = await apiBrowse();
    state.browseShards = data.shards || [];
    renderShardList(state.browseShard);
  } catch(_) {}
}

// Browse filter debounce timer
let browseFilterTimer = null;

function loadShardView(shard) {
  state.browseShard = shard;
  if (state.browseView === 'stats') {
    loadShardStats(shard);
  } else {
    loadShardDocs(shard, 1);
  }
}

function renderBrowseViewTabs(shard, activeView) {
  return `
    <div class="flex items-center gap-4 border-b mb-4 pb-0">
      <button onclick="state.browseView='docs';loadShardDocs('${esc(shard)}',1)"
        class="text-xs pb-2 transition-colors ${activeView==='docs' ? 'tab-active' : 'tab-inactive'}">Docs</button>
      <button onclick="state.browseView='stats';loadShardStats('${esc(shard)}')"
        class="text-xs pb-2 transition-colors ${activeView==='stats' ? 'tab-active' : 'tab-inactive'}">Stats</button>
    </div>`;
}

async function loadShardDocs(shard, page = 1) {
  state.browseView = 'docs';
  const el = $('browse-content');
  if (!el) return;
  el.innerHTML = `<div class="ui-empty">loading\u2026</div>`;

  const q = state.browseQ || '';
  const sort = state.browseSort || 'date';

  try {
    const data = await apiBrowse(shard, {page, pageSize: 100, q, sort});

    if (data.not_scanned) {
      el.innerHTML = renderBrowseViewTabs(shard, 'docs') + `
        <div class="ui-empty mt-6 text-center">
          <div class="mb-3">Shard not yet indexed.</div>
          ${isDashboard ? `<button onclick="apiMetaScanDocs().then(()=>loadShardDocs('${esc(shard)}',1))" class="ui-btn px-4 py-2 text-xs font-mono">Scan Docs</button>` : ''}
        </div>`;
      return;
    }

    renderDocTable(shard, data, page);
  } catch(e) {
    el.innerHTML = `<div class="text-xs text-red-400 py-4">${esc(e.message)}</div>`;
  }
}

function renderDocTable(shard, data, page) {
  const el = $('browse-content');
  if (!el) return;
  const docs = data.docs || [];
  const total = data.total || 0;
  const pageSize = data.page_size || 100;
  const totalPages = Math.ceil(total / pageSize);
  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, total);
  const stale = data.meta_stale;

  el.innerHTML = `
    ${renderBrowseViewTabs(shard, 'docs')}
    ${stale ? `<div class="mb-3 px-3 py-2 text-xs text-amber-600 bg-amber-50 dark:bg-amber-950/30 dark:text-amber-400 border border-amber-200 dark:border-amber-900">Refreshing document metadata in the background…</div>` : ''}
    <div class="flex items-center gap-3 mb-4 flex-wrap">
      <span class="meta-line">${start}–${end} of ${total.toLocaleString()}</span>
      <input id="browse-filter" type="search" placeholder="Filter by title or URL…" value="${esc(state.browseQ || '')}"
        class="ml-auto ui-input text-xs px-2 py-1 w-56" oninput="debounceBrowseFilter(this.value, '${esc(shard)}')">
      <select id="browse-sort" class="ui-input text-xs px-2 py-1" onchange="state.browseSort=this.value;loadShardDocs('${esc(shard)}',1)">
        <option value="date" ${(state.browseSort||'date')==='date'?'selected':''}>Date ↓</option>
        <option value="size" ${state.browseSort==='size'?'selected':''}>Size ↓</option>
        <option value="words" ${state.browseSort==='words'?'selected':''}>Words ↓</option>
        <option value="title" ${state.browseSort==='title'?'selected':''}>Title A–Z</option>
        <option value="url" ${state.browseSort==='url'?'selected':''}>URL A–Z</option>
      </select>
    </div>
    <div class="overflow-x-auto">
    <table class="w-full text-xs ui-table">
      <thead>
        <tr class="text-left">
          <th class="pb-2 pr-3 font-medium w-2/5">Title</th>
          <th class="pb-2 pr-3 font-medium">URL</th>
          <th class="pb-2 pr-3 font-medium text-right whitespace-nowrap">Date</th>
          <th class="pb-2 pr-3 font-medium text-right">Size</th>
          <th class="pb-2 font-medium text-right">Words</th>
        </tr>
      </thead>
      <tbody>
        ${docs.map((d, i) => `
          <tr class="file-row anim-fade-up" style="animation-delay:${Math.min(i,20)*10}ms">
            <td class="py-2 pr-3">
              <a href="#/doc/${shard}/${encodeURIComponent(d.doc_id)}" class="ui-link font-medium truncate block max-w-xs" title="${esc(d.title||d.doc_id)}">
                ${esc(d.title || d.doc_id)}
              </a>
            </td>
            <td class="py-2 pr-3 max-w-[240px]">
              ${d.url ? `<a href="${esc(d.url)}" target="_blank" rel="noopener noreferrer" class="ui-subtle hover:text-[var(--accent)] font-mono truncate block" title="${esc(d.url)}">${truncateURL(d.url, 40)}</a>` : ''}
            </td>
            <td class="py-2 pr-3 ui-subtle text-right whitespace-nowrap">${d.crawl_date ? fmtDate(d.crawl_date) : ''}</td>
            <td class="py-2 pr-3 ui-subtle text-right whitespace-nowrap">${d.size_bytes ? fmtBytes(d.size_bytes) : ''}</td>
            <td class="py-2 ui-subtle text-right whitespace-nowrap">${d.word_count ? d.word_count.toLocaleString() : ''}</td>
          </tr>`).join('')}
      </tbody>
    </table>
    </div>
    ${totalPages > 1 ? `
    <div class="flex items-center justify-between mt-4 text-xs">
      <button onclick="loadShardDocs('${esc(shard)}', ${page - 1})" ${page <= 1 ? 'disabled' : ''} class="ui-btn px-3 py-1.5">&larr; Prev</button>
      <span class="ui-subtle">Page ${page} of ${totalPages}</span>
      <button onclick="loadShardDocs('${esc(shard)}', ${page + 1})" ${page >= totalPages ? 'disabled' : ''} class="ui-btn px-3 py-1.5">Next &rarr;</button>
    </div>` : ''}`;
}

async function loadShardStats(shard) {
  state.browseView = 'stats';
  const el = $('browse-content');
  if (!el) return;
  el.innerHTML = renderBrowseViewTabs(shard, 'stats') + `<div class="ui-empty">loading stats\u2026</div>`;
  try {
    const stats = await apiBrowseStats(shard);
    renderShardStats(shard, stats);
  } catch(e) {
    el.innerHTML = renderBrowseViewTabs(shard, 'stats') + `<div class="text-xs text-red-400 py-4">${esc(e.message)}</div>`;
  }
}

function renderShardStats(shard, stats) {
  const el = $('browse-content');
  if (!el) return;

  const s = stats || {};
  const totalDocs = (s.total_docs || 0).toLocaleString();
  const totalSize = s.total_size ? fmtBytes(s.total_size) : '—';
  const avgSize = s.avg_size ? fmtBytes(Math.round(s.avg_size)) : '—';
  const minSize = s.min_size ? fmtBytes(s.min_size) : '—';
  const maxSize = s.max_size ? fmtBytes(s.max_size) : '—';
  const dateFrom = s.date_from ? fmtDate(s.date_from) : '—';
  const dateTo = s.date_to ? fmtDate(s.date_to) : '—';

  // Summary stat cards
  const statCards = [
    {label: 'Documents', value: totalDocs},
    {label: 'Total Size', value: totalSize},
    {label: 'Avg Size', value: avgSize},
    {label: 'Size Range', value: `${minSize} – ${maxSize}`},
    {label: 'Date Range', value: dateFrom === dateTo ? dateFrom : `${dateFrom} – ${dateTo}`},
  ];

  // Top domains bar chart
  const domains = (s.top_domains || []).slice(0, 20);
  const domainMax = domains.reduce((m, d) => Math.max(m, d.count || 0), 0) || 1;

  // Size buckets
  const buckets = s.size_buckets || [];
  const bucketTotal = buckets.reduce((m, b) => m + (b.count || 0), 0) || 1;
  const bucketColors = ['ov-c2', 'ov-c4', 'ov-c3', 'ov-c1', 'ov-c5'];

  // Date histogram
  const histogram = (s.date_histogram || []).slice(-60);
  const histMax = histogram.reduce((m, h) => Math.max(m, h.count || 0), 0) || 1;

  el.innerHTML = `
    ${renderBrowseViewTabs(shard, 'stats')}

    <!-- Stat cards -->
    <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-px border border-[var(--border)] mb-6" style="background:var(--border)">
      ${statCards.map(c => `
        <div class="bg-[var(--panel)] px-4 py-3">
          <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider mb-1">${esc(c.label)}</div>
          <div class="text-base font-semibold tracking-tight">${esc(c.value)}</div>
        </div>`).join('')}
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
      <!-- Top Domains -->
      <div class="surface p-4">
        <div class="text-xs font-semibold mb-4 uppercase tracking-wider ui-subtle">Top Domains by Doc Count</div>
        ${domains.length === 0
          ? `<div class="ui-empty">No data</div>`
          : `<div class="space-y-1.5">
            ${domains.map(d => `
              <div class="flex items-center gap-2 text-xs">
                <span class="w-32 shrink-0 font-mono ui-subtle truncate" title="${esc(d.domain)}">${esc(d.domain)}</span>
                <div class="flex-1 progress-track" style="height:4px">
                  <div class="progress-fill" style="width:${Math.max(2, (d.count/domainMax)*100).toFixed(1)}%"></div>
                </div>
                <span class="w-10 shrink-0 text-right font-mono ui-subtle">${(d.count||0).toLocaleString()}</span>
              </div>`).join('')}
          </div>`}
      </div>

      <!-- Size Distribution -->
      <div class="surface p-4">
        <div class="text-xs font-semibold mb-4 uppercase tracking-wider ui-subtle">Size Distribution</div>
        ${buckets.length === 0
          ? `<div class="ui-empty">No data</div>`
          : `
          <div class="ov-stacked mb-4">
            ${buckets.map((b, i) => `
              <div class="${bucketColors[i%bucketColors.length]} ov-stacked-seg" style="width:${Math.max(1,(b.count/bucketTotal)*100).toFixed(1)}%" title="${esc(b.label)}: ${(b.count||0).toLocaleString()}"></div>`).join('')}
          </div>
          <div class="space-y-2">
            ${buckets.map((b, i) => `
              <div class="flex items-center gap-2 text-xs">
                <div class="ov-legend-dot ${bucketColors[i%bucketColors.length]}"></div>
                <span class="flex-1 font-mono ui-subtle">${esc(b.label)}</span>
                <span class="font-mono ui-subtle">${(b.count||0).toLocaleString()}</span>
                <span class="font-mono ui-subtle w-10 text-right">${((b.count/bucketTotal)*100).toFixed(0)}%</span>
              </div>`).join('')}
          </div>`}
      </div>
    </div>

    <!-- Date Histogram -->
    <div class="surface p-4">
      <div class="text-xs font-semibold mb-4 uppercase tracking-wider ui-subtle">Documents by Crawl Date (last 60 days)</div>
      ${histogram.length === 0
        ? `<div class="ui-empty">No date data</div>`
        : `
        <div class="flex items-end gap-px" style="height:80px">
          ${histogram.map(h => {
            const pct = Math.max(2, ((h.count||0) / histMax) * 100);
            return `<div class="flex-1 bg-[var(--accent)] opacity-70 hover:opacity-100 transition-opacity cursor-default"
              style="height:${pct.toFixed(1)}%" title="${esc(h.date)}: ${(h.count||0).toLocaleString()}"></div>`;
          }).join('')}
        </div>
        <div class="flex justify-between mt-1 text-[9px] font-mono ui-subtle">
          <span>${histogram[0] ? esc(histogram[0].date) : ''}</span>
          <span>${histogram[histogram.length-1] ? esc(histogram[histogram.length-1].date) : ''}</span>
        </div>`}
    </div>`;
}

function debounceBrowseFilter(val, shard) {
  state.browseQ = val;
  clearTimeout(browseFilterTimer);
  browseFilterTimer = setTimeout(() => loadShardDocs(shard, 1), 300);
}

// ── Browse helpers ────────────────────────────────────────────────────────────

function fmtDate(isoStr) {
  if (!isoStr) return '';
  const d = new Date(isoStr);
  if (isNaN(d)) return isoStr;
  return d.toLocaleDateString('en-US', {month: 'short', day: 'numeric', year: 'numeric'});
}

function truncateURL(url, maxLen) {
  if (!url || url.length <= maxLen) return esc(url);
  const half = Math.floor((maxLen - 3) / 2);
  return esc(url.slice(0, half) + '…' + url.slice(-half));
}
