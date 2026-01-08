package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestGenerateSecret_Strength verifies that generated secrets are cryptographically strong.
func TestGenerateSecret_Strength(t *testing.T) {
	// Generate multiple secrets to ensure randomness
	secrets := make(map[string]bool)
	for i := 0; i < 100; i++ {
		secret := GenerateSecret()

		// Verify minimum length (32 bytes base64 encoded = ~43 chars)
		if len(secret) < 32 {
			t.Errorf("Secret too short: %d chars, want at least 32", len(secret))
		}

		// Verify uniqueness
		if secrets[secret] {
			t.Errorf("Duplicate secret generated: %s", secret)
		}
		secrets[secret] = true
	}
}

// TestGenerateSecret_NotTimeBased verifies the secret is not predictable from time.
func TestGenerateSecret_NotTimeBased(t *testing.T) {
	// Generate two secrets at nearly the same time
	secret1 := GenerateSecret()
	secret2 := GenerateSecret()

	// They should be different
	if secret1 == secret2 {
		t.Error("Secrets generated at same time should be different")
	}

	// Verify they don't contain predictable patterns
	if strings.Contains(secret1, "spreadsheet-secret-") {
		t.Error("Secret should not contain predictable prefix")
	}
}

// TestVerify_RejectsExpiredToken verifies expired tokens are rejected.
func TestVerify_RejectsExpiredToken(t *testing.T) {
	secret := GenerateSecret()

	// Create token with past expiration
	claims := Claims{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(-time.Hour).Unix(),
		IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
	}

	token := createTestToken(t, &claims, secret)

	// Verify should reject
	_, err := Verify(token, secret)
	if err != ErrExpiredToken {
		t.Errorf("Expected ErrExpiredToken, got: %v", err)
	}
}

// TestVerify_RejectsFutureToken verifies tokens with future NotBefore are rejected.
func TestVerify_RejectsFutureToken(t *testing.T) {
	secret := GenerateSecret()

	// Create token with future NotBefore by manually constructing
	// (Token() sets NotBefore to now, so we need custom construction)
	claims := Claims{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
		NotBefore: time.Now().Add(30 * time.Minute).Unix(),
	}

	token := createCustomToken(t, &claims, secret)

	// Verify should reject
	_, err := Verify(token, secret)
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken for future NotBefore, got: %v", err)
	}
}

// createCustomToken creates a token with custom claims for testing.
func createCustomToken(t *testing.T, claims *Claims, secret string) string {
	t.Helper()

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	message := headerB64 + "." + claimsB64

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return message + "." + signature
}

// TestVerify_RejectsTamperedToken verifies modified tokens are rejected.
func TestVerify_RejectsTamperedToken(t *testing.T) {
	secret := GenerateSecret()

	// Create valid token
	claims := Claims{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	token := createTestToken(t, &claims, secret)

	// Tamper with the payload (change user ID)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatal("Invalid token format")
	}

	// Modify the middle part (payload)
	tamperedToken := parts[0] + ".dGVzdA." + parts[2]

	// Verify should reject
	_, err := Verify(tamperedToken, secret)
	if err == nil {
		t.Error("Expected error for tampered token, got nil")
	}
}

// TestVerify_RejectsWrongSecret verifies tokens with wrong secret are rejected.
func TestVerify_RejectsWrongSecret(t *testing.T) {
	correctSecret := GenerateSecret()
	wrongSecret := GenerateSecret()

	// Create token with correct secret
	claims := Claims{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	token := createTestToken(t, &claims, correctSecret)

	// Verify with wrong secret should fail
	_, err := Verify(token, wrongSecret)
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got: %v", err)
	}
}

// TestVerify_RejectsMalformedToken verifies malformed tokens are rejected.
func TestVerify_RejectsMalformedToken(t *testing.T) {
	secret := GenerateSecret()

	testCases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no_dots", "notokenhere"},
		{"one_dot", "part1.part2"},
		{"too_many_dots", "part1.part2.part3.part4"},
		{"invalid_base64", "!!!.@@@.###"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Verify(tc.token, secret)
			if err == nil {
				t.Error("Expected error for malformed token, got nil")
			}
		})
	}
}

// TestUserID_RequiresSecret verifies UserID requires secret for verification.
func TestUserID_RequiresSecret(t *testing.T) {
	secret := GenerateSecret()
	wrongSecret := GenerateSecret()

	// Create valid token
	claims := Claims{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	token := createTestToken(t, &claims, secret)

	// UserID should fail with wrong secret
	_, err := UserID(token, wrongSecret)
	if err == nil {
		t.Error("UserID should fail with wrong secret")
	}

	// UserID should succeed with correct secret
	userID, err := UserID(token, secret)
	if err != nil {
		t.Errorf("UserID failed with correct secret: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("UserID = %s, want user-1", userID)
	}
}

// TestToken_ValidToken verifies Token creates valid tokens.
func TestToken_ValidToken(t *testing.T) {
	secret := GenerateSecret()

	token, err := Token(secret, "user-123", time.Hour)
	if err != nil {
		t.Fatalf("Token failed: %v", err)
	}

	// Verify the token
	claims, err := Verify(token, secret)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %s, want user-123", claims.UserID)
	}

	// Check expiration is roughly correct
	expectedExpiry := time.Now().Add(time.Hour).Unix()
	if claims.ExpiresAt < expectedExpiry-10 || claims.ExpiresAt > expectedExpiry+10 {
		t.Errorf("ExpiresAt = %d, want ~%d", claims.ExpiresAt, expectedExpiry)
	}
}

// createTestToken creates a token for testing purposes.
func createTestToken(t *testing.T, claims *Claims, secret string) string {
	t.Helper()

	token, err := Token(secret, claims.UserID, time.Until(time.Unix(claims.ExpiresAt, 0)))
	if err != nil {
		t.Fatalf("Failed to create test token: %v", err)
	}
	return token
}
