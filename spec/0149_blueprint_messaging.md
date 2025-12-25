# Messaging Blueprint - Full Feature Implementation Spec

## Overview

The **Messaging Blueprint** is a comprehensive, production-ready messaging platform inspired by WhatsApp and Telegram. It provides personal messaging, group chats, media sharing, voice messages, stories/status updates, and real-time communication through WebSocket. The implementation follows the established blueprint architecture patterns using Mizu framework with DuckDB persistence.

## Goals

### Primary Goals
1. **Personal Messaging**: One-to-one private conversations with full message lifecycle
2. **Group Chats**: Multi-participant group conversations with admin controls
3. **Real-time Delivery**: WebSocket-based instant message delivery with offline queuing
4. **Rich Media**: Image, video, audio, document sharing with compression and thumbnails
5. **Voice Messages**: Audio recording and playback support
6. **Stories/Status**: 24-hour ephemeral content sharing (like WhatsApp Status / Telegram Stories)
7. **Message Status**: Sent, delivered, and read receipt tracking
8. **End-to-End Encryption**: Architectural support for E2EE (client-side implementation)

### Secondary Goals
1. **Contacts Management**: Phone/username-based contact discovery
2. **Broadcast Lists**: One-to-many messaging without group context
3. **Starred Messages**: Save important messages for quick access
4. **Archived Chats**: Hide inactive conversations
5. **Message Search**: Full-text search across conversations
6. **Voice/Video Calls**: Architecture for WebRTC-based calling (signaling server)
7. **Stickers & Reactions**: Expressive emoji reactions and sticker packs
8. **Disappearing Messages**: Auto-delete messages after set time

## Requirements

### Functional Requirements

#### FR-1: User Management
- FR-1.1: User registration with phone number or email
- FR-1.2: Username selection with uniqueness validation
- FR-1.3: Profile management (name, bio, avatar, status)
- FR-1.4: Last seen / online status (configurable privacy)
- FR-1.5: Two-factor authentication support
- FR-1.6: Account deletion with data purge

#### FR-2: Contacts
- FR-2.1: Add contacts by phone number or username
- FR-2.2: Contact name customization (display name override)
- FR-2.3: Block/unblock users
- FR-2.4: Contact synchronization from phonebook
- FR-2.5: Find users by proximity/QR code

#### FR-3: Personal Messaging (1-on-1)
- FR-3.1: Start conversation with any contact
- FR-3.2: Send text messages with markdown support
- FR-3.3: Send media (images, videos, audio, documents)
- FR-3.4: Voice message recording and playback
- FR-3.5: Reply to specific messages
- FR-3.6: Forward messages to other chats
- FR-3.7: Delete messages (for me / for everyone)
- FR-3.8: Edit sent messages within time window
- FR-3.9: Message reactions with emoji
- FR-3.10: Typing indicators

#### FR-4: Group Messaging
- FR-4.1: Create groups with name, icon, and description
- FR-4.2: Add/remove participants (admin only)
- FR-4.3: Admin and super-admin roles
- FR-4.4: Group invite links with expiry
- FR-4.5: Restrict who can send messages (admin only mode)
- FR-4.6: Restrict who can change group info
- FR-4.7: Pin important messages
- FR-4.8: Leave group (with exit message option)
- FR-4.9: Group member limit (up to 1024)
- FR-4.10: Mention specific users (@username)
- FR-4.11: Mention all participants (@everyone)

#### FR-5: Message Delivery
- FR-5.1: Real-time delivery via WebSocket
- FR-5.2: Offline message queuing
- FR-5.3: Message status: pending → sent → delivered → read
- FR-5.4: Delivery receipts (configurable)
- FR-5.5: Read receipts (configurable)
- FR-5.6: Push notification integration (architecture)

#### FR-6: Stories/Status
- FR-6.1: Post text, image, or video stories
- FR-6.2: 24-hour auto-expiry
- FR-6.3: View count tracking
- FR-6.4: Reply to stories (creates DM)
- FR-6.5: Privacy controls (contacts only, selected, everyone)
- FR-6.6: Mute specific contacts' stories
- FR-6.7: Story highlights (save beyond 24 hours)

#### FR-7: Broadcast Lists
- FR-7.1: Create broadcast list with recipients
- FR-7.2: Send message to all recipients as individual DMs
- FR-7.3: Recipients cannot see other recipients
- FR-7.4: Manage broadcast list members

#### FR-8: Media & Attachments
- FR-8.1: Image upload with compression (multiple sizes)
- FR-8.2: Video upload with thumbnail generation
- FR-8.3: Audio file upload
- FR-8.4: Document upload (PDF, Office, etc.)
- FR-8.5: GIF support via URL or upload
- FR-8.6: Sticker packs (system and custom)
- FR-8.7: Location sharing
- FR-8.8: Contact card sharing

#### FR-9: Search & Organization
- FR-9.1: Search messages by content
- FR-9.2: Search by sender
- FR-9.3: Filter by media type
- FR-9.4: Star/favorite messages
- FR-9.5: Archive conversations
- FR-9.6: Pin important conversations
- FR-9.7: Mute notifications per chat

#### FR-10: Calls (Architecture Only)
- FR-10.1: Voice call signaling (WebRTC SDP/ICE)
- FR-10.2: Video call signaling
- FR-10.3: Call history
- FR-10.4: Missed call notifications
- FR-10.5: Group call support (up to 8 participants)

#### FR-11: Security & Privacy
- FR-11.1: E2EE key exchange architecture
- FR-11.2: Disappearing messages (per-chat setting)
- FR-11.3: View-once media
- FR-11.4: Screen capture prevention flag
- FR-11.5: Two-factor authentication
- FR-11.6: Session management (logout other devices)
- FR-11.7: Privacy settings (last seen, profile photo, about)

### Non-Functional Requirements

#### NFR-1: Performance
- Message delivery latency < 100ms for online users
- Support 100k concurrent WebSocket connections per instance
- Message history pagination with cursor-based scrolling
- Media upload/download with resumable transfers

#### NFR-2: Scalability
- Horizontal scaling via message broker (architecture support)
- Shardable by user ID for database scaling
- CDN-ready media storage abstraction

#### NFR-3: Reliability
- Message persistence before acknowledgment
- Idempotent message delivery
- Graceful degradation when services unavailable

#### NFR-4: Security
- HTTPS/WSS for all communications
- Rate limiting on all endpoints
- Input validation and sanitization
- Secure session management

## Design

### System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Load Balancer                              │
└─────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────┐           ┌───────────────┐           ┌───────────────┐
│  Web Server   │           │  Web Server   │           │  Web Server   │
│   (Mizu)      │           │   (Mizu)      │           │   (Mizu)      │
└───────────────┘           └───────────────┘           └───────────────┘
        │                           │                           │
        └───────────────────────────┼───────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────┐           ┌───────────────┐           ┌───────────────┐
│    DuckDB     │           │  Media Store  │           │  Redis/NATS   │
│  (Database)   │           │  (S3/Local)   │           │  (Pub/Sub)    │
└───────────────┘           └───────────────┘           └───────────────┘
```

### Component Architecture

```
messaging/
├── cmd/messaging/
│   └── main.go                 # CLI entry point
├── cli/
│   ├── root.go                 # Root command
│   ├── serve.go                # Start HTTP server
│   ├── init.go                 # Initialize database
│   └── seed.go                 # Seed sample data
├── app/web/
│   ├── server.go               # HTTP/WebSocket server
│   ├── handler/
│   │   ├── auth.go             # Authentication endpoints
│   │   ├── user.go             # User profile endpoints
│   │   ├── contact.go          # Contact management
│   │   ├── chat.go             # Chat/conversation endpoints
│   │   ├── message.go          # Message CRUD
│   │   ├── group.go            # Group management
│   │   ├── media.go            # Media upload/download
│   │   ├── story.go            # Stories/status
│   │   ├── call.go             # Call signaling
│   │   ├── page.go             # HTML pages
│   │   └── response.go         # Response helpers
│   └── ws/
│       ├── hub.go              # WebSocket connection manager
│       └── connection.go       # Individual connection
├── feature/
│   ├── accounts/
│   │   ├── service.go          # User account logic
│   │   └── api.go              # Interfaces & models
│   ├── contacts/
│   │   ├── service.go          # Contact management
│   │   └── api.go
│   ├── chats/
│   │   ├── service.go          # Chat/conversation logic
│   │   └── api.go
│   ├── messages/
│   │   ├── service.go          # Message processing
│   │   └── api.go
│   ├── groups/
│   │   ├── service.go          # Group management
│   │   └── api.go
│   ├── media/
│   │   ├── service.go          # Media handling
│   │   └── api.go
│   ├── stories/
│   │   ├── service.go          # Stories/status
│   │   └── api.go
│   ├── calls/
│   │   ├── service.go          # Call signaling
│   │   └── api.go
│   └── presence/
│       ├── service.go          # Online status
│       └── api.go
├── store/duckdb/
│   ├── store.go                # Store initialization
│   ├── schema.sql              # Database schema
│   ├── users_store.go
│   ├── contacts_store.go
│   ├── chats_store.go
│   ├── messages_store.go
│   ├── groups_store.go
│   ├── media_store.go
│   ├── stories_store.go
│   └── calls_store.go
├── assets/
│   ├── embed.go
│   ├── static/
│   │   ├── js/app.js
│   │   └── css/
│   └── views/default/
│       ├── layouts/default.html
│       ├── pages/
│       │   ├── app.html        # Main messaging UI
│       │   ├── home.html
│       │   ├── login.html
│       │   └── register.html
│       └── components/
│           ├── chat_list.html
│           ├── message_view.html
│           └── user_panel.html
├── pkg/
│   ├── ulid/ulid.go
│   └── password/password.go
├── go.mod
├── Makefile
└── README.md
```

### Database Schema Design

#### Core Entities

```sql
-- Users: Account and profile information
users (
  id, phone, email, username, display_name, bio,
  avatar_url, password_hash, status, last_seen_at,
  privacy_settings, e2e_public_key, two_fa_enabled,
  created_at, updated_at
)

-- Sessions: Authentication sessions
sessions (
  id, user_id, token, device_name, device_type,
  push_token, ip_address, expires_at, created_at
)

-- Contacts: User's contact list
contacts (
  user_id, contact_user_id, display_name, is_blocked,
  is_favorite, created_at
)
```

#### Messaging Entities

```sql
-- Chats: Conversation containers
chats (
  id, type (direct|group|broadcast), name,
  last_message_id, last_message_at, created_at
)

-- Chat participants
chat_participants (
  chat_id, user_id, role (member|admin|owner),
  joined_at, is_muted, mute_until, unread_count,
  last_read_message_id
)

-- Messages: All message types
messages (
  id, chat_id, sender_id, type (text|image|video|audio|
    voice|document|sticker|location|contact|system),
  content, content_html, reply_to_id, forward_from_id,
  status (pending|sent|delivered|read),
  delivered_at, read_at, is_edited, edited_at,
  expires_at, created_at
)

-- Message media
message_media (
  id, message_id, type, filename, content_type,
  size, url, thumbnail_url, duration, width, height,
  is_view_once, view_count, created_at
)

-- Message recipients (for delivery tracking)
message_recipients (
  message_id, user_id, status, delivered_at, read_at
)

-- Message reactions
message_reactions (
  message_id, user_id, emoji, created_at
)

-- Starred messages
starred_messages (
  user_id, message_id, created_at
)
```

#### Group Entities

```sql
-- Groups: Group-specific metadata
groups (
  chat_id, name, description, icon_url, owner_id,
  invite_link, invite_link_expires_at, member_count,
  max_members, only_admins_can_send, only_admins_can_edit,
  disappearing_messages_ttl, created_at
)

-- Group invites
group_invites (
  code, chat_id, created_by, max_uses, uses,
  expires_at, created_at
)
```

#### Stories Entities

```sql
-- Stories
stories (
  id, user_id, type (text|image|video), content,
  media_url, thumbnail_url, background_color, text_style,
  view_count, privacy (contacts|selected|everyone),
  is_highlight, expires_at, created_at
)

-- Story views
story_views (
  story_id, viewer_id, viewed_at
)

-- Story privacy
story_privacy (
  story_id, user_id, is_allowed
)

-- Story mutes
story_mutes (
  user_id, muted_user_id, created_at
)
```

#### Call Entities

```sql
-- Calls
calls (
  id, chat_id, caller_id, type (voice|video),
  status (ringing|ongoing|ended|missed|declined),
  started_at, ended_at, duration
)

-- Call participants
call_participants (
  call_id, user_id, joined_at, left_at, is_muted, is_video_off
)
```

#### Settings & Metadata

```sql
-- User settings
user_settings (
  user_id, theme, font_size, notification_sound,
  message_preview, enter_to_send, media_auto_download,
  two_column_layout
)

-- Archived chats
archived_chats (
  user_id, chat_id, archived_at
)

-- Pinned chats
pinned_chats (
  user_id, chat_id, position, pinned_at
)

-- Broadcast lists
broadcast_lists (
  id, user_id, name, recipient_count, created_at
)

-- Broadcast recipients
broadcast_recipients (
  list_id, user_id, created_at
)

-- E2E key bundles (for Signal protocol)
key_bundles (
  user_id, identity_key, signed_prekey, prekey_signature,
  one_time_prekeys, updated_at
)
```

### API Design

#### Authentication
```
POST   /api/v1/auth/register          Register new account
POST   /api/v1/auth/login             Login with credentials
POST   /api/v1/auth/verify-phone      Verify phone number
POST   /api/v1/auth/logout            Logout current session
POST   /api/v1/auth/logout-all        Logout all sessions
GET    /api/v1/auth/me                Get current user
PATCH  /api/v1/auth/me                Update profile
DELETE /api/v1/auth/me                Delete account
```

#### Users & Contacts
```
GET    /api/v1/users/{id}             Get user profile
GET    /api/v1/users/search           Search users
GET    /api/v1/contacts               List contacts
POST   /api/v1/contacts               Add contact
DELETE /api/v1/contacts/{id}          Remove contact
POST   /api/v1/contacts/{id}/block    Block user
DELETE /api/v1/contacts/{id}/block    Unblock user
```

#### Chats
```
GET    /api/v1/chats                  List all chats
POST   /api/v1/chats                  Create chat (direct/group)
GET    /api/v1/chats/{id}             Get chat details
PATCH  /api/v1/chats/{id}             Update chat settings
DELETE /api/v1/chats/{id}             Delete/leave chat
POST   /api/v1/chats/{id}/archive     Archive chat
DELETE /api/v1/chats/{id}/archive     Unarchive chat
POST   /api/v1/chats/{id}/pin         Pin chat
DELETE /api/v1/chats/{id}/pin         Unpin chat
POST   /api/v1/chats/{id}/mute        Mute chat
DELETE /api/v1/chats/{id}/mute        Unmute chat
```

#### Messages
```
GET    /api/v1/chats/{id}/messages           List messages (paginated)
POST   /api/v1/chats/{id}/messages           Send message
GET    /api/v1/chats/{id}/messages/{msg_id}  Get message
PATCH  /api/v1/chats/{id}/messages/{msg_id}  Edit message
DELETE /api/v1/chats/{id}/messages/{msg_id}  Delete message
POST   /api/v1/chats/{id}/messages/{msg_id}/react   Add reaction
DELETE /api/v1/chats/{id}/messages/{msg_id}/react   Remove reaction
POST   /api/v1/chats/{id}/messages/{msg_id}/forward Forward message
POST   /api/v1/chats/{id}/messages/{msg_id}/star    Star message
DELETE /api/v1/chats/{id}/messages/{msg_id}/star    Unstar message
POST   /api/v1/chats/{id}/read               Mark as read
POST   /api/v1/chats/{id}/typing             Send typing indicator
```

#### Groups
```
POST   /api/v1/groups                        Create group
GET    /api/v1/groups/{id}                   Get group info
PATCH  /api/v1/groups/{id}                   Update group
DELETE /api/v1/groups/{id}                   Delete group
GET    /api/v1/groups/{id}/members           List members
POST   /api/v1/groups/{id}/members           Add members
DELETE /api/v1/groups/{id}/members/{user_id} Remove member
PATCH  /api/v1/groups/{id}/members/{user_id} Update member role
POST   /api/v1/groups/{id}/leave             Leave group
GET    /api/v1/groups/{id}/invite            Get invite link
POST   /api/v1/groups/{id}/invite            Generate new invite
POST   /api/v1/invite/{code}                 Join via invite
```

#### Media
```
POST   /api/v1/media/upload           Upload media file
GET    /api/v1/media/{id}             Get media file
GET    /api/v1/media/{id}/thumbnail   Get thumbnail
```

#### Stories
```
GET    /api/v1/stories                List stories (contacts)
POST   /api/v1/stories                Create story
GET    /api/v1/stories/{id}           Get story
DELETE /api/v1/stories/{id}           Delete story
POST   /api/v1/stories/{id}/view      Mark as viewed
POST   /api/v1/stories/{id}/reply     Reply to story
GET    /api/v1/stories/archive        Get story archive
POST   /api/v1/stories/{id}/highlight Save as highlight
```

#### Calls
```
POST   /api/v1/calls                  Initiate call
GET    /api/v1/calls/{id}             Get call info
POST   /api/v1/calls/{id}/answer      Answer call
POST   /api/v1/calls/{id}/decline     Decline call
POST   /api/v1/calls/{id}/end         End call
POST   /api/v1/calls/{id}/signal      Exchange signaling data
GET    /api/v1/calls/history          Call history
```

#### Search
```
GET    /api/v1/search/messages        Search messages
GET    /api/v1/search/media           Search media
GET    /api/v1/search/users           Search users
```

#### WebSocket
```
GET    /ws?token={session_token}      WebSocket connection
```

### WebSocket Protocol

#### Op Codes
```go
const (
    OpDispatch     = 0  // Server → Client: Event dispatch
    OpHeartbeat    = 1  // Client → Server: Keepalive
    OpHeartbeatAck = 2  // Server → Client: Heartbeat acknowledged
    OpIdentify     = 3  // Client → Server: Authenticate
    OpReady        = 4  // Server → Client: Connection ready
    OpTyping       = 5  // Bidirectional: Typing indicator
    OpPresence     = 6  // Bidirectional: Presence update
    OpAck          = 7  // Server → Client: Message acknowledged
    OpCallSignal   = 8  // Bidirectional: WebRTC signaling
)
```

#### Event Types
```go
const (
    EventMessageCreate   = "MESSAGE_CREATE"
    EventMessageUpdate   = "MESSAGE_UPDATE"
    EventMessageDelete   = "MESSAGE_DELETE"
    EventMessageAck      = "MESSAGE_ACK"
    EventTypingStart     = "TYPING_START"
    EventTypingStop      = "TYPING_STOP"
    EventPresenceUpdate  = "PRESENCE_UPDATE"
    EventChatCreate      = "CHAT_CREATE"
    EventChatUpdate      = "CHAT_UPDATE"
    EventChatDelete      = "CHAT_DELETE"
    EventMemberJoin      = "MEMBER_JOIN"
    EventMemberLeave     = "MEMBER_LEAVE"
    EventStoryCreate     = "STORY_CREATE"
    EventStoryDelete     = "STORY_DELETE"
    EventCallIncoming    = "CALL_INCOMING"
    EventCallAccepted    = "CALL_ACCEPTED"
    EventCallEnded       = "CALL_ENDED"
    EventCallSignal      = "CALL_SIGNAL"
)
```

#### Message Payload Structure
```json
{
  "op": 0,
  "t": "MESSAGE_CREATE",
  "d": {
    "id": "01HXYZ...",
    "chat_id": "01HABC...",
    "sender": {
      "id": "01H123...",
      "display_name": "Alice",
      "avatar_url": "..."
    },
    "type": "text",
    "content": "Hello!",
    "reply_to": null,
    "created_at": "2025-01-15T10:30:00Z"
  }
}
```

### UI Design

#### Main Layout (Desktop)
```
+-----------------------------------------------------------+
|  [Logo]  Search...                    [Status] [Settings] |
+----------+------------------------------------------------+
|          |                                                 |
| Chats    |  Chat Header: Name, Avatar, Status, Actions    |
| ──────── |  ─────────────────────────────────────────────  |
| [Avatar] |                                                 |
| Alice    |  [Message bubbles - alternating left/right]    |
| Last msg |                                                 |
| ──────── |  [Date separator]                               |
| [Avatar] |                                                 |
| Bob      |  [Message with media]                           |
| Last msg |                                                 |
| ──────── |  [Voice message player]                         |
| [Avatar] |                                                 |
| Group    |  ─────────────────────────────────────────────  |
| Last msg |  [Typing indicator...]                          |
|          |  [Message input] [Attach] [Voice] [Send]        |
+----------+------------------------------------------------+
```

#### Stories View
```
+-----------------------------------------------------------+
|  Status                                           [+] Add  |
+-----------------------------------------------------------+
| [My Status] | [Contact 1] | [Contact 2] | [Contact 3] ... |
|    ring     |    ring     |    ring     |    ring         |
+-----------------------------------------------------------+
| [Full-screen story viewer with tap navigation]            |
+-----------------------------------------------------------+
```

#### Message Types Visual

**Text Message:**
```
┌────────────────────────────────┐
│ Hello! How are you?          │
│                    10:30 ✓✓  │
└────────────────────────────────┘
```

**Voice Message:**
```
┌────────────────────────────────┐
│ [▶] ═══════════○═══  0:42     │
│                    10:31 ✓✓  │
└────────────────────────────────┘
```

**Media Message:**
```
┌────────────────────────────────┐
│ ┌──────────────────────────┐  │
│ │                          │  │
│ │     [Image/Video]        │  │
│ │                          │  │
│ └──────────────────────────┘  │
│ Caption text here             │
│                    10:32 ✓✓  │
└────────────────────────────────┘
```

### Security Considerations

#### End-to-End Encryption Architecture
The system is designed to support E2EE with the Signal Protocol:

1. **Key Generation**: Each user generates identity key pair
2. **Key Exchange**: X3DH (Extended Triple Diffie-Hellman) for initial key exchange
3. **Double Ratchet**: Session keys ratcheted for forward secrecy
4. **Server Role**: Store encrypted messages only; cannot decrypt

```
User A                    Server                    User B
  │                         │                         │
  ├── Upload Key Bundle ────►                         │
  │                         │◄── Upload Key Bundle ───┤
  │                         │                         │
  ├── Request B's Bundle ───►                         │
  │◄── Return B's Bundle ───┤                         │
  │                         │                         │
  ├── Encrypted Message ────►                         │
  │                         ├── Forward Encrypted ────►
  │                         │                         │
```

#### Privacy Settings
```go
type PrivacySettings struct {
    LastSeen        string `json:"last_seen"`        // everyone|contacts|nobody
    ProfilePhoto    string `json:"profile_photo"`    // everyone|contacts|nobody
    About           string `json:"about"`            // everyone|contacts|nobody
    Groups          string `json:"groups"`           // everyone|contacts|nobody
    ReadReceipts    bool   `json:"read_receipts"`    // enable/disable
    TypingIndicator bool   `json:"typing_indicator"` // enable/disable
}
```

## Implementation Plan

### Phase 1: Core Foundation (MVP)
1. Project setup with go.mod, Makefile, CLI structure
2. Database schema implementation
3. User registration, authentication, sessions
4. Basic contact management
5. Direct messaging (1-on-1 chats)
6. WebSocket connection and message delivery
7. Message status tracking (sent/delivered/read)
8. Basic web UI

### Phase 2: Rich Messaging
1. Media upload and message attachments
2. Voice message recording/playback
3. Message replies and forwarding
4. Message editing and deletion
5. Emoji reactions
6. Typing indicators
7. Read receipts

### Phase 3: Groups
1. Group creation and management
2. Member roles (admin/member)
3. Group invite links
4. Group settings (admin-only messaging, etc.)
5. Mention system (@user, @everyone)
6. Pinned messages

### Phase 4: Stories & Advanced Features
1. Story creation and viewing
2. Story privacy controls
3. Story replies
4. Chat archiving and pinning
5. Starred messages
6. Message search

### Phase 5: Calls & Polish
1. Call signaling (WebRTC setup)
2. Voice call support
3. Video call support
4. Push notification architecture
5. E2EE key exchange architecture
6. Disappearing messages
7. Performance optimization

## Success Metrics

1. **Functionality**: All 11 functional requirement categories implemented
2. **Performance**: < 100ms message delivery for online users
3. **Reliability**: Zero message loss in normal operation
4. **Usability**: Intuitive UI matching WhatsApp/Telegram UX patterns
5. **Code Quality**: Comprehensive test coverage (>80%)

## References

- [WhatsApp Features](https://www.whatsapp.com/features)
- [Telegram Features](https://telegram.org/tour)
- [Signal Protocol](https://signal.org/docs/)
- [WebRTC API](https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API)
- [Chat Blueprint](../blueprints/chat/) - Reference implementation
