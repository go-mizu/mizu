# Svelte SPA Template Design Spec

## Overview

This specification describes the implementation of the Svelte SPA template for the Mizu CLI. The template follows the same structure as the React and Vue templates but uses Svelte 5 with TypeScript, svelte-routing for client-side navigation, and Vite for the build system.

## Goals

1. **Modern Svelte Setup**: Svelte 5 with TypeScript and Vite
2. **Consistent Structure**: Mirror the React/Vue template structure for familiarity
3. **Svelte Best Practices**: Use runes, proper TypeScript support, and component patterns
4. **Mizu Integration**: Seamless integration with the `frontend` middleware
5. **Developer Experience**: Hot reload, dev/production modes, and proper tooling
6. **Lightweight**: Leverage Svelte's compile-time approach for minimal runtime

## Template Location

```
cmd/cli/templates/frontend/spa/svelte/
├── template.json              # Template metadata
├── Makefile.tmpl              # Build and development commands
├── cmd/server/
│   └── main.go.tmpl           # Go entry point
├── app/server/
│   ├── app.go.tmpl            # App setup with embedded frontend
│   ├── config.go.tmpl         # Server configuration
│   └── routes.go.tmpl         # API routes
├── client/
│   ├── package.json           # Svelte dependencies
│   ├── vite.config.ts         # Vite configuration
│   ├── tsconfig.json          # TypeScript configuration
│   ├── tsconfig.node.json     # TypeScript config for Vite
│   ├── svelte.config.js       # Svelte configuration
│   ├── index.html             # HTML template
│   ├── src/
│   │   ├── main.ts            # Svelte app entry point
│   │   ├── App.svelte         # Root component with router
│   │   ├── vite-env.d.ts      # Vite TypeScript declarations
│   │   ├── components/
│   │   │   └── Layout.svelte  # Layout with navigation
│   │   ├── pages/
│   │   │   ├── Home.svelte    # Home page with API call
│   │   │   └── About.svelte   # About page
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
  "name": "frontend/spa/svelte",
  "description": "Svelte SPA with Vite, TypeScript, and Mizu backend",
  "tags": ["go", "mizu", "frontend", "spa", "svelte", "vite", "typescript"],
  "variables": {
    "name": { "description": "Project name", "default": "" },
    "module": { "description": "Go module path", "default": "" }
  }
}
```

## Go Backend Files

The Go backend files are identical to React/Vue templates, providing a consistent server experience:

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

// New creates a new Mizu app configured for the Svelte SPA.
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

## Svelte Frontend Files

### client/package.json

```json
{
  "name": "{{.Name}}-client",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview",
    "check": "svelte-check --tsconfig ./tsconfig.json"
  },
  "dependencies": {
    "svelte-routing": "^2.13.0"
  },
  "devDependencies": {
    "@sveltejs/vite-plugin-svelte": "^5.0.3",
    "@tsconfig/svelte": "^5.0.4",
    "svelte": "^5.16.0",
    "svelte-check": "^4.1.1",
    "typescript": "~5.6.3",
    "vite": "^6.0.5"
  }
}
```

### client/vite.config.ts

```typescript
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
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

### client/svelte.config.js

```javascript
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

export default {
  preprocess: vitePreprocess(),
}
```

### client/tsconfig.json

```json
{
  "extends": "@tsconfig/svelte/tsconfig.json",
  "compilerOptions": {
    "target": "ESNext",
    "useDefineForClassFields": true,
    "module": "ESNext",
    "resolveJsonModule": true,
    "allowJs": true,
    "checkJs": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src/**/*.ts", "src/**/*.svelte"],
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
    "noFallthroughCasesInSwitch": true,
    "noUncheckedSideEffectImports": true
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
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

### client/src/main.ts

```typescript
import App from './App.svelte'
import './styles/index.css'

const app = new App({
  target: document.getElementById('app')!,
})

export default app
```

### client/src/vite-env.d.ts

```typescript
/// <reference types="svelte" />
/// <reference types="vite/client" />
```

### client/src/App.svelte

```svelte
<script lang="ts">
  import { Router } from 'svelte-routing'
  import Layout from './components/Layout.svelte'
</script>

<Router>
  <Layout />
</Router>
```

### client/src/components/Layout.svelte

```svelte
<script lang="ts">
  import { Route, Link } from 'svelte-routing'
  import Home from '../pages/Home.svelte'
  import About from '../pages/About.svelte'
</script>

<div class="app">
  <header class="header">
    <nav>
      <Link to="/">Home</Link>
      <Link to="/about">About</Link>
    </nav>
  </header>
  <main class="main">
    <Route path="/" component={Home} />
    <Route path="/about" component={About} />
  </main>
  <footer class="footer">
    <p>Built with Mizu + Svelte</p>
  </footer>
</div>
```

### client/src/pages/Home.svelte

```svelte
<script lang="ts">
  import { onMount } from 'svelte'

  let message = $state('')
  let loading = $state(true)

  onMount(async () => {
    try {
      const res = await fetch('/api/hello')
      const data = await res.json()
      message = data.message
    } catch {
      message = 'Failed to load message'
    } finally {
      loading = false
    }
  })
</script>

<div class="page home">
  <h1>Welcome</h1>
  {#if loading}
    <p>Loading...</p>
  {:else}
    <p class="api-message">{message}</p>
  {/if}
</div>
```

### client/src/pages/About.svelte

```svelte
<div class="page about">
  <h1>About</h1>
  <p>
    This is a Svelte SPA powered by Mizu, a lightweight Go web framework.
  </p>
  <ul>
    <li>Svelte 5 with TypeScript</li>
    <li>Vite for fast development and optimized builds</li>
    <li>svelte-routing for client-side routing</li>
    <li>Mizu backend with the frontend middleware</li>
  </ul>
</div>
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

.header nav :global(a) {
  color: var(--text);
  text-decoration: none;
  font-weight: 500;
}

.header nav :global(a:hover) {
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
# Create a new Svelte SPA project
mizu new ./myapp --template frontend/spa/svelte

# List all templates (shows Svelte alongside React and Vue)
mizu new --list

# Preview the generated files
mizu new ./myapp --template frontend/spa/svelte --dry-run
```

## Testing Strategy

### Unit Tests

Add tests to `templates_test.go`:

```go
func TestListTemplatesIncludesSvelte(t *testing.T) {
    templates, err := listTemplates()
    if err != nil {
        t.Fatalf("listTemplates() error: %v", err)
    }

    found := false
    for _, tmpl := range templates {
        if tmpl.Name == "frontend/spa/svelte" {
            found = true
            break
        }
    }

    if !found {
        t.Error("listTemplates() did not include nested template 'frontend/spa/svelte'")
    }
}

func TestTemplateExistsSvelte(t *testing.T) {
    if !templateExists("frontend/spa/svelte") {
        t.Error("templateExists('frontend/spa/svelte') returned false")
    }
}

func TestLoadTemplateFilesSvelte(t *testing.T) {
    files, err := loadTemplateFiles("frontend/spa/svelte")
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
        "client/svelte.config.js",
        "client/src/App.svelte",
        "client/src/main.ts",
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

func TestBuildPlanSvelteTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/spa/svelte", tmpDir, vars)
    if err != nil {
        t.Fatalf("buildPlan() error: %v", err)
    }

    if len(p.files) == 0 {
        t.Error("buildPlan() produced empty plan")
    }
}

func TestApplyPlanSvelteTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/spa/svelte", tmpDir, vars)
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
        "client/svelte.config.js",
        "client/src/App.svelte",
        "client/src/main.ts",
        "Makefile",
    }

    for _, file := range expectedFiles {
        path := filepath.Join(tmpDir, file)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            t.Errorf("expected file %q does not exist", file)
        }
    }
}

func TestSvelteTemplateVariableSubstitution(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("mysvelte", "example.com/mysvelte", "MIT", nil)

    p, err := buildPlan("frontend/spa/svelte", tmpDir, vars)
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

    if !strings.Contains(string(content), "module example.com/mysvelte") {
        t.Error("go.mod does not contain expected module path")
    }

    // Check package.json has correct name
    pkgPath := filepath.Join(tmpDir, "client/package.json")
    pkgContent, err := os.ReadFile(pkgPath)
    if err != nil {
        t.Fatalf("failed to read package.json: %v", err)
    }

    if !strings.Contains(string(pkgContent), `"name": "mysvelte-client"`) {
        t.Error("package.json does not contain expected project name")
    }
}

func TestSvelteTemplateHasSvelteSpecificContent(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("mysvelte", "example.com/mysvelte", "MIT", nil)

    p, err := buildPlan("frontend/spa/svelte", tmpDir, vars)
    if err != nil {
        t.Fatalf("buildPlan() error: %v", err)
    }

    if err := p.apply(false); err != nil {
        t.Fatalf("apply() error: %v", err)
    }

    // Verify App.svelte contains Svelte-specific syntax
    appPath := filepath.Join(tmpDir, "client/src/App.svelte")
    content, err := os.ReadFile(appPath)
    if err != nil {
        t.Fatalf("failed to read App.svelte: %v", err)
    }

    sveltePatterns := []string{
        "<script lang=\"ts\">",
        "import { Router }",
        "svelte-routing",
    }

    for _, pattern := range sveltePatterns {
        if !strings.Contains(string(content), pattern) {
            t.Errorf("App.svelte missing expected Svelte pattern: %s", pattern)
        }
    }

    // Verify svelte.config.js exists with proper content
    configPath := filepath.Join(tmpDir, "client/svelte.config.js")
    configContent, err := os.ReadFile(configPath)
    if err != nil {
        t.Fatalf("failed to read svelte.config.js: %v", err)
    }

    if !strings.Contains(string(configContent), "vitePreprocess") {
        t.Error("svelte.config.js missing vitePreprocess configuration")
    }
}
```

## Key Differences from React/Vue Templates

| Aspect | React | Vue | Svelte |
|--------|-------|-----|--------|
| Entry Point | `main.tsx` | `main.ts` | `main.ts` |
| Root Component | `App.tsx` (JSX) | `App.vue` (SFC) | `App.svelte` |
| Router | react-router-dom | vue-router | svelte-routing |
| Build Command | `tsc && vite build` | `vue-tsc -b && vite build` | `vite build` |
| Vite Plugin | `@vitejs/plugin-react` | `@vitejs/plugin-vue` | `@sveltejs/vite-plugin-svelte` |
| Type Checking | TypeScript | vue-tsc | svelte-check |
| Component Format | JSX/TSX | Single File Components | Svelte Components |
| State Management | useState/useEffect | ref/onMounted | $state rune/onMount |
| Config File | - | - | svelte.config.js |

## Implementation Notes

1. **Svelte 5 Runes**: Uses Svelte 5's new rune syntax (`$state`, `$derived`, etc.) for reactive state
2. **svelte-routing**: Lightweight router specifically designed for Svelte SPAs (vs SvelteKit's built-in routing)
3. **TypeScript Support**: Full TypeScript support with `svelte-check` for type checking Svelte files
4. **Compile-Time**: Svelte compiles components at build time, resulting in smaller runtime bundles
5. **Consistent Styling**: Shares the same CSS as React/Vue templates for visual consistency
6. **Same Backend**: Go backend is identical to React/Vue templates (framework-agnostic)
7. **Scoped Styles**: Svelte component styles use `:global()` selector for child component styling

## Why svelte-routing

For a simple SPA template, `svelte-routing` is preferred over SvelteKit because:
- Lighter weight and simpler API
- No server-side rendering complexity
- Direct integration with the Mizu Go backend
- Familiar pattern for developers coming from React Router or Vue Router
- Better suited for embedding in a non-Node.js server environment
