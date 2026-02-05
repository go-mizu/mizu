import { Router } from '../lib/router';
import { api } from '../api';
import type { SearchResult } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;
const ICON_PLAY = `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>`;
const ICON_MUSIC = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>`;

export function renderMusicPage(query: string): string {
  return `
    <div class="min-h-screen flex flex-col">
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${renderSearchBox({ size: 'sm', initialValue: query })}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors">
            ${ICON_SETTINGS}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${renderTabs({ query, active: 'music' })}
        </div>
      </header>
      <main class="flex-1">
        <div id="music-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `;
}

export function initMusicPage(router: Router, query: string): void {
  initSearchBox((q) => router.navigate(`/music?q=${encodeURIComponent(q)}`));
  initTabs();
  if (query) addRecentSearch(query);
  fetchAndRenderMusic(query);
}

async function fetchAndRenderMusic(query: string): Promise<void> {
  const content = document.getElementById('music-content');
  if (!content || !query) return;

  try {
    const response = await api.searchMusic(query);
    const results = response.results;

    if (results.length === 0) {
      content.innerHTML = `<div class="py-8 text-secondary">No music results found for "${escapeHtml(query)}"</div>`;
      return;
    }

    content.innerHTML = `
      <div class="text-xs text-tertiary mb-4">
        About ${response.total_results.toLocaleString()} results (${(response.search_time_ms / 1000).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        ${results.map(renderMusicCard).join('')}
      </div>
    `;
  } catch (err) {
    content.innerHTML = `<div class="py-8 text-red text-sm">Failed to load results. ${escapeHtml(String(err))}</div>`;
  }
}

function renderMusicCard(result: SearchResult): string {
  const source = (result as any).metadata?.source || extractDomain(result.url);
  const artist = (result as any).metadata?.artist || '';
  const album = (result as any).metadata?.album || '';
  const duration = (result as any).metadata?.duration || '';
  const thumbnail = result.thumbnail?.url || '';
  const isGenius = source.toLowerCase().includes('genius');

  return `
    <article class="music-card bg-white border border-border rounded-xl overflow-hidden hover:shadow-md transition-shadow">
      <a href="${escapeAttr(result.url)}" target="_blank" rel="noopener" class="block">
        <div class="relative aspect-square bg-surface">
          ${thumbnail
            ? `<img src="${escapeAttr(thumbnail)}" alt="" class="w-full h-full object-cover" loading="lazy" onerror="this.style.display='none'" />`
            : `<div class="w-full h-full flex items-center justify-center text-border">${ICON_MUSIC}</div>`
          }
          <div class="absolute inset-0 bg-black/40 opacity-0 hover:opacity-100 transition-opacity flex items-center justify-center">
            <span class="w-12 h-12 rounded-full bg-white flex items-center justify-center text-primary">${ICON_PLAY}</span>
          </div>
        </div>
        <div class="p-3">
          <span class="text-xs px-2 py-0.5 rounded-full font-medium ${getSourceColor(source)}">${escapeHtml(source)}</span>
          <h3 class="text-sm font-medium text-primary mt-2 line-clamp-2">${escapeHtml(result.title)}</h3>
          ${artist ? `<p class="text-xs text-secondary mt-1">${escapeHtml(artist)}</p>` : ''}
          ${album ? `<p class="text-xs text-tertiary">${escapeHtml(album)}</p>` : ''}
          ${duration ? `<p class="text-xs text-tertiary mt-1">${escapeHtml(duration)}</p>` : ''}
          ${isGenius && result.snippet ? `<p class="text-xs text-snippet mt-2 line-clamp-2 italic">"${escapeHtml(result.snippet.slice(0, 100))}..."</p>` : ''}
        </div>
      </a>
    </article>
  `;
}

function getSourceColor(source: string): string {
  if (source.toLowerCase().includes('soundcloud')) return 'bg-orange-500 text-white';
  if (source.toLowerCase().includes('bandcamp')) return 'bg-teal-500 text-white';
  if (source.toLowerCase().includes('genius')) return 'bg-yellow-400 text-black';
  return 'bg-surface text-secondary';
}

function extractDomain(url: string): string {
  try { return new URL(url).hostname.replace('www.', ''); } catch { return ''; }
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
