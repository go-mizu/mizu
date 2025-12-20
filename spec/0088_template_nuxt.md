# Nuxt SPA Template Design Spec

## Overview

This specification describes the implementation of a Nuxt SPA template for the Mizu CLI. Nuxt is a Vue-based meta-framework that provides file-based routing, auto-imports, and optimized builds. This template uses Nuxt in **static generation mode** (SPA) to generate a static site that can be served by the Mizu backend.

## Goals

1. **Modern Nuxt Setup**: Nuxt 3 with TypeScript and Tailwind CSS
2. **Static Generation**: Configure Nuxt for static site generation (`nuxt generate`)
3. **Mizu Integration**: Seamless integration with the `frontend` middleware for dev/production serving
4. **Best Practices**: File-based routing, auto-imports, and proper project structure
5. **Developer Experience**: Hot reload, TypeScript support, and modern tooling

## Template Hierarchy

```
cmd/cli/templates/frontend/spa/nuxt/
├── template.json              # Template metadata
├── Makefile.tmpl              # Build/dev commands
├── cmd/
│   └── server/
│       └── main.go.tmpl       # Go entry point
├── app/
│   └── server/
│       ├── app.go.tmpl        # Mizu app setup with frontend middleware
│       ├── config.go.tmpl     # Configuration
│       └── routes.go.tmpl     # API route definitions
├── dist/
│   ├── embed.go.tmpl          # Embed directive for built assets
│   └── placeholder.txt        # Output directory placeholder
└── client/                    # Nuxt application
    ├── package.json           # Node dependencies
    ├── nuxt.config.ts         # Nuxt configuration
    ├── tsconfig.json          # TypeScript config
    ├── tailwind.config.ts     # Tailwind CSS config
    ├── app.vue                # Root component
    ├── pages/                 # File-based routing
    │   ├── index.vue          # Home page
    │   └── about.vue          # About page
    ├── components/            # Auto-imported components
    │   └── AppNavigation.vue  # Navigation component
    ├── layouts/               # Layout components
    │   └── default.vue        # Default layout
    ├── assets/
    │   └── css/
    │       └── main.css       # Global styles with Tailwind
    └── public/
        └── nuxt.svg           # Static assets
```

## Template Metadata

### template.json

```json
{
  "name": "frontend/spa/nuxt",
  "description": "Nuxt SPA with TypeScript, Tailwind, and Mizu backend",
  "tags": ["go", "mizu", "frontend", "spa", "nuxt", "vue", "typescript", "tailwind"],
  "variables": {
    "name": { "description": "Project name", "default": "" },
    "module": { "description": "Go module path", "default": "" }
  }
}
```

## Go Backend Files

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
    "io/fs"

    "github.com/go-mizu/mizu"
    "github.com/go-mizu/mizu/middlewares/frontend"
    "{{.Module}}/dist"
)

func New(cfg *Config) *mizu.App {
    app := mizu.New()

    // API routes
    setupRoutes(app)

    // Frontend middleware (auto-detects dev/production mode)
    distFS, _ := fs.Sub(dist.FS, ".")
    app.Use(frontend.WithOptions(frontend.Options{
        Mode:        frontend.ModeAuto,
        FS:          distFS,
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

type Config struct {
    Port    string
    DevPort string
    Env     string
}

func LoadConfig() *Config {
    return &Config{
        Port:    getEnv("PORT", "3000"),
        DevPort: getEnv("DEV_PORT", "3001"),
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

## Nuxt Frontend Files

### client/package.json

```json
{
  "name": "{{.Name}}-client",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "nuxt dev --port 3001",
    "build": "nuxt generate",
    "preview": "nuxt preview",
    "postinstall": "nuxt prepare"
  },
  "dependencies": {
    "nuxt": "^3.14.0",
    "vue": "^3.5.13"
  },
  "devDependencies": {
    "@nuxtjs/tailwindcss": "^6.12.2",
    "typescript": "^5.7.2"
  }
}
```

### client/nuxt.config.ts

Nuxt configured for static generation with API proxy for development:

```typescript
export default defineNuxtConfig({
  compatibilityDate: '2024-12-01',
  devtools: { enabled: true },
  ssr: false,
  modules: ['@nuxtjs/tailwindcss'],
  nitro: {
    output: {
      publicDir: '../dist'
    }
  },
  vite: {
    server: {
      proxy: {
        '/api': {
          target: 'http://localhost:3000',
          changeOrigin: true,
        },
      },
    },
  },
})
```

### client/tsconfig.json

```json
{
  "extends": "./.nuxt/tsconfig.json"
}
```

### client/tailwind.config.ts

```typescript
import type { Config } from 'tailwindcss'

const config: Config = {
  content: [
    './components/**/*.{js,vue,ts}',
    './layouts/**/*.vue',
    './pages/**/*.vue',
    './plugins/**/*.{js,ts}',
    './app.vue',
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}

export default config
```

### client/app.vue

```vue
<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
```

### client/layouts/default.vue

```vue
<template>
  <div class="min-h-screen bg-slate-50 text-slate-900 flex flex-col">
    <AppNavigation />
    <main class="flex-1 max-w-5xl mx-auto w-full p-8">
      <slot />
    </main>
    <footer class="border-t border-slate-200 bg-white py-4 text-center text-slate-500">
      <p>Built with Mizu + Nuxt</p>
    </footer>
  </div>
</template>
```

### client/components/AppNavigation.vue

```vue
<script setup lang="ts">
const route = useRoute()

const navLinks = [
  { href: '/', label: 'Home' },
  { href: '/about', label: 'About' },
]
</script>

<template>
  <header class="bg-white border-b border-slate-200">
    <nav class="max-w-5xl mx-auto px-8 py-4">
      <ul class="flex gap-6">
        <li v-for="link in navLinks" :key="link.href">
          <NuxtLink
            :to="link.href"
            class="font-medium transition-colors"
            :class="route.path === link.href
              ? 'text-blue-600'
              : 'text-slate-600 hover:text-blue-600'"
          >
            {{ link.label }}
          </NuxtLink>
        </li>
      </ul>
    </nav>
  </header>
</template>
```

### client/pages/index.vue

```vue
<script setup lang="ts">
const message = ref('')
const loading = ref(true)

onMounted(async () => {
  try {
    const data = await $fetch<{ message: string }>('/api/hello')
    message.value = data.message
  } catch {
    message.value = 'Failed to load message'
  } finally {
    loading.value = false
  }
})

useHead({
  title: '{{.Name}}',
})
</script>

<template>
  <div class="space-y-6">
    <h1 class="text-4xl font-bold">Welcome to {{.Name}}</h1>
    <p v-if="loading" class="text-slate-500">Loading...</p>
    <div v-else class="bg-white rounded-lg border border-slate-200 p-4">
      <p class="text-lg">{{ message }}</p>
    </div>
  </div>
</template>
```

### client/pages/about.vue

```vue
<script setup lang="ts">
useHead({
  title: 'About - {{.Name}}',
})
</script>

<template>
  <div class="space-y-6">
    <h1 class="text-4xl font-bold">About</h1>
    <p class="text-slate-600">
      This is a Nuxt SPA powered by Mizu, a lightweight Go web framework.
    </p>
    <ul class="list-disc list-inside space-y-2 text-slate-600">
      <li>Nuxt 3 with TypeScript</li>
      <li>Static generation for SPA deployment</li>
      <li>File-based routing with auto-imports</li>
      <li>Tailwind CSS for styling</li>
      <li>Mizu backend with the frontend middleware</li>
    </ul>
  </div>
</template>
```

### client/assets/css/main.css

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

## Makefile

### Makefile.tmpl

```makefile
.PHONY: dev build run clean install

# Development mode: run Nuxt dev server and Go server concurrently
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
	rm -rf client/.nuxt
	rm -rf client/.output

# Install dependencies
install:
	cd client && npm install
	go mod tidy
```

## Production Embed

### dist/embed.go.tmpl

```go
package dist

import "embed"

// FS contains the built Nuxt static export.
// After running `npm run build` in the client directory,
// the dist folder will contain the production-ready static assets.
//
//go:embed all:*
var FS embed.FS
```

### dist/placeholder.txt

```
This directory will contain the built Nuxt static assets.
Run 'make build' or 'cd client && npm run build' to generate them.
```

## CLI Usage

```bash
# Create a new Nuxt SPA project
mizu new ./myapp --template frontend/spa/nuxt

# With custom module path
mizu new ./myapp --template frontend/spa/nuxt --module github.com/user/myapp

# Preview the generated files
mizu new ./myapp --template frontend/spa/nuxt --dry-run
```

## Key Differences from Other SPA Templates

| Feature | Nuxt | Next.js | Vue (Vite) | React (Vite) |
|---------|------|---------|------------|--------------|
| Framework | Vue 3 | React 19 | Vue 3 | React 18 |
| Build Tool | Nuxt CLI | Next.js CLI | Vite | Vite |
| Router | File-based (built-in) | App Router (built-in) | Vue Router | React Router |
| Auto-imports | Yes | No | No | No |
| Dev Port | 3001 | 3001 | 5173 | 5173 |
| Output | Static export to `dist/` | Static export to `dist/` | `dist/` | `dist/` |
| Styling | Tailwind CSS | Tailwind CSS | CSS | CSS |

## Nuxt-Specific Features

1. **Auto-imports**: Nuxt automatically imports Vue APIs (`ref`, `computed`, `onMounted`), composables (`useRoute`, `useHead`, `$fetch`), and components.

2. **File-based Routing**: Pages in the `pages/` directory automatically become routes without explicit configuration.

3. **Layouts**: The `layouts/` directory provides reusable page layouts with the `<slot />` pattern.

4. **Built-in Composables**: Nuxt provides `$fetch` for data fetching, `useHead` for meta tags, and many other utilities.

5. **TypeScript Support**: Full TypeScript support with auto-generated types in `.nuxt/`.

## Testing Strategy

### Unit Tests

```go
func TestNuxtTemplateExists(t *testing.T) {
    if !templateExists("frontend/spa/nuxt") {
        t.Error("templateExists('frontend/spa/nuxt') returned false")
    }
}

func TestLoadTemplateFilesNuxt(t *testing.T) {
    files, err := loadTemplateFiles("frontend/spa/nuxt")
    require.NoError(t, err)

    expectedFiles := []string{
        ".gitignore",
        "go.mod",
        "cmd/server/main.go",
        "app/server/app.go",
        "client/package.json",
        "client/nuxt.config.ts",
        "client/app.vue",
        "client/pages/index.vue",
    }

    fileMap := make(map[string]bool)
    for _, f := range files {
        fileMap[f.path] = true
    }

    for _, expected := range expectedFiles {
        if !fileMap[expected] {
            t.Errorf("expected file %q not found", expected)
        }
    }
}

func TestNuxtTemplateHasNuxtSpecificContent(t *testing.T) {
    files, err := loadTemplateFiles("frontend/spa/nuxt")
    require.NoError(t, err)

    fileMap := make(map[string]templateFile)
    for _, f := range files {
        fileMap[f.path] = f
    }

    // Verify nuxt.config.ts contains ssr: false
    nuxtConfig := fileMap["client/nuxt.config.ts"]
    if !strings.Contains(nuxtConfig.content, "ssr: false") {
        t.Error("nuxt.config.ts should contain ssr: false for SPA mode")
    }

    // Verify package.json contains "nuxt" dependency
    packageJSON := fileMap["client/package.json"]
    if !strings.Contains(packageJSON.content, `"nuxt"`) {
        t.Error("package.json should contain nuxt dependency")
    }

    // Verify app.vue uses NuxtLayout and NuxtPage
    appVue := fileMap["client/app.vue"]
    if !strings.Contains(appVue.content, "<NuxtLayout>") {
        t.Error("app.vue should use NuxtLayout")
    }
    if !strings.Contains(appVue.content, "<NuxtPage />") {
        t.Error("app.vue should use NuxtPage")
    }
}

func TestApplyPlanNuxtTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/spa/nuxt", tmpDir, vars)
    require.NoError(t, err)

    if err := p.apply(false); err != nil {
        t.Fatalf("apply() error: %v", err)
    }

    // Verify key files exist
    expectedFiles := []string{
        "go.mod",
        "cmd/server/main.go",
        "app/server/app.go",
        "client/package.json",
        "client/nuxt.config.ts",
        "client/app.vue",
        "client/pages/index.vue",
        "client/pages/about.vue",
        "client/layouts/default.vue",
        "client/components/AppNavigation.vue",
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

## Implementation Notes

1. **Static Generation**: Nuxt `nuxt generate` command produces static HTML/JS/CSS that can be served by any static file server, including Mizu's frontend middleware.

2. **API Proxying**: During development, Vite (used by Nuxt) proxies `/api/*` requests to the Go backend. In production, both frontend and API are served from the same origin.

3. **Port Configuration**: Nuxt dev server runs on port 3001 to avoid conflict with the Go server on port 3000.

4. **SSR Disabled**: Setting `ssr: false` in nuxt.config.ts ensures the app runs as a pure SPA.

5. **Tailwind CSS**: Included via the `@nuxtjs/tailwindcss` module for streamlined setup.

6. **Auto-imports**: Unlike Vue with Vite, Nuxt auto-imports Vue APIs and composables, reducing boilerplate.
