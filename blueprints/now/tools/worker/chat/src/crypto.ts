const encoder = new TextEncoder();

// --- base64url ---

export function base64url(buf: ArrayBuffer | Uint8Array): string {
  const bytes = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  let s = "";
  for (const b of bytes) s += String.fromCharCode(b);
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function base64urlDecode(s: string): Uint8Array {
  const padded = s.replace(/-/g, "+").replace(/_/g, "/");
  const bin = atob(padded);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

// --- SHA-256 ---

export async function sha256hex(data: string | Uint8Array): Promise<string> {
  const input = typeof data === "string" ? encoder.encode(data) : data;
  const hash = await crypto.subtle.digest("SHA-256", input);
  return Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0")).join("");
}

// --- Ed25519 key import ---

export async function importEd25519PublicKey(base64urlKey: string): Promise<CryptoKey> {
  const raw = base64urlDecode(base64urlKey);
  return crypto.subtle.importKey("raw", raw, { name: "Ed25519" }, false, ["verify"]);
}

// --- Canonical request ---

export function buildCanonicalRequest(
  method: string,
  path: string,
  query: string,
  bodyHash: string
): string {
  return `${method}\n${path}\n${query}\n${bodyHash}`;
}

export function buildStringToSign(
  timestamp: string,
  actor: string,
  canonicalRequestHash: string
): string {
  return `CHAT-ED25519\n${timestamp}\n${actor}\n${canonicalRequestHash}`;
}

export function sortedQueryString(url: URL): string {
  const raw = url.search.startsWith("?") ? url.search.slice(1) : url.search;
  if (!raw) return "";
  const pairs = raw.split("&");
  pairs.sort();
  return pairs.join("&");
}

// --- Signature verification ---

export async function verifyEd25519(
  publicKey: CryptoKey,
  signature: Uint8Array,
  data: string
): Promise<boolean> {
  return crypto.subtle.verify("Ed25519", publicKey, signature, encoder.encode(data));
}

// --- Recovery code ---

export function generateRecoveryCode(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return base64url(bytes);
}
