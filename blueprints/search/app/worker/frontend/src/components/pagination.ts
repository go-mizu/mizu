const ICON_CHEVRON_LEFT = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>`;
const ICON_CHEVRON_RIGHT = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>`;

export interface PaginationOptions {
  currentPage: number;
  hasMore: boolean;
  totalResults: number;
  perPage: number;
}

export function renderPagination(options: PaginationOptions): string {
  const { currentPage, hasMore, totalResults, perPage } = options;
  const totalPages = Math.min(Math.ceil(totalResults / perPage), 100);

  if (totalPages <= 1) return '';

  // Compute window of page numbers
  let start = Math.max(1, currentPage - 4);
  let end = Math.min(totalPages, start + 9);
  if (end - start < 9) {
    start = Math.max(1, end - 9);
  }

  const pages: number[] = [];
  for (let i = start; i <= end; i++) {
    pages.push(i);
  }

  // Render the Google-style logo letters
  const logoLetters = renderPaginationLogo(currentPage);

  const prevDisabled = currentPage <= 1 ? 'disabled' : '';
  const nextDisabled = !hasMore && currentPage >= totalPages ? 'disabled' : '';

  return `
    <div class="pagination" id="pagination">
      <div class="flex flex-col items-center gap-3">
        ${logoLetters}
        <div class="flex items-center gap-1">
          <button class="pagination-btn ${prevDisabled}" data-page="${currentPage - 1}" ${currentPage <= 1 ? 'disabled' : ''} aria-label="Previous page">
            ${ICON_CHEVRON_LEFT}
          </button>
          ${pages
            .map(
              (p) => `
            <button class="pagination-btn ${p === currentPage ? 'active' : ''}" data-page="${p}">
              ${p}
            </button>
          `
            )
            .join('')}
          <button class="pagination-btn ${nextDisabled}" data-page="${currentPage + 1}" ${!hasMore && currentPage >= totalPages ? 'disabled' : ''} aria-label="Next page">
            ${ICON_CHEVRON_RIGHT}
          </button>
        </div>
      </div>
    </div>
  `;
}

function renderPaginationLogo(currentPage: number): string {
  // Google-style "Miiizuuu" with colored dots for page numbers
  const colors = ['#4285F4', '#EA4335', '#FBBC05', '#4285F4', '#34A853', '#EA4335'];
  const baseLetters = ['M', 'i', 'z', 'u'];

  // Add extra letters based on pages
  const extraCount = Math.min(currentPage - 1, 6);
  let letters = [baseLetters[0]];
  for (let i = 0; i < 1 + extraCount; i++) letters.push('i');
  letters.push('z');
  for (let i = 0; i < 1 + extraCount; i++) letters.push('u');

  return `
    <div class="flex items-center text-2xl font-semibold tracking-wide select-none">
      ${letters
        .map((letter, i) => {
          const color = colors[i % colors.length];
          return `<span style="color: ${color}">${letter}</span>`;
        })
        .join('')}
    </div>
  `;
}

export function initPagination(onPageChange: (page: number) => void): void {
  const container = document.getElementById('pagination');
  if (!container) return;

  container.querySelectorAll('.pagination-btn').forEach((btn) => {
    btn.addEventListener('click', () => {
      const page = parseInt((btn as HTMLElement).dataset.page || '1');
      if (isNaN(page) || (btn as HTMLButtonElement).disabled) return;
      onPageChange(page);
      // Scroll to top
      window.scrollTo({ top: 0, behavior: 'smooth' });
    });
  });
}
