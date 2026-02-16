/**
 * AES-256-GCM encryption/decryption using Web Crypto API.
 * Used to store temp email passwords encrypted (recoverable for re-login).
 *
 * Format: {iv_hex}:{ciphertext_base64}
 * Key derived from secret string via SHA-256 (deterministic, no salt needed).
 */

const ALGO = 'AES-GCM'
const IV_BYTES = 12

/** Derive AES-256 key from a secret string. */
async function deriveKey(secret: string): Promise<CryptoKey> {
  const raw = new TextEncoder().encode(secret)
  const hash = await crypto.subtle.digest('SHA-256', raw)
  return crypto.subtle.importKey('raw', hash, { name: ALGO }, false, ['encrypt', 'decrypt'])
}

/** Encrypt plaintext. Returns "iv_hex:ciphertext_base64". */
export async function encrypt(plaintext: string, secret: string): Promise<string> {
  const key = await deriveKey(secret)
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES))
  const data = new TextEncoder().encode(plaintext)
  const encrypted = await crypto.subtle.encrypt({ name: ALGO, iv }, key, data)

  const ivHex = Array.from(iv).map(b => b.toString(16).padStart(2, '0')).join('')
  const ctBase64 = btoa(String.fromCharCode(...new Uint8Array(encrypted)))
  return `${ivHex}:${ctBase64}`
}

/** Decrypt "iv_hex:ciphertext_base64". Returns plaintext. */
export async function decrypt(ciphertext: string, secret: string): Promise<string> {
  const [ivHex, ctBase64] = ciphertext.split(':')
  if (!ivHex || !ctBase64) throw new Error('Invalid ciphertext format')

  const key = await deriveKey(secret)
  const iv = new Uint8Array(ivHex.match(/.{2}/g)!.map(h => parseInt(h, 16)))
  const ct = Uint8Array.from(atob(ctBase64), c => c.charCodeAt(0))
  const decrypted = await crypto.subtle.decrypt({ name: ALGO, iv }, key, ct)
  return new TextDecoder().decode(decrypted)
}
