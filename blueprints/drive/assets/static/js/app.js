// Drive JavaScript

class Drive {
  constructor() {
    this.currentFolder = null;
    this.selectedItems = new Set();
    this.viewMode = 'grid'; // 'grid' or 'list'
    this.init();
  }

  init() {
    this.setupEventListeners();
    this.loadFiles();
  }

  setupEventListeners() {
    // Upload area drag and drop
    const uploadArea = document.querySelector('.upload-area');
    if (uploadArea) {
      uploadArea.addEventListener('dragover', (e) => {
        e.preventDefault();
        uploadArea.classList.add('drag-over');
      });

      uploadArea.addEventListener('dragleave', () => {
        uploadArea.classList.remove('drag-over');
      });

      uploadArea.addEventListener('drop', (e) => {
        e.preventDefault();
        uploadArea.classList.remove('drag-over');
        this.handleFileUpload(e.dataTransfer.files);
      });

      uploadArea.addEventListener('click', () => {
        const input = document.createElement('input');
        input.type = 'file';
        input.multiple = true;
        input.onchange = (e) => this.handleFileUpload(e.target.files);
        input.click();
      });
    }

    // File selection
    document.addEventListener('click', (e) => {
      const fileItem = e.target.closest('.file-item, .file-row');
      if (fileItem) {
        this.selectItem(fileItem, e);
      }
    });

    // Context menu
    document.addEventListener('contextmenu', (e) => {
      const fileItem = e.target.closest('.file-item, .file-row');
      if (fileItem) {
        e.preventDefault();
        this.showContextMenu(e, fileItem);
      }
    });

    // Close context menu
    document.addEventListener('click', () => {
      this.hideContextMenu();
    });

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Delete' && this.selectedItems.size > 0) {
        this.deleteSelected();
      }
      if (e.key === 'Escape') {
        this.selectedItems.clear();
        this.updateSelection();
      }
    });
  }

  async loadFiles(folderId = null) {
    try {
      const url = folderId ? `/api/v1/files?parent_id=${folderId}` : '/api/v1/files';
      const response = await fetch(url, {
        credentials: 'include'
      });

      if (response.ok) {
        const files = await response.json();
        this.renderFiles(files);
      }
    } catch (error) {
      this.showToast('Failed to load files', 'error');
    }
  }

  renderFiles(files) {
    const container = document.querySelector('.file-container');
    if (!container) return;

    if (this.viewMode === 'grid') {
      container.innerHTML = `
        <div class="file-grid">
          ${files.map(file => this.renderFileGridItem(file)).join('')}
        </div>
      `;
    } else {
      container.innerHTML = `
        <div class="file-list">
          ${files.map(file => this.renderFileListItem(file)).join('')}
        </div>
      `;
    }
  }

  renderFileGridItem(file) {
    const icon = this.getFileIcon(file.mime_type);
    return `
      <div class="file-item" data-id="${file.id}" data-type="file">
        <div class="file-icon">${icon}</div>
        <div class="file-name">${file.name}</div>
        <div class="file-meta">${this.formatSize(file.size)}</div>
      </div>
    `;
  }

  renderFileListItem(file) {
    const icon = this.getFileIcon(file.mime_type);
    return `
      <div class="file-row" data-id="${file.id}" data-type="file">
        <div class="file-icon">${icon}</div>
        <div class="file-name" style="flex: 1">${file.name}</div>
        <div class="file-meta">${this.formatSize(file.size)}</div>
        <div class="file-meta">${this.formatDate(file.updated_at)}</div>
      </div>
    `;
  }

  getFileIcon(mimeType) {
    if (mimeType.startsWith('image/')) return '\uD83D\uDDBC\uFE0F';
    if (mimeType.startsWith('video/')) return '\uD83C\uDFAC';
    if (mimeType.startsWith('audio/')) return '\uD83C\uDFB5';
    if (mimeType.includes('pdf')) return '\uD83D\uDCC4';
    if (mimeType.includes('document')) return '\uD83D\uDCC4';
    if (mimeType.includes('spreadsheet')) return '\uD83D\uDCCA';
    if (mimeType.includes('presentation')) return '\uD83D\uDCC8';
    return '\uD83D\uDCC1';
  }

  formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString();
  }

  selectItem(item, event) {
    const id = item.dataset.id;

    if (event.ctrlKey || event.metaKey) {
      if (this.selectedItems.has(id)) {
        this.selectedItems.delete(id);
      } else {
        this.selectedItems.add(id);
      }
    } else {
      this.selectedItems.clear();
      this.selectedItems.add(id);
    }

    this.updateSelection();
  }

  updateSelection() {
    document.querySelectorAll('.file-item, .file-row').forEach(item => {
      if (this.selectedItems.has(item.dataset.id)) {
        item.classList.add('selected');
      } else {
        item.classList.remove('selected');
      }
    });
  }

  showContextMenu(event, item) {
    this.hideContextMenu();

    const menu = document.createElement('div');
    menu.className = 'context-menu';
    menu.innerHTML = `
      <div class="context-menu-item" data-action="open">Open</div>
      <div class="context-menu-item" data-action="download">Download</div>
      <div class="context-menu-separator"></div>
      <div class="context-menu-item" data-action="rename">Rename</div>
      <div class="context-menu-item" data-action="move">Move to</div>
      <div class="context-menu-item" data-action="copy">Copy</div>
      <div class="context-menu-separator"></div>
      <div class="context-menu-item" data-action="share">Share</div>
      <div class="context-menu-item" data-action="star">Add to starred</div>
      <div class="context-menu-separator"></div>
      <div class="context-menu-item" data-action="trash">Move to trash</div>
    `;

    menu.style.left = `${event.clientX}px`;
    menu.style.top = `${event.clientY}px`;

    menu.querySelectorAll('.context-menu-item').forEach(menuItem => {
      menuItem.addEventListener('click', (e) => {
        e.stopPropagation();
        this.handleContextAction(menuItem.dataset.action, item.dataset.id);
        this.hideContextMenu();
      });
    });

    document.body.appendChild(menu);
  }

  hideContextMenu() {
    const existing = document.querySelector('.context-menu');
    if (existing) {
      existing.remove();
    }
  }

  async handleContextAction(action, id) {
    switch (action) {
      case 'open':
        window.location.href = `/files/${id}`;
        break;
      case 'download':
        window.location.href = `/api/v1/files/${id}/download`;
        break;
      case 'rename':
        this.showRenameModal(id);
        break;
      case 'share':
        this.showShareModal(id);
        break;
      case 'star':
        await this.starFile(id);
        break;
      case 'trash':
        await this.trashFile(id);
        break;
    }
  }

  async handleFileUpload(files) {
    for (const file of files) {
      await this.uploadFile(file);
    }
    this.loadFiles(this.currentFolder);
  }

  async uploadFile(file) {
    try {
      const formData = new FormData();
      formData.append('file', file);
      if (this.currentFolder) {
        formData.append('parent_id', this.currentFolder);
      }

      const response = await fetch('/api/v1/files', {
        method: 'POST',
        body: formData,
        credentials: 'include'
      });

      if (response.ok) {
        this.showToast(`Uploaded ${file.name}`, 'success');
      } else {
        this.showToast(`Failed to upload ${file.name}`, 'error');
      }
    } catch (error) {
      this.showToast(`Failed to upload ${file.name}`, 'error');
    }
  }

  async starFile(id) {
    try {
      await fetch(`/api/v1/files/${id}/star`, {
        method: 'PUT',
        credentials: 'include'
      });
      this.showToast('Added to starred', 'success');
    } catch (error) {
      this.showToast('Failed to star file', 'error');
    }
  }

  async trashFile(id) {
    try {
      await fetch(`/api/v1/files/${id}/trash`, {
        method: 'POST',
        credentials: 'include'
      });
      this.showToast('Moved to trash', 'success');
      this.loadFiles(this.currentFolder);
    } catch (error) {
      this.showToast('Failed to move to trash', 'error');
    }
  }

  async deleteSelected() {
    for (const id of this.selectedItems) {
      await this.trashFile(id);
    }
    this.selectedItems.clear();
  }

  showRenameModal(id) {
    // TODO: Implement rename modal
    const name = prompt('Enter new name:');
    if (name) {
      this.renameFile(id, name);
    }
  }

  async renameFile(id, name) {
    try {
      await fetch(`/api/v1/files/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ name }),
        credentials: 'include'
      });
      this.showToast('File renamed', 'success');
      this.loadFiles(this.currentFolder);
    } catch (error) {
      this.showToast('Failed to rename file', 'error');
    }
  }

  showShareModal(id) {
    // TODO: Implement share modal
    alert('Share modal - TODO');
  }

  showToast(message, type = 'info') {
    let container = document.querySelector('.toast-container');
    if (!container) {
      container = document.createElement('div');
      container.className = 'toast-container';
      document.body.appendChild(container);
    }

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    container.appendChild(toast);

    setTimeout(() => {
      toast.remove();
    }, 3000);
  }
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
  window.drive = new Drive();
});
