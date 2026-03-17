# Bot Registry System + ClaudeStatus Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the hardcoded bot dispatch system with a self-registering registry, move echo/chinese into dedicated modules, and add a live-data ClaudeStatus bot with two-layer caching.

**Architecture:** A module-level `Map` in `src/bot/registry.ts` serves as the single source of truth. Each bot self-registers via a top-level `registerBot()` call at import time (side-effect import). `bots.ts` becomes a thin wrapper that imports all bots and re-exports registry helpers. `message.ts` room @mention dispatch iterates `listBotActors()` instead of hardcoded regexes.

**Tech Stack:** TypeScript, Cloudflare Workers (Wrangler), Hono, D1 SQLite, Vitest (test runner: `npm run test:bot` from the `chat/` directory)

**Spec:** `spec/0749_bot_system.md`

**Working directory for all commands:** `blueprints/now/tools/worker/chat/`

---

## File Map

| Action | File | Responsibility |
|---|---|---|
| New | `src/bot/registry.ts` | Bot registration store + dispatch |
| New | `src/bot/registry.test.ts` | Registry contract tests |
| New | `src/bot/echo/index.ts` | Echo bot (extracted from bots.ts) |
| New | `src/bot/chinese/index.ts` | Chinese translation bot (extracted from bots.ts) |
| Modify | `src/bot/scout/index.ts` | Add registerBot call at bottom |
| New | `src/bot/claudestatus/intent.ts` | Intent detection (keyword matching) |
| New | `src/bot/claudestatus/intent.test.ts` | Intent tests |
| New | `src/bot/claudestatus/format.ts` | Response formatters (Markdown) |
| New | `src/bot/claudestatus/format.test.ts` | Format tests |
| New | `src/bot/claudestatus/fetch.ts` | Two-layer cache + API fetch |
| New | `src/bot/claudestatus/index.ts` | registerBot + claudestatusReply |
| Modify | `src/bots.ts` | Thin wrapper: side-effect imports + re-exports |
| Modify | `src/message.ts` | Room @mention: registry loop replaces hardcoded regexes |
| Modify | `src/chatview.ts` | Bot welcome panel (bio + chips) when thread is empty |
| Modify | `src/landing.ts` | Add ClaudeStatus fieldset section |
| New | `migrate-0749.sql` | `bot_cache` D1 table |

---

## Task 1: Registry Core

**Files:**
- Create: `src/bot/registry.ts`
- Create: `src/bot/registry.test.ts`

- [ ] **Step 1: Write the failing registry tests**

Create `src/bot/registry.test.ts`:

```ts
import { describe, it, expect, beforeEach } from "vitest";
// Each vitest test file gets its own module scope, so cross-file state is isolated.
// Within this file, `registry` is a module singleton — we call _resetForTesting()
// in beforeEach so every test starts with an empty registry.
import {
  registerBot, isBuiltInBot, getBotProfile,
  listBotActors, dispatchReply, _resetForTesting,
} from "./registry";

beforeEach(() => {
  _resetForTesting();
});

const testProfile = {
  bio: "A test bot.",
  examples: ["hello", "world"],
};

describe("registerBot / isBuiltInBot", () => {
  it("registers a bot and recognises it", () => {
    registerBot({ actor: "a/test1", profile: testProfile, reply: () => "hi" });
    expect(isBuiltInBot("a/test1")).toBe(true);
  });

  it("returns false for unknown actors", () => {
    expect(isBuiltInBot("a/nobody")).toBe(false);
  });

  it("throws when registering a duplicate actor", () => {
    registerBot({ actor: "a/dup", profile: testProfile, reply: () => "x" });
    expect(() =>
      registerBot({ actor: "a/dup", profile: testProfile, reply: () => "y" })
    ).toThrow("Bot already registered: a/dup");
  });
});

describe("getBotProfile", () => {
  it("returns the profile for a registered bot", () => {
    registerBot({ actor: "a/profiled", profile: testProfile, reply: () => "hi" });
    expect(getBotProfile("a/profiled")).toEqual(testProfile);
  });

  it("returns null for unknown actors", () => {
    expect(getBotProfile("a/ghost")).toBeNull();
  });
});

describe("listBotActors", () => {
  it("includes all registered actors", () => {
    registerBot({ actor: "a/botA", profile: testProfile, reply: () => "a" });
    registerBot({ actor: "a/botB", profile: testProfile, reply: () => "b" });
    const list = listBotActors();
    expect(list).toContain("a/botA");
    expect(list).toContain("a/botB");
  });

  it("returns empty array when no bots registered", () => {
    expect(listBotActors()).toEqual([]);
  });
});

describe("dispatchReply", () => {
  it("calls the bot reply and returns the string", async () => {
    registerBot({ actor: "a/pong", profile: testProfile, reply: () => "pong" });
    const result = await dispatchReply("a/pong", "ping", {} as D1Database);
    expect(result).toBe("pong");
  });

  it("returns null for unregistered actors", async () => {
    const result = await dispatchReply("a/unknown", "hi", {} as D1Database);
    expect(result).toBeNull();
  });

  it("awaits async reply functions", async () => {
    registerBot({
      actor: "a/async",
      profile: testProfile,
      reply: async (msg) => `async:${msg}`,
    });
    const result = await dispatchReply("a/async", "hello", {} as D1Database);
    expect(result).toBe("async:hello");
  });
});
```

- [ ] **Step 2: Run tests — expect FAIL (registry.ts doesn't exist)**

```bash
npm run test:bot 2>&1 | head -30
```

Expected: error like `Cannot find module './registry'`

- [ ] **Step 3: Implement `src/bot/registry.ts`**

```ts
export interface BotProfile {
  bio: string;
  examples: string[];
}

export interface BotDef {
  actor: string;
  profile: BotProfile;
  reply: (msg: string, db: D1Database) => Promise<string> | string;
}

const registry = new Map<string, BotDef>();

export function registerBot(def: BotDef): void {
  if (registry.has(def.actor)) {
    throw new Error(`Bot already registered: ${def.actor}`);
  }
  registry.set(def.actor, def);
}

/** Only for use in tests — clears all registered bots. */
export function _resetForTesting(): void {
  registry.clear();
}

export function isBuiltInBot(actor: string): boolean {
  return registry.has(actor);
}

export function getBotProfile(actor: string): BotProfile | null {
  return registry.get(actor)?.profile ?? null;
}

export function listBotActors(): string[] {
  return Array.from(registry.keys());
}

export async function dispatchReply(
  actor: string,
  msg: string,
  db: D1Database
): Promise<string | null> {
  const bot = registry.get(actor);
  if (!bot) return null;
  return await bot.reply(msg, db);
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
npm run test:bot 2>&1 | tail -20
```

Expected: all registry tests green

- [ ] **Step 5: Commit**

```bash
git add src/bot/registry.ts src/bot/registry.test.ts
git commit -m "feat(chat): add bot registry with registerBot, dispatch, profile"
```

---

## Task 2: Move Echo + Chinese into Bot Modules

**Files:**
- Create: `src/bot/echo/index.ts`
- Create: `src/bot/chinese/index.ts`

Note: No new tests — logic is unchanged, just relocated.

- [ ] **Step 1: Create `src/bot/echo/index.ts`**

```ts
import { registerBot } from "../registry";

registerBot({
  actor: "a/echo",
  profile: {
    bio: "Echo repeats your message back verbatim. Useful for testing the chat pipeline.",
    examples: ["hello world", "test message", "ping"],
  },
  reply: (msg) => `Echo: ${msg}`,
});
```

- [ ] **Step 2: Create `src/bot/chinese/index.ts`**

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
        const data = (await res.json()) as {
          responseData?: { translatedText?: string };
        };
        if (data?.responseData?.translatedText) {
          translated = data.responseData.translatedText;
        }
      }
    } catch {
      /* keep original if translation fails */
    }
    return `回声：${translated}`;
  },
});
```

- [ ] **Step 3: Commit**

```bash
git add src/bot/echo/index.ts src/bot/chinese/index.ts
git commit -m "feat(chat): extract echo and chinese bots into dedicated modules"
```

---

## Task 3: Add Registration to Scout

**Files:**
- Modify: `src/bot/scout/index.ts`

The existing file exports `scoutReply`. We append a `registerBot` call at the bottom. The `scoutReply` function is already in scope — no circular import needed.

- [ ] **Step 1: Read the current `src/bot/scout/index.ts`** to know what's there

- [ ] **Step 2: Edit `src/bot/scout/index.ts` — add import at top, registerBot call at bottom**

At the **top** of the file, add the import on a new line after existing imports:

```ts
import { registerBot } from "../registry";
```

At the **bottom** of the file (after the `scoutReply` function export), add:

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
  // db is part of BotDef interface but scout is synchronous and needs no DB access
});
```

- [ ] **Step 3: Run existing scout tests — expect still PASS**

```bash
npm run test:bot 2>&1 | tail -20
```

Expected: all scout tests green (intent + format)

- [ ] **Step 4: Commit**

```bash
git add src/bot/scout/index.ts
git commit -m "feat(chat): register scout bot in registry"
```

---

## Task 4: D1 Migration File

**Files:**
- Create: `migrate-0749.sql`

- [ ] **Step 1: Create `migrate-0749.sql`**

```sql
CREATE TABLE IF NOT EXISTS bot_cache (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bot_cache_expires ON bot_cache(expires_at);
```

- [ ] **Step 2: Commit**

```bash
git add migrate-0749.sql
git commit -m "feat(chat): add bot_cache table migration for claudestatus caching"
```

---

## Task 5: ClaudeStatus Intent Detection

**Files:**
- Create: `src/bot/claudestatus/intent.ts`
- Create: `src/bot/claudestatus/intent.test.ts`

- [ ] **Step 1: Write failing intent tests**

Create `src/bot/claudestatus/intent.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { detectIntent } from "./intent";

describe("detectIntent — status", () => {
  it("detects 'is claude down'", () => {
    expect(detectIntent("is claude down?").intent).toBe("status");
  });
  it("detects 'is it up'", () => {
    expect(detectIntent("is it up?").intent).toBe("status");
  });
  it("detects 'outage'", () => {
    expect(detectIntent("any outage today?").intent).toBe("status");
  });
  it("detects 'operational'", () => {
    expect(detectIntent("is everything operational?").intent).toBe("status");
  });
  it("detects 'working'", () => {
    expect(detectIntent("is claude working").intent).toBe("status");
  });
});

describe("detectIntent — components", () => {
  it("detects 'api'", () => {
    expect(detectIntent("is the api up?").intent).toBe("components");
  });
  it("detects 'claude code'", () => {
    expect(detectIntent("how is claude code doing?").intent).toBe("components");
  });
  it("detects 'platform'", () => {
    expect(detectIntent("check platform status").intent).toBe("components");
  });
  it("detects 'component'", () => {
    expect(detectIntent("show me component status").intent).toBe("components");
  });
});

describe("detectIntent — incidents", () => {
  it("detects 'incident'", () => {
    expect(detectIntent("any incidents recently?").intent).toBe("incidents");
  });
  it("detects 'what happened'", () => {
    expect(detectIntent("what happened today?").intent).toBe("incidents");
  });
  it("detects 'past issues'", () => {
    expect(detectIntent("show me past issues").intent).toBe("incidents");
  });
});

describe("detectIntent — incident_detail", () => {
  it("detects 'latest incident'", () => {
    expect(detectIntent("latest incident details").intent).toBe("incident_detail");
  });
  it("detects 'last incident'", () => {
    expect(detectIntent("what was the last incident?").intent).toBe("incident_detail");
  });
  it("detects 'incident detail'", () => {
    expect(detectIntent("incident detail").intent).toBe("incident_detail");
  });
  it("'tell me about the api' does NOT hit incident_detail", () => {
    expect(detectIntent("tell me about the api").intent).toBe("components");
  });
});

describe("detectIntent — uptime", () => {
  it("detects 'uptime'", () => {
    expect(detectIntent("what is the uptime?").intent).toBe("uptime");
  });
  it("detects 'availability'", () => {
    expect(detectIntent("check availability").intent).toBe("uptime");
  });
  it("detects 'sla'", () => {
    expect(detectIntent("what is your sla?").intent).toBe("uptime");
  });
  it("detects 'reliability'", () => {
    expect(detectIntent("how reliable is claude?").intent).toBe("uptime");
  });
});

describe("detectIntent — help fallback", () => {
  it("returns help for unknown messages", () => {
    expect(detectIntent("hello there").intent).toBe("help");
  });
  it("returns help for empty string", () => {
    expect(detectIntent("").intent).toBe("help");
  });
});

describe("priority order", () => {
  it("incident_detail beats incidents", () => {
    expect(detectIntent("show latest incident history").intent).toBe("incident_detail");
  });
  it("incidents beats uptime", () => {
    expect(detectIntent("past issues availability").intent).toBe("incidents");
  });
});
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
npm run test:bot 2>&1 | grep -E "FAIL|Cannot find"
```

Expected: `Cannot find module './intent'`

- [ ] **Step 3: Implement `src/bot/claudestatus/intent.ts`**

```ts
export type Intent =
  | "status"
  | "components"
  | "incidents"
  | "incident_detail"
  | "uptime"
  | "help";

export interface DetectedIntent {
  intent: Intent;
}

const INCIDENT_DETAIL_KEYWORDS = [
  "latest incident", "last incident", "most recent incident", "incident detail",
];
const INCIDENT_KEYWORDS = [
  "incident", "what happened", "recent issues", "history", "past issues",
];
const UPTIME_KEYWORDS = [
  "uptime", "reliability", "availability", "sla", "percentage",
];
const COMPONENT_KEYWORDS = [
  "api", "claude code", "platform", "claude.ai", "government", "component",
];
const STATUS_KEYWORDS = [
  "status", "down", "outage", "operational", "is claude", "is it up", "working",
];

function matches(msg: string, keywords: string[]): boolean {
  return keywords.some((k) => msg.includes(k));
}

export function detectIntent(message: string): DetectedIntent {
  const msg = ` ${message.toLowerCase()} `;

  if (matches(msg, INCIDENT_DETAIL_KEYWORDS)) return { intent: "incident_detail" };
  if (matches(msg, INCIDENT_KEYWORDS))        return { intent: "incidents" };
  if (matches(msg, UPTIME_KEYWORDS))          return { intent: "uptime" };
  if (matches(msg, COMPONENT_KEYWORDS))       return { intent: "components" };
  if (matches(msg, STATUS_KEYWORDS))          return { intent: "status" };
  return { intent: "help" };
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
npm run test:bot 2>&1 | tail -20
```

Expected: all claudestatus intent tests green

- [ ] **Step 5: Commit**

```bash
git add src/bot/claudestatus/intent.ts src/bot/claudestatus/intent.test.ts
git commit -m "feat(chat): add claudestatus intent detection"
```

---

## Task 6: ClaudeStatus Response Formatters

**Files:**
- Create: `src/bot/claudestatus/format.ts`
- Create: `src/bot/claudestatus/format.test.ts`

**API type reference** (used in both files):

```ts
// SummaryResponse from /api/v2/summary.json
interface StatusObj { indicator: string; description: string; }
interface ComponentObj { id: string; name: string; status: string; updated_at: string; group_id: string | null; group: boolean; }
interface SummaryResponse { status: StatusObj; components: ComponentObj[]; }

// IncidentUpdate + Incident from /api/v2/incidents.json
interface IncidentUpdate { status: string; body: string; created_at: string; }
interface Incident {
  id: string; name: string; status: string; impact: string;
  created_at: string; resolved_at: string | null;
  incident_updates: IncidentUpdate[];
}
interface IncidentsResponse { incidents: Incident[]; }
```

- [ ] **Step 1: Write failing format tests**

Create `src/bot/claudestatus/format.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import {
  formatStatus, formatComponents, formatIncidents,
  formatIncidentDetail, formatUptime, formatHelp,
} from "./format";

const okSummary = {
  status: { indicator: "none", description: "All Systems Operational" },
  components: [
    { id: "1", name: "claude.ai",            status: "operational", updated_at: "2026-03-17T15:00:00Z", group_id: null, group: false },
    { id: "2", name: "Claude API",            status: "operational", updated_at: "2026-03-17T15:00:00Z", group_id: null, group: false },
    { id: "3", name: "Claude Code",           status: "operational", updated_at: "2026-03-17T15:00:00Z", group_id: null, group: false },
    { id: "4", name: "Claude API sub",        status: "operational", updated_at: "2026-03-17T15:00:00Z", group_id: "2", group: false },
  ],
};

const degradedSummary = {
  status: { indicator: "minor", description: "Partial System Outage" },
  components: [
    { id: "1", name: "claude.ai",  status: "degraded_performance", updated_at: "2026-03-17T15:00:00Z", group_id: null, group: false },
    { id: "2", name: "Claude API", status: "operational",          updated_at: "2026-03-17T15:00:00Z", group_id: null, group: false },
  ],
};

const incidents = [
  {
    id: "abc1", name: "Elevated errors on Claude Sonnet 4.6",
    status: "resolved", impact: "minor",
    created_at: "2026-03-17T14:07:53Z", resolved_at: "2026-03-17T15:45:20Z",
    incident_updates: [
      { status: "investigating", body: "We are looking into this.", created_at: "2026-03-17T14:07:53Z" },
      { status: "resolved",      body: "Issue has been resolved.",  created_at: "2026-03-17T15:45:20Z" },
    ],
  },
];

describe("formatStatus", () => {
  it("shows ✅ for indicator=none", () => {
    expect(formatStatus(okSummary)).toContain("✅");
  });
  it("shows ⚠ for indicator=minor", () => {
    expect(formatStatus(degradedSummary)).toContain("⚠");
  });
  it("includes description", () => {
    expect(formatStatus(okSummary)).toContain("All Systems Operational");
  });
  it("lists top-level component names", () => {
    const out = formatStatus(okSummary);
    expect(out).toContain("claude.ai");
    expect(out).toContain("Claude API");
  });
  it("does NOT include sub-components (group_id != null)", () => {
    expect(formatStatus(okSummary)).not.toContain("Claude API sub");
  });
});

describe("formatComponents", () => {
  it("renders a markdown table", () => {
    const out = formatComponents(okSummary);
    expect(out).toContain("| Component |");
    expect(out).toContain("| Status |");
  });
  it("excludes sub-components", () => {
    expect(formatComponents(okSummary)).not.toContain("Claude API sub");
  });
  it("includes top-level components", () => {
    expect(formatComponents(okSummary)).toContain("claude.ai");
  });
});

describe("formatIncidents", () => {
  it("includes incident name", () => {
    expect(formatIncidents(incidents)).toContain("Elevated errors on Claude Sonnet 4.6");
  });
  it("shows impact", () => {
    expect(formatIncidents(incidents)).toContain("minor");
  });
  it("shows resolved status", () => {
    expect(formatIncidents(incidents)).toContain("resolved");
  });
  it("shows 'No recent incidents' when list is empty", () => {
    expect(formatIncidents([])).toContain("No recent incidents");
  });
});

describe("formatIncidentDetail", () => {
  it("shows incident name as heading", () => {
    expect(formatIncidentDetail(incidents)).toContain("Elevated errors on Claude Sonnet 4.6");
  });
  it("shows each update body", () => {
    const out = formatIncidentDetail(incidents);
    expect(out).toContain("We are looking into this.");
    expect(out).toContain("Issue has been resolved.");
  });
  it("shows update statuses", () => {
    const out = formatIncidentDetail(incidents);
    expect(out).toContain("investigating");
    expect(out).toContain("resolved");
  });
  it("handles empty incident list gracefully", () => {
    expect(formatIncidentDetail([])).toContain("No incidents");
  });
});

describe("formatUptime", () => {
  it("explains API limitation", () => {
    expect(formatUptime(okSummary)).toContain("status.claude.com");
  });
  it("shows component statuses", () => {
    expect(formatUptime(okSummary)).toContain("claude.ai");
  });
  it("uses ✅ for operational", () => {
    expect(formatUptime(okSummary)).toContain("✅");
  });
});

describe("formatHelp", () => {
  it("mentions claudestatus", () => {
    expect(formatHelp()).toContain("ClaudeStatus");
  });
  it("shows example questions", () => {
    const out = formatHelp();
    expect(out).toContain("Is Claude down");
    expect(out).toContain("incident");
  });
});
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
npm run test:bot 2>&1 | grep "FAIL\|Cannot find"
```

Expected: `Cannot find module './format'`

- [ ] **Step 3: Implement `src/bot/claudestatus/format.ts`**

```ts
// --- Shared types ---

interface StatusObj { indicator: string; description: string; }
interface ComponentObj {
  id: string; name: string; status: string;
  updated_at: string; group_id: string | null; group: boolean;
}
export interface SummaryResponse { status: StatusObj; components: ComponentObj[]; }

interface IncidentUpdate { status: string; body: string; created_at: string; }
export interface Incident {
  id: string; name: string; status: string; impact: string;
  created_at: string; resolved_at: string | null;
  incident_updates: IncidentUpdate[];
}
export interface IncidentsResponse { incidents: Incident[]; }

// --- Helpers ---

function indicatorEmoji(indicator: string): string {
  if (indicator === "none") return "✅";
  if (indicator === "minor") return "⚠";
  if (indicator === "major" || indicator === "critical") return "🔴";
  return "⚠";
}

function componentEmoji(status: string): string {
  return status === "operational" ? "✅" : "⚠";
}

function topLevel(components: ComponentObj[]): ComponentObj[] {
  return components.filter((c) => c.group_id === null && !c.group);
}

function fmtDate(iso: string): string {
  return new Date(iso).toLocaleString("en-US", {
    month: "short", day: "numeric", hour: "2-digit", minute: "2-digit",
    timeZone: "UTC", hour12: false,
  }) + " UTC";
}

// --- Formatters ---

export function formatStatus(summary: SummaryResponse): string {
  const emoji = indicatorEmoji(summary.status.indicator);
  const lines = [
    `## ${emoji} ${summary.status.description}`,
    ``,
    ...topLevel(summary.components).map(
      (c) => `- ${componentEmoji(c.status)} **${c.name}** — ${c.status.replace(/_/g, " ")}`
    ),
  ];
  return lines.join("\n");
}

export function formatComponents(summary: SummaryResponse): string {
  const rows = topLevel(summary.components).map(
    (c) => `| ${c.name} | ${componentEmoji(c.status)} ${c.status.replace(/_/g, " ")} | ${fmtDate(c.updated_at)} |`
  );
  return [
    `## 📋 Component Status`,
    ``,
    `| Component | Status | Updated |`,
    `|---|---|---|`,
    ...rows,
  ].join("\n");
}

export function formatIncidents(incidents: Incident[], limit = 5): string {
  if (incidents.length === 0) {
    return `> ✅ No recent incidents found.`;
  }
  const lines = [`## ⚠ Recent Incidents\n`];
  for (const inc of incidents.slice(0, limit)) {
    const date = fmtDate(inc.created_at);
    const resolved = inc.resolved_at ? `resolved ${fmtDate(inc.resolved_at)}` : "ongoing";
    lines.push(`- **${inc.name}** — ${inc.impact} · ${date} · ${resolved}`);
  }
  return lines.join("\n");
}

export function formatIncidentDetail(incidents: Incident[]): string {
  if (incidents.length === 0) {
    return `> ✅ No incidents on record.`;
  }
  const inc = incidents[0];
  const lines = [
    `## 🔍 ${inc.name}`,
    ``,
    `**Impact:** ${inc.impact} · **Status:** ${inc.status}`,
    `**Started:** ${fmtDate(inc.created_at)}`,
    inc.resolved_at ? `**Resolved:** ${fmtDate(inc.resolved_at)}` : `**Status:** ongoing`,
    ``,
    `### Timeline`,
  ];
  for (const update of [...inc.incident_updates].reverse()) {
    lines.push(`- **${update.status}** (${fmtDate(update.created_at)}): ${update.body}`);
  }
  return lines.join("\n");
}

export function formatUptime(summary: SummaryResponse): string {
  const componentList = topLevel(summary.components)
    .map((c) => `${c.name} ${componentEmoji(c.status)}`)
    .join(" · ");
  return [
    `## ℹ Uptime`,
    ``,
    `Uptime percentages are only shown on [status.claude.com](https://status.claude.com) and aren't available via the JSON API.`,
    ``,
    `**Current component status:** ${componentList}`,
  ].join("\n");
}

export function formatHelp(): string {
  return [
    `## 📡 ClaudeStatus — your Anthropic service monitor`,
    ``,
    `Ask me anything about Claude's service health:`,
    ``,
    `- **"Is Claude down?"** — overall status`,
    `- **"Is the API up?"** — per-component status`,
    `- **"Any recent incidents?"** — incident list`,
    `- **"Latest incident details"** — full incident timeline`,
    `- **"What's the uptime?"** — availability info`,
  ].join("\n");
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
npm run test:bot 2>&1 | tail -20
```

Expected: all claudestatus format tests green

- [ ] **Step 5: Commit**

```bash
git add src/bot/claudestatus/format.ts src/bot/claudestatus/format.test.ts
git commit -m "feat(chat): add claudestatus response formatters"
```

---

## Task 7: ClaudeStatus Cache + Fetch Layer

**Files:**
- Create: `src/bot/claudestatus/fetch.ts`

No unit tests for this layer — it depends on D1 and `globalThis.fetch`, both unavailable in the plain Vitest node environment. Correctness is verified at integration time.

- [ ] **Step 1: Implement `src/bot/claudestatus/fetch.ts`**

```ts
import type { SummaryResponse, IncidentsResponse } from "./format";

const MEM_TTL_MS = 5 * 60 * 1000;   // 5 minutes
const D1_TTL_MS  = 15 * 60 * 1000;  // 15 minutes

interface CacheEntry { data: unknown; expiresAt: number; }
const memCache = new Map<string, CacheEntry>();

const BASE = "https://status.claude.com/api/v2";

async function fetchWithCache<T>(
  key: string,
  url: string,
  db: D1Database
): Promise<{ data: T; stale: boolean } | null> {
  const now = Date.now();

  // 1. Memory cache
  const mem = memCache.get(key);
  if (mem && mem.expiresAt > now) {
    return { data: mem.data as T, stale: false };
  }

  // 2. D1 cache
  const row = await db
    .prepare("SELECT value, expires_at FROM bot_cache WHERE key = ?")
    .bind(key)
    .first<{ value: string; expires_at: number }>();

  if (row && row.expires_at > now) {
    const parsed = JSON.parse(row.value) as T;
    memCache.set(key, { data: parsed, expiresAt: row.expires_at });
    return { data: parsed, stale: false };
  }

  // 3. Live fetch
  try {
    const res = await fetch(url, { headers: { "User-Agent": "chat.now/1.0" } });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = (await res.json()) as T;

    const expiresAt = now + D1_TTL_MS;
    // Write D1 (upsert)
    try {
      await db
        .prepare("INSERT OR REPLACE INTO bot_cache (key, value, expires_at) VALUES (?, ?, ?)")
        .bind(key, JSON.stringify(data), expiresAt)
        .run();
    } catch (e) {
      console.error("[claudestatus] D1 write failed:", e);
    }
    // Write memory
    memCache.set(key, { data, expiresAt: now + MEM_TTL_MS });

    return { data, stale: false };
  } catch {
    // 4. Stale fallback
    if (row) {
      const parsed = JSON.parse(row.value) as T;
      return { data: parsed, stale: true };
    }
    return null;
  }
}

export async function fetchSummary(
  db: D1Database
): Promise<{ data: SummaryResponse; stale: boolean } | null> {
  return fetchWithCache<SummaryResponse>(
    "claudestatus:summary",
    `${BASE}/summary.json`,
    db
  );
}

export async function fetchIncidents(
  db: D1Database
): Promise<{ data: IncidentsResponse; stale: boolean } | null> {
  return fetchWithCache<IncidentsResponse>(
    "claudestatus:incidents",
    `${BASE}/incidents.json`,
    db
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add src/bot/claudestatus/fetch.ts
git commit -m "feat(chat): add claudestatus two-layer cache (memory + D1)"
```

---

## Task 8: ClaudeStatus Bot Index + Wire into bots.ts

**Files:**
- Create: `src/bot/claudestatus/index.ts`
- Modify: `src/bots.ts`

- [ ] **Step 1: Create `src/bot/claudestatus/index.ts`**

```ts
import { registerBot } from "../registry";
import { detectIntent } from "./intent";
import {
  formatStatus, formatComponents, formatIncidents,
  formatIncidentDetail, formatUptime, formatHelp,
} from "./format";
import { fetchSummary, fetchIncidents } from "./fetch";

async function claudestatusReply(msg: string, db: D1Database): Promise<string> {
  const { intent } = detectIntent(msg);

  switch (intent) {
    case "status":
    case "components":
    case "uptime": {
      const result = await fetchSummary(db);
      if (!result) return "⚠ Could not reach status.claude.com. Try again in a moment.";
      const prefix = result.stale ? "⚠ Data may be stale.\n\n" : "";
      if (intent === "status")     return prefix + formatStatus(result.data);
      if (intent === "components") return prefix + formatComponents(result.data);
      return prefix + formatUptime(result.data);
    }

    case "incidents":
    case "incident_detail": {
      const result = await fetchIncidents(db);
      if (!result) return "⚠ Could not reach status.claude.com. Try again in a moment.";
      const prefix = result.stale ? "⚠ Data may be stale.\n\n" : "";
      if (intent === "incident_detail") return prefix + formatIncidentDetail(result.data.incidents);
      return prefix + formatIncidents(result.data.incidents);
    }

    default:
      return formatHelp();
  }
}

registerBot({
  actor: "a/claudestatus",
  profile: {
    bio: "ClaudeStatus monitors Anthropic's services in real time. Ask about current status, recent incidents, component health, or uptime.",
    examples: [
      "Is Claude down?",
      "Any recent incidents?",
      "Latest incident details",
      "Is the API up?",
      "What's the uptime?",
    ],
  },
  reply: (msg, db) => claudestatusReply(msg, db),
});
```

- [ ] **Step 2: Refactor `src/bots.ts`**

Replace the entire file with:

```ts
import { messageId } from "./id";
import { isBuiltInBot, getBotProfile, listBotActors, dispatchReply } from "./bot/registry";

// Side-effect imports: each module calls registerBot() at load time
import "./bot/echo";
import "./bot/chinese";
import "./bot/scout";
import "./bot/claudestatus";

export { isBuiltInBot, getBotProfile, listBotActors, dispatchReply };

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
    .prepare(
      "INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES (?, ?, ?, ?, ?, ?)"
    )
    .bind(id, chatId, botActor, replyText, null, now)
    .run();
}
```

- [ ] **Step 3: Run all bot tests — expect PASS**

```bash
npm run test:bot 2>&1 | tail -20
```

Expected: all tests green (registry + scout intent/format + claudestatus intent/format)

- [ ] **Step 4: Commit**

```bash
git add src/bot/claudestatus/index.ts src/bots.ts
git commit -m "feat(chat): wire claudestatus bot + refactor bots.ts to thin registry wrapper"
```

---

## Task 9: Refactor Room @Mention Dispatch in message.ts

**Files:**
- Modify: `src/message.ts`

- [ ] **Step 1: Read `src/message.ts` lines 191–215** to locate the room @mention block

The block to replace is (approximately lines 202–211):
```ts
} else if (chat.kind === "room") {
  const chineseMention = text.match(/^@chinese\s+([\s\S]+)/i);
  if (chineseMention) {
    c.executionCtx.waitUntil(handleBotReply(c.env.DB, chatIdParam, "a/chinese", chineseMention[1].trim()));
  }
  const scoutMention = text.match(/^@scout\s+([\s\S]+)/i);
  if (scoutMention) {
    c.executionCtx.waitUntil(handleBotReply(c.env.DB, chatIdParam, "a/scout", scoutMention[1].trim()));
  }
}
```

- [ ] **Step 2: Update the import at top of message.ts**

The existing import line `import { isBuiltInBot, handleBotReply } from "./bots";` must also import `listBotActors`:

```ts
import { isBuiltInBot, handleBotReply, listBotActors } from "./bots";
```

- [ ] **Step 3: Replace the room @mention block**

Replace the entire `else if (chat.kind === "room")` block with:

```ts
} else if (chat.kind === "room") {
  // Registry-driven @mention dispatch. Break after first match (one bot per message).
  for (const botActor of listBotActors()) {
    const shortName = botActor.slice(2); // "a/scout" -> "scout"
    const pattern = new RegExp(`^@${shortName}\\s+([\\s\\S]+)`, "i");
    const match = text.match(pattern);
    if (match) {
      c.executionCtx.waitUntil(
        handleBotReply(c.env.DB, chatIdParam, botActor, match[1].trim())
      );
      break;
    }
  }
}
```

**Behavior note:** The old code had two separate if-blocks (no early exit), so a message could theoretically trigger both bots. The new loop uses `break` after the first match — only one bot responds. This is an intentional correctness improvement.

- [ ] **Step 4: Note on testing**

`message.ts` uses the Hono request context and D1, so it is not covered by `npm run test:bot` (which only runs `src/bot/**/*.test.ts`). Correctness is verified at integration time via `npm run dev`. No unit test step needed here.

- [ ] **Step 5: Commit**

```bash
git add src/message.ts
git commit -m "feat(chat): use registry loop for room @mention dispatch in message.ts"
```

---

## Task 10: Chat View — Bot Welcome Panel

**Files:**
- Modify: `src/chatview.ts`

- [ ] **Step 1: Add `getBotProfile` import at the top of `src/chatview.ts`**

Find the existing import block (currently `import type { Context }...`). Add:

```ts
import { getBotProfile } from "./bots";
```

- [ ] **Step 2: Add welcome panel CSS to the `<style>` block**

Locate the `/* SSE status dot */` comment block in the `<style>` section. Add the following CSS immediately before it:

```css
/* Bot welcome panel */
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

- [ ] **Step 3: Replace the empty-thread placeholder with the welcome panel**

Find the existing empty-thread placeholder in the `<div id="thread">` section:
```ts
${msgHtml || `<div style="flex:1;display:flex;align-items:center;justify-content:center;color:var(--text-3);font-size:14px">No messages yet. Say something!</div>`}
```

Replace with:

```ts
${(() => {
  if (msgHtml) return msgHtml;
  const profile = peerActor ? getBotProfile(peerActor) : null;
  if (profile) {
    const chips = profile.examples
      .map(ex => `<button class="chip" onclick="fillInput(this)">${esc(ex)}</button>`)
      .join("");
    return `<div class="bot-welcome" id="bot-welcome">
  <div class="bot-welcome-avatar">${botAvatar(peerActor!, 48)}</div>
  <div class="bot-welcome-name">${esc(peerActor!.slice(2))}</div>
  <div class="bot-welcome-bio">${esc(profile.bio)}</div>
  <div class="bot-welcome-chips">${chips}</div>
</div>`;
  }
  return `<div style="flex:1;display:flex;align-items:center;justify-content:center;color:var(--text-3);font-size:14px">No messages yet. Say something!</div>`;
})()}
```

- [ ] **Step 4: Add `fillInput` and update `appendMsg` in the `<script>` block**

Find the `appendMsg` function. Add this line **at the start of `appendMsg`**, before the `const wasBottom` line:

```js
const welcome = document.getElementById('bot-welcome');
if (welcome) welcome.remove();
```

Then add `fillInput` as a new function anywhere in the script block (e.g. after `appendMsg`):

```js
function fillInput(btn) {
  const input = document.getElementById('msg-input');
  input.value = btn.textContent;
  input.focus();
  input.style.height = 'auto';
  input.style.height = Math.min(input.scrollHeight, 120) + 'px';
}
```

- [ ] **Step 5: Commit**

```bash
git add src/chatview.ts
git commit -m "feat(chat): add bot welcome panel with bio and sample question chips"
```

---

## Task 11: Landing Page — ClaudeStatus Section

**Files:**
- Modify: `src/landing.ts`

- [ ] **Step 1: Locate the Scout section in `src/landing.ts`**

Search for `Talk to Scout ⚽` — this is a `<fieldset class="s">` block (approximately line 341–353 in the signed-in human view). The ClaudeStatus section goes **after** it, before the `"What's next"` fieldset.

- [ ] **Step 2: Insert the ClaudeStatus fieldset**

Add the following HTML immediately after the closing `</fieldset>` of the Scout section:

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

- [ ] **Step 3: Commit**

```bash
git add src/landing.ts
git commit -m "feat(chat): add ClaudeStatus section to landing page"
```

---

## Task 12: Apply D1 Migration

This step applies the new `bot_cache` table to the local development database. Run against remote only when ready to deploy.

- [ ] **Step 1: Apply migration to local D1**

```bash
wrangler d1 execute chat-db --local --file=migrate-0749.sql
```

Expected: `✅ Done` (no errors)

- [ ] **Step 2: Smoke test locally**

```bash
npm run dev
```

Open the chat with `a/claudestatus` and send "is claude down?". Expect a formatted status response.

Note: `migrate-0749.sql` was already committed in Task 4 — no additional commit needed here.

---

## Summary of All Commits (in order)

1. `feat(chat): add bot registry with registerBot, dispatch, profile`
2. `feat(chat): extract echo and chinese bots into dedicated modules`
3. `feat(chat): register scout bot in registry`
4. `feat(chat): add bot_cache table migration for claudestatus caching`
5. `feat(chat): add claudestatus intent detection`
6. `feat(chat): add claudestatus response formatters`
7. `feat(chat): add claudestatus two-layer cache (memory + D1)`
8. `feat(chat): wire claudestatus bot + refactor bots.ts to thin registry wrapper`
9. `feat(chat): use registry loop for room @mention dispatch in message.ts`
10. `feat(chat): add bot welcome panel with bio and sample question chips`
11. `feat(chat): add ClaudeStatus section to landing page`
12. `feat(chat): apply bot_cache migration for claudestatus caching`
