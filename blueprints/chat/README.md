# Chat Blueprint

A realtime chat application inspired by Discord and Slack, built with Mizu.

## Features

- **Realtime Messaging**: WebSocket-based instant message delivery
- **Servers & Channels**: Organize conversations in servers with text channels
- **Direct Messages**: Private conversations between users
- **Presence System**: Online/offline status with typing indicators
- **Message Features**: Reactions, replies, editing, pinning
- **Role-Based Permissions**: Fine-grained access control
- **Search**: Full-text search across messages

## Quick Start

```bash
# Initialize the database
make init

# Seed sample data (optional)
make seed

# Start the server
make run

# Or build and run
make build
chat serve
```

Then open http://localhost:8080 in your browser.

## Commands

```bash
chat serve     # Start the web server
chat init      # Initialize the database
chat seed      # Seed sample data
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Create account
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/logout` - Logout
- `GET /api/v1/auth/me` - Get current user

### Servers
- `GET /api/v1/servers` - List user's servers
- `POST /api/v1/servers` - Create server
- `GET /api/v1/servers/{id}` - Get server
- `PATCH /api/v1/servers/{id}` - Update server
- `DELETE /api/v1/servers/{id}` - Delete server

### Channels
- `GET /api/v1/servers/{id}/channels` - List channels
- `POST /api/v1/servers/{id}/channels` - Create channel
- `GET /api/v1/channels/{id}` - Get channel
- `PATCH /api/v1/channels/{id}` - Update channel
- `DELETE /api/v1/channels/{id}` - Delete channel

### Messages
- `GET /api/v1/channels/{id}/messages` - List messages
- `POST /api/v1/channels/{id}/messages` - Send message
- `PATCH /api/v1/channels/{id}/messages/{msg_id}` - Edit message
- `DELETE /api/v1/channels/{id}/messages/{msg_id}` - Delete message
- `POST /api/v1/channels/{id}/typing` - Trigger typing indicator

### Reactions
- `PUT /api/v1/channels/{id}/messages/{msg_id}/reactions/{emoji}` - Add reaction
- `DELETE /api/v1/channels/{id}/messages/{msg_id}/reactions/{emoji}` - Remove reaction

## WebSocket

Connect to `/ws?token={session_token}` for realtime events.

### Events
- `MESSAGE_CREATE` - New message
- `MESSAGE_UPDATE` - Message edited
- `MESSAGE_DELETE` - Message deleted
- `TYPING_START` - User typing
- `PRESENCE_UPDATE` - User status change
- `CHANNEL_CREATE` - Channel created
- `CHANNEL_UPDATE` - Channel updated
- `CHANNEL_DELETE` - Channel deleted
- `MEMBER_ADD` - Member joined
- `MEMBER_REMOVE` - Member left

## Architecture

```
chat/
├── cmd/chat/          # CLI entry point
├── cli/               # CLI commands
├── app/web/           # HTTP server & handlers
│   ├── handler/       # HTTP handlers
│   └── ws/            # WebSocket hub
├── feature/           # Business logic
│   ├── accounts/      # User management
│   ├── servers/       # Server management
│   ├── channels/      # Channel management
│   ├── messages/      # Message management
│   ├── members/       # Member management
│   ├── roles/         # Role & permissions
│   └── presence/      # User presence
├── store/duckdb/      # Database layer
├── assets/            # Static files & templates
└── pkg/               # Shared utilities
```

## Sample Users

After running `make seed`:

| Username | Email | Password |
|----------|-------|----------|
| alice | alice@example.com | password123 |
| bob | bob@example.com | password123 |
| charlie | charlie@example.com | password123 |
| diana | diana@example.com | password123 |
