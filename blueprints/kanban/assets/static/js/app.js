/**
 * Kanban Blueprint - Main JavaScript
 * Handles drag-and-drop, command palette, and UI interactions
 */

(function() {
    'use strict';

    // ==========================================================================
    // Configuration & State
    // ==========================================================================
    const state = {
        draggedCard: null,
        commandPaletteOpen: false,
        selectedCommandIndex: 0,
        sidebarOpen: false
    };

    // ==========================================================================
    // Initialization
    // ==========================================================================
    document.addEventListener('DOMContentLoaded', () => {
        initTheme();
        initDragAndDrop();
        initCommandPalette();
        initDropdowns();
        initModals();
        initSidebar();
        initToasts();
        initKeyboardShortcuts();
    });

    // ==========================================================================
    // Theme Management
    // ==========================================================================
    function initTheme() {
        const savedTheme = localStorage.getItem('theme') || 'dark';
        document.documentElement.setAttribute('data-theme', savedTheme);
    }

    function setTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('theme', theme);
    }

    window.setTheme = setTheme;

    // ==========================================================================
    // Drag and Drop
    // ==========================================================================
    function initDragAndDrop() {
        const cards = document.querySelectorAll('.issue-card[draggable="true"]');
        const columns = document.querySelectorAll('.column-content');

        cards.forEach(card => {
            card.addEventListener('dragstart', handleDragStart);
            card.addEventListener('dragend', handleDragEnd);
        });

        columns.forEach(column => {
            column.addEventListener('dragover', handleDragOver);
            column.addEventListener('dragenter', handleDragEnter);
            column.addEventListener('dragleave', handleDragLeave);
            column.addEventListener('drop', handleDrop);
        });
    }

    function handleDragStart(e) {
        state.draggedCard = e.target;
        e.target.classList.add('dragging');
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', e.target.dataset.issueId);

        // Create ghost image
        const ghost = e.target.cloneNode(true);
        ghost.style.opacity = '0.8';
        ghost.style.transform = 'rotate(3deg)';
        document.body.appendChild(ghost);
        ghost.style.position = 'absolute';
        ghost.style.top = '-1000px';
        e.dataTransfer.setDragImage(ghost, 0, 0);
        setTimeout(() => ghost.remove(), 0);
    }

    function handleDragEnd(e) {
        e.target.classList.remove('dragging');
        state.draggedCard = null;

        document.querySelectorAll('.column-content').forEach(col => {
            col.classList.remove('drag-over');
        });
    }

    function handleDragOver(e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';

        const column = e.currentTarget;
        const afterElement = getDragAfterElement(column, e.clientY);
        const dragging = state.draggedCard;

        if (dragging && afterElement == null) {
            column.appendChild(dragging);
        } else if (dragging && afterElement) {
            column.insertBefore(dragging, afterElement);
        }
    }

    function handleDragEnter(e) {
        e.preventDefault();
        e.currentTarget.classList.add('drag-over');
    }

    function handleDragLeave(e) {
        // Only remove class if we're actually leaving the column
        const rect = e.currentTarget.getBoundingClientRect();
        const x = e.clientX;
        const y = e.clientY;

        if (x < rect.left || x >= rect.right || y < rect.top || y >= rect.bottom) {
            e.currentTarget.classList.remove('drag-over');
        }
    }

    function handleDrop(e) {
        e.preventDefault();
        const column = e.currentTarget;
        column.classList.remove('drag-over');

        const issueId = e.dataTransfer.getData('text/plain');
        const newStatus = column.dataset.status;

        if (issueId && newStatus) {
            updateIssueStatus(issueId, newStatus);
        }
    }

    function getDragAfterElement(container, y) {
        const draggableElements = [...container.querySelectorAll('.issue-card:not(.dragging)')];

        return draggableElements.reduce((closest, child) => {
            const box = child.getBoundingClientRect();
            const offset = y - box.top - box.height / 2;

            if (offset < 0 && offset > closest.offset) {
                return { offset: offset, element: child };
            } else {
                return closest;
            }
        }, { offset: Number.NEGATIVE_INFINITY }).element;
    }

    async function updateIssueStatus(issueId, status) {
        try {
            const response = await fetch(`/api/v1/issues/${issueId}`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ status })
            });

            if (!response.ok) {
                throw new Error('Failed to update issue');
            }

            showToast('Issue updated', 'success');
            updateColumnCounts();
        } catch (error) {
            console.error('Error updating issue:', error);
            showToast('Failed to update issue', 'error');
            // Reload to reset positions
            window.location.reload();
        }
    }

    function updateColumnCounts() {
        document.querySelectorAll('.board-column').forEach(column => {
            const count = column.querySelectorAll('.issue-card').length;
            const countEl = column.querySelector('.column-count');
            if (countEl) {
                countEl.textContent = count;
            }
        });
    }

    // ==========================================================================
    // Command Palette
    // ==========================================================================
    function initCommandPalette() {
        const overlay = document.querySelector('.command-palette-overlay');
        const input = document.querySelector('.command-input');

        if (!overlay || !input) return;

        input.addEventListener('input', handleCommandInput);
        input.addEventListener('keydown', handleCommandKeydown);
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                closeCommandPalette();
            }
        });
    }

    function openCommandPalette() {
        const overlay = document.querySelector('.command-palette-overlay');
        const input = document.querySelector('.command-input');

        if (!overlay) return;

        overlay.classList.add('open');
        state.commandPaletteOpen = true;
        state.selectedCommandIndex = 0;

        if (input) {
            input.value = '';
            input.focus();
        }

        updateCommandResults('');
    }

    function closeCommandPalette() {
        const overlay = document.querySelector('.command-palette-overlay');
        if (!overlay) return;

        overlay.classList.remove('open');
        state.commandPaletteOpen = false;
    }

    function toggleCommandPalette() {
        if (state.commandPaletteOpen) {
            closeCommandPalette();
        } else {
            openCommandPalette();
        }
    }

    function handleCommandInput(e) {
        const query = e.target.value;
        updateCommandResults(query);
    }

    function handleCommandKeydown(e) {
        const results = document.querySelector('.command-results');
        if (!results) return;

        const items = results.querySelectorAll('.command-item');

        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                state.selectedCommandIndex = Math.min(state.selectedCommandIndex + 1, items.length - 1);
                updateCommandSelection(items);
                break;
            case 'ArrowUp':
                e.preventDefault();
                state.selectedCommandIndex = Math.max(state.selectedCommandIndex - 1, 0);
                updateCommandSelection(items);
                break;
            case 'Enter':
                e.preventDefault();
                if (items[state.selectedCommandIndex]) {
                    executeCommand(items[state.selectedCommandIndex]);
                }
                break;
            case 'Escape':
                e.preventDefault();
                closeCommandPalette();
                break;
        }
    }

    function updateCommandSelection(items) {
        items.forEach((item, index) => {
            item.classList.toggle('selected', index === state.selectedCommandIndex);
        });
    }

    function updateCommandResults(query) {
        const results = document.querySelector('.command-results');
        if (!results) return;

        const commands = getCommands();
        const filtered = query
            ? commands.filter(cmd =>
                cmd.label.toLowerCase().includes(query.toLowerCase()) ||
                cmd.keywords?.some(k => k.toLowerCase().includes(query.toLowerCase()))
              )
            : commands;

        const grouped = groupBy(filtered, 'group');

        let html = '';
        for (const [group, items] of Object.entries(grouped)) {
            html += `<div class="command-group">
                <div class="command-group-title">${group}</div>
                ${items.map((item, i) => `
                    <div class="command-item ${i === 0 && state.selectedCommandIndex === 0 ? 'selected' : ''}"
                         data-action="${item.action}"
                         data-url="${item.url || ''}">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            ${item.icon}
                        </svg>
                        <span class="command-item-text">${item.label}</span>
                        ${item.shortcut ? `
                            <div class="command-item-shortcut">
                                ${item.shortcut.split('+').map(k => `<kbd>${k}</kbd>`).join('')}
                            </div>
                        ` : ''}
                    </div>
                `).join('')}
            </div>`;
        }

        results.innerHTML = html;

        // Add click handlers
        results.querySelectorAll('.command-item').forEach(item => {
            item.addEventListener('click', () => executeCommand(item));
        });

        state.selectedCommandIndex = 0;
    }

    function getCommands() {
        const workspaceSlug = document.body.dataset.workspace;
        const projectKey = document.body.dataset.project;

        return [
            { group: 'Navigation', label: 'Go to Home', action: 'navigate', url: '/', icon: '<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/>', shortcut: 'g+h', keywords: ['home', 'dashboard'] },
            { group: 'Navigation', label: 'Go to Board', action: 'navigate', url: projectKey ? `/${workspaceSlug}/${projectKey}/board` : '#', icon: '<rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/>', shortcut: 'g+b', keywords: ['board', 'kanban'] },
            { group: 'Navigation', label: 'Go to List', action: 'navigate', url: projectKey ? `/${workspaceSlug}/${projectKey}/list` : '#', icon: '<line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/>', shortcut: 'g+l', keywords: ['list', 'issues'] },
            { group: 'Navigation', label: 'Go to Backlog', action: 'navigate', url: projectKey ? `/${workspaceSlug}/${projectKey}/backlog` : '#', icon: '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>', shortcut: 'g+k', keywords: ['backlog'] },
            { group: 'Navigation', label: 'Go to Settings', action: 'navigate', url: '/settings', icon: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>', shortcut: 'g+s', keywords: ['settings', 'preferences'] },
            { group: 'Actions', label: 'Create Issue', action: 'createIssue', icon: '<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>', shortcut: 'c', keywords: ['new', 'add', 'create', 'issue', 'task'] },
            { group: 'Actions', label: 'Create Project', action: 'createProject', icon: '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/><line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/>', keywords: ['new', 'add', 'create', 'project'] },
            { group: 'Theme', label: 'Switch to Dark Mode', action: 'theme', theme: 'dark', icon: '<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>', keywords: ['dark', 'theme', 'night'] },
            { group: 'Theme', label: 'Switch to Light Mode', action: 'theme', theme: 'light', icon: '<circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>', keywords: ['light', 'theme', 'day'] },
        ];
    }

    function executeCommand(item) {
        const action = item.dataset.action;

        switch (action) {
            case 'navigate':
                window.location.href = item.dataset.url;
                break;
            case 'createIssue':
                closeCommandPalette();
                openCreateIssueModal();
                break;
            case 'createProject':
                closeCommandPalette();
                openCreateProjectModal();
                break;
            case 'theme':
                setTheme(item.dataset.theme);
                closeCommandPalette();
                break;
        }
    }

    window.toggleCommandPalette = toggleCommandPalette;

    // ==========================================================================
    // Dropdowns
    // ==========================================================================
    function initDropdowns() {
        document.querySelectorAll('.dropdown').forEach(dropdown => {
            const trigger = dropdown.querySelector('[data-dropdown-trigger]');
            if (trigger) {
                trigger.addEventListener('click', (e) => {
                    e.stopPropagation();
                    toggleDropdown(dropdown);
                });
            }
        });

        document.addEventListener('click', () => {
            document.querySelectorAll('.dropdown.open').forEach(d => d.classList.remove('open'));
        });
    }

    function toggleDropdown(dropdown) {
        const wasOpen = dropdown.classList.contains('open');
        document.querySelectorAll('.dropdown.open').forEach(d => d.classList.remove('open'));
        if (!wasOpen) {
            dropdown.classList.add('open');
        }
    }

    // ==========================================================================
    // Modals
    // ==========================================================================
    function initModals() {
        document.querySelectorAll('.modal-overlay').forEach(overlay => {
            overlay.addEventListener('click', (e) => {
                if (e.target === overlay) {
                    closeModal(overlay);
                }
            });
        });

        document.querySelectorAll('.modal-close').forEach(btn => {
            btn.addEventListener('click', () => {
                const modal = btn.closest('.modal-overlay');
                if (modal) closeModal(modal);
            });
        });
    }

    function openModal(id) {
        const modal = document.getElementById(id);
        if (modal) {
            modal.classList.add('open');
        }
    }

    function closeModal(modal) {
        if (typeof modal === 'string') {
            modal = document.getElementById(modal);
        }
        if (modal) {
            modal.classList.remove('open');
        }
    }

    function openCreateIssueModal() {
        openModal('create-issue-modal');
    }

    function openCreateProjectModal() {
        openModal('create-project-modal');
    }

    window.openModal = openModal;
    window.closeModal = closeModal;
    window.openCreateIssueModal = openCreateIssueModal;
    window.openCreateProjectModal = openCreateProjectModal;

    // ==========================================================================
    // Sidebar
    // ==========================================================================
    function initSidebar() {
        const toggle = document.querySelector('[data-sidebar-toggle]');
        const sidebar = document.querySelector('.sidebar');

        if (toggle && sidebar) {
            toggle.addEventListener('click', () => {
                sidebar.classList.toggle('open');
                state.sidebarOpen = sidebar.classList.contains('open');
            });
        }
    }

    // ==========================================================================
    // Toasts
    // ==========================================================================
    function initToasts() {
        // Create toast container if it doesn't exist
        if (!document.querySelector('.toast-container')) {
            const container = document.createElement('div');
            container.className = 'toast-container';
            document.body.appendChild(container);
        }
    }

    function showToast(message, type = 'info', duration = 3000) {
        const container = document.querySelector('.toast-container');
        if (!container) return;

        const icons = {
            success: '<path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/>',
            error: '<circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>',
            warning: '<path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>',
            info: '<circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/>'
        };

        const toast = document.createElement('div');
        toast.className = 'toast';
        toast.innerHTML = `
            <svg class="toast-icon ${type}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                ${icons[type] || icons.info}
            </svg>
            <span class="toast-message">${message}</span>
            <svg class="toast-close" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
        `;

        toast.querySelector('.toast-close').addEventListener('click', () => {
            toast.remove();
        });

        container.appendChild(toast);

        if (duration > 0) {
            setTimeout(() => {
                toast.style.animation = 'slideIn 0.3s ease reverse';
                setTimeout(() => toast.remove(), 300);
            }, duration);
        }
    }

    window.showToast = showToast;

    // ==========================================================================
    // Keyboard Shortcuts
    // ==========================================================================
    function initKeyboardShortcuts() {
        let keys = [];
        let keyTimer;

        document.addEventListener('keydown', (e) => {
            // Ignore if typing in input
            if (e.target.matches('input, textarea, select, [contenteditable]')) {
                if (e.key === 'Escape') {
                    e.target.blur();
                }
                return;
            }

            // Command palette
            if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
                e.preventDefault();
                toggleCommandPalette();
                return;
            }

            // Escape
            if (e.key === 'Escape') {
                if (state.commandPaletteOpen) {
                    closeCommandPalette();
                    return;
                }
                document.querySelectorAll('.modal-overlay.open').forEach(m => closeModal(m));
                document.querySelectorAll('.dropdown.open').forEach(d => d.classList.remove('open'));
                return;
            }

            // Track key sequences
            clearTimeout(keyTimer);
            keys.push(e.key.toLowerCase());
            keyTimer = setTimeout(() => keys = [], 500);

            const sequence = keys.join('+');

            // Go shortcuts
            if (sequence === 'g+h') {
                window.location.href = '/';
            } else if (sequence === 'g+b') {
                navigateToProjectView('board');
            } else if (sequence === 'g+l') {
                navigateToProjectView('list');
            } else if (sequence === 'g+k') {
                navigateToProjectView('backlog');
            } else if (sequence === 'g+s') {
                window.location.href = '/settings';
            }

            // Single key shortcuts
            if (keys.length === 1) {
                if (e.key === 'c') {
                    openCreateIssueModal();
                } else if (e.key === '?') {
                    toggleCommandPalette();
                }
            }
        });
    }

    function navigateToProjectView(view) {
        const workspaceSlug = document.body.dataset.workspace;
        const projectKey = document.body.dataset.project;

        if (workspaceSlug && projectKey) {
            window.location.href = `/${workspaceSlug}/${projectKey}/${view}`;
        }
    }

    // ==========================================================================
    // API Helpers
    // ==========================================================================
    async function api(url, options = {}) {
        const defaults = {
            headers: {
                'Content-Type': 'application/json',
            },
        };

        const config = { ...defaults, ...options };
        if (config.body && typeof config.body === 'object') {
            config.body = JSON.stringify(config.body);
        }

        const response = await fetch(url, config);

        if (!response.ok) {
            const error = await response.json().catch(() => ({ message: 'Request failed' }));
            throw new Error(error.message || 'Request failed');
        }

        if (response.status === 204) {
            return null;
        }

        return response.json();
    }

    window.api = api;

    // ==========================================================================
    // Utility Functions
    // ==========================================================================
    function groupBy(array, key) {
        return array.reduce((groups, item) => {
            const group = item[key];
            groups[group] = groups[group] || [];
            groups[group].push(item);
            return groups;
        }, {});
    }

    function debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    function throttle(func, limit) {
        let inThrottle;
        return function(...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    }

    window.debounce = debounce;
    window.throttle = throttle;

    // ==========================================================================
    // Issue Card Quick Actions
    // ==========================================================================
    function initIssueCards() {
        document.querySelectorAll('.issue-card').forEach(card => {
            card.addEventListener('click', (e) => {
                // Don't navigate if clicking on actions
                if (e.target.closest('.issue-card-actions')) return;

                const issueKey = card.dataset.issueKey;
                const workspaceSlug = document.body.dataset.workspace;
                const projectKey = document.body.dataset.project;

                if (issueKey && workspaceSlug && projectKey) {
                    window.location.href = `/${workspaceSlug}/${projectKey}/issues/${issueKey}`;
                }
            });
        });
    }

    // Run on load
    document.addEventListener('DOMContentLoaded', initIssueCards);

    // ==========================================================================
    // Inline Editing
    // ==========================================================================
    function initInlineEditing() {
        document.querySelectorAll('[data-editable]').forEach(el => {
            el.addEventListener('blur', async () => {
                const field = el.dataset.editable;
                const value = el.value || el.textContent;
                const issueId = el.dataset.issueId;

                if (issueId && field) {
                    try {
                        await api(`/api/v1/issues/${issueId}`, {
                            method: 'PATCH',
                            body: { [field]: value }
                        });
                        showToast('Updated', 'success');
                    } catch (error) {
                        showToast('Failed to update', 'error');
                    }
                }
            });
        });
    }

    document.addEventListener('DOMContentLoaded', initInlineEditing);

    // ==========================================================================
    // Search
    // ==========================================================================
    let searchDebounce;

    function initSearch() {
        const searchInput = document.querySelector('[data-search]');
        if (!searchInput) return;

        searchInput.addEventListener('input', (e) => {
            clearTimeout(searchDebounce);
            searchDebounce = setTimeout(() => {
                performSearch(e.target.value);
            }, 300);
        });
    }

    async function performSearch(query) {
        if (!query || query.length < 2) {
            hideSearchResults();
            return;
        }

        const workspaceSlug = document.body.dataset.workspace;
        if (!workspaceSlug) return;

        try {
            const results = await api(`/api/v1/workspaces/${workspaceSlug}/search?q=${encodeURIComponent(query)}`);
            showSearchResults(results);
        } catch (error) {
            console.error('Search failed:', error);
        }
    }

    function showSearchResults(results) {
        let container = document.querySelector('.search-results');
        if (!container) {
            container = document.createElement('div');
            container.className = 'search-results dropdown-content';
            document.querySelector('[data-search]')?.parentElement?.appendChild(container);
        }

        if (!results.issues?.length) {
            container.innerHTML = '<div class="dropdown-item">No results found</div>';
        } else {
            container.innerHTML = results.issues.map(issue => `
                <a href="/${document.body.dataset.workspace}/${issue.project_key}/issues/${issue.key}" class="dropdown-item">
                    <span class="font-mono text-muted">${issue.key}</span>
                    <span>${issue.title}</span>
                </a>
            `).join('');
        }

        container.style.display = 'block';
    }

    function hideSearchResults() {
        const container = document.querySelector('.search-results');
        if (container) {
            container.style.display = 'none';
        }
    }

    document.addEventListener('DOMContentLoaded', initSearch);

})();
