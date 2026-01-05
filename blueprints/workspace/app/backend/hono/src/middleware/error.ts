import type { ErrorHandler } from 'hono';
import type { Env, Variables } from '../env';
import { ZodError } from 'zod';

export const errorHandler: ErrorHandler<{ Bindings: Env; Variables: Variables }> = (
  err,
  c
) => {
  console.error('Error:', err);

  // Zod validation errors
  if (err instanceof ZodError) {
    const errors = err.errors.map((e) => ({
      path: e.path.join('.'),
      message: e.message,
    }));
    return c.json({ error: 'Validation error', details: errors }, 400);
  }

  // Known errors with messages
  if (err instanceof Error) {
    const message = err.message;

    // Auth errors
    if (message.includes('Unauthorized') || message.includes('Session')) {
      return c.json({ error: message }, 401);
    }

    // Permission errors
    if (message.includes('Permission denied')) {
      return c.json({ error: message }, 403);
    }

    // Not found errors
    if (message.includes('not found')) {
      return c.json({ error: message }, 404);
    }

    // Conflict errors
    if (message.includes('already') || message.includes('exists')) {
      return c.json({ error: message }, 409);
    }

    // Rate limit errors
    if (message.includes('Too many')) {
      return c.json({ error: message }, 429);
    }

    // Validation errors
    if (message.includes('Invalid') || message.includes('required')) {
      return c.json({ error: message }, 400);
    }
  }

  // Default to 500
  return c.json({ error: 'Internal server error' }, 500);
};
