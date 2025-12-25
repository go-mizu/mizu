# Spec 0153: AOL Instant Messenger 1.0 Theme

**Status:** Implemented

## Overview

Implement an authentic AOL Instant Messenger 1.0 (circa 1997) theme for the messaging blueprint that recreates the Windows 3.1/95 era aesthetic.

## Visual Reference

AIM 1.0 on Windows 3.1 had these distinctive characteristics:

### Color Palette
- **Window Background**: #c0c0c0 (Windows gray)
- **Title Bar Active**: #000080 (Navy blue)
- **Title Bar Inactive**: #808080 (Gray)
- **Button Face**: #c0c0c0 (Gray)
- **Button Highlight**: #ffffff (White - top/left bevel)
- **Button Shadow**: #808080 (Dark gray - bottom/right bevel)
- **Button Dark Shadow**: #000000 (Black - outer shadow)
- **Message Sent Bubble**: #ffff99 (Yellow)
- **Message Received Bubble**: #ffffff (White)
- **Buddy Online**: #00aa00 (Green)
- **Buddy Away**: #ffaa00 (Orange)
- **Buddy Offline**: #808080 (Gray)

### Typography
- System font: "MS Sans Serif", "Segoe UI", system-ui, sans-serif
- 8pt/11px base size
- Bold for headers and buddy names

### UI Elements
- 3D beveled buttons (raised appearance)
- Sunken input fields and lists
- Classic Windows scrollbars with arrow buttons
- Title bar with minimize/maximize/close buttons
- Status bar at bottom
- Menu bar appearance

### AIM-Specific Elements
- Buddy List with categories (Buddies, Family, Co-Workers)
- Running man icon for online buddies
- Door icon for away buddies
- IM window with yellow/white message bubbles
- "Send" button styling
- Classic AIM logo/branding

## Implementation Plan

### 1. Theme System Enhancement

#### 1.1 Multi-Theme Support
Update the theme system to support multiple named themes beyond just dark/light:
- `default` - Modern dark theme (current)
- `light` - Modern light theme (current)
- `aim1.0` - AOL Instant Messenger 1.0

#### 1.2 Theme Storage
Store theme preference as string in localStorage:
```javascript
localStorage.setItem('theme', 'aim1.0');
```

#### 1.3 CSS Variable Approach
Use `data-theme="aim1.0"` attribute on `<html>` element with complete variable override.

### 2. Directory Structure

```
assets/views/
├── default/
│   ├── layouts/
│   │   └── default.html     # Contains all theme CSS variables
│   └── pages/
│       ├── home.html
│       ├── login.html
│       ├── register.html
│       ├── app.html
│       └── settings.html
```

The AIM 1.0 theme will be implemented entirely through CSS variables in the existing layout, not as a separate view directory.

### 3. CSS Implementation

#### 3.1 CSS Variables for AIM 1.0
```css
[data-theme="aim1.0"] {
    /* Windows 3.1 Gray System Colors */
    --bg-primary: #c0c0c0;
    --bg-secondary: #c0c0c0;
    --bg-tertiary: #ffffff;
    --bg-hover: #d4d4d4;

    /* Text Colors */
    --text-primary: #000000;
    --text-secondary: #404040;
    --text-muted: #808080;

    /* Accent (AIM Yellow/Blue) */
    --accent: #000080;
    --accent-hover: #0000a0;

    /* Borders */
    --border: #808080;
    --border-light: #ffffff;
    --border-dark: #404040;

    /* Message Bubbles */
    --sent-bubble: #ffff99;
    --received-bubble: #ffffff;

    /* 3D Effect Colors */
    --bevel-light: #ffffff;
    --bevel-dark: #808080;
    --bevel-darker: #404040;

    /* Title Bar */
    --titlebar-bg: #000080;
    --titlebar-text: #ffffff;
}
```

#### 3.2 3D Bevel Effects
```css
[data-theme="aim1.0"] .win-button {
    border: 2px solid;
    border-color: var(--bevel-light) var(--bevel-dark) var(--bevel-dark) var(--bevel-light);
    background: var(--bg-primary);
    box-shadow: 1px 1px 0 var(--bevel-darker);
}

[data-theme="aim1.0"] .win-button:active {
    border-color: var(--bevel-dark) var(--bevel-light) var(--bevel-light) var(--bevel-dark);
}

[data-theme="aim1.0"] .win-inset {
    border: 2px solid;
    border-color: var(--bevel-dark) var(--bevel-light) var(--bevel-light) var(--bevel-dark);
}
```

#### 3.3 Classic Scrollbar
```css
[data-theme="aim1.0"] ::-webkit-scrollbar {
    width: 16px;
    background: #c0c0c0;
}

[data-theme="aim1.0"] ::-webkit-scrollbar-thumb {
    background: #c0c0c0;
    border: 2px solid;
    border-color: #ffffff #808080 #808080 #ffffff;
}

[data-theme="aim1.0"] ::-webkit-scrollbar-button {
    background: #c0c0c0;
    border: 2px solid;
    border-color: #ffffff #808080 #808080 #ffffff;
}
```

### 4. UI Components

#### 4.1 Buddy List Sidebar
- Title bar with "Buddy List" text
- Categories: "Buddies", "Family", "Co-Workers"
- Each buddy shows: icon + name + status
- Running man icon (SVG) for online
- Collapsible groups

#### 4.2 Message Window
- Title bar with contact name
- Message area with classic bubbles
- Yellow background for sent messages
- White background for received messages
- Bottom toolbar with Send button

#### 4.3 Classic Window Chrome
```css
.aim-window {
    border: 2px solid;
    border-color: #dfdfdf #404040 #404040 #dfdfdf;
    box-shadow: 1px 1px 0 #000;
}

.aim-titlebar {
    background: linear-gradient(90deg, #000080, #1084d0);
    color: white;
    font-weight: bold;
    padding: 2px 4px;
    display: flex;
    align-items: center;
}
```

### 5. Theme Selector UI

Add theme selector to Settings page under "Appearance" section:

```html
<div class="theme-selector">
    <label>Theme</label>
    <select id="theme-select">
        <option value="dark">Modern Dark</option>
        <option value="light">Modern Light</option>
        <option value="aim1.0">AOL Instant Messenger 1.0</option>
    </select>
</div>
```

Preview cards showing each theme style with radio button selection.

### 6. JavaScript Updates

#### 6.1 app.js Theme Functions
```javascript
function setTheme(themeName) {
    document.documentElement.setAttribute('data-theme', themeName);
    localStorage.setItem('theme', themeName);

    // Update dark mode toggle if on settings page
    const darkModeToggle = document.getElementById('dark-mode');
    if (darkModeToggle) {
        darkModeToggle.checked = themeName === 'dark';
    }
}

function getTheme() {
    return localStorage.getItem('theme') || 'dark';
}
```

### 7. Files to Modify

1. **`assets/views/default/layouts/default.html`**
   - Add AIM 1.0 CSS variables and styles
   - Add win-button, win-inset utility classes

2. **`assets/views/default/pages/settings.html`**
   - Replace dark mode toggle with theme selector
   - Add theme preview cards

3. **`assets/views/default/pages/app.html`**
   - Add conditional classes for AIM styling
   - Add buddy list structure

4. **`assets/static/js/app.js`**
   - Update theme toggle to setTheme function
   - Add theme selector handling

### 8. Testing

- Verify all themes render correctly
- Test theme persistence across page reloads
- Test theme switching without full page reload
- Verify all UI elements are visible and readable in each theme
- Test on different browsers (Chrome, Firefox, Safari)

## Deliverables

1. Spec document (this file)
2. Updated layout with AIM 1.0 CSS
3. Theme selector UI in settings
4. Updated JavaScript for multi-theme support
5. Authentic Windows 3.1/AIM 1.0 visual appearance

---

## Final Implementation

### Directory Structure

```
assets/
├── embed.go                    # Multi-theme template loading
├── static/
│   ├── css/
│   │   ├── aim.css            # Windows 3.1 / AIM styles
│   │   └── default.css        # Modern dark/light theme
│   └── js/
│       └── app.js             # Theme switching with cookie support
└── views/
    ├── aim1.0/                 # Complete AIM 1.0 theme
    │   ├── layouts/
    │   │   └── default.html
    │   └── pages/
    │       ├── app.html       # Buddy List + IM Window
    │       ├── home.html      # Welcome screen
    │       ├── login.html     # Sign On dialog
    │       ├── register.html  # Get Screen Name dialog
    │       └── settings.html  # Preferences with tabs
    └── default/                # Modern theme
        ├── layouts/
        │   └── default.html
        └── pages/
            └── ... (5 pages)
```

### Files Created/Modified

1. **`assets/static/css/default.css`** - Modern theme CSS variables
2. **`assets/static/css/aim.css`** - Complete Windows 3.1 styling (400+ lines)
3. **`assets/views/aim1.0/`** - Complete AIM 1.0 view directory
4. **`assets/embed.go`** - Added `AllTemplates()` and `TemplatesForTheme()`
5. **`app/web/handler/page.go`** - Theme detection from cookie
6. **`app/web/server.go`** - Multi-theme initialization
7. **`assets/static/js/app.js`** - Cookie-based theme switching

### AIM 1.0 Theme Features

**Authentic Windows 3.1 UI:**
- Teal desktop background (#008080)
- Gray window chrome (#c0c0c0) with 3D beveled borders
- Navy blue title bars with gradient (linear-gradient)
- Classic title bar buttons (minimize, maximize, close)
- Menu bars with underlined hotkeys
- Status bars at window bottom

**AIM-Specific Design:**
- Buddy List window with collapsible categories
- Running man icon for online buddies
- Door icon for away status
- IM window with classic message formatting
- Red sender name, blue receiver name
- Font/formatting toolbar
- Send/Close buttons

**Windows 3.1 UI Components:**
- `.win-window` - 3D window frame
- `.win-titlebar` - Blue gradient title bar
- `.win-button` - Raised 3D buttons
- `.win-input` - Sunken input fields
- `.win-listbox` - Classic list selection
- `.win-groupbox` - Labeled group boxes
- `.win-tabs` - Tabbed interface
- Classic scrollbars with arrow buttons

### Theme Switching

1. User selects theme in Settings → Appearance
2. JavaScript stores in `localStorage` + `theme` cookie
3. Page reloads if switching between view themes
4. Server reads cookie via `getTheme()` in page handler
5. Server selects appropriate template set
