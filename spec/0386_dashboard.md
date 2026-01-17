# 0386 - Localbase Dashboard (Supabase-Style)

## Overview

This document describes the design and implementation of a Supabase-style dashboard for Localbase, providing a local development experience that mirrors the official Supabase Studio interface with 100% light theme styling accuracy.

## Goals

1. **100% Supabase Light Theme Match** - Pixel-perfect recreation of Supabase's design system
2. **Full Backend Integration** - No mocking; all features work with the real Go backend
3. **Embedded Assets** - Frontend built and embedded into Go binary for single-binary deployment
4. **Feature Parity** - All core Supabase Studio features: Database, Auth, Storage, Realtime, Functions

## Technology Stack

### Frontend
- **React 19** - UI framework with latest features
- **TypeScript 5.7** - Type safety
- **Vite 6** - Build tool with HMR
- **Mantine 8** - Component library (styled to match Supabase)
- **React Router DOM 7** - Client-side routing
- **Monaco Editor 4.6** - SQL query editor
- **Zustand 5** - State management
- **Tabler Icons** - Icon library

### Backend (Existing)
- **Go + Mizu Framework** - HTTP server
- **PostgreSQL** - Database
- **Supabase-compatible APIs** - GoTrue, PostgREST, Storage

## Design System

### Colors (Supabase Light Theme)

```css
/* Brand Colors */
--supabase-brand: #3ECF8E           /* Primary green */
--supabase-brand-hover: #24B47E     /* Hover state */
--supabase-brand-light: rgba(62, 207, 142, 0.1)   /* Light background */
--supabase-brand-muted: rgba(62, 207, 142, 0.15)  /* Muted background */

/* Background Colors */
--supabase-bg: #FFFFFF              /* Main background */
--supabase-bg-surface: #F8F9FA      /* Surface/card background */
--supabase-bg-surface-alt: #F4F4F5  /* Alternative surface */
--supabase-bg-surface-hover: #F1F3F5 /* Hover state */
--supabase-bg-sidebar: #FAFAFA      /* Sidebar background */
--supabase-bg-muted: #F4F4F5        /* Muted elements */

/* Text Colors (12-step scale) */
--supabase-text-1200: #11181C       /* Darkest */
--supabase-text-1100: #1C1C1C       /* Primary text */
--supabase-text-1000: #3F3F46       /* Secondary text */
--supabase-text-900: #52525B        /* Tertiary text */
--supabase-text: #1C1C1C            /* Default text */
--supabase-text-secondary: #666666  /* Secondary */
--supabase-text-muted: #888888      /* Muted */
--supabase-text-light: #AAAAAA      /* Light/disabled */
--supabase-text-placeholder: #A1A1AA /* Placeholders */

/* Border Colors */
--supabase-border: #E6E8EB          /* Default border */
--supabase-border-muted: #EAEAEA    /* Subtle border */
--supabase-border-strong: #C9CDD3   /* Emphasized border */
--supabase-border-focus: #3ECF8E    /* Focus state */

/* Semantic Colors */
--supabase-success: #3ECF8E
--supabase-success-bg: rgba(62, 207, 142, 0.15)
--supabase-success-text: #059669

--supabase-warning: #F5A623
--supabase-warning-bg: rgba(245, 166, 35, 0.15)
--supabase-warning-text: #D97706

--supabase-error: #EF4444
--supabase-error-bg: rgba(239, 68, 68, 0.15)
--supabase-error-text: #DC2626

--supabase-info: #3B82F6
--supabase-info-bg: rgba(59, 130, 246, 0.15)
--supabase-info-text: #2563EB
```

### Typography

```css
/* Font Stack */
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;

/* Code/Mono */
font-family: 'Source Code Pro', 'Menlo', 'Monaco', monospace;

/* Sizes */
--text-xs: 0.75rem;   /* 12px */
--text-sm: 0.875rem;  /* 14px */
--text-base: 1rem;    /* 16px */
--text-lg: 1.125rem;  /* 18px */
--text-xl: 1.25rem;   /* 20px */
--text-2xl: 1.5rem;   /* 24px */
```

### Spacing & Sizing

```css
/* Border Radius */
--supabase-radius-sm: 4px
--supabase-radius: 6px
--supabase-radius-lg: 8px
--supabase-radius-xl: 12px

/* Shadows */
--supabase-shadow-xs: 0 1px 2px rgba(0, 0, 0, 0.04)
--supabase-shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.04)
--supabase-shadow: 0 1px 3px rgba(0, 0, 0, 0.04)
--supabase-shadow-md: 0 4px 6px rgba(0, 0, 0, 0.04)
--supabase-shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.04)
--supabase-shadow-dropdown: 0 4px 20px rgba(0, 0, 0, 0.08)

/* Transitions */
--supabase-transition: 0.15s ease
```

## Project Structure

```
blueprints/localbase/
├── app/
│   ├── frontend/                    # React frontend
│   │   ├── src/
│   │   │   ├── api/                 # API client layer
│   │   │   │   ├── client.ts        # Base HTTP client
│   │   │   │   ├── auth.ts          # Auth API
│   │   │   │   ├── storage.ts       # Storage API
│   │   │   │   ├── database.ts      # Database API
│   │   │   │   ├── functions.ts     # Functions API
│   │   │   │   ├── realtime.ts      # Realtime API
│   │   │   │   ├── dashboard.ts     # Dashboard API
│   │   │   │   └── index.ts         # Barrel export
│   │   │   ├── components/
│   │   │   │   ├── common/          # Reusable components
│   │   │   │   │   ├── DataTable.tsx
│   │   │   │   │   ├── StatusBadge.tsx
│   │   │   │   │   ├── ConfirmModal.tsx
│   │   │   │   │   ├── EmptyState.tsx
│   │   │   │   │   └── SearchInput.tsx
│   │   │   │   ├── forms/           # Form components
│   │   │   │   └── layout/          # Layout components
│   │   │   │       ├── Sidebar.tsx
│   │   │   │       └── PageContainer.tsx
│   │   │   ├── hooks/
│   │   │   │   └── useApi.ts        # API integration hooks
│   │   │   ├── pages/               # Page components
│   │   │   │   ├── Dashboard.tsx
│   │   │   │   ├── ApiDocs.tsx
│   │   │   │   ├── auth/
│   │   │   │   │   └── Users.tsx
│   │   │   │   ├── database/
│   │   │   │   │   ├── TableEditor.tsx
│   │   │   │   │   └── SQLEditor.tsx
│   │   │   │   ├── storage/
│   │   │   │   │   └── Storage.tsx
│   │   │   │   ├── realtime/
│   │   │   │   │   └── Realtime.tsx
│   │   │   │   ├── functions/
│   │   │   │   │   └── Functions.tsx
│   │   │   │   └── settings/
│   │   │   │       └── Settings.tsx
│   │   │   ├── stores/
│   │   │   │   └── appStore.ts      # Zustand store
│   │   │   ├── types/
│   │   │   │   └── index.ts         # TypeScript types
│   │   │   ├── styles/
│   │   │   │   ├── index.css        # Global styles
│   │   │   │   └── supabase-theme.css # Supabase theme
│   │   │   ├── App.tsx              # Root component
│   │   │   └── main.tsx             # Entry point
│   │   ├── index.html
│   │   ├── vite.config.ts
│   │   ├── tsconfig.json
│   │   └── package.json
│   └── web/                         # Go HTTP handlers
│       ├── server.go
│       ├── handler/
│       │   └── api/
│       │       ├── auth.go
│       │       ├── storage.go
│       │       ├── database.go
│       │       ├── functions.go
│       │       ├── realtime.go
│       │       └── dashboard.go
│       └── middleware/
│           ├── apikey.go
│           └── ratelimit.go
├── assets/
│   ├── assets.go                    # Go embed directive
│   └── static/                      # Built frontend output
│       ├── index.html
│       └── assets/
│           ├── *.js
│           └── *.css
└── store/
    └── postgres/                    # Database layer
```

## Features

### 1. Dashboard (Home)

**Route:** `/`

**Features:**
- Statistics cards: Users, Tables, Buckets, Functions
- Service health status indicators
- Quick links to main features

**API Endpoints:**
- `GET /api/dashboard/stats` - Dashboard statistics
- `GET /api/dashboard/health` - Service health

### 2. Table Editor

**Route:** `/table-editor`

**Features:**
- Schema selector dropdown
- Table list sidebar with row counts
- Data grid with column type display
- RLS status indicator
- Create/delete tables
- View table data (max 100 rows)

**API Endpoints:**
- `GET /api/database/schemas` - List schemas
- `GET /api/database/tables?schema=` - List tables
- `GET /api/database/tables/{schema}/{name}` - Get table
- `POST /api/database/tables` - Create table
- `DELETE /api/database/tables/{schema}/{name}` - Drop table
- `GET /api/database/tables/{schema}/{name}/columns` - List columns
- `GET /rest/v1/{table}` - Select data (PostgREST)

### 3. SQL Editor

**Route:** `/sql-editor`

**Features:**
- Monaco code editor with SQL syntax highlighting
- Execute queries with Ctrl/Cmd+Enter
- Results table with pagination
- Saved queries (persisted in localStorage)
- CSV export

**API Endpoints:**
- `POST /api/database/query` - Execute SQL

### 4. Authentication (Users)

**Route:** `/auth/users`

**Features:**
- User list with search
- User creation (email + password)
- User editing
- User deletion
- Email verification status
- Last sign-in display

**API Endpoints:**
- `GET /auth/v1/admin/users` - List users
- `POST /auth/v1/admin/users` - Create user
- `GET /auth/v1/admin/users/{id}` - Get user
- `PUT /auth/v1/admin/users/{id}` - Update user
- `DELETE /auth/v1/admin/users/{id}` - Delete user

### 5. Storage

**Route:** `/storage`

**Features:**
- Bucket list sidebar
- File browser with breadcrumb navigation
- Drag-drop file upload
- File download
- Copy public/signed URLs
- Create/delete buckets
- Public/private bucket toggle

**API Endpoints:**
- `GET /storage/v1/bucket` - List buckets
- `POST /storage/v1/bucket` - Create bucket
- `DELETE /storage/v1/bucket/{id}` - Delete bucket
- `POST /storage/v1/object/list/{bucket}` - List objects
- `POST /storage/v1/object/{bucket}/{path}` - Upload
- `GET /storage/v1/object/{bucket}/{path}` - Download
- `DELETE /storage/v1/object/{bucket}/{path}` - Delete
- `POST /storage/v1/object/sign/{bucket}/{path}` - Signed URL

### 6. Realtime

**Route:** `/realtime`

**Features:**
- Connection statistics
- Active channels list
- Real-time message capture
- WebSocket connection status

**API Endpoints:**
- `GET /api/realtime/stats` - Connection stats
- `GET /api/realtime/channels` - Active channels
- `WS /realtime/v1/websocket` - WebSocket connection

### 7. Edge Functions

**Route:** `/functions`

**Features:**
- Function list with status
- Create/edit/delete functions
- Deploy functions
- View deployment history
- Secrets management

**API Endpoints:**
- `GET /api/functions` - List functions
- `POST /api/functions` - Create function
- `PUT /api/functions/{id}` - Update function
- `DELETE /api/functions/{id}` - Delete function
- `POST /api/functions/{id}/deploy` - Deploy
- `GET /api/functions/{id}/deployments` - Deployments
- `GET /api/functions/secrets` - List secrets
- `POST /api/functions/secrets` - Create secret
- `POST /functions/v1/{name}` - Invoke function

### 8. API Docs

**Route:** `/api-docs`

**Features:**
- Base URL display
- Authentication instructions
- REST endpoint reference
- Copy buttons for endpoints

### 9. Settings

**Route:** `/settings`

**Features:**
- Project name configuration
- API keys display (anon, service, JWT secret)
- Copy buttons for keys
- Database connection info

## Component Library

### Layout Components

#### Sidebar
```tsx
interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}
```

Features:
- Collapsible (70px vs 250px)
- Project name + status badge
- Navigation with active states
- Tooltips when collapsed
- Settings at bottom

#### PageContainer
```tsx
interface PageContainerProps {
  title: string;
  description?: string;
  action?: ReactNode;
  fullWidth?: boolean;
  noPadding?: boolean;
  children: ReactNode;
}
```

### Common Components

#### DataTable
```tsx
interface DataTableProps<T> {
  data: T[];
  columns: Column[];
  loading?: boolean;
  onRowClick?: (row: T) => void;
  selectable?: boolean;
  pagination?: boolean;
}
```

#### ConfirmModal
```tsx
interface ConfirmModalProps {
  opened: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmLabel?: string;
  danger?: boolean;
  loading?: boolean;
}
```

#### EmptyState
```tsx
interface EmptyStateProps {
  icon: ReactNode;
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}
```

## State Management

### Zustand Store
```typescript
interface AppState {
  // UI
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;

  // Project
  projectName: string;
  setProjectName: (name: string) => void;
  serviceKey: string;
  setServiceKey: (key: string) => void;

  // Storage
  selectedBucket: string | null;
  setSelectedBucket: (bucket: string | null) => void;
  currentPath: string;
  setCurrentPath: (path: string) => void;

  // Database
  selectedSchema: string;
  setSelectedSchema: (schema: string) => void;
  selectedTable: string | null;
  setSelectedTable: (table: string | null) => void;

  // SQL Editor
  savedQueries: SavedQuery[];
  addSavedQuery: (query: SavedQuery) => void;
  removeSavedQuery: (id: string) => void;
}
```

**Persistence:** Selected fields persisted to localStorage via Zustand persist middleware.

## API Client

### Base Client
```typescript
class ApiClient {
  private getServiceKey(): string;

  async get<T>(path: string): Promise<T>;
  async post<T>(path: string, body?: unknown): Promise<T>;
  async put<T>(path: string, body?: unknown): Promise<T>;
  async patch<T>(path: string, body?: unknown): Promise<T>;
  async delete<T>(path: string): Promise<T>;

  async uploadFile(path: string, file: File): Promise<any>;
  getAuthenticatedUrl(path: string): string;
}
```

### Authentication Headers
```
apikey: {service_key}
Authorization: Bearer {service_key}
Content-Type: application/json
```

### Custom Hooks
```typescript
// Single request hook
function useApi<T, P extends any[]>(
  fn: (...args: P) => Promise<T>,
  options?: UseApiOptions
): {
  data: T | null;
  loading: boolean;
  error: string | null;
  execute: (...args: P) => Promise<T | null>;
  reset: () => void;
  setData: (data: T | null) => void;
}

// Paginated request hook
function usePaginatedApi<T>(
  fn: (page: number, perPage: number) => Promise<PaginatedResponse<T>>,
  options?: UsePaginatedApiOptions
): {
  data: T[];
  loading: boolean;
  error: string | null;
  page: number;
  totalPages: number;
  total: number;
  setPage: (page: number) => void;
  refresh: () => Promise<void>;
}
```

## Build & Deployment

### Development
```bash
# Start frontend dev server (port 5173)
cd app/frontend && npm run dev

# Start backend (port 54321)
make run

# Or run both together
make dev-all
```

### Production Build
```bash
# Build frontend to assets/static/
cd app/frontend && npm run build

# Build Go binary with embedded assets
make build
```

### Vite Configuration
```typescript
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
      '@components': resolve(__dirname, './src/components'),
      '@pages': resolve(__dirname, './src/pages'),
      '@api': resolve(__dirname, './src/api'),
      '@stores': resolve(__dirname, './src/stores'),
      '@hooks': resolve(__dirname, './src/hooks'),
      '@types': resolve(__dirname, './src/types'),
    },
  },
  build: {
    outDir: '../../assets/static',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-react': ['react', 'react-dom', 'react-router-dom'],
          'vendor-mantine': ['@mantine/core', '@mantine/hooks', '@mantine/notifications'],
          'vendor-monaco': ['@monaco-editor/react'],
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://localhost:54321', changeOrigin: true },
      '/auth': { target: 'http://localhost:54321', changeOrigin: true },
      '/storage': { target: 'http://localhost:54321', changeOrigin: true },
      '/rest': { target: 'http://localhost:54321', changeOrigin: true },
      '/realtime': { target: 'http://localhost:54321', changeOrigin: true, ws: true },
    },
  },
})
```

### Go Asset Embedding
```go
package assets

import "embed"

//go:embed static/*
var StaticFS embed.FS
```

## Security

### API Key Middleware
All API endpoints require authentication via:
- `apikey` header with service role JWT
- `Authorization: Bearer` header with same key

### Service Role Requirement
Admin operations (user management, database schema, functions) require `service_role` claim in JWT.

### Rate Limiting
Auth endpoints have rate limiting for brute force protection.

## Testing

### Frontend Tests
```bash
cd app/frontend && npm test
```

### Backend Tests
```bash
make test
```

### Integration Tests
- All API endpoints tested with real database
- Full end-to-end flows verified

## Responsive Design

### Breakpoints
- `sm`: 576px
- `md`: 768px
- `lg`: 992px
- `xl`: 1200px

### Sidebar Behavior
- Collapsible at all sizes
- 70px collapsed / 250px expanded
- Tooltips shown when collapsed

### Grid Layouts
- Dashboard cards: 1 col (mobile) / 2 cols (tablet) / 4 cols (desktop)
- Split panels: Stack on mobile, side-by-side on desktop

## Accessibility

- WCAG 2.1 AA compliant
- Keyboard navigation support
- Focus indicators
- Screen reader labels
- Semantic HTML structure
- Color contrast ratios meet standards

## Performance

### Optimizations
- Code splitting with manual chunks
- Tree shaking
- Gzip compression
- Lazy loading for Monaco editor
- Debounced search inputs
- Virtual scrolling for large tables

### Bundle Sizes (target)
- Initial: < 200KB gzipped
- Monaco chunk: lazy loaded
- Vendor chunks: cached separately

## Future Enhancements

1. **Dark Theme Toggle** - Add dark mode support
2. **Real-time Data Grid** - WebSocket updates for table changes
3. **SQL Autocomplete** - Intelligent code completion
4. **Database Migrations** - Version-controlled schema changes
5. **Logs Viewer** - Real-time log streaming
6. **Performance Dashboard** - Query analytics and metrics
