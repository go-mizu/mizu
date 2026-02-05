import { Router } from '../lib/router';
import { api } from '../api';
import type { SearchResult } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;
const ICON_PDF = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><path d="M12 18v-6"/><path d="m9 15 3 3 3-3"/></svg>`;
const ICON_CITE = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V21"/><path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3"/></svg>`;

export function renderSciencePage(query: string): string {
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
          ${renderTabs({ query, active: 'science' })}
        </div>
      </header>
      <main class="flex-1">
        <div id="science-content" class="search-content-area">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `;
}

export function initSciencePage(router: Router, query: string): void {
  initSearchBox((q) => {
    router.navigate(`/science?q=${encodeURIComponent(q)}`);
  });

  initTabs();

  if (query) {
    addRecentSearch(query);
  }

  fetchAndRenderScience(query);
}

async function fetchAndRenderScience(query: string): Promise<void> {
  const content = document.getElementById('science-content');
  if (!content || !query) return;

  try {
    const response = await api.searchScience(query);
    const results = response.results;

    if (results.length === 0) {
      content.innerHTML = `
        <div class="w-full">
          <div class="py-8 text-secondary">No academic results found for "<strong>${escapeHtml(query)}</strong>"</div>
        </div>
      `;
      return;
    }

    content.innerHTML = `
      <div class="w-full">
        <div class="text-xs text-tertiary mb-4">
          About ${response.total_results.toLocaleString()} results (${(response.search_time_ms / 1000).toFixed(2)} seconds)
        </div>
        <div class="space-y-6">
          ${results.map(renderPaperCard).join('')}
        </div>
      </div>
    `;
  } catch (err) {
    content.innerHTML = `
      <div class="w-full">
        <div class="py-8">
          <p class="text-red text-sm">Failed to load academic results. Please try again.</p>
          <p class="text-tertiary text-xs mt-2">${escapeHtml(String(err))}</p>
        </div>
      </div>
    `;
  }
}

function renderPaperCard(paper: SearchResult): string {
  // Extract metadata from paper result
  const metadata = paper as SearchResult & {
    metadata?: {
      authors?: string;
      year?: string;
      citations?: number;
      doi?: string;
      pdf_url?: string;
      source?: string;
    };
  };
  const authors = metadata.metadata?.authors || '';
  const year = metadata.metadata?.year || '';
  const citations = metadata.metadata?.citations;
  const doi = metadata.metadata?.doi || '';
  const pdfUrl = metadata.metadata?.pdf_url || '';
  const source = metadata.metadata?.source || extractDomain(paper.url);

  return `
    <article class="paper-card bg-white border border-border rounded-xl p-5 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3 mb-2">
        <span class="text-xs px-2 py-0.5 bg-blue/10 text-blue rounded-full font-medium">${escapeHtml(source)}</span>
        ${year ? `<span class="text-xs text-tertiary">${escapeHtml(year)}</span>` : ''}
      </div>
      <h3 class="text-lg font-medium text-primary mb-2">
        <a href="${escapeAttr(paper.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${escapeHtml(paper.title)}</a>
      </h3>
      ${authors ? `<p class="text-sm text-secondary mb-2">${escapeHtml(authors)}</p>` : ''}
      <p class="text-sm text-snippet line-clamp-3 mb-3">${escapeHtml(paper.snippet ?? '')}</p>
      <div class="flex items-center gap-4 text-xs">
        ${citations !== undefined && citations !== null ? `<span class="flex items-center gap-1 text-tertiary">${ICON_CITE} ${escapeHtml(String(citations))} citations</span>` : ''}
        ${doi ? `<a href="https://doi.org/${escapeAttr(doi)}" target="_blank" rel="noopener" class="text-tertiary hover:text-blue">DOI: ${escapeHtml(doi)}</a>` : ''}
        ${pdfUrl ? `<a href="${escapeAttr(pdfUrl)}" target="_blank" rel="noopener" class="flex items-center gap-1 text-blue hover:underline">${ICON_PDF} PDF</a>` : ''}
      </div>
    </article>
  `;
}

function extractDomain(url: string): string {
  try {
    return new URL(url).hostname.replace('www.', '');
  } catch {
    return '';
  }
}

function escapeHtml(value: unknown): string {
  const str = value === null || value === undefined ? '' : String(value);
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(value: unknown): string {
  const str = value === null || value === undefined ? '' : String(value);
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
