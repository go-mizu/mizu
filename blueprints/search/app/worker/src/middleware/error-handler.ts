/**
 * Global error handler middleware.
 * Catches all errors and returns structured JSON responses.
 */

import { createMiddleware } from 'hono/factory';
import { isAppError, RateLimitError } from '../errors';

/**
 * Error handler middleware that catches all errors and returns
 * consistent JSON error responses.
 */
export const errorHandler = createMiddleware(async (c, next) => {
  try {
    await next();
  } catch (error) {
    // Log error with context (in production, send to logging service)
    console.error('[Error]', {
      path: c.req.path,
      method: c.req.method,
      error: error instanceof Error ? error.message : String(error),
      stack: error instanceof Error ? error.stack : undefined,
      timestamp: new Date().toISOString(),
    });

    // Handle AppError instances
    if (isAppError(error)) {
      // Add Retry-After header for rate limit errors
      if (error instanceof RateLimitError && error.retryAfter) {
        c.header('Retry-After', String(error.retryAfter));
      }

      return c.json(error.toJSON(), error.statusCode as 400 | 401 | 403 | 404 | 429 | 500 | 502);
    }

    // Handle unknown errors
    // Note: In Cloudflare Workers, we don't have process.env.NODE_ENV
    // You can check c.env.ENVIRONMENT instead if needed
    return c.json(
      {
        error: {
          code: 'INTERNAL_ERROR',
          message: 'An unexpected error occurred',
        },
      },
      500
    );
  }
});

/**
 * Not found handler for unmatched routes.
 * Should be added after all other routes.
 */
export const notFoundHandler = createMiddleware(async (c) => {
  return c.json(
    {
      error: {
        code: 'NOT_FOUND',
        message: `Route not found: ${c.req.method} ${c.req.path}`,
      },
    },
    404
  );
});
