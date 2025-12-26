// Messaging App JavaScript

// Available themes
// 'default' themes (dark/light) use CSS variables only
// View themes use completely different view directories
const THEMES = ['dark', 'light', 'aim1.0', 'ymxp', 'im26', 'imos9', 'imosx'];
const VIEW_THEMES = ['aim1.0', 'ymxp', 'im26', 'imos9', 'imosx']; // Themes that require different server-side views

// Theme handling - set data-theme attribute on page load
(function() {
    const theme = localStorage.getItem('theme') || 'dark';
    document.documentElement.setAttribute('data-theme', theme);
})();

// Get current theme
function getTheme() {
    return localStorage.getItem('theme') || 'dark';
}

// Set cookie for server-side theme detection
function setThemeCookie(themeName) {
    // Map dark/light to 'default' for server-side, aim1.0 stays as is
    const serverTheme = VIEW_THEMES.includes(themeName) ? themeName : 'default';
    document.cookie = `theme=${serverTheme}; path=/; max-age=31536000; SameSite=Lax`;
}

// Set theme by name
function setTheme(themeName, reload = true) {
    if (!THEMES.includes(themeName)) {
        themeName = 'dark';
    }

    const currentTheme = getTheme();
    const currentIsViewTheme = VIEW_THEMES.includes(currentTheme);
    const newIsViewTheme = VIEW_THEMES.includes(themeName);

    // Store in localStorage and cookie
    document.documentElement.setAttribute('data-theme', themeName);
    localStorage.setItem('theme', themeName);
    setThemeCookie(themeName);

    // Update theme selector if on settings page
    const themeSelect = document.getElementById('theme-select');
    if (themeSelect) {
        themeSelect.value = themeName;
    }

    // Update dark mode toggle for backwards compatibility
    const darkModeToggle = document.getElementById('dark-mode');
    if (darkModeToggle) {
        darkModeToggle.checked = themeName === 'dark';
    }

    // If switching between view themes (e.g., default <-> aim1.0), reload page
    if (reload && (currentIsViewTheme !== newIsViewTheme || (currentIsViewTheme && newIsViewTheme && currentTheme !== themeName))) {
        window.location.reload();
    }
}

// Toggle between dark/light (legacy function for backwards compatibility)
function toggleTheme() {
    const current = getTheme();
    // If using a special theme, toggle between dark and that theme
    if (current === 'dark') {
        setTheme('light');
    } else if (current === 'light') {
        setTheme('dark');
    } else {
        // For special themes like aim1.0, toggle to dark
        setTheme('dark');
    }
}

// Initialize theme cookie on page load
(function() {
    const theme = getTheme();
    setThemeCookie(theme);
})();

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    // Ctrl/Cmd + K: Focus search
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        document.querySelector('#search-input')?.focus();
    }
    // Escape: Close modals and pickers
    if (e.key === 'Escape') {
        document.querySelectorAll('.modal').forEach(m => m.classList.add('hidden'));
        closeAllPickers();
    }
});

// ============================================
// EMOJI & STICKER PICKER
// ============================================

let currentPicker = null; // 'emoji' or 'sticker' or null

// Close all pickers
function closeAllPickers() {
    document.querySelectorAll('.picker-container').forEach(p => p.remove());
    document.querySelectorAll('.reaction-picker').forEach(p => p.remove());
    document.querySelectorAll('.input-emoji-btn, .input-sticker-btn, .aim-input-btn, .ym-input-btn').forEach(b => b.classList.remove('active'));
    currentPicker = null;
}

// Create emoji picker HTML
function createEmojiPicker(onSelect, isRetroTheme = false) {
    const picker = document.createElement('div');
    picker.className = 'picker-container emoji-picker';

    // Get recent emoji
    const recent = typeof getRecentEmoji === 'function' ? getRecentEmoji() : [];

    // Build category tabs
    const categories = EMOJI_DATA?.categories || {};
    const categoryKeys = Object.keys(categories);

    let tabsHTML = '';
    categoryKeys.forEach((key, i) => {
        const cat = categories[key];
        tabsHTML += `<div class="picker-tab${i === 0 ? ' active' : ''}" data-category="${key}">${cat.icon}</div>`;
    });

    // Build emoji grid (start with first category)
    const firstCat = categories[categoryKeys[0]];
    let gridHTML = '';
    if (recent.length > 0) {
        gridHTML += `<div class="emoji-category-title">Recent</div><div class="emoji-grid">`;
        recent.forEach(emoji => {
            gridHTML += `<div class="emoji-item" data-emoji="${emoji}">${emoji}</div>`;
        });
        gridHTML += `</div>`;
    }
    if (firstCat) {
        gridHTML += `<div class="emoji-category-title">${firstCat.name}</div><div class="emoji-grid">`;
        firstCat.emoji.slice(0, 48).forEach(emoji => {
            gridHTML += `<div class="emoji-item" data-emoji="${emoji}">${emoji}</div>`;
        });
        gridHTML += `</div>`;
    }

    if (isRetroTheme) {
        picker.innerHTML = `
            <div class="picker-titlebar">
                <span>Emoticons</span>
                <div class="picker-titlebar-close" onclick="closeAllPickers()">X</div>
            </div>
            <div class="picker-header">
                <input type="text" class="picker-search" placeholder="Search emoji...">
            </div>
            <div class="picker-tabs">${tabsHTML}</div>
            <div class="picker-content">${gridHTML}</div>
            <div class="picker-statusbar">Click an emoticon to insert</div>
        `;
    } else {
        picker.innerHTML = `
            <div class="picker-header">
                <input type="text" class="picker-search" placeholder="Search emoji...">
            </div>
            <div class="picker-tabs">${tabsHTML}</div>
            <div class="picker-content">${gridHTML}</div>
        `;
    }

    // Event listeners
    picker.querySelectorAll('.emoji-item').forEach(item => {
        item.addEventListener('click', () => {
            const emoji = item.dataset.emoji;
            if (typeof addRecentEmoji === 'function') {
                addRecentEmoji(emoji);
            }
            onSelect(emoji);
            closeAllPickers();
        });
    });

    // Category tab switching
    picker.querySelectorAll('.picker-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            picker.querySelectorAll('.picker-tab').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            const catKey = tab.dataset.category;
            const cat = categories[catKey];
            if (cat) {
                let html = `<div class="emoji-category-title">${cat.name}</div><div class="emoji-grid">`;
                cat.emoji.forEach(emoji => {
                    html += `<div class="emoji-item" data-emoji="${emoji}">${emoji}</div>`;
                });
                html += `</div>`;
                picker.querySelector('.picker-content').innerHTML = html;

                // Re-attach event listeners
                picker.querySelectorAll('.emoji-item').forEach(item => {
                    item.addEventListener('click', () => {
                        const emoji = item.dataset.emoji;
                        if (typeof addRecentEmoji === 'function') {
                            addRecentEmoji(emoji);
                        }
                        onSelect(emoji);
                        closeAllPickers();
                    });
                });
            }
        });
    });

    // Search functionality
    const searchInput = picker.querySelector('.picker-search');
    searchInput.addEventListener('input', () => {
        const query = searchInput.value.toLowerCase();
        if (query.length < 2) return;

        let results = [];
        Object.values(categories).forEach(cat => {
            cat.emoji.forEach(emoji => {
                if (results.length < 48) {
                    results.push(emoji);
                }
            });
        });

        let html = `<div class="emoji-category-title">Search Results</div><div class="emoji-grid">`;
        results.slice(0, 48).forEach(emoji => {
            html += `<div class="emoji-item" data-emoji="${emoji}">${emoji}</div>`;
        });
        html += `</div>`;
        picker.querySelector('.picker-content').innerHTML = html;

        picker.querySelectorAll('.emoji-item').forEach(item => {
            item.addEventListener('click', () => {
                const emoji = item.dataset.emoji;
                if (typeof addRecentEmoji === 'function') {
                    addRecentEmoji(emoji);
                }
                onSelect(emoji);
                closeAllPickers();
            });
        });
    });

    return picker;
}

// Create sticker picker HTML
function createStickerPicker(onSelect, isRetroTheme = false) {
    const picker = document.createElement('div');
    picker.className = 'picker-container sticker-picker';

    const packs = window.STICKER_PACKS || {};
    const packKeys = Object.keys(packs);
    const firstPack = packs[packKeys[0]];

    // Build pack tabs
    let packTabsHTML = '';
    packKeys.forEach((key, i) => {
        const pack = packs[key];
        const thumbSticker = pack.stickers.find(s => s.id === pack.thumbnail);
        const thumbSVG = thumbSticker ? thumbSticker.svg : '';
        packTabsHTML += `<div class="sticker-pack-tab${i === 0 ? ' active' : ''}" data-pack="${key}" title="${pack.name}">${thumbSVG}</div>`;
    });

    // Build sticker grid
    let gridHTML = '';
    if (firstPack) {
        firstPack.stickers.forEach(sticker => {
            gridHTML += `<div class="sticker-item" data-pack="${firstPack.id}" data-sticker="${sticker.id}" title="${sticker.name}">${sticker.svg}</div>`;
        });
    }

    if (isRetroTheme) {
        picker.innerHTML = `
            <div class="picker-titlebar">
                <span>Stickers</span>
                <div class="picker-titlebar-close" onclick="closeAllPickers()">X</div>
            </div>
            <div class="sticker-grid">${gridHTML}</div>
            <div class="sticker-pack-tabs">${packTabsHTML}</div>
            <div class="picker-statusbar">Click a sticker to send</div>
        `;
    } else {
        picker.innerHTML = `
            <div class="sticker-grid">${gridHTML}</div>
            <div class="sticker-pack-tabs">${packTabsHTML}</div>
        `;
    }

    // Event listeners for stickers
    picker.querySelectorAll('.sticker-item').forEach(item => {
        item.addEventListener('click', () => {
            const packId = item.dataset.pack;
            const stickerId = item.dataset.sticker;
            if (typeof addRecentSticker === 'function') {
                addRecentSticker(packId, stickerId);
            }
            onSelect(packId, stickerId);
            closeAllPickers();
        });
    });

    // Pack tab switching
    picker.querySelectorAll('.sticker-pack-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            picker.querySelectorAll('.sticker-pack-tab').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            const packKey = tab.dataset.pack;
            const pack = packs[packKey];
            if (pack) {
                let html = '';
                pack.stickers.forEach(sticker => {
                    html += `<div class="sticker-item" data-pack="${pack.id}" data-sticker="${sticker.id}" title="${sticker.name}">${sticker.svg}</div>`;
                });
                picker.querySelector('.sticker-grid').innerHTML = html;

                // Re-attach event listeners
                picker.querySelectorAll('.sticker-item').forEach(item => {
                    item.addEventListener('click', () => {
                        const packId = item.dataset.pack;
                        const stickerId = item.dataset.sticker;
                        if (typeof addRecentSticker === 'function') {
                            addRecentSticker(packId, stickerId);
                        }
                        onSelect(packId, stickerId);
                        closeAllPickers();
                    });
                });
            }
        });
    });

    return picker;
}

// Create quick reaction picker
function createReactionPicker(messageId, onReact) {
    const picker = document.createElement('div');
    picker.className = 'reaction-picker';

    const quickEmoji = EMOJI_DATA?.quick || ['👍', '❤️', '😂', '😮', '😢', '🙏'];

    let html = '';
    quickEmoji.forEach(emoji => {
        html += `<div class="emoji-item" data-emoji="${emoji}">${emoji}</div>`;
    });
    html += `<div class="more-btn" title="More reactions">+</div>`;

    picker.innerHTML = html;

    // Event listeners
    picker.querySelectorAll('.emoji-item').forEach(item => {
        item.addEventListener('click', (e) => {
            e.stopPropagation();
            onReact(messageId, item.dataset.emoji);
            picker.remove();
        });
    });

    picker.querySelector('.more-btn').addEventListener('click', (e) => {
        e.stopPropagation();
        picker.remove();
        // Show full emoji picker for reactions
        const wrapper = document.querySelector(`[data-message-id="${messageId}"]`);
        if (wrapper) {
            const fullPicker = createEmojiPicker((emoji) => {
                onReact(messageId, emoji);
            });
            wrapper.appendChild(fullPicker);
        }
    });

    return picker;
}

// Render reaction badges for a message
function renderReactions(reactions, messageId, currentUserId) {
    if (!Array.isArray(reactions) || reactions.length === 0) return '';

    let html = '<div class="message-reactions">';
    reactions.forEach(r => {
        const isMe = r.me || (r.users && r.users.includes(currentUserId));
        html += `<div class="reaction-badge${isMe ? ' me' : ''}" data-message-id="${messageId}" data-emoji="${r.emoji}">
            <span class="emoji">${r.emoji}</span>
            <span class="count">${r.count}</span>
        </div>`;
    });
    html += '</div>';
    return html;
}

// Render a sticker message
function renderStickerMessage(packId, stickerId) {
    const packs = window.STICKER_PACKS || {};
    const pack = packs[packId];
    if (!pack) return '<div class="message-sticker">[Sticker]</div>';

    const sticker = pack.stickers.find(s => s.id === stickerId);
    if (!sticker) return '<div class="message-sticker">[Sticker]</div>';

    return `<div class="message-sticker" onclick="showStickerLightbox('${packId}', '${stickerId}')" style="cursor: pointer;">${sticker.svg}</div>`;
}

// Show sticker in a larger lightbox view
function showStickerLightbox(packId, stickerId) {
    const packs = window.STICKER_PACKS || {};
    const pack = packs[packId];
    if (!pack) return;

    const sticker = pack.stickers.find(s => s.id === stickerId);
    if (!sticker) return;

    // Create overlay
    const overlay = document.createElement('div');
    overlay.className = 'sticker-lightbox-overlay';
    overlay.id = 'sticker-lightbox-overlay';
    overlay.innerHTML = `
        <div class="sticker-lightbox-content">
            <div class="sticker-lightbox-sticker">${sticker.svg}</div>
            <div class="sticker-lightbox-name">${sticker.name}</div>
            <div class="sticker-lightbox-pack">${pack.name}</div>
        </div>
    `;

    // Close on click
    overlay.onclick = (e) => {
        if (e.target === overlay || e.target.closest('.sticker-lightbox-content')) {
            overlay.remove();
        }
    };

    // Close on Escape
    const escHandler = (e) => {
        if (e.key === 'Escape') {
            overlay.remove();
            document.removeEventListener('keydown', escHandler);
        }
    };
    document.addEventListener('keydown', escHandler);

    document.body.appendChild(overlay);
}

// Toggle emoji picker
function toggleEmojiPicker(button, inputElement) {
    const isRetro = VIEW_THEMES.includes(getTheme());

    if (currentPicker === 'emoji') {
        closeAllPickers();
        return;
    }

    closeAllPickers();
    currentPicker = 'emoji';
    button.classList.add('active');

    const picker = createEmojiPicker((emoji) => {
        if (inputElement) {
            const start = inputElement.selectionStart || 0;
            const end = inputElement.selectionEnd || 0;
            const text = inputElement.value;
            inputElement.value = text.substring(0, start) + emoji + text.substring(end);
            inputElement.focus();
            inputElement.setSelectionRange(start + emoji.length, start + emoji.length);
        }
    }, isRetro);

    // Find the closest positioned ancestor or use a wrapper with relative positioning
    const container = button.closest('.relative') || button.closest('#message-input-wrapper') || button.closest('#aim-picker-wrapper') || button.closest('#ym-picker-wrapper') || button.parentElement;
    if (!container.style.position) {
        container.style.position = 'relative';
    }
    container.appendChild(picker);
}

// Toggle sticker picker
function toggleStickerPicker(button, onSendSticker) {
    const isRetro = VIEW_THEMES.includes(getTheme());

    if (currentPicker === 'sticker') {
        closeAllPickers();
        return;
    }

    closeAllPickers();
    currentPicker = 'sticker';
    button.classList.add('active');

    const picker = createStickerPicker((packId, stickerId) => {
        if (onSendSticker) {
            onSendSticker(packId, stickerId);
        }
    }, isRetro);

    // Find the closest positioned ancestor or use a wrapper with relative positioning
    const container = button.closest('.relative') || button.closest('#message-input-wrapper') || button.closest('#aim-picker-wrapper') || button.closest('#ym-picker-wrapper') || button.parentElement;
    if (!container.style.position) {
        container.style.position = 'relative';
    }
    container.appendChild(picker);
}

// Show reaction picker on message
function showReactionPicker(messageElement, messageId, onReact) {
    // Remove any existing reaction pickers
    document.querySelectorAll('.reaction-picker').forEach(p => p.remove());

    const picker = createReactionPicker(messageId, onReact);
    messageElement.style.position = 'relative';
    messageElement.appendChild(picker);
}

// Close pickers when clicking outside
document.addEventListener('click', (e) => {
    if (!e.target.closest('.picker-container') &&
        !e.target.closest('.reaction-picker') &&
        !e.target.closest('.input-emoji-btn') &&
        !e.target.closest('.input-sticker-btn') &&
        !e.target.closest('.aim-input-btn') &&
        !e.target.closest('.ym-input-btn') &&
        !e.target.closest('.ym-emoticon-btn') &&
        !e.target.closest('.message-action-btn')) {
        closeAllPickers();
    }
});

// ============================================
// API HELPERS FOR REACTIONS
// ============================================

// Add reaction to a message
async function addReaction(chatId, messageId, emoji) {
    try {
        const response = await fetch(`/api/v1/chats/${chatId}/messages/${messageId}/react`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ emoji }),
        });
        return response.ok;
    } catch (err) {
        console.error('Failed to add reaction:', err);
        return false;
    }
}

// Remove reaction from a message
async function removeReaction(chatId, messageId) {
    try {
        const response = await fetch(`/api/v1/chats/${chatId}/messages/${messageId}/react`, {
            method: 'DELETE',
        });
        return response.ok;
    } catch (err) {
        console.error('Failed to remove reaction:', err);
        return false;
    }
}

// Send a sticker message
async function sendStickerMessage(chatId, packId, stickerId) {
    try {
        const packs = window.STICKER_PACKS || {};
        const pack = packs[packId];
        const sticker = pack?.stickers.find(s => s.id === stickerId);

        const response = await fetch(`/api/v1/chats/${chatId}/messages`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                type: 'sticker',
                content: sticker ? sticker.name : stickerId,
                sticker_pack_id: packId,
                sticker_id: stickerId,
            }),
        });
        return response.ok;
    } catch (err) {
        console.error('Failed to send sticker:', err);
        return false;
    }
}
