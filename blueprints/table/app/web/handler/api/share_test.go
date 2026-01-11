package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/table/feature/shares"
	"github.com/go-mizu/blueprints/table/store/duckdb"
)

func TestShareWorkflow(t *testing.T) {
	ts := newTestServer(t)
	token, user := registerUser(t, ts, "share@example.com")
	userID := requireString(t, user, "id")

	ws := createWorkspace(t, ts, token, "Workspace", "share-workspace")
	base := createBase(t, ts, token, requireString(t, ws, "id"), "Base")
	baseID := requireString(t, base, "id")
	table := createTable(t, ts, token, baseID, "Table")

	status, data := ts.doJSON(http.MethodGet, "/shares/base/"+baseID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "shares")) != 0 {
		t.Fatalf("expected empty share list")
	}

	share := createShare(t, ts, token, baseID, "link", "read")
	shareID := requireString(t, share, "id")
	shareToken := requireString(t, share, "token")

	status, data = ts.doJSON(http.MethodGet, "/shares/base/"+baseID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "shares")) != 1 {
		t.Fatalf("expected 1 share in list")
	}

	status, data = ts.doJSON(http.MethodGet, "/shares/token/"+shareToken, nil, "")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	baseGet := requireMap(t, data, "base")
	if requireString(t, baseGet, "id") != baseID {
		t.Fatalf("expected base in share response")
	}
	if len(requireSlice(t, data, "tables")) != 1 {
		t.Fatalf("expected table list in share response")
	}

	status, _ = ts.doJSON(http.MethodDelete, "/shares/"+shareID, nil, token)
	if status != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", status)
	}

	status, _ = ts.doJSON(http.MethodGet, "/shares/token/not-found", nil, "")
	if status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", status)
	}

	status, data = ts.doJSON(http.MethodGet, "/shares/base/"+baseID, nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if len(requireSlice(t, data, "shares")) != 0 {
		t.Fatalf("expected empty share list after delete")
	}

	expiredAt := time.Now().Add(-1 * time.Hour)
	store, err := duckdb.Open(ts.dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	shareSvc := shares.NewService(store.Shares())
	expiredShare, err := shareSvc.Create(context.Background(), userID, shares.CreateIn{
		BaseID:     baseID,
		TableID:    requireString(t, table, "id"),
		Type:       shares.TypeLink,
		Permission: shares.PermRead,
		ExpiresAt:  &expiredAt,
	})
	if err != nil {
		t.Fatalf("create expired share: %v", err)
	}

	status, _ = ts.doJSON(http.MethodGet, "/shares/token/"+expiredShare.Token, nil, "")
	if status != http.StatusBadRequest {
		t.Fatalf("expected status 400 for expired token, got %d", status)
	}
}
