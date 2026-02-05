const ICON_SEARCH = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>`;
const ICON_IMAGE = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>`;
const ICON_VIDEO = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>`;
const ICON_NEWS = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/><path d="M18 14h-8"/><path d="M15 18h-5"/><path d="M10 6h8v4h-8V6Z"/></svg>`;

interface TabsOptions {
  query: string;
  active: 'all' | 'images' | 'videos' | 'news';
}

export function renderTabs(options: TabsOptions): string {
  const { query, active } = options;
  const q = encodeURIComponent(query);

  const tabs = [
    { id: 'all', label: 'All', icon: ICON_SEARCH, href: `/search?q=${q}` },
    { id: 'images', label: 'Images', icon: ICON_IMAGE, href: `/images?q=${q}` },
    { id: 'videos', label: 'Videos', icon: ICON_VIDEO, href: `/videos?q=${q}` },
    { id: 'news', label: 'News', icon: ICON_NEWS, href: `/news?q=${q}` },
  ];

  return `
    <div class="search-tabs" id="search-tabs">
      ${tabs
        .map(
          (tab) => `
        <a class="search-tab ${tab.id === active ? 'active' : ''}" href="${tab.href}" data-link data-tab="${tab.id}">
          ${tab.icon}
          <span>${tab.label}</span>
        </a>
      `
        )
        .join('')}
    </div>
  `;
}

export function initTabs(): void {
  // Tabs use data-link attributes, so the router handles navigation.
  // No additional initialization needed.
}
