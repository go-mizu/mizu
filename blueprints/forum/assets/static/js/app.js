// Forum JavaScript - Modern UI Interactions

document.addEventListener('DOMContentLoaded', function() {
    // Vote handling
    document.addEventListener('click', function(e) {
        const voteBtn = e.target.closest('.vote-btn');
        if (!voteBtn) return;

        e.preventDefault();
        const action = voteBtn.dataset.action || (voteBtn.classList.contains('upvote') ? 'upvote' : 'downvote');
        const threadId = voteBtn.dataset.thread;
        const commentId = voteBtn.dataset.comment;

        if (threadId) {
            handleVote('thread', threadId, action === 'upvote' ? 1 : -1, voteBtn);
        } else if (commentId) {
            handleVote('comment', commentId, action === 'upvote' ? 1 : -1, voteBtn);
        }
    });

    // Bookmark handling
    document.addEventListener('click', function(e) {
        const bookmarkBtn = e.target.closest('.bookmark-btn, [data-action="bookmark"]');
        if (!bookmarkBtn) return;

        e.preventDefault();
        const threadId = bookmarkBtn.dataset.thread;
        const commentId = bookmarkBtn.dataset.comment;
        const isBookmarked = bookmarkBtn.classList.contains('active');

        if (threadId) {
            handleBookmark('thread', threadId, !isBookmarked, bookmarkBtn);
        } else if (commentId) {
            handleBookmark('comment', commentId, !isBookmarked, bookmarkBtn);
        }
    });

    // Share button
    document.addEventListener('click', function(e) {
        const shareBtn = e.target.closest('.share-btn, [data-action="share"]');
        if (!shareBtn) return;

        e.preventDefault();
        const url = shareBtn.dataset.url || window.location.href;

        if (navigator.share) {
            navigator.share({ url: url }).catch(() => {});
        } else if (navigator.clipboard) {
            navigator.clipboard.writeText(window.location.origin + url)
                .then(() => {
                    showToast('Link copied to clipboard');
                });
        }
    });

    // Join/Leave board
    document.addEventListener('click', function(e) {
        const btn = e.target.closest('[data-action="join"], [data-action="leave"]');
        if (!btn) return;

        e.preventDefault();
        const board = btn.dataset.board;
        const action = btn.dataset.action;

        handleBoardMembership(board, action, btn);
    });

    // Comment reply toggle
    document.addEventListener('click', function(e) {
        const replyBtn = e.target.closest('[data-action="reply"]');
        if (!replyBtn) return;

        e.preventDefault();
        const commentId = replyBtn.dataset.comment;
        const form = document.getElementById('reply-form-' + commentId);
        if (form) {
            // Hide other reply forms
            document.querySelectorAll('.reply-form').forEach(f => {
                if (f.id !== 'reply-form-' + commentId) {
                    f.classList.add('hidden');
                }
            });
            form.classList.toggle('hidden');
            if (!form.classList.contains('hidden')) {
                form.querySelector('textarea').focus();
            }
        }
    });

    // Cancel reply
    document.addEventListener('click', function(e) {
        const cancelBtn = e.target.closest('[data-action="cancel-reply"]');
        if (!cancelBtn) return;

        e.preventDefault();
        const form = cancelBtn.closest('.reply-form');
        if (form) {
            form.classList.add('hidden');
            form.querySelector('textarea').value = '';
        }
    });

    // Comment collapse/expand
    document.addEventListener('click', function(e) {
        const collapseLine = e.target.closest('.comment-collapse-line, [data-action="toggle-collapse"]');
        if (!collapseLine) return;

        e.preventDefault();
        const comment = collapseLine.closest('.comment');
        if (comment) {
            comment.classList.toggle('collapsed');
        }
    });

    // Expand collapsed comment
    document.addEventListener('click', function(e) {
        const expandBtn = e.target.closest('[data-action="expand"]');
        if (!expandBtn) return;

        e.preventDefault();
        const comment = expandBtn.closest('.comment');
        if (comment) {
            comment.classList.remove('collapsed');
        }
    });

    // Form submissions
    document.addEventListener('submit', function(e) {
        const form = e.target;
        const action = form.dataset.action;

        if (!action) return;

        e.preventDefault();

        switch (action) {
            case 'login':
                handleLogin(form);
                break;
            case 'register':
                handleRegister(form);
                break;
            case 'submit-thread':
                handleSubmitThread(form);
                break;
            case 'submit-comment':
                handleSubmitComment(form);
                break;
            case 'submit-reply':
                handleSubmitReply(form);
                break;
            case 'update-profile':
                handleUpdateProfile(form);
                break;
            case 'update-password':
                handleUpdatePassword(form);
                break;
        }
    });

    // Post type tabs
    document.addEventListener('click', function(e) {
        const typeTab = e.target.closest('.type-tab');
        if (!typeTab) return;

        e.preventDefault();
        const type = typeTab.dataset.type;
        const tabs = document.querySelectorAll('.type-tab');
        const textInput = document.querySelector('.text-input');
        const linkInput = document.querySelector('.link-input');

        tabs.forEach(t => t.classList.remove('active'));
        typeTab.classList.add('active');

        if (type === 'link') {
            if (textInput) textInput.classList.add('hidden');
            if (linkInput) linkInput.classList.remove('hidden');
        } else {
            if (textInput) textInput.classList.remove('hidden');
            if (linkInput) linkInput.classList.add('hidden');
        }
    });

    // Comment sort dropdown
    document.addEventListener('change', function(e) {
        const select = e.target.closest('[data-action="sort-comments"]');
        if (!select) return;

        const url = new URL(window.location);
        url.searchParams.set('sort', select.value);
        window.location = url.toString();
    });

    // Mark all notifications as read
    document.addEventListener('click', function(e) {
        const btn = e.target.closest('[data-action="mark-all-read"]');
        if (!btn) return;

        e.preventDefault();
        handleMarkAllRead(btn);
    });

    // Delete account confirmation
    document.addEventListener('click', function(e) {
        const btn = e.target.closest('[data-action="delete-account"]');
        if (!btn) return;

        e.preventDefault();
        if (confirm('Are you sure you want to delete your account? This cannot be undone.')) {
            handleDeleteAccount();
        }
    });
});

// Vote handler
async function handleVote(type, id, value, btn) {
    const endpoint = type === 'thread' ? `/api/threads/${id}/vote` : `/api/comments/${id}/vote`;
    const isActive = btn.classList.contains('active');

    try {
        if (isActive) {
            await fetch(endpoint, { method: 'DELETE' });
        } else {
            await fetch(endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ value: value })
            });
        }

        // Find vote container (could be vote-column or vote-inline)
        const container = btn.closest('.vote-column, .vote-inline');
        if (!container) return;

        const upBtn = container.querySelector('.upvote');
        const downBtn = container.querySelector('.downvote');
        const scoreEl = container.querySelector('.vote-score');

        // Remove active state from both
        if (upBtn) upBtn.classList.remove('active');
        if (downBtn) downBtn.classList.remove('active');

        // Update score
        if (scoreEl) {
            let score = parseInt(scoreEl.textContent.replace(/[^-\d]/g, '')) || 0;

            // Calculate new score based on previous state and new action
            if (isActive) {
                score -= value;
            } else {
                // If clicking opposite vote, need to remove old vote too
                const wasUpvoted = upBtn && upBtn === btn ? false : (upBtn && upBtn.classList.contains('active'));
                const wasDownvoted = downBtn && downBtn === btn ? false : (downBtn && downBtn.classList.contains('active'));

                if (wasUpvoted && value === -1) score -= 2;
                else if (wasDownvoted && value === 1) score += 2;
                else score += value;
            }

            scoreEl.textContent = formatScore(score);
        }

        // Add active state to clicked button if it wasn't already active
        if (!isActive) {
            btn.classList.add('active');
        }
    } catch (err) {
        console.error('Vote failed:', err);
    }
}

// Bookmark handler
async function handleBookmark(type, id, save, btn) {
    const endpoint = type === 'thread' ? `/api/threads/${id}/bookmark` : `/api/comments/${id}/bookmark`;

    try {
        await fetch(endpoint, { method: save ? 'POST' : 'DELETE' });

        btn.classList.toggle('active', save);

        // Update SVG fill
        const svg = btn.querySelector('svg');
        if (svg) {
            svg.setAttribute('fill', save ? 'currentColor' : 'none');
        }

        // Update text
        const textNode = Array.from(btn.childNodes).find(n => n.nodeType === Node.TEXT_NODE);
        if (textNode) {
            textNode.textContent = save ? 'Saved' : 'Save';
        } else {
            // Text might be inside the button after SVG
            btn.innerHTML = btn.innerHTML.replace(save ? 'Save' : 'Saved', save ? 'Saved' : 'Save');
        }
    } catch (err) {
        console.error('Bookmark failed:', err);
    }
}

// Board membership handler
async function handleBoardMembership(board, action, btn) {
    const endpoint = `/api/boards/${board}/join`;

    try {
        await fetch(endpoint, { method: action === 'join' ? 'POST' : 'DELETE' });

        if (action === 'join') {
            btn.textContent = 'Joined';
            btn.dataset.action = 'leave';
            btn.classList.remove('btn-primary');
            btn.classList.add('btn-secondary');
        } else {
            btn.textContent = 'Join';
            btn.dataset.action = 'join';
            btn.classList.remove('btn-secondary');
            btn.classList.add('btn-primary');
        }
    } catch (err) {
        console.error('Membership failed:', err);
    }
}

// Auth handlers
async function handleLogin(form) {
    const data = new FormData(form);
    const body = {
        username: data.get('username'),
        password: data.get('password')
    };

    try {
        const res = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });

        const json = await res.json();

        if (!res.ok || json.error) {
            showFormError(form, json.error?.message || 'Login failed');
        } else {
            window.location.href = '/';
        }
    } catch (err) {
        showFormError(form, 'An error occurred');
    }
}

async function handleRegister(form) {
    const data = new FormData(form);
    const body = {
        username: data.get('username'),
        email: data.get('email'),
        password: data.get('password')
    };

    try {
        const res = await fetch('/api/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });

        const json = await res.json();

        if (!res.ok || json.error) {
            showFormError(form, json.error?.message || 'Registration failed');
        } else {
            window.location.href = '/';
        }
    } catch (err) {
        showFormError(form, 'An error occurred');
    }
}

// Thread submission
async function handleSubmitThread(form) {
    const data = new FormData(form);
    const board = form.action.split('/boards/')[1]?.split('/')[0];

    const body = {
        title: data.get('title'),
        content: data.get('content') || '',
        url: data.get('url') || '',
        is_nsfw: data.get('is_nsfw') === 'true',
        is_spoiler: data.get('is_spoiler') === 'true'
    };

    // Determine type based on which tab is active
    const activeTab = document.querySelector('.type-tab.active');
    body.type = activeTab?.dataset.type || (body.url ? 'link' : 'text');

    try {
        const res = await fetch(form.action, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });

        const json = await res.json();

        if (!res.ok || json.error) {
            showFormError(form, json.error?.message || 'Failed to create post');
        } else {
            window.location.href = '/b/' + board;
        }
    } catch (err) {
        showFormError(form, 'An error occurred');
    }
}

// Comment submission
async function handleSubmitComment(form) {
    const data = new FormData(form);
    const body = { content: data.get('content') };

    try {
        const res = await fetch(form.action, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });

        const json = await res.json();

        if (!res.ok || json.error) {
            showFormError(form, json.error?.message || 'Failed to post comment');
        } else {
            window.location.reload();
        }
    } catch (err) {
        showFormError(form, 'An error occurred');
    }
}

// Reply submission
async function handleSubmitReply(form) {
    const parentId = form.dataset.parent;
    const content = form.querySelector('textarea').value;

    // Extract thread ID from URL
    const pathParts = window.location.pathname.split('/');
    const threadId = pathParts[3]; // /b/boardname/threadid/...

    const body = {
        content: content,
        parent_id: parentId
    };

    try {
        const res = await fetch(`/api/threads/${threadId}/comments`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });

        const json = await res.json();

        if (!res.ok || json.error) {
            showToast(json.error?.message || 'Failed to post reply', 'error');
        } else {
            window.location.reload();
        }
    } catch (err) {
        showToast('An error occurred', 'error');
    }
}

// Profile update
async function handleUpdateProfile(form) {
    const data = new FormData(form);
    const body = {
        display_name: data.get('display_name'),
        bio: data.get('bio'),
        avatar_url: data.get('avatar_url')
    };

    try {
        const res = await fetch('/api/auth/me', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });

        if (res.ok) {
            showToast('Profile updated successfully');
        } else {
            const json = await res.json();
            showToast(json.error?.message || 'Failed to update profile', 'error');
        }
    } catch (err) {
        showToast('An error occurred', 'error');
    }
}

// Password update
async function handleUpdatePassword(form) {
    const data = new FormData(form);

    if (data.get('new_password') !== data.get('confirm_password')) {
        showFormError(form, 'Passwords do not match');
        return;
    }

    const body = {
        current_password: data.get('current_password'),
        new_password: data.get('new_password')
    };

    try {
        const res = await fetch('/api/auth/password', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });

        if (res.ok) {
            form.reset();
            showToast('Password updated successfully');
        } else {
            const json = await res.json();
            showFormError(form, json.error?.message || 'Failed to update password');
        }
    } catch (err) {
        showFormError(form, 'An error occurred');
    }
}

// Mark all notifications as read
async function handleMarkAllRead(btn) {
    try {
        await fetch('/api/notifications/read', { method: 'POST' });

        document.querySelectorAll('.notification.unread').forEach(n => {
            n.classList.remove('unread');
        });

        btn.remove();
    } catch (err) {
        console.error('Failed to mark notifications as read:', err);
    }
}

// Delete account
async function handleDeleteAccount() {
    try {
        const res = await fetch('/api/auth/me', { method: 'DELETE' });

        if (res.ok) {
            window.location.href = '/';
        } else {
            showToast('Failed to delete account', 'error');
        }
    } catch (err) {
        showToast('An error occurred', 'error');
    }
}

// UI Helpers
function showFormError(form, message) {
    let errorEl = form.querySelector('.form-error');
    if (!errorEl) {
        errorEl = document.createElement('div');
        errorEl.className = 'form-error';
        const submitBtn = form.querySelector('button[type="submit"]');
        if (submitBtn) {
            submitBtn.parentNode.insertBefore(errorEl, submitBtn);
        } else {
            form.appendChild(errorEl);
        }
    }
    errorEl.textContent = message;
    errorEl.classList.remove('hidden');
}

function showToast(message, type = 'success') {
    // Create toast element if it doesn't exist
    let toast = document.getElementById('toast');
    if (!toast) {
        toast = document.createElement('div');
        toast.id = 'toast';
        toast.style.cssText = `
            position: fixed;
            bottom: 20px;
            left: 50%;
            transform: translateX(-50%);
            padding: 12px 24px;
            border-radius: 8px;
            color: white;
            font-size: 14px;
            font-weight: 500;
            z-index: 1000;
            opacity: 0;
            transition: opacity 0.3s ease;
        `;
        document.body.appendChild(toast);
    }

    toast.textContent = message;
    toast.style.background = type === 'error' ? '#ef4444' : '#22c55e';
    toast.style.opacity = '1';

    setTimeout(() => {
        toast.style.opacity = '0';
    }, 3000);
}

function formatScore(score) {
    if (Math.abs(score) >= 1000000) {
        return (score / 1000000).toFixed(1).replace(/\.0$/, '') + 'm';
    }
    if (Math.abs(score) >= 1000) {
        return (score / 1000).toFixed(1).replace(/\.0$/, '') + 'k';
    }
    return score.toString();
}
