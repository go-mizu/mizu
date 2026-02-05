import type { Context, Next } from 'hono';

const ALLOWED_METHODS = 'GET, POST, PUT, DELETE, PATCH, OPTIONS';
const ALLOWED_HEADERS = 'Content-Type, Authorization, X-Requested-With, Accept';
const MAX_AGE = '86400'; // 24 hours

/**
 * CORS middleware for Hono.
 * Allows all origins in development, and handles OPTIONS preflight requests.
 */
export function cors() {
  return async (c: Context, next: Next) => {
    const origin = c.req.header('Origin') ?? '*';

    // Handle preflight OPTIONS requests
    if (c.req.method === 'OPTIONS') {
      return new Response(null, {
        status: 204,
        headers: {
          'Access-Control-Allow-Origin': origin,
          'Access-Control-Allow-Methods': ALLOWED_METHODS,
          'Access-Control-Allow-Headers': ALLOWED_HEADERS,
          'Access-Control-Max-Age': MAX_AGE,
          'Access-Control-Allow-Credentials': 'true',
        },
      });
    }

    // Process the request
    await next();

    // Add CORS headers to the response
    c.res.headers.set('Access-Control-Allow-Origin', origin);
    c.res.headers.set('Access-Control-Allow-Methods', ALLOWED_METHODS);
    c.res.headers.set('Access-Control-Allow-Headers', ALLOWED_HEADERS);
    c.res.headers.set('Access-Control-Allow-Credentials', 'true');
  };
}
