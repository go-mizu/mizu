# Localbase Security Audit Report

**Audit Date:** 2026-01-16
**Scope:** Full security review of `/blueprints/localbase`
**Severity Levels:** CRITICAL, HIGH, MEDIUM, LOW

---

## Executive Summary

This security audit identified **15 security vulnerabilities** across the localbase codebase, including 4 critical issues, 6 high severity issues, and 5 medium severity issues. The most critical findings involve hardcoded JWT secrets, SQL injection vulnerabilities, and missing authentication on administrative endpoints.

---

## Critical Vulnerabilities

### SEC-001: Hardcoded Default JWT Secret (CRITICAL)

**Location:**
- `app/web/middleware/apikey.go:67`
- `app/web/handler/api/auth.go:22`

**Description:**
The default JWT secret is hardcoded as `"super-secret-jwt-token-with-at-least-32-characters-long"`. This secret is publicly known (used in Supabase documentation) and allows anyone to forge valid JWT tokens.

**Risk:** Complete authentication bypass. Attackers can create arbitrary JWT tokens with any role (including `service_role`) to gain full administrative access.

**Affected Code:**
```go
const DefaultJWTSecret = "super-secret-jwt-token-with-at-least-32-characters-long"
```

**Remediation:**
1. Generate a cryptographically secure random secret on first startup if not configured
2. Log a warning when using the default secret in non-development environments
3. Require explicit secret configuration for production deployments

**Status:** FIXED

---

### SEC-002: Test API Key Grants service_role (CRITICAL)

**Location:** `app/web/middleware/apikey.go:154-156`

**Description:**
The hardcoded string `"test-api-key"` is accepted as valid authentication and grants `service_role` privileges, providing full administrative access.

**Risk:** Trivial privilege escalation. Any attacker can gain `service_role` access by using `"test-api-key"` as their API key.

**Affected Code:**
```go
} else if apiKey == "test-api-key" {
    // Legacy test key for backward compatibility
    role = "service_role"
}
```

**Remediation:**
1. Remove the hardcoded test key entirely
2. Implement proper test key configuration via environment variables if needed

**Status:** FIXED

---

### SEC-003: SQL Injection in RLS Policy Creation (CRITICAL)

**Location:** `store/postgres/database.go:516-540`

**Description:**
The `CreatePolicy` function directly interpolates user-provided `Definition` and `CheckExpr` values into SQL without sanitization. While identifier quoting is applied to names, the policy expressions themselves are raw SQL that gets executed.

**Risk:** Full SQL injection. Attackers can execute arbitrary SQL commands by crafting malicious policy definitions.

**Affected Code:**
```go
if policy.Definition != "" {
    sql += " USING (" + policy.Definition + ")"
}
if policy.CheckExpr != "" {
    sql += " WITH CHECK (" + policy.CheckExpr + ")"
}
```

**Remediation:**
1. Validate policy expressions against a whitelist of allowed SQL constructs
2. Implement a policy expression parser that only allows safe operations
3. Require service_role for policy creation (already admin operation)

**Status:** FIXED - Added service_role requirement and input validation

---

### SEC-004: Raw SQL Execution Endpoint Without Authentication (CRITICAL)

**Location:** `app/web/handler/api/database.go:331-363`

**Description:**
The `/api/database/query` endpoint allows execution of arbitrary SQL queries. While the handler checks for SELECT/WITH prefixes, the entire `/api/database/*` group lacks authentication.

**Risk:** Complete database compromise. Any unauthenticated user can execute arbitrary SQL queries, including data exfiltration and modification.

**Affected Code:**
```go
// server.go - No auth middleware applied
app.Group("/api/database", func(database *mizu.Router) {
    // ... routes defined without authentication
    database.Post("/query", databaseHandler.ExecuteQuery)
})
```

**Remediation:**
1. Apply `serviceRoleMw` to `/api/database/*` endpoints
2. Consider implementing query whitelisting for non-admin users

**Status:** FIXED

---

## High Severity Vulnerabilities

### SEC-005: MFA TOTP Verification Not Implemented (HIGH)

**Location:** `app/web/handler/api/auth.go:532-534`

**Description:**
The MFA verification endpoint accepts any 6-digit code without actual TOTP verification. The comment explicitly states "For now, accept any 6-digit code."

**Risk:** Complete MFA bypass. Users who enable MFA are not actually protected by a second factor.

**Affected Code:**
```go
// In production, verify TOTP code
// For now, accept any 6-digit code
if len(req.Code) != 6 {
    return authError(c, 400, "mfa_verification_failed", "Invalid verification code")
}
```

**Remediation:**
1. Implement proper TOTP verification using a library like `pquerna/otp`
2. Store and validate against the generated TOTP secret

**Status:** FIXED

---

### SEC-006: Missing Authentication on Functions API (HIGH)

**Location:** `app/web/server.go:143-156`

**Description:**
The `/api/functions/*` endpoints for managing edge functions have no authentication, allowing anyone to create, modify, or delete functions.

**Risk:** Arbitrary code execution. Attackers can deploy malicious functions that execute server-side code.

**Remediation:**
1. Apply `apiKeyMw` to `/api/functions/*` group
2. Consider requiring `service_role` for function management

**Status:** FIXED

---

### SEC-007: Secrets Stored in Plaintext (HIGH)

**Location:** `app/web/handler/api/functions.go:272`

**Description:**
Function secrets are stored in the database without encryption. A comment even acknowledges this: "In production, encrypt this."

**Risk:** Secret exposure. If the database is compromised, all function secrets are immediately accessible.

**Affected Code:**
```go
secret := &store.Secret{
    Name:  req.Name,
    Value: req.Value, // In production, encrypt this
}
```

**Remediation:**
1. Implement encryption at rest using a key derived from environment variable
2. Use envelope encryption with key rotation support

**Status:** FIXED

---

### SEC-008: Storage UpdateObject Missing Access Control (HIGH)

**Location:** `app/web/handler/api/storage.go:405-454`

**Description:**
The `UpdateObject` method does not call `checkStorageAccess` before allowing object updates, unlike `UploadObject` and `DeleteObject`.

**Risk:** Unauthorized file modification. Users can update any object regardless of permissions.

**Remediation:**
1. Add `checkStorageAccess` call with `StorageAccessWrite` level

**Status:** FIXED

---

### SEC-009: Missing Authentication on Realtime API (HIGH)

**Location:** `app/web/server.go:162-165`

**Description:**
The `/api/realtime/*` endpoints have no authentication middleware.

**Risk:** Information disclosure. Attackers can enumerate active channels and connection statistics.

**Remediation:**
1. Apply `apiKeyMw` to `/api/realtime/*` group

**Status:** FIXED

---

### SEC-010: Signed URL Token Not Validated (HIGH)

**Location:** `app/web/handler/api/storage.go:766-771`

**Description:**
When creating signed URLs, a random UUID token is generated but never stored. There's no mechanism to validate the token when the signed URL is accessed.

**Risk:** Signed URLs provide no actual security. Any URL in the signed format would be accepted.

**Remediation:**
1. Store signed URL tokens with expiration in database
2. Validate token and expiration on signed URL access
3. Implement HMAC-based signatures as an alternative

**Status:** FIXED - Implemented HMAC-based signature validation

---

## Medium Severity Vulnerabilities

### SEC-011: No Rate Limiting on Authentication Endpoints (MEDIUM)

**Location:** `app/web/handler/api/auth.go` (all endpoints)

**Description:**
Authentication endpoints like `/auth/v1/token` have no rate limiting, making them vulnerable to brute force attacks.

**Risk:** Account compromise through password brute forcing.

**Remediation:**
1. Implement rate limiting middleware
2. Add exponential backoff for failed login attempts
3. Consider implementing account lockout

**Status:** FIXED - Added rate limiting middleware

---

### SEC-012: JWT Expiration Not Checked When Validation Disabled (MEDIUM)

**Location:** `app/web/middleware/apikey.go:230-248`

**Description:**
When `ValidateSignature` is false or when using known keys, the `parseJWTClaims` function doesn't verify token expiration.

**Risk:** Expired tokens can be used indefinitely.

**Remediation:**
1. Always check expiration regardless of signature validation setting

**Status:** FIXED

---

### SEC-013: Path Traversal Risk in Storage Paths (MEDIUM)

**Location:** `app/web/handler/api/storage.go:309`

**Description:**
Object paths are not validated for path traversal sequences like `../`.

**Risk:** Potential access to files outside the intended bucket.

**Remediation:**
1. Sanitize paths to remove `..` sequences
2. Validate that normalized path doesn't escape bucket root

**Status:** FIXED

---

### SEC-014: Content-Disposition Header Injection (MEDIUM)

**Location:** `app/web/handler/api/storage.go:480`

**Description:**
The filename in Content-Disposition header is taken directly from the object path without sanitization.

**Risk:** HTTP header injection if filename contains newlines or special characters.

**Affected Code:**
```go
c.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(obj.Name))
```

**Remediation:**
1. Sanitize filename to remove special characters
2. Use proper RFC 5987 encoding for non-ASCII filenames

**Status:** FIXED

---

### SEC-015: Insecure Password Policy (MEDIUM)

**Location:** `app/web/handler/api/auth.go:108`

**Description:**
No password complexity requirements. Users can set empty or single-character passwords.

**Risk:** Weak passwords are easily compromised.

**Remediation:**
1. Implement minimum password length (8+ characters)
2. Consider password strength validation

**Status:** FIXED

---

## Additional Vulnerabilities Found (Second Review)

### SEC-016: WebSocket CORS Not Configured (MEDIUM)

**Location:** `app/web/handler/api/realtime.go:24-27`

**Description:**
The WebSocket upgrader doesn't have `CheckOrigin` configured, meaning it accepts WebSocket connections from any origin by default.

**Risk:** WebSocket hijacking from malicious websites. An attacker could create a malicious page that connects to the WebSocket endpoint from a different origin.

**Affected Code:**
```go
upgrader: websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    // Missing CheckOrigin
},
```

**Remediation:**
1. Configure CheckOrigin to validate the request origin
2. Only allow connections from trusted domains

**Status:** FIXED

---

### SEC-017: WebSocket Authentication Missing (HIGH)

**Location:** `app/web/handler/api/realtime.go:58-98`

**Description:**
The WebSocket endpoint at `/realtime/v1/websocket` doesn't verify authentication before accepting connections. Anyone can connect and receive realtime data.

**Risk:** Unauthorized access to realtime database changes. Attackers can subscribe to data changes without authentication.

**Remediation:**
1. Apply authentication middleware to WebSocket endpoint
2. Validate JWT token from query parameter or Authorization header

**Status:** FIXED

---

### SEC-018: Dashboard API Missing Authentication (HIGH)

**Location:** `app/web/server.go:171-174`

**Description:**
The `/api/dashboard/*` endpoints expose system statistics without authentication, including user counts, function counts, and database table counts.

**Risk:** Information disclosure. Attackers can enumerate system resources and potentially use this information for targeted attacks.

**Remediation:**
1. Apply authentication middleware to dashboard endpoints
2. Require at least `authenticated` role for stats access

**Status:** FIXED

---

### SEC-019: MFA Secret Stored Unencrypted (HIGH)

**Location:** `store/postgres/auth.go:404-418`

**Description:**
MFA TOTP secrets are stored in plaintext in the database. If the database is compromised, all MFA secrets are exposed.

**Risk:** MFA bypass. Attackers with database access can generate valid TOTP codes for any user's MFA factor.

**Remediation:**
1. Encrypt MFA secrets at rest using envelope encryption
2. Use a separate key management system for encryption keys

**Status:** DOCUMENTED (requires key management infrastructure)

---

### SEC-020: Seeded Password Hash Hardcoded (LOW)

**Location:** `store/postgres/store.go:320`

**Description:**
The seed users have a hardcoded password hash for "password123". This is acceptable for development but could be a risk if seed data is accidentally used in production.

**Risk:** Known credentials could allow unauthorized access if seed data leaks to production.

**Remediation:**
1. Document that seed data is for development only
2. Generate random passwords for seed users
3. Add a production check that prevents seeding

**Status:** DOCUMENTED

---

## Additional Security Recommendations

### SEC-R01: Implement HTTPS Enforcement
Currently, there's no enforcement of HTTPS. Consider adding HSTS headers and redirect middleware.

### SEC-R02: Add Security Headers
Missing security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Content-Security-Policy`
- `Strict-Transport-Security`

### SEC-R03: Implement Audit Logging
Critical operations (authentication, authorization changes, data access) should be logged for security monitoring.

### SEC-R04: Add Request Size Limits
Consider adding middleware to limit request body sizes to prevent DoS attacks.

### SEC-R05: Database Connection Security
Ensure database connections use TLS in production environments.

---

## Summary of Changes Made

| Issue ID | Severity | Status | File(s) Modified |
|----------|----------|--------|------------------|
| SEC-001 | CRITICAL | FIXED | `middleware/apikey.go`, `auth.go` |
| SEC-002 | CRITICAL | FIXED | `middleware/apikey.go` |
| SEC-003 | CRITICAL | FIXED | `database.go`, `server.go` |
| SEC-004 | CRITICAL | FIXED | `server.go` |
| SEC-005 | HIGH | FIXED | `auth.go` |
| SEC-006 | HIGH | FIXED | `server.go` |
| SEC-007 | HIGH | FIXED | `functions.go`, `store.go` |
| SEC-008 | HIGH | FIXED | `storage.go` |
| SEC-009 | HIGH | FIXED | `server.go` |
| SEC-010 | HIGH | FIXED | `storage.go` |
| SEC-011 | MEDIUM | FIXED | `middleware/ratelimit.go`, `server.go` |
| SEC-012 | MEDIUM | FIXED | `middleware/apikey.go` |
| SEC-013 | MEDIUM | FIXED | `storage.go` |
| SEC-014 | MEDIUM | FIXED | `storage.go` |
| SEC-015 | MEDIUM | FIXED | `auth.go` |
| SEC-016 | MEDIUM | FIXED | `realtime.go` |
| SEC-017 | HIGH | FIXED | `realtime.go`, `server.go` |
| SEC-018 | HIGH | FIXED | `server.go` |
| SEC-019 | HIGH | DOCUMENTED | N/A (requires key mgmt) |
| SEC-020 | LOW | DOCUMENTED | N/A (dev only) |

---

## Test Coverage

Security tests have been added in `test/security_test.go`:
- JWT token validation tests
- Authentication bypass tests
- SQL injection tests
- Path traversal tests
- Rate limiting tests
- MFA verification tests
- Signed URL validation tests

---

*Report generated by security audit tool*
