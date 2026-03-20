import type { Context } from "hono";
import type { Env, Variables, SignedUrlRow, ObjectRow, BucketRow } from "../types";
import { signedUrlId, signedUrlToken, objectId } from "../lib/id";
import { mimeFromName, isInlineType } from "../lib/mime";
import { errorResponse } from "../lib/error";
import { validatePath } from "../lib/path";
import { requireScope, sanitizeFilename } from "../middleware/authorize";
import { audit } from "../lib/audit";
import { resolveBucket, resolveBucketById } from "./buckets";
import { presignGet } from "../lib/r2";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const MAX_FILE_SIZE = 100 * 1024 * 1024;
const DEFAULT_EXPIRES = 3600; // 1 hour
const MAX_EXPIRES = 7 * 24 * 3600; // 7 days

// POST /object/sign/:bucket — create signed download URL(s)
export async function createSignedUrl(c: AppContext) {
  const scopeErr = requireScope(c, "object:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const bucketName = c.req.param("bucket")!;

  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  const body = await c.req.json<{
    path?: string;
    paths?: string[];
    expires_in?: number;
    presign?: boolean;
  }>();

  const expiresIn = Math.min(body.expires_in || DEFAULT_EXPIRES, MAX_EXPIRES);
  const now = Date.now();
  const expiresAt = now + expiresIn * 1000;

  // Single path
  if (body.path) {
    const pathErr = validatePath(body.path);
    if (pathErr) return errorResponse(c, "invalid_request", pathErr);

    const obj = await c.env.DB
      .prepare("SELECT r2_key FROM objects WHERE bucket_id = ? AND path = ?")
      .bind(bucket.id, body.path)
      .first<{ r2_key: string }>();

    if (!obj) return errorResponse(c, "not_found", `Object not found: ${body.path}`);

    // presign: true — return direct R2 presigned URL (0-hop download)
    if (body.presign) {
      const presignedUrl = await presignGet(c, obj.r2_key, expiresIn);
      if (!presignedUrl) {
        return errorResponse(c, "not_configured", "Presigned URLs require R2 credentials (R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY)");
      }
      audit(c, "signed_url.presign", `${bucketName}/${body.path}`);
      return c.json({ signed_url: presignedUrl, path: body.path, expires_at: expiresAt });
    }

    const id = signedUrlId();
    const token = signedUrlToken();

    await c.env.DB
      .prepare(
        "INSERT INTO signed_urls (id, owner, bucket_id, path, token, type, expires_at, created_at) VALUES (?, ?, ?, ?, ?, 'download', ?, ?)",
      )
      .bind(id, actor, bucket.id, body.path, token, expiresAt, now)
      .run();

    audit(c, "signed_url.create", `${bucketName}/${body.path}`);

    return c.json({
      signed_url: `/sign/${token}`,
      token,
      path: body.path,
      expires_at: expiresAt,
    });
  }

  // Batch paths
  if (body.paths && Array.isArray(body.paths)) {
    if (body.paths.length > 100) {
      return errorResponse(c, "invalid_request", "Maximum 100 paths per batch");
    }

    const results = [];
    for (const path of body.paths) {
      const pathErr = validatePath(path);
      if (pathErr) {
        results.push({ path, error: pathErr });
        continue;
      }

      const obj = await c.env.DB
        .prepare("SELECT 1 FROM objects WHERE bucket_id = ? AND path = ?")
        .bind(bucket.id, path)
        .first();

      if (!obj) {
        results.push({ path, error: "Object not found" });
        continue;
      }

      const id = signedUrlId();
      const token = signedUrlToken();

      await c.env.DB
        .prepare(
          "INSERT INTO signed_urls (id, owner, bucket_id, path, token, type, expires_at, created_at) VALUES (?, ?, ?, ?, ?, 'download', ?, ?)",
        )
        .bind(id, actor, bucket.id, path, token, expiresAt, now)
        .run();

      results.push({ signed_url: `/sign/${token}`, path, token });
    }

    audit(c, "signed_url.batch", bucketName, { count: body.paths.length });

    return c.json(results);
  }

  return errorResponse(c, "invalid_request", "Provide path or paths");
}

// POST /object/upload/sign/:bucket/*path — create signed upload URL
export async function createSignedUploadUrl(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const url = new URL(c.req.url);
  const raw = decodeURIComponent(url.pathname);
  const prefix = "/object/upload/sign/";
  const rest = raw.slice(raw.indexOf(prefix) + prefix.length);
  const slashIdx = rest.indexOf("/");
  if (slashIdx === -1) return errorResponse(c, "invalid_request", "Object path is required");
  const bucketName = rest.slice(0, slashIdx);
  const filePath = rest.slice(slashIdx + 1);

  if (!filePath) return errorResponse(c, "invalid_request", "Object path is required");
  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  const now = Date.now();
  const expiresAt = now + DEFAULT_EXPIRES * 1000;
  const id = signedUrlId();
  const token = signedUrlToken();

  await c.env.DB
    .prepare(
      "INSERT INTO signed_urls (id, owner, bucket_id, path, token, type, expires_at, created_at) VALUES (?, ?, ?, ?, ?, 'upload', ?, ?)",
    )
    .bind(id, actor, bucket.id, filePath, token, expiresAt, now)
    .run();

  audit(c, "signed_url.upload_create", `${bucketName}/${filePath}`);

  return c.json({
    signed_url: `/upload/sign/${token}`,
    token,
    path: filePath,
    expires_at: expiresAt,
  });
}

// GET /sign/:token — download via signed URL (no auth)
export async function accessSignedUrl(c: AppContext) {
  const token = c.req.param("token")!;

  const signed = await c.env.DB
    .prepare("SELECT * FROM signed_urls WHERE token = ? AND type = 'download'")
    .bind(token)
    .first<SignedUrlRow>();

  if (!signed) return errorResponse(c, "not_found", "Signed URL not found");

  if (Date.now() > signed.expires_at) {
    return errorResponse(c, "forbidden", "Signed URL has expired");
  }

  const bucket = await resolveBucketById(c.env.DB, signed.bucket_id);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  const obj = await c.env.DB
    .prepare("SELECT * FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, signed.path)
    .first<ObjectRow>();

  if (!obj) return errorResponse(c, "not_found", "Object not found");

  // ?redirect=1 — redirect to presigned R2 URL (0-hop download)
  if (c.req.query("redirect") !== undefined) {
    const presignedUrl = await presignGet(c, obj.r2_key, 300);
    if (presignedUrl) return c.redirect(presignedUrl, 302);
    // Fall through to streaming if presigning not configured
  }

  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (!r2Obj) return errorResponse(c, "not_found", "Object data not found");

  const headers = new Headers();
  headers.set("Content-Type", obj.content_type || "application/octet-stream");
  headers.set("Content-Length", obj.size.toString());
  headers.set("ETag", r2Obj.etag);
  headers.set("Cache-Control", "private, max-age=3600");

  const safeName = sanitizeFilename(obj.name);
  if (isInlineType(obj.content_type) && c.req.query("download") === undefined) {
    headers.set("Content-Disposition", `inline; filename="${safeName}"`);
  } else {
    const downloadName = c.req.query("download") || safeName;
    headers.set("Content-Disposition", `attachment; filename="${sanitizeFilename(downloadName)}"`);
  }

  return new Response(r2Obj.body, { headers });
}

// PUT /upload/sign/:token — upload via signed URL (no auth)
export async function uploadViaSignedUrl(c: AppContext) {
  const token = c.req.param("token")!;

  const signed = await c.env.DB
    .prepare("SELECT * FROM signed_urls WHERE token = ? AND type = 'upload'")
    .bind(token)
    .first<SignedUrlRow>();

  if (!signed) return errorResponse(c, "not_found", "Signed upload URL not found");

  if (Date.now() > signed.expires_at) {
    return errorResponse(c, "forbidden", "Signed upload URL has expired");
  }

  const bucket = await resolveBucketById(c.env.DB, signed.bucket_id);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  const cl = c.req.header("Content-Length");
  if (cl && parseInt(cl, 10) > MAX_FILE_SIZE) {
    return errorResponse(c, "too_large", "File exceeds 100MB limit");
  }

  const body = await c.req.arrayBuffer();
  if (body.byteLength > MAX_FILE_SIZE) {
    return errorResponse(c, "too_large", "File exceeds 100MB limit");
  }

  const name = signed.path.split("/").pop() || signed.path;
  const contentType = c.req.header("Content-Type") || mimeFromName(name);

  // Validate bucket limits
  if (bucket.file_size_limit && body.byteLength > bucket.file_size_limit) {
    return errorResponse(c, "too_large", `File exceeds bucket limit of ${bucket.file_size_limit} bytes`);
  }

  if (bucket.allowed_mime_types) {
    const allowed: string[] = JSON.parse(bucket.allowed_mime_types);
    if (allowed.length > 0 && !allowed.includes(contentType)) {
      return errorResponse(c, "invalid_request", `MIME type ${contentType} not allowed`);
    }
  }

  const r2Key = `${signed.owner}/${bucket.name}/${signed.path}`;

  await c.env.BUCKET.put(r2Key, body, {
    httpMetadata: { contentType },
  });

  const now = Date.now();
  const existing = await c.env.DB
    .prepare("SELECT id FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, signed.path)
    .first<{ id: string }>();

  let id: string;
  if (existing) {
    id = existing.id;
    await c.env.DB
      .prepare("UPDATE objects SET content_type = ?, size = ?, r2_key = ?, updated_at = ? WHERE id = ?")
      .bind(contentType, body.byteLength, r2Key, now, id)
      .run();
  } else {
    id = objectId();
    await c.env.DB
      .prepare(
        "INSERT INTO objects (id, owner, bucket_id, path, name, content_type, size, r2_key, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '{}', ?, ?)",
      )
      .bind(id, signed.owner, bucket.id, signed.path, name, contentType, body.byteLength, r2Key, now, now)
      .run();
  }

  // Consume the signed URL (one-time use)
  await c.env.DB.prepare("DELETE FROM signed_urls WHERE id = ?").bind(signed.id).run();

  return c.json(
    { id, bucket: bucket.name, path: signed.path, size: body.byteLength, content_type: contentType, created_at: now },
    201,
  );
}
