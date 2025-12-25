# ChatGPT Latest Design Implementation Spec

## Overview

This spec documents the redesign of Mizu Chat UI to match ChatGPT's 2025 design system. The update focuses on modern aesthetics, improved typography, enhanced message rendering, and a cleaner visual hierarchy.

## Design References

- [OpenAI Apps SDK UI Guidelines](https://developers.openai.com/apps-sdk/concepts/ui-guidelines/)
- [OpenAI Apps SDK UI Components](https://github.com/openai/apps-sdk-ui)
- ChatGPT Web Interface (December 2025)

## Color System

### Dark Mode (Default)

```css
--bg-primary: #0d0d0d;        /* Main background - true black */
--bg-secondary: #171717;      /* Sidebar, cards */
--bg-tertiary: #212121;       /* Input fields, elevated surfaces */
--bg-hover: #2f2f2f;          /* Hover states */
--bg-active: #343541;         /* Active/selected states */

--border-default: #2f2f2f;    /* Default borders */
--border-subtle: #ffffff0d;   /* Subtle borders (5% white) */
--border-strong: #4d4d4d;     /* Strong borders */

--text-primary: #ececec;      /* Primary text */
--text-secondary: #b4b4b4;    /* Secondary/muted text */
--text-tertiary: #8e8ea0;     /* Tertiary/placeholder */
--text-disabled: #565869;     /* Disabled text */

--accent-primary: #10a37f;    /* Primary green accent (ChatGPT signature) */
--accent-hover: #1a7f64;      /* Accent hover state */
--accent-muted: #10a37f33;    /* Muted accent (20% opacity) */

--user-bubble: #2f2f2f;       /* User message background */
--bot-bubble: transparent;    /* AI message (no background) */

--success: #10a37f;           /* Success states */
--warning: #f59e0b;           /* Warning states */
--error: #ef4444;             /* Error states */
--info: #3b82f6;              /* Info states */
```

### Light Mode

```css
--bg-primary: #ffffff;        /* Main background */
--bg-secondary: #f7f7f8;      /* Sidebar, cards */
--bg-tertiary: #ececec;       /* Input fields */
--bg-hover: #e5e5e5;          /* Hover states */
--bg-active: #d9d9e3;         /* Active states */

--border-default: #e5e5e5;    /* Default borders */
--border-subtle: #0000000d;   /* Subtle borders */
--border-strong: #c5c5d2;     /* Strong borders */

--text-primary: #0d0d0d;      /* Primary text */
--text-secondary: #6e6e80;    /* Secondary text */
--text-tertiary: #8e8ea0;     /* Tertiary text */
--text-disabled: #acacbe;     /* Disabled text */

--accent-primary: #10a37f;    /* Same green accent */
--accent-hover: #1a7f64;
--accent-muted: #10a37f1a;

--user-bubble: #f7f7f8;       /* User message background */
--bot-bubble: transparent;    /* AI message */
```

## Typography

### Font Stack

```css
font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
```

### Font Sizes

| Name | Size | Line Height | Weight | Usage |
|------|------|-------------|--------|-------|
| heading-xl | 24px | 1.3 | 600 | Page titles |
| heading-lg | 20px | 1.3 | 600 | Section headers |
| heading-md | 16px | 1.4 | 600 | Card titles |
| body | 14px | 1.6 | 400 | Default text |
| body-sm | 13px | 1.5 | 400 | Secondary text |
| caption | 12px | 1.4 | 400 | Labels, timestamps |
| code | 13px | 1.5 | 400 | Code blocks |

### Font Features

- `-webkit-font-smoothing: antialiased`
- `-moz-osx-font-smoothing: grayscale`
- `font-feature-settings: 'cv11' 1` (Alternative lowercase 'l')

## Component Specifications

### 1. Message Bubbles

**User Messages:**
- Background: `var(--user-bubble)`
- Border radius: 20px (rounded-2xl)
- Padding: 12px 16px
- Max width: 70%
- Float: right aligned
- Avatar: Hidden (ChatGPT style) or shown to the right

**AI/Bot Messages:**
- Background: transparent
- Border: none
- Padding: 0
- Full width content
- Avatar: 28px circle, left aligned
- Markdown rendering with proper styling

**Message Container:**
```css
.message-group {
    display: flex;
    gap: 16px;
    padding: 16px 24px;
    max-width: 768px;
    margin: 0 auto;
}

.message-content {
    flex: 1;
    min-width: 0;
    line-height: 1.6;
}
```

### 2. Input Box (Composer)

**Container:**
- Position: Sticky bottom
- Background: `var(--bg-primary)` with gradient fade above
- Padding: 16px 24px
- Max width: 768px
- Centered

**Input Field:**
- Background: `var(--bg-tertiary)`
- Border: 1px solid `var(--border-default)`
- Border radius: 16px (rounded-2xl)
- Padding: 14px 48px 14px 16px
- Min height: 48px
- Max height: 200px (auto-grow)
- Placeholder color: `var(--text-tertiary)`

**Send Button:**
- Position: Absolute right
- Size: 32px circle
- Background: `var(--accent-primary)` when active, `var(--bg-hover)` when empty
- Border radius: 8px
- Icon: Arrow-up, 18px

### 3. Sidebar

**Server Rail (Left):**
- Width: 72px
- Background: `var(--bg-secondary)`
- Icons: 48px circles, 12px radius
- Active indicator: 4px left border accent
- Hover: background lightens, border-radius reduces

**Channel List:**
- Width: 240px
- Background: `var(--bg-secondary)`
- Header height: 48px
- Channel items: 32px height
- Active: `var(--bg-hover)` background
- Hash icon: `var(--text-tertiary)`

**User Panel:**
- Height: 56px
- Fixed to bottom
- Avatar: 32px
- Status indicator: 10px green dot

### 4. Buttons

**Primary Button:**
```css
.btn-primary {
    background: var(--accent-primary);
    color: white;
    padding: 10px 16px;
    border-radius: 8px;
    font-weight: 500;
    font-size: 14px;
    transition: background 0.15s ease;
}
.btn-primary:hover {
    background: var(--accent-hover);
}
.btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}
```

**Secondary Button:**
```css
.btn-secondary {
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border: 1px solid var(--border-default);
}
```

### 5. Modals

- Background overlay: rgba(0, 0, 0, 0.6)
- Modal background: `var(--bg-primary)`
- Border radius: 16px
- Padding: 24px
- Max width: 440px
- Centered with flex
- Backdrop blur: 4px

### 6. Form Inputs

```css
.input {
    background: var(--bg-tertiary);
    border: 1px solid var(--border-default);
    border-radius: 8px;
    padding: 10px 12px;
    font-size: 14px;
    color: var(--text-primary);
    transition: border-color 0.15s ease;
}
.input:focus {
    border-color: var(--accent-primary);
    outline: none;
}
.input::placeholder {
    color: var(--text-tertiary);
}
```

### 7. Scrollbars

```css
::-webkit-scrollbar {
    width: 8px;
}
::-webkit-scrollbar-track {
    background: transparent;
}
::-webkit-scrollbar-thumb {
    background: var(--border-default);
    border-radius: 4px;
}
::-webkit-scrollbar-thumb:hover {
    background: var(--text-tertiary);
}
```

## Layout Structure

### Main Chat Layout

```
+--------------------------------------------------+
|                    Header (48px)                  |
+-------+--------+---------------------------------+
| Rail  | Channel|         Chat Area               |
| 72px  | 240px  |        (flex-1)                 |
|       |        |                                  |
|       |        |  +------------------------+     |
|       |        |  |    Message List       |     |
|       |        |  |    max-w: 768px       |     |
|       |        |  |    centered           |     |
|       |        |  +------------------------+     |
|       |        |                                  |
|       +--------+  +------------------------+     |
|       | User   |  |    Input Composer     |     |
|       | Panel  |  +------------------------+     |
+-------+--------+---------------------------------+
```

### Responsive Breakpoints

| Breakpoint | Behavior |
|------------|----------|
| < 640px | Hide channel sidebar, show hamburger menu |
| < 768px | Full width messages, reduced padding |
| < 1024px | Collapsed server rail icons only |
| >= 1280px | Optimal desktop layout |

## Animations

### Transitions

- Default: `150ms ease`
- Button hover: `background 150ms ease`
- Modal open: `opacity 200ms ease, transform 200ms ease`
- Message appear: `opacity 300ms ease, transform 300ms ease`

### Keyframes

**Message Entry:**
```css
@keyframes messageIn {
    from {
        opacity: 0;
        transform: translateY(8px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}
```

**Typing Indicator:**
```css
@keyframes typingDot {
    0%, 60%, 100% { transform: translateY(0); }
    30% { transform: translateY(-4px); }
}
```

## Implementation Checklist

### Phase 1: Foundation
- [x] Define CSS custom properties in default.html
- [ ] Update Tailwind config with custom theme
- [ ] Add Inter font from Google Fonts
- [ ] Implement scrollbar styling

### Phase 2: Core Components
- [ ] Redesign message bubbles (user vs bot distinction)
- [ ] Update input composer with auto-resize
- [ ] Modernize sidebar components
- [ ] Update user panel

### Phase 3: Pages
- [ ] Update login page
- [ ] Update register page
- [ ] Update settings page

### Phase 4: Polish
- [ ] Add animations and transitions
- [ ] Implement responsive breakpoints
- [ ] Add loading states
- [ ] Test light/dark mode toggle

## File Changes Required

1. `assets/views/default/layouts/default.html`
   - Update CSS variables
   - Add Inter font
   - Extend Tailwind config

2. `assets/views/default/pages/app.html`
   - Redesign message rendering
   - Update input composer
   - Add typing indicator support

3. `assets/views/default/components/server_list.html`
   - Update server icons styling
   - Add active indicator

4. `assets/views/default/components/channel_list.html`
   - Update channel list styling
   - Improve section headers

5. `assets/views/default/components/user_panel.html`
   - Add status indicator
   - Update layout

6. `assets/views/default/pages/login.html`
   - Center card layout
   - Update form styling

7. `assets/views/default/pages/register.html`
   - Match login page style

8. `assets/views/default/pages/settings.html`
   - Update navigation
   - Modernize form sections
