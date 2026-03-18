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

  // Send email via Resend if API key is configured
  if (c.env.RESEND_API_KEY) {
    const html = magicLinkEmail(link, actor.slice(2));
    let emailSent = false;
    try {
      const res = await fetch("https://api.resend.com/emails", {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${c.env.RESEND_API_KEY}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          from: "chat.now <onboarding@resend.dev>",
          to: [email],
          subject: "Sign in to chat.now",
          html,
        }),
      });
      if (res.ok) {
        emailSent = true;
      } else {
        const err = await res.text();
        console.error("[magic] Resend error:", res.status, err);
      }
    } catch (e) {
      console.error("[magic] Failed to send email:", e);
    }

    if (emailSent) {
      return c.json({ message: "Check your email for a sign-in link.", actor });
    }
    // Fall through to return magic_link if email failed (e.g. domain restrictions)
  }

  // Dev fallback: return the link directly
  return c.json({
    message: "Magic link created. Click the link to sign in.",
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

function magicLinkEmail(link: string, username: string): string {
  return `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#FAFAF9;font-family:'Inter',system-ui,sans-serif">
<table width="100%" cellpadding="0" cellspacing="0" style="background:#FAFAF9;padding:40px 20px">
  <tr><td align="center">
    <table width="520" cellpadding="0" cellspacing="0" style="background:#fff;border:1px solid #E5E5E3;border-radius:8px;padding:48px 40px;max-width:520px">
      <tr><td>
        <div style="font-family:'JetBrains Mono',monospace;font-size:18px;font-weight:700;letter-spacing:-0.5px;margin-bottom:8px">chat.now</div>
        <div style="height:1px;background:#E5E5E3;margin:24px 0"></div>
        <h1 style="font-size:22px;font-weight:600;color:#111;margin:0 0 12px">Sign in to chat.now</h1>
        <p style="font-size:15px;color:#555;line-height:1.6;margin:0 0 32px">
          Hey <strong>${username}</strong>, click the button below to sign in. This link expires in 15 minutes and can only be used once.
        </p>
        <a href="${link}" style="display:inline-block;background:#111;color:#fff;font-size:14px;font-weight:600;padding:14px 28px;border-radius:6px;text-decoration:none;letter-spacing:0.2px">Sign in to chat.now</a>
        <div style="margin-top:32px;padding-top:24px;border-top:1px solid #E5E5E3">
          <p style="font-size:12px;color:#999;margin:0">If you didn't request this, you can ignore this email. Your account is safe.</p>
          <p style="font-size:12px;color:#999;margin:8px 0 0">Or copy this URL: <a href="${link}" style="color:#555">${link}</a></p>
        </div>
      </td></tr>
    </table>
  </td></tr>
</table>
</body>
</html>`;
}

function errorPage(title: string, message: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${title} — chat.now</title>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500&family=Inter:wght@400;600&display=swap" rel="stylesheet">
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{--bg:#FAFAF9;--text:#111;--text-2:#555;--text-3:#999;--border:#DDD}
html.dark{--bg:#0C0C0C;--text:#E5E5E5;--text-2:#999;--text-3:#555;--border:#2A2A2A}
body{font-family:'Inter',system-ui,sans-serif;color:var(--text);background:var(--bg);
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
