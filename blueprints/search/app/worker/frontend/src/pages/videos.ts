import { Router } from '../lib/router';
import { api } from '../api';
import type { VideoResult } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;
const ICON_FILTER = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>`;
const ICON_CHEVRON_DOWN = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>`;

interface VideoFilters {
  duration?: string;
  quality?: string;
  time?: string;
  source?: string;
  sort?: string;
}

let currentQuery = '';
let currentFilters: VideoFilters = {};
let filtersVisible = false;

export function renderVideosPage(query: string): string {
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
        <div class="search-tabs-row flex items-center gap-1">
          ${renderTabs({ query, active: 'videos' })}
          <button id="tools-btn" class="filter-btn ml-4">
            ${ICON_FILTER}
            <span class="hidden sm:inline">Tools</span>
            ${ICON_CHEVRON_DOWN}
          </button>
        </div>
        <!-- Filter toolbar (hidden by default) -->
        <div id="filter-toolbar" class="filter-toolbar hidden">
          ${renderFilterToolbar()}
        </div>
      </header>

      <!-- Content - Full width -->
      <main class="flex-1">
        <div id="videos-content" class="px-2 sm:px-4 lg:px-6 xl:px-8 py-4">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `;
}

function renderFilterToolbar(): string {
  const filters = [
    { id: 'duration', label: 'Duration', options: ['any', 'short', 'medium', 'long'] },
    { id: 'time', label: 'Upload date', options: ['any', 'hour', 'day', 'week', 'month', 'year'] },
    { id: 'quality', label: 'Quality', options: ['any', 'hd', '4k'] },
    { id: 'sort', label: 'Sort by', options: ['relevance', 'date', 'views'] },
  ];

  return `
    <div class="filter-chips">
      ${filters.map(f => `
        <div class="filter-chip-wrapper">
          <button class="filter-chip" data-filter="${f.id}" data-value="any">
            <span class="filter-chip-label">${f.label}</span>
            ${ICON_CHEVRON_DOWN}
          </button>
          <div class="filter-dropdown hidden" data-dropdown="${f.id}">
            ${f.options.map(opt => `
              <button class="filter-option${opt === 'any' || opt === 'relevance' ? ' active' : ''}" data-value="${opt}">
                ${formatFilterOption(f.id, opt)}
              </button>
            `).join('')}
          </div>
        </div>
      `).join('')}
      <button id="clear-filters" class="clear-filters-btn hidden">Clear</button>
    </div>
  `;
}

function formatFilterOption(filterId: string, option: string): string {
  const labels: Record<string, Record<string, string>> = {
    duration: {
      any: 'Any duration',
      short: 'Under 4 min',
      medium: '4-20 min',
      long: 'Over 20 min',
    },
    time: {
      any: 'Any time',
      hour: 'Last hour',
      day: 'Today',
      week: 'This week',
      month: 'This month',
      year: 'This year',
    },
    quality: {
      any: 'Any quality',
      hd: 'HD',
      '4k': '4K',
    },
    sort: {
      relevance: 'Relevance',
      date: 'Upload date',
      views: 'View count',
    },
  };
  return labels[filterId]?.[option] || option.charAt(0).toUpperCase() + option.slice(1);
}

export function initVideosPage(router: Router, query: string): void {
  currentQuery = query;
  currentFilters = {};
  filtersVisible = false;

  initSearchBox((q) => {
    router.navigate(`/videos?q=${encodeURIComponent(q)}`);
  });

  initTabs();
  initToolsButton();
  initFilters(router);

  if (query) {
    addRecentSearch(query);
  }

  fetchAndRenderVideos(query, currentFilters);
}

function initToolsButton(): void {
  const btn = document.getElementById('tools-btn');
  const toolbar = document.getElementById('filter-toolbar');

  if (!btn || !toolbar) return;

  btn.addEventListener('click', () => {
    filtersVisible = !filtersVisible;
    toolbar.classList.toggle('hidden', !filtersVisible);
    btn.classList.toggle('active', filtersVisible);
  });
}

function initFilters(router: Router): void {
  const toolbar = document.getElementById('filter-toolbar');
  if (!toolbar) return;

  toolbar.querySelectorAll('.filter-chip').forEach(chip => {
    chip.addEventListener('click', (e) => {
      e.stopPropagation();
      const filterId = (chip as HTMLElement).dataset.filter;
      const dropdown = toolbar.querySelector(`[data-dropdown="${filterId}"]`);

      toolbar.querySelectorAll('.filter-dropdown').forEach(d => {
        if (d !== dropdown) d.classList.add('hidden');
      });

      dropdown?.classList.toggle('hidden');
    });
  });

  toolbar.querySelectorAll('.filter-option').forEach(option => {
    option.addEventListener('click', () => {
      const dropdown = option.closest('.filter-dropdown') as HTMLElement;
      const filterId = dropdown?.dataset.dropdown;
      const value = (option as HTMLElement).dataset.value;
      const chip = toolbar.querySelector(`[data-filter="${filterId}"]`) as HTMLElement;

      if (!filterId || !value || !chip) return;

      dropdown.querySelectorAll('.filter-option').forEach(o => o.classList.remove('active'));
      option.classList.add('active');

      const isDefault = value === 'any' || (filterId === 'sort' && value === 'relevance');

      if (isDefault) {
        delete (currentFilters as Record<string, string>)[filterId];
        chip.classList.remove('has-value');
        chip.querySelector('.filter-chip-label')!.textContent = formatFilterOption(filterId, isDefault ? 'any' : value).replace(/^Any /, '');
      } else {
        (currentFilters as Record<string, string>)[filterId] = value;
        chip.classList.add('has-value');
        chip.querySelector('.filter-chip-label')!.textContent = formatFilterOption(filterId, value);
      }

      dropdown.classList.add('hidden');
      updateClearButton();

      fetchAndRenderVideos(currentQuery, currentFilters);
    });
  });

  document.addEventListener('click', () => {
    toolbar.querySelectorAll('.filter-dropdown').forEach(d => d.classList.add('hidden'));
  });

  const clearBtn = document.getElementById('clear-filters');
  if (clearBtn) {
    clearBtn.addEventListener('click', () => {
      currentFilters = {};

      toolbar.querySelectorAll('.filter-chip').forEach(chip => {
        const filterId = (chip as HTMLElement).dataset.filter;
        chip.classList.remove('has-value');
        const defaultLabel = filterId === 'sort' ? 'Sort by' : formatFilterOption(filterId!, 'any').replace(/^Any /, '');
        (chip.querySelector('.filter-chip-label') as HTMLElement).textContent = defaultLabel;
      });

      toolbar.querySelectorAll('.filter-dropdown').forEach(dropdown => {
        dropdown.querySelectorAll('.filter-option').forEach((opt, i) => {
          opt.classList.toggle('active', i === 0);
        });
      });

      updateClearButton();
      fetchAndRenderVideos(currentQuery, currentFilters);
    });
  }
}

function updateClearButton(): void {
  const clearBtn = document.getElementById('clear-filters');
  if (!clearBtn) return;
  clearBtn.classList.toggle('hidden', Object.keys(currentFilters).length === 0);
}

async function fetchAndRenderVideos(query: string, filters: VideoFilters): Promise<void> {
  const content = document.getElementById('videos-content');
  if (!content || !query) return;

  content.innerHTML = '<div class="flex items-center justify-center py-16"><div class="spinner"></div></div>';

  try {
    const response = await api.searchVideos(query, {
      page: 1,
      per_page: 24,
      ...filters,
    });
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
      <div class="video-grid">
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
  const meta = [video.channel, views, published].filter(Boolean).join(' · ');

  return `
    <div class="video-card">
      <a href="${escapeAttr(video.url)}" target="_blank" rel="noopener" class="block">
        <div class="video-thumb">
          ${
            thumbnailUrl
              ? `<img src="${escapeAttr(thumbnailUrl)}" alt="${escapeAttr(video.title)}" loading="lazy" onerror="this.style.display='none'; this.nextElementSibling.style.display='flex'" />`
              : ''
          }
          <div class="video-thumb-placeholder" ${thumbnailUrl ? 'style="display:none"' : ''}>
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>
          </div>
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
