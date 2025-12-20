# Alpine.js SPA Template Design Spec

## Overview

This specification describes the implementation of the Alpine.js SPA template for the Mizu CLI. Alpine.js is a lightweight JavaScript framework (~17kB minified) that offers reactive and declarative capabilities similar to Vue, but with a much smaller footprint and no build step required. The template uses Alpine.js with Vite for an optimal development experience and production builds.

## Goals

1. **Lightweight Performance**: Leverage Alpine.js's minimal footprint (~17kB minified, ~7kB gzipped)
2. **No Build Required (Optional)**: Alpine.js can work without a build step, but we use Vite for better DX
3. **Vue-like Syntax**: Familiar directive-based reactive programming
4. **Progressive Enhancement**: Works well for adding interactivity to HTML
5. **Mizu Integration**: Seamless integration with the `frontend` middleware
6. **Modern Tooling**: Vite for fast development and optimized builds

## Template Location

```
cmd/cli/templates/frontend/spa/alpine/
├── template.json              # Template metadata
├── Makefile.tmpl              # Build and development commands
├── cmd/server/
│   └── main.go.tmpl           # Go entry point
├── app/server/
│   ├── app.go.tmpl            # App setup with embedded frontend
│   ├── config.go.tmpl         # Server configuration
│   └── routes.go.tmpl         # API routes
├── client/
│   ├── package.json           # Alpine.js dependencies
│   ├── vite.config.ts         # Vite configuration
│   ├── tsconfig.json          # TypeScript configuration
│   ├── tsconfig.node.json     # TypeScript config for Vite
│   ├── index.html             # HTML template with Alpine directives
│   ├── src/
│   │   ├── main.ts            # Alpine.js entry point
│   │   ├── app.ts             # Alpine stores and data
│   │   ├── router.ts          # Client-side router
│   │   └── styles/
│   │       └── index.css      # Global styles
│   └── public/
│       └── vite.svg           # Favicon
└── dist/
    └── placeholder.txt        # Placeholder for build output
```

## Template Metadata

### template.json

```json
{
  "name": "frontend/spa/alpine",
  "description": "Alpine.js SPA with Vite and Mizu backend",
  "tags": ["go", "mizu", "frontend", "spa", "alpine", "vite", "typescript"],
  "variables": {
    "name": { "description": "Project name", "default": "" },
    "module": { "description": "Go module path", "default": "" }
  }
}
```

## Go Backend Files

The Go backend files are identical to other SPA templates, providing a consistent server experience:

### cmd/server/main.go.tmpl

```go
package main

import (
    "log"
    "os"

    "{{.Module}}/app/server"
)

func main() {
    cfg := server.LoadConfig()
    app := server.New(cfg)

    log.Printf("Starting server on :%s", cfg.Port)
    if err := app.Listen(":" + cfg.Port); err != nil {
        log.Fatal(err)
        os.Exit(1)
    }
}
```

### app/server/app.go.tmpl

```go
package server

import (
    "embed"
    "io/fs"

    "github.com/go-mizu/mizu"
    "github.com/go-mizu/mizu/middlewares/frontend"
)

//go:embed all:../../dist
var distFS embed.FS

// New creates a new Mizu app configured for the Alpine.js SPA.
func New(cfg *Config) *mizu.App {
    app := mizu.New()

    // API routes
    setupRoutes(app)

    // Frontend middleware (auto-detects dev/production mode)
    dist, _ := fs.Sub(distFS, "dist")
    app.Use(frontend.WithOptions(frontend.Options{
        Mode:        frontend.ModeAuto,
        FS:          dist,
        DevServer:   "http://localhost:" + cfg.DevPort,
        IgnorePaths: []string{"/api"},
    }))

    return app
}
```

### app/server/config.go.tmpl

```go
package server

import "os"

// Config holds server configuration.
type Config struct {
    Port    string
    DevPort string
    Env     string
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
    return &Config{
        Port:    getEnv("PORT", "3000"),
        DevPort: getEnv("DEV_PORT", "5173"),
        Env:     getEnv("MIZU_ENV", "development"),
    }
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

### app/server/routes.go.tmpl

```go
package server

import (
    "github.com/go-mizu/mizu"
)

func setupRoutes(app *mizu.App) {
    api := app.Prefix("/api")

    api.Get("/health", func(c *mizu.Ctx) error {
        return c.JSON(200, map[string]any{
            "status": "ok",
        })
    })

    api.Get("/hello", func(c *mizu.Ctx) error {
        return c.JSON(200, map[string]any{
            "message": "Hello from {{.Name}}!",
        })
    })
}
```

## Alpine.js Frontend Files

### client/package.json

```json
{
  "name": "{{.Name}}-client",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "alpinejs": "^3.14.3"
  },
  "devDependencies": {
    "typescript": "~5.6.3",
    "vite": "^6.0.5"
  }
}
```

### client/vite.config.ts

```typescript
import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../dist',
    emptyOutDir: true,
  },
})
```

### client/tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

### client/tsconfig.node.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2023"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["vite.config.ts"]
}
```

### client/index.html

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>{{.Name}}</title>
  </head>
  <body>
    <div id="app" x-data="app" x-cloak>
      <div class="app">
        <header class="header">
          <nav>
            <a href="/" @click.prevent="navigate('/')">Home</a>
            <a href="/about" @click.prevent="navigate('/about')">About</a>
          </nav>
        </header>
        <main class="main">
          <!-- Home Page -->
          <template x-if="route === '/'">
            <div class="page home">
              <h1>Welcome</h1>
              <template x-if="loading">
                <p>Loading...</p>
              </template>
              <template x-if="!loading">
                <p class="api-message" x-text="message"></p>
              </template>
            </div>
          </template>

          <!-- About Page -->
          <template x-if="route === '/about'">
            <div class="page about">
              <h1>About</h1>
              <p>
                This is an Alpine.js SPA powered by Mizu, a lightweight Go web framework.
              </p>
              <ul>
                <li>Alpine.js (~17kB minified, ~7kB gzipped)</li>
                <li>Vite for fast development and optimized builds</li>
                <li>Client-side routing with History API</li>
                <li>Mizu backend with the frontend middleware</li>
              </ul>
            </div>
          </template>
        </main>
        <footer class="footer">
          <p>Built with Mizu + Alpine.js</p>
        </footer>
      </div>
    </div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

### client/src/main.ts

```typescript
import Alpine from 'alpinejs'
import { createApp } from './app'
import './styles/index.css'

// Register the main app component
Alpine.data('app', createApp)

// Start Alpine
Alpine.start()

// Make Alpine available globally for debugging
declare global {
  interface Window {
    Alpine: typeof Alpine
  }
}
window.Alpine = Alpine
```

### client/src/app.ts

```typescript
export function createApp() {
  return {
    route: window.location.pathname,
    message: '',
    loading: true,

    init() {
      // Handle browser back/forward
      window.addEventListener('popstate', () => {
        this.route = window.location.pathname
        this.onRouteChange()
      })

      // Initial route load
      this.onRouteChange()
    },

    navigate(path: string) {
      if (this.route === path) return
      window.history.pushState({}, '', path)
      this.route = path
      this.onRouteChange()
    },

    onRouteChange() {
      if (this.route === '/') {
        this.fetchMessage()
      }
    },

    async fetchMessage() {
      this.loading = true
      try {
        const res = await fetch('/api/hello')
        const data = await res.json()
        this.message = data.message
      } catch (error) {
        this.message = 'Failed to load message'
      } finally {
        this.loading = false
      }
    }
  }
}
```

### client/src/vite-env.d.ts

```typescript
/// <reference types="vite/client" />

declare module 'alpinejs' {
  interface Alpine {
    data(name: string, callback: () => object): void
    start(): void
  }
  const Alpine: Alpine
  export default Alpine
}
```

### client/src/styles/index.css

Shared CSS for consistent look across all frontend templates:

```css
:root {
  --primary: #3b82f6;
  --primary-dark: #2563eb;
  --bg: #f8fafc;
  --text: #1e293b;
  --text-muted: #64748b;
  --border: #e2e8f0;
}

[x-cloak] {
  display: none !important;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: system-ui, -apple-system, sans-serif;
  background: var(--bg);
  color: var(--text);
  line-height: 1.6;
}

.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.header {
  background: white;
  border-bottom: 1px solid var(--border);
  padding: 1rem 2rem;
}

.header nav {
  display: flex;
  gap: 1.5rem;
}

.header a {
  color: var(--text);
  text-decoration: none;
  font-weight: 500;
  cursor: pointer;
}

.header a:hover {
  color: var(--primary);
}

.main {
  flex: 1;
  padding: 2rem;
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
}

.footer {
  background: white;
  border-top: 1px solid var(--border);
  padding: 1rem 2rem;
  text-align: center;
  color: var(--text-muted);
}

.page h1 {
  margin-bottom: 1rem;
}

.api-message {
  background: white;
  padding: 1rem;
  border-radius: 0.5rem;
  border: 1px solid var(--border);
}

.about ul {
  margin-top: 1rem;
  padding-left: 1.5rem;
}

.about li {
  margin-bottom: 0.5rem;
}
```

## Makefile

### Makefile.tmpl

```makefile
.PHONY: dev build run clean install

# Development mode: run Vite and Go server concurrently
dev:
	@echo "Starting development servers..."
	@(cd client && npm run dev) & \
	MIZU_ENV=development go run ./cmd/server

# Build frontend for production
build:
	@echo "Building frontend..."
	cd client && npm run build
	@echo "Build complete: dist/"

# Run production server
run: build
	@echo "Starting production server..."
	MIZU_ENV=production go run ./cmd/server

# Clean build artifacts
clean:
	rm -rf dist
	rm -rf client/node_modules

# Install dependencies
install:
	cd client && npm install
	go mod tidy
```

## CLI Usage

```bash
# Create a new Alpine.js SPA project
mizu new ./myapp --template frontend/spa --sub alpine

# List all templates (shows Alpine.js alongside React and Vue)
mizu new --list

# Preview the generated files
mizu new ./myapp --template frontend/spa --sub alpine --dry-run
```

## Testing Strategy

### Unit Tests

Add tests to `templates_test.go`:

```go
func TestListTemplatesIncludesAlpine(t *testing.T) {
    templates, err := listTemplates()
    if err != nil {
        t.Fatalf("listTemplates() error: %v", err)
    }

    // Alpine should be in frontend/spa sub-templates
    var frontendSPA *templateMeta
    for i, tmpl := range templates {
        if tmpl.Name == "frontend/spa" {
            frontendSPA = &templates[i]
            break
        }
    }

    if frontendSPA == nil {
        t.Fatal("frontend/spa template not found")
    }

    found := false
    for _, st := range frontendSPA.SubTemplates {
        if st.Name == "alpine" {
            found = true
            break
        }
    }

    if !found {
        t.Error("frontend/spa should have alpine sub-template")
    }
}

func TestTemplateExistsAlpine(t *testing.T) {
    if !templateExists("frontend/spa/alpine") {
        t.Error("templateExists('frontend/spa/alpine') returned false")
    }
}

func TestLoadTemplateFilesAlpine(t *testing.T) {
    files, err := loadTemplateFiles("frontend/spa/alpine")
    if err != nil {
        t.Fatalf("loadTemplateFiles() error: %v", err)
    }

    expectedFiles := []string{
        ".gitignore",
        "go.mod",
        "cmd/server/main.go",
        "app/server/app.go",
        "client/package.json",
        "client/vite.config.ts",
        "client/index.html",
        "client/src/main.ts",
        "client/src/app.ts",
        "Makefile",
    }

    fileMap := make(map[string]bool)
    for _, f := range files {
        fileMap[f.path] = true
    }

    for _, expected := range expectedFiles {
        if !fileMap[expected] {
            t.Errorf("expected file %q not found in template files", expected)
        }
    }
}

func TestBuildPlanAlpineTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/spa/alpine", tmpDir, vars)
    if err != nil {
        t.Fatalf("buildPlan() error: %v", err)
    }

    if len(p.files) == 0 {
        t.Error("buildPlan() produced empty plan")
    }
}

func TestApplyPlanAlpineTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/spa/alpine", tmpDir, vars)
    if err != nil {
        t.Fatalf("buildPlan() error: %v", err)
    }

    if err := p.apply(false); err != nil {
        t.Fatalf("apply() error: %v", err)
    }

    // Verify key files exist
    expectedFiles := []string{
        "go.mod",
        "cmd/server/main.go",
        "app/server/app.go",
        "client/package.json",
        "client/vite.config.ts",
        "client/index.html",
        "client/src/main.ts",
        "client/src/app.ts",
        "Makefile",
    }

    for _, file := range expectedFiles {
        path := filepath.Join(tmpDir, file)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            t.Errorf("expected file %q does not exist", file)
        }
    }
}

func TestAlpineTemplateVariableSubstitution(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myalpine", "example.com/myalpine", "MIT", nil)

    p, err := buildPlan("frontend/spa/alpine", tmpDir, vars)
    if err != nil {
        t.Fatalf("buildPlan() error: %v", err)
    }

    if err := p.apply(false); err != nil {
        t.Fatalf("apply() error: %v", err)
    }

    // Check go.mod has correct module path
    goModPath := filepath.Join(tmpDir, "go.mod")
    content, err := os.ReadFile(goModPath)
    if err != nil {
        t.Fatalf("failed to read go.mod: %v", err)
    }

    if !strings.Contains(string(content), "module example.com/myalpine") {
        t.Error("go.mod does not contain expected module path")
    }

    // Check package.json has correct name
    pkgPath := filepath.Join(tmpDir, "client/package.json")
    pkgContent, err := os.ReadFile(pkgPath)
    if err != nil {
        t.Fatalf("failed to read package.json: %v", err)
    }

    if !strings.Contains(string(pkgContent), `"name": "myalpine-client"`) {
        t.Error("package.json does not contain expected project name")
    }
}

func TestAlpineTemplateHasAlpineSpecificContent(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myalpine", "example.com/myalpine", "MIT", nil)

    p, err := buildPlan("frontend/spa/alpine", tmpDir, vars)
    if err != nil {
        t.Fatalf("buildPlan() error: %v", err)
    }

    if err := p.apply(false); err != nil {
        t.Fatalf("apply() error: %v", err)
    }

    // Verify index.html contains Alpine-specific syntax
    indexPath := filepath.Join(tmpDir, "client/index.html")
    content, err := os.ReadFile(indexPath)
    if err != nil {
        t.Fatalf("failed to read index.html: %v", err)
    }

    alpinePatterns := []string{
        "x-data",
        "x-cloak",
        "@click.prevent",
        "x-if",
        "x-text",
    }

    for _, pattern := range alpinePatterns {
        if !strings.Contains(string(content), pattern) {
            t.Errorf("index.html missing expected Alpine pattern: %s", pattern)
        }
    }

    // Verify main.ts uses Alpine
    mainPath := filepath.Join(tmpDir, "client/src/main.ts")
    mainContent, err := os.ReadFile(mainPath)
    if err != nil {
        t.Fatalf("failed to read main.ts: %v", err)
    }

    if !strings.Contains(string(mainContent), "import Alpine from 'alpinejs'") {
        t.Error("main.ts missing Alpine import")
    }

    if !strings.Contains(string(mainContent), "Alpine.start()") {
        t.Error("main.ts missing Alpine.start()")
    }

    // Verify package.json has Alpine dependency
    pkgPath := filepath.Join(tmpDir, "client/package.json")
    pkgContent, err := os.ReadFile(pkgPath)
    if err != nil {
        t.Fatalf("failed to read package.json: %v", err)
    }

    if !strings.Contains(string(pkgContent), `"alpinejs"`) {
        t.Error("package.json missing alpinejs dependency")
    }
}
```

## Key Differences from Other SPA Templates

| Aspect | React/Vue/Preact | Alpine.js |
|--------|------------------|-----------|
| Bundle Size | 30-100kB | ~17kB minified |
| Approach | Virtual DOM | Real DOM with reactivity |
| Components | JSX/SFC | HTML with directives |
| Build Step | Required | Optional (we use Vite for DX) |
| TypeScript | Native support | Via bundler |
| Routing | Router library | Simple History API |
| Learning Curve | Higher | Lower (Vue-like syntax) |

## Implementation Notes

1. **Directive-based**: Alpine uses `x-` prefixed attributes for reactivity (similar to Vue's `v-`)
2. **x-cloak**: Hides elements until Alpine initializes (prevents flash of unstyled content)
3. **No JSX**: All logic is in HTML attributes and separate TypeScript files
4. **Alpine.data()**: Registers reusable component data/methods
5. **History API Routing**: Simple SPA routing without a separate library
6. **Same Backend**: Go backend is identical to other SPA templates (framework-agnostic)

## Why Alpine.js

Alpine.js is an excellent choice for SPAs that prioritize:

1. **Simplicity**: No build step required, just a script tag (though we use Vite for better DX)
2. **Small Bundle**: ~17kB minified, ~7kB gzipped
3. **Vue-like Syntax**: Familiar directive-based reactive programming
4. **Progressive Enhancement**: Easy to add to existing HTML
5. **Low Learning Curve**: No complex concepts like Virtual DOM or JSX
6. **Quick Prototyping**: Fast to get started, minimal boilerplate
7. **Server-Side Friendly**: Works great with server-rendered HTML

## Alpine.js vs HTMX

Both are lightweight alternatives to React/Vue, but they serve different use cases:

| Aspect | Alpine.js | HTMX |
|--------|-----------|------|
| Focus | Client-side reactivity | Server-driven updates |
| Rendering | Client-side | Server-side |
| State | Client-side stores | Minimal client state |
| API Style | JSON APIs | HTML over the wire |
| Use Case | SPAs with client logic | MPAs with enhanced UX |

Alpine.js is better for traditional SPA patterns where you want client-side routing and state management, while HTMX is better for server-rendered applications with progressive enhancement.
