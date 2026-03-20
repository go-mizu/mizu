/**
 * Storage API v3 — integration tests
 *
 * Tests the flat file model endpoints:
 *   /f/*   (files: DELETE/HEAD — no data proxying)
 *   /presign/* (presigned upload/download via R2)
 *   /ls/*  (directory listing)
 *   /find  (search)
 *   /mv    (move/rename)
 *   /stat  (usage stats)
 *   /share (create share URL)
 *   /s/:token (access shared file)
 *   /auth/*  (register, challenge, verify, logout)
 *   /auth/keys/* (API key management)
 *   /mcp   (MCP JSON-RPC)
 */
import { describe, it, expect, beforeAll } from "vitest";
import { SELF } from "cloudflare:test";

// Helper to make requests with a bearer token
function api(path: string, opts: RequestInit & { token?: string } = {}) {
  const headers = new Headers(opts.headers);
  if (opts.token) headers.set("Authorization", `Bearer ${opts.token}`);
  return SELF.fetch(`http://localhost${path}`, { ...opts, headers });
}

// Helper to seed files directly into R2 + D1 (no Worker data proxy)
async function seedFile(actor: string, path: string, content: string, contentType = "text/plain") {
  const { env } = await import("cloudflare:test");
  const key = `${actor}/${path}`;
  const data = new TextEncoder().encode(content);
  await env.BUCKET.put(key, data, { httpMetadata: { contentType } });
  const name = path.split("/").pop()!;
  await env.DB.prepare(
    "INSERT INTO files (owner, path, name, size, type, updated_at) VALUES (?, ?, ?, ?, ?, ?) " +
      "ON CONFLICT (owner, path) DO UPDATE SET size = excluded.size, type = excluded.type, updated_at = excluded.updated_at",
  ).bind(actor, path, name, data.byteLength, contentType, Date.now()).run();
}

describe("Storage API", () => {
  let token: string;

  // ── Auth flow ──────────────────────────────────────────────────────
  describe("Auth", () => {
    it("POST /auth/register creates an actor", async () => {
      // Generate an Ed25519 keypair for testing
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

    // Apply schema (each statement must be a single db.exec call for D1)
    const stmts = [
      `CREATE TABLE IF NOT EXISTS actors (actor TEXT PRIMARY KEY, type TEXT NOT NULL DEFAULT 'human' CHECK(type IN ('human','agent')), public_key TEXT, email TEXT UNIQUE, created_at INTEGER NOT NULL)`,
      `CREATE TABLE IF NOT EXISTS challenges (id TEXT PRIMARY KEY, actor TEXT NOT NULL, nonce TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
      `CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, actor TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_sessions_actor ON sessions(actor, expires_at)`,
      `CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, actor TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, name TEXT NOT NULL DEFAULT '', prefix TEXT NOT NULL DEFAULT '', expires_at INTEGER, created_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_api_keys_actor ON api_keys(actor)`,
      `CREATE TABLE IF NOT EXISTS files (owner TEXT NOT NULL, path TEXT NOT NULL, name TEXT NOT NULL, size INTEGER NOT NULL DEFAULT 0, type TEXT NOT NULL DEFAULT 'application/octet-stream', updated_at INTEGER NOT NULL, PRIMARY KEY (owner, path))`,
      `CREATE INDEX IF NOT EXISTS idx_files_name ON files(owner, name COLLATE NOCASE)`,
      `CREATE TABLE IF NOT EXISTS audit_log (id INTEGER PRIMARY KEY AUTOINCREMENT, actor TEXT, action TEXT NOT NULL, path TEXT, ip TEXT, ts INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log(actor, ts)`,
      `CREATE TABLE IF NOT EXISTS share_links (token TEXT PRIMARY KEY, actor TEXT NOT NULL, path TEXT NOT NULL, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_share_links_actor ON share_links(actor, created_at)`,
      `CREATE INDEX IF NOT EXISTS idx_share_links_expires ON share_links(expires_at)`,
      `CREATE TABLE IF NOT EXISTS oauth_clients (client_id TEXT PRIMARY KEY, redirect_uris TEXT NOT NULL DEFAULT '[]', client_name TEXT NOT NULL DEFAULT '', token_endpoint_auth_method TEXT NOT NULL DEFAULT 'none', created_at INTEGER NOT NULL)`,
      `CREATE TABLE IF NOT EXISTS oauth_codes (code TEXT PRIMARY KEY, actor TEXT NOT NULL, client_id TEXT NOT NULL, redirect_uri TEXT NOT NULL, scope TEXT NOT NULL DEFAULT '*', code_challenge TEXT NOT NULL, code_challenge_method TEXT NOT NULL DEFAULT 'S256', expires_at INTEGER NOT NULL)`,
      `CREATE INDEX IF NOT EXISTS idx_oauth_codes_expires ON oauth_codes(expires_at)`,
    ];
    for (const sql of stmts) await db.exec(sql);

    // Seed test actor + session
    await db.prepare(
      "INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES (?, 'agent', 'testkey', ?)",
    ).bind("testuser", Date.now()).run();
    await db.prepare(
      "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
    ).bind("test-session-token", "testuser", Date.now() + 86400000).run();

    token = "test-session-token";

    // Seed test files directly into R2 + D1
    await seedFile("testuser", "hello.txt", "Hello, world!", "text/plain");
    await seedFile("testuser", "docs/readme.md", "# README", "text/markdown");
    await seedFile("testuser", "taocp/vol_1/1.2/1.md", "# TAOCP Volume 1, Section 1.2, Problem 1", "text/markdown");
  });

  // ── Files (DELETE + HEAD only — no data proxy) ────────────────────
  describe("Files", () => {
    it("HEAD /f/* returns metadata headers", async () => {
      const res = await api("/f/hello.txt", { method: "HEAD", token });
      expect(res.status).toBe(200);
      expect(res.headers.get("Content-Length")).toBe("13");
      expect(res.headers.get("ETag")).toBeDefined();
    });

    it("HEAD /f/* returns 404 for missing file", async () => {
      const res = await api("/f/nonexistent.txt", { method: "HEAD", token });
      expect(res.status).toBe(404);
    });

    it("ls at each level of deeply nested path works", async () => {
      // Root should show taocp/
      const r1 = await api("/ls", { token });
      const b1 = await r1.json() as any;
      expect(b1.entries.map((e: any) => e.name)).toContain("taocp/");

      // taocp/ should show vol_1/
      const r2 = await api("/ls/taocp/", { token });
      const b2 = await r2.json() as any;
      expect(b2.entries.map((e: any) => e.name)).toContain("vol_1/");

      // taocp/vol_1/ should show 1.2/
      const r3 = await api("/ls/taocp/vol_1/", { token });
      const b3 = await r3.json() as any;
      expect(b3.entries.map((e: any) => e.name)).toContain("1.2/");

      // taocp/vol_1/1.2/ should show 1.md
      const r4 = await api("/ls/taocp/vol_1/1.2/", { token });
      const b4 = await r4.json() as any;
      expect(b4.entries.map((e: any) => e.name)).toContain("1.md");
    });

    it("DELETE /f/* deletes a single file", async () => {
      await seedFile("testuser", "to-delete.txt", "delete me");

      const res = await api("/f/to-delete.txt", { method: "DELETE", token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.deleted).toBe(1);

      // Verify it's gone via HEAD
      const check = await api("/f/to-delete.txt", { method: "HEAD", token });
      expect(check.status).toBe(404);
    });

    it("DELETE /f/*/ deletes directory recursively", async () => {
      await seedFile("testuser", "tmp/a.txt", "a");
      await seedFile("testuser", "tmp/b.txt", "b");

      const res = await api("/f/tmp/", { method: "DELETE", token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.deleted).toBeGreaterThanOrEqual(2);
    });

    it("rejects requests without auth", async () => {
      const res = await api("/f/hello.txt", { method: "HEAD" });
      expect(res.status).toBe(401);
    });
  });

  // ── Presign ───────────────────────────────────────────────────────
  describe("Presign", () => {
    it("POST /presign/upload returns a presigned PUT URL", async () => {
      const res = await api("/presign/upload", {
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

    it("POST /presign/complete verifies upload in R2", async () => {
      // Seed a file directly to R2 (simulating a completed presigned upload)
      const { env } = await import("cloudflare:test");
      await env.BUCKET.put("testuser/presign-complete-test.txt", "hello presign", {
        httpMetadata: { contentType: "text/plain" },
      });

      const res = await api("/presign/complete", {
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
      expect(body.type).toBe("text/plain");
    });

    it("GET /presign/read/* returns a presigned GET URL", async () => {
      const res = await api("/presign/read/hello.txt", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.url).toContain("r2.cloudflarestorage.com");
      expect(body.url).toContain("X-Amz-Signature");
      expect(body.size).toBe(13);
      expect(body.type).toBe("text/plain");
    });

    it("GET /presign/read/* returns 404 for missing file", async () => {
      const res = await api("/presign/read/nonexistent.txt", { token });
      expect(res.status).toBe(404);
    });

    it("POST /presign/upload rejects without auth", async () => {
      const res = await api("/presign/upload", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: "test.txt" }),
      });
      expect(res.status).toBe(401);
    });

    it("POST /presign/complete returns 404 for missing upload", async () => {
      const res = await api("/presign/complete", {
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
    it("GET /ls returns root entries", async () => {
      const res = await api("/ls", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.entries).toBeInstanceOf(Array);
      expect(body.entries.length).toBeGreaterThan(0);
    });

    it("GET /ls/* lists directory contents", async () => {
      const res = await api("/ls/docs/", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      const names = body.entries.map((e: any) => e.name);
      expect(names).toContain("readme.md");
    });

    it("GET /ls supports limit and offset", async () => {
      const res = await api("/ls?limit=1", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.entries.length).toBeLessThanOrEqual(1);
    });
  });

  // ── Search ─────────────────────────────────────────────────────────
  describe("Search", () => {
    it("GET /find?q= finds files by name", async () => {
      const res = await api("/find?q=hello", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.results.length).toBeGreaterThan(0);
      expect(body.results[0].name).toContain("hello");
    });

    it("GET /find?q= returns empty for no match", async () => {
      const res = await api("/find?q=zzzznonexistent", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.results.length).toBe(0);
    });

    it("GET /find requires q parameter", async () => {
      const res = await api("/find", { token });
      expect(res.status).toBe(400);
    });
  });

  // ── Move ───────────────────────────────────────────────────────────
  describe("Move", () => {
    it("POST /mv moves a file", async () => {
      await seedFile("testuser", "moveme.txt", "move me");

      const res = await api("/mv", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ from: "moveme.txt", to: "moved.txt" }),
      });
      expect(res.status).toBe(200);

      // Old path should be gone (HEAD)
      const old = await api("/f/moveme.txt", { method: "HEAD", token });
      expect(old.status).toBe(404);

      // New path should exist (HEAD)
      const newFile = await api("/f/moved.txt", { method: "HEAD", token });
      expect(newFile.status).toBe(200);
    });

    it("POST /mv returns 404 for nonexistent source", async () => {
      const res = await api("/mv", {
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
    it("GET /stat returns usage", async () => {
      const res = await api("/stat", { token });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.files).toBeGreaterThan(0);
      expect(body.bytes).toBeGreaterThanOrEqual(0);
    });
  });

  // ── Sharing ────────────────────────────────────────────────────────
  describe("Sharing", () => {
    it("POST /share creates a share URL", async () => {
      const res = await api("/share", {
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
      const shareRes = await api("/share", {
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
      expect(location).toContain("hello.txt");
      expect(location).toContain("X-Amz-Signature");
    });

    it("POST /share returns 404 for nonexistent file", async () => {
      const res = await api("/share", {
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
      const res = await api("/f/hello.txt", { method: "HEAD", token: keyToken });
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

      // Key should no longer work
      const check = await api("/f/hello.txt", { method: "HEAD", token: keyToken });
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
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 1,
          method: "initialize",
        }),
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
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 2,
          method: "tools/list",
        }),
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
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 3,
          method: "tools/call",
          params: { name: "storage_list", arguments: {} },
        }),
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
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 4,
          method: "tools/call",
          params: { name: "storage_stats", arguments: {} },
        }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.result.isError).toBeFalsy();
      const text = body.result.content[0].text;
      expect(text).toContain("file_count");
    });

    it("POST /mcp storage_list with 'path' param (LLM alias for prefix)", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 10,
          method: "tools/call",
          params: { name: "storage_list", arguments: { path: "taocp/" } },
        }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.result.isError).toBeFalsy();
      const data = JSON.parse(body.result.content[0].text);
      expect(data.prefix).toBe("taocp/");
      expect(data.entries.some((e: any) => e.name === "vol_1/")).toBe(true);
    });

    it("POST /mcp storage_list with leading slash prefix", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 11,
          method: "tools/call",
          params: { name: "storage_list", arguments: { prefix: "/taocp/" } },
        }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.result.isError).toBeFalsy();
      const data = JSON.parse(body.result.content[0].text);
      expect(data.prefix).toBe("taocp/");
      expect(data.entries.some((e: any) => e.name === "vol_1/")).toBe(true);
    });

    it("POST /mcp storage_read with leading slash path", async () => {
      const res = await api("/mcp", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 12,
          method: "tools/call",
          params: { name: "storage_read", arguments: { path: "/taocp/vol_1/1.2/1.md" } },
        }),
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
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 5,
          method: "unknown/method",
        }),
      });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      expect(body.error.code).toBe(-32601);
    });
  });

  // ── Path validation ────────────────────────────────────────────────
  describe("Path validation", () => {
    it("rejects paths with null bytes via presign", async () => {
      const res = await api("/presign/upload", {
        method: "POST",
        token,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: "a\x00b.txt" }),
      });
      expect(res.status).toBe(400);
    });

    it("rejects absolute paths via delete", async () => {
      const res = await api("/f//etc/passwd", { method: "DELETE", token });
      expect(res.status).toBe(400);
    });

    it("PUT /f/* returns 410 gone", async () => {
      const res = await api("/f/test.txt", { method: "PUT", token, body: "test" });
      expect(res.status).toBe(410);
    });

    it("GET /f/* returns 410 gone", async () => {
      const res = await api("/f/test.txt", { token });
      expect(res.status).toBe(410);
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

    it("user cannot read another user's files via presign", async () => {
      const res = await api("/presign/read/hello.txt", { token: otherToken });
      expect(res.status).toBe(404);
    });

    it("user cannot list another user's files", async () => {
      const res = await api("/ls", { token: otherToken });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      // Other user should have no files
      expect(body.entries.length).toBe(0);
    });

    it("user cannot delete another user's files", async () => {
      const res = await api("/f/hello.txt", { method: "DELETE", token: otherToken });
      expect(res.status).toBe(200);
      const body = await res.json() as any;
      // Nothing to delete for this user
      expect(body.deleted).toBe(1); // The R2 delete succeeds but deletes nothing

      // Original file should still exist
      const check = await api("/f/hello.txt", { method: "HEAD", token });
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
