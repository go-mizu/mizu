import { Router } from './lib/router';
import { renderHomePage, initHomePage } from './pages/home';
import { renderSearchPage, initSearchPage } from './pages/search';
import { renderImagesPage, initImagesPage } from './pages/images';
import { renderVideosPage, initVideosPage } from './pages/videos';
import { renderNewsPage, initNewsPage } from './pages/news';
import { renderNewsHomePage, initNewsHomePage } from './pages/news-home';
import { renderSciencePage, initSciencePage } from './pages/science';
import { renderCodePage, initCodePage } from './pages/code';
import { renderMusicPage, initMusicPage } from './pages/music';
import { renderSocialPage, initSocialPage } from './pages/social';
import { renderMapsPage, initMapsPage } from './pages/maps';
import { renderSettingsPage, initSettingsPage } from './pages/settings';
import { renderHistoryPage, initHistoryPage } from './pages/history';

const app = document.getElementById('app');
if (!app) throw new Error('App container not found');

const router = new Router();

// Home page
router.addRoute('', (params, query) => {
  app.innerHTML = renderHomePage();
  initHomePage(router);
});

// Search results
router.addRoute('search', (params, query) => {
  const q = query.q || '';
  const filters = {
    timeRange: query.time_range || '',
    region: query.region || '',
    verbatim: query.verbatim === '1',
    site: query.site || '',
  };
  app.innerHTML = renderSearchPage(q, filters);
  initSearchPage(router, q, query);
});

// Images
router.addRoute('images', (params, query) => {
  const q = query.q || '';
  app.innerHTML = renderImagesPage(q);
  initImagesPage(router, q, query);
});

// Videos
router.addRoute('videos', (params, query) => {
  const q = query.q || '';
  app.innerHTML = renderVideosPage(q);
  initVideosPage(router, q);
});

// News Search
router.addRoute('news', (params, query) => {
  const q = query.q || '';
  app.innerHTML = renderNewsPage(q);
  initNewsPage(router, q);
});

// News Home (Google News clone)
router.addRoute('news-home', (params, query) => {
  app.innerHTML = renderNewsHomePage();
  initNewsHomePage(router, query);
});

// Science/Academic Search
router.addRoute('science', (params, query) => {
  const q = query.q || '';
  app.innerHTML = renderSciencePage(q);
  initSciencePage(router, q);
});

// Code/IT Search
router.addRoute('code', (params, query) => {
  const q = query.q || '';
  app.innerHTML = renderCodePage(q);
  initCodePage(router, q);
});

// Music Search
router.addRoute('music', (params, query) => {
  const q = query.q || '';
  app.innerHTML = renderMusicPage(q);
  initMusicPage(router, q);
});

// Social Search
router.addRoute('social', (params, query) => {
  const q = query.q || '';
  app.innerHTML = renderSocialPage(q);
  initSocialPage(router, q);
});

// Maps Search
router.addRoute('maps', (params, query) => {
  const q = query.q || '';
  app.innerHTML = renderMapsPage(q);
  initMapsPage(router, q);
});

// Settings
router.addRoute('settings', (params, query) => {
  app.innerHTML = renderSettingsPage();
  initSettingsPage(router);
});

// History
router.addRoute('history', (params, query) => {
  app.innerHTML = renderHistoryPage();
  initHistoryPage(router);
});

// 404 Not Found
router.setNotFound((params, query) => {
  app.innerHTML = `
    <div class="min-h-screen flex flex-col items-center justify-center px-4">
      <h1 class="text-4xl font-semibold mb-4">
        <span style="color: #4285F4">4</span><span style="color: #EA4335">0</span><span style="color: #FBBC05">4</span>
      </h1>
      <p class="text-secondary mb-6">Page not found</p>
      <a href="/" data-link class="text-blue hover:underline">Go home</a>
    </div>
  `;
});

// Listen for custom navigate events
window.addEventListener('router:navigate', (e: Event) => {
  const customEvent = e as CustomEvent<{ path: string }>;
  router.navigate(customEvent.detail.path);
});

// Start the router
router.start();

// ========== Global Keyboard Shortcuts ==========

function initKeyboardShortcuts(): void {
  document.addEventListener('keydown', (e) => {
    // Skip if user is typing in an input or textarea
    const target = e.target as HTMLElement;
    const isInput = target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable;

    // "/" to focus search - works even in inputs if they're not the search box
    if (e.key === '/' && !isInput) {
      e.preventDefault();
      const searchInput = document.getElementById('search-input') as HTMLInputElement;
      if (searchInput) {
        searchInput.focus();
        searchInput.select();
      }
    }

    // "Escape" to close modals and overlays
    if (e.key === 'Escape') {
      // Close any visible modals
      const modals = document.querySelectorAll('.modal:not(.hidden), .preview-panel:not(.hidden), .lightbox:not(.hidden)');
      modals.forEach(modal => {
        modal.classList.add('hidden');
      });

      // Close dropdowns
      const dropdowns = document.querySelectorAll('.autocomplete-dropdown:not(.hidden), .filter-dropdown:not(.hidden), .filter-pill-dropdown:not(.hidden), .more-tabs-dropdown:not(.hidden), .time-filter-dropdown:not(.hidden), .search-tool-menu:not(.hidden)');
      dropdowns.forEach(dropdown => {
        dropdown.classList.add('hidden');
      });

      // Restore body scroll if it was hidden
      document.body.style.overflow = '';

      // Blur search input if focused
      if (document.activeElement?.id === 'search-input') {
        (document.activeElement as HTMLElement).blur();
      }
    }

    // "?" to show keyboard shortcuts help (optional)
    if (e.key === '?' && !isInput) {
      showKeyboardShortcutsHelp();
    }
  });
}

function showKeyboardShortcutsHelp(): void {
  // Check if help is already shown
  let helpModal = document.getElementById('keyboard-shortcuts-help');
  if (helpModal) {
    helpModal.classList.toggle('hidden');
    return;
  }

  // Create help modal
  helpModal = document.createElement('div');
  helpModal.id = 'keyboard-shortcuts-help';
  helpModal.className = 'modal';
  helpModal.innerHTML = `
    <div class="modal-content" style="max-width: 400px;">
      <div class="modal-header">
        <h2>Keyboard Shortcuts</h2>
        <button class="modal-close" id="shortcuts-close">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
        </button>
      </div>
      <div class="modal-body">
        <div class="shortcuts-list">
          <div class="shortcut-item">
            <kbd>/</kbd>
            <span>Focus search box</span>
          </div>
          <div class="shortcut-item">
            <kbd>Escape</kbd>
            <span>Close modal / unfocus</span>
          </div>
          <div class="shortcut-item">
            <kbd>?</kbd>
            <span>Show this help</span>
          </div>
          <div class="shortcut-item">
            <kbd>&#8593;</kbd> <kbd>&#8595;</kbd>
            <span>Navigate suggestions</span>
          </div>
          <div class="shortcut-item">
            <kbd>Enter</kbd>
            <span>Select / submit</span>
          </div>
        </div>
      </div>
    </div>
  `;

  document.body.appendChild(helpModal);

  // Close handlers
  helpModal.addEventListener('click', (e) => {
    if (e.target === helpModal) {
      helpModal.classList.add('hidden');
    }
  });

  document.getElementById('shortcuts-close')?.addEventListener('click', () => {
    helpModal!.classList.add('hidden');
  });
}

// Initialize shortcuts when app starts
initKeyboardShortcuts();

// ========== Service Worker Registration (PWA Support) ==========

if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js')
      .then((reg) => {
        console.log('Service Worker registered:', reg.scope);
      })
      .catch((err) => {
        console.log('Service Worker registration failed:', err);
      });
  });
}
