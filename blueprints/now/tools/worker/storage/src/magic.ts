import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { magicToken, sessionToken } from "./id";
import { errorResponse } from "./error";

const MAGIC_TTL_MS = 15 * 60 * 1000;
const SESSION_TTL_MS = 2 * 60 * 60 * 1000;

export async function requestMagicLink(
  c: Context<{ Bindings: Env; Variables: Variables }>,
) {
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

  // Find or create actor from email
  let actorName: string;
  const existing = await c.env.DB.prepare("SELECT actor FROM actors WHERE email = ?")
    .bind(email)
    .first<{ actor: string }>();

  if (existing) {
    actorName = existing.actor;
  } else {
    const local = email.split("@")[0].replace(/[^a-zA-Z0-9._-]/g, "").slice(0, 32);
    actorName = `u/${local}`;

    // Handle collision
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

  // If RESEND_API_KEY is set, send email; otherwise return link directly
  if (c.env.RESEND_API_KEY) {
    await fetch("https://api.resend.com/emails", {
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
    });
    return c.json({ ok: true, message: "Check your inbox" });
  }

  return c.json({ ok: true, magic_link: magicLink });
}

export async function verifyMagicLink(
  c: Context<{ Bindings: Env; Variables: Variables }>,
) {
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

export async function logout(c: Context<{ Bindings: Env; Variables: Variables }>) {
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
