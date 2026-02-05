import { Router } from '../lib/router';
import { api } from '../api';
import type { SearchResult } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;
const ICON_UPVOTE = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m18 15-6-6-6 6"/></svg>`;
const ICON_COMMENT = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>`;

export function renderSocialPage(query: string): string {
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
          ${renderTabs({ query, active: 'social' })}
        </div>
      </header>
      <main class="flex-1">
        <div id="social-content" class="search-content-area">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `;
}

export function initSocialPage(router: Router, query: string): void {
  initSearchBox((q) => router.navigate(`/social?q=${encodeURIComponent(q)}`));
  initTabs();
  if (query) addRecentSearch(query);
  fetchAndRenderSocial(query);
}

async function fetchAndRenderSocial(query: string): Promise<void> {
  const content = document.getElementById('social-content');
  if (!content || !query) return;

  try {
    const response = await api.searchSocial(query);
    const results = response.results;

    if (results.length === 0) {
      content.innerHTML = `<div class="py-8 text-secondary">No social results found for "${escapeHtml(query)}"</div>`;
      return;
    }

    content.innerHTML = `
      <div class="text-xs text-tertiary mb-4">
        About ${response.total_results.toLocaleString()} results (${(response.search_time_ms / 1000).toFixed(2)} seconds)
      </div>
      <div class="w-full space-y-4">
        ${results.map(renderSocialCard).join('')}
      </div>
    `;
  } catch (err) {
    content.innerHTML = `<div class="py-8 text-red text-sm">Failed to load results. ${escapeHtml(String(err))}</div>`;
  }
}

function renderSocialCard(result: SearchResult): string {
  const metadata = (result as any).metadata || {};
  const source = metadata.source || extractDomain(result.url);
  const upvotes = metadata.upvotes || metadata.score || 0;
  const comments = metadata.comments || 0;
  const author = metadata.author || '';
  const subreddit = metadata.subreddit || '';
  const published = metadata.published || '';
  const thumbnailUrl = result.thumbnail?.url || '';
  const content = result.snippet || '';

  return `
    <article class="social-card bg-white border border-border rounded-xl p-4 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3">
        <!-- Upvote column -->
        <div class="flex flex-col items-center text-tertiary text-sm">
          ${ICON_UPVOTE}
          <span class="font-medium ${upvotes > 0 ? 'text-orange-500' : ''}">${formatNumber(upvotes)}</span>
        </div>
        <!-- Content -->
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 mb-1 flex-wrap">
            <span class="text-xs px-2 py-0.5 rounded-full font-medium ${getSourceColor(source)}">${escapeHtml(source)}</span>
            ${subreddit ? `<span class="text-xs text-blue">r/${escapeHtml(subreddit)}</span>` : ''}
            ${author ? `<span class="text-xs text-tertiary">by ${escapeHtml(author)}</span>` : ''}
            ${published ? `<span class="text-xs text-tertiary">${formatDate(published)}</span>` : ''}
          </div>
          <h3 class="text-base font-medium text-primary mb-1">
            <a href="${escapeAttr(result.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${escapeHtml(result.title)}</a>
          </h3>
          ${content ? `<p class="text-sm text-snippet line-clamp-3 mb-2">${escapeHtml(content)}</p>` : ''}
          <div class="flex items-center gap-4 text-xs text-tertiary">
            <span class="flex items-center gap-1">${ICON_COMMENT} ${formatNumber(comments)} comments</span>
          </div>
        </div>
        <!-- Thumbnail if available -->
        ${thumbnailUrl ? `
          <img src="${escapeAttr(thumbnailUrl)}" alt="" class="w-20 h-20 rounded-lg object-cover flex-shrink-0" loading="lazy" onerror="this.style.display='none'" />
        ` : ''}
      </div>
    </article>
  `;
}

function getSourceColor(source: string): string {
  const s = source.toLowerCase();
  if (s.includes('reddit')) return 'bg-orange-500 text-white';
  if (s.includes('hacker') || s.includes('hn')) return 'bg-orange-600 text-white';
  if (s.includes('mastodon')) return 'bg-purple-500 text-white';
  if (s.includes('lemmy')) return 'bg-green-500 text-white';
  return 'bg-surface text-secondary';
}

function formatNumber(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K';
  return String(n);
}

function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - d.getTime();
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    if (hours < 1) return 'just now';
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    if (days < 30) return `${Math.floor(days / 7)}w ago`;
    return `${Math.floor(days / 30)}mo ago`;
  } catch { return ''; }
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
