import type { Context, Next } from 'hono';

/**
 * Server-Timing middleware for Hono.
 * Records the total request processing time and adds it as a
 * Server-Timing header to the response.
 *
 * The header format follows the W3C Server-Timing specification:
 *   Server-Timing: total;dur=<ms>;desc="Total request time"
 */
export function serverTiming() {
  return async (c: Context, next: Next) => {
    const start = performance.now();

    await next();

    const duration = performance.now() - start;
    const rounded = Math.round(duration * 100) / 100;

    c.res.headers.set(
      'Server-Timing',
      `total;dur=${rounded};desc="Total request time"`,
    );
  };
}
