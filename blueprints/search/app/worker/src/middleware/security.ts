/**
 * Security headers middleware for Cloudflare Workers.
 * Adds standard security headers to all responses.
 */

import { createMiddleware } from 'hono/factory';

export interface SecurityHeadersConfig {
  /** Enable/disable Content-Security-Policy header (default: true) */
  contentSecurityPolicy?: boolean | string;
  /** Enable/disable X-Frame-Options header (default: true) */
  frameOptions?: boolean | 'DENY' | 'SAMEORIGIN';
  /** Enable/disable X-Content-Type-Options header (default: true) */
  contentTypeOptions?: boolean;
  /** Enable/disable X-XSS-Protection header (default: true) */
  xssProtection?: boolean;
  /** Enable/disable Referrer-Policy header (default: true) */
  referrerPolicy?: boolean | string;
  /** Enable/disable Strict-Transport-Security header (default: true in production) */
  hsts?: boolean | { maxAge?: number; includeSubDomains?: boolean; preload?: boolean };
  /** Enable/disable Permissions-Policy header (default: false) */
  permissionsPolicy?: boolean | string;
}

const DEFAULT_CSP = [
  "default-src 'self'",
  "script-src 'self' 'unsafe-inline'",  // Required for some inline scripts
  "style-src 'self' 'unsafe-inline'",   // Required for inline styles
  "img-src 'self' data: https:",        // Allow images from HTTPS sources
  "font-src 'self' data:",
  "connect-src 'self' https:",          // Allow API calls to HTTPS
  "frame-ancestors 'none'",             // Prevent embedding in frames
  "base-uri 'self'",
  "form-action 'self'",
].join('; ');

const DEFAULT_PERMISSIONS_POLICY = [
  'accelerometer=()',
  'camera=()',
  'geolocation=()',
  'gyroscope=()',
  'magnetometer=()',
  'microphone=()',
  'payment=()',
  'usb=()',
].join(', ');

/**
 * Create security headers middleware with the given configuration.
 *
 * @example
 * ```typescript
 * // Basic usage with defaults
 * app.use('*', securityHeaders());
 *
 * // Custom configuration
 * app.use('*', securityHeaders({
 *   contentSecurityPolicy: "default-src 'self'",
 *   frameOptions: 'SAMEORIGIN',
 * }));
 * ```
 */
export function securityHeaders(config: SecurityHeadersConfig = {}) {
  const {
    contentSecurityPolicy = true,
    frameOptions = true,
    contentTypeOptions = true,
    xssProtection = true,
    referrerPolicy = true,
    hsts = true,
    permissionsPolicy = false,
  } = config;

  return createMiddleware(async (c, next) => {
    await next();

    // X-Content-Type-Options - Prevent MIME type sniffing
    if (contentTypeOptions) {
      c.header('X-Content-Type-Options', 'nosniff');
    }

    // X-Frame-Options - Prevent clickjacking
    if (frameOptions) {
      const value = typeof frameOptions === 'string' ? frameOptions : 'DENY';
      c.header('X-Frame-Options', value);
    }

    // X-XSS-Protection - Enable browser XSS filter
    if (xssProtection) {
      c.header('X-XSS-Protection', '1; mode=block');
    }

    // Referrer-Policy - Control referrer information
    if (referrerPolicy) {
      const value = typeof referrerPolicy === 'string'
        ? referrerPolicy
        : 'strict-origin-when-cross-origin';
      c.header('Referrer-Policy', value);
    }

    // Content-Security-Policy - Prevent XSS and injection attacks
    if (contentSecurityPolicy) {
      const value = typeof contentSecurityPolicy === 'string'
        ? contentSecurityPolicy
        : DEFAULT_CSP;
      c.header('Content-Security-Policy', value);
    }

    // Strict-Transport-Security - Force HTTPS
    if (hsts) {
      let value: string;
      if (typeof hsts === 'object') {
        const maxAge = hsts.maxAge ?? 31536000; // 1 year default
        const parts = [`max-age=${maxAge}`];
        if (hsts.includeSubDomains) parts.push('includeSubDomains');
        if (hsts.preload) parts.push('preload');
        value = parts.join('; ');
      } else {
        value = 'max-age=31536000; includeSubDomains';
      }
      c.header('Strict-Transport-Security', value);
    }

    // Permissions-Policy - Restrict browser features
    if (permissionsPolicy) {
      const value = typeof permissionsPolicy === 'string'
        ? permissionsPolicy
        : DEFAULT_PERMISSIONS_POLICY;
      c.header('Permissions-Policy', value);
    }

    // Additional security headers
    c.header('X-Permitted-Cross-Domain-Policies', 'none');
    c.header('Cross-Origin-Opener-Policy', 'same-origin');
    c.header('Cross-Origin-Resource-Policy', 'same-origin');
  });
}

/**
 * Security headers preset for API-only endpoints (no CSP needed).
 */
export function apiSecurityHeaders() {
  return securityHeaders({
    contentSecurityPolicy: false,  // Not needed for JSON APIs
    frameOptions: true,
    contentTypeOptions: true,
    xssProtection: false,          // Not applicable to JSON
    referrerPolicy: true,
    hsts: true,
    permissionsPolicy: false,
  });
}

/**
 * Relaxed security headers for development.
 */
export function devSecurityHeaders() {
  return securityHeaders({
    contentSecurityPolicy: false,
    frameOptions: false,
    contentTypeOptions: true,
    xssProtection: false,
    referrerPolicy: true,
    hsts: false,
    permissionsPolicy: false,
  });
}
