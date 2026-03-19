import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { apiKeyId, apiKeyToken } from "./id";
import { errorResponse } from "./error";
import { requireScope } from "./authorize";
import { audit } from "./audit";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const VALID_SCOPES = new Set([
  "*",
  "files:read",
  "files:write",
  "folders:read",
  "folders:write",
  "shares:read",
  "shares:write",
  "drive:read",
  "drive:write",
  "links:manage",
]);

const MAX_KEYS_PER_ACTOR = 20;

// POST /api-keys — create a scoped API key
export async function createApiKey(c: AppContext) {
  // API keys can only be created with session tokens (full access)
  const scopes = c.get("scopes") || "*";
  if (scopes !== "*") {
    return errorResponse(c, "forbidden", "API keys can only be created with a session token");
  }

  const actor = c.get("actor");

  let body: {
    name?: string;
    scopes?: string[];
    path_prefix?: string;
    expires_in?: number;
  };
  try {
    body = await c.req.json();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.name || typeof body.name !== "string" || body.name.length > 64) {
    return errorResponse(c, "invalid_request", "name is required (max 64 chars)");
  }

  // Validate scopes
  const requestedScopes = body.scopes || ["*"];
  for (const scope of requestedScopes) {
    if (!VALID_SCOPES.has(scope)) {
      return errorResponse(c, "invalid_request", `Invalid scope: ${scope}`);
    }
  }

  // Check key limit
  const count = await c.env.DB.prepare(
    "SELECT COUNT(*) as count FROM api_keys WHERE actor = ?",
  )
    .bind(actor)
    .first<{ count: number }>();

  if (count && count.count >= MAX_KEYS_PER_ACTOR) {
    return errorResponse(c, "conflict", `Maximum ${MAX_KEYS_PER_ACTOR} API keys per actor`);
  }

  const id = apiKeyId();
  const token = apiKeyToken();
  const tokenHash = await sha256(token);
  const now = Date.now();
  const expiresAt = body.expires_in ? now + body.expires_in * 1000 : null;
  const scopesStr = requestedScopes.join(",");
  const pathPrefix = body.path_prefix?.replace(/^\/+/, "") || "";

  await c.env.DB.prepare(
    `INSERT INTO api_keys (id, actor, token_hash, name, scopes, path_prefix, expires_at, last_used_at, created_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?)`,
  )
    .bind(id, actor, tokenHash, body.name, scopesStr, pathPrefix, expiresAt, now)
    .run();

  audit(c, "apikey.create", undefined, { key_id: id, name: body.name, scopes: requestedScopes });

  return c.json({
    id,
    token, // Only shown once!
    name: body.name,
    scopes: requestedScopes,
    path_prefix: pathPrefix || undefined,
    expires_at: expiresAt ? new Date(expiresAt).toISOString() : null,
    created_at: now,
  }, 201);
}

// GET /api-keys — list my API keys (no tokens shown)
export async function listApiKeys(c: AppContext) {
  const actor = c.get("actor");

  const { results } = await c.env.DB.prepare(`
    SELECT id, name, scopes, path_prefix, expires_at, last_used_at, created_at
    FROM api_keys
    WHERE actor = ?
    ORDER BY created_at DESC
  `)
    .bind(actor)
    .all();

  const items = (results || []).map((row: any) => ({
    id: row.id,
    name: row.name,
    scopes: row.scopes.split(","),
    path_prefix: row.path_prefix || undefined,
    expires_at: row.expires_at ? new Date(row.expires_at).toISOString() : null,
    last_used_at: row.last_used_at ? new Date(row.last_used_at).toISOString() : null,
    created_at: row.created_at,
  }));

  return c.json({ items });
}

// DELETE /api-keys/:id — revoke an API key
export async function deleteApiKey(c: AppContext) {
  const actor = c.get("actor");
  const id = c.req.param("id");

  const key = await c.env.DB.prepare(
    "SELECT actor FROM api_keys WHERE id = ?",
  )
    .bind(id)
    .first<{ actor: string }>();

  if (!key) {
    return errorResponse(c, "not_found", "API key not found");
  }
  if (key.actor !== actor) {
    return errorResponse(c, "forbidden", "Not your API key");
  }

  await c.env.DB.prepare("DELETE FROM api_keys WHERE id = ?").bind(id).run();
  audit(c, "apikey.revoke", undefined, { key_id: id });

  return c.json({ deleted: true });
}

async function sha256(data: string): Promise<string> {
  const encoded = new TextEncoder().encode(data);
  const hash = await crypto.subtle.digest("SHA-256", encoded);
  return Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0")).join("");
}
