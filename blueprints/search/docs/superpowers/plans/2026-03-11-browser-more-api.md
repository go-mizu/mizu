# Browser Worker: More REST API Endpoints — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add eight new endpoints to `tools/browser` that are 100% API-compatible with the Cloudflare Browser Rendering REST API, with a 4-layer stack: in-memory cache → D1 cache → CF proxy → own fallback.

**Architecture:** Each text endpoint runs: in-memory Map (L1, no TTL) → D1 `page_cache` table (L2) → CF Browser Rendering API proxy (L3) → plain `fetch()` + our HTML parser (L4 fallback). Binary endpoints (screenshot, pdf) skip caching and go straight to CF, returning 503 on rate limit.

**Tech Stack:** Hono 4, Cloudflare Workers, D1 SQLite, TypeScript 5, vitest + @cloudflare/vitest-pool-workers for testing.

**Spec:** `spec/0702_browser_more_api.md`

---

## Chunk 1: Test harness + D1 schema

### Task 1: Add vitest + workers test pool

**Files:**
- Create: `tools/browser/vitest.config.ts`
- Modify: `tools/browser/package.json`

Existing worker has no tests. We need vitest with the Cloudflare Workers pool so tests run in a real Workers runtime (real `crypto.subtle`, `HTMLRewriter`, etc.).

- [ ] **Step 1.1: Install vitest and pool**

```bash
cd tools/browser
npm install --save-dev vitest @cloudflare/vitest-pool-workers
```

- [ ] **Step 1.2: Create vitest config**

Create `tools/browser/vitest.config.ts`:

```typescript
import { defineConfig } from "vitest/config";
import { defineWorkersConfig } from "@cloudflare/vitest-pool-workers/config";

export default defineWorkersConfig({
  test: {
    poolOptions: {
      workers: {
        wrangler: { configPath: "./wrangler.toml" },
      },
    },
  },
});
```

- [ ] **Step 1.3: Add test script to package.json**

In `tools/browser/package.json`, add to `"scripts"`:

```json
"test": "vitest run",
"test:watch": "vitest"
```

- [ ] **Step 1.4: Verify vitest installs and empty run passes**

```bash
cd tools/browser
npx vitest run
```

Expected: `No test files found` or `0 tests passed` — no error.

- [ ] **Step 1.5: Commit**

```bash
git add tools/browser/package.json tools/browser/vitest.config.ts tools/browser/package-lock.json
git commit -m "chore(browser): add vitest + cloudflare workers pool"
```

---

### Task 2: D1 schema — add page_cache table

**Files:**
- Modify: `tools/browser/schema.sql`

- [ ] **Step 2.1: Add page_cache table to schema.sql**

Append to the end of `tools/browser/schema.sql`:

```sql
-- Cache for single-URL rendering endpoints
-- PK (url, endpoint, params_hash) covers parameterized variants
-- params_hash is '' for simple URL-only endpoints, 16-char hex for parameterized ones
CREATE TABLE IF NOT EXISTS page_cache (
  url          TEXT    NOT NULL,
  endpoint     TEXT    NOT NULL,
  params_hash  TEXT    NOT NULL DEFAULT '',
  html         TEXT,
  markdown     TEXT,
  result       TEXT,
  title        TEXT,
  created_at   INTEGER NOT NULL,

  PRIMARY KEY (url, endpoint, params_hash)
);

-- Fast lookup by URL (invalidation, cross-endpoint queries)
CREATE INDEX IF NOT EXISTS idx_page_cache_url ON page_cache(url);

-- TTL sweeping by age
CREATE INDEX IF NOT EXISTS idx_page_cache_created ON page_cache(created_at);
```

- [ ] **Step 2.2: Apply migration locally**

```bash
cd tools/browser
npm run db:migrate
```

Expected: `Executing on local database browser-db... ✓`

- [ ] **Step 2.3: Commit**

```bash
git add tools/browser/schema.sql
git commit -m "feat(browser): add page_cache table to D1 schema"
```

---

### Task 3: Extend types.ts with new request/response interfaces

**Files:**
- Modify: `tools/browser/src/types.ts`

- [ ] **Step 3.1: Add new types to types.ts**

Append to `tools/browser/src/types.ts`:

```typescript
// ── Shared option types ──────────────────────────────────────────────────────

export interface GotoOptions {
  waitUntil?: "domcontentloaded" | "networkidle0" | "networkidle2";
  timeout?: number;
}

export interface Cookie {
  name: string;
  value: string;
  domain?: string;
  path?: string;
  secure?: boolean;
  httpOnly?: boolean;
}

export interface AuthCredentials {
  username: string;
  password: string;
}

export interface Viewport {
  width?: number;
  height?: number;
  deviceScaleFactor?: number;
}

export interface WaitForSelector {
  selector: string;
  timeout?: number;
  visible?: boolean;
}

export interface ScriptTag {
  content: string;
}

export interface StyleTag {
  content?: string;
  url?: string;
}

// Shared fields present on all single-URL endpoint requests
export interface BaseRequest {
  url?: string;
  html?: string;
  gotoOptions?: GotoOptions;
  cookies?: Cookie[];
  authenticate?: AuthCredentials;
  setExtraHTTPHeaders?: Record<string, string>;
  userAgent?: string;
  viewport?: Viewport;
  waitForSelector?: string | WaitForSelector;
  addScriptTag?: ScriptTag[];
  addStyleTag?: StyleTag[];
  setJavaScriptEnabled?: boolean;
  rejectResourceTypes?: string[];
  rejectRequestPattern?: string[];
  allowResourceTypes?: string[];
  allowRequestPattern?: string[];
}

// ── /api/content ─────────────────────────────────────────────────────────────

export type ContentRequest = BaseRequest;
// Response: ApiResponse<string>  (result = full HTML string)

// ── /api/screenshot ───────────────────────────────────────────────────────────

export interface ScreenshotOptions {
  type?: "png" | "jpeg";
  quality?: number;
  fullPage?: boolean;
  omitBackground?: boolean;
  clip?: { x: number; y: number; width: number; height: number };
  captureBeyondViewport?: boolean;
}

export interface ScreenshotRequest extends BaseRequest {
  screenshotOptions?: ScreenshotOptions;
  selector?: string;
}
// Response: binary image/png or image/jpeg

// ── /api/pdf ──────────────────────────────────────────────────────────────────

export interface PdfOptions {
  format?: string;
  landscape?: boolean;
  printBackground?: boolean;
  preferCSSPageSize?: boolean;
  scale?: number;
  displayHeaderFooter?: boolean;
  headerTemplate?: string;
  footerTemplate?: string;
  margin?: { top?: string; bottom?: string; left?: string; right?: string };
  timeout?: number;
}

export interface PdfRequest extends BaseRequest {
  pdfOptions?: PdfOptions;
}
// Response: binary application/pdf

// ── /api/markdown ─────────────────────────────────────────────────────────────

export type MarkdownRequest = BaseRequest;
// Response: ApiResponse<string>  (result = markdown string)

// ── /api/snapshot ─────────────────────────────────────────────────────────────

export interface SnapshotScreenshotOptions {
  fullPage?: boolean;
}

export interface SnapshotRequest extends BaseRequest {
  screenshotOptions?: SnapshotScreenshotOptions;
}

export interface SnapshotResult {
  content: string;
  screenshot: string | null;  // base64 PNG, or null when fallback used
}
// Response: ApiResponse<SnapshotResult>

// ── /api/scrape ───────────────────────────────────────────────────────────────

export interface ScrapeElement {
  selector: string;
}

export interface ScrapeRequest extends BaseRequest {
  elements: ScrapeElement[];
}

export interface ScrapeNodeResult {
  text: string;
  html: string;
  attributes: Array<{ name: string; value: string }>;
  height: number;
  width: number;
  top: number;
  left: number;
}

export interface ScrapeSelectorResult {
  selector: string;
  results: ScrapeNodeResult[];
}
// Response: ApiResponse<ScrapeSelectorResult[]>

// ── /api/json ─────────────────────────────────────────────────────────────────

export interface CustomAiModel {
  model: string;
  authorization: string;
}

export interface JsonRequest extends BaseRequest {
  prompt?: string;
  response_format?: {
    type: "json_schema";
    schema: Record<string, unknown>;
  };
  custom_ai?: CustomAiModel[];
}
// Response: ApiResponse<Record<string, unknown>>

// ── /api/links ────────────────────────────────────────────────────────────────

export interface LinksRequest extends BaseRequest {
  visibleLinksOnly?: boolean;
  excludeExternalLinks?: boolean;
}
// Response: ApiResponse<string[]>

// ── D1 cache row ──────────────────────────────────────────────────────────────

export interface PageCacheRow {
  url: string;
  endpoint: string;
  params_hash: string;
  html: string | null;
  markdown: string | null;
  result: string | null;
  title: string | null;
  created_at: number;
}
```

- [ ] **Step 3.2: Verify TypeScript compiles**

```bash
cd tools/browser
npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3.3: Commit**

```bash
git add tools/browser/src/types.ts
git commit -m "feat(browser): add new endpoint request/response types"
```

---

### Task 4: cache.ts — in-memory L1 + D1 L2 helpers

**Files:**
- Create: `tools/browser/src/cache.ts`
- Create: `tools/browser/src/__tests__/cache.test.ts`

This module owns the two-layer cache. It is pure functions (no Hono context), making it easy to test.

- [ ] **Step 4.1: Write failing tests first**

Create `tools/browser/src/__tests__/cache.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { makeCacheKey, hashParams } from "../cache";

describe("makeCacheKey", () => {
  it("joins url, endpoint, hash with null byte", () => {
    expect(makeCacheKey("https://example.com", "content", "")).toBe(
      "https://example.com\0content\0"
    );
  });

  it("includes params hash when provided", () => {
    expect(makeCacheKey("https://x.com", "links", "abc123")).toBe(
      "https://x.com\0links\0abc123"
    );
  });
});

describe("hashParams", () => {
  it("returns empty string for null/undefined", async () => {
    expect(await hashParams(null)).toBe("");
    expect(await hashParams(undefined)).toBe("");
  });

  it("returns 16-char hex for an object", async () => {
    const h = await hashParams({ a: 1, b: 2 });
    expect(h).toMatch(/^[0-9a-f]{16}$/);
  });

  it("is deterministic (same input → same output)", async () => {
    const a = await hashParams({ x: true, y: false });
    const b = await hashParams({ y: false, x: true });  // different key order
    expect(a).toBe(b);  // keys sorted before hashing
  });

  it("produces different hashes for different inputs", async () => {
    const a = await hashParams({ selector: "h1" });
    const b = await hashParams({ selector: "h2" });
    expect(a).not.toBe(b);
  });
});
```

- [ ] **Step 4.2: Run tests — expect failures**

```bash
cd tools/browser
npx vitest run src/__tests__/cache.test.ts
```

Expected: fail with `Cannot find module '../cache'`.

- [ ] **Step 4.3: Implement cache.ts**

Create `tools/browser/src/cache.ts`:

```typescript
import type { D1Database } from "@cloudflare/workers-types";
import type { PageCacheRow } from "./types";

// ── In-memory L1 cache ────────────────────────────────────────────────────────
// Module-level Map; lives for the isolate lifetime (no TTL).
// Key: makeCacheKey(url, endpoint, paramsHash)

interface MemEntry {
  html: string | null;
  markdown: string | null;
  result: string | null;
  title: string | null;
}

const MEM: Map<string, MemEntry> = new Map();

export function makeCacheKey(url: string, endpoint: string, paramsHash: string): string {
  return `${url}\0${endpoint}\0${paramsHash}`;
}

// ── params hashing ────────────────────────────────────────────────────────────

/**
 * Deterministic 16-char hex hash of an arbitrary object.
 * Keys are sorted before serialisation so {a,b} and {b,a} hash identically.
 * Returns "" for null/undefined (signals "no params").
 */
export async function hashParams(obj: unknown): Promise<string> {
  if (obj === null || obj === undefined) return "";
  const sorted = sortedStringify(obj);
  const buf = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(sorted)
  );
  return Array.from(new Uint8Array(buf))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("")
    .slice(0, 16);
}

function sortedStringify(val: unknown): string {
  if (typeof val !== "object" || val === null) return JSON.stringify(val);
  if (Array.isArray(val)) return "[" + val.map(sortedStringify).join(",") + "]";
  const keys = Object.keys(val as object).sort();
  const pairs = keys.map((k) => JSON.stringify(k) + ":" + sortedStringify((val as Record<string, unknown>)[k]));
  return "{" + pairs.join(",") + "}";
}

// ── L1: in-memory ─────────────────────────────────────────────────────────────

export function memGet(key: string): MemEntry | undefined {
  return MEM.get(key);
}

export function memSet(key: string, entry: MemEntry): void {
  MEM.set(key, entry);
}

// ── L2: D1 ───────────────────────────────────────────────────────────────────

export async function d1Get(
  db: D1Database,
  url: string,
  endpoint: string,
  paramsHash: string
): Promise<PageCacheRow | null> {
  return db
    .prepare(
      "SELECT * FROM page_cache WHERE url = ? AND endpoint = ? AND params_hash = ? LIMIT 1"
    )
    .bind(url, endpoint, paramsHash)
    .first<PageCacheRow>();
}

export async function d1Set(
  db: D1Database,
  row: Omit<PageCacheRow, "created_at">
): Promise<void> {
  await db
    .prepare(
      `INSERT OR REPLACE INTO page_cache
         (url, endpoint, params_hash, html, markdown, result, title, created_at)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
    )
    .bind(
      row.url,
      row.endpoint,
      row.params_hash,
      row.html ?? null,
      row.markdown ?? null,
      row.result ?? null,
      row.title ?? null,
      Date.now()
    )
    .run();
}

// ── Combined read (L1 → L2) ───────────────────────────────────────────────────

export async function cacheGet(
  db: D1Database,
  url: string,
  endpoint: string,
  paramsHash: string
): Promise<MemEntry | null> {
  const key = makeCacheKey(url, endpoint, paramsHash);

  // L1 hit
  const mem = memGet(key);
  if (mem) return mem;

  // L2 hit
  const row = await d1Get(db, url, endpoint, paramsHash);
  if (row) {
    const entry: MemEntry = {
      html: row.html,
      markdown: row.markdown,
      result: row.result,
      title: row.title,
    };
    memSet(key, entry);
    return entry;
  }

  return null;
}

// ── Combined write (L1 + L2) ──────────────────────────────────────────────────

export async function cacheSet(
  db: D1Database,
  url: string,
  endpoint: string,
  paramsHash: string,
  entry: MemEntry & { title?: string | null }
): Promise<void> {
  const key = makeCacheKey(url, endpoint, paramsHash);
  memSet(key, entry);
  await d1Set(db, {
    url,
    endpoint,
    params_hash: paramsHash,
    html: entry.html ?? null,
    markdown: entry.markdown ?? null,
    result: entry.result ?? null,
    title: entry.title ?? null,
  });
}
```

- [ ] **Step 4.4: Run tests — expect pass**

```bash
cd tools/browser
npx vitest run src/__tests__/cache.test.ts
```

Expected: `4 passed`.

- [ ] **Step 4.5: Commit**

```bash
git add tools/browser/src/cache.ts tools/browser/src/__tests__/cache.test.ts
git commit -m "feat(browser): add two-layer cache module (mem + D1)"
```

---

### Task 5: cf.ts — Cloudflare Browser Rendering proxy

**Files:**
- Create: `tools/browser/src/cf.ts`

- [ ] **Step 5.1: Create the CF proxy module**

Create `tools/browser/src/cf.ts`:

```typescript
/**
 * Proxy layer (L3) for Cloudflare Browser Rendering REST API.
 * When CF_ACCOUNT_ID and CF_API_TOKEN secrets are set, requests are
 * forwarded to api.cloudflare.com and the response is returned verbatim.
 */

export interface CfProxyResult {
  ok: boolean;
  rateLimited: boolean;   // HTTP 429
  status: number;
  // For text/JSON responses:
  body: unknown;
  // For binary responses (screenshot, pdf):
  blob: Blob | null;
  // CF timing header, forwarded to caller
  browserMsUsed: string | null;
}

export function cfAvailable(env: { CF_ACCOUNT_ID?: string; CF_API_TOKEN?: string }): boolean {
  return Boolean(env.CF_ACCOUNT_ID && env.CF_API_TOKEN);
}

export async function proxyCF(
  endpoint: string,
  requestBody: unknown,
  env: { CF_ACCOUNT_ID: string; CF_API_TOKEN: string },
  binary = false
): Promise<CfProxyResult> {
  const url = `https://api.cloudflare.com/client/v4/accounts/${env.CF_ACCOUNT_ID}/browser-rendering/${endpoint}`;

  const res = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${env.CF_API_TOKEN}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(requestBody),
  });

  const browserMsUsed = res.headers.get("X-Browser-Ms-Used");

  if (!res.ok) {
    let body: unknown = null;
    try { body = await res.json(); } catch { /* ignore */ }
    return {
      ok: false,
      rateLimited: res.status === 429,
      status: res.status,
      body,
      blob: null,
      browserMsUsed,
    };
  }

  if (binary) {
    return {
      ok: true,
      rateLimited: false,
      status: res.status,
      body: null,
      blob: await res.blob(),
      browserMsUsed,
    };
  }

  const body = await res.json();
  return { ok: true, rateLimited: false, status: res.status, body, blob: null, browserMsUsed };
}
```

- [ ] **Step 5.2: Verify TypeScript compiles**

```bash
cd tools/browser
npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 5.3: Commit**

```bash
git add tools/browser/src/cf.ts
git commit -m "feat(browser): add CF Browser Rendering proxy module"
```

---

## Chunk 2: Text endpoints — content, markdown, links

### Task 6: /api/content handler

**Files:**
- Create: `tools/browser/src/content.ts`
- Create: `tools/browser/src/__tests__/content.test.ts`

- [ ] **Step 6.1: Write failing tests**

Create `tools/browser/src/__tests__/content.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";
import { buildContentResult } from "../content";
import { extractTitle } from "../markdown";

describe("buildContentResult", () => {
  it("returns html string for valid HTML", () => {
    const html = "<html><head><title>Test</title></head><body>Hello</body></html>";
    const result = buildContentResult(html);
    expect(result.html).toBe(html);
    expect(result.title).toBe("Test");
  });

  it("returns empty title when no <title> tag", () => {
    const html = "<html><body>No title</body></html>";
    expect(buildContentResult(html).title).toBe("");
  });
});
```

- [ ] **Step 6.2: Run — expect fail**

```bash
npx vitest run src/__tests__/content.test.ts
```

Expected: `Cannot find module '../content'`.

- [ ] **Step 6.3: Implement content.ts**

Create `tools/browser/src/content.ts`:

```typescript
import type { Context } from "hono";
import type { Env, ContentRequest } from "./types";
import { cacheGet, cacheSet, hashParams } from "./cache";
import { cfAvailable, proxyCF } from "./cf";
import { extractTitle } from "./markdown";

// Pure helper — exported for testing
export function buildContentResult(html: string): { html: string; title: string } {
  return { html, title: extractTitle(html) };
}

export async function handleContent(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: ContentRequest;
  try {
    body = await c.req.json<ContentRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  if (body.url) {
    try { new URL(body.url); } catch {
      return c.json({ success: false, errors: [{ code: 1001, message: "url is not a valid URL" }], result: null }, 400);
    }
  }

  // If raw html provided, skip cache and return immediately
  if (body.html && !body.url) {
    return c.json({ success: true, result: body.html });
  }

  const url = body.url!;
  const paramsHash = "";  // content caches by URL only

  // L1 + L2 cache
  const cached = await cacheGet(c.env.DB, url, "content", paramsHash);
  if (cached?.html != null) {
    return c.json({ success: true, result: cached.html });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("content", body, c.env as any);
    if (cf.ok) {
      const result = (cf.body as any)?.result ?? "";
      const title = extractTitle(typeof result === "string" ? result : "");
      await cacheSet(c.env.DB, url, "content", paramsHash, { html: result, markdown: null, result: null, title });
      const res = c.json({ success: true, result });
      if (cf.browserMsUsed) res.headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
      return res;
    }
    if (!cf.rateLimited) {
      return c.json({ success: false, errors: [{ code: cf.status, message: "CF error" }], result: null }, cf.status as any);
    }
    // rate limited → fall through to L4
  }

  // L4: own fallback — plain fetch
  try {
    const resp = await fetch(url, {
      headers: {
        "User-Agent": body.userAgent ?? "mizu-browser/1.0",
        Accept: "text/html,*/*",
        ...(body.setExtraHTTPHeaders ?? {}),
      },
      redirect: "follow",
    });
    const html = await resp.text();
    const { title } = buildContentResult(html);
    await cacheSet(c.env.DB, url, "content", paramsHash, { html, markdown: null, result: null, title });
    return c.json({ success: true, result: html });
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}
```

- [ ] **Step 6.4: Run tests — expect pass**

```bash
npx vitest run src/__tests__/content.test.ts
```

Expected: `2 passed`.

- [ ] **Step 6.5: Commit**

```bash
git add tools/browser/src/content.ts tools/browser/src/__tests__/content.test.ts
git commit -m "feat(browser): add /api/content handler with 4-layer stack"
```

---

### Task 7: /api/markdown handler

**Files:**
- Create: `tools/browser/src/markdown-ep.ts`
- Create: `tools/browser/src/__tests__/markdown-ep.test.ts`

- [ ] **Step 7.1: Write failing tests**

Create `tools/browser/src/__tests__/markdown-ep.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { htmlToMarkdown } from "../markdown";

// Reuse the existing converter — just verify it works for the endpoint's purpose
describe("htmlToMarkdown (used by /api/markdown fallback)", () => {
  it("converts h1 to # heading", () => {
    expect(htmlToMarkdown("<h1>Hello</h1>")).toContain("# Hello");
  });

  it("strips script tags", () => {
    const out = htmlToMarkdown("<script>alert(1)</script><p>text</p>");
    expect(out).not.toContain("alert");
    expect(out).toContain("text");
  });

  it("converts links", () => {
    const out = htmlToMarkdown('<a href="https://example.com">click</a>');
    expect(out).toContain("[click](https://example.com)");
  });
});
```

- [ ] **Step 7.2: Run — expect pass** (these reuse existing `markdown.ts`)

```bash
npx vitest run src/__tests__/markdown-ep.test.ts
```

Expected: `3 passed` (converter already works).

- [ ] **Step 7.3: Implement markdown-ep.ts**

Create `tools/browser/src/markdown-ep.ts`:

```typescript
import type { Context } from "hono";
import type { Env, MarkdownRequest } from "./types";
import { cacheGet, cacheSet } from "./cache";
import { cfAvailable, proxyCF } from "./cf";
import { htmlToMarkdown, extractTitle } from "./markdown";

export async function handleMarkdown(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: MarkdownRequest;
  try {
    body = await c.req.json<MarkdownRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  // Raw html → convert immediately, no cache
  if (body.html && !body.url) {
    return c.json({ success: true, result: htmlToMarkdown(body.html) });
  }

  const url = body.url!;
  const paramsHash = "";

  // L1 + L2
  const cached = await cacheGet(c.env.DB, url, "markdown", paramsHash);
  if (cached?.markdown != null) {
    return c.json({ success: true, result: cached.markdown });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("markdown", body, c.env as any);
    if (cf.ok) {
      const md = (cf.body as any)?.result ?? "";
      await cacheSet(c.env.DB, url, "markdown", paramsHash, { html: null, markdown: md, result: null, title: null });
      const res = c.json({ success: true, result: md });
      if (cf.browserMsUsed) res.headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
      return res;
    }
    if (!cf.rateLimited) {
      return c.json({ success: false, errors: [{ code: cf.status, message: "CF error" }], result: null }, cf.status as any);
    }
  }

  // L4: own fallback
  try {
    const resp = await fetch(url, {
      headers: { "User-Agent": body.userAgent ?? "mizu-browser/1.0", Accept: "text/html,*/*", ...(body.setExtraHTTPHeaders ?? {}) },
      redirect: "follow",
    });
    const html = await resp.text();
    const md = htmlToMarkdown(html);
    const title = extractTitle(html);
    await cacheSet(c.env.DB, url, "markdown", paramsHash, { html: null, markdown: md, result: null, title });
    return c.json({ success: true, result: md });
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}
```

- [ ] **Step 7.4: Verify TypeScript**

```bash
npx tsc --noEmit
```

- [ ] **Step 7.5: Commit**

```bash
git add tools/browser/src/markdown-ep.ts tools/browser/src/__tests__/markdown-ep.test.ts
git commit -m "feat(browser): add /api/markdown handler"
```

---

### Task 8: /api/links handler

**Files:**
- Create: `tools/browser/src/links-ep.ts`
- Create: `tools/browser/src/__tests__/links-ep.test.ts`

- [ ] **Step 8.1: Write failing tests**

Create `tools/browser/src/__tests__/links-ep.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { filterLinksForEndpoint } from "../links-ep";

describe("filterLinksForEndpoint", () => {
  const links = [
    "https://example.com/a",
    "https://other.com/b",
    "https://example.com/c",
  ];

  it("returns all links when no filters set", () => {
    expect(filterLinksForEndpoint(links, "https://example.com", false, false)).toEqual(links);
  });

  it("excludes external links when excludeExternalLinks=true", () => {
    const result = filterLinksForEndpoint(links, "https://example.com", false, true);
    expect(result).toEqual(["https://example.com/a", "https://example.com/c"]);
  });

  it("returns all when visibleLinksOnly=true (fallback: same as all)", () => {
    // In fallback mode we cannot determine visibility, so we return all
    const result = filterLinksForEndpoint(links, "https://example.com", true, false);
    expect(result).toEqual(links);
  });
});
```

- [ ] **Step 8.2: Run — expect fail**

```bash
npx vitest run src/__tests__/links-ep.test.ts
```

Expected: `Cannot find module '../links-ep'`.

- [ ] **Step 8.3: Implement links-ep.ts**

Create `tools/browser/src/links-ep.ts`:

```typescript
import type { Context } from "hono";
import type { Env, LinksRequest } from "./types";
import { cacheGet, cacheSet } from "./cache";
import { cfAvailable, proxyCF } from "./cf";
import { extractLinks } from "./links";

// paramsHash input for /links: the two boolean flags
export function linksParamsObj(req: LinksRequest) {
  return { visibleLinksOnly: req.visibleLinksOnly ?? false, excludeExternalLinks: req.excludeExternalLinks ?? false };
}

/**
 * Filter already-resolved absolute URLs.
 * In fallback mode, visibleLinksOnly is not enforceable (no layout engine)
 * so we return all links and the caller knows via X-Fallback header.
 */
export function filterLinksForEndpoint(
  links: string[],
  pageUrl: string,
  _visibleLinksOnly: boolean,
  excludeExternalLinks: boolean
): string[] {
  if (!excludeExternalLinks) return links;
  const host = new URL(pageUrl).hostname;
  return links.filter((l) => {
    try { return new URL(l).hostname === host; } catch { return false; }
  });
}

export async function handleLinks(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: LinksRequest;
  try {
    body = await c.req.json<LinksRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  // Raw html: extract immediately, no cache
  if (body.html && !body.url) {
    const links = await extractLinks(body.html, "https://localhost");
    return c.json({ success: true, result: links });
  }

  const url = body.url!;
  // params_hash encodes the two filter flags (short enough to use as literal)
  const visibleLinksOnly = body.visibleLinksOnly ?? false;
  const excludeExternalLinks = body.excludeExternalLinks ?? false;
  const paramsHash = `${visibleLinksOnly}:${excludeExternalLinks}`;

  // L1 + L2
  const cached = await cacheGet(c.env.DB, url, "links", paramsHash);
  if (cached?.result != null) {
    return c.json({ success: true, result: JSON.parse(cached.result) });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("links", body, c.env as any);
    if (cf.ok) {
      const result: string[] = (cf.body as any)?.result ?? [];
      await cacheSet(c.env.DB, url, "links", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
      const res = c.json({ success: true, result });
      if (cf.browserMsUsed) res.headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
      return res;
    }
    if (!cf.rateLimited) {
      return c.json({ success: false, errors: [{ code: cf.status, message: "CF error" }], result: null }, cf.status as any);
    }
  }

  // L4: own fallback
  try {
    const resp = await fetch(url, {
      headers: { "User-Agent": body.userAgent ?? "mizu-browser/1.0", Accept: "text/html,*/*", ...(body.setExtraHTTPHeaders ?? {}) },
      redirect: "follow",
    });
    const html = await resp.text();
    const rawLinks = await extractLinks(html, url);
    const result = filterLinksForEndpoint(rawLinks, url, visibleLinksOnly, excludeExternalLinks);
    await cacheSet(c.env.DB, url, "links", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
    const res = c.json({ success: true, result });
    res.headers.set("X-Fallback", "true");  // caller: screenshot not available
    return res;
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}
```

- [ ] **Step 8.4: Run tests — expect pass**

```bash
npx vitest run src/__tests__/links-ep.test.ts
```

Expected: `3 passed`.

- [ ] **Step 8.5: Commit**

```bash
git add tools/browser/src/links-ep.ts tools/browser/src/__tests__/links-ep.test.ts
git commit -m "feat(browser): add /api/links handler"
```

---

## Chunk 3: Complex text endpoints — snapshot, scrape, json

### Task 9: /api/snapshot handler

**Files:**
- Create: `tools/browser/src/snapshot.ts`
- Create: `tools/browser/src/__tests__/snapshot.test.ts`

- [ ] **Step 9.1: Write failing tests**

Create `tools/browser/src/__tests__/snapshot.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { buildSnapshotFallback } from "../snapshot";

describe("buildSnapshotFallback", () => {
  it("returns content and null screenshot", () => {
    const html = "<html><body>Hello</body></html>";
    const result = buildSnapshotFallback(html);
    expect(result.content).toBe(html);
    expect(result.screenshot).toBeNull();
  });
});
```

- [ ] **Step 9.2: Run — expect fail**

```bash
npx vitest run src/__tests__/snapshot.test.ts
```

- [ ] **Step 9.3: Implement snapshot.ts**

Create `tools/browser/src/snapshot.ts`:

```typescript
import type { Context } from "hono";
import type { Env, SnapshotRequest, SnapshotResult } from "./types";
import { cacheGet, cacheSet } from "./cache";
import { cfAvailable, proxyCF } from "./cf";
import { extractTitle } from "./markdown";

export function buildSnapshotFallback(html: string): SnapshotResult {
  return { content: html, screenshot: null };
}

export async function handleSnapshot(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: SnapshotRequest;
  try {
    body = await c.req.json<SnapshotRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  if (body.html && !body.url) {
    return c.json({ success: true, result: buildSnapshotFallback(body.html) });
  }

  const url = body.url!;
  const paramsHash = "";

  // L1 + L2 — html col = content, result col = {"screenshot":"..."}
  const cached = await cacheGet(c.env.DB, url, "snapshot", paramsHash);
  if (cached?.html != null) {
    const screenshot = cached.result ? (JSON.parse(cached.result) as { screenshot: string | null }).screenshot : null;
    return c.json({ success: true, result: { content: cached.html, screenshot } });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("snapshot", body, c.env as any);
    if (cf.ok) {
      const cfResult = (cf.body as any)?.result as SnapshotResult;
      const title = extractTitle(cfResult.content ?? "");
      await cacheSet(c.env.DB, url, "snapshot", paramsHash, {
        html: cfResult.content ?? null,
        markdown: null,
        result: JSON.stringify({ screenshot: cfResult.screenshot }),
        title,
      });
      const res = c.json({ success: true, result: cfResult });
      if (cf.browserMsUsed) res.headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
      return res;
    }
    if (!cf.rateLimited) {
      return c.json({ success: false, errors: [{ code: cf.status, message: "CF error" }], result: null }, cf.status as any);
    }
  }

  // L4: own fallback — fetch HTML, screenshot = null
  try {
    const resp = await fetch(url, {
      headers: { "User-Agent": body.userAgent ?? "mizu-browser/1.0", Accept: "text/html,*/*", ...(body.setExtraHTTPHeaders ?? {}) },
      redirect: "follow",
    });
    const html = await resp.text();
    const title = extractTitle(html);
    const result = buildSnapshotFallback(html);
    await cacheSet(c.env.DB, url, "snapshot", paramsHash, {
      html,
      markdown: null,
      result: JSON.stringify({ screenshot: null }),
      title,
    });
    const res = c.json({ success: true, result });
    res.headers.set("X-Fallback", "true");
    return res;
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}
```

- [ ] **Step 9.4: Run tests — expect pass**

```bash
npx vitest run src/__tests__/snapshot.test.ts
```

Expected: `1 passed`.

- [ ] **Step 9.5: Commit**

```bash
git add tools/browser/src/snapshot.ts tools/browser/src/__tests__/snapshot.test.ts
git commit -m "feat(browser): add /api/snapshot handler"
```

---

### Task 10: /api/scrape handler

**Files:**
- Create: `tools/browser/src/scrape.ts`
- Create: `tools/browser/src/__tests__/scrape.test.ts`

- [ ] **Step 10.1: Write failing tests**

Create `tools/browser/src/__tests__/scrape.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { scrapeHtml } from "../scrape";

describe("scrapeHtml", () => {
  const html = `<html><body>
    <h1 class="title">Main Title</h1>
    <h2>Sub One</h2>
    <h2>Sub Two</h2>
    <p id="intro">Intro text</p>
  </body></html>`;

  it("extracts text from a single selector", async () => {
    const results = await scrapeHtml(html, [{ selector: "h1" }]);
    expect(results).toHaveLength(1);
    expect(results[0].selector).toBe("h1");
    expect(results[0].results).toHaveLength(1);
    expect(results[0].results[0].text).toContain("Main Title");
  });

  it("extracts multiple matches for the same selector", async () => {
    const results = await scrapeHtml(html, [{ selector: "h2" }]);
    expect(results[0].results).toHaveLength(2);
  });

  it("extracts attributes", async () => {
    const results = await scrapeHtml(html, [{ selector: "h1" }]);
    const attrs = results[0].results[0].attributes;
    expect(attrs).toContainEqual({ name: "class", value: "title" });
  });

  it("returns zero dimensions (fallback, no layout engine)", async () => {
    const results = await scrapeHtml(html, [{ selector: "h1" }]);
    expect(results[0].results[0].height).toBe(0);
    expect(results[0].results[0].width).toBe(0);
  });

  it("returns empty results for non-matching selector", async () => {
    const results = await scrapeHtml(html, [{ selector: "table" }]);
    expect(results[0].results).toHaveLength(0);
  });

  it("handles multiple selectors in one call", async () => {
    const results = await scrapeHtml(html, [{ selector: "h1" }, { selector: "p" }]);
    expect(results).toHaveLength(2);
    expect(results[1].results[0].text).toContain("Intro text");
  });
});
```

- [ ] **Step 10.2: Run — expect fail**

```bash
npx vitest run src/__tests__/scrape.test.ts
```

- [ ] **Step 10.3: Implement scrape.ts**

Create `tools/browser/src/scrape.ts`:

```typescript
import type { Context } from "hono";
import type { Env, ScrapeRequest, ScrapeSelectorResult, ScrapeNodeResult } from "./types";
import { cacheGet, cacheSet, hashParams } from "./cache";
import { cfAvailable, proxyCF } from "./cf";

/**
 * Fallback scraper using HTMLRewriter.
 * Bounding box (height, width, top, left) = 0 — no layout engine available.
 */
export async function scrapeHtml(
  html: string,
  elements: Array<{ selector: string }>
): Promise<ScrapeSelectorResult[]> {
  return Promise.all(
    elements.map(async ({ selector }) => {
      const nodes: ScrapeNodeResult[] = [];
      let current: ScrapeNodeResult | null = null;
      let textBuf = "";
      let htmlBuf = "";

      const rw = new HTMLRewriter()
        .on(selector, {
          element(el) {
            // Save previous node if any
            if (current) {
              current.text = textBuf.trim();
              current.html = htmlBuf.trim();
              nodes.push(current);
            }
            const attrs: Array<{ name: string; value: string }> = [];
            for (const [name, value] of el.attributes) attrs.push({ name, value });
            current = { text: "", html: "", attributes: attrs, height: 0, width: 0, top: 0, left: 0 };
            textBuf = "";
            htmlBuf = el.tagName ? `<${el.tagName}` + attrs.map(a => ` ${a.name}="${a.value}"`).join("") + ">" : "";
            el.onEndTag((tag) => {
              htmlBuf += `</${tag.name}>`;
              if (current) {
                current.text = textBuf.trim();
                current.html = htmlBuf.trim();
                nodes.push(current);
                current = null;
                textBuf = "";
                htmlBuf = "";
              }
            });
          },
          text(chunk) {
            if (current) {
              textBuf += chunk.text;
              htmlBuf += chunk.text;
            }
          },
        })
        .transform(new Response(html, { headers: { "Content-Type": "text/html" } }));

      await rw.text();
      return { selector, results: nodes };
    })
  );
}

export async function handleScrape(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: ScrapeRequest;
  try {
    body = await c.req.json<ScrapeRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }
  if (!body.elements || body.elements.length === 0) {
    return c.json({ success: false, errors: [{ code: 1001, message: "elements is required" }], result: null }, 400);
  }

  // Sort selectors for deterministic hashing
  const sortedSelectors = [...body.elements].sort((a, b) => a.selector.localeCompare(b.selector));
  const paramsHash = await hashParams(sortedSelectors.map(e => e.selector));

  if (body.html && !body.url) {
    const result = await scrapeHtml(body.html, body.elements);
    return c.json({ success: true, result });
  }

  const url = body.url!;

  // L1 + L2
  const cached = await cacheGet(c.env.DB, url, "scrape", paramsHash);
  if (cached?.result != null) {
    return c.json({ success: true, result: JSON.parse(cached.result) });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("scrape", body, c.env as any);
    if (cf.ok) {
      const result = (cf.body as any)?.result ?? [];
      await cacheSet(c.env.DB, url, "scrape", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
      const res = c.json({ success: true, result });
      if (cf.browserMsUsed) res.headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
      return res;
    }
    if (!cf.rateLimited) {
      return c.json({ success: false, errors: [{ code: cf.status, message: "CF error" }], result: null }, cf.status as any);
    }
  }

  // L4: own fallback
  try {
    const resp = await fetch(url, {
      headers: { "User-Agent": body.userAgent ?? "mizu-browser/1.0", Accept: "text/html,*/*", ...(body.setExtraHTTPHeaders ?? {}) },
      redirect: "follow",
    });
    const html = await resp.text();
    const result = await scrapeHtml(html, body.elements);
    await cacheSet(c.env.DB, url, "scrape", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
    const res = c.json({ success: true, result });
    res.headers.set("X-Fallback", "true");
    return res;
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}
```

- [ ] **Step 10.4: Run tests — expect pass**

```bash
npx vitest run src/__tests__/scrape.test.ts
```

Expected: `6 passed`.

- [ ] **Step 10.5: Commit**

```bash
git add tools/browser/src/scrape.ts tools/browser/src/__tests__/scrape.test.ts
git commit -m "feat(browser): add /api/scrape handler with HTMLRewriter fallback"
```

---

### Task 11: /api/json handler

**Files:**
- Create: `tools/browser/src/json-ep.ts`
- Create: `tools/browser/src/__tests__/json-ep.test.ts`

- [ ] **Step 11.1: Write failing tests**

Create `tools/browser/src/__tests__/json-ep.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { jsonParamsObj } from "../json-ep";

describe("jsonParamsObj", () => {
  it("returns prompt and schema from request", () => {
    const req = {
      prompt: "extract the title",
      response_format: { type: "json_schema" as const, schema: { type: "object" } },
    };
    const obj = jsonParamsObj(req);
    expect(obj.prompt).toBe("extract the title");
    expect(obj.schema).toEqual({ type: "object" });
  });

  it("handles missing prompt and schema", () => {
    const obj = jsonParamsObj({});
    expect(obj.prompt).toBeUndefined();
    expect(obj.schema).toBeUndefined();
  });
});
```

- [ ] **Step 11.2: Run — expect fail**

```bash
npx vitest run src/__tests__/json-ep.test.ts
```

- [ ] **Step 11.3: Implement json-ep.ts**

Create `tools/browser/src/json-ep.ts`:

```typescript
import type { Context } from "hono";
import type { Env, JsonRequest } from "./types";
import { cacheGet, cacheSet, hashParams } from "./cache";
import { cfAvailable, proxyCF } from "./cf";
import { htmlToMarkdown } from "./markdown";

export function jsonParamsObj(req: Partial<JsonRequest>) {
  return {
    prompt: req.prompt,
    schema: req.response_format?.schema,
  };
}

export async function handleJson(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: JsonRequest;
  try {
    body = await c.req.json<JsonRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  const paramsHash = await hashParams(jsonParamsObj(body));

  // Raw html: skip cache and CF, try AI fallback directly
  if (body.html && !body.url) {
    return await runJsonFallback(c, body.html, body);
  }

  const url = body.url!;

  // L1 + L2
  const cached = await cacheGet(c.env.DB, url, "json", paramsHash);
  if (cached?.result != null) {
    return c.json({ success: true, result: JSON.parse(cached.result) });
  }

  // L3: CF proxy
  if (cfAvailable(c.env)) {
    const cf = await proxyCF("json", body, c.env as any);
    if (cf.ok) {
      const result = (cf.body as any)?.result ?? {};
      await cacheSet(c.env.DB, url, "json", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
      const res = c.json({ success: true, result });
      if (cf.browserMsUsed) res.headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
      return res;
    }
    if (!cf.rateLimited) {
      return c.json({ success: false, errors: [{ code: cf.status, message: "CF error" }], result: null }, cf.status as any);
    }
  }

  // L4: own fallback — fetch HTML, run Workers AI if binding available
  try {
    const resp = await fetch(url, {
      headers: { "User-Agent": body.userAgent ?? "mizu-browser/1.0", Accept: "text/html,*/*", ...(body.setExtraHTTPHeaders ?? {}) },
      redirect: "follow",
    });
    const html = await resp.text();
    return await runJsonFallback(c, html, body, url, paramsHash);
  } catch {
    return c.json({ success: false, errors: [{ code: 502, message: "Failed to fetch URL" }], result: null }, 502);
  }
}

async function runJsonFallback(
  c: Context<{ Bindings: Env }>,
  html: string,
  body: JsonRequest,
  url?: string,
  paramsHash?: string
): Promise<Response> {
  const env = c.env as any;

  // Try Workers AI if binding is available
  if (env.AI && (body.prompt || body.response_format)) {
    try {
      const md = htmlToMarkdown(html);
      const systemPrompt = body.response_format
        ? `Extract data matching this JSON schema: ${JSON.stringify(body.response_format.schema)}. Return only valid JSON.`
        : "Extract structured data from the content. Return only valid JSON.";
      const userPrompt = body.prompt ? `${body.prompt}\n\n${md}` : md;

      const aiResp: any = await env.AI.run("@cf/meta/llama-3.1-8b-instruct-fast", {
        messages: [
          { role: "system", content: systemPrompt },
          { role: "user", content: userPrompt },
        ],
      });

      const text: string = aiResp?.response ?? "";
      const jsonMatch = text.match(/\{[\s\S]*\}|\[[\s\S]*\]/);
      if (jsonMatch) {
        const result = JSON.parse(jsonMatch[0]);
        if (url && paramsHash !== undefined) {
          await cacheSet(c.env.DB, url, "json", paramsHash, { html: null, markdown: null, result: JSON.stringify(result), title: null });
        }
        const res = c.json({ success: true, result });
        res.headers.set("X-Fallback", "true");
        return res;
      }
    } catch {
      // AI failed — fall through to error
    }
  }

  // No AI binding or AI failed
  return c.json({
    success: false,
    errors: [{ code: 503, message: "AI extraction unavailable; CF rate limited and AI binding not configured" }],
    result: null,
  }, 503);
}
```

- [ ] **Step 11.4: Run tests — expect pass**

```bash
npx vitest run src/__tests__/json-ep.test.ts
```

Expected: `2 passed`.

- [ ] **Step 11.5: Commit**

```bash
git add tools/browser/src/json-ep.ts tools/browser/src/__tests__/json-ep.test.ts
git commit -m "feat(browser): add /api/json handler with Workers AI fallback"
```

---

## Chunk 4: Binary endpoints — screenshot, pdf

### Task 12: /api/screenshot handler

**Files:**
- Create: `tools/browser/src/screenshot.ts`

No unit tests: the handler is CF-only (no pure logic to test independently). Integration-tested via `wrangler dev`.

- [ ] **Step 12.1: Implement screenshot.ts**

Create `tools/browser/src/screenshot.ts`:

```typescript
import type { Context } from "hono";
import type { Env, ScreenshotRequest } from "./types";
import { cfAvailable, proxyCF } from "./cf";

export async function handleScreenshot(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: ScreenshotRequest;
  try {
    body = await c.req.json<ScreenshotRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  if (!cfAvailable(c.env)) {
    return c.json(
      { success: false, errors: [{ code: 503, message: "CF credentials not configured; screenshot requires a real browser" }], result: null },
      503
    );
  }

  const type = body.screenshotOptions?.type ?? "png";
  const contentType = type === "jpeg" ? "image/jpeg" : "image/png";

  const cf = await proxyCF("screenshot", body, c.env as any, true);

  if (cf.ok && cf.blob) {
    const headers = new Headers({ "Content-Type": contentType });
    if (cf.browserMsUsed) headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
    return new Response(cf.blob, { headers });
  }

  if (cf.rateLimited) {
    return c.json(
      { success: false, errors: [{ code: 429, message: "CF rate limited; screenshot requires a real browser" }], result: null },
      503
    );
  }

  return c.json(
    { success: false, errors: [{ code: cf.status, message: "CF error" }], result: null },
    cf.status as any
  );
}
```

- [ ] **Step 12.2: Verify TypeScript**

```bash
npx tsc --noEmit
```

- [ ] **Step 12.3: Commit**

```bash
git add tools/browser/src/screenshot.ts
git commit -m "feat(browser): add /api/screenshot handler (CF proxy, no cache)"
```

---

### Task 13: /api/pdf handler

**Files:**
- Create: `tools/browser/src/pdf.ts`

- [ ] **Step 13.1: Implement pdf.ts**

Create `tools/browser/src/pdf.ts`:

```typescript
import type { Context } from "hono";
import type { Env, PdfRequest } from "./types";
import { cfAvailable, proxyCF } from "./cf";

export async function handlePdf(c: Context<{ Bindings: Env }>): Promise<Response> {
  let body: PdfRequest;
  try {
    body = await c.req.json<PdfRequest>();
  } catch {
    return c.json({ success: false, errors: [{ code: 1001, message: "Invalid JSON body" }], result: null }, 400);
  }

  if (!body.url && !body.html) {
    return c.json({ success: false, errors: [{ code: 1001, message: "url or html is required" }], result: null }, 400);
  }

  if (!cfAvailable(c.env)) {
    return c.json(
      { success: false, errors: [{ code: 503, message: "CF credentials not configured; PDF rendering requires a real browser" }], result: null },
      503
    );
  }

  const cf = await proxyCF("pdf", body, c.env as any, true);

  if (cf.ok && cf.blob) {
    const headers = new Headers({ "Content-Type": "application/pdf" });
    if (cf.browserMsUsed) headers.set("X-Browser-Ms-Used", cf.browserMsUsed);
    return new Response(cf.blob, { headers });
  }

  if (cf.rateLimited) {
    return c.json(
      { success: false, errors: [{ code: 429, message: "CF rate limited; PDF rendering requires a real browser" }], result: null },
      503
    );
  }

  return c.json(
    { success: false, errors: [{ code: cf.status, message: "CF error" }], result: null },
    cf.status as any
  );
}
```

- [ ] **Step 13.2: Verify TypeScript**

```bash
npx tsc --noEmit
```

- [ ] **Step 13.3: Commit**

```bash
git add tools/browser/src/pdf.ts
git commit -m "feat(browser): add /api/pdf handler (CF proxy, no cache)"
```

---

## Chunk 5: Route wiring + wrangler config

### Task 14: Wire all routes in index.ts

**Files:**
- Modify: `tools/browser/src/index.ts`

- [ ] **Step 14.1: Add imports and routes**

Replace the content of `tools/browser/src/index.ts` with:

```typescript
import { Hono } from "hono";
import { cors } from "hono/cors";
import { authMiddleware } from "./auth";
import { handlePost, handleGet, handleDelete } from "./crawl";
import { handleCrawlQueue } from "./queue";
import { handleContent } from "./content";
import { handleScreenshot } from "./screenshot";
import { handlePdf } from "./pdf";
import { handleMarkdown } from "./markdown-ep";
import { handleSnapshot } from "./snapshot";
import { handleScrape } from "./scrape";
import { handleJson } from "./json-ep";
import { handleLinks } from "./links-ep";
import type { Env, CrawlMessage } from "./types";

const app = new Hono<{ Bindings: Env }>();

app.use("*", cors());

// Health check (no auth)
app.get("/", (c) => c.json({ ok: true, service: "browser-worker" }));

// All /api/* routes require auth
app.use("/api/*", authMiddleware);

// Crawl endpoints (existing)
app.post("/api/crawl", handlePost);
app.get("/api/crawl/:id", handleGet);
app.delete("/api/crawl/:id", handleDelete);

// Single-URL rendering endpoints (new)
app.post("/api/content",    handleContent);
app.post("/api/screenshot", handleScreenshot);
app.post("/api/pdf",        handlePdf);
app.post("/api/markdown",   handleMarkdown);
app.post("/api/snapshot",   handleSnapshot);
app.post("/api/scrape",     handleScrape);
app.post("/api/json",       handleJson);
app.post("/api/links",      handleLinks);

// 404 fallback
app.notFound((c) =>
  c.json({ success: false, errors: [{ code: 404, message: "Not found" }], result: null }, 404)
);

// Error handler
app.onError((err, c) => {
  console.error("[worker] unhandled error:", err);
  return c.json({ success: false, errors: [{ code: 500, message: "Internal server error" }], result: null }, 500);
});

export default {
  fetch: app.fetch,

  async queue(batch: MessageBatch<unknown>, env: Env): Promise<void> {
    await handleCrawlQueue(batch as MessageBatch<CrawlMessage>, env);
  },
} satisfies ExportedHandler<Env>;
```

- [ ] **Step 14.2: Verify TypeScript**

```bash
npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 14.3: Commit**

```bash
git add tools/browser/src/index.ts
git commit -m "feat(browser): wire 8 new endpoints in index.ts"
```

---

### Task 15: Update Env type and wrangler.toml for CF secrets

**Files:**
- Modify: `tools/browser/src/types.ts`
- Modify: `tools/browser/wrangler.toml`

- [ ] **Step 15.1: Add CF secret fields to Env interface**

In `tools/browser/src/types.ts`, update the `Env` interface:

```typescript
export interface Env {
  DB: D1Database;
  CRAWL_QUEUE: Queue;
  AUTH_TOKEN: string;
  // CF Browser Rendering credentials (optional secrets)
  // Set via: wrangler secret put CF_ACCOUNT_ID / CF_API_TOKEN
  CF_ACCOUNT_ID?: string;
  CF_API_TOKEN?: string;
  // Optional Workers AI binding for /api/json fallback
  // Uncomment [[ai]] in wrangler.toml to enable
  AI?: any;
}
```

- [ ] **Step 15.2: Document secrets in wrangler.toml**

Append to `tools/browser/wrangler.toml`:

```toml
# Optional: CF Browser Rendering credentials for L3 proxy
# Set with: wrangler secret put CF_ACCOUNT_ID
#            wrangler secret put CF_API_TOKEN
# Token must have "Browser Rendering > Edit" permission.
# When absent, all endpoints use L4 own-fallback only.

# Optional: Workers AI binding for /api/json AI extraction fallback
# Uncomment to enable:
# [ai]
# binding = "AI"
```

- [ ] **Step 15.3: Verify TypeScript**

```bash
npx tsc --noEmit
```

- [ ] **Step 15.4: Commit**

```bash
git add tools/browser/src/types.ts tools/browser/wrangler.toml
git commit -m "feat(browser): add CF_ACCOUNT_ID/CF_API_TOKEN + AI binding to Env"
```

---

### Task 16: Full test run + smoke test

- [ ] **Step 16.1: Run all tests**

```bash
cd tools/browser
npx vitest run
```

Expected: all tests pass, 0 failures.

- [ ] **Step 16.2: Smoke test with wrangler dev**

```bash
wrangler dev --local
```

In a second terminal:

```bash
# Health check
curl http://localhost:8787/

# /api/markdown with own fallback (no CF creds in local dev)
curl -s -X POST http://localhost:8787/api/markdown \
  -H "Authorization: Bearer <your-AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}' | jq .

# Expected: {"success":true,"result":"# Example Domain\n\n..."}

# /api/links
curl -s -X POST http://localhost:8787/api/links \
  -H "Authorization: Bearer <your-AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}' | jq .

# Expected: {"success":true,"result":["https://www.iana.org/domains/reserved"]}

# /api/scrape
curl -s -X POST http://localhost:8787/api/scrape \
  -H "Authorization: Bearer <your-AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","elements":[{"selector":"h1"},{"selector":"p"}]}' | jq .

# /api/screenshot (no CF creds → 503)
curl -s -X POST http://localhost:8787/api/screenshot \
  -H "Authorization: Bearer <your-AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}' | jq .
# Expected: 503 {"success":false,"errors":[{"code":503,...}]}
```

- [ ] **Step 16.3: Apply schema migration to remote D1**

```bash
npm run db:migrate:remote
```

Expected: `Executing on remote database browser-db... ✓`

- [ ] **Step 16.4: Deploy**

```bash
wrangler deploy
```

Expected: `Deployed browser (X versions)` and worker URL printed.

- [ ] **Step 16.5: Final commit**

```bash
git add -A
git commit -m "feat(browser): complete browser more API endpoints (content/screenshot/pdf/markdown/snapshot/scrape/json/links)"
```

---

## Summary

| Task | Files | Commit message |
|---|---|---|
| 1 | vitest config, package.json | `chore: add vitest + cloudflare workers pool` |
| 2 | schema.sql | `feat: add page_cache table` |
| 3 | types.ts | `feat: add new endpoint types` |
| 4 | cache.ts, cache.test.ts | `feat: add two-layer cache module` |
| 5 | cf.ts | `feat: add CF proxy module` |
| 6 | content.ts, content.test.ts | `feat: add /api/content` |
| 7 | markdown-ep.ts, markdown-ep.test.ts | `feat: add /api/markdown` |
| 8 | links-ep.ts, links-ep.test.ts | `feat: add /api/links` |
| 9 | snapshot.ts, snapshot.test.ts | `feat: add /api/snapshot` |
| 10 | scrape.ts, scrape.test.ts | `feat: add /api/scrape` |
| 11 | json-ep.ts, json-ep.test.ts | `feat: add /api/json` |
| 12 | screenshot.ts | `feat: add /api/screenshot` |
| 13 | pdf.ts | `feat: add /api/pdf` |
| 14 | index.ts | `feat: wire 8 new endpoints` |
| 15 | types.ts, wrangler.toml | `feat: add CF secrets to Env` |
| 16 | — | smoke test + deploy |
