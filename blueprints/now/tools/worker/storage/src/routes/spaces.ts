import type { Context } from "hono";
import type { Env, Variables } from "../types";
import { spaceId, spaceMemberId, spaceSectionId, spaceItemId, spaceActivityId } from "../lib/id";
import { errorResponse } from "../lib/error";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

// ── GET /spaces — list spaces (owned + member of) ─────────────────────
export async function listSpaces(c: AppContext) {
  const actor = c.get("actor");

  // Spaces owned by actor
  const { results: owned } = await c.env.DB.prepare(`
    SELECT s.*,
      (SELECT COUNT(*) FROM space_items WHERE space_id = s.id) AS item_count,
      (SELECT COUNT(*) FROM space_members WHERE space_id = s.id) + 1 AS member_count,
      (SELECT GROUP_CONCAT(actor) FROM (SELECT actor FROM space_members WHERE space_id = s.id ORDER BY created_at ASC LIMIT 4)) AS top_members
    FROM spaces s
    WHERE s.owner = ?
    ORDER BY s.updated_at DESC
  `).bind(actor).all();

  // Spaces where actor is a member
  const { results: memberOf } = await c.env.DB.prepare(`
    SELECT s.*,
      sm.role AS my_role,
      (SELECT COUNT(*) FROM space_items WHERE space_id = s.id) AS item_count,
      (SELECT COUNT(*) FROM space_members WHERE space_id = s.id) + 1 AS member_count,
      (SELECT GROUP_CONCAT(actor) FROM (SELECT actor FROM space_members WHERE space_id = s.id ORDER BY created_at ASC LIMIT 4)) AS top_members
    FROM space_members sm
    JOIN spaces s ON sm.space_id = s.id
    WHERE sm.actor = ?
    ORDER BY s.updated_at DESC
  `).bind(actor).all();

  return c.json({ owned: owned || [], member_of: memberOf || [] });
}

// ── POST /spaces — create a space ─────────────────────────────────────
export async function createSpace(c: AppContext) {
  const actor = c.get("actor");
  let body: { title?: string; description?: string; icon?: string; visibility?: string };
  try { body = await c.req.json(); } catch { return errorResponse(c, "invalid_request", "Invalid JSON"); }

  const title = body.title?.trim();
  if (!title) return errorResponse(c, "invalid_request", "Title is required");

  const id = spaceId();
  const now = Date.now();
  await c.env.DB.prepare(`
    INSERT INTO spaces (id, owner, title, description, cover_url, icon, visibility, created_at, updated_at)
    VALUES (?, ?, ?, ?, '', ?, ?, ?, ?)
  `).bind(id, actor, title, body.description || "", body.icon || "", body.visibility || "private", now, now).run();

  // Log activity
  await c.env.DB.prepare(`
    INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
    VALUES (?, ?, ?, 'created', ?, ?)
  `).bind(spaceActivityId(), id, actor, title, now).run();

  return c.json({ id, title, owner: actor, created_at: now }, 201);
}

// ── GET /spaces/:id — space detail with sections, items, members ──────
export async function getSpace(c: AppContext) {
  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const space = await c.env.DB.prepare("SELECT * FROM spaces WHERE id = ?").bind(id).first();
  if (!space) return errorResponse(c, "not_found", "Space not found");

  // Check access: owner or member
  const isOwner = (space as any).owner === actor;
  if (!isOwner) {
    const membership = await c.env.DB.prepare(
      "SELECT role FROM space_members WHERE space_id = ? AND actor = ?"
    ).bind(id, actor).first();
    if (!membership && (space as any).visibility !== "public") {
      return errorResponse(c, "forbidden", "No access to this space");
    }
  }

  // Sections
  const { results: sections } = await c.env.DB.prepare(
    "SELECT * FROM space_sections WHERE space_id = ? ORDER BY position ASC"
  ).bind(id).all();

  // Items with file info
  const { results: items } = await c.env.DB.prepare(`
    SELECT si.*, o.path AS file_path, o.name AS file_name, o.content_type AS file_content_type, o.size AS file_size
    FROM space_items si
    LEFT JOIN objects o ON si.object_id = o.id
    WHERE si.space_id = ?
    ORDER BY si.section_id, si.position ASC
  `).bind(id).all();

  // Members
  const { results: members } = await c.env.DB.prepare(`
    SELECT sm.*, a.type AS actor_type, a.email AS actor_email, a.bio AS actor_bio
    FROM space_members sm
    LEFT JOIN actors a ON sm.actor = a.actor
    WHERE sm.space_id = ?
  `).bind(id).all();

  // Owner info
  const ownerInfo = await c.env.DB.prepare(
    "SELECT actor, type, email, bio FROM actors WHERE actor = ?"
  ).bind((space as any).owner).first();

  // Recent activity
  const { results: activity } = await c.env.DB.prepare(`
    SELECT sa.*, a.type AS actor_type
    FROM space_activity sa
    LEFT JOIN actors a ON sa.actor = a.actor
    WHERE sa.space_id = ?
    ORDER BY sa.created_at DESC LIMIT 20
  `).bind(id).all();

  return c.json({
    space,
    owner_info: ownerInfo,
    sections: sections || [],
    items: items || [],
    members: members || [],
    activity: activity || [],
  });
}

// ── PATCH /spaces/:id — update space ──────────────────────────────────
export async function updateSpace(c: AppContext) {
  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const space = await c.env.DB.prepare("SELECT owner FROM spaces WHERE id = ?").bind(id).first<{ owner: string }>();
  if (!space) return errorResponse(c, "not_found", "Space not found");
  if (space.owner !== actor) {
    const m = await c.env.DB.prepare(
      "SELECT role FROM space_members WHERE space_id = ? AND actor = ?"
    ).bind(id, actor).first<{ role: string }>();
    if (!m || (m.role !== "editor" && m.role !== "admin"))
      return errorResponse(c, "forbidden", "Not allowed");
  }

  let body: Record<string, any>;
  try { body = await c.req.json(); } catch { return errorResponse(c, "invalid_request", "Invalid JSON"); }

  const sets: string[] = [];
  const vals: any[] = [];
  for (const key of ["title", "description", "cover_url", "icon", "visibility"]) {
    if (body[key] !== undefined) { sets.push(`${key} = ?`); vals.push(body[key]); }
  }
  if (!sets.length) return errorResponse(c, "invalid_request", "Nothing to update");

  sets.push("updated_at = ?"); vals.push(Date.now());
  vals.push(id);

  await c.env.DB.prepare(`UPDATE spaces SET ${sets.join(", ")} WHERE id = ?`).bind(...vals).run();
  return c.json({ ok: true });
}

// ── DELETE /spaces/:id — delete space (owner only) ────────────────────
export async function deleteSpace(c: AppContext) {
  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const space = await c.env.DB.prepare("SELECT owner FROM spaces WHERE id = ?").bind(id).first<{ owner: string }>();
  if (!space) return errorResponse(c, "not_found", "Space not found");
  if (space.owner !== actor) return errorResponse(c, "forbidden", "Only the owner can delete a space");

  await c.env.DB.prepare("DELETE FROM spaces WHERE id = ?").bind(id).run();
  return c.json({ ok: true, deleted: id });
}

// ── POST /spaces/:id/sections — add section ──────────────────────────
export async function addSection(c: AppContext) {
  const actor = c.get("actor");
  const spaceId_ = c.req.param("id")!;
  if (!(await canEdit(c.env.DB, spaceId_, actor)))
    return errorResponse(c, "forbidden", "Not allowed");

  let body: { title?: string; description?: string };
  try { body = await c.req.json(); } catch { return errorResponse(c, "invalid_request", "Invalid JSON"); }

  const title = body.title?.trim();
  if (!title) return errorResponse(c, "invalid_request", "Title required");

  // Get next position
  const last = await c.env.DB.prepare(
    "SELECT MAX(position) AS p FROM space_sections WHERE space_id = ?"
  ).bind(spaceId_).first<{ p: number | null }>();
  const pos = (last?.p ?? -1) + 1;

  const id = spaceSectionId();
  const now = Date.now();
  await c.env.DB.prepare(`
    INSERT INTO space_sections (id, space_id, title, description, position, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, ?)
  `).bind(id, spaceId_, title, body.description || "", pos, now, now).run();

  await logActivity(c.env.DB, spaceId_, actor, "added_section", title);
  return c.json({ id, title, position: pos }, 201);
}

// ── POST /spaces/:id/items — add item ────────────────────────────────
export async function addItem(c: AppContext) {
  const actor = c.get("actor");
  const spaceId_ = c.req.param("id")!;
  if (!(await canEdit(c.env.DB, spaceId_, actor)))
    return errorResponse(c, "forbidden", "Not allowed");

  let body: {
    section_id?: string; item_type?: string; title?: string; description?: string;
    object_id?: string; url?: string; note_body?: string;
  };
  try { body = await c.req.json(); } catch { return errorResponse(c, "invalid_request", "Invalid JSON"); }

  if (!body.section_id || !body.item_type || !body.title?.trim())
    return errorResponse(c, "invalid_request", "section_id, item_type, and title required");

  const last = await c.env.DB.prepare(
    "SELECT MAX(position) AS p FROM space_items WHERE section_id = ?"
  ).bind(body.section_id).first<{ p: number | null }>();
  const pos = (last?.p ?? -1) + 1;

  const id = spaceItemId();
  const now = Date.now();
  await c.env.DB.prepare(`
    INSERT INTO space_items (id, section_id, space_id, item_type, title, description, object_id, url, note_body, position, added_by, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  `).bind(
    id, body.section_id, spaceId_, body.item_type, body.title.trim(),
    body.description || "", body.object_id || null, body.url || null,
    body.note_body || null, pos, actor, now, now,
  ).run();

  await logActivity(c.env.DB, spaceId_, actor, "added_item", body.title.trim());
  await c.env.DB.prepare("UPDATE spaces SET updated_at = ? WHERE id = ?").bind(now, spaceId_).run();
  return c.json({ id, title: body.title.trim(), position: pos }, 201);
}

// ── POST /spaces/:id/members — add member ────────────────────────────
export async function addMember(c: AppContext) {
  const actor = c.get("actor");
  const spaceId_ = c.req.param("id")!;

  const space = await c.env.DB.prepare("SELECT owner FROM spaces WHERE id = ?").bind(spaceId_).first<{ owner: string }>();
  if (!space) return errorResponse(c, "not_found", "Space not found");
  if (space.owner !== actor) {
    const m = await c.env.DB.prepare(
      "SELECT role FROM space_members WHERE space_id = ? AND actor = ?"
    ).bind(spaceId_, actor).first<{ role: string }>();
    if (!m || m.role !== "admin") return errorResponse(c, "forbidden", "Only owner/admin can add members");
  }

  let body: { actor?: string; role?: string };
  try { body = await c.req.json(); } catch { return errorResponse(c, "invalid_request", "Invalid JSON"); }

  if (!body.actor) return errorResponse(c, "invalid_request", "actor required");

  const id = spaceMemberId();
  const now = Date.now();
  await c.env.DB.prepare(`
    INSERT OR REPLACE INTO space_members (id, space_id, actor, role, created_at)
    VALUES (?, ?, ?, ?, ?)
  `).bind(id, spaceId_, body.actor, body.role || "viewer", now).run();

  await logActivity(c.env.DB, spaceId_, actor, "shared", `with ${body.actor}`);
  return c.json({ id, actor: body.actor, role: body.role || "viewer" }, 201);
}

// ── GET /spaces/:id/members — list members ───────────────────────────
export async function listMembers(c: AppContext) {
  const id = c.req.param("id")!;
  const { results } = await c.env.DB.prepare(`
    SELECT sm.*, a.type AS actor_type, a.email AS actor_email, a.bio AS actor_bio
    FROM space_members sm LEFT JOIN actors a ON sm.actor = a.actor
    WHERE sm.space_id = ?
  `).bind(id).all();
  return c.json({ members: results || [] });
}

// ── GET /spaces/:id/activity — activity feed ─────────────────────────
export async function listActivity(c: AppContext) {
  const id = c.req.param("id")!;
  const { results } = await c.env.DB.prepare(`
    SELECT sa.*, a.type AS actor_type
    FROM space_activity sa LEFT JOIN actors a ON sa.actor = a.actor
    WHERE sa.space_id = ?
    ORDER BY sa.created_at DESC LIMIT 50
  `).bind(id).all();
  return c.json({ activity: results || [] });
}

// ── GET /spaces/feed — aggregated activity across all spaces ─────────
export async function spacesFeed(c: AppContext) {
  const actor = c.get("actor");
  const { results } = await c.env.DB.prepare(`
    SELECT sa.*, a.type AS actor_type, s.title AS space_title, s.icon AS space_icon
    FROM space_activity sa
    LEFT JOIN actors a ON sa.actor = a.actor
    LEFT JOIN spaces s ON sa.space_id = s.id
    WHERE sa.space_id IN (
      SELECT id FROM spaces WHERE owner = ?
      UNION
      SELECT space_id FROM space_members WHERE actor = ?
    )
    ORDER BY sa.created_at DESC
    LIMIT 30
  `).bind(actor, actor).all();
  return c.json({ feed: results || [] });
}

// ── Helpers ──────────────────────────────────────────────────────────

async function canEdit(db: D1Database, spaceId: string, actor: string): Promise<boolean> {
  const space = await db.prepare("SELECT owner FROM spaces WHERE id = ?").bind(spaceId).first<{ owner: string }>();
  if (!space) return false;
  if (space.owner === actor) return true;
  const m = await db.prepare(
    "SELECT role FROM space_members WHERE space_id = ? AND actor = ?"
  ).bind(spaceId, actor).first<{ role: string }>();
  return m?.role === "editor" || m?.role === "admin";
}

async function logActivity(db: D1Database, spaceId: string, actor: string, action: string, target: string) {
  await db.prepare(`
    INSERT INTO space_activity (id, space_id, actor, action, target, created_at)
    VALUES (?, ?, ?, ?, ?, ?)
  `).bind(spaceActivityId(), spaceId, actor, action, target, Date.now()).run();
}
