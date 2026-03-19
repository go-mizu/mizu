import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { apiKeyId, apiKeyToken, magicToken, sessionToken } from "./id";
import { esc } from "./layout";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const CODE_TTL_MS = 5 * 60 * 1000; // 5 minutes
const API_KEY_TTL_MS = 90 * 24 * 60 * 60 * 1000; // 90 days
const SESSION_TTL_MS = 2 * 60 * 60 * 1000;
const MAGIC_TTL_MS = 15 * 60 * 1000;

function rand(n: number): string {
  const bytes = new Uint8Array(n);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

async function sha256(data: string): Promise<string> {
  const encoded = new TextEncoder().encode(data);
  const hash = await crypto.subtle.digest("SHA-256", encoded);
  return Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0")).join("");
}

function base64urlEncode(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

// ── Well-known endpoints ──────────────────────────────────────────────

export function protectedResourceMetadata(c: AppContext) {
  const origin = new URL(c.req.url).origin;
  return c.json({
    resource: `${origin}/mcp`,
    authorization_servers: [origin],
    scopes_supported: ["storage:read", "storage:write", "storage:admin"],
    resource_name: "storage.now",
  });
}

export function authorizationServerMetadata(c: AppContext) {
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

export async function registerClient(c: AppContext) {
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

  // Accept user-provided client_id or generate one
  const clientId = body.client_id || `oc_${rand(16)}`;
  const clientName = body.client_name || "";
  const authMethod = body.token_endpoint_auth_method || "none";

  // Upsert: if client_id already exists, update redirect_uris
  await c.env.DB.prepare(
    "INSERT OR REPLACE INTO oauth_clients (client_id, redirect_uris, client_name, token_endpoint_auth_method, created_at) VALUES (?, ?, ?, ?, ?)",
  )
    .bind(clientId, JSON.stringify(redirectUris), clientName, authMethod, Date.now())
    .run();

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

export async function authorizeEndpoint(c: AppContext) {
  const url = new URL(c.req.url);
  const clientId = url.searchParams.get("client_id") || "";
  const redirectUri = url.searchParams.get("redirect_uri") || "";
  const state = url.searchParams.get("state") || "";
  const responseType = url.searchParams.get("response_type") || "";
  const codeChallenge = url.searchParams.get("code_challenge") || "";
  const codeChallengeMethod = url.searchParams.get("code_challenge_method") || "";
  const scope = url.searchParams.get("scope") || "";

  // Validate required params
  if (responseType !== "code") {
    return c.html(errorPage("Invalid request", "response_type must be 'code'"), 400);
  }
  if (!clientId) {
    return c.html(errorPage("Invalid request", "client_id is required"), 400);
  }
  if (!redirectUri) {
    return c.html(errorPage("Invalid request", "redirect_uri is required"), 400);
  }
  if (!codeChallenge || codeChallengeMethod !== "S256") {
    return c.html(errorPage("Invalid request", "PKCE with S256 is required"), 400);
  }

  // Validate client (if registered)
  const client = await c.env.DB.prepare(
    "SELECT redirect_uris FROM oauth_clients WHERE client_id = ?",
  ).bind(clientId).first<{ redirect_uris: string }>();

  if (client) {
    const uris = JSON.parse(client.redirect_uris) as string[];
    if (!uris.includes(redirectUri)) {
      return c.html(errorPage("Invalid request", "redirect_uri does not match registered URIs"), 400);
    }
  }
  // If client not registered, we still allow it (for user-defined clients in ChatGPT)

  // Check if user has a session
  const actor = await getSessionFromCookie(c);
  const oauthParams = url.search; // preserve all OAuth params

  if (actor) {
    // Show consent page
    return c.html(consentPage(actor, clientId, scope, oauthParams));
  }

  // No session — show login page
  return c.html(loginPage(clientId, oauthParams));
}

// POST /oauth/authorize — handle consent or login
export async function authorizeSubmit(c: AppContext) {
  const form = await c.req.formData();
  const action = form.get("action") as string;

  if (action === "login") {
    return handleLoginSubmit(c, form);
  }
  if (action === "authorize") {
    return handleConsentSubmit(c, form);
  }
  return c.html(errorPage("Invalid request", "Unknown action"), 400);
}

async function handleLoginSubmit(c: AppContext, form: FormData) {
  const email = (form.get("email") as string || "").trim().toLowerCase();
  const oauthParams = form.get("oauth_params") as string || "";

  if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    return c.html(errorPage("Invalid email", "Please enter a valid email address"), 400);
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
      "INSERT INTO actors (actor, type, email, bio, created_at) VALUES (?, 'human', ?, '', ?)",
    ).bind(actorName, email, Date.now()).run();
  }

  // Create magic link that redirects back to /oauth/authorize with params
  const token = magicToken();
  const expiresAt = Date.now() + MAGIC_TTL_MS;
  await c.env.DB.prepare(
    "INSERT INTO magic_tokens (token, email, actor, expires_at) VALUES (?, ?, ?, ?)",
  ).bind(token, email, actorName, expiresAt).run();

  const origin = new URL(c.req.url).origin;
  const magicLink = `${origin}/oauth/callback/${token}${oauthParams}`;

  // OAuth flow: redirect directly — user is already in the browser.
  // No email round-trip needed (unlike regular magic link auth).
  return Response.redirect(magicLink, 302);
}

// GET /oauth/callback/:token — magic link lands here, sets session, redirects to authorize
export async function oauthMagicCallback(c: AppContext) {
  const token = c.req.param("token");
  const row = await c.env.DB.prepare(
    "SELECT email, actor, expires_at FROM magic_tokens WHERE token = ?",
  ).bind(token).first<{ email: string; actor: string; expires_at: number }>();

  if (!row || Date.now() > row.expires_at) {
    return c.html(errorPage("Invalid link", "This link has expired or is invalid"), 400);
  }

  await c.env.DB.prepare("DELETE FROM magic_tokens WHERE token = ?").bind(token).run();

  // Create session
  const sessToken = sessionToken();
  const expiresAt = Date.now() + SESSION_TTL_MS;
  await c.env.DB.prepare(
    "INSERT INTO sessions (token, actor, expires_at) VALUES (?, ?, ?)",
  ).bind(sessToken, row.actor, expiresAt).run();

  // Redirect back to /oauth/authorize with original OAuth params
  const url = new URL(c.req.url);
  const oauthSearch = url.search; // everything after ?
  const origin = url.origin;

  return new Response(null, {
    status: 302,
    headers: {
      Location: `${origin}/oauth/authorize${oauthSearch}`,
      "Set-Cookie": `session=${sessToken}; Path=/; HttpOnly; SameSite=Lax; Max-Age=7200`,
    },
  });
}

async function handleConsentSubmit(c: AppContext, form: FormData) {
  const oauthParams = form.get("oauth_params") as string || "";
  const actor = await getSessionFromCookie(c);

  if (!actor) {
    return c.html(errorPage("Session expired", "Please try again"), 401);
  }

  // Parse OAuth params
  const params = new URLSearchParams(oauthParams.startsWith("?") ? oauthParams.slice(1) : oauthParams);
  const clientId = params.get("client_id") || "";
  const redirectUri = params.get("redirect_uri") || "";
  const state = params.get("state") || "";
  const codeChallenge = params.get("code_challenge") || "";
  const scope = params.get("scope") || "*";

  if (!clientId || !redirectUri || !codeChallenge) {
    return c.html(errorPage("Invalid request", "Missing OAuth parameters"), 400);
  }

  // Generate authorization code
  const code = rand(32);
  const expiresAt = Date.now() + CODE_TTL_MS;

  await c.env.DB.prepare(
    "INSERT INTO oauth_codes (code, actor, client_id, redirect_uri, scope, code_challenge, code_challenge_method, expires_at) VALUES (?, ?, ?, ?, ?, ?, 'S256', ?)",
  ).bind(code, actor, clientId, redirectUri, scope, codeChallenge, expiresAt).run();

  // Redirect to callback
  const callbackUrl = new URL(redirectUri);
  callbackUrl.searchParams.set("code", code);
  if (state) callbackUrl.searchParams.set("state", state);

  return Response.redirect(callbackUrl.toString(), 302);
}

// ── Token endpoint ────────────────────────────────────────────────────

export async function tokenEndpoint(c: AppContext) {
  let params: URLSearchParams;
  const ct = c.req.header("Content-Type") || "";
  if (ct.includes("application/x-www-form-urlencoded")) {
    const text = await c.req.text();
    params = new URLSearchParams(text);
  } else if (ct.includes("application/json")) {
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

  // Look up auth code
  const authCode = await c.env.DB.prepare(
    "SELECT * FROM oauth_codes WHERE code = ?",
  ).bind(code).first<{
    code: string; actor: string; client_id: string; redirect_uri: string;
    scope: string; code_challenge: string; code_challenge_method: string; expires_at: number;
  }>();

  if (!authCode) {
    return c.json({ error: "invalid_grant", error_description: "Invalid authorization code" }, 400);
  }

  // Delete immediately (single-use)
  await c.env.DB.prepare("DELETE FROM oauth_codes WHERE code = ?").bind(code).run();

  // Validate expiry
  if (Date.now() > authCode.expires_at) {
    return c.json({ error: "invalid_grant", error_description: "Authorization code expired" }, 400);
  }

  // Validate client_id
  if (clientId && authCode.client_id !== clientId) {
    return c.json({ error: "invalid_grant", error_description: "client_id mismatch" }, 400);
  }

  // Validate redirect_uri
  if (redirectUri && authCode.redirect_uri !== redirectUri) {
    return c.json({ error: "invalid_grant", error_description: "redirect_uri mismatch" }, 400);
  }

  // Validate PKCE: SHA256(code_verifier) must match code_challenge
  const verifierHash = base64urlEncode(
    await crypto.subtle.digest("SHA-256", new TextEncoder().encode(codeVerifier)),
  );
  if (verifierHash !== authCode.code_challenge) {
    return c.json({ error: "invalid_grant", error_description: "PKCE verification failed" }, 400);
  }

  // Create an API key as the access token
  const token = apiKeyToken();
  const tokenHash = await sha256(token);
  const id = apiKeyId();
  const now = Date.now();
  const expiresAt = now + API_KEY_TTL_MS;

  // Map OAuth scopes to storage scopes
  const storageScopes = mapOAuthScopes(authCode.scope);

  await c.env.DB.prepare(
    "INSERT INTO api_keys (id, actor, token_hash, name, scopes, path_prefix, expires_at, created_at) VALUES (?, ?, ?, ?, ?, '', ?, ?)",
  ).bind(id, authCode.actor, tokenHash, `oauth:${authCode.client_id}`, storageScopes, expiresAt, now).run();

  return c.json({
    access_token: token,
    token_type: "bearer",
    expires_in: Math.floor(API_KEY_TTL_MS / 1000),
    scope: authCode.scope === "*" ? "storage:read storage:write storage:admin" : authCode.scope,
  });
}

function mapOAuthScopes(scope: string): string {
  if (!scope || scope === "*") return "*";
  const parts = scope.split(/[\s,]+/);
  const mapped: string[] = [];
  for (const p of parts) {
    switch (p) {
      case "storage:read":
        mapped.push("files:read", "folders:read", "drive:read");
        break;
      case "storage:write":
        mapped.push("files:write", "folders:write", "drive:write");
        break;
      case "storage:admin":
        mapped.push("shares:read", "shares:write", "links:manage");
        break;
    }
  }
  return mapped.length > 0 ? mapped.join(",") : "*";
}

// ── Helpers ───────────────────────────────────────────────────────────

async function getSessionFromCookie(c: AppContext): Promise<string | null> {
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

function loginPage(clientId: string, oauthParams: string): string {
  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Sign in — storage.now</title>${PAGE_STYLE}</head><body>
<div class="card">
  <div class="brand">storage.now</div>
  <h1>Sign in to authorize</h1>
  <p><strong>${esc(clientId)}</strong> wants to access your storage. Enter your email to continue.</p>
  <form method="POST" action="/oauth/authorize">
    <input type="hidden" name="action" value="login">
    <input type="hidden" name="oauth_params" value="${esc(oauthParams)}">
    <label for="email">Email</label>
    <input type="email" id="email" name="email" placeholder="you@example.com" required autofocus>
    <button type="submit">Continue with email</button>
  </form>
</div></body></html>`;
}

function consentPage(actor: string, clientId: string, scope: string, oauthParams: string): string {
  const scopes = (!scope || scope === "*")
    ? ["Read files and folders", "Write and delete files", "Manage shares and links"]
    : scope.split(/[\s,]+/).map((s) => {
        if (s === "storage:read") return "Read files and folders";
        if (s === "storage:write") return "Write and delete files";
        if (s === "storage:admin") return "Manage shares and links";
        return s;
      });

  const scopeItems = scopes.map((s) => `<div class="scope-item"><span class="scope-icon">✓</span> ${esc(s)}</div>`).join("");

  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Authorize — storage.now</title>${PAGE_STYLE}</head><body>
<div class="card">
  <div class="brand">storage.now</div>
  <h1>Authorize access</h1>
  <p><strong>${esc(clientId)}</strong> wants to access your storage.</p>
  <div class="actor-badge">${esc(actor)}</div>
  <div class="scope-list">${scopeItems}</div>
  <form method="POST" action="/oauth/authorize">
    <input type="hidden" name="action" value="authorize">
    <input type="hidden" name="oauth_params" value="${esc(oauthParams)}">
    <button type="submit">Authorize</button>
    <button type="button" class="secondary" onclick="window.close()">Cancel</button>
  </form>
</div></body></html>`;
}

function checkEmailPage(email: string): string {
  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Check your email — storage.now</title>${PAGE_STYLE}</head><body>
<div class="card">
  <div class="brand">storage.now</div>
  <h1>Check your email</h1>
  <p>We sent a sign-in link to <strong>${esc(email)}</strong>. Click it to continue authorization.</p>
</div></body></html>`;
}

function errorPage(title: string, message: string): string {
  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>${esc(title)} — storage.now</title>${PAGE_STYLE}</head><body>
<div class="card">
  <div class="brand">storage.now</div>
  <h1>${esc(title)}</h1>
  <p>${esc(message)}</p>
</div></body></html>`;
}
