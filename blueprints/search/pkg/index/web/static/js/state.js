// ===================================================================
// State
// ===================================================================
const state = {
  query: '',
  searchEngine: '',
  results: null,
  stats: null,
  loading: false,
  currentPage: '',
  browseShards: null,
  browseFiles: null,
  browseShard: '',
  browsePage: 1,
  browseQ: '',
  browseSort: 'date',
  browseView: 'docs',
  doc: null,
  theme: localStorage.getItem('fts-theme') || 'dark',
  // Central state — single source of truth shared across all pages
  // Restored from localStorage so overview renders instantly without any loading state
  central: (() => {
    try {
      const v = JSON.parse(localStorage.getItem('fts-central') || 'null');
      if (v) return { overview: v.overview || null, jobs: v.jobs || null, meta: v.meta || null, loadedAt: v.loadedAt || 0, loading: false };
    } catch (_) {}
    return { overview: null, jobs: null, meta: null, loadedAt: 0, loading: false };
  })(),
  // Legacy aliases — kept for backward compat during migration
  get overview() { return this.central.overview; },
  set overview(v) { this.central.overview = v; },
  get jobs() { return this.central.jobs; },
  set jobs(v) { this.central.jobs = v; },
  get metaStatus() { return this.central.meta; },
  set metaStatus(v) { this.central.meta = v; },
  overviewLoadedAt: 0,
  jobsStreamSubscribed: false,
  engines: null,
  crawls: null,
  warcRows: null,
  warcDetail: null,
  warcSummary: null,
  warcOffset: 0,
  warcLimit: 100,
  warcQuery: '',
  warcPhase: '',
  warcPageSize: 100,
  warcTotal: 0,
  warcSystem: null,
  // Parquet tab state
  parquetSubset: '',
  parquetQuery: '',
  parquetOffset: 0,
  parquetManifest: null,
  parquetSQL: '',
  parquetDetail: null,
  parquetDetailIdx: null,
  parquetDetailPage: 1,
  parquetDetailFilter: '',
  parquetDetailSort: '',
};

// ===================================================================
// Theme
// ===================================================================
function applyTheme() {
  const html = document.documentElement;
  if (state.theme === 'dark') {
    html.classList.add('dark');
    html.classList.remove('light');
    document.getElementById('icon-sun').classList.remove('hidden');
    document.getElementById('icon-moon').classList.add('hidden');
  } else {
    html.classList.remove('dark');
    html.classList.add('light');
    document.getElementById('icon-sun').classList.add('hidden');
    document.getElementById('icon-moon').classList.remove('hidden');
  }
}
function toggleTheme() {
  state.theme = state.theme === 'dark' ? 'light' : 'dark';
  localStorage.setItem('fts-theme', state.theme);
  applyTheme();
}
