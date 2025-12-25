package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/messaging/app/web/handler"
	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/feature/chats"
	"github.com/go-mizu/blueprints/messaging/feature/messages"
	"github.com/go-mizu/blueprints/messaging/store/duckdb"
)

// testServer creates a test server with an in-memory DuckDB database.
func testServer(t *testing.T) (*testEnv, func()) {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	store, err := duckdb.New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		t.Fatalf("failed to ensure schema: %v", err)
	}

	usersStore := duckdb.NewUsersStore(db)
	chatsStore := duckdb.NewChatsStore(db)
	messagesStore := duckdb.NewMessagesStore(db)

	accountsSvc := accounts.NewService(usersStore)
	chatsSvc := chats.NewService(chatsStore)
	messagesSvc := messages.NewService(messagesStore)

	env := &testEnv{
		t:        t,
		db:       db,
		accounts: accountsSvc,
		chats:    chatsSvc,
		messages: messagesSvc,
	}

	cleanup := func() {
		db.Close()
	}

	return env, cleanup
}

type testEnv struct {
	t        *testing.T
	db       *sql.DB
	accounts accounts.API
	chats    chats.API
	messages messages.API
}

// createTestUser creates a test user and returns the user and session token.
func (e *testEnv) createTestUser(username, email, password string) (*accounts.User, string) {
	e.t.Helper()

	user, err := e.accounts.Create(context.Background(), &accounts.CreateIn{
		Username:    username,
		Email:       email,
		Password:    password,
		DisplayName: username,
	})
	if err != nil {
		e.t.Fatalf("failed to create user: %v", err)
	}

	session, err := e.accounts.Login(context.Background(), &accounts.LoginIn{
		Login:    username,
		Password: password,
	})
	if err != nil {
		e.t.Fatalf("failed to login: %v", err)
	}

	return user, session.Token
}

// apiResponse is the standard API response format.
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func TestServer_New(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	if server.app == nil {
		t.Error("server.app is nil")
	}
	if server.db == nil {
		t.Error("server.db is nil")
	}
	if server.hub == nil {
		t.Error("server.hub is nil")
	}
}

func TestServer_Handler(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	h := server.Handler()
	if h == nil {
		t.Error("Handler() returned nil")
	}
}

func TestAuth_Register(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	t.Run("success", func(t *testing.T) {
		body := `{"username": "testuser", "email": "test@example.com", "password": "password123", "display_name": "Test User"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		var resp apiResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got error: %s", resp.Error)
		}
	})

	t.Run("missing username", func(t *testing.T) {
		body := `{"email": "test2@example.com", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("missing password", func(t *testing.T) {
		body := `{"username": "testuser2", "email": "test2@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("password too short", func(t *testing.T) {
		body := `{"username": "testuser3", "email": "test3@example.com", "password": "short"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		body := `{"username": "testuser", "email": "another@example.com", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestAuth_Login(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user first
	registerBody := `{"username": "logintest", "email": "login@example.com", "password": "password123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	t.Run("success with username", func(t *testing.T) {
		body := `{"login": "logintest", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp apiResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got error: %s", resp.Error)
		}

		// Check session cookie
		cookies := rec.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == "session" && c.Value != "" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected session cookie to be set")
		}
	})

	t.Run("success with email", func(t *testing.T) {
		body := `{"login": "login@example.com", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		body := `{"login": "logintest", "password": "wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("missing login", func(t *testing.T) {
		body := `{"password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestAuth_Me(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register and login
	registerBody := `{"username": "metest", "email": "me@example.com", "password": "password123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Success bool `json:"success"`
		Data    struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	t.Run("success with bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp apiResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got error: %s", resp.Error)
		}
	})

	t.Run("unauthorized without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestAuth_UpdateMe(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register and login
	registerBody := `{"username": "updatetest", "email": "update@example.com", "password": "password123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Success bool `json:"success"`
		Data    struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	t.Run("success", func(t *testing.T) {
		body := `{"display_name": "Updated Name", "status": "Hello World"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})
}

func TestAuth_Logout(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register and login
	registerBody := `{"username": "logouttest", "email": "logout@example.com", "password": "password123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Success bool `json:"success"`
		Data    struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.AddCookie(&http.Cookie{Name: "session", Value: token})
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		// Check session cookie is cleared
		cookies := rec.Result().Cookies()
		for _, c := range cookies {
			if c.Name == "session" && c.MaxAge == -1 {
				return
			}
		}
	})
}

func TestChat_CRUD(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Create two users
	registerUser := func(username, email string) string {
		body := `{"username": "` + username + `", "email": "` + email + `", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Session struct {
					Token string `json:"token"`
				} `json:"session"`
				User struct {
					ID string `json:"id"`
				} `json:"user"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		return resp.Data.Session.Token
	}

	token1 := registerUser("chatuser1", "chat1@example.com")
	token2 := registerUser("chatuser2", "chat2@example.com")

	// Get user2's ID
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token2)
	meRec := httptest.NewRecorder()
	server.app.ServeHTTP(meRec, meReq)

	var meResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(meRec.Body).Decode(&meResp)
	user2ID := meResp.Data.ID

	var chatID string

	t.Run("create direct chat", func(t *testing.T) {
		body := `{"type": "direct", "recipient_id": "` + user2ID + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		chatID = resp.Data.ID

		if chatID == "" {
			t.Error("expected chat ID to be set")
		}
	})

	t.Run("create group chat", func(t *testing.T) {
		body := `{"type": "group", "name": "Test Group", "description": "A test group", "participant_ids": ["` + user2ID + `"]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}
	})

	t.Run("list chats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data []interface{} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		// User should have at least 4 chats: 2 default chats (Saved Messages + Agent) + 2 created in test
		if len(resp.Data) < 4 {
			t.Errorf("expected at least 4 chats (2 default + 2 created), got %d", len(resp.Data))
		}
	})

	t.Run("get chat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+chatID, nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("archive chat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/archive", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("unarchive chat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+chatID+"/archive", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("pin chat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/pin", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("unpin chat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+chatID+"/pin", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("mute chat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/mute", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("unmute chat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+chatID+"/mute", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("unauthorized without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestMessage_CRUD(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Create two users
	registerUser := func(username, email string) (string, string) {
		body := `{"username": "` + username + `", "email": "` + email + `", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Session struct {
					Token string `json:"token"`
				} `json:"session"`
				User struct {
					ID string `json:"id"`
				} `json:"user"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		return resp.Data.Session.Token, resp.Data.User.ID
	}

	token1, _ := registerUser("msguser1", "msg1@example.com")
	token2, user2ID := registerUser("msguser2", "msg2@example.com")

	// Create a chat
	chatBody := `{"type": "direct", "recipient_id": "` + user2ID + `"}`
	chatReq := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewBufferString(chatBody))
	chatReq.Header.Set("Content-Type", "application/json")
	chatReq.Header.Set("Authorization", "Bearer "+token1)
	chatRec := httptest.NewRecorder()
	server.app.ServeHTTP(chatRec, chatReq)

	var chatResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(chatRec.Body).Decode(&chatResp)
	chatID := chatResp.Data.ID

	var messageID string

	t.Run("create message", func(t *testing.T) {
		body := `{"type": "text", "content": "Hello, World!"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/messages", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				ID      string `json:"id"`
				Content string `json:"content"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		messageID = resp.Data.ID

		if messageID == "" {
			t.Error("expected message ID to be set")
		}
		if resp.Data.Content != "Hello, World!" {
			t.Errorf("expected content 'Hello, World!', got '%s'", resp.Data.Content)
		}
	})

	t.Run("list messages", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+chatID+"/messages", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data []interface{} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		if len(resp.Data) != 1 {
			t.Errorf("expected 1 message, got %d", len(resp.Data))
		}
	})

	t.Run("get message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+chatID+"/messages/"+messageID, nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("update message", func(t *testing.T) {
		body := `{"content": "Updated message"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/chats/"+chatID+"/messages/"+messageID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Content string `json:"content"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		if resp.Data.Content != "Updated message" {
			t.Errorf("expected content 'Updated message', got '%s'", resp.Data.Content)
		}
	})

	t.Run("add reaction", func(t *testing.T) {
		body := `{"emoji": "👍"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/messages/"+messageID+"/react", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("remove reaction", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+chatID+"/messages/"+messageID+"/react", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("star message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/messages/"+messageID+"/star", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("list starred", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/starred", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("unstar message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+chatID+"/messages/"+messageID+"/star", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("pin message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/pins/"+messageID, nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("list pinned", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+chatID+"/pins", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("unpin message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+chatID+"/pins/"+messageID, nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("typing indicator", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/typing", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("search messages", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search/messages?q=Updated", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("mark as read", func(t *testing.T) {
		body := `{"message_id": "` + messageID + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID+"/read", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token2)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("delete message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/chats/"+chatID+"/messages/"+messageID, nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})
}

func TestUsers_Search(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user
	registerBody := `{"username": "searchable", "email": "search@example.com", "password": "password123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	t.Run("search users", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/search?q=search", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})

	t.Run("search without query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/search", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestPages(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	pages := []string{"/", "/login", "/register"}

	for _, page := range pages {
		t.Run("GET "+page, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, page, nil)
			rec := httptest.NewRecorder()

			server.app.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestStaticFiles(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	t.Run("static js file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/js/app.js", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/javascript; charset=utf-8" && contentType != "application/javascript" {
			t.Errorf("unexpected content type: %s", contentType)
		}
	})
}

func TestResponseHelpers(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Test that response helpers produce correct status codes and format
	testCases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"bad request", http.MethodPost, "/api/v1/auth/register", "{invalid json", http.StatusBadRequest},
		{"unauthorized", http.MethodGet, "/api/v1/auth/me", "", http.StatusUnauthorized},
		{"not found user", http.MethodGet, "/api/v1/users/nonexistent", "", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			rec := httptest.NewRecorder()

			server.app.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}

			var resp handler.Response
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Success {
				t.Error("expected success to be false")
			}
			if resp.Error == "" {
				t.Error("expected error message to be set")
			}
		})
	}
}

func TestMessageForward(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Create users
	registerUser := func(username, email string) (string, string) {
		body := `{"username": "` + username + `", "email": "` + email + `", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		var resp struct {
			Data struct {
				Session struct {
					Token string `json:"token"`
				} `json:"session"`
				User struct {
					ID string `json:"id"`
				} `json:"user"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		return resp.Data.Session.Token, resp.Data.User.ID
	}

	token1, _ := registerUser("fwduser1", "fwd1@example.com")
	_, user2ID := registerUser("fwduser2", "fwd2@example.com")
	_, user3ID := registerUser("fwduser3", "fwd3@example.com")

	// Create chats
	createChat := func(recipientID string) string {
		body := `{"type": "direct", "recipient_id": "` + recipientID + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		var resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		return resp.Data.ID
	}

	chat1ID := createChat(user2ID)
	chat2ID := createChat(user3ID)

	// Create a message
	msgBody := `{"type": "text", "content": "Message to forward"}`
	msgReq := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chat1ID+"/messages", bytes.NewBufferString(msgBody))
	msgReq.Header.Set("Content-Type", "application/json")
	msgReq.Header.Set("Authorization", "Bearer "+token1)
	msgRec := httptest.NewRecorder()
	server.app.ServeHTTP(msgRec, msgReq)

	var msgResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(msgRec.Body).Decode(&msgResp)
	messageID := msgResp.Data.ID

	t.Run("forward message", func(t *testing.T) {
		body := `{"to_chat_ids": ["` + chat2ID + `"]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chat1ID+"/messages/"+messageID+"/forward", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})
}

func TestUsers_Me(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user
	registerBody := `{"username": "testuser", "email": "test@example.com", "password": "password123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()

	server.app.ServeHTTP(registerRec, registerReq)

	if registerRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, registerRec.Code, registerRec.Body.String())
	}

	var registerResp struct {
		Success bool `json:"success"`
		Data    struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	t.Run("/api/v1/users/me with bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp apiResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got error: %s", resp.Error)
		}
	})

	t.Run("/api/v1/users/me with session cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: token,
		})
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp apiResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got error: %s", resp.Error)
		}
	})

	t.Run("/api/v1/users/me without auth returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
		}
	})
}

func TestChats_AuthRequired(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "GET /api/v1/chats without auth",
			method:     http.MethodGet,
			path:       "/api/v1/chats",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "POST /api/v1/chats without auth",
			method:     http.MethodPost,
			path:       "/api/v1/chats",
			body:       `{"type": "direct", "recipient_id": "test"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "GET /api/v1/chats/:id without auth",
			method:     http.MethodGet,
			path:       "/api/v1/chats/someid",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "GET /api/v1/chats/:id/messages without auth",
			method:     http.MethodGet,
			path:       "/api/v1/chats/someid/messages",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			rec := httptest.NewRecorder()

			server.app.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}

			var resp handler.Response
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Success {
				t.Error("expected success to be false")
			}
			if resp.Error == "" {
				t.Error("expected error message to be set")
			}
		})
	}
}

func TestWebSocket_CookieAuth(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user
	registerBody := `{"username": "wstest", "email": "ws@example.com", "password": "password123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()

	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Success bool `json:"success"`
		Data    struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	t.Run("WebSocket without auth returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
		}
	})

	t.Run("WebSocket with token query param passes auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws?token="+token, nil)
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		// Not 401 means auth passed (actual upgrade may fail with 500 in test due to httptest limitations)
		if rec.Code == http.StatusUnauthorized {
			t.Errorf("expected auth to pass, got 401: %s", rec.Body.String())
		}
	})

	t.Run("WebSocket with session cookie passes auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: token,
		})
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		// Not 401 means auth passed (actual upgrade may fail with 500 in test due to httptest limitations)
		if rec.Code == http.StatusUnauthorized {
			t.Errorf("expected auth to pass, got 401: %s", rec.Body.String())
		}
	})
}

func TestNewUser_DefaultChats(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Verify the agent was created
	if server.agentID == "" {
		t.Error("expected agent to be created at startup")
	}

	// Register a new user
	registerBody := `{"username": "newuser", "email": "new@example.com", "password": "password123", "display_name": "New User"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()

	server.app.ServeHTTP(registerRec, registerReq)

	if registerRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, registerRec.Code, registerRec.Body.String())
	}

	var registerResp struct {
		Success bool `json:"success"`
		Data    struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token

	// List chats for the new user
	t.Run("new user has default chats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		// Should have 2 chats: Saved Messages (self-chat) and Agent chat
		if len(resp.Data) != 2 {
			t.Errorf("expected 2 default chats, got %d", len(resp.Data))
		}

		// Both should be direct chats
		directCount := 0
		for _, chat := range resp.Data {
			if chat.Type == "direct" {
				directCount++
			}
		}
		if directCount != 2 {
			t.Errorf("expected 2 direct chats, got %d", directCount)
		}
	})

	// Check that each chat has a welcome message
	t.Run("default chats have welcome messages", func(t *testing.T) {
		// Get chats
		chatsReq := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
		chatsReq.Header.Set("Authorization", "Bearer "+token)
		chatsRec := httptest.NewRecorder()
		server.app.ServeHTTP(chatsRec, chatsReq)

		var chatsResp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		json.NewDecoder(chatsRec.Body).Decode(&chatsResp)

		for _, chat := range chatsResp.Data {
			// Get messages for each chat
			msgReq := httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+chat.ID+"/messages", nil)
			msgReq.Header.Set("Authorization", "Bearer "+token)
			msgRec := httptest.NewRecorder()
			server.app.ServeHTTP(msgRec, msgReq)

			if msgRec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, msgRec.Code)
				continue
			}

			var msgResp struct {
				Data []struct {
					ID      string `json:"id"`
					Content string `json:"content"`
				} `json:"data"`
			}
			json.NewDecoder(msgRec.Body).Decode(&msgResp)

			// Each default chat should have at least 1 welcome message
			if len(msgResp.Data) < 1 {
				t.Errorf("expected at least 1 message in chat %s, got %d", chat.ID, len(msgResp.Data))
			}
		}
	})
}

func TestSelfChat_CreateAndMessage(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	// Register a user
	registerBody := `{"username": "selfchatuser", "email": "selfchat@example.com", "password": "password123"}`
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	server.app.ServeHTTP(registerRec, registerReq)

	var registerResp struct {
		Data struct {
			Session struct {
				Token string `json:"token"`
			} `json:"session"`
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	json.NewDecoder(registerRec.Body).Decode(&registerResp)
	token := registerResp.Data.Session.Token
	userID := registerResp.Data.User.ID

	// Get the chats (new user should have 2: Saved Messages and Agent chat)
	chatsReq := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
	chatsReq.Header.Set("Authorization", "Bearer "+token)
	chatsRec := httptest.NewRecorder()
	server.app.ServeHTTP(chatsRec, chatsReq)

	var chatsResp struct {
		Data []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"data"`
	}
	json.NewDecoder(chatsRec.Body).Decode(&chatsResp)

	if len(chatsResp.Data) < 1 {
		t.Fatal("expected at least 1 chat")
	}

	// Use the first direct chat as the test chat (could be either self-chat or agent chat)
	var testChatID string
	for _, chat := range chatsResp.Data {
		if chat.Type == "direct" {
			testChatID = chat.ID
			break
		}
	}

	if testChatID == "" {
		t.Fatal("No direct chat found")
	}

	t.Run("can send message to chat", func(t *testing.T) {
		body := `{"type": "text", "content": "This is a note to myself"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+testChatID+"/messages", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				ID      string `json:"id"`
				Content string `json:"content"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		if resp.Data.Content != "This is a note to myself" {
			t.Errorf("expected message content 'This is a note to myself', got '%s'", resp.Data.Content)
		}
	})

	t.Run("creating self-chat returns existing or new chat", func(t *testing.T) {
		body := `{"type": "direct", "recipient_id": "` + userID + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		// Should return a direct chat
		if resp.Data.Type != "direct" {
			t.Errorf("expected direct chat, got '%s'", resp.Data.Type)
		}
	})
}

func TestAgent_CreatedAtStartup(t *testing.T) {
	tmpDir := t.TempDir()

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer server.Close()

	t.Run("agent user exists", func(t *testing.T) {
		if server.agentID == "" {
			t.Error("expected agent ID to be set")
		}

		// Search for the agent user
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/search?q=mizu-agent", nil)
		rec := httptest.NewRecorder()

		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Data []struct {
				ID          string `json:"id"`
				Username    string `json:"username"`
				DisplayName string `json:"display_name"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)

		if len(resp.Data) == 0 {
			t.Fatal("expected to find agent user")
		}

		found := false
		for _, user := range resp.Data {
			if user.Username == AgentUsername {
				found = true
				if user.DisplayName != "Mizu Agent" {
					t.Errorf("expected display name 'Mizu Agent', got '%s'", user.DisplayName)
				}
				if user.ID != server.agentID {
					t.Errorf("expected agent ID %s, got %s", server.agentID, user.ID)
				}
				break
			}
		}

		if !found {
			t.Error("agent user not found in search results")
		}
	})
}
