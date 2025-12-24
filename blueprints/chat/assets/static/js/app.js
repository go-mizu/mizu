// Chat Application JavaScript

// Utility functions
const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => document.querySelectorAll(sel);

// API helper
async function api(method, path, body) {
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json' },
  };
  if (body) opts.body = JSON.stringify(body);

  const res = await fetch(`/api/v1${path}`, opts);
  const json = await res.json();

  if (!res.ok) {
    throw new Error(json.error?.message || 'Request failed');
  }

  return json.data;
}

// Time formatting
function formatTimestamp(iso) {
  const date = new Date(iso);
  const now = new Date();
  const diff = now - date;

  // Today
  if (diff < 86400000 && date.getDate() === now.getDate()) {
    return 'Today at ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  // Yesterday
  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  if (date.getDate() === yesterday.getDate()) {
    return 'Yesterday at ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  // Older
  return date.toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });
}

// Escape HTML
function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
  // Ctrl/Cmd + K: Quick switcher
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault();
    // TODO: Show quick switcher
  }

  // Escape: Close modals
  if (e.key === 'Escape') {
    // TODO: Close active modal
  }
});

// Auto-resize textareas
document.addEventListener('input', (e) => {
  if (e.target.tagName === 'TEXTAREA') {
    e.target.style.height = 'auto';
    e.target.style.height = e.target.scrollHeight + 'px';
  }
});

console.log('Chat app loaded');
