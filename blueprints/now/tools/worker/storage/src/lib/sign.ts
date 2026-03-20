/**
 * HMAC-SHA256 signing for stateless share tokens.
 *
 * Token format: base64url(JSON claims) + "." + hex(HMAC-SHA256)
 * Claims: { o: owner, p: path, x: expires_at_ms }
 */

async function hmac(data: string, secret: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const sig = await crypto.subtle.sign("HMAC", key, new TextEncoder().encode(data));
  return Array.from(new Uint8Array(sig), (b) => b.toString(16).padStart(2, "0")).join("");
}

export interface ShareClaims {
  o: string; // owner
  p: string; // path
  x: number; // expires_at (epoch ms)
}

export async function createShareToken(claims: ShareClaims, secret: string): Promise<string> {
  const payload = btoa(JSON.stringify(claims))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
  const sig = await hmac(payload, secret);
  return `${payload}.${sig}`;
}

export async function verifyShareToken(token: string, secret: string): Promise<ShareClaims | null> {
  const dot = token.indexOf(".");
  if (dot === -1) return null;

  const payload = token.slice(0, dot);
  const sig = token.slice(dot + 1);

  const expected = await hmac(payload, secret);
  if (sig !== expected) return null;

  try {
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    const claims = JSON.parse(json) as ShareClaims;
    if (claims.x < Date.now()) return null;
    return claims;
  } catch {
    return null;
  }
}
