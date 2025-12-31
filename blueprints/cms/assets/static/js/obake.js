// Obake - CMS Admin JavaScript

(function() {
    'use strict';

    // ================================
    // Theme Toggle
    // ================================
    function getStoredTheme() {
        return localStorage.getItem('obake-theme');
    }

    function setStoredTheme(theme) {
        localStorage.setItem('obake-theme', theme);
    }

    function getPreferredTheme() {
        const stored = getStoredTheme();
        if (stored) {
            return stored;
        }
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    function setTheme(theme) {
        if (theme === 'dark') {
            document.documentElement.setAttribute('data-theme', 'dark');
        } else {
            document.documentElement.removeAttribute('data-theme');
        }
        updateThemeIcons(theme);
    }

    function updateThemeIcons(theme) {
        const sunIcon = document.querySelector('.gh-theme-sun');
        const moonIcon = document.querySelector('.gh-theme-moon');
        if (sunIcon && moonIcon) {
            if (theme === 'dark') {
                sunIcon.classList.remove('active');
                moonIcon.classList.add('active');
            } else {
                sunIcon.classList.add('active');
                moonIcon.classList.remove('active');
            }
        }
    }

    // Initialize theme on page load (before DOMContentLoaded for no flash)
    setTheme(getPreferredTheme());

    window.toggleTheme = function() {
        const currentTheme = document.documentElement.getAttribute('data-theme');
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        setTheme(newTheme);
        setStoredTheme(newTheme);
    };

    // Listen for OS theme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
        if (!getStoredTheme()) {
            setTheme(e.matches ? 'dark' : 'light');
        }
    });

    // ================================
    // Navigation Section Toggle
    // ================================
    window.toggleNavSection = function(header) {
        const section = header.closest('.gh-nav-section');
        if (section) {
            section.classList.toggle('expanded');
        }
    };

    // ================================
    // User Menu Toggle
    // ================================
    window.toggleUserMenu = function() {
        const menu = document.getElementById('userMenu');
        if (menu) {
            menu.classList.toggle('active');
        }
    };

    // Close user menu when clicking outside
    document.addEventListener('click', function(e) {
        const menu = document.getElementById('userMenu');
        const btn = document.querySelector('.gh-nav-user-btn');
        if (menu && btn && !menu.contains(e.target) && !btn.contains(e.target)) {
            menu.classList.remove('active');
        }
    });

    // ================================
    // Toast Notifications
    // ================================
    window.closeToast = function() {
        const toast = document.getElementById('toast');
        if (toast) {
            toast.style.animation = 'slideOut 0.3s ease forwards';
            setTimeout(() => toast.remove(), 300);
        }
    };

    // Auto-close toasts after 5 seconds
    document.addEventListener('DOMContentLoaded', function() {
        const toast = document.getElementById('toast');
        if (toast) {
            setTimeout(() => closeToast(), 5000);
        }
    });

    // ================================
    // Search Modal
    // ================================
    window.openSearch = function() {
        const modal = document.getElementById('searchModal');
        if (modal) {
            modal.classList.add('active');
            const input = document.getElementById('searchInput');
            if (input) {
                input.focus();
            }
        }
    };

    window.closeSearch = function() {
        const modal = document.getElementById('searchModal');
        if (modal) {
            modal.classList.remove('active');
        }
    };

    // Keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        // Cmd/Ctrl + K to open search
        if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
            e.preventDefault();
            openSearch();
        }
        // Escape to close search
        if (e.key === 'Escape') {
            closeSearch();
        }
    });

    // Close search when clicking backdrop
    document.addEventListener('click', function(e) {
        const modal = document.getElementById('searchModal');
        if (modal && e.target === modal) {
            closeSearch();
        }
    });

    // Search functionality
    let searchTimeout;
    document.addEventListener('DOMContentLoaded', function() {
        const searchInput = document.getElementById('searchInput');
        const searchResults = document.getElementById('searchResults');

        if (searchInput && searchResults) {
            searchInput.addEventListener('input', function() {
                clearTimeout(searchTimeout);
                const query = this.value.trim();

                if (query.length < 2) {
                    searchResults.innerHTML = '';
                    return;
                }

                searchTimeout = setTimeout(function() {
                    fetch('/obake/search/?q=' + encodeURIComponent(query))
                        .then(res => res.json())
                        .then(data => {
                            if (data.Results && data.Results.length > 0) {
                                searchResults.innerHTML = data.Results.map(item => `
                                    <a href="${item.URL}" class="gh-search-result">
                                        <span class="gh-search-result-type">${item.Type}</span>
                                        <span class="gh-search-result-title">${item.Title}</span>
                                    </a>
                                `).join('');
                            } else {
                                searchResults.innerHTML = '<div class="gh-search-empty">No results found</div>';
                            }
                        })
                        .catch(err => {
                            console.error('Search error:', err);
                        });
                }, 300);
            });
        }
    });

    // ================================
    // Editor Sidebar Toggle
    // ================================
    window.toggleSettings = function() {
        const sidebar = document.getElementById('editorSidebar');
        if (sidebar) {
            sidebar.classList.toggle('active');
        }
    };

    // ================================
    // Card Menu Toggle
    // ================================
    window.toggleCardMenu = function() {
        const menu = document.getElementById('cardMenu');
        if (menu) {
            menu.classList.toggle('active');
        }
    };

    window.insertCard = function(type) {
        const textarea = document.getElementById('editorContent');
        if (!textarea) return;

        let cardContent = '';
        switch (type) {
            case 'image':
                cardContent = '\n\n![Image description](image-url)\n\n';
                break;
            case 'markdown':
                cardContent = '\n\n```markdown\nYour markdown content here\n```\n\n';
                break;
            case 'html':
                cardContent = '\n\n<div class="custom-html">\n  <!-- Your HTML here -->\n</div>\n\n';
                break;
            case 'divider':
                cardContent = '\n\n---\n\n';
                break;
            case 'bookmark':
                cardContent = '\n\n[Bookmark: Title](url)\n\n';
                break;
            case 'callout':
                cardContent = '\n\n> **Note:** Your callout text here\n\n';
                break;
            case 'toggle':
                cardContent = '\n\n<details>\n<summary>Toggle title</summary>\nHidden content here\n</details>\n\n';
                break;
            case 'button':
                cardContent = '\n\n<a href="#" class="button">Button Text</a>\n\n';
                break;
            default:
                cardContent = '\n\n' + type + '\n\n';
        }

        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const text = textarea.value;
        textarea.value = text.substring(0, start) + cardContent + text.substring(end);
        textarea.selectionStart = textarea.selectionEnd = start + cardContent.length;
        textarea.focus();

        toggleCardMenu();
    };

    // Close card menu when clicking outside
    document.addEventListener('click', function(e) {
        const menu = document.getElementById('cardMenu');
        const btn = document.querySelector('.gh-editor-plus-btn');
        if (menu && btn && !menu.contains(e.target) && !btn.contains(e.target)) {
            menu.classList.remove('active');
        }
    });

    // ================================
    // Expandable Sections
    // ================================
    window.toggleExpand = function(btn) {
        btn.classList.toggle('active');
        const content = btn.nextElementSibling;
        if (content) {
            content.classList.toggle('active');
        }
    };

    // ================================
    // Password Toggle
    // ================================
    window.togglePassword = function() {
        const input = document.getElementById('password');
        const eyeOpen = document.querySelector('.gh-auth-eye-open');
        const eyeClosed = document.querySelector('.gh-auth-eye-closed');

        if (input) {
            if (input.type === 'password') {
                input.type = 'text';
                if (eyeOpen) eyeOpen.style.display = 'none';
                if (eyeClosed) eyeClosed.style.display = 'block';
            } else {
                input.type = 'password';
                if (eyeOpen) eyeOpen.style.display = 'block';
                if (eyeClosed) eyeClosed.style.display = 'none';
            }
        }
    };

    // ================================
    // Navigation Item Add
    // ================================
    window.addNavItem = function(containerId) {
        const container = document.getElementById(containerId);
        if (!container) return;

        const item = document.createElement('div');
        item.className = 'gh-nav-item';
        item.innerHTML = `
            <input type="text" name="${containerId === 'primaryNav' ? 'primary' : 'secondary'}_nav_label[]" placeholder="Label" class="gh-form-input">
            <input type="text" name="${containerId === 'primaryNav' ? 'primary' : 'secondary'}_nav_url[]" placeholder="URL" class="gh-form-input">
            <button type="button" class="gh-nav-item-remove" onclick="removeNavItem(this)">&times;</button>
        `;
        container.insertBefore(item, container.lastElementChild);
    };

    window.removeNavItem = function(btn) {
        const item = btn.closest('.gh-nav-item');
        if (item) {
            item.remove();
        }
    };

    // ================================
    // Slug Generation
    // ================================
    function generateSlug(text) {
        return text
            .toLowerCase()
            .replace(/[^\w\s-]/g, '')
            .replace(/\s+/g, '-')
            .replace(/-+/g, '-')
            .trim();
    }

    // Auto-generate slug from title
    document.addEventListener('DOMContentLoaded', function() {
        const titleInput = document.querySelector('.gh-editor-title');
        const slugInput = document.querySelector('input[name="slug"]');

        if (titleInput && slugInput && !slugInput.value) {
            titleInput.addEventListener('blur', function() {
                if (!slugInput.dataset.modified) {
                    slugInput.value = generateSlug(this.value);
                }
            });

            slugInput.addEventListener('input', function() {
                this.dataset.modified = 'true';
            });
        }
    });

    // ================================
    // Auto-save Draft
    // ================================
    let autoSaveTimeout;
    document.addEventListener('DOMContentLoaded', function() {
        const form = document.querySelector('.gh-editor-form');
        const textarea = document.getElementById('editorContent');
        const titleInput = document.querySelector('.gh-editor-title');

        if (form && textarea) {
            const autoSave = function() {
                clearTimeout(autoSaveTimeout);
                autoSaveTimeout = setTimeout(function() {
                    const status = document.getElementById('editorStatus');
                    if (status) {
                        status.textContent = 'Saving...';
                        // In a real implementation, this would save to the server
                        setTimeout(function() {
                            status.textContent = 'Draft';
                        }, 500);
                    }
                }, 2000);
            };

            textarea.addEventListener('input', autoSave);
            if (titleInput) {
                titleInput.addEventListener('input', autoSave);
            }
        }
    });

    // ================================
    // Image Upload Preview
    // ================================
    document.addEventListener('DOMContentLoaded', function() {
        const imageUploads = document.querySelectorAll('.gh-sidebar-image-upload, .gh-branding-upload');

        imageUploads.forEach(function(upload) {
            upload.addEventListener('click', function() {
                const input = document.createElement('input');
                input.type = 'file';
                input.accept = 'image/*';

                input.addEventListener('change', function(e) {
                    const file = e.target.files[0];
                    if (file) {
                        const reader = new FileReader();
                        reader.onload = function(event) {
                            // Preview the image
                            let img = upload.querySelector('img');
                            if (!img) {
                                img = document.createElement('img');
                                upload.innerHTML = '';
                                upload.appendChild(img);
                            }
                            img.src = event.target.result;
                            img.style.width = '100%';
                            img.style.height = '100%';
                            img.style.objectFit = 'cover';
                        };
                        reader.readAsDataURL(file);
                    }
                });

                input.click();
            });
        });
    });

    // ================================
    // Color Picker Sync
    // ================================
    document.addEventListener('DOMContentLoaded', function() {
        const colorInputs = document.querySelectorAll('.gh-color-input');
        colorInputs.forEach(function(input) {
            const textInput = input.nextElementSibling;
            if (textInput && textInput.type === 'text') {
                input.addEventListener('input', function() {
                    textInput.value = this.value;
                });
                textInput.addEventListener('input', function() {
                    if (/^#[0-9A-Fa-f]{6}$/.test(this.value)) {
                        input.value = this.value;
                    }
                });
            }
        });
    });

    // ================================
    // Keyboard Shortcuts
    // ================================
    document.addEventListener('keydown', function(e) {
        // Cmd/Ctrl + S to save
        if ((e.metaKey || e.ctrlKey) && e.key === 's') {
            e.preventDefault();
            const saveBtn = document.querySelector('.gh-editor-form button[type="submit"]');
            if (saveBtn) {
                saveBtn.click();
            }
        }
    });

    // ================================
    // Initialize
    // ================================
    document.addEventListener('DOMContentLoaded', function() {
        // Add animation class for CSS
        const style = document.createElement('style');
        style.textContent = `
            @keyframes slideOut {
                from { transform: translateX(0); opacity: 1; }
                to { transform: translateX(100%); opacity: 0; }
            }
        `;
        document.head.appendChild(style);
    });

})();
