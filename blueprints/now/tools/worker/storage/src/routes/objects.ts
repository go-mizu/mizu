import type { Context } from "hono";
import type { Env, Variables, ObjectRow, BucketRow } from "../types";
import { objectId } from "../lib/id";
import { mimeFromName, isInlineType } from "../lib/mime";
import { errorResponse } from "../lib/error";
import { validatePath } from "../lib/path";
import { requireScope, sanitizeFilename } from "../middleware/authorize";
import { audit } from "../lib/audit";
import { resolveBucket } from "./buckets";
import { presignGet } from "../lib/r2";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const MAX_FILE_SIZE = 100 * 1024 * 1024; // 100MB

/** Extract bucket name and object path from URL */
function parseBucketPath(c: AppContext, prefix: string): { bucket: string; path: string } {
  const url = new URL(c.req.url);
  const raw = decodeURIComponent(url.pathname);
  const rest = raw.slice(raw.indexOf(prefix) + prefix.length);
  const slashIdx = rest.indexOf("/");
  if (slashIdx === -1) return { bucket: rest, path: "" };
  return { bucket: rest.slice(0, slashIdx), path: rest.slice(slashIdx + 1) };
}

/** Validate bucket access and MIME/size limits */
async function validateUpload(
  c: AppContext,
  bucket: BucketRow,
  contentType: string,
  size: number,
): Promise<Response | null> {
  if (bucket.file_size_limit && size > bucket.file_size_limit) {
    return errorResponse(c, "too_large", `File exceeds bucket limit of ${bucket.file_size_limit} bytes`);
  }

  if (bucket.allowed_mime_types) {
    const allowed: string[] = JSON.parse(bucket.allowed_mime_types);
    if (allowed.length > 0 && !allowed.includes(contentType)) {
      return errorResponse(c, "invalid_request", `MIME type ${contentType} not allowed in this bucket`);
    }
  }

  return null;
}

// POST /object/:bucket/*path — upload file (error if exists)
export async function createObject(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const { bucket: bucketName, path: filePath } = parseBucketPath(c, "/object/");

  if (!filePath) return errorResponse(c, "invalid_request", "Object path is required");
  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  const existing = await c.env.DB
    .prepare("SELECT 1 FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, filePath)
    .first();

  if (existing) return errorResponse(c, "conflict", "Object already exists — use PUT to upsert");

  return doUpload(c, actor, bucket, filePath, 201);
}

// PUT /object/:bucket/*path — upsert file
export async function upsertObject(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const { bucket: bucketName, path: filePath } = parseBucketPath(c, "/object/");

  if (!filePath) return errorResponse(c, "invalid_request", "Object path is required");
  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  return doUpload(c, actor, bucket, filePath, null);
}

async function doUpload(
  c: AppContext,
  actor: string,
  bucket: BucketRow,
  filePath: string,
  forceStatus: number | null,
) {
  const cl = c.req.header("Content-Length");
  if (cl && parseInt(cl, 10) > MAX_FILE_SIZE) {
    return errorResponse(c, "too_large", "File exceeds 100MB limit — use signed upload URLs for larger files");
  }

  const body = await c.req.arrayBuffer();
  if (body.byteLength > MAX_FILE_SIZE) {
    return errorResponse(c, "too_large", "File exceeds 100MB limit");
  }

  const name = filePath.split("/").pop() || filePath;
  const contentType = c.req.header("Content-Type") || mimeFromName(name);

  const limitErr = await validateUpload(c, bucket, contentType, body.byteLength);
  if (limitErr) return limitErr;

  const r2Key = `${actor}/${bucket.name}/${filePath}`;

  await c.env.BUCKET.put(r2Key, body, {
    httpMetadata: { contentType },
  });

  const now = Date.now();
  const existingObj = await c.env.DB
    .prepare("SELECT id FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, filePath)
    .first<{ id: string }>();

  let id: string;
  let status: number;

  if (existingObj) {
    id = existingObj.id;
    status = 200;
    await c.env.DB
      .prepare("UPDATE objects SET content_type = ?, size = ?, r2_key = ?, updated_at = ? WHERE id = ?")
      .bind(contentType, body.byteLength, r2Key, now, id)
      .run();
  } else {
    id = objectId();
    status = 201;
    await c.env.DB
      .prepare(
        "INSERT INTO objects (id, owner, bucket_id, path, name, content_type, size, r2_key, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '{}', ?, ?)",
      )
      .bind(id, actor, bucket.id, filePath, name, contentType, body.byteLength, r2Key, now, now)
      .run();
  }

  audit(c, "object.upload", `${bucket.name}/${filePath}`);

  return c.json(
    { id, bucket: bucket.name, path: filePath, name, content_type: contentType, size: body.byteLength, created_at: now },
    (forceStatus || status) as any,
  );
}

// GET /object/:bucket/*path — download (auth required)
export async function downloadObject(c: AppContext) {
  const scopeErr = requireScope(c, "object:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const { bucket: bucketName, path: filePath } = parseBucketPath(c, "/object/");

  if (!filePath) return errorResponse(c, "invalid_request", "Object path is required");
  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  return serveObject(c, bucket, filePath);
}

// GET /object/public/:bucket/*path — download from public bucket (no auth)
export async function downloadPublicObject(c: AppContext) {
  const { bucket: bucketName, path: filePath } = parseBucketPath(c, "/object/public/");

  if (!filePath) return errorResponse(c, "invalid_request", "Object path is required");
  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  // Find bucket by name across all owners
  const bucket = await c.env.DB
    .prepare("SELECT * FROM buckets WHERE name = ? AND public = 1 LIMIT 1")
    .bind(bucketName)
    .first<BucketRow>();

  if (!bucket) return errorResponse(c, "not_found", "Public bucket not found");

  return serveObject(c, bucket, filePath);
}

async function serveObject(c: AppContext, bucket: BucketRow, filePath: string) {
  const obj = await c.env.DB
    .prepare("SELECT * FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, filePath)
    .first<ObjectRow>();

  if (!obj) return errorResponse(c, "not_found", "Object not found");

  // Update accessed_at
  c.executionCtx.waitUntil(
    c.env.DB.prepare("UPDATE objects SET accessed_at = ? WHERE id = ?").bind(Date.now(), obj.id).run(),
  );

  // ?redirect=1 — redirect to presigned R2 URL (0-hop download)
  if (c.req.query("redirect") !== undefined) {
    const presignedUrl = await presignGet(c, obj.r2_key, 300);
    if (presignedUrl) return c.redirect(presignedUrl, 302);
    // Fall through to streaming if presigning not configured
  }

  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (!r2Obj) return errorResponse(c, "not_found", "Object data not found in storage");

  const headers = new Headers();
  headers.set("Content-Type", obj.content_type || "application/octet-stream");
  headers.set("Content-Length", obj.size.toString());
  headers.set("ETag", r2Obj.etag);
  headers.set("Last-Modified", new Date(obj.updated_at).toUTCString());
  headers.set("Cache-Control", bucket.public ? "public, max-age=3600" : "private, no-cache");

  const safeName = sanitizeFilename(obj.name);
  if (isInlineType(obj.content_type)) {
    headers.set("Content-Disposition", `inline; filename="${safeName}"`);
  } else {
    headers.set("Content-Disposition", `attachment; filename="${safeName}"`);
  }

  // ?download forces attachment
  if (c.req.query("download") !== undefined) {
    const downloadName = c.req.query("download") || safeName;
    headers.set("Content-Disposition", `attachment; filename="${sanitizeFilename(downloadName)}"`);
  }

  return new Response(r2Obj.body, { headers });
}

// HEAD /object/:bucket/*path — metadata headers
export async function headObject(c: AppContext) {
  const scopeErr = requireScope(c, "object:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const { bucket: bucketName, path: filePath } = parseBucketPath(c, "/object/");

  const pathErr = validatePath(filePath);
  if (pathErr) return c.body(null, 400);

  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return c.body(null, 404);

  const obj = await c.env.DB
    .prepare("SELECT * FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, filePath)
    .first<ObjectRow>();

  if (!obj) return c.body(null, 404);

  return c.body(null, 200, {
    "Content-Type": obj.content_type || "application/octet-stream",
    "Content-Length": obj.size.toString(),
    "Last-Modified": new Date(obj.updated_at).toUTCString(),
  });
}

// GET /object/info/:bucket/*path — metadata as JSON
export async function objectInfo(c: AppContext) {
  const scopeErr = requireScope(c, "object:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const { bucket: bucketName, path: filePath } = parseBucketPath(c, "/object/info/");

  if (!filePath) return errorResponse(c, "invalid_request", "Object path is required");
  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  const obj = await c.env.DB
    .prepare("SELECT * FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, filePath)
    .first<ObjectRow>();

  if (!obj) return errorResponse(c, "not_found", "Object not found");

  return c.json({
    id: obj.id,
    bucket: bucket.name,
    path: obj.path,
    name: obj.name,
    content_type: obj.content_type,
    size: obj.size,
    metadata: JSON.parse(obj.metadata || "{}"),
    created_at: obj.created_at,
    updated_at: obj.updated_at,
    accessed_at: obj.accessed_at,
  });
}

// DELETE /object/:bucket — batch delete
export async function deleteObjects(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const bucketName = c.req.param("bucket")!;

  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  const body = await c.req.json<{ paths: string[] }>();
  if (!body.paths || !Array.isArray(body.paths) || body.paths.length === 0) {
    return errorResponse(c, "invalid_request", "paths array is required");
  }

  if (body.paths.length > 100) {
    return errorResponse(c, "invalid_request", "Maximum 100 paths per batch delete");
  }

  const deleted: string[] = [];

  for (const path of body.paths) {
    const obj = await c.env.DB
      .prepare("SELECT id, r2_key FROM objects WHERE bucket_id = ? AND path = ?")
      .bind(bucket.id, path)
      .first<{ id: string; r2_key: string }>();

    if (obj) {
      if (obj.r2_key) await c.env.BUCKET.delete(obj.r2_key);
      await c.env.DB.prepare("DELETE FROM objects WHERE id = ?").bind(obj.id).run();
      deleted.push(path);
    }
  }

  audit(c, "object.delete", bucket.name, { paths: deleted });

  return c.json({ deleted });
}

// POST /object/list/:bucket — list objects
export async function listObjects(c: AppContext) {
  const scopeErr = requireScope(c, "object:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const bucketName = c.req.param("bucket")!;

  const bucket = await resolveBucket(c.env.DB, actor, bucketName);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  let body: {
    prefix?: string;
    limit?: number;
    offset?: number;
    sort_by?: { column: string; order: string };
    search?: string;
  } = {};
  try {
    body = await c.req.json();
  } catch {
    // empty body is fine — all fields are optional
  }

  const limit = Math.min(body.limit || 100, 1000);
  const offset = body.offset || 0;
  const sortCol = body.sort_by?.column === "updated_at" ? "updated_at"
    : body.sort_by?.column === "size" ? "size"
    : body.sort_by?.column === "created_at" ? "created_at"
    : "name";
  const sortOrder = body.sort_by?.order === "desc" ? "DESC" : "ASC";

  let sql = "SELECT id, path, name, content_type, size, metadata, created_at, updated_at FROM objects WHERE bucket_id = ?";
  const binds: any[] = [bucket.id];

  if (body.prefix) {
    sql += " AND path LIKE ?";
    binds.push(body.prefix.replace(/%/g, "\\%") + "%");
  }

  if (body.search) {
    sql += " AND name LIKE ?";
    binds.push("%" + body.search.replace(/%/g, "\\%") + "%");
  }

  sql += ` ORDER BY ${sortCol} ${sortOrder} LIMIT ? OFFSET ?`;
  binds.push(limit, offset);

  const { results } = await c.env.DB.prepare(sql).bind(...binds).all();

  return c.json(
    (results || []).map((r: any) => ({
      id: r.id,
      name: r.name,
      path: r.path,
      content_type: r.content_type,
      size: r.size,
      metadata: r.metadata ? JSON.parse(r.metadata) : {},
      created_at: r.created_at,
      updated_at: r.updated_at,
    })),
  );
}

// POST /object/move — move/rename
export async function moveObject(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const body = await c.req.json<{
    bucket: string;
    from: string;
    to: string;
  }>();

  if (!body.bucket || !body.from || !body.to) {
    return errorResponse(c, "invalid_request", "bucket, from, and to are required");
  }

  const pathErr = validatePath(body.to);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const bucket = await resolveBucket(c.env.DB, actor, body.bucket);
  if (!bucket) return errorResponse(c, "not_found", "Bucket not found");

  const obj = await c.env.DB
    .prepare("SELECT * FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, body.from)
    .first<ObjectRow>();

  if (!obj) return errorResponse(c, "not_found", "Source object not found");

  // Check target doesn't exist
  const targetExists = await c.env.DB
    .prepare("SELECT 1 FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(bucket.id, body.to)
    .first();

  if (targetExists) return errorResponse(c, "conflict", "Target path already exists");

  // Move R2 object
  const newR2Key = `${actor}/${bucket.name}/${body.to}`;
  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (r2Obj) {
    await c.env.BUCKET.put(newR2Key, r2Obj.body, {
      httpMetadata: { contentType: obj.content_type },
    });
    await c.env.BUCKET.delete(obj.r2_key);
  }

  const newName = body.to.split("/").pop() || body.to;
  const now = Date.now();
  await c.env.DB
    .prepare("UPDATE objects SET path = ?, name = ?, r2_key = ?, updated_at = ? WHERE id = ?")
    .bind(body.to, newName, newR2Key, now, obj.id)
    .run();

  audit(c, "object.move", `${bucket.name}/${body.from}`, { to: body.to });

  return c.json({ path: body.to });
}

// POST /object/copy — copy object
export async function copyObject(c: AppContext) {
  const scopeErr = requireScope(c, "object:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const body = await c.req.json<{
    from_bucket: string;
    from_path: string;
    to_bucket: string;
    to_path: string;
  }>();

  if (!body.from_bucket || !body.from_path || !body.to_bucket || !body.to_path) {
    return errorResponse(c, "invalid_request", "from_bucket, from_path, to_bucket, and to_path are required");
  }

  const pathErr = validatePath(body.to_path);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const srcBucket = await resolveBucket(c.env.DB, actor, body.from_bucket);
  if (!srcBucket) return errorResponse(c, "not_found", "Source bucket not found");

  const dstBucket = await resolveBucket(c.env.DB, actor, body.to_bucket);
  if (!dstBucket) return errorResponse(c, "not_found", "Destination bucket not found");

  const srcObj = await c.env.DB
    .prepare("SELECT * FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(srcBucket.id, body.from_path)
    .first<ObjectRow>();

  if (!srcObj) return errorResponse(c, "not_found", "Source object not found");

  const targetExists = await c.env.DB
    .prepare("SELECT 1 FROM objects WHERE bucket_id = ? AND path = ?")
    .bind(dstBucket.id, body.to_path)
    .first();

  if (targetExists) return errorResponse(c, "conflict", "Target path already exists");

  // Copy R2 object
  const newR2Key = `${actor}/${dstBucket.name}/${body.to_path}`;
  const r2Obj = await c.env.BUCKET.get(srcObj.r2_key);
  if (!r2Obj) return errorResponse(c, "not_found", "Source object data not found");

  await c.env.BUCKET.put(newR2Key, r2Obj.body, {
    httpMetadata: { contentType: srcObj.content_type },
  });

  const id = objectId();
  const now = Date.now();
  const newName = body.to_path.split("/").pop() || body.to_path;

  await c.env.DB
    .prepare(
      "INSERT INTO objects (id, owner, bucket_id, path, name, content_type, size, r2_key, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
    )
    .bind(id, actor, dstBucket.id, body.to_path, newName, srcObj.content_type, srcObj.size, newR2Key, srcObj.metadata || "{}", now, now)
    .run();

  audit(c, "object.copy", `${body.from_bucket}/${body.from_path}`, { to: `${body.to_bucket}/${body.to_path}` });

  return c.json({ id, path: body.to_path }, 201);
}
