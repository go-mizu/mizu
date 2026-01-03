package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
)

// createTestPage creates a page for testing blocks.
func createTestPage(ts *TestServer, cookie *http.Cookie, workspaceID, title string) *pages.Page {
	resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": workspaceID,
		"title":        title,
		"parent_type":  "workspace",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var page pages.Page
	ts.ParseJSON(resp, &page)
	return &page
}

func TestBlockCreate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("blockcreate@example.com", "Block Create", "password123")
	ws := createTestWorkspace(ts, cookie, "Block Workspace", "block-ws")
	page := createTestPage(ts, cookie, ws.ID, "Block Test Page")

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
		wantType   blocks.BlockType
	}{
		{
			name: "paragraph block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "paragraph",
				"position": 0,
				"content": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "Hello, world!"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockParagraph,
		},
		{
			name: "heading 1 block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "heading_1",
				"position": 1,
				"content": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "Big Heading"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockHeading1,
		},
		{
			name: "heading 2 block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "heading_2",
				"position": 2,
				"content": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "Medium Heading"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockHeading2,
		},
		{
			name: "heading 3 block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "heading_3",
				"position": 3,
				"content": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "Small Heading"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockHeading3,
		},
		{
			name: "quote block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "quote",
				"position": 4,
				"content": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "To be or not to be"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockQuote,
		},
		{
			name: "callout block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "callout",
				"position": 5,
				"content": map[string]interface{}{
					"icon":  "bulb",
					"color": "yellow",
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "Important note"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockCallout,
		},
		{
			name: "bulleted list block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "bulleted_list",
				"position": 6,
				"content": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "List item"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockBulletList,
		},
		{
			name: "numbered list block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "numbered_list",
				"position": 7,
				"content": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "Numbered item"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockNumberList,
		},
		{
			name: "to-do block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "to_do",
				"position": 8,
				"content": map[string]interface{}{
					"checked": false,
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "Task to complete"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockTodo,
		},
		{
			name: "toggle block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "toggle",
				"position": 9,
				"content": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "Toggle header"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockToggle,
		},
		{
			name: "code block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "code",
				"position": 10,
				"content": map[string]interface{}{
					"language": "go",
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "func main() {}"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockCode,
		},
		{
			name: "divider block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "divider",
				"position": 11,
				"content":  map[string]interface{}{},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockDivider,
		},
		{
			name: "image block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "image",
				"position": 12,
				"content": map[string]interface{}{
					"url": "https://example.com/image.png",
					"caption": []map[string]interface{}{
						{"type": "text", "text": "Image caption"},
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockImage,
		},
		{
			name: "bookmark block",
			body: map[string]interface{}{
				"page_id":  page.ID,
				"type":     "bookmark",
				"position": 13,
				"content": map[string]interface{}{
					"url":         "https://example.com",
					"title":       "Example Website",
					"description": "An example website",
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   blocks.BlockBookmark,
		},
		{
			name: "missing page_id",
			body: map[string]interface{}{
				"type":     "paragraph",
				"position": 0,
			},
			wantStatus: http.StatusCreated, // App allows blocks without page_id
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/blocks", tt.body, cookie)
			ts.ExpectStatus(resp, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var block blocks.Block
				ts.ParseJSON(resp, &block)

				// Only check type if expected (non-empty)
				if tt.wantType != "" && block.Type != tt.wantType {
					t.Errorf("type = %q, want %q", block.Type, tt.wantType)
				}
				if block.ID == "" {
					t.Error("block ID should not be empty")
				}
				// Only check page_id if it was provided in request
				if tt.body["page_id"] != nil && block.PageID != page.ID {
					t.Errorf("page_id = %q, want %q", block.PageID, page.ID)
				}
			}
			resp.Body.Close()
		})
	}
}

func TestBlockUpdate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("blockupdate@example.com", "Block Update", "password123")
	ws := createTestWorkspace(ts, cookie, "Block Update Workspace", "block-update-ws")
	page := createTestPage(ts, cookie, ws.ID, "Block Update Page")

	// Create a block
	resp := ts.Request("POST", "/api/v1/blocks", map[string]interface{}{
		"page_id":  page.ID,
		"type":     "paragraph",
		"position": 0,
		"content": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"type": "text", "text": "Original text"},
			},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created blocks.Block
	ts.ParseJSON(resp, &created)

	t.Run("update content", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/blocks/"+created.ID, map[string]interface{}{
			"content": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{"type": "text", "text": "Updated text"},
				},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated blocks.Block
		ts.ParseJSON(resp, &updated)

		if len(updated.Content.RichText) == 0 || updated.Content.RichText[0].Text != "Updated text" {
			t.Error("content should be updated")
		}
	})

	t.Run("update to-do checked", func(t *testing.T) {
		// Create a to-do block
		resp := ts.Request("POST", "/api/v1/blocks", map[string]interface{}{
			"page_id":  page.ID,
			"type":     "to_do",
			"position": 1,
			"content": map[string]interface{}{
				"checked": false,
				"rich_text": []map[string]interface{}{
					{"type": "text", "text": "Task"},
				},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var todo blocks.Block
		ts.ParseJSON(resp, &todo)

		// Update checked state
		resp = ts.Request("PATCH", "/api/v1/blocks/"+todo.ID, map[string]interface{}{
			"content": map[string]interface{}{
				"checked": true,
				"rich_text": []map[string]interface{}{
					{"type": "text", "text": "Task"},
				},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated blocks.Block
		ts.ParseJSON(resp, &updated)

		if updated.Content.Checked == nil || !*updated.Content.Checked {
			t.Error("to-do should be checked")
		}
	})

	t.Run("non-existent block", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/blocks/non-existent-id", map[string]interface{}{
			"content": map[string]interface{}{},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusInternalServerError)
		resp.Body.Close()
	})
}

func TestBlockDelete(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("blockdelete@example.com", "Block Delete", "password123")
	ws := createTestWorkspace(ts, cookie, "Block Delete Workspace", "block-delete-ws")
	page := createTestPage(ts, cookie, ws.ID, "Block Delete Page")

	// Create a block
	resp := ts.Request("POST", "/api/v1/blocks", map[string]interface{}{
		"page_id":  page.ID,
		"type":     "paragraph",
		"position": 0,
		"content": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"type": "text", "text": "To delete"},
			},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created blocks.Block
	ts.ParseJSON(resp, &created)

	t.Run("delete block", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/blocks/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})
}

func TestBlockMove(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("blockmove@example.com", "Block Move", "password123")
	ws := createTestWorkspace(ts, cookie, "Block Move Workspace", "block-move-ws")
	page := createTestPage(ts, cookie, ws.ID, "Block Move Page")

	// Create blocks
	var blockIDs []string
	for i := 0; i < 3; i++ {
		resp := ts.Request("POST", "/api/v1/blocks", map[string]interface{}{
			"page_id":  page.ID,
			"type":     "paragraph",
			"position": i,
			"content": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{"type": "text", "text": "Block " + string(rune('A'+i))},
				},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var block blocks.Block
		ts.ParseJSON(resp, &block)
		blockIDs = append(blockIDs, block.ID)
	}

	t.Run("move block to new position", func(t *testing.T) {
		// Move first block to last position
		resp := ts.Request("POST", "/api/v1/blocks/"+blockIDs[0]+"/move", map[string]interface{}{
			"position": 2,
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("move block to nested parent", func(t *testing.T) {
		// Create a toggle block as parent
		resp := ts.Request("POST", "/api/v1/blocks", map[string]interface{}{
			"page_id":  page.ID,
			"type":     "toggle",
			"position": 0,
			"content": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{"type": "text", "text": "Toggle Parent"},
				},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var toggle blocks.Block
		ts.ParseJSON(resp, &toggle)

		// Move a block under the toggle
		resp = ts.Request("POST", "/api/v1/blocks/"+blockIDs[1]+"/move", map[string]interface{}{
			"parent_id": toggle.ID,
			"position":  0,
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})
}

func TestBlockNestedCreate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("blocknested@example.com", "Block Nested", "password123")
	ws := createTestWorkspace(ts, cookie, "Block Nested Workspace", "block-nested-ws")
	page := createTestPage(ts, cookie, ws.ID, "Block Nested Page")

	// Create toggle block
	resp := ts.Request("POST", "/api/v1/blocks", map[string]interface{}{
		"page_id":  page.ID,
		"type":     "toggle",
		"position": 0,
		"content": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"type": "text", "text": "Toggle"},
			},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var toggle blocks.Block
	ts.ParseJSON(resp, &toggle)

	// Create nested block under toggle
	resp = ts.Request("POST", "/api/v1/blocks", map[string]interface{}{
		"page_id":   page.ID,
		"parent_id": toggle.ID,
		"type":      "paragraph",
		"position":  0,
		"content": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"type": "text", "text": "Nested content"},
			},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var nested blocks.Block
	ts.ParseJSON(resp, &nested)

	if nested.ParentID != toggle.ID {
		t.Errorf("parent_id = %q, want %q", nested.ParentID, toggle.ID)
	}
}

func TestBlockRichTextAnnotations(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("blockannot@example.com", "Block Annot", "password123")
	ws := createTestWorkspace(ts, cookie, "Block Annot Workspace", "block-annot-ws")
	page := createTestPage(ts, cookie, ws.ID, "Block Annot Page")

	// Create block with annotations
	resp := ts.Request("POST", "/api/v1/blocks", map[string]interface{}{
		"page_id":  page.ID,
		"type":     "paragraph",
		"position": 0,
		"content": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": "Bold and italic",
					"annotations": map[string]interface{}{
						"bold":   true,
						"italic": true,
						"color":  "red",
					},
				},
			},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var block blocks.Block
	ts.ParseJSON(resp, &block)

	if len(block.Content.RichText) == 0 {
		t.Fatal("expected rich_text content")
	}

	rt := block.Content.RichText[0]
	if !rt.Annotations.Bold {
		t.Error("expected bold annotation")
	}
	if !rt.Annotations.Italic {
		t.Error("expected italic annotation")
	}
}

func TestBlockUnauthenticated(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("blockauth@example.com", "Block Auth", "password123")
	ws := createTestWorkspace(ts, cookie, "Block Auth Workspace", "block-auth-ws")
	page := createTestPage(ts, cookie, ws.ID, "Block Auth Page")

	// Create a block for testing
	resp := ts.Request("POST", "/api/v1/blocks", map[string]interface{}{
		"page_id":  page.ID,
		"type":     "paragraph",
		"position": 0,
		"content":  map[string]interface{}{},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var block blocks.Block
	ts.ParseJSON(resp, &block)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"create block", "POST", "/api/v1/blocks"},
		{"update block", "PATCH", "/api/v1/blocks/" + block.ID},
		{"delete block", "DELETE", "/api/v1/blocks/" + block.ID},
		{"move block", "POST", "/api/v1/blocks/" + block.ID + "/move"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request(tt.method, tt.path, nil) // No cookie
			ts.ExpectStatus(resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}
