import type { Context } from "hono";
import type { App, Env, Variables } from "../types";
import { apiKeyId, apiKeyToken, magicToken, sessionToken } from "../lib/id";
import { sha256 } from "../middleware/auth";
import { esc } from "../pages/layout";

type C = Context<{ Bindings: Env; Variables: Variables }>;

const CODE_TTL_MS = 5 * 60 * 1000; // 5 minutes
const API_KEY_TTL_MS = 90 * 24 * 60 * 60 * 1000; // 90 days
const SESSION_TTL_MS = 2 * 60 * 60 * 1000;
const MAGIC_TTL_MS = 15 * 60 * 1000; // 15 minutes

function rand(n: number): string {
  const bytes = new Uint8Array(n);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

function base64urlEncode(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

// ── Well-known endpoints ──────────────────────────────────────────────

function protectedResourceMetadata(c: C) {
  const origin = new URL(c.req.url).origin;
  return c.json({
    resource: `${origin}/mcp`,
    authorization_servers: [origin],
    scopes_supported: ["storage:read", "storage:write", "storage:admin"],
    resource_name: "Storage",
  });
}

function authorizationServerMetadata(c: C) {
  const origin = new URL(c.req.url).origin;
  return c.json({
    issuer: origin,
    authorization_endpoint: `${origin}/oauth/authorize`,
    token_endpoint: `${origin}/oauth/token`,
    registration_endpoint: `${origin}/oauth/register`,
    response_types_supported: ["code"],
    grant_types_supported: ["authorization_code"],
    code_challenge_methods_supported: ["S256"],
    token_endpoint_auth_methods_supported: ["none"],
    scopes_supported: ["storage:read", "storage:write", "storage:admin"],
  });
}

// ── Dynamic Client Registration (RFC 7591) ────────────────────────────

async function registerClient(c: C) {
  let body: any;
  try {
    body = await c.req.json();
  } catch {
    return c.json({ error: "invalid_request" }, 400);
  }

  const redirectUris = body.redirect_uris;
  if (!Array.isArray(redirectUris) || redirectUris.length === 0) {
    return c.json({ error: "invalid_client_metadata", error_description: "redirect_uris required" }, 400);
  }

  const clientId = body.client_id || `oc_${rand(16)}`;
  const clientName = body.client_name || "";
  const authMethod = body.token_endpoint_auth_method || "none";

  await c.env.DB.prepare(
    "INSERT OR REPLACE INTO oauth_clients (client_id, redirect_uris, client_name, token_endpoint_auth_method, created_at) VALUES (?, ?, ?, ?, ?)",
  ).bind(clientId, JSON.stringify(redirectUris), clientName, authMethod, Date.now()).run();

  return c.json({
    client_id: clientId,
    client_name: clientName,
    redirect_uris: redirectUris,
    grant_types: ["authorization_code"],
    response_types: ["code"],
    token_endpoint_auth_method: authMethod,
  }, 201);
}

// ── Authorization endpoint ────────────────────────────────────────────

async function authorizeEndpoint(c: C) {
  const url = new URL(c.req.url);
  const clientId = url.searchParams.get("client_id") || "";
  const redirectUri = url.searchParams.get("redirect_uri") || "";
  const responseType = url.searchParams.get("response_type") || "";
  const codeChallenge = url.searchParams.get("code_challenge") || "";
  const codeChallengeMethod = url.searchParams.get("code_challenge_method") || "";
  const scope = url.searchParams.get("scope") || "";

  if (responseType !== "code") {
    return c.html(errorPage("Invalid request", "response_type must be 'code'"), 400);
  }
  if (!clientId) return c.html(errorPage("Invalid request", "client_id is required"), 400);
  if (!redirectUri) return c.html(errorPage("Invalid request", "redirect_uri is required"), 400);
  if (!codeChallenge || codeChallengeMethod !== "S256") {
    return c.html(errorPage("Invalid request", "PKCE with S256 is required"), 400);
  }

  const client = await c.env.DB.prepare(
    "SELECT redirect_uris, client_name FROM oauth_clients WHERE client_id = ?",
  ).bind(clientId).first<{ redirect_uris: string; client_name: string }>();

  if (client) {
    const uris = JSON.parse(client.redirect_uris) as string[];
    if (!uris.includes(redirectUri)) {
      return c.html(errorPage("Invalid request", "redirect_uri does not match registered URIs"), 400);
    }
  }

  // Use client_name for display (falls back to a friendly label if raw oc_ ID)
  const displayName = client?.client_name || friendlyClientName(clientId);

  const actor = await getSessionFromCookie(c);
  const oauthParams = url.search;

  if (actor) {
    return c.html(consentPage(actor, displayName, scope, oauthParams));
  }
  return c.html(loginPage(displayName, oauthParams));
}

// POST /oauth/authorize — handle consent or login
async function authorizeSubmit(c: C) {
  const form = await c.req.formData();
  const action = form.get("action") as string;

  if (action === "login") return handleLoginSubmit(c, form);
  if (action === "authorize") return handleConsentSubmit(c, form);
  return c.html(errorPage("Invalid request", "Unknown action"), 400);
}

async function handleLoginSubmit(c: C, form: FormData) {
  const email = (form.get("email") as string || "").trim().toLowerCase();
  const oauthParams = form.get("oauth_params") as string || "";

  if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    return c.html(errorPage("Invalid email", "Please enter a valid email address"), 400);
  }

  if (!c.env.RESEND_API_KEY) {
    console.error("[oauth] RESEND_API_KEY not configured");
    return c.html(errorPage("Service unavailable", "Email service not configured"), 500);
  }

  // Find or create actor
  let actorName: string;
  const existing = await c.env.DB.prepare("SELECT actor FROM actors WHERE email = ?")
    .bind(email).first<{ actor: string }>();

  if (existing) {
    actorName = existing.actor;
  } else {
    const local = email.split("@")[0].replace(/[^a-zA-Z0-9._-]/g, "").slice(0, 32);
    actorName = `u/${local}`;
    const collision = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
      .bind(actorName).first();
    if (collision) {
      actorName = `u/${local}.${Date.now().toString(36).slice(-4)}`;
    }
    await c.env.DB.prepare(
      "INSERT INTO actors (actor, type, email, created_at) VALUES (?, 'human', ?, ?)",
    ).bind(actorName, email, Date.now()).run();
  }

  // Create magic link token (email verification required — no direct session)
  const token = magicToken();
  const expiresAt = Date.now() + MAGIC_TTL_MS;
  await c.env.DB.prepare(
    "INSERT INTO magic_tokens (token, email, actor, expires_at) VALUES (?, ?, ?, ?)",
  ).bind(token, email, actorName, expiresAt).run();

  // Build magic link that returns to the OAuth flow after verification
  const origin = new URL(c.req.url).origin;
  const returnTo = `/oauth/authorize${oauthParams}`;
  const link = `${origin}/auth/magic/${token}?return_to=${encodeURIComponent(returnTo)}`;

  // Send email via Resend
  try {
    const res = await fetch("https://api.resend.com/emails", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${c.env.RESEND_API_KEY}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        from: "Storage <noreply@liteio.dev>",
        to: [email],
        subject: "Your sign-in link for Storage",
        html: oauthMagicLinkEmail(link, actorName.slice(2), origin),
      }),
    });

    if (!res.ok) {
      const err = await res.text();
      console.error("[oauth] Resend error:", res.status, err);
      await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();
      return c.html(errorPage("Email failed", "Could not send email. Please try again."), 500);
    }
  } catch (e) {
    console.error("[oauth] Failed to send email:", e);
    await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();
    return c.html(errorPage("Email failed", "Could not send email. Please try again."), 500);
  }

  return c.html(checkEmailPage(email));
}

async function handleConsentSubmit(c: C, form: FormData) {
  const oauthParams = form.get("oauth_params") as string || "";
  const actor = await getSessionFromCookie(c);

  if (!actor) return c.html(errorPage("Session expired", "Please try again"), 401);

  const params = new URLSearchParams(oauthParams.startsWith("?") ? oauthParams.slice(1) : oauthParams);
  const clientId = params.get("client_id") || "";
  const redirectUri = params.get("redirect_uri") || "";
  const state = params.get("state") || "";
  const codeChallenge = params.get("code_challenge") || "";
  const scope = params.get("scope") || "*";

  if (!clientId || !redirectUri || !codeChallenge) {
    return c.html(errorPage("Invalid request", "Missing OAuth parameters"), 400);
  }

  const code = rand(32);
  const expiresAt = Date.now() + CODE_TTL_MS;

  await c.env.DB.prepare(
    "INSERT INTO oauth_codes (code, actor, client_id, redirect_uri, scope, code_challenge, code_challenge_method, expires_at) VALUES (?, ?, ?, ?, ?, ?, 'S256', ?)",
  ).bind(code, actor, clientId, redirectUri, scope, codeChallenge, expiresAt).run();

  const callbackUrl = new URL(redirectUri);
  callbackUrl.searchParams.set("code", code);
  if (state) callbackUrl.searchParams.set("state", state);

  return Response.redirect(callbackUrl.toString(), 302);
}

// ── Token endpoint ────────────────────────────────────────────────────

async function tokenEndpoint(c: C) {
  let params: URLSearchParams;
  const ct = c.req.header("Content-Type") || "";
  if (ct.includes("application/json")) {
    const body = await c.req.json();
    params = new URLSearchParams();
    for (const [k, v] of Object.entries(body as Record<string, string>)) {
      params.set(k, v);
    }
  } else {
    const text = await c.req.text();
    params = new URLSearchParams(text);
  }

  const grantType = params.get("grant_type");
  if (grantType !== "authorization_code") {
    return c.json({ error: "unsupported_grant_type" }, 400);
  }

  const code = params.get("code") || "";
  const redirectUri = params.get("redirect_uri") || "";
  const clientId = params.get("client_id") || "";
  const codeVerifier = params.get("code_verifier") || "";

  if (!code || !codeVerifier) {
    return c.json({ error: "invalid_request", error_description: "code and code_verifier required" }, 400);
  }

  const authCode = await c.env.DB.prepare(
    "SELECT code, actor, client_id, redirect_uri, scope, code_challenge, expires_at FROM oauth_codes WHERE code = ?",
  ).bind(code).first<{
    code: string; actor: string; client_id: string; redirect_uri: string;
    scope: string; code_challenge: string; expires_at: number;
  }>();

  if (!authCode) return c.json({ error: "invalid_grant", error_description: "Invalid authorization code" }, 400);

  await c.env.DB.prepare("DELETE FROM oauth_codes WHERE code = ?").bind(code).run();

  if (Date.now() > authCode.expires_at) {
    return c.json({ error: "invalid_grant", error_description: "Authorization code expired" }, 400);
  }
  if (clientId && authCode.client_id !== clientId) {
    return c.json({ error: "invalid_grant", error_description: "client_id mismatch" }, 400);
  }
  if (redirectUri && authCode.redirect_uri !== redirectUri) {
    return c.json({ error: "invalid_grant", error_description: "redirect_uri mismatch" }, 400);
  }

  const verifierHash = base64urlEncode(
    await crypto.subtle.digest("SHA-256", new TextEncoder().encode(codeVerifier)),
  );
  if (verifierHash !== authCode.code_challenge) {
    return c.json({ error: "invalid_grant", error_description: "PKCE verification failed" }, 400);
  }

  let token: string;
  try {
    token = apiKeyToken();
    const tokenHash = await sha256(token);
    const id = apiKeyId();
    const now = Date.now();
    const expiresAt = now + API_KEY_TTL_MS;

    await c.env.DB.prepare(
      "INSERT INTO api_keys (id, actor, token_hash, name, prefix, expires_at, created_at) VALUES (?, ?, ?, ?, '', ?, ?)",
    ).bind(id, authCode.actor, tokenHash, `oauth:${authCode.client_id}`, expiresAt, now).run();
  } catch (e: any) {
    console.error(JSON.stringify({
      level: "error", component: "oauth", step: "token_create",
      actor: authCode.actor, client_id: authCode.client_id,
      error: e?.message, stack: e?.stack, ts: Date.now(),
    }));
    return c.json({ error: "server_error", error_description: "Failed to create access token" }, 500);
  }

  return c.json({
    access_token: token,
    token_type: "bearer",
    expires_in: Math.floor(API_KEY_TTL_MS / 1000),
    scope: authCode.scope === "*" ? "storage:read storage:write storage:admin" : authCode.scope,
  });
}

// ── Helpers ───────────────────────────────────────────────────────────

async function getSessionFromCookie(c: C): Promise<string | null> {
  const cookie = c.req.header("Cookie");
  if (!cookie) return null;
  const match = cookie.match(/(?:^|;\s*)session=([^;]+)/);
  if (!match) return null;
  const token = match[1];
  const session = await c.env.DB.prepare(
    "SELECT actor, expires_at FROM sessions WHERE token = ?",
  ).bind(token).first<{ actor: string; expires_at: number }>();
  if (!session || Date.now() > session.expires_at) return null;
  return session.actor;
}

// ── Helpers ───────────────────────────────────────────────────────────

/** Turn a raw client_id like "oc_74bfcc..." into a readable name */
function friendlyClientName(clientId: string): string {
  // Known clients
  const id = clientId.toLowerCase();
  if (id.includes("claude") || id.includes("anthropic")) return "Claude";
  if (id.includes("chatgpt") || id.includes("openai")) return "ChatGPT";
  // Auto-generated IDs (oc_ prefix) — show generic label
  if (clientId.startsWith("oc_")) return "An application";
  return clientId;
}

// ── HTML Pages ────────────────────────────────────────────────────────

const PAGE_STYLE = `
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: 'Inter', system-ui, sans-serif; background: #fafafa; color: #111;
    display: flex; align-items: center; justify-content: center; min-height: 100vh; }
  .dark body, html.dark body { background: #111; color: #eee; }
  .card { background: #fff; border: 1px solid #ddd; padding: 2.5rem; width: 100%; max-width: 420px; }
  .dark .card, html.dark .card { background: #1a1a1a; border-color: #333; }
  .card h1 { font-size: 1.1rem; font-weight: 600; margin-bottom: 0.25rem; }
  .card p { font-size: 0.85rem; color: #666; margin-bottom: 1.5rem; line-height: 1.5; }
  .dark .card p, html.dark .card p { color: #999; }
  .card label { display: block; font-size: 0.8rem; font-weight: 500; margin-bottom: 0.3rem; }
  .card input[type="email"] { width: 100%; padding: 0.6rem 0.75rem; border: 1px solid #ccc;
    font-size: 0.85rem; font-family: inherit; background: #fff; color: #111; margin-bottom: 1rem; }
  .dark .card input[type="email"], html.dark .card input[type="email"] {
    background: #222; color: #eee; border-color: #444; }
  .card button { width: 100%; padding: 0.65rem; background: #111; color: #fff; border: none;
    font-size: 0.85rem; font-weight: 500; cursor: pointer; font-family: inherit; }
  .dark .card button, html.dark .card button { background: #eee; color: #111; }
  .card button:hover { opacity: 0.9; }
  .card .secondary { background: transparent; color: #111; border: 1px solid #ccc; margin-top: 0.5rem; }
  .dark .card .secondary, html.dark .card .secondary { color: #eee; border-color: #444; }
  .scope-list { margin-bottom: 1.5rem; }
  .scope-item { display: flex; align-items: center; gap: 0.5rem; padding: 0.4rem 0;
    font-size: 0.85rem; border-bottom: 1px solid #eee; }
  .dark .scope-item, html.dark .scope-item { border-color: #333; }
  .scope-icon { font-size: 1rem; }
  .actor-badge { display: inline-block; font-family: 'JetBrains Mono', monospace;
    font-size: 0.8rem; background: #f0f0f0; padding: 0.15rem 0.5rem; margin-bottom: 1rem; }
  .dark .actor-badge, html.dark .actor-badge { background: #2a2a2a; }
  .brand { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.08em;
    color: #999; margin-bottom: 1rem; }
</style>
<script>
if(localStorage.getItem('theme')==='dark'||(!localStorage.getItem('theme')&&window.matchMedia('(prefers-color-scheme:dark)').matches)){
  document.documentElement.classList.add('dark');
}
</script>`;

function loginPage(clientName: string, oauthParams: string): string {
  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Sign in — Storage</title>${PAGE_STYLE}</head><body>
<div class="card">
  <div class="brand">Storage</div>
  <h1>Sign in to continue</h1>
  <p><strong>${esc(clientName)}</strong> wants to connect to your Storage files. Enter your email to sign in.</p>
  <form method="POST" action="/oauth/authorize">
    <input type="hidden" name="action" value="login">
    <input type="hidden" name="oauth_params" value="${esc(oauthParams)}">
    <label for="email">Email</label>
    <input type="email" id="email" name="email" placeholder="you@email.com" required autofocus>
    <button type="submit">Continue with email</button>
  </form>
</div></body></html>`;
}

function consentPage(actor: string, clientName: string, scope: string, oauthParams: string): string {
  const scopes = (!scope || scope === "*")
    ? ["Read your stored files", "Upload, edit, and delete files", "Create share links for your files"]
    : scope.split(/[\s,]+/).map((s) => {
        if (s === "storage:read") return "Read your stored files";
        if (s === "storage:write") return "Upload, edit, and delete files";
        if (s === "storage:admin") return "Create share links and manage API keys";
        return s;
      });

  const scopeItems = scopes.map((s) => `<div class="scope-item"><span class="scope-icon">&check;</span> ${esc(s)}</div>`).join("");
  const displayName = actor.startsWith("u/") ? actor.slice(2) : actor;

  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Authorize — Storage</title>${PAGE_STYLE}</head><body>
<div class="card">
  <div class="brand">Storage</div>
  <h1>Allow access?</h1>
  <p><strong>${esc(clientName)}</strong> wants to access your cloud files. This only applies to files on Storage, not your device.</p>
  <div class="actor-badge">${esc(displayName)}</div>
  <div class="scope-list">${scopeItems}</div>
  <form method="POST" action="/oauth/authorize">
    <input type="hidden" name="action" value="authorize">
    <input type="hidden" name="oauth_params" value="${esc(oauthParams)}">
    <button type="submit">Allow</button>
    <button type="button" class="secondary" onclick="window.close()">Deny</button>
  </form>
</div></body></html>`;
}

function checkEmailPage(email: string): string {
  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Check your email — Storage</title>${PAGE_STYLE}</head><body>
<div class="card">
  <div class="brand">Storage</div>
  <h1>Check your email</h1>
  <p>We sent a sign-in link to <strong>${esc(email)}</strong>. Click the link in the email to continue.</p>
  <p style="font-size:0.8rem;color:#999">The link expires in 15 minutes. Check your spam folder if you don't see it.</p>
</div></body></html>`;
}

function oauthMagicLinkEmail(link: string, username: string, origin: string): string {
  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Sign in to Storage</title></head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background:#f4f4f5">
  <tr><td align="center" style="padding:40px 16px">
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:480px;background:#fff;border-radius:12px;overflow:hidden">
      <tr><td style="height:4px;background:#111;font-size:0">&nbsp;</td></tr>
      <tr><td style="padding:32px 40px 0;font-size:16px;font-weight:600;color:#111">Storage</td></tr>
      <tr><td style="padding:24px 40px 0">
        <h1 style="margin:0 0 8px;font-size:18px;font-weight:600;color:#111">Sign in to continue</h1>
        <p style="margin:0 0 24px;font-size:14px;color:#71717a;line-height:1.6">
          Hi <strong style="color:#3f3f46">${username}</strong>, click the button to sign in and authorize access to your files.
        </p>
      </td></tr>
      <tr><td style="padding:0 40px">
        <a href="${link}" target="_blank" style="display:inline-block;padding:12px 28px;font-size:14px;font-weight:600;color:#fff;background:#111;text-decoration:none;border-radius:6px">
          Sign in to Storage
        </a>
      </td></tr>
      <tr><td style="padding:20px 40px">
        <p style="margin:0;font-size:12px;color:#a1a1aa">This link expires in 15 minutes and can only be used once.</p>
      </td></tr>
      <tr><td style="padding:0 40px"><div style="height:1px;background:#e4e4e7"></div></td></tr>
      <tr><td style="padding:16px 40px 32px">
        <p style="margin:0;font-size:11px;color:#a1a1aa">If you didn't request this, you can ignore this email.</p>
      </td></tr>
    </table>
  </td></tr>
</table>
</body></html>`;
}

function errorPage(title: string, message: string): string {
  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>${esc(title)} — Storage</title>${PAGE_STYLE}</head><body>
<div class="card">
  <div class="brand">Storage</div>
  <h1>${esc(title)}</h1>
  <p>${esc(message)}</p>
</div></body></html>`;
}

// ── Registration ──────────────────────────────────────────────────────

export function register(app: App) {
  app.get("/.well-known/oauth-protected-resource", protectedResourceMetadata);
  app.get("/.well-known/oauth-authorization-server", authorizationServerMetadata);
  app.post("/oauth/register", registerClient);
  app.get("/oauth/authorize", authorizeEndpoint);
  app.post("/oauth/authorize", authorizeSubmit);
  app.post("/oauth/token", tokenEndpoint);
}
