import { createMiddleware } from 'hono/factory';
import type { Env, Variables } from '../env';
import { getSessionCookie, clearSessionCookie } from '../utils/cookie';

export const authMiddleware = createMiddleware<{
  Bindings: Env;
  Variables: Variables;
}>(async (c, next) => {
  const store = c.get('store');
  if (!store) {
    return c.json({ error: 'Store not initialized' }, 500);
  }

  // Check for dev mode
  const devMode = c.env?.DEV_MODE === 'true' || c.env?.ENVIRONMENT === 'development';
  if (devMode) {
    // Use dev user
    const devUser = await store.users.getById('dev-user-001');
    if (devUser) {
      c.set('user', devUser);
      c.set('userId', devUser.id);
      return next();
    }
  }

  const sessionId = getSessionCookie(c);
  if (!sessionId) {
    return c.json({ error: 'Unauthorized' }, 401);
  }

  const session = await store.sessions.getById(sessionId);
  if (!session) {
    clearSessionCookie(c);
    return c.json({ error: 'Session not found' }, 401);
  }

  // Check if session is expired
  if (new Date(session.expiresAt) < new Date()) {
    await store.sessions.deleteById(sessionId);
    clearSessionCookie(c);
    return c.json({ error: 'Session expired' }, 401);
  }

  const user = await store.users.getById(session.userId);
  if (!user) {
    await store.sessions.deleteById(sessionId);
    clearSessionCookie(c);
    return c.json({ error: 'User not found' }, 401);
  }

  c.set('user', user);
  c.set('userId', user.id);

  await next();
});

export const optionalAuthMiddleware = createMiddleware<{
  Bindings: Env;
  Variables: Variables;
}>(async (c, next) => {
  const store = c.get('store');
  if (!store) {
    return next();
  }

  const sessionId = getSessionCookie(c);
  if (!sessionId) {
    c.set('user', null);
    c.set('userId', null);
    return next();
  }

  const session = await store.sessions.getById(sessionId);
  if (!session || new Date(session.expiresAt) < new Date()) {
    c.set('user', null);
    c.set('userId', null);
    return next();
  }

  const user = await store.users.getById(session.userId);
  c.set('user', user);
  c.set('userId', user?.id ?? null);

  await next();
});
