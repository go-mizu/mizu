// Microblog - Client-side JavaScript

(function() {
    'use strict';

    // API helper
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
            const data = await response.json();
            if (!response.ok) {
                throw new Error(data.error?.message || 'Request failed');
            }
            return data;
        },
        get(url) { return this.fetch(url); },
        post(url, body) { return this.fetch(url, { method: 'POST', body: JSON.stringify(body) }); },
        put(url, body) { return this.fetch(url, { method: 'PUT', body: JSON.stringify(body) }); },
        delete(url) { return this.fetch(url, { method: 'DELETE' }); },
    };

    // Character counter for compose box
    function initCharCounter() {
        const textarea = document.querySelector('textarea[name="content"]');
        const counter = document.getElementById('charCount');
        if (textarea && counter) {
            textarea.addEventListener('input', () => {
                const count = [...textarea.value].length;
                counter.textContent = count;
                counter.classList.toggle('text-red-600', count > 500);
            });
        }
    }

    // Post actions (like, repost, bookmark)
    function initPostActions() {
        document.addEventListener('click', async (e) => {
            const button = e.target.closest('[data-action]');
            if (!button) return;

            const action = button.dataset.action;
            const postId = button.dataset.postId;
            if (!postId) return;

            e.preventDefault();

            try {
                switch (action) {
                    case 'like':
                        if (button.classList.contains('text-red-600')) {
                            await api.delete(`/api/v1/posts/${postId}/like`);
                            button.classList.remove('text-red-600');
                        } else {
                            await api.post(`/api/v1/posts/${postId}/like`);
                            button.classList.add('text-red-600');
                        }
                        break;

                    case 'repost':
                        if (button.classList.contains('text-green-600')) {
                            await api.delete(`/api/v1/posts/${postId}/repost`);
                            button.classList.remove('text-green-600');
                        } else {
                            await api.post(`/api/v1/posts/${postId}/repost`);
                            button.classList.add('text-green-600');
                        }
                        break;

                    case 'bookmark':
                        if (button.classList.contains('text-blue-600')) {
                            await api.delete(`/api/v1/posts/${postId}/bookmark`);
                            button.classList.remove('text-blue-600');
                        } else {
                            await api.post(`/api/v1/posts/${postId}/bookmark`);
                            button.classList.add('text-blue-600');
                        }
                        break;
                }
            } catch (err) {
                console.error('Action failed:', err.message);
                // Could show a toast notification here
            }
        });
    }

    // Form submission
    function initForms() {
        // Compose form
        const composeForm = document.querySelector('form[action="/api/v1/posts"]');
        if (composeForm) {
            composeForm.addEventListener('submit', async (e) => {
                e.preventDefault();

                const textarea = composeForm.querySelector('textarea[name="content"]');
                const content = textarea.value.trim();

                if (!content) return;

                try {
                    const result = await api.post('/api/v1/posts', { content });
                    textarea.value = '';
                    document.getElementById('charCount').textContent = '0';

                    // Prepend new post to timeline
                    const timeline = document.getElementById('timeline');
                    if (timeline && result.data) {
                        // Reload page to show new post (simple approach)
                        window.location.reload();
                    }
                } catch (err) {
                    alert('Failed to create post: ' + err.message);
                }
            });
        }

        // Login form
        const loginForm = document.querySelector('form[action="/login"]');
        if (loginForm) {
            loginForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                const formData = new FormData(loginForm);
                try {
                    const result = await api.post('/api/v1/auth/login', {
                        username: formData.get('username'),
                        password: formData.get('password'),
                    });
                    localStorage.setItem('token', result.data.token);
                    window.location.href = '/';
                } catch (err) {
                    alert('Login failed: ' + err.message);
                }
            });
        }

        // Register form
        const registerForm = document.querySelector('form[action="/register"]');
        if (registerForm) {
            registerForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                const formData = new FormData(registerForm);
                try {
                    const result = await api.post('/api/v1/auth/register', {
                        username: formData.get('username'),
                        email: formData.get('email'),
                        password: formData.get('password'),
                    });
                    localStorage.setItem('token', result.data.token);
                    window.location.href = '/';
                } catch (err) {
                    alert('Registration failed: ' + err.message);
                }
            });
        }
    }

    // Follow/unfollow buttons
    function initFollowButtons() {
        document.addEventListener('click', async (e) => {
            const button = e.target.closest('[data-follow]');
            if (!button) return;

            const accountId = button.dataset.follow;
            const isFollowing = button.dataset.following === 'true';

            try {
                if (isFollowing) {
                    await api.post(`/api/v1/accounts/${accountId}/unfollow`);
                    button.textContent = 'Follow';
                    button.dataset.following = 'false';
                } else {
                    await api.post(`/api/v1/accounts/${accountId}/follow`);
                    button.textContent = 'Unfollow';
                    button.dataset.following = 'true';
                }
            } catch (err) {
                alert('Action failed: ' + err.message);
            }
        });
    }

    // Infinite scroll for timeline
    function initInfiniteScroll() {
        const timeline = document.getElementById('timeline');
        if (!timeline) return;

        let loading = false;
        let maxId = null;

        // Get the last post's ID
        const lastPost = timeline.querySelector('article:last-child');
        if (lastPost) {
            maxId = lastPost.dataset.postId;
        }

        window.addEventListener('scroll', async () => {
            if (loading) return;

            const scrollBottom = window.innerHeight + window.scrollY >= document.body.offsetHeight - 500;
            if (!scrollBottom) return;

            loading = true;

            try {
                const url = new URL(window.location.href);
                url.pathname = '/api/v1/timelines/home';
                if (maxId) url.searchParams.set('max_id', maxId);

                const result = await api.get(url.toString());
                if (result.data && result.data.length > 0) {
                    // Append posts (would need server-side rendering or client-side template)
                    maxId = result.data[result.data.length - 1].id;
                }
            } catch (err) {
                console.error('Failed to load more posts:', err);
            }

            loading = false;
        });
    }

    // Initialize on DOM ready
    document.addEventListener('DOMContentLoaded', () => {
        initCharCounter();
        initPostActions();
        initForms();
        initFollowButtons();
        initInfiniteScroll();
    });
})();
