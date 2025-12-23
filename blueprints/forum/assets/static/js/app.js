// Forum JavaScript

document.addEventListener('DOMContentLoaded', function() {
    // Vote handling
    document.addEventListener('click', function(e) {
        const voteBtn = e.target.closest('.vote-btn');
        if (!voteBtn) return;

        const action = voteBtn.dataset.action;
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
        const bookmarkBtn = e.target.closest('.bookmark-btn');
        if (!bookmarkBtn) return;

        const threadId = bookmarkBtn.dataset.thread;
        const commentId = bookmarkBtn.dataset.comment;
        const isBookmarked = bookmarkBtn.classList.contains('active');

        if (threadId) {
            handleBookmark('thread', threadId, !isBookmarked, bookmarkBtn);
        } else if (commentId) {
            handleBookmark('comment', commentId, !isBookmarked, bookmarkBtn);
        }
    });

    // Join/Leave board
    document.addEventListener('click', function(e) {
        const btn = e.target.closest('[data-action="join"], [data-action="leave"]');
        if (!btn) return;

        const board = btn.dataset.board;
        const action = btn.dataset.action;

        handleBoardMembership(board, action, btn);
    });

    // Comment reply
    document.addEventListener('click', function(e) {
        const replyBtn = e.target.closest('[data-action="reply"]');
        if (!replyBtn) return;

        const commentId = replyBtn.dataset.comment;
        const form = document.getElementById('reply-form-' + commentId);
        if (form) {
            form.classList.toggle('hidden');
        }
    });

    document.addEventListener('click', function(e) {
        const cancelBtn = e.target.closest('[data-action="cancel-reply"]');
        if (!cancelBtn) return;

        const form = cancelBtn.closest('.reply-form');
        if (form) {
            form.classList.add('hidden');
        }
    });

    // Comment collapse
    document.addEventListener('click', function(e) {
        const collapseLine = e.target.closest('.collapse-line');
        if (!collapseLine) return;

        const comment = collapseLine.closest('.comment');
        if (comment) {
            comment.classList.toggle('collapsed');
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
        }
    });

    // Post type tabs
    document.addEventListener('click', function(e) {
        const typeTab = e.target.closest('.type-tab');
        if (!typeTab) return;

        const type = typeTab.dataset.type;
        const tabs = document.querySelectorAll('.type-tab');
        const textInput = document.querySelector('.text-input');
        const linkInput = document.querySelector('.link-input');

        tabs.forEach(t => t.classList.remove('active'));
        typeTab.classList.add('active');

        if (type === 'link') {
            textInput.classList.add('hidden');
            linkInput.classList.remove('hidden');
        } else {
            textInput.classList.remove('hidden');
            linkInput.classList.add('hidden');
        }
    });
});

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

        // Toggle state
        const container = btn.closest('.vote-buttons, .vote-buttons-inline');
        const upBtn = container.querySelector('.upvote');
        const downBtn = container.querySelector('.downvote');
        const scoreEl = container.querySelector('.vote-score');

        upBtn.classList.remove('active');
        downBtn.classList.remove('active');

        if (!isActive) {
            btn.classList.add('active');
        }

        // Update score display (simplified)
        if (scoreEl) {
            let score = parseInt(scoreEl.textContent) || 0;
            if (isActive) {
                score -= value;
            } else {
                score += value;
            }
            scoreEl.textContent = score;
        }
    } catch (err) {
        console.error('Vote failed:', err);
    }
}

async function handleBookmark(type, id, save, btn) {
    const endpoint = type === 'thread' ? `/api/threads/${id}/bookmark` : `/api/comments/${id}/bookmark`;

    try {
        await fetch(endpoint, { method: save ? 'POST' : 'DELETE' });

        btn.classList.toggle('active');
        const icon = btn.querySelector('.icon');
        if (icon) {
            icon.textContent = save ? '★' : '☆';
        }
        btn.childNodes[btn.childNodes.length - 1].textContent = save ? 'Saved' : 'Save';
    } catch (err) {
        console.error('Bookmark failed:', err);
    }
}

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

        if (json.error) {
            showFormError(form, json.error.message);
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

        if (json.error) {
            showFormError(form, json.error.message);
        } else {
            window.location.href = '/';
        }
    } catch (err) {
        showFormError(form, 'An error occurred');
    }
}

async function handleSubmitThread(form) {
    const data = new FormData(form);
    const board = form.action.split('/')[4]; // Extract board from URL

    const body = {
        title: data.get('title'),
        content: data.get('content'),
        url: data.get('url'),
        is_nsfw: data.get('is_nsfw') === 'true',
        is_spoiler: data.get('is_spoiler') === 'true'
    };

    // Determine type
    if (body.url) {
        body.type = 'link';
    } else {
        body.type = 'text';
    }

    try {
        const res = await fetch(form.action, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });

        const json = await res.json();

        if (json.error) {
            showFormError(form, json.error.message);
        } else {
            window.location.href = '/b/' + board;
        }
    } catch (err) {
        showFormError(form, 'An error occurred');
    }
}

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

        if (json.error) {
            showFormError(form, json.error.message);
        } else {
            window.location.reload();
        }
    } catch (err) {
        showFormError(form, 'An error occurred');
    }
}

async function handleSubmitReply(form) {
    const parentId = form.dataset.parent;
    const content = form.querySelector('textarea').value;
    const threadId = window.location.pathname.split('/')[3];

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

        if (json.error) {
            alert(json.error.message);
        } else {
            window.location.reload();
        }
    } catch (err) {
        alert('An error occurred');
    }
}

function showFormError(form, message) {
    let errorEl = form.querySelector('.form-error');
    if (!errorEl) {
        errorEl = document.createElement('div');
        errorEl.className = 'form-error';
        form.insertBefore(errorEl, form.querySelector('button[type="submit"]'));
    }
    errorEl.textContent = message;
    errorEl.classList.remove('hidden');
}
