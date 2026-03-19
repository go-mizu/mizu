import type { Context } from "hono";
import type { Env, Variables } from "../types";
import { challengeId, nonce as generateNonce, sessionToken, magicToken } from "../lib/id";
import { base64urlDecode, importEd25519PublicKey, verifyEd25519 } from "../lib/crypto";
import { errorResponse } from "../lib/error";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const CHALLENGE_TTL_MS = 5 * 60 * 1000;
const SESSION_TTL_MS = 2 * 60 * 60 * 1000;
const MAGIC_TTL_MS = 15 * 60 * 1000;
const ACTOR_RE = /^[ua]\/[\w.@-]{1,64}$/;

// POST /auth/challenge
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

// POST /auth/verify
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

// POST /auth/magic-link
export async function requestMagicLink(c: AppContext) {
  let body: { email?: string };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  const email = body.email?.trim().toLowerCase();
  if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    return errorResponse(c, "invalid_request", "Valid email is required");
  }

  let actorName: string;
  const existing = await c.env.DB.prepare("SELECT actor FROM actors WHERE email = ?")
    .bind(email)
    .first<{ actor: string }>();

  if (existing) {
    actorName = existing.actor;
  } else {
    const local = email.split("@")[0].replace(/[^a-zA-Z0-9._-]/g, "").slice(0, 32);
    actorName = `u/${local}`;

    const collision = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
      .bind(actorName)
      .first();
    if (collision) {
      actorName = `u/${local}.${Date.now().toString(36).slice(-4)}`;
    }

    await c.env.DB.prepare(
      "INSERT INTO actors (actor, type, email, bio, created_at) VALUES (?, 'human', ?, '', ?)",
    )
      .bind(actorName, email, Date.now())
      .run();
  }

  const token = magicToken();
  const expiresAt = Date.now() + MAGIC_TTL_MS;
  await c.env.DB.prepare(
    "INSERT INTO magic_tokens (token, email, actor, expires_at) VALUES (?, ?, ?, ?)",
  )
    .bind(token, email, actorName, expiresAt)
    .run();

  const url = new URL(c.req.url);
  const magicLink = `${url.origin}/auth/magic/${token}`;

  if (c.env.RESEND_API_KEY) {
    fetch("https://api.resend.com/emails", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${c.env.RESEND_API_KEY}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        from: "storage.now <noreply@liteio.com>",
        to: email,
        subject: "Sign in to storage.now",
        text: `Click to sign in:\n\n${magicLink}\n\nExpires in 15 minutes.`,
      }),
    }).catch(() => {});
  }

  return c.json({ ok: true, magic_link: magicLink });
}

// GET /auth/magic/:token
export async function verifyMagicLink(c: AppContext) {
  const token = c.req.param("token");
  const row = await c.env.DB.prepare(
    "SELECT email, actor, expires_at FROM magic_tokens WHERE token = ?",
  )
    .bind(token)
    .first<{ email: string; actor: string; expires_at: number }>();

  if (!row) {
    return c.html("<h1>Invalid or expired link</h1>", 400);
  }

  if (Date.now() > row.expires_at) {
    await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();
    return c.html("<h1>Link expired</h1>", 400);
  }

  await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();

  const sessToken = sessionToken();
  const expiresAt = Date.now() + SESSION_TTL_MS;
  await c.env.DB.prepare(
    "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
  )
    .bind(sessToken, row.actor, expiresAt)
    .run();

  return new Response(null, {
    status: 302,
    headers: {
      Location: "/browse",
      "Set-Cookie": `session=${sessToken}; Path=/; HttpOnly; SameSite=Lax; Max-Age=7200`,
    },
  });
}

// POST /auth/logout  GET /auth/logout
export async function logout(c: AppContext) {
  const cookie = c.req.header("Cookie");
  if (cookie) {
    const match = cookie.match(/(?:^|;\s*)session=([^;]+)/);
    if (match) {
      await c.env.DB.prepare("DELETE FROM sessions WHERE token = ?").bind(match[1]).run();
    }
  }

  return new Response(null, {
    status: 302,
    headers: {
      Location: "/",
      "Set-Cookie": "session=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0",
    },
  });
}

// POST /actors
export async function registerActor(c: AppContext) {
  let body: { actor?: string; type?: string; public_key?: string; email?: string; bio?: string };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.actor || !ACTOR_RE.test(body.actor)) {
    return errorResponse(c, "invalid_request", "Invalid actor format (u/name or a/name)");
  }

  const expectedType = body.actor.startsWith("u/") ? "human" : "agent";
  if (body.type && body.type !== expectedType) {
    return errorResponse(c, "invalid_request", `Actor prefix doesn't match type`);
  }

  if (expectedType === "agent" && !body.public_key) {
    return errorResponse(c, "invalid_request", "Agents require a public_key");
  }

  const existing = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
    .bind(body.actor)
    .first();
  if (existing) {
    return c.json({ actor: body.actor, created: false });
  }

  const now = Date.now();
  await c.env.DB.prepare(
    "INSERT INTO actors (actor, type, public_key, email, bio, created_at) VALUES (?, ?, ?, ?, ?, ?)",
  )
    .bind(body.actor, expectedType, body.public_key || null, body.email || null, body.bio || "", now)
    .run();

  return c.json({ actor: body.actor, created: true }, 201);
}
