/**
 * Kanban AI - Frontend JavaScript
 * Handles interactions, API calls, modals, dropdowns, drag-and-drop, and command palette
 */

// ========================================
// API Helper
// ========================================

const api = {
  async request(method, path, data, options = {}) {
    const config = {
      method,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      credentials: 'include',
    };

    if (data && method !== 'GET') {
      config.body = JSON.stringify(data);
    }

    const response = await fetch(`/api/v1${path}`, config);

    if (!response.ok) {
      const error = new Error(await response.text());
      error.status = response.status;
      throw error;
    }

    // Handle empty responses
    const text = await response.text();
    return text ? JSON.parse(text) : null;
  },

  get: (path) => api.request('GET', path),
  post: (path, data) => api.request('POST', path, data),
  patch: (path, data) => api.request('PATCH', path, data),
  put: (path, data) => api.request('PUT', path, data),
  delete: (path) => api.request('DELETE', path),
};

// ========================================
// Modal Management
// ========================================

const modals = {
  open(id) {
    const modal = document.getElementById(id);
    if (modal) {
      modal.classList.remove('hidden');
      document.body.style.overflow = 'hidden';

      // Focus first input
      const input = modal.querySelector('input, textarea');
      if (input) {
        setTimeout(() => input.focus(), 100);
      }

      // Trigger event
      modal.dispatchEvent(new CustomEvent('modal:open'));
    }
  },

  close(id) {
    const modal = document.getElementById(id);
    if (modal) {
      modal.classList.add('hidden');
      document.body.style.overflow = '';
      modal.dispatchEvent(new CustomEvent('modal:close'));
    }
  },

  closeAll() {
    document.querySelectorAll('.modal:not(.hidden)').forEach(modal => {
      modal.classList.add('hidden');
    });
    document.body.style.overflow = '';
  },

  init() {
    // Modal trigger buttons
    document.addEventListener('click', (e) => {
      const trigger = e.target.closest('[data-modal]');
      if (trigger) {
        e.preventDefault();
        const modalId = trigger.dataset.modal;
        modals.open(modalId);
      }
    });

    // Close buttons
    document.addEventListener('click', (e) => {
      if (e.target.closest('.modal-close')) {
        const modal = e.target.closest('.modal');
        if (modal) {
          modals.close(modal.id);
        }
      }
    });

    // Backdrop click
    document.addEventListener('click', (e) => {
      if (e.target.classList.contains('modal-backdrop')) {
        const modal = e.target.closest('.modal');
        if (modal) {
          modals.close(modal.id);
        }
      }
    });

    // Escape key
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        modals.closeAll();
        dropdowns.closeAll();
        commandPalette.close();
      }
    });
  }
};

// ========================================
// Dropdown Management
// ========================================

const dropdowns = {
  closeAll() {
    document.querySelectorAll('.dropdown-menu:not(.hidden)').forEach(menu => {
      menu.classList.add('hidden');
    });
  },

  toggle(dropdown) {
    const menu = dropdown.querySelector('.dropdown-menu');
    if (menu) {
      const isHidden = menu.classList.contains('hidden');
      dropdowns.closeAll();
      if (isHidden) {
        menu.classList.remove('hidden');
      }
    }
  },

  init() {
    // Toggle dropdown on trigger click
    document.addEventListener('click', (e) => {
      const trigger = e.target.closest('.dropdown-trigger');
      if (trigger) {
        e.preventDefault();
        e.stopPropagation();
        const dropdown = trigger.closest('.dropdown');
        if (dropdown) {
          dropdowns.toggle(dropdown);
        }
      }
    });

    // Close on outside click
    document.addEventListener('click', (e) => {
      if (!e.target.closest('.dropdown')) {
        dropdowns.closeAll();
      }
    });

    // Handle dropdown item clicks
    document.addEventListener('click', (e) => {
      const item = e.target.closest('.dropdown-item');
      if (item) {
        dropdowns.closeAll();
      }
    });
  }
};

// ========================================
// Sidebar Toggle
// ========================================

const sidebar = {
  toggle() {
    const el = document.querySelector('.sidebar');
    if (el) {
      el.classList.toggle('collapsed');
      localStorage.setItem('sidebar-collapsed', el.classList.contains('collapsed'));
    }
  },

  init() {
    // Restore state from localStorage
    const el = document.querySelector('.sidebar');
    if (el && localStorage.getItem('sidebar-collapsed') === 'true') {
      el.classList.add('collapsed');
    }

    // Toggle button
    document.addEventListener('click', (e) => {
      if (e.target.closest('.sidebar-toggle')) {
        sidebar.toggle();
      }
    });
  }
};

// ========================================
// Command Palette (Cmd+K)
// ========================================

const commandPalette = {
  el: null,
  input: null,
  results: null,
  selectedIndex: 0,
  items: [],
  issues: [],
  issuesLoaded: false,
  searchDebounce: null,

  commands: [
    { label: 'Go to Home', shortcut: 'G H', action: () => location.href = '/app', group: 'Navigation', icon: 'H' },
    { label: 'Go to Issues', shortcut: 'G I', action: () => location.href = location.pathname.replace(/\/board\/.*$/, '/issues'), group: 'Navigation', icon: 'I' },
    { label: 'Go to Board', shortcut: 'G B', action: () => {}, group: 'Navigation', icon: 'B' },
    { label: 'Go to Cycles', shortcut: 'G C', action: () => location.href = location.pathname.replace(/\/board\/.*$/, '/cycles'), group: 'Navigation', icon: 'C' },
    { label: 'Create new issue', shortcut: 'C', action: () => modals.open('create-issue-modal'), group: 'Actions', icon: '+' },
  ],

  async loadIssues() {
    if (this.issuesLoaded) return;
    try {
      // Get workspace from URL
      const match = location.pathname.match(/\/w\/([^\/]+)/);
      if (match) {
        const issues = await api.get(`/workspaces/${match[1]}/issues?limit=100`);
        this.issues = (issues || []).map(issue => ({
          label: `${issue.key}: ${issue.title}`,
          action: () => location.href = `/w/${match[1]}/issue/${issue.key}`,
          group: 'Issues',
          icon: issue.key.split('-')[0]?.charAt(0) || 'I',
          key: issue.key,
          title: issue.title,
        }));
        this.issuesLoaded = true;
      }
    } catch (error) {
      console.error('Failed to load issues for search:', error);
    }
  },

  open() {
    if (!this.el) return;
    this.el.classList.remove('hidden');
    this.input.value = '';
    this.input.focus();
    this.render('');
    document.body.style.overflow = 'hidden';
    // Load issues in background
    this.loadIssues();
  },

  close() {
    if (!this.el) return;
    this.el.classList.add('hidden');
    document.body.style.overflow = '';
    this.selectedIndex = 0;
  },

  render(query) {
    // Filter commands
    const filteredCommands = this.commands.filter(cmd =>
      cmd.label.toLowerCase().includes(query.toLowerCase())
    );

    // Filter issues if query is present
    let filteredIssues = [];
    if (query.length >= 1) {
      const lowerQuery = query.toLowerCase();
      filteredIssues = this.issues.filter(issue =>
        issue.key.toLowerCase().includes(lowerQuery) ||
        issue.title.toLowerCase().includes(lowerQuery)
      ).slice(0, 10); // Limit to 10 issue results
    }

    // Combine items: commands first, then issues
    const allItems = [...filteredCommands, ...filteredIssues];

    // Group by category
    const groups = {};
    allItems.forEach(item => {
      if (!groups[item.group]) groups[item.group] = [];
      groups[item.group].push(item);
    });

    this.items = allItems;
    this.selectedIndex = 0;

    let html = '';
    Object.entries(groups).forEach(([groupName, items]) => {
      html += `<div class="command-group">
        <div class="command-group-title">${groupName}</div>`;
      items.forEach((item, i) => {
        const globalIndex = allItems.indexOf(item);
        html += `<button class="command-item${globalIndex === this.selectedIndex ? ' selected' : ''}" data-index="${globalIndex}">
          <span class="command-icon">${item.icon}</span>
          <span class="command-label">${item.label}</span>
          ${item.shortcut ? `<span class="command-shortcut">${item.shortcut}</span>` : ''}
        </button>`;
      });
      html += '</div>';
    });

    this.results.innerHTML = html || '<div class="empty-state"><p class="text-muted">No results found</p></div>';
  },

  selectNext() {
    if (this.items.length === 0) return;
    this.selectedIndex = (this.selectedIndex + 1) % this.items.length;
    this.updateSelection();
  },

  selectPrev() {
    if (this.items.length === 0) return;
    this.selectedIndex = (this.selectedIndex - 1 + this.items.length) % this.items.length;
    this.updateSelection();
  },

  updateSelection() {
    this.results.querySelectorAll('.command-item').forEach((el, i) => {
      el.classList.toggle('selected', parseInt(el.dataset.index) === this.selectedIndex);
    });
  },

  execute() {
    if (this.items[this.selectedIndex]) {
      this.items[this.selectedIndex].action();
      this.close();
    }
  },

  init() {
    // Create command palette HTML if it doesn't exist
    if (!document.getElementById('command-palette')) {
      const html = `
        <div id="command-palette" class="modal hidden">
          <div class="modal-backdrop"></div>
          <div class="modal-content" style="max-width: 640px; padding: 0;">
            <input type="text" class="command-input" placeholder="Type a command or search...">
            <div class="command-results"></div>
          </div>
        </div>
      `;
      document.body.insertAdjacentHTML('beforeend', html);
    }

    this.el = document.getElementById('command-palette');
    if (!this.el) return;

    this.input = this.el.querySelector('.command-input');
    this.results = this.el.querySelector('.command-results');

    // Input handling
    this.input.addEventListener('input', (e) => {
      this.render(e.target.value);
    });

    // Keyboard navigation
    this.input.addEventListener('keydown', (e) => {
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          this.selectNext();
          break;
        case 'ArrowUp':
          e.preventDefault();
          this.selectPrev();
          break;
        case 'Enter':
          e.preventDefault();
          this.execute();
          break;
        case 'Escape':
          e.preventDefault();
          this.close();
          break;
      }
    });

    // Click on item
    this.results.addEventListener('click', (e) => {
      const item = e.target.closest('.command-item');
      if (item) {
        this.selectedIndex = parseInt(item.dataset.index);
        this.execute();
      }
    });

    // Cmd+K to open
    document.addEventListener('keydown', (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        this.open();
      }
    });

    // Search trigger click
    document.addEventListener('click', (e) => {
      if (e.target.closest('.search-trigger')) {
        this.open();
      }
    });
  }
};

// ========================================
// Kanban Board Drag & Drop
// ========================================

const kanban = {
  draggedCard: null,
  draggedColumn: null,

  init() {
    const board = document.getElementById('board');
    if (!board) return;

    // Make issue cards draggable
    board.querySelectorAll('.issue-card').forEach(card => {
      card.setAttribute('draggable', 'true');
    });

    // Drag start
    board.addEventListener('dragstart', (e) => {
      const card = e.target.closest('.issue-card');
      if (card) {
        this.draggedCard = card;
        card.setAttribute('dragging', '');
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', card.dataset.issueId);
      }
    });

    // Drag end
    board.addEventListener('dragend', (e) => {
      if (this.draggedCard) {
        this.draggedCard.removeAttribute('dragging');
        this.draggedCard = null;
      }

      // Remove all drag indicators
      board.querySelectorAll('.drag-over').forEach(el => {
        el.classList.remove('drag-over');
      });
      board.querySelectorAll('.drop-indicator').forEach(el => {
        el.remove();
      });
    });

    // Drag over column
    board.addEventListener('dragover', (e) => {
      e.preventDefault();
      const columnBody = e.target.closest('.column-body');
      if (columnBody && this.draggedCard) {
        e.dataTransfer.dropEffect = 'move';
        columnBody.classList.add('drag-over');

        // Find insertion point
        const cards = [...columnBody.querySelectorAll('.issue-card:not([dragging])')];
        const afterElement = cards.find(card => {
          const rect = card.getBoundingClientRect();
          return e.clientY < rect.top + rect.height / 2;
        });

        // Remove existing indicator
        columnBody.querySelectorAll('.drop-indicator').forEach(el => el.remove());

        // Add new indicator
        const indicator = document.createElement('div');
        indicator.className = 'drop-indicator';

        if (afterElement) {
          columnBody.insertBefore(indicator, afterElement);
        } else {
          columnBody.appendChild(indicator);
        }
      }
    });

    // Drag leave column
    board.addEventListener('dragleave', (e) => {
      const columnBody = e.target.closest('.column-body');
      if (columnBody && !columnBody.contains(e.relatedTarget)) {
        columnBody.classList.remove('drag-over');
        columnBody.querySelectorAll('.drop-indicator').forEach(el => el.remove());
      }
    });

    // Drop
    board.addEventListener('drop', async (e) => {
      e.preventDefault();
      const columnBody = e.target.closest('.column-body');

      if (!columnBody || !this.draggedCard) return;

      const column = columnBody.closest('.board-column');
      const columnId = column?.dataset.columnId;
      const issueId = this.draggedCard.dataset.issueId;
      const issueKey = this.draggedCard.dataset.issueKey;

      // Validate columnId before proceeding
      if (!columnId) return;

      // Calculate position
      const indicator = columnBody.querySelector('.drop-indicator');
      let position = 0;

      if (indicator) {
        const cards = [...columnBody.querySelectorAll('.issue-card:not([dragging])')];
        const nextCard = indicator.nextElementSibling;
        if (nextCard) {
          position = parseInt(nextCard.dataset.position || '0');
        } else {
          const lastCard = cards[cards.length - 1];
          position = (parseInt(lastCard?.dataset.position || '0') + 1000);
        }
      }

      // Move card in DOM (optimistic update)
      if (indicator) {
        columnBody.insertBefore(this.draggedCard, indicator);
        indicator.remove();
      } else {
        columnBody.appendChild(this.draggedCard);
      }

      // Update column count
      this.updateColumnCounts();

      // API call
      try {
        await api.post(`/issues/${issueKey}/move`, {
          column_id: columnId,
          position: position,
        });
      } catch (error) {
        console.error('Failed to move issue:', error);
        // Could revert the DOM change here
      }
    });
  },

  updateColumnCounts() {
    document.querySelectorAll('.board-column').forEach(column => {
      const count = column.querySelectorAll('.issue-card').length;
      const countEl = column.querySelector('.column-count');
      if (countEl) {
        countEl.textContent = count;
      }
    });
  }
};

// ========================================
// Quick Add Issue
// ========================================

const quickAdd = {
  init() {
    document.querySelectorAll('.quick-add-form').forEach(form => {
      const input = form.querySelector('input');
      if (!input) return;

      form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const title = input.value.trim();
        if (!title) return;

        const column = form.closest('.board-column');
        const columnId = column?.dataset.columnId;
        const projectId = document.getElementById('board')?.dataset.projectId;

        if (!projectId || !columnId) return;

        // Clear input immediately
        input.value = '';

        try {
          const issue = await api.post(`/projects/${projectId}/issues`, {
            title,
            column_id: columnId,
          });

          // Add card to column (optimistic would add before API call)
          const columnBody = column.querySelector('.column-body');
          const card = document.createElement('div');
          card.className = 'issue-card';
          card.draggable = true;
          card.dataset.issueId = issue.id;
          card.dataset.issueKey = issue.key;
          card.innerHTML = `
            <div class="issue-card-header">
              <span class="issue-key">${issue.key}</span>
            </div>
            <div class="issue-card-title">${title}</div>
          `;
          columnBody.appendChild(card);

          // Update count
          kanban.updateColumnCounts();
        } catch (error) {
          console.error('Failed to create issue:', error);
          input.value = title; // Restore input on error
        }
      });

      // Submit on Enter
      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          form.dispatchEvent(new Event('submit'));
        }
      });
    });
  }
};

// ========================================
// Form Handling
// ========================================

const forms = {
  init() {
    // Auth forms
    document.querySelectorAll('form[data-auth]').forEach(form => {
      form.addEventListener('submit', async (e) => {
        e.preventDefault();

        const submitBtn = form.querySelector('button[type="submit"]');
        const errorEl = form.querySelector('.alert-error');

        // Disable button
        if (submitBtn) {
          submitBtn.disabled = true;
          submitBtn.textContent = 'Loading...';
        }

        // Hide previous error
        if (errorEl) {
          errorEl.classList.add('hidden');
        }

        const formData = new FormData(form);
        const data = Object.fromEntries(formData);
        const action = form.dataset.auth;

        try {
          await api.post(`/auth/${action}`, data);

          // Redirect on success
          if (action === 'login' || action === 'register') {
            location.href = '/app';
          } else if (action === 'logout') {
            location.href = '/login';
          }
        } catch (error) {
          // Show error
          if (errorEl) {
            errorEl.textContent = error.message || 'An error occurred';
            errorEl.classList.remove('hidden');
          }

          // Re-enable button
          if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.textContent = action === 'login' ? 'Sign in' : 'Create account';
          }
        }
      });
    });

    // Create issue modal form
    const createIssueForm = document.getElementById('create-issue-form');
    if (createIssueForm) {
      createIssueForm.addEventListener('submit', async (e) => {
        e.preventDefault();

        const formData = new FormData(createIssueForm);
        const data = {
          title: formData.get('title'),
          column_id: formData.get('column_id'),
        };

        const projectId = createIssueForm.dataset.projectId;

        try {
          const issue = await api.post(`/projects/${projectId}/issues`, data);
          modals.close('create-issue-modal');

          // Reload page or add issue to DOM
          location.reload();
        } catch (error) {
          alert(error.message || 'Failed to create issue');
        }
      });
    }

    // Add column modal form
    const addColumnForm = document.getElementById('add-column-form');
    if (addColumnForm) {
      addColumnForm.addEventListener('submit', async (e) => {
        e.preventDefault();

        const formData = new FormData(addColumnForm);
        const data = {
          name: formData.get('name'),
        };

        const projectId = addColumnForm.dataset.projectId;

        try {
          await api.post(`/projects/${projectId}/columns`, data);
          modals.close('add-column-modal');
          location.reload();
        } catch (error) {
          alert(error.message || 'Failed to add column');
        }
      });
    }
  }
};

// ========================================
// Inline Editing
// ========================================

const inlineEdit = {
  init() {
    // Editable issue title
    document.querySelectorAll('[contenteditable="true"]').forEach(el => {
      let originalValue = el.textContent;

      el.addEventListener('focus', () => {
        originalValue = el.textContent;
      });

      el.addEventListener('blur', async () => {
        const newValue = el.textContent.trim();
        if (newValue === originalValue) return;

        const issueKey = el.dataset.issueKey;
        const field = el.dataset.field || 'title';

        if (issueKey) {
          try {
            await api.patch(`/issues/${issueKey}`, { [field]: newValue });
          } catch (error) {
            console.error('Failed to update:', error);
            el.textContent = originalValue; // Revert
          }
        }
      });

      // Save on Enter, cancel on Escape
      el.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
          e.preventDefault();
          el.blur();
        } else if (e.key === 'Escape') {
          el.textContent = originalValue;
          el.blur();
        }
      });
    });
  }
};

// ========================================
// Comments
// ========================================

const comments = {
  init() {
    const form = document.querySelector('.comment-form');
    if (!form) return;

    form.addEventListener('submit', async (e) => {
      e.preventDefault();

      const textarea = form.querySelector('textarea[name="body"]');
      const submitBtn = form.querySelector('button[type="submit"]');
      const content = textarea.value.trim();

      if (!content) return;

      const issueId = form.dataset.issueId;

      submitBtn.disabled = true;

      try {
        const comment = await api.post(`/issues/${issueId}/comments`, { content });

        // Add comment to list
        const list = document.querySelector('.comment-list');
        if (list) {
          const html = `
            <div class="comment-item flex gap-3">
              <div class="avatar avatar-sm">${comment.author?.display_name?.charAt(0) || 'U'}</div>
              <div class="comment-content">
                <div class="comment-header">
                  <span class="comment-author">${comment.author?.display_name || 'User'}</span>
                  <span class="comment-time">Just now</span>
                </div>
                <div class="comment-body">${content}</div>
              </div>
            </div>
          `;
          list.insertAdjacentHTML('beforeend', html);
        }

        textarea.value = '';
      } catch (error) {
        alert(error.message || 'Failed to add comment');
      } finally {
        submitBtn.disabled = false;
      }
    });
  }
};

// ========================================
// Issue Detail - Property Changes
// ========================================

const issueProperties = {
  init() {
    // Status select
    document.getElementById('issue-status')?.addEventListener('change', async (e) => {
      const issueKey = e.target.dataset.issueKey;
      const columnId = e.target.value;

      try {
        await api.post(`/issues/${issueKey}/move`, { column_id: columnId });
      } catch (error) {
        console.error('Failed to update status:', error);
      }
    });

    // Cycle select
    document.getElementById('issue-cycle')?.addEventListener('change', async (e) => {
      const issueKey = e.target.dataset.issueKey;
      const cycleId = e.target.value;

      try {
        if (cycleId) {
          await api.post(`/issues/${issueKey}/cycle`, { cycle_id: cycleId });
        } else {
          await api.delete(`/issues/${issueKey}/cycle`);
        }
      } catch (error) {
        console.error('Failed to update cycle:', error);
      }
    });
  }
};

// ========================================
// Delete Issue
// ========================================

const deleteIssue = {
  init() {
    document.addEventListener('click', async (e) => {
      const btn = e.target.closest('[data-action="delete-issue"]');
      if (!btn) return;

      const issueKey = btn.dataset.issueKey;
      if (!issueKey) return;

      if (!confirm('Are you sure you want to delete this issue?')) return;

      try {
        await api.delete(`/issues/${issueKey}`);
        // Navigate back to issues list
        const workspace = location.pathname.split('/')[2];
        location.href = `/w/${workspace}/issues`;
      } catch (error) {
        alert(error.message || 'Failed to delete issue');
      }
    });
  }
};

// ========================================
// Keyboard Navigation
// ========================================

const keyboard = {
  init() {
    document.addEventListener('keydown', (e) => {
      // Ignore if typing in input
      if (e.target.matches('input, textarea, [contenteditable]')) return;

      // Ignore if modal is open (except command palette keys)
      const modalOpen = document.querySelector('.modal:not(.hidden):not(#command-palette)');
      if (modalOpen) return;

      switch (e.key) {
        case 'c':
          // Create new issue
          if (!e.metaKey && !e.ctrlKey) {
            e.preventDefault();
            modals.open('create-issue-modal');
          }
          break;

        case 'g':
          // Start navigation sequence
          this.awaitingNav = true;
          setTimeout(() => { this.awaitingNav = false; }, 1000);
          break;

        case 'h':
          if (this.awaitingNav) {
            e.preventDefault();
            location.href = '/app';
          }
          break;

        case 'i':
          if (this.awaitingNav) {
            e.preventDefault();
            // Navigate to issues
            const path = location.pathname;
            const match = path.match(/\/w\/([^\/]+)/);
            if (match) {
              location.href = `/w/${match[1]}/issues`;
            }
          }
          break;

        case '?':
          // Show keyboard shortcuts help
          if (!e.shiftKey) return;
          // Could open a help modal here
          break;
      }
    });
  },

  awaitingNav: false
};

// ========================================
// Issue Card Click Navigation
// ========================================

const issueNavigation = {
  init() {
    document.addEventListener('click', (e) => {
      const card = e.target.closest('.issue-card');
      if (!card) return;

      // Don't navigate if clicking a button inside the card
      if (e.target.closest('button, .dropdown')) return;

      const issueKey = card.dataset.issueKey;
      if (!issueKey) return;

      // Get workspace from URL
      const path = location.pathname;
      const match = path.match(/\/w\/([^\/]+)/);
      if (match) {
        location.href = `/w/${match[1]}/issue/${issueKey}`;
      }
    });

    // Table row clicks
    document.addEventListener('click', (e) => {
      const row = e.target.closest('.table tbody tr');
      if (!row) return;

      const keyEl = row.querySelector('.issue-key');
      if (!keyEl) return;

      const issueKey = keyEl.textContent.trim();
      const path = location.pathname;
      const match = path.match(/\/w\/([^\/]+)/);
      if (match) {
        location.href = `/w/${match[1]}/issue/${issueKey}`;
      }
    });
  }
};

// ========================================
// Logout Handler
// ========================================

const logout = {
  init() {
    document.addEventListener('click', async (e) => {
      const btn = e.target.closest('[data-action="logout"]');
      if (!btn) return;

      e.preventDefault();

      try {
        await api.post('/auth/logout');
        location.href = '/login';
      } catch (error) {
        console.error('Logout failed:', error);
        location.href = '/login';
      }
    });
  }
};

// ========================================
// Initialize Everything
// ========================================

document.addEventListener('DOMContentLoaded', () => {
  modals.init();
  dropdowns.init();
  sidebar.init();
  commandPalette.init();
  kanban.init();
  quickAdd.init();
  forms.init();
  inlineEdit.init();
  comments.init();
  issueProperties.init();
  deleteIssue.init();
  keyboard.init();
  issueNavigation.init();
  logout.init();
});

// Export for use in templates
window.app = {
  api,
  modals,
  dropdowns,
  sidebar,
  commandPalette,
  kanban,
};
