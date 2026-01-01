/**
 * Drive - Dropbox-style File Management Application
 * Complete frontend implementation with all features
 */

const API_BASE = '/api/v1';

// ============================================
// State Management
// ============================================

const state = {
  currentFolder: null,
  selectedItems: new Set(),
  viewMode: localStorage.getItem('viewMode') || 'grid',
  sortBy: localStorage.getItem('sortBy') || 'name',
  sortOrder: localStorage.getItem('sortOrder') || 'asc',
  theme: localStorage.getItem('theme') || 'light',
  sidebarCollapsed: localStorage.getItem('sidebarCollapsed') === 'true',
  uploads: new Map(),
  previewFile: null,
};

// ============================================
// API Client
// ============================================

const api = {
  async request(method, path, data = null, options = {}) {
    const config = {
      method,
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      ...options,
    };

    if (data && method !== 'GET') {
      config.body = JSON.stringify(data);
    }

    const response = await fetch(`${API_BASE}${path}`, config);
    const json = await response.json();

    if (!response.ok) {
      throw new Error(json.error?.message || 'Request failed');
    }

    return json.data;
  },

  get: (path) => api.request('GET', path),
  post: (path, data) => api.request('POST', path, data),
  patch: (path, data) => api.request('PATCH', path, data),
  delete: (path) => api.request('DELETE', path),

  // File operations
  files: {
    list: (folderId = '', params = {}) => {
      const query = new URLSearchParams({ folder_id: folderId, ...params });
      return api.get(`/files?${query}`);
    },
    get: (id) => api.get(`/files/${id}`),
    upload: (file, folderId, onProgress) => uploadFile(file, folderId, onProgress),
    update: (id, data) => api.patch(`/files/${id}`, data),
    delete: (id) => api.delete(`/files/${id}`),
    star: (id, starred) => api.post(`/files/${id}/star`, { starred }),
    move: (id, folderId) => api.post(`/files/${id}/move`, { folder_id: folderId }),
    copy: (id, folderId, name) => api.post(`/files/${id}/copy`, { folder_id: folderId, name }),
    versions: (id) => api.get(`/files/${id}/versions`),
    restoreVersion: (id, version) => api.post(`/files/${id}/versions/${version}/restore`),
  },

  // Folder operations
  folders: {
    list: (parentId = '') => {
      const query = new URLSearchParams({ parent_id: parentId });
      return api.get(`/folders?${query}`);
    },
    get: (id) => api.get(`/folders/${id}`),
    create: (name, parentId = '') => api.post('/folders', { name, parent_id: parentId }),
    update: (id, data) => api.patch(`/folders/${id}`, data),
    delete: (id) => api.delete(`/folders/${id}`),
    star: (id, starred) => api.post(`/folders/${id}/star`, { starred }),
    move: (id, parentId) => api.post(`/folders/${id}/move`, { parent_id: parentId }),
    setColor: (id, color) => api.patch(`/folders/${id}/color`, { color }),
  },

  // Share operations
  shares: {
    list: (itemId, itemType) => api.get(`/shares?item_id=${itemId}&item_type=${itemType}`),
    create: (data) => api.post('/shares', data),
    update: (id, data) => api.patch(`/shares/${id}`, data),
    delete: (id) => api.delete(`/shares/${id}`),
    createLink: (id) => api.post(`/shares/${id}/link`),
    deleteLink: (id) => api.delete(`/shares/${id}/link`),
  },

  // Comments
  comments: {
    list: (fileId) => api.get(`/files/${fileId}/comments`),
    create: (fileId, content) => api.post(`/files/${fileId}/comments`, { content }),
    update: (id, content) => api.patch(`/comments/${id}`, { content }),
    delete: (id) => api.delete(`/comments/${id}`),
    resolve: (id) => api.post(`/comments/${id}/resolve`),
  },

  // File requests
  requests: {
    list: () => api.get('/file-requests'),
    get: (id) => api.get(`/file-requests/${id}`),
    create: (data) => api.post('/file-requests', data),
    update: (id, data) => api.patch(`/file-requests/${id}`, data),
    delete: (id) => api.delete(`/file-requests/${id}`),
    close: (id) => api.post(`/file-requests/${id}/close`),
  },

  // Notifications
  notifications: {
    list: () => api.get('/notifications'),
    markRead: (id) => api.patch(`/notifications/${id}/read`),
    markAllRead: () => api.post('/notifications/read-all'),
  },

  // Search
  search: (query, filters = {}) => {
    const params = new URLSearchParams({ q: query, ...filters });
    return api.get(`/search?${params}`);
  },
};

// ============================================
// File Upload
// ============================================

async function uploadFile(file, folderId = '', onProgress = null) {
  const formData = new FormData();
  formData.append('file', file);
  if (folderId) formData.append('folder_id', folderId);

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();

    xhr.upload.addEventListener('progress', (e) => {
      if (e.lengthComputable && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100));
      }
    });

    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(JSON.parse(xhr.responseText).data);
      } else {
        reject(new Error('Upload failed'));
      }
    });

    xhr.addEventListener('error', () => reject(new Error('Upload failed')));
    xhr.open('POST', `${API_BASE}/files`);
    xhr.withCredentials = true;
    xhr.send(formData);
  });
}

// Chunked upload for large files
class ChunkedUpload {
  constructor(file, options = {}) {
    this.file = file;
    this.folderId = options.folderId || '';
    this.chunkSize = options.chunkSize || 5 * 1024 * 1024; // 5MB
    this.onProgress = options.onProgress;
    this.uploadId = null;
    this.totalChunks = 0;
    this.uploadedChunks = new Set();
    this.aborted = false;
  }

  async start() {
    const session = await api.post('/uploads', {
      filename: this.file.name,
      size: this.file.size,
      mime_type: this.file.type,
      folder_id: this.folderId,
    });

    this.uploadId = session.upload_id;
    this.totalChunks = session.total_chunks;
    this.chunkSize = session.chunk_size;

    // Upload chunks with concurrency limit
    const concurrency = 3;
    const chunks = Array.from({ length: this.totalChunks }, (_, i) => i);

    for (let i = 0; i < chunks.length; i += concurrency) {
      if (this.aborted) throw new Error('Upload cancelled');
      const batch = chunks.slice(i, i + concurrency);
      await Promise.all(batch.map(idx => this.uploadChunk(idx)));
    }

    return await api.post(`/uploads/${this.uploadId}/complete`);
  }

  async uploadChunk(index) {
    if (this.aborted) return;

    const start = index * this.chunkSize;
    const end = Math.min(start + this.chunkSize, this.file.size);
    const chunk = this.file.slice(start, end);

    const response = await fetch(`${API_BASE}/uploads/${this.uploadId}/chunk/${index}`, {
      method: 'PUT',
      body: chunk,
      credentials: 'include',
    });

    if (!response.ok) throw new Error(`Chunk ${index} upload failed`);

    this.uploadedChunks.add(index);
    if (this.onProgress) {
      this.onProgress(Math.round((this.uploadedChunks.size / this.totalChunks) * 100));
    }
  }

  abort() {
    this.aborted = true;
    if (this.uploadId) {
      api.delete(`/uploads/${this.uploadId}`).catch(() => {});
    }
  }
}

// ============================================
// Upload Manager
// ============================================

const uploadManager = {
  queue: [],
  active: new Map(),
  maxConcurrent: 3,

  add(files, folderId) {
    const uploadPanel = document.getElementById('upload-panel');
    const uploadList = document.getElementById('upload-list');
    const uploadCount = document.getElementById('upload-count');

    uploadPanel.style.display = 'block';

    for (const file of files) {
      const id = `upload-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      const item = {
        id,
        file,
        folderId,
        progress: 0,
        status: 'pending',
        uploader: null,
      };

      this.queue.push(item);
      this.renderItem(item, uploadList);
    }

    uploadCount.textContent = this.queue.length + this.active.size;
    this.processQueue();
  },

  renderItem(item, container) {
    const div = document.createElement('div');
    div.className = 'upload-item';
    div.id = `upload-item-${item.id}`;
    div.innerHTML = `
      <div class="upload-item-icon">
        ${getFileIconSvg(item.file.type)}
      </div>
      <div class="upload-item-info">
        <div class="upload-item-name">${escapeHtml(item.file.name)}</div>
        <div class="upload-item-status">Waiting...</div>
        <div class="upload-item-progress">
          <div class="upload-item-progress-bar" style="width: 0%"></div>
        </div>
      </div>
      <button class="btn-icon upload-item-action" data-id="${item.id}">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    `;
    container.appendChild(div);
  },

  async processQueue() {
    while (this.queue.length > 0 && this.active.size < this.maxConcurrent) {
      const item = this.queue.shift();
      if (!item) continue;

      this.active.set(item.id, item);
      this.updateItemUI(item.id, 'uploading', 0);

      try {
        const isLarge = item.file.size > 10 * 1024 * 1024;

        if (isLarge) {
          item.uploader = new ChunkedUpload(item.file, {
            folderId: item.folderId,
            onProgress: (pct) => this.updateItemUI(item.id, 'uploading', pct),
          });
          await item.uploader.start();
        } else {
          await uploadFile(item.file, item.folderId, (pct) => {
            this.updateItemUI(item.id, 'uploading', pct);
          });
        }

        this.updateItemUI(item.id, 'complete', 100);
        showToast(`${item.file.name} uploaded successfully`, 'success');
      } catch (error) {
        this.updateItemUI(item.id, 'error', 0);
        showToast(`Failed to upload ${item.file.name}`, 'error');
      }

      this.active.delete(item.id);
      this.processQueue();
    }

    // Refresh file list when all uploads complete
    if (this.queue.length === 0 && this.active.size === 0) {
      setTimeout(() => location.reload(), 1000);
    }
  },

  updateItemUI(id, status, progress) {
    const item = document.getElementById(`upload-item-${id}`);
    if (!item) return;

    const statusEl = item.querySelector('.upload-item-status');
    const progressBar = item.querySelector('.upload-item-progress-bar');

    item.className = `upload-item ${status}`;
    progressBar.style.width = `${progress}%`;

    switch (status) {
      case 'uploading':
        statusEl.textContent = `Uploading... ${progress}%`;
        break;
      case 'complete':
        statusEl.textContent = 'Complete';
        break;
      case 'error':
        statusEl.textContent = 'Failed';
        break;
    }
  },

  cancel(id) {
    // Remove from queue
    const queueIdx = this.queue.findIndex(i => i.id === id);
    if (queueIdx > -1) {
      this.queue.splice(queueIdx, 1);
    }

    // Cancel active upload
    const active = this.active.get(id);
    if (active?.uploader) {
      active.uploader.abort();
    }
    this.active.delete(id);

    // Remove UI
    document.getElementById(`upload-item-${id}`)?.remove();
    document.getElementById('upload-count').textContent = this.queue.length + this.active.size;

    if (this.queue.length === 0 && this.active.size === 0) {
      document.getElementById('upload-panel').style.display = 'none';
    }
  },

  cancelAll() {
    for (const item of this.queue) {
      this.cancel(item.id);
    }
    for (const [id] of this.active) {
      this.cancel(id);
    }
  },
};

// ============================================
// Selection System
// ============================================

const selection = {
  clear() {
    state.selectedItems.clear();
    document.querySelectorAll('.file-card.selected, .file-list-item.selected').forEach(el => {
      el.classList.remove('selected');
      const checkbox = el.querySelector('input[type="checkbox"]');
      if (checkbox) checkbox.checked = false;
    });
    this.updateUI();
  },

  toggle(id, type) {
    const key = `${type}:${id}`;
    if (state.selectedItems.has(key)) {
      state.selectedItems.delete(key);
    } else {
      state.selectedItems.add(key);
    }
    this.updateUI();
  },

  select(id, type) {
    state.selectedItems.add(`${type}:${id}`);
    this.updateUI();
  },

  deselect(id, type) {
    state.selectedItems.delete(`${type}:${id}`);
    this.updateUI();
  },

  selectAll() {
    document.querySelectorAll('.file-card, .file-list-item').forEach(el => {
      const id = el.dataset.id;
      const type = el.dataset.type || (el.classList.contains('folder') ? 'folder' : 'file');
      state.selectedItems.add(`${type}:${id}`);
      el.classList.add('selected');
      const checkbox = el.querySelector('input[type="checkbox"]');
      if (checkbox) checkbox.checked = true;
    });
    this.updateUI();
  },

  range(startId, endId, type) {
    const items = Array.from(document.querySelectorAll('.file-card, .file-list-item'));
    let inRange = false;

    for (const item of items) {
      const id = item.dataset.id;
      if (id === startId || id === endId) {
        inRange = !inRange;
        this.select(id, type);
        if (!inRange) break;
      } else if (inRange) {
        this.select(id, item.dataset.type || 'file');
      }
    }
    this.updateUI();
  },

  updateUI() {
    const count = state.selectedItems.size;
    const toolbar = document.querySelector('.selection-info');

    if (count > 0) {
      if (!toolbar) {
        const toolbarLeft = document.querySelector('.toolbar-left');
        if (toolbarLeft) {
          const info = document.createElement('div');
          info.className = 'selection-info';
          info.innerHTML = `
            <span><strong>${count}</strong> selected</span>
            <button class="selection-clear" onclick="selection.clear()">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          `;
          toolbarLeft.appendChild(info);
        }
      } else {
        toolbar.querySelector('span').innerHTML = `<strong>${count}</strong> selected`;
      }
    } else {
      toolbar?.remove();
    }

    // Update item states
    document.querySelectorAll('.file-card, .file-list-item').forEach(el => {
      const id = el.dataset.id;
      const type = el.dataset.type || (el.classList.contains('folder') ? 'folder' : 'file');
      const key = `${type}:${id}`;
      const selected = state.selectedItems.has(key);

      el.classList.toggle('selected', selected);
      const checkbox = el.querySelector('input[type="checkbox"]');
      if (checkbox) checkbox.checked = selected;
    });
  },

  getItems() {
    return Array.from(state.selectedItems).map(key => {
      const [type, id] = key.split(':');
      return { type, id };
    });
  },
};

// ============================================
// Context Menu
// ============================================

const contextMenu = {
  currentTarget: null,
  currentType: null,

  show(x, y, target, type) {
    this.currentTarget = target;
    this.currentType = type;

    const menu = document.getElementById('context-menu');
    menu.style.left = `${x}px`;
    menu.style.top = `${y}px`;
    menu.classList.add('open');

    // Adjust position if off screen
    const rect = menu.getBoundingClientRect();
    if (rect.right > window.innerWidth) {
      menu.style.left = `${window.innerWidth - rect.width - 10}px`;
    }
    if (rect.bottom > window.innerHeight) {
      menu.style.top = `${window.innerHeight - rect.height - 10}px`;
    }

    // Update star text based on current state
    const starItem = menu.querySelector('[data-action="star"]');
    if (starItem) {
      const isStarred = target?.dataset?.starred === 'true';
      starItem.querySelector('span').textContent = isStarred ? 'Remove from starred' : 'Add to starred';
    }
  },

  hide() {
    document.querySelectorAll('.context-menu.open').forEach(m => m.classList.remove('open'));
    this.currentTarget = null;
    this.currentType = null;
  },

  async handleAction(action) {
    const target = this.currentTarget;
    const type = this.currentType;
    const id = target?.dataset?.id;

    this.hide();

    switch (action) {
      case 'preview':
        if (type === 'file') preview.open(id);
        break;

      case 'open':
        if (type === 'folder') {
          window.location.href = `/folder/${id}`;
        } else {
          window.location.href = `/file/${id}`;
        }
        break;

      case 'download':
        if (type === 'file') {
          window.location.href = `${API_BASE}/files/${id}/download`;
        }
        break;

      case 'share':
        modals.share.open(id, type);
        break;

      case 'get-link':
        await this.copyLink(id, type);
        break;

      case 'star':
        await this.toggleStar(id, type);
        break;

      case 'rename':
        modals.rename.open(id, type, target?.querySelector('.file-card-name, .file-list-item-name span')?.textContent);
        break;

      case 'copy':
        modals.moveCopy.open(id, type, 'copy');
        break;

      case 'move':
        modals.moveCopy.open(id, type, 'move');
        break;

      case 'version-history':
        modals.versionHistory.open(id);
        break;

      case 'delete':
        await this.deleteItem(id, type);
        break;
    }
  },

  async copyLink(id, type) {
    try {
      const endpoint = type === 'file' ? `/files/${id}/link` : `/folders/${id}/link`;
      const result = await api.post(endpoint);
      await navigator.clipboard.writeText(result.url);
      showToast('Link copied to clipboard');
    } catch (error) {
      showToast('Failed to copy link', 'error');
    }
  },

  async toggleStar(id, type) {
    try {
      const isStarred = this.currentTarget?.dataset?.starred === 'true';
      if (type === 'file') {
        await api.files.star(id, !isStarred);
      } else {
        await api.folders.star(id, !isStarred);
      }
      location.reload();
    } catch (error) {
      showToast('Failed to update starred status', 'error');
    }
  },

  async deleteItem(id, type) {
    const name = this.currentTarget?.querySelector('.file-card-name, .file-list-item-name span')?.textContent || 'this item';

    if (!confirm(`Move "${name}" to trash?`)) return;

    try {
      if (type === 'file') {
        await api.files.delete(id);
      } else {
        await api.folders.delete(id);
      }
      showToast('Moved to trash');
      location.reload();
    } catch (error) {
      showToast('Failed to delete', 'error');
    }
  },
};

// ============================================
// Preview Panel
// ============================================

const preview = {
  isOpen: false,
  currentFile: null,

  async open(fileId) {
    const panel = document.getElementById('preview-panel');
    const content = document.getElementById('preview-content');
    const title = document.getElementById('preview-title');

    try {
      const file = await api.files.get(fileId);
      this.currentFile = file;

      title.textContent = file.name;
      document.getElementById('preview-type').textContent = file.extension?.toUpperCase() || 'Unknown';
      document.getElementById('preview-size').textContent = formatSize(file.size);
      document.getElementById('preview-modified').textContent = formatDate(file.updated_at);
      document.getElementById('preview-created').textContent = formatDate(file.created_at);

      // Render preview based on type
      content.innerHTML = this.renderContent(file);

      // Load comments
      await this.loadComments(fileId);

      panel.classList.add('open');
      this.isOpen = true;
    } catch (error) {
      showToast('Failed to load preview', 'error');
    }
  },

  renderContent(file) {
    const previewUrl = `${API_BASE}/files/${file.id}/preview`;
    const mime = file.mime_type || '';

    if (mime.startsWith('image/')) {
      return `<img src="${previewUrl}" alt="${escapeHtml(file.name)}">`;
    }

    if (mime.startsWith('video/')) {
      return `<video src="${previewUrl}" controls></video>`;
    }

    if (mime.startsWith('audio/')) {
      return `
        <div style="text-align: center; padding: 40px;">
          ${getFileIconSvg(mime, 64)}
          <audio src="${previewUrl}" controls style="width: 100%; margin-top: 20px;"></audio>
        </div>
      `;
    }

    if (mime === 'application/pdf') {
      return `<iframe src="${previewUrl}" style="width: 100%; height: 100%;"></iframe>`;
    }

    if (mime.startsWith('text/') || ['application/json', 'application/javascript'].includes(mime)) {
      return `<iframe src="${previewUrl}" style="width: 100%; height: 100%;"></iframe>`;
    }

    return `
      <div class="empty-state">
        ${getFileIconSvg(mime, 80)}
        <h3 style="margin-top: 20px;">No preview available</h3>
        <p>This file type cannot be previewed</p>
        <button class="btn btn-primary" onclick="window.location.href='${API_BASE}/files/${file.id}/download'">
          Download
        </button>
      </div>
    `;
  },

  async loadComments(fileId) {
    try {
      const comments = await api.comments.list(fileId);
      const list = document.getElementById('comments-list');
      const count = document.getElementById('comment-count');

      count.textContent = comments.length;
      list.innerHTML = comments.map(c => this.renderComment(c)).join('');
    } catch (error) {
      console.error('Failed to load comments:', error);
    }
  },

  renderComment(comment) {
    const initials = comment.author_name?.split(' ').map(n => n[0]).join('').toUpperCase() || '?';
    return `
      <div class="comment-item" data-id="${comment.id}">
        <div class="comment-header">
          <div class="comment-avatar">${initials}</div>
          <span class="comment-author">${escapeHtml(comment.author_name)}</span>
          <span class="comment-time">${formatDate(comment.created_at)}</span>
        </div>
        <div class="comment-content">${escapeHtml(comment.content)}</div>
        <div class="comment-actions">
          <span class="comment-action" onclick="preview.replyToComment('${comment.id}')">Reply</span>
          ${comment.is_owner ? `<span class="comment-action" onclick="preview.deleteComment('${comment.id}')">Delete</span>` : ''}
        </div>
      </div>
    `;
  },

  async addComment() {
    const input = document.getElementById('comment-input');
    const content = input.value.trim();

    if (!content || !this.currentFile) return;

    try {
      await api.comments.create(this.currentFile.id, content);
      input.value = '';
      await this.loadComments(this.currentFile.id);
      showToast('Comment added');
    } catch (error) {
      showToast('Failed to add comment', 'error');
    }
  },

  async deleteComment(commentId) {
    if (!confirm('Delete this comment?')) return;

    try {
      await api.comments.delete(commentId);
      await this.loadComments(this.currentFile.id);
      showToast('Comment deleted');
    } catch (error) {
      showToast('Failed to delete comment', 'error');
    }
  },

  close() {
    document.getElementById('preview-panel').classList.remove('open');
    this.isOpen = false;
    this.currentFile = null;
  },
};

// ============================================
// Modals
// ============================================

const modals = {
  // Generic modal functions
  open(id) {
    document.getElementById(id)?.classList.add('open');
  },

  close(id) {
    document.getElementById(id)?.classList.remove('open');
  },

  closeAll() {
    document.querySelectorAll('.modal-overlay.open').forEach(m => m.classList.remove('open'));
  },

  // Share modal
  share: {
    currentId: null,
    currentType: null,

    async open(id, type) {
      modals.share.currentId = id;
      modals.share.currentType = type;

      // Create modal if doesn't exist
      let modal = document.getElementById('share-modal');
      if (!modal) {
        modal = document.createElement('div');
        modal.id = 'share-modal';
        modal.className = 'modal-overlay';
        modal.innerHTML = `
          <div class="modal modal-md">
            <div class="modal-header">
              <span class="modal-title">Share</span>
              <button class="modal-close" onclick="modals.close('share-modal')">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <div class="share-input-group">
                <input type="email" class="share-input form-input" id="share-email" placeholder="Add people by email">
                <button class="btn btn-primary" onclick="modals.share.addPerson()">Add</button>
              </div>
              <div class="share-people" id="share-people">
                <div class="share-people-title">People with access</div>
              </div>
              <div class="share-link-section">
                <div class="share-link-header">
                  <svg class="share-link-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                    <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                  </svg>
                  <span class="share-link-title">Link sharing</span>
                  <div class="share-link-toggle" id="share-link-toggle" onclick="modals.share.toggleLink()"></div>
                </div>
                <div class="share-link-url" id="share-link-url" style="display: none;">
                  <input type="text" class="share-link-input" id="share-link-input" readonly>
                  <button class="btn btn-secondary" onclick="modals.share.copyLink()">Copy</button>
                </div>
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-secondary" onclick="modals.close('share-modal')">Done</button>
            </div>
          </div>
        `;
        document.body.appendChild(modal);
      }

      modal.classList.add('open');
      await this.loadShares();
    },

    async loadShares() {
      try {
        const shares = await api.shares.list(this.currentId, this.currentType);
        const container = document.getElementById('share-people');

        container.innerHTML = '<div class="share-people-title">People with access</div>';

        for (const share of shares) {
          const initials = share.shared_with_name?.split(' ').map(n => n[0]).join('').toUpperCase() || '?';
          container.innerHTML += `
            <div class="share-person" data-id="${share.id}">
              <div class="user-avatar">${initials}</div>
              <div class="share-person-info">
                <div class="share-person-name">${escapeHtml(share.shared_with_name || share.shared_with_email)}</div>
                <div class="share-person-email">${escapeHtml(share.shared_with_email)}</div>
              </div>
              <select class="share-person-role" onchange="modals.share.updatePermission('${share.id}', this.value)">
                <option value="viewer" ${share.permission === 'viewer' ? 'selected' : ''}>Viewer</option>
                <option value="commenter" ${share.permission === 'commenter' ? 'selected' : ''}>Commenter</option>
                <option value="editor" ${share.permission === 'editor' ? 'selected' : ''}>Editor</option>
              </select>
              <button class="btn-icon" onclick="modals.share.removePerson('${share.id}')">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
          `;
        }

        // Check if link sharing is enabled
        const hasLink = shares.some(s => s.link_enabled);
        document.getElementById('share-link-toggle').classList.toggle('active', hasLink);
        document.getElementById('share-link-url').style.display = hasLink ? 'flex' : 'none';
        if (hasLink) {
          const linkShare = shares.find(s => s.link_enabled);
          document.getElementById('share-link-input').value = linkShare?.link_url || '';
        }
      } catch (error) {
        showToast('Failed to load shares', 'error');
      }
    },

    async addPerson() {
      const email = document.getElementById('share-email').value.trim();
      if (!email) return;

      try {
        await api.shares.create({
          item_id: this.currentId,
          item_type: this.currentType,
          shared_with_email: email,
          permission: 'viewer',
        });
        document.getElementById('share-email').value = '';
        await this.loadShares();
        showToast('Shared successfully');
      } catch (error) {
        showToast('Failed to share', 'error');
      }
    },

    async updatePermission(shareId, permission) {
      try {
        await api.shares.update(shareId, { permission });
      } catch (error) {
        showToast('Failed to update permission', 'error');
      }
    },

    async removePerson(shareId) {
      try {
        await api.shares.delete(shareId);
        await this.loadShares();
      } catch (error) {
        showToast('Failed to remove', 'error');
      }
    },

    async toggleLink() {
      const toggle = document.getElementById('share-link-toggle');
      const isActive = toggle.classList.contains('active');

      try {
        if (isActive) {
          // Disable link
          toggle.classList.remove('active');
          document.getElementById('share-link-url').style.display = 'none';
        } else {
          // Enable link
          const result = await api.post(`/${this.currentType}s/${this.currentId}/link`);
          toggle.classList.add('active');
          document.getElementById('share-link-url').style.display = 'flex';
          document.getElementById('share-link-input').value = result.url;
        }
      } catch (error) {
        showToast('Failed to toggle link', 'error');
      }
    },

    async copyLink() {
      const input = document.getElementById('share-link-input');
      try {
        await navigator.clipboard.writeText(input.value);
        showToast('Link copied to clipboard');
      } catch (error) {
        showToast('Failed to copy link', 'error');
      }
    },
  },

  // Rename modal
  rename: {
    currentId: null,
    currentType: null,

    open(id, type, currentName) {
      this.currentId = id;
      this.currentType = type;

      let modal = document.getElementById('rename-modal');
      if (!modal) {
        modal = document.createElement('div');
        modal.id = 'rename-modal';
        modal.className = 'modal-overlay';
        modal.innerHTML = `
          <div class="modal modal-sm">
            <div class="modal-header">
              <span class="modal-title">Rename</span>
              <button class="modal-close" onclick="modals.close('rename-modal')">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <div class="form-group">
                <label class="form-label">Name</label>
                <input type="text" class="form-input" id="rename-input">
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-secondary" onclick="modals.close('rename-modal')">Cancel</button>
              <button class="btn btn-primary" onclick="modals.rename.submit()">Rename</button>
            </div>
          </div>
        `;
        document.body.appendChild(modal);
      }

      document.getElementById('rename-input').value = currentName || '';
      modal.classList.add('open');
      document.getElementById('rename-input').focus();
      document.getElementById('rename-input').select();
    },

    async submit() {
      const name = document.getElementById('rename-input').value.trim();
      if (!name) return;

      try {
        if (this.currentType === 'file') {
          await api.files.update(this.currentId, { name });
        } else {
          await api.folders.update(this.currentId, { name });
        }
        modals.close('rename-modal');
        showToast('Renamed successfully');
        location.reload();
      } catch (error) {
        showToast('Failed to rename', 'error');
      }
    },
  },

  // New folder modal
  newFolder: {
    open(parentId = '') {
      let modal = document.getElementById('new-folder-modal');
      if (!modal) {
        modal = document.createElement('div');
        modal.id = 'new-folder-modal';
        modal.className = 'modal-overlay';
        modal.innerHTML = `
          <div class="modal modal-sm">
            <div class="modal-header">
              <span class="modal-title">New folder</span>
              <button class="modal-close" onclick="modals.close('new-folder-modal')">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <div class="form-group">
                <label class="form-label">Folder name</label>
                <input type="text" class="form-input" id="new-folder-name" placeholder="Untitled folder">
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-secondary" onclick="modals.close('new-folder-modal')">Cancel</button>
              <button class="btn btn-primary" onclick="modals.newFolder.create()">Create</button>
            </div>
          </div>
        `;
        document.body.appendChild(modal);
      }

      modal.dataset.parentId = parentId;
      document.getElementById('new-folder-name').value = '';
      modal.classList.add('open');
      document.getElementById('new-folder-name').focus();
    },

    async create() {
      const name = document.getElementById('new-folder-name').value.trim() || 'Untitled folder';
      const parentId = document.getElementById('new-folder-modal').dataset.parentId || '';

      try {
        await api.folders.create(name, parentId);
        modals.close('new-folder-modal');
        showToast('Folder created');
        location.reload();
      } catch (error) {
        showToast('Failed to create folder', 'error');
      }
    },
  },

  // Version history modal
  versionHistory: {
    currentFileId: null,

    async open(fileId) {
      this.currentFileId = fileId;

      let modal = document.getElementById('version-modal');
      if (!modal) {
        modal = document.createElement('div');
        modal.id = 'version-modal';
        modal.className = 'modal-overlay';
        modal.innerHTML = `
          <div class="modal modal-lg">
            <div class="modal-header">
              <span class="modal-title">Version history</span>
              <button class="modal-close" onclick="modals.close('version-modal')">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <div class="version-list" id="version-list">
                <div class="spinner"></div>
              </div>
            </div>
          </div>
        `;
        document.body.appendChild(modal);
      }

      modal.classList.add('open');
      await this.loadVersions();
    },

    async loadVersions() {
      try {
        const versions = await api.files.versions(this.currentFileId);
        const container = document.getElementById('version-list');

        container.innerHTML = versions.map((v, i) => `
          <div class="version-item ${i === 0 ? 'current' : ''}">
            <div class="version-icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
              </svg>
            </div>
            <div class="version-info">
              <div class="version-header">
                <span class="version-number">v${v.version_number}</span>
                <span class="version-date">${formatDate(v.created_at)}</span>
              </div>
              <div class="version-author">${escapeHtml(v.uploaded_by_name || 'Unknown')}</div>
              ${v.comment ? `<div class="version-comment">"${escapeHtml(v.comment)}"</div>` : ''}
              <div class="version-actions">
                ${i === 0 ? '<span class="btn btn-sm btn-ghost" disabled>Current</span>' : `
                  <button class="btn btn-sm btn-secondary" onclick="modals.versionHistory.restore(${v.version_number})">Restore</button>
                `}
                <button class="btn btn-sm btn-ghost" onclick="window.location.href='${API_BASE}/files/${this.currentFileId}/versions/${v.version_number}/download'">Download</button>
              </div>
            </div>
          </div>
        `).join('');
      } catch (error) {
        showToast('Failed to load versions', 'error');
      }
    },

    async restore(version) {
      if (!confirm(`Restore version ${version}? This will create a new version.`)) return;

      try {
        await api.files.restoreVersion(this.currentFileId, version);
        modals.close('version-modal');
        showToast('Version restored');
        location.reload();
      } catch (error) {
        showToast('Failed to restore version', 'error');
      }
    },
  },
};

// ============================================
// Search
// ============================================

const search = {
  debounceTimer: null,

  init() {
    const input = document.getElementById('search-input');
    if (!input) return;

    input.addEventListener('input', () => {
      clearTimeout(this.debounceTimer);
      this.debounceTimer = setTimeout(() => this.search(input.value), 300);
    });

    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        clearTimeout(this.debounceTimer);
        this.search(input.value);
      }
      if (e.key === 'Escape') {
        input.value = '';
        input.blur();
      }
    });
  },

  async search(query) {
    if (!query.trim()) return;

    try {
      window.location.href = `/search?q=${encodeURIComponent(query)}`;
    } catch (error) {
      showToast('Search failed', 'error');
    }
  },
};

// ============================================
// Drag and Drop
// ============================================

const dragDrop = {
  init() {
    // Global drag and drop for file upload
    document.addEventListener('dragover', (e) => {
      e.preventDefault();
      document.body.classList.add('dragging');
    });

    document.addEventListener('dragleave', (e) => {
      if (e.relatedTarget === null) {
        document.body.classList.remove('dragging');
      }
    });

    document.addEventListener('drop', (e) => {
      e.preventDefault();
      document.body.classList.remove('dragging');

      const files = Array.from(e.dataTransfer.files);
      if (files.length > 0) {
        const folderId = document.body.dataset.currentFolder || '';
        uploadManager.add(files, folderId);
      }
    });

    // Upload zone specific
    const uploadZone = document.getElementById('upload-zone');
    if (uploadZone) {
      uploadZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        uploadZone.classList.add('dragover');
      });

      uploadZone.addEventListener('dragleave', () => {
        uploadZone.classList.remove('dragover');
      });

      uploadZone.addEventListener('drop', (e) => {
        e.preventDefault();
        uploadZone.classList.remove('dragover');

        const files = Array.from(e.dataTransfer.files);
        if (files.length > 0) {
          const folderId = document.body.dataset.currentFolder || '';
          uploadManager.add(files, folderId);
        }
      });
    }
  },
};

// ============================================
// Keyboard Shortcuts
// ============================================

const keyboard = {
  init() {
    document.addEventListener('keydown', (e) => {
      // Ignore when typing in inputs
      if (e.target.matches('input, textarea, [contenteditable]')) {
        if (e.key === 'Escape') e.target.blur();
        return;
      }

      const key = e.key.toLowerCase();
      const ctrl = e.ctrlKey || e.metaKey;
      const shift = e.shiftKey;

      // Global shortcuts
      if (key === 'escape') {
        contextMenu.hide();
        modals.closeAll();
        preview.close();
        selection.clear();
        return;
      }

      if (key === '/' || (ctrl && key === 'k')) {
        e.preventDefault();
        document.getElementById('search-input')?.focus();
        return;
      }

      if (key === '?') {
        e.preventDefault();
        this.showHelp();
        return;
      }

      // Selection shortcuts
      if (ctrl && key === 'a') {
        e.preventDefault();
        selection.selectAll();
        return;
      }

      // Navigation
      if (key === 'g') {
        this.waitForSecond = true;
        setTimeout(() => { this.waitForSecond = false; }, 500);
        return;
      }

      if (this.waitForSecond) {
        this.waitForSecond = false;
        switch (key) {
          case 'h': window.location.href = '/'; break;
          case 'r': window.location.href = '/recent'; break;
          case 's': window.location.href = '/starred'; break;
          case 't': window.location.href = '/trash'; break;
        }
        return;
      }

      // File actions (when items selected)
      if (state.selectedItems.size > 0) {
        const items = selection.getItems();

        switch (key) {
          case 'delete':
          case 'backspace':
            e.preventDefault();
            items.forEach(i => contextMenu.deleteItem(i.id, i.type));
            break;

          case 'd':
            e.preventDefault();
            items.filter(i => i.type === 'file').forEach(i => {
              window.location.href = `${API_BASE}/files/${i.id}/download`;
            });
            break;

          case 's':
            if (!shift) {
              e.preventDefault();
              items.forEach(i => contextMenu.toggleStar(i.id, i.type));
            } else {
              e.preventDefault();
              if (items.length === 1) {
                modals.share.open(items[0].id, items[0].type);
              }
            }
            break;

          case 'enter':
            e.preventDefault();
            if (items.length === 1) {
              const item = items[0];
              if (item.type === 'folder') {
                window.location.href = `/folder/${item.id}`;
              } else {
                window.location.href = `/file/${item.id}`;
              }
            }
            break;

          case ' ':
            e.preventDefault();
            if (items.length === 1 && items[0].type === 'file') {
              preview.open(items[0].id);
            }
            break;

          case 'f2':
            e.preventDefault();
            if (items.length === 1) {
              const el = document.querySelector(`[data-id="${items[0].id}"]`);
              const name = el?.querySelector('.file-card-name, .file-list-item-name span')?.textContent;
              modals.rename.open(items[0].id, items[0].type, name);
            }
            break;
        }
      }

      // New shortcuts
      if (key === 'n' && !ctrl) {
        e.preventDefault();
        document.getElementById('new-btn')?.click();
      }

      if (key === 'u' && !ctrl) {
        e.preventDefault();
        document.getElementById('file-input')?.click();
      }
    });
  },

  showHelp() {
    let modal = document.getElementById('shortcuts-modal');
    if (!modal) {
      modal = document.createElement('div');
      modal.id = 'shortcuts-modal';
      modal.className = 'modal-overlay';
      modal.innerHTML = `
        <div class="modal modal-lg">
          <div class="modal-header">
            <span class="modal-title">Keyboard shortcuts</span>
            <button class="modal-close" onclick="modals.close('shortcuts-modal')">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>
          <div class="modal-body" style="column-count: 2; column-gap: 40px;">
            <div style="break-inside: avoid; margin-bottom: 24px;">
              <h4 style="margin-bottom: 8px;">Navigation</h4>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Go to Home</span><kbd>G then H</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Go to Recent</span><kbd>G then R</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Go to Starred</span><kbd>G then S</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Go to Trash</span><kbd>G then T</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Search</span><kbd>/ or Cmd+K</kbd>
              </div>
            </div>
            <div style="break-inside: avoid; margin-bottom: 24px;">
              <h4 style="margin-bottom: 8px;">Actions</h4>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>New</span><kbd>N</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Upload</span><kbd>U</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Select all</span><kbd>Cmd+A</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Clear selection</span><kbd>Esc</kbd>
              </div>
            </div>
            <div style="break-inside: avoid; margin-bottom: 24px;">
              <h4 style="margin-bottom: 8px;">File actions</h4>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Open</span><kbd>Enter</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Preview</span><kbd>Space</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Download</span><kbd>D</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Star</span><kbd>S</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Share</span><kbd>Shift+S</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Rename</span><kbd>F2</kbd>
              </div>
              <div style="display: flex; justify-content: space-between; padding: 4px 0;">
                <span>Delete</span><kbd>Delete</kbd>
              </div>
            </div>
          </div>
        </div>
      `;
      document.body.appendChild(modal);
    }
    modal.classList.add('open');
  },
};

// ============================================
// Theme
// ============================================

const theme = {
  init() {
    const saved = localStorage.getItem('theme') || 'light';
    this.set(saved);
  },

  toggle() {
    const current = document.documentElement.dataset.theme || 'light';
    const next = current === 'light' ? 'dark' : 'light';
    this.set(next);
  },

  set(value) {
    document.documentElement.dataset.theme = value;
    localStorage.setItem('theme', value);
    state.theme = value;
  },
};

// ============================================
// View Mode
// ============================================

const viewMode = {
  set(mode) {
    state.viewMode = mode;
    localStorage.setItem('viewMode', mode);

    document.querySelectorAll('.view-toggle-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.view === mode);
    });

    const grid = document.querySelector('.file-grid');
    const list = document.querySelector('.file-list');

    if (grid) grid.style.display = mode === 'grid' ? 'grid' : 'none';
    if (list) list.style.display = mode === 'list' ? 'flex' : 'none';
  },
};

// ============================================
// Notifications
// ============================================

const notifications = {
  dropdown: null,
  isOpen: false,

  async toggle() {
    if (this.isOpen) {
      this.close();
    } else {
      await this.open();
    }
  },

  async open() {
    const dropdown = document.getElementById('notifications-dropdown');
    const btn = document.getElementById('notifications-btn');

    if (!dropdown) return;

    try {
      const items = await api.notifications.list();
      const list = document.getElementById('notifications-list');

      list.innerHTML = items.length > 0
        ? items.map(n => `
            <div class="notification-item ${n.read ? '' : 'unread'}" data-id="${n.id}" onclick="notifications.handleClick('${n.id}', '${n.item_type}', '${n.item_id}')">
              <div class="notification-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  ${this.getIcon(n.type)}
                </svg>
              </div>
              <div class="notification-content">
                <div class="notification-text">${escapeHtml(n.message)}</div>
                <div class="notification-time">${formatDate(n.created_at)}</div>
              </div>
            </div>
          `).join('')
        : '<div style="padding: 40px; text-align: center; color: var(--color-text-secondary);">No notifications</div>';

      dropdown.style.display = 'block';

      // Position dropdown
      const rect = btn.getBoundingClientRect();
      dropdown.style.top = `${rect.bottom + 8}px`;
      dropdown.style.right = `${window.innerWidth - rect.right}px`;

      this.isOpen = true;
    } catch (error) {
      console.error('Failed to load notifications:', error);
    }
  },

  close() {
    document.getElementById('notifications-dropdown').style.display = 'none';
    this.isOpen = false;
  },

  getIcon(type) {
    switch (type) {
      case 'share_received':
        return '<circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/>';
      case 'comment_added':
      case 'comment_mention':
        return '<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>';
      default:
        return '<circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>';
    }
  },

  async handleClick(id, itemType, itemId) {
    await api.notifications.markRead(id);

    if (itemType === 'file') {
      window.location.href = `/file/${itemId}`;
    } else if (itemType === 'folder') {
      window.location.href = `/folder/${itemId}`;
    }
  },

  async markAllRead() {
    try {
      await api.notifications.markAllRead();
      document.querySelectorAll('.notification-item.unread').forEach(el => {
        el.classList.remove('unread');
      });
      document.querySelector('.notification-badge')?.remove();
    } catch (error) {
      showToast('Failed to mark as read', 'error');
    }
  },
};

// ============================================
// Utility Functions
// ============================================

function formatSize(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDate(dateString) {
  const date = new Date(dateString);
  const now = new Date();
  const diff = now - date;

  if (diff < 60000) return 'Just now';
  if (diff < 3600000) return `${Math.floor(diff / 60000)} min ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} hours ago`;
  if (diff < 604800000) return `${Math.floor(diff / 86400000)} days ago`;

  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  });
}

function escapeHtml(str) {
  if (!str) return '';
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

function showToast(message, type = 'info') {
  const container = document.getElementById('toast-container');
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.innerHTML = `
    <span>${escapeHtml(message)}</span>
    <button class="toast-close" onclick="this.parentElement.remove()">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
      </svg>
    </button>
  `;
  container.appendChild(toast);

  setTimeout(() => toast.remove(), 5000);
}

function getFileIconSvg(mimeType, size = 24) {
  let icon;
  const mime = mimeType || '';

  if (mime.startsWith('image/')) {
    icon = '<rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/>';
  } else if (mime.startsWith('video/')) {
    icon = '<polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/>';
  } else if (mime.startsWith('audio/')) {
    icon = '<path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>';
  } else if (mime.includes('pdf')) {
    icon = '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/>';
  } else if (mime.includes('word') || mime.includes('document')) {
    icon = '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/>';
  } else if (mime.includes('sheet') || mime.includes('excel')) {
    icon = '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="8" y1="13" x2="16" y2="13"/><line x1="8" y1="17" x2="16" y2="17"/><line x1="12" y1="9" x2="12" y2="21"/>';
  } else if (mime.includes('zip') || mime.includes('archive') || mime.includes('compressed')) {
    icon = '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/><line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/>';
  } else {
    icon = '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>';
  }

  return `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">${icon}</svg>`;
}

// ============================================
// Initialize Application
// ============================================

document.addEventListener('DOMContentLoaded', () => {
  // Initialize modules
  theme.init();
  search.init();
  dragDrop.init();
  keyboard.init();

  // Set initial view mode
  viewMode.set(state.viewMode);

  // Sidebar toggle
  const sidebarToggle = document.getElementById('sidebar-toggle');
  const sidebar = document.getElementById('sidebar');

  if (sidebarToggle && sidebar) {
    if (state.sidebarCollapsed) {
      sidebar.classList.add('collapsed');
    }

    sidebarToggle.addEventListener('click', () => {
      sidebar.classList.toggle('collapsed');
      state.sidebarCollapsed = sidebar.classList.contains('collapsed');
      localStorage.setItem('sidebarCollapsed', state.sidebarCollapsed);
    });
  }

  // Theme toggle
  document.getElementById('theme-toggle')?.addEventListener('click', () => theme.toggle());

  // New button dropdown
  const newBtn = document.getElementById('new-btn');
  const newMenu = document.getElementById('new-menu');

  if (newBtn && newMenu) {
    newBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      const rect = newBtn.getBoundingClientRect();
      newMenu.style.left = `${rect.left}px`;
      newMenu.style.top = `${rect.bottom + 4}px`;
      newMenu.classList.toggle('open');
    });
  }

  // User menu dropdown
  const userMenu = document.getElementById('user-menu');
  const userDropdown = document.getElementById('user-dropdown');

  if (userMenu && userDropdown) {
    userMenu.addEventListener('click', (e) => {
      e.stopPropagation();
      const rect = userMenu.getBoundingClientRect();
      userDropdown.style.right = `${window.innerWidth - rect.right}px`;
      userDropdown.style.top = `${rect.bottom + 4}px`;
      userDropdown.classList.toggle('open');
    });
  }

  // Notifications
  document.getElementById('notifications-btn')?.addEventListener('click', () => notifications.toggle());
  document.getElementById('mark-all-read')?.addEventListener('click', () => notifications.markAllRead());

  // New menu actions
  newMenu?.addEventListener('click', (e) => {
    const action = e.target.closest('[data-action]')?.dataset.action;
    contextMenu.hide();

    switch (action) {
      case 'new-folder':
        modals.newFolder.open(document.body.dataset.currentFolder || '');
        break;
      case 'upload-files':
        document.getElementById('file-input')?.click();
        break;
      case 'upload-folder':
        document.getElementById('folder-input')?.click();
        break;
    }
  });

  // File input handlers
  document.getElementById('file-input')?.addEventListener('change', (e) => {
    const files = Array.from(e.target.files);
    if (files.length > 0) {
      uploadManager.add(files, document.body.dataset.currentFolder || '');
    }
    e.target.value = '';
  });

  document.getElementById('folder-input')?.addEventListener('change', (e) => {
    const files = Array.from(e.target.files);
    if (files.length > 0) {
      uploadManager.add(files, document.body.dataset.currentFolder || '');
    }
    e.target.value = '';
  });

  // Context menu for files/folders
  document.addEventListener('contextmenu', (e) => {
    const card = e.target.closest('.file-card, .file-list-item');
    if (card) {
      e.preventDefault();
      const type = card.dataset.type || (card.classList.contains('folder') ? 'folder' : 'file');
      contextMenu.show(e.clientX, e.clientY, card, type);
    }
  });

  // Context menu item clicks
  document.getElementById('context-menu')?.addEventListener('click', (e) => {
    const action = e.target.closest('[data-action]')?.dataset.action;
    if (action) contextMenu.handleAction(action);
  });

  // Close dropdowns on outside click
  document.addEventListener('click', (e) => {
    if (!e.target.closest('.context-menu, #new-btn, #user-menu, #notifications-btn')) {
      contextMenu.hide();
    }
    if (!e.target.closest('#notifications-dropdown, #notifications-btn')) {
      notifications.close();
    }
  });

  // File card clicks
  document.querySelectorAll('.file-card, .file-list-item').forEach(card => {
    card.addEventListener('click', (e) => {
      const isCheckbox = e.target.closest('.file-card-checkbox, .file-list-item-checkbox');
      const isAction = e.target.closest('.file-card-actions, .file-list-item-actions, .btn-icon');

      if (isCheckbox || isAction) return;

      const id = card.dataset.id;
      const type = card.dataset.type || (card.classList.contains('folder') ? 'folder' : 'file');

      if (e.ctrlKey || e.metaKey) {
        selection.toggle(id, type);
      } else if (e.shiftKey && state.selectedItems.size > 0) {
        const lastSelected = Array.from(state.selectedItems).pop()?.split(':')[1];
        selection.range(lastSelected, id, type);
      } else {
        selection.clear();
        if (type === 'folder') {
          window.location.href = `/folder/${id}`;
        } else {
          window.location.href = `/file/${id}`;
        }
      }
    });

    // Checkbox clicks
    card.querySelector('input[type="checkbox"]')?.addEventListener('change', (e) => {
      const id = card.dataset.id;
      const type = card.dataset.type || (card.classList.contains('folder') ? 'folder' : 'file');

      if (e.target.checked) {
        selection.select(id, type);
      } else {
        selection.deselect(id, type);
      }
    });
  });

  // View toggle
  document.querySelectorAll('.view-toggle-btn').forEach(btn => {
    btn.addEventListener('click', () => viewMode.set(btn.dataset.view));
  });

  // Preview panel
  document.getElementById('preview-close')?.addEventListener('click', () => preview.close());
  document.getElementById('submit-comment')?.addEventListener('click', () => preview.addComment());
  document.getElementById('comment-input')?.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') preview.addComment();
  });

  // Upload panel
  document.getElementById('upload-minimize')?.addEventListener('click', () => {
    document.getElementById('upload-panel').style.display = 'none';
  });
  document.getElementById('upload-cancel-all')?.addEventListener('click', () => {
    uploadManager.cancelAll();
  });
  document.getElementById('upload-list')?.addEventListener('click', (e) => {
    const cancelBtn = e.target.closest('[data-id]');
    if (cancelBtn) uploadManager.cancel(cancelBtn.dataset.id);
  });

  // Mobile menu
  const mobileMenuBtn = document.getElementById('mobile-menu-btn');
  if (mobileMenuBtn) {
    if (window.innerWidth < 768) {
      mobileMenuBtn.style.display = 'flex';
    }

    mobileMenuBtn.addEventListener('click', () => {
      sidebar?.classList.toggle('mobile-open');
    });

    window.addEventListener('resize', () => {
      mobileMenuBtn.style.display = window.innerWidth < 768 ? 'flex' : 'none';
      if (window.innerWidth >= 768) {
        sidebar?.classList.remove('mobile-open');
      }
    });
  }

  console.log('Drive application initialized');
});

// Export for global access
window.Drive = {
  api,
  uploadManager,
  selection,
  contextMenu,
  preview,
  modals,
  search,
  notifications,
  theme,
  viewMode,
  formatSize,
  formatDate,
  showToast,
};
