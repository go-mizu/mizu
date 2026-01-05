import { cors } from 'hono/cors';

export const corsMiddleware = cors({
  origin: (origin) => {
    // Allow all origins in development
    if (!origin) return '*';

    // Allow localhost in development
    if (origin.includes('localhost') || origin.includes('127.0.0.1')) {
      return origin;
    }

    // In production, you might want to restrict this
    return origin;
  },
  credentials: true,
  allowMethods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'],
  allowHeaders: ['Content-Type', 'Authorization', 'X-Requested-With'],
  exposeHeaders: ['X-RateLimit-Limit', 'X-RateLimit-Remaining', 'X-RateLimit-Reset'],
  maxAge: 86400, // 24 hours
});
