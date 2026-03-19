import type { Context } from "hono";
import type { Env, Variables } from "../types";
import { errorResponse } from "./error";
import { requireScope } from "../middleware/authorize";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

/**
 * Log an audit event asynchronously (non-blocking via waitUntil).
 */
export function audit(
  c: AppContext,
  action: string,
  resource?: string,
  detail?: Record<string, unknown>,
) {
  const actor = c.get("actor") || null;
  const ip = c.req.header("CF-Connecting-IP") || c.req.header("X-Forwarded-For") || "";
  const ts = Date.now();
  const detailJson = detail ? JSON.stringify(detail) : null;

  c.executionCtx.waitUntil(
    (async () => {
      try {
        await c.env.DB.prepare(
          "INSERT INTO audit_log (actor, action, resource, detail, ip, ts) VALUES (?, ?, ?, ?, ?, ?)",
        )
          .bind(actor, action, resource || null, detailJson, ip, ts)
          .run();

        // Probabilistic cleanup: ~1% chance per write, remove entries older than 90 days
        if (Math.random() < 0.01) {
          const cutoff = ts - 90 * 24 * 60 * 60 * 1000;
          await c.env.DB.prepare("DELETE FROM audit_log WHERE ts < ?").bind(cutoff).run();
        }
      } catch {
        // Audit logging should never break the request
      }
    })(),
  );
}

/**
 * Log an audit event for unauthenticated requests (no actor context).
 * Used for auth endpoints where c.get("actor") isn't set.
 */
export function auditAnon(
  c: AppContext,
  action: string,
  actor: string | null,
  resource?: string,
  detail?: Record<string, unknown>,
) {
  const ip = c.req.header("CF-Connecting-IP") || c.req.header("X-Forwarded-For") || "";
  const ts = Date.now();
  const detailJson = detail ? JSON.stringify(detail) : null;

  c.executionCtx.waitUntil(
    (async () => {
      try {
        await c.env.DB.prepare(
          "INSERT INTO audit_log (actor, action, resource, detail, ip, ts) VALUES (?, ?, ?, ?, ?, ?)",
        )
          .bind(actor, action, resource || null, detailJson, ip, ts)
          .run();
      } catch {
        // Audit logging should never break the request
      }
    })(),
  );
}

// GET /audit?action=&limit=&before=
export async function getAuditLog(c: AppContext) {
  const scopeErr = requireScope(c, "*");
  if (scopeErr) return scopeErr;

  const actor = c.get("actor");
  const action = c.req.query("action");
  const limit = Math.min(parseInt(c.req.query("limit") || "50", 10), 200);
  const before = c.req.query("before") ? parseInt(c.req.query("before")!, 10) : null;

  let sql = "SELECT id, actor, action, resource, detail, ip, ts FROM audit_log WHERE actor = ?";
  const binds: any[] = [actor];

  if (action) {
    sql += " AND action = ?";
    binds.push(action);
  }
  if (before) {
    sql += " AND ts < ?";
    binds.push(before);
  }

  sql += " ORDER BY ts DESC LIMIT ?";
  binds.push(limit);

  const { results } = await c.env.DB.prepare(sql).bind(...binds).all();

  return c.json({
    items: (results || []).map((row: any) => ({
      ...row,
      detail: row.detail ? JSON.parse(row.detail) : null,
    })),
  });
}
