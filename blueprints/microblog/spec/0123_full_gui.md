# Microblog Full GUI Design Specification

## Overview

This specification defines the complete GUI implementation for the Microblog application - a self-hosted microblogging platform inspired by X/Twitter, Threads, and Mastodon.

## Design Philosophy

- **Modern & Clean**: Minimalist design with ample whitespace, subtle shadows, and rounded corners
- **Responsive**: Mobile-first approach, works on all screen sizes
- **Accessible**: High contrast, keyboard navigation, ARIA labels
- **Fast**: Optimistic UI updates, lazy loading, minimal JavaScript
- **Consistent**: Unified color palette, typography, and spacing

## Color Palette

```
Primary:        #3B82F6 (Blue-500)
Primary Dark:   #2563EB (Blue-600)
Primary Light:  #EFF6FF (Blue-50)

Success:        #22C55E (Green-500)
Warning:        #F59E0B (Amber-500)
Error:          #EF4444 (Red-500)

Background:     #F8FAFC (Slate-50)
Surface:        #FFFFFF (White)
Border:         #E2E8F0 (Slate-200)

Text Primary:   #0F172A (Slate-900)
Text Secondary: #64748B (Slate-500)
Text Muted:     #94A3B8 (Slate-400)
```

## Typography

```
Font Family:    Inter, system-ui, -apple-system, sans-serif
Heading:        font-weight: 700
Subheading:     font-weight: 600
Body:           font-weight: 400
Small:          font-size: 0.875rem
Extra Small:    font-size: 0.75rem
```

---

## Site Map

```
/                           Home Timeline
├── /login                  Login Page
├── /register               Registration Page
├── /explore                Explore/Discover
│   └── /explore/trending   Trending Topics
├── /search                 Search Results
│   └── /search?q={query}   Search with query
├── /tags/{tag}             Hashtag Timeline
├── /notifications          Notifications Center
├── /bookmarks              Saved Bookmarks
├── /settings               Settings Hub
│   ├── /settings/profile   Profile Settings
│   ├── /settings/account   Account Settings
│   ├── /settings/privacy   Privacy Settings
│   └── /settings/appearance Appearance Settings
├── /@{username}            User Profile
│   ├── /@{username}/followers   Follower List
│   ├── /@{username}/following   Following List
│   └── /@{username}/likes       Liked Posts
└── /@{username}/{post_id}  Single Post / Thread View
```

---

## User Interactions & Features

### 1. Authentication Flow

| Action | Trigger | Behavior |
|--------|---------|----------|
| Login | Submit form | Validate credentials, store token, redirect to home |
| Register | Submit form | Create account, auto-login, redirect to home |
| Logout | Click logout | Clear token, redirect to home |
| Session expired | API 401 | Show login modal or redirect |

### 2. Post Interactions

| Action | Trigger | Behavior |
|--------|---------|----------|
| Create post | Submit compose | Optimistic add to timeline, API call |
| Reply to post | Click reply | Open reply composer with context |
| Like post | Click heart | Toggle color, update count optimistically |
| Unlike post | Click filled heart | Toggle color, update count optimistically |
| Repost | Click repost | Toggle color, update count, add to profile |
| Unrepost | Click filled repost | Remove from profile |
| Bookmark | Click bookmark | Toggle color, add to bookmarks |
| Delete post | Click delete (own) | Confirm modal, remove from UI |
| Edit post | Click edit (own) | Open editor with existing content |
| Quote post | Click quote | Open composer with quote attachment |
| Share post | Click share | Copy link / native share dialog |

### 3. User Interactions

| Action | Trigger | Behavior |
|--------|---------|----------|
| Follow user | Click follow button | Update button state, increment count |
| Unfollow user | Click unfollow | Confirm if enabled, update UI |
| Block user | Profile menu | Hide user's content, prevent interaction |
| Mute user | Profile menu | Hide from timelines, keep follow |
| Report user | Profile menu | Open report modal |
| View followers | Click follower count | Navigate to followers list |
| View following | Click following count | Navigate to following list |

### 4. Timeline Interactions

| Action | Trigger | Behavior |
|--------|---------|----------|
| Load more | Scroll to bottom | Fetch older posts, append |
| Refresh | Pull down / click | Fetch new posts, prepend |
| Switch timeline | Click tab | Switch between home/explore |
| Filter timeline | Select filter | Apply visibility filter |

### 5. Search & Discovery

| Action | Trigger | Behavior |
|--------|---------|----------|
| Search | Submit query | Show results by type |
| Filter results | Click tab | Filter by accounts/posts/tags |
| View trending | Click trending item | Navigate to hashtag page |
| Clear search | Click X | Reset to explore page |

### 6. Notification Interactions

| Action | Trigger | Behavior |
|--------|---------|----------|
| View notifications | Click bell | Navigate to notifications page |
| Mark as read | Click notification | Mark single as read |
| Mark all read | Click button | Mark all as read |
| Filter by type | Click filter | Show only selected types |

---

## Page Designs

### Layout Structure

```
┌──────────────────────────────────────────────────────────────┐
│  Header (Nav)                                                │
├──────────────────────────────────────────────────────────────┤
│        │                    │                                │
│  Nav   │   Main Content     │   Sidebar                      │
│ (Left) │                    │  (Right)                       │
│        │                    │                                │
│  Home  │   [Page Content]   │  - Search                      │
│  Explore                    │  - Trending                    │
│  Notif │                    │  - Suggestions                 │
│  Book  │                    │  - Footer                      │
│  Prof  │                    │                                │
│  More  │                    │                                │
│        │                    │                                │
└──────────────────────────────────────────────────────────────┘
```

### Mobile Layout (< 768px)

```
┌─────────────────────┐
│  Header             │
├─────────────────────┤
│                     │
│   Main Content      │
│                     │
│                     │
│                     │
├─────────────────────┤
│  Bottom Nav Bar     │
└─────────────────────┘
```

---

## Component Designs

### 1. Post Card (`components/post_card.html`)

```
┌────────────────────────────────────────────────────┐
│ [Avatar] Display Name @username · 2h      [···]    │
│                                                    │
│ Post content goes here. Can include #hashtags     │
│ and @mentions that are clickable links.           │
│                                                    │
│ [Media Grid - if present]                          │
│                                                    │
│ [Quote Card - if quoting another post]             │
│                                                    │
│ [💬 12]  [🔄 5]  [❤️ 42]  [🔖]  [↗️]              │
└────────────────────────────────────────────────────┘
```

### 2. User Card (`components/user_card.html`)

```
┌────────────────────────────────────────────────────┐
│ [Avatar] Display Name @username    [Follow]        │
│          Bio text preview...                       │
└────────────────────────────────────────────────────┘
```

### 3. Compose Box (`components/compose_box.html`)

```
┌────────────────────────────────────────────────────┐
│ [Avatar] What's happening?                         │
│                                                    │
│ [Textarea - expandable]                            │
│                                                    │
│ [📷] [📊] [😊] [📍]          0/500  [Post]        │
└────────────────────────────────────────────────────┘
```

### 4. Notification Item (`components/notification_item.html`)

```
┌────────────────────────────────────────────────────┐
│ [❤️] User liked your post               2h        │
│      [Post preview...]                             │
└────────────────────────────────────────────────────┘
```

### 5. Trending Item (`components/trending_item.html`)

```
┌────────────────────────────────────────────────────┐
│ Trending in Technology                             │
│ #OpenSource                                        │
│ 1,234 posts                                        │
└────────────────────────────────────────────────────┘
```

---

## Pages Specification

### 1. Home Page (`pages/home.html`)

**Purpose**: Main timeline for logged-in users

**Elements**:
- Left sidebar with navigation
- Compose box (authenticated only)
- Timeline tabs (For You / Following)
- Post list with infinite scroll
- Right sidebar with search, trending, suggestions

**Data Requirements**:
- `Account` - current user (nullable)
- `Posts` - timeline posts
- `TrendingTags` - top hashtags
- `Suggestions` - suggested users

### 2. Login Page (`pages/login.html`)

**Purpose**: User authentication

**Elements**:
- Logo and branding
- Username/email input
- Password input with show/hide toggle
- "Remember me" checkbox
- Submit button
- Link to registration
- Link to password reset

### 3. Register Page (`pages/register.html`)

**Purpose**: New user registration

**Elements**:
- Logo and branding
- Username input with availability check
- Email input
- Password input with strength indicator
- Confirm password input
- Terms acceptance checkbox
- Submit button
- Link to login

### 4. Profile Page (`pages/profile.html`)

**Purpose**: View user profile and posts

**Elements**:
- Header image (banner)
- Avatar with online indicator
- Display name and verification badge
- Username and bio
- Profile fields (links, location)
- Stats (posts, followers, following)
- Follow/Edit Profile button
- Profile tabs (Posts, Replies, Media, Likes)
- Post list

**Data Requirements**:
- `Profile` - viewed user
- `IsOwner` - boolean
- `IsFollowing` - boolean
- `Posts` - user's posts
- `ActiveTab` - current tab

### 5. Post/Thread Page (`pages/post.html`)

**Purpose**: Single post with full thread context

**Elements**:
- Ancestor posts (context chain)
- Featured post (full detail)
- Reply composer (authenticated)
- Descendant posts (replies)
- Engagement stats

**Data Requirements**:
- `Thread.Ancestors` - parent posts
- `Thread.Post` - main post
- `Thread.Descendants` - replies

### 6. Explore Page (`pages/explore.html`)

**Purpose**: Discover content and trends

**Elements**:
- Search bar (prominent)
- Trending topics section
- Trending posts section
- Category tabs (For You, Trending, News, etc.)

**Data Requirements**:
- `TrendingTags` - top hashtags
- `TrendingPosts` - popular posts
- `Categories` - discovery categories

### 7. Hashtag Page (`pages/tag.html`)

**Purpose**: Posts with specific hashtag

**Elements**:
- Tag header with post count
- Post list with hashtag
- Related tags sidebar

**Data Requirements**:
- `Tag` - hashtag name
- `Posts` - posts with tag
- `RelatedTags` - similar tags

### 8. Notifications Page (`pages/notifications.html`)

**Purpose**: User notifications center

**Elements**:
- Filter tabs (All, Mentions, Likes, Follows)
- Notification list grouped by day
- Mark all read button
- Settings link

**Data Requirements**:
- `Notifications` - notification list
- `UnreadCount` - unread count
- `Filter` - current filter

### 9. Bookmarks Page (`pages/bookmarks.html`)

**Purpose**: Saved posts collection

**Elements**:
- Header with count
- Post list
- Empty state when no bookmarks

**Data Requirements**:
- `Posts` - bookmarked posts

### 10. Settings Page (`pages/settings.html`)

**Purpose**: User settings hub

**Elements**:
- Settings navigation sidebar
- Profile settings form
- Account settings form
- Privacy settings
- Appearance settings (theme toggle)
- Danger zone (delete account)

**Data Requirements**:
- `Account` - current user

### 11. Search Page (`pages/search.html`)

**Purpose**: Search results display

**Elements**:
- Search input (pre-filled)
- Result tabs (Top, Accounts, Posts, Tags)
- Result lists per type
- Empty state for no results

**Data Requirements**:
- `Query` - search query
- `Accounts` - matching accounts
- `Posts` - matching posts
- `Hashtags` - matching tags

### 12. Followers/Following Page (`pages/follow_list.html`)

**Purpose**: User relationship lists

**Elements**:
- Header with user info
- Tab toggle (Followers/Following)
- User cards with follow buttons
- Search/filter

**Data Requirements**:
- `Profile` - whose list
- `Users` - list of followers/following
- `ListType` - "followers" or "following"

---

## File Structure

```
assets/
├── static/
│   ├── css/
│   │   └── app.css           # Complete stylesheet
│   ├── js/
│   │   └── app.js            # Client-side JavaScript
│   └── img/
│       └── logo.svg          # Logo and icons
│
└── views/
    ├── layouts/
    │   └── default.html      # Main layout wrapper
    │
    ├── pages/
    │   ├── home.html         # Home timeline
    │   ├── login.html        # Login form
    │   ├── register.html     # Registration form
    │   ├── profile.html      # User profile
    │   ├── post.html         # Single post/thread
    │   ├── explore.html      # Explore/discover
    │   ├── tag.html          # Hashtag timeline
    │   ├── notifications.html # Notifications
    │   ├── bookmarks.html    # Saved bookmarks
    │   ├── settings.html     # Settings pages
    │   ├── search.html       # Search results
    │   └── follow_list.html  # Followers/following
    │
    └── components/
        ├── post_card.html        # Post display
        ├── user_card.html        # User display
        ├── compose_box.html      # Post composer
        ├── notification_item.html # Notification
        ├── trending_item.html    # Trending tag
        ├── nav_sidebar.html      # Left navigation
        ├── right_sidebar.html    # Right sidebar
        ├── modal.html            # Modal wrapper
        ├── toast.html            # Toast notification
        ├── empty_state.html      # Empty states
        └── loading.html          # Loading spinner
```

---

## Handler Refactoring Plan

Split `handlers.go` into focused handler files:

```
app/web/
├── server.go              # Server setup & routes
├── config.go              # Configuration
├── middleware.go          # Middleware functions
├── render.go              # Template rendering helper
│
├── handlers/
│   ├── auth.go            # Login, register, logout
│   ├── accounts.go        # Account CRUD, profile
│   ├── posts.go           # Post CRUD, thread
│   ├── interactions.go    # Like, repost, bookmark
│   ├── relationships.go   # Follow, block, mute
│   ├── timelines.go       # Home, local, hashtag
│   ├── notifications.go   # Notification handlers
│   ├── search.go          # Search handler
│   ├── trending.go        # Trending handlers
│   └── pages.go           # Web page handlers
│
└── helpers.go             # Shared utilities
```

---

## JavaScript Enhancements

### Required Features

1. **Form Validation**
   - Real-time validation feedback
   - Submit prevention on invalid
   - Loading states

2. **Optimistic Updates**
   - Instant UI feedback
   - Rollback on error
   - Count animations

3. **Infinite Scroll**
   - Lazy load on scroll
   - Loading indicator
   - End-of-list detection

4. **Modal System**
   - Compose modal
   - Confirm dialogs
   - Image lightbox
   - Report modal

5. **Toast Notifications**
   - Success/error messages
   - Auto-dismiss
   - Action buttons

6. **Keyboard Shortcuts**
   - `n` - New post
   - `j/k` - Navigate posts
   - `l` - Like
   - `r` - Reply
   - `?` - Show shortcuts

7. **Theme Toggle**
   - Light/dark mode
   - System preference
   - Persist choice

---

## Implementation Order

1. **Phase 1: Foundation**
   - [ ] Complete CSS design system
   - [ ] Layout components (nav, sidebar)
   - [ ] Refactor handlers into separate files

2. **Phase 2: Core Pages**
   - [ ] Home page with compose
   - [ ] Login & register pages
   - [ ] Profile page
   - [ ] Post/thread page

3. **Phase 3: Features**
   - [ ] Explore page
   - [ ] Notifications page
   - [ ] Bookmarks page
   - [ ] Search page

4. **Phase 4: Enhancement**
   - [ ] Settings pages
   - [ ] Follow lists
   - [ ] Modals and toasts
   - [ ] Dark mode

5. **Phase 5: Polish**
   - [ ] Loading states
   - [ ] Error handling
   - [ ] Empty states
   - [ ] Keyboard shortcuts

---

## API Endpoints Reference

### Authentication
- `POST /api/v1/auth/register` - Register new account
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/logout` - Logout

### Accounts
- `GET /api/v1/accounts/verify_credentials` - Current user
- `PATCH /api/v1/accounts/update_credentials` - Update profile
- `GET /api/v1/accounts/{id}` - Get account
- `GET /api/v1/accounts/{id}/posts` - Account posts
- `GET /api/v1/accounts/{id}/followers` - Followers
- `GET /api/v1/accounts/{id}/following` - Following
- `POST /api/v1/accounts/{id}/follow` - Follow
- `POST /api/v1/accounts/{id}/unfollow` - Unfollow
- `POST /api/v1/accounts/{id}/block` - Block
- `POST /api/v1/accounts/{id}/unblock` - Unblock
- `POST /api/v1/accounts/{id}/mute` - Mute
- `POST /api/v1/accounts/{id}/unmute` - Unmute

### Posts
- `POST /api/v1/posts` - Create post
- `GET /api/v1/posts/{id}` - Get post
- `PUT /api/v1/posts/{id}` - Update post
- `DELETE /api/v1/posts/{id}` - Delete post
- `GET /api/v1/posts/{id}/context` - Thread context
- `POST /api/v1/posts/{id}/like` - Like
- `DELETE /api/v1/posts/{id}/like` - Unlike
- `POST /api/v1/posts/{id}/repost` - Repost
- `DELETE /api/v1/posts/{id}/repost` - Unrepost
- `POST /api/v1/posts/{id}/bookmark` - Bookmark
- `DELETE /api/v1/posts/{id}/bookmark` - Unbookmark

### Timelines
- `GET /api/v1/timelines/home` - Home timeline
- `GET /api/v1/timelines/local` - Local timeline
- `GET /api/v1/timelines/tag/{tag}` - Hashtag timeline

### Notifications
- `GET /api/v1/notifications` - List notifications
- `POST /api/v1/notifications/clear` - Clear all
- `POST /api/v1/notifications/{id}/dismiss` - Dismiss one

### Search & Trends
- `GET /api/v1/search?q={query}` - Search
- `GET /api/v1/trends/tags` - Trending tags
- `GET /api/v1/trends/posts` - Trending posts

### Bookmarks
- `GET /api/v1/bookmarks` - User's bookmarks
