// Drive Frontend JavaScript

(function() {
  'use strict';

  // State
  let selectedItems = new Set();
  let currentPath = '';
  let clipboard = { items: [], action: null };

  // Preview state
  let previewState = {
    isOpen: false,
    currentFile: null,
    files: [], // All previewable files in current view
    currentIndex: -1,
    pdfDoc: null,
    pdfPage: 1,
    pdfScale: 1.0
  };

  // Column View State
  let columnState = {
    selectedItem: null,
    columns: [],
    currentPath: ''
  };

  // Gallery View State
  let galleryState = {
    items: [],
    currentIndex: -1,
    currentItem: null
  };

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
    setupPreviewOverlay();
    collectPreviewableFiles();
    setupColumnView();
    setupGalleryView();
  }

  // Collect all files for navigation
  function collectPreviewableFiles() {
    previewState.files = [];
    document.querySelectorAll('[data-type="file"]').forEach(item => {
      previewState.files.push({
        id: item.dataset.id,
        name: item.dataset.name,
        mime: item.dataset.mime || 'application/octet-stream'
      });
    });
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

    document.addEventListener('click', closeAllDropdowns);
  }

  function closeAllDropdowns() {
    document.querySelectorAll('.dropdown-menu').forEach(menu => {
      menu.classList.add('hidden');
    });
  }

  // Modals
  function setupModals() {
    document.querySelectorAll('[data-action="close-modal"]').forEach(btn => {
      btn.addEventListener('click', () => {
        const modal = btn.closest('.fixed');
        if (modal) {
          modal.classList.add('hidden');
        }
      });
    });

    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        if (previewState.isOpen) {
          closePreview();
          return;
        }
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

    document.querySelectorAll('[data-action="new-folder"]').forEach(btn => {
      btn.addEventListener('click', () => {
        closeAllDropdowns();
        openModal('folder-modal');
      });
    });

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
        const name = item.dataset.name;
        const mime = item.dataset.mime;

        if (e.ctrlKey || e.metaKey) {
          // Multi-select
          if (selectedItems.has(id)) {
            selectedItems.delete(id);
            item.classList.remove('ring-2', 'ring-zinc-900', 'dark:ring-zinc-100');
          } else {
            selectedItems.add(id);
            item.classList.add('ring-2', 'ring-zinc-900', 'dark:ring-zinc-100');
          }
        } else if (e.shiftKey) {
          // Range select (TODO)
        } else {
          // Single click
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
            // Open preview overlay for files
            openPreview({ id, name, mime });
          }
        }
      });

      // Double-click to open in new tab
      item.addEventListener('dblclick', (e) => {
        e.preventDefault();
        const type = item.dataset.type;
        const id = item.dataset.id;
        if (type === 'folder') {
          window.location.href = '/files/' + id;
        } else {
          // Open in new tab
          window.open('/api/v1/content/' + id, '_blank');
        }
      });
    });
  }

  function clearSelection() {
    selectedItems.clear();
    document.querySelectorAll('[data-type="file"], [data-type="folder"]').forEach(item => {
      item.classList.remove('ring-2', 'ring-zinc-900', 'dark:ring-zinc-100');
    });
  }

  // Preview Overlay
  function setupPreviewOverlay() {
    const overlay = document.getElementById('preview-overlay');
    if (!overlay) return;

    // Close button
    document.getElementById('preview-close')?.addEventListener('click', closePreview);

    // Click outside content to close
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay || e.target.id === 'preview-content') {
        closePreview();
      }
    });

    // Navigation buttons
    document.getElementById('preview-prev')?.addEventListener('click', () => navigatePreview(-1));
    document.getElementById('preview-next')?.addEventListener('click', () => navigatePreview(1));

    // Fullscreen
    document.getElementById('preview-fullscreen')?.addEventListener('click', toggleFullscreen);

    // PDF controls
    document.getElementById('pdf-prev-page')?.addEventListener('click', () => changePdfPage(-1));
    document.getElementById('pdf-next-page')?.addEventListener('click', () => changePdfPage(1));
    document.getElementById('pdf-zoom-in')?.addEventListener('click', () => changePdfZoom(1.25));
    document.getElementById('pdf-zoom-out')?.addEventListener('click', () => changePdfZoom(0.8));

    // Show nav arrows on mouse move
    overlay.addEventListener('mousemove', () => {
      document.getElementById('preview-prev')?.classList.add('opacity-60');
      document.getElementById('preview-next')?.classList.add('opacity-60');
    });
  }

  function openPreview(file) {
    const overlay = document.getElementById('preview-overlay');
    if (!overlay) return;

    previewState.isOpen = true;
    previewState.currentFile = file;
    previewState.currentIndex = previewState.files.findIndex(f => f.id === file.id);

    // Update header
    document.getElementById('preview-filename').textContent = file.name;

    const contentUrl = '/api/v1/content/' + file.id;
    document.getElementById('preview-download').href = contentUrl;
    document.getElementById('preview-download').download = file.name;
    document.getElementById('preview-open-tab').href = contentUrl;

    // Update counter
    if (previewState.files.length > 1) {
      document.getElementById('preview-counter').textContent =
        `${previewState.currentIndex + 1} of ${previewState.files.length}`;
    } else {
      document.getElementById('preview-counter').textContent = '';
    }

    // Update nav buttons
    document.getElementById('preview-prev').disabled = previewState.currentIndex <= 0;
    document.getElementById('preview-next').disabled = previewState.currentIndex >= previewState.files.length - 1;

    // Hide all preview containers
    hideAllPreviews();

    // Show loading
    document.getElementById('preview-loading').classList.remove('hidden');

    // Show overlay
    overlay.classList.remove('hidden');
    document.body.style.overflow = 'hidden';

    // Determine preview type and load
    const previewType = getPreviewType(file.mime, file.name);
    loadPreview(file, previewType, contentUrl);
  }

  function hideAllPreviews() {
    document.getElementById('preview-loading')?.classList.add('hidden');
    document.getElementById('preview-image-container')?.classList.add('hidden');
    document.getElementById('preview-video-container')?.classList.add('hidden');
    document.getElementById('preview-audio-container')?.classList.add('hidden');
    document.getElementById('preview-pdf-container')?.classList.add('hidden');
    document.getElementById('preview-text-container')?.classList.add('hidden');
    document.getElementById('preview-unsupported')?.classList.add('hidden');

    // Stop any playing media
    const video = document.getElementById('preview-video');
    const audio = document.getElementById('preview-audio');
    if (video) { video.pause(); video.src = ''; }
    if (audio) { audio.pause(); audio.src = ''; }
  }

  function getPreviewType(mime, filename) {
    const ext = filename.split('.').pop().toLowerCase();

    if (mime.startsWith('image/')) return 'image';
    if (mime.startsWith('video/')) return 'video';
    if (mime.startsWith('audio/')) return 'audio';
    if (mime === 'application/pdf') return 'pdf';

    // Code files
    const codeExts = ['js', 'ts', 'jsx', 'tsx', 'py', 'rb', 'go', 'rs', 'java', 'c', 'cpp', 'h', 'hpp',
                      'cs', 'php', 'swift', 'kt', 'scala', 'html', 'css', 'scss', 'sass', 'less',
                      'xml', 'json', 'yaml', 'yml', 'toml', 'ini', 'sql', 'sh', 'bash', 'zsh',
                      'dockerfile', 'makefile', 'vue', 'svelte', 'graphql', 'gql', 'md', 'markdown'];
    if (codeExts.includes(ext)) return 'code';

    // Plain text
    const textExts = ['txt', 'log', 'text', 'readme', 'license', 'changelog', 'gitignore'];
    if (textExts.includes(ext) || mime.startsWith('text/')) return 'text';

    return 'unsupported';
  }

  function getLanguage(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    const langMap = {
      'js': 'javascript', 'jsx': 'javascript', 'ts': 'typescript', 'tsx': 'typescript',
      'py': 'python', 'rb': 'ruby', 'go': 'go', 'rs': 'rust', 'java': 'java',
      'c': 'c', 'cpp': 'cpp', 'h': 'c', 'hpp': 'cpp', 'cs': 'csharp', 'php': 'php',
      'swift': 'swift', 'kt': 'kotlin', 'scala': 'scala', 'html': 'html',
      'css': 'css', 'scss': 'scss', 'sass': 'scss', 'less': 'less',
      'xml': 'xml', 'json': 'json', 'yaml': 'yaml', 'yml': 'yaml',
      'toml': 'toml', 'ini': 'ini', 'sql': 'sql', 'sh': 'bash', 'bash': 'bash',
      'zsh': 'bash', 'dockerfile': 'dockerfile', 'makefile': 'makefile',
      'vue': 'html', 'svelte': 'html', 'graphql': 'graphql', 'gql': 'graphql',
      'md': 'markdown', 'markdown': 'markdown'
    };
    return langMap[ext] || 'plaintext';
  }

  async function loadPreview(file, type, url) {
    document.getElementById('preview-loading').classList.add('hidden');

    switch (type) {
      case 'image':
        loadImagePreview(url);
        break;
      case 'video':
        loadVideoPreview(url, file.mime);
        break;
      case 'audio':
        loadAudioPreview(url, file.name, file.mime);
        break;
      case 'pdf':
        await loadPdfPreview(url);
        break;
      case 'code':
      case 'text':
        await loadTextPreview(url, file.name, type);
        break;
      default:
        loadUnsupportedPreview(file.name, url);
    }
  }

  function loadImagePreview(url) {
    const container = document.getElementById('preview-image-container');
    const img = document.getElementById('preview-image');

    img.onload = () => {
      document.getElementById('preview-fileinfo').textContent = `${img.naturalWidth} × ${img.naturalHeight}`;
    };
    img.src = url;
    container.classList.remove('hidden');
  }

  function loadVideoPreview(url, mime) {
    const container = document.getElementById('preview-video-container');
    const video = document.getElementById('preview-video');

    video.src = url;
    video.type = mime;
    container.classList.remove('hidden');
    video.play().catch(() => {});
  }

  function loadAudioPreview(url, name, mime) {
    const container = document.getElementById('preview-audio-container');
    const audio = document.getElementById('preview-audio');

    document.getElementById('preview-audio-name').textContent = name;
    audio.src = url;
    audio.type = mime;
    container.classList.remove('hidden');
    audio.play().catch(() => {});
  }

  async function loadPdfPreview(url) {
    const container = document.getElementById('preview-pdf-container');
    container.classList.remove('hidden');

    try {
      const pdfjsLib = await loadPdfJs();
      const loadingTask = pdfjsLib.getDocument(url);
      previewState.pdfDoc = await loadingTask.promise;
      previewState.pdfPage = 1;
      previewState.pdfScale = 1.0;

      document.getElementById('pdf-total-pages').textContent = previewState.pdfDoc.numPages;
      document.getElementById('preview-fileinfo').textContent = `${previewState.pdfDoc.numPages} pages`;

      await renderPdfPage();
    } catch (err) {
      console.error('Failed to load PDF:', err);
      container.classList.add('hidden');
      loadUnsupportedPreview(previewState.currentFile.name, url);
    }
  }

  async function renderPdfPage() {
    if (!previewState.pdfDoc) return;

    const page = await previewState.pdfDoc.getPage(previewState.pdfPage);
    const canvas = document.getElementById('pdf-canvas');
    const ctx = canvas.getContext('2d');
    const viewer = document.getElementById('pdf-viewer');

    // Calculate scale to fit width
    const viewport = page.getViewport({ scale: 1.0 });
    const containerWidth = viewer.clientWidth - 32;
    let scale = previewState.pdfScale;

    if (scale === 1.0) {
      scale = containerWidth / viewport.width;
      previewState.pdfScale = scale;
    }

    const scaledViewport = page.getViewport({ scale });
    canvas.height = scaledViewport.height;
    canvas.width = scaledViewport.width;

    await page.render({
      canvasContext: ctx,
      viewport: scaledViewport
    }).promise;

    // Update UI
    document.getElementById('pdf-current-page').textContent = previewState.pdfPage;
    document.getElementById('pdf-zoom').textContent = Math.round(previewState.pdfScale * 100) + '%';
    document.getElementById('pdf-prev-page').disabled = previewState.pdfPage <= 1;
    document.getElementById('pdf-next-page').disabled = previewState.pdfPage >= previewState.pdfDoc.numPages;
  }

  function changePdfPage(delta) {
    if (!previewState.pdfDoc) return;
    const newPage = previewState.pdfPage + delta;
    if (newPage >= 1 && newPage <= previewState.pdfDoc.numPages) {
      previewState.pdfPage = newPage;
      renderPdfPage();
    }
  }

  function changePdfZoom(factor) {
    previewState.pdfScale = Math.min(Math.max(previewState.pdfScale * factor, 0.25), 5.0);
    renderPdfPage();
  }

  async function loadTextPreview(url, filename, type) {
    const container = document.getElementById('preview-text-container');

    try {
      const response = await fetch(url);
      const text = await response.text();

      document.getElementById('preview-text-filename').textContent = filename;
      const lang = getLanguage(filename);
      document.getElementById('preview-text-lang').textContent = lang;

      const codeEl = document.getElementById('preview-text-content');
      codeEl.textContent = text;
      codeEl.className = 'hljs language-' + lang;

      if (type === 'code') {
        const hljs = await loadHighlightJs();
        hljs.highlightElement(codeEl);
      }

      container.classList.remove('hidden');
      document.getElementById('preview-fileinfo').textContent = `${text.split('\n').length} lines`;
    } catch (err) {
      console.error('Failed to load text:', err);
      loadUnsupportedPreview(filename, url);
    }
  }

  function loadUnsupportedPreview(name, url) {
    const container = document.getElementById('preview-unsupported');
    document.getElementById('preview-unsupported-name').textContent = name;
    document.getElementById('preview-unsupported-download').href = url;
    document.getElementById('preview-unsupported-download').download = name;
    container.classList.remove('hidden');
    document.getElementById('preview-fileinfo').textContent = '';
  }

  function closePreview() {
    const overlay = document.getElementById('preview-overlay');
    if (!overlay) return;

    previewState.isOpen = false;
    previewState.currentFile = null;
    previewState.pdfDoc = null;

    hideAllPreviews();
    overlay.classList.add('hidden');
    document.body.style.overflow = '';
  }

  function navigatePreview(delta) {
    const newIndex = previewState.currentIndex + delta;
    if (newIndex >= 0 && newIndex < previewState.files.length) {
      const file = previewState.files[newIndex];
      openPreview(file);
    }
  }

  function toggleFullscreen() {
    const overlay = document.getElementById('preview-overlay');
    if (document.fullscreenElement) {
      document.exitFullscreen();
    } else {
      overlay.requestFullscreen();
    }
  }

  // Drag and Drop Upload
  function setupDragAndDrop() {
    const dropZone = document.getElementById('drop-zone');
    const fileInput = document.getElementById('file-input');
    const uploadList = document.getElementById('upload-list');
    const uploadSubmit = document.getElementById('upload-submit');

    if (!dropZone) return;

    dropZone.addEventListener('click', () => {
      fileInput.click();
    });

    fileInput.addEventListener('change', () => {
      handleFiles(fileInput.files);
    });

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

    if (uploadSubmit) {
      uploadSubmit.addEventListener('click', async () => {
        const files = fileInput.files;
        if (!files || files.length === 0) return;

        uploadSubmit.disabled = true;
        uploadSubmit.textContent = 'Uploading...';

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

    document.querySelectorAll('[data-type="file"], [data-type="folder"]').forEach(item => {
      item.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        currentItem = item;
        showContextMenu(e.clientX, e.clientY);
      });
    });

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
      contextMenu.style.left = x + 'px';
      contextMenu.style.top = y + 'px';

      const rect = contextMenu.getBoundingClientRect();
      if (rect.right > window.innerWidth) {
        contextMenu.style.left = (x - rect.width) + 'px';
      }
      if (rect.bottom > window.innerHeight) {
        contextMenu.style.top = (y - rect.height) + 'px';
      }

      contextMenu.classList.remove('hidden');
    }

    document.addEventListener('click', () => {
      contextMenu.classList.add('hidden');
      currentItem = null;
    });

    contextMenu.querySelectorAll('[data-action]').forEach(btn => {
      btn.addEventListener('click', () => {
        if (!currentItem) return;

        const action = btn.dataset.action;
        const id = currentItem.dataset.id;
        const type = currentItem.dataset.type;
        const name = currentItem.dataset.name;
        const mime = currentItem.dataset.mime;

        handleAction(action, { id, type, name, mime });
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
          openPreview(item);
        }
        break;

      case 'open':
        if (item.type === 'folder') {
          window.open('/files/' + item.id, '_blank');
        } else {
          window.open('/api/v1/content/' + item.id, '_blank');
        }
        break;

      case 'download':
        if (item.type === 'file') {
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

      // Preview overlay shortcuts
      if (previewState.isOpen) {
        switch (e.key) {
          case 'ArrowLeft':
            e.preventDefault();
            navigatePreview(-1);
            return;
          case 'ArrowRight':
            e.preventDefault();
            navigatePreview(1);
            return;
          case 'f':
            e.preventDefault();
            toggleFullscreen();
            return;
          case ' ':
            e.preventDefault();
            // Play/pause media
            const video = document.getElementById('preview-video');
            const audio = document.getElementById('preview-audio');
            if (video && !video.classList.contains('hidden')) {
              video.paused ? video.play() : video.pause();
            } else if (audio && !audio.classList.contains('hidden')) {
              audio.paused ? audio.play() : audio.pause();
            }
            return;
          case 'PageDown':
          case 'ArrowDown':
            if (previewState.pdfDoc) {
              e.preventDefault();
              changePdfPage(1);
            }
            return;
          case 'PageUp':
          case 'ArrowUp':
            if (previewState.pdfDoc) {
              e.preventDefault();
              changePdfPage(-1);
            }
            return;
        }
      }

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
          if (selectedItems.size === 1) {
            e.preventDefault();
            const id = Array.from(selectedItems)[0];
            const item = document.querySelector(`[data-id="${id}"]`);
            if (item) {
              handleAction('preview', {
                id: item.dataset.id,
                type: item.dataset.type,
                name: item.dataset.name,
                mime: item.dataset.mime,
              });
            }
          }
          break;
      }
    });
  }

  // Forms
  function setupForms() {
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

  // Load PDF.js on demand
  async function loadPdfJs() {
    if (window.pdfjsLib) return window.pdfjsLib;

    return new Promise((resolve, reject) => {
      import('https://cdnjs.cloudflare.com/ajax/libs/pdf.js/4.0.379/pdf.min.mjs')
        .then(pdfjsLib => {
          pdfjsLib.GlobalWorkerOptions.workerSrc = 'https://cdnjs.cloudflare.com/ajax/libs/pdf.js/4.0.379/pdf.worker.min.mjs';
          window.pdfjsLib = pdfjsLib;
          resolve(pdfjsLib);
        })
        .catch(reject);
    });
  }

  // Load highlight.js on demand
  async function loadHighlightJs() {
    if (window.hljs) return window.hljs;

    return new Promise((resolve) => {
      const link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = 'https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css';
      document.head.appendChild(link);

      const script = document.createElement('script');
      script.src = 'https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js';
      script.onload = () => resolve(window.hljs);
      document.head.appendChild(script);
    });
  }

  // =====================
  // COLUMN VIEW
  // =====================
  function setupColumnView() {
    const columnView = document.getElementById('column-view');
    if (!columnView) return;

    const columnsContainer = document.getElementById('columns-container');
    const previewPanel = document.getElementById('column-preview-panel');

    // Initialize first column
    const firstColumn = columnsContainer.querySelector('.column');
    if (firstColumn) {
      columnState.columns = [firstColumn];
      setupColumnItems(firstColumn);
    }

    // Setup preview panel buttons
    document.getElementById('column-preview-open')?.addEventListener('click', () => {
      if (columnState.selectedItem) {
        const { id, type, name, mime } = columnState.selectedItem;
        if (type === 'folder') {
          window.location.href = '/files/' + id;
        } else {
          openPreview({ id, name, mime });
        }
      }
    });

    document.getElementById('column-preview-download')?.addEventListener('click', () => {
      if (columnState.selectedItem && columnState.selectedItem.type === 'file') {
        const a = document.createElement('a');
        a.href = '/api/v1/content/' + columnState.selectedItem.id;
        a.download = columnState.selectedItem.name;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
      }
    });

    // Keyboard navigation for column view
    columnView.addEventListener('keydown', handleColumnKeydown);
    columnView.setAttribute('tabindex', '0');
  }

  function setupColumnItems(column) {
    const items = column.querySelectorAll('.column-item');
    items.forEach(item => {
      item.addEventListener('click', () => handleColumnItemClick(item, column));
    });
  }

  async function handleColumnItemClick(item, column) {
    const type = item.dataset.type;
    const id = item.dataset.id;
    const name = item.dataset.name;
    const mime = item.dataset.mime;
    const size = item.dataset.size;

    // Clear selection in this column and all following
    const columnIndex = columnState.columns.indexOf(column);
    columnState.columns.slice(columnIndex).forEach(col => {
      col.querySelectorAll('.column-item').forEach(i => {
        i.classList.remove('bg-blue-500', 'text-white');
        i.querySelector('svg')?.classList.remove('text-white');
        i.querySelector('span')?.classList.remove('text-white');
      });
    });

    // Remove columns after this one
    const columnsContainer = document.getElementById('columns-container');
    while (columnState.columns.length > columnIndex + 1) {
      const colToRemove = columnState.columns.pop();
      colToRemove.remove();
    }

    // Highlight selected item
    item.classList.add('bg-blue-500');
    item.querySelector('svg')?.classList.add('text-white');
    item.querySelector('span')?.classList.add('text-white');

    columnState.selectedItem = { id, type, name, mime, size };

    if (type === 'folder') {
      // Load folder contents in new column
      await loadColumnContents(id, name);
      // Hide preview panel for folders (or show folder info)
      updateColumnPreview({ id, type, name });
    } else {
      // Show file preview panel
      updateColumnPreview({ id, type, name, mime, size });
    }

    // Scroll to show new column
    columnsContainer.scrollLeft = columnsContainer.scrollWidth;
  }

  async function loadColumnContents(folderId, folderName) {
    const columnsContainer = document.getElementById('columns-container');

    try {
      const response = await fetch('/api/v1/folders/children/' + folderId);
      if (!response.ok) throw new Error('Failed to load folder');

      const data = await response.json();

      // Create new column
      const newColumn = document.createElement('div');
      newColumn.className = 'column flex-shrink-0 w-56 border-r border-zinc-200 flex flex-col';
      newColumn.dataset.path = folderId;

      let itemsHtml = '';

      // Add folders
      if (data.folders) {
        data.folders.forEach(folder => {
          itemsHtml += `
            <div class="column-item group flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-zinc-100 transition-colors" data-type="folder" data-id="${escapeHtml(folder.id)}" data-name="${escapeHtml(folder.name)}">
              <svg class="w-4 h-4 text-zinc-400 flex-shrink-0" viewBox="0 0 24 24" fill="currentColor" opacity="0.3" stroke="currentColor" stroke-width="1.5">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              </svg>
              <span class="text-sm text-zinc-700 truncate flex-1">${escapeHtml(folder.name)}</span>
              <svg class="w-4 h-4 text-zinc-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m9 18 6-6-6-6"/></svg>
            </div>
          `;
        });
      }

      // Add files
      if (data.files) {
        data.files.forEach(file => {
          const icon = getFileIcon(file.mime_type);
          itemsHtml += `
            <div class="column-item group flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-zinc-100 transition-colors" data-type="file" data-id="${escapeHtml(file.id)}" data-name="${escapeHtml(file.name)}" data-mime="${escapeHtml(file.mime_type)}" data-size="${file.size}">
              <div class="w-4 h-4 text-zinc-400 flex-shrink-0">${icon}</div>
              <span class="text-sm text-zinc-700 truncate flex-1">${escapeHtml(file.name)}</span>
            </div>
          `;
        });
      }

      newColumn.innerHTML = `
        <div class="column-header px-3 py-2 bg-zinc-50 border-b border-zinc-200 text-xs font-medium text-zinc-500 uppercase tracking-wider truncate">
          ${escapeHtml(folderName)}
        </div>
        <div class="column-content flex-1 overflow-y-auto">
          ${itemsHtml || '<div class="px-3 py-4 text-sm text-zinc-400 text-center">Empty folder</div>'}
        </div>
      `;

      columnsContainer.appendChild(newColumn);
      columnState.columns.push(newColumn);
      setupColumnItems(newColumn);

    } catch (err) {
      console.error('Failed to load column:', err);
    }
  }

  async function updateColumnPreview(item) {
    const previewPanel = document.getElementById('column-preview-panel');
    const previewImg = document.getElementById('column-preview-img');
    const previewIcon = document.getElementById('column-preview-icon');
    const previewName = document.getElementById('column-preview-name');
    const previewKind = document.getElementById('column-preview-kind');
    const previewSize = document.getElementById('column-preview-size');
    const previewDate = document.getElementById('column-preview-date');
    const dimsRow = document.getElementById('column-preview-dimensions');
    const dimsValue = document.getElementById('column-preview-dims');
    const cameraRow = document.getElementById('column-preview-camera');
    const cameraValue = document.getElementById('column-preview-camera-model');
    const durationRow = document.getElementById('column-preview-duration');
    const durationValue = document.getElementById('column-preview-dur');

    previewPanel.classList.remove('hidden');
    previewName.textContent = item.name;

    // Hide optional rows
    dimsRow?.classList.add('hidden');
    cameraRow?.classList.add('hidden');
    durationRow?.classList.add('hidden');

    if (item.type === 'folder') {
      previewImg.classList.add('hidden');
      previewIcon.classList.remove('hidden');
      previewIcon.innerHTML = `<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>`;
      previewKind.textContent = 'Folder';
      previewSize.textContent = '-';
      previewDate.textContent = '-';
    } else {
      // Check if image
      if (item.mime && item.mime.startsWith('image/')) {
        previewImg.src = '/api/v1/thumbnail/' + item.id;
        previewImg.classList.remove('hidden');
        previewIcon.classList.add('hidden');
      } else {
        previewImg.classList.add('hidden');
        previewIcon.classList.remove('hidden');
      }

      previewKind.textContent = getMimeDescription(item.mime);
      previewSize.textContent = item.size ? formatSize(parseInt(item.size)) : '-';
      previewDate.textContent = '-';

      // Fetch metadata for more details
      try {
        const response = await fetch('/api/v1/metadata/' + item.id);
        if (response.ok) {
          const meta = await response.json();

          if (meta.size) {
            previewSize.textContent = formatSize(meta.size);
          }

          // Image metadata
          if (meta.image) {
            if (meta.image.width && meta.image.height) {
              dimsRow?.classList.remove('hidden');
              dimsValue.textContent = `${meta.image.width} × ${meta.image.height}`;
            }
            if (meta.image.camera_make || meta.image.camera_model) {
              cameraRow?.classList.remove('hidden');
              cameraValue.textContent = [meta.image.camera_make, meta.image.camera_model].filter(Boolean).join(' ');
            }
          }

          // Video metadata
          if (meta.video) {
            if (meta.video.width && meta.video.height) {
              dimsRow?.classList.remove('hidden');
              dimsValue.textContent = `${meta.video.width} × ${meta.video.height}`;
            }
            if (meta.video.duration) {
              durationRow?.classList.remove('hidden');
              durationValue.textContent = formatDuration(meta.video.duration);
            }
          }

          // Audio metadata
          if (meta.audio && meta.audio.duration) {
            durationRow?.classList.remove('hidden');
            durationValue.textContent = formatDuration(meta.audio.duration);
          }
        }
      } catch (err) {
        console.error('Failed to fetch metadata:', err);
      }
    }
  }

  function handleColumnKeydown(e) {
    const columnsContainer = document.getElementById('columns-container');
    if (!columnsContainer) return;

    const activeColumn = columnState.columns[columnState.columns.length - 1];
    if (!activeColumn) return;

    const items = activeColumn.querySelectorAll('.column-item');
    const selectedItem = activeColumn.querySelector('.column-item.bg-blue-500');
    let currentIndex = Array.from(items).indexOf(selectedItem);

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        if (currentIndex < items.length - 1) {
          items[currentIndex + 1].click();
        }
        break;

      case 'ArrowUp':
        e.preventDefault();
        if (currentIndex > 0) {
          items[currentIndex - 1].click();
        }
        break;

      case 'ArrowRight':
        e.preventDefault();
        if (columnState.selectedItem?.type === 'folder') {
          // Move to next column if exists
          const nextColumnIndex = columnState.columns.indexOf(activeColumn) + 1;
          if (nextColumnIndex < columnState.columns.length) {
            const nextColumn = columnState.columns[nextColumnIndex];
            const firstItem = nextColumn.querySelector('.column-item');
            if (firstItem) firstItem.click();
          }
        }
        break;

      case 'ArrowLeft':
        e.preventDefault();
        // Move to previous column
        const prevColumnIndex = columnState.columns.indexOf(activeColumn) - 1;
        if (prevColumnIndex >= 0) {
          const prevColumn = columnState.columns[prevColumnIndex];
          const selectedInPrev = prevColumn.querySelector('.column-item.bg-blue-500');
          if (selectedInPrev) {
            // Re-select to update state
            handleColumnItemClick(selectedInPrev, prevColumn);
          }
        }
        break;

      case 'Enter':
        e.preventDefault();
        if (columnState.selectedItem) {
          const { id, type, name, mime } = columnState.selectedItem;
          if (type === 'folder') {
            window.location.href = '/files/' + id;
          } else {
            openPreview({ id, name, mime });
          }
        }
        break;
    }
  }

  // =====================
  // GALLERY VIEW
  // =====================
  function setupGalleryView() {
    const galleryView = document.getElementById('gallery-view');
    if (!galleryView) return;

    // Collect all items
    const thumbs = document.querySelectorAll('.gallery-thumb');
    galleryState.items = Array.from(thumbs).map(thumb => ({
      id: thumb.dataset.id,
      type: thumb.dataset.type,
      name: thumb.dataset.name,
      mime: thumb.dataset.mime,
      element: thumb
    }));

    // Setup thumbnail click handlers
    thumbs.forEach((thumb, index) => {
      thumb.addEventListener('click', () => {
        selectGalleryItem(index);
      });
    });

    // Setup navigation buttons
    document.getElementById('gallery-prev')?.addEventListener('click', () => navigateGallery(-1));
    document.getElementById('gallery-next')?.addEventListener('click', () => navigateGallery(1));

    // Keyboard navigation
    galleryView.addEventListener('keydown', handleGalleryKeydown);
    galleryView.setAttribute('tabindex', '0');

    // Auto-select first file item
    const firstFileIndex = galleryState.items.findIndex(item => item.type === 'file');
    if (firstFileIndex >= 0) {
      selectGalleryItem(firstFileIndex);
    }

    // Show navigation on hover
    const mainArea = document.getElementById('gallery-main');
    if (mainArea) {
      mainArea.addEventListener('mouseenter', () => {
        document.getElementById('gallery-prev')?.classList.add('opacity-60');
        document.getElementById('gallery-next')?.classList.add('opacity-60');
      });
      mainArea.addEventListener('mouseleave', () => {
        document.getElementById('gallery-prev')?.classList.remove('opacity-60');
        document.getElementById('gallery-next')?.classList.remove('opacity-60');
      });
    }
  }

  async function selectGalleryItem(index) {
    if (index < 0 || index >= galleryState.items.length) return;

    const item = galleryState.items[index];
    galleryState.currentIndex = index;
    galleryState.currentItem = item;

    // Update thumbnail selection
    galleryState.items.forEach((i, idx) => {
      if (idx === index) {
        i.element.classList.add('ring-2', 'ring-white');
      } else {
        i.element.classList.remove('ring-2', 'ring-white');
      }
    });

    // Scroll thumbnail into view
    item.element.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' });

    // Update navigation buttons
    document.getElementById('gallery-prev').disabled = index === 0;
    document.getElementById('gallery-next').disabled = index === galleryState.items.length - 1;

    // Handle folders (double-click to navigate)
    if (item.type === 'folder') {
      showGalleryPlaceholder('Double-click to open folder');
      updateGalleryInfo(item);
      return;
    }

    // Update main preview
    await updateGalleryPreview(item);

    // Update info panel
    await updateGalleryInfo(item);
  }

  async function updateGalleryPreview(item) {
    const img = document.getElementById('gallery-preview-img');
    const video = document.getElementById('gallery-preview-video');
    const audio = document.getElementById('gallery-preview-audio');
    const placeholder = document.getElementById('gallery-preview-placeholder');
    const loading = document.getElementById('gallery-loading');

    // Hide all
    img.classList.add('hidden');
    video.classList.add('hidden');
    audio.classList.add('hidden');
    placeholder.classList.add('hidden');
    loading.classList.remove('hidden');

    // Stop any playing media
    video.pause();
    video.src = '';
    audio.pause();
    audio.src = '';

    const contentUrl = '/api/v1/content/' + item.id;
    const mime = item.mime || '';

    if (mime.startsWith('image/')) {
      img.onload = () => loading.classList.add('hidden');
      img.onerror = () => {
        loading.classList.add('hidden');
        showGalleryPlaceholder('Failed to load image');
      };
      img.src = contentUrl;
      img.classList.remove('hidden');
    } else if (mime.startsWith('video/')) {
      loading.classList.add('hidden');
      video.src = contentUrl;
      video.classList.remove('hidden');
      video.play().catch(() => {});
    } else if (mime.startsWith('audio/')) {
      loading.classList.add('hidden');
      audio.src = contentUrl;
      audio.classList.remove('hidden');
      audio.play().catch(() => {});
    } else {
      loading.classList.add('hidden');
      showGalleryPlaceholder('Preview not available');
    }
  }

  function showGalleryPlaceholder(message) {
    const placeholder = document.getElementById('gallery-preview-placeholder');
    const img = document.getElementById('gallery-preview-img');
    const video = document.getElementById('gallery-preview-video');
    const audio = document.getElementById('gallery-preview-audio');

    img.classList.add('hidden');
    video.classList.add('hidden');
    audio.classList.add('hidden');
    placeholder.querySelector('span').textContent = message;
    placeholder.classList.remove('hidden');
  }

  async function updateGalleryInfo(item) {
    // Update basic info
    document.getElementById('gallery-info-name').textContent = item.name;
    document.getElementById('gallery-info-type').textContent = item.mime || (item.type === 'folder' ? 'Folder' : '');
    document.getElementById('gallery-info-kind').textContent = item.type === 'folder' ? 'Folder' : getMimeDescription(item.mime);
    document.getElementById('gallery-info-size').textContent = '-';
    document.getElementById('gallery-info-created').textContent = '-';
    document.getElementById('gallery-info-modified').textContent = '-';

    // Hide all optional sections
    document.getElementById('gallery-media-info')?.classList.add('hidden');
    document.getElementById('gallery-camera-info')?.classList.add('hidden');
    document.getElementById('gallery-audio-info')?.classList.add('hidden');
    document.getElementById('gallery-location-info')?.classList.add('hidden');

    // Hide all optional rows
    document.querySelectorAll('[id^="gallery-"][id$="-row"]').forEach(row => {
      if (!row.id.includes('info-name') && !row.id.includes('info-type')) {
        row.classList.add('hidden');
      }
    });

    if (item.type === 'folder') return;

    // Fetch metadata
    try {
      const response = await fetch('/api/v1/metadata/' + item.id);
      if (!response.ok) return;

      const meta = await response.json();

      // Basic info
      if (meta.size) {
        document.getElementById('gallery-info-size').textContent = formatSize(meta.size);
      }

      // Image metadata
      if (meta.image) {
        const mediaInfo = document.getElementById('gallery-media-info');
        mediaInfo?.classList.remove('hidden');

        if (meta.image.width && meta.image.height) {
          document.getElementById('gallery-dims-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-dims').textContent = `${meta.image.width} × ${meta.image.height}`;
        }

        // Camera info
        if (meta.image.camera_make || meta.image.camera_model || meta.image.aperture || meta.image.exposure_time || meta.image.iso) {
          const cameraInfo = document.getElementById('gallery-camera-info');
          cameraInfo?.classList.remove('hidden');

          if (meta.image.camera_make || meta.image.camera_model) {
            document.getElementById('gallery-camera-row')?.classList.remove('hidden');
            document.getElementById('gallery-info-camera').textContent = [meta.image.camera_make, meta.image.camera_model].filter(Boolean).join(' ');
          }
          if (meta.image.aperture) {
            document.getElementById('gallery-aperture-row')?.classList.remove('hidden');
            document.getElementById('gallery-info-aperture').textContent = `f/${meta.image.aperture}`;
          }
          if (meta.image.exposure_time) {
            document.getElementById('gallery-exposure-row')?.classList.remove('hidden');
            document.getElementById('gallery-info-exposure').textContent = meta.image.exposure_time;
          }
          if (meta.image.iso) {
            document.getElementById('gallery-iso-row')?.classList.remove('hidden');
            document.getElementById('gallery-info-iso').textContent = meta.image.iso;
          }
          if (meta.image.focal_length) {
            document.getElementById('gallery-focal-row')?.classList.remove('hidden');
            document.getElementById('gallery-info-focal').textContent = `${meta.image.focal_length}mm`;
          }
        }

        // GPS location
        if (meta.image.latitude && meta.image.longitude) {
          const locationInfo = document.getElementById('gallery-location-info');
          locationInfo?.classList.remove('hidden');
          document.getElementById('gallery-location-coords').textContent =
            `${meta.image.latitude.toFixed(6)}, ${meta.image.longitude.toFixed(6)}`;
        }
      }

      // Video metadata
      if (meta.video) {
        const mediaInfo = document.getElementById('gallery-media-info');
        mediaInfo?.classList.remove('hidden');

        if (meta.video.width && meta.video.height) {
          document.getElementById('gallery-dims-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-dims').textContent = `${meta.video.width} × ${meta.video.height}`;
        }
        if (meta.video.duration) {
          document.getElementById('gallery-duration-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-duration').textContent = formatDuration(meta.video.duration);
        }
        if (meta.video.video_codec) {
          document.getElementById('gallery-codec-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-codec').textContent = meta.video.video_codec;
        }
      }

      // Audio metadata
      if (meta.audio) {
        const mediaInfo = document.getElementById('gallery-media-info');
        const audioInfo = document.getElementById('gallery-audio-info');
        mediaInfo?.classList.remove('hidden');
        audioInfo?.classList.remove('hidden');

        if (meta.audio.duration) {
          document.getElementById('gallery-duration-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-duration').textContent = formatDuration(meta.audio.duration);
        }
        if (meta.audio.artist) {
          document.getElementById('gallery-artist-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-artist').textContent = meta.audio.artist;
        }
        if (meta.audio.album) {
          document.getElementById('gallery-album-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-album').textContent = meta.audio.album;
        }
        if (meta.audio.genre) {
          document.getElementById('gallery-genre-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-genre').textContent = meta.audio.genre;
        }
        if (meta.audio.year) {
          document.getElementById('gallery-year-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-year').textContent = meta.audio.year;
        }
        if (meta.audio.bitrate) {
          document.getElementById('gallery-bitrate-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-bitrate').textContent = `${Math.round(meta.audio.bitrate / 1000)} kbps`;
        }
      }

      // Document metadata
      if (meta.document) {
        const mediaInfo = document.getElementById('gallery-media-info');
        mediaInfo?.classList.remove('hidden');

        if (meta.document.page_count) {
          document.getElementById('gallery-dims-row')?.classList.remove('hidden');
          document.getElementById('gallery-info-dims').textContent = `${meta.document.page_count} pages`;
        }
      }

    } catch (err) {
      console.error('Failed to fetch metadata:', err);
    }
  }

  function navigateGallery(delta) {
    const newIndex = galleryState.currentIndex + delta;
    if (newIndex >= 0 && newIndex < galleryState.items.length) {
      selectGalleryItem(newIndex);
    }
  }

  function handleGalleryKeydown(e) {
    switch (e.key) {
      case 'ArrowLeft':
        e.preventDefault();
        navigateGallery(-1);
        break;

      case 'ArrowRight':
        e.preventDefault();
        navigateGallery(1);
        break;

      case 'Enter':
        e.preventDefault();
        if (galleryState.currentItem) {
          if (galleryState.currentItem.type === 'folder') {
            window.location.href = '/files/' + galleryState.currentItem.id;
          } else {
            openPreview(galleryState.currentItem);
          }
        }
        break;

      case ' ':
        e.preventDefault();
        // Toggle play/pause for media
        const video = document.getElementById('gallery-preview-video');
        const audio = document.getElementById('gallery-preview-audio');
        if (!video.classList.contains('hidden')) {
          video.paused ? video.play() : video.pause();
        } else if (!audio.classList.contains('hidden')) {
          audio.paused ? audio.play() : audio.pause();
        }
        break;
    }
  }

  // Helper: Get file icon SVG
  function getFileIcon(mime) {
    if (!mime) {
      return '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>';
    }
    if (mime.startsWith('image/')) {
      return '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect width="18" height="18" x="3" y="3" rx="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>';
    }
    if (mime.startsWith('video/')) {
      return '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2"/></svg>';
    }
    if (mime.startsWith('audio/')) {
      return '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>';
    }
    return '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>';
  }

  // Helper: Get MIME type description
  function getMimeDescription(mime) {
    if (!mime) return 'File';
    const descriptions = {
      'image/jpeg': 'JPEG Image',
      'image/png': 'PNG Image',
      'image/gif': 'GIF Image',
      'image/webp': 'WebP Image',
      'image/svg+xml': 'SVG Image',
      'video/mp4': 'MP4 Video',
      'video/webm': 'WebM Video',
      'video/quicktime': 'QuickTime Video',
      'audio/mpeg': 'MP3 Audio',
      'audio/mp3': 'MP3 Audio',
      'audio/wav': 'WAV Audio',
      'audio/flac': 'FLAC Audio',
      'audio/ogg': 'OGG Audio',
      'audio/aac': 'AAC Audio',
      'application/pdf': 'PDF Document',
      'application/zip': 'ZIP Archive',
      'text/plain': 'Text File',
      'text/html': 'HTML Document',
      'text/css': 'CSS Stylesheet',
      'application/javascript': 'JavaScript File',
      'application/json': 'JSON File'
    };
    return descriptions[mime] || mime.split('/')[1]?.toUpperCase() + ' File' || 'File';
  }

  // Helper: Format duration (seconds to MM:SS or HH:MM:SS)
  function formatDuration(seconds) {
    if (!seconds || seconds <= 0) return '-';
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);
    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }
})();
