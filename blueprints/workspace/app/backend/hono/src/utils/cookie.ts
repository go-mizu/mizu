import { getCookie, setCookie, deleteCookie } from 'hono/cookie';
import type { AppContext } from './response';

export const SESSION_COOKIE = 'workspace_session';
export const SESSION_DURATION_MS = 7 * 24 * 60 * 60 * 1000; // 7 days

export function getSessionCookie(c: AppContext): string | undefined {
  return getCookie(c, SESSION_COOKIE);
}

export function setSessionCookie(c: AppContext, sessionId: string): void {
  const isSecure =
    c.req.header('x-forwarded-proto') === 'https' ||
    c.req.url.startsWith('https://');

  setCookie(c, SESSION_COOKIE, sessionId, {
    path: '/',
    httpOnly: true,
    secure: isSecure,
    sameSite: 'Lax',
    maxAge: SESSION_DURATION_MS / 1000,
  });
}

export function clearSessionCookie(c: AppContext): void {
  deleteCookie(c, SESSION_COOKIE, { path: '/' });
}
