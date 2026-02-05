import { Router } from '../lib/router';
import { api } from '../api';
import type { SearchHistory } from '../api';

const ICON_ARROW_LEFT = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>`;
const ICON_TRASH = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>`;
const ICON_SEARCH = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>`;
const ICON_HISTORY_CLOCK = `<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>`;

export function renderHistoryPage(): string {
  return `
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
              ${ICON_ARROW_LEFT}
            </a>
            <h1 class="text-xl font-semibold text-primary">Search History</h1>
          </div>
          <button id="clear-all-btn" class="text-sm text-red hover:text-red/80 font-medium cursor-pointer hidden">
            Clear all
          </button>
        </div>
      </header>

      <!-- Content -->
      <main class="max-w-[700px] mx-auto px-4 py-6">
        <div id="history-content">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `;
}

export function initHistoryPage(router: Router): void {
  const clearAllBtn = document.getElementById('clear-all-btn');

  fetchAndRenderHistory(router);

  clearAllBtn?.addEventListener('click', async () => {
    if (!confirm('Are you sure you want to clear all search history?')) return;

    try {
      await api.clearHistory();
      renderEmptyState();
      clearAllBtn.classList.add('hidden');
    } catch (err) {
      console.error('Failed to clear history:', err);
    }
  });
}

async function fetchAndRenderHistory(router: Router): Promise<void> {
  const content = document.getElementById('history-content');
  const clearAllBtn = document.getElementById('clear-all-btn');
  if (!content) return;

  try {
    const history = await api.getHistory();

    if (history.length === 0) {
      renderEmptyState();
      return;
    }

    if (clearAllBtn) {
      clearAllBtn.classList.remove('hidden');
    }

    content.innerHTML = `
      <div id="history-list">
        ${history.map((item) => renderHistoryItem(item)).join('')}
      </div>
    `;

    initHistoryItems(router);
  } catch (err) {
    content.innerHTML = `
      <div class="py-8 text-center">
        <p class="text-red text-sm">Failed to load search history.</p>
        <p class="text-tertiary text-xs mt-2">${escapeHtml(String(err))}</p>
      </div>
    `;
  }
}

function renderHistoryItem(item: SearchHistory): string {
  const date = formatDate(item.searched_at);

  return `
    <div class="history-item flex items-center gap-3 py-3 px-2 border-b border-border hover:bg-surface-hover rounded transition-colors group" data-history-id="${escapeAttr(item.id)}">
      <span class="text-light flex-shrink-0">${ICON_SEARCH}</span>
      <div class="flex-1 min-w-0">
        <a href="/search?q=${encodeURIComponent(item.query)}" data-link class="text-sm text-primary hover:text-link font-medium truncate block">
          ${escapeHtml(item.query)}
        </a>
        <div class="flex items-center gap-2 text-xs text-light mt-0.5">
          <span>${escapeHtml(date)}</span>
          ${item.results > 0 ? `<span>&middot; ${item.results} results</span>` : ''}
          ${item.clicked_url ? `<span>&middot; visited</span>` : ''}
        </div>
      </div>
      <button class="history-delete-btn text-light hover:text-red p-1.5 rounded-full hover:bg-red/10 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0 cursor-pointer"
              data-delete-id="${escapeAttr(item.id)}" aria-label="Delete">
        ${ICON_TRASH}
      </button>
    </div>
  `;
}

function initHistoryItems(router: Router): void {
  document.querySelectorAll('.history-delete-btn').forEach((btn) => {
    btn.addEventListener('click', async (e) => {
      e.preventDefault();
      e.stopPropagation();

      const id = (btn as HTMLElement).dataset.deleteId || '';
      const item = btn.closest('.history-item');

      try {
        await api.deleteHistoryItem(id);
        if (item) {
          item.remove();
        }

        // Check if list is now empty
        const list = document.getElementById('history-list');
        if (list && list.children.length === 0) {
          renderEmptyState();
          const clearAllBtn = document.getElementById('clear-all-btn');
          if (clearAllBtn) clearAllBtn.classList.add('hidden');
        }
      } catch (err) {
        console.error('Failed to delete history item:', err);
      }
    });
  });
}

function renderEmptyState(): void {
  const content = document.getElementById('history-content');
  if (!content) return;

  content.innerHTML = `
    <div class="py-16 flex flex-col items-center text-center">
      ${ICON_HISTORY_CLOCK}
      <h2 class="text-lg font-medium text-primary mt-4 mb-2">No search history</h2>
      <p class="text-sm text-tertiary max-w-[300px]">
        Your recent searches will appear here. Start searching to build your history.
      </p>
      <a href="/" data-link class="mt-4 text-sm text-blue hover:underline">Go to search</a>
    </div>
  `;
}

function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - d.getTime();
    const mins = Math.floor(diff / (1000 * 60));
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (mins < 1) return 'Just now';
    if (mins < 60) return `${mins}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days === 1) return 'Yesterday';
    if (days < 7) return `${days} days ago`;

    return d.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: d.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
    });
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
