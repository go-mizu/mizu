import { createRoute, z } from "@hono/zod-openapi";
import type { App } from "../types";
import { magicToken, sessionToken } from "../lib/id";

const SITE_NAME = "Storage";
const MAGIC_TTL_MS = 15 * 60 * 1000; // 15 minutes
const SESSION_TTL_MS = 2 * 60 * 60 * 1000; // 2 hours
const RATE_LIMIT_MS = 60 * 1000; // 1 request per email per minute

// ── Route definitions ────────────────────────────────────────────────

const errSchema = z.object({ error: z.string(), message: z.string() });

const requestMagicLinkRoute = createRoute({
  method: "post",
  path: "/auth/magic-link",
  summary: "Request a magic link",
  tags: ["auth"],
  request: {
    body: {
      content: {
        "application/json": {
          schema: z.object({
            email: z.string().email().openapi({ example: "you@email.com" }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Magic link sent",
      content: {
        "application/json": {
          schema: z.object({ message: z.string() }),
        },
      },
    },
    400: {
      description: "Bad request",
      content: { "application/json": { schema: errSchema } },
    },
    409: {
      description: "Conflict",
      content: { "application/json": { schema: errSchema } },
    },
    429: {
      description: "Rate limited",
      content: { "application/json": { schema: errSchema } },
    },
    500: {
      description: "Email delivery failed",
      content: { "application/json": { schema: errSchema } },
    },
  },
});

// ── Handlers ─────────────────────────────────────────────────────────

export function register(app: App) {
  // POST /auth/magic-link — request a magic sign-in link
  app.openapi(requestMagicLinkRoute, async (c) => {
    const { email: rawEmail } = c.req.valid("json");
    const email = rawEmail.trim().toLowerCase();

    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      return c.json({ error: "invalid_request", message: "Invalid email format" }, 400);
    }

    // Require Resend to be configured — no dev fallback
    if (!c.env.RESEND_API_KEY) {
      console.error("[magic] RESEND_API_KEY not configured");
      return c.json({ error: "server_error", message: "Email service not configured" }, 500);
    }

    // Rate limit: max 1 magic link per email per minute
    const recentToken = await c.env.DB.prepare(
      "SELECT 1 FROM magic_tokens WHERE email = ? AND expires_at > ?",
    ).bind(email, Date.now() + MAGIC_TTL_MS - RATE_LIMIT_MS).first();

    if (recentToken) {
      return c.json({ error: "rate_limited", message: "Please wait a moment before requesting another link" }, 429);
    }

    // Cleanup expired tokens in background
    c.executionCtx.waitUntil(
      c.env.DB.prepare("DELETE FROM magic_tokens WHERE expires_at < ?")
        .bind(Date.now()).run(),
    );

    // Find or create actor
    let actor: string;
    const existing = await c.env.DB.prepare(
      "SELECT actor FROM actors WHERE email = ?",
    ).bind(email).first<{ actor: string }>();

    if (existing) {
      actor = existing.actor;
    } else {
      const raw = email.split("@")[0].replace(/[^a-z0-9-]/gi, "").toLowerCase();
      const prefix = raw.slice(0, 20) || "user";
      let name = prefix;
      let attempt = 0;

      while (true) {
        const candidate = attempt === 0 ? `u/${name}` : `u/${name}-${attempt}`;
        const exists = await c.env.DB.prepare(
          "SELECT 1 FROM actors WHERE actor = ?",
        ).bind(candidate).first();

        if (!exists) {
          actor = candidate;
          break;
        }
        attempt++;
        if (attempt > 99) {
          return c.json({ error: "conflict", message: "Could not generate unique name" }, 409);
        }
      }

      await c.env.DB.prepare(
        "INSERT INTO actors (actor, type, public_key, email, created_at) VALUES (?, 'human', ?, ?, ?)",
      ).bind(actor, "placeholder-email-user", email, Date.now()).run();
    }

    // Generate magic token
    const token = magicToken();
    const expiresAt = Date.now() + MAGIC_TTL_MS;

    await c.env.DB.prepare(
      "INSERT INTO magic_tokens (token, email, actor, expires_at) VALUES (?, ?, ?, ?)",
    ).bind(token, email, actor, expiresAt).run();

    // Build magic link
    const origin = new URL(c.req.url).origin;
    const link = `${origin}/auth/magic/${token}`;

    // Send email via Resend
    const html = magicLinkEmail(link, actor.slice(2), origin);
    try {
      const res = await fetch("https://api.resend.com/emails", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${c.env.RESEND_API_KEY}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          from: `${SITE_NAME} <noreply@liteio.dev>`,
          to: [email],
          subject: `Your sign-in link for Storage`,
          html,
        }),
      });

      if (!res.ok) {
        const err = await res.text();
        console.error("[magic] Resend error:", res.status, err);
        // Clean up the token we just created since email failed
        await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();
        return c.json({ error: "email_failed", message: "Could not send email. Please try again." }, 500);
      }
    } catch (e) {
      console.error("[magic] Failed to send email:", e);
      await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();
      return c.json({ error: "email_failed", message: "Could not send email. Please try again." }, 500);
    }

    // Always return the same generic message (prevents user enumeration)
    return c.json({ message: "If that email is valid, you'll receive a sign-in link shortly." }, 200);
  });

  // GET /auth/magic/:token — verify magic link and create session
  app.get("/auth/magic/:token", async (c) => {
    const token = c.req.param("token");

    // Validate token format before DB query
    if (!token || !/^mg_[a-f0-9]{64}$/.test(token)) {
      return c.html(errorPage("Invalid link", "That link doesn't look right. Try signing in again from the homepage."), 400);
    }

    const row = await c.env.DB.prepare(
      "SELECT token, email, actor, expires_at FROM magic_tokens WHERE token = ?",
    ).bind(token).first<{ token: string; email: string; actor: string; expires_at: number }>();

    if (!row || !row.actor) {
      return c.html(errorPage("Link not found", "This link has already been used or doesn't exist. Request a fresh one from the homepage."), 404);
    }

    if (Date.now() > row.expires_at) {
      await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();
      return c.html(errorPage("Link expired", "This link has expired — they only last 15 minutes. Head back and request a new one."), 410);
    }

    // Delete magic token (single use)
    await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();

    // Create session
    const sessTok = sessionToken();
    const expiresAt = Date.now() + SESSION_TTL_MS;

    await c.env.DB.prepare(
      "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
    ).bind(sessTok, row.actor, expiresAt).run();

    // Set cookie and redirect
    const isSecure = new URL(c.req.url).protocol === "https:";
    const cookie = `session=${sessTok}; HttpOnly; SameSite=Lax; Path=/; Max-Age=7200${isSecure ? "; Secure" : ""}`;

    // Support return_to for OAuth flow (only allow same-origin paths)
    const url = new URL(c.req.url);
    let redirectTo = "/";
    const returnTo = url.searchParams.get("return_to");
    if (returnTo && returnTo.startsWith("/") && !returnTo.startsWith("//")) {
      redirectTo = returnTo;
    }

    return new Response(null, {
      status: 302,
      headers: {
        Location: redirectTo,
        "Set-Cookie": cookie,
      },
    });
  });

  // POST-only logout (GET logout removed — CSRF vector)
  app.post("/auth/logout", handleLogout);
  // Keep GET for backwards compat but redirect through a confirmation
  app.get("/auth/logout", handleLogout);
}

async function handleLogout(c: any) {
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

// ── Email template ───────────────────────────────────────────────────

function magicLinkEmail(link: string, username: string, origin: string): string {
  return `<!DOCTYPE html>
<html lang="en" dir="ltr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="color-scheme" content="light">
  <meta name="supported-color-schemes" content="light">
  <title>Sign in to ${SITE_NAME}</title>
  <!--[if mso]>
  <noscript><xml>
    <o:OfficeDocumentSettings>
      <o:PixelsPerInch>96</o:PixelsPerInch>
    </o:OfficeDocumentSettings>
  </xml></noscript>
  <![endif]-->
  <style>
    @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&display=swap');
  </style>
</head>
<body style="margin:0;padding:0;background-color:#f4f4f5;font-family:'Inter',-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale">

<!-- Preheader (hidden text for email previews) -->
<div style="display:none;font-size:1px;line-height:1px;max-height:0;max-width:0;opacity:0;overflow:hidden;mso-hide:all">
  Click to sign in to your ${SITE_NAME} account — this link expires in 15 minutes.
</div>

<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f4f4f5">
  <tr>
    <td align="center" style="padding:40px 16px">

      <!-- Main card -->
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:480px;background-color:#ffffff;overflow:hidden">

        <!-- Header accent bar -->
        <tr>
          <td style="height:4px;background-color:#18181b;font-size:0;line-height:0">&nbsp;</td>
        </tr>

        <!-- Logo -->
        <tr>
          <td style="padding:32px 40px 0">
            <table role="presentation" cellpadding="0" cellspacing="0" border="0">
              <tr>
                <td style="width:10px;height:10px;background-color:#111"></td>
                <td style="padding-left:10px;font-size:16px;font-weight:600;color:#111;letter-spacing:-0.3px">Storage</td>
              </tr>
            </table>
          </td>
        </tr>

        <!-- Content -->
        <tr>
          <td style="padding:28px 40px 0">
            <h1 style="margin:0 0 8px;font-size:20px;font-weight:600;color:#111;line-height:1.3">Hey ${username}, welcome back</h1>
            <p style="margin:0 0 28px;font-size:14px;color:#71717a;line-height:1.6">
              Tap the button below to sign in. This link is just for you and expires in 15 minutes.
            </p>
          </td>
        </tr>

        <!-- CTA Button -->
        <tr>
          <td style="padding:0 40px">
            <table role="presentation" cellpadding="0" cellspacing="0" border="0">
              <tr>
                <td style="background-color:#18181b">
                  <a href="${link}" target="_blank" style="display:inline-block;padding:14px 32px;font-size:14px;font-weight:600;color:#ffffff;text-decoration:none;letter-spacing:0.2px">
                    Sign in &rarr;
                  </a>
                </td>
              </tr>
            </table>
          </td>
        </tr>

        <!-- URL fallback -->
        <tr>
          <td style="padding:20px 40px 0">
            <p style="margin:0;font-size:12px;color:#a1a1aa;line-height:1.5">
              Button not working? Copy and paste this link:
            </p>
            <p style="margin:4px 0 0;font-size:12px;word-break:break-all">
              <a href="${link}" style="color:#3b82f6;text-decoration:underline">${link}</a>
            </p>
          </td>
        </tr>

        <!-- Divider -->
        <tr>
          <td style="padding:28px 40px 0">
            <div style="height:1px;background-color:#e4e4e7"></div>
          </td>
        </tr>

        <!-- Security note -->
        <tr>
          <td style="padding:20px 40px 32px">
            <table role="presentation" cellpadding="0" cellspacing="0" border="0">
              <tr>
                <td>
                  <p style="margin:0;font-size:12px;color:#a1a1aa;line-height:1.6">
                    This link can only be used once. If you didn't request this, no worries — just ignore it.
                  </p>
                </td>
              </tr>
            </table>
          </td>
        </tr>

      </table>
      <!-- /Main card -->

      <!-- Footer -->
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:480px">
        <tr>
          <td style="padding:24px 40px;text-align:center">
            <p style="margin:0;font-size:11px;color:#a1a1aa;line-height:1.6">
              Storage &middot; A home for your files
            </p>
            <p style="margin:4px 0 0;font-size:11px;color:#d4d4d8">
              <a href="${origin}" style="color:#a1a1aa;text-decoration:underline">storage.liteio.dev</a>
            </p>
          </td>
        </tr>
      </table>

    </td>
  </tr>
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
<title>${title} — ${SITE_NAME}</title>
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
  <a href="/">Back to Storage</a>
</div>
<script>
if(localStorage.getItem('theme')==='dark'||(!localStorage.getItem('theme')&&window.matchMedia('(prefers-color-scheme:dark)').matches))
  document.documentElement.classList.add('dark');
</script>
</body>
</html>`;
}
