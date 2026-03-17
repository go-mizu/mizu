import type { Context, Next } from "hono";
import type { Env, Variables } from "./types";
import {
  base64urlDecode,
  sha256hex,
  importEd25519PublicKey,
  buildCanonicalRequest,
  buildStringToSign,
  sortedQueryString,
  verifyEd25519,
} from "./crypto";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

// Per-isolate public key cache with 5-minute TTL
const KEY_CACHE = new Map<string, { key: CryptoKey; cachedAt: number }>();
const CACHE_TTL_MS = 5 * 60 * 1000;

async function getPublicKey(db: D1Database, actor: string): Promise<CryptoKey | null> {
  const cached = KEY_CACHE.get(actor);
  if (cached && Date.now() - cached.cachedAt < CACHE_TTL_MS) {
    return cached.key;
  }

  const row = await db.prepare("SELECT public_key FROM actors WHERE actor = ?")
    .bind(actor)
    .first<{ public_key: string }>();
  if (!row) return null;

  try {
    const key = await importEd25519PublicKey(row.public_key);
    KEY_CACHE.set(actor, { key, cachedAt: Date.now() });
    return key;
  } catch {
    return null;
  }
}

export function invalidateKeyCache(actor: string): void {
  KEY_CACHE.delete(actor);
}

interface ParsedAuth {
  actor: string;
  timestamp: string;
  signature: Uint8Array;
}

function parseAuthHeader(header: string): ParsedAuth | null {
  if (!header.startsWith("CHAT-ED25519 ")) return null;
  const rest = header.slice("CHAT-ED25519 ".length);

  let actor = "";
  let timestamp = "";
  let sigB64 = "";

  for (const part of rest.split(", ")) {
    const eq = part.indexOf("=");
    if (eq === -1) continue;
    const key = part.slice(0, eq);
    const val = part.slice(eq + 1);
    if (key === "Credential") actor = val;
    else if (key === "Timestamp") timestamp = val;
    else if (key === "Signature") sigB64 = val;
  }

  if (!actor || !timestamp || !sigB64) return null;

  try {
    return { actor, timestamp, signature: base64urlDecode(sigB64) };
  } catch {
    return null;
  }
}

const TIMESTAMP_WINDOW_S = 5 * 60;

export async function signatureAuth(c: AppContext, next: Next) {
  const authHeader = c.req.header("Authorization");
  if (!authHeader) return c.json({ error: "Unauthorized" }, 401);

  const parsed = parseAuthHeader(authHeader);
  if (!parsed) return c.json({ error: "Invalid authorization format" }, 401);

  // Timestamp check
  const ts = parseInt(parsed.timestamp, 10);
  if (isNaN(ts)) return c.json({ error: "Invalid timestamp" }, 401);
  const now = Math.floor(Date.now() / 1000);
  if (Math.abs(now - ts) > TIMESTAMP_WINDOW_S) {
    return c.json({ error: "Timestamp out of range" }, 401);
  }

  // Look up public key
  const publicKey = await getPublicKey(c.env.DB, parsed.actor);
  if (!publicKey) return c.json({ error: "Unknown actor" }, 401);

  // Reconstruct canonical request
  const url = new URL(c.req.url);
  const body = await c.req.raw.clone().text();
  const bodyHash = await sha256hex(body);
  const query = sortedQueryString(url);
  const canonical = buildCanonicalRequest(c.req.method, url.pathname, query, bodyHash);
  const canonicalHash = await sha256hex(canonical);
  const stringToSign = buildStringToSign(parsed.timestamp, parsed.actor, canonicalHash);

  // Verify signature
  const valid = await verifyEd25519(publicKey, parsed.signature, stringToSign);
  if (!valid) return c.json({ error: "Invalid signature" }, 401);

  c.set("actor", parsed.actor);
  await next();
}
