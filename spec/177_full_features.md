# Full Feature UI Support & Modernization Plan

## Overview

This document outlines the comprehensive plan to:
1. Add missing UI pages/components for backend features
2. Modernize the UI by removing rounded corners and shadows
3. Ensure font-size and styling consistency across all components

---

## Part 1: Feature-to-UI Coverage Analysis

### Current Features (11 total in `feature/` directory)

| Feature | Backend API | UI Pages | Status |
|---------|-------------|----------|--------|
| **Workspaces** | handler/workspace.go | None | Missing UI |
| **Teams** | handler/team.go | team.html | Complete |
| **Projects** | handler/project.go | home.html (list), board.html | Missing settings page |
| **Issues** | handler/issue.go | issues.html, issue.html, board.html | Complete |
| **Columns** | handler/column.go | board.html (inline) | Missing management UI |
| **Cycles** | handler/cycle.go | cycles.html | Complete |
| **Assignees** | handler/assignee.go | issue.html (inline) | Complete |
| **Comments** | handler/comment.go | issue.html (inline) | Complete |
| **Users** | handler/auth.go | login.html, register.html | Complete |
| **Fields** | handler/field.go | None | Missing UI |
| **Values** | handler/value.go | None | Missing UI |

---

## Part 2: Missing UI Components

### 2.1 Workspace Management UI (Priority: High)

**Current State**: Backend supports full CRUD for workspaces, but no UI exists.

**Required Components**:

#### A. Workspace Switcher (Sidebar)
- Add workspace dropdown in sidebar header
- Show current workspace name
- List all user's workspaces
- Quick create option

#### B. Workspace Settings Page (`workspace-settings.html`)
- URL: `/w/{slug}/settings`
- Features:
  - Edit workspace name and slug
  - Manage workspace members (invite, remove, change roles)
  - Delete workspace (with confirmation)
  - Workspace-level settings

**Files to Create/Modify**:
- `assets/views/default/pages/workspace-settings.html` (new)
- `assets/views/default/layouts/default.html` (add workspace switcher)
- `app/web/handler/page.go` (add page handler)
- `assets/static/js/app.js` (add workspace switching logic)

---

### 2.2 Custom Fields Management UI (Priority: High)

**Current State**: Backend supports custom fields (text, number, bool, date, ts, select, user, json), but no UI exists.

**Required Components**:

#### A. Project Fields Page (`project-fields.html`)
- URL: `/w/{slug}/project/{id}/fields`
- Features:
  - List all custom fields for project
  - Create new field (modal):
    - Key (machine-readable)
    - Name (display name)
    - Kind (dropdown: text, number, bool, date, select, etc.)
    - Required toggle
    - Settings JSON for select options
  - Edit field (modal)
  - Reorder fields (drag & drop)
  - Archive/unarchive fields
  - Delete field (with confirmation)

#### B. Field Values in Issue Detail
- Show custom fields section in issue.html sidebar
- Render appropriate input based on field kind:
  - text: input
  - number: number input
  - bool: checkbox
  - date: date picker
  - select: dropdown
  - user: user picker
- Save on change

**Files to Create/Modify**:
- `assets/views/default/pages/project-fields.html` (new)
- `assets/views/default/pages/issue.html` (add custom fields section)
- `app/web/handler/page.go` (add page handler)
- `assets/static/css/default.css` (add field-related styles)
- `assets/static/js/app.js` (add field management logic)

---

### 2.3 Project Settings Page (Priority: Medium)

**Current State**: Projects can be created from home page, but no dedicated settings page.

**Required Components**:

#### A. Project Settings Page (`project-settings.html`)
- URL: `/w/{slug}/project/{id}/settings`
- Features:
  - Edit project name, key, description
  - Manage columns (board workflow):
    - Add/remove columns
    - Reorder columns (drag & drop)
    - Edit column names
    - Set default column
  - Archive/unarchive project
  - Delete project (with confirmation)

**Files to Create/Modify**:
- `assets/views/default/pages/project-settings.html` (new)
- `assets/views/default/layouts/default.html` (add settings link in project nav)
- `app/web/handler/page.go` (add page handler)

---

### 2.4 Enhanced Navigation

**Add to Sidebar**:
- Workspace switcher dropdown (top of sidebar)
- Settings link per project (gear icon)
- Fields management link per project

**Add to User Menu**:
- Workspace settings link
- User profile/settings link

---

## Part 3: UI Modernization - Remove Rounded & Shadows

### 3.1 CSS Variables to Update

```css
/* Change from */
--radius: 0.5rem;

/* Change to */
--radius: 0;
```

### 3.2 Components with Border-Radius to Remove

| Component | Current | Target |
|-----------|---------|--------|
| `.btn` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.input`, `.textarea`, `.select` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.card` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.modal-content` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.modal-close` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.dropdown-menu` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.dropdown-item` | `border-radius: calc(var(--radius) - 2px)` | `border-radius: 0` |
| `.alert` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.nav-item` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.sidebar-toggle` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.search-trigger` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.board-column` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.issue-card` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.issue-key-badge` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.issue-description` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.auth-card` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.quick-add-form input` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.filter-checkbox` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.priority-badge` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.command-item` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.member-item` | `border-radius: var(--radius)` | `border-radius: 0` |
| `.filter-menu` | (inherited from dropdown) | `border-radius: 0` |

**Pill shapes to change (currently 9999px):**
| Component | Current | Target |
|-----------|---------|--------|
| `.status-badge` | `border-radius: 9999px` | `border-radius: 0` |
| `.role-badge` | `border-radius: 9999px` | `border-radius: 0` |
| `.progress-bar` | `border-radius: 9999px` | `border-radius: 0` |
| `.progress-fill` | `border-radius: 9999px` | `border-radius: 0` |
| `.column-count` | `border-radius: 9999px` | `border-radius: 0` |
| `.filter-badge` | `border-radius: 9999px` | `border-radius: 0` |
| `.avatar` | `border-radius: 50%` | `border-radius: 0` |
| `.rounded-full` | `border-radius: 9999px` | `border-radius: 0` |

**Special cases:**
| Component | Current | Target |
|-----------|---------|--------|
| `.search-trigger kbd` | `border-radius: 4px` | `border-radius: 0` |
| `.command-shortcut` | `border-radius: 4px` | `border-radius: 0` |
| `.prose code` | `border-radius: 4px` | `border-radius: 0` |
| `.priority-indicator` | `border-radius: 2px` | `border-radius: 0` |
| `.status-badge::before` (dot) | `border-radius: 50%` | Keep for status indicator |
| Sidebar logo SVG `rx="8"` | rounded corners | `rx="0"` |

---

### 3.3 Components with Box-Shadow to Remove

| Component | Current Shadow | Target |
|-----------|----------------|--------|
| `.card` | `0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)` | `none` |
| `.modal-content` | `0 25px 50px -12px rgb(0 0 0 / 0.25)` | `none` |
| `.dropdown-menu` | `0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)` | `none` |
| `.auth-card` | `0 25px 50px -12px rgb(0 0 0 / 0.1)` | `none` |
| `.issue-card:hover` | `0 2px 4px rgb(0 0 0 / 0.05)` | `none` |
| `.btn:focus-visible` | `0 0 0 2px hsl(var(--background)), 0 0 0 4px hsl(var(--ring))` | Replace with border outline |
| `.input:focus`, `.textarea:focus`, `.select:focus` | `0 0 0 2px hsl(var(--ring) / 0.2)` | Replace with border only |

**Focus states replacement:**
Instead of box-shadow for focus, use border-color change only:
```css
.btn:focus-visible {
  outline: 2px solid hsl(var(--ring));
  outline-offset: 2px;
}

.input:focus, .textarea:focus, .select:focus {
  border-color: hsl(var(--ring));
  /* No box-shadow */
}
```

---

## Part 4: Font Size & Styling Consistency

### 4.1 Typography Scale (Standardize)

**Current scale** (keep but enforce):
```css
--text-xs: 0.75rem;    /* 10.5px at 14px base */
--text-sm: 0.875rem;   /* 12.25px */
--text-base: 1rem;     /* 14px */
--text-lg: 1.125rem;   /* 15.75px */
--text-xl: 1.25rem;    /* 17.5px */
--text-2xl: 1.5rem;    /* 21px */
--text-3xl: 1.875rem;  /* 26.25px */
```

### 4.2 Component Font Size Audit

| Component | Current Size | Standardized Size |
|-----------|--------------|-------------------|
| Body text | 1rem | 1rem (--text-base) |
| Page title (h1) | 1.5rem | 1.5rem (--text-2xl) |
| Section title | 1.125rem | 1.125rem (--text-lg) |
| Card title | 1rem | 1rem (--text-base) |
| Labels | 0.875rem | 0.875rem (--text-sm) |
| Input text | 0.875rem | 0.875rem (--text-sm) |
| Button text | 0.875rem | 0.875rem (--text-sm) |
| Button small | 0.8125rem | 0.75rem (--text-xs) |
| Nav items | 0.875rem | 0.875rem (--text-sm) |
| Nav section title | 0.6875rem | 0.75rem (--text-xs) |
| Status badge | 0.75rem | 0.75rem (--text-xs) |
| Issue key | 0.75rem | 0.75rem (--text-xs) |
| Timestamps | 0.75rem | 0.75rem (--text-xs) |
| Muted/helper text | 0.8125rem | 0.75rem (--text-xs) |

**Issues to fix:**
- `0.8125rem` is non-standard - use `0.75rem` or `0.875rem`
- `0.6875rem` is non-standard - use `0.75rem`
- `0.625rem` is non-standard - use `0.75rem`

### 4.3 Spacing Consistency

**Standardize to 4px scale:**
```css
--space-1: 0.25rem;  /* 4px */
--space-2: 0.5rem;   /* 8px */
--space-3: 0.75rem;  /* 12px */
--space-4: 1rem;     /* 16px */
--space-5: 1.25rem;  /* 20px */
--space-6: 1.5rem;   /* 24px */
--space-8: 2rem;     /* 32px */
```

### 4.4 Button Height Consistency

| Variant | Current | Standardized |
|---------|---------|--------------|
| Default | 36px | 32px |
| Small | 32px | 28px |
| Large | 44px | 40px |
| Icon | 36px | 32px |

### 4.5 Input Height Consistency

| Component | Current | Standardized |
|-----------|---------|--------------|
| Input | 36px | 32px |
| Textarea | auto (min 80px) | auto (min 80px) |
| Select | 36px | 32px |

---

## Part 5: Implementation Order

### Phase 1: CSS Modernization (No new pages)
1. Update `--radius` to `0` in CSS variables
2. Remove all `box-shadow` declarations
3. Update focus states to use outline instead of box-shadow
4. Standardize font sizes (remove non-standard values)
5. Update button/input heights
6. Test all existing pages

### Phase 2: Workspace UI
1. Add workspace switcher to sidebar layout
2. Create workspace settings page
3. Add workspace member management
4. Update navigation

### Phase 3: Custom Fields UI
1. Create project fields management page
2. Add field display to issue detail sidebar
3. Add field editing on issue page
4. Add field filtering to issues list

### Phase 4: Project Settings
1. Create project settings page
2. Add column management UI
3. Add project archive/delete

### Phase 5: Polish & Testing
1. Full UI review for consistency
2. Responsive testing
3. E2E tests for new pages

---

## Part 6: File Changes Summary

### New Files
- `assets/views/default/pages/workspace-settings.html`
- `assets/views/default/pages/project-settings.html`
- `assets/views/default/pages/project-fields.html`

### Modified Files
- `assets/static/css/default.css` - Major styling updates
- `assets/views/default/layouts/default.html` - Workspace switcher, navigation updates
- `assets/views/default/pages/issue.html` - Custom fields section
- `assets/views/default/pages/home.html` - Project settings links
- `assets/views/default/pages/board.html` - Settings link
- `assets/static/js/app.js` - New functionality for workspaces, fields
- `app/web/handler/page.go` - New page handlers
- `app/web/server.go` - New routes

---

## Part 7: CSS Changes Detail

### 7.1 Variables Section Update

```css
:root {
  /* Remove radius */
  --radius: 0;

  /* Add standardized spacing */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-3: 0.75rem;
  --space-4: 1rem;
  --space-6: 1.5rem;
  --space-8: 2rem;

  /* Standardized heights */
  --height-sm: 28px;
  --height-md: 32px;
  --height-lg: 40px;
}
```

### 7.2 Remove All Shadows

Search and replace in CSS:
- `box-shadow: 0 1px 3px` -> `box-shadow: none` or remove
- `box-shadow: 0 25px 50px` -> `box-shadow: none` or remove
- `box-shadow: 0 10px 15px` -> `box-shadow: none` or remove
- `box-shadow: 0 2px 4px` -> `box-shadow: none` or remove

### 7.3 Fix Focus States

```css
/* Before */
.btn:focus-visible {
  outline: none;
  box-shadow: 0 0 0 2px hsl(var(--background)), 0 0 0 4px hsl(var(--ring));
}

/* After */
.btn:focus-visible {
  outline: 2px solid hsl(var(--ring));
  outline-offset: 2px;
}

/* Before */
.input:focus {
  outline: none;
  border-color: hsl(var(--ring));
  box-shadow: 0 0 0 2px hsl(var(--ring) / 0.2);
}

/* After */
.input:focus {
  outline: none;
  border-color: hsl(var(--ring));
}
```

### 7.4 Standardize Non-Standard Font Sizes

```css
/* Replace */
font-size: 0.8125rem; -> font-size: 0.875rem; /* or 0.75rem */
font-size: 0.6875rem; -> font-size: 0.75rem;
font-size: 0.625rem;  -> font-size: 0.75rem;
```

---

## Appendix: Complete Component Checklist

### Buttons
- [ ] Remove border-radius
- [ ] Update focus state (outline instead of box-shadow)
- [ ] Standardize heights (32px default)
- [ ] Ensure consistent font-size (0.875rem)

### Inputs/Forms
- [ ] Remove border-radius
- [ ] Update focus state (border only, no shadow)
- [ ] Standardize heights (32px)
- [ ] Ensure consistent font-size (0.875rem)

### Cards
- [ ] Remove border-radius
- [ ] Remove box-shadow
- [ ] Keep border for definition

### Modals
- [ ] Remove border-radius
- [ ] Remove box-shadow
- [ ] Update backdrop (keep semi-transparent)

### Dropdowns
- [ ] Remove border-radius
- [ ] Remove box-shadow
- [ ] Keep border

### Badges/Tags
- [ ] Remove pill shape (9999px -> 0)
- [ ] Keep colored backgrounds

### Avatars
- [ ] Remove circular shape (50% -> 0)
- [ ] Square avatars

### Progress Bars
- [ ] Remove border-radius from bar and fill

### Navigation
- [ ] Remove border-radius from nav items
- [ ] Consistent spacing

### Tables
- [ ] Already minimal, verify consistency

### Alerts
- [ ] Remove border-radius
- [ ] Keep colored backgrounds/borders
