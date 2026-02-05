import { Router } from '../lib/router';
import { api } from '../api';
import type { SearchResponse, ImageResult } from '../api';
import { addRecentSearch, appState } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderSearchResult, initSearchResults } from '../components/search-result';
import { renderInstantAnswer } from '../components/instant-answer';
import { renderKnowledgePanel, initKnowledgePanel } from '../components/knowledge-panel';
import { renderPagination, initPagination } from '../components/pagination';
import { renderTabs, initTabs } from '../components/tabs';
import { renderPeopleAlsoAsk, initPeopleAlsoAsk } from '../components/people-also-ask';
import { renderSearchTools, initSearchTools, type SearchFilters } from '../components/search-tools';

const ICON_IMAGES = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>`;
const ICON_ARROW_RIGHT = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>`;

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;

export function renderSearchPage(query: string, filters: SearchFilters = {}): string {
  return `
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
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

  // Split results to insert image carousel after first 3 results
  const firstResults = response.results.slice(0, 3);
  const remainingResults = response.results.slice(3);

  const firstResultsHtml = firstResults.length > 0
    ? firstResults.map((r, i) => renderSearchResult(r, i)).join('')
    : '';

  const remainingResultsHtml = remainingResults.length > 0
    ? remainingResults.map((r, i) => renderSearchResult(r, i + 3)).join('')
    : '';

  const noResultsHtml = response.results.length === 0
    ? `<div class="py-8 text-secondary">No results found for "<strong>${escapeHtml(query)}</strong>"</div>`
    : '';

  // Enhanced related searches with icons
  const relatedHtml =
    response.related_searches && response.related_searches.length > 0
      ? `
      <div class="related-searches-section">
        <h3 class="related-title">Related searches</h3>
        <div class="related-grid">
          ${response.related_searches
            .map(
              (r) => `
            <a href="/search?q=${encodeURIComponent(r)}" data-link class="related-item">
              <span class="related-icon">${ICON_SEARCH}</span>
              <span class="related-text">${escapeHtml(r)}</span>
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
        ${noResultsHtml}
        ${firstResultsHtml}
        <div id="images-carousel-slot"></div>
        ${paaHtml}
        ${remainingResultsHtml}
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

  // Load image carousel asynchronously
  if (page === 1 && response.results.length > 0) {
    loadImageCarousel(query, router);
  }
}

const ICON_SEARCH = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>`;

async function loadImageCarousel(query: string, router: Router): Promise<void> {
  const slot = document.getElementById('images-carousel-slot');
  if (!slot) return;

  try {
    const imageResponse = await api.searchImages(query, { per_page: 8 });
    const images = imageResponse.results as ImageResult[];

    if (images.length < 4) {
      slot.remove();
      return;
    }

    slot.innerHTML = `
      <div class="image-preview-carousel">
        <div class="carousel-header">
          <div class="carousel-title">
            ${ICON_IMAGES}
            <span>Images for "${escapeHtml(query)}"</span>
          </div>
          <a href="/images?q=${encodeURIComponent(query)}" data-link class="carousel-more">
            View all ${ICON_ARROW_RIGHT}
          </a>
        </div>
        <div class="carousel-images">
          ${images.map((img, i) => `
            <a href="/images?q=${encodeURIComponent(query)}" data-link class="carousel-image" data-index="${i}">
              <img src="${escapeAttr(img.thumbnail_url || img.url)}" alt="${escapeAttr(img.title)}" loading="lazy" />
            </a>
          `).join('')}
        </div>
      </div>
    `;

    // Add click handlers for images
    slot.querySelectorAll('.carousel-image').forEach((img) => {
      img.addEventListener('click', (e) => {
        e.preventDefault();
        router.navigate(`/images?q=${encodeURIComponent(query)}`);
      });
    });
  } catch {
    // Silently fail - image carousel is optional
    slot.remove();
  }
}

function formatNumber(n: number): string {
  return n.toLocaleString('en-US');
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
