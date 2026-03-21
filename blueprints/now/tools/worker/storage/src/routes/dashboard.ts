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
    "SELECT actor, email, created_at FROM actors WHERE actor = ?",
  )
    .bind(actor)
    .first<{ actor: string; email: string | null; created_at: number }>();

  const sessions = await c.env.DB.prepare(
    "SELECT COUNT(*) as count FROM sessions WHERE actor = ? AND expires_at > ?",
  )
    .bind(actor, Date.now())
    .first<{ count: number }>();

  return c.json({
    actor: row?.actor || actor,
    email: row?.email || null,
    created_at: row?.created_at || null,
    active_sessions: sessions?.count || 0,
  });
}

// ── Registration ─────────────────────────────────────────────────────
export function register(app: App) {
  app.use("/dashboard/*", auth);
  app.get("/dashboard/audit", auditHandler);
  app.get("/dashboard/shares", sharesHandler);
  app.delete("/dashboard/shares/:token", deleteShareHandler);
  app.get("/dashboard/account", accountHandler);
}
