# Forum UI Implementation Plan

**Date:** 2025-12-22
**Status:** In Progress
**Goal:** Create a modern, consistent, production-ready forum UI with excellent developer experience

---

## Executive Summary

This specification outlines the complete UI implementation for the Mizu forum blueprint. The design focuses on:
- **Modern aesthetics**: Clean, minimal design inspired by modern forums (Reddit, Discourse, HackerNews)
- **Excellent DX**: Template-based architecture with reusable components
- **Responsive design**: Mobile-first approach
- **Accessibility**: Semantic HTML, ARIA labels, keyboard navigation
- **Performance**: Minimal JavaScript, progressive enhancement
- **Consistency**: Design system with consistent spacing, colors, typography

---

## Sitemap & Page Structure

### Public Pages (No Auth Required)

1. **Home** `/`
   - Hero section with site description
   - List of top-level forums/categories
   - Trending threads sidebar
   - Recent activity feed
   - Login/Register CTA for non-authenticated users

2. **Forum Category** `/f/{slug}`
   - Forum header (name, description, member count, join button)
   - Subforums list (if nested)
   - Thread list with sorting (hot, new, top, rising)
   - Sticky threads pinned at top
   - Pagination
   - "New Thread" button (auth required)

3. **Thread Detail** `/f/{slug}/t/{id}/{optional-slug}`
   - Thread header (title, author, timestamp, vote controls)
   - Thread content (markdown rendered)
   - Thread metadata (views, score, tags, locked/sticky status)
   - Comment tree (nested replies)
   - Comment form (auth required)
   - Sorting options (best, new, controversial)

4. **User Profile** `/u/{username}`
   - User info (avatar, display name, join date, karma)
   - Activity tabs: Overview, Posts, Comments, Saved
   - User statistics
   - Follow button (auth required)

### Auth Pages

5. **Login** `/login`
   - Username/email + password form
   - Remember me checkbox
   - Links to register and password reset
   - Social login placeholders

6. **Register** `/register`
   - Username, email, password fields
   - Display name (optional)
   - Terms acceptance checkbox
   - Link to login

### Authenticated Pages

7. **Submit Thread** `/f/{slug}/submit`
   - Forum selector
   - Thread type selector (discussion, question, poll)
   - Title input
   - Content editor (markdown)
   - Tags input
   - NSFW/Spoiler toggles
   - Preview mode

8. **Edit Thread** `/f/{slug}/t/{id}/edit`
   - Same as submit but pre-filled

9. **User Settings** `/settings`
   - Profile settings
   - Account settings (email, password)
   - Notification preferences
   - Privacy settings

10. **Notifications** `/notifications`
    - List of notifications (replies, mentions, votes)
    - Mark as read functionality

### Moderator Pages

11. **Moderation Queue** `/f/{slug}/mod/queue`
    - Reported posts/threads
    - Approval queue (if enabled)

12. **Forum Settings** `/f/{slug}/settings`
    - Forum configuration
    - Rules management
    - Moderator management

### Error Pages

13. **404 Not Found** `/404`
14. **403 Forbidden** `/403`
15. **500 Server Error** `/500`

---

## Design System

### Color Palette

```css
/* Light mode (default) */
--color-primary: #0066cc;        /* Links, primary actions */
--color-primary-hover: #0052a3;
--color-secondary: #6366f1;      /* Accents */
--color-success: #10b981;        /* Upvotes, success messages */
--color-danger: #ef4444;         /* Downvotes, destructive actions */
--color-warning: #f59e0b;        /* Warnings, NSFW */

--color-bg: #ffffff;             /* Page background */
--color-bg-alt: #f9fafb;         /* Alternate backgrounds */
--color-bg-card: #ffffff;        /* Card backgrounds */
--color-border: #e5e7eb;         /* Borders */

--color-text: #111827;           /* Primary text */
--color-text-muted: #6b7280;     /* Secondary text */
--color-text-subtle: #9ca3af;    /* Tertiary text */

/* Dark mode */
@media (prefers-color-scheme: dark) {
  --color-primary: #3b82f6;
  --color-primary-hover: #60a5fa;

  --color-bg: #0f172a;
  --color-bg-alt: #1e293b;
  --color-bg-card: #1e293b;
  --color-border: #334155;

  --color-text: #f1f5f9;
  --color-text-muted: #cbd5e1;
  --color-text-subtle: #94a3b8;
}
```

### Typography

```css
--font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
--font-mono: "SF Mono", Monaco, "Cascadia Code", "Roboto Mono", Consolas, monospace;

--text-xs: 0.75rem;    /* 12px */
--text-sm: 0.875rem;   /* 14px */
--text-base: 1rem;     /* 16px */
--text-lg: 1.125rem;   /* 18px */
--text-xl: 1.25rem;    /* 20px */
--text-2xl: 1.5rem;    /* 24px */
--text-3xl: 1.875rem;  /* 30px */
```

### Spacing

```css
--space-1: 0.25rem;   /* 4px */
--space-2: 0.5rem;    /* 8px */
--space-3: 0.75rem;   /* 12px */
--space-4: 1rem;      /* 16px */
--space-5: 1.25rem;   /* 20px */
--space-6: 1.5rem;    /* 24px */
--space-8: 2rem;      /* 32px */
--space-10: 2.5rem;   /* 40px */
--space-12: 3rem;     /* 48px */
```

### Components

- Border radius: `4px` (small), `8px` (medium), `12px` (large)
- Shadows: Subtle elevation for cards
- Transitions: `150ms ease-in-out` for interactions

---

## Template Architecture

### Directory Structure

```
assets/views/
├── layouts/
│   ├── default.html          # Base layout (existing)
│   ├── auth.html             # Minimal layout for login/register
│   └── minimal.html          # Minimal layout for errors
├── pages/
│   ├── home.html             # Homepage
│   ├── forum.html            # Forum category page
│   ├── thread.html           # Thread detail page
│   ├── submit.html           # Submit thread page
│   ├── profile.html          # User profile page
│   ├── settings.html         # User settings
│   ├── notifications.html    # Notifications page
│   ├── login.html            # Login page
│   ├── register.html         # Register page
│   ├── moderation.html       # Moderation queue
│   ├── 404.html              # Not found
│   ├── 403.html              # Forbidden
│   └── 500.html              # Server error
├── components/
│   ├── header.html           # Site header with navigation
│   ├── footer.html           # Site footer
│   ├── sidebar.html          # Sidebar for trending/info
│   ├── forum_card.html       # Forum card display
│   ├── thread_card.html      # Thread list item
│   ├── post_card.html        # Comment/post display
│   ├── vote_controls.html    # Upvote/downvote buttons
│   ├── user_badge.html       # User avatar + name
│   ├── markdown_editor.html  # Markdown editor component
│   ├── tag_list.html         # Tag display
│   ├── breadcrumb.html       # Breadcrumb navigation
│   ├── pagination.html       # Pagination controls
│   └── notification.html     # Toast/flash notification
└── partials/
    ├── meta.html             # Meta tags
    └── scripts.html          # Common scripts
```

### Layout Hierarchy

1. **layouts/default.html** - Main layout
   - Includes header, footer
   - Provides navigation
   - Shows user auth state
   - Flash messages
   - Content block

2. **layouts/auth.html** - Auth pages
   - Minimal header
   - Centered content
   - No navigation

3. **layouts/minimal.html** - Error pages
   - Minimal styling
   - Centered error message

---

## Component Specifications

### Header Component

**File:** `components/header.html`

**Features:**
- Logo/brand (links to home)
- Search bar (optional, v2)
- Navigation links
- User menu (if authenticated)
  - Profile link
  - Settings
  - Notifications (with count badge)
  - Logout
- Login/Register buttons (if not authenticated)

**Responsive:**
- Mobile: Hamburger menu
- Desktop: Full navigation bar

---

### Thread Card Component

**File:** `components/thread_card.html`

**Props:**
- Thread object
- Show forum badge (yes/no)
- Compact mode (yes/no)

**Layout:**
```
[Vote] [Content Block              ] [Meta]
 ↑↓     Title (with flair)            42 💬
 123    by @user · 2h ago in /f/tech  👁 1.2k
        Preview text...
        [tag1] [tag2] [sticky] [locked]
```

**Features:**
- Vote controls (upvote/downvote)
- Title (with link to thread)
- Author + timestamp
- Forum badge (if cross-forum view)
- Preview snippet (first 200 chars)
- Tags
- Status badges (sticky, locked, NSFW)
- Metadata (comments, views)

---

### Post Card Component

**File:** `components/post_card.html`

**Props:**
- Post object
- Depth level (for indentation)
- Collapsed state

**Layout:**
```
@username · 2h ago · edited
[vote] Post content goes here with markdown rendered
 ↑↓
 42     [Reply] [Edit] [Delete] [Save] [Report]
        └─ Nested replies (collapsed/expanded)
```

**Features:**
- User avatar + name
- Timestamp + edited indicator
- Vote controls
- Markdown-rendered content
- Action buttons (reply, edit, delete, save, report)
- Nested reply indicator
- Collapse/expand for long threads
- Best answer badge (if applicable)

---

### Forum Card Component

**File:** `components/forum_card.html`

**Props:**
- Forum object
- Show description (yes/no)

**Layout:**
```
[Icon] Forum Name                    [Join Button]
       Brief description
       42k members · 123 active now
```

**Features:**
- Forum icon
- Name + link
- Description
- Member count
- Active users count
- Join/Leave button (if authenticated)
- Moderator badge (if user is mod)

---

### Vote Controls Component

**File:** `components/vote_controls.html`

**Props:**
- Item type (thread/post)
- Item ID
- Current score
- User's vote (-1, 0, 1)

**Layout:**
```
 ↑   upvote arrow
123  score
 ↓   downvote arrow
```

**Features:**
- Upvote button (highlighted if voted)
- Score display
- Downvote button (highlighted if voted)
- Disabled state for non-authenticated users
- API integration for vote submission

---

## Page Specifications

### Home Page (`pages/home.html`)

**Sections:**
1. Hero
   - Site name + tagline
   - Quick stats (users, threads, posts)

2. Forum List
   - Top-level forums grouped by category
   - Each forum shows: icon, name, description, member count
   - Join button for each forum

3. Sidebar
   - Trending threads today
   - Active users
   - Quick links

**Data Required:**
- List of forums (with stats)
- Trending threads (top 5)
- Site statistics

---

### Forum Page (`pages/forum.html`)

**Sections:**
1. Forum Header
   - Forum name + icon
   - Description
   - Subscribe button
   - Member count
   - Moderators list

2. Subforums (if any)
   - List of child forums

3. Thread List
   - Sorting tabs (hot, new, top, rising)
   - Sticky threads at top
   - Regular threads
   - Pagination

4. Sidebar
   - Forum rules
   - About section
   - Moderators

**Data Required:**
- Forum object (with stats)
- List of threads (paginated)
- Subforums (if any)
- Forum rules
- Moderators

---

### Thread Page (`pages/thread.html`)

**Sections:**
1. Breadcrumb
   - Home → Forum → Thread

2. Thread Header
   - Vote controls
   - Title
   - Author info
   - Timestamp
   - View count
   - Tags

3. Thread Content
   - Markdown-rendered content
   - Edit/Delete buttons (if owner)
   - Lock/Sticky buttons (if moderator)

4. Comments Section
   - Sort options (best, new, old, controversial)
   - Comment tree
   - Reply form

5. Sidebar
   - Forum info
   - Related threads
   - Thread stats

**Data Required:**
- Thread object (with author, forum)
- List of posts/comments (tree structure)
- User's votes on thread/posts
- Forum rules

---

### Profile Page (`pages/profile.html`)

**Sections:**
1. Profile Header
   - Avatar
   - Username + display name
   - Karma scores (post/comment)
   - Join date
   - Bio
   - Follow button

2. Tabs
   - Overview (recent activity)
   - Threads (authored threads)
   - Comments (authored comments)
   - Saved (bookmarked items)

3. Activity Feed
   - Mixed list of threads/comments
   - Pagination

**Data Required:**
- Account object
- User's threads (paginated)
- User's comments (paginated)
- User statistics

---

## Handler Updates

### New/Updated Handlers

**File:** `app/web/handler/pages.go` (new file)

```go
type Pages struct {
    templates *template.Template
    forums    forums.API
    threads   threads.API
    posts     posts.API
    accounts  accounts.API
    votes     votes.API
}

func NewPages(...) *Pages

// Page handlers
func (h *Pages) Home(c *mizu.Ctx) error
func (h *Pages) ForumPage(c *mizu.Ctx) error
func (h *Pages) ThreadPage(c *mizu.Ctx) error
func (h *Pages) SubmitPage(c *mizu.Ctx) error
func (h *Pages) EditThreadPage(c *mizu.Ctx) error
func (h *Pages) ProfilePage(c *mizu.Ctx) error
func (h *Pages) SettingsPage(c *mizu.Ctx) error
func (h *Pages) NotificationsPage(c *mizu.Ctx) error
func (h *Pages) ModerationPage(c *mizu.Ctx) error

// Auth pages
func (h *Pages) LoginPage(c *mizu.Ctx) error
func (h *Pages) RegisterPage(c *mizu.Ctx) error

// Error pages
func (h *Pages) NotFound(c *mizu.Ctx) error
func (h *Pages) Forbidden(c *mizu.Ctx) error
func (h *Pages) ServerError(c *mizu.Ctx) error
```

**Template Data Structures:**

```go
type HomeData struct {
    Forums    []*forums.Forum
    Trending  []*threads.Thread
    Stats     *SiteStats
    User      *accounts.Account  // Current user (if authenticated)
}

type ForumPageData struct {
    Forum      *forums.Forum
    Subforums  []*forums.Forum
    Threads    []*threads.Thread
    Sort       string
    Page       int
    User       *accounts.Account
}

type ThreadPageData struct {
    Forum      *forums.Forum
    Thread     *threads.Thread
    Posts      []*posts.Post  // Tree structure
    Sort       string
    User       *accounts.Account
    UserVotes  map[string]int  // postID -> vote
}

type ProfilePageData struct {
    Account   *accounts.Account
    Threads   []*threads.Thread
    Posts     []*posts.Post
    Tab       string
    User      *accounts.Account  // Current user
}
```

---

## Route Updates

**File:** `app/web/server.go` - `setupRoutes()`

```go
// Create pages handler
pagesHandler := handler.NewPages(tmpl, accountsSvc, forumsSvc, threadsSvc, postsSvc, votesSvc)

// Public pages
s.app.Get("/", pagesHandler.Home)
s.app.Get("/f/{slug}", pagesHandler.ForumPage)
s.app.Get("/f/{slug}/t/{id}", pagesHandler.ThreadPage)
s.app.Get("/f/{slug}/t/{id}/{slug}", pagesHandler.ThreadPage)  // SEO-friendly URL
s.app.Get("/u/{username}", pagesHandler.ProfilePage)

// Auth pages
s.app.Get("/login", pagesHandler.LoginPage)
s.app.Get("/register", pagesHandler.RegisterPage)

// Authenticated pages
s.app.Get("/f/{slug}/submit", s.authRequired(pagesHandler.SubmitPage))
s.app.Get("/f/{slug}/t/{id}/edit", s.authRequired(pagesHandler.EditThreadPage))
s.app.Get("/settings", s.authRequired(pagesHandler.SettingsPage))
s.app.Get("/notifications", s.authRequired(pagesHandler.NotificationsPage))

// Moderator pages
s.app.Get("/f/{slug}/mod/queue", s.moderatorRequired(pagesHandler.ModerationPage))
s.app.Get("/f/{slug}/settings", s.moderatorRequired(pagesHandler.ForumSettingsPage))

// Error handlers
s.app.NotFoundHandler(pagesHandler.NotFound)
```

---

## JavaScript Architecture

### Minimal JavaScript Approach

**Progressive Enhancement Strategy:**
- All core functionality works without JavaScript
- JavaScript enhances UX (live voting, inline editing, etc.)

**File:** `assets/static/js/app.js`

**Modules:**
1. **Vote Handler**
   - Intercept vote button clicks
   - Send API request
   - Update UI optimistically
   - Rollback on error

2. **Comment Form**
   - Inline reply forms
   - Markdown preview
   - Auto-save drafts

3. **Infinite Scroll**
   - Optional on thread lists
   - Load more threads/posts

4. **Notification Poller**
   - Poll for new notifications
   - Update badge count

5. **Toast Notifications**
   - Show success/error messages
   - Auto-dismiss after 3s

**Dependencies:**
- None (vanilla JS)
- Consider Alpine.js for reactivity (optional)

---

## CSS Architecture

**File:** `assets/static/css/app.css`

**Structure:**
1. **Reset/Base**
   - Normalize styles
   - CSS custom properties (design tokens)
   - Typography base styles

2. **Layout**
   - Container, grid, flexbox utilities
   - Responsive breakpoints

3. **Components**
   - Header, footer, sidebar
   - Cards, buttons, forms
   - Thread cards, post cards
   - Vote controls

4. **Pages**
   - Page-specific styles
   - Home, forum, thread, profile

5. **Utilities**
   - Spacing utilities (margin, padding)
   - Text utilities (color, size, weight)
   - Display utilities

**Approach:**
- BEM naming convention for components
- Utility classes for common patterns
- Mobile-first responsive design
- Dark mode via CSS custom properties

---

## E2E Test Plan

**File:** `app/web/server_e2e_test.go`

### New Test Cases

1. **TestPageHome**
   - Verify home page loads
   - Check forums list is present
   - Verify HTML structure

2. **TestPageForum**
   - Navigate to forum page
   - Verify thread list loads
   - Check sorting options

3. **TestPageThread**
   - Navigate to thread page
   - Verify thread content renders
   - Check comments are displayed
   - Verify markdown rendering

4. **TestPageProfile**
   - Navigate to user profile
   - Verify user info displays
   - Check tabs work

5. **TestPageLoginRegister**
   - Verify login page renders
   - Verify register page renders
   - Check forms are present

6. **TestVoteWorkflow**
   - Create thread
   - Upvote thread
   - Verify score updates
   - Downvote thread
   - Verify score updates

7. **TestThreadCreation**
   - Login
   - Navigate to submit page
   - Create thread
   - Verify redirect to thread page
   - Verify thread displays correctly

8. **TestCommentWorkflow**
   - Login
   - Navigate to thread
   - Post comment
   - Verify comment appears
   - Reply to comment
   - Verify nested reply

9. **TestForumJoinLeave**
   - Login
   - Join forum
   - Verify member count increments
   - Leave forum
   - Verify member count decrements

10. **TestResponsiveDesign**
    - Test pages at different viewport sizes
    - Verify mobile navigation works

---

## Implementation Phases

### Phase 1: Foundation (Core Layout & Components)
**Time:** Day 1
1. Update `layouts/default.html` with modern structure
2. Create `components/header.html`
3. Create `components/footer.html`
4. Create `components/sidebar.html`
5. Build CSS design system in `app.css`
6. Implement responsive navigation

**Deliverables:**
- Modern base layout
- Reusable header/footer components
- CSS design system with variables
- Responsive navigation

---

### Phase 2: Forum & Thread Pages
**Time:** Day 2
1. Create `pages/home.html`
2. Create `pages/forum.html`
3. Create `pages/thread.html`
4. Create `components/forum_card.html`
5. Create `components/thread_card.html`
6. Create `components/post_card.html`
7. Create `components/vote_controls.html`
8. Implement handlers in `handler/pages.go`
9. Update routes in `server.go`

**Deliverables:**
- Fully functional home page with forum list
- Forum category page with thread list
- Thread detail page with comments
- Vote controls

---

### Phase 3: User Features
**Time:** Day 3
1. Create `pages/profile.html`
2. Create `pages/submit.html`
3. Create `pages/settings.html`
4. Create `components/user_badge.html`
5. Create `components/markdown_editor.html`
6. Implement user page handlers
7. Add tests for user workflows

**Deliverables:**
- User profile page
- Thread submission page
- User settings page
- Profile editing

---

### Phase 4: Auth & Interactions
**Time:** Day 4
1. Create `pages/login.html`
2. Create `pages/register.html`
3. Create `layouts/auth.html`
4. Implement JavaScript vote handler
5. Implement comment form interactions
6. Add flash message system
7. Add client-side form validation

**Deliverables:**
- Modern login/register pages
- Interactive voting
- Live comment posting
- Flash notifications

---

### Phase 5: Polish & Testing
**Time:** Day 5
1. Create error pages (404, 403, 500)
2. Add loading states
3. Optimize CSS (remove unused styles)
4. Add animations/transitions
5. Write comprehensive E2E tests
6. Accessibility audit (WCAG 2.1 AA)
7. Performance optimization
8. Cross-browser testing

**Deliverables:**
- Error pages
- Comprehensive E2E test suite
- Accessibility compliance
- Performance optimization
- Production-ready UI

---

## Success Criteria

- [ ] All pages render without errors
- [ ] Responsive design works on mobile, tablet, desktop
- [ ] Dark mode support
- [ ] All E2E tests pass
- [ ] No JavaScript errors in console
- [ ] Lighthouse score > 90 (performance, accessibility, best practices)
- [ ] Works in Chrome, Firefox, Safari, Edge
- [ ] Keyboard navigation works for all interactive elements
- [ ] Screen reader compatible
- [ ] Forum workflows complete (browse, post, comment, vote)
- [ ] User workflows complete (register, login, profile, settings)

---

## Future Enhancements (v2)

- Real-time updates via WebSockets/SSE
- Rich text editor (WYSIWYG)
- Image upload and hosting
- Poll creation and voting
- User mentions and autocomplete
- Advanced search
- Moderation tools (ban users, remove content)
- Email notifications
- User avatars and customization
- Theme customization
- i18n support
- Mobile app (PWA)

---

## Technical Decisions

### Why Template-Based Over SPA?

1. **Simplicity**: Server-rendered HTML is easier to reason about
2. **SEO**: Better search engine indexing
3. **Performance**: Faster initial page load
4. **Progressive Enhancement**: Works without JavaScript
5. **Fits Mizu Philosophy**: Lightweight, Go-native approach

### Why Minimal JavaScript?

1. **Accessibility**: Server-rendered content is more accessible
2. **Performance**: Less client-side computation
3. **Maintainability**: Less code to maintain
4. **Reliability**: Works even if JS fails to load

### Why No CSS Framework?

1. **Learning**: Custom CSS teaches better practices
2. **Performance**: No unused CSS
3. **Flexibility**: Full control over design
4. **Bundle Size**: Smaller CSS files
5. **Uniqueness**: Avoid "Bootstrap look"

---

## Conclusion

This plan provides a comprehensive roadmap for building a modern, production-ready forum UI. The template-based architecture ensures excellent performance and SEO while the component-based design promotes reusability and consistency. The phased implementation allows for iterative development and testing, ensuring quality at each step.

The result will be a forum that rivals modern platforms in aesthetics and usability while maintaining the simplicity and performance advantages of server-rendered templates.
