# Frontend/SPA Package Design Spec

## Overview

The `frontend` package provides a comprehensive solution for integrating modern frontend frameworks with Mizu applications. It offers best-in-class developer experience for both development (with hot reload proxy) and production (with optimized asset serving).

## Goals

1. **Best DX**: Zero-config for common cases, powerful configuration for advanced scenarios
2. **Framework Agnostic**: Works with React, Vue, Svelte, Solid, Angular, Astro, and any Vite-based framework
3. **Production Ready**: Optimized caching, compression, security headers, and performance
4. **Development First**: Seamless dev server proxy with HMR, WebSocket forwarding, and error overlays
5. **Full Feature Set**: SSR support, asset pipeline, env injection, source maps, and service workers

## Package Structure

```
frontend/
├── frontend.go       # Core types and middleware
├── dev.go            # Development server proxy
├── static.go         # Production static serving
├── manifest.go       # Build manifest parsing (Vite, Webpack, etc.)
├── inject.go         # HTML/env injection utilities
├── ssr.go            # Server-side rendering support
├── adapters/
│   ├── react.go      # React-specific optimizations
│   ├── vue.go        # Vue-specific optimizations
│   ├── svelte.go     # Svelte-specific optimizations
│   ├── solid.go      # Solid-specific optimizations
│   └── astro.go      # Astro-specific optimizations
├── frontend_test.go
└── doc.go
```

## Core API Design

### Options Pattern

```go
package frontend

// Mode determines how the frontend is served.
type Mode string

const (
    ModeDev        Mode = "dev"        // Proxy to dev server
    ModeProduction Mode = "production" // Serve from dist
    ModeAuto       Mode = "auto"       // Auto-detect based on environment
)

// Options configures the frontend middleware.
type Options struct {
    // Mode determines serving behavior.
    // Default: ModeAuto (uses DEV if MIZU_ENV != "production")
    Mode Mode

    // --- Production Options ---

    // Root is the directory containing built assets.
    // Default: "dist"
    Root string

    // FS is an embedded filesystem for production builds.
    // Takes precedence over Root.
    FS fs.FS

    // Index is the fallback file for SPA routing.
    // Default: "index.html"
    Index string

    // Prefix is the URL prefix for serving (e.g., "/app").
    // Default: "/"
    Prefix string

    // IgnorePaths are paths that bypass SPA fallback.
    // Default: []string{"/api", "/health", "/metrics"}
    IgnorePaths []string

    // CacheControl configures caching strategy.
    CacheControl CacheConfig

    // --- Development Options ---

    // DevServer is the development server URL.
    // Default: "http://localhost:5173" (Vite default)
    DevServer string

    // DevServerTimeout is the timeout for dev server requests.
    // Default: 30s
    DevServerTimeout time.Duration

    // ProxyWebSocket enables WebSocket proxying for HMR.
    // Default: true
    ProxyWebSocket bool

    // --- Advanced Options ---

    // Manifest is the path to build manifest (vite/webpack).
    // Used for asset fingerprinting and preloading.
    Manifest string

    // InjectEnv injects environment variables into index.html.
    // Variables are exposed as window.__ENV__
    InjectEnv []string

    // InjectMeta adds custom meta tags to index.html.
    InjectMeta map[string]string

    // SecurityHeaders adds recommended security headers.
    // Default: true
    SecurityHeaders bool

    // Compression enables gzip/brotli compression.
    // Default: true in production
    Compression bool

    // SourceMaps controls source map serving.
    // Default: true in dev, false in production
    SourceMaps *bool

    // ServiceWorker is the path to service worker file.
    // If set, proper headers are added for SW scope.
    ServiceWorker string

    // SSR enables server-side rendering support.
    SSR *SSRConfig

    // ErrorHandler handles errors in development mode.
    ErrorHandler func(*mizu.Ctx, error) error

    // NotFoundHandler handles 404s before SPA fallback.
    // Return nil to continue to SPA fallback.
    NotFoundHandler func(*mizu.Ctx) error
}

// CacheConfig configures caching behavior.
type CacheConfig struct {
    // Static assets with hash in filename (e.g., app.a1b2c3.js)
    // Default: 1 year (immutable)
    HashedAssets time.Duration

    // Unhashed assets (e.g., images, fonts without hash)
    // Default: 1 week
    UnhashedAssets time.Duration

    // Index/HTML files
    // Default: no-cache (always revalidate)
    HTML time.Duration

    // Custom patterns: map of glob pattern to duration
    // e.g., {"*.woff2": 30 * 24 * time.Hour}
    Patterns map[string]time.Duration
}

// SSRConfig configures server-side rendering.
type SSRConfig struct {
    // Entry is the SSR entry point (e.g., "dist/server/entry.js")
    Entry string

    // Renderer is the SSR render function.
    // For Node-based SSR, use NodeRenderer().
    // For Go-based SSR (via wasm), use WasmRenderer().
    Renderer SSRRenderer

    // FallbackToCSR falls back to client-side rendering on SSR errors.
    // Default: true
    FallbackToCSR bool

    // Cache caches rendered pages.
    Cache SSRCache
}
```

### Constructors

```go
// New creates frontend middleware with sensible defaults.
// Auto-detects mode based on MIZU_ENV environment variable.
func New(root string) mizu.Middleware

// Dev creates development-only middleware that proxies to dev server.
func Dev(devServerURL string) mizu.Middleware

// WithFS creates production middleware with embedded filesystem.
func WithFS(fsys fs.FS) mizu.Middleware

// WithOptions creates middleware with full configuration.
func WithOptions(opts Options) mizu.Middleware
```

### Usage Examples

#### Basic SPA Serving (Production)

```go
package main

import (
    "embed"
    "github.com/go-mizu/mizu"
    "github.com/go-mizu/mizu/frontend"
)

//go:embed dist/*
var distFS embed.FS

func main() {
    app := mizu.New()

    // API routes first (won't fallback to SPA)
    app.Get("/api/users", listUsers)
    app.Post("/api/users", createUser)

    // Frontend middleware last (catches all other routes)
    app.Use(frontend.WithFS(distFS))

    app.Listen(":3000")
}
```

#### Development with Vite

```go
func main() {
    app := mizu.New()

    // API routes
    api := app.Prefix("/api")
    api.Get("/users", listUsers)

    // Development: proxy to Vite dev server
    // Production: serve from dist
    app.Use(frontend.New("./dist"))

    app.Listen(":3000")
}
```

#### Advanced Configuration

```go
app.Use(frontend.WithOptions(frontend.Options{
    Mode:     frontend.ModeAuto,
    Root:     "./client/dist",
    Prefix:   "/app",
    DevServer: "http://localhost:3000",

    IgnorePaths: []string{"/api", "/auth", "/webhooks"},

    CacheControl: frontend.CacheConfig{
        HashedAssets:   365 * 24 * time.Hour, // 1 year
        UnhashedAssets: 7 * 24 * time.Hour,   // 1 week
        HTML:           0,                     // no-cache
    },

    InjectEnv: []string{
        "API_URL",
        "ANALYTICS_ID",
        "FEATURE_FLAGS",
    },

    SecurityHeaders: true,
    Compression:     true,
}))
```

#### Server-Side Rendering

```go
app.Use(frontend.WithOptions(frontend.Options{
    Root: "./dist/client",
    SSR: &frontend.SSRConfig{
        Entry:         "./dist/server/entry.js",
        Renderer:      frontend.NodeRenderer(),
        FallbackToCSR: true,
        Cache:         frontend.LRUCache(1000),
    },
}))
```

## Development Server Proxy

The dev server proxy provides seamless integration with modern build tools:

### Features

1. **HTTP Proxy**: Forwards all non-API requests to dev server
2. **WebSocket Proxy**: Forwards HMR WebSocket connections
3. **Error Handling**: Shows friendly errors when dev server is down
4. **Auto-retry**: Automatically reconnects when dev server restarts
5. **Request Rewriting**: Handles path prefixes and rewrites

### Implementation

```go
// dev.go

type devProxy struct {
    target       *url.URL
    httpProxy    *httputil.ReverseProxy
    wsDialer     *websocket.Dialer
    timeout      time.Duration
    retryBackoff time.Duration
}

func newDevProxy(target string, timeout time.Duration) (*devProxy, error) {
    u, err := url.Parse(target)
    if err != nil {
        return nil, err
    }

    proxy := &devProxy{
        target:       u,
        timeout:      timeout,
        retryBackoff: 100 * time.Millisecond,
    }

    proxy.httpProxy = &httputil.ReverseProxy{
        Director: proxy.director,
        ModifyResponse: proxy.modifyResponse,
        ErrorHandler: proxy.errorHandler,
        Transport: &http.Transport{
            DialContext: (&net.Dialer{
                Timeout:   timeout,
                KeepAlive: 30 * time.Second,
            }).DialContext,
            MaxIdleConns:        100,
            IdleConnTimeout:     90 * time.Second,
            TLSHandshakeTimeout: 10 * time.Second,
        },
    }

    return proxy, nil
}

func (p *devProxy) ServeHTTP(c *mizu.Ctx) error {
    // Check if WebSocket upgrade
    if isWebSocketRequest(c.Request()) {
        return p.proxyWebSocket(c)
    }

    p.httpProxy.ServeHTTP(c.Writer(), c.Request())
    return nil
}

func (p *devProxy) proxyWebSocket(c *mizu.Ctx) error {
    // Upgrade client connection
    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }

    clientConn, err := upgrader.Upgrade(c.Writer(), c.Request(), nil)
    if err != nil {
        return err
    }
    defer clientConn.Close()

    // Connect to dev server
    serverURL := *p.target
    serverURL.Scheme = "ws"
    if p.target.Scheme == "https" {
        serverURL.Scheme = "wss"
    }
    serverURL.Path = c.Request().URL.Path

    serverConn, _, err := p.wsDialer.Dial(serverURL.String(), nil)
    if err != nil {
        return err
    }
    defer serverConn.Close()

    // Bidirectional proxy
    return p.pipeWebSocket(clientConn, serverConn)
}
```

### Dev Server Detection

```go
// Auto-detect common dev server ports
var defaultDevServers = map[string]string{
    "vite":     "http://localhost:5173",
    "webpack":  "http://localhost:8080",
    "next":     "http://localhost:3000",
    "nuxt":     "http://localhost:3000",
    "angular":  "http://localhost:4200",
    "svelte":   "http://localhost:5173",
    "astro":    "http://localhost:4321",
}

func detectDevServer() string {
    // Check environment variable first
    if url := os.Getenv("FRONTEND_DEV_SERVER"); url != "" {
        return url
    }

    // Check for running dev servers
    for _, url := range defaultDevServers {
        if isServerRunning(url) {
            return url
        }
    }

    return "http://localhost:5173" // Vite default
}
```

## Build Manifest Integration

### Vite Manifest

```go
// manifest.go

// ViteManifest represents a Vite build manifest.
type ViteManifest map[string]ViteChunk

type ViteChunk struct {
    File           string   `json:"file"`
    Name           string   `json:"name"`
    Src            string   `json:"src"`
    IsEntry        bool     `json:"isEntry"`
    IsDynamicEntry bool     `json:"isDynamicEntry"`
    Imports        []string `json:"imports"`
    DynamicImports []string `json:"dynamicImports"`
    CSS            []string `json:"css"`
    Assets         []string `json:"assets"`
}

// LoadViteManifest loads and parses a Vite manifest.
func LoadViteManifest(fsys fs.FS, path string) (*Manifest, error) {
    data, err := fs.ReadFile(fsys, path)
    if err != nil {
        return nil, err
    }

    var vm ViteManifest
    if err := json.Unmarshal(data, &vm); err != nil {
        return nil, err
    }

    return &Manifest{
        entries:  buildEntryMap(vm),
        chunks:   vm,
        preloads: buildPreloadMap(vm),
    }, nil
}

// Manifest provides access to build artifacts.
type Manifest struct {
    entries  map[string]string
    chunks   ViteManifest
    preloads map[string][]string
}

// Entry returns the output file for an entry point.
func (m *Manifest) Entry(name string) string {
    return m.entries[name]
}

// Preloads returns module preload hints for an entry.
func (m *Manifest) Preloads(entry string) []string {
    return m.preloads[entry]
}

// PreloadTags generates <link rel="modulepreload"> tags.
func (m *Manifest) PreloadTags(entry string) template.HTML {
    var b strings.Builder
    for _, path := range m.Preloads(entry) {
        b.WriteString(`<link rel="modulepreload" href="`)
        b.WriteString(path)
        b.WriteString(`">` + "\n")
    }
    return template.HTML(b.String())
}

// CSSTags generates <link rel="stylesheet"> tags.
func (m *Manifest) CSSTags(entry string) template.HTML {
    chunk := m.chunks[entry]
    var b strings.Builder
    for _, css := range chunk.CSS {
        b.WriteString(`<link rel="stylesheet" href="/`)
        b.WriteString(css)
        b.WriteString(`">` + "\n")
    }
    return template.HTML(b.String())
}
```

### Webpack Manifest

```go
type WebpackManifest struct {
    assets map[string]string
}

func LoadWebpackManifest(fsys fs.FS, path string) (*Manifest, error) {
    // Similar implementation for webpack-manifest-plugin format
}
```

## HTML Injection

### Environment Variable Injection

```go
// inject.go

// InjectEnv injects environment variables into HTML.
func InjectEnv(html []byte, vars []string) []byte {
    env := make(map[string]string)
    for _, key := range vars {
        if val := os.Getenv(key); val != "" {
            env[key] = val
        }
    }

    if len(env) == 0 {
        return html
    }

    script := fmt.Sprintf(
        `<script>window.__ENV__=%s;</script>`,
        mustJSON(env),
    )

    // Insert before </head>
    return insertBeforeTag(html, "</head>", script)
}

// InjectMeta injects meta tags into HTML.
func InjectMeta(html []byte, meta map[string]string) []byte {
    var b strings.Builder
    for name, content := range meta {
        b.WriteString(`<meta name="`)
        b.WriteString(template.HTMLEscapeString(name))
        b.WriteString(`" content="`)
        b.WriteString(template.HTMLEscapeString(content))
        b.WriteString(`">` + "\n")
    }

    return insertBeforeTag(html, "</head>", b.String())
}

// InjectPreloads injects resource preload hints.
func InjectPreloads(html []byte, manifest *Manifest, entry string) []byte {
    preloads := manifest.PreloadTags(entry)
    css := manifest.CSSTags(entry)
    return insertBeforeTag(html, "</head>", string(preloads)+string(css))
}
```

## Caching Strategy

### Asset Classification

```go
// static.go

type assetType int

const (
    assetHashed   assetType = iota // app.a1b2c3d4.js
    assetUnhashed                  // logo.png
    assetHTML                      // index.html
    assetMap                       // app.js.map
)

// classifyAsset determines the asset type for caching.
func classifyAsset(path string) assetType {
    if strings.HasSuffix(path, ".html") {
        return assetHTML
    }
    if strings.HasSuffix(path, ".map") {
        return assetMap
    }

    // Check for content hash pattern: name.HASH.ext
    // Common patterns: [name].[hash].js, [name]-[hash].css
    base := filepath.Base(path)
    ext := filepath.Ext(base)
    name := strings.TrimSuffix(base, ext)

    // Match patterns like: app.a1b2c3d4, vendor-abc123
    if hashPattern.MatchString(name) {
        return assetHashed
    }

    return assetUnhashed
}

var hashPattern = regexp.MustCompile(`[.-][a-f0-9]{6,}$`)

// cacheHeaders returns cache headers for asset type.
func cacheHeaders(t assetType, cfg CacheConfig) string {
    switch t {
    case assetHashed:
        if cfg.HashedAssets > 0 {
            return fmt.Sprintf("public, max-age=%d, immutable", int(cfg.HashedAssets.Seconds()))
        }
        return "public, max-age=31536000, immutable"
    case assetUnhashed:
        if cfg.UnhashedAssets > 0 {
            return fmt.Sprintf("public, max-age=%d", int(cfg.UnhashedAssets.Seconds()))
        }
        return "public, max-age=604800"
    case assetHTML:
        return "no-cache, no-store, must-revalidate"
    case assetMap:
        return "no-cache"
    }
    return ""
}
```

## Security Headers

```go
// Default security headers for SPA
var defaultSecurityHeaders = map[string]string{
    "X-Content-Type-Options": "nosniff",
    "X-Frame-Options":        "SAMEORIGIN",
    "X-XSS-Protection":       "1; mode=block",
    "Referrer-Policy":        "strict-origin-when-cross-origin",
}

// Additional headers for production
var prodSecurityHeaders = map[string]string{
    "Strict-Transport-Security": "max-age=31536000; includeSubDomains",
}

func applySecurityHeaders(w http.ResponseWriter, isProd bool) {
    for k, v := range defaultSecurityHeaders {
        w.Header().Set(k, v)
    }
    if isProd {
        for k, v := range prodSecurityHeaders {
            w.Header().Set(k, v)
        }
    }
}
```

## Server-Side Rendering

### SSR Interface

```go
// ssr.go

// SSRRenderer renders pages on the server.
type SSRRenderer interface {
    Render(ctx context.Context, url string, data any) (SSRResult, error)
    Close() error
}

// SSRResult is the result of server-side rendering.
type SSRResult struct {
    HTML        string
    Head        string
    InitialData any
    StatusCode  int
    Redirect    string
}

// SSRCache caches rendered pages.
type SSRCache interface {
    Get(key string) (SSRResult, bool)
    Set(key string, result SSRResult, ttl time.Duration)
}
```

### Node.js Renderer

```go
// NodeRenderer creates an SSR renderer using Node.js.
func NodeRenderer(entry string) SSRRenderer {
    return &nodeRenderer{
        entry: entry,
        pool:  newNodePool(runtime.NumCPU()),
    }
}

type nodeRenderer struct {
    entry string
    pool  *nodePool
}

func (r *nodeRenderer) Render(ctx context.Context, url string, data any) (SSRResult, error) {
    worker := r.pool.Get()
    defer r.pool.Put(worker)

    return worker.Render(ctx, r.entry, url, data)
}
```

### Go/WASM Renderer (for frameworks with WASM SSR)

```go
// WasmRenderer creates an SSR renderer using WebAssembly.
func WasmRenderer(wasmPath string) SSRRenderer {
    return &wasmRenderer{
        wasmPath: wasmPath,
    }
}
```

## Framework Adapters

### React Adapter

```go
// adapters/react.go

// React provides React-specific optimizations.
func React(opts Options) mizu.Middleware {
    opts = applyReactDefaults(opts)

    return func(next mizu.Handler) mizu.Handler {
        return func(c *mizu.Ctx) error {
            // Add React-specific headers
            if opts.Mode != ModeDev {
                // Enable concurrent features
                c.Writer().Header().Set("Document-Policy", "js-profiling")
            }

            return WithOptions(opts)(next)(c)
        }
    }
}

func applyReactDefaults(opts Options) Options {
    if opts.InjectMeta == nil {
        opts.InjectMeta = make(map[string]string)
    }
    // React DevTools detection
    if opts.Mode == ModeDev {
        opts.InjectMeta["react-devtools-enabled"] = "true"
    }
    return opts
}
```

### Vue Adapter

```go
// adapters/vue.go

func Vue(opts Options) mizu.Middleware {
    opts = applyVueDefaults(opts)
    return WithOptions(opts)
}

func applyVueDefaults(opts Options) Options {
    // Vue-specific manifest path
    if opts.Manifest == "" {
        opts.Manifest = ".vite/manifest.json"
    }
    return opts
}
```

## Helper Functions

### Template Helpers for View Integration

```go
// Integration with view package

// ViewHelpers returns template functions for frontend integration.
func ViewHelpers(manifest *Manifest) template.FuncMap {
    return template.FuncMap{
        "vite_entry": func(name string) template.HTML {
            chunk := manifest.chunks[name]
            var b strings.Builder

            // CSS
            for _, css := range chunk.CSS {
                b.WriteString(`<link rel="stylesheet" href="/`)
                b.WriteString(css)
                b.WriteString(`">` + "\n")
            }

            // Preloads
            for _, imp := range chunk.Imports {
                b.WriteString(`<link rel="modulepreload" href="/`)
                b.WriteString(manifest.chunks[imp].File)
                b.WriteString(`">` + "\n")
            }

            // Main script
            b.WriteString(`<script type="module" src="/`)
            b.WriteString(chunk.File)
            b.WriteString(`"></script>`)

            return template.HTML(b.String())
        },

        "vite_asset": func(path string) string {
            if chunk, ok := manifest.chunks[path]; ok {
                return "/" + chunk.File
            }
            return "/" + path
        },

        "env_json": func(keys ...string) template.JS {
            env := make(map[string]string)
            for _, k := range keys {
                if v := os.Getenv(k); v != "" {
                    env[k] = v
                }
            }
            data, _ := json.Marshal(env)
            return template.JS(data)
        },
    }
}
```

## Performance Considerations

1. **Lazy Loading**: Manifest is loaded once on startup, not per-request
2. **Compiled Regexes**: All patterns compiled at init time
3. **HTTP/2 Push**: Support for pushing critical assets (when available)
4. **Preload Hints**: Automatic `<link rel="modulepreload">` generation
5. **Compression**: Built-in gzip/brotli support via composition with compress middleware
6. **Caching**: Proper ETags and Last-Modified headers via http.ServeContent

## Error Handling

### Development Error Overlay

```go
func devErrorPage(err error, target string) []byte {
    return []byte(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Dev Server Error</title>
    <style>
        body { font-family: system-ui; padding: 40px; background: #1a1a1a; color: #fff; }
        .error { background: #2d1b1b; border: 1px solid #5c2626; padding: 20px; border-radius: 8px; }
        h1 { color: #ff6b6b; margin: 0 0 10px 0; }
        code { background: #333; padding: 2px 6px; border-radius: 4px; }
        .retry { margin-top: 20px; padding: 10px 20px; background: #4a4a4a; border: none; color: #fff; cursor: pointer; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="error">
        <h1>Unable to connect to dev server</h1>
        <p>Could not connect to <code>%s</code></p>
        <p>Make sure your development server is running:</p>
        <pre>npm run dev</pre>
        <button class="retry" onclick="location.reload()">Retry</button>
    </div>
    <script>
        // Auto-retry every 2 seconds
        setTimeout(() => location.reload(), 2000);
    </script>
</body>
</html>`, target))
}
```

## Testing Utilities

```go
// testing.go

// TestFS creates an in-memory filesystem for testing.
func TestFS(files map[string]string) fs.FS {
    return fstest.MapFS(toMapFS(files))
}

// TestManifest creates a test manifest.
func TestManifest(entries map[string]string) *Manifest {
    // Helper for testing manifest-dependent code
}
```

## Implementation Plan

### Phase 1: Core (This PR)

1. Basic middleware with Options pattern
2. Production static file serving with SPA fallback
3. Caching strategy implementation
4. Security headers

### Phase 2: Development Server

1. HTTP proxy implementation
2. WebSocket proxy for HMR
3. Error handling and auto-retry
4. Dev server detection

### Phase 3: Build Integration

1. Vite manifest parsing
2. Webpack manifest parsing
3. Asset fingerprinting detection
4. Preload generation

### Phase 4: Advanced Features

1. HTML injection (env, meta)
2. SSR support
3. Framework adapters
4. View package integration

## File Locations

```
middlewares/frontend/
├── frontend.go          # Core types, Options, constructors
├── dev.go               # Development server proxy
├── static.go            # Production static serving
├── manifest.go          # Build manifest parsing
├── inject.go            # HTML injection utilities
├── cache.go             # Caching strategy
├── security.go          # Security headers
├── ssr.go               # SSR support
├── adapters/
│   ├── adapters.go      # Common adapter utilities
│   ├── react.go
│   ├── vue.go
│   └── svelte.go
├── frontend_test.go
└── doc.go
```
