# 0740 chat worker

## Objective

Deploy a Cloudflare Worker at `chat.go-mizu.workers.dev` that serves a landing page, API documentation, and the chat HTTP API. Uses D1 for persistent storage, Bearer token auth via `MIZU_CHAT_API_TOKEN`, and follows the same patterns as `blueprints/search/tools/browser`.

## User Requirements

1. Landing page at `GET /` — clean, monochrome design inspired by here.now.
2. Documentation at `GET /docs` — full API reference rendered as HTML from markdown content.
3. Chat API at `/api/*` — six endpoints protected by Bearer token.
4. D1 database (`chat-db`) stores chats, members, and messages.
5. Security: timing-safe auth, membership enforcement, actor validation, input limits.

## Non-Goals

1. WebSocket / real-time push.
2. Rate limiting or abuse protection.
3. File attachments or rich media.

## Pages

### Landing (`GET /`)

Monochrome design with:
- Hero section: title, tagline, CTA to docs
- Feature cards: API-first, D1-powered, agent-ready
- Quick start code snippet
- Footer with links

### Docs (`GET /docs`)

Single-page API reference covering:
- Overview and quick start
- Authentication (Bearer token, X-Actor header)
- All six API endpoints with request/response examples
- Error format
- Pagination

## API Routes

All routes under `/api/chat`. Request/response JSON matches `api/chat/README.md`.

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | /api/chat | createChat | Create a chat |
| GET | /api/chat | listChats | List chats |
| GET | /api/chat/:id | getChat | Get a chat |
| POST | /api/chat/:id/join | joinChat | Join a chat |
| POST | /api/chat/:id/messages | sendMessage | Send a message |
| GET | /api/chat/:id/messages | listMessages | List messages |

## D1 Schema

```sql
CREATE TABLE IF NOT EXISTS chats (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  creator TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'public',
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS members (
  chat_id TEXT NOT NULL,
  actor TEXT NOT NULL,
  joined_at INTEGER NOT NULL,
  PRIMARY KEY (chat_id, actor),
  FOREIGN KEY (chat_id) REFERENCES chats(id)
);

CREATE TABLE IF NOT EXISTS messages (
  id TEXT PRIMARY KEY,
  chat_id TEXT NOT NULL,
  actor TEXT NOT NULL,
  text TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  FOREIGN KEY (chat_id) REFERENCES chats(id)
);

CREATE INDEX IF NOT EXISTS idx_messages_chat ON messages(chat_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_members_chat ON members(chat_id);
```

## Security

1. **Timing-safe token comparison** — SHA-256 hash both sides, constant-time XOR.
2. **Actor format validation** — must match `^[ua]/[\w.@-]{1,64}$`.
3. **Membership enforcement** — send requires membership; private chats require membership to read.
4. **Visibility enforcement** — private chats hidden from list, blocked from join.
5. **Input length limits** — title 200 chars, text 4000 chars.
6. **JSON parse error handling** — returns 400 not 500.
7. **Cursor scoping** — before cursor scoped to chat_id.

## File Layout

```
tools/worker/chat/
├── package.json
├── wrangler.toml
├── tsconfig.json
├── schema.sql
└── src/
    ├── index.ts       # Hono app, routes, pages
    ├── types.ts       # Env, request/response types
    ├── auth.ts        # Timing-safe Bearer token middleware
    ├── actor.ts       # Actor validation + membership check
    ├── chat.ts        # Chat CRUD handlers
    ├── message.ts     # Message handlers
    ├── id.ts          # ID generation
    ├── landing.ts     # Landing page HTML
    └── docs.ts        # Docs page HTML
```

## Deployment

```bash
cd blueprints/now/worker
npm install
npx wrangler d1 create chat-db
npx wrangler secret put AUTH_TOKEN    # paste MIZU_CHAT_API_TOKEN value
npm run db:migrate:remote
npx wrangler deploy
```

## Acceptance Criteria

1. `GET /` returns the landing page HTML.
2. `GET /docs` returns the docs page HTML with full API reference.
3. `POST /api/chat` with valid Bearer token creates a chat and returns 201.
4. `GET /api/chat` lists chats with `items` array.
5. `GET /api/chat/:id` returns a single chat.
6. `POST /api/chat/:id/join` returns 204.
7. `POST /api/chat/:id/messages` creates a message and returns 201.
8. `GET /api/chat/:id/messages` returns messages with cursor pagination.
9. All `/api/*` routes return 401 without valid Bearer token.
10. Membership enforcement works for send and private chat access.
