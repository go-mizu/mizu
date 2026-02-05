const ICON_CHEVRON_DOWN = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>`;
const ICON_CLOSE = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>`;

export interface FilterOption {
  value: string;
  label: string;
}

export interface FilterConfig {
  id: string;
  label: string;
  options: FilterOption[];
  defaultValue?: string;
}

export interface FilterState {
  [filterId: string]: string;
}

export function renderFilterPills(filters: FilterConfig[], state: FilterState = {}): string {
  const hasActiveFilters = Object.values(state).some(v => v && v !== 'any');

  return `
    <div class="filter-pills-container">
      <div class="filter-pills-list">
        ${filters.map(filter => {
          const currentValue = state[filter.id] || filter.defaultValue || 'any';
          const currentLabel = filter.options.find(o => o.value === currentValue)?.label || filter.label;
          const isActive = currentValue !== 'any' && currentValue !== filter.defaultValue;

          return `
            <div class="filter-pill-wrapper" data-filter-id="${filter.id}">
              <button class="filter-pill ${isActive ? 'active' : ''}" data-current="${currentValue}">
                <span class="filter-pill-label">${isActive ? escapeHtml(currentLabel) : escapeHtml(filter.label)}</span>
                ${ICON_CHEVRON_DOWN}
              </button>
              <div class="filter-pill-dropdown hidden">
                ${filter.options.map(opt => `
                  <button class="filter-pill-option ${opt.value === currentValue ? 'selected' : ''}" data-value="${opt.value}">
                    ${escapeHtml(opt.label)}
                  </button>
                `).join('')}
              </div>
            </div>
          `;
        }).join('')}
        ${hasActiveFilters ? `
          <button class="filter-clear-btn">
            ${ICON_CLOSE}
            Clear all
          </button>
        ` : ''}
      </div>
    </div>
  `;
}

export function initFilterPills(onChange: (state: FilterState) => void): void {
  const container = document.querySelector('.filter-pills-container');
  if (!container) return;

  const state: FilterState = {};

  // Initialize state from current values
  container.querySelectorAll('.filter-pill-wrapper').forEach((wrapper) => {
    const filterId = (wrapper as HTMLElement).dataset.filterId!;
    const btn = wrapper.querySelector('.filter-pill') as HTMLElement;
    state[filterId] = btn.dataset.current || 'any';
  });

  // Handle pill clicks
  container.querySelectorAll('.filter-pill').forEach((pill) => {
    pill.addEventListener('click', (e) => {
      e.stopPropagation();
      const wrapper = pill.closest('.filter-pill-wrapper');
      const dropdown = wrapper?.querySelector('.filter-pill-dropdown');

      // Close other dropdowns
      container.querySelectorAll('.filter-pill-dropdown').forEach(d => {
        if (d !== dropdown) d.classList.add('hidden');
      });

      dropdown?.classList.toggle('hidden');
    });
  });

  // Handle option selection
  container.querySelectorAll('.filter-pill-option').forEach((option) => {
    option.addEventListener('click', () => {
      const wrapper = option.closest('.filter-pill-wrapper') as HTMLElement;
      const filterId = wrapper.dataset.filterId!;
      const value = (option as HTMLElement).dataset.value!;
      const dropdown = wrapper.querySelector('.filter-pill-dropdown');
      const pill = wrapper.querySelector('.filter-pill') as HTMLElement;

      state[filterId] = value;
      pill.dataset.current = value;

      // Update selected state
      wrapper.querySelectorAll('.filter-pill-option').forEach(o => {
        o.classList.toggle('selected', (o as HTMLElement).dataset.value === value);
      });

      // Update pill appearance
      const config = getFilterConfig(filterId);
      const isActive = value !== 'any' && value !== config?.defaultValue;
      pill.classList.toggle('active', isActive);

      const label = wrapper.querySelector('.filter-pill-label')!;
      label.textContent = isActive
        ? config?.options.find(o => o.value === value)?.label || value
        : config?.label || filterId;

      dropdown?.classList.add('hidden');
      onChange(state);
    });
  });

  // Handle clear all
  container.querySelector('.filter-clear-btn')?.addEventListener('click', () => {
    Object.keys(state).forEach(key => state[key] = 'any');

    container.querySelectorAll('.filter-pill-wrapper').forEach((wrapper) => {
      const pill = wrapper.querySelector('.filter-pill') as HTMLElement;
      const filterId = (wrapper as HTMLElement).dataset.filterId!;
      const config = getFilterConfig(filterId);

      pill.classList.remove('active');
      pill.dataset.current = 'any';
      (wrapper.querySelector('.filter-pill-label') as HTMLElement).textContent = config?.label || filterId;

      wrapper.querySelectorAll('.filter-pill-option').forEach((o, i) => {
        o.classList.toggle('selected', i === 0);
      });
    });

    onChange(state);
  });

  // Close dropdowns on outside click
  document.addEventListener('click', () => {
    container.querySelectorAll('.filter-pill-dropdown').forEach(d => d.classList.add('hidden'));
  });
}

// Helper to get filter config by ID - can be customized per page
let filterConfigs: FilterConfig[] = [];
export function setFilterConfigs(configs: FilterConfig[]): void {
  filterConfigs = configs;
}
function getFilterConfig(id: string): FilterConfig | undefined {
  return filterConfigs.find(f => f.id === id);
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
