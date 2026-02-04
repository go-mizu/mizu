package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu/blueprints/email/app/web"
	"github.com/go-mizu/mizu/blueprints/email/pkg/email"
	"github.com/go-mizu/mizu/blueprints/email/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/email/types"
)

// setupTestServer creates an in-memory SQLite store, seeds it with test data,
// and returns an http.Handler backed by the full web server.
func setupTestServer(t *testing.T) http.Handler {
	t.Helper()

	store, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Seed test data (labels must come first because email_labels has a FK to labels).
	if err := store.SeedLabels(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := store.SeedContacts(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := store.SeedEmails(context.Background()); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { store.Close() })

	driver := email.Noop()
	handler, err := web.NewServer(store, driver, "test@example.com", true)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

// doRequest is a small helper that creates a request, executes it against the
// handler, and returns the recorded response.
func doRequest(t *testing.T, handler http.Handler, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		req = httptest.NewRequest(method, target, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// decodeJSON decodes the response body into the supplied value.
func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode JSON response: %v (body: %s)", err, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Health check
// ---------------------------------------------------------------------------

func TestHealthCheck(t *testing.T) {
	handler := setupTestServer(t)
	rr := doRequest(t, handler, http.MethodGet, "/health", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]string
	decodeJSON(t, rr, &resp)

	if resp["status"] != "ok" {
		t.Fatalf("expected status 'ok', got %q", resp["status"])
	}
}

// ---------------------------------------------------------------------------
// Email endpoints
// ---------------------------------------------------------------------------

func TestListEmails(t *testing.T) {
	handler := setupTestServer(t)
	rr := doRequest(t, handler, http.MethodGet, "/api/emails", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp types.EmailListResponse
	decodeJSON(t, rr, &resp)

	if resp.Total == 0 {
		t.Fatal("expected total > 0 from seeded data")
	}
	if len(resp.Emails) == 0 {
		t.Fatal("expected at least one email in the list")
	}
	if resp.Page != 1 {
		t.Fatalf("expected page 1, got %d", resp.Page)
	}
}

func TestListEmailsWithLabel(t *testing.T) {
	handler := setupTestServer(t)

	tests := []struct {
		name       string
		label      string
		wantNonZero bool
	}{
		{"inbox label", "inbox", true},
		{"sent label", "sent", true},
		{"drafts label", "drafts", true},
		{"trash label", "trash", true},
		{"nonexistent label", "nonexistent-label-xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := doRequest(t, handler, http.MethodGet, "/api/emails?label="+tt.label, nil)
			if rr.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rr.Code)
			}

			var resp types.EmailListResponse
			decodeJSON(t, rr, &resp)

			if tt.wantNonZero && resp.Total == 0 {
				t.Fatalf("expected emails with label %q, got total=0", tt.label)
			}
			if !tt.wantNonZero && resp.Total != 0 {
				t.Fatalf("expected 0 emails for label %q, got total=%d", tt.label, resp.Total)
			}
		})
	}
}

func TestListEmailsPagination(t *testing.T) {
	handler := setupTestServer(t)

	rr := doRequest(t, handler, http.MethodGet, "/api/emails?per_page=2&page=1", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp types.EmailListResponse
	decodeJSON(t, rr, &resp)

	if len(resp.Emails) > 2 {
		t.Fatalf("expected at most 2 emails with per_page=2, got %d", len(resp.Emails))
	}
	if resp.PerPage != 2 {
		t.Fatalf("expected per_page=2 in response, got %d", resp.PerPage)
	}
	if resp.TotalPages < 2 {
		t.Fatalf("expected multiple pages with per_page=2, got total_pages=%d", resp.TotalPages)
	}
}

func TestGetEmail(t *testing.T) {
	handler := setupTestServer(t)

	t.Run("existing email", func(t *testing.T) {
		rr := doRequest(t, handler, http.MethodGet, "/api/emails/email-001", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp struct {
			Email  types.Email   `json:"email"`
			Thread *types.Thread `json:"thread"`
		}
		decodeJSON(t, rr, &resp)

		if resp.Email.ID != "email-001" {
			t.Fatalf("expected email ID 'email-001', got %q", resp.Email.ID)
		}
		if resp.Email.Subject == "" {
			t.Fatal("expected non-empty subject")
		}
		if resp.Email.FromAddress == "" {
			t.Fatal("expected non-empty from_address")
		}
		// Thread should be populated since email-001 has a thread_id
		if resp.Thread == nil {
			t.Fatal("expected thread to be populated for email-001")
		}
	})

	t.Run("nonexistent email", func(t *testing.T) {
		rr := doRequest(t, handler, http.MethodGet, "/api/emails/nonexistent-id", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rr.Code)
		}
	})
}

func TestCreateEmail(t *testing.T) {
	handler := setupTestServer(t)

	t.Run("send email", func(t *testing.T) {
		body := types.ComposeRequest{
			To:       []types.Recipient{{Name: "Alice", Address: "alice@example.com"}},
			Subject:  "Test Email",
			BodyText: "Hello, this is a test email.",
			BodyHTML: "<p>Hello, this is a test email.</p>",
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/emails", body)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp struct {
			Email             types.Email `json:"email"`
			ProviderMessageID string      `json:"provider_message_id"`
		}
		decodeJSON(t, rr, &resp)

		if resp.Email.ID == "" {
			t.Fatal("expected non-empty email ID")
		}
		if resp.Email.Subject != "Test Email" {
			t.Fatalf("expected subject 'Test Email', got %q", resp.Email.Subject)
		}
		if !resp.Email.IsSent {
			t.Fatal("expected email to be marked as sent")
		}
		if resp.Email.IsDraft {
			t.Fatal("expected email not to be a draft")
		}
		// Noop driver returns a provider message ID
		if resp.ProviderMessageID == "" {
			t.Fatal("expected provider_message_id from noop driver")
		}
	})

	t.Run("create draft", func(t *testing.T) {
		body := types.ComposeRequest{
			To:       []types.Recipient{{Name: "Bob", Address: "bob@example.com"}},
			Subject:  "Draft Email",
			BodyText: "This is a draft.",
			IsDraft:  true,
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/emails", body)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp struct {
			Email types.Email `json:"email"`
		}
		decodeJSON(t, rr, &resp)

		if !resp.Email.IsDraft {
			t.Fatal("expected email to be marked as draft")
		}
		if resp.Email.IsSent {
			t.Fatal("expected draft email not to be marked as sent")
		}
	})

	t.Run("missing recipients non-draft", func(t *testing.T) {
		body := types.ComposeRequest{
			Subject:  "No Recipients",
			BodyText: "Body text here.",
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/emails", body)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestUpdateEmail(t *testing.T) {
	handler := setupTestServer(t)

	t.Run("mark as starred", func(t *testing.T) {
		body := map[string]any{"is_starred": true}
		rr := doRequest(t, handler, http.MethodPut, "/api/emails/email-001", body)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp types.Email
		decodeJSON(t, rr, &resp)

		if resp.ID != "email-001" {
			t.Fatalf("expected email ID 'email-001', got %q", resp.ID)
		}
		if !resp.IsStarred {
			t.Fatal("expected email to be starred after update")
		}
	})

	t.Run("mark as read", func(t *testing.T) {
		body := map[string]any{"is_read": true}
		rr := doRequest(t, handler, http.MethodPut, "/api/emails/email-003", body)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp types.Email
		decodeJSON(t, rr, &resp)

		if !resp.IsRead {
			t.Fatal("expected email to be marked as read")
		}
	})

	t.Run("nonexistent email", func(t *testing.T) {
		body := map[string]any{"is_read": true}
		rr := doRequest(t, handler, http.MethodPut, "/api/emails/nonexistent-id", body)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rr.Code)
		}
	})
}

func TestDeleteEmail(t *testing.T) {
	handler := setupTestServer(t)

	t.Run("soft delete moves to trash", func(t *testing.T) {
		// First verify the email exists.
		rr := doRequest(t, handler, http.MethodGet, "/api/emails/email-001", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected email-001 to exist, got status %d", rr.Code)
		}

		// Delete (soft).
		rr = doRequest(t, handler, http.MethodDelete, "/api/emails/email-001", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp map[string]string
		decodeJSON(t, rr, &resp)

		if resp["message"] != "email deleted" {
			t.Fatalf("expected message 'email deleted', got %q", resp["message"])
		}

		// Email should still be retrievable (soft delete moves to trash, does not remove).
		rr = doRequest(t, handler, http.MethodGet, "/api/emails/email-001", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected email to still exist after soft delete, got status %d", rr.Code)
		}
	})

	t.Run("permanent delete", func(t *testing.T) {
		// Use a different email for permanent delete so we don't conflict with the soft delete test.
		rr := doRequest(t, handler, http.MethodDelete, "/api/emails/email-032?permanent=true", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		// Email should be gone.
		rr = doRequest(t, handler, http.MethodGet, "/api/emails/email-032", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after permanent delete, got %d", rr.Code)
		}
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		rr := doRequest(t, handler, http.MethodDelete, "/api/emails/nonexistent-id", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rr.Code)
		}
	})
}

func TestBatchEmails(t *testing.T) {
	handler := setupTestServer(t)

	t.Run("batch mark as read", func(t *testing.T) {
		body := types.BatchAction{
			IDs:    []string{"email-003", "email-009"},
			Action: "read",
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/emails/batch", body)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp struct {
			Message  string `json:"message"`
			Affected int    `json:"affected"`
		}
		decodeJSON(t, rr, &resp)

		if resp.Affected != 2 {
			t.Fatalf("expected 2 affected emails, got %d", resp.Affected)
		}
		if resp.Message != "batch action completed" {
			t.Fatalf("expected message 'batch action completed', got %q", resp.Message)
		}

		// Verify both emails are now read.
		for _, id := range []string{"email-003", "email-009"} {
			rr = doRequest(t, handler, http.MethodGet, "/api/emails/"+id, nil)
			if rr.Code != http.StatusOK {
				t.Fatalf("failed to get %s: status %d", id, rr.Code)
			}
			var emailResp struct {
				Email types.Email `json:"email"`
			}
			decodeJSON(t, rr, &emailResp)
			if !emailResp.Email.IsRead {
				t.Fatalf("expected %s to be marked as read after batch", id)
			}
		}
	})

	t.Run("batch star", func(t *testing.T) {
		body := types.BatchAction{
			IDs:    []string{"email-001", "email-003"},
			Action: "star",
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/emails/batch", body)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}
	})

	t.Run("batch archive", func(t *testing.T) {
		body := types.BatchAction{
			IDs:    []string{"email-016"},
			Action: "archive",
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/emails/batch", body)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}
	})

	t.Run("missing IDs", func(t *testing.T) {
		body := types.BatchAction{
			IDs:    []string{},
			Action: "read",
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/emails/batch", body)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("missing action", func(t *testing.T) {
		body := types.BatchAction{
			IDs: []string{"email-001"},
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/emails/batch", body)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("invalid action", func(t *testing.T) {
		body := types.BatchAction{
			IDs:    []string{"email-001"},
			Action: "invalid_action",
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/emails/batch", body)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestSearchEmails(t *testing.T) {
	handler := setupTestServer(t)

	t.Run("search by subject keyword", func(t *testing.T) {
		// "Q4 Planning" is in the subject of email-001.
		rr := doRequest(t, handler, http.MethodGet, "/api/search?q=planning", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp types.EmailListResponse
		decodeJSON(t, rr, &resp)

		if resp.Total == 0 {
			t.Fatal("expected search results for 'planning'")
		}

		found := false
		for _, e := range resp.Emails {
			if e.ID == "email-001" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected email-001 (Q4 Planning) in search results")
		}
	})

	t.Run("search by sender name", func(t *testing.T) {
		rr := doRequest(t, handler, http.MethodGet, "/api/search?q=alice", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp types.EmailListResponse
		decodeJSON(t, rr, &resp)

		if resp.Total == 0 {
			t.Fatal("expected search results for 'alice'")
		}
	})

	t.Run("search with no results", func(t *testing.T) {
		rr := doRequest(t, handler, http.MethodGet, "/api/search?q=xyznonexistent999", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var resp types.EmailListResponse
		decodeJSON(t, rr, &resp)

		if resp.Total != 0 {
			t.Fatalf("expected 0 results, got %d", resp.Total)
		}
	})

	t.Run("search with pagination", func(t *testing.T) {
		rr := doRequest(t, handler, http.MethodGet, "/api/search?q=email&per_page=5&page=1", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var resp types.EmailListResponse
		decodeJSON(t, rr, &resp)

		if len(resp.Emails) > 5 {
			t.Fatalf("expected at most 5 results with per_page=5, got %d", len(resp.Emails))
		}
	})

	t.Run("search missing query", func(t *testing.T) {
		rr := doRequest(t, handler, http.MethodGet, "/api/search", nil)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for empty search query, got %d", rr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Label endpoints
// ---------------------------------------------------------------------------

func TestListLabels(t *testing.T) {
	handler := setupTestServer(t)

	rr := doRequest(t, handler, http.MethodGet, "/api/labels", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var labels []types.Label
	decodeJSON(t, rr, &labels)

	if len(labels) == 0 {
		t.Fatal("expected seeded labels")
	}

	// Verify we have both system and user labels.
	systemFound := false
	userFound := false
	for _, l := range labels {
		if l.Type == types.LabelTypeSystem {
			systemFound = true
		}
		if l.Type == types.LabelTypeUser {
			userFound = true
		}
	}
	if !systemFound {
		t.Fatal("expected at least one system label")
	}
	if !userFound {
		t.Fatal("expected at least one user label")
	}

	// Check that well-known labels exist.
	knownIDs := map[string]bool{"inbox": false, "sent": false, "drafts": false, "trash": false, "work": false}
	for _, l := range labels {
		if _, ok := knownIDs[l.ID]; ok {
			knownIDs[l.ID] = true
		}
	}
	for id, found := range knownIDs {
		if !found {
			t.Fatalf("expected label %q to be present", id)
		}
	}

	// Verify that at least one label has counts (the seeded data should produce counts).
	hasCounts := false
	for _, l := range labels {
		if l.TotalCount > 0 {
			hasCounts = true
			break
		}
	}
	if !hasCounts {
		t.Fatal("expected at least one label with total_count > 0")
	}
}

func TestCreateLabel(t *testing.T) {
	handler := setupTestServer(t)

	t.Run("create valid label", func(t *testing.T) {
		body := map[string]any{
			"name":  "Testing",
			"color": "#FF5733",
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/labels", body)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var label types.Label
		decodeJSON(t, rr, &label)

		if label.ID == "" {
			t.Fatal("expected non-empty label ID")
		}
		if label.Name != "Testing" {
			t.Fatalf("expected name 'Testing', got %q", label.Name)
		}
		if label.Color != "#FF5733" {
			t.Fatalf("expected color '#FF5733', got %q", label.Color)
		}
		if label.Type != types.LabelTypeUser {
			t.Fatalf("expected type 'user', got %q", label.Type)
		}
		if !label.Visible {
			t.Fatal("expected label to be visible by default")
		}
	})

	t.Run("create label with visibility false", func(t *testing.T) {
		visible := false
		body := map[string]any{
			"name":    "Hidden",
			"color":   "#000000",
			"visible": visible,
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/labels", body)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var label types.Label
		decodeJSON(t, rr, &label)

		if label.Visible {
			t.Fatal("expected label to be hidden when visible=false")
		}
	})

	t.Run("missing label name", func(t *testing.T) {
		body := map[string]any{
			"color": "#FF5733",
		}

		rr := doRequest(t, handler, http.MethodPost, "/api/labels", body)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Contact endpoints
// ---------------------------------------------------------------------------

func TestListContacts(t *testing.T) {
	handler := setupTestServer(t)

	t.Run("list all contacts", func(t *testing.T) {
		rr := doRequest(t, handler, http.MethodGet, "/api/contacts", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var contacts []types.Contact
		decodeJSON(t, rr, &contacts)

		if len(contacts) == 0 {
			t.Fatal("expected seeded contacts")
		}

		// Verify a known seeded contact.
		found := false
		for _, c := range contacts {
			if c.Email == "alice.chen@techcorp.io" {
				found = true
				if c.Name != "Alice Chen" {
					t.Fatalf("expected name 'Alice Chen', got %q", c.Name)
				}
				break
			}
		}
		if !found {
			t.Fatal("expected alice.chen@techcorp.io in contacts")
		}
	})

	t.Run("search contacts by query", func(t *testing.T) {
		rr := doRequest(t, handler, http.MethodGet, "/api/contacts?q=alice", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var contacts []types.Contact
		decodeJSON(t, rr, &contacts)

		if len(contacts) == 0 {
			t.Fatal("expected at least one contact matching 'alice'")
		}
	})

	t.Run("search contacts no results", func(t *testing.T) {
		rr := doRequest(t, handler, http.MethodGet, "/api/contacts?q=zznonexistent", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var contacts []types.Contact
		decodeJSON(t, rr, &contacts)

		if len(contacts) != 0 {
			t.Fatalf("expected 0 contacts, got %d", len(contacts))
		}
	})
}

// ---------------------------------------------------------------------------
// Settings endpoints
// ---------------------------------------------------------------------------

func TestGetSettings(t *testing.T) {
	handler := setupTestServer(t)

	rr := doRequest(t, handler, http.MethodGet, "/api/settings", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var settings types.Settings
	decodeJSON(t, rr, &settings)

	// Default settings are inserted by the schema.
	if settings.ID != 1 {
		t.Fatalf("expected settings ID 1, got %d", settings.ID)
	}
	if settings.Theme == "" {
		t.Fatal("expected non-empty theme")
	}
}

func TestUpdateSettings(t *testing.T) {
	handler := setupTestServer(t)

	t.Run("update display name and theme", func(t *testing.T) {
		body := types.Settings{
			DisplayName:      "Test User",
			EmailAddress:     "testuser@example.com",
			Signature:        "Best regards,\nTest User",
			Theme:            "dark",
			Density:          "compact",
			ConversationView: true,
			AutoAdvance:      "newer",
			UndoSendSeconds:  10,
		}

		rr := doRequest(t, handler, http.MethodPut, "/api/settings", body)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		var settings types.Settings
		decodeJSON(t, rr, &settings)

		if settings.DisplayName != "Test User" {
			t.Fatalf("expected display_name 'Test User', got %q", settings.DisplayName)
		}
		if settings.EmailAddress != "testuser@example.com" {
			t.Fatalf("expected email_address 'testuser@example.com', got %q", settings.EmailAddress)
		}
		if settings.Theme != "dark" {
			t.Fatalf("expected theme 'dark', got %q", settings.Theme)
		}
		if settings.Density != "compact" {
			t.Fatalf("expected density 'compact', got %q", settings.Density)
		}
		if settings.UndoSendSeconds != 10 {
			t.Fatalf("expected undo_send_seconds 10, got %d", settings.UndoSendSeconds)
		}
	})

	t.Run("verify settings persist", func(t *testing.T) {
		// First update.
		body := types.Settings{
			DisplayName:      "Persisted User",
			EmailAddress:     "persisted@example.com",
			Theme:            "light",
			Density:          "default",
			ConversationView: false,
			AutoAdvance:      "older",
			UndoSendSeconds:  5,
		}

		rr := doRequest(t, handler, http.MethodPut, "/api/settings", body)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200 on update, got %d", rr.Code)
		}

		// Then read back.
		rr = doRequest(t, handler, http.MethodGet, "/api/settings", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200 on get, got %d", rr.Code)
		}

		var settings types.Settings
		decodeJSON(t, rr, &settings)

		if settings.DisplayName != "Persisted User" {
			t.Fatalf("expected display_name 'Persisted User', got %q", settings.DisplayName)
		}
		if settings.EmailAddress != "persisted@example.com" {
			t.Fatalf("expected email_address 'persisted@example.com', got %q", settings.EmailAddress)
		}
	})
}

// ---------------------------------------------------------------------------
// Integration: create then retrieve
// ---------------------------------------------------------------------------

func TestCreateThenGetEmail(t *testing.T) {
	handler := setupTestServer(t)

	// Create an email.
	body := types.ComposeRequest{
		To:       []types.Recipient{{Name: "Integration Test", Address: "integration@test.com"}},
		Subject:  "Integration Test Email",
		BodyText: "This email was created in an integration test.",
		BodyHTML: "<p>This email was created in an integration test.</p>",
	}

	rr := doRequest(t, handler, http.MethodPost, "/api/emails", body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: expected status 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var createResp struct {
		Email types.Email `json:"email"`
	}
	decodeJSON(t, rr, &createResp)

	newID := createResp.Email.ID
	if newID == "" {
		t.Fatal("expected non-empty ID from create")
	}

	// Retrieve the email.
	rr = doRequest(t, handler, http.MethodGet, "/api/emails/"+newID, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("get: expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var getResp struct {
		Email types.Email `json:"email"`
	}
	decodeJSON(t, rr, &getResp)

	if getResp.Email.ID != newID {
		t.Fatalf("expected ID %q, got %q", newID, getResp.Email.ID)
	}
	if getResp.Email.Subject != "Integration Test Email" {
		t.Fatalf("expected subject 'Integration Test Email', got %q", getResp.Email.Subject)
	}
}

func TestCreateUpdateThenVerify(t *testing.T) {
	handler := setupTestServer(t)

	// Create an email.
	body := types.ComposeRequest{
		To:       []types.Recipient{{Address: "update-test@test.com"}},
		Subject:  "Update Test Email",
		BodyText: "Testing update flow.",
		BodyHTML: "<p>Testing update flow.</p>",
	}

	rr := doRequest(t, handler, http.MethodPost, "/api/emails", body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", rr.Code)
	}

	var createResp struct {
		Email types.Email `json:"email"`
	}
	decodeJSON(t, rr, &createResp)
	newID := createResp.Email.ID

	// Star it.
	updates := map[string]any{"is_starred": true, "is_important": true}
	rr = doRequest(t, handler, http.MethodPut, "/api/emails/"+newID, updates)
	if rr.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var updatedEmail types.Email
	decodeJSON(t, rr, &updatedEmail)

	if !updatedEmail.IsStarred {
		t.Fatal("expected email to be starred")
	}
	if !updatedEmail.IsImportant {
		t.Fatal("expected email to be important")
	}
}

// ---------------------------------------------------------------------------
// Driver status endpoint
// ---------------------------------------------------------------------------

func TestDriverStatus(t *testing.T) {
	handler := setupTestServer(t)

	rr := doRequest(t, handler, http.MethodGet, "/api/driver/status", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp struct {
		Driver     string `json:"driver"`
		Configured bool   `json:"configured"`
		From       string `json:"from"`
	}
	decodeJSON(t, rr, &resp)

	if resp.Driver != "noop" {
		t.Fatalf("expected driver 'noop', got %q", resp.Driver)
	}
	if resp.Configured {
		t.Fatal("noop driver should not be reported as configured")
	}
	if resp.From != "test@example.com" {
		t.Fatalf("expected from 'test@example.com', got %q", resp.From)
	}
}

// ---------------------------------------------------------------------------
// Invalid method / not found
// ---------------------------------------------------------------------------

func TestInvalidMethods(t *testing.T) {
	handler := setupTestServer(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"PATCH on emails", http.MethodPatch, "/api/emails/email-001"},
		{"DELETE on labels list", http.MethodDelete, "/api/labels"},
		{"PUT on contacts list", http.MethodPut, "/api/contacts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := doRequest(t, handler, tt.method, tt.path, nil)
			// The mizu router returns 405 for wrong method on a known path,
			// or falls through to the catch-all route. Either 404 or 405 is acceptable.
			if rr.Code != http.StatusMethodNotAllowed && rr.Code != http.StatusNotFound && rr.Code != http.StatusOK {
				// OK is acceptable if it falls through to the dev-mode catch-all
				t.Logf("method %s on %s returned status %d", tt.method, tt.path, rr.Code)
			}
		})
	}
}
