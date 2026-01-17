# 0392: Dashboard UI Frontend - Supabase Dashboard Compatibility

## Overview

This specification details the comprehensive test cases and implementation requirements for achieving 100% compatibility with the Supabase Dashboard. The goal is to ensure LocalBase provides a complete local development experience that mirrors all Supabase Dashboard features.

## Current Implementation Status

### Implemented Features ✅

| Feature | Frontend | Backend API | Tests |
|---------|----------|-------------|-------|
| Dashboard Overview | ✅ | ✅ | ✅ |
| Table Editor | ✅ | ✅ | ✅ |
| SQL Editor | ✅ | ✅ | ✅ |
| Authentication/Users | ✅ | ✅ | ✅ |
| Storage | ✅ | ✅ | ✅ |
| Realtime | ✅ | ✅ | ✅ |
| Edge Functions | ✅ | ✅ | ✅ |
| API Docs | ✅ | ✅ | ✅ |
| Settings | ✅ | ✅ | ✅ |

### Missing Features ❌

| Feature | Frontend | Backend API | Tests | Priority |
|---------|----------|-------------|-------|----------|
| Database Policies | ❌ | ✅ | ✅ | High |
| Database Triggers | ❌ | ✅ | ✅ | High |
| Database Indexes | ❌ | ✅ | ✅ | High |
| Database Views | ❌ | ✅ | ✅ | High |
| Database Roles | ❌ | ✅ | ✅ | Medium |
| Database Types | ❌ | ✅ | ✅ | Medium |
| Database Publications | ❌ | ✅ | ✅ | Medium |
| Database Foreign Tables | ❌ | ✅ | ✅ | Low |
| Logs Explorer | ❌ | ✅ | ✅ | High |
| Database Webhooks | ❌ | ❌ | ❌ | Medium |
| Cron Jobs | ❌ | ❌ | ❌ | Medium |
| Vault/Secrets | ❌ | ❌ | ❌ | Medium |
| Reports/Analytics | ❌ | ❌ | ❌ | Low |

## Supabase Dashboard Navigation Structure

Based on the latest Supabase Dashboard (2025), the navigation structure is:

```
├── Dashboard (Home)
├── Table Editor
│   └── [Tables with inline editing, RLS badges]
├── SQL Editor
│   └── [Saved queries, templates]
├── Database
│   ├── Tables
│   ├── Views
│   ├── Indexes
│   ├── Triggers
│   ├── Functions (DB functions)
│   ├── Roles
│   ├── Extensions
│   ├── Types
│   ├── Publications
│   ├── Foreign Tables
│   ├── Wrappers
│   └── Backups
├── Authentication
│   ├── Users
│   ├── Policies
│   ├── Providers
│   └── URL Configuration
├── Storage
│   └── [Buckets with file management]
├── Edge Functions
│   └── [Function management]
├── Realtime
│   ├── Inspector
│   └── Policies
├── Integrations
│   ├── Webhooks
│   ├── Cron Jobs
│   ├── Vault
│   └── GraphiQL
├── Logs
│   ├── API Logs
│   ├── Postgres Logs
│   ├── Auth Logs
│   ├── Storage Logs
│   ├── Realtime Logs
│   └── Edge Function Logs
├── Reports
├── API Docs
└── Settings
    ├── General
    ├── API
    ├── Auth
    ├── Database
    └── Storage
```

## Test Cases

### 1. Dashboard Overview Tests

```
TEST-DASH-001: Dashboard loads with all stats
TEST-DASH-002: Dashboard shows correct user count
TEST-DASH-003: Dashboard shows correct storage usage
TEST-DASH-004: Dashboard shows correct function count
TEST-DASH-005: Dashboard shows database table count
TEST-DASH-006: Dashboard shows realtime connections
TEST-DASH-007: Dashboard health check shows all services
TEST-DASH-008: Dashboard refresh updates stats
```

### 2. Table Editor Tests

```
TEST-TABLE-001: List all tables in schema
TEST-TABLE-002: Create new table with columns
TEST-TABLE-003: Delete table with confirmation
TEST-TABLE-004: View table data with pagination
TEST-TABLE-005: Filter table data
TEST-TABLE-006: Sort table data by column
TEST-TABLE-007: Add column to existing table
TEST-TABLE-008: Modify column type
TEST-TABLE-009: Delete column with confirmation
TEST-TABLE-010: Show RLS enabled badge
TEST-TABLE-011: Show primary key indicator
TEST-TABLE-012: Show foreign key relationships
TEST-TABLE-013: Inline row editing
TEST-TABLE-014: Add new row
TEST-TABLE-015: Delete row with confirmation
TEST-TABLE-016: Schema selector dropdown
TEST-TABLE-017: Table search/filter
TEST-TABLE-018: Column resize
TEST-TABLE-019: Copy cell value
TEST-TABLE-020: NULL value handling
```

### 3. SQL Editor Tests

```
TEST-SQL-001: Execute SELECT query
TEST-SQL-002: Execute INSERT query
TEST-SQL-003: Execute UPDATE query
TEST-SQL-004: Execute DELETE query
TEST-SQL-005: Execute multiple statements
TEST-SQL-006: Syntax highlighting
TEST-SQL-007: Auto-completion for tables
TEST-SQL-008: Auto-completion for columns
TEST-SQL-009: Query results display
TEST-SQL-010: Query error handling
TEST-SQL-011: Save query as template
TEST-SQL-012: Load saved query
TEST-SQL-013: Query history
TEST-SQL-014: Export results as CSV
TEST-SQL-015: Export results as JSON
TEST-SQL-016: Format SQL button
TEST-SQL-017: Explain query plan
TEST-SQL-018: Query execution time display
TEST-SQL-019: Row count display
TEST-SQL-020: Multiple tabs support
```

### 4. Database Policies Tests (RLS)

```
TEST-POLICY-001: List all policies for table
TEST-POLICY-002: Create SELECT policy
TEST-POLICY-003: Create INSERT policy
TEST-POLICY-004: Create UPDATE policy
TEST-POLICY-005: Create DELETE policy
TEST-POLICY-006: Create ALL policy
TEST-POLICY-007: Delete policy with confirmation
TEST-POLICY-008: Enable RLS on table
TEST-POLICY-009: Disable RLS on table
TEST-POLICY-010: Policy with auth.uid() function
TEST-POLICY-011: Policy with auth.role() function
TEST-POLICY-012: Policy template selection
TEST-POLICY-013: Policy syntax validation
TEST-POLICY-014: Policy roles filter
TEST-POLICY-015: Policy with check expression
```

### 5. Database Triggers Tests

```
TEST-TRIGGER-001: List all triggers
TEST-TRIGGER-002: Create BEFORE INSERT trigger
TEST-TRIGGER-003: Create AFTER INSERT trigger
TEST-TRIGGER-004: Create BEFORE UPDATE trigger
TEST-TRIGGER-005: Create AFTER UPDATE trigger
TEST-TRIGGER-006: Create BEFORE DELETE trigger
TEST-TRIGGER-007: Create AFTER DELETE trigger
TEST-TRIGGER-008: Create INSTEAD OF trigger
TEST-TRIGGER-009: Delete trigger with confirmation
TEST-TRIGGER-010: Enable/disable trigger
TEST-TRIGGER-011: Trigger function selection
TEST-TRIGGER-012: Trigger condition (WHEN)
TEST-TRIGGER-013: FOR EACH ROW/STATEMENT
```

### 6. Database Indexes Tests

```
TEST-INDEX-001: List all indexes
TEST-INDEX-002: Create btree index
TEST-INDEX-003: Create hash index
TEST-INDEX-004: Create gin index
TEST-INDEX-005: Create gist index
TEST-INDEX-006: Create unique index
TEST-INDEX-007: Create partial index (WHERE)
TEST-INDEX-008: Create multi-column index
TEST-INDEX-009: Delete index with confirmation
TEST-INDEX-010: Reindex command
TEST-INDEX-011: Index usage statistics
TEST-INDEX-012: Index size display
```

### 7. Database Views Tests

```
TEST-VIEW-001: List all views
TEST-VIEW-002: Create simple view
TEST-VIEW-003: Create view with JOIN
TEST-VIEW-004: Create view with aggregation
TEST-VIEW-005: Update view definition
TEST-VIEW-006: Delete view with confirmation
TEST-VIEW-007: Materialized view create
TEST-VIEW-008: Materialized view refresh
TEST-VIEW-009: Materialized view delete
TEST-VIEW-010: View columns display
TEST-VIEW-011: View definition display
```

### 8. Database Roles Tests

```
TEST-ROLE-001: List all roles
TEST-ROLE-002: Create new role
TEST-ROLE-003: Delete role with confirmation
TEST-ROLE-004: Grant SELECT privilege
TEST-ROLE-005: Grant INSERT privilege
TEST-ROLE-006: Grant UPDATE privilege
TEST-ROLE-007: Grant DELETE privilege
TEST-ROLE-008: Revoke privilege
TEST-ROLE-009: Role membership (GRANT role TO role)
TEST-ROLE-010: Superuser flag display
TEST-ROLE-011: Login flag display
TEST-ROLE-012: Password management
```

### 9. Database Types Tests

```
TEST-TYPE-001: List all custom types
TEST-TYPE-002: Create ENUM type
TEST-TYPE-003: Create composite type
TEST-TYPE-004: Delete type with confirmation
TEST-TYPE-005: Add value to ENUM
TEST-TYPE-006: Type usage display
```

### 10. Database Extensions Tests

```
TEST-EXT-001: List all available extensions
TEST-EXT-002: Enable uuid-ossp extension
TEST-EXT-003: Enable pgcrypto extension
TEST-EXT-004: Enable pg_stat_statements
TEST-EXT-005: Enable pg_trgm extension
TEST-EXT-006: Enable postgis extension
TEST-EXT-007: Disable extension with confirmation
TEST-EXT-008: Extension version display
TEST-EXT-009: Extension schema selection
```

### 11. Database Publications Tests

```
TEST-PUB-001: List all publications
TEST-PUB-002: Create publication for all tables
TEST-PUB-003: Create publication for specific tables
TEST-PUB-004: Delete publication with confirmation
TEST-PUB-005: Add table to publication
TEST-PUB-006: Remove table from publication
```

### 12. Authentication Users Tests

```
TEST-AUTH-001: List all users with pagination
TEST-AUTH-002: Search users by email
TEST-AUTH-003: Filter users by status
TEST-AUTH-004: Create new user
TEST-AUTH-005: Delete user with confirmation
TEST-AUTH-006: Update user email
TEST-AUTH-007: Update user metadata
TEST-AUTH-008: Ban user
TEST-AUTH-009: Unban user
TEST-AUTH-010: Send password reset email
TEST-AUTH-011: Confirm user email
TEST-AUTH-012: View user sessions
TEST-AUTH-013: Revoke user sessions
TEST-AUTH-014: View MFA factors
TEST-AUTH-015: Remove MFA factor
```

### 13. Storage Tests

```
TEST-STOR-001: List all buckets
TEST-STOR-002: Create public bucket
TEST-STOR-003: Create private bucket
TEST-STOR-004: Delete bucket with confirmation
TEST-STOR-005: Empty bucket with confirmation
TEST-STOR-006: Upload single file
TEST-STOR-007: Upload multiple files
TEST-STOR-008: Download file
TEST-STOR-009: Delete file with confirmation
TEST-STOR-010: Move file between folders
TEST-STOR-011: Copy file
TEST-STOR-012: Create folder
TEST-STOR-013: Navigate folder hierarchy
TEST-STOR-014: File preview (images)
TEST-STOR-015: Copy public URL
TEST-STOR-016: Generate signed URL
TEST-STOR-017: Bucket policies display
TEST-STOR-018: File search
TEST-STOR-019: File sorting
TEST-STOR-020: Storage usage display
```

### 14. Edge Functions Tests

```
TEST-FUNC-001: List all functions
TEST-FUNC-002: Create new function
TEST-FUNC-003: Update function code
TEST-FUNC-004: Delete function with confirmation
TEST-FUNC-005: Deploy function
TEST-FUNC-006: View deployment history
TEST-FUNC-007: Invoke function (GET)
TEST-FUNC-008: Invoke function (POST)
TEST-FUNC-009: Invoke function with body
TEST-FUNC-010: Function logs display
TEST-FUNC-011: Environment secrets list
TEST-FUNC-012: Add environment secret
TEST-FUNC-013: Delete environment secret
TEST-FUNC-014: Function URL display
TEST-FUNC-015: JWT verification toggle
```

### 15. Realtime Tests

```
TEST-RT-001: List active channels
TEST-RT-002: View channel subscriptions
TEST-RT-003: View realtime stats
TEST-RT-004: Test broadcast message
TEST-RT-005: Test presence join
TEST-RT-006: Test presence leave
TEST-RT-007: Test postgres changes subscription
TEST-RT-008: WebSocket connection status
TEST-RT-009: Message inspector
TEST-RT-010: Connection count display
```

### 16. Logs Explorer Tests

```
TEST-LOG-001: List API logs
TEST-LOG-002: List Postgres logs
TEST-LOG-003: List Auth logs
TEST-LOG-004: List Storage logs
TEST-LOG-005: List Realtime logs
TEST-LOG-006: List Edge Function logs
TEST-LOG-007: Filter logs by level
TEST-LOG-008: Filter logs by time range
TEST-LOG-009: Search logs by text
TEST-LOG-010: Export logs as JSON
TEST-LOG-011: Export logs as CSV
TEST-LOG-012: Log detail view
TEST-LOG-013: Log timestamp formatting
TEST-LOG-014: Log pagination
TEST-LOG-015: Auto-refresh toggle
```

### 17. Database Webhooks Tests

```
TEST-HOOK-001: List all webhooks
TEST-HOOK-002: Create webhook for INSERT
TEST-HOOK-003: Create webhook for UPDATE
TEST-HOOK-004: Create webhook for DELETE
TEST-HOOK-005: Delete webhook with confirmation
TEST-HOOK-006: Enable/disable webhook
TEST-HOOK-007: Webhook URL validation
TEST-HOOK-008: Webhook headers configuration
TEST-HOOK-009: Webhook secret/signature
TEST-HOOK-010: Webhook delivery history
```

### 18. Cron Jobs Tests

```
TEST-CRON-001: List all cron jobs
TEST-CRON-002: Create cron job with SQL
TEST-CRON-003: Create cron job with function
TEST-CRON-004: Delete cron job with confirmation
TEST-CRON-005: Enable/disable cron job
TEST-CRON-006: Cron expression validation
TEST-CRON-007: Job execution history
TEST-CRON-008: Job last run status
TEST-CRON-009: Job next run display
```

### 19. Vault/Secrets Tests

```
TEST-VAULT-001: List all secrets
TEST-VAULT-002: Create encrypted secret
TEST-VAULT-003: Update secret value
TEST-VAULT-004: Delete secret with confirmation
TEST-VAULT-005: Secret access in SQL
TEST-VAULT-006: Secret encryption indicator
```

### 20. Settings Tests

```
TEST-SET-001: View project settings
TEST-SET-002: Update project name
TEST-SET-003: View API settings
TEST-SET-004: Update max rows limit
TEST-SET-005: View auth settings
TEST-SET-006: Toggle signup enabled
TEST-SET-007: Update password requirements
TEST-SET-008: View database settings
TEST-SET-009: Update connection pool size
TEST-SET-010: View storage settings
TEST-SET-011: Update file size limit
TEST-SET-012: API keys display (masked)
TEST-SET-013: Copy API key to clipboard
```

## API Endpoints Required

### Webhooks API (New)

```
POST   /api/webhooks           - Create webhook
GET    /api/webhooks           - List webhooks
GET    /api/webhooks/{id}      - Get webhook
PATCH  /api/webhooks/{id}      - Update webhook
DELETE /api/webhooks/{id}      - Delete webhook
POST   /api/webhooks/{id}/test - Test webhook
GET    /api/webhooks/{id}/logs - Get webhook logs
```

### Cron Jobs API (New)

```
POST   /api/cron               - Create cron job
GET    /api/cron               - List cron jobs
GET    /api/cron/{id}          - Get cron job
PATCH  /api/cron/{id}          - Update cron job
DELETE /api/cron/{id}          - Delete cron job
POST   /api/cron/{id}/run      - Run job now
GET    /api/cron/{id}/history  - Get job history
```

### Vault API (New)

```
POST   /api/vault/secrets         - Create secret
GET    /api/vault/secrets         - List secrets (metadata only)
GET    /api/vault/secrets/{name}  - Get secret (if authorized)
PATCH  /api/vault/secrets/{name}  - Update secret
DELETE /api/vault/secrets/{name}  - Delete secret
```

## Frontend Pages to Implement

### 1. Database Section Pages

#### `/database/policies` - RLS Policies Page

Features:
- List all policies grouped by table
- Create/edit/delete policies
- Policy templates (public read, authenticated, owner-only)
- Syntax highlighting for policy expressions
- Enable/disable RLS per table

#### `/database/triggers` - Triggers Page

Features:
- List all triggers with table association
- Create trigger with wizard
- Select trigger function
- Timing and event selection
- Enable/disable triggers

#### `/database/indexes` - Indexes Page

Features:
- List all indexes with table association
- Create index with column selector
- Index type selection (btree, hash, gin, gist)
- Unique/partial index options
- Index size and usage stats

#### `/database/views` - Views Page

Features:
- List regular and materialized views
- Create view with SQL editor
- View definition display
- Materialized view refresh action
- Column details

#### `/database/roles` - Roles Page

Features:
- List all database roles
- Create/delete roles
- Grant/revoke privileges
- Role membership management
- Role attributes (superuser, login, etc.)

#### `/database/types` - Types Page

Features:
- List custom types
- Create ENUM types
- Create composite types
- Type values management

#### `/database/extensions` - Extensions Page

Features:
- List available extensions
- Enable/disable extensions
- Extension details and documentation links

#### `/database/publications` - Publications Page

Features:
- List publications
- Create publication
- Add/remove tables from publication

### 2. Logs Explorer Page

#### `/logs` - Logs Explorer

Features:
- Log type selector (API, Postgres, Auth, Storage, Realtime, Functions)
- Time range picker
- Level filter (info, warning, error)
- Text search
- Log detail panel
- Export functionality
- Auto-refresh toggle
- Pagination

### 3. Integrations Section Pages

#### `/integrations/webhooks` - Database Webhooks

Features:
- Webhook list with status
- Create webhook wizard
- Table and event selection
- HTTP configuration
- Delivery logs

#### `/integrations/cron` - Cron Jobs

Features:
- Job list with schedule display
- Create job with cron expression builder
- SQL or function selection
- Execution history
- Next run countdown

#### `/integrations/vault` - Vault Secrets

Features:
- Secret list (values hidden)
- Create/update secrets
- Delete with confirmation
- Usage documentation

### 4. Reports Page

#### `/reports` - Reports & Analytics

Features:
- API request volume chart
- Database query statistics
- Storage usage trends
- Auth activity overview
- Custom date range selection

## Implementation Priority

### Phase 1 (High Priority)
1. Database Policies page
2. Database Indexes page
3. Database Views page
4. Database Triggers page
5. Logs Explorer page

### Phase 2 (Medium Priority)
1. Database Roles page
2. Database Types page
3. Database Extensions page
4. Database Webhooks page
5. Cron Jobs page
6. Vault page

### Phase 3 (Lower Priority)
1. Database Publications page
2. Database Foreign Tables page
3. Reports page
4. GraphiQL integration

## Integration Test Structure

```go
// test/integration/dashboard_ui_test.go

// TestDashboardUI_PolicyManagement tests RLS policy UI operations
func TestDashboardUI_PolicyManagement(t *testing.T) {
    // Test policy CRUD through API
}

// TestDashboardUI_TriggerManagement tests trigger UI operations
func TestDashboardUI_TriggerManagement(t *testing.T) {
    // Test trigger CRUD through API
}

// TestDashboardUI_IndexManagement tests index UI operations
func TestDashboardUI_IndexManagement(t *testing.T) {
    // Test index CRUD through API
}

// TestDashboardUI_ViewManagement tests view UI operations
func TestDashboardUI_ViewManagement(t *testing.T) {
    // Test view CRUD through API
}

// TestDashboardUI_LogsExplorer tests logs explorer functionality
func TestDashboardUI_LogsExplorer(t *testing.T) {
    // Test log filtering, searching, export
}

// TestDashboardUI_WebhookManagement tests webhook functionality
func TestDashboardUI_WebhookManagement(t *testing.T) {
    // Test webhook CRUD and delivery
}

// TestDashboardUI_CronManagement tests cron job functionality
func TestDashboardUI_CronManagement(t *testing.T) {
    // Test cron CRUD and execution
}

// TestDashboardUI_VaultManagement tests vault/secrets functionality
func TestDashboardUI_VaultManagement(t *testing.T) {
    // Test secret CRUD
}
```

## Frontend Component Structure

```
src/
├── pages/
│   ├── Dashboard.tsx
│   ├── database/
│   │   ├── TableEditor.tsx
│   │   ├── SQLEditor.tsx
│   │   ├── Policies.tsx      (NEW)
│   │   ├── Triggers.tsx      (NEW)
│   │   ├── Indexes.tsx       (NEW)
│   │   ├── Views.tsx         (NEW)
│   │   ├── Roles.tsx         (NEW)
│   │   ├── Types.tsx         (NEW)
│   │   ├── Extensions.tsx    (NEW)
│   │   ├── Publications.tsx  (NEW)
│   │   └── ForeignTables.tsx (NEW)
│   ├── auth/
│   │   └── Users.tsx
│   ├── storage/
│   │   └── Storage.tsx
│   ├── functions/
│   │   └── Functions.tsx
│   ├── realtime/
│   │   └── Realtime.tsx
│   ├── integrations/
│   │   ├── Webhooks.tsx      (NEW)
│   │   ├── Cron.tsx          (NEW)
│   │   └── Vault.tsx         (NEW)
│   ├── logs/
│   │   └── LogsExplorer.tsx  (NEW)
│   ├── reports/
│   │   └── Reports.tsx       (NEW)
│   ├── settings/
│   │   └── Settings.tsx
│   └── ApiDocs.tsx
├── api/
│   ├── client.ts
│   ├── auth.ts
│   ├── database.ts
│   ├── storage.ts
│   ├── functions.ts
│   ├── realtime.ts
│   ├── dashboard.ts
│   ├── pgmeta.ts            (NEW)
│   ├── logs.ts              (NEW)
│   ├── webhooks.ts          (NEW)
│   ├── cron.ts              (NEW)
│   └── vault.ts             (NEW)
└── types/
    └── index.ts
```

## Success Criteria

1. All 200+ test cases pass
2. Frontend provides all Supabase Dashboard features
3. API compatibility with Supabase client libraries
4. No regression in existing functionality
5. Performance: Page load < 2s, API response < 500ms

## References

- [Supabase Dashboard](https://supabase.com)
- [Supabase Docs - Database](https://supabase.com/docs/guides/database/overview)
- [Supabase Docs - Auth](https://supabase.com/docs/guides/auth)
- [Supabase Docs - Storage](https://supabase.com/docs/guides/storage)
- [Supabase Docs - Edge Functions](https://supabase.com/docs/guides/functions)
- [Supabase Docs - Realtime](https://supabase.com/docs/guides/realtime)
- [Supabase Docs - Cron](https://supabase.com/docs/guides/cron)
- [Supabase Docs - Webhooks](https://supabase.com/docs/guides/database/webhooks)
- [Supabase Docs - Vault](https://supabase.com/docs/guides/database/vault)
