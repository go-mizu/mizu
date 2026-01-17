# Localbase Auth - Supabase GoTrue API Compatibility Testing Plan

**Spec ID**: 0380
**Status**: Implemented
**Last Updated**: 2026-01-16

## Compatibility Status

| Category | Status | Details |
|----------|--------|---------|
| JWT Claims | **100%** | 14/14 claims matching |
| User Object | **100%** | 13/13 fields matching |
| Identity Object | **100%** | 9/9 fields matching |
| Error Format | **100%** | Supabase-compatible `{code, error_code, msg}` |
| Signup Endpoint | **100%** | Full compatibility |
| Login Endpoint | **100%** | Full compatibility |
| Refresh Token | **100%** | Full compatibility |
| Get User | **100%** | Full compatibility |

### JWT Claims (14/14)
- `aal`, `amr`, `app_metadata`, `aud`, `email`, `exp`, `iat`, `is_anonymous`, `iss`, `phone`, `role`, `session_id`, `sub`, `user_metadata`

### User Object Fields (13/13)
- `app_metadata`, `aud`, `created_at`, `email`, `email_confirmed_at`, `id`, `identities`, `is_anonymous`, `last_sign_in_at`, `phone`, `role`, `updated_at`, `user_metadata`

### Identity Object Fields (9/9)
- `created_at`, `email`, `id`, `identity_data`, `identity_id`, `last_sign_in_at`, `provider`, `updated_at`, `user_id`

---

## Overview

This document outlines a comprehensive testing plan to achieve 100% API compatibility between Localbase Auth and Supabase GoTrue (Supabase Auth). All tests will use real database connections (no mocks) and validate compatibility against Supabase Local running in parallel.

## Table of Contents

1. [API Endpoints Reference](#1-api-endpoints-reference)
2. [Error Codes Reference](#2-error-codes-reference)
3. [JWT Claims Structure](#3-jwt-claims-structure)
4. [Test Categories](#4-test-categories)
5. [Detailed Test Cases](#5-detailed-test-cases)
6. [Security Test Cases](#6-security-test-cases)
7. [Business Workflow Tests](#7-business-workflow-tests)
8. [Edge Cases](#8-edge-cases)
9. [Compatibility Verification Strategy](#9-compatibility-verification-strategy)

---

## 1. API Endpoints Reference

### 1.1 Core Authentication Endpoints

| Method | Endpoint | Description | Status |
|--------|----------|-------------|--------|
| POST | `/auth/v1/signup` | User registration | Implemented |
| POST | `/auth/v1/token?grant_type=password` | Password login | Implemented |
| POST | `/auth/v1/token?grant_type=refresh_token` | Refresh token | Implemented |
| POST | `/auth/v1/token?grant_type=id_token` | ID token exchange | **Missing** |
| POST | `/auth/v1/token?grant_type=pkce` | PKCE flow | **Missing** |
| POST | `/auth/v1/logout` | Logout user | Implemented |
| POST | `/auth/v1/logout?scope=global` | Logout all sessions | **Missing** |
| POST | `/auth/v1/logout?scope=others` | Logout other sessions | **Missing** |
| POST | `/auth/v1/recover` | Password recovery | Implemented |
| POST | `/auth/v1/magiclink` | Magic link (deprecated) | **Missing** |
| POST | `/auth/v1/otp` | Send OTP | Implemented |
| POST | `/auth/v1/verify` | Verify OTP/token | Implemented |
| GET | `/auth/v1/verify` | Verify via GET (email links) | **Missing** |
| POST | `/auth/v1/resend` | Resend OTP | **Missing** |
| GET | `/auth/v1/reauthenticate` | Request reauthentication | **Missing** |

### 1.2 User Management Endpoints

| Method | Endpoint | Description | Status |
|--------|----------|-------------|--------|
| GET | `/auth/v1/user` | Get current user | Implemented |
| PUT | `/auth/v1/user` | Update current user | Implemented |
| GET | `/auth/v1/user/identities` | List user identities | **Missing** |
| POST | `/auth/v1/user/identities/authorize` | Link identity | **Missing** |
| DELETE | `/auth/v1/user/identities/{id}` | Unlink identity | **Missing** |

### 1.3 MFA Endpoints

| Method | Endpoint | Description | Status |
|--------|----------|-------------|--------|
| POST | `/auth/v1/factors` | Enroll MFA factor | Implemented |
| GET | `/auth/v1/factors` | List MFA factors | **Missing** |
| DELETE | `/auth/v1/factors/{id}` | Unenroll factor | Implemented |
| POST | `/auth/v1/factors/{id}/challenge` | Create challenge | Implemented |
| POST | `/auth/v1/factors/{id}/verify` | Verify challenge | Implemented |

### 1.4 Admin Endpoints

| Method | Endpoint | Description | Status |
|--------|----------|-------------|--------|
| GET | `/auth/v1/admin/users` | List all users | Implemented |
| POST | `/auth/v1/admin/users` | Create user | Implemented |
| GET | `/auth/v1/admin/users/{id}` | Get user by ID | Implemented |
| PUT | `/auth/v1/admin/users/{id}` | Update user | Implemented |
| DELETE | `/auth/v1/admin/users/{id}` | Delete user | Implemented |
| POST | `/auth/v1/admin/invite` | Invite user by email | **Missing** |
| POST | `/auth/v1/admin/generate_link` | Generate auth link | **Missing** |
| GET | `/auth/v1/admin/audit` | Get audit logs | **Missing** |
| DELETE | `/auth/v1/admin/users/{id}/factors` | Delete all factors | **Missing** |
| DELETE | `/auth/v1/admin/users/{id}/sessions` | Delete user sessions | **Missing** |

### 1.5 OAuth/SSO Endpoints

| Method | Endpoint | Description | Status |
|--------|----------|-------------|--------|
| GET | `/auth/v1/authorize` | OAuth authorize | **Missing** |
| GET | `/auth/v1/callback` | OAuth callback | **Missing** |
| POST | `/auth/v1/sso` | Initiate SSO | **Missing** |
| GET | `/auth/v1/saml/metadata` | SAML metadata | **Missing** |
| POST | `/auth/v1/saml/acs` | SAML ACS | **Missing** |

### 1.6 System Endpoints

| Method | Endpoint | Description | Status |
|--------|----------|-------------|--------|
| GET | `/auth/v1/health` | Health check | **Missing** |
| GET | `/auth/v1/settings` | Public settings | **Missing** |

---

## 2. Error Codes Reference

All error responses must match Supabase's format:
```json
{
  "error": "error_type",
  "error_description": "Human readable message",
  "error_code": "specific_error_code",
  "msg": "Short message"
}
```

### 2.1 Authentication Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `invalid_credentials` | 401 | Login credentials or grant type not recognized |
| `bad_jwt` | 401 | JWT in Authorization header is not valid |
| `no_authorization` | 401 | Authorization header not provided |
| `user_banned` | 403 | User has active banned_until timestamp |

### 2.2 Email Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `email_exists` | 422 | Email address already registered |
| `email_not_confirmed` | 401 | Email address not confirmed |
| `email_address_invalid` | 422 | Invalid email format or test domain |
| `email_address_not_authorized` | 422 | Email domain not authorized |
| `email_provider_disabled` | 422 | Email signups disabled |
| `email_conflict_identity_not_deletable` | 422 | Unlinking causes email collision |

### 2.3 Phone Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `phone_exists` | 422 | Phone number already registered |
| `phone_not_confirmed` | 401 | Phone number not confirmed |
| `phone_provider_disabled` | 422 | Phone signups disabled |

### 2.4 Password Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `weak_password` | 422 | Password fails strength requirements |
| `same_password` | 422 | New password same as current |
| `reauthentication_needed` | 401 | Must reauthenticate to change password |
| `reauthentication_not_valid` | 401 | Reauthentication code incorrect |

### 2.5 Session/Token Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `session_expired` | 401 | Session exceeded timebox value |
| `session_not_found` | 401 | Session no longer exists |
| `refresh_token_already_used` | 401 | Token revoked or outside reuse interval |
| `refresh_token_not_found` | 401 | Refresh token not found |

### 2.6 MFA Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `insufficient_aal` | 403 | Higher AAL required for operation |
| `mfa_challenge_expired` | 401 | MFA challenge expired |
| `mfa_factor_not_found` | 404 | MFA factor not found |
| `mfa_factor_name_conflict` | 422 | Duplicate factor friendly name |
| `mfa_verification_failed` | 401 | Wrong TOTP code |
| `mfa_verification_rejected` | 401 | Hook rejected verification |
| `mfa_verified_factor_exists` | 422 | Verified phone factor exists |
| `mfa_totp_enroll_not_enabled` | 422 | TOTP enrollment disabled |
| `mfa_totp_verify_not_enabled` | 422 | TOTP verification disabled |
| `mfa_phone_enroll_not_enabled` | 422 | Phone MFA enrollment disabled |
| `mfa_phone_verify_not_enabled` | 422 | Phone MFA verification disabled |
| `mfa_web_authn_enroll_not_enabled` | 422 | WebAuthn enrollment disabled |
| `mfa_web_authn_verify_not_enabled` | 422 | WebAuthn verification disabled |
| `mfa_ip_address_mismatch` | 422 | Enrollment IP mismatch |
| `too_many_enrolled_mfa_factors` | 422 | Exceeded max factors (10) |

### 2.7 OAuth/SSO Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `bad_oauth_state` | 422 | OAuth state not in correct format |
| `bad_oauth_callback` | 422 | Missing OAuth required attributes |
| `oauth_provider_not_supported` | 422 | Provider disabled |
| `provider_disabled` | 422 | OAuth provider disabled |
| `provider_email_needs_verification` | 422 | OAuth email verification required |

### 2.8 SAML Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `saml_provider_disabled` | 422 | Enterprise SSO not enabled |
| `saml_idp_not_found` | 404 | SAML identity provider not found |
| `saml_idp_already_exists` | 422 | SAML provider already registered |
| `saml_assertion_no_email` | 422 | SAML response missing email |
| `saml_assertion_no_user_id` | 422 | SAML response missing NameID |
| `saml_entity_id_mismatch` | 422 | SAML metadata entity ID mismatch |
| `saml_metadata_fetch_failed` | 422 | SAML metadata URL unreachable |
| `saml_relay_state_expired` | 422 | SAML relay state timeout |
| `saml_relay_state_not_found` | 422 | SAML relay state not found |

### 2.9 PKCE Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `bad_code_verifier` | 422 | PKCE code verifier mismatch |
| `flow_state_expired` | 422 | PKCE flow state expired |
| `flow_state_not_found` | 422 | PKCE flow state not found |

### 2.10 Rate Limiting Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `over_request_rate_limit` | 429 | Too many requests from IP |
| `over_email_send_rate_limit` | 429 | Too many emails to address |
| `over_sms_send_rate_limit` | 429 | Too many SMS to phone |

### 2.11 User Management Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `user_already_exists` | 422 | User cannot be created again |
| `user_not_found` | 404 | User no longer exists |
| `not_admin` | 403 | User accessing API is not admin |
| `user_sso_managed` | 422 | SSO users cannot update certain fields |
| `single_identity_not_deletable` | 422 | User must have at least one identity |

### 2.12 Identity Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `identity_already_exists` | 422 | Identity already linked to user |
| `identity_not_found` | 404 | Identity does not exist |
| `manual_linking_disabled` | 422 | linkUser() API not enabled |

### 2.13 OTP Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `otp_disabled` | 422 | OTP sign-in disabled |
| `otp_expired` | 422 | OTP code expired |

### 2.14 Invite Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `invite_not_found` | 404 | Invite expired or already used |

### 2.15 Hook Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `hook_timeout` | 500 | Hook unreachable within time limit |
| `hook_timeout_after_retry` | 500 | Hook unreachable after retries |
| `hook_payload_invalid_content_type` | 500 | Invalid Content-Type header |
| `hook_payload_over_size_limit` | 500 | Payload exceeds max size |

### 2.16 Configuration Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `anonymous_provider_disabled` | 422 | Anonymous sign-ins disabled |
| `signup_disabled` | 422 | Sign ups disabled on server |
| `captcha_failed` | 422 | CAPTCHA verification failed |

### 2.17 General Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `bad_json` | 400 | Request body not valid JSON |
| `conflict` | 409 | General database conflict |
| `validation_failed` | 422 | Parameters not in expected format |
| `unexpected_failure` | 500 | Auth service degraded or bug |
| `request_timeout` | 504 | Request took too long |
| `unexpected_audience` | 401 | X-JWT-AUD claim mismatch |

---

## 3. JWT Claims Structure

### 3.1 Required Claims

| Claim | Type | Description |
|-------|------|-------------|
| `iss` | string | Issuer URL (e.g., `https://project.supabase.co/auth/v1`) |
| `aud` | string/array | Audience identifier (e.g., `authenticated`) |
| `exp` | number | Unix timestamp when token expires |
| `iat` | number | Unix timestamp when token was issued |
| `sub` | string (UUID) | User ID |
| `role` | string | User role (e.g., `authenticated`, `anon`, `service_role`) |
| `aal` | string | Authenticator Assurance Level (`aal1`, `aal2`) |
| `session_id` | string (UUID) | Unique session identifier |
| `email` | string | User's email address |
| `phone` | string | User's phone number |
| `is_anonymous` | boolean | Whether user is anonymous |

### 3.2 Optional Claims

| Claim | Type | Description |
|-------|------|-------------|
| `jti` | string | Unique JWT identifier |
| `nbf` | number | Not valid before timestamp |
| `app_metadata` | object | Application-specific metadata |
| `user_metadata` | object | User-specific custom data |
| `amr` | array | Authentication methods used |

### 3.3 AMR (Authentication Method Reference) Format

```json
{
  "amr": [
    {"method": "password", "timestamp": 1234567890},
    {"method": "totp", "timestamp": 1234567891}
  ]
}
```

---

## 4. Test Categories

### 4.1 Unit Tests
- Request/response serialization
- JWT generation and validation
- Password hashing
- Token generation

### 4.2 Integration Tests
- Database operations
- Session management
- Refresh token rotation

### 4.3 API Compatibility Tests
- Request format matching
- Response format matching
- Error code matching
- HTTP status code matching

### 4.4 Security Tests
- SQL injection prevention
- XSS prevention
- CSRF protection
- Rate limiting
- Token security

### 4.5 Business Workflow Tests
- Complete user journeys
- Multi-step flows
- Edge cases

---

## 5. Detailed Test Cases

### 5.1 Signup Tests

```go
// File: auth_test.go

// T5.1.1: Basic email/password signup
func TestSignup_EmailPassword_Success(t *testing.T) {
    // Request: POST /auth/v1/signup
    // Body: {"email": "test@example.com", "password": "SecurePass123!"}
    // Expected: 201 Created
    // Response must include: access_token, refresh_token, user object
}

// T5.1.2: Signup with user metadata
func TestSignup_WithMetadata_Success(t *testing.T) {
    // Body: {"email": "...", "password": "...", "data": {"name": "John"}}
    // user.user_metadata should contain {"name": "John"}
}

// T5.1.3: Signup with phone number
func TestSignup_Phone_Success(t *testing.T) {
    // Body: {"phone": "+1234567890", "password": "..."}
}

// T5.1.4: Duplicate email
func TestSignup_DuplicateEmail_Error(t *testing.T) {
    // Expected: 422, error_code: "email_exists"
}

// T5.1.5: Duplicate phone
func TestSignup_DuplicatePhone_Error(t *testing.T) {
    // Expected: 422, error_code: "phone_exists"
}

// T5.1.6: Weak password
func TestSignup_WeakPassword_Error(t *testing.T) {
    // Expected: 422, error_code: "weak_password"
    // Response includes weak_password.reasons array
}

// T5.1.7: Invalid email format
func TestSignup_InvalidEmail_Error(t *testing.T) {
    // Expected: 422, error_code: "validation_failed"
}

// T5.1.8: Missing required fields
func TestSignup_MissingEmail_Error(t *testing.T) {
    // Body: {"password": "..."}
    // Expected: 400, error_code: "validation_failed"
}

// T5.1.9: Missing password
func TestSignup_MissingPassword_Error(t *testing.T) {
    // Expected: 400, error: "password required"
}

// T5.1.10: Signup disabled
func TestSignup_Disabled_Error(t *testing.T) {
    // Config: signup disabled
    // Expected: 422, error_code: "signup_disabled"
}

// T5.1.11: Anonymous signup
func TestSignup_Anonymous_Success(t *testing.T) {
    // POST /auth/v1/signup with empty body (or specific flag)
    // Expected: 201, user.is_anonymous: true
}

// T5.1.12: Email confirmation required
func TestSignup_EmailConfirmationRequired_Success(t *testing.T) {
    // Config: email confirmation enabled
    // Expected: 201, user.email_confirmed_at: null
    // Must NOT return access_token until confirmed
}
```

### 5.2 Token (Login) Tests

```go
// T5.2.1: Password grant - email
func TestToken_PasswordGrant_Email_Success(t *testing.T) {
    // POST /auth/v1/token?grant_type=password
    // Body: {"email": "...", "password": "..."}
    // Expected: 200
    // Response: access_token, refresh_token, expires_in, expires_at, user
}

// T5.2.2: Password grant - phone
func TestToken_PasswordGrant_Phone_Success(t *testing.T) {
    // Body: {"phone": "+1234567890", "password": "..."}
}

// T5.2.3: Invalid credentials
func TestToken_InvalidCredentials_Error(t *testing.T) {
    // Expected: 401, error_code: "invalid_credentials"
}

// T5.2.4: User not found
func TestToken_UserNotFound_Error(t *testing.T) {
    // Expected: 401, error_code: "invalid_credentials"
    // Note: Same error as invalid password (security)
}

// T5.2.5: Email not confirmed
func TestToken_EmailNotConfirmed_Error(t *testing.T) {
    // Expected: 401, error_code: "email_not_confirmed"
}

// T5.2.6: User banned
func TestToken_UserBanned_Error(t *testing.T) {
    // Expected: 403, error_code: "user_banned"
}

// T5.2.7: Refresh token grant
func TestToken_RefreshTokenGrant_Success(t *testing.T) {
    // POST /auth/v1/token?grant_type=refresh_token
    // Body: {"refresh_token": "..."}
    // Expected: 200, new access_token, new refresh_token
}

// T5.2.8: Invalid refresh token
func TestToken_InvalidRefreshToken_Error(t *testing.T) {
    // Expected: 401, error_code: "refresh_token_not_found"
}

// T5.2.9: Revoked refresh token
func TestToken_RevokedRefreshToken_Error(t *testing.T) {
    // Expected: 401, error_code: "refresh_token_already_used"
}

// T5.2.10: Refresh token rotation
func TestToken_RefreshTokenRotation_Success(t *testing.T) {
    // Old refresh token should be revoked after rotation
    // Attempting to use old token should fail
}

// T5.2.11: Refresh token reuse detection
func TestToken_RefreshTokenReuseDetection_Error(t *testing.T) {
    // Attempting to reuse a used refresh token should:
    // 1. Return error
    // 2. Revoke all descendant tokens (security)
}

// T5.2.12: PKCE grant
func TestToken_PKCEGrant_Success(t *testing.T) {
    // POST /auth/v1/token?grant_type=pkce
    // Body: {"code": "...", "code_verifier": "..."}
}

// T5.2.13: Invalid PKCE verifier
func TestToken_PKCEInvalidVerifier_Error(t *testing.T) {
    // Expected: 422, error_code: "bad_code_verifier"
}

// T5.2.14: ID token grant
func TestToken_IDTokenGrant_Success(t *testing.T) {
    // POST /auth/v1/token?grant_type=id_token
    // Body: {"id_token": "...", "provider": "google"}
}
```

### 5.3 Logout Tests

```go
// T5.3.1: Basic logout
func TestLogout_Success(t *testing.T) {
    // POST /auth/v1/logout
    // Header: Authorization: Bearer <token>
    // Expected: 204 No Content
}

// T5.3.2: Logout without auth
func TestLogout_NoAuth_Error(t *testing.T) {
    // Expected: 401, error_code: "no_authorization"
}

// T5.3.3: Global logout (all sessions)
func TestLogout_Global_Success(t *testing.T) {
    // POST /auth/v1/logout?scope=global
    // All refresh tokens should be revoked
}

// T5.3.4: Logout others
func TestLogout_Others_Success(t *testing.T) {
    // POST /auth/v1/logout?scope=others
    // Current session preserved, others revoked
}

// T5.3.5: Invalid token logout
func TestLogout_InvalidToken_Error(t *testing.T) {
    // Expected: 401, error_code: "bad_jwt"
}

// T5.3.6: Expired token logout
func TestLogout_ExpiredToken_Error(t *testing.T) {
    // Expected: 401, error_code: "session_expired" or "bad_jwt"
}
```

### 5.4 User Management Tests

```go
// T5.4.1: Get current user
func TestGetUser_Success(t *testing.T) {
    // GET /auth/v1/user
    // Expected: 200, full user object
}

// T5.4.2: Get user without auth
func TestGetUser_NoAuth_Error(t *testing.T) {
    // Expected: 401, error_code: "no_authorization"
}

// T5.4.3: Update user email
func TestUpdateUser_Email_Success(t *testing.T) {
    // PUT /auth/v1/user
    // Body: {"email": "newemail@example.com"}
    // Should trigger confirmation email
}

// T5.4.4: Update user password
func TestUpdateUser_Password_Success(t *testing.T) {
    // Body: {"password": "NewSecurePass123!"}
    // May require reauthentication
}

// T5.4.5: Update user metadata
func TestUpdateUser_Metadata_Success(t *testing.T) {
    // Body: {"data": {"name": "Jane"}}
}

// T5.4.6: Update with same password
func TestUpdateUser_SamePassword_Error(t *testing.T) {
    // Expected: 422, error_code: "same_password"
}

// T5.4.7: Update with weak password
func TestUpdateUser_WeakPassword_Error(t *testing.T) {
    // Expected: 422, error_code: "weak_password"
}
```

### 5.5 Password Recovery Tests

```go
// T5.5.1: Request password recovery
func TestRecover_Success(t *testing.T) {
    // POST /auth/v1/recover
    // Body: {"email": "user@example.com"}
    // Expected: 200 (always, even if user doesn't exist - security)
}

// T5.5.2: Recovery with redirect
func TestRecover_WithRedirect_Success(t *testing.T) {
    // Body: {"email": "...", "redirect_to": "https://app.com/reset"}
}

// T5.5.3: Verify recovery token
func TestVerify_Recovery_Success(t *testing.T) {
    // POST /auth/v1/verify
    // Body: {"type": "recovery", "token": "...", "email": "..."}
}

// T5.5.4: Invalid recovery token
func TestVerify_InvalidRecoveryToken_Error(t *testing.T) {
    // Expected: 422, error_code: "otp_expired" or validation error
}
```

### 5.6 OTP Tests

```go
// T5.6.1: Send OTP to email
func TestOTP_Email_Success(t *testing.T) {
    // POST /auth/v1/otp
    // Body: {"email": "user@example.com"}
    // Expected: 200
}

// T5.6.2: Send OTP to phone
func TestOTP_Phone_Success(t *testing.T) {
    // Body: {"phone": "+1234567890"}
}

// T5.6.3: Send OTP with channel (SMS/WhatsApp)
func TestOTP_Channel_Success(t *testing.T) {
    // Body: {"phone": "...", "channel": "whatsapp"}
}

// T5.6.4: Verify OTP - signup
func TestVerify_OTP_Signup_Success(t *testing.T) {
    // Body: {"type": "signup", "token": "123456", "email": "..."}
}

// T5.6.5: Verify OTP - magiclink
func TestVerify_OTP_Magiclink_Success(t *testing.T) {
    // Body: {"type": "magiclink", "token": "...", "email": "..."}
}

// T5.6.6: Expired OTP
func TestVerify_OTP_Expired_Error(t *testing.T) {
    // Expected: 422, error_code: "otp_expired"
}

// T5.6.7: Invalid OTP
func TestVerify_OTP_Invalid_Error(t *testing.T) {
    // Expected: 422, error_code: "otp_expired" (same as expired for security)
}

// T5.6.8: OTP disabled
func TestOTP_Disabled_Error(t *testing.T) {
    // Config: OTP disabled
    // Expected: 422, error_code: "otp_disabled"
}

// T5.6.9: Resend OTP
func TestResend_OTP_Success(t *testing.T) {
    // POST /auth/v1/resend
    // Body: {"type": "signup", "email": "..."}
}
```

### 5.7 MFA Tests

```go
// T5.7.1: Enroll TOTP factor
func TestMFA_EnrollTOTP_Success(t *testing.T) {
    // POST /auth/v1/factors
    // Body: {"factor_type": "totp", "friendly_name": "My Authenticator"}
    // Expected: 201
    // Response: id, type, totp.qr_code, totp.secret, totp.uri
}

// T5.7.2: Enroll phone factor
func TestMFA_EnrollPhone_Success(t *testing.T) {
    // Body: {"factor_type": "phone", "phone": "+1234567890"}
}

// T5.7.3: Enroll WebAuthn factor
func TestMFA_EnrollWebAuthn_Success(t *testing.T) {
    // Body: {"factor_type": "web_authn"}
}

// T5.7.4: List factors
func TestMFA_ListFactors_Success(t *testing.T) {
    // GET /auth/v1/factors
    // Expected: 200, array of factors
}

// T5.7.5: Challenge TOTP factor
func TestMFA_ChallengeTOTP_Success(t *testing.T) {
    // POST /auth/v1/factors/{id}/challenge
    // Expected: 200, challenge_id, expires_at
}

// T5.7.6: Verify TOTP challenge
func TestMFA_VerifyTOTP_Success(t *testing.T) {
    // POST /auth/v1/factors/{id}/verify
    // Body: {"challenge_id": "...", "code": "123456"}
    // Expected: 200, new access_token with aal2
}

// T5.7.7: Wrong TOTP code
func TestMFA_VerifyTOTP_WrongCode_Error(t *testing.T) {
    // Expected: 401, error_code: "mfa_verification_failed"
}

// T5.7.8: Expired challenge
func TestMFA_ExpiredChallenge_Error(t *testing.T) {
    // Expected: 401, error_code: "mfa_challenge_expired"
}

// T5.7.9: Unenroll factor
func TestMFA_Unenroll_Success(t *testing.T) {
    // DELETE /auth/v1/factors/{id}
    // Expected: 200
}

// T5.7.10: Unenroll requires aal2
func TestMFA_Unenroll_RequiresAAL2_Error(t *testing.T) {
    // When user has verified factors, must have aal2 to unenroll
    // Expected: 403, error_code: "insufficient_aal"
}

// T5.7.11: Factor not found
func TestMFA_FactorNotFound_Error(t *testing.T) {
    // Expected: 404, error_code: "mfa_factor_not_found"
}

// T5.7.12: Duplicate factor name
func TestMFA_DuplicateName_Error(t *testing.T) {
    // Expected: 422, error_code: "mfa_factor_name_conflict"
}

// T5.7.13: Too many factors
func TestMFA_TooManyFactors_Error(t *testing.T) {
    // Limit: 10 factors
    // Expected: 422, error_code: "too_many_enrolled_mfa_factors"
}

// T5.7.14: TOTP enrollment disabled
func TestMFA_TOTPDisabled_Error(t *testing.T) {
    // Expected: 422, error_code: "mfa_totp_enroll_not_enabled"
}

// T5.7.15: AAL check
func TestMFA_GetAAL_Success(t *testing.T) {
    // JWT should include aal claim
    // aal1: password/otp/social login only
    // aal2: + MFA verified
}
```

### 5.8 Admin Tests

```go
// T5.8.1: List users
func TestAdmin_ListUsers_Success(t *testing.T) {
    // GET /auth/v1/admin/users
    // Header: Authorization: Bearer <service_role_token>
    // Expected: 200, users array, total count
}

// T5.8.2: List users with pagination
func TestAdmin_ListUsers_Pagination_Success(t *testing.T) {
    // GET /auth/v1/admin/users?page=1&per_page=10
}

// T5.8.3: Create user as admin
func TestAdmin_CreateUser_Success(t *testing.T) {
    // POST /auth/v1/admin/users
    // Body: {"email": "...", "password": "...", "email_confirm": true}
}

// T5.8.4: Create user with custom role
func TestAdmin_CreateUser_CustomRole_Success(t *testing.T) {
    // Body: {"email": "...", "password": "...", "role": "admin"}
}

// T5.8.5: Get user by ID
func TestAdmin_GetUser_Success(t *testing.T) {
    // GET /auth/v1/admin/users/{id}
}

// T5.8.6: Update user as admin
func TestAdmin_UpdateUser_Success(t *testing.T) {
    // PUT /auth/v1/admin/users/{id}
    // Body: {"email": "...", "email_confirm": true}
}

// T5.8.7: Ban user
func TestAdmin_BanUser_Success(t *testing.T) {
    // PUT /auth/v1/admin/users/{id}
    // Body: {"ban_duration": "100h"} or {"banned_until": "..."}
}

// T5.8.8: Unban user
func TestAdmin_UnbanUser_Success(t *testing.T) {
    // Body: {"ban_duration": "none"} or {"banned_until": null}
}

// T5.8.9: Delete user
func TestAdmin_DeleteUser_Success(t *testing.T) {
    // DELETE /auth/v1/admin/users/{id}
    // Expected: 200 or 204
}

// T5.8.10: Non-admin access
func TestAdmin_NonAdmin_Error(t *testing.T) {
    // Using regular user token
    // Expected: 403, error_code: "not_admin"
}

// T5.8.11: User not found
func TestAdmin_UserNotFound_Error(t *testing.T) {
    // Expected: 404, error_code: "user_not_found"
}

// T5.8.12: Invite user by email
func TestAdmin_InviteUser_Success(t *testing.T) {
    // POST /auth/v1/admin/invite
    // Body: {"email": "newuser@example.com"}
}

// T5.8.13: Generate magic link
func TestAdmin_GenerateLink_Success(t *testing.T) {
    // POST /auth/v1/admin/generate_link
    // Body: {"type": "magiclink", "email": "..."}
}

// T5.8.14: Generate signup link
func TestAdmin_GenerateSignupLink_Success(t *testing.T) {
    // Body: {"type": "signup", "email": "...", "password": "..."}
}

// T5.8.15: Generate recovery link
func TestAdmin_GenerateRecoveryLink_Success(t *testing.T) {
    // Body: {"type": "recovery", "email": "..."}
}
```

---

## 6. Security Test Cases

### 6.1 SQL Injection Prevention

```go
// T6.1.1: Email field SQL injection
func TestSecurity_SQLInjection_Email(t *testing.T) {
    // Email: "'; DROP TABLE auth.users; --"
    // Expected: Validation error, no SQL execution
}

// T6.1.2: Password field SQL injection
func TestSecurity_SQLInjection_Password(t *testing.T)

// T6.1.3: User ID parameter SQL injection
func TestSecurity_SQLInjection_UserID(t *testing.T)

// T6.1.4: Metadata JSON SQL injection
func TestSecurity_SQLInjection_Metadata(t *testing.T)
```

### 6.2 JWT Security

```go
// T6.2.1: JWT signature validation
func TestSecurity_JWT_InvalidSignature(t *testing.T) {
    // Tampered JWT should be rejected
    // Expected: 401, error_code: "bad_jwt"
}

// T6.2.2: JWT algorithm confusion
func TestSecurity_JWT_AlgorithmConfusion(t *testing.T) {
    // Attempt to use "none" algorithm
    // Expected: 401, error_code: "bad_jwt"
}

// T6.2.3: Expired JWT
func TestSecurity_JWT_Expired(t *testing.T) {
    // Expected: 401
}

// T6.2.4: Future nbf claim
func TestSecurity_JWT_FutureNBF(t *testing.T)

// T6.2.5: Wrong audience
func TestSecurity_JWT_WrongAudience(t *testing.T) {
    // Expected: 401, error_code: "unexpected_audience"
}
```

### 6.3 Password Security

```go
// T6.3.1: Bcrypt cost factor
func TestSecurity_Password_BcryptCost(t *testing.T) {
    // Verify bcrypt cost >= 10
}

// T6.3.2: Password in response
func TestSecurity_Password_NotInResponse(t *testing.T) {
    // Password hash should never appear in API responses
}

// T6.3.3: Password minimum length
func TestSecurity_Password_MinLength(t *testing.T) {
    // Default minimum: 6 characters
}

// T6.3.4: Password complexity
func TestSecurity_Password_Complexity(t *testing.T) {
    // Configurable complexity requirements
}

// T6.3.5: Common password rejection
func TestSecurity_Password_CommonPasswords(t *testing.T)
```

### 6.4 Rate Limiting

```go
// T6.4.1: Signup rate limit
func TestSecurity_RateLimit_Signup(t *testing.T) {
    // Exceed rate limit
    // Expected: 429, error_code: "over_request_rate_limit"
}

// T6.4.2: Login rate limit
func TestSecurity_RateLimit_Login(t *testing.T)

// T6.4.3: Email send rate limit
func TestSecurity_RateLimit_Email(t *testing.T) {
    // Expected: 429, error_code: "over_email_send_rate_limit"
}

// T6.4.4: SMS send rate limit
func TestSecurity_RateLimit_SMS(t *testing.T) {
    // Expected: 429, error_code: "over_sms_send_rate_limit"
}

// T6.4.5: Rate limit by IP
func TestSecurity_RateLimit_ByIP(t *testing.T)

// T6.4.6: Rate limit headers
func TestSecurity_RateLimit_Headers(t *testing.T) {
    // X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
}
```

### 6.5 Session Security

```go
// T6.5.1: Session hijacking prevention
func TestSecurity_Session_HijackPrevention(t *testing.T) {
    // Different IP/User-Agent should trigger security check
}

// T6.5.2: Concurrent session limit
func TestSecurity_Session_ConcurrentLimit(t *testing.T)

// T6.5.3: Session timeout
func TestSecurity_Session_Timeout(t *testing.T)

// T6.5.4: Refresh token binding
func TestSecurity_RefreshToken_Binding(t *testing.T) {
    // Refresh token should be bound to session
}
```

### 6.6 XSS Prevention

```go
// T6.6.1: User metadata XSS
func TestSecurity_XSS_Metadata(t *testing.T) {
    // Metadata: {"name": "<script>alert('xss')</script>"}
    // Should be escaped or rejected
}

// T6.6.2: Email XSS
func TestSecurity_XSS_Email(t *testing.T)
```

### 6.7 CSRF Protection

```go
// T6.7.1: State parameter validation
func TestSecurity_CSRF_StateParam(t *testing.T) {
    // OAuth state must match
}
```

---

## 7. Business Workflow Tests

### 7.1 Complete Signup Flow

```go
// T7.1.1: Email confirmation flow
func TestWorkflow_SignupWithEmailConfirmation(t *testing.T) {
    // 1. Signup -> returns user without tokens
    // 2. Confirm email via token
    // 3. Login succeeds
}

// T7.1.2: Phone confirmation flow
func TestWorkflow_SignupWithPhoneConfirmation(t *testing.T)

// T7.1.3: Signup then immediately login
func TestWorkflow_SignupThenLogin(t *testing.T)
```

### 7.2 Password Recovery Flow

```go
// T7.2.1: Complete password recovery
func TestWorkflow_PasswordRecovery(t *testing.T) {
    // 1. Request recovery
    // 2. Verify recovery token
    // 3. Set new password
    // 4. Login with new password
}

// T7.2.2: Recovery invalidates old sessions
func TestWorkflow_RecoveryInvalidatesSessions(t *testing.T)
```

### 7.3 MFA Enrollment Flow

```go
// T7.3.1: Complete TOTP enrollment
func TestWorkflow_TOTPEnrollment(t *testing.T) {
    // 1. Login (aal1)
    // 2. Enroll TOTP
    // 3. Verify TOTP (factor becomes verified)
    // 4. Logout
    // 5. Login -> aal1
    // 6. Challenge TOTP
    // 7. Verify -> aal2
}

// T7.3.2: MFA required for sensitive operations
func TestWorkflow_MFARequired(t *testing.T)
```

### 7.4 Email Change Flow

```go
// T7.4.1: Email change with confirmation
func TestWorkflow_EmailChange(t *testing.T) {
    // 1. Request email change
    // 2. Confirm new email
    // 3. New email active
}

// T7.4.2: Email change requires reauthentication
func TestWorkflow_EmailChangeReauth(t *testing.T)
```

### 7.5 OAuth Linking Flow

```go
// T7.5.1: Link OAuth identity
func TestWorkflow_LinkOAuth(t *testing.T) {
    // 1. User exists with email/password
    // 2. Link Google identity
    // 3. Can login with either method
}

// T7.5.2: Unlink identity
func TestWorkflow_UnlinkIdentity(t *testing.T)

// T7.5.3: Cannot unlink last identity
func TestWorkflow_CannotUnlinkLastIdentity(t *testing.T)
```

### 7.6 Session Management Flow

```go
// T7.6.1: Multiple devices
func TestWorkflow_MultipleDevices(t *testing.T) {
    // 1. Login on device A
    // 2. Login on device B
    // 3. Both sessions active
}

// T7.6.2: Logout from all devices
func TestWorkflow_LogoutAllDevices(t *testing.T) {
    // 1. Login on multiple devices
    // 2. Global logout
    // 3. All sessions invalid
}
```

---

## 8. Edge Cases

### 8.1 Concurrent Operations

```go
// T8.1.1: Concurrent signup same email
func TestEdge_ConcurrentSignup(t *testing.T) {
    // Race condition: same email signup
    // Expected: One succeeds, others get email_exists
}

// T8.1.2: Concurrent refresh token use
func TestEdge_ConcurrentRefresh(t *testing.T) {
    // Expected: One succeeds, others fail
}

// T8.1.3: Concurrent password change
func TestEdge_ConcurrentPasswordChange(t *testing.T)
```

### 8.2 Unicode and Special Characters

```go
// T8.2.1: Unicode email local part
func TestEdge_UnicodeEmail(t *testing.T) {
    // Email: "用户@example.com" (if supported)
}

// T8.2.2: Unicode password
func TestEdge_UnicodePassword(t *testing.T) {
    // Password with emoji, CJK characters
}

// T8.2.3: Unicode metadata
func TestEdge_UnicodeMetadata(t *testing.T)

// T8.2.4: Very long values
func TestEdge_VeryLongEmail(t *testing.T) {
    // 254 character email (RFC max)
}
```

### 8.3 Boundary Conditions

```go
// T8.3.1: Empty string handling
func TestEdge_EmptyStrings(t *testing.T)

// T8.3.2: Null handling in metadata
func TestEdge_NullMetadata(t *testing.T)

// T8.3.3: Zero timestamp
func TestEdge_ZeroTimestamp(t *testing.T)

// T8.3.4: Maximum page size
func TestEdge_MaxPageSize(t *testing.T)
```

### 8.4 Token Edge Cases

```go
// T8.4.1: Just-expired token
func TestEdge_JustExpiredToken(t *testing.T)

// T8.4.2: Token with no claims
func TestEdge_EmptyToken(t *testing.T)

// T8.4.3: Malformed bearer header
func TestEdge_MalformedBearer(t *testing.T) {
    // "Bearer" (no token)
    // "bearer token" (lowercase)
    // "BearerTOKEN" (no space)
}
```

### 8.5 Database Edge Cases

```go
// T8.5.1: Database connection failure
func TestEdge_DBConnectionFailure(t *testing.T)

// T8.5.2: Database timeout
func TestEdge_DBTimeout(t *testing.T)

// T8.5.3: Constraint violation
func TestEdge_ConstraintViolation(t *testing.T)
```

---

## 9. Compatibility Verification Strategy

### 9.1 Parallel Testing Architecture

```
┌─────────────────┐     ┌─────────────────┐
│   Test Runner   │     │   Test Runner   │
│  (Localbase)    │     │  (Supabase)     │
└────────┬────────┘     └────────┬────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│   Localbase     │     │   Supabase      │
│   Auth API      │     │   Local Auth    │
│  localhost:8080 │     │  localhost:54321│
└────────┬────────┘     └────────┬────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│   PostgreSQL    │     │   PostgreSQL    │
│   (localbase)   │     │   (supabase)    │
└─────────────────┘     └─────────────────┘
```

### 9.2 Test Execution Flow

```go
func TestCompatibility(t *testing.T) {
    // 1. Send identical request to both services
    localbaseResp := httpClient.Post(localbaseURL + "/auth/v1/signup", body)
    supabaseResp := httpClient.Post(supabaseURL + "/auth/v1/signup", body)

    // 2. Compare responses
    compareStatusCode(t, localbaseResp, supabaseResp)
    compareErrorCode(t, localbaseResp, supabaseResp)
    compareResponseStructure(t, localbaseResp, supabaseResp)
}
```

### 9.3 Response Comparison Rules

1. **Status Code**: Must be identical
2. **Error Code**: Must be identical for error responses
3. **Response Structure**: All required fields must be present
4. **Field Types**: Must match (string, number, boolean, object, array)
5. **Optional Fields**: Should be present when Supabase includes them
6. **Value Comparison**: Skip dynamic values (tokens, timestamps, UUIDs)

### 9.4 Known Differences (Acceptable)

| Field | Localbase | Supabase | Reason |
|-------|-----------|----------|--------|
| `user.id` | ULID | UUID | ID format |
| `access_token` | Different | Different | Unique generation |
| `refresh_token` | Different | Different | Unique generation |
| `created_at` | Different | Different | Timestamp |
| `updated_at` | Different | Different | Timestamp |

### 9.5 Test Data Setup

```go
var testUsers = []TestUser{
    {Email: "test1@example.com", Password: "TestPass123!"},
    {Email: "test2@example.com", Password: "TestPass123!"},
    {Phone: "+15551234567", Password: "TestPass123!"},
}

func setupTestData(t *testing.T) {
    // Create identical test data in both databases
}
```

### 9.6 Continuous Compatibility Testing

```yaml
# GitHub Actions workflow
name: Compatibility Tests

on:
  push:
  pull_request:

jobs:
  compatibility:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
      supabase:
        image: supabase/gotrue:latest
    steps:
      - uses: actions/checkout@v3
      - name: Start Localbase
        run: make start
      - name: Run Compatibility Tests
        run: go test -v ./tests/compatibility/...
```

---

## 10. Implementation Priority

### Phase 1: Core Auth (High Priority)
1. Fix error response format to match Supabase
2. Implement missing JWT claims (aal, session_id, is_anonymous, amr)
3. Add health and settings endpoints
4. Fix logout scope support

### Phase 2: Security (High Priority)
1. Implement rate limiting
2. Add password strength validation
3. Implement refresh token rotation properly
4. Add PKCE flow support

### Phase 3: MFA (Medium Priority)
1. Fix factor enrollment response format
2. Add factor listing endpoint
3. Implement proper challenge/verify flow
4. Add WebAuthn support

### Phase 4: Admin (Medium Priority)
1. Add invite user endpoint
2. Add generate link endpoint
3. Add audit log endpoint
4. Proper admin token validation

### Phase 5: OAuth/SSO (Lower Priority)
1. OAuth provider support
2. SAML support
3. Identity management endpoints

---

## 11. Test File Structure

```
localbase/
├── tests/
│   ├── auth/
│   │   ├── signup_test.go
│   │   ├── token_test.go
│   │   ├── logout_test.go
│   │   ├── user_test.go
│   │   ├── recover_test.go
│   │   ├── otp_test.go
│   │   ├── verify_test.go
│   │   └── mfa_test.go
│   ├── admin/
│   │   ├── users_test.go
│   │   ├── invite_test.go
│   │   └── audit_test.go
│   ├── security/
│   │   ├── injection_test.go
│   │   ├── jwt_test.go
│   │   ├── ratelimit_test.go
│   │   └── session_test.go
│   ├── workflow/
│   │   ├── signup_flow_test.go
│   │   ├── recovery_flow_test.go
│   │   ├── mfa_flow_test.go
│   │   └── oauth_flow_test.go
│   ├── edge/
│   │   ├── concurrent_test.go
│   │   ├── unicode_test.go
│   │   └── boundary_test.go
│   └── compatibility/
│       ├── setup_test.go
│       ├── response_compare_test.go
│       └── parallel_runner_test.go
├── testdata/
│   ├── fixtures/
│   │   ├── users.json
│   │   └── tokens.json
│   └── golden/
│       ├── signup_response.json
│       └── error_response.json
└── testutil/
    ├── client.go
    ├── compare.go
    └── setup.go
```

---

## 12. Running Tests

```bash
# Run all auth tests
make test-auth

# Run compatibility tests against Supabase Local
make test-compat

# Run security tests
make test-security

# Run with coverage
make test-coverage

# Run specific test
go test -v -run TestSignup_EmailPassword_Success ./tests/auth/
```

---

## Appendix A: Request/Response Examples

### A.1 Signup Request/Response

**Request:**
```http
POST /auth/v1/signup HTTP/1.1
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "data": {
    "name": "John Doe"
  }
}
```

**Response (Success):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer",
  "expires_in": 3600,
  "expires_at": 1737018000,
  "refresh_token": "abc123...",
  "user": {
    "id": "01HQWX5...",
    "aud": "authenticated",
    "role": "authenticated",
    "email": "user@example.com",
    "email_confirmed_at": "2026-01-16T12:00:00Z",
    "phone": "",
    "confirmed_at": "2026-01-16T12:00:00Z",
    "last_sign_in_at": "2026-01-16T12:00:00Z",
    "app_metadata": {
      "provider": "email",
      "providers": ["email"]
    },
    "user_metadata": {
      "name": "John Doe"
    },
    "identities": [],
    "created_at": "2026-01-16T12:00:00Z",
    "updated_at": "2026-01-16T12:00:00Z"
  }
}
```

**Response (Error):**
```json
{
  "error": "invalid_request",
  "error_description": "Email address already registered",
  "error_code": "email_exists",
  "msg": "A user with this email address has already been registered"
}
```

### A.2 Token Request/Response

**Request (Password Grant):**
```http
POST /auth/v1/token?grant_type=password HTTP/1.1
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer",
  "expires_in": 3600,
  "expires_at": 1737018000,
  "refresh_token": "xyz789...",
  "user": { ... }
}
```

### A.3 MFA Enroll Response

```json
{
  "id": "01HQWX5...",
  "type": "totp",
  "friendly_name": "My Authenticator",
  "totp": {
    "qr_code": "data:image/svg+xml;base64,...",
    "secret": "JBSWY3DPEHPK3PXP",
    "uri": "otpauth://totp/Localbase:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Localbase"
  },
  "status": "unverified"
}
```

---

## Appendix B: Configuration Options

```yaml
# auth.yaml
auth:
  # JWT Configuration
  jwt:
    secret: "your-super-secret-jwt-key-min-32-chars"
    expiry: 3600  # seconds
    algorithm: HS256  # HS256, RS256, ES256

  # Signup Configuration
  signup:
    enabled: true
    require_email_confirmation: true
    double_confirm_changes: true

  # Password Configuration
  password:
    min_length: 6
    require_uppercase: false
    require_lowercase: false
    require_digit: false
    require_special: false

  # Rate Limiting
  rate_limit:
    signup: 5/hour
    login: 10/minute
    email: 3/hour
    sms: 3/hour

  # MFA Configuration
  mfa:
    totp_enabled: true
    phone_enabled: true
    webauthn_enabled: false
    max_factors: 10

  # Session Configuration
  session:
    timebox: 24h
    inactivity_timeout: 0  # disabled
```

---

## Appendix C: Database Schema Requirements

```sql
-- Required tables in auth schema
CREATE TABLE auth.users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(50) UNIQUE,
    encrypted_password VARCHAR(255),
    email_confirmed_at TIMESTAMPTZ,
    phone_confirmed_at TIMESTAMPTZ,
    confirmation_token VARCHAR(255),
    confirmation_sent_at TIMESTAMPTZ,
    recovery_token VARCHAR(255),
    recovery_sent_at TIMESTAMPTZ,
    email_change_token_new VARCHAR(255),
    email_change VARCHAR(255),
    email_change_sent_at TIMESTAMPTZ,
    last_sign_in_at TIMESTAMPTZ,
    raw_app_meta_data JSONB DEFAULT '{}'::jsonb,
    raw_user_meta_data JSONB DEFAULT '{}'::jsonb,
    is_super_admin BOOLEAN DEFAULT FALSE,
    role VARCHAR(255) DEFAULT 'authenticated',
    banned_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    reauthentication_token VARCHAR(255),
    reauthentication_sent_at TIMESTAMPTZ,
    is_anonymous BOOLEAN DEFAULT FALSE
);

CREATE TABLE auth.sessions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    factor_id UUID,
    aal VARCHAR(10) DEFAULT 'aal1',
    not_after TIMESTAMPTZ,
    ip VARCHAR(45),
    user_agent TEXT
);

CREATE TABLE auth.refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR(255) UNIQUE NOT NULL,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE CASCADE,
    parent VARCHAR(255),
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE auth.mfa_factors (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    friendly_name VARCHAR(255),
    factor_type VARCHAR(50) NOT NULL, -- totp, phone, web_authn
    status VARCHAR(50) NOT NULL, -- unverified, verified
    secret TEXT,
    phone VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, friendly_name)
);

CREATE TABLE auth.mfa_challenges (
    id UUID PRIMARY KEY,
    factor_id UUID REFERENCES auth.mfa_factors(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    verified_at TIMESTAMPTZ,
    ip_address VARCHAR(45)
);

CREATE TABLE auth.identities (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    provider VARCHAR(255) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    identity_data JSONB DEFAULT '{}'::jsonb,
    last_sign_in_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_id)
);

CREATE TABLE auth.audit_log_entries (
    id UUID PRIMARY KEY,
    instance_id UUID,
    payload JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address VARCHAR(45)
);
```

---

*Document End*
