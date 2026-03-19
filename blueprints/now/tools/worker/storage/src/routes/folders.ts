import type { Context } from "hono";
import type { Env, Variables, ObjectRow } from "../types";
import { objectId } from "../lib/id";
import { errorResponse } from "../lib/error";
import { wildcardPath, validatePath } from "../lib/path";
import { requireScope } from "../middleware/authorize";
import { ensureDefaultBucket } from "./buckets";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

// POST /folders  { "path": "docs/reports" }
export async function createFolder(c: AppContext) {
  const scopeErr = requireScope(c, "folders:write");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");

  let body: { path?: string };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  let folderPath = body.path?.trim();
  if (!folderPath) {
    return errorResponse(c, "invalid_request", "path is required");
  }

  if (!folderPath.endsWith("/")) folderPath += "/";
  if (folderPath.startsWith("/")) folderPath = folderPath.slice(1);

  const pathErr = validatePath(folderPath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const name = folderPath.replace(/\/$/, "").split("/").pop() || folderPath;

  const existing = await c.env.DB.prepare(
    "SELECT id FROM objects WHERE owner = ? AND path = ?",
  )
    .bind(actor, folderPath)
    .first();

  if (existing) {
    return c.json({ path: folderPath, name, created: false });
  }

  const bkId = await ensureDefaultBucket(c.env.DB, actor);

  const parts = folderPath.replace(/\/$/, "").split("/");
  parts.pop();
  let current = "";
  for (const part of parts) {
    current = current ? `${current}/${part}` : part;
    const parentPath = current + "/";
    const parentExists = await c.env.DB.prepare(
      "SELECT 1 FROM objects WHERE owner = ? AND path = ?",
    )
      .bind(actor, parentPath)
      .first();
    if (!parentExists) {
      const now = Date.now();
      await c.env.DB.prepare(
        "INSERT OR IGNORE INTO objects (id, owner, bucket_id, path, name, is_folder, content_type, size, r2_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 1, '', 0, '', ?, ?)",
      )
        .bind(objectId(), actor, bkId, parentPath, part, now, now)
        .run();
    }
  }

  const now = Date.now();
  const id = objectId();
  await c.env.DB.prepare(
    "INSERT INTO objects (id, owner, bucket_id, path, name, is_folder, content_type, size, r2_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 1, '', 0, '', ?, ?)",
  )
    .bind(id, actor, bkId, folderPath, name, now, now)
    .run();

  return c.json({ id, path: folderPath, name, created: true }, 201);
}

// GET /folders  or  GET /folders/*path
export async function listFolder(c: AppContext) {
  const scopeErr = requireScope(c, "folders:read");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  let prefix = wildcardPath(c, "/folders/");

  if (prefix && !prefix.endsWith("/")) prefix += "/";

  if (prefix) {
    const pathErr = validatePath(prefix);
    if (pathErr) return errorResponse(c, "invalid_request", pathErr);
  }

  const { results } = await c.env.DB.prepare(`
    SELECT id, path, name, is_folder, content_type, size, starred, created_at, updated_at
    FROM objects
    WHERE owner = ? AND path LIKE ? AND path != ? AND trashed_at IS NULL
    ORDER BY is_folder DESC, name ASC
  `)
    .bind(actor, prefix + "%", prefix)
    .all<ObjectRow>();

  const items = (results || []).filter((obj) => {
    const rest = obj.path.slice(prefix.length);
    if (obj.is_folder) {
      return rest.replace(/\/$/, "").indexOf("/") === -1;
    }
    return rest.indexOf("/") === -1;
  });

  const mapped = items.map((o) => ({
    id: o.id,
    name: o.name,
    path: o.path,
    is_folder: !!o.is_folder,
    content_type: o.content_type,
    size: o.size,
    starred: !!o.starred,
    created_at: o.created_at,
    updated_at: o.updated_at,
  }));

  return c.json({ path: prefix || "/", items: mapped });
}

// DELETE /folders/*path
export async function deleteFolder(c: AppContext) {
  const scopeErr = requireScope(c, "folders:write");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  let folderPath = wildcardPath(c, "/folders/");

  if (!folderPath) {
    return errorResponse(c, "invalid_request", "Folder path is required");
  }
  if (!folderPath.endsWith("/")) folderPath += "/";

  const pathErr = validatePath(folderPath);
  if (pathErr) return errorResponse(c, "invalid_request", pathErr);

  const folder = await c.env.DB.prepare(
    "SELECT id FROM objects WHERE owner = ? AND path = ? AND is_folder = 1",
  )
    .bind(actor, folderPath)
    .first<{ id: string }>();

  if (!folder) {
    return errorResponse(c, "not_found", "Folder not found");
  }

  const child = await c.env.DB.prepare(
    "SELECT 1 FROM objects WHERE owner = ? AND path LIKE ? AND path != ? LIMIT 1",
  )
    .bind(actor, folderPath + "%", folderPath)
    .first();

  if (child) {
    return errorResponse(c, "conflict", "Folder is not empty");
  }

  await c.env.DB.prepare("DELETE FROM objects WHERE id = ?").bind(folder.id).run();
  return c.json({ deleted: true });
}
