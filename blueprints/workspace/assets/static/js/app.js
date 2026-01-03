// Workspace App JavaScript

(function() {
    'use strict';

    // API helper
    const api = {
        async request(method, path, data) {
            const options = {
                method,
                headers: { 'Content-Type': 'application/json' },
                credentials: 'same-origin'
            };
            if (data) {
                options.body = JSON.stringify(data);
            }
            const response = await fetch('/api/v1' + path, options);
            if (!response.ok) {
                let errorMessage = 'Request failed';
                try {
                    const error = await response.json();
                    errorMessage = error.error || errorMessage;
                } catch {
                    errorMessage = response.statusText || errorMessage;
                }
                throw new Error(errorMessage);
            }
            return response.json();
        },
        get: (path) => api.request('GET', path),
        post: (path, data) => api.request('POST', path, data),
        put: (path, data) => api.request('PUT', path, data),
        patch: (path, data) => api.request('PATCH', path, data),
        delete: (path) => api.request('DELETE', path)
    };

    // Auth forms
    const loginForm = document.getElementById('login-form');
    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const errorDiv = document.getElementById('login-error');
            const submitBtn = loginForm.querySelector('button[type="submit"]');

            if (errorDiv) errorDiv.textContent = '';
            if (submitBtn) {
                submitBtn.disabled = true;
                submitBtn.textContent = 'Logging in...';
            }

            try {
                const emailInput = document.getElementById('email');
                const passwordInput = document.getElementById('password');

                if (!emailInput || !passwordInput) {
                    throw new Error('Form inputs not found');
                }

                await api.post('/auth/login', {
                    email: emailInput.value,
                    password: passwordInput.value
                });
                window.location.href = '/app';
            } catch (err) {
                if (submitBtn) {
                    submitBtn.disabled = false;
                    submitBtn.textContent = 'Log in';
                }
                const errorMessage = err.message || 'Login failed. Please try again.';
                if (errorDiv) {
                    errorDiv.textContent = errorMessage;
                    errorDiv.style.display = 'block';
                } else {
                    alert(errorMessage);
                }
            }
        });
    }

    const registerForm = document.getElementById('register-form');
    if (registerForm) {
        registerForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const errorDiv = document.getElementById('register-error');
            const submitBtn = registerForm.querySelector('button[type="submit"]');

            if (errorDiv) errorDiv.textContent = '';
            if (submitBtn) {
                submitBtn.disabled = true;
                submitBtn.textContent = 'Creating account...';
            }

            try {
                const nameInput = document.getElementById('name');
                const emailInput = document.getElementById('email');
                const passwordInput = document.getElementById('password');

                if (!emailInput || !passwordInput) {
                    throw new Error('Form inputs not found');
                }

                await api.post('/auth/register', {
                    name: nameInput ? nameInput.value : '',
                    email: emailInput.value,
                    password: passwordInput.value
                });
                window.location.href = '/app';
            } catch (err) {
                if (submitBtn) {
                    submitBtn.disabled = false;
                    submitBtn.textContent = 'Sign up';
                }
                const errorMessage = err.message || 'Registration failed. Please try again.';
                if (errorDiv) {
                    errorDiv.textContent = errorMessage;
                    errorDiv.style.display = 'block';
                } else {
                    alert(errorMessage);
                }
            }
        });
    }

    // Logout
    const logoutBtn = document.getElementById('logout-btn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', async () => {
            await api.post('/auth/logout', {});
            window.location.href = '/login';
        });
    }

    // Dropdowns
    function setupDropdown(toggleId, dropdownId) {
        const toggle = document.getElementById(toggleId);
        const dropdown = document.getElementById(dropdownId);
        if (!toggle || !dropdown) return;

        toggle.addEventListener('click', (e) => {
            e.stopPropagation();
            dropdown.classList.toggle('open');
        });

        document.addEventListener('click', () => {
            dropdown.classList.remove('open');
        });
    }

    setupDropdown('workspace-switcher', 'workspace-dropdown');
    setupDropdown('user-menu', 'user-dropdown');

    // Page title editing
    const pageTitle = document.getElementById('page-title');
    if (pageTitle) {
        let debounceTimer;
        pageTitle.addEventListener('input', () => {
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(async () => {
                const pageView = document.querySelector('.page-view');
                const pageId = pageView?.dataset.pageId;
                if (pageId) {
                    try {
                        await api.patch('/pages/' + pageId, {
                            title: pageTitle.textContent
                        });
                    } catch (err) {
                        console.error('Failed to update title:', err);
                    }
                }
            }, 500);
        });
    }

    // Block editing (only for non-React pages)
    // Skip if BlockNote React editor is present
    const blockNoteEditor = document.querySelector('.block-editor');
    const blocksContainer = document.getElementById('blocks-container');
    const blockMenu = document.getElementById('block-menu');

    if (blocksContainer && blockMenu && !blockNoteEditor) {
        let currentBlock = null;
        let currentInput = null;

        // Show block menu on slash
        blocksContainer.addEventListener('keydown', (e) => {
            if (e.key === '/') {
                currentInput = e.target;
                currentBlock = e.target.closest('.block') || e.target.closest('.block-placeholder');
                const rect = e.target.getBoundingClientRect();
                blockMenu.style.top = (rect.bottom + 4) + 'px';
                blockMenu.style.left = rect.left + 'px';
                blockMenu.classList.remove('hidden');
            } else if (e.key === 'Escape') {
                blockMenu.classList.add('hidden');
            }
        });

        // Hide menu on click outside
        document.addEventListener('click', (e) => {
            if (!blockMenu.contains(e.target)) {
                blockMenu.classList.add('hidden');
            }
        });

        // Handle block type selection
        blockMenu.addEventListener('click', async (e) => {
            const menuItem = e.target.closest('.menu-item');
            if (!menuItem) return;

            const blockType = menuItem.dataset.type;
            blockMenu.classList.add('hidden');

            // Clear the slash
            if (currentInput) {
                currentInput.textContent = currentInput.textContent.replace(/\/$/, '');
            }

            const pageView = document.querySelector('.page-view');
            const pageId = pageView?.dataset.pageId;

            if (pageId) {
                try {
                    const block = await api.post('/blocks', {
                        page_id: pageId,
                        type: blockType,
                        content: { text: '' }
                    });
                    // Reload page to show new block
                    window.location.reload();
                } catch (err) {
                    console.error('Failed to create block:', err);
                }
            }
        });

        // Save block content on blur
        blocksContainer.addEventListener('blur', async (e) => {
            const block = e.target.closest('.block');
            if (!block) return;

            const blockId = block.dataset.blockId;
            if (!blockId) return;

            try {
                await api.patch('/blocks/' + blockId, {
                    content: { text: e.target.textContent }
                });
            } catch (err) {
                console.error('Failed to update block:', err);
            }
        }, true);

        // Handle todo checkboxes
        blocksContainer.addEventListener('change', async (e) => {
            if (e.target.type !== 'checkbox') return;

            const block = e.target.closest('.block');
            if (!block) return;

            const blockId = block.dataset.blockId;
            if (!blockId) return;

            try {
                await api.patch('/blocks/' + blockId, {
                    content: { checked: e.target.checked }
                });
            } catch (err) {
                console.error('Failed to update block:', err);
            }
        });
    }

    // Add page button
    const addPageBtn = document.getElementById('add-page-btn');
    if (addPageBtn) {
        addPageBtn.addEventListener('click', async () => {
            const workspaceId = addPageBtn.dataset.workspace;
            try {
                const page = await api.post('/pages', {
                    workspace_id: workspaceId,
                    title: 'Untitled',
                    parent_type: 'workspace',
                    parent_id: workspaceId
                });
                // Navigate to new page
                const workspaceSlug = window.location.pathname.split('/')[2];
                window.location.href = '/w/' + workspaceSlug + '/p/' + page.id;
            } catch (err) {
                console.error('Failed to create page:', err);
            }
        });
    }

    // Quick actions on workspace home
    const newPageAction = document.getElementById('new-page-action');
    if (newPageAction) {
        newPageAction.addEventListener('click', async () => {
            const workspaceId = newPageAction.dataset.workspace;
            try {
                const page = await api.post('/pages', {
                    workspace_id: workspaceId,
                    title: 'Untitled',
                    parent_type: 'workspace',
                    parent_id: workspaceId
                });
                const workspaceSlug = window.location.pathname.split('/')[2];
                window.location.href = '/w/' + workspaceSlug + '/p/' + page.id;
            } catch (err) {
                console.error('Failed to create page:', err);
            }
        });
    }

    const newDatabaseAction = document.getElementById('new-database-action');
    if (newDatabaseAction) {
        newDatabaseAction.addEventListener('click', async () => {
            const workspaceId = newDatabaseAction.dataset.workspace;
            try {
                const db = await api.post('/databases', {
                    workspace_id: workspaceId,
                    name: 'Untitled Database'
                });
                const workspaceSlug = window.location.pathname.split('/')[2];
                window.location.href = '/w/' + workspaceSlug + '/d/' + db.id;
            } catch (err) {
                console.error('Failed to create database:', err);
            }
        });
    }

    // Favorite button
    const favoriteBtn = document.getElementById('favorite-btn');
    if (favoriteBtn) {
        favoriteBtn.addEventListener('click', async () => {
            const pageView = document.querySelector('.page-view');
            const pageId = pageView?.dataset.pageId;
            if (!pageId) return;

            try {
                await api.post('/favorites', {
                    target_type: 'page',
                    target_id: pageId
                });
                favoriteBtn.classList.toggle('active');
            } catch (err) {
                console.error('Failed to toggle favorite:', err);
            }
        });
    }

    // Search
    const searchInput = document.getElementById('search-input');
    const searchResults = document.getElementById('search-results');

    if (searchInput && searchResults) {
        let debounceTimer;

        searchInput.addEventListener('input', () => {
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(async () => {
                const query = searchInput.value.trim();
                if (query.length < 2) {
                    searchResults.innerHTML = '<div class="search-empty"><div class="empty-icon">🔍</div><p>Search for pages, databases, and content</p></div>';
                    return;
                }

                searchResults.innerHTML = '<div class="results-loading">Searching...</div>';

                try {
                    const workspaceSlug = window.location.pathname.split('/')[2];
                    // Get workspace ID first
                    const ws = await api.get('/workspaces/' + workspaceSlug);
                    const results = await api.get('/search?workspace_id=' + ws.id + '&q=' + encodeURIComponent(query));

                    if (results.length === 0) {
                        searchResults.innerHTML = '<div class="search-empty"><p>No results found</p></div>';
                        return;
                    }

                    let html = '<div class="results-list">';
                    for (const result of results) {
                        html += `
                            <a href="/w/${workspaceSlug}/p/${result.id}" class="result-item">
                                <span class="result-icon">${result.icon || '📄'}</span>
                                <div class="result-info">
                                    <div class="result-title">${escapeHtml(result.title)}</div>
                                    ${result.snippet ? `<div class="result-snippet">${escapeHtml(result.snippet)}</div>` : ''}
                                </div>
                            </a>
                        `;
                    }
                    html += '</div>';
                    searchResults.innerHTML = html;

                    // Save to recent searches
                    saveRecentSearch(query);
                } catch (err) {
                    console.error('Search failed:', err);
                    searchResults.innerHTML = '<div class="search-empty"><p>Search failed</p></div>';
                }
            }, 300);
        });

        // Load recent searches
        loadRecentSearches();
    }

    function saveRecentSearch(query) {
        const key = 'workspace_recent_searches';
        let searches = JSON.parse(localStorage.getItem(key) || '[]');
        searches = searches.filter(s => s !== query);
        searches.unshift(query);
        searches = searches.slice(0, 5);
        localStorage.setItem(key, JSON.stringify(searches));
    }

    function loadRecentSearches() {
        const container = document.querySelector('#recent-searches .recent-list');
        if (!container) return;

        const searches = JSON.parse(localStorage.getItem('workspace_recent_searches') || '[]');
        if (searches.length === 0) {
            container.innerHTML = '<p class="empty-hint">No recent searches</p>';
            return;
        }

        let html = '';
        for (const query of searches) {
            html += `<button class="recent-item" data-query="${escapeHtml(query)}">${escapeHtml(query)}</button>`;
        }
        container.innerHTML = html;

        container.addEventListener('click', (e) => {
            const item = e.target.closest('.recent-item');
            if (item) {
                const searchInput = document.getElementById('search-input');
                if (searchInput) {
                    searchInput.value = item.dataset.query;
                    searchInput.dispatchEvent(new Event('input'));
                }
            }
        });
    }

    // Settings navigation
    const settingsNav = document.querySelectorAll('.settings-nav-item');
    const settingsSections = document.querySelectorAll('.settings-section');

    settingsNav.forEach(navItem => {
        navItem.addEventListener('click', () => {
            const section = navItem.dataset.section;

            settingsNav.forEach(n => n.classList.remove('active'));
            navItem.classList.add('active');

            settingsSections.forEach(s => {
                s.classList.toggle('active', s.id === 'section-' + section);
            });
        });
    });

    // Workspace settings form
    const workspaceSettingsForm = document.getElementById('workspace-settings-form');
    if (workspaceSettingsForm) {
        workspaceSettingsForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const workspaceSlug = window.location.pathname.split('/')[2];

            try {
                const ws = await api.get('/workspaces/' + workspaceSlug);
                await api.put('/workspaces/' + ws.id, {
                    name: document.getElementById('ws-name').value,
                    slug: document.getElementById('ws-slug').value
                });
                // Redirect to new slug if changed
                const newSlug = document.getElementById('ws-slug').value;
                if (newSlug !== workspaceSlug) {
                    window.location.href = '/w/' + newSlug + '/settings';
                }
            } catch (err) {
                console.error('Failed to update workspace:', err);
                alert('Failed to update workspace: ' + err.message);
            }
        });
    }

    // Invite form
    const inviteForm = document.getElementById('invite-form');
    if (inviteForm) {
        inviteForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const workspaceSlug = window.location.pathname.split('/')[2];

            try {
                const ws = await api.get('/workspaces/' + workspaceSlug);
                await api.post('/workspaces/' + ws.id + '/members', {
                    email: document.getElementById('invite-email').value,
                    role: document.getElementById('invite-role').value
                });
                window.location.reload();
            } catch (err) {
                console.error('Failed to invite member:', err);
                alert('Failed to invite member: ' + err.message);
            }
        });
    }

    // Delete workspace
    const deleteWorkspaceBtn = document.getElementById('delete-workspace-btn');
    if (deleteWorkspaceBtn) {
        deleteWorkspaceBtn.addEventListener('click', async () => {
            if (!confirm('Are you sure you want to delete this workspace? This action cannot be undone.')) {
                return;
            }

            const workspaceSlug = window.location.pathname.split('/')[2];

            try {
                const ws = await api.get('/workspaces/' + workspaceSlug);
                await api.delete('/workspaces/' + ws.id);
                window.location.href = '/app';
            } catch (err) {
                console.error('Failed to delete workspace:', err);
                alert('Failed to delete workspace: ' + err.message);
            }
        });
    }

    // Database title editing
    const databaseTitle = document.getElementById('database-title');
    if (databaseTitle) {
        let debounceTimer;
        databaseTitle.addEventListener('input', () => {
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(async () => {
                const databaseView = document.querySelector('.database-view');
                const databaseId = databaseView?.dataset.databaseId;
                if (databaseId) {
                    try {
                        await api.patch('/databases/' + databaseId, {
                            name: databaseTitle.textContent
                        });
                    } catch (err) {
                        console.error('Failed to update database name:', err);
                    }
                }
            }, 500);
        });
    }

    // View tabs
    const viewTabs = document.querySelectorAll('.view-tab:not(.add-view)');
    viewTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            viewTabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            // TODO: Load view data
        });
    });

    // Theme toggle
    const themeToggle = document.getElementById('theme-toggle');
    if (themeToggle) {
        themeToggle.addEventListener('click', () => {
            const current = document.documentElement.getAttribute('data-theme') || 'light';
            const next = current === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-theme', next);
            localStorage.setItem('theme', next);
        });
    }

    // Helper function
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

})();
