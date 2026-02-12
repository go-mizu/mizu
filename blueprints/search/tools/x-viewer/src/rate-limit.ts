export function isRateLimitedError(err: unknown): boolean {
  const msg = err instanceof Error ? err.message : String(err)
  return msg.includes('rate limited') || msg.includes('(429)')
}
