/**
 * D1-backed rate limiter.
 *
 * Uses the `rate_limits` table to count requests per (endpoint, key) in a
 * sliding window. Probabilistic cleanup (1% chance) keeps the table small.
 */

import type { Context } from "hono";
import type { Env, Variables } from "../types";

type C = Context<{ Bindings: Env; Variables: Variables }>;

export interface RateLimitConfig {
  /** Identifier for the endpoint (e.g. "auth/register") */
  endpoint: string;
  /** Max requests allowed in the window */
  limit: number;
  /** Window size in milliseconds */
  windowMs: number;
}

export interface RateLimitResult {
  allowed: boolean;
  remaining: number;
  resetAt: number;
}

/**
 * Check and record a rate-limited request.
 * Returns whether the request is allowed.
 */
export async function checkRateLimit(
  db: D1Database,
  config: RateLimitConfig,
  key: string,
): Promise<RateLimitResult> {
  const now = Date.now();
  const windowStart = now - config.windowMs;

  // Probabilistic cleanup of old entries (1% of requests)
  if (Math.random() < 0.01) {
    db.prepare("DELETE FROM rate_limits WHERE ts < ?")
      .bind(windowStart)
      .run()
      .catch(() => {}); // fire-and-forget
  }

  // Count recent requests in window
  const row = await db
    .prepare(
      "SELECT COUNT(*) as cnt FROM rate_limits WHERE endpoint = ? AND key = ? AND ts > ?",
    )
    .bind(config.endpoint, key, windowStart)
    .first<{ cnt: number }>();

  const count = row?.cnt ?? 0;

  if (count >= config.limit) {
    return {
      allowed: false,
      remaining: 0,
      resetAt: now + config.windowMs,
    };
  }

  // Record this request
  await db
    .prepare("INSERT INTO rate_limits (endpoint, key, ts) VALUES (?, ?, ?)")
    .bind(config.endpoint, key, now)
    .run();

  return {
    allowed: true,
    remaining: config.limit - count - 1,
    resetAt: now + config.windowMs,
  };
}

/** Get the client IP from Cloudflare headers. */
export function getClientIp(c: C): string {
  return c.req.header("CF-Connecting-IP") || c.req.header("X-Forwarded-For")?.split(",")[0]?.trim() || "unknown";
}

/** Build a 429 JSON response with Retry-After header. */
export function rateLimitResponse(c: C, retryAfterSec = 60) {
  return c.json(
    { error: "rate_limited" as const, message: "Too many requests. Please try again later." },
    429 as const,
    { "Retry-After": String(retryAfterSec) },
  );
}
