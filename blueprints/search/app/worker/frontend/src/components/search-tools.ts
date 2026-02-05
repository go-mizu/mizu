const ICON_CHEVRON_DOWN = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>`;
const ICON_CLOCK = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>`;
const ICON_GLOBE = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`;
const ICON_QUOTE = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V21c0 1 0 1 1 1z"/><path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3c0 1 0 1 1 1z"/></svg>`;
const ICON_LINK = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>`;
const ICON_CLOSE = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>`;

export const TIME_RANGES = [
  { value: '', label: 'Any time' },
  { value: 'hour', label: 'Past hour' },
  { value: 'day', label: 'Past 24 hours' },
  { value: 'week', label: 'Past week' },
  { value: 'month', label: 'Past month' },
  { value: 'year', label: 'Past year' },
];

export const REGIONS = [
  { value: '', label: 'Any region' },
  { value: 'us', label: 'United States' },
  { value: 'gb', label: 'United Kingdom' },
  { value: 'ca', label: 'Canada' },
  { value: 'au', label: 'Australia' },
  { value: 'de', label: 'Germany' },
  { value: 'fr', label: 'France' },
  { value: 'jp', label: 'Japan' },
  { value: 'in', label: 'India' },
  { value: 'br', label: 'Brazil' },
];

export interface SearchFilters {
  timeRange?: string;
  region?: string;
  verbatim?: boolean;
  site?: string;
}

export function renderSearchTools(currentFilters: SearchFilters = {}): string {
  const { timeRange = '', region = '', verbatim = false, site = '' } = currentFilters;

  const activeTimeLabel = TIME_RANGES.find((t) => t.value === timeRange)?.label || 'Any time';
  const activeRegionLabel = REGIONS.find((r) => r.value === region)?.label || 'Any region';

  const hasTimeFilter = timeRange !== '';
  const hasRegionFilter = region !== '';
  const hasSiteFilter = site !== '';
  const hasAnyFilter = hasTimeFilter || hasRegionFilter || verbatim || hasSiteFilter;

  return `
    <div class="search-tools" id="search-tools">
      <div class="search-tools-row">
        <!-- Time Filter -->
        <div class="search-tool-dropdown" data-tool="time">
          <button class="search-tool-btn ${hasTimeFilter ? 'active' : ''}" type="button">
            ${ICON_CLOCK}
            <span class="search-tool-label">${escapeHtml(activeTimeLabel)}</span>
            ${ICON_CHEVRON_DOWN}
          </button>
          <div class="search-tool-menu hidden">
            ${TIME_RANGES.map(t => `
              <button class="search-tool-option ${t.value === timeRange ? 'selected' : ''}" data-value="${t.value}">
                ${escapeHtml(t.label)}
              </button>
            `).join('')}
          </div>
        </div>

        <!-- Region Filter -->
        <div class="search-tool-dropdown" data-tool="region">
          <button class="search-tool-btn ${hasRegionFilter ? 'active' : ''}" type="button">
            ${ICON_GLOBE}
            <span class="search-tool-label">${escapeHtml(activeRegionLabel)}</span>
            ${ICON_CHEVRON_DOWN}
          </button>
          <div class="search-tool-menu hidden">
            ${REGIONS.map(r => `
              <button class="search-tool-option ${r.value === region ? 'selected' : ''}" data-value="${r.value}">
                ${escapeHtml(r.label)}
              </button>
            `).join('')}
          </div>
        </div>

        <!-- Verbatim Toggle -->
        <button class="search-tool-toggle ${verbatim ? 'active' : ''}" data-tool="verbatim" type="button">
          ${ICON_QUOTE}
          <span>Verbatim</span>
        </button>

        <!-- Site Search -->
        <div class="search-tool-site" data-tool="site">
          <div class="search-tool-site-input ${hasSiteFilter ? 'has-value' : ''}">
            ${ICON_LINK}
            <input
              type="text"
              id="site-filter-input"
              placeholder="Filter by site..."
              value="${escapeHtml(site)}"
              autocomplete="off"
              spellcheck="false"
            />
            ${hasSiteFilter ? `
              <button class="search-tool-site-clear" type="button" aria-label="Clear site filter">
                ${ICON_CLOSE}
              </button>
            ` : ''}
          </div>
        </div>

        <!-- Clear All Filters -->
        ${hasAnyFilter ? `
          <button class="search-tool-clear" id="clear-all-filters" type="button">
            ${ICON_CLOSE}
            <span>Clear filters</span>
          </button>
        ` : ''}
      </div>
    </div>
  `;
}

export function initSearchTools(onFilterChange: (filters: SearchFilters) => void): void {
  const container = document.getElementById('search-tools');
  if (!container) return;

  // Current filter state
  const state: SearchFilters = {
    timeRange: '',
    region: '',
    verbatim: false,
    site: '',
  };

  // Initialize state from current DOM values
  const timeDropdown = container.querySelector('[data-tool="time"]');
  const regionDropdown = container.querySelector('[data-tool="region"]');
  const verbatimToggle = container.querySelector('[data-tool="verbatim"]');
  const siteInput = container.querySelector('#site-filter-input') as HTMLInputElement;

  if (timeDropdown) {
    const selectedOption = timeDropdown.querySelector('.search-tool-option.selected') as HTMLElement;
    state.timeRange = selectedOption?.dataset.value || '';
  }

  if (regionDropdown) {
    const selectedOption = regionDropdown.querySelector('.search-tool-option.selected') as HTMLElement;
    state.region = selectedOption?.dataset.value || '';
  }

  if (verbatimToggle?.classList.contains('active')) {
    state.verbatim = true;
  }

  if (siteInput) {
    state.site = siteInput.value;
  }

  // Handle dropdown clicks
  container.querySelectorAll('.search-tool-dropdown').forEach((dropdown) => {
    const btn = dropdown.querySelector('.search-tool-btn');
    const menu = dropdown.querySelector('.search-tool-menu');
    const tool = (dropdown as HTMLElement).dataset.tool;

    btn?.addEventListener('click', (e) => {
      e.stopPropagation();

      // Close other dropdowns
      container.querySelectorAll('.search-tool-menu').forEach((m) => {
        if (m !== menu) m.classList.add('hidden');
      });

      menu?.classList.toggle('hidden');
    });

    menu?.querySelectorAll('.search-tool-option').forEach((option) => {
      option.addEventListener('click', () => {
        const value = (option as HTMLElement).dataset.value || '';

        // Update selection state
        menu.querySelectorAll('.search-tool-option').forEach((o) => {
          o.classList.toggle('selected', o === option);
        });

        // Update button state
        const isActive = value !== '';
        btn?.classList.toggle('active', isActive);

        // Update label
        const label = btn?.querySelector('.search-tool-label');
        if (label) {
          if (tool === 'time') {
            label.textContent = TIME_RANGES.find((t) => t.value === value)?.label || 'Any time';
            state.timeRange = value;
          } else if (tool === 'region') {
            label.textContent = REGIONS.find((r) => r.value === value)?.label || 'Any region';
            state.region = value;
          }
        }

        menu?.classList.add('hidden');
        onFilterChange({ ...state });
      });
    });
  });

  // Handle verbatim toggle
  verbatimToggle?.addEventListener('click', () => {
    state.verbatim = !state.verbatim;
    verbatimToggle.classList.toggle('active', state.verbatim);
    onFilterChange({ ...state });
  });

  // Handle site input
  let siteDebounceTimer: ReturnType<typeof setTimeout>;

  siteInput?.addEventListener('input', () => {
    clearTimeout(siteDebounceTimer);
    siteDebounceTimer = setTimeout(() => {
      state.site = siteInput.value.trim();
      updateSiteInputState(container, state.site);
      onFilterChange({ ...state });
    }, 500);
  });

  siteInput?.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      clearTimeout(siteDebounceTimer);
      state.site = siteInput.value.trim();
      updateSiteInputState(container, state.site);
      onFilterChange({ ...state });
    }
  });

  // Handle site clear button
  container.querySelector('.search-tool-site-clear')?.addEventListener('click', () => {
    state.site = '';
    if (siteInput) siteInput.value = '';
    updateSiteInputState(container, '');
    onFilterChange({ ...state });
  });

  // Handle clear all filters
  container.querySelector('#clear-all-filters')?.addEventListener('click', () => {
    state.timeRange = '';
    state.region = '';
    state.verbatim = false;
    state.site = '';
    onFilterChange({ ...state });
  });

  // Close dropdowns on outside click
  document.addEventListener('click', () => {
    container.querySelectorAll('.search-tool-menu').forEach((m) => m.classList.add('hidden'));
  });
}

function updateSiteInputState(container: Element, value: string): void {
  const siteWrapper = container.querySelector('.search-tool-site-input');
  siteWrapper?.classList.toggle('has-value', value !== '');
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
