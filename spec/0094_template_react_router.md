# React Router v7 Frontend Template Design Spec

## Overview

This specification describes the implementation of a React Router v7 template for the Mizu frontend template system. React Router v7 represents the evolution of both React Router and Remix, providing a modern, type-safe, full-featured routing framework with first-class support for data loading, actions, and static export.

## Goals

1. **Modern React Router v7**: Provide a production-ready React Router v7 setup with latest features
2. **Type-Safe Routing**: Leverage React Router v7's type-safe routing capabilities
3. **File-Based Routing**: Use React Router v7's intuitive file-based routing system
4. **Data Loading Patterns**: Implement loader/action patterns for optimal data fetching
5. **Static Export**: Support static export mode for deployment with Mizu backend
6. **Best DX**: Provide the best developer experience with hot reload, TypeScript, and modern tooling
7. **Production Ready**: Include optimizations, error boundaries, and production best practices
8. **Mizu Integration**: Seamless integration with Mizu backend via the frontend middleware

## Why React Router v7?

React Router v7 (launched December 2024) is the merger of Remix and React Router, bringing:

- **Type-safe routing**: End-to-end type safety from routes to loaders
- **File-based routing**: Convention over configuration with routes/ directory
- **Data loading**: Built-in loader pattern for data fetching
- **Actions**: Form handling with type-safe actions
- **Error boundaries**: Per-route error handling
- **Pending states**: Built-in loading UI support
- **Static export**: Can be exported as static HTML for Mizu serving
- **Vite integration**: Fast builds and HMR via Vite
- **Modern patterns**: Web Fetch API, Web Forms, progressive enhancement

### React Router v7 vs Regular React

| Feature | React Router v7 | React + React Router v6 |
|---------|----------------|------------------------|
| **Routing** | File-based | Manual routes |
| **Data Loading** | Built-in loaders | Manual useEffect |
| **Forms** | Type-safe actions | Manual submit |
| **Type Safety** | Routes + data | Manual types |
| **Error Handling** | Per-route boundaries | Manual boundaries |
| **Bundle** | ~50kB | ~45kB |
| **Learning Curve** | ⚠️ Moderate | ⚡ Easy |
| **Best for** | Data-heavy apps | Simple SPAs |

## Template Structure

```
my-react-router-app/
├── cmd/
│   └── server/
│       └── main.go                    # Go entry point
├── app/
│   └── server/
│       ├── app.go                     # Mizu app configuration
│       ├── config.go                  # Server configuration
│       └── routes.go                  # API routes (Go backend)
├── client/                            # React Router v7 application
│   ├── app/
│   │   ├── root.tsx                   # Root layout
│   │   ├── routes.ts                  # Route definitions
│   │   └── routes/                    # Route modules
│   │       ├── _index.tsx             # Home page (/)
│   │       ├── about.tsx              # About page (/about)
│   │       └── users/
│   │           ├── _layout.tsx        # Users layout
│   │           ├── _index.tsx         # Users list (/users)
│   │           └── $id.tsx            # User detail (/users/:id)
│   ├── public/
│   │   └── favicon.ico                # Public assets
│   ├── package.json                   # npm dependencies
│   ├── vite.config.ts                 # Vite configuration
│   ├── tsconfig.json                  # TypeScript config
│   └── react-router.config.ts         # React Router config
├── dist/                              # Built files (after build)
│   └── placeholder.txt
├── go.mod
├── go.sum
└── Makefile
```

## React Router v7 Configuration

### File-Based Routing Structure

React Router v7 uses file-based routing similar to Next.js but with more flexibility:

- `routes/_index.tsx` → `/`
- `routes/about.tsx` → `/about`
- `routes/users/_index.tsx` → `/users`
- `routes/users/$id.tsx` → `/users/:id`
- `routes/users/_layout.tsx` → Layout for all `/users/*` routes
- `routes/admin._index.tsx` → `/admin` (pathless layout with .)

### Route Module Structure

Each route module can export:
- `loader`: Data loading function
- `action`: Form submission handler
- `default`: Component to render
- `ErrorBoundary`: Error handling component
- `meta`: Page metadata

## Template Metadata

### template.json

```json
{
  "name": "frontend/reactrouter",
  "description": "React Router v7 framework with Vite, TypeScript, and Mizu backend",
  "tags": ["go", "mizu", "frontend", "spa", "react", "react-router", "react-router-v7", "vite", "typescript"],
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
	cfg := &Config{
		Port:    getEnv("PORT", "3000"),
		DevPort: getEnv("DEV_PORT", "5173"),
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

func New(cfg *Config) *mizu.App {
	app := mizu.New()

	// API routes come first (before frontend middleware)
	setupRoutes(app)

	// Frontend middleware handles all non-API routes
	dist, _ := fs.Sub(distFS, "dist")
	app.Use(frontend.WithOptions(frontend.Options{
		Mode:        frontend.ModeAuto,       // Auto-detect dev/prod
		FS:          dist,                     // Embedded build files
		Root:        "./dist",                 // Fallback to filesystem in dev
		DevServer:   "http://localhost:" + cfg.DevPort,  // Vite dev server
		IgnorePaths: []string{"/api"},        // Don't proxy /api to Vite
	}))

	return app
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

	// Health check
	api.Get("/health", handleHealth)

	// User endpoints
	api.Get("/users", handleUsers)
	api.Post("/users", createUser)
	api.Get("/users/{id}", getUser)
	api.Put("/users/{id}", updateUser)
	api.Delete("/users/{id}", deleteUser)
}

func handleHealth(c *mizu.Ctx) error {
	return c.JSON(200, map[string]any{
		"status": "ok",
		"service": "{{.Name}}",
	})
}

func handleUsers(c *mizu.Ctx) error {
	users := []map[string]any{
		{"id": 1, "name": "Alice Johnson", "email": "alice@example.com", "role": "admin"},
		{"id": 2, "name": "Bob Smith", "email": "bob@example.com", "role": "user"},
		{"id": 3, "name": "Charlie Brown", "email": "charlie@example.com", "role": "user"},
	}
	return c.JSON(200, users)
}

func createUser(c *mizu.Ctx) error {
	var user map[string]any
	if err := c.BodyJSON(&user); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid JSON"})
	}

	if user["name"] == nil || user["email"] == nil {
		return c.JSON(400, map[string]string{"error": "Name and email required"})
	}

	user["id"] = 4
	user["role"] = "user"

	return c.JSON(201, user)
}

func getUser(c *mizu.Ctx) error {
	id := c.Param("id")
	user := map[string]any{
		"id":    id,
		"name":  "User " + id,
		"email": "user" + id + "@example.com",
		"role":  "user",
	}
	return c.JSON(200, user)
}

func updateUser(c *mizu.Ctx) error {
	id := c.Param("id")
	var updates map[string]any
	if err := c.BodyJSON(&updates); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid JSON"})
	}

	user := map[string]any{
		"id":    id,
		"name":  updates["name"],
		"email": updates["email"],
		"role":  updates["role"],
	}
	return c.JSON(200, user)
}

func deleteUser(c *mizu.Ctx) error {
	return c.JSON(204, nil)
}
```

## React Router v7 Frontend Files

### client/package.json

```json
{
  "name": "{{.Name}}-client",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "react-router dev",
    "build": "react-router build",
    "start": "react-router-serve ./build/server/index.js",
    "typecheck": "react-router typegen && tsc"
  },
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "react-router": "^7.1.1"
  },
  "devDependencies": {
    "@react-router/dev": "^7.1.1",
    "@react-router/node": "^7.1.1",
    "@react-router/serve": "^7.1.1",
    "@types/react": "^19.0.1",
    "@types/react-dom": "^19.0.2",
    "typescript": "^5.7.2",
    "vite": "^6.0.3"
  }
}
```

### client/react-router.config.ts

```typescript
import type { Config } from "@react-router/dev/config";

export default {
  // Configure for static export mode
  ssr: false,

  // Build output directory (relative to client/)
  buildDirectory: "../dist",

  // Server build configuration
  serverBuildFile: "index.js",

  // Vite configuration
  vite: {
    server: {
      port: 5173,
      strictPort: true,
    },
  },
} satisfies Config;
```

### client/vite.config.ts

```typescript
import { reactRouter } from "@react-router/dev/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [reactRouter()],
});
```

### client/tsconfig.json

```json
{
  "include": [
    "**/*",
    "**/.server/**/*",
    "**/.client/**/*"
  ],
  "compilerOptions": {
    "lib": ["DOM", "DOM.Iterable", "ES2022"],
    "types": ["@react-router/node", "vite/client"],
    "isolatedModules": true,
    "esModuleInterop": true,
    "jsx": "react-jsx",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "resolveJsonModule": true,
    "target": "ES2022",
    "strict": true,
    "allowJs": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "baseUrl": ".",
    "paths": {
      "~/*": ["./app/*"]
    },
    "noEmit": true
  }
}
```

### client/app/root.tsx

```typescript
import {
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
} from "react-router";

import type { Route } from "./+types/root";
import stylesheet from "./styles/app.css?url";

export const links: Route.LinksFunction = () => [
  { rel: "stylesheet", href: stylesheet },
];

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <Meta />
        <Links />
      </head>
      <body>
        {children}
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function Root() {
  return <Outlet />;
}
```

### client/app/routes.ts

```typescript
import {
  type RouteConfig,
  route,
  layout,
  index,
} from "@react-router/dev/routes";

export default [
  layout("routes/_layout.tsx", [
    index("routes/_index.tsx"),
    route("about", "routes/about.tsx"),
    route("users", "routes/users/_layout.tsx", [
      index("routes/users/_index.tsx"),
      route(":id", "routes/users/$id.tsx"),
    ]),
  ]),
] satisfies RouteConfig;
```

### client/app/routes/_layout.tsx

```typescript
import { Link, Outlet } from "react-router";

export default function Layout() {
  return (
    <div className="app">
      <header className="header">
        <div className="header-content">
          <h1 className="logo">{{.Name}}</h1>
          <nav className="nav">
            <Link to="/" className="nav-link">Home</Link>
            <Link to="/about" className="nav-link">About</Link>
            <Link to="/users" className="nav-link">Users</Link>
          </nav>
        </div>
      </header>

      <main className="main">
        <Outlet />
      </main>

      <footer className="footer">
        <p>Built with Mizu + React Router v7</p>
      </footer>
    </div>
  );
}
```

### client/app/routes/_index.tsx

```typescript
import { useLoaderData } from "react-router";
import type { Route } from "./+types/_index";

export async function loader() {
  const res = await fetch("/api/health");
  const data = await res.json();
  return { health: data };
}

export function meta({}: Route.MetaArgs) {
  return [
    { title: "{{.Name}}" },
    { name: "description", content: "Welcome to React Router v7 with Mizu!" },
  ];
}

export default function Index({ loaderData }: Route.ComponentProps) {
  return (
    <div className="home">
      <h1>Welcome to React Router v7</h1>
      <p className="subtitle">
        Modern routing framework powered by Mizu backend
      </p>

      <div className="features">
        <div className="feature">
          <h3>🚀 React Router v7</h3>
          <p>Type-safe routing with file-based conventions</p>
        </div>
        <div className="feature">
          <h3>⚡ Vite</h3>
          <p>Lightning fast builds and hot module replacement</p>
        </div>
        <div className="feature">
          <h3>🔷 TypeScript</h3>
          <p>Full type safety from routes to data</p>
        </div>
        <div className="feature">
          <h3>💧 Mizu</h3>
          <p>Lightweight Go backend with powerful middleware</p>
        </div>
      </div>

      {loaderData?.health && (
        <div className="status">
          <span className="status-indicator"></span>
          <span>Backend Status: {loaderData.health.status}</span>
        </div>
      )}
    </div>
  );
}
```

### client/app/routes/about.tsx

```typescript
import type { Route } from "./+types/about";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "About - {{.Name}}" },
    { name: "description", content: "Learn more about our stack" },
  ];
}

export default function About() {
  return (
    <div className="page">
      <h1>About This App</h1>
      <p className="lead">
        This application demonstrates the power of React Router v7 combined with a Mizu backend.
      </p>

      <section className="section">
        <h2>Technology Stack</h2>
        <ul className="tech-list">
          <li><strong>React Router v7</strong> - Modern React framework with type-safe routing</li>
          <li><strong>React 19</strong> - Latest React with improved performance</li>
          <li><strong>Vite</strong> - Next generation frontend tooling</li>
          <li><strong>TypeScript</strong> - Type-safe development</li>
          <li><strong>Mizu</strong> - Lightweight Go web framework</li>
        </ul>
      </section>

      <section className="section">
        <h2>Key Features</h2>
        <ul className="feature-list">
          <li>File-based routing with type generation</li>
          <li>Built-in data loading with loaders</li>
          <li>Type-safe form actions</li>
          <li>Error boundaries per route</li>
          <li>Optimistic UI updates</li>
          <li>Static export for production</li>
          <li>Single binary deployment</li>
        </ul>
      </section>
    </div>
  );
}
```

### client/app/routes/users/_layout.tsx

```typescript
import { Outlet } from "react-router";

export default function UsersLayout() {
  return (
    <div className="users-layout">
      <div className="users-header">
        <h1>Users</h1>
      </div>
      <Outlet />
    </div>
  );
}
```

### client/app/routes/users/_index.tsx

```typescript
import { Link, useLoaderData } from "react-router";
import type { Route } from "./+types/_index";

interface User {
  id: number;
  name: string;
  email: string;
  role: string;
}

export async function loader() {
  const res = await fetch("/api/users");
  const users: User[] = await res.json();
  return { users };
}

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Users - {{.Name}}" },
    { name: "description", content: "View all users" },
  ];
}

export default function Users({ loaderData }: Route.ComponentProps) {
  const { users } = loaderData;

  return (
    <div className="users-list">
      <div className="users-grid">
        {users.map((user) => (
          <Link
            key={user.id}
            to={`/users/${user.id}`}
            className="user-card"
          >
            <div className="user-avatar">
              {user.name.charAt(0).toUpperCase()}
            </div>
            <div className="user-info">
              <h3>{user.name}</h3>
              <p>{user.email}</p>
              <span className={`badge badge-${user.role}`}>
                {user.role}
              </span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
```

### client/app/routes/users/$id.tsx

```typescript
import { useLoaderData, useNavigate } from "react-router";
import type { Route } from "./+types/$id";

interface User {
  id: string;
  name: string;
  email: string;
  role: string;
}

export async function loader({ params }: Route.LoaderArgs) {
  const res = await fetch(`/api/users/${params.id}`);

  if (!res.ok) {
    throw new Response("User not found", { status: 404 });
  }

  const user: User = await res.json();
  return { user };
}

export function meta({ data }: Route.MetaArgs) {
  return [
    { title: `${data?.user.name || "User"} - {{.Name}}` },
    { name: "description", content: `View ${data?.user.name}'s profile` },
  ];
}

export default function UserDetail({ loaderData }: Route.ComponentProps) {
  const { user } = loaderData;
  const navigate = useNavigate();

  return (
    <div className="user-detail">
      <button onClick={() => navigate(-1)} className="back-button">
        ← Back to Users
      </button>

      <div className="user-profile">
        <div className="user-avatar-large">
          {user.name.charAt(0).toUpperCase()}
        </div>

        <div className="user-details">
          <h2>{user.name}</h2>

          <div className="detail-row">
            <span className="label">Email:</span>
            <span className="value">{user.email}</span>
          </div>

          <div className="detail-row">
            <span className="label">Role:</span>
            <span className={`badge badge-${user.role}`}>
              {user.role}
            </span>
          </div>

          <div className="detail-row">
            <span className="label">ID:</span>
            <span className="value">{user.id}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  return (
    <div className="error-page">
      <h1>Oops!</h1>
      <p>
        {error instanceof Error
          ? error.message
          : "Something went wrong loading this user."}
      </p>
      <a href="/users" className="button">
        Back to Users
      </a>
    </div>
  );
}
```

### client/app/styles/app.css

```css
:root {
  --primary: #3b82f6;
  --primary-dark: #2563eb;
  --success: #10b981;
  --warning: #f59e0b;
  --danger: #ef4444;
  --bg: #f8fafc;
  --bg-card: #ffffff;
  --text: #1e293b;
  --text-muted: #64748b;
  --border: #e2e8f0;
  --shadow: 0 1px 3px 0 rgb(0 0 0 / 0.1);
  --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1);
  --radius: 0.5rem;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  background: var(--bg);
  color: var(--text);
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
}

.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

/* Header */
.header {
  background: var(--bg-card);
  border-bottom: 1px solid var(--border);
  box-shadow: var(--shadow);
}

.header-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1rem 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.logo {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--primary);
}

.nav {
  display: flex;
  gap: 2rem;
}

.nav-link {
  color: var(--text);
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s;
}

.nav-link:hover {
  color: var(--primary);
}

/* Main */
.main {
  flex: 1;
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
  width: 100%;
}

/* Footer */
.footer {
  background: var(--bg-card);
  border-top: 1px solid var(--border);
  padding: 1.5rem 2rem;
  text-align: center;
  color: var(--text-muted);
  font-size: 0.875rem;
}

/* Home Page */
.home {
  text-align: center;
  padding: 2rem 0;
}

.home h1 {
  font-size: 3rem;
  margin-bottom: 0.5rem;
  background: linear-gradient(135deg, var(--primary) 0%, var(--primary-dark) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.subtitle {
  font-size: 1.25rem;
  color: var(--text-muted);
  margin-bottom: 3rem;
}

.features {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 2rem;
  margin: 3rem 0;
}

.feature {
  background: var(--bg-card);
  padding: 2rem;
  border-radius: var(--radius);
  box-shadow: var(--shadow);
  border: 1px solid var(--border);
  transition: transform 0.2s, box-shadow 0.2s;
}

.feature:hover {
  transform: translateY(-4px);
  box-shadow: var(--shadow-lg);
}

.feature h3 {
  font-size: 1.25rem;
  margin-bottom: 0.5rem;
}

.feature p {
  color: var(--text-muted);
}

.status {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  background: var(--bg-card);
  padding: 0.75rem 1.5rem;
  border-radius: 2rem;
  border: 1px solid var(--border);
  margin-top: 2rem;
}

.status-indicator {
  width: 8px;
  height: 8px;
  background: var(--success);
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* Pages */
.page {
  max-width: 800px;
  margin: 0 auto;
}

.page h1 {
  font-size: 2.5rem;
  margin-bottom: 1rem;
}

.lead {
  font-size: 1.25rem;
  color: var(--text-muted);
  margin-bottom: 2rem;
}

.section {
  margin: 2rem 0;
}

.section h2 {
  font-size: 1.5rem;
  margin-bottom: 1rem;
  color: var(--primary);
}

.tech-list, .feature-list {
  list-style: none;
  padding-left: 0;
}

.tech-list li, .feature-list li {
  padding: 0.5rem 0;
  padding-left: 1.5rem;
  position: relative;
}

.tech-list li::before, .feature-list li::before {
  content: "→";
  position: absolute;
  left: 0;
  color: var(--primary);
  font-weight: bold;
}

/* Users */
.users-layout {
  width: 100%;
}

.users-header {
  margin-bottom: 2rem;
}

.users-header h1 {
  font-size: 2rem;
}

.users-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
}

.user-card {
  display: flex;
  gap: 1rem;
  padding: 1.5rem;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  text-decoration: none;
  color: var(--text);
  transition: transform 0.2s, box-shadow 0.2s;
}

.user-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
  border-color: var(--primary);
}

.user-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--primary);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.25rem;
  font-weight: 600;
  flex-shrink: 0;
}

.user-info {
  flex: 1;
}

.user-info h3 {
  font-size: 1.125rem;
  margin-bottom: 0.25rem;
}

.user-info p {
  color: var(--text-muted);
  font-size: 0.875rem;
  margin-bottom: 0.5rem;
}

.badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 1rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.badge-admin {
  background: #fef3c7;
  color: #92400e;
}

.badge-user {
  background: #dbeafe;
  color: #1e40af;
}

/* User Detail */
.user-detail {
  max-width: 600px;
  margin: 0 auto;
}

.back-button {
  background: none;
  border: none;
  color: var(--primary);
  font-size: 1rem;
  cursor: pointer;
  padding: 0.5rem 0;
  margin-bottom: 2rem;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  transition: color 0.2s;
}

.back-button:hover {
  color: var(--primary-dark);
}

.user-profile {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 2rem;
  box-shadow: var(--shadow);
}

.user-avatar-large {
  width: 96px;
  height: 96px;
  border-radius: 50%;
  background: var(--primary);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 2.5rem;
  font-weight: 600;
  margin: 0 auto 2rem;
}

.user-details h2 {
  text-align: center;
  margin-bottom: 2rem;
  font-size: 1.75rem;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 0;
  border-bottom: 1px solid var(--border);
}

.detail-row:last-child {
  border-bottom: none;
}

.label {
  font-weight: 600;
  color: var(--text-muted);
}

.value {
  color: var(--text);
}

/* Error Page */
.error-page {
  text-align: center;
  padding: 4rem 2rem;
}

.error-page h1 {
  font-size: 3rem;
  margin-bottom: 1rem;
  color: var(--danger);
}

.error-page p {
  color: var(--text-muted);
  margin-bottom: 2rem;
}

.button {
  display: inline-block;
  padding: 0.75rem 2rem;
  background: var(--primary);
  color: white;
  text-decoration: none;
  border-radius: var(--radius);
  font-weight: 600;
  transition: background 0.2s;
}

.button:hover {
  background: var(--primary-dark);
}

/* Responsive */
@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    gap: 1rem;
  }

  .home h1 {
    font-size: 2rem;
  }

  .features {
    grid-template-columns: 1fr;
  }

  .users-grid {
    grid-template-columns: 1fr;
  }
}
```

### Makefile.tmpl

```makefile
.PHONY: dev build run clean install test

# Development mode: run React Router dev server and Go server
dev:
	@echo "Starting development servers..."
	@(cd client && npm run dev) & \
	MIZU_ENV=development go run ./cmd/server

# Build frontend for production (static export)
build:
	@echo "Building React Router app..."
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
	rm -rf client/build
	rm -rf client/.react-router

# Install dependencies
install:
	cd client && npm install
	go mod tidy

# Type check
typecheck:
	cd client && npm run typecheck

# Run tests
test:
	go test ./...
```

## CLI Usage

```bash
# Create a new React Router v7 project
mizu new ./myapp --template frontend/reactrouter

# Navigate and install dependencies
cd myapp
make install

# Start development mode
make dev

# Build for production
make build

# Run production server
make run
```

## Development Workflow

### Development Mode

When running `make dev`:

1. Vite dev server starts on port 5173
2. Go server starts on port 3000
3. Frontend middleware proxies to Vite in development
4. Hot Module Replacement (HMR) works seamlessly
5. API requests to `/api/*` go to Go backend
6. All other requests go to Vite dev server

### Production Build

When running `make build`:

1. React Router v7 builds the app in static export mode
2. HTML, CSS, JS files generated to `dist/`
3. Assets are optimized and fingerprinted
4. Go binary embeds the `dist/` directory
5. Single binary contains both frontend and backend

## Type Safety Features

### Route Types

React Router v7 automatically generates types for routes:

```typescript
import type { Route } from "./+types/users/$id";

// Loader args are fully typed
export async function loader({ params }: Route.LoaderArgs) {
  // params.id is typed as string
  const user = await fetchUser(params.id);
  return { user };
}

// Component props are fully typed
export default function UserDetail({ loaderData }: Route.ComponentProps) {
  // loaderData.user is fully typed
  return <div>{loaderData.user.name}</div>;
}
```

### Meta Function Types

```typescript
export function meta({ data, params }: Route.MetaArgs) {
  return [
    { title: `${data.user.name} - Profile` },
    { name: "description", content: `View ${data.user.name}'s profile` },
  ];
}
```

## Error Handling

React Router v7 provides per-route error boundaries:

```typescript
export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  return (
    <div>
      <h1>Error!</h1>
      <p>{error.message}</p>
    </div>
  );
}
```

## Static Export Configuration

React Router v7 supports static export via the `ssr: false` option in `react-router.config.ts`. This generates:

- Static HTML files for each route
- Optimized JavaScript bundles
- CSS files with fingerprinted names
- Asset manifests for loading

The static files are then served by Mizu's frontend middleware, which handles:
- SPA fallback routing (index.html for client-side routes)
- Asset serving with proper caching headers
- Development mode proxying to Vite

## Comparison with Other Templates

| Feature | reactrouter | react | next |
|---------|------------|-------|------|
| **Framework** | React Router v7 | React + RR v6 | Next.js |
| **Routing** | File-based | Manual config | File-based (pages/) |
| **Data Loading** | Built-in loaders | Manual fetch | getStaticProps |
| **Type Safety** | Auto-generated | Manual | Manual |
| **Bundle Size** | ~50kB | ~45kB | ~90kB |
| **Learning Curve** | ⚠️ Moderate | ⚡ Easy | ⚠️ Moderate |
| **Best For** | Data-heavy SPAs | Simple apps | Full-stack SSR |

## Best Practices Included

1. **Type Safety**: Full TypeScript support with generated route types
2. **Error Boundaries**: Per-route error handling
3. **Meta Tags**: SEO-friendly meta tag generation
4. **Loading States**: Built-in pending UI support
5. **Code Splitting**: Automatic route-based code splitting
6. **CSS Architecture**: Modern CSS with CSS variables
7. **Responsive Design**: Mobile-first responsive layouts
8. **Accessibility**: Semantic HTML and ARIA attributes
9. **Performance**: Optimized builds with Vite
10. **Developer Experience**: Fast HMR, TypeScript, ESLint

## Future Enhancements

Potential additions to the template:

1. **Form Actions**: Add example forms with type-safe actions
2. **Optimistic UI**: Show optimistic updates examples
3. **Deferred Data**: Demonstrate streaming data loading
4. **Prefetching**: Add link prefetching for faster navigation
5. **Authentication**: Add auth context and protected routes
6. **State Management**: Optional Zustand or Jotai integration
7. **Testing**: Add Vitest + Testing Library setup
8. **Styling Options**: Add Tailwind CSS variant

## Documentation Requirements

The template will need comprehensive documentation covering:

1. React Router v7 concepts (loaders, actions, routes)
2. File-based routing conventions
3. Type safety features
4. Data loading patterns
5. Form handling with actions
6. Error boundary usage
7. Meta tags and SEO
8. Static export configuration
9. Mizu integration details
10. Migration from React Router v6
11. Comparison with Next.js/Remix

## Testing Strategy

### Template Generation Test

```go
func TestReactRouterTemplateGeneration(t *testing.T) {
    tmpDir := t.TempDir()
    vars := newTemplateVars("myapp", "example.com/myapp", "MIT", nil)

    p, err := buildPlan("frontend/reactrouter", tmpDir, vars)
    require.NoError(t, err)

    err = p.apply(false)
    require.NoError(t, err)

    // Verify key files exist
    require.FileExists(t, filepath.Join(tmpDir, "client/app/root.tsx"))
    require.FileExists(t, filepath.Join(tmpDir, "client/app/routes.ts"))
    require.FileExists(t, filepath.Join(tmpDir, "client/react-router.config.ts"))
}
```

## Implementation Checklist

- [ ] Create directory structure `cmd/cli/templates/frontend/reactrouter/`
- [ ] Create `template.json` metadata file
- [ ] Create Go backend files (cmd/server, app/server)
- [ ] Create React Router v7 config files
- [ ] Create root.tsx and routes.ts
- [ ] Create route modules (_layout, _index, about, users/*)
- [ ] Create CSS styling
- [ ] Create Makefile
- [ ] Add documentation to docs/frontend/reactrouter.mdx
- [ ] Update docs/docs.json
- [ ] Test template generation
- [ ] Test development workflow
- [ ] Test production build
- [ ] Verify type generation works
- [ ] Verify HMR works correctly

## Success Criteria

The template is successful if:

1. ✅ `mizu new` generates a working React Router v7 app
2. ✅ `make dev` starts both servers with working HMR
3. ✅ `make build` produces optimized static files
4. ✅ Production build runs as single Go binary
5. ✅ Type generation works correctly
6. ✅ All routes render properly
7. ✅ API integration works in dev and prod
8. ✅ Error boundaries catch errors correctly
9. ✅ Documentation is clear and comprehensive
10. ✅ Developer experience is smooth and intuitive
