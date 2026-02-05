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
import { renderPeopleAlsoAsk, initPeopleAlsoAsk } from '../components/people-also-ask';
import { renderSearchTools, initSearchTools, type SearchFilters } from '../components/search-tools';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;

export function renderSearchPage(query: string, filters: SearchFilters = {}): string {
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
          ${renderTabs({ query, active: 'all' })}
        </div>
        <!-- Search Tools Bar -->
        ${renderSearchTools(filters)}
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
  const filters: SearchFilters = {
    timeRange: queryParams.time_range || '',
    region: queryParams.region || '',
    verbatim: queryParams.verbatim === '1',
    site: queryParams.site || '',
  };
  const settings = appState.get().settings;

  // Init search box
  initSearchBox((q) => {
    router.navigate(`/search?q=${encodeURIComponent(q)}`);
  });

  // Init tabs
  initTabs();

  // Init search tools
  initSearchTools((newFilters) => {
    const url = buildSearchUrl(query, newFilters);
    router.navigate(url);
  });

  // Record search
  if (query) {
    addRecentSearch(query);
  }

  // Fetch results
  fetchAndRenderResults(router, query, page, filters, settings.results_per_page);
}

function buildSearchUrl(query: string, filters: SearchFilters, page?: number): string {
  const params = new URLSearchParams();
  params.set('q', query);

  if (page && page > 1) {
    params.set('page', String(page));
  }
  if (filters.timeRange) {
    params.set('time_range', filters.timeRange);
  }
  if (filters.region) {
    params.set('region', filters.region);
  }
  if (filters.verbatim) {
    params.set('verbatim', '1');
  }
  if (filters.site) {
    params.set('site', filters.site);
  }

  return `/search?${params.toString()}`;
}

async function fetchAndRenderResults(
  router: Router,
  query: string,
  page: number,
  filters: SearchFilters,
  perPage: number
): Promise<void> {
  const content = document.getElementById('search-content');
  if (!content || !query) return;

  // Build the effective query with site filter
  let effectiveQuery = query;
  if (filters.site) {
    effectiveQuery = `site:${filters.site} ${query}`;
  }

  try {
    const response = await api.search(effectiveQuery, {
      page,
      per_page: perPage,
      time_range: filters.timeRange || undefined,
      region: filters.region || undefined,
      verbatim: filters.verbatim || undefined,
    });

    // Handle bang redirect
    if (response.redirect) {
      window.location.href = response.redirect;
      return;
    }

    renderResults(content, router, response, query, page, filters);
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
  filters: SearchFilters
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

  // Generate People Also Ask section from related searches
  const paaQuestions = response.related_searches?.slice(0, 4).map((q) => ({
    question: q,
    answer: undefined, // Would be fetched on demand
  })) || [];
  const paaHtml = paaQuestions.length > 0 ? renderPeopleAlsoAsk(paaQuestions) : '';

  const resultsHtml = response.results.length > 0
    ? response.results.map((r, i) => renderSearchResult(r, i)).join('')
    : `<div class="py-8 text-secondary">No results found for "<strong>${escapeHtml(query)}</strong>"</div>`;

  const relatedHtml =
    response.related_searches && response.related_searches.length > 0
      ? `
      <div class="mt-8 mb-4">
        <h3 class="text-lg font-medium text-primary mb-3">Related searches</h3>
        <div class="related-searches-pills">
          ${response.related_searches
            .map(
              (r) => `
            <a href="/search?q=${encodeURIComponent(r)}" data-link class="related-pill">${escapeHtml(r)}</a>
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
        ${paaHtml}
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
  initPeopleAlsoAsk();
  initPagination((newPage) => {
    const url = buildSearchUrl(query, filters, newPage);
    router.navigate(url);
  });
}

function formatNumber(n: number): string {
  return n.toLocaleString('en-US');
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
