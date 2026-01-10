import { createMiddleware } from 'hono/factory';
import * as jose from 'jose';
import type { Env, Variables, UserPublic } from '../types/index.js';
import type { Database } from '../db/types.js';

/**
 * JWT payload structure
 */
interface JwtPayload {
  sub: string;
  session_id: string;
  exp: number;
}

/**
 * Create JWT token for user session
 */
export async function createToken(
  userId: string,
  sessionId: string,
  secret: string,
  expiresIn: string = '7d'
): Promise<string> {
  const secretKey = new TextEncoder().encode(secret);

  const token = await new jose.SignJWT({
    sub: userId,
    session_id: sessionId,
  })
    .setProtectedHeader({ alg: 'HS256' })
    .setIssuedAt()
    .setExpirationTime(expiresIn)
    .sign(secretKey);

  return token;
}

/**
 * Verify and decode JWT token
 */
export async function verifyToken(
  token: string,
  secret: string
): Promise<JwtPayload | null> {
  try {
    const secretKey = new TextEncoder().encode(secret);
    const { payload } = await jose.jwtVerify(token, secretKey);
    return payload as unknown as JwtPayload;
  } catch {
    return null;
  }
}

/**
 * Extract token from Authorization header or cookie
 */
function extractToken(request: Request): string | null {
  // Try Authorization header first
  const authHeader = request.headers.get('Authorization');
  if (authHeader?.startsWith('Bearer ')) {
    return authHeader.slice(7);
  }

  // Try cookie
  const cookies = request.headers.get('Cookie');
  if (cookies) {
    const tokenCookie = cookies
      .split(';')
      .map(c => c.trim())
      .find(c => c.startsWith('token='));
    if (tokenCookie) {
      return tokenCookie.slice(6);
    }
  }

  return null;
}

/**
 * Authentication middleware - requires valid JWT
 */
export const authRequired = createMiddleware<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>(async (c, next) => {
  const token = extractToken(c.req.raw);

  if (!token) {
    return c.json(
      { error: 'Unauthorized', message: 'No token provided', status: 401 },
      401
    );
  }

  const jwtSecret = c.env.JWT_SECRET;
  if (!jwtSecret) {
    console.error('JWT_SECRET not configured');
    return c.json(
      { error: 'Internal Server Error', message: 'Server configuration error', status: 500 },
      500
    );
  }

  const payload = await verifyToken(token, jwtSecret);
  if (!payload) {
    return c.json(
      { error: 'Unauthorized', message: 'Invalid or expired token', status: 401 },
      401
    );
  }

  // Verify session exists in database
  const db = c.get('db');
  const session = await db.getSessionById(payload.session_id);

  if (!session) {
    return c.json(
      { error: 'Unauthorized', message: 'Session not found or expired', status: 401 },
      401
    );
  }

  // Get user
  const user = await db.getUserById(session.user_id);
  if (!user) {
    return c.json(
      { error: 'Unauthorized', message: 'User not found', status: 401 },
      401
    );
  }

  // Set user in context (without password_hash)
  const userPublic: UserPublic = {
    id: user.id,
    email: user.email,
    name: user.name,
    avatar_url: user.avatar_url,
    created_at: user.created_at,
  };

  c.set('user', userPublic);
  c.set('session', session);

  await next();
});

/**
 * Optional authentication - populates user if token present, but doesn't require it
 */
export const authOptional = createMiddleware<{
  Bindings: Env;
  Variables: Variables & { db: Database };
}>(async (c, next) => {
  const token = extractToken(c.req.raw);

  if (token) {
    const jwtSecret = c.env.JWT_SECRET;
    if (jwtSecret) {
      const payload = await verifyToken(token, jwtSecret);
      if (payload) {
        const db = c.get('db');
        const session = await db.getSessionById(payload.session_id);
        if (session) {
          const user = await db.getUserById(session.user_id);
          if (user) {
            c.set('user', {
              id: user.id,
              email: user.email,
              name: user.name,
              avatar_url: user.avatar_url,
              created_at: user.created_at,
            });
            c.set('session', session);
          }
        }
      }
    }
  }

  await next();
});
