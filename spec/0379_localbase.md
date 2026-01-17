# Localbase - Offline Supabase Clone

## Overview

Localbase is a comprehensive, offline-first implementation of Supabase's core features. It provides a local development environment with 100% feature parity with Supabase (2025), including a full-featured dashboard UI.

## Architecture Design

### Directory Structure

```
localbase/
├── app/                          # Web application
│   ├── frontend/                 # React TypeScript frontend (Vite + Mantine UI)
│   │   ├── src/
│   │   │   ├── api/              # API client definitions
│   │   │   ├── components/       # Reusable React components
│   │   │   ├── pages/            # Page components
│   │   │   ├── stores/           # Zustand state management
│   │   │   ├── types/            # TypeScript type definitions
│   │   │   ├── App.tsx           # Main app component
│   │   │   └── main.tsx          # Entry point
│   │   ├── package.json
│   │   ├── tsconfig.json
│   │   └── vite.config.ts
│   └── web/                      # Go HTTP server
│       ├── handler/api/          # API route handlers
│       ├── server.go             # Main server setup & routing
│       └── store_adapter.go      # Store adapter implementations
│
├── feature/                      # Feature modules (Supabase APIs)
│   ├── database/                 # PostgreSQL database management
│   │   ├── api.go                # Type definitions
│   │   └── service.go            # Business logic
│   ├── auth/                     # Authentication (GoTrue compatible)
│   │   ├── api.go
│   │   └── service.go
│   ├── storage/                  # File storage (S3 compatible)
│   │   ├── api.go
│   │   └── service.go
│   ├── realtime/                 # WebSocket realtime
│   │   ├── api.go
│   │   └── service.go
│   ├── edge_functions/           # Deno-compatible edge functions
│   │   ├── api.go
│   │   └── service.go
│   ├── vectors/                  # pgvector embeddings
│   │   ├── api.go
│   │   └── service.go
│   ├── rest/                     # PostgREST auto-generated API
│   │   ├── api.go
│   │   └── service.go
│   └── graphql/                  # GraphQL API (pg_graphql)
│       ├── api.go
│       └── service.go
│
├── store/                        # Data persistence layer
│   ├── postgres/                 # PostgreSQL implementations
│   │   ├── store.go              # Main PostgreSQL store
│   │   ├── database.go           # Database management
│   │   ├── auth.go               # Auth store
│   │   ├── storage.go            # Storage metadata
│   │   └── ...
│   └── store.go                  # Store interface definitions
│
├── cli/                          # Command-line interface
│   ├── root.go                   # Root command
│   ├── serve.go                  # serve command
│   ├── init.go                   # init command
│   ├── seed.go                   # seed command
│   └── ui.go                     # UI utilities
│
├── pkg/                          # Reusable packages
│   ├── postgrest/                # PostgREST implementation
│   ├── gotrue/                   # GoTrue auth implementation
│   ├── realtime/                 # Realtime WebSocket server
│   ├── storage/                  # S3-compatible storage
│   ├── deno/                     # Deno runtime for edge functions
│   └── graphql/                  # GraphQL engine
│
├── runtime/                      # Edge Functions runtime
│   ├── runtime.go                # Core Deno runtime
│   ├── bindings.go               # API bindings
│   └── pool.go                   # Worker pool
│
├── cmd/                          # Entry points
│   └── localbase/
│       └── main.go
│
├── docker/                       # Docker configurations
│   ├── postgres/                 # PostgreSQL with extensions
│   │   └── docker-compose.yml
│   └── all/                      # Full stack
│       └── docker-compose.yml
│
├── assets/                       # Static assets
│   ├── static/                   # Compiled frontend
│   ├── views/                    # HTML templates
│   └── assets.go                 # Asset embedding
│
├── Makefile                      # Build automation
├── go.mod                        # Go module
└── go.sum                        # Dependencies
```

## Core Features (Supabase 2025 Parity)

### 1. Database (PostgreSQL)
- **Full Postgres database** with pgvector, pg_graphql, pg_stat_statements
- **Visual Schema Designer** - Create/edit tables, columns, relationships
- **SQL Editor** with syntax highlighting, auto-completion, AI assistance
- **Table Editor** - Spreadsheet-like interface for data
- **Row Level Security (RLS)** - Visual policy editor
- **Database Backups** - Point-in-time recovery
- **Database Branches** - Preview changes before applying
- **Extensions** - Enable/manage Postgres extensions
- **Replication** - Read replicas support
- **Connection Pooling** - Via Supavisor-compatible pooler

### 2. Authentication (GoTrue Compatible)
- **Email/Password** authentication
- **Magic Links** - Passwordless login
- **Phone Auth** - SMS OTP
- **Social Providers** - Google, GitHub, Apple, Discord, etc.
- **Multi-Factor Auth (MFA)** - TOTP support
- **CAPTCHA Protection** - hCaptcha, Turnstile
- **Row Level Security** integration
- **Auth Hooks** - Custom logic on auth events
- **SSO/SAML** - Enterprise single sign-on
- **User Management** - Dashboard for user CRUD

### 3. Storage (S3 Compatible)
- **File Upload/Download** - Resumable uploads
- **Buckets** - Organize files
- **Access Policies** - RLS for storage
- **Image Transformations** - Resize, crop, format
- **CDN Integration** - Edge caching
- **S3 API** - Compatible with AWS SDK
- **Signed URLs** - Temporary access
- **Metadata** - Custom file metadata

### 4. Realtime
- **Postgres Changes** - Listen to INSERT/UPDATE/DELETE
- **Broadcast** - Pub/sub messaging
- **Presence** - Track online users
- **Channel Authorization** - Secure channels
- **Rate Limiting** - Per-connection limits

### 5. Edge Functions (Deno)
- **TypeScript/JavaScript** execution
- **NPM Modules** - Full Node.js compatibility
- **Deno Deploy** compatible runtime
- **Environment Variables** - Secure secrets
- **Scheduled Jobs** (Cron)
- **Regional Invocations** - Execute near database
- **Live Editor** - Edit and deploy from dashboard

### 6. REST API (PostgREST)
- **Auto-generated** from database schema
- **Filtering** - eq, neq, gt, lt, like, etc.
- **Ordering** - Single and multiple columns
- **Pagination** - offset, limit, range
- **Embedding** - Join related tables
- **Upsert** - Insert or update
- **Bulk Operations** - Batch insert/update

### 7. GraphQL API (pg_graphql)
- **Auto-generated** from database schema
- **Queries** - Read operations
- **Mutations** - Write operations
- **Subscriptions** - Realtime updates
- **Custom Resolvers** - Via SQL functions

### 8. Vectors (pgvector)
- **Vector Storage** - Store embeddings
- **Similarity Search** - cosine, euclidean, inner product
- **Indexing** - IVFFlat, HNSW
- **AI Integration** - OpenAI, Hugging Face
- **Vector Buckets** - Scale embeddings storage

### 9. Dashboard UI
All features accessible via web dashboard:
- **Project Overview** - Stats, health, usage
- **Table Editor** - Visual data management
- **SQL Editor** - Write and execute queries
- **Schema Visualizer** - ERD diagrams
- **Auth Users** - User management
- **Storage Browser** - File manager
- **Realtime Inspector** - Debug connections
- **Functions Editor** - Edge functions IDE
- **Logs Explorer** - Query logs
- **API Docs** - Auto-generated documentation
- **Settings** - Project configuration

## Implementation Checklist

### Phase 1: Core Infrastructure
- [ ] Project structure setup
- [ ] Makefile with all targets
- [ ] CLI scaffolding (Cobra + Fang)
- [ ] Go module initialization
- [ ] Store interface definitions
- [ ] PostgreSQL connection management
- [ ] Basic HTTP server (Mizu framework)

### Phase 2: Database Features
- [ ] PostgreSQL store implementation
- [ ] Schema management (migrations)
- [ ] Table CRUD operations
- [ ] Column management
- [ ] Index management
- [ ] Foreign key relationships
- [ ] RLS policy management
- [ ] SQL query execution
- [ ] Query results formatting
- [ ] Extension management

### Phase 3: Authentication
- [ ] GoTrue-compatible auth service
- [ ] Email/password registration
- [ ] Email/password login
- [ ] Magic link authentication
- [ ] Password reset flow
- [ ] Email verification
- [ ] Session management (JWT)
- [ ] Refresh token rotation
- [ ] OAuth provider framework
- [ ] Google OAuth
- [ ] GitHub OAuth
- [ ] MFA TOTP support
- [ ] User profile management
- [ ] Admin user management

### Phase 4: Storage
- [ ] S3-compatible storage service
- [ ] Bucket management
- [ ] File upload (single)
- [ ] File upload (multipart/resumable)
- [ ] File download
- [ ] File listing
- [ ] File deletion
- [ ] Storage policies (RLS)
- [ ] Image transformations
- [ ] Signed URLs
- [ ] Public/private buckets

### Phase 5: Realtime
- [ ] WebSocket server
- [ ] Postgres LISTEN/NOTIFY integration
- [ ] Channel subscription
- [ ] Broadcast messaging
- [ ] Presence tracking
- [ ] Authorization middleware
- [ ] Rate limiting

### Phase 6: Edge Functions
- [ ] Deno runtime integration
- [ ] Function deployment
- [ ] Function invocation (HTTP)
- [ ] Environment variables
- [ ] Supabase client binding
- [ ] Database access binding
- [ ] Storage access binding
- [ ] Cron trigger support
- [ ] Function logs
- [ ] Hot reload (dev mode)

### Phase 7: REST API (PostgREST)
- [ ] Schema introspection
- [ ] Auto endpoint generation
- [ ] GET with filters
- [ ] POST (insert)
- [ ] PATCH (update)
- [ ] DELETE
- [ ] Upsert support
- [ ] Bulk operations
- [ ] Embedding/joins
- [ ] RLS enforcement

### Phase 8: GraphQL API
- [ ] pg_graphql emulation
- [ ] Schema generation
- [ ] Query resolver
- [ ] Mutation resolver
- [ ] Subscription support
- [ ] Custom function resolvers

### Phase 9: Vectors
- [ ] pgvector extension setup
- [ ] Vector column support
- [ ] Similarity search
- [ ] Index creation (HNSW)
- [ ] Embedding generation (OpenAI)
- [ ] Vector bucket storage

### Phase 10: Frontend Dashboard
- [ ] React + Vite setup
- [ ] Mantine UI integration
- [ ] Zustand state management
- [ ] API client layer
- [ ] Authentication flow
- [ ] Layout/navigation
- [ ] Dashboard home page
- [ ] Table Editor page
- [ ] SQL Editor page
- [ ] Schema Visualizer
- [ ] Auth Users page
- [ ] Storage Browser page
- [ ] Realtime Inspector
- [ ] Edge Functions page
- [ ] API Docs page
- [ ] Settings page
- [ ] Logs Explorer page

### Phase 11: CLI & DevX
- [ ] `localbase init` - Initialize project
- [ ] `localbase serve` - Start all services
- [ ] `localbase db push` - Apply migrations
- [ ] `localbase db pull` - Generate from DB
- [ ] `localbase seed` - Seed sample data
- [ ] `localbase functions serve` - Local functions
- [ ] `localbase status` - Health check

### Phase 12: Docker & Deployment
- [ ] PostgreSQL Dockerfile (with extensions)
- [ ] Localbase server Dockerfile
- [ ] docker-compose.yml (all services)
- [ ] Health checks
- [ ] Graceful shutdown

## API Endpoints

### Database API
```
GET    /api/database/tables           # List tables
POST   /api/database/tables           # Create table
GET    /api/database/tables/:name     # Get table details
PUT    /api/database/tables/:name     # Update table
DELETE /api/database/tables/:name     # Delete table
GET    /api/database/tables/:name/columns
POST   /api/database/tables/:name/columns
GET    /api/database/schemas          # List schemas
POST   /api/database/query            # Execute SQL
GET    /api/database/extensions       # List extensions
POST   /api/database/extensions       # Enable extension
GET    /api/database/policies         # List RLS policies
POST   /api/database/policies         # Create policy
```

### Auth API (GoTrue Compatible)
```
POST   /auth/v1/signup                # Register user
POST   /auth/v1/token                 # Login (get token)
POST   /auth/v1/token?grant_type=refresh_token
POST   /auth/v1/logout                # Logout
POST   /auth/v1/recover               # Password recovery
PUT    /auth/v1/user                  # Update user
GET    /auth/v1/user                  # Get current user
POST   /auth/v1/otp                   # Send OTP
POST   /auth/v1/verify                # Verify OTP/Magic link
POST   /auth/v1/factors               # Enroll MFA
DELETE /auth/v1/factors/:id           # Unenroll MFA
POST   /auth/v1/factors/:id/challenge # MFA challenge
POST   /auth/v1/factors/:id/verify    # Verify MFA
GET    /auth/v1/admin/users           # List users (admin)
POST   /auth/v1/admin/users           # Create user (admin)
DELETE /auth/v1/admin/users/:id       # Delete user (admin)
```

### Storage API (S3 Compatible)
```
GET    /storage/v1/bucket             # List buckets
POST   /storage/v1/bucket             # Create bucket
GET    /storage/v1/bucket/:id         # Get bucket
PUT    /storage/v1/bucket/:id         # Update bucket
DELETE /storage/v1/bucket/:id         # Delete bucket
POST   /storage/v1/object/:bucket     # Upload file
GET    /storage/v1/object/:bucket/*   # Download file
DELETE /storage/v1/object/:bucket/*   # Delete file
POST   /storage/v1/object/list/:bucket
POST   /storage/v1/object/move        # Move/rename file
POST   /storage/v1/object/copy        # Copy file
POST   /storage/v1/object/sign/:bucket# Create signed URL
GET    /storage/v1/render/image/*     # Transform image
```

### Realtime API
```
WS     /realtime/v1/websocket         # WebSocket connection
GET    /api/realtime/channels         # List active channels
GET    /api/realtime/stats            # Connection stats
```

### Edge Functions API
```
GET    /api/functions                 # List functions
POST   /api/functions                 # Deploy function
GET    /api/functions/:name           # Get function
PUT    /api/functions/:name           # Update function
DELETE /api/functions/:name           # Delete function
POST   /api/functions/:name/invoke    # Invoke function
GET    /api/functions/:name/logs      # Get function logs
POST   /functions/v1/:name            # Public invoke endpoint
```

### REST API (PostgREST)
```
GET    /rest/v1/:table                # List records
POST   /rest/v1/:table                # Insert record(s)
PATCH  /rest/v1/:table                # Update record(s)
DELETE /rest/v1/:table                # Delete record(s)
POST   /rest/v1/rpc/:function         # Call function
```

### GraphQL API
```
POST   /graphql/v1                    # GraphQL endpoint
GET    /graphql/v1                    # GraphQL playground
```

### Dashboard API
```
GET    /api/dashboard/stats           # Overview stats
GET    /api/dashboard/health          # Health check
GET    /api/logs                      # Query logs
GET    /api/settings                  # Get settings
PUT    /api/settings                  # Update settings
```

## Technology Stack

### Backend
- **Go 1.22+** - Server implementation
- **Mizu** - HTTP framework
- **PostgreSQL 16+** - Primary database
  - pgvector - Vector embeddings
  - pg_graphql - GraphQL support
  - pg_stat_statements - Query statistics
- **Deno** - Edge functions runtime
- **gorilla/websocket** - WebSocket support
- **pgx** - PostgreSQL driver

### Frontend
- **React 19** - UI framework
- **TypeScript 5.9** - Type safety
- **Vite 7** - Build tool
- **Mantine 8** - UI component library
- **Zustand** - State management
- **React Router** - Routing
- **Monaco Editor** - SQL/Code editor
- **Recharts** - Data visualization
- **Framer Motion** - Animations

### CLI
- **Cobra** - CLI framework
- **Fang** - Enhanced CLI UX
- **Lipgloss** - Terminal styling

## Database Schema (Internal)

```sql
-- Auth tables
CREATE TABLE auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(50) UNIQUE,
    encrypted_password TEXT,
    email_confirmed_at TIMESTAMPTZ,
    phone_confirmed_at TIMESTAMPTZ,
    raw_app_meta_data JSONB DEFAULT '{}',
    raw_user_meta_data JSONB DEFAULT '{}',
    is_super_admin BOOLEAN DEFAULT FALSE,
    role VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_sign_in_at TIMESTAMPTZ,
    banned_until TIMESTAMPTZ,
    confirmation_token VARCHAR(255),
    recovery_token VARCHAR(255)
);

CREATE TABLE auth.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    factor_id UUID,
    aal VARCHAR(50),
    not_after TIMESTAMPTZ
);

CREATE TABLE auth.refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR(255) UNIQUE,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE CASCADE,
    parent VARCHAR(255),
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE auth.mfa_factors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    friendly_name TEXT,
    factor_type VARCHAR(50), -- totp, webauthn
    status VARCHAR(50), -- unverified, verified
    secret TEXT, -- TOTP secret (encrypted)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE auth.identities (
    id TEXT PRIMARY KEY,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    provider VARCHAR(50),
    provider_id TEXT,
    identity_data JSONB,
    last_sign_in_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Storage tables
CREATE TABLE storage.buckets (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    public BOOLEAN DEFAULT FALSE,
    file_size_limit BIGINT,
    allowed_mime_types TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE storage.objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket_id TEXT REFERENCES storage.buckets(id),
    name TEXT NOT NULL,
    owner UUID REFERENCES auth.users(id),
    path_tokens TEXT[] GENERATED ALWAYS AS (string_to_array(name, '/')) STORED,
    version TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ,
    UNIQUE(bucket_id, name)
);

-- Functions tables
CREATE TABLE functions.functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    version INTEGER DEFAULT 1,
    status VARCHAR(50) DEFAULT 'active',
    entrypoint TEXT DEFAULT 'index.ts',
    import_map TEXT,
    verify_jwt BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE functions.deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID REFERENCES functions.functions(id) ON DELETE CASCADE,
    version INTEGER,
    source_code TEXT,
    bundle_path TEXT,
    status VARCHAR(50),
    deployed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE functions.secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    value TEXT NOT NULL, -- encrypted
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Realtime tables
CREATE TABLE realtime.channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    inserted_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE realtime.subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID REFERENCES realtime.channels(id),
    user_id UUID REFERENCES auth.users(id),
    filters JSONB,
    claims JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## Configuration

### Environment Variables
```bash
# Database
DATABASE_URL=postgres://localbase:localbase@localhost:5432/localbase
DATABASE_POOL_SIZE=10

# Auth
JWT_SECRET=super-secret-jwt-key-at-least-32-chars
JWT_EXPIRY=3600
SITE_URL=http://localhost:3000

# Storage
STORAGE_BACKEND=local  # local, s3
STORAGE_PATH=/data/storage
S3_ENDPOINT=
S3_ACCESS_KEY=
S3_SECRET_KEY=
S3_BUCKET=

# Edge Functions
DENO_PATH=/usr/bin/deno
FUNCTIONS_PATH=/data/functions

# Server
PORT=54321
API_PORT=54321
STUDIO_PORT=3000
```

## Makefile Targets

```makefile
# Core
build           # Build binary with frontend
build-quick     # Build binary only
install         # Install to $HOME/bin
run             # Run development server
clean           # Remove artifacts

# Database
db-init         # Initialize database
db-reset        # Reset database
db-seed         # Seed sample data
db-migrate      # Run migrations

# Frontend
frontend-install  # Install dependencies
frontend-build    # Production build
frontend-dev      # Dev server
frontend-test     # Run tests

# Docker
docker-build    # Build images
docker-up       # Start services
docker-down     # Stop services
docker-logs     # View logs

# Development
dev             # Show dev instructions
dev-all         # Run all services
test            # Run tests
lint            # Run linter
```

## Sources
- [Supabase Features](https://supabase.com/features)
- [Supabase Documentation](https://supabase.com/docs/guides/getting-started/features)
- [Supabase Changelog 2025](https://supabase.com/changelog)
- [SQL Editor Features](https://supabase.com/features/sql-editor)
