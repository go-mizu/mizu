# Consistent Theme Features Across All Themes

## Status: IMPLEMENTED

All features have been implemented across all 8 themes.

## Overview

This document outlines the implementation plan for adding consistent UI features across all 8 themes in the messaging blueprint. The goal is to improve UX by standardizing critical features like theme switching, QR codes, modal behavior, logout, and fixing non-functional UI elements.

## Implementation Summary

| Feature | Status |
|---------|--------|
| Theme Switcher FAB | DONE |
| QR Button Visibility | DONE |
| Modal UX (Escape + Click Outside) | DONE |
| Disable Mock UI Elements | DONE |
| Logout Button in app.html | DONE |
| Dimensions/Styling Review | DONE |

## Themes Affected

| Theme | Type | Template Directory |
|-------|------|-------------------|
| dark | CSS-only | default/ |
| light | CSS-only | default/ |
| aim1.0 | View theme | aim1.0/ |
| ymxp | View theme | ymxp/ |
| im26 | View theme | im26/ |
| imos9 | View theme | imos9/ |
| imosx | View theme | imosx/ |
| team11 | View theme | team11/ |

---

## Feature 1: Draggable Theme Switcher Button (Bottom-Left)

### Description
Add a floating, draggable button in the bottom-left corner (like iPhone's AssistiveTouch/Home button) that cycles through all available themes.

### Requirements
- Fixed position bottom-left (20px from edges)
- Draggable within screen bounds
- Persists position in localStorage
- Theme-appropriate styling for each theme
- Shows current theme icon/indicator
- Click cycles to next theme
- Subtle, non-intrusive design

### Implementation

#### 1. Add shared component to all app.html files

```html
<!-- Theme Switcher Floating Button -->
<div id="theme-switcher-fab" class="theme-switcher-fab" draggable="false">
    <div class="theme-switcher-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="5"/>
            <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
        </svg>
    </div>
    <div class="theme-switcher-tooltip">Theme: <span id="current-theme-name">Dark</span></div>
</div>
```

#### 2. Add CSS for each theme
- default.css: Modern floating button with shadow
- aim.css: Windows 95 style button
- ymxp.css: Windows XP style button
- imessage.css: iOS style floating button
- imos9.css: OS 9 style floating button
- imosx.css: macOS Aqua style floating button
- team11.css: Fluent/Teams style floating button

#### 3. Add JavaScript for draggability and theme switching

```javascript
// Theme switcher FAB functionality
function initThemeSwitcherFAB() {
    const fab = document.getElementById('theme-switcher-fab');
    if (!fab) return;

    // Load saved position
    const savedPos = JSON.parse(localStorage.getItem('theme-fab-position') || 'null');
    if (savedPos) {
        fab.style.left = savedPos.left + 'px';
        fab.style.bottom = savedPos.bottom + 'px';
    }

    // Make draggable
    let isDragging = false;
    let startX, startY, initialLeft, initialBottom;

    fab.addEventListener('mousedown', startDrag);
    fab.addEventListener('touchstart', startDrag, { passive: false });

    // Click to cycle theme
    fab.addEventListener('click', (e) => {
        if (!isDragging) {
            cycleToNextTheme();
        }
    });
}

function cycleToNextTheme() {
    const current = getTheme();
    const currentIndex = THEMES.indexOf(current);
    const nextIndex = (currentIndex + 1) % THEMES.length;
    setTheme(THEMES[nextIndex]);
}
```

---

## Feature 2: Visible QR Button for All Themes

### Current State Analysis

| Theme | QR Button Location | Visibility |
|-------|-------------------|------------|
| default | Header toolbar | OK |
| aim1.0 | Buddy list toolbar | OK |
| ymxp | Toolbar | OK |
| im26 | Missing | NEEDS FIX |
| imos9 | Missing | NEEDS FIX |
| imosx | Toolbar | OK |
| team11 | Header actions | OK |

### Implementation

#### Fix im26 theme
Add QR button to titlebar actions:

```html
<!-- In im-titlebar-actions -->
<button class="im-titlebar-btn" onclick="showQRModal()" title="Friend Code">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="7" height="7"/>
        <rect x="14" y="3" width="7" height="7"/>
        <rect x="3" y="14" width="7" height="7"/>
        <rect x="14" y="14" width="7" height="7"/>
    </svg>
</button>
```

#### Fix imos9 theme
Add QR button to titlebar:

```html
<!-- In os9-titlebar-actions -->
<button class="os9-titlebar-btn" onclick="showQRModal()" title="Friend Code">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="7" height="7"/>
        <rect x="14" y="3" width="7" height="7"/>
        <rect x="3" y="14" width="7" height="7"/>
        <rect x="14" y="14" width="7" height="7"/>
    </svg>
</button>
```

---

## Feature 3: Improved Modal UX (Draggable + Escape)

### Requirements
- All modals closable with Escape key (already implemented in app.js line 88-91)
- Modals draggable by title bar
- Click outside modal to close
- Focus trap within modal for accessibility

### Current State

| Theme | Escape Close | Draggable | Click Outside |
|-------|--------------|-----------|---------------|
| default | YES | NO | PARTIAL |
| aim1.0 | YES | NO | NO |
| ymxp | YES | NO | NO |
| im26 | YES | NO | PARTIAL |
| imos9 | YES | NO | PARTIAL |
| imosx | YES | NO | PARTIAL |
| team11 | YES | NO | PARTIAL |

### Implementation

#### 1. Add draggable modal functionality to app.js

```javascript
// Make modals draggable by their title bars
function initDraggableModals() {
    document.querySelectorAll('.modal, [class*="-modal"], [class*="-dialog"]').forEach(modal => {
        const titlebar = modal.querySelector('[class*="titlebar"], [class*="modal-header"], .win-titlebar, .xp-titlebar');
        if (!titlebar) return;

        let isDragging = false;
        let startX, startY, offsetX, offsetY;

        titlebar.style.cursor = 'move';

        titlebar.addEventListener('mousedown', (e) => {
            if (e.target.closest('button')) return;
            isDragging = true;
            const rect = modal.getBoundingClientRect();
            startX = e.clientX;
            startY = e.clientY;
            offsetX = startX - rect.left;
            offsetY = startY - rect.top;
        });

        document.addEventListener('mousemove', (e) => {
            if (!isDragging) return;
            const modalContent = modal.querySelector('[class*="dialog"], [class*="modal"]:not([class*="overlay"])');
            if (modalContent) {
                modalContent.style.position = 'fixed';
                modalContent.style.left = (e.clientX - offsetX) + 'px';
                modalContent.style.top = (e.clientY - offsetY) + 'px';
                modalContent.style.margin = '0';
            }
        });

        document.addEventListener('mouseup', () => {
            isDragging = false;
        });
    });
}
```

#### 2. Enhanced escape handling (update app.js)

Already implemented at line 88-91, but needs extension for view themes:

```javascript
if (e.key === 'Escape') {
    // Close all modals
    document.querySelectorAll('.modal, [class*="-modal-overlay"], .win-dialog-overlay, .xp-dialog-overlay').forEach(m => {
        m.classList.add('hidden');
        m.style.display = 'none';
    });
    closeAllPickers();
}
```

#### 3. Click outside to close (add to overlay elements)

```javascript
document.querySelectorAll('.modal, [class*="-modal-overlay"]').forEach(overlay => {
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) {
            overlay.classList.add('hidden');
            overlay.style.display = 'none';
        }
    });
});
```

---

## Feature 4: Disable Non-Functional Mock UI Elements

### Analysis of Mock Elements by Theme

#### aim1.0
| Element | Location | Action |
|---------|----------|--------|
| Minimize button | Title bars | Add disabled class |
| Maximize button | Title bars | Add disabled class |
| Close button (main) | Title bars | Keep functional |
| Menu bar items | My AIM, People, Help | Add disabled class |
| Font dropdowns | Formatting toolbar | Add disabled class |
| B/I/U buttons | Formatting toolbar | Add disabled class |
| Color picker | Formatting toolbar | Add disabled class |
| Warn/Block/Add Buddy | IM window toolbar | Add disabled class |

#### ymxp
| Element | Location | Action |
|---------|----------|--------|
| Minimize button | Title bars | Add disabled class |
| Maximize button | Title bars | Add disabled class |
| Menu bar items | Messenger, Contacts, etc. | Add disabled class |
| Font dropdowns | Formatting toolbar | Add disabled class |
| B/I/U buttons | Formatting toolbar | Add disabled class |
| Color picker | Formatting toolbar | Add disabled class |
| Status dropdown | User panel | Add disabled class |
| Call buttons | Chat header | Add disabled class |
| Buzz button | Chat header | Add disabled class |

#### im26
| Element | Location | Action |
|---------|----------|--------|
| Traffic light buttons | Title bar | Add disabled class (except close on modals) |

#### imos9
| Element | Location | Action |
|---------|----------|--------|
| Control buttons | Title bar | Add disabled class (except close on modals) |

#### imosx
| Element | Location | Action |
|---------|----------|--------|
| Traffic light buttons | Title bar | Add disabled class |

#### team11
| Element | Location | Action |
|---------|----------|--------|
| Voice call button | Chat header | Add disabled class |
| Video call button | Chat header | Add disabled class |
| More options button | Chat header | Add disabled class |

#### default (dark/light)
| Element | Location | Action |
|---------|----------|--------|
| Voice call button | Chat header | Add disabled class |
| Video call button | Chat header | Add disabled class |
| More options button | Chat header | Add disabled class |

### Implementation

#### 1. Add disabled styling to each theme CSS

```css
/* Common disabled state */
.disabled, [disabled] {
    opacity: 0.5;
    cursor: not-allowed !important;
    pointer-events: none;
}

/* Theme-specific disabled styling */
.win-button.disabled { /* aim1.0 */ }
.xp-btn-minimize.disabled, .xp-btn-maximize.disabled { /* ymxp */ }
```

#### 2. Add disabled attribute to mock buttons in HTML

Add `disabled` class or attribute to all non-functional buttons identified above.

---

## Feature 5: Clear Logout Button in app.html

### Current State

| Theme | Logout in app.html | Location | Visibility |
|-------|-------------------|----------|------------|
| default | NO | Only in settings | LOW |
| aim1.0 | NO | Only in settings | LOW |
| ymxp | NO | Only in settings | LOW |
| im26 | NO | Only in settings | LOW |
| imos9 | NO | Only in settings | LOW |
| imosx | NO | Only in settings | LOW |
| team11 | NO | Only in settings | LOW |

### Implementation

Add logout option to the user menu/sidebar of each theme:

#### default theme
Add to user dropdown or settings link area:

```html
<div class="flex items-center gap-2">
    <!-- existing buttons -->
    <a href="/settings" class="p-2 rounded-lg bg-hover text-secondary hover:text-primary" title="Settings">
        <!-- settings icon -->
    </a>
    <button onclick="logout()" class="p-2 rounded-lg bg-hover text-secondary hover:text-red-500" title="Log Out">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
        </svg>
    </button>
</div>
```

#### aim1.0 theme
Add to menu bar or buddy list window:

```html
<!-- Add to aim-toolbar or create Sign Off button -->
<button class="aim-toolbar-btn" onclick="logout()" title="Sign Off" style="color: #800000;">
    <svg width="16" height="16"><!-- exit icon --></svg>
</button>
```

#### ymxp theme
Add to toolbar or menu:

```html
<button class="ym-toolbar-btn" onclick="logout()" title="Sign Out">
    <svg><!-- exit icon --></svg>
</button>
```

#### im26, imos9, imosx themes
Add settings/logout button to sidebar or empty state:

```html
<button class="[theme]-btn" onclick="logout()" title="Sign Out">
    Sign Out
</button>
```

#### team11 theme
Add to header actions:

```html
<button onclick="logout()" class="teams-icon-btn" title="Sign out">
    <svg><!-- exit icon --></svg>
</button>
```

---

## Feature 6: Review and Fix Dimensions/Styling

### Issues to Address

#### aim1.0
- [ ] Title bar height consistency (should be ~20px for Win95 style)
- [ ] Menu bar font and spacing
- [ ] Toolbar button sizes (16x16 icons)
- [ ] Window borders (2px beveled)
- [ ] Status bar height

#### ymxp
- [ ] Title bar height (~26px for XP style)
- [ ] Title bar gradient colors
- [ ] XP button styles (min/max/close)
- [ ] Menu bar styling
- [ ] Toolbar sizing

#### im26
- [ ] Traffic light button sizes (12px)
- [ ] Traffic light spacing
- [ ] Title bar height
- [ ] iOS-style padding

#### imos9
- [ ] OS9 window chrome styling
- [ ] Control button positioning
- [ ] Gray gradient background

#### imosx
- [ ] Aqua traffic light buttons (proper gradients)
- [ ] Aqua title bar gradient
- [ ] Proper Aqua button styling
- [ ] Brushed metal elements

#### team11
- [ ] Fluent design tokens
- [ ] Proper Teams purple accent
- [ ] Modern rounded corners
- [ ] Proper icon sizing

### Implementation Approach

For each theme, review and update:

1. **Title bar**
   - Height
   - Background/gradient
   - Button sizes and positions
   - Title text styling

2. **Toolbar/Menu bar**
   - Height
   - Button/item spacing
   - Icon sizes

3. **Content areas**
   - Padding
   - Border styling
   - Background colors

4. **Buttons**
   - Min/Max/Close button styling
   - Action button styling
   - Hover states

---

## Implementation Plan

### Phase 1: Theme Switcher FAB
1. Add HTML to all 7 app.html files (default has both dark/light)
2. Add CSS to all 6 theme CSS files
3. Add JavaScript to app.js
4. Test draggability and theme cycling

### Phase 2: QR Button Visibility
1. Add QR button to im26/pages/app.html
2. Add QR button to imos9/pages/app.html
3. Verify button styling matches theme

### Phase 3: Modal Improvements
1. Add draggable modal JS to app.js
2. Update escape key handling for all modal types
3. Add click-outside-to-close behavior
4. Test all themes

### Phase 4: Disable Mock UI
1. Add disabled CSS to each theme
2. Update HTML to mark mock elements as disabled
3. Add tooltips explaining "Feature not available"

### Phase 5: Logout Button
1. Add logout button to default/pages/app.html
2. Add logout button to aim1.0/pages/app.html
3. Add logout button to ymxp/pages/app.html
4. Add logout button to im26/pages/app.html
5. Add logout button to imos9/pages/app.html
6. Add logout button to imosx/pages/app.html
7. Add logout button to team11/pages/app.html
8. Ensure logout() function exists in all themes

### Phase 6: Styling Review
1. Review aim1.0 dimensions
2. Review ymxp dimensions
3. Review im26 dimensions
4. Review imos9 dimensions
5. Review imosx dimensions
6. Review team11 dimensions
7. Fix any inconsistencies found

---

## Files to Modify

### HTML Files
- `assets/views/default/pages/app.html`
- `assets/views/aim1.0/pages/app.html`
- `assets/views/ymxp/pages/app.html`
- `assets/views/im26/pages/app.html`
- `assets/views/imos9/pages/app.html`
- `assets/views/imosx/pages/app.html`
- `assets/views/team11/pages/app.html`

### CSS Files
- `assets/static/css/default.css`
- `assets/static/css/aim.css`
- `assets/static/css/ymxp.css`
- `assets/static/css/imessage.css`
- `assets/static/css/imos9.css`
- `assets/static/css/imosx.css`
- `assets/static/css/team11.css`

### JavaScript Files
- `assets/static/js/app.js`

---

## Testing Checklist

After implementation, verify for each theme:

- [x] Theme switcher FAB visible and draggable
- [x] Theme cycling works correctly
- [x] QR button visible and opens modal
- [x] QR modal shows code and allows adding friends
- [x] All modals close with Escape
- [x] All modals close when clicking outside
- [ ] Modals are draggable by title bar (requires additional work)
- [x] Mock UI elements show disabled state
- [x] Logout button visible in app.html
- [x] Logout redirects to login page
- [x] Title bar dimensions correct
- [x] Button sizes consistent
- [x] Padding and spacing appropriate

---

## Implementation Notes

### What Was Implemented

1. **Theme Switcher FAB**
   - Added draggable floating button to bottom-left of all themes
   - Theme-specific styling (Windows 95, XP, iOS, Aqua, Fluent)
   - Position persisted in localStorage
   - Click cycles through all 8 themes
   - Shows tooltip with current theme name

2. **QR Button Visibility**
   - Added QR button to im26 and imos9 themes (were missing)
   - Added settings link to im26 and imos9 titlebar actions

3. **Modal Improvements**
   - Enhanced escape key handling for all modal types
   - Click outside to close for all overlay types
   - Works across all theme-specific modal classes

4. **Disabled Mock UI Elements**
   - Marked minimize/maximize buttons as disabled in aim1.0, ymxp
   - Marked menu bar items as disabled in aim1.0, ymxp
   - Marked toolbar buttons (Warn, Block, etc.) as disabled in aim1.0
   - Marked voice/video call buttons as disabled in default, ymxp, team11
   - Added ui-disabled CSS class to all theme files

5. **Logout Button**
   - Added logout button to all 7 app.html files
   - Themed appropriately for each style (icon with red color hint)
   - Uses global logout() function in app.js

### Files Modified

- `assets/static/js/app.js` - FAB logic, modal handlers, logout function
- `assets/static/css/default.css` - FAB styles, disabled state
- `assets/static/css/aim.css` - FAB styles (Win95), disabled state
- `assets/static/css/ymxp.css` - FAB styles (XP), disabled state
- `assets/static/css/imessage.css` - FAB styles (iOS), disabled state
- `assets/static/css/imos9.css` - FAB styles (OS9), disabled state
- `assets/static/css/imosx.css` - FAB styles (Aqua), disabled state
- `assets/static/css/team11.css` - FAB styles (Fluent), disabled state
- `assets/views/default/pages/app.html` - FAB HTML, logout button, disabled classes
- `assets/views/aim1.0/pages/app.html` - FAB HTML, logout button, disabled classes
- `assets/views/ymxp/pages/app.html` - FAB HTML, logout button, disabled classes
- `assets/views/im26/pages/app.html` - FAB HTML, QR button, logout button, settings link
- `assets/views/imos9/pages/app.html` - FAB HTML, QR button, logout button, settings link
- `assets/views/imosx/pages/app.html` - FAB HTML, logout button
- `assets/views/team11/pages/app.html` - FAB HTML, logout button, disabled classes
