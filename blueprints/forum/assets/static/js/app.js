/**
 * Mizu Forum - Client-side JavaScript
 * Progressive enhancement for voting, forms, and interactions
 */

(function() {
    'use strict';

    // ============================================
    // Utility Functions
    // ============================================

    function getAuthToken() {
        return localStorage.getItem('auth_token') || '';
    }

    function setAuthToken(token) {
        localStorage.setItem('auth_token', token);
    }

    function clearAuthToken() {
        localStorage.removeItem('auth_token');
    }

    async function apiRequest(url, options = {}) {
        const token = getAuthToken();
        const headers = {
            'Content-Type': 'application/json',
            ...options.headers
        };

        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        const response = await fetch(url, {
            ...options,
            headers
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || 'Request failed');
        }

        return data;
    }

    function showToast(message, type = 'info') {
        const container = document.getElementById('toast-container') || createToastContainer();
        const toast = document.createElement('div');
        toast.className = `flash-message flash-message--${type}`;
        toast.textContent = message;
        toast.style.marginBottom = 'var(--space-3)';

        container.appendChild(toast);

        setTimeout(() => {
            toast.style.animation = 'slideOut 0.3s ease-out';
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }

    function createToastContainer() {
        const container = document.createElement('div');
        container.id = 'toast-container';
        container.className = 'flash-messages';
        document.body.appendChild(container);
        return container;
    }

    // ============================================
    // Vote Handling
    // ============================================

    function initVoteControls() {
        document.addEventListener('click', async function(e) {
            const voteBtn = e.target.closest('.vote-btn');
            if (!voteBtn) return;

            e.preventDefault();

            if (voteBtn.disabled) {
                showToast('Please login to vote', 'warning');
                return;
            }

            const type = voteBtn.dataset.type; // 'thread' or 'post'
            const id = voteBtn.dataset.id;
            const action = voteBtn.dataset.action; // 'upvote' or 'downvote'

            try {
                const value = action === 'upvote' ? 1 : -1;
                const endpoint = `/api/v1/${type}s/${id}/vote`;

                const result = await apiRequest(endpoint, {
                    method: 'POST',
                    body: JSON.stringify({ value })
                });

                // Update UI
                updateVoteUI(id, result.data.score, result.data.user_vote);
                showToast('Vote recorded!', 'success');
            } catch (error) {
                showToast(error.message || 'Failed to vote', 'error');
            }
        });
    }

    function updateVoteUI(id, newScore, userVote) {
        const scoreEl = document.querySelector(`[data-score-id="${id}"]`);
        if (scoreEl) {
            scoreEl.textContent = newScore;
        }

        const container = scoreEl?.closest('.vote-controls');
        if (!container) return;

        const upvoteBtn = container.querySelector('.vote-btn--upvote');
        const downvoteBtn = container.querySelector('.vote-btn--downvote');

        upvoteBtn?.classList.toggle('vote-btn--active', userVote === 1);
        downvoteBtn?.classList.toggle('vote-btn--active', userVote === -1);
    }

    // ============================================
    // Forum Join/Leave
    // ============================================

    function initForumActions() {
        document.addEventListener('click', async function(e) {
            const btn = e.target.closest('[data-forum-id]');
            if (!btn) return;

            e.preventDefault();

            const forumId = btn.dataset.forumId;
            const action = btn.dataset.action; // 'join' or 'leave'

            try {
                const endpoint = `/api/v1/forums/${forumId}/${action}`;
                await apiRequest(endpoint, { method: 'POST' });

                // Update button
                if (action === 'join') {
                    btn.textContent = 'Joined';
                    btn.classList.remove('btn--primary');
                    btn.classList.add('btn--secondary');
                    btn.dataset.action = 'leave';
                } else {
                    btn.textContent = 'Join';
                    btn.classList.remove('btn--secondary');
                    btn.classList.add('btn--primary');
                    btn.dataset.action = 'join';
                }

                showToast(`Successfully ${action === 'join' ? 'joined' : 'left'} forum!`, 'success');
            } catch (error) {
                showToast(error.message || `Failed to ${action} forum`, 'error');
            }
        });
    }

    // ============================================
    // Comment Form Handling
    // ============================================

    function initCommentForms() {
        document.addEventListener('submit', async function(e) {
            if (!e.target.classList.contains('comment-form')) return;

            e.preventDefault();
            const form = e.target;
            const formData = new FormData(form);
            const submitBtn = form.querySelector('button[type="submit"]');

            try {
                submitBtn.disabled = true;
                submitBtn.textContent = 'Posting...';

                const result = await apiRequest(form.action, {
                    method: 'POST',
                    body: JSON.stringify({
                        content: formData.get('content'),
                        parent_id: formData.get('parent_id') || ''
                    })
                });

                showToast('Comment posted successfully!', 'success');
                form.reset();

                // Reload page to show new comment (TODO: add comment dynamically)
                setTimeout(() => window.location.reload(), 1000);
            } catch (error) {
                showToast(error.message || 'Failed to post comment', 'error');
            } finally {
                submitBtn.disabled = false;
                submitBtn.textContent = 'Post Comment';
            }
        });
    }

    // ============================================
    // Post Actions (Reply, Edit, Delete, Save)
    // ============================================

    function initPostActions() {
        document.addEventListener('click', async function(e) {
            const actionLink = e.target.closest('[data-action]');
            if (!actionLink || !actionLink.classList.contains('post-card__action')) return;

            e.preventDefault();
            const action = actionLink.dataset.action;
            const postId = actionLink.dataset.postId;

            switch (action) {
                case 'reply':
                    showReplyForm(postId);
                    break;
                case 'edit':
                    showEditForm(postId);
                    break;
                case 'delete':
                    await handleDelete(postId, 'post');
                    break;
                case 'save':
                    await handleSave(postId, 'post');
                    break;
                case 'delete-thread':
                    await handleDelete(window.location.pathname.split('/').pop(), 'thread');
                    break;
                case 'save-thread':
                    await handleSave(window.location.pathname.split('/').pop(), 'thread');
                    break;
                case 'subscribe':
                    await handleSubscribe();
                    break;
                case 'report':
                case 'report-thread':
                    handleReport(postId || 'thread');
                    break;
            }
        });
    }

    function showReplyForm(postId) {
        // TODO: Implement inline reply form
        showToast('Reply functionality coming soon!', 'info');
    }

    function showEditForm(postId) {
        // TODO: Implement inline edit form
        showToast('Edit functionality coming soon!', 'info');
    }

    async function handleDelete(id, type) {
        if (!confirm(`Are you sure you want to delete this ${type}?`)) {
            return;
        }

        try {
            await apiRequest(`/api/v1/${type}s/${id}`, {
                method: 'DELETE'
            });

            showToast(`${type.charAt(0).toUpperCase() + type.slice(1)} deleted successfully!`, 'success');

            if (type === 'thread') {
                // Redirect to forum
                setTimeout(() => window.history.back(), 1000);
            } else {
                // Remove post from DOM
                setTimeout(() => window.location.reload(), 1000);
            }
        } catch (error) {
            showToast(error.message || `Failed to delete ${type}`, 'error');
        }
    }

    async function handleSave(id, type) {
        try {
            await apiRequest(`/api/v1/${type}s/${id}/save`, {
                method: 'POST'
            });

            showToast('Saved!', 'success');
        } catch (error) {
            showToast(error.message || 'Failed to save', 'error');
        }
    }

    async function handleSubscribe() {
        try {
            const threadId = window.location.pathname.split('/').pop();
            await apiRequest(`/api/v1/threads/${threadId}/subscribe`, {
                method: 'POST'
            });

            showToast('Subscribed to thread!', 'success');
        } catch (error) {
            showToast(error.message || 'Failed to subscribe', 'error');
        }
    }

    function handleReport(id) {
        // TODO: Implement report modal
        showToast('Report functionality coming soon!', 'info');
    }

    // ============================================
    // Markdown Preview
    // ============================================

    function initMarkdownPreview() {
        const textareas = document.querySelectorAll('.form-textarea');
        textareas.forEach(textarea => {
            // TODO: Add markdown preview toggle
        });
    }

    // ============================================
    // Auto-save Drafts
    // ============================================

    function initAutoSave() {
        const forms = document.querySelectorAll('form[data-autosave]');
        forms.forEach(form => {
            const key = `draft_${form.dataset.autosave}`;
            const textarea = form.querySelector('textarea');

            if (!textarea) return;

            // Load saved draft
            const saved = localStorage.getItem(key);
            if (saved && !textarea.value) {
                textarea.value = saved;
                showToast('Draft restored', 'info');
            }

            // Save on input
            textarea.addEventListener('input', () => {
                localStorage.setItem(key, textarea.value);
            });

            // Clear on submit
            form.addEventListener('submit', () => {
                localStorage.removeItem(key);
            });
        });
    }

    // ============================================
    // Keyboard Shortcuts
    // ============================================

    function initKeyboardShortcuts() {
        document.addEventListener('keydown', function(e) {
            // Cmd/Ctrl + K for search (TODO)
            if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
                e.preventDefault();
                // TODO: Open search modal
            }

            // Escape to close modals
            if (e.key === 'Escape') {
                closeAllModals();
            }
        });
    }

    function closeAllModals() {
        // TODO: Close any open modals
    }

    // ============================================
    // Infinite Scroll (Optional)
    // ============================================

    function initInfiniteScroll() {
        const loadMoreBtn = document.querySelector('[data-load-more]');
        if (!loadMoreBtn) return;

        const observer = new IntersectionObserver(entries => {
            entries.forEach(entry => {
                if (entry.isIntersecting && !loadMoreBtn.disabled) {
                    loadMoreBtn.click();
                }
            });
        });

        observer.observe(loadMoreBtn);
    }

    // ============================================
    // Image Upload (for future use)
    // ============================================

    function initImageUpload() {
        // TODO: Implement image upload handling
    }

    // ============================================
    // Initialize Everything
    // ============================================

    function init() {
        console.log('Mizu Forum initialized');

        initVoteControls();
        initForumActions();
        initCommentForms();
        initPostActions();
        initMarkdownPreview();
        initAutoSave();
        initKeyboardShortcuts();
        initInfiniteScroll();
        initImageUpload();
    }

    // Run on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();
