// Social - TikTok-Style Client-side JavaScript

(function() {
    'use strict';

    // ========================================
    // Theme Management
    // ========================================

    function initTheme() {
        const savedTheme = localStorage.getItem('theme');
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        const theme = savedTheme || (prefersDark ? 'dark' : 'light');
        setTheme(theme);
    }

    function setTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('theme', theme);

        // Update toggle icons
        const sunIcon = document.querySelector('.sun-icon');
        const moonIcon = document.querySelector('.moon-icon');
        const desktopToggle = document.getElementById('theme-toggle-desktop');

        if (sunIcon && moonIcon) {
            if (theme === 'dark') {
                sunIcon.classList.add('hidden');
                moonIcon.classList.remove('hidden');
            } else {
                sunIcon.classList.remove('hidden');
                moonIcon.classList.add('hidden');
            }
        }

        if (desktopToggle) {
            desktopToggle.checked = theme === 'dark';
        }

        // Update meta theme-color
        const metaTheme = document.querySelector('meta[name="theme-color"]');
        if (metaTheme) {
            metaTheme.content = theme === 'dark' ? '#000000' : '#ffffff';
        }
    }

    function toggleTheme() {
        const current = document.documentElement.getAttribute('data-theme');
        const newTheme = current === 'dark' ? 'light' : 'dark';
        setTheme(newTheme);
    }

    // ========================================
    // Compose Modal
    // ========================================

    function initComposeModal() {
        const modal = document.getElementById('compose-modal');
        const closeBtn = document.getElementById('compose-close');
        const mobileBtn = document.getElementById('compose-mobile');
        const desktopBtn = document.getElementById('compose-desktop');
        const form = document.getElementById('compose-form');
        const textarea = form?.querySelector('textarea');
        const charCount = document.getElementById('modal-char-count');

        function openModal() {
            if (modal) {
                modal.classList.add('active');
                textarea?.focus();
            }
        }

        function closeModal() {
            if (modal) {
                modal.classList.remove('active');
            }
        }

        if (mobileBtn) mobileBtn.addEventListener('click', openModal);
        if (desktopBtn) desktopBtn.addEventListener('click', openModal);
        if (closeBtn) closeBtn.addEventListener('click', closeModal);

        // Close on backdrop click
        modal?.addEventListener('click', function(e) {
            if (e.target === modal) closeModal();
        });

        // Close on Escape
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape' && modal?.classList.contains('active')) {
                closeModal();
            }
        });

        // Character counter
        textarea?.addEventListener('input', function() {
            const remaining = 500 - this.value.length;
            if (charCount) {
                charCount.textContent = remaining;
                charCount.className = 'char-count';
                if (remaining < 50) charCount.classList.add('warning');
                if (remaining < 0) charCount.classList.add('danger');
            }
        });

        // Form submission
        form?.addEventListener('submit', async function(e) {
            e.preventDefault();
            const content = textarea?.value.trim();
            if (!content) return;

            const submitBtn = this.querySelector('button[type="submit"]');
            if (submitBtn) {
                submitBtn.disabled = true;
                submitBtn.textContent = 'Posting...';
            }

            try {
                const res = await fetch('/api/v1/posts', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({content})
                });
                if (res.ok) {
                    if (textarea) textarea.value = '';
                    if (charCount) charCount.textContent = '500';
                    closeModal();
                    // Reload timeline if on home page
                    if (typeof loadTimeline === 'function') {
                        loadTimeline();
                    } else {
                        window.location.reload();
                    }
                }
            } catch (err) {
                console.error('Failed to post:', err);
            } finally {
                if (submitBtn) {
                    submitBtn.disabled = false;
                    submitBtn.textContent = 'Post';
                }
            }
        });
    }

    // ========================================
    // Feed Tabs
    // ========================================

    function initFeedTabs() {
        const tabs = document.querySelectorAll('.feed-tab');
        tabs.forEach(tab => {
            tab.addEventListener('click', function() {
                tabs.forEach(t => t.classList.remove('active'));
                this.classList.add('active');
                const tabType = this.dataset.tab;
                if (typeof loadTimeline === 'function') {
                    loadTimeline(tabType);
                }
            });
        });
    }

    // ========================================
    // Post Interactions
    // ========================================

    document.addEventListener('click', async function(e) {
        const btn = e.target.closest('.action-btn');
        if (!btn) return;

        const postId = btn.dataset.id;
        if (!postId) return;

        e.preventDefault();
        e.stopPropagation();

        if (btn.classList.contains('like')) {
            await toggleLike(btn, postId);
        } else if (btn.classList.contains('repost')) {
            await toggleRepost(btn, postId);
        } else if (btn.classList.contains('bookmark')) {
            await toggleBookmark(btn, postId);
        }
    });

    async function toggleLike(button, postId) {
        const isLiked = button.classList.contains('active');
        const method = isLiked ? 'DELETE' : 'POST';
        const endpoint = isLiked ? 'unlike' : 'like';

        try {
            const res = await fetch(`/api/v1/posts/${postId}/${endpoint}`, { method: 'POST' });
            if (res.ok) {
                button.classList.toggle('active');
                const countEl = button.querySelector('.count');
                const svg = button.querySelector('svg');
                if (countEl) {
                    const count = parseInt(countEl.textContent) || 0;
                    countEl.textContent = isLiked ? (count - 1 || '') : count + 1;
                }
                if (svg) {
                    svg.setAttribute('fill', isLiked ? 'none' : 'currentColor');
                }
            }
        } catch (err) {
            console.error('Like failed:', err);
        }
    }

    async function toggleRepost(button, postId) {
        const isReposted = button.classList.contains('active');
        const endpoint = isReposted ? 'unrepost' : 'repost';

        try {
            const res = await fetch(`/api/v1/posts/${postId}/${endpoint}`, { method: 'POST' });
            if (res.ok) {
                button.classList.toggle('active');
                const countEl = button.querySelector('.count');
                if (countEl) {
                    const count = parseInt(countEl.textContent) || 0;
                    countEl.textContent = isReposted ? (count - 1 || '') : count + 1;
                }
            }
        } catch (err) {
            console.error('Repost failed:', err);
        }
    }

    async function toggleBookmark(button, postId) {
        const isBookmarked = button.classList.contains('active');
        const endpoint = isBookmarked ? 'unbookmark' : 'bookmark';

        try {
            const res = await fetch(`/api/v1/posts/${postId}/${endpoint}`, { method: 'POST' });
            if (res.ok) {
                button.classList.toggle('active');
                const svg = button.querySelector('svg');
                if (svg) {
                    svg.setAttribute('fill', isBookmarked ? 'none' : 'currentColor');
                }
            }
        } catch (err) {
            console.error('Bookmark failed:', err);
        }
    }

    // ========================================
    // Follow Button
    // ========================================

    document.addEventListener('click', async function(e) {
        const button = e.target.closest('.btn-follow, .follow-chip');
        if (!button) return;

        const accountId = button.dataset.id;
        if (!accountId) return;

        const isFollowing = button.classList.contains('following');
        const endpoint = isFollowing ? 'unfollow' : 'follow';

        button.disabled = true;

        try {
            const res = await fetch(`/api/v1/accounts/${accountId}/${endpoint}`, { method: 'POST' });
            if (res.ok) {
                const rel = await res.json();
                button.classList.toggle('following', rel.following);
                if (rel.following) {
                    button.textContent = 'Following';
                } else if (rel.requested) {
                    button.textContent = 'Requested';
                } else {
                    button.textContent = 'Follow';
                }
            }
        } catch (err) {
            console.error('Follow action failed:', err);
        } finally {
            button.disabled = false;
        }
    });

    // ========================================
    // Trending Tags
    // ========================================

    async function loadTrendingTags() {
        const container = document.getElementById('trending-tags');
        if (!container) return;

        try {
            const res = await fetch('/api/v1/trends/tags?limit=5');
            const tags = await res.json();
            if (tags && tags.length > 0) {
                container.innerHTML = tags.map(t => `
                    <a href="/tags/${t.name}" class="trending-item">
                        <span class="tag-name">#${t.name}</span>
                        <span class="tag-count">${formatCount(t.posts_count)} posts</span>
                    </a>
                `).join('');
            } else {
                container.innerHTML = '<p style="color: var(--text-tertiary); font-size: var(--text-sm);">No trends yet</p>';
            }
        } catch (err) {
            console.error('Failed to load trends:', err);
        }
    }

    // ========================================
    // Utility Functions
    // ========================================

    function formatCount(num) {
        if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
        if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
        return num.toString();
    }

    window.formatRelative = function(dateStr) {
        const date = new Date(dateStr);
        const now = new Date();
        const diff = now - date;
        const mins = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (mins < 1) return 'now';
        if (mins < 60) return mins + 'm';
        if (hours < 24) return hours + 'h';
        if (days < 7) return days + 'd';
        return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    };

    window.formatContent = function(text) {
        if (!text) return '';
        let html = escapeHtml(text);
        // Highlight hashtags
        html = html.replace(/#(\w+)/g, '<a href="/tags/$1" class="hashtag" onclick="event.stopPropagation()">#$1</a>');
        // Highlight mentions
        html = html.replace(/@(\w+)/g, '<a href="/@$1" class="mention" onclick="event.stopPropagation()">@$1</a>');
        return html;
    };

    window.escapeHtml = function(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    };

    // ========================================
    // Initialization
    // ========================================

    document.addEventListener('DOMContentLoaded', function() {
        initTheme();
        initComposeModal();
        initFeedTabs();
        loadTrendingTags();

        // Theme toggle buttons
        const themeToggle = document.getElementById('theme-toggle');
        const themeToggleDesktop = document.getElementById('theme-toggle-desktop');

        themeToggle?.addEventListener('click', toggleTheme);
        themeToggleDesktop?.addEventListener('change', toggleTheme);
    });

})();
