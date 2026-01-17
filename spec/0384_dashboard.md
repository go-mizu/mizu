# 0384: Localbase Dashboard - Supabase-Style UI

## Overview

This specification defines the implementation of a production-grade dashboard for Localbase that closely mirrors the Supabase Dashboard design (light theme). The dashboard provides a visual interface for managing authentication, storage, database, realtime, and edge functions.

## Design Philosophy

### Supabase Design System Reference

The Supabase design system is built on:
- **Radix UI** primitives for accessible components
- **Tailwind CSS** for utility-first styling
- **Custom color system** based on Radix Colors with 12-step color scales

### Color Palette (Light Theme)

Based on the official Supabase design system:

```css
/* Primary Brand Colors */
--brand-primary: #3ECF8E;        /* Supabase Green */
--brand-secondary: #24B47E;      /* Darker green */

/* Background Colors */
--bg-default: #FFFFFF;           /* Main background */
--bg-surface-100: #F8F9FA;       /* Card backgrounds */
--bg-surface-200: #F1F3F5;       /* Alternate surfaces */
--bg-surface-300: #E9ECEF;       /* Hover states */
--bg-sidebar: #FAFAFA;           /* Sidebar background */
--bg-canvas: #F8F9FA;            /* Main canvas area */

/* Text Colors */
--text-primary: #1C1C1C;         /* Primary text */
--text-secondary: #666666;       /* Secondary text */
--text-muted: #888888;           /* Muted/placeholder text */
--text-light: #AAAAAA;           /* Light text */

/* Border Colors */
--border-default: #E6E8EB;       /* Default borders */
--border-muted: #EAEAEA;         /* Subtle borders */
--border-strong: #C9CDD3;        /* Strong emphasis */

/* Semantic Colors */
--success: #3ECF8E;              /* Success/green */
--warning: #F5A623;              /* Warning/amber */
--error: #EF4444;                /* Error/red */
--info: #3B82F6;                 /* Info/blue */
```

### Typography

```css
/* Font Stack */
--font-sans: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
--font-mono: 'Source Code Pro', 'Menlo', 'Monaco', monospace;

/* Font Sizes */
--text-xs: 0.75rem;    /* 12px */
--text-sm: 0.875rem;   /* 14px */
--text-base: 1rem;     /* 16px */
--text-lg: 1.125rem;   /* 18px */
--text-xl: 1.25rem;    /* 20px */
--text-2xl: 1.5rem;    /* 24px */
```

### Spacing System

```css
/* Consistent spacing based on 4px grid */
--space-1: 0.25rem;    /* 4px */
--space-2: 0.5rem;     /* 8px */
--space-3: 0.75rem;    /* 12px */
--space-4: 1rem;       /* 16px */
--space-5: 1.25rem;    /* 20px */
--space-6: 1.5rem;     /* 24px */
--space-8: 2rem;       /* 32px */
--space-10: 2.5rem;    /* 40px */
```

## Architecture

### Frontend Stack

The existing frontend uses:
- **React 19** for UI components
- **Mantine v8** for component library
- **React Router v7** for routing
- **Zustand** for state management
- **Monaco Editor** for SQL editing
- **Recharts** for data visualization
- **Vite** for build tooling

### Asset Embedding Pattern (from Localflare)

Following the pattern used in the codebase:

```go
// assets/assets.go
package assets

import "embed"

//go:embed static/*
var StaticFS embed.FS
```

The frontend builds to `../../assets/static` directory and is embedded at compile time.

### Directory Structure

```
app/frontend/
├── src/
│   ├── components/          # Reusable UI components
│   │   ├── layout/          # Layout components
│   │   │   ├── Sidebar.tsx
│   │   │   ├── Header.tsx
│   │   │   └── PageContainer.tsx
│   │   ├── common/          # Common UI elements
│   │   │   ├── DataTable.tsx
│   │   │   ├── EmptyState.tsx
│   │   │   ├── ConfirmModal.tsx
│   │   │   └── StatusBadge.tsx
│   │   └── forms/           # Form components
│   │       ├── FormField.tsx
│   │       └── SearchInput.tsx
│   ├── pages/               # Page components
│   │   ├── Dashboard.tsx
│   │   ├── auth/
│   │   │   └── Users.tsx
│   │   ├── storage/
│   │   │   └── Storage.tsx
│   │   ├── database/
│   │   │   ├── TableEditor.tsx
│   │   │   └── SQLEditor.tsx
│   │   ├── realtime/
│   │   │   └── Realtime.tsx
│   │   ├── functions/
│   │   │   └── Functions.tsx
│   │   └── settings/
│   │       └── Settings.tsx
│   ├── hooks/               # Custom React hooks
│   │   ├── useApi.ts
│   │   └── useStore.ts
│   ├── stores/              # Zustand stores
│   │   └── appStore.ts
│   ├── api/                 # API client functions
│   │   ├── client.ts
│   │   ├── auth.ts
│   │   ├── storage.ts
│   │   ├── database.ts
│   │   └── realtime.ts
│   ├── styles/              # Global styles
│   │   ├── index.css
│   │   └── supabase-theme.css
│   ├── types/               # TypeScript types
│   │   └── index.ts
│   ├── App.tsx
│   └── main.tsx
├── index.html
├── package.json
├── vite.config.ts
└── tsconfig.json
```

## Pages Specification

### 1. Dashboard (Home)

**Route:** `/`

**Layout:**
- Stats cards grid (4 columns on desktop)
- Quick actions section
- Recent activity feed

**Stats Cards:**
1. **Users** - Total registered users with trend indicator
2. **Tables** - Database tables count
3. **Storage** - Total buckets and storage usage
4. **Functions** - Active edge functions

**API Integration:**
- `GET /api/dashboard/stats` - Fetch all statistics
- `GET /api/dashboard/health` - Service health check

### 2. Authentication - Users Page

**Route:** `/auth/users`

**Layout:**
- Header with "Add User" button
- Search/filter bar
- Users data table with pagination
- User detail side panel (slide over)

**Table Columns:**
| Column | Description |
|--------|-------------|
| Email | User email with verification badge |
| Phone | Phone number (if set) |
| Provider | Auth provider (email, google, etc.) |
| Created | Account creation date |
| Last Sign In | Most recent login |
| Actions | Edit, Delete buttons |

**Features:**
- Create user modal
- Edit user modal
- Delete confirmation
- Search by email/phone
- Pagination (20 per page)
- Ban/unban user toggle

**API Integration:**
- `GET /auth/v1/admin/users` - List users
- `POST /auth/v1/admin/users` - Create user
- `GET /auth/v1/admin/users/{id}` - Get user details
- `PUT /auth/v1/admin/users/{id}` - Update user
- `DELETE /auth/v1/admin/users/{id}` - Delete user

### 3. Storage Page

**Route:** `/storage`

**Layout:**
- Bucket sidebar (left)
- File browser (center)
- File details panel (right, collapsible)

**Bucket List:**
- Create bucket button
- List of buckets with icons
- Public/private badge
- Bucket actions (settings, delete)

**File Browser:**
- Breadcrumb navigation
- Grid/List view toggle
- Upload button (drag & drop support)
- Multi-select for bulk operations
- Context menu (download, copy URL, delete)

**File Details:**
- Preview (images)
- Metadata display
- Public URL (for public buckets)
- Signed URL generator
- Delete button

**API Integration:**
- `GET /storage/v1/bucket` - List buckets
- `POST /storage/v1/bucket` - Create bucket
- `PUT /storage/v1/bucket/{id}` - Update bucket
- `DELETE /storage/v1/bucket/{id}` - Delete bucket
- `POST /storage/v1/object/list/{bucket}` - List objects
- `POST /storage/v1/object/{bucket}/{path}` - Upload object
- `GET /storage/v1/object/{bucket}/{path}` - Download object
- `DELETE /storage/v1/object/{bucket}/{path}` - Delete object
- `POST /storage/v1/object/sign/{bucket}/{path}` - Create signed URL

### 4. Database - Table Editor

**Route:** `/table-editor`

**Layout:**
- Schema/Table sidebar (left)
- Data grid (center)
- Column details panel (right)

**Features:**
- Schema selector dropdown
- Table list with row counts
- Create table modal
- Inline cell editing
- Add/edit/delete rows
- Column management (add, alter, drop)
- Foreign key visualization
- RLS policy management

**Data Grid:**
- Sortable columns
- Filterable columns
- Pagination
- Row selection
- Inline editing with validation

**API Integration:**
- `GET /api/database/schemas` - List schemas
- `GET /api/database/tables` - List tables
- `POST /api/database/tables` - Create table
- `DELETE /api/database/tables/{schema}/{name}` - Drop table
- `GET /rest/v1/{table}` - Fetch table data
- `POST /rest/v1/{table}` - Insert row
- `PATCH /rest/v1/{table}` - Update row
- `DELETE /rest/v1/{table}` - Delete row

### 5. Database - SQL Editor

**Route:** `/sql-editor`

**Layout:**
- Query editor (Monaco) with syntax highlighting
- Results panel (bottom)
- Query history sidebar (right, collapsible)

**Features:**
- SQL syntax highlighting
- Auto-completion for tables/columns
- Multiple query tabs
- Run query (Cmd/Ctrl + Enter)
- Format SQL button
- Export results (CSV, JSON)
- Save queries (local storage)
- Query execution time display

**API Integration:**
- `POST /api/database/query` - Execute SQL query

### 6. Realtime Page

**Route:** `/realtime`

**Layout:**
- Connection stats header
- Active channels list
- Message inspector (live WebSocket messages)

**Stats Display:**
- Active connections count
- Channel count
- Messages per second

**Channels Table:**
| Column | Description |
|--------|-------------|
| Name | Channel name |
| Subscribers | Active subscriber count |
| Created | Channel creation time |

**Message Inspector:**
- Live message stream
- Filter by channel
- JSON pretty-print
- Clear button

**API Integration:**
- `GET /api/realtime/channels` - List channels
- `GET /api/realtime/stats` - Get connection stats
- WebSocket `/realtime/v1/websocket` - Live message stream

### 7. Edge Functions Page

**Route:** `/functions`

**Layout:**
- Functions list (table)
- Function detail panel
- Logs viewer

**Functions Table:**
| Column | Description |
|--------|-------------|
| Name | Function name/slug |
| Status | Active/Inactive badge |
| Version | Current version |
| Last Deployed | Deployment timestamp |
| Actions | Deploy, Edit, Delete |

**Function Detail:**
- Code editor (Monaco)
- Deploy button
- Invoke test panel
- Deployment history
- Secret management

**API Integration:**
- `GET /api/functions` - List functions
- `POST /api/functions` - Create function
- `GET /api/functions/{id}` - Get function details
- `PUT /api/functions/{id}` - Update function
- `DELETE /api/functions/{id}` - Delete function
- `POST /api/functions/{id}/deploy` - Deploy function
- `GET /api/functions/{id}/deployments` - List deployments
- `GET /api/functions/secrets` - List secrets
- `POST /api/functions/secrets` - Create secret

### 8. API Docs Page

**Route:** `/api-docs`

**Layout:**
- Endpoint categories sidebar
- API documentation (center)
- Code examples panel (right)

**Features:**
- Auto-generated from routes
- Interactive "Try it" functionality
- Code examples (curl, JavaScript, Python)
- Authentication instructions

### 9. Settings Page

**Route:** `/settings`

**Layout:**
- Settings navigation tabs
- Settings form content

**Tabs:**
1. **General** - Project name, URL configuration
2. **API** - API keys display, JWT settings
3. **Auth** - Auth providers configuration
4. **Storage** - Default bucket settings
5. **Database** - Connection info, pooler settings

## Component Specifications

### Sidebar Component

```tsx
interface SidebarProps {
  collapsed?: boolean;
  onCollapse?: (collapsed: boolean) => void;
}
```

**Design:**
- 250px width expanded, 60px collapsed
- Logo + project name header
- Navigation items with icons
- Active state indicator (green left border)
- Collapse toggle button at bottom
- Settings link at bottom

**Navigation Items:**
1. Dashboard (IconDashboard)
2. Table Editor (IconTable)
3. SQL Editor (IconCode)
4. Authentication (IconUsers)
5. Storage (IconFolder)
6. Realtime (IconBolt)
7. Edge Functions (IconCloudCode)
8. API Docs (IconApi)
9. Settings (IconSettings)

### DataTable Component

```tsx
interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  loading?: boolean;
  pagination?: PaginationConfig;
  onRowClick?: (row: T) => void;
  selectable?: boolean;
  onSelectionChange?: (selected: T[]) => void;
  emptyState?: React.ReactNode;
}
```

**Features:**
- Sortable columns
- Loading skeleton
- Empty state
- Row selection
- Pagination controls
- Column resize (optional)

### EmptyState Component

```tsx
interface EmptyStateProps {
  icon: React.ReactNode;
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}
```

### ConfirmModal Component

```tsx
interface ConfirmModalProps {
  opened: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
}
```

## API Client Implementation

### Base Client

```typescript
// api/client.ts
const API_BASE = '';  // Same origin

interface ApiConfig {
  headers?: Record<string, string>;
}

class ApiClient {
  private serviceKey: string;

  constructor() {
    // Get service key from environment or localStorage
    this.serviceKey = import.meta.env.VITE_SERVICE_KEY ||
      localStorage.getItem('serviceKey') ||
      'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    config?: ApiConfig
  ): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'apikey': this.serviceKey,
        'Authorization': `Bearer ${this.serviceKey}`,
        ...config?.headers,
      },
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new ApiError(response.status, error.message || 'Request failed');
    }

    return response.json();
  }

  get<T>(path: string, config?: ApiConfig) {
    return this.request<T>('GET', path, undefined, config);
  }

  post<T>(path: string, body?: unknown, config?: ApiConfig) {
    return this.request<T>('POST', path, body, config);
  }

  put<T>(path: string, body?: unknown, config?: ApiConfig) {
    return this.request<T>('PUT', path, body, config);
  }

  patch<T>(path: string, body?: unknown, config?: ApiConfig) {
    return this.request<T>('PATCH', path, body, config);
  }

  delete<T>(path: string, config?: ApiConfig) {
    return this.request<T>('DELETE', path, undefined, config);
  }
}

export const api = new ApiClient();
```

### Auth API

```typescript
// api/auth.ts
import { api } from './client';

export interface User {
  id: string;
  email: string;
  phone: string;
  role: string;
  created_at: string;
  last_sign_in_at: string;
  app_metadata: Record<string, any>;
  user_metadata: Record<string, any>;
}

export const authApi = {
  listUsers: (page = 1, perPage = 20) =>
    api.get<{ users: User[]; total: number }>(
      `/auth/v1/admin/users?page=${page}&per_page=${perPage}`
    ),

  getUser: (id: string) =>
    api.get<User>(`/auth/v1/admin/users/${id}`),

  createUser: (data: { email: string; password: string; user_metadata?: Record<string, any> }) =>
    api.post<User>('/auth/v1/admin/users', data),

  updateUser: (id: string, data: Partial<User>) =>
    api.put<User>(`/auth/v1/admin/users/${id}`, data),

  deleteUser: (id: string) =>
    api.delete(`/auth/v1/admin/users/${id}`),
};
```

### Storage API

```typescript
// api/storage.ts
import { api } from './client';

export interface Bucket {
  id: string;
  name: string;
  public: boolean;
  created_at: string;
  file_size_limit?: number;
  allowed_mime_types?: string[];
}

export interface StorageObject {
  id: string;
  name: string;
  bucket_id: string;
  size: number;
  content_type: string;
  created_at: string;
  updated_at: string;
  metadata?: Record<string, string>;
}

export const storageApi = {
  // Buckets
  listBuckets: () =>
    api.get<Bucket[]>('/storage/v1/bucket'),

  createBucket: (data: { name: string; public?: boolean }) =>
    api.post<Bucket>('/storage/v1/bucket', data),

  getBucket: (id: string) =>
    api.get<Bucket>(`/storage/v1/bucket/${id}`),

  updateBucket: (id: string, data: Partial<Bucket>) =>
    api.put<Bucket>(`/storage/v1/bucket/${id}`, data),

  deleteBucket: (id: string) =>
    api.delete(`/storage/v1/bucket/${id}`),

  // Objects
  listObjects: (bucket: string, prefix = '', limit = 100, offset = 0) =>
    api.post<StorageObject[]>(`/storage/v1/object/list/${bucket}`, {
      prefix,
      limit,
      offset,
    }),

  uploadObject: async (bucket: string, path: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch(`/storage/v1/object/${bucket}/${path}`, {
      method: 'POST',
      headers: {
        'apikey': localStorage.getItem('serviceKey') || '',
      },
      body: formData,
    });

    if (!response.ok) {
      throw new Error('Upload failed');
    }

    return response.json();
  },

  downloadObject: (bucket: string, path: string) =>
    `/storage/v1/object/${bucket}/${path}`,

  deleteObject: (bucket: string, path: string) =>
    api.delete(`/storage/v1/object/${bucket}/${path}`),

  createSignedUrl: (bucket: string, path: string, expiresIn = 3600) =>
    api.post<{ signedURL: string }>(`/storage/v1/object/sign/${bucket}/${path}`, {
      expiresIn,
    }),
};
```

### Database API

```typescript
// api/database.ts
import { api } from './client';

export interface Table {
  id: number;
  schema: string;
  name: string;
  row_count: number;
  size_bytes: number;
  rls_enabled: boolean;
}

export interface Column {
  name: string;
  type: string;
  is_nullable: boolean;
  is_primary_key: boolean;
  default_value?: string;
}

export interface QueryResult {
  columns: string[];
  rows: Record<string, any>[];
  row_count: number;
  duration_ms: number;
}

export const databaseApi = {
  // Schemas
  listSchemas: () =>
    api.get<string[]>('/api/database/schemas'),

  // Tables
  listTables: (schema = 'public') =>
    api.get<Table[]>(`/api/database/tables?schema=${schema}`),

  getTable: (schema: string, name: string) =>
    api.get<Table>(`/api/database/tables/${schema}/${name}`),

  createTable: (schema: string, name: string, columns: Column[]) =>
    api.post('/api/database/tables', { schema, name, columns }),

  dropTable: (schema: string, name: string) =>
    api.delete(`/api/database/tables/${schema}/${name}`),

  // Columns
  listColumns: (schema: string, table: string) =>
    api.get<Column[]>(`/api/database/tables/${schema}/${table}/columns`),

  // Query
  executeQuery: (sql: string) =>
    api.post<QueryResult>('/api/database/query', { query: sql }),

  // REST API (PostgREST compatible)
  selectTable: (table: string, query?: string) =>
    api.get<any[]>(`/rest/v1/${table}${query ? `?${query}` : ''}`),

  insertRow: (table: string, data: Record<string, any>) =>
    api.post(`/rest/v1/${table}`, data),

  updateRow: (table: string, query: string, data: Record<string, any>) =>
    api.patch(`/rest/v1/${table}?${query}`, data),

  deleteRow: (table: string, query: string) =>
    api.delete(`/rest/v1/${table}?${query}`),
};
```

## State Management

### App Store (Zustand)

```typescript
// stores/appStore.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AppState {
  // Sidebar
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;

  // Current project
  projectName: string;
  setProjectName: (name: string) => void;

  // API configuration
  serviceKey: string;
  setServiceKey: (key: string) => void;

  // UI preferences
  theme: 'light' | 'dark';
  setTheme: (theme: 'light' | 'dark') => void;
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),

      projectName: 'localbase',
      setProjectName: (name) => set({ projectName: name }),

      serviceKey: '',
      setServiceKey: (key) => set({ serviceKey: key }),

      theme: 'light',
      setTheme: (theme) => set({ theme }),
    }),
    {
      name: 'localbase-app',
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        serviceKey: state.serviceKey,
        theme: state.theme,
      }),
    }
  )
);
```

## CSS Customization

### Supabase Theme Override

```css
/* styles/supabase-theme.css */

:root {
  /* Supabase Light Theme Colors */
  --supabase-brand: #3ECF8E;
  --supabase-brand-hover: #24B47E;

  /* Backgrounds */
  --supabase-bg: #FFFFFF;
  --supabase-bg-surface: #F8F9FA;
  --supabase-bg-surface-hover: #F1F3F5;
  --supabase-bg-sidebar: #FAFAFA;

  /* Text */
  --supabase-text: #1C1C1C;
  --supabase-text-secondary: #666666;
  --supabase-text-muted: #888888;

  /* Borders */
  --supabase-border: #E6E8EB;
  --supabase-border-muted: #EAEAEA;
}

/* Override Mantine defaults for Supabase look */
.mantine-AppShell-navbar {
  background-color: var(--supabase-bg-sidebar);
  border-right: 1px solid var(--supabase-border);
}

.mantine-NavLink-root {
  border-radius: 6px;
  margin: 2px 0;
}

.mantine-NavLink-root[data-active] {
  background-color: rgba(62, 207, 142, 0.1);
  border-left: 3px solid var(--supabase-brand);
}

.mantine-NavLink-root[data-active] .mantine-NavLink-label {
  color: var(--supabase-brand);
  font-weight: 500;
}

.mantine-Card-root {
  border: 1px solid var(--supabase-border);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.mantine-Button-root[data-variant="filled"] {
  background-color: var(--supabase-brand);
}

.mantine-Button-root[data-variant="filled"]:hover {
  background-color: var(--supabase-brand-hover);
}

/* Table styling */
.mantine-Table-root {
  border: 1px solid var(--supabase-border);
  border-radius: 8px;
  overflow: hidden;
}

.mantine-Table-thead {
  background-color: var(--supabase-bg-surface);
}

.mantine-Table-tr:hover {
  background-color: var(--supabase-bg-surface-hover);
}

/* Input styling */
.mantine-TextInput-input,
.mantine-Select-input,
.mantine-Textarea-input {
  border-color: var(--supabase-border);
}

.mantine-TextInput-input:focus,
.mantine-Select-input:focus,
.mantine-Textarea-input:focus {
  border-color: var(--supabase-brand);
}
```

## Build Configuration

### Vite Configuration

```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

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
          'vendor-recharts': ['recharts'],
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:54321',
        changeOrigin: true,
      },
      '/auth': {
        target: 'http://localhost:54321',
        changeOrigin: true,
      },
      '/storage': {
        target: 'http://localhost:54321',
        changeOrigin: true,
      },
      '/rest': {
        target: 'http://localhost:54321',
        changeOrigin: true,
      },
      '/realtime': {
        target: 'http://localhost:54321',
        changeOrigin: true,
        ws: true,
      },
      '/functions': {
        target: 'http://localhost:54321',
        changeOrigin: true,
      },
    },
  },
})
```

## Implementation Plan

### Phase 1: Core Infrastructure
1. Create directory structure
2. Set up Supabase theme CSS
3. Create base API client
4. Implement app store
5. Create layout components (Sidebar, Header, PageContainer)

### Phase 2: Dashboard & Auth
1. Implement Dashboard home page with live stats
2. Create Users page with data table
3. Add user CRUD modals
4. Implement search and pagination

### Phase 3: Storage
1. Implement Storage page layout
2. Create bucket sidebar
3. Build file browser with upload
4. Add file preview and actions

### Phase 4: Database
1. Implement Table Editor page
2. Build data grid with inline editing
3. Create SQL Editor with Monaco
4. Add query history

### Phase 5: Realtime & Functions
1. Implement Realtime page with live stats
2. Create Functions page
3. Add function deployment UI

### Phase 6: Polish
1. Add loading states
2. Implement error handling
3. Add empty states
4. Performance optimization
5. Build and test production bundle

## Testing Requirements

### Manual Testing Checklist

- [ ] Dashboard loads with correct stats from backend
- [ ] Users page lists actual users from database
- [ ] Create user works and appears in list
- [ ] Edit user updates database
- [ ] Delete user removes from database
- [ ] Storage buckets list from backend
- [ ] File upload stores to actual bucket
- [ ] File download works
- [ ] File delete removes from storage
- [ ] Table editor shows real tables
- [ ] SQL editor executes queries
- [ ] Query results display correctly
- [ ] Realtime shows connection stats
- [ ] Functions list from backend
- [ ] All API errors display properly
- [ ] Production build serves correctly

## Security Considerations

1. **API Key Handling**: Service key stored in localStorage, never exposed in URLs
2. **CORS**: Already configured in backend
3. **Input Validation**: Client-side validation before API calls
4. **Error Messages**: Don't expose sensitive info in error displays
5. **File Upload**: Respect bucket MIME type restrictions

## Appendix: Supabase Dashboard Screenshots Reference

The implementation should visually match these Supabase Dashboard characteristics:

1. **Clean white background** with subtle gray card surfaces
2. **Green accent color** (#3ECF8E) for primary actions and active states
3. **Left sidebar** with icon + text navigation
4. **Rounded corners** (8px default)
5. **Subtle shadows** (0 1px 3px rgba(0,0,0,0.04))
6. **14px base font size** with consistent spacing
7. **Tables with header backgrounds** in light gray
8. **Modal dialogs** with clean white backgrounds
9. **Form inputs** with light borders darkening on focus
10. **Status badges** using semantic colors
