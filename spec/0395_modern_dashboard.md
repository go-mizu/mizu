# Modern Dashboard Enhancement Specification

## Overview

This document outlines the comprehensive modernization of the Localbase dashboard to align with the latest Supabase Studio features (2024-2025). Based on analysis of the current implementation and the latest Supabase updates.

## Current State Analysis

### What's Implemented

| Feature | Status | Notes |
|---------|--------|-------|
| Dashboard | Basic | 4 stat cards, health status, quick links |
| Table Editor | Good | CRUD, inline editing, sorting, schema selector |
| SQL Editor | Basic | Monaco editor, run queries, save queries |
| Database > Policies | Good | RLS toggle, create/delete policies with templates |
| Database > Indexes | Good | Create indexes with type selection |
| Database > Views | Good | Regular and materialized views |
| Database > Triggers | Good | BEFORE/AFTER triggers, event selection |
| Database > Roles | Good | Create/delete roles, attribute toggles |
| Authentication > Users | Good | User CRUD, search, provider badges |
| Storage | Good | Bucket management, file browser, uploads |
| Realtime | Basic | Connection count, message inspector |
| Edge Functions | Basic | Create/delete, deploy, invoke |
| Logs | Good | Multi-filter, severity levels, export |
| API Docs | Basic | Endpoint list with method badges |
| Settings | Good | General, API Keys, Database tabs |

### Missing Features (Gap Analysis)

Based on the latest Supabase Studio (2024-2025):

#### Critical Missing Features

1. **SQL Editor Enhancements**
   - [ ] Tabs support (multiple query tabs)
   - [ ] CMD+K AI assistant integration placeholder
   - [ ] Query organization: Shared, Favorites, Private, Community sections
   - [ ] Results/Explain/Chart tabs
   - [ ] Source database selector
   - [ ] Role selector dropdown
   - [ ] Templates library
   - [ ] Quickstarts section
   - [ ] Running queries view
   - [ ] Query explain plan visualization

2. **Table Editor Enhancements**
   - [ ] Tabs support (multiple tables open)
   - [ ] Column filtering
   - [ ] Advanced search/filter UI
   - [ ] Foreign key visualization
   - [ ] Relationship navigator
   - [ ] Column reordering
   - [ ] Bulk operations panel

3. **New Pages Required**
   - [ ] Project Overview (replace basic Dashboard)
   - [ ] Advisors (Security + Performance)
   - [ ] Observability (Metrics + Monitoring)
   - [ ] Integrations (Cron Jobs, Queues, Wrappers)

4. **Header Improvements**
   - [ ] Global search (CMD+K)
   - [ ] Connect button with connection dialog
   - [ ] Environment badge (Production/Staging/Local)
   - [ ] Organization/Project selector
   - [ ] Feedback button
   - [ ] Help menu

5. **Sidebar Improvements**
   - [ ] Collapsible sections
   - [ ] Keyboard shortcuts hints
   - [ ] Better nested navigation
   - [ ] Bottom action bar

6. **Realtime Enhancements**
   - [ ] Realtime Settings page
   - [ ] Channel restrictions configuration
   - [ ] Database connection pool settings
   - [ ] Broadcast/Presence monitoring

7. **Authentication Enhancements**
   - [ ] Auth Providers configuration UI
   - [ ] Email templates
   - [ ] URL configuration
   - [ ] Auth policies
   - [ ] Rate limiting settings

8. **Edge Functions Enhancements**
   - [ ] Inline code editor
   - [ ] Function tester (Postman-like)
   - [ ] Logs integration
   - [ ] Secrets management

9. **Storage Enhancements**
   - [ ] Analytics integration
   - [ ] Transformation settings
   - [ ] CDN configuration

## Implementation Status

### Completed Enhancements

#### SQL Editor (Fully Modernized)
- [x] Tabs support (multiple query tabs with close/new)
- [x] Query organization: Shared, Favorites, Private, Community sections
- [x] Results/Explain/Chart tabs
- [x] Source database selector
- [x] Role selector dropdown (postgres, anon, authenticated, service_role)
- [x] Templates library (Common, Auth, Database, RLS categories)
- [x] Quickstarts section
- [x] Running queries button
- [x] CMD+K placeholder in editor
- [x] Keyboard shortcut hints (⌘↵ for Run)

#### Project Overview (New Page)
- [x] Stats cards (Users, Tables, Storage Buckets, Edge Functions)
- [x] Service health status
- [x] Recent activity feed
- [x] Quick actions grid
- [x] Getting started checklist
- [x] Resources section

#### Advisors Page (New)
- [x] Security tab with score and issues
- [x] Performance tab with metrics
- [x] Issue accordion with severity levels
- [x] SQL fix suggestions with copy/open in editor
- [x] Quick stats (Active Connections, Avg Query Time, Cache Hit Ratio)

#### Integrations Page (New)
- [x] Cron Jobs tab (pg_cron management)
- [x] Queues tab (pgmq management)
- [x] Extensions tab (PostgreSQL extensions)
- [x] Create/delete/toggle jobs
- [x] Common cron schedules reference

#### Modern Header (New)
- [x] Organization/Project breadcrumb
- [x] Environment badge (Local)
- [x] Connect button with connection dialog
- [x] Connection strings (Direct, Pooler)
- [x] Database credentials display
- [x] API keys (anon, service_role)
- [x] Search, Help, Settings, User icons

#### Enhanced Sidebar
- [x] Collapsible Database section with children
- [x] Tools section (Advisors, Logs, API Docs, Integrations)
- [x] "New" badges on new features
- [x] Keyboard-accessible navigation
- [x] Collapse/expand toggle

#### Realtime Page (Enhanced)
- [x] Inspector tab (channels + messages)
- [x] Channels tab (table view)
- [x] Settings tab with connection limits
- [x] Feature toggles (Presence, Broadcast, Postgres Changes)
- [x] Channel restrictions management
- [x] Messages/sec metric

#### Authentication Page (Enhanced)
- [x] Users tab (existing + improved)
- [x] Providers tab (Email, Phone, Social)
- [x] Social providers: Google, GitHub, Apple, Twitter, Discord
- [x] Provider configuration modal
- [x] Settings tab (URL config, Security, Features)
- [x] Active providers summary

## Implementation Plan

### Phase 1: SQL Editor Modernization (High Priority)

Transform the SQL Editor to match modern Supabase Studio:

```
┌──────────────────────────────────────────────────────────────────────────┐
│  SQL Editor                                                              │
├─────────────────────┬────────────────────────────────────────────────────┤
│ [Search queries...] │  [Tab 1] [Tab 2] [+ New]                           │
├─────────────────────┤────────────────────────────────────────────────────│
│ > SHARED            │  1 | Hit CMD+K to generate query or just start    │
│ > FAVORITES         │  2 | typing                                        │
│ v PRIVATE           │  3 |                                               │
│   dummy_query_1     │  ...                                               │
│   dummy_query_2     │                                                    │
│ > COMMUNITY         │                                                    │
│   Templates         │                                                    │
│   Quickstarts       │                                                    │
│                     │                                                    │
│                     ├────────────────────────────────────────────────────│
│ [Running queries]   │ Results | Explain | Chart    [Source] [Role] [Run]│
│                     │ Click Run to execute your query.                   │
└─────────────────────┴────────────────────────────────────────────────────┘
```

#### 1.1 Tab System
- Multiple query tabs with close button
- New tab button
- Tab persistence across sessions
- Unsaved changes indicator

#### 1.2 Query Organization Sidebar
- **Shared**: Queries shared with team
- **Favorites**: Starred queries
- **Private**: Personal saved queries
- **Community**: Templates and Quickstarts

#### 1.3 Results Panel
- **Results Tab**: Query results in data grid
- **Explain Tab**: Query execution plan
- **Chart Tab**: Basic visualization

#### 1.4 Toolbar
- Source database selector
- Role selector (postgres, anon, etc.)
- Run button with keyboard shortcut hint

### Phase 2: Modern Header & Navigation

#### 2.1 Global Header
```
┌──────────────────────────────────────────────────────────────────────────┐
│ [Logo] Org > Project [ENV] │ [Connect] │ [Search ⌘K] │ [?] │ [Settings] │
└──────────────────────────────────────────────────────────────────────────┘
```

- Organization and project breadcrumb
- Environment badge (Production/Local)
- Connect button with connection string dialog
- Global search
- Help/feedback links

#### 2.2 Enhanced Sidebar
```
┌─────────────────────┐
│ Project Overview    │
│ Table Editor        │
│ SQL Editor          │
│ Database ─────────┐ │
│   Tables          │ │
│   Views           │ │
│   Functions       │ │
│   Triggers        │ │
│   Roles           │ │
│   Policies        │ │
│   Indexes         │ │
│   Extensions      │ │
│ Authentication      │
│ Storage             │
│ Edge Functions      │
│ Realtime            │
│ ─────────────────── │
│ Advisors            │
│ Observability       │
│ Logs                │
│ API Docs            │
│ Integrations        │
│ ─────────────────── │
│ Project Settings    │
└─────────────────────┘
```

### Phase 3: New Pages

#### 3.1 Project Overview
Replace basic Dashboard with comprehensive overview:
- Project health status
- Quick stats (Users, Tables, Storage, Functions)
- Recent activity
- Quick actions
- Getting started guides for new projects

#### 3.2 Advisors Page
Two sections:
- **Security**: RLS warnings, exposed tables, auth issues
- **Performance**: Slow queries, missing indexes, table bloat

#### 3.3 Observability Page
- Database metrics (connections, queries/sec)
- API metrics (requests, latency)
- Storage metrics (size, bandwidth)
- Custom reports builder

#### 3.4 Integrations Page
- **Cron Jobs**: pg_cron management
- **Queues**: pg_queue management
- **Wrappers**: Foreign data wrappers
- **Extensions**: PostgreSQL extensions manager

### Phase 4: Page Enhancements

#### 4.1 Table Editor Enhancements
- Tab support for multiple tables
- Column filtering UI
- Advanced filter builder
- Foreign key navigation
- Relationship diagram view

#### 4.2 Realtime Settings
- Channel restrictions
- Connection pool configuration
- Authorization settings
- Broadcast/Presence settings

#### 4.3 Authentication Enhancements
- Providers tab (Email, Phone, OAuth)
- Email Templates tab
- URL Configuration tab
- Auth Hooks tab

#### 4.4 Edge Functions Enhancements
- Code tab with Monaco editor
- Tester tab (HTTP request builder)
- Logs tab (function-specific logs)
- Secrets management

#### 4.5 API Docs Enhancements
- Code snippets for multiple languages (JS, Python, Go, cURL)
- Interactive API explorer
- Authentication examples
- Real-time subscription examples

### Phase 5: UX Improvements

#### 5.1 Global Features
- Keyboard navigation (CMD+K for search, shortcuts)
- Toast notifications with undo
- Breadcrumb navigation
- Loading skeletons
- Error boundaries with retry

#### 5.2 Responsive Design
- Mobile-friendly sidebar (hamburger menu)
- Responsive data tables
- Touch-friendly controls

#### 5.3 Accessibility
- ARIA labels
- Focus management
- Screen reader support
- High contrast mode support

## Technical Specifications

### Component Architecture

```
src/
├── components/
│   ├── common/
│   │   ├── DataTable.tsx
│   │   ├── EmptyState.tsx
│   │   ├── ConfirmModal.tsx
│   │   ├── TabPanel.tsx (NEW)
│   │   ├── CommandPalette.tsx (NEW)
│   │   └── ConnectionDialog.tsx (NEW)
│   ├── layout/
│   │   ├── Sidebar.tsx (ENHANCE)
│   │   ├── Header.tsx (NEW)
│   │   ├── PageContainer.tsx
│   │   └── BreadcrumbNav.tsx (NEW)
│   └── editors/
│       ├── SqlEditor.tsx (NEW - Monaco wrapper)
│       ├── CodeEditor.tsx (NEW)
│       └── QueryTabs.tsx (NEW)
├── pages/
│   ├── project-overview/
│   │   └── ProjectOverview.tsx (NEW)
│   ├── database/
│   │   ├── SQLEditor.tsx (ENHANCE)
│   │   ├── TableEditor.tsx (ENHANCE)
│   │   ├── Extensions.tsx (NEW)
│   │   └── ...
│   ├── advisors/
│   │   ├── Advisors.tsx (NEW)
│   │   ├── SecurityAdvisor.tsx (NEW)
│   │   └── PerformanceAdvisor.tsx (NEW)
│   ├── observability/
│   │   └── Observability.tsx (NEW)
│   ├── integrations/
│   │   ├── Integrations.tsx (NEW)
│   │   ├── CronJobs.tsx (NEW)
│   │   └── Queues.tsx (NEW)
│   └── ...
└── stores/
    ├── appStore.ts (ENHANCE)
    ├── sqlEditorStore.ts (NEW)
    └── tabStore.ts (NEW)
```

### State Management

New Zustand stores needed:

```typescript
// sqlEditorStore.ts
interface SqlEditorStore {
  tabs: QueryTab[];
  activeTabId: string | null;
  sharedQueries: SavedQuery[];
  favoriteQueries: SavedQuery[];
  privateQueries: SavedQuery[];
  templates: QueryTemplate[];

  addTab: () => void;
  closeTab: (id: string) => void;
  setActiveTab: (id: string) => void;
  updateTabContent: (id: string, content: string) => void;
  saveQuery: (query: SavedQuery) => void;
  toggleFavorite: (id: string) => void;
}

// tabStore.ts (for Table Editor)
interface TabStore {
  openTables: TableTab[];
  activeTableId: string | null;

  openTable: (schema: string, table: string) => void;
  closeTable: (id: string) => void;
  setActiveTable: (id: string) => void;
}
```

### API Endpoints Required

New backend endpoints needed:

```
# Advisors
GET  /api/advisors/security     - Security issues
GET  /api/advisors/performance  - Performance issues

# Observability
GET  /api/metrics/database      - Database metrics
GET  /api/metrics/api           - API metrics
GET  /api/metrics/storage       - Storage metrics

# Integrations
GET  /api/integrations/cron     - List cron jobs
POST /api/integrations/cron     - Create cron job
GET  /api/integrations/queues   - List queues
GET  /api/extensions            - List extensions
POST /api/extensions/:name      - Enable extension

# Query explain
POST /api/sql/explain           - Get query plan

# Auth providers
GET  /api/auth/providers        - List providers
PUT  /api/auth/providers/:id    - Configure provider

# Realtime settings
GET  /api/realtime/settings     - Get settings
PUT  /api/realtime/settings     - Update settings
```

## Design System Updates

### Colors (Supabase Brand)
```css
:root {
  /* Primary Brand */
  --brand-primary: #3ECF8E;
  --brand-secondary: #24B47E;

  /* Backgrounds */
  --bg-default: #FFFFFF;
  --bg-surface: #FAFAFA;
  --bg-overlay: #F4F4F5;

  /* Environment Badges */
  --env-production: #F97316;
  --env-staging: #8B5CF6;
  --env-local: #3ECF8E;

  /* Status */
  --status-success: #22C55E;
  --status-warning: #F59E0B;
  --status-error: #EF4444;
  --status-info: #3B82F6;
}
```

### Typography
- Headings: Inter/System font
- Code: JetBrains Mono / Monaco
- Body: System font stack

### Spacing
- 4px base unit
- Consistent padding: 16px (md), 24px (lg)
- Section gaps: 24px

## Migration Strategy

### Phase 1 (Immediate)
1. SQL Editor tabs and organization
2. Modern header with Connect button
3. Enhanced sidebar structure

### Phase 2 (Short-term)
1. Project Overview page
2. Advisors page
3. Table Editor tabs

### Phase 3 (Medium-term)
1. Observability page
2. Integrations page
3. Auth providers UI

### Phase 4 (Long-term)
1. Edge Functions code editor
2. Advanced API docs
3. Mobile responsive design

## Summary of Changes Made

### Files Created
- `src/components/layout/Header.tsx` - Modern header with Connect button
- `src/pages/project-overview/ProjectOverview.tsx` - New project overview page
- `src/pages/advisors/Advisors.tsx` - Security and performance advisors
- `src/pages/integrations/Integrations.tsx` - Cron jobs, queues, extensions

### Files Modified
- `src/App.tsx` - Added header, new routes
- `src/components/layout/Sidebar.tsx` - Modern navigation structure
- `src/components/layout/PageContainer.tsx` - Added noHeader prop
- `src/pages/database/SQLEditor.tsx` - Full modernization with tabs
- `src/pages/realtime/Realtime.tsx` - Added settings, channels tabs
- `src/pages/auth/Users.tsx` - Added providers and settings tabs

### Key Improvements Over Original
1. **SQL Editor**: From basic editor to full IDE-like experience with tabs, query organization, and results panel
2. **Navigation**: Added collapsible database section, tools section, and modern header
3. **New Features**: Advisors for security/performance, Integrations for cron/queues
4. **Authentication**: From users-only to full auth management with providers
5. **Realtime**: From inspector-only to full configuration panel
6. **UX**: Environment badges, Connect dialog, keyboard shortcuts

## References

- [Supabase Changelog](https://supabase.com/changelog)
- [Supabase Blog - Tabs Dashboard Updates](https://supabase.com/blog/tabs-dashboard-updates)
- [Supabase Studio 3.0](https://supabase.com/blog/supabase-studio-3-0)
- [Supabase AI Assistant](https://supabase.com/features/ai-assistant)
- [Supabase SQL Editor Features](https://supabase.com/features/sql-editor)
- [GitHub - Supabase Releases](https://github.com/supabase/supabase/releases)
- [Developer Update March 2025](https://github.com/orgs/supabase/discussions/34839)
