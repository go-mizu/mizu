# 0390: Supabase Dashboard 100% Compatibility Specification

## Overview

This specification documents all APIs required for 100% Supabase Dashboard compatibility, including the postgres-meta API, enhanced auth API, storage management, and dashboard-specific endpoints. It includes comprehensive test cases to verify compatibility.

## Reference Architecture

The Supabase Dashboard uses multiple API layers:

1. **postgres-meta API** - Database schema introspection and DDL operations
2. **GoTrue/Auth API** - User authentication and management
3. **Storage API** - S3-compatible object storage
4. **PostgREST API** - Auto-generated REST from PostgreSQL
5. **Realtime API** - WebSocket-based real-time messaging
6. **Management API** - Dashboard-specific endpoints

## Current Implementation Status

### Fully Implemented
- Auth API (GoTrue compatible)
- Storage API (Supabase Storage compatible)
- PostgREST API (basic CRUD)
- Functions API (Edge Functions compatible)
- Realtime API (Phoenix protocol)
- Dashboard stats/health endpoints

### Missing for Full Dashboard Compatibility

The following postgres-meta endpoints are required by Supabase Dashboard but not yet implemented:

## postgres-meta API Specification

### Base Path: `/api/pg`

All postgres-meta endpoints require `service_role` authentication.

---

### 1. Database Version & Config

#### GET `/api/pg/config/version`
Returns PostgreSQL version information.

**Response:**
```json
{
  "version": "PostgreSQL 15.4",
  "version_number": 150400,
  "active_connections": 5,
  "max_connections": 100
}
```

---

### 2. Indexes

#### GET `/api/pg/indexes`
List all indexes with optional schema filter.

**Query Parameters:**
- `included_schemas` (optional) - Comma-separated schema list
- `include_columns` (optional) - Include column details

**Response:**
```json
[
  {
    "id": 16391,
    "schema": "public",
    "table": "users",
    "name": "users_pkey",
    "columns": ["id"],
    "is_unique": true,
    "is_primary": true,
    "is_exclusion": false,
    "is_valid": true,
    "index_definition": "CREATE UNIQUE INDEX users_pkey ON public.users USING btree (id)"
  }
]
```

#### POST `/api/pg/indexes`
Create a new index.

**Request:**
```json
{
  "schema": "public",
  "table": "users",
  "name": "users_email_idx",
  "columns": ["email"],
  "unique": false,
  "using": "btree"
}
```

#### DELETE `/api/pg/indexes/{id}`
Drop an index by ID.

---

### 3. Views

#### GET `/api/pg/views`
List all views.

**Query Parameters:**
- `included_schemas` (optional) - Comma-separated schema list

**Response:**
```json
[
  {
    "id": 16401,
    "schema": "public",
    "name": "active_users",
    "is_updatable": true,
    "comment": "Users who have logged in within 30 days",
    "definition": "SELECT * FROM users WHERE last_sign_in > now() - interval '30 days'"
  }
]
```

#### POST `/api/pg/views`
Create a new view.

**Request:**
```json
{
  "schema": "public",
  "name": "active_users",
  "definition": "SELECT * FROM users WHERE deleted_at IS NULL",
  "check_option": "local"
}
```

#### PATCH `/api/pg/views/{id}`
Update a view (CREATE OR REPLACE).

#### DELETE `/api/pg/views/{id}`
Drop a view.

---

### 4. Materialized Views

#### GET `/api/pg/materialized-views`
List all materialized views.

**Response:**
```json
[
  {
    "id": 16410,
    "schema": "public",
    "name": "user_stats",
    "is_populated": true,
    "definition": "SELECT user_id, count(*) as orders FROM orders GROUP BY user_id"
  }
]
```

#### POST `/api/pg/materialized-views`
Create a materialized view.

#### POST `/api/pg/materialized-views/{id}/refresh`
Refresh a materialized view.

#### DELETE `/api/pg/materialized-views/{id}`
Drop a materialized view.

---

### 5. Foreign Tables

#### GET `/api/pg/foreign-tables`
List foreign tables (for postgres_fdw, etc.).

**Response:**
```json
[
  {
    "id": 16420,
    "schema": "public",
    "name": "remote_users",
    "server": "remote_server",
    "columns": [
      {"name": "id", "type": "integer"},
      {"name": "email", "type": "text"}
    ]
  }
]
```

---

### 6. Triggers

#### GET `/api/pg/triggers`
List all triggers.

**Response:**
```json
[
  {
    "id": 16430,
    "name": "set_updated_at",
    "schema": "public",
    "table": "users",
    "function_schema": "public",
    "function_name": "update_modified_column",
    "events": ["UPDATE"],
    "orientation": "ROW",
    "timing": "BEFORE",
    "condition": null,
    "enabled": true
  }
]
```

#### POST `/api/pg/triggers`
Create a trigger.

**Request:**
```json
{
  "name": "audit_changes",
  "schema": "public",
  "table": "orders",
  "function_schema": "public",
  "function_name": "log_audit",
  "events": ["INSERT", "UPDATE", "DELETE"],
  "timing": "AFTER",
  "orientation": "ROW"
}
```

#### DELETE `/api/pg/triggers/{id}`
Drop a trigger.

---

### 7. Types (Custom Types/Enums)

#### GET `/api/pg/types`
List custom types including enums.

**Response:**
```json
[
  {
    "id": 16440,
    "schema": "public",
    "name": "order_status",
    "type": "enum",
    "enums": ["pending", "processing", "completed", "cancelled"],
    "comment": "Order status enumeration"
  }
]
```

#### POST `/api/pg/types`
Create a custom type.

**Request (Enum):**
```json
{
  "schema": "public",
  "name": "priority",
  "type": "enum",
  "values": ["low", "medium", "high"]
}
```

#### DELETE `/api/pg/types/{id}`
Drop a type.

---

### 8. Roles

#### GET `/api/pg/roles`
List database roles.

**Response:**
```json
[
  {
    "id": 10,
    "name": "postgres",
    "is_superuser": true,
    "can_create_role": true,
    "can_create_db": true,
    "can_login": true,
    "is_replication_role": false,
    "inherit_role": true,
    "config": {}
  },
  {
    "id": 16388,
    "name": "anon",
    "is_superuser": false,
    "can_login": false,
    "inherit_role": true
  },
  {
    "id": 16389,
    "name": "authenticated",
    "is_superuser": false,
    "can_login": false,
    "inherit_role": true
  }
]
```

#### POST `/api/pg/roles`
Create a role.

**Request:**
```json
{
  "name": "app_user",
  "is_superuser": false,
  "can_login": true,
  "password": "secure_password",
  "inherit_role": true
}
```

#### PATCH `/api/pg/roles/{id}`
Update a role.

#### DELETE `/api/pg/roles/{id}`
Drop a role.

---

### 9. Publications (Replication)

#### GET `/api/pg/publications`
List publications for logical replication.

**Response:**
```json
[
  {
    "id": 16450,
    "name": "supabase_realtime",
    "owner": "postgres",
    "tables": [
      {"schema": "public", "name": "messages"}
    ],
    "all_tables": false,
    "insert": true,
    "update": true,
    "delete": true,
    "truncate": true
  }
]
```

#### POST `/api/pg/publications`
Create a publication.

#### PATCH `/api/pg/publications/{id}`
Update a publication.

#### DELETE `/api/pg/publications/{id}`
Drop a publication.

---

### 10. Privileges

#### GET `/api/pg/table-privileges`
List table-level privileges.

**Response:**
```json
[
  {
    "schema": "public",
    "table": "users",
    "grantee": "authenticated",
    "privileges": ["SELECT", "INSERT", "UPDATE"],
    "is_grantable": false
  }
]
```

#### GET `/api/pg/column-privileges`
List column-level privileges.

**Response:**
```json
[
  {
    "schema": "public",
    "table": "users",
    "column": "email",
    "grantee": "anon",
    "privilege_type": "SELECT"
  }
]
```

---

### 11. SQL Utilities

#### POST `/api/pg/format`
Format SQL query.

**Request:**
```json
{
  "query": "select * from users where id=1"
}
```

**Response:**
```json
{
  "formatted": "SELECT *\nFROM users\nWHERE id = 1"
}
```

#### POST `/api/pg/parse`
Parse SQL into AST.

**Request:**
```json
{
  "query": "SELECT * FROM users"
}
```

#### POST `/api/pg/explain`
Explain query execution plan.

**Request:**
```json
{
  "query": "SELECT * FROM users WHERE id = 1",
  "analyze": true,
  "buffers": true,
  "format": "json"
}
```

---

### 12. Table Constraints

#### GET `/api/pg/constraints`
List all constraints (primary key, foreign key, check, unique).

**Query Parameters:**
- `table_id` (optional) - Filter by table

**Response:**
```json
[
  {
    "id": 16460,
    "schema": "public",
    "table": "orders",
    "name": "orders_user_id_fkey",
    "type": "FOREIGN KEY",
    "definition": "FOREIGN KEY (user_id) REFERENCES users(id)",
    "columns": ["user_id"],
    "ref_schema": "public",
    "ref_table": "users",
    "ref_columns": ["id"],
    "on_update": "NO ACTION",
    "on_delete": "CASCADE"
  }
]
```

#### POST `/api/pg/constraints`
Add a constraint.

#### DELETE `/api/pg/constraints/{id}`
Drop a constraint.

---

### 13. Primary Keys

#### GET `/api/pg/primary-keys`
List primary keys.

**Response:**
```json
[
  {
    "schema": "public",
    "table": "users",
    "name": "users_pkey",
    "columns": ["id"]
  }
]
```

---

### 14. Foreign Keys

#### GET `/api/pg/foreign-keys`
List foreign key relationships.

**Response:**
```json
[
  {
    "id": 16470,
    "schema": "public",
    "table": "orders",
    "name": "orders_user_id_fkey",
    "columns": ["user_id"],
    "target_schema": "public",
    "target_table": "users",
    "target_columns": ["id"],
    "on_update": "NO ACTION",
    "on_delete": "CASCADE"
  }
]
```

---

### 15. Table Relationships

#### GET `/api/pg/relationships`
Get table relationships (combines foreign keys for UI).

**Response:**
```json
[
  {
    "id": 16470,
    "source_schema": "public",
    "source_table": "orders",
    "source_columns": ["user_id"],
    "target_schema": "public",
    "target_table": "users",
    "target_columns": ["id"],
    "constraint_name": "orders_user_id_fkey"
  }
]
```

---

## Enhanced Auth API Endpoints

### Additional Admin Endpoints

#### POST `/auth/v1/admin/users/{id}/mfa`
Manage user MFA settings.

**Request:**
```json
{
  "enabled": false,
  "remove_factors": true
}
```

#### POST `/auth/v1/admin/generate_link`
Generate authentication links (invite, recovery, magic link).

**Request:**
```json
{
  "type": "invite",
  "email": "user@example.com",
  "data": {"role": "admin"},
  "redirect_to": "https://app.example.com/welcome"
}
```

**Response:**
```json
{
  "action_link": "https://project.supabase.co/auth/v1/verify?token=...",
  "email_otp": "123456",
  "hashed_token": "...",
  "verification_type": "invite",
  "redirect_to": "https://app.example.com/welcome"
}
```

#### GET `/auth/v1/admin/audit`
Get authentication audit logs.

**Query Parameters:**
- `page` (default: 1)
- `per_page` (default: 50)
- `user_id` (optional)

**Response:**
```json
{
  "audit_logs": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "action": "login",
      "ip_address": "1.2.3.4",
      "user_agent": "...",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 100
}
```

---

## Enhanced Storage API Endpoints

### Additional Endpoints

#### GET `/storage/v1/bucket/{id}/size`
Get bucket size statistics.

**Response:**
```json
{
  "bucket_id": "avatars",
  "total_size": 1048576000,
  "object_count": 250
}
```

#### POST `/storage/v1/object/search/{bucket}`
Search objects with advanced filtering.

**Request:**
```json
{
  "prefix": "uploads/",
  "search": "profile",
  "limit": 100,
  "offset": 0,
  "sortBy": {
    "column": "created_at",
    "order": "desc"
  }
}
```

---

## Enhanced Dashboard API Endpoints

### Base Path: `/api/dashboard`

#### GET `/api/dashboard/stats`
Extended dashboard statistics.

**Response:**
```json
{
  "users": {
    "total": 1250,
    "active_today": 45,
    "new_this_week": 23
  },
  "storage": {
    "buckets": 5,
    "total_size": 5242880000,
    "objects": 1500
  },
  "database": {
    "tables": 12,
    "total_rows": 50000,
    "schemas": ["public", "auth", "storage"]
  },
  "functions": {
    "total": 8,
    "active": 6,
    "invocations_today": 1234
  },
  "realtime": {
    "active_connections": 25,
    "channels": 10
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### GET `/api/dashboard/health`
Extended health check.

**Response:**
```json
{
  "status": "healthy",
  "services": {
    "database": {
      "status": "healthy",
      "version": "PostgreSQL 15.4",
      "latency_ms": 2
    },
    "auth": {
      "status": "healthy",
      "version": "2.40.0"
    },
    "storage": {
      "status": "healthy",
      "type": "local"
    },
    "realtime": {
      "status": "healthy",
      "connections": 25
    }
  },
  "version": "1.0.0",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### GET `/api/dashboard/logs`
Get recent system logs.

**Query Parameters:**
- `source` - Log source (auth, storage, database, functions)
- `level` - Log level (debug, info, warn, error)
- `limit` (default: 100)
- `since` - ISO timestamp

**Response:**
```json
{
  "logs": [
    {
      "id": "uuid",
      "source": "auth",
      "level": "info",
      "message": "User login successful",
      "metadata": {"user_id": "uuid"},
      "timestamp": "2024-01-01T12:00:00Z"
    }
  ]
}
```

---

## TypeScript Type Generators

### GET `/api/pg/generators/typescript`
Generate TypeScript types from database schema.

**Query Parameters:**
- `included_schemas` (default: "public")
- `detect_one_to_one` (default: false)

**Response:**
```typescript
export type Json = string | number | boolean | null | { [key: string]: Json } | Json[];

export interface Database {
  public: {
    Tables: {
      users: {
        Row: {
          id: string;
          email: string;
          created_at: string;
        };
        Insert: {
          id?: string;
          email: string;
          created_at?: string;
        };
        Update: {
          id?: string;
          email?: string;
          created_at?: string;
        };
      };
    };
  };
}
```

### GET `/api/pg/generators/openapi`
Generate OpenAPI specification from database schema.

---

## Test Cases

### 1. postgres-meta API Tests

```go
// TestPGMeta_Indexes tests index management
func TestPGMeta_Indexes(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    // List indexes
    t.Run("list indexes", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/indexes?included_schemas=public", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var indexes []map[string]any
        require.NoError(t, json.Unmarshal(body, &indexes))
        assert.GreaterOrEqual(t, len(indexes), 0)
    })

    // Create index
    t.Run("create index", func(t *testing.T) {
        reqBody := map[string]any{
            "schema":  "public",
            "table":   "users",
            "name":    fmt.Sprintf("test_idx_%d", time.Now().UnixNano()),
            "columns": []string{"email"},
            "unique":  false,
        }
        status, body, _, err := client.Request("POST", "/api/pg/indexes", reqBody, nil)
        require.NoError(t, err)
        if status == 201 {
            var idx map[string]any
            require.NoError(t, json.Unmarshal(body, &idx))
            assert.NotEmpty(t, idx["id"])

            // Cleanup
            client.Request("DELETE", fmt.Sprintf("/api/pg/indexes/%v", idx["id"]), nil, nil)
        }
    })
}

// TestPGMeta_Views tests view management
func TestPGMeta_Views(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    // List views
    t.Run("list views", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/views?included_schemas=public", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var views []map[string]any
        require.NoError(t, json.Unmarshal(body, &views))
    })

    // Create and drop view
    t.Run("create view", func(t *testing.T) {
        viewName := fmt.Sprintf("test_view_%d", time.Now().UnixNano())
        reqBody := map[string]any{
            "schema":     "public",
            "name":       viewName,
            "definition": "SELECT 1 as value",
        }
        status, body, _, err := client.Request("POST", "/api/pg/views", reqBody, nil)
        require.NoError(t, err)
        if status == 201 {
            var view map[string]any
            require.NoError(t, json.Unmarshal(body, &view))

            // Cleanup
            client.Request("DELETE", fmt.Sprintf("/api/pg/views/%v", view["id"]), nil, nil)
        }
    })
}

// TestPGMeta_Triggers tests trigger management
func TestPGMeta_Triggers(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("list triggers", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/triggers", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var triggers []map[string]any
        require.NoError(t, json.Unmarshal(body, &triggers))
    })
}

// TestPGMeta_Types tests custom type management
func TestPGMeta_Types(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("list types", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/types", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var types []map[string]any
        require.NoError(t, json.Unmarshal(body, &types))
    })

    t.Run("create enum type", func(t *testing.T) {
        typeName := fmt.Sprintf("test_enum_%d", time.Now().UnixNano())
        reqBody := map[string]any{
            "schema": "public",
            "name":   typeName,
            "type":   "enum",
            "values": []string{"a", "b", "c"},
        }
        status, body, _, err := client.Request("POST", "/api/pg/types", reqBody, nil)
        require.NoError(t, err)
        if status == 201 {
            var typ map[string]any
            require.NoError(t, json.Unmarshal(body, &typ))
            client.Request("DELETE", fmt.Sprintf("/api/pg/types/%v", typ["id"]), nil, nil)
        }
    })
}

// TestPGMeta_Roles tests role management
func TestPGMeta_Roles(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("list roles", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/roles", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var roles []map[string]any
        require.NoError(t, json.Unmarshal(body, &roles))
        assert.GreaterOrEqual(t, len(roles), 1) // At least postgres role
    })

    t.Run("verify standard roles exist", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/roles", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var roles []map[string]any
        require.NoError(t, json.Unmarshal(body, &roles))

        roleNames := make(map[string]bool)
        for _, r := range roles {
            roleNames[r["name"].(string)] = true
        }

        // Supabase standard roles
        assert.True(t, roleNames["anon"] || roleNames["postgres"])
    })
}

// TestPGMeta_Publications tests publication management
func TestPGMeta_Publications(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("list publications", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/publications", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var pubs []map[string]any
        require.NoError(t, json.Unmarshal(body, &pubs))
    })
}

// TestPGMeta_Privileges tests privilege listing
func TestPGMeta_Privileges(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("list table privileges", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/table-privileges", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var privs []map[string]any
        require.NoError(t, json.Unmarshal(body, &privs))
    })

    t.Run("list column privileges", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/column-privileges", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var privs []map[string]any
        require.NoError(t, json.Unmarshal(body, &privs))
    })
}

// TestPGMeta_Config tests database config endpoints
func TestPGMeta_Config(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("get version", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/config/version", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var version map[string]any
        require.NoError(t, json.Unmarshal(body, &version))
        assert.Contains(t, version["version"], "PostgreSQL")
    })
}

// TestPGMeta_SQLUtilities tests SQL utility endpoints
func TestPGMeta_SQLUtilities(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("format SQL", func(t *testing.T) {
        reqBody := map[string]any{
            "query": "select * from users where id=1",
        }
        status, body, _, err := client.Request("POST", "/api/pg/format", reqBody, nil)
        require.NoError(t, err)
        if status == 200 {
            var result map[string]any
            require.NoError(t, json.Unmarshal(body, &result))
            assert.NotEmpty(t, result["formatted"])
        }
    })

    t.Run("explain query", func(t *testing.T) {
        reqBody := map[string]any{
            "query":  "SELECT 1",
            "format": "json",
        }
        status, body, _, err := client.Request("POST", "/api/pg/explain", reqBody, nil)
        require.NoError(t, err)
        if status == 200 {
            var result []map[string]any
            require.NoError(t, json.Unmarshal(body, &result))
        }
    })
}

// TestPGMeta_Constraints tests constraint management
func TestPGMeta_Constraints(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("list constraints", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/constraints", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var constraints []map[string]any
        require.NoError(t, json.Unmarshal(body, &constraints))
    })

    t.Run("list primary keys", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/primary-keys", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var pks []map[string]any
        require.NoError(t, json.Unmarshal(body, &pks))
    })

    t.Run("list foreign keys", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/foreign-keys", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var fks []map[string]any
        require.NoError(t, json.Unmarshal(body, &fks))
    })

    t.Run("list relationships", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/relationships", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var rels []map[string]any
        require.NoError(t, json.Unmarshal(body, &rels))
    })
}

// TestPGMeta_TypeGenerators tests TypeScript/OpenAPI generation
func TestPGMeta_TypeGenerators(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("generate typescript types", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/generators/typescript?included_schemas=public", nil, nil)
        require.NoError(t, err)
        if status == 200 {
            assert.Contains(t, string(body), "export")
        }
    })

    t.Run("generate openapi spec", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/generators/openapi", nil, nil)
        require.NoError(t, err)
        if status == 200 {
            var spec map[string]any
            require.NoError(t, json.Unmarshal(body, &spec))
            assert.Contains(t, spec, "openapi")
        }
    })
}
```

### 2. Enhanced Dashboard Tests

```go
// TestDashboard_ExtendedStats tests enhanced statistics
func TestDashboard_ExtendedStats(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("get extended stats", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/dashboard/stats", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var stats map[string]any
        require.NoError(t, json.Unmarshal(body, &stats))

        // Verify all sections present
        assert.Contains(t, stats, "users")
        assert.Contains(t, stats, "storage")
        assert.Contains(t, stats, "database")
        assert.Contains(t, stats, "functions")
        assert.Contains(t, stats, "timestamp")

        // Verify users section
        users := stats["users"].(map[string]any)
        assert.Contains(t, users, "total")

        // Verify database section
        db := stats["database"].(map[string]any)
        assert.Contains(t, db, "tables")
    })

    t.Run("stats unauthorized without service role", func(t *testing.T) {
        anonClient := NewClient(localbaseURL, localbaseAPIKey)
        status, _, _, err := anonClient.Request("GET", "/api/dashboard/stats", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 403, status)
    })
}

// TestDashboard_ExtendedHealth tests enhanced health check
func TestDashboard_ExtendedHealth(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("get extended health", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/dashboard/health", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var health map[string]any
        require.NoError(t, json.Unmarshal(body, &health))

        assert.Contains(t, health, "status")
        assert.Contains(t, health, "services")
        assert.Contains(t, health, "version")

        services := health["services"].(map[string]any)
        assert.Contains(t, services, "database")
        assert.Contains(t, services, "auth")
        assert.Contains(t, services, "storage")
    })
}

// TestDashboard_Logs tests log retrieval
func TestDashboard_Logs(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("get logs", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/dashboard/logs?limit=10", nil, nil)
        require.NoError(t, err)
        if status == 200 {
            var result map[string]any
            require.NoError(t, json.Unmarshal(body, &result))
            assert.Contains(t, result, "logs")
        }
    })

    t.Run("filter logs by source", func(t *testing.T) {
        status, _, _, err := client.Request("GET", "/api/dashboard/logs?source=auth&limit=10", nil, nil)
        require.NoError(t, err)
        // 200 or 404 both acceptable
        assert.Contains(t, []int{200, 404}, status)
    })
}
```

### 3. Enhanced Auth Tests

```go
// TestAuth_GenerateLink tests admin link generation
func TestAuth_GenerateLink(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("generate invite link", func(t *testing.T) {
        email := fmt.Sprintf("invite-%d@example.com", time.Now().UnixNano())
        reqBody := map[string]any{
            "type":  "invite",
            "email": email,
        }
        status, body, _, err := client.Request("POST", "/auth/v1/admin/generate_link", reqBody, nil)
        require.NoError(t, err)
        if status == 200 || status == 201 {
            var result map[string]any
            require.NoError(t, json.Unmarshal(body, &result))
            assert.Contains(t, result, "action_link")
        }
    })

    t.Run("generate recovery link", func(t *testing.T) {
        // First create a user
        email := fmt.Sprintf("recovery-%d@example.com", time.Now().UnixNano())
        createBody := map[string]any{
            "email":    email,
            "password": "password123",
        }
        status, _, _, _ := client.Request("POST", "/auth/v1/admin/users", createBody, nil)
        if status == 201 {
            reqBody := map[string]any{
                "type":  "recovery",
                "email": email,
            }
            status, body, _, err := client.Request("POST", "/auth/v1/admin/generate_link", reqBody, nil)
            require.NoError(t, err)
            if status == 200 {
                var result map[string]any
                require.NoError(t, json.Unmarshal(body, &result))
                assert.Contains(t, result, "action_link")
            }
        }
    })

    t.Run("requires service role", func(t *testing.T) {
        anonClient := NewClient(localbaseURL, localbaseAPIKey)
        reqBody := map[string]any{
            "type":  "invite",
            "email": "test@example.com",
        }
        status, _, _, err := anonClient.Request("POST", "/auth/v1/admin/generate_link", reqBody, nil)
        require.NoError(t, err)
        require.Equal(t, 403, status)
    })
}

// TestAuth_AuditLogs tests audit log retrieval
func TestAuth_AuditLogs(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("get audit logs", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/auth/v1/admin/audit?per_page=10", nil, nil)
        require.NoError(t, err)
        if status == 200 {
            var result map[string]any
            require.NoError(t, json.Unmarshal(body, &result))
            assert.Contains(t, result, "audit_logs")
        }
    })

    t.Run("filter by user_id", func(t *testing.T) {
        userID := "00000000-0000-0000-0000-000000000001"
        status, _, _, err := client.Request("GET", "/auth/v1/admin/audit?user_id="+userID, nil, nil)
        require.NoError(t, err)
        assert.Contains(t, []int{200, 404}, status)
    })
}
```

### 4. Enhanced Storage Tests

```go
// TestStorage_BucketSize tests bucket size statistics
func TestStorage_BucketSize(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    // Create a test bucket first
    bucketName := fmt.Sprintf("size-test-%d", time.Now().UnixNano())
    createBody := map[string]any{
        "name":   bucketName,
        "public": false,
    }
    status, _, _, err := client.Request("POST", "/storage/v1/bucket", createBody, nil)
    require.NoError(t, err)
    if status != 200 {
        t.Skip("Could not create test bucket")
    }
    defer client.Request("DELETE", "/storage/v1/bucket/"+bucketName, nil, nil)

    t.Run("get bucket size", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/storage/v1/bucket/"+bucketName+"/size", nil, nil)
        require.NoError(t, err)
        if status == 200 {
            var result map[string]any
            require.NoError(t, json.Unmarshal(body, &result))
            assert.Contains(t, result, "total_size")
            assert.Contains(t, result, "object_count")
        }
    })
}

// TestStorage_ObjectSearch tests advanced object search
func TestStorage_ObjectSearch(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    bucketName := fmt.Sprintf("search-test-%d", time.Now().UnixNano())
    createBody := map[string]any{
        "name":   bucketName,
        "public": true,
    }
    status, _, _, _ := client.Request("POST", "/storage/v1/bucket", createBody, nil)
    if status != 200 {
        t.Skip("Could not create test bucket")
    }
    defer client.Request("DELETE", "/storage/v1/bucket/"+bucketName, nil, nil)

    t.Run("search objects", func(t *testing.T) {
        reqBody := map[string]any{
            "prefix": "",
            "limit":  100,
            "sortBy": map[string]any{
                "column": "name",
                "order":  "asc",
            },
        }
        status, body, _, err := client.Request("POST", "/storage/v1/object/search/"+bucketName, reqBody, nil)
        require.NoError(t, err)
        if status == 200 {
            var results []map[string]any
            require.NoError(t, json.Unmarshal(body, &results))
        }
    })
}
```

### 5. Materialized View Tests

```go
// TestPGMeta_MaterializedViews tests materialized view management
func TestPGMeta_MaterializedViews(t *testing.T) {
    client := NewClient(localbaseURL, serviceRoleKey)

    t.Run("list materialized views", func(t *testing.T) {
        status, body, _, err := client.Request("GET", "/api/pg/materialized-views", nil, nil)
        require.NoError(t, err)
        require.Equal(t, 200, status)

        var mvs []map[string]any
        require.NoError(t, json.Unmarshal(body, &mvs))
    })

    t.Run("create and refresh materialized view", func(t *testing.T) {
        mvName := fmt.Sprintf("test_mv_%d", time.Now().UnixNano())
        reqBody := map[string]any{
            "schema":     "public",
            "name":       mvName,
            "definition": "SELECT 1 as value",
        }
        status, body, _, err := client.Request("POST", "/api/pg/materialized-views", reqBody, nil)
        require.NoError(t, err)
        if status == 201 {
            var mv map[string]any
            require.NoError(t, json.Unmarshal(body, &mv))
            mvID := mv["id"]

            // Refresh
            refreshStatus, _, _, _ := client.Request("POST", fmt.Sprintf("/api/pg/materialized-views/%v/refresh", mvID), nil, nil)
            assert.Contains(t, []int{200, 204}, refreshStatus)

            // Cleanup
            client.Request("DELETE", fmt.Sprintf("/api/pg/materialized-views/%v", mvID), nil, nil)
        }
    })
}
```

---

## Implementation Checklist

### postgres-meta API
- [ ] `GET /api/pg/config/version` - Database version
- [ ] `GET /api/pg/indexes` - List indexes
- [ ] `POST /api/pg/indexes` - Create index
- [ ] `DELETE /api/pg/indexes/{id}` - Drop index
- [ ] `GET /api/pg/views` - List views
- [ ] `POST /api/pg/views` - Create view
- [ ] `PATCH /api/pg/views/{id}` - Update view
- [ ] `DELETE /api/pg/views/{id}` - Drop view
- [ ] `GET /api/pg/materialized-views` - List materialized views
- [ ] `POST /api/pg/materialized-views` - Create materialized view
- [ ] `POST /api/pg/materialized-views/{id}/refresh` - Refresh
- [ ] `DELETE /api/pg/materialized-views/{id}` - Drop materialized view
- [ ] `GET /api/pg/foreign-tables` - List foreign tables
- [ ] `GET /api/pg/triggers` - List triggers
- [ ] `POST /api/pg/triggers` - Create trigger
- [ ] `DELETE /api/pg/triggers/{id}` - Drop trigger
- [ ] `GET /api/pg/types` - List custom types
- [ ] `POST /api/pg/types` - Create type
- [ ] `DELETE /api/pg/types/{id}` - Drop type
- [ ] `GET /api/pg/roles` - List roles
- [ ] `POST /api/pg/roles` - Create role
- [ ] `PATCH /api/pg/roles/{id}` - Update role
- [ ] `DELETE /api/pg/roles/{id}` - Drop role
- [ ] `GET /api/pg/publications` - List publications
- [ ] `POST /api/pg/publications` - Create publication
- [ ] `DELETE /api/pg/publications/{id}` - Drop publication
- [ ] `GET /api/pg/table-privileges` - List table privileges
- [ ] `GET /api/pg/column-privileges` - List column privileges
- [ ] `POST /api/pg/format` - Format SQL
- [ ] `POST /api/pg/parse` - Parse SQL
- [ ] `POST /api/pg/explain` - Explain query
- [ ] `GET /api/pg/constraints` - List constraints
- [ ] `POST /api/pg/constraints` - Add constraint
- [ ] `DELETE /api/pg/constraints/{id}` - Drop constraint
- [ ] `GET /api/pg/primary-keys` - List primary keys
- [ ] `GET /api/pg/foreign-keys` - List foreign keys
- [ ] `GET /api/pg/relationships` - Get relationships
- [ ] `GET /api/pg/generators/typescript` - Generate TypeScript
- [ ] `GET /api/pg/generators/openapi` - Generate OpenAPI

### Enhanced Auth API
- [ ] `POST /auth/v1/admin/users/{id}/mfa` - Manage MFA
- [ ] `POST /auth/v1/admin/generate_link` - Generate links
- [ ] `GET /auth/v1/admin/audit` - Audit logs

### Enhanced Storage API
- [ ] `GET /storage/v1/bucket/{id}/size` - Bucket size
- [ ] `POST /storage/v1/object/search/{bucket}` - Object search

### Enhanced Dashboard API
- [ ] Extended `/api/dashboard/stats` with more metrics
- [ ] Extended `/api/dashboard/health` with service details
- [ ] `GET /api/dashboard/logs` - System logs

---

## Compatibility Matrix

| Feature | Supabase Dashboard | Localbase | Status |
|---------|-------------------|-----------|--------|
| Table Editor | Yes | Yes | Implemented |
| SQL Editor | Yes | Yes | Implemented |
| User Management | Yes | Yes | Implemented |
| Storage Management | Yes | Yes | Implemented |
| Edge Functions | Yes | Yes | Implemented |
| Realtime Inspector | Yes | Yes | Implemented |
| Index Management | Yes | Pending | Spec Ready |
| View Management | Yes | Pending | Spec Ready |
| Trigger Management | Yes | Pending | Spec Ready |
| Role Management | Yes | Pending | Spec Ready |
| Type/Enum Editor | Yes | Pending | Spec Ready |
| Publication Editor | Yes | Pending | Spec Ready |
| TypeScript Generator | Yes | Pending | Spec Ready |
| OpenAPI Generator | Yes | Pending | Spec Ready |
| Audit Logs | Yes | Pending | Spec Ready |

---

## Summary

This specification provides a complete roadmap for achieving 100% Supabase Dashboard compatibility. The key additions are:

1. **postgres-meta API** - Full schema introspection and DDL operations
2. **Enhanced Dashboard Stats** - More detailed statistics and health info
3. **Audit Logging** - Track authentication events
4. **Type Generators** - TypeScript and OpenAPI generation

Implementation priority:
1. Core postgres-meta endpoints (indexes, views, triggers, types, roles)
2. Constraint and relationship endpoints
3. Type generators
4. Enhanced dashboard stats
5. Audit logging
