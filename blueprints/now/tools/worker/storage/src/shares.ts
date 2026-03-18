import type { Context } from "hono";
import type { Env, Variables, ShareRow, ObjectRow } from "./types";
import { shareId } from "./id";
import { isInlineType } from "./mime";
import { errorResponse } from "./error";
import { wildcardPath } from "./path";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

// POST /shares  { "path": "docs/readme.md", "grantee": "u/bob", "permission": "read" }
export async function createShare(c: AppContext) {
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

  const permission = body.permission === "write" ? "write" : "read";

  // Find the object
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

  return c.json(
    { id, path: body.path, grantee: body.grantee, permission, created_at: now },
    201,
  );
}

// GET /shares — list shares I created + shares granted to me
export async function listShares(c: AppContext) {
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

// DELETE /shares/:id
export async function deleteShare(c: AppContext) {
  const actor = c.get("actor");
  const id = c.req.param("id");

  const share = await c.env.DB.prepare(
    "SELECT owner FROM shares WHERE id = ?",
  )
    .bind(id)
    .first<{ owner: string }>();

  if (!share) {
    return errorResponse(c, "not_found", "Share not found");
  }
  if (share.owner !== actor) {
    return errorResponse(c, "forbidden", "Not your share");
  }

  await c.env.DB.prepare("DELETE FROM shares WHERE id = ?").bind(id).run();
  return c.json({ deleted: true });
}

// GET /shared — list files shared with me
export async function listSharedWithMe(c: AppContext) {
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
  const actor = c.get("actor");
  // URL: /shared/u%2Falice/docs/readme.md → owner="u/alice", filePath="docs/readme.md"
  const rest = wildcardPath(c, "/shared/");
  const slashIdx = rest.indexOf("/");
  if (slashIdx === -1) {
    return errorResponse(c, "invalid_request", "Owner and path are required");
  }
  const owner = rest.slice(0, slashIdx);
  const filePath = rest.slice(slashIdx + 1);

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

  // Check share permission
  const share = await c.env.DB.prepare(
    "SELECT 1 FROM shares WHERE object_id = ? AND grantee = ?",
  )
    .bind(obj.id, actor)
    .first();

  if (!share) {
    return errorResponse(c, "forbidden", "No access to this file");
  }

  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (!r2Obj) {
    return errorResponse(c, "not_found", "File data not found");
  }

  const headers = new Headers();
  headers.set("Content-Type", obj.content_type || "application/octet-stream");
  headers.set("Content-Length", obj.size.toString());

  if (isInlineType(obj.content_type)) {
    headers.set("Content-Disposition", "inline");
  } else {
    headers.set("Content-Disposition", `attachment; filename="${obj.name}"`);
  }

  return new Response(r2Obj.body, { headers });
}
