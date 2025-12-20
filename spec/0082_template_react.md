# Frontend SPA Template System Design Spec

## Overview

This specification describes the implementation of a nested template system for frontend SPA frameworks, starting with React. The template system will support the hierarchy `frontend/spa/{framework}` where framework can be `react`, `vue`, `solid`, `angular`, etc.

## Goals

1. **Nested Template Support**: Extend the existing template system to support hierarchical template paths (e.g., `frontend/spa/react`)
2. **Modern React Setup**: Provide a production-ready React + Vite template with TypeScript support
3. **Mizu Integration**: Seamless integration with the `frontend` middleware for dev/production serving
4. **Best Practices**: Include modern patterns like React Router, proper project structure, and build tooling
5. **Extensibility**: Design for easy addition of Vue, Solid, Angular, and other frameworks

## Template Hierarchy

```
cmd/cli/templates/
├── _common/                    # Shared across ALL templates
│   ├── gitignore.tmpl
│   ├── go.mod.tmpl
│   └── readme.md.tmpl
├── frontend/
│   ├── _common/                # Shared across all frontend templates
│   │   └── Makefile.tmpl       # Common frontend Makefile with dev/build targets
│   ├── spa/
│   │   ├── _common/            # Shared across all SPA templates
│   │   │   └── ...             # Common SPA files
│   │   ├── react/              # React-specific template
│   │   │   ├── template.json
│   │   │   ├── cmd/server/main.go.tmpl
│   │   │   ├── app/server/app.go.tmpl
│   │   │   ├── app/server/config.go.tmpl
│   │   │   ├── app/server/routes.go.tmpl
│   │   │   ├── client/         # React app
│   │   │   │   ├── package.json
│   │   │   │   ├── vite.config.ts
│   │   │   │   ├── tsconfig.json
│   │   │   │   ├── index.html
│   │   │   │   └── src/
│   │   │   │       ├── main.tsx
│   │   │   │       ├── App.tsx
│   │   │   │       ├── components/
│   │   │   │       ├── pages/
│   │   │   │       └── styles/
│   │   │   └── dist/           # Production embed
│   │   │       └── embed.go.tmpl
│   │   ├── vue/                # (Future) Vue-specific template
│   │   ├── solid/              # (Future) Solid-specific template
│   │   └── angular/            # (Future) Angular-specific template
│   └── ssr/                    # (Future) SSR templates
│       └── ...
├── api/
├── web/
├── live/
├── sync/
├── contract/
└── minimal/
```

## Template Resolution Algorithm

When loading a nested template like `frontend/spa/react`, the system will:

1. Load files from `_common/` (root level)
2. Load files from `frontend/_common/` (category level)
3. Load files from `frontend/spa/_common/` (subcategory level)
4. Load files from `frontend/spa/react/` (template level)

Each level overrides files from the previous level with the same relative path.

### Implementation

```go
// loadTemplateFiles loads all files for a template, including nested common directories.
func loadTemplateFiles(name string) ([]templateFile, error) {
    fileMap := make(map[string]templateFile)

    // Always load root common files first
    loadFilesInto(fileMap, "templates/_common")

    // Parse template path for nested structure
    parts := strings.Split(name, "/")

    // Load each level's _common directory
    // e.g., for "frontend/spa/react":
    //   - templates/frontend/_common
    //   - templates/frontend/spa/_common
    for i := 1; i <= len(parts); i++ {
        commonPath := path.Join("templates", path.Join(parts[:i]...), "_common")
        loadFilesInto(fileMap, commonPath)
    }

    // Load template-specific files (override everything)
    templatePath := path.Join("templates", name)
    loadFilesInto(fileMap, templatePath)

    return mapToSlice(fileMap), nil
}
```

## Template Metadata

### template.json

```json
{
  "name": "frontend/spa/react",
  "description": "React SPA with Vite, TypeScript, and Mizu backend",
  "tags": ["go", "mizu", "frontend", "spa", "react", "vite", "typescript"],
  "category": "frontend",
  "subcategory": "spa",
  "framework": "react",
  "variables": {
    "name": { "description": "Project name", "default": "" },
    "module": { "description": "Go module path", "default": "" },
    "port": { "description": "Server port", "default": "3000" },
    "devPort": { "description": "Vite dev server port", "default": "5173" }
  },
  "postCreate": [
    "cd client && npm install",
    "npm run build"
  ]
}
```

## React Template Structure

### Go Backend Files

#### cmd/server/main.go.tmpl
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

#### app/server/app.go.tmpl
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

#### app/server/config.go.tmpl
```go
package server

import "os"

type Config struct {
    Port    string
    DevPort string
    Env     string
}

func LoadConfig() *Config {
    cfg := &Config{
        Port:    getEnv("PORT", "{{.Vars.port}}"),
        DevPort: getEnv("DEV_PORT", "{{.Vars.devPort}}"),
        Env:     getEnv("MIZU_ENV", "development"),
    }
    return cfg
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

#### app/server/routes.go.tmpl
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

### React Frontend Files

#### client/package.json
```json
{
  "name": "{{.Name}}-client",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "lint": "eslint . --ext ts,tsx --report-unused-disable-directives --max-warnings 0",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.28.0"
  },
  "devDependencies": {
    "@types/react": "^18.3.12",
    "@types/react-dom": "^18.3.1",
    "@vitejs/plugin-react": "^4.3.4",
    "typescript": "~5.6.3",
    "vite": "^6.0.3"
  }
}
```

#### client/vite.config.ts
```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
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

#### client/tsconfig.json
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

#### client/tsconfig.node.json
```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true,
    "strict": true
  },
  "include": ["vite.config.ts"]
}
```

#### client/index.html
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
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

#### client/src/main.tsx
```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import './styles/index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
)
```

#### client/src/App.tsx
```tsx
import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import Home from './pages/Home'
import About from './pages/About'

function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<Home />} />
        <Route path="about" element={<About />} />
      </Route>
    </Routes>
  )
}

export default App
```

#### client/src/components/Layout.tsx
```tsx
import { Outlet, Link } from 'react-router-dom'

function Layout() {
  return (
    <div className="app">
      <header className="header">
        <nav>
          <Link to="/">Home</Link>
          <Link to="/about">About</Link>
        </nav>
      </header>
      <main className="main">
        <Outlet />
      </main>
      <footer className="footer">
        <p>Built with Mizu + React</p>
      </footer>
    </div>
  )
}

export default Layout
```

#### client/src/pages/Home.tsx
```tsx
import { useState, useEffect } from 'react'

function Home() {
  const [message, setMessage] = useState<string>('')
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
      <h1>Welcome to {{.Name}}</h1>
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

#### client/src/pages/About.tsx
```tsx
function About() {
  return (
    <div className="page about">
      <h1>About</h1>
      <p>
        This is a React SPA powered by Mizu, a lightweight Go web framework.
      </p>
      <ul>
        <li>React 18 with TypeScript</li>
        <li>Vite for fast development and optimized builds</li>
        <li>React Router for client-side routing</li>
        <li>Mizu backend with the frontend middleware</li>
      </ul>
    </div>
  )
}

export default About
```

#### client/src/styles/index.css
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

### Makefile

#### Makefile.tmpl
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

### Production Embed

#### dist/embed.go.tmpl
```go
package dist

import "embed"

// FS contains the built frontend assets.
// This file is a placeholder. After running `npm run build` in the client directory,
// the dist folder will contain the production-ready assets.
//go:embed all:*
var FS embed.FS
```

## CLI Usage

```bash
# Create a new React SPA project
mizu new ./myapp --template frontend/spa/react

# With custom variables
mizu new ./myapp --template frontend/spa/react --var port=8080 --var devPort=3000

# List all templates (including nested)
mizu new --list

# Preview the generated files
mizu new ./myapp --template frontend/spa/react --dry-run
```

## Template Listing Enhancement

The `--list` command will show nested templates with their full paths:

```
Template              Purpose
────────────────────────────────────────────────────────────
api                   JSON API service with a recommended layout
contract              Code-first API contract with mizu integration
frontend/spa/react    React SPA with Vite, TypeScript, and Mizu backend
live                  Real-time interactive app with live views
minimal               Smallest runnable Mizu project
sync                  Offline-first app with sync and reactive state
web                   Full-stack web app with views and static assets
```

## Testing Strategy

### Unit Tests

```go
// templates_test.go

func TestNestedTemplateLoading(t *testing.T) {
    files, err := loadTemplateFiles("frontend/spa/react")
    require.NoError(t, err)

    // Verify common files are loaded
    assertFileExists(t, files, ".gitignore")
    assertFileExists(t, files, "go.mod")

    // Verify template-specific files
    assertFileExists(t, files, "cmd/server/main.go")
    assertFileExists(t, files, "client/package.json")
    assertFileExists(t, files, "client/src/App.tsx")
}

func TestTemplateHierarchyOverride(t *testing.T) {
    // Test that template-specific files override common files
    files, err := loadTemplateFiles("frontend/spa/react")
    require.NoError(t, err)

    // The Makefile from frontend/spa/react should override
    // any Makefile from _common directories
    makefile := findFile(files, "Makefile")
    require.NotNil(t, makefile)
    require.Contains(t, string(makefile.content), "npm run build")
}

func TestNestedTemplateExists(t *testing.T) {
    require.True(t, templateExists("frontend/spa/react"))
    require.False(t, templateExists("frontend/spa/nonexistent"))
}
```

### Integration Tests

```go
func TestReactTemplateGeneration(t *testing.T) {
    tmpDir := t.TempDir()

    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)
    p, err := buildPlan("frontend/spa/react", tmpDir, vars)
    require.NoError(t, err)

    err = p.apply(false)
    require.NoError(t, err)

    // Verify key files exist
    require.FileExists(t, filepath.Join(tmpDir, "go.mod"))
    require.FileExists(t, filepath.Join(tmpDir, "cmd/server/main.go"))
    require.FileExists(t, filepath.Join(tmpDir, "client/package.json"))
    require.FileExists(t, filepath.Join(tmpDir, "client/vite.config.ts"))
    require.FileExists(t, filepath.Join(tmpDir, "client/src/App.tsx"))

    // Verify template variable substitution
    content, _ := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
    require.Contains(t, string(content), "example.com/myapp")

    // Verify package.json has correct name
    pkgJSON, _ := os.ReadFile(filepath.Join(tmpDir, "client/package.json"))
    require.Contains(t, string(pkgJSON), "myapp-client")
}

func TestReactTemplateBuildable(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping build test in short mode")
    }

    tmpDir := t.TempDir()
    vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)
    p, _ := buildPlan("frontend/spa/react", tmpDir, vars)
    p.apply(false)

    // Verify Go code compiles
    cmd := exec.Command("go", "build", "./...")
    cmd.Dir = tmpDir
    output, err := cmd.CombinedOutput()
    require.NoError(t, err, "Go build failed: %s", output)
}
```

## Implementation Plan

### Phase 1: Template System Enhancement

1. Modify `loadTemplateFiles()` to support nested paths
2. Modify `loadTemplateMeta()` for nested template.json files
3. Update `listTemplates()` to show nested templates
4. Update `templateExists()` for nested templates

### Phase 2: React Template Creation

1. Create directory structure `templates/frontend/spa/react/`
2. Create `template.json` metadata
3. Create Go backend files (cmd/server, app/server)
4. Create React frontend files (client/*)
5. Create Makefile

### Phase 3: Testing

1. Unit tests for nested template loading
2. Integration tests for template generation
3. Build verification tests

### Phase 4: Documentation

1. Update README with new template
2. Add examples to CLI help

## Future Templates

The same structure supports additional frameworks:

- **frontend/spa/vue**: Vue 3 + Vite + TypeScript
- **frontend/spa/solid**: SolidJS + Vite + TypeScript
- **frontend/spa/angular**: Angular 17+ standalone components
- **frontend/spa/svelte**: SvelteKit or Svelte + Vite
- **frontend/ssr/react**: React with SSR support
- **frontend/ssr/vue**: Nuxt.js integration

Each framework template will:
1. Share common Go backend code via `frontend/_common` and `frontend/spa/_common`
2. Have framework-specific client code
3. Integrate with the Mizu frontend middleware
