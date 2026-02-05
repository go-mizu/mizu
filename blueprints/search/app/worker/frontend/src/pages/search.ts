import { Router } from '../lib/router';
import { api } from '../api';
import type { SearchResponse } from '../api';
import { addRecentSearch, appState } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderSearchResult, initSearchResults } from '../components/search-result';
import { renderInstantAnswer } from '../components/instant-answer';
import { renderKnowledgePanel, initKnowledgePanel } from '../components/knowledge-panel';
import { renderPagination, initPagination } from '../components/pagination';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;
const ICON_CHEVRON_DOWN = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>`;

const TIME_RANGES = [
  { value: '', label: 'Any time' },
  { value: 'day', label: 'Past 24 hours' },
  { value: 'week', label: 'Past week' },
  { value: 'month', label: 'Past month' },
  { value: 'year', label: 'Past year' },
];

export function renderSearchPage(query: string, timeRange: string): string {
  const activeTimeLabel = TIME_RANGES.find((t) => t.value === timeRange)?.label || 'Any time';
  const hasTimeFilter = timeRange !== '';

  return `
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="search-header-box">
            ${renderSearchBox({ size: 'sm', initialValue: query })}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${ICON_SETTINGS}
          </a>
        </div>
        <div class="search-tabs-row">
          <div class="flex items-center gap-2">
            ${renderTabs({ query, active: 'all' })}
            <div class="time-filter ml-2" id="time-filter-wrapper">
              <button class="time-filter-btn ${hasTimeFilter ? 'active-filter' : ''}" id="time-filter-btn" type="button">
                <span id="time-filter-label">${escapeHtml(activeTimeLabel)}</span>
                ${ICON_CHEVRON_DOWN}
              </button>
              <div class="time-filter-dropdown hidden" id="time-filter-dropdown">
                ${TIME_RANGES.map(
                  (t) => `
                  <button class="time-filter-option ${t.value === timeRange ? 'active' : ''}" data-time-range="${t.value}">
                    ${escapeHtml(t.label)}
                  </button>
                `
                ).join('')}
              </div>
            </div>
          </div>
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="search-content" class="search-content-area">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `;
}

export function initSearchPage(router: Router, query: string, queryParams: Record<string, string>): void {
  const page = parseInt(queryParams.page || '1');
  const timeRange = queryParams.time_range || '';
  const settings = appState.get().settings;

  // Init search box
  initSearchBox((q) => {
    router.navigate(`/search?q=${encodeURIComponent(q)}`);
  });

  // Init tabs
  initTabs();

  // Init time filter
  initTimeFilter(router, query, timeRange);

  // Record search
  if (query) {
    addRecentSearch(query);
  }

  // Fetch results
  fetchAndRenderResults(router, query, page, timeRange, settings.results_per_page);
}

function initTimeFilter(router: Router, query: string, currentTimeRange: string): void {
  const btn = document.getElementById('time-filter-btn');
  const dropdown = document.getElementById('time-filter-dropdown');

  if (!btn || !dropdown) return;

  btn.addEventListener('click', (e) => {
    e.stopPropagation();
    dropdown.classList.toggle('hidden');
  });

  dropdown.querySelectorAll('.time-filter-option').forEach((opt) => {
    opt.addEventListener('click', () => {
      const range = (opt as HTMLElement).dataset.timeRange || '';
      dropdown.classList.add('hidden');
      let url = `/search?q=${encodeURIComponent(query)}`;
      if (range) url += `&time_range=${range}`;
      router.navigate(url);
    });
  });

  // Close on outside click
  document.addEventListener('click', (e) => {
    if (!dropdown.contains(e.target as Node) && e.target !== btn) {
      dropdown.classList.add('hidden');
    }
  });
}

async function fetchAndRenderResults(
  router: Router,
  query: string,
  page: number,
  timeRange: string,
  perPage: number
): Promise<void> {
  const content = document.getElementById('search-content');
  if (!content || !query) return;

  try {
    const response = await api.search(query, {
      page,
      per_page: perPage,
      time_range: timeRange || undefined,
    });

    // Handle bang redirect
    if (response.redirect) {
      window.location.href = response.redirect;
      return;
    }

    renderResults(content, router, response, query, page, timeRange);
  } catch (err) {
    content.innerHTML = `
      <div class="py-8">
        <p class="text-red text-sm">Failed to load search results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${escapeHtml(String(err))}</p>
      </div>
    `;
  }
}

function renderResults(
  container: HTMLElement,
  router: Router,
  response: SearchResponse,
  query: string,
  page: number,
  timeRange: string
): void {
  const correctedHtml = response.corrected_query
    ? `<p class="text-sm text-secondary mb-4">
        Showing results for <a href="/search?q=${encodeURIComponent(response.corrected_query)}" data-link class="text-link font-medium">${escapeHtml(response.corrected_query)}</a>.
        Search instead for <a href="/search?q=${encodeURIComponent(query)}&exact=1" data-link class="text-link">${escapeHtml(query)}</a>.
      </p>`
    : '';

  const statsHtml = `
    <div class="text-xs text-tertiary mb-4">
      About ${formatNumber(response.total_results)} results (${(response.search_time_ms / 1000).toFixed(2)} seconds)
    </div>
  `;

  const instantHtml = response.instant_answer ? renderInstantAnswer(response.instant_answer) : '';

  const resultsHtml = response.results.length > 0
    ? response.results.map((r, i) => renderSearchResult(r, i)).join('')
    : `<div class="py-8 text-secondary">No results found for "<strong>${escapeHtml(query)}</strong>"</div>`;

  const relatedHtml =
    response.related_searches && response.related_searches.length > 0
      ? `
      <div class="mt-8 mb-4">
        <h3 class="text-lg font-medium text-primary mb-3">Related searches</h3>
        <div class="grid grid-cols-2 gap-2 max-w-[600px]">
          ${response.related_searches
            .map(
              (r) => `
            <a href="/search?q=${encodeURIComponent(r)}" data-link class="flex items-center gap-2 p-3 rounded-lg bg-surface hover:bg-surface-hover text-sm text-primary transition-colors">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#9aa0a6" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              ${escapeHtml(r)}
            </a>
          `
            )
            .join('')}
        </div>
      </div>
    `
      : '';

  const paginationHtml = renderPagination({
    currentPage: page,
    hasMore: response.has_more,
    totalResults: response.total_results,
    perPage: response.per_page,
  });

  const knowledgePanelHtml = response.knowledge_panel
    ? renderKnowledgePanel(response.knowledge_panel)
    : '';

  container.innerHTML = `
    <div class="search-results-layout">
      <div class="search-results-main">
        ${correctedHtml}
        ${statsHtml}
        ${instantHtml}
        ${resultsHtml}
        ${relatedHtml}
        ${paginationHtml}
      </div>
      ${knowledgePanelHtml ? `<aside class="search-results-sidebar">${knowledgePanelHtml}</aside>` : ''}
    </div>
  `;

  // Init interactive components
  initSearchResults();
  initKnowledgePanel();
  initPagination((newPage) => {
    let url = `/search?q=${encodeURIComponent(query)}&page=${newPage}`;
    if (timeRange) url += `&time_range=${timeRange}`;
    router.navigate(url);
  });
}

function formatNumber(n: number): string {
  return n.toLocaleString('en-US');
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
