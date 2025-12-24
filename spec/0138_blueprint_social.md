# Blueprint: Social Network

A general-purpose social network platform with user profiles, relationship management, content feeds, and social interactions.

## Overview

The Social blueprint provides a complete social networking experience similar to platforms like Twitter/X, Mastodon, or Bluesky. It features user profiles with customizable fields, follow/follower relationships, a post-based content system, multiple timeline views, and comprehensive interaction capabilities.

## Features

### 1. User Accounts & Authentication

#### 1.1 Registration
- Username (unique, alphanumeric with underscores, 3-30 characters)
- Email (unique, validated format)
- Password (minimum 8 characters, hashed with bcrypt)
- Display name (optional, up to 50 characters)

#### 1.2 Authentication
- Session-based authentication with secure tokens
- Login via username or email + password
- Session expiration (30 days default)
- Logout with session invalidation
- Multiple concurrent sessions supported

#### 1.3 Account Management
- Email change (with re-verification)
- Password change
- Account deletion (soft delete with 30-day recovery)
- Account suspension (admin only)
- Verified badge (admin granted)

### 2. User Profiles

#### 2.1 Profile Information
- Display name (50 characters max)
- Bio/description (500 characters max, supports mentions and hashtags)
- Avatar image URL
- Header/banner image URL
- Location (100 characters max)
- Website URL (validated)
- Custom fields (up to 4 key-value pairs, 32/128 chars)
- Join date (immutable)

#### 2.2 Profile Stats (Computed)
- Posts count
- Followers count
- Following count
- Likes received (karma)

#### 2.3 Profile Visibility
- Public profiles (default)
- Private/locked profiles (requires follow approval)
- Profile discoverable in search toggle

### 3. Relationships

#### 3.1 Following
- Follow any public account
- Request to follow private accounts
- Accept/reject follow requests
- Unfollow at any time
- View followers list
- View following list

#### 3.2 Blocking
- Block users to hide their content
- Blocked users cannot:
  - View your profile
  - See your posts
  - Follow you
  - Interact with your content
- View blocked users list
- Unblock at any time

#### 3.3 Muting
- Mute users to hide their posts without unfollowing
- Optional: hide notifications from muted users
- Timed mutes (1 hour, 1 day, 1 week, forever)
- View muted users list
- Unmute at any time

### 4. Posts

#### 4.1 Post Creation
- Text content (up to 500 characters)
- Media attachments (up to 4 images/videos per post)
- Content warnings/spoiler tags
- Visibility levels:
  - **Public**: visible to everyone, appears in timelines
  - **Followers**: visible only to followers
  - **Mentioned**: visible only to mentioned users (DM-like)
  - **Private**: visible only to self (drafts)
- Language tag (optional)
- Sensitive content flag

#### 4.2 Post Types
- **Original**: standalone post
- **Reply**: response to another post (threaded)
- **Quote**: post with embedded reference to another post
- **Repost**: share another user's post to followers

#### 4.3 Post Metadata
- Author information
- Creation timestamp
- Edit timestamp (if edited)
- Reply count
- Like count
- Repost count
- Quote count

#### 4.4 Post Editing
- Edit content within 30 minutes of creation
- Edit history preserved
- Edit indicator shown

#### 4.5 Post Deletion
- Soft delete (content replaced with "[deleted]")
- Replies preserved with deleted parent indicator

### 5. Timelines

#### 5.1 Home Timeline
- Posts from followed accounts
- Reposts from followed accounts
- Replies from followed accounts (to people you also follow)
- Chronological order (newest first)
- Pagination with cursor-based navigation

#### 5.2 Local/Public Timeline
- All public posts from all users
- No replies (top-level posts only)
- Chronological order
- Optional: federated content (future)

#### 5.3 User Timeline
- All posts from a specific user
- Includes replies and reposts
- Filterable: posts only, posts + replies, media only

#### 5.4 Hashtag Timeline
- Posts containing a specific hashtag
- All visibility public posts
- Chronological order

#### 5.5 List Timelines
- Custom curated lists of accounts
- Multiple lists per user
- Separate timeline per list

### 6. Interactions

#### 6.1 Likes
- Like/unlike any visible post
- View posts you've liked
- View who liked a post
- Notification to author

#### 6.2 Reposts (Boosts)
- Repost to share with followers
- Undo repost
- View who reposted
- Notification to author

#### 6.3 Bookmarks
- Save posts privately
- View bookmarked posts
- Remove bookmark
- No notification to author

#### 6.4 Replies
- Reply to any post you can see
- Threaded conversation view
- Nested replies (unlimited depth)
- Notification to parent author and thread participants

#### 6.5 Quotes
- Quote post with your own commentary
- Original post embedded in quote
- Notification to original author

### 7. Notifications

#### 7.1 Notification Types
- **follow**: someone followed you
- **follow_request**: someone requested to follow you
- **mention**: someone mentioned you in a post
- **reply**: someone replied to your post
- **like**: someone liked your post
- **repost**: someone reposted your post
- **quote**: someone quoted your post

#### 7.2 Notification Features
- Mark as read (individual/all)
- Dismiss notification
- Filter by type
- Unread count badge
- Real-time updates (polling or WebSocket)

### 8. Search

#### 8.1 Search Targets
- Posts (full-text search on content)
- Accounts (username and display name)
- Hashtags (name prefix matching)

#### 8.2 Search Filters
- Type filter (posts/accounts/hashtags)
- From user
- Date range
- Has media
- Minimum likes/reposts

#### 8.3 Search Features
- Recent searches history
- Saved searches
- Typeahead suggestions

### 9. Trending

#### 9.1 Trending Hashtags
- Most used hashtags in past 24 hours
- Usage count and trend direction
- Configurable time window

#### 9.2 Trending Posts
- Most engaged posts (likes + reposts)
- Time-decayed scoring
- Diversity (no duplicate authors)

### 10. Media

#### 10.1 Supported Types
- Images: JPEG, PNG, GIF, WebP
- Videos: MP4, WebM (future)
- Audio: MP3, OGG (future)

#### 10.2 Media Metadata
- URL (external or internal)
- Preview/thumbnail URL
- Alt text (accessibility)
- Dimensions (width/height)
- Blurhash for loading placeholder (future)

### 11. Moderation (Admin)

#### 11.1 User Moderation
- Suspend/unsuspend accounts
- Grant/revoke verified status
- Grant/revoke admin status
- View user reports

#### 11.2 Content Moderation
- Remove/restore posts
- Add content warnings retroactively
- Lock threads (prevent replies)

## Technical Specification

### Database Schema

```sql
-- Core tables
accounts (id, username, email, password_hash, display_name, bio, avatar_url,
          header_url, location, website, fields, verified, admin, suspended,
          private, discoverable, created_at, updated_at)

posts (id, account_id, content, content_warning, visibility, reply_to_id,
       thread_id, quote_of_id, language, sensitive, edited_at, created_at,
       likes_count, reposts_count, replies_count, quotes_count)

media (id, post_id, type, url, preview_url, alt_text, width, height, position)

-- Relationships
follows (id, follower_id, following_id, pending, created_at)
blocks (id, account_id, target_id, created_at)
mutes (id, account_id, target_id, hide_notifications, expires_at, created_at)

-- Interactions
likes (id, account_id, post_id, created_at)
reposts (id, account_id, post_id, created_at)
bookmarks (id, account_id, post_id, created_at)

-- Content discovery
hashtags (id, name, posts_count, last_used_at, created_at)
post_hashtags (post_id, hashtag_id)
mentions (id, post_id, account_id, created_at)

-- Notifications
notifications (id, account_id, type, actor_id, post_id, read, created_at)

-- Lists
lists (id, account_id, title, created_at)
list_members (list_id, account_id, created_at)

-- Edit history
edit_history (id, post_id, content, content_warning, sensitive, created_at)

-- Auth
sessions (id, account_id, token, user_agent, ip_address, expires_at, created_at)
```

### API Endpoints

#### Authentication
```
POST   /api/v1/auth/register          Create account
POST   /api/v1/auth/login             Login
POST   /api/v1/auth/logout            Logout (auth required)
```

#### Accounts
```
GET    /api/v1/accounts/verify_credentials    Get current user
PATCH  /api/v1/accounts/update_credentials    Update profile
GET    /api/v1/accounts/:id                   Get account
GET    /api/v1/accounts/:id/posts             Get account posts
GET    /api/v1/accounts/:id/followers         Get followers
GET    /api/v1/accounts/:id/following         Get following
GET    /api/v1/accounts/:id/lists             Get lists containing user
POST   /api/v1/accounts/:id/follow            Follow user
POST   /api/v1/accounts/:id/unfollow          Unfollow user
POST   /api/v1/accounts/:id/block             Block user
POST   /api/v1/accounts/:id/unblock           Unblock user
POST   /api/v1/accounts/:id/mute              Mute user
POST   /api/v1/accounts/:id/unmute            Unmute user
GET    /api/v1/accounts/relationships         Get relationship to multiple users
GET    /api/v1/accounts/search                Search accounts
```

#### Posts
```
POST   /api/v1/posts                  Create post
GET    /api/v1/posts/:id              Get post
PUT    /api/v1/posts/:id              Edit post
DELETE /api/v1/posts/:id              Delete post
GET    /api/v1/posts/:id/context      Get thread context (ancestors + descendants)
POST   /api/v1/posts/:id/like         Like post
DELETE /api/v1/posts/:id/like         Unlike post
POST   /api/v1/posts/:id/repost       Repost
DELETE /api/v1/posts/:id/repost       Undo repost
POST   /api/v1/posts/:id/bookmark     Bookmark
DELETE /api/v1/posts/:id/bookmark     Remove bookmark
GET    /api/v1/posts/:id/liked_by     Get users who liked
GET    /api/v1/posts/:id/reposted_by  Get users who reposted
GET    /api/v1/posts/:id/quotes       Get quote posts
```

#### Timelines
```
GET    /api/v1/timelines/home         Home timeline (auth required)
GET    /api/v1/timelines/public       Public timeline
GET    /api/v1/timelines/tag/:tag     Hashtag timeline
GET    /api/v1/timelines/list/:id     List timeline (auth required)
```

#### Notifications
```
GET    /api/v1/notifications          Get notifications (auth required)
POST   /api/v1/notifications/clear    Clear all notifications
POST   /api/v1/notifications/:id/dismiss  Dismiss single notification
GET    /api/v1/notifications/unread_count  Get unread count
```

#### Search
```
GET    /api/v1/search                 Search (posts, accounts, hashtags)
GET    /api/v1/trends/tags            Trending hashtags
GET    /api/v1/trends/posts           Trending posts
```

#### Lists
```
GET    /api/v1/lists                  Get user's lists (auth required)
POST   /api/v1/lists                  Create list
GET    /api/v1/lists/:id              Get list
PUT    /api/v1/lists/:id              Update list
DELETE /api/v1/lists/:id              Delete list
GET    /api/v1/lists/:id/accounts     Get list members
POST   /api/v1/lists/:id/accounts     Add members
DELETE /api/v1/lists/:id/accounts     Remove members
```

#### Bookmarks
```
GET    /api/v1/bookmarks              Get bookmarked posts (auth required)
```

#### Follow Requests (Private Accounts)
```
GET    /api/v1/follow_requests        Get pending requests (auth required)
POST   /api/v1/follow_requests/:id/authorize  Accept request
POST   /api/v1/follow_requests/:id/reject     Reject request
```

### Web Routes (HTML Pages)

```
GET    /                              Home (public timeline or login prompt)
GET    /login                         Login page
GET    /register                      Registration page
GET    /explore                       Explore/trending page
GET    /search                        Search page with results
GET    /notifications                 Notifications page
GET    /bookmarks                     Bookmarks page
GET    /lists                         Lists management page
GET    /lists/:id                     Single list view
GET    /settings                      Settings page
GET    /settings/:page                Settings subpage
GET    /@:username                    User profile page
GET    /@:username/post/:id           Single post page with thread
GET    /@:username/followers          Followers list page
GET    /@:username/following          Following list page
GET    /tags/:tag                     Hashtag page
```

### Feature Packages

```
feature/
├── accounts/         # User accounts, auth, sessions
│   ├── api.go       # Types: Account, Session, CreateIn, UpdateIn, LoginIn
│   └── service.go   # Business logic: registration, auth, profile updates
├── profiles/         # Extended profile functionality
│   ├── api.go       # Types: Profile, Field, Stats
│   └── service.go   # Profile enrichment, stats computation
├── posts/            # Post creation and management
│   ├── api.go       # Types: Post, CreateIn, UpdateIn, Visibility
│   └── service.go   # CRUD, visibility checks, content parsing
├── relationships/    # Social graph management
│   ├── api.go       # Types: Relationship, FollowRequest
│   └── service.go   # Follow/block/mute logic
├── timelines/        # Timeline generation
│   ├── api.go       # Types: TimelineOpts, Cursor
│   └── service.go   # Home/public/hashtag timeline generation
├── interactions/     # Likes, reposts, bookmarks
│   ├── api.go       # Types: Like, Repost, Bookmark
│   └── service.go   # Interaction CRUD with count updates
├── notifications/    # Notification system
│   ├── api.go       # Types: Notification, NotificationType
│   └── service.go   # Create/list/dismiss/mark-read
├── search/           # Search functionality
│   ├── api.go       # Types: SearchOpts, SearchResult
│   └── service.go   # Full-text search across content types
├── trending/         # Trending content
│   ├── api.go       # Types: TrendingTag, TrendingPost
│   └── service.go   # Trending computation and caching
├── lists/            # User-created lists
│   ├── api.go       # Types: List, ListMember
│   └── service.go   # List CRUD and membership
└── media/            # Media handling
    ├── api.go       # Types: Media, MediaType
    └── service.go   # Upload processing, validation
```

### Store Layer

```
store/duckdb/
├── store.go              # Core store with schema initialization
├── schema.sql            # Embedded SQL schema
├── accounts_store.go     # Account CRUD + sessions
├── posts_store.go        # Post CRUD + counts
├── relationships_store.go # Follows, blocks, mutes
├── timelines_store.go    # Timeline queries
├── interactions_store.go # Likes, reposts, bookmarks
├── notifications_store.go # Notification storage
├── search_store.go       # Full-text search queries
├── trending_store.go     # Trending aggregations
├── lists_store.go        # List storage
└── media_store.go        # Media metadata storage
```

### CLI Commands

```bash
# Start the server
social serve [--addr :8080] [--data ./data] [--dev]

# Initialize database
social init [--data ./data]

# Seed with sample data
social seed [--data ./data] [--users 100] [--posts 1000]

# Create admin user
social admin create <username> <email> <password>

# Manage users
social admin suspend <username>
social admin unsuspend <username>
social admin verify <username>
social admin unverify <username>
```

## Implementation Priority

### Phase 1: Core Foundation
1. Database schema
2. Account registration/login
3. Basic profiles
4. Post creation (text only)
5. Public timeline

### Phase 2: Social Graph
1. Follow/unfollow
2. Home timeline
3. User profile pages
4. Followers/following lists

### Phase 3: Interactions
1. Likes
2. Reposts
3. Replies (threading)
4. Bookmarks

### Phase 4: Discovery
1. Notifications
2. Search (basic)
3. Hashtags
4. Trending

### Phase 5: Advanced Features
1. Quote posts
2. Media attachments
3. Lists
4. Private accounts
5. Muting/blocking

### Phase 6: Polish
1. Edit history
2. Content warnings
3. Advanced search
4. Admin tools
5. Web UI refinements

## Security Considerations

- Password hashing with bcrypt (cost 12)
- CSRF protection on forms
- Rate limiting on auth endpoints
- Input sanitization (XSS prevention)
- SQL parameterized queries (injection prevention)
- Session token rotation on privilege changes
- Secure cookie settings (HttpOnly, Secure, SameSite)

## Performance Considerations

- Indexed queries on all foreign keys
- Denormalized counts (likes_count, etc.) for fast reads
- Cursor-based pagination for timelines
- Connection pooling for database
- Static asset caching with versioned URLs
- Template caching in production mode
