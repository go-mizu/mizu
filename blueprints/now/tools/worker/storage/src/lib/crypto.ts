export function base64urlDecode(s: string): Uint8Array {
  let b64 = s.replace(/-/g, "+").replace(/_/g, "/");
  while (b64.length % 4) b64 += "=";
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

export function base64urlEncode(bytes: Uint8Array): string {
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export async function importEd25519PublicKey(b64url: string): Promise<CryptoKey> {
  const raw = base64urlDecode(b64url);
  return crypto.subtle.importKey("raw", raw, { name: "Ed25519" } as any, false, ["verify"]);
}

export async function verifyEd25519(
  publicKey: CryptoKey,
  signature: Uint8Array,
  message: string,
): Promise<boolean> {
  const data = new TextEncoder().encode(message);
  return crypto.subtle.verify({ name: "Ed25519" } as any, publicKey, signature, data);
}
