import { Router } from '../lib/router';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { fetchTrending } from '../api';

// Icons
const ICON_TRENDING = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>`;
const ICON_IMAGE = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>`;
const ICON_NEWS = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/></svg>`;
const ICON_CALC = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/></svg>`;
const ICON_CONVERT = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>`;
const ICON_CURRENCY = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>`;
const ICON_WEATHER = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/></svg>`;
const ICON_TIME = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>`;
const ICON_DEFINE = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>`;

const BANG_SHORTCUTS = [
  { trigger: '!g', label: 'Google' },
  { trigger: '!yt', label: 'YouTube' },
  { trigger: '!gh', label: 'GitHub' },
  { trigger: '!w', label: 'Wikipedia' },
  { trigger: '!r', label: 'Reddit' },
];

const INSTANT_BUTTONS = [
  { label: 'Calculator', icon: ICON_CALC, query: '2+2', colorClass: 'blue' },
  { label: 'Conversion', icon: ICON_CONVERT, query: '10 miles in km', colorClass: 'green' },
  { label: 'Currency', icon: ICON_CURRENCY, query: '100 USD to EUR', colorClass: 'yellow' },
  { label: 'Weather', icon: ICON_WEATHER, query: 'weather New York', colorClass: 'blue' },
  { label: 'Time', icon: ICON_TIME, query: 'time in Tokyo', colorClass: 'green' },
  { label: 'Define', icon: ICON_DEFINE, query: 'define serendipity', colorClass: 'red' },
];

export function renderHomePage(): string {
  return `
    <div class="home-container">
      <main class="home-main">
        <!-- Logo -->
        <h1 class="home-logo">
          <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
        </h1>
        <p class="home-tagline">Privacy-first search</p>

        <!-- Search Box -->
        <div class="home-search">
          ${renderSearchBox({ size: 'lg', autofocus: true })}
        </div>

        <!-- Search Buttons -->
        <div class="home-buttons">
          <button id="home-search-btn" class="btn btn-secondary">
            Mizu Search
          </button>
          <button id="home-lucky-btn" class="btn btn-ghost">
            I'm Feeling Lucky
          </button>
        </div>

        <!-- Trending Searches -->
        <div class="home-trending hidden" id="trending-container">
          <p class="home-trending-title">Trending</p>
          <div class="home-trending-chips" id="trending-chips"></div>
        </div>

        <!-- Bang Shortcuts -->
        <div class="home-bangs">
          ${BANG_SHORTCUTS.map(b => `
            <button class="bang-chip" data-bang="${b.trigger}">
              <span class="bang-trigger">${escapeHtml(b.trigger)}</span>
              <span class="bang-label">${escapeHtml(b.label)}</span>
            </button>
          `).join('')}
        </div>

        <!-- Instant Answers -->
        <div class="home-instant">
          <p class="home-instant-title">Instant Answers</p>
          <div class="home-instant-buttons">
            ${INSTANT_BUTTONS.map(btn => `
              <button class="instant-btn ${btn.colorClass}" data-query="${escapeAttr(btn.query)}">
                ${btn.icon}
                <span>${escapeHtml(btn.label)}</span>
              </button>
            `).join('')}
          </div>
        </div>

        <!-- Quick Links -->
        <div class="home-links">
          <a href="/images" data-link>
            ${ICON_IMAGE}
            Images
          </a>
          <a href="/news" data-link>
            ${ICON_NEWS}
            News
          </a>
        </div>
      </main>

      <!-- Footer -->
      <footer class="home-footer">
        <div class="home-footer-links">
          <span>Use <strong>!bangs</strong> to search other sites</span>
          <span>&middot;</span>
          <a href="/settings" data-link>Settings</a>
          <span>&middot;</span>
          <a href="/history" data-link>History</a>
        </div>
      </footer>
    </div>
  `;
}

export function initHomePage(router: Router): void {
  initSearchBox((query) => {
    router.navigate(`/search?q=${encodeURIComponent(query)}`);
  });

  // Search button
  const searchBtn = document.getElementById('home-search-btn');
  searchBtn?.addEventListener('click', () => {
    const input = document.getElementById('search-input') as HTMLInputElement;
    const q = input?.value?.trim();
    if (q) {
      router.navigate(`/search?q=${encodeURIComponent(q)}`);
    }
  });

  // Lucky button
  const luckyBtn = document.getElementById('home-lucky-btn');
  luckyBtn?.addEventListener('click', () => {
    const input = document.getElementById('search-input') as HTMLInputElement;
    const q = input?.value?.trim();
    if (q) {
      router.navigate(`/search?q=${encodeURIComponent(q)}&lucky=1`);
    }
  });

  // Bang shortcuts
  document.querySelectorAll('.bang-chip').forEach((btn) => {
    btn.addEventListener('click', () => {
      const bang = (btn as HTMLElement).dataset.bang || '';
      const input = document.getElementById('search-input') as HTMLInputElement;
      if (input) {
        input.value = bang + ' ';
        input.focus();
      }
    });
  });

  // Instant answer buttons
  document.querySelectorAll('.instant-btn').forEach((btn) => {
    btn.addEventListener('click', () => {
      const query = (btn as HTMLElement).dataset.query || '';
      if (query) {
        router.navigate(`/search?q=${encodeURIComponent(query)}`);
      }
    });
  });

  // Load trending searches
  loadTrending();
}

async function loadTrending(): Promise<void> {
  try {
    const trending = await fetchTrending();
    const container = document.getElementById('trending-container');
    const chipsContainer = document.getElementById('trending-chips');

    if (container && chipsContainer && trending.length > 0) {
      chipsContainer.innerHTML = trending.slice(0, 8).map(term => `
        <a href="/search?q=${encodeURIComponent(term)}" data-link class="trending-chip">
          ${ICON_TRENDING}
          ${escapeHtml(term)}
        </a>
      `).join('');
      container.classList.remove('hidden');
    }
  } catch {
    // Silently fail - trending is optional
  }
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
