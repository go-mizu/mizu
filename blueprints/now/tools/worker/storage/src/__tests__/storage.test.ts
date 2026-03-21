/**
 * Storage API — integration tests
 *
 * Tests the /files/* protocol:
 *   /files          (list, with ?prefix=)
 *   /files/{path}   (GET=download redirect, HEAD=metadata, DELETE=delete)
 *   /files/search   (search)
 *   /files/stats    (usage)
 *   /files/move     (move/rename)
 *   /files/share    (create share link)
 *   /files/uploads  (presigned upload initiation)
 *   /files/uploads/complete (confirm upload)
 *   /s/:token       (access shared file)
 *   /auth/*         (register, challenge, verify, logout)
 *   /auth/keys/*    (API key management)
 *   /mcp            (MCP JSON-RPC)
 */
import { describe, it, expect, beforeAll } from "vitest";
import { SELF } from "cloudflare:test";

// Helper to make requests with a bearer token
function api(path: string, opts: RequestInit & { token?: string } = {}) {
  const headers = new Headers(opts.headers);
  if (opts.token) headers.set("Authorization", `Bearer ${opts.token}`);
  // Always request JSON for API calls (GET /files/{path} uses Accept to decide redirect vs JSON)
  if (!headers.has("Accept")) headers.set("Accept", "application/json");
  return SELF.fetch(`http://localhost${path}`, { ...opts, headers });
}

// Helper to seed files through the engine (handles sharding automatically)
async function seedFile(actor: string, path: string, content: string, contentType = "text/plain") {
  const { env } = await import("cloudflare:test");
  const { D1Engine } = await import("../storage/d1_driver");
  const engine = new D1Engine({
    db: env.DB,
    bucket: env.BUCKET,
    r2Endpoint: "https://test.r2.cloudflarestorage.com",
    r2AccessKeyId: "test-key-id",
    r2SecretAccessKey: "test-secret-key",
  });
  const data = new TextEncoder().encode(content);
  await engine.write(actor, path, data.buffer as ArrayBuffer, contentType);
}

describe("Storage API", () => {
  let token: string;

  // ── Auth flow ──────────────────────────────────────────────────────
  describe("Auth", () => {
    it("POST /auth/register creates an actor", async () => {
      const keyPair = await crypto.subtle.generateKey("Ed25519" as any, true, ["sign", "verify"]) as CryptoKeyPair;
      const rawKey = await crypto.subtle.exportKey("raw", keyPair.publicKey) as ArrayBuffer;
      const pubKeyB64 = btoa(String.fromCharCode(...new Uint8Array(rawKey)))
        .replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");

      const res = await api("/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "newuser", type: "agent", public_key: pubKeyB64 }),
      });
      expect(res.status).toBe(201);
      const body = await res.json() as any;
      expect(body.actor).toBe("newuser");
      expect(body.type).toBe("agent");
    });

    it("POST /auth/register rejects duplicate actor", async () => {
      const res = await api("/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "newuser", public_key: "dummykey" }),
      });
      expect(res.status).toBe(409);
    });

    it("POST /auth/challenge returns a nonce", async () => {
      const res = await api("/auth/challenge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "testuser" }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.challenge_id).toBeDefined();
      expect(body.nonce).toBeDefined();
      expect(body.expires_at).toBeGreaterThan(Date.now());
    });

    it("POST /auth/challenge returns 404 for unknown actor", async () => {
      const res = await api("/auth/challenge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "nonexistent" }),
      });
      expect(res.status).toBe(404);
    });

    it("POST /auth/logout works", async () => {
      const res = await api("/auth/logout", { method: "POST" });
      expect(res.status).toBe(200);
    });
  });

  // ── Setup: apply schema + seed test data ──────────────────────────
  beforeAll(async () => {
    const { env } = await import("cloudflare:test");
    const db = env.DB;

    const stmts = [
      // Auth & identity
      `CREATE TABLE IF NOT EXISTS actors (actor TEXT PRIMARY KEY, type TEXT NOT NULL DEFAULT 'human' CHECK(type IN ('human','agent')), public_key TEXT, email TEXT UNIQUE, created_at INTEGER NOT NULL)`,
      `CREATE TABLE IF NOT EXISTS challenges (id TEXT PRIMARY KEY, actor TEXT NOT NULL, nonce TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
      `CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, actor TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_sessions_actor ON sessions(actor, expires_at)`,
      `CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, actor TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, name TEXT NOT NULL DEFAULT '', prefix TEXT NOT NULL DEFAULT '', expires_at INTEGER, created_at INTEGER NOT NULL, last_accessed INTEGER)`,
      `CREATE INDEX IF NOT EXISTS idx_api_keys_actor ON api_keys(actor)`,
      // Storage — legacy shared tables (used for migration into per-actor shards)
      `CREATE TABLE IF NOT EXISTS files (owner TEXT NOT NULL, path TEXT NOT NULL, name TEXT NOT NULL, size INTEGER NOT NULL DEFAULT 0, type TEXT NOT NULL DEFAULT 'application/octet-stream', addr TEXT, tx INTEGER, tx_time INTEGER, updated_at INTEGER NOT NULL, PRIMARY KEY (owner, path))`,
      `CREATE INDEX IF NOT EXISTS idx_files_name ON files(owner, name COLLATE NOCASE)`,
      `CREATE TABLE IF NOT EXISTS tx_counter (actor TEXT PRIMARY KEY, next_tx INTEGER NOT NULL DEFAULT 1)`,
      `CREATE TABLE IF NOT EXISTS events (id INTEGER PRIMARY KEY AUTOINCREMENT, tx INTEGER NOT NULL, actor TEXT NOT NULL, action TEXT NOT NULL CHECK(action IN ('write','move','delete')), path TEXT NOT NULL, addr TEXT, size INTEGER NOT NULL DEFAULT 0, type TEXT, meta TEXT, msg TEXT, ts INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_events_actor_tx ON events(actor, tx)`,
      `CREATE TABLE IF NOT EXISTS blobs (addr TEXT NOT NULL, actor TEXT NOT NULL, size INTEGER NOT NULL, ref_count INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL, PRIMARY KEY (addr, actor))`,
      // Per-actor shard registry
      `CREATE TABLE IF NOT EXISTS shards (actor TEXT PRIMARY KEY, shard TEXT NOT NULL UNIQUE, next_tx INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL)`,
      // Audit & sharing
      `CREATE TABLE IF NOT EXISTS audit (id INTEGER PRIMARY KEY AUTOINCREMENT, actor TEXT, action TEXT NOT NULL, path TEXT, ip TEXT, ts INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_audit_actor_ts ON audit(actor, ts)`,
      `CREATE TABLE IF NOT EXISTS share_links (token TEXT PRIMARY KEY, actor TEXT NOT NULL, path TEXT NOT NULL, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_share_links_actor ON share_links(actor, created_at)`,
      `CREATE INDEX IF NOT EXISTS idx_share_links_expires ON share_links(expires_at)`,
      // OAuth
      `CREATE TABLE IF NOT EXISTS oauth_clients (client_id TEXT PRIMARY KEY, redirect_uris TEXT NOT NULL DEFAULT '[]', client_name TEXT NOT NULL DEFAULT '', token_endpoint_auth_method TEXT NOT NULL DEFAULT 'none', created_at INTEGER NOT NULL)`,
      `CREATE TABLE IF NOT EXISTS oauth_codes (code TEXT PRIMARY KEY, actor TEXT NOT NULL, client_id TEXT NOT NULL, redirect_uri TEXT NOT NULL, scope TEXT NOT NULL DEFAULT '*', code_challenge TEXT NOT NULL, code_challenge_method TEXT NOT NULL DEFAULT 'S256', expires_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_oauth_codes_expires ON oauth_codes(expires_at)`,
      // Rate limiting (bot protection)
      `CREATE TABLE IF NOT EXISTS rate_limits (endpoint TEXT NOT NULL, key TEXT NOT NULL, ts INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_rl_lookup ON rate_limits(endpoint, key, ts)`,
      // Magic link tokens
      `CREATE TABLE IF NOT EXISTS magic_tokens (token TEXT PRIMARY KEY, email TEXT NOT NULL, actor TEXT, expires_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_magic_tokens_email ON magic_tokens(email)`,
      `CREATE INDEX IF NOT EXISTS idx_magic_tokens_expires ON magic_tokens(expires_at)`,
    ];
    for (const sql of stmts) await db.exec(sql);

    await db.prepare(
      "INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES (?, 'agent', 'testkey', ?)",
    ).bind("testuser", Date.now()).run();
    await db.prepare(
      "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
    ).bind("test-session-token", "testuser", Date.now() + 86400000).run();

    token = "test-session-token";

    await seedFile("testuser", "hello.txt", "Hello, world!", "text/plain");
    await seedFile("testuser", "docs/readme.md", "# README", "text/markdown");
    await seedFile("testuser", "taocp/vol_1/1.2/1.md", "# TAOCP Volume 1, Section 1.2, Problem 1", "text/markdown");
  });

  // ── Files (/files/*) ──────────────────────────────────────────────
  describe("Files", () => {
    it("HEAD /files/{path} returns metadata headers", async () => {
      const res = await api("/files/hello.txt", { method: "HEAD", token });
      expect(res.status).toBe(200);
      expect(res.headers.get("Content-Length")).toBe("13");
      expect(res.headers.get("X-Tx")).toBeDefined();
    });

    it("HEAD /files/{path} returns 404 for missing file", async () => {
      const res = await api("/files/nonexistent.txt", { method: "HEAD", token });
      expect(res.status).toBe(404);
    });

    it("GET /files/{path} returns presigned URL as JSON", async () => {
      const res = await api("/files/hello.txt", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.url).toContain("r2.cloudflarestorage.com");
      expect(body.url).toContain("X-Amz-Signature");
      expect(body.size).toBe(13);
      expect(body.type).toBe("text/plain");
    });

    it("GET /files/{path} returns 404 for missing file", async () => {
      const res = await api("/files/nonexistent.txt", { token });
      expect(res.status).toBe(404);
    });

    it("DELETE /files/{path} deletes a single file", async () => {
      await seedFile("testuser", "to-delete.txt", "delete me");

      const res = await api("/files/to-delete.txt", { method: "DELETE", token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.deleted).toBe(1);

      const check = await api("/files/to-delete.txt", { method: "HEAD", token });
      expect(check.status).toBe(404);
    });

    it("DELETE /files/{path}/ deletes directory recursively", async () => {
      await seedFile("testuser", "tmp/a.txt", "a");
      await seedFile("testuser", "tmp/b.txt", "b");

      const res = await api("/files/tmp/", { method: "DELETE", token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.deleted).toBeGreaterThanOrEqual(2);
    });

    it("rejects requests without auth", async () => {
      const res = await api("/files/hello.txt", { method: "HEAD" });
      expect(res.status).toBe(401);
    });
  });

  // ── Uploads ────────────────────────────────────────────────────────
  describe("Uploads", () => {
    it("POST /files/uploads returns a presigned PUT URL", async () => {
      const res = await api("/files/uploads", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: "presign-test.txt", content_type: "text/plain" }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.url).toContain("r2.cloudflarestorage.com");
      expect(body.url).toContain("X-Amz-Signature");
      expect(body.content_type).toBe("text/plain");
      expect(body.expires_in).toBe(3600);
    });

    it("POST /files/uploads/complete verifies upload in R2", async () => {
      const { env } = await import("cloudflare:test");
      await env.BUCKET.put("testuser/presign-complete-test.txt", "hello presign", {
        httpMetadata: { contentType: "text/plain" },
      });

      const res = await api("/files/uploads/complete", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: "presign-complete-test.txt" }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.path).toBe("presign-complete-test.txt");
      expect(body.name).toBe("presign-complete-test.txt");
      expect(body.size).toBe(13);
    });

    it("POST /files/uploads rejects without auth", async () => {
      const res = await api("/files/uploads", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: "test.txt" }),
      });
      expect(res.status).toBe(401);
    });

    it("POST /files/uploads/complete returns 404 for missing upload", async () => {
      const res = await api("/files/uploads/complete", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: "never-uploaded.txt" }),
      });
      expect(res.status).toBe(404);
    });
  });

  // ── Listing ────────────────────────────────────────────────────────
  describe("Listing", () => {
    it("GET /files returns root entries", async () => {
      const res = await api("/files", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.entries).toBeInstanceOf(Array);
      expect(body.entries.length).toBeGreaterThan(0);
    });

    it("GET /files?prefix= lists directory contents", async () => {
      const res = await api("/files?prefix=docs/", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      const names = body.entries.map((e: any) => e.name);
      expect(names).toContain("readme.md");
    });

    it("GET /files supports limit and offset", async () => {
      const res = await api("/files?limit=1", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.entries.length).toBeLessThanOrEqual(1);
    });

    it("ls at each level of deeply nested path works", async () => {
      const r1 = await api("/files", { token });
      const b1 = await r1.json() as any;
      expect(b1.entries.map((e: any) => e.name)).toContain("taocp/");

      const r2 = await api("/files?prefix=taocp/", { token });
      const b2 = await r2.json() as any;
      expect(b2.entries.map((e: any) => e.name)).toContain("vol_1/");

      const r3 = await api("/files?prefix=taocp/vol_1/", { token });
      const b3 = await r3.json() as any;
      expect(b3.entries.map((e: any) => e.name)).toContain("1.2/");

      const r4 = await api("/files?prefix=taocp/vol_1/1.2/", { token });
      const b4 = await r4.json() as any;
      expect(b4.entries.map((e: any) => e.name)).toContain("1.md");
    });
  });

  // ── Search ─────────────────────────────────────────────────────────
  describe("Search", () => {
    it("GET /files/search?q= finds files by name", async () => {
      const res = await api("/files/search?q=hello", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.results.length).toBeGreaterThan(0);
      expect(body.results[0].name).toContain("hello");
    });

    it("GET /files/search?q= returns empty for no match", async () => {
      const res = await api("/files/search?q=zzzznonexistent", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.results.length).toBe(0);
    });

    it("GET /files/search requires q parameter", async () => {
      const res = await api("/files/search", { token });
      expect(res.status).toBe(400);
    });
  });

  // ── Move ───────────────────────────────────────────────────────────
  describe("Move", () => {
    it("POST /files/move moves a file", async () => {
      await seedFile("testuser", "moveme.txt", "move me");

      const res = await api("/files/move", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ from: "moveme.txt", to: "moved.txt" }),
      });
      expect(res.status).toBe(200);

      const old = await api("/files/moveme.txt", { method: "HEAD", token });
      expect(old.status).toBe(404);

      const newFile = await api("/files/moved.txt", { method: "HEAD", token });
      expect(newFile.status).toBe(200);
    });

    it("POST /files/move returns 404 for nonexistent source", async () => {
      const res = await api("/files/move", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ from: "nope.txt", to: "dest.txt" }),
      });
      expect(res.status).toBe(404);
    });
  });

  // ── Stats ──────────────────────────────────────────────────────────
  describe("Stats", () => {
    it("GET /files/stats returns usage", async () => {
      const res = await api("/files/stats", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.files).toBeGreaterThan(0);
      expect(body.bytes).toBeGreaterThanOrEqual(0);
    });
  });

  // ── Sharing ────────────────────────────────────────────────────────
  describe("Sharing", () => {
    it("POST /files/share creates a share URL", async () => {
      const res = await api("/files/share", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: "hello.txt", ttl: 3600 }),
      });
      expect(res.status).toBe(201);
      const body = await res.json() as any;
      expect(body.url).toContain("/s/");
      expect(body.token).toBeDefined();
      expect(body.expires_at).toBeGreaterThan(Date.now());
    });

    it("GET /s/:token redirects to presigned R2 URL", async () => {
      const shareRes = await api("/files/share", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: "hello.txt" }),
      });
      const { token: shareToken } = await shareRes.json() as any;

      const res = await api(`/s/${shareToken}`, { redirect: "manual" });
      expect(res.status).toBe(302);
      const location = res.headers.get("Location")!;
      expect(location).toContain("r2.cloudflarestorage.com");
      // Content-addressed: URL contains blob hash key, not the file name
      expect(location).toContain("blobs/testuser/");
      expect(location).toContain("X-Amz-Signature");
    });

    it("POST /files/share returns 404 for nonexistent file", async () => {
      const res = await api("/files/share", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: "nonexistent.txt" }),
      });
      expect(res.status).toBe(404);
    });
  });

  // ── API Keys ───────────────────────────────────────────────────────
  describe("API Keys", () => {
    let keyToken: string;
    let keyId: string;

    it("POST /auth/keys creates a key", async () => {
      const res = await api("/auth/keys", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: "test-key" }),
      });
      expect(res.status).toBe(201);
      const body = await res.json() as any;
      expect(body.token).toBeDefined();
      expect(body.id).toBeDefined();
      keyToken = body.token;
      keyId = body.id;
    });

    it("API key can access files via HEAD", async () => {
      const res = await api("/files/hello.txt", { method: "HEAD", token: keyToken });
      expect(res.status).toBe(200);
    });

    it("GET /auth/keys lists keys", async () => {
      const res = await api("/auth/keys", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.keys.length).toBeGreaterThan(0);
    });

    it("DELETE /auth/keys/:id revokes a key", async () => {
      const res = await api(`/auth/keys/${keyId}`, { method: "DELETE", token });
      expect(res.status).toBe(200);

      const check = await api("/files/hello.txt", { method: "HEAD", token: keyToken });
      expect(check.status).toBe(401);
    });
  });

  // ── MCP ────────────────────────────────────────────────────────────
  describe("MCP", () => {
    it("GET /mcp returns transport info", async () => {
      const res = await api("/mcp", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.server).toBe("Storage");
    });

    it("POST /mcp initialize returns capabilities", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize" }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.result.serverInfo.name).toBe("Storage");
      expect(body.result.capabilities.tools).toBeDefined();
    });

    it("POST /mcp tools/list returns 8 tools", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list" }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.result.tools.length).toBe(8);
    });

    it("POST /mcp tools/call storage_list works", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ jsonrpc: "2.0", id: 3, method: "tools/call", params: { name: "storage_list", arguments: {} } }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.result.isError).toBeFalsy();
    });

    it("POST /mcp tools/call storage_stats works", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ jsonrpc: "2.0", id: 4, method: "tools/call", params: { name: "storage_stats", arguments: {} } }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.result.isError).toBeFalsy();
      const text = body.result.content[0].text;
      expect(text).toContain("file(s)");
    });

    it("POST /mcp storage_list with 'path' param (LLM alias for prefix)", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ jsonrpc: "2.0", id: 10, method: "tools/call", params: { name: "storage_list", arguments: { path: "taocp/" } } }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.result.isError).toBeFalsy();
      const text = body.result.content[0].text;
      expect(text).toContain("vol_1/");
    });

    it("POST /mcp storage_list with leading slash prefix", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ jsonrpc: "2.0", id: 11, method: "tools/call", params: { name: "storage_list", arguments: { prefix: "/taocp/" } } }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.result.isError).toBeFalsy();
      const text = body.result.content[0].text;
      expect(text).toContain("vol_1/");
    });

    it("POST /mcp storage_read with leading slash path", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ jsonrpc: "2.0", id: 12, method: "tools/call", params: { name: "storage_read", arguments: { path: "/taocp/vol_1/1.2/1.md" } } }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.result.isError).toBeFalsy();
      expect(body.result.content[0].text).toContain("TAOCP Volume 1");
    });

    it("POST /mcp returns error for unknown method", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ jsonrpc: "2.0", id: 5, method: "unknown/method" }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.error.code).toBe(-32601);
    });
  });

  // ── Path validation ────────────────────────────────────────────────
  describe("Path validation", () => {
    it("rejects paths with null bytes via uploads", async () => {
      const res = await api("/files/uploads", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: "a\x00b.txt" }),
      });
      expect(res.status).toBe(400);
    });

    it("rejects absolute paths via delete", async () => {
      const res = await api("/files//etc/passwd", { method: "DELETE", token });
      expect(res.status).toBe(400);
    });
  });

  // ── Tenant isolation ───────────────────────────────────────────────
  describe("Tenant isolation", () => {
    let otherToken: string;

    beforeAll(async () => {
      const { env } = await import("cloudflare:test");
      await env.DB.prepare(
        "INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES (?, 'agent', 'key2', ?)",
      ).bind("otheruser", Date.now()).run();
      await env.DB.prepare(
        "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
      ).bind("other-session-token", "otheruser", Date.now() + 86400000).run();
      otherToken = "other-session-token";
    });

    it("user cannot read another user's files", async () => {
      const res = await api("/files/hello.txt", { token: otherToken });
      expect(res.status).toBe(404);
    });

    it("user cannot list another user's files", async () => {
      const res = await api("/files", { token: otherToken });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.entries.length).toBe(0);
    });

    it("user cannot delete another user's files", async () => {
      const res = await api("/files/hello.txt", { method: "DELETE", token: otherToken });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.deleted).toBe(1); // R2 delete succeeds but deletes nothing in other user's namespace

      // Original file should still exist
      const check = await api("/files/hello.txt", { method: "HEAD", token });
      expect(check.status).toBe(200);
    });
  });

  // ── OpenAPI / Docs ─────────────────────────────────────────────────
  describe("Docs", () => {
    it("GET /openapi.json returns spec", async () => {
      const res = await api("/openapi.json");
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.openapi).toBe("3.1.0");
      expect(body.info.title).toBe("Storage API");
    });

    it("GET /docs returns Swagger UI", async () => {
      const res = await api("/docs");
      expect(res.status).toBe(200);
      const html = await res.text();
      expect(html).toContain("swagger-ui");
    });

    it("GET /api returns markdown docs", async () => {
      const res = await api("/api");
      expect(res.status).toBe(200);
      const html = await res.text();
      expect(html).toContain("API Reference");
    });
  });

  // ── Pages ──────────────────────────────────────────────────────────
  describe("Pages", () => {
    it("GET / returns home page", async () => {
      const res = await api("/");
      expect(res.status).toBe(200);
      const html = await res.text();
      expect(html).toContain("Storage");
    });

    it("GET /developers returns developers page", async () => {
      const res = await api("/developers");
      expect(res.status).toBe(200);
    });

    it("GET /pricing returns pricing page", async () => {
      const res = await api("/pricing");
      expect(res.status).toBe(200);
    });

    it("GET /cli returns CLI page", async () => {
      const res = await api("/cli");
      expect(res.status).toBe(200);
    });

    it("GET /ai returns AI page", async () => {
      const res = await api("/ai");
      expect(res.status).toBe(200);
    });

    it("GET /browse returns browse page", async () => {
      const res = await api("/browse");
      expect(res.status).toBe(200);
      const html = await res.text();
      expect(html).toContain("browse");
    });
  });

  // ── Error handling ─────────────────────────────────────────────────
  describe("Error handling", () => {
    it("returns 404 JSON for unknown routes", async () => {
      const res = await api("/nonexistent");
      expect(res.status).toBe(404);
      const body = await res.json() as any;
      expect(body.error).toBe("not_found");
    });
  });
});
