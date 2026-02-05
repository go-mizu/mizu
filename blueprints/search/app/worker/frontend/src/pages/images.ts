import { Router } from '../lib/router';
import { api } from '../api';
import type { ImageResult, ImageSearchFilters } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;

const ICON_CAMERA = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>`;

const ICON_CLOSE = `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>`;

const ICON_EXTERNAL = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>`;

// Current state
let currentQuery = '';
let currentFilters: ImageSearchFilters = {};
let currentPage = 1;
let isLoading = false;
let hasMore = true;
let allImages: ImageResult[] = [];

export function renderImagesPage(query: string): string {
  return `
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1400px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px] flex items-center gap-2">
            ${renderSearchBox({ size: 'sm', initialValue: query })}
            <button id="reverse-search-btn" class="flex-shrink-0 p-2 text-tertiary hover:text-primary hover:bg-surface-hover rounded-full transition-colors" title="Search by image">
              ${ICON_CAMERA}
            </button>
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${ICON_SETTINGS}
          </a>
        </div>
        <div class="max-w-[1400px] pl-[170px]">
          ${renderTabs({ query, active: 'images' })}
        </div>
        <!-- Filter toolbar -->
        <div id="filter-toolbar" class="max-w-[1400px] px-4 py-2 flex flex-wrap gap-2 items-center border-t border-border/50">
          ${renderFilterToolbar()}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1 flex">
        <div id="images-content" class="flex-1 max-w-[1400px] mx-auto px-4 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>

        <!-- Preview panel (hidden by default) -->
        <div id="preview-panel" class="preview-panel hidden">
          <div class="preview-panel-content">
            <button id="preview-close" class="preview-close" aria-label="Close">${ICON_CLOSE}</button>
            <div id="preview-image-container" class="preview-image-container">
              <img id="preview-image" src="" alt="" />
            </div>
            <div id="preview-details" class="preview-details"></div>
          </div>
        </div>
      </main>

      <!-- Reverse image search modal -->
      <div id="reverse-modal" class="modal hidden">
        <div class="modal-content">
          <div class="modal-header">
            <h2>Search by image</h2>
            <button id="reverse-modal-close" class="modal-close">${ICON_CLOSE}</button>
          </div>
          <div class="modal-body">
            <div id="drop-zone" class="drop-zone">
              <p>Drag and drop an image here or</p>
              <label class="file-upload-btn">
                Upload a file
                <input type="file" id="image-upload" accept="image/*" hidden />
              </label>
            </div>
            <div class="url-input-section">
              <p>Or paste an image URL:</p>
              <div class="url-input-container">
                <input type="text" id="image-url-input" placeholder="https://example.com/image.jpg" />
                <button id="url-search-btn">Search</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  `;
}

function renderFilterToolbar(): string {
  const filters = [
    { id: 'size', label: 'Size', options: ['any', 'large', 'medium', 'small', 'icon'] },
    { id: 'color', label: 'Color', options: ['any', 'color', 'gray', 'transparent', 'red', 'orange', 'yellow', 'green', 'teal', 'blue', 'purple', 'pink', 'white', 'black', 'brown'] },
    { id: 'type', label: 'Type', options: ['any', 'photo', 'clipart', 'lineart', 'animated', 'face'] },
    { id: 'aspect', label: 'Aspect', options: ['any', 'tall', 'square', 'wide', 'panoramic'] },
    { id: 'time', label: 'Time', options: ['any', 'day', 'week', 'month', 'year'] },
    { id: 'rights', label: 'Rights', options: ['any', 'creative_commons', 'commercial'] },
  ];

  return filters.map(f => `
    <select id="filter-${f.id}" class="filter-select" data-filter="${f.id}">
      ${f.options.map(opt => `<option value="${opt}">${formatFilterOption(f.id, opt)}</option>`).join('')}
    </select>
  `).join('') + `
    <button id="clear-filters" class="filter-clear hidden">Clear filters</button>
  `;
}

function formatFilterOption(filterId: string, option: string): string {
  if (option === 'any') {
    return `Any ${filterId}`;
  }
  return option.charAt(0).toUpperCase() + option.slice(1).replace('_', ' ');
}

export function initImagesPage(router: Router, query: string): void {
  currentQuery = query;
  currentFilters = {};
  currentPage = 1;
  allImages = [];
  hasMore = true;

  initSearchBox((q) => {
    router.navigate(`/images?q=${encodeURIComponent(q)}`);
  });

  initTabs();

  if (query) {
    addRecentSearch(query);
  }

  initFilters(router);
  initReverseSearch(router);
  initPreviewPanel();
  initInfiniteScroll();

  fetchAndRenderImages(query, currentFilters);
}

function initFilters(router: Router): void {
  const toolbar = document.getElementById('filter-toolbar');
  if (!toolbar) return;

  toolbar.querySelectorAll('.filter-select').forEach(select => {
    select.addEventListener('change', () => {
      const filterId = (select as HTMLSelectElement).dataset.filter;
      const value = (select as HTMLSelectElement).value;

      if (value === 'any') {
        delete (currentFilters as Record<string, string>)[filterId!];
      } else {
        (currentFilters as Record<string, string>)[filterId!] = value;
      }

      // Reset and refetch
      currentPage = 1;
      allImages = [];
      hasMore = true;
      updateClearButton();
      fetchAndRenderImages(currentQuery, currentFilters);
    });
  });

  const clearBtn = document.getElementById('clear-filters');
  if (clearBtn) {
    clearBtn.addEventListener('click', () => {
      currentFilters = {};
      currentPage = 1;
      allImages = [];
      hasMore = true;

      // Reset all selects
      toolbar.querySelectorAll('.filter-select').forEach(select => {
        (select as HTMLSelectElement).value = 'any';
      });

      updateClearButton();
      fetchAndRenderImages(currentQuery, currentFilters);
    });
  }
}

function updateClearButton(): void {
  const clearBtn = document.getElementById('clear-filters');
  if (!clearBtn) return;

  const hasFilters = Object.keys(currentFilters).length > 0;
  clearBtn.classList.toggle('hidden', !hasFilters);
}

function initReverseSearch(router: Router): void {
  const btn = document.getElementById('reverse-search-btn');
  const modal = document.getElementById('reverse-modal');
  const closeBtn = document.getElementById('reverse-modal-close');
  const dropZone = document.getElementById('drop-zone');
  const fileInput = document.getElementById('image-upload') as HTMLInputElement;
  const urlInput = document.getElementById('image-url-input') as HTMLInputElement;
  const urlSearchBtn = document.getElementById('url-search-btn');

  if (!btn || !modal) return;

  btn.addEventListener('click', () => {
    modal.classList.remove('hidden');
  });

  closeBtn?.addEventListener('click', () => {
    modal.classList.add('hidden');
  });

  modal.addEventListener('click', (e) => {
    if (e.target === modal) {
      modal.classList.add('hidden');
    }
  });

  // Drag and drop
  if (dropZone) {
    dropZone.addEventListener('dragover', (e) => {
      e.preventDefault();
      dropZone.classList.add('drag-over');
    });

    dropZone.addEventListener('dragleave', () => {
      dropZone.classList.remove('drag-over');
    });

    dropZone.addEventListener('drop', (e) => {
      e.preventDefault();
      dropZone.classList.remove('drag-over');

      const files = e.dataTransfer?.files;
      if (files && files[0]) {
        handleImageFile(files[0], router);
        modal.classList.add('hidden');
      }
    });
  }

  if (fileInput) {
    fileInput.addEventListener('change', () => {
      if (fileInput.files && fileInput.files[0]) {
        handleImageFile(fileInput.files[0], router);
        modal.classList.add('hidden');
      }
    });
  }

  if (urlSearchBtn && urlInput) {
    urlSearchBtn.addEventListener('click', () => {
      const url = urlInput.value.trim();
      if (url) {
        searchByImageUrl(url, router);
        modal.classList.add('hidden');
      }
    });

    urlInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        const url = urlInput.value.trim();
        if (url) {
          searchByImageUrl(url, router);
          modal.classList.add('hidden');
        }
      }
    });
  }
}

async function handleImageFile(_file: File, _router: Router): Promise<void> {
  // For now, show a message that file upload requires URL
  alert('Image upload coming soon. Please use the URL option for now.');
}

async function searchByImageUrl(url: string, _router: Router): Promise<void> {
  const content = document.getElementById('images-content');
  if (!content) return;

  content.innerHTML = `
    <div class="flex items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="ml-3 text-secondary">Searching for similar images...</span>
    </div>
  `;

  try {
    const response = await api.reverseImageSearch(url);

    content.innerHTML = `
      <div class="reverse-results">
        <div class="query-image-section">
          <h3>Search image</h3>
          <img src="${escapeAttr(url)}" alt="Query image" class="query-image" />
        </div>

        ${response.similar_images.length > 0 ? `
          <div class="similar-images-section">
            <h3>Similar images (${response.similar_images.length})</h3>
            <div class="image-grid">
              ${response.similar_images.map((img, i) => renderImageCard(img, i)).join('')}
            </div>
          </div>
        ` : `
          <div class="py-8 text-secondary">No similar images found.</div>
        `}
      </div>
    `;

    // Attach click handlers
    content.querySelectorAll('.image-card').forEach((card) => {
      card.addEventListener('click', () => {
        const idx = parseInt((card as HTMLElement).dataset.imageIndex || '0', 10);
        openPreview(response.similar_images[idx]);
      });
    });
  } catch (err) {
    content.innerHTML = `
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${escapeHtml(String(err))}</p>
      </div>
    `;
  }
}

function initPreviewPanel(): void {
  const closeBtn = document.getElementById('preview-close');
  if (closeBtn) {
    closeBtn.addEventListener('click', closePreview);
  }

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      closePreview();
    }
  });
}

function openPreview(image: ImageResult): void {
  const panel = document.getElementById('preview-panel');
  const imgEl = document.getElementById('preview-image') as HTMLImageElement;
  const details = document.getElementById('preview-details');

  if (!panel || !imgEl || !details) return;

  imgEl.src = image.url;
  imgEl.alt = image.title;

  details.innerHTML = `
    <h3 class="preview-title">${escapeHtml(image.title || 'Untitled')}</h3>
    <p class="preview-dimensions">${image.width} x ${image.height} ${image.format ? `- ${image.format.toUpperCase()}` : ''}</p>
    <p class="preview-source">${escapeHtml(image.source_domain)}</p>
    <div class="preview-actions">
      <a href="${escapeAttr(image.url)}" target="_blank" class="preview-btn">View image ${ICON_EXTERNAL}</a>
      <a href="${escapeAttr(image.source_url)}" target="_blank" class="preview-btn preview-btn-primary">Visit page ${ICON_EXTERNAL}</a>
    </div>
  `;

  panel.classList.remove('hidden');
  document.body.style.overflow = 'hidden';
}

function closePreview(): void {
  const panel = document.getElementById('preview-panel');
  if (!panel) return;

  panel.classList.add('hidden');
  document.body.style.overflow = '';
}

function initInfiniteScroll(): void {
  const sentinel = document.createElement('div');
  sentinel.id = 'scroll-sentinel';
  sentinel.style.height = '1px';

  const observer = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting && !isLoading && hasMore && currentQuery) {
      loadMoreImages();
    }
  }, { rootMargin: '200px' });

  // Will be added after initial render
  setTimeout(() => {
    const content = document.getElementById('images-content');
    if (content) {
      const existing = document.getElementById('scroll-sentinel');
      if (existing) existing.remove();
      content.appendChild(sentinel);
      observer.observe(sentinel);
    }
  }, 100);
}

async function loadMoreImages(): Promise<void> {
  if (isLoading || !hasMore) return;

  isLoading = true;
  currentPage++;

  try {
    const response = await api.searchImages(currentQuery, { ...currentFilters, page: currentPage });
    const newImages = response.results as ImageResult[];

    hasMore = response.has_more;
    allImages = [...allImages, ...newImages];

    // Append new images to grid
    const grid = document.querySelector('.image-grid');
    if (grid && newImages.length > 0) {
      const startIdx = allImages.length - newImages.length;
      const html = newImages.map((img, i) => renderImageCard(img, startIdx + i)).join('');
      grid.insertAdjacentHTML('beforeend', html);

      // Attach click handlers to new cards
      const newCards = grid.querySelectorAll('.image-card:not([data-initialized])');
      newCards.forEach((card) => {
        card.setAttribute('data-initialized', 'true');
        card.addEventListener('click', () => {
          const idx = parseInt((card as HTMLElement).dataset.imageIndex || '0', 10);
          openPreview(allImages[idx]);
        });
      });
    }

    if (!hasMore) {
      const sentinel = document.getElementById('scroll-sentinel');
      if (sentinel) {
        sentinel.innerHTML = '<div class="text-center text-tertiary py-4 text-sm">No more images</div>';
      }
    }
  } catch {
    // Silently fail on load more
  } finally {
    isLoading = false;
  }
}

async function fetchAndRenderImages(query: string, filters: ImageSearchFilters): Promise<void> {
  const content = document.getElementById('images-content');
  if (!content || !query) return;

  isLoading = true;

  try {
    const response = await api.searchImages(query, { ...filters, page: 1, per_page: 30 });
    const results = response.results as ImageResult[];

    hasMore = response.has_more;
    allImages = results;

    if (results.length === 0) {
      content.innerHTML = `
        <div class="py-8 text-secondary">No image results found for "<strong>${escapeHtml(query)}</strong>"</div>
      `;
      return;
    }

    const activeFilters = Object.entries(filters)
      .filter(([_, v]) => v && v !== 'any')
      .map(([k, v]) => `${k}: ${v}`)
      .join(', ');

    content.innerHTML = `
      <div class="text-xs text-tertiary mb-4">
        About ${response.total_results.toLocaleString()} image results (${(response.search_time_ms / 1000).toFixed(2)} seconds)
        ${activeFilters ? `<span class="ml-2 text-blue">Filters: ${escapeHtml(activeFilters)}</span>` : ''}
      </div>
      <div class="image-grid">
        ${results.map((img, i) => renderImageCard(img, i)).join('')}
      </div>
    `;

    // Attach click handlers to image cards
    content.querySelectorAll('.image-card').forEach((card) => {
      card.setAttribute('data-initialized', 'true');
      card.addEventListener('click', () => {
        const idx = parseInt((card as HTMLElement).dataset.imageIndex || '0', 10);
        openPreview(allImages[idx]);
      });
    });
  } catch (err) {
    content.innerHTML = `
      <div class="py-8">
        <p class="text-red text-sm">Failed to load image results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${escapeHtml(String(err))}</p>
      </div>
    `;
  } finally {
    isLoading = false;
  }
}

function renderImageCard(img: ImageResult, index: number): string {
  return `
    <div class="image-card" data-image-index="${index}" data-full-url="${escapeAttr(img.url)}" data-source-url="${escapeAttr(img.source_url)}">
      <img
        src="${escapeAttr(img.thumbnail_url || img.url)}"
        alt="${escapeAttr(img.title)}"
        loading="lazy"
        onerror="this.parentElement.style.display='none'"
      />
      <div class="image-info">
        <div class="image-title">${escapeHtml(img.title || '')}</div>
        <div class="image-source">${escapeHtml(img.source_domain)}</div>
        ${img.width && img.height ? `<div class="image-dimensions">${img.width} x ${img.height}</div>` : ''}
      </div>
    </div>
  `;
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
