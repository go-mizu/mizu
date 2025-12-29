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

    // Dropdown menus
    function initDropdowns() {
        document.querySelectorAll('[data-dropdown-toggle]').forEach(function(toggle) {
            toggle.addEventListener('click', function(e) {
                e.preventDefault();
                const target = document.querySelector(toggle.getAttribute('data-dropdown-toggle'));
                if (target) {
                    target.classList.toggle('open');
                }
            });
        });

        // Close dropdowns when clicking outside
        document.addEventListener('click', function(e) {
            if (!e.target.closest('[data-dropdown-toggle]') && !e.target.closest('.dropdown-menu')) {
                document.querySelectorAll('.dropdown-menu.open').forEach(function(menu) {
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

    // Initialize
    document.addEventListener('DOMContentLoaded', function() {
        loadTheme();
        initDropdowns();
        initFlashMessages();
        initConfirmDialogs();
    });
})();
