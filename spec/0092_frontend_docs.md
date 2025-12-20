# Frontend Documentation Plan

## Overview

This spec outlines a comprehensive frontend documentation section for Mizu. The documentation targets absolute beginners and covers both traditional server-rendered approaches (HTMX) and modern SPA frameworks (React, Vue, Svelte, Angular, etc.).

## Documentation Structure

The Frontend documentation will be organized into the following sections:

### 1. Getting Started (Fundamentals)
- **Overview** (`docs/frontend/overview.mdx`)
  - What is the frontend middleware
  - Development vs Production modes
  - Supported frameworks
  - When to use each approach (HTMX vs SPA)

- **Quick Start** (`docs/frontend/quick-start.mdx`)
  - Creating your first frontend app
  - Using the CLI to scaffold projects
  - Running in development mode
  - Building for production

### 2. Core Concepts
- **Development Mode** (`docs/frontend/development.mdx`)
  - How dev mode works (proxy to Vite/webpack)
  - Hot Module Replacement (HMR)
  - WebSocket proxying
  - Configuration options
  - Troubleshooting dev server issues

- **Production Mode** (`docs/frontend/production.mdx`)
  - Serving static files
  - Asset optimization
  - Embedded filesystems
  - Cache strategies (hashed vs unhashed assets)
  - Security headers
  - Source maps handling

- **Configuration** (`docs/frontend/configuration.mdx`)
  - Mode selection (Auto/Dev/Production)
  - Environment variables (MIZU_ENV)
  - Directory structure
  - Ignore paths for APIs
  - URL prefixes
  - Custom error handlers

### 3. Framework Integration
- **React** (`docs/frontend/react.mdx`)
  - Setting up React with Vite
  - Project structure
  - Development workflow
  - Building and deployment
  - Using the React adapter
  - API integration examples
  - State management patterns

- **Vue** (`docs/frontend/vue.mdx`)
  - Setting up Vue with Vite
  - Project structure
  - Vue Router integration
  - Development workflow
  - Building and deployment
  - Using the Vue adapter
  - API integration examples

- **Svelte** (`docs/frontend/svelte.mdx`)
  - Setting up Svelte with Vite
  - Project structure
  - Svelte routing
  - Development workflow
  - Building and deployment
  - Using the Svelte adapter

- **SvelteKit** (`docs/frontend/sveltekit.mdx`)
  - SvelteKit as a full-stack framework
  - Static adapter configuration
  - Routing with SvelteKit
  - API routes vs Mizu backend
  - When to use SvelteKit with Mizu

- **Angular** (`docs/frontend/angular.mdx`)
  - Setting up Angular
  - Angular CLI integration
  - Project structure
  - Development workflow
  - Building and deployment
  - Using the Angular adapter

- **HTMX** (`docs/frontend/htmx.mdx`)
  - What is HTMX and when to use it
  - Server-rendered HTML approach
  - Go templates integration
  - Partial updates with HTMX
  - Forms and validation
  - Real-time updates
  - Combining with Alpine.js for reactivity

- **Next.js** (`docs/frontend/nextjs.mdx`)
  - Using Next.js with Mizu
  - Static export mode
  - Development setup
  - API routes vs Mizu backend
  - When to use Next.js with Mizu

- **Nuxt** (`docs/frontend/nuxt.mdx`)
  - Using Nuxt with Mizu
  - Static generation mode
  - Development setup
  - API routes vs Mizu backend

- **Preact** (`docs/frontend/preact.mdx`)
  - Lightweight React alternative
  - Setup and configuration
  - Development workflow

- **Alpine.js** (`docs/frontend/alpine.mdx`)
  - Minimal JavaScript framework
  - Server-rendered templates
  - Progressive enhancement
  - Combining with HTMX

### 4. Advanced Features
- **Caching Strategy** (`docs/frontend/caching.mdx`)
  - Understanding cache headers
  - Hashed assets (immutable)
  - Unhashed assets
  - HTML files (no-cache)
  - Custom cache patterns
  - Cache-Control directives
  - Best practices

- **Environment Injection** (`docs/frontend/env-injection.mdx`)
  - Injecting server-side env vars
  - window.__ENV__ pattern
  - Security considerations
  - Example use cases
  - Runtime configuration vs build-time

- **Build Manifest** (`docs/frontend/manifest.mdx`)
  - What is a build manifest
  - Vite manifest format
  - Webpack manifest format
  - Asset fingerprinting
  - Preloading modules
  - CSS extraction
  - Using manifest in templates

- **Security** (`docs/frontend/security.mdx`)
  - Default security headers
  - Content Security Policy (CSP)
  - X-Frame-Options
  - X-Content-Type-Options
  - Referrer-Policy
  - Custom security headers
  - Source map protection

- **Service Workers** (`docs/frontend/service-workers.mdx`)
  - Configuring service workers
  - Service worker scope
  - Caching strategies
  - Offline support
  - Progressive Web Apps (PWA)

### 5. Working with APIs
- **API Integration** (`docs/frontend/api-integration.mdx`)
  - Separating frontend and backend routes
  - Ignore paths configuration
  - CORS in development
  - Authentication patterns
  - Error handling
  - TypeScript types sharing

- **SSR vs SPA** (`docs/frontend/ssr-vs-spa.mdx`)
  - Server-Side Rendering explained
  - Single Page Applications explained
  - Hybrid approaches
  - SEO considerations
  - Performance trade-offs
  - Choosing the right approach

### 6. Deployment
- **Building for Production** (`docs/frontend/building.mdx`)
  - Build process overview
  - Optimizations (minification, tree-shaking)
  - Asset fingerprinting
  - Output directory structure
  - Multi-environment builds

- **Embedded Filesystems** (`docs/frontend/embed.mdx`)
  - Using Go embed directive
  - Benefits of embedded FS
  - Setup and configuration
  - Single binary deployment
  - Updating embedded assets

- **Static Hosting** (`docs/frontend/static-hosting.mdx`)
  - Deploying to CDN
  - Separating frontend and backend
  - Environment configuration
  - Cache headers for CDN

### 7. Templates
- **Template Overview** (`docs/frontend/templates.mdx`)
  - Available CLI templates
  - Template structure
  - Customizing templates
  - Creating custom templates

- **Minimal Setup** (`docs/frontend/minimal-setup.mdx`)
  - Manual setup without CLI
  - Installing dependencies
  - Configuring build tools
  - Creating the middleware

### 8. Recipes & Examples
- **Common Patterns** (`docs/frontend/patterns.mdx`)
  - Authentication flow
  - Form handling
  - File uploads
  - Real-time updates
  - Pagination
  - Infinite scroll

- **Troubleshooting** (`docs/frontend/troubleshooting.mdx`)
  - Dev server connection issues
  - Build errors
  - Runtime errors
  - CORS issues
  - Path resolution problems
  - Common mistakes

### 9. Reference
- **API Reference** (`docs/frontend/api-reference.mdx`)
  - Complete API documentation
  - Options struct
  - CacheConfig struct
  - Mode constants
  - Helper functions
  - Manifest API

- **Framework Adapters** (`docs/frontend/adapters.mdx`)
  - Using framework adapters
  - Available adapters
  - Adapter-specific defaults
  - Creating custom adapters

## Writing Guidelines

### Tone and Style
- Write for absolute beginners
- Explain concepts before showing code
- Use clear, simple language
- Avoid jargon or explain it when used
- Include "why" along with "how"
- Progressive disclosure (simple → advanced)

### Code Examples
- Always include complete, working examples
- Show both development and production setups
- Include comments for clarity
- Demonstrate common patterns
- Show error handling
- Include TypeScript examples where relevant

### Structure
- No level 1 headers (#) - use frontmatter title
- Keep titles short (e.g., "Overview" not "Frontend Overview")
- Start with a clear description
- Use practical examples early
- Include visual aids (diagrams, screenshots) where helpful
- End with "Next Steps" linking to related topics

### Comparisons and Context
- Compare with familiar technologies
- Explain when to use each approach
- Discuss trade-offs honestly
- Provide decision-making guidance

## Implementation Notes

### Phase 1: Core Documentation
1. Overview and Quick Start
2. Development and Production modes
3. Configuration

### Phase 2: Framework Guides
4. React, Vue, Svelte (most popular)
5. HTMX (server-rendered alternative)
6. Other frameworks

### Phase 3: Advanced Topics
7. Caching, Security, Manifest
8. Service Workers, API Integration

### Phase 4: Deployment & Reference
9. Building, Embed, Deployment
10. API Reference, Troubleshooting

## Success Criteria

The documentation should enable a beginner to:
1. Understand what frontend options Mizu provides
2. Choose the right approach for their project
3. Set up a development environment
4. Build and deploy a production application
5. Troubleshoot common issues
6. Optimize performance and security
7. Integrate with backend APIs effectively

## Cross-References

Link to related documentation:
- Core Mizu concepts (routing, handlers, middleware)
- View middleware for server-rendered templates
- Static middleware for simple file serving
- CORS middleware for API integration
- Embed middleware for embedded filesystems

## Examples to Include

Each framework guide should include:
1. "Hello World" example
2. API integration example
3. Form handling example
4. Authentication example
5. Production deployment example

The examples should be realistic and production-ready, not just minimal demonstrations.
