export type ToastType = 'success' | 'error' | 'info' | 'warning';

interface ToastOptions {
  type?: ToastType;
  duration?: number; // ms, 0 = no auto-dismiss
}

const ICONS = {
  success: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>`,
  error: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/></svg>`,
  info: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/></svg>`,
  warning: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><path d="M12 9v4"/><path d="M12 17h.01"/></svg>`,
};

let toastContainer: HTMLElement | null = null;

function getContainer(): HTMLElement {
  if (!toastContainer) {
    toastContainer = document.createElement('div');
    toastContainer.className = 'toast-container';
    document.body.appendChild(toastContainer);
  }
  return toastContainer;
}

export function showToast(message: string, options: ToastOptions = {}): void {
  const { type = 'info', duration = 4000 } = options;
  const container = getContainer();

  const toast = document.createElement('div');
  toast.className = `toast toast-${type}`;
  toast.innerHTML = `
    <span class="toast-icon">${ICONS[type]}</span>
    <span class="toast-message">${escapeHtml(message)}</span>
    <button class="toast-close" aria-label="Close">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
    </button>
    ${duration > 0 ? '<div class="toast-progress"></div>' : ''}
  `;

  // Close button handler
  toast.querySelector('.toast-close')?.addEventListener('click', () => {
    dismissToast(toast);
  });

  container.appendChild(toast);

  // Trigger animation
  requestAnimationFrame(() => {
    toast.classList.add('toast-visible');
  });

  // Auto-dismiss
  if (duration > 0) {
    const progress = toast.querySelector('.toast-progress') as HTMLElement;
    if (progress) {
      progress.style.animationDuration = `${duration}ms`;
    }
    setTimeout(() => dismissToast(toast), duration);
  }
}

function dismissToast(toast: HTMLElement): void {
  toast.classList.remove('toast-visible');
  toast.classList.add('toast-hiding');
  setTimeout(() => {
    toast.remove();
  }, 300);
}

// Convenience functions
export const toast = {
  success: (message: string, duration?: number) => showToast(message, { type: 'success', duration }),
  error: (message: string, duration?: number) => showToast(message, { type: 'error', duration }),
  info: (message: string, duration?: number) => showToast(message, { type: 'info', duration }),
  warning: (message: string, duration?: number) => showToast(message, { type: 'warning', duration }),
};

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
