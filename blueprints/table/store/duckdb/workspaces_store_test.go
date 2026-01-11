package duckdb_test

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/table/feature/workspaces"
)

func TestWorkspacesStoreBehaviors(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	owner := createTestUser(t, store)
	member := createTestUser(t, store)

	t.Run("GetMember not member", func(t *testing.T) {
		ws := createTestWorkspace(t, store, owner)
		if _, err := store.Workspaces().GetMember(ctx, ws.ID, member.ID); err != workspaces.ErrNotMember {
			t.Errorf("Expected ErrNotMember, got %v", err)
		}
	})

	t.Run("AddMember upsert and ListByUser", func(t *testing.T) {
		ws := createTestWorkspace(t, store, owner)

		if err := store.Workspaces().AddMember(ctx, &workspaces.Member{
			WorkspaceID: ws.ID,
			UserID:      member.ID,
			Role:        workspaces.RoleMember,
		}); err != nil {
			t.Fatalf("AddMember failed: %v", err)
		}

		if err := store.Workspaces().AddMember(ctx, &workspaces.Member{
			WorkspaceID: ws.ID,
			UserID:      member.ID,
			Role:        workspaces.RoleAdmin,
		}); err != nil {
			t.Fatalf("AddMember upsert failed: %v", err)
		}

		got, err := store.Workspaces().GetMember(ctx, ws.ID, member.ID)
		if err != nil {
			t.Fatalf("GetMember failed: %v", err)
		}
		if got.Role != workspaces.RoleAdmin {
			t.Errorf("Expected role admin, got %s", got.Role)
		}

		owned, err := store.Workspaces().ListByUser(ctx, owner.ID)
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}
		if len(owned) == 0 {
			t.Errorf("Expected owner to see workspaces")
		}

		memberList, err := store.Workspaces().ListByUser(ctx, member.ID)
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}
		if len(memberList) == 0 {
			t.Errorf("Expected member to see workspace")
		}
	})
}
