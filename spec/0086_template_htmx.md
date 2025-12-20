# HTMX Template Design Spec

## Overview

This specification describes the implementation of an HTMX template for the `frontend/spa/htmx` path. Unlike other SPA templates (React, Vue, etc.), HTMX uses a hypermedia-driven approach with server-side rendering, eliminating the need for Node.js build tools.

## Goals

1. **No Build Step**: Pure HTML/CSS/JS with no Node.js or npm required
2. **Server-Side Rendering**: Go templates for HTML generation
3. **Hypermedia-Driven**: HTMX for dynamic updates via HTML fragments
4. **Lightweight Client**: Optional Alpine.js for minimal client-side interactivity
5. **Modern Styling**: Tailwind CSS via CDN for rapid styling
6. **Hot Reload**: Live reload during development via HTMX polling or WebSocket
7. **Production Ready**: Embedded assets, proper caching, and security headers

## Why HTMX?

HTMX represents the "hypermedia" approach to web development:

- **Simpler Mental Model**: HTML over the wire instead of JSON APIs
- **Less JavaScript**: Server handles logic; HTMX handles dynamic updates
- **Progressively Enhanced**: Works without JavaScript (degraded but functional)
- **No Build Pipeline**: Deploy Go binary with embedded assets
- **Smaller Bundles**: ~14KB for HTMX vs 40KB+ React runtime

## Template Structure

```
cmd/cli/templates/frontend/spa/htmx/
├── template.json
├── Makefile.tmpl
├── cmd/server/main.go.tmpl
├── app/
│   └── server/
│       ├── app.go.tmpl
│       ├── config.go.tmpl
│       ├── routes.go.tmpl
│       ├── handlers.go.tmpl
│       └── views.go.tmpl
├── views/
│   ├── embed.go.tmpl
│   ├── layouts/
│   │   └── base.html
│   ├── pages/
│   │   ├── home.html
│   │   └── about.html
│   ├── partials/
│   │   ├── header.html
│   │   ├── footer.html
│   │   └── nav.html
│   └── components/
│       ├── greeting.html
│       └── counter.html
├── static/
│   ├── embed.go.tmpl
│   ├── css/
│   │   └── app.css
│   └── js/
│       └── app.js
└── dist/
    └── placeholder.txt
```

## Key Design Decisions

### 1. Template Engine Integration

Use Mizu's `view` package for server-side rendering:

```go
import "github.com/go-mizu/mizu/view"

// Initialize views with embedded templates
views := view.New(viewsFS, view.Options{
    Extension: ".html",
    Layouts:   "layouts",
})

// Render a page
c.Render("pages/home", data)
```

### 2. HTMX Patterns

#### Partial Updates
```html
<!-- Trigger partial update -->
<button hx-get="/api/greeting" hx-target="#greeting" hx-swap="innerHTML">
    Refresh Greeting
</button>

<!-- Target element -->
<div id="greeting">
    {{ template "components/greeting" . }}
</div>
```

#### Form Handling
```html
<form hx-post="/api/contact" hx-target="#result" hx-swap="innerHTML">
    <input type="text" name="email" required>
    <button type="submit">Submit</button>
</form>
```

#### Infinite Scroll
```html
<div hx-get="/api/items?page=2" hx-trigger="revealed" hx-swap="afterend">
    <!-- Items here -->
</div>
```

### 3. Alpine.js Integration (Optional)

For client-side state that doesn't need server roundtrips:

```html
<div x-data="{ count: 0 }">
    <button @click="count++">Count: <span x-text="count"></span></button>
</div>
```

### 4. No Frontend Middleware Needed

Unlike other SPA templates, HTMX doesn't require the `frontend` middleware:

```go
// Other SPAs: frontend.WithOptions(...)
// HTMX: Direct static file serving + view rendering

app.Static("/static", staticFS)
app.Get("/", homeHandler)
```

## Template Files

### template.json

```json
{
  "name": "frontend/spa/htmx",
  "description": "HTMX hypermedia app with Go templates and Tailwind CSS",
  "tags": ["go", "mizu", "frontend", "spa", "htmx", "tailwind", "alpine"],
  "variables": {
    "name": { "description": "Project name", "default": "" },
    "module": { "description": "Go module path", "default": "" }
  }
}
```

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
    "github.com/go-mizu/mizu"
    "github.com/go-mizu/mizu/view"

    "{{.Module}}/static"
    "{{.Module}}/views"
)

func New(cfg *Config) *mizu.App {
    app := mizu.New()

    // Initialize view engine
    v := view.New(views.FS, view.Options{
        Extension: ".html",
        Layouts:   "layouts",
    })
    app.SetView(v)

    // Serve static assets
    app.Static("/static", static.FS)

    // Setup routes
    setupRoutes(app)

    return app
}
```

### app/server/config.go.tmpl

```go
package server

import "os"

type Config struct {
    Port string
    Env  string
}

func LoadConfig() *Config {
    return &Config{
        Port: getEnv("PORT", "3000"),
        Env:  getEnv("MIZU_ENV", "development"),
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

import "github.com/go-mizu/mizu"

func setupRoutes(app *mizu.App) {
    // Pages (full page renders)
    app.Get("/", homeHandler)
    app.Get("/about", aboutHandler)

    // API/HTMX endpoints (partial renders)
    api := app.Prefix("/api")
    api.Get("/health", healthHandler)
    api.Get("/greeting", greetingHandler)
    api.Post("/counter/increment", counterIncrementHandler)
    api.Post("/counter/decrement", counterDecrementHandler)
}
```

### app/server/handlers.go.tmpl

```go
package server

import "github.com/go-mizu/mizu"

func homeHandler(c *mizu.Ctx) error {
    return c.Render("pages/home", mizu.Map{
        "title":   "Home",
        "message": "Hello from {{.Name}}!",
        "count":   0,
    })
}

func aboutHandler(c *mizu.Ctx) error {
    return c.Render("pages/about", mizu.Map{
        "title": "About",
    })
}

func healthHandler(c *mizu.Ctx) error {
    return c.JSON(200, mizu.Map{"status": "ok"})
}

func greetingHandler(c *mizu.Ctx) error {
    return c.Render("components/greeting", mizu.Map{
        "message": "Hello from {{.Name}}!",
    }, view.WithoutLayout())
}

func counterIncrementHandler(c *mizu.Ctx) error {
    count := c.QueryInt("count", 0) + 1
    return c.Render("components/counter", mizu.Map{
        "count": count,
    }, view.WithoutLayout())
}

func counterDecrementHandler(c *mizu.Ctx) error {
    count := c.QueryInt("count", 0) - 1
    return c.Render("components/counter", mizu.Map{
        "count": count,
    }, view.WithoutLayout())
}
```

### views/embed.go.tmpl

```go
package views

import "embed"

//go:embed all:layouts all:pages all:partials all:components
var FS embed.FS
```

### views/layouts/base.html

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{`{{.title}}`}} - {{.Name}}</title>

    <!-- Tailwind CSS via CDN -->
    <script src="https://cdn.tailwindcss.com"></script>

    <!-- HTMX -->
    <script src="https://unpkg.com/htmx.org@2.0.4"
            integrity="sha384-HGfztofotfshcF7+8n44JQL2oJmowVChPTg48S+jvZoztPfvwD79OC/LTtG6dMp+"
            crossorigin="anonymous"></script>

    <!-- Alpine.js (optional, for client-side state) -->
    <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>

    <!-- App styles -->
    <link rel="stylesheet" href="/static/css/app.css">
</head>
<body class="min-h-screen bg-gray-50 text-gray-900">
    {{`{{template "partials/header" .}}`}}

    <main class="container mx-auto px-4 py-8">
        {{`{{embed}}`}}
    </main>

    {{`{{template "partials/footer" .}}`}}

    <!-- App scripts -->
    <script src="/static/js/app.js"></script>
</body>
</html>
```

### views/partials/header.html

```html
<header class="bg-white shadow-sm">
    <nav class="container mx-auto px-4 py-4">
        <div class="flex items-center justify-between">
            <a href="/" class="text-xl font-bold text-blue-600">{{.Name}}</a>
            <div class="flex gap-6">
                <a href="/" class="text-gray-600 hover:text-blue-600 transition-colors"
                   hx-get="/" hx-target="body" hx-push-url="true">
                    Home
                </a>
                <a href="/about" class="text-gray-600 hover:text-blue-600 transition-colors"
                   hx-get="/about" hx-target="body" hx-push-url="true">
                    About
                </a>
            </div>
        </div>
    </nav>
</header>
```

### views/partials/footer.html

```html
<footer class="bg-white border-t border-gray-200 mt-auto">
    <div class="container mx-auto px-4 py-6 text-center text-gray-500">
        <p>Built with <a href="https://github.com/go-mizu/mizu" class="text-blue-600 hover:underline">Mizu</a> +
           <a href="https://htmx.org" class="text-blue-600 hover:underline">HTMX</a></p>
    </div>
</footer>
```

### views/pages/home.html

```html
{{`{{define "content"}}`}}
<div class="max-w-2xl mx-auto space-y-8">
    <section class="text-center">
        <h1 class="text-4xl font-bold text-gray-900 mb-4">Welcome to {{.Name}}</h1>
        <p class="text-lg text-gray-600">A hypermedia-driven application built with HTMX</p>
    </section>

    <!-- Dynamic greeting section -->
    <section class="bg-white rounded-lg shadow p-6">
        <h2 class="text-xl font-semibold mb-4">Server Message</h2>
        <div id="greeting" class="p-4 bg-blue-50 rounded text-blue-800">
            {{`{{template "components/greeting" .}}`}}
        </div>
        <button hx-get="/api/greeting"
                hx-target="#greeting"
                hx-swap="innerHTML"
                class="mt-4 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors">
            Refresh Message
        </button>
    </section>

    <!-- Interactive counter -->
    <section class="bg-white rounded-lg shadow p-6">
        <h2 class="text-xl font-semibold mb-4">HTMX Counter</h2>
        <div id="counter">
            {{`{{template "components/counter" .}}`}}
        </div>
    </section>

    <!-- Alpine.js example -->
    <section class="bg-white rounded-lg shadow p-6">
        <h2 class="text-xl font-semibold mb-4">Alpine.js Counter (Client-side)</h2>
        <div x-data="{ count: 0 }" class="flex items-center gap-4">
            <button @click="count--"
                    class="px-4 py-2 bg-gray-200 rounded hover:bg-gray-300 transition-colors">
                -
            </button>
            <span x-text="count" class="text-2xl font-bold w-16 text-center"></span>
            <button @click="count++"
                    class="px-4 py-2 bg-gray-200 rounded hover:bg-gray-300 transition-colors">
                +
            </button>
        </div>
        <p class="mt-2 text-sm text-gray-500">This counter uses Alpine.js (no server requests)</p>
    </section>
</div>
{{`{{end}}`}}
```

### views/pages/about.html

```html
{{`{{define "content"}}`}}
<div class="max-w-2xl mx-auto">
    <h1 class="text-4xl font-bold text-gray-900 mb-6">About</h1>

    <div class="prose prose-lg">
        <p class="text-gray-600 mb-4">
            This is an HTMX-powered application built with the Mizu web framework.
        </p>

        <h2 class="text-2xl font-semibold mt-8 mb-4">Stack</h2>
        <ul class="space-y-2 text-gray-600">
            <li class="flex items-center gap-2">
                <span class="text-green-500">&#10003;</span>
                <strong>Mizu</strong> - Lightweight Go web framework
            </li>
            <li class="flex items-center gap-2">
                <span class="text-green-500">&#10003;</span>
                <strong>HTMX</strong> - HTML over the wire
            </li>
            <li class="flex items-center gap-2">
                <span class="text-green-500">&#10003;</span>
                <strong>Alpine.js</strong> - Lightweight reactivity
            </li>
            <li class="flex items-center gap-2">
                <span class="text-green-500">&#10003;</span>
                <strong>Tailwind CSS</strong> - Utility-first styling
            </li>
        </ul>

        <h2 class="text-2xl font-semibold mt-8 mb-4">Benefits</h2>
        <ul class="space-y-2 text-gray-600">
            <li>&#8226; No build step required</li>
            <li>&#8226; Single Go binary deployment</li>
            <li>&#8226; Server-side rendering with Go templates</li>
            <li>&#8226; Progressive enhancement</li>
            <li>&#8226; Minimal JavaScript footprint</li>
        </ul>
    </div>
</div>
{{`{{end}}`}}
```

### views/components/greeting.html

```html
<p class="font-medium">{{`{{.message}}`}}</p>
<p class="text-sm text-blue-600 mt-1">Updated at: {{`{{now | date "15:04:05"}}`}}</p>
```

### views/components/counter.html

```html
<div class="flex items-center gap-4">
    <button hx-post="/api/counter/decrement?count={{`{{.count}}`}}"
            hx-target="#counter"
            hx-swap="innerHTML"
            class="px-4 py-2 bg-red-100 text-red-700 rounded hover:bg-red-200 transition-colors">
        -
    </button>
    <span class="text-2xl font-bold w-16 text-center">{{`{{.count}}`}}</span>
    <button hx-post="/api/counter/increment?count={{`{{.count}}`}}"
            hx-target="#counter"
            hx-swap="innerHTML"
            class="px-4 py-2 bg-green-100 text-green-700 rounded hover:bg-green-200 transition-colors">
        +
    </button>
</div>
<p class="mt-2 text-sm text-gray-500">This counter uses HTMX (server requests)</p>
```

### static/embed.go.tmpl

```go
package static

import "embed"

//go:embed all:css all:js
var FS embed.FS
```

### static/css/app.css

```css
/* Custom styles beyond Tailwind */

/* HTMX loading indicator */
.htmx-request .htmx-indicator {
    display: inline-block;
}

.htmx-indicator {
    display: none;
}

/* Smooth transitions for HTMX swaps */
.htmx-swapping {
    opacity: 0;
    transition: opacity 200ms ease-out;
}

.htmx-settling {
    opacity: 1;
    transition: opacity 200ms ease-in;
}

/* Focus styles */
button:focus-visible,
a:focus-visible,
input:focus-visible {
    outline: 2px solid #3b82f6;
    outline-offset: 2px;
}
```

### static/js/app.js

```javascript
// HTMX configuration
document.body.addEventListener('htmx:configRequest', function(event) {
    // Add any custom headers here
    // event.detail.headers['X-Custom-Header'] = 'value';
});

// Handle HTMX errors
document.body.addEventListener('htmx:responseError', function(event) {
    console.error('HTMX request failed:', event.detail.error);
});

// Loading indicator
document.body.addEventListener('htmx:beforeRequest', function(event) {
    event.target.classList.add('opacity-50');
});

document.body.addEventListener('htmx:afterRequest', function(event) {
    event.target.classList.remove('opacity-50');
});
```

### Makefile.tmpl

```makefile
.PHONY: dev build run clean

# Development mode with live reload
dev:
	@echo "Starting development server..."
	MIZU_ENV=development go run ./cmd/server

# Build production binary
build:
	@echo "Building..."
	go build -o bin/server ./cmd/server
	@echo "Build complete: bin/server"

# Run production server
run: build
	@echo "Starting production server..."
	MIZU_ENV=production ./bin/server

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
install:
	go mod tidy
```

### dist/placeholder.txt

```
This directory is a placeholder for the dist folder.
In the HTMX template, assets are served directly from static/ and views/.
```

## Development Workflow

### Starting Development

```bash
# Create new project
mizu new ./myapp --template frontend/spa/htmx

# Start development server
cd myapp
make dev

# Open browser at http://localhost:3000
```

### File Structure After Creation

```
myapp/
├── go.mod
├── Makefile
├── cmd/
│   └── server/
│       └── main.go
├── app/
│   └── server/
│       ├── app.go
│       ├── config.go
│       ├── routes.go
│       └── handlers.go
├── views/
│   ├── embed.go
│   ├── layouts/
│   ├── pages/
│   ├── partials/
│   └── components/
└── static/
    ├── embed.go
    ├── css/
    └── js/
```

## Comparison with Other Templates

| Aspect | HTMX | React/Vue/Svelte | Angular |
|--------|------|------------------|---------|
| Build Tool | None | Vite | Angular CLI |
| Node.js | No | Yes | Yes |
| Bundle Size | ~14KB (HTMX) | 40-100KB+ | 100KB+ |
| Rendering | Server | Client | Client |
| State | Server | Client | Client |
| API Format | HTML | JSON | JSON |
| Hot Reload | Go (fsnotify) | Vite HMR | ng serve |

## Testing Strategy

### Unit Tests

```go
func TestHTMXTemplateGeneration(t *testing.T) {
    tmpDir := t.TempDir()

    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)
    p, err := buildPlan("frontend/spa/htmx", tmpDir, vars)
    require.NoError(t, err)

    err = p.apply(false)
    require.NoError(t, err)

    // Verify key files exist
    require.FileExists(t, filepath.Join(tmpDir, "go.mod"))
    require.FileExists(t, filepath.Join(tmpDir, "cmd/server/main.go"))
    require.FileExists(t, filepath.Join(tmpDir, "app/server/app.go"))
    require.FileExists(t, filepath.Join(tmpDir, "app/server/handlers.go"))
    require.FileExists(t, filepath.Join(tmpDir, "views/embed.go"))
    require.FileExists(t, filepath.Join(tmpDir, "views/layouts/base.html"))
    require.FileExists(t, filepath.Join(tmpDir, "views/pages/home.html"))
    require.FileExists(t, filepath.Join(tmpDir, "static/embed.go"))

    // Verify no package.json (no Node.js)
    require.NoFileExists(t, filepath.Join(tmpDir, "package.json"))
    require.NoFileExists(t, filepath.Join(tmpDir, "client/package.json"))
}

func TestHTMXTemplateBuildable(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping build test in short mode")
    }

    tmpDir := t.TempDir()
    vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)
    p, _ := buildPlan("frontend/spa/htmx", tmpDir, vars)
    p.apply(false)

    // Verify Go code compiles
    cmd := exec.Command("go", "build", "./...")
    cmd.Dir = tmpDir
    output, err := cmd.CombinedOutput()
    require.NoError(t, err, "Go build failed: %s", output)
}

func TestHTMXTemplateVariableSubstitution(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myhtmxapp", "github.com/user/myhtmxapp", "MIT", nil)
    p, _ := buildPlan("frontend/spa/htmx", tmpDir, vars)
    p.apply(false)

    // Check module path in go.mod
    content, _ := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
    require.Contains(t, string(content), "github.com/user/myhtmxapp")

    // Check project name in handlers
    handlers, _ := os.ReadFile(filepath.Join(tmpDir, "app/server/handlers.go"))
    require.Contains(t, string(handlers), "myhtmxapp")
}
```

### Integration Tests

```go
func TestHTMXTemplateServer(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    tmpDir := t.TempDir()
    vars := newTemplateVars("testapp", "example.com/testapp", "MIT", nil)
    p, _ := buildPlan("frontend/spa/htmx", tmpDir, vars)
    p.apply(false)

    // Build and run
    cmd := exec.Command("go", "run", "./cmd/server")
    cmd.Dir = tmpDir
    cmd.Env = append(os.Environ(), "PORT=0") // Random port

    // Start server and test endpoints
    // ...
}
```

## Implementation Plan

### Phase 1: Template Structure
1. Create `templates/frontend/spa/htmx/` directory
2. Create `template.json` metadata
3. Create placeholder files

### Phase 2: Go Backend
1. Implement `cmd/server/main.go.tmpl`
2. Implement `app/server/*.go.tmpl` files
3. Implement view embedding

### Phase 3: Views and Static Assets
1. Create HTML templates (layouts, pages, partials, components)
2. Create CSS and JavaScript files
3. Setup proper embed directives

### Phase 4: Testing
1. Add template generation tests
2. Add build verification tests
3. Add variable substitution tests

### Phase 5: Documentation
1. Update CLI help with new template
2. Add examples to README

## Future Enhancements

1. **Live Reload**: Add fsnotify-based live reload for development
2. **Tailwind Build**: Optional Tailwind CLI integration for production CSS
3. **SSE Support**: Server-Sent Events for real-time updates
4. **WebSocket**: HTMX WebSocket extension support
5. **Form Validation**: Server-side validation with HTMX responses
6. **Toast Notifications**: OOB swaps for notification system
