import type { Context, Next } from "hono";
import type { Env, Variables } from "../types";
import { err } from "../lib/error";

type C = Context<{ Bindings: Env; Variables: Variables }>;

export async function auth(c: C, next: Next) {
  let token: string | undefined;

  const h = c.req.header("Authorization");
  if (h?.startsWith("Bearer ")) token = h.slice(7).trim();

  if (!token) {
    const cookie = c.req.header("Cookie");
    if (cookie) {
      const m = cookie.match(/(?:^|;\s*)session=([^;]+)/);
      if (m) token = m[1];
    }
  }

  if (!token) return err(c, "unauthorized", "Missing authorization");

  // Try session token
  const session = await c.env.DB.prepare(
    "SELECT actor, expires_at FROM sessions WHERE token = ?",
  )
    .bind(token)
    .first<{ actor: string; expires_at: number }>();

  if (session) {
    if (Date.now() > session.expires_at) {
      c.executionCtx.waitUntil(
        c.env.DB.prepare("DELETE FROM sessions WHERE token = ?").bind(token).run(),
      );
      return err(c, "unauthorized", "Token expired");
    }
    c.set("actor", session.actor);
    c.set("prefix", "");
    return next();
  }

  // Try API key (hash-based lookup)
  const hash = await sha256(token);
  const key = await c.env.DB.prepare(
    "SELECT actor, prefix, expires_at FROM api_keys WHERE token_hash = ?",
  )
    .bind(hash)
    .first<{ actor: string; prefix: string; expires_at: number | null }>();

  if (key) {
    if (key.expires_at && Date.now() > key.expires_at) {
      return err(c, "unauthorized", "API key expired");
    }
    c.set("actor", key.actor);
    c.set("prefix", key.prefix || "");
    return next();
  }

  return err(c, "unauthorized", "Invalid token");
}

export async function sha256(data: string): Promise<string> {
  const buf = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(data));
  return Array.from(new Uint8Array(buf), (b) => b.toString(16).padStart(2, "0")).join("");
}
