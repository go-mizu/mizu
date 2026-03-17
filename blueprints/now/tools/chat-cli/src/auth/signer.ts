import * as ed from "@noble/ed25519";
import { sha512 } from "@noble/hashes/sha512";
import { sha256 } from "@noble/hashes/sha256";

// @noble/ed25519 v2 needs sha512 configured
ed.etc.sha512Sync = (...m: Uint8Array[]) => {
  const h = sha512.create();
  for (const msg of m) h.update(msg);
  return h.digest();
};

const encoder = new TextEncoder();

export function base64url(buf: Uint8Array): string {
  let s = "";
  for (const b of buf) s += String.fromCharCode(b);
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function base64urlDecode(s: string): Uint8Array {
  const padded = s.replace(/-/g, "+").replace(/_/g, "/");
  const bin = atob(padded);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

export async function sha256hex(data: string | Uint8Array): Promise<string> {
  const input = typeof data === "string" ? encoder.encode(data) : data;
  const hash = sha256(input);
  return Array.from(hash, (b) => b.toString(16).padStart(2, "0")).join("");
}

export function fingerprintSync(publicKey: Uint8Array): string {
  const hash = sha256(publicKey);
  return Array.from(hash.slice(0, 8), (b) => b.toString(16).padStart(2, "0")).join("");
}

export async function fingerprintAsync(publicKey: Uint8Array): Promise<string> {
  const hash = await sha256hex(publicKey);
  return hash.slice(0, 16);
}

export async function buildCanonicalRequest(
  method: string,
  path: string,
  query: string,
  body: string,
): Promise<string> {
  const sortedQuery = query ? query.split("&").sort().join("&") : "";
  const bodyHash = await sha256hex(body);
  return `${method}\n${path}\n${sortedQuery}\n${bodyHash}`;
}

export async function buildStringToSign(
  timestamp: number,
  actor: string,
  canonicalHash: string,
): Promise<string> {
  return `CHAT-ED25519\n${timestamp}\n${actor}\n${canonicalHash}`;
}

export function buildAuthHeader(actor: string, timestamp: number, signatureB64: string): string {
  return `CHAT-ED25519 Credential=${actor}, Timestamp=${timestamp}, Signature=${signatureB64}`;
}

export async function generateKeypair(): Promise<{ publicKey: Uint8Array; privateKey: Uint8Array }> {
  const seed = ed.utils.randomPrivateKey();
  const publicKey = ed.getPublicKey(seed);
  const full = new Uint8Array(64);
  full.set(seed, 0);
  full.set(publicKey, 32);
  return { publicKey, privateKey: full };
}

interface SignRequestOpts {
  actor: string;
  privateKey: Uint8Array;
  method: string;
  path: string;
  query: string;
  body: string;
}

export async function signRequest(opts: SignRequestOpts): Promise<string> {
  const { actor, privateKey, method, path, query, body } = opts;
  const timestamp = Math.floor(Date.now() / 1000);

  const canonical = await buildCanonicalRequest(method, path, query, body);
  const canonicalHash = await sha256hex(canonical);
  const stringToSign = await buildStringToSign(timestamp, actor, canonicalHash);

  const seed = privateKey.slice(0, 32);
  const signature = ed.sign(encoder.encode(stringToSign), seed);
  const sigB64 = base64url(signature);

  return buildAuthHeader(actor, timestamp, sigB64);
}
