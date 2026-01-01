// File: lib/storage/transport/webdav/jwt_test.go

package webdav

import (
	"testing"
	"time"
)

func TestValidateJWT(t *testing.T) {
	secret := "test-secret-key"

	tests := []struct {
		name    string
		claims  map[string]any
		secret  string
		wantErr bool
	}{
		{
			name: "valid token",
			claims: map[string]any{
				"sub":  "user123",
				"name": "Test User",
				"exp":  float64(time.Now().Add(time.Hour).Unix()),
			},
			secret:  secret,
			wantErr: false,
		},
		{
			name: "expired token",
			claims: map[string]any{
				"sub":  "user123",
				"name": "Test User",
				"exp":  float64(time.Now().Add(-time.Hour).Unix()),
			},
			secret:  secret,
			wantErr: true,
		},
		{
			name: "wrong secret",
			claims: map[string]any{
				"sub":  "user123",
				"name": "Test User",
				"exp":  float64(time.Now().Add(time.Hour).Unix()),
			},
			secret:  "wrong-secret",
			wantErr: true,
		},
		{
			name: "no expiration",
			claims: map[string]any{
				"sub":  "user123",
				"name": "Test User",
			},
			secret:  secret,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := createToken(tt.claims, secret)
			if err != nil {
				t.Fatalf("createToken: %v", err)
			}

			claims, err := validateJWT(token, tt.secret)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("validateJWT: %v", err)
			}

			// Verify claims match
			if claims["sub"] != tt.claims["sub"] {
				t.Errorf("sub = %v, want %v", claims["sub"], tt.claims["sub"])
			}
			if claims["name"] != tt.claims["name"] {
				t.Errorf("name = %v, want %v", claims["name"], tt.claims["name"])
			}
		})
	}
}

func TestValidateJWT_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"one part", "abc"},
		{"two parts", "abc.def"},
		{"four parts", "abc.def.ghi.jkl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateJWT(tt.token, "secret")
			if err == nil {
				t.Error("expected error for invalid format")
			}
		})
	}
}

func TestCreateToken(t *testing.T) {
	secret := "test-secret"
	claims := map[string]any{
		"sub":  "user123",
		"role": "admin",
	}

	token, err := createToken(claims, secret)
	if err != nil {
		t.Fatalf("createToken: %v", err)
	}

	// Token should have 3 parts
	parts := 0
	for _, c := range token {
		if c == '.' {
			parts++
		}
	}
	if parts != 2 {
		t.Errorf("token has %d dots, want 2", parts)
	}

	// Verify token can be validated
	parsedClaims, err := validateJWT(token, secret)
	if err != nil {
		t.Fatalf("validateJWT: %v", err)
	}

	if parsedClaims["sub"] != "user123" {
		t.Errorf("sub = %v, want user123", parsedClaims["sub"])
	}
	if parsedClaims["role"] != "admin" {
		t.Errorf("role = %v, want admin", parsedClaims["role"])
	}
}

func TestBase64URLEncodeDecode(t *testing.T) {
	tests := []struct {
		input []byte
	}{
		{[]byte("hello")},
		{[]byte("hello world")},
		{[]byte("a")},
		{[]byte("ab")},
		{[]byte("abc")},
		{[]byte("abcd")},
		{[]byte{0, 1, 2, 3, 255, 254, 253}},
	}

	for _, tt := range tests {
		encoded := base64URLEncode(tt.input)
		decoded, err := base64URLDecode(encoded)
		if err != nil {
			t.Errorf("decode %q: %v", encoded, err)
			continue
		}
		if string(decoded) != string(tt.input) {
			t.Errorf("roundtrip %v: got %v", tt.input, decoded)
		}
	}
}
