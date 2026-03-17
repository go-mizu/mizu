# Ed25519 Public-Key Auth Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace HMAC-signed actor tokens with Ed25519 public-key auth, update landing page and docs to match.

**Architecture:** New `crypto.ts` module handles Ed25519 + SHA-256 + canonical request building. New `register.ts` and `keys.ts` handle identity management. Rewritten `auth.ts` parses the `CHAT-ED25519` Authorization header and verifies signatures. Remove `token.ts` and all `AUTH_TOKEN` references. Full-width docs rewrite with working code examples.

**Tech Stack:** Cloudflare Workers, Hono, D1, Web Crypto API (Ed25519), TypeScript

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `schema.sql` | Modify | Add `actors` table |
| `src/types.ts` | Modify | Remove `AUTH_TOKEN` from Env, add registration/rotation types |
| `src/crypto.ts` | Create | base64url, SHA-256 hex, Ed25519 key import, canonical request, string-to-sign |
| `src/auth.ts` | Rewrite | Parse CHAT-ED25519 header, verify signature, in-memory key cache |
| `src/register.ts` | Create | POST /api/register handler with IP rate limiting |
| `src/keys.ts` | Create | Key rotation + recovery code rotation + account deletion |
| `src/index.ts` | Modify | Wire new routes, remove admin auth + token minting |
| `src/token.ts` | Delete | No longer needed |
| `src/actor.ts` | Keep | isValidActor + isMember unchanged |
| `src/chat.ts` | Keep | Unchanged (already uses `c.get("actor")`) |
| `src/message.ts` | Keep | Unchanged |
| `src/id.ts` | Keep | Unchanged |
| `src/landing.ts` | Rewrite | Agent icons, copy button fix, Ed25519 setup instructions |
| `src/docs.ts` | Rewrite | Full-width, human-friendly, working code examples |
| `wrangler.toml` | Keep | No changes needed (AUTH_TOKEN is a secret, not in toml) |

---

### Task 1: Schema — Add actors table

**Files:**
- Modify: `schema.sql`

- [ ] **Step 1: Add actors table to schema.sql**

Append after the existing tables:

```sql
CREATE TABLE IF NOT EXISTS actors (
  actor TEXT PRIMARY KEY,
  public_key TEXT NOT NULL,
  recovery_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  created_ip_hash TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_actors_ip ON actors(created_ip_hash, created_at);
```

- [ ] **Step 2: Run migration on remote D1**

```bash
cd blueprints/now/tools/worker/chat
npx wrangler d1 execute chat-db --remote --file=schema.sql
```

Expected: `🌀 Executing on remote database chat-db` with success

- [ ] **Step 3: Commit**

```bash
git add schema.sql
git commit -m "schema: add actors table for Ed25519 public-key auth"
```

---

### Task 2: Types — Update Env and add new request types

**Files:**
- Modify: `src/types.ts`

- [ ] **Step 1: Update types.ts**

Remove `AUTH_TOKEN` from `Env`. Remove `CreateTokenRequest`. Keep ALL existing domain
types (Chat, Message, CreateChatRequest, SendMessageRequest, ChatRow, MemberRow,
MessageRow) unchanged. Add new request types at the end:

```typescript
export interface Env {
  DB: D1Database;
  // AUTH_TOKEN removed
}

// Hono variables, domain types, existing request types, DB row types
// ALL STAY EXACTLY THE SAME — only remove CreateTokenRequest

// --- Registration & key management (ADD these) ---

export interface RegisterRequest {
  actor: string;
  public_key: string;
}

export interface RotateKeyRequest {
  actor: string;
  recovery_code: string;
  new_public_key: string;
}

export interface RotateRecoveryRequest {
  actor: string;
  recovery_code: string;
}

export interface DeleteActorRequest {
  actor: string;
  recovery_code: string;
}
```

Remove `CreateTokenRequest`.

- [ ] **Step 2: Verify types compile**

```bash
npx tsc --noEmit
```

Expected: errors in `index.ts` and `auth.ts` (they still reference AUTH_TOKEN and token.ts) — that's fine, we'll fix those next.

- [ ] **Step 3: Commit**

```bash
git add src/types.ts
git commit -m "types: remove AUTH_TOKEN, add Ed25519 registration types"
```

---

### Task 3: Crypto — Ed25519 + SHA-256 + canonical request

**Files:**
- Create: `src/crypto.ts`

- [ ] **Step 1: Create crypto.ts**

```typescript
const encoder = new TextEncoder();

// --- base64url ---

export function base64url(buf: ArrayBuffer | Uint8Array): string {
  const bytes = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  let s = "";
  for (const b of bytes) s += String.fromCharCode(b);
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function base64urlDecode(s: string): Uint8Array {
  const padded = s.replace(/-/g, "+").replace(/_/g, "/");
  const bin = atob(padded);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

// --- SHA-256 ---

export async function sha256hex(data: string | Uint8Array): Promise<string> {
  const input = typeof data === "string" ? encoder.encode(data) : data;
  const hash = await crypto.subtle.digest("SHA-256", input);
  return Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0")).join("");
}

// --- Ed25519 key import ---

export async function importEd25519PublicKey(base64urlKey: string): Promise<CryptoKey> {
  const raw = base64urlDecode(base64urlKey);
  return crypto.subtle.importKey("raw", raw, { name: "Ed25519" }, false, ["verify"]);
}

// --- Canonical request ---

export function buildCanonicalRequest(
  method: string,
  path: string,
  query: string,
  bodyHash: string
): string {
  return `${method}\n${path}\n${query}\n${bodyHash}`;
}

export function buildStringToSign(
  timestamp: string,
  actor: string,
  canonicalRequestHash: string
): string {
  return `CHAT-ED25519\n${timestamp}\n${actor}\n${canonicalRequestHash}`;
}

export function sortedQueryString(url: URL): string {
  // Use raw query string to preserve URL-encoded form (spec requirement)
  const raw = url.search.startsWith("?") ? url.search.slice(1) : url.search;
  if (!raw) return "";
  const pairs = raw.split("&");
  pairs.sort();
  return pairs.join("&");
}

// --- Signature verification ---

export async function verifyEd25519(
  publicKey: CryptoKey,
  signature: Uint8Array,
  data: string
): Promise<boolean> {
  return crypto.subtle.verify("Ed25519", publicKey, signature, encoder.encode(data));
}

// --- Recovery code ---

export function generateRecoveryCode(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return base64url(bytes);
}
```

- [ ] **Step 2: Verify compiles**

```bash
npx tsc --noEmit 2>&1 | grep crypto || echo "crypto.ts OK"
```

- [ ] **Step 3: Commit**

```bash
git add src/crypto.ts
git commit -m "feat: add crypto module for Ed25519, SHA-256, canonical requests"
```

---

### Task 4: Auth middleware — CHAT-ED25519 signature verification

**Files:**
- Rewrite: `src/auth.ts`

- [ ] **Step 1: Rewrite auth.ts**

```typescript
import type { Context, Next } from "hono";
import type { Env, Variables } from "./types";
import {
  base64urlDecode,
  sha256hex,
  importEd25519PublicKey,
  buildCanonicalRequest,
  buildStringToSign,
  sortedQueryString,
  verifyEd25519,
} from "./crypto";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

// Per-isolate public key cache with 5-minute TTL
const KEY_CACHE = new Map<string, { key: CryptoKey; cachedAt: number }>();
const CACHE_TTL_MS = 5 * 60 * 1000;

async function getPublicKey(db: D1Database, actor: string): Promise<CryptoKey | null> {
  const cached = KEY_CACHE.get(actor);
  if (cached && Date.now() - cached.cachedAt < CACHE_TTL_MS) {
    return cached.key;
  }

  const row = await db.prepare("SELECT public_key FROM actors WHERE actor = ?")
    .bind(actor)
    .first<{ public_key: string }>();
  if (!row) return null;

  try {
    const key = await importEd25519PublicKey(row.public_key);
    KEY_CACHE.set(actor, { key, cachedAt: Date.now() });
    return key;
  } catch {
    return null;
  }
}

export function invalidateKeyCache(actor: string): void {
  KEY_CACHE.delete(actor);
}

interface ParsedAuth {
  actor: string;
  timestamp: string;
  signature: Uint8Array;
}

function parseAuthHeader(header: string): ParsedAuth | null {
  // Format: CHAT-ED25519 Credential=u/alice, Timestamp=1710000000, Signature=<base64url>
  if (!header.startsWith("CHAT-ED25519 ")) return null;
  const rest = header.slice("CHAT-ED25519 ".length);

  let actor = "";
  let timestamp = "";
  let sigB64 = "";

  for (const part of rest.split(", ")) {
    const eq = part.indexOf("=");
    if (eq === -1) continue;
    const key = part.slice(0, eq);
    const val = part.slice(eq + 1);
    if (key === "Credential") actor = val;
    else if (key === "Timestamp") timestamp = val;
    else if (key === "Signature") sigB64 = val;
  }

  if (!actor || !timestamp || !sigB64) return null;

  try {
    return { actor, timestamp, signature: base64urlDecode(sigB64) };
  } catch {
    return null;
  }
}

const TIMESTAMP_WINDOW_S = 5 * 60; // ±5 minutes

export async function signatureAuth(c: AppContext, next: Next) {
  const authHeader = c.req.header("Authorization");
  if (!authHeader) return c.json({ error: "Unauthorized" }, 401);

  const parsed = parseAuthHeader(authHeader);
  if (!parsed) return c.json({ error: "Invalid authorization format" }, 401);

  // Timestamp check
  const ts = parseInt(parsed.timestamp, 10);
  if (isNaN(ts)) return c.json({ error: "Invalid timestamp" }, 401);
  const now = Math.floor(Date.now() / 1000);
  if (Math.abs(now - ts) > TIMESTAMP_WINDOW_S) {
    return c.json({ error: "Timestamp out of range" }, 401);
  }

  // Look up public key
  const publicKey = await getPublicKey(c.env.DB, parsed.actor);
  if (!publicKey) return c.json({ error: "Unknown actor" }, 401);

  // Reconstruct canonical request
  const url = new URL(c.req.url);
  const body = await c.req.raw.clone().text();
  const bodyHash = await sha256hex(body);
  const query = sortedQueryString(url);
  const canonical = buildCanonicalRequest(c.req.method, url.pathname, query, bodyHash);
  const canonicalHash = await sha256hex(canonical);
  const stringToSign = buildStringToSign(parsed.timestamp, parsed.actor, canonicalHash);

  // Verify signature
  const valid = await verifyEd25519(publicKey, parsed.signature, stringToSign);
  if (!valid) return c.json({ error: "Invalid signature" }, 401);

  c.set("actor", parsed.actor);
  await next();
}
```

- [ ] **Step 2: Commit**

```bash
git add src/auth.ts
git commit -m "feat: rewrite auth middleware for CHAT-ED25519 signature verification"
```

---

### Task 5: Register — Open registration with rate limiting

**Files:**
- Create: `src/register.ts`

- [ ] **Step 1: Create register.ts**

```typescript
import type { Context } from "hono";
import type { Env, Variables, RegisterRequest } from "./types";
import { isValidActor } from "./actor";
import { sha256hex, generateRecoveryCode, importEd25519PublicKey } from "./crypto";

export async function registerActor(c: Context<{ Bindings: Env; Variables: Variables }>) {
  let body: RegisterRequest;
  try {
    body = await c.req.json<RegisterRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.actor || typeof body.actor !== "string") {
    return c.json({ error: "actor is required" }, 400);
  }

  if (!isValidActor(body.actor)) {
    return c.json({ error: "Invalid actor format (use u/<name> or a/<name>, max 64 chars)" }, 400);
  }

  if (!body.public_key || typeof body.public_key !== "string") {
    return c.json({ error: "public_key is required" }, 400);
  }

  // Validate public key format by attempting import
  try {
    await importEd25519PublicKey(body.public_key);
  } catch {
    return c.json({ error: "Invalid public key format (expected base64url Ed25519 public key)" }, 400);
  }

  // Rate limit: 5 registrations per IP per hour
  const ip = c.req.header("CF-Connecting-IP") || "unknown";
  const ipHash = await sha256hex(ip);
  const oneHourAgo = Date.now() - 3600_000;

  const rateCheck = await c.env.DB.prepare(
    "SELECT COUNT(*) as count FROM actors WHERE created_ip_hash = ? AND created_at > ?"
  ).bind(ipHash, oneHourAgo).first<{ count: number }>();

  if ((rateCheck?.count ?? 0) >= 5) {
    return c.json({ error: "Rate limit exceeded (max 5 registrations per hour)" }, 429);
  }

  // Generate recovery code
  const recoveryCode = generateRecoveryCode();
  const recoveryHash = await sha256hex(recoveryCode);
  const now = Date.now();

  // Use INSERT and catch UNIQUE constraint violation for race-safe 409
  try {
    await c.env.DB.prepare(
      "INSERT INTO actors (actor, public_key, recovery_hash, created_at, created_ip_hash) VALUES (?, ?, ?, ?, ?)"
    ).bind(body.actor, body.public_key, recoveryHash, now, ipHash).run();
  } catch (e: unknown) {
    if (e instanceof Error && e.message.includes("UNIQUE")) {
      return c.json({ error: "Actor name already taken" }, 409);
    }
    throw e;
  }

  return c.json({ actor: body.actor, recovery_code: recoveryCode }, 201);
}
```

- [ ] **Step 2: Commit**

```bash
git add src/register.ts
git commit -m "feat: add open registration with IP rate limiting"
```

---

### Task 6: Keys — Rotation and account deletion

**Files:**
- Create: `src/keys.ts`

- [ ] **Step 1: Create keys.ts**

```typescript
import type { Context } from "hono";
import type { Env, Variables, RotateKeyRequest, RotateRecoveryRequest, DeleteActorRequest } from "./types";
import { sha256hex, generateRecoveryCode, importEd25519PublicKey } from "./crypto";
import { invalidateKeyCache } from "./auth";

async function verifyRecoveryCode(
  db: D1Database,
  actor: string,
  code: string
): Promise<{ valid: boolean; found: boolean }> {
  const row = await db.prepare("SELECT recovery_hash FROM actors WHERE actor = ?")
    .bind(actor)
    .first<{ recovery_hash: string }>();
  if (!row) return { valid: false, found: false };
  const hash = await sha256hex(code);
  return { valid: hash === row.recovery_hash, found: true };
}

// Rate limit: 5 failed recovery attempts per actor per hour
const ROTATION_FAIL_LIMIT = 5;
const ROTATION_WINDOW_MS = 3600_000;
// In-memory per-isolate tracking (simple approach — resets on isolate restart)
const rotationFailures = new Map<string, { count: number; firstAt: number }>();

function checkRotationRateLimit(actor: string): boolean {
  const entry = rotationFailures.get(actor);
  if (!entry) return true;
  if (Date.now() - entry.firstAt > ROTATION_WINDOW_MS) {
    rotationFailures.delete(actor);
    return true;
  }
  return entry.count < ROTATION_FAIL_LIMIT;
}

function recordRotationFailure(actor: string): void {
  const entry = rotationFailures.get(actor);
  if (!entry || Date.now() - entry.firstAt > ROTATION_WINDOW_MS) {
    rotationFailures.set(actor, { count: 1, firstAt: Date.now() });
  } else {
    entry.count++;
  }
}

function clearRotationFailures(actor: string): void {
  rotationFailures.delete(actor);
}

export async function rotateKey(c: Context<{ Bindings: Env; Variables: Variables }>) {
  let body: RotateKeyRequest;
  try {
    body = await c.req.json<RotateKeyRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.actor || !body.recovery_code || !body.new_public_key) {
    return c.json({ error: "actor, recovery_code, and new_public_key are required" }, 400);
  }

  if (!checkRotationRateLimit(body.actor)) {
    return c.json({ error: "Too many failed attempts, try again later" }, 429);
  }

  // Validate new key format
  try {
    await importEd25519PublicKey(body.new_public_key);
  } catch {
    return c.json({ error: "Invalid public key format" }, 400);
  }

  const { valid, found } = await verifyRecoveryCode(c.env.DB, body.actor, body.recovery_code);
  if (!found) return c.json({ error: "Actor not found" }, 404);
  if (!valid) {
    recordRotationFailure(body.actor);
    return c.json({ error: "Invalid recovery code" }, 401);
  }

  clearRotationFailures(body.actor);

  await c.env.DB.prepare("UPDATE actors SET public_key = ? WHERE actor = ?")
    .bind(body.new_public_key, body.actor).run();

  invalidateKeyCache(body.actor);

  return c.json({ actor: body.actor });
}

export async function rotateRecovery(c: Context<{ Bindings: Env; Variables: Variables }>) {
  let body: RotateRecoveryRequest;
  try {
    body = await c.req.json<RotateRecoveryRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.actor || !body.recovery_code) {
    return c.json({ error: "actor and recovery_code are required" }, 400);
  }

  if (!checkRotationRateLimit(body.actor)) {
    return c.json({ error: "Too many failed attempts, try again later" }, 429);
  }

  const { valid, found } = await verifyRecoveryCode(c.env.DB, body.actor, body.recovery_code);
  if (!found) return c.json({ error: "Actor not found" }, 404);
  if (!valid) {
    recordRotationFailure(body.actor);
    return c.json({ error: "Invalid recovery code" }, 401);
  }

  clearRotationFailures(body.actor);

  const newCode = generateRecoveryCode();
  const newHash = await sha256hex(newCode);

  await c.env.DB.prepare("UPDATE actors SET recovery_hash = ? WHERE actor = ?")
    .bind(newHash, body.actor).run();

  return c.json({ recovery_code: newCode });
}

export async function deleteActor(c: Context<{ Bindings: Env; Variables: Variables }>) {
  // Actor name passed in body (not URL param) because names contain "/"
  let body: DeleteActorRequest;
  try {
    body = await c.req.json<DeleteActorRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.actor || !body.recovery_code) {
    return c.json({ error: "actor and recovery_code are required" }, 400);
  }

  if (!checkRotationRateLimit(body.actor)) {
    return c.json({ error: "Too many failed attempts, try again later" }, 429);
  }

  const { valid, found } = await verifyRecoveryCode(c.env.DB, body.actor, body.recovery_code);
  if (!found) return c.json({ error: "Actor not found" }, 404);
  if (!valid) {
    recordRotationFailure(body.actor);
    return c.json({ error: "Invalid recovery code" }, 401);
  }

  await c.env.DB.prepare("DELETE FROM actors WHERE actor = ?")
    .bind(body.actor).run();

  invalidateKeyCache(body.actor);

  return c.json({ deleted: body.actor });
}
```

- [ ] **Step 2: Commit**

```bash
git add src/keys.ts
git commit -m "feat: add key rotation, recovery rotation, and account deletion"
```

---

### Task 7: Index — Wire new routes, remove old auth

**Files:**
- Modify: `src/index.ts`
- Delete: `src/token.ts`

- [ ] **Step 1: Rewrite index.ts**

```typescript
import { Hono } from "hono";
import { cors } from "hono/cors";
import { signatureAuth } from "./auth";
import { registerActor } from "./register";
import { rotateKey, rotateRecovery, deleteActor } from "./keys";
import { createChat, getChat, listChats, joinChat } from "./chat";
import { sendMessage, listMessages } from "./message";
import { landingPage } from "./landing";
import { docsPage } from "./docs";
import type { Env, Variables } from "./types";

const app = new Hono<{ Bindings: Env; Variables: Variables }>();

app.use("*", cors());

// Pages (no auth)
app.get("/", (c) => c.html(landingPage()));
app.get("/docs", (c) => c.html(docsPage()));

// Body size limit for API routes
const MAX_BODY_SIZE = 65_536;
app.use("/api/*", async (c, next) => {
  const cl = c.req.header("Content-Length");
  if (cl && parseInt(cl, 10) > MAX_BODY_SIZE) {
    return c.json({ error: "Request body too large" }, 413);
  }
  await next();
});

// Registration (no auth, rate limited internally)
app.post("/api/register", registerActor);

// Key management (no signature auth, uses recovery code)
app.post("/api/keys/rotate", rotateKey);
app.post("/api/keys/rotate-recovery", rotateRecovery);
app.post("/api/actors/delete", deleteActor);

// Chat & message routes (Ed25519 signature auth)
app.use("/api/chat/*", signatureAuth);
app.use("/api/chat", signatureAuth);

app.post("/api/chat", createChat);
app.get("/api/chat", listChats);
app.get("/api/chat/:id", getChat);
app.post("/api/chat/:id/join", joinChat);
app.post("/api/chat/:id/messages", sendMessage);
app.get("/api/chat/:id/messages", listMessages);

// 404 fallback
app.notFound((c) => c.json({ error: "Not found" }, 404));

// Error handler
app.onError((err, c) => {
  console.error("[chat-worker] unhandled error:", err);
  return c.json({ error: "Internal server error" }, 500);
});

export default {
  fetch: app.fetch,
} satisfies ExportedHandler<Env>;
```

- [ ] **Step 2: Delete token.ts**

```bash
rm src/token.ts
```

- [ ] **Step 3: Verify clean compilation**

```bash
npx tsc --noEmit
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add src/index.ts && git rm src/token.ts
git commit -m "feat: wire Ed25519 auth routes, remove HMAC token system"
```

---

### Task 8: Landing page — Agent icons + copy button

**Files:**
- Rewrite: `src/landing.ts`

- [ ] **Step 1: Rewrite landing.ts**

Replace the generic agent SVG icons with recognizable, clean monochrome icons for Claude Code (terminal/CLI icon), Cursor (cursor icon), Codex (code icon), OpenClaw (claw icon), and OpenCode (brackets icon). Each should be a distinct, identifiable shape — not generic circles.

Update the `copySetup()` function content for the new Ed25519 flow:
- Generate Ed25519 keypair
- Register at /api/register
- Sign requests with CHAT-ED25519 scheme
- Include a working code example

Fix the "Copied!" state so the button styling doesn't break (ensure the text swap doesn't change button dimensions).

Key changes:
- Agent icons: use inline SVGs with distinct silhouettes (terminal prompt `>_` for Claude Code, cursor arrow for Cursor, `</>` for Codex, claw shape for OpenClaw, brackets for OpenCode)
- Copy content: Ed25519 setup flow with actual commands
- Button: add `min-width` to prevent layout shift on "Copied!"

- [ ] **Step 2: Verify renders**

```bash
npx wrangler deploy
curl -s https://chat.go-mizu.workers.dev/ | head -5
```

- [ ] **Step 3: Commit**

```bash
git add src/landing.ts
git commit -m "feat: update landing page with agent icons and Ed25519 setup"
```

---

### Task 9: Docs page — Full-width, human-friendly, working examples

**Files:**
- Rewrite: `src/docs.ts`

- [ ] **Step 1: Rewrite docs.ts**

Complete rewrite with these changes:

**Layout:**
- Remove max-width constraint on main content (full-width with comfortable padding)
- Keep sidebar navigation but make content area fluid

**Content — human-friendly DX writing style:**
- Conversational tone: "You'll need an Ed25519 keypair. Most languages have built-in support..."
- Explain the "why" alongside the "how"
- Every code example must be copy-pasteable and work against the live API

**Sections:**
1. **Overview** — what chat.now is, endpoint table
2. **Getting Started** — step-by-step: generate keypair → register → sign first request. Include working examples in curl (with openssl for signing), Python, and TypeScript.
3. **Authentication** — the CHAT-ED25519 signing protocol explained step by step with a worked example showing each intermediate value
4. **Registration** — POST /api/register with examples
5. **Key Management** — rotation and recovery
6. **Chats** — create, list, get, join
7. **Messages** — send, list, pagination
8. **Security** — signing protocol properties, what's protected
9. **Error Codes** — reference table
10. **Account Deletion** — DELETE /api/actors/:actor

**Code examples must include:**
- curl examples for every endpoint (these are the ones we'll test)
- Python example showing keypair generation + request signing + API call
- TypeScript/Node example showing the same

- [ ] **Step 2: Commit**

```bash
git add src/docs.ts
git commit -m "feat: rewrite docs with full-width layout and working Ed25519 examples"
```

---

### Task 10: Remove AUTH_TOKEN secret

- [ ] **Step 1: Delete the Cloudflare secret**

```bash
npx wrangler secret delete AUTH_TOKEN
```

- [ ] **Step 2: Remove from .local.env**

Remove the `MIZU_CHAT_API_TOKEN` line from `$HOME/data/.local.env`.

- [ ] **Step 3: Commit any remaining changes**

```bash
git add -A
git commit -m "chore: remove AUTH_TOKEN secret and MIZU_CHAT_API_TOKEN"
```

---

### Task 11: Deploy and run full test suite

- [ ] **Step 1: Run D1 migration**

```bash
npx wrangler d1 execute chat-db --remote --file=schema.sql
```

- [ ] **Step 2: Deploy**

```bash
npx wrangler deploy
```

- [ ] **Step 3: Test registration**

```bash
# Generate Ed25519 keypair using openssl
openssl genpkey -algorithm Ed25519 -out /tmp/chat_test_key.pem
openssl pkey -in /tmp/chat_test_key.pem -pubout -out /tmp/chat_test_pub.pem

# Extract raw public key as base64url
PUBKEY=$(openssl pkey -in /tmp/chat_test_pub.pem -pubin -outform DER | tail -c 32 | base64 | tr '+/' '-_' | tr -d '=')

# Register
curl -s -X POST https://chat.go-mizu.workers.dev/api/register \
  -H "Content-Type: application/json" \
  -d "{\"actor\":\"u/testplan\",\"public_key\":\"$PUBKEY\"}"
# Expected: 201 with actor + recovery_code
```

- [ ] **Step 4: Test signed request**

Write a small script that:
1. Builds canonical request for GET /api/chat
2. Builds string-to-sign
3. Signs with Ed25519 private key
4. Sends request with CHAT-ED25519 Authorization header
5. Expects 200 with chat list

- [ ] **Step 5: Test duplicate registration → 409**

```bash
curl -s -X POST https://chat.go-mizu.workers.dev/api/register \
  -H "Content-Type: application/json" \
  -d "{\"actor\":\"u/testplan\",\"public_key\":\"$PUBKEY\"}"
# Expected: 409
```

- [ ] **Step 6: Test invalid signature → 401**

```bash
curl -s https://chat.go-mizu.workers.dev/api/chat \
  -H "Authorization: CHAT-ED25519 Credential=u/testplan, Timestamp=$(date +%s), Signature=invalidsig"
# Expected: 401
```

- [ ] **Step 7: Test expired timestamp → 401**

```bash
# Use timestamp from 10 minutes ago
OLD_TS=$(($(date +%s) - 600))
# Sign with correct key but old timestamp, expect 401
```

- [ ] **Step 8: Test chat create + join + send + read flow**

Using signed requests:
1. POST /api/chat — create room
2. POST /api/chat/:id/join — join (with second actor)
3. POST /api/chat/:id/messages — send message
4. GET /api/chat/:id/messages — read messages
5. Verify non-member gets 403 on send

- [ ] **Step 9: Test key rotation**

```bash
# Generate new keypair
# POST /api/keys/rotate with recovery_code + new_public_key
# Verify old key stops working
# Verify new key works
```

- [ ] **Step 10: Test recovery rotation**

```bash
# POST /api/keys/rotate-recovery with recovery_code
# Get new recovery_code
# Verify old recovery_code no longer works for rotation
# Verify new recovery_code works
```

- [ ] **Step 11: Test account deletion**

```bash
# DELETE /api/actors/u/testplan with recovery_code
# Verify actor no longer authenticates
# Verify name can be re-registered
```

- [ ] **Step 12: Test landing page and docs**

```bash
curl -s -o /dev/null -w "%{http_code}" https://chat.go-mizu.workers.dev/
# Expected: 200

curl -s -o /dev/null -w "%{http_code}" https://chat.go-mizu.workers.dev/docs
# Expected: 200
```

- [ ] **Step 13: Commit test results / final adjustments**

```bash
git add -A
git commit -m "chore: deploy and verify Ed25519 auth end-to-end"
```

---

### Task 12: Update security spec

**Files:**
- Modify: `blueprints/now/spec/0741_chat_worker_security.md`

- [ ] **Step 1: Update security spec**

Replace all X-Actor and HMAC token references with the new Ed25519 public-key auth model. Update:
- Identity model section
- Permission matrix
- Threat model (add key compromise, recovery code attacks)
- Security controls (add signature verification, rate limiting)

- [ ] **Step 2: Commit**

```bash
git add blueprints/now/spec/0741_chat_worker_security.md
git commit -m "spec: update security doc for Ed25519 public-key auth model"
```
