import { describe, it, expect, beforeAll } from "vitest";
import { SELF, env } from "cloudflare:test";

// ── Helpers ─────────────────────────────────────────────────────────

async function setupSchema() {
  const db = env.DB;
  // Create tables individually (D1 exec in miniflare has multi-statement issues)
  const stmts = [
    `CREATE TABLE IF NOT EXISTS actors (actor TEXT PRIMARY KEY, type TEXT NOT NULL CHECK(type IN ('human','agent')), public_key TEXT, email TEXT UNIQUE, bio TEXT DEFAULT '', created_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS challenges (id TEXT PRIMARY KEY, actor TEXT NOT NULL, nonce TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, actor TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS magic_tokens (token TEXT PRIMARY KEY, email TEXT NOT NULL, actor TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS objects (id TEXT PRIMARY KEY, owner TEXT NOT NULL, path TEXT NOT NULL, name TEXT NOT NULL, is_folder INTEGER NOT NULL DEFAULT 0, content_type TEXT DEFAULT '', size INTEGER DEFAULT 0, r2_key TEXT DEFAULT '', starred INTEGER DEFAULT 0, trashed_at INTEGER DEFAULT NULL, accessed_at INTEGER DEFAULT NULL, description TEXT DEFAULT '', created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)`,
    `CREATE UNIQUE INDEX IF NOT EXISTS idx_objects_owner_path ON objects(owner, path)`,
    `CREATE TABLE IF NOT EXISTS shares (id TEXT PRIMARY KEY, object_id TEXT NOT NULL, owner TEXT NOT NULL, grantee TEXT NOT NULL, permission TEXT NOT NULL CHECK(permission IN ('viewer','editor','uploader')), created_at INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_shares_grantee ON shares(grantee)`,
    `CREATE INDEX IF NOT EXISTS idx_shares_object ON shares(object_id)`,
    `CREATE TABLE IF NOT EXISTS public_links (id TEXT PRIMARY KEY, object_id TEXT NOT NULL, owner TEXT NOT NULL, token TEXT NOT NULL UNIQUE, permission TEXT NOT NULL DEFAULT 'viewer' CHECK(permission IN ('viewer','editor')), password_hash TEXT, expires_at INTEGER, max_downloads INTEGER, download_count INTEGER DEFAULT 0, created_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, actor TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, name TEXT NOT NULL, scopes TEXT NOT NULL DEFAULT '*', path_prefix TEXT DEFAULT '', expires_at INTEGER, last_used_at INTEGER, created_at INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_api_keys_actor ON api_keys(actor)`,
    `CREATE TABLE IF NOT EXISTS audit_log (id INTEGER PRIMARY KEY AUTOINCREMENT, actor TEXT, action TEXT NOT NULL, resource TEXT, detail TEXT, ip TEXT, ts INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS rate_limits (key TEXT PRIMARY KEY, count INTEGER NOT NULL DEFAULT 1, window INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS oauth_clients (client_id TEXT PRIMARY KEY, redirect_uris TEXT NOT NULL, client_name TEXT DEFAULT '', token_endpoint_auth_method TEXT DEFAULT 'none', created_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS oauth_codes (code TEXT PRIMARY KEY, actor TEXT NOT NULL, client_id TEXT NOT NULL, redirect_uri TEXT NOT NULL, scope TEXT DEFAULT '*', code_challenge TEXT NOT NULL, code_challenge_method TEXT DEFAULT 'S256', expires_at INTEGER NOT NULL)`,
  ];
  for (const sql of stmts) {
    await db.exec(sql);
  }
}

async function createActor(name: string, type: "human" | "agent" = "agent") {
  const now = Date.now();
  await env.DB.prepare(
    "INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES (?, ?, ?, ?)",
  ).bind(name, type, "test-key", now).run();
}

async function createSession(actor: string): Promise<string> {
  const token = `ses_${actor}_${Date.now()}`;
  await env.DB.prepare(
    "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
  ).bind(token, actor, Date.now() + 7200000).run();
  return token;
}

async function uploadFile(token: string, path: string, content: string = "hello") {
  return SELF.fetch(`https://test.local/files/${path}`, {
    method: "PUT",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "text/plain" },
    body: content,
  });
}

async function readFile(token: string, path: string) {
  return SELF.fetch(`https://test.local/files/${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
}

async function deleteFile(token: string, path: string) {
  return SELF.fetch(`https://test.local/files/${path}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
}

async function createShare(
  token: string,
  path: string,
  grantee: string,
  permission: string = "viewer",
) {
  return SELF.fetch("https://test.local/shares", {
    method: "POST",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify({ path, grantee, permission }),
  });
}

async function createApiKey(
  token: string,
  name: string,
  scopes: string[] = ["*"],
  path_prefix: string = "",
) {
  return SELF.fetch("https://test.local/api-keys", {
    method: "POST",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify({ name, scopes, path_prefix }),
  });
}

// ── Setup ───────────────────────────────────────────────────────────

beforeAll(async () => {
  await setupSchema();
  await createActor("u/alice");
  await createActor("u/bob");
  await createActor("a/ci-bot");
});

// ═══════════════════════════════════════════════════════════════════
// 1. TENANT ISOLATION
// ═══════════════════════════════════════════════════════════════════

describe("Tenant isolation", () => {
  it("alice cannot read bob's files", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    await uploadFile(bobToken, "secret.txt", "bob's secret");
    const res = await readFile(aliceToken, "secret.txt");
    expect(res.status).toBe(404);
  });

  it("alice cannot delete bob's files", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    await uploadFile(bobToken, "deleteme.txt", "data");
    const res = await deleteFile(aliceToken, "deleteme.txt");
    expect(res.status).toBe(404);

    // Verify bob's file still exists
    const check = await readFile(bobToken, "deleteme.txt");
    expect(check.status).toBe(200);
  });

  it("alice cannot list bob's folders", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    await uploadFile(bobToken, "docs/report.txt", "report");

    const res = await SELF.fetch("https://test.local/folders/docs", {
      headers: { Authorization: `Bearer ${aliceToken}` },
    });
    expect(res.status).toBe(200);
    const data = await res.json() as any;
    // alice should see NO items in docs/ (she has no docs/)
    expect(data.items.length).toBe(0);
  });

  it("alice cannot HEAD bob's files", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    await uploadFile(bobToken, "headtest.txt", "data");
    const res = await SELF.fetch("https://test.local/files/headtest.txt", {
      method: "HEAD",
      headers: { Authorization: `Bearer ${aliceToken}` },
    });
    expect(res.status).toBe(404);
  });
});

// ═══════════════════════════════════════════════════════════════════
// 2. SHARE PERMISSIONS
// ═══════════════════════════════════════════════════════════════════

describe("Share permissions", () => {
  it("viewer can read but not write shared files", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    await uploadFile(aliceToken, "shared-view.txt", "alice's file");
    await createShare(aliceToken, "shared-view.txt", "u/bob", "viewer");

    // Bob can read via /shared
    const readRes = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/shared-view.txt`,
      { headers: { Authorization: `Bearer ${bobToken}` } },
    );
    expect(readRes.status).toBe(200);
    expect(await readRes.text()).toBe("alice's file");

    // Bob cannot write
    const writeRes = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/shared-view.txt`,
      {
        method: "PUT",
        headers: { Authorization: `Bearer ${bobToken}`, "Content-Type": "text/plain" },
        body: "hacked",
      },
    );
    expect(writeRes.status).toBe(403);
  });

  it("editor can read, write, and delete shared files", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    await uploadFile(aliceToken, "shared-edit.txt", "original");
    await createShare(aliceToken, "shared-edit.txt", "u/bob", "editor");

    // Bob can read
    const readRes = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/shared-edit.txt`,
      { headers: { Authorization: `Bearer ${bobToken}` } },
    );
    expect(readRes.status).toBe(200);

    // Bob can overwrite
    const writeRes = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/shared-edit.txt`,
      {
        method: "PUT",
        headers: { Authorization: `Bearer ${bobToken}`, "Content-Type": "text/plain" },
        body: "updated by bob",
      },
    );
    expect(writeRes.status).toBe(200);

    // Bob can delete
    const delRes = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/shared-edit.txt`,
      { method: "DELETE", headers: { Authorization: `Bearer ${bobToken}` } },
    );
    expect(delRes.status).toBe(200);
  });

  it("uploader can write new files but not overwrite or read", async () => {
    const aliceToken = await createSession("u/alice");
    const botToken = await createSession("a/ci-bot");

    // Create a folder and share it with uploader permission
    await SELF.fetch("https://test.local/folders", {
      method: "POST",
      headers: { Authorization: `Bearer ${aliceToken}`, "Content-Type": "application/json" },
      body: JSON.stringify({ path: "uploads" }),
    });
    await createShare(aliceToken, "uploads/", "a/ci-bot", "uploader");

    // Bot can upload a new file
    const uploadRes = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/uploads/build.log`,
      {
        method: "PUT",
        headers: { Authorization: `Bearer ${botToken}`, "Content-Type": "text/plain" },
        body: "build output",
      },
    );
    expect(uploadRes.status).toBe(201);

    // Bot cannot overwrite
    const overwriteRes = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/uploads/build.log`,
      {
        method: "PUT",
        headers: { Authorization: `Bearer ${botToken}`, "Content-Type": "text/plain" },
        body: "overwrite attempt",
      },
    );
    expect(overwriteRes.status).toBe(403);

    // Bot cannot read
    const readRes = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/uploads/build.log`,
      { headers: { Authorization: `Bearer ${botToken}` } },
    );
    expect(readRes.status).toBe(403);

    // Bot cannot delete
    const delRes = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/uploads/build.log`,
      { method: "DELETE", headers: { Authorization: `Bearer ${botToken}` } },
    );
    expect(delRes.status).toBe(403);
  });
});

// ═══════════════════════════════════════════════════════════════════
// 3. SHARE INHERITANCE
// ═══════════════════════════════════════════════════════════════════

describe("Share inheritance", () => {
  it("folder share cascades to child files", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    await uploadFile(aliceToken, "project/src/main.ts", "code");
    await createShare(aliceToken, "project/", "u/bob", "viewer");

    // Bob can read child file through inherited permission
    const res = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/project/src/main.ts`,
      { headers: { Authorization: `Bearer ${bobToken}` } },
    );
    expect(res.status).toBe(200);
  });

  it("more specific share overrides parent share", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    await uploadFile(aliceToken, "team/docs/sensitive.txt", "classified");

    // Share parent folder as viewer
    await SELF.fetch("https://test.local/folders", {
      method: "POST",
      headers: { Authorization: `Bearer ${aliceToken}`, "Content-Type": "application/json" },
      body: JSON.stringify({ path: "team" }),
    });
    await createShare(aliceToken, "team/", "u/bob", "viewer");

    // Share specific file as editor
    await createShare(aliceToken, "team/docs/sensitive.txt", "u/bob", "editor");

    // Bob should be able to write (editor on file beats viewer on parent)
    const writeRes = await SELF.fetch(
      `https://test.local/shared/${encodeURIComponent("u/alice")}/team/docs/sensitive.txt`,
      {
        method: "PUT",
        headers: { Authorization: `Bearer ${bobToken}`, "Content-Type": "text/plain" },
        body: "updated",
      },
    );
    expect(writeRes.status).toBe(200);
  });
});

// ═══════════════════════════════════════════════════════════════════
// 4. SELF-SHARE PREVENTION
// ═══════════════════════════════════════════════════════════════════

describe("Self-share prevention", () => {
  it("cannot share with yourself", async () => {
    const aliceToken = await createSession("u/alice");
    await uploadFile(aliceToken, "selfshare.txt", "test");

    const res = await createShare(aliceToken, "selfshare.txt", "u/alice", "viewer");
    expect(res.status).toBe(400);
    const data = await res.json() as any;
    expect(data.error.message).toContain("yourself");
  });
});

// ═══════════════════════════════════════════════════════════════════
// 5. SHARE OWNERSHIP
// ═══════════════════════════════════════════════════════════════════

describe("Share ownership", () => {
  it("only owner can create shares", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    await uploadFile(aliceToken, "alice-only.txt", "data");

    // Bob tries to share alice's file → object not found (he doesn't own it)
    const res = await createShare(bobToken, "alice-only.txt", "a/ci-bot", "viewer");
    expect(res.status).toBe(404);
  });

  it("only owner can update share permission", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    await uploadFile(aliceToken, "for-bob.txt", "data");
    const shareRes = await createShare(aliceToken, "for-bob.txt", "u/bob", "viewer");
    const share = await shareRes.json() as any;

    // Bob tries to upgrade himself to editor
    const patchRes = await SELF.fetch(`https://test.local/shares/${share.id}`, {
      method: "PATCH",
      headers: { Authorization: `Bearer ${bobToken}`, "Content-Type": "application/json" },
      body: JSON.stringify({ permission: "editor" }),
    });
    expect(patchRes.status).toBe(403);
  });

  it("both owner and grantee can delete a share", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");

    // Grantee can delete
    await uploadFile(aliceToken, "revoke-test1.txt", "data");
    const share1Res = await createShare(aliceToken, "revoke-test1.txt", "u/bob", "viewer");
    const share1 = await share1Res.json() as any;

    const del1 = await SELF.fetch(`https://test.local/shares/${share1.id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${bobToken}` },
    });
    expect(del1.status).toBe(200);

    // Owner can delete
    await uploadFile(aliceToken, "revoke-test2.txt", "data");
    const share2Res = await createShare(aliceToken, "revoke-test2.txt", "u/bob", "viewer");
    const share2 = await share2Res.json() as any;

    const del2 = await SELF.fetch(`https://test.local/shares/${share2.id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${aliceToken}` },
    });
    expect(del2.status).toBe(200);
  });

  it("third party cannot delete a share", async () => {
    const aliceToken = await createSession("u/alice");
    const botToken = await createSession("a/ci-bot");

    await uploadFile(aliceToken, "third-party.txt", "data");
    const shareRes = await createShare(aliceToken, "third-party.txt", "u/bob", "viewer");
    const share = await shareRes.json() as any;

    // ci-bot is neither owner nor grantee
    const delRes = await SELF.fetch(`https://test.local/shares/${share.id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${botToken}` },
    });
    expect(delRes.status).toBe(403);
  });
});

// ═══════════════════════════════════════════════════════════════════
// 6. API KEY SCOPING
// ═══════════════════════════════════════════════════════════════════

describe("API key scopes", () => {
  it("read-only key cannot write files", async () => {
    const aliceToken = await createSession("u/alice");
    const keyRes = await createApiKey(aliceToken, "read-only", ["files:read"]);
    const key = await keyRes.json() as any;

    const writeRes = await uploadFile(key.token, "via-key.txt", "test");
    expect(writeRes.status).toBe(403);
    const err = await writeRes.json() as any;
    expect(err.error.message).toContain("scope");
  });

  it("write-only key cannot read files", async () => {
    const aliceToken = await createSession("u/alice");
    await uploadFile(aliceToken, "key-read-test.txt", "hello");

    const keyRes = await createApiKey(aliceToken, "write-only", ["files:write"]);
    const key = await keyRes.json() as any;

    const readRes = await readFile(key.token, "key-read-test.txt");
    expect(readRes.status).toBe(403);
  });

  it("files scope cannot manage shares", async () => {
    const aliceToken = await createSession("u/alice");
    await uploadFile(aliceToken, "scope-test.txt", "data");

    const keyRes = await createApiKey(aliceToken, "files-only", ["files:read", "files:write"]);
    const key = await keyRes.json() as any;

    const shareRes = await SELF.fetch("https://test.local/shares", {
      method: "POST",
      headers: { Authorization: `Bearer ${key.token}`, "Content-Type": "application/json" },
      body: JSON.stringify({ path: "scope-test.txt", grantee: "u/bob", permission: "viewer" }),
    });
    expect(shareRes.status).toBe(403);
  });

  it("API key cannot create other API keys", async () => {
    const aliceToken = await createSession("u/alice");
    const keyRes = await createApiKey(aliceToken, "all-scopes", ["*"]);
    const key = await keyRes.json() as any;

    const newKeyRes = await createApiKey(key.token, "nested-key", ["files:read"]);
    expect(newKeyRes.status).toBe(403);
    const err = await newKeyRes.json() as any;
    expect(err.error.message).toContain("session token");
  });
});

// ═══════════════════════════════════════════════════════════════════
// 7. PATH PREFIX RESTRICTION
// ═══════════════════════════════════════════════════════════════════

describe("Path prefix restriction", () => {
  it("key with prefix cannot access files outside prefix", async () => {
    const aliceToken = await createSession("u/alice");
    await uploadFile(aliceToken, "public/index.html", "<h1>hi</h1>");
    await uploadFile(aliceToken, "private/secret.txt", "secret");

    const keyRes = await createApiKey(aliceToken, "public-only", ["files:read", "files:write"], "public/");
    const key = await keyRes.json() as any;

    // Can read within prefix
    const okRes = await readFile(key.token, "public/index.html");
    expect(okRes.status).toBe(200);

    // Cannot read outside prefix
    const failRes = await readFile(key.token, "private/secret.txt");
    expect(failRes.status).toBe(403);
    const err = await failRes.json() as any;
    expect(err.error.message).toContain("Path not allowed");
  });

  it("key with prefix cannot write outside prefix", async () => {
    const aliceToken = await createSession("u/alice");

    const keyRes = await createApiKey(aliceToken, "uploads-only", ["files:write"], "uploads/");
    const key = await keyRes.json() as any;

    const okRes = await uploadFile(key.token, "uploads/file.txt", "ok");
    expect(okRes.status).toBe(201);

    const failRes = await uploadFile(key.token, "secrets/hack.txt", "bad");
    expect(failRes.status).toBe(403);
  });
});

// ═══════════════════════════════════════════════════════════════════
// 8. PATH TRAVERSAL
// ═══════════════════════════════════════════════════════════════════

describe("Path traversal prevention", () => {
  // NOTE: URL-based paths (PUT /files/docs/../x) are normalized by the URL parser
  // before reaching our handler. That's defense layer 1. Our validatePath() is
  // defense layer 2, catching traversal in JSON bodies (e.g. folder creation).

  it("URL normalization resolves .. before handler", async () => {
    const aliceToken = await createSession("u/alice");

    // URL parser resolves "docs/../etc/passwd" → "etc/passwd"
    // So the file is safely created at "etc/passwd" (no traversal)
    const res = await uploadFile(aliceToken, "docs/../etc/passwd", "test");
    expect([200, 201].includes(res.status)).toBe(true);

    // Verify it was stored at the normalized path, not a dangerous one
    const read = await readFile(aliceToken, "etc/passwd");
    expect(read.status).toBe(200);
  });

  it("rejects .. in JSON body paths (folder creation)", async () => {
    const aliceToken = await createSession("u/alice");

    const res = await SELF.fetch("https://test.local/folders", {
      method: "POST",
      headers: { Authorization: `Bearer ${aliceToken}`, "Content-Type": "application/json" },
      body: JSON.stringify({ path: "docs/../../etc" }),
    });
    expect(res.status).toBe(400);
  });

  it("rejects paths exceeding max length", async () => {
    const aliceToken = await createSession("u/alice");

    const longPath = "a".repeat(1025) + ".txt";
    const res = await uploadFile(aliceToken, longPath, "test");
    expect(res.status).toBe(400);
    const data = await res.json() as any;
    expect(data.error.message).toContain("1024");
  });

  it("rejects paths with null bytes", async () => {
    const aliceToken = await createSession("u/alice");

    const res = await uploadFile(aliceToken, "file\0.txt", "test");
    expect(res.status).toBe(400);
  });

  it("validatePath rejects traversal patterns directly", async () => {
    // Import and test the validation function directly
    const { validatePath } = await import("../lib/path");

    expect(validatePath("docs/../etc")).toContain("..");
    expect(validatePath("./file.txt")).toContain(".");
    expect(validatePath("docs//file.txt")).toContain("//");
    expect(validatePath("/absolute/path")).toContain("/");
    expect(validatePath("a".repeat(1025))).toContain("1024");
    expect(validatePath("file\0.txt")).toContain("null");
    expect(validatePath("")).toContain("required");
    // Valid paths should return null
    expect(validatePath("docs/file.txt")).toBeNull();
    expect(validatePath("docs/reports/")).toBeNull();
  });
});

// ═══════════════════════════════════════════════════════════════════
// 9. PUBLIC LINK SECURITY
// ═══════════════════════════════════════════════════════════════════

describe("Public link security", () => {
  it("expired link returns 410", async () => {
    const aliceToken = await createSession("u/alice");
    await uploadFile(aliceToken, "expire-test.txt", "data");

    // Insert an already-expired link directly
    const obj = await env.DB.prepare(
      "SELECT id FROM objects WHERE owner = ? AND path = ?",
    ).bind("u/alice", "expire-test.txt").first<{ id: string }>();

    await env.DB.prepare(
      "INSERT INTO public_links (id, object_id, owner, token, permission, expires_at, download_count, created_at) VALUES (?, ?, ?, ?, 'viewer', ?, 0, ?)",
    ).bind("pl_exp1", obj!.id, "u/alice", "tok_expired", Date.now() - 1000, Date.now()).run();

    const res = await SELF.fetch("https://test.local/p/tok_expired");
    expect(res.status).toBe(410);
  });

  it("max downloads reached returns 410", async () => {
    const aliceToken = await createSession("u/alice");
    await uploadFile(aliceToken, "maxdl-test.txt", "data");

    const obj = await env.DB.prepare(
      "SELECT id FROM objects WHERE owner = ? AND path = ?",
    ).bind("u/alice", "maxdl-test.txt").first<{ id: string }>();

    await env.DB.prepare(
      "INSERT INTO public_links (id, object_id, owner, token, permission, max_downloads, download_count, created_at) VALUES (?, ?, ?, ?, 'viewer', 1, 1, ?)",
    ).bind("pl_max1", obj!.id, "u/alice", "tok_maxed", Date.now()).run();

    const res = await SELF.fetch("https://test.local/p/tok_maxed");
    expect(res.status).toBe(410);
  });

  it("password-protected link rejects wrong password", async () => {
    const aliceToken = await createSession("u/alice");
    await uploadFile(aliceToken, "passwd-test.txt", "protected data");

    // Hash "secret123" with SHA-256
    const passHash = Array.from(
      new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode("secret123"))),
      (b) => b.toString(16).padStart(2, "0"),
    ).join("");

    const obj = await env.DB.prepare(
      "SELECT id FROM objects WHERE owner = ? AND path = ?",
    ).bind("u/alice", "passwd-test.txt").first<{ id: string }>();

    await env.DB.prepare(
      "INSERT INTO public_links (id, object_id, owner, token, permission, password_hash, download_count, created_at) VALUES (?, ?, ?, ?, 'viewer', ?, 0, ?)",
    ).bind("pl_pw1", obj!.id, "u/alice", "tok_passwd", passHash, Date.now()).run();

    // No password → 401
    const res1 = await SELF.fetch("https://test.local/p/tok_passwd");
    expect(res1.status).toBe(401);

    // Wrong password → 401
    const res2 = await SELF.fetch("https://test.local/p/tok_passwd?password=wrong");
    expect(res2.status).toBe(401);

    // Correct password → 200
    const res3 = await SELF.fetch("https://test.local/p/tok_passwd?password=secret123");
    expect(res3.status).toBe(200);
    expect(await res3.text()).toBe("protected data");
  });

  it("invalid token returns 404", async () => {
    const res = await SELF.fetch("https://test.local/p/nonexistent_token");
    expect(res.status).toBe(404);
  });
});

// ═══════════════════════════════════════════════════════════════════
// 10. AUTHENTICATION
// ═══════════════════════════════════════════════════════════════════

describe("Authentication", () => {
  it("no token returns 401", async () => {
    const res = await SELF.fetch("https://test.local/files/test.txt");
    expect(res.status).toBe(401);
  });

  it("invalid token returns 401", async () => {
    const res = await readFile("bogus_token_12345", "test.txt");
    expect(res.status).toBe(401);
  });

  it("expired session returns 401", async () => {
    await env.DB.prepare(
      "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
    ).bind("ses_expired", "u/alice", Date.now() - 1000).run();

    const res = await readFile("ses_expired", "test.txt");
    expect(res.status).toBe(401);
  });

  it("expired API key returns 401", async () => {
    const tokenHash = Array.from(
      new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode("sk_expired_key"))),
      (b) => b.toString(16).padStart(2, "0"),
    ).join("");

    await env.DB.prepare(
      "INSERT INTO api_keys (id, actor, token_hash, name, scopes, path_prefix, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
    ).bind("ak_exp1", "u/alice", tokenHash, "expired", "*", "", Date.now() - 1000, Date.now()).run();

    const res = await readFile("sk_expired_key", "test.txt");
    expect(res.status).toBe(401);
  });
});

// ═══════════════════════════════════════════════════════════════════
// 11. FILE DELETION CASCADES SHARES
// ═══════════════════════════════════════════════════════════════════

describe("Deletion cascades", () => {
  it("deleting a file removes its shares", async () => {
    const aliceToken = await createSession("u/alice");

    await uploadFile(aliceToken, "cascade-test.txt", "data");
    const shareRes = await createShare(aliceToken, "cascade-test.txt", "u/bob", "viewer");
    const share = await shareRes.json() as any;

    // Verify share exists
    const before = await env.DB.prepare("SELECT 1 FROM shares WHERE id = ?").bind(share.id).first();
    expect(before).toBeTruthy();

    // Delete the file
    await deleteFile(aliceToken, "cascade-test.txt");

    // Share should be gone
    const after = await env.DB.prepare("SELECT 1 FROM shares WHERE id = ?").bind(share.id).first();
    expect(after).toBeNull();
  });
});

// ═══════════════════════════════════════════════════════════════════
// 12. VALID PATH OPERATIONS
// ═══════════════════════════════════════════════════════════════════

describe("Valid path operations", () => {
  it("normal file operations work correctly", async () => {
    const aliceToken = await createSession("u/alice");

    // Write
    const writeRes = await uploadFile(aliceToken, "normal/path/file.txt", "content");
    expect(writeRes.status).toBe(201);

    // Read
    const readRes = await readFile(aliceToken, "normal/path/file.txt");
    expect(readRes.status).toBe(200);
    expect(await readRes.text()).toBe("content");

    // Head
    const headRes = await SELF.fetch("https://test.local/files/normal/path/file.txt", {
      method: "HEAD",
      headers: { Authorization: `Bearer ${aliceToken}` },
    });
    expect(headRes.status).toBe(200);

    // Delete
    const delRes = await deleteFile(aliceToken, "normal/path/file.txt");
    expect(delRes.status).toBe(200);

    // Confirm deleted
    const goneRes = await readFile(aliceToken, "normal/path/file.txt");
    expect(goneRes.status).toBe(404);
  });

  it("special characters in filenames work", async () => {
    const aliceToken = await createSession("u/alice");

    const res = await uploadFile(aliceToken, "docs/my file (1).txt", "content");
    expect(res.status).toBe(201);

    const readRes = await SELF.fetch(
      `https://test.local/files/${encodeURIComponent("docs/my file (1).txt")}`,
      { headers: { Authorization: `Bearer ${aliceToken}` } },
    );
    expect(readRes.status).toBe(200);
  });
});
