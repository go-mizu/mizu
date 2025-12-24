package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// setupTestServer creates a test server with an in-memory database.
func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	// Create temp directory for test data
	tempDir, err := os.MkdirTemp("", "chat-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	cfg := Config{
		Addr:    ":0", // random port
		DataDir: tempDir,
		Dev:     true,
	}

	srv, err := New(cfg)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("create server: %v", err)
	}

	cleanup := func() {
		srv.Close()
		os.RemoveAll(tempDir)
	}

	return srv, cleanup
}

// doRequest performs an HTTP request and returns the response.
func doRequest(t *testing.T, handler http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// parseResponse parses the JSON response body.
func parseResponse(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, rec.Body.String())
	}
}

// TestE2E_Auth_Register tests user registration.
func TestE2E_Auth_Register(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := map[string]string{
		"username":     "testuser",
		"email":        "test@example.com",
		"password":     "password123",
		"display_name": "Test User",
	}

	rec := doRequest(t, srv.app, "POST", "/api/v1/auth/register", body, "")

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp map[string]interface{}
	parseResponse(t, rec, &resp)

	// Register returns {data: user} - extract the user from data
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("response should contain data object")
	}
	if data["username"] != "testuser" {
		t.Errorf("username = %v, want testuser", data["username"])
	}
}

// TestE2E_Auth_Register_Validation tests registration validation.
func TestE2E_Auth_Register_Validation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing username", map[string]string{"email": "test@example.com", "password": "pass123"}},
		{"missing email", map[string]string{"username": "test", "password": "pass123"}},
		{"missing password", map[string]string{"username": "test", "email": "test@example.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, srv.app, "POST", "/api/v1/auth/register", tt.body, "")
			if rec.Code == http.StatusOK {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

// TestE2E_Auth_Login tests user login.
func TestE2E_Auth_Login(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Register first
	registerBody := map[string]string{
		"username": "loginuser",
		"email":    "login@example.com",
		"password": "password123",
	}
	doRequest(t, srv.app, "POST", "/api/v1/auth/register", registerBody, "")

	// Login
	loginBody := map[string]string{
		"login":    "loginuser",
		"password": "password123",
	}
	rec := doRequest(t, srv.app, "POST", "/api/v1/auth/login", loginBody, "")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]interface{}
	parseResponse(t, rec, &resp)

	// Login returns {data: {token: ..., user: ...}}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("response should contain data object")
	}
	if data["token"] == nil {
		t.Error("response should contain token")
	}
}

// TestE2E_Auth_Login_InvalidCredentials tests login with wrong password.
func TestE2E_Auth_Login_InvalidCredentials(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Register
	registerBody := map[string]string{
		"username": "badpassuser",
		"email":    "badpass@example.com",
		"password": "correctpassword",
	}
	doRequest(t, srv.app, "POST", "/api/v1/auth/register", registerBody, "")

	// Login with wrong password
	loginBody := map[string]string{
		"login":    "badpassuser",
		"password": "wrongpassword",
	}
	rec := doRequest(t, srv.app, "POST", "/api/v1/auth/login", loginBody, "")

	if rec.Code == http.StatusOK {
		t.Error("login with wrong password should fail")
	}
}

// TestE2E_Auth_Me tests getting current user.
func TestE2E_Auth_Me(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Register and login to get token
	token := registerAndGetToken(t, srv.app, "meuser")

	// Get me
	rec := doRequest(t, srv.app, "GET", "/api/v1/auth/me", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]interface{}
	parseResponse(t, rec, &resp)

	// Response is {data: user}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("response should contain data object")
	}
	if data["username"] != "meuser" {
		t.Errorf("username = %v, want meuser", data["username"])
	}
}

// TestE2E_Auth_Logout tests logout.
func TestE2E_Auth_Logout(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Register and login to get token
	token := registerAndGetToken(t, srv.app, "logoutuser")

	// Logout
	rec := doRequest(t, srv.app, "POST", "/api/v1/auth/logout", nil, token)

	// Logout returns 204 No Content
	if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 204 or 200 (body: %s)", rec.Code, rec.Body.String())
	}

	// Verify token is invalidated
	meRec := doRequest(t, srv.app, "GET", "/api/v1/auth/me", nil, token)
	if meRec.Code == http.StatusOK {
		t.Error("token should be invalidated after logout")
	}
}

// TestE2E_Servers_Create tests server creation.
func TestE2E_Servers_Create(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Register user
	token := registerAndGetToken(t, srv.app, "serverowner")

	// Create server
	body := map[string]interface{}{
		"name":        "Test Server",
		"description": "A test server",
		"is_public":   true,
	}
	rec := doRequest(t, srv.app, "POST", "/api/v1/servers", body, token)

	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 200 or 201 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	parseResponse(t, rec, &resp)

	data := resp["data"].(map[string]interface{})
	if data["name"] != "Test Server" {
		t.Errorf("name = %v, want Test Server", data["name"])
	}
}

// TestE2E_Servers_List tests listing user's servers.
func TestE2E_Servers_List(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	token := registerAndGetToken(t, srv.app, "listuser")

	// Create servers
	for i := 0; i < 3; i++ {
		body := map[string]interface{}{
			"name": "Server-" + string(rune('A'+i)),
		}
		doRequest(t, srv.app, "POST", "/api/v1/servers", body, token)
	}

	// List servers
	rec := doRequest(t, srv.app, "GET", "/api/v1/servers", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]interface{}
	parseResponse(t, rec, &resp)

	data := resp["data"].([]interface{})
	if len(data) != 3 {
		t.Errorf("len(servers) = %d, want 3", len(data))
	}
}

// TestE2E_Servers_ListPublic tests listing public servers.
func TestE2E_Servers_ListPublic(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	token := registerAndGetToken(t, srv.app, "publicuser")

	// Create public server
	body := map[string]interface{}{
		"name":      "Public Server",
		"is_public": true,
	}
	doRequest(t, srv.app, "POST", "/api/v1/servers", body, token)

	// Create private server
	body = map[string]interface{}{
		"name":      "Private Server",
		"is_public": false,
	}
	doRequest(t, srv.app, "POST", "/api/v1/servers", body, token)

	// List public servers (no auth required)
	rec := doRequest(t, srv.app, "GET", "/api/v1/servers/public", nil, "")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]interface{}
	parseResponse(t, rec, &resp)

	servers := resp["data"].([]interface{})
	// Should only have public servers
	for _, s := range servers {
		server := s.(map[string]interface{})
		if !server["is_public"].(bool) {
			t.Error("ListPublic returned private server")
		}
	}
}

// TestE2E_Channels_Create tests channel creation.
func TestE2E_Channels_Create(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	token := registerAndGetToken(t, srv.app, "channelowner")

	// Create server first
	serverBody := map[string]interface{}{"name": "Channel Test Server"}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, token)

	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverData := serverResp["data"].(map[string]interface{})
	serverID := serverData["id"].(string)

	// Create channel
	body := map[string]interface{}{
		"name":  "general",
		"type":  "text",
		"topic": "General discussion",
	}
	rec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", body, token)

	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 200 or 201 (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestE2E_Messages_Create tests message creation.
func TestE2E_Messages_Create(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	token := registerAndGetToken(t, srv.app, "msguser")

	// Create server and channel
	serverBody := map[string]interface{}{"name": "Message Test Server"}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, token)

	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	channelBody := map[string]interface{}{"name": "test-channel", "type": "text"}
	channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, token)

	var channelResp map[string]interface{}
	parseResponse(t, channelRec, &channelResp)
	channelID := channelResp["data"].(map[string]interface{})["id"].(string)

	// Create message
	body := map[string]interface{}{
		"content": "Hello, world!",
	}
	rec := doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelID+"/messages", body, token)

	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 200 or 201 (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestE2E_Messages_List tests message listing with pagination.
func TestE2E_Messages_List(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	token := registerAndGetToken(t, srv.app, "listmsguser")

	// Create server and channel
	serverBody := map[string]interface{}{"name": "List Messages Server"}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, token)

	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	channelBody := map[string]interface{}{"name": "messages-channel", "type": "text"}
	channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, token)

	var channelResp map[string]interface{}
	parseResponse(t, channelRec, &channelResp)
	channelID := channelResp["data"].(map[string]interface{})["id"].(string)

	// Create multiple messages
	for i := 0; i < 5; i++ {
		body := map[string]interface{}{
			"content": "Message " + string(rune('A'+i)),
		}
		doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelID+"/messages", body, token)
	}

	// List messages
	rec := doRequest(t, srv.app, "GET", "/api/v1/channels/"+channelID+"/messages?limit=10", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]interface{}
	parseResponse(t, rec, &resp)

	messages := resp["data"].([]interface{})
	if len(messages) != 5 {
		t.Errorf("len(messages) = %d, want 5", len(messages))
	}
}

// TestE2E_Users_Search tests user search.
func TestE2E_Users_Search(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Register users
	registerAndGetToken(t, srv.app, "searchalice")
	registerAndGetToken(t, srv.app, "searchbob")
	token := registerAndGetToken(t, srv.app, "searchcharlie")

	// Search for 'alice'
	rec := doRequest(t, srv.app, "GET", "/api/v1/users/search?q=alice", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]interface{}
	parseResponse(t, rec, &resp)

	users := resp["data"].([]interface{})
	if len(users) != 1 {
		t.Errorf("len(users) = %d, want 1", len(users))
	}
}

// TestE2E_FullFlow_CreateServerAndChat tests a complete flow.
func TestE2E_FullFlow_CreateServerAndChat(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Register two users
	aliceToken := registerAndGetToken(t, srv.app, "flowalice")
	bobToken := registerAndGetToken(t, srv.app, "flowbob")

	// 2. Alice creates a server
	serverBody := map[string]interface{}{
		"name":        "Flow Test Server",
		"description": "Testing the complete flow",
		"is_public":   true,
	}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, aliceToken)
	if serverRec.Code != http.StatusOK && serverRec.Code != http.StatusCreated {
		t.Fatalf("create server failed: %s", serverRec.Body.String())
	}

	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	// 3. Alice creates a channel
	channelBody := map[string]interface{}{
		"name":  "flow-general",
		"type":  "text",
		"topic": "General chat",
	}
	channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, aliceToken)
	if channelRec.Code != http.StatusOK && channelRec.Code != http.StatusCreated {
		t.Fatalf("create channel failed: %s", channelRec.Body.String())
	}

	var channelResp map[string]interface{}
	parseResponse(t, channelRec, &channelResp)
	channelID := channelResp["data"].(map[string]interface{})["id"].(string)

	// 4. Bob joins the server
	joinRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/join", nil, bobToken)
	if joinRec.Code != http.StatusOK && joinRec.Code != http.StatusCreated {
		t.Fatalf("join server failed: %s", joinRec.Body.String())
	}

	// 5. Alice sends a message
	aliceMsgBody := map[string]interface{}{"content": "Hello Bob!"}
	doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelID+"/messages", aliceMsgBody, aliceToken)

	// 6. Bob sends a message
	bobMsgBody := map[string]interface{}{"content": "Hey Alice!"}
	doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelID+"/messages", bobMsgBody, bobToken)

	// 7. Verify messages
	msgRec := doRequest(t, srv.app, "GET", "/api/v1/channels/"+channelID+"/messages?limit=10", nil, aliceToken)
	if msgRec.Code != http.StatusOK {
		t.Fatalf("list messages failed: %s", msgRec.Body.String())
	}

	var msgResp map[string]interface{}
	parseResponse(t, msgRec, &msgResp)

	messages := msgResp["data"].([]interface{})
	if len(messages) != 2 {
		t.Errorf("len(messages) = %d, want 2", len(messages))
	}

	// 8. Verify server members
	membersRec := doRequest(t, srv.app, "GET", "/api/v1/servers/"+serverID+"/members", nil, aliceToken)
	if membersRec.Code != http.StatusOK {
		t.Fatalf("list members failed: %s", membersRec.Body.String())
	}

	var membersResp map[string]interface{}
	parseResponse(t, membersRec, &membersResp)

	members := membersResp["data"].([]interface{})
	if len(members) != 2 {
		t.Errorf("len(members) = %d, want 2", len(members))
	}
}

// Helper function to register a user and return their token.
func registerAndGetToken(t *testing.T, handler http.Handler, username string) string {
	t.Helper()

	// Register user
	registerBody := map[string]string{
		"username":     username,
		"email":        username + "@example.com",
		"password":     "password123",
		"display_name": username,
	}

	regRec := doRequest(t, handler, "POST", "/api/v1/auth/register", registerBody, "")
	if regRec.Code != http.StatusCreated {
		t.Fatalf("register user %s failed: %s", username, regRec.Body.String())
	}

	// Login to get token
	loginBody := map[string]string{
		"login":    username,
		"password": "password123",
	}
	loginRec := doRequest(t, handler, "POST", "/api/v1/auth/login", loginBody, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login user %s failed: %s", username, loginRec.Body.String())
	}

	var resp map[string]interface{}
	parseResponse(t, loginRec, &resp)

	// Extract token from data.token
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("login response missing data: %v", resp)
	}
	token, ok := data["token"].(string)
	if !ok {
		t.Fatalf("login response missing token: %v", data)
	}
	return token
}
