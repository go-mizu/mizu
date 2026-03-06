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
  central: {
    overview: null,   // OverviewResponse from /api/overview
    jobs: null,       // []*Job from /api/jobs
    meta: null,       // MetaStatus from /api/meta/status
    loadedAt: 0,      // timestamp of last successful refresh
    loading: false,   // currently refreshing
  },
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
