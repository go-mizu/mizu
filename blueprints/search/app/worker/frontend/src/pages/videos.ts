import { Router } from '../lib/router';
import { api } from '../api';
import type { VideoResult } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;

export function renderVideosPage(query: string): string {
  return `
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1200px]">
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
        <div class="max-w-[1200px] pl-[170px]">
          ${renderTabs({ query, active: 'videos' })}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="videos-content" class="max-w-[1200px] mx-auto px-4 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `;
}

export function initVideosPage(router: Router, query: string): void {
  initSearchBox((q) => {
    router.navigate(`/videos?q=${encodeURIComponent(q)}`);
  });

  initTabs();

  if (query) {
    addRecentSearch(query);
  }

  fetchAndRenderVideos(query);
}

async function fetchAndRenderVideos(query: string): Promise<void> {
  const content = document.getElementById('videos-content');
  if (!content || !query) return;

  try {
    const response = await api.searchVideos(query);
    const results = response.results as VideoResult[];

    if (results.length === 0) {
      content.innerHTML = `
        <div class="py-8 text-secondary">No video results found for "<strong>${escapeHtml(query)}</strong>"</div>
      `;
      return;
    }

    content.innerHTML = `
      <div class="text-xs text-tertiary mb-4">
        About ${response.total_results.toLocaleString()} video results (${(response.search_time_ms / 1000).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        ${results.map((video) => renderVideoCard(video)).join('')}
      </div>
    `;
  } catch (err) {
    content.innerHTML = `
      <div class="py-8">
        <p class="text-red text-sm">Failed to load video results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${escapeHtml(String(err))}</p>
      </div>
    `;
  }
}

function renderVideoCard(video: VideoResult): string {
  const thumbnailUrl = video.thumbnail?.url || '';
  const views = video.views ? formatViews(video.views) : '';
  const published = video.published ? formatDate(video.published) : '';
  const meta = [video.channel, views, published].filter(Boolean).join(' \u00B7 ');

  return `
    <div class="video-card">
      <a href="${escapeAttr(video.url)}" target="_blank" rel="noopener" class="block">
        <div class="video-thumb">
          ${
            thumbnailUrl
              ? `<img src="${escapeAttr(thumbnailUrl)}" alt="${escapeAttr(video.title)}" loading="lazy" onerror="this.style.display='none'" />`
              : `<div class="w-full h-full flex items-center justify-center bg-surface">
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>
                </div>`
          }
          ${video.duration ? `<span class="video-duration">${escapeHtml(video.duration)}</span>` : ''}
        </div>
      </a>
      <div class="video-info">
        <div class="video-title">
          <a href="${escapeAttr(video.url)}" target="_blank" rel="noopener">${escapeHtml(video.title)}</a>
        </div>
        <div class="video-meta">${escapeHtml(meta)}</div>
        ${video.platform ? `<div class="text-xs text-light mt-1">${escapeHtml(video.platform)}</div>` : ''}
      </div>
    </div>
  `;
}

function formatViews(views: number): string {
  if (views >= 1_000_000) return `${(views / 1_000_000).toFixed(1)}M views`;
  if (views >= 1_000) return `${(views / 1_000).toFixed(1)}K views`;
  return `${views} views`;
}

function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - d.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) return 'Today';
    if (days === 1) return '1 day ago';
    if (days < 7) return `${days} days ago`;
    if (days < 30) return `${Math.floor(days / 7)} weeks ago`;
    if (days < 365) return `${Math.floor(days / 30)} months ago`;
    return `${Math.floor(days / 365)} years ago`;
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
