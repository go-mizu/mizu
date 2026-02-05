import { Router } from '../lib/router';
import { api } from '../api';
import type { SearchResult } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;
const ICON_STAR = `<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>`;
const ICON_FORK = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="18" r="3"/><circle cx="6" cy="6" r="3"/><circle cx="18" cy="6" r="3"/><path d="M18 9v1a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V9"/><path d="M12 12v3"/></svg>`;
const ICON_DOWNLOAD = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/></svg>`;

export function renderCodePage(query: string): string {
  return `
    <div class="min-h-screen flex flex-col">
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box">
            ${renderSearchBox({ size: 'sm', initialValue: query })}
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${ICON_SETTINGS}
          </a>
        </div>
        <div class="search-tabs-row">
          ${renderTabs({ query, active: 'code' })}
        </div>
      </header>
      <main class="flex-1">
        <div id="code-content" class="search-content-area">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `;
}

export function initCodePage(router: Router, query: string): void {
  initSearchBox((q) => router.navigate(`/code?q=${encodeURIComponent(q)}`));
  initTabs();
  if (query) addRecentSearch(query);
  fetchAndRenderCode(query);
}

async function fetchAndRenderCode(query: string): Promise<void> {
  const content = document.getElementById('code-content');
  if (!content || !query) return;

  try {
    const response = await api.searchCode(query);
    const results = response.results;

    if (results.length === 0) {
      content.innerHTML = `<div class="py-8 text-secondary">No code results found for "${escapeHtml(query)}"</div>`;
      return;
    }

    content.innerHTML = `
      <div class="text-xs text-tertiary mb-4">
        About ${response.total_results.toLocaleString()} results (${(response.search_time_ms / 1000).toFixed(2)} seconds)
      </div>
      <div class="w-full space-y-4">
        ${results.map(renderCodeCard).join('')}
      </div>
    `;
  } catch (err) {
    content.innerHTML = `<div class="py-8 text-red text-sm">Failed to load results. ${escapeHtml(String(err))}</div>`;
  }
}

function renderCodeCard(result: SearchResult): string {
  const source = result.metadata?.source || extractDomain(result.url);
  const stars = result.metadata?.stars;
  const forks = result.metadata?.forks;
  const downloads = result.metadata?.downloads;
  const language = result.metadata?.language || '';
  const votes = result.metadata?.votes;
  const answers = result.metadata?.answers;

  return `
    <article class="code-card bg-white border border-border rounded-xl p-4 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3 mb-2">
        <span class="text-xs px-2 py-0.5 rounded-full font-medium ${getSourceColor(source)}">${escapeHtml(source)}</span>
        ${language ? `<span class="text-xs px-2 py-0.5 bg-surface text-secondary rounded-full">${escapeHtml(language)}</span>` : ''}
      </div>
      <h3 class="text-base font-medium text-primary mb-1">
        <a href="${escapeAttr(result.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${escapeHtml(result.title)}</a>
      </h3>
      <p class="text-sm text-snippet line-clamp-2 mb-3">${result.content || ''}</p>
      <div class="flex items-center gap-4 text-xs text-tertiary">
        ${stars !== undefined ? `<span class="flex items-center gap-1">${ICON_STAR} <span class="text-yellow-500">${formatNumber(stars)}</span></span>` : ''}
        ${forks !== undefined ? `<span class="flex items-center gap-1">${ICON_FORK} ${formatNumber(forks)}</span>` : ''}
        ${downloads !== undefined ? `<span class="flex items-center gap-1">${ICON_DOWNLOAD} ${formatNumber(downloads)}</span>` : ''}
        ${votes !== undefined ? `<span class="flex items-center gap-1">▲ ${formatNumber(votes)}</span>` : ''}
        ${answers !== undefined ? `<span class="flex items-center gap-1">${answers} answers</span>` : ''}
      </div>
    </article>
  `;
}

function getSourceColor(source: string): string {
  if (source.includes('github')) return 'bg-gray-900 text-white';
  if (source.includes('gitlab')) return 'bg-orange-500 text-white';
  if (source.includes('stackoverflow')) return 'bg-orange-400 text-white';
  if (source.includes('npm')) return 'bg-red-500 text-white';
  if (source.includes('pypi')) return 'bg-blue-500 text-white';
  if (source.includes('crates')) return 'bg-orange-600 text-white';
  return 'bg-surface text-secondary';
}

function formatNumber(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K';
  return String(n);
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
