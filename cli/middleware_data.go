package cli

// middlewareInfo describes a middleware package.
type middlewareInfo struct {
	Name        string   // Package name
	Description string   // Short description
	Category    string   // Category name
	Import      string   // Full import path
	QuickStart  string   // Quick start code snippet
	Related     []string // Related middleware names
}

// categories defines the middleware categories and their display order.
var categories = []string{
	"security",
	"logging",
	"ratelimit",
	"cache",
	"encoding",
	"resilience",
	"routing",
	"content",
	"api",
	"session",
	"observability",
	"misc",
}

// categoryDescriptions provides human-readable category names.
var categoryDescriptions = map[string]string{
	"security":      "Authentication and security headers",
	"logging":       "Request/response logging and tracing",
	"ratelimit":     "Rate limiting and flow control",
	"cache":         "Caching headers and strategies",
	"encoding":      "Compression and content encoding",
	"resilience":    "Circuit breaker, retry, timeout",
	"routing":       "URL rewriting, redirects, method override",
	"content":       "Static files, favicon, SPA",
	"api":           "JSON-RPC, GraphQL, validation",
	"session":       "Sessions, cookies, state",
	"observability": "Metrics, profiling, tracing",
	"misc":          "Other utilities",
}

// middlewares contains metadata for all available middlewares.
var middlewares = []middlewareInfo{
	// Security
	{
		Name:        "basicauth",
		Description: "HTTP Basic authentication",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/basicauth",
		QuickStart:  "app.Use(basicauth.New(basicauth.Accounts{\"admin\": \"secret\"}))",
		Related:     []string{"bearerauth", "keyauth", "jwt"},
	},
	{
		Name:        "bearerauth",
		Description: "Bearer token authentication",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/bearerauth",
		QuickStart:  "app.Use(bearerauth.New(validateToken))",
		Related:     []string{"basicauth", "jwt", "keyauth"},
	},
	{
		Name:        "cors",
		Description: "Cross-Origin Resource Sharing",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/cors",
		QuickStart:  "app.Use(cors.AllowAll())",
		Related:     []string{"cors2", "helmet", "csrf"},
	},
	{
		Name:        "cors2",
		Description: "CORS with additional options",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/cors2",
		QuickStart:  "app.Use(cors2.Default())",
		Related:     []string{"cors", "helmet"},
	},
	{
		Name:        "csrf",
		Description: "CSRF protection with tokens",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/csrf",
		QuickStart:  "app.Use(csrf.Default())",
		Related:     []string{"csrf2", "nonce", "helmet"},
	},
	{
		Name:        "csrf2",
		Description: "CSRF protection (alternative)",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/csrf2",
		QuickStart:  "app.Use(csrf2.New())",
		Related:     []string{"csrf", "nonce"},
	},
	{
		Name:        "helmet",
		Description: "Security headers",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/helmet",
		QuickStart:  "app.Use(helmet.Default())",
		Related:     []string{"cors", "csrf", "secure"},
	},
	{
		Name:        "ipfilter",
		Description: "IP-based access control",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/ipfilter",
		QuickStart:  "app.Use(ipfilter.Allow(\"192.168.1.0/24\"))",
		Related:     []string{"rbac", "keyauth"},
	},
	{
		Name:        "jwt",
		Description: "JSON Web Token validation",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/jwt",
		QuickStart:  "app.Use(jwt.New(jwt.Options{Secret: []byte(\"secret\")}))",
		Related:     []string{"bearerauth", "oauth2", "oidc"},
	},
	{
		Name:        "keyauth",
		Description: "API key authentication",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/keyauth",
		QuickStart:  "app.Use(keyauth.New(validateKey))",
		Related:     []string{"basicauth", "bearerauth", "jwt"},
	},
	{
		Name:        "nonce",
		Description: "CSP nonce generation",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/nonce",
		QuickStart:  "app.Use(nonce.New())",
		Related:     []string{"csrf", "helmet"},
	},
	{
		Name:        "oauth2",
		Description: "OAuth 2.0 authentication",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/oauth2",
		QuickStart:  "app.Use(oauth2.New(oauth2.Config{...}))",
		Related:     []string{"oidc", "jwt"},
	},
	{
		Name:        "oidc",
		Description: "OpenID Connect authentication",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/oidc",
		QuickStart:  "app.Use(oidc.New(oidc.Config{...}))",
		Related:     []string{"oauth2", "jwt"},
	},
	{
		Name:        "rbac",
		Description: "Role-based access control",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/rbac",
		QuickStart:  "app.Use(rbac.RequireRole(\"admin\"))",
		Related:     []string{"jwt", "keyauth"},
	},
	{
		Name:        "secure",
		Description: "General security middleware",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/secure",
		QuickStart:  "app.Use(secure.Default())",
		Related:     []string{"helmet", "cors"},
	},
	{
		Name:        "signature",
		Description: "Request signature validation",
		Category:    "security",
		Import:      "github.com/go-mizu/mizu/middlewares/signature",
		QuickStart:  "app.Use(signature.New(secret))",
		Related:     []string{"keyauth", "jwt"},
	},

	// Logging
	{
		Name:        "audit",
		Description: "Audit logging for compliance",
		Category:    "logging",
		Import:      "github.com/go-mizu/mizu/middlewares/audit",
		QuickStart:  "app.Use(audit.New(auditLogger))",
		Related:     []string{"logger", "trace"},
	},
	{
		Name:        "bodydump",
		Description: "Request/response body capture",
		Category:    "logging",
		Import:      "github.com/go-mizu/mizu/middlewares/bodydump",
		QuickStart:  "app.Use(bodydump.New(handler))",
		Related:     []string{"requestlog", "responselog"},
	},
	{
		Name:        "logger",
		Description: "HTTP request logging",
		Category:    "logging",
		Import:      "github.com/go-mizu/mizu/middlewares/logger",
		QuickStart:  "app.Use(logger.New())",
		Related:     []string{"requestlog", "trace"},
	},
	{
		Name:        "requestlog",
		Description: "Request body logging",
		Category:    "logging",
		Import:      "github.com/go-mizu/mizu/middlewares/requestlog",
		QuickStart:  "app.Use(requestlog.New())",
		Related:     []string{"logger", "responselog", "bodydump"},
	},
	{
		Name:        "responselog",
		Description: "Response body logging",
		Category:    "logging",
		Import:      "github.com/go-mizu/mizu/middlewares/responselog",
		QuickStart:  "app.Use(responselog.New())",
		Related:     []string{"logger", "requestlog", "bodydump"},
	},
	{
		Name:        "timing",
		Description: "Request timing headers",
		Category:    "logging",
		Import:      "github.com/go-mizu/mizu/middlewares/timing",
		QuickStart:  "app.Use(timing.New())",
		Related:     []string{"logger", "metrics"},
	},
	{
		Name:        "trace",
		Description: "Distributed request tracing",
		Category:    "logging",
		Import:      "github.com/go-mizu/mizu/middlewares/trace",
		QuickStart:  "app.Use(trace.New())",
		Related:     []string{"otel", "requestid"},
	},

	// Rate Limiting
	{
		Name:        "adaptive",
		Description: "Adaptive rate limiting",
		Category:    "ratelimit",
		Import:      "github.com/go-mizu/mizu/middlewares/adaptive",
		QuickStart:  "app.Use(adaptive.New())",
		Related:     []string{"ratelimit", "throttle"},
	},
	{
		Name:        "bulkhead",
		Description: "Bulkhead pattern for isolation",
		Category:    "ratelimit",
		Import:      "github.com/go-mizu/mizu/middlewares/bulkhead",
		QuickStart:  "app.Use(bulkhead.New(10))",
		Related:     []string{"concurrency", "circuitbreaker"},
	},
	{
		Name:        "concurrency",
		Description: "Concurrent request limiting",
		Category:    "ratelimit",
		Import:      "github.com/go-mizu/mizu/middlewares/concurrency",
		QuickStart:  "app.Use(concurrency.Limit(100))",
		Related:     []string{"bulkhead", "maxconns"},
	},
	{
		Name:        "maxconns",
		Description: "Maximum connections limit",
		Category:    "ratelimit",
		Import:      "github.com/go-mizu/mizu/middlewares/maxconns",
		QuickStart:  "app.Use(maxconns.New(1000))",
		Related:     []string{"concurrency", "ratelimit"},
	},
	{
		Name:        "ratelimit",
		Description: "Token bucket rate limiting",
		Category:    "ratelimit",
		Import:      "github.com/go-mizu/mizu/middlewares/ratelimit",
		QuickStart:  "app.Use(ratelimit.PerMinute(100))",
		Related:     []string{"throttle", "adaptive"},
	},
	{
		Name:        "throttle",
		Description: "Simple request throttling",
		Category:    "ratelimit",
		Import:      "github.com/go-mizu/mizu/middlewares/throttle",
		QuickStart:  "app.Use(throttle.New(100, time.Second))",
		Related:     []string{"ratelimit", "adaptive"},
	},

	// Cache
	{
		Name:        "cache",
		Description: "HTTP caching with Cache-Control",
		Category:    "cache",
		Import:      "github.com/go-mizu/mizu/middlewares/cache",
		QuickStart:  "app.Use(cache.New(time.Hour))",
		Related:     []string{"etag", "lastmodified"},
	},
	{
		Name:        "etag",
		Description: "ETag header generation",
		Category:    "cache",
		Import:      "github.com/go-mizu/mizu/middlewares/etag",
		QuickStart:  "app.Use(etag.New())",
		Related:     []string{"cache", "lastmodified"},
	},
	{
		Name:        "lastmodified",
		Description: "Last-Modified header support",
		Category:    "cache",
		Import:      "github.com/go-mizu/mizu/middlewares/lastmodified",
		QuickStart:  "app.Use(lastmodified.New())",
		Related:     []string{"cache", "etag"},
	},
	{
		Name:        "nocache",
		Description: "Disable caching headers",
		Category:    "cache",
		Import:      "github.com/go-mizu/mizu/middlewares/nocache",
		QuickStart:  "app.Use(nocache.New())",
		Related:     []string{"cache"},
	},
	{
		Name:        "surrogate",
		Description: "Surrogate/CDN cache headers",
		Category:    "cache",
		Import:      "github.com/go-mizu/mizu/middlewares/surrogate",
		QuickStart:  "app.Use(surrogate.New())",
		Related:     []string{"cache", "vary"},
	},
	{
		Name:        "vary",
		Description: "Vary header management",
		Category:    "cache",
		Import:      "github.com/go-mizu/mizu/middlewares/vary",
		QuickStart:  "app.Use(vary.New(\"Accept-Encoding\"))",
		Related:     []string{"cache", "compress"},
	},

	// Encoding
	{
		Name:        "compress",
		Description: "Response compression (gzip/deflate)",
		Category:    "encoding",
		Import:      "github.com/go-mizu/mizu/middlewares/compress",
		QuickStart:  "app.Use(compress.Default())",
		Related:     []string{"vary"},
	},
	{
		Name:        "contenttype",
		Description: "Content-Type validation",
		Category:    "encoding",
		Import:      "github.com/go-mizu/mizu/middlewares/contenttype",
		QuickStart:  "app.Use(contenttype.Require(\"application/json\"))",
		Related:     []string{"validator"},
	},
	{
		Name:        "msgpack",
		Description: "MessagePack encoding",
		Category:    "encoding",
		Import:      "github.com/go-mizu/mizu/middlewares/msgpack",
		QuickStart:  "app.Use(msgpack.New())",
		Related:     []string{"xml"},
	},
	{
		Name:        "xml",
		Description: "XML content handling",
		Category:    "encoding",
		Import:      "github.com/go-mizu/mizu/middlewares/xml",
		QuickStart:  "app.Use(xml.New())",
		Related:     []string{"msgpack"},
	},

	// Resilience
	{
		Name:        "circuitbreaker",
		Description: "Circuit breaker pattern",
		Category:    "resilience",
		Import:      "github.com/go-mizu/mizu/middlewares/circuitbreaker",
		QuickStart:  "app.Use(circuitbreaker.New())",
		Related:     []string{"retry", "timeout", "bulkhead"},
	},
	{
		Name:        "fallback",
		Description: "Fallback handler on errors",
		Category:    "resilience",
		Import:      "github.com/go-mizu/mizu/middlewares/fallback",
		QuickStart:  "app.Use(fallback.New(handler))",
		Related:     []string{"recover", "errorpage"},
	},
	{
		Name:        "hedge",
		Description: "Hedged requests for latency",
		Category:    "resilience",
		Import:      "github.com/go-mizu/mizu/middlewares/hedge",
		QuickStart:  "app.Use(hedge.New(100*time.Millisecond))",
		Related:     []string{"retry", "timeout"},
	},
	{
		Name:        "mirror",
		Description: "Request mirroring/shadowing",
		Category:    "resilience",
		Import:      "github.com/go-mizu/mizu/middlewares/mirror",
		QuickStart:  "app.Use(mirror.New(mirrorURL))",
		Related:     []string{"proxy"},
	},
	{
		Name:        "recover",
		Description: "Panic recovery",
		Category:    "resilience",
		Import:      "github.com/go-mizu/mizu/middlewares/recover",
		QuickStart:  "app.Use(recover.New())",
		Related:     []string{"errorpage", "sentry"},
	},
	{
		Name:        "retry",
		Description: "Automatic request retry",
		Category:    "resilience",
		Import:      "github.com/go-mizu/mizu/middlewares/retry",
		QuickStart:  "app.Use(retry.New(3))",
		Related:     []string{"circuitbreaker", "timeout"},
	},
	{
		Name:        "timeout",
		Description: "Request timeout",
		Category:    "resilience",
		Import:      "github.com/go-mizu/mizu/middlewares/timeout",
		QuickStart:  "app.Use(timeout.New(30*time.Second))",
		Related:     []string{"circuitbreaker", "retry"},
	},

	// Routing
	{
		Name:        "conditional",
		Description: "Conditional middleware execution",
		Category:    "routing",
		Import:      "github.com/go-mizu/mizu/middlewares/conditional",
		QuickStart:  "app.Use(conditional.If(condition, middleware))",
		Related:     []string{"filter"},
	},
	{
		Name:        "filter",
		Description: "Request filtering",
		Category:    "routing",
		Import:      "github.com/go-mizu/mizu/middlewares/filter",
		QuickStart:  "app.Use(filter.Path(\"/api/*\", middleware))",
		Related:     []string{"conditional"},
	},
	{
		Name:        "methodoverride",
		Description: "HTTP method override",
		Category:    "routing",
		Import:      "github.com/go-mizu/mizu/middlewares/methodoverride",
		QuickStart:  "app.Use(methodoverride.New())",
		Related:     []string{"rewrite"},
	},
	{
		Name:        "redirect",
		Description: "URL redirects",
		Category:    "routing",
		Import:      "github.com/go-mizu/mizu/middlewares/redirect",
		QuickStart:  "app.Use(redirect.HTTPS())",
		Related:     []string{"rewrite", "slash"},
	},
	{
		Name:        "rewrite",
		Description: "URL rewriting",
		Category:    "routing",
		Import:      "github.com/go-mizu/mizu/middlewares/rewrite",
		QuickStart:  "app.Use(rewrite.New(\"/old/*\", \"/new/$1\"))",
		Related:     []string{"redirect"},
	},
	{
		Name:        "slash",
		Description: "Trailing slash handling",
		Category:    "routing",
		Import:      "github.com/go-mizu/mizu/middlewares/slash",
		QuickStart:  "app.Use(slash.StripSlash())",
		Related:     []string{"redirect", "rewrite"},
	},

	// Content
	{
		Name:        "embed",
		Description: "Embedded file serving",
		Category:    "content",
		Import:      "github.com/go-mizu/mizu/middlewares/embed",
		QuickStart:  "app.Use(embed.New(fs))",
		Related:     []string{"static", "spa"},
	},
	{
		Name:        "errorpage",
		Description: "Custom error pages",
		Category:    "content",
		Import:      "github.com/go-mizu/mizu/middlewares/errorpage",
		QuickStart:  "app.Use(errorpage.New(pages))",
		Related:     []string{"recover", "fallback"},
	},
	{
		Name:        "favicon",
		Description: "Favicon serving",
		Category:    "content",
		Import:      "github.com/go-mizu/mizu/middlewares/favicon",
		QuickStart:  "app.Use(favicon.New(\"favicon.ico\"))",
		Related:     []string{"static"},
	},
	{
		Name:        "spa",
		Description: "Single-page application routing",
		Category:    "content",
		Import:      "github.com/go-mizu/mizu/middlewares/spa",
		QuickStart:  "app.Use(spa.New(\"./dist\"))",
		Related:     []string{"static", "embed"},
	},
	{
		Name:        "static",
		Description: "Static file serving",
		Category:    "content",
		Import:      "github.com/go-mizu/mizu/middlewares/static",
		QuickStart:  "app.Use(static.New(\"./public\"))",
		Related:     []string{"embed", "spa", "favicon"},
	},

	// API
	{
		Name:        "envelope",
		Description: "Response envelope wrapping",
		Category:    "api",
		Import:      "github.com/go-mizu/mizu/middlewares/envelope",
		QuickStart:  "app.Use(envelope.New())",
		Related:     []string{"transformer"},
	},
	{
		Name:        "graphql",
		Description: "GraphQL endpoint support",
		Category:    "api",
		Import:      "github.com/go-mizu/mizu/middlewares/graphql",
		QuickStart:  "app.Use(graphql.New(schema))",
		Related:     []string{"jsonrpc"},
	},
	{
		Name:        "hypermedia",
		Description: "Hypermedia links in responses",
		Category:    "api",
		Import:      "github.com/go-mizu/mizu/middlewares/hypermedia",
		QuickStart:  "app.Use(hypermedia.New())",
		Related:     []string{"envelope"},
	},
	{
		Name:        "jsonrpc",
		Description: "JSON-RPC 2.0 support",
		Category:    "api",
		Import:      "github.com/go-mizu/mizu/middlewares/jsonrpc",
		QuickStart:  "app.Use(jsonrpc.New(handler))",
		Related:     []string{"graphql"},
	},
	{
		Name:        "sanitizer",
		Description: "Input sanitization",
		Category:    "api",
		Import:      "github.com/go-mizu/mizu/middlewares/sanitizer",
		QuickStart:  "app.Use(sanitizer.New())",
		Related:     []string{"validator"},
	},
	{
		Name:        "transformer",
		Description: "Response transformation",
		Category:    "api",
		Import:      "github.com/go-mizu/mizu/middlewares/transformer",
		QuickStart:  "app.Use(transformer.New(transformFunc))",
		Related:     []string{"envelope"},
	},
	{
		Name:        "validator",
		Description: "Request validation",
		Category:    "api",
		Import:      "github.com/go-mizu/mizu/middlewares/validator",
		QuickStart:  "app.Use(validator.New())",
		Related:     []string{"sanitizer", "contenttype"},
	},

	// Session
	{
		Name:        "feature",
		Description: "Feature flags",
		Category:    "session",
		Import:      "github.com/go-mizu/mizu/middlewares/feature",
		QuickStart:  "app.Use(feature.New(flags))",
		Related:     []string{"canary"},
	},
	{
		Name:        "fingerprint",
		Description: "Client fingerprinting",
		Category:    "session",
		Import:      "github.com/go-mizu/mizu/middlewares/fingerprint",
		QuickStart:  "app.Use(fingerprint.New())",
		Related:     []string{"session"},
	},
	{
		Name:        "language",
		Description: "Language/locale detection",
		Category:    "session",
		Import:      "github.com/go-mizu/mizu/middlewares/language",
		QuickStart:  "app.Use(language.New())",
		Related:     []string{"timezone"},
	},
	{
		Name:        "session",
		Description: "Session management",
		Category:    "session",
		Import:      "github.com/go-mizu/mizu/middlewares/session",
		QuickStart:  "app.Use(session.New())",
		Related:     []string{"csrf", "fingerprint"},
	},
	{
		Name:        "timezone",
		Description: "Timezone detection",
		Category:    "session",
		Import:      "github.com/go-mizu/mizu/middlewares/timezone",
		QuickStart:  "app.Use(timezone.New())",
		Related:     []string{"language"},
	},

	// Observability
	{
		Name:        "expvar",
		Description: "Exported variables endpoint",
		Category:    "observability",
		Import:      "github.com/go-mizu/mizu/middlewares/expvar",
		QuickStart:  "app.Use(expvar.New())",
		Related:     []string{"pprof", "metrics"},
	},
	{
		Name:        "healthcheck",
		Description: "Health check endpoint",
		Category:    "observability",
		Import:      "github.com/go-mizu/mizu/middlewares/healthcheck",
		QuickStart:  "app.Use(healthcheck.New())",
		Related:     []string{"metrics"},
	},
	{
		Name:        "metrics",
		Description: "Request metrics collection",
		Category:    "observability",
		Import:      "github.com/go-mizu/mizu/middlewares/metrics",
		QuickStart:  "app.Use(metrics.New())",
		Related:     []string{"prometheus", "otel"},
	},
	{
		Name:        "otel",
		Description: "OpenTelemetry integration",
		Category:    "observability",
		Import:      "github.com/go-mizu/mizu/middlewares/otel",
		QuickStart:  "app.Use(otel.New())",
		Related:     []string{"trace", "prometheus"},
	},
	{
		Name:        "pprof",
		Description: "Go profiling endpoints",
		Category:    "observability",
		Import:      "github.com/go-mizu/mizu/middlewares/pprof",
		QuickStart:  "app.Use(pprof.New())",
		Related:     []string{"expvar", "metrics"},
	},
	{
		Name:        "prometheus",
		Description: "Prometheus metrics",
		Category:    "observability",
		Import:      "github.com/go-mizu/mizu/middlewares/prometheus",
		QuickStart:  "app.Use(prometheus.New())",
		Related:     []string{"metrics", "otel"},
	},
	{
		Name:        "sentry",
		Description: "Sentry error tracking",
		Category:    "observability",
		Import:      "github.com/go-mizu/mizu/middlewares/sentry",
		QuickStart:  "app.Use(sentry.New(dsn))",
		Related:     []string{"recover"},
	},

	// Misc
	{
		Name:        "bodyclose",
		Description: "Ensure request body closure",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/bodyclose",
		QuickStart:  "app.Use(bodyclose.New())",
		Related:     []string{"bodylimit"},
	},
	{
		Name:        "bodylimit",
		Description: "Request body size limit",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/bodylimit",
		QuickStart:  "app.Use(bodylimit.New(1024*1024))",
		Related:     []string{"requestsize"},
	},
	{
		Name:        "bot",
		Description: "Bot detection",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/bot",
		QuickStart:  "app.Use(bot.Block())",
		Related:     []string{"captcha", "honeypot"},
	},
	{
		Name:        "canary",
		Description: "Canary deployment routing",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/canary",
		QuickStart:  "app.Use(canary.New(percentage))",
		Related:     []string{"feature"},
	},
	{
		Name:        "captcha",
		Description: "CAPTCHA verification",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/captcha",
		QuickStart:  "app.Use(captcha.New(secret))",
		Related:     []string{"bot", "honeypot"},
	},
	{
		Name:        "chaos",
		Description: "Chaos engineering",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/chaos",
		QuickStart:  "app.Use(chaos.New())",
		Related:     []string{"mock"},
	},
	{
		Name:        "forwarded",
		Description: "Forwarded header parsing",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/forwarded",
		QuickStart:  "app.Use(forwarded.New())",
		Related:     []string{"realip"},
	},
	{
		Name:        "h2c",
		Description: "HTTP/2 cleartext support",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/h2c",
		QuickStart:  "app.Use(h2c.New())",
		Related:     []string{"websocket"},
	},
	{
		Name:        "header",
		Description: "Custom header setting",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/header",
		QuickStart:  "app.Use(header.Set(\"X-Custom\", \"value\"))",
		Related:     []string{"version"},
	},
	{
		Name:        "honeypot",
		Description: "Honeypot fields for bots",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/honeypot",
		QuickStart:  "app.Use(honeypot.New(\"hidden_field\"))",
		Related:     []string{"bot", "captcha"},
	},
	{
		Name:        "idempotency",
		Description: "Idempotent request handling",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/idempotency",
		QuickStart:  "app.Use(idempotency.New())",
		Related:     []string{"cache"},
	},
	{
		Name:        "keepalive",
		Description: "Keep-alive management",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/keepalive",
		QuickStart:  "app.Use(keepalive.New())",
		Related:     []string{"timeout"},
	},
	{
		Name:        "maintenance",
		Description: "Maintenance mode",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/maintenance",
		QuickStart:  "app.Use(maintenance.New(isEnabled))",
		Related:     []string{"feature"},
	},
	{
		Name:        "mock",
		Description: "Request mocking",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/mock",
		QuickStart:  "app.Use(mock.New(responses))",
		Related:     []string{"chaos"},
	},
	{
		Name:        "multitenancy",
		Description: "Multi-tenant support",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/multitenancy",
		QuickStart:  "app.Use(multitenancy.New())",
		Related:     []string{"session"},
	},
	{
		Name:        "proxy",
		Description: "Reverse proxy",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/proxy",
		QuickStart:  "app.Use(proxy.New(targetURL))",
		Related:     []string{"mirror"},
	},
	{
		Name:        "realip",
		Description: "Real client IP extraction",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/realip",
		QuickStart:  "app.Use(realip.New())",
		Related:     []string{"forwarded", "ipfilter"},
	},
	{
		Name:        "requestid",
		Description: "Unique request ID generation",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/requestid",
		QuickStart:  "app.Use(requestid.New())",
		Related:     []string{"trace"},
	},
	{
		Name:        "requestsize",
		Description: "Request size tracking",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/requestsize",
		QuickStart:  "app.Use(requestsize.New())",
		Related:     []string{"bodylimit", "responsesize"},
	},
	{
		Name:        "responsesize",
		Description: "Response size tracking",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/responsesize",
		QuickStart:  "app.Use(responsesize.New())",
		Related:     []string{"requestsize"},
	},
	{
		Name:        "sse",
		Description: "Server-Sent Events",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/sse",
		QuickStart:  "app.Use(sse.New())",
		Related:     []string{"websocket"},
	},
	{
		Name:        "version",
		Description: "API version headers",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/version",
		QuickStart:  "app.Use(version.New(\"1.0.0\"))",
		Related:     []string{"header"},
	},
	{
		Name:        "websocket",
		Description: "WebSocket support",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/websocket",
		QuickStart:  "app.Use(websocket.New(handler))",
		Related:     []string{"sse", "h2c"},
	},
	{
		Name:        "xrequestedwith",
		Description: "X-Requested-With validation",
		Category:    "misc",
		Import:      "github.com/go-mizu/mizu/middlewares/xrequestedwith",
		QuickStart:  "app.Use(xrequestedwith.Require())",
		Related:     []string{"csrf"},
	},
}

// getMiddlewares returns all middleware info.
func getMiddlewares() []middlewareInfo {
	return middlewares
}

// findMiddleware finds a middleware by name.
func findMiddleware(name string) *middlewareInfo {
	for i := range middlewares {
		if middlewares[i].Name == name {
			return &middlewares[i]
		}
	}
	return nil
}

// filterByCategory returns middlewares in a specific category.
func filterByCategory(mws []middlewareInfo, category string) []middlewareInfo {
	var result []middlewareInfo
	for _, mw := range mws {
		if mw.Category == category {
			result = append(result, mw)
		}
	}
	return result
}

// groupByCategory groups middlewares by their category.
func groupByCategory(mws []middlewareInfo) map[string][]middlewareInfo {
	result := make(map[string][]middlewareInfo)
	for _, mw := range mws {
		result[mw.Category] = append(result[mw.Category], mw)
	}
	return result
}
