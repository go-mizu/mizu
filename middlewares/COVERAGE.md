# Test Coverage Plan for Mizu Middlewares

## Current Coverage Status

| Package | Coverage | Status |
|---------|----------|--------|
| bodylimit | 100.0% | Complete |
| contenttype | 100.0% | Complete |
| helmet | 100.0% | Complete |
| keepalive | 100.0% | Complete |
| methodoverride | 100.0% | Complete |
| nocache | 100.0% | Complete |
| recover | 100.0% | Complete |
| rewrite | 100.0% | Complete |
| slash | 100.0% | Complete |
| timeout | 100.0% | Complete |
| timing | 100.0% | Complete |
| vary | 100.0% | Complete |
| xrequestedwith | 100.0% | Complete |
| cors | 98.0% | Near Complete |
| hedge | 97.5% | Near Complete |
| keyauth | 97.8% | Near Complete |
| metrics | 97.6% | Near Complete |
| filter | 97.3% | Near Complete |
| lastmodified | 97.4% | Near Complete |
| cors2 | 96.9% | Near Complete |
| timezone | 96.8% | Near Complete |
| version | 96.1% | Near Complete |
| fingerprint | 95.9% | Near Complete |
| requestid | 95.8% | Near Complete |
| bearerauth | 95.7% | Near Complete |
| bodydump | 95.6% | Near Complete |
| ipfilter | 95.6% | Near Complete |
| header | 95.5% | Near Complete |
| etag | 94.6% | Near Complete |
| maintenance | 93.9% | Near Complete |
| redirect | 93.7% | Near Complete |
| bodyclose | 93.3% | Near Complete |
| expvar | 93.3% | Near Complete |
| mock | 93.2% | Near Complete |
| secure | 93.2% | Near Complete |
| realip | 93.0% | Near Complete |
| bot | 92.7% | Near Complete |
| bulkhead | 92.7% | Near Complete |
| cache | 92.7% | Near Complete |
| idempotency | 92.4% | Near Complete |
| spa | 92.3% | Near Complete |
| requestsize | 91.7% | Near Complete |
| responselog | 91.5% | Near Complete |
| responsesize | 91.3% | Near Complete |
| audit | 90.9% | Near Complete |
| conditional | 90.9% | Near Complete |
| graphql | 90.9% | Near Complete |
| retry | 90.7% | Near Complete |
| pprof | 90.5% | Near Complete |
| canary | 90.2% | Near Complete |
| csrf | 89.8% | Near Complete |
| basicauth | 89.7% | Near Complete |
| prometheus | 88.9% | Needs Work |
| circuitbreaker | 88.3% | Needs Work |
| static | 88.6% | Needs Work |
| proxy | 88.2% | Needs Work |
| compress | 88.0% | Needs Work |
| requestlog | 88.0% | Needs Work |
| validator | 87.8% | Needs Work |
| xml | 87.8% | Needs Work |
| transformer | 87.9% | Needs Work |
| ratelimit | 87.5% | Needs Work |
| maxconns | 87.3% | Needs Work |
| throttle | 87.0% | Needs Work |
| forwarded | 87.2% | Needs Work |
| favicon | 86.8% | Needs Work |
| language | 86.5% | Needs Work |
| nonce | 86.2% | Needs Work |
| otel | 85.8% | Needs Work |
| multitenancy | 85.9% | Needs Work |
| feature | 85.5% | Needs Work |
| honeypot | 85.5% | Needs Work |
| trace | 85.2% | Needs Work |
| logger | 85.3% | Needs Work |
| errorpage | 84.8% | Needs Work |
| hypermedia | 83.8% | Needs Work |
| envelope | 83.6% | Needs Work |
| session | 83.5% | Needs Work |
| chaos | 83.3% | Needs Work |
| sanitizer | 82.8% | Needs Work |
| csrf2 | 82.3% | Needs Work |
| surrogate | 81.0% | Needs Work |
| sentry | 79.0% | Needs Work |
| healthcheck | 76.6% | Needs Work |
| rbac | 76.1% | Needs Work |
| jsonrpc | 75.0% | Needs Work |
| oidc | 74.4% | Needs Work |
| concurrency | 73.5% | Needs Work |
| mirror | 72.9% | Needs Work |
| embed | 71.1% | Needs Work |
| msgpack | 70.8% | Needs Work |
| jwt | 67.7% | Needs Work |
| signature | 67.5% | Needs Work |
| oauth2 | 65.2% | Needs Work |
| adaptive | 63.6% | Critical |
| fallback | 56.4% | Critical |
| h2c | 49.0% | Critical |
| captcha | 46.5% | Critical |
| sse | 39.0% | Critical |
| websocket | 29.5% | Critical |

## Coverage Improvement Plan

### Priority 1: Critical (< 60% coverage)

#### websocket (29.5%)
- [ ] Test `ReadMessage` with various payload lengths (126 bytes, 127+ bytes)
- [ ] Test `WriteMessage` with various payload lengths
- [ ] Test `WriteText` and `WriteBinary` helpers
- [ ] Test `Close`, `Ping`, `Pong` methods
- [ ] Test subprotocol selection
- [ ] Test origin checking with specific origins
- [ ] Test forbidden origin response

#### sse (39.0%)
- [ ] Test `Client.Send` with closed connection
- [ ] Test `Client.SendData` and `Client.SendEvent`
- [ ] Test `Client.Close` idempotent behavior
- [ ] Test `Broker.Register` and unregister
- [ ] Test `Broker.Broadcast` with full client buffers
- [ ] Test `Broker.ClientCount`
- [ ] Test event sending with ID, Event, Retry, Data fields

#### captcha (46.5%)
- [ ] Test `New` convenience function
- [ ] Test `verifyToken` with mock HTTP server
- [ ] Test `getClientIP` with X-Forwarded-For and X-Real-IP
- [ ] Test `ReCaptchaV2`, `ReCaptchaV3`, `HCaptcha`, `Turnstile` constructors
- [ ] Test score threshold validation for v3 providers

#### h2c (49.0%)
- [ ] Test `handleUpgrade` with hijackable connection
- [ ] Test `ServerHandler.ServeHTTP` with upgrade request
- [ ] Test `ParseSettings` with valid/invalid base64
- [ ] Test `BufferedConn.Read` and `Peek`
- [ ] Test `IsHTTP2Preface` with valid/invalid prefaces
- [ ] Test `Detect` middleware
- [ ] Test `Wrap` function

#### fallback (56.4%)
- [ ] Test `responseCapture.WriteHeader` and `Write`
- [ ] Test `panicError.Error` with non-error values
- [ ] Test `NotFound` middleware
- [ ] Test `ForStatus` middleware
- [ ] Test `Error` middleware (if exists)

### Priority 2: Medium (60-80% coverage)

#### adaptive (63.6%)
- [ ] Test `adjust` function with various load patterns
- [ ] Test limiter with CPU/memory metrics
- [ ] Test edge cases in `allow` function

#### oauth2 (65.2%)
- [ ] Test token refresh flow
- [ ] Test error handling in OAuth flow
- [ ] Test callback handling

#### jwt (67.7%)
- [ ] Test various JWT algorithms
- [ ] Test token expiration handling
- [ ] Test custom claims extraction

#### signature (67.5%)
- [ ] Test signature generation and verification
- [ ] Test timestamp validation
- [ ] Test various hash algorithms

#### embed (71.1%)
- [ ] Test SPA mode
- [ ] Test `itoa` with negative numbers
- [ ] Test `HandlerWithOptions` with various configurations

#### msgpack (70.8%)
- [ ] Test encoding/decoding edge cases
- [ ] Test with various data types

#### mirror (72.9%)
- [ ] Test mirroring with failed backend
- [ ] Test concurrent mirroring

#### concurrency (73.5%)
- [ ] Test `WithContext` option
- [ ] Test blocking behavior with timeout

#### oidc (74.4%)
- [ ] Test discovery endpoint
- [ ] Test token validation
- [ ] Test user info endpoint

#### jsonrpc (75.0%)
- [ ] Test batch requests
- [ ] Test error responses
- [ ] Test notification handling

#### rbac (76.1%)
- [ ] Test role hierarchy
- [ ] Test permission inheritance
- [ ] Test denied access scenarios

#### healthcheck (76.6%)
- [ ] Test all health check types
- [ ] Test degraded state
- [ ] Test concurrent health checks

### Priority 3: High (80-90% coverage)

Most middlewares in this category need just a few additional tests to reach 100%:

- **sentry (79.0%)**: Test error capturing, breadcrumbs
- **surrogate (81.0%)**: Test cache key variations
- **csrf2 (82.3%)**: Test `validateOrigin`, `GetToken` error case
- **sanitizer (82.8%)**: Test all sanitization rules
- **session (83.5%)**: Test session expiry, concurrent access
- **chaos (83.3%)**: Test `SetLatency`, `SetSelector`, `Controller.Middleware`
- **envelope (83.6%)**: Test `Write` error case
- **hypermedia (83.8%)**: Test link generation edge cases
- **errorpage (84.8%)**: Test `Write` with non-error status
- **logger (85.3%)**: Test all log formats
- **trace (85.2%)**: Test span propagation
- **honeypot (85.5%)**: Test field detection
- **feature (85.5%)**: Test `GetFlags` error case, `RequireAll/RequireAny` paths
- **multitenancy (85.9%)**: Test tenant extraction methods
- **otel (85.8%)**: Test span attributes
- **nonce (86.2%)**: Test nonce generation edge cases
- **language (86.5%)**: Test all language negotiation scenarios
- **favicon (86.8%)**: Test `Empty`, `Redirect`, `SVG` error paths
- **forwarded (87.2%)**: Test all forwarded header variations
- **throttle (87.0%)**: Test rate limiting edge cases
- **maxconns (87.3%)**: Test connection limit reached
- **ratelimit (87.5%)**: Test rate limiter with various windows
- **transformer (87.9%)**: Test all transformation types
- **xml (87.8%)**: Test XML encoding/decoding edge cases
- **validator (87.8%)**: Test all validation rules
- **compress (88.0%)**: Test `Flush`, `Write` error path
- **requestlog (88.0%)**: Test all log fields
- **proxy (88.2%)**: Test proxy error handling
- **circuitbreaker (88.3%)**: Test `GetState` method
- **static (88.6%)**: Test file serving edge cases
- **prometheus (88.9%)**: Test all metric types

### Priority 4: Near Complete (90-99% coverage)

These need minimal additions:

- **basicauth (89.7%)**: Test `WithRealm` with invalid realm
- **csrf (89.8%)**: Test `Token` error case, `TokenExpiry`
- **canary (90.2%)**: Test `RandomSelector`, `CookieSelector` edge case
- **pprof (90.5%)**: Test all pprof endpoints
- **retry (90.7%)**: Test max retries exceeded
- **graphql (90.9%)**: Test query validation
- **conditional (90.9%)**: Test all conditional request types
- **audit (90.9%)**: Test `Write` error case, `defaultHandler`
- **responsesize (91.3%)**: Test size limit exceeded
- **responselog (91.5%)**: Test log truncation
- **requestsize (91.7%)**: Test size validation
- **spa (92.3%)**: Test fallback routing
- **idempotency (92.4%)**: Test key extraction methods
- **bulkhead (92.7%)**: Test queue timeout
- **cache (92.7%)**: Test all cache-control directives
- **bot (92.7%)**: Test `Category`, `AllowSearchEngines`, `AllowSocialBots`, `Get` error case
- **realip (93.0%)**: Test all IP extraction methods
- **mock (93.2%)**: Test all mock scenarios
- **secure (93.2%)**: Test all security headers
- **expvar (93.3%)**: Test `Publish`, `Handler`
- **bodyclose (93.3%)**: Test drain behavior
- **redirect (93.7%)**: Test all redirect types
- **maintenance (93.9%)**: Test schedule-based maintenance
- **etag (94.6%)**: Test `Flush` method
- **header (95.5%)**: Test header manipulation
- **ipfilter (95.6%)**: Test CIDR ranges
- **bodydump (95.6%)**: Test `Write` error case
- **bearerauth (95.7%)**: Test `Token` helper error case
- **requestid (95.8%)**: Test custom generators
- **fingerprint (95.9%)**: Test all fingerprint sources
- **version (96.1%)**: Test version comparison
- **timezone (96.8%)**: Test invalid timezone handling
- **cors2 (96.9%)**: Test `matchOrigin` edge cases
- **lastmodified (97.4%)**: Test time formatting
- **filter (97.3%)**: Test all filter conditions
- **metrics (97.6%)**: Test all metric types
- **keyauth (97.8%)**: Test key extraction
- **hedge (97.5%)**: Test hedge timeout
- **cors (98.0%)**: Test wildcard origins

## Implementation Strategy

1. Start with critical packages (< 60%) as they have the most impact
2. Focus on uncovered functions shown in `go tool cover -func`
3. Use mock HTTP servers for external service tests (captcha, oauth2)
4. Use fake connections for websocket/sse/h2c testing
5. Ensure tests are deterministic and don't rely on timing

## Running Coverage

```bash
# Full coverage report
go test -coverprofile=coverage.out ./middlewares/...
go tool cover -func=coverage.out

# HTML report
go tool cover -html=coverage.out -o coverage.html

# Single package
go test -cover ./middlewares/websocket/
```
