import type { Context } from 'hono';
import type { Env, Variables } from '../env';

export type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

export function success<T>(c: AppContext, data: T, status: 200 | 201 = 200) {
  return c.json(data, status);
}

export function created<T>(c: AppContext, data: T) {
  return c.json(data, 201);
}

export function noContent(c: AppContext) {
  return c.body(null, 204);
}

export function badRequest(c: AppContext, message: string) {
  return c.json({ error: message }, 400);
}

export function unauthorized(c: AppContext, message = 'Unauthorized') {
  return c.json({ error: message }, 401);
}

export function forbidden(c: AppContext, message = 'Forbidden') {
  return c.json({ error: message }, 403);
}

export function notFound(c: AppContext, message = 'Not found') {
  return c.json({ error: message }, 404);
}

export function conflict(c: AppContext, message: string) {
  return c.json({ error: message }, 409);
}

export function tooManyRequests(c: AppContext, message = 'Too many requests') {
  return c.json({ error: message }, 429);
}

export function serverError(c: AppContext, message = 'Internal server error') {
  return c.json({ error: message }, 500);
}
