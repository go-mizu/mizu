# Spec 0387: Supabase Edge Functions API Compatibility Testing Plan

## Document Info

| Field | Value |
|-------|-------|
| Spec ID | 0387 |
| Version | 1.0 |
| Date | 2025-01-17 |
| Status | **In Progress** |
| Priority | Critical |
| Estimated Tests | 150+ |
| Supabase Functions Version | Latest (2025) |
| Supabase Local Port | 54421 |
| Localbase Port | 54321 |

## Overview

This document outlines a comprehensive testing plan for the Localbase Edge Functions API to achieve 100% compatibility with Supabase's Edge Functions implementation. Testing will be performed against both Supabase Local and Localbase to verify identical behavior for inputs, outputs, and error codes.

### Testing Philosophy

- **No mocks**: All tests run against real function backends
- **Side-by-side comparison**: Every request runs against both Supabase and Localbase
- **Comprehensive coverage**: Every endpoint, edge case, and error condition
- **Regression prevention**: Tests ensure compatibility is maintained over time
- **Response accuracy**: Response bodies, headers, and error codes must match exactly

### Compatibility Target

| Aspect | Target |
|--------|--------|
| HTTP Status Codes | 100% match |
| Error Response Format | 100% match |
| Response Headers | 100% match |
| Response Body Structure | 100% match |
| CORS Headers | 100% match |
| JWT Validation | 100% match |

## Reference Documentation

### Official Supabase Edge Functions Documentation
- [Edge Functions Guide](https://supabase.com/docs/guides/functions)
- [Edge Functions Architecture](https://supabase.com/docs/guides/functions/architecture)
- [Edge Functions Quickstart](https://supabase.com/docs/guides/functions/quickstart)
- [JavaScript API Reference](https://supabase.com/docs/reference/javascript/functions-invoke)
- [CORS Configuration](https://supabase.com/docs/guides/functions/cors)
- [Management API Reference](https://supabase.com/docs/reference/api/introduction)

### Supabase Management API Endpoints
- Deploy Function: `POST /v1/projects/{ref}/functions/deploy`
- List Functions: `GET /v1/projects/{ref}/functions`
- Get Function: `GET /v1/projects/{ref}/functions/{function_slug}`
- Update Function: `PATCH /v1/projects/{ref}/functions/{function_slug}`
- Delete Function: `DELETE /v1/projects/{ref}/functions/{function_slug}`
- Get Function Body: `GET /v1/projects/{ref}/functions/{function_slug}/body`
- Bulk Update: `PUT /v1/projects/{ref}/functions`

## Test Environment Setup

### Supabase Local Configuration
```
Functions API: http://127.0.0.1:54421/functions/v1
Management API: http://127.0.0.1:54421/api/functions
Anon Key: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6ImFub24iLCJleHAiOjE5ODM4MTI5OTZ9.CRXP1A7WOeoJeXxjNni43kdQwgnWNReilDMblYTn_I0
Service Role Key: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU
```

### Localbase Configuration
```
Functions API: http://localhost:54321/functions/v1
Management API: http://localhost:54321/api/functions
Anon Key: Same as Supabase (compatible keys)
Service Role Key: Same as Supabase (compatible keys)
```

---

## 1. Function Invocation API (`/functions/v1/{function-name}`)

### 1.1 Basic Function Invocation

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| INVOKE-001 | Invoke function with POST | `POST /functions/v1/hello` | 200 OK with function response |
| INVOKE-002 | Invoke function with GET | `GET /functions/v1/hello` | 200 OK with function response |
| INVOKE-003 | Invoke function with PUT | `PUT /functions/v1/hello` | 200 OK with function response |
| INVOKE-004 | Invoke function with DELETE | `DELETE /functions/v1/hello` | 200 OK with function response |
| INVOKE-005 | Invoke function with PATCH | `PATCH /functions/v1/hello` | 200 OK with function response |
| INVOKE-006 | Invoke non-existent function | `POST /functions/v1/nonexistent` | 404 Not Found |
| INVOKE-007 | Invoke inactive function | `POST /functions/v1/inactive-fn` | 503 Service Unavailable |
| INVOKE-008 | Invoke with empty body | `POST /functions/v1/hello` with empty body | 200 OK |

### 1.2 Request Body Handling

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| BODY-001 | JSON body | `POST /functions/v1/echo` with `{"name": "test"}` | 200 OK, echoed JSON |
| BODY-002 | Text body | `POST /functions/v1/echo` with plain text | 200 OK, Content-Type: text/plain |
| BODY-003 | Binary body | `POST /functions/v1/echo` with binary data | 200 OK, Content-Type: application/octet-stream |
| BODY-004 | FormData body | `POST /functions/v1/upload` with multipart/form-data | 200 OK |
| BODY-005 | URL-encoded body | `POST /functions/v1/form` with application/x-www-form-urlencoded | 200 OK |
| BODY-006 | Large body (1MB) | `POST /functions/v1/large` with 1MB payload | 200 OK |
| BODY-007 | Very large body (10MB) | `POST /functions/v1/large` with 10MB payload | 200 OK or 413 |

### 1.3 Request Headers

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| HEADER-001 | Custom headers passed | `X-Custom-Header: value` | Function receives header |
| HEADER-002 | Content-Type honored | `Content-Type: application/json` | Function receives correct content type |
| HEADER-003 | Accept header | `Accept: application/json` | Response Content-Type matches |
| HEADER-004 | X-Client-Info header | `X-Client-Info: supabase-js/2.0` | Function receives header |
| HEADER-005 | Authorization header | `Authorization: Bearer token` | Function receives header |

### 1.4 Authentication & Authorization

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| AUTH-001 | Invoke with anon key (JWT required) | `apikey: <anon>`, function has verify_jwt=true | 401 Unauthorized |
| AUTH-002 | Invoke with valid user JWT | Valid authenticated JWT | 200 OK |
| AUTH-003 | Invoke with service role key | `apikey: <service_role>` | 200 OK |
| AUTH-004 | Invoke with expired JWT | Expired token | 401 Unauthorized |
| AUTH-005 | Invoke with invalid JWT signature | Tampered token | 401 Unauthorized |
| AUTH-006 | Invoke with verify_jwt=false | No auth header | 200 OK |
| AUTH-007 | Missing Authorization header (verify_jwt=true) | No Authorization header | 401 Unauthorized |
| AUTH-008 | Malformed Authorization header | `Authorization: invalid` | 401 Unauthorized |

### 1.5 CORS Preflight Handling

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| CORS-001 | OPTIONS preflight | `OPTIONS /functions/v1/hello` | 200 OK with CORS headers |
| CORS-002 | Access-Control-Allow-Origin | OPTIONS request | Header: `Access-Control-Allow-Origin: *` |
| CORS-003 | Access-Control-Allow-Headers | OPTIONS request | Header includes: `authorization, x-client-info, apikey, content-type` |
| CORS-004 | Access-Control-Allow-Methods | OPTIONS request | Header includes: `POST, GET, OPTIONS, PUT, DELETE` |
| CORS-005 | CORS headers on response | Normal POST request | CORS headers present |
| CORS-006 | Custom Origin header | `Origin: https://example.com` | Origin reflected or * |

### 1.6 Error Responses

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| ERROR-001 | Function throws error | Function throws exception | 500 Internal Server Error |
| ERROR-002 | Function timeout | Function exceeds timeout | 504 Gateway Timeout |
| ERROR-003 | Invalid function name | `POST /functions/v1/` | 404 Not Found |
| ERROR-004 | Function returns 4xx | Function returns error | 4xx status preserved |
| ERROR-005 | Function returns 5xx | Function returns error | 5xx status preserved |

### 1.7 Response Handling

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| RESP-001 | JSON response | Function returns JSON | 200 OK, Content-Type: application/json |
| RESP-002 | Text response | Function returns text | 200 OK, Content-Type: text/plain |
| RESP-003 | HTML response | Function returns HTML | 200 OK, Content-Type: text/html |
| RESP-004 | Binary response | Function returns binary | 200 OK, Content-Type: application/octet-stream |
| RESP-005 | Streaming response | Function returns SSE | 200 OK, chunked transfer |
| RESP-006 | Custom headers | Function sets headers | Custom headers preserved |
| RESP-007 | Custom status code | Function returns 201 | 201 Created |

---

## 2. Function Management API (`/api/functions`)

### 2.1 List Functions

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| LIST-001 | List all functions | `GET /api/functions` | 200 OK, array of functions |
| LIST-002 | List empty | No functions exist | 200 OK, `[]` |
| LIST-003 | List requires service_role | Use anon key | 403 Forbidden |
| LIST-004 | Function properties | `GET /api/functions` | Each has id, name, slug, status, version |

### 2.2 Create Function

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| CREATE-001 | Create basic function | `POST /api/functions` with `{"name": "hello"}` | 201 Created |
| CREATE-002 | Create with source code | Include `source_code` field | 201 Created with deployment |
| CREATE-003 | Create with verify_jwt=true | `{"verify_jwt": true}` | 201 Created |
| CREATE-004 | Create with verify_jwt=false | `{"verify_jwt": false}` | 201 Created |
| CREATE-005 | Create with custom entrypoint | `{"entrypoint": "main.ts"}` | 201 Created |
| CREATE-006 | Create with import_map | `{"import_map": "..."}` | 201 Created |
| CREATE-007 | Create duplicate name | Same name twice | 400 Bad Request |
| CREATE-008 | Create with empty name | `{"name": ""}` | 400 Bad Request |
| CREATE-009 | Create requires service_role | Use anon key | 403 Forbidden |

### 2.3 Get Function

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| GET-001 | Get existing function | `GET /api/functions/{id}` | 200 OK, function details |
| GET-002 | Get non-existent function | `GET /api/functions/nonexistent` | 404 Not Found |
| GET-003 | Get requires service_role | Use anon key | 403 Forbidden |
| GET-004 | Get returns all properties | `GET /api/functions/{id}` | id, name, slug, status, version, verify_jwt, entrypoint |

### 2.4 Update Function

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| UPDATE-001 | Update name | `PUT /api/functions/{id}` with `{"name": "new-name"}` | 200 OK |
| UPDATE-002 | Update verify_jwt | `{"verify_jwt": true}` | 200 OK |
| UPDATE-003 | Update status | `{"status": "inactive"}` | 200 OK |
| UPDATE-004 | Update entrypoint | `{"entrypoint": "new.ts"}` | 200 OK |
| UPDATE-005 | Update non-existent | `PUT /api/functions/nonexistent` | 404 Not Found |
| UPDATE-006 | Update requires service_role | Use anon key | 403 Forbidden |

### 2.5 Delete Function

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| DELETE-001 | Delete existing function | `DELETE /api/functions/{id}` | 204 No Content |
| DELETE-002 | Delete non-existent function | `DELETE /api/functions/nonexistent` | 500 or 404 |
| DELETE-003 | Delete requires service_role | Use anon key | 403 Forbidden |

### 2.6 Deploy Function

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| DEPLOY-001 | Deploy with source code | `POST /api/functions/{id}/deploy` | 201 Created |
| DEPLOY-002 | Deploy increments version | Deploy twice | Version incremented |
| DEPLOY-003 | Deploy non-existent function | Deploy to nonexistent | 404 Not Found |
| DEPLOY-004 | Deploy without source_code | Empty body | 400 Bad Request |
| DEPLOY-005 | Deploy requires service_role | Use anon key | 403 Forbidden |

### 2.7 List Deployments

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| DEPLOYS-001 | List deployments | `GET /api/functions/{id}/deployments` | 200 OK, array |
| DEPLOYS-002 | List with limit | `?limit=5` | Max 5 deployments |
| DEPLOYS-003 | Deployments ordered by version | `GET /api/functions/{id}/deployments` | Newest first |
| DEPLOYS-004 | List requires service_role | Use anon key | 403 Forbidden |

---

## 3. Secrets Management API

### 3.1 List Secrets

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| SECRETS-001 | List all secrets | `GET /api/functions/secrets` | 200 OK, array of secret names |
| SECRETS-002 | List empty | No secrets exist | 200 OK, `[]` |
| SECRETS-003 | Secrets do not expose values | List secrets | No `value` field in response |
| SECRETS-004 | List requires service_role | Use anon key | 403 Forbidden |

### 3.2 Create Secret

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| SECRETS-010 | Create new secret | `POST /api/functions/secrets` | 201 Created |
| SECRETS-011 | Create with name and value | `{"name": "API_KEY", "value": "secret"}` | 201 Created |
| SECRETS-012 | Create duplicate (upsert) | Same name twice | Updates value |
| SECRETS-013 | Create with empty name | `{"name": "", "value": "x"}` | 400 Bad Request |
| SECRETS-014 | Create with empty value | `{"name": "x", "value": ""}` | 400 Bad Request |
| SECRETS-015 | Create requires service_role | Use anon key | 403 Forbidden |

### 3.3 Delete Secret

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| SECRETS-020 | Delete existing secret | `DELETE /api/functions/secrets/{name}` | 204 No Content |
| SECRETS-021 | Delete non-existent secret | `DELETE /api/functions/secrets/nonexistent` | 500 or 404 |
| SECRETS-022 | Delete requires service_role | Use anon key | 403 Forbidden |

---

## 4. Function Runtime Compatibility

### 4.1 Environment Variables

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| ENV-001 | Access SUPABASE_URL | Function reads env | Variable available |
| ENV-002 | Access SUPABASE_ANON_KEY | Function reads env | Variable available |
| ENV-003 | Access SUPABASE_SERVICE_ROLE_KEY | Function reads env | Variable available |
| ENV-004 | Access custom secrets | Function reads secret | Secret value available |

### 4.2 Request Object Compatibility

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| REQ-001 | req.method | Various methods | Correct method returned |
| REQ-002 | req.url | Function access URL | Full URL available |
| REQ-003 | req.headers | Custom headers | All headers accessible |
| REQ-004 | req.body | JSON body | Parsed body available |
| REQ-005 | req.json() | JSON body | Async JSON parsing |
| REQ-006 | req.text() | Text body | Text content |
| REQ-007 | req.formData() | FormData body | FormData parsing |
| REQ-008 | req.arrayBuffer() | Binary body | ArrayBuffer |

### 4.3 Response Object Compatibility

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| RESP-101 | new Response(body) | Simple response | Body returned |
| RESP-102 | new Response(body, {status}) | Custom status | Status code set |
| RESP-103 | new Response(body, {headers}) | Custom headers | Headers set |
| RESP-104 | Response.json() | JSON helper | JSON response |
| RESP-105 | Response.redirect() | Redirect helper | 302 redirect |

---

## 5. Error Response Format Compatibility

### 5.1 Standard Error Format

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| ERR-001 | 400 Bad Request format | Invalid request | `{"error": "...", "message": "..."}` |
| ERR-002 | 401 Unauthorized format | No auth | `{"error": "Unauthorized", "message": "..."}` |
| ERR-003 | 403 Forbidden format | No permission | `{"error": "Forbidden", "message": "..."}` |
| ERR-004 | 404 Not Found format | Unknown function | `{"error": "Not Found", "message": "..."}` |
| ERR-005 | 500 Internal Server Error | Function crash | `{"error": "Internal Server Error", "message": "..."}` |

### 5.2 Function Error Types

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| FERR-001 | FunctionsHttpError | Function returns error | Error with statusCode |
| FERR-002 | FunctionsRelayError | Relay service error | Relay error format |
| FERR-003 | FunctionsFetchError | Network error | Fetch error format |

---

## 6. Side-by-Side Comparison Tests

### 6.1 Invocation Comparison

| Test Case | Description | Expected |
|-----------|-------------|----------|
| CMP-001 | POST /functions/v1/hello | Status, headers, body match |
| CMP-002 | OPTIONS /functions/v1/hello | CORS headers match |
| CMP-003 | POST with JSON body | Response format matches |
| CMP-004 | Invoke with auth | Auth handling matches |
| CMP-005 | Error responses | Error format matches |

### 6.2 Management API Comparison

| Test Case | Description | Expected |
|-----------|-------------|----------|
| CMP-010 | GET /api/functions | Response structure matches |
| CMP-011 | POST /api/functions | Create response matches |
| CMP-012 | DELETE /api/functions/{id} | Delete behavior matches |
| CMP-013 | Error responses | Error format matches |

---

## 7. Performance & Limits

### 7.1 Request Limits

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| LIMIT-001 | Max body size | 10MB+ body | 413 or accepted |
| LIMIT-002 | Max header size | Large headers | 431 or accepted |
| LIMIT-003 | Max URL length | Very long URL | 414 or accepted |

### 7.2 Timeout Handling

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| TIMEOUT-001 | Function execution timeout | Slow function | 504 Gateway Timeout |
| TIMEOUT-002 | Connection timeout | No response | 504 Gateway Timeout |

---

## 8. Integration with Other Services

### 8.1 Database Integration

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| DB-001 | Function queries database | CRUD operation | Data returned |
| DB-002 | Function respects RLS | User-scoped query | Only user's data |
| DB-003 | Service role bypasses RLS | Admin query | All data |

### 8.2 Storage Integration

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| STORAGE-001 | Function reads storage | Read file | File content |
| STORAGE-002 | Function writes storage | Upload file | Success |
| STORAGE-003 | Storage RLS applied | User upload | Only own folder |

### 8.3 Auth Integration

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| AUTH-101 | Function creates user | Admin signup | User created |
| AUTH-102 | Function validates JWT | Token check | Claims accessible |

---

## 9. Implementation Requirements

### 9.1 HTTP Handler Requirements

```go
// Required handler behaviors:
// 1. Support all HTTP methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
// 2. Handle OPTIONS preflight with correct CORS headers
// 3. Pass all headers to function (except hop-by-hop)
// 4. Stream request body to function
// 5. Stream response body from function
// 6. Preserve status codes and headers from function response
```

### 9.2 CORS Header Requirements

```go
corsHeaders := map[string]string{
    "Access-Control-Allow-Origin":  "*",
    "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type",
    "Access-Control-Allow-Methods": "POST, GET, OPTIONS, PUT, DELETE",
}
```

### 9.3 JWT Validation Requirements

```go
// JWT validation must:
// 1. Verify signature using configured JWT secret
// 2. Check expiration (exp claim)
// 3. Extract role claim for authorization
// 4. Pass claims to function in header or context
// 5. Return 401 for invalid/expired tokens when verify_jwt=true
// 6. Skip validation when verify_jwt=false
```

### 9.4 Error Response Format

```go
type FunctionError struct {
    Error   string `json:"error"`
    Message string `json:"message"`
}
```

---

## 10. Test Implementation

### 10.1 Test File Location

```
blueprints/localbase/test/integration/functions_test.go
```

### 10.2 Test Structure

```go
//go:build integration

package integration

import (
    "testing"
)

// TestFunctions_Invoke tests function invocation endpoints
func TestFunctions_Invoke(t *testing.T) { ... }

// TestFunctions_Management tests function management API
func TestFunctions_Management(t *testing.T) { ... }

// TestFunctions_Secrets tests secrets management
func TestFunctions_Secrets(t *testing.T) { ... }

// TestFunctions_CORS tests CORS preflight handling
func TestFunctions_CORS(t *testing.T) { ... }

// TestFunctions_Auth tests authentication/authorization
func TestFunctions_Auth(t *testing.T) { ... }

// TestFunctions_SideBySide runs comparison tests
func TestFunctions_SideBySide(t *testing.T) { ... }
```

### 10.3 Helper Functions Required

```go
// Helper functions for tests
func createTestFunction(t *testing.T, name string) string
func deleteTestFunction(t *testing.T, id string)
func createTestSecret(t *testing.T, name, value string)
func deleteTestSecret(t *testing.T, name string)
func invokeFunction(t *testing.T, name string, body any) (int, []byte, http.Header)
func createUserJWT(userID, email string) string
```

---

## 11. Database Schema

### 11.1 Functions Schema

```sql
-- Functions schema
CREATE SCHEMA IF NOT EXISTS functions;

-- Functions table
CREATE TABLE functions.functions (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    version INTEGER DEFAULT 1,
    status VARCHAR(20) DEFAULT 'active',
    entrypoint VARCHAR(255) DEFAULT 'index.ts',
    import_map TEXT,
    verify_jwt BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Deployments table
CREATE TABLE functions.deployments (
    id VARCHAR(26) PRIMARY KEY,
    function_id VARCHAR(26) REFERENCES functions.functions(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    source_code TEXT NOT NULL,
    bundle_path TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    deployed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Secrets table
CREATE TABLE functions.secrets (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_functions_slug ON functions.functions(slug);
CREATE INDEX idx_deployments_function_id ON functions.deployments(function_id);
CREATE INDEX idx_deployments_version ON functions.deployments(function_id, version DESC);
CREATE INDEX idx_secrets_name ON functions.secrets(name);
```

---

## 12. API Response Examples

### 12.1 List Functions Response

```json
[
  {
    "id": "01HXY1234567890ABCDEF",
    "name": "hello-world",
    "slug": "hello-world",
    "version": 3,
    "status": "active",
    "entrypoint": "index.ts",
    "import_map": "",
    "verify_jwt": true,
    "created_at": "2025-01-17T00:00:00Z",
    "updated_at": "2025-01-17T12:00:00Z"
  }
]
```

### 12.2 Create Function Response

```json
{
  "id": "01HXY1234567890ABCDEF",
  "name": "my-function",
  "slug": "my-function",
  "version": 1,
  "status": "active",
  "entrypoint": "index.ts",
  "import_map": "",
  "verify_jwt": true,
  "created_at": "2025-01-17T00:00:00Z",
  "updated_at": "2025-01-17T00:00:00Z"
}
```

### 12.3 Deploy Response

```json
{
  "id": "01HXY1234567890DEPLOY",
  "function_id": "01HXY1234567890ABCDEF",
  "version": 2,
  "source_code": "export default...",
  "status": "deployed",
  "deployed_at": "2025-01-17T00:00:00Z"
}
```

### 12.4 Invocation Response

```json
{
  "message": "Function executed",
  "function": "hello-world",
  "version": 2,
  "method": "POST",
  "executed_at": "2025-01-17T00:00:00Z"
}
```

### 12.5 Error Responses

**404 Not Found:**
```json
{
  "error": "Not Found",
  "message": "function not found"
}
```

**401 Unauthorized:**
```json
{
  "error": "Unauthorized",
  "message": "authorization required"
}
```

**503 Service Unavailable:**
```json
{
  "error": "Service Unavailable",
  "message": "function is not active"
}
```

---

## 13. Implementation Notes

### 13.1 Slug Generation

Function slugs are automatically generated from the function name:
- Convert to lowercase
- Replace spaces with hyphens
- Preserve alphanumeric characters and hyphens

```go
slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
```

### 13.2 CORS Headers

All function invocation responses must include these headers:

```go
corsHeaders := map[string]string{
    "Access-Control-Allow-Origin":  "*",
    "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type, accept, accept-language, x-authorization",
    "Access-Control-Allow-Methods": "POST, GET, OPTIONS, PUT, DELETE, PATCH",
}
```

### 13.3 JWT Verification Flow

1. Check if function requires JWT (`verify_jwt=true`)
2. If not required, allow request
3. If required, check for Authorization header
4. Service role always allowed
5. Authenticated role allowed with valid JWT
6. Anon role without valid JWT rejected (401)

### 13.4 Secrets Security

- Secret values are stored in the database (should be encrypted in production)
- Secret values are NEVER exposed in API responses
- Only secret names and creation timestamps are returned
- Secrets use upsert semantics (create or update)

---

## 14. Test Summary

### 14.1 Total Test Cases

| Category | Count |
|----------|-------|
| Function Invocation | 25+ |
| Function Management | 20+ |
| Deployments | 10+ |
| Secrets | 10+ |
| Authentication | 15+ |
| CORS | 10+ |
| Error Handling | 10+ |
| Performance | 5+ |
| Side-by-Side | 10+ |
| **Total** | **115+** |

### 14.2 Test File Location

```
blueprints/localbase/test/integration/functions_test.go
```

### 14.3 Running Tests

```bash
# Run all integration tests
go test -tags=integration ./test/integration/...

# Run only functions tests
go test -tags=integration -run TestFunctions ./test/integration/...

# Run with verbose output
go test -tags=integration -v -run TestFunctions ./test/integration/...

# Run benchmarks
go test -tags=integration -bench=BenchmarkFunctions ./test/integration/...
```

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-01-17 | Initial specification |
| 1.1 | 2025-01-17 | Added comprehensive test cases, database schema, API examples |
