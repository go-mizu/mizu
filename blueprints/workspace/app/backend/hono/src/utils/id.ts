export function generateId(): string {
  // Generate ULID-like ID without the ulid library (which calls crypto in global scope)
  const timestamp = Date.now().toString(36).padStart(10, '0');
  const randomBytes = new Uint8Array(10);
  crypto.getRandomValues(randomBytes);
  const random = Array.from(randomBytes)
    .map((b) => b.toString(36).padStart(2, '0'))
    .join('')
    .slice(0, 16);
  return (timestamp + random).toUpperCase();
}

export function generateToken(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}
