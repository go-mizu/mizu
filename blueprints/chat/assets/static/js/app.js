// Mizu Chat - Minimal JavaScript
// Most logic is in page templates. This file handles theme initialization.

// Apply saved theme on load
(function() {
    const theme = localStorage.getItem('theme') || 'dark';
    if (theme === 'dark') {
        document.documentElement.classList.add('dark');
    } else {
        document.documentElement.classList.remove('dark');
    }
})();

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    // Ctrl/Cmd + K: Focus message input
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        const input = document.getElementById('message-input');
        if (input) input.focus();
    }
});
