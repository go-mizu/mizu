/**
 * Rate limiting middleware for Cloudflare Workers.
 * Uses in-memory store with configurable limits per IP address.
 */

import { createMiddleware } from 'hono/factory';
import type { Context } from 'hono';
import { RateLimitError } from '../errors';

interface RateLimitConfig {
  /** Time window in milliseconds (default: 60000 = 1 minute) */
  windowMs?: number;
  /** Maximum requests per window (default: 100) */
  maxRequests?: number;
  /** Custom key generator function (default: uses CF-Connecting-IP header) */
  keyGenerator?: (c: Context) => string;
  /** Skip rate limiting for certain requests */
  skip?: (c: Context) => boolean;
  /** Custom message for rate limit errors */
  message?: string;
}

interface RateLimitEntry {
  count: number;
  resetAt: number;
}

// In-memory store for rate limit tracking
// Note: In production, consider using Cloudflare Durable Objects or KV for distributed rate limiting
const rateLimitStore = new Map<string, RateLimitEntry>();

// Clean up expired entries periodically
let lastCleanup = Date.now();
const CLEANUP_INTERVAL = 60_000; // Clean up every minute

function cleanupExpiredEntries(): void {
  const now = Date.now();
  if (now - lastCleanup < CLEANUP_INTERVAL) return;

  lastCleanup = now;
  for (const [key, entry] of rateLimitStore.entries()) {
    if (now > entry.resetAt) {
      rateLimitStore.delete(key);
    }
  }
}

/**
 * Create a rate limiting middleware with the given configuration.
 *
 * @example
 * ```typescript
 * // Basic usage - 100 requests per minute
 * app.use('/api/*', rateLimit());
 *
 * // Custom configuration
 * app.use('/api/*', rateLimit({
 *   windowMs: 60_000,    // 1 minute
 *   maxRequests: 50,     // 50 requests per minute
 *   keyGenerator: (c) => c.req.header('x-api-key') ?? 'anonymous',
 * }));
 * ```
 */
export function rateLimit(config: RateLimitConfig = {}) {
  const {
    windowMs = 60_000,
    maxRequests = 100,
    keyGenerator = defaultKeyGenerator,
    skip,
  } = config;

  return createMiddleware(async (c, next) => {
    // Check if we should skip rate limiting for this request
    if (skip?.(c)) {
      return next();
    }

    // Clean up expired entries periodically
    cleanupExpiredEntries();

    const key = keyGenerator(c);
    const now = Date.now();

    let entry = rateLimitStore.get(key);

    // Create new entry or reset if window has passed
    if (!entry || now > entry.resetAt) {
      entry = { count: 0, resetAt: now + windowMs };
      rateLimitStore.set(key, entry);
    }

    // Increment request count
    entry.count++;

    // Calculate remaining requests and reset time
    const remaining = Math.max(0, maxRequests - entry.count);
    const resetSeconds = Math.ceil((entry.resetAt - now) / 1000);

    // Set rate limit headers
    c.header('X-RateLimit-Limit', String(maxRequests));
    c.header('X-RateLimit-Remaining', String(remaining));
    c.header('X-RateLimit-Reset', String(Math.ceil(entry.resetAt / 1000)));

    // Check if rate limit exceeded
    if (entry.count > maxRequests) {
      throw new RateLimitError(resetSeconds);
    }

    return next();
  });
}

/**
 * Default key generator using Cloudflare's CF-Connecting-IP header.
 */
function defaultKeyGenerator(c: Context): string {
  // Cloudflare provides the real client IP in CF-Connecting-IP
  return (
    c.req.header('cf-connecting-ip') ??
    c.req.header('x-forwarded-for')?.split(',')[0]?.trim() ??
    c.req.header('x-real-ip') ??
    'anonymous'
  );
}

/**
 * Create a stricter rate limiter for sensitive endpoints.
 *
 * @example
 * ```typescript
 * app.use('/api/auth/*', strictRateLimit());
 * ```
 */
export function strictRateLimit(config: Partial<RateLimitConfig> = {}) {
  return rateLimit({
    windowMs: 60_000,      // 1 minute window
    maxRequests: 10,       // Only 10 requests per minute
    ...config,
  });
}

/**
 * Create a lenient rate limiter for read-only endpoints.
 *
 * @example
 * ```typescript
 * app.use('/api/search', lenientRateLimit());
 * ```
 */
export function lenientRateLimit(config: Partial<RateLimitConfig> = {}) {
  return rateLimit({
    windowMs: 60_000,      // 1 minute window
    maxRequests: 200,      // 200 requests per minute
    ...config,
  });
}
