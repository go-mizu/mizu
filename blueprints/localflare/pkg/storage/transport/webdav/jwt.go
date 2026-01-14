// File: lib/storage/transport/webdav/jwt.go

package webdav

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// validateJWT verifies and parses a JWT token, returning the claims.
func validateJWT(token, secret string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	headerB64, claimsB64, signatureB64 := parts[0], parts[1], parts[2]

	// Verify signature
	signingInput := headerB64 + "." + claimsB64
	expectedSignature := signHS256(signingInput, secret)
	expectedSignatureB64 := base64URLEncode(expectedSignature)

	if !hmac.Equal([]byte(signatureB64), []byte(expectedSignatureB64)) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode claims
	claimsJSON, err := base64URLDecode(claimsB64)
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	// Check expiration
	if exp, ok := claims["exp"].(float64); ok && exp > 0 {
		if time.Now().Unix() > int64(exp) {
			return nil, fmt.Errorf("token expired")
		}
	}

	return claims, nil
}

// signHS256 signs the input using HMAC-SHA256.
func signHS256(input, secret string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(input))
	return h.Sum(nil)
}

// base64URLEncode encodes bytes to base64url without padding.
func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// base64URLDecode decodes base64url (with or without padding).
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

// createToken creates a JWT token with the given claims and secret.
// This is primarily for testing purposes.
func createToken(claims map[string]any, secret string) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	headerB64 := base64URLEncode(headerJSON)
	claimsB64 := base64URLEncode(claimsJSON)

	signingInput := headerB64 + "." + claimsB64

	signature := signHS256(signingInput, secret)
	signatureB64 := base64URLEncode(signature)

	return signingInput + "." + signatureB64, nil
}
