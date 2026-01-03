package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/members"
	"github.com/go-mizu/blueprints/workspace/feature/workspaces"
)

func TestWorkspaceCreate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register a user
	_, cookie := ts.Register("workspace@example.com", "Workspace Test", "password123")

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "valid workspace",
			body: map[string]interface{}{
				"name": "My Workspace",
				"slug": "my-workspace",
				"icon": "rocket",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "workspace without icon",
			body: map[string]interface{}{
				"name": "Another Workspace",
				"slug": "another-workspace",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "missing name",
			body: map[string]interface{}{
				"slug": "no-name",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing slug",
			body: map[string]interface{}{
				"name": "No Slug Workspace",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/workspaces", tt.body, cookie)
			ts.ExpectStatus(resp, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var ws workspaces.Workspace
				ts.ParseJSON(resp, &ws)

				if ws.Name != tt.body["name"] {
					t.Errorf("name = %q, want %q", ws.Name, tt.body["name"])
				}
				if ws.Slug != tt.body["slug"] {
					t.Errorf("slug = %q, want %q", ws.Slug, tt.body["slug"])
				}
				if ws.ID == "" {
					t.Error("workspace ID should not be empty")
				}
			}
			resp.Body.Close()
		})
	}
}

func TestWorkspaceCreateUnauthenticated(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	resp := ts.Request("POST", "/api/v1/workspaces", map[string]interface{}{
		"name": "Test Workspace",
		"slug": "test-workspace",
	})
	ts.ExpectStatus(resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestWorkspaceList(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register a user
	_, cookie := ts.Register("list@example.com", "List Test", "password123")

	t.Run("empty list", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var list []*workspaces.Workspace
		ts.ParseJSON(resp, &list)

		// Initially empty or may have default workspace
	})

	t.Run("after creating workspaces", func(t *testing.T) {
		// Create workspaces
		for i := 0; i < 3; i++ {
			resp := ts.Request("POST", "/api/v1/workspaces", map[string]interface{}{
				"name": "Workspace " + string(rune('A'+i)),
				"slug": "workspace-" + string(rune('a'+i)),
			}, cookie)
			ts.ExpectStatus(resp, http.StatusCreated)
			resp.Body.Close()
		}

		// List workspaces
		resp := ts.Request("GET", "/api/v1/workspaces", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var list []*workspaces.Workspace
		ts.ParseJSON(resp, &list)

		if len(list) < 3 {
			t.Errorf("expected at least 3 workspaces, got %d", len(list))
		}
	})
}

func TestWorkspaceGet(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register and create workspace
	_, cookie := ts.Register("get@example.com", "Get Test", "password123")

	resp := ts.Request("POST", "/api/v1/workspaces", map[string]interface{}{
		"name": "Get Test Workspace",
		"slug": "get-test-workspace",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created workspaces.Workspace
	ts.ParseJSON(resp, &created)

	t.Run("get by ID", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var ws workspaces.Workspace
		ts.ParseJSON(resp, &ws)

		if ws.ID != created.ID {
			t.Errorf("ID = %q, want %q", ws.ID, created.ID)
		}
		if ws.Name != created.Name {
			t.Errorf("Name = %q, want %q", ws.Name, created.Name)
		}
	})

	t.Run("get by slug", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+created.Slug, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var ws workspaces.Workspace
		ts.ParseJSON(resp, &ws)

		if ws.Slug != created.Slug {
			t.Errorf("Slug = %q, want %q", ws.Slug, created.Slug)
		}
	})

	t.Run("non-existent workspace", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/non-existent-id", nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

func TestWorkspaceUpdate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register and create workspace
	_, cookie := ts.Register("update@example.com", "Update Test", "password123")

	resp := ts.Request("POST", "/api/v1/workspaces", map[string]interface{}{
		"name": "Original Name",
		"slug": "original-slug",
		"icon": "star",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created workspaces.Workspace
	ts.ParseJSON(resp, &created)

	t.Run("update name", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/workspaces/"+created.ID, map[string]interface{}{
			"name": "Updated Name",
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated workspaces.Workspace
		ts.ParseJSON(resp, &updated)

		if updated.Name != "Updated Name" {
			t.Errorf("Name = %q, want %q", updated.Name, "Updated Name")
		}
	})

	t.Run("update icon", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/workspaces/"+created.ID, map[string]interface{}{
			"icon": "moon",
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated workspaces.Workspace
		ts.ParseJSON(resp, &updated)

		if updated.Icon != "moon" {
			t.Errorf("Icon = %q, want %q", updated.Icon, "moon")
		}
	})
}

func TestWorkspaceDelete(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register and create workspace
	_, cookie := ts.Register("delete@example.com", "Delete Test", "password123")

	resp := ts.Request("POST", "/api/v1/workspaces", map[string]interface{}{
		"name": "To Delete",
		"slug": "to-delete",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created workspaces.Workspace
	ts.ParseJSON(resp, &created)

	t.Run("delete workspace", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/workspaces/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("deleted workspace not found", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

func TestWorkspaceMembers(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register owner
	owner, ownerCookie := ts.Register("owner@example.com", "Owner", "password123")

	// Register another user
	member, _ := ts.Register("member@example.com", "Member", "password123")

	// Create workspace
	resp := ts.Request("POST", "/api/v1/workspaces", map[string]interface{}{
		"name": "Team Workspace",
		"slug": "team-workspace",
	}, ownerCookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var ws workspaces.Workspace
	ts.ParseJSON(resp, &ws)

	t.Run("list members", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/members", nil, ownerCookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var memberList []*members.Member
		ts.ParseJSON(resp, &memberList)

		// Owner should be in the list
		var foundOwner bool
		for _, m := range memberList {
			if m.UserID == owner.ID {
				foundOwner = true
				if m.Role != members.RoleOwner {
					t.Errorf("owner role = %q, want %q", m.Role, members.RoleOwner)
				}
			}
		}
		if !foundOwner {
			t.Error("owner not found in member list")
		}
	})

	t.Run("add member", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/workspaces/"+ws.ID+"/members", map[string]interface{}{
			"user_id": member.ID,
			"role":    "member",
		}, ownerCookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var newMember members.Member
		ts.ParseJSON(resp, &newMember)

		if newMember.UserID != member.ID {
			t.Errorf("member user_id = %q, want %q", newMember.UserID, member.ID)
		}
		if newMember.Role != members.RoleMember {
			t.Errorf("member role = %q, want %q", newMember.Role, members.RoleMember)
		}
	})

	t.Run("list members after adding", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/members", nil, ownerCookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var memberList []*members.Member
		ts.ParseJSON(resp, &memberList)

		if len(memberList) < 2 {
			t.Errorf("expected at least 2 members, got %d", len(memberList))
		}
	})
}

func TestWorkspaceDuplicateSlug(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register a user
	_, cookie := ts.Register("dupslug@example.com", "Dup Slug", "password123")

	// Create first workspace
	resp := ts.Request("POST", "/api/v1/workspaces", map[string]interface{}{
		"name": "First Workspace",
		"slug": "duplicate-slug",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)
	resp.Body.Close()

	// Try to create second workspace with same slug
	resp = ts.Request("POST", "/api/v1/workspaces", map[string]interface{}{
		"name": "Second Workspace",
		"slug": "duplicate-slug",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusBadRequest)
	resp.Body.Close()
}
