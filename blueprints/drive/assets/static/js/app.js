// Drive Frontend JavaScript

(function() {
  'use strict';

  // State
  let selectedItems = new Set();
  let currentPath = '';
  let clipboard = { items: [], action: null };

  // DOM Ready
  document.addEventListener('DOMContentLoaded', init);

  function init() {
    setupDropdowns();
    setupModals();
    setupFileSelection();
    setupDragAndDrop();
    setupContextMenu();
    setupKeyboardShortcuts();
    setupForms();
    setupNewMenu();
  }

  // Dropdowns
  function setupDropdowns() {
    document.querySelectorAll('[data-action="toggle-dropdown"]').forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const dropdown = btn.closest('.relative, [id$="-dropdown"]');
        const menu = dropdown.querySelector('.dropdown-menu');
        if (menu) {
          const wasHidden = menu.classList.contains('hidden');
          closeAllDropdowns();
          if (wasHidden) {
            menu.classList.remove('hidden');
          }
        }
      });
    });

    // Also handle dropdown triggers
    document.querySelectorAll('.dropdown-trigger').forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const menu = btn.nextElementSibling;
        if (menu && menu.classList.contains('dropdown-menu')) {
          const wasHidden = menu.classList.contains('hidden');
          closeAllDropdowns();
          if (wasHidden) {
            menu.classList.remove('hidden');
          }
        }
      });
    });

    // Close dropdowns on click outside
    document.addEventListener('click', closeAllDropdowns);
  }

  function closeAllDropdowns() {
    document.querySelectorAll('.dropdown-menu').forEach(menu => {
      menu.classList.add('hidden');
    });
  }

  // Modals
  function setupModals() {
    // Close modal buttons
    document.querySelectorAll('[data-action="close-modal"]').forEach(btn => {
      btn.addEventListener('click', () => {
        const modal = btn.closest('.fixed');
        if (modal) {
          modal.classList.add('hidden');
        }
      });
    });

    // Close on escape
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        document.querySelectorAll('.fixed:not(.hidden)').forEach(modal => {
          if (modal.id && modal.id.includes('modal')) {
            modal.classList.add('hidden');
          }
        });
        closeAllDropdowns();
      }
    });
  }

  function openModal(id) {
    const modal = document.getElementById(id);
    if (modal) {
      modal.classList.remove('hidden');
      const input = modal.querySelector('input:not([type="hidden"])');
      if (input) {
        setTimeout(() => input.focus(), 50);
      }
    }
  }

  // New Menu
  function setupNewMenu() {
    const newBtn = document.getElementById('new-btn');
    const newMenu = document.getElementById('new-menu');

    if (newBtn && newMenu) {
      newBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        const rect = newBtn.getBoundingClientRect();
        newMenu.style.top = (rect.bottom + 8) + 'px';
        newMenu.style.left = rect.left + 'px';
        newMenu.classList.toggle('hidden');
      });
    }

    // New folder action
    document.querySelectorAll('[data-action="new-folder"]').forEach(btn => {
      btn.addEventListener('click', () => {
        closeAllDropdowns();
        openModal('folder-modal');
      });
    });

    // Upload file action
    document.querySelectorAll('[data-action="upload-file"]').forEach(btn => {
      btn.addEventListener('click', () => {
        closeAllDropdowns();
        openModal('upload-modal');
      });
    });
  }

  // File Selection
  function setupFileSelection() {
    document.querySelectorAll('[data-type="file"], [data-type="folder"]').forEach(item => {
      item.addEventListener('click', (e) => {
        if (e.target.closest('button') || e.target.closest('a')) return;

        const id = item.dataset.id;
        const type = item.dataset.type;

        if (e.ctrlKey || e.metaKey) {
          // Multi-select
          if (selectedItems.has(id)) {
            selectedItems.delete(id);
            item.classList.remove('ring-2', 'ring-zinc-900');
          } else {
            selectedItems.add(id);
            item.classList.add('ring-2', 'ring-zinc-900');
          }
        } else if (e.shiftKey) {
          // Range select (TODO)
        } else {
          // Single select
          clearSelection();
          if (type === 'folder') {
            // Navigate to folder
            const link = item.querySelector('a');
            if (link) {
              window.location.href = link.href;
            } else {
              window.location.href = '/files/' + id;
            }
          } else {
            // Select file
            selectedItems.add(id);
            item.classList.add('ring-2', 'ring-zinc-900');
          }
        }
      });

      // Double-click to open/preview
      item.addEventListener('dblclick', () => {
        const type = item.dataset.type;
        const id = item.dataset.id;
        if (type === 'folder') {
          window.location.href = '/files/' + id;
        } else {
          // Open preview page
          window.location.href = '/preview/' + id;
        }
      });
    });
  }

  function clearSelection() {
    selectedItems.clear();
    document.querySelectorAll('[data-type="file"], [data-type="folder"]').forEach(item => {
      item.classList.remove('ring-2', 'ring-zinc-900');
    });
  }

  // Drag and Drop Upload
  function setupDragAndDrop() {
    const dropZone = document.getElementById('drop-zone');
    const fileInput = document.getElementById('file-input');
    const uploadList = document.getElementById('upload-list');
    const uploadSubmit = document.getElementById('upload-submit');

    if (!dropZone) return;

    // Click to browse
    dropZone.addEventListener('click', () => {
      fileInput.click();
    });

    // File input change
    fileInput.addEventListener('change', () => {
      handleFiles(fileInput.files);
    });

    // Drag events
    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
      dropZone.addEventListener(eventName, preventDefaults);
    });

    ['dragenter', 'dragover'].forEach(eventName => {
      dropZone.addEventListener(eventName, () => {
        dropZone.classList.add('border-zinc-500', 'bg-zinc-50');
      });
    });

    ['dragleave', 'drop'].forEach(eventName => {
      dropZone.addEventListener(eventName, () => {
        dropZone.classList.remove('border-zinc-500', 'bg-zinc-50');
      });
    });

    dropZone.addEventListener('drop', (e) => {
      handleFiles(e.dataTransfer.files);
    });

    // Main content drop zone
    const mainContent = document.querySelector('main');
    if (mainContent) {
      ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        mainContent.addEventListener(eventName, preventDefaults);
      });

      mainContent.addEventListener('drop', (e) => {
        openModal('upload-modal');
        handleFiles(e.dataTransfer.files);
      });
    }

    function handleFiles(files) {
      if (!files || files.length === 0) return;

      uploadList.classList.remove('hidden');
      uploadList.innerHTML = '';

      Array.from(files).forEach(file => {
        const item = document.createElement('div');
        item.className = 'flex items-center gap-3 p-2 bg-zinc-50 rounded-lg';
        item.innerHTML = `
          <svg class="w-5 h-5 text-zinc-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
            <polyline points="14 2 14 8 20 8"/>
          </svg>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-zinc-900 truncate">${escapeHtml(file.name)}</p>
            <p class="text-xs text-zinc-500">${formatSize(file.size)}</p>
          </div>
          <button type="button" class="text-zinc-400 hover:text-zinc-600" data-action="remove-file">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 6 6 18M6 6l12 12"/>
            </svg>
          </button>
        `;
        uploadList.appendChild(item);

        item.querySelector('[data-action="remove-file"]').addEventListener('click', () => {
          item.remove();
          if (uploadList.children.length === 0) {
            uploadList.classList.add('hidden');
            uploadSubmit.disabled = true;
          }
        });
      });

      uploadSubmit.disabled = false;
    }

    // Upload submit
    if (uploadSubmit) {
      uploadSubmit.addEventListener('click', async () => {
        const files = fileInput.files;
        if (!files || files.length === 0) return;

        uploadSubmit.disabled = true;
        uploadSubmit.textContent = 'Uploading...';

        // Get current path from URL
        const path = window.location.pathname.replace('/files', '').replace(/^\//, '');

        for (const file of files) {
          const formData = new FormData();
          formData.append('file', file);
          if (path) {
            formData.append('parent_id', path);
          }

          try {
            await fetch('/api/v1/files', {
              method: 'POST',
              body: formData,
            });
          } catch (err) {
            console.error('Upload failed:', err);
          }
        }

        // Refresh page
        window.location.reload();
      });
    }
  }

  function preventDefaults(e) {
    e.preventDefault();
    e.stopPropagation();
  }

  // Context Menu
  function setupContextMenu() {
    const contextMenu = document.getElementById('context-menu');
    if (!contextMenu) return;

    let currentItem = null;

    // Right-click on items
    document.querySelectorAll('[data-type="file"], [data-type="folder"]').forEach(item => {
      item.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        currentItem = item;
        showContextMenu(e.clientX, e.clientY);
      });
    });

    // Context menu button
    document.querySelectorAll('[data-action="context-menu"]').forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const item = btn.closest('[data-type="file"], [data-type="folder"]');
        if (item) {
          currentItem = item;
          const rect = btn.getBoundingClientRect();
          showContextMenu(rect.right, rect.bottom);
        }
      });
    });

    function showContextMenu(x, y) {
      // Position menu
      contextMenu.style.left = x + 'px';
      contextMenu.style.top = y + 'px';

      // Adjust if off screen
      const rect = contextMenu.getBoundingClientRect();
      if (rect.right > window.innerWidth) {
        contextMenu.style.left = (x - rect.width) + 'px';
      }
      if (rect.bottom > window.innerHeight) {
        contextMenu.style.top = (y - rect.height) + 'px';
      }

      contextMenu.classList.remove('hidden');
    }

    // Close on click
    document.addEventListener('click', () => {
      contextMenu.classList.add('hidden');
      currentItem = null;
    });

    // Context menu actions
    contextMenu.querySelectorAll('[data-action]').forEach(btn => {
      btn.addEventListener('click', () => {
        if (!currentItem) return;

        const action = btn.dataset.action;
        const id = currentItem.dataset.id;
        const type = currentItem.dataset.type;
        const name = currentItem.dataset.name;

        handleAction(action, { id, type, name });
        contextMenu.classList.add('hidden');
      });
    });
  }

  function handleAction(action, item) {
    switch (action) {
      case 'preview':
        if (item.type === 'folder') {
          window.location.href = '/files/' + item.id;
        } else {
          // Open preview page
          window.location.href = '/preview/' + item.id;
        }
        break;

      case 'open':
        if (item.type === 'folder') {
          window.open('/files/' + item.id, '_blank');
        } else {
          // Open file content in new tab
          window.open('/api/v1/content/' + item.id, '_blank');
        }
        break;

      case 'download':
        if (item.type === 'file') {
          // Create a temporary link to trigger download
          const a = document.createElement('a');
          a.href = '/api/v1/content/' + item.id;
          a.download = item.name;
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
        }
        break;

      case 'rename':
        const newName = prompt('Enter new name:', item.name);
        if (newName && newName !== item.name) {
          const endpoint = item.type === 'folder' ? '/api/v1/folders/' : '/api/v1/files/';
          fetch(endpoint + item.id, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: newName }),
          }).then(() => window.location.reload());
        }
        break;

      case 'star':
        const starEndpoint = item.type === 'folder' ? '/api/v1/folders/' : '/api/v1/files/';
        fetch(starEndpoint + item.id + '/star', { method: 'PUT' })
          .then(() => window.location.reload());
        break;

      case 'trash':
        if (confirm('Move "' + item.name + '" to trash?')) {
          const trashEndpoint = item.type === 'folder' ? '/api/v1/folders/' : '/api/v1/files/';
          fetch(trashEndpoint + item.id + '/trash', { method: 'POST' })
            .then(() => window.location.reload());
        }
        break;

      case 'copy-link':
        const url = window.location.origin + '/files/' + item.id;
        navigator.clipboard.writeText(url);
        // TODO: Show toast notification
        break;

      case 'restore':
        const restoreEndpoint = item.type === 'folder' ? '/api/v1/folders/' : '/api/v1/files/';
        fetch(restoreEndpoint + item.id + '/restore', { method: 'POST' })
          .then(() => window.location.reload());
        break;

      case 'delete-permanent':
        if (confirm('Permanently delete "' + item.name + '"? This cannot be undone.')) {
          const deleteEndpoint = item.type === 'folder' ? '/api/v1/folders/' : '/api/v1/files/';
          fetch(deleteEndpoint + item.id, { method: 'DELETE' })
            .then(() => window.location.reload());
        }
        break;
    }
  }

  // Keyboard Shortcuts
  function setupKeyboardShortcuts() {
    document.addEventListener('keydown', (e) => {
      // Ignore if in input/textarea
      if (e.target.matches('input, textarea, select')) return;

      switch (e.key) {
        case 'n':
          if (!e.ctrlKey && !e.metaKey) {
            e.preventDefault();
            openModal('folder-modal');
          }
          break;

        case 'u':
          if (!e.ctrlKey && !e.metaKey) {
            e.preventDefault();
            openModal('upload-modal');
          }
          break;

        case '/':
          e.preventDefault();
          document.querySelector('input[name="q"]')?.focus();
          break;

        case 'Delete':
        case 'Backspace':
          if (selectedItems.size > 0 && !e.target.matches('input, textarea')) {
            e.preventDefault();
            // Trash selected items
          }
          break;

        case 'a':
          if (e.ctrlKey || e.metaKey) {
            e.preventDefault();
            document.querySelectorAll('[data-type="file"], [data-type="folder"]').forEach(item => {
              selectedItems.add(item.dataset.id);
              item.classList.add('ring-2', 'ring-zinc-900');
            });
          }
          break;

        case 'Escape':
          clearSelection();
          break;

        case 'F2':
          if (selectedItems.size === 1) {
            e.preventDefault();
            const id = Array.from(selectedItems)[0];
            const item = document.querySelector(`[data-id="${id}"]`);
            if (item) {
              handleAction('rename', {
                id: item.dataset.id,
                type: item.dataset.type,
                name: item.dataset.name,
              });
            }
          }
          break;

        case 'Enter':
        case 'p':
          // Preview selected file
          if (selectedItems.size === 1) {
            e.preventDefault();
            const id = Array.from(selectedItems)[0];
            const item = document.querySelector(`[data-id="${id}"]`);
            if (item) {
              handleAction('preview', {
                id: item.dataset.id,
                type: item.dataset.type,
                name: item.dataset.name,
              });
            }
          }
          break;
      }
    });
  }

  // Forms
  function setupForms() {
    // Login form
    const loginForm = document.getElementById('login-form');
    if (loginForm) {
      loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(loginForm);
        const errorDiv = document.getElementById('error-message');

        try {
          const res = await fetch('/api/v1/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              email: formData.get('email'),
              password: formData.get('password'),
            }),
          });

          if (res.ok) {
            window.location.href = '/files';
          } else {
            const data = await res.json();
            errorDiv.textContent = data.error || 'Login failed';
            errorDiv.classList.remove('hidden');
          }
        } catch (err) {
          errorDiv.textContent = 'An error occurred';
          errorDiv.classList.remove('hidden');
        }
      });
    }

    // Register form
    const registerForm = document.getElementById('register-form');
    if (registerForm) {
      registerForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(registerForm);
        const errorDiv = document.getElementById('error-message');

        try {
          const res = await fetch('/api/v1/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              name: formData.get('name'),
              email: formData.get('email'),
              password: formData.get('password'),
            }),
          });

          if (res.ok) {
            window.location.href = '/files';
          } else {
            const data = await res.json();
            errorDiv.textContent = data.error || 'Registration failed';
            errorDiv.classList.remove('hidden');
          }
        } catch (err) {
          errorDiv.textContent = 'An error occurred';
          errorDiv.classList.remove('hidden');
        }
      });
    }

    // New folder form
    const folderForm = document.getElementById('folder-form');
    if (folderForm) {
      folderForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(folderForm);
        const name = formData.get('name');
        const path = window.location.pathname.replace('/files', '').replace(/^\//, '');

        try {
          await fetch('/api/v1/folders', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              name: name,
              parent_id: path || undefined,
            }),
          });
          window.location.reload();
        } catch (err) {
          console.error('Failed to create folder:', err);
        }
      });
    }

    // Empty trash
    document.querySelectorAll('[data-action="empty-trash"]').forEach(btn => {
      btn.addEventListener('click', async () => {
        if (confirm('Permanently delete all items in trash? This cannot be undone.')) {
          try {
            await fetch('/api/v1/trash', { method: 'DELETE' });
            window.location.reload();
          } catch (err) {
            console.error('Failed to empty trash:', err);
          }
        }
      });
    });

    // Logout
    document.querySelectorAll('[data-action="logout"]').forEach(btn => {
      btn.addEventListener('click', async () => {
        try {
          await fetch('/api/v1/auth/logout', { method: 'POST' });
          window.location.href = '/login';
        } catch (err) {
          console.error('Logout failed:', err);
        }
      });
    });
  }

  // Helpers
  function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }
})();
