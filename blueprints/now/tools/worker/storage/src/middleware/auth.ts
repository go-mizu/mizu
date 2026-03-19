import type { Context, Next } from "hono";
import type { Env, Variables, SessionRow, ApiKeyRow } from "../types";
import { errorResponse } from "../lib/error";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

/**
 * Bearer auth middleware.
 *
 * Resolves token from:
 *   1. Authorization: Bearer <token> header
 *   2. session=<token> cookie
 *
 * Checks (in order):
 *   1. Session tokens → actor with '*' scopes
 *   2. Scoped API keys → actor with limited scopes + optional path_prefix
 */
export async function bearerAuth(c: AppContext, next: Next) {
  let token: string | undefined;

  const authHeader = c.req.header("Authorization");
  if (authHeader && authHeader.startsWith("Bearer ")) {
    token = authHeader.slice("Bearer ".length).trim();
  }

  if (!token) {
    const cookie = c.req.header("Cookie");
    if (cookie) {
      const match = cookie.match(/(?:^|;\s*)session=([^;]+)/);
      if (match) token = match[1];
    }
  }

  if (!token) {
    return errorResponse(c, "unauthorized", "Missing authorization");
  }

  // Try session token first
  const session = await c.env.DB.prepare(
    "SELECT actor, expires_at FROM sessions WHERE token = ?",
  )
    .bind(token)
    .first<SessionRow>();

  if (session) {
    if (Date.now() > session.expires_at) {
      await c.env.DB.prepare("DELETE FROM sessions WHERE token = ?").bind(token).run();
      return errorResponse(c, "unauthorized", "Token expired");
    }
    console.log(`[auth] session → actor=${session.actor} token=${token.slice(0, 8)}...`);
    c.set("actor", session.actor);
    c.set("authType", "session");
    c.set("scopes", "*");
    c.set("pathPrefix", "");
    return next();
  }

  // Try API key (hash the token, look up)
  const hash = await sha256(token);
  const apiKey = await c.env.DB.prepare(
    "SELECT * FROM api_keys WHERE token_hash = ?",
  )
    .bind(hash)
    .first<ApiKeyRow>();

  if (apiKey) {
    if (apiKey.expires_at && Date.now() > apiKey.expires_at) {
      return errorResponse(c, "unauthorized", "API key expired");
    }
    console.log(`[auth] apikey → actor=${apiKey.actor} id=${apiKey.id} name=${apiKey.name} hash=${hash.slice(0, 8)}...`);
    c.set("actor", apiKey.actor);
    c.set("authType", "apikey");
    c.set("scopes", apiKey.scopes);
    c.set("pathPrefix", apiKey.path_prefix || "");

    // Update last_used_at asynchronously
    c.executionCtx.waitUntil(
      c.env.DB.prepare("UPDATE api_keys SET last_used_at = ? WHERE id = ?")
        .bind(Date.now(), apiKey.id)
        .run(),
    );
    return next();
  }

  return errorResponse(c, "unauthorized", "Invalid token");
}

export async function sha256(data: string): Promise<string> {
  const encoded = new TextEncoder().encode(data);
  const hash = await crypto.subtle.digest("SHA-256", encoded);
  return Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0")).join("");
}
