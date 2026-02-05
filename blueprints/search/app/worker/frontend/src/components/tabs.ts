const ICON_SEARCH = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>`;
const ICON_IMAGE = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>`;
const ICON_VIDEO = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>`;
const ICON_NEWS = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/><path d="M18 14h-8"/><path d="M15 18h-5"/><path d="M10 6h8v4h-8V6Z"/></svg>`;
const ICON_SCIENCE = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 2v7.527a2 2 0 0 1-.211.896L4.72 20.55a1 1 0 0 0 .9 1.45h12.76a1 1 0 0 0 .9-1.45l-5.069-10.127A2 2 0 0 1 14 9.527V2"/><path d="M8.5 2h7"/><path d="M7 16h10"/></svg>`;
const ICON_CODE = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>`;
const ICON_MUSIC = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>`;
const ICON_SOCIAL = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`;
const ICON_MAPS = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>`;
const ICON_MORE = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>`;

interface TabsOptions {
  query: string;
  active: 'all' | 'images' | 'videos' | 'news' | 'science' | 'code' | 'music' | 'social' | 'maps';
}

export function renderTabs(options: TabsOptions): string {
  const { query, active } = options;
  const q = encodeURIComponent(query);

  const primaryTabs = [
    { id: 'all', label: 'All', icon: ICON_SEARCH, href: `/search?q=${q}` },
    { id: 'images', label: 'Images', icon: ICON_IMAGE, href: `/images?q=${q}` },
    { id: 'videos', label: 'Videos', icon: ICON_VIDEO, href: `/videos?q=${q}` },
    { id: 'news', label: 'News', icon: ICON_NEWS, href: `/news?q=${q}` },
  ];

  const moreTabs = [
    { id: 'science', label: 'Science', icon: ICON_SCIENCE, href: `/science?q=${q}` },
    { id: 'code', label: 'Code', icon: ICON_CODE, href: `/code?q=${q}` },
    { id: 'music', label: 'Music', icon: ICON_MUSIC, href: `/music?q=${q}` },
    { id: 'social', label: 'Social', icon: ICON_SOCIAL, href: `/social?q=${q}` },
    { id: 'maps', label: 'Maps', icon: ICON_MAPS, href: `/maps?q=${q}` },
  ];

  // Check if active tab is in moreTabs
  const activeMoreTab = moreTabs.find(t => t.id === active);

  return `
    <div class="search-tabs-container">
      <div class="search-tabs" id="search-tabs">
        ${primaryTabs.map(tab => `
          <a class="search-tab ${tab.id === active ? 'active' : ''}" href="${tab.href}" data-link data-tab="${tab.id}">
            ${tab.icon}
            <span>${tab.label}</span>
          </a>
        `).join('')}
        <div class="search-tab-more">
          <button class="search-tab ${activeMoreTab ? 'active' : ''}" id="more-tabs-btn" type="button">
            ${activeMoreTab ? activeMoreTab.icon : ICON_MORE}
            <span>${activeMoreTab ? activeMoreTab.label : 'More'}</span>
          </button>
          <div class="more-tabs-dropdown hidden" id="more-tabs-dropdown">
            ${moreTabs.map(tab => `
              <a class="more-tab-item ${tab.id === active ? 'active' : ''}" href="${tab.href}" data-link>
                ${tab.icon}
                <span>${tab.label}</span>
              </a>
            `).join('')}
          </div>
        </div>
      </div>
    </div>
  `;
}

export function initTabs(): void {
  const moreBtn = document.getElementById('more-tabs-btn');
  const dropdown = document.getElementById('more-tabs-dropdown');

  if (moreBtn && dropdown) {
    moreBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      dropdown.classList.toggle('hidden');
    });

    document.addEventListener('click', () => {
      dropdown.classList.add('hidden');
    });
  }
}
