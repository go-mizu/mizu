// ===================================================================
// Tab 4: Search
// ===================================================================
function renderSearchHome() {
  state.currentPage = 'search';
  $('main').innerHTML = `
    <div class="flex flex-col items-center justify-center min-h-[calc(100vh-6rem)] anim-fade-in">
      <div class="w-full max-w-xl space-y-8">
        <div class="text-center space-y-3">
          <h1 class="text-3xl font-semibold tracking-tight">What are you looking for?</h1>
        </div>
        <div class="relative">
          <input id="home-search-input" type="text"
            placeholder="Type to search\u2026"
            class="ui-input w-full h-12 px-4 text-base"
            onkeydown="if(event.key==='Enter')doSearch(this.value)" autofocus>
          <kbd class="absolute right-3 top-1/2 -translate-y-1/2 ui-kbd pointer-events-none"><span class="text-[10px]">&#8984;</span>K</kbd>
        </div>
        <div class="flex items-center justify-center">
          <label class="text-xs font-mono ui-subtle mr-2">Engine</label>
          <select id="home-search-engine"
            class="ui-select h-8 text-xs px-2"
            onchange="applySearchEngine(this.value)">
          </select>
        </div>
        <div id="home-stats" class="text-center">
          <span class="text-xs font-mono ui-subtle">loading\u2026</span>
        </div>
        ${isDashboard ? `<div class="text-center"><button onclick="refreshDashboardMeta(true)" class="ui-btn px-3 py-2 text-xs font-mono">Refresh Metadata</button></div>` : ''}
        <div id="home-meta" class="text-center">
          <span class="meta-line">${isDashboard ? 'loading metadata...' : ''}</span>
        </div>
      </div>
    </div>`;
  syncSearchEngineControls();
  ensureEnginesLoaded().then(() => loadHomeStats());
  if (isDashboard) {
    refreshDashboardContext().then(() => {
      const m = $('home-meta');
      if (m) m.innerHTML = `<span class="meta-line">${esc(renderMetaSummaryLine())}</span>`;
    }).catch(() => {});
  }
  setTimeout(() => { const i = $('home-search-input'); if (i) i.focus(); }, 50);
}

async function loadHomeStats() {
  const el = $('home-stats');
  if (!el) return;
  try {
    const d = await apiStats(currentSearchEngine());
    state.stats = d;
    if (d.engine) state.searchEngine = d.engine;
    syncSearchEngineControls();
    el.innerHTML = `<span class="meta-line">${esc(d.engine)} &middot; ${(d.total_docs||0).toLocaleString()} docs &middot; ${d.shards||0} shards &middot; ${d.total_disk||'\u2014'}</span>`;
  } catch(e) {
    el.innerHTML = `<span class="meta-line">No index yet. Run <code>search cc fts index</code> to get started.</span>`;
  }
}

function doSearch(query) {
  if (!query.trim()) return;
  state.query = query.trim();
  navigateTo(buildSearchURL(state.query, 0));
}

async function doSearchWithRender(query, offset = 0, engine = '') {
  if (!query.trim()) { renderSearchHome(); return; }
  if (engine) state.searchEngine = engine;
  await ensureEnginesLoaded();
  state.query = query.trim();
  state.loading = true;
  state.currentPage = 'search';
  syncSearchEngineControls();
  renderSearchLoading();

  try {
    const data = await apiSearch(query, 20, offset, currentSearchEngine());
    if (data.engine) state.searchEngine = data.engine;
    syncSearchEngineControls();
    state.results = data;
    state.loading = false;
    renderSearchResults(data, query, offset);
  } catch (err) {
    state.loading = false;
    renderError('Search failed: ' + err.message);
  }
}

function renderSearchLoading() {
  const lines = Array(6).fill('').map((_, i) => `
    <div class="py-5 anim-fade-up" style="animation-delay:${i*40}ms">
      <div class="h-3 w-48 ui-skeleton mb-2"></div>
      <div class="h-4 w-2/5 ui-skeleton mb-2"></div>
      <div class="h-3 w-full ui-skeleton mb-1"></div>
      <div class="h-3 w-4/5 ui-skeleton"></div>
    </div>`).join('<div class="border-t"></div>');
  $('main').innerHTML = `<div class="py-8 max-w-3xl mx-auto">${lines}</div>`;
}

function renderSearchResults(data, query, offset) {
  if (!data || !data.hits || data.hits.length === 0) {
    $('main').innerHTML = `
      <div class="py-20 text-center anim-fade-in max-w-3xl mx-auto">
        <p class="ui-subtle text-sm mb-1">Nothing matched <strong>&laquo;${esc(query)}&raquo;</strong></p>
        <p class="ui-subtle text-xs">Try different terms or broaden your query</p>
      </div>`;
    return;
  }

  const pageSize = 20;
  const page = Math.floor(offset / pageSize) + 1;
  const totalPages = Math.ceil((data.total || data.hits.length) / pageSize);

  const results = data.hits.map((hit, i) => {
    const title = hit.title || docTitle(hit.doc_id);
    const url = hit.url || '';
    const domain = url ? (() => { try { return new URL(url).hostname; } catch { return ''; } })() : '';
    const chips = [];
    if (hit.crawl_date) chips.push(`<span>${fmtDate(hit.crawl_date)}</span>`);
    if (hit.size_bytes) chips.push(`<span>${fmtBytes(hit.size_bytes)}</span>`);
    if (hit.word_count) chips.push(`<span>${hit.word_count.toLocaleString()} words</span>`);
    if (hit.score) chips.push(`<span class="opacity-50">score ${hit.score.toFixed(2)}</span>`);

    return `
    <a href="#/doc/${hit.shard || '00000'}/${encodeURIComponent(hit.doc_id)}"
       class="block result-item py-5 -mx-2 px-2 cursor-pointer anim-fade-up"
       style="animation-delay:${i * 30}ms">
      ${domain ? `<div class="flex items-center gap-1.5 mb-1">
        <span class="text-[10px] font-mono ui-subtle">${esc(domain)}</span>
        ${url ? `<span class="text-[10px] font-mono ui-subtle opacity-50">·</span>
        <span class="text-[10px] font-mono ui-subtle truncate max-w-sm">${truncateURL(url, 60)}</span>` : ''}
      </div>` : `<div class="text-[10px] font-mono ui-subtle mb-1">${esc(hit.shard || '')} · ${esc(hit.doc_id)}</div>`}
      <div class="text-base font-medium leading-snug mb-1.5">${esc(title)}</div>
      ${hit.snippet ? `<p class="text-sm ui-subtle leading-relaxed line-clamp-3 mb-2">${highlight(hit.snippet, query)}</p>` : ''}
      ${chips.length ? `<div class="flex items-center gap-3 text-[10px] font-mono ui-subtle flex-wrap">${chips.join('')}</div>` : ''}
    </a>`;
  }).join('<div class="border-t"></div>');

  $('main').innerHTML = `
    <div class="py-8 max-w-3xl mx-auto anim-fade-in">
      <div class="meta-line mb-6">
        ${(data.total || data.hits.length).toLocaleString()} results${data.elapsed_ms ? ` · ${data.elapsed_ms}ms` : ''}${data.shards ? ` · ${data.shards} shard${data.shards!==1?'s':''}` : ''}${data.engine ? ` · ${esc(data.engine)}` : ''}
      </div>
      <div>${results}</div>
      ${totalPages > 1 ? renderSearchPagination(page, totalPages, query, pageSize, data.engine || currentSearchEngine()) : ''}
    </div>`;
}

function renderSearchPagination(cur, total, query, pageSize, engine) {
  let pages = [];
  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i);
  } else {
    pages.push(1);
    if (cur > 3) pages.push('\u2026');
    for (let i = Math.max(2, cur - 1); i <= Math.min(total - 1, cur + 1); i++) pages.push(i);
    if (cur < total - 2) pages.push('\u2026');
    pages.push(total);
  }

  const btn = (p, active) => {
    if (p === '\u2026') return `<span class="px-2 py-1 text-xs ui-subtle">\u2026</span>`;
    const cls = active
      ? 'px-2 py-1 text-xs font-mono font-medium'
      : 'px-2 py-1 text-xs font-mono ui-link transition-colors';
    const params = new URLSearchParams({ q: query, offset: String((p-1)*pageSize), engine: engine || currentSearchEngine() });
    return `<a href="#/search?${params.toString()}" class="${cls}">${p}</a>`;
  };

  const prevParams = new URLSearchParams({ q: query, offset: String((cur-2)*pageSize), engine: engine || currentSearchEngine() });
  const nextParams = new URLSearchParams({ q: query, offset: String(cur*pageSize), engine: engine || currentSearchEngine() });
  return `
    <div class="flex items-center justify-center gap-1 pt-8 pb-4">
      ${cur > 1 ? `<a href="#/search?${prevParams.toString()}" class="text-xs font-mono ui-link transition-colors mr-2">&larr; prev</a>` : ''}
      ${pages.map(p => btn(p, p === cur)).join('')}
      ${cur < total ? `<a href="#/search?${nextParams.toString()}" class="text-xs font-mono ui-link transition-colors ml-2">next &rarr;</a>` : ''}
    </div>`;
}
