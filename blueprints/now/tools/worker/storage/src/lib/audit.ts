import type { Context } from "hono";
import type { Env, Variables } from "../types";

type C = Context<{ Bindings: Env; Variables: Variables }>;

export function audit(c: C, action: string, path?: string) {
  const actor = c.get("actor") || null;
  const ip = c.req.header("CF-Connecting-IP") || c.req.header("X-Forwarded-For") || "";
  const ts = Date.now();

  c.executionCtx.waitUntil(
    c.env.DB.prepare(
      "INSERT INTO audit_log (actor, action, path, ip, ts) VALUES (?, ?, ?, ?, ?)",
    )
      .bind(actor, action, path || null, ip, ts)
      .run()
      .then(() => {
        // Probabilistic cleanup: ~1% chance, remove entries > 90 days old
        if (Math.random() < 0.01) {
          return c.env.DB.prepare("DELETE FROM audit_log WHERE ts < ?")
            .bind(ts - 90 * 86400000)
            .run();
        }
      })
      .catch(() => {}),
  );
}
