# SvelteKit SPA Template Design Spec

## Overview

This specification describes the implementation of a SvelteKit SPA template for the Mizu CLI. SvelteKit is the official Svelte meta-framework providing file-based routing, layouts, server-side rendering capabilities, and a robust build system. This template uses SvelteKit in **static adapter mode** (SPA) to generate a static site that can be served by the Mizu backend.

## Goals

1. **Modern SvelteKit Setup**: SvelteKit 2 with Svelte 5, TypeScript, and Tailwind CSS
2. **Static Generation**: Configure SvelteKit with `@sveltejs/adapter-static` for SPA deployment
3. **Mizu Integration**: Seamless integration with the `frontend` middleware for dev/production serving
4. **Best Practices**: File-based routing, layouts, and proper project structure
5. **Developer Experience**: Hot reload, TypeScript support, and modern tooling

## Template Hierarchy

```
cmd/cli/templates/frontend/sveltekit/
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
└── client/                    # SvelteKit application
    ├── package.json           # Node dependencies
    ├── svelte.config.js       # SvelteKit configuration
    ├── vite.config.ts         # Vite configuration
    ├── tsconfig.json          # TypeScript config
    ├── tailwind.config.ts     # Tailwind CSS config
    ├── postcss.config.js      # PostCSS config for Tailwind
    ├── src/
    │   ├── app.html           # HTML template
    │   ├── app.css            # Global styles with Tailwind
    │   ├── app.d.ts           # TypeScript declarations
    │   ├── lib/
    │   │   └── index.ts       # Library exports ($lib alias)
    │   └── routes/
    │       ├── +layout.svelte # Root layout with navigation
    │       ├── +page.svelte   # Home page
    │       └── about/
    │           └── +page.svelte # About page
    └── static/
        └── favicon.svg        # Favicon (Svelte logo)
```

## Template Metadata

### template.json

```json
{
  "name": "frontend/sveltekit",
  "description": "SvelteKit SPA with TypeScript, Tailwind, and Mizu backend",
  "tags": ["go", "mizu", "frontend", "spa", "sveltekit", "svelte", "typescript", "tailwind"],
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

## SvelteKit Frontend Files

### client/package.json

```json
{
  "name": "{{.Name}}-client",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite dev --port 5173",
    "build": "vite build",
    "preview": "vite preview",
    "check": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json",
    "check:watch": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json --watch"
  },
  "dependencies": {
    "@sveltejs/adapter-static": "^3.0.6",
    "@sveltejs/kit": "^2.9.0",
    "@sveltejs/vite-plugin-svelte": "^4.0.0",
    "svelte": "^5.12.0"
  },
  "devDependencies": {
    "@tailwindcss/postcss": "^4.0.0",
    "autoprefixer": "^10.4.20",
    "postcss": "^8.4.49",
    "svelte-check": "^4.1.1",
    "tailwindcss": "^4.0.0",
    "typescript": "^5.7.2",
    "vite": "^6.0.3"
  }
}
```

### client/svelte.config.js

SvelteKit configured for static generation with API proxy for development:

```javascript
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      pages: '../dist',
      assets: '../dist',
      fallback: 'index.html',
      precompress: false,
      strict: true
    }),
    paths: {
      base: ''
    }
  }
};

export default config;
```

### client/vite.config.ts

```typescript
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
    },
  },
});
```

### client/tsconfig.json

```json
{
  "extends": "./.svelte-kit/tsconfig.json",
  "compilerOptions": {
    "allowJs": true,
    "checkJs": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "skipLibCheck": true,
    "sourceMap": true,
    "strict": true,
    "moduleResolution": "bundler"
  }
}
```

### client/tailwind.config.ts

```typescript
import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {},
  },
  plugins: [],
};

export default config;
```

### client/postcss.config.js

```javascript
export default {
  plugins: {
    '@tailwindcss/postcss': {},
    autoprefixer: {},
  },
};
```

### client/src/app.html

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <link rel="icon" type="image/svg+xml" href="%sveltekit.assets%/favicon.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    %sveltekit.head%
  </head>
  <body data-sveltekit-preload-data="hover">
    <div style="display: contents">%sveltekit.body%</div>
  </body>
</html>
```

### client/src/app.css

```css
@import 'tailwindcss';
```

### client/src/app.d.ts

```typescript
declare global {
  namespace App {
    // interface Error {}
    // interface Locals {}
    // interface PageData {}
    // interface PageState {}
    // interface Platform {}
  }
}

export {};
```

### client/src/lib/index.ts

```typescript
// Library exports - place your shared code here
// Import using $lib alias: import { ... } from '$lib';
```

### client/src/routes/+layout.svelte

```svelte
<script lang="ts">
  import '../app.css';

  let { children } = $props();

  const navLinks = [
    { href: '/', label: 'Home' },
    { href: '/about', label: 'About' },
  ];
</script>

<svelte:head>
  <title>{{.Name}}</title>
</svelte:head>

<div class="min-h-screen bg-slate-50 text-slate-900 flex flex-col">
  <header class="bg-white border-b border-slate-200">
    <nav class="max-w-5xl mx-auto px-8 py-4">
      <ul class="flex gap-6">
        {#each navLinks as link}
          <li>
            <a
              href={link.href}
              class="font-medium text-slate-600 hover:text-blue-600 transition-colors"
            >
              {link.label}
            </a>
          </li>
        {/each}
      </ul>
    </nav>
  </header>

  <main class="flex-1 max-w-5xl mx-auto w-full p-8">
    {@render children()}
  </main>

  <footer class="border-t border-slate-200 bg-white py-4 text-center text-slate-500">
    <p>Built with Mizu + SvelteKit</p>
  </footer>
</div>
```

### client/src/routes/+page.svelte

```svelte
<script lang="ts">
  import { onMount } from 'svelte';

  let message = $state('');
  let loading = $state(true);

  onMount(async () => {
    try {
      const res = await fetch('/api/hello');
      const data = await res.json();
      message = data.message;
    } catch {
      message = 'Failed to load message';
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head>
  <title>Home - {{.Name}}</title>
</svelte:head>

<div class="space-y-6">
  <h1 class="text-4xl font-bold">Welcome to {{.Name}}</h1>
  {#if loading}
    <p class="text-slate-500">Loading...</p>
  {:else}
    <div class="bg-white rounded-lg border border-slate-200 p-4">
      <p class="text-lg">{message}</p>
    </div>
  {/if}
</div>
```

### client/src/routes/about/+page.svelte

```svelte
<svelte:head>
  <title>About - {{.Name}}</title>
</svelte:head>

<div class="space-y-6">
  <h1 class="text-4xl font-bold">About</h1>
  <p class="text-slate-600">
    This is a SvelteKit SPA powered by Mizu, a lightweight Go web framework.
  </p>
  <ul class="list-disc list-inside space-y-2 text-slate-600">
    <li>SvelteKit 2 with Svelte 5</li>
    <li>Static adapter for SPA deployment</li>
    <li>File-based routing with layouts</li>
    <li>Tailwind CSS for styling</li>
    <li>Mizu backend with the frontend middleware</li>
  </ul>
</div>
```

## Makefile

### Makefile.tmpl

```makefile
.PHONY: dev build run clean install

# Development mode: run SvelteKit dev server and Go server concurrently
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
	rm -rf client/.svelte-kit

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

// FS contains the built SvelteKit static export.
// After running `npm run build` in the client directory,
// the dist folder will contain the production-ready static assets.
//
//go:embed all:*
var FS embed.FS
```

### dist/placeholder.txt

```
This directory will contain the built SvelteKit static assets.
Run 'make build' or 'cd client && npm run build' to generate them.
```

## CLI Usage

```bash
# Create a new SvelteKit SPA project
mizu new ./myapp --template frontend/sveltekit

# With custom module path
mizu new ./myapp --template frontend/sveltekit --module github.com/user/myapp

# Preview the generated files
mizu new ./myapp --template frontend/sveltekit --dry-run
```

## Key Differences from Other SPA Templates

| Feature | SvelteKit | Svelte (Vite) | Nuxt | Next.js |
|---------|-----------|---------------|------|---------|
| Framework | Svelte 5 | Svelte 5 | Vue 3 | React 19 |
| Build Tool | SvelteKit/Vite | Vite | Nuxt CLI | Next.js CLI |
| Router | File-based (built-in) | svelte-routing | File-based (built-in) | App Router (built-in) |
| Layouts | File-based (+layout.svelte) | Manual | layouts/ directory | layout.tsx |
| Auto-imports | No | No | Yes | No |
| Dev Port | 5173 | 5173 | 3001 | 3001 |
| Output | Static export to `dist/` | `dist/` | Static export to `dist/` | Static export to `dist/` |
| Styling | Tailwind CSS | Custom CSS | Tailwind CSS | Tailwind CSS |

## SvelteKit-Specific Features

1. **File-based Routing**: Routes in `src/routes/` automatically become pages. `+page.svelte` files define page content.

2. **Layouts**: `+layout.svelte` files provide shared layouts that wrap child routes. The root layout applies to all pages.

3. **Static Adapter**: The `@sveltejs/adapter-static` generates a fully static site with SPA fallback, perfect for embedding in the Go binary.

4. **Svelte 5 Runes**: Uses Svelte 5's new reactivity system with `$state`, `$derived`, and `$props()`.

5. **TypeScript Support**: Full TypeScript support with auto-generated types in `.svelte-kit/`.

6. **$lib Alias**: The `$lib` import alias provides convenient access to shared code in `src/lib/`.

7. **Preload Hints**: SvelteKit automatically generates preload hints for better performance.

## Comparison: SvelteKit vs Svelte (Vite)

| Aspect | SvelteKit | Svelte (Vite) |
|--------|-----------|---------------|
| Purpose | Full-stack framework | UI library with build tool |
| Routing | Built-in file-based | Requires svelte-routing |
| Layouts | Built-in (+layout.svelte) | Manual implementation |
| API Routes | Supported (+server.ts) | Not available |
| SSR/SSG | Built-in (configurable) | Client-only |
| Adapter System | Yes (static, node, etc.) | No |
| Best For | Full applications | Simple SPAs, embedded UIs |

## Testing Strategy

### Unit Tests

```go
func TestSvelteKitTemplateExists(t *testing.T) {
    if !templateExists("frontend/sveltekit") {
        t.Error("templateExists('frontend/sveltekit') returned false")
    }
}

func TestLoadTemplateFilesSvelteKit(t *testing.T) {
    files, err := loadTemplateFiles("frontend/sveltekit")
    require.NoError(t, err)

    expectedFiles := []string{
        ".gitignore",
        "go.mod",
        "cmd/server/main.go",
        "app/server/app.go",
        "client/package.json",
        "client/svelte.config.js",
        "client/vite.config.ts",
        "client/src/app.html",
        "client/src/routes/+layout.svelte",
        "client/src/routes/+page.svelte",
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

func TestSvelteKitTemplateHasSvelteKitSpecificContent(t *testing.T) {
    files, err := loadTemplateFiles("frontend/sveltekit")
    require.NoError(t, err)

    fileMap := make(map[string]templateFile)
    for _, f := range files {
        fileMap[f.path] = f
    }

    // Verify svelte.config.js contains adapter-static
    svelteConfig := fileMap["client/svelte.config.js"]
    if !strings.Contains(svelteConfig.content, "adapter-static") {
        t.Error("svelte.config.js should import adapter-static")
    }

    // Verify package.json contains "@sveltejs/kit"
    packageJSON := fileMap["client/package.json"]
    if !strings.Contains(packageJSON.content, `"@sveltejs/kit"`) {
        t.Error("package.json should contain @sveltejs/kit dependency")
    }

    // Verify +layout.svelte uses Svelte 5 syntax
    layout := fileMap["client/src/routes/+layout.svelte"]
    if !strings.Contains(layout.content, "$props()") {
        t.Error("+layout.svelte should use Svelte 5 $props()")
    }
    if !strings.Contains(layout.content, "{@render children()}") {
        t.Error("+layout.svelte should use Svelte 5 {@render} for children")
    }
}

func TestApplyPlanSvelteKitTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/sveltekit", tmpDir, vars)
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
        "client/svelte.config.js",
        "client/vite.config.ts",
        "client/src/app.html",
        "client/src/routes/+layout.svelte",
        "client/src/routes/+page.svelte",
        "client/src/routes/about/+page.svelte",
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

1. **Static Adapter**: SvelteKit's `@sveltejs/adapter-static` generates static HTML/JS/CSS with a fallback `index.html` for client-side routing. This is perfect for embedding in Mizu's frontend middleware.

2. **API Proxying**: During development, Vite proxies `/api/*` requests to the Go backend. In production, both frontend and API are served from the same origin.

3. **Port Configuration**: SvelteKit dev server runs on port 5173 (Vite default) to avoid conflict with the Go server on port 3000.

4. **SPA Fallback**: The `fallback: 'index.html'` option in the static adapter ensures client-side routing works for all routes.

5. **Tailwind CSS 4**: Uses the latest Tailwind CSS 4 with the new `@tailwindcss/postcss` plugin.

6. **Svelte 5**: Uses Svelte 5's new runes system (`$state`, `$derived`, `$props()`) for modern reactivity.

7. **Clean .svelte-kit**: The clean target removes the `.svelte-kit` directory which contains generated types and build artifacts.
