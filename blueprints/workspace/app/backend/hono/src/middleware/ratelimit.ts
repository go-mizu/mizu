import { createMiddleware } from 'hono/factory';
import type { Env, Variables } from '../env';

interface RateLimitEntry {
  count: number;
  resetAt: number;
}

// In-memory rate limiter (resets on worker restart)
const rateLimitStore = new Map<string, RateLimitEntry>();

// Cleanup old entries on each request (lazy cleanup)
function cleanupExpiredEntries() {
  const now = Date.now();
  for (const [key, entry] of rateLimitStore.entries()) {
    if (entry.resetAt < now) {
      rateLimitStore.delete(key);
    }
  }
}

export function rateLimit(limit: number, windowMs: number) {
  return createMiddleware<{
    Bindings: Env;
    Variables: Variables;
  }>(async (c, next) => {
    // Lazy cleanup of expired entries
    cleanupExpiredEntries();

    // Get client IP
    const ip =
      c.req.header('cf-connecting-ip') ??
      c.req.header('x-forwarded-for')?.split(',')[0]?.trim() ??
      c.req.header('x-real-ip') ??
      'unknown';

    const key = `${ip}:${c.req.path}`;
    const now = Date.now();

    const entry = rateLimitStore.get(key);

    if (!entry || entry.resetAt < now) {
      // Start new window
      rateLimitStore.set(key, { count: 1, resetAt: now + windowMs });
    } else if (entry.count >= limit) {
      // Rate limit exceeded
      const retryAfter = Math.ceil((entry.resetAt - now) / 1000);
      c.header('Retry-After', String(retryAfter));
      c.header('X-RateLimit-Limit', String(limit));
      c.header('X-RateLimit-Remaining', '0');
      c.header('X-RateLimit-Reset', String(Math.ceil(entry.resetAt / 1000)));
      return c.json({ error: 'Too many requests' }, 429);
    } else {
      // Increment counter
      entry.count++;
    }

    // Add rate limit headers
    const current = rateLimitStore.get(key)!;
    c.header('X-RateLimit-Limit', String(limit));
    c.header('X-RateLimit-Remaining', String(Math.max(0, limit - current.count)));
    c.header('X-RateLimit-Reset', String(Math.ceil(current.resetAt / 1000)));

    await next();
  });
}

// Default rate limits
export const authRateLimit = rateLimit(10, 60 * 1000); // 10 req/min for auth
export const apiRateLimit = rateLimit(100, 60 * 1000); // 100 req/min for API

// Clear rate limit store (for testing)
export function clearRateLimitStore() {
  rateLimitStore.clear();
}
