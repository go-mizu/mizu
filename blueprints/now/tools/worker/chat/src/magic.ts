import type { Context } from "hono";
import type { Env, Variables, MagicLinkRequest, MagicTokenRow } from "./types";
import { magicToken, placeholderKey, sessionToken } from "./id";
import { errorResponse } from "./error";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const MAGIC_TTL_MS = 15 * 60 * 1000; // 15 minutes
const SESSION_TTL_MS = 2 * 60 * 60 * 1000; // 2 hours

// --- POST /auth/magic-link ---
export async function requestMagicLink(c: AppContext) {
  let body: MagicLinkRequest;
  try {
    body = await c.req.json<MagicLinkRequest>();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.email || typeof body.email !== "string") {
    return errorResponse(c, "invalid_request", "email is required");
  }

  const email = body.email.trim().toLowerCase();
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    return errorResponse(c, "invalid_request", "Invalid email format");
  }

  // Check if actor with this email exists
  let actor: string;
  const existing = await c.env.DB.prepare(
    "SELECT actor FROM actors WHERE email = ?"
  ).bind(email).first<{ actor: string }>();

  if (existing) {
    actor = existing.actor;
  } else {
    // Auto-generate name from email prefix
    const raw = email.split("@")[0].replace(/[^a-z0-9-]/gi, "").toLowerCase();
    const prefix = raw.slice(0, 20) || "user";
    let name = prefix;
    let attempt = 0;

    while (true) {
      const candidate = attempt === 0 ? `u/${name}` : `u/${name}-${attempt}`;
      const exists = await c.env.DB.prepare(
        "SELECT 1 FROM actors WHERE actor = ?"
      ).bind(candidate).first();

      if (!exists) {
        actor = candidate;
        break;
      }
      attempt++;
      if (attempt > 99) {
        return errorResponse(c, "conflict", "Could not generate unique name");
      }
    }

    // Create actor with placeholder key (email users don't use challenge-response)
    const now = Date.now();
    await c.env.DB.prepare(
      "INSERT INTO actors (actor, type, public_key, email, created_at) VALUES (?, 'human', ?, ?, ?)"
    ).bind(actor, placeholderKey(), email, now).run();
  }

  // Generate magic token
  const token = magicToken();
  const expiresAt = Date.now() + MAGIC_TTL_MS;

  await c.env.DB.prepare(
    "INSERT INTO magic_tokens (token, email, actor, expires_at) VALUES (?, ?, ?, ?)"
  ).bind(token, email, actor, expiresAt).run();

  // Build magic link
  const origin = new URL(c.req.url).origin;
  const link = `${origin}/auth/magic/${token}`;

  // In production: send email with this link
  // For now: return the link directly
  return c.json({
    message: "Magic link created",
    magic_link: link,
    actor: actor,
  });
}

// --- GET /auth/magic/:token ---
export async function verifyMagicLink(c: AppContext) {
  const token = c.req.param("token");

  const row = await c.env.DB.prepare(
    "SELECT token, email, actor, expires_at FROM magic_tokens WHERE token = ?"
  ).bind(token).first<MagicTokenRow>();

  if (!row || !row.actor) {
    return c.html(errorPage("Link not found", "This magic link is invalid or has already been used."), 404);
  }

  if (Date.now() > row.expires_at) {
    await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();
    return c.html(errorPage("Link expired", "This magic link has expired. Please request a new one."), 410);
  }

  // Delete magic token (single use)
  await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();

  // Create session
  const sessTok = sessionToken();
  const expiresAt = Date.now() + SESSION_TTL_MS;

  await c.env.DB.prepare(
    "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)"
  ).bind(sessTok, row.actor, expiresAt).run();

  // Set cookie and redirect to home
  const isSecure = new URL(c.req.url).protocol === "https:";
  const cookie = `session=${sessTok}; HttpOnly; SameSite=Lax; Path=/; Max-Age=7200${isSecure ? "; Secure" : ""}`;

  return new Response(null, {
    status: 302,
    headers: {
      Location: "/",
      "Set-Cookie": cookie,
    },
  });
}

// --- POST /auth/logout ---
export async function logout(c: AppContext) {
  const cookie = c.req.header("Cookie");
  if (cookie) {
    const match = cookie.match(/(?:^|;\s*)session=([^;]+)/);
    if (match) {
      await c.env.DB.prepare("DELETE FROM sessions WHERE token = ?").bind(match[1]).run();
    }
  }

  const isSecure = new URL(c.req.url).protocol === "https:";
  const clearCookie = `session=; HttpOnly; SameSite=Lax; Path=/; Max-Age=0${isSecure ? "; Secure" : ""}`;

  return new Response(null, {
    status: 302,
    headers: {
      Location: "/",
      "Set-Cookie": clearCookie,
    },
  });
}

function errorPage(title: string, message: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${title} — chat.now</title>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500&family=DM+Sans:wght@400;600&display=swap" rel="stylesheet">
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{--bg:#FAFAF9;--text:#111;--text-2:#555;--text-3:#999;--border:#DDD}
html.dark{--bg:#0C0C0C;--text:#E5E5E5;--text-2:#999;--text-3:#555;--border:#2A2A2A}
body{font-family:'DM Sans',system-ui,sans-serif;color:var(--text);background:var(--bg);
display:flex;align-items:center;justify-content:center;min-height:100vh;padding:20px}
.box{max-width:400px;text-align:center}
h1{font-size:24px;font-weight:600;margin-bottom:12px}
p{font-size:14px;color:var(--text-2);line-height:1.7;margin-bottom:24px}
a{font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--text);
text-decoration:underline;text-underline-offset:3px}
</style>
</head>
<body>
<div class="box">
  <h1>${title}</h1>
  <p>${message}</p>
  <a href="/">Back to chat.now</a>
</div>
<script>
if(localStorage.getItem('theme')==='dark'||(!localStorage.getItem('theme')&&window.matchMedia('(prefers-color-scheme:dark)').matches))
  document.documentElement.classList.add('dark');
</script>
</body>
</html>`;
}
