// Social - Client-side JavaScript

(function() {
    'use strict';

    // Interaction handlers
    document.addEventListener('click', async function(e) {
        const target = e.target.closest('.action');
        if (!target) return;

        const postId = target.dataset.id;
        if (!postId) return;

        if (target.classList.contains('like')) {
            await toggleLike(target, postId);
        } else if (target.classList.contains('repost')) {
            await toggleRepost(target, postId);
        } else if (target.classList.contains('bookmark')) {
            await toggleBookmark(target, postId);
        }
    });

    async function toggleLike(button, postId) {
        const isLiked = button.classList.contains('active');
        const method = isLiked ? 'DELETE' : 'POST';

        try {
            const res = await fetch(`/api/v1/posts/${postId}/like`, { method });
            if (res.ok) {
                const post = await res.json();
                button.classList.toggle('active');
                const countEl = button.querySelector('.count');
                if (countEl) countEl.textContent = post.likes_count || '';
            }
        } catch (err) {
            console.error('Like failed:', err);
        }
    }

    async function toggleRepost(button, postId) {
        const isReposted = button.classList.contains('active');
        const method = isReposted ? 'DELETE' : 'POST';

        try {
            const res = await fetch(`/api/v1/posts/${postId}/repost`, { method });
            if (res.ok) {
                const post = await res.json();
                button.classList.toggle('active');
                const countEl = button.querySelector('.count');
                if (countEl) countEl.textContent = post.reposts_count || '';
            }
        } catch (err) {
            console.error('Repost failed:', err);
        }
    }

    async function toggleBookmark(button, postId) {
        const isBookmarked = button.classList.contains('active');
        const method = isBookmarked ? 'DELETE' : 'POST';

        try {
            const res = await fetch(`/api/v1/posts/${postId}/bookmark`, { method });
            if (res.ok) {
                button.classList.toggle('active');
            }
        } catch (err) {
            console.error('Bookmark failed:', err);
        }
    }

    // Follow button handler
    document.addEventListener('click', async function(e) {
        const button = e.target.closest('#follow-btn');
        if (!button) return;

        const accountId = button.dataset.id;
        if (!accountId) return;

        const isFollowing = button.textContent === 'Unfollow';
        const endpoint = isFollowing ? 'unfollow' : 'follow';

        try {
            const res = await fetch(`/api/v1/accounts/${accountId}/${endpoint}`, { method: 'POST' });
            if (res.ok) {
                const rel = await res.json();
                if (rel.following) {
                    button.textContent = 'Unfollow';
                } else if (rel.requested) {
                    button.textContent = 'Requested';
                } else {
                    button.textContent = 'Follow';
                }
            }
        } catch (err) {
            console.error('Follow action failed:', err);
        }
    });

    // Character counter for compose
    document.addEventListener('input', function(e) {
        if (e.target.matches('.compose-box textarea')) {
            const maxLen = 500;
            const remaining = maxLen - e.target.value.length;
            const counter = document.querySelector('.char-count');
            if (counter) {
                counter.textContent = remaining;
                counter.style.color = remaining < 0 ? 'var(--danger)' : 'var(--text-secondary)';
            }
        }
    });

    // Load trending tags in sidebar
    async function loadTrendingTags() {
        const container = document.getElementById('trending-tags');
        if (!container) return;

        try {
            const res = await fetch('/api/v1/trends/tags?limit=5');
            const tags = await res.json();
            if (tags && tags.length > 0) {
                container.innerHTML = tags.map(t => `
                    <a href="/tags/${t.name}" class="trending-tag">
                        <span class="tag-name">#${t.name}</span>
                        <span class="tag-count">${t.posts_count} posts</span>
                    </a>
                `).join('');
            } else {
                container.innerHTML = '<p class="empty">No trends</p>';
            }
        } catch (err) {
            console.error('Failed to load trends:', err);
        }
    }

    // Initialize
    document.addEventListener('DOMContentLoaded', function() {
        loadTrendingTags();
    });
})();
