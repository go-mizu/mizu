# Spec 0146: Enhanced Chat UI - Shadcn-Inspired Redesign

## Overview

Complete redesign of all chat UI pages with shadcn-inspired styling and real, functional UX improvements from a user perspective.

## Design Principles

1. **Clean Typography**: Inter font family with refined size hierarchy
2. **Subtle Gradients**: Modern gradient accents without being overwhelming
3. **Refined Spacing**: Consistent spacing system (4px base grid)
4. **Micro-interactions**: Smooth hover states and transitions
5. **Accessibility**: Proper focus states, contrast ratios, and keyboard navigation
6. **Real Features**: Functional UX, not just placeholders

---

## Page-by-Page Redesign

### 1. Landing Page (`landing.html`)

**Current Issues:**
- Generic hero section
- Feature cards lack depth
- No social proof or credibility indicators
- No clear value proposition hierarchy

**Redesign:**
- Sticky navigation with blur backdrop
- Hero with animated gradient background
- Stats bar showing active users/servers (social proof)
- Feature section with icons and better descriptions
- Testimonial/community section
- Call-to-action sections with better hierarchy
- Footer with links and legal info

**New Sections:**
1. Navigation (sticky, blur backdrop)
2. Hero (gradient text, dual CTA, stats)
3. Features Grid (4 cards with icons)
4. How It Works (3-step process)
5. Community Showcase (sample servers)
6. Final CTA
7. Footer

---

### 2. Login Page (`login.html`)

**Current Issues:**
- No password visibility toggle
- No "Remember me" option
- No "Forgot password" link
- No social login options placeholder
- Generic error handling

**Redesign:**
- Password visibility toggle button
- "Remember me" checkbox
- "Forgot password?" link
- Divider with "or continue with"
- Social login buttons (Google, GitHub, Discord)
- Better error states with inline validation
- Loading spinner on submit
- Success redirect with animation

**New Features:**
- Password show/hide toggle
- Remember me checkbox
- Forgot password link (to /forgot-password)
- OAuth placeholders (Google, GitHub)
- Real-time validation feedback

---

### 3. Register Page (`register.html`)

**Current Issues:**
- No password strength indicator
- No real-time username availability check
- No terms of service agreement
- Generic validation

**Redesign:**
- Password strength meter (weak/medium/strong)
- Real-time username availability check
- Confirm password field
- Terms of Service checkbox with link
- Privacy Policy link
- Social registration options
- Step-by-step feel (email -> username -> password)

**New Features:**
- Password strength indicator
- Username availability check (debounced)
- Password confirmation
- ToS/Privacy agreement checkbox
- Birthday field (optional, for moderation)

---

### 4. App Page (`app.html`)

**Current Issues:**
- Message editing not exposed in UI
- No message deletion confirmation
- Emoji picker missing
- File upload not functional
- No search functionality
- No pinned messages view
- No reply threading UI

**Redesign:**
- Collapsible member sidebar
- Message context menu (edit, delete, pin, reply, copy)
- Inline message editing with save/cancel
- Delete confirmation modal
- Emoji picker popover
- File upload with preview
- Search modal (Cmd/Ctrl + K)
- Pinned messages panel
- Reply preview above input
- Thread sidebar for threaded messages
- Message grouping by time
- "New messages" divider

**New Features:**
- Cmd/Ctrl+K search modal
- Emoji picker with categories
- Right-click context menu on messages
- Inline edit mode for own messages
- Delete confirmation dialog
- Pinned messages dropdown
- File upload with drag-and-drop
- Reply to message (quote)
- Jump to message (from search)
- Unread message indicator line

---

### 5. Settings Page (`settings.html`)

**Current Issues:**
- Missing many settings categories
- No account deletion flow
- No session management
- No connected accounts
- No data export

**Redesign - Full Settings Structure:**

**User Settings:**
- My Account (email, username, password change)
- Profile (display name, avatar upload, bio, banner)
- Appearance (theme toggle, message display)

**App Settings:**
- Notifications (desktop, sounds, message content)
- Privacy & Safety (DM settings, friend requests, blocked users)
- Voice & Video (placeholder for future)

**Danger Zone:**
- Disable Account
- Delete Account (with confirmation flow)
- Log Out of All Devices

**Activity:**
- Sessions (list of active sessions with logout option)
- Connected Accounts (OAuth connections)
- Data Export Request

**New Features:**
- Avatar upload with crop modal
- Password change form (current + new + confirm)
- Theme toggle (light/dark/system)
- Notification preferences checkboxes
- Privacy toggles
- Session list with device info
- Delete account flow (type confirmation)

---

### 6. Explore Page (`explore.html`)

**Current Issues:**
- No category filters
- Search not functional
- No sorting options
- No pagination

**Redesign:**
- Category tabs (All, Gaming, Art, Music, Tech, etc.)
- Functional search with debounced query
- Sort dropdown (Popular, Newest, Most Members)
- Featured servers banner
- Server cards with hover effects
- Join button with loading state
- Pagination or infinite scroll
- Empty state for no results

**New Features:**
- Category filter tabs
- Live search with results
- Sort options dropdown
- Featured servers carousel
- "Already a member" badge
- Quick join button on cards

---

## CSS Design System Updates

### New Color Tokens
```css
/* Shadcn-inspired additions */
--ring: var(--brand-primary);
--ring-offset: var(--bg-primary);
--muted: var(--bg-secondary);
--muted-foreground: var(--text-muted);
--accent: var(--bg-modifier-hover);
--accent-foreground: var(--text-primary);
--destructive: var(--text-danger);
--destructive-foreground: white;
```

### Component Patterns

**Cards:**
- `border` instead of shadow for cleaner look
- Subtle hover state with border color change
- Optional header/footer sections

**Buttons:**
- `btn-outline` variant (border only)
- `btn-ghost` variant (no background)
- `btn-icon` variant (square, icon only)
- Consistent focus rings

**Inputs:**
- Border visible by default
- Stronger focus ring
- Error state with red border
- Success state with green border
- Label + input + hint structure

**Modals:**
- Overlay with backdrop blur
- Slide-in animation
- Close button in header
- Footer with action buttons

---

## Implementation Order

1. Update CSS with new design tokens and component styles
2. Redesign components (server_list, channel_list, member_list, user_panel)
3. Landing page redesign
4. Auth pages (login, register)
5. Main app page with new features
6. Settings page with all sections
7. Explore page with filters

---

## Files to Modify

### Pages
- `assets/views/default/pages/landing.html`
- `assets/views/default/pages/login.html`
- `assets/views/default/pages/register.html`
- `assets/views/default/pages/app.html`
- `assets/views/default/pages/settings.html`
- `assets/views/default/pages/explore.html`

### Components
- `assets/views/default/components/server_list.html`
- `assets/views/default/components/channel_list.html`
- `assets/views/default/components/member_list.html`
- `assets/views/default/components/user_panel.html`
- `assets/views/default/components/message.html`

### Styles
- `assets/static/css/app.css`

### Layouts
- `assets/views/default/layouts/default.html`
