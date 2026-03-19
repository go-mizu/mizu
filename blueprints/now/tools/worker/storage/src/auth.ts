import type { Context, Next } from "hono";
import type { Env, Variables, SessionRow, ApiKeyRow } from "./types";
import { challengeId, nonce as generateNonce, sessionToken } from "./id";
import { base64urlDecode, importEd25519PublicKey, verifyEd25519 } from "./crypto";
import { errorResponse } from "./error";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const CHALLENGE_TTL_MS = 5 * 60 * 1000;
const SESSION_TTL_MS = 2 * 60 * 60 * 1000;

export async function createChallenge(c: AppContext) {
  let body: { actor?: string };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.actor || typeof body.actor !== "string") {
    return errorResponse(c, "invalid_request", "actor is required");
  }

  const actor = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
    .bind(body.actor)
    .first();
  if (!actor) {
    return errorResponse(c, "not_found", "Actor not found");
  }

  const id = challengeId();
  const nonceVal = generateNonce();
  const now = Date.now();
  const expiresAt = now + CHALLENGE_TTL_MS;

  await c.env.DB.prepare(
    "INSERT INTO challenges (id, actor, nonce, expires_at) VALUES (?, ?, ?, ?)",
  )
    .bind(id, body.actor, nonceVal, expiresAt)
    .run();

  return c.json({
    challenge_id: id,
    nonce: nonceVal,
    expires_at: new Date(expiresAt).toISOString(),
  });
}

export async function verifyChallenge(c: AppContext) {
  let body: { challenge_id?: string; actor?: string; signature?: string };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.challenge_id || !body.actor || !body.signature) {
    return errorResponse(c, "invalid_request", "challenge_id, actor, and signature are required");
  }

  const challenge = await c.env.DB.prepare(
    "SELECT * FROM challenges WHERE id = ? AND actor = ?",
  )
    .bind(body.challenge_id, body.actor)
    .first<{ id: string; actor: string; nonce: string; expires_at: number }>();

  if (!challenge) {
    return errorResponse(c, "not_found", "Challenge not found");
  }

  if (Date.now() > challenge.expires_at) {
    await c.env.DB.prepare("DELETE FROM challenges WHERE id = ?").bind(body.challenge_id).run();
    return errorResponse(c, "unauthorized", "Challenge expired");
  }

  const actorRow = await c.env.DB.prepare("SELECT public_key FROM actors WHERE actor = ?")
    .bind(body.actor)
    .first<{ public_key: string }>();
  if (!actorRow) {
    return errorResponse(c, "not_found", "Actor not found");
  }

  let publicKey: CryptoKey;
  try {
    publicKey = await importEd25519PublicKey(actorRow.public_key);
  } catch {
    return errorResponse(c, "unauthorized", "Invalid public key");
  }

  let sigBytes: Uint8Array;
  try {
    sigBytes = base64urlDecode(body.signature);
  } catch {
    return errorResponse(c, "invalid_request", "Invalid signature encoding");
  }

  const valid = await verifyEd25519(publicKey, sigBytes, challenge.nonce);
  if (!valid) {
    return errorResponse(c, "unauthorized", "Invalid signature");
  }

  await c.env.DB.prepare("DELETE FROM challenges WHERE id = ?").bind(body.challenge_id).run();

  const token = sessionToken();
  const now = Date.now();
  const expiresAt = now + SESSION_TTL_MS;

  await c.env.DB.prepare(
    "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
  )
    .bind(token, body.actor, expiresAt)
    .run();

  return c.json({
    access_token: token,
    expires_at: new Date(expiresAt).toISOString(),
  });
}

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
    c.set("actor", session.actor);
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
    c.set("actor", apiKey.actor);
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

async function sha256(data: string): Promise<string> {
  const encoded = new TextEncoder().encode(data);
  const hash = await crypto.subtle.digest("SHA-256", encoded);
  return Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0")).join("");
}
