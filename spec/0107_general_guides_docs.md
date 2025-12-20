# Spec 0107: General Guides Documentation

## Overview

This spec outlines the plan for the new "Guides" tab in the Mizu documentation. The Guides tab serves as the primary entry point for all users, providing a high-level introduction to Mizu and its component ecosystem before directing users to detailed documentation.

## Purpose

The Guides tab will:
1. Welcome new users and explain what Mizu is
2. Provide installation and setup instructions
3. Introduce each major component (Core, Middlewares, Contract, View, Frontend, Mobile, CLI)
4. Offer tutorials that integrate multiple components
5. Explain the overall architecture
6. Answer frequently asked questions
7. Connect users to the community

## Tab Structure

```
Guides/
├── Introduction/
│   ├── welcome.mdx          - Landing page, what is Mizu
│   ├── what-is-mizu.mdx     - Detailed explanation
│   └── installation.mdx     - Getting started
├── Building Blocks/
│   ├── core.mdx             - Core framework introduction
│   ├── middlewares.mdx      - Middleware ecosystem
│   ├── contract.mdx         - Contract system
│   ├── view.mdx             - View engine & templates
│   ├── frontend.mdx         - Frontend integration
│   ├── mobile.mdx           - Mobile backend support
│   └── cli.mdx              - CLI tools
├── Tutorials/
│   ├── first-api.mdx        - Build your first API
│   ├── first-website.mdx    - Build a website with views
│   └── first-fullstack.mdx  - Fullstack app tutorial
└── Learn More/
    ├── architecture.mdx     - How Mizu is designed
    ├── faq.mdx              - Frequently asked questions
    └── community.mdx        - Discord, GitHub, contributing
```

## Page Specifications

---

### 1. guides/welcome.mdx

**Purpose:** Landing page that gives a quick overview and directs users.

**Content:**
- Hero section: "Welcome to Mizu"
- Tagline: "A lightweight, composable web framework for Go"
- 3 key benefits:
  1. Built on net/http (no magic, just Go)
  2. Composable middleware ecosystem
  3. Multi-transport contracts with SDK generation
- Quick navigation cards to:
  - Quick Start (get-started/quick-start)
  - Core Concepts (concepts/overview)
  - Building Blocks (this section's components)
- "What can you build?" section with links to use cases
- Version badge and Go version requirement

**Target Length:** 150-200 lines

---

### 2. guides/what-is-mizu.mdx

**Purpose:** Detailed explanation of what Mizu is and its philosophy.

**Content:**
- What is Mizu (expanded)
- The Mizu philosophy:
  - Enhance, don't replace the standard library
  - Explicit over magic
  - Composable over monolithic
  - Transport-neutral contracts
- What problems Mizu solves
- Comparison with other frameworks (feature matrix)
- When to use Mizu (and when not to)
- The name "Mizu" (Japanese for water - flows naturally)

**Target Length:** 300-350 lines

---

### 3. guides/installation.mdx

**Purpose:** Getting started with Mizu installation and setup.

**Content:**
- Requirements:
  - Go 1.22 or later
  - Operating system support
- Installation methods:
  1. As a library: `go get github.com/go-mizu/mizu`
  2. Using CLI: Installing mizu CLI
  3. Using templates: `mizu new`
- First program (minimal example)
- IDE setup:
  - VS Code with Go extension
  - GoLand configuration
  - Other editors
- Verifying installation
- Next steps links

**Target Length:** 250-300 lines

---

### 4. guides/core.mdx

**Purpose:** Introduction to the Core framework (the main Mizu package).

**Content:**
- What is the Core framework?
- Key concepts overview:
  - App: Server lifecycle management
  - Router: URL pattern matching
  - Handler: Request processing function
  - Context (Ctx): Request/response wrapper
  - Middleware: Request pipeline
- How they work together (diagram)
- Simple example tying all concepts together
- Link table to detailed documentation in Core tab:

| Concept | What it does | Learn more |
|---------|--------------|------------|
| App | Manages server lifecycle | [App →](/concepts/app) |
| Routing | Maps URLs to handlers | [Routing →](/concepts/routing) |
| Handler | Processes requests | [Handler →](/concepts/handler) |
| Context | Access request/response | [Context →](/concepts/context) |
| Request | Read incoming data | [Request →](/concepts/request) |
| Response | Send data back | [Response →](/concepts/response) |
| Static | Serve files | [Static →](/concepts/static) |
| Logging | Structured logging | [Logging →](/concepts/logging) |
| Error | Handle errors | [Error →](/concepts/error) |
| Middleware | Process pipeline | [Middleware →](/concepts/middleware) |

**Target Length:** 350-400 lines

---

### 5. guides/middlewares.mdx

**Purpose:** Introduction to the middleware ecosystem.

**Content:**
- What are middlewares?
- How middlewares work in Mizu
- Categories of available middlewares:
  - Authentication (Basic, Bearer, JWT, OAuth2, OIDC)
  - Security (CORS, CSRF, Helmet, Rate Limiting)
  - Request Processing (Validation, Body Limit, Timeout)
  - Response Processing (Compression, Caching, ETag)
  - Observability (Logger, Metrics, Tracing)
  - Real-time (WebSocket, SSE)
- How to use middlewares:
  - Global: `app.Use()`
  - Scoped: `app.With()`
  - Route groups
- Example: Building a secure API with multiple middlewares
- Link to detailed Middlewares tab

**Target Length:** 400-450 lines

---

### 6. guides/contract.mdx

**Purpose:** Introduction to the Contract system.

**Content:**
- What is a Contract?
  - Transport-neutral service definition
  - Define once, use anywhere
- Why use contracts?
  - Type safety across boundaries
  - Automatic client generation
  - Multiple transport support
- Supported transports:
  - REST (HTTP/JSON)
  - JSON-RPC
  - MCP (Model Context Protocol)
- SDK generation:
  - Go
  - Python
  - TypeScript
- Simple contract example with code
- How contracts compare to:
  - gRPC
  - OpenAPI
  - GraphQL
- Link to detailed Contract tab

**Target Length:** 400-450 lines

---

### 7. guides/view.mdx

**Purpose:** Introduction to the View engine and template system.

**Content:**
- What is the View engine?
- Template system features:
  - Layouts
  - Components
  - Partials
  - Template functions
- Live (Realtime) features:
  - Server-driven UI updates
  - PubSub integration
  - Session management
- Sync (State) features:
  - Real-time state synchronization
  - Reactive data binding
  - Collection management
- Example: Building an interactive dashboard
- Comparison with other approaches:
  - vs. React/Vue (frontend frameworks)
  - vs. HTMX
  - vs. LiveView (Phoenix)
- Link to detailed View tab

**Target Length:** 400-450 lines

---

### 8. guides/frontend.mdx

**Purpose:** Introduction to frontend integration.

**Content:**
- What is Frontend integration?
- Supported frameworks:
  - React (with React Router)
  - Vue
  - Svelte (with SvelteKit)
  - Angular
  - HTMX
  - Next.js
  - Nuxt
  - Preact
  - Alpine.js
- Development workflow:
  - Dev proxy for hot reload
  - Building for production
  - Embedding in Go binary
- Key features:
  - Configuration management
  - Environment injection
  - Manifest support
  - Service workers
- Example: React app with Mizu backend
- Link to detailed Frontend tab

**Target Length:** 350-400 lines

---

### 9. guides/mobile.mdx

**Purpose:** Introduction to mobile backend support.

**Content:**
- What is Mobile support?
- Backend features for mobile:
  - Device detection
  - API versioning
  - Pagination patterns
  - Push notifications
  - Deep linking
  - App store integration
- Supported platforms:
  - iOS (Swift)
  - Android (Kotlin)
  - Flutter
  - React Native
  - Kotlin Multiplatform
  - .NET MAUI
  - PWA
  - Game engines
- SDK generation for mobile
- Example: iOS app with Mizu backend
- Link to detailed Mobile tab

**Target Length:** 350-400 lines

---

### 10. guides/cli.mdx

**Purpose:** Introduction to the Mizu CLI.

**Content:**
- What is the Mizu CLI?
- Installation methods
- Key commands:
  - `mizu new` - Create new projects
  - `mizu dev` - Development server
  - `mizu contract` - Contract operations
  - `mizu version` - Version information
- Available templates:
  - Minimal - Basic server
  - API - REST API
  - Contract - Contract-based service
  - Web - Server-rendered website
  - Live - Real-time app
  - Sync - State synchronization
- Example: Creating and running a new project
- Link to detailed CLI tab

**Target Length:** 300-350 lines

---

### 11. guides/first-api.mdx

**Purpose:** Tutorial to build a complete REST API.

**Content:**
- What we'll build: A task management API
- Prerequisites
- Step-by-step guide:
  1. Create project
  2. Define routes
  3. Create handlers
  4. Add middleware (auth, validation)
  5. Error handling
  6. Testing
  7. Deployment preparation
- Complete code at each step
- Testing with curl
- What's next (adding frontend, database)

**Target Length:** 500-600 lines

---

### 12. guides/first-website.mdx

**Purpose:** Tutorial to build a server-rendered website.

**Content:**
- What we'll build: A blog website
- Prerequisites
- Step-by-step guide:
  1. Create project with web template
  2. Set up view engine
  3. Create layouts
  4. Add pages (home, blog list, blog post)
  5. Add forms
  6. Static files (CSS, images)
  7. Deploy
- Complete code at each step
- What's next (adding Live features)

**Target Length:** 500-600 lines

---

### 13. guides/first-fullstack.mdx

**Purpose:** Tutorial to build a fullstack application.

**Content:**
- What we'll build: A real-time dashboard
- Prerequisites
- Step-by-step guide:
  1. Backend API setup
  2. Frontend setup (React or Vue)
  3. Connect frontend to API
  4. Add authentication
  5. Add real-time updates (SSE or WebSocket)
  6. Embed frontend in Go binary
  7. Deploy as single binary
- Complete code at each step
- Production considerations
- What's next

**Target Length:** 600-700 lines

---

### 14. guides/architecture.mdx

**Purpose:** Explain Mizu's overall architecture.

**Content:**
- Design principles
- Architecture diagram
- Core components and their relationships
- Request lifecycle (detailed)
- Extension points:
  - Middlewares
  - Transports
  - View engines
  - Storage adapters
- Performance characteristics
- Comparison with other architectures
- Contributing to Mizu architecture

**Target Length:** 400-450 lines

---

### 15. guides/faq.mdx

**Purpose:** Answer common questions.

**Content:**
- General questions:
  - What Go version is required?
  - Can I use Mizu with existing net/http code?
  - How does Mizu compare to X?
- Performance questions:
  - Is Mizu fast?
  - Memory usage
  - Concurrency handling
- Feature questions:
  - WebSocket support?
  - Database integration?
  - Authentication options?
- Development questions:
  - Hot reload?
  - Testing?
  - Debugging?
- Deployment questions:
  - Docker?
  - Cloud platforms?
  - Kubernetes?
- Troubleshooting:
  - Common errors
  - Debug mode
  - Getting help

**Target Length:** 400-500 lines

---

### 16. guides/community.mdx

**Purpose:** Connect users to the community.

**Content:**
- Discord server (link, what to expect)
- GitHub:
  - Main repository
  - Issues
  - Discussions
  - Contributing guidelines
- Twitter/X
- Blog posts and articles
- Video tutorials
- Books and courses (if any)
- Showcase: Apps built with Mizu
- How to contribute:
  - Code contributions
  - Documentation
  - Examples
  - Bug reports
- Code of conduct
- Maintainers and sponsors

**Target Length:** 250-300 lines

---

## Implementation Priority

### Phase 1: Essential Pages
1. welcome.mdx - Entry point
2. what-is-mizu.mdx - Core understanding
3. installation.mdx - Getting started
4. core.mdx - Main framework

### Phase 2: Building Blocks
5. middlewares.mdx
6. contract.mdx
7. view.mdx
8. frontend.mdx
9. mobile.mdx
10. cli.mdx

### Phase 3: Tutorials
11. first-api.mdx
12. first-website.mdx
13. first-fullstack.mdx

### Phase 4: Reference
14. architecture.mdx
15. faq.mdx
16. community.mdx

## Writing Guidelines

### Tone
- Welcoming and friendly
- Assume reader is new to Mizu (but knows Go basics)
- Focus on "why" before "how"
- Use analogies and real-world examples

### Structure
- Start with clear introduction
- Use progressive disclosure
- Include visual diagrams where helpful
- End with clear next steps

### Code Examples
- Keep examples short and focused
- Show complete, runnable code
- Include expected output
- Use comments to explain

### Links
- Link liberally to detailed documentation
- Use relative links for internal pages
- Open external links in new tabs

## Success Metrics

- User can understand what Mizu is after reading welcome page
- User can install and run first app after installation page
- User understands all components after building blocks section
- User can complete tutorials independently
- User knows where to get help after community page
