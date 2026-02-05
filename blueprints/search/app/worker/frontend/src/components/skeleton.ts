/**
 * Skeleton loading component for smooth loading states
 */

export function renderResultSkeleton(count: number = 5): string {
  return Array(count).fill(0).map(() => `
    <div class="skeleton-result">
      <div class="skeleton-line skeleton-url"></div>
      <div class="skeleton-line skeleton-title"></div>
      <div class="skeleton-line skeleton-snippet"></div>
      <div class="skeleton-line skeleton-snippet short"></div>
    </div>
  `).join('');
}

export function renderImageGridSkeleton(count: number = 20): string {
  // Create varied heights for masonry effect
  const heights = ['150px', '200px', '180px', '220px', '170px', '190px'];
  return `
    <div class="image-grid">
      ${Array(count).fill(0).map((_, i) => `
        <div class="skeleton-image" style="height: ${heights[i % heights.length]}"></div>
      `).join('')}
    </div>
  `;
}

export function renderVideoGridSkeleton(count: number = 8): string {
  return `
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
      ${Array(count).fill(0).map(() => `
        <div class="skeleton-video">
          <div class="skeleton-video-thumb"></div>
          <div class="skeleton-video-info">
            <div class="skeleton-line skeleton-title"></div>
            <div class="skeleton-line skeleton-meta"></div>
          </div>
        </div>
      `).join('')}
    </div>
  `;
}

export function renderNewsListSkeleton(count: number = 6): string {
  return Array(count).fill(0).map(() => `
    <div class="skeleton-news">
      <div class="skeleton-news-content">
        <div class="skeleton-line skeleton-source"></div>
        <div class="skeleton-line skeleton-title"></div>
        <div class="skeleton-line skeleton-snippet"></div>
      </div>
      <div class="skeleton-news-image"></div>
    </div>
  `).join('');
}

export function renderCardSkeleton(): string {
  return `
    <div class="skeleton-card">
      <div class="skeleton-line skeleton-badge"></div>
      <div class="skeleton-line skeleton-title"></div>
      <div class="skeleton-line skeleton-snippet"></div>
      <div class="skeleton-line skeleton-snippet short"></div>
      <div class="skeleton-line skeleton-meta"></div>
    </div>
  `;
}

export function renderListSkeleton(count: number = 5): string {
  return Array(count).fill(0).map(() => renderCardSkeleton()).join('');
}
