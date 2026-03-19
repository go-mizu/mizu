import type { Context } from "hono";
import type { Env, Variables, ObjectRow, PublicLinkRow } from "../types";
import { publicLinkId, publicLinkToken } from "../lib/id";
import { isInlineType } from "../lib/mime";
import { errorResponse } from "../lib/error";
import { wildcardPath } from "../lib/path";
import { requireScope, sanitizeFilename } from "../middleware/authorize";
import { audit } from "../lib/audit";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

// POST /links — create a public link
export async function createLink(c: AppContext) {
  const scopeErr = requireScope(c, "links:manage");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");

  let body: {
    path?: string;
    permission?: string;
    password?: string;
    expires_in?: number;
    max_downloads?: number;
  };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.path) {
    return errorResponse(c, "invalid_request", "path is required");
  }

  const obj = await c.env.DB.prepare(
    "SELECT id, is_folder FROM objects WHERE owner = ? AND path = ? AND trashed_at IS NULL",
  )
    .bind(actor, body.path)
    .first<{ id: string; is_folder: number }>();

  if (!obj) {
    return errorResponse(c, "not_found", "Object not found");
  }

  const permission = body.permission === "editor" ? "editor" : "viewer";
  const token = publicLinkToken();
  const id = publicLinkId();
  const now = Date.now();
  const expiresAt = body.expires_in ? now + body.expires_in * 1000 : null;

  // Hash password if provided
  let passwordHash: string | null = null;
  if (body.password) {
    passwordHash = await sha256(body.password);
  }

  await c.env.DB.prepare(
    `INSERT INTO public_links (id, object_id, owner, token, permission, password_hash, expires_at, max_downloads, download_count, created_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`,
  )
    .bind(id, obj.id, actor, token, permission, passwordHash, expiresAt, body.max_downloads || null, now)
    .run();

  const url = new URL(c.req.url);
  const linkUrl = `${url.origin}/p/${token}`;

  audit(c, "link.create", body.path, { link_id: id, permission });

  return c.json({
    id,
    url: linkUrl,
    token,
    path: body.path,
    permission,
    password_protected: !!body.password,
    expires_at: expiresAt,
    max_downloads: body.max_downloads || null,
    created_at: now,
  }, 201);
}

// GET /links — list my public links
export async function listLinks(c: AppContext) {
  const scopeErr = requireScope(c, "links:manage");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");

  const { results } = await c.env.DB.prepare(`
    SELECT pl.id, pl.token, pl.permission, pl.password_hash, pl.expires_at,
           pl.max_downloads, pl.download_count, pl.created_at,
           o.path, o.name, o.is_folder
    FROM public_links pl
    JOIN objects o ON pl.object_id = o.id
    WHERE pl.owner = ?
    ORDER BY pl.created_at DESC
  `)
    .bind(actor)
    .all();

  const url = new URL(c.req.url);
  const items = (results || []).map((row: any) => ({
    id: row.id,
    url: `${url.origin}/p/${row.token}`,
    path: row.path,
    name: row.name,
    is_folder: !!row.is_folder,
    permission: row.permission,
    password_protected: !!row.password_hash,
    expires_at: row.expires_at,
    max_downloads: row.max_downloads,
    download_count: row.download_count,
    created_at: row.created_at,
  }));

  return c.json({ items });
}

// DELETE /links/:id — revoke a public link
export async function deleteLink(c: AppContext) {
  const scopeErr = requireScope(c, "links:manage");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const id = c.req.param("id");

  const link = await c.env.DB.prepare(
    "SELECT owner FROM public_links WHERE id = ?",
  )
    .bind(id)
    .first<{ owner: string }>();

  if (!link) {
    return errorResponse(c, "not_found", "Link not found");
  }
  if (link.owner !== actor) {
    return errorResponse(c, "forbidden", "Not your link");
  }

  await c.env.DB.prepare("DELETE FROM public_links WHERE id = ?").bind(id).run();
  audit(c, "link.revoke", undefined, { link_id: id });

  return c.json({ deleted: true });
}

// GET /p/:token — access via public link (download file or list folder)
export async function accessPublicLink(c: AppContext) {
  const token = c.req.param("token");

  const link = await c.env.DB.prepare(`
    SELECT pl.*, o.path, o.name, o.is_folder, o.content_type, o.size, o.r2_key, o.owner as file_owner
    FROM public_links pl
    JOIN objects o ON pl.object_id = o.id
    WHERE pl.token = ?
  `)
    .bind(token)
    .first<PublicLinkRow & { path: string; name: string; is_folder: number; content_type: string; size: number; r2_key: string; file_owner: string }>();

  if (!link) {
    return errorResponse(c, "not_found", "Link not found or expired");
  }

  // Check expiry
  if (link.expires_at && Date.now() > link.expires_at) {
    return c.json({ error: { code: "gone", message: "Link has expired" } }, 410);
  }

  // Check max downloads
  if (link.max_downloads && link.download_count >= link.max_downloads) {
    return c.json({ error: { code: "gone", message: "Download limit reached" } }, 410);
  }

  // Check password
  if (link.password_hash) {
    const password = c.req.header("X-Link-Password") || c.req.query("password") || "";
    if (!password) {
      return c.json({ error: { code: "unauthorized", message: "Password required" } }, 401);
    }
    const hash = await sha256(password);
    if (hash !== link.password_hash) {
      return c.json({ error: { code: "unauthorized", message: "Invalid password" } }, 401);
    }
  }

  if (link.is_folder) {
    // List folder contents
    return listPublicFolder(c, link.file_owner, link.path, token!);
  }

  // Download file
  const r2Obj = await c.env.BUCKET.get(link.r2_key);
  if (!r2Obj) {
    return errorResponse(c, "not_found", "File data not found");
  }

  // Increment download count
  c.executionCtx.waitUntil(
    c.env.DB.prepare("UPDATE public_links SET download_count = download_count + 1 WHERE id = ?")
      .bind(link.id)
      .run(),
  );

  const headers = new Headers();
  headers.set("Content-Type", link.content_type || "application/octet-stream");
  headers.set("Content-Length", link.size.toString());

  const safeName = sanitizeFilename(link.name);
  if (isInlineType(link.content_type)) {
    headers.set("Content-Disposition", `inline; filename="${safeName}"`);
  } else {
    headers.set("Content-Disposition", `attachment; filename="${safeName}"`);
  }

  return new Response(r2Obj.body, { headers });
}

// GET /p/:token/*path — access file within a shared folder
export async function accessPublicLinkFile(c: AppContext) {
  const token = c.req.param("token");
  const subPath = wildcardPath(c, `/p/${token}/`);

  const link = await c.env.DB.prepare(`
    SELECT pl.*, o.path, o.is_folder, o.owner as file_owner
    FROM public_links pl
    JOIN objects o ON pl.object_id = o.id
    WHERE pl.token = ? AND o.is_folder = 1
  `)
    .bind(token)
    .first<PublicLinkRow & { path: string; is_folder: number; file_owner: string }>();

  if (!link) {
    return errorResponse(c, "not_found", "Link not found or not a folder link");
  }

  // Check expiry
  if (link.expires_at && Date.now() > link.expires_at) {
    return c.json({ error: { code: "gone", message: "Link has expired" } }, 410);
  }

  // Check password
  if (link.password_hash) {
    const password = c.req.header("X-Link-Password") || c.req.query("password") || "";
    if (!password) {
      return c.json({ error: { code: "unauthorized", message: "Password required" } }, 401);
    }
    const hash = await sha256(password);
    if (hash !== link.password_hash) {
      return c.json({ error: { code: "unauthorized", message: "Invalid password" } }, 401);
    }
  }

  // Resolve the file within the shared folder
  const filePath = link.path + subPath;

  const obj = await c.env.DB.prepare(
    "SELECT * FROM objects WHERE owner = ? AND path = ? AND is_folder = 0 AND trashed_at IS NULL",
  )
    .bind(link.file_owner, filePath)
    .first<ObjectRow>();

  if (!obj) {
    return errorResponse(c, "not_found", "File not found");
  }

  const r2Obj = await c.env.BUCKET.get(obj.r2_key);
  if (!r2Obj) {
    return errorResponse(c, "not_found", "File data not found");
  }

  // Increment download count
  c.executionCtx.waitUntil(
    c.env.DB.prepare("UPDATE public_links SET download_count = download_count + 1 WHERE id = ?")
      .bind(link.id)
      .run(),
  );

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

async function listPublicFolder(c: AppContext, owner: string, folderPath: string, token: string) {
  const prefix = folderPath.endsWith("/") ? folderPath : folderPath + "/";

  const { results } = await c.env.DB.prepare(`
    SELECT id, path, name, is_folder, content_type, size, created_at, updated_at
    FROM objects
    WHERE owner = ? AND path LIKE ? AND path != ? AND trashed_at IS NULL
    ORDER BY is_folder DESC, name ASC
  `)
    .bind(owner, prefix + "%", prefix)
    .all<ObjectRow>();

  // Filter to direct children only
  const items = (results || []).filter((obj) => {
    const rest = obj.path.slice(prefix.length);
    if (obj.is_folder) {
      return rest.replace(/\/$/, "").indexOf("/") === -1;
    }
    return rest.indexOf("/") === -1;
  });

  const url = new URL(c.req.url);
  return c.json({
    path: prefix,
    items: items.map((o) => ({
      name: o.name,
      path: o.path.slice(prefix.length),
      is_folder: !!o.is_folder,
      content_type: o.content_type,
      size: o.size,
      download_url: o.is_folder
        ? undefined
        : `${url.origin}/p/${token}/${o.path.slice(prefix.length)}`,
    })),
  });
}

async function sha256(data: string): Promise<string> {
  const encoded = new TextEncoder().encode(data);
  const hash = await crypto.subtle.digest("SHA-256", encoded);
  return Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0")).join("");
}
