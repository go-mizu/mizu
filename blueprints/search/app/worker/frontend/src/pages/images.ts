import { Router } from '../lib/router';
import { api } from '../api';
import type { ImageResult } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;

export function renderImagesPage(query: string): string {
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
          ${renderTabs({ query, active: 'images' })}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="images-content" class="max-w-[1200px] mx-auto px-4 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>

      <!-- Lightbox -->
      <div id="lightbox" class="lightbox hidden">
        <button class="lightbox-close" id="lightbox-close" aria-label="Close">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
        </button>
        <img id="lightbox-img" src="" alt="" />
      </div>
    </div>
  `;
}

export function initImagesPage(router: Router, query: string): void {
  initSearchBox((q) => {
    router.navigate(`/images?q=${encodeURIComponent(q)}`);
  });

  initTabs();

  if (query) {
    addRecentSearch(query);
  }

  fetchAndRenderImages(query);
  initLightbox();
}

async function fetchAndRenderImages(query: string): Promise<void> {
  const content = document.getElementById('images-content');
  if (!content || !query) return;

  try {
    const response = await api.searchImages(query);
    const results = response.results as ImageResult[];

    if (results.length === 0) {
      content.innerHTML = `
        <div class="py-8 text-secondary">No image results found for "<strong>${escapeHtml(query)}</strong>"</div>
      `;
      return;
    }

    content.innerHTML = `
      <div class="text-xs text-tertiary mb-4">
        About ${response.total_results.toLocaleString()} image results (${(response.search_time_ms / 1000).toFixed(2)} seconds)
      </div>
      <div class="image-grid">
        ${results
          .map(
            (img, i) => `
          <div class="image-card" data-image-index="${i}" data-full-url="${escapeAttr(img.url)}" data-source-url="${escapeAttr(img.source_url)}">
            <img
              src="${escapeAttr(img.thumbnail?.url || img.url)}"
              alt="${escapeAttr(img.title)}"
              loading="lazy"
              onerror="this.parentElement.style.display='none'"
            />
            <div class="image-info">
              <div class="image-title">${escapeHtml(img.title)}</div>
              <div class="image-source">${escapeHtml(img.domain)}</div>
            </div>
          </div>
        `
          )
          .join('')}
      </div>
    `;

    // Attach click handlers to image cards
    content.querySelectorAll('.image-card').forEach((card) => {
      card.addEventListener('click', () => {
        const fullUrl = (card as HTMLElement).dataset.fullUrl || '';
        openLightbox(fullUrl);
      });
    });
  } catch (err) {
    content.innerHTML = `
      <div class="py-8">
        <p class="text-red text-sm">Failed to load image results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${escapeHtml(String(err))}</p>
      </div>
    `;
  }
}

function initLightbox(): void {
  const lightbox = document.getElementById('lightbox');
  const closeBtn = document.getElementById('lightbox-close');

  if (!lightbox || !closeBtn) return;

  closeBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    closeLightbox();
  });

  lightbox.addEventListener('click', (e) => {
    if (e.target === lightbox) {
      closeLightbox();
    }
  });

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      closeLightbox();
    }
  });
}

function openLightbox(url: string): void {
  const lightbox = document.getElementById('lightbox');
  const img = document.getElementById('lightbox-img') as HTMLImageElement;
  if (!lightbox || !img) return;

  img.src = url;
  lightbox.classList.remove('hidden');
  document.body.style.overflow = 'hidden';
}

function closeLightbox(): void {
  const lightbox = document.getElementById('lightbox');
  if (!lightbox) return;

  lightbox.classList.add('hidden');
  document.body.style.overflow = '';
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
