# Preact SPA Template Design Spec

## Overview

This specification describes the implementation of the Preact SPA template for the Mizu CLI. Preact is a fast 3kB alternative to React with the same modern API, making it ideal for lightweight, performance-focused SPAs. The template uses Preact with TypeScript, preact-router for client-side navigation, and Vite for the build system.

## Goals

1. **Lightweight Performance**: Leverage Preact's 3kB size for minimal bundle overhead
2. **React-Compatible API**: Familiar hooks and component patterns from React ecosystem
3. **Modern Tooling**: Vite with the official Preact preset for optimal DX
4. **Consistent Structure**: Mirror other SPA templates (React, Vue, Svelte) for familiarity
5. **Mizu Integration**: Seamless integration with the `frontend` middleware
6. **TypeScript Support**: Full TypeScript support with proper JSX configuration

## Template Location

```
cmd/cli/templates/frontend/spa/preact/
├── template.json              # Template metadata
├── Makefile.tmpl              # Build and development commands
├── cmd/server/
│   └── main.go.tmpl           # Go entry point
├── app/server/
│   ├── app.go.tmpl            # App setup with embedded frontend
│   ├── config.go.tmpl         # Server configuration
│   └── routes.go.tmpl         # API routes
├── client/
│   ├── package.json           # Preact dependencies
│   ├── vite.config.ts         # Vite configuration with Preact plugin
│   ├── tsconfig.json          # TypeScript configuration
│   ├── tsconfig.node.json     # TypeScript config for Vite
│   ├── index.html             # HTML template
│   ├── src/
│   │   ├── main.tsx           # Preact app entry point
│   │   ├── App.tsx            # Root component with router
│   │   ├── vite-env.d.ts      # Vite TypeScript declarations
│   │   ├── components/
│   │   │   └── Layout.tsx     # Layout with navigation
│   │   ├── pages/
│   │   │   ├── Home.tsx       # Home page with API call
│   │   │   └── About.tsx      # About page
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
  "name": "frontend/spa/preact",
  "description": "Preact SPA with Vite, TypeScript, and Mizu backend",
  "tags": ["go", "mizu", "frontend", "spa", "preact", "vite", "typescript"],
  "variables": {
    "name": { "description": "Project name", "default": "" },
    "module": { "description": "Go module path", "default": "" }
  }
}
```

## Go Backend Files

The Go backend files are identical to React/Vue/Svelte templates, providing a consistent server experience:

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

// New creates a new Mizu app configured for the Preact SPA.
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

## Preact Frontend Files

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
    "preact": "^10.25.4",
    "preact-router": "^4.1.2"
  },
  "devDependencies": {
    "@preact/preset-vite": "^2.9.4",
    "typescript": "~5.6.3",
    "vite": "^6.0.5"
  }
}
```

### client/vite.config.ts

```typescript
import { defineConfig } from 'vite'
import preact from '@preact/preset-vite'

export default defineConfig({
  plugins: [preact()],
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
    "jsx": "react-jsx",
    "jsxImportSource": "preact",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": {
      "react": ["./node_modules/preact/compat"],
      "react-dom": ["./node_modules/preact/compat"],
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
    <div id="app"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

### client/src/main.tsx

```tsx
import { render } from 'preact'
import App from './App'
import './styles/index.css'

render(<App />, document.getElementById('app')!)
```

### client/src/vite-env.d.ts

```typescript
/// <reference types="vite/client" />
```

### client/src/App.tsx

```tsx
import Router from 'preact-router'
import Layout from './components/Layout'
import Home from './pages/Home'
import About from './pages/About'

function App() {
  return (
    <Layout>
      <Router>
        <Home path="/" />
        <About path="/about" />
      </Router>
    </Layout>
  )
}

export default App
```

### client/src/components/Layout.tsx

```tsx
import { ComponentChildren } from 'preact'

interface LayoutProps {
  children: ComponentChildren
}

function Layout({ children }: LayoutProps) {
  return (
    <div className="app">
      <header className="header">
        <nav>
          <a href="/">Home</a>
          <a href="/about">About</a>
        </nav>
      </header>
      <main className="main">
        {children}
      </main>
      <footer className="footer">
        <p>Built with Mizu + Preact</p>
      </footer>
    </div>
  )
}

export default Layout
```

### client/src/pages/Home.tsx

```tsx
import { useState, useEffect } from 'preact/hooks'

interface HomeProps {
  path: string
}

function Home(_props: HomeProps) {
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/api/hello')
      .then(res => res.json())
      .then(data => {
        setMessage(data.message)
        setLoading(false)
      })
      .catch(() => setLoading(false))
  }, [])

  return (
    <div className="page home">
      <h1>Welcome</h1>
      {loading ? (
        <p>Loading...</p>
      ) : (
        <p className="api-message">{message}</p>
      )}
    </div>
  )
}

export default Home
```

### client/src/pages/About.tsx

```tsx
interface AboutProps {
  path: string
}

function About(_props: AboutProps) {
  return (
    <div className="page about">
      <h1>About</h1>
      <p>
        This is a Preact SPA powered by Mizu, a lightweight Go web framework.
      </p>
      <ul>
        <li>Preact 10 with TypeScript (~3kB gzipped)</li>
        <li>Vite for fast development and optimized builds</li>
        <li>preact-router for client-side routing</li>
        <li>Mizu backend with the frontend middleware</li>
      </ul>
    </div>
  )
}

export default About
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

.header a {
  color: var(--text);
  text-decoration: none;
  font-weight: 500;
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
# Create a new Preact SPA project
mizu new ./myapp --template frontend/spa/preact

# List all templates (shows Preact alongside React and Vue)
mizu new --list

# Preview the generated files
mizu new ./myapp --template frontend/spa/preact --dry-run
```

## Testing Strategy

### Unit Tests

Add tests to `templates_test.go`:

```go
func TestListTemplatesIncludesPreact(t *testing.T) {
    templates, err := listTemplates()
    if err != nil {
        t.Fatalf("listTemplates() error: %v", err)
    }

    found := false
    for _, tmpl := range templates {
        if tmpl.Name == "frontend/spa/preact" {
            found = true
            break
        }
    }

    if !found {
        t.Error("listTemplates() did not include nested template 'frontend/spa/preact'")
    }
}

func TestTemplateExistsPreact(t *testing.T) {
    if !templateExists("frontend/spa/preact") {
        t.Error("templateExists('frontend/spa/preact') returned false")
    }
}

func TestLoadTemplateFilesPreact(t *testing.T) {
    files, err := loadTemplateFiles("frontend/spa/preact")
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
        "client/src/App.tsx",
        "client/src/main.tsx",
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

func TestBuildPlanPreactTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/spa/preact", tmpDir, vars)
    if err != nil {
        t.Fatalf("buildPlan() error: %v", err)
    }

    if len(p.files) == 0 {
        t.Error("buildPlan() produced empty plan")
    }
}

func TestApplyPlanPreactTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/spa/preact", tmpDir, vars)
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
        "client/src/App.tsx",
        "client/src/main.tsx",
        "Makefile",
    }

    for _, file := range expectedFiles {
        path := filepath.Join(tmpDir, file)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            t.Errorf("expected file %q does not exist", file)
        }
    }
}

func TestPreactTemplateVariableSubstitution(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("mypreact", "example.com/mypreact", "MIT", nil)

    p, err := buildPlan("frontend/spa/preact", tmpDir, vars)
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

    if !strings.Contains(string(content), "module example.com/mypreact") {
        t.Error("go.mod does not contain expected module path")
    }

    // Check package.json has correct name
    pkgPath := filepath.Join(tmpDir, "client/package.json")
    pkgContent, err := os.ReadFile(pkgPath)
    if err != nil {
        t.Fatalf("failed to read package.json: %v", err)
    }

    if !strings.Contains(string(pkgContent), `"name": "mypreact-client"`) {
        t.Error("package.json does not contain expected project name")
    }
}

func TestPreactTemplateHasPreactSpecificContent(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("mypreact", "example.com/mypreact", "MIT", nil)

    p, err := buildPlan("frontend/spa/preact", tmpDir, vars)
    if err != nil {
        t.Fatalf("buildPlan() error: %v", err)
    }

    if err := p.apply(false); err != nil {
        t.Fatalf("apply() error: %v", err)
    }

    // Verify App.tsx contains Preact-specific syntax
    appPath := filepath.Join(tmpDir, "client/src/App.tsx")
    content, err := os.ReadFile(appPath)
    if err != nil {
        t.Fatalf("failed to read App.tsx: %v", err)
    }

    preactPatterns := []string{
        "preact-router",
        "Router",
    }

    for _, pattern := range preactPatterns {
        if !strings.Contains(string(content), pattern) {
            t.Errorf("App.tsx missing expected Preact pattern: %s", pattern)
        }
    }

    // Verify main.tsx uses Preact render
    mainPath := filepath.Join(tmpDir, "client/src/main.tsx")
    mainContent, err := os.ReadFile(mainPath)
    if err != nil {
        t.Fatalf("failed to read main.tsx: %v", err)
    }

    if !strings.Contains(string(mainContent), "import { render } from 'preact'") {
        t.Error("main.tsx missing Preact render import")
    }

    // Verify vite.config.ts uses Preact preset
    viteConfigPath := filepath.Join(tmpDir, "client/vite.config.ts")
    viteContent, err := os.ReadFile(viteConfigPath)
    if err != nil {
        t.Fatalf("failed to read vite.config.ts: %v", err)
    }

    if !strings.Contains(string(viteContent), "@preact/preset-vite") {
        t.Error("vite.config.ts missing @preact/preset-vite import")
    }

    // Verify package.json has Preact dependencies
    pkgPath := filepath.Join(tmpDir, "client/package.json")
    pkgContent, err := os.ReadFile(pkgPath)
    if err != nil {
        t.Fatalf("failed to read package.json: %v", err)
    }

    preactDeps := []string{
        `"preact"`,
        `"preact-router"`,
        `"@preact/preset-vite"`,
    }

    for _, dep := range preactDeps {
        if !strings.Contains(string(pkgContent), dep) {
            t.Errorf("package.json missing Preact dependency: %s", dep)
        }
    }
}
```

## Key Differences from React Template

| Aspect | React | Preact |
|--------|-------|--------|
| Bundle Size | ~40kB gzipped | ~3kB gzipped |
| Library | `react`, `react-dom` | `preact` |
| Router | `react-router-dom` | `preact-router` |
| Hooks Import | `react` | `preact/hooks` |
| Render Function | `ReactDOM.createRoot().render()` | `render()` from preact |
| Vite Plugin | `@vitejs/plugin-react` | `@preact/preset-vite` |
| JSX Runtime | `react-jsx` | `react-jsx` with `jsxImportSource: "preact"` |
| Compat Layer | N/A | `preact/compat` for React compatibility |
| Router Pattern | Nested routes with Outlet | Flat routes as children |

## Implementation Notes

1. **JSX Configuration**: Uses `jsxImportSource: "preact"` in tsconfig.json for proper Preact JSX transformation
2. **Hooks from preact/hooks**: Unlike React, Preact hooks are imported from `preact/hooks` submodule
3. **preact-router**: Uses declarative routing with path props on components (vs react-router's Route components)
4. **ComponentChildren**: Preact's equivalent of React's ReactNode for typing children props
5. **No StrictMode**: Preact doesn't have StrictMode wrapper (not needed due to simpler rendering)
6. **Consistent Styling**: Shares the same CSS as React/Vue/Svelte templates for visual consistency
7. **Same Backend**: Go backend is identical to React template (framework-agnostic)

## Why Preact

Preact is an excellent choice for SPAs that prioritize:

1. **Bundle Size**: At ~3kB gzipped, Preact is significantly smaller than React (~40kB)
2. **Performance**: Faster initial load and time-to-interactive
3. **Familiar API**: Same hooks and component patterns as React
4. **Ecosystem Compatibility**: Many React libraries work via preact/compat
5. **Mobile/Low-bandwidth**: Ideal for users on slower connections
6. **Embedded/IoT**: Great for resource-constrained environments

## Preact Router vs React Router

The template uses `preact-router` instead of attempting to use `react-router-dom` with `preact/compat` because:

1. Native integration with Preact's lifecycle
2. Smaller bundle size
3. Simpler API for basic SPA routing needs
4. No compat layer complexity
5. Better TypeScript support for Preact components
