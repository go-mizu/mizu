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
  overview: null,
  overviewLoadedAt: 0,
  metaStatus: null,
  jobs: null,
  jobsStreamSubscribed: false,
  engines: null,
  crawls: null,
  warcRows: null,
  warcDetail: null,
  warcSummary: null,
  warcOffset: 0,
  warcLimit: 100,
  warcQuery: '',
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
