# 0385: Localbase Dashboard - Supabase 100% Visual Match (Light Theme)

## Overview

This specification extends `0384_dashboard.md` with detailed enhancements to achieve a 100% visual match with the Supabase Dashboard (light theme). It documents the exact design tokens, component styling, and implementation details derived from research on the Supabase UI Library and design system.

## Reference Sources

- [Supabase UI Library](https://supabase.com/ui) - Official component library built on shadcn/ui
- [Supabase Design System](https://supabase-design-system.vercel.app/) - Component showcase
- [Mobbin Colors](https://mobbin.com/colors/brand/supabase) - Brand color palette analysis
- [How Design Works at Supabase](https://supabase.com/blog/how-design-works-at-supabase) - Design philosophy

## Supabase Design System Deep Dive

### Core Principles

1. **Radix UI Foundation**: All components built on Radix UI primitives
2. **Tailwind CSS Styling**: Utility-first approach with custom configuration
3. **12-Step Color Scales**: Based on Radix Colors for accessibility
4. **shadcn/ui Compatibility**: 100% compatible with shadcn/ui component registry

### Exact Color Palette (Light Theme)

From official Supabase sources:

```css
:root {
  /* Brand Colors */
  --supabase-brand-green: #3ECF8E;        /* Jungle Green - Primary */
  --supabase-brand-green-hover: #24B47E;  /* Darker variant */
  --supabase-brand-green-light: rgba(62, 207, 142, 0.1); /* Light bg */
  --supabase-brand-green-muted: rgba(62, 207, 142, 0.15); /* Badge bg */

  /* Core Backgrounds */
  --supabase-bg-default: #FFFFFF;         /* Athens Gray White */
  --supabase-bg-surface: #F8F9FA;         /* Light surface */
  --supabase-bg-surface-alt: #F4F4F5;     /* Alternate surface */
  --supabase-bg-surface-hover: #F1F3F5;   /* Hover state */
  --supabase-bg-sidebar: #FAFAFA;         /* Sidebar background */
  --supabase-bg-overlay: rgba(0, 0, 0, 0.4); /* Modal overlay */
  --supabase-bg-canvas: #F8F9FA;          /* Main content area */

  /* Text Colors (12-step scale) */
  --supabase-text-1200: #11181C;          /* Bunker - Headlines */
  --supabase-text-1100: #1C1C1C;          /* Primary text */
  --supabase-text-1000: #3F3F46;          /* Strong secondary */
  --supabase-text-900: #52525B;           /* Secondary text */
  --supabase-text-800: #666666;           /* Muted text */
  --supabase-text-700: #71717A;           /* Light muted */
  --supabase-text-600: #888888;           /* Placeholder */
  --supabase-text-500: #A1A1AA;           /* Disabled */
  --supabase-text-400: #AAAAAA;           /* Very light */

  /* Border Colors */
  --supabase-border-default: #E6E8EB;     /* Default */
  --supabase-border-muted: #EAEAEA;       /* Subtle */
  --supabase-border-strong: #C9CDD3;      /* Emphasized */
  --supabase-border-focus: #3ECF8E;       /* Focus ring */

  /* Semantic Colors */
  --supabase-success: #3ECF8E;            /* Same as brand */
  --supabase-success-text: #059669;       /* Dark green text */
  --supabase-warning: #F5A623;            /* Amber */
  --supabase-warning-bg: rgba(245, 166, 35, 0.15);
  --supabase-warning-text: #D97706;       /* Dark amber */
  --supabase-error: #EF4444;              /* Red */
  --supabase-error-bg: rgba(239, 68, 68, 0.15);
  --supabase-error-text: #DC2626;         /* Dark red */
  --supabase-info: #3B82F6;               /* Blue */
  --supabase-info-bg: rgba(59, 130, 246, 0.15);
  --supabase-info-text: #2563EB;          /* Dark blue */

  /* Shadows */
  --supabase-shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.04);
  --supabase-shadow-default: 0 1px 3px rgba(0, 0, 0, 0.04);
  --supabase-shadow-md: 0 4px 6px rgba(0, 0, 0, 0.04);
  --supabase-shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.04);
  --supabase-shadow-dropdown: 0 4px 20px rgba(0, 0, 0, 0.08);

  /* Radius */
  --supabase-radius-sm: 4px;
  --supabase-radius-default: 6px;
  --supabase-radius-lg: 8px;
  --supabase-radius-xl: 12px;
  --supabase-radius-full: 9999px;
}
```

### Typography

```css
/* Font Families */
--font-sans: -apple-system, BlinkMacSystemFont, 'Segoe UI',
             Roboto, 'Helvetica Neue', Arial, sans-serif;
--font-mono: 'Source Code Pro', 'Menlo', 'Monaco',
             'Consolas', 'Liberation Mono', monospace;

/* Font Sizes */
--text-2xs: 0.625rem;    /* 10px - micro labels */
--text-xs: 0.75rem;      /* 12px - badges, hints */
--text-sm: 0.875rem;     /* 14px - body, labels */
--text-base: 1rem;       /* 16px - larger body */
--text-lg: 1.125rem;     /* 18px - subheadings */
--text-xl: 1.25rem;      /* 20px - small headings */
--text-2xl: 1.5rem;      /* 24px - page titles */
--text-3xl: 1.875rem;    /* 30px - hero text */

/* Font Weights */
--font-normal: 400;
--font-medium: 500;
--font-semibold: 600;
--font-bold: 700;

/* Line Heights */
--leading-tight: 1.25;
--leading-normal: 1.5;
--leading-relaxed: 1.625;
```

### Spacing System

```css
/* Based on 4px grid */
--space-0: 0;
--space-1: 0.25rem;    /* 4px */
--space-2: 0.5rem;     /* 8px */
--space-3: 0.75rem;    /* 12px */
--space-4: 1rem;       /* 16px */
--space-5: 1.25rem;    /* 20px */
--space-6: 1.5rem;     /* 24px */
--space-8: 2rem;       /* 32px */
--space-10: 2.5rem;    /* 40px */
--space-12: 3rem;      /* 48px */
--space-16: 4rem;      /* 64px */
```

## Component Styling Specifications

### 1. Sidebar Navigation

```css
/* Sidebar Container */
.sidebar {
  width: 250px;
  background: var(--supabase-bg-sidebar);
  border-right: 1px solid var(--supabase-border-default);
  padding: 16px 0;
}

.sidebar-collapsed {
  width: 64px;
}

/* Logo Section */
.sidebar-logo {
  padding: 0 16px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.sidebar-logo-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: linear-gradient(135deg, #3ECF8E 0%, #24B47E 100%);
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Navigation Items */
.nav-item {
  margin: 2px 8px;
  padding: 8px 12px;
  border-radius: 6px;
  color: var(--supabase-text-800);
  font-size: 14px;
  font-weight: 400;
  display: flex;
  align-items: center;
  gap: 12px;
  transition: background-color 0.15s ease;
}

.nav-item:hover {
  background: var(--supabase-bg-surface-hover);
}

.nav-item-active {
  background: var(--supabase-brand-green-light);
  color: var(--supabase-brand-green);
  font-weight: 500;
  border-left: 3px solid var(--supabase-brand-green);
  margin-left: 5px;
  padding-left: 9px;
}

.nav-item-active svg {
  color: var(--supabase-brand-green);
}
```

### 2. Cards & Surfaces

```css
/* Base Card */
.card {
  background: var(--supabase-bg-default);
  border: 1px solid var(--supabase-border-default);
  border-radius: var(--supabase-radius-lg);
  box-shadow: var(--supabase-shadow-default);
}

/* Stat Card (Dashboard) */
.stat-card {
  padding: 20px;
  background: var(--supabase-bg-default);
  border: 1px solid var(--supabase-border-default);
  border-radius: var(--supabase-radius-lg);
  transition: border-color 0.15s ease;
}

.stat-card:hover {
  border-color: var(--supabase-border-strong);
}

.stat-card-value {
  font-size: 32px;
  font-weight: 600;
  color: var(--supabase-text-1100);
  line-height: 1;
}

.stat-card-label {
  font-size: 14px;
  color: var(--supabase-text-800);
  margin-top: 8px;
}
```

### 3. Buttons

```css
/* Primary Button (Green) */
.btn-primary {
  background: var(--supabase-brand-green);
  color: white;
  padding: 6px 12px;
  border-radius: var(--supabase-radius-default);
  font-size: 14px;
  font-weight: 500;
  border: none;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.btn-primary:hover {
  background: var(--supabase-brand-green-hover);
}

/* Secondary Button (Outline) */
.btn-outline {
  background: transparent;
  color: var(--supabase-text-1100);
  padding: 6px 12px;
  border-radius: var(--supabase-radius-default);
  border: 1px solid var(--supabase-border-default);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-outline:hover {
  background: var(--supabase-bg-surface);
  border-color: var(--supabase-border-strong);
}

/* Danger Button */
.btn-danger {
  background: var(--supabase-error);
  color: white;
  padding: 6px 12px;
  border-radius: var(--supabase-radius-default);
  font-size: 14px;
  font-weight: 500;
  border: none;
}

/* Ghost Button */
.btn-ghost {
  background: transparent;
  color: var(--supabase-text-800);
  padding: 6px 12px;
  border: none;
  border-radius: var(--supabase-radius-default);
}

.btn-ghost:hover {
  background: var(--supabase-bg-surface);
}

/* Icon Button (Action Icon) */
.btn-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--supabase-radius-default);
  background: transparent;
  border: none;
  color: var(--supabase-text-800);
}

.btn-icon:hover {
  background: var(--supabase-bg-surface);
}
```

### 4. Tables

```css
/* Table Container */
.table-container {
  border: 1px solid var(--supabase-border-default);
  border-radius: var(--supabase-radius-lg);
  overflow: hidden;
}

/* Table Header */
.table-header {
  background: var(--supabase-bg-surface);
  border-bottom: 1px solid var(--supabase-border-default);
}

.table-header-cell {
  padding: 12px 16px;
  font-size: 12px;
  font-weight: 500;
  color: var(--supabase-text-800);
  text-transform: uppercase;
  letter-spacing: 0.025em;
}

/* Table Body */
.table-row {
  border-bottom: 1px solid var(--supabase-border-muted);
  transition: background-color 0.15s ease;
}

.table-row:last-child {
  border-bottom: none;
}

.table-row:hover {
  background: var(--supabase-bg-surface-hover);
}

.table-cell {
  padding: 12px 16px;
  font-size: 14px;
  color: var(--supabase-text-1100);
}
```

### 5. Form Inputs

```css
/* Text Input */
.input {
  width: 100%;
  padding: 8px 12px;
  font-size: 14px;
  border: 1px solid var(--supabase-border-default);
  border-radius: var(--supabase-radius-default);
  background: var(--supabase-bg-default);
  color: var(--supabase-text-1100);
  transition: all 0.15s ease;
}

.input:focus {
  outline: none;
  border-color: var(--supabase-brand-green);
  box-shadow: 0 0 0 1px var(--supabase-brand-green);
}

.input::placeholder {
  color: var(--supabase-text-600);
}

/* Input Label */
.input-label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--supabase-text-1100);
  margin-bottom: 6px;
}

/* Input Description */
.input-description {
  font-size: 12px;
  color: var(--supabase-text-800);
  margin-top: 4px;
}

/* Select */
.select {
  appearance: none;
  padding-right: 32px;
  background-image: url("data:image/svg+xml...");
  background-repeat: no-repeat;
  background-position: right 8px center;
}
```

### 6. Badges

```css
/* Base Badge */
.badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 500;
  border-radius: var(--supabase-radius-default);
  text-transform: none;
}

/* Success Badge */
.badge-success {
  background: var(--supabase-brand-green-muted);
  color: var(--supabase-success-text);
}

/* Warning Badge */
.badge-warning {
  background: var(--supabase-warning-bg);
  color: var(--supabase-warning-text);
}

/* Error Badge */
.badge-error {
  background: var(--supabase-error-bg);
  color: var(--supabase-error-text);
}

/* Info Badge */
.badge-info {
  background: var(--supabase-info-bg);
  color: var(--supabase-info-text);
}

/* Neutral Badge */
.badge-neutral {
  background: var(--supabase-bg-surface);
  color: var(--supabase-text-800);
}
```

### 7. Modals

```css
/* Modal Overlay */
.modal-overlay {
  background: var(--supabase-bg-overlay);
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Modal Content */
.modal-content {
  background: var(--supabase-bg-default);
  border-radius: var(--supabase-radius-lg);
  box-shadow: var(--supabase-shadow-lg);
  max-width: 480px;
  width: 100%;
  max-height: 85vh;
  overflow: auto;
}

/* Modal Header */
.modal-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--supabase-border-default);
}

.modal-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--supabase-text-1100);
}

/* Modal Body */
.modal-body {
  padding: 20px;
}

/* Modal Footer */
.modal-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--supabase-border-default);
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
```

### 8. Dropdowns/Menus

```css
/* Menu Container */
.menu {
  background: var(--supabase-bg-default);
  border: 1px solid var(--supabase-border-default);
  border-radius: var(--supabase-radius-lg);
  box-shadow: var(--supabase-shadow-dropdown);
  padding: 4px;
  min-width: 180px;
}

/* Menu Item */
.menu-item {
  padding: 8px 12px;
  font-size: 14px;
  color: var(--supabase-text-1100);
  border-radius: var(--supabase-radius-default);
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.menu-item:hover {
  background: var(--supabase-bg-surface);
}

.menu-item-danger {
  color: var(--supabase-error);
}

.menu-item-danger:hover {
  background: var(--supabase-error-bg);
}

/* Menu Divider */
.menu-divider {
  height: 1px;
  background: var(--supabase-border-muted);
  margin: 4px 0;
}
```

### 9. Tabs

```css
/* Tabs Container */
.tabs {
  border-bottom: 1px solid var(--supabase-border-default);
}

/* Tab Item */
.tab {
  padding: 12px 16px;
  font-size: 14px;
  font-weight: 500;
  color: var(--supabase-text-800);
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}

.tab:hover {
  color: var(--supabase-text-1100);
}

.tab-active {
  color: var(--supabase-brand-green);
  border-bottom-color: var(--supabase-brand-green);
}
```

### 10. Notifications/Toasts

```css
/* Toast Container */
.toast {
  background: var(--supabase-bg-default);
  border: 1px solid var(--supabase-border-default);
  border-radius: var(--supabase-radius-lg);
  box-shadow: var(--supabase-shadow-lg);
  padding: 16px;
  min-width: 300px;
  max-width: 420px;
}

/* Toast Variants */
.toast-success {
  border-left: 4px solid var(--supabase-success);
}

.toast-error {
  border-left: 4px solid var(--supabase-error);
}

.toast-warning {
  border-left: 4px solid var(--supabase-warning);
}
```

## Frontend Organization (from Localflare Pattern)

### Directory Structure

```
app/frontend/
├── src/
│   ├── api/                 # API client modules
│   │   ├── client.ts        # Base HTTP client with JWT auth
│   │   ├── auth.ts          # Auth API methods
│   │   ├── storage.ts       # Storage API methods
│   │   ├── database.ts      # Database API methods
│   │   ├── functions.ts     # Functions API methods
│   │   ├── realtime.ts      # Realtime WebSocket client
│   │   ├── dashboard.ts     # Dashboard API methods
│   │   └── index.ts         # Re-exports
│   ├── components/
│   │   ├── common/          # Shared UI components
│   │   │   ├── DataTable.tsx
│   │   │   ├── EmptyState.tsx
│   │   │   ├── ConfirmModal.tsx
│   │   │   ├── StatusBadge.tsx
│   │   │   └── index.ts
│   │   ├── forms/           # Form components
│   │   │   └── SearchInput.tsx
│   │   └── layout/          # Layout components
│   │       ├── Sidebar.tsx
│   │       └── PageContainer.tsx
│   ├── hooks/               # Custom React hooks
│   │   └── useApi.ts
│   ├── pages/               # Page components
│   │   ├── Dashboard.tsx
│   │   ├── ApiDocs.tsx
│   │   ├── auth/
│   │   │   └── Users.tsx
│   │   ├── database/
│   │   │   ├── TableEditor.tsx
│   │   │   └── SQLEditor.tsx
│   │   ├── storage/
│   │   │   └── Storage.tsx
│   │   ├── functions/
│   │   │   └── Functions.tsx
│   │   ├── realtime/
│   │   │   └── Realtime.tsx
│   │   └── settings/
│   │       └── Settings.tsx
│   ├── stores/              # Zustand state management
│   │   └── appStore.ts
│   ├── styles/              # Global styles
│   │   ├── index.css
│   │   └── supabase-theme.css
│   ├── types/               # TypeScript definitions
│   │   └── index.ts
│   ├── App.tsx
│   └── main.tsx
├── index.html
├── package.json
├── vite.config.ts
├── tsconfig.json
└── tsconfig.node.json
```

### Asset Embedding (Go)

```go
// assets/assets.go
package assets

import "embed"

//go:embed static/*
var StaticFS embed.FS
```

### Build Configuration (Vite)

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
          'vendor-mantine': ['@mantine/core', '@mantine/hooks', '@mantine/notifications', '@mantine/dropzone'],
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
      '/functions': { target: 'http://localhost:54321', changeOrigin: true },
    },
  },
})
```

## Implementation Status

### Completed Features

1. **Dashboard Page**
   - Stats cards with icons (Users, Tables, Buckets, Functions)
   - Service health status display
   - Quick links section

2. **Authentication/Users Page**
   - User listing with DataTable
   - Create/Edit/Delete user modals
   - Search functionality
   - Provider and verification badges

3. **Storage Page**
   - Bucket sidebar with create/select
   - File browser with breadcrumb navigation
   - File upload with drag-and-drop (Mantine Dropzone)
   - Download and URL copy functionality
   - Delete confirmation

4. **Table Editor**
   - Schema selector dropdown
   - Table listing with row counts
   - Column display with types and primary key indicators
   - Data grid with table rows
   - Create/Delete table modals

5. **SQL Editor**
   - Monaco editor integration
   - Query execution
   - Results display with columns and rows

6. **Realtime Page**
   - Connection stats display
   - Channel listing
   - WebSocket integration

7. **Functions Page**
   - Function listing
   - Status badges
   - Create/Deploy functionality

8. **Settings Page**
   - API key display
   - Project configuration

### Styling Enhancements Required

The following CSS enhancements ensure 100% Supabase visual match:

1. **Active NavLink styling** - Left border accent
2. **Card hover states** - Border color transition
3. **Badge color variants** - Success, warning, error, info
4. **Table header styling** - Uppercase, letter-spacing
5. **Input focus states** - Green border + shadow
6. **Modal header/footer borders** - Proper separation
7. **Menu dropdown shadows** - Deeper shadow
8. **Scrollbar styling** - Custom webkit styles

## Backend API Integration

All pages integrate with real backend APIs (no mocking):

| Page | Endpoints Used |
|------|----------------|
| Dashboard | `GET /api/dashboard/stats`, `GET /api/dashboard/health` |
| Users | `GET /auth/v1/admin/users`, `POST /auth/v1/admin/users`, `PUT /auth/v1/admin/users/{id}`, `DELETE /auth/v1/admin/users/{id}` |
| Storage | `GET /storage/v1/bucket`, `POST /storage/v1/bucket`, `POST /storage/v1/object/list/{bucket}`, `POST /storage/v1/object/{bucket}/{path}`, `GET /storage/v1/object/{bucket}/{path}`, `DELETE /storage/v1/object/{bucket}/{path}` |
| Table Editor | `GET /api/database/schemas`, `GET /api/database/tables`, `GET /api/database/tables/{schema}/{name}/columns`, `GET /rest/v1/{table}`, `POST /api/database/tables`, `DELETE /api/database/tables/{schema}/{name}` |
| SQL Editor | `POST /api/database/query` |
| Realtime | `GET /api/realtime/channels`, `GET /api/realtime/stats`, `WS /realtime/v1/websocket` |
| Functions | `GET /api/functions`, `POST /api/functions`, `PUT /api/functions/{id}`, `DELETE /api/functions/{id}`, `POST /api/functions/{id}/deploy` |

## Security Notes

1. **API Key Storage**: Service key stored in localStorage, sent as `apikey` header
2. **Authorization**: Bearer token authentication for all API calls
3. **Service Role**: Dashboard requires `service_role` JWT for admin operations
4. **Input Validation**: Client-side validation before API calls
5. **Error Handling**: API errors displayed via Mantine notifications

## Testing Checklist

- [ ] Dashboard stats load from `/api/dashboard/stats`
- [ ] Service health updates from `/api/dashboard/health`
- [ ] User list populates from `/auth/v1/admin/users`
- [ ] Create user persists to database
- [ ] Delete user removes from database
- [ ] Bucket list shows from `/storage/v1/bucket`
- [ ] File upload stores in bucket
- [ ] File download retrieves content
- [ ] Table list shows from `/api/database/tables`
- [ ] SQL queries execute successfully
- [ ] Query results display correctly
- [ ] Realtime stats update
- [ ] Functions list and deploy work
- [ ] All modals open/close properly
- [ ] Loading states display during fetches
- [ ] Empty states show when no data
- [ ] Error notifications appear on failures
- [ ] Sidebar collapse/expand works
- [ ] Navigation between pages works
- [ ] Production build serves correctly
