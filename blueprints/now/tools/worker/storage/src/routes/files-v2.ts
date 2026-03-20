import { z } from "@hono/zod-openapi";
import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { err } from "../lib/error";
import { wildcardPath, validatePath } from "../lib/path";
import { mimeFromName } from "../lib/mime";
import { presignUrl } from "../lib/presign";
import { audit } from "../lib/audit";
import { shareToken } from "../lib/id";
import { invalidateCache, getCachedNames } from "./find";

type C = Context<{ Bindings: Env; Variables: Variables }>;

function checkPrefix(c: C, path: string): Response | null {
  const pfx = c.get("prefix");
  if (pfx && !path.startsWith(pfx)) return err(c, "forbidden", "Path not allowed for this token");
  return null;
}

function r2Config(c: C) {
  const endpoint = c.env.R2_ENDPOINT;
  const accessKeyId = c.env.R2_ACCESS_KEY_ID;
  const secretAccessKey = c.env.R2_SECRET_ACCESS_KEY;
  if (!endpoint || !accessKeyId || !secretAccessKey) return null;
  return { endpoint, accessKeyId, secretAccessKey, bucket: c.env.R2_BUCKET_NAME || "storage-files" };
}

// ── GET /files ──────────────────────────────────────────────────────
// List files and folders. Query params: ?prefix=docs/&limit=200&offset=0
async function listFiles(c: C) {
  const actor = c.get("actor");
  const prefix = c.req.query("prefix") || "";

  const pfx = c.get("prefix");
  if (pfx && prefix && !prefix.startsWith(pfx)) return err(c, "forbidden", "Path not allowed");

  const limit = Math.min(parseInt(c.req.query("limit") || "200", 10), 1000);
  const offset = parseInt(c.req.query("offset") || "0", 10);

  const { results } = await c.env.DB.prepare(
    "SELECT path, name, size, type, updated_at FROM files WHERE owner = ? AND path LIKE ? ORDER BY path LIMIT ? OFFSET ?",
  )
    .bind(actor, `${prefix}%`, limit + 1, offset)
    .all();

  const rows = results || [];
  const truncated = rows.length > limit;
  if (truncated) rows.pop();

  const entries: { name: string; type: string; size?: number; updated_at?: number }[] = [];
  const dirs = new Set<string>();

  for (const row of rows) {
    const relative = (row.path as string).slice(prefix.length);
    const slash = relative.indexOf("/");
    if (slash === -1) {
      entries.push({
        name: relative,
        type: row.type as string,
        size: row.size as number,
        updated_at: row.updated_at as number,
      });
    } else {
      const dir = relative.slice(0, slash + 1);
      if (!dirs.has(dir)) {
        dirs.add(dir);
        entries.push({ name: dir, type: "directory" });
      }
    }
  }

  return c.json({ prefix: prefix || "/", entries, truncated });
}

// ── GET /files/search ───────────────────────────────────────────────
async function searchFiles(c: C) {
  const q = c.req.query("q")?.trim();
  if (!q) return err(c, "bad_request", "q parameter required");
  const query = q.toLowerCase();
  const limit = Math.min(parseInt(c.req.query("limit") || "50", 10), 200);

  const actor = c.get("actor");
  const prefix = c.get("prefix");

  const names = await getCachedNames(c.env.DB, actor);
  const tokens = query.split(/\s+/);
  const hits: { path: string; name: string; score: number }[] = [];

  for (const { path, name } of names) {
    if (prefix && !path.startsWith(prefix)) continue;

    const lp = path.toLowerCase();
    const ln = name.toLowerCase();
    if (!tokens.every((t) => lp.includes(t))) continue;

    let score = 0;
    for (const t of tokens) {
      if (ln === t) score += 10;
      else if (ln.includes(t)) score += 5;
      else score += 1;
    }
    hits.push({ path, name, score });
  }

  hits.sort((a, b) => b.score - a.score);
  return c.json({
    query: q,
    results: hits.slice(0, limit).map(({ path, name }) => ({ path, name })),
  });
}

// ── GET /files/stats ────────────────────────────────────────────────
async function statsHandler(c: C) {
  const actor = c.get("actor");
  const row = await c.env.DB.prepare(
    "SELECT COUNT(*) as files, COALESCE(SUM(size), 0) as bytes FROM files WHERE owner = ?",
  )
    .bind(actor)
    .first<{ files: number; bytes: number }>();

  return c.json({ files: row?.files || 0, bytes: row?.bytes || 0 });
}

// ── POST /files/move ────────────────────────────────────────────────
async function moveFile(c: C) {
  const body = await c.req.json<{ from?: string; to?: string }>();
  const from = body.from || "";
  const to = body.to || "";

  if (!from || !to) return err(c, "bad_request", "from and to are required");
  const fromErr = validatePath(from);
  if (fromErr) return err(c, "bad_request", `from: ${fromErr}`);
  const toErr = validatePath(to);
  if (toErr) return err(c, "bad_request", `to: ${toErr}`);

  const actor = c.get("actor");
  const prefix = c.get("prefix");
  if (prefix && (!from.startsWith(prefix) || !to.startsWith(prefix))) {
    return err(c, "forbidden", "Path not allowed");
  }

  const fromKey = `${actor}/${from}`;
  const obj = await c.env.BUCKET.get(fromKey);
  if (!obj) return err(c, "not_found", "Source not found");

  await c.env.BUCKET.put(`${actor}/${to}`, obj.body, {
    httpMetadata: obj.httpMetadata,
    customMetadata: obj.customMetadata,
  });
  await c.env.BUCKET.delete(fromKey);

  const name = to.split("/").pop()!;
  const type = obj.httpMetadata?.contentType || mimeFromName(name);
  const now = Date.now();

  await c.env.DB.batch([
    c.env.DB.prepare("DELETE FROM files WHERE owner = ? AND path = ?").bind(actor, from),
    c.env.DB.prepare(
      "INSERT INTO files (owner, path, name, size, type, updated_at) VALUES (?, ?, ?, ?, ?, ?) " +
        "ON CONFLICT (owner, path) DO UPDATE SET name = excluded.name, size = excluded.size, type = excluded.type, updated_at = excluded.updated_at",
    ).bind(actor, to, name, obj.size, type, now),
  ]);

  invalidateCache(actor);
  audit(c, "mv", `${from} → ${to}`);
  return c.json({ from, to });
}

// ── POST /files/share ───────────────────────────────────────────────
const DEFAULT_TTL = 3600;
const MAX_TTL = 7 * 86400;

async function shareFile(c: C) {
  const body = await c.req.json<{ path?: string; ttl?: number; expires_in?: number }>();
  const path = body.path || "";

  if (!path) return err(c, "bad_request", "path is required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const head = await c.env.BUCKET.head(`${actor}/${path}`);
  if (!head) return err(c, "not_found", "File not found");

  const ttl = Math.min(Math.max(body.ttl || body.expires_in || DEFAULT_TTL, 60), MAX_TTL);
  const now = Date.now();
  const expiresAt = now + ttl * 1000;
  const token = shareToken();

  await c.env.DB.prepare(
    "INSERT INTO share_links (token, actor, path, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
  )
    .bind(token, actor, path, expiresAt, now)
    .run();

  const origin = new URL(c.req.url).origin;
  audit(c, "share", path);

  return c.json({ url: `${origin}/s/${token}`, token, expires_at: expiresAt, ttl }, 201);
}

// ── POST /files/mkdir ───────────────────────────────────────────────
// Creates a folder marker (zero-byte object in R2).
async function mkdirHandler(c: C) {
  const body = await c.req.json<{ path?: string }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path || !path.endsWith("/")) return err(c, "bad_request", "Path must end with /");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  await c.env.BUCKET.put(`${actor}/${path}`, new Uint8Array(0));

  return c.json({ path, created: true });
}

// ── POST /files/uploads ─────────────────────────────────────────────
// Initiate a presigned upload — returns a signed PUT URL for direct R2 upload.
const UPLOAD_EXPIRES = 3600;

async function initiateUpload(c: C) {
  const cfg = r2Config(c);
  if (!cfg) return err(c, "not_configured", "Presigned URLs not configured");

  const body = await c.req.json<{ path?: string; content_type?: string }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path || path.endsWith("/")) return err(c, "bad_request", "File path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const name = path.split("/").pop()!;
  const contentType = body.content_type || mimeFromName(name);
  const key = `${actor}/${path}`;

  const url = await presignUrl({
    method: "PUT",
    key,
    bucket: cfg.bucket,
    endpoint: cfg.endpoint,
    accessKeyId: cfg.accessKeyId,
    secretAccessKey: cfg.secretAccessKey,
    expiresIn: UPLOAD_EXPIRES,
    contentType,
  });

  return c.json({ url, content_type: contentType, expires_in: UPLOAD_EXPIRES });
}

// ── POST /files/uploads/complete ────────────────────────────────────
async function completeUpload(c: C) {
  const body = await c.req.json<{ path?: string }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path) return err(c, "bad_request", "path is required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const key = `${actor}/${path}`;

  const head = await c.env.BUCKET.head(key);
  if (!head) return err(c, "not_found", "Upload not found in storage — did the upload complete?");

  const name = path.split("/").pop()!;
  const contentType = head.httpMetadata?.contentType || mimeFromName(name);
  const now = Date.now();

  await c.env.DB.prepare(
    "INSERT INTO files (owner, path, name, size, type, updated_at) VALUES (?, ?, ?, ?, ?, ?) " +
      "ON CONFLICT (owner, path) DO UPDATE SET size = excluded.size, type = excluded.type, updated_at = excluded.updated_at",
  )
    .bind(actor, path, name, head.size, contentType, now)
    .run();

  invalidateCache(actor);
  audit(c, "write", path);

  return c.json({ path, name, size: head.size, type: contentType, updated_at: now });
}

// ── POST /files/uploads/multipart ───────────────────────────────────
async function initiateMultipart(c: C) {
  const cfg = r2Config(c);
  if (!cfg) return err(c, "not_configured", "Presigned URLs not configured");

  const body = await c.req.json<{ path?: string; content_type?: string; part_count?: number }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path || path.endsWith("/")) return err(c, "bad_request", "File path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const name = path.split("/").pop()!;
  const contentType = body.content_type || mimeFromName(name);
  const key = `${actor}/${path}`;

  const mpu = await c.env.BUCKET.createMultipartUpload(key, {
    httpMetadata: { contentType },
  });

  const partCount = Math.min(body.part_count || 1, 10000);
  const partUrls: string[] = [];
  for (let i = 1; i <= partCount; i++) {
    const url = await presignUrl({
      method: "PUT",
      key,
      bucket: cfg.bucket,
      endpoint: cfg.endpoint,
      accessKeyId: cfg.accessKeyId,
      secretAccessKey: cfg.secretAccessKey,
      expiresIn: 86400,
      queryParams: { partNumber: String(i), uploadId: mpu.uploadId },
    });
    partUrls.push(url);
  }

  return c.json({
    upload_id: mpu.uploadId,
    key: path,
    content_type: contentType,
    part_urls: partUrls,
    expires_in: 86400,
  });
}

// ── POST /files/uploads/multipart/complete ──────────────────────────
async function completeMultipart(c: C) {
  const body = await c.req.json<{
    path?: string;
    upload_id?: string;
    parts?: { part_number: number; etag: string }[];
  }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path) return err(c, "bad_request", "path is required");
  if (!body.upload_id) return err(c, "bad_request", "upload_id is required");
  if (!body.parts?.length) return err(c, "bad_request", "parts is required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const key = `${actor}/${path}`;

  const mpu = c.env.BUCKET.resumeMultipartUpload(key, body.upload_id);
  const uploaded = await mpu.complete(
    body.parts.map((p) => ({
      partNumber: p.part_number,
      etag: p.etag,
    })),
  );

  const name = path.split("/").pop()!;
  const contentType = uploaded.httpMetadata?.contentType || mimeFromName(name);
  const now = Date.now();

  const head = await c.env.BUCKET.head(key);
  const size = head?.size ?? 0;

  await c.env.DB.prepare(
    "INSERT INTO files (owner, path, name, size, type, updated_at) VALUES (?, ?, ?, ?, ?, ?) " +
      "ON CONFLICT (owner, path) DO UPDATE SET size = excluded.size, type = excluded.type, updated_at = excluded.updated_at",
  )
    .bind(actor, path, name, size, contentType, now)
    .run();

  invalidateCache(actor);
  audit(c, "write", path);

  return c.json({ path, name, size, type: contentType, updated_at: now });
}

// ── POST /files/uploads/multipart/abort ─────────────────────────────
async function abortMultipart(c: C) {
  const body = await c.req.json<{ path?: string; upload_id?: string }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path) return err(c, "bad_request", "path is required");
  if (!body.upload_id) return err(c, "bad_request", "upload_id is required");
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const key = `${actor}/${path}`;

  const mpu = c.env.BUCKET.resumeMultipartUpload(key, body.upload_id);
  await mpu.abort();

  return c.json({ aborted: true });
}

// ── GET /files/{path} — download ────────────────────────────────────
// Accept: application/json → returns JSON with presigned URL + metadata.
// Otherwise → 302 redirect to presigned R2 URL (browsers, curl).
async function downloadFile(c: C) {
  const cfg = r2Config(c);
  if (!cfg) return err(c, "not_configured", "Presigned URLs not configured");

  const path = wildcardPath(c, "/files/");
  if (!path) return err(c, "bad_request", "File path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const key = `${actor}/${path}`;

  const head = await c.env.BUCKET.head(key);
  if (!head) return err(c, "not_found", "File not found");

  const url = await presignUrl({
    method: "GET",
    key,
    bucket: cfg.bucket,
    endpoint: cfg.endpoint,
    accessKeyId: cfg.accessKeyId,
    secretAccessKey: cfg.secretAccessKey,
    expiresIn: 3600,
  });

  audit(c, "read", path);

  // Programmatic clients (CLI, SDKs) get JSON with the URL
  const accept = c.req.header("Accept") || "";
  if (accept.includes("application/json")) {
    return c.json({
      url,
      size: head.size,
      type: head.httpMetadata?.contentType || "application/octet-stream",
      etag: head.etag,
      expires_in: 3600,
    });
  }

  // Browsers, curl — redirect directly
  return c.redirect(url, 302) as any;
}

// ── HEAD /files/{path} — file metadata ──────────────────────────────
async function headFile(c: C) {
  const path = wildcardPath(c, "/files/");
  if (!path) return c.body(null, 400);
  const pathErr = validatePath(path);
  if (pathErr) return c.body(null, 400);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const obj = await c.env.BUCKET.head(`${actor}/${path}`);
  if (!obj) return c.body(null, 404);

  return c.body(null, 200, {
    "Content-Type": obj.httpMetadata?.contentType || "application/octet-stream",
    "Content-Length": obj.size.toString(),
    "ETag": obj.etag,
  });
}

// ── DELETE /files/{path} — delete file or folder ────────────────────
async function deleteHandler(c: C) {
  const path = wildcardPath(c, "/files/");
  if (!path) return err(c, "bad_request", "Path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");

  if (path.endsWith("/")) {
    const prefix = `${actor}/${path}`;
    let cursor: string | undefined;
    let deleted = 0;
    do {
      const list = await c.env.BUCKET.list({ prefix, cursor, limit: 1000 });
      if (list.objects.length) {
        await c.env.BUCKET.delete(list.objects.map((o) => o.key));
        deleted += list.objects.length;
      }
      cursor = list.truncated ? list.cursor : undefined;
    } while (cursor);

    await c.env.DB.prepare("DELETE FROM files WHERE owner = ? AND path LIKE ?")
      .bind(actor, `${path}%`).run();
    invalidateCache(actor);
    audit(c, "rm", path);
    return c.json({ deleted });
  }

  await c.env.BUCKET.delete(`${actor}/${path}`);
  await c.env.DB.prepare("DELETE FROM files WHERE owner = ? AND path = ?")
    .bind(actor, path).run();
  invalidateCache(actor);
  audit(c, "rm", path);
  return c.json({ deleted: 1 });
}

// ── Wildcard dispatcher for /files/* ────────────────────────────────
// Routes /files/search, /files/stats, /files/uploads etc. are registered
// before this catch-all, so they take priority in Hono's router.
async function filesWildcard(c: C) {
  const method = c.req.method;
  if (method === "GET") return downloadFile(c);
  if (method === "HEAD") return headFile(c);
  if (method === "DELETE") return deleteHandler(c);
  return err(c, "bad_request", "Method not allowed");
}

// ── Registration ────────────────────────────────────────────────────
export function register(app: App) {
  // Specific routes MUST be registered before the wildcard catch-all

  // List
  app.get("/files", auth, listFiles);

  // Search
  app.get("/files/search", auth, searchFiles);

  // Stats
  app.get("/files/stats", auth, statsHandler);

  // Move
  app.post("/files/move", auth, moveFile);

  // Mkdir
  app.post("/files/mkdir", auth, mkdirHandler);

  // Share
  app.post("/files/share", auth, shareFile);

  // Uploads (single)
  app.post("/files/uploads", auth, initiateUpload);
  app.post("/files/uploads/complete", auth, completeUpload);

  // Uploads (multipart)
  app.post("/files/uploads/multipart", auth, initiateMultipart);
  app.post("/files/uploads/multipart/complete", auth, completeMultipart);
  app.post("/files/uploads/multipart/abort", auth, abortMultipart);

  // Wildcard catch-all: GET (download redirect), HEAD (metadata), DELETE
  app.all("/files/*", auth, filesWildcard);

  // ── OpenAPI docs ──────────────────────────────────────────────────
  const pathParam = z.object({ path: z.string().openapi({ description: "File path (e.g. docs/report.pdf)" }) });
  const filesTags = ["files"];
  const sec = [{ bearer: [] as string[] }];

  app.openAPIRegistry.registerPath({
    method: "get",
    path: "/files",
    summary: "List files",
    tags: filesTags,
    security: sec,
    request: {
      query: z.object({
        prefix: z.string().optional().openapi({ description: "Folder prefix (e.g. docs/)" }),
        limit: z.coerce.number().int().default(200).optional(),
        offset: z.coerce.number().int().default(0).optional(),
      }),
    },
    responses: {
      200: {
        description: "Directory listing",
        content: {
          "application/json": {
            schema: z.object({
              prefix: z.string(),
              entries: z.array(z.object({
                name: z.string(),
                type: z.string().openapi({ description: "MIME type or 'directory'" }),
                size: z.number().int().optional(),
                updated_at: z.number().int().optional(),
              })),
              truncated: z.boolean(),
            }),
          },
        },
      },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "get",
    path: "/files/search",
    summary: "Search files",
    tags: filesTags,
    security: sec,
    request: {
      query: z.object({
        q: z.string().openapi({ description: "Search query" }),
        limit: z.coerce.number().int().default(50).optional(),
      }),
    },
    responses: {
      200: {
        description: "Search results",
        content: {
          "application/json": {
            schema: z.object({
              query: z.string(),
              results: z.array(z.object({ path: z.string(), name: z.string() })),
            }),
          },
        },
      },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "get",
    path: "/files/stats",
    summary: "Retrieve storage stats",
    tags: filesTags,
    security: sec,
    responses: {
      200: {
        description: "Usage",
        content: {
          "application/json": {
            schema: z.object({
              files: z.number().int(),
              bytes: z.number().int(),
            }),
          },
        },
      },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "post",
    path: "/files/move",
    summary: "Move a file",
    tags: filesTags,
    security: sec,
    request: {
      body: { content: { "application/json": { schema: z.object({ from: z.string(), to: z.string() }) } } },
    },
    responses: {
      200: { description: "Moved", content: { "application/json": { schema: z.object({ from: z.string(), to: z.string() }) } } },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "post",
    path: "/files/share",
    summary: "Share a file",
    tags: filesTags,
    security: sec,
    request: {
      body: {
        content: {
          "application/json": {
            schema: z.object({
              path: z.string(),
              ttl: z.number().int().optional().openapi({ description: "Seconds (default 3600, max 604800)" }),
            }),
          },
        },
      },
    },
    responses: {
      201: {
        description: "Share link created",
        content: {
          "application/json": {
            schema: z.object({ url: z.string(), token: z.string(), expires_at: z.number().int(), ttl: z.number().int() }),
          },
        },
      },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "get",
    path: "/files/{path}",
    summary: "Retrieve a file",
    tags: filesTags,
    security: sec,
    request: { params: pathParam },
    responses: {
      302: { description: "Redirect to presigned R2 URL" },
      404: { description: "File not found" },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "head",
    path: "/files/{path}",
    summary: "Retrieve file metadata",
    tags: filesTags,
    security: sec,
    request: { params: pathParam },
    responses: {
      200: { description: "Metadata in headers" },
      404: { description: "Not found" },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "delete",
    path: "/files/{path}",
    summary: "Delete a file",
    tags: filesTags,
    security: sec,
    request: { params: pathParam },
    responses: {
      200: { description: "Delete count", content: { "application/json": { schema: z.object({ deleted: z.number().int() }) } } },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "post",
    path: "/files/mkdir",
    summary: "Create a folder",
    tags: filesTags,
    security: sec,
    request: {
      body: {
        content: {
          "application/json": {
            schema: z.object({
              path: z.string().openapi({ example: "docs/drafts/", description: "Folder path (must end with /)" }),
            }),
          },
        },
      },
    },
    responses: {
      200: {
        description: "Folder created",
        content: { "application/json": { schema: z.object({ path: z.string(), created: z.boolean() }) } },
      },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "post",
    path: "/files/uploads",
    summary: "Create an upload",
    tags: ["uploads"],
    security: sec,
    request: {
      body: {
        content: {
          "application/json": {
            schema: z.object({
              path: z.string().openapi({ example: "docs/report.pdf" }),
              content_type: z.string().optional(),
            }),
          },
        },
      },
    },
    responses: {
      200: {
        description: "Presigned upload URL",
        content: {
          "application/json": {
            schema: z.object({ url: z.string(), content_type: z.string(), expires_in: z.number().int() }),
          },
        },
      },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "post",
    path: "/files/uploads/complete",
    summary: "Complete an upload",
    tags: ["uploads"],
    security: sec,
    request: {
      body: { content: { "application/json": { schema: z.object({ path: z.string() }) } } },
    },
    responses: {
      200: {
        description: "File metadata",
        content: {
          "application/json": {
            schema: z.object({ path: z.string(), name: z.string(), size: z.number().int(), type: z.string(), updated_at: z.number().int() }),
          },
        },
      },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "post",
    path: "/files/uploads/multipart",
    summary: "Create a multipart upload",
    tags: ["uploads"],
    security: sec,
    request: {
      body: {
        content: {
          "application/json": {
            schema: z.object({
              path: z.string(),
              content_type: z.string().optional(),
              part_count: z.number().int().optional(),
            }),
          },
        },
      },
    },
    responses: {
      200: {
        description: "Multipart upload initiated",
        content: {
          "application/json": {
            schema: z.object({
              upload_id: z.string(),
              key: z.string(),
              content_type: z.string(),
              part_urls: z.array(z.string()),
              expires_in: z.number().int(),
            }),
          },
        },
      },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "post",
    path: "/files/uploads/multipart/complete",
    summary: "Complete a multipart upload",
    tags: ["uploads"],
    security: sec,
    request: {
      body: {
        content: {
          "application/json": {
            schema: z.object({
              path: z.string(),
              upload_id: z.string(),
              parts: z.array(z.object({ part_number: z.number().int(), etag: z.string() })),
            }),
          },
        },
      },
    },
    responses: {
      200: {
        description: "Upload completed",
        content: {
          "application/json": {
            schema: z.object({ path: z.string(), name: z.string(), size: z.number().int(), type: z.string(), updated_at: z.number().int() }),
          },
        },
      },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "post",
    path: "/files/uploads/multipart/abort",
    summary: "Abort a multipart upload",
    tags: ["uploads"],
    security: sec,
    request: {
      body: {
        content: {
          "application/json": {
            schema: z.object({ path: z.string(), upload_id: z.string() }),
          },
        },
      },
    },
    responses: {
      200: {
        description: "Upload aborted",
        content: { "application/json": { schema: z.object({ aborted: z.boolean() }) } },
      },
    },
  });
}
