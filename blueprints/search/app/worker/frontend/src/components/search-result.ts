import { api } from '../api';
import type { SearchResult } from '../api';

const ICON_DOTS = `<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>`;
const ICON_THUMBS_UP = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 10v12"/><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2h0a3.13 3.13 0 0 1 3 3.88Z"/></svg>`;
const ICON_THUMBS_DOWN = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 14V2"/><path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22h0a3.13 3.13 0 0 1-3-3.88Z"/></svg>`;
const ICON_BAN = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>`;
const ICON_GLOBE_FALLBACK = `<svg class="favicon-fallback" style="display:none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`;

export function renderSearchResult(result: SearchResult, index: number): string {
  const faviconUrl = result.favicon || `https://www.google.com/s2/favicons?domain=${encodeURIComponent(result.domain)}&sz=32`;
  const breadcrumbs = getBreadcrumbs(result.url);
  const published = result.published ? formatDate(result.published) : '';
  const snippet = result.snippet || '';
  const thumbnailHtml = result.thumbnail
    ? `<img src="${escapeAttr(result.thumbnail.url)}" alt="" class="w-[120px] h-[80px] rounded-lg object-cover flex-shrink-0 ml-4" loading="lazy" />`
    : '';

  // Sitelinks in 2-column grid (up to 4 items)
  const sitelinksHtml = result.sitelinks && result.sitelinks.length > 0
    ? `<div class="sitelinks-grid">
        ${result.sitelinks.slice(0, 4).map((sl) => `
          <div class="sitelink">
            <a href="${escapeAttr(sl.url)}" target="_blank" rel="noopener">${escapeHtml(sl.title)}</a>
          </div>
        `).join('')}
       </div>`
    : '';

  // Rich snippet (rating/price) from metadata
  const meta = result.metadata || {};
  const rating = typeof meta.rating === 'number' ? meta.rating : null;
  const richSnippetHtml = rating !== null ? `
    <div class="rich-snippet">
      <span class="rating-stars">${'★'.repeat(Math.round(rating))}${'☆'.repeat(5 - Math.round(rating))}</span>
      <span class="rating-value">${rating.toFixed(1)}</span>
      ${meta.reviewCount ? `<span class="rating-count">(${formatNumber(meta.reviewCount)} reviews)</span>` : ''}
      ${meta.price ? `<span class="price">${escapeHtml(String(meta.price))}</span>` : ''}
    </div>
  ` : '';

  return `
    <div class="search-result" data-result-index="${index}" data-domain="${escapeAttr(result.domain)}">
      <div class="result-url">
        <div class="favicon">
          <img src="${escapeAttr(faviconUrl)}" alt="" loading="lazy" onerror="this.style.display='none'; this.nextElementSibling.style.display='block';" />
          ${ICON_GLOBE_FALLBACK}
        </div>
        <div>
          <span class="text-sm">${escapeHtml(result.domain)}</span>
          <span class="breadcrumbs">${breadcrumbs}</span>
        </div>
      </div>
      <div class="flex items-start">
        <div class="flex-1">
          <div class="result-title">
            <a href="${escapeAttr(result.url)}" target="_blank" rel="noopener">${escapeHtml(result.title)}</a>
          </div>
          ${richSnippetHtml}
          <div class="result-snippet">
            ${published ? `<span class="result-date">${escapeHtml(published)} — </span>` : ''}${snippet}
          </div>
          ${sitelinksHtml}
        </div>
        ${thumbnailHtml}
      </div>
      <button class="result-menu-btn" data-menu-index="${index}" aria-label="More options">
        ${ICON_DOTS}
      </button>
      <div id="domain-menu-${index}" class="domain-menu hidden"></div>
    </div>
  `;
}

function formatNumber(num: number): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1).replace(/\.0$/, '') + 'M';
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1).replace(/\.0$/, '') + 'K';
  }
  return num.toLocaleString();
}

export function initSearchResults(): void {
  document.querySelectorAll('.result-menu-btn').forEach((btn) => {
    btn.addEventListener('click', (e) => {
      e.stopPropagation();
      const index = (btn as HTMLElement).dataset.menuIndex;
      const menu = document.getElementById(`domain-menu-${index}`);
      const resultEl = btn.closest('.search-result') as HTMLElement;
      const domain = resultEl?.dataset.domain || '';

      if (!menu) return;

      if (!menu.classList.contains('hidden')) {
        menu.classList.add('hidden');
        return;
      }

      // Close all other menus
      document.querySelectorAll('.domain-menu').forEach((m) => m.classList.add('hidden'));

      menu.innerHTML = `
        <button class="domain-menu-item boost" data-action="boost" data-domain="${escapeAttr(domain)}">
          ${ICON_THUMBS_UP}
          <span>Boost ${escapeHtml(domain)}</span>
        </button>
        <button class="domain-menu-item lower" data-action="lower" data-domain="${escapeAttr(domain)}">
          ${ICON_THUMBS_DOWN}
          <span>Lower ${escapeHtml(domain)}</span>
        </button>
        <button class="domain-menu-item block" data-action="block" data-domain="${escapeAttr(domain)}">
          ${ICON_BAN}
          <span>Block ${escapeHtml(domain)}</span>
        </button>
      `;
      menu.classList.remove('hidden');

      menu.querySelectorAll('.domain-menu-item').forEach((item) => {
        item.addEventListener('click', async () => {
          const action = (item as HTMLElement).dataset.action || '';
          const dom = (item as HTMLElement).dataset.domain || '';
          try {
            await api.setPreference(dom, action);
            menu.classList.add('hidden');
            showToast(`${action.charAt(0).toUpperCase() + action.slice(1)}ed ${dom}`);
          } catch (err) {
            console.error('Failed to set preference:', err);
          }
        });
      });

      // Close on outside click
      const closeHandler = (ev: MouseEvent) => {
        if (!menu.contains(ev.target as Node) && ev.target !== btn) {
          menu.classList.add('hidden');
          document.removeEventListener('click', closeHandler);
        }
      };
      setTimeout(() => document.addEventListener('click', closeHandler), 0);
    });
  });
}

function showToast(message: string): void {
  const existing = document.getElementById('toast');
  if (existing) existing.remove();

  const toast = document.createElement('div');
  toast.id = 'toast';
  toast.className = 'fixed bottom-6 left-1/2 -translate-x-1/2 bg-primary text-white px-5 py-3 rounded-lg shadow-lg text-sm z-50 transition-opacity duration-300';
  toast.textContent = message;
  document.body.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    setTimeout(() => toast.remove(), 300);
  }, 2000);
}

function getBreadcrumbs(url: string): string {
  try {
    const u = new URL(url);
    const parts = u.pathname.split('/').filter(Boolean);
    if (parts.length === 0) return '';
    return ' > ' + parts.map((p) => escapeHtml(decodeURIComponent(p))).join(' > ');
  } catch {
    return '';
  }
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

    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  } catch {
    return dateStr;
  }
}

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}
