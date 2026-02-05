const ICON_ALERT = `<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M12 8v4"/><path d="M12 16h.01"/></svg>`;
const ICON_WIFI_OFF = `<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M1 1l22 22"/><path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55"/><path d="M5 12.55a10.94 10.94 0 0 1 5.17-2.39"/><path d="M10.71 5.05A16 16 0 0 1 22.58 9"/><path d="M1.42 9a15.91 15.91 0 0 1 4.7-2.88"/><path d="M8.53 16.11a6 6 0 0 1 6.95 0"/><circle cx="12" cy="20" r="1"/></svg>`;

export interface ErrorStateOptions {
  type?: 'network' | 'server' | 'not-found' | 'generic';
  message?: string;
  details?: string;
  onRetry?: () => void;
}

export function renderErrorState(options: ErrorStateOptions = {}): string {
  const { type = 'generic', message, details, onRetry } = options;

  let icon = ICON_ALERT;
  let title = message || 'Something went wrong';
  let description = details || 'Please try again later.';

  switch (type) {
    case 'network':
      icon = ICON_WIFI_OFF;
      title = message || 'No internet connection';
      description = details || 'Check your network connection and try again.';
      break;
    case 'server':
      title = message || 'Server error';
      description = details || 'Our servers are having issues. Please try again in a moment.';
      break;
    case 'not-found':
      title = message || 'Page not found';
      description = details || 'The page you\'re looking for doesn\'t exist.';
      break;
  }

  return `
    <div class="error-state">
      <div class="error-icon text-tertiary">${icon}</div>
      <h3 class="error-title">${escapeHtml(title)}</h3>
      <p class="error-description">${escapeHtml(description)}</p>
      ${onRetry ? '<button class="error-retry-btn" id="error-retry-btn">Try again</button>' : ''}
    </div>
  `;
}

export function initErrorState(onRetry?: () => void): void {
  const retryBtn = document.getElementById('error-retry-btn');
  if (retryBtn && onRetry) {
    retryBtn.addEventListener('click', onRetry);
  }
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
