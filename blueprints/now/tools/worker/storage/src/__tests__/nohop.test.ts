/**
 * Tests for no-hop improvements:
 * - GET /object/:bucket/*path?redirect=1  → 302 to presigned R2 URL (or stream fallback)
 * - GET /files/*path?redirect=1           → same for files API
 * - GET /sign/:token?redirect=1           → same for signed URLs
 * - POST /object/sign/:bucket presign:true → direct R2 presigned URL
 * - POST /object/upload/token             → presigned PUT URL + commit URL
 * - POST /object/commit                   → register metadata after direct upload
 * - TUS PATCH part_count tracking (no R2.list())
 */
import { describe, it, expect, beforeAll } from "vitest";
import { SELF, env } from "cloudflare:test";

const BASE = "https://test.local";

// ── Schema setup ─────────────────────────────────────────────────────

async function setupSchema() {
  const stmts = [
    `CREATE TABLE IF NOT EXISTS actors (actor TEXT PRIMARY KEY, type TEXT NOT NULL CHECK(type IN ('human','agent')), public_key TEXT, email TEXT UNIQUE, bio TEXT DEFAULT '', created_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, actor TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS buckets (id TEXT PRIMARY KEY, owner TEXT NOT NULL, name TEXT NOT NULL, public INTEGER NOT NULL DEFAULT 0, file_size_limit INTEGER DEFAULT NULL, allowed_mime_types TEXT DEFAULT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)`,
    `CREATE UNIQUE INDEX IF NOT EXISTS idx_buckets_owner_name ON buckets(owner, name)`,
    `CREATE TABLE IF NOT EXISTS objects (id TEXT PRIMARY KEY, owner TEXT NOT NULL, bucket_id TEXT NOT NULL, path TEXT NOT NULL, name TEXT NOT NULL, is_folder INTEGER NOT NULL DEFAULT 0, content_type TEXT DEFAULT '', size INTEGER DEFAULT 0, r2_key TEXT DEFAULT '', metadata TEXT DEFAULT '{}', accessed_at INTEGER DEFAULT NULL, trashed_at INTEGER DEFAULT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)`,
    `CREATE UNIQUE INDEX IF NOT EXISTS idx_objects_bucket_path ON objects(bucket_id, path)`,
    `CREATE INDEX IF NOT EXISTS idx_objects_owner ON objects(owner)`,
    `CREATE TABLE IF NOT EXISTS tus_uploads (id TEXT PRIMARY KEY, owner TEXT NOT NULL, bucket_id TEXT NOT NULL, path TEXT NOT NULL, upload_length INTEGER NOT NULL, upload_offset INTEGER NOT NULL DEFAULT 0, part_count INTEGER NOT NULL DEFAULT 0, content_type TEXT DEFAULT '', metadata TEXT DEFAULT '{}', upsert INTEGER NOT NULL DEFAULT 0, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_tus_uploads_owner ON tus_uploads(owner)`,
    `CREATE INDEX IF NOT EXISTS idx_tus_uploads_expires ON tus_uploads(expires_at)`,
    `CREATE TABLE IF NOT EXISTS signed_urls (id TEXT PRIMARY KEY, owner TEXT NOT NULL, bucket_id TEXT NOT NULL REFERENCES buckets(id), path TEXT NOT NULL, token TEXT NOT NULL UNIQUE, type TEXT NOT NULL CHECK(type IN ('download','upload')), expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_signed_urls_token ON signed_urls(token)`,
    `CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, actor TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, name TEXT NOT NULL, scopes TEXT NOT NULL DEFAULT '*', path_prefix TEXT DEFAULT '', expires_at INTEGER, last_used_at INTEGER, created_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS audit_log (id INTEGER PRIMARY KEY AUTOINCREMENT, actor TEXT, action TEXT NOT NULL, resource TEXT, detail TEXT, ip TEXT, ts INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS rate_limits (key TEXT PRIMARY KEY, count INTEGER NOT NULL DEFAULT 1, window INTEGER NOT NULL)`,
  ];
  for (const sql of stmts) {
    await env.DB.exec(sql);
  }
}

async function createActor(name: string) {
  await env.DB.prepare(
    "INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES (?, ?, ?, ?)",
  ).bind(name, "agent", "test-key", Date.now()).run();
}

async function createSession(actor: string): Promise<string> {
  const token = `ses_nohop_${actor}_${Date.now()}_${Math.random().toString(36).slice(2)}`;
  await env.DB.prepare(
    "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
  ).bind(token, actor, Date.now() + 7200000).run();
  return token;
}

async function apiAuth(token: string) {
  return { Authorization: `Bearer ${token}`, "Content-Type": "application/json" };
}

async function createBucket(token: string, name: string, opts: Record<string, any> = {}) {
  const res = await SELF.fetch(`${BASE}/bucket`, {
    method: "POST",
    headers: await apiAuth(token),
    body: JSON.stringify({ name, ...opts }),
  });
  return res.json<{ id: string; name: string }>();
}

async function uploadObject(token: string, bucket: string, path: string, content: string | Uint8Array, contentType = "text/plain") {
  const res = await SELF.fetch(`${BASE}/object/${bucket}/${path}`, {
    method: "PUT",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": contentType },
    body: content,
  });
  return res.json<{ id: string; path: string }>();
}

// ── Setup ─────────────────────────────────────────────────────────────

beforeAll(async () => {
  await setupSchema();
  await createActor("u/nohop-alice");
});

// ═════════════════════════════════════════════════════════════════════
// 1. Object download redirect (?redirect=1)
// ═════════════════════════════════════════════════════════════════════

describe("GET /object/:bucket/*path?redirect=1", () => {
  it("falls through to streaming when R2 credentials not configured", async () => {
    const token = await createSession("u/nohop-alice");
    await createBucket(token, "nohop-redirect-bucket");
    await uploadObject(token, "nohop-redirect-bucket", "test.txt", "hello redirect");

    // R2 credentials not set in test env — should fall through to streaming (200)
    const res = await SELF.fetch(`${BASE}/object/nohop-redirect-bucket/test.txt?redirect=1`, {
      headers: { Authorization: `Bearer ${token}` },
      redirect: "manual",
    });
    // No credentials configured → falls through to streaming
    expect(res.status).toBe(200);
    const text = await res.text();
    expect(text).toBe("hello redirect");
  });

  it("returns 404 for non-existent object", async () => {
    const token = await createSession("u/nohop-alice");
    const res = await SELF.fetch(`${BASE}/object/nohop-redirect-bucket/does-not-exist.txt?redirect=1`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(404);
  });

  it("streams object without ?redirect", async () => {
    const token = await createSession("u/nohop-alice");
    const res = await SELF.fetch(`${BASE}/object/nohop-redirect-bucket/test.txt`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status).toBe(200);
    expect(await res.text()).toBe("hello redirect");
  });
});

// ═════════════════════════════════════════════════════════════════════
// 2. Files download redirect (?redirect=1)
// ═════════════════════════════════════════════════════════════════════

describe("GET /files/*path?redirect=1", () => {
  it("falls through to streaming when R2 credentials not configured", async () => {
    const token = await createSession("u/nohop-alice");

    // Upload via files API
    await SELF.fetch(`${BASE}/files/nohop/redirect-test.txt`, {
      method: "PUT",
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "text/plain" },
      body: "files redirect content",
    });

    const res = await SELF.fetch(`${BASE}/files/nohop/redirect-test.txt?redirect=1`, {
      headers: { Authorization: `Bearer ${token}` },
      redirect: "manual",
    });
    // No credentials → fall through to streaming
    expect(res.status).toBe(200);
    expect(await res.text()).toBe("files redirect content");
  });
});

// ═════════════════════════════════════════════════════════════════════
// 3. Signed URL redirect (?redirect=1)
// ═════════════════════════════════════════════════════════════════════

describe("GET /sign/:token?redirect=1", () => {
  it("falls through to streaming when R2 credentials not configured", async () => {
    const token = await createSession("u/nohop-alice");

    // Create signed URL
    const signRes = await SELF.fetch(`${BASE}/object/sign/nohop-redirect-bucket`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ path: "test.txt" }),
    });
    expect(signRes.status).toBe(200);
    const { signed_url } = await signRes.json<{ signed_url: string }>();
    const signToken = signed_url.replace("/sign/", "");

    const res = await SELF.fetch(`${BASE}/sign/${signToken}?redirect=1`, {
      redirect: "manual",
    });
    // No credentials → fall through to streaming
    expect(res.status).toBe(200);
    expect(await res.text()).toBe("hello redirect");
  });
});

// ═════════════════════════════════════════════════════════════════════
// 4. createSignedUrl with presign: true
// ═════════════════════════════════════════════════════════════════════

describe("POST /object/sign/:bucket with presign: true", () => {
  it("returns error when R2 credentials not configured", async () => {
    const token = await createSession("u/nohop-alice");

    const res = await SELF.fetch(`${BASE}/object/sign/nohop-redirect-bucket`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ path: "test.txt", presign: true }),
    });
    // No R2 credentials in test env
    expect(res.status).toBe(400);
    const body = await res.json<{ error: { code: string } }>();
    expect(body.error.code).toBe("not_configured");
  });

  it("still works normally without presign flag", async () => {
    const token = await createSession("u/nohop-alice");

    const res = await SELF.fetch(`${BASE}/object/sign/nohop-redirect-bucket`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ path: "test.txt" }),
    });
    expect(res.status).toBe(200);
    const body = await res.json<{ signed_url: string; token: string }>();
    expect(body.signed_url).toMatch(/^\/sign\//);
    expect(body.token).toBeTruthy();
  });

  it("returns 404 for non-existent path", async () => {
    const token = await createSession("u/nohop-alice");

    const res = await SELF.fetch(`${BASE}/object/sign/nohop-redirect-bucket`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ path: "does-not-exist.txt" }),
    });
    expect(res.status).toBe(404);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 5. POST /object/upload/token
// ═════════════════════════════════════════════════════════════════════

describe("POST /object/upload/token", () => {
  it("requires authentication", async () => {
    const res = await SELF.fetch(`${BASE}/object/upload/token`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ bucket: "test", path: "file.txt" }),
    });
    expect(res.status).toBe(401);
  });

  it("returns error when R2 credentials not configured", async () => {
    const token = await createSession("u/nohop-alice");
    await createBucket(token, "nohop-token-bucket");

    const res = await SELF.fetch(`${BASE}/object/upload/token`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ bucket: "nohop-token-bucket", path: "direct.txt" }),
    });
    // No R2 credentials in test environment
    expect(res.status).toBe(400);
    const body = await res.json<{ error: { code: string } }>();
    expect(body.error.code).toBe("not_configured");
  });

  it("returns 404 for non-existent bucket", async () => {
    const token = await createSession("u/nohop-alice");

    const res = await SELF.fetch(`${BASE}/object/upload/token`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ bucket: "no-such-bucket", path: "file.txt" }),
    });
    expect(res.status).toBe(404);
  });

  it("validates required fields", async () => {
    const token = await createSession("u/nohop-alice");

    const resMissingBucket = await SELF.fetch(`${BASE}/object/upload/token`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ path: "file.txt" }),
    });
    expect(resMissingBucket.status).toBe(400);

    const resMissingPath = await SELF.fetch(`${BASE}/object/upload/token`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ bucket: "nohop-token-bucket" }),
    });
    expect(resMissingPath.status).toBe(400);
  });

  it("rejects folder paths", async () => {
    const token = await createSession("u/nohop-alice");

    const res = await SELF.fetch(`${BASE}/object/upload/token`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ bucket: "nohop-token-bucket", path: "folder/" }),
    });
    expect(res.status).toBe(400);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 6. POST /object/commit
// ═════════════════════════════════════════════════════════════════════

describe("POST /object/commit", () => {
  it("requires authentication", async () => {
    const res = await SELF.fetch(`${BASE}/object/commit`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ bucket: "test", path: "file.txt" }),
    });
    expect(res.status).toBe(401);
  });

  it("returns 404 for non-existent bucket", async () => {
    const token = await createSession("u/nohop-alice");

    const res = await SELF.fetch(`${BASE}/object/commit`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ bucket: "no-such-bucket", path: "file.txt" }),
    });
    expect(res.status).toBe(404);
  });

  it("returns 404 when R2 object not found", async () => {
    const token = await createSession("u/nohop-alice");
    await createBucket(token, "nohop-commit-bucket");

    // File was never uploaded to R2
    const res = await SELF.fetch(`${BASE}/object/commit`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ bucket: "nohop-commit-bucket", path: "not-uploaded.txt" }),
    });
    expect(res.status).toBe(404);
    const body = await res.json<{ error: { code: string } }>();
    expect(body.error.code).toBe("not_found");
  });

  it("registers metadata for object already in R2", async () => {
    const token = await createSession("u/nohop-alice");
    await createBucket(token, "nohop-commit-bucket2");

    // Pre-seed R2 with the object (simulating a direct upload)
    const actor = "u/nohop-alice";
    const r2Key = `${actor}/nohop-commit-bucket2/direct-upload.txt`;
    await env.BUCKET.put(r2Key, new TextEncoder().encode("direct content"), {
      httpMetadata: { contentType: "text/plain" },
    });

    const res = await SELF.fetch(`${BASE}/object/commit`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ bucket: "nohop-commit-bucket2", path: "direct-upload.txt" }),
    });
    expect(res.status).toBe(201);
    const body = await res.json<{
      id: string;
      bucket: string;
      path: string;
      name: string;
      content_type: string;
      size: number;
    }>();
    expect(body.path).toBe("direct-upload.txt");
    expect(body.bucket).toBe("nohop-commit-bucket2");
    expect(body.name).toBe("direct-upload.txt");
    expect(body.content_type).toBe("text/plain");
    expect(body.size).toBeGreaterThan(0);
    expect(body.id).toBeTruthy();
  });

  it("upserts metadata on second commit for same path", async () => {
    const token = await createSession("u/nohop-alice");
    await createBucket(token, "nohop-commit-bucket3");

    const actor = "u/nohop-alice";
    const r2Key = `${actor}/nohop-commit-bucket3/update-me.txt`;

    // First commit
    await env.BUCKET.put(r2Key, new TextEncoder().encode("v1"), { httpMetadata: { contentType: "text/plain" } });
    const res1 = await SELF.fetch(`${BASE}/object/commit`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ bucket: "nohop-commit-bucket3", path: "update-me.txt" }),
    });
    expect(res1.status).toBe(201);
    const body1 = await res1.json<{ id: string; size: number }>();
    expect(body1.size).toBe(2); // "v1"

    // Update R2 object
    await env.BUCKET.put(r2Key, new TextEncoder().encode("version2"), { httpMetadata: { contentType: "text/plain" } });

    // Second commit — should update D1 metadata
    const res2 = await SELF.fetch(`${BASE}/object/commit`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ bucket: "nohop-commit-bucket3", path: "update-me.txt" }),
    });
    expect(res2.status).toBe(200);
    const body2 = await res2.json<{ id: string; size: number }>();
    expect(body2.id).toBe(body1.id); // Same object, updated
    expect(body2.size).toBe(8); // "version2"
  });

  it("validates required fields", async () => {
    const token = await createSession("u/nohop-alice");

    const resMissingBucket = await SELF.fetch(`${BASE}/object/commit`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ path: "file.txt" }),
    });
    expect(resMissingBucket.status).toBe(400);

    const resMissingPath = await SELF.fetch(`${BASE}/object/commit`, {
      method: "POST",
      headers: await apiAuth(token),
      body: JSON.stringify({ bucket: "nohop-commit-bucket" }),
    });
    expect(resMissingPath.status).toBe(400);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 7. TUS part_count tracking (replaces R2.list() per PATCH)
// ═════════════════════════════════════════════════════════════════════

describe("TUS part_count D1 tracking", () => {
  async function createTusBucket(token: string) {
    return createBucket(token, "nohop-tus-bucket");
  }

  function tusHdr(token: string, extra?: Record<string, string>) {
    return { Authorization: `Bearer ${token}`, "Tus-Resumable": "1.0.0", ...extra };
  }

  function b64(s: string) { return btoa(s); }

  it("part_count is 0 after creation", async () => {
    const token = await createSession("u/nohop-alice");
    await createTusBucket(token);

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHdr(token, {
        "Upload-Length": "20",
        "Upload-Metadata": `bucketName ${b64("nohop-tus-bucket")},objectName ${b64("part-count-test.bin")}`,
      }),
    });
    expect(res.status).toBe(201);
    const location = res.headers.get("Location")!;
    const id = location.split("/").pop()!;

    const row = await env.DB
      .prepare("SELECT part_count FROM tus_uploads WHERE id = ?")
      .bind(id)
      .first<{ part_count: number }>();
    expect(row?.part_count).toBe(0);
  });

  it("part_count increments on each PATCH", async () => {
    const token = await createSession("u/nohop-alice");

    // Create upload
    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHdr(token, {
        "Upload-Length": "20",
        "Upload-Metadata": `bucketName ${b64("nohop-tus-bucket")},objectName ${b64("part-count-patch.bin")}`,
      }),
    });
    const location = createRes.headers.get("Location")!;
    const id = location.split("/").pop()!;

    // First PATCH (10 bytes)
    const patch1 = await SELF.fetch(`${BASE}/upload/resumable/${id}`, {
      method: "PATCH",
      headers: tusHdr(token, {
        "Upload-Offset": "0",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: new Uint8Array(10).fill(1),
    });
    expect(patch1.status).toBe(204);

    const row1 = await env.DB
      .prepare("SELECT part_count, upload_offset FROM tus_uploads WHERE id = ?")
      .bind(id)
      .first<{ part_count: number; upload_offset: number }>();
    expect(row1?.part_count).toBe(1);
    expect(row1?.upload_offset).toBe(10);

    // Second PATCH (10 bytes — completes upload, so row is deleted)
    const patch2 = await SELF.fetch(`${BASE}/upload/resumable/${id}`, {
      method: "PATCH",
      headers: tusHdr(token, {
        "Upload-Offset": "10",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: new Uint8Array(10).fill(2),
    });
    expect(patch2.status).toBe(204);
    expect(patch2.headers.get("Tus-Complete")).toBe("1");

    // After assembly, row is deleted
    const rowDone = await env.DB
      .prepare("SELECT id FROM tus_uploads WHERE id = ?")
      .bind(id)
      .first();
    expect(rowDone).toBeNull();

    // Verify the assembled object is in R2
    const r2Obj = await env.BUCKET.get(`u/nohop-alice/nohop-tus-bucket/part-count-patch.bin`);
    expect(r2Obj).not.toBeNull();
  });

  it("DELETE uses part_count to clean up R2 parts", async () => {
    const token = await createSession("u/nohop-alice");

    // Create upload
    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHdr(token, {
        "Upload-Length": "20",
        "Upload-Metadata": `bucketName ${b64("nohop-tus-bucket")},objectName ${b64("delete-parts-test.bin")}`,
      }),
    });
    const location = createRes.headers.get("Location")!;
    const id = location.split("/").pop()!;

    // Upload one part
    await SELF.fetch(`${BASE}/upload/resumable/${id}`, {
      method: "PATCH",
      headers: tusHdr(token, {
        "Upload-Offset": "0",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: new Uint8Array(10).fill(42),
    });

    // Verify R2 part exists
    const partBefore = await env.BUCKET.get(`__tus/${id}/0`);
    expect(partBefore).not.toBeNull();

    // Cancel the upload
    const deleteRes = await SELF.fetch(`${BASE}/upload/resumable/${id}`, {
      method: "DELETE",
      headers: tusHdr(token),
    });
    expect(deleteRes.status).toBe(204);

    // R2 part should be deleted
    const partAfter = await env.BUCKET.get(`__tus/${id}/0`);
    expect(partAfter).toBeNull();

    // D1 row should be gone
    const rowAfter = await env.DB
      .prepare("SELECT id FROM tus_uploads WHERE id = ?")
      .bind(id)
      .first();
    expect(rowAfter).toBeNull();
  });

  it("creation-with-upload sets part_count=1 and assembles immediately", async () => {
    const token = await createSession("u/nohop-alice");
    const data = new TextEncoder().encode("small file content");

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHdr(token, {
        "Upload-Length": data.length.toString(),
        "Upload-Metadata": `bucketName ${b64("nohop-tus-bucket")},objectName ${b64("single-post-file.txt")}`,
        "Content-Type": "application/offset+octet-stream",
      }),
      body: data,
    });
    expect(res.status).toBe(201);
    expect(res.headers.get("Tus-Complete")).toBe("1");

    // Object should be in D1
    const obj = await env.DB
      .prepare("SELECT id, size FROM objects WHERE path = ? AND owner = ?")
      .bind("single-post-file.txt", "u/nohop-alice")
      .first<{ id: string; size: number }>();
    expect(obj).not.toBeNull();
    expect(obj?.size).toBe(data.length);

    // TUS upload record cleaned up
    const location = res.headers.get("Location")!;
    const id = location.split("/").pop()!;
    const row = await env.DB
      .prepare("SELECT id FROM tus_uploads WHERE id = ?")
      .bind(id)
      .first();
    expect(row).toBeNull();
  });
});

// ═════════════════════════════════════════════════════════════════════
// 8. Tenant isolation for commit
// ═════════════════════════════════════════════════════════════════════

describe("objectCommit tenant isolation", () => {
  it("cannot commit to another user's bucket", async () => {
    await createActor("u/nohop-bob");
    const aliceToken = await createSession("u/nohop-alice");
    const bobToken = await createSession("u/nohop-bob");

    await createBucket(aliceToken, "nohop-alice-private");

    // Bob tries to commit to Alice's bucket
    const res = await SELF.fetch(`${BASE}/object/commit`, {
      method: "POST",
      headers: { Authorization: `Bearer ${bobToken}`, "Content-Type": "application/json" },
      body: JSON.stringify({ bucket: "nohop-alice-private", path: "sneaky.txt" }),
    });
    expect(res.status).toBe(404); // Bucket not found for Bob
  });
});
