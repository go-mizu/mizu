function rand(n: number): string {
  const bytes = new Uint8Array(n);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

export function objectId(): string {
  return `o_${rand(12)}`;
}

export function shareId(): string {
  return `sh_${rand(12)}`;
}

export function challengeId(): string {
  return `ch_${rand(12)}`;
}

export function sessionToken(): string {
  return rand(32);
}

export function nonce(): string {
  return rand(32);
}

export function magicToken(): string {
  return rand(32);
}

export function publicLinkId(): string {
  return `pl_${rand(12)}`;
}

export function publicLinkToken(): string {
  return rand(24);
}

export function apiKeyId(): string {
  return `ak_${rand(12)}`;
}

export function apiKeyToken(): string {
  return `sk_${rand(32)}`;
}
