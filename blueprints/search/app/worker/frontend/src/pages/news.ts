import { Router } from '../lib/router';
import { api } from '../api';
import type { NewsResult } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;

export function renderNewsPage(query: string): string {
  return `
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${renderSearchBox({ size: 'sm', initialValue: query })}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${ICON_SETTINGS}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${renderTabs({ query, active: 'news' })}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="news-content" class="px-4 lg:px-8 py-6">
          <div class="max-w-[900px]">
            <div class="flex items-center justify-center py-16">
              <div class="spinner"></div>
            </div>
          </div>
        </div>
      </main>
    </div>
  `;
}

export function initNewsPage(router: Router, query: string): void {
  initSearchBox((q) => {
    router.navigate(`/news?q=${encodeURIComponent(q)}`);
  });

  initTabs();

  if (query) {
    addRecentSearch(query);
  }

  fetchAndRenderNews(query);
}

async function fetchAndRenderNews(query: string): Promise<void> {
  const content = document.getElementById('news-content');
  if (!content || !query) return;

  try {
    const response = await api.searchNews(query);
    const results = response.results as NewsResult[];

    if (results.length === 0) {
      content.innerHTML = `
        <div class="max-w-[900px]">
          <div class="py-8 text-secondary">No news results found for "<strong>${escapeHtml(query)}</strong>"</div>
        </div>
      `;
      return;
    }

    content.innerHTML = `
      <div class="max-w-[900px]">
        <div class="text-xs text-tertiary mb-6">
          About ${response.total_results.toLocaleString()} news results (${(response.search_time_ms / 1000).toFixed(2)} seconds)
        </div>
        <div class="space-y-4">
          ${results.map((article) => renderNewsCard(article)).join('')}
        </div>
      </div>
    `;
  } catch (err) {
    content.innerHTML = `
      <div class="max-w-[900px]">
        <div class="py-8">
          <p class="text-red text-sm">Failed to load news results. Please try again.</p>
          <p class="text-tertiary text-xs mt-2">${escapeHtml(String(err))}</p>
        </div>
      </div>
    `;
  }
}

function renderNewsCard(article: NewsResult): string {
  const thumbnailUrl = article.thumbnail?.url || '';
  const publishedDate = article.published_date ? formatDate(article.published_date) : '';

  return `
    <div class="news-card">
      <div class="flex-1 min-w-0">
        <div class="news-source">
          ${escapeHtml(article.source || article.domain)}
          ${publishedDate ? ` \u00B7 ${escapeHtml(publishedDate)}` : ''}
        </div>
        <div class="news-title">
          <a href="${escapeAttr(article.url)}" target="_blank" rel="noopener">${escapeHtml(article.title)}</a>
        </div>
        <div class="news-snippet">${article.snippet || ''}</div>
      </div>
      ${
        thumbnailUrl
          ? `<img class="news-image" src="${escapeAttr(thumbnailUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`
          : ''
      }
    </div>
  `;
}

function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - d.getTime();
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (hours < 1) return 'Just now';
    if (hours < 24) return `${hours}h ago`;
    if (days === 1) return '1 day ago';
    if (days < 7) return `${days} days ago`;
    if (days < 30) return `${Math.floor(days / 7)} weeks ago`;

    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  } catch {
    return dateStr;
  }
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
