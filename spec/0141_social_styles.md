# Social Blueprint - Modern UI Redesign

## Overview

Complete UI redesign for the Social blueprint, inspired by TikTok, Instagram, and Kwai - focusing on clean aesthetics, modern typography, engaging interactions, and mobile-first responsive design.

## Current State Analysis

### Issues with Current Design
1. **Typography**: Generic system fonts, inconsistent sizing, limited hierarchy
2. **Colors**: Twitter/X clone aesthetic (dark with blue accent) - not distinctive
3. **Layout**: Rigid 3-column grid, lacks visual breathing room
4. **Components**: Basic styling, no depth/shadows, minimal visual feedback
5. **Interactions**: No animations, abrupt state changes
6. **Icons**: Text-only buttons, no visual icons

## Design Goals

1. **Modern & Distinctive**: Stand out from Twitter clones with unique visual identity
2. **Engaging**: Smooth animations, satisfying interactions, visual delight
3. **Clean & Minimal**: Focus on content, reduce visual noise
4. **Mobile-First**: Excellent mobile experience, natural gestures
5. **Accessible**: Proper contrast, readable typography, keyboard navigation

---

## Design System

### Color Palette

```css
/* Primary - Warm gradient accent (Instagram/TikTok inspired) */
--accent-start: #f97316;      /* Orange */
--accent-end: #ec4899;        /* Pink */
--accent-gradient: linear-gradient(135deg, var(--accent-start), var(--accent-end));

/* Neutral - Softer dark theme */
--bg-primary: #0a0a0a;        /* Near black */
--bg-secondary: #141414;      /* Card background */
--bg-tertiary: #1f1f1f;       /* Elevated surfaces */
--bg-hover: #262626;          /* Hover states */

/* Text */
--text-primary: #fafafa;      /* White text */
--text-secondary: #a3a3a3;    /* Muted text */
--text-tertiary: #737373;     /* Subtle text */

/* Semantic */
--success: #22c55e;           /* Green */
--danger: #ef4444;            /* Red */
--warning: #eab308;           /* Yellow */
--info: #3b82f6;              /* Blue */

/* Interactive */
--like-active: #ef4444;       /* Heart red */
--repost-active: #22c55e;     /* Repost green */
--bookmark-active: #3b82f6;   /* Bookmark blue */

/* Border & Divider */
--border: rgba(255, 255, 255, 0.08);
--border-strong: rgba(255, 255, 255, 0.12);
```

### Typography

**Font Stack**: Inter (modern, highly legible, variable font)

```css
/* Font import */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&display=swap');

/* Scale */
--text-xs: 0.75rem;     /* 12px - meta */
--text-sm: 0.875rem;    /* 14px - secondary */
--text-base: 1rem;      /* 16px - body */
--text-lg: 1.125rem;    /* 18px - emphasis */
--text-xl: 1.25rem;     /* 20px - headings */
--text-2xl: 1.5rem;     /* 24px - page titles */
--text-3xl: 1.875rem;   /* 30px - hero */

/* Weights */
--font-normal: 400;
--font-medium: 500;
--font-semibold: 600;
--font-bold: 700;
--font-extrabold: 800;
```

### Spacing & Sizing

```css
/* Spacing scale (4px base) */
--space-1: 0.25rem;   /* 4px */
--space-2: 0.5rem;    /* 8px */
--space-3: 0.75rem;   /* 12px */
--space-4: 1rem;      /* 16px */
--space-5: 1.25rem;   /* 20px */
--space-6: 1.5rem;    /* 24px */
--space-8: 2rem;      /* 32px */
--space-10: 2.5rem;   /* 40px */
--space-12: 3rem;     /* 48px */

/* Border radius */
--radius-sm: 0.375rem;    /* 6px - buttons, inputs */
--radius-md: 0.5rem;      /* 8px - cards */
--radius-lg: 0.75rem;     /* 12px - modals */
--radius-xl: 1rem;        /* 16px - larger cards */
--radius-2xl: 1.5rem;     /* 24px - panels */
--radius-full: 9999px;    /* Pills, avatars */
```

### Shadows & Effects

```css
/* Elevation */
--shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.4);
--shadow-md: 0 4px 6px rgba(0, 0, 0, 0.4);
--shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.4);
--shadow-xl: 0 20px 25px rgba(0, 0, 0, 0.5);

/* Glassmorphism */
--glass-bg: rgba(20, 20, 20, 0.8);
--glass-blur: blur(20px);
--glass-border: 1px solid rgba(255, 255, 255, 0.1);

/* Glow (for buttons/icons) */
--glow-accent: 0 0 20px rgba(249, 115, 22, 0.3);
```

### Icons

Use **Lucide Icons** (modern, consistent, open-source):
```html
<script src="https://unpkg.com/lucide@latest/dist/umd/lucide.min.js"></script>
```

Key icons:
- Home: `home`
- Explore: `compass`
- Notifications: `bell`
- Search: `search`
- Bookmarks: `bookmark`
- Lists: `list`
- Settings: `settings`
- Post: `plus`, `pen-square`
- Like: `heart`
- Repost: `repeat-2`
- Reply: `message-circle`
- Share: `share`

---

## Layout Architecture

### Base Layout (Desktop)

```
┌─────────────────────────────────────────────────────────────────┐
│                         Sticky Header                            │
│  [Logo]              [Search]              [Avatar] [Compose]    │
├──────────────────┬────────────────────────┬─────────────────────┤
│   Left Sidebar   │     Main Content       │   Right Sidebar     │
│   (Fixed, 240px) │     (Fluid, 600px max) │   (Fixed, 320px)    │
│                  │                        │                      │
│   • Navigation   │   • Page Header        │   • Trending        │
│   • Quick Links  │   • Content Area       │   • Suggestions     │
│                  │                        │   • Who to Follow   │
└──────────────────┴────────────────────────┴─────────────────────┘
```

### Mobile Layout

```
┌────────────────────────────────────┐
│ [Hamburger] Logo      [Search] [+] │  ← Sticky header
├────────────────────────────────────┤
│                                    │
│         Main Content               │
│                                    │
├────────────────────────────────────┤
│ [Home] [Explore] [+] [Notif] [Me]  │  ← Bottom nav
└────────────────────────────────────┘
```

---

## Component Designs

### 1. Navigation (Left Sidebar)

**Desktop**: Vertical icon + text nav with subtle hover backgrounds
- Active state: gradient text + indicator line
- Hover: subtle bg fill
- Compose button: gradient fill, subtle glow on hover

**Mobile**: Bottom tab bar with 5 items
- Center item is prominent compose button
- Icons only, active indicator dot

### 2. Post Card

```
┌─────────────────────────────────────────────────────┐
│  [Avatar]  DisplayName  @username  •  2h            │
│                                                     │
│  Post content here with proper line spacing and     │
│  word wrapping. Hashtags and @mentions are          │
│  highlighted with gradient colors.                  │
│                                                     │
│  [Media Preview - rounded corners, max height]      │
│                                                     │
│  ──────────────────────────────────────────────     │
│  [Reply 12]    [Repost 5]    [Heart 42]    [Save]   │
└─────────────────────────────────────────────────────┘
```

Features:
- Card with subtle border, lifts on hover
- Avatar with online indicator option
- Animated action buttons (heart fills, repost bounces)
- Media grid with rounded corners
- Smooth transitions

### 3. Profile Header

```
┌─────────────────────────────────────────────────────┐
│     Cover Photo (gradient if none)                  │
│                                                     │
├─────────────────────────────────────────────────────┤
│ [Large Avatar]                                      │
│                        [Edit Profile] or [Follow]   │
│ Display Name  ✓                                     │
│ @username                                           │
│                                                     │
│ Bio text here, can be multiple lines               │
│                                                     │
│ 📍 Location  🔗 website.com  📅 Joined Jan 2024    │
│                                                     │
│ 1.2K Following    5.4K Followers                   │
├─────────────────────────────────────────────────────┤
│  [Posts]    [Replies]    [Media]    [Likes]         │
└─────────────────────────────────────────────────────┘
```

### 4. Auth Pages

**Login/Register**: Full-screen centered card with:
- Gradient background (subtle)
- Floating card with shadow
- Large logo at top
- Clean input fields with focus states
- Gradient submit button
- Social login options (styled)

### 5. Compose Modal

- Overlay with backdrop blur
- Avatar + textarea
- Character count (circular progress)
- Media attachment options
- Gradient post button
- Close with X or click outside

---

## Animation & Interactions

### Transitions

```css
--transition-fast: 150ms ease;
--transition-base: 200ms ease;
--transition-slow: 300ms ease;
--transition-spring: 300ms cubic-bezier(0.34, 1.56, 0.64, 1);
```

### Key Animations

1. **Like Button**: Heart scales up + color fill with particles
2. **Repost Button**: Rotation + scale bounce
3. **Follow Button**: Morph from "Follow" to "Following"
4. **Post Appear**: Fade in + slide up
5. **Page Transitions**: Cross-fade between pages
6. **Loading States**: Skeleton screens with shimmer

---

## File Changes

### Files to Modify

1. **`static/css/style.css`** - Complete rewrite with new design system
2. **`views/default/layouts/base.html`** - New layout structure, icons, mobile nav
3. **`views/default/pages/home.html`** - Updated post rendering, animations
4. **`views/default/pages/profile.html`** - New profile header design
5. **`views/default/pages/login.html`** - Modern auth page
6. **`views/default/pages/register.html`** - Modern auth page
7. **`views/default/pages/explore.html`** - Discovery-focused layout
8. **`views/default/pages/notifications.html`** - Grouped notifications
9. **`views/default/pages/search.html`** - Improved search UI
10. **`views/default/pages/settings.html`** - Settings redesign
11. **`views/default/pages/post.html`** - Thread view improvements
12. **`views/default/pages/bookmarks.html`** - Grid/list toggle
13. **`views/default/pages/lists.html`** - List cards
14. **`views/default/pages/404.html`** - Friendly error page

### Files to Add

1. **`static/js/animations.js`** - Shared animation utilities
2. **`views/default/components/post.html`** - Reusable post component
3. **`views/default/components/avatar.html`** - Avatar component
4. **`views/default/components/button.html`** - Button variants

---

## Implementation Order

1. **Phase 1: Foundation**
   - [ ] CSS design system (variables, base styles)
   - [ ] Base layout template with new structure
   - [ ] Icon integration (Lucide)

2. **Phase 2: Core Pages**
   - [ ] Home page with new post cards
   - [ ] Profile page with new header
   - [ ] Auth pages (login/register)

3. **Phase 3: Discovery**
   - [ ] Explore page
   - [ ] Search page
   - [ ] Tag page

4. **Phase 4: Features**
   - [ ] Notifications
   - [ ] Bookmarks
   - [ ] Lists
   - [ ] Settings

5. **Phase 5: Polish**
   - [ ] Animations
   - [ ] Loading states
   - [ ] Error states
   - [ ] Mobile optimizations

---

## Responsive Breakpoints

```css
/* Mobile first approach */
@media (min-width: 640px) { /* sm - Tablet */ }
@media (min-width: 768px) { /* md - Small laptop */ }
@media (min-width: 1024px) { /* lg - Desktop */ }
@media (min-width: 1280px) { /* xl - Large desktop */ }
```

---

## Success Criteria

1. ✓ Distinctive visual identity (not a Twitter clone)
2. ✓ Smooth, delightful interactions
3. ✓ Excellent mobile experience
4. ✓ Consistent component design
5. ✓ Fast perceived performance (loading states)
6. ✓ Accessible (WCAG AA contrast)
7. ✓ Modern typography and spacing
