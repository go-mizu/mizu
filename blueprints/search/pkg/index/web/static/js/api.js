// ===================================================================
// API Helpers
// ===================================================================
async function apiFetch(url) {
  const res = await fetch(url);
  if (!res.ok) {
    const body = await res.text();
    let msg = `HTTP ${res.status}`;
    try { const j = JSON.parse(body); if (j.error) msg = j.error; } catch {}
    throw new Error(msg);
  }
  return res.json();
}

async function apiPost(url, data) {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const body = await res.text();
    let msg = `HTTP ${res.status}`;
    try { const j = JSON.parse(body); if (j.error) msg = j.error; } catch {}
    throw new Error(msg);
  }
  return res.json();
}

async function apiDelete(url) {
  const res = await fetch(url, { method: 'DELETE' });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

async function apiSearch(q, limit = 20, offset = 0, engine = '') {
  const engineQ = engine ? `&engine=${encodeURIComponent(engine)}` : '';
  return apiFetch(`/api/search?q=${encodeURIComponent(q)}&limit=${limit}&offset=${offset}${engineQ}`);
}

async function apiStats(engine = '') {
  const engineQ = engine ? `?engine=${encodeURIComponent(engine)}` : '';
  return apiFetch('/api/stats' + engineQ);
}

async function apiDoc(shard, docid) {
  return apiFetch(`/api/doc/${shard}/${encodeURIComponent(docid)}`);
}

async function apiBrowse(shard = '', {page = 1, pageSize = 100, q = '', sort = ''} = {}) {
  if (!shard) return apiFetch('/api/browse');
  let url = `/api/browse?shard=${encodeURIComponent(shard)}&page=${page}&page_size=${pageSize}`;
  if (q) url += `&q=${encodeURIComponent(q)}`;
  if (sort) url += `&sort=${encodeURIComponent(sort)}`;
  return apiFetch(url);
}

async function apiMetaScanDocs() {
  return apiPost('/api/meta/scan-docs', {});
}

async function apiBrowseStats(shard) {
  return apiFetch(`/api/browse/stats?shard=${encodeURIComponent(shard)}`);
}

async function apiOverview() {
  return apiFetch('/api/overview');
}

async function apiMetaStatus(crawl = '') {
  const url = crawl ? `/api/meta/status?crawl=${encodeURIComponent(crawl)}` : '/api/meta/status';
  return apiFetch(url);
}

async function apiMetaRefresh(crawl = '', force = true) {
  return apiPost('/api/meta/refresh', { crawl, force });
}

async function apiEngines() {
  return apiFetch('/api/engines');
}

async function apiJobs() {
  return apiFetch('/api/jobs');
}

async function apiWARCList(opts = {}) {
  const params = new URLSearchParams();
  if (opts.offset !== undefined) params.set('offset', String(opts.offset));
  if (opts.limit !== undefined) params.set('limit', String(opts.limit));
  if (opts.q) params.set('q', opts.q);
  if (opts.phase) params.set('phase', opts.phase);
  if (opts.crawl) params.set('crawl', opts.crawl);
  const suffix = params.toString() ? `?${params.toString()}` : '';
  return apiFetch('/api/warc' + suffix);
}

async function apiWARCDetail(index, crawl = '') {
  const suffix = crawl ? `?crawl=${encodeURIComponent(crawl)}` : '';
  return apiFetch(`/api/warc/${encodeURIComponent(index)}${suffix}`);
}

async function apiWARCAction(index, payload) {
  return apiPost(`/api/warc/${encodeURIComponent(index)}/action`, payload || {});
}

async function apiCreateJob(cfg) {
  return apiPost('/api/jobs', cfg);
}

async function apiCancelJob(id) {
  return apiDelete(`/api/jobs/${id}`);
}

async function apiClearJobs() {
  return apiDelete('/api/jobs');
}

async function apiGetJob(id) {
  return apiFetch(`/api/jobs/${encodeURIComponent(id)}`);
}

async function apiDomains(opts = {}) {
  const p = new URLSearchParams();
  if (opts.sort) p.set('sort', opts.sort);
  if (opts.page) p.set('page', String(opts.page));
  if (opts.pageSize) p.set('page_size', String(opts.pageSize));
  if (opts.q) p.set('q', opts.q);
  const qs = p.toString();
  return apiFetch('/api/domains' + (qs ? '?' + qs : ''));
}

async function apiDomainDetail(domain, opts = {}) {
  const p = new URLSearchParams();
  if (opts.sort) p.set('sort', opts.sort);
  if (opts.page) p.set('page', String(opts.page));
  if (opts.pageSize) p.set('page_size', String(opts.pageSize));
  const qs = p.toString();
  return apiFetch('/api/domains/' + encodeURIComponent(domain) + (qs ? '?' + qs : ''));
}

function currentSearchEngine() {
  if (state.searchEngine) return state.searchEngine;
  if (state.stats && state.stats.engine) return state.stats.engine;
  if (state.engines && state.engines.length > 0) return state.engines[0];
  return DEFAULT_ENGINE;
}

function searchEngineOptions() {
  const engines = (state.engines && state.engines.length > 0) ? state.engines.slice() : [currentSearchEngine()];
  const selected = currentSearchEngine();
  return engines.map(e => `<option value="${esc(e)}"${e === selected ? ' selected' : ''}>${esc(e)}</option>`).join('');
}

function syncSearchEngineControls() {
  const selected = currentSearchEngine();
  const ids = ['header-search-engine', 'home-search-engine'];
  for (const id of ids) {
    const el = $(id);
    if (!el) continue;
    el.innerHTML = searchEngineOptions();
    el.value = selected;
  }
}

async function ensureEnginesLoaded() {
  if (state.engines !== null) return state.engines;
  try {
    const d = await apiEngines();
    state.engines = d.engines || [];
  } catch {
    state.engines = [];
  }
  syncSearchEngineControls();
  return state.engines;
}

function buildSearchURL(query, offset = 0) {
  const params = new URLSearchParams();
  params.set('q', query);
  if (offset > 0) params.set('offset', String(offset));
  const engine = currentSearchEngine();
  if (engine) params.set('engine', engine);
  return `/search?${params.toString()}`;
}

function applySearchEngine(engine) {
  state.searchEngine = (engine || '').trim();
  syncSearchEngineControls();
  if (state.query) {
    navigateTo(buildSearchURL(state.query, 0));
    return;
  }
  if (state.currentPage === 'search') {
    loadHomeStats();
  }
}
