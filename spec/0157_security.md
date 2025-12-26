# Security Enhancement Plan for Messaging Blueprint

## Overview

This document outlines a comprehensive security enhancement plan for the Mizu Messaging Blueprint, based on modern security best practices from applications like Signal, WhatsApp, Telegram, and industry standards (OWASP).

## Current Security Audit Summary

### Existing Strengths
1. **Password Hashing**: Bcrypt with cost factor 12 (industry standard)
2. **Session Tokens**: 32 bytes cryptographically random, base64-encoded
3. **Cookie Security**: HttpOnly, Secure (on TLS), SameSite=Lax
4. **Ownership Checks**: Users can only edit/delete their own messages
5. **Parameterized Queries**: SQL injection protection via DuckDB driver
6. **Session Expiration**: 30-day session lifetime

### Critical Vulnerabilities Identified

| Severity | Issue | Risk |
|----------|-------|------|
| CRITICAL | WebSocket CheckOrigin always returns true | CSRF attacks on WebSocket |
| HIGH | No rate limiting on auth endpoints | Brute-force attacks |
| HIGH | Weak password requirements (6 chars) | Account compromise |
| MEDIUM | No CSRF token for state-changing ops | Cross-site request forgery |
| MEDIUM | No input sanitization for XSS | Stored XSS attacks |
| MEDIUM | Session tokens in URL query params | Token leakage in logs |
| LOW | No security headers | Clickjacking, MIME sniffing |
| LOW | No session cleanup for expired tokens | Resource exhaustion |

---

## Security Enhancements

### 1. WebSocket Origin Validation (CRITICAL)

**File**: `app/web/server.go`

**Current Issue**:
```go
upgrader: websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true }, // VULNERABLE
}
```

**Solution**: Implement proper origin validation

```go
func (s *Server) checkWebSocketOrigin(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    if origin == "" {
        return true // Allow non-browser clients
    }

    allowedOrigins := []string{
        "https://mizu.dev",
        "https://app.mizu.dev",
    }

    // In dev mode, allow localhost
    if s.cfg.Dev {
        allowedOrigins = append(allowedOrigins,
            "http://localhost:8080",
            "http://127.0.0.1:8080",
        )
    }

    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}
```

### 2. Rate Limiting Middleware

**New File**: `app/web/middleware/ratelimit.go`

Implement token bucket rate limiting for:
- **Login**: 5 attempts per IP per minute
- **Register**: 3 attempts per IP per 10 minutes
- **Password Reset**: 3 attempts per IP per hour
- **API Endpoints**: 100 requests per minute per user

```go
type RateLimiter struct {
    store    sync.Map // IP -> *rateLimitEntry
    limit    int
    window   time.Duration
    cleanup  time.Duration
}

type rateLimitEntry struct {
    count     int
    expiresAt time.Time
    mu        sync.Mutex
}

func (rl *RateLimiter) Allow(key string) bool {
    entry, _ := rl.store.LoadOrStore(key, &rateLimitEntry{
        expiresAt: time.Now().Add(rl.window),
    })
    e := entry.(*rateLimitEntry)
    e.mu.Lock()
    defer e.mu.Unlock()

    if time.Now().After(e.expiresAt) {
        e.count = 0
        e.expiresAt = time.Now().Add(rl.window)
    }

    e.count++
    return e.count <= rl.limit
}
```

### 3. Password Strength Validation

**File**: `pkg/password/password.go`

Implement comprehensive password validation:

```go
type PasswordPolicy struct {
    MinLength        int  // 8 minimum
    MaxLength        int  // 128 maximum (bcrypt limit is 72)
    RequireUppercase bool
    RequireLowercase bool
    RequireDigit     bool
    RequireSpecial   bool
}

var DefaultPolicy = PasswordPolicy{
    MinLength:        8,
    MaxLength:        128,
    RequireUppercase: false, // Optional for UX
    RequireLowercase: false,
    RequireDigit:     false,
    RequireSpecial:   false,
}

func Validate(password string) error {
    if len(password) < DefaultPolicy.MinLength {
        return fmt.Errorf("password must be at least %d characters", DefaultPolicy.MinLength)
    }
    if len(password) > DefaultPolicy.MaxLength {
        return fmt.Errorf("password must not exceed %d characters", DefaultPolicy.MaxLength)
    }

    // Check for common passwords
    if isCommonPassword(password) {
        return errors.New("password is too common")
    }

    return nil
}
```

### 4. Input Sanitization & XSS Protection

**New File**: `pkg/sanitize/sanitize.go`

```go
package sanitize

import (
    "html"
    "strings"
    "regexp"
)

var (
    scriptTagPattern = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
    eventHandlers    = regexp.MustCompile(`(?i)\s*on\w+\s*=`)
)

// Text sanitizes user text input for safe storage and display.
func Text(input string) string {
    // Remove null bytes
    input = strings.ReplaceAll(input, "\x00", "")

    // HTML escape
    input = html.EscapeString(input)

    return input
}

// Username validates and sanitizes usernames.
func Username(input string) (string, error) {
    input = strings.TrimSpace(input)

    if len(input) < 3 || len(input) > 32 {
        return "", errors.New("username must be 3-32 characters")
    }

    // Allow only alphanumeric, underscore, hyphen
    if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(input) {
        return "", errors.New("username can only contain letters, numbers, underscore, and hyphen")
    }

    return strings.ToLower(input), nil
}
```

Apply sanitization in handlers:
```go
// In message handler
in.Content = sanitize.Text(in.Content)
```

### 5. Session Security Enhancements

**File**: `feature/accounts/service.go`

Improvements:
1. Bind sessions to User-Agent and IP (optional, configurable)
2. Session rotation on privilege changes
3. Limit concurrent sessions per user
4. Automatic cleanup of expired sessions

```go
type SessionConfig struct {
    MaxConcurrent    int           // Max sessions per user (default: 5)
    BindToIP         bool          // Bind session to IP (default: false)
    BindToUserAgent  bool          // Bind to User-Agent (default: true)
    RotateOnAction   bool          // Rotate token on sensitive actions
    CleanupInterval  time.Duration // How often to cleanup expired sessions
}

func (s *Service) CreateSession(ctx context.Context, userID string, meta *SessionMeta) (*Session, error) {
    // Enforce max concurrent sessions
    sessions, _ := s.store.GetUserSessions(ctx, userID)
    if len(sessions) >= s.config.MaxConcurrent {
        // Delete oldest session
        oldest := sessions[len(sessions)-1]
        s.store.DeleteSession(ctx, oldest.Token)
    }

    // Create new session with metadata
    return s.store.CreateSession(ctx, userID, meta)
}
```

### 6. CSRF Protection

**New File**: `app/web/middleware/csrf.go`

For state-changing operations via forms (not needed for pure API with Bearer tokens):

```go
package middleware

import (
    "crypto/rand"
    "encoding/base64"
    "net/http"
    "sync"
    "time"
)

type CSRFStore struct {
    tokens sync.Map // token -> expiry
}

func (s *CSRFStore) Generate() string {
    b := make([]byte, 32)
    rand.Read(b)
    token := base64.URLEncoding.EncodeToString(b)
    s.tokens.Store(token, time.Now().Add(time.Hour))
    return token
}

func (s *CSRFStore) Validate(token string) bool {
    if exp, ok := s.tokens.Load(token); ok {
        if time.Now().Before(exp.(time.Time)) {
            s.tokens.Delete(token)
            return true
        }
        s.tokens.Delete(token)
    }
    return false
}
```

### 7. Message Content Validation

**File**: `feature/messages/service.go`

```go
const (
    MaxMessageLength = 4096  // Characters
    MaxMediaSize     = 10 << 20 // 10MB
)

func (s *Service) validateMessage(in *CreateIn) error {
    // Content length
    if len(in.Content) > MaxMessageLength {
        return ErrMessageTooLong
    }

    // Prevent empty messages
    if strings.TrimSpace(in.Content) == "" && in.MediaURL == "" {
        return ErrEmptyMessage
    }

    // Validate message type
    switch in.Type {
    case TypeText, TypeImage, TypeVideo, TypeAudio, TypeDocument, TypeSticker:
        // Valid
    default:
        return ErrInvalidMessageType
    }

    return nil
}
```

### 8. Security Headers Middleware

**New File**: `app/web/middleware/security.go`

```go
package middleware

import "github.com/go-mizu/mizu"

func SecurityHeaders() mizu.Middleware {
    return func(next mizu.Handler) mizu.Handler {
        return func(c *mizu.Ctx) error {
            h := c.Writer().Header()

            // Prevent clickjacking
            h.Set("X-Frame-Options", "DENY")

            // Prevent MIME sniffing
            h.Set("X-Content-Type-Options", "nosniff")

            // XSS protection (legacy browsers)
            h.Set("X-XSS-Protection", "1; mode=block")

            // Referrer policy
            h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

            // Content Security Policy
            h.Set("Content-Security-Policy",
                "default-src 'self'; "+
                "script-src 'self' 'unsafe-inline'; "+
                "style-src 'self' 'unsafe-inline'; "+
                "img-src 'self' data: blob:; "+
                "connect-src 'self' wss:")

            // Permissions Policy
            h.Set("Permissions-Policy",
                "geolocation=(), microphone=(), camera=()")

            return next(c)
        }
    }
}
```

### 9. Authorization Improvements

Ensure all endpoints verify user authorization:

```go
// Chat access verification
func (s *Service) VerifyAccess(ctx context.Context, userID, chatID string) error {
    chat, err := s.store.GetByID(ctx, chatID)
    if err != nil {
        return ErrNotFound
    }

    // Check if user is participant
    for _, p := range chat.Participants {
        if p.UserID == userID {
            return nil
        }
    }

    return ErrForbidden
}
```

### 10. Audit Logging

**New File**: `pkg/audit/audit.go`

Log security-sensitive operations:

```go
package audit

import (
    "context"
    "log"
    "time"
)

type Event struct {
    Timestamp time.Time `json:"timestamp"`
    UserID    string    `json:"user_id"`
    Action    string    `json:"action"`
    Resource  string    `json:"resource"`
    IP        string    `json:"ip"`
    UserAgent string    `json:"user_agent"`
    Success   bool      `json:"success"`
    Details   string    `json:"details,omitempty"`
}

var (
    ActionLogin         = "auth.login"
    ActionLogout        = "auth.logout"
    ActionRegister      = "auth.register"
    ActionPasswordChange = "auth.password_change"
    ActionSessionRevoke = "auth.session_revoke"
    ActionMessageDelete = "message.delete"
)

func Log(ctx context.Context, event Event) {
    // In production, send to structured logging service
    log.Printf("[AUDIT] %s user=%s action=%s resource=%s success=%v",
        event.Timestamp.Format(time.RFC3339),
        event.UserID,
        event.Action,
        event.Resource,
        event.Success,
    )
}
```

---

## E2E Security Tests

### Test Categories

1. **Authentication Security Tests**
   - Rate limiting prevents brute force
   - Session tokens are cryptographically secure
   - Sessions expire correctly
   - Invalid credentials don't reveal user existence

2. **Authorization Security Tests**
   - Users cannot access other users' chats
   - Users cannot edit other users' messages
   - Users cannot delete other users' data
   - Chat participant verification

3. **Input Validation Tests**
   - XSS prevention in message content
   - SQL injection prevention
   - Path traversal prevention
   - Max length enforcement

4. **WebSocket Security Tests**
   - Origin validation blocks unauthorized origins
   - Token validation on connection
   - Session expiry disconnects client
   - Message routing respects authorization

5. **CSRF Protection Tests**
   - State-changing requests require valid tokens
   - Tokens are single-use
   - Expired tokens are rejected

6. **Session Security Tests**
   - Logout invalidates session
   - Concurrent session limits work
   - Session binding (IP/UA) works

---

## Implementation Order

### Phase 1: Critical Fixes
1. WebSocket origin validation
2. Rate limiting for auth endpoints
3. Password strength validation

### Phase 2: Core Security
4. Input sanitization
5. Security headers
6. Session enhancements

### Phase 3: Advanced Security
7. CSRF protection
8. Audit logging
9. Message validation

### Phase 4: Testing
10. Comprehensive e2e security tests

---

## Files to Create/Modify

### New Files
- `app/web/middleware/ratelimit.go` - Rate limiting
- `app/web/middleware/security.go` - Security headers
- `app/web/middleware/csrf.go` - CSRF protection
- `pkg/sanitize/sanitize.go` - Input sanitization
- `pkg/audit/audit.go` - Audit logging
- `app/web/server_security_e2e_test.go` - Security e2e tests

### Modified Files
- `app/web/server.go` - WebSocket origin validation, middleware integration
- `app/web/handler/auth.go` - Rate limiting integration, audit logging
- `pkg/password/password.go` - Password validation
- `feature/messages/service.go` - Message validation
- `feature/accounts/service.go` - Session enhancements

---

## Common Password List

Include a list of top 1000 common passwords for validation:
- `123456`, `password`, `12345678`, `qwerty`, etc.

Store as embedded file or generate at build time.

---

## Security Headers for Different Environments

### Development
```
Content-Security-Policy: default-src * 'unsafe-inline' 'unsafe-eval'
```

### Production
```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self' wss:
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

---

## Best Practices from Modern Messaging Apps

### Signal
- End-to-end encryption (future consideration)
- Sealed sender (hide sender metadata)
- Disappearing messages

### Telegram
- Two-factor authentication
- Secret chats with self-destruct
- Session management with device list

### WhatsApp
- End-to-end encryption
- Security notifications for key changes
- Biometric authentication for app access

### Slack
- Enterprise audit logs
- DLP (Data Loss Prevention)
- SSO/SAML integration

---

## Future Considerations

1. **End-to-End Encryption**: Implement Signal Protocol
2. **Two-Factor Authentication**: TOTP/WebAuthn
3. **Key Transparency**: Public key verification
4. **Message Retention Policies**: Auto-delete old messages
5. **Device Management**: View and revoke sessions
6. **Report/Block System**: User safety features
