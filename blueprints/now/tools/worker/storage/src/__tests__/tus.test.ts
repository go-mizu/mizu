import { describe, it, expect, beforeAll } from "vitest";
import { SELF, env } from "cloudflare:test";

// ── Helpers ─────────────────────────────────────────────────────────

const BASE = "https://test.local";

async function setupSchema() {
  const stmts = [
    `CREATE TABLE IF NOT EXISTS actors (actor TEXT PRIMARY KEY, type TEXT NOT NULL CHECK(type IN ('human','agent')), public_key TEXT, email TEXT UNIQUE, bio TEXT DEFAULT '', created_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, actor TEXT NOT NULL, expires_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS buckets (id TEXT PRIMARY KEY, owner TEXT NOT NULL, name TEXT NOT NULL, public INTEGER NOT NULL DEFAULT 0, file_size_limit INTEGER DEFAULT NULL, allowed_mime_types TEXT DEFAULT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)`,
    `CREATE UNIQUE INDEX IF NOT EXISTS idx_buckets_owner_name ON buckets(owner, name)`,
    `CREATE TABLE IF NOT EXISTS objects (id TEXT PRIMARY KEY, owner TEXT NOT NULL, bucket_id TEXT NOT NULL, path TEXT NOT NULL, name TEXT NOT NULL, content_type TEXT DEFAULT '', size INTEGER DEFAULT 0, r2_key TEXT DEFAULT '', metadata TEXT DEFAULT '{}', accessed_at INTEGER DEFAULT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)`,
    `CREATE UNIQUE INDEX IF NOT EXISTS idx_objects_bucket_path ON objects(bucket_id, path)`,
    `CREATE INDEX IF NOT EXISTS idx_objects_owner ON objects(owner)`,
    `CREATE TABLE IF NOT EXISTS tus_uploads (id TEXT PRIMARY KEY, owner TEXT NOT NULL, bucket_id TEXT NOT NULL, path TEXT NOT NULL, upload_length INTEGER NOT NULL, upload_offset INTEGER NOT NULL DEFAULT 0, part_count INTEGER NOT NULL DEFAULT 0, content_type TEXT DEFAULT '', metadata TEXT DEFAULT '{}', upsert INTEGER NOT NULL DEFAULT 0, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_tus_uploads_owner ON tus_uploads(owner)`,
    `CREATE INDEX IF NOT EXISTS idx_tus_uploads_expires ON tus_uploads(expires_at)`,
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
  const token = `ses_${actor}_${Date.now()}_${Math.random().toString(36).slice(2)}`;
  await env.DB.prepare(
    "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
  ).bind(token, actor, Date.now() + 7200000).run();
  return token;
}

async function createBucket(token: string, name: string, opts?: { file_size_limit?: number; allowed_mime_types?: string[] }) {
  return SELF.fetch(`${BASE}/bucket`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify({ name, ...opts }),
  });
}

function b64(s: string): string {
  return btoa(s);
}

function tusHeaders(token: string, extra?: Record<string, string>): Record<string, string> {
  return {
    Authorization: `Bearer ${token}`,
    "Tus-Resumable": "1.0.0",
    ...extra,
  };
}

// ── Setup ───────────────────────────────────────────────────────────

beforeAll(async () => {
  await setupSchema();
  await createActor("u/alice");
  await createActor("u/bob");
});

// ═════════════════════════════════════════════════════════════════════
// 1. OPTIONS — Server capability discovery
// ═════════════════════════════════════════════════════════════════════

describe("TUS OPTIONS", () => {
  it("returns TUS capabilities", async () => {
    const res = await SELF.fetch(`${BASE}/upload/resumable`, { method: "OPTIONS" });
    expect(res.status).toBe(204);
    expect(res.headers.get("Tus-Resumable")).toBe("1.0.0");
    expect(res.headers.get("Tus-Version")).toBe("1.0.0");
    expect(res.headers.get("Tus-Extension")).toContain("creation");
    expect(res.headers.get("Tus-Extension")).toContain("creation-with-upload");
    expect(res.headers.get("Tus-Extension")).toContain("termination");
    expect(res.headers.get("Tus-Extension")).toContain("expiration");
    expect(res.headers.get("Tus-Max-Size")).toBeTruthy();
  });
});

// ═════════════════════════════════════════════════════════════════════
// 2. POST — Create upload
// ═════════════════════════════════════════════════════════════════════

describe("TUS POST (create upload)", () => {
  it("creates an upload and returns Location", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "tus-test");

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "1024",
        "Upload-Metadata": `bucketName ${b64("tus-test")},objectName ${b64("file.bin")}`,
      }),
    });

    expect(res.status).toBe(201);
    expect(res.headers.get("Tus-Resumable")).toBe("1.0.0");
    expect(res.headers.get("Location")).toBeTruthy();
    expect(res.headers.get("Upload-Offset")).toBe("0");
    expect(res.headers.get("Upload-Expires")).toBeTruthy();
  });

  it("rejects missing Upload-Length", async () => {
    const token = await createSession("u/alice");

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Metadata": `bucketName ${b64("tus-test")},objectName ${b64("no-length.bin")}`,
      }),
    });

    expect(res.status).toBe(400);
  });

  it("rejects missing bucketName in metadata", async () => {
    const token = await createSession("u/alice");

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "100",
        "Upload-Metadata": `objectName ${b64("file.bin")}`,
      }),
    });

    expect(res.status).toBe(400);
  });

  it("rejects missing objectName in metadata", async () => {
    const token = await createSession("u/alice");

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("tus-test")}`,
      }),
    });

    expect(res.status).toBe(400);
  });

  it("rejects non-existent bucket", async () => {
    const token = await createSession("u/alice");

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("no-such-bucket")},objectName ${b64("file.bin")}`,
      }),
    });

    expect(res.status).toBe(404);
  });

  it("rejects wrong TUS version", async () => {
    const token = await createSession("u/alice");

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Tus-Resumable": "0.9.0",
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("tus-test")},objectName ${b64("file.bin")}`,
      },
    });

    expect(res.status).toBe(412);
  });

  it("rejects without auth", async () => {
    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: {
        "Tus-Resumable": "1.0.0",
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("tus-test")},objectName ${b64("file.bin")}`,
      },
    });

    expect(res.status).toBe(401);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 3. Full lifecycle — create, patch, complete
// ═════════════════════════════════════════════════════════════════════

describe("TUS full lifecycle", () => {
  it("uploads a file in two chunks", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "chunks");

    const data = "Hello, TUS world! This is a chunked upload test.";
    const chunk1 = data.slice(0, 24);
    const chunk2 = data.slice(24);

    // POST: create upload
    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": data.length.toString(),
        "Upload-Metadata": `bucketName ${b64("chunks")},objectName ${b64("chunked.txt")},contentType ${b64("text/plain")}`,
      }),
    });
    expect(createRes.status).toBe(201);

    const location = createRes.headers.get("Location")!;
    expect(location).toBeTruthy();

    // PATCH: send chunk 1
    const patch1 = await SELF.fetch(location, {
      method: "PATCH",
      headers: tusHeaders(token, {
        "Upload-Offset": "0",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: chunk1,
    });
    expect(patch1.status).toBe(204);
    expect(patch1.headers.get("Upload-Offset")).toBe(chunk1.length.toString());
    expect(patch1.headers.get("Tus-Complete")).toBeNull(); // not complete yet

    // PATCH: send chunk 2
    const patch2 = await SELF.fetch(location, {
      method: "PATCH",
      headers: tusHeaders(token, {
        "Upload-Offset": chunk1.length.toString(),
        "Content-Type": "application/offset+octet-stream",
      }),
      body: chunk2,
    });
    expect(patch2.status).toBe(204);
    expect(patch2.headers.get("Upload-Offset")).toBe(data.length.toString());
    expect(patch2.headers.get("Tus-Complete")).toBe("1");

    // Verify: download the object via the bucket API
    const download = await SELF.fetch(`${BASE}/object/chunks/chunked.txt`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(download.status).toBe(200);
    expect(await download.text()).toBe(data);
  });

  it("uploads a small file in single POST (creation-with-upload)", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "single-post");

    const data = "Small file via creation-with-upload";

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": data.length.toString(),
        "Upload-Metadata": `bucketName ${b64("single-post")},objectName ${b64("small.txt")},contentType ${b64("text/plain")}`,
        "Content-Type": "application/offset+octet-stream",
      }),
      body: data,
    });

    expect(res.status).toBe(201);
    expect(res.headers.get("Tus-Complete")).toBe("1");
    expect(res.headers.get("Upload-Offset")).toBe(data.length.toString());

    // Verify: download
    const download = await SELF.fetch(`${BASE}/object/single-post/small.txt`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(download.status).toBe(200);
    expect(await download.text()).toBe(data);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 4. HEAD — Check upload status
// ═════════════════════════════════════════════════════════════════════

describe("TUS HEAD (check offset)", () => {
  it("returns current offset and length", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "head-test");

    const data = "0123456789";

    // Create upload
    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": data.length.toString(),
        "Upload-Metadata": `bucketName ${b64("head-test")},objectName ${b64("offset-check.bin")}`,
      }),
    });
    expect(createRes.status).toBe(201);
    const location = createRes.headers.get("Location")!;

    // HEAD before any patches
    const head1 = await SELF.fetch(location, {
      method: "HEAD",
      headers: tusHeaders(token),
    });
    expect(head1.status).toBe(200);
    expect(head1.headers.get("Upload-Offset")).toBe("0");
    expect(head1.headers.get("Upload-Length")).toBe("10");
    expect(head1.headers.get("Cache-Control")).toBe("no-store");

    // Send half the data
    await SELF.fetch(location, {
      method: "PATCH",
      headers: tusHeaders(token, {
        "Upload-Offset": "0",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: data.slice(0, 5),
    });

    // HEAD after partial upload
    const head2 = await SELF.fetch(location, {
      method: "HEAD",
      headers: tusHeaders(token),
    });
    expect(head2.status).toBe(200);
    expect(head2.headers.get("Upload-Offset")).toBe("5");
    expect(head2.headers.get("Upload-Length")).toBe("10");
  });

  it("returns 404 for non-existent upload", async () => {
    const token = await createSession("u/alice");

    const res = await SELF.fetch(`${BASE}/upload/resumable/tu_nonexistent`, {
      method: "HEAD",
      headers: tusHeaders(token),
    });
    expect(res.status).toBe(404);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 5. Offset mismatch
// ═════════════════════════════════════════════════════════════════════

describe("TUS offset mismatch", () => {
  it("returns 409 when client offset does not match server", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "mismatch");

    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("mismatch")},objectName ${b64("mis.bin")}`,
      }),
    });
    const location = createRes.headers.get("Location")!;

    // Try patching with wrong offset
    const patch = await SELF.fetch(location, {
      method: "PATCH",
      headers: tusHeaders(token, {
        "Upload-Offset": "50",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: "data",
    });
    expect(patch.status).toBe(409);
    expect(patch.headers.get("Upload-Offset")).toBe("0"); // server tells correct offset
  });
});

// ═════════════════════════════════════════════════════════════════════
// 6. DELETE — Cancel upload
// ═════════════════════════════════════════════════════════════════════

describe("TUS DELETE (cancel)", () => {
  it("cancels an in-progress upload", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "cancel-test");

    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("cancel-test")},objectName ${b64("cancelled.bin")}`,
      }),
    });
    const location = createRes.headers.get("Location")!;

    // Send some data
    await SELF.fetch(location, {
      method: "PATCH",
      headers: tusHeaders(token, {
        "Upload-Offset": "0",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: "partial data",
    });

    // DELETE
    const delRes = await SELF.fetch(location, {
      method: "DELETE",
      headers: tusHeaders(token),
    });
    expect(delRes.status).toBe(204);

    // HEAD should now 404
    const headRes = await SELF.fetch(location, {
      method: "HEAD",
      headers: tusHeaders(token),
    });
    expect(headRes.status).toBe(404);
  });

  it("returns 404 when deleting non-existent upload", async () => {
    const token = await createSession("u/alice");

    const res = await SELF.fetch(`${BASE}/upload/resumable/tu_doesnotexist`, {
      method: "DELETE",
      headers: tusHeaders(token),
    });
    expect(res.status).toBe(404);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 7. Tenant isolation
// ═════════════════════════════════════════════════════════════════════

describe("TUS tenant isolation", () => {
  it("bob cannot HEAD or PATCH alice's upload", async () => {
    const aliceToken = await createSession("u/alice");
    const bobToken = await createSession("u/bob");
    await createBucket(aliceToken, "alice-tus");

    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(aliceToken, {
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("alice-tus")},objectName ${b64("private.bin")}`,
      }),
    });
    const location = createRes.headers.get("Location")!;

    // Bob tries HEAD
    const headRes = await SELF.fetch(location, {
      method: "HEAD",
      headers: tusHeaders(bobToken),
    });
    expect(headRes.status).toBe(404);

    // Bob tries PATCH
    const patchRes = await SELF.fetch(location, {
      method: "PATCH",
      headers: tusHeaders(bobToken, {
        "Upload-Offset": "0",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: "hack",
    });
    expect(patchRes.status).toBe(404);

    // Bob tries DELETE
    const delRes = await SELF.fetch(location, {
      method: "DELETE",
      headers: tusHeaders(bobToken),
    });
    expect(delRes.status).toBe(404);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 8. Content-Type validation on PATCH
// ═════════════════════════════════════════════════════════════════════

describe("TUS Content-Type validation", () => {
  it("rejects PATCH with wrong Content-Type", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "ct-test");

    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("ct-test")},objectName ${b64("wrong-ct.bin")}`,
      }),
    });
    const location = createRes.headers.get("Location")!;

    const patchRes = await SELF.fetch(location, {
      method: "PATCH",
      headers: tusHeaders(token, {
        "Upload-Offset": "0",
        "Content-Type": "application/json",
      }),
      body: "data",
    });
    expect(patchRes.status).toBe(415);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 9. Size limit enforcement
// ═════════════════════════════════════════════════════════════════════

describe("TUS size limits", () => {
  it("rejects upload exceeding bucket size limit", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "limited", { file_size_limit: 50 });

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("limited")},objectName ${b64("too-big.bin")}`,
      }),
    });
    expect(res.status).toBe(413);
  });

  it("rejects chunk that would exceed declared length", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "overshoot");

    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "5",
        "Upload-Metadata": `bucketName ${b64("overshoot")},objectName ${b64("over.bin")}`,
      }),
    });
    const location = createRes.headers.get("Location")!;

    const patchRes = await SELF.fetch(location, {
      method: "PATCH",
      headers: tusHeaders(token, {
        "Upload-Offset": "0",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: "too long data",
    });
    expect(patchRes.status).toBe(413);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 10. Expiration
// ═════════════════════════════════════════════════════════════════════

describe("TUS expiration", () => {
  it("returns 410 for expired uploads", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "expiry-test");

    // Create an upload, then manually expire it
    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("expiry-test")},objectName ${b64("expired.bin")}`,
      }),
    });
    const location = createRes.headers.get("Location")!;
    const uploadId = location.split("/").pop()!;

    // Manually expire the upload
    await env.DB.prepare("UPDATE tus_uploads SET expires_at = ? WHERE id = ?")
      .bind(Date.now() - 1000, uploadId)
      .run();

    // HEAD should return 410
    const headRes = await SELF.fetch(location, {
      method: "HEAD",
      headers: tusHeaders(token),
    });
    expect(headRes.status).toBe(410);

    // PATCH should return 410
    const patchRes = await SELF.fetch(location, {
      method: "PATCH",
      headers: tusHeaders(token, {
        "Upload-Offset": "0",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: "data",
    });
    expect(patchRes.status).toBe(410);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 11. Upsert behavior
// ═════════════════════════════════════════════════════════════════════

describe("TUS upsert", () => {
  it("rejects duplicate path without X-Upsert", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "upsert-test");

    // Upload object via regular API
    await SELF.fetch(`${BASE}/object/upsert-test/existing.txt`, {
      method: "PUT",
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "text/plain" },
      body: "original",
    });

    // TUS create should fail (object exists)
    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "10",
        "Upload-Metadata": `bucketName ${b64("upsert-test")},objectName ${b64("existing.txt")}`,
      }),
    });
    expect(res.status).toBe(409);
  });

  it("allows overwrite with X-Upsert: true", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "upsert-ok");

    // Upload original
    await SELF.fetch(`${BASE}/object/upsert-ok/replace-me.txt`, {
      method: "PUT",
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "text/plain" },
      body: "original",
    });

    const newData = "replaced via TUS";

    // TUS create with X-Upsert
    const createRes = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": newData.length.toString(),
        "Upload-Metadata": `bucketName ${b64("upsert-ok")},objectName ${b64("replace-me.txt")},contentType ${b64("text/plain")}`,
        "X-Upsert": "true",
        "Content-Type": "application/offset+octet-stream",
      }),
      body: newData,
    });
    expect(createRes.status).toBe(201);
    expect(createRes.headers.get("Tus-Complete")).toBe("1");

    // Verify replaced content
    const download = await SELF.fetch(`${BASE}/object/upsert-ok/replace-me.txt`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(await download.text()).toBe(newData);
  });
});

// ═════════════════════════════════════════════════════════════════════
// 12. MIME type restriction
// ═════════════════════════════════════════════════════════════════════

describe("TUS MIME type restriction", () => {
  it("rejects disallowed MIME types", async () => {
    const token = await createSession("u/alice");
    await createBucket(token, "images-only", { allowed_mime_types: ["image/png", "image/jpeg"] });

    const res = await SELF.fetch(`${BASE}/upload/resumable`, {
      method: "POST",
      headers: tusHeaders(token, {
        "Upload-Length": "100",
        "Upload-Metadata": `bucketName ${b64("images-only")},objectName ${b64("hack.exe")},contentType ${b64("application/x-executable")}`,
      }),
    });
    expect(res.status).toBe(400);
    const body = await res.json() as any;
    expect(body.error.message).toContain("MIME type");
  });
});
