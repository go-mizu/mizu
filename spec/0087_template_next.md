# Next.js SPA Template Design Spec

## Overview

This specification describes the implementation of a Next.js SPA template for the Mizu CLI. Next.js is a React framework that provides file-based routing, automatic code splitting, and optimized builds out of the box. This template uses Next.js in **static export mode** (SPA) to generate a static site that can be served by the Mizu backend.

## Goals

1. **Modern Next.js Setup**: Next.js 14+ with App Router, TypeScript, and Tailwind CSS
2. **Static Export**: Configure Next.js for static site generation (output: 'export')
3. **Mizu Integration**: Seamless integration with the `frontend` middleware for dev/production serving
4. **Best Practices**: File-based routing, proper project structure, and optimized builds
5. **Developer Experience**: Fast refresh, TypeScript support, and modern tooling

## Template Hierarchy

```
cmd/cli/templates/frontend/next/
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
│   └── placeholder.txt        # Output directory placeholder
└── client/                    # Next.js application
    ├── package.json           # Node dependencies
    ├── next.config.ts         # Next.js configuration
    ├── tsconfig.json          # TypeScript config
    ├── tailwind.config.ts     # Tailwind CSS config
    ├── postcss.config.mjs     # PostCSS config
    ├── src/
    │   ├── app/               # App Router pages
    │   │   ├── layout.tsx     # Root layout
    │   │   ├── page.tsx       # Home page
    │   │   └── about/
    │   │       └── page.tsx   # About page
    │   ├── components/        # Shared components
    │   │   ├── Layout.tsx     # Layout component
    │   │   └── Navigation.tsx # Navigation component
    │   └── styles/
    │       └── globals.css    # Global styles with Tailwind
    └── public/
        └── next.svg           # Static assets
```

## Template Metadata

### template.json

```json
{
  "name": "frontend/next",
  "description": "Next.js SPA with App Router, TypeScript, Tailwind, and Mizu backend",
  "tags": ["go", "mizu", "frontend", "spa", "next", "nextjs", "react", "typescript", "tailwind"],
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

## Next.js Frontend Files

### client/package.json

```json
{
  "name": "{{.Name}}-client",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev -p 3001",
    "build": "next build",
    "start": "next start",
    "lint": "next lint"
  },
  "dependencies": {
    "next": "^15.1.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  },
  "devDependencies": {
    "@types/node": "^22.10.2",
    "@types/react": "^19.0.1",
    "@types/react-dom": "^19.0.1",
    "postcss": "^8.4.49",
    "tailwindcss": "^3.4.16",
    "typescript": "^5.7.2"
  }
}
```

### client/next.config.ts

Next.js configured for static export with API rewrites for development:

```typescript
import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
  output: 'export',
  distDir: '../dist',
  images: {
    unoptimized: true,
  },
  // Rewrite API calls to Go backend in development
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:3000/api/:path*',
      },
    ]
  },
}

export default nextConfig
```

### client/tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2017",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "plugins": [
      {
        "name": "next"
      }
    ],
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  "exclude": ["node_modules"]
}
```

### client/tailwind.config.ts

```typescript
import type { Config } from 'tailwindcss'

const config: Config = {
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}

export default config
```

### client/postcss.config.mjs

```javascript
const config = {
  plugins: {
    tailwindcss: {},
  },
}

export default config
```

### client/src/app/layout.tsx

```tsx
import type { Metadata } from 'next'
import '@/styles/globals.css'
import Navigation from '@/components/Navigation'

export const metadata: Metadata = {
  title: '{{.Name}}',
  description: 'Built with Next.js and Mizu',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-slate-50 text-slate-900">
        <div className="flex flex-col min-h-screen">
          <Navigation />
          <main className="flex-1 max-w-5xl mx-auto w-full p-8">
            {children}
          </main>
          <footer className="border-t border-slate-200 bg-white py-4 text-center text-slate-500">
            <p>Built with Mizu + Next.js</p>
          </footer>
        </div>
      </body>
    </html>
  )
}
```

### client/src/app/page.tsx

```tsx
'use client'

import { useState, useEffect } from 'react'

export default function Home() {
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
    <div className="space-y-6">
      <h1 className="text-4xl font-bold">Welcome to {{.Name}}</h1>
      {loading ? (
        <p className="text-slate-500">Loading...</p>
      ) : (
        <div className="bg-white rounded-lg border border-slate-200 p-4">
          <p className="text-lg">{message}</p>
        </div>
      )}
    </div>
  )
}
```

### client/src/app/about/page.tsx

```tsx
export default function About() {
  return (
    <div className="space-y-6">
      <h1 className="text-4xl font-bold">About</h1>
      <p className="text-slate-600">
        This is a Next.js SPA powered by Mizu, a lightweight Go web framework.
      </p>
      <ul className="list-disc list-inside space-y-2 text-slate-600">
        <li>Next.js 15 with App Router and TypeScript</li>
        <li>Static export for SPA deployment</li>
        <li>Tailwind CSS for styling</li>
        <li>Mizu backend with the frontend middleware</li>
      </ul>
    </div>
  )
}
```

### client/src/components/Navigation.tsx

```tsx
'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

export default function Navigation() {
  const pathname = usePathname()

  const navLinks = [
    { href: '/', label: 'Home' },
    { href: '/about', label: 'About' },
  ]

  return (
    <header className="bg-white border-b border-slate-200">
      <nav className="max-w-5xl mx-auto px-8 py-4">
        <ul className="flex gap-6">
          {navLinks.map(({ href, label }) => (
            <li key={href}>
              <Link
                href={href}
                className={`font-medium transition-colors ${
                  pathname === href
                    ? 'text-blue-600'
                    : 'text-slate-600 hover:text-blue-600'
                }`}
              >
                {label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>
    </header>
  )
}
```

### client/src/styles/globals.css

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

## Makefile

### Makefile.tmpl

```makefile
.PHONY: dev build run clean install

# Development mode: run Next.js dev server and Go server concurrently
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
	rm -rf client/.next

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

// FS contains the built Next.js static export.
// After running `npm run build` in the client directory,
// the dist folder will contain the production-ready static assets.
//go:embed all:*
var FS embed.FS
```

### dist/placeholder.txt

```
This directory will contain the built Next.js static assets.
Run 'make build' or 'cd client && npm run build' to generate them.
```

## CLI Usage

```bash
# Create a new Next.js SPA project
mizu new ./myapp --template frontend/next

# With custom module path
mizu new ./myapp --template frontend/next --module github.com/user/myapp

# Preview the generated files
mizu new ./myapp --template frontend/next --dry-run
```

## Key Differences from Other SPA Templates

| Feature | Next.js | React (Vite) | Vue | Angular |
|---------|---------|--------------|-----|---------|
| Build Tool | Next.js CLI | Vite | Vite | Angular CLI |
| Router | App Router (built-in) | React Router | Vue Router | Angular Router |
| Dev Port | 3001 | 5173 | 5173 | 4200 |
| Output | Static export to `dist/` | `dist/` | `dist/` | `dist/` |
| Styling | Tailwind CSS | CSS | CSS | CSS |

## Testing Strategy

### Unit Tests

```go
func TestNextTemplateExists(t *testing.T) {
    if !templateExists("frontend/next") {
        t.Error("templateExists('frontend/next') returned false")
    }
}

func TestLoadTemplateFilesNext(t *testing.T) {
    files, err := loadTemplateFiles("frontend/next")
    require.NoError(t, err)

    expectedFiles := []string{
        ".gitignore",
        "go.mod",
        "cmd/server/main.go",
        "app/server/app.go",
        "client/package.json",
        "client/next.config.ts",
        "client/src/app/layout.tsx",
        "client/src/app/page.tsx",
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

func TestNextTemplateHasNextSpecificContent(t *testing.T) {
    // Verify:
    // - next.config.ts contains output: 'export'
    // - package.json contains "next" dependency
    // - layout.tsx contains RootLayout
    // - page.tsx uses 'use client' directive
}
```

## Implementation Notes

1. **Static Export**: Next.js `output: 'export'` mode generates static HTML/JS/CSS that can be served by any static file server, including Mizu's frontend middleware.

2. **API Proxying**: During development, Next.js rewrites `/api/*` requests to the Go backend. In production, both frontend and API are served from the same origin.

3. **Port Configuration**: Next.js dev server runs on port 3001 to avoid conflict with the Go server on port 3000.

4. **App Router**: Uses Next.js 14+ App Router for modern file-based routing with layouts and server components support (though in static export mode, all components are client-side).

5. **Tailwind CSS**: Included by default for rapid UI development.
