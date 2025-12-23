# Forum UI Redesign Spec

## Overview

Redesign the forum UI with a modern, Reddit-inspired layout that emphasizes readability, clean aesthetics, and intuitive navigation.

## Design Principles

1. **Clean & Minimal**: Reduce visual clutter, use whitespace effectively
2. **Modern Card-Based Layout**: Content in well-defined cards with subtle shadows
3. **Accessible**: Clear contrast, readable typography, keyboard-friendly
4. **Responsive**: Mobile-first approach with desktop enhancements
5. **Dark Mode Ready**: CSS variables for easy theming

## Template Architecture Fix

### Current Issue
Each page template defines its own `{{define "content"}}` block. When templates are parsed together via `ParseFS`, later definitions override earlier ones. This works in Go templates but is fragile.

### Solution
Use unique block names per page that the layout includes conditionally, OR keep current structure but ensure templates are loaded in correct order (which Go's ParseFS handles correctly by template name lookup).

The current structure actually works because:
- Each page file defines both `pagename.html` and `content`
- When rendering `home.html`, Go executes that template which includes `default.html`
- The last defined `content` block (from the current file being rendered) is used

We'll keep the current pattern as it's working correctly.

## Color Palette

```css
/* Light Theme */
--bg-canvas: #f6f8fa          /* Page background */
--bg-primary: #ffffff          /* Card/content background */
--bg-secondary: #f6f8fa        /* Secondary surfaces */
--bg-tertiary: #eaeef2         /* Borders, dividers */

--text-primary: #1f2328        /* Main text */
--text-secondary: #656d76      /* Secondary/muted text */
--text-tertiary: #8b949e       /* Placeholder, hints */

--accent-primary: #2563eb      /* Primary brand color */
--accent-primary-hover: #1d4ed8
--accent-success: #22c55e
--accent-danger: #ef4444
--accent-warning: #f59e0b

--upvote: #ff4500
--downvote: #7193ff

--border-default: #d1d9e0
--border-muted: #eaeef2

--shadow-sm: 0 1px 2px rgba(0,0,0,0.04)
--shadow-md: 0 4px 6px rgba(0,0,0,0.04), 0 1px 3px rgba(0,0,0,0.08)
```

## Typography

```css
--font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans', Helvetica, Arial, sans-serif
--font-mono: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace

--text-xs: 0.75rem    /* 12px */
--text-sm: 0.875rem   /* 14px */
--text-base: 1rem     /* 16px */
--text-lg: 1.125rem   /* 18px */
--text-xl: 1.25rem    /* 20px */
--text-2xl: 1.5rem    /* 24px */

--leading-tight: 1.25
--leading-normal: 1.5
--leading-relaxed: 1.75
```

## Layout Structure

### Header (64px height)
- Logo on left
- Centered search bar (max 600px)
- User actions on right (notifications, profile, create post)

### Main Content
- Max width: 1280px centered
- Content area: flex-grow (min 640px)
- Sidebar: 312px fixed (hidden on mobile)
- Gap: 24px

### Thread Card
- Horizontal layout: vote buttons | content
- Vote buttons: vertical, centered
- Content: metadata → title → preview → actions
- Subtle hover state, clean borders

### Comments
- Threaded with collapse lines
- Depth-based indentation (16px per level)
- Interactive collapse on line click
- Vote buttons inline with actions

## Component Specifications

### Navigation Bar
```
┌─────────────────────────────────────────────────────────────────┐
│ [Logo] Forum     [━━━━━━━ Search ━━━━━━━]     [+] [🔔] [Avatar] │
└─────────────────────────────────────────────────────────────────┘
```

### Thread Card
```
┌─────────────────────────────────────────────────────────────────┐
│ ▲  │ b/boardname · u/author · 3h                                │
│ 42 │ Thread Title Goes Here                                [IMG]│
│ ▼  │ [💬 24 Comments] [⭐ Save] [↗ Share]                        │
└─────────────────────────────────────────────────────────────────┘
```

### Comment Thread
```
│ u/username · 2h · 15 points
│ Comment content here...
│ [▲] [▼] [Reply] [Save] [···]
│
├─│ u/replier · 1h · 5 points
│ │ Reply content here...
│ │ [▲] [▼] [Reply] [Save] [···]
│ │
│ └─│ Nested reply...
```

## Page Templates

### Layout (default.html)
- Clean HTML5 structure
- Meta viewport, charset
- CSS link, deferred JS
- Nav component
- Main content area with slot
- Footer (optional)

### Home Page
- Feed with sort tabs (Hot, New, Top, Rising)
- Thread list
- Sidebar: Popular communities, CTA for guests

### Board Page
- Board header with banner/icon
- Description, stats, join button
- Thread list with board-specific sort
- Sidebar: About, Rules, Moderators

### Thread Page
- Full thread content
- Comment form
- Threaded comments with sorting

### Auth Pages
- Centered card layout
- Clean form design
- Social proof/benefits sidebar

## Files to Update

1. `assets/views/layouts/default.html` - Base layout
2. `assets/views/components/nav.html` - Navigation
3. `assets/views/components/thread_card.html` - Thread card
4. `assets/views/components/comment.html` - Comment component
5. `assets/views/pages/*.html` - All 13 page templates
6. `assets/static/css/app.css` - Complete stylesheet
7. `assets/static/js/app.js` - JavaScript interactions

## Test Plan

Create `app/web/server_ui_test.go` with tests for:
1. All page routes return 200 and valid HTML
2. Navigation renders correctly
3. Thread cards display properly
4. Comment threads render recursively
5. Forms are present on appropriate pages
6. Static assets are served
7. Auth-required pages redirect appropriately

## Implementation Order

1. CSS Variables and Reset
2. Layout (default.html)
3. Navigation (nav.html)
4. Thread Card Component
5. Comment Component
6. Home Page
7. Board Page
8. Thread Page
9. Auth Pages (login, register)
10. User Page
11. Settings Page
12. Search Page
13. Bookmarks & Notifications
14. Error Page
15. All Page
16. JavaScript Updates
17. UI Tests
