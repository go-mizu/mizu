import { Router } from './lib/router';
import { renderHomePage, initHomePage } from './pages/home';
import { renderSearchPage, initSearchPage } from './pages/search';
import { renderImagesPage, initImagesPage } from './pages/images';
import { renderVideosPage, initVideosPage } from './pages/videos';
import { renderNewsPage, initNewsPage } from './pages/news';
import { renderNewsHomePage, initNewsHomePage } from './pages/news-home';
import { renderSciencePage, initSciencePage } from './pages/science';
import { renderCodePage, initCodePage } from './pages/code';
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
  const timeRange = query.time_range || '';
  app.innerHTML = renderSearchPage(q, timeRange);
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
