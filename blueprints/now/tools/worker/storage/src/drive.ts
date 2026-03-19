import type { Context } from "hono";
import type { Env, Variables, ObjectRow } from "./types";
import { objectId } from "./id";
import { errorResponse } from "./error";
import { requireScope } from "./authorize";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

// ── PATCH /drive/star — toggle star ──────────────────────────────────
export async function toggleStar(c: AppContext) {
  const scopeErr = requireScope(c, "drive:write");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const { path, starred } = await c.req.json<{ path: string; starred: number }>();
  if (!path) return errorResponse(c, "invalid_request", "path required");

  const result = await c.env.DB.prepare(
    "UPDATE objects SET starred = ?, updated_at = ? WHERE owner = ? AND path = ? AND trashed_at IS NULL",
  )
    .bind(starred ? 1 : 0, Date.now(), actor, path)
    .run();

  if (!result.meta.changes) return errorResponse(c, "not_found", "Not found");
  return c.json({ path, starred: starred ? 1 : 0 });
}

// ── POST /drive/rename ───────────────────────────────────────────────
export async function renameItem(c: AppContext) {
  const scopeErr = requireScope(c, "drive:write");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const { path, new_name } = await c.req.json<{ path: string; new_name: string }>();
  if (!path || !new_name) return errorResponse(c, "invalid_request", "path and new_name required");
  if (new_name.includes("/")) return errorResponse(c, "invalid_request", "Name cannot contain /");

  const obj = await c.env.DB.prepare(
    "SELECT * FROM objects WHERE owner = ? AND path = ? AND trashed_at IS NULL",
  )
    .bind(actor, path)
    .first<ObjectRow>();
  if (!obj) return errorResponse(c, "not_found", "Not found");

  // Compute new path
  const parts = path.replace(/\/$/, "").split("/");
  parts[parts.length - 1] = new_name;
  let newPath = parts.join("/");
  if (obj.is_folder) newPath += "/";

  // Check no conflict
  const conflict = await c.env.DB.prepare(
    "SELECT 1 FROM objects WHERE owner = ? AND path = ?",
  ).bind(actor, newPath).first();
  if (conflict) return errorResponse(c, "conflict", "An item with that name already exists");

  const now = Date.now();
  if (obj.is_folder) {
    // Rename folder + update all children paths
    const oldPrefix = path.endsWith("/") ? path : path + "/";
    const newPrefix = newPath.endsWith("/") ? newPath : newPath + "/";
    await c.env.DB.prepare(
      "UPDATE objects SET path = ? || substr(path, ?), updated_at = ? WHERE owner = ? AND path LIKE ?",
    ).bind(newPrefix, oldPrefix.length + 1, now, actor, oldPrefix + "%").run();
    // Rename the folder itself
    await c.env.DB.prepare(
      "UPDATE objects SET path = ?, name = ?, updated_at = ? WHERE id = ?",
    ).bind(newPath, new_name, now, obj.id).run();
    // Update R2 keys for files under this folder
    const { results } = await c.env.DB.prepare(
      "SELECT id, path FROM objects WHERE owner = ? AND path LIKE ? AND is_folder = 0",
    ).bind(actor, newPrefix + "%").all<{ id: string; path: string }>();
    for (const child of results || []) {
      const newR2Key = `${actor}/${child.path}`;
      await c.env.DB.prepare("UPDATE objects SET r2_key = ? WHERE id = ?").bind(newR2Key, child.id).run();
    }
  } else {
    const newR2Key = `${actor}/${newPath}`;
    // Rename in R2: copy then delete
    const oldR2Obj = await c.env.BUCKET.get(obj.r2_key);
    if (oldR2Obj) {
      await c.env.BUCKET.put(newR2Key, oldR2Obj.body, {
        httpMetadata: { contentType: obj.content_type },
      });
      await c.env.BUCKET.delete(obj.r2_key);
    }
    await c.env.DB.prepare(
      "UPDATE objects SET path = ?, name = ?, r2_key = ?, updated_at = ? WHERE id = ?",
    ).bind(newPath, new_name, newR2Key, now, obj.id).run();
  }

  return c.json({ id: obj.id, path: newPath, name: new_name });
}

// ── POST /drive/move ─────────────────────────────────────────────────
export async function moveItems(c: AppContext) {
  const scopeErr = requireScope(c, "drive:write");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const { paths, destination } = await c.req.json<{ paths: string[]; destination: string }>();
  if (!paths?.length) return errorResponse(c, "invalid_request", "paths required");

  const dest = destination?.endsWith("/") ? destination : (destination || "") + "/";
  // Normalize: empty string or "/" means root
  const destPrefix = dest === "/" ? "" : dest;

  const now = Date.now();
  let moved = 0;

  for (const path of paths) {
    const obj = await c.env.DB.prepare(
      "SELECT * FROM objects WHERE owner = ? AND path = ? AND trashed_at IS NULL",
    ).bind(actor, path).first<ObjectRow>();
    if (!obj) continue;

    const newPath = destPrefix + obj.name + (obj.is_folder ? "/" : "");

    // Check no conflict
    const conflict = await c.env.DB.prepare(
      "SELECT 1 FROM objects WHERE owner = ? AND path = ?",
    ).bind(actor, newPath).first();
    if (conflict) continue;

    if (obj.is_folder) {
      const oldPrefix = path.endsWith("/") ? path : path + "/";
      const newPrefix = newPath.endsWith("/") ? newPath : newPath + "/";
      // Move children
      await c.env.DB.prepare(
        "UPDATE objects SET path = ? || substr(path, ?), updated_at = ? WHERE owner = ? AND path LIKE ?",
      ).bind(newPrefix, oldPrefix.length + 1, now, actor, oldPrefix + "%").run();
      await c.env.DB.prepare(
        "UPDATE objects SET path = ?, updated_at = ? WHERE id = ?",
      ).bind(newPath, now, obj.id).run();
    } else {
      const newR2Key = `${actor}/${newPath}`;
      const r2Obj = await c.env.BUCKET.get(obj.r2_key);
      if (r2Obj) {
        await c.env.BUCKET.put(newR2Key, r2Obj.body, {
          httpMetadata: { contentType: obj.content_type },
        });
        await c.env.BUCKET.delete(obj.r2_key);
      }
      await c.env.DB.prepare(
        "UPDATE objects SET path = ?, r2_key = ?, updated_at = ? WHERE id = ?",
      ).bind(newPath, newR2Key, now, obj.id).run();
    }
    moved++;
  }

  return c.json({ moved });
}

// ── POST /drive/copy ─────────────────────────────────────────────────
export async function copyFile(c: AppContext) {
  const scopeErr = requireScope(c, "drive:write");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const { path, destination } = await c.req.json<{ path: string; destination?: string }>();
  if (!path) return errorResponse(c, "invalid_request", "path required");

  const obj = await c.env.DB.prepare(
    "SELECT * FROM objects WHERE owner = ? AND path = ? AND is_folder = 0 AND trashed_at IS NULL",
  ).bind(actor, path).first<ObjectRow>();
  if (!obj) return errorResponse(c, "not_found", "File not found");

  const destDir = destination || path.substring(0, path.lastIndexOf("/") + 1);
  const ext = obj.name.includes(".") ? "." + obj.name.split(".").pop() : "";
  const base = ext ? obj.name.slice(0, -ext.length) : obj.name;
  let copyName = `${base} (copy)${ext}`;
  let copyPath = destDir + copyName;

  // Ensure unique name
  let i = 2;
  while (await c.env.DB.prepare("SELECT 1 FROM objects WHERE owner = ? AND path = ?").bind(actor, copyPath).first()) {
    copyName = `${base} (copy ${i})${ext}`;
    copyPath = destDir + copyName;
    i++;
  }

  const now = Date.now();
  const newId = objectId();
  const newR2Key = `${actor}/${copyPath}`;

  // Copy R2 object
  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (r2Obj) {
    await c.env.BUCKET.put(newR2Key, r2Obj.body, {
      httpMetadata: { contentType: obj.content_type },
    });
  }

  await c.env.DB.prepare(
    `INSERT INTO objects (id, owner, path, name, is_folder, content_type, size, r2_key, starred, description, created_at, updated_at)
     VALUES (?, ?, ?, ?, 0, ?, ?, ?, 0, '', ?, ?)`,
  ).bind(newId, actor, copyPath, copyName, obj.content_type, obj.size, newR2Key, now, now).run();

  return c.json({ id: newId, path: copyPath, name: copyName }, 201);
}

// ── POST /drive/trash ────────────────────────────────────────────────
export async function trashItems(c: AppContext) {
  const scopeErr = requireScope(c, "drive:write");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const { paths } = await c.req.json<{ paths: string[] }>();
  if (!paths?.length) return errorResponse(c, "invalid_request", "paths required");

  const now = Date.now();
  let trashed = 0;

  for (const path of paths) {
    const obj = await c.env.DB.prepare(
      "SELECT id, is_folder, path FROM objects WHERE owner = ? AND path = ? AND trashed_at IS NULL",
    ).bind(actor, path).first<{ id: string; is_folder: number; path: string }>();
    if (!obj) continue;

    if (obj.is_folder) {
      const prefix = path.endsWith("/") ? path : path + "/";
      await c.env.DB.prepare(
        "UPDATE objects SET trashed_at = ? WHERE owner = ? AND path LIKE ?",
      ).bind(now, actor, prefix + "%").run();
    }
    await c.env.DB.prepare(
      "UPDATE objects SET trashed_at = ? WHERE id = ?",
    ).bind(now, obj.id).run();
    trashed++;
  }

  return c.json({ trashed });
}

// ── POST /drive/restore ──────────────────────────────────────────────
export async function restoreItems(c: AppContext) {
  const scopeErr = requireScope(c, "drive:write");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const { paths } = await c.req.json<{ paths: string[] }>();
  if (!paths?.length) return errorResponse(c, "invalid_request", "paths required");

  let restored = 0;
  for (const path of paths) {
    const obj = await c.env.DB.prepare(
      "SELECT id, is_folder FROM objects WHERE owner = ? AND path = ? AND trashed_at IS NOT NULL",
    ).bind(actor, path).first<{ id: string; is_folder: number }>();
    if (!obj) continue;

    if (obj.is_folder) {
      const prefix = path.endsWith("/") ? path : path + "/";
      await c.env.DB.prepare(
        "UPDATE objects SET trashed_at = NULL WHERE owner = ? AND path LIKE ?",
      ).bind(actor, prefix + "%").run();
    }
    await c.env.DB.prepare("UPDATE objects SET trashed_at = NULL WHERE id = ?").bind(obj.id).run();
    restored++;
  }

  return c.json({ restored });
}

// ── DELETE /drive/trash — empty trash ────────────────────────────────
export async function emptyTrash(c: AppContext) {
  const scopeErr = requireScope(c, "drive:write");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");

  // Get all trashed files to delete from R2
  const { results } = await c.env.DB.prepare(
    "SELECT id, r2_key FROM objects WHERE owner = ? AND trashed_at IS NOT NULL AND is_folder = 0",
  ).bind(actor).all<{ id: string; r2_key: string }>();

  for (const obj of results || []) {
    if (obj.r2_key) await c.env.BUCKET.delete(obj.r2_key);
  }

  // Delete shares for trashed objects
  const { results: trashedIds } = await c.env.DB.prepare(
    "SELECT id FROM objects WHERE owner = ? AND trashed_at IS NOT NULL",
  ).bind(actor).all<{ id: string }>();
  for (const { id } of trashedIds || []) {
    await c.env.DB.prepare("DELETE FROM shares WHERE object_id = ?").bind(id).run();
  }

  // Delete all trashed objects
  const del = await c.env.DB.prepare(
    "DELETE FROM objects WHERE owner = ? AND trashed_at IS NOT NULL",
  ).bind(actor).run();

  return c.json({ deleted: del.meta.changes || 0 });
}

// ── GET /drive/trash — list trashed items ────────────────────────────
export async function listTrash(c: AppContext) {
  const scopeErr = requireScope(c, "drive:read");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const { results } = await c.env.DB.prepare(
    `SELECT id, path, name, is_folder, content_type, size, starred, trashed_at, created_at, updated_at
     FROM objects WHERE owner = ? AND trashed_at IS NOT NULL
     ORDER BY trashed_at DESC`,
  ).bind(actor).all<ObjectRow>();

  return c.json({ items: (results || []).map(mapObj) });
}

// ── GET /drive/recent ────────────────────────────────────────────────
export async function listRecent(c: AppContext) {
  const scopeErr = requireScope(c, "drive:read");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const { results } = await c.env.DB.prepare(
    `SELECT id, path, name, is_folder, content_type, size, starred, accessed_at, created_at, updated_at
     FROM objects WHERE owner = ? AND trashed_at IS NULL AND accessed_at IS NOT NULL AND is_folder = 0
     ORDER BY accessed_at DESC LIMIT 50`,
  ).bind(actor).all<ObjectRow>();

  return c.json({ items: (results || []).map(mapObj) });
}

// ── GET /drive/starred ───────────────────────────────────────────────
export async function listStarred(c: AppContext) {
  const scopeErr = requireScope(c, "drive:read");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const { results } = await c.env.DB.prepare(
    `SELECT id, path, name, is_folder, content_type, size, starred, created_at, updated_at
     FROM objects WHERE owner = ? AND starred = 1 AND trashed_at IS NULL
     ORDER BY updated_at DESC`,
  ).bind(actor).all<ObjectRow>();

  return c.json({ items: (results || []).map(mapObj) });
}

// ── GET /drive/search?q=&type=&starred= ──────────────────────────────
export async function searchFiles(c: AppContext) {
  const scopeErr = requireScope(c, "drive:read");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const q = c.req.query("q") || "";
  const type = c.req.query("type") || "";
  const starred = c.req.query("starred");

  let sql = `SELECT id, path, name, is_folder, content_type, size, starred, created_at, updated_at
    FROM objects WHERE owner = ? AND trashed_at IS NULL`;
  const binds: any[] = [actor];

  if (q) {
    sql += " AND name LIKE ?";
    binds.push(`%${q}%`);
  }
  if (type) {
    sql += " AND content_type LIKE ?";
    binds.push(`${type}%`);
  }
  if (starred === "1") {
    sql += " AND starred = 1";
  }

  sql += " ORDER BY updated_at DESC LIMIT 100";

  const { results } = await c.env.DB.prepare(sql).bind(...binds).all<ObjectRow>();

  return c.json({ items: (results || []).map(mapObj), query: q, count: results?.length || 0 });
}

// ── GET /drive/stats — storage statistics ────────────────────────────
export async function driveStats(c: AppContext) {
  const scopeErr = requireScope(c, "drive:read");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");

  const stats = await c.env.DB.prepare(
    `SELECT COUNT(*) as file_count, COALESCE(SUM(size),0) as total_size
     FROM objects WHERE owner = ? AND is_folder = 0 AND trashed_at IS NULL`,
  ).bind(actor).first<{ file_count: number; total_size: number }>();

  const folderCount = await c.env.DB.prepare(
    "SELECT COUNT(*) as count FROM objects WHERE owner = ? AND is_folder = 1 AND trashed_at IS NULL",
  ).bind(actor).first<{ count: number }>();

  const trashCount = await c.env.DB.prepare(
    "SELECT COUNT(*) as count FROM objects WHERE owner = ? AND trashed_at IS NOT NULL",
  ).bind(actor).first<{ count: number }>();

  return c.json({
    file_count: stats?.file_count || 0,
    folder_count: folderCount?.count || 0,
    total_size: stats?.total_size || 0,
    trash_count: trashCount?.count || 0,
    quota: 5 * 1024 * 1024 * 1024, // 5GB free tier
  });
}

// ── PATCH /drive/description ─────────────────────────────────────────
export async function updateDescription(c: AppContext) {
  const scopeErr = requireScope(c, "drive:write");
  if (scopeErr) return scopeErr;
  const actor = c.get("actor");
  const { path, description } = await c.req.json<{ path: string; description: string }>();
  if (!path) return errorResponse(c, "invalid_request", "path required");

  await c.env.DB.prepare(
    "UPDATE objects SET description = ?, updated_at = ? WHERE owner = ? AND path = ? AND trashed_at IS NULL",
  ).bind(description || "", Date.now(), actor, path).run();

  return c.json({ path, description: description || "" });
}

// ── Helper: map ObjectRow to API response ────────────────────────────
function mapObj(o: ObjectRow) {
  return {
    id: o.id,
    path: o.path,
    name: o.name,
    is_folder: !!o.is_folder,
    content_type: o.content_type,
    size: o.size,
    starred: !!o.starred,
    trashed_at: o.trashed_at,
    accessed_at: o.accessed_at,
    created_at: o.created_at,
    updated_at: o.updated_at,
  };
}
