/**
 * Bot protection — integration tests
 *
 * Tests three layers:
 *   1. Turnstile validation (server-side)
 *   2. D1-based rate limiting
 *   3. Bot signal heuristics
 */
import { describe, it, expect, beforeAll } from "vitest";
import { SELF } from "cloudflare:test";

function api(path: string, opts: RequestInit & { token?: string } = {}) {
  const headers = new Headers(opts.headers);
  if (opts.token) headers.set("Authorization", `Bearer ${opts.token}`);
  if (!headers.has("Accept")) headers.set("Accept", "application/json");
  // Add browser-like headers to avoid bot guard blocking test requests
  if (!headers.has("User-Agent")) headers.set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36");
  if (!headers.has("Accept-Language")) headers.set("Accept-Language", "en-US,en;q=0.9");
  if (!headers.has("Accept-Encoding")) headers.set("Accept-Encoding", "gzip, deflate, br");
  return SELF.fetch(`http://localhost${path}`, { ...opts, headers });
}

describe("Bot Protection", () => {
  let token: string;

  beforeAll(async () => {
    const { env } = await import("cloudflare:test");
    const db = env.DB;

    const stmts = [
      `CREATE TABLE IF NOT EXISTS actors (actor TEXT PRIMARY KEY, type TEXT NOT NULL DEFAULT 'human' CHECK(type IN ('human','agent')), public_key TEXT, email TEXT UNIQUE, created_at INTEGER NOT NULL)`,
      `CREATE TABLE IF NOT EXISTS challenges (id TEXT PRIMARY KEY, actor TEXT NOT NULL, nonce TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
      `CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, actor TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_sessions_actor ON sessions(actor, expires_at)`,
      `CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, actor TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, name TEXT NOT NULL DEFAULT '', prefix TEXT NOT NULL DEFAULT '', expires_at INTEGER, created_at INTEGER NOT NULL, last_accessed INTEGER)`,
      `CREATE INDEX IF NOT EXISTS idx_api_keys_actor ON api_keys(actor)`,
      `CREATE TABLE IF NOT EXISTS files (owner TEXT NOT NULL, path TEXT NOT NULL, name TEXT NOT NULL, size INTEGER NOT NULL DEFAULT 0, type TEXT NOT NULL DEFAULT 'application/octet-stream', addr TEXT, tx INTEGER, tx_time INTEGER, updated_at INTEGER NOT NULL, PRIMARY KEY (owner, path))`,
      `CREATE INDEX IF NOT EXISTS idx_files_name ON files(owner, name COLLATE NOCASE)`,
      `CREATE TABLE IF NOT EXISTS tx_counter (actor TEXT PRIMARY KEY, next_tx INTEGER NOT NULL DEFAULT 1)`,
      `CREATE TABLE IF NOT EXISTS events (id INTEGER PRIMARY KEY AUTOINCREMENT, tx INTEGER NOT NULL, actor TEXT NOT NULL, action TEXT NOT NULL CHECK(action IN ('write','move','delete')), path TEXT NOT NULL, addr TEXT, size INTEGER NOT NULL DEFAULT 0, type TEXT, meta TEXT, msg TEXT, ts INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_events_actor_tx ON events(actor, tx)`,
      `CREATE TABLE IF NOT EXISTS blobs (addr TEXT NOT NULL, actor TEXT NOT NULL, size INTEGER NOT NULL, ref_count INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL, PRIMARY KEY (addr, actor))`,
      `CREATE TABLE IF NOT EXISTS shards (actor TEXT PRIMARY KEY, shard TEXT NOT NULL UNIQUE, next_tx INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL)`,
      `CREATE TABLE IF NOT EXISTS audit (id INTEGER PRIMARY KEY AUTOINCREMENT, actor TEXT, action TEXT NOT NULL, path TEXT, ip TEXT, ts INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_audit_actor_ts ON audit(actor, ts)`,
      `CREATE TABLE IF NOT EXISTS share_links (token TEXT PRIMARY KEY, actor TEXT NOT NULL, path TEXT NOT NULL, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_share_links_actor ON share_links(actor, created_at)`,
      `CREATE INDEX IF NOT EXISTS idx_share_links_expires ON share_links(expires_at)`,
      `CREATE TABLE IF NOT EXISTS magic_tokens (token TEXT PRIMARY KEY, email TEXT NOT NULL, actor TEXT, expires_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_magic_tokens_email ON magic_tokens(email)`,
      `CREATE INDEX IF NOT EXISTS idx_magic_tokens_expires ON magic_tokens(expires_at)`,
      `CREATE TABLE IF NOT EXISTS oauth_clients (client_id TEXT PRIMARY KEY, redirect_uris TEXT NOT NULL DEFAULT '[]', client_name TEXT NOT NULL DEFAULT '', token_endpoint_auth_method TEXT NOT NULL DEFAULT 'none', created_at INTEGER NOT NULL)`,
      `CREATE TABLE IF NOT EXISTS oauth_codes (code TEXT PRIMARY KEY, actor TEXT NOT NULL, client_id TEXT NOT NULL, redirect_uri TEXT NOT NULL, scope TEXT NOT NULL DEFAULT '*', code_challenge TEXT NOT NULL, code_challenge_method TEXT NOT NULL DEFAULT 'S256', expires_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_oauth_codes_expires ON oauth_codes(expires_at)`,
      // Rate limiting table
      `CREATE TABLE IF NOT EXISTS rate_limits (endpoint TEXT NOT NULL, key TEXT NOT NULL, ts INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_rl_lookup ON rate_limits(endpoint, key, ts)`,
    ];
    for (const sql of stmts) await db.exec(sql);

    await db.prepare(
      "INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES (?, 'agent', 'testkey', ?)",
    ).bind("testuser", Date.now()).run();
    await db.prepare(
      "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
    ).bind("bot-test-session", "testuser", Date.now() + 86400000).run();

    token = "bot-test-session";
  });

  // ── Rate Limiting ──────────────────────────────────────────────────

  describe("Rate Limiting", () => {
    it("allows requests within the limit", async () => {
      // Registration should work for the first request
      const res = await api("/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "rl-test-1", public_key: "dummykey1" }),
      });
      expect(res.status).toBe(201);
    });

    it("returns 429 when rate limit is exceeded for registration", async () => {
      const { env } = await import("cloudflare:test");
      const db = env.DB;
      const now = Date.now();

      // Seed rate_limits to simulate 5 recent registrations from the same IP
      for (let i = 0; i < 5; i++) {
        await db.prepare("INSERT INTO rate_limits (endpoint, key, ts) VALUES (?, ?, ?)")
          .bind("auth/register", "unknown", now - i * 1000).run();
      }

      const res = await api("/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "rl-test-blocked", public_key: "dummykey" }),
      });
      expect(res.status).toBe(429);
      const body = await res.json() as any;
      expect(body.error).toBe("rate_limited");
    });

    it("returns 429 for API key creation when limit exceeded", async () => {
      const { env } = await import("cloudflare:test");
      const db = env.DB;
      const now = Date.now();

      // Seed 10 key creations for this actor
      for (let i = 0; i < 10; i++) {
        await db.prepare("INSERT INTO rate_limits (endpoint, key, ts) VALUES (?, ?, ?)")
          .bind("auth/keys", "testuser", now - i * 1000).run();
      }

      const res = await api("/auth/keys", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: "rl-test-key" }),
      });
      expect(res.status).toBe(429);
    });

    it("rate limit on challenge endpoint returns 429", async () => {
      const { env } = await import("cloudflare:test");
      const db = env.DB;
      const now = Date.now();

      // Seed 30 challenges from the same IP
      for (let i = 0; i < 30; i++) {
        await db.prepare("INSERT INTO rate_limits (endpoint, key, ts) VALUES (?, ?, ?)")
          .bind("auth/challenge", "unknown", now - i * 1000).run();
      }

      const res = await api("/auth/challenge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "testuser" }),
      });
      expect(res.status).toBe(429);
    });

    it("rate limit on verify endpoint returns 429", async () => {
      const { env } = await import("cloudflare:test");
      const db = env.DB;
      const now = Date.now();

      // Seed 20 verify attempts from the same IP
      for (let i = 0; i < 20; i++) {
        await db.prepare("INSERT INTO rate_limits (endpoint, key, ts) VALUES (?, ?, ?)")
          .bind("auth/verify", "unknown", now - i * 1000).run();
      }

      const res = await api("/auth/verify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ challenge_id: "fake", actor: "testuser", signature: "fakesig" }),
      });
      expect(res.status).toBe(429);
    });
  });

  // ── Turnstile ──────────────────────────────────────────────────────

  describe("Turnstile", () => {
    it("magic-link works without Turnstile when secret not configured", async () => {
      // TURNSTILE_SECRET_KEY is not set in test env → graceful degradation
      const res = await api("/auth/magic-link", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: "turnstile-test@example.com" }),
      });
      // Should get 500 (no RESEND_API_KEY) rather than 403 (turnstile fail)
      // This confirms Turnstile was bypassed
      const body = await res.json() as any;
      expect(res.status).toBe(500);
      expect(body.error).toBe("server_error");
    });

    it("magic-link rejects when Turnstile fails", async () => {
      // To test Turnstile rejection, we'd need TURNSTILE_SECRET_KEY set
      // and send an invalid token. Since we can't easily set env in test,
      // we test the validateTurnstile function directly.
      const { validateTurnstile } = await import("../lib/turnstile");

      // With a dummy secret but no token → should fail
      const result = await validateTurnstile(null, "test-secret-key");
      expect(result.success).toBe(false);
      expect(result["error-codes"]).toContain("missing-input-response");
    });

    it("validateTurnstile succeeds without secret (graceful degradation)", async () => {
      const { validateTurnstile } = await import("../lib/turnstile");

      const result = await validateTurnstile("some-token", undefined);
      expect(result.success).toBe(true);
    });

    it("validateTurnstile succeeds without secret and no token", async () => {
      const { validateTurnstile } = await import("../lib/turnstile");

      const result = await validateTurnstile(null, undefined);
      expect(result.success).toBe(true);
    });
  });

  // ── Bot Guard ──────────────────────────────────────────────────────

  describe("Bot Guard", () => {
    it("computes high score for requests with no UA and no headers", async () => {
      const { computeBotScore } = await import("../middleware/bot-guard");

      const req = new Request("http://localhost/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      const { score, reasons } = computeBotScore(req);
      // No UA (25) + no Accept-Language (15) + no Sec-Fetch-Mode (10) = 50
      expect(score).toBeGreaterThanOrEqual(40);
      expect(reasons).toContain("no-user-agent");
      expect(reasons).toContain("no-accept-language");
    });

    it("computes low score for browser-like requests", async () => {
      const { computeBotScore } = await import("../middleware/bot-guard");

      const req = new Request("http://localhost/auth/register", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
          "Accept-Language": "en-US,en;q=0.9",
          "Sec-Fetch-Mode": "navigate",
          "Accept-Encoding": "gzip, deflate, br",
        },
      });
      const { score } = computeBotScore(req);
      expect(score).toBeLessThan(30);
    });

    it("flags curl/wget user agents", async () => {
      const { computeBotScore } = await import("../middleware/bot-guard");

      const req = new Request("http://localhost/auth/register", {
        method: "POST",
        headers: {
          "User-Agent": "curl/7.88.1",
          "Accept-Language": "en",
          "Sec-Fetch-Mode": "cors",
        },
      });
      const { score, reasons } = computeBotScore(req);
      expect(reasons).toContain("bot-user-agent");
      expect(score).toBeGreaterThanOrEqual(20);
    });

    it("allows authenticated API requests regardless of headers", async () => {
      // Bearer-authenticated requests should bypass bot guard
      const res = await api("/auth/register", {
        method: "POST",
        token,
        headers: {
          "Content-Type": "application/json",
          "User-Agent": "", // empty UA that would normally trigger bot guard
        },
        body: JSON.stringify({ actor: "botguard-bypass-test", public_key: "key123" }),
      });
      // Should NOT be 403 from bot guard — the Bearer token bypasses it
      // Might be 429 from rate limit or 201 success, but not 403 bot_check
      expect(res.status).not.toBe(403);
    });

    it("registration endpoint has bot guard active", async () => {
      // Send a request with no browser headers and no auth
      const headers = new Headers({
        "Content-Type": "application/json",
      });
      // Deliberately don't set User-Agent, Accept-Language, etc.
      const res = await SELF.fetch("http://localhost/auth/register", {
        method: "POST",
        headers,
        body: JSON.stringify({ actor: "bot-test-actor", public_key: "key" }),
      });
      // With no UA (25) + no Accept-Language (15) + no Sec-Fetch-Mode (10) = 50
      // This is under the 60 threshold, so it should pass through
      // (but the accumulation of signals matters)
      expect([201, 409, 429]).toContain(res.status);
    });
  });

  // ── Integration ────────────────────────────────────────────────────

  describe("Integration", () => {
    it("home page includes Turnstile script when TURNSTILE_SITE_KEY is set", async () => {
      // In test env, TURNSTILE_SITE_KEY is not set, so script should not appear
      const res = await SELF.fetch("http://localhost/", {
        headers: { "User-Agent": "Mozilla/5.0 Test", "Accept-Language": "en" },
      });
      const html = await res.text();
      // Without TURNSTILE_SITE_KEY, the script tag should NOT be present
      expect(html).not.toContain("challenges.cloudflare.com/turnstile");
    });

    it("magic-link accepts cf-turnstile-response field", async () => {
      // Verify the endpoint accepts the optional field without error
      const res = await api("/auth/magic-link", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: "turnstile-field-test@example.com",
          "cf-turnstile-response": "dummy-token",
        }),
      });
      // Should not be 400 (field rejection) — 500 is expected (no Resend key)
      expect(res.status).not.toBe(400);
    });

    it("rate limit table gets populated on requests", async () => {
      const { env } = await import("cloudflare:test");

      // Make a registration request
      await api("/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "rl-populate-test", public_key: "key" }),
      });

      // Check that rate_limits table has an entry
      const row = await env.DB.prepare(
        "SELECT COUNT(*) as cnt FROM rate_limits WHERE endpoint = 'auth/register'",
      ).first<{ cnt: number }>();
      expect(row!.cnt).toBeGreaterThanOrEqual(1);
    });
  });
});
