# Angular SPA Template Design Spec

## Overview

This specification describes the implementation of the Angular SPA template for the Mizu CLI. The template follows the same structure as the React, Vue, and Svelte templates but uses Angular 19 with TypeScript, Angular Router for client-side navigation, and the Angular CLI/esbuild for the build system.

## Goals

1. **Modern Angular Setup**: Angular 19 with TypeScript and standalone components
2. **Consistent Structure**: Mirror the React/Vue/Svelte template structure for familiarity
3. **Angular Best Practices**: Standalone components, signals, modern control flow syntax
4. **Mizu Integration**: Seamless integration with the `frontend` middleware
5. **Developer Experience**: Hot reload, dev/production modes, and proper tooling
6. **Enterprise Ready**: Leverage Angular's built-in dependency injection and modularity

## Template Location

```
cmd/cli/templates/frontend/angular/
├── template.json              # Template metadata
├── Makefile.tmpl              # Build and development commands
├── cmd/server/
│   └── main.go.tmpl           # Go entry point
├── app/server/
│   ├── app.go.tmpl            # App setup with embedded frontend
│   ├── config.go.tmpl         # Server configuration
│   └── routes.go.tmpl         # API routes
├── client/
│   ├── package.json           # Angular dependencies
│   ├── angular.json           # Angular CLI configuration
│   ├── tsconfig.json          # TypeScript configuration
│   ├── tsconfig.app.json      # TypeScript config for app
│   ├── index.html             # HTML template (in client/src/)
│   ├── src/
│   │   ├── main.ts            # Angular app entry point
│   │   ├── app/
│   │   │   ├── app.component.ts      # Root component
│   │   │   ├── app.component.html    # Root component template
│   │   │   ├── app.component.css     # Root component styles
│   │   │   ├── app.config.ts         # App configuration
│   │   │   ├── app.routes.ts         # Route definitions
│   │   │   ├── components/
│   │   │   │   ├── layout/
│   │   │   │   │   ├── layout.component.ts
│   │   │   │   │   ├── layout.component.html
│   │   │   │   │   └── layout.component.css
│   │   │   └── pages/
│   │   │       ├── home/
│   │   │       │   ├── home.component.ts
│   │   │       │   ├── home.component.html
│   │   │       │   └── home.component.css
│   │   │       └── about/
│   │   │           ├── about.component.ts
│   │   │           ├── about.component.html
│   │   │           └── about.component.css
│   │   └── styles.css          # Global styles
│   └── public/
│       └── favicon.ico         # Favicon
└── dist/
    └── placeholder.txt        # Placeholder for build output
```

## Template Metadata

### template.json

```json
{
  "name": "frontend/angular",
  "description": "Angular SPA with TypeScript and Mizu backend",
  "tags": ["go", "mizu", "frontend", "spa", "angular", "typescript"],
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

// New creates a new Mizu app configured for the Angular SPA.
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
        DevPort: getEnv("DEV_PORT", "4200"),
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

## Angular Frontend Files

### client/package.json

```json
{
  "name": "{{.Name}}-client",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "ng serve --port 4200 --proxy-config proxy.conf.json",
    "build": "ng build --output-path ../dist",
    "watch": "ng build --watch --output-path ../dist",
    "test": "ng test"
  },
  "dependencies": {
    "@angular/common": "^19.0.0",
    "@angular/compiler": "^19.0.0",
    "@angular/core": "^19.0.0",
    "@angular/forms": "^19.0.0",
    "@angular/platform-browser": "^19.0.0",
    "@angular/router": "^19.0.0",
    "rxjs": "~7.8.0",
    "zone.js": "~0.15.0"
  },
  "devDependencies": {
    "@angular-devkit/build-angular": "^19.0.0",
    "@angular/cli": "^19.0.0",
    "@angular/compiler-cli": "^19.0.0",
    "typescript": "~5.6.3"
  }
}
```

### client/angular.json

```json
{
  "$schema": "./node_modules/@angular/cli/lib/config/schema.json",
  "version": 1,
  "newProjectRoot": "projects",
  "projects": {
    "client": {
      "projectType": "application",
      "root": "",
      "sourceRoot": "src",
      "prefix": "app",
      "architect": {
        "build": {
          "builder": "@angular-devkit/build-angular:application",
          "options": {
            "outputPath": "../dist",
            "index": "src/index.html",
            "browser": "src/main.ts",
            "polyfills": ["zone.js"],
            "tsConfig": "tsconfig.app.json",
            "assets": [
              {
                "glob": "**/*",
                "input": "public"
              }
            ],
            "styles": ["src/styles.css"],
            "scripts": []
          },
          "configurations": {
            "production": {
              "budgets": [
                {
                  "type": "initial",
                  "maximumWarning": "500kB",
                  "maximumError": "1MB"
                }
              ],
              "outputHashing": "all"
            },
            "development": {
              "optimization": false,
              "extractLicenses": false,
              "sourceMap": true
            }
          },
          "defaultConfiguration": "production"
        },
        "serve": {
          "builder": "@angular-devkit/build-angular:dev-server",
          "configurations": {
            "production": {
              "buildTarget": "client:build:production"
            },
            "development": {
              "buildTarget": "client:build:development"
            }
          },
          "defaultConfiguration": "development"
        },
        "test": {
          "builder": "@angular-devkit/build-angular:karma",
          "options": {
            "polyfills": ["zone.js", "zone.js/testing"],
            "tsConfig": "tsconfig.spec.json",
            "assets": [
              {
                "glob": "**/*",
                "input": "public"
              }
            ],
            "styles": ["src/styles.css"],
            "scripts": []
          }
        }
      }
    }
  }
}
```

### client/proxy.conf.json

```json
{
  "/api": {
    "target": "http://localhost:3000",
    "secure": false,
    "changeOrigin": true
  }
}
```

### client/tsconfig.json

```json
{
  "compileOnSave": false,
  "compilerOptions": {
    "outDir": "./dist/out-tsc",
    "strict": true,
    "noImplicitOverride": true,
    "noPropertyAccessFromIndexSignature": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "skipLibCheck": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "sourceMap": true,
    "declaration": false,
    "experimentalDecorators": true,
    "moduleResolution": "bundler",
    "importHelpers": true,
    "target": "ES2022",
    "module": "ES2022",
    "lib": ["ES2022", "dom"],
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "angularCompilerOptions": {
    "enableI18nLegacyMessageIdFormat": false,
    "strictInjectionParameters": true,
    "strictInputAccessModifiers": true,
    "strictTemplates": true
  }
}
```

### client/tsconfig.app.json

```json
{
  "extends": "./tsconfig.json",
  "compilerOptions": {
    "outDir": "./out-tsc/app",
    "types": []
  },
  "files": ["src/main.ts"],
  "include": ["src/**/*.d.ts"]
}
```

### client/src/index.html

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/x-icon" href="favicon.ico" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>{{.Name}}</title>
  </head>
  <body>
    <app-root></app-root>
  </body>
</html>
```

### client/src/main.ts

```typescript
import { bootstrapApplication } from '@angular/platform-browser';
import { appConfig } from './app/app.config';
import { AppComponent } from './app/app.component';

bootstrapApplication(AppComponent, appConfig)
  .catch((err) => console.error(err));
```

### client/src/app/app.config.ts

```typescript
import { ApplicationConfig, provideZoneChangeDetection } from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';

import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(routes),
    provideHttpClient()
  ]
};
```

### client/src/app/app.routes.ts

```typescript
import { Routes } from '@angular/router';
import { LayoutComponent } from './components/layout/layout.component';
import { HomeComponent } from './pages/home/home.component';
import { AboutComponent } from './pages/about/about.component';

export const routes: Routes = [
  {
    path: '',
    component: LayoutComponent,
    children: [
      { path: '', component: HomeComponent },
      { path: 'about', component: AboutComponent }
    ]
  }
];
```

### client/src/app/app.component.ts

```typescript
import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet],
  template: '<router-outlet></router-outlet>',
  styles: []
})
export class AppComponent {}
```

### client/src/app/components/layout/layout.component.ts

```typescript
import { Component } from '@angular/core';
import { RouterOutlet, RouterLink } from '@angular/router';

@Component({
  selector: 'app-layout',
  standalone: true,
  imports: [RouterOutlet, RouterLink],
  templateUrl: './layout.component.html',
  styleUrl: './layout.component.css'
})
export class LayoutComponent {}
```

### client/src/app/components/layout/layout.component.html

```html
<div class="app">
  <header class="header">
    <nav>
      <a routerLink="/">Home</a>
      <a routerLink="/about">About</a>
    </nav>
  </header>
  <main class="main">
    <router-outlet></router-outlet>
  </main>
  <footer class="footer">
    <p>Built with Mizu + Angular</p>
  </footer>
</div>
```

### client/src/app/components/layout/layout.component.css

```css
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

.header nav a {
  color: var(--text);
  text-decoration: none;
  font-weight: 500;
}

.header nav a:hover {
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
```

### client/src/app/pages/home/home.component.ts

```typescript
import { Component, OnInit, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [],
  templateUrl: './home.component.html',
  styleUrl: './home.component.css'
})
export class HomeComponent implements OnInit {
  message = signal('');
  loading = signal(true);

  constructor(private http: HttpClient) {}

  ngOnInit() {
    this.http.get<{ message: string }>('/api/hello').subscribe({
      next: (data) => {
        this.message.set(data.message);
        this.loading.set(false);
      },
      error: () => {
        this.message.set('Failed to load message');
        this.loading.set(false);
      }
    });
  }
}
```

### client/src/app/pages/home/home.component.html

```html
<div class="page home">
  <h1>Welcome</h1>
  @if (loading()) {
    <p>Loading...</p>
  } @else {
    <p class="api-message">{{ message() }}</p>
  }
</div>
```

### client/src/app/pages/home/home.component.css

```css
.page h1 {
  margin-bottom: 1rem;
}

.api-message {
  background: white;
  padding: 1rem;
  border-radius: 0.5rem;
  border: 1px solid var(--border);
}
```

### client/src/app/pages/about/about.component.ts

```typescript
import { Component } from '@angular/core';

@Component({
  selector: 'app-about',
  standalone: true,
  imports: [],
  templateUrl: './about.component.html',
  styleUrl: './about.component.css'
})
export class AboutComponent {}
```

### client/src/app/pages/about/about.component.html

```html
<div class="page about">
  <h1>About</h1>
  <p>
    This is an Angular SPA powered by Mizu, a lightweight Go web framework.
  </p>
  <ul>
    <li>Angular 19 with TypeScript</li>
    <li>Standalone components with signals</li>
    <li>Angular Router for client-side routing</li>
    <li>Mizu backend with the frontend middleware</li>
  </ul>
</div>
```

### client/src/app/pages/about/about.component.css

```css
.page h1 {
  margin-bottom: 1rem;
}

.about ul {
  margin-top: 1rem;
  padding-left: 1.5rem;
}

.about li {
  margin-bottom: 0.5rem;
}
```

### client/src/styles.css

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
```

### client/public/favicon.ico

Standard Angular favicon (binary file, placeholder for build).

## Makefile

### Makefile.tmpl

```makefile
.PHONY: dev build run clean install

# Development mode: run Angular and Go server concurrently
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
	rm -rf client/.angular

# Install dependencies
install:
	cd client && npm install
	go mod tidy
```

## CLI Usage

```bash
# Create a new Angular SPA project
mizu new ./myapp --template frontend/angular

# List all templates (shows Angular alongside React, Vue, and Svelte)
mizu new --list

# Preview the generated files
mizu new ./myapp --template frontend/angular --dry-run
```

## Testing Strategy

### Unit Tests

Add tests to `templates_test.go`:

```go
func TestListTemplatesIncludesAngular(t *testing.T) {
    templates, err := listTemplates()
    if err != nil {
        t.Fatalf("listTemplates() error: %v", err)
    }

    found := false
    for _, tmpl := range templates {
        if tmpl.Name == "frontend/angular" {
            found = true
            break
        }
    }

    if !found {
        t.Error("listTemplates() did not include nested template 'frontend/angular'")
    }
}

func TestTemplateExistsAngular(t *testing.T) {
    if !templateExists("frontend/angular") {
        t.Error("templateExists('frontend/angular') returned false")
    }
}

func TestLoadTemplateFilesAngular(t *testing.T) {
    files, err := loadTemplateFiles("frontend/angular")
    if err != nil {
        t.Fatalf("loadTemplateFiles() error: %v", err)
    }

    expectedFiles := []string{
        ".gitignore",
        "go.mod",
        "cmd/server/main.go",
        "app/server/app.go",
        "client/package.json",
        "client/angular.json",
        "client/src/main.ts",
        "client/src/app/app.component.ts",
        "client/src/app/app.routes.ts",
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

func TestBuildPlanAngularTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/angular", tmpDir, vars)
    if err != nil {
        t.Fatalf("buildPlan() error: %v", err)
    }

    if len(p.files) == 0 {
        t.Error("buildPlan() produced empty plan")
    }
}

func TestApplyPlanAngularTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/angular", tmpDir, vars)
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
        "client/angular.json",
        "client/src/main.ts",
        "client/src/app/app.component.ts",
        "Makefile",
    }

    for _, file := range expectedFiles {
        path := filepath.Join(tmpDir, file)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            t.Errorf("expected file %q does not exist", file)
        }
    }
}

func TestAngularTemplateVariableSubstitution(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myangular", "example.com/myangular", "MIT", nil)

    p, err := buildPlan("frontend/angular", tmpDir, vars)
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

    if !strings.Contains(string(content), "module example.com/myangular") {
        t.Error("go.mod does not contain expected module path")
    }

    // Check package.json has correct name
    pkgPath := filepath.Join(tmpDir, "client/package.json")
    pkgContent, err := os.ReadFile(pkgPath)
    if err != nil {
        t.Fatalf("failed to read package.json: %v", err)
    }

    if !strings.Contains(string(pkgContent), `"name": "myangular-client"`) {
        t.Error("package.json does not contain expected project name")
    }
}

func TestAngularTemplateHasAngularSpecificContent(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myangular", "example.com/myangular", "MIT", nil)

    p, err := buildPlan("frontend/angular", tmpDir, vars)
    if err != nil {
        t.Fatalf("buildPlan() error: %v", err)
    }

    if err := p.apply(false); err != nil {
        t.Fatalf("apply() error: %v", err)
    }

    // Verify app.component.ts contains Angular-specific syntax
    appPath := filepath.Join(tmpDir, "client/src/app/app.component.ts")
    content, err := os.ReadFile(appPath)
    if err != nil {
        t.Fatalf("failed to read app.component.ts: %v", err)
    }

    angularPatterns := []string{
        "@Component",
        "standalone: true",
        "@angular/core",
    }

    for _, pattern := range angularPatterns {
        if !strings.Contains(string(content), pattern) {
            t.Errorf("app.component.ts missing expected Angular pattern: %s", pattern)
        }
    }

    // Verify angular.json exists with proper content
    configPath := filepath.Join(tmpDir, "client/angular.json")
    configContent, err := os.ReadFile(configPath)
    if err != nil {
        t.Fatalf("failed to read angular.json: %v", err)
    }

    if !strings.Contains(string(configContent), "@angular-devkit/build-angular") {
        t.Error("angular.json missing @angular-devkit/build-angular configuration")
    }
}
```

## Key Differences from React/Vue/Svelte Templates

| Aspect | React | Vue | Svelte | Angular |
|--------|-------|-----|--------|---------|
| Entry Point | `main.tsx` | `main.ts` | `main.ts` | `main.ts` |
| Root Component | `App.tsx` (JSX) | `App.vue` (SFC) | `App.svelte` | `app.component.ts` (TypeScript) |
| Router | react-router-dom | vue-router | svelte-routing | @angular/router |
| Build Tool | Vite | Vite | Vite | Angular CLI (esbuild) |
| Dev Server Port | 5173 | 5173 | 5173 | 4200 |
| Build Command | `vite build` | `vite build` | `vite build` | `ng build` |
| Type Checking | TypeScript | vue-tsc | svelte-check | Angular CLI |
| Component Format | JSX/TSX | Single File Components | Svelte Components | Decorators + Templates |
| State Management | useState/useEffect | ref/onMounted | $state rune/onMount | Signals |
| Config File | vite.config.ts | vite.config.ts | vite.config.ts, svelte.config.js | angular.json |
| Proxy Config | vite.config.ts | vite.config.ts | vite.config.ts | proxy.conf.json |

## Implementation Notes

1. **Angular 19**: Uses the latest Angular with standalone components by default (no NgModules required)
2. **Signals**: Uses Angular's new signals API for reactive state management
3. **Modern Control Flow**: Uses the new `@if`, `@for` syntax instead of `*ngIf`, `*ngFor` directives
4. **Angular CLI**: Uses the Angular CLI with esbuild for fast builds (not Vite)
5. **Consistent Styling**: Shares the same CSS variables as React/Vue/Svelte templates for visual consistency
6. **Same Backend**: Go backend is identical to React/Vue/Svelte templates (framework-agnostic)
7. **HttpClient**: Uses Angular's HttpClient for API calls with RxJS observables
8. **Standalone Components**: All components are standalone for simpler architecture

## Why Angular CLI Instead of Vite

For an Angular SPA template, the Angular CLI with esbuild is preferred over Vite because:
- Native support for all Angular features (decorators, templates, styles)
- Built-in proxy configuration via `proxy.conf.json`
- Seamless integration with Angular's build system and optimizations
- Official tooling with long-term support
- Angular 19 uses esbuild by default which is extremely fast
- Better compatibility with Angular-specific features like SSR and SSG

## Development Workflow

1. **Start Development**:
   ```bash
   make dev
   ```
   This runs Angular dev server on port 4200 and Go server on port 3000. Angular proxies `/api` requests to Go.

2. **Production Build**:
   ```bash
   make build
   ```
   Builds Angular app to `dist/` which is embedded in the Go binary.

3. **Run Production**:
   ```bash
   make run
   ```
   Builds frontend and starts Go server serving the embedded frontend.

## Frontend Middleware Integration

The Angular template integrates with the Mizu frontend middleware using:
- **Dev Mode**: Proxies to Angular dev server on `http://localhost:4200`
- **Production Mode**: Serves embedded `dist/` files with SPA fallback
- **API Bypass**: `/api` paths are handled by Go routes, not the SPA fallback
