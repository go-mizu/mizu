import type { Context } from "hono";
import type { Env, Variables, ObjectRow } from "../types";
import { objectId } from "../lib/id";
import { mimeFromName, isInlineType } from "../lib/mime";
import { errorResponse } from "../lib/error";
import { wildcardPath, validatePath } from "../lib/path";
import { requireScope, checkPathPrefix, sanitizeFilename } from "../middleware/authorize";
import { audit } from "../lib/audit";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const MAX_FILE_SIZE = 100 * 1024 * 1024; // 100MB

/**
 * Ensure all parent folders exist in D1 for a given path.
 * e.g. for "docs/reports/q1.pdf" → creates "docs/" and "docs/reports/"
 */
export async function ensureParentFolders(db: D1Database, owner: string, filePath: string) {
  const parts = filePath.split("/");
  parts.pop();

  let current = "";
  for (const part of parts) {
    current = current ? `${current}/${part}` : part;
    const folderPath = current + "/";

    const existing = await db
      .prepare("SELECT 1 FROM objects WHERE owner = ? AND path = ?")
      .bind(owner, folderPath)
      .first();

    if (!existing) {
      const now = Date.now();
      await db
        .prepare(
          "INSERT OR IGNORE INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, created_at, updated_at) VALUES (?, ?, ?, ?, 1, '', 0, '', ?, ?)",
        )
        .bind(objectId(), owner, folderPath, part, now, now)
        .run();
    }
  }
}

// PUT /files/*path
export async function uploadFile(c: AppContext) {
  const scopeErr = requireScope(c, "files:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const filePath = wildcardPath(c, "/files/");

  if (!filePath || filePath.endsWith("/")) {
    return errorResponse(c, "invalid_request", "File path is required (not a folder)");
  }

  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const prefixErr = checkPathPrefix(c, filePath);
  if (prefixErr) return prefixErr;

  const cl = c.req.header("Content-Length");
  if (cl && parseInt(cl, 10) > MAX_FILE_SIZE) {
    return errorResponse(c, "too_large", "File exceeds 100MB limit");
  }

  const body = await c.req.arrayBuffer();
  if (body.byteLength > MAX_FILE_SIZE) {
    return errorResponse(c, "too_large", "File exceeds 100MB limit");
  }

  const name = filePath.split("/").pop() || filePath;
  const contentType = c.req.header("Content-Type") || mimeFromName(name);
  const r2Key = `${actor}/${filePath}`;

  await c.env.BUCKET.put(r2Key, body, {
    httpMetadata: { contentType },
  });

  await ensureParentFolders(c.env.DB, actor, filePath);

  const now = Date.now();
  const existing = await c.env.DB.prepare(
    "SELECT id FROM objects WHERE owner = ? AND path = ?",
  )
    .bind(actor, filePath)
    .first<{ id: string }>();

  let id: string;
  if (existing) {
    id = existing.id;
    await c.env.DB.prepare(
      "UPDATE objects SET content_type = ?, size = ?, r2_key = ?, updated_at = ? WHERE id = ?",
    )
      .bind(contentType, body.byteLength, r2Key, now, id)
      .run();
  } else {
    id = objectId();
    await c.env.DB.prepare(
      "INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, created_at, updated_at) VALUES (?, ?, ?, ?, 0, ?, ?, ?, ?, ?)",
    )
      .bind(id, actor, filePath, name, contentType, body.byteLength, r2Key, now, now)
      .run();
  }

  audit(c, "file.upload", filePath);

  return c.json(
    { id, path: filePath, name, content_type: contentType, size: body.byteLength, created_at: now },
    existing ? 200 : 201,
  );
}

// GET /files/*path
export async function downloadFile(c: AppContext) {
  const scopeErr = requireScope(c, "files:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const filePath = wildcardPath(c, "/files/");

  if (!filePath) {
    return errorResponse(c, "invalid_request", "File path is required");
  }

  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const prefixErr = checkPathPrefix(c, filePath);
  if (prefixErr) return prefixErr;

  const obj = await c.env.DB.prepare(
    "SELECT * FROM objects WHERE owner = ? AND path = ? AND is_folder = 0 AND trashed_at IS NULL",
  )
    .bind(actor, filePath)
    .first<ObjectRow>();

  if (!obj) {
    return errorResponse(c, "not_found", "File not found");
  }

  await c.env.DB.prepare("UPDATE objects SET accessed_at = ? WHERE id = ?")
    .bind(Date.now(), obj.id).run();

  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (!r2Obj) {
    return errorResponse(c, "not_found", "File data not found");
  }

  audit(c, "file.download", filePath);

  const headers = new Headers();
  headers.set("Content-Type", obj.content_type || "application/octet-stream");
  headers.set("Content-Length", obj.size.toString());
  headers.set("ETag", r2Obj.etag);
  headers.set("Last-Modified", new Date(obj.updated_at).toUTCString());

  const safeName = sanitizeFilename(obj.name);
  if (isInlineType(obj.content_type)) {
    headers.set("Content-Disposition", `inline; filename="${safeName}"`);
  } else {
    headers.set("Content-Disposition", `attachment; filename="${safeName}"`);
  }

  return new Response(r2Obj.body, { headers });
}

// DELETE /files/*path
export async function deleteFile(c: AppContext) {
  const scopeErr = requireScope(c, "files:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const filePath = wildcardPath(c, "/files/");

  if (!filePath) {
    return errorResponse(c, "invalid_request", "File path is required");
  }

  const pathErr = validatePath(filePath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const prefixErr = checkPathPrefix(c, filePath);
  if (prefixErr) return prefixErr;

  const obj = await c.env.DB.prepare(
    "SELECT id, r2_key FROM objects WHERE owner = ? AND path = ? AND is_folder = 0 AND trashed_at IS NULL",
  )
    .bind(actor, filePath)
    .first<{ id: string; r2_key: string }>();

  if (!obj) {
    return errorResponse(c, "not_found", "File not found");
  }

  await c.env.BUCKET.delete(obj.r2_key);
  await c.env.DB.prepare("DELETE FROM shares WHERE object_id = ?").bind(obj.id).run();
  await c.env.DB.prepare("DELETE FROM objects WHERE id = ?").bind(obj.id).run();

  audit(c, "file.delete", filePath);

  return c.json({ deleted: true });
}

// HEAD /files/*path
export async function headFile(c: AppContext) {
  const scopeErr = requireScope(c, "files:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const filePath = wildcardPath(c, "/files/");

  const pathErr = validatePath(filePath);
  if (pathErr) return c.body(null, 400);

  const obj = await c.env.DB.prepare(
    "SELECT * FROM objects WHERE owner = ? AND path = ? AND is_folder = 0 AND trashed_at IS NULL",
  )
    .bind(actor, filePath)
    .first<ObjectRow>();

  if (!obj) {
    return c.body(null, 404);
  }

  return c.body(null, 200, {
    "Content-Type": obj.content_type || "application/octet-stream",
    "Content-Length": obj.size.toString(),
    "Last-Modified": new Date(obj.updated_at).toUTCString(),
  });
}
