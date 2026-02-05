/**
 * Input validation middleware for Cloudflare Workers.
 * Provides type-safe validation for query parameters and request bodies.
 *
 * Note: This is a lightweight validation system. For more complex schemas,
 * consider using Zod and replacing the Validator type with ZodSchema.
 */

import { createMiddleware } from 'hono/factory';
import { ValidationError } from '../errors';

// ========== Validator Types ==========

/**
 * Validation result type.
 */
export type ValidationResult<T> =
  | { success: true; data: T }
  | { success: false; errors: Record<string, string[]> };

/**
 * Generic validator interface.
 */
export interface Validator<T> {
  parse(input: unknown): ValidationResult<T>;
}

// ========== Built-in Validators ==========

/**
 * String validator with constraints.
 */
export function string(options: {
  minLength?: number;
  maxLength?: number;
  pattern?: RegExp;
  enum?: readonly string[];
} = {}): Validator<string> {
  return {
    parse(input: unknown): ValidationResult<string> {
      if (typeof input !== 'string') {
        return { success: false, errors: { _: ['Expected string'] } };
      }

      const errors: string[] = [];

      if (options.minLength !== undefined && input.length < options.minLength) {
        errors.push(`Must be at least ${options.minLength} characters`);
      }

      if (options.maxLength !== undefined && input.length > options.maxLength) {
        errors.push(`Must be at most ${options.maxLength} characters`);
      }

      if (options.pattern && !options.pattern.test(input)) {
        errors.push('Invalid format');
      }

      if (options.enum && !options.enum.includes(input)) {
        errors.push(`Must be one of: ${options.enum.join(', ')}`);
      }

      if (errors.length > 0) {
        return { success: false, errors: { _: errors } };
      }

      return { success: true, data: input };
    },
  };
}

/**
 * Number validator with constraints.
 */
export function number(options: {
  min?: number;
  max?: number;
  integer?: boolean;
} = {}): Validator<number> {
  return {
    parse(input: unknown): ValidationResult<number> {
      const num = typeof input === 'string' ? parseFloat(input) : input;

      if (typeof num !== 'number' || isNaN(num)) {
        return { success: false, errors: { _: ['Expected number'] } };
      }

      const errors: string[] = [];

      if (options.integer && !Number.isInteger(num)) {
        errors.push('Must be an integer');
      }

      if (options.min !== undefined && num < options.min) {
        errors.push(`Must be at least ${options.min}`);
      }

      if (options.max !== undefined && num > options.max) {
        errors.push(`Must be at most ${options.max}`);
      }

      if (errors.length > 0) {
        return { success: false, errors: { _: errors } };
      }

      return { success: true, data: num };
    },
  };
}

/**
 * Boolean validator.
 */
export function boolean(): Validator<boolean> {
  return {
    parse(input: unknown): ValidationResult<boolean> {
      if (typeof input === 'boolean') {
        return { success: true, data: input };
      }

      if (input === 'true' || input === '1') {
        return { success: true, data: true };
      }

      if (input === 'false' || input === '0' || input === '') {
        return { success: true, data: false };
      }

      return { success: false, errors: { _: ['Expected boolean'] } };
    },
  };
}

/**
 * Optional wrapper for validators.
 */
export function optional<T>(validator: Validator<T>): Validator<T | undefined> {
  return {
    parse(input: unknown): ValidationResult<T | undefined> {
      if (input === undefined || input === null || input === '') {
        return { success: true, data: undefined };
      }
      return validator.parse(input);
    },
  };
}

/**
 * Create an enum validator from const array.
 */
export function enumValidator<T extends readonly string[]>(
  values: T
): Validator<T[number]> {
  return string({ enum: values }) as Validator<T[number]>;
}

// ========== Object Schema ==========

type SchemaDefinition = Record<string, Validator<unknown>>;

type InferSchema<T extends SchemaDefinition> = {
  [K in keyof T]: T[K] extends Validator<infer U> ? U : never;
};

/**
 * Create an object validator from a schema definition.
 */
export function object<T extends SchemaDefinition>(
  schema: T
): Validator<InferSchema<T>> {
  return {
    parse(input: unknown): ValidationResult<InferSchema<T>> {
      if (typeof input !== 'object' || input === null) {
        return { success: false, errors: { _: ['Expected object'] } };
      }

      const data: Record<string, unknown> = {};
      const errors: Record<string, string[]> = {};

      for (const [key, validator] of Object.entries(schema)) {
        const value = (input as Record<string, unknown>)[key];
        const result = validator.parse(value);

        if (result.success) {
          data[key] = result.data;
        } else {
          errors[key] = result.errors._ ?? ['Invalid value'];
        }
      }

      if (Object.keys(errors).length > 0) {
        return { success: false, errors };
      }

      return { success: true, data: data as InferSchema<T> };
    },
  };
}

// ========== Validation Middleware ==========

/**
 * Create a middleware that validates query parameters.
 *
 * @example
 * ```typescript
 * const SearchQuerySchema = object({
 *   q: string({ minLength: 1, maxLength: 500 }),
 *   page: optional(number({ min: 1, max: 100, integer: true })),
 *   per_page: optional(number({ min: 1, max: 100, integer: true })),
 * });
 *
 * app.get('/search', validateQuery(SearchQuerySchema), async (c) => {
 *   const { q, page, per_page } = c.get('validatedQuery');
 *   // ...
 * });
 * ```
 */
export function validateQuery<T>(validator: Validator<T>) {
  return createMiddleware<{ Variables: { validatedQuery: T } }>(
    async (c, next) => {
      const url = new URL(c.req.url);
      const params: Record<string, string> = {};

      for (const [key, value] of url.searchParams.entries()) {
        params[key] = value;
      }

      const result = validator.parse(params);

      if (!result.success) {
        throw new ValidationError('Invalid query parameters', {
          errors: result.errors,
        });
      }

      c.set('validatedQuery', result.data);
      return next();
    }
  );
}

/**
 * Create a middleware that validates JSON request body.
 *
 * @example
 * ```typescript
 * const CreateBangSchema = object({
 *   trigger: string({ minLength: 1, maxLength: 10 }),
 *   name: string({ minLength: 1, maxLength: 50 }),
 *   url_template: string({ pattern: /\{query\}/ }),
 * });
 *
 * app.post('/bangs', validateBody(CreateBangSchema), async (c) => {
 *   const data = c.get('validatedBody');
 *   // ...
 * });
 * ```
 */
export function validateBody<T>(validator: Validator<T>) {
  return createMiddleware<{ Variables: { validatedBody: T } }>(
    async (c, next) => {
      let body: unknown;

      try {
        body = await c.req.json();
      } catch {
        throw new ValidationError('Invalid JSON body');
      }

      const result = validator.parse(body);

      if (!result.success) {
        throw new ValidationError('Invalid request body', {
          errors: result.errors,
        });
      }

      c.set('validatedBody', result.data);
      return next();
    }
  );
}

// ========== Pre-built Schemas for Common Use Cases ==========

/**
 * Common filter values for image search.
 */
export const IMAGE_SIZES = ['any', 'large', 'medium', 'small', 'icon'] as const;
export const IMAGE_COLORS = [
  'any', 'color', 'gray', 'transparent',
  'red', 'orange', 'yellow', 'green', 'teal',
  'blue', 'purple', 'pink', 'white', 'black', 'brown',
] as const;
export const IMAGE_TYPES = ['any', 'face', 'photo', 'clipart', 'lineart', 'animated'] as const;
export const IMAGE_ASPECTS = ['any', 'tall', 'square', 'wide', 'panoramic'] as const;
export const TIME_RANGES = ['any', 'hour', 'day', 'week', 'month', 'year'] as const;
export const SAFE_SEARCH_LEVELS = ['off', 'moderate', 'strict'] as const;

/**
 * Base search query fields (reusable in composed schemas).
 */
const baseSearchFields = {
  q: string({ minLength: 1, maxLength: 500 }),
  page: optional(number({ min: 1, max: 100, integer: true })),
  per_page: optional(number({ min: 1, max: 100, integer: true })),
  time: optional(enumValidator(TIME_RANGES)),
  region: optional(string({ maxLength: 10 })),
  lang: optional(string({ maxLength: 10 })),
  safe: optional(enumValidator(SAFE_SEARCH_LEVELS)),
};

/**
 * Image search specific fields.
 */
const imageSearchFields = {
  ...baseSearchFields,
  size: optional(enumValidator(IMAGE_SIZES)),
  color: optional(enumValidator(IMAGE_COLORS)),
  type: optional(enumValidator(IMAGE_TYPES)),
  aspect: optional(enumValidator(IMAGE_ASPECTS)),
};

/**
 * Schema for basic search query parameters.
 */
export const SearchQuerySchema = object(baseSearchFields);

/**
 * Schema for image search query parameters.
 */
export const ImageSearchQuerySchema = object(imageSearchFields);

export type SearchQuery = InferSchema<typeof baseSearchFields>;
export type ImageSearchQuery = InferSchema<typeof imageSearchFields>;
