// Drive Frontend JavaScript

(function() {
  'use strict';

  // State
  let selectedItems = new Set();
  let currentPath = '';
  let clipboard = { items: [], action: null };
  let selectedFile = null; // Currently selected (highlighted) file

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

  // Sidebar State (for List/Grid views)
  let sidebarState = {
    isOpen: false,
    width: 288, // default 18rem
    selectedItem: null,
    viewType: null // 'list' or 'grid'
  };

  // =====================
  // VIEW PERSISTENCE
  // =====================
  function saveViewPreference(view) {
    localStorage.setItem('drive_view_mode', view);
  }

  function getViewPreference() {
    return localStorage.getItem('drive_view_mode') || 'grid';
  }

  function saveSortPreference(sortBy, sortOrder) {
    localStorage.setItem('drive_sort_by', sortBy);
    localStorage.setItem('drive_sort_order', sortOrder);
  }

  function getSortPreference() {
    return {
      sortBy: localStorage.getItem('drive_sort_by') || 'name',
      sortOrder: localStorage.getItem('drive_sort_order') || 'asc'
    };
  }

  // Initialize view mode from localStorage - redirect if needed
  function initViewMode() {
    const urlParams = new URLSearchParams(window.location.search);
    const currentView = urlParams.get('view');
    const savedView = getViewPreference();
    const { sortBy, sortOrder } = getSortPreference();

    // If no view in URL and saved preference differs from default
    if (!currentView && savedView !== 'grid') {
      const url = new URL(window.location);
      url.searchParams.set('view', savedView);
      if (!urlParams.get('sort')) url.searchParams.set('sort', sortBy);
      if (!urlParams.get('order')) url.searchParams.set('order', sortOrder);
      window.location.replace(url);
      return false;
    }

    // Save current view if specified in URL
    if (currentView) {
      saveViewPreference(currentView);
    }

    // Save sort if specified
    if (urlParams.get('sort')) {
      saveSortPreference(urlParams.get('sort'), urlParams.get('order') || 'asc');
    }

    return true;
  }

  // Update navigation links to preserve view preference
  function updateNavigationLinks() {
    const viewMode = getViewPreference();
    const { sortBy, sortOrder } = getSortPreference();

    // Update folder links to include view parameters
    document.querySelectorAll('[data-type="folder"] a').forEach(link => {
      const url = new URL(link.href, window.location.origin);
      url.searchParams.set('view', viewMode);
      url.searchParams.set('sort', sortBy);
      url.searchParams.set('order', sortOrder);
      link.href = url.toString();
    });

    // Update breadcrumb links
    document.querySelectorAll('nav[aria-label="Breadcrumb"] a').forEach(link => {
      const url = new URL(link.href, window.location.origin);
      url.searchParams.set('view', viewMode);
      url.searchParams.set('sort', sortBy);
      url.searchParams.set('order', sortOrder);
      link.href = url.toString();
    });
  }

  // DOM Ready
  document.addEventListener('DOMContentLoaded', init);

  function init() {
    // Check view mode first - may redirect
    if (!initViewMode()) return;

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
    setupSidebar();
    updateNavigationLinks();
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

  // File Selection - Finder-like behavior
  // Single click: select item
  // Double click: open folder / preview file
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
            item.classList.remove('ring-2', 'ring-blue-500');
          } else {
            selectedItems.add(id);
            item.classList.add('ring-2', 'ring-blue-500');
          }
        } else if (e.shiftKey) {
          // Range select (TODO)
        } else {
          // Single click - select item (Finder-like behavior)
          clearSelection();
          selectedItems.add(id);
          item.classList.add('ring-2', 'ring-blue-500');
          selectedFile = { id, type, name, mime, element: item };
        }
      });

      // Double-click to navigate/preview
      item.addEventListener('dblclick', (e) => {
        e.preventDefault();
        const type = item.dataset.type;
        const id = item.dataset.id;
        const name = item.dataset.name;
        const mime = item.dataset.mime;

        if (type === 'folder') {
          // Navigate to folder with preserved view preferences
          const viewMode = getViewPreference();
          const { sortBy, sortOrder } = getSortPreference();
          const url = new URL('/files/' + id, window.location.origin);
          url.searchParams.set('view', viewMode);
          url.searchParams.set('sort', sortBy);
          url.searchParams.set('order', sortOrder);
          window.location.href = url.toString();
        } else {
          // Open preview overlay for files
          openPreview({ id, name, mime });
        }
      });
    });
  }

  function clearSelection() {
    selectedItems.clear();
    selectedFile = null;
    document.querySelectorAll('[data-type="file"], [data-type="folder"]').forEach(item => {
      item.classList.remove('ring-2', 'ring-blue-500', 'ring-zinc-900', 'dark:ring-zinc-100');
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

    // Add error handling
    video.onerror = () => {
      container.classList.add('hidden');
      showMediaError('Video format not supported by your browser', url, previewState.currentFile?.name);
    };

    video.oncanplay = () => {
      document.getElementById('preview-fileinfo').textContent = `${video.videoWidth} × ${video.videoHeight}`;
    };

    video.src = url;
    video.type = mime;
    container.classList.remove('hidden');
    video.play().catch(() => {});
  }

  function loadAudioPreview(url, name, mime) {
    const container = document.getElementById('preview-audio-container');
    const audio = document.getElementById('preview-audio');

    // Add error handling
    audio.onerror = () => {
      container.classList.add('hidden');
      showMediaError('Audio format not supported by your browser', url, name);
    };

    audio.onloadedmetadata = () => {
      document.getElementById('preview-fileinfo').textContent = formatDuration(audio.duration);
    };

    document.getElementById('preview-audio-name').textContent = name;
    audio.src = url;
    audio.type = mime;
    container.classList.remove('hidden');
    audio.play().catch(() => {});
  }

  // Show error message for unsupported media formats
  function showMediaError(message, url, filename) {
    const container = document.getElementById('preview-unsupported');
    document.getElementById('preview-unsupported-name').textContent = filename || 'Unknown file';
    document.getElementById('preview-unsupported-download').href = url;
    document.getElementById('preview-unsupported-download').download = filename || '';

    // Update message if there's a message element
    const msgEl = document.getElementById('preview-unsupported-message');
    if (msgEl) {
      msgEl.textContent = message;
    }

    container.classList.remove('hidden');
    document.getElementById('preview-fileinfo').textContent = message;
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
              item.classList.add('ring-2', 'ring-blue-500');
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

        case ' ':
          // Space bar - Quick Look style preview (like Finder)
          if (selectedFile && selectedFile.type === 'file') {
            e.preventDefault();
            openPreview({
              id: selectedFile.id,
              name: selectedFile.name,
              mime: selectedFile.mime
            });
          }
          break;

        case 'Enter':
          // Enter key - Open folder or preview file
          if (selectedFile) {
            e.preventDefault();
            if (selectedFile.type === 'folder') {
              const viewMode = getViewPreference();
              const { sortBy, sortOrder } = getSortPreference();
              const url = new URL('/files/' + selectedFile.id, window.location.origin);
              url.searchParams.set('view', viewMode);
              url.searchParams.set('sort', sortBy);
              url.searchParams.set('order', sortOrder);
              window.location.href = url.toString();
            } else {
              openPreview({
                id: selectedFile.id,
                name: selectedFile.name,
                mime: selectedFile.mime
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

    // Initialize columns from current URL path (for deep links)
    initColumnsFromPath();

    // Setup preview panel buttons
    document.getElementById('column-preview-open')?.addEventListener('click', () => {
      if (columnState.selectedItem) {
        const { id, type, name, mime } = columnState.selectedItem;
        if (type === 'folder') {
          const viewMode = getViewPreference();
          const { sortBy, sortOrder } = getSortPreference();
          const url = new URL('/files/' + id, window.location.origin);
          url.searchParams.set('view', viewMode);
          url.searchParams.set('sort', sortBy);
          url.searchParams.set('order', sortOrder);
          window.location.href = url.toString();
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

    // Setup close button for big preview
    document.getElementById('column-big-preview-close')?.addEventListener('click', () => {
      hideBigPreview();
    });

    // Keyboard navigation for column view
    columnView.addEventListener('keydown', handleColumnKeydown);
    columnView.setAttribute('tabindex', '0');
  }

  // Initialize columns from URL path - load all ancestor folders
  async function initColumnsFromPath() {
    const path = window.location.pathname.replace('/files', '').replace(/^\//, '');
    if (!path) return;

    const parts = path.split('/').filter(p => p);
    if (parts.length === 0) return;

    let currentPath = '';

    // Load each level of the path hierarchy
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const prevPath = currentPath;
      currentPath = currentPath ? `${currentPath}/${part}` : part;

      // Find the item in the current column and select it
      const currentColumn = columnState.columns[columnState.columns.length - 1];
      if (currentColumn) {
        const item = currentColumn.querySelector(`[data-id="${CSS.escape(currentPath)}"]`) ||
                     currentColumn.querySelector(`[data-name="${CSS.escape(part)}"]`);

        if (item) {
          // Highlight the item
          currentColumn.querySelectorAll('.column-item').forEach(i => {
            i.classList.remove('bg-blue-500', 'text-white');
            i.querySelector('svg')?.classList.remove('text-white');
            i.querySelector('span')?.classList.remove('text-white');
          });

          item.classList.add('bg-blue-500');
          item.querySelector('svg')?.classList.add('text-white');
          item.querySelector('span')?.classList.add('text-white');

          // If it's a folder (not the last item or if the last item is a folder), load its contents
          if (item.dataset.type === 'folder' && i < parts.length - 1) {
            await loadColumnContents(currentPath, part);
          } else if (i === parts.length - 1) {
            // Last item - update preview panel
            columnState.selectedItem = {
              id: item.dataset.id,
              type: item.dataset.type,
              name: item.dataset.name,
              mime: item.dataset.mime,
              size: item.dataset.size
            };

            if (item.dataset.type === 'folder') {
              await loadColumnContents(currentPath, part);
            }
            updateColumnPreview(columnState.selectedItem);
          }
        }
      }
    }

    // Scroll to show the last column
    const columnsContainer = document.getElementById('columns-container');
    if (columnsContainer) {
      columnsContainer.scrollLeft = columnsContainer.scrollWidth;
    }
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
      // Hide big preview for folders
      hideBigPreview();
      // Load folder contents in new column
      await loadColumnContents(id, name);
      // Hide preview panel for folders (or show folder info)
      updateColumnPreview({ id, type, name });
    } else {
      // Show big preview column for files
      showBigPreview({ id, type, name, mime, size });
      // Show file preview panel
      updateColumnPreview({ id, type, name, mime, size });
    }

    // Scroll to show new column
    columnsContainer.scrollLeft = columnsContainer.scrollWidth;
  }

  // Column preview state
  let columnPdfState = {
    doc: null,
    page: 1,
    scale: 1.0
  };

  // Show big preview column for files in column view
  async function showBigPreview(item) {
    const bigPreview = document.getElementById('column-big-preview');
    if (!bigPreview) return;

    const img = document.getElementById('column-big-preview-img');
    const video = document.getElementById('column-big-preview-video');
    const audioContainer = document.getElementById('column-big-preview-audio-container');
    const audio = document.getElementById('column-big-preview-audio');
    const pdfContainer = document.getElementById('column-big-preview-pdf');
    const markdownContainer = document.getElementById('column-big-preview-markdown');
    const codeContainer = document.getElementById('column-big-preview-code');
    const placeholder = document.getElementById('column-big-preview-placeholder');
    const loading = document.getElementById('column-big-preview-loading');
    const title = document.getElementById('column-big-preview-title');

    // Hide all content types first
    img?.classList.add('hidden');
    video?.classList.add('hidden');
    audioContainer?.classList.add('hidden');
    pdfContainer?.classList.add('hidden');
    markdownContainer?.classList.add('hidden');
    codeContainer?.classList.add('hidden');
    placeholder?.classList.add('hidden');
    loading?.classList.remove('hidden');

    // Stop any playing media
    if (video) {
      video.pause();
      video.src = '';
    }
    if (audio) {
      audio.pause();
      audio.src = '';
    }

    // Show the preview column
    bigPreview.classList.remove('hidden');
    if (title) title.textContent = item.name;

    const contentUrl = '/api/v1/content/' + item.id;
    const mime = item.mime || '';
    const ext = item.name.split('.').pop().toLowerCase();

    // Determine preview type
    const previewType = getColumnPreviewType(mime, ext);

    try {
      switch (previewType) {
        case 'image':
          loadColumnImagePreview(img, contentUrl, loading, item.name);
          break;
        case 'video':
          loading?.classList.add('hidden');
          video.onerror = () => {
            video.classList.add('hidden');
            showBigPreviewPlaceholder(item.name);
          };
          video.src = contentUrl;
          video.classList.remove('hidden');
          break;
        case 'audio':
          loading?.classList.add('hidden');
          audio.onerror = () => {
            audioContainer.classList.add('hidden');
            showBigPreviewPlaceholder(item.name);
          };
          audio.src = contentUrl;
          audioContainer.classList.remove('hidden');
          break;
        case 'pdf':
          await loadColumnPdfPreview(contentUrl);
          loading?.classList.add('hidden');
          break;
        case 'markdown':
          await loadColumnMarkdownPreview(contentUrl, item.name);
          loading?.classList.add('hidden');
          break;
        case 'code':
          await loadColumnCodePreview(contentUrl, item.name);
          loading?.classList.add('hidden');
          break;
        default:
          loading?.classList.add('hidden');
          showBigPreviewPlaceholder(item.name);
      }
    } catch (err) {
      console.error('Preview error:', err);
      loading?.classList.add('hidden');
      showBigPreviewPlaceholder(item.name);
    }

    // Scroll to show preview
    const columnsContainer = document.getElementById('columns-container');
    if (columnsContainer) {
      setTimeout(() => {
        columnsContainer.scrollLeft = columnsContainer.scrollWidth;
      }, 50);
    }
  }

  function getColumnPreviewType(mime, ext) {
    if (mime.startsWith('image/')) return 'image';
    if (mime.startsWith('video/')) return 'video';
    if (mime.startsWith('audio/')) return 'audio';
    if (mime === 'application/pdf') return 'pdf';

    // Markdown files
    if (['md', 'markdown', 'mdown', 'mkdn'].includes(ext)) return 'markdown';

    // Code files
    const codeExts = ['js', 'ts', 'jsx', 'tsx', 'py', 'rb', 'go', 'rs', 'java', 'c', 'cpp', 'h', 'hpp',
                      'cs', 'php', 'swift', 'kt', 'scala', 'html', 'css', 'scss', 'sass', 'less',
                      'xml', 'json', 'yaml', 'yml', 'toml', 'ini', 'sql', 'sh', 'bash', 'zsh',
                      'dockerfile', 'makefile', 'vue', 'svelte', 'graphql', 'gql'];
    if (codeExts.includes(ext)) return 'code';

    // Plain text
    if (['txt', 'log', 'text', 'readme', 'license', 'changelog', 'gitignore'].includes(ext) || mime.startsWith('text/')) return 'code';

    return 'unsupported';
  }

  function loadColumnImagePreview(img, url, loading, name) {
    img.onload = () => {
      loading?.classList.add('hidden');
      img.classList.remove('hidden');
    };
    img.onerror = () => {
      loading?.classList.add('hidden');
      showBigPreviewPlaceholder(name);
    };
    img.src = url;
  }

  // Generate PDF thumbnail as data URL
  async function generatePdfThumbnail(fileId, maxSize = 200) {
    try {
      const pdfjsLib = await window.loadPdfJs();
      const url = '/api/v1/content/' + fileId;
      const loadingTask = pdfjsLib.getDocument(url);
      const pdf = await loadingTask.promise;
      const page = await pdf.getPage(1);

      // Calculate scale to fit within maxSize
      const originalViewport = page.getViewport({ scale: 1 });
      const scale = Math.min(maxSize / originalViewport.width, maxSize / originalViewport.height);
      const viewport = page.getViewport({ scale });

      // Create offscreen canvas
      const canvas = document.createElement('canvas');
      canvas.width = viewport.width;
      canvas.height = viewport.height;
      const ctx = canvas.getContext('2d');

      await page.render({ canvasContext: ctx, viewport }).promise;

      return canvas.toDataURL('image/png');
    } catch (err) {
      console.warn('PDF thumbnail generation failed:', err);
      return null;
    }
  }

  async function loadColumnPdfPreview(url) {
    const container = document.getElementById('column-big-preview-pdf');
    const canvas = document.getElementById('column-pdf-canvas');
    const pageEl = document.getElementById('column-pdf-page');
    const totalEl = document.getElementById('column-pdf-total');
    const prevBtn = document.getElementById('column-pdf-prev');
    const nextBtn = document.getElementById('column-pdf-next');
    const zoomEl = document.getElementById('column-pdf-zoom');
    const zoomInBtn = document.getElementById('column-pdf-zoom-in');
    const zoomOutBtn = document.getElementById('column-pdf-zoom-out');

    if (!container || !canvas) return;

    // Load PDF.js
    const pdfjsLib = await window.loadPdfJs();
    const loadingTask = pdfjsLib.getDocument(url);
    const pdf = await loadingTask.promise;

    columnPdfState.doc = pdf;
    columnPdfState.page = 1;
    columnPdfState.scale = 1.0;

    totalEl.textContent = pdf.numPages;

    // Setup controls
    const renderPage = async (pageNum) => {
      const page = await pdf.getPage(pageNum);
      const viewport = page.getViewport({ scale: columnPdfState.scale });

      canvas.width = viewport.width;
      canvas.height = viewport.height;

      const ctx = canvas.getContext('2d');
      await page.render({ canvasContext: ctx, viewport }).promise;

      pageEl.textContent = pageNum;
      prevBtn.disabled = pageNum <= 1;
      nextBtn.disabled = pageNum >= pdf.numPages;
      zoomEl.textContent = Math.round(columnPdfState.scale * 100) + '%';
    };

    // Initial render
    await renderPage(1);
    container.classList.remove('hidden');

    // Event handlers (remove old ones first)
    prevBtn.onclick = async () => {
      if (columnPdfState.page > 1) {
        columnPdfState.page--;
        await renderPage(columnPdfState.page);
      }
    };
    nextBtn.onclick = async () => {
      if (columnPdfState.page < pdf.numPages) {
        columnPdfState.page++;
        await renderPage(columnPdfState.page);
      }
    };
    zoomInBtn.onclick = async () => {
      columnPdfState.scale *= 1.25;
      await renderPage(columnPdfState.page);
    };
    zoomOutBtn.onclick = async () => {
      columnPdfState.scale *= 0.8;
      await renderPage(columnPdfState.page);
    };
  }

  async function loadColumnMarkdownPreview(url, filename) {
    const container = document.getElementById('column-big-preview-markdown');
    const content = document.getElementById('column-markdown-content');

    if (!container || !content) return;

    // Fetch content
    const response = await fetch(url);
    const text = await response.text();

    // Load marked.js and render
    const marked = await window.loadMarked();
    content.innerHTML = marked.parse(text);

    // Highlight code blocks if any
    try {
      const hljs = await window.loadHighlightJs();
      content.querySelectorAll('pre code').forEach(block => {
        hljs.highlightElement(block);
      });
    } catch (e) {
      // Ignore highlighting errors
    }

    container.classList.remove('hidden');
  }

  async function loadColumnCodePreview(url, filename) {
    const container = document.getElementById('column-big-preview-code');
    const content = document.getElementById('column-code-content');
    const filenameEl = document.getElementById('column-code-filename');
    const langEl = document.getElementById('column-code-lang');

    if (!container || !content) return;

    // Fetch content
    const response = await fetch(url);
    const text = await response.text();

    // Get language
    const lang = getLanguage(filename);
    if (filenameEl) filenameEl.textContent = filename;
    if (langEl) langEl.textContent = lang;

    // Set content
    content.textContent = text;
    content.className = `hljs language-${lang}`;

    // Load highlight.js and highlight
    try {
      const hljs = await window.loadHighlightJs();
      hljs.highlightElement(content);
    } catch (e) {
      console.warn('Code highlighting failed:', e);
    }

    container.classList.remove('hidden');
  }

  function showBigPreviewPlaceholder(name) {
    const placeholder = document.getElementById('column-big-preview-placeholder');
    const filename = document.getElementById('column-big-preview-filename');
    if (filename) filename.textContent = name;
    placeholder?.classList.remove('hidden');
  }

  function hideBigPreview() {
    const bigPreview = document.getElementById('column-big-preview');
    const video = document.getElementById('column-big-preview-video');
    const audio = document.getElementById('column-big-preview-audio');

    if (!bigPreview) return;

    // Stop any playing media
    if (video) {
      video.pause();
      video.src = '';
    }
    if (audio) {
      audio.pause();
      audio.src = '';
    }

    // Reset PDF state
    columnPdfState.doc = null;

    bigPreview.classList.add('hidden');
  }

  async function loadColumnContents(folderId, folderName) {
    const columnsContainer = document.getElementById('columns-container');

    try {
      const response = await fetch('/api/v1/folder-children/' + folderId);
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

    // Hide all optional rows
    const optionalRows = [
      'column-preview-dimensions', 'column-preview-camera', 'column-preview-lens',
      'column-preview-exposure', 'column-preview-location', 'column-preview-duration',
      'column-preview-codec', 'column-preview-framerate', 'column-preview-artist',
      'column-preview-album', 'column-preview-bitrate', 'column-preview-pages', 'column-preview-author'
    ];
    optionalRows.forEach(id => document.getElementById(id)?.classList.add('hidden'));

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
      } else if (item.mime === 'application/pdf') {
        // Generate PDF thumbnail using PDF.js
        previewImg.classList.add('hidden');
        previewIcon.classList.remove('hidden');
        previewIcon.innerHTML = `<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/><path d="M10 12h4"/><path d="M10 16h4"/>`;
        // Async PDF thumbnail generation
        generatePdfThumbnail(item.id).then(dataUrl => {
          if (dataUrl && columnState.selectedItem?.id === item.id) {
            previewImg.src = dataUrl;
            previewImg.classList.remove('hidden');
            previewIcon.classList.add('hidden');
          }
        }).catch(() => {});
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

          // Display modified date
          if (meta.modified_at) {
            previewDate.textContent = formatDate(meta.modified_at);
          }

          // Image metadata
          if (meta.image) {
            const img = meta.image;
            if (img.width && img.height) {
              dimsRow?.classList.remove('hidden');
              dimsValue.textContent = `${img.width} × ${img.height}`;
            }
            if (img.make || img.model) {
              cameraRow?.classList.remove('hidden');
              cameraValue.textContent = [img.make, img.model].filter(Boolean).join(' ');
            }
            if (img.lens_model) {
              document.getElementById('column-preview-lens')?.classList.remove('hidden');
              document.getElementById('column-preview-lens-model').textContent = img.lens_model;
            }
            // Exposure summary
            if (img.f_number || img.exposure_time || img.iso) {
              const expParts = [];
              if (img.f_number) expParts.push(`f/${img.f_number}`);
              if (img.exposure_time) expParts.push(img.exposure_time);
              if (img.iso) expParts.push(`ISO ${img.iso}`);
              if (expParts.length) {
                document.getElementById('column-preview-exposure')?.classList.remove('hidden');
                document.getElementById('column-preview-exposure-info').textContent = expParts.join(' · ');
              }
            }
            // Location
            if (img.gps_latitude && img.gps_longitude) {
              const locRow = document.getElementById('column-preview-location');
              const locLink = document.getElementById('column-preview-location-link');
              if (locRow && locLink) {
                locRow.classList.remove('hidden');
                locLink.href = `https://www.google.com/maps?q=${img.gps_latitude},${img.gps_longitude}`;
                locLink.textContent = `${img.gps_latitude.toFixed(4)}, ${img.gps_longitude.toFixed(4)}`;
              }
            }
          }

          // Video metadata
          if (meta.video) {
            const vid = meta.video;
            if (vid.width && vid.height) {
              dimsRow?.classList.remove('hidden');
              dimsValue.textContent = `${vid.width} × ${vid.height}`;
            }
            if (vid.duration || vid.duration_str) {
              durationRow?.classList.remove('hidden');
              durationValue.textContent = vid.duration_str || formatDuration(vid.duration);
            }
            if (vid.video_codec) {
              document.getElementById('column-preview-codec')?.classList.remove('hidden');
              document.getElementById('column-preview-codec-info').textContent = vid.video_codec.toUpperCase();
            }
            if (vid.frame_rate) {
              document.getElementById('column-preview-framerate')?.classList.remove('hidden');
              document.getElementById('column-preview-fps').textContent = `${vid.frame_rate.toFixed(2)} fps`;
            }
          }

          // Audio metadata
          if (meta.audio) {
            const aud = meta.audio;
            if (aud.duration || aud.duration_str) {
              durationRow?.classList.remove('hidden');
              durationValue.textContent = aud.duration_str || formatDuration(aud.duration);
            }
            if (aud.artist) {
              document.getElementById('column-preview-artist')?.classList.remove('hidden');
              document.getElementById('column-preview-artist-name').textContent = aud.artist;
            }
            if (aud.album) {
              document.getElementById('column-preview-album')?.classList.remove('hidden');
              document.getElementById('column-preview-album-name').textContent = aud.album;
            }
            if (aud.bitrate) {
              document.getElementById('column-preview-bitrate')?.classList.remove('hidden');
              document.getElementById('column-preview-bitrate-info').textContent = `${Math.round(aud.bitrate)} kbps`;
            }
          }

          // Document metadata
          if (meta.document) {
            const doc = meta.document;
            if (doc.page_count) {
              document.getElementById('column-preview-pages')?.classList.remove('hidden');
              document.getElementById('column-preview-page-count').textContent = doc.page_count;
            }
            if (doc.author) {
              document.getElementById('column-preview-author')?.classList.remove('hidden');
              document.getElementById('column-preview-author-name').textContent = doc.author;
            }
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
      // Single click to select and preview
      thumb.addEventListener('click', () => {
        selectGalleryItem(index);
      });

      // Double click to open full preview overlay or navigate to folder
      thumb.addEventListener('dblclick', () => {
        const item = galleryState.items[index];
        if (item.type === 'folder') {
          const viewMode = getViewPreference();
          const { sortBy, sortOrder } = getSortPreference();
          const url = new URL('/files/' + item.id, window.location.origin);
          url.searchParams.set('view', viewMode);
          url.searchParams.set('sort', sortBy);
          url.searchParams.set('order', sortOrder);
          window.location.href = url.toString();
        } else {
          openPreview({ id: item.id, name: item.name, mime: item.mime });
        }
      });
    });

    // Setup navigation buttons
    document.getElementById('gallery-prev')?.addEventListener('click', () => navigateGallery(-1));
    document.getElementById('gallery-next')?.addEventListener('click', () => navigateGallery(1));

    // Keyboard navigation
    galleryView.addEventListener('keydown', handleGalleryKeydown);
    galleryView.setAttribute('tabindex', '0');

    // Auto-select first item (file or folder)
    if (galleryState.items.length > 0) {
      // Prefer first file, fallback to first item
      const firstFileIndex = galleryState.items.findIndex(item => item.type === 'file');
      selectGalleryItem(firstFileIndex >= 0 ? firstFileIndex : 0);
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

      // Double click on main preview to open full preview overlay
      mainArea.addEventListener('dblclick', () => {
        if (galleryState.currentItem && galleryState.currentItem.type === 'file') {
          openPreview(galleryState.currentItem);
        }
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
        showGalleryPlaceholder('Failed to load image', true);
      };
      img.src = contentUrl;
      img.classList.remove('hidden');
    } else if (mime.startsWith('video/')) {
      loading.classList.add('hidden');
      video.onerror = () => {
        video.classList.add('hidden');
        showGalleryPlaceholder('Video format not supported', true);
      };
      video.src = contentUrl;
      video.classList.remove('hidden');
      video.play().catch(() => {});
    } else if (mime.startsWith('audio/')) {
      loading.classList.add('hidden');
      audio.onerror = () => {
        audio.classList.add('hidden');
        showGalleryPlaceholder('Audio format not supported', true);
      };
      audio.src = contentUrl;
      audio.classList.remove('hidden');
      audio.play().catch(() => {});
    } else {
      loading.classList.add('hidden');
      // Show preview not available with Open Preview button for documents, code, etc.
      showGalleryPlaceholder('Double-click or press Enter to preview', true);
    }
  }

  function showGalleryPlaceholder(message, showOpenButton = false) {
    const placeholder = document.getElementById('gallery-preview-placeholder');
    const img = document.getElementById('gallery-preview-img');
    const video = document.getElementById('gallery-preview-video');
    const audio = document.getElementById('gallery-preview-audio');

    img.classList.add('hidden');
    video.classList.add('hidden');
    audio.classList.add('hidden');

    // Update message
    const msgSpan = placeholder.querySelector('span');
    if (msgSpan) {
      msgSpan.textContent = message;
    }

    // Show or create open button for non-previewable files
    let openBtn = placeholder.querySelector('.gallery-open-btn');
    if (showOpenButton && galleryState.currentItem?.type === 'file') {
      if (!openBtn) {
        openBtn = document.createElement('button');
        openBtn.className = 'gallery-open-btn mt-4 px-4 py-2 bg-white/20 hover:bg-white/30 rounded-lg text-white transition-colors';
        openBtn.textContent = 'Open Preview';
        openBtn.addEventListener('click', (e) => {
          e.stopPropagation();
          if (galleryState.currentItem) {
            openPreview(galleryState.currentItem);
          }
        });
        placeholder.appendChild(openBtn);
      }
      openBtn.classList.remove('hidden');
    } else if (openBtn) {
      openBtn.classList.add('hidden');
    }

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
    const allSections = [
      'gallery-media-info', 'gallery-camera-info', 'gallery-audio-info', 'gallery-location-info',
      'gallery-exposure-info', 'gallery-format-info', 'gallery-audio-tech-info', 'gallery-track-info',
      'gallery-video-info', 'gallery-video-audio-info', 'gallery-subtitles-info', 'gallery-document-info'
    ];
    allSections.forEach(id => document.getElementById(id)?.classList.add('hidden'));

    // Hide all optional rows
    document.querySelectorAll('[id^="gallery-"][id$="-row"]').forEach(row => {
      if (!row.id.includes('info-name') && !row.id.includes('info-type')) {
        row.classList.add('hidden');
      }
    });

    // Hide map link
    document.getElementById('gallery-map-link')?.classList.add('hidden');

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

      // Display dates
      if (meta.created_at) {
        document.getElementById('gallery-info-created').textContent = formatDate(meta.created_at);
      }
      if (meta.modified_at) {
        document.getElementById('gallery-info-modified').textContent = formatDate(meta.modified_at);
      }

      // Image metadata
      if (meta.image) {
        const img = meta.image;
        const mediaInfo = document.getElementById('gallery-media-info');
        mediaInfo?.classList.remove('hidden');

        if (img.width && img.height) {
          showRow('gallery-dims-row', 'gallery-info-dims', `${img.width} × ${img.height}`);
        }

        // Camera info section
        if (img.make || img.model || img.f_number || img.exposure_time || img.iso) {
          document.getElementById('gallery-camera-info')?.classList.remove('hidden');

          if (img.make || img.model) {
            showRow('gallery-camera-row', 'gallery-info-camera', [img.make, img.model].filter(Boolean).join(' '));
          }
          if (img.f_number) {
            showRow('gallery-aperture-row', 'gallery-info-aperture', `f/${img.f_number}`);
          }
          if (img.exposure_time) {
            showRow('gallery-exposure-row', 'gallery-info-exposure', img.exposure_time);
          }
          if (img.iso) {
            showRow('gallery-iso-row', 'gallery-info-iso', img.iso);
          }
          if (img.focal_length) {
            let focalText = `${img.focal_length}mm`;
            if (img.focal_length_35mm) focalText += ` (${img.focal_length_35mm}mm equiv)`;
            showRow('gallery-focal-row', 'gallery-info-focal', focalText);
          }
        }

        // Exposure/additional EXIF section
        if (img.lens_model || img.flash || img.metering_mode || img.exposure_program || img.white_balance || img.software || img.date_time_original) {
          document.getElementById('gallery-exposure-info')?.classList.remove('hidden');

          if (img.lens_model) showRow('gallery-lens-row', 'gallery-info-lens', img.lens_model);
          if (img.flash) showRow('gallery-flash-row', 'gallery-info-flash', img.flash);
          if (img.metering_mode) showRow('gallery-metering-row', 'gallery-info-metering', img.metering_mode);
          if (img.exposure_program) showRow('gallery-program-row', 'gallery-info-program', img.exposure_program);
          if (img.white_balance) showRow('gallery-wb-row', 'gallery-info-wb', img.white_balance);
          if (img.software) showRow('gallery-software-row', 'gallery-info-software', img.software);
          if (img.date_time_original) showRow('gallery-datetime-row', 'gallery-info-datetime', img.date_time_original);
        }

        // Format info section
        if (img.color_space || img.bit_depth || img.has_alpha !== undefined || img.is_animated) {
          document.getElementById('gallery-format-info')?.classList.remove('hidden');

          if (img.color_space) showRow('gallery-colorspace-row', 'gallery-info-colorspace', img.color_space);
          if (img.bit_depth) showRow('gallery-bitdepth-row', 'gallery-info-bitdepth', `${img.bit_depth}-bit`);
          if (img.has_alpha !== undefined) showRow('gallery-alpha-row', 'gallery-info-alpha', img.has_alpha ? 'Yes' : 'No');
          if (img.is_animated) {
            let animText = 'Yes';
            if (img.frame_count) animText += ` (${img.frame_count} frames)`;
            showRow('gallery-animated-row', 'gallery-info-animated', animText);
          }
        }

        // GPS location
        if (img.gps_latitude && img.gps_longitude) {
          document.getElementById('gallery-location-info')?.classList.remove('hidden');
          document.getElementById('gallery-location-coords').textContent =
            `${img.gps_latitude.toFixed(6)}, ${img.gps_longitude.toFixed(6)}`;

          // Map link
          const mapLink = document.getElementById('gallery-map-link');
          if (mapLink) {
            mapLink.href = `https://www.google.com/maps?q=${img.gps_latitude},${img.gps_longitude}`;
            mapLink.classList.remove('hidden');
          }

          // Altitude
          if (img.gps_altitude) {
            document.getElementById('gallery-altitude-row')?.classList.remove('hidden');
            document.getElementById('gallery-info-altitude').textContent = `${img.gps_altitude.toFixed(1)}m`;
          }
        }
      }

      // Video metadata
      if (meta.video) {
        const vid = meta.video;
        document.getElementById('gallery-media-info')?.classList.remove('hidden');

        if (vid.width && vid.height) {
          showRow('gallery-dims-row', 'gallery-info-dims', `${vid.width} × ${vid.height}`);
        }
        if (vid.duration || vid.duration_str) {
          showRow('gallery-duration-row', 'gallery-info-duration', vid.duration_str || formatDuration(vid.duration));
        }
        if (vid.video_codec) {
          showRow('gallery-codec-row', 'gallery-info-codec', vid.video_codec.toUpperCase());
        }

        // Video track info section
        if (vid.video_codec || vid.aspect_ratio || vid.frame_rate || vid.video_bitrate || vid.container || vid.hdr_format || vid.bit_depth) {
          document.getElementById('gallery-video-info')?.classList.remove('hidden');

          if (vid.video_codec) showRow('gallery-videocodec-row', 'gallery-info-videocodec', vid.video_codec.toUpperCase());
          if (vid.aspect_ratio) showRow('gallery-aspectratio-row', 'gallery-info-aspectratio', vid.aspect_ratio);
          if (vid.frame_rate) showRow('gallery-framerate-row', 'gallery-info-framerate', `${vid.frame_rate.toFixed(2)} fps`);
          if (vid.video_bitrate) showRow('gallery-videobitrate-row', 'gallery-info-videobitrate', `${Math.round(vid.video_bitrate)} kbps`);
          if (vid.container) showRow('gallery-container-row', 'gallery-info-container', vid.container.toUpperCase());
          if (vid.hdr_format) showRow('gallery-hdr-row', 'gallery-info-hdr', vid.hdr_format);
          if (vid.bit_depth) showRow('gallery-videobitdepth-row', 'gallery-info-videobitdepth', `${vid.bit_depth}-bit`);
        }

        // Video audio track section
        if (vid.has_audio && (vid.audio_codec || vid.audio_bitrate || vid.audio_channels || vid.audio_sample_rate)) {
          document.getElementById('gallery-video-audio-info')?.classList.remove('hidden');

          if (vid.audio_codec) showRow('gallery-va-codec-row', 'gallery-info-va-codec', vid.audio_codec.toUpperCase());
          if (vid.audio_bitrate) showRow('gallery-va-bitrate-row', 'gallery-info-va-bitrate', `${Math.round(vid.audio_bitrate)} kbps`);
          if (vid.audio_channels) showRow('gallery-va-channels-row', 'gallery-info-va-channels', formatChannels(vid.audio_channels));
          if (vid.audio_sample_rate) showRow('gallery-va-samplerate-row', 'gallery-info-va-samplerate', `${vid.audio_sample_rate} Hz`);
        }

        // Subtitles section
        if (vid.subtitle_tracks && vid.subtitle_tracks.length > 0) {
          document.getElementById('gallery-subtitles-info')?.classList.remove('hidden');
          const subList = document.getElementById('gallery-subtitles-list');
          if (subList) {
            subList.innerHTML = vid.subtitle_tracks.map(t =>
              `<div class="flex justify-between"><span>${t.language || 'Unknown'}</span><span class="text-zinc-500">${t.format || ''}</span></div>`
            ).join('');
          }
        }
      }

      // Audio metadata
      if (meta.audio) {
        const aud = meta.audio;
        document.getElementById('gallery-media-info')?.classList.remove('hidden');
        document.getElementById('gallery-audio-info')?.classList.remove('hidden');

        if (aud.duration || aud.duration_str) {
          showRow('gallery-duration-row', 'gallery-info-duration', aud.duration_str || formatDuration(aud.duration));
        }
        if (aud.artist) showRow('gallery-artist-row', 'gallery-info-artist', aud.artist);
        if (aud.album) showRow('gallery-album-row', 'gallery-info-album', aud.album);
        if (aud.genre) showRow('gallery-genre-row', 'gallery-info-genre', aud.genre);
        if (aud.year) showRow('gallery-year-row', 'gallery-info-year', aud.year);
        if (aud.bitrate) showRow('gallery-bitrate-row', 'gallery-info-bitrate', `${Math.round(aud.bitrate)} kbps`);

        // Audio technical info section
        if (aud.sample_rate || aud.channels || aud.bit_depth || aud.codec) {
          document.getElementById('gallery-audio-tech-info')?.classList.remove('hidden');

          if (aud.sample_rate) showRow('gallery-samplerate-row', 'gallery-info-samplerate', `${aud.sample_rate} Hz`);
          if (aud.channels) showRow('gallery-channels-row', 'gallery-info-channels', formatChannels(aud.channels));
          if (aud.bit_depth) showRow('gallery-audiobitdepth-row', 'gallery-info-audiobitdepth', `${aud.bit_depth}-bit`);
          if (aud.codec) showRow('gallery-audiocodec-row', 'gallery-info-audiocodec', aud.codec.toUpperCase());
        }

        // Track info section
        if (aud.title || aud.album_artist || aud.composer || aud.track_number || aud.disc_number || aud.bpm || aud.publisher || aud.comment) {
          document.getElementById('gallery-track-info')?.classList.remove('hidden');

          if (aud.title) showRow('gallery-title-row', 'gallery-info-title', aud.title);
          if (aud.album_artist) showRow('gallery-albumartist-row', 'gallery-info-albumartist', aud.album_artist);
          if (aud.composer) showRow('gallery-composer-row', 'gallery-info-composer', aud.composer);
          if (aud.track_number) {
            let trackText = String(aud.track_number);
            if (aud.track_total) trackText += ` of ${aud.track_total}`;
            showRow('gallery-track-row', 'gallery-info-track', trackText);
          }
          if (aud.disc_number) {
            let discText = String(aud.disc_number);
            if (aud.disc_total) discText += ` of ${aud.disc_total}`;
            showRow('gallery-disc-row', 'gallery-info-disc', discText);
          }
          if (aud.bpm) showRow('gallery-bpm-row', 'gallery-info-bpm', aud.bpm);
          if (aud.publisher) showRow('gallery-publisher-row', 'gallery-info-publisher', aud.publisher);
          if (aud.comment) showRow('gallery-comment-row', 'gallery-info-comment', aud.comment);
        }
      }

      // Document metadata
      if (meta.document) {
        const doc = meta.document;
        document.getElementById('gallery-media-info')?.classList.remove('hidden');

        if (doc.page_count) {
          showRow('gallery-dims-row', 'gallery-info-dims', `${doc.page_count} pages`);
        }

        // Document info section
        if (doc.page_count || doc.author || doc.title || doc.subject || doc.keywords || doc.creator || doc.producer || doc.pdf_version || doc.word_count || doc.is_encrypted !== undefined || doc.created_at || doc.modified_at) {
          document.getElementById('gallery-document-info')?.classList.remove('hidden');

          if (doc.page_count) showRow('gallery-pages-row', 'gallery-info-pages', doc.page_count);
          if (doc.author) showRow('gallery-author-row', 'gallery-info-author', doc.author);
          if (doc.title) showRow('gallery-doctitle-row', 'gallery-info-doctitle', doc.title);
          if (doc.subject) showRow('gallery-subject-row', 'gallery-info-subject', doc.subject);
          if (doc.keywords) showRow('gallery-keywords-row', 'gallery-info-keywords', doc.keywords);
          if (doc.creator) showRow('gallery-creator-row', 'gallery-info-creator', doc.creator);
          if (doc.producer) showRow('gallery-producer-row', 'gallery-info-producer', doc.producer);
          if (doc.pdf_version) showRow('gallery-pdfversion-row', 'gallery-info-pdfversion', doc.pdf_version);
          if (doc.word_count) showRow('gallery-wordcount-row', 'gallery-info-wordcount', doc.word_count.toLocaleString());
          if (doc.is_encrypted !== undefined) showRow('gallery-encrypted-row', 'gallery-info-encrypted', doc.is_encrypted ? 'Yes' : 'No');
          if (doc.created_at) showRow('gallery-doccreated-row', 'gallery-info-doccreated', doc.created_at);
          if (doc.modified_at) showRow('gallery-docmodified-row', 'gallery-info-docmodified', doc.modified_at);
        }
      }

    } catch (err) {
      console.error('Failed to fetch metadata:', err);
    }
  }

  // Helper to show a metadata row
  function showRow(rowId, valueId, value) {
    const row = document.getElementById(rowId);
    const valueEl = document.getElementById(valueId);
    if (row && valueEl) {
      row.classList.remove('hidden');
      valueEl.textContent = value;
    }
  }

  // =====================
  // SIDEBAR (List/Grid Views)
  // =====================
  function setupSidebar() {
    // Detect which view we're in
    const listSidebar = document.getElementById('list-sidebar');
    const gridSidebar = document.getElementById('grid-sidebar');

    if (listSidebar) {
      sidebarState.viewType = 'list';
      setupSidebarControls('list');
    } else if (gridSidebar) {
      sidebarState.viewType = 'grid';
      setupSidebarControls('grid');
    } else {
      return; // No sidebar in this view
    }

    // Load sidebar state from localStorage
    const savedOpen = localStorage.getItem('drive_sidebar_open') === 'true';
    const savedWidth = parseInt(localStorage.getItem('drive_sidebar_width')) || 288;
    sidebarState.width = savedWidth;

    if (savedOpen) {
      toggleSidebar(true);
    }
  }

  function setupSidebarControls(viewType) {
    const prefix = viewType;
    const sidebar = document.getElementById(`${prefix}-sidebar`);
    const toggleBtn = document.getElementById(`${prefix}-sidebar-toggle`);
    const resizeHandle = document.getElementById(`${prefix}-sidebar-resize`);
    const openBtn = document.getElementById(`${prefix}-preview-open`);
    const downloadBtn = document.getElementById(`${prefix}-preview-download`);

    // Toggle button
    toggleBtn?.addEventListener('click', () => toggleSidebar());

    // Keyboard shortcut (I for Info)
    document.addEventListener('keydown', (e) => {
      if (e.key === 'i' && !e.ctrlKey && !e.metaKey && !e.target.matches('input, textarea')) {
        e.preventDefault();
        toggleSidebar();
      }
    });

    // Resize handle
    if (resizeHandle) {
      let isResizing = false;
      let startX, startWidth;

      resizeHandle.addEventListener('mousedown', (e) => {
        isResizing = true;
        startX = e.clientX;
        startWidth = sidebarState.width;
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
      });

      document.addEventListener('mousemove', (e) => {
        if (!isResizing) return;
        const diff = startX - e.clientX;
        const newWidth = Math.max(200, Math.min(500, startWidth + diff));
        sidebarState.width = newWidth;
        sidebar.style.width = `${newWidth}px`;
        localStorage.setItem('drive_sidebar_width', newWidth);
      });

      document.addEventListener('mouseup', () => {
        if (isResizing) {
          isResizing = false;
          document.body.style.cursor = '';
          document.body.style.userSelect = '';
        }
      });
    }

    // Open button
    openBtn?.addEventListener('click', () => {
      if (sidebarState.selectedItem) {
        const { id, type, name, mime } = sidebarState.selectedItem;
        if (type === 'folder') {
          const viewMode = getViewPreference();
          const { sortBy, sortOrder } = getSortPreference();
          const url = new URL('/files/' + id, window.location.origin);
          url.searchParams.set('view', viewMode);
          url.searchParams.set('sort', sortBy);
          url.searchParams.set('order', sortOrder);
          window.location.href = url.toString();
        } else {
          openPreview({ id, name, mime });
        }
      }
    });

    // Download button
    downloadBtn?.addEventListener('click', () => {
      if (sidebarState.selectedItem && sidebarState.selectedItem.type === 'file') {
        const a = document.createElement('a');
        a.href = '/api/v1/content/' + sidebarState.selectedItem.id;
        a.download = sidebarState.selectedItem.name;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
      }
    });

    // Setup click handlers on items
    setupSidebarItemHandlers(viewType);
  }

  function setupSidebarItemHandlers(viewType) {
    const items = document.querySelectorAll('[data-type="file"], [data-type="folder"]');
    items.forEach(item => {
      item.addEventListener('click', (e) => {
        // Don't trigger if clicking on a link or button
        if (e.target.closest('a') || e.target.closest('button')) return;

        const id = item.dataset.id;
        const type = item.dataset.type;
        const name = item.dataset.name;
        const mime = item.dataset.mime || '';
        const size = item.dataset.size || '';

        // Highlight selected item
        items.forEach(i => i.classList.remove('ring-2', 'ring-blue-500', 'bg-blue-50'));
        item.classList.add('ring-2', 'ring-blue-500', 'bg-blue-50');

        // Update sidebar
        sidebarState.selectedItem = { id, type, name, mime, size };
        updateSidebarPreview(viewType, sidebarState.selectedItem);

        // Auto-open sidebar on first selection if closed
        if (!sidebarState.isOpen) {
          toggleSidebar(true);
        }
      });
    });
  }

  function toggleSidebar(forceOpen) {
    const viewType = sidebarState.viewType;
    if (!viewType) return;

    const sidebar = document.getElementById(`${viewType}-sidebar`);
    const toggleBtn = document.getElementById(`${viewType}-sidebar-toggle`);
    if (!sidebar) return;

    const shouldOpen = forceOpen !== undefined ? forceOpen : !sidebarState.isOpen;
    sidebarState.isOpen = shouldOpen;

    if (shouldOpen) {
      sidebar.classList.remove('hidden', 'w-0');
      sidebar.style.width = `${sidebarState.width}px`;
      toggleBtn?.classList.add('bg-blue-600');
    } else {
      sidebar.classList.add('w-0');
      sidebar.style.width = '0px';
      setTimeout(() => {
        if (!sidebarState.isOpen) sidebar.classList.add('hidden');
      }, 200);
      toggleBtn?.classList.remove('bg-blue-600');
    }

    localStorage.setItem('drive_sidebar_open', shouldOpen);
  }

  async function updateSidebarPreview(viewType, item) {
    const prefix = viewType;
    const previewImg = document.getElementById(`${prefix}-preview-img`);
    const previewIcon = document.getElementById(`${prefix}-preview-icon`);
    const previewName = document.getElementById(`${prefix}-preview-name`);
    const previewKind = document.getElementById(`${prefix}-preview-kind`);
    const previewSize = document.getElementById(`${prefix}-preview-size`);
    const previewDate = document.getElementById(`${prefix}-preview-date`);

    previewName.textContent = item.name;

    // Hide all optional rows
    const optionalRows = [`${prefix}-preview-dimensions`, `${prefix}-preview-duration`, `${prefix}-preview-camera`, `${prefix}-preview-artist`, `${prefix}-preview-album`];
    optionalRows.forEach(id => document.getElementById(id)?.classList.add('hidden'));

    if (item.type === 'folder') {
      previewImg?.classList.add('hidden');
      previewIcon?.classList.remove('hidden');
      if (previewIcon) {
        previewIcon.innerHTML = `<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" fill="currentColor" opacity="0.2" stroke="currentColor" stroke-width="1"/>`;
      }
      previewKind.textContent = 'Folder';
      previewSize.textContent = '-';
      previewDate.textContent = '-';
    } else {
      // Check if image
      if (item.mime && item.mime.startsWith('image/')) {
        previewImg.src = '/api/v1/thumbnail/' + item.id;
        previewImg?.classList.remove('hidden');
        previewIcon?.classList.add('hidden');
      } else {
        previewImg?.classList.add('hidden');
        previewIcon?.classList.remove('hidden');
      }

      previewKind.textContent = getMimeDescription(item.mime);
      previewSize.textContent = item.size ? formatSize(parseInt(item.size)) : '-';
      previewDate.textContent = '-';

      // Fetch metadata
      try {
        const response = await fetch('/api/v1/metadata/' + item.id);
        if (response.ok) {
          const meta = await response.json();
          console.debug('[metadata]', item.id, meta);

          if (meta.size) {
            previewSize.textContent = formatSize(meta.size);
          }

          if (meta.modified_at) {
            previewDate.textContent = formatDate(meta.modified_at);
          }

          // Image metadata
          if (meta.image) {
            console.debug('[metadata] image details:', meta.image);
            const img = meta.image;
            if (img.width && img.height) {
              const row = document.getElementById(`${prefix}-preview-dimensions`);
              const value = document.getElementById(`${prefix}-preview-dims`);
              if (row && value) {
                row.classList.remove('hidden');
                value.textContent = `${img.width} × ${img.height}`;
              }
            }
            if (img.make || img.model) {
              const row = document.getElementById(`${prefix}-preview-camera`);
              const value = document.getElementById(`${prefix}-preview-camera-model`);
              if (row && value) {
                row.classList.remove('hidden');
                value.textContent = [img.make, img.model].filter(Boolean).join(' ');
              }
            }
          }

          // Video/Audio metadata
          if (meta.video || meta.audio) {
            const media = meta.video || meta.audio;
            if (media.duration || media.duration_str) {
              const row = document.getElementById(`${prefix}-preview-duration`);
              const value = document.getElementById(`${prefix}-preview-dur`);
              if (row && value) {
                row.classList.remove('hidden');
                value.textContent = media.duration_str || formatDuration(media.duration);
              }
            }
            if (meta.audio?.artist) {
              const row = document.getElementById(`${prefix}-preview-artist`);
              const value = document.getElementById(`${prefix}-preview-artist-name`);
              if (row && value) {
                row.classList.remove('hidden');
                value.textContent = meta.audio.artist;
              }
            }
            if (meta.audio?.album) {
              const row = document.getElementById(`${prefix}-preview-album`);
              const value = document.getElementById(`${prefix}-preview-album-name`);
              if (row && value) {
                row.classList.remove('hidden');
                value.textContent = meta.audio.album;
              }
            }
          }
        }
      } catch (err) {
        console.error('Failed to fetch metadata:', err);
      }
    }
  }

  // Helper to format channel count
  function formatChannels(channels) {
    switch (channels) {
      case 1: return 'Mono';
      case 2: return 'Stereo';
      case 6: return '5.1 Surround';
      case 8: return '7.1 Surround';
      default: return `${channels} channels`;
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
      // Images
      'image/jpeg': 'JPEG Image',
      'image/png': 'PNG Image',
      'image/gif': 'GIF Image',
      'image/webp': 'WebP Image',
      'image/svg+xml': 'SVG Image',
      'image/bmp': 'BMP Image',
      'image/tiff': 'TIFF Image',
      'image/heic': 'HEIC Image',
      'image/heif': 'HEIF Image',
      // Video
      'video/mp4': 'MP4 Video',
      'video/webm': 'WebM Video',
      'video/quicktime': 'QuickTime Video',
      'video/x-msvideo': 'AVI Video',
      'video/x-matroska': 'MKV Video',
      'video/x-ms-wmv': 'WMV Video',
      'video/x-flv': 'FLV Video',
      'video/x-m4v': 'M4V Video',
      'video/3gpp': '3GP Video',
      'video/ogg': 'OGG Video',
      // Audio
      'audio/mpeg': 'MP3 Audio',
      'audio/mp3': 'MP3 Audio',
      'audio/wav': 'WAV Audio',
      'audio/flac': 'FLAC Audio',
      'audio/ogg': 'OGG Audio',
      'audio/aac': 'AAC Audio',
      'audio/mp4': 'M4A Audio',
      'audio/x-ms-wma': 'WMA Audio',
      'audio/aiff': 'AIFF Audio',
      'audio/opus': 'Opus Audio',
      'audio/midi': 'MIDI Audio',
      // Documents
      'application/pdf': 'PDF Document',
      'application/zip': 'ZIP Archive',
      'application/msword': 'Word Document',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'Word Document',
      'application/vnd.ms-excel': 'Excel Spreadsheet',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'Excel Spreadsheet',
      'application/vnd.ms-powerpoint': 'PowerPoint Presentation',
      'application/vnd.openxmlformats-officedocument.presentationml.presentation': 'PowerPoint Presentation',
      // Text
      'text/plain': 'Text File',
      'text/html': 'HTML Document',
      'text/css': 'CSS Stylesheet',
      'text/javascript': 'JavaScript File',
      'text/typescript': 'TypeScript File',
      'text/markdown': 'Markdown Document',
      'text/x-go': 'Go Source',
      'text/x-python': 'Python Script',
      'text/x-rust': 'Rust Source',
      'text/x-java': 'Java Source',
      'text/x-c': 'C Source',
      'text/x-c++': 'C++ Source',
      'text/x-ruby': 'Ruby Script',
      'text/x-php': 'PHP Script',
      'text/x-swift': 'Swift Source',
      'text/x-kotlin': 'Kotlin Source',
      'text/yaml': 'YAML File',
      'text/toml': 'TOML File',
      'text/x-sql': 'SQL File',
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

  // Helper: Format date from ISO string
  function formatDate(isoString) {
    if (!isoString) return '-';
    try {
      const date = new Date(isoString);
      if (isNaN(date.getTime())) return '-';

      const now = new Date();
      const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      const yesterday = new Date(today.getTime() - 86400000);
      const dateOnly = new Date(date.getFullYear(), date.getMonth(), date.getDate());

      const timeStr = date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });

      if (dateOnly.getTime() === today.getTime()) {
        return `Today at ${timeStr}`;
      } else if (dateOnly.getTime() === yesterday.getTime()) {
        return `Yesterday at ${timeStr}`;
      } else if (now.getTime() - date.getTime() < 7 * 86400000) {
        return date.toLocaleDateString(undefined, { weekday: 'short' }) + ` at ${timeStr}`;
      } else {
        return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
      }
    } catch {
      return '-';
    }
  }
})();
