// HTMX configuration
document.body.addEventListener('htmx:configRequest', function(event) {
    // Add any custom headers here
    // event.detail.headers['X-Custom-Header'] = 'value';
});

// Handle HTMX errors
document.body.addEventListener('htmx:responseError', function(event) {
    console.error('HTMX request failed:', event.detail.error);
});

// Loading indicator using classes
document.body.addEventListener('htmx:beforeRequest', function(event) {
    event.target.classList.add('htmx-request');
});

document.body.addEventListener('htmx:afterRequest', function(event) {
    event.target.classList.remove('htmx-request');
});

// Log HTMX events in development (remove in production)
if (window.location.hostname === 'localhost') {
    document.body.addEventListener('htmx:afterSwap', function(event) {
        console.log('HTMX swap completed:', event.detail.target.id);
    });
}
