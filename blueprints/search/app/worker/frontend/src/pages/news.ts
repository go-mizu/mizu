import { Router } from '../lib/router';
import { api } from '../api';
import type { NewsResult } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;
const ICON_FILTER = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>`;
const ICON_CHEVRON_DOWN = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>`;

interface NewsFilters {
  time?: string;
  sort?: string;
}

let currentQuery = '';
let currentFilters: NewsFilters = {};
let filtersVisible = false;

export function renderNewsPage(query: string): string {
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
          ${renderTabs({ query, active: 'news' })}
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
        <div id="news-content" class="px-2 sm:px-4 lg:px-6 xl:px-8 py-4">
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
    { id: 'time', label: 'Time', options: ['any', 'hour', 'day', 'week', 'month', 'year'] },
    { id: 'sort', label: 'Sort by', options: ['relevance', 'date'] },
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
    time: {
      any: 'Any time',
      hour: 'Past hour',
      day: 'Past 24 hours',
      week: 'Past week',
      month: 'Past month',
      year: 'Past year',
    },
    sort: {
      relevance: 'Relevance',
      date: 'Date',
    },
  };
  return labels[filterId]?.[option] || option.charAt(0).toUpperCase() + option.slice(1);
}

export function initNewsPage(router: Router, query: string): void {
  currentQuery = query;
  currentFilters = {};
  filtersVisible = false;

  initSearchBox((q) => {
    router.navigate(`/news?q=${encodeURIComponent(q)}`);
  });

  initTabs();
  initToolsButton();
  initFilters();

  if (query) {
    addRecentSearch(query);
  }

  fetchAndRenderNews(query, currentFilters);
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

function initFilters(): void {
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
        chip.querySelector('.filter-chip-label')!.textContent = filterId === 'sort' ? 'Sort by' : 'Time';
      } else {
        (currentFilters as Record<string, string>)[filterId] = value;
        chip.classList.add('has-value');
        chip.querySelector('.filter-chip-label')!.textContent = formatFilterOption(filterId, value);
      }

      dropdown.classList.add('hidden');
      updateClearButton();

      fetchAndRenderNews(currentQuery, currentFilters);
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
        (chip.querySelector('.filter-chip-label') as HTMLElement).textContent = filterId === 'sort' ? 'Sort by' : 'Time';
      });

      toolbar.querySelectorAll('.filter-dropdown').forEach(dropdown => {
        dropdown.querySelectorAll('.filter-option').forEach((opt, i) => {
          opt.classList.toggle('active', i === 0);
        });
      });

      updateClearButton();
      fetchAndRenderNews(currentQuery, currentFilters);
    });
  }
}

function updateClearButton(): void {
  const clearBtn = document.getElementById('clear-filters');
  if (!clearBtn) return;
  clearBtn.classList.toggle('hidden', Object.keys(currentFilters).length === 0);
}

async function fetchAndRenderNews(query: string, filters: NewsFilters): Promise<void> {
  const content = document.getElementById('news-content');
  if (!content || !query) return;

  content.innerHTML = '<div class="flex items-center justify-center py-16"><div class="spinner"></div></div>';

  try {
    const response = await api.searchNews(query, {
      page: 1,
      per_page: 20,
      time_range: filters.time,
    });
    const results = response.results as NewsResult[];

    if (results.length === 0) {
      content.innerHTML = `
        <div class="py-8 text-secondary">No news results found for "<strong>${escapeHtml(query)}</strong>"</div>
      `;
      return;
    }

    // Top stories: first 5 articles (show all, not just with images)
    const topStories = results.slice(0, 5);
    const remainingResults = results.slice(5);

    const topStoriesHtml = topStories.length > 0 ? `
      <div class="news-top-stories mb-8">
        <h2 class="text-lg font-medium text-primary mb-4">Top Stories</h2>
        <div class="news-carousel">
          ${topStories.map((article) => renderTopStoryCard(article)).join('')}
        </div>
      </div>
    ` : '';

    content.innerHTML = `
      <div class="news-results-container">
        <div class="text-xs text-tertiary mb-6">
          About ${response.total_results.toLocaleString()} news results (${(response.search_time_ms / 1000).toFixed(2)} seconds)
        </div>
        ${topStoriesHtml}
        ${remainingResults.length > 0 ? `
          <div class="news-list">
            ${remainingResults.map((article) => renderNewsCard(article)).join('')}
          </div>
        ` : ''}
      </div>
    `;
  } catch (err) {
    content.innerHTML = `
      <div class="py-8">
        <p class="text-red text-sm">Failed to load news results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${escapeHtml(String(err))}</p>
      </div>
    `;
  }
}

function renderTopStoryCard(article: NewsResult): string {
  // Use thumbnail if available, otherwise use a gradient placeholder
  const thumbnailUrl = article.thumbnail?.url || '';
  const publishedDate = article.published_date ? formatDate(article.published_date) : '';

  return `
    <a href="${escapeAttr(article.url)}" target="_blank" rel="noopener" class="news-top-card">
      <div class="news-top-card-image">
        ${thumbnailUrl
          ? `<img src="${escapeAttr(thumbnailUrl)}" alt="" loading="lazy" onerror="this.style.display='none'; this.nextElementSibling.style.display='flex'" />`
          : ''
        }
        <div class="news-image-placeholder" ${thumbnailUrl ? 'style="display:none"' : ''}>
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#9aa0a6" stroke-width="1.5">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
            <circle cx="9" cy="9" r="2"></circle>
            <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path>
          </svg>
        </div>
      </div>
      <div class="news-top-card-content">
        <div class="news-source">
          <img class="news-source-icon" src="https://www.google.com/s2/favicons?domain=${encodeURIComponent(article.domain)}&sz=16" alt="" onerror="this.style.display='none'" />
          <span>${escapeHtml(article.source || article.domain)}</span>
          ${publishedDate ? `<span class="news-time">· ${escapeHtml(publishedDate)}</span>` : ''}
        </div>
        <h3 class="news-top-card-title">${escapeHtml(article.title)}</h3>
      </div>
    </a>
  `;
}

function renderNewsCard(article: NewsResult): string {
  const thumbnailUrl = article.thumbnail?.url || '';
  const publishedDate = article.published_date ? formatDate(article.published_date) : '';

  return `
    <article class="news-card-item">
      <div class="news-card-main">
        <div class="news-card-source">
          <img class="news-favicon" src="https://www.google.com/s2/favicons?domain=${encodeURIComponent(article.domain)}&sz=16" alt="" onerror="this.style.display='none'" />
          <span>${escapeHtml(article.source || article.domain)}</span>
          ${publishedDate ? `<span class="news-card-time">· ${escapeHtml(publishedDate)}</span>` : ''}
        </div>
        <h3 class="news-card-headline">
          <a href="${escapeAttr(article.url)}" target="_blank" rel="noopener">${escapeHtml(article.title)}</a>
        </h3>
        <p class="news-card-snippet">${escapeHtml(article.snippet || '')}</p>
      </div>
      <div class="news-card-thumb">
        ${thumbnailUrl
          ? `<img src="${escapeAttr(thumbnailUrl)}" alt="" loading="lazy" onerror="this.style.display='none'; this.nextElementSibling.style.display='flex'" />`
          : ''
        }
        <div class="news-thumb-placeholder" ${thumbnailUrl ? 'style="display:none"' : ''}>
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#9aa0a6" stroke-width="1.5">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
            <circle cx="9" cy="9" r="2"></circle>
            <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path>
          </svg>
        </div>
      </div>
    </article>
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
