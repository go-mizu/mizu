import type { Context, ErrorHandler as HonoErrorHandler } from 'hono';
import type { Env, Variables } from '../types/index.js';
import type { Database } from '../db/types.js';

type AppVariables = Variables & { db: Database };

/**
 * API Error class with HTTP status
 */
export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.status = status;
    this.code = code;
    this.name = 'ApiError';
  }

  static badRequest(message: string, code?: string): ApiError {
    return new ApiError(400, message, code || 'BAD_REQUEST');
  }

  static unauthorized(message: string = 'Unauthorized'): ApiError {
    return new ApiError(401, message, 'UNAUTHORIZED');
  }

  static forbidden(message: string = 'Forbidden'): ApiError {
    return new ApiError(403, message, 'FORBIDDEN');
  }

  static notFound(message: string = 'Not found'): ApiError {
    return new ApiError(404, message, 'NOT_FOUND');
  }

  static conflict(message: string): ApiError {
    return new ApiError(409, message, 'CONFLICT');
  }

  static internal(message: string = 'Internal server error'): ApiError {
    return new ApiError(500, message, 'INTERNAL_ERROR');
  }
}

/**
 * Global error handler
 */
export const errorHandler: HonoErrorHandler<{ Bindings: Env; Variables: AppVariables }> = (
  err: Error | ApiError,
  c: Context<{ Bindings: Env; Variables: AppVariables }>
) => {
  console.error('Error:', err);

  // Handle ApiError
  if (err instanceof ApiError) {
    return c.json(
      {
        error: err.code || 'ERROR',
        message: err.message,
        status: err.status,
      },
      err.status as 400 | 401 | 403 | 404 | 409 | 500
    );
  }

  // Handle Zod validation errors
  if (err.name === 'ZodError') {
    return c.json(
      {
        error: 'VALIDATION_ERROR',
        message: 'Invalid request data',
        status: 400,
        details: JSON.parse(err.message),
      },
      400
    );
  }

  // Handle unknown errors
  return c.json(
    {
      error: 'INTERNAL_ERROR',
      message: process.env.NODE_ENV === 'production' ? 'Internal server error' : err.message,
      status: 500,
    },
    500
  );
};
