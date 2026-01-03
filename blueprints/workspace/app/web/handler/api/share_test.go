package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/sharing"
)

func TestShareCreate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	owner, ownerCookie := ts.Register("shareowner@example.com", "Share Owner", "password123")
	targetUser, _ := ts.Register("sharetarget@example.com", "Share Target", "password123")
	_ = owner // avoid unused warning

	ws := createTestWorkspace(ts, ownerCookie, "Share Workspace", "share-ws")
	page := createTestPage(ts, ownerCookie, ws.ID, "Share Test Page")

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "share with user - read",
			body: map[string]interface{}{
				"type":       "user",
				"user_id":    targetUser.ID,
				"permission": "read",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "create share link",
			body: map[string]interface{}{
				"type":       "link",
				"permission": "read",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "enable public",
			body: map[string]interface{}{
				"type":       "public",
				"permission": "read",
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/shares", tt.body, ownerCookie)
			ts.ExpectStatus(resp, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var share sharing.Share
				ts.ParseJSON(resp, &share)

				if share.ID == "" {
					t.Error("share ID should not be empty")
				}
				if share.PageID != page.ID {
					t.Errorf("page_id = %q, want %q", share.PageID, page.ID)
				}
			}
			resp.Body.Close()
		})
	}
}

func TestShareWithUserPermissions(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, ownerCookie := ts.Register("permowner@example.com", "Perm Owner", "password123")

	ws := createTestWorkspace(ts, ownerCookie, "Perm Workspace", "perm-ws")

	permissionTests := []struct {
		name       string
		permission string
	}{
		{"read permission", "read"},
		{"comment permission", "comment"},
		{"edit permission", "edit"},
		{"full access", "full_access"},
	}

	for _, tt := range permissionTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new user and page for each test
			targetUser, _ := ts.Register(tt.permission+"user@example.com", "User", "password123")
			page := createTestPage(ts, ownerCookie, ws.ID, "Page for "+tt.permission)

			resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/shares", map[string]interface{}{
				"type":       "user",
				"user_id":    targetUser.ID,
				"permission": tt.permission,
			}, ownerCookie)
			ts.ExpectStatus(resp, http.StatusCreated)

			var share sharing.Share
			ts.ParseJSON(resp, &share)

			if string(share.Permission) != tt.permission {
				t.Errorf("permission = %q, want %q", share.Permission, tt.permission)
			}
		})
	}
}

func TestShareList(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, ownerCookie := ts.Register("listowner@example.com", "List Owner", "password123")
	user1, _ := ts.Register("listuser1@example.com", "User 1", "password123")
	user2, _ := ts.Register("listuser2@example.com", "User 2", "password123")

	ws := createTestWorkspace(ts, ownerCookie, "List Workspace", "list-share-ws")
	page := createTestPage(ts, ownerCookie, ws.ID, "List Test Page")

	// Create multiple shares
	for _, user := range []*struct {
		ID string
	}{
		{user1.ID},
		{user2.ID},
	} {
		resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/shares", map[string]interface{}{
			"type":       "user",
			"user_id":    user.ID,
			"permission": "read",
		}, ownerCookie)
		ts.ExpectStatus(resp, http.StatusCreated)
		resp.Body.Close()
	}

	// Create a link share
	resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/shares", map[string]interface{}{
		"type":       "link",
		"permission": "read",
	}, ownerCookie)
	ts.ExpectStatus(resp, http.StatusCreated)
	resp.Body.Close()

	t.Run("list page shares", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/pages/"+page.ID+"/shares", nil, ownerCookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var shareList []*sharing.Share
		ts.ParseJSON(resp, &shareList)

		if len(shareList) < 3 {
			t.Errorf("expected at least 3 shares, got %d", len(shareList))
		}
	})
}

func TestShareDelete(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, ownerCookie := ts.Register("deleteowner@example.com", "Delete Owner", "password123")
	targetUser, _ := ts.Register("deletetarget@example.com", "Delete Target", "password123")

	ws := createTestWorkspace(ts, ownerCookie, "Delete Workspace", "delete-share-ws")
	page := createTestPage(ts, ownerCookie, ws.ID, "Delete Test Page")

	// Create share
	resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/shares", map[string]interface{}{
		"type":       "user",
		"user_id":    targetUser.ID,
		"permission": "read",
	}, ownerCookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created sharing.Share
	ts.ParseJSON(resp, &created)

	t.Run("delete share", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/shares/"+created.ID, nil, ownerCookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("share removed from list", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/pages/"+page.ID+"/shares", nil, ownerCookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var shareList []*sharing.Share
		ts.ParseJSON(resp, &shareList)

		for _, s := range shareList {
			if s.ID == created.ID {
				t.Error("deleted share should not be in list")
			}
		}
	})
}

func TestShareLink(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, ownerCookie := ts.Register("linkowner@example.com", "Link Owner", "password123")

	ws := createTestWorkspace(ts, ownerCookie, "Link Workspace", "link-ws")
	page := createTestPage(ts, ownerCookie, ws.ID, "Link Test Page")

	t.Run("create share link", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/shares", map[string]interface{}{
			"type":       "link",
			"permission": "read",
		}, ownerCookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var share sharing.Share
		ts.ParseJSON(resp, &share)

		if share.Type != sharing.ShareLink {
			t.Errorf("type = %q, want %q", share.Type, sharing.ShareLink)
		}
		if share.Token == "" {
			t.Error("share link should have a token")
		}
	})
}

func TestSharePublic(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, ownerCookie := ts.Register("publicowner@example.com", "Public Owner", "password123")

	ws := createTestWorkspace(ts, ownerCookie, "Public Workspace", "public-ws")
	page := createTestPage(ts, ownerCookie, ws.ID, "Public Test Page")

	t.Run("enable public", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/shares", map[string]interface{}{
			"type":       "public",
			"permission": "read",
		}, ownerCookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var share sharing.Share
		ts.ParseJSON(resp, &share)

		if share.Type != sharing.SharePublic {
			t.Errorf("type = %q, want %q", share.Type, sharing.SharePublic)
		}
	})
}

func TestShareMultipleUsers(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, ownerCookie := ts.Register("multiowner@example.com", "Multi Owner", "password123")

	ws := createTestWorkspace(ts, ownerCookie, "Multi Workspace", "multi-ws")
	page := createTestPage(ts, ownerCookie, ws.ID, "Multi Test Page")

	// Create multiple users and share with each
	for i := 0; i < 5; i++ {
		user, _ := ts.Register(
			"multiuser"+string(rune('1'+i))+"@example.com",
			"User "+string(rune('1'+i)),
			"password123",
		)

		resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/shares", map[string]interface{}{
			"type":       "user",
			"user_id":    user.ID,
			"permission": "read",
		}, ownerCookie)
		ts.ExpectStatus(resp, http.StatusCreated)
		resp.Body.Close()
	}

	// Verify all shares
	resp := ts.Request("GET", "/api/v1/pages/"+page.ID+"/shares", nil, ownerCookie)
	ts.ExpectStatus(resp, http.StatusOK)

	var shareList []*sharing.Share
	ts.ParseJSON(resp, &shareList)

	userShareCount := 0
	for _, s := range shareList {
		if s.Type == sharing.ShareUser {
			userShareCount++
		}
	}

	if userShareCount < 5 {
		t.Errorf("expected at least 5 user shares, got %d", userShareCount)
	}
}

func TestShareUnauthenticated(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, ownerCookie := ts.Register("authowner@example.com", "Auth Owner", "password123")
	targetUser, _ := ts.Register("authtarget@example.com", "Auth Target", "password123")

	ws := createTestWorkspace(ts, ownerCookie, "Auth Workspace", "auth-share-ws")
	page := createTestPage(ts, ownerCookie, ws.ID, "Auth Test Page")

	// Create share
	resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/shares", map[string]interface{}{
		"type":       "user",
		"user_id":    targetUser.ID,
		"permission": "read",
	}, ownerCookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var share sharing.Share
	ts.ParseJSON(resp, &share)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"create share", "POST", "/api/v1/pages/" + page.ID + "/shares"},
		{"list shares", "GET", "/api/v1/pages/" + page.ID + "/shares"},
		{"delete share", "DELETE", "/api/v1/shares/" + share.ID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request(tt.method, tt.path, nil) // No cookie
			ts.ExpectStatus(resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}
