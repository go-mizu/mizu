import type { Context, Next } from "hono";
import type { Env, Variables } from "../types";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

interface RateLimitConfig {
  /** Max requests allowed in the window */
  limit: number;
  /** Window size in seconds */
  window: number;
  /** Key function: returns a unique key for the request (e.g., IP or actor) */
  keyFn: (c: AppContext) => string;
}

/**
 * Create a rate limiting middleware.
 *
 * Uses D1 with a simple sliding window counter.
 * The window is quantized to the nearest `config.window` seconds.
 */
export function rateLimit(config: RateLimitConfig) {
  return async (c: AppContext, next: Next) => {
    const key = config.keyFn(c);
    const now = Math.floor(Date.now() / 1000);
    const windowStart = now - (now % config.window);

    const rlKey = `${key}:${windowStart}`;

    try {
      // Atomic upsert + increment
      const row = await c.env.DB.prepare(
        "SELECT count FROM rate_limits WHERE key = ? AND window = ?",
      )
        .bind(rlKey, windowStart)
        .first<{ count: number }>();

      const currentCount = row?.count || 0;

      if (currentCount >= config.limit) {
        const retryAfter = windowStart + config.window - now;
        return c.json(
          { error: { code: "rate_limited", message: "Too many requests" } },
          { status: 429, headers: { "Retry-After": retryAfter.toString() } } as any,
        );
      }

      // Increment (or insert)
      if (row) {
        await c.env.DB.prepare(
          "UPDATE rate_limits SET count = count + 1 WHERE key = ? AND window = ?",
        )
          .bind(rlKey, windowStart)
          .run();
      } else {
        await c.env.DB.prepare(
          "INSERT OR REPLACE INTO rate_limits (key, count, window) VALUES (?, 1, ?)",
        )
          .bind(rlKey, windowStart)
          .run();
      }

      // Probabilistic cleanup of old windows (~2% chance)
      if (Math.random() < 0.02) {
        c.executionCtx.waitUntil(
          c.env.DB.prepare("DELETE FROM rate_limits WHERE window < ?")
            .bind(windowStart - config.window * 2)
            .run(),
        );
      }
    } catch {
      // If rate limiting fails, allow the request through
    }

    return next();
  };
}

/**
 * Extract client IP for rate limiting key.
 */
export function clientIP(c: AppContext): string {
  return c.req.header("CF-Connecting-IP") || c.req.header("X-Forwarded-For") || "unknown";
}

/**
 * Extract actor for rate limiting key (requires auth).
 */
export function actorKey(c: AppContext): string {
  return c.get("actor") || clientIP(c);
}

// Pre-configured rate limiters
export const authRateLimit = rateLimit({
  limit: 10,
  window: 60,
  keyFn: (c) => `auth:${clientIP(c)}`,
});

export const magicLinkRateLimit = rateLimit({
  limit: 5,
  window: 300,
  keyFn: (c) => `magic:${clientIP(c)}`,
});

export const registerRateLimit = rateLimit({
  limit: 5,
  window: 300,
  keyFn: (c) => `register:${clientIP(c)}`,
});

export const uploadRateLimit = rateLimit({
  limit: 100,
  window: 60,
  keyFn: (c) => `upload:${actorKey(c)}`,
});

export const shareRateLimit = rateLimit({
  limit: 30,
  window: 60,
  keyFn: (c) => `share:${actorKey(c)}`,
});

export const linkRateLimit = rateLimit({
  limit: 20,
  window: 60,
  keyFn: (c) => `link:${actorKey(c)}`,
});

export const publicAccessRateLimit = rateLimit({
  limit: 60,
  window: 60,
  keyFn: (c) => `public:${clientIP(c)}`,
});
