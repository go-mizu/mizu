# RFC 0383: Supabase Service Roles and RLS Compatibility

**Status**: Active (Implementation Complete)
**Created**: 2026-01-16
**Updated**: 2026-01-16
**Component**: blueprints/localbase

## Summary

This document specifies the implementation requirements for 100% Supabase compatibility in the Localbase blueprint, focusing on:
1. API key authentication (anon vs service_role)
2. JWT validation and claims propagation
3. Row Level Security (RLS) enforcement
4. Storage access control
5. Security hardening

## Background

### How Supabase API Keys Work

Supabase uses two primary API keys:

| Key Type | Role | RLS Behavior | Use Case |
|----------|------|--------------|----------|
| `anon` (publishable) | `anon` | Subject to RLS policies | Client-side, browser code |
| `service_role` (secret) | `service_role` | **Bypasses all RLS** | Server-side, trusted code |

Both keys are JWTs signed with the project's JWT secret. The JWT payload contains:

```json
{
  "iss": "supabase",
  "role": "anon",        // or "service_role"
  "exp": 1983812996,
  "iat": 1698412996
}
```

### How Supabase RLS Works

1. **Policy Creation**: SQL policies define access rules per table/operation
2. **JWT Context**: The `request.jwt.claims` PostgreSQL GUC stores the current user's JWT
3. **Helper Functions**: `auth.uid()` and `auth.role()` read from JWT claims
4. **Role Switching**: Database connection uses appropriate role (`anon`, `authenticated`, or `service_role`)

```sql
-- Example RLS policy
CREATE POLICY "Users can view own data" ON todos
  FOR SELECT USING (auth.uid() = user_id);

-- auth.uid() implementation
CREATE FUNCTION auth.uid() RETURNS UUID AS $$
  SELECT NULLIF(current_setting('request.jwt.claims', TRUE)::json->>'sub', '')::UUID
$$ LANGUAGE SQL STABLE;
```

## Current State Analysis

### 1. API Key Middleware - IMPLEMENTED ✓

**File**: `app/web/middleware/apikey.go`

| Feature | Status | Implementation |
|---------|--------|----------------|
| JWT signature validation | ✓ DONE | `validateAndParseJWT()` with HMAC-SHA256 |
| Expiration checking | ✓ DONE | Checks `exp` claim against current time |
| Full claims extraction | ✓ DONE | Extracts sub, role, email, app_metadata, user_metadata, etc. |
| Header propagation | ✓ DONE | Sets X-Localbase-Role, X-Localbase-JWT-Claims, X-Localbase-User-ID |
| Configurable via env | ✓ DONE | LOCALBASE_JWT_SECRET, LOCALBASE_VALIDATE_JWT |

**Current Flow**:
```
Request → Extract JWT (apikey or Bearer) → Validate signature →
Check expiration → Extract all claims → Set headers → Continue
```

### 2. Database Handler - IMPLEMENTED ✓

**File**: `pkg/postgrest/handler.go`, `store/postgres/database.go`

| Feature | Status | Implementation |
|---------|--------|----------------|
| JWT claims propagation | ✓ DONE | `setRLSContext()` sets GUC variables |
| Service role bypass | ✓ DONE | Direct query for service_role |
| RLS-aware queries | ✓ DONE | `QueryWithRLS()`, `ExecWithRLS()` |
| GUC variables | ✓ DONE | request.jwt.claims, request.jwt.claim.sub/role/email |

**Implementation**:
```go
// For non-service_role, sets up transaction with RLS context
func (s *DatabaseStore) QueryWithRLS(ctx context.Context, rlsCtx *RLSContext, sql string, params ...interface{}) (*QueryResult, error) {
    if rlsCtx == nil || rlsCtx.Role == "service_role" {
        return s.Query(ctx, sql, params...) // Bypass RLS
    }
    // Set GUC variables in transaction for RLS policy evaluation
    tx, _ := s.pool.Begin(ctx)
    s.setRLSContext(ctx, tx, rlsCtx)
    return s.executeInTransaction(ctx, tx, sql, params...)
}
```

### 3. Storage Handler - IMPLEMENTED ✓

**File**: `app/web/handler/api/storage.go`

| Feature | Status | Implementation |
|---------|--------|----------------|
| Service role bypass | ✓ DONE | Full access for service_role |
| Public bucket access | ✓ DONE | Anon can read public buckets |
| Private bucket protection | ✓ DONE | Requires authentication |
| User folder pattern | ✓ DONE | Users can access paths containing their ID |
| Bucket visibility | ✓ DONE | ListBuckets filters by role |

**Note**: Currently uses app-level RLS rather than PostgreSQL policies on storage.objects.

### 4. Auth Handler - IMPLEMENTED ✓

**File**: `app/web/handler/api/auth.go`, `app/web/server.go`

| Feature | Status | Implementation |
|---------|--------|----------------|
| Admin endpoint protection | ✓ DONE | RequireServiceRole() middleware |
| Configurable JWT secret | ✓ DONE | Via LOCALBASE_JWT_SECRET env var |
| Admin user management | ✓ DONE | List, create, update, delete users |

## Specification

### 1. JWT Validation Implementation

```go
// ValidateJWT validates a JWT and returns claims
func ValidateJWT(token string, secret []byte) (*JWTClaims, error) {
    parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return secret, nil
    })

    if err != nil {
        return nil, fmt.Errorf("invalid JWT: %w", err)
    }

    claims, ok := parsedToken.Claims.(jwt.MapClaims)
    if !ok || !parsedToken.Valid {
        return nil, fmt.Errorf("invalid JWT claims")
    }

    // Check expiration
    if exp, ok := claims["exp"].(float64); ok {
        if time.Now().Unix() > int64(exp) {
            return nil, fmt.Errorf("JWT expired")
        }
    }

    return &JWTClaims{
        Sub:  claims["sub"].(string),
        Role: claims["role"].(string),
        Aud:  claims["aud"].(string),
        Exp:  int64(claims["exp"].(float64)),
        Raw:  claims,
    }, nil
}
```

### 2. Database Context Propagation

Before every database query, set the JWT context:

```go
func (q *Querier) SetJWTContext(ctx context.Context, claims *JWTClaims) error {
    // Serialize claims to JSON
    claimsJSON, _ := json.Marshal(claims.Raw)

    // Set the GUC variables
    _, err := q.pool.Exec(ctx, `
        SELECT set_config('request.jwt.claims', $1, TRUE);
        SELECT set_config('request.jwt.claim.sub', $2, TRUE);
        SELECT set_config('request.jwt.claim.role', $3, TRUE);
    `, string(claimsJSON), claims.Sub, claims.Role)

    return err
}
```

### 3. Role-Based Connection Management

```go
type RoleAwareQuerier struct {
    pool        *pgxpool.Pool
    servicePool *pgxpool.Pool  // Connected as service_role (BYPASSRLS)
}

func (q *RoleAwareQuerier) Query(ctx context.Context, role string, claims *JWTClaims, sql string, args ...any) (*QueryResult, error) {
    var conn *pgxpool.Conn
    var err error

    if role == "service_role" {
        // Use service role connection that bypasses RLS
        conn, err = q.servicePool.Acquire(ctx)
    } else {
        // Use regular connection subject to RLS
        conn, err = q.pool.Acquire(ctx)
        if err == nil && claims != nil {
            // Set JWT context for RLS policies
            err = q.SetJWTContext(ctx, conn, claims)
        }
    }

    if err != nil {
        return nil, err
    }
    defer conn.Release()

    return q.execQuery(ctx, conn, sql, args...)
}
```

### 4. Storage RLS Enforcement

Storage should check policies before operations:

```go
func (h *StorageHandler) UploadObject(c *mizu.Ctx) error {
    role := middleware.GetRole(c)
    claims := middleware.GetClaims(c)

    // For non-service_role, check storage policies
    if role != "service_role" {
        allowed, err := h.checkStoragePolicy(c.Context(), claims, bucketID, path, "INSERT")
        if err != nil || !allowed {
            return storageError(c, http.StatusForbidden, "Forbidden", "access denied")
        }
    }

    // Proceed with upload...
}

func (h *StorageHandler) checkStoragePolicy(ctx context.Context, claims *JWTClaims, bucket, path, operation string) (bool, error) {
    // Execute policy check query with JWT context
    sql := `
        SELECT EXISTS (
            SELECT 1 FROM storage.objects o
            WHERE o.bucket_id = $1
            AND o.name = $2
            AND -- policy conditions using auth.uid(), auth.role()
        )
    `
    // ...
}
```

### 5. Admin Endpoint Protection

```go
func RequireServiceRole() mizu.Middleware {
    return func(next mizu.Handler) mizu.Handler {
        return func(c *mizu.Ctx) error {
            if middleware.GetRole(c) != "service_role" {
                return c.JSON(403, map[string]any{
                    "error": "forbidden",
                    "message": "service_role required for admin endpoints",
                })
            }
            return next(c)
        }
    }
}

// Usage:
router.Group("/auth/v1/admin", RequireServiceRole(), func(r *mizu.Router) {
    r.GET("/users", authHandler.ListUsers)
    // ...
})
```

## Test Cases

### Category 1: API Key Authentication

```go
// TEST-AUTH-001: Valid anon key should return role=anon
func TestValidAnonKey(t *testing.T) {
    // Given: Valid Supabase anon key
    // When: Request to any endpoint
    // Then: X-Localbase-Role = "anon"
}

// TEST-AUTH-002: Valid service_role key should return role=service_role
func TestValidServiceRoleKey(t *testing.T) {
    // Given: Valid Supabase service_role key
    // When: Request to any endpoint
    // Then: X-Localbase-Role = "service_role"
}

// TEST-AUTH-003: Invalid JWT signature should be rejected
func TestInvalidJWTSignature(t *testing.T) {
    // Given: JWT signed with wrong secret
    // When: Request to any endpoint
    // Then: 401 Unauthorized
}

// TEST-AUTH-004: Expired JWT should be rejected
func TestExpiredJWT(t *testing.T) {
    // Given: JWT with exp in the past
    // When: Request to any endpoint
    // Then: 401 Unauthorized
}

// TEST-AUTH-005: User JWT should extract sub claim as user ID
func TestUserJWTClaims(t *testing.T) {
    // Given: User JWT from login
    // When: Request to /auth/v1/user
    // Then: Returns user matching sub claim
}
```

### Category 2: RLS Enforcement

```go
// TEST-RLS-001: Anon role should see only rows matching policy
func TestAnonRLSEnforcement(t *testing.T) {
    // Given: Table with RLS policy "users see own data"
    // And: Multiple rows with different user_ids
    // When: Query with anon key (no user context)
    // Then: Returns empty array or error
}

// TEST-RLS-002: Authenticated user should see own rows
func TestAuthenticatedUserRLS(t *testing.T) {
    // Given: Table with RLS policy "users see own data"
    // And: User A has 5 todos, User B has 3 todos
    // When: Query with User A's JWT
    // Then: Returns only User A's 5 todos
}

// TEST-RLS-003: Service role should bypass RLS
func TestServiceRoleBypassRLS(t *testing.T) {
    // Given: Table with RLS enabled
    // When: Query with service_role key
    // Then: Returns all rows regardless of policy
}

// TEST-RLS-004: Insert should respect RLS policy
func TestInsertRLSPolicy(t *testing.T) {
    // Given: Policy "users can only insert with own user_id"
    // When: Insert with different user_id
    // Then: Operation denied
}

// TEST-RLS-005: Update should respect RLS policy
func TestUpdateRLSPolicy(t *testing.T) {
    // Given: Policy "users can update own rows"
    // When: Update another user's row
    // Then: 0 rows affected (not error)
}

// TEST-RLS-006: Delete should respect RLS policy
func TestDeleteRLSPolicy(t *testing.T) {
    // Given: Policy "users can delete own rows"
    // When: Delete another user's row
    // Then: 0 rows affected
}
```

### Category 3: Storage Access Control

```go
// TEST-STORAGE-001: Public bucket accessible without auth
func TestPublicBucketAccess(t *testing.T) {
    // Given: Public bucket with objects
    // When: GET object without auth
    // Then: 200 OK with content
}

// TEST-STORAGE-002: Private bucket requires auth
func TestPrivateBucketAccess(t *testing.T) {
    // Given: Private bucket with objects
    // When: GET object without auth
    // Then: 401 or 403
}

// TEST-STORAGE-003: Storage policy enforcement
func TestStoragePolicyEnforcement(t *testing.T) {
    // Given: Policy "users can only access files in user/{uid}/*"
    // And: User A uploads to user/A/file.txt
    // When: User B tries to access user/A/file.txt
    // Then: 403 Forbidden
}

// TEST-STORAGE-004: Service role bypasses storage policies
func TestServiceRoleStorageAccess(t *testing.T) {
    // Given: Any storage policy configuration
    // When: Access with service_role key
    // Then: Full access to all buckets and objects
}

// TEST-STORAGE-005: Upload policy enforcement
func TestStorageUploadPolicy(t *testing.T) {
    // Given: Policy "only INSERT to user/{uid}/*"
    // When: Upload to different user's folder
    // Then: 403 Forbidden
}
```

### Category 4: Admin Endpoints

```go
// TEST-ADMIN-001: Admin endpoints require service_role
func TestAdminEndpointAuth(t *testing.T) {
    // Given: Valid anon key
    // When: GET /auth/v1/admin/users
    // Then: 403 Forbidden
}

// TEST-ADMIN-002: Service role can access admin endpoints
func TestAdminEndpointServiceRole(t *testing.T) {
    // Given: Valid service_role key
    // When: GET /auth/v1/admin/users
    // Then: 200 OK with user list
}

// TEST-ADMIN-003: Admin can create users
func TestAdminCreateUser(t *testing.T) {
    // Given: Service role key
    // When: POST /auth/v1/admin/users
    // Then: 201 Created with new user
}

// TEST-ADMIN-004: Admin can delete users
func TestAdminDeleteUser(t *testing.T) {
    // Given: Service role key and existing user
    // When: DELETE /auth/v1/admin/users/{id}
    // Then: 204 No Content
}
```

### Category 5: Cross-Compatibility Tests

```go
// TEST-COMPAT-001: Same keys work on both Localbase and Supabase
func TestCrossCompatibleKeys(t *testing.T) {
    // Given: Supabase default demo keys
    // When: Use same key on Localbase and Supabase local
    // Then: Same role extracted
}

// TEST-COMPAT-002: Same RLS behavior
func TestCrossCompatibleRLS(t *testing.T) {
    // Given: Identical RLS policies on both
    // When: Same query with same JWT
    // Then: Same results returned
}

// TEST-COMPAT-003: Same error responses
func TestCrossCompatibleErrors(t *testing.T) {
    // Given: Invalid request
    // When: Send to both endpoints
    // Then: Same error code and structure
}

// TEST-COMPAT-004: Same auth response format
func TestCrossCompatibleAuthResponse(t *testing.T) {
    // Given: Valid credentials
    // When: Login on both systems
    // Then: Same response structure (access_token, refresh_token, user)
}

// TEST-COMPAT-005: User JWT works across systems
func TestCrossCompatibleUserJWT(t *testing.T) {
    // Given: JWT from Localbase auth
    // When: Use to query Supabase (or vice versa)
    // Then: Same results (with same JWT secret)
}
```

## Database Schema Requirements

### Auth Functions

```sql
-- Must exist in auth schema
CREATE OR REPLACE FUNCTION auth.uid() RETURNS UUID AS $$
  SELECT NULLIF(
    current_setting('request.jwt.claims', TRUE)::json->>'sub',
    ''
  )::UUID
$$ LANGUAGE SQL STABLE;

CREATE OR REPLACE FUNCTION auth.role() RETURNS TEXT AS $$
  SELECT NULLIF(
    current_setting('request.jwt.claims', TRUE)::json->>'role',
    ''
  )::TEXT
$$ LANGUAGE SQL STABLE;

CREATE OR REPLACE FUNCTION auth.jwt() RETURNS JSON AS $$
  SELECT current_setting('request.jwt.claims', TRUE)::json
$$ LANGUAGE SQL STABLE;

CREATE OR REPLACE FUNCTION auth.email() RETURNS TEXT AS $$
  SELECT NULLIF(
    current_setting('request.jwt.claims', TRUE)::json->>'email',
    ''
  )::TEXT
$$ LANGUAGE SQL STABLE;
```

### Database Roles

```sql
-- Create required roles
CREATE ROLE anon NOINHERIT NOLOGIN;
CREATE ROLE authenticated NOINHERIT NOLOGIN;
CREATE ROLE service_role NOINHERIT NOLOGIN BYPASSRLS;

-- Grant permissions
GRANT USAGE ON SCHEMA public TO anon, authenticated, service_role;
GRANT ALL ON ALL TABLES IN SCHEMA public TO anon, authenticated, service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO anon, authenticated, service_role;

-- For storage
GRANT USAGE ON SCHEMA storage TO anon, authenticated, service_role;
GRANT ALL ON ALL TABLES IN SCHEMA storage TO anon, authenticated, service_role;
```

## Configuration

```yaml
# localbase.yaml
auth:
  jwt_secret: "your-super-secret-jwt-key-min-32-characters"
  jwt_expiry: 3600
  issuer: "http://localhost:54321/auth/v1"

api:
  anon_key: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  service_key: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

database:
  # Connection for anon/authenticated (subject to RLS)
  url: "postgres://authenticator:password@localhost:5432/localbase"
  # Connection for service_role (bypasses RLS)
  service_url: "postgres://service_role:password@localhost:5432/localbase"
```

## Implementation Checklist

### Phase 1: JWT Validation (Critical) - COMPLETED
- [x] Add JWT signature validation to API key middleware
- [x] Add JWT expiration checking
- [x] Extract full claims (sub, role, email, app_metadata, user_metadata, etc.)
- [x] Support both Bearer token and apikey header
- [x] Add configurable validation via `LOCALBASE_VALIDATE_JWT` env var

### Phase 2: RLS Context Propagation (Critical) - COMPLETED
- [x] Create connection pool management for roles
- [x] Implement `SET LOCAL request.jwt.claims` before queries
- [x] Implement role switching based on JWT
- [x] Test with existing RLS policies
- [x] Add `QueryWithRLS()` and `ExecWithRLS()` methods

### Phase 3: Service Role Bypass (Critical) - COMPLETED
- [x] Service role queries bypass RLS context setup
- [x] Route service_role requests to direct query execution
- [x] Verify RLS is bypassed for service_role

### Phase 4: Storage RLS (High) - PARTIALLY COMPLETED
- [x] Implement storage policy checking (app-level)
- [x] Pass JWT context to storage queries (via middleware headers)
- [x] Implement bucket-level access control (public/private)
- [x] Implement object-level access control (user folder pattern)
- [ ] Migrate to PostgreSQL-level storage RLS (future enhancement)

### Phase 5: Admin Security (High) - COMPLETED
- [x] Add service_role requirement to admin endpoints
- [x] Audit all admin endpoints for proper access control
- [ ] Add rate limiting to auth endpoints (future enhancement)

### Phase 6: Compatibility Testing (High) - IN PROGRESS
- [x] Set up side-by-side Supabase local environment
- [x] Create compatibility test framework
- [x] Run all compatibility tests
- [ ] Document any intentional differences
- [ ] Create automated CI compatibility tests

## Migration Notes for seed/database_test.go

The current test file at `pkg/seed/database_test.go` contains Supabase compatibility comparison tests.

**Test File Locations**:
1. `pkg/seed/database_test.go` - Original comparison tests (compares Localbase vs Supabase side-by-side)
2. `test/integration/supabase_compat_test.go` - New comprehensive integration tests (tests Localbase features independently)

**Reorganization Plan**:
1. Keep `pkg/seed/database_test.go` for now (useful for A/B comparison testing)
2. Add build tag `//go:build integration` to both files
3. Run integration tests with: `go test -tags integration ./test/integration/...`
4. Run comparison tests with: `go test -tags integration ./pkg/seed/... -run TestCompat`

**Running Tests**:
```bash
# Run integration tests (Localbase only)
go test -tags integration ./test/integration/...

# Run comparison tests (requires both Localbase and Supabase running)
LOCALBASE_URL=http://localhost:54321 \
SUPABASE_URL=http://localhost:54421 \
go test -tags integration ./pkg/seed/...
```

## Security Considerations

1. **JWT Secret Protection**: Never log or expose the JWT secret
2. **Service Key Protection**: Service key should never be exposed to clients
3. **Rate Limiting**: Add rate limiting to auth endpoints to prevent brute force
4. **Audit Logging**: Log all admin operations and authentication failures
5. **Input Validation**: Sanitize all user inputs before database queries

## Test Infrastructure

### Test Files

| File | Purpose | Build Tag |
|------|---------|-----------|
| `test/integration/supabase_compat_test.go` | Comprehensive integration tests | `integration` |
| `pkg/seed/database_test.go` | A/B comparison with Supabase | `integration` |

### Test Categories

The integration test suite (`test/integration/supabase_compat_test.go`) covers:

1. **API Key Authentication Tests**
   - `TestAPIKey_AnonKeyRole` - Anon key returns role=anon
   - `TestAPIKey_ServiceRoleRole` - Service role can access admin endpoints
   - `TestAPIKey_AnonCantAccessAdmin` - Anon cannot access admin endpoints
   - `TestAPIKey_InvalidKeyRejected` - Invalid keys are rejected

2. **JWT Claims Tests**
   - `TestJWT_UserClaimsExtracted` - User claims extracted from JWT

3. **RLS Tests**
   - `TestRLS_ServiceRoleBypassesRLS` - Service role bypasses all RLS
   - `TestRLS_AnonCannotAccessRLSProtectedTable` - Anon sees empty/filtered results
   - `TestRLS_AuthenticatedUserSeesOwnRows` - Users see only their own data

4. **Storage RLS Tests**
   - `TestStorage_ListBucketsPublicOnly` - Anon sees only public buckets
   - `TestStorage_ServiceRoleSeesAllBuckets` - Service role sees all buckets
   - `TestStorage_AnonCannotUploadToPrivateBucket` - Anon cannot write private
   - `TestStorage_UserCanUploadToOwnFolder` - Users can upload to own folder
   - `TestStorage_UserCannotUploadToOtherUserFolder` - Users cannot write to other folders

5. **Admin Endpoint Tests**
   - `TestAdmin_ListUsersRequiresServiceRole`
   - `TestAdmin_CreateUserRequiresServiceRole`

6. **PostgREST Compatibility Tests**
   - `TestPostgREST_SelectWithFilters` - Filter operations (eq, neq, gt, like, in, etc.)
   - `TestPostgREST_InsertWithReturn` - INSERT with return=representation
   - `TestPostgREST_UpdateWithReturn` - UPDATE with return=representation
   - `TestPostgREST_DeleteWithReturn` - DELETE with return=representation
   - `TestPostgREST_RPC` - RPC function calls

7. **Side-by-Side Comparison Tests** (requires Supabase running)
   - `TestSideBySide_Select` - Compare SELECT responses
   - `TestSideBySide_Storage_ListBuckets` - Compare storage responses
   - `TestSideBySide_Auth_AdminUsers` - Compare admin responses

### Running Tests

```bash
# Run all integration tests (requires Localbase running)
cd blueprints/localbase
go test -tags integration ./test/integration/... -v

# Run specific test
go test -tags integration ./test/integration/... -run TestAPIKey_AnonKeyRole -v

# Run side-by-side comparison (requires both Localbase and Supabase)
LOCALBASE_URL=http://localhost:54321 \
SUPABASE_URL=http://localhost:54421 \
go test -tags integration ./test/integration/... -run TestSideBySide -v

# Run benchmarks
go test -tags integration ./test/integration/... -bench=. -run=^$
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOCALBASE_URL` | `http://localhost:54321` | Localbase API URL |
| `LOCALBASE_ANON_KEY` | (Supabase default) | Anon API key |
| `LOCALBASE_SERVICE_KEY` | (Supabase default) | Service role key |
| `LOCALBASE_JWT_SECRET` | (Supabase default) | JWT signing secret |
| `SUPABASE_URL` | `http://localhost:54421` | Supabase API URL (for comparison) |
| `SUPABASE_ANON_KEY` | (same as Localbase) | Supabase anon key |

## Implementation Summary

### Files Modified

| File | Changes |
|------|---------|
| `app/web/middleware/apikey.go` | JWT validation, claims extraction, header propagation |
| `app/web/handler/api/auth.go` | Configurable JWT secret, issuer from env vars |
| `app/web/handler/api/storage.go` | User folder access control, owner tracking |
| `app/web/handler/api/database.go` | RLS context extraction and propagation |
| `store/postgres/database.go` | QueryWithRLS, ExecWithRLS methods |
| `pkg/postgrest/handler.go` | RLSContext support in all operations |
| `test/integration/supabase_compat_test.go` | Comprehensive test suite |

### Key Implementation Details

1. **JWT Secret Sharing**: Both API key middleware and auth handler use `LOCALBASE_JWT_SECRET`
2. **RLS Context Flow**: JWT claims → middleware headers → RLSContext → PostgreSQL GUC variables
3. **Service Role Bypass**: Direct query execution for service_role (no RLS context setup)
4. **Storage Access**: Path-based folder matching for user-specific access control
5. **Owner Tracking**: Objects store owner user ID for ownership-based access control

## References

- [Supabase API Keys Documentation](https://supabase.com/docs/guides/api/api-keys)
- [Supabase Row Level Security](https://supabase.com/docs/guides/database/postgres/row-level-security)
- [Supabase Storage Access Control](https://supabase.com/docs/guides/storage/security/access-control)
- [PostgreSQL Row Security Policies](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
