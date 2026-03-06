// ===================================================================
// Error renderer
// ===================================================================
function renderError(msg) {
  $('main').innerHTML = `<div class="py-16 text-center anim-fade-in"><p class="text-red-400 text-sm">${esc(msg)}</p></div>`;
}

// ===================================================================
// Keyboard Shortcuts
// ===================================================================
document.addEventListener('keydown', (e) => {
  const isInput = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement.tagName);

  // Cmd+K — focus search input
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault();
    const h = $('header-search-input');
    const home = $('home-search-input');
    if (home && !home.closest('.hidden')) { home.focus(); home.select(); }
    else if (h && !h.closest('.hidden')) { h.focus(); h.select(); }
    else {
      navigateTo('/search');
    }
  }

  // / — focus search (when not in input)
  if (e.key === '/' && !isInput) {
    e.preventDefault();
    const h = $('header-search-input');
    const home = $('home-search-input');
    if (home && !home.closest('.hidden')) home.focus();
    else if (h && !h.closest('.hidden')) h.focus();
    else navigateTo('/search');
  }

  // Escape — blur active input
  if (e.key === 'Escape') document.activeElement.blur();

  // Number keys — switch tabs (when not in input)
  if (!isInput && e.key >= '1' && e.key <= '7' && !e.metaKey && !e.ctrlKey && !e.altKey) {
    const tabs = isDashboard
      ? ['/', '/search', '/browse', '/warc', '/jobs']
      : ['/search', '/browse'];
    const idx = parseInt(e.key) - 1;
    if (idx >= 0 && idx < tabs.length) {
      e.preventDefault();
      navigateTo(tabs[idx]);
    }
  }
});

// ===================================================================
// Init
// ===================================================================
applyTheme();

// In search-only mode: hide dashboard tabs, narrow layout.
if (!isDashboard) {
  document.querySelectorAll('#main-nav a[data-tab]').forEach(a => {
    if (['overview', 'jobs', 'warc'].includes(a.dataset.tab)) a.style.display = 'none';
  });
  const statusEl = $('header-status');
  if (statusEl) statusEl.style.display = 'none';
  const main = document.getElementById('main');
  if (main) { main.classList.remove('max-w-5xl', 'max-w-6xl'); main.classList.add('max-w-3xl'); }
} else {
  startMetaWatchdog();
}

// WebSocket only in dashboard mode.
if (isDashboard) { wsClient.connect(); }

window.addEventListener('hashchange', route);
route();
