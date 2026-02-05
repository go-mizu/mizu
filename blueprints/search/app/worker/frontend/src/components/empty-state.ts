const ICON_SEARCH_OFF = `<svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/><path d="M8 8h6"/></svg>`;
const ICON_IMAGE_OFF = `<svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/><path d="M2 2l20 20"/></svg>`;
const ICON_VIDEO_OFF = `<svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/><path d="M2 2l20 20"/></svg>`;

export interface EmptyStateOptions {
  type?: 'search' | 'images' | 'videos' | 'news' | 'generic';
  query?: string;
  suggestions?: string[];
}

export function renderEmptyState(options: EmptyStateOptions = {}): string {
  const { type = 'search', query, suggestions = [] } = options;

  let icon = ICON_SEARCH_OFF;
  let title = 'No results found';
  let description = query
    ? `We couldn't find any results for "${escapeHtml(query)}"`
    : 'Try a different search term';

  switch (type) {
    case 'images':
      icon = ICON_IMAGE_OFF;
      title = 'No images found';
      break;
    case 'videos':
      icon = ICON_VIDEO_OFF;
      title = 'No videos found';
      break;
    case 'news':
      title = 'No news found';
      description = query
        ? `We couldn't find any news articles for "${escapeHtml(query)}"`
        : 'Try searching for a topic';
      break;
  }

  const suggestionsHtml = suggestions.length > 0 ? `
    <div class="empty-suggestions">
      <p>Try searching for:</p>
      <div class="empty-suggestion-chips">
        ${suggestions.slice(0, 4).map(s => `
          <a href="/search?q=${encodeURIComponent(s)}" data-link class="empty-suggestion-chip">${escapeHtml(s)}</a>
        `).join('')}
      </div>
    </div>
  ` : '';

  const tipsHtml = `
    <div class="empty-tips">
      <p>Suggestions:</p>
      <ul>
        <li>Check the spelling of your search terms</li>
        <li>Try more general keywords</li>
        <li>Try different keywords</li>
      </ul>
    </div>
  `;

  return `
    <div class="empty-state">
      <div class="empty-icon text-border">${icon}</div>
      <h3 class="empty-title">${title}</h3>
      <p class="empty-description">${description}</p>
      ${suggestionsHtml}
      ${tipsHtml}
    </div>
  `;
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
