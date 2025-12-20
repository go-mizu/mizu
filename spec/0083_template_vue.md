# Vue SPA Template Design Spec

## Overview

This specification describes the implementation of the Vue.js SPA template for the Mizu CLI. The template follows the same structure as the React template (`frontend/react`) but uses Vue 3 with Composition API, TypeScript, and Vue Router.

## Goals

1. **Modern Vue Setup**: Vue 3 with Composition API, TypeScript, and Vite
2. **Consistent Structure**: Mirror the React template structure for familiarity
3. **Vue Best Practices**: Use `<script setup>`, Vue Router 4, and proper TypeScript support
4. **Mizu Integration**: Seamless integration with the `frontend` middleware
5. **Developer Experience**: Hot reload, dev/production modes, and proper tooling

## Template Location

```
cmd/cli/templates/frontend/vue/
├── template.json              # Template metadata
├── Makefile.tmpl              # Build and development commands
├── cmd/server/
│   └── main.go.tmpl           # Go entry point
├── app/server/
│   ├── app.go.tmpl            # App setup with embedded frontend
│   ├── config.go.tmpl         # Server configuration
│   └── routes.go.tmpl         # API routes
├── client/
│   ├── package.json           # Vue dependencies
│   ├── vite.config.ts         # Vite configuration
│   ├── tsconfig.json          # TypeScript configuration
│   ├── tsconfig.node.json     # TypeScript config for Vite
│   ├── index.html             # HTML template
│   ├── env.d.ts               # Vue TypeScript declarations
│   ├── src/
│   │   ├── main.ts            # Vue app entry point
│   │   ├── App.vue            # Root component with router
│   │   ├── router/
│   │   │   └── index.ts       # Vue Router configuration
│   │   ├── components/
│   │   │   └── Layout.vue     # Layout with navigation
│   │   ├── pages/
│   │   │   ├── Home.vue       # Home page with API call
│   │   │   └── About.vue      # About page
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
  "name": "frontend/vue",
  "description": "Vue SPA with Vite, TypeScript, and Mizu backend",
  "tags": ["go", "mizu", "frontend", "spa", "vue", "vite", "typescript"],
  "variables": {
    "name": { "description": "Project name", "default": "" },
    "module": { "description": "Go module path", "default": "" }
  }
}
```

## Go Backend Files

The Go backend files are identical to the React template, with only comments updated to reference Vue:

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

// New creates a new Mizu app configured for the Vue SPA.
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

## Vue Frontend Files

### client/package.json

```json
{
  "name": "{{.Name}}-client",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc -b && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "vue": "^3.5.13",
    "vue-router": "^4.5.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.2.1",
    "typescript": "~5.6.3",
    "vite": "^6.0.5",
    "vue-tsc": "^2.2.0"
  }
}
```

### client/vite.config.ts

```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
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
    "module": "ESNext",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "preserve",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedSideEffectImports": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src/**/*.ts", "src/**/*.tsx", "src/**/*.vue"]
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

### client/env.d.ts

```typescript
/// <reference types="vite/client" />
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
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './styles/index.css'

createApp(App).use(router).mount('#app')
```

### client/src/App.vue

```vue
<script setup lang="ts">
import Layout from './components/Layout.vue'
</script>

<template>
  <Layout />
</template>
```

### client/src/router/index.ts

```typescript
import { createRouter, createWebHistory } from 'vue-router'
import Home from '../pages/Home.vue'
import About from '../pages/About.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/about', component: About },
  ],
})

export default router
```

### client/src/components/Layout.vue

```vue
<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <div class="app">
    <header class="header">
      <nav>
        <RouterLink to="/">Home</RouterLink>
        <RouterLink to="/about">About</RouterLink>
      </nav>
    </header>
    <main class="main">
      <RouterView />
    </main>
    <footer class="footer">
      <p>Built with Mizu + Vue</p>
    </footer>
  </div>
</template>
```

### client/src/pages/Home.vue

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'

const message = ref('')
const loading = ref(true)

onMounted(async () => {
  try {
    const res = await fetch('/api/hello')
    const data = await res.json()
    message.value = data.message
  } catch {
    message.value = 'Failed to load message'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="page home">
    <h1>Welcome</h1>
    <p v-if="loading">Loading...</p>
    <p v-else class="api-message">{{ message }}</p>
  </div>
</template>
```

### client/src/pages/About.vue

```vue
<template>
  <div class="page about">
    <h1>About</h1>
    <p>
      This is a Vue SPA powered by Mizu, a lightweight Go web framework.
    </p>
    <ul>
      <li>Vue 3 with Composition API and TypeScript</li>
      <li>Vite for fast development and optimized builds</li>
      <li>Vue Router for client-side routing</li>
      <li>Mizu backend with the frontend middleware</li>
    </ul>
  </div>
</template>
```

### client/src/styles/index.css

Same as React template - shared CSS for consistent look:

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
# Create a new Vue SPA project
mizu new ./myapp --template frontend/vue

# List all templates (shows Vue alongside React)
mizu new --list

# Preview the generated files
mizu new ./myapp --template frontend/vue --dry-run
```

## Testing Strategy

### Unit Tests

Add tests to `templates_test.go`:

```go
func TestListTemplatesIncludesVue(t *testing.T) {
    templates, err := listTemplates()
    if err != nil {
        t.Fatalf("listTemplates() error: %v", err)
    }

    found := false
    for _, tmpl := range templates {
        if tmpl.Name == "frontend/vue" {
            found = true
            break
        }
    }

    if !found {
        t.Error("listTemplates() did not include nested template 'frontend/vue'")
    }
}

func TestTemplateExistsVue(t *testing.T) {
    if !templateExists("frontend/vue") {
        t.Error("templateExists('frontend/vue') returned false")
    }
}

func TestLoadTemplateFilesVue(t *testing.T) {
    files, err := loadTemplateFiles("frontend/vue")
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
        "client/src/App.vue",
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

func TestApplyPlanVueTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/vue", tmpDir, vars)
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
        "client/src/App.vue",
        "client/src/main.ts",
        "client/src/router/index.ts",
        "Makefile",
    }

    for _, file := range expectedFiles {
        path := filepath.Join(tmpDir, file)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            t.Errorf("expected file %q does not exist", file)
        }
    }
}
```

## Key Differences from React Template

| Aspect | React | Vue |
|--------|-------|-----|
| Entry Point | `main.tsx` | `main.ts` |
| Root Component | `App.tsx` (JSX) | `App.vue` (SFC) |
| Router | react-router-dom | vue-router |
| Build Command | `tsc && vite build` | `vue-tsc -b && vite build` |
| Vite Plugin | `@vitejs/plugin-react` | `@vitejs/plugin-vue` |
| Type Checking | TypeScript | vue-tsc |
| Component Format | JSX/TSX | Single File Components (`.vue`) |
| State Management | useState/useEffect | ref/onMounted |

## Implementation Notes

1. **Vue Single File Components**: Uses `.vue` files with `<script setup>` for the most concise and modern Vue 3 syntax
2. **TypeScript Support**: Full TypeScript support with `vue-tsc` for type checking Vue files
3. **Composition API**: All components use Composition API with `<script setup>` for better TypeScript inference
4. **Consistent Styling**: Shares the same CSS as React template for visual consistency
5. **Same Backend**: Go backend is identical to React template (framework-agnostic)
