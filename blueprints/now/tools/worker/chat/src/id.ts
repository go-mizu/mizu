export function chatId(): string {
  return "c_" + hex(16);
}

export function messageId(): string {
  return "m_" + hex(16);
}

function hex(len: number): string {
  const bytes = new Uint8Array(len / 2);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
