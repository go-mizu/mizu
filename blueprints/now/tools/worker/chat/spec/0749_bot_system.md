# 0749: Bot Registry System + ClaudeStatus Bot

## Problem

`src/bots.ts` uses a hardcoded `BUILT_IN_BOTS` Set and a growing if/else chain to dispatch bot replies. Adding a new bot requires edits in two places (the Set and the dispatch chain), and there is no standard place for bot metadata (bio, sample questions). Echo and Chinese bots live inline in `bots.ts` rather than in dedicated modules. Room @mention dispatch in `src/message.ts` is also hardcoded per-bot (separate regexes for `@chinese` and `@scout`). There is no bot for Claude service status queries.

## Goals

1. Replace the hardcoded dispatch chain with a self-registering bot registry.
2. Move `echo` and `chinese` into `src/bot/` alongside `scout`.
3. Each bot owns its profile (bio + example questions) — no central metadata file.
4. Refactor room @mention dispatch in `message.ts` to iterate the registry — no hardcoded bot names.
5. Add `a/claudestatus` bot that answers questions about Claude uptime, incidents, and component health by calling `status.claude.com/api/v2/` with two-layer caching (in-memory + D1).
6. Surface bot bio and sample questions in the chat view (empty-thread welcome panel) and on the landing page.

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
  actor: string;         // e.g. "a/claudestatus"
  profile: BotProfile;
  reply: (msg: string, db: D1Database) => Promise<string> | string;
}

const registry = new Map<string, BotDef>();

export function registerBot(def: BotDef): void  // throws if actor already registered
export function isBuiltInBot(actor: string): boolean
export function getBotProfile(actor: string): BotProfile | null
export function listBotActors(): string[]           // used by message.ts @mention dispatch
export async function dispatchReply(actor: string, msg: string, db: D1Database): Promise<string | null>
```

`dispatchReply` returns `null` if the actor is not registered, allowing `handleBotReply` to skip the DB write.

**Module load order:** Bot modules self-register via top-level side effects (`registerBot(...)` runs at import time). Cloudflare Workers use ES module syntax; top-level side effects run once per isolate startup, which is reliable. The esbuild bundler used by Wrangler must be told these imports are not tree-shakeable. In `wrangler.toml` or the esbuild config, no special flag is needed for side-effect imports that are explicitly listed — but `bots.ts` must **not** use `import type` for these, and the imports must reference the module directly (not re-exported through an index barrel). This is the existing Wrangler/esbuild default behavior for named imports.

### `src/bots.ts` (refactored)

```ts
// Side-effect imports register each bot into the registry
import "./bot/echo";
import "./bot/chinese";
import "./bot/scout";
import "./bot/claudestatus";

export { isBuiltInBot, getBotProfile, listBotActors, dispatchReply } from "./bot/registry";
export { messageId } from "./id"; // re-exported for callers that previously imported from bots

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

### `src/message.ts` (room @mention dispatch — refactored)

The existing `sendMessageExplicit` room branch (lines 202–211) has hardcoded regexes for `@chinese` and `@scout`. Replace with a registry-driven loop:

```ts
} else if (chat.kind === "room") {
  for (const botActor of listBotActors()) {
    const shortName = botActor.slice(2); // "a/scout" -> "scout"
    const pattern = new RegExp(`^@${shortName}\\s+([\\s\\S]+)`, "i");
    const match = text.match(pattern);
    if (match) {
      c.executionCtx.waitUntil(handleBotReply(c.env.DB, chatIdParam, botActor, match[1].trim()));
      break; // only one bot per message
    }
  }
}
```

This means any registered bot automatically gets room @mention support. No further changes to `message.ts` are needed when adding new bots.

**Behavior change note:** The existing code checks `@chinese` and `@scout` in two separate if-blocks with no early exit, so a message starting with `@chinese` could theoretically also match a `@scout` pattern. The new loop uses `break` after the first match, ensuring only one bot responds per message. This is an intentional correctness improvement.

---

## Bot Modules

### `src/bot/echo/index.ts`

```ts
import { registerBot } from "../registry";

registerBot({
  actor: "a/echo",
  profile: {
    bio: "Echo repeats your message back verbatim. Useful for testing the chat pipeline.",
    examples: ["hello world", "test message", "ping"],
  },
  reply: (msg) => `Echo: ${msg}`,
  // db parameter is part of the BotDef interface but not used by echo
});
```

### `src/bot/chinese/index.ts`

Extracts the translation logic verbatim from `bots.ts`. Calls `api.mymemory.translated.net` live (no caching needed — external service handles rate limiting).

```ts
import { registerBot } from "../registry";

registerBot({
  actor: "a/chinese",
  profile: {
    bio: "Translates your message from English to Chinese (Simplified).",
    examples: ["good morning", "how are you?", "deploy complete"],
  },
  reply: async (msg) => {
    let translated = msg;
    try {
      const res = await fetch(
        `https://api.mymemory.translated.net/get?q=${encodeURIComponent(msg)}&langpair=en|zh-CN`,
        { headers: { "User-Agent": "chat.now/1.0" } }
      );
      if (res.ok) {
        const data = await res.json() as { responseData?: { translatedText?: string } };
        if (data?.responseData?.translatedText) translated = data.responseData.translatedText;
      }
    } catch { /* keep original if translation fails */ }
    return `回声：${translated}`;
  },
});
```

### `src/bot/scout/index.ts`

The existing `src/bot/scout/index.ts` already exports `scoutReply` and is the entry point for the scout module. Add a `registerBot` call at the bottom of the same file — no new import of `scoutReply` is needed since it is defined in the same module. The existing `intent.ts`, `format.ts`, and `data.ts` files are **unchanged**.

```ts
// (existing scoutReply function stays at top of file, unchanged)
// Add at the bottom:

import { registerBot } from "../registry";

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
  // db parameter is part of BotDef interface but not used by scout (synchronous, no DB access)
});

// scoutReply is already exported at the top of this file; no re-export needed
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

Keyword groups (case-insensitive, message padded with spaces for word-boundary matching):

| Intent | Keywords |
|---|---|
| `status` | "status", "down", "outage", "operational", "is claude", "is it up", "working" |
| `components` | "api", "claude code", "platform", "claude.ai", "government", "component" |
| `incidents` | "incident", "what happened", "recent issues", "history", "past issues" |
| `incident_detail` | "latest incident", "last incident", "most recent incident", "incident detail" |
| `uptime` | "uptime", "reliability", "availability", "sla", "percentage" |
| `help` | fallback |

Priority order: `incident_detail` > `incidents` > `uptime` > `components` > `status` > `help`.

Note: `"tell me about"` is intentionally excluded from `incident_detail` keywords because it is too generic (e.g., "tell me about the API" should hit `components`, not `incident_detail`). Use explicit incident keywords only.

### Cache Layer (`fetch.ts`)

**In-memory cache:**
```ts
const memCache = new Map<string, { data: unknown; expiresAt: number }>();
const MEM_TTL_MS = 5 * 60 * 1000; // 5 minutes
```

**D1 cache table** (new migration `migrate-0749.sql`):
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
3. Fetch from `https://status.claude.com/api/v2/<endpoint>.json`.
4. On success: upsert D1 row (`INSERT OR REPLACE`), write memory entry, return data.
5. On fetch failure: if any D1 row exists for that key (whether fresh or expired), return its parsed data as a stale fallback. If no row exists, return `null`.

**Cache keys and endpoints:**

| Cache key | API endpoint | Used by intents |
|---|---|---|
| `claudestatus:summary` | `/api/v2/summary.json` | `status`, `components`, `uptime` |
| `claudestatus:incidents` | `/api/v2/incidents.json` | `incidents`, `incident_detail` |

Note: Both `status` and `components` data come from the same `/api/v2/summary.json` endpoint. There is no separate `/api/v2/components.json` call — components are embedded in the summary response under the `components` array. Only two cache keys are needed.

### API Response Types

From `/api/v2/summary.json`:
```ts
interface SummaryResponse {
  status: { indicator: string; description: string };
  components: Array<{
    id: string; name: string; status: string; updated_at: string;
  }>;
  incidents: unknown[]; // always empty in summary; use incidents endpoint
}
```

From `/api/v2/incidents.json`:
```ts
interface IncidentsResponse {
  incidents: Array<{
    id: string; name: string; status: string; impact: string;
    created_at: string; resolved_at: string | null;
    incident_updates: Array<{
      status: string; body: string; created_at: string;
    }>;
  }>;
}
```

### Response Format (`format.ts`)

All responses use Markdown.

**`formatStatus(summary)`** — Overall indicator emoji + description + per-component status list. Indicator mapping from `status.indicator` API field:
- `"none"` → ✅
- `"minor"` → ⚠
- `"major"` or `"critical"` → 🔴

**`formatComponents(summary)`** — Markdown table: `| Component | Status | Updated |`. Only include top-level components (`group: false` and `group_id: null`). Sub-components (those with a non-null `group_id`) are excluded to keep the response concise.

**`formatIncidents(incidents, limit=5)`** — List of last N incidents: name, impact badge, date, resolved/ongoing.

**`formatIncidentDetail(incidents)`** — Full timeline of the most recent incident: name, then each `incident_update` with its status label and body text.

**`formatUptime(summary)`** — Note explaining the JSON API does not expose uptime percentages (they are rendered client-side on status.claude.com). Shows current component statuses as the best available proxy. Example response:
> "ℹ Uptime percentages are only shown on status.claude.com and aren't available via the API. Current component status: claude.ai ✅ · API ✅ · Claude Code ✅ · platform ✅ · Government ✅"

**`formatHelp()`** — Bot intro + all example questions.

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

**Import:** Add `import { getBotProfile } from "./bots";` at the top of `chatview.ts`. `isBuiltInBot` is not needed here — `getBotProfile(peerActor) !== null` is the sufficient guard.

**Welcome panel:** When the thread is empty (`messages.length === 0`) and the peer is a built-in bot (`peerActor?.startsWith("a/")` and `getBotProfile(peerActor)` returns a non-null value), render a welcome panel as the sole content of `#thread` instead of the "No messages yet" placeholder:

```html
<div class="bot-welcome" id="bot-welcome">
  <div class="bot-welcome-avatar">${botAvatar(peerActor, 48)}</div>
  <div class="bot-welcome-name">${esc(peerActor.slice(2))}</div>
  <div class="bot-welcome-bio">${esc(profile.bio)}</div>
  <div class="bot-welcome-chips">
    ${profile.examples.map(ex =>
      `<button class="chip" onclick="fillInput(this)">${esc(ex)}</button>`
    ).join("")}
  </div>
</div>
```

`botAvatar(peerActor, 48)` is an existing function imported from `./avatar` already imported in `chatview.ts`.

**CSS** (added to the `<style>` block in `chatview.ts`):
```css
.bot-welcome{display:flex;flex-direction:column;align-items:center;
  justify-content:center;flex:1;padding:32px 24px;gap:12px;text-align:center}
.bot-welcome-name{font-family:'JetBrains Mono',monospace;font-size:13px;
  font-weight:600;color:var(--text)}
.bot-welcome-bio{font-size:14px;color:var(--text-2);max-width:360px;line-height:1.6}
.bot-welcome-chips{display:flex;flex-wrap:wrap;gap:8px;justify-content:center;margin-top:4px}
.chip{font-family:'JetBrains Mono',monospace;font-size:12px;padding:6px 14px;
  border:1px solid var(--border);background:none;color:var(--text-2);
  cursor:pointer;transition:all .15s}
.chip:hover{border-color:var(--text-3);color:var(--text)}
```

**Hiding the panel:** When `messages.length > 0` (SSR path), the welcome panel is not rendered. In the JS path (first new message appended via `appendMsg`), add to `appendMsg`:
```js
const welcome = document.getElementById('bot-welcome');
if (welcome) welcome.remove();
```

**`fillInput` function** (added to the `<script>` block):
```js
function fillInput(btn) {
  const input = document.getElementById('msg-input');
  input.value = btn.textContent;
  input.focus();
  input.style.height = 'auto';
  input.style.height = Math.min(input.scrollHeight, 120) + 'px';
}
```

### Landing Page (`src/landing.ts`)

Add a `"Talk to ClaudeStatus 📡"` fieldset after the Scout section, matching the same `fieldset.s` + `.convo` pattern:

```html
<fieldset class="s">
  <legend>Talk to ClaudeStatus 📡</legend>
  <div class="prose">
    <p>ClaudeStatus is a built-in bot that monitors Anthropic's services. Ask it about current status, recent incidents, or component health — it fetches live data from status.claude.com.</p>
    <div class="convo">
      <div class="convo-line"><div class="convo-who">you</div><div class="convo-text">is claude down?</div></div>
      <div class="convo-line"><div class="convo-who">claudestatus</div><div class="convo-text">✅ All Systems Operational — claude.ai, API, Claude Code, platform all green.</div></div>
      <div class="convo-line"><div class="convo-who">you</div><div class="convo-text">any recent incidents?</div></div>
      <div class="convo-line"><div class="convo-who">claudestatus</div><div class="convo-text">⚠ 2 incidents today — "Elevated errors on Claude Sonnet 4.6" (minor, resolved 15:45 UTC).</div></div>
    </div>
    <p>Message <strong>a/claudestatus</strong> directly, or tag <strong>@claudestatus</strong> in any room.</p>
  </div>
</fieldset>
```

---

## D1 Migration

New file: `migrate-0749.sql`

```sql
CREATE TABLE IF NOT EXISTS bot_cache (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bot_cache_expires ON bot_cache(expires_at);
```

Applied to the D1 database before deploying. Existing migration files (`migrate-0746.sql`, `migrate-0746b.sql`, `migrate-magic.sql`) remain untouched.

---

## Error Handling

| Scenario | Behaviour |
|---|---|
| API fetch fails, D1 has fresh data | Serve from D1, no error shown |
| API fetch fails, D1 has stale data | Serve stale data + prefix: "⚠ Data may be stale." |
| API fetch fails, no D1 data | Return: "⚠ Could not reach status.claude.com. Try again in a moment." |
| API returns non-200 | Same as fetch failure |
| D1 write fails | Log silently (`console.error`), serve data anyway (memory cache still populated) |

---

## Testing

- `src/bot/claudestatus/intent.test.ts` — unit tests for `detectIntent()` covering all 6 intents + edge cases (e.g., "tell me about the API" hits `components` not `incident_detail`).
- `src/bot/claudestatus/format.test.ts` — unit tests for each format function with fixture JSON responses.
- `src/bot/registry.test.ts` — verify `registerBot`, `isBuiltInBot`, `getBotProfile`, `listBotActors`, `dispatchReply` contract; verify that registering a duplicate actor name throws an error.
- Echo and Chinese: no new tests needed (logic unchanged, just moved to new files).
- Scout: existing tests in `src/bot/scout/intent.test.ts` and `src/bot/scout/format.test.ts` remain valid and unchanged.

---

## File Changelist

| Action | File |
|---|---|
| New | `src/bot/registry.ts` |
| New | `src/bot/echo/index.ts` |
| New | `src/bot/chinese/index.ts` |
| Modified | `src/bot/scout/index.ts` (add registerBot wrapper, preserve scoutReply export) |
| New | `src/bot/claudestatus/index.ts` |
| New | `src/bot/claudestatus/intent.ts` |
| New | `src/bot/claudestatus/format.ts` |
| New | `src/bot/claudestatus/fetch.ts` |
| New | `src/bot/claudestatus/intent.test.ts` |
| New | `src/bot/claudestatus/format.test.ts` |
| New | `src/bot/registry.test.ts` |
| Modified | `src/bots.ts` (refactored to thin wrapper) |
| Modified | `src/message.ts` (room @mention dispatch uses registry loop) |
| Modified | `src/chatview.ts` (bot welcome panel + fillInput + CSS) |
| Modified | `src/landing.ts` (ClaudeStatus section) |
| New | `migrate-0749.sql` |
