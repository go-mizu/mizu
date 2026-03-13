# 0724 — Discord Scraper

## Overview

Scrape Discord guilds, channels, messages, and user profiles into a local DuckDB database using a Discord user token. The implementation follows the dedicated scraper pattern already used by `pkg/scrape/spotify` and `pkg/scrape/goodread`: entity-specific task structs implementing `pkg/core.Task[State, Metric]`, a bucket-aware REST API client, a separate state DB for the crawl queue, and CLI subcommands under `cli/discord.go`.

Primary sources used for the design:

- Discord REST API v10 at `https://discord.com/api/v10/`
- User token authentication (`Authorization: <token>` header, no "Bearer" prefix)
- Bucket-based rate limiting via `X-RateLimit-*` response headers
- Cursor-based message pagination via `?before=<snowflake>&limit=100`

This implementation uses a user token (not a bot token) to access all guilds, channels, and messages that the authenticated account can see.

**Test server:** Reactiflux (discord.gg/reactiflux) — guild ID `102860784329052160`. One of the largest public programming communities on Discord (React, JavaScript, TypeScript ecosystem).

## Scope

Implemented entity types:

- `guild` — server metadata
- `channel` — text channels within a guild
- `message_page` — paginated message fetch (100 msgs/page, cursor-based)
- `user` — user profiles

Implemented crawl capabilities:

- `me` — fetch `/users/@me`, list joined guilds, enqueue all
- Fetch a single guild by ID or URL → enqueue all text channels
- Fetch a single channel → enqueue first message page
- Fetch a page of messages → store rows, enqueue authors as users, enqueue next page
- Fetch a single user profile
- Bulk-crawl the queue with worker concurrency using `pkg/core.Task`
- Inspect queue/jobs/stats from CLI
- Seed the queue from a file of guild/channel IDs

Explicitly out of scope for this iteration:

- Voice channels and stage channels
- DM / group DM channels (not accessible without additional API calls)
- Reactions, attachments, embeds stored as separate rows (stored as JSON in message row)
- Thread enumeration beyond active threads list
- Guild member list crawl (requires GUILD_MEMBERS privileged intent; only available for bot tokens)
- Webhook messages

## Package Layout

```text
blueprints/search/
├── pkg/scrape/discord/
│   ├── types.go           # Guild, Channel, Message, User, QueueItem, DBStats, ParsedRef
│   ├── config.go          # Config + DefaultConfig()
│   ├── client.go          # Discord REST API v10 client + bucket-aware rate limiter
│   ├── db.go              # discord.duckdb schema + upsert methods
│   ├── state.go           # state.duckdb queue/jobs/visited (same as spotify)
│   ├── display.go         # stats + crawl progress printing
│   ├── task_guild.go      # GuildTask → fetch guild → enqueue text channels
│   ├── task_channel.go    # ChannelTask → fetch channel info → enqueue message pages
│   ├── task_messages.go   # MessagesTask → fetch 100 msgs → store + enqueue users + next page
│   ├── task_user.go       # UserTask → fetch user profile
│   └── task_crawl.go      # CrawlTask → bulk queue dispatcher
└── cli/discord.go         # CLI subcommands
```

## API Client Design

### Authentication

Discord user token is stored in `Config.Token`. Every request sets:

```
Authorization: <token>
User-Agent: Mozilla/5.0 (compatible; discord-archiver/1.0)
Content-Type: application/json
```

No "Bearer" prefix — user tokens are passed raw.

### Rate Limiter

Discord enforces per-route buckets. The client tracks:

- `buckets map[string]*routeBucket` keyed by `X-RateLimit-Bucket` header value
- Each bucket: `remaining int`, `resetAt time.Time`
- Global rate limit flag: `globalResetAt time.Time`

On every response:
- Parse `X-RateLimit-Bucket`, `X-RateLimit-Remaining`, `X-RateLimit-Reset-After`
- On 429: parse body `{"retry_after": 1.5, "global": true/false}`, sleep `retry_after` seconds

Additional floor: `Config.Delay` (default 500ms) enforced globally between requests.

Route key format: `METHOD /path/template` (e.g. `GET /guilds/{guild_id}`).

### Key Endpoints

| Entity | Method | Endpoint |
|--------|--------|----------|
| Self | GET | `/users/@me` |
| Joined guilds | GET | `/users/@me/guilds` |
| Guild info | GET | `/guilds/{guild_id}` |
| Guild channels | GET | `/guilds/{guild_id}/channels` |
| Channel info | GET | `/channels/{channel_id}` |
| Messages (page) | GET | `/channels/{channel_id}/messages?before={id}&limit=100` |
| User info | GET | `/users/{user_id}` |

### URL/ID Normalization

Accepted input forms:
- raw snowflake ID: `102860784329052160`
- Discord channel URL: `https://discord.com/channels/{guild_id}/{channel_id}`
- Discord guild invite: `https://discord.gg/{code}` (invite resolve not implemented; use guild ID)
- Queue URL scheme: `discord://guilds/{id}`, `discord://channels/{id}`, `discord://channels/{channel_id}/messages?before={id}`

`ParseRef(raw, expected)` normalizes any of the above to a `ParsedRef{EntityType, ID, URL}`.

## Data Model

Driver: `github.com/duckdb/duckdb-go/v2`

Arrays/JSON stored as `VARCHAR` (JSON text).

### discord.duckdb

```sql
CREATE TABLE IF NOT EXISTS guilds (
  guild_id          VARCHAR PRIMARY KEY,
  name              VARCHAR,
  description       VARCHAR,
  icon_url          VARCHAR,
  member_count      BIGINT,
  approximate_presence_count BIGINT,
  owner_id          VARCHAR,
  region            VARCHAR,
  features_json     VARCHAR,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS channels (
  channel_id        VARCHAR PRIMARY KEY,
  guild_id          VARCHAR,
  name              VARCHAR,
  channel_type      INTEGER,
  topic             VARCHAR,
  position          INTEGER,
  parent_id         VARCHAR,
  nsfw              BOOLEAN,
  last_message_id   VARCHAR,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
  message_id        VARCHAR PRIMARY KEY,
  channel_id        VARCHAR,
  guild_id          VARCHAR,
  author_id         VARCHAR,
  author_username   VARCHAR,
  content           VARCHAR,
  timestamp         TIMESTAMP,
  edited_timestamp  TIMESTAMP,
  message_type      INTEGER,
  pinned            BOOLEAN,
  mention_everyone  BOOLEAN,
  attachments_json  VARCHAR,
  embeds_json       VARCHAR,
  reactions_json    VARCHAR,
  referenced_message_id VARCHAR,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_messages_channel ON messages(channel_id, message_id);
CREATE INDEX IF NOT EXISTS idx_messages_author  ON messages(author_id);

CREATE TABLE IF NOT EXISTS users (
  user_id           VARCHAR PRIMARY KEY,
  username          VARCHAR,
  global_name       VARCHAR,
  discriminator     VARCHAR,
  avatar_url        VARCHAR,
  bot               BOOLEAN,
  fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### state.duckdb

Same queue/jobs/visited pattern as `pkg/scrape/spotify/state.go`.

`entity_type` values:

- `guild` — priority 15
- `channel` — priority 12
- `message_page` — priority 8 (URL encodes cursor: `discord://channels/{id}/messages?before={snowflake}`)
- `user` — priority 5

## Task Behavior

### `GuildTask`

- parse input to guild ID
- check visited; skip if already done
- `GET /guilds/{guild_id}?with_counts=true`
- store guild row
- `GET /guilds/{guild_id}/channels`
- enqueue all text channels (type=0) and news channels (type=5) at priority 12
- mark visited

### `ChannelTask`

- parse input to channel ID
- `GET /channels/{channel_id}`
- store channel row
- enqueue first message page: `discord://channels/{channel_id}/messages?before=` (empty = fetch latest)
- mark visited

### `MessagesTask`

- parse channel ID + optional `before` cursor from URL
- `GET /channels/{channel_id}/messages?limit=100[&before={id}]`
- store all message rows
- for each message: enqueue author as `user` at priority 5 (if not visited)
- if response contains 100 messages: enqueue next page with `before=<oldest_message_id_in_batch>`
- mark this page URL as visited

### `UserTask`

- parse user ID
- check visited; skip if already done
- `GET /users/{user_id}`
- store user row
- mark visited

### `CrawlTask`

Queue-driven worker pool — identical structure to `pkg/scrape/spotify/task_crawl.go`:

- pop pending rows from `state.duckdb`
- dispatch to entity-specific tasks via switch on `entity_type`
- emit periodic `CrawlState` snapshots (done/pending/failed/rps)

## CLI

`search discord`

Subcommands:

- `me` — fetch `/users/@me`, list and enqueue all joined guilds
- `guild <id>` — fetch a single guild → enqueue channels
- `channel <id>` — fetch a single channel → enqueue messages
- `messages <channel_id> [--before <snowflake>] [--limit N]` — fetch a page of messages
- `user <id>` — fetch a single user profile
- `seed --file <path> [--entity guild|channel] [--priority N]` — seed queue from file
- `crawl [--workers N] [--delay MS]` — bulk crawl from queue
- `info` — DB stats + queue depth
- `jobs [--limit N]` — list recent jobs
- `queue [--status pending] [--limit N]` — inspect queue items

Token flag: `--token` (default: `$DISCORD_TOKEN` env var).

Examples:

```bash
# Set token
export DISCORD_TOKEN="your_user_token_here"

# Test with Reactiflux (large public coding server)
search discord guild 102860784329052160
search discord channel 103882387330457600   # #general channel in Reactiflux
search discord messages 103882387330457600

# Or enqueue all joined guilds at once
search discord me

# Bulk crawl everything
search discord crawl --workers 2 --delay 500

# Inspect
search discord info
search discord queue --status pending --limit 20
```

## Rate Limit Notes

Discord enforces strict per-route buckets:

- Message fetch (`GET /channels/{id}/messages`) typically allows 5 requests per bucket window
- User fetch (`GET /users/{id}`) shares a bucket with generous limits
- Guild fetch is rarely rate-limited in practice for single-server usage
- With `DefaultDelay=500ms` and `DefaultWorkers=2`, the scraper stays well within limits

## External Learnings Applied

From popular Discord archiving tools on GitHub (e.g., DiscordChatExporter, discord-scraper):

- Always paginate messages using snowflake `before` cursor, not page numbers
- Store raw `attachments` and `embeds` as JSON — don't try to normalize every field
- User profile data from messages (author object) is sufficient for most analysis; `GET /users/{id}` is only needed for bio/banner
- Respect 429 `retry_after` precisely — Discord bans accounts that ignore rate limits
- Filter channels by type=0 (GUILD_TEXT) and type=5 (GUILD_NEWS) to avoid voice/category channels

## Validation

Implementation acceptance criteria:

- `go build ./cli/...` succeeds
- `search discord guild 102860784329052160` fetches Reactiflux and enqueues channels
- `search discord messages <channel_id>` stores at least one message row
- `search discord crawl` processes queue without 429 errors
- `search discord info` shows correct row counts per entity type
