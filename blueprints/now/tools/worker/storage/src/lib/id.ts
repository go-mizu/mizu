function rand(n: number): string {
  const bytes = new Uint8Array(n);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

/** URL-safe base62: [0-9A-Za-z] */
function randBase62(n: number): string {
  const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
  const bytes = new Uint8Array(n);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => chars[b % 62]).join("");
}

export const challengeId  = () => `ch_${rand(12)}`;
export const sessionToken = () => rand(32);
export const nonce        = () => rand(32);
export const apiKeyId     = () => `ak_${rand(12)}`;
export const apiKeyToken  = () => `sk_${rand(32)}`;
export const shareToken   = () => randBase62(22);
export const magicToken   = () => `mg_${rand(32)}`;
