package api_test

import (
	"net/http"
	"testing"
)

func TestAuthWorkflow(t *testing.T) {
	ts := newTestServer(t)

	status, _ := ts.doJSON(http.MethodGet, "/auth/me", nil, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", status)
	}

	token, user := registerUser(t, ts, "user@example.com")
	userID := requireString(t, user, "id")

	status, _ = ts.doJSON(http.MethodPost, "/auth/register", map[string]any{
		"email":    "user@example.com",
		"name":     "Dup User",
		"password": "secret123!",
	}, "")
	if status != http.StatusBadRequest {
		t.Fatalf("expected status 400 for duplicate register, got %d", status)
	}

	status, _ = ts.doJSON(http.MethodPost, "/auth/login", map[string]any{
		"email":    "user@example.com",
		"password": "wrong",
	}, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", status)
	}

	status, data := ts.doJSON(http.MethodPost, "/auth/login", map[string]any{
		"email":    "user@example.com",
		"password": "secret123!",
	}, "")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	loginToken := requireString(t, data, "token")

	status, data = ts.doJSON(http.MethodGet, "/auth/me", nil, loginToken)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	me := requireMap(t, data, "user")
	if requireString(t, me, "id") != userID {
		t.Fatalf("unexpected user id in /auth/me")
	}

	status, _ = ts.doJSON(http.MethodPost, "/auth/logout", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
}
