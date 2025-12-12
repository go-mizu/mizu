# Mizu Middlewares Specification

This document specifies 100 middlewares for the Mizu web framework, organized into logical sub-packages following Go standard library naming conventions.

## Design Principles

1. **Go Standard Library Style**: All names follow `net/http` and standard library conventions
2. **Zero External Dependencies**: All middlewares use only the Go standard library
3. **Composable**: Each middleware works independently and composes with others
4. **Configurable**: Options pattern for configuration where needed
5. **Safe Defaults**: Secure and sensible defaults out of the box
6. **Well-Tested**: Comprehensive test coverage for each middleware

## Package Structure

```
middlewares/
├── SPEC.md                 # This specification
├── doc.go                  # Package documentation
├── basicauth/              # HTTP Basic Authentication
├── bearerauth/             # Bearer token authentication
├── bodylimit/              # Request body size limiting
├── cache/                  # Cache-Control headers
├── circuitbreaker/         # Circuit breaker pattern
├── compress/               # Compression (gzip, deflate)
├── contenttype/            # Content-Type validation
├── cors/                   # Cross-Origin Resource Sharing
├── csrf/                   # CSRF protection
├── etag/                   # ETag generation
├── expvar/                 # Expvar metrics endpoint
├── forwarded/              # X-Forwarded-* header handling
├── healthcheck/            # Health check endpoints
├── helmet/                 # Security headers collection
├── idempotency/            # Idempotency key support
├── ipfilter/               # IP whitelist/blacklist
├── jwt/                    # JWT authentication
├── keyauth/                # API key authentication
├── methodoverride/         # HTTP method override
├── nocache/                # No-cache headers
├── pprof/                  # Profiling endpoints
├── proxy/                  # Reverse proxy support
├── ratelimit/              # Rate limiting
├── realip/                 # Real client IP extraction
├── recover/                # Panic recovery
├── redirect/               # URL redirection
├── requestid/              # Request ID generation
├── rewrite/                # URL rewriting
├── secure/                 # HTTPS redirect & security
├── session/                # Session management
├── slash/                  # Trailing slash handling
├── timeout/                # Request timeout
├── timing/                 # Server-Timing header
└── version/                # API versioning
```

---

## Middleware Specifications

### 1. basicauth - HTTP Basic Authentication

**Package**: `middlewares/basicauth`

**Purpose**: Authenticate requests using HTTP Basic Authentication (RFC 7617).

**Functions**:
```go
// New creates a middleware that validates credentials against a static map.
func New(credentials map[string]string) mizu.Middleware

// WithValidator creates a middleware using a custom validator function.
func WithValidator(fn ValidatorFunc) mizu.Middleware

// WithRealm creates a middleware with a custom realm name.
func WithRealm(realm string, credentials map[string]string) mizu.Middleware
```

**Types**:
```go
type ValidatorFunc func(username, password string) bool
```

**Behavior**:
- Returns 401 with `WWW-Authenticate` header if credentials missing/invalid
- Calls next handler if credentials valid
- Constant-time comparison to prevent timing attacks

---

### 2. bearerauth - Bearer Token Authentication

**Package**: `middlewares/bearerauth`

**Purpose**: Authenticate requests using Bearer tokens (RFC 6750).

**Functions**:
```go
// New creates a middleware that validates bearer tokens.
func New(validator TokenValidator) mizu.Middleware

// WithHeader creates a middleware reading from a custom header.
func WithHeader(header string, validator TokenValidator) mizu.Middleware
```

**Types**:
```go
type TokenValidator func(token string) bool
```

**Behavior**:
- Extracts token from `Authorization: Bearer <token>` header
- Returns 401 if token missing, 403 if invalid
- Supports custom header names

---

### 3. bodylimit - Request Body Size Limiting

**Package**: `middlewares/bodylimit`

**Purpose**: Limit request body size to prevent resource exhaustion.

**Functions**:
```go
// New creates a middleware limiting body to n bytes.
func New(n int64) mizu.Middleware

// WithHandler creates a middleware with custom error handling.
func WithHandler(n int64, handler func(*mizu.Ctx) error) mizu.Middleware
```

**Behavior**:
- Wraps request body with `io.LimitReader`
- Returns 413 Request Entity Too Large if exceeded
- Default limit: 1MB if n <= 0

---

### 4. cache - Cache-Control Headers

**Package**: `middlewares/cache`

**Purpose**: Set Cache-Control response headers.

**Functions**:
```go
// New creates a middleware setting Cache-Control with max-age.
func New(maxAge time.Duration) mizu.Middleware

// WithOptions creates a middleware with full cache control options.
func WithOptions(opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    MaxAge               time.Duration
    SMaxAge              time.Duration
    Public               bool
    Private              bool
    NoCache              bool
    NoStore              bool
    NoTransform          bool
    MustRevalidate       bool
    ProxyRevalidate      bool
    Immutable            bool
    StaleWhileRevalidate time.Duration
    StaleIfError         time.Duration
}
```

**Behavior**:
- Sets `Cache-Control` header based on options
- Only applies to successful responses (2xx)
- Skips if `Cache-Control` already set

---

### 5. circuitbreaker - Circuit Breaker Pattern

**Package**: `middlewares/circuitbreaker`

**Purpose**: Implement circuit breaker pattern to prevent cascade failures.

**Functions**:
```go
// New creates a circuit breaker with default settings.
func New() mizu.Middleware

// WithOptions creates a circuit breaker with custom settings.
func WithOptions(opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    Threshold   int           // Failures before opening (default: 5)
    Timeout     time.Duration // Time before half-open (default: 30s)
    MaxRequests int           // Requests allowed in half-open (default: 1)
    OnStateChange func(from, to State)
}

type State int
const (
    StateClosed State = iota
    StateOpen
    StateHalfOpen
)
```

**Behavior**:
- Tracks consecutive failures
- Opens circuit after threshold reached
- Returns 503 when circuit is open
- Allows test requests in half-open state

---

### 6. compress - Response Compression

**Package**: `middlewares/compress`

**Purpose**: Compress response bodies using gzip or deflate.

**Functions**:
```go
// Gzip creates gzip compression middleware.
func Gzip() mizu.Middleware

// GzipLevel creates gzip compression with specified level.
func GzipLevel(level int) mizu.Middleware

// Deflate creates deflate compression middleware.
func Deflate() mizu.Middleware

// DeflateLevel creates deflate compression with specified level.
func DeflateLevel(level int) mizu.Middleware

// New creates compression middleware supporting multiple algorithms.
func New(opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    Level         int      // Compression level (1-9, default: 6)
    MinSize       int      // Minimum size to compress (default: 1024)
    ContentTypes  []string // Types to compress (default: text/*, application/json, etc.)
}
```

**Behavior**:
- Checks `Accept-Encoding` header for supported algorithm
- Sets `Content-Encoding` and `Vary: Accept-Encoding`
- Skips small responses and binary content types
- Supports streaming compression

---

### 7. contenttype - Content-Type Validation

**Package**: `middlewares/contenttype`

**Purpose**: Validate and enforce Content-Type headers.

**Functions**:
```go
// Require creates middleware requiring specific content types.
func Require(types ...string) mizu.Middleware

// RequireJSON requires application/json content type.
func RequireJSON() mizu.Middleware

// RequireForm requires form content types.
func RequireForm() mizu.Middleware

// Default sets default Content-Type if not present.
func Default(contentType string) mizu.Middleware
```

**Behavior**:
- Returns 415 Unsupported Media Type if Content-Type doesn't match
- Only checks requests with body (POST, PUT, PATCH)
- Supports media type parameters (charset, boundary)

---

### 8. cors - Cross-Origin Resource Sharing

**Package**: `middlewares/cors`

**Purpose**: Handle CORS preflight requests and response headers.

**Functions**:
```go
// New creates CORS middleware with options.
func New(opts Options) mizu.Middleware

// AllowAll creates permissive CORS middleware (for development).
func AllowAll() mizu.Middleware
```

**Types**:
```go
type Options struct {
    AllowOrigins     []string      // Allowed origins ("*" for any)
    AllowMethods     []string      // Allowed methods (default: GET, POST, HEAD)
    AllowHeaders     []string      // Allowed request headers
    ExposeHeaders    []string      // Headers exposed to browser
    AllowCredentials bool          // Allow credentials
    MaxAge           time.Duration // Preflight cache duration
    AllowOriginFunc  func(origin string) bool // Dynamic origin validation
}
```

**Behavior**:
- Handles OPTIONS preflight requests
- Sets `Access-Control-*` headers
- Validates origin against allowed list
- Supports wildcard and dynamic origin validation

---

### 9. csrf - CSRF Protection

**Package**: `middlewares/csrf`

**Purpose**: Protect against Cross-Site Request Forgery attacks.

**Functions**:
```go
// New creates CSRF protection middleware.
func New(opts Options) mizu.Middleware

// Token extracts CSRF token from context for templates.
func Token(c *mizu.Ctx) string
```

**Types**:
```go
type Options struct {
    Secret        []byte        // Secret for token generation
    TokenLength   int           // Token length (default: 32)
    TokenLookup   string        // Where to find token (default: "header:X-CSRF-Token")
    CookieName    string        // Cookie name (default: "_csrf")
    CookiePath    string        // Cookie path (default: "/")
    CookieMaxAge  int           // Cookie max age (default: 86400)
    CookieSecure  bool          // Secure cookie
    CookieHTTPOnly bool         // HTTPOnly cookie
    SameSite      http.SameSite // SameSite attribute
    ErrorHandler  func(*mizu.Ctx, error) error
}
```

**Behavior**:
- Generates and validates CSRF tokens
- Stores token in cookie, validates from header/form
- Skips safe methods (GET, HEAD, OPTIONS, TRACE)
- Uses double submit cookie pattern

---

### 10. etag - ETag Generation

**Package**: `middlewares/etag`

**Purpose**: Generate ETag headers and handle conditional requests.

**Functions**:
```go
// New creates ETag middleware with default settings.
func New() mizu.Middleware

// WithOptions creates ETag middleware with options.
func WithOptions(opts Options) mizu.Middleware

// Weak creates middleware generating weak ETags.
func Weak() mizu.Middleware
```

**Types**:
```go
type Options struct {
    Weak     bool                      // Generate weak ETags
    HashFunc func([]byte) string       // Custom hash function (default: CRC32)
}
```

**Behavior**:
- Generates ETag from response body hash
- Handles `If-None-Match` for conditional GET
- Returns 304 Not Modified when appropriate
- Buffers response for hashing

---

### 11. expvar - Expvar Metrics Endpoint

**Package**: `middlewares/expvar`

**Purpose**: Expose expvar metrics via HTTP endpoint.

**Functions**:
```go
// Handler returns an HTTP handler for expvar.
func Handler() mizu.Handler

// Middleware wraps handler and tracks request metrics.
func Middleware() mizu.Middleware
```

**Behavior**:
- Exposes `/debug/vars` endpoint
- Tracks request count, duration, active requests
- Thread-safe counter updates

---

### 12. forwarded - X-Forwarded-* Header Handling

**Package**: `middlewares/forwarded`

**Purpose**: Parse and trust proxy forwarding headers.

**Functions**:
```go
// New creates middleware trusting X-Forwarded-* headers.
func New(opts Options) mizu.Middleware

// TrustProxy creates middleware trusting specific proxy IPs.
func TrustProxy(proxies ...string) mizu.Middleware
```

**Types**:
```go
type Options struct {
    TrustProxy     []string // Trusted proxy IPs/CIDRs
    ForwardedFor   bool     // Trust X-Forwarded-For (default: true)
    ForwardedProto bool     // Trust X-Forwarded-Proto (default: true)
    ForwardedHost  bool     // Trust X-Forwarded-Host (default: true)
}
```

**Behavior**:
- Updates `c.Request().RemoteAddr` from `X-Forwarded-For`
- Updates scheme from `X-Forwarded-Proto`
- Only trusts headers from known proxy IPs

---

### 13. healthcheck - Health Check Endpoints

**Package**: `middlewares/healthcheck`

**Purpose**: Provide health check endpoints for load balancers.

**Functions**:
```go
// New creates a health check handler.
func New(opts Options) mizu.Handler

// Liveness creates a simple liveness probe handler.
func Liveness() mizu.Handler

// Readiness creates a readiness probe with checks.
func Readiness(checks ...Check) mizu.Handler
```

**Types**:
```go
type Options struct {
    LivenessPath  string   // Path for liveness (default: "/healthz")
    ReadinessPath string   // Path for readiness (default: "/readyz")
    Checks        []Check  // Health checks to run
}

type Check struct {
    Name    string
    Check   func(context.Context) error
    Timeout time.Duration
}

type Status struct {
    Status  string            `json:"status"` // "ok" or "error"
    Checks  map[string]string `json:"checks,omitempty"`
}
```

**Behavior**:
- Returns 200 for healthy, 503 for unhealthy
- Runs checks concurrently with timeout
- Returns JSON with check details

---

### 14. helmet - Security Headers Collection

**Package**: `middlewares/helmet`

**Purpose**: Set security-related HTTP headers.

**Functions**:
```go
// Default creates middleware with recommended security headers.
func Default() mizu.Middleware

// New creates middleware with custom options.
func New(opts Options) mizu.Middleware

// ContentSecurityPolicy sets CSP header.
func ContentSecurityPolicy(policy string) mizu.Middleware

// XFrameOptions sets X-Frame-Options header.
func XFrameOptions(value string) mizu.Middleware

// XContentTypeOptions sets X-Content-Type-Options header.
func XContentTypeOptions() mizu.Middleware

// ReferrerPolicy sets Referrer-Policy header.
func ReferrerPolicy(policy string) mizu.Middleware

// StrictTransportSecurity sets HSTS header.
func StrictTransportSecurity(maxAge time.Duration, includeSubDomains, preload bool) mizu.Middleware

// PermissionsPolicy sets Permissions-Policy header.
func PermissionsPolicy(policy string) mizu.Middleware

// CrossOriginOpenerPolicy sets COOP header.
func CrossOriginOpenerPolicy(policy string) mizu.Middleware

// CrossOriginEmbedderPolicy sets COEP header.
func CrossOriginEmbedderPolicy(policy string) mizu.Middleware

// CrossOriginResourcePolicy sets CORP header.
func CrossOriginResourcePolicy(policy string) mizu.Middleware

// OriginAgentCluster sets Origin-Agent-Cluster header.
func OriginAgentCluster() mizu.Middleware

// XDNSPrefetchControl sets X-DNS-Prefetch-Control header.
func XDNSPrefetchControl(on bool) mizu.Middleware

// XDownloadOptions sets X-Download-Options header.
func XDownloadOptions() mizu.Middleware

// XPermittedCrossDomainPolicies sets X-Permitted-Cross-Domain-Policies header.
func XPermittedCrossDomainPolicies(policy string) mizu.Middleware
```

**Types**:
```go
type Options struct {
    ContentSecurityPolicy           string
    XFrameOptions                   string // DENY, SAMEORIGIN
    XContentTypeOptions             bool   // nosniff
    ReferrerPolicy                  string
    StrictTransportSecurity         *HSTSOptions
    PermissionsPolicy               string
    CrossOriginOpenerPolicy         string // same-origin, unsafe-none, same-origin-allow-popups
    CrossOriginEmbedderPolicy       string // require-corp, credentialless, unsafe-none
    CrossOriginResourcePolicy       string // same-site, same-origin, cross-origin
    OriginAgentCluster              bool
    XDNSPrefetchControl             *bool  // nil = don't set
    XDownloadOptions                bool   // noopen
    XPermittedCrossDomainPolicies   string // none, master-only, by-content-type, all
}

type HSTSOptions struct {
    MaxAge            time.Duration
    IncludeSubDomains bool
    Preload           bool
}
```

**Default Headers**:
```
X-Content-Type-Options: nosniff
X-Frame-Options: SAMEORIGIN
X-DNS-Prefetch-Control: off
X-Download-Options: noopen
X-Permitted-Cross-Domain-Policies: none
Referrer-Policy: strict-origin-when-cross-origin
Cross-Origin-Opener-Policy: same-origin
Cross-Origin-Resource-Policy: same-origin
Origin-Agent-Cluster: ?1
```

---

### 15. idempotency - Idempotency Key Support

**Package**: `middlewares/idempotency`

**Purpose**: Support idempotent requests using idempotency keys.

**Functions**:
```go
// New creates idempotency middleware with in-memory store.
func New(opts Options) mizu.Middleware

// WithStore creates idempotency middleware with custom store.
func WithStore(store Store, opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    Header     string        // Header name (default: "Idempotency-Key")
    TTL        time.Duration // Response cache TTL (default: 24h)
    MaxKeyLen  int           // Max key length (default: 256)
    Methods    []string      // Methods to handle (default: POST, PUT, PATCH)
}

type Store interface {
    Get(key string) (*Response, bool)
    Set(key string, resp *Response, ttl time.Duration) error
    Delete(key string) error
}

type Response struct {
    Status  int
    Headers http.Header
    Body    []byte
}
```

**Behavior**:
- Caches responses by idempotency key
- Returns cached response for duplicate requests
- Thread-safe in-memory store included
- Supports custom stores (Redis, etc.)

---

### 16. ipfilter - IP Whitelist/Blacklist

**Package**: `middlewares/ipfilter`

**Purpose**: Allow or deny requests based on client IP.

**Functions**:
```go
// Allow creates middleware allowing only listed IPs.
func Allow(ips ...string) mizu.Middleware

// Deny creates middleware denying listed IPs.
func Deny(ips ...string) mizu.Middleware

// New creates middleware with options.
func New(opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    AllowList      []string     // IPs/CIDRs to allow
    DenyList       []string     // IPs/CIDRs to deny
    DenyByDefault  bool         // Deny unless in allow list (default: false)
    TrustProxy     bool         // Use X-Forwarded-For (default: false)
    ErrorHandler   func(*mizu.Ctx) error
}
```

**Behavior**:
- Supports both IPv4 and IPv6
- Supports CIDR notation
- Returns 403 for denied IPs
- Deny list takes precedence

---

### 17. jwt - JWT Authentication

**Package**: `middlewares/jwt`

**Purpose**: Authenticate requests using JSON Web Tokens.

**Functions**:
```go
// New creates JWT middleware with HMAC signing.
func New(secret []byte) mizu.Middleware

// WithOptions creates JWT middleware with options.
func WithOptions(opts Options) mizu.Middleware

// Claims extracts claims from context.
func Claims(c *mizu.Ctx) map[string]any

// Subject extracts subject claim from context.
func Subject(c *mizu.Ctx) string
```

**Types**:
```go
type Options struct {
    Secret          []byte                    // HMAC secret
    PublicKey       any                       // RSA/ECDSA public key
    Algorithm       string                    // HS256, RS256, ES256 (default: HS256)
    TokenLookup     string                    // Where to find token (default: "header:Authorization")
    AuthScheme      string                    // Auth scheme (default: "Bearer")
    Claims          map[string]any            // Required claims
    Issuer          string                    // Required issuer
    Audience        []string                  // Required audience
    ContextKey      string                    // Context key for claims (default: "jwt_claims")
    ErrorHandler    func(*mizu.Ctx, error) error
}
```

**Behavior**:
- Validates JWT signature
- Validates standard claims (exp, nbf, iat, iss, aud)
- Stores claims in request context
- Supports RS256, ES256 in addition to HS256

---

### 18. keyauth - API Key Authentication

**Package**: `middlewares/keyauth`

**Purpose**: Authenticate requests using API keys.

**Functions**:
```go
// New creates API key middleware with validator.
func New(validator KeyValidator) mizu.Middleware

// WithOptions creates API key middleware with options.
func WithOptions(opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    Validator    KeyValidator
    KeyLookup    string // Where to find key (default: "header:X-API-Key")
    AuthScheme   string // For header lookup (default: "")
    ContextKey   string // Context key (default: "api_key")
    ErrorHandler func(*mizu.Ctx, error) error
}

type KeyValidator func(key string) (bool, error)
```

**Behavior**:
- Extracts key from header, query, or cookie
- Validates against provided validator
- Returns 401 if key missing, 403 if invalid

---

### 19. methodoverride - HTTP Method Override

**Package**: `middlewares/methodoverride`

**Purpose**: Override HTTP method via header or form field.

**Functions**:
```go
// New creates method override middleware.
func New() mizu.Middleware

// WithOptions creates method override with options.
func WithOptions(opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    Header    string   // Header name (default: "X-HTTP-Method-Override")
    FormField string   // Form field name (default: "_method")
    Methods   []string // Allowed override methods (default: PUT, PATCH, DELETE)
}
```

**Behavior**:
- Only overrides POST requests
- Validates override method against allowed list
- Updates `c.Request().Method`

---

### 20. nocache - No-Cache Headers

**Package**: `middlewares/nocache`

**Purpose**: Set headers to prevent caching.

**Functions**:
```go
// New creates no-cache middleware.
func New() mizu.Middleware
```

**Behavior**:
Sets headers:
```
Cache-Control: no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0
Pragma: no-cache
Expires: 0
Surrogate-Control: no-store
```

---

### 21. pprof - Profiling Endpoints

**Package**: `middlewares/pprof`

**Purpose**: Expose pprof profiling endpoints.

**Functions**:
```go
// Register registers pprof routes on a router.
func Register(r *mizu.Router)

// Handler returns pprof index handler.
func Handler() mizu.Handler

// Cmdline returns cmdline handler.
func Cmdline() mizu.Handler

// Profile returns CPU profile handler.
func Profile() mizu.Handler

// Symbol returns symbol handler.
func Symbol() mizu.Handler

// Trace returns trace handler.
func Trace() mizu.Handler

// Heap returns heap profile handler.
func Heap() mizu.Handler

// Goroutine returns goroutine profile handler.
func Goroutine() mizu.Handler

// Block returns block profile handler.
func Block() mizu.Handler

// ThreadCreate returns thread creation profile handler.
func ThreadCreate() mizu.Handler

// Mutex returns mutex profile handler.
func Mutex() mizu.Handler

// Allocs returns allocation profile handler.
func Allocs() mizu.Handler
```

**Behavior**:
- Wraps standard `net/http/pprof` handlers
- Registers routes at `/debug/pprof/*`
- Should be protected by authentication in production

---

### 22. proxy - Reverse Proxy Support

**Package**: `middlewares/proxy`

**Purpose**: Forward requests to backend servers.

**Functions**:
```go
// New creates a reverse proxy middleware.
func New(target *url.URL) mizu.Middleware

// WithOptions creates reverse proxy with options.
func WithOptions(target *url.URL, opts Options) mizu.Middleware

// LoadBalance creates a load-balanced reverse proxy.
func LoadBalance(targets []*url.URL, opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    Transport       http.RoundTripper
    ModifyRequest   func(*http.Request)
    ModifyResponse  func(*http.Response) error
    ErrorHandler    func(http.ResponseWriter, *http.Request, error)
    BufferPool      httputil.BufferPool
    FlushInterval   time.Duration
    Balancer        Balancer
}

type Balancer interface {
    Next([]*url.URL) *url.URL
}
```

**Behavior**:
- Uses `httputil.ReverseProxy`
- Supports request/response modification
- Supports load balancing (round-robin, random, least-conn)
- Sets X-Forwarded-* headers

---

### 23. ratelimit - Rate Limiting

**Package**: `middlewares/ratelimit`

**Purpose**: Limit request rate per client.

**Functions**:
```go
// New creates rate limiter with requests per interval.
func New(rate int, interval time.Duration) mizu.Middleware

// WithOptions creates rate limiter with options.
func WithOptions(opts Options) mizu.Middleware

// WithStore creates rate limiter with custom store.
func WithStore(store Store, opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    Rate         int                       // Requests per interval
    Interval     time.Duration             // Time interval
    Burst        int                       // Burst capacity (default: Rate)
    KeyFunc      func(*mizu.Ctx) string    // Key extraction (default: client IP)
    Headers      bool                      // Include rate limit headers (default: true)
    ErrorHandler func(*mizu.Ctx) error
}

type Store interface {
    Allow(key string, rate int, interval time.Duration, burst int) (bool, RateLimitInfo)
}

type RateLimitInfo struct {
    Limit     int
    Remaining int
    Reset     time.Time
}
```

**Behavior**:
- Token bucket algorithm
- Per-client rate limiting by IP
- Sets `X-RateLimit-*` and `Retry-After` headers
- Returns 429 when limit exceeded
- In-memory store with automatic cleanup

---

### 24. realip - Real Client IP Extraction

**Package**: `middlewares/realip`

**Purpose**: Extract real client IP from proxy headers.

**Functions**:
```go
// New creates realip middleware with defaults.
func New() mizu.Middleware

// WithTrustedProxies creates middleware with trusted proxy list.
func WithTrustedProxies(proxies ...string) mizu.Middleware

// FromContext extracts the real IP from context.
func FromContext(c *mizu.Ctx) string
```

**Types**:
```go
type Options struct {
    TrustedProxies []string // Trusted proxy IPs/CIDRs
    TrustedHeaders []string // Headers to check (default: X-Forwarded-For, X-Real-IP)
}
```

**Behavior**:
- Checks headers in order: `X-Forwarded-For`, `X-Real-IP`, `CF-Connecting-IP`
- Only trusts headers from known proxies
- Stores real IP in context for later use
- Falls back to `RemoteAddr`

---

### 25. recover - Panic Recovery

**Package**: `middlewares/recover`

**Purpose**: Recover from panics and return error response.

**Functions**:
```go
// New creates recovery middleware.
func New() mizu.Middleware

// WithOptions creates recovery middleware with options.
func WithOptions(opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    StackSize        int                    // Stack trace buffer size (default: 4KB)
    DisableStackAll  bool                   // Only capture current goroutine
    DisablePrintStack bool                  // Don't log stack trace
    ErrorHandler     func(*mizu.Ctx, any, []byte) error
    Logger           *slog.Logger
}
```

**Behavior**:
- Recovers from panics
- Logs error with stack trace
- Returns 500 Internal Server Error
- Custom error handler supported

---

### 26. redirect - URL Redirection

**Package**: `middlewares/redirect`

**Purpose**: Redirect requests based on rules.

**Functions**:
```go
// HTTPSRedirect redirects HTTP to HTTPS.
func HTTPSRedirect() mizu.Middleware

// HTTPSRedirectCode redirects HTTP to HTTPS with specific code.
func HTTPSRedirectCode(code int) mizu.Middleware

// WWWRedirect redirects to www subdomain.
func WWWRedirect() mizu.Middleware

// NonWWWRedirect redirects to non-www domain.
func NonWWWRedirect() mizu.Middleware

// New creates redirect middleware with rules.
func New(rules []Rule) mizu.Middleware
```

**Types**:
```go
type Rule struct {
    From     string // Path or pattern
    To       string // Target URL (supports $1, $2 for captures)
    Code     int    // Redirect code (default: 301)
    Regex    bool   // Use regex matching
}
```

**Behavior**:
- Supports permanent (301) and temporary (302, 307, 308) redirects
- Supports regex patterns with capture groups
- Preserves query string

---

### 27. requestid - Request ID Generation

**Package**: `middlewares/requestid`

**Purpose**: Generate and propagate request IDs.

**Functions**:
```go
// New creates request ID middleware.
func New() mizu.Middleware

// WithOptions creates request ID middleware with options.
func WithOptions(opts Options) mizu.Middleware

// FromContext extracts request ID from context.
func FromContext(c *mizu.Ctx) string
```

**Types**:
```go
type Options struct {
    Header     string            // Header name (default: "X-Request-ID")
    Generator  func() string     // ID generator (default: UUID v4)
    ContextKey string            // Context key (default: "request_id")
}
```

**Behavior**:
- Uses existing request ID from header if present
- Generates new ID if not present
- Stores in context for logging
- Sets response header

---

### 28. rewrite - URL Rewriting

**Package**: `middlewares/rewrite`

**Purpose**: Rewrite request URLs.

**Functions**:
```go
// New creates URL rewrite middleware.
func New(rules []Rule) mizu.Middleware

// StripPrefix strips URL prefix.
func StripPrefix(prefix string) mizu.Middleware

// AddPrefix adds URL prefix.
func AddPrefix(prefix string) mizu.Middleware
```

**Types**:
```go
type Rule struct {
    Match   string // Path or regex pattern
    Rewrite string // Rewrite target (supports $1, $2)
    Regex   bool   // Use regex matching
}
```

**Behavior**:
- Rewrites URL internally (no redirect)
- Supports regex capture groups
- Processes rules in order, first match wins

---

### 29. secure - HTTPS & Security

**Package**: `middlewares/secure`

**Purpose**: Enforce HTTPS and security best practices.

**Functions**:
```go
// New creates secure middleware with options.
func New(opts Options) mizu.Middleware

// Default creates secure middleware with recommended settings.
func Default() mizu.Middleware
```

**Types**:
```go
type Options struct {
    // HTTPS
    HTTPSRedirect          bool
    HTTPSRedirectCode      int
    HTTPSHost              string // Host for HTTPS redirect

    // Host
    AllowedHosts           []string // Allowed Host header values
    HostsProxyHeaders      []string // Headers to check for host

    // SSL Proxy
    SSLProxyHeaders        map[string]string // e.g., {"X-Forwarded-Proto": "https"}

    // Frame options
    FrameDeny              bool   // Set X-Frame-Options: DENY
    CustomFrameOptions     string // Custom X-Frame-Options value

    // Content type
    ContentTypeNosniff     bool   // Set X-Content-Type-Options: nosniff

    // XSS
    BrowserXSSFilter       bool   // Set X-XSS-Protection

    // HSTS
    STSSeconds             int64  // Strict-Transport-Security max-age
    STSIncludeSubdomains   bool
    STSPreload             bool

    // Content Security Policy
    ContentSecurityPolicy  string

    // Referrer
    ReferrerPolicy         string

    // Development
    IsDevelopment          bool   // Disable some features for development
}
```

**Behavior**:
- Combines multiple security headers
- HTTPS redirect with configurable host
- Host header validation

---

### 30. session - Session Management

**Package**: `middlewares/session`

**Purpose**: Cookie-based session management.

**Functions**:
```go
// New creates session middleware with in-memory store.
func New(opts Options) mizu.Middleware

// WithStore creates session middleware with custom store.
func WithStore(store Store, opts Options) mizu.Middleware

// Get retrieves session from context.
func Get(c *mizu.Ctx) *Session

// FromContext retrieves session from context (alias).
func FromContext(c *mizu.Ctx) *Session
```

**Types**:
```go
type Options struct {
    CookieName    string        // Session cookie name (default: "session_id")
    CookiePath    string        // Cookie path (default: "/")
    CookieDomain  string        // Cookie domain
    CookieMaxAge  int           // Cookie max age in seconds (default: 86400)
    CookieSecure  bool          // Secure cookie
    CookieHTTPOnly bool         // HTTPOnly cookie
    SameSite      http.SameSite // SameSite attribute (default: Lax)
    IdleTimeout   time.Duration // Session idle timeout
    Lifetime      time.Duration // Absolute session lifetime
    KeyGenerator  func() string // Session ID generator
}

type Store interface {
    Get(id string) (*SessionData, error)
    Save(id string, data *SessionData, lifetime time.Duration) error
    Delete(id string) error
    GC() // Garbage collection
}

type Session struct {
    ID   string
    data map[string]any
    // ... internal fields
}

func (s *Session) Get(key string) any
func (s *Session) Set(key string, value any)
func (s *Session) Delete(key string)
func (s *Session) Clear()
func (s *Session) Regenerate() error // New ID, keep data
func (s *Session) Destroy() error    // Delete session
func (s *Session) Flash(key string, value any)
func (s *Session) GetFlash(key string) any
```

**Behavior**:
- Cookie-based session ID
- Lazy loading (only loads when accessed)
- Flash messages support
- Automatic garbage collection
- Session regeneration for security

---

### 31. slash - Trailing Slash Handling

**Package**: `middlewares/slash`

**Purpose**: Normalize trailing slashes in URLs.

**Functions**:
```go
// Add redirects to URL with trailing slash.
func Add() mizu.Middleware

// AddCode redirects with specific status code.
func AddCode(code int) mizu.Middleware

// Remove redirects to URL without trailing slash.
func Remove() mizu.Middleware

// RemoveCode redirects with specific status code.
func RemoveCode(code int) mizu.Middleware
```

**Behavior**:
- Redirects requests to normalized URL
- Default redirect code: 301
- Preserves query string
- Skips root path "/"

---

### 32. timeout - Request Timeout

**Package**: `middlewares/timeout`

**Purpose**: Limit request processing time.

**Functions**:
```go
// New creates timeout middleware.
func New(timeout time.Duration) mizu.Middleware

// WithOptions creates timeout middleware with options.
func WithOptions(opts Options) mizu.Middleware
```

**Types**:
```go
type Options struct {
    Timeout      time.Duration
    ErrorHandler func(*mizu.Ctx) error
    ErrorMessage string // Default: "Request Timeout"
}
```

**Behavior**:
- Creates context with deadline
- Returns 503 Service Unavailable on timeout
- Handler should check `c.Context().Done()`
- Custom error handler supported

---

### 33. timing - Server-Timing Header

**Package**: `middlewares/timing`

**Purpose**: Add Server-Timing header for performance monitoring.

**Functions**:
```go
// New creates timing middleware.
func New() mizu.Middleware

// Add adds a metric to Server-Timing header.
func Add(c *mizu.Ctx, name string, duration time.Duration, description string)

// Start starts a timing measurement.
func Start(c *mizu.Ctx, name string) func(description string)
```

**Behavior**:
- Automatically measures total request time
- Allows adding custom timing metrics
- Format: `Server-Timing: total;dur=123.4, db;dur=56.7;desc="Database query"`

---

### 34. version - API Versioning

**Package**: `middlewares/version`

**Purpose**: API versioning via header, URL, or query parameter.

**Functions**:
```go
// New creates versioning middleware.
func New(opts Options) mizu.Middleware

// FromHeader reads version from header.
func FromHeader(header string) mizu.Middleware

// FromPath extracts version from URL path.
func FromPath() mizu.Middleware

// FromQuery reads version from query parameter.
func FromQuery(param string) mizu.Middleware

// GetVersion extracts version from context.
func GetVersion(c *mizu.Ctx) string
```

**Types**:
```go
type Options struct {
    DefaultVersion string   // Default if not specified
    Header         string   // Header name (default: "Accept-Version")
    QueryParam     string   // Query param name (default: "version")
    PathPrefix     bool     // Extract from path prefix (/v1/...)
    Supported      []string // Supported versions (for validation)
    Deprecated     []string // Deprecated versions (adds Deprecation header)
}
```

**Behavior**:
- Extracts version and stores in context
- Returns 400 for unsupported versions
- Adds `Deprecation` header for deprecated versions

---

## Additional Middlewares (35-100)

### Authentication & Authorization

**35. oauth2** (`middlewares/oauth2`)
- OAuth 2.0 middleware for resource servers
- Validates access tokens, extracts scopes

**36. oidc** (`middlewares/oidc`)
- OpenID Connect authentication
- ID token validation, user info extraction

**37. rbac** (`middlewares/rbac`)
- Role-based access control
- Check user roles against required roles

**38. casbin** (`middlewares/casbin`)
- Casbin authorization middleware
- Flexible PERM model support

---

### Request Processing

**39. validator** (`middlewares/validator`)
- Request validation middleware
- JSON schema validation, struct tags

**40. sanitizer** (`middlewares/sanitizer`)
- Input sanitization
- HTML escaping, SQL injection prevention

**41. transformer** (`middlewares/transformer`)
- Request/response transformation
- Field renaming, data conversion

**42. filter** (`middlewares/filter`)
- Request filtering
- Query parameter filtering, field selection

---

### Response Processing

**43. envelope** (`middlewares/envelope`)
- Response envelope wrapping
- Consistent API response format

**44. hypermedia** (`middlewares/hypermedia`)
- HATEOAS link injection
- Hypermedia controls for REST APIs

**45. xml** (`middlewares/xml`)
- XML response handling
- Content negotiation for XML

**46. msgpack** (`middlewares/msgpack`)
- MessagePack serialization
- Binary format response support

**47. protobuf** (`middlewares/protobuf`)
- Protocol Buffers support
- Binary serialization for gRPC-web

**48. cbor** (`middlewares/cbor`)
- CBOR serialization
- Compact binary format

---

### Caching

**49. lastmodified** (`middlewares/lastmodified`)
- Last-Modified header handling
- If-Modified-Since support

**50. conditional** (`middlewares/conditional`)
- Combined conditional request handling
- ETag + Last-Modified

**51. vary** (`middlewares/vary`)
- Vary header management
- Proper cache key variation

**52. surrogate** (`middlewares/surrogate`)
- Surrogate-Key headers for CDN
- Cache invalidation support

---

### Monitoring & Observability

**53. prometheus** (`middlewares/prometheus`)
- Prometheus metrics
- Request duration, status codes, path labels

**54. otel** (`middlewares/otel`)
- OpenTelemetry tracing
- Span creation, context propagation

**55. sentry** (`middlewares/sentry`)
- Sentry error reporting integration
- Panic capturing, breadcrumbs

**56. audit** (`middlewares/audit`)
- Audit logging middleware
- Who did what, when

**57. requestlog** (`middlewares/requestlog`)
- Detailed request logging
- Headers, body, timing

**58. responselog** (`middlewares/responselog`)
- Response body logging
- Debug and audit purposes

---

### Traffic Management

**59. throttle** (`middlewares/throttle`)
- Sliding window rate limiting
- More accurate than token bucket

**60. concurrency** (`middlewares/concurrency`)
- Limit concurrent requests
- Semaphore-based limiting

**61. bulkhead** (`middlewares/bulkhead`)
- Bulkhead isolation pattern
- Resource isolation

**62. adaptive** (`middlewares/adaptive`)
- Adaptive rate limiting
- Based on server load/response time

**63. retry** (`middlewares/retry`)
- Retry middleware for proxied requests
- Exponential backoff

**64. hedge** (`middlewares/hedge`)
- Hedged requests
- Send duplicate requests after timeout

---

### Content Delivery

**65. static** (`middlewares/static`)
- Static file serving with options
- Index files, directory listing

**66. spa** (`middlewares/spa`)
- Single Page Application support
- Fallback to index.html

**67. favicon** (`middlewares/favicon`)
- Favicon handling
- Caching, custom path

**68. embed** (`middlewares/embed`)
- Embedded filesystem serving
- go:embed support

---

### Protocol Handling

**69. websocket** (`middlewares/websocket`)
- WebSocket upgrade handling
- Connection management

**70. sse** (`middlewares/sse`)
- Server-Sent Events support
- Event broadcasting

**71. grpcweb** (`middlewares/grpcweb`)
- gRPC-Web support
- Browser-compatible gRPC

**72. h2c** (`middlewares/h2c`)
- HTTP/2 cleartext upgrade
- h2c protocol support

---

### Security Extensions

**73. csrf2** (`middlewares/csrf2`)
- Double submit cookie CSRF
- Alternative CSRF implementation

**74. nonce** (`middlewares/nonce`)
- CSP nonce generation
- Script/style nonces

**75. captcha** (`middlewares/captcha`)
- CAPTCHA verification
- Rate limiting integration

**76. honeypot** (`middlewares/honeypot`)
- Honeypot form fields
- Bot detection

**77. bot** (`middlewares/bot`)
- Bot detection and filtering
- User-agent analysis

**78. fingerprint** (`middlewares/fingerprint`)
- Request fingerprinting
- TLS fingerprint, headers

---

### Localization

**79. language** (`middlewares/language`)
- Accept-Language negotiation
- Locale detection

**80. timezone** (`middlewares/timezone`)
- Timezone detection and handling
- Header/cookie based

---

### Error Handling

**81. errorpage** (`middlewares/errorpage`)
- Custom error pages
- Status-specific templates

**82. fallback** (`middlewares/fallback`)
- Fallback response handling
- Default responses for errors

**83. maintenance** (`middlewares/maintenance`)
- Maintenance mode
- 503 with custom message

---

### Headers Management

**84. header** (`middlewares/header`)
- Generic header manipulation
- Add, remove, modify headers

**85. cors2** (`middlewares/cors2`)
- Simple CORS with single origin
- Lightweight alternative

**86. xrequestedwith** (`middlewares/xrequestedwith`)
- X-Requested-With validation
- AJAX request detection

---

### Connection Management

**87. keepalive** (`middlewares/keepalive`)
- Keep-alive header management
- Connection reuse optimization

**88. maxconns** (`middlewares/maxconns`)
- Maximum connections limit
- Per-client connection limiting

---

### Request Manipulation

**89. bodyclose** (`middlewares/bodyclose`)
- Ensure request body is closed
- Resource leak prevention

**90. bodydump** (`middlewares/bodydump`)
- Dump request/response bodies
- Debugging middleware

**91. requestsize** (`middlewares/requestsize`)
- Track request sizes
- Metrics and logging

**92. responsesize** (`middlewares/responsesize`)
- Track response sizes
- Metrics and logging

---

### Specialized

**93. graphql** (`middlewares/graphql`)
- GraphQL query validation
- Depth limiting, complexity analysis

**94. jsonrpc** (`middlewares/jsonrpc`)
- JSON-RPC 2.0 support
- Request routing

**95. multitenancy** (`middlewares/multitenancy`)
- Multi-tenant support
- Tenant isolation

**96. feature** (`middlewares/feature`)
- Feature flags middleware
- A/B testing support

**97. canary** (`middlewares/canary`)
- Canary deployment support
- Traffic splitting

**98. mirror** (`middlewares/mirror`)
- Traffic mirroring
- Shadow traffic for testing

**99. chaos** (`middlewares/chaos`)
- Chaos engineering middleware
- Fault injection for testing

**100. mock** (`middlewares/mock`)
- Request mocking
- Development and testing support

---

## Implementation Guidelines

### Function Signatures

All middlewares should follow these patterns:

```go
// Simple middleware
func New() mizu.Middleware

// Middleware with single parameter
func New(param Type) mizu.Middleware

// Middleware with options
func WithOptions(opts Options) mizu.Middleware

// Context helpers
func FromContext(c *mizu.Ctx) Type
```

### Options Pattern

Use Options structs for configuration:

```go
type Options struct {
    // Required fields first
    Required string

    // Optional with defaults (document defaults in comments)
    Optional int // Default: 100
}
```

### Error Handling

- Return appropriate HTTP status codes
- Support custom error handlers
- Log errors using provided logger

### Testing

Each middleware must include:
- Unit tests for all functionality
- Table-driven tests for multiple cases
- Benchmark tests for performance-critical code
- Examples in test files

### Documentation

Each package must include:
- Package documentation in `doc.go`
- Function documentation with examples
- Option field documentation

---

## File Structure Per Package

```
middlewares/example/
├── doc.go           # Package documentation
├── example.go       # Main implementation
├── options.go       # Options struct (if complex)
├── store.go         # Store interface (if applicable)
├── memory.go        # In-memory store (if applicable)
└── example_test.go  # Tests
```

---

## Dependencies

All middlewares use **only the Go standard library**. No external dependencies allowed.

Required Go version: **1.22+** (for http.ServeMux routing enhancements)
