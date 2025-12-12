# Middleware Documentation Plan

This document tracks documentation status for all 100 Mizu middlewares.

## Documentation Structure

Each middleware documentation file follows this template:

```
---
title: "Middleware Name"
description: "Brief description of what the middleware does."
---

## Overview
Brief explanation of the middleware purpose and when to use it.

## Installation
Import statement for the middleware.

## Quick Start
Minimal example to get started.

## Configuration
All available options with descriptions and defaults.

## Examples
- Basic usage
- Common configurations
- Advanced patterns

## API Reference
- Functions and their signatures
- Types and structs
- Helper functions

## Best Practices
Tips for effective use.

## Related Middlewares
Links to related middlewares.
```

## Middleware Status (38/100 documented)

### Authentication (8 middlewares)
- [x] basicauth - HTTP Basic Authentication
- [x] bearerauth - Bearer token authentication
- [x] keyauth - API key authentication
- [x] csrf - Cross-Site Request Forgery protection
- [ ] csrf2 - Enhanced CSRF protection
- [ ] jwt - JWT authentication
- [ ] oauth2 - OAuth 2.0 authentication
- [ ] oidc - OpenID Connect authentication

### Security (9 middlewares)
- [x] helmet - Security headers
- [x] secure - HTTPS enforcement
- [x] ipfilter - IP whitelist/blacklist
- [x] honeypot - Honeypot middleware
- [x] captcha - CAPTCHA verification
- [ ] cors - CORS handling
- [ ] cors2 - Enhanced CORS
- [ ] rbac - Role-based access control
- [ ] signature - Request signature verification

### Rate Limiting & Resilience (6 middlewares)
- [x] ratelimit - Token bucket rate limiting
- [x] circuitbreaker - Circuit breaker pattern
- [ ] bulkhead - Bulkhead pattern
- [ ] throttle - Request throttling
- [ ] concurrency - Concurrency limiting
- [ ] adaptive - Adaptive rate limiting

### Request Processing (11 middlewares)
- [x] bodylimit - Request body size limiting
- [x] contenttype - Content-Type validation
- [x] validator - Request validation
- [x] header - Header manipulation
- [x] methodoverride - HTTP method override
- [ ] bodyclose - Auto body close
- [ ] bodydump - Dump request/response bodies
- [ ] requestsize - Request size tracking
- [ ] sanitizer - Input sanitization
- [ ] transformer - Request transformation
- [ ] filter - Request filtering

### Response Processing (6 middlewares)
- [ ] compress - Response compression
- [ ] envelope - Response envelope wrapper
- [ ] responsesize - Response size tracking
- [ ] vary - Vary header handling
- [ ] errorpage - Custom error pages
- [ ] hypermedia - Hypermedia response helpers

### Caching (4 middlewares)
- [x] cache - Cache-Control headers
- [x] nocache - Prevent caching
- [ ] etag - ETag generation
- [ ] lastmodified - Last-Modified headers

### URL Handling (3 middlewares)
- [x] redirect - URL redirection
- [x] slash - Trailing slash handling
- [x] rewrite - URL rewriting

### Networking & Proxy (5 middlewares)
- [x] proxy - Reverse proxy
- [x] forwarded - X-Forwarded-* headers
- [x] realip - Real client IP extraction
- [ ] h2c - HTTP/2 cleartext
- [ ] surrogate - Surrogate headers (CDN)

### Request Context (6 middlewares)
- [x] requestid - Request ID generation
- [x] timeout - Request timeout
- [x] recover - Panic recovery
- [x] timing - Server-Timing header
- [ ] trace - Distributed tracing
- [ ] conditional - Conditional middleware

### Real-time (2 middlewares)
- [x] websocket - WebSocket connections
- [x] sse - Server-Sent Events

### Static Files (4 middlewares)
- [ ] static - Static file serving
- [ ] spa - Single Page Application
- [ ] favicon - Favicon serving
- [ ] embed - Embedded filesystem

### Operations & Monitoring (10 middlewares)
- [x] version - API versioning
- [x] maintenance - Maintenance mode
- [x] pprof - Profiling endpoints
- [ ] healthcheck - Health check endpoints
- [ ] metrics - Custom metrics
- [ ] prometheus - Prometheus metrics
- [ ] expvar - Expvar endpoint
- [ ] logger - Request logging
- [ ] requestlog - Request logging (detailed)
- [ ] responselog - Response logging

### Advanced (12 middlewares)
- [x] feature - Feature flags
- [x] multitenancy - Multi-tenant support
- [x] chaos - Chaos engineering
- [x] mirror - Request mirroring
- [x] fingerprint - Request fingerprinting
- [ ] canary - Canary deployments
- [ ] audit - Audit logging
- [ ] idempotency - Idempotency keys
- [ ] retry - Automatic retries
- [ ] hedge - Hedge requests
- [ ] fallback - Fallback handlers
- [ ] mock - Request mocking

### Connection & Protocol (6 middlewares)
- [ ] keepalive - Connection keep-alive
- [ ] maxconns - Max connections
- [ ] msgpack - MessagePack handling
- [ ] jsonrpc - JSON-RPC handling
- [ ] graphql - GraphQL middleware
- [ ] xml - XML handling

### Internationalization (3 middlewares)
- [ ] language - Language detection
- [ ] timezone - Timezone detection
- [ ] nonce - Nonce generation

### External Integrations (3 middlewares)
- [ ] otel - OpenTelemetry
- [ ] sentry - Sentry error tracking
- [ ] session - Session management

### Bot & Client Detection (2 middlewares)
- [ ] bot - Bot detection
- [ ] xrequestedwith - X-Requested-With validation

## Summary

| Status | Count |
|--------|-------|
| Documented | 38 |
| Missing | 62 |
| **Total** | **100** |

## Quality Checklist

- [ ] Clear title and description
- [ ] Working code examples
- [ ] All options documented
- [ ] Default values specified
- [ ] Error handling explained
- [ ] Best practices included
- [ ] Related middlewares linked
- [ ] Spell-checked and formatted
