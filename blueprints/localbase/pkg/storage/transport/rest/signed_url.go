package rest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// SignedURLPayload contains the claims for a signed URL token.
type SignedURLPayload struct {
	Bucket  string `json:"bucket"`
	Path    string `json:"path"`
	Method  string `json:"method"` // GET or PUT
	Expires int64  `json:"exp"`    // Unix timestamp
}

// generateSignedURLToken creates a signed token for accessing an object.
func generateSignedURLToken(secret, bucket, path, method string, expires time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("signing secret is required")
	}

	payload := SignedURLPayload{
		Bucket:  bucket,
		Path:    path,
		Method:  method,
		Expires: time.Now().Add(expires).Unix(),
	}

	// Encode payload as JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Base64 encode the payload
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// Create HMAC signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	// Return token in format: payload.signature
	return encodedPayload + "." + signature, nil
}

// validateSignedURLToken validates a signed URL token and returns the payload.
func validateSignedURLToken(secret, token string) (*SignedURLPayload, error) {
	if secret == "" {
		return nil, errors.New("signing secret is required")
	}

	// Split token into payload and signature
	parts := splitToken(token)
	if len(parts) != 2 {
		return nil, errors.New("invalid token format")
	}

	encodedPayload := parts[0]
	providedSignature := parts[1]

	// Verify signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(encodedPayload))
	expectedSignature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(providedSignature), []byte(expectedSignature)) {
		return nil, errors.New("invalid signature")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload SignedURLPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Check expiration
	if time.Now().Unix() > payload.Expires {
		return nil, errors.New("token expired")
	}

	return &payload, nil
}

// splitToken splits a token at the last period.
func splitToken(token string) []string {
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			return []string{token[:i], token[i+1:]}
		}
	}
	return []string{token}
}

// buildSignedURL creates a full signed URL for object access.
func buildSignedURL(baseURL, bucket, path, token string) string {
	// Build URL: baseURL/object/render/bucket/path?token=...
	u, err := url.Parse(baseURL)
	if err != nil {
		// Fallback to simple path construction
		return fmt.Sprintf("%s/object/render/%s/%s?token=%s",
			baseURL, url.PathEscape(bucket), path, url.QueryEscape(token))
	}

	u.Path = fmt.Sprintf("%s/object/render/%s/%s", u.Path, bucket, path)
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()

	return u.String()
}

// buildUploadSignedURL creates a full signed URL for object upload.
func buildUploadSignedURL(baseURL, bucket, path, token string) string {
	// Build URL: baseURL/object/upload/bucket/path?token=...
	u, err := url.Parse(baseURL)
	if err != nil {
		// Fallback to simple path construction
		return fmt.Sprintf("%s/object/%s/%s?token=%s",
			baseURL, url.PathEscape(bucket), path, url.QueryEscape(token))
	}

	u.Path = fmt.Sprintf("%s/object/%s/%s", u.Path, bucket, path)
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()

	return u.String()
}

// getRequestBaseURL extracts the base URL from a request.
// This is used to construct signed URLs that point back to this server.
func getRequestBaseURL(host, scheme, basePath string) string {
	if scheme == "" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, basePath)
}

// parseExpiresIn parses the expiresIn parameter from query string.
func parseExpiresIn(value string, defaultVal int) int {
	if value == "" {
		return defaultVal
	}
	if i, err := strconv.Atoi(value); err == nil && i > 0 {
		return i
	}
	return defaultVal
}
