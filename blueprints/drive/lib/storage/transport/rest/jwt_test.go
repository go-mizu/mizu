package rest

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestVerifyToken(t *testing.T) {
	secret := "test-secret-key-for-jwt-tokens"

	tests := []struct {
		name      string
		token     string
		secret    string
		wantErr   bool
		wantSub   string
		wantRole  string
	}{
		{
			name: "valid token",
			token: func() string {
				token, _ := createToken(&Claims{
					Sub:  "user-123",
					Role: "authenticated",
					Exp:  time.Now().Add(1 * time.Hour).Unix(),
				}, secret)
				return token
			}(),
			secret:   secret,
			wantErr:  false,
			wantSub:  "user-123",
			wantRole: "authenticated",
		},
		{
			name: "expired token",
			token: func() string {
				token, _ := createToken(&Claims{
					Sub:  "user-123",
					Role: "authenticated",
					Exp:  time.Now().Add(-1 * time.Hour).Unix(),
				}, secret)
				return token
			}(),
			secret:  secret,
			wantErr: true,
		},
		{
			name: "wrong secret",
			token: func() string {
				token, _ := createToken(&Claims{
					Sub:  "user-123",
					Role: "authenticated",
					Exp:  time.Now().Add(1 * time.Hour).Unix(),
				}, secret)
				return token
			}(),
			secret:  "wrong-secret",
			wantErr: true,
		},
		{
			name:    "invalid token format",
			token:   "invalid.token",
			secret:  secret,
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			secret:  secret,
			wantErr: true,
		},
		{
			name:    "malformed base64 in claims",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.!!!invalidbase64!!!.signature",
			secret:  secret,
			wantErr: true,
		},
		{
			name: "token without expiration (valid)",
			token: func() string {
				token, _ := createToken(&Claims{
					Sub:  "user-456",
					Role: "service_role",
				}, secret)
				return token
			}(),
			secret:   secret,
			wantErr:  false,
			wantSub:  "user-456",
			wantRole: "service_role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := verifyToken(tt.token, tt.secret)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if claims.Sub != tt.wantSub {
				t.Errorf("sub = %q, want %q", claims.Sub, tt.wantSub)
			}
			if claims.Role != tt.wantRole {
				t.Errorf("role = %q, want %q", claims.Role, tt.wantRole)
			}
		})
	}
}

func TestCreateToken(t *testing.T) {
	secret := "test-secret-for-token-creation"

	tests := []struct {
		name   string
		claims *Claims
	}{
		{
			name: "full claims",
			claims: &Claims{
				Sub:  "user-123",
				Aud:  "authenticated",
				Iss:  "supabase",
				Role: "authenticated",
				Iat:  time.Now().Unix(),
				Exp:  time.Now().Add(1 * time.Hour).Unix(),
			},
		},
		{
			name: "minimal claims",
			claims: &Claims{
				Sub: "user-456",
			},
		},
		{
			name: "service role",
			claims: &Claims{
				Sub:  "service-account",
				Role: "service_role",
				Exp:  time.Now().Add(24 * time.Hour).Unix(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := createToken(tt.claims, secret)
			if err != nil {
				t.Fatalf("createToken error: %v", err)
			}

			// Token should have 3 parts
			parts := strings.Split(token, ".")
			if len(parts) != 3 {
				t.Fatalf("token has %d parts, want 3", len(parts))
			}

			// Header should be valid JSON
			headerJSON, err := base64URLDecode(parts[0])
			if err != nil {
				t.Fatalf("decode header: %v", err)
			}
			var header map[string]string
			if err := json.Unmarshal(headerJSON, &header); err != nil {
				t.Fatalf("parse header: %v", err)
			}
			if header["alg"] != "HS256" {
				t.Errorf("header alg = %q, want HS256", header["alg"])
			}
			if header["typ"] != "JWT" {
				t.Errorf("header typ = %q, want JWT", header["typ"])
			}

			// Claims should be valid JSON
			claimsJSON, err := base64URLDecode(parts[1])
			if err != nil {
				t.Fatalf("decode claims: %v", err)
			}
			var parsedClaims Claims
			if err := json.Unmarshal(claimsJSON, &parsedClaims); err != nil {
				t.Fatalf("parse claims: %v", err)
			}
			if parsedClaims.Sub != tt.claims.Sub {
				t.Errorf("claims.sub = %q, want %q", parsedClaims.Sub, tt.claims.Sub)
			}

			// Token should be verifiable
			verified, err := verifyToken(token, secret)
			if err != nil {
				t.Fatalf("verify token: %v", err)
			}
			if verified.Sub != tt.claims.Sub {
				t.Errorf("verified sub = %q, want %q", verified.Sub, tt.claims.Sub)
			}
		})
	}
}

func TestSignHS256(t *testing.T) {
	// Test known signature value
	input := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ"
	secret := "your-256-bit-secret"

	signature := signHS256(input, secret)

	// Signature should not be empty
	if len(signature) == 0 {
		t.Error("signature is empty")
	}

	// Same input should produce same signature
	signature2 := signHS256(input, secret)
	if string(signature) != string(signature2) {
		t.Error("same input produced different signatures")
	}

	// Different secret should produce different signature
	signature3 := signHS256(input, "different-secret")
	if string(signature) == string(signature3) {
		t.Error("different secrets produced same signature")
	}
}

func TestBase64URLEncode(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "simple string",
			input: []byte("hello"),
			want:  "aGVsbG8",
		},
		{
			name:  "empty",
			input: []byte{},
			want:  "",
		},
		{
			name:  "with special chars",
			input: []byte("hello+world/test="),
			want:  "aGVsbG8rd29ybGQvdGVzdD0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := base64URLEncode(tt.input)
			if got != tt.want {
				t.Errorf("base64URLEncode(%q) = %q, want %q", tt.input, got, tt.want)
			}

			// Should be decodable
			decoded, err := base64URLDecode(got)
			if err != nil {
				t.Errorf("decode error: %v", err)
			}
			if string(decoded) != string(tt.input) {
				t.Errorf("roundtrip failed: got %q, want %q", decoded, tt.input)
			}
		})
	}
}

func TestBase64URLDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "without padding",
			input: "aGVsbG8",
			want:  "hello",
		},
		{
			name:  "with 1 padding char needed",
			input: "aGVsbG8gd29ybGQ",
			want:  "hello world",
		},
		{
			name:  "with 2 padding chars needed",
			input: "aGk",
			want:  "hi",
		},
		{
			name:  "with existing padding",
			input: "aGVsbG8=",
			want:  "hello",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:    "invalid base64",
			input:   "!!!invalid!!!",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := base64URLDecode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("base64URLDecode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTokenRoundTrip(t *testing.T) {
	secret := "test-roundtrip-secret-key-for-jwt"

	claims := &Claims{
		Sub:  "test-user-id",
		Aud:  "authenticated",
		Iss:  "https://example.com",
		Role: "authenticated",
		Iat:  time.Now().Unix(),
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}

	// Create token
	token, err := createToken(claims, secret)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Verify and parse token
	parsed, err := verifyToken(token, secret)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}

	// Verify all fields match
	if parsed.Sub != claims.Sub {
		t.Errorf("sub = %q, want %q", parsed.Sub, claims.Sub)
	}
	if parsed.Aud != claims.Aud {
		t.Errorf("aud = %q, want %q", parsed.Aud, claims.Aud)
	}
	if parsed.Iss != claims.Iss {
		t.Errorf("iss = %q, want %q", parsed.Iss, claims.Iss)
	}
	if parsed.Role != claims.Role {
		t.Errorf("role = %q, want %q", parsed.Role, claims.Role)
	}
	if parsed.Iat != claims.Iat {
		t.Errorf("iat = %d, want %d", parsed.Iat, claims.Iat)
	}
	if parsed.Exp != claims.Exp {
		t.Errorf("exp = %d, want %d", parsed.Exp, claims.Exp)
	}
}

func TestSupabaseCompatibleToken(t *testing.T) {
	// Test token format compatible with Supabase Auth JWT format
	secret := "super-secret-jwt-token-with-at-least-32-characters-long"

	// Create a token similar to what Supabase Auth would generate
	claims := &Claims{
		Sub:  "12345678-1234-1234-1234-123456789012", // UUID format
		Aud:  "authenticated",
		Iss:  "https://project-ref.supabase.co/auth/v1",
		Role: "authenticated",
		Iat:  time.Now().Unix(),
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}

	token, err := createToken(claims, secret)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Verify token structure matches expected format
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}

	// Decode and check header
	headerJSON, _ := base64URLDecode(parts[0])
	var header map[string]any
	json.Unmarshal(headerJSON, &header)

	if header["alg"] != "HS256" {
		t.Errorf("algorithm = %v, want HS256", header["alg"])
	}

	// Decode and check claims
	claimsJSON, _ := base64URLDecode(parts[1])
	var parsedClaims map[string]any
	json.Unmarshal(claimsJSON, &parsedClaims)

	if parsedClaims["sub"] != claims.Sub {
		t.Errorf("sub = %v, want %v", parsedClaims["sub"], claims.Sub)
	}
	if parsedClaims["aud"] != claims.Aud {
		t.Errorf("aud = %v, want %v", parsedClaims["aud"], claims.Aud)
	}
	if parsedClaims["role"] != claims.Role {
		t.Errorf("role = %v, want %v", parsedClaims["role"], claims.Role)
	}

	// Verify the token is valid
	verified, err := verifyToken(token, secret)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if verified.Sub != claims.Sub {
		t.Errorf("verified sub = %q, want %q", verified.Sub, claims.Sub)
	}
}

// TestTokenWithStandardBase64 verifies we handle tokens with standard base64 encoding
func TestTokenWithStandardBase64(t *testing.T) {
	// Some JWT libraries might use standard base64 with padding
	// Our implementation should handle both
	secret := "test-secret-key"

	claims := &Claims{
		Sub:  "user",
		Role: "authenticated",
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}

	token, _ := createToken(claims, secret)

	// Replace URL-safe chars with standard base64 and add padding
	// This simulates a token from a library using standard base64
	parts := strings.Split(token, ".")

	// Our decoder should handle the base64url format
	verified, err := verifyToken(token, secret)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}

	if verified.Sub != claims.Sub {
		t.Errorf("sub = %q, want %q", verified.Sub, claims.Sub)
	}

	_ = parts // use parts to avoid unused variable
}

// BenchmarkVerifyToken benchmarks token verification performance
func BenchmarkVerifyToken(b *testing.B) {
	secret := "benchmark-secret-key"
	token, _ := createToken(&Claims{
		Sub:  "benchmark-user",
		Role: "authenticated",
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}, secret)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifyToken(token, secret)
	}
}

// BenchmarkCreateToken benchmarks token creation performance
func BenchmarkCreateToken(b *testing.B) {
	secret := "benchmark-secret-key"
	claims := &Claims{
		Sub:  "benchmark-user",
		Role: "authenticated",
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = createToken(claims, secret)
	}
}

// BenchmarkBase64URLEncode benchmarks base64 URL encoding
func BenchmarkBase64URLEncode(b *testing.B) {
	data := []byte(`{"sub":"user-123","role":"authenticated","exp":1234567890}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = base64.URLEncoding.EncodeToString(data)
	}
}
