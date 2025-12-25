// Messaging App JavaScript

// Available themes
// 'default' themes (dark/light) use CSS variables only
// 'aim1.0' uses a completely different view directory
const THEMES = ['dark', 'light', 'aim1.0'];
const VIEW_THEMES = ['aim1.0']; // Themes that require different server-side views

// Theme handling - set data-theme attribute on page load
(function() {
    const theme = localStorage.getItem('theme') || 'dark';
    document.documentElement.setAttribute('data-theme', theme);
})();

// Get current theme
function getTheme() {
    return localStorage.getItem('theme') || 'dark';
}

// Set cookie for server-side theme detection
function setThemeCookie(themeName) {
    // Map dark/light to 'default' for server-side, aim1.0 stays as is
    const serverTheme = VIEW_THEMES.includes(themeName) ? themeName : 'default';
    document.cookie = `theme=${serverTheme}; path=/; max-age=31536000; SameSite=Lax`;
}

// Set theme by name
function setTheme(themeName, reload = true) {
    if (!THEMES.includes(themeName)) {
        themeName = 'dark';
    }

    const currentTheme = getTheme();
    const currentIsViewTheme = VIEW_THEMES.includes(currentTheme);
    const newIsViewTheme = VIEW_THEMES.includes(themeName);

    // Store in localStorage and cookie
    document.documentElement.setAttribute('data-theme', themeName);
    localStorage.setItem('theme', themeName);
    setThemeCookie(themeName);

    // Update theme selector if on settings page
    const themeSelect = document.getElementById('theme-select');
    if (themeSelect) {
        themeSelect.value = themeName;
    }

    // Update dark mode toggle for backwards compatibility
    const darkModeToggle = document.getElementById('dark-mode');
    if (darkModeToggle) {
        darkModeToggle.checked = themeName === 'dark';
    }

    // If switching between view themes (e.g., default <-> aim1.0), reload page
    if (reload && (currentIsViewTheme !== newIsViewTheme || (currentIsViewTheme && newIsViewTheme && currentTheme !== themeName))) {
        window.location.reload();
    }
}

// Toggle between dark/light (legacy function for backwards compatibility)
function toggleTheme() {
    const current = getTheme();
    // If using a special theme, toggle between dark and that theme
    if (current === 'dark') {
        setTheme('light');
    } else if (current === 'light') {
        setTheme('dark');
    } else {
        // For special themes like aim1.0, toggle to dark
        setTheme('dark');
    }
}

// Initialize theme cookie on page load
(function() {
    const theme = getTheme();
    setThemeCookie(theme);
})();

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    // Ctrl/Cmd + K: Focus search
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        document.querySelector('#search-input')?.focus();
    }
    // Escape: Close modals
    if (e.key === 'Escape') {
        document.querySelectorAll('.modal').forEach(m => m.classList.add('hidden'));
    }
});
