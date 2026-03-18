import type { Context } from "hono";
import type { Env, Variables, ObjectRow } from "./types";
import { objectId } from "./id";
import { errorResponse } from "./error";
import { wildcardPath } from "./path";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

// POST /folders  { "path": "docs/reports" }
export async function createFolder(c: AppContext) {
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

  // Normalize: ensure trailing slash
  if (!folderPath.endsWith("/")) folderPath += "/";
  // Remove leading slash
  if (folderPath.startsWith("/")) folderPath = folderPath.slice(1);

  const name = folderPath.replace(/\/$/, "").split("/").pop() || folderPath;

  // Check if already exists
  const existing = await c.env.DB.prepare(
    "SELECT id FROM objects WHERE owner = ? AND path = ?",
  )
    .bind(actor, folderPath)
    .first();

  if (existing) {
    return c.json({ path: folderPath, name, created: false });
  }

  // Ensure parent folders exist
  const parts = folderPath.replace(/\/$/, "").split("/");
  parts.pop(); // remove the folder we're creating
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
        "INSERT OR IGNORE INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, created_at, updated_at) VALUES (?, ?, ?, ?, 1, '', 0, '', ?, ?)",
      )
        .bind(objectId(), actor, parentPath, part, now, now)
        .run();
    }
  }

  const now = Date.now();
  const id = objectId();
  await c.env.DB.prepare(
    "INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, created_at, updated_at) VALUES (?, ?, ?, ?, 1, '', 0, '', ?, ?)",
  )
    .bind(id, actor, folderPath, name, now, now)
    .run();

  return c.json({ id, path: folderPath, name, created: true }, 201);
}

// GET /folders  or  GET /folders/*path
export async function listFolder(c: AppContext) {
  const actor = c.get("actor");
  let prefix = wildcardPath(c, "/folders/");

  // Normalize prefix
  if (prefix && !prefix.endsWith("/")) prefix += "/";

  // List direct children: objects whose path starts with prefix
  // and has no additional "/" after the prefix (except for folders which end in /)
  const { results } = await c.env.DB.prepare(`
    SELECT id, path, name, is_folder, content_type, size, created_at, updated_at
    FROM objects
    WHERE owner = ? AND path LIKE ? AND path != ?
    ORDER BY is_folder DESC, name ASC
  `)
    .bind(actor, prefix + "%", prefix)
    .all<ObjectRow>();

  // Filter to direct children only
  const items = (results || []).filter((obj) => {
    const rest = obj.path.slice(prefix.length);
    if (obj.is_folder) {
      // Folder: rest should be "name/" (no additional slashes except trailing)
      return rest.replace(/\/$/, "").indexOf("/") === -1;
    }
    // File: rest should have no slashes
    return rest.indexOf("/") === -1;
  });

  const mapped = items.map((o) => ({
    id: o.id,
    name: o.name,
    path: o.path,
    is_folder: !!o.is_folder,
    content_type: o.content_type,
    size: o.size,
    created_at: o.created_at,
    updated_at: o.updated_at,
  }));

  return c.json({ path: prefix || "/", items: mapped });
}

// DELETE /folders/*path
export async function deleteFolder(c: AppContext) {
  const actor = c.get("actor");
  let folderPath = wildcardPath(c, "/folders/");

  if (!folderPath) {
    return errorResponse(c, "invalid_request", "Folder path is required");
  }
  if (!folderPath.endsWith("/")) folderPath += "/";

  // Check folder exists
  const folder = await c.env.DB.prepare(
    "SELECT id FROM objects WHERE owner = ? AND path = ? AND is_folder = 1",
  )
    .bind(actor, folderPath)
    .first<{ id: string }>();

  if (!folder) {
    return errorResponse(c, "not_found", "Folder not found");
  }

  // Check if folder has children
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
