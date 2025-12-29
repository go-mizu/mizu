// GitHome - Minimal JavaScript
// Only for essential client-side interactivity

(function() {
    'use strict';

    // Theme toggle (if implemented)
    function toggleTheme() {
        const html = document.documentElement;
        const currentTheme = html.getAttribute('data-theme') || 'light';
        const newTheme = currentTheme === 'light' ? 'dark' : 'light';
        html.setAttribute('data-theme', newTheme);
        localStorage.setItem('theme', newTheme);
    }

    // Load saved theme
    function loadTheme() {
        const savedTheme = localStorage.getItem('theme');
        if (savedTheme) {
            document.documentElement.setAttribute('data-theme', savedTheme);
        }
    }

    // Generic dropdown handler
    function setupDropdown(buttonId, menuId) {
        const button = document.getElementById(buttonId);
        const menu = document.getElementById(menuId);
        if (button && menu) {
            button.addEventListener('click', function(e) {
                e.stopPropagation();
                // Close all other dropdowns
                document.querySelectorAll('.dropdown-menu').forEach(function(m) {
                    if (m !== menu) m.style.display = 'none';
                });
                menu.style.display = menu.style.display === 'none' ? 'block' : 'none';
            });
        }
    }

    // Dropdown menus
    function initDropdowns() {
        // Legacy data-attribute based dropdowns
        document.querySelectorAll('[data-dropdown-toggle]').forEach(function(toggle) {
            toggle.addEventListener('click', function(e) {
                e.preventDefault();
                const target = document.querySelector(toggle.getAttribute('data-dropdown-toggle'));
                if (target) {
                    target.classList.toggle('open');
                }
            });
        });

        // New ID-based dropdowns
        setupDropdown('createNewBtn', 'createNewMenu');
        setupDropdown('addFileBtn', 'addFileMenu');
        setupDropdown('codeBtn', 'codeMenu');

        // Close dropdowns when clicking outside
        document.addEventListener('click', function(e) {
            if (!e.target.closest('.dropdown-menu') &&
                !e.target.closest('#createNewBtn') &&
                !e.target.closest('#addFileBtn') &&
                !e.target.closest('#codeBtn') &&
                !e.target.closest('[data-dropdown-toggle]')) {
                document.querySelectorAll('.dropdown-menu').forEach(function(menu) {
                    menu.style.display = 'none';
                    menu.classList.remove('open');
                });
            }
        });
    }

    // Flash message auto-dismiss
    function initFlashMessages() {
        document.querySelectorAll('.flash').forEach(function(flash) {
            setTimeout(function() {
                flash.style.opacity = '0';
                setTimeout(function() {
                    flash.remove();
                }, 300);
            }, 5000);
        });
    }

    // Confirm dialogs
    function initConfirmDialogs() {
        document.querySelectorAll('[data-confirm]').forEach(function(element) {
            element.addEventListener('click', function(e) {
                if (!confirm(element.getAttribute('data-confirm'))) {
                    e.preventDefault();
                }
            });
        });
    }

    // Global search keyboard shortcut
    function initSearchShortcut() {
        document.addEventListener('keydown', function(e) {
            // Focus search on '/' key when not in an input
            if (e.key === '/' && !e.target.matches('input, textarea, [contenteditable]')) {
                e.preventDefault();
                const searchInput = document.getElementById('globalSearch');
                if (searchInput) {
                    searchInput.focus();
                }
            }
        });
    }

    // Copy to clipboard helper
    window.copyToClipboard = function(inputId) {
        const input = document.getElementById(inputId);
        if (input) {
            input.select();
            input.setSelectionRange(0, 99999);
            navigator.clipboard.writeText(input.value).then(function() {
                // Could show a tooltip here
            });
        }
    };

    // Initialize
    document.addEventListener('DOMContentLoaded', function() {
        loadTheme();
        initDropdowns();
        initFlashMessages();
        initConfirmDialogs();
        initSearchShortcut();
    });
})();
