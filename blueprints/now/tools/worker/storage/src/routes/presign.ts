import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { wildcardPath, validatePath } from "../lib/path";
import { mimeFromName } from "../lib/mime";
import { presignUrl } from "../lib/presign";
import { err } from "../lib/error";
import { audit } from "../lib/audit";
import { invalidateCache } from "./find";

type C = Context<{ Bindings: Env; Variables: Variables }>;

const UPLOAD_EXPIRES = 3600; // 1 hour
const READ_EXPIRES = 3600;

function r2Config(c: C) {
  const endpoint = c.env.R2_ENDPOINT;
  const accessKeyId = c.env.R2_ACCESS_KEY_ID;
  const secretAccessKey = c.env.R2_SECRET_ACCESS_KEY;
  if (!endpoint || !accessKeyId || !secretAccessKey) return null;
  return { endpoint, accessKeyId, secretAccessKey, bucket: c.env.R2_BUCKET_NAME || "storage-files" };
}

function checkPrefix(c: C, path: string): Response | null {
  const pfx = c.get("prefix");
  if (pfx && !path.startsWith(pfx)) return err(c, "forbidden", "Path not allowed for this token");
  return null;
}

// ── POST /presign/upload ────────────────────────────────────────────
// Returns a presigned PUT URL for direct R2 upload.
// Client uploads to that URL, then calls POST /presign/complete.
async function uploadPresign(c: C) {
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

// ── POST /presign/complete ──────────────────────────────────────────
// Called after a presigned upload finishes. Verifies the object in R2
// and updates the D1 search index.
async function uploadComplete(c: C) {
  const body = await c.req.json<{ path?: string }>();
  const path = (body.path || "").replace(/^\/+/, "");
  if (!path) return err(c, "bad_request", "path is required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const key = `${actor}/${path}`;

  // Verify the object actually exists in R2
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

// ── GET /presign/read/* ─────────────────────────────────────────────
// Returns a presigned GET URL for direct R2 download.
async function readPresign(c: C) {
  const cfg = r2Config(c);
  if (!cfg) return err(c, "not_configured", "Presigned URLs not configured");

  const path = wildcardPath(c, "/presign/read/");
  if (!path) return err(c, "bad_request", "File path required");
  const pathErr = validatePath(path);
  if (pathErr) return err(c, "bad_request", pathErr);
  const pfxErr = checkPrefix(c, path);
  if (pfxErr) return pfxErr;

  const actor = c.get("actor");
  const key = `${actor}/${path}`;

  // Verify object exists
  const head = await c.env.BUCKET.head(key);
  if (!head) return err(c, "not_found", "File not found");

  const url = await presignUrl({
    method: "GET",
    key,
    bucket: cfg.bucket,
    endpoint: cfg.endpoint,
    accessKeyId: cfg.accessKeyId,
    secretAccessKey: cfg.secretAccessKey,
    expiresIn: READ_EXPIRES,
  });

  audit(c, "read", path);
  return c.json({
    url,
    size: head.size,
    type: head.httpMetadata?.contentType || "application/octet-stream",
    etag: head.etag,
    expires_in: READ_EXPIRES,
  });
}

// ── POST /presign/multipart/create ────────────────────────────────
// Initiates a multipart upload and returns presigned URLs for each part.
async function multipartCreate(c: C) {
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

  // Create multipart upload via R2 binding
  const mpu = await c.env.BUCKET.createMultipartUpload(key, {
    httpMetadata: { contentType },
  });

  // Generate presigned PUT URLs for each part
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
      expiresIn: 86400, // 24 hours for large uploads
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

// ── POST /presign/multipart/complete ──────────────────────────────
// Completes a multipart upload and updates D1 index.
async function multipartComplete(c: C) {
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

  // Complete the multipart upload via R2 binding
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

  // HEAD to get final size
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

// ── POST /presign/multipart/abort ──────────────────────────────────
// Aborts a multipart upload.
async function multipartAbort(c: C) {
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

export function register(app: App) {
  app.post("/presign/upload", auth, uploadPresign);
  app.post("/presign/complete", auth, uploadComplete);
  app.get("/presign/read/*", auth, readPresign);

  // Multipart upload routes
  app.post("/presign/multipart/create", auth, multipartCreate);
  app.post("/presign/multipart/complete", auth, multipartComplete);
  app.post("/presign/multipart/abort", auth, multipartAbort);
}
