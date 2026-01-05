import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import type { Env, Variables } from '../env';
import { CreateUserSchema, LoginSchema, toPublicUser } from '../models/user';
import { authMiddleware } from '../middleware/auth';
import { authRateLimit } from '../middleware/ratelimit';
import { setSessionCookie, clearSessionCookie } from '../utils/cookie';
import { createServices } from '../services';

export const authRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

// Apply rate limiting to auth endpoints
authRoutes.use('/*', authRateLimit);

// Register
authRoutes.post(
  '/register',
  zValidator('json', CreateUserSchema),
  async (c) => {
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const { user, session } = await services.users.register(input);

    setSessionCookie(c, session.id);

    return c.json({ user: toPublicUser(user) }, 201);
  }
);

// Login
authRoutes.post(
  '/login',
  zValidator('json', LoginSchema),
  async (c) => {
    const input = c.req.valid('json');
    const store = c.get('store');
    const services = createServices(store);

    const { user, session } = await services.users.login(input);

    setSessionCookie(c, session.id);

    return c.json({ user: toPublicUser(user) });
  }
);

// Logout
authRoutes.post('/logout', authMiddleware, async (c) => {
  const store = c.get('store');
  const services = createServices(store);
  const sessionId = c.req.header('cookie')?.match(/workspace_session=([^;]+)/)?.[1];

  if (sessionId) {
    await services.users.logout(sessionId);
  }

  clearSessionCookie(c);

  return c.json({ success: true });
});

// Get current user
authRoutes.get('/me', authMiddleware, async (c) => {
  const user = c.get('user');
  if (!user) {
    return c.json({ error: 'Not authenticated' }, 401);
  }

  return c.json({ user: toPublicUser(user) });
});
