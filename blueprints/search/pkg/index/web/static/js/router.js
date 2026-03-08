// ===================================================================
// Router
// ===================================================================
function navigateTo(path) { location.hash = path; }

function route() {
  const hash = (location.hash || '#/').slice(1);
  const qIdx = hash.indexOf('?');
  const path = qIdx >= 0 ? hash.slice(0, qIdx) : hash;
  const params = new URLSearchParams(qIdx >= 0 ? hash.slice(qIdx + 1) : '');
  if (jobsPollingTimer && path !== '/jobs') {
    clearInterval(jobsPollingTimer);
    jobsPollingTimer = null;
  }
  if (isDashboard) {
    refreshCentralState().catch(() => {});
  }

  updateActiveTab(path);

  if (path === '/' || path === '') {
    showHeaderSearch(false);
    if (isDashboard) { renderOverview(); } else { renderSearchHome(); }
  } else if (path === '/jobs' && isDashboard) {
    showHeaderSearch(false);
    renderJobs();
  } else if (path.startsWith('/jobs/') && isDashboard) {
    showHeaderSearch(false);
    const jobId = path.split('/')[2] || '';
    renderJobDetail(jobId);
  } else if (path === '/search') {
    const q = params.get('q') || '';
    const engine = params.get('engine') || '';
    if (q) {
      showHeaderSearch(true, q, engine);
      doSearchWithRender(q, parseInt(params.get('offset') || '0', 10), engine);
    } else {
      if (engine) state.searchEngine = engine;
      showHeaderSearch(false);
      renderSearchHome();
    }
  } else if (path === '/parquet' && isDashboard) {
    showHeaderSearch(false);
    renderParquet();
  } else if (path.startsWith('/parquet/subset/') && isDashboard) {
    showHeaderSearch(false);
    const subset = path.split('/')[3] || '';
    renderParquetSubsetStats(subset);
  } else if (path.startsWith('/parquet/') && isDashboard) {
    showHeaderSearch(false);
    const idx = path.split('/')[2] || '';
    renderParquetDetail(idx);
  } else if (path === '/warc' && isDashboard) {
    showHeaderSearch(false);
    renderWARC();
  } else if (path.startsWith('/warc/') && isDashboard) {
    showHeaderSearch(false);
    const idx = path.split('/')[2] || '';
    renderWARCDetail(idx);
  } else if (path === '/domains' && isDashboard) {
    showHeaderSearch(false);
    renderDomains();
  } else if (path.startsWith('/domains/') && isDashboard) {
    showHeaderSearch(false);
    const domain = decodeURIComponent(path.slice('/domains/'.length));
    renderDomainDetail(domain);
  } else if (path.startsWith('/browse')) {
    const shard = path.split('/')[2] || '';
    showHeaderSearch(false);
    // Avoid full re-render if already on browse page — just switch shard.
    if (state.currentPage === 'browse' && shard && state.browseShards) {
      switchBrowseShard(shard);
    } else {
      renderBrowse(shard);
    }
  } else if (path.startsWith('/doc/')) {
    const parts = path.split('/');
    const shard = parts[2];
    const docid = parts.slice(3).join('/');
    showHeaderSearch(true, state.query, state.searchEngine);
    renderDoc(shard, docid);
  } else {
    showHeaderSearch(false);
    if (isDashboard) { renderOverview(); } else { renderSearchHome(); }
  }
}

function updateActiveTab(path) {
  let activeTab = isDashboard ? 'overview' : 'search';
  if (path === '/jobs' || path.startsWith('/jobs/')) activeTab = 'jobs';
  else if (path === '/search' || path === '/' && !isDashboard || path.startsWith('/doc/')) activeTab = 'search';
  else if (path === '/parquet' || path.startsWith('/parquet/')) activeTab = 'parquet';
  else if (path === '/domains' || path.startsWith('/domains/')) activeTab = 'domains';
  else if (path === '/warc' || path.startsWith('/warc/')) activeTab = 'warc';
  else if (path.startsWith('/browse')) activeTab = 'browse';
  else if (path === '/' && isDashboard) activeTab = 'overview';

  document.querySelectorAll('#main-nav a[data-tab]').forEach(a => {
    if (a.dataset.tab === activeTab) {
      a.className = 'text-sm pb-1 tab-active transition-colors cursor-pointer';
    } else {
      a.className = 'text-sm pb-1 tab-inactive transition-colors cursor-pointer';
    }
  });
}

function showHeaderSearch(visible, value, engine) {
  const el = document.getElementById('header-search');
  const input = document.getElementById('header-search-input');
  if (visible) {
    el.classList.remove('hidden');
    el.classList.add('flex');
    if (value !== undefined) input.value = value;
    if (engine) state.searchEngine = engine;
    syncSearchEngineControls();
  } else {
    el.classList.add('hidden');
    el.classList.remove('flex');
  }
}
