import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import { hash, compare } from 'bcryptjs';
import { ulid } from 'ulid';
import type { Env, Variables } from '../types/index.js';
import { CreateUserSchema, LoginSchema } from '../types/index.js';
import type { Database } from '../db/types.js';
import { createToken } from '../middleware/auth.js';
import { ApiError } from '../middleware/error.js';
import { authRequired } from '../middleware/auth.js';

const auth = new Hono<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>();

/**
 * POST /auth/register - Register new user
 */
auth.post('/register', zValidator('json', CreateUserSchema), async (c) => {
  const { email, name, password } = c.req.valid('json');
  const db = c.get('db');

  // Check if user exists
  const existing = await db.getUserByEmail(email);
  if (existing) {
    throw ApiError.conflict('User with this email already exists');
  }

  // Hash password
  const passwordHash = await hash(password, 12);

  // Create user
  const userId = ulid();
  const user = await db.createUser({
    id: userId,
    email,
    name,
    password: password, // This is used by the type but not the actual insert
    password_hash: passwordHash,
  });

  // Create session
  const sessionId = ulid();
  const expiresAt = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();
  const token = await createToken(user.id, sessionId, c.env.JWT_SECRET);

  await db.createSession({
    id: sessionId,
    user_id: user.id,
    token,
    expires_at: expiresAt,
  });

  return c.json({
    token,
    user: {
      id: user.id,
      email: user.email,
      name: user.name,
      created_at: user.created_at,
    },
  }, 201);
});

/**
 * POST /auth/login - Login user
 */
auth.post('/login', zValidator('json', LoginSchema), async (c) => {
  const { email, password } = c.req.valid('json');
  const db = c.get('db');

  // Find user
  const user = await db.getUserByEmail(email);
  if (!user) {
    throw ApiError.unauthorized('Invalid email or password');
  }

  // Verify password
  const valid = await compare(password, user.password_hash);
  if (!valid) {
    throw ApiError.unauthorized('Invalid email or password');
  }

  // Create session
  const sessionId = ulid();
  const expiresAt = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();
  const token = await createToken(user.id, sessionId, c.env.JWT_SECRET);

  await db.createSession({
    id: sessionId,
    user_id: user.id,
    token,
    expires_at: expiresAt,
  });

  return c.json({
    token,
    user: {
      id: user.id,
      email: user.email,
      name: user.name,
      created_at: user.created_at,
    },
  });
});

/**
 * POST /auth/logout - Logout user
 */
auth.post('/logout', authRequired, async (c) => {
  const session = c.get('session');
  const db = c.get('db');

  await db.deleteSession(session.token);

  return c.json({ message: 'Logged out successfully' });
});

/**
 * GET /auth/me - Get current user
 */
auth.get('/me', authRequired, async (c) => {
  const user = c.get('user');
  return c.json({ user });
});

export { auth };
