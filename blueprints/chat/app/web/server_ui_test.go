package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestUI_AllPages verifies that all pages render without template errors.
// This catches issues like missing struct fields referenced in templates.
func TestUI_AllPages(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a user and get auth token
	token := registerAndGetToken(t, srv.app, "uitestuser")

	// Create a server for testing server-specific pages
	serverBody := map[string]interface{}{
		"name":        "UI Test Server",
		"description": "Server for UI testing",
		"is_public":   true,
	}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, token)
	if serverRec.Code != http.StatusOK && serverRec.Code != http.StatusCreated {
		t.Fatalf("create server failed: %s", serverRec.Body.String())
	}

	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverData := serverResp["data"].(map[string]interface{})
	serverID := serverData["id"].(string)

	// Create a channel
	channelBody := map[string]interface{}{
		"name":  "ui-test-channel",
		"type":  "text",
		"topic": "Channel for UI testing",
	}
	channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, token)
	if channelRec.Code != http.StatusOK && channelRec.Code != http.StatusCreated {
		t.Fatalf("create channel failed: %s", channelRec.Body.String())
	}

	var channelResp map[string]interface{}
	parseResponse(t, channelRec, &channelResp)
	channelData := channelResp["data"].(map[string]interface{})
	channelID := channelData["id"].(string)

	// Create some messages to test message rendering
	for i := 0; i < 3; i++ {
		msgBody := map[string]interface{}{
			"content": "Test message " + string(rune('A'+i)),
		}
		doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelID+"/messages", msgBody, token)
	}

	// Test cases for all pages
	tests := []struct {
		name          string
		path          string
		authenticated bool
		wantStatus    int
		wantContains  []string // Check that response contains these strings
	}{
		// Public pages (no auth required)
		{
			name:         "landing page (unauthenticated)",
			path:         "/",
			wantStatus:   http.StatusOK,
			wantContains: []string{"<!DOCTYPE html>", "</html>"},
		},
		{
			name:         "login page",
			path:         "/login",
			wantStatus:   http.StatusOK,
			wantContains: []string{"<!DOCTYPE html>", "</html>", "Login"},
		},
		{
			name:         "register page",
			path:         "/register",
			wantStatus:   http.StatusOK,
			wantContains: []string{"<!DOCTYPE html>", "</html>", "Register"},
		},
		{
			name:         "explore page (unauthenticated)",
			path:         "/explore",
			wantStatus:   http.StatusOK,
			wantContains: []string{"<!DOCTYPE html>", "</html>"},
		},

		// Authenticated pages
		{
			name:          "home page (authenticated)",
			path:          "/",
			authenticated: true,
			wantStatus:    http.StatusOK,
			wantContains:  []string{"<!DOCTYPE html>", "</html>"},
		},
		{
			name:          "explore page (authenticated)",
			path:          "/explore",
			authenticated: true,
			wantStatus:    http.StatusOK,
			wantContains:  []string{"<!DOCTYPE html>", "</html>"},
		},
		{
			name:          "settings page",
			path:          "/settings",
			authenticated: true,
			wantStatus:    http.StatusOK,
			wantContains:  []string{"<!DOCTYPE html>", "</html>", "Settings"},
		},
		{
			name:          "server view with channel",
			path:          "/channels/" + serverID + "/" + channelID,
			authenticated: true,
			wantStatus:    http.StatusOK,
			wantContains:  []string{"<!DOCTYPE html>", "</html>", "UI Test Server"},
		},

		// Edge cases
		{
			name:          "settings redirects when unauthenticated",
			path:          "/settings",
			authenticated: false,
			wantStatus:    http.StatusFound,
		},
		{
			name:          "server view redirects when unauthenticated",
			path:          "/channels/" + serverID + "/" + channelID,
			authenticated: false,
			wantStatus:    http.StatusFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec *httptest.ResponseRecorder
			if tt.authenticated {
				rec = doHTMLRequest(t, srv.app, "GET", tt.path, token)
			} else {
				rec = doHTMLRequest(t, srv.app, "GET", tt.path, "")
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d\nbody: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			body := rec.Body.String()

			// Check for template errors - these patterns indicate template execution failed
			checkTemplateError(t, body)

			// Check expected content is present
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("response should contain %q\nbody length: %d", want, len(body))
				}
			}
		})
	}
}

// TestUI_ServerViewWithData tests the server view page with various data scenarios.
func TestUI_ServerViewWithData(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create two users
	aliceToken := registerAndGetToken(t, srv.app, "uialice")
	bobToken := registerAndGetToken(t, srv.app, "uibob")

	// Alice creates a server
	serverBody := map[string]interface{}{
		"name":        "Data Test Server",
		"description": "Testing with real data",
		"is_public":   true,
	}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, aliceToken)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	// Alice creates multiple channels
	channelIDs := make([]string, 0)
	for _, name := range []string{"general", "random", "help"} {
		channelBody := map[string]interface{}{
			"name":  name,
			"type":  "text",
			"topic": "Topic for " + name,
		}
		channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, aliceToken)
		var channelResp map[string]interface{}
		parseResponse(t, channelRec, &channelResp)
		channelIDs = append(channelIDs, channelResp["data"].(map[string]interface{})["id"].(string))
	}

	// Bob joins the server
	doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/join", nil, bobToken)

	// Both users send messages
	msgBody := map[string]interface{}{"content": "Hello from Alice!"}
	doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelIDs[0]+"/messages", msgBody, aliceToken)

	msgBody = map[string]interface{}{"content": "Hello from Bob!"}
	doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelIDs[0]+"/messages", msgBody, bobToken)

	// Test rendering the page with all this data
	t.Run("server view with multiple members and messages", func(t *testing.T) {
		rec := doHTMLRequest(t, srv.app, "GET", "/channels/"+serverID+"/"+channelIDs[0], aliceToken)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		body := rec.Body.String()

		// Check for template errors
		checkTemplateError(t, body)

		// Verify expected content is rendered
		expectations := []string{
			"Data Test Server",
			"general",
			"Hello from Alice",
			"Hello from Bob",
		}

		for _, exp := range expectations {
			if !strings.Contains(body, exp) {
				t.Errorf("expected %q in response", exp)
			}
		}
	})

	// Test each channel renders correctly
	for i, chID := range channelIDs {
		t.Run("channel "+string(rune('0'+i)), func(t *testing.T) {
			rec := doHTMLRequest(t, srv.app, "GET", "/channels/"+serverID+"/"+chID, aliceToken)

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			checkTemplateError(t, rec.Body.String())
		})
	}
}

// TestUI_EmptyStates tests pages with empty/minimal data.
func TestUI_EmptyStates(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a user with no servers
	token := registerAndGetToken(t, srv.app, "emptyuser")

	t.Run("home page with no servers", func(t *testing.T) {
		rec := doHTMLRequest(t, srv.app, "GET", "/", token)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		checkTemplateError(t, rec.Body.String())
	})

	// Create a server with no channels (besides default)
	serverBody := map[string]interface{}{
		"name": "Empty Server",
	}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, token)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	// Get default channel
	channelsRec := doRequest(t, srv.app, "GET", "/api/v1/servers/"+serverID+"/channels", nil, token)
	var channelsResp map[string]interface{}
	parseResponse(t, channelsRec, &channelsResp)
	channels := channelsResp["data"].([]interface{})

	if len(channels) > 0 {
		channelID := channels[0].(map[string]interface{})["id"].(string)

		t.Run("channel with no messages", func(t *testing.T) {
			rec := doHTMLRequest(t, srv.app, "GET", "/channels/"+serverID+"/"+channelID, token)

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			checkTemplateError(t, rec.Body.String())
		})
	}
}

// TestUI_TemplateErrorDetection specifically tests that template errors are caught.
func TestUI_TemplateErrorDetection(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	token := registerAndGetToken(t, srv.app, "templateuser")

	// Create server and channel
	serverBody := map[string]interface{}{"name": "Template Test"}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, token)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	channelBody := map[string]interface{}{"name": "test", "type": "text"}
	channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, token)
	var channelResp map[string]interface{}
	parseResponse(t, channelRec, &channelResp)
	channelID := channelResp["data"].(map[string]interface{})["id"].(string)

	// Render the page and check for any template-related errors
	rec := doHTMLRequest(t, srv.app, "GET", "/channels/"+serverID+"/"+channelID, token)

	// The page should render successfully
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Check for template errors
	checkTemplateError(t, rec.Body.String())

	// Additionally check for complete HTML
	body := rec.Body.String()
	if !strings.Contains(body, "</html>") {
		t.Errorf("incomplete HTML response, template failed mid-execution")
	}
}

// TestUI_ComponentsWithData tests that all UI components render correctly with real data.
// This catches template field mismatches like accessing .UnreadCount on Channel.
func TestUI_ComponentsWithData(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create users
	aliceToken := registerAndGetToken(t, srv.app, "compAlice")
	bobToken := registerAndGetToken(t, srv.app, "compBob")

	// Alice creates a server
	serverBody := map[string]interface{}{
		"name":        "Component Test Server",
		"description": "Testing all components",
		"is_public":   true,
	}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, aliceToken)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	// Create multiple channels (tests channel_list.html with multiple items)
	var channelIDs []string
	for _, ch := range []struct{ name, topic string }{
		{"general", "General discussion"},
		{"announcements", "Important updates"},
		{"random", "Off-topic chat"},
	} {
		channelBody := map[string]interface{}{
			"name":  ch.name,
			"type":  "text",
			"topic": ch.topic,
		}
		channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, aliceToken)
		var channelResp map[string]interface{}
		parseResponse(t, channelRec, &channelResp)
		channelIDs = append(channelIDs, channelResp["data"].(map[string]interface{})["id"].(string))
	}

	// Bob joins (tests member_list.html with multiple members)
	doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/join", nil, bobToken)

	// Both users send messages (tests message.html with author data)
	messages := []struct {
		token   string
		content string
	}{
		{aliceToken, "Hello from Alice!"},
		{bobToken, "Hi Alice, Bob here."},
		{aliceToken, "Welcome to the server!"},
	}
	for _, msg := range messages {
		msgBody := map[string]interface{}{"content": msg.content}
		doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelIDs[0]+"/messages", msgBody, msg.token)
	}

	// Test the full page render (exercises all components together)
	t.Run("full page with all components", func(t *testing.T) {
		rec := doHTMLRequest(t, srv.app, "GET", "/channels/"+serverID+"/"+channelIDs[0], aliceToken)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		body := rec.Body.String()
		checkTemplateError(t, body)

		// Verify key component content is rendered
		componentChecks := []struct {
			desc    string
			content string
		}{
			{"server name in channel list", "Component Test Server"},
			{"channel name", "general"},
			{"message content", "Hello from Alice"},
			{"member section", "member"},
		}
		for _, check := range componentChecks {
			if !strings.Contains(body, check.content) {
				t.Errorf("%s: expected %q in response", check.desc, check.content)
			}
		}
	})

	// Test each channel (ensures channel_list.html active state works)
	for i, chID := range channelIDs {
		t.Run("channel "+string(rune('a'+i)), func(t *testing.T) {
			rec := doHTMLRequest(t, srv.app, "GET", "/channels/"+serverID+"/"+chID, aliceToken)
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			checkTemplateError(t, rec.Body.String())
		})
	}

	// Test as different user (tests user_panel.html with different user data)
	t.Run("view as Bob", func(t *testing.T) {
		rec := doHTMLRequest(t, srv.app, "GET", "/channels/"+serverID+"/"+channelIDs[0], bobToken)
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		checkTemplateError(t, rec.Body.String())
	})
}

// doHTMLRequest performs an HTML page request with optional auth via cookie.
func doHTMLRequest(t *testing.T, handler http.Handler, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "text/html")

	if token != "" {
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: token,
		})
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// checkTemplateError checks if the response body indicates a template error.
// Template errors can manifest as:
// - Incomplete HTML (missing closing tags)
// - Error messages in the output
// - Empty responses when content is expected
func checkTemplateError(t *testing.T, body string) {
	t.Helper()

	// These patterns indicate template execution errors
	errorPatterns := []string{
		"can't evaluate field",
		"undefined function",
		"nil pointer",
		"wrong type for value",
		"executing \"",
		"at <.",
	}

	for _, pattern := range errorPatterns {
		if strings.Contains(body, pattern) {
			t.Errorf("template error pattern %q found in response:\n%s", pattern, truncateBody(body, 500))
		}
	}

	// Check for incomplete HTML - if we have DOCTYPE but no closing html tag
	if strings.Contains(body, "<!DOCTYPE html>") && !strings.Contains(body, "</html>") {
		t.Errorf("incomplete HTML detected (missing </html>), template likely failed mid-execution\nbody length: %d\nlast 200 chars: %s",
			len(body), truncateBody(body[max(0, len(body)-200):], 200))
	}
}

// truncateBody truncates the body for error messages
func truncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "...(truncated)"
}

// max returns the larger of two ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
