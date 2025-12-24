// Mizu Chat - Minimal JavaScript
// Most rendering is done server-side. This file handles:
// - API helpers
// - Keyboard shortcuts
// - Small UI enhancements

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

// Toast notifications
function showToast(message, type = 'info') {
    let container = document.getElementById('toast-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'toast-container';
        container.className = 'toast-container';
        document.body.appendChild(container);
    }

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    container.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => toast.remove(), 200);
    }, 3000);
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    // Ctrl/Cmd + K: Focus search/quick switcher
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        const input = document.getElementById('message-input') || document.getElementById('search-input');
        if (input) input.focus();
    }

    // Escape: Close modals
    if (e.key === 'Escape') {
        const overlay = document.querySelector('.modal-overlay.open');
        if (overlay) {
            overlay.classList.remove('open');
        }
    }
});

// Auto-resize textareas
document.addEventListener('input', (e) => {
    if (e.target.tagName === 'TEXTAREA' && e.target.dataset.autosize !== 'false') {
        e.target.style.height = 'auto';
        e.target.style.height = Math.min(e.target.scrollHeight, 300) + 'px';
    }
});

// Smooth scroll behavior for messages
document.addEventListener('DOMContentLoaded', () => {
    const messages = document.getElementById('messages');
    if (messages) {
        // Check if we should auto-scroll
        const shouldAutoScroll = () => {
            return messages.scrollHeight - messages.scrollTop <= messages.clientHeight + 100;
        };

        // Store scroll state
        let autoScroll = true;
        messages.addEventListener('scroll', () => {
            autoScroll = shouldAutoScroll();
        });

        // Expose for use in templates
        window.shouldAutoScroll = () => autoScroll;
    }
});

// Theme toggle (for future use)
function setTheme(theme) {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem('theme', theme);
}

function getTheme() {
    return localStorage.getItem('theme') || 'light';
}

function toggleTheme() {
    const current = getTheme();
    const next = current === 'dark' ? 'light' : 'dark';
    setTheme(next);
}

// Apply saved theme on load
document.documentElement.dataset.theme = getTheme();

console.log('Mizu Chat loaded');
