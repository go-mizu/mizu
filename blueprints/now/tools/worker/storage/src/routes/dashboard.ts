import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { auth } from "../middleware/auth";
import { err } from "../lib/error";

type C = Context<{ Bindings: Env; Variables: Variables }>;

// ── GET /dashboard/audit ─────────────────────────────────────────────
async function auditHandler(c: C) {
  const actor = c.get("actor");
  const limit = Math.min(parseInt(c.req.query("limit") || "50", 10), 200);
  const offset = parseInt(c.req.query("offset") || "0", 10);
  const action = c.req.query("action") || "";

  let sql = "SELECT action, path, ip, ts FROM audit WHERE actor = ?";
  const binds: (string | number)[] = [actor];

  if (action) {
    sql += " AND action = ?";
    binds.push(action);
  }

  // Get total count
  const countSql = sql.replace("SELECT action, path, ip, ts", "SELECT COUNT(*) as total");
  const countRow = await c.env.DB.prepare(countSql)
    .bind(...binds)
    .first<{ total: number }>();

  sql += " ORDER BY ts DESC LIMIT ? OFFSET ?";
  binds.push(limit, offset);

  const { results } = await c.env.DB.prepare(sql).bind(...binds).all();

  return c.json({
    entries: (results || []) as any[],
    total: countRow?.total || 0,
  });
}

// ── GET /dashboard/shares ────────────────────────────────────────────
async function sharesHandler(c: C) {
  const actor = c.get("actor");
  const now = Date.now();

  const { results } = await c.env.DB.prepare(
    "SELECT token, path, expires_at, created_at, COALESCE(views, 0) as views FROM share_links WHERE actor = ? AND expires_at > ? ORDER BY created_at DESC",
  )
    .bind(actor, now)
    .all();

  return c.json({ shares: (results || []) as any[] });
}

// ── DELETE /dashboard/shares/:token ──────────────────────────────────
async function deleteShareHandler(c: C) {
  const actor = c.get("actor");
  const token = c.req.param("token");

  const row = await c.env.DB.prepare(
    "SELECT 1 FROM share_links WHERE token = ? AND actor = ?",
  )
    .bind(token, actor)
    .first();

  if (!row) return err(c, "not_found", "Share not found");

  await c.env.DB.prepare("DELETE FROM share_links WHERE token = ?")
    .bind(token)
    .run();

  return c.json({ deleted: true });
}

// ── GET /dashboard/account ───────────────────────────────────────────
async function accountHandler(c: C) {
  const actor = c.get("actor");

  const row = await c.env.DB.prepare(
    "SELECT actor, email, username, display_name, public_key, created_at FROM actors WHERE actor = ?",
  )
    .bind(actor)
    .first<{ actor: string; email: string | null; username: string | null; display_name: string | null; public_key: string | null; created_at: number }>();

  const sessions = await c.env.DB.prepare(
    "SELECT COUNT(*) as count FROM sessions WHERE actor = ? AND expires_at > ?",
  )
    .bind(actor, Date.now())
    .first<{ count: number }>();

  return c.json({
    actor: row?.actor || actor,
    email: row?.email || null,
    username: row?.username || null,
    display_name: row?.display_name || null,
    needs_onboarding: row?.public_key === "placeholder-email-user",
    created_at: row?.created_at || null,
    active_sessions: sessions?.count || 0,
  });
}

// ── PATCH /dashboard/profile ────────────────────────────────────────
const RESERVED_USERNAMES = new Set([
  "admin", "system", "storage", "api", "auth", "dashboard", "root",
  "null", "undefined", "test", "demo", "help", "support", "billing",
  "settings", "config", "files", "upload", "download", "share",
  "public", "private", "user", "users", "account", "login", "logout",
  "register", "signup", "signin", "oauth", "callback", "webhook",
  "status", "health", "ping", "www", "mail", "email", "noreply",
]);

const USERNAME_RE = /^[a-z0-9][a-z0-9-]*[a-z0-9]$/;

async function profileHandler(c: C) {
  const actor = c.get("actor");

  // Parse body
  let body: { username?: string; display_name?: string };
  try {
    body = await c.req.json();
  } catch {
    return err(c, "invalid_request", "Invalid JSON body");
  }

  const username = (body.username || "").trim().toLowerCase();
  const displayName = (body.display_name || "").trim();

  // Validate username
  if (!username) {
    return err(c, "invalid_request", "Username is required");
  }
  if (username.length < 3 || username.length > 20) {
    return err(c, "invalid_request", "Username must be 3-20 characters");
  }
  if (username.length === 1 || username.length === 2) {
    if (!/^[a-z0-9]+$/.test(username)) {
      return err(c, "invalid_request", "Username can only contain lowercase letters, numbers, and hyphens");
    }
  } else if (!USERNAME_RE.test(username)) {
    return err(c, "invalid_request", "Username can only contain lowercase letters, numbers, and hyphens, and cannot start or end with a hyphen");
  }
  if (/--/.test(username)) {
    return err(c, "invalid_request", "Username cannot contain consecutive hyphens");
  }
  if (RESERVED_USERNAMES.has(username)) {
    return err(c, "invalid_request", "That username is reserved");
  }

  // Validate display name
  if (!displayName) {
    return err(c, "invalid_request", "Display name is required");
  }
  if (displayName.length > 50) {
    return err(c, "invalid_request", "Display name must be 50 characters or less");
  }

  // Check current actor — must still need onboarding
  const row = await c.env.DB.prepare(
    "SELECT public_key, username FROM actors WHERE actor = ?",
  ).bind(actor).first<{ public_key: string | null; username: string | null }>();

  if (!row) {
    return err(c, "not_found", "Actor not found");
  }
  if (row.public_key !== "placeholder-email-user") {
    return err(c, "forbidden", "Username has already been set");
  }

  // Check username uniqueness
  const existing = await c.env.DB.prepare(
    "SELECT 1 FROM actors WHERE username = ?",
  ).bind(username).first();

  if (existing) {
    return err(c, "conflict", "That username is already taken");
  }

  // Also check no actor has this as their actor name
  const actorConflict = await c.env.DB.prepare(
    "SELECT 1 FROM actors WHERE actor = ?",
  ).bind(`u/${username}`).first();

  if (actorConflict && `u/${username}` !== actor) {
    return err(c, "conflict", "That username is already taken");
  }

  // Update actor record
  const result = await c.env.DB.prepare(
    "UPDATE actors SET username = ?, display_name = ?, public_key = 'onboarded' WHERE actor = ? AND public_key = 'placeholder-email-user'",
  ).bind(username, displayName, actor).run();

  if (!result.meta.changes) {
    return err(c, "conflict", "Could not update profile — please try again");
  }

  // Audit log
  try {
    await c.env.DB.prepare(
      "INSERT INTO audit (actor, action, path, ip, ts) VALUES (?, 'onboard', ?, ?, ?)",
    ).bind(actor, username, c.req.header("cf-connecting-ip") || "unknown", Date.now()).run();
  } catch { /* non-critical */ }

  return c.json({
    actor,
    username,
    display_name: displayName,
  });
}

// ── Registration ─────────────────────────────────────────────────────
export function register(app: App) {
  app.use("/dashboard/*", auth);
  app.get("/dashboard/audit", auditHandler);
  app.get("/dashboard/shares", sharesHandler);
  app.delete("/dashboard/shares/:token", deleteShareHandler);
  app.get("/dashboard/account", accountHandler);
  app.patch("/dashboard/profile", profileHandler);
}
