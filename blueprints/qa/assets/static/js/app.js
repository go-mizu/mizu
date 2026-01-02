/**
 * QA Overflow - Interactive Features
 * Based on StackOverflow UX patterns
 */

(function() {
  'use strict';

  // ============================================
  // Theme Toggle (Dark Mode)
  // ============================================
  const ThemeManager = {
    STORAGE_KEY: 'theme',
    DARK: 'dark',
    LIGHT: 'light',

    init() {
      this.bindToggle();
      this.updateIcons();
    },

    getTheme() {
      return localStorage.getItem(this.STORAGE_KEY) ||
        (window.matchMedia('(prefers-color-scheme: dark)').matches ? this.DARK : this.LIGHT);
    },

    setTheme(theme) {
      localStorage.setItem(this.STORAGE_KEY, theme);
      if (theme === this.DARK) {
        document.documentElement.setAttribute('data-theme', 'dark');
      } else {
        document.documentElement.removeAttribute('data-theme');
      }
      this.updateIcons();
    },

    toggle() {
      const current = this.getTheme();
      this.setTheme(current === this.DARK ? this.LIGHT : this.DARK);
    },

    updateIcons() {
      const isDark = document.documentElement.hasAttribute('data-theme');
      document.querySelectorAll('.theme-toggle').forEach(btn => {
        const sunIcon = btn.querySelector('.icon-sun');
        const moonIcon = btn.querySelector('.icon-moon');
        if (sunIcon && moonIcon) {
          sunIcon.style.display = isDark ? 'none' : 'block';
          moonIcon.style.display = isDark ? 'block' : 'none';
        }
      });
    },

    bindToggle() {
      document.querySelectorAll('[data-action="toggle-theme"]').forEach(btn => {
        btn.addEventListener('click', () => this.toggle());
      });
    }
  };

  // ============================================
  // Mobile Navigation
  // ============================================
  const MobileNav = {
    init() {
      this.nav = document.getElementById('left-nav');
      this.bindToggle();
      this.bindClickOutside();
    },

    toggle() {
      if (!this.nav) return;
      const isOpen = this.nav.classList.toggle('is-open');
      document.querySelectorAll('.mobile-nav-toggle').forEach(btn => {
        btn.setAttribute('aria-expanded', isOpen.toString());
      });
    },

    close() {
      if (!this.nav) return;
      this.nav.classList.remove('is-open');
      document.querySelectorAll('.mobile-nav-toggle').forEach(btn => {
        btn.setAttribute('aria-expanded', 'false');
      });
    },

    bindToggle() {
      document.querySelectorAll('[data-action="toggle-nav"]').forEach(btn => {
        btn.addEventListener('click', () => this.toggle());
      });
    },

    bindClickOutside() {
      document.addEventListener('click', (e) => {
        if (!this.nav) return;
        const toggle = e.target.closest('[data-action="toggle-nav"]');
        const nav = e.target.closest('#left-nav');
        if (!toggle && !nav && this.nav.classList.contains('is-open')) {
          this.close();
        }
      });
    }
  };

  // ============================================
  // Voting System
  // ============================================
  const VoteManager = {
    init() {
      this.bindVoteButtons();
    },

    bindVoteButtons() {
      document.querySelectorAll('.vote-btn').forEach(btn => {
        btn.addEventListener('click', (e) => this.handleVote(e));
      });
    },

    async handleVote(e) {
      const btn = e.currentTarget;
      const voteBox = btn.closest('.vote-box');
      if (!voteBox) return;

      const isUpvote = btn.getAttribute('aria-label')?.includes('up');
      const scoreEl = voteBox.querySelector('.score');
      if (!scoreEl) return;

      // Get post info from data attributes or URL
      const postLayout = btn.closest('.post-layout');
      const postType = postLayout?.classList.contains('answer-post') ? 'answer' : 'question';

      // Toggle active state
      const wasActive = btn.classList.contains('is-active');

      // Remove active from sibling vote buttons
      voteBox.querySelectorAll('.vote-btn').forEach(b => b.classList.remove('is-active'));

      if (!wasActive) {
        btn.classList.add('is-active');
      }

      // Update score visually (optimistic update)
      const currentScore = parseInt(scoreEl.textContent.replace(/,/g, ''), 10) || 0;
      let newScore = currentScore;

      if (wasActive) {
        newScore = isUpvote ? currentScore - 1 : currentScore + 1;
      } else {
        newScore = isUpvote ? currentScore + 1 : currentScore - 1;
      }

      scoreEl.textContent = newScore.toLocaleString();

      // Show visual feedback
      btn.style.transform = 'scale(1.1)';
      setTimeout(() => {
        btn.style.transform = '';
      }, 150);

      // TODO: Send vote to server
      // try {
      //   const response = await fetch('/api/votes', {
      //     method: 'POST',
      //     headers: { 'Content-Type': 'application/json' },
      //     body: JSON.stringify({ postType, postId, value: isUpvote ? 1 : -1 })
      //   });
      //   if (!response.ok) throw new Error('Vote failed');
      // } catch (error) {
      //   // Revert on error
      //   scoreEl.textContent = currentScore.toLocaleString();
      //   Toast.show('Failed to record vote', 'error');
      // }
    }
  };

  // ============================================
  // Bookmark/Save Feature
  // ============================================
  const BookmarkManager = {
    init() {
      this.bindBookmarkButtons();
    },

    bindBookmarkButtons() {
      document.querySelectorAll('.bookmark-btn').forEach(btn => {
        btn.addEventListener('click', (e) => this.handleBookmark(e));
      });
    },

    handleBookmark(e) {
      const btn = e.currentTarget;
      const wasActive = btn.classList.contains('is-active');
      btn.classList.toggle('is-active');

      // Update count if present
      const countEl = btn.nextElementSibling;
      if (countEl && countEl.classList.contains('favorite-count')) {
        const current = parseInt(countEl.textContent, 10) || 0;
        countEl.textContent = wasActive ? current - 1 : current + 1;
      }

      // Show feedback
      Toast.show(wasActive ? 'Bookmark removed' : 'Question saved', 'success');

      // TODO: Send to server
    }
  };

  // ============================================
  // Comments
  // ============================================
  const CommentManager = {
    init() {
      this.bindCommentActions();
    },

    bindCommentActions() {
      document.querySelectorAll('.comment-actions a').forEach(link => {
        link.addEventListener('click', (e) => {
          if (link.textContent.includes('Add a comment')) {
            e.preventDefault();
            this.showCommentForm(link);
          }
          if (link.textContent.includes('Show more')) {
            e.preventDefault();
            this.loadMoreComments(link);
          }
        });
      });
    },

    showCommentForm(trigger) {
      const comments = trigger.closest('.comments');
      if (!comments) return;

      // Check if form already exists
      let form = comments.querySelector('.comment-form');
      if (form) {
        form.style.display = form.style.display === 'none' ? 'block' : 'none';
        return;
      }

      // Create comment form
      form = document.createElement('div');
      form.className = 'comment-form';
      form.style.marginTop = '12px';
      form.innerHTML = `
        <textarea class="s-textarea" rows="3" placeholder="Add a comment..." style="min-height: 80px;"></textarea>
        <div style="display: flex; gap: 8px; margin-top: 8px;">
          <button class="s-btn s-btn__primary s-btn__sm" type="button">Add Comment</button>
          <button class="s-btn s-btn__muted s-btn__sm" type="button">Cancel</button>
        </div>
      `;

      comments.appendChild(form);

      // Bind form buttons
      const [submitBtn, cancelBtn] = form.querySelectorAll('button');
      const textarea = form.querySelector('textarea');

      submitBtn.addEventListener('click', () => {
        const text = textarea.value.trim();
        if (text.length < 15) {
          Toast.show('Comment must be at least 15 characters', 'error');
          return;
        }
        // TODO: Submit comment
        Toast.show('Comment added', 'success');
        form.style.display = 'none';
        textarea.value = '';
      });

      cancelBtn.addEventListener('click', () => {
        form.style.display = 'none';
        textarea.value = '';
      });

      textarea.focus();
    },

    loadMoreComments(trigger) {
      // TODO: Load more comments via AJAX
      Toast.show('Loading more comments...', 'success');
    }
  };

  // ============================================
  // Toast Notifications
  // ============================================
  const Toast = {
    element: null,
    timeout: null,

    init() {
      this.element = document.getElementById('toast');
    },

    show(message, type = 'info', duration = 3000) {
      if (!this.element) return;

      // Clear existing timeout
      if (this.timeout) {
        clearTimeout(this.timeout);
      }

      // Update toast
      const messageEl = this.element.querySelector('.toast-message');
      if (messageEl) {
        messageEl.textContent = message;
      } else {
        this.element.textContent = message;
      }

      // Set type class
      this.element.className = 'toast is-visible';
      if (type === 'success') this.element.classList.add('toast--success');
      if (type === 'error') this.element.classList.add('toast--error');

      // Auto hide
      this.timeout = setTimeout(() => {
        this.element.classList.remove('is-visible');
      }, duration);
    }
  };

  // ============================================
  // Form Enhancements
  // ============================================
  const FormEnhancements = {
    init() {
      this.addCharacterCounters();
      this.enhanceTextareas();
    },

    addCharacterCounters() {
      document.querySelectorAll('textarea[maxlength]').forEach(textarea => {
        const max = parseInt(textarea.getAttribute('maxlength'), 10);
        if (!max) return;

        const counter = document.createElement('div');
        counter.className = 's-description';
        counter.style.textAlign = 'right';
        counter.textContent = `0 / ${max}`;
        textarea.parentNode.appendChild(counter);

        textarea.addEventListener('input', () => {
          counter.textContent = `${textarea.value.length} / ${max}`;
          if (textarea.value.length > max * 0.9) {
            counter.style.color = 'var(--red-400)';
          } else {
            counter.style.color = '';
          }
        });
      });
    },

    enhanceTextareas() {
      // Auto-resize textareas
      document.querySelectorAll('textarea').forEach(textarea => {
        textarea.addEventListener('input', () => {
          textarea.style.height = 'auto';
          textarea.style.height = Math.min(textarea.scrollHeight, 500) + 'px';
        });
      });
    }
  };

  // ============================================
  // Search Enhancements
  // ============================================
  const SearchEnhancements = {
    init() {
      this.bindSearchShortcut();
    },

    bindSearchShortcut() {
      document.addEventListener('keydown', (e) => {
        // Focus search on '/' key (if not in input)
        if (e.key === '/' && !this.isInputFocused()) {
          e.preventDefault();
          const searchInput = document.querySelector('.topbar-search input');
          if (searchInput) {
            searchInput.focus();
            searchInput.select();
          }
        }
      });
    },

    isInputFocused() {
      const active = document.activeElement;
      return active && (
        active.tagName === 'INPUT' ||
        active.tagName === 'TEXTAREA' ||
        active.isContentEditable
      );
    }
  };

  // ============================================
  // Copy to Clipboard
  // ============================================
  const CopyManager = {
    init() {
      this.bindShareLinks();
    },

    bindShareLinks() {
      document.querySelectorAll('.post-menu a').forEach(link => {
        if (link.textContent.trim() === 'Share') {
          link.addEventListener('click', (e) => {
            e.preventDefault();
            this.copyCurrentUrl();
          });
        }
      });
    },

    async copyCurrentUrl() {
      try {
        await navigator.clipboard.writeText(window.location.href);
        Toast.show('Link copied to clipboard', 'success');
      } catch (err) {
        // Fallback for older browsers
        const input = document.createElement('input');
        input.value = window.location.href;
        document.body.appendChild(input);
        input.select();
        document.execCommand('copy');
        document.body.removeChild(input);
        Toast.show('Link copied to clipboard', 'success');
      }
    }
  };

  // ============================================
  // Accessibility Enhancements
  // ============================================
  const A11y = {
    init() {
      this.enhanceKeyboardNav();
      this.addFocusIndicators();
    },

    enhanceKeyboardNav() {
      // Make vote buttons keyboard accessible
      document.querySelectorAll('.vote-btn, .bookmark-btn').forEach(btn => {
        if (!btn.hasAttribute('tabindex')) {
          btn.setAttribute('tabindex', '0');
        }
        btn.addEventListener('keydown', (e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            btn.click();
          }
        });
      });
    },

    addFocusIndicators() {
      // Add keyboard-only focus styles
      document.addEventListener('keydown', (e) => {
        if (e.key === 'Tab') {
          document.body.classList.add('keyboard-nav');
        }
      });
      document.addEventListener('mousedown', () => {
        document.body.classList.remove('keyboard-nav');
      });
    }
  };

  // ============================================
  // Initialize Everything
  // ============================================
  function init() {
    document.body.classList.add('ready');

    ThemeManager.init();
    MobileNav.init();
    VoteManager.init();
    BookmarkManager.init();
    CommentManager.init();
    Toast.init();
    FormEnhancements.init();
    SearchEnhancements.init();
    CopyManager.init();
    A11y.init();
  }

  // Run on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Expose Toast globally for use in other scripts
  window.QA = window.QA || {};
  window.QA.Toast = Toast;

})();
