import { cors } from 'hono/cors';

/**
 * CORS configuration for the API
 */
export const corsMiddleware = cors({
  origin: (origin) => {
    // Allow requests with no origin (mobile apps, curl, etc.)
    if (!origin) return '*';

    // Allow localhost for development
    if (origin.includes('localhost') || origin.includes('127.0.0.1')) {
      return origin;
    }

    // Allow Vercel preview deployments
    if (origin.includes('.vercel.app')) {
      return origin;
    }

    // Allow Cloudflare Pages/Workers
    if (origin.includes('.pages.dev') || origin.includes('.workers.dev')) {
      return origin;
    }

    // Default: allow same origin
    return origin;
  },
  allowMethods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'],
  allowHeaders: ['Content-Type', 'Authorization', 'X-Requested-With'],
  credentials: true,
  maxAge: 86400, // 24 hours
});
