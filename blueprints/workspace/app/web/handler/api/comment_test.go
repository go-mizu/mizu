package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/comments"
)

func TestCommentCreate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("commentcreate@example.com", "Comment Create", "password123")
	ws := createTestWorkspace(ts, cookie, "Comment Workspace", "comment-ws")
	page := createTestPage(ts, cookie, ws.ID, "Comment Test Page")

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "page comment",
			body: map[string]interface{}{
				"workspace_id": ws.ID,
				"target_type":  "page",
				"target_id":    page.ID,
				"content": []map[string]interface{}{
					{"type": "text", "text": "This is a comment"},
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "comment with formatting",
			body: map[string]interface{}{
				"workspace_id": ws.ID,
				"target_type":  "page",
				"target_id":    page.ID,
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Bold comment",
						"annotations": map[string]interface{}{
							"bold": true,
						},
					},
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "missing target",
			body: map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "No target"},
				},
			},
			wantStatus: http.StatusCreated, // App allows comments without target
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/comments", tt.body, cookie)
			ts.ExpectStatus(resp, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var comment comments.Comment
				ts.ParseJSON(resp, &comment)

				if comment.ID == "" {
					t.Error("comment ID should not be empty")
				}
				// Only check target_id if target_id was provided in request
				if tt.body["target_id"] != nil && comment.TargetID != page.ID {
					t.Errorf("target_id = %q, want %q", comment.TargetID, page.ID)
				}
			}
			resp.Body.Close()
		})
	}
}

func TestCommentCreateReply(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("commentreply@example.com", "Comment Reply", "password123")
	ws := createTestWorkspace(ts, cookie, "Reply Workspace", "reply-ws")
	page := createTestPage(ts, cookie, ws.ID, "Reply Test Page")

	// Create parent comment
	resp := ts.Request("POST", "/api/v1/comments", map[string]interface{}{
		"workspace_id": ws.ID,
		"target_type":  "page",
		"target_id":    page.ID,
		"content": []map[string]interface{}{
			{"type": "text", "text": "Parent comment"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var parent comments.Comment
	ts.ParseJSON(resp, &parent)

	// Create reply
	resp = ts.Request("POST", "/api/v1/comments", map[string]interface{}{
		"workspace_id": ws.ID,
		"target_type":  "page",
		"target_id":    page.ID,
		"parent_id":    parent.ID,
		"content": []map[string]interface{}{
			{"type": "text", "text": "Reply comment"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var reply comments.Comment
	ts.ParseJSON(resp, &reply)

	if reply.ParentID != parent.ID {
		t.Errorf("parent_id = %q, want %q", reply.ParentID, parent.ID)
	}
}

func TestCommentUpdate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("commentupdate@example.com", "Comment Update", "password123")
	ws := createTestWorkspace(ts, cookie, "Update Workspace", "update-comment-ws")
	page := createTestPage(ts, cookie, ws.ID, "Update Test Page")

	// Create comment
	resp := ts.Request("POST", "/api/v1/comments", map[string]interface{}{
		"workspace_id": ws.ID,
		"target_type":  "page",
		"target_id":    page.ID,
		"content": []map[string]interface{}{
			{"type": "text", "text": "Original comment"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created comments.Comment
	ts.ParseJSON(resp, &created)

	t.Run("update content", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/comments/"+created.ID, map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "Updated comment"},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated comments.Comment
		ts.ParseJSON(resp, &updated)

		if len(updated.Content) == 0 || updated.Content[0].Text != "Updated comment" {
			t.Error("content should be updated")
		}
	})

	t.Run("non-existent comment", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/comments/non-existent-id", map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "Update"},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusInternalServerError)
		resp.Body.Close()
	})
}

func TestCommentDelete(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("commentdelete@example.com", "Comment Delete", "password123")
	ws := createTestWorkspace(ts, cookie, "Delete Workspace", "delete-comment-ws")
	page := createTestPage(ts, cookie, ws.ID, "Delete Test Page")

	// Create comment
	resp := ts.Request("POST", "/api/v1/comments", map[string]interface{}{
		"workspace_id": ws.ID,
		"target_type":  "page",
		"target_id":    page.ID,
		"content": []map[string]interface{}{
			{"type": "text", "text": "To delete"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created comments.Comment
	ts.ParseJSON(resp, &created)

	t.Run("delete comment", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/comments/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})
}

func TestCommentList(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("commentlist@example.com", "Comment List", "password123")
	ws := createTestWorkspace(ts, cookie, "List Workspace", "list-comment-ws")
	page := createTestPage(ts, cookie, ws.ID, "List Test Page")

	// Create comments
	for i := 0; i < 3; i++ {
		resp := ts.Request("POST", "/api/v1/comments", map[string]interface{}{
			"workspace_id": ws.ID,
			"target_type":  "page",
			"target_id":    page.ID,
			"content": []map[string]interface{}{
				{"type": "text", "text": "Comment " + string(rune('A'+i))},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)
		resp.Body.Close()
	}

	t.Run("list page comments", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/pages/"+page.ID+"/comments", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var commentList []*comments.Comment
		ts.ParseJSON(resp, &commentList)

		if len(commentList) < 3 {
			t.Errorf("expected at least 3 comments, got %d", len(commentList))
		}
	})
}

func TestCommentResolve(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("commentresolve@example.com", "Comment Resolve", "password123")
	ws := createTestWorkspace(ts, cookie, "Resolve Workspace", "resolve-ws")
	page := createTestPage(ts, cookie, ws.ID, "Resolve Test Page")

	// Create comment
	resp := ts.Request("POST", "/api/v1/comments", map[string]interface{}{
		"workspace_id": ws.ID,
		"target_type":  "page",
		"target_id":    page.ID,
		"content": []map[string]interface{}{
			{"type": "text", "text": "To resolve"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created comments.Comment
	ts.ParseJSON(resp, &created)

	if created.IsResolved {
		t.Error("new comment should not be resolved")
	}

	t.Run("resolve comment", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/comments/"+created.ID+"/resolve", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})
}

func TestCommentThreaded(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("commentthread@example.com", "Comment Thread", "password123")
	ws := createTestWorkspace(ts, cookie, "Thread Workspace", "thread-ws")
	page := createTestPage(ts, cookie, ws.ID, "Thread Test Page")

	// Create parent comment
	resp := ts.Request("POST", "/api/v1/comments", map[string]interface{}{
		"workspace_id": ws.ID,
		"target_type":  "page",
		"target_id":    page.ID,
		"content": []map[string]interface{}{
			{"type": "text", "text": "Thread starter"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var parent comments.Comment
	ts.ParseJSON(resp, &parent)

	// Create multiple replies
	for i := 0; i < 3; i++ {
		resp := ts.Request("POST", "/api/v1/comments", map[string]interface{}{
			"workspace_id": ws.ID,
			"target_type":  "page",
			"target_id":    page.ID,
			"parent_id":    parent.ID,
			"content": []map[string]interface{}{
				{"type": "text", "text": "Reply " + string(rune('1'+i))},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)
		resp.Body.Close()
	}

	// List all comments - replies are nested in parent's Replies field
	resp = ts.Request("GET", "/api/v1/pages/"+page.ID+"/comments", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)

	var commentList []*comments.Comment
	ts.ParseJSON(resp, &commentList)

	// Should have 1 root comment with 3 replies nested inside
	if len(commentList) != 1 {
		t.Errorf("expected 1 root comment, got %d", len(commentList))
	}
	if len(commentList) > 0 && len(commentList[0].Replies) != 3 {
		t.Errorf("expected 3 replies, got %d", len(commentList[0].Replies))
	}
}

func TestCommentWithMention(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup - create two users
	user1, cookie1 := ts.Register("mentioner@example.com", "Mentioner", "password123")
	user2, _ := ts.Register("mentioned@example.com", "Mentioned", "password123")

	ws := createTestWorkspace(ts, cookie1, "Mention Workspace", "mention-ws")
	page := createTestPage(ts, cookie1, ws.ID, "Mention Test Page")

	// Create comment with mention
	resp := ts.Request("POST", "/api/v1/comments", map[string]interface{}{
		"workspace_id": ws.ID,
		"target_type":  "page",
		"target_id":    page.ID,
		"content": []map[string]interface{}{
			{"type": "text", "text": "Hey "},
			{
				"type": "text",
				"text": "@Mentioned",
				"mention": map[string]interface{}{
					"type":    "user",
					"user_id": user2.ID,
				},
			},
			{"type": "text", "text": " check this out!"},
		},
	}, cookie1)
	ts.ExpectStatus(resp, http.StatusCreated)

	var comment comments.Comment
	ts.ParseJSON(resp, &comment)

	if comment.AuthorID != user1.ID {
		t.Errorf("author_id = %q, want %q", comment.AuthorID, user1.ID)
	}
}

func TestCommentUnauthenticated(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("commentauth@example.com", "Comment Auth", "password123")
	ws := createTestWorkspace(ts, cookie, "Auth Workspace", "auth-comment-ws")
	page := createTestPage(ts, cookie, ws.ID, "Auth Test Page")

	// Create comment
	resp := ts.Request("POST", "/api/v1/comments", map[string]interface{}{
		"workspace_id": ws.ID,
		"target_type":  "page",
		"target_id":    page.ID,
		"content": []map[string]interface{}{
			{"type": "text", "text": "Auth test"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var comment comments.Comment
	ts.ParseJSON(resp, &comment)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"create comment", "POST", "/api/v1/comments"},
		{"update comment", "PATCH", "/api/v1/comments/" + comment.ID},
		{"delete comment", "DELETE", "/api/v1/comments/" + comment.ID},
		{"list comments", "GET", "/api/v1/pages/" + page.ID + "/comments"},
		{"resolve comment", "POST", "/api/v1/comments/" + comment.ID + "/resolve"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request(tt.method, tt.path, nil) // No cookie
			ts.ExpectStatus(resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}
