import type { Context } from 'hono';
import type { Env, Variables } from '../types/index.js';
import type { Database } from '../db/types.js';
import * as z from 'zod';

/**
 * Custom API error class
 */
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public error: string = 'Error'
  ) {
    super(message);
    this.name = 'ApiError';
  }

  static badRequest(message: string): ApiError {
    return new ApiError(400, message, 'Bad Request');
  }

  static unauthorized(message: string = 'Unauthorized'): ApiError {
    return new ApiError(401, message, 'Unauthorized');
  }

  static forbidden(message: string = 'Forbidden'): ApiError {
    return new ApiError(403, message, 'Forbidden');
  }

  static notFound(message: string = 'Not found'): ApiError {
    return new ApiError(404, message, 'Not Found');
  }

  static conflict(message: string): ApiError {
    return new ApiError(409, message, 'Conflict');
  }

  static internal(message: string = 'Internal server error'): ApiError {
    return new ApiError(500, message, 'Internal Server Error');
  }
}

type AppEnv = {
  Bindings: Env;
  Variables: Variables & { db: Database };
};

/**
 * Global error handler
 */
export const errorHandler = (err: Error, c: Context<AppEnv>) => {
  console.error('Error:', err);

  // Handle API errors
  if (err instanceof ApiError) {
    return c.json(
      {
        error: err.error,
        message: err.message,
        status: err.status,
      },
      err.status as 400 | 401 | 403 | 404 | 409 | 500
    );
  }

  // Handle Zod validation errors (zod v4 uses issues instead of errors)
  if ('issues' in err && Array.isArray((err as z.core.$ZodError).issues)) {
    const zodErr = err as z.core.$ZodError;
    const message = zodErr.issues.map((issue: z.core.$ZodIssue) =>
      `${issue.path.join('.')}: ${issue.message}`
    ).join(', ');
    return c.json(
      {
        error: 'Validation Error',
        message,
        status: 400,
      },
      400
    );
  }

  // Handle database errors
  if (err.message?.includes('UNIQUE constraint failed') ||
      err.message?.includes('duplicate key')) {
    return c.json(
      {
        error: 'Conflict',
        message: 'Resource already exists',
        status: 409,
      },
      409
    );
  }

  // Handle unknown errors
  const isDev = c.env?.NODE_ENV === 'development';
  return c.json(
    {
      error: 'Internal Server Error',
      message: isDev ? err.message : 'An unexpected error occurred',
      status: 500,
    },
    500
  );
};
