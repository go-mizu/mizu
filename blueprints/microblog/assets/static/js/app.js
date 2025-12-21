// Microblog - Client-side JavaScript
// Enhanced version with full feature set

(function() {
    'use strict';

    // =========================================================================
    // API Helper
    // =========================================================================
    const api = {
        async fetch(url, options = {}) {
            const token = localStorage.getItem('token');
            const headers = {
                'Content-Type': 'application/json',
                ...options.headers,
            };
            if (token) {
                headers['Authorization'] = `Bearer ${token}`;
            }

            const response = await fetch(url, { ...options, headers });

            // Handle non-JSON responses
            const contentType = response.headers.get('content-type');
            if (!contentType || !contentType.includes('application/json')) {
                if (!response.ok) {
                    throw new Error(`Request failed with status ${response.status}`);
                }
                return { data: null };
            }

            const data = await response.json();
            if (!response.ok) {
                throw new Error(data.error?.message || 'Request failed');
            }
            return data;
        },
        get(url) { return this.fetch(url); },
        post(url, body) { return this.fetch(url, { method: 'POST', body: JSON.stringify(body) }); },
        put(url, body) { return this.fetch(url, { method: 'PUT', body: JSON.stringify(body) }); },
        patch(url, body) { return this.fetch(url, { method: 'PATCH', body: JSON.stringify(body) }); },
        delete(url) { return this.fetch(url, { method: 'DELETE' }); },
    };

    // =========================================================================
    // Toast Notifications System
    // =========================================================================
    const toast = {
        container: null,

        init() {
            this.container = document.getElementById('toast-container');
            if (!this.container) {
                this.container = document.createElement('div');
                this.container.id = 'toast-container';
                this.container.className = 'toast-container';
                document.body.appendChild(this.container);
            }
        },

        show(message, type = 'info', duration = 4000, action = null) {
            const toast = document.createElement('div');
            toast.className = `toast toast-${type}`;

            const icons = {
                success: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg>',
                error: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>',
                warning: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>',
                info: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>',
            };

            let html = `
                <div class="toast-icon">${icons[type] || icons.info}</div>
                <div class="toast-content">
                    <span class="toast-message">${this.escapeHtml(message)}</span>
                </div>
                <button class="toast-close" aria-label="Close">&times;</button>
            `;

            if (action) {
                html = `
                    <div class="toast-icon">${icons[type] || icons.info}</div>
                    <div class="toast-content">
                        <span class="toast-message">${this.escapeHtml(message)}</span>
                        <button class="toast-action" data-action="toast-action">${this.escapeHtml(action.label)}</button>
                    </div>
                    <button class="toast-close" aria-label="Close">&times;</button>
                `;
            }

            toast.innerHTML = html;
            this.container.appendChild(toast);

            // Trigger animation
            requestAnimationFrame(() => toast.classList.add('toast-visible'));

            // Close button
            const closeBtn = toast.querySelector('.toast-close');
            closeBtn.addEventListener('click', () => this.dismiss(toast));

            // Action button
            if (action) {
                const actionBtn = toast.querySelector('.toast-action');
                actionBtn.addEventListener('click', () => {
                    action.callback();
                    this.dismiss(toast);
                });
            }

            // Auto dismiss
            if (duration > 0) {
                setTimeout(() => this.dismiss(toast), duration);
            }

            return toast;
        },

        dismiss(toastElement) {
            toastElement.classList.remove('toast-visible');
            toastElement.classList.add('toast-hiding');
            setTimeout(() => toastElement.remove(), 300);
        },

        success(message, duration) { return this.show(message, 'success', duration); },
        error(message, duration) { return this.show(message, 'error', duration); },
        warning(message, duration) { return this.show(message, 'warning', duration); },
        info(message, duration) { return this.show(message, 'info', duration); },

        escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
    };

    // =========================================================================
    // Modal System
    // =========================================================================
    const modal = {
        container: null,
        activeModal: null,

        init() {
            this.container = document.getElementById('modal-container');
            if (!this.container) {
                this.container = document.createElement('div');
                this.container.id = 'modal-container';
                this.container.className = 'modal-backdrop';
                this.container.style.display = 'none';
                document.body.appendChild(this.container);
            }

            // Close on backdrop click
            this.container.addEventListener('click', (e) => {
                if (e.target === this.container) {
                    this.close();
                }
            });

            // Close on escape
            document.addEventListener('keydown', (e) => {
                if (e.key === 'Escape' && this.activeModal) {
                    this.close();
                }
            });
        },

        open(content, options = {}) {
            const { className = '', onClose = null } = options;

            this.container.innerHTML = `
                <div class="modal ${className}" role="dialog" aria-modal="true">
                    <button class="modal-close" aria-label="Close modal">&times;</button>
                    <div class="modal-content">${content}</div>
                </div>
            `;

            this.activeModal = this.container.querySelector('.modal');
            this.container.style.display = 'flex';
            document.body.style.overflow = 'hidden';

            // Focus first focusable element
            const focusable = this.activeModal.querySelector('input, textarea, button:not(.modal-close)');
            if (focusable) focusable.focus();

            // Close button
            const closeBtn = this.activeModal.querySelector('.modal-close');
            closeBtn.addEventListener('click', () => this.close());

            if (onClose) {
                this.onCloseCallback = onClose;
            }

            return this.activeModal;
        },

        close() {
            if (!this.activeModal) return;

            this.container.style.display = 'none';
            document.body.style.overflow = '';

            if (this.onCloseCallback) {
                this.onCloseCallback();
                this.onCloseCallback = null;
            }

            this.activeModal = null;
            this.container.innerHTML = '';
        },

        confirm(message, options = {}) {
            return new Promise((resolve) => {
                const {
                    title = 'Confirm',
                    confirmText = 'Confirm',
                    cancelText = 'Cancel',
                    danger = false
                } = options;

                const content = `
                    <div class="modal-header">
                        <h3 class="modal-title">${this.escapeHtml(title)}</h3>
                    </div>
                    <div class="modal-body">
                        <p>${this.escapeHtml(message)}</p>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" data-action="cancel">${this.escapeHtml(cancelText)}</button>
                        <button class="btn ${danger ? 'btn-danger' : 'btn-primary'}" data-action="confirm">${this.escapeHtml(confirmText)}</button>
                    </div>
                `;

                const modalEl = this.open(content, { className: 'modal-sm' });

                modalEl.querySelector('[data-action="cancel"]').addEventListener('click', () => {
                    this.close();
                    resolve(false);
                });

                modalEl.querySelector('[data-action="confirm"]').addEventListener('click', () => {
                    this.close();
                    resolve(true);
                });

                this.onCloseCallback = () => resolve(false);
            });
        },

        compose(options = {}) {
            const { replyTo = null, quote = null, initialContent = '' } = options;

            let header = 'New Post';
            let placeholder = "What's on your mind?";

            if (replyTo) {
                header = 'Reply';
                placeholder = 'Write your reply...';
            } else if (quote) {
                header = 'Quote Post';
            }

            const content = `
                <div class="modal-header">
                    <h3 class="modal-title">${header}</h3>
                </div>
                <div class="modal-body">
                    ${replyTo ? `<div class="reply-context">Replying to @${this.escapeHtml(replyTo.username)}</div>` : ''}
                    <form id="modal-compose-form" class="compose-form">
                        <textarea
                            name="content"
                            class="compose-textarea"
                            placeholder="${placeholder}"
                            maxlength="500"
                            rows="4"
                        >${this.escapeHtml(initialContent)}</textarea>
                        <div class="compose-footer">
                            <div class="compose-actions">
                                <button type="button" class="compose-action" title="Add image" disabled>
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="m21 15-5-5L5 21"/></svg>
                                </button>
                                <button type="button" class="compose-action" title="Add poll" disabled>
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 20V10M12 20V4M6 20v-6"/></svg>
                                </button>
                            </div>
                            <div class="compose-submit">
                                <span class="char-count"><span id="modal-char-count">0</span>/500</span>
                                <button type="submit" class="btn btn-primary">Post</button>
                            </div>
                        </div>
                    </form>
                    ${quote ? `
                        <div class="quote-preview post-card post-card-compact">
                            <div class="post-header">
                                <span class="display-name">${this.escapeHtml(quote.displayName)}</span>
                                <span class="username">@${this.escapeHtml(quote.username)}</span>
                            </div>
                            <div class="post-content">${this.escapeHtml(quote.content)}</div>
                        </div>
                    ` : ''}
                </div>
            `;

            const modalEl = this.open(content, { className: 'modal-compose' });

            const textarea = modalEl.querySelector('textarea');
            const charCount = modalEl.querySelector('#modal-char-count');
            const form = modalEl.querySelector('#modal-compose-form');

            // Character counter
            textarea.addEventListener('input', () => {
                const count = [...textarea.value].length;
                charCount.textContent = count;
                charCount.parentElement.classList.toggle('char-limit', count > 480);
                charCount.parentElement.classList.toggle('char-over', count > 500);
            });

            // Submit handler
            form.addEventListener('submit', async (e) => {
                e.preventDefault();
                const content = textarea.value.trim();
                if (!content) return;

                const submitBtn = form.querySelector('button[type="submit"]');
                submitBtn.disabled = true;
                submitBtn.textContent = 'Posting...';

                try {
                    const payload = { content };
                    if (replyTo) payload.reply_to_id = replyTo.id;
                    if (quote) payload.quote_id = quote.id;

                    await api.post('/api/v1/posts', payload);
                    this.close();
                    toast.success('Post created successfully!');

                    // Reload timeline if on home page
                    if (window.location.pathname === '/') {
                        setTimeout(() => window.location.reload(), 500);
                    }
                } catch (err) {
                    toast.error('Failed to create post: ' + err.message);
                    submitBtn.disabled = false;
                    submitBtn.textContent = 'Post';
                }
            });

            return modalEl;
        },

        lightbox(imageUrl, alt = '') {
            const content = `
                <img src="${this.escapeHtml(imageUrl)}" alt="${this.escapeHtml(alt)}" class="lightbox-image" />
            `;
            return this.open(content, { className: 'modal-lightbox' });
        },

        escapeHtml(text) {
            if (!text) return '';
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
    };

    // =========================================================================
    // Theme System
    // =========================================================================
    const theme = {
        init() {
            // Check for saved preference or system preference
            const saved = localStorage.getItem('theme');
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;

            if (saved) {
                this.set(saved);
            } else if (prefersDark) {
                this.set('dark');
            } else {
                this.set('light');
            }

            // Listen for system preference changes
            window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
                if (!localStorage.getItem('theme')) {
                    this.set(e.matches ? 'dark' : 'light');
                }
            });

            // Theme toggle button
            document.addEventListener('click', (e) => {
                const toggle = e.target.closest('[data-action="toggle-theme"]');
                if (toggle) {
                    this.toggle();
                }
            });
        },

        get() {
            return document.documentElement.getAttribute('data-theme') || 'light';
        },

        set(themeName) {
            document.documentElement.setAttribute('data-theme', themeName);
            localStorage.setItem('theme', themeName);

            // Update toggle button if exists
            const toggles = document.querySelectorAll('[data-action="toggle-theme"]');
            toggles.forEach(toggle => {
                const icon = toggle.querySelector('.theme-icon');
                if (icon) {
                    icon.innerHTML = themeName === 'dark'
                        ? '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>'
                        : '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>';
                }
            });
        },

        toggle() {
            const current = this.get();
            this.set(current === 'dark' ? 'light' : 'dark');
        }
    };

    // =========================================================================
    // Form Validation
    // =========================================================================
    const validation = {
        rules: {
            required: (value) => value.trim() !== '' || 'This field is required',
            email: (value) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value) || 'Please enter a valid email',
            minLength: (min) => (value) => value.length >= min || `Must be at least ${min} characters`,
            maxLength: (max) => (value) => value.length <= max || `Must be at most ${max} characters`,
            match: (fieldName, fieldLabel) => (value, form) => {
                const other = form.querySelector(`[name="${fieldName}"]`);
                return !other || value === other.value || `Must match ${fieldLabel}`;
            },
            username: (value) => /^[a-zA-Z0-9_]{3,20}$/.test(value) || 'Username must be 3-20 alphanumeric characters',
            password: (value) => value.length >= 8 || 'Password must be at least 8 characters',
        },

        init() {
            // Real-time validation
            document.addEventListener('input', (e) => {
                if (!e.target.dataset.validate) return;
                this.validateField(e.target);
            });

            document.addEventListener('blur', (e) => {
                if (!e.target.dataset.validate) return;
                this.validateField(e.target);
            }, true);
        },

        validateField(field) {
            const rules = field.dataset.validate.split(' ');
            const form = field.closest('form');
            let error = null;

            for (const ruleName of rules) {
                let rule = this.rules[ruleName];

                // Handle parameterized rules
                if (ruleName.includes(':')) {
                    const [name, param] = ruleName.split(':');
                    if (name === 'minLength' || name === 'maxLength') {
                        rule = this.rules[name](parseInt(param));
                    } else if (name === 'match') {
                        const [fieldName, label] = param.split(',');
                        rule = this.rules[name](fieldName, label || fieldName);
                    }
                }

                if (rule) {
                    const result = rule(field.value, form);
                    if (result !== true) {
                        error = result;
                        break;
                    }
                }
            }

            this.showFieldError(field, error);
            return !error;
        },

        validateForm(form) {
            const fields = form.querySelectorAll('[data-validate]');
            let valid = true;

            fields.forEach(field => {
                if (!this.validateField(field)) {
                    valid = false;
                }
            });

            return valid;
        },

        showFieldError(field, error) {
            const group = field.closest('.form-group') || field.parentElement;
            let errorEl = group.querySelector('.field-error');

            if (error) {
                field.classList.add('field-invalid');
                field.classList.remove('field-valid');

                if (!errorEl) {
                    errorEl = document.createElement('span');
                    errorEl.className = 'field-error';
                    group.appendChild(errorEl);
                }
                errorEl.textContent = error;
            } else {
                field.classList.remove('field-invalid');
                if (field.value) field.classList.add('field-valid');

                if (errorEl) errorEl.remove();
            }
        }
    };

    // =========================================================================
    // Character Counter
    // =========================================================================
    function initCharCounter() {
        document.addEventListener('input', (e) => {
            const textarea = e.target.closest('textarea[data-char-counter]');
            if (!textarea) return;

            const counterId = textarea.dataset.charCounter;
            const counter = document.getElementById(counterId);
            if (!counter) return;

            const max = parseInt(textarea.getAttribute('maxlength')) || 500;
            const count = [...textarea.value].length;

            counter.textContent = count;

            const wrapper = counter.closest('.char-count');
            if (wrapper) {
                wrapper.classList.toggle('char-limit', count > max - 20);
                wrapper.classList.toggle('char-over', count > max);
            }
        });
    }

    // =========================================================================
    // Post Actions (Like, Repost, Bookmark) with Optimistic Updates
    // =========================================================================
    function initPostActions() {
        document.addEventListener('click', async (e) => {
            const button = e.target.closest('[data-action]');
            if (!button) return;

            const action = button.dataset.action;
            const postId = button.dataset.postId;
            if (!postId) return;

            // Skip non-post actions
            if (!['like', 'repost', 'bookmark', 'reply', 'share', 'delete'].includes(action)) return;

            e.preventDefault();

            if (action === 'reply') {
                // Open reply modal
                const postCard = button.closest('.post-card');
                if (postCard) {
                    modal.compose({
                        replyTo: {
                            id: postId,
                            username: postCard.querySelector('.username')?.textContent?.replace('@', '') || ''
                        }
                    });
                }
                return;
            }

            if (action === 'share') {
                // Use Web Share API or copy link
                const postUrl = `${window.location.origin}/@${button.dataset.username || 'user'}/${postId}`;
                if (navigator.share) {
                    try {
                        await navigator.share({ url: postUrl });
                    } catch (err) {
                        // User cancelled or error
                    }
                } else {
                    await navigator.clipboard.writeText(postUrl);
                    toast.success('Link copied to clipboard!');
                }
                return;
            }

            if (action === 'delete') {
                const confirmed = await modal.confirm(
                    'Are you sure you want to delete this post? This action cannot be undone.',
                    { title: 'Delete Post', confirmText: 'Delete', danger: true }
                );

                if (!confirmed) return;

                try {
                    await api.delete(`/api/v1/posts/${postId}`);
                    const postCard = button.closest('.post-card');
                    if (postCard) {
                        postCard.style.opacity = '0';
                        postCard.style.transform = 'translateX(-20px)';
                        setTimeout(() => postCard.remove(), 200);
                    }
                    toast.success('Post deleted');
                } catch (err) {
                    toast.error('Failed to delete post: ' + err.message);
                }
                return;
            }

            // Optimistic update for like/repost/bookmark
            const isActive = button.classList.contains('active');
            const countEl = button.querySelector('.action-count');
            const currentCount = countEl ? parseInt(countEl.textContent) || 0 : 0;

            // Toggle state immediately (optimistic)
            button.classList.toggle('active');
            if (countEl) {
                countEl.textContent = isActive ? Math.max(0, currentCount - 1) : currentCount + 1;

                // Animate count change
                countEl.classList.add('count-animate');
                setTimeout(() => countEl.classList.remove('count-animate'), 300);
            }

            try {
                const method = isActive ? 'delete' : 'post';
                await api[method](`/api/v1/posts/${postId}/${action}`);
            } catch (err) {
                // Rollback on error
                button.classList.toggle('active');
                if (countEl) {
                    countEl.textContent = currentCount;
                }
                toast.error(`Failed to ${action} post`);
            }
        });
    }

    // =========================================================================
    // Form Submission Handlers
    // =========================================================================
    function initForms() {
        // Compose form (inline, not modal)
        document.addEventListener('submit', async (e) => {
            const form = e.target;

            // Compose form
            if (form.matches('[data-form="compose"]')) {
                e.preventDefault();

                const textarea = form.querySelector('textarea[name="content"]');
                const content = textarea.value.trim();
                const replyToId = form.dataset.replyTo;

                if (!content) {
                    toast.warning('Please enter some content');
                    return;
                }

                const submitBtn = form.querySelector('button[type="submit"]');
                const originalText = submitBtn.textContent;
                submitBtn.disabled = true;
                submitBtn.textContent = 'Posting...';

                try {
                    const payload = { content };
                    if (replyToId) payload.reply_to_id = replyToId;

                    await api.post('/api/v1/posts', payload);
                    textarea.value = '';

                    const charCount = form.querySelector('.char-count span');
                    if (charCount) charCount.textContent = '0';

                    toast.success('Post created!');

                    // Reload timeline
                    setTimeout(() => window.location.reload(), 500);
                } catch (err) {
                    toast.error('Failed to create post: ' + err.message);
                } finally {
                    submitBtn.disabled = false;
                    submitBtn.textContent = originalText;
                }
                return;
            }

            // Login form
            if (form.matches('[data-form="login"]')) {
                e.preventDefault();

                if (!validation.validateForm(form)) return;

                const submitBtn = form.querySelector('button[type="submit"]');
                submitBtn.disabled = true;
                submitBtn.innerHTML = '<span class="loading-spinner"></span> Logging in...';

                try {
                    const result = await api.post('/api/v1/auth/login', {
                        username: form.username.value,
                        password: form.password.value,
                    });
                    localStorage.setItem('token', result.data.token);
                    toast.success('Welcome back!');
                    setTimeout(() => window.location.href = '/', 500);
                } catch (err) {
                    toast.error('Login failed: ' + err.message);
                    submitBtn.disabled = false;
                    submitBtn.textContent = 'Login';
                }
                return;
            }

            // Register form
            if (form.matches('[data-form="register"]')) {
                e.preventDefault();

                if (!validation.validateForm(form)) return;

                const submitBtn = form.querySelector('button[type="submit"]');
                submitBtn.disabled = true;
                submitBtn.innerHTML = '<span class="loading-spinner"></span> Creating account...';

                try {
                    const result = await api.post('/api/v1/auth/register', {
                        username: form.username.value,
                        email: form.email.value,
                        password: form.password.value,
                        display_name: form.display_name?.value || form.username.value,
                    });
                    localStorage.setItem('token', result.data.token);
                    toast.success('Account created! Welcome!');
                    setTimeout(() => window.location.href = '/', 500);
                } catch (err) {
                    toast.error('Registration failed: ' + err.message);
                    submitBtn.disabled = false;
                    submitBtn.textContent = 'Create Account';
                }
                return;
            }

            // Settings form
            if (form.matches('[data-form="settings"]')) {
                e.preventDefault();

                const submitBtn = form.querySelector('button[type="submit"]');
                submitBtn.disabled = true;
                submitBtn.innerHTML = '<span class="loading-spinner"></span> Saving...';

                try {
                    const formData = new FormData(form);
                    const data = Object.fromEntries(formData.entries());

                    await api.patch('/api/v1/accounts/update_credentials', data);
                    toast.success('Settings saved!');
                } catch (err) {
                    toast.error('Failed to save settings: ' + err.message);
                } finally {
                    submitBtn.disabled = false;
                    submitBtn.textContent = 'Save Changes';
                }
                return;
            }
        });
    }

    // =========================================================================
    // Follow/Unfollow Buttons
    // =========================================================================
    function initFollowButtons() {
        document.addEventListener('click', async (e) => {
            const button = e.target.closest('[data-follow]');
            if (!button) return;

            e.preventDefault();

            const accountId = button.dataset.follow;
            const isFollowing = button.dataset.following === 'true';

            // Optimistic update
            button.disabled = true;
            button.classList.toggle('btn-following', !isFollowing);
            button.dataset.following = (!isFollowing).toString();
            button.textContent = isFollowing ? 'Follow' : 'Following';

            try {
                const endpoint = isFollowing ? 'unfollow' : 'follow';
                await api.post(`/api/v1/accounts/${accountId}/${endpoint}`);
                toast.success(isFollowing ? 'Unfollowed' : 'Followed!');
            } catch (err) {
                // Rollback
                button.classList.toggle('btn-following', isFollowing);
                button.dataset.following = isFollowing.toString();
                button.textContent = isFollowing ? 'Following' : 'Follow';
                toast.error(`Failed to ${isFollowing ? 'unfollow' : 'follow'}: ` + err.message);
            } finally {
                button.disabled = false;
            }
        });

        // Hover effect for following button
        document.addEventListener('mouseenter', (e) => {
            const button = e.target.closest('[data-follow][data-following="true"]');
            if (button) button.textContent = 'Unfollow';
        }, true);

        document.addEventListener('mouseleave', (e) => {
            const button = e.target.closest('[data-follow][data-following="true"]');
            if (button) button.textContent = 'Following';
        }, true);
    }

    // =========================================================================
    // Infinite Scroll
    // =========================================================================
    function initInfiniteScroll() {
        const timeline = document.getElementById('timeline');
        if (!timeline) return;

        let loading = false;
        let hasMore = true;
        let maxId = null;

        // Create loading indicator
        const loadingIndicator = document.createElement('div');
        loadingIndicator.className = 'loading-more';
        loadingIndicator.innerHTML = '<span class="loading-spinner"></span> Loading more...';
        loadingIndicator.style.display = 'none';
        timeline.parentElement.appendChild(loadingIndicator);

        // Get initial max_id from last post
        const posts = timeline.querySelectorAll('[data-post-id]');
        if (posts.length > 0) {
            maxId = posts[posts.length - 1].dataset.postId;
        }

        // Determine API endpoint based on current page
        let endpoint = '/api/v1/timelines/home';
        const path = window.location.pathname;

        if (path === '/explore') {
            endpoint = '/api/v1/trends/posts';
        } else if (path.startsWith('/tags/')) {
            const tag = path.split('/tags/')[1];
            endpoint = `/api/v1/timelines/tag/${tag}`;
        } else if (path.startsWith('/@') && !path.includes('/', 2)) {
            // Profile page - would need account ID
            return; // Skip for now
        } else if (path === '/bookmarks') {
            endpoint = '/api/v1/bookmarks';
        }

        const observer = new IntersectionObserver(async (entries) => {
            const entry = entries[0];
            if (!entry.isIntersecting || loading || !hasMore) return;

            loading = true;
            loadingIndicator.style.display = 'flex';

            try {
                const url = new URL(endpoint, window.location.origin);
                if (maxId) url.searchParams.set('max_id', maxId);
                url.searchParams.set('limit', '20');

                const result = await api.get(url.toString());
                const newPosts = result.data || [];

                if (newPosts.length === 0) {
                    hasMore = false;
                    loadingIndicator.innerHTML = '<span class="text-muted">No more posts</span>';
                    observer.disconnect();
                    return;
                }

                maxId = newPosts[newPosts.length - 1].id;

                // Append new posts (would need server-side rendering or template)
                // For now, reload the page with new posts appended
                // In production, you'd render posts client-side here

            } catch (err) {
                console.error('Failed to load more posts:', err);
                toast.error('Failed to load more posts');
            } finally {
                loading = false;
                loadingIndicator.style.display = 'none';
            }
        }, {
            rootMargin: '200px',
            threshold: 0
        });

        // Create sentinel element
        const sentinel = document.createElement('div');
        sentinel.className = 'scroll-sentinel';
        timeline.parentElement.appendChild(sentinel);
        observer.observe(sentinel);
    }

    // =========================================================================
    // Keyboard Shortcuts
    // =========================================================================
    const shortcuts = {
        enabled: true,
        focusedPostIndex: -1,

        init() {
            document.addEventListener('keydown', (e) => {
                // Don't trigger shortcuts when typing in inputs
                if (e.target.matches('input, textarea, [contenteditable]')) return;
                if (!this.enabled) return;

                // Modifier keys check
                if (e.ctrlKey || e.altKey || e.metaKey) return;

                switch (e.key.toLowerCase()) {
                    case 'n':
                        e.preventDefault();
                        this.newPost();
                        break;
                    case 'j':
                        e.preventDefault();
                        this.navigatePosts(1);
                        break;
                    case 'k':
                        e.preventDefault();
                        this.navigatePosts(-1);
                        break;
                    case 'l':
                        e.preventDefault();
                        this.likeCurrentPost();
                        break;
                    case 'r':
                        e.preventDefault();
                        this.replyToCurrentPost();
                        break;
                    case 'b':
                        e.preventDefault();
                        this.bookmarkCurrentPost();
                        break;
                    case 'o':
                    case 'enter':
                        e.preventDefault();
                        this.openCurrentPost();
                        break;
                    case 'g':
                        // Wait for second key
                        this.waitForSecondKey();
                        break;
                    case '?':
                        e.preventDefault();
                        this.showHelp();
                        break;
                    case '/':
                        e.preventDefault();
                        this.focusSearch();
                        break;
                    case 'escape':
                        this.clearFocus();
                        break;
                }
            });
        },

        newPost() {
            modal.compose();
        },

        navigatePosts(direction) {
            const posts = document.querySelectorAll('.post-card');
            if (posts.length === 0) return;

            this.focusedPostIndex += direction;
            this.focusedPostIndex = Math.max(0, Math.min(this.focusedPostIndex, posts.length - 1));

            // Remove previous focus
            document.querySelectorAll('.post-card.keyboard-focus').forEach(p => {
                p.classList.remove('keyboard-focus');
            });

            // Add focus to current
            const post = posts[this.focusedPostIndex];
            post.classList.add('keyboard-focus');
            post.scrollIntoView({ behavior: 'smooth', block: 'center' });
        },

        getCurrentPost() {
            const focused = document.querySelector('.post-card.keyboard-focus');
            if (focused) return focused;

            const posts = document.querySelectorAll('.post-card');
            return posts[0] || null;
        },

        likeCurrentPost() {
            const post = this.getCurrentPost();
            if (!post) return;
            const likeBtn = post.querySelector('[data-action="like"]');
            if (likeBtn) likeBtn.click();
        },

        replyToCurrentPost() {
            const post = this.getCurrentPost();
            if (!post) return;
            const replyBtn = post.querySelector('[data-action="reply"]');
            if (replyBtn) replyBtn.click();
        },

        bookmarkCurrentPost() {
            const post = this.getCurrentPost();
            if (!post) return;
            const bookmarkBtn = post.querySelector('[data-action="bookmark"]');
            if (bookmarkBtn) bookmarkBtn.click();
        },

        openCurrentPost() {
            const post = this.getCurrentPost();
            if (!post) return;
            const link = post.querySelector('.post-content a, .post-timestamp');
            if (link) {
                window.location.href = link.href;
            }
        },

        waitForSecondKey() {
            const handler = (e) => {
                document.removeEventListener('keydown', handler);

                switch (e.key.toLowerCase()) {
                    case 'h':
                        window.location.href = '/';
                        break;
                    case 'n':
                        window.location.href = '/notifications';
                        break;
                    case 'e':
                        window.location.href = '/explore';
                        break;
                    case 'p':
                        // Go to own profile
                        const profileLink = document.querySelector('.nav-profile a');
                        if (profileLink) window.location.href = profileLink.href;
                        break;
                    case 's':
                        window.location.href = '/settings';
                        break;
                }
            };

            setTimeout(() => {
                document.removeEventListener('keydown', handler);
            }, 1000);

            document.addEventListener('keydown', handler);
        },

        focusSearch() {
            const searchInput = document.querySelector('.search-input, input[type="search"], input[name="q"]');
            if (searchInput) {
                searchInput.focus();
                searchInput.select();
            }
        },

        clearFocus() {
            document.querySelectorAll('.post-card.keyboard-focus').forEach(p => {
                p.classList.remove('keyboard-focus');
            });
            this.focusedPostIndex = -1;

            // Also close modal if open
            if (modal.activeModal) {
                modal.close();
            }
        },

        showHelp() {
            const content = `
                <div class="modal-header">
                    <h3 class="modal-title">Keyboard Shortcuts</h3>
                </div>
                <div class="modal-body shortcuts-help">
                    <div class="shortcut-section">
                        <h4>Navigation</h4>
                        <div class="shortcut"><kbd>j</kbd> <span>Next post</span></div>
                        <div class="shortcut"><kbd>k</kbd> <span>Previous post</span></div>
                        <div class="shortcut"><kbd>o</kbd> / <kbd>Enter</kbd> <span>Open post</span></div>
                        <div class="shortcut"><kbd>/</kbd> <span>Focus search</span></div>
                        <div class="shortcut"><kbd>Esc</kbd> <span>Clear focus / Close modal</span></div>
                    </div>
                    <div class="shortcut-section">
                        <h4>Actions</h4>
                        <div class="shortcut"><kbd>n</kbd> <span>New post</span></div>
                        <div class="shortcut"><kbd>l</kbd> <span>Like post</span></div>
                        <div class="shortcut"><kbd>r</kbd> <span>Reply to post</span></div>
                        <div class="shortcut"><kbd>b</kbd> <span>Bookmark post</span></div>
                    </div>
                    <div class="shortcut-section">
                        <h4>Go to</h4>
                        <div class="shortcut"><kbd>g</kbd> <kbd>h</kbd> <span>Home</span></div>
                        <div class="shortcut"><kbd>g</kbd> <kbd>e</kbd> <span>Explore</span></div>
                        <div class="shortcut"><kbd>g</kbd> <kbd>n</kbd> <span>Notifications</span></div>
                        <div class="shortcut"><kbd>g</kbd> <kbd>p</kbd> <span>Profile</span></div>
                        <div class="shortcut"><kbd>g</kbd> <kbd>s</kbd> <span>Settings</span></div>
                    </div>
                </div>
            `;
            modal.open(content, { className: 'modal-shortcuts' });
        }
    };

    // =========================================================================
    // Dropdown Menus
    // =========================================================================
    function initDropdowns() {
        document.addEventListener('click', (e) => {
            const trigger = e.target.closest('[data-dropdown]');

            if (trigger) {
                e.preventDefault();
                e.stopPropagation();

                const dropdown = document.getElementById(trigger.dataset.dropdown);
                if (!dropdown) return;

                // Close other dropdowns
                document.querySelectorAll('.dropdown-menu.show').forEach(d => {
                    if (d !== dropdown) d.classList.remove('show');
                });

                // Toggle this dropdown
                dropdown.classList.toggle('show');

                // Position dropdown
                const rect = trigger.getBoundingClientRect();
                dropdown.style.top = `${rect.bottom + 4}px`;
                dropdown.style.right = `${window.innerWidth - rect.right}px`;

                return;
            }

            // Close dropdowns when clicking outside
            document.querySelectorAll('.dropdown-menu.show').forEach(d => {
                d.classList.remove('show');
            });
        });
    }

    // =========================================================================
    // Tabs
    // =========================================================================
    function initTabs() {
        document.addEventListener('click', (e) => {
            const tab = e.target.closest('[data-tab]');
            if (!tab) return;

            e.preventDefault();

            const tabGroup = tab.closest('.tabs');
            const targetId = tab.dataset.tab;

            // Update tab states
            tabGroup.querySelectorAll('[data-tab]').forEach(t => {
                t.classList.toggle('active', t === tab);
                t.setAttribute('aria-selected', t === tab);
            });

            // Update tab panels
            const panel = document.getElementById(targetId);
            if (panel) {
                panel.parentElement.querySelectorAll('.tab-panel').forEach(p => {
                    p.classList.toggle('active', p === panel);
                });
            }
        });
    }

    // =========================================================================
    // Image Lightbox
    // =========================================================================
    function initLightbox() {
        document.addEventListener('click', (e) => {
            const img = e.target.closest('.post-media img, .lightbox-trigger');
            if (!img) return;

            e.preventDefault();

            const src = img.dataset.fullsize || img.src;
            const alt = img.alt || '';

            modal.lightbox(src, alt);
        });
    }

    // =========================================================================
    // Logout Handler
    // =========================================================================
    function initLogout() {
        document.addEventListener('click', async (e) => {
            const logoutBtn = e.target.closest('[data-action="logout"]');
            if (!logoutBtn) return;

            e.preventDefault();

            try {
                await api.post('/api/v1/auth/logout');
            } catch (err) {
                // Ignore errors, still logout locally
            }

            localStorage.removeItem('token');
            toast.success('Logged out successfully');
            setTimeout(() => window.location.href = '/', 500);
        });
    }

    // =========================================================================
    // Notification Badge
    // =========================================================================
    async function updateNotificationBadge() {
        const badge = document.querySelector('.notification-badge');
        if (!badge) return;

        try {
            const result = await api.get('/api/v1/notifications?limit=1');
            const unread = result.data?.filter(n => !n.read).length || 0;

            if (unread > 0) {
                badge.textContent = unread > 99 ? '99+' : unread;
                badge.style.display = 'flex';
            } else {
                badge.style.display = 'none';
            }
        } catch (err) {
            // Ignore errors
        }
    }

    // =========================================================================
    // Auto-resize Textareas
    // =========================================================================
    function initAutoResize() {
        document.addEventListener('input', (e) => {
            if (!e.target.matches('textarea[data-autoresize]')) return;

            const textarea = e.target;
            textarea.style.height = 'auto';
            textarea.style.height = textarea.scrollHeight + 'px';
        });
    }

    // =========================================================================
    // Mobile Navigation
    // =========================================================================
    function initMobileNav() {
        const menuToggle = document.querySelector('[data-action="toggle-menu"]');
        const sidebar = document.querySelector('.app-sidebar');

        if (!menuToggle || !sidebar) return;

        menuToggle.addEventListener('click', () => {
            sidebar.classList.toggle('show');
            document.body.classList.toggle('menu-open');
        });

        // Close on overlay click
        document.addEventListener('click', (e) => {
            if (sidebar.classList.contains('show') && !sidebar.contains(e.target) && !menuToggle.contains(e.target)) {
                sidebar.classList.remove('show');
                document.body.classList.remove('menu-open');
            }
        });
    }

    // =========================================================================
    // Compose Button (FAB on mobile and Sidebar)
    // =========================================================================
    function initComposeFAB() {
        const fab = document.querySelector('.compose-fab');
        if (fab) {
            fab.addEventListener('click', () => {
                modal.compose();
            });
        }
    }

    function initComposeButton() {
        const composeBtn = document.getElementById('composeBtn');
        if (!composeBtn) return;

        composeBtn.addEventListener('click', (e) => {
            e.preventDefault();

            // Check if there's an inline compose box on the page
            const composeInput = document.getElementById('composeInput');
            if (composeInput) {
                // Scroll to the compose box and focus it
                composeInput.scrollIntoView({ behavior: 'smooth', block: 'center' });
                setTimeout(() => {
                    composeInput.focus();
                }, 300);
            } else {
                // If no inline compose box, open modal
                modal.compose();
            }
        });
    }

    // =========================================================================
    // Initialize Everything
    // =========================================================================
    function init() {
        // Core systems
        toast.init();
        modal.init();
        theme.init();
        validation.init();
        shortcuts.init();

        // Features
        initCharCounter();
        initPostActions();
        initForms();
        initFollowButtons();
        initInfiniteScroll();
        initDropdowns();
        initTabs();
        initLightbox();
        initLogout();
        initAutoResize();
        initMobileNav();
        initComposeFAB();
        initComposeButton();

        // Periodic updates
        if (localStorage.getItem('token')) {
            updateNotificationBadge();
            setInterval(updateNotificationBadge, 60000); // Update every minute
        }

        console.log('Microblog app initialized');
    }

    // Start when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Expose some utilities globally for debugging
    window.microblog = { api, toast, modal, theme, shortcuts };
})();
