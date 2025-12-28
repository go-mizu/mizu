// GitHome - Client-side JavaScript

(function() {
  'use strict';

  // Initialize when DOM is ready
  document.addEventListener('DOMContentLoaded', function() {
    initDropdowns();
    initForms();
    initStarButtons();
    initTabs();
    initModals();
    document.body.classList.add('ready');
  });

  // Dropdown menus
  function initDropdowns() {
    document.addEventListener('click', function(e) {
      // Toggle dropdown
      var trigger = e.target.closest('.dropdown-trigger');
      if (trigger) {
        e.preventDefault();
        var dropdown = trigger.closest('.dropdown');
        var menu = dropdown.querySelector('.dropdown-menu');

        // Close other dropdowns
        document.querySelectorAll('.dropdown-menu').forEach(function(m) {
          if (m !== menu) m.classList.add('hidden');
        });

        menu.classList.toggle('hidden');
        return;
      }

      // Close dropdown when clicking outside
      if (!e.target.closest('.dropdown')) {
        document.querySelectorAll('.dropdown-menu').forEach(function(m) {
          m.classList.add('hidden');
        });
      }
    });
  }

  // Form handling
  function initForms() {
    // Auth forms
    var authForms = document.querySelectorAll('[data-auth]');
    authForms.forEach(function(form) {
      form.addEventListener('submit', function(e) {
        e.preventDefault();
        var action = form.getAttribute('data-auth');
        handleAuthForm(form, action);
      });
    });

    // API forms
    var apiForms = document.querySelectorAll('[data-api]');
    apiForms.forEach(function(form) {
      form.addEventListener('submit', function(e) {
        e.preventDefault();
        handleApiForm(form);
      });
    });
  }

  function handleAuthForm(form, action) {
    var url = '/api/v1/auth/' + action;
    var formData = new FormData(form);
    var data = {};
    formData.forEach(function(value, key) {
      data[key] = value;
    });

    var submitBtn = form.querySelector('[type="submit"]');
    var originalText = submitBtn.textContent;
    submitBtn.disabled = true;
    submitBtn.textContent = 'Loading...';

    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
      credentials: 'same-origin'
    })
    .then(function(response) {
      return response.json().then(function(data) {
        return { ok: response.ok, data: data };
      });
    })
    .then(function(result) {
      if (result.ok) {
        // Redirect on success
        window.location.href = '/';
      } else {
        showFormError(form, result.data.error || 'An error occurred');
      }
    })
    .catch(function(error) {
      showFormError(form, 'Network error. Please try again.');
    })
    .finally(function() {
      submitBtn.disabled = false;
      submitBtn.textContent = originalText;
    });
  }

  function handleApiForm(form) {
    var url = form.getAttribute('data-api');
    var method = form.getAttribute('data-method') || 'POST';
    var redirect = form.getAttribute('data-redirect');

    var formData = new FormData(form);
    var data = {};
    formData.forEach(function(value, key) {
      data[key] = value;
    });

    var submitBtn = form.querySelector('[type="submit"]');
    var originalText = submitBtn.textContent;
    submitBtn.disabled = true;
    submitBtn.textContent = 'Loading...';

    fetch(url, {
      method: method,
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
      credentials: 'same-origin'
    })
    .then(function(response) {
      return response.json().then(function(data) {
        return { ok: response.ok, status: response.status, data: data };
      });
    })
    .then(function(result) {
      if (result.ok) {
        if (redirect) {
          window.location.href = redirect;
        } else if (result.data.redirect) {
          window.location.href = result.data.redirect;
        } else {
          // Reload current page
          window.location.reload();
        }
      } else {
        showFormError(form, result.data.error || 'An error occurred');
      }
    })
    .catch(function(error) {
      showFormError(form, 'Network error. Please try again.');
    })
    .finally(function() {
      submitBtn.disabled = false;
      submitBtn.textContent = originalText;
    });
  }

  function showFormError(form, message) {
    var errorEl = form.querySelector('.form-error, .flash-error');
    if (!errorEl) {
      errorEl = document.createElement('div');
      errorEl.className = 'flash flash-error mb-3';
      form.insertBefore(errorEl, form.firstChild);
    }
    errorEl.textContent = message;
    errorEl.classList.remove('d-none');
  }

  // Star/Unstar buttons
  function initStarButtons() {
    document.addEventListener('click', function(e) {
      var starBtn = e.target.closest('[data-star]');
      if (!starBtn) return;

      e.preventDefault();
      var owner = starBtn.getAttribute('data-owner');
      var repo = starBtn.getAttribute('data-repo');
      var isStarred = starBtn.getAttribute('data-starred') === 'true';

      var method = isStarred ? 'DELETE' : 'PUT';
      var url = '/api/v1/user/starred/' + owner + '/' + repo;

      fetch(url, {
        method: method,
        credentials: 'same-origin'
      })
      .then(function(response) {
        if (response.ok || response.status === 204) {
          // Toggle state
          starBtn.setAttribute('data-starred', (!isStarred).toString());

          // Update button text
          var text = starBtn.querySelector('.star-text');
          if (text) {
            text.textContent = isStarred ? 'Star' : 'Starred';
          }

          // Update count
          var count = starBtn.querySelector('.star-count');
          if (count) {
            var currentCount = parseInt(count.textContent) || 0;
            count.textContent = isStarred ? currentCount - 1 : currentCount + 1;
          }
        }
      })
      .catch(function(error) {
        console.error('Failed to toggle star:', error);
      });
    });
  }

  // Tab switching
  function initTabs() {
    document.addEventListener('click', function(e) {
      var tab = e.target.closest('[data-tab]');
      if (!tab) return;

      e.preventDefault();
      var tabGroup = tab.closest('[data-tab-group]');
      var targetId = tab.getAttribute('data-tab');

      // Update active tab
      tabGroup.querySelectorAll('[data-tab]').forEach(function(t) {
        t.classList.remove('selected');
      });
      tab.classList.add('selected');

      // Show target panel
      var panels = document.querySelectorAll('[data-tab-panel]');
      panels.forEach(function(panel) {
        if (panel.getAttribute('data-tab-panel') === targetId) {
          panel.classList.remove('d-none');
        } else {
          panel.classList.add('d-none');
        }
      });
    });
  }

  // Modal dialogs
  function initModals() {
    // Open modal
    document.addEventListener('click', function(e) {
      var trigger = e.target.closest('[data-modal]');
      if (!trigger) return;

      e.preventDefault();
      var modalId = trigger.getAttribute('data-modal');
      var modal = document.getElementById(modalId);
      if (modal) {
        modal.classList.remove('d-none');
        document.body.style.overflow = 'hidden';

        // Focus first input
        var firstInput = modal.querySelector('input, textarea');
        if (firstInput) {
          firstInput.focus();
        }
      }
    });

    // Close modal
    document.addEventListener('click', function(e) {
      var closeBtn = e.target.closest('[data-close]');
      if (closeBtn) {
        var modalId = closeBtn.getAttribute('data-close');
        closeModal(modalId);
        return;
      }

      // Click on backdrop
      if (e.target.classList.contains('modal-backdrop')) {
        var modal = e.target.closest('.modal');
        if (modal) {
          modal.classList.add('d-none');
          document.body.style.overflow = '';
        }
      }
    });

    // Close with Escape
    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') {
        var openModal = document.querySelector('.modal:not(.d-none)');
        if (openModal) {
          openModal.classList.add('d-none');
          document.body.style.overflow = '';
        }
      }
    });
  }

  function closeModal(modalId) {
    var modal = document.getElementById(modalId);
    if (modal) {
      modal.classList.add('d-none');
      document.body.style.overflow = '';
    }
  }

  // Logout handler
  document.addEventListener('click', function(e) {
    var logoutBtn = e.target.closest('[data-action="logout"]');
    if (!logoutBtn) return;

    e.preventDefault();
    fetch('/api/v1/auth/logout', {
      method: 'POST',
      credentials: 'same-origin'
    })
    .then(function() {
      window.location.href = '/';
    })
    .catch(function(error) {
      console.error('Logout failed:', error);
      window.location.href = '/';
    });
  });

  // Delete confirmation
  document.addEventListener('click', function(e) {
    var deleteBtn = e.target.closest('[data-confirm]');
    if (!deleteBtn) return;

    e.preventDefault();
    var message = deleteBtn.getAttribute('data-confirm');

    if (confirm(message)) {
      var url = deleteBtn.getAttribute('data-delete-url');
      if (url) {
        fetch(url, {
          method: 'DELETE',
          credentials: 'same-origin'
        })
        .then(function(response) {
          if (response.ok) {
            var redirect = deleteBtn.getAttribute('data-redirect') || '/';
            window.location.href = redirect;
          } else {
            alert('Failed to delete. Please try again.');
          }
        })
        .catch(function(error) {
          alert('Network error. Please try again.');
        });
      }
    }
  });

})();
