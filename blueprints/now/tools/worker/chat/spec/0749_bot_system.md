# 0749: Bot Registry System + ClaudeStatus Bot

## Problem

`src/bots.ts` uses a hardcoded `BUILT_IN_BOTS` Set and a growing if/else chain to dispatch bot replies. Adding a new bot requires edits in two places (the Set and the dispatch chain), and there is no standard place for bot metadata (bio, sample questions). Echo and Chinese bots live inline in `bots.ts` rather than in dedicated modules. There is no bot for Claude service status queries.

## Goals

1. Replace the hardcoded dispatch chain with a self-registering bot registry.
2. Move `echo` and `chinese` into `src/bot/` alongside `scout`.
3. Each bot owns its profile (bio + example questions) — no central metadata file.
4. Add `a/claudestatus` bot that answers questions about Claude uptime, incidents, and component health by calling `status.claude.com/api/v2/` with two-layer caching (in-memory + D1).
5. Surface bot bio and sample questions in the chat view (empty-thread welcome panel) and on the landing page.

## Non-Goals

- External/user-registered bots with dynamic profiles stored in D1.
- Bot-to-bot communication.
- Webhook or push notifications for incidents.

---

## Architecture

### Registry (`src/bot/registry.ts`)

Single source of truth for all built-in bots.

```ts
interface BotProfile {
  bio: string;
  examples: string[];
}

interface BotDef {
  actor: string;
  profile: BotProfile;
  reply: (msg: string, db: D1Database) => Promise<string> | string;
}

const registry = new Map<string, BotDef>();

export function registerBot(def: BotDef): void
export function isBuiltInBot(actor: string): boolean
export function getBotProfile(actor: string): BotProfile | null
export async function dispatchReply(actor: string, msg: string, db: D1Database): Promise<string | null>
```

`dispatchReply` returns `null` if the actor is not registered, allowing `handleBotReply` to skip the DB write.

### `src/bots.ts` (refactored)

```ts
import "./bot/echo";
import "./bot/chinese";
import "./bot/scout";
import "./bot/claudestatus";
export { isBuiltInBot, getBotProfile, dispatchReply } from "./bot/registry";

export async function handleBotReply(
  db: D1Database,
  chatId: string,
  botActor: string,
  userMessage: string
): Promise<void> {
  const replyText = await dispatchReply(botActor, userMessage, db);
  if (replyText === null) return;

  const id = messageId();
  const now = Date.now();
  await db
    .prepare("INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES (?, ?, ?, ?, ?, ?)")
    .bind(id, chatId, botActor, replyText, null, now)
    .run();
}
```

---

## Bot Modules

### `src/bot/echo/index.ts`

```ts
registerBot({
  actor: "a/echo",
  profile: {
    bio: "Echo repeats your message back verbatim. Useful for testing the chat pipeline.",
    examples: ["hello world", "test message", "ping"],
  },
  reply: (msg) => `Echo: ${msg}`,
});
```

### `src/bot/chinese/index.ts`

Extracts the translation logic from `bots.ts`. Calls `api.mymemory.translated.net` live (no caching needed — external service handles rate limiting).

```ts
registerBot({
  actor: "a/chinese",
  profile: {
    bio: "Translates your message from English to Chinese (Simplified).",
    examples: ["good morning", "how are you?", "deploy complete"],
  },
  reply: async (msg) => { /* existing translation logic */ },
});
```

### `src/bot/scout/index.ts`

Wraps existing `scoutReply` from `src/bot/scout/` with a registration call. No logic changes to `intent.ts`, `format.ts`, or `data.ts`.

```ts
registerBot({
  actor: "a/scout",
  profile: {
    bio: "Scout is your football companion. Ask about standings, fixtures, and club info across 7 major leagues.",
    examples: [
      "Premier League table",
      "When is Arsenal's next match?",
      "Tell me about Barcelona",
      "Champions League fixtures",
    ],
  },
  reply: (msg) => scoutReply(msg),
});
```

---

## ClaudeStatus Bot (`src/bot/claudestatus/`)

### Files

```
src/bot/claudestatus/
  index.ts     — registerBot + claudestatusReply()
  intent.ts    — detectIntent()
  format.ts    — formatStatus, formatComponents, formatIncidents,
                 formatIncidentDetail, formatUptime, formatHelp
  fetch.ts     — fetchWithCache(key, url, db): two-layer cache
```

### Intent Detection (`intent.ts`)

```ts
export type Intent =
  | "status"
  | "components"
  | "incidents"
  | "incident_detail"
  | "uptime"
  | "help";
```

Keyword groups (case-insensitive, padded with spaces):

| Intent | Keywords |
|---|---|
| `status` | "status", "down", "outage", "operational", "is claude", "is it up", "working" |
| `components` | "api", "claude code", "platform.claude", "claude.ai", "government", "component" |
| `incidents` | "incident", "what happened", "recent", "history", "past issues" |
| `incident_detail` | "latest incident", "last incident", "tell me about", "most recent incident" |
| `uptime` | "uptime", "99", "reliability", "availability", "sla", "percentage" |
| `help` | fallback |

Priority order: `incident_detail` > `incidents` > `uptime` > `components` > `status` > `help`.

### Cache Layer (`fetch.ts`)

**In-memory cache:**
```ts
const memCache = new Map<string, { data: unknown; expiresAt: number }>();
const MEM_TTL_MS = 5 * 60 * 1000; // 5 minutes
```

**D1 cache table** (new migration `migrate-0748.sql`):
```sql
CREATE TABLE IF NOT EXISTS bot_cache (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bot_cache_expires ON bot_cache(expires_at);
```
D1 TTL: 15 minutes.

**Read path:**
1. Check memory cache — if hit and not expired, return.
2. Check D1 `bot_cache` — if row exists and `expires_at > now`, parse JSON, populate memory, return.
3. Fetch from `status.claude.com/api/v2/<endpoint>.json`.
4. On success: write D1 row (upsert), write memory entry, return data.
5. On fetch failure: if stale D1 row exists (expired), return it as fallback with a staleness note. If nothing, return `null`.

**Cache keys:** `claudestatus:summary`, `claudestatus:incidents`, `claudestatus:components`.

### API Endpoints Used

| Data needed | Endpoint |
|---|---|
| Overall status + component list | `/api/v2/summary.json` |
| Incident list + detail | `/api/v2/incidents.json` |

Uptime percentages are not available in the JSON API (they are rendered client-side on the status page). The `uptime` intent uses the component statuses from `summary.json` as a proxy and notes the limitation.

### Response Format (`format.ts`)

All responses use Markdown (rendered client-side in chat view).

**`formatStatus`** — overall indicator + per-component status badges.

**`formatComponents`** — table: component name | status | last updated.

**`formatIncidents`** — list of last 5 incidents: name, impact, date, duration.

**`formatIncidentDetail`** — most recent incident full timeline (each `incident_update` with status and body).

**`formatUptime`** — disclaimer about API limitation + component operational status as best proxy.

**`formatHelp`** — bot intro + example questions.

### Profile

```ts
{
  bio: "ClaudeStatus monitors Anthropic's services in real time. Ask about current status, recent incidents, component health, or uptime.",
  examples: [
    "Is Claude down?",
    "Any recent incidents?",
    "Latest incident details",
    "Is the API up?",
    "What's the uptime?",
  ],
}
```

---

## Frontend Changes

### Chat View (`src/chatview.ts`)

When the thread is empty and the peer is a built-in bot (`peerActor?.startsWith("a/")` and `getBotProfile(peerActor)` returns a value), render a welcome panel inside `#thread`:

```html
<div class="bot-welcome">
  <div class="bot-welcome-avatar"><!-- botAvatar() --></div>
  <div class="bot-welcome-name">claudestatus</div>
  <div class="bot-welcome-bio">{bio}</div>
  <div class="bot-welcome-chips">
    <button class="chip" onclick="fillInput(this)">{example}</button>
    ...
  </div>
</div>
```

`fillInput(btn)` sets `#msg-input` value to `btn.textContent` and focuses it. The panel disappears as soon as the first message is appended (it sits before the message list in DOM order, hidden once `thread` has `.has-messages` class).

### Landing Page (`src/landing.ts`)

Add a `"Talk to ClaudeStatus 📡"` fieldset after the Scout section, matching the same `fieldset.s` + `.convo` pattern:

```
you          is claude down?
claudestatus ✅ All Systems Operational — claude.ai, API, Claude Code, platform all green.
you          any recent incidents?
claudestatus ⚠ 2 incidents today — "Elevated errors on Claude Sonnet 4.6" (minor, resolved).
```

Ends with: `Message **a/claudestatus** directly, or tag **@claudestatus** in any room.`

---

## D1 Migration

New file: `migrate-0748.sql`

```sql
CREATE TABLE IF NOT EXISTS bot_cache (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bot_cache_expires ON bot_cache(expires_at);
```

Applied to the D1 database before deploying. Old `migrate-0746.sql` / `migrate-0746b.sql` remain untouched.

---

## Error Handling

| Scenario | Behaviour |
|---|---|
| API fetch fails, D1 has fresh data | Serve from D1, no error shown |
| API fetch fails, D1 has stale data | Serve stale data + note: "⚠ Data may be up to X minutes old." |
| API fetch fails, no D1 data | Return: "⚠ Could not reach status.claude.com. Try again in a moment." |
| API returns non-200 | Same as fetch failure |
| D1 write fails | Log silently, serve data anyway (memory cache still populated) |

---

## Testing

- `src/bot/claudestatus/intent.test.ts` — unit tests for `detectIntent()` covering all 6 intents + edge cases.
- `src/bot/claudestatus/format.test.ts` — unit tests for each format function with fixture API responses.
- `src/bot/registry.test.ts` — verify `registerBot`, `isBuiltInBot`, `getBotProfile`, `dispatchReply` contract.
- Echo and Chinese: no new tests needed (logic unchanged, just moved).
- Scout: existing tests in `src/bot/scout/intent.test.ts` and `format.test.ts` remain valid.

---

## File Changelist

| Action | File |
|---|---|
| New | `src/bot/registry.ts` |
| New | `src/bot/echo/index.ts` |
| New | `src/bot/chinese/index.ts` |
| Modified | `src/bot/scout/index.ts` (add registerBot wrapper) |
| New | `src/bot/claudestatus/index.ts` |
| New | `src/bot/claudestatus/intent.ts` |
| New | `src/bot/claudestatus/format.ts` |
| New | `src/bot/claudestatus/fetch.ts` |
| New | `src/bot/claudestatus/intent.test.ts` |
| New | `src/bot/claudestatus/format.test.ts` |
| New | `src/bot/registry.test.ts` |
| Modified | `src/bots.ts` (refactored to thin wrapper) |
| Modified | `src/chatview.ts` (bot welcome panel) |
| Modified | `src/landing.ts` (ClaudeStatus section) |
| New | `migrate-0748.sql` |
