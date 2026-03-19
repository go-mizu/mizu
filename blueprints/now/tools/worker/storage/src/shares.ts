import type { Context } from "hono";
import type { Env, Variables, ShareRow, ObjectRow } from "./types";
import { shareId } from "./id";
import { isInlineType } from "./mime";
import { errorResponse } from "./error";
import { wildcardPath } from "./path";
import { resolvePermission, hasPermission, normalizeSharePermission, requireScope, sanitizeFilename } from "./authorize";
import { audit } from "./audit";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

/**
 * Parse owner and file path from /shared/ wildcard path.
 * Actor names always have the form "prefix/name" (e.g. "u/alice", "a/ci-bot"),
 * so we split on the second "/" to separate owner from file path.
 */
function parseSharedPath(c: AppContext): { owner: string; filePath: string } {
  const rest = wildcardPath(c, "/shared/");
  // Actor names are "u/xxx" or "a/xxx", so find the second slash
  const firstSlash = rest.indexOf("/");
  if (firstSlash === -1) return { owner: "", filePath: "" };
  const secondSlash = rest.indexOf("/", firstSlash + 1);
  if (secondSlash === -1) return { owner: rest, filePath: "" };
  return { owner: rest.slice(0, secondSlash), filePath: rest.slice(secondSlash + 1) };
}

// POST /shares  { "path": "docs/readme.md", "grantee": "u/bob", "permission": "editor" }
export async function createShare(c: AppContext) {
  const scopeErr = requireScope(c, "shares:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");

  let body: { path?: string; grantee?: string; permission?: string };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.path || !body.grantee) {
    return errorResponse(c, "invalid_request", "path and grantee are required");
  }

  const permission = normalizeSharePermission(body.permission || "viewer");

  // Find the object (must be owned by the actor)
  const obj = await c.env.DB.prepare(
    "SELECT id FROM objects WHERE owner = ? AND path = ?",
  )
    .bind(actor, body.path)
    .first<{ id: string }>();

  if (!obj) {
    return errorResponse(c, "not_found", "Object not found");
  }

  // Check grantee exists
  const grantee = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
    .bind(body.grantee)
    .first();
  if (!grantee) {
    return errorResponse(c, "not_found", "Grantee actor not found");
  }

  // Check duplicate
  const existing = await c.env.DB.prepare(
    "SELECT id FROM shares WHERE object_id = ? AND grantee = ?",
  )
    .bind(obj.id, body.grantee)
    .first();
  if (existing) {
    return c.json({ id: (existing as any).id, updated: false });
  }

  const id = shareId();
  const now = Date.now();
  await c.env.DB.prepare(
    "INSERT INTO shares (id, object_id, owner, grantee, permission, created_at) VALUES (?, ?, ?, ?, ?, ?)",
  )
    .bind(id, obj.id, actor, body.grantee, permission, now)
    .run();

  audit(c, "share.create", body.path, { grantee: body.grantee, permission });

  return c.json(
    { id, path: body.path, grantee: body.grantee, permission, created_at: now },
    201,
  );
}

// GET /shares — list shares I created + shares granted to me
export async function listShares(c: AppContext) {
  const scopeErr = requireScope(c, "shares:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");

  const { results: given } = await c.env.DB.prepare(`
    SELECT s.id, s.object_id, s.owner, s.grantee, s.permission, s.created_at,
           o.path, o.name, o.is_folder
    FROM shares s JOIN objects o ON s.object_id = o.id
    WHERE s.owner = ?
    ORDER BY s.created_at DESC
  `)
    .bind(actor)
    .all();

  const { results: received } = await c.env.DB.prepare(`
    SELECT s.id, s.object_id, s.owner, s.grantee, s.permission, s.created_at,
           o.path, o.name, o.is_folder
    FROM shares s JOIN objects o ON s.object_id = o.id
    WHERE s.grantee = ?
    ORDER BY s.created_at DESC
  `)
    .bind(actor)
    .all();

  return c.json({ given: given || [], received: received || [] });
}

// PATCH /shares/:id — update share permission
export async function updateShare(c: AppContext) {
  const scopeErr = requireScope(c, "shares:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const id = c.req.param("id");

  let body: { permission?: string };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.permission) {
    return errorResponse(c, "invalid_request", "permission is required");
  }

  const share = await c.env.DB.prepare(
    "SELECT owner, grantee FROM shares WHERE id = ?",
  )
    .bind(id)
    .first<{ owner: string; grantee: string }>();

  if (!share) {
    return errorResponse(c, "not_found", "Share not found");
  }
  if (share.owner !== actor) {
    return errorResponse(c, "forbidden", "Only the share owner can update permissions");
  }

  const permission = normalizeSharePermission(body.permission);

  await c.env.DB.prepare("UPDATE shares SET permission = ? WHERE id = ?")
    .bind(permission, id)
    .run();

  return c.json({ id, permission });
}

// DELETE /shares/:id — owner or grantee can revoke
export async function deleteShare(c: AppContext) {
  const scopeErr = requireScope(c, "shares:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const id = c.req.param("id");

  const share = await c.env.DB.prepare(
    "SELECT owner, grantee FROM shares WHERE id = ?",
  )
    .bind(id)
    .first<{ owner: string; grantee: string }>();

  if (!share) {
    return errorResponse(c, "not_found", "Share not found");
  }
  // Both owner and grantee can revoke
  if (share.owner !== actor && share.grantee !== actor) {
    return errorResponse(c, "forbidden", "Not your share");
  }

  await c.env.DB.prepare("DELETE FROM shares WHERE id = ?").bind(id).run();
  audit(c, "share.revoke", undefined, { share_id: id });

  return c.json({ deleted: true });
}

// GET /shared — list files shared with me
export async function listSharedWithMe(c: AppContext) {
  const scopeErr = requireScope(c, "shares:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");

  const { results } = await c.env.DB.prepare(`
    SELECT o.id, o.owner, o.path, o.name, o.is_folder, o.content_type, o.size, o.updated_at,
           s.permission
    FROM shares s JOIN objects o ON s.object_id = o.id
    WHERE s.grantee = ?
    ORDER BY o.updated_at DESC
  `)
    .bind(actor)
    .all();

  return c.json({ items: results || [] });
}

// GET /shared/:owner/*path — download a shared file
export async function downloadSharedFile(c: AppContext) {
  const scopeErr = requireScope(c, "files:read");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const { owner, filePath } = parseSharedPath(c);

  if (!owner || !filePath) {
    return errorResponse(c, "invalid_request", "Owner and path are required");
  }

  // Find the object
  const obj = await c.env.DB.prepare(
    "SELECT * FROM objects WHERE owner = ? AND path = ? AND is_folder = 0",
  )
    .bind(owner, filePath)
    .first<ObjectRow>();

  if (!obj) {
    return errorResponse(c, "not_found", "File not found");
  }

  if (obj.trashed_at) {
    return errorResponse(c, "not_found", "File is in trash");
  }

  // Check permission using inheritance
  const role = await resolvePermission(c.env.DB, actor, owner, filePath);
  if (!role || !hasPermission(role, "viewer")) {
    return errorResponse(c, "forbidden", "No access to this file");
  }

  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (!r2Obj) {
    return errorResponse(c, "not_found", "File data not found");
  }

  audit(c, "file.download", filePath, { owner, via: "shared" });

  const headers = new Headers();
  headers.set("Content-Type", obj.content_type || "application/octet-stream");
  headers.set("Content-Length", obj.size.toString());

  const safeName = sanitizeFilename(obj.name);
  if (isInlineType(obj.content_type)) {
    headers.set("Content-Disposition", `inline; filename="${safeName}"`);
  } else {
    headers.set("Content-Disposition", `attachment; filename="${safeName}"`);
  }

  return new Response(r2Obj.body, { headers });
}

// PUT /shared/:owner/*path — upload/overwrite a shared file
export async function uploadSharedFile(c: AppContext) {
  const scopeErr = requireScope(c, "files:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const { owner, filePath } = parseSharedPath(c);

  if (!owner || !filePath || filePath.endsWith("/")) {
    return errorResponse(c, "invalid_request", "Owner and file path are required");
  }

  // Check permission: need editor or uploader
  const role = await resolvePermission(c.env.DB, actor, owner, filePath);
  if (!role || !hasPermission(role, "uploader")) {
    return errorResponse(c, "forbidden", "No write access to this path");
  }

  const MAX_FILE_SIZE = 100 * 1024 * 1024;
  const cl = c.req.header("Content-Length");
  if (cl && parseInt(cl, 10) > MAX_FILE_SIZE) {
    return errorResponse(c, "too_large", "File exceeds 100MB limit");
  }

  const body = await c.req.arrayBuffer();
  if (body.byteLength > MAX_FILE_SIZE) {
    return errorResponse(c, "too_large", "File exceeds 100MB limit");
  }

  const name = filePath.split("/").pop() || filePath;
  const { mimeFromName } = await import("./mime");
  const contentType = c.req.header("Content-Type") || mimeFromName(name);
  const r2Key = `${owner}/${filePath}`;

  // Upload to R2 under the owner's namespace
  await c.env.BUCKET.put(r2Key, body, {
    httpMetadata: { contentType },
  });

  // Ensure parent folders exist under the owner
  const { ensureParentFolders } = await import("./files");
  await ensureParentFolders(c.env.DB, owner, filePath);

  // Upsert metadata (file belongs to owner, not uploader)
  const now = Date.now();
  const { objectId } = await import("./id");
  const existing = await c.env.DB.prepare(
    "SELECT id FROM objects WHERE owner = ? AND path = ?",
  )
    .bind(owner, filePath)
    .first<{ id: string }>();

  let id: string;
  if (existing) {
    // Uploaders can only overwrite their own uploads (not others')
    // Editors can overwrite anything
    if (role === "uploader") {
      return errorResponse(c, "forbidden", "Uploaders cannot overwrite existing files");
    }
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
      .bind(id, owner, filePath, name, contentType, body.byteLength, r2Key, now, now)
      .run();
  }

  audit(c, "file.upload", filePath, { owner, via: "shared", uploader: actor });

  return c.json(
    { id, path: filePath, name, content_type: contentType, size: body.byteLength, created_at: now },
    existing ? 200 : 201,
  );
}

// DELETE /shared/:owner/*path — delete a shared file (editor only)
export async function deleteSharedFile(c: AppContext) {
  const scopeErr = requireScope(c, "files:write");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const { owner, filePath } = parseSharedPath(c);

  if (!owner || !filePath) {
    return errorResponse(c, "invalid_request", "Owner and file path are required");
  }

  // Need editor permission
  const role = await resolvePermission(c.env.DB, actor, owner, filePath);
  if (!role || !hasPermission(role, "editor")) {
    return errorResponse(c, "forbidden", "No delete access to this file");
  }

  const obj = await c.env.DB.prepare(
    "SELECT id, r2_key FROM objects WHERE owner = ? AND path = ? AND is_folder = 0 AND trashed_at IS NULL",
  )
    .bind(owner, filePath)
    .first<{ id: string; r2_key: string }>();

  if (!obj) {
    return errorResponse(c, "not_found", "File not found");
  }

  await c.env.BUCKET.delete(obj.r2_key);
  await c.env.DB.prepare("DELETE FROM shares WHERE object_id = ?").bind(obj.id).run();
  await c.env.DB.prepare("DELETE FROM objects WHERE id = ?").bind(obj.id).run();

  audit(c, "file.delete", filePath, { owner, via: "shared", deleted_by: actor });

  return c.json({ deleted: true });
}
