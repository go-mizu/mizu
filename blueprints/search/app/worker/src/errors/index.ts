/**
 * Structured error types for the Mizu Search Worker.
 * All errors extend AppError and include a code, statusCode, and optional context.
 */

/**
 * Base error class for all application errors.
 * Provides consistent error structure for API responses.
 */
export class AppError extends Error {
  readonly code: string;
  readonly statusCode: number;
  readonly context?: Record<string, unknown>;

  constructor(
    message: string,
    code: string,
    statusCode: number = 500,
    context?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'AppError';
    this.code = code;
    this.statusCode = statusCode;
    this.context = context;

    // Maintain proper prototype chain
    Object.setPrototypeOf(this, AppError.prototype);
  }

  /**
   * Convert error to JSON-serializable object for API responses.
   * Note: Context is always included for debugging. In production, you may want to filter sensitive data.
   */
  toJSON(): { error: { code: string; message: string; context?: Record<string, unknown> } } {
    return {
      error: {
        code: this.code,
        message: this.message,
        ...(this.context && { context: this.context }),
      },
    };
  }
}

/**
 * Validation error for invalid input parameters.
 */
export class ValidationError extends AppError {
  constructor(message: string, context?: Record<string, unknown>) {
    super(message, 'VALIDATION_ERROR', 400, context);
    this.name = 'ValidationError';
    Object.setPrototypeOf(this, ValidationError.prototype);
  }
}

/**
 * Not found error for missing resources.
 */
export class NotFoundError extends AppError {
  constructor(resource: string, identifier?: string) {
    const message = identifier
      ? `${resource} not found: ${identifier}`
      : `${resource} not found`;
    super(message, 'NOT_FOUND', 404, { resource, identifier });
    this.name = 'NotFoundError';
    Object.setPrototypeOf(this, NotFoundError.prototype);
  }
}

/**
 * Engine error for search engine failures.
 */
export class EngineError extends AppError {
  readonly engineName: string;
  readonly cause?: Error;

  constructor(engineName: string, cause?: Error, message?: string) {
    const msg = message ?? `Engine ${engineName} failed${cause ? `: ${cause.message}` : ''}`;
    super(msg, 'ENGINE_ERROR', 502, { engine: engineName });
    this.name = 'EngineError';
    this.engineName = engineName;
    this.cause = cause;
    Object.setPrototypeOf(this, EngineError.prototype);
  }
}

/**
 * Rate limit error for too many requests.
 */
export class RateLimitError extends AppError {
  readonly retryAfter?: number;

  constructor(retryAfter?: number) {
    super('Rate limit exceeded', 'RATE_LIMIT', 429, { retryAfter });
    this.name = 'RateLimitError';
    this.retryAfter = retryAfter;
    Object.setPrototypeOf(this, RateLimitError.prototype);
  }
}

/**
 * Cache error for KV/cache operation failures.
 */
export class CacheError extends AppError {
  readonly operation: string;

  constructor(operation: string, cause?: Error) {
    const message = cause
      ? `Cache ${operation} failed: ${cause.message}`
      : `Cache ${operation} failed`;
    super(message, 'CACHE_ERROR', 500, { operation });
    this.name = 'CacheError';
    this.operation = operation;
    Object.setPrototypeOf(this, CacheError.prototype);
  }
}

/**
 * External API error for third-party service failures.
 */
export class ExternalApiError extends AppError {
  readonly serviceName: string;
  readonly httpStatus?: number;

  constructor(serviceName: string, httpStatus?: number, message?: string) {
    const msg = message ?? `External API ${serviceName} failed${httpStatus ? ` with status ${httpStatus}` : ''}`;
    super(msg, 'EXTERNAL_API_ERROR', 502, { service: serviceName, httpStatus });
    this.name = 'ExternalApiError';
    this.serviceName = serviceName;
    this.httpStatus = httpStatus;
    Object.setPrototypeOf(this, ExternalApiError.prototype);
  }
}

/**
 * Configuration error for missing or invalid configuration.
 */
export class ConfigError extends AppError {
  constructor(message: string, missingKey?: string) {
    super(message, 'CONFIG_ERROR', 500, { missingKey });
    this.name = 'ConfigError';
    Object.setPrototypeOf(this, ConfigError.prototype);
  }
}

/**
 * Bang error for invalid bang operations.
 */
export class BangError extends AppError {
  readonly trigger: string;

  constructor(trigger: string, message: string) {
    super(message, 'BANG_ERROR', 400, { trigger });
    this.name = 'BangError';
    this.trigger = trigger;
    Object.setPrototypeOf(this, BangError.prototype);
  }
}

/**
 * Type guard to check if an error is an AppError.
 */
export function isAppError(error: unknown): error is AppError {
  return error instanceof AppError;
}

/**
 * Wrap an unknown error into an AppError.
 */
export function wrapError(error: unknown, defaultMessage = 'An unexpected error occurred'): AppError {
  if (error instanceof AppError) {
    return error;
  }

  if (error instanceof Error) {
    return new AppError(error.message, 'INTERNAL_ERROR', 500, {
      originalName: error.name,
    });
  }

  return new AppError(
    typeof error === 'string' ? error : defaultMessage,
    'INTERNAL_ERROR',
    500
  );
}
