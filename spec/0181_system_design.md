# Kanban Design System Specification

## Overview

This document defines the design system for the Kanban application, establishing consistent patterns, components, and principles for all UI elements. The goal is to create a cohesive, modern, and maintainable design language inspired by Linear.app's minimal aesthetic.

---

## 1. Design Principles

### 1.1 Core Philosophy

1. **Minimal Chrome** - Reduce visual noise; let content breathe
2. **Purposeful Color** - Use color semantically (status, priority, actions)
3. **Consistent Spacing** - Follow the 4px grid system strictly
4. **Subtle Depth** - Shadows over borders for hierarchy
5. **Smooth Interactions** - All transitions at 150ms for responsiveness
6. **Accessibility First** - WCAG 2.1 AA compliance minimum

### 1.2 Design Tokens Philosophy

All design decisions are expressed as CSS custom properties (tokens). No magic numbers in component styles. Every spacing, color, and dimension references a token.

---

## 2. Design Tokens

### 2.1 Color System

#### Base Colors
```css
--color-white: 0 0% 100%;
--color-black: 222 47% 11%;

/* Grayscale */
--gray-50: 210 20% 98%;
--gray-100: 220 14% 96%;
--gray-200: 220 13% 91%;
--gray-300: 216 12% 84%;
--gray-400: 218 11% 65%;
--gray-500: 220 9% 46%;
--gray-600: 215 14% 34%;
--gray-700: 217 19% 27%;
--gray-800: 215 28% 17%;
--gray-900: 221 39% 11%;
```

#### Semantic Colors
```css
/* Primary - Main brand/action color */
--primary: 222 47% 11%;
--primary-foreground: 0 0% 100%;

/* Secondary - Supporting actions */
--secondary: 220 14% 96%;
--secondary-foreground: 222 47% 11%;

/* Destructive - Danger/delete actions */
--destructive: 0 72% 51%;
--destructive-foreground: 0 0% 100%;

/* Success */
--success: 142 71% 45%;
--success-foreground: 0 0% 100%;

/* Warning */
--warning: 38 92% 50%;
--warning-foreground: 0 0% 100%;

/* Info */
--info: 217 91% 60%;
--info-foreground: 0 0% 100%;
```

#### Status Colors (Workflow States)
```css
--status-backlog: 220 9% 46%;      /* Gray - Not started */
--status-todo: 221 83% 53%;        /* Blue - Ready to start */
--status-in-progress: 38 92% 50%;  /* Yellow - Active work */
--status-done: 142 71% 45%;        /* Green - Completed */
--status-canceled: 220 9% 46%;     /* Gray - Canceled */
```

#### Priority Colors
```css
--priority-none: 220 9% 46%;       /* Gray - No priority */
--priority-low: 221 83% 53%;       /* Blue - Low */
--priority-medium: 38 92% 50%;     /* Yellow - Medium */
--priority-high: 25 95% 53%;       /* Orange - High */
--priority-urgent: 0 72% 51%;      /* Red - Urgent */
```

### 2.2 Typography Scale

#### Font Families
```css
--font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
--font-mono: 'JetBrains Mono', 'Fira Code', Consolas, monospace;
```

#### Font Sizes
```css
--text-xs: 0.6875rem;    /* 11px */
--text-sm: 0.75rem;      /* 12px */
--text-base: 0.875rem;   /* 14px - Default */
--text-md: 1rem;         /* 16px */
--text-lg: 1.125rem;     /* 18px */
--text-xl: 1.25rem;      /* 20px */
--text-2xl: 1.5rem;      /* 24px */
--text-3xl: 1.875rem;    /* 30px */
```

#### Font Weights
```css
--font-normal: 400;
--font-medium: 500;
--font-semibold: 600;
--font-bold: 700;
```

#### Line Heights
```css
--leading-none: 1;
--leading-tight: 1.25;
--leading-snug: 1.375;
--leading-normal: 1.5;
--leading-relaxed: 1.625;
```

### 2.3 Spacing Scale (4px Grid)

```css
--space-0: 0;
--space-1: 0.25rem;   /* 4px */
--space-2: 0.5rem;    /* 8px */
--space-3: 0.75rem;   /* 12px */
--space-4: 1rem;      /* 16px */
--space-5: 1.25rem;   /* 20px */
--space-6: 1.5rem;    /* 24px */
--space-8: 2rem;      /* 32px */
--space-10: 2.5rem;   /* 40px */
--space-12: 3rem;     /* 48px */
--space-16: 4rem;     /* 64px */
```

### 2.4 Sizing

#### Component Heights
```css
--height-xs: 1.5rem;    /* 24px */
--height-sm: 1.75rem;   /* 28px */
--height-md: 2rem;      /* 32px - Default */
--height-lg: 2.5rem;    /* 40px */
--height-xl: 3rem;      /* 48px */
```

#### Layout Dimensions
```css
--sidebar-width: 240px;
--sidebar-collapsed: 64px;
--topbar-height: 56px;
--content-max-width: 1200px;
--modal-width-sm: 400px;
--modal-width-md: 500px;
--modal-width-lg: 640px;
--modal-width-xl: 800px;
```

### 2.5 Border Radius

```css
--radius-sm: 0.375rem;  /* 6px */
--radius-md: 0.5rem;    /* 8px - Default */
--radius-lg: 0.75rem;   /* 12px */
--radius-xl: 1rem;      /* 16px */
--radius-full: 9999px;  /* Pill shape */
```

### 2.6 Shadows (Elevation)

```css
/* Level 0 - Flat */
--shadow-none: none;

/* Level 1 - Subtle lift */
--shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.04);

/* Level 2 - Cards, dropdowns */
--shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.05),
             0 2px 4px -2px rgba(0, 0, 0, 0.05);

/* Level 3 - Modals, popovers */
--shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.08),
             0 4px 6px -4px rgba(0, 0, 0, 0.05);

/* Level 4 - Dialogs */
--shadow-xl: 0 20px 25px -5px rgba(0, 0, 0, 0.1),
             0 8px 10px -6px rgba(0, 0, 0, 0.05);

/* Focus ring */
--ring-shadow: 0 0 0 3px hsl(var(--primary) / 0.15);
```

### 2.7 Transitions

```css
--transition-fast: 100ms;
--transition-base: 150ms;
--transition-slow: 200ms;
--transition-slower: 300ms;

--ease-default: cubic-bezier(0.4, 0, 0.2, 1);
--ease-in: cubic-bezier(0.4, 0, 1, 1);
--ease-out: cubic-bezier(0, 0, 0.2, 1);
--ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);
```

### 2.8 Z-Index Scale

```css
--z-dropdown: 50;
--z-sticky: 100;
--z-fixed: 150;
--z-modal-backdrop: 200;
--z-modal: 210;
--z-popover: 250;
--z-tooltip: 300;
```

---

## 3. Component Specifications

### 3.1 Buttons

#### Base Button
All buttons share these properties:
- Height: `var(--height-md)` (32px)
- Padding: `0 var(--space-4)` (0 16px)
- Border radius: `var(--radius-md)` (8px)
- Font size: `var(--text-base)` (14px)
- Font weight: `var(--font-medium)` (500)
- Transition: `var(--transition-base)` (150ms)
- Cursor: pointer
- Display: inline-flex, align-items: center, justify-content: center
- Gap: `var(--space-2)` (8px) for icon + text

#### Button Variants

| Variant | Background | Text | Border | Use Case |
|---------|------------|------|--------|----------|
| `btn-primary` | `--primary` | `--primary-foreground` | none | Primary actions |
| `btn-secondary` | `--color-white` | `--foreground` | `--border` | Secondary actions |
| `btn-ghost` | transparent | `--foreground` | none | Tertiary actions |
| `btn-destructive` | `--destructive` | `--destructive-foreground` | none | Delete/danger |
| `btn-outline` | transparent | `--foreground` | `--border` | Alternative secondary |
| `btn-link` | transparent | `--primary` | none | Inline text links |

#### Button Sizes

| Size | Height | Padding | Font Size | Class |
|------|--------|---------|-----------|-------|
| Small | 28px | 0 12px | 12px | `btn-sm` |
| Medium | 32px | 0 16px | 14px | (default) |
| Large | 40px | 0 24px | 14px | `btn-lg` |

#### Icon Buttons
- Class: `btn-icon`
- Square shape: width = height
- Sizes: 28px, 32px, 40px matching button sizes

#### Button States
- **Hover**: Slight background shift (10% darker or lighter)
- **Active/Pressed**: Scale(0.98), darker background
- **Disabled**: Opacity 0.5, cursor: not-allowed
- **Loading**: Spinner icon, pointer-events: none

### 3.2 Form Controls

#### Input Fields
All form inputs share:
- Height: `var(--height-md)` (32px)
- Padding: `0 var(--space-3)` (0 12px)
- Border: 1px solid `var(--border)`
- Border radius: `var(--radius-md)` (8px)
- Font size: `var(--text-base)` (14px)
- Background: `var(--background)`

**States:**
- Default: `border-color: var(--border)`
- Focus: `border-color: var(--primary)`, `box-shadow: var(--ring-shadow)`
- Error: `border-color: var(--destructive)`
- Disabled: `opacity: 0.5`, `background: var(--muted)`

**Consolidated Classes:**
- `.form-control` - Base class for all inputs (replaces `.input`, `.create-box-title`, etc.)
- `.form-control-sm` - 28px height
- `.form-control-lg` - 40px height

#### Textarea
- Class: `.form-control.form-textarea`
- Min-height: 80px
- Resize: vertical
- Auto-grow option via JavaScript

#### Select
- Class: `.form-control.form-select`
- Custom arrow indicator
- Same height/padding as inputs

#### Checkbox & Radio
- Size: 16px × 16px
- Border radius: 4px (checkbox), 50% (radio)
- Checked state: Primary color background

#### Form Layout
```html
<div class="form-group">
  <label class="form-label">Label</label>
  <input class="form-control" />
  <span class="form-hint">Helper text</span>
  <span class="form-error">Error message</span>
</div>
```

### 3.3 Cards

#### Base Card
- Background: `var(--card)`
- Border radius: `var(--radius-lg)` (12px)
- Shadow: `var(--shadow-sm)`
- Border: none (use shadow for depth)

#### Card Anatomy
```html
<div class="card">
  <div class="card-header">
    <h3 class="card-title">Title</h3>
    <div class="card-actions">...</div>
  </div>
  <div class="card-body">Content</div>
  <div class="card-footer">Actions</div>
</div>
```

**Padding:**
- Header: `var(--space-4) var(--space-5)` (16px 20px)
- Body: `var(--space-4) var(--space-5)` (16px 20px)
- Footer: `var(--space-3) var(--space-5)` (12px 20px)

#### Card Variants
- `.card-flat` - No shadow, subtle border
- `.card-interactive` - Hover effect (lift + shadow increase)
- `.card-selected` - Primary border color

### 3.4 Modal / Dialog

#### Structure
```html
<div class="modal-backdrop">
  <div class="modal">
    <div class="modal-header">
      <h2 class="modal-title">Title</h2>
      <button class="btn-icon btn-ghost modal-close">×</button>
    </div>
    <div class="modal-body">Content</div>
    <div class="modal-footer">
      <button class="btn btn-secondary">Cancel</button>
      <button class="btn btn-primary">Confirm</button>
    </div>
  </div>
</div>
```

#### Specifications
- Backdrop: `rgba(0, 0, 0, 0.4)`, `backdrop-filter: blur(4px)`
- Modal: White background, `var(--radius-xl)`, `var(--shadow-xl)`
- Width: `var(--modal-width-md)` (500px) default
- Max-height: `calc(100vh - 128px)`
- Overflow: `modal-body` scrolls if needed
- Z-index: `var(--z-modal)`

#### Size Variants
| Variant | Width | Class |
|---------|-------|-------|
| Small | 400px | `modal-sm` |
| Medium | 500px | (default) |
| Large | 640px | `modal-lg` |
| XL | 800px | `modal-xl` |
| Full | 100% - 64px | `modal-full` |

#### Animations
- Enter: Scale 0.95 → 1, opacity 0 → 1, translateY(-10px) → 0
- Exit: Reverse of enter
- Duration: 200ms

### 3.5 Dropdown Menu

#### Structure
```html
<div class="dropdown">
  <button class="dropdown-trigger">Trigger</button>
  <div class="dropdown-menu">
    <div class="dropdown-item">Item 1</div>
    <div class="dropdown-item">Item 2</div>
    <div class="dropdown-divider"></div>
    <div class="dropdown-item dropdown-item-danger">Delete</div>
  </div>
</div>
```

#### Specifications
- Position: Absolute, below trigger
- Background: White
- Shadow: `var(--shadow-lg)`
- Border radius: `var(--radius-md)` (8px)
- Min-width: 160px
- Padding: `var(--space-1)` (4px)
- Z-index: `var(--z-dropdown)`

#### Dropdown Item
- Height: 32px
- Padding: `0 var(--space-3)` (0 12px)
- Border radius: `var(--radius-sm)` (6px)
- Hover: `var(--muted)` background
- Gap: `var(--space-2)` for icon + text

#### Animation
- Enter: translateY(-4px) → 0, opacity 0 → 1
- Duration: 150ms

### 3.6 Status Badge

Displays workflow status with color coding.

```html
<span class="badge badge-status badge-status-todo">
  <svg class="badge-icon">...</svg>
  <span>Todo</span>
</span>
```

#### Specifications
- Display: inline-flex, align-items: center
- Height: 24px
- Padding: `var(--space-1) var(--space-2)` (4px 8px)
- Border radius: `var(--radius-full)` (pill)
- Font size: `var(--text-sm)` (12px)
- Font weight: `var(--font-medium)` (500)
- Icon size: 14px × 14px
- Gap: `var(--space-1)` (4px)

#### Status Variants
| Status | Background | Text Color | Class |
|--------|------------|------------|-------|
| Backlog | Gray 10% | Gray 100% | `badge-status-backlog` |
| Todo | Blue 10% | Blue 100% | `badge-status-todo` |
| In Progress | Yellow 10% | Yellow 100% | `badge-status-in-progress` |
| Done | Green 10% | Green 100% | `badge-status-done` |
| Canceled | Gray 10% | Gray 100% | `badge-status-canceled` |

### 3.7 Priority Badge

Displays priority level with color coding.

```html
<span class="badge badge-priority badge-priority-high">
  <svg class="badge-icon">...</svg>
  <span>High</span>
</span>
```

#### Priority Variants
| Priority | Color | Class |
|----------|-------|-------|
| None | Gray | `badge-priority-none` |
| Low | Blue | `badge-priority-low` |
| Medium | Yellow | `badge-priority-medium` |
| High | Orange | `badge-priority-high` |
| Urgent | Red | `badge-priority-urgent` |

### 3.8 Avatar

User/team member representation.

```html
<div class="avatar avatar-md">
  <img src="..." alt="Name" />
</div>
<!-- or with initials -->
<div class="avatar avatar-md">
  <span class="avatar-initials">JD</span>
</div>
```

#### Sizes
| Size | Dimensions | Class |
|------|------------|-------|
| XS | 20px | `avatar-xs` |
| SM | 24px | `avatar-sm` |
| MD | 32px | `avatar-md` (default) |
| LG | 40px | `avatar-lg` |
| XL | 48px | `avatar-xl` |

#### Avatar Group
```html
<div class="avatar-group">
  <div class="avatar avatar-sm">...</div>
  <div class="avatar avatar-sm">...</div>
  <div class="avatar avatar-sm avatar-more">+3</div>
</div>
```
- Overlap: -8px margin-left
- Border: 2px solid white

### 3.9 Table

Data table with sorting and selection.

```html
<table class="table">
  <thead>
    <tr>
      <th class="table-th-sortable" data-sort="title">
        Title <span class="sort-indicator"></span>
      </th>
    </tr>
  </thead>
  <tbody>
    <tr class="table-row-clickable">
      <td>Content</td>
    </tr>
  </tbody>
</table>
```

#### Specifications
- Border-collapse: separate
- Border-spacing: 0
- Font size: `var(--text-base)` (14px)

**Header:**
- Font size: `var(--text-xs)` (11px)
- Font weight: `var(--font-semibold)` (600)
- Text transform: uppercase
- Letter spacing: 0.05em
- Color: `var(--muted-foreground)`
- Padding: `var(--space-3) var(--space-4)` (12px 16px)

**Body:**
- Padding: `var(--space-3) var(--space-4)` (12px 16px)
- Border-bottom: 1px solid `var(--border)`
- Hover: `var(--muted)` background

### 3.10 Empty State

Placeholder when no data exists.

```html
<div class="empty-state">
  <div class="empty-state-icon">
    <svg>...</svg>
  </div>
  <h3 class="empty-state-title">No items found</h3>
  <p class="empty-state-description">Get started by creating your first item.</p>
  <button class="btn btn-primary">Create Item</button>
</div>
```

#### Specifications
- Display: flex, flex-direction: column, align-items: center
- Padding: `var(--space-12) var(--space-4)` (48px 16px)
- Text-align: center
- Icon size: 48px, color: `var(--muted-foreground)`
- Title: `var(--text-md)`, `var(--font-semibold)`
- Description: `var(--text-base)`, `var(--muted-foreground)`
- Max-width: 400px

### 3.11 Alert / Toast

System messages and notifications.

```html
<div class="alert alert-success">
  <svg class="alert-icon">...</svg>
  <div class="alert-content">
    <div class="alert-title">Success</div>
    <div class="alert-message">Your changes have been saved.</div>
  </div>
  <button class="alert-close">×</button>
</div>
```

#### Variants
| Variant | Icon | Color | Class |
|---------|------|-------|-------|
| Success | Check | Green | `alert-success` |
| Error | X | Red | `alert-error` |
| Warning | Alert | Yellow | `alert-warning` |
| Info | Info | Blue | `alert-info` |

#### Specifications
- Display: flex, align-items: flex-start
- Padding: `var(--space-3) var(--space-4)` (12px 16px)
- Border radius: `var(--radius-md)` (8px)
- Border-left: 4px solid variant color
- Background: variant color at 5% opacity

### 3.12 Tooltip

Contextual information on hover.

```html
<div class="tooltip-trigger" data-tooltip="Tooltip content">
  Hover me
</div>
```

#### Specifications
- Position: Absolute, above/below trigger
- Background: `var(--gray-900)`
- Color: white
- Font size: `var(--text-sm)` (12px)
- Padding: `var(--space-1) var(--space-2)` (4px 8px)
- Border radius: `var(--radius-sm)` (6px)
- Max-width: 200px
- Z-index: `var(--z-tooltip)`
- Arrow: 6px triangle

---

## 4. Layout System

### 4.1 App Shell

```
┌─────────────────────────────────────────────────────┐
│ Sidebar │           Topbar                          │
│ (240px) │───────────────────────────────────────────│
│         │                                           │
│  Logo   │           Main Content                    │
│         │                                           │
│  Nav    │                                           │
│         │                                           │
│         │                                           │
│  User   │                                           │
└─────────┴───────────────────────────────────────────┘
```

#### Sidebar
- Width: 240px (64px collapsed)
- Position: Fixed left
- Height: 100vh
- Background: `var(--sidebar-bg)` (subtle gray)
- Sections: Logo, Navigation, User Menu

#### Topbar
- Height: 56px
- Position: Sticky top
- Background: `var(--background)` with blur
- Z-index: `var(--z-sticky)`
- Contains: Toggle, Breadcrumb, Search, Actions

#### Main Content
- Margin-left: `var(--sidebar-width)`
- Padding: `var(--space-6)` (24px)
- Max-width: `var(--content-max-width)` (optional)

### 4.2 Page Templates

#### List Page (Issues, Cycles, Team)
```html
<main class="page page-list">
  <header class="page-header">
    <h1 class="page-title">Page Title</h1>
    <div class="page-actions">...</div>
  </header>
  <div class="page-toolbar">
    <!-- Filters, search, view options -->
  </div>
  <div class="page-content">
    <table class="table">...</table>
  </div>
</main>
```

#### Board Page
```html
<main class="page page-board">
  <header class="page-header">...</header>
  <div class="board">
    <div class="board-column">...</div>
    <div class="board-column">...</div>
  </div>
</main>
```

#### Detail Page (Issue Detail)
```html
<main class="page page-detail">
  <div class="detail-layout">
    <div class="detail-main">
      <!-- Issue content, comments -->
    </div>
    <aside class="detail-sidebar">
      <!-- Properties, metadata -->
    </aside>
  </div>
</main>
```

#### Settings Page
```html
<main class="page page-settings">
  <div class="settings-layout">
    <nav class="settings-nav">...</nav>
    <div class="settings-content">
      <div class="settings-section">...</div>
    </div>
  </div>
</main>
```

### 4.3 Responsive Breakpoints

```css
--breakpoint-sm: 640px;   /* Mobile landscape */
--breakpoint-md: 768px;   /* Tablet */
--breakpoint-lg: 1024px;  /* Desktop */
--breakpoint-xl: 1280px;  /* Large desktop */
```

#### Responsive Behavior
- **< 768px**: Sidebar hidden (overlay mode), single column layouts
- **768px - 1024px**: Sidebar collapsed, 2-column layouts
- **> 1024px**: Full sidebar, all columns visible

---

## 5. Icon System

### 5.1 Icon Specifications

- Library: Custom SVG icons (Lucide-inspired)
- Default size: 18px × 18px
- Stroke width: 2px
- Stroke linecap/linejoin: round
- Color: currentColor (inherits from parent)

### 5.2 Icon Sizes

| Size | Dimensions | Use Case |
|------|------------|----------|
| XS | 14px | Badges, inline text |
| SM | 16px | Button icons, form icons |
| MD | 18px | Navigation, default |
| LG | 20px | Headers, emphasis |
| XL | 24px | Empty states, features |

### 5.3 Icon Categories

- **Navigation**: Home, Board, List, Calendar, Settings
- **Actions**: Plus, Edit, Trash, Check, X
- **Status**: Circle, CircleDot, Loader, CheckCircle, XCircle
- **Priority**: SignalLow, SignalMedium, SignalHigh, AlertTriangle
- **Misc**: User, Users, Search, Filter, Sort

---

## 6. Accessibility Standards

### 6.1 Color Contrast

- Text on background: 4.5:1 minimum (WCAG AA)
- Large text: 3:1 minimum
- Interactive elements: 3:1 against adjacent colors

### 6.2 Focus States

All interactive elements must have visible focus indicators:
- Focus ring: 3px offset, primary color at 15% opacity
- No removal of focus outlines
- Skip navigation link for keyboard users

### 6.3 ARIA Patterns

- Modals: `role="dialog"`, `aria-modal="true"`, `aria-labelledby`
- Dropdowns: `aria-expanded`, `aria-haspopup`
- Buttons: Descriptive `aria-label` for icon-only buttons
- Status changes: `aria-live="polite"` for dynamic updates

### 6.4 Keyboard Navigation

- Tab order follows visual order
- Escape closes modals/dropdowns
- Arrow keys for menu navigation
- Enter/Space activates buttons and links

---

## 7. Motion & Animation

### 7.1 Principles

1. **Purposeful**: Animations guide attention and provide feedback
2. **Quick**: Keep under 300ms for responsiveness
3. **Subtle**: Avoid flashy or distracting effects
4. **Consistent**: Same easing and duration across similar interactions

### 7.2 Animation Tokens

```css
/* Durations */
--duration-fast: 100ms;
--duration-normal: 150ms;
--duration-slow: 200ms;
--duration-slower: 300ms;

/* Easings */
--ease-default: cubic-bezier(0.4, 0, 0.2, 1);
--ease-in: cubic-bezier(0.4, 0, 1, 1);
--ease-out: cubic-bezier(0, 0, 0.2, 1);
--ease-bounce: cubic-bezier(0.34, 1.56, 0.64, 1);
```

### 7.3 Common Animations

| Animation | Duration | Easing | Properties |
|-----------|----------|--------|------------|
| Fade | 150ms | ease-out | opacity |
| Slide | 200ms | ease-out | transform |
| Scale | 150ms | ease-out | transform |
| Modal enter | 200ms | ease-out | opacity, transform |
| Dropdown | 150ms | ease-out | opacity, transform |

### 7.4 Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## 8. Refactoring Plan

### 8.1 Current Inconsistencies

| Issue | Current State | Target State |
|-------|---------------|--------------|
| Form inputs | Multiple classes (`.input`, `.create-box-title`, etc.) | Single `.form-control` class |
| Button states | Mix of `.active`, `.selected` | Standardized `.is-active`, `.is-selected` |
| Dropdown positioning | Mixed absolute/fixed | Unified positioning utility |
| Property chips | Overlapping `.property-chip`, `.status-badge` | Clear hierarchy |
| Modal footers | Inconsistent layouts | Standard left/right patterns |
| Empty states | Varied structures | Single `.empty-state` pattern |
| Spacing | Arbitrary values | Token-based spacing only |
| Color usage | Some hardcoded values | All via CSS variables |

### 8.2 CSS Refactoring Tasks

#### Phase 1: Design Tokens (default.css)
- [ ] Consolidate all color definitions into `:root`
- [ ] Add missing tokens (breakpoints, z-index scale)
- [ ] Remove duplicate/unused variables
- [ ] Organize tokens by category with comments

#### Phase 2: Base Styles
- [ ] Normalize reset styles
- [ ] Typography hierarchy refinement
- [ ] Base focus states
- [ ] Scrollbar styling

#### Phase 3: Component Consolidation
- [ ] Unify form controls to `.form-control`
- [ ] Standardize button variants and states
- [ ] Consolidate card patterns
- [ ] Unify badge/chip components
- [ ] Standardize modal patterns
- [ ] Unify dropdown behavior

#### Phase 4: Layout Patterns
- [ ] Refine app shell structure
- [ ] Create page template classes
- [ ] Standardize responsive breakpoints
- [ ] Grid/flex utility classes

### 8.3 Template Refactoring Tasks

#### Layouts
| File | Changes Required |
|------|------------------|
| `default.html` | Update sidebar nav structure, standardize topbar |
| `auth.html` | Align form styling with design system |

#### Pages
| Page | Priority | Changes Required |
|------|----------|------------------|
| `board.html` | High | Standardize column/card components, fix form inputs |
| `issue.html` | High | Unify property cards, standardize badges |
| `inbox.html` | High | Consolidate form classes, fix create box |
| `issues.html` | Medium | Standardize table, filters, toolbar |
| `calendar.html` | Medium | Align with card patterns |
| `gantt.html` | Medium | Consistent styling with other views |
| `cycles.html` | Medium | Table and form standardization |
| `team.html` | Low | Avatar and table patterns |
| `project-settings.html` | Low | Form and layout patterns |
| `project-fields.html` | Low | Form and table patterns |
| `workspace-settings.html` | Low | Form patterns |
| `login.html` | Low | Form control consolidation |
| `register.html` | Low | Form control consolidation |

### 8.4 Component Mapping

#### Old Class → New Class

```
.input → .form-control
.textarea → .form-control.form-textarea
.select → .form-control.form-select
.create-box-title → .form-control
.create-box-description → .form-control.form-textarea
.comment-input → .form-control.form-textarea

.status-badge → .badge.badge-status
.priority-badge → .badge.badge-priority
.property-chip → .chip

.modal-status-selector → .dropdown-menu
.property-dropdown → .dropdown-menu

.active (on buttons) → .is-active
.selected → .is-selected
.hidden → .is-hidden
```

### 8.5 Implementation Order

1. **CSS Tokens & Base** - Establish foundation
2. **Form Controls** - High impact, used everywhere
3. **Buttons** - Standardize all interactive elements
4. **Badges & Chips** - Status/priority display
5. **Cards** - Container components
6. **Modals & Dropdowns** - Overlay components
7. **Layout Shell** - App structure
8. **Individual Pages** - Apply patterns page by page

---

## 9. File Structure

### 9.1 Recommended CSS Organization

```
assets/static/css/
├── tokens/
│   ├── colors.css
│   ├── typography.css
│   ├── spacing.css
│   └── animations.css
├── base/
│   ├── reset.css
│   ├── typography.css
│   └── utilities.css
├── components/
│   ├── buttons.css
│   ├── forms.css
│   ├── cards.css
│   ├── modals.css
│   ├── dropdowns.css
│   ├── badges.css
│   ├── tables.css
│   └── avatars.css
├── layout/
│   ├── app-shell.css
│   ├── sidebar.css
│   ├── topbar.css
│   └── pages.css
└── default.css (imports all above)
```

*Note: For simplicity, this project keeps everything in `default.css` with clear section comments.*

### 9.2 Section Order in default.css

```css
/* ===================================
   1. CSS Custom Properties (Tokens)
   =================================== */

/* ===================================
   2. CSS Reset & Base Styles
   =================================== */

/* ===================================
   3. Typography
   =================================== */

/* ===================================
   4. Utility Classes
   =================================== */

/* ===================================
   5. Components
   5.1 Buttons
   5.2 Form Controls
   5.3 Cards
   5.4 Badges & Chips
   5.5 Avatars
   5.6 Tables
   5.7 Modals
   5.8 Dropdowns
   5.9 Tooltips
   5.10 Alerts
   5.11 Empty States
   =================================== */

/* ===================================
   6. Layout
   6.1 App Shell
   6.2 Sidebar
   6.3 Topbar
   6.4 Page Templates
   =================================== */

/* ===================================
   7. Page-Specific Styles
   7.1 Board
   7.2 Issue Detail
   7.3 Calendar
   7.4 Gantt
   7.5 Settings
   =================================== */

/* ===================================
   8. Animations
   =================================== */

/* ===================================
   9. Responsive / Media Queries
   =================================== */

/* ===================================
   10. Dark Mode (Future)
   =================================== */
```

---

## 10. Quality Checklist

### Before Shipping

- [ ] All colors use CSS variables
- [ ] All spacing uses token values
- [ ] All components follow naming convention
- [ ] Focus states visible on all interactive elements
- [ ] No inline styles in templates
- [ ] Responsive behavior tested at all breakpoints
- [ ] Animations respect reduced-motion preference
- [ ] Console shows no CSS errors
- [ ] Visual regression tested against design

---

## Appendix A: Color Palette Reference

### Light Theme

| Token | HSL | Hex | Preview |
|-------|-----|-----|---------|
| `--primary` | 222 47% 11% | #0f172a | Navy |
| `--destructive` | 0 72% 51% | #dc2626 | Red |
| `--success` | 142 71% 45% | #22c55e | Green |
| `--warning` | 38 92% 50% | #f59e0b | Orange |
| `--info` | 217 91% 60% | #3b82f6 | Blue |
| `--muted` | 220 14% 96% | #f1f5f9 | Light Gray |
| `--border` | 220 13% 91% | #e2e8f0 | Border Gray |

### Status Colors (with 10% opacity backgrounds)

| Status | Foreground | Background |
|--------|------------|------------|
| Backlog | `hsl(220 9% 46%)` | `hsl(220 9% 46% / 0.1)` |
| Todo | `hsl(221 83% 53%)` | `hsl(221 83% 53% / 0.1)` |
| In Progress | `hsl(38 92% 50%)` | `hsl(38 92% 50% / 0.1)` |
| Done | `hsl(142 71% 45%)` | `hsl(142 71% 45% / 0.1)` |
| Canceled | `hsl(220 9% 46%)` | `hsl(220 9% 46% / 0.1)` |

---

## Appendix B: Component Quick Reference

### Buttons
```html
<button class="btn btn-primary">Primary</button>
<button class="btn btn-secondary">Secondary</button>
<button class="btn btn-ghost">Ghost</button>
<button class="btn btn-destructive">Delete</button>
<button class="btn btn-sm">Small</button>
<button class="btn btn-lg">Large</button>
<button class="btn btn-icon"><svg>...</svg></button>
```

### Forms
```html
<input class="form-control" type="text" />
<textarea class="form-control form-textarea"></textarea>
<select class="form-control form-select">...</select>
```

### Badges
```html
<span class="badge badge-status badge-status-todo">Todo</span>
<span class="badge badge-priority badge-priority-high">High</span>
```

### Cards
```html
<div class="card">
  <div class="card-header">
    <h3 class="card-title">Title</h3>
  </div>
  <div class="card-body">Content</div>
</div>
```

### Modals
```html
<div class="modal-backdrop">
  <div class="modal modal-md">
    <div class="modal-header">...</div>
    <div class="modal-body">...</div>
    <div class="modal-footer">...</div>
  </div>
</div>
```

---

*Document Version: 1.0*
*Last Updated: 2025-12-27*
*Author: System Design Team*
