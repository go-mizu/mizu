import { Router } from '../lib/router';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { fetchTrending } from '../api';

const BANG_SHORTCUTS = [
  { trigger: '!g', label: 'Google', color: '#4285F4' },
  { trigger: '!yt', label: 'YouTube', color: '#EA4335' },
  { trigger: '!gh', label: 'GitHub', color: '#24292e' },
  { trigger: '!w', label: 'Wikipedia', color: '#636466' },
  { trigger: '!r', label: 'Reddit', color: '#FF5700' },
];

const INSTANT_BUTTONS = [
  { label: 'Calculator', icon: calcIcon(), query: '2+2', color: 'bg-blue/10 text-blue' },
  { label: 'Conversion', icon: convertIcon(), query: '10 miles in km', color: 'bg-green/10 text-green' },
  { label: 'Currency', icon: currencyIcon(), query: '100 USD to EUR', color: 'bg-yellow/10 text-yellow' },
  { label: 'Weather', icon: weatherIcon(), query: 'weather New York', color: 'bg-blue/10 text-blue' },
  { label: 'Time', icon: timeIcon(), query: 'time in Tokyo', color: 'bg-green/10 text-green' },
  { label: 'Define', icon: defineIcon(), query: 'define serendipity', color: 'bg-red/10 text-red' },
];

export function renderHomePage(): string {
  return `
    <div class="min-h-screen flex flex-col">
      <div class="flex-1 flex flex-col items-center justify-center px-4 -mt-20">
        <!-- Logo -->
        <div class="mb-8 text-center">
          <h1 class="text-6xl font-semibold mb-2 select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </h1>
          <p class="text-secondary text-lg">Privacy-first search</p>
        </div>

        <!-- Search Box -->
        <div class="w-full max-w-2xl mb-6">
          ${renderSearchBox({ size: 'lg', autofocus: true })}
        </div>

        <!-- Search Buttons -->
        <div class="flex gap-3 mb-8">
          <button id="home-search-btn" class="px-5 py-2 bg-surface hover:bg-surface-hover border border-border rounded text-sm text-primary cursor-pointer">
            Mizu Search
          </button>
          <button id="home-lucky-btn" class="px-5 py-2 bg-surface hover:bg-surface-hover border border-border rounded text-sm text-primary cursor-pointer">
            I'm Feeling Lucky
          </button>
        </div>

        <!-- Trending Searches -->
        <div class="trending-section mb-8 hidden" id="trending-container">
          <p class="text-center text-xs text-light mb-3 uppercase tracking-wider">Trending Searches</p>
          <div class="trending-chips flex flex-wrap justify-center gap-2" id="trending-chips">
            <!-- Populated by JS -->
          </div>
        </div>

        <!-- Bang Shortcuts -->
        <div class="flex flex-wrap justify-center gap-2 mb-8">
          ${BANG_SHORTCUTS.map(
            (b) => `
            <button class="bang-shortcut px-3 py-1.5 rounded-full text-xs font-medium border border-border hover:shadow-sm transition-shadow cursor-pointer"
                    data-bang="${b.trigger}"
                    style="color: ${b.color}; border-color: ${b.color}20;">
              <span class="font-semibold">${escapeHtml(b.trigger)}</span>
              <span class="text-tertiary ml-1">${escapeHtml(b.label)}</span>
            </button>
          `
          ).join('')}
        </div>

        <!-- Instant Answers Showcase -->
        <div class="mb-8">
          <p class="text-center text-xs text-light mb-3 uppercase tracking-wider">Instant Answers</p>
          <div class="flex flex-wrap justify-center gap-2">
            ${INSTANT_BUTTONS.map(
              (btn) => `
              <button class="instant-showcase-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium ${btn.color} hover:opacity-80 transition-opacity cursor-pointer"
                      data-query="${escapeAttr(btn.query)}">
                ${btn.icon}
                <span>${escapeHtml(btn.label)}</span>
              </button>
            `
            ).join('')}
          </div>
        </div>

        <!-- Category Links -->
        <div class="flex gap-6 text-sm">
          <a href="/images" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${imageIcon()}
            Images
          </a>
          <a href="/news" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${newsIcon()}
            News
          </a>
        </div>
      </div>

      <!-- Footer -->
      <footer class="py-4 text-center">
        <div class="text-xs text-light space-x-4">
          <span>Use <strong>!bangs</strong> to search other sites directly</span>
          <span>&middot;</span>
          <a href="/settings" data-link class="hover:text-secondary">Settings</a>
          <span>&middot;</span>
          <a href="/history" data-link class="hover:text-secondary">History</a>
        </div>
      </footer>
    </div>
  `;
}

export function initHomePage(router: Router): void {
  initSearchBox((query) => {
    router.navigate(`/search?q=${encodeURIComponent(query)}`);
  });

  // Search buttons
  const searchBtn = document.getElementById('home-search-btn');
  searchBtn?.addEventListener('click', () => {
    const input = document.getElementById('search-input') as HTMLInputElement;
    const q = input?.value?.trim();
    if (q) {
      router.navigate(`/search?q=${encodeURIComponent(q)}`);
    }
  });

  const luckyBtn = document.getElementById('home-lucky-btn');
  luckyBtn?.addEventListener('click', () => {
    const input = document.getElementById('search-input') as HTMLInputElement;
    const q = input?.value?.trim();
    if (q) {
      router.navigate(`/search?q=${encodeURIComponent(q)}&lucky=1`);
    }
  });

  // Bang shortcuts
  document.querySelectorAll('.bang-shortcut').forEach((btn) => {
    btn.addEventListener('click', () => {
      const bang = (btn as HTMLElement).dataset.bang || '';
      const input = document.getElementById('search-input') as HTMLInputElement;
      if (input) {
        input.value = bang + ' ';
        input.focus();
      }
    });
  });

  // Instant answer showcase buttons
  document.querySelectorAll('.instant-showcase-btn').forEach((btn) => {
    btn.addEventListener('click', () => {
      const query = (btn as HTMLElement).dataset.query || '';
      if (query) {
        router.navigate(`/search?q=${encodeURIComponent(query)}`);
      }
    });
  });

  // Load and render trending searches
  async function loadTrending() {
    const trending = await fetchTrending();
    const container = document.getElementById('trending-container');
    const chipsContainer = document.getElementById('trending-chips');
    if (container && chipsContainer && trending.length > 0) {
      chipsContainer.innerHTML = trending.slice(0, 10).map(term => `
        <a href="/search?q=${encodeURIComponent(term)}" data-link
           class="trending-chip px-3 py-1.5 bg-secondary hover:bg-accent rounded-full text-xs font-medium text-secondary-foreground hover:text-accent-foreground transition-colors cursor-pointer no-underline">
          ${trendingIcon()}
          ${escapeHtml(term)}
        </a>
      `).join('');
      container.classList.remove('hidden');
    }
  }
  loadTrending();
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function calcIcon(): string {
  return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/></svg>`;
}

function convertIcon(): string {
  return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>`;
}

function currencyIcon(): string {
  return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>`;
}

function weatherIcon(): string {
  return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/></svg>`;
}

function timeIcon(): string {
  return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>`;
}

function defineIcon(): string {
  return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>`;
}

function imageIcon(): string {
  return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>`;
}

function newsIcon(): string {
  return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/></svg>`;
}

function trendingIcon(): string {
  return `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="inline-block mr-1"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>`;
}
