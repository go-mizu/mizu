/**
 * App JavaScript
 *
 * This file is for custom JavaScript functionality.
 * It's loaded at the end of the body for optimal performance.
 */

// DOM ready helper
function ready(fn) {
    if (document.readyState !== 'loading') {
        fn();
    } else {
        document.addEventListener('DOMContentLoaded', fn);
    }
}

// Initialize app
ready(function() {
    console.log('App initialized');

    // Add any custom initialization code here
    // Examples:
    // - Event listeners
    // - Form handling
    // - HTMX configuration
    // - Alpine.js components
});

// Utility: Debounce function
function debounce(fn, wait) {
    let timeout;
    return function(...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => fn.apply(this, args), wait);
    };
}

// Utility: Format date
function formatDate(date, locale = 'en-US') {
    return new Date(date).toLocaleDateString(locale, {
        year: 'numeric',
        month: 'long',
        day: 'numeric'
    });
}
