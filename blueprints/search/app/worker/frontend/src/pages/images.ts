import { Router } from '../lib/router';
import { api } from '../api';
import type { ImageResult, ImageSearchFilters } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;

const ICON_CAMERA = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>`;

const ICON_CLOSE = `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>`;

const ICON_EXTERNAL = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>`;

const ICON_FILTER = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>`;

const ICON_CHEVRON_DOWN = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>`;

const ICON_CHEVRON_LEFT = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>`;

const ICON_CHEVRON_RIGHT = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>`;

// Current state
let currentQuery = '';
let currentFilters: ImageSearchFilters = {};
let currentPage = 1;
let isLoading = false;
let hasMore = true;
let allImages: ImageResult[] = [];
let filtersVisible = false;
let relatedSearches: string[] = [];
let scrollObserver: IntersectionObserver | null = null;

export function renderImagesPage(query: string): string {
  return `
    <div class="min-h-screen flex flex-col bg-white">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 shadow-sm">
        <div class="flex items-center gap-4 px-4 py-2">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[600px] flex items-center gap-2">
            ${renderSearchBox({ size: 'sm', initialValue: query })}
            <button id="reverse-search-btn" class="flex-shrink-0 p-2 text-tertiary hover:text-primary hover:bg-surface-hover rounded-full transition-colors" title="Search by image">
              ${ICON_CAMERA}
            </button>
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${ICON_SETTINGS}
          </a>
        </div>
        <div class="pl-[56px] flex items-center gap-1">
          ${renderTabs({ query, active: 'images' })}
          <button id="tools-btn" class="tools-btn ml-4">
            ${ICON_FILTER}
            <span>Tools</span>
            ${ICON_CHEVRON_DOWN}
          </button>
        </div>
        <!-- Filter toolbar (hidden by default) -->
        <div id="filter-toolbar" class="filter-toolbar hidden">
          ${renderFilterToolbar()}
        </div>
      </header>

      <!-- Related searches bar -->
      <div id="related-searches" class="related-searches-bar hidden">
        <div class="related-searches-scroll">
          <button class="related-scroll-btn related-scroll-left hidden">${ICON_CHEVRON_LEFT}</button>
          <div class="related-searches-list"></div>
          <button class="related-scroll-btn related-scroll-right hidden">${ICON_CHEVRON_RIGHT}</button>
        </div>
      </div>

      <!-- Content -->
      <main class="flex-1 flex">
        <div id="images-content" class="flex-1 p-3">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>

        <!-- Preview panel (hidden by default) -->
        <div id="preview-panel" class="preview-panel hidden">
          <div class="preview-overlay"></div>
          <div class="preview-container">
            <button id="preview-close" class="preview-close-btn" aria-label="Close">${ICON_CLOSE}</button>
            <div class="preview-main">
              <div class="preview-image-wrap">
                <img id="preview-image" src="" alt="" />
              </div>
              <div class="preview-sidebar">
                <div id="preview-details" class="preview-info"></div>
              </div>
            </div>
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
    { id: 'rights', label: 'Usage rights', options: ['any', 'creative_commons', 'commercial'] },
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
              <button class="filter-option${opt === 'any' ? ' active' : ''}" data-value="${opt}">
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
  filtersVisible = false;
  relatedSearches = [];

  initSearchBox((q) => {
    router.navigate(`/images?q=${encodeURIComponent(q)}`);
  });

  initTabs();

  if (query) {
    addRecentSearch(query);
  }

  initToolsButton();
  initFilters(router);
  initReverseSearch(router);
  initPreviewPanel();
  initRelatedSearches(router);

  fetchAndRenderImages(query, currentFilters);
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

function initFilters(_router: Router): void {
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

      if (value === 'any') {
        delete (currentFilters as Record<string, string>)[filterId];
        chip.classList.remove('has-value');
        chip.querySelector('.filter-chip-label')!.textContent = formatFilterOption(filterId, 'any').replace('Any ', '');
      } else {
        (currentFilters as Record<string, string>)[filterId] = value;
        chip.classList.add('has-value');
        chip.querySelector('.filter-chip-label')!.textContent = formatFilterOption(filterId, value);
      }

      dropdown.classList.add('hidden');
      updateClearButton();

      currentPage = 1;
      allImages = [];
      hasMore = true;
      fetchAndRenderImages(currentQuery, currentFilters);
    });
  });

  document.addEventListener('click', () => {
    toolbar.querySelectorAll('.filter-dropdown').forEach(d => d.classList.add('hidden'));
  });

  const clearBtn = document.getElementById('clear-filters');
  if (clearBtn) {
    clearBtn.addEventListener('click', () => {
      currentFilters = {};
      currentPage = 1;
      allImages = [];
      hasMore = true;

      toolbar.querySelectorAll('.filter-chip').forEach(chip => {
        const filterId = (chip as HTMLElement).dataset.filter;
        chip.classList.remove('has-value');
        (chip.querySelector('.filter-chip-label') as HTMLElement).textContent =
          formatFilterOption(filterId!, 'any').replace('Any ', '');
      });

      toolbar.querySelectorAll('.filter-dropdown').forEach(dropdown => {
        dropdown.querySelectorAll('.filter-option').forEach((opt, i) => {
          opt.classList.toggle('active', i === 0);
        });
      });

      updateClearButton();
      fetchAndRenderImages(currentQuery, currentFilters);
    });
  }
}

function updateClearButton(): void {
  const clearBtn = document.getElementById('clear-filters');
  if (!clearBtn) return;
  clearBtn.classList.toggle('hidden', Object.keys(currentFilters).length === 0);
}

function initRelatedSearches(router: Router): void {
  const container = document.getElementById('related-searches');
  if (!container) return;

  container.addEventListener('click', (e) => {
    const chip = (e.target as HTMLElement).closest('.related-chip');
    if (chip) {
      const query = chip.getAttribute('data-query');
      if (query) {
        router.navigate(`/images?q=${encodeURIComponent(query)}`);
      }
    }
  });

  // Scroll buttons
  const leftBtn = container.querySelector('.related-scroll-left');
  const rightBtn = container.querySelector('.related-scroll-right');
  const list = container.querySelector('.related-searches-list');

  if (leftBtn && rightBtn && list) {
    leftBtn.addEventListener('click', () => {
      list.scrollBy({ left: -200, behavior: 'smooth' });
    });
    rightBtn.addEventListener('click', () => {
      list.scrollBy({ left: 200, behavior: 'smooth' });
    });

    list.addEventListener('scroll', () => {
      updateScrollButtons();
    });
  }
}

function updateScrollButtons(): void {
  const container = document.getElementById('related-searches');
  if (!container) return;

  const list = container.querySelector('.related-searches-list') as HTMLElement;
  const leftBtn = container.querySelector('.related-scroll-left');
  const rightBtn = container.querySelector('.related-scroll-right');

  if (!list || !leftBtn || !rightBtn) return;

  leftBtn.classList.toggle('hidden', list.scrollLeft <= 0);
  rightBtn.classList.toggle('hidden', list.scrollLeft >= list.scrollWidth - list.clientWidth - 10);
}

function renderRelatedSearches(searches: string[]): void {
  const container = document.getElementById('related-searches');
  if (!container) return;

  if (!searches || searches.length === 0) {
    container.classList.add('hidden');
    return;
  }

  const list = container.querySelector('.related-searches-list');
  if (!list) return;

  list.innerHTML = searches.map(s => `
    <button class="related-chip" data-query="${escapeAttr(s)}">
      <span class="related-chip-text">${escapeHtml(s)}</span>
    </button>
  `).join('');

  container.classList.remove('hidden');

  // Update scroll buttons after render
  setTimeout(updateScrollButtons, 50);
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

  btn.addEventListener('click', () => modal.classList.remove('hidden'));
  closeBtn?.addEventListener('click', () => modal.classList.add('hidden'));
  modal.addEventListener('click', (e) => {
    if (e.target === modal) modal.classList.add('hidden');
  });

  if (dropZone) {
    dropZone.addEventListener('dragover', (e) => {
      e.preventDefault();
      dropZone.classList.add('drag-over');
    });
    dropZone.addEventListener('dragleave', () => dropZone.classList.remove('drag-over'));
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
        ` : `<div class="py-8 text-secondary">No similar images found.</div>`}
      </div>
    `;

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
  const panel = document.getElementById('preview-panel');
  const closeBtn = document.getElementById('preview-close');
  const overlay = panel?.querySelector('.preview-overlay');

  closeBtn?.addEventListener('click', closePreview);
  overlay?.addEventListener('click', closePreview);

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closePreview();
  });
}

function openPreview(image: ImageResult): void {
  const panel = document.getElementById('preview-panel');
  const imgEl = document.getElementById('preview-image') as HTMLImageElement;
  const details = document.getElementById('preview-details');

  if (!panel || !imgEl || !details) return;

  imgEl.src = image.url;
  imgEl.alt = image.title;

  const hasDimensions = image.width && image.height && image.width > 0 && image.height > 0;

  details.innerHTML = `
    <div class="preview-header">
      <img src="${escapeAttr(image.thumbnail_url || image.url)}" class="preview-thumb" alt="" />
      <div class="preview-header-info">
        <h3 class="preview-title">${escapeHtml(image.title || 'Untitled')}</h3>
        <a href="${escapeAttr(image.source_url)}" target="_blank" class="preview-domain">${escapeHtml(image.source_domain)}</a>
      </div>
    </div>
    <div class="preview-meta">
      ${hasDimensions ? `<div class="preview-meta-item"><span class="preview-meta-label">Size</span><span>${image.width} × ${image.height}</span></div>` : ''}
      ${image.format ? `<div class="preview-meta-item"><span class="preview-meta-label">Type</span><span>${image.format.toUpperCase()}</span></div>` : ''}
    </div>
    <div class="preview-actions">
      <a href="${escapeAttr(image.source_url)}" target="_blank" class="preview-btn preview-btn-primary">
        Visit page ${ICON_EXTERNAL}
      </a>
      <a href="${escapeAttr(image.url)}" target="_blank" class="preview-btn">
        View full image ${ICON_EXTERNAL}
      </a>
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

function setupInfiniteScroll(): void {
  // Clean up existing observer
  if (scrollObserver) {
    scrollObserver.disconnect();
  }

  const content = document.getElementById('images-content');
  if (!content) return;

  // Remove existing sentinel
  const existingSentinel = document.getElementById('scroll-sentinel');
  if (existingSentinel) existingSentinel.remove();

  // Create sentinel
  const sentinel = document.createElement('div');
  sentinel.id = 'scroll-sentinel';
  sentinel.className = 'scroll-sentinel';
  content.appendChild(sentinel);

  // Create observer
  scrollObserver = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting && !isLoading && hasMore && currentQuery) {
      loadMoreImages();
    }
  }, { rootMargin: '400px' });

  scrollObserver.observe(sentinel);
}

async function loadMoreImages(): Promise<void> {
  if (isLoading || !hasMore) return;

  isLoading = true;
  currentPage++;

  const sentinel = document.getElementById('scroll-sentinel');
  if (sentinel) {
    sentinel.innerHTML = '<div class="loading-more"><div class="spinner-sm"></div></div>';
  }

  try {
    const response = await api.searchImages(currentQuery, { ...currentFilters, page: currentPage });
    const newImages = response.results as ImageResult[];

    hasMore = response.has_more;
    allImages = [...allImages, ...newImages];

    const grid = document.querySelector('.image-grid');
    if (grid && newImages.length > 0) {
      const startIdx = allImages.length - newImages.length;
      const html = newImages.map((img, i) => renderImageCard(img, startIdx + i)).join('');
      grid.insertAdjacentHTML('beforeend', html);

      grid.querySelectorAll('.image-card:not([data-initialized])').forEach((card) => {
        card.setAttribute('data-initialized', 'true');
        card.addEventListener('click', () => {
          const idx = parseInt((card as HTMLElement).dataset.imageIndex || '0', 10);
          openPreview(allImages[idx]);
        });
      });
    }

    if (sentinel) {
      sentinel.innerHTML = hasMore ? '' : '<div class="no-more-results">No more images</div>';
    }
  } catch {
    if (sentinel) sentinel.innerHTML = '';
  } finally {
    isLoading = false;
  }
}

async function fetchAndRenderImages(query: string, filters: ImageSearchFilters): Promise<void> {
  const content = document.getElementById('images-content');
  if (!content || !query) return;

  isLoading = true;
  content.innerHTML = '<div class="flex items-center justify-center py-16"><div class="spinner"></div></div>';

  try {
    const response = await api.searchImages(query, { ...filters, page: 1, per_page: 50 });
    const results = response.results as ImageResult[];

    hasMore = response.has_more;
    allImages = results;

    // Use API related searches or generate fallback
    relatedSearches = response.related_searches?.length
      ? response.related_searches
      : generateRelatedSearches(query);

    // Render related searches
    renderRelatedSearches(relatedSearches);

    if (results.length === 0) {
      content.innerHTML = `<div class="py-8 text-secondary">No image results found for "<strong>${escapeHtml(query)}</strong>"</div>`;
      return;
    }

    content.innerHTML = `<div class="image-grid">${results.map((img, i) => renderImageCard(img, i)).join('')}</div>`;

    content.querySelectorAll('.image-card').forEach((card) => {
      card.setAttribute('data-initialized', 'true');
      card.addEventListener('click', () => {
        const idx = parseInt((card as HTMLElement).dataset.imageIndex || '0', 10);
        openPreview(allImages[idx]);
      });
    });

    // Setup infinite scroll after content is rendered
    setupInfiniteScroll();
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
    <div class="image-card" data-image-index="${index}">
      <div class="image-card-img">
        <img
          src="${escapeAttr(img.thumbnail_url || img.url)}"
          alt="${escapeAttr(img.title)}"
          loading="lazy"
          onerror="this.closest('.image-card').style.display='none'"
        />
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

/**
 * Generate related searches based on the current query.
 * This provides fallback suggestions when the API doesn't return any.
 */
function generateRelatedSearches(query: string): string[] {
  const words = query.toLowerCase().trim().split(/\s+/).filter(w => w.length > 1);
  if (words.length === 0) return [];

  const suggestions: string[] = [];

  // Common modifiers for image searches
  const modifiers = [
    'wallpaper', 'hd', '4k', 'aesthetic', 'cute', 'beautiful',
    'background', 'art', 'photography', 'design', 'illustration',
    'vintage', 'modern', 'minimalist', 'colorful', 'dark', 'light'
  ];

  // Category-specific suggestions
  const categories: Record<string, string[]> = {
    'cat': ['kitten', 'cats playing', 'black cat', 'tabby cat', 'cat meme'],
    'dog': ['puppy', 'dogs playing', 'golden retriever', 'german shepherd', 'dog meme'],
    'nature': ['forest', 'mountains', 'ocean', 'sunset nature', 'flowers'],
    'food': ['dessert', 'healthy food', 'breakfast', 'dinner', 'food photography'],
    'car': ['sports car', 'luxury car', 'vintage car', 'car interior', 'supercar'],
    'house': ['modern house', 'interior design', 'living room', 'bedroom design', 'architecture'],
    'city': ['skyline', 'night city', 'urban photography', 'street photography', 'downtown'],
  };

  // Add query + modifier combinations
  const queryBase = words.slice(0, 2).join(' ');
  for (const mod of modifiers) {
    if (!query.includes(mod) && suggestions.length < 4) {
      suggestions.push(`${queryBase} ${mod}`);
    }
  }

  // Add category-specific suggestions if query matches
  for (const [key, related] of Object.entries(categories)) {
    if (words.some(w => w.includes(key) || key.includes(w))) {
      for (const r of related) {
        if (!suggestions.includes(r) && suggestions.length < 8) {
          suggestions.push(r);
        }
      }
      break;
    }
  }

  // Add variations with different word orders
  if (words.length >= 2 && suggestions.length < 8) {
    suggestions.push(words.reverse().join(' '));
  }

  // Ensure we have at least some suggestions
  if (suggestions.length < 4) {
    suggestions.push(`${queryBase} images`, `${queryBase} photos`, `best ${queryBase}`);
  }

  return suggestions.slice(0, 8);
}
