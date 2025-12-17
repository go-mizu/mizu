// {{.Name}} - Custom JavaScript
// The Mizu Live runtime handles all WebSocket and DOM patching automatically.
// This file is for optional enhancements only.

(function() {
    'use strict';

    // Example: Clear form inputs after submit
    document.addEventListener('submit', function(e) {
        const form = e.target;
        if (form.hasAttribute('data-lv-submit')) {
            // Clear text inputs after a short delay to allow the event to process
            setTimeout(function() {
                form.querySelectorAll('input[type="text"], input:not([type])').forEach(function(input) {
                    input.value = '';
                });
            }, 100);
        }
    });

    // Example: Auto-focus first input on page load
    document.addEventListener('DOMContentLoaded', function() {
        const firstInput = document.querySelector('[data-lv] input[type="text"]:not([readonly])');
        if (firstInput) {
            firstInput.focus();
        }
    });
})();
