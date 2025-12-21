package handler_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-mizu/blueprints/microblog/app/web/handler"
	"github.com/go-mizu/blueprints/microblog/feature/accounts"
)

func TestAuth_Register(t *testing.T) {
	_, accountsSvc, _, _, cleanup := setupTestEnv(t)
	defer cleanup()

	h := handler.NewAuth(accountsSvc)

	registerBody := []byte(`{
		"username": "newuser",
		"email": "newuser@example.com",
		"password": "password123"
	}`)

	rec, ctx := testRequest("POST", "/api/v1/auth/register", registerBody, "")

	if err := h.Register(ctx); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in response")
	}

	account, ok := data["account"].(map[string]any)
	if !ok {
		t.Fatal("expected account field in data")
	}

	if account["username"] != "newuser" {
		t.Errorf("expected username newuser, got %v", account["username"])
	}

	token, ok := data["token"].(string)
	if !ok || token == "" {
		t.Error("expected token to be set")
	}
}

func TestAuth_RegisterInvalidJSON(t *testing.T) {
	_, accountsSvc, _, _, cleanup := setupTestEnv(t)
	defer cleanup()

	h := handler.NewAuth(accountsSvc)

	registerBody := []byte(`{invalid json}`)

	rec, ctx := testRequest("POST", "/api/v1/auth/register", registerBody, "")

	if err := h.Register(ctx); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if rec.Code != 400 {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestAuth_RegisterDuplicateUsername(t *testing.T) {
	_, accountsSvc, _, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create first account
	_, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "duplicateuser",
		Email:    "first@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create first account: %v", err)
	}

	h := handler.NewAuth(accountsSvc)

	// Try to register with same username
	registerBody := []byte(`{
		"username": "duplicateuser",
		"email": "second@example.com",
		"password": "password123"
	}`)

	rec, ctx := testRequest("POST", "/api/v1/auth/register", registerBody, "")

	if err := h.Register(ctx); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if rec.Code != 400 {
		t.Errorf("expected status 400 for duplicate username, got %d", rec.Code)
	}
}

func TestAuth_Login(t *testing.T) {
	_, accountsSvc, _, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create account first
	_, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "loginuser",
		Email:    "login@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	h := handler.NewAuth(accountsSvc)

	loginBody := []byte(`{
		"username": "loginuser",
		"password": "password123"
	}`)

	rec, ctx := testRequest("POST", "/api/v1/auth/login", loginBody, "")

	if err := h.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in response")
	}

	token, ok := data["token"].(string)
	if !ok || token == "" {
		t.Error("expected token to be set")
	}
}

func TestAuth_LoginInvalidCredentials(t *testing.T) {
	_, accountsSvc, _, _, cleanup := setupTestEnv(t)
	defer cleanup()

	h := handler.NewAuth(accountsSvc)

	loginBody := []byte(`{
		"username": "nonexistent",
		"password": "wrongpassword"
	}`)

	rec, ctx := testRequest("POST", "/api/v1/auth/login", loginBody, "")

	if err := h.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if rec.Code != 401 {
		t.Errorf("expected status 401, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errData, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error field in response")
	}

	if errData["code"] != "LOGIN_FAILED" {
		t.Errorf("expected error code LOGIN_FAILED, got %v", errData["code"])
	}
}

func TestAuth_LoginWrongPassword(t *testing.T) {
	_, accountsSvc, _, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create account
	_, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "wrongpassuser",
		Email:    "wrongpass@example.com",
		Password: "correctpassword",
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	h := handler.NewAuth(accountsSvc)

	loginBody := []byte(`{
		"username": "wrongpassuser",
		"password": "wrongpassword"
	}`)

	rec, ctx := testRequest("POST", "/api/v1/auth/login", loginBody, "")

	if err := h.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if rec.Code != 401 {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestAuth_Logout(t *testing.T) {
	_, accountsSvc, _, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create account and session
	account, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "logoutuser",
		Email:    "logout@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	session, err := accountsSvc.CreateSession(context.Background(), account.ID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	h := handler.NewAuth(accountsSvc)

	rec, ctx := testRequest("POST", "/api/v1/auth/logout", nil, "")
	ctx.Request().Header.Set("Authorization", "Bearer "+session.Token)

	if err := h.Logout(ctx); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in response")
	}

	if data["success"] != true {
		t.Error("expected success to be true")
	}
}

func TestAuth_LogoutNoToken(t *testing.T) {
	_, accountsSvc, _, _, cleanup := setupTestEnv(t)
	defer cleanup()

	h := handler.NewAuth(accountsSvc)

	rec, ctx := testRequest("POST", "/api/v1/auth/logout", nil, "")

	if err := h.Logout(ctx); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
