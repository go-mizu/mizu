# Forum API Specification

## Overview

This document specifies the complete REST API for the Forum blueprint, a production-ready discussion platform combining features from Reddit, Discourse, and traditional forums.

## Authentication

All authenticated endpoints require a session token passed via:
- Cookie: `session_token`
- Header: `Authorization: Bearer <token>`

## Response Format

All API responses follow this structure:

```json
{
  "data": {...},
  "error": "error message if any",
  "meta": {
    "total": 100,
    "page": 1,
    "limit": 25
  }
}
```

## Error Codes

- `400` - Bad Request (validation errors)
- `401` - Unauthorized (not authenticated)
- `403` - Forbidden (not authorized)
- `404` - Not Found
- `409` - Conflict (e.g., username taken)
- `422` - Unprocessable Entity (business logic error)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error

## Endpoints

### Authentication

#### Register Account

```
POST /api/v1/auth/register
```

Request:
```json
{
  "username": "string (3-20 chars, alphanumeric + underscore)",
  "email": "string (valid email)",
  "password": "string (min 8 chars)",
  "display_name": "string (optional)"
}
```

Response:
```json
{
  "data": {
    "session": {
      "token": "string",
      "expires_at": "2025-12-22T10:00:00Z"
    },
    "account": {
      "id": "string",
      "username": "string",
      "display_name": "string",
      "email": "string",
      "post_karma": 0,
      "comment_karma": 0,
      "total_karma": 0,
      "trust_level": 0,
      "created_at": "2025-12-22T10:00:00Z"
    }
  }
}
```

Errors:
- `409` - Username or email already taken
- `400` - Invalid input

---

#### Login

```
POST /api/v1/auth/login
```

Request:
```json
{
  "username_or_email": "string",
  "password": "string"
}
```

Response:
```json
{
  "data": {
    "session": {
      "token": "string",
      "account_id": "string",
      "expires_at": "2025-12-22T10:00:00Z"
    },
    "account": {
      "id": "string",
      "username": "string",
      "display_name": "string",
      "post_karma": 100,
      "comment_karma": 250,
      "total_karma": 350,
      "trust_level": 2,
      "verified": false,
      "admin": false,
      "created_at": "2025-01-01T00:00:00Z"
    }
  }
}
```

Errors:
- `401` - Invalid credentials
- `403` - Account suspended

---

#### Logout

```
POST /api/v1/auth/logout
```

Requires: Authentication

Response:
```json
{
  "data": {
    "message": "Logged out successfully"
  }
}
```

---

#### Verify Credentials

```
GET /api/v1/auth/verify
```

Requires: Authentication

Response:
```json
{
  "data": {
    "account": {
      "id": "string",
      "username": "string",
      "email": "string",
      "...": "..."
    }
  }
}
```

---

### Accounts

#### Get Account

```
GET /api/v1/accounts/{id_or_username}
```

Response:
```json
{
  "data": {
    "account": {
      "id": "string",
      "username": "string",
      "display_name": "string",
      "bio": "string",
      "avatar_url": "string",
      "header_url": "string",
      "post_karma": 1500,
      "comment_karma": 3200,
      "total_karma": 4700,
      "trust_level": 3,
      "verified": true,
      "created_at": "2024-06-15T10:00:00Z"
    }
  }
}
```

---

#### Update Account

```
PATCH /api/v1/accounts/{id}
```

Requires: Authentication (must be account owner)

Request:
```json
{
  "display_name": "string (optional)",
  "bio": "string (optional)",
  "avatar_url": "string (optional)",
  "header_url": "string (optional)",
  "signature": "string (optional)"
}
```

Response:
```json
{
  "data": {
    "account": {
      "...": "updated account"
    }
  }
}
```

---

#### Search Accounts

```
GET /api/v1/accounts/search?q={query}&limit={limit}
```

Query params:
- `q` - Search query (username, display name)
- `limit` - Results limit (default: 20, max: 100)

Response:
```json
{
  "data": {
    "accounts": [
      {
        "id": "string",
        "username": "string",
        "display_name": "string",
        "avatar_url": "string",
        "total_karma": 4700
      }
    ]
  }
}
```

---

### Forums

#### List All Forums

```
GET /api/v1/forums
```

Query params:
- `parent_id` - Filter by parent forum (optional)

Response:
```json
{
  "data": {
    "forums": [
      {
        "id": "string",
        "parent_id": "string",
        "name": "General Discussion",
        "slug": "general",
        "description": "General topics",
        "icon": "💬",
        "type": "public",
        "nsfw": false,
        "archived": false,
        "thread_count": 1250,
        "post_count": 15600,
        "member_count": 3420,
        "position": 1,
        "is_member": true,
        "is_moderator": false,
        "created_at": "2025-01-01T00:00:00Z"
      }
    ],
    "total": 25
  }
}
```

---

#### Get Forum

```
GET /api/v1/forums/{id_or_slug}
```

Response:
```json
{
  "data": {
    "forum": {
      "id": "string",
      "name": "Technology",
      "slug": "technology",
      "description": "Tech discussions",
      "icon": "💻",
      "banner": "https://...",
      "type": "public",
      "nsfw": false,
      "thread_count": 5420,
      "post_count": 82150,
      "member_count": 12850,
      "settings": {
        "allow_polls": true,
        "allow_images": true,
        "allow_videos": true,
        "require_approval": false,
        "min_karma_to_post": 10,
        "min_age_to_post": 7,
        "rate_limit_posts": 10,
        "rate_limit_comments": 50
      },
      "rules": [
        {
          "id": "string",
          "title": "Be respectful",
          "description": "No personal attacks or harassment",
          "position": 1
        }
      ],
      "is_member": true,
      "is_moderator": false,
      "role": null,
      "created_at": "2025-01-01T00:00:00Z"
    }
  }
}
```

---

#### Create Forum

```
POST /api/v1/forums
```

Requires: Authentication (admin only)

Request:
```json
{
  "parent_id": "string (optional)",
  "name": "string",
  "slug": "string (optional, auto-generated)",
  "description": "string",
  "icon": "string (emoji or URL)",
  "type": "public|restricted|private",
  "nsfw": false,
  "settings": {
    "allow_polls": true,
    "allow_images": true,
    "min_karma_to_post": 0
  }
}
```

Response:
```json
{
  "data": {
    "forum": {
      "...": "created forum"
    }
  }
}
```

---

#### Update Forum

```
PATCH /api/v1/forums/{id}
```

Requires: Authentication (admin or moderator with permissions)

Request:
```json
{
  "name": "string (optional)",
  "description": "string (optional)",
  "icon": "string (optional)",
  "banner": "string (optional)",
  "type": "string (optional)",
  "nsfw": "boolean (optional)",
  "archived": "boolean (optional)",
  "settings": {
    "...": "optional settings"
  }
}
```

Response:
```json
{
  "data": {
    "forum": {
      "...": "updated forum"
    }
  }
}
```

---

#### Delete Forum

```
DELETE /api/v1/forums/{id}
```

Requires: Authentication (admin only)

Response:
```json
{
  "data": {
    "message": "Forum deleted successfully"
  }
}
```

---

#### Join Forum

```
POST /api/v1/forums/{id}/join
```

Requires: Authentication

Response:
```json
{
  "data": {
    "message": "Joined forum successfully"
  }
}
```

---

#### Leave Forum

```
POST /api/v1/forums/{id}/leave
```

Requires: Authentication

Response:
```json
{
  "data": {
    "message": "Left forum successfully"
  }
}
```

---

#### List Forum Moderators

```
GET /api/v1/forums/{id}/moderators
```

Response:
```json
{
  "data": {
    "moderators": [
      {
        "account": {
          "id": "string",
          "username": "string",
          "display_name": "string",
          "avatar_url": "string"
        },
        "role": "owner|admin|moderator",
        "added_at": "2025-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

#### Add Moderator

```
POST /api/v1/forums/{id}/moderators
```

Requires: Authentication (forum owner only)

Request:
```json
{
  "account_id": "string",
  "role": "admin|moderator"
}
```

Response:
```json
{
  "data": {
    "message": "Moderator added successfully"
  }
}
```

---

#### Remove Moderator

```
DELETE /api/v1/forums/{id}/moderators/{account_id}
```

Requires: Authentication (forum owner only)

Response:
```json
{
  "data": {
    "message": "Moderator removed successfully"
  }
}
```

---

### Threads

#### List Threads in Forum

```
GET /api/v1/forums/{id}/threads
```

Query params:
- `sort` - Sorting: `hot`, `new`, `top`, `best`, `rising`, `controversial` (default: `hot`)
- `limit` - Results per page (default: 25, max: 100)
- `after` - Cursor for pagination (thread ID)

Response:
```json
{
  "data": {
    "threads": [
      {
        "id": "string",
        "forum_id": "string",
        "account_id": "string",
        "type": "discussion",
        "title": "What's your favorite programming language?",
        "content": "I'm curious to hear...",
        "slug": "whats-your-favorite-programming-language",
        "sticky": false,
        "locked": false,
        "nsfw": false,
        "state": "open",
        "view_count": 1523,
        "score": 342,
        "upvotes": 389,
        "downvotes": 47,
        "post_count": 156,
        "hot_score": 235.42,
        "last_post_at": "2025-12-22T09:45:00Z",
        "created_at": "2025-12-20T14:30:00Z",
        "forum": {
          "id": "string",
          "name": "Programming",
          "slug": "programming"
        },
        "account": {
          "id": "string",
          "username": "coder123",
          "display_name": "Coder",
          "avatar_url": "https://..."
        },
        "tags": ["languages", "discussion"],
        "user_vote": 1,
        "is_saved": false,
        "is_subscribed": true,
        "is_owner": false
      }
    ],
    "max_id": "string",
    "min_id": "string"
  }
}
```

---

#### Get Thread

```
GET /api/v1/threads/{id}
```

Response:
```json
{
  "data": {
    "thread": {
      "id": "string",
      "forum_id": "string",
      "account_id": "string",
      "type": "question",
      "title": "How do I implement authentication?",
      "content": "I'm building a web app and...",
      "slug": "how-do-i-implement-authentication",
      "sticky": false,
      "locked": false,
      "nsfw": false,
      "state": "open",
      "view_count": 2341,
      "score": 87,
      "upvotes": 92,
      "downvotes": 5,
      "post_count": 23,
      "best_post_id": "post_abc123",
      "created_at": "2025-12-21T10:00:00Z",
      "edited_at": "2025-12-21T10:15:00Z",
      "forum": {...},
      "account": {...},
      "tags": ["authentication", "security"],
      "user_vote": 0,
      "is_saved": false,
      "is_subscribed": false,
      "is_owner": false
    }
  }
}
```

---

#### Create Thread

```
POST /api/v1/forums/{id}/threads
```

Requires: Authentication

Request:
```json
{
  "type": "discussion|question|poll|announcement",
  "title": "string (min 3, max 300 chars)",
  "content": "string (min 1 char, markdown supported)",
  "nsfw": false,
  "spoiler": false,
  "tags": ["tag1", "tag2"]
}
```

Response:
```json
{
  "data": {
    "thread": {
      "...": "created thread"
    }
  }
}
```

Errors:
- `403` - Insufficient karma or account age
- `429` - Rate limit exceeded

---

#### Update Thread

```
PATCH /api/v1/threads/{id}
```

Requires: Authentication (thread owner or moderator)

Request:
```json
{
  "title": "string (optional)",
  "content": "string (optional)",
  "nsfw": "boolean (optional)",
  "spoiler": "boolean (optional)"
}
```

Response:
```json
{
  "data": {
    "thread": {
      "...": "updated thread"
    }
  }
}
```

---

#### Delete Thread

```
DELETE /api/v1/threads/{id}
```

Requires: Authentication (thread owner or moderator)

Response:
```json
{
  "data": {
    "message": "Thread deleted successfully"
  }
}
```

---

#### Vote on Thread

```
POST /api/v1/threads/{id}/vote
```

Requires: Authentication

Request:
```json
{
  "value": -1 | 0 | 1
}
```

- `1` - Upvote
- `0` - Remove vote
- `-1` - Downvote

Response:
```json
{
  "data": {
    "score": 342,
    "upvotes": 389,
    "downvotes": 47,
    "user_vote": 1
  }
}
```

Errors:
- `403` - Cannot vote on own content

---

#### Subscribe to Thread

```
POST /api/v1/threads/{id}/subscribe
```

Requires: Authentication

Response:
```json
{
  "data": {
    "message": "Subscribed to thread"
  }
}
```

---

#### Unsubscribe from Thread

```
DELETE /api/v1/threads/{id}/subscribe
```

Requires: Authentication

Response:
```json
{
  "data": {
    "message": "Unsubscribed from thread"
  }
}
```

---

#### Lock Thread (Moderation)

```
POST /api/v1/threads/{id}/lock
```

Requires: Authentication (moderator)

Response:
```json
{
  "data": {
    "message": "Thread locked"
  }
}
```

---

#### Unlock Thread (Moderation)

```
POST /api/v1/threads/{id}/unlock
```

Requires: Authentication (moderator)

Response:
```json
{
  "data": {
    "message": "Thread unlocked"
  }
}
```

---

#### Sticky Thread (Moderation)

```
POST /api/v1/threads/{id}/sticky
```

Requires: Authentication (moderator)

Response:
```json
{
  "data": {
    "message": "Thread stickied"
  }
}
```

---

#### Unsticky Thread (Moderation)

```
POST /api/v1/threads/{id}/unsticky
```

Requires: Authentication (moderator)

Response:
```json
{
  "data": {
    "message": "Thread unstickied"
  }
}
```

---

### Posts (Comments)

#### List Posts in Thread

```
GET /api/v1/threads/{id}/posts
```

Query params:
- `sort` - Sorting: `best`, `new`, `top`, `controversial` (default: `best`)
- `limit` - Results per page (default: 50, max: 200)
- `offset` - Offset for pagination

Response:
```json
{
  "data": {
    "posts": [
      {
        "id": "string",
        "thread_id": "string",
        "account_id": "string",
        "parent_id": null,
        "content": "Great question! Here's my take...",
        "depth": 0,
        "score": 42,
        "upvotes": 45,
        "downvotes": 3,
        "is_best": true,
        "type": "comment",
        "created_at": "2025-12-21T10:30:00Z",
        "account": {
          "id": "string",
          "username": "expert_dev",
          "display_name": "Expert Developer",
          "avatar_url": "https://...",
          "total_karma": 5420
        },
        "children": [
          {
            "id": "string",
            "parent_id": "parent_post_id",
            "content": "Thanks for the detailed explanation!",
            "depth": 1,
            "score": 12,
            "...": "..."
          }
        ],
        "user_vote": 1,
        "is_saved": false,
        "is_owner": false
      }
    ],
    "total": 156
  }
}
```

---

#### Get Post Tree (Nested)

```
GET /api/v1/threads/{id}/posts/tree
```

Returns all posts in a hierarchical tree structure.

Response:
```json
{
  "data": {
    "posts": [
      {
        "id": "string",
        "content": "Root comment",
        "depth": 0,
        "children": [
          {
            "id": "string",
            "content": "Nested reply",
            "depth": 1,
            "children": []
          }
        ]
      }
    ]
  }
}
```

---

#### Create Post

```
POST /api/v1/threads/{id}/posts
```

Requires: Authentication

Request:
```json
{
  "parent_id": "string (optional, for nested replies)",
  "content": "string (min 1 char, markdown supported)"
}
```

Response:
```json
{
  "data": {
    "post": {
      "...": "created post"
    }
  }
}
```

Errors:
- `403` - Thread locked
- `422` - Max depth exceeded
- `429` - Rate limit exceeded

---

#### Update Post

```
PATCH /api/v1/posts/{id}
```

Requires: Authentication (post owner)

Request:
```json
{
  "content": "string"
}
```

Response:
```json
{
  "data": {
    "post": {
      "...": "updated post"
    }
  }
}
```

---

#### Delete Post

```
DELETE /api/v1/posts/{id}
```

Requires: Authentication (post owner or moderator)

Response:
```json
{
  "data": {
    "message": "Post deleted successfully"
  }
}
```

---

#### Vote on Post

```
POST /api/v1/posts/{id}/vote
```

Requires: Authentication

Request:
```json
{
  "value": -1 | 0 | 1
}
```

Response:
```json
{
  "data": {
    "score": 42,
    "upvotes": 45,
    "downvotes": 3,
    "user_vote": 1
  }
}
```

---

#### Mark as Best Answer

```
POST /api/v1/posts/{id}/best
```

Requires: Authentication (thread owner or moderator)

Response:
```json
{
  "data": {
    "message": "Marked as best answer"
  }
}
```

---

### Search

#### Search

```
GET /api/v1/search
```

Query params:
- `q` - Search query (required)
- `type` - Content type: `threads`, `posts`, `forums`, `accounts` (default: `threads`)
- `forum_id` - Filter by forum (optional)
- `sort` - Sorting: `relevance`, `new`, `top` (default: `relevance`)
- `limit` - Results per page (default: 25, max: 100)
- `offset` - Offset for pagination

Response:
```json
{
  "data": {
    "results": [
      {
        "type": "thread",
        "item": {
          "...": "thread object"
        },
        "highlight": "...search query highlights..."
      }
    ],
    "total": 342,
    "took_ms": 45
  }
}
```

---

### Trending

#### Trending Forums

```
GET /api/v1/trending/forums
```

Query params:
- `limit` - Results limit (default: 10, max: 50)

Response:
```json
{
  "data": {
    "forums": [
      {
        "forum": {...},
        "trend_score": 234.5,
        "growth_rate": 0.45
      }
    ]
  }
}
```

---

#### Trending Tags

```
GET /api/v1/trending/tags
```

Query params:
- `limit` - Results limit (default: 20, max: 100)

Response:
```json
{
  "data": {
    "tags": [
      {
        "tag": "golang",
        "count": 1523,
        "trend_score": 456.7
      }
    ]
  }
}
```

---

#### Trending Threads

```
GET /api/v1/trending/threads
```

Query params:
- `limit` - Results limit (default: 25, max: 100)
- `timeframe` - Time window: `hour`, `day`, `week`, `month` (default: `day`)

Response:
```json
{
  "data": {
    "threads": [
      {
        "...": "thread object with trend metrics"
      }
    ]
  }
}
```

---

### User Content

#### List User Threads

```
GET /api/v1/accounts/{id}/threads
```

Query params:
- `limit` - Results per page (default: 25, max: 100)
- `after` - Cursor for pagination

Response:
```json
{
  "data": {
    "threads": [...],
    "max_id": "string",
    "min_id": "string"
  }
}
```

---

#### List User Posts

```
GET /api/v1/accounts/{id}/posts
```

Query params:
- `limit` - Results per page (default: 50, max: 200)
- `offset` - Offset for pagination

Response:
```json
{
  "data": {
    "posts": [...],
    "total": 523
  }
}
```

---

### Moderation

#### Get Moderation Queue

```
GET /api/v1/forums/{id}/queue
```

Requires: Authentication (moderator)

Query params:
- `type` - Content type: `threads`, `posts`, `reports` (default: `all`)
- `limit` - Results per page (default: 50)

Response:
```json
{
  "data": {
    "items": [
      {
        "type": "thread",
        "item": {...},
        "reports": 3,
        "flagged_at": "2025-12-22T09:00:00Z"
      }
    ],
    "total": 12
  }
}
```

---

#### Approve Content

```
POST /api/v1/posts/{id}/approve
```

Requires: Authentication (moderator)

Response:
```json
{
  "data": {
    "message": "Content approved"
  }
}
```

---

#### Remove Content

```
POST /api/v1/posts/{id}/remove
```

Requires: Authentication (moderator)

Request:
```json
{
  "reason": "string (optional)"
}
```

Response:
```json
{
  "data": {
    "message": "Content removed"
  }
}
```

---

#### List Reports

```
GET /api/v1/forums/{id}/reports
```

Requires: Authentication (moderator)

Query params:
- `status` - Filter: `pending`, `resolved`, `dismissed` (default: `pending`)
- `limit` - Results per page (default: 50)

Response:
```json
{
  "data": {
    "reports": [
      {
        "id": "string",
        "target_type": "thread|post",
        "target_id": "string",
        "reporter_id": "string",
        "reason": "Spam",
        "description": "This is clearly spam...",
        "status": "pending",
        "created_at": "2025-12-22T08:30:00Z"
      }
    ],
    "total": 8
  }
}
```

---

#### Create Report

```
POST /api/v1/reports
```

Requires: Authentication

Request:
```json
{
  "target_type": "thread|post",
  "target_id": "string",
  "reason": "spam|harassment|inappropriate|other",
  "description": "string (optional)"
}
```

Response:
```json
{
  "data": {
    "message": "Report submitted successfully"
  }
}
```

---

#### Ban User from Forum

```
POST /api/v1/forums/{id}/ban
```

Requires: Authentication (moderator)

Request:
```json
{
  "account_id": "string",
  "reason": "string",
  "duration": 86400,
  "permanent": false
}
```

Response:
```json
{
  "data": {
    "message": "User banned successfully"
  }
}
```

---

#### Unban User

```
DELETE /api/v1/forums/{id}/ban/{account_id}
```

Requires: Authentication (moderator)

Response:
```json
{
  "data": {
    "message": "User unbanned successfully"
  }
}
```

---

#### Get Moderation Log

```
GET /api/v1/forums/{id}/logs
```

Requires: Authentication (moderator)

Query params:
- `limit` - Results per page (default: 100)
- `offset` - Offset for pagination

Response:
```json
{
  "data": {
    "logs": [
      {
        "id": "string",
        "moderator_id": "string",
        "action": "remove_post",
        "target_type": "post",
        "target_id": "string",
        "reason": "Spam",
        "created_at": "2025-12-22T09:15:00Z",
        "moderator": {
          "username": "mod_user",
          "display_name": "Moderator"
        }
      }
    ],
    "total": 1523
  }
}
```

---

## Rate Limits

Default rate limits per authenticated user:
- **Threads**: 10 per hour
- **Posts**: 50 per hour
- **Votes**: 200 per hour
- **Search**: 60 per minute
- **API calls**: 1000 per hour

Rate limit headers included in all responses:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 995
X-RateLimit-Reset: 1640174400
```

## Pagination

### Cursor-based (for threads)

Uses `after` parameter with thread ID:
```
GET /api/v1/forums/{id}/threads?limit=25&after=thread_xyz
```

Response includes `max_id` and `min_id` for next/previous pages.

### Offset-based (for posts)

Uses `limit` and `offset` parameters:
```
GET /api/v1/threads/{id}/posts?limit=50&offset=100
```

Response includes `total` count.

## Webhooks

Forum owners can configure webhooks for events:
- `thread.created`
- `thread.deleted`
- `post.created`
- `post.deleted`
- `moderation.report`
- `moderation.action`

Webhook payload:
```json
{
  "event": "thread.created",
  "timestamp": "2025-12-22T10:00:00Z",
  "data": {
    "...": "event data"
  }
}
```

## Implementation Notes

1. **Authentication Middleware**: All authenticated endpoints must use `authRequired` middleware that extracts session from cookie/header and validates it.

2. **Optional Authentication**: Some endpoints support optional auth (e.g., viewing threads) to customize response with user-specific data (votes, subscriptions).

3. **Authorization**: Forum-level permissions (owner, admin, moderator) must be checked before modifying forum settings or moderating content.

4. **Scoring Algorithms**: Thread scores (hot, best, controversial) should be computed in the database using the algorithms defined in the README.

5. **Transactions**: Multi-step operations (e.g., create thread + increment forum counter + add karma) should use database transactions.

6. **Caching**: Frequently accessed data (forum lists, trending content) should be cached with appropriate TTL.

7. **Real-time Updates**: Consider WebSocket or SSE for live updates of votes, new posts, etc.

8. **Content Sanitization**: All user-submitted content must be sanitized to prevent XSS attacks.

9. **Markdown Rendering**: Thread and post content supports markdown, rendered server-side for security.

10. **File Uploads**: Media upload endpoints (avatars, images, videos) not included in this spec but should follow multipart/form-data with size limits.
