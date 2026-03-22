import type { Context } from "hono";
import type { Env, Variables } from "./types";

/**
 * Optionally reads session from cookie. Returns actor string or null.
 * Used by page handlers to detect signed-in users (not for auth enforcement).
 */
export async function getSessionActor(c: Context<{ Bindings: Env; Variables: Variables }>): Promise<string | null> {
  const cookie = c.req.header("Cookie");
  if (!cookie) return null;

  const match = cookie.match(/(?:^|;\s*)session=([^;]+)/);
  if (!match) return null;

  const token = match[1];
  const session = await c.env.DB.prepare(
    "SELECT actor, expires_at FROM sessions WHERE token = ?"
  ).bind(token).first<{ actor: string; expires_at: number }>();

  if (!session || Date.now() > session.expires_at) return null;

  return session.actor;
}
