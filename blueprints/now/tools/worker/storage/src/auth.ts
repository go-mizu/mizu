import type { Context, Next } from "hono";
import type { Env, Variables, ChallengeRequest, VerifyRequest, SessionRow } from "./types";
import { challengeId, nonce as generateNonce, sessionToken } from "./id";
import { base64urlDecode, importEd25519PublicKey, verifyEd25519 } from "./crypto";
import { errorResponse } from "./error";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const CHALLENGE_TTL_MS = 5 * 60 * 1000;
const SESSION_TTL_MS = 2 * 60 * 60 * 1000;

export async function createChallenge(c: AppContext) {
  let body: ChallengeRequest;
  try {
    body = await c.req.json<ChallengeRequest>();
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
  let body: VerifyRequest;
  try {
    body = await c.req.json<VerifyRequest>();
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

  const session = await c.env.DB.prepare(
    "SELECT actor, expires_at FROM sessions WHERE token = ?",
  )
    .bind(token)
    .first<SessionRow>();

  if (!session) {
    return errorResponse(c, "unauthorized", "Invalid token");
  }

  if (Date.now() > session.expires_at) {
    await c.env.DB.prepare("DELETE FROM sessions WHERE token = ?").bind(token).run();
    return errorResponse(c, "unauthorized", "Token expired");
  }

  c.set("actor", session.actor);
  await next();
}
