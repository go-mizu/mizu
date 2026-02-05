/**
 * Session middleware for anonymous user identification.
 * Uses cookies to maintain session ID without requiring login.
 */

import type { Context, Next } from 'hono';
import { getCookie, setCookie } from 'hono/cookie';

const SESSION_COOKIE = 'mizu_session';
const SESSION_MAX_AGE = 60 * 60 * 24 * 365; // 1 year in seconds

/**
 * Generate a UUID v4 for session identification.
 */
function generateSessionId(): string {
  return crypto.randomUUID();
}

/**
 * Session middleware that ensures every request has a session ID.
 * Sets a cookie if none exists, refreshes expiry on each visit.
 */
export function sessionMiddleware() {
  return async (c: Context, next: Next) => {
    let sessionId = getCookie(c, SESSION_COOKIE);

    if (!sessionId) {
      sessionId = generateSessionId();
    }

    // Store session ID in context for easy access
    c.set('sessionId', sessionId);

    // Continue processing
    await next();

    // Set/refresh cookie after response
    setCookie(c, SESSION_COOKIE, sessionId, {
      path: '/',
      maxAge: SESSION_MAX_AGE,
      httpOnly: true,
      secure: true,
      sameSite: 'Lax',
    });
  };
}

/**
 * Helper to get session ID from context.
 */
export function getSessionId(c: Context): string {
  return c.get('sessionId') || '';
}
