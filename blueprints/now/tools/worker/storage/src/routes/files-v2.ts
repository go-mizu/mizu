import { z } from "@hono/zod-openapi";
import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import type { StorageEngine } from "../storage/engine";
import { auth } from "../middleware/auth";
import { err } from "../lib/error";
import { wildcardPath, validatePath } from "../lib/path";
import { mimeFromName } from "../lib/mime";
import { audit } from "../lib/audit";
import { shareToken } from "../lib/id";
import { invalidateCache, getCachedNames } from "./find";
import { checkRateLimit, rateLimitResponse } from "../middleware/rate-limit";

type C = Context<{ Bindings: Env; Variables: Variables }>;

function checkPrefix(c: C, path: string): Response | null {
  const pfx = c.get("prefix");
  if (pfx && !path.startsWith(pfx)) return err(c, "forbidden", "Path not allowed for this token");
  return null;
}

function engine(c: C): StorageEngine {
  return c.get("engine");
}

// ── GET /files ──────────────────────────────────────────────────────
async function listFiles(c: C) {
  const actor = c.get("actor");
  const prefix = c.req.query("prefix") || "";

  const pfx = c.get("prefix");
  if (pfx && prefix && !prefix.startsWith(pfx)) return err(c, "forbidden", "Path not allowed");

  const limit = Math.min(parseInt(c.req.query("limit") || "200", 10), 1000);
  const offset = parseInt(c.req.query("offset") || "0", 10);

  const result = await engine(c).list(actor, { prefix, limit, offset });

  return c.json({ prefix: prefix || "/", entries: result.entries, truncated: result.truncated });
}

// ── GET /files/search ───────────────────────────────────────────────
async function searchFiles(c: C) {
  const q = c.req.query("q")?.trim();
  if (!q) return err(c, "bad_request", "q parameter required");
  const query = q.toLowerCase();
  const limit = Math.min(parseInt(c.req.query("limit") || "50", 10), 200);

  const actor = c.get("actor");
  const prefix = c.get("prefix");

  const names = await getCachedNames(engine(c), actor);
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
  const stats = await engine(c).stats(actor);
  return c.json(stats);
}

// ── POST /files/move ────────────────────────────────────────────────
async function moveFile(c: C) {
  const body = await c.req.json<{ from?: string; to?: string; message?: string }>();
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

  let result;
  try {
    result = await engine(c).move(actor, from, to, body.message);
  } catch (e: any) {
    if (e?.message?.includes("not found")) return err(c, "not_found", "Source file not found");
    throw e;
  }

  invalidateCache(actor);
  audit(c, "mv", `${from} → ${to}`);
  return c.json({ from, to, tx: result.tx, time: result.time });
}

// ── POST /files/share ───────────────────────────────────────────────
const DEFAULT_TTL = 3600;
const MAX_TTL = 30 * 86400;

async function shareFile(c: C) {
  // Rate limit: 50 share links per hour per actor
  const actor = c.get("actor");
  const rl = await checkRateLimit(c.env.DB, { endpoint: "files/share", limit: 50, windowMs: 3600_000 }, actor);
  if (!rl.allowed) return rateLimitResponse(c);

  const body = await c.req.json<{ path?: string; ttl?: number; expires_in?: number }>();
  const path = body.path || "";

  if (!path) return err(c, "bad_request", "path is required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const meta = await engine(c).head(actor, path);
  if (!meta) return err(c, "not_found", "File not found");

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
async function mkdirHandler(c: C) {
  const body = await c.req.json<{ path?: string }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path || !path.endsWith("/")) return err(c, "bad_request", "Path must end with /");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  // Mkdir: write a zero-byte blob
  await engine(c).write(actor, path, new ArrayBuffer(0), "application/x-directory", `mkdir ${path}`);

  return c.json({ path, created: true });
}

// ── POST /files/uploads ─────────────────────────────────────────────
const UPLOAD_EXPIRES = 3600;

async function initiateUpload(c: C) {
  // Rate limit: 200 uploads per hour per actor
  const actor = c.get("actor");
  const rl = await checkRateLimit(c.env.DB, { endpoint: "files/uploads", limit: 200, windowMs: 3600_000 }, actor);
  if (!rl.allowed) return rateLimitResponse(c);

  const body = await c.req.json<{ path?: string; content_type?: string; content_hash?: string }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path || path.endsWith("/")) return err(c, "bad_request", "File path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const name = path.split("/").pop()!;
  const contentType = body.content_type || mimeFromName(name);
  const contentHash = body.content_hash;

  // Client-side SHA-256: check for instant dedup
  if (contentHash) {
    const existingSize = await engine(c).blobExists(actor, contentHash);
    if (existingSize !== null) {
      // Blob already exists — record metadata without upload
      const result = await engine(c).confirmUpload(actor, path, undefined, contentHash);
      invalidateCache(actor);
      audit(c, "write", path);
      return c.json({ path, name, size: result.size, tx: result.tx, time: result.time, deduplicated: true });
    }
  }

  const url = await engine(c).presignUpload(actor, path, contentType, UPLOAD_EXPIRES, contentHash);

  return c.json({ url, content_type: contentType, content_hash: contentHash || undefined, expires_in: UPLOAD_EXPIRES });
}

// ── POST /files/uploads/complete ────────────────────────────────────
async function completeUpload(c: C) {
  const body = await c.req.json<{ path?: string; message?: string; content_hash?: string }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path) return err(c, "bad_request", "path is required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  let result;
  try {
    result = await engine(c).confirmUpload(actor, path, body.message, body.content_hash);
  } catch (e: any) {
    if (e?.message?.includes("not found")) return err(c, "not_found", "Upload not found in storage");
    throw e;
  }

  const name = path.split("/").pop()!;
  invalidateCache(actor);
  audit(c, "write", path);

  return c.json({ path, name, size: result.size, tx: result.tx, time: result.time });
}

// ── POST /files/uploads/multipart ───────────────────────────────────
async function initiateMultipart(c: C) {
  // Rate limit: 50 multipart uploads per hour per actor
  const actor = c.get("actor");
  const rl = await checkRateLimit(c.env.DB, { endpoint: "files/uploads/multipart", limit: 50, windowMs: 3600_000 }, actor);
  if (!rl.allowed) return rateLimitResponse(c);

  const body = await c.req.json<{ path?: string; content_type?: string; part_count?: number; content_hash?: string }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path || path.endsWith("/")) return err(c, "bad_request", "File path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const name = path.split("/").pop()!;
  const contentType = body.content_type || mimeFromName(name);
  const partCount = Math.min(body.part_count || 1, 10000);
  const contentHash = body.content_hash;

  // Client-side SHA-256: check for instant dedup
  if (contentHash) {
    const existingSize = await engine(c).blobExists(actor, contentHash);
    if (existingSize !== null) {
      const result = await engine(c).confirmUpload(actor, path, undefined, contentHash);
      invalidateCache(actor);
      audit(c, "write", path);
      return c.json({ path, name, size: result.size, tx: result.tx, time: result.time, deduplicated: true });
    }
  }

  const result = await engine(c).initiateMultipart(actor, path, contentType, partCount, contentHash);

  return c.json({
    upload_id: result.upload_id,
    key: path,
    content_type: contentType,
    content_hash: contentHash || undefined,
    part_urls: result.part_urls,
    expires_in: result.expires_in,
  });
}

// ── POST /files/uploads/multipart/complete ──────────────────────────
async function completeMultipart(c: C) {
  const body = await c.req.json<{
    path?: string;
    upload_id?: string;
    parts?: { part_number: number; etag: string }[];
    message?: string;
    content_hash?: string;
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
  const result = await engine(c).completeMultipart(actor, path, body.upload_id, body.parts, body.message, body.content_hash);

  const name = path.split("/").pop()!;
  invalidateCache(actor);
  audit(c, "write", path);

  return c.json({ path, name, size: result.size, tx: result.tx, time: result.time });
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
  await engine(c).abortMultipart(actor, path, body.upload_id);

  return c.json({ aborted: true });
}

// ── GET /files/{path} — download ────────────────────────────────────
async function downloadFile(c: C) {
  const path = wildcardPath(c, "/files/");
  if (!path) return err(c, "bad_request", "File path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const meta = await engine(c).head(actor, path);
  if (!meta) return err(c, "not_found", "File not found");

  const url = await engine(c).presignRead(actor, path);
  if (!url) return err(c, "not_configured", "Presigned URLs not configured");

  audit(c, "read", path);

  const accept = c.req.header("Accept") || "";
  if (accept.includes("application/json")) {
    return c.json({
      url,
      size: meta.size,
      type: meta.type,
      tx: meta.tx,
      time: meta.tx_time,
      expires_in: 3600,
    });
  }

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
  const meta = await engine(c).head(actor, path);
  if (!meta) return c.body(null, 404);

  return c.body(null, 200, {
    "Content-Type": meta.type || "application/octet-stream",
    "Content-Length": meta.size.toString(),
    "X-Tx": String(meta.tx),
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
  const result = await engine(c).delete(actor, [path]);

  invalidateCache(actor);
  audit(c, "rm", path);
  return c.json({ deleted: result.deleted, tx: result.tx, time: result.time });
}

// ── GET /files/log ──────────────────────────────────────────────────
async function logHandler(c: C) {
  const actor = c.get("actor");
  const path = c.req.query("path");
  const since_tx = c.req.query("since_tx") ? parseInt(c.req.query("since_tx")!, 10) : undefined;
  const before_tx = c.req.query("before_tx") ? parseInt(c.req.query("before_tx")!, 10) : undefined;
  const limit = Math.min(parseInt(c.req.query("limit") || "50", 10), 500);

  const events = await engine(c).log(actor, { path, since_tx, before_tx, limit });
  return c.json({ events });
}

// ── Wildcard dispatcher for /files/* ────────────────────────────────
async function filesWildcard(c: C) {
  const method = c.req.method;
  if (method === "GET") return downloadFile(c);
  if (method === "HEAD") return headFile(c);
  if (method === "DELETE") return deleteHandler(c);
  return err(c, "bad_request", "Method not allowed");
}

// ── POST /files/upload-url ─── fetch a URL and store it ─────────────
async function uploadFromUrl(c: C) {
  const actor = c.get("actor");
  const body = await c.req.json<{ url: string; path?: string }>();

  if (!body.url) return err(c, "bad_request", "url is required");

  let parsed: URL;
  try {
    parsed = new URL(body.url);
  } catch {
    return err(c, "bad_request", "Invalid URL");
  }

  if (!["http:", "https:"].includes(parsed.protocol)) {
    return err(c, "bad_request", "Only HTTP/HTTPS URLs supported");
  }

  const response = await fetch(body.url, {
    headers: { "User-Agent": "Storage/1.0" },
    redirect: "follow",
  });

  if (!response.ok) {
    return err(c, "bad_request", `Failed to fetch URL: ${response.status}`);
  }

  const contentType = response.headers.get("content-type") || "application/octet-stream";
  const data = await response.arrayBuffer();

  let targetPath = body.path;
  if (!targetPath) {
    const segments = parsed.pathname.split("/").filter(Boolean);
    targetPath = segments.pop() || "download";
  }

  const pathErr = validatePath(targetPath);
  if (pathErr) return err(c, "bad_request", pathErr);

  const pfxErr = checkPrefix(c, targetPath);
  if (pfxErr) return pfxErr;

  const result = await engine(c).write(
    actor,
    targetPath,
    data,
    contentType.split(";")[0].trim(),
    `Upload from URL: ${body.url}`,
  );

  audit(c, "write", targetPath);
  return c.json({
    path: targetPath,
    size: data.byteLength,
    type: contentType,
    tx: result.tx,
    time: result.time,
  }, 201);
}

// ── Registration ────────────────────────────────────────────────────
export function register(app: App) {
  app.get("/files", auth, listFiles);
  app.get("/files/search", auth, searchFiles);
  app.get("/files/stats", auth, statsHandler);
  app.get("/files/log", auth, logHandler);
  app.post("/files/move", auth, moveFile);
  app.post("/files/mkdir", auth, mkdirHandler);
  app.post("/files/share", auth, shareFile);
  app.post("/files/upload-url", auth, uploadFromUrl);
  app.post("/files/uploads", auth, initiateUpload);
  app.post("/files/uploads/complete", auth, completeUpload);
  app.post("/files/uploads/multipart", auth, initiateMultipart);
  app.post("/files/uploads/multipart/complete", auth, completeMultipart);
  app.post("/files/uploads/multipart/abort", auth, abortMultipart);
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
                tx: z.number().int().optional().openapi({ description: "Last-write transaction number" }),
                tx_time: z.number().int().optional().openapi({ description: "Timestamp of last-write tx" }),
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
    method: "get",
    path: "/files/log",
    summary: "View event log",
    tags: filesTags,
    security: sec,
    request: {
      query: z.object({
        path: z.string().optional().openapi({ description: "Filter events by file path" }),
        since_tx: z.coerce.number().int().optional().openapi({ description: "Return events after this tx number" }),
        limit: z.coerce.number().int().default(50).optional(),
      }),
    },
    responses: {
      200: {
        description: "Event log",
        content: {
          "application/json": {
            schema: z.object({
              events: z.array(z.object({
                tx: z.number().int(),
                action: z.string(),
                path: z.string(),
                size: z.number().int(),
                msg: z.string().nullable(),
                ts: z.number().int(),
              })),
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
      body: {
        content: {
          "application/json": {
            schema: z.object({
              from: z.string(),
              to: z.string(),
              message: z.string().optional().openapi({ description: "Commit message for the move" }),
            }),
          },
        },
      },
    },
    responses: {
      200: {
        description: "Moved",
        content: {
          "application/json": {
            schema: z.object({ from: z.string(), to: z.string(), tx: z.number().int(), time: z.number().int() }),
          },
        },
      },
    },
  });

  app.openAPIRegistry.registerPath({
    method: "post",
    path: "/files/share",
    summary: "Share a file",
    tags: ["sharing"],
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
      302: { description: "Redirect to presigned URL" },
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
      200: {
        description: "Delete result",
        content: {
          "application/json": {
            schema: z.object({
              deleted: z.number().int(),
              tx: z.number().int(),
              time: z.number().int(),
            }),
          },
        },
      },
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
      body: {
        content: {
          "application/json": {
            schema: z.object({
              path: z.string(),
              message: z.string().optional().openapi({ description: "Commit message" }),
            }),
          },
        },
      },
    },
    responses: {
      200: {
        description: "File metadata",
        content: {
          "application/json": {
            schema: z.object({
              path: z.string(),
              name: z.string(),
              size: z.number().int(),
              tx: z.number().int(),
              time: z.number().int(),
            }),
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
              message: z.string().optional().openapi({ description: "Commit message" }),
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
            schema: z.object({
              path: z.string(),
              name: z.string(),
              size: z.number().int(),
              tx: z.number().int(),
              time: z.number().int(),
            }),
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
