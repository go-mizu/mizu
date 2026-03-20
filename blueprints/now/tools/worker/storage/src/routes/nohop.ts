/**
 * No-hop upload/download: client talks directly to R2 via presigned URLs.
 *
 * POST /object/upload/token  — get a presigned PUT URL + commit URL
 * POST /object/commit        — register object metadata after direct upload
 */
import type { Context } from "hono";
import type { Env, Variables, BucketRow } from "../types";
import { objectId } from "../lib/id";
import { mimeFromName } from "../lib/mime";
import { errorResponse } from "../lib/error";
import { validatePath } from "../lib/path";
import { requireScope } from "../middleware/authorize";
import { audit } from "../lib/audit";
import { resolveBucket } from "./buckets";
import { presignPut } from "../lib/r2";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const DEFAULT_EXPIRES = 3600;
const MAX_EXPIRES = 86400;

// POST /object/upload/token
export async function objectUploadToken(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  let body: {
    bucket?: string;
    path?: string;
    content_type?: string;
    expires_in?: number;
  };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  const bucketName = body.bucket;
  const filePath = body.path?.replace(/^\/+/, "");

  if (!bucketName) return errorResponse(c, "invalid_request", "bucket is required");
  if (!filePath || filePath.endsWith("/")) return errorResponse(c, "invalid_request", "path is required (not a folder)");

  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const actor = c.get("actor");
  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  const contentType = body.content_type || mimeFromName(filePath.split("/").pop() || filePath);
  const expiresIn = Math.min(Math.max(body.expires_in || DEFAULT_EXPIRES, 1), MAX_EXPIRES);
  const r2Key = `${actor}/${bucket.name}/${filePath}`;

  const uploadUrl = await presignPut(c, r2Key, expiresIn);
  if (!uploadUrl) {
    return errorResponse(c, "not_configured", "Presigned uploads require R2 credentials (R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY)");
  }

  // Build absolute commit URL
  const origin = new URL(c.req.url).origin;

  return c.json({
    upload_url: uploadUrl,
    commit_url: `${origin}/object/commit`,
    r2_key: r2Key,
    bucket: bucketName,
    path: filePath,
    content_type: contentType,
    expires_in: expiresIn,
    method: "PUT",
    headers: { "Content-Type": contentType },
  });
}

// POST /object/commit
export async function objectCommit(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  let body: {
    bucket?: string;
    path?: string;
  };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  const bucketName = body.bucket;
  const filePath = body.path?.replace(/^\/+/, "");

  if (!bucketName) return errorResponse(c, "invalid_request", "bucket is required");
  if (!filePath || filePath.endsWith("/")) return errorResponse(c, "invalid_request", "path is required (not a folder)");

  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const actor = c.get("actor");
  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  const r2Key = `${actor}/${bucket.name}/${filePath}`;

  // Verify the object actually exists in R2
  const r2Head = await c.env.BUCKET.head(r2Key);
  if (!r2Head) {
    return errorResponse(c, "not_found", "Object not found in storage — upload may have failed or not yet completed");
  }

  const name = filePath.split("/").pop() || filePath;
  const contentType = r2Head.httpMetadata?.contentType || mimeFromName(name);
  const size = r2Head.size;

  const now = Date.now();
  const existing = await c.env.DB
    .prepare("SELECT id FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, filePath)
    .first<{ id: string }>();

  let id: string;
  if (existing) {
    id = existing.id;
    await c.env.DB
      .prepare("UPDATE objects SET content_type = ?, size = ?, r2_key = ?, updated_at = ? WHERE id = ?")
      .bind(contentType, size, r2Key, now, id)
      .run();
  } else {
    id = objectId();
    await c.env.DB
      .prepare(
        "INSERT INTO objects (id, owner, bucket_id, path, name, content_type, size, r2_key, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '{}', ?, ?)",
      )
      .bind(id, actor, bucket.id, filePath, name, contentType, size, r2Key, now, now)
      .run();
  }

  audit(c, "object.commit", `${bucketName}/${filePath}`, { size });

  return c.json({ id, bucket: bucketName, path: filePath, name, content_type: contentType, size, created_at: now }, existing ? 200 : 201);
}
