export function chatId(): string {
  return "c_" + hex(16);
}

export function messageId(): string {
  return "m_" + hex(16);
}

export function challengeId(): string {
  return "ch_" + hex(16);
}

export function nonce(): string {
  return hex(32);
}

export function sessionToken(): string {
  return hex(32);
}

export function magicToken(): string {
  return "ml_" + hex(32);
}

export function placeholderKey(): string {
  return "email-auth-" + hex(8);
}

function hex(len: number): string {
  const bytes = new Uint8Array(len);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
