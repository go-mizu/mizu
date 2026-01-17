# 0393: Dashboard UI Frontend - Complete Supabase Dashboard Compatibility

**Version:** 2.0.0
**Date:** 2026-01-17
**Status:** Active Implementation

## Overview

This comprehensive specification documents all UI components, pages, features, and test cases required for 100% compatibility with the Supabase Dashboard (Studio). It covers every page, component, and interaction pattern based on the latest Supabase Dashboard (2025-2026).

## Table of Contents

1. [Current Implementation Status](#current-implementation-status)
2. [Navigation Structure](#navigation-structure)
3. [Page Specifications](#page-specifications)
4. [Component Library](#component-library)
5. [API Integration](#api-integration)
6. [E2E Test Cases](#e2e-test-cases)
7. [Accessibility Requirements](#accessibility-requirements)
8. [Performance Benchmarks](#performance-benchmarks)

---

## Current Implementation Status

### Implemented Pages (15 total)

| Page | Route | Components | API Integration | E2E Tests |
|------|-------|------------|-----------------|-----------|
| Dashboard | `/` | Stats cards, health status, quick links | ✅ | Pending |
| Table Editor | `/table-editor` | Schema selector, table list, data grid, CRUD modals | ✅ | Pending |
| SQL Editor | `/sql-editor` | Monaco editor, results table, saved queries | ✅ | Pending |
| Policies | `/database/policies` | Policy list, create/edit modals | ✅ | Pending |
| Indexes | `/database/indexes` | Index list, create/edit modals | ✅ | Pending |
| Views | `/database/views` | Regular/materialized views, refresh action | ✅ | Pending |
| Triggers | `/database/triggers` | Trigger list, create modal | ✅ | Pending |
| Roles | `/database/roles` | Role list, permissions, CRUD | ✅ | Pending |
| Authentication | `/auth/users` | User list, search, CRUD | ✅ | Pending |
| Storage | `/storage` | Bucket list, file browser, upload | ✅ | Pending |
| Realtime | `/realtime` | Connection stats, message inspector | ✅ | Pending |
| Edge Functions | `/functions` | Function list, deploy, invoke | ✅ | Pending |
| Logs Explorer | `/logs` | Log list, filters, search, export | ✅ | Pending |
| API Docs | `/api-docs` | Endpoint reference, examples | ✅ | Pending |
| Settings | `/settings` | Project, API, Database config | ✅ | Pending |

### Missing Features for Full Compatibility

| Feature | Priority | Status | Implementation Notes |
|---------|----------|--------|---------------------|
| Database Types/Enums | High | Not started | Need `/database/types` page |
| Database Extensions | High | Not started | Need `/database/extensions` page |
| Database Publications | Medium | Not started | Need `/database/publications` page |
| Database Functions (PL/pgSQL) | Medium | Not started | Need `/database/functions` page |
| Webhooks | Medium | Not started | Need `/integrations/webhooks` page |
| Cron Jobs | Medium | Not started | Need `/integrations/cron` page |
| Vault/Secrets | Medium | Not started | Need `/integrations/vault` page |
| Reports/Analytics | Low | Not started | Need `/reports` page |
| GraphiQL | Low | Not started | Need `/graphiql` page |
| Inline SQL Editor (CMD+K) | Medium | Not started | AI-assisted SQL editing |
| Table Tabs | Medium | Not started | Multiple table tabs in Table Editor |
| Query Tabs | Medium | Not started | Multiple query tabs in SQL Editor |

---

## Navigation Structure

### Supabase Dashboard 2025-2026 Navigation

```
├── Dashboard (Home)
│   ├── Project Overview
│   ├── Quick Stats (Users, Tables, Storage, Functions)
│   └── Service Health
│
├── Table Editor
│   ├── Schema Selector
│   ├── Table List (with search)
│   ├── Data Grid (sortable, filterable)
│   ├── Row Actions (edit, delete, duplicate)
│   └── Column Management (add, edit, delete)
│
├── SQL Editor
│   ├── Query Tabs
│   ├── Monaco Editor (SQL syntax highlighting)
│   ├── AI Assistant (CMD+K)
│   ├── Saved Queries
│   ├── Query History
│   ├── Results Table
│   └── Explain Analyze
│
├── Database
│   ├── Tables (mirrors Table Editor)
│   ├── Views (regular & materialized)
│   ├── Indexes (btree, hash, gin, gist)
│   ├── Triggers (before/after, row/statement)
│   ├── Functions (PL/pgSQL functions)
│   ├── Roles (permissions, membership)
│   ├── Extensions (enable/disable)
│   ├── Types (enums, composite)
│   ├── Publications (replication)
│   ├── Foreign Tables (FDW)
│   ├── Wrappers (postgres_fdw)
│   └── Backups (point-in-time recovery)
│
├── Authentication
│   ├── Users (list, CRUD, search)
│   ├── Policies (provider settings)
│   ├── Providers (email, OAuth, phone)
│   ├── URL Configuration
│   ├── Email Templates
│   ├── Rate Limits
│   └── Hooks
│
├── Storage
│   ├── Buckets (list, create, delete)
│   ├── File Browser (navigate, upload, download)
│   ├── Policies (RLS for storage)
│   └── Transformations (image resize)
│
├── Edge Functions
│   ├── Function List
│   ├── Deploy (via Editor or CLI)
│   ├── Logs (per function)
│   ├── Secrets/Env Vars
│   └── Invoke (test execution)
│
├── Realtime
│   ├── Inspector (message viewer)
│   ├── Policies (broadcast/presence RLS)
│   └── Connection Stats
│
├── Integrations
│   ├── Webhooks (database webhooks)
│   ├── Cron Jobs (pg_cron)
│   ├── Vault (secret management)
│   └── GraphiQL (GraphQL explorer)
│
├── Logs
│   ├── API Logs (REST requests)
│   ├── Postgres Logs (database queries)
│   ├── Auth Logs (authentication events)
│   ├── Storage Logs (file operations)
│   ├── Realtime Logs (WebSocket events)
│   └── Edge Function Logs
│
├── Reports
│   ├── API Usage
│   ├── Database Performance
│   ├── Storage Analytics
│   └── Auth Metrics
│
├── API Docs
│   ├── Auto-generated from schema
│   ├── Endpoint reference
│   └── Code examples
│
└── Settings
    ├── General (project name, region)
    ├── API (keys, JWT config)
    ├── Database (connection, pooling)
    ├── Auth (providers, templates)
    ├── Storage (file limits)
    └── Billing (usage, plans)
```

---

## Page Specifications

### 1. Dashboard (`/`)

**Purpose:** Project overview with quick stats and health status

**Components:**
- `StatsCard` - Displays metric with icon and value
- `HealthBadge` - Service status indicator
- `QuickAction` - Navigation shortcuts

**Data Requirements:**
```typescript
interface DashboardStats {
  users: { total: number; active_today: number; new_this_week: number };
  storage: { buckets: number; total_size: number; objects: number };
  database: { tables: number; total_rows: number; schemas: string[] };
  functions: { total: number; active: number; invocations_today: number };
  realtime: { active_connections: number; channels: number };
}

interface HealthStatus {
  status: 'healthy' | 'degraded' | 'unhealthy';
  services: {
    database: ServiceHealth;
    auth: ServiceHealth;
    storage: ServiceHealth;
    realtime: ServiceHealth;
  };
}
```

**Test Cases:**
```
E2E-DASH-001: Dashboard loads within 2 seconds
E2E-DASH-002: All stat cards display correct values
E2E-DASH-003: Health indicators show service status
E2E-DASH-004: Quick links navigate to correct pages
E2E-DASH-005: Refresh button updates stats
E2E-DASH-006: Responsive layout on mobile
```

---

### 2. Table Editor (`/table-editor`)

**Purpose:** Visual database table management with spreadsheet-like interface

**Components:**
- `SchemaSelector` - Dropdown to select database schema
- `TableList` - Sidebar list of tables with row counts
- `DataGrid` - Spreadsheet-like data viewer/editor
- `ColumnHeader` - Column name, type, key indicators
- `RowActions` - Edit, delete, duplicate row
- `CreateTableModal` - New table wizard
- `EditColumnModal` - Column modification

**Features:**
1. Schema selector (public, auth, storage, extensions)
2. Table list with search filter
3. Row count badges
4. RLS enabled indicators
5. Primary key column markers
6. Foreign key relationship display
7. Inline cell editing
8. Pagination (100 rows per page)
9. Column sorting
10. Column filtering
11. Export to CSV
12. Copy cell/row to clipboard

**Test Cases:**
```
E2E-TABLE-001: Table list loads with all tables
E2E-TABLE-002: Schema selector changes table list
E2E-TABLE-003: Table data loads when table selected
E2E-TABLE-004: Pagination works correctly
E2E-TABLE-005: Create table modal opens
E2E-TABLE-006: Table created with columns
E2E-TABLE-007: Delete table with confirmation
E2E-TABLE-008: Column headers show types
E2E-TABLE-009: Primary key indicator visible
E2E-TABLE-010: RLS badge shows correct status
E2E-TABLE-011: Cell values display correctly
E2E-TABLE-012: NULL values shown as placeholder
E2E-TABLE-013: Search filters tables
E2E-TABLE-014: Table row count accurate
E2E-TABLE-015: Inline editing updates cell
E2E-TABLE-016: Add row inserts new record
E2E-TABLE-017: Delete row with confirmation
E2E-TABLE-018: Sort by column works
E2E-TABLE-019: Export to CSV downloads file
E2E-TABLE-020: Copy cell to clipboard
```

---

### 3. SQL Editor (`/sql-editor`)

**Purpose:** Execute SQL queries with syntax highlighting and results

**Components:**
- `MonacoEditor` - SQL code editor
- `QueryToolbar` - Run, format, explain buttons
- `ResultsTable` - Query results display
- `SavedQueries` - Sidebar with saved queries
- `QueryTabs` - Multiple query tabs

**Features:**
1. Monaco editor with SQL syntax highlighting
2. Auto-completion for table/column names
3. Execute query (Ctrl+Enter / Cmd+Enter)
4. Format SQL button
5. Explain query plan
6. Query results table
7. Row count and duration display
8. Export results (CSV, JSON)
9. Save query with name
10. Load saved queries
11. Query history
12. Multiple query tabs

**Test Cases:**
```
E2E-SQL-001: Monaco editor loads
E2E-SQL-002: Syntax highlighting works
E2E-SQL-003: Execute SELECT query
E2E-SQL-004: Results table shows data
E2E-SQL-005: Row count displays correctly
E2E-SQL-006: Query duration shown
E2E-SQL-007: Execute INSERT query
E2E-SQL-008: Execute UPDATE query
E2E-SQL-009: Execute DELETE query
E2E-SQL-010: Query error shows message
E2E-SQL-011: Save query opens modal
E2E-SQL-012: Query saved to sidebar
E2E-SQL-013: Load saved query
E2E-SQL-014: Delete saved query
E2E-SQL-015: Export CSV downloads file
E2E-SQL-016: Export JSON downloads file
E2E-SQL-017: Ctrl+Enter executes query
E2E-SQL-018: Format SQL button works
E2E-SQL-019: Explain shows query plan
E2E-SQL-020: Multiple statements execute
```

---

### 4. Policies (`/database/policies`)

**Purpose:** Row Level Security policy management

**Components:**
- `PolicyList` - Grouped by table
- `PolicyCard` - Policy details display
- `CreatePolicyModal` - Policy wizard
- `PolicyEditor` - SQL expression editor

**Features:**
1. List policies grouped by schema/table
2. Policy details (command, using, with check)
3. Create policy with templates
4. Edit policy expression
5. Delete policy with confirmation
6. Enable/disable RLS per table
7. Policy roles filter

**Test Cases:**
```
E2E-POLICY-001: Policy list loads
E2E-POLICY-002: Policies grouped by table
E2E-POLICY-003: Create SELECT policy
E2E-POLICY-004: Create INSERT policy
E2E-POLICY-005: Create UPDATE policy
E2E-POLICY-006: Create DELETE policy
E2E-POLICY-007: Policy shows correct roles
E2E-POLICY-008: Edit policy expression
E2E-POLICY-009: Delete policy with confirm
E2E-POLICY-010: Enable RLS on table
E2E-POLICY-011: Disable RLS on table
E2E-POLICY-012: Policy template selection
E2E-POLICY-013: USING expression saved
E2E-POLICY-014: WITH CHECK expression saved
E2E-POLICY-015: Search/filter policies
```

---

### 5. Indexes (`/database/indexes`)

**Purpose:** Database index management for query optimization

**Components:**
- `IndexList` - All indexes with details
- `IndexCard` - Index information display
- `CreateIndexModal` - Index creation wizard
- `IndexStats` - Usage statistics

**Features:**
1. List all indexes with table association
2. Index type (btree, hash, gin, gist)
3. Column selection
4. Unique index toggle
5. Partial index condition
6. Index size display
7. Usage statistics
8. Create/drop indexes

**Test Cases:**
```
E2E-INDEX-001: Index list loads
E2E-INDEX-002: Indexes show table name
E2E-INDEX-003: Index type displayed
E2E-INDEX-004: Create btree index
E2E-INDEX-005: Create unique index
E2E-INDEX-006: Create multi-column index
E2E-INDEX-007: Index size shown
E2E-INDEX-008: Delete index with confirm
E2E-INDEX-009: Index definition displayed
E2E-INDEX-010: Search indexes by name
```

---

### 6. Views (`/database/views`)

**Purpose:** Manage database views and materialized views

**Components:**
- `ViewList` - Regular and materialized views
- `ViewDefinition` - SQL definition display
- `CreateViewModal` - View creation
- `RefreshButton` - Materialized view refresh

**Features:**
1. List regular views
2. List materialized views
3. View SQL definition
4. Create view with SQL
5. Edit view definition
6. Drop view
7. Refresh materialized view
8. View columns display

**Test Cases:**
```
E2E-VIEW-001: View list loads
E2E-VIEW-002: Regular views displayed
E2E-VIEW-003: Materialized views displayed
E2E-VIEW-004: Create simple view
E2E-VIEW-005: Create materialized view
E2E-VIEW-006: View definition shown
E2E-VIEW-007: Edit view SQL
E2E-VIEW-008: Delete view with confirm
E2E-VIEW-009: Refresh materialized view
E2E-VIEW-010: View columns listed
```

---

### 7. Triggers (`/database/triggers`)

**Purpose:** Database trigger management

**Components:**
- `TriggerList` - All triggers grouped by table
- `TriggerCard` - Trigger details
- `CreateTriggerModal` - Trigger creation wizard

**Features:**
1. List triggers by table
2. Trigger timing (BEFORE/AFTER/INSTEAD OF)
3. Trigger events (INSERT/UPDATE/DELETE)
4. Trigger function association
5. Row/statement orientation
6. Enable/disable trigger
7. Trigger condition (WHEN)

**Test Cases:**
```
E2E-TRIGGER-001: Trigger list loads
E2E-TRIGGER-002: Triggers grouped by table
E2E-TRIGGER-003: Create BEFORE INSERT trigger
E2E-TRIGGER-004: Create AFTER UPDATE trigger
E2E-TRIGGER-005: Trigger function shown
E2E-TRIGGER-006: Trigger events displayed
E2E-TRIGGER-007: Delete trigger with confirm
E2E-TRIGGER-008: Enable/disable trigger
E2E-TRIGGER-009: FOR EACH ROW shown
E2E-TRIGGER-010: Search triggers
```

---

### 8. Roles (`/database/roles`)

**Purpose:** Database role and permission management

**Components:**
- `RoleList` - All database roles
- `RoleCard` - Role permissions display
- `CreateRoleModal` - Role creation
- `PrivilegeMatrix` - Grant/revoke UI

**Features:**
1. List all roles
2. Role attributes (superuser, login, create_db)
3. Create new role
4. Edit role permissions
5. Delete role
6. Grant/revoke privileges
7. Role membership
8. Active connections count

**Test Cases:**
```
E2E-ROLE-001: Role list loads
E2E-ROLE-002: Standard roles shown (anon, authenticated)
E2E-ROLE-003: Create new role
E2E-ROLE-004: Role permissions displayed
E2E-ROLE-005: Edit role attributes
E2E-ROLE-006: Delete role with confirm
E2E-ROLE-007: Superuser badge shown
E2E-ROLE-008: Login capability indicated
E2E-ROLE-009: Connection count shown
E2E-ROLE-010: Search roles by name
```

---

### 9. Authentication/Users (`/auth/users`)

**Purpose:** User management for authentication

**Components:**
- `UserList` - Paginated user table
- `UserSearch` - Email/phone search
- `UserCard` - User details
- `CreateUserModal` - New user form
- `EditUserModal` - User modification

**Features:**
1. List users with pagination
2. Search by email/phone
3. Filter by verification status
4. User metadata display
5. Provider badges
6. Last sign-in time
7. Create user
8. Edit user metadata
9. Delete user
10. Ban/unban user
11. Send magic link
12. Reset password

**Test Cases:**
```
E2E-AUTH-001: User list loads
E2E-AUTH-002: Pagination works
E2E-AUTH-003: Search by email
E2E-AUTH-004: Create new user
E2E-AUTH-005: User email displayed
E2E-AUTH-006: Provider shown
E2E-AUTH-007: Last sign-in time shown
E2E-AUTH-008: Edit user metadata
E2E-AUTH-009: Delete user with confirm
E2E-AUTH-010: Email verified badge
E2E-AUTH-011: Filter by status
E2E-AUTH-012: User ID displayed
```

---

### 10. Storage (`/storage`)

**Purpose:** Object storage management (S3-compatible)

**Components:**
- `BucketList` - Sidebar bucket list
- `FileBrowser` - File/folder navigation
- `FileUploader` - Drag-drop upload zone
- `FileActions` - Download, delete, copy URL
- `CreateBucketModal` - New bucket form

**Features:**
1. List storage buckets
2. Public/private bucket badges
3. Create bucket
4. Delete bucket (empty first)
5. File browser with breadcrumbs
6. Upload files (drag-drop)
7. Upload multiple files
8. Create folders
9. Download files
10. Delete files
11. Copy public URL
12. Generate signed URL
13. File preview (images)
14. File search

**Test Cases:**
```
E2E-STORAGE-001: Bucket list loads
E2E-STORAGE-002: Create public bucket
E2E-STORAGE-003: Create private bucket
E2E-STORAGE-004: Delete empty bucket
E2E-STORAGE-005: Upload single file
E2E-STORAGE-006: Upload multiple files
E2E-STORAGE-007: File appears in list
E2E-STORAGE-008: Download file
E2E-STORAGE-009: Delete file with confirm
E2E-STORAGE-010: Navigate into folder
E2E-STORAGE-011: Breadcrumb navigation
E2E-STORAGE-012: Copy public URL
E2E-STORAGE-013: Generate signed URL
E2E-STORAGE-014: Public badge shown
E2E-STORAGE-015: File size displayed
```

---

### 11. Realtime (`/realtime`)

**Purpose:** WebSocket connection monitoring and testing

**Components:**
- `ConnectionStats` - Active connections count
- `ChannelList` - Active channels
- `MessageInspector` - Live message viewer
- `ConnectionStatus` - WebSocket status badge

**Features:**
1. Active connection count
2. Channel list with subscriptions
3. Message inspector (last 100)
4. Message type filtering
5. Clear messages
6. Auto-refresh stats
7. WebSocket connection status
8. Test broadcast message
9. Test presence join/leave

**Test Cases:**
```
E2E-RT-001: Realtime page loads
E2E-RT-002: Connection count shown
E2E-RT-003: Channel list displayed
E2E-RT-004: Message inspector works
E2E-RT-005: WebSocket status shown
E2E-RT-006: Auto-refresh updates stats
E2E-RT-007: Clear messages works
E2E-RT-008: Message timestamp shown
E2E-RT-009: Message type displayed
E2E-RT-010: Channel name in message
```

---

### 12. Edge Functions (`/functions`)

**Purpose:** Deno Edge Function management

**Components:**
- `FunctionList` - All functions table
- `FunctionCard` - Function details
- `DeployModal` - Code deployment
- `InvokeModal` - Test invocation
- `SecretsPanel` - Environment variables

**Features:**
1. List all functions
2. Function status (active/inactive)
3. JWT verification toggle
4. Create function
5. Deploy function code
6. Invoke function (test)
7. View function logs
8. Manage secrets/env vars
9. Delete function
10. Function URL display

**Test Cases:**
```
E2E-FUNC-001: Function list loads
E2E-FUNC-002: Create new function
E2E-FUNC-003: Function appears in list
E2E-FUNC-004: Deploy function code
E2E-FUNC-005: Invoke function GET
E2E-FUNC-006: Invoke function POST
E2E-FUNC-007: View function response
E2E-FUNC-008: Delete function with confirm
E2E-FUNC-009: Function URL shown
E2E-FUNC-010: JWT toggle works
E2E-FUNC-011: Status badge displayed
E2E-FUNC-012: Last updated time shown
```

---

### 13. Logs Explorer (`/logs`)

**Purpose:** Centralized log viewing and search

**Components:**
- `LogTypeSelector` - Type filter dropdown
- `LogLevelFilter` - Level segmented control
- `TimeRangePicker` - Date/time range
- `LogSearch` - Text search input
- `LogTable` - Log entries display
- `LogDetailDrawer` - Full log details

**Features:**
1. List logs by type
2. Filter by level (error, warning, info, debug)
3. Time range selection
4. Text search in logs
5. Log detail view
6. Export logs (JSON, CSV)
7. Auto-refresh toggle
8. Pagination
9. Log metadata display

**Test Cases:**
```
E2E-LOG-001: Logs page loads
E2E-LOG-002: Log types available
E2E-LOG-003: Filter by error level
E2E-LOG-004: Filter by warning level
E2E-LOG-005: Time range filter works
E2E-LOG-006: Search logs by text
E2E-LOG-007: Log detail drawer opens
E2E-LOG-008: Export to JSON works
E2E-LOG-009: Export to CSV works
E2E-LOG-010: Auto-refresh toggle
E2E-LOG-011: Pagination works
E2E-LOG-012: Timestamp formatted
E2E-LOG-013: Level color coding
E2E-LOG-014: Metadata displayed
E2E-LOG-015: Clear filters button
```

---

### 14. API Docs (`/api-docs`)

**Purpose:** Auto-generated API documentation

**Components:**
- `EndpointList` - API endpoints by category
- `EndpointCard` - Endpoint details
- `CodeExample` - Request/response examples
- `AuthSection` - Authentication info

**Features:**
1. Base URL display
2. Authentication examples
3. Endpoint reference (Auth, REST, Storage, Realtime, Functions)
4. HTTP method badges
5. Path parameters
6. Request/response examples
7. Copy path button
8. Try it out (optional)

**Test Cases:**
```
E2E-API-001: API docs page loads
E2E-API-002: Base URL displayed
E2E-API-003: Auth section expanded
E2E-API-004: REST endpoints listed
E2E-API-005: Storage endpoints listed
E2E-API-006: Method badges colored
E2E-API-007: Copy path works
E2E-API-008: Examples shown
E2E-API-009: Accordion expand/collapse
E2E-API-010: Search endpoints
```

---

### 15. Settings (`/settings`)

**Purpose:** Project configuration and settings

**Components:**
- `SettingsTabs` - Tab navigation
- `GeneralSettings` - Project name, region
- `APISettings` - Keys, JWT config
- `DatabaseSettings` - Connection details
- `CopyButton` - Copy to clipboard

**Features:**
1. General settings (project name)
2. API keys display (anon, service role)
3. JWT secret (hidden)
4. Database connection details
5. Connection string
6. Copy to clipboard buttons
7. Save settings

**Test Cases:**
```
E2E-SET-001: Settings page loads
E2E-SET-002: General tab active
E2E-SET-003: Project name editable
E2E-SET-004: API keys tab works
E2E-SET-005: Anon key displayed
E2E-SET-006: Service role key displayed
E2E-SET-007: Copy anon key works
E2E-SET-008: Copy service key works
E2E-SET-009: Database tab works
E2E-SET-010: Connection string shown
E2E-SET-011: Copy connection string
E2E-SET-012: Save settings works
```

---

## Component Library

### Common Components

| Component | Purpose | Location |
|-----------|---------|----------|
| `PageContainer` | Page wrapper with title | `components/layout/PageContainer.tsx` |
| `Sidebar` | Navigation sidebar | `components/layout/Sidebar.tsx` |
| `DataTable` | Reusable data table | `components/common/DataTable.tsx` |
| `ConfirmModal` | Delete/destructive confirmation | `components/common/ConfirmModal.tsx` |
| `EmptyState` | Empty data display | `components/common/EmptyState.tsx` |
| `StatusBadge` | Status indicators | `components/common/StatusBadge.tsx` |
| `SearchInput` | Search field | `components/forms/SearchInput.tsx` |
| `CopyButton` | Copy to clipboard | `components/common/CopyButton.tsx` |
| `LoadingState` | Loading skeleton | `components/common/LoadingState.tsx` |

### New Components Needed

| Component | Purpose | Priority |
|-----------|---------|----------|
| `QueryTabs` | Multiple SQL query tabs | High |
| `TableTabs` | Multiple table tabs | High |
| `AIAssistant` | CMD+K inline AI | Medium |
| `CodeEditor` | Generic code editor | Medium |
| `TimeRangePicker` | Date/time range selector | High |
| `ExportButton` | Export dropdown | Medium |
| `BulkActions` | Multi-select actions | Medium |
| `SchemaViewer` | Schema visualization | Low |

---

## API Integration

### API Modules

| Module | File | Endpoints |
|--------|------|-----------|
| Auth | `api/auth.ts` | listUsers, createUser, updateUser, deleteUser |
| Database | `api/database.ts` | schemas, tables, columns, policies, query |
| Storage | `api/storage.ts` | buckets, objects, upload, download |
| Functions | `api/functions.ts` | list, create, deploy, invoke, delete |
| Realtime | `api/realtime.ts` | stats, channels, WebSocket client |
| Dashboard | `api/dashboard.ts` | stats, health |
| PGMeta | `api/pgmeta.ts` | indexes, views, triggers, roles, types |
| Logs | `api/logs.ts` | listLogs, searchLogs, exportLogs |

### New API Modules Needed

| Module | File | Endpoints |
|--------|------|-----------|
| Types | `api/types.ts` | listTypes, createType, dropType |
| Extensions | `api/extensions.ts` | listExtensions, enableExtension, disableExtension |
| Publications | `api/publications.ts` | listPublications, createPublication, dropPublication |
| Webhooks | `api/webhooks.ts` | listWebhooks, createWebhook, deleteWebhook |
| Cron | `api/cron.ts` | listJobs, createJob, deleteJob, runJob |
| Vault | `api/vault.ts` | listSecrets, createSecret, deleteSecret |

---

## E2E Test Case Summary

### Total Test Cases by Category

| Category | Count | Priority |
|----------|-------|----------|
| Dashboard | 6 | High |
| Table Editor | 20 | High |
| SQL Editor | 20 | High |
| Policies | 15 | High |
| Indexes | 10 | Medium |
| Views | 10 | Medium |
| Triggers | 10 | Medium |
| Roles | 10 | Medium |
| Authentication | 12 | High |
| Storage | 15 | High |
| Realtime | 10 | Medium |
| Edge Functions | 12 | High |
| Logs Explorer | 15 | Medium |
| API Docs | 10 | Low |
| Settings | 12 | Medium |
| **Total** | **177** | - |

---

## Accessibility Requirements

### WCAG 2.1 Level AA Compliance

1. **Keyboard Navigation** - All interactive elements accessible via keyboard
2. **Focus Indicators** - Visible focus states on all controls
3. **Color Contrast** - Minimum 4.5:1 for text, 3:1 for UI components
4. **Screen Reader Support** - Proper ARIA labels and roles
5. **Error Messages** - Clear, descriptive error messages
6. **Form Labels** - All form inputs have associated labels

---

## Performance Benchmarks

| Metric | Target | Measurement |
|--------|--------|-------------|
| Initial Page Load | < 2s | Time to interactive |
| API Response | < 500ms | 95th percentile |
| Table Data Load | < 1s | 1000 rows |
| File Upload | < 5s | 10MB file |
| Search Response | < 300ms | 100+ results |

---

## Implementation Phases

### Phase 1: Core E2E Testing (Current)
- Set up Playwright framework
- Implement tests for all existing pages
- Ensure 100% pass rate

### Phase 2: Missing Pages
- Implement Types page
- Implement Extensions page
- Implement Publications page
- Implement Database Functions page

### Phase 3: Integrations
- Implement Webhooks page
- Implement Cron Jobs page
- Implement Vault page

### Phase 4: Enhanced Features
- Query tabs in SQL Editor
- Table tabs in Table Editor
- AI Assistant (CMD+K)
- Reports/Analytics page

---

## References

- [Supabase Dashboard](https://supabase.com)
- [Supabase GitHub - Studio](https://github.com/supabase/supabase/tree/master/apps/studio)
- [Supabase Docs - Database](https://supabase.com/docs/guides/database)
- [Supabase Docs - Auth](https://supabase.com/docs/guides/auth)
- [Supabase Docs - Storage](https://supabase.com/docs/guides/storage)
- [Supabase Docs - Edge Functions](https://supabase.com/docs/guides/functions)
- [Supabase Docs - Realtime](https://supabase.com/docs/guides/realtime)
- [Supabase UI Library](https://supabase.com/docs/guides/ui-library)
- [Playwright Testing](https://playwright.dev/)

---

## Changelog

- **2026-01-17**: Initial comprehensive specification created
  - Documented all 15 implemented pages
  - Listed 177 E2E test cases
  - Identified missing features for full compatibility
  - Defined component library requirements
  - Set performance benchmarks
