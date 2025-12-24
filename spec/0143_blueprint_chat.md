# Blueprint: Chat - Realtime Chat System

**Spec ID**: 0143
**Status**: Draft
**Inspired By**: Discord, Slack

## Overview

A modern realtime chat system with rooms (channels), presence tracking, message delivery guarantees, and rich interactions. Built for teams, communities, and direct messaging.

## Goals

1. **Realtime Communication**: WebSocket-based messaging with instant delivery
2. **Room Organization**: Hierarchical structure with servers, categories, and channels
3. **Presence System**: Online/offline/idle/DND status with typing indicators
4. **Message Features**: Rich text, reactions, threads, editing, deletion
5. **Delivery Guarantees**: Read receipts, delivery confirmation, offline message queue
6. **Search**: Full-text search across messages with filters
7. **Notifications**: Push notifications, mentions, @here/@everyone

---

## Data Model

### Core Entities

#### Server (Workspace)
A top-level container for organizing channels and members (like Discord servers or Slack workspaces).

```go
type Server struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    IconURL     string    `json:"icon_url"`
    OwnerID     string    `json:"owner_id"`
    IsPublic    bool      `json:"is_public"`
    InviteCode  string    `json:"invite_code"`
    MemberCount int       `json:"member_count"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

#### Category
Groups channels within a server for organization.

```go
type Category struct {
    ID        string    `json:"id"`
    ServerID  string    `json:"server_id"`
    Name      string    `json:"name"`
    Position  int       `json:"position"`
    CreatedAt time.Time `json:"created_at"`
}
```

#### Channel
A communication space within a server.

```go
type ChannelType string

const (
    ChannelTypeText     ChannelType = "text"
    ChannelTypeVoice    ChannelType = "voice"
    ChannelTypeDM       ChannelType = "dm"
    ChannelTypeGroupDM  ChannelType = "group_dm"
    ChannelTypeThread   ChannelType = "thread"
)

type Channel struct {
    ID            string      `json:"id"`
    ServerID      string      `json:"server_id,omitempty"`  // null for DMs
    CategoryID    string      `json:"category_id,omitempty"`
    Type          ChannelType `json:"type"`
    Name          string      `json:"name"`
    Topic         string      `json:"topic"`
    Position      int         `json:"position"`
    IsPrivate     bool        `json:"is_private"`
    SlowModeDelay int         `json:"slow_mode_delay"` // seconds
    LastMessageID string      `json:"last_message_id"`
    LastMessageAt time.Time   `json:"last_message_at"`
    CreatedAt     time.Time   `json:"created_at"`
    UpdatedAt     time.Time   `json:"updated_at"`
}
```

#### Message
The core message entity with rich content support.

```go
type Message struct {
    ID              string        `json:"id"`
    ChannelID       string        `json:"channel_id"`
    AuthorID        string        `json:"author_id"`
    Content         string        `json:"content"`
    ContentHTML     string        `json:"content_html"`
    ReplyToID       string        `json:"reply_to_id,omitempty"`
    ThreadID        string        `json:"thread_id,omitempty"`
    Type            MessageType   `json:"type"`
    Attachments     []Attachment  `json:"attachments,omitempty"`
    Embeds          []Embed       `json:"embeds,omitempty"`
    Mentions        []string      `json:"mentions,omitempty"`
    MentionRoles    []string      `json:"mention_roles,omitempty"`
    MentionEveryone bool          `json:"mention_everyone"`
    Reactions       []Reaction    `json:"reactions,omitempty"`
    IsPinned        bool          `json:"is_pinned"`
    IsEdited        bool          `json:"is_edited"`
    EditedAt        *time.Time    `json:"edited_at,omitempty"`
    CreatedAt       time.Time     `json:"created_at"`
}

type MessageType string

const (
    MessageTypeDefault       MessageType = "default"
    MessageTypeReply         MessageType = "reply"
    MessageTypeSystemJoin    MessageType = "system_join"
    MessageTypeSystemLeave   MessageType = "system_leave"
    MessageTypeChannelPinned MessageType = "channel_pinned"
)

type Attachment struct {
    ID          string `json:"id"`
    Filename    string `json:"filename"`
    ContentType string `json:"content_type"`
    Size        int64  `json:"size"`
    URL         string `json:"url"`
    Width       int    `json:"width,omitempty"`
    Height      int    `json:"height,omitempty"`
}

type Embed struct {
    Type        string `json:"type"`
    Title       string `json:"title,omitempty"`
    Description string `json:"description,omitempty"`
    URL         string `json:"url,omitempty"`
    Color       int    `json:"color,omitempty"`
    ImageURL    string `json:"image_url,omitempty"`
    VideoURL    string `json:"video_url,omitempty"`
}

type Reaction struct {
    Emoji string   `json:"emoji"`
    Count int      `json:"count"`
    Users []string `json:"users"`
    Me    bool     `json:"me"`
}
```

#### User & Member
User identity and server-specific membership.

```go
type User struct {
    ID            string    `json:"id"`
    Username      string    `json:"username"`
    Discriminator string    `json:"discriminator"` // 4-digit tag like Discord
    DisplayName   string    `json:"display_name"`
    AvatarURL     string    `json:"avatar_url"`
    Bio           string    `json:"bio"`
    Status        Status    `json:"status"`
    CustomStatus  string    `json:"custom_status"`
    IsBot         bool      `json:"is_bot"`
    CreatedAt     time.Time `json:"created_at"`
}

type Status string

const (
    StatusOnline    Status = "online"
    StatusIdle      Status = "idle"
    StatusDND       Status = "dnd"
    StatusInvisible Status = "invisible"
    StatusOffline   Status = "offline"
)

type Member struct {
    UserID    string    `json:"user_id"`
    ServerID  string    `json:"server_id"`
    Nickname  string    `json:"nickname"`
    RoleIDs   []string  `json:"role_ids"`
    JoinedAt  time.Time `json:"joined_at"`
    IsMuted   bool      `json:"is_muted"`
    IsDeafened bool     `json:"is_deafened"`
}
```

#### Role & Permissions
Role-based access control.

```go
type Role struct {
    ID          string      `json:"id"`
    ServerID    string      `json:"server_id"`
    Name        string      `json:"name"`
    Color       int         `json:"color"`
    Position    int         `json:"position"`
    Permissions Permissions `json:"permissions"`
    IsDefault   bool        `json:"is_default"` // @everyone role
    IsMentionable bool      `json:"is_mentionable"`
    CreatedAt   time.Time   `json:"created_at"`
}

type Permissions uint64

const (
    PermissionViewChannel      Permissions = 1 << 0
    PermissionSendMessages     Permissions = 1 << 1
    PermissionManageMessages   Permissions = 1 << 2
    PermissionManageChannels   Permissions = 1 << 3
    PermissionManageServer     Permissions = 1 << 4
    PermissionKickMembers      Permissions = 1 << 5
    PermissionBanMembers       Permissions = 1 << 6
    PermissionManageRoles      Permissions = 1 << 7
    PermissionMentionEveryone  Permissions = 1 << 8
    PermissionAddReactions     Permissions = 1 << 9
    PermissionAttachFiles      Permissions = 1 << 10
    PermissionCreateInvite     Permissions = 1 << 11
    PermissionAdministrator    Permissions = 1 << 31
)
```

#### Presence & Typing
Real-time status tracking.

```go
type Presence struct {
    UserID       string    `json:"user_id"`
    Status       Status    `json:"status"`
    CustomStatus string    `json:"custom_status"`
    Activities   []Activity `json:"activities"`
    ClientInfo   ClientInfo `json:"client_info"`
    LastSeenAt   time.Time `json:"last_seen_at"`
}

type Activity struct {
    Type    string `json:"type"` // playing, streaming, listening, watching
    Name    string `json:"name"`
    Details string `json:"details"`
    State   string `json:"state"`
}

type ClientInfo struct {
    Desktop string `json:"desktop,omitempty"` // online/idle/dnd
    Mobile  string `json:"mobile,omitempty"`
    Web     string `json:"web,omitempty"`
}

type TypingIndicator struct {
    UserID    string    `json:"user_id"`
    ChannelID string    `json:"channel_id"`
    Timestamp time.Time `json:"timestamp"`
}
```

#### Read State & Delivery
Message delivery tracking.

```go
type ReadState struct {
    UserID          string    `json:"user_id"`
    ChannelID       string    `json:"channel_id"`
    LastReadID      string    `json:"last_read_id"`
    MentionCount    int       `json:"mention_count"`
    LastPinTimestamp time.Time `json:"last_pin_timestamp"`
}

type MessageAck struct {
    MessageID   string    `json:"message_id"`
    UserID      string    `json:"user_id"`
    DeliveredAt time.Time `json:"delivered_at"`
    ReadAt      *time.Time `json:"read_at,omitempty"`
}
```

---

## WebSocket Protocol

### Connection
```
ws://host/ws?token={auth_token}
```

### Message Format
```go
type WSMessage struct {
    Op   OpCode          `json:"op"`
    T    string          `json:"t,omitempty"`    // Event type
    D    json.RawMessage `json:"d,omitempty"`    // Data payload
    S    int64           `json:"s,omitempty"`    // Sequence number
}

type OpCode int

const (
    OpDispatch        OpCode = 0  // Server -> Client: Event dispatch
    OpHeartbeat       OpCode = 1  // Client -> Server: Heartbeat
    OpIdentify        OpCode = 2  // Client -> Server: Auth
    OpPresenceUpdate  OpCode = 3  // Client -> Server: Update presence
    OpVoiceStateUpdate OpCode = 4 // Client -> Server: Voice state
    OpResume          OpCode = 6  // Client -> Server: Resume connection
    OpReconnect       OpCode = 7  // Server -> Client: Reconnect request
    OpRequestMembers  OpCode = 8  // Client -> Server: Request members
    OpInvalidSession  OpCode = 9  // Server -> Client: Invalid session
    OpHello           OpCode = 10 // Server -> Client: Hello (heartbeat interval)
    OpHeartbeatAck    OpCode = 11 // Server -> Client: Heartbeat acknowledged
)
```

### Events (Server -> Client)

| Event | Description |
|-------|-------------|
| `READY` | Initial state after identify |
| `MESSAGE_CREATE` | New message in channel |
| `MESSAGE_UPDATE` | Message edited |
| `MESSAGE_DELETE` | Message deleted |
| `MESSAGE_REACTION_ADD` | Reaction added |
| `MESSAGE_REACTION_REMOVE` | Reaction removed |
| `CHANNEL_CREATE` | New channel created |
| `CHANNEL_UPDATE` | Channel settings changed |
| `CHANNEL_DELETE` | Channel deleted |
| `MEMBER_ADD` | Member joined server |
| `MEMBER_REMOVE` | Member left server |
| `MEMBER_UPDATE` | Member details changed |
| `PRESENCE_UPDATE` | User presence changed |
| `TYPING_START` | User started typing |
| `SERVER_UPDATE` | Server settings changed |
| `ROLE_CREATE` | Role created |
| `ROLE_UPDATE` | Role modified |
| `ROLE_DELETE` | Role deleted |

### Events (Client -> Server via REST or WS)

- Send message: `POST /channels/{id}/messages`
- Edit message: `PATCH /channels/{id}/messages/{msg_id}`
- Delete message: `DELETE /channels/{id}/messages/{msg_id}`
- Add reaction: `PUT /channels/{id}/messages/{msg_id}/reactions/{emoji}`
- Typing indicator: `POST /channels/{id}/typing`
- Ack messages: `POST /channels/{id}/messages/{msg_id}/ack`

---

## REST API Endpoints

### Authentication
```
POST   /auth/register         # Create account
POST   /auth/login            # Login, returns token
POST   /auth/logout           # Logout, invalidate token
GET    /auth/me               # Get current user
PATCH  /auth/me               # Update profile
```

### Servers
```
GET    /servers               # List user's servers
POST   /servers               # Create server
GET    /servers/{id}          # Get server details
PATCH  /servers/{id}          # Update server
DELETE /servers/{id}          # Delete server
GET    /servers/{id}/channels # List channels
POST   /servers/{id}/channels # Create channel
GET    /servers/{id}/members  # List members
POST   /servers/{id}/members  # Join server (with invite)
DELETE /servers/{id}/members/{user_id}  # Remove member/leave
GET    /servers/{id}/roles    # List roles
POST   /servers/{id}/roles    # Create role
```

### Channels
```
GET    /channels/{id}         # Get channel
PATCH  /channels/{id}         # Update channel
DELETE /channels/{id}         # Delete channel
GET    /channels/{id}/messages # List messages (paginated)
POST   /channels/{id}/messages # Send message
GET    /channels/{id}/messages/{msg_id} # Get message
PATCH  /channels/{id}/messages/{msg_id} # Edit message
DELETE /channels/{id}/messages/{msg_id} # Delete message
GET    /channels/{id}/pins    # List pinned messages
PUT    /channels/{id}/pins/{msg_id} # Pin message
DELETE /channels/{id}/pins/{msg_id} # Unpin message
POST   /channels/{id}/typing  # Trigger typing indicator
```

### Direct Messages
```
GET    /users/@me/channels    # List DM channels
POST   /users/@me/channels    # Create/get DM channel
```

### Reactions
```
PUT    /channels/{id}/messages/{msg_id}/reactions/{emoji}/@me  # Add reaction
DELETE /channels/{id}/messages/{msg_id}/reactions/{emoji}/@me  # Remove reaction
GET    /channels/{id}/messages/{msg_id}/reactions/{emoji}      # Get users who reacted
```

### Invites
```
POST   /servers/{id}/invites  # Create invite
GET    /invites/{code}        # Get invite info
POST   /invites/{code}        # Accept invite
DELETE /invites/{code}        # Delete invite
```

### Search
```
GET    /servers/{id}/messages/search?q=query&channel_id=...&author_id=...
```

---

## Feature Modules

### 1. accounts
- User registration, login, password management
- Profile management (avatar, display name, bio)
- Session management with JWT tokens
- Two-factor authentication (optional)

### 2. servers
- Server CRUD operations
- Server settings (name, icon, description)
- Invite code management
- Server discovery (for public servers)

### 3. channels
- Channel types: text, voice, DM, group DM, thread
- Category organization
- Permission overrides per channel
- Slow mode, NSFW flags

### 4. messages
- Message creation with markdown support
- Editing and deletion
- Reply threads
- Attachments and embeds
- System messages (join, leave, pin)

### 5. presence
- Online status (online, idle, DND, invisible)
- Custom status messages
- Activity tracking (playing, streaming, etc.)
- Multi-device presence

### 6. reactions
- Emoji reactions on messages
- Reaction counts and user lists
- Super reactions (animated, premium)

### 7. roles
- Role hierarchy and permissions
- Role assignment to members
- Role colors and display

### 8. notifications
- Push notifications for mentions
- @everyone and @here mentions
- DM notifications
- Notification preferences per channel/server

### 9. search
- Full-text search across messages
- Filters: author, channel, date range, has attachments
- Search highlighting

### 10. threads
- Thread creation from messages
- Thread-specific member list
- Auto-archive after inactivity

---

## WebSocket Hub Architecture

```go
type Hub struct {
    // Active connections by user ID
    connections map[string]map[*Connection]bool

    // Channel subscriptions
    channels map[string]map[*Connection]bool

    // Server subscriptions
    servers map[string]map[*Connection]bool

    // Message channels
    broadcast   chan *Broadcast
    subscribe   chan *Subscription
    unsubscribe chan *Subscription
    register    chan *Connection
    unregister  chan *Connection
}

type Connection struct {
    ID        string
    UserID    string
    Conn      *websocket.Conn
    Send      chan []byte
    Hub       *Hub
    Servers   map[string]bool // Subscribed servers
    Channels  map[string]bool // Subscribed channels
    Sequence  int64           // Event sequence number
    SessionID string
}

type Broadcast struct {
    Target   BroadcastTarget
    TargetID string          // Server ID, Channel ID, or User ID
    Event    string
    Data     any
    Exclude  string          // User ID to exclude
}

type BroadcastTarget int

const (
    TargetServer  BroadcastTarget = iota
    TargetChannel
    TargetUser
)
```

### Connection Lifecycle

1. **Connect**: Client establishes WebSocket connection
2. **Hello**: Server sends heartbeat interval
3. **Identify**: Client sends auth token and intents
4. **Ready**: Server sends initial state (servers, channels, presence)
5. **Heartbeat**: Client sends heartbeat every N ms
6. **Events**: Server dispatches events, client acks
7. **Resume**: On reconnect, client resumes with last sequence

---

## UI Pages

### Landing Page
- Hero section with features
- Login/Register CTAs
- Public server discovery

### App Shell (Post-login)
- **Left Sidebar**: Server list (icons)
- **Channel Sidebar**: Categories and channels
- **Main Area**: Message list with input
- **Right Sidebar**: Member list (collapsible)
- **Header**: Channel name, topic, search, settings

### Server Views
- Server settings (owner/admin)
- Channel management
- Role management
- Member management
- Invite management
- Audit log

### User Settings
- Profile settings
- Privacy settings
- Notification preferences
- Appearance (themes)
- Connections

### Direct Messages
- DM list
- Group DM creation
- Friend requests (optional)

---

## CSS Theme Variables

```css
:root {
    /* Discord-inspired colors */
    --bg-primary: #313338;
    --bg-secondary: #2b2d31;
    --bg-tertiary: #1e1f22;
    --bg-accent: #404249;

    --text-primary: #f2f3f5;
    --text-secondary: #b5bac1;
    --text-muted: #949ba4;
    --text-link: #00a8fc;

    --accent-primary: #5865f2;
    --accent-success: #23a55a;
    --accent-danger: #f23f43;
    --accent-warning: #f0b232;

    --online: #23a55a;
    --idle: #f0b232;
    --dnd: #f23f43;
    --offline: #80848e;

    /* Interactive states */
    --interactive-normal: #b5bac1;
    --interactive-hover: #dbdee1;
    --interactive-active: #ffffff;

    /* Input */
    --input-bg: #1e1f22;
    --input-border: transparent;
    --input-placeholder: #87898c;

    /* Mention highlight */
    --mention-bg: rgba(88, 101, 242, 0.3);
    --mention-text: #c9cdfb;
}

[data-theme="light"] {
    --bg-primary: #ffffff;
    --bg-secondary: #f2f3f5;
    --bg-tertiary: #e3e5e8;
    --bg-accent: #ebedef;

    --text-primary: #060607;
    --text-secondary: #4e5058;
    --text-muted: #6d6f78;
}
```

---

## Message Rendering

### Markdown Support
- **Bold**: `**text**` or `__text__`
- **Italic**: `*text*` or `_text_`
- **Strikethrough**: `~~text~~`
- **Code**: `` `code` `` or ``` ```code block``` ```
- **Quotes**: `> quote`
- **Spoilers**: `||spoiler||`
- **Links**: Auto-link URLs
- **Mentions**: `@username`, `@role`, `#channel`
- **Emoji**: `:emoji_name:` and Unicode

### Embed Types
- URL previews (Open Graph)
- Image embeds
- Video embeds (YouTube, etc.)
- Rich embeds (bots/webhooks)

---

## Delivery Guarantees

### Message Ordering
- Messages ordered by ULID (time-sortable)
- Server assigns message ID
- Client displays optimistically, reconciles

### Delivery States
1. **Sending**: Message submitted, not confirmed
2. **Sent**: Server received, ID assigned
3. **Delivered**: Recipient's client received
4. **Read**: Recipient scrolled to message

### Offline Handling
- Messages queued when offline
- Sent on reconnection
- Conflict resolution by server timestamp

### Read Receipts
- Per-channel read state
- Batch ack on scroll
- Unread counts per channel

---

## Performance Considerations

### Message Pagination
- Load 50 messages initially
- Infinite scroll (both directions)
- Jump to message by ID
- "New messages" indicator

### Presence Updates
- Batch presence updates
- Rate limit typing indicators (5s debounce)
- Lazy load member list

### Caching Strategy
- Cache messages in IndexedDB
- Cache channel metadata
- Invalidate on relevant events

---

## Security

### Authentication
- JWT with refresh tokens
- Token stored in httpOnly cookie (web)
- Rate limiting on auth endpoints

### Authorization
- Permission checks on all operations
- Channel-level permission overrides
- Role hierarchy enforcement

### Content Security
- HTML sanitization on render
- File upload validation
- Rate limiting on messages

---

## CLI Commands

```bash
# Initialize database
chat init

# Start server
chat serve --addr :8080 --theme default

# Seed sample data
chat seed --users 100 --servers 5 --messages 10000

# Create admin user
chat admin create --username admin --password secret

# Generate invite
chat invite create --server-id <id> --max-uses 100
```

---

## Directory Structure

```
chat/
├── cmd/chat/main.go
├── cli/
│   ├── root.go
│   ├── serve.go
│   ├── init.go
│   ├── seed.go
│   └── ui.go
├── app/web/
│   ├── server.go
│   ├── ws/
│   │   ├── hub.go
│   │   ├── connection.go
│   │   ├── events.go
│   │   └── handlers.go
│   └── handler/
│       ├── auth.go
│       ├── server.go
│       ├── channel.go
│       ├── message.go
│       ├── member.go
│       ├── reaction.go
│       ├── response.go
│       └── page.go
├── feature/
│   ├── accounts/
│   ├── servers/
│   ├── channels/
│   ├── messages/
│   ├── members/
│   ├── roles/
│   ├── presence/
│   ├── reactions/
│   ├── notifications/
│   └── search/
├── store/duckdb/
│   ├── store.go
│   ├── schema.sql
│   └── *_store.go
├── assets/
│   ├── embed.go
│   ├── static/
│   │   ├── css/
│   │   └── js/
│   └── views/
│       ├── layouts/
│       ├── pages/
│       └── components/
├── pkg/
│   ├── ulid/
│   ├── password/
│   ├── jwt/
│   └── markdown/
├── go.mod
├── Makefile
└── README.md
```

---

## Implementation Phases

### Phase 1: Core Infrastructure
- Database schema
- User authentication
- Server/channel CRUD
- Basic message send/receive

### Phase 2: Realtime
- WebSocket hub
- Event dispatch
- Presence system
- Typing indicators

### Phase 3: Rich Features
- Reactions
- Message editing/deletion
- Replies and threads
- File attachments

### Phase 4: Polish
- Search
- Notifications
- Role permissions
- UI refinements

---

## Success Metrics

- Message latency < 100ms (p95)
- Support 10,000 concurrent connections per instance
- 99.9% message delivery rate
- Full-text search < 500ms
